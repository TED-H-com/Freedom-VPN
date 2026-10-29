//go:build netlab

package at

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/at/isolation"
	"github.com/google/uuid"
)

const (
	memberA = "019a0000-0000-7000-8000-00000000000a"
	memberB = "019a0000-0000-7000-8000-00000000000b"
	devA1   = "019a0000-0000-7000-8000-0000000000a1"
	devA2   = "019a0000-0000-7000-8000-0000000000a2"
	devB1   = "019a0000-0000-7000-8000-0000000000b1"
)

func labPrefix() string {
	if v := os.Getenv("FVPN_LAB_PREFIX"); v != "" {
		return v
	}
	return "fvpn-w4-at-isolation"
}

func skipLab(t *testing.T) {
	t.Helper()
	if os.Getenv("FVPN_NETLAB") != "1" {
		t.Skip("set FVPN_NETLAB=1; the lab container must already be up")
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", labPrefix()).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil || msg != "true" {
		t.Fatalf("lab container %s is not running: %s", labPrefix(), msg)
	}
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd
		}
		dir = parent
	}
}

func evidenceDir() string {
	if dir := os.Getenv("FVPN_AT_EVIDENCE_DIR"); dir != "" {
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(repoRoot(), dir)
		}
		return dir
	}
	return filepath.Join(repoRoot(), "docs", "e2e", "at", "isolation")
}

