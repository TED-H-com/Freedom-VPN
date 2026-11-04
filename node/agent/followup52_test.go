package agent

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func f52Pending(t *testing.T, r *rig) []agentstore.OutboxItem {
	t.Helper()
	items, err := r.eng.store.Pending(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func f52NoCapacityGate(t *testing.T, r *rig) {
	t.Helper()
	if r.eng.outboxFull || slices.Contains(r.eng.decision.Reasons, api.ForwardingBlockReasonOutboxFull) || len(f52Pending(t, r)) >= 32 {
		t.Fatal("unchanged outage filled observation outbox")
	}
}

// TEST ONLY: actual control Steps, with the fake issuer following monotonic time.
// Only the observations endpoint fails; fresh leases keep the closed model valid.
func TestD157_F52_IsolationLatchStepsOutlast32Intervals(t *testing.T) {
	r, base := f50Rig(t)
	p := &r31IsolationQuarantinePlane{r27BootMissingInterfacePlane: base, failIsolationQuarantine: true}
	r.eng.plane = p
	r.eng.cfg.ObservationSeconds = 30
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	if r.eng.containment == nil || r.eng.containment.Code != api.ErrorCodeISOLATIONPOLICYINVALID {
		t.Fatal("missing isolation latch")
	}
	p.failIsolationQuarantine = false
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetObservationRetryAfter(1000, 60)
	advanceToRenewal(r)
	stepOK(t, r)
	state, pending := r.eng.lastObservationState, f52Pending(t, r)
	if !r.eng.closedRecoveryHeld || r.eng.closedRecoveryRenewal.reportID != uuid.Nil || len(pending) != 1 {
		t.Fatal("missing unbound closed recovery")
	}
	for range 35 {
		r.clock.Advance(30 * time.Second)
		stepOK(t, r)
		f52NoCapacityGate(t, r)
		got := f52Pending(t, r)
		if !r.eng.closedRecoveryHeld || r.eng.serving || r.eng.closedRecoveryRenewal.reportID != uuid.Nil || r.eng.lastObservationState != state || len(got) != 1 || got[0].ReportID != pending[0].ReportID {
			t.Fatalf("held state changed or duplicated: held=%v renewal=%+v pending=%d", r.eng.closedRecoveryHeld, r.eng.closedRecoveryRenewal, len(got))
		}
	}
	// A new generation queues its transition even while the historical error
	// still prohibits binding. A subsequent real foreign-tc rejection replaces
	// that historical code; clean closed Apply then binds its own fresh report.
	r.srv.SetFixture(generationFixture(15, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	if len(f52Pending(t, r)) <= len(pending) {
		t.Fatal("generation transition suppressed")
	}
	p.foreignTC = true
	stepOK(t, r)
	p.foreignTC = false
	stepOK(t, r)
	id := r.eng.closedRecoveryRenewal.reportID
	if id == uuid.Nil || r.eng.closedRecoveryRenewal.accepted || !r.eng.closedRecoveryHeld || r.eng.serving {
		t.Fatal("clean recovery not bound before acceptance")
	}
	found := false
	for _, item := range f52Pending(t, r) {
		if item.ReportID == id {
			found = true
		}
	}
	if !found {
		t.Fatal("bound report not queued")
	}
	r.srv.SetObservationRetryAfter(0, 0)
	r.clock.Advance(max(0, r.eng.observationRetryAt-r.clock.Mono()))
	stepOK(t, r)
	if !r.eng.closedRecoveryRenewal.accepted {
		t.Fatal("own report acceptance did not arm recovery")
	}
}

// A stale closed recovery is reachable, but retention refuses it next Step.
// Exercise both desired-with-bindings and the stale, empty-desired variant.
func TestD157_F52_StaleLeaseStepsOutlast32Intervals(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "bindings", true: "empty_desired"}[empty], func(t *testing.T) {
			r, p := f50Rig(t)
			p.foreignTC = true
			stepOK(t, r)
			p.foreignTC = false
			fixture := generationFixture(14, oneBinding(r.clock.Wall()))
			if empty {
				fixture.Bindings = nil
			}
			r.srv.SetFixture(fixture)
			r.srv.FailLeases(1000)
			r.srv.SetObservationRetryAfter(1000, 60)
			stepOK(t, r)
			if !r.eng.closedRecoveryHeld || r.eng.closedRecoveryRenewal.reportID != uuid.Nil || !r.eng.leaseBehindApplied() {
				t.Fatal("missing transient stale recovery")
			}
			r.clock.Advance(time.Second)
			stepOK(t, r)
			if r.eng.closedRecoveryHeld || r.eng.closedRecoveryRenewal.armed || r.eng.serving {
				t.Fatal("stale recovery retained across enforcement")
			}
			pending, state := f52Pending(t, r), r.eng.lastObservationState
			for range 35 {
				// Keep this at 1s TEST ONLY to stay inside the original lease lifetime;
				// a real expiry is a separate semantic transition, not a duplicate.
				r.eng.cfg.ObservationSeconds = 1
				r.clock.Advance(time.Second)
				stepOK(t, r)
				f52NoCapacityGate(t, r)
				if r.eng.lastObservationState != state || len(f52Pending(t, r)) != len(pending) {
					t.Fatal("stale unchanged state duplicated")
				}
			}
			r.srv.FailLeases(0)
			r.srv.SetMemberGrantsReady(false)
			r.clock.Advance(time.Second)
			before := len(f52Pending(t, r))
			stepOK(t, r)
			if len(f52Pending(t, r)) <= before {
				t.Fatal("catch-up transition suppressed")
			}
			if r.eng.closedRecoveryRenewal.armed {
				t.Fatal("baseline recreated an early opportunity")
			}
		})
	}
}

func TestD157_F52_CatchupBindsWaitingRecovery(t *testing.T) {
	r, p := f50Rig(t)
	p.foreignTC = true
	stepOK(t, r)
	p.foreignTC = false
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	r.srv.FailLeases(1)
	r.srv.SetObservationRetryAfter(1000, 60)
	stepOK(t, r)
	if !r.eng.closedRecoveryHeld || r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
		t.Fatal("missing stale unbound recovery")
	}
	before := len(f52Pending(t, r))
	r.srv.SetMemberGrantsReady(false)
	r.clock.Advance(time.Second)
	stepOK(t, r)
	id := r.eng.closedRecoveryRenewal.reportID
	if id == uuid.Nil || r.eng.serving || !r.eng.closedRecoveryHeld || len(f52Pending(t, r)) != before+1 {
		t.Fatal("matching empty lease did not immediately bind transition")
	}
	var rep api.ObservationReport
	for _, item := range f52Pending(t, r) {
		if item.ReportID == id {
			if err := json.Unmarshal(item.Payload, &rep); err != nil {
				t.Fatal(err)
			}
		}
	}
	if rep.ReportId != id || !slices.Contains(rep.ForwardingBlockReasons, api.ForwardingBlockReasonAwaitingMemberGrants) {
		t.Fatal("wrong recovery body bound")
	}
}

