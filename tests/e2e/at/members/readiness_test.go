package members

import (
	"encoding/json"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func servingProducer(trace []json.RawMessage, boot, lease uuid.UUID, generation decstr.U64) json.RawMessage {
	if boot == uuid.Nil || lease == uuid.Nil || generation == 0 {
		return nil
	}
	for i := len(trace) - 1; i >= 0; i-- {
		var m agent.Measurement
		if json.Unmarshal(trace[i], &m) != nil {
			return nil
		}
		if m.BootID == boot && (m.Baseline != nil || (m.Timing != nil && !m.Timing.Succeeded)) {
			return nil
		}
		if m.BootID != boot || m.Heartbeat == nil {
			continue
		}
		// A newer non-serving heartbeat must not be hidden by an older serving row.
		if m.RunID != uuid.Nil && m.Sequence > 0 && m.Heartbeat.Mono > 0 && m.Heartbeat.Serving && m.Heartbeat.Generation == generation && m.Lease != nil && m.Lease.ID == lease && m.Lease.Generation == generation && m.Lease.Members > 0 {
			return trace[i]
		}
		return nil
	}
	return nil
}

func TestAT02_F12ReadinessRejectsEmptyLeaseAndLaterWithdrawal(t *testing.T) {
	boot, lease, run := uuid.New(), uuid.New(), uuid.New()
	m := agent.Measurement{BootID: boot, RunID: run, Sequence: 1, Lease: &agent.MeasurementLease{ID: lease, Generation: 10}, Heartbeat: &agent.MeasurementHeartbeat{Mono: 100, Serving: true, Generation: 10}}
	raw, _ := json.Marshal(m)
	if servingProducer([]json.RawMessage{raw}, boot, lease, 10) != nil {
		t.Fatal("empty grant passed")
	}
	m.Lease.Members = 1
	raw, _ = json.Marshal(m)
	if servingProducer([]json.RawMessage{raw}, boot, lease, 10) == nil {
		t.Fatal("populated serving rejected")
	}
	m.Sequence = 2
	m.Heartbeat.Serving = false
	withdrawn, _ := json.Marshal(m)
	if servingProducer([]json.RawMessage{raw, withdrawn}, boot, lease, 10) != nil {
		t.Fatal("later withdrawal hidden")
	}
	m.Heartbeat = nil
	m.Baseline = &agent.MeasurementBaseline{Cause: "policy_missing", Start: 101, End: 110, Succeeded: true}
	baseline, _ := json.Marshal(m)
	if servingProducer([]json.RawMessage{raw, baseline}, boot, lease, 10) != nil {
		t.Fatal("later baseline hidden")
	}
	if servingProducer([]json.RawMessage{raw}, boot, uuid.New(), 10) != nil {
		t.Fatal("wrong lease joined")
	}
	if servingProducer([]json.RawMessage{raw}, uuid.New(), lease, 10) != nil {
		t.Fatal("wrong boot joined")
	}
}
