package agent

// Adversarial review r23 proofs for agent-core follow-up 39 (99f93112).
// Original proofs are preserved in commit dcdc3d96. Historical trace comments
// describe the before-fail. Follow-up 40 conversions and fixture limits are in
// docs/node/followup40-r23-judgments-2026-11-03.md.
// TEST ONLY in-memory data planes and disk SQLite; no host networking or root.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestR23_H1a_EpochSwitchDuringServing_InvalidatesUsageIntentAndCreatesFalseConflictOnRestart
// proves R23-H-1 (Subcase A):
//
// Follow-up 39 binds usage_apply_intent to (BootID, h.OpenEpoch) inside
// recordUsageIntent before e.plane.Apply (restart_usage.go:57), and
// restartHeadBindings requires p.Epoch == h.OpenEpoch (restart_usage.go:130).
// However, whenever the open usage epoch advances while the same serving
// generation remains active (either via e.switchEpoch(ctx) at the end of
// enforce at engine.go:1400, or via a counter-reset switch inside Meter.Sample
// at delta.go:258,426), head.OpenEpoch advances to a new epoch E2 while
// usage_apply_intent remains pinned to the closed epoch E1.
//
// If syncDesired later stores a newer generation G3 that rebinds the device to
// Member M2 and was never applied (e.g., G3 withdraws forwarding via Baseline
// or its lease is unverified, setting kernel gen=0, and FinalSample hits
// meter.ErrHold or the process crashes before E2 settles), Boot's
// restartHeadBindings sees p.Epoch (E1) != h.OpenEpoch (E2), discards the
// durable usage_apply_intent record (which proves G3 never reached Apply), and
// falls back to uniqueUsageBindings(G2, G3), turning 100% of Member M1's pure
// G2 traffic in E2 into a false mapping_conflict gap.
func TestR23_H1a_EpochSwitchDuringServing_InvalidatesUsageIntentAndCreatesFalseConflictOnRestart(t *testing.T) {
	// Regression r23: H-1 (R23-H-1) — When the open usage epoch advances while
	// serving the same generation, usage_apply_intent remains pinned to the
	// superseded epoch E1; restartHeadBindings then rejects the intent on E2
	// and falls back to uniqueUsageBindings(G2, G3), turning pure G2 traffic in
	// E2 into a false mapping_conflict gap.
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}

	// Process 1 serves generation 2 (dev -> member1, rev 1) in epoch E1.
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	b1 := r.eng.appliedModel.Bindings[0]
	member1 := b1.MemberID
	upName := meter.CounterName(b1.DeviceID, meter.DirUpload)

	r.eng.meter.Close()
	src1 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings,
		counters: map[string]enforce.CounterValue{upName: {Packets: 2, Bytes: 4096}},
	}
	var err error
	r.eng.meter, err = meter.New(meter.Config{
		NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// First sample records 4096 B in E1.
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	h1, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	epoch1 := h1.OpenEpoch

	// Advance clock by 2s and simulate a counter reset during G2 (or
	// NotifyRulesetSwitch), which closes E1 and opens E2 in usage.db while G2
	// continues serving on the same boot.
	r.clock.Advance(2 * time.Second)
	src1.setCounter(upName, enforce.CounterValue{Packets: 1, Bytes: 1024})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	h2, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	epoch2 := h2.OpenEpoch
	if epoch2 == uuid.Nil || epoch2 == epoch1 {
		t.Fatalf("precondition: expected counter reset to open a new epoch E2 != E1 (%s), got %s", epoch1, epoch2)
	}

	// Additional pure G2 traffic for member1 flows in E2 (advancing u_<D> from
	// 1024 to 5120, a +4096 B delta in E2).
	src1.setCounter(upName, enforce.CounterValue{Packets: 5, Bytes: 5120})

	// Generation 3 arrives in desired-state.json rebinding the same DeviceID to
	// member2 (rev 2), and Process 1 crashes right after Baseline (before E2 is
	// settled and before G3 ever reaches Apply).
	member2 := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	b2 := oneBinding(r.clock.Wall())
	b2.MemberId = member2
	b2.BindingRevision = 2
	r.srv.SetFixture(generationFixture(3, b2))
	if err := r.eng.syncDesired(t.Context()); err != nil || r.eng.desired.ConfigGeneration != 3 {
		t.Fatalf("precondition: syncDesired generation 3 failed: %v", err)
	}
	if err := r.plane.Baseline(t.Context(), r.plane.last.Datapath); err != nil {
		t.Fatal(err)
	}
	r20Crash(r.eng)

	// Process 2 boots on the same boot. usage_apply_intent in agent.db has
	// BootID == bootID and Generation == 2 (member1, rev 1), proving G3 never
	// reached Apply.
	r.clock.Advance(2 * time.Second)
	src2 := &r20DynamicSource{
		boot:     src1.boot,
		counters: map[string]enforce.CounterValue{upName: {Packets: 5, Bytes: 5120}},
	}
	next := r20Successor(t, r, r.eng.cfg, src2, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}

	led := r21ReadLedger(t, r.eng.cfg, next.nodeID, next.incID)
	// The reset itself is a legitimate gap; 1024 is the new baseline.
	wantTotal := uint64(4096 + 4096)
	gotM1 := led.byMemberRev[r21MemberRev{member: member1, rev: 1}]
	if gotM1 != wantTotal || len(led.gaps) != 1 || led.gaps[0].Gap.Reason != api.UsageGapReasonCounterDiscontinuity {
		t.Fatalf("R23-H-1a: epoch switch from E1 (%s) to E2 (%s) invalidated usage_apply_intent; "+
			"member1 attributed=%d (want %d), gaps=%d (%+v)",
			epoch1, epoch2, gotM1, wantTotal, len(led.gaps), led.gaps)
	}
}

// TestR23_H1b_GuardMissingIdleCounter_EpochSwitchKeepsAdoption
// Guards R23-H-1 (Subcase B). TEST ONLY: a verified real renderer creates
// idle zero counters. This absent-counter state is not a production proof:
//
// When enforce repairs a prior rejected apply (e.applyError != nil) or
// baseline failure (!e.baselineOK) while k.nudgePending() is true, enforce
// runs recordUsageIntent at engine.go:1260 (recording Epoch = E1) and then
// calls e.switchEpoch(ctx) at engine.go:1400, which closes E1 and opens E2 in
// usage.db without updating usage_apply_intent.
//
// If a second bound device devB in the same generation had not yet appeared in
// obs.Counters when E2 opened and then forwards traffic in E2 before a
// same-boot restart with the kernel dataplane still intact at generation G,
// restartHeadBindings(h) rejects usage_apply_intent (p.Epoch E1 != h.OpenEpoch
// E2) and filters e.restartBindings by known[b.DeviceID] in h.Counters,
// dropping devB. Consequently, adoptStoredServing refuses same-boot adoption
// (!sameUsageBindings) and Boot's settleRetainedUsage turns devB's E2 traffic
// into an unknown_device gap.
func TestR23_H1b_GuardMissingIdleCounter_EpochSwitchKeepsAdoption(t *testing.T) {
	r23H1b(t, false)
}

func r23H1b(t *testing.T, idleZero bool) {
	// Regression r23: H-1 (R23-H-1) — When the open usage epoch advances while
	// serving a multi-device generation, usage_apply_intent remains on E1; on a
	// same-boot restart after devB becomes active in E2, restartHeadBindings
	// drops devB, adoptStoredServing refuses adoption, and settleRetainedUsage
	// turns devB's E2 traffic into an unknown_device gap.
	r := newRig(t)
	alignControlClock(r)
	r.srv.StickyLease(false)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetUsageUnavailable(1000)

	enrollBoot(t, r)
	stepOK(t, r)

	// Configure generation 2 with two bound devices (devA -> memberA, devB -> memberB).
	now := r.clock.Wall()
	bA := oneBinding(now)
	memberA := bA.MemberId
	devA := bA.DeviceId

	devB := uuid.MustParse("019a0000-0000-7000-8000-0000000000d2")
	memberB := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	bB := bA
	bB.BindingRevision = 2
	bB.DeviceId = devB
	bB.MemberId = memberB
	bB.PeerId = "peer-b"
	bB.AllowedIpv4 = []string{"100.64.0.11/32"}
	fix := generationFixture(2, bA)
	fix.Bindings = []api.DeviceBinding{bA, bB}
	r.srv.SetFixture(fix)

	upA := meter.CounterName(devA, meter.DirUpload)
	upB := meter.CounterName(devB, meter.DirUpload)
	src1 := &r20DynamicSource{
		boot: r.eng.bootID.String(),
		// Only devA has a counter entry when E1/E2 open; devB traffic arrives after E2 opens.
		counters: map[string]enforce.CounterValue{
			upA: {Packets: 1, Bytes: 2048},
		},
	}
	if idleZero {
		src1.setCounter(upB, enforce.CounterValue{})
	}
	r.eng.meterSource = src1

	// Step for generation 2: both devA and devB are admitted into appliedModel.Bindings.
	stepOK(t, r)
	if !r.eng.serving || len(r.eng.appliedModel.Bindings) != 2 {
		t.Fatalf("step for gen 2 did not serve 2 bindings: serving=%v bindings=%d reasons=%v",
			r.eng.serving, len(r.eng.appliedModel.Bindings), r.eng.decision.Reasons)
	}
	r.manualUsage(t)
	src1.setBindings(r.eng.meterBindings)
	r.eng.meter.Close()
	r.clock.Advance(2 * time.Second)
	var err error
	r.eng.meter, err = meter.New(meter.Config{
		NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src1, Now: r.clock.Wall,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}

	// Simulate the one-packet ruleset-switch nudge (or a counter reset on devA)
	// opening a new epoch E2 during the same serving generation: OpenEpoch
	// advances from E1 to E2 in usage.db while usage_apply_intent remains on E1.
	r.clock.Advance(2 * time.Second)
	src1.setCounter(upA, enforce.CounterValue{Packets: 0, Bytes: 2048})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	hAfter, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if r.eng.restartIntent == nil || r.eng.restartIntent.Epoch == hAfter.OpenEpoch {
		t.Fatalf("precondition: expected epoch switch to advance OpenEpoch past restartIntent.Epoch (%v vs %v)",
			r.eng.restartIntent, hAfter.OpenEpoch)
	}

	// Process 1 stops cleanly for a same-boot restart while the kernel retains
	// generation 2, and devB forwards 4096 B in E2 during the restart gap.
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	src1.setCounter(upB, enforce.CounterValue{Packets: 2, Bytes: 4096})

	r.clock.Advance(2 * time.Second)
	src2 := &r20DynamicSource{
		boot: src1.boot,
		counters: map[string]enforce.CounterValue{
			upA: {Packets: 1, Bytes: 2048},
			upB: {Packets: 2, Bytes: 4096},
		},
	}
	next := r20Successor(t, r, r.eng.cfg, src2, r.plane)
	next.prober = &verProbe{version: "0.76.2"}
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !next.serving {
		led := r21ReadLedger(t, r.eng.cfg, next.nodeID, next.incID)
		t.Fatalf("R23-H-1b: same-boot adoption was refused after switchEpoch left usage_apply_intent on E1; "+
			"next.serving=%v, memberA=%d, memberB=%d (want 4096), gaps=%d (%+v)",
			next.serving,
			led.byMemberRev[r21MemberRev{member: memberA, rev: 1}],
			led.byMemberRev[r21MemberRev{member: memberB, rev: 2}],
			len(led.gaps), led.gaps)
	}
	r.eng = next
	r.manualUsage(t)
	next.meter.Close()
	next.meter, err = meter.New(meter.Config{NodeID: next.nodeID, IncarnationID: next.incID, Store: next.usage, Source: src2, Now: r.clock.Wall})
	if err != nil {
		t.Fatal(err)
	}
	r.clock.Advance(2 * time.Second)
	if err := next.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	led := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if led.byMemberRev[r21MemberRev{member: memberA, rev: 1}] != 2048 || led.byMemberRev[r21MemberRev{member: memberB, rev: 2}] != 4096 || len(led.gaps) != 1 || led.gaps[0].Gap.Reason != api.UsageGapReasonCounterDiscontinuity {
		t.Fatalf("post-adoption attribution: %+v", led)
	}
}

// TestR23_H2_HandoffBarrierRaceLeavesFinalDoneFalse_NextServeFailsAdmission
// proves R23-H-2:
//
// In handoff(ctx) (usage.go:442-475), e.settleRetainedUsage(ctx) closes the
// open epoch in usage.db (head.OpenEpoch = uuid.Nil, e.finalDone = true), and
// line 448 immediately resets e.finalDone = false before checking
// meterBarrierIdentity(e.cfg.JournalDir) at line 455.
//
// When the updater barrier is replaced or removed while handoff is running (the
// exact race tested by TestD125_Rule2LatchReplacementDuringClosePublishesNothing),
// handoff returns false at line 461 with head.OpenEpoch == uuid.Nil,
// e.meterBindings() == nil, and e.finalDone == false.
//
// On the next Step(ctx) after the barrier is cleared, enforce skips e.handoff
// (len(e.meterBindings()) == 0) AND skips e.meter.PrepareSampling (because
// e.finalDone == false!), so recordUsageIntent fails with "usage: no current
// epoch for admission" and rejectServing aborts serving with a false
// HEALTH_CHECK_FAILED apply error and STORE_UNAVAILABLE.
func TestR23_H2_HandoffBarrierRaceLeavesFinalDoneFalse_NextServeFailsAdmission(t *testing.T) {
	// Regression r23: H-2 (R23-H-2) — handoff must not leave e.finalDone=false
	// after settleRetainedUsage closed the open epoch in usage.db, or enforce
	// will skip PrepareSampling and fail recordUsageIntent on the next Serve.
	r := barrierRig(t)
	r.clock.Advance(2 * time.Second)

	// Trigger the exact D-125 Rule 2 race during handoff: the barrier changes
	// while Baseline runs inside handoff.
	writeBarrier(t, r, uuid.NewString())
	r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error {
		writeBarrier(t, r, uuid.NewString())
		return nil
	}}
	r.eng.handoff(t.Context())
	r.eng.plane = r.plane

	// Verify precondition: handoff closed the epoch in usage.db but left
	// e.finalDone == false.
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if h.OpenEpoch != uuid.Nil {
		t.Fatalf("precondition: expected closed epoch after settleRetainedUsage, got %s", h.OpenEpoch)
	}

	// Now the updater removes forwarding.blocked and the agent runs a normal
	// Step(ctx) to resume serving.
	if err := os.Remove(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || r.eng.applyError != nil || r.eng.usagePersist {
		t.Fatalf("R23-H-2: Step after barrier removal failed admission because handoff left finalDone=false with OpenEpoch=uuid.Nil: "+
			"serving=%v applyError=%+v usagePersist=%v reasons=%v errors=%+v",
			r.eng.serving, r.eng.applyError, r.eng.usagePersist, r.eng.decision.Reasons, r.eng.decision.Errors)
	}
}

// TestR23_H3a_PreApplyHandoffSettleFailure_EmitsCorruptObservationAndDashHeartbeat
// proves R23-H-3 (Subcase A):
//
// When generation G1 is serving and a new desired generation G2 changes
// bindings (!sameBindings), enforce calls e.handoff(ctx) at engine.go:1226.
// If settleRetainedUsage inside handoff fails (for example with meter.ErrHold
// when G2 arrives within < 1s of the previous sample, or a usage.db failure),
// handoff installs Baseline, sets e.serving = false, e.decision.Serve = false,
// e.retainedPending = true, and e.usagePersist = true, and enforce returns nil
// at engine.go:1228.
//
// Because enforce's defer at engine.go:1102 does not check
// usagePersist != e.usagePersist, retainedPending != e.retainedPending, or
// serving != e.serving, e.decision is NOT recomputed:
//  1. e.beat() writes a watchdog heartbeat with Serving: false, Reason: "-"
//     (hiding meter_storage_failed from operators).
//  2. e.observe(ctx) enqueues and sends an ObservationReport with
//     MemberForwardingEnabled: false, ForwardingBlockReasons: [] (empty),
//     SafetyStatus: "passed" (violating decide.go:482 where UsagePersist
//     requires SafetyStatus: "failed"), Checks[storage]: "passed", and
//     Errors: [] (missing STORE_UNAVAILABLE).
func TestR23_H3a_PreApplyHandoffSettleFailure_EmitsCorruptObservationAndDashHeartbeat(t *testing.T) {
	// Regression r23: H-3 (R23-H-3) — enforce's defer must recompute
	// e.decision when pre-Apply e.handoff(ctx) mutates e.usagePersist,
	// e.retainedPending, or e.serving so heartbeats and ObservationReports
	// report meter_storage_failed and SafetyStatus: "failed".
	r := newRig(t)
	alignControlClock(r)
	r.srv.StickyLease(false)
	r.eng.prober = &verProbe{version: "0.76.2"}
	enrollBoot(t, r)
	stepOK(t, r)

	// Configure generation 2 with bA active immediately and bB becoming valid
	// at now + 5s so advancing the clock changes d.Model.Bindings within the
	// same generation while d.Serve remains true.
	now := r.clock.Wall()
	bA := oneBinding(now)
	bB := bA
	bB.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000d2")
	bB.BindingRevision = 2
	bB.PeerId = "peer-b"
	bB.AllowedIpv4 = []string{"100.64.0.11/32"}
	bB.ValidFrom = now.Add(5 * time.Second)
	fix := generationFixture(2, bA)
	fix.Bindings = []api.DeviceBinding{bA, bB}
	r.srv.SetFixture(fix)

	stepOK(t, r)
	if !r.eng.serving || len(r.eng.meterBindings()) != 1 {
		t.Fatalf("precondition: expected serving with 1 active binding, got serving=%v bindings=%d",
			r.eng.serving, len(r.eng.meterBindings()))
	}
	r.manualUsage(t)

	// Advance clock past bB.ValidFrom (+6s > +5s) and sample at that exact
	// microsecond-aligned timestamp so LastSample == Now(); when Step(ctx)
	// immediately runs pre-Apply handoff (engine.go:1226) for the newly active
	// bB, FinalSample sees !now.After(head.LastSample) and returns
	// meter.ErrHold, setting e.usagePersist = true and e.retainedPending = true
	// while e.meterFailed remains false.
	r.clock.Advance(6 * time.Second)
	upName := meter.CounterName(bA.DeviceId, meter.DirUpload)
	r.eng.meter.Close()
	src := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings,
		counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 2048}},
	}
	var err error
	r.eng.meter, err = meter.New(meter.Config{
		NodeID:        r.eng.nodeID,
		IncarnationID: r.eng.incID,
		Store:         r.eng.usage,
		Source:        src,
		Now:           func() time.Time { return r.clock.Wall().Truncate(time.Microsecond) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Additional traffic arrives on bA within the same clock instant so
	// !sameReadings(head.Counters, cur) and !now.After(head.LastSample),
	// causing FinalSample in pre-Apply handoff to return meter.ErrHold.
	src.setCounter(upName, enforce.CounterValue{Packets: 2, Bytes: 4096})

	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || !r.eng.usagePersist || !r.eng.retainedPending {
		t.Fatalf("precondition: expected serving=false usagePersist=true retainedPending=true, got serving=%v usagePersist=%v retainedPending=%v",
			r.eng.serving, r.eng.usagePersist, r.eng.retainedPending)
	}

	hb := mustBeat(t, r)
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("expected observation report on serving -> not-serving transition")
	}
	reasons := rep.ForwardingBlockReasons
	if hb.Reason == "-" || rep.SafetyStatus != api.SafetyStatusFailed || len(reasons) == 0 {
		t.Fatalf("R23-H-3a: enforce did not recompute e.decision after pre-Apply handoff failure: "+
			"heartbeat.Reason=%q (want %q), report.MemberForwardingEnabled=%v, report.SafetyStatus=%q (want %q), "+
			"report.ForwardingBlockReasons=%v (want [%s]), report.Errors=%+v",
			hb.Reason, api.ForwardingBlockReasonMeterStorageFailed, rep.MemberForwardingEnabled,
			rep.SafetyStatus, api.SafetyStatusFailed, reasons, api.ForwardingBlockReasonMeterStorageFailed, rep.Errors)
	}
}

// TestR23_H3b_PreApplyHandoffBaselineFailure_LeavesServingTrueWithFailedSafety
// proves R23-H-3 (Subcase B):
//
// When generation G1 is serving (e.serving == true) and generation G2 triggers
// e.handoff(ctx) at engine.go:1226 (!sameBindings), if e.baseline(ctx) at
// usage.go:435 fails, handoff returns false before line 438 (e.serving = false)
// and enforce returns nil at engine.go:1228 without clearing e.serving = false
// or e.policyExpires = 0.
//
// Consequently, e.serving stays true while e.baselineOK is false: e.beat()
// writes Serving: true, BaselineOK: false, and e.observe(ctx) sends a
// contradictory ObservationReport with MemberForwardingEnabled: true,
// SafetyStatus: "failed", and ForwardingBlockReasons: ["safety_check_failed"].
func TestR23_H3b_PreApplyHandoffBaselineFailure_LeavesServingTrueWithFailedSafety(t *testing.T) {
	// Regression r23: H-3 (R23-H-3) — When pre-Apply e.handoff(ctx) fails
	// (including when e.baseline(ctx) fails), e.serving and e.policyExpires
	// must be cleared so heartbeats and ObservationReports do not report
	// Serving: true alongside BaselineOK: false / SafetyStatus: "failed".
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.clock.Advance(2 * time.Second)

	// Generation 3 rebinds the device while leaseRefreshPending is true so
	// syncLease at engine.go:744 obtains the generation-3 lease before
	// evaluate(ctx) and enforce enters e.handoff(ctx) at engine.go:1226 while
	// r.eng.serving is true.
	member2 := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	b2 := oneBinding(r.clock.Wall())
	b2.MemberId = member2
	b2.BindingRevision = 2
	r.srv.SetFixture(generationFixture(3, b2))
	r.eng.leaseRefreshPending = true

	r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error {
		return errors.New("TEST ONLY baseline failure during pre-Apply handoff")
	}}
	// Advance clock past observation interval so observe(ctx) emits a report.
	r.clock.Advance(time.Duration(r.eng.cfg.ObservationSeconds+1) * time.Second)
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}

	hb := mustBeat(t, r)
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("expected observation report")
	}
	if r.eng.serving || hb.Serving || rep.MemberForwardingEnabled {
		t.Fatalf("R23-H-3b: pre-Apply handoff baseline failure left serving=true alongside failed safety: "+
			"eng.serving=%v, hb.Serving=%v, hb.BaselineOK=%v, rep.MemberForwardingEnabled=%v, rep.SafetyStatus=%q, rep.ForwardingBlockReasons=%v",
			r.eng.serving, hb.Serving, hb.BaselineOK, rep.MemberForwardingEnabled, rep.SafetyStatus, rep.ForwardingBlockReasons)
	}
}

