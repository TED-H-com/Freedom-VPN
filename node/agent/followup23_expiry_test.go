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
	// Isolate the lease deadline: reissued desired snapshots must not be the
	// earlier, independently renewable policy bound in this failure test.
	binding := oneBinding(r.clock.Wall())
	binding.ValidUntil = r.clock.Wall().Add(2 * time.Hour)
	fixture := generationFixture(2, binding)
	fixture.TTL = 2 * time.Hour
	r.srv.SetFixture(fixture)
	stepOK(t, r)
	deadline := r.eng.policyExpires
	r.srv.FailLeases(100)
	advanceToRenewal(r)
	before := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || !r.eng.serving || r.eng.policyExpires != deadline {
		t.Fatalf("renewal failure changed deadline: renewals=%d want=%d serving=%t before=%s after=%s errors=%+v", r.srv.LeaseRenewals(), before+1, r.eng.serving, deadline, r.eng.policyExpires, r.eng.decision.Errors)
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

func TestD104_MemberDeadlinesAreIndependentAndBindingOrderInvariant(t *testing.T) {
	in := withAdmittedMember(servingInput())
	bindings := threeBindings(in.Lease.IssuedAt)
	// Devices 0 and 1 share a member; its shortest binding must win in both
	// orders, without changing the other member's distinct absolute pointer.
	bindings[1].MemberId = bindings[0].MemberId
	bindings[0].ValidUntil = in.Lease.IssuedAt.Add(2500 * time.Millisecond)
	bindings[1].ValidUntil = in.Lease.IssuedAt.Add(7500 * time.Millisecond)
	bindings[2].ValidUntil = in.Lease.IssuedAt.Add(9500 * time.Millisecond)
	in.Desired.DeviceBindings = bindings
	in.Lease.Members = []api.MemberLease{oneMember(in.Lease.IssuedAt, bindings[0].MemberId), oneMember(in.Lease.IssuedAt, bindings[2].MemberId)}
	for range 2 {
		m, expiry, err := buildModel(in)
		if err != nil || len(m.Leases) != 2 {
			t.Fatal("member deadlines", m, err)
		}
		if m.Leases[0].DeadlineMono == m.Leases[1].DeadlineMono {
			t.Fatal("aliased deadlines")
		}
		for _, lease := range m.Leases {
			want := in.LeaseMonoSent + 2500*time.Millisecond
			if lease.MemberID == bindings[2].MemberId {
				want = in.LeaseMonoSent + 9500*time.Millisecond
			}
			if *lease.DeadlineMono != want {
				t.Fatal("wrong member minimum", *lease.DeadlineMono, want)
			}
		}
		if expiry != in.LeaseMonoSent+2500*time.Millisecond {
			t.Fatal("wrong wake deadline")
		}
		bindings[0], bindings[1] = bindings[1], bindings[0]
	}
}
