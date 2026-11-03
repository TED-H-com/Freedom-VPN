package agent

// Adversarial security review r25 proof tests for agent-core follow-ups 40 and 41
// (commit 8cc2760965abbbb1af3340976c6c4adf1be3dc37).
// TEST ONLY in-memory data planes and disk SQLite; no host networking or root.

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass
// proves Finding H-1 (Subcase A) against 8cc27609:
//
// In buildModel (node/agent/decide.go:663-670), e.policyExpires is set to the
// earliest deadline across all admitted bindings (whether a shorter sibling
// device ValidUntil on the same member or a shorter member LeaseEnd).
// When the clock reaches that earliest deadline while another binding remains
// valid, Step (node/agent/engine.go:696, added in follow-up 41 commit 77689438)
// calls e.expireServing(ctx) (node/agent/serving.go:119-129) before the
// control-plane pass.
//
// expireServing sees e.clock.Mono() >= e.policyExpires and calls
// e.withdraw(cleanup) -> e.handoff(cleanup) -> e.baseline(cleanup) AND a second
// e.baseline(cleanup) in withdraw (engine.go:1485-1486), tearing down the
// ENTIRE node ruleset to fvpn_baseline=1 (baselines +2) and deleting all tc
// qdiscs BEFORE Step runs its HTTP control pass (flush, syncDesired, syncLease,
// syncCert, syncBootstrap).
//
// As a result:
//  1. Every time a single device or member grant reaches its deadline while
//     other bindings remain valid, all surviving bindings are blackholed in
//     Baseline across the entire HTTP control pass (instead of keeping the
//     surviving bindings serving and atomically applying the reduced binding set
//     in enforce).
//  2. If the control pass is interrupted (e.g., caller context canceled at
//     engine.go:717 or syncDesired/syncLease context timeout at engine.go:735,747),
//     Step exits in Baseline with e.serving=false and e.decision.Serve=true
//     (Reasons=nil), writing a corrupt Heartbeat{Serving:false, Reason:"-"} and
//     POSTing a contradictory ObservationReport{MemberForwardingEnabled:false,
//     ForwardingBlockReasons:[], DataHealth:"healthy", SafetyStatus:"passed"}.
type r25ProbeRecorder struct {
	next    Prober
	onProbe func()
}

func (p r25ProbeRecorder) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	if p.onProbe != nil {
		p.onProbe()
	}
	return p.next.Probe(ctx, req)
}

func TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass(t *testing.T) {
	for _, kind := range []string{"sibling_device_expiry", "sibling_member_expiry"} {
		t.Run(kind, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)

			now := r.clock.Wall()
			surviving := oneBinding(now)
			surviving.ValidUntil = now.Add(time.Hour)
			expiring := surviving
			expiring.DeviceId = uuid.Must(uuid.NewV7())
			expiring.AllowedIpv4 = []string{"100.64.0.12/32"}
			if kind == "sibling_member_expiry" {
				expiring.MemberId = uuid.Must(uuid.NewV7())
			} else {
				expiring.ValidUntil = now.Add(20 * time.Second)
			}

			f := fixtureWithBindings(3, []api.DeviceBinding{surviving, expiring})
			f.TTL = 2 * time.Hour
			r.srv.SetFixture(f)
			stepOK(t, r)
			if kind == "sibling_member_expiry" {
				for i := range r.eng.lease.Members {
					if r.eng.lease.Members[i].MemberId == expiring.MemberId {
						r.eng.lease.Members[i].LeaseEnd = now.Add(20 * time.Second)
					}
				}
				stepOK(t, r)
			}

			r.clock.Advance(10 * time.Second)
			stepOK(t, r)
			if !r.eng.serving || len(r.plane.last.Bindings) != 2 {
				t.Fatalf("precondition: expected 2 bindings serving at T+10s, got serving=%v bindings=%d", r.eng.serving, len(r.plane.last.Bindings))
			}

			baselinesBefore := r.plane.baselines
			servingDuringControlPass := true
			bindingsDuringControlPass := -1
			r.eng.prober = r25ProbeRecorder{
				next: r.eng.prober,
				onProbe: func() {
					servingDuringControlPass = r.eng.serving
					bindingsDuringControlPass = len(r.plane.last.Bindings)
				},
			}

			r.clock.Advance(10 * time.Second)
			stepOK(t, r)

			// While `surviving` remains valid for 59m40s, Step entry must NOT tear
			// down the whole node to Baseline across the HTTP control/probe pass
			// before enforce applies the reduced 1-binding set.
			if !servingDuringControlPass || bindingsDuringControlPass == 0 {
				// TODO(review r25): H-1 — When a single device or member grant reaches its deadline while another binding on the node remains valid, Step's expireServing tears down the entire node ruleset to Baseline before the HTTP control and probe pass instead of keeping surviving bindings serving until enforce applies the reduced binding set.
				t.Fatalf("TODO(review r25): H-1 (%s): single-binding expiry while sibling remains valid tore down the entire node to Baseline before the control/probe pass (servingDuringControlPass=%v bindingsDuringControlPass=%d baselines=%d->%d)",
					kind, servingDuringControlPass, bindingsDuringControlPass, baselinesBefore, r.plane.baselines)
			}
		})
	}
}

// TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat
// proves Finding H-1 (Subcase B) against 8cc27609:
//
// When a Step starts shortly before the earliest binding deadline (so
// e.clock.Mono() < e.policyExpires at Step entry line 696) and the clock
// reaches e.policyExpires either:
//  1. During verifyCurrentServing (node/agent/serving.go:111), or
//  2. During enforce's canRetainServing live verification (node/agent/engine.go:1182-1186),
//
// e.expireServing(ctx) withdraws the entire node to Baseline (e.serving = false)
// and computes e.decision = Decide(e.input()) (which has Serve = true,
// Reasons = nil, DataHealth = "healthy", Safety = "passed" for the surviving sibling).
// Step then returns early (at engine.go:712 or engine.go:1186) without applying
// the surviving sibling's model!
// Finally, finishServingStep (serving.go:167-186) -> e.beat() (engine.go:1496-1501)
// sees !e.serving && len(e.decision.Reasons) == 0, writes a corrupt watchdog
// heartbeat with Serving=false, PolicyExpires=0, Reason="-", and POSTs a
// contradictory ObservationReport with MemberForwardingEnabled=false,
// ForwardingBlockReasons=[], DataHealth="healthy", SafetyStatus="passed", leaving
// the surviving sibling completely unserved in Baseline for the next poll interval.
func TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat(t *testing.T) {
	for _, path := range []string{"during_verifyCurrentServing", "during_enforce_canRetainServing"} {
		t.Run(path, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)

			now := r.clock.Wall()
			surviving := oneBinding(now)
			surviving.ValidUntil = now.Add(time.Hour)
			expiring := surviving
			expiring.DeviceId = uuid.Must(uuid.NewV7())
			expiring.AllowedIpv4 = []string{"100.64.0.12/32"}
			expiring.ValidUntil = now.Add(20 * time.Second)

			f := fixtureWithBindings(3, []api.DeviceBinding{surviving, expiring})
			f.TTL = 2 * time.Hour
			r.srv.SetFixture(f)
			stepOK(t, r)
			// Pin fake control's signer clock so subsequent polls return 304 Not Modified.
			signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
			r.srv.SetClock(func() time.Time { return signedAt })

			deadline := r.eng.policyExpires
			switch path {
			case "during_verifyCurrentServing":
				// Advance to 1ms before e.policyExpires; the live read in
				// verifyCurrentServing spans 1ms to e.policyExpires.
				r.clock.Advance(deadline - r.clock.Mono() - time.Millisecond)
				r.eng.plane = &livePlane{
					memPlane: r.plane,
					read: func(context.Context) error {
						r.clock.Advance(time.Millisecond)
						return nil
					},
				}
			case "during_enforce_canRetainServing":
				// Advance to 200ms before e.policyExpires so verifyCurrentServing at
				// Step line 709 and Decide at Step line 765 both see >101ms remaining
				// (both bindings still admitted, canRetainServing == true at engine.go:1181),
				// and the second VerifyServing call (at engine.go:1182) spans 200ms to
				// e.policyExpires.
				r.clock.Advance(deadline - r.clock.Mono() - 200*time.Millisecond)
				calls := 0
				r.eng.plane = &livePlane{
					memPlane: r.plane,
					read: func(context.Context) error {
						calls++
						if calls == 2 {
							r.clock.Advance(200 * time.Millisecond)
						}
						return nil
					},
				}
			}

			stepOK(t, r)
			hb := mustBeat(t, r)
			rep, ok := r.srv.LastObservation()
			if !ok {
				t.Fatal("expected an observation report")
			}

			if !r.eng.serving || len(r.plane.last.Bindings) != 1 || !hb.Serving || hb.Reason == "-" || !rep.MemberForwardingEnabled {
				// TODO(review r25): H-1 — When the earliest device or member deadline is reached during verifyCurrentServing or enforce's canRetainServing while a sibling binding remains valid, expireServing withdraws the entire node to Baseline, writes a corrupt Reason="-" watchdog heartbeat, POSTs a contradictory ObservationReport (MemberForwardingEnabled=false, ForwardingBlockReasons=[], DataHealth=healthy, SafetyStatus=passed), and returns early from Step without re-applying the surviving sibling.
				t.Fatalf("TODO(review r25): H-1 (%s): surviving sibling left unserved in Baseline after expireServing early return: eng.serving=%v bindings=%d heartbeat={Serving:%v PolicyExpires:%s Reason:%q} lastObs={MemberForwardingEnabled:%v Reasons:%v DataHealth:%v SafetyStatus:%v}",
					path, r.eng.serving, len(r.plane.last.Bindings), hb.Serving, hb.PolicyExpires, hb.Reason,
					rep.MemberForwardingEnabled, rep.ForwardingBlockReasons, rep.DataHealth, rep.SafetyStatus)
			}
		})
	}
}

// TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick
// proves Finding M-1 against 8cc27609:
//
// Whenever the open usage epoch in usage.db advances via a 'counter_discontinuity'
// switch while the same serving generation remains active — either:
//  1. At the end of enforce (engine.go:1416) when e.switchEpoch(ctx) calls
//     e.meter.NotifyRulesetSwitch(clean) after e.recordUsageIntent(ctx, model)
//     at engine.go:1268 already recorded the pre-Apply epoch E1, or
//  2. During serving when Meter.Sample observes a counter decrease / ruleset
//     reset and opens a successor epoch E2 with 'counter_discontinuity' —
//
// Head.OpenEpoch advances to E2 (with SameCountingInterval(E1, E2) == true),
// while e.restartIntent.Epoch remains pinned to E1.
// On the next 10-second steady-state 304 poll tick, canRetainServing
// (node/agent/retain.go:47-54) requires strict equality `intent.Epoch == h.OpenEpoch`
// instead of checking `e.usage.SameCountingInterval` (or refreshing the intent
// in switchEpoch), rejects retention, and executes a redundant Plane.Apply
// (inflicting a second tc -batch + fv_deadline_guard drop window) even though
// DesiredState returned 304 Not Modified and the lease and model are unchanged.
func TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick(t *testing.T) {
	for _, trigger := range []string{"post_apply_switchEpoch", "sample_counter_discontinuity"} {
		t.Run(trigger, func(t *testing.T) {
			r, src, up := f40Live(t)

			switch trigger {
			case "post_apply_switchEpoch":
				// Wire Engine.kernel as the meter's source (matching production where
				// Options.MeterSource is nil) with a pending ruleset-switch nudge (as
				// set after a baseline fallback or generation repair).
				ks := &kernelSource{
					boot:     r.eng.bootID.String(),
					bindings: r.eng.meterBindings,
					expect:   func() bool { return r.eng.expectServing.Load() },
					read: func(context.Context) (enforce.State, error) {
						return enforce.State{
							Generation: r.plane.last.Generation,
							Counters:   map[string]enforce.CounterValue{up: {Packets: 2, Bytes: 8192}},
						}, nil
					},
					prev:  map[string]enforce.CounterValue{up: {Packets: 2, Bytes: 8192}},
					nudge: true,
				}
				r.eng.kernel = ks
				r.eng.meter.Close()
				var err error
				r.eng.meter, err = meter.New(meter.Config{
					NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: ks,
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := r.eng.meter.Sample(t.Context()); err != nil {
					t.Fatal(err)
				}
				// Re-arm the nudge before the Apply pass so enforce runs recordUsageIntent
				// on E1 (engine.go:1268), applies the model, and then switchEpoch(ctx)
				// (engine.go:1416) calls NotifyRulesetSwitch, closing E1 and opening E2.
				ks.mu.Lock()
				ks.nudge = true
				ks.mu.Unlock()
				r.clock.Advance(time.Second)
				r.eng.leaseRefreshPending = true
				stepOK(t, r)

			case "sample_counter_discontinuity":
				// Setup renewal + Apply so the current lease and model are synced.
				r.clock.Advance(time.Second)
				r.eng.leaseRefreshPending = true
				stepOK(t, r)
				// A counter reset during serving closes E1 and opens E2 with
				// 'counter_discontinuity' in usage.db while G2 continues serving.
				r.clock.Advance(time.Second)
				src.setCounter(up, enforce.CounterValue{Packets: 1, Bytes: 1024})
				if err := r.eng.meter.Sample(t.Context()); err != nil {
					t.Fatal(err)
				}
			}

			// Pin fake control's signer clock so the next 10s poll is an actual 304 Not Modified.
			signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
			r.srv.SetClock(func() time.Time { return signedAt })

			hAfterSwitch, err := r.eng.usage.Head(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if r.eng.restartIntent == nil {
				t.Fatal("expected non-nil restartIntent")
			}
			intentEpochBefore := r.eng.restartIntent.Epoch
			sameInterval, err := r.eng.usage.SameCountingInterval(
				t.Context(),
				intentEpochBefore,
				hAfterSwitch.OpenEpoch,
				hAfterSwitch.BootID,
			)
			if err != nil || !sameInterval || intentEpochBefore == hAfterSwitch.OpenEpoch {
				t.Fatalf("precondition (%s): expected intent.Epoch (%s) != Head.OpenEpoch (%s) with SameCountingInterval=true, got sameInterval=%v err=%v",
					trigger, intentEpochBefore, hAfterSwitch.OpenEpoch, sameInterval, err)
			}

			appliesBefore := r.plane.applies
			renewalsBefore := r.srv.LeaseRenewals()

			// Run a normal 10-second steady-state 304 poll tick.
			r.clock.Advance(10 * time.Second)
			stepOK(t, r)

			if r.srv.LeaseRenewals() != renewalsBefore {
				t.Fatalf("unexpected lease renewal on 304 tick: %d -> %d", renewalsBefore, r.srv.LeaseRenewals())
			}
			if r.plane.applies != appliesBefore {
				// TODO(review r25): M-1 — When the open usage epoch advances via NotifyRulesetSwitch in enforce's post-Apply switchEpoch (engine.go:1416) or via a counter-discontinuity switch in Sample while the same generation and bindings remain active, e.restartIntent.Epoch remains on the prior epoch E1; canRetainServing (retain.go:53) requires strict intent.Epoch == h.OpenEpoch instead of SameCountingInterval (or refreshing the intent), forcing a redundant Plane.Apply on the next 304 tick.
				t.Fatalf("TODO(review r25): M-1 (%s): stale restartIntent.Epoch=%s != Head.OpenEpoch=%s (SameCountingInterval=true) forced a redundant Plane.Apply on a steady-state 304 tick (applies %d -> %d, want %d)",
					trigger, intentEpochBefore, hAfterSwitch.OpenEpoch, appliesBefore, r.plane.applies, appliesBefore)
			}
		})
	}
}

// TestR25_Held_AQ1_RetainedTickVerifiesUnderLockAndBlocksOnAnyPendingGateOrClockSkew
// verifies Attack Question 1 ("What held"):
//  1. A retained 304 tick acquires the node lock, runs live VerifyServing and
//     MemberPath checks, and refreshes the watchdog heartbeat without calling Apply.
//  2. Every pending recovery, revocation, usage, or health gate on Engine
//     blocks canRetainServing.
//  3. A wall-clock jump exceeding DefaultBackwardSkew (2s) or DefaultForwardSkew
//     (60s) trips verifyTime() (e.anchorOK = false) so Decide blocks serving
//     with clock_untrusted instead of retaining.
func TestR25_Held_AQ1_RetainedTickVerifiesUnderLockAndBlocksOnAnyPendingGateOrClockSkew(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.clock.Advance(time.Second)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)

	p := &f41ReadPlane{memPlane: r.plane, lock: r.eng.cfg.LockPath}
	r.eng.plane = p
	signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
	r.srv.SetClock(func() time.Time { return signedAt })

	appliesBefore := r.plane.applies
	readsBefore := p.lockedReads
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	hb := mustBeat(t, r)
	if r.plane.applies != appliesBefore || p.lockedReads != readsBefore+1 || !hb.Serving || hb.Reason != "serving" {
		t.Fatalf("retained tick did not verify under lock or heartbeat: applies=%d/%d reads=%d/%d hb=%+v",
			r.plane.applies, appliesBefore, p.lockedReads, readsBefore, hb)
	}

	d := Decide(r.eng.input())
	if !r.eng.canRetainServing(t.Context(), d) {
		t.Fatal("baseline canRetainServing should be true on healthy retained state")
	}

	// Verify each pending/failure gate on Engine blocks canRetainServing.
	gates := []struct {
		name   string
		mutate func(*Engine) func()
	}{
		{"applyError", func(e *Engine) func() {
			e.applyError = &api.NodeError{Code: api.ErrorCodeHEALTHCHECKFAILED, Message: "test"}
			return func() { e.applyError = nil }
		}},
		{"failedCandidate", func(e *Engine) func() {
			e.failedCandidate = &failedCandidate{Generation: e.appliedGen}
			return func() { e.failedCandidate = nil }
		}},
		{"latchBlocked", func(e *Engine) func() {
			e.latchBlocked = true
			return func() { e.latchBlocked = false }
		}},
		{"repairing", func(e *Engine) func() {
			e.repairing = true
			return func() { e.repairing = false }
		}},
		{"pendingRevoke", func(e *Engine) func() {
			e.pendingRevoke = []addrID{{Addr: netip.MustParseAddr("100.64.0.99")}}
			return func() { e.pendingRevoke = nil }
		}},
		{"conntrackAttempts", func(e *Engine) func() {
			e.conntrackAttempts = 1
			return func() { e.conntrackAttempts = 0 }
		}},
		{"conntrackBlocked", func(e *Engine) func() {
			e.conntrackBlocked = true
			return func() { e.conntrackBlocked = false }
		}},
		{"memberPathNil", func(e *Engine) func() {
			prev := e.memberPath
			e.memberPath = nil
			return func() { e.memberPath = prev }
		}},
		{"memberPathPending", func(e *Engine) func() {
			e.memberPathPending = true
			return func() { e.memberPathPending = false }
		}},
		{"quarantinePending", func(e *Engine) func() {
			e.quarantinePending = true
			return func() { e.quarantinePending = false }
		}},
		{"probeCleanup", func(e *Engine) func() {
			e.probeCleanup = &probeCleanup{}
			return func() { e.probeCleanup = nil }
		}},
		{"retainedPending", func(e *Engine) func() {
			e.retainedPending = true
			return func() { e.retainedPending = false }
		}},
		{"usagePersist", func(e *Engine) func() {
			e.usagePersist = true
			return func() { e.usagePersist = false }
		}},
		{"usageAdopting", func(e *Engine) func() {
			e.usageAdopting = true
			return func() { e.usageAdopting = false }
		}},
		{"epochClosed", func(e *Engine) func() {
			e.epochClosed = true
			return func() { e.epochClosed = false }
		}},
		{"finalDone", func(e *Engine) func() {
			e.finalDone = true
			return func() { e.finalDone = false }
		}},
		{"finalBarrierID", func(e *Engine) func() {
			e.finalBarrierID = "barrier-1"
			return func() { e.finalBarrierID = "" }
		}},
		{"shutUsage", func(e *Engine) func() {
			e.shutUsage = true
			return func() { e.shutUsage = false }
		}},
		{"sampleOff", func(e *Engine) func() {
			e.sampleOn = false
			return func() { e.sampleOn = true }
		}},
		{"expectServingFalse", func(e *Engine) func() {
			e.expectServing.Store(false)
			return func() { e.expectServing.Store(true) }
		}},
		{"kernelNudgePending", func(e *Engine) func() {
			prev := e.kernel
			e.kernel = &kernelSource{nudge: true}
			return func() { e.kernel = prev }
		}},
	}
	for _, g := range gates {
		restore := g.mutate(r.eng)
		ok := r.eng.canRetainServing(t.Context(), d)
		restore()
		if ok {
			t.Fatalf("gate %s did not block canRetainServing", g.name)
		}
	}

	// Verify wall-clock jump outside DefaultBackwardSkew trips verifyTime() and
	// withdraws when no fresh lease re-anchors the clock.
	r.srv.FailLeases(1)
	r.clock.JumpWall(-5 * time.Second)
	stepOK(t, r)
	if r.eng.serving || mustBeat(t, r).Serving || r.eng.anchorOK {
		t.Fatalf("backward wall-clock jump retained serving: serving=%v anchorOK=%v", r.eng.serving, r.eng.anchorOK)
	}
}

// TestR25_Held_AQ2_PerDeviceDeadlinesAndEdgeCases verifies Attack Question 2 ("What held"):
//  1. Two devices on the same member with equal or unequal ValidUntil in both
//     orders keep MemberLease.DeadlineMono/EffectiveServiceEnd/NodeLeaseEnd at
//     the member-only cap while giving each device its own timeout in enforce.Render.
//  2. Distant ValidUntil (+300y), zero ValidUntil, ValidUntil before IssuedAt,
//     and future ValidFrom are handled without overflow or order dependence.
func TestR25_Held_AQ2_PerDeviceDeadlinesAndEdgeCases(t *testing.T) {
	in := withAdmittedMember(servingInput())
	start := in.Lease.IssuedAt
	in.Now = start
	in.MonoNow = in.LeaseMonoSent

	// 1. Equal and unequal ValidUntil in both orders, plus zero ValidUntil (no device cap).
	d1 := oneBinding(start)
	d1.ValidUntil = start.Add(40 * time.Minute)
	d2 := d1
	d2.DeviceId = uuid.Must(uuid.NewV7())
	d2.AllowedIpv4 = []string{"100.64.0.12/32"}
	d2.ValidUntil = start.Add(10 * time.Minute)
	d3 := d1
	d3.DeviceId = uuid.Must(uuid.NewV7())
	d3.AllowedIpv4 = []string{"100.64.0.13/32"}
	d3.ValidUntil = time.Time{} // zero ValidUntil: uses member cap (1h)

	for _, order := range [][]api.DeviceBinding{{d1, d2, d3}, {d2, d3, d1}, {d3, d2, d1}} {
		cp := in
		ds := *in.Desired
		ds.DeviceBindings = append([]api.DeviceBinding(nil), order...)
		cp.Desired = &ds
		m, expiry, err := buildModel(cp)
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Leases) != 1 || *m.Leases[0].DeadlineMono != in.LeaseMonoSent+time.Hour {
			t.Fatalf("member lease deadline changed by device order: %+v", m.Leases)
		}
		if expiry != in.LeaseMonoSent+10*time.Minute {
			t.Fatalf("policyExpires=%s want=%s", expiry, in.LeaseMonoSent+10*time.Minute)
		}
		plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
		if err != nil {
			t.Fatal(err)
		}
		if got := plan.ElementTimeouts[d1.DeviceId.String()]; got != 40*time.Minute {
			t.Fatalf("d1 timeout=%s want=40m", got)
		}
		if got := plan.ElementTimeouts[d2.DeviceId.String()]; got != 10*time.Minute {
			t.Fatalf("d2 timeout=%s want=10m", got)
		}
		if got := plan.ElementTimeouts[d3.DeviceId.String()]; got != time.Hour {
			t.Fatalf("d3 (zero ValidUntil) timeout=%s want=1h", got)
		}
	}

	// 2. ValidUntil before IssuedAt (already expired) and future ValidFrom (not yet active)
	// are excluded from admitted bindings while keeping the active sibling admitted.
	expiredBeforeIssued := d1
	expiredBeforeIssued.DeviceId = uuid.Must(uuid.NewV7())
	expiredBeforeIssued.AllowedIpv4 = []string{"100.64.0.14/32"}
	expiredBeforeIssued.ValidUntil = start.Add(-time.Minute)

	futureValidFrom := d1
	futureValidFrom.DeviceId = uuid.Must(uuid.NewV7())
	futureValidFrom.AllowedIpv4 = []string{"100.64.0.15/32"}
	futureValidFrom.ValidFrom = start.Add(15 * time.Minute)

	cp := in
	ds := *in.Desired
	ds.DeviceBindings = []api.DeviceBinding{expiredBeforeIssued, futureValidFrom, d1}
	cp.Desired = &ds
	m, expiry, err := buildModel(cp)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Bindings) != 1 || m.Bindings[0].DeviceID != d1.DeviceId || expiry != in.LeaseMonoSent+40*time.Minute {
		t.Fatalf("expected only d1 admitted with 40m expiry, got bindings=%+v expiry=%s", m.Bindings, expiry)
	}
}

