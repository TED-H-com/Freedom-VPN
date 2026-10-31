//go:build netlab

package netlab

import (
	"strings"
	"testing"
	"time"
)

// TestReviewR15_TcMirredAndPolicyRoutingBypass runs two paths nftables does
// not list. VerifyServing reads `nft -j list ruleset`. A clsact mirred
// redirect and an ip rule are not in that document. A finding requires the
// counters to show the packet leaving on the uplink while verify still
// returns 0.
func TestReviewR15_TcMirredAndPolicyRoutingBypass(t *testing.T) {
	skipLab(t)
	t.Cleanup(func() {
		_, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
		_, _, _ = dockerExec("ip", "rule", "del", "from", "100.64.9.9", "lookup", "100")
		_, _, _ = dockerExec("ip", "route", "flush", "table", "100")
	})
	_, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
	_, _, _ = dockerExec("ip", "rule", "del", "from", "100.64.9.9", "lookup", "100")
	_, _, _ = dockerExec("ip", "route", "flush", "table", "100")

	now := time.Now().UTC()
	m := functionalModel(now, 91)
	applyAt(t, m)
	flushConntrack(t)
	if _, stderr, code := runLab(t, m, "verify"); code != 0 {
		t.Fatalf("clean verify exit %d\n%s", code, stderr)
	}

	t.Run("tc_clsact_mirred", func(t *testing.T) {
		t.Cleanup(func() {
			_, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
		})
		mustExec(t, "tc", "qdisc", "add", "dev", "fv-fh", "clsact")
		mustExec(t, "tc", "filter", "add", "dev", "fv-fh", "ingress", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
		if _, stderr, code := runLab(t, m, "verify"); code != 0 {
			t.Fatalf("verify saw the tc filter: exit %d\n%s", code, stderr)
		}
		var obs IngressObservation
		uplink := captureDuring(t, "root", "fv-ih", "udp port 39997", func() {
			obs = injectPort(t, "fv-fh", "100.64.9.9", "203.0.113.10", "39997")
		})
		ctunku := counterDelta(obs, "ctunku").Packets
		uplinkN := 0
		if strings.TrimSpace(uplink) != "" {
			uplinkN = 1
		}
		t.Logf("tc mirred: ctunku=%d uplink=%q", ctunku, uplink)
		if uplinkN > 0 && ctunku == 0 {
			t.Fatalf("TODO(review): tc clsact mirred on fv-fh redirected a non-admitted packet onto fv-ih (capture %q) without incrementing ctunku, and fvpn-netlab verify still returned 0. The redirect is not in the nft ruleset VerifyServing reads.", uplink)
		}
	})

	t.Run("ip_rule", func(t *testing.T) {
		t.Cleanup(func() {
			_, _, _ = dockerExec("ip", "rule", "del", "from", "100.64.9.9", "lookup", "100")
			_, _, _ = dockerExec("ip", "route", "flush", "table", "100")
		})
		mustExec(t, "ip", "route", "add", "203.0.113.10/32", "dev", "fv-ih", "table", "100")
		mustExec(t, "ip", "rule", "add", "from", "100.64.9.9", "lookup", "100")
		if _, stderr, code := runLab(t, m, "verify"); code != 0 {
			t.Fatalf("verify saw the ip rule: exit %d\n%s", code, stderr)
		}
		flushConntrack(t)
		var obs IngressObservation
		uplink := captureDuring(t, "root", "fv-ih", "udp port 39996", func() {
			obs = injectPort(t, "fv-fh", "100.64.9.9", "203.0.113.10", "39996")
		})
		ctunku := counterDelta(obs, "ctunku").Packets
		uplinkN := 0
		if strings.TrimSpace(uplink) != "" {
			uplinkN = 1
		}
		t.Logf("ip rule: ctunku=%d uplink=%q", ctunku, uplink)
		if uplinkN > 0 && ctunku == 0 {
			t.Fatalf("TODO(review): ip rule from 100.64.9.9 lookup 100 sent a non-admitted packet out fv-ih (capture %q) without incrementing ctunku, and fvpn-netlab verify still returned 0. Policy routing is not in the nft ruleset VerifyServing reads.", uplink)
		}
	})
}
