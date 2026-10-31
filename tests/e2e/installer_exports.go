//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// InstallerHashes are the files a re-run must keep stable. Absent files are
// "absent". Current is the readlink text of /opt/vpn-node/current, not a digest.
type InstallerHashes struct {
	Key       string `json:"key_sha256,omitempty"`
	Cert      string `json:"cert_sha256,omitempty"`
	NodeJSON  string `json:"node_json_sha256,omitempty"`
	UsageDB   string `json:"usage_db_sha256,omitempty"`
	UsageWAL  string `json:"usage_wal_sha256,omitempty"`
	MachineID string `json:"machine_id_sha256,omitempty"`
	Current   string `json:"current_target,omitempty"`
}

// InstallerOutcome is one fvpn-node-install invocation. Exit follows the
// installer: 0 ok, 1 failed, 2 check drift, 3 busy, 4 platform, 5 dependency,
// 64 usage. upgrade can set Status to staged and still exit 1.
type InstallerOutcome struct {
	Exit              int
	Status            string
	Code              string
	Incarnation       string
	NodeID            string
	RetiredMode       string
	RetiredHashes     InstallerHashes
	BoundStateBefore  map[string]string
	RetiredBoundState map[string]string
	Message           string
	Before            InstallerHashes
	Hashes            InstallerHashes
	// Services is the state to pass to RestoreInstallerServices after sampling.
	// It preserves active services from before and from the installer command.
	Services InstallerServiceState
}

type installerSpec struct {
	Manifest             string `json:"manifest"`
	Artifact             string `json:"artifact"`
	Trust                string `json:"trust"`
	ControlURL           string `json:"control_url"`
	ServerCA             string `json:"server_ca"`
	TrustBundle          string `json:"trust_bundle"`
	HostReserveBPS       string `json:"host_reserve_bps"`
	SNAT                 string `json:"snat"`
	NetbirdManagementURL string `json:"netbird_management_url"`
	TokenFile            string `json:"token_file"`
}

func (s *Stack) installerContainer() string { return s.lab.prefix + "-installer-node" }
func (s *Stack) installerImage() string     { return s.lab.prefix + "-installer-node:24.04" }

func (s *Stack) requireInstallerSlot(slot string) error {
	if slot != "installer-node" {
		return fmt.Errorf("installer slot must be installer-node")
	}
	return nil
}

