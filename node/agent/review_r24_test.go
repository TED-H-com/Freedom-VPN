package agent

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick proves Finding H-1
// against main: in Engine.Step (engine.go:619-632), `retain` is gated on
// `ioCtx.Err() != nil` (the D-100 control-plane timeout path). On every normal
// 10-second poll/heartbeat Step where `ioCtx.Err() == nil`, `syncDesired`
// receives HTTP 304 Not Modified (`previousDesired == e.desired`) and
// `syncLease` does not renew (`previousLease == e.lease`), yet `retain` is
// false and `Step` calls `e.enforce(ctx, e.decision)` -> `e.plane.Apply`,
// inflicting a full `tc -batch` + `fv_deadline_guard` drop window on every
// 10-second tick (360 times/hour, 8,640 times/day).
func TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	// Keep the fake signer at this snapshot so later polls are actual 304s.
	signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
	r.srv.SetClock(func() time.Time { return signedAt })

	if !r.eng.serving || r.plane.applies < 1 {
		t.Fatalf("expected serving after serveBoundGeneration, got serving=%v applies=%d", r.eng.serving, r.plane.applies)
	}
	appliesBefore := r.plane.applies
	initialLeaseID := r.eng.lease.LeaseId
	initialGen := r.eng.appliedGen
	initialRenewals := r.srv.LeaseRenewals()

	// Run 3 normal 10-second poll/heartbeat ticks: desired state returns 304
	// Not Modified and the 1-hour lease is nowhere near its 120s renew lead.
	for i := 0; i < 3; i++ {
		r.clock.Advance(10 * time.Second)
		if err := r.eng.Step(context.Background()); err != nil {
			t.Fatalf("tick %d: %v", i+1, err)
		}
	}

	if r.srv.LeaseRenewals() != initialRenewals || r.eng.lease.LeaseId != initialLeaseID || r.eng.appliedGen != initialGen {
		t.Fatalf("unexpected generation or lease change on 304 ticks: gen %d->%d renewals %d->%d",
			initialGen, r.eng.appliedGen, initialRenewals, r.srv.LeaseRenewals())
	}
	if r.plane.applies != appliesBefore {
		// TODO(review r24): H-1 — Engine.Step (engine.go:619) requires `ioCtx.Err() != nil` to set `retain = true`, so every healthy 10-second 304 poll/heartbeat tick calls `e.enforce` -> `e.plane.Apply` and triggers the `fv_deadline_guard` drop window even when nothing changed.
		t.Fatalf("TODO(review r24): H-1: 3 steady-state 10s Steps with HTTP 304 and an unchanged lease executed %d extra Plane.Apply calls (applies %d -> %d, want %d)",
			r.plane.applies-appliesBefore, appliesBefore, r.plane.applies, appliesBefore)
	}
}

// TestR24_H2_MemberRevocationMutatesSiblingQoSRateInDecide proves Finding H-2's
// eligibility contradiction between `docs/poc/G2-followup31-design/candidates.md`
// (lines 38-40, 78-80: any member QoS rate/class change or class deletion
// disqualifies C1 and selects the unconditional guard) and `proof-obligations.md`
// (lines 120-125: AT13 active-stream revocation of an independent member has
// zero guard drops on unchanged member A1): in `buildQoS` (`decide.go:766-812`),
// `fairUp = (upParent - host) / uint64(len(admitted))` and `fairDown =
// (downParent - host) / uint64(len(admitted))` depend on the total count of
// admitted members. Removing member B changes `len(admitted)` from 2 to 1 and
// doubles unchanged member A's HTB `RateBPS`.
func TestR24_H2_MemberRevocationMutatesSiblingQoSRateInDecide(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	b1, b2 := servePair(t, r)
	r.manualUsage(t)

	if len(r.plane.last.QoS.Members) != 2 {
		t.Fatalf("expected 2 admitted members in QoS, got %d", len(r.plane.last.QoS.Members))
	}
	var beforeA enforce.MemberQoS
	for _, mq := range r.plane.last.QoS.Members {
		if mq.MemberID == b1.MemberId {
			beforeA = mq
		}
	}

	// Revoke independent member B (b2) in generation 4 while member A (b1) is unchanged.
	r.srv.StickyLease(false)
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{b1}))
	stepOK(t, r)
	if !r.eng.serving || len(r.plane.last.QoS.Members) != 1 {
		t.Fatalf("expected member A to remain serving alone, got serving=%v qos=%+v", r.eng.serving, r.plane.last.QoS.Members)
	}
	afterA := r.plane.last.QoS.Members[0]
	if afterA.MemberID != b1.MemberId {
		t.Fatalf("unexpected remaining member %s, want %s", afterA.MemberID, b1.MemberId)
	}
	// Proves H-2: revoking independent member B mutates unchanged member A's
	// HTB guaranteed RateBPS (from (parent-host)/2 to (parent-host)/1), which
	// violates C1.1 ("no roots/leaves/classes/rates change; any emitted
	// idempotent class replacements must be byte-equivalent in meaning").
	if beforeA.Up.RateBPS == afterA.Up.RateBPS || afterA.Up.RateBPS != 2*beforeA.Up.RateBPS {
		t.Fatalf("expected buildQoS fair-share recalculation to double A's RateBPS on B's removal, got %d -> %d (member B was %s)",
			beforeA.Up.RateBPS, afterA.Up.RateBPS, b2.MemberId)
	}
}

