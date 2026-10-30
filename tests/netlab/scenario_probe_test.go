//go:build netlab

package netlab

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TestProbe_BaselineException is the §13.9 proof. Member forwarding stays on
// the boot ruleset. Only the dedicated probe peer's /32 may pass, and only
// to the listed socket. The kernel timeout removes that exception.
func TestProbe_BaselineException(t *testing.T) {
	skipLab(t)
	startAdminListener(t)
	t.Cleanup(func() { applyBaseline(t) })

	now := time.Now().UTC()
	m := functionalModel(now, 1)
	m.Probe = &enforce.ProbeException{
		PeerID:   "peer-probe",
		SourceV4: mustPrefix("100.64.8.1/32"),
		Destinations: []enforce.ProbeDestination{{
			AddrV4: mustPrefix("203.0.113.10/32"),
			Proto:  "tcp",
			Port:   80,
		}},
		ExpiresAt: now.Add(8 * time.Second),
	}
	before := readCounters(t, m)
	start := time.Now()
	applyBaselineModel(t, m)
	listed := mustExec(t, "nft", "-j", "list", "set", "inet", "fvpn", "probe_v4")
	if !strings.Contains(listed, "100.64.8.1") || (!strings.Contains(listed, "timeout") && !strings.Contains(listed, "expires")) {
		t.Fatalf("probe source is not a timed element: %s", listed)
	}
	save(t, "probe-set.json", listed)

	flushConntrack(t)
	startCapture(t, "fv-inet", "fv-ii", "tcp and dst host 203.0.113.10 and dst port 80", "/scratch/cap-probe-snat.txt")
	time.Sleep(300 * time.Millisecond)
	positive := runProbe(t, []probeCase{
		{"probe-ok", "fv-probe", "tcp", "203.0.113.10", 80, "100.64.8.1", "allow", 1500, "", "", "", "", 0},
	})
	stopCaptures(t)
	printProbe(t, "probe positive", positive)
	requireProbe(t, positive)
	snat := captureBody(t, "/scratch/cap-probe-snat.txt")
	save(t, "snat.txt", snat)
	if !strings.Contains(snat, "192.0.2.37") || strings.Contains(snat, "100.64.8.1") {
		t.Fatalf("probe was not SNATed to 192.0.2.37:\n%s", snat)
	}
	mid := readCounters(t, m)
	if counterPackets(mid.Counters, "probe_up") <= counterPackets(before.Counters, "probe_up") ||
		counterPackets(mid.Counters, "probe_down") <= counterPackets(before.Counters, "probe_down") {
		t.Fatalf("probe counters did not advance: before %+v after %+v", before.Counters, mid.Counters)
	}
	if err := sameUsage(before.Counters, mid.Counters); err != nil {
		t.Fatal(err)
	}

	neg := runProbe(t, []probeCase{
		{"neg-a-sibling", "fv-probe", "tcp", "203.0.113.10", 80, "100.64.8.2", "deny", 600, "", "", "", "", 0},
		{"neg-b-dest", "fv-probe", "tcp", "203.0.113.20", 80, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"neg-b-port", "fv-probe", "tcp", "203.0.113.10", 9001, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"neg-c-admin", "fv-probe", "tcp", "192.0.2.37", 22, "100.64.8.1", "deny", 600, "", "", "", "", 0},
	})
	printProbe(t, "probe negative", neg)
	requireProbe(t, neg)
	afterNeg := readCounters(t, m)
	if err := sameUsage(before.Counters, afterNeg.Counters); err != nil {
		t.Fatal(err)
	}

	wait := 12*time.Second - time.Since(start)
	if wait > 0 {
		time.Sleep(wait)
	}
	flushConntrack(t)
	expired := runProbe(t, []probeCase{
		{"neg-d-expired", "fv-probe", "tcp", "203.0.113.10", 80, "100.64.8.1", "deny", 800, "", "", "", "", 0},
	})
	printProbe(t, "probe expired", expired)
	requireProbe(t, expired)
	for _, set := range []struct{ name, addr string }{{"probe_v4", "100.64.8.1"}, {"probe_dst_v4", "203.0.113.10"}} {
		out, errOut, err := dockerExec("nft", "-j", "list", "set", "inet", "fvpn", set.name)
		if err == nil && strings.Contains(out, set.addr) {
			t.Fatalf("%s still contains %s after expiry: %s", set.name, set.addr, out)
		}
		if err != nil && !strings.Contains(errOut, "No such file") && !strings.Contains(err.Error(), "No such file") {
			t.Fatalf("list %s: %v %s", set.name, err, errOut)
		}
	}

	dnsNow := time.Now().UTC()
	dnsModel := functionalModel(dnsNow, 1)
	dnsModel.Probe = &enforce.ProbeException{
		PeerID:      "peer-probe",
		SourceV4:    mustPrefix("100.64.8.1/32"),
		ResolverDNS: true,
		ExpiresAt:   dnsNow.Add(60 * time.Second),
	}
	applyBaselineModel(t, dnsModel)
	dnsBefore := readCounters(t, dnsModel)
	dns := runProbe(t, []probeCase{
		{"dns-udp", "fv-probe", "dns", "198.51.100.53", 53, "100.64.8.1", "allow", 1000, "", "", "", "", 0},
		{"dns-tcp", "fv-probe", "tcp", "198.51.100.53", 53, "100.64.8.1", "allow", 1000, "", "", "", "", 0},
		{"dns-other-port", "fv-probe", "tcp", "198.51.100.53", 80, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"dns-other-dest", "fv-probe", "tcp", "203.0.113.10", 53, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"dns-other-udp", "fv-probe", "dns", "203.0.113.10", 53, "100.64.8.1", "deny", 600, "", "", "", "", 0},
	})
	printProbe(t, "probe dns", dns)
	requireProbe(t, dns)
	dnsAfter := readCounters(t, dnsModel)
	if counterPackets(dnsAfter.Counters, "probe_up") <= counterPackets(dnsBefore.Counters, "probe_up") {
		t.Fatal("resolver probe did not count probe_up")
	}
	if err := sameUsage(dnsBefore.Counters, dnsAfter.Counters); err != nil {
		t.Fatal(err)
	}
	noteSummary(fmt.Sprintf("PROBE snat=1 up=%d down=%d neg=4 expired=1 dns_only_53=1",
		counterPackets(mid.Counters, "probe_up")-counterPackets(before.Counters, "probe_up"),
		counterPackets(mid.Counters, "probe_down")-counterPackets(before.Counters, "probe_down")))
}

