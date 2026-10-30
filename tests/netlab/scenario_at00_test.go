//go:build netlab

package netlab

import (
	"strings"
	"testing"
)

func TestAT00_Topology(t *testing.T) {
	skipLab(t)
	if !scratchExists("/scratch/topology.ok") {
		t.Fatal("topology did not finish; /scratch/topology.ok is missing")
	}
	out := mustExec(t, "ip", "netns", "exec", "fv-fwd", "/usr/local/bin/wg", "show", "fv-wg0", "latest-handshakes")
	n := 0
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] != "0" {
			n++
		}
	}
	if n != 5 {
		t.Fatalf("want 5 wireguard handshakes, got %d", n)
	}
	rules := mustExec(t, "nft", "list", "table", "inet", "fvpn")
	if !strings.Contains(rules, "fvpn_generation=0") || !strings.Contains(rules, "fvpn_baseline=1") {
		t.Fatalf("boot baseline marker missing")
	}
	if strings.Contains(rules, "policy accept") && strings.Contains(rules, "hook forward") && strings.Contains(rules, "policy accept") {
		// forward policy is drop; output policy is accept. Check the forward chain specifically.
	}
	if !strings.Contains(rules, "hook forward priority -150; policy drop") && !strings.Contains(rules, "hook forward priority -150; policy drop;") {
		// nft list prints "priority filter - 150" or "priority -150" depending on version.
		if !strings.Contains(rules, "type filter hook forward") || !strings.Contains(rules, "policy drop") {
			t.Fatalf("forward baseline is not drop priority -150")
		}
	}
	save(t, "baseline.nft", rules)
	fwd := mustExec(t, "sysctl", "-n", "net.ipv4.ip_forward")
	noteSummary("AT00 handshakes=5 forward=" + strings.TrimSpace(fwd) + " baseline=1")
}
