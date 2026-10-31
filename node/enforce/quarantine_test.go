package enforce

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Stateful executor keeps kernel markers separate from each function call,
// allowing recovery to exercise the same contract as a fresh process.
type quarantineExec struct {
	m                             Model
	plan                          RenderPlan
	up, marker, baseline, foreign bool
	fault                         string
	ops                           []string
}

func newQuarantineExec(t *testing.T) *quarantineExec {
	t.Helper()
	m := smallModel()
	m.Revoke = nil
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	return &quarantineExec{m: m, plan: plan, up: true, foreign: true}
}

func (f *quarantineExec) LookPath(name string) (string, error) { return name, nil }
func (f *quarantineExec) NftBatch(_ context.Context, script string) error {
	kind := "serving"
	switch {
	case strings.HasPrefix(script, "delete set"):
		kind = "clear"
	case strings.Contains(script, "fvpn_baseline=1"):
		kind = "baseline"
	case strings.Contains(script, "type ifname"):
		kind = "marker"
	}
	f.ops = append(f.ops, kind)
	if f.fault == kind {
		return fmt.Errorf("TEST ONLY: command output must not enter evidence")
	}
	switch kind {
	case "clear":
		f.marker = false
	case "marker":
		f.marker = true
	case "baseline":
		f.baseline = true
	case "serving":
		f.baseline = false
	}
	return nil
}
func (f *quarantineExec) TCBatch(_ context.Context, script string) error {
	f.ops = append(f.ops, "tc_batch")
	if f.foreign {
		return fmt.Errorf("foreign tc must be checked before mutation")
	}
	return nil
}
func (f *quarantineExec) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.ops = append(f.ops, key)
	if key == "ip link set dev fv-fh down" || key == "ip link set dev fv-fh up" {
		direction := args[len(args)-1]
		if f.fault == direction {
			return nil, fmt.Errorf("TEST ONLY: link mutation failed")
		}
		f.up = direction == "up"
		if f.up && f.fault == "foreign_after_up" {
			f.foreign = true
		}
		return nil, nil
	}
	if key == "ip -j link show dev fv-fh" {
		if f.fault == "link_read" {
			return []byte(`[]`), nil
		}
		if f.up {
			return []byte(`[{"ifname":"fv-fh","flags":["UP"],"operstate":"UNKNOWN"}]`), nil
		}
		return []byte(`[{"ifname":"fv-fh","flags":[],"operstate":"DOWN"}]`), nil
	}
	if name == "nft" {
		if f.fault == "nft_read" {
			return nil, fmt.Errorf("TEST ONLY: nft read failed")
		}
		raw := fixtureServing(f.m, f.plan.Assignments)
		if f.baseline {
			raw = fixtureBaseline()
		}
		if f.marker {
			var doc struct {
				Nftables []any `json:"nftables"`
			}
			_ = json.Unmarshal(raw, &doc)
			if f.baseline {
				for id, minor := range f.plan.Assignments {
					doc.Nftables = append(doc.Nftables, map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "comment": fmt.Sprintf("fvpn_class=%s:%x", id, minor), "expr": []any{map[string]any{"match": map[string]any{"left": "never", "right": "never"}}}}})
				}
			}
			doc.Nftables = append(doc.Nftables, map[string]any{"set": map[string]any{"family": "inet", "table": "fvpn", "name": "quarantine_fv-fh", "type": "ifname"}})
			raw, _ = json.Marshal(doc)
		}
		return raw, nil
	}
	if name == "tc" && f.fault == "tc_read" {
		return nil, fmt.Errorf("TEST ONLY: unavailable tc")
	}
	if name == "tc" && len(args) > 1 && args[0] == "-j" {
		if args[1] == "filter" {
			if f.foreign && key == "tc -j filter show dev fv-fh ingress" {
				return []byte(`[{"kind":"matchall","options":{"actions":[{"kind":"mirred"}]}}]`), nil
			}
			return []byte(`[]`), nil
		}
		pair := fixtureQoS(f.m, f.plan.Assignments)[f.m.Datapath.MemberIngress]
		if args[1] == "qdisc" {
			return pair[0], nil
		}
		if args[1] == "class" {
			return pair[1], nil
		}
	}
	if name == "ip" && len(args) > 2 && args[0] == "-j" && (args[1] == "-4" || args[1] == "-6") {
		return []byte(`[]`), nil
	}
	return nil, fmt.Errorf("unexpected command %s", key)
}

