//go:build netlab

package netlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

const (
	// TEST ONLY. These are not product plan values (§11.3, §11.7).
	testRateBPS = uint64(20_000_000)
	testCapBPS  = uint64(20_000_000)
)

var (
	memberA = uuid.MustParse("019a0000-0000-7000-8000-00000000000a")
	memberB = uuid.MustParse("019a0000-0000-7000-8000-00000000000b")
	devA1   = uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	devA2   = uuid.MustParse("019a0000-0000-7000-8000-0000000000a2")
	devB1   = uuid.MustParse("019a0000-0000-7000-8000-0000000000b1")
)

func prefix() string {
	if v := os.Getenv("FVPN_LAB_PREFIX"); v != "" {
		return v
	}
	return "fvpn-netlab"
}

func skipLab(t *testing.T) {
	t.Helper()
	if os.Getenv("FVPN_NETLAB") != "1" {
		t.Skip("netlab: set FVPN_NETLAB=1 to run the privileged lab; the container must already be up")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("netlab: docker is not installed")
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "permission denied") || strings.Contains(lower, "cannot connect") || strings.Contains(lower, "got permission denied") {
			t.Skip("netlab: docker is not usable: " + msg)
		}
		t.Fatalf("netlab: FVPN_NETLAB=1 but container %s is not running: %s", prefix(), msg)
	}
	if msg != "true" {
		t.Fatalf("netlab: container %s is not running", prefix())
	}
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(wd) == "netlab" {
		return filepath.Clean(filepath.Join(wd, "../.."))
	}
	return wd
}

