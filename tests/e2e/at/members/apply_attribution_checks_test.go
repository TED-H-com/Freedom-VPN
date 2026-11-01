package members

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

func TestAT02_F12HistoricalInstallDoesNotProveContention(t *testing.T) {
	raw, e := os.ReadFile("../../../../docs/e2e/at/members/history/follow-up-11/AT02-enforcement-timing.json")
	if e != nil {
		t.Fatal(e)
	}
	var artifact struct {
		Records []json.RawMessage `json:"records"`
	}
	if json.Unmarshal(raw, &artifact) != nil {
		t.Fatal("trace")
	}
	count := 0
	for _, row := range artifact.Records {
		var m agent.Measurement
		_ = json.Unmarshal(row, &m)
		if m.Timing == nil || m.Timing.Operation != "apply" || m.Timing.CandidateGeneration != 13 || m.Timing.Succeeded {
			continue
		}
		count++
		v, e := diagnoseApply(row)
		if e != nil || v.FailureStage != "install" || v.Verification != "missing" || v.Label != "failed Apply" || v.Cause != "unproven" {
			t.Fatalf("unsupported attribution: %+v %v", v, e)
		}
		m.Host = &agent.MeasurementHost{Before: agent.HostSample{Start: m.Timing.Start - 2*time.Millisecond, End: m.Timing.Start - time.Millisecond, OK: true, Total: 100, Idle: 50, Running: 10}, After: agent.HostSample{Start: m.Timing.End + time.Millisecond, End: m.Timing.End + 2*time.Millisecond, OK: true, Total: 200, Idle: 50, Running: 10}}
		encoded, _ := json.Marshal(m)
		v, e = diagnoseApply(encoded)
		if e != nil || v.Label != "fail-closed install overrun with recorded host contention" || v.Cause != "unproven" || v.Idle == nil || *v.Idle != 0 {
			t.Fatal("lost joined host context", v, e)
		}
		m.Host.After.OK = false
		encoded, _ = json.Marshal(m)
		v, _ = diagnoseApply(encoded)
		if v.Idle != nil || v.Label != "failed Apply" {
			t.Fatal("unavailable is not quiet or contention")
		}
	}
	if count != 8 {
		t.Fatalf("historical failed installs=%d", count)
	}
}

func TestAT02_F12MissingCaptureCannotPassRecovery(t *testing.T) {
	_, ok, observed := siblingRecovery(nil, nil)
	if ok || observed {
		t.Fatal("missing measurement passed")
	}
}

func TestAT02_F12RecoveryRequiresNextApplyAndCountsEveryLoss(t *testing.T) {
	// TEST ONLY synthetic command and packet clocks, never a measured AT row.
	boot := uuid.New()
	run := uuid.New()
	fail := agent.Measurement{RunID: run, BootID: boot, Sequence: 1, Timing: &enforce.TimingEvidence{Operation: "apply", Start: 100, End: 200}}
	base := agent.Measurement{RunID: run, BootID: boot, Sequence: 2, Baseline: &agent.MeasurementBaseline{Cause: "apply_rollback", Start: 201, End: 210, Succeeded: true}}
	good := agent.Measurement{RunID: run, BootID: boot, Sequence: 3, Timing: &enforce.TimingEvidence{Operation: "apply", Start: 300, End: 310, Succeeded: true}}
	trace := []json.RawMessage{}
	for _, m := range []agent.Measurement{fail, base, good} {
		b, _ := json.Marshal(m)
		trace = append(trace, b)
	}
	cap := siblingObservation{Boot: boot.String(), Start: 50, End: 500, StatsAvailable: true, Packets: []siblingPacket{{1, 80, "fvmb0"}, {1, 81, "fvup0"}, {2, 220, "fvmb0"}, {3, 320, "fvmb0"}, {3, 321, "fvup0"}}}
	raw, _ := json.Marshal(cap)
	out, ok, observed := siblingRecovery(trace, raw)
	if !ok || !observed || out["lost_admitted_packets"] != 1 {
		t.Fatal("valid bracket lost", out)
	}
	cap.Packets = append(cap.Packets, siblingPacket{4, 400, "fvmb0"})
	raw, _ = json.Marshal(cap)
	out, ok, _ = siblingRecovery(trace, raw)
	if ok || out["lost_admitted_packets"] != 2 || out["losses_outside_baseline_or_completed_guard"] != 1 {
		t.Fatal("post-recovery loss waived", out)
	}
	raw, _ = json.Marshal(cap)
	_, ok, _ = siblingRecovery(trace[:2], raw)
	if ok {
		t.Fatal("missing next Apply passed")
	}
}