// PrepareInstallerNode starts fvpn-<prefix>-installer-node, a privileged
// systemd container on the lab network, and stages two signed releases.
// The regular node container is not changed. e2e-down removes this container.
func (s *Stack) PrepareInstallerNode(ctx context.Context) error {
	if s.installerReady(ctx) {
		if _, err := s.readInstallerSpec(ctx); err == nil {
			return nil
		}
	}
	if s.lab.labPrefix == "" || s.lab.originIP == "" {
		return fmt.Errorf("lab addresses unavailable")
	}
	if err := s.startInstaller(ctx); err != nil {
		return err
	}
	// Docker's embedded DNS nat is present because this container runs under
	// Docker. A real exit does not. Delete it here for the same reason the
	// node entrypoint does (D-079). inet fvpn belongs to the installer.
	_ = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "nft", "delete", "table", "ip", "nat").Run()
	_, err := s.buildInstallerBins(ctx)
	if err != nil {
		return err
	}
	base := filepath.Join(s.lab.root, ".agent", "tmp", "e2e-installer-rel")
	// A new container needs new signed fixtures even after an interrupted setup.
	if err = os.RemoveAll(base); err != nil {
		return err
	}
	if err = os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	older, err := s.BuildInstallerRelease(ctx, filepath.Join(base, "older"), "1.1.0", "10", "TEST-ONLY-installer-older", ReleaseOptions{})
	if err != nil {
		return err
	}
	newer, err := s.BuildInstallerRelease(ctx, filepath.Join(base, "newer"), "1.1.1", "11", "TEST-ONLY-installer-newer", ReleaseOptions{})
	if err != nil {
		return err
	}
	c := s.installerContainer()
	if out, err := exec.CommandContext(ctx, "docker", "exec", c, "mkdir", "-p", "/var/tmp/e2e-rel/older", "/var/tmp/e2e-rel/newer", "/usr/local/bin").CombinedOutput(); err != nil {
		return fmt.Errorf("installer dirs: %s", clipText(string(out), 200))
	}
	copies := [][2]string{
		{filepath.Join(older.Directory, "manifest.json"), c + ":/var/tmp/e2e-rel/older/manifest.json"},
		{filepath.Join(older.Directory, "amd64.tar.zst"), c + ":/var/tmp/e2e-rel/older/artifact.tar.zst"},
		{filepath.Join(s.lab.state, "node", "etc", "trust.json"), c + ":/var/tmp/e2e-rel/older/trust.json"},
		{filepath.Join(newer.Directory, "manifest.json"), c + ":/var/tmp/e2e-rel/newer/manifest.json"},
		{filepath.Join(newer.Directory, "amd64.tar.zst"), c + ":/var/tmp/e2e-rel/newer/artifact.tar.zst"},
		{filepath.Join(s.lab.state, "node", "etc", "trust.json"), c + ":/var/tmp/e2e-rel/newer/trust.json"},
		{filepath.Join(s.lab.state, "ca.pem"), c + ":/var/tmp/e2e-rel/ca.pem"},
		{filepath.Join(s.lab.state, "node", "etc", "trust.json"), c + ":/var/tmp/e2e-rel/node-trust.json"},
		{filepath.Join(s.lab.root, ".agent", "tmp", "e2e-installer-bins", "fvpn-node-install"), c + ":/usr/local/bin/fvpn-node-install"},
	}
	for _, name := range []string{"agent-ca.pem", "fault-proxy.crt", "fault-proxy.key"} {
		copies = append(copies, [2]string{filepath.Join(s.lab.state, "node", "etc", name), c + ":/var/tmp/e2e-rel/" + name})
	}
	helper := filepath.Join(base, "fvpn-e2e-nodeconfig")
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", helper, "./tests/e2e/cmd/fvpn-e2e-nodeconfig")
	build.Dir = s.lab.root
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if err := build.Run(); err != nil {
		return err
	}
	copies = append(copies, [2]string{helper, c + ":/usr/local/bin/fvpn-e2e-nodeconfig"})
	for _, pair := range copies {
		if err = dockerCopy(ctx, pair[0], pair[1]); err != nil {
			return err
		}
	}
	if out, err := exec.CommandContext(ctx, "docker", "exec", c, "chmod", "0755", "/usr/local/bin/fvpn-node-install").CombinedOutput(); err != nil {
		return fmt.Errorf("installer mode: %s", clipText(string(out), 200))
	}
	snat, err := installerSNAT(s.lab.originIP)
	if err != nil {
		return err
	}
	common := installerSpec{
		ControlURL: "https://127.0.0.1:9444", ServerCA: "/var/tmp/e2e-rel/ca.pem",
		TrustBundle: "/var/tmp/e2e-rel/node-trust.json", HostReserveBPS: "1000000", SNAT: snat,
		NetbirdManagementURL: "https://netbird.example.invalid", TokenFile: "/var/tmp/e2e-rel/token",
	}
	olderSpec, newerSpec := common, common
	olderSpec.Manifest, olderSpec.Artifact, olderSpec.Trust = "/var/tmp/e2e-rel/older/manifest.json", "/var/tmp/e2e-rel/older/artifact.tar.zst", "/var/tmp/e2e-rel/older/trust.json"
	newerSpec.Manifest, newerSpec.Artifact, newerSpec.Trust = "/var/tmp/e2e-rel/newer/manifest.json", "/var/tmp/e2e-rel/newer/artifact.tar.zst", "/var/tmp/e2e-rel/newer/trust.json"
	if err = s.writeInstallerSpec(ctx, "/var/tmp/e2e-rel/older.json", olderSpec); err != nil {
		return err
	}
	if err = s.writeInstallerSpec(ctx, "/var/tmp/e2e-rel/newer.json", newerSpec); err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, "docker", "exec", c, "sh", "-c", `cp /var/tmp/e2e-rel/ca.pem /usr/local/share/ca-certificates/lab-ca.crt && update-ca-certificates >/dev/null 2>&1`).Run(); err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, "docker", "exec", c, "systemd-run", "--unit=fvpn-e2e-proxy", "--collect", "/usr/local/bin/fvpn-e2e-nodeconfig", "installer-proxy").Run(); err != nil {
		return fmt.Errorf("installer proxy: %w", err)
	}
	ready, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if !waitUntil(ready, 200*time.Millisecond, func() bool {
		return exec.CommandContext(ready, "docker", "exec", c, "curl", "--silent", "--cacert", "/var/tmp/e2e-rel/ca.pem", "https://127.0.0.1:9444/readyz").Run() == nil
	}) {
		return fmt.Errorf("installer proxy not ready")
	}
	return s.SelectInstallerRelease(ctx, "older")
}