// TestR24_H3_AppliedModelRetainedAcrossWithdrawAndMultiDeviceDeadlineCollapse
// proves Finding H-3 against main:
//  1. When a member has two devices with different `ValidUntil` timestamps
//     (Device 1 with ValidUntil = T+1h visited first, Device 2 with
//     ValidUntil = T+5m visited second), `Decide` (`decide.go:682-695`)
//     collapses both devices into a single per-member `MemberLease` whose
//     `DeadlineMono` is tightened to T+5m (so `enforce.Render` gives Device 1
//     a 5m kernel element timeout instead of 1h!) while leaving
//     `MemberLease.EffectiveServiceEnd` and `NodeLeaseEnd` at Device 1's T+1h.
//  2. When the node withdraws to Baseline (generation 0), `e.appliedModel` in
//     `Engine` remains non-nil and retains the previous serving model.
func TestR24_H3_AppliedModelRetainedAcrossWithdrawAndMultiDeviceDeadlineCollapse(t *testing.T) {
	t.Run("multi_device_same_member_collapses_deadline_mono_and_leaves_service_end_inconsistent", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		now := r.clock.Wall()
		memberID := uuid.MustParse("019a0000-0000-7000-8000-0000000000aa")
		dev1ID := uuid.MustParse("019a0000-0000-7000-8000-0000000000d1")
		dev2ID := uuid.MustParse("019a0000-0000-7000-8000-0000000000d2")
		bindings := []api.DeviceBinding{
			{
				MemberId:        memberID,
				DeviceId:        dev1ID,
				PeerId:          "peer-d1",
				AllowedIpv4:     []string{"100.64.9.1/32"},
				BindingRevision: decstr.U64(1),
				ValidFrom:       now.Add(-time.Minute),
				ValidUntil:      now.Add(time.Hour),
			},
			{
				MemberId:        memberID,
				DeviceId:        dev2ID,
				PeerId:          "peer-d2",
				AllowedIpv4:     []string{"100.64.9.2/32"},
				BindingRevision: decstr.U64(1),
				ValidFrom:       now.Add(-time.Minute),
				ValidUntil:      now.Add(5 * time.Minute),
			},
		}
		r.srv.StickyLease(false)
		r.srv.SetFixture(fixtureWithBindings(3, bindings))
		if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
			t.Fatal(err)
		}
		if err := r.eng.Boot(context.Background()); err != nil {
			t.Fatal(err)
		}
		stepOK(t, r)
		r.manualUsage(t)
		if !r.eng.serving || len(r.plane.last.Leases) != 1 {
			t.Fatalf("expected 1 MemberLease for shared member, got serving=%v leases=%+v", r.eng.serving, r.plane.last.Leases)
		}
		plan, err := enforce.Render(r.plane.last, enforce.State{}, enforce.TCState{})
		if err != nil {
			t.Fatal(err)
		}
		t1 := plan.ElementTimeouts[dev1ID.String()]
		t2 := plan.ElementTimeouts[dev2ID.String()]
		if t1 <= 5*time.Minute {
			// TODO(review r24): H-3 — decide.go:682-695 stores a single DeadlineMono per MemberID (not per DeviceID), so Device 2's 5m ValidUntil overwrites Device 1's 1h ValidUntil in MemberLease.DeadlineMono while leaving MemberLease.EffectiveServiceEnd at 1h.
			t.Fatalf("TODO(review r24): H-3: Device 1 (ValidUntil=1h) received kernel element timeout %s (same as Device 2's %s) because decide.go:682-695 collapses DeadlineMono per MemberID while leaving EffectiveServiceEnd=%s",
				t1, t2, r.plane.last.Leases[0].EffectiveServiceEnd.Sub(r.plane.last.Leases[0].SignedServerTime))
		}
	})

	t.Run("appliedModel_remains_non_nil_after_withdraw_to_baseline", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		if !r.eng.serving || r.eng.appliedModel == nil {
			t.Fatal("expected serving with non-nil appliedModel")
		}
		// Unpublish the node so Step withdraws to baseline.
		r.srv.SetPublication(api.PublicationIntentUnpublished)
		stepOK(t, r)
		if r.eng.serving {
			t.Fatal("expected node to withdraw to baseline when unpublished")
		}
		// Proves H-3's architectural warning for C1: e.appliedModel is intentionally
		// retained across withdraw for foreign-tc containment auditing, so C1 cannot
		// treat `e.appliedModel != nil` as proof that the kernel is currently serving
		// that model.
		if r.eng.appliedModel == nil {
			t.Fatal("expected e.appliedModel to remain populated after withdraw")
		}
	})
}