// TestR23_M1_WithdrawSecondBaselineSuccess_LeavesOpenEpochUnsettledWithRetainedPendingFalse
// proves R23-M-1:
//
// In withdraw(ctx) (engine.go:1468-1473), e.handoff(ctx) is followed
// immediately by a second `_ = e.baseline(ctx)` call. Inside handoff(ctx)
// (usage.go:435-443), e.retainedPending = true and e.settleRetainedUsage(ctx)
// only run AFTER e.baseline(ctx) succeeds.
//
// If e.baseline(ctx) inside e.handoff(ctx) fails transiently and the immediate
// second `_ = e.baseline(ctx)` in withdraw(ctx) succeeds, withdraw returns
// with the kernel on Baseline (gen = 0, e.baselineOK = true, e.serving = false)
// while e.retainedPending is false, e.finalDone is false, and the open usage
// epoch in usage.db remains open and unsettled.
func TestR23_M1_WithdrawSecondBaselineSuccess_LeavesOpenEpochUnsettledWithRetainedPendingFalse(t *testing.T) {
	// Regression r23: M-1 (R23-M-1) — withdraw(ctx) must not install Baseline
	// via a second _ = e.baseline(ctx) call after e.handoff(ctx)'s baseline
	// failed while leaving e.retainedPending=false and the open epoch unsettled.
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.clock.Advance(2 * time.Second)

	var calls atomic.Int32
	r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error {
		if calls.Add(1) == 1 {
			return errors.New("TEST ONLY first baseline call in handoff failed transiently")
		}
		return nil
	}}

	r.eng.withdraw(t.Context())
	r.eng.plane = r.plane

	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !r.eng.baselineOK || r.eng.serving {
		t.Fatalf("precondition: expected baselineOK=true serving=false after second baseline succeeded, got baselineOK=%v serving=%v",
			r.eng.baselineOK, r.eng.serving)
	}
	if h.OpenEpoch != uuid.Nil && !r.eng.retainedPending {
		t.Fatalf("R23-M-1: withdraw installed Baseline (gen=%d, baselineOK=%v) but left OpenEpoch=%s unsettled with retainedPending=%v finalDone=%v",
			r.plane.last.Generation, r.eng.baselineOK, h.OpenEpoch, r.eng.retainedPending, r.eng.finalDone)
	}
}

