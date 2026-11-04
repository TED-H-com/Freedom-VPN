package updater

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

// TestR30_H2_StaleThenLiveLockMetadataAbortsEarlyWithExitBusy proves H-2:
// in acquireCommandLock (node/updater/lock_wait.go:47-55), once attempt 1
// observes a non-"lock" busy detail (such as stale holder metadata left in
// update.lock before a new live holder finishes truncating/writing its own
// metadata), `if staleAttempts > 0 { staleAttempts++ }` increments
// staleAttempts on every subsequent busy poll even when lockBusyDetail(err) ==
// "lock" (live holder). On attempt 5 (~80ms), acquireCommandLock aborts and
// returns the attempt-5 bare nodelock.ErrBusy, causing Run/Seed/Apply/Flush to
// fail with ExitBusy (exit 3, UPDATE_BUSY: lock) after ~80ms instead of waiting
// up to the 10s live lock budget.
func TestR30_H2_StaleThenLiveLockMetadataAbortsEarlyWithExitBusy(t *testing.T) {
	e := d158Engine(t)
	release := d158Holder(t, e.Layout.Lock)

	liveMetadata := fileBytes(t, e.Layout.Lock)
	staleMetadata := d158StaleMetadata(t, e.Layout.Lock, "process")

	// Poll 1 (t=0ms) sees stale metadata before the live holder finishes writing
	// its own metadata; before poll 2 (t=20ms), the live holder's metadata is in
	// place, and the live holder releases at t=250ms (well within the 10s budget).
	if err := os.WriteFile(e.Layout.Lock, staleMetadata, 0o600); err != nil {
		t.Fatal(err)
	}
	restoreLive := time.AfterFunc(8*time.Millisecond, func() {
		_ = os.WriteFile(e.Layout.Lock, liveMetadata, 0o600)
	})
	defer restoreLive.Stop()
	releaseHolder := time.AfterFunc(250*time.Millisecond, release)
	defer releaseHolder.Stop()

	started := time.Now()
	err := e.Flush(t.Context())
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("Flush aborted after %s with %v (exit %d); want wait until 250ms release within 10s live lock budget once metadata became live on poll 2",
			elapsed, err, ExitCode(err))
	}
}

// TestR30_M1_NonBusyFirstLockErrorEmitsSpuriousWaitLog proves M-1:
// in acquireCommandLock (node/updater/lock_wait.go:43-47), `waited = true` is
// set before `if !errors.Is(err, nodelock.ErrBusy)`, so a non-busy lock error
// on the very first attempt (no lock contention and zero wait) still emits an
// "update lock wait" structured log record, violating D-158 / Invariant 8
// ("when and only when the first lock attempt missed").
func TestR30_M1_NonBusyFirstLockErrorEmitsSpuriousWaitLog(t *testing.T) {
	e := d158Engine(t)

	// Make the lock path a directory so the very first open fails with EISDIR
	// (a non-busy filesystem error, not nodelock.ErrBusy).
	if err := os.RemoveAll(e.Layout.Lock); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(e.Layout.Lock, 0o755); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	e.UseLog(slog.New(slog.NewJSONHandler(&logs, nil)))

	err := e.Flush(context.Background())
	if err == nil || ExitCode(err) == ExitBusy {
		t.Fatalf("Flush err = %v (exit %d), want non-busy lock failure", err, ExitCode(err))
	}
	if strings.Contains(logs.String(), `"msg":"update lock wait"`) {
		t.Fatalf("non-busy first-attempt lock failure emitted spurious wait log: %s", logs.String())
	}
}

// TestR30_D155_UpdaterRecoverStaysPromptNoWaitLog verifies that Recover and
// Retain remain on the immediate single-attempt lock path (no 10s wait, no
// "update lock wait" log), while operator commands emit the structured wait log
// only when the first attempt misses on nodelock.ErrBusy.
func TestR30_D155_UpdaterRecoverStaysPromptNoWaitLog(t *testing.T) {
	e := d158Engine(t)

	var logs bytes.Buffer
	e.UseLog(slog.New(slog.NewJSONHandler(&logs, nil)))

	// Uncontended Flush: no wait log.
	if err := e.Flush(context.Background()); err != nil {
		t.Fatalf("uncontended Flush: %v", err)
	}
	if logs.Len() != 0 {
		t.Fatalf("uncontended Flush logged: %s", logs.String())
	}

	// Contended Recover and Retain: immediate ExitBusy, no wait log.
	held, err := nodelock.Acquire(context.Background(), filepath.Clean(e.Layout.Lock), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()

	started := time.Now()
	if err := e.Recover(context.Background()); ExitCode(err) != ExitBusy {
		t.Fatalf("Recover under lock = %v (exit %d), want ExitBusy", err, ExitCode(err))
	}
	if err := e.Retain(context.Background()); ExitCode(err) != ExitBusy {
		t.Fatalf("Retain under lock = %v (exit %d), want ExitBusy", err, ExitCode(err))
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("prompt Recover/Retain took %s under lock, want prompt 80ms path (<500ms)", elapsed)
	}
	if logs.Len() != 0 {
		t.Fatalf("prompt Recover/Retain logged wait record: %s", logs.String())
	}
}
