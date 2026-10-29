//go:build netlab

package netlab

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAT12_ISO01_ISO04_AllowDeny(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	_, _, _ = dockerExec("ip", "netns", "exec", "fv-fwd", "ip", "addr", "del", "100.64.7.7/32", "dev", "fv-ff")
	mustExec(t, "ip", "netns", "exec", "fv-fwd", "ip", "addr", "add", "100.64.7.7/32", "dev", "fv-ff")
	t.Cleanup(func() {
		_, _, _ = dockerExec("ip", "netns", "exec", "fv-fwd", "ip", "addr", "del", "100.64.7.7/32", "dev", "fv-ff")
	})

	now := time.Now().UTC()
	m := functionalModel(now, 12)
	doc := applyAt(t, m)
	if doc.Admitted != 3 {
		t.Fatalf("admitted %d, want 3", doc.Admitted)
	}
	save(t, "ruleset.nft", mustExec(t, "nft", "list", "table", "inet", "fvpn"))
	save(t, "tc-ih.txt", mustExec(t, "tc", "class", "show", "dev", "fv-ih"))

	dstMAC := ifaceMAC(t, "root", "fv-fh")
	srcMAC := ifaceMAC(t, "fv-fwd", "fv-ff")

	deny := []probeCase{
		{"a1-to-a2", "fv-a1", "tcp", "100.64.1.2", 8080, "", "deny", 800, "", "", "", "", 0},
		{"a1-to-b1", "fv-a1", "tcp", "100.64.2.1", 8080, "", "deny", 800, "", "", "", "", 0},
		{"a1-to-overlay", "fv-a1", "tcp", "100.64.5.5", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-ssh", "fv-a1", "tcp", "192.0.2.33", 2222, "", "deny", 800, "", "", "", "", 0},
		{"a1-api", "fv-a1", "tcp", "192.0.2.37", 8443, "", "deny", 800, "", "", "", "", 0},
		{"a1-lan-gw", "fv-a1", "tcp", "192.168.0.1", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-rfc1918-192", "fv-a1", "tcp", "192.168.10.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-rfc1918-10", "fv-a1", "tcp", "10.1.2.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-rfc1918-172", "fv-a1", "tcp", "172.16.5.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-metadata", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-linklocal", "fv-a1", "tcp", "169.254.1.50", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-public-prefix-private", "fv-a1", "tcp", "192.0.2.200", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-resolver-http", "fv-a1", "tcp", "198.51.100.53", 8080, "", "deny", 800, "", "", "", "", 0},
		{"a1-resolver-icmp", "fv-a1", "icmp", "198.51.100.53", 0, "", "deny", 800, "", "", "", "", 0},
		{"rogue-internet", "fv-rogue", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"spoof-internet", "fv-fwd", "icmp", "203.0.113.10", 0, "100.64.7.7", "deny", 800, "", "", "", "", 0},
		{"loopback-inject", "fv-fwd", "raw4", "127.0.0.1", 9, "", "sent", 800, "100.64.1.1", srcMAC, dstMAC, "fv-ff", 0},
	}

	before := readCounters(t, m)
	lanLog := logLines(t, "/scratch/svc-lan.log")
	hostLog := logLines(t, "/scratch/svc-host.log")
	memberLog := logLines(t, "/scratch/svc-member.log")
	startCapture(t, "fv-lan", "fv-ll", "", "/scratch/cap-at12-lan.txt")
	startCapture(t, "fv-a2", "fv-wg0", "dst port 8080", "/scratch/cap-at12-a2.txt")
	startCapture(t, "root", "lo", "host 127.0.0.1", "/scratch/cap-at12-lo.txt")
	time.Sleep(300 * time.Millisecond)
	denied := runProbe(t, deny)
	stopCaptures(t)
	table := printProbe(t, "AT12 deny", denied)
	requireProbe(t, denied)
	for _, path := range []string{"/scratch/cap-at12-lan.txt", "/scratch/cap-at12-a2.txt", "/scratch/cap-at12-lo.txt"} {
		body := captureBody(t, path)
		if strings.TrimSpace(body) != "" {
			t.Fatalf("denied target capture %s was not empty:\n%s", path, body)
		}
		save(t, strings.TrimPrefix(path, "/scratch/"), "# empty target capture\n")
	}
	if logLines(t, "/scratch/svc-lan.log") != lanLog || logLines(t, "/scratch/svc-host.log") != hostLog || logLines(t, "/scratch/svc-member.log") != memberLog {
		t.Fatal("a denied target accepted a connection")
	}
	mid := readCounters(t, m)
	if sampleBytes(mid.Samples, devA1, "upload") != sampleBytes(before.Samples, devA1, "upload") {
		t.Fatalf("denied traffic counted as usage: before %d after %d", sampleBytes(before.Samples, devA1, "upload"), sampleBytes(mid.Samples, devA1, "upload"))
	}
	diag := counterPackets(mid.Counters, "ctdeny") + counterPackets(mid.Counters, "ctunku") + counterPackets(mid.Counters, "cthost") + counterPackets(mid.Counters, "ctin") + counterPackets(mid.Counters, "ctbadns")
	diagBefore := counterPackets(before.Counters, "ctdeny") + counterPackets(before.Counters, "ctunku") + counterPackets(before.Counters, "cthost") + counterPackets(before.Counters, "ctin") + counterPackets(before.Counters, "ctbadns")
	if diag <= diagBefore {
		t.Fatalf("diagnostic counters did not move (%d -> %d)", diagBefore, diag)
	}

	dns := runProbe(t, []probeCase{
		{"a1-dns-udp", "fv-a1", "dns", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
		{"a1-dns-tcp", "fv-a1", "tcp", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
	})
	table += printProbe(t, "AT12 dns", dns)
	requireProbe(t, dns)
	afterDNS := readCounters(t, m)
	if sampleBytes(afterDNS.Samples, devA1, "upload") != sampleBytes(mid.Samples, devA1, "upload") {
		t.Fatal("dns was counted in forward_admission_v1")
	}
	if sampleBytes(afterDNS.Samples, devA1, "dns_upload") <= sampleBytes(mid.Samples, devA1, "dns_upload") {
		t.Fatal("dns_service_v1 counter did not increase")
	}

	allow := runProbe(t, []probeCase{
		{"a1-tcp", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0},
		{"a1-udp", "fv-a1", "udp", "203.0.113.10", 9000, "", "allow", 800, "", "", "", "", 0},
		{"a1-icmp", "fv-a1", "icmp", "203.0.113.10", 0, "", "allow", 800, "", "", "", "", 0},
		{"a2-tcp", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0},
		{"b1-tcp", "fv-b1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0},
	})
	table += printProbe(t, "AT12 allow", allow)
	requireProbe(t, allow)
	final := readCounters(t, m)
	if sampleBytes(final.Samples, devA1, "upload") <= sampleBytes(afterDNS.Samples, devA1, "upload") {
		t.Fatal("allowed upload did not increase the device counter")
	}
	save(t, "matrix.txt", table)
	save(t, "counters.json", pretty(final))
	noteSummary(fmt.Sprintf("AT12 deny_diag=%d dns_upload=%d a1_upload=%d admitted=%d", diag-diagBefore, sampleBytes(final.Samples, devA1, "dns_upload"), sampleBytes(final.Samples, devA1, "upload"), doc.Admitted))
}

func pretty(v any) string {
	return fmt.Sprintf("%+v\n", v)
}
