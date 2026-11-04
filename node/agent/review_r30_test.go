package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestR30_H1_CanceledBootContextSkipsRecoverAndAdmitsServing proves H-1
// (fail-closed violation):
//  1. In recoverOnce (node/agent/updaterecover.go:36-40), `if e.runCtx != nil &&
//     e.runCtx.Err() != nil { return }` executes BEFORE `e.recoverBlocked = true`
//     at line 40, leaving `e.recovered == false` and `e.recoverBlocked == false`.
//  2. In bootWithBudget (node/agent/engine.go:471), `e.runCtx = caller` pins
//     `e.runCtx` to Boot's caller context without clearing it when Boot returns,
//     and Step never updates `e.runCtx`.
//
// As a result, if Boot is called with an already-canceled context (which still
// completes its detached bootLocal baseline) and Step(context.Background()) is
// then called, Step also returns early in recoverOnce with `e.recoverBlocked ==
// false` and `e.recovered == false`, admitting member forwarding (`e.serving ==
// true`) without ever invoking `fvpn-updater recover` (even when UpdaterConfig
// is empty, which must fail closed with recovery_required).
func TestR30_H1_CanceledBootContextSkipsRecoverAndAdmitsServing(t *testing.T) {
	// TODO(review r30): H-1
	r := newRig(t)
	serveBoundGeneration(t, r)
	cfg, boot, log := r.eng.cfg, r.eng.bootID, r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	// Configure UpdaterRecover with a missing UpdaterConfig: any real execution
	// of recoverOnce must fail closed with recovery_required and never serve.
	cfg.UpdaterRecover = "/bin/true"
	cfg.UpdaterConfig = ""
	cand, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, Log: log, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()

	bootCtx, cancelBoot := context.WithCancel(context.Background())
	cancelBoot()
	_ = cand.Boot(bootCtx)

	if err := cand.Step(context.Background()); err != nil {
		t.Fatalf("Step after canceled Boot: %v", err)
	}
	if cand.serving || !cand.recoverBlocked || !slices.Contains(cand.decision.Reasons, api.ForwardingBlockReasonRecoveryRequired) {
		t.Fatalf("Step admitted serving without running updater recover: serving=%v recoverBlocked=%v recovered=%v reasons=%v",
			cand.serving, cand.recoverBlocked, cand.recovered, cand.decision.Reasons)
	}
}

