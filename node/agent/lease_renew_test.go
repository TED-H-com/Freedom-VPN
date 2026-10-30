package agent

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestNeedRenewDecision is the renewal decision: a generation change renews
// immediately, the lead still renews, and a lease that is not for the
// applied generation does not forward and is not lease_expired.
func TestNeedRenewDecision(t *testing.T) {
	issued := time.Date(2026, 10, 30, 4, 17, 40, 0, time.UTC)
	window := 24 * time.Hour
	leaseMono := time.Hour

	t.Run("generation bump renews now", func(t *testing.T) {
		e := renewFixture(2, 1, leaseMono, leaseMono, issued, window)
		if !e.needRenew() {
			t.Fatal("applied generation 2 kept the generation 1 lease")
		}
	})

	t.Run("same generation inside the deadline", func(t *testing.T) {
		e := renewFixture(1, 1, leaseMono, leaseMono, issued, window)
		if e.needRenew() {
			t.Fatal("renewed a current lease inside the lead")
		}
		in := withAdmittedMember(servingInput())
		d := Decide(in)
		if !d.Serve || hasReason(d.Reasons, api.ForwardingBlockReasonLeaseExpired) || hasReason(d.Reasons, api.ForwardingBlockReasonPolicyMissing) {
			t.Fatalf("serve %v reasons %v", d.Serve, d.Reasons)
		}
	})

	t.Run("deadline renews", func(t *testing.T) {
		e := renewFixture(1, 1, leaseMono+window-time.Second, leaseMono, issued, window)
		if !e.needRenew() {
			t.Fatal("lead did not renew")
		}
		in := withAdmittedMember(servingInput())
		in.MonoNow = in.LeaseMonoSent + in.Lease.ExpiresAt.Sub(in.Lease.IssuedAt) + time.Second
		d := Decide(in)
		if d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonLeaseExpired) || hasReason(d.Reasons, api.ForwardingBlockReasonPolicyMissing) {
			t.Fatalf("deadline: serve %v reasons %v", d.Serve, d.Reasons)
		}
	})

	t.Run("renew failure", func(t *testing.T) {
		// The renewal did not return a lease for the applied generation.
		// The previous lease is still inside its window.
		in := withAdmittedMember(servingInput())
		ds := *in.Desired
		ds.ConfigGeneration = 2
		in.Desired = &ds
		d := Decide(in)
		if d.Serve {
			t.Fatal("forwarded on a lease for the previous generation")
		}
		if hasReason(d.Reasons, api.ForwardingBlockReasonLeaseExpired) || !hasReason(d.Reasons, api.ForwardingBlockReasonPolicyMissing) {
			t.Fatalf("reasons %v", d.Reasons)
		}
		if d.Safety != api.SafetyStatusPassed {
			t.Fatalf("safety %s", d.Safety)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameLease)
		if !ok || c.Status != api.HealthCheckStatusFailed || codeOf(t, c) != api.ErrorCodeSTALEGENERATION {
			t.Fatalf("lease check %+v", c)
		}
	})
}

func renewFixture(applied, leaseGen uint64, monoNow, leaseMono time.Duration, issued time.Time, window time.Duration) *Engine {
	return &Engine{
		clock:      timeanchor.NewManual(issued, monoNow),
		cfg:        Config{RenewLeadSeconds: 120},
		leaseFresh: true,
		anchorOK:   true,
		leaseMono:  leaseMono,
		appliedGen: applied,
		lease: &api.ServingLease{
			ConfigGeneration: decstr.U64(leaseGen),
			IssuedAt:         issued,
			ExpiresAt:        issued.Add(window),
		},
	}
}