// TestR24_Held_Q1_MonotonicAnchorAndSameBootAdoption proves What Held on main
// for Question 1 (G1 & G9):
//   - Decide anchors MemberLease.DeadlineMono to in.LeaseMonoSent + window and
//     fails closed with LEASE_EXPIRED ("monotonic clock is before lease
//     mono_sent") if MonoNow < LeaseMonoSent;
//   - Boot adopts an already-verified serving ruleset on the same boot without
//     calling Plane.Apply.
func TestR24_Held_Q1_MonotonicAnchorAndSameBootAdoption(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		closeAdoptionControl(t, r)

		in := r.eng.input()
		in.ProbeRan = true
		in.ProbeCheck = api.HealthCheckStatusPassed
		in.ProbeHealth = api.DataHealthHealthy
		validDec := Decide(in)
		if !validDec.Serve || len(validDec.Model.Leases) != 1 || validDec.Model.Leases[0].DeadlineMono == nil {
			t.Fatalf("expected valid Decide to serve with anchored DeadlineMono, got serve=%v reasons=%v errors=%+v", validDec.Serve, validDec.Reasons, validDec.Errors)
		}
		window := in.Lease.ExpiresAt.Sub(in.Lease.IssuedAt)
		if desWindow := in.Desired.ExpiresAt.Sub(in.Lease.IssuedAt); desWindow < window {
			window = desWindow
		}
		if maxOff := time.Duration(in.Lease.MaxOfflineSeconds) * time.Second; maxOff < window {
			window = maxOff
		}
		for _, ml := range in.Lease.Members {
			if lWindow := ml.LeaseEnd.Sub(in.Lease.IssuedAt); lWindow < window {
				window = lWindow
			}
			if mWindow := ml.EffectiveServiceEnd.Sub(in.Lease.IssuedAt); mWindow < window {
				window = mWindow
			}
		}
		for _, db := range in.Desired.DeviceBindings {
			if dWindow := db.ValidUntil.Sub(in.Lease.IssuedAt); dWindow < window {
				window = dWindow
			}
		}
		if want := in.LeaseMonoSent + window; *validDec.Model.Leases[0].DeadlineMono != want {
			t.Fatalf("DeadlineMono=%s want=%s", *validDec.Model.Leases[0].DeadlineMono, want)
		}

		in.MonoNow = in.LeaseMonoSent - time.Millisecond
		dec := Decide(in)
		if dec.Serve || !hasReason(dec.Reasons, api.ForwardingBlockReasonLeaseExpired) {
			t.Fatalf("expected ForwardingBlockReasonLeaseExpired when MonoNow < LeaseMonoSent, got serve=%v reasons=%v", dec.Serve, dec.Reasons)
		}

		// Same-boot restart adopts without calling Plane.Apply.
		appliesBefore := r.plane.applies
		if err := r.eng.Close(); err != nil {
			t.Fatal(err)
		}
		next, err := Start(Options{
			Config:      r.eng.cfg,
			Clock:       r.clock,
			Plane:       r.plane,
			MeterSource: staticSource{boot: r.eng.bootID.String()},
			Prober:      passProbe{},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = next.Close() }()
		if err := next.Boot(context.Background()); err != nil {
			t.Fatal(err)
		}
		if !next.serving || r.plane.applies != appliesBefore {
			t.Fatalf("same-boot Boot did not adopt without Apply: serving=%v applies=%d->%d",
				next.serving, appliesBefore, r.plane.applies)
		}
	})
}