// TEST ONLY: a verified closed Apply can return failed revoke addresses. Its
// report cannot bind, and the next full enforcement disarms through baseline.
type f52PendingRevokePlane struct {
	*r27BootMissingInterfacePlane
	pending bool
}

func (p *f52PendingRevokePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.r27BootMissingInterfacePlane.Apply(ctx, m, prev)
	if err == nil && p.pending && len(m.Revoke) > 0 {
		return res, &enforce.ConntrackDeleteError{Failed: m.Revoke, Pending: len(m.Revoke)}
	}
	return res, err
}
func TestD157_F52_PendingRevokeStepsOutlast32Intervals(t *testing.T) {
	r, base := f50Rig(t)
	p := &f52PendingRevokePlane{r27BootMissingInterfacePlane: base, pending: true}
	r.eng.plane = p
	base.foreignTC = true
	stepOK(t, r)
	base.foreignTC = false
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	r.srv.SetObservationRetryAfter(1000, 60)
	stepOK(t, r)
	if !r.eng.closedRecoveryHeld || len(r.eng.pendingRevoke) == 0 || r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
		t.Fatal("missing transient pending-revoke recovery")
	}
	stepOK(t, r)
	if r.eng.closedRecoveryHeld || r.eng.closedRecoveryRenewal.armed {
		t.Fatal("pending revoke retained recovery")
	}
	before, state := len(f52Pending(t, r)), r.eng.lastObservationState
	r.eng.cfg.ObservationSeconds = 1
	for range 35 {
		r.clock.Advance(time.Second)
		stepOK(t, r)
		f52NoCapacityGate(t, r)
		if r.eng.lastObservationState != state || len(f52Pending(t, r)) != before {
			t.Fatal("pending-revoke unchanged state duplicated")
		}
	}
}
