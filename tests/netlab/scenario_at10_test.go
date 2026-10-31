//go:build netlab

package netlab

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAT10_AT13_ReloadRevokeBaseline(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/denyloop.stop /scratch/deny-hits.log /scratch/stream-a1.log /scratch/hold.ready /scratch/at10-apply.jsonl /scratch/stream-a1.log.jsonl")
	_, _, _ = dockerExec("ip", "netns", "exec", "fv-fwd", "ip", "addr", "del", "100.64.7.7/32", "dev", "fv-ff")
	mustExec(t, "ip", "netns", "exec", "fv-fwd", "ip", "addr", "add", "100.64.7.7/32", "dev", "fv-ff")
	t.Cleanup(func() {
		_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop")
		killPidFile("/scratch/denyloop.pid")
		killPidFile("/scratch/stream-a1.log.pid")
		stopCaptures(t)
		_, _, _ = dockerExec("ip", "netns", "exec", "fv-fwd", "ip", "addr", "del", "100.64.7.7/32", "dev", "fv-ff")
		_, _, _ = dockerExec("ip", "link", "del", "fv-px")
	})

	now := time.Now().UTC()
	initial := functionalModel(now, 1000)
	applyAt(t, initial)

	denyCases := []probeCase{
		{"lan10", "fv-a1", "tcp", "10.1.2.10", 80, "", "deny", 400, "", "", "", "", 0},
		{"lan192", "fv-a1", "tcp", "192.168.10.10", 80, "", "deny", 400, "", "", "", "", 0},
		{"lan172", "fv-a1", "tcp", "172.16.5.10", 80, "", "deny", 400, "", "", "", "", 0},
		{"meta", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 400, "", "", "", "", 0},
		{"pubpriv", "fv-a1", "tcp", "192.0.2.200", 80, "", "deny", 400, "", "", "", "", 0},
		{"a2", "fv-a1", "tcp", "100.64.1.2", 8080, "", "deny", 400, "", "", "", "", 0},
		{"b1", "fv-a1", "tcp", "100.64.2.1", 8080, "", "deny", 400, "", "", "", "", 0},
		{"ssh", "fv-a1", "tcp", "192.0.2.33", 2222, "", "deny", 400, "", "", "", "", 0},
		{"resolver8080", "fv-a1", "tcp", "198.51.100.53", 8080, "", "deny", 400, "", "", "", "", 0},
		{"rogue", "fv-rogue", "tcp", "203.0.113.10", 80, "", "deny", 400, "", "", "", "", 0},
		{"spoof", "fv-fwd", "icmp", "203.0.113.10", 0, "100.64.7.7", "deny", 800, "", "", "", "", 0},
	}
	body := mustJSON(map[string]any{"cases": denyCases})
	writeScratch(t, "/scratch/deny-cases.json", body)
	if err := dockerDetach("python3", "/opt/netlab/scripts/denyloop.py", "/scratch/deny-cases.json"); err != nil {
		t.Fatal(err)
	}
	startReloadTrace(t)
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/stream-a1.log"); err != nil {
		t.Fatal(err)
	}
	startCapture(t, "fv-lan", "fv-ll", "ip and (dst host 10.1.2.10 or dst host 192.168.10.10 or dst host 172.16.5.10 or dst host 169.254.169.254 or dst host 192.0.2.200)", "/scratch/cap-at10-lan.txt")
	startCapture(t, "fv-a2", "fv-wg0", "dst port 8080", "/scratch/cap-at10-a2.txt")
	startCapture(t, "fv-inet", "fv-ii", "dst port 8080", "/scratch/cap-at10-dns.txt")
	waitStreamReady(t, "/scratch/stream-a1.log", 8*time.Second)

	prevA1 := sampleBytes(readCounters(t, initial).Samples, devA1, "upload")
	prevA2 := sampleBytes(readCounters(t, initial).Samples, devA2, "upload")
	var lowA1, lowA2 uint64 = prevA1, prevA2
	for i := 1; i <= 100; i++ {
		gen := uint64(1000 + i)
		binds := allBinds()
		if i%10 == 0 {
			binds = []idBind{{memberA, devA1, "peer-a1", "100.64.1.1"}, {memberB, devB1, "peer-b1", "100.64.2.1"}}
		}
		m := deviceModel(time.Now().UTC(), gen, binds, generousQoS(), 48*time.Hour)
		doc := applyAt(t, m)
		a1 := sampleBytes(doc.Samples, devA1, "upload")
		a2 := sampleBytes(doc.Samples, devA2, "upload")
		if a1 < lowA1 {
			t.Fatalf("generation %d a1 upload went backwards %d -> %d", gen, lowA1, a1)
		}
		if a2 < lowA2 {
			t.Fatalf("generation %d a2 upload went backwards %d -> %d", gen, lowA2, a2)
		}
		lowA1, lowA2 = a1, a2
		if i%10 == 0 {
			if !finalHas(doc, devA2) {
				t.Fatalf("generation %d omitted a2 without a final sample", gen)
			}
		}
		if i == 50 {
			flapLAN(t)
		}
		if i%25 == 0 {
			fmt.Printf("AT10 regen %d a1=%d a2=%d\n", gen, a1, a2)
		}
	}
	// Let an in-flight 1.5 s read finish so the last reload cannot hide a timeout.
	time.Sleep(1600 * time.Millisecond)
	finishReloadTrace(t)
	if !streamHealthy(t, "/scratch/stream-a1.log") {
		save(t, "stream-during-regen.log", readScratch(t, "/scratch/stream-a1.log"))
		t.Fatal("allowed flow failed during ruleset regeneration")
	}

	ctBefore := execText("conntrack", "-L")
	save(t, "conntrack-before-revoke.txt", sanitize(ctBefore))
	if !strings.Contains(ctBefore, "100.64.1.1") {
		t.Fatal("expected an established conntrack entry for 100.64.1.1 before revoke")
	}
	revokeAt := time.Now()
	revoked := deviceModel(revokeAt.UTC(), 2000, []idBind{
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, generousQoS(), 48*time.Hour)
	revoked.Revoke = []netip.Addr{mustIP("100.64.1.1")}
	applyAt(t, revoked)
	stopped := waitStreamFail(t, "/scratch/stream-a1.log", revokeAt, 5*time.Second)
	ctAfter := execText("conntrack", "-L", "-s", "100.64.1.1")
	save(t, "conntrack-after-revoke.txt", sanitize(ctAfter))
	if strings.Contains(ctAfter, "100.64.1.1") {
		t.Fatal("conntrack entry for the revoked device survived")
	}
	a2ok := runProbe(t, []probeCase{{"a2-still", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}})
	requireProbe(t, a2ok)

	hold := deviceModel(time.Now().UTC(), 2100, []idBind{
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, generousQoS(), 48*time.Hour)
	raw := mustJSON(hold)
	writeScratch(t, "/scratch/hold.json", raw)
	_, _, _ = dockerExec("rm", "-f", "/scratch/hold.ready")
	if err := dockerDetach("fvpn-netlab", "hold", "-f", "/scratch/hold.json"); err != nil {
		t.Fatal(err)
	}
	waitScratch(t, "/scratch/hold.ready", 10*time.Second)
	mustExec(t, "sh", "-c", "kill -9 $(pidof fvpn-netlab)")
	time.Sleep(200 * time.Millisecond)
	if strings.TrimSpace(execText("pidof", "fvpn-netlab")) != "" && !strings.Contains(execText("pidof", "fvpn-netlab"), "exit status") {
		// pidof returns non-zero when empty; execText then contains the error text.
	}
	alive := execText("sh", "-c", "pidof fvpn-netlab || true")
	if strings.TrimSpace(alive) != "" {
		t.Fatalf("agent still running after kill: %s", alive)
	}
	rules := mustExec(t, "nft", "list", "table", "inet", "fvpn")
	if !strings.Contains(rules, "fvpn_generation=2100") {
		t.Fatal("killing the agent removed or replaced the ruleset")
	}
	still := runProbe(t, []probeCase{
		{"a1-stays-revoked", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"a2-stays-up", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0},
		{"lan-stays-denied", "fv-a2", "tcp", "10.1.2.10", 80, "", "deny", 800, "", "", "", "", 0},
	})
	table := printProbe(t, "AT10 after kill", still)
	requireProbe(t, still)

	_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop")
	killPidFile("/scratch/denyloop.pid")
	applyBaseline(t)
	baseProbe := runProbe(t, []probeCase{
		{"baseline-a2", "fv-a2", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"baseline-a1", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
	})
	table += printProbe(t, "AT10 baseline", baseProbe)
	requireProbe(t, baseProbe)
	local := mustExec(t, "python3", "-c", "import socket; s=socket.create_connection(('127.0.0.1',2222),2); s.close(); print('local-ok')")
	if !strings.Contains(local, "local-ok") {
		t.Fatalf("host control listener failed under baseline: %s", local)
	}
	stopCaptures(t)
	for _, path := range []string{"/scratch/cap-at10-lan.txt", "/scratch/cap-at10-a2.txt", "/scratch/cap-at10-dns.txt"} {
		body := captureBody(t, path)
		if strings.TrimSpace(body) != "" {
			save(t, "capture-leak.txt", body)
			t.Fatalf("denied packet reached %s", path)
		}
	}
	hits := ""
	if scratchExists("/scratch/deny-hits.log") {
		hits = readScratch(t, "/scratch/deny-hits.log")
	}
	if strings.TrimSpace(hits) != "" {
		save(t, "deny-hits.log", hits)
		t.Fatalf("deny loop saw allowed probes:\n%s", hits)
	}
	loops := "0"
	if scratchExists("/scratch/denyloop.count") {
		loops = strings.TrimSpace(readScratch(t, "/scratch/denyloop.count"))
	}
	save(t, "reload.txt", table+fmt.Sprintf("regens=100 revoke_stop=%s deny_loops=%s a1_upload=%d\n", stopped, loops, lowA1))
	noteSummary(fmt.Sprintf("AT10 regens=100 revoke_stop=%s deny_loops=%s hits=0 baseline_forwarding=off", stopped, loops))
}

func flapLAN(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("ip", "link", "del", "fv-px")
	mustExec(t, "ip", "link", "add", "fv-px", "type", "veth", "peer", "name", "fv-py")
	mustExec(t, "ip", "addr", "add", "198.18.0.1/30", "dev", "fv-px")
	mustExec(t, "ip", "link", "set", "fv-px", "up")
	mustExec(t, "ip", "link", "set", "fv-py", "up")
	mustExec(t, "ip", "addr", "add", "198.18.0.2/30", "dev", "fv-py")
	mustExec(t, "ip", "addr", "add", "198.18.0.10/32", "dev", "fv-py")
	mustExec(t, "ip", "route", "add", "198.18.0.10/32", "via", "198.18.0.2", "dev", "fv-px")
	// Bringing the veth up emits IPv6 neighbor discovery from the host stack.
	// That is not a forwarded member packet. Capture only IPv4 to the new target.
	startCapture(t, "root", "fv-py", "ip and host 198.18.0.10", "/scratch/cap-at10-flap.txt")
	time.Sleep(200 * time.Millisecond)
	res := runProbe(t, []probeCase{{"flap", "fv-a1", "tcp", "198.18.0.10", 80, "", "deny", 800, "", "", "", "", 0}})
	requireProbe(t, res)
	stopOneCapture("/scratch/cap-at10-flap.txt")
	body := captureBody(t, "/scratch/cap-at10-flap.txt")
	if strings.TrimSpace(body) != "" {
		t.Fatalf("flapped LAN received a packet:\n%s", body)
	}
	mustExec(t, "ip", "link", "del", "fv-px")
}

func streamHealthy(t *testing.T, path string) bool {
	t.Helper()
	body := readScratch(t, path)
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) == 0 {
		return false
	}
	var lastOK time.Time
	var fails int
	for _, line := range lines {
		ts, kind, ok := parseStream(line)
		if !ok {
			continue
		}
		if kind == "ok" {
			lastOK = ts
		}
		if strings.HasPrefix(kind, "fail") {
			fails++
		}
	}
	if fails > 0 {
		return false
	}
	return time.Since(lastOK) < 3*time.Second
}

func waitStreamFail(t *testing.T, path string, after time.Time, limit time.Duration) time.Duration {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		body := readScratch(t, path)
		for _, line := range strings.Split(body, "\n") {
			ts, kind, ok := parseStream(line)
			if ok && strings.HasPrefix(kind, "fail") && !ts.Before(after.Add(-500*time.Millisecond)) {
				d := ts.Sub(after)
				if d < 0 {
					d = 0
				}
				if d > limit {
					t.Fatalf("revoked flow stopped after %s", d)
				}
				return d.Round(time.Millisecond)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	save(t, "stream-revoke.log", readScratch(t, path))
	t.Fatal("revoked flow did not stop")
	return 0
}

func parseStream(line string) (time.Time, string, bool) {
	fields := strings.SplitN(strings.TrimSpace(line), " ", 2)
	if len(fields) != 2 {
		return time.Time{}, "", false
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return time.Time{}, "", false
	}
	return time.Unix(0, int64(sec*1e9)), fields[1], true
}

func finalHas(doc applyDoc, id uuid.UUID) bool {
	for _, s := range doc.Finals {
		if s.DeviceID == id && s.Direction == "upload" {
			return true
		}
	}
	return false
}

func waitScratch(t *testing.T, path string, limit time.Duration) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if scratchExists(path) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func waitStreamReady(t *testing.T, path string, limit time.Duration) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if scratchExists(path) && strings.Contains(readScratch(t, path), " ok") {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	body := ""
	if scratchExists(path) {
		body = readScratch(t, path)
	}
	t.Fatalf("allowed flow did not become ready: %s", body)
}

func sanitize(s string) string {
	if publicLeak(s) {
		return "redacted\n"
	}
	return s
}