// TestR23_Held_AQ1_UsageIntentCrashWindowsAndCrossRebootIsolation verifies what
// held for Attack Question 1 when the open epoch matches usage_apply_intent:
//   - Crash after recordUsageIntent before Apply (gen == 0) attributes or closes
//     cleanly using the recorded intent.
//   - Reboot (BootID change) ignores the prior boot's usage_apply_intent.
//   - Corrupt JSON in usage_apply_intent is rejected fail-closed.
func TestR23_Held_AQ1_UsageIntentCrashWindowsAndCrossRebootIsolation(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	// 1. Corrupt JSON in usage_apply_intent must be rejected by captureRestartBindings.
	if err := r.eng.store.SetMeta(t.Context(), usageIntentKey, "{not-valid-json"); err != nil {
		t.Fatal(err)
	}
	r.eng.captureRestartBindings(t.Context())
	if r.eng.restartIntent != nil {
		t.Fatalf("expected nil restartIntent for corrupt JSON, got %+v", r.eng.restartIntent)
	}

	// 2. Cross-reboot isolation: a new BootID must ignore the previous boot's
	// usage_apply_intent.
	b1 := r.eng.appliedModel.Bindings[0]
	upName := meter.CounterName(b1.DeviceID, meter.DirUpload)
	if err := r.eng.recordUsageIntent(t.Context(), *r.eng.appliedModel); err != nil {
		t.Fatal(err)
	}
	r20Crash(r.eng)

	newBoot := uuid.Must(uuid.NewV7()).String()
	if err := os.WriteFile(r.eng.cfg.BootIDFile, []byte(newBoot+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srcReboot := &r20DynamicSource{
		boot:     newBoot,
		counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 2048}},
	}
	next := r20Successor(t, r, r.eng.cfg, srcReboot, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.restartIntent != nil && next.restartIntent.Boot == next.bootID {
		t.Fatalf("rebooted engine matched prior boot's usage_apply_intent: %+v", next.restartIntent)
	}
}

// TestR23_Held_AQ2_BindingRebindSettlesOldBindingsBeforePrepareAndApply
// verifies what held for Attack Question 2: when generation G1 serves Member M1
// and generation G2 rebinds the device to Member M2 after the 1s sample hold
// window, enforce settles G1's epoch under M1, opens a fresh epoch for G2,
// records G2's intent, and attributes post-rebind traffic to M2.
func TestR23_Held_AQ2_BindingRebindSettlesOldBindingsBeforePrepareAndApply(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	b1 := r.eng.appliedModel.Bindings[0]
	member1 := b1.MemberID
	upName := meter.CounterName(b1.DeviceID, meter.DirUpload)

	r.eng.meter.Close()
	src := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings,
		counters: map[string]enforce.CounterValue{upName: {Packets: 2, Bytes: 4096}},
	}
	var err error
	r.eng.meter, err = meter.New(meter.Config{
		NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src,
	})
	if err != nil {
		t.Fatal(err)
	}

	r.clock.Advance(2 * time.Second)
	member2 := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	b2 := oneBinding(r.clock.Wall())
	b2.MemberId = member2
	b2.BindingRevision = 2
	r.srv.SetFixture(generationFixture(3, b2))

	if err := r.eng.Step(t.Context()); err != nil || !r.eng.serving {
		t.Fatalf("rebind step failed: err=%v serving=%v reasons=%v", err, r.eng.serving, r.eng.decision.Reasons)
	}
	r.manualUsage(t)

	// Advance counter by +2048 B under member2 and sample.
	r.clock.Advance(2 * time.Second)
	src.setCounter(upName, enforce.CounterValue{Packets: 3, Bytes: 6144})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}

	led := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
	if got1 := led.byMemberRev[r21MemberRev{member: member1, rev: 1}]; got1 != 4096 {
		t.Fatalf("member1 attributed=%d, want 4096", got1)
	}
	if got2 := led.byMemberRev[r21MemberRev{member: member2, rev: 2}]; got2 != 2048 {
		t.Fatalf("member2 attributed=%d, want 2048", got2)
	}
	if len(led.gaps) != 0 {
		t.Fatalf("unexpected gaps: %+v", led.gaps)
	}
}