// TestR30_H1_PerCallBootContextWedgesStepRecovery proves H-1 (permanent
// recovery wedge across Boot -> Step):
// When a caller passes a per-call timeout/cancelable context to Boot(bootCtx)
// while update.lock is busy and cancels bootCtx after Boot returns, e.runCtx
// remains pinned to the canceled bootCtx. Every subsequent Step(context.Background())
// returns early at updaterecover.go:37 (`if e.runCtx != nil && e.runCtx.Err() != nil`)
// without retrying `fvpn-updater recover`, wedging `recovery_required` forever
// even after update.lock is released.
func TestR30_H1_PerCallBootContextWedgesStepRecovery(t *testing.T) {
	// TODO(review r30): H-1
	r := newRig(t)
	serveBoundGeneration(t, r)
	marker := filepath.Join(t.TempDir(), "recovered")
	f49RecoverCommand(t, r, fmt.Sprintf("if [ -f %q ]; then exit 0; fi\nexit 3", marker))
	cfg, boot, log := r.eng.cfg, r.eng.bootID, r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	cand, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, Log: log, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()

	bootCtx, cancelBoot := context.WithTimeout(context.Background(), 5*time.Second)
	if err := cand.Boot(bootCtx); err != nil {
		cancelBoot()
		t.Fatalf("Boot: %v", err)
	}
	// Caller finishes its Boot call and cancels the per-call bootCtx before
	// entering its Step loop with a fresh context.
	cancelBoot()

	if !cand.recoverBlocked || cand.recovered {
		t.Fatalf("after busy Boot: recoverBlocked=%v recovered=%v, want true/false", cand.recoverBlocked, cand.recovered)
	}

	// Now the update lock is free (recover script exits 0).
	if err := os.WriteFile(marker, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cand.Step(context.Background()); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if cand.recoverBlocked || !cand.recovered || !cand.serving {
		t.Fatalf("Step remained wedged in recovery_required after per-call Boot context cancel: recoverBlocked=%v recovered=%v serving=%v reasons=%v",
			cand.recoverBlocked, cand.recovered, cand.serving, cand.decision.Reasons)
	}
}

// TestR30_M2_PreEnforceLeaseAttemptBlocksPostRecoveryRenewal proves M-2:
// During Step (node/agent/engine.go:742-766, 789), syncLease runs BEFORE
// syncDesired/enforce and sets `e.leaseAttempted = true`. When syncDesired
// receives a new ConfigGeneration (15) while that pre-enforce lease request saw
// a transient failure (or still returned generation 14 before the generation
// change was applied), enforce completes closed recovery for generation 15 and
// updates `e.appliedGen = 15`. However, renewAppliedLease (line 789) is
// suppressed by `if e.leaseAttempted`, leaving `e.lease` at generation 14
// (`Reasons = [policy_missing]`) for the rest of the step so noteRecoveryReport
// refuses to arm D-157's early recovery renewal on that recovery step.
func TestR30_M2_PreEnforceLeaseAttemptBlocksPostRecoveryRenewal(t *testing.T) {
	// TODO(review r30): M-2
	r, p := f50Rig(t)

	// Enter grant-less state at generation 14 and record a retained Apply failure.
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	p.foreignTC = true
	stepOK(t, r)
	if r.eng.applyError == nil || !r.eng.leaseGrantless() {
		t.Fatalf("precondition failed: applyError=%v leaseGrantless=%v", r.eng.applyError, r.eng.leaseGrantless())
	}
	p.foreignTC = false

	// Advance to the grant-less renewal slot and publish generation 15; only
	// the single pre-enforce lease call fails transiently (FailLeases(1)), while
	// central is healthy by the time enforce finishes and calls renewAppliedLease.
	r.clock.Advance(r.eng.provisionalRetryAt - r.clock.Mono())
	r.srv.SetFixture(generationFixture(15, oneBinding(r.clock.Wall())))
	r.srv.FailLeases(1)

	stepOK(t, r)
	if r.eng.appliedGen != 15 || !r.eng.closedRecoveryHeld {
		t.Fatalf("after recovery step: appliedGen=%d closedRecoveryHeld=%v, want 15/true", r.eng.appliedGen, r.eng.closedRecoveryHeld)
	}
	if !r.eng.earlyRecoveryRenewal() {
		t.Fatalf("D-157 earlyRecoveryRenewal not armed on closed recovery step because pre-enforce syncLease consumed leaseAttempted and suppressed renewAppliedLease (lease gen=%d, appliedGen=%d, reasons=%v)",
			r.eng.lease.ConfigGeneration, r.eng.appliedGen, r.eng.decision.Reasons)
	}
}

// TestR30_M3_PendingOutboxDedupSkipsNewClosedRecoveryReport proves M-3:
// In observe (node/agent/observe.go:38-46), `if state == e.lastObservationState`
// returns nil whenever `len(items) > 0` in the outbox, even when a new closed
// recovery was just armed (`e.closedRecoveryRenewal.armed && reportID == uuid.Nil`).
// If an earlier report with the same clean observationState is still pending in
// the outbox and an intermediate failure was not recorded in lastObservationState
// (e.g. while outboxFull was true), observe() skips queuing the new recovery's
// report and leaves `closedRecoveryRenewal.reportID == uuid.Nil`, violating
// engine.go:1384 ("each recovery needs its own report, even at the same generation").
func TestR30_M3_PendingOutboxDedupSkipsNewClosedRecoveryReport(t *testing.T) {
	// TODO(review r30): M-3
	r, p := f50Rig(t)
	r.srv.SetObservationRetryAfter(1, 5)
	f50Recovery(t, r, p, 14)

	firstReportID := r.eng.closedRecoveryRenewal.reportID
	if firstReportID == uuid.Nil {
		t.Fatal("closed recovery #1 did not bind a reportID")
	}

	// Simulate an intermediate withdrawal + failure whose observation was
	// suppressed (for example while outboxFull was true), leaving
	// lastObservationState at the clean awaiting-grants state from recovery #1
	// while recovery #1's report is still pending in store.Pending.
	if err := r.eng.baseline(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.eng.noteApply(errors.New("unrecorded intermediate failure"))

	// Now central accepts observations again and enforce runs closed recovery #2
	// at the same generation 14.
	r.srv.SetObservationRetryAfter(0, 0)
	r.clock.Advance(time.Second)
	if err := r.eng.enforce(context.Background(), r.eng.decision); err != nil {
		t.Fatal(err)
	}
	if r.eng.closedRecoveryHeld && r.eng.applyError == nil {
		r.eng.decision = Decide(r.eng.input())
	}
	if err := r.eng.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.closedRecoveryRenewal.reportID == uuid.Nil || r.eng.closedRecoveryRenewal.reportID == firstReportID {
		t.Fatalf("closed recovery #2 did not queue its own report: reportID=%v (first=%v)",
			r.eng.closedRecoveryRenewal.reportID, firstReportID)
	}
}

// TestR30_D155_D156_D157_HeldInvariants verifies the core invariants that held
// across D-155, D-156, and D-157 on normal and negative paths.
func TestR30_D155_D156_D157_HeldInvariants(t *testing.T) {
	t.Run("D156_second_startup_over_boot_adopting_downgrades_to_unverified", func(t *testing.T) {
		r := newRig(t)
		boot := r.eng.bootID
		prior := watchdog.Beat{
			BootID:        boot,
			Mono:          r.clock.Mono() - 2*time.Second,
			Wall:          r.clock.Wall().Add(-2 * time.Second).UTC(),
			PolicyExpires: r.clock.Mono() + 5*time.Minute,
			BaselineOK:    true,
			Serving:       true,
			Generation:    3,
			Reason:        "serving",
		}
		if err := r.eng.startupHeartbeat(context.Background(), prior); err != nil {
			t.Fatal(err)
		}
		b1, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath)
		if err != nil {
			t.Fatal(err)
		}
		if b1.Reason != watchdog.AdoptionPending || b1.Serving || b1.BaselineOK || b1.Mono != prior.Mono || b1.PolicyExpires != prior.PolicyExpires {
			t.Fatalf("first startupHeartbeat = %+v, want boot_adopting with preserved Mono/PolicyExpires", b1)
		}
		// A second startup before adoption completes sees b1 (Serving=false,
		// BaselineOK=false) and must NOT preserve boot_adopting or renew liveness.
		r.clock.Advance(3 * time.Second)
		if err := r.eng.startupHeartbeat(context.Background(), b1); err != nil {
			t.Fatal(err)
		}
		b2, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath)
		if err != nil {
			t.Fatal(err)
		}
		if b2.Reason != "boot_unverified" || b2.BaselineOK || b2.Serving || b2.PolicyExpires != 0 {
			t.Fatalf("second startupHeartbeat over boot_adopting = %+v, want boot_unverified with zero deadline", b2)
		}
	})

	t.Run("D157_baseline_disarms_early_recovery_renewal", func(t *testing.T) {
		r, p := f50Rig(t)
		f50Recovery(t, r, p, 14)
		if !r.eng.earlyRecoveryRenewal() {
			t.Fatal("expected earlyRecoveryRenewal armed after f50Recovery")
		}
		if err := r.eng.baseline(context.Background()); err != nil {
			t.Fatal(err)
		}
		if r.eng.closedRecoveryHeld || r.eng.closedRecoveryRenewal.armed || r.eng.closedRecoveryRenewal.reportID != uuid.Nil || r.eng.earlyRecoveryRenewal() {
			t.Fatalf("baseline did not disarm closedRecoveryRenewal: %+v", r.eng.closedRecoveryRenewal)
		}
	})
}
