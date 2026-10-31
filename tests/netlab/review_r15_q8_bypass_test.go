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

// TestReviewR15_TcMirredAndPolicyRoutingBypass retains the measured tc bypass
// and policy-routing control. Serving must reject the live tc redirect even
// though its packets never reach the nft forward counter.
func TestReviewR15_TcMirredAndPolicyRoutingBypass(t *testing.T) {
	skipLab(t)
	t.Cleanup(func() { applyBaseline(t) })
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
		stdout, stderr, code := runLab(t, m, "verify")
		if code != 3 || !strings.Contains(stderr, "TC_FOREIGN_FILTER") || !strings.Contains(stderr, "fv-fh") {
			t.Fatalf("verify did not reject the tc filter: exit %d\n%s", code, stderr)
		}
		if !strings.Contains(stdout, `"kind":"mirred"`) && !strings.Contains(stdout, `"actions":["mirred"]`) {
			t.Fatalf("missing mirred read-back evidence: %s", stdout)
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
		if uplinkN != 1 || ctunku != 0 {
			t.Fatalf("tc bypass trigger was not exercised: capture %q ctunku=%d", uplink, ctunku)
		}
		stdout, stderr, code = runLab(t, m, "apply")
		if code != 3 || !strings.Contains(stderr, "TC_FOREIGN_FILTER") || !strings.Contains(stdout, `"verified":false`) {
			t.Fatalf("apply accepted foreign tc: exit %d stdout=%s stderr=%s", code, stdout, stderr)
		}
		if !strings.Contains(mustExec(t, "nft", "list", "table", "inet", "fvpn"), "fvpn_baseline=1") {
			t.Fatal("apply did not restore the baseline")
		}
	})

	t.Run("ip_rule", func(t *testing.T) {
		applyAt(t, m)
		t.Cleanup(func() {
			_, _, _ = dockerExec("ip", "rule", "del", "from", "100.64.9.9", "lookup", "100")
			_, _, _ = dockerExec("ip", "route", "flush", "table", "100")
		})
		mustExec(t, "ip", "route", "add", "203.0.113.10/32", "dev", "fv-ih", "table", "100")
		mustExec(t, "ip", "rule", "add", "from", "100.64.9.9", "lookup", "100")
		stdout, stderr, code := runLab(t, m, "verify")
		if code != 0 {
			t.Fatalf("verify saw the ip rule: exit %d\n%s", code, stderr)
		}
		var evidence enforce.ServingEvidence
		if err := json.Unmarshal([]byte(stdout), &evidence); err != nil || len(evidence.PolicyRules) != 2 || evidence.PolicyRules[0].Count < 4 || evidence.PolicyRules[0].Error != "" {
			t.Fatalf("policy routing evidence missing: %s, err=%v", stdout, err)
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
		if uplinkN != 0 || ctunku != 1 {
			t.Fatalf("policy routing escaped admission: capture %q ctunku=%d", uplink, ctunku)
		}
	})
}

// Root, HTB class and fq_codel filter blocks are independent of clsact.
// Exercise real kernel attachments so the command coverage cannot be proved
// solely by a fake that returns the same filters for every parent.
func TestReviewR15_TCFilterAttachments(t *testing.T) {
	skipLab(t)
	t.Cleanup(func() { applyBaseline(t) })
	m := functionalModel(time.Now().UTC(), 92)
	plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatal(err)
	}
	minor := plan.Assignments[m.QoS.Members[0].MemberID]
	for _, iface := range []string{"fv-fh", "fv-ih"} {
		for _, parent := range []string{"1:", "1:1", "1:fffe", fmt.Sprintf("1:%x", minor), fmt.Sprintf("%x:", minor), "egress"} {
			t.Run(iface+"_"+parent, func(t *testing.T) {
				applyAt(t, m)
				args := []string{"tc", "filter", "add", "dev", iface}
				if parent == "egress" {
					mustExec(t, "tc", "qdisc", "add", "dev", iface, "clsact")
					t.Cleanup(func() { _, _, _ = dockerExec("tc", "qdisc", "del", "dev", iface, "clsact") })
					args = append(args, "egress")
				} else {
					args = append(args, "parent", parent)
					t.Cleanup(func() { mustExec(t, "tc", "filter", "del", "dev", iface, "parent", parent, "chain", "7") })
				}
				// Chain 7 proves the listing is not restricted to chain 0.
				args = append(args, "chain", "7", "matchall", "action", "skbedit", "mark", "0x1234")
				mustExec(t, args...)
				stdout, stderr, code := runLab(t, m, "verify")
				if code != 3 || !strings.Contains(stderr, "TC_FOREIGN_FILTER") || !strings.Contains(stderr, iface) || !strings.Contains(stdout, `"actions":["skbedit"]`) {
					t.Fatalf("attachment accepted or unrecorded: exit %d stdout=%s stderr=%s", code, stdout, stderr)
				}
			})
		}
	}
}
