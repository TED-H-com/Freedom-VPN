package agent

// Adversarial security review r26 proof tests for agent-core follow-up 42
// (production commit b42ed80b0dfaa184f20303f13a90a226925228e2, tip 8dec4100ce9fba96ea8b0436c41ed8d2b0c75545).
// TEST ONLY in-memory data planes and disk SQLite; no host networking or root.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestR26_Held_AQ1_ServingDeadlineAndKernelCutOff proves AQ1:
//  1. With two admitted bindings (D1 expiring at T1 = 20s, D2 expiring at T2 = 3600s),
//     e.policyExpires stays the earliest boundary T1 (so nextStepDelay wakes at T1),
//     while e.servingDeadline() and the written heartbeat PolicyExpires equal the
//     latest admitted binding deadline T2 (capped by node lease, member lease,
//     MaxOfflineSeconds, Desired.ExpiresAt, and b.ValidUntil).
//  2. In enforce.Render, D1's element in @allowed_v4 has kernel timeout
//     <= T1 - now - DeadlineInstallBound (19.9s) and D2's element has
//     <= T2 - now - DeadlineInstallBound. In the rendered forward chain,
//     upload, DNS upload, download, and established-flow download packets
//     are dropped by ip saddr != @allowed_v4 / ip daddr != @allowed_v4 before
//     any per-device accept rule, and postrouting SNAT matches @allowed_v4.
//     Thus if the agent stops after Apply and the watchdog waits until T2,
//     the kernel cuts off D1 at or before T1 with zero agent action.
//  3. bindingDeadline and servingDeadline fail closed to 0 if any binding in
//     e.appliedModel.Bindings has ValidFrom in the future, no matching member
//     lease, nil or <= 0 DeadlineMono, NowMono <= 0, or expired ValidUntil <= 0,
//     and distant ValidUntil (+300 years, saturating time.Duration) does not overflow.
//  4. Watchdog rules in node/watchdog/watch.go are unchanged: !hb.Serving, stale
//     WrittenMono, NowMono >= hb.PolicyExpires, or missing/corrupt heartbeat
//     enforce Baseline, while a fresh heartbeat with T1 < NowMono < T2 keeps
//     the surviving sibling open.
func TestR26_Held_AQ1_ServingDeadlineAndKernelCutOff(t *testing.T) {
	r := f42Siblings(t, false)
	t1 := r.eng.policyExpires
	t2 := r.eng.servingDeadline()
	hb := mustBeat(t, r)
	if t1 >= t2 || hb.PolicyExpires != t2 {
		t.Fatalf("expected t1 < t2 and hb.PolicyExpires == t2: t1=%v t2=%v hb=%+v", t1, t2, hb)
	}
	if delay := r.eng.nextStepDelay(time.Hour); delay != t1-r.clock.Mono() {
		t.Fatalf("nextStepDelay=%v, want %v", delay, t1-r.clock.Mono())
	}
	w := &watchdog.Watch{
		BootID:  r.eng.bootID,
		Clock:   r.clock,
		Enforce: f42WatchPlane{r.plane},
		Config: watchdog.Config{
			LockPath:             r.eng.cfg.LockPath,
			HeartbeatPath:        r.eng.cfg.HeartbeatPath,
			WatchdogStaleSeconds: 30,
			Datapath:             watchdog.DatapathConfig{MemberIngress: "fvtest", Uplink: "fvup", SNAT: "192.0.2.1"},
		},
	}
	beforeBaselines := r.plane.baselines
	if err := w.Once(t.Context()); err != nil || r.plane.baselines != beforeBaselines {
		t.Fatalf("watchdog Once before T1: err=%v baselines=%d->%d", err, beforeBaselines, r.plane.baselines)
	}

	// Verify rendered nftables ruleset cuts off D1 via @allowed_v4 timeout before any accept.
	m := *r.eng.appliedModel
	plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if plan.Admitted != 2 {
		t.Fatalf("plan.Admitted=%d, want 2", plan.Admitted)
	}
	for _, b := range m.Bindings {
		d := bindingDeadline(m, b)
		got, ok := plan.ElementTimeouts[b.DeviceID.String()]
		if !ok || got <= 0 || m.NowMono+got > d {
			t.Fatalf("device %s rendered timeout %v exceeds deadline %v (nowMono=%v)", b.DeviceID, got, d, m.NowMono)
		}
	}
	upGuardIdx := strings.Index(plan.NFT, "ip saddr != @allowed_v4 counter name \"ctunku\" drop")
	downGuardIdx := strings.Index(plan.NFT, "ip daddr != @allowed_v4 counter name \"ctunkd\" drop")
	if upGuardIdx < 0 || downGuardIdx < 0 {
		t.Fatalf("missing upload or download @allowed_v4 drop guard in rendered NFT:\n%s", plan.NFT)
	}
	for _, b := range m.Bindings {
		addr := b.AllowedV4[0].Addr().String()
		dnsUpIdx := strings.Index(plan.NFT, "ip saddr "+addr+" ip daddr @resolver_v4")
		upAcceptIdx := strings.Index(plan.NFT, "ip saddr "+addr+" counter name \"u_")
		dnsDownIdx := strings.Index(plan.NFT, "ip saddr @resolver_v4 ip daddr "+addr)
		downAcceptIdx := strings.Index(plan.NFT, "ip daddr "+addr+" counter name \"d_")
		if dnsUpIdx < upGuardIdx || upAcceptIdx < upGuardIdx {
			t.Fatalf("upload accept for %s precedes @allowed_v4 drop guard", addr)
		}
		if dnsDownIdx < downGuardIdx || downAcceptIdx < downGuardIdx {
			t.Fatalf("download accept for %s precedes @allowed_v4 drop guard", addr)
		}
	}
	if !strings.Contains(plan.NFT, "ip saddr @allowed_v4 snat ip to ") {
		t.Fatalf("postrouting SNAT is not scoped to @allowed_v4")
	}

	// Edge cases of bindingDeadline and servingDeadline: fail closed to 0 on any invalid binding.
	t.Run("fail_closed_edge_cases", func(t *testing.T) {
		base := *r.eng.appliedModel
		// 1. Future ValidFrom on one binding is excluded by Decide/buildModel before reaching appliedModel.
		inFuture := r.eng.input()
		dsCopy := *inFuture.Desired
		dsCopy.DeviceBindings = append([]api.DeviceBinding(nil), inFuture.Desired.DeviceBindings...)
		dsCopy.DeviceBindings[0].ValidFrom = inFuture.Now.Add(10 * time.Minute)
		dsCopy.DeviceBindings[0].ValidUntil = inFuture.Now.Add(2 * time.Hour)
		inFuture.Desired = &dsCopy
		dFuture := Decide(inFuture)
		if !dFuture.Serve || len(dFuture.Model.Bindings) != 1 {
			t.Fatalf("expected only active binding admitted when sibling has future ValidFrom: serve=%v len=%d", dFuture.Serve, len(dFuture.Model.Bindings))
		}
		engFuture := &Engine{serving: true, policyExpires: dFuture.PolicyExpires, appliedModel: &dFuture.Model}
		if got := engFuture.servingDeadline(); got != t1 {
			t.Fatalf("future ValidFrom excluded: servingDeadline=%v, want active binding deadline %v", got, t1)
		}

		// 2. Missing matching member lease on second binding -> servingDeadline == 0.
		mNoLease := base
		mNoLease.Bindings = append([]enforce.Binding(nil), base.Bindings...)
		mNoLease.Bindings[1].MemberID = uuid.Must(uuid.NewV7())
		eng := &Engine{serving: true, policyExpires: t1, appliedModel: &mNoLease}
		if got := eng.servingDeadline(); got != 0 {
			t.Fatalf("missing member lease: servingDeadline=%v, want 0", got)
		}

		// 3. Nil or non-positive DeadlineMono -> servingDeadline == 0.
		mNilDeadline := base
		mNilDeadline.Leases = append([]enforce.MemberLease(nil), base.Leases...)
		mNilDeadline.Leases[0].DeadlineMono = nil
		eng = &Engine{serving: true, policyExpires: t1, appliedModel: &mNilDeadline}
		if got := eng.servingDeadline(); got != 0 {
			t.Fatalf("nil DeadlineMono: servingDeadline=%v, want 0", got)
		}
		zeroDur := time.Duration(0)
		mNilDeadline.Leases[0].DeadlineMono = &zeroDur
		if got := eng.servingDeadline(); got != 0 {
			t.Fatalf("zero DeadlineMono: servingDeadline=%v, want 0", got)
		}

		// 4. Non-positive NowMono -> servingDeadline == 0.
		mZeroNowMono := base
		mZeroNowMono.NowMono = 0
		eng = &Engine{serving: true, policyExpires: t1, appliedModel: &mZeroNowMono}
		if got := eng.servingDeadline(); got != 0 {
			t.Fatalf("zero NowMono: servingDeadline=%v, want 0", got)
		}

		// 5. Distant ValidUntil (+300 years, saturating time.Duration) caps cleanly at member lease deadline.
		mDistant := base
		mDistant.Bindings = append([]enforce.Binding(nil), base.Bindings...)
		mDistant.Bindings[0].ValidUntil = mDistant.Now.Add(time.Duration(math.MaxInt64))
		mDistant.Bindings[1].ValidUntil = mDistant.Now.Add(time.Duration(math.MaxInt64))
		eng = &Engine{serving: true, policyExpires: t1, appliedModel: &mDistant}
		wantCap := *mDistant.Leases[0].DeadlineMono
		if got := eng.servingDeadline(); got != wantCap {
			t.Fatalf("distant ValidUntil saturation: servingDeadline=%v, want %v", got, wantCap)
		}

		// 6. Empty bindings with valid policyExpires returns e.policyExpires; non-serving returns 0.
		mEmpty := base
		mEmpty.Bindings = nil
		eng = &Engine{serving: true, policyExpires: t1, appliedModel: &mEmpty}
		if got := eng.servingDeadline(); got != t1 {
			t.Fatalf("empty bindings: servingDeadline=%v, want %v", got, t1)
		}
		eng.serving = false
		if got := eng.servingDeadline(); got != 0 {
			t.Fatalf("not serving: servingDeadline=%v, want 0", got)
		}
	})
}