// TestR25_Held_AQ3_SameCountingIntervalAndHandoffRecoveryInvariants verifies
// Attack Question 3 ("What held"):
//  1. SameCountingInterval accepts chained 'counter_discontinuity' epochs on
//     the same boot_id and rejects reversed order, closed target epochs,
//     'final_sample' boundaries, or boot_id mismatches.
//  2. handoff's defer preserves a pre-existing usagePersist flag across a
//     healing FinalSample (H-3a), withdraw keeps retainedPending=true when
//     handoff's initial baseline fails (M-1), and enforce line 1189 blocks
//     Apply on subsequent Steps while MeterCheckpoint invalidation fails (H-3b').
func TestR25_Held_AQ3_SameCountingIntervalAndHandoffRecoveryInvariants(t *testing.T) {
	r, src, up := f40Live(t)
	h1, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	e1 := h1.OpenEpoch

	// Counter reset 1 -> opens E2 with 'counter_discontinuity'.
	r.clock.Advance(time.Second)
	src.setCounter(up, enforce.CounterValue{Packets: 1, Bytes: 512})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	h2, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	e2 := h2.OpenEpoch

	// Counter reset 2 -> opens E3 with 'counter_discontinuity'.
	r.clock.Advance(time.Second)
	src.setCounter(up, enforce.CounterValue{Packets: 1, Bytes: 256})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	h3, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	e3 := h3.OpenEpoch
	if e1 == e2 || e2 == e3 {
		t.Fatalf("expected distinct epochs E1=%s E2=%s E3=%s", e1, e2, e3)
	}

	// Forward chain E1 -> E3 on same boot is true; reversed E3 -> E1 is false;
	// E1 -> E2 (where E2 is now closed) is false; wrong boot_id is false.
	if ok, err := r.eng.usage.SameCountingInterval(t.Context(), e1, e3, h3.BootID); err != nil || !ok {
		t.Fatalf("expected SameCountingInterval(E1, E3)=true, got %v err=%v", ok, err)
	}
	if ok, err := r.eng.usage.SameCountingInterval(t.Context(), e3, e1, h3.BootID); err != nil || ok {
		t.Fatalf("expected reversed SameCountingInterval(E3, E1)=false, got %v err=%v", ok, err)
	}
	if ok, err := r.eng.usage.SameCountingInterval(t.Context(), e1, e2, h3.BootID); err != nil || ok {
		t.Fatalf("expected closed target SameCountingInterval(E1, E2)=false, got %v err=%v", ok, err)
	}
	if ok, err := r.eng.usage.SameCountingInterval(t.Context(), e1, e3, "other-boot"); err != nil || ok {
		t.Fatalf("expected wrong boot SameCountingInterval=false, got %v err=%v", ok, err)
	}

	// Verify handoff's defer preserves pre-existing usagePersist across a
	// healing FinalSample (H-3a), and closes E3 with 'final_sample'.
	r.clock.Advance(time.Second)
	r.eng.usagePersist = true
	r.eng.handoff(t.Context())
	if !r.eng.usagePersist {
		t.Fatal("handoff cleared pre-existing usagePersist flag")
	}

	// Verify a settled 'final_sample' boundary breaks SameCountingInterval
	// when a new epoch opens on the same boot.
	if err := r.eng.meter.PrepareSampling(t.Context()); err != nil {
		t.Fatal(err)
	}
	r.eng.meter.ResumeSampling()
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	h4, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := r.eng.usage.SameCountingInterval(t.Context(), e3, h4.OpenEpoch, h4.BootID); err != nil || ok {
		t.Fatalf("expected SameCountingInterval across final_sample boundary to be false, got %v err=%v", ok, err)
	}

	// Verify enforce line 1189 blocks Apply on subsequent Steps while
	// MeterCheckpoint invalidation fails (H-3b').
	r2, _, _ := f40Live(t)
	b := oneBinding(r2.clock.Wall())
	b.MemberId = uuid.Must(uuid.NewV7())
	b.BindingRevision = 2
	r2.srv.SetFixture(generationFixture(3, b))
	r2.eng.leaseRefreshPending = true
	r2.eng.plane = &f40CheckpointRace{memPlane: r2.plane, once: func() {
		if err := os.MkdirAll(r2.eng.cfg.MeterCheckpoint, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r2.eng.cfg.MeterCheckpoint, "held"), []byte("TEST ONLY"), 0o600); err != nil {
			t.Fatal(err)
		}
	}}
	appliesBefore := r2.plane.applies
	stepOK(t, r2)
	stepOK(t, r2)
	if r2.plane.applies != appliesBefore || r2.eng.serving || !r2.eng.usagePersist {
		t.Fatalf("expected MeterCheckpoint invalidation failure to keep serving blocked across Steps, got applies=%d/%d serving=%v usagePersist=%v",
			r2.plane.applies, appliesBefore, r2.eng.serving, r2.eng.usagePersist)
	}
	_ = usagestore.Options{}
}