func dockerExec(args ...string) (string, string, error) {
	cmd := exec.Command("docker", append([]string{"exec", prefix()}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func dockerExecStdin(args []string, stdin []byte) (string, string, error) {
	cmd := exec.Command("docker", append([]string{"exec", "-i", prefix()}, args...)...)
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func dockerDetach(args ...string) error {
	cmd := exec.Command("docker", append([]string{"exec", "-d", prefix()}, args...)...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, errb.String())
	}
	return nil
}

func mustExec(t *testing.T, args ...string) string {
	t.Helper()
	out, errOut, err := dockerExec(args...)
	if err != nil {
		t.Fatalf("docker exec %s: %v\n%s%s", strings.Join(args, " "), err, out, errOut)
	}
	return out
}

func execText(args ...string) string {
	out, errOut, err := dockerExec(args...)
	if err != nil {
		return out + errOut + "\n" + err.Error()
	}
	return out
}

func publicLeak(s string) bool {
	// Split so the source does not contain a home-directory literal.
	return strings.Contains(s, "/home"+"/") || strings.Contains(s, "PRIVATE KEY")
}

// evidenceRoot is the test's temporary directory unless the caller sets
// FVPN_NETLAB_EVIDENCE_DIR. The make target that refreshes docs/poc/G2-evidence
// sets that variable. A normal lab run must not dirty the tracked tree.
func evidenceRoot(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("FVPN_NETLAB_EVIDENCE_DIR"); dir != "" {
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(repoRoot(), dir)
		}
		return dir
	}
	return t.TempDir()
}

func save(t *testing.T, name, body string) {
	t.Helper()
	if publicLeak(body) {
		t.Fatalf("evidence %s failed public hygiene", name)
	}
	dir := filepath.Join(evidenceRoot(t), t.Name())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func noteSummary(line string) {
	fmt.Printf("SUMMARY %s\n", line)
	dir := os.Getenv("FVPN_NETLAB_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoRoot(), dir)
	}
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "SUMMARY.txt"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, line)
}

func debugPlane() string {
	var b strings.Builder
	for _, args := range [][]string{
		{"nft", "list", "table", "inet", "fvpn"},
		{"tc", "-s", "qdisc", "show", "dev", "fv-ih"},
		{"tc", "-s", "class", "show", "dev", "fv-ih"},
		{"tc", "-s", "class", "show", "dev", "fv-fh"},
	} {
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", strings.Join(args, " "), execText(args...))
	}
	return b.String()
}

type applyDoc struct {
	Generation  uint64            `json:"generation"`
	Admitted    int               `json:"admitted"`
	Resets      []enforce.Reset   `json:"resets"`
	Finals      []enforce.Sample  `json:"finals"`
	Samples     []enforce.Sample  `json:"samples"`
	Assignments map[string]string `json:"assignments"`
}

type counterDoc struct {
	Counters map[string]enforce.CounterValue `json:"counters"`
	Samples  []enforce.Sample                `json:"samples"`
}

func applyAt(t *testing.T, m enforce.Model) applyDoc {
	t.Helper()
	return applyPrev(t, m, "")
}

func applyPrev(t *testing.T, m enforce.Model, prevPath string) applyDoc {
	t.Helper()
	if m.Now.IsZero() {
		m.Now = time.Now().UTC()
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"fvpn-netlab", "apply"}
	if t.Name() == "TestAT10_AT13_ReloadRevokeBaseline" {
		args = append(args, "-trace", "/scratch/at10-apply.jsonl")
	}
	if prevPath != "" {
		args = append(args, "-prev", prevPath)
	}
	out, errOut, err := dockerExecStdin(args, raw)
	if err != nil {
		save(t, "apply-failure.txt", out+"\n"+errOut+"\n"+debugPlane())
		t.Fatalf("apply generation %d: %v\n%s\n%s", m.Generation, err, errOut, out)
	}
	var doc applyDoc
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("apply json: %v\n%s", jerr, out)
	}
	return doc
}

func applyBaseline(t *testing.T) {
	t.Helper()
	out, errOut, err := dockerExec("fvpn-netlab", "baseline")
	if err != nil {
		save(t, "baseline-failure.txt", out+"\n"+errOut+"\n"+debugPlane())
		t.Fatalf("baseline: %v\n%s%s", err, out, errOut)
	}
}

func writeScratch(t *testing.T, path string, body []byte) {
	t.Helper()
	out, errOut, err := dockerExecStdin([]string{"tee", path}, body)
	if err != nil {
		t.Fatalf("tee %s: %v\n%s%s", path, err, out, errOut)
	}
}

func readScratch(t *testing.T, path string) string {
	t.Helper()
	out, errOut, err := dockerExec("cat", path)
	if err != nil {
		t.Fatalf("cat %s: %v %s", path, err, errOut)
	}
	return out
}

func scratchExists(path string) bool {
	_, _, err := dockerExec("test", "-f", path)
	return err == nil
}

type idBind struct {
	member uuid.UUID
	device uuid.UUID
	peer   string
	ip     string
}

func allBinds() []idBind {
	return []idBind{
		{memberA, devA1, "peer-a1", "100.64.1.1"},
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}
}

func functionalModel(now time.Time, gen uint64) enforce.Model {
	return deviceModel(now, gen, allBinds(), generousQoS(), 48*time.Hour)
}

func deviceModel(now time.Time, gen uint64, binds []idBind, qos enforce.QoS, leaseFor time.Duration) enforce.Model {
	var bindings []enforce.Binding
	seen := map[uuid.UUID]bool{}
	var leases []enforce.MemberLease
	for _, b := range binds {
		bindings = append(bindings, enforce.Binding{
			MemberID:        b.member,
			DeviceID:        b.device,
			PeerID:          b.peer,
			AllowedV4:       []netip.Prefix{mustPrefix(b.ip + "/32")},
			BindingRevision: 1,
			ValidFrom:       now.Add(-time.Minute),
			ValidUntil:      now.Add(leaseFor),
		})
		if !seen[b.member] {
			seen[b.member] = true
			leases = append(leases, enforce.MemberLease{
				MemberID:            b.member,
				EffectiveServiceEnd: now.Add(leaseFor),
				NodeLeaseEnd:        now.Add(leaseFor),
				SignedServerTime:    now,
				MaxOfflineSeconds:   3600,
			})
		}
	}
	return enforce.Model{
		Generation: gen,
		IPv6Mode:   enforce.IPv6Block,
		Now:        now,
		Datapath: enforce.Datapath{
			MemberIngress: "fv-fh",
			Uplink:        "fv-ih",
			SNAT:          mustIP("192.0.2.37"),
		},
		Bindings:  bindings,
		Leases:    leases,
		Resolvers: []netip.Addr{mustIP("198.51.100.53")},
		// 203.0.113.0/24 is the simulated Internet inside this lab only.
		// Production callers leave DeniedV4 nil and keep the spec list (§8.1).
		DeniedV4:      enforce.Without(enforce.SpecDeniedV4(), mustPrefix("203.0.113.0/24")),
		ExtraDeniedV4: []netip.Prefix{mustPrefix("192.0.2.200/32")},
		HostV4: []netip.Prefix{
			mustPrefix("192.0.2.33/32"),
			mustPrefix("192.0.2.37/32"),
			mustPrefix("192.168.0.1/32"),
		},
		QoS: qos,
	}
}

func generousQoS() enforce.QoS {
	wide := enforce.Rate{RateBPS: 200_000_000, CeilBPS: 200_000_000}
	host := enforce.Rate{RateBPS: 5_000_000, CeilBPS: 200_000_000}
	member := enforce.Rate{RateBPS: 50_000_000, CeilBPS: 50_000_000}
	return enforce.QoS{
		Up: wide, Down: wide, HostUp: host, HostDown: host,
		Members: []enforce.MemberQoS{
			{MemberID: memberA, Up: member, Down: member},
			{MemberID: memberB, Up: member, Down: member},
		},
	}
}

func mustIP(s string) netip.Addr { return netip.MustParseAddr(s) }

func mustPrefix(s string) netip.Prefix { return netip.MustParsePrefix(s) }

func usageName(prefix string, id uuid.UUID) string {
	return prefix + strings.ReplaceAll(id.String(), "-", "")
}

func sampleBytes(samples []enforce.Sample, id uuid.UUID, dir string) uint64 {
	for _, s := range samples {
		if s.DeviceID == id && s.Direction == dir {
			return s.Bytes
		}
	}
	return 0
}

func counterBytes(c map[string]enforce.CounterValue, name string) uint64 {
	if c == nil {
		return 0
	}
	return c[name].Bytes
}

func counterPackets(c map[string]enforce.CounterValue, name string) uint64 {
	if c == nil {
		return 0
	}
	return c[name].Packets
}

func readCounters(t *testing.T, m enforce.Model) counterDoc {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/counters-model.json", raw)
	out, errOut, err := dockerExec("fvpn-netlab", "counters", "-f", "/scratch/counters-model.json")
	if err != nil {
		t.Fatalf("counters: %v\n%s%s", err, out, errOut)
	}
	var doc counterDoc
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("counters json: %v\n%s", jerr, out)
	}
	return doc
}

