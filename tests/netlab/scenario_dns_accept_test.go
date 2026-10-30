//go:build netlab

package netlab

import (
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TestProbe_DNSAcceptBound is the live proof that resolver DNS for the probe
// is only the probe source /32 to the configured resolver, port 53, and only
// while that element is alive. Every other denied destination stays dropped.
func TestProbe_DNSAcceptBound(t *testing.T) {
	skipLab(t)
	t.Cleanup(func() {
		_, _, _ = dockerExec("ip", "route", "del", "192.88.99.1/32")
		stopCaptures(t)
		applyBaseline(t)
	})
	// 192.88.99.0/24 is denied and has no lab route of its own. A route makes
	// a forwarded packet the only way it could reach the simulated uplink.
	mustExec(t, "ip", "route", "replace", "192.88.99.1/32", "via", "192.0.2.38", "dev", "fv-ih")

	now := time.Now().UTC()
	m := functionalModel(now, 1)
	m.Probe = &enforce.ProbeException{
		PeerID:      "peer-probe",
		SourceV4:    mustPrefix("100.64.8.1/32"),
		ResolverDNS: true,
		ExpiresAt:   now.Add(45 * time.Second),
	}
	applyBaselineModel(t, m)
	rules := mustExec(t, "nft", "list", "table", "inet", "fvpn")
	save(t, "ruleset.nft", rules)
	flat := strings.Join(strings.Fields(rules), " ")
	if strings.Count(flat, "ip saddr @probe_v4 ip daddr @resolver_v4 tcp dport 53") != 1 ||
		strings.Count(flat, "ip saddr @probe_v4 ip daddr @resolver_v4 udp dport 53") != 1 {
		t.Fatalf("probe DNS accept is not one tcp and one udp port 53:\n%s", rules)
	}
	denied := mustExec(t, "nft", "list", "set", "inet", "fvpn", "denied_v4")
	save(t, "denied_v4.nft", denied)
	if !strings.Contains(denied, "192.88.99.0/24") {
		t.Fatal("live @denied_v4 missing 192.88.99.0/24")
	}
	for _, prefix := range []string{"192.31.196.0/24", "192.52.193.0/24", "192.175.48.0/24"} {
		if strings.Contains(denied, prefix) {
			t.Fatalf("live @denied_v4 contains globally reachable %s", prefix)
		}
	}
	resolvers := mustExec(t, "nft", "list", "set", "inet", "fvpn", "resolver_v4")
	save(t, "resolver_v4.nft", resolvers)
	if strings.Count(resolvers, "198.51.100.53") != 1 || strings.Contains(resolvers, "198.51.100.54") {
		t.Fatalf("resolver_v4 is not exactly 198.51.100.53:\n%s", resolvers)
	}
	probeSet := mustExec(t, "nft", "list", "set", "inet", "fvpn", "probe_v4")
	if !strings.Contains(probeSet, "100.64.8.1") || (!strings.Contains(probeSet, "timeout") && !strings.Contains(probeSet, "expires")) {
		t.Fatalf("probe source is not a timed /32: %s", probeSet)
	}

	startCapture(t, "fv-inet", "fv-ii", "host 192.88.99.1", "/scratch/cap-dns-6to4.txt")
	startCapture(t, "fv-lan", "fv-ll", "host 10.1.2.10", "/scratch/cap-dns-lan.txt")
	time.Sleep(300 * time.Millisecond)
	matrix := runProbe(t, []probeCase{
		{"dns-udp", "fv-probe", "dns", "198.51.100.53", 53, "100.64.8.1", "allow", 1000, "", "", "", "", 0},
		{"dns-tcp", "fv-probe", "tcp", "198.51.100.53", 53, "100.64.8.1", "allow", 1000, "", "", "", "", 0},
		{"resolver-tcp-80", "fv-probe", "tcp", "198.51.100.53", 80, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"resolver-udp-853", "fv-probe", "udp", "198.51.100.53", 853, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"lan-tcp-53", "fv-probe", "tcp", "10.1.2.10", 53, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"lan-tcp-9", "fv-probe", "tcp", "10.1.2.10", 9, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"lan-udp-53", "fv-probe", "dns", "10.1.2.10", 53, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"sixto4-tcp-53", "fv-probe", "tcp", "192.88.99.1", 53, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"sixto4-udp-80", "fv-probe", "udp", "192.88.99.1", 80, "100.64.8.1", "deny", 600, "", "", "", "", 0},
		{"sibling-dns", "fv-probe", "dns", "198.51.100.53", 53, "100.64.8.2", "deny", 600, "", "", "", "", 0},
	})
	stopCaptures(t)
	table := printProbe(t, "probe dns accept", matrix)
	requireProbe(t, matrix)
	for _, path := range []string{"/scratch/cap-dns-6to4.txt", "/scratch/cap-dns-lan.txt"} {
		body := captureBody(t, path)
		save(t, strings.TrimPrefix(path, "/scratch/"), body)
		if n := packetLines(body); n != 0 {
			t.Fatalf("%s saw %d packets from a denied destination", path, n)
		}
	}

	expAt := time.Now().UTC()
	short := functionalModel(expAt, 1)
	short.Probe = &enforce.ProbeException{
		PeerID:      "peer-probe",
		SourceV4:    mustPrefix("100.64.8.1/32"),
		ResolverDNS: true,
		ExpiresAt:   expAt.Add(8 * time.Second),
	}
	start := time.Now()
	applyBaselineModel(t, short)
	live := runProbe(t, []probeCase{
		{"dns-while-live", "fv-probe", "dns", "198.51.100.53", 53, "100.64.8.1", "allow", 1000, "", "", "", "", 0},
	})
	table += printProbe(t, "probe dns while live", live)
	requireProbe(t, live)
	wait := 12*time.Second - time.Since(start)
	if wait > 0 {
		time.Sleep(wait)
	}
	flushConntrack(t)
	expired := runProbe(t, []probeCase{
		{"dns-after-expiry", "fv-probe", "dns", "198.51.100.53", 53, "100.64.8.1", "deny", 800, "", "", "", "", 0},
		{"dns-tcp-after-expiry", "fv-probe", "tcp", "198.51.100.53", 53, "100.64.8.1", "deny", 800, "", "", "", "", 0},
	})
	table += printProbe(t, "probe dns after expiry", expired)
	requireProbe(t, expired)
	left, errOut, err := dockerExec("nft", "list", "set", "inet", "fvpn", "probe_v4")
	if err == nil && strings.Contains(left, "100.64.8.1") {
		t.Fatalf("probe_v4 still contains the source after expiry: %s", left)
	}
	if err != nil && !strings.Contains(errOut, "No such file") && !strings.Contains(err.Error(), "No such file") {
		t.Fatalf("list probe_v4: %v %s", err, errOut)
	}
	save(t, "dns-accept.txt", table)
	noteSummary("DNS-ACCEPT tcp53=1 udp53=1 other_port=2 other_denied=5 sibling=1 expired=1 ruleset_narrow=1 capture=0")
}

func packetLines(body string) int {
	n := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, " IP ") || strings.HasPrefix(strings.TrimSpace(line), "IP ") {
			n++
		}
	}
	return n
}