func dockerExec(args ...string) (string, string, error) {
	cmd := exec.Command("docker", append([]string{"exec", labPrefix()}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func dockerExecStdin(args []string, stdin []byte) (string, string, error) {
	cmd := exec.Command("docker", append([]string{"exec", "-i", labPrefix()}, args...)...)
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func dockerDetach(args ...string) error {
	cmd := exec.Command("docker", append([]string{"exec", "-d", labPrefix()}, args...)...)
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

type applyDoc struct {
	Generation uint64 `json:"generation"`
	Admitted   int    `json:"admitted"`
}

func labModel(gen uint64, binds []bind, revoke []string) enforce.Model {
	now := time.Now().UTC()
	var bindings []enforce.Binding
	seen := map[uuid.UUID]bool{}
	var leases []enforce.MemberLease
	for _, b := range binds {
		member := uuid.MustParse(b.member)
		bindings = append(bindings, enforce.Binding{
			MemberID: member, DeviceID: uuid.MustParse(b.device), PeerID: b.peer,
			AllowedV4: []netip.Prefix{netip.MustParsePrefix(b.ip + "/32")}, BindingRevision: 1,
			ValidFrom: now.Add(-time.Minute), ValidUntil: now.Add(48 * time.Hour),
		})
		if !seen[member] {
			seen[member] = true
			leases = append(leases, enforce.MemberLease{
				MemberID: member, EffectiveServiceEnd: now.Add(48 * time.Hour), NodeLeaseEnd: now.Add(48 * time.Hour),
				SignedServerTime: now, MaxOfflineSeconds: 3600,
			})
		}
	}
	wide := enforce.Rate{RateBPS: 200_000_000, CeilBPS: 200_000_000}
	host := enforce.Rate{RateBPS: 5_000_000, CeilBPS: 200_000_000}
	memberRate := enforce.Rate{RateBPS: 50_000_000, CeilBPS: 50_000_000}
	var revoked []netip.Addr
	for _, ip := range revoke {
		revoked = append(revoked, netip.MustParseAddr(ip))
	}
	return enforce.Model{
		Generation: gen, IPv6Mode: enforce.IPv6Block, Now: now,
		Datapath: enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.37")},
		Bindings: bindings, Leases: leases,
		Resolvers:     []netip.Addr{netip.MustParseAddr("198.51.100.53")},
		DeniedV4:      enforce.Without(enforce.SpecDeniedV4(), netip.MustParsePrefix("203.0.113.0/24")),
		ExtraDeniedV4: []netip.Prefix{netip.MustParsePrefix("192.0.2.200/32")},
		HostV4:        []netip.Prefix{netip.MustParsePrefix("192.0.2.33/32"), netip.MustParsePrefix("192.0.2.37/32"), netip.MustParsePrefix("192.168.0.1/32")},
		QoS: enforce.QoS{
			Up: wide, Down: wide, HostUp: host, HostDown: host,
			Members: []enforce.MemberQoS{
				{MemberID: uuid.MustParse(memberA), Up: memberRate, Down: memberRate},
				{MemberID: uuid.MustParse(memberB), Up: memberRate, Down: memberRate},
			},
		},
		Revoke: revoked,
	}
}

type bind struct{ member, device, peer, ip string }

func allBinds() []bind {
	return []bind{
		{memberA, devA1, "peer-a1", "100.64.1.1"},
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}
}

func applyModel(t *testing.T, m enforce.Model) applyDoc {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	out, errOut, err := dockerExecStdin([]string{"fvpn-netlab", "apply"}, raw)
	if err != nil {
		t.Fatalf("apply generation %d: %v\n%s\n%s", m.Generation, err, errOut, out)
	}
	var doc applyDoc
	if json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("apply json: %s", out)
	}
	return doc
}

func applyBaseline(t *testing.T) {
	t.Helper()
	out, errOut, err := dockerExec("fvpn-netlab", "baseline")
	if err != nil {
		t.Fatalf("baseline: %v\n%s%s", err, out, errOut)
	}
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
}

type probeResult struct {
	Name     string `json:"name"`
	NS       string `json:"ns"`
	Proto    string `json:"proto"`
	Dst      string `json:"dst"`
	Port     int    `json:"port"`
	Expect   string `json:"expect"`
	Observed string `json:"observed"`
	OK       bool   `json:"ok"`
}

func runProbe(t *testing.T, cases []probeCase) []probeResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{"cases": cases})
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/at-cases.json", body)
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/probe.py", "/scratch/at-cases.json")
	if err != nil && !strings.Contains(err.Error(), "exit status 2") {
		t.Fatalf("probe: %v\n%s%s", err, out, errOut)
	}
	var doc struct {
		Results []probeResult `json:"results"`
	}
	if json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("probe json: %s\n%s", out, errOut)
	}
	return doc.Results
}

func matrixText(results []probeResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-28s %-8s %-6s %-22s %-6s %-8s %-14s %s\n", "name", "ns", "proto", "dst", "port", "expect", "observed", "ok")
	for _, r := range results {
		fmt.Fprintf(&b, "%-28s %-8s %-6s %-22s %-6d %-8s %-14s %t\n", r.Name, r.NS, r.Proto, r.Dst, r.Port, r.Expect, r.Observed, r.OK)
	}
	return b.String()
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

func startCapture(t *testing.T, ns, iface, filter, path string) {
	t.Helper()
	expr := ""
	if strings.TrimSpace(filter) != "" {
		expr = " " + strconv.Quote(filter)
	}
	script := fmt.Sprintf("echo $$ > %s.pid; exec tcpdump -n -l -s 96 -i %s%s > %s 2>%s.err", path, iface, expr, path, path)
	var err error
	if ns == "" || ns == "root" {
		err = dockerDetach("sh", "-c", script)
	} else {
		err = dockerDetach("ip", "netns", "exec", ns, "sh", "-c", script)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func stopCaptures() {
	_, _, _ = dockerExec("sh", "-c", "pids=$(pidof tcpdump || true); if [ -n \"$pids\" ]; then kill $pids; fi")
	time.Sleep(300 * time.Millisecond)
}

func captureBody(t *testing.T, path string) string {
	t.Helper()
	if !scratchExists(path) {
		return ""
	}
	out, _, err := dockerExec("cat", path)
	if err != nil {
		return ""
	}
	return out
}

func counterPackets(t *testing.T, m enforce.Model, name string) uint64 {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/at-counters.json", raw)
	out, errOut, err := dockerExec("fvpn-netlab", "counters", "-f", "/scratch/at-counters.json")
	if err != nil {
		t.Fatalf("counters: %v\n%s%s", err, out, errOut)
	}
	var doc struct {
		Counters map[string]struct {
			Packets uint64 `json:"packets"`
		} `json:"counters"`
	}
	if json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("counters json: %s", out)
	}
	if doc.Counters == nil {
		return 0
	}
	return doc.Counters[name].Packets
}

func ruleset(t *testing.T) string {
	t.Helper()
	return mustExec(t, "nft", "list", "table", "inet", "fvpn")
}

func labMeta(policy string) isolation.Record {
	id := imageID(labPrefix() + ":lab")
	return isolation.Record{
		Build: gitSHA(), ImageDigests: []string{labPrefix() + ":lab " + id},
		PolicyRevision: policy, NodeIncarnation: "mechanism-lab",
		Commands: []string{
			"FVPN_LAB_PREFIX=" + labPrefix() + " make -f mk/netlab.mk netlab-up",
			"FVPN_NETLAB=1 FVPN_LAB_PREFIX=" + labPrefix() + " FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at",
		},
	}
}

func gitSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func imageID(name string) string {
	out, err := exec.Command("docker", "image", "inspect", "-f", "{{.Id}}", name).Output()
	if err != nil {
		return "absent"
	}
	return strings.TrimSpace(string(out))
}

func finish(t *testing.T, rec isolation.Record) {
	t.Helper()
	if err := isolation.Write(evidenceDir(), rec); err != nil {
		t.Errorf("evidence %s: %v", rec.ID, err)
	}
}