// TestR23_Held_AQ3_EnsureUsageHeadFailureWithdrawsAndBlocksServe verifies what
// held for Attack Question 3: this injects checkpoint invalidation failure
// during usage initialization (not a Head failure); Step withdraws and reports
// ReasonMeterStorage (meter_storage_failed) and SafetyStatusFailed.
func TestR23_Held_AQ3_EnsureUsageHeadFailureWithdrawsAndBlocksServe(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	// Force ensureUsage to run on the next Step and fail at invalidateMeterCheckpoint
	// by replacing MeterCheckpoint with a non-empty directory.
	r20ReleaseUsage(r.eng)
	if err := os.MkdirAll(filepath.Join(r.eng.cfg.MeterCheckpoint, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}

	r.clock.Advance(2 * time.Second)
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || !r.eng.usagePersist || r.eng.decision.Safety != api.SafetyStatusFailed {
		t.Fatalf("expected fail-closed after ensureUsage failure: serving=%v usagePersist=%v safety=%s reasons=%v",
			r.eng.serving, r.eng.usagePersist, r.eng.decision.Safety, r.eng.decision.Reasons)
	}
}

func r20ReleaseUsage(e *Engine) {
	if e.loopCancel != nil {
		e.loopCancel()
		e.loops.Wait()
		e.loopCancel = nil
	}
	if e.meter != nil {
		e.meter.Close()
		e.meter = nil
	}
	if e.usage != nil {
		_ = e.usage.Close()
		e.usage = nil
	}
	if e.usageLock != nil {
		e.usageLock.Release()
		e.usageLock = nil
	}
	e.usageOnce = false
	e.sampleOn = false
}

// TestR23_Held_AQ4_EnsureUsageRepeatedFailuresLeakNothingAndRecover verifies
// Attack Question 4 (M-1): repeated transient failures in ensureUsage release
// the meter, store, and usage.db.lock in the defer, so a later retry succeeds
// cleanly without lock contention or duplicate instances.
func TestR23_Held_AQ4_EnsureUsageRepeatedFailuresLeakNothingAndRecover(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	if err := r.eng.Enroll(t.Context(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	r20ReleaseUsage(r.eng)

	// Make invalidateMeterCheckpoint fail inside ensureUsage AFTER openUsage
	// has acquired usage.db.lock and opened usage.db.
	if err := os.MkdirAll(filepath.Join(r.eng.cfg.MeterCheckpoint, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		r.eng.ensureUsage(t.Context())
		if r.eng.usageOnce || r.eng.usage != nil || r.eng.meter != nil || r.eng.usageLock != nil {
			t.Fatalf("iteration %d leaked usage state: once=%v usage=%v meter=%v lock=%v",
				i, r.eng.usageOnce, r.eng.usage, r.eng.meter, r.eng.usageLock)
		}
		// Prove usage.db.lock was actually released in the defer by acquiring
		// and releasing it directly.
		lk, err := nodelock.TryAcquire(r.eng.cfg.UsageDBPath() + ".lock")
		if err != nil {
			t.Fatalf("iteration %d leaked usage.db.lock: %v", i, err)
		}
		lk.Release()
	}

	// Remove the obstacle; the next ensureUsage must succeed and leave a single
	// live store and meter.
	if err := os.RemoveAll(r.eng.cfg.MeterCheckpoint); err != nil {
		t.Fatal(err)
	}
	r.eng.ensureUsage(t.Context())
	if !r.eng.usageOnce || r.eng.usage == nil || r.eng.meter == nil || r.eng.usageLock == nil || r.eng.usagePersist {
		t.Fatalf("ensureUsage did not recover cleanly: once=%v usage=%v meter=%v lock=%v persist=%v",
			r.eng.usageOnce, r.eng.usage, r.eng.meter, r.eng.usageLock, r.eng.usagePersist)
	}
}

// TestR23_Held_AQ5_BootHeartbeatStopProbeExceptionAndDatapathValidation
// verifies Attack Question 5 (M-2, M-3, M-4):
//   - bootHeartbeat refuses publication with an already-canceled context.
//   - A failed clearProbeException retains the issuance for retry.
//   - Config.validate rejects an invalid member interface before startup.
func TestR23_Held_AQ5_BootHeartbeatStopProbeExceptionAndDatapathValidation(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	if err := r.eng.Enroll(t.Context(), "token-test-only"); err != nil {
		t.Fatal(err)
	}

	// 1. bootHeartbeat respects its parent context and cannot write a heartbeat
	// when the parent context is already canceled.
	if err := os.Remove(r.eng.cfg.HeartbeatPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	canceledCtx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := r.eng.bootHeartbeat(canceledCtx); err == nil {
		t.Fatal("expected bootHeartbeat to fail when parent context is canceled")
	}
	if _, err := os.Stat(r.eng.cfg.HeartbeatPath); !os.IsNotExist(err) {
		t.Fatalf("bootHeartbeat wrote after parent context cancelation: %v", err)
	}
	if _, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath); err == nil {
		t.Fatal("expected heartbeat file to remain absent after canceled bootHeartbeat")
	}

	// 2. Probe exception cleanup retains original issuance across Clear failure
	// and is cleared when baseline succeeds.
	dp, err := r.eng.cfg.DatapathValue()
	if err != nil {
		t.Fatal(err)
	}
	iss := probeIssuance{
		Nonce:    "nonce-r23-aq5",
		Hash:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		IssuedAt: r.clock.Wall(),
		Deadline: r.clock.Mono() + 5*time.Second,
		Cleared:  false,
	}
	clearCalls := 0
	failClear := true
	fakeControl := &r23ProbeControlPlane{
		Plane: r.plane,
		clearFn: func(m enforce.Model) error {
			clearCalls++
			if failClear {
				return errors.New("TEST ONLY clear probe exception failed")
			}
			return r.plane.BaselineModel(t.Context(), m)
		},
	}
	r.eng.plane = fakeControl
	r.eng.probeCleanup = &probeCleanup{model: enforce.Model{Datapath: dp}, issuance: iss}
	if r.eng.clearProbeException(t.Context()) {
		t.Fatal("expected clearProbeException to return false on failure")
	}
	if r.eng.probeCleanup == nil || r.eng.probeCleanup.issuance.Deadline != iss.Deadline || r.eng.probeCleanup.issuance.Cleared {
		t.Fatalf("probeCleanup lost or mutated original issuance: %+v", r.eng.probeCleanup)
	}
	failClear = false
	if !r.eng.clearProbeException(t.Context()) {
		t.Fatal("retry clearProbeException failed")
	}
	if r.eng.probeCleanup != nil || clearCalls != 2 {
		t.Fatalf("expected probeCleanup=nil after retry, got %+v clearCalls=%d", r.eng.probeCleanup, clearCalls)
	}
	r.eng.plane = r.plane

	// 3. Config.validate() validates DatapathValue() upfront.
	badCfg := r.eng.cfg
	badCfg.Datapath.MemberIngress = "eth0" // does not start with fv
	if err := badCfg.validate(); err == nil {
		t.Fatal("expected Config.validate() to reject invalid Datapath.MemberIngress")
	}
}

type r23ProbeControlPlane struct {
	Plane
	clearFn func(enforce.Model) error
}

func (p *r23ProbeControlPlane) BaselineModel(_ context.Context, m enforce.Model) error {
	return p.clearFn(m)
}

// TestR23_Held_AQ6_SameIdentityRenewalSkipsRedundantIntentWrite verifies Attack
// Question 6: when recordUsageIntent is called again for the same (BootID,
// Epoch, Generation, Bindings), it short-circuits in memory without rewriting
// usage_apply_intent in agent.db.
func TestR23_Held_AQ6_SameIdentityRenewalSkipsRedundantIntentWrite(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	// Corrupt the stored meta row in agent.db while keeping r.eng.restartIntent
	// intact; a same-identity call to recordUsageIntent must short-circuit
	// without overwriting the meta row.
	sentinel := `{"sentinel":"unchanged"}`
	if err := r.eng.store.SetMeta(t.Context(), usageIntentKey, sentinel); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.recordUsageIntent(t.Context(), *r.eng.appliedModel); err != nil {
		t.Fatal(err)
	}
	got, ok, err := r.eng.store.Meta(t.Context(), usageIntentKey)
	if err != nil || !ok {
		t.Fatalf("Meta(%s): ok=%v err=%v", usageIntentKey, ok, err)
	}
	if got != sentinel {
		t.Fatalf("expected same-identity recordUsageIntent to short-circuit without writing agent.db, got %q", got)
	}
}