func flushConntrack(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("conntrack", "-F")
	_, _, _ = dockerExec("conntrack", "-F", "-f", "ipv6")
}

type probeCase struct {
	Name      string `json:"name"`
	NS        string `json:"ns"`
	Proto     string `json:"proto"`
	Dst       string `json:"dst,omitempty"`
	Port      int    `json:"port,omitempty"`
	Bind      string `json:"bind,omitempty"`
	Expect    string `json:"expect"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
	Src       string `json:"src,omitempty"`
	SrcMAC    string `json:"src_mac,omitempty"`
	DstMAC    string `json:"dst_mac,omitempty"`
	Iface     string `json:"iface,omitempty"`
	Size      int    `json:"size,omitempty"`
}

type probeResult struct {
	Name      string `json:"name"`
	NS        string `json:"ns"`
	Proto     string `json:"proto"`
	Dst       string `json:"dst"`
	Port      int    `json:"port"`
	Expect    string `json:"expect"`
	Observed  string `json:"observed"`
	OK        bool   `json:"ok"`
	ElapsedMS int    `json:"elapsed_ms"`
}

func runProbe(t *testing.T, cases []probeCase) []probeResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{"cases": cases})
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/cases.json", body)
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/probe.py", "/scratch/cases.json")
	if err != nil && !strings.Contains(err.Error(), "exit status 2") {
		t.Fatalf("probe: %v\n%s%s", err, out, errOut)
	}
	var doc struct {
		Results []probeResult `json:"results"`
	}
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("probe json: %v\n%s\n%s", jerr, out, errOut)
	}
	return doc.Results
}

func printProbe(t *testing.T, title string, results []probeResult) string {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", title)
	fmt.Fprintf(&b, "%-32s %-8s %-6s %-22s %-6s %-8s %-14s %s\n", "name", "ns", "proto", "dst", "port", "expect", "observed", "ok")
	failed := 0
	for _, r := range results {
		fmt.Fprintf(&b, "%-32s %-8s %-6s %-22s %-6d %-8s %-14s %t\n", r.Name, r.NS, r.Proto, r.Dst, r.Port, r.Expect, r.Observed, r.OK)
		if !r.OK {
			failed++
		}
	}
	fmt.Fprintf(&b, "probes=%d failed=%d\n", len(results), failed)
	fmt.Print(b.String())
	return b.String()
}

func requireProbe(t *testing.T, results []probeResult) {
	t.Helper()
	var bad []string
	for _, r := range results {
		if !r.OK {
			bad = append(bad, fmt.Sprintf("%s expect %s observed %s", r.Name, r.Expect, r.Observed))
		}
	}
	if len(bad) > 0 {
		save(t, "probe-debug.txt", debugPlane())
		t.Fatalf("probe mismatches:\n%s", strings.Join(bad, "\n"))
	}
}

func startCapture(t *testing.T, ns, iface, filter, path string) {
	t.Helper()
	// Readiness must belong to this capture, not an earlier use of the path.
	mustExec(t, "rm", "-f", path, path+".err", path+".pid")
	// Quote the filter. Parentheses are part of the tcpdump expression, and
	// an unquoted "(" is a shell subshell, so the capture file is never created.
	expr := ""
	if strings.TrimSpace(filter) != "" {
		expr = " " + strconv.Quote(filter)
	}
	// A short successful probe can finish before a buffered capture block is
	// delivered. Immediate mode preserves those packets when the capture stops;
	// line buffering alone only flushes packets already received by tcpdump.
	script := fmt.Sprintf("echo $$ > %s.pid; exec tcpdump --immediate-mode -n -l -s 96 -i %s%s > %s 2>%s.err", path, iface, expr, path, path)
	var err error
	if ns == "" || ns == "root" {
		err = dockerDetach("sh", "-c", script)
	} else {
		err = dockerDetach("ip", "netns", "exec", ns, "sh", "-c", script)
	}
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		errText := execText("sh", "-c", "cat "+path+".err 2>/dev/null || true")
		if strings.Contains(errText, "listening") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("capture %s did not become ready: %s", path, oneLine(errText))
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func captureBody(t *testing.T, path string) string {
	t.Helper()
	errText := execText("sh", "-c", "cat "+path+".err 2>/dev/null || true")
	if !strings.Contains(errText, "listening") {
		t.Fatalf("capture %s did not start: %s", path, oneLine(errText))
	}
	return readScratch(t, path)
}

func stopOneCapture(path string) {
	killPidFile(path + ".pid")
	time.Sleep(200 * time.Millisecond)
}

func stopCaptures(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("sh", "-c", "pids=$(pidof tcpdump || true); if [ -n \"$pids\" ]; then kill $pids; fi")
	time.Sleep(300 * time.Millisecond)
}

func killPidFile(path string) {
	_, _, _ = dockerExec("sh", "-c", "if [ -f "+path+" ]; then kill $(cat "+path+") 2>/dev/null || true; fi")
}

type linkBytes struct {
	RX uint64
	TX uint64
}

func linkStats(t *testing.T, ns, dev string) linkBytes {
	t.Helper()
	args := []string{"ip", "-s", "-j", "link", "show", "dev", dev}
	if ns != "" && ns != "root" {
		args = append([]string{"ip", "netns", "exec", ns}, args...)
	}
	out := mustExec(t, args...)
	if i := strings.Index(out, "["); i > 0 {
		out = out[i:]
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("link json %s: %v\n%s", dev, err, out)
	}
	if len(arr) == 0 {
		t.Fatalf("link %s missing", dev)
	}
	stats, _ := arr[0]["stats64"].(map[string]any)
	if stats == nil {
		stats, _ = arr[0]["stats"].(map[string]any)
	}
	return linkBytes{RX: statBytes(stats, "rx"), TX: statBytes(stats, "tx")}
}

func statBytes(stats map[string]any, dir string) uint64 {
	if stats == nil {
		return 0
	}
	side, _ := stats[dir].(map[string]any)
	if side == nil {
		return 0
	}
	switch v := side["bytes"].(type) {
	case float64:
		return uint64(v)
	case json.Number:
		n, _ := strconv.ParseUint(v.String(), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return n
	default:
		return 0
	}
}

type classStat struct {
	Handle  string
	Bytes   uint64
	Packets uint64
	Dropped uint64
}

var classRE = regexp.MustCompile(`(?m)^class htb (\S+)[^\n]*\n Sent (\d+) bytes (\d+) pkt \(dropped (\d+)`)

func classStats(t *testing.T, dev string) map[string]classStat {
	t.Helper()
	out := mustExec(t, "tc", "-s", "class", "show", "dev", dev)
	got := map[string]classStat{}
	for _, m := range classRE.FindAllStringSubmatch(out, -1) {
		b, _ := strconv.ParseUint(m[2], 10, 64)
		p, _ := strconv.ParseUint(m[3], 10, 64)
		d, _ := strconv.ParseUint(m[4], 10, 64)
		got[m[1]] = classStat{Handle: m[1], Bytes: b, Packets: p, Dropped: d}
	}
	if len(got) == 0 {
		t.Fatalf("no htb classes on %s:\n%s", dev, out)
	}
	return got
}

func cpuJiffies() (idle, total uint64) {
	out, _, err := dockerExec("sh", "-c", "awk '/^cpu /{print}' /proc/stat")
	if err != nil {
		return 0, 0
	}
	fields := strings.Fields(out)
	if len(fields) < 5 {
		return 0, 0
	}
	var sum uint64
	for _, f := range fields[1:] {
		n, _ := strconv.ParseUint(f, 10, 64)
		sum += n
	}
	idleN, _ := strconv.ParseUint(fields[4], 10, 64)
	return idleN, sum
}

func windows() (warm, measure time.Duration, reps int) {
	if os.Getenv("FVPN_NETLAB_FULL") == "1" {
		return 30 * time.Second, 120 * time.Second, 3
	}
	return 5 * time.Second, 20 * time.Second, 1
}

func minorOf(t *testing.T, doc applyDoc, id uuid.UUID) uint16 {
	t.Helper()
	raw, ok := doc.Assignments[id.String()]
	if !ok {
		t.Fatalf("no class assignment for %s in %+v", id, doc.Assignments)
	}
	n, err := strconv.ParseUint(raw, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	return uint16(n)
}

func classHandle(minor uint16) string {
	return fmt.Sprintf("1:%x", minor)
}

type blastResult struct {
	Sent int `json:"sent"`
	Got  int `json:"got"`
}

func blast(t *testing.T, ns, proto, dst string, port, count, size int) blastResult {
	t.Helper()
	out, errOut, err := dockerExec("ip", "netns", "exec", ns, "python3", "/opt/netlab/scripts/blast.py", proto, dst, strconv.Itoa(port), strconv.Itoa(count), strconv.Itoa(size))
	if err != nil {
		t.Fatalf("blast: %v\n%s%s", err, out, errOut)
	}
	var doc blastResult
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("blast json: %v\n%s", jerr, out)
	}
	return doc
}

func ifaceMAC(t *testing.T, ns, dev string) string {
	t.Helper()
	path := "/sys/class/net/" + dev + "/address"
	var out string
	if ns == "" || ns == "root" {
		out = mustExec(t, "cat", path)
	} else {
		out = mustExec(t, "ip", "netns", "exec", ns, "cat", path)
	}
	return strings.TrimSpace(out)
}

func logLines(t *testing.T, path string) int {
	t.Helper()
	out, errOut, err := dockerExec("sh", "-c", "if [ -f "+path+" ]; then wc -l < "+path+"; else echo 0; fi")
	if err != nil {
		t.Fatalf("wc %s: %v %s", path, err, errOut)
	}
	n, _ := strconv.Atoi(strings.TrimSpace(out))
	return n
}
