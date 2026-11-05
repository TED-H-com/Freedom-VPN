package isolation

// TEST ONLY read-only container snapshot. Called by the short-lived probe, never
// installed as a service or proxy. No identity file, environment or arbitrary
// argv is exported. Bounds describe reads, not the bytes loaded at process start.
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const runtimeBase = "/var/lib/vpn-node/e2e/"
const runtimeReadLimit = 1 << 20

type RuntimeRead struct {
	Error    string `json:"error,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Start    string `json:"read_start"`
	End      string `json:"read_end"`
	Status   string `json:"status"`
	Text     string `json:"text,omitempty"`
	Bytes    int    `json:"bytes"`
	Size     int64  `json:"size"`
	Inode    uint64 `json:"inode"`
	SHA256   string `json:"sha256,omitempty"`
	Modified string `json:"modified_at,omitempty"`
}

func runtimeNow() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func runtimeRead(path string) RuntimeRead {
	r := RuntimeRead{Start: runtimeNow(), Status: "unavailable"}
	f, err := os.Open(path)
	if err != nil {
		r.Error = "file read unavailable"
		if errors.Is(err, os.ErrNotExist) {
			r.Error = "file absent"
		}
		if errors.Is(err, os.ErrPermission) {
			r.Error = "file permission denied"
		}
	}
	if err == nil {
		defer f.Close()
		before, e := f.Stat()
		if e == nil && before.Mode().IsRegular() {
			r.Size, r.Modified = before.Size(), before.ModTime().UTC().Format(time.RFC3339Nano)
			if st, ok := before.Sys().(*syscall.Stat_t); ok {
				r.Inode = st.Ino
			}
			raw, e := io.ReadAll(io.LimitReader(f, runtimeReadLimit+1))
			r.Status = "available"
			if e != nil {
				r.Status = "incomplete"
			}
			if len(raw) > runtimeReadLimit {
				raw = raw[:runtimeReadLimit]
				r.Status = "truncated"
			}
			after, e := f.Stat()
			if e != nil || after.Size() < before.Size() || !after.ModTime().Equal(before.ModTime()) {
				r.Status = "changed during read"
			}
			if current, e := os.Stat(path); e != nil || !os.SameFile(before, current) {
				r.Status = "path replaced during read"
			}
			r.Bytes, r.Text = len(raw), Redact(string(raw))
			digest := sha256.Sum256(raw)
			r.SHA256 = hex.EncodeToString(digest[:])
		}
	}
	r.End = runtimeNow()
	return r
}

func approvedRuntimeConfig(path string) bool {
	if path == "/etc/vpn-node/agent-run.json" {
		return true
	}
	return filepath.Dir(path) == strings.TrimSuffix(runtimeBase, "/") && strings.HasPrefix(filepath.Base(path), "agent-") && strings.HasSuffix(path, ".json") && filepath.Base(path) != "agent-.json"
}

// projectRuntimeConfig never returns unapproved config fields, even on errors.
func projectRuntimeConfig(path string, r RuntimeRead) map[string]any {
	out := map[string]any{"path": Redact(path), "read_start": r.Start, "read_end": r.End, "status": r.Status, "sha256": r.SHA256, "modified_at": r.Modified, "bytes": r.Bytes, "inode": r.Inode, "scope": "current file only; loaded bytes at launch not attested"}
	if !approvedRuntimeConfig(path) {
		out["status"] = "unapproved path"
		return out
	}
	var cfg map[string]json.RawMessage
	if r.Status != "available" {
		return out
	}
	if json.Unmarshal([]byte(r.Text), &cfg) != nil || cfg == nil {
		out["status"] = "parse error"
		return out
	}
	safe := map[string]json.RawMessage{}
	for _, key := range []string{"control_url", "heartbeat_seconds", "poll_seconds", "observation_seconds"} {
		if v, ok := cfg[key]; ok {
			safe[key] = v
		}
	}
	var dp map[string]json.RawMessage
	if json.Unmarshal(cfg["datapath"], &dp) != nil {
		out["status"] = "parse error"
		return out
	}
	data := map[string]json.RawMessage{}
	for _, key := range []string{"member_ingress", "uplink", "snat"} {
		if v, ok := dp[key]; ok {
			data[key] = v
		}
	}
	out["fields"], out["datapath"] = safe, data
	return out
}

// approvedProcess drops the complete argv unless its shape is exactly known.
func approvedProcess(raw []byte) (role, binary, config string) {
	args := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	if len(args) == 4 && args[1] == "run" && args[2] == "-config" && approvedRuntimeConfig(args[3]) {
		base := filepath.Base(args[0])
		if base == "fvpn-agent" || base == "fvpn-watchdog" {
			return base, args[0], args[3]
		}
	}
	if len(args) == 2 && args[1] == "resolver-proxy" && (filepath.Base(args[0]) == "fvpn-e2e-nodeconfig" || filepath.Base(args[0]) == "fvpn-e2e-nodeconfig-original") {
		return "resolver-proxy", args[0], ""
	}
	return "", "", ""
}

func runtimeProcess(pid string) map[string]any {
	p, err := strconv.Atoi(pid)
	row := map[string]any{"pid": p, "status": "unavailable"}
	if err != nil || p <= 1 {
		return row
	}
	base := "/proc/" + pid + "/"
	stat := runtimeRead(base + "stat")
	at := strings.LastIndex(stat.Text, ")")
	if at < 0 {
		return row
	}
	fields := strings.Fields(stat.Text[at+1:])
	if len(fields) < 20 || fields[0] == "Z" {
		return row
	}
	role, binary, config := approvedProcess([]byte(runtimeRead(base + "cmdline").Text))
	exe, e := os.Readlink(base + "exe")
	if e != nil || role == "" {
		row["status"] = "unapproved or exited process"
		return row
	}
	row["status"], row["start_ticks"], row["role"] = "available", fields[19], role
	row["executable"], row["command_binary"], row["config_path"] = Redact(exe), Redact(binary), Redact(config)
	row["command"] = "run -config"
	if role == "resolver-proxy" {
		row["command"] = "resolver-proxy"
	}
	if st, e := os.Stat(base + "fd/1"); e == nil {
		if s, ok := st.Sys().(*syscall.Stat_t); ok {
			row["stdout_inode"] = s.Ino
		}
	}
	return row
}

func runtimeDigest(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, 128<<20))
	if err != nil || n == 128<<20 {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func runtimeCommand(ctx context.Context, args ...string) RuntimeRead {
	r := RuntimeRead{Start: runtimeNow(), Status: "unavailable"}
	commandCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, args[0], args[1:]...)
	// These fixed readbacks are small; cap both streams and retain overflow as incomplete.
	w := &runtimeBuffer{}
	cmd.Stdout = w
	cmd.Stderr = w
	err := cmd.Run()
	code := 0
	if err != nil {
		code = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		}
		r.Error = "readback command failed"
	}
	if commandCtx.Err() != nil {
		r.Error = "readback deadline exceeded"
	}
	r.ExitCode = &code
	r.Text, r.Bytes = Redact(w.text), len(w.text)
	if err == nil {
		r.Status = "available"
	}
	if w.overflow {
		r.Status = "truncated"
	}
	r.End = runtimeNow()
	return r
}

type runtimeBuffer struct {
	text     string
	overflow bool
}

func (w *runtimeBuffer) Write(p []byte) (int, error) {
	n := len(p)
	left := runtimeReadLimit - len(w.text)
	if len(p) > left {
		p = p[:left]
		w.overflow = true
	}
	w.text += string(p)
	return n, nil
}

// CollectRuntimeEvidence has no waits, signals, writes or active network probes.
// It is deliberately best effort: unavailable evidence cannot abort teardown.
func CollectRuntimeEvidence(ctx context.Context) map[string]any {
	start := runtimeNow()
	selected := runtimeRead(runtimeBase + "agent-selected")
	pid := runtimeRead(runtimeBase + "agent.pid")
	launch := runtimeRead(runtimeBase + "agent-process-id")
	before := runtimeProcess(strings.TrimSpace(pid.Text))
	configs := map[string]any{}
	paths := []string{"/etc/vpn-node/agent-run.json"}
	release := strings.TrimSpace(selected.Text)
	if strings.HasPrefix(release, "/opt/vpn-node/releases/") && filepath.Dir(release) == "/opt/vpn-node/releases" {
		paths = append(paths, runtimeBase+"agent-"+filepath.Base(release)+".json")
	}
	if path, ok := before["config_path"].(string); ok && approvedRuntimeConfig(path) {
		paths = append(paths, path)
	}
	for _, path := range paths {
		if _, ok := configs[path]; !ok {
			configs[path] = projectRuntimeConfig(path, runtimeRead(path))
		}
	}
	executableDigest := ""
	if before["status"] == "available" {
		executableDigest = runtimeDigest(fmt.Sprintf("/proc/%v/exe", before["pid"]))
	}
	services := []map[string]any{}
	dirs, _ := filepath.Glob("/proc/[0-9]*")
	for _, d := range dirs {
		raw := runtimeRead(d + "/cmdline")
		role, _, _ := approvedProcess([]byte(raw.Text))
		if role == "resolver-proxy" || role == "fvpn-watchdog" {
			services = append(services, runtimeProcess(filepath.Base(d)))
		}
	}
	logs := map[string]RuntimeRead{}
	for _, name := range []string{"agent", "watchdog", "resolver-proxy"} {
		logs[name] = runtimeRead(runtimeBase + name + ".log")
	}
	network := map[string]RuntimeRead{}
	for _, command := range []struct {
		name string
		args []string
	}{
		{"addresses", []string{"ip", "-j", "address", "show"}},
		{"routes", []string{"ip", "-j", "route", "show", "table", "all"}},
		{"rules", []string{"ip", "-j", "rule", "show"}},
		{"ruleset", []string{"nft", "list", "table", "inet", "fvpn"}},
		{"proxy_listener", []string{"ss", "-Hlnpt", "sport = :9444"}},
		{"transport_sockets", []string{"ss", "-Hntp", "( sport = :9444 or dport = :9444 or dport = :9443 )"}},
	} {
		network[command.name] = runtimeCommand(ctx, command.args...)
	}
	heartbeat := runtimeRead("/run/vpn-node/heartbeat")
	hostRead := runtimeRead("/etc/hosts")
	controlHosts := []string{}
	if hostRead.Status == "available" {
		for _, line := range strings.Split(hostRead.Text, "\n") {
			fields := strings.Fields(strings.SplitN(line, "#", 2)[0])
			if len(fields) < 2 {
				continue
			}
			for _, name := range fields[1:] {
				if name == "agent.example.invalid" {
					controlHosts = append(controlHosts, fields[0])
				}
			}
		}
	}
	controlHostRead := map[string]any{"status": hostRead.Status, "read_start": hostRead.Start, "read_end": hostRead.End, "name": "agent.example.invalid", "addresses": controlHosts}
	after := runtimeProcess(strings.TrimSpace(pid.Text))
	stable := selected.Status == "available" && pid.Status == "available" && launch.Status == "available" && strings.TrimSpace(launch.Text) != "" && before["status"] == "available" && reflect.DeepEqual(before, after) && pid.Text == runtimeRead(runtimeBase+"agent.pid").Text && launch.Text == runtimeRead(runtimeBase+"agent-process-id").Text && selected.Text == runtimeRead(runtimeBase+"agent-selected").Text
	for path, value := range configs {
		row := value.(map[string]any)
		last := projectRuntimeConfig(path, runtimeRead(path))
		row["stable_during_snapshot"] = row["status"] == "available" && last["status"] == "available" && row["sha256"] == last["sha256"] && row["inode"] == last["inode"] && row["modified_at"] == last["modified_at"]
	}
	if stable && executableDigest != "" {
		stable = executableDigest == runtimeDigest(fmt.Sprintf("/proc/%v/exe", before["pid"]))
	} else {
		stable = false
	}
	return map[string]any{"read_start": start, "read_end": runtimeNow(), "agent_selected": selected, "agent_pid": pid, "agent_process_id": launch, "process_before": before, "process_after": after, "process_coherent": stable, "executable_sha256": executableDigest, "configs": configs, "services": services, "logs": logs, "network": network, "heartbeat": heartbeat, "control_host": controlHostRead, "scope": "read bounds only; file digests do not attest earlier loaded bytes; copied log history is not a new launch; watchdog is observed, not stopped"}
}

// CheckContainerScratch runs before writing the existing probe path. Docker's
// backing filesystem is also checked by the launch preflight; tmpfs is refused.
func CheckContainerScratch(ctx context.Context, container string) error {
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	raw, err := exec.CommandContext(readCtx, "docker", "exec", container, "stat", "-f", "-c", "%T", "/tmp").Output()
	if err != nil {
		return fmt.Errorf("container scratch read unavailable")
	}
	switch strings.TrimSpace(string(raw)) {
	case "overlayfs", "ext2/ext3", "xfs", "btrfs":
		return nil
	}
	return fmt.Errorf("container scratch filesystem not established as disk-backed")
}
