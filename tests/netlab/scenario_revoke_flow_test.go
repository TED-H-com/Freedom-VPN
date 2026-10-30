//go:build netlab

package netlab

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRevoke_RulesetStopsEstablished checks whether removing a binding stops
// an already-established flow when the conntrack entry is left in place.
// Revoke is empty, so Apply does not call conntrack -D.
func TestRevoke_RulesetStopsEstablished(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/stream-revoke.log /scratch/stream-revoke.log.pid")
	t.Cleanup(func() {
		killPidFile("/scratch/stream-revoke.log.pid")
		stopCaptures(t)
		applyBaseline(t)
	})

	now := time.Now().UTC()
	initial := functionalModel(now, 4100)
	applyAt(t, initial)
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/stream-revoke.log"); err != nil {
		t.Fatal(err)
	}
	waitStreamReady(t, "/scratch/stream-revoke.log", 8*time.Second)

	ct := mustExec(t, "conntrack", "-L", "-p", "udp", "-s", "100.64.1.1")
	save(t, "conntrack-before.txt", ct)
	sport, replyPort, ok := udpReplyPorts(ct)
	if !ok {
		t.Fatalf("no udp conntrack tuple for 100.64.1.1 -> 203.0.113.10:9000:\n%s", ct)
	}

	revoked := deviceModel(time.Now().UTC(), 4101, []idBind{
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, generousQoS(), 48*time.Hour)
	revoked.Revoke = nil
	applyAt(t, revoked)

	rules := mustExec(t, "nft", "list", "table", "inet", "fvpn")
	if strings.Contains(rules, "fvpn_baseline=1") {
		t.Fatal("apply replaced the serving ruleset with the baseline")
	}
	if !strings.Contains(rules, "fvpn_generation=4101") {
		t.Fatal("serving generation 4101 is not installed")
	}
	ctAfter := mustExec(t, "conntrack", "-L", "-p", "udp", "-s", "100.64.1.1")
	save(t, "conntrack-after.txt", ctAfter)
	if !strings.Contains(ctAfter, "100.64.1.1") || !strings.Contains(ctAfter, "dport=9000") {
		t.Fatal("conntrack entry was removed; this proof needs it to remain")
	}

	before := readCounters(t, revoked)
	startCapture(t, "fv-inet", "fv-ii", "udp and src host 192.0.2.37 and dst host 203.0.113.10 and dst port 9000", "/scratch/cap-revoke-uplink.txt")
	startCapture(t, "root", "fv-fh", fmt.Sprintf("udp and dst host 100.64.1.1 and dst port %d", sport), "/scratch/cap-revoke-return.txt")
	time.Sleep(300 * time.Millisecond)
	time.Sleep(1200 * time.Millisecond)
	injected := injectReply(t, replyPort, 4)
	time.Sleep(400 * time.Millisecond)
	stopCaptures(t)

	uplink := captureBody(t, "/scratch/cap-revoke-uplink.txt")
	ret := captureBody(t, "/scratch/cap-revoke-return.txt")
	save(t, "uplink.txt", uplink)
	save(t, "return.txt", ret)
	upN := packetLines(uplink)
	retN := packetLines(ret)
	after := readCounters(t, revoked)
	ctunku := counterPackets(after.Counters, "ctunku") - counterPackets(before.Counters, "ctunku")
	ctunkd := counterPackets(after.Counters, "ctunkd") - counterPackets(before.Counters, "ctunkd")
	still := runProbe(t, []probeCase{
		{"a2-still", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0},
		{"a1-denied", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 600, "", "", "", "", 0},
	})
	table := printProbe(t, "revoke flow neighbors", still)
	requireProbe(t, still)

	line := fmt.Sprintf("REVOKE-FLOW decision=ruleset-stops serving=4101 conntrack_kept=1 uplink=%d return=%d ctunku=%d ctunkd=%d injected=%d sport=%d reply=%d",
		upN, retN, ctunku, ctunkd, injected, sport, replyPort)
	save(t, "revoke-flow.txt", line+"\n"+table)
	noteSummary(line)
	if upN != 0 || retN != 0 {
		t.Fatalf("revoked flow still forwarded: uplink=%d return=%d\nuplink:\n%s\nreturn:\n%s", upN, retN, uplink, ret)
	}
	if ctunku == 0 || ctunkd == 0 {
		t.Fatalf("ruleset did not count the dropped directions: ctunku=%d ctunkd=%d", ctunku, ctunkd)
	}
}

var udpTuple = regexp.MustCompile(`src=100\.64\.1\.1 dst=203\.0\.113\.10 sport=(\d+) dport=9000\b.*src=203\.0\.113\.10 dst=192\.0\.2\.37 sport=9000 dport=(\d+)`)

func udpReplyPorts(listing string) (sport, replyPort int, ok bool) {
	m := udpTuple.FindStringSubmatch(listing)
	if m == nil {
		return 0, 0, false
	}
	sport, err1 := strconv.Atoi(m[1])
	replyPort, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || sport == 0 || replyPort == 0 {
		return 0, 0, false
	}
	return sport, replyPort, true
}

func injectReply(t *testing.T, dport, n int) int {
	t.Helper()
	script := fmt.Sprintf(`import socket, struct
def checksum(data):
    if len(data) %% 2:
        data += b"\x00"
    total = 0
    for i in range(0, len(data), 2):
        total += (data[i] << 8) + data[i + 1]
    while total >> 16:
        total = (total & 0xFFFF) + (total >> 16)
    return (~total) & 0xFFFF
src = socket.inet_aton("203.0.113.10")
dst = socket.inet_aton("192.0.2.37")
payload = b"r"
udp_len = 8 + len(payload)
pseudo = src + dst + struct.pack("!BBH", 0, 17, udp_len)
udp = struct.pack("!HHHH", 9000, %d, udp_len, 0) + payload
csum = checksum(pseudo + udp)
if csum == 0:
    csum = 0xFFFF
udp = struct.pack("!HHHH", 9000, %d, udp_len, csum) + payload
ip_len = 20 + len(udp)
ip = struct.pack("!BBHHHBBH4s4s", 0x45, 0, ip_len, 0, 0, 64, 17, 0, src, dst)
ip = struct.pack("!BBHHHBBH4s4s", 0x45, 0, ip_len, 0, 0, 64, 17, checksum(ip), src, dst)
s = socket.socket(socket.AF_INET, socket.SOCK_RAW, socket.IPPROTO_RAW)
for _ in range(%d):
    s.sendto(ip + udp, ("192.0.2.37", 0))
print(%d)
`, dport, dport, n, n)
	writeScratch(t, "/scratch/inject-reply.py", []byte(script))
	out := mustExec(t, "ip", "netns", "exec", "fv-inet", "python3", "/scratch/inject-reply.py")
	got, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || got != n {
		t.Fatalf("inject reply: %q", out)
	}
	return got
}