// RerunInstaller runs install --json. --token-file is passed only when
// identity/node.json is absent. The agent is stopped before the hash set so
// the SQLite WAL is not mid-write. Call RestoreInstallerServices with the
// returned Services snapshot after comparing Before/Hashes or reading the CLI.
func (s *Stack) RerunInstaller(ctx context.Context, slot string) (out InstallerOutcome, err error) {
	if err := s.requireInstallerSlot(slot); err != nil {
		return InstallerOutcome{}, err
	}
	spec, err := s.readInstallerSpec(ctx)
	if err != nil {
		return InstallerOutcome{}, err
	}
	_ = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "nft", "delete", "table", "ip", "nat").Run()
	args := s.installArgs(spec)
	if s.installerAbsent(ctx, "/var/lib/vpn-node/identity/node.json") {
		args = append(args, "--token-file", spec.TokenFile)
	}
	services, err := s.InstallerServices(ctx, slot)
	if err != nil {
		return out, err
	}
	defer func() {
		if out.Services == nil {
			out.Services = services
		}
	}()
	if err := s.stopInstallerAgent(ctx); err != nil {
		return InstallerOutcome{}, err
	}
	before, err := s.installerHashes(ctx)
	if err != nil {
		return InstallerOutcome{}, err
	}
	out = s.runInstaller(ctx, args)
	out.Before = before
	if err = s.retainInstallerServices(ctx, &out, services); err != nil {
		return out, err
	}
	if out.Exit < 0 {
		return out, fmt.Errorf("%s", out.Message)
	}
	if stopErr := s.stopInstallerAgent(ctx); stopErr != nil {
		return out, stopErr
	}
	out.Hashes, err = s.installerHashes(ctx)
	s.noteIdentity(ctx, &out)
	return out, err
}

// ControlledReinstall runs reenroll --json --token-file. enrollment is the
// token and is not logged. The previous identity moves under identity.retired.
func (s *Stack) ControlledReinstall(ctx context.Context, slot, enrollment string) (out InstallerOutcome, err error) {
	if err := s.requireInstallerSlot(slot); err != nil {
		return InstallerOutcome{}, err
	}
	if strings.TrimSpace(enrollment) == "" {
		return InstallerOutcome{}, fmt.Errorf("enrollment token required")
	}
	const path = "/var/tmp/e2e-rel/reenroll.token"
	if err := writeContainerFile(ctx, s.installerContainer(), path, "600", []byte(strings.TrimSpace(enrollment)+"\n")); err != nil {
		return InstallerOutcome{}, err
	}
	enrollment = ""
	services, err := s.InstallerServices(ctx, slot)
	if err != nil {
		return out, err
	}
	defer func() {
		if out.Services == nil {
			out.Services = services
		}
	}()
	_ = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "nft", "delete", "table", "ip", "nat").Run()
	if err := s.stopInstallerAgent(ctx); err != nil {
		return InstallerOutcome{}, err
	}
	before, err := s.installerHashes(ctx)
	if err != nil {
		return InstallerOutcome{}, err
	}
	bound, err := s.installerBoundHashes(ctx, "/var/lib/vpn-node/state", false)
	if err != nil {
		return InstallerOutcome{}, err
	}
	out = s.runInstaller(ctx, []string{"/usr/local/bin/fvpn-node-install", "reenroll", "--json", "--token-file", path})
	out.Before = before
	out.BoundStateBefore = bound
	_ = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "rm", "-f", path).Run()
	if err = s.retainInstallerServices(ctx, &out, services); err != nil {
		return out, err
	}
	if out.Exit < 0 {
		return out, fmt.Errorf("%s", out.Message)
	}
	if stopErr := s.stopInstallerAgent(ctx); stopErr != nil {
		return out, stopErr
	}
	out.Hashes, err = s.installerHashes(ctx)
	retired, retiredErr := s.retiredDirectory(ctx)
	if retiredErr == nil && retired != "" {
		out.RetiredMode, retiredErr = s.fileMode(ctx, retired)
		if retiredErr != nil {
			return out, retiredErr
		}
		for _, item := range []struct {
			dest *string
			name string
		}{{&out.RetiredHashes.Key, "key.pem"}, {&out.RetiredHashes.Cert, "cert.pem"}, {&out.RetiredHashes.NodeJSON, "node.json"}} {
			*item.dest, retiredErr = s.hashInside(ctx, retired+"/"+item.name, false)
			if retiredErr != nil {
				return out, retiredErr
			}
		}
		out.RetiredBoundState, retiredErr = s.installerBoundHashes(ctx, retired, true)
	}
	if err == nil {
		err = retiredErr
	}
	s.noteIdentity(ctx, &out)
	return out, err
}

