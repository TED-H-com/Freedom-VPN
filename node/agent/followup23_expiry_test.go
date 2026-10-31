package agent

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func TestISOF603_RenewalFailureAndDeadline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	deadline := r.eng.policyExpires
	r.srv.FailLeases(100)
	advanceToRenewal(r)
	before := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || !r.eng.serving || r.eng.policyExpires != deadline {
		t.Fatal("renewal failure shortened a valid lease")
	}
	r.clock.Advance(deadline - r.clock.Mono() - 1500*time.Millisecond)
	stepOK(t, r)
	if !r.eng.serving || len(r.plane.last.Bindings) != 1 {
		t.Fatal("early withdrawal before renderer precision limit")
	}
	// Deterministic read spanning the deadline: neither an intact model nor
	// a busy control path may produce a serving heartbeat after this read.
	r.clock.Advance(1499 * time.Millisecond)
	p := &livePlane{memPlane: r.plane, read: func(context.Context) error { r.clock.Advance(time.Millisecond); return nil }}
	r.eng.plane = p
	if got := r.eng.nextStepDelay(10 * time.Second); got != time.Millisecond {
		t.Fatalf("deadline wake=%s", got)
	}
	r.eng.finishServingStep(t.Context())
	if mustBeat(t, r).Serving || r.eng.serving || len(r.plane.last.Bindings) != 0 {
		t.Fatal("read spanning expiry refreshed serving heartbeat")
	}
	stepOK(t, r)
	if r.eng.serving {
		t.Fatal("failed post-deadline renewal reopened forwarding")
	}
}

// Diagnostic proof, NOT an AT13 pass. This isolates the integer-second
// contract that the enforce owner must change before near-deadline continuity
// can pass. Do not round up: kernel expiry must still survive agent death.
func TestISOF603_WholeSecondRendererLimit(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	deadline := r.eng.policyExpires
	r.clock.Advance(deadline - r.clock.Mono() - 1900*time.Millisecond)
	d := Decide(r.eng.input())
	plan, err := enforce.Render(d.Model, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatal(err)
	}
	for _, seconds := range plan.Timeouts {
		if seconds != 1 {
			t.Fatalf("expected floor reproduction, got %d", seconds)
		}
	}
	if len(plan.Timeouts) != 1 {
		t.Fatal("missing admitted member")
	}
	r.clock.Advance(time.Second)
	d = Decide(r.eng.input())
	if len(d.Model.Bindings) != 0 {
		t.Fatal("integer-second boundary changed; replace diagnostic with continuity proof")
	}
	t.Log("known open precision gap: 1.9s remaining renders 1s; 0.9s remaining omits binding; no transient renewal failure is required")
}
