package agent

import (
	"context"
	"reflect"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// bindingDeadline uses the installed model's clock mapping. Member limits
// already include the node lease, desired expiry and offline window. Cap the
// wall duration before adding Mono so distant ValidUntil values cannot overflow.
func bindingDeadline(m enforce.Model, b enforce.Binding) time.Duration {
	for _, l := range m.Leases {
		if l.MemberID != b.MemberID {
			continue
		}
		if l.DeadlineMono == nil || *l.DeadlineMono <= 0 || m.NowMono <= 0 {
			return 0
		}
		d := *l.DeadlineMono
		if !b.ValidUntil.IsZero() {
			d = m.NowMono + min(d-m.NowMono, b.ValidUntil.Sub(m.Now))
		}
		return d
	}
	return 0
}

// servingDeadline is the whole-node watchdog limit, not the earliest wake.
// Earlier devices expire independently in the kernel. Never derive this from
// a fetched candidate: only the installed model can describe live admission.
func (e *Engine) servingDeadline() time.Duration {
	if !e.serving || e.policyExpires <= 0 || e.appliedModel == nil {
		return 0
	}
	m := *e.appliedModel
	if len(m.Bindings) == 0 {
		return e.policyExpires // intentionally empty policy: retain its node cap
	}
	var last time.Duration
	for _, b := range m.Bindings {
		d := bindingDeadline(m, b)
		if d <= 0 {
			return 0 // malformed retained state must not authorize a heartbeat
		}
		last = max(last, d)
	}
	return last
}

// Only already-expired removals can settle without a whole-node baseline.
// Every survivor keeps its exact binding identity and authorization fields;
// a new generation, addition, rebind or unexpired removal uses normal handoff.
func (e *Engine) onlyExpiredBindingsRemoved(next enforce.Model) bool {
	if !e.serving || e.appliedModel == nil || e.appliedModel.Generation != next.Generation ||
		!sameUsageBindings(e.meterBindings(), e.appliedModel.Bindings) || len(next.Bindings) == 0 ||
		len(next.Bindings) >= len(e.appliedModel.Bindings) {
		return false
	}
	remaining := make(map[uuid.UUID]enforce.Binding, len(next.Bindings))
	for _, b := range next.Bindings {
		remaining[b.DeviceID] = b
	}
	for _, old := range e.appliedModel.Bindings {
		if b, found := remaining[old.DeviceID]; found {
			if !reflect.DeepEqual(old, b) {
				return false
			}
			delete(remaining, old.DeviceID)
		} else {
			d := bindingDeadline(*e.appliedModel, old)
			if d <= 0 || d > e.clock.Mono() {
				return false
			}
		}
	}
	return len(remaining) == 0
}

func (e *Engine) settleExpiredBindings(ctx context.Context) bool {
	if !e.verifyServingModel(ctx, *e.appliedModel, e.appliedAssignments) {
		return false
	}
	// The live rules prove retired elements are past their kernel deadlines.
	// Pause and checkpoint their final deltas before replacing the intent.
	// Surviving counters may advance: their identities never change here.
	if e.meter == nil || e.meter.CheckpointForRestart(ctx) != nil {
		e.usagePersist = true
		e.withdraw(ctx)
		return false
	}
	return true // ordinary Apply resumes sampling; failure paths withdraw
}
