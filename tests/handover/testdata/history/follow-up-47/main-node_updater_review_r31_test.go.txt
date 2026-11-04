package updater

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

// TestR31_Held_D158_AnswerSequencesAndRecordInvariants verifies Question 5's
// answer-sequence termination and single-record emission invariants on
// acquireCommandLock (node/updater/lock_wait.go:24-70):
//  1. Alternating stale and live answers reset staleAttempts on every live
//     answer and terminate at 10020ms with exit 3 (`updater: UPDATE_BUSY: lock`).
//  2. Four stale then one live, repeated, resets staleAttempts on every 5th
//     poll and terminates at 10080ms with exit 3 (`updater: UPDATE_BUSY: lock`).
//  3. Live until the 10s deadline (9980ms) followed by 5 consecutive stale
//     polls starting at 10000ms terminates at 10080ms with exit 1
//     (`updater: UPDATE_BUSY: lock pid`).
//  4. A non-busy lock error after an initial busy attempt returns the non-busy
//     error (exit 1) and still emits exactly one `update lock wait` record
//     with `acquired=false`.
func TestR31_Held_D158_AnswerSequencesAndRecordInvariants(t *testing.T) {
	t.Run("alternating_stale_and_live_exits_3_at_10020ms", func(t *testing.T) {
		e := d158Engine(t)
		held, err := nodelock.TryAcquire(e.Layout.Lock)
		if err != nil {
			t.Fatal(err)
		}
		defer held.Release()
		live := fileBytes(t, e.Layout.Lock)
		stale := d158StaleMetadata(t, e.Layout.Lock, "pid")
		var log bytes.Buffer
		e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
		d158WriteMetadata(t, e.Layout.Lock, stale)

		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				time.Sleep(10 * time.Millisecond)
				for i := 1; i <= 501; i++ {
					if i%2 == 1 {
						d158WriteMetadata(t, e.Layout.Lock, live)
					} else {
						d158WriteMetadata(t, e.Layout.Lock, stale)
					}
					if i < 501 {
						time.Sleep(20 * time.Millisecond)
					}
				}
			}()
			started := time.Now()
			err := e.Flush(t.Context())
			elapsed := time.Since(started)
			<-done
			if ExitCode(err) != 3 || err.Error() != "updater: UPDATE_BUSY: lock" || elapsed != 10020*time.Millisecond {
				t.Fatalf("alternating stale/live: exit=%d err=%v elapsed=%s, want exit=3 elapsed=10.02s",
					ExitCode(err), err, elapsed)
			}
			d158Record(t, &log, "flush", false, elapsed)
		})
	})

	t.Run("four_stale_then_live_repeated_exits_3_at_10080ms", func(t *testing.T) {
		e := d158Engine(t)
		held, err := nodelock.TryAcquire(e.Layout.Lock)
		if err != nil {
			t.Fatal(err)
		}
		defer held.Release()
		live := fileBytes(t, e.Layout.Lock)
		stale := d158StaleMetadata(t, e.Layout.Lock, "pid")
		var log bytes.Buffer
		e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
		d158WriteMetadata(t, e.Layout.Lock, stale)

		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				time.Sleep(10 * time.Millisecond)
				for i := 1; i <= 504; i++ {
					if i%5 == 4 {
						d158WriteMetadata(t, e.Layout.Lock, live)
					} else {
						d158WriteMetadata(t, e.Layout.Lock, stale)
					}
					if i < 504 {
						time.Sleep(20 * time.Millisecond)
					}
				}
			}()
			started := time.Now()
			err := e.Flush(t.Context())
			elapsed := time.Since(started)
			<-done
			if ExitCode(err) != 3 || err.Error() != "updater: UPDATE_BUSY: lock" || elapsed != 10080*time.Millisecond {
				t.Fatalf("4-stale-1-live repeated: exit=%d err=%v elapsed=%s, want exit=3 elapsed=10.08s",
					ExitCode(err), err, elapsed)
			}
			d158Record(t, &log, "flush", false, elapsed)
		})
	})

	t.Run("live_until_deadline_then_stale_exits_1_at_10080ms", func(t *testing.T) {
		e := d158Engine(t)
		held, err := nodelock.TryAcquire(e.Layout.Lock)
		if err != nil {
			t.Fatal(err)
		}
		defer held.Release()
		stale := d158StaleMetadata(t, e.Layout.Lock, "pid")
		var log bytes.Buffer
		e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))

		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				// Keep live through poll 500 (t=9980ms), switch to stale at t=9990ms
				// so polls 501..505 (10000..10080ms) see 5 consecutive stale answers.
				time.Sleep(9990 * time.Millisecond)
				d158WriteMetadata(t, e.Layout.Lock, stale)
			}()
			started := time.Now()
			err := e.Flush(t.Context())
			elapsed := time.Since(started)
			<-done
			if ExitCode(err) != 1 || err.Error() != "updater: UPDATE_BUSY: lock pid" || elapsed != 10080*time.Millisecond {
				t.Fatalf("live-until-deadline-then-stale: exit=%d err=%v elapsed=%s, want exit=1 elapsed=10.08s",
					ExitCode(err), err, elapsed)
			}
			d158Record(t, &log, "flush", false, elapsed)
		})
	})

	t.Run("busy_first_attempt_then_non_busy_error_emits_one_record", func(t *testing.T) {
		e := d158Engine(t)
		held, err := nodelock.TryAcquire(e.Layout.Lock)
		if err != nil {
			t.Fatal(err)
		}
		defer held.Release()
		var log bytes.Buffer
		e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))

		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				time.Sleep(10 * time.Millisecond)
				// Replace the lock path with a directory before poll 2 (t=20ms)
				// so OpenFile fails with EISDIR (non-ErrBusy).
				_ = os.Remove(e.Layout.Lock)
				_ = os.MkdirAll(filepath.Join(e.Layout.Lock, "sub"), 0o700)
			}()
			started := time.Now()
			err := e.Flush(t.Context())
			elapsed := time.Since(started)
			<-done
			if err == nil || ExitCode(err) != 1 || elapsed != 20*time.Millisecond {
				t.Fatalf("busy then non-busy error: exit=%d err=%v elapsed=%s, want exit=1 elapsed=20ms",
					ExitCode(err), err, elapsed)
			}
			d158Record(t, &log, "flush", false, elapsed)
		})
	})
}