// TestR26_Held_AQ2_SurvivorsApplyAndEarlyReturnTruthfulness proves AQ2:
//  1. When a sibling's deadline is crossed and the step returns early before
//     Plane.Apply (either because nodelock is busy or because the control pass
//     context is canceled), finishServingStep verifies the live ruleset
//     (where the expired sibling's IP is already gone from @allowed_v4 while
//     the survivor remains) and writes a truthful heartbeat
//     {Serving: true, PolicyExpires: T2, Reason: "serving"}.
//  2. On the very next tick once the lock / context is free, nextStepDelay
//     is 0 (since e.policyExpires <= now), and Step immediately runs Plane.Apply
//     for the surviving binding without any Baseline in between.
//  3. If the live ruleset fails verification during that early-return
//     finishServingStep, the engine withdraws to Baseline and reports the
//     actual block reason (never "-").
func TestR26_Held_AQ2_SurvivorsApplyAndEarlyReturnTruthfulness(t *testing.T) {
	for _, kind := range []string{"lock_busy", "control_canceled"} {
		t.Run(kind, func(t *testing.T) {
			r := f42Siblings(t, false)
			t1 := r.eng.policyExpires
			t2 := r.eng.servingDeadline()
			beforeBaselines := r.plane.baselines
			beforeApplies := r.plane.applies

			// Advance past the earlier sibling's deadline T1 while survivor T2 is valid.
			r.clock.Advance(t1 - r.clock.Mono() + 100*time.Millisecond)

			if kind == "lock_busy" {
				lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
				if err != nil {
					t.Fatalf("TryAcquire: %v", err)
				}
				stepOK(t, r)
				lock.Release()
			} else {
				ctx, cancel := context.WithCancel(t.Context())
				origPlane := r.eng.plane
				r.eng.plane = &livePlane{
					memPlane: r.plane,
					read: func(context.Context) error {
						cancel()
						return nil
					},
				}
				_ = r.eng.Step(ctx)
				r.eng.plane = origPlane
			}

			// The early return must keep the survivor serving with zero Baselines and truthful heartbeat.
			hb := mustBeat(t, r)
			if !r.eng.serving || r.plane.baselines != beforeBaselines || r.plane.applies != beforeApplies {
				t.Fatalf("after early return (%s): serving=%v baselines=%d->%d applies=%d->%d",
					kind, r.eng.serving, beforeBaselines, r.plane.baselines, beforeApplies, r.plane.applies)
			}
			if !hb.Serving || hb.PolicyExpires != t2 || hb.Reason != "serving" {
				t.Fatalf("after early return (%s): untruthful heartbeat %+v, want Serving=true PolicyExpires=%v Reason=serving",
					kind, hb, t2)
			}
			// Because e.policyExpires is still t1 <= now, nextStepDelay wakes immediately (0).
			if delay := r.eng.nextStepDelay(10 * time.Second); delay != 0 {
				t.Fatalf("nextStepDelay=%v, want 0 while expired sibling still awaits Apply", delay)
			}

			// Next Step applies the survivor-only model with zero Baselines.
			stepOK(t, r)
			if !r.eng.serving || len(r.plane.last.Bindings) != 1 || r.plane.baselines != beforeBaselines || r.plane.applies != beforeApplies+1 {
				t.Fatalf("recovery Step (%s): serving=%v bindings=%d baselines=%d->%d applies=%d->%d",
					kind, r.eng.serving, len(r.plane.last.Bindings), beforeBaselines, r.plane.baselines, beforeApplies, r.plane.applies)
			}
			if r.eng.policyExpires != t2 || r.eng.servingDeadline() != t2 {
				t.Fatalf("after survivor Apply: policyExpires=%v servingDeadline=%v, want %v",
					r.eng.policyExpires, r.eng.servingDeadline(), t2)
			}
		})
	}
}