func TestD090_ISO01_QuarantineAndVerifiedRecovery(t *testing.T) {
	ctx := context.Background()
	f := newQuarantineExec(t)
	for i := range 3 {
		if i == 2 {
			f.up = true
		} // another daemon raised the member link
		q, err := Quarantine(ctx, f, f.m)
		if err != nil || !q.Quarantined || !q.Recorded || !q.Baseline.Installed || !q.Baseline.Verified || len(q.Links) != 1 || q.Links[0].Before.AdminUp != (i != 1) || q.Links[0].After.AdminUp {
			t.Fatalf("step %d: %+v %v", i, q, err)
		}
		ev, err := VerifyServing(ctx, f, f.m, f.plan.Assignments)
		if ReasonCode(err) != ReasonTCForeignFilter || !ev.Quarantined || len(ev.TC) == 0 {
			t.Fatalf("hidden filter: %+v %v", ev, err)
		}
	}
	result, err := Apply(ctx, f, f.m, nil)
	if ReasonCode(err) != ReasonTCForeignFilter || result.Verified || f.up || !f.foreign || result.Quarantine == nil || !result.Quarantine.Quarantined {
		t.Fatalf("dirty recovery: %+v %v", result, err)
	}
	for _, op := range f.ops {
		if op == "tc_batch" || strings.Contains(op, "qdisc del") || strings.Contains(op, "filter del") || strings.Contains(op, "link set dev fv-ih") {
			t.Fatalf("ownership violation: %s", op)
		}
	}
	// A baseline refresh also retains the marker and all tc objects.
	if err := ApplyBaseline(ctx, f, f.m.Datapath); err != nil || !f.marker || f.up {
		t.Fatalf("baseline refresh: %v", err)
	}
	f.foreign = false
	f.ops = nil
	result, err = Apply(ctx, f, f.m, nil)
	if err != nil || !result.Verified || result.Evidence.Quarantined || !f.up || f.marker {
		t.Fatalf("recovery: %+v %v", result, err)
	}
	upAt, readsBefore, readsAfter := -1, 0, 0
	for i, op := range f.ops {
		if op == "ip link set dev fv-fh up" {
			upAt = i
		}
		if op == "nft -j list ruleset" {
			if upAt < 0 {
				readsBefore++
			} else {
				readsAfter++
			}
		}
	}
	if upAt < 0 || readsBefore != 1 || readsAfter != 1 {
		t.Fatalf("recovery verification order: %v", f.ops)
	}
}

func TestD090_QuarantinePartialFailures(t *testing.T) {
	for _, fault := range []string{"marker", "baseline", "link_read", "down", "nft_read"} {
		t.Run(fault, func(t *testing.T) {
			f := newQuarantineExec(t)
			f.fault = fault
			q, err := Quarantine(context.Background(), f, f.m)
			if err == nil {
				t.Fatalf("accepted fault %+v", q)
			}
			ops := strings.Join(f.ops, "\n")
			if !strings.Contains(ops, "ip link set dev fv-fh down") || (fault != "nft_read" && !strings.Contains(ops, "baseline")) {
				t.Fatalf("skipped independent containment: %s", ops)
			}
			raw, _ := json.Marshal(q)
			if strings.Contains(string(raw), "TEST ONLY") || strings.Contains(err.Error(), "TEST ONLY") {
				t.Fatal("command output leaked")
			}
			if fault == "down" && (q.Quarantined || !q.Baseline.Verified) {
				t.Fatalf("incorrect partial evidence: %+v", q)
			}
			if fault == "baseline" && (q.Quarantined || f.up) {
				t.Fatalf("baseline failure prevented link-down: %+v", q)
			}
		})
	}
}

func TestD090_RecoveryFailureRequarantines(t *testing.T) {
	for _, fault := range []string{"up", "foreign_after_up", "clear"} {
		t.Run(fault, func(t *testing.T) {
			f := newQuarantineExec(t)
			if _, err := Quarantine(context.Background(), f, f.m); err != nil {
				t.Fatal(err)
			}
			f.foreign, f.fault = false, fault
			res, err := Apply(context.Background(), f, f.m, nil)
			if err == nil || res.Verified || f.up || !f.marker || !f.baseline || res.Quarantine == nil || !res.Quarantine.Quarantined {
				t.Fatalf("failed recovery: %+v %v", res, err)
			}
		})
	}
}

func TestD090_QuarantineRejectsUplinkAlias(t *testing.T) {
	f := newQuarantineExec(t)
	f.m.Datapath.MemberIngress = f.m.Datapath.Uplink
	if _, err := Quarantine(context.Background(), f, f.m); err == nil || len(f.ops) != 0 {
		t.Fatal("mutated ambiguous member/uplink")
	}
}

func TestD090_BaselineRetainsForeignRootAttachments(t *testing.T) {
	f := newQuarantineExec(t)
	// No quarantine marker yet: even an ordinary baseline must not delete
	// a foreign filter by deleting its parent root/class.
	if err := ApplyBaseline(context.Background(), f, f.m.Datapath); err != nil {
		t.Fatal(err)
	}
	for _, op := range f.ops {
		if strings.Contains(op, "qdisc del") || strings.Contains(op, "filter del") {
			t.Fatalf("deleted a foreign attachment: %s", op)
		}
	}
}

func TestD090_PublicVerificationCannotReleaseMarker(t *testing.T) {
	f := newQuarantineExec(t)
	f.marker, f.foreign = true, false
	ev, err := VerifyServing(context.Background(), f, f.m, f.plan.Assignments)
	if err == nil || !ev.Quarantined || !f.marker {
		t.Fatalf("verification released quarantine: %+v %v", ev, err)
	}
	for _, op := range f.ops {
		if strings.Contains(op, "link set") || op == "clear" {
			t.Fatalf("verification mutated state: %s", op)
		}
	}
}

func TestD090_ApplyTCReadFailureQuarantines(t *testing.T) {
	f := newQuarantineExec(t)
	f.fault = "tc_read"
	res, err := Apply(context.Background(), f, f.m, nil)
	if ReasonCode(err) != ReasonTCForeignFilter || res.Verified || f.up || res.Quarantine == nil || !res.Quarantine.Quarantined {
		t.Fatalf("unreadable tc remained serving: verified=%v up=%v err=%v", res.Verified, f.up, err)
	}
}