// ProbeInstallerExit runs one installer command and returns its exit code.
// Exit 4 (UNSUPPORTED_PLATFORM) is not produced on Ubuntu 24.04.
func (s *Stack) ProbeInstallerExit(ctx context.Context, slot string, args []string) (InstallerOutcome, error) {
	if err := s.requireInstallerSlot(slot); err != nil {
		return InstallerOutcome{}, err
	}
	full := args
	if len(args) == 0 || !strings.Contains(args[0], "fvpn-node-install") {
		full = append([]string{"/usr/local/bin/fvpn-node-install"}, args...)
	}
	out := s.runInstaller(ctx, full)
	if out.Exit < 0 {
		return out, fmt.Errorf("%s", out.Message)
	}
	return out, nil
}

func (s *Stack) SelectInstallerRelease(ctx context.Context, which string) error {
	if which != "older" && which != "newer" {
		return fmt.Errorf("unknown installer release")
	}
	out, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "cp", "/var/tmp/e2e-rel/"+which+".json", "/var/tmp/e2e-installer.json").CombinedOutput()
	if err != nil {
		return fmt.Errorf("select release: %s", clipText(string(out), 200))
	}
	return nil
}

func (s *Stack) StageInstallerToken(ctx context.Context, slot, token string) error {
	if err := s.requireInstallerSlot(slot); err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token required")
	}
	err := writeContainerFile(ctx, s.installerContainer(), "/var/tmp/e2e-rel/token", "600", []byte(strings.TrimSpace(token)+"\n"))
	token = ""
	return err
}

func (s *Stack) startInstaller(ctx context.Context) error {
	if s.installerReady(ctx) {
		return nil
	}
	image := s.installerImage()
	if exec.CommandContext(ctx, "docker", "image", "inspect", image).Run() != nil {
		cmd := exec.CommandContext(ctx, "docker", "build", "-t", image, "-f", filepath.Join(s.lab.root, "deploy/devstack/Dockerfile.installer-node"), filepath.Join(s.lab.root, "deploy/devstack"))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("installer image: %s", clipText(string(out), 400))
		}
	}
	name := s.installerContainer()
	var last string
	for attempt := 1; attempt <= 3; attempt++ {
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
		args := []string{"run", "-d", "--name", name, "--privileged", "--cgroupns=host",
			"-v", "/sys/fs/cgroup:/sys/fs/cgroup:rw",
			"--network", s.lab.prefix, "--ip", s.lab.labPrefix + ".42",
			"--add-host", "agent.example.invalid:" + s.lab.labPrefix + ".30",
			"--add-host", "netbird.example.invalid:" + s.lab.labPrefix + ".21",
			"--add-host", "portal.example.invalid:" + s.lab.labPrefix + ".21",
			"--add-host", "idp.example.invalid:" + s.lab.labPrefix + ".11",
			"--add-host", "releases.example.invalid:" + s.lab.originIP,
			image}
		if out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("installer run: %s", clipText(string(out), 300))
		}
		ip := net.ParseIP(s.lab.originIP).To4()
		ip[3] = 24
		if err := exec.CommandContext(ctx, "docker", "network", "connect", "--ip", ip.String(), s.lab.prefix+"-origin", name).Run(); err != nil {
			return fmt.Errorf("installer origin attachment: %w", err)
		}
		deadline := time.Now().Add(45 * time.Second)
		for time.Now().Before(deadline) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			inspect, _ := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name).Output()
			if strings.TrimSpace(string(inspect)) != "true" {
				last = "container exited"
				break
			}
			out, _ := exec.CommandContext(ctx, "docker", "exec", name, "systemctl", "is-system-running").CombinedOutput()
			last = strings.TrimSpace(string(out))
			if last == "running" || last == "degraded" {
				return exec.CommandContext(ctx, "docker", "exec", name, "sh", "-c", `dev=$(ip -o -4 addr show | awk '$4 ~ /^192[.]0[.]/ {print $2; exit}'); test -n "$dev"; ip link set "$dev" down && ip link set "$dev" name fvup0 && ip link set fvup0 up`).Run()
			}
			time.Sleep(time.Second)
		}
	}
	return fmt.Errorf("installer systemd did not come up (%s)", last)
}