func TestStep_RenewFailureStaysClosed(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	stepOK(t, r)
	if !r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != 1 {
		t.Fatalf("serving %v lease gen %s", r.eng.serving, r.eng.lease.ConfigGeneration)
	}
	before := r.srv.LeaseRenewals()
	r.srv.StickyLease(false)
	r.srv.FailLeases(1)
	r.srv.SetFixture(generationFixture(2, oneBinding(time.Now().UTC())))
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 {
		t.Fatalf("renewals %d before %d", r.srv.LeaseRenewals(), before)
	}
	if r.eng.serving || r.eng.appliedGen != 2 || uint64(r.eng.lease.ConfigGeneration) != 1 {
		t.Fatalf("serving %v applied %d lease %s", r.eng.serving, r.eng.appliedGen, r.eng.lease.ConfigGeneration)
	}
	if hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("reasons %v", r.eng.decision.Reasons)
	}
	beat := mustBeat(t, r)
	if beat.Serving || beat.Reason != string(api.ForwardingBlockReasonPolicyMissing) || beat.Generation != 2 {
		t.Fatalf("heartbeat serving %v reason %s gen %d", beat.Serving, beat.Reason, beat.Generation)
	}
	rep, ok := r.srv.LastObservation()
	if !ok || rep.MemberForwardingEnabled {
		t.Fatal("observation forwarded")
	}
	if len(rep.ForwardingBlockReasons) == 0 || rep.ForwardingBlockReasons[0] != api.ForwardingBlockReasonPolicyMissing {
		t.Fatalf("observation reasons %v", rep.ForwardingBlockReasons)
	}
	for _, reason := range rep.ForwardingBlockReasons {
		if reason == api.ForwardingBlockReasonLeaseExpired {
			t.Fatal("observation reported lease_expired for a generation change")
		}
	}
}

func TestFakecontrol_LeaseFollowsAppliedGeneration(t *testing.T) {
	r := newRig(t)
	r.srv.StickyLease(false)
	enrollBoot(t, r)
	stepOK(t, r)
	if !r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != 1 || len(r.eng.lease.Members) != 0 {
		t.Fatalf("gen1 serving %v lease %s members %d reasons %v", r.eng.serving, r.eng.lease.ConfigGeneration, len(r.eng.lease.Members), r.eng.decision.Reasons)
	}
	first := r.srv.LeaseRenewals()
	if first < 1 {
		t.Fatalf("renewals %d", first)
	}
	member := uuid.MustParse("019a0000-0000-7000-8000-00000000000a")
	binding := oneBinding(time.Now().UTC())
	binding.MemberId = member
	r.srv.SetFixture(generationFixture(2, binding))
	stepOK(t, r)
	if r.srv.LeaseRenewals() <= first {
		t.Fatal("generation change did not renew")
	}
	if !r.eng.serving || r.eng.appliedGen != 2 || uint64(r.eng.lease.ConfigGeneration) != 2 || len(r.eng.lease.Members) == 0 {
		t.Fatalf("serving %v applied %d lease %s members %d reasons %v", r.eng.serving, r.eng.appliedGen, r.eng.lease.ConfigGeneration, len(r.eng.lease.Members), r.eng.decision.Reasons)
	}
	if r.plane.last.Generation != 2 || len(r.plane.last.Bindings) == 0 {
		t.Fatalf("model gen %d bindings %d", r.plane.last.Generation, len(r.plane.last.Bindings))
	}
	beat := mustBeat(t, r)
	if !beat.Serving || beat.Reason != "serving" || beat.Generation != 2 {
		t.Fatalf("heartbeat serving %v reason %s gen %d", beat.Serving, beat.Reason, beat.Generation)
	}
}

func generationFixture(gen uint64, binding api.DeviceBinding) fakecontrol.Fixture {
	return fakecontrol.Fixture{
		ReleaseID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
		Publication: api.PublicationIntentPublished,
		Generation:  gen,
		TTL:         time.Hour,
		Capacity:    &api.NodeCapacity{UpBps: 200_000_000, DownBps: 200_000_000},
		Bindings:    []api.DeviceBinding{binding},
	}
}

func mustBeat(t *testing.T, r *rig) watchdog.Beat {
	t.Helper()
	b, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
