package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestR29_H1_UpdaterExitCodeDistinguishesLiveUpdateBusy(t *testing.T) {
	// TODO(review r29): H-1
	// Assumes Option B: updater.ExitCode returns ExitBusy (3) when Recover fails
	// with UPDATE_BUSY on a live same-boot update.lock holder, so fvpn-agent can
	// distinguish transient lock contention from terminal errors (1) and
	// ErrMembersBlocked (2) without scraping stderr text.
	w := newWorld(t)
	lock, err := nodelock.AcquireWithBootID(context.Background(), w.layout.Lock, 1, 0, w.node.BootID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	recErr := w.eng.Recover(context.Background())
	var op *OpError
	if !errors.As(recErr, &op) || op.Code != apierr.UpdateBusy || op.Detail != "lock" || !op.Retry {
		t.Fatalf("Recover under held update.lock = %v, want retryable UPDATE_BUSY: lock", recErr)
	}
	if code := ExitCode(recErr); code != 3 {
		t.Fatalf("ExitCode(%v) = %d, want 3 (ExitBusy)", recErr, code)
	}
}

func TestR29_L1_ContendedRecoverChecksUpdateLockBeforeOpeningJournal(t *testing.T) {
	// TODO(review r29): L-1
	// Engine.Recover (and Run/Flush/Retain) currently calls e.open(ctx) before
	// e.withLock, opening and migrating journal.db RW even when update.lock is
	// held by another process.
	root := t.TempDir()
	lay := testLayout(root)
	bootID := uuid.Must(uuid.NewV7())
	lock, err := nodelock.AcquireWithBootID(context.Background(), lay.Lock, 1, 0, bootID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	eng := &Engine{
		Layout:   lay,
		Node:     Node{ID: uuid.Must(uuid.NewV7()), Incarnation: uuid.Must(uuid.NewV7()), BootID: bootID},
		Platform: testPlatform(),
	}
	recErr := eng.Recover(context.Background())
	var op *OpError
	if !errors.As(recErr, &op) || op.Code != apierr.UpdateBusy {
		t.Fatalf("Recover under held update.lock = %v, want UPDATE_BUSY", recErr)
	}
	if _, statErr := os.Stat(filepath.Join(lay.Journal, "journal.db")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("contended Recover opened/migrated journal.db before acquiring update.lock (stat err = %v)", statErr)
	}
}

func TestR29_Held_ContendedRecoverDuringStepStartAndCheckDoesNotMutateTxnOrGate(t *testing.T) {
	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	w.rollbackAllow = []uuid.UUID{r1.id}
	w.releaseSeq = "5"
	r2 := w.makeRelease("r2", 1, []int{1})
	w.seed(r1)
	w.setWork(w.work(r2, "10", api.PublicationIntentPublished))

	var contendedErr error
	var stageDuringContended updatejournal.Stage
	var blockedDuringContended bool

	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage != string(updatejournal.StageChecking) {
			return nil
		}
		// While the parent updater is inside stepStart -> stepCheck holding update.lock,
		// a candidate agent's fvpn-updater recover runs against the same layout.
		candUpdater := &Engine{
			Layout: w.layout, Node: w.node, Platform: testPlatform(),
			ReleaseTrust: w.trust, Source: w.src, HTTP: w.srv.Client,
			Meter: w.meter, Services: w.svc, Health: w.health, Gate: w.gate, Migrator: w.mig,
			Journal: w.eng.Journal, Now: w.eng.Now, Sleep: w.eng.Sleep, Jitter: w.eng.Jitter,
			AllowLoopback: true,
		}
		contendedErr = candUpdater.Recover(context.Background())
		txn, err := w.eng.Journal.ActiveTxn(context.Background())
		if err == nil {
			stageDuringContended = txn.Stage
		}
		blockedDuringContended, _, _ = w.gate.snapshot()
		return nil
	}

	if err := w.run(); err != nil {
		t.Fatalf("parent updater Run failed: %v", err)
	}
	var op *OpError
	if !errors.As(contendedErr, &op) || op.Code != apierr.UpdateBusy || op.Detail != "lock" {
		t.Fatalf("contended Recover during stepCheck = %v, want UPDATE_BUSY: lock", contendedErr)
	}
	if stageDuringContended != updatejournal.StageChecking || !blockedDuringContended {
		t.Fatalf("contended Recover mutated active transaction or gate: stage=%q blocked=%v", stageDuringContended, blockedDuringContended)
	}
	w.mustCurrent(r2.id)
	blockedAfter, _, unblocks := w.gate.snapshot()
	if blockedAfter || unblocks == 0 {
		t.Fatalf("parent seal did not unblock gate after completing stepCheck: blocked=%v unblocks=%d", blockedAfter, unblocks)
	}
	// Once the parent updater releases update.lock, a subsequent Recover succeeds with exit 0.
	if err := w.eng.Recover(context.Background()); err != nil || ExitCode(err) != 0 {
		t.Fatalf("Recover after parent seal released update.lock = %v (exit %d), want nil (0)", err, ExitCode(err))
	}
}

func TestR29_Held_CrashedSwitchedAndCheckingTxnsRecoveredUnderFreeLock(t *testing.T) {
	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	w.rollbackAllow = []uuid.UUID{r1.id}
	w.releaseSeq = "5"
	r2 := w.makeRelease("r2", 1, []int{1})
	w.seed(r1)
	work := w.work(r2, "10", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{r1.id, r2.id}
	w.setWork(work)

	// Crash right after switching symlink to r2 (StageSwitched), releasing update.lock.
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage == string(updatejournal.StageSwitched) {
			return ErrCrashed
		}
		return nil
	}
	if err := w.run(); !errors.Is(err, ErrCrashed) {
		t.Fatalf("Run with StageSwitched crash = %v, want ErrCrashed", err)
	}
	w.eng.Hooks.AfterStage = nil

	txn, err := w.eng.Journal.ActiveTxn(context.Background())
	if err != nil || txn.Stage != updatejournal.StageSwitched {
		t.Fatalf("active txn after crash = %+v, err = %v; want StageSwitched", txn, err)
	}
	blocked, _, _ := w.gate.snapshot()
	if !blocked {
		t.Fatal("forwarding gate must stay blocked across AfterSwitch crash")
	}

	// If candidate health fails during recovery, Recover rolls back to r1 (LastGood),
	// unblocks forwarding for published LastGood, and exits 0.
	w.health.fail[r2.id.String()] = true
	if err := w.eng.Recover(context.Background()); err != nil || ExitCode(err) != 0 {
		t.Fatalf("Recover of crashed StageSwitched txn with failing candidate health = %v (exit %d), want nil (0)", err, ExitCode(err))
	}
	w.mustCurrent(r1.id)
	blocked, _, _ = w.gate.snapshot()
	if blocked {
		t.Fatal("forwarding gate must be unblocked after successful rollback to published LastGood")
	}
}
