//go:build netlab

package netlab

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// TestReviewR19_Q4_EstablishedDownloadSurvivesDestinationDeny checks the
// fixed forward chain (the historical proof name is retained) against an established flow whose destination was
// added to denied_v4. Revoke stays empty, so Apply does not call conntrack -D.
func TestReviewR19_Q4_EstablishedDownloadSurvivesDestinationDeny(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/stream-r19.log /scratch/stream-r19.log.pid")
	t.Cleanup(func() {
		killPidFile("/scratch/stream-r19.log.pid")
		stopCaptures(t)
		applyBaseline(t)
	})

	now := time.Now().UTC()
	initial := functionalModel(now, 4200)
	applyAt(t, initial)
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/stream-r19.log"); err != nil {
		t.Fatal(err)
	}
	waitStreamReady(t, "/scratch/stream-r19.log", 8*time.Second)
	ct := mustExec(t, "conntrack", "-L", "-p", "udp", "-s", "100.64.1.1")
	sport, replyPort, ok := udpReplyPorts(ct)
	if !ok {
		t.Fatalf("no udp conntrack tuple:\n%s", ct)
	}

	restricted := initial
	restricted.Generation = 4201
	restricted.Now = time.Now().UTC()
	restricted.ExtraDeniedV4 = append(append([]netip.Prefix{}, initial.ExtraDeniedV4...), mustPrefix("203.0.113.10/32"))
	restricted.Revoke = nil
	applyAt(t, restricted)

	rules := mustExec(t, "nft", "list", "table", "inet", "fvpn")
	if strings.Contains(rules, "fvpn_baseline=1") || !strings.Contains(rules, "fvpn_generation=4201") {
		t.Fatalf("destination deny did not stay on the serving ruleset:\n%s", rules)
	}
	ctAfter := mustExec(t, "conntrack", "-L", "-p", "udp", "-s", "100.64.1.1")
	if !strings.Contains(ctAfter, "100.64.1.1") || !strings.Contains(ctAfter, "dport=9000") {
		t.Fatal("conntrack entry was removed")
	}

	before := readCounters(t, restricted)
	startCapture(t, "fv-inet", "fv-ii", "udp and src host 192.0.2.37 and dst host 203.0.113.10 and dst port 9000", "/scratch/cap-r19-uplink.txt")
	startCapture(t, "root", "fv-fh", fmt.Sprintf("udp and dst host 100.64.1.1 and dst port %d", sport), "/scratch/cap-r19-return.txt")
	time.Sleep(300 * time.Millisecond)
	time.Sleep(1200 * time.Millisecond)
	injected := injectReply(t, replyPort, 4)
	time.Sleep(400 * time.Millisecond)
	stopCaptures(t)
	upN := packetLines(captureBody(t, "/scratch/cap-r19-uplink.txt"))
	retN := packetLines(captureBody(t, "/scratch/cap-r19-return.txt"))
	after := readCounters(t, restricted)
	denied := counterPackets(after.Counters, "ctdeny") - counterPackets(before.Counters, "ctdeny")
	sibling := runProbe(t, []probeCase{
		{"a2-other-destination", "fv-a2", "tcp", "203.0.113.20", 80, "", "allow", 800, "", "", "", "", 0},
		{"resolver-udp", "fv-a1", "dns", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
		{"resolver-tcp", "fv-a1", "tcp", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
	})
	printProbe(t, "destination deny controls", sibling)
	requireProbe(t, sibling)
	if upN != 0 {
		t.Fatalf("upload to the newly denied destination was forwarded: %d", upN)
	}
	if retN != 0 {
		t.Fatalf("established return still forwarded (uplink=%d return=%d ctdeny=%d injected=%d). The new denial must stop it without conntrack deletion", upN, retN, denied, injected)
	}
	deniedDown := counterPackets(after.Counters, "ctdenyd") - counterPackets(before.Counters, "ctdenyd")
	if denied == 0 || deniedDown < uint64(injected) {
		t.Fatalf("drops not counted: upload=%d download=%d injected=%d", denied, deniedDown, injected)
	}
	line := fmt.Sprintf("DESTINATION-DENY serving=4201 conntrack_kept=1 uplink=%d return=%d ctdeny=%d ctdenyd=%d injected=%d sibling=allow dns_udp=allow dns_tcp=allow", upN, retN, denied, deniedDown, injected)
	save(t, "destination-deny.txt", line+"\n")
	t.Log(line)
}