func applyBaselineModel(t *testing.T, m enforce.Model) {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/probe-baseline.json", raw)
	out, errOut, err := dockerExec("fvpn-netlab", "baseline", "-f", "/scratch/probe-baseline.json")
	if err != nil {
		save(t, "probe-baseline-failure.txt", out+"\n"+errOut+"\n"+debugPlane())
		t.Fatalf("baseline probe: %v\n%s%s", err, out, errOut)
	}
}

func startAdminListener(t *testing.T) {
	t.Helper()
	body := []byte("import socket\n" +
		"s=socket.socket()\n" +
		"s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)\n" +
		"s.bind(('192.0.2.37', 22))\n" +
		"s.listen(8)\n" +
		"open('/scratch/svc-admin22.ready','w').write('ok\\n')\n" +
		"while True:\n" +
		"    c,_=s.accept()\n" +
		"    c.close()\n")
	writeScratch(t, "/scratch/admin22.py", body)
	if err := dockerDetach("sh", "-c", "echo $$ > /scratch/admin22.pid; exec python3 /scratch/admin22.py"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { killPidFile("/scratch/admin22.pid") })
	for i := 0; i < 20; i++ {
		if scratchExists("/scratch/svc-admin22.ready") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("admin listener on 192.0.2.37:22 did not start")
}

func sameUsage(before, after map[string]enforce.CounterValue) error {
	for name, prev := range before {
		if !usageCounterName(name) {
			continue
		}
		cur := after[name]
		if cur.Packets != prev.Packets || counterBytes(after, name) != counterBytes(before, name) {
			return fmt.Errorf("member counter %s changed %d/%d -> %d/%d", name, prev.Packets, prev.Bytes, cur.Packets, cur.Bytes)
		}
	}
	for name := range after {
		if usageCounterName(name) {
			if _, ok := before[name]; !ok && (after[name].Packets != 0 || after[name].Bytes != 0) {
				return fmt.Errorf("member counter %s appeared", name)
			}
		}
	}
	return nil
}

func usageCounterName(name string) bool {
	return strings.HasPrefix(name, "nu_") || strings.HasPrefix(name, "nd_") || strings.HasPrefix(name, "u_") || strings.HasPrefix(name, "d_")
}