func (s *Stack) installerReady(ctx context.Context) bool {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", s.installerContainer()).Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return false
	}
	state, _ := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "systemctl", "is-system-running").CombinedOutput()
	text := strings.TrimSpace(string(state))
	return text == "running" || text == "degraded"
}

func (s *Stack) buildInstallerBins(ctx context.Context) (map[string][]byte, error) {
	dir := filepath.Join(s.lab.root, ".agent", "tmp", "e2e-installer-bins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	pkgs := []struct{ out, pkg string }{
		{"fvpn-agent", "./node/agent/cmd/fvpn-agent"},
		{"fvpn-updater", "./node/updater/cmd/fvpn-updater"},
		{"fvpn-watchdog", "./node/watchdog/cmd/fvpn-watchdog"},
		{"fvpn-node-install", "./packaging/installer/cmd/fvpn-node-install"},
	}
	bins := map[string][]byte{}
	env := append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	for _, p := range pkgs {
		dest := filepath.Join(dir, p.out)
		cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", dest, p.pkg)
		cmd.Dir, cmd.Env = s.lab.root, env
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("build %s: %s", p.out, clipText(string(out), 400))
		}
		raw, err := os.ReadFile(dest)
		if err != nil {
			return nil, err
		}
		bins[p.out] = raw
	}
	return bins, nil
}

func (s *Stack) installArgs(spec installerSpec) []string {
	return []string{"/usr/local/bin/fvpn-node-install", "install", "--json",
		"--manifest", spec.Manifest, "--artifact", spec.Artifact, "--trust", spec.Trust,
		"--control-url", spec.ControlURL, "--server-ca", spec.ServerCA, "--trust-bundle", spec.TrustBundle,
		"--host-reserve-bps", spec.HostReserveBPS, "--snat", spec.SNAT,
		"--netbird-management-url", spec.NetbirdManagementURL}
}

func (s *Stack) runInstaller(ctx context.Context, args []string) InstallerOutcome {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", s.installerContainer()}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	out := InstallerOutcome{}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			out.Exit = exit.ExitCode()
		} else {
			out.Exit = -1
			out.Message = clipText(err.Error(), 200)
			return out
		}
	}
	return parseInstallerOutcome(stdout.Bytes(), stderr.Bytes(), out.Exit)
}

func parseInstallerOutcome(stdout, stderr []byte, code int) InstallerOutcome {
	out := InstallerOutcome{Exit: code}
	if i := bytes.IndexByte(stdout, '{'); i >= 0 {
		var res struct {
			Status      string `json:"status"`
			Incarnation string `json:"incarnation_id"`
			Message     string `json:"message"`
		}
		if json.Unmarshal(stdout[i:], &res) == nil {
			out.Status, out.Incarnation = res.Status, res.Incarnation
			if res.Message != "" {
				out.Message = res.Message
			}
		}
	}
	out.Code = installerCode(string(stderr))
	if out.Exit != 0 {
		lines := strings.Split(strings.TrimSpace(string(stderr)), "\n")
		if len(lines) > 0 {
			out.Message = clipText(lines[len(lines)-1], 300)
		}
	}
	return out
}

func (s *Stack) stopInstallerAgent(ctx context.Context) error {
	return exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "sh", "-c", `for unit in fvpn-updater.timer fvpn-updater.service fvpn-watchdog.service fvpn-agent.service; do
 state=$(systemctl show -p LoadState --value "$unit") || exit 1
 if [ "$state" != not-found ]; then systemctl stop "$unit" || exit 1; fi
done`).Run()
}

