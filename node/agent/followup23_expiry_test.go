package agent

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
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

// D-104 keeps sub-second authorization; Apply separately reserves its bounded
// guarded-install interval. Render's deprecated whole-second projection is not
// evidence of the installed lifetime.
func TestD104_SubsecondDeadline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	deadline := r.eng.policyExpires
	for _, remaining := range []time.Duration{1900 * time.Millisecond, 900 * time.Millisecond, time.Millisecond, time.Microsecond} {
		r.clock.Advance(deadline - r.clock.Mono() - remaining)
		d := Decide(r.eng.input())
		if len(d.Model.Bindings) != 1 || len(d.Model.Leases) != 1 || d.Model.NowMono != r.clock.Mono() {
			t.Fatalf("lost positive allowance %s: %+v", remaining, d)
		}
		lease := d.Model.Leases[0]
		if lease.DeadlineMono == nil || *lease.DeadlineMono != deadline || lease.TimeoutCeilSeconds != 0 {
			t.Fatalf("deadline lost: %+v", lease)
		}
		plan, err := enforce.Render(d.Model, enforce.State{}, enforce.TCState{})
		if err != nil {
			t.Fatal(err)
		}
		want := remaining.Truncate(enforce.DeadlineGranularity)
		if want == 0 {
			if len(plan.ElementTimeouts) != 0 {
				t.Fatal("sub-quantum admitted")
			}
			continue
		}
		if len(plan.ElementTimeouts) != 1 {
			t.Fatal("missing element")
		}
		for _, got := range plan.ElementTimeouts {
			if got != want {
				t.Fatalf("remaining=%s rendered=%s", remaining, got)
			}
		}
	}
	r.clock.Advance(time.Microsecond)
	if d := Decide(r.eng.input()); d.Serve || len(d.Model.Bindings) != 0 {
		t.Fatal("deadline admitted")
	}
}

func TestD104_DeadlineMinimum(t *testing.T) {
	for _, limit := range []string{"node", "policy", "offline", "member", "service", "binding"} {
		t.Run(limit, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			in := r.eng.input()
			desired, lease := *in.Desired, *in.Lease
			desired.DeviceBindings = append([]api.DeviceBinding(nil), desired.DeviceBindings...)
			lease.Members = append([]api.MemberLease(nil), lease.Members...)
			in.Desired, in.Lease = &desired, &lease
			end := lease.IssuedAt.Add(2500 * time.Millisecond)
			want := in.LeaseMonoSent + 2500*time.Millisecond
			switch limit {
			case "node":
				lease.ExpiresAt = end
			case "policy":
				desired.ExpiresAt = end
			case "offline":
				lease.MaxOfflineSeconds = 2
				want = in.LeaseMonoSent + 2*time.Second
			case "member":
				lease.Members[0].LeaseEnd = end
			case "service":
				lease.Members[0].EffectiveServiceEnd = end
			case "binding":
				desired.DeviceBindings[0].ValidUntil = end
			}
			model, expiry, err := buildModel(in)
			if err != nil || len(model.Leases) != 1 || model.Leases[0].DeadlineMono == nil || *model.Leases[0].DeadlineMono != want || expiry != want {
				t.Fatalf("minimum lost: model=%+v expiry=%s want=%s err=%v", model, expiry, want, err)
			}
		})
	}
}