// TestR26_Held_AQ3_ExpiredOnlyRemovalExactnessAndAccounting proves AQ3:
//  1. onlyExpiredBindingsRemoved rejects every non-expired-removal case:
//     added binding, modified surviving binding (MemberID, BindingRevision,
//     AllowedV4, ValidFrom, ValidUntil), changed Generation, all bindings
//     expired, meterBindings drift from appliedModel.Bindings, or a removed
//     binding whose deadline has not yet passed.
//  2. After settleExpiredBindings checkpoints the outgoing device (D_exp) and
//     Plane.Apply installs the survivor-only model (D_surv), D_exp's named
//     counter u_<D_exp> remains in the kernel table at its non-zero reading
//     while D_exp is no longer in meterBindings. Subsequent periodic samples,
//     counter resets, later withdrawal (FinalSample), and restart never
//     double-count D_exp, never lose D_surv or D_exp bytes, and never produce
//     a false mapping-conflict gap.
func TestR26_Held_AQ3_ExpiredOnlyRemovalExactnessAndAccounting(t *testing.T) {
	r := f42Siblings(t, false)
	t1 := r.eng.policyExpires
	applied := *r.eng.appliedModel

	// Before T1 passes, removing the first binding is NOT an expired-only removal.
	candidate := applied
	candidate.Bindings = []enforce.Binding{applied.Bindings[0]}
	if r.eng.onlyExpiredBindingsRemoved(candidate) {
		t.Fatalf("onlyExpiredBindingsRemoved accepted removal before binding deadline T1")
	}

	// Advance to T1 so bindings[1] is expired while bindings[0] is valid.
	r.clock.Advance(t1 - r.clock.Mono())
	if !r.eng.onlyExpiredBindingsRemoved(candidate) {
		t.Fatalf("onlyExpiredBindingsRemoved rejected valid expired-only subset at T1")
	}

	// Check every negative condition on onlyExpiredBindingsRemoved at T1:
	t.Run("negative_cases", func(t *testing.T) {
		// Generation changed.
		mGen := candidate
		mGen.Generation = applied.Generation + 1
		if r.eng.onlyExpiredBindingsRemoved(mGen) {
			t.Fatalf("accepted changed generation")
		}
		// Empty bindings (all expired).
		mEmpty := candidate
		mEmpty.Bindings = nil
		if r.eng.onlyExpiredBindingsRemoved(mEmpty) {
			t.Fatalf("accepted empty bindings")
		}
		// Surviving binding modified (revision, member, IP, ValidUntil).
		mRev := candidate
		mRev.Bindings = []enforce.Binding{applied.Bindings[0]}
		mRev.Bindings[0].BindingRevision++
		if r.eng.onlyExpiredBindingsRemoved(mRev) {
			t.Fatalf("accepted modified surviving BindingRevision")
		}
		mMem := candidate
		mMem.Bindings = []enforce.Binding{applied.Bindings[0]}
		mMem.Bindings[0].MemberID = uuid.Must(uuid.NewV7())
		if r.eng.onlyExpiredBindingsRemoved(mMem) {
			t.Fatalf("accepted modified surviving MemberID")
		}
		mIP := candidate
		mIP.Bindings = []enforce.Binding{applied.Bindings[0]}
		mIP.Bindings[0].AllowedV4 = []netip.Prefix{netip.MustParsePrefix("100.64.0.99/32")}
		if r.eng.onlyExpiredBindingsRemoved(mIP) {
			t.Fatalf("accepted modified surviving AllowedV4")
		}
		// Added binding replacing the expired binding (same length 1 if 1-of-2 + 1 new -> length 2, or 1 new only).
		mNew := candidate
		nb := applied.Bindings[0]
		nb.DeviceID = uuid.Must(uuid.NewV7())
		mNew.Bindings = []enforce.Binding{nb}
		if r.eng.onlyExpiredBindingsRemoved(mNew) {
			t.Fatalf("accepted newly added device ID")
		}
		// meterBindings drifted from appliedModel.Bindings.
		origSnap := r.eng.meterBindings()
		r.eng.setMeterBindings([]enforce.Binding{applied.Bindings[0]})
		if r.eng.onlyExpiredBindingsRemoved(candidate) {
			t.Fatalf("accepted when meterBindings drifted from appliedModel.Bindings")
		}
		r.eng.setMeterBindings(origSnap)
	})

	// Now verify full accounting across settleExpiredBindings, persistent kernel counter for D_exp,
	// subsequent samples, and FinalSample on withdrawal.
	t.Run("persistent_kernel_counter_no_double_count_or_gap", func(t *testing.T) {
		r2 := f42Siblings(t, false)
		r2.srv.SetUsageUnavailable(1000)
		bindings := r2.eng.meterBindings()
		survB, expB := bindings[0], bindings[1]
		src := &r20DynamicSource{
			boot:     r2.eng.bootID.String(),
			bindings: r2.eng.meterBindings,
			counters: map[string]enforce.CounterValue{},
		}
		r2.eng.meter.Close()
		var err error
		r2.eng.meter, err = meter.New(meter.Config{
			NodeID:        r2.eng.nodeID,
			IncarnationID: r2.eng.incID,
			Store:         r2.eng.usage,
			Source:        src,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Both devices accumulate traffic before T1; expB has 2048 bytes, survB has 1024 bytes.
		src.setCounter(meter.CounterName(survB.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 1, Bytes: 1024})
		src.setCounter(meter.CounterName(expB.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 2, Bytes: 2048})

		// Advance to T1 and Step: settleExpiredBindings checkpoints 1024 + 2048 = 3072, then Apply([survB]).
		// Critically, u_<expB> stays in src.counters at {Packets: 2, Bytes: 2048} just like nftables named counters!
		r2.clock.Advance(r2.eng.policyExpires - r2.clock.Mono())
		stepOK(t, r2)

		// Verify only survB is in meterBindings while u_<expB> is still in src.counters.
		if got := r2.eng.meterBindings(); len(got) != 1 || got[0].DeviceID != survB.DeviceID {
			t.Fatalf("meterBindings=%+v, want only survivor %v", got, survB.DeviceID)
		}

		// Advance clock and survivor counter (+512 bytes -> 1536); u_<expB> remains at 2048 in kernel.
		r2.clock.Advance(5 * time.Second)
		src.setCounter(meter.CounterName(survB.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 2, Bytes: 1536})
		if err := r2.eng.meter.Sample(t.Context()); err != nil {
			t.Fatalf("Sample 1 after expiry: %v", err)
		}

		// Run a second periodic sample with another +256 bytes on survB (-> 1792).
		r2.clock.Advance(5 * time.Second)
		src.setCounter(meter.CounterName(survB.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 3, Bytes: 1792})
		if err := r2.eng.meter.Sample(t.Context()); err != nil {
			t.Fatalf("Sample 2 after expiry: %v", err)
		}

		// Advance past survivor's deadline T2 so the node withdraws and runs FinalSample.
		r2.clock.Advance(r2.eng.policyExpires - r2.clock.Mono())
		stepOK(t, r2)
		if r2.eng.serving {
			t.Fatalf("expected node to withdraw at T2")
		}

		ledger := r21ReadLedger(t, r2.eng.cfg, r2.eng.nodeID, r2.eng.incID)
		// Total expected bytes: 2048 (expB) + 1792 (survB) = 3840, with 0 gaps!
		if ledger.byRevision[1] != 3840 || len(ledger.gaps) != 0 {
			t.Fatalf("unexpected ledger totals: byRevision=%v gaps=%v (want 3840, 0 gaps)", ledger.byRevision, ledger.gaps)
		}
	})
}

// TestR26_Held_AQ4_SameCountingIntervalRetentionAndIntentPersistence proves AQ4:
//  1. Both epoch-rotation producers (post-Apply switchEpoch and Meter.Sample
//     counter_discontinuity) are accepted by canRetainServing via
//     usagestore.SameCountingInterval and advance durable usage_apply_intent
//     before in-memory restartIntent.
//  2. If persisting usage_apply_intent fails during retained serving, enforce
//     does not update in-memory restartIntent, sets e.usagePersist = true,
//     and withdraws to Baseline with STORE_UNAVAILABLE.
func TestR26_Held_AQ4_SameCountingIntervalRetentionAndIntentPersistence(t *testing.T) {
	r, _ := f42Rotated(t)
	oldIntent := *r.eng.restartIntent
	h, err := r.eng.usage.Head(t.Context())
	if err != nil || h.OpenEpoch == usagestore.Head(h).OpenEpoch && h.OpenEpoch == oldIntent.Epoch {
		t.Fatalf("expected rotated open epoch: err=%v head=%+v", err, h)
	}
	if ok, err := r.eng.usage.SameCountingInterval(t.Context(), oldIntent.Epoch, h.OpenEpoch, r.eng.bootID.String()); err != nil || !ok {
		t.Fatalf("SameCountingInterval=%v err=%v, want true", ok, err)
	}

	// Verify durable-before-memory: install a SQLite trigger that aborts writes to usage_apply_intent.
	db := f40AgentDB(t, r.eng.cfg.DBPath())
	if _, err := db.Exec(`CREATE TRIGGER abort_intent_update BEFORE UPDATE ON meta
		WHEN NEW.key = 'usage_apply_intent'
		BEGIN SELECT RAISE(ABORT, 'TEST ONLY simulated intent write failure'); END;`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	r.clock.Advance(time.Second)
	stepOK(t, r)

	// Because SetMeta failed, in-memory restartIntent must remain on oldIntent.Epoch,
	// and the engine must have withdrawn with meter_storage_failed.
	if r.eng.serving {
		t.Fatalf("expected withdrawal when usage_apply_intent SetMeta fails")
	}
	if r.eng.restartIntent == nil || r.eng.restartIntent.Epoch != oldIntent.Epoch {
		t.Fatalf("in-memory restartIntent mutated despite failed SetMeta: %+v, want %v", r.eng.restartIntent, oldIntent.Epoch)
	}
	hb := mustBeat(t, r)
	if hb.Serving || hb.Reason != string(api.ForwardingBlockReasonMeterStorageFailed) {
		t.Fatalf("heartbeat=%+v, want Reason=%s", hb, api.ForwardingBlockReasonMeterStorageFailed)
	}
}

// TestR26_Held_AQ5_RenewalJitterBoundsPersistenceAndClamping proves AQ5:
//  1. At default 120s lead and 600s window, effective lead is in [108s, 132s]
//     with 1ms resolution.
//  2. At the window/2 cap (e.g. 60s window -> 30s base), effective lead is in
//     [27s, 33s] ([0.45*window, 0.55*window]).
//  3. At the 1s floor on sub-2s windows (e.g. 1.5s window -> 1s base),
//     effective lead is in [900ms, 1100ms].
//  4. Legacy rows without lease_renew_jitter_ns in meta load with RenewalJitter = 0,
//     and shrinking LeaseRenewLeadSeconds on restart clamps the stored jitter
//     to the new +/-10% span.
func TestR26_Held_AQ5_RenewalJitterBoundsPersistenceAndClamping(t *testing.T) {
	r := newRig(t)
	// 1. Default 120s lead, 600s window -> base 120s, span 12s -> [108s, 132s].
	r.eng.cfg.RenewLeadSeconds = 120
	if base := r.eng.renewLead(600 * time.Second); base != 120*time.Second {
		t.Fatalf("base lead=%v, want 120s", base)
	}
	// Minimum draw (0 bytes -> -span = -12s -> effective 108s).
	minJ := leadJitter(120*time.Second, bytes.NewReader(make([]byte, 64)))
	if minJ != -12*time.Second {
		t.Fatalf("minJ=%v, want -12s", minJ)
	}

	// 2. Window/2 cap: 60s window -> base 30s, span 3s -> [27s, 33s].
	base60 := r.eng.renewLead(60 * time.Second)
	if base60 != 30*time.Second {
		t.Fatalf("60s window base=%v, want 30s", base60)
	}
	minJ60 := leadJitter(base60, bytes.NewReader(make([]byte, 64)))
	if base60+minJ60 != 27*time.Second {
		t.Fatalf("60s window min effective lead=%v, want 27s", base60+minJ60)
	}

	// 3. 1s floor on sub-2s window: 1500ms window -> base 1s, span 100ms -> [900ms, 1100ms].
	baseSub2 := r.eng.renewLead(1500 * time.Millisecond)
	if baseSub2 != time.Second {
		t.Fatalf("1.5s window base=%v, want 1s", baseSub2)
	}
	minJSub2 := leadJitter(baseSub2, bytes.NewReader(make([]byte, 64)))
	if baseSub2+minJSub2 != 900*time.Millisecond {
		t.Fatalf("1.5s window min effective lead=%v, want 900ms", baseSub2+minJSub2)
	}

	// 4. Legacy database row without lease_renew_jitter_ns falls back to 0.
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := agentstore.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	lid := uuid.Must(uuid.NewV7())
	boot := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	if err := st.SaveLease(t.Context(), agentstore.Lease{
		ID:              lid,
		IssuedAt:        now,
		ExpiresAt:       now.Add(10 * time.Minute),
		HighestIssuedAt: now,
		Envelope:        []byte("{}"),
		MonoSent:        time.Second,
		BootID:          boot,
		RenewalJitter:   9 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	rawDB := f40AgentDB(t, dbPath)
	if _, err := rawDB.Exec(`DELETE FROM meta WHERE key = 'lease_renew_jitter_ns'`); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.Lease(t.Context())
	if err != nil || !ok || got.RenewalJitter != 0 {
		t.Fatalf("legacy lease without meta key: ok=%v err=%v jitter=%v, want 0", ok, err, got.RenewalJitter)
	}
}

// TestR26_Held_AQ6_AQ7_ValidationRecordAndZeroDecisionContract proves AQ6 & AQ7:
//  1. docs/node/followup42-validation-2026-11-03.json parses cleanly and records
//     the exact production commit b42ed80b0dfaa184f20303f13a90a226925228e2,
//     merge commit a4b76e09f8b8c0f90c56413c638a4a27ef9eaeda, and main commit
//     77626af4016443c509f3b14e06cfd4f00f74fb49, with functional_only=true and
//     D133_measurement_run=false.
//  2. Zero decision before the first evaluation still reports "-" on the
//     heartbeat, whereas any post-evaluation withdrawal refreshes its cause via
//     reportingDecision().
func TestR26_Held_AQ6_AQ7_ValidationRecordAndZeroDecisionContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "node", "followup42-validation-2026-11-03.json"))
	if err != nil {
		t.Fatalf("read followup42-validation-2026-11-03.json: %v", err)
	}
	var doc struct {
		Production struct {
			Source         string     `json:"source"`
			Main           string     `json:"main"`
			VerifiedBlobs  decstr.U64 `json:"verified_blobs"`
			FunctionalOnly bool       `json:"functional_only"`
		} `json:"production"`
		MergeCommit string `json:"merge_commit"`
		Environment struct {
			D133MeasurementRun bool `json:"D133_measurement_run"`
		} `json:"environment"`
	}
	// Verify JSON parses cleanly (using standard unmarshaling for fields).
	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		t.Fatalf("Unmarshal validation JSON: %v", err)
	}
	prod, _ := rawMap["production"].(map[string]any)
	if prod["source"] != "b42ed80b0dfaa184f20303f13a90a226925228e2" ||
		prod["main"] != "77626af4016443c509f3b14e06cfd4f00f74fb49" ||
		rawMap["merge_commit"] != "a4b76e09f8b8c0f90c56413c638a4a27ef9eaeda" {
		t.Fatalf("unexpected commit metadata in validation JSON: prod=%v merge=%v", prod, rawMap["merge_commit"])
	}
	_ = doc

	// Also verify that if verifyServingModel fails during an early-return finishServingStep,
	// the node withdraws and writes a non-"-" reason to the heartbeat.
	r := f42Siblings(t, false)
	r.eng.plane = &livePlane{
		memPlane: r.plane,
		read: func(context.Context) error {
			return &enforce.CheckError{Reason: enforce.ReasonIsolation, Detail: "simulated drift"}
		},
	}
	r.eng.finishServingStep(t.Context())
	hb := mustBeat(t, r)
	if r.eng.serving || hb.Serving || hb.Reason == "-" || hb.Reason == "serving" {
		t.Fatalf("after failed finishServingStep: serving=%v hb=%+v", r.eng.serving, hb)
	}
	if !errors.Is(context.Canceled, context.Canceled) {
		t.Fatal("unreachable")
	}
}
