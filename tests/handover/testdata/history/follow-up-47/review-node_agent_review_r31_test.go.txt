package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type r31IsolationQuarantinePlane struct {
	*r27BootMissingInterfacePlane
	failIsolationQuarantine bool
}

func (p *r31IsolationQuarantinePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.failIsolationQuarantine {
		return enforce.ApplyResult{
			Quarantine: &enforce.QuarantineEvidence{
				Quarantined: true,
				Baseline:    enforce.BaselineEvidence{Verified: true},
			},
		}, &enforce.CheckError{Reason: enforce.ReasonIsolation, Detail: "isolation check failed"}
	}
	return p.r27BootMissingInterfacePlane.Apply(ctx, m, prev)
}

// TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox proves M-1:
// In observe (node/agent/observe.go:39), the pending-outbox deduplication
// exception checks only `e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil`.
// It does not check `e.recoveryRenewalEligible()`, nor does `observe` / `noteRecoveryReport`
// guard against a closed recovery whose report cannot bind (for example after an
// isolation error in `rep.Errors`, a failed meter path, or no `awaiting_member_grants`
// such as `e.leaseBehindApplied()`).
// During an observation outage, every due observation at an unchanged state
// bypasses pending-outbox deduplication and enqueues another duplicate report
// until the 32-entry outbox fills (`e.outboxFull = true`), where the pre-r30
// deduplication held a single report. Once `e.outboxFull` is set, `Decide` and
// `recoveryDecision` add `outbox_full` (`d.Serve = false`), tearing down the
// held closed recovery via `baseline` and blocking member forwarding and new
// state-transition observations until `observationRetryAt` expires and pending
// observations drain below 32.
func TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox(t *testing.T) {
	t.Run("isolation_error_in_report_fills_outbox_and_tears_down_recovery", func(t *testing.T) {
		r, basePlane := f50Rig(t)
		p := &r31IsolationQuarantinePlane{r27BootMissingInterfacePlane: basePlane}
		r.eng.plane = p

		// 1. Advance desired state to generation 14 so enforce calls Apply, and
		// fail that Apply with an isolation CheckError and rollback quarantine.
		// recordApplyQuarantine latches `e.containment.Code = ISOLATION_POLICY_INVALID`,
		// which `attachContainment` includes on subsequent observation reports.
		p.failIsolationQuarantine = true
		r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
		advanceToRenewal(r)
		stepOK(t, r)
		if r.eng.applyError == nil || r.eng.containment == nil || r.eng.containment.Code != api.ErrorCodeISOLATIONPOLICYINVALID {
			t.Fatalf("expected isolation containment after failed Apply, got applyError=%v containment=%+v", r.eng.applyError, r.eng.containment)
		}

		// 2. Central enters an observation outage while issuing a grant-less lease
		// for generation 14. The closed-recovery Apply succeeds and arms
		// `closedRecoveryRenewal`, but `noteRecoveryReport` refuses to bind
		// `reportID` because `rep.Errors` carries `ISOLATION_POLICY_INVALID`.
		p.failIsolationQuarantine = false
		r.srv.SetMemberGrantsReady(false)
		r.srv.SetObservationRetryAfter(64, 60)
		advanceToRenewal(r)
		stepOK(t, r)

		if !r.eng.closedRecoveryHeld || !r.eng.closedRecoveryRenewal.armed || r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
			t.Fatalf("expected held armed closed recovery with unbound reportID, got held=%v renewal=%+v",
				r.eng.closedRecoveryHeld, r.eng.closedRecoveryRenewal)
		}
		if !r.eng.recoveryRenewalEligible() {
			t.Fatal("expected recoveryRenewalEligible() == true on grant-less generation 14 recovery")
		}
		state := r.eng.lastObservationState

		obsInterval := time.Duration(r.eng.cfg.ObservationSeconds) * time.Second
		if obsInterval <= 0 {
			obsInterval = 10 * time.Second
		}
		for i := 0; i < 31; i++ {
			r.clock.Advance(obsInterval)
			if err := r.eng.observe(context.Background()); err != nil {
				t.Fatal(err)
			}
		}

		pending, err := r.eng.store.Pending(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		// Run one more Step during the observation outage: `Step` refreshes
		// `e.outboxFull = len(pending) >= 32`, which forces `outbox_full` in
		// `Decide` / `recoveryDecision` and tears down `closedRecoveryHeld`.
		stepOK(t, r)

		// TODO(review r31): M-1 — Pending-outbox deduplication in observe.go:39 must
		// not enqueue 32 duplicate reports of an unchanged state and trip outboxFull
		// when a closed recovery's report cannot bind (isolation error).
		if len(pending) > 1 || r.eng.outboxFull || !r.eng.closedRecoveryHeld || slices.Contains(r.eng.decision.Reasons, api.ForwardingBlockReasonOutboxFull) {
			t.Fatalf("M-1 (isolation_error): unbindable closed recovery bypassed pending-outbox dedup at unchanged state=%q: len(pending)=%d (want 1), outboxFull=%v, closedRecoveryHeld=%v, reasons=%v",
				state, len(pending), r.eng.outboxFull, r.eng.closedRecoveryHeld, r.eng.decision.Reasons)
		}
	})

	t.Run("failed_meter_path_fills_outbox", func(t *testing.T) {
		r, p := f50Rig(t)
		r.srv.SetObservationRetryAfter(64, 60)
		f50Recovery(t, r, p, 14)

		// Simulate a closed recovery whose report cannot bind because the meter
		// path is failed (`e.meterFailed = true`) while `closedRecoveryRenewal`
		// is armed with `reportID == uuid.Nil`.
		r.eng.closedRecoveryRenewal = recoveryRenewal{armed: true}
		r.eng.meterFailed = true
		r.eng.decision = Decide(r.eng.input())
		r.eng.lastObs = time.Time{}
		if err := r.eng.observe(context.Background()); err != nil {
			t.Fatal(err)
		}
		if r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
			t.Fatalf("expected nil reportID when meterFailed is true, got %v", r.eng.closedRecoveryRenewal.reportID)
		}
		beforePending, err := r.eng.store.Pending(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		obsInterval := time.Duration(r.eng.cfg.ObservationSeconds) * time.Second
		if obsInterval <= 0 {
			obsInterval = 10 * time.Second
		}
		for i := 0; i < 32; i++ {
			r.clock.Advance(obsInterval)
			if err := r.eng.observe(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		afterPending, err := r.eng.store.Pending(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		// TODO(review r31): M-1 — Pending-outbox deduplication in observe.go:39 must
		// not enqueue duplicate reports at an unchanged state when meterFailed prevents binding.
		if len(afterPending) > len(beforePending) {
			t.Fatalf("M-1 (failed_meter_path): pending outbox grew from %d to %d at unchanged state when meterFailed prevented binding",
				len(beforePending), len(afterPending))
		}
	})

	t.Run("no_awaiting_member_grants_lease_behind_applied_fills_outbox", func(t *testing.T) {
		r, p := f50Rig(t)
		p.foreignTC = true
		stepOK(t, r)
		p.foreignTC = false

		// Advance desired state to generation 14 while both /leases:renew and
		// /observations fail with 503 during an outage. The closed-recovery
		// Apply for generation 14 succeeds (`appliedGen = 14`), while the lease
		// stays at generation 13 (`leaseBehindApplied() == true`, `Reasons = [policy_missing]`,
		// no `awaiting_member_grants`, `recoveryRenewalEligible() == false`).
		r.srv.SetObservationRetryAfter(64, 60)
		r.srv.FailLeases(2)
		r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
		stepOK(t, r)

		if !r.eng.closedRecoveryHeld || !r.eng.closedRecoveryRenewal.armed || r.eng.closedRecoveryRenewal.reportID != uuid.Nil || r.eng.recoveryRenewalEligible() {
			t.Fatalf("expected armed ineligible closed recovery with leaseBehindApplied, got held=%v renewal=%+v eligible=%v",
				r.eng.closedRecoveryHeld, r.eng.closedRecoveryRenewal, r.eng.recoveryRenewalEligible())
		}
		beforePending, err := r.eng.store.Pending(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		obsInterval := time.Duration(r.eng.cfg.ObservationSeconds) * time.Second
		if obsInterval <= 0 {
			obsInterval = 10 * time.Second
		}
		for i := 0; i < 31; i++ {
			r.clock.Advance(obsInterval)
			if err := r.eng.observe(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		afterPending, err := r.eng.store.Pending(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		// TODO(review r31): M-1 — Pending-outbox deduplication in observe.go:39 must
		// check eligibility and not enqueue duplicate reports at an unchanged state
		// when no `awaiting_member_grants` is present (`!recoveryRenewalEligible()`).
		if len(afterPending) > len(beforePending) {
			t.Fatalf("M-1 (no_awaiting_member_grants): pending outbox grew from %d to %d at unchanged state when !recoveryRenewalEligible()",
				len(beforePending), len(afterPending))
		}
	})
}

// TestR31_Held_H1_RecoverLifetimesAndFailClosedGate verifies that PR #470's H-1
// fix preserves fail-closed recovery gating before, during, and after Boot/Run/Step,
// detaches standalone Boot context cancellation from the recovery child, and
// preserves upload loop lifetime across Run, once mode, and Close.
func TestR31_Held_H1_RecoverLifetimesAndFailClosedGate(t *testing.T) {
	t.Run("input_recover_blocked_follows_config_and_completion", func(t *testing.T) {
		r := newRig(t)
		r.eng.cfg.UpdaterRecover = ""
		r.eng.recovered = false
		r.eng.recoverBlocked = false
		if r.eng.input().RecoverBlocked {
			t.Fatal("RecoverBlocked must be false when UpdaterRecover is unconfigured")
		}

		r.eng.cfg.UpdaterRecover = "/bin/true"
		r.eng.cfg.UpdaterConfig = filepath.Join(r.dir, "updater.json")
		if !r.eng.input().RecoverBlocked {
			t.Fatal("RecoverBlocked must be true when UpdaterRecover is configured and !recovered")
		}
		r.eng.recoverOnce(context.Background())
		if !r.eng.recovered || r.eng.recoverBlocked || r.eng.input().RecoverBlocked {
			t.Fatalf("after /bin/true recoverOnce: recovered=%v recoverBlocked=%v input.RecoverBlocked=%v",
				r.eng.recovered, r.eng.recoverBlocked, r.eng.input().RecoverBlocked)
		}
	})

	t.Run("run_stop_sets_recover_blocked_without_spawning_child", func(t *testing.T) {
		r := newRig(t)
		serveBoundGeneration(t, r)
		marker := filepath.Join(t.TempDir(), "recover.ran")
		script := filepath.Join(t.TempDir(), "fake-updater.sh")
		body := "#!/bin/sh\n: > " + marker + "\nexit 0\n"
		if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg, boot, log := r.eng.cfg, r.eng.bootID, r.eng.log
		if err := r.eng.Close(); err != nil {
			t.Fatal(err)
		}
		cfg.UpdaterRecover = script
		cfg.UpdaterConfig = filepath.Join(r.dir, "updater.json")
		cand, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, Log: log, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
		if err != nil {
			t.Fatal(err)
		}
		defer cand.Close()

		runCtx, cancel := context.WithCancel(t.Context())
		cancel()
		_ = cand.Run(runCtx)

		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatalf("canceled Run spawned updater recover: stat err=%v", err)
		}
		if !cand.recoverBlocked || cand.recovered || !cand.input().RecoverBlocked || !slices.Contains(cand.decision.Reasons, api.ForwardingBlockReasonRecoveryRequired) {
			t.Fatalf("after canceled Run: recoverBlocked=%v recovered=%v input.RecoverBlocked=%v reasons=%v",
				cand.recoverBlocked, cand.recovered, cand.input().RecoverBlocked, cand.decision.Reasons)
		}
	})
}

// TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport verifies
// Question 3 and Question 7: when a closed recovery uses both M-2's same-Step
// generation catch-up (clearing leaseAttempted in renewAppliedLease) and M-3's
// pending-outbox deduplication exception, the catch-up request does not consume
// closedRecoveryRenewal, no D-157 early renewal fires before the recovery's own
// bound report is accepted, and at most one early renewal fires inside the D-067 slot.
func TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport(t *testing.T) {
	r, p := f50Rig(t)
	oldLease := *r.eng.lease // generation 13 lease

	// Hold observation #1 in Pending at generation 14's awaiting-grants state.
	r.srv.SetObservationRetryAfter(1, 5)
	f50Recovery(t, r, p, 14)
	firstReportID := r.eng.closedRecoveryRenewal.reportID
	if firstReportID == uuid.Nil || r.eng.closedRecoveryRenewal.accepted {
		t.Fatalf("unexpected recovery #1 state: %+v", r.eng.closedRecoveryRenewal)
	}

	// Simulate an unrecorded withdrawal + failure while the lease is still behind
	// generation 14 (`oldLease` at gen 13) and observations are still pending.
	if err := r.eng.baseline(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.eng.noteApply(errors.New("unrecorded intermediate failure"))
	r.eng.lease = &oldLease
	r.srv.SetObservationRetryAfter(0, 0)
	r.srv.FailLeases(1)
	r.eng.plane = &f51RecoveryPlane{r27BootMissingInterfacePlane: p, afterApply: func() { r.clock.Advance(time.Second) }}
	r.clock.Advance(r.eng.provisionalRetryAt - r.clock.Mono())

	// Step's pre-enforce syncLease fails (FailLeases(1)) with a null ACK
	// (leaseAttemptGen == 0), enforce runs closed recovery #2 at generation 14,
	// renewAppliedLease clears leaseAttempted (M-2) and fetches generation 14's
	// grant-less lease, and observe (M-3 exception at unchanged state) + flush
	// binds and accepts recovery #2's new report.
	beforeRenewals := r.srv.LeaseRenewals()
	stepOK(t, r)
	afterRecoveryStepRenewals := r.srv.LeaseRenewals()
	if afterRecoveryStepRenewals != beforeRenewals+2 {
		t.Fatalf("expected 2 lease requests on recovery step (pre-enforce null-ACK + post-enforce M-2 catch-up), got %d -> %d",
			beforeRenewals, afterRecoveryStepRenewals)
	}
	if !r.eng.earlyRecoveryRenewal() || r.eng.closedRecoveryRenewal.reportID == uuid.Nil || r.eng.closedRecoveryRenewal.reportID == firstReportID {
		t.Fatalf("expected D-157 earlyRecoveryRenewal armed with fresh reportID after M-2 + M-3 interaction, got %+v (first=%v)",
			r.eng.closedRecoveryRenewal, firstReportID)
	}

	// Inside the D-067 slot, the next Step dispatches exactly one D-157 early
	// renewal and consumes closedRecoveryRenewal; further Steps inside the slot
	// send no additional lease requests.
	slot := r.eng.provisionalRetryAt
	stepOK(t, r)
	if r.clock.Mono() >= slot || r.srv.LeaseRenewals() != afterRecoveryStepRenewals+1 || r.eng.closedRecoveryRenewal.armed || r.eng.earlyRecoveryRenewal() {
		t.Fatalf("expected 1 consumed D-157 early renewal inside D-067 slot, got renewals=%d (want %d) renewal=%+v",
			r.srv.LeaseRenewals(), afterRecoveryStepRenewals+1, r.eng.closedRecoveryRenewal)
	}
	stepOK(t, r)
	if r.srv.LeaseRenewals() != afterRecoveryStepRenewals+1 {
		t.Fatalf("second early renewal fired inside same D-067 slot: renewals=%d, want %d",
			r.srv.LeaseRenewals(), afterRecoveryStepRenewals+1)
	}
}
