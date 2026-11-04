package isolation

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAT13_MemberExpiryRejectsFollowup20Replay(t *testing.T) {
	const path = "../../../../docs/e2e/at/isolation/history/follow-up-20-run/AT13.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.RunID != "c52bea6c-cd91-4086-a779-656b01277dcc" {
		t.Fatal("wrong retained run")
	}
	for _, c := range rec.Checks {
		if c.Name != "lease member expiry under control loss" {
			continue
		}
		if c.Status != StatusPass {
			t.Fatal("historical pass must remain recorded")
		}
		var read MemberExpiryServingRead
		for _, o := range c.Observations {
			if o.Name == "heartbeat" {
				read.Heartbeat = o.Detail
			}
			if o.Name == "forward_chain" {
				read.ForwardChain = o.Detail
			}
		}
		// The original record's heartbeat alone refutes serving. Do not invent a
		// chain read absent from this original check or rewrite its historical pass.
		status, detail := MemberExpiryStatus(c.Status, []MemberExpiryServingRead{read})
		if status != StatusFail || !strings.Contains(detail, `serving="0" reason="safety_check_failed"`) {
			t.Fatalf("%s: %s", status, detail)
		}
		t.Logf("%s: historical=%s, new check=%s: %s", rec.RunID, c.Status, status, detail)
		return
	}
	t.Fatal("retained member expiry check missing")
}

func TestAT13_MemberExpiryServingNode(t *testing.T) {
	serving := MemberExpiryServingRead{Heartbeat: "v1\nserving 1\nreason serving\n", ForwardChain: `table inet fvpn {
 chain forward {
 type filter hook forward priority filter; policy drop;
 iifname "fvmb0" ip saddr @allowed_v4 counter accept
 }
}`}
	reads := []MemberExpiryServingRead{serving, serving, serving, serving, serving, serving}
	if status, detail := MemberExpiryStatus(StatusPass, reads); status != StatusPass {
		t.Fatal(status, detail)
	}
	for _, prior := range []string{StatusFail, StatusBlocked} {
		if got, _ := MemberExpiryStatus(prior, reads); got != prior {
			t.Fatal("changed denial verdict", got)
		}
	}
	for name, bad := range map[string]MemberExpiryServingRead{
		"withdrawn during probes":               {Heartbeat: "serving 0\nreason safety_check_failed", ForwardChain: serving.ForwardChain},
		"stale serving heartbeat over baseline": {Heartbeat: serving.Heartbeat, ForwardChain: `chain forward { comment "fvpn_baseline=1"; policy drop; }`},
		"missing chain":                         {Heartbeat: serving.Heartbeat},
		"failed read":                           {Heartbeat: serving.Heartbeat, ForwardChain: serving.ForwardChain, ReadError: "read failed"},
		"ambiguous heartbeat":                   {Heartbeat: "serving 0\nserving 1", ForwardChain: serving.ForwardChain},
	} {
		t.Run(name, func(t *testing.T) {
			copyReads := append([]MemberExpiryServingRead(nil), reads...)
			copyReads[2] = bad
			if got, detail := MemberExpiryStatus(StatusPass, copyReads); got != StatusFail {
				t.Fatal(got, detail)
			}
		})
	}
	if got, _ := MemberExpiryStatus(StatusPass, nil); got != StatusFail {
		t.Fatal("missing evidence passed")
	}
}
