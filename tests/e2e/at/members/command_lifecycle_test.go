package members

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Additive TEST ONLY records belong to their enclosing Apply identity. Missing
// instrumentation is unavailable, including every historical pre-opt-in stream.
func validateCommandLifecycle(t *enforce.TimingEvidence) error {
	if t == nil || len(t.Commands) == 0 {
		return nil
	}
	if t.Operation != "apply" || t.Clock != "CLOCK_MONOTONIC" || t.Start <= 0 || t.End < t.Start {
		return fmt.Errorf("commands without an ordered Apply")
	}
	previous := t.Start
	for i, c := range t.Commands {
		bad := func() error { return fmt.Errorf("command %d invalid lifecycle", i) }
		switch c.Kind {
		case "nft", "tc", "conntrack", "ip", "other":
		default:
			return bad()
		}
		switch c.Phase {
		case "prepare", "quarantine", "shape", "install", "release", "verify", "revoke", "rollback":
		default:
			return bad()
		}
		// Available samples must be ordered; missing samples never become zero
		// timestamps or exact child/kernel execution claims.
		for _, s := range []enforce.LifecycleStamp{c.Before.Bracket.Before, c.Before.Bracket.After, c.StartAttempt, c.StartReturn, c.WaitReturn, c.After.Bracket.Before, c.After.Bracket.After} {
			if s.Available {
				if s.NS <= 0 || s.NS < previous || s.NS > t.End {
					return bad()
				}
				previous = s.NS
			} else if s.NS != 0 {
				return bad()
			}
		}
		if c.ChildStartExact.Available || c.ChildStartExact.NS != 0 || c.ChildStartedBy.Available && (!c.Started || c.ChildStartedBy != c.StartReturn) || !c.Started && (c.WaitReturn.Available || c.ExitAvailable || c.OK) {
			return bad()
		}
		if c.CancelObserved.Available && (c.CancelObserved.NS < c.StartAttempt.NS || c.WaitReturn.Available && c.CancelObserved.NS > c.WaitReturn.NS) {
			return bad()
		}
		if c.ContextDeadline.Before.Available != c.ContextDeadline.After.Available || c.ContextDeadline.Before.Available && (c.ContextDeadline.Before.NS <= 0 || c.ContextDeadline.After.NS < c.ContextDeadline.Before.NS) {
			return bad()
		}
		if c.UserCPU < 0 || c.SystemCPU < 0 || c.VoluntarySwitches < 0 || c.InvoluntarySwitches < 0 {
			return bad()
		}
	}
	return nil
}

func validateProducerCommands(raw json.RawMessage) error {
	var row struct {
		Timing *enforce.TimingEvidence `json:"timing"`
	}
	if err := json.Unmarshal(raw, &row); err != nil {
		return err
	}
	return validateCommandLifecycle(row.Timing)
}

func TestAT02_F14CommandLifecycleJoinsApply(t *testing.T) {
	stamp := func(n time.Duration) enforce.LifecycleStamp { return enforce.LifecycleStamp{NS: n, Available: true} }
	timing := enforce.TimingEvidence{Clock: "CLOCK_MONOTONIC", Operation: "apply", Start: 1, End: 20, Commands: []enforce.CommandLifecycle{{Kind: "nft", Phase: "install", StartAttempt: stamp(2), StartReturn: stamp(3), ChildStartedBy: stamp(3), WaitReturn: stamp(5), Started: true, OK: true}}}
	for _, failed := range []bool{false, true} {
		timing.Succeeded = !failed
		raw, _ := json.Marshal(map[string]any{"timing": timing, "sequence": "9007199254740993"})
		d, err := diagnoseApply(raw)
		if err != nil || !d.CommandLifecycleAvailable || len(d.Commands) != 1 || d.Sequence.String() != "9007199254740993" {
			t.Fatal("lost Apply join", err)
		}
	}
	timing.Commands[0].WaitReturn = stamp(21)
	if validateCommandLifecycle(&timing) == nil {
		t.Fatal("accepted command outside Apply")
	}
	timing.Commands = nil
	if validateCommandLifecycle(&timing) != nil {
		t.Fatal("rejected historical instrumentation absence")
	}
	d, err := diagnoseApply(json.RawMessage(`{"timing":{"operation":"apply"}}`))
	if err != nil || d.CommandLifecycleStatus != "unavailable" {
		t.Fatal("invented historical lifecycle")
	}
}
