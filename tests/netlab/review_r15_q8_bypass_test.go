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
		t.Logf("D090 before quarantine: verify_exit=%d reason=TC_FOREIGN_FILTER uplink_packets=%d ctunku_delta=%d", code, uplinkN, ctunku)
		// There is deliberately no Apply between the injection, verification
		// and quarantine: this is the agent's pre-heartbeat failure path.
		stdout, stderr, code = runLab(t, m, "quarantine")
		var q enforce.QuarantineEvidence
		if err := json.Unmarshal([]byte(stdout), &q); err != nil || code != 0 || !q.Quarantined || !q.Recorded || !q.Baseline.Verified || len(q.Links) != 1 || q.Links[0].Interface != "fv-fh" || q.Links[0].Before == nil || !q.Links[0].Before.AdminUp || q.Links[0].After == nil || q.Links[0].After.AdminUp {
			t.Fatalf("quarantine exit=%d evidence=%s stderr=%s err=%v", code, stdout, stderr, err)
		}
		t.Logf("D090 quarantine: %s", strings.TrimSpace(stdout))
		save(t, "quarantine.json", stdout)
		filter := mustExec(t, "tc", "-j", "filter", "show", "dev", "fv-fh", "ingress")
		if !strings.Contains(filter, "mirred") || !strings.Contains(mustExec(t, "tc", "-j", "qdisc", "show", "dev", "fv-fh"), "clsact") {
			t.Fatal("quarantine removed foreign tc state")
		}
		stdout, stderr, code = runLab(t, m, "verify")
		if code != 3 || !strings.Contains(stderr, "TC_FOREIGN_FILTER") || !strings.Contains(stdout, `"quarantined":true`) || !strings.Contains(stdout, `"actions":["mirred"]`) {
			t.Fatalf("down link hid foreign filter: exit=%d stdout=%s stderr=%s", code, stdout, stderr)
		}
		uplink = captureDuring(t, "root", "fv-ih", "udp port 39997", func() {
			obs = injectPort(t, "fv-fh", "100.64.9.9", "203.0.113.10", "39997")
		})
		ctunku = counterDelta(obs, "ctunku").Packets
		if strings.TrimSpace(uplink) != "" || ctunku != 0 {
			t.Fatalf("quarantine did not contain bypass: capture=%q ctunku_delta=%d", uplink, ctunku)
		}
		control := mustExec(t, "python3", "-c", "import socket; s=socket.create_connection(('203.0.113.10',80),3); s.sendall(b'GET / HTTP/1.0\\r\\n\\r\\n'); assert s.recv(64); s.close(); print('control-ok')")
		assertLabLink(t, "fv-fh", false)
		assertLabLink(t, "fv-ih", true)
		t.Logf("D090 contained: verify_exit=%d reason=TC_FOREIGN_FILTER quarantined=true uplink_packets=0 ctunku_before=%d ctunku_after=%d control=%s", code, obs.Before["ctunku"].Packets, obs.After["ctunku"].Packets, strings.TrimSpace(control))
		// Idempotent repeat, and a daemon-like raise followed by the next step.
		for _, raised := range []bool{false, true} {
			if raised {
				mustExec(t, "ip", "link", "set", "dev", "fv-fh", "up")
			}
			stdout, stderr, code = runLab(t, m, "quarantine")
			if code != 0 || json.Unmarshal([]byte(stdout), &q) != nil || !q.Quarantined || q.Links[0].Before.AdminUp != raised {
				t.Fatalf("repeat raised=%v exit=%d stdout=%s stderr=%s", raised, code, stdout, stderr)
			}
		}
		stdout, stderr, code = runLab(t, m, "apply")
		if code != 3 || !strings.Contains(stderr, "TC_FOREIGN_FILTER") || !strings.Contains(stdout, `"verified":false`) {
			t.Fatalf("apply accepted foreign tc: exit %d stdout=%s stderr=%s", code, stdout, stderr)
		}
		assertLabLink(t, "fv-fh", false)
		// The test owns its injection; enforcement never deletes this state.
		mustExec(t, "tc", "qdisc", "del", "dev", "fv-fh", "clsact")
		applyAt(t, m)
		assertLabLink(t, "fv-fh", true)
		assertLabLink(t, "fv-ih", true)
		stdout, stderr, code = runLab(t, m, "verify")
		if code != 0 || !strings.Contains(stdout, `"quarantined":false`) {
			t.Fatalf("clean recovery: exit=%d stdout=%s stderr=%s", code, stdout, stderr)
		}
		restored := runProbe(t, []probeCase{{"recovered-member", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 2000, "", "", "", "", 0}})
		requireProbe(t, restored)
		// Re-run the existing boot-baseline oracle before the next scenario.
		applyBaseline(t)
		TestAT00_Topology(t)
		noteSummary("D090 post-Apply injection verify=3 TC_FOREIGN_FILTER; pre-quarantine uplink=1 ctunku_delta=0; quarantined uplink=0 ctunku_delta=0 control=pass; repeated quarantine=pass; clean Apply links=up verify=0 member=pass AT00=pass")

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

func assertLabLink(t *testing.T, iface string, up bool) {
	t.Helper()
	raw := mustExec(t, "ip", "-j", "link", "show", "dev", iface)
	var links []struct {
		Flags []string `json:"flags"`
	}
	if err := json.Unmarshal([]byte(raw), &links); err != nil || len(links) != 1 {
		t.Fatalf("link %s: %s (%v)", iface, raw, err)
	}
	found := false
	for _, flag := range links[0].Flags {
		found = found || flag == "UP"
	}
	if found != up {
		t.Fatalf("link %s admin_up=%v want %v", iface, found, up)
	}
}
