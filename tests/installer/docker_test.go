//go:build installer

package installertest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
)

const (
	imageRef = "fvpn-w5-installer-ubuntu:24.04"
	// TEST ONLY enrollment material. These are not product tokens.
	token1 = "test-only-enrollment-token"
	token2 = "test-only-enrollment-token-2"
	// TEST ONLY. Not a product reserve (§8, 待實測).
	reserveBPS = "1000000"
)

var (
	rootDir string
	binDir  string
	rel1    installer.ReleaseFixture
	rel2    installer.ReleaseFixture
)

func TestMain(m *testing.M) { os.Exit(runSuite(m)) }

func runSuite(m *testing.M) int {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rootDir = wd
	for {
		if _, err := os.Stat(filepath.Join(rootDir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(rootDir)
		if parent == rootDir {
			fmt.Fprintln(os.Stderr, "go.mod not found")
			return 1
		}
		rootDir = parent
	}
	binDir, err = os.MkdirTemp("", "fvpn-w5-installer-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(binDir)
	defer cleanupDocker()
	if err := buildBins(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rel1, err = buildRel(binDir, "019a0000-0000-7000-8000-0000000000c1", "1")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rel2, err = buildRel(binDir, "019a0000-0000-7000-8000-0000000000c2", "2")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := trustBothReleases(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command("docker", "build", "-t", imageRef, "-f", filepath.Join(rootDir, "tests/installer/Dockerfile"), filepath.Join(rootDir, "tests/installer"))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return m.Run()
}

func buildBins() error {
	pkgs := []struct{ out, pkg string }{
		{"fvpn-agent", "./node/agent/cmd/fvpn-agent"},
		{"fvpn-updater", "./node/updater/cmd/fvpn-updater"},
		{"fvpn-watchdog", "./node/watchdog/cmd/fvpn-watchdog"},
		{"fvpn-node-install", "./packaging/installer/cmd/fvpn-node-install"},
	}
	env := append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	for _, p := range pkgs {
		cmd := exec.Command("go", "build", "-trimpath", "-o", filepath.Join(binDir, p.out), p.pkg)
		cmd.Dir, cmd.Env = rootDir, env
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build %s: %w", p.out, err)
		}
	}
	cmd := exec.Command("go", "build", "-tags", "installer", "-trimpath", "-o", filepath.Join(binDir, "fvpn-w5-control"), "./tests/installer/cmd/fvpn-w5-control")
	cmd.Dir, cmd.Env = rootDir, env
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build control: %w", err)
	}
	return nil
}

func buildRel(dir, id, seq string) (installer.ReleaseFixture, error) {
	bins := map[string][]byte{}
	for _, name := range []string{"fvpn-agent", "fvpn-updater", "fvpn-watchdog", "fvpn-node-install"} {
		raw, err := os.ReadFile(filepath.Join(binDir, name))
		if err != nil {
			return installer.ReleaseFixture{}, err
		}
		bins[name] = raw
	}
	out := filepath.Join(dir, "rel-"+seq)
	return installer.BuildReleaseFixture(out, filepath.Join(rootDir, "deploy/node/systemd"), id, seq, bins)
}

// TEST ONLY: both independently generated signing keys are provisioned before
// install. The production updater must not trust a key merely because an
// upgrade command supplied it.
func trustBothReleases() error {
	var combined struct {
		Keys []json.RawMessage `json:"keys"`
	}
	for _, rel := range []installer.ReleaseFixture{rel1, rel2} {
		raw, err := os.ReadFile(rel.Trust)
		if err != nil {
			return err
		}
		var one struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.Unmarshal(raw, &one); err != nil {
			return err
		}
		combined.Keys = append(combined.Keys, one.Keys...)
	}
	raw, err := json.Marshal(combined)
	if err != nil {
		return err
	}
	for _, rel := range []installer.ReleaseFixture{rel1, rel2} {
		if err := os.WriteFile(rel.Trust, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func cleanupDocker() {
	out, _ := exec.Command("docker", "ps", "-aq", "--filter", "name=fvpn-w5-installer-").Output()
	ids := strings.Fields(string(out))
	if len(ids) > 0 {
		args := append([]string{"rm", "-fv"}, ids...)
		_ = exec.Command("docker", args...).Run()
	}
	out, _ = exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "fvpn-w5-installer-") {
			_ = exec.Command("docker", "rmi", "-f", line).Run()
		}
	}
}

func runCmd(t *testing.T, d time.Duration, name string, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s timed out: %s", name, out)
	}
	if err == nil {
		return string(out), 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	return "", 1
}

func dexec(t *testing.T, c string, args ...string) (string, int) {
	t.Helper()
	return runCmd(t, 4*time.Minute, "docker", append([]string{"exec", c}, args...)...)
}

func mustExec(t *testing.T, c string, args ...string) string {
	t.Helper()
	out, code := dexec(t, c, args...)
	if code != 0 {
		t.Fatalf("exec %v -> %d\n%s", args, code, out)
	}
	return out
}

func waitSystemd(t *testing.T, name string, d time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		inspect, _ := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name).Output()
		if strings.TrimSpace(string(inspect)) != "true" {
			return false
		}
		out, _ := dexec(t, name, "systemctl", "is-system-running")
		state := strings.TrimSpace(out)
		if state == "running" || state == "degraded" {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}

func startSystemd(t *testing.T, name string) {
	t.Helper()
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-fv", name).Run() })
	var last string
	// A privileged systemd PID 1 occasionally exits in the first instant
	// when the host cgroup is busy. Retry that start; do not treat it as
	// an installer failure.
	for attempt := 1; attempt <= 3; attempt++ {
		_ = exec.Command("docker", "rm", "-fv", name).Run()
		out, code := runCmd(t, time.Minute, "docker", "run", "-d", "--name", name,
			"--privileged", "--cgroupns=host",
			"-v", "/sys/fs/cgroup:/sys/fs/cgroup:rw",
			imageRef)
		if code != 0 {
			t.Fatalf("docker run %s: %s", name, out)
		}
		deadline := time.Now().Add(45 * time.Second)
		for time.Now().Before(deadline) {
			inspect, _ := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name).Output()
			if strings.TrimSpace(string(inspect)) != "true" {
				last = "container exited"
				break
			}
			out, _ = dexec(t, name, "systemctl", "is-system-running")
			last = strings.TrimSpace(out)
			if last == "running" || last == "degraded" {
				return
			}
			time.Sleep(time.Second)
		}
	}
	logs, _ := runCmd(t, 20*time.Second, "docker", "logs", name)
	t.Fatalf("systemd did not come up (%q)\n%s", last, logs)
}

func startSleep(t *testing.T, name string) {
	t.Helper()
	_ = exec.Command("docker", "rm", "-fv", name).Run()
	out, code := runCmd(t, time.Minute, "docker", "run", "-d", "--name", name, "--entrypoint", "sleep", imageRef, "infinity")
	if code != 0 {
		t.Fatalf("sleep container: %s", out)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-fv", name).Run() })
}

func copyIn(t *testing.T, c, src, dest string) {
	t.Helper()
	out, code := runCmd(t, time.Minute, "docker", "cp", src, c+":"+dest)
	if code != 0 {
		t.Fatalf("docker cp %s: %s", src, out)
	}
}

func stage(t *testing.T, c string) {
	t.Helper()
	mustExec(t, c, "mkdir", "-p", "/usr/local/bin", "/var/tmp/w5/rel1", "/var/tmp/w5/rel2")
	copyIn(t, c, filepath.Join(binDir, "fvpn-node-install"), "/usr/local/bin/fvpn-node-install")
	copyIn(t, c, filepath.Join(binDir, "fvpn-w5-control"), "/usr/local/bin/fvpn-w5-control")
	mustExec(t, c, "chmod", "0755", "/usr/local/bin/fvpn-node-install", "/usr/local/bin/fvpn-w5-control")
	copyIn(t, c, rel1.Dir+"/.", "/var/tmp/w5/rel1")
	copyIn(t, c, rel2.Dir+"/.", "/var/tmp/w5/rel2")
}

type crel struct{ Manifest, Artifact, Trust, ID string }

func relIn(seq, id string) crel {
	base := "/var/tmp/w5/rel" + seq
	return crel{
		Manifest: base + "/manifest-" + seq + ".json",
		Artifact: base + "/amd64-" + seq + ".tar.zst",
		Trust:    base + "/trust-" + seq + ".json",
		ID:       id,
	}
}

func writeStdin(t *testing.T, c, path, body string, mode string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", c, "sh", "-c", `umask 077; cat > "$1" && chmod "$2" "$1"`, "sh", path, mode)
	cmd.Stdin = strings.NewReader(body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("write %s: %v\n%s", path, err, out)
	}
}

func startControl(t *testing.T, c string) (url, ca, bundle string) {
	t.Helper()
	mustExec(t, c, "mkdir", "-p", "/run/w5-control")
	writeStdin(t, c, "/run/w5-control/token.in", token1+"\n", "0600")
	writeStdin(t, c, "/run/w5-control/token2.in", token2+"\n", "0600")
	out, code := runCmd(t, 30*time.Second, "docker", "exec", "-d", c,
		"/usr/local/bin/fvpn-w5-control",
		"-dir", "/run/w5-control",
		"-token-file", "/run/w5-control/token.in",
		"-extra-token-file", "/run/w5-control/token2.in")
	if code != 0 {
		t.Fatalf("control: %s", out)
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, code := dexec(t, c, "test", "-f", "/run/w5-control/ready"); code == 0 {
			raw := mustExec(t, c, "cat", "/run/w5-control/url")
			return strings.TrimSpace(raw), "/run/w5-control/ca.pem", "/run/w5-control/trust.json"
		}
		time.Sleep(200 * time.Millisecond)
	}
	logs := mustExec(t, c, "sh", "-c", "ls -la /run/w5-control || true")
	t.Fatalf("control did not become ready\n%s", logs)
	return "", "", ""
}

func installArgs(rel crel, url, ca, bundle, token string, extra ...string) []string {
	args := []string{"/usr/local/bin/fvpn-node-install", "install", "--json",
		"--manifest", rel.Manifest, "--artifact", rel.Artifact, "--trust", rel.Trust,
		"--control-url", url, "--server-ca", ca, "--trust-bundle", bundle,
		"--host-reserve-bps", reserveBPS, "--snat", "192.0.2.1",
		"--netbird-management-url", "https://netbird.example.invalid"}
	if token != "" {
		args = append(args, "--token-file", token)
	}
	return append(args, extra...)
}

func doInstall(t *testing.T, c string, rel crel, url, ca, bundle, token string, extra ...string) (installer.Result, string, int) {
	t.Helper()
	out, code := dexec(t, c, installArgs(rel, url, ca, bundle, token, extra...)...)
	var res installer.Result
	if i := strings.Index(out, "{"); i >= 0 {
		_ = json.Unmarshal([]byte(out[i:]), &res)
	}
	return res, out, code
}

func sha(t *testing.T, c, path string) string {
	t.Helper()
	out := mustExec(t, c, "sha256sum", path)
	f := strings.Fields(out)
	if len(f) == 0 {
		t.Fatalf("hash %s: %s", path, out)
	}
	return f[0]
}

func incarnation(t *testing.T, c string) string {
	t.Helper()
	raw := mustExec(t, c, "cat", "/var/lib/vpn-node/identity/node.json")
	var n struct {
		Inc string `json:"node_incarnation_id"`
	}
	if err := json.Unmarshal([]byte(raw), &n); err != nil || n.Inc == "" {
		t.Fatalf("node.json %s: %v", raw, err)
	}
	return n.Inc
}

func seedForeign(t *testing.T, c string) (defRoute, resolv string) {
	t.Helper()
	mustExec(t, c, "nft", "add", "table", "inet", "foreignw5")
	mustExec(t, c, "nft", "add", "chain", "inet", "foreignw5", "keep", "{ type filter hook input priority 0; policy accept; }")
	mustExec(t, c, "ip", "route", "add", "203.0.113.0/24", "dev", "lo")
	defRoute = strings.TrimSpace(mustExec(t, c, "ip", "route", "show", "default"))
	resolv = sha(t, c, "/etc/resolv.conf")
	return defRoute, resolv
}

func assertForeign(t *testing.T, c, defRoute, resolv string) {
	t.Helper()
	out := mustExec(t, c, "nft", "list", "table", "inet", "foreignw5")
	if !strings.Contains(out, "keep") {
		t.Fatalf("foreign table changed: %s", out)
	}
	out = mustExec(t, c, "ip", "route", "show", "203.0.113.0/24")
	if !strings.Contains(out, "203.0.113.0/24") {
		t.Fatalf("foreign route missing: %s", out)
	}
	if got := strings.TrimSpace(mustExec(t, c, "ip", "route", "show", "default")); got != defRoute {
		t.Fatalf("default route changed\n%s\n%s", defRoute, got)
	}
	if got := sha(t, c, "/etc/resolv.conf"); got != resolv {
		t.Fatalf("resolv.conf changed")
	}
}

func TestMissingTool(t *testing.T) {
	const name = "fvpn-w5-installer-missing"
	startSleep(t, name)
	stage(t, name)
	mustExec(t, name, "mv", "/usr/sbin/nft", "/usr/sbin/nft.aside")
	_, out, code := doInstall(t, name, relIn("1", rel1.ReleaseID), "https://control.example.invalid", "/no/ca", "/no/bundle", "/no/token")
	if code != installer.ExitDependency {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "DEPENDENCY_UNAVAILABLE") || !strings.Contains(out, "nft") {
		t.Fatalf("stderr %s", out)
	}
	if _, code := dexec(t, name, "test", "-e", "/var/lib/vpn-node"); code == 0 {
		t.Fatal("journal directory was created before the tool check")
	}
}

func TestTokenArgvRefused(t *testing.T) {
	const name = "fvpn-w5-installer-token"
	startSleep(t, name)
	copyIn(t, name, filepath.Join(binDir, "fvpn-node-install"), "/usr/local/bin/fvpn-node-install")
	out, code := dexec(t, name, "/usr/local/bin/fvpn-node-install", "install", "--token", "not-a-file")
	if code != installer.ExitUsage || !strings.Contains(out, "token") {
		t.Fatalf("exit %d\n%s", code, out)
	}
}

func TestFaultRollback(t *testing.T) {
	const name = "fvpn-w5-installer-fault"
	startSystemd(t, name)
	stage(t, name)
	defRoute, resolv := seedForeign(t, name)
	writeStdin(t, name, "/run/w5-token", token1+"\n", "0600")
	res, out, code := doInstall(t, name, relIn("1", rel1.ReleaseID),
		"https://control.example.invalid", "/no/ca", "/no/bundle", "/run/w5-token",
		"--fault-after", "directories")
	if code != installer.ExitFailed || res.Status != "" && !strings.Contains(out, "rolled_back") {
		t.Fatalf("exit %d status %s\n%s", code, res.Status, out)
	}
	if !strings.Contains(out, "rolled_back") {
		t.Fatalf("output %s", out)
	}
	for _, p := range []string{"/opt/vpn-node", "/etc/vpn-node", "/var/lib/vpn-node/identity"} {
		if _, code := dexec(t, name, "test", "-e", p); code == 0 {
			t.Fatalf("%s survived rollback", p)
		}
	}
	raw := mustExec(t, name, "cat", "/var/lib/vpn-node/install/journal.json")
	if !strings.Contains(raw, `"rolled_back"`) {
		t.Fatalf("journal %s", raw)
	}
	if strings.Contains(raw, "flush ruleset") {
		t.Fatal("journal records a ruleset flush")
	}
	if _, code := dexec(t, name, "nft", "list", "table", "inet", "fvpn"); code == 0 {
		t.Fatal("rollback left inet fvpn")
	}
	assertForeign(t, name, defRoute, resolv)
	if _, code := dexec(t, name, "ip", "link", "show", "fvwprobe"); code == 0 {
		t.Fatal("wireguard probe interface remains")
	}
}

func TestInstallLifecycle(t *testing.T) {
	const name = "fvpn-w5-installer-main"
	startSystemd(t, name)
	stage(t, name)
	defRoute, resolv := seedForeign(t, name)
	url, ca, bundle := startControl(t, name)
	writeStdin(t, name, "/run/w5-token", token1+"\n", "0600")
	res, out, code := doInstall(t, name, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 || res.Status != "installed" {
		t.Fatalf("install %d %+v\n%s", code, res, out)
	}
	if _, code := dexec(t, name, "test", "!", "-e", "/run/w5-token"); code != 0 {
		t.Fatal("token file was not consumed")
	}
	waitPath(t, name, "/var/lib/vpn-node/state/usage.db")
	key1 := sha(t, name, "/var/lib/vpn-node/identity/key.pem")
	cert1 := sha(t, name, "/var/lib/vpn-node/identity/cert.pem")
	node1 := sha(t, name, "/var/lib/vpn-node/identity/node.json")
	usage1 := sha(t, name, "/var/lib/vpn-node/state/usage.db")
	inc1 := incarnation(t, name)
	check := mustCheck(t, name, 0)
	if check.Status != "healthy" || !check.Identity.Present || check.Identity.IncarnationID != inc1 || !check.UsageStore {
		t.Fatalf("check %+v", check)
	}
	if check.ReleaseID != rel1.ReleaseID || check.ManifestSHA256 == "" {
		t.Fatalf("release in check %+v", check)
	}
	assertBaseline(t, check.Baseline, out)
	updaterConfig := mustExec(t, name, "cat", "/etc/vpn-node/updater.json")
	if strings.Contains(updaterConfig, `"boot_id"`) {
		t.Fatal("generated config should omit boot_id")
	}
	noWork := mustExec(t, name, "/opt/vpn-node/current/fvpn-updater", "run", "--config", "/etc/vpn-node/updater.json")
	if !strings.Contains(noWork, "desired-state file is absent") {
		t.Fatalf("fresh run: %s", noWork)
	}
	res, out, code = doInstall(t, name, relIn("1", rel1.ReleaseID), url, ca, bundle, "")
	if code != 0 || res.Status != "already_current" {
		t.Fatalf("re-run %d %+v\n%s", code, res, out)
	}
	if sha(t, name, "/var/lib/vpn-node/identity/key.pem") != key1 ||
		sha(t, name, "/var/lib/vpn-node/identity/cert.pem") != cert1 ||
		sha(t, name, "/var/lib/vpn-node/identity/node.json") != node1 ||
		sha(t, name, "/var/lib/vpn-node/state/usage.db") != usage1 {
		t.Fatal("identity or usage store changed on re-run")
	}
	mustExec(t, name, "rm", "-f", "/etc/systemd/system/fvpn-agent.service")
	mustExec(t, name, "chmod", "0755", "/var/lib/vpn-node/identity")
	drift := mustCheck(t, name, installer.ExitDrifted)
	joined := strings.Join(drift.Drifts, " ")
	if !strings.Contains(joined, "units") || !strings.Contains(joined, "mode /var/lib/vpn-node/identity") {
		t.Fatalf("drifts %v", drift.Drifts)
	}
	res, out, code = doInstall(t, name, relIn("1", rel1.ReleaseID), url, ca, bundle, "")
	if code != 0 || res.Status != "repaired" {
		t.Fatalf("repair %d %+v\n%s", code, res, out)
	}
	if sha(t, name, "/var/lib/vpn-node/identity/key.pem") != key1 {
		t.Fatal("repair changed identity")
	}
	if _, code := dexec(t, name, "test", "-f", "/etc/systemd/system/fvpn-agent.service"); code != 0 {
		t.Fatal("repair did not restore the unit")
	}
	mode := strings.TrimSpace(mustExec(t, name, "stat", "-c", "%a", "/var/lib/vpn-node/identity"))
	if mode != "700" {
		t.Fatalf("identity mode %s", mode)
	}
	testPendingHandoff(t, name, check.Baseline)
	res, out, code = doInstall(t, name, relIn("2", rel2.ReleaseID), url, ca, bundle, "")
	if code != 0 || res.ReleaseID != rel2.ReleaseID {
		t.Fatalf("newer install %d %+v\n%s", code, res, out)
	}
	if got := strings.TrimSpace(mustExec(t, name, "readlink", "/opt/vpn-node/current")); got != "releases/"+rel2.ReleaseID {
		t.Fatalf("current %s", got)
	}
	if sha(t, name, "/var/lib/vpn-node/identity/key.pem") != key1 || incarnation(t, name) != inc1 {
		t.Fatal("newer install changed identity")
	}
	usageBefore := sha(t, name, "/var/lib/vpn-node/state/usage.db")
	writeStdin(t, name, "/run/w5-token2", token2+"\n", "0600")
	reenrollOut, reenrollCode := dexec(t, name, "/usr/local/bin/fvpn-node-install", "reenroll", "--json", "--token-file", "/run/w5-token2")
	if reenrollCode != 0 || !strings.Contains(reenrollOut, `"status": "reenrolled"`) {
		t.Fatalf("reenroll %d\n%s", reenrollCode, reenrollOut)
	}
	var reenrolled installer.Result
	if err := decodeJSON(reenrollOut, &reenrolled); err != nil {
		t.Fatal(err)
	}
	assertBaseline(t, reenrolled.Baseline, reenrollOut)
	inc2 := incarnation(t, name)
	if inc2 == inc1 || sha(t, name, "/var/lib/vpn-node/identity/key.pem") == key1 {
		t.Fatal("reenroll did not replace the identity")
	}
	if sha(t, name, "/var/lib/vpn-node/state/usage.db") != usageBefore {
		t.Fatal("reenroll changed the usage store")
	}
	retired := strings.TrimSpace(mustExec(t, name, "sh", "-c", "stat -c %a /var/lib/vpn-node/identity.retired/*/"))
	if !strings.Contains(retired, "700") {
		t.Fatalf("retired mode %s", retired)
	}
	if _, code := dexec(t, name, "test", "!", "-e", "/run/w5-token2"); code != 0 {
		t.Fatal("reenroll token was not consumed")
	}
	rmOut, rmCode := dexec(t, name, "/usr/local/bin/fvpn-node-install", "remove", "--json")
	if rmCode != 0 || !strings.Contains(rmOut, "identity, usage store and update journal kept") {
		t.Fatalf("remove %d\n%s", rmCode, rmOut)
	}
	if _, code := dexec(t, name, "test", "-f", "/var/lib/vpn-node/identity/key.pem"); code != 0 {
		t.Fatal("remove deleted identity")
	}
	if _, code := dexec(t, name, "test", "-f", "/var/lib/vpn-node/state/usage.db"); code != 0 {
		t.Fatal("remove deleted the usage store")
	}
	if _, code := dexec(t, name, "test", "-e", "/etc/systemd/system/fvpn-agent.service"); code == 0 {
		t.Fatal("remove left the unit file")
	}
	if _, code := dexec(t, name, "nft", "list", "table", "inet", "fvpn"); code == 0 {
		t.Fatal("remove left inet fvpn")
	}
	assertForeign(t, name, defRoute, resolv)
	analyze, analyzeCode := dexec(t, name, "systemd-analyze", "verify",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-agent.service",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-watchdog.service",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-updater.service",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-updater.timer",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-updater-recover.service",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-netbird.service",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-node-netbird-up.service",
		"/opt/vpn-node/releases/"+rel2.ReleaseID+"/lib/systemd/fvpn-node-firstboot.service")
	if analyzeCode != 0 || strings.Contains(analyze, "Unknown key") || strings.Contains(analyze, "Failed to parse") {
		t.Fatalf("systemd-analyze %d\n%s", analyzeCode, analyze)
	}
}

func TestRemovePurge(t *testing.T) {
	const name = "fvpn-w5-installer-purge"
	startSystemd(t, name)
	stage(t, name)
	url, ca, bundle := startControl(t, name)
	writeStdin(t, name, "/run/w5-token", token1+"\n", "0600")
	res, out, code := doInstall(t, name, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 || res.Status != "installed" {
		t.Fatalf("install %d %+v\n%s", code, res, out)
	}
	rmOut, rmCode := dexec(t, name, "/usr/local/bin/fvpn-node-install", "remove", "--purge", "--json")
	if rmCode != 0 || !strings.Contains(rmOut, "purge:") {
		t.Fatalf("purge %d\n%s", rmCode, rmOut)
	}
	for _, p := range []string{"/var/lib/vpn-node/identity", "/var/lib/vpn-node/state/usage.db", "/var/lib/vpn-node/update-journal", "/opt/vpn-node"} {
		if _, code := dexec(t, name, "test", "-e", p); code == 0 {
			t.Fatalf("purge left %s", p)
		}
	}
}

func TestImagePrepareTwoBoots(t *testing.T) {
	const src = "fvpn-w5-installer-image-src"
	startSystemd(t, src)
	stage(t, src)
	url, ca, bundle := startControl(t, src)
	writeStdin(t, src, "/run/w5-token", token1+"\n", "0600")
	res, out, code := doInstall(t, src, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 || res.Status != "installed" {
		t.Fatalf("install %d %+v\n%s", code, res, out)
	}
	mustExec(t, src, "mkdir", "-p", "/etc/ssh", "/var/lib/netbird")
	writeStdin(t, src, "/etc/ssh/ssh_host_ed25519_key", "test-host-key\n", "0600")
	writeStdin(t, src, "/var/lib/netbird/private.key", "test-netbird-key\n", "0600")
	refuse, rcode := dexec(t, src, "/usr/local/bin/fvpn-node-install", "image-prepare")
	if rcode != installer.ExitFailed || !strings.Contains(refuse, "identity_present") {
		t.Fatalf("image-prepare without flag %d\n%s", rcode, refuse)
	}
	if _, code := dexec(t, src, "test", "-f", "/var/lib/vpn-node/identity/key.pem"); code != 0 {
		t.Fatal("refused image-prepare deleted the key")
	}
	prep, pcode := dexec(t, src, "/usr/local/bin/fvpn-node-install", "image-prepare", "--i-am-building-an-image", "--json")
	if pcode != 0 || !strings.Contains(prep, `"status": "prepared"`) {
		t.Fatalf("image-prepare %d\n%s", pcode, prep)
	}
	for _, p := range []string{
		"/var/lib/vpn-node/identity/key.pem",
		"/var/lib/vpn-node/identity/node.json",
		"/var/lib/vpn-node/state/usage.db",
		"/etc/ssh/ssh_host_ed25519_key",
		"/var/lib/netbird/private.key",
	} {
		if _, code := dexec(t, src, "test", "-e", p); code == 0 {
			t.Fatalf("image-prepare left %s", p)
		}
	}
	mid := strings.TrimSpace(mustExec(t, src, "cat", "/etc/machine-id"))
	if mid != "" {
		t.Fatalf("machine-id %q", mid)
	}
	if _, code := dexec(t, src, "test", "-x", "/opt/vpn-node/current/fvpn-agent"); code != 0 {
		t.Fatal("image-prepare removed the agent")
	}
	if _, code := dexec(t, src, "test", "-f", "/etc/systemd/system/fvpn-node-firstboot.service"); code != 0 {
		t.Fatal("image-prepare removed the first-boot unit")
	}
	if _, code := runCmd(t, time.Minute, "docker", "stop", src); code != 0 {
		t.Fatal("stop before commit")
	}
	commitOut, commitCode := runCmd(t, 2*time.Minute, "docker", "commit", src, "fvpn-w5-installer-image:prepared")
	if commitCode != 0 {
		t.Fatalf("commit %s", commitOut)
	}
	a := bootClone(t, "fvpn-w5-installer-boot-a")
	b := bootClone(t, "fvpn-w5-installer-boot-b")
	if a.key == b.key || a.inc == b.inc || a.machine == b.machine {
		t.Fatalf("clones are not distinct\n%+v\n%+v", a, b)
	}
	if a.machine == "" || b.machine == "" {
		t.Fatalf("machine-id was not regenerated\n%+v\n%+v", a, b)
	}
}

type bootID struct{ key, inc, machine string }

func bootClone(t *testing.T, name string) bootID {
	t.Helper()
	_ = exec.Command("docker", "rm", "-fv", name).Run()
	out, code := runCmd(t, time.Minute, "docker", "run", "-d", "--name", name,
		"--privileged", "--cgroupns=host",
		"-v", "/sys/fs/cgroup:/sys/fs/cgroup:rw",
		"fvpn-w5-installer-image:prepared")
	if code != 0 {
		t.Fatalf("clone %s: %s", name, out)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-fv", name).Run() })
	if !waitSystemd(t, name, 45*time.Second) {
		_ = exec.Command("docker", "rm", "-fv", name).Run()
		out, code = runCmd(t, time.Minute, "docker", "run", "-d", "--name", name,
			"--privileged", "--cgroupns=host",
			"-v", "/sys/fs/cgroup:/sys/fs/cgroup:rw",
			"fvpn-w5-installer-image:prepared")
		if code != 0 {
			t.Fatalf("clone retry %s: %s", name, out)
		}
		if !waitSystemd(t, name, 45*time.Second) {
			logs, _ := runCmd(t, 20*time.Second, "docker", "logs", name)
			t.Fatalf("clone systemd did not come up\n%s", logs)
		}
	}
	_, _ = dexec(t, name, "systemctl", "stop",
		"fvpn-agent.service", "fvpn-watchdog.service", "fvpn-updater.timer",
		"fvpn-node-netbird-up.service", "fvpn-netbird.service")
	url, _, _ := startControl(t, name)
	rewriteControl(t, name, url)
	writeStdin(t, name, "/var/lib/vpn-node/firstboot.token", token1+"\n", "0600")
	fb, fbCode := dexec(t, name, "/usr/local/bin/fvpn-node-install", "firstboot", "--json")
	if fbCode != 0 || !strings.Contains(fb, `"status": "enrolled"`) {
		logs, _ := dexec(t, name, "journalctl", "-u", "fvpn-agent", "-u", "fvpn-node-firstboot", "--no-pager", "-n", "80")
		t.Fatalf("firstboot %d\n%s\n%s", fbCode, fb, logs)
	}
	return bootID{
		key:     sha(t, name, "/var/lib/vpn-node/identity/key.pem"),
		inc:     incarnation(t, name),
		machine: strings.TrimSpace(mustExec(t, name, "cat", "/etc/machine-id")),
	}
}

func rewriteControl(t *testing.T, c, url string) {
	t.Helper()
	raw := mustExec(t, c, "cat", "/etc/vpn-node/agent.json")
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	m["control_url"] = url
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeStdin(t, c, "/etc/vpn-node/agent.json", string(body)+"\n", "0644")
	mustExec(t, c, "cp", "/run/w5-control/ca.pem", "/etc/vpn-node/ca.pem")
	mustExec(t, c, "cp", "/run/w5-control/trust.json", "/etc/vpn-node/trust.json")
}

func mustCheck(t *testing.T, c string, want int) installer.CheckReport {
	t.Helper()
	out, code := dexec(t, c, "/usr/local/bin/fvpn-node-install", "check", "--json")
	if code != want {
		t.Fatalf("check exit %d want %d\n%s", code, want, out)
	}
	var rep installer.CheckReport
	if err := decodeJSON(out, &rep); err != nil {
		t.Fatalf("check json: %v\n%s", err, out)
	}
	return rep
}

// decodeJSON reads the first JSON value. check --json also prints a
// one-line error after the object when the host has drifted.
func decodeJSON(out string, dest any) error {
	i := strings.Index(out, "{")
	if i < 0 {
		return fmt.Errorf("no json")
	}
	dec := json.NewDecoder(strings.NewReader(out[i:]))
	return dec.Decode(dest)
}

func waitPath(t *testing.T, c, path string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, code := dexec(t, c, "test", "-f", path); code == 0 {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("missing %s", path)
}

func TestBootstrap(t *testing.T) {
	for _, tool := range []string{"zstd", "tar", "sha256sum", "curl"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("host is missing %s, which the bootstrap needs", tool)
		}
	}
	bin, err := os.ReadFile(filepath.Join(binDir, "fvpn-node-install"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bin)
	pin := hex.EncodeToString(sum[:])
	trust, err := os.ReadFile(rel1.Trust)
	if err != nil {
		t.Fatal(err)
	}
	scriptRaw, err := os.ReadFile(filepath.Join(rootDir, "deploy/node/install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/manifest.json":
			http.ServeFile(w, r, rel1.Manifest)
		case "/artifact.tar.zst":
			http.ServeFile(w, r, rel1.Artifact)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.sh")
	if err := os.WriteFile(plain, scriptRaw, 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := runCmd(t, 30*time.Second, "sh", plain, "--manifest-url", "https://releases.example.invalid/latest", "--artifact-url", "https://releases.example.invalid/a.tar.zst", "--out", dir, "--", "version")
	if code != 64 || !strings.Contains(out, "installer pin") {
		t.Fatalf("placeholder %d\n%s", code, out)
	}
	ready := bytes.ReplaceAll(scriptRaw, []byte("@@INSTALLER_SHA256@@"), []byte(pin))
	ready = bytes.ReplaceAll(ready, []byte("@@TRUST_ROOT@@"), bytes.TrimSpace(trust))
	path := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(path, ready, 0o755); err != nil {
		t.Fatal(err)
	}
	out, code = runCmd(t, 30*time.Second, "sh", path, "--manifest-url", srv.URL+"/manifest.json", "--artifact-url", srv.URL+"/latest", "--out", filepath.Join(dir, "nope"), "--", "version")
	if code != 64 || !strings.Contains(out, "latest") || hits != 0 {
		t.Fatalf("latest %d hits %d\n%s", code, hits, out)
	}
	outDir := filepath.Join(dir, "out")
	out, code = runCmd(t, 2*time.Minute, "sh", path,
		"--manifest-url", srv.URL+"/manifest.json",
		"--artifact-url", srv.URL+"/artifact.tar.zst",
		"--out", outDir, "--", "version")
	if code != 0 || !strings.Contains(out, "fvpn-node-install dev") {
		t.Fatalf("bootstrap %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(outDir, "fvpn-node-install")); err != nil {
		t.Fatal(err)
	}
}
