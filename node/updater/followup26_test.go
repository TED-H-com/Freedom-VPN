package updater

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

// Reuse the D-158 child, but join it at release so the virtual clock cannot
// advance to the next poll before the real second process has dropped flock.
func d158JoinedHolder(t *testing.T, path string) func() {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestD158_HolderProcess$", "--", "--d158-holder", path)
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			_ = in.Close()
			err := cmd.Wait()
			cancel()
			if err != nil {
				t.Errorf("holder: %v; %s", err, &stderr)
			}
		})
	}
	t.Cleanup(release)
	if ready, err := bufio.NewReader(out).ReadString('\n'); err != nil || ready != "ready\n" {
		t.Fatalf("holder readiness: %q %v", ready, err)
	}
	return release
}

func d158WriteMetadata(t *testing.T, path string, metadata []byte) {
	t.Helper()
	if err := os.WriteFile(path, metadata, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestD158_ConsecutiveStaleResetAcquiresRealHolder(t *testing.T) {
	for _, live := range []string{"valid", "empty", "unreadable"} {
		t.Run(live, func(t *testing.T) {
			e := d158Engine(t)
			release := d158JoinedHolder(t, e.Layout.Lock)
			liveMetadata := fileBytes(t, e.Layout.Lock)
			staleMetadata := d158StaleMetadata(t, e.Layout.Lock, "process")
			switch live {
			case "empty":
				liveMetadata = nil
			case "unreadable":
				liveMetadata = []byte("{")
			}
			var log bytes.Buffer
			e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
			d158WriteMetadata(t, e.Layout.Lock, staleMetadata)
			synctest.Test(t, func(t *testing.T) {
				defer e.Close() // Join journal goroutines before the clock bubble exits.
				// Polls: stale at 0/20/40/60, live at 80, stale at
				// 100/120/140/160, success at 180 ms. Writes are between polls.
				done := make(chan struct{})
				go func() {
					defer close(done)
					time.Sleep(70 * time.Millisecond)
					d158WriteMetadata(t, e.Layout.Lock, liveMetadata)
					time.Sleep(20 * time.Millisecond)
					d158WriteMetadata(t, e.Layout.Lock, staleMetadata)
					time.Sleep(80 * time.Millisecond)
					release()
				}()
				started := time.Now()
				err := e.Flush(t.Context())
				elapsed := time.Since(started)
				<-done
				t.Logf("four stale/live/four stale/release: elapsed=%s exit=%d", elapsed, ExitCode(err))
				if err != nil || elapsed != 180*time.Millisecond || e.Journal == nil {
					t.Fatalf("did not acquire after release: %v elapsed=%s journal=%v", err, elapsed, e.Journal)
				}
				d158Record(t, &log, "flush", true, elapsed)
			})
		})
	}
}

func TestD158_ConsecutiveStaleAfterLiveExactWindow(t *testing.T) {
	for _, detail := range []string{"boot id", "pid", "process", "start time"} {
		t.Run(detail, func(t *testing.T) {
			e := d158Engine(t)
			held, err := nodelock.TryAcquire(e.Layout.Lock)
			if err != nil {
				t.Fatal(err)
			}
			defer held.Release()
			stale := d158StaleMetadata(t, e.Layout.Lock, detail)
			var log bytes.Buffer
			e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
			synctest.Test(t, func(t *testing.T) {
				go func() {
					time.Sleep(70 * time.Millisecond)
					d158WriteMetadata(t, e.Layout.Lock, stale)
				}()
				started := time.Now()
				err := e.Flush(t.Context())
				elapsed := time.Since(started)
				// Four earlier live answers; first stale at 80 ms, fifth at 160 ms.
				if ExitCode(err) != 1 || err.Error() != "updater: UPDATE_BUSY: lock "+detail || elapsed != 160*time.Millisecond || e.Journal != nil {
					t.Fatalf("five consecutive stale: %v elapsed=%s journal=%v", err, elapsed, e.Journal)
				}
				t.Logf("first stale=80ms last stale=%s exit=1 error=%s", elapsed, err)
				d158Record(t, &log, "flush", false, elapsed)
			})
		})
	}
}

func TestD158_ConsecutiveStalePreservesOriginalLiveBudget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		lateStale bool
		restore   bool
		elapsed   time.Duration
		exit      int
		detail    string
	}{
		{"live at deadline", false, false, 10 * time.Second, 3, "lock"},
		{"first live past deadline", true, true, 10040 * time.Millisecond, 3, "lock"},
		{"fifth stale past deadline", true, false, 10060 * time.Millisecond, 1, "lock pid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
					d158WriteMetadata(t, e.Layout.Lock, live)
					if tc.lateStale {
						time.Sleep(9960 * time.Millisecond) // 9.97 s, before the 9.98 s poll
						d158WriteMetadata(t, e.Layout.Lock, stale)
						if tc.restore {
							time.Sleep(60 * time.Millisecond) // 10.03 s, live on the 10.04 s poll
							d158WriteMetadata(t, e.Layout.Lock, live)
						}
					}
				}()
				started := time.Now()
				err := e.Flush(t.Context())
				elapsed := time.Since(started)
				<-done
				t.Logf("original budget: elapsed=%s exit=%d error=%v", elapsed, ExitCode(err), err)
				if ExitCode(err) != tc.exit || err.Error() != "updater: UPDATE_BUSY: "+tc.detail || elapsed != tc.elapsed || e.Journal != nil {
					t.Fatalf("budget/stale termination: %v elapsed=%s journal=%v", err, elapsed, e.Journal)
				}
				d158Record(t, &log, "flush", false, elapsed)
			})
		})
	}
}

func TestD158_ConsecutiveWaitRecordRequiresFirstBusy(t *testing.T) {
	for _, command := range []string{"run", "seed", "apply", "flush"} {
		for _, outcome := range []string{"cancel on entry", "first non-busy error", "busy then cancel"} {
			t.Run(command+"/"+outcome, func(t *testing.T) {
				e := d158Engine(t)
				var log bytes.Buffer
				e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
				busy := outcome == "busy then cancel"
				if busy {
					held, err := nodelock.TryAcquire(e.Layout.Lock)
					if err != nil {
						t.Fatal(err)
					}
					defer held.Release()
				} else if outcome == "first non-busy error" {
					if err := os.MkdirAll(e.Layout.Lock, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()
					if outcome == "cancel on entry" {
						cancel()
					}
					if busy {
						go func() {
							time.Sleep(30 * time.Millisecond)
							cancel()
						}()
					}
					started := time.Now()
					err := d158Command(e, command)(ctx)
					elapsed := time.Since(started)
					if err == nil || ExitCode(err) != 1 || e.Journal != nil {
						t.Fatalf("terminal error: %v journal=%v", err, e.Journal)
					}
					wantElapsed := time.Duration(0)
					if busy {
						wantElapsed = 30 * time.Millisecond
					}
					if elapsed != wantElapsed {
						t.Fatalf("elapsed=%s want %s", elapsed, wantElapsed)
					}
					if outcome == "cancel on entry" || outcome == "busy then cancel" {
						if !errors.Is(err, context.Canceled) {
							t.Fatalf("cancellation: %v", err)
						}
					}
					if outcome == "cancel on entry" {
						if _, err := os.Stat(e.Layout.Lock); !errors.Is(err, os.ErrNotExist) {
							t.Fatalf("cancelled entry attempted lock creation: %v", err)
						}
					}
					if busy {
						d158Record(t, &log, command, false, elapsed)
					} else if log.Len() != 0 {
						t.Fatalf("no busy first attempt, unexpected record: %s", &log)
					}
				})
			})
		}
	}
}