func (s *Stack) installerHashes(ctx context.Context) (InstallerHashes, error) {
	var h InstallerHashes
	var err error
	paths := []struct {
		dest *string
		path string
	}{
		{&h.Key, "/var/lib/vpn-node/identity/key.pem"},
		{&h.Cert, "/var/lib/vpn-node/identity/cert.pem"},
		{&h.NodeJSON, "/var/lib/vpn-node/identity/node.json"},
		{&h.UsageDB, "/var/lib/vpn-node/state/usage.db"},
		{&h.UsageWAL, "/var/lib/vpn-node/state/usage.db-wal"},
		{&h.MachineID, "/etc/machine-id"},
	}
	for _, item := range paths {
		*item.dest, err = s.hashInside(ctx, item.path, false)
		if err != nil {
			return h, err
		}
	}
	h.Current, err = s.hashInside(ctx, "/opt/vpn-node/current", true)
	return h, err
}

func (s *Stack) hashInside(ctx context.Context, path string, link bool) (string, error) {
	c := s.installerContainer()
	script := `if [ ! -e "$1" ]; then printf 'absent\n'; else sha256sum "$1"; fi`
	if link {
		script = `if [ ! -L "$1" ]; then printf 'absent\n'; else readlink "$1"; fi`
	}
	out, err := exec.CommandContext(ctx, "docker", "exec", c, "sh", "-c", script, "sh", path).Output()
	if err != nil {
		return "", err
	}
	if link {
		return strings.TrimSpace(string(out)), nil
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "", fmt.Errorf("hash %s", path)
	}
	return fields[0], nil
}

func (s *Stack) installerAbsent(ctx context.Context, path string) bool {
	return exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "test", "-e", path).Run() != nil
}

func (s *Stack) fileMode(ctx context.Context, path string) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "stat", "-c", "%a", path).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *Stack) retiredDirectory(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "sh", "-c", `for d in /var/lib/vpn-node/identity.retired/*; do [ -d "$d" ] && printf '%s\n' "$d"; done | sort | tail -1`).Output()
	return strings.TrimSpace(string(out)), err
}
func (s *Stack) installerBoundHashes(ctx context.Context, dir string, retired bool) (map[string]string, error) {
	result := map[string]string{}
	for _, name := range []string{"agent.db", "agent.db-wal", "agent.db-shm", "desired-state.json"} {
		path := dir + "/"
		if retired {
			path += "state-"
		}
		path += name
		digest, err := s.hashInside(ctx, path, false)
		if err != nil {
			return nil, err
		}
		key := strings.NewReplacer(".", "_", "-", "_").Replace(name) + "_sha256"
		result[key] = digest
	}
	return result, nil
}

func (s *Stack) noteIdentity(ctx context.Context, out *InstallerOutcome) {
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "cat", "/var/lib/vpn-node/identity/node.json").Output()
	if err != nil {
		return
	}
	var body struct {
		Node string `json:"node_id"`
		Inc  string `json:"node_incarnation_id"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return
	}
	if out.NodeID == "" {
		out.NodeID = body.Node
	}
	if out.Incarnation == "" {
		out.Incarnation = body.Inc
	}
}

func (s *Stack) readInstallerSpec(ctx context.Context) (installerSpec, error) {
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "cat", "/var/tmp/e2e-installer.json").Output()
	if err != nil {
		return installerSpec{}, fmt.Errorf("installer release is not staged")
	}
	var spec installerSpec
	if json.Unmarshal(raw, &spec) != nil || spec.Manifest == "" {
		return installerSpec{}, fmt.Errorf("installer release spec")
	}
	return spec, nil
}

func (s *Stack) writeInstallerSpec(ctx context.Context, path string, spec installerSpec) error {
	raw, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return writeContainerFile(ctx, s.installerContainer(), path, "644", append(raw, '\n'))
}

func writeContainerFile(ctx context.Context, container, path, mode string, body []byte) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container, "sh", "-c", `umask 077; mkdir -p "$(dirname "$1")"; cat > "$1" && chmod "$2" "$1"`, "sh", path, mode)
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("container file: %s", clipText(string(out), 200))
	}
	return nil
}

func installerCode(stderr string) string {
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "{") {
			continue
		}
		if i := strings.IndexByte(line, ':'); i > 0 {
			return strings.TrimSpace(line[:i])
		}
	}
	return ""
}

func installerSNAT(origin string) (string, error) {
	ip := net.ParseIP(origin).To4()
	if ip == nil {
		return "", fmt.Errorf("origin address")
	}
	ip[3] = 24
	return ip.String(), nil
}

func sameInstallerHashes(a, b InstallerHashes) bool {
	return a == b
}

func sameExceptCurrent(a, b InstallerHashes) bool {
	a.Current, b.Current = "", ""
	return a == b
}
