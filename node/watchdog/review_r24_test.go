package watchdog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
)

// TestR24_Held_Q3_WatchdogStaleAndHeldLockBehavior proves What Held for
// Question 3 (the trade & watchdog behavior on main vs C1):
//   - When the agent process exits (releasing LockPath) and its heartbeat goes
//     stale (> WatchdogStaleSeconds = 30s) or reaches PolicyExpires, Watch.Once
//     acquires LockPath and applies Baseline;
//   - When the agent process is stopped (e.g. SIGSTOP) while holding LockPath,
//     Watch.Once cannot acquire LockPath and defers Baseline on both main and
//     C1 until the lock is released (leaving kernel element timeouts in
//     @allowed_v4 / @fv_allow_A as the active enforcement ceiling).
func TestR24_Held_Q3_WatchdogStaleAndHeldLockBehavior(t *testing.T) {
	w, x, b, _ := watchRig(t)
	w.Config.LockPath = filepath.Join(t.TempDir(), "update.lock")
	now, _ := w.mono()

	// Make heartbeat stale (> 30s).
	b.Mono = now - 31*time.Second
	if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
		t.Fatal(err)
	}

	// While a stopped process holds update.lock, Watch.Once defers Baseline.
	held, err := nodelock.TryAcquire(w.Config.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if x.calls != 0 {
		t.Fatalf("expected Watch.Once to defer Baseline while LockPath is held, got %d calls", x.calls)
	}

	// Once the lock holder exits (releasing update.lock), Watch.Once applies Baseline immediately.
	held.Release()
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if x.calls != 1 {
		t.Fatalf("expected Watch.Once to apply Baseline once LockPath was released, got %d calls", x.calls)
	}
}
