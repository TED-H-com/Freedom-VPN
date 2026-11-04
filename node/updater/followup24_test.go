package updater

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func TestD155_ExitCode(t *testing.T) {
	busy := transient(apierr.UpdateBusy, "lock")
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"blocked", ErrMembersBlocked, 2},
		{"wrapped blocked", fmt.Errorf("recovery: %w", ErrMembersBlocked), 2},
		{"live holder", busy, ExitBusy},
		{"wrapped live holder", fmt.Errorf("recovery: %w", busy), ExitBusy},
		{"retry flag is not the classifier", permanent(apierr.UpdateBusy, "lock"), ExitBusy},
		{"blocked wins", errors.Join(busy, ErrMembersBlocked), 2},
		{"blocked wins reversed", errors.Join(ErrMembersBlocked, busy), 2},
		{"stale boot", transient(apierr.UpdateBusy, "lock boot id"), 1},
		{"stale pid", transient(apierr.UpdateBusy, "lock pid"), 1},
		{"stale process", transient(apierr.UpdateBusy, "lock process"), 1},
		{"stale start", transient(apierr.UpdateBusy, "lock start time"), 1},
		{"other busy", transient(apierr.UpdateBusy, "other"), 1},
		{"empty busy", transient(apierr.UpdateBusy, ""), 1},
		{"not exact detail", transient(apierr.UpdateBusy, "lock "), 1},
		{"not busy", transient(apierr.ValidationFailed, "lock"), 1},
		{"plain", errors.New("updater: UPDATE_BUSY: lock"), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
	if ExitBusy != 3 {
		t.Fatalf("ExitBusy = %d, want 3", ExitBusy)
	}
}

func TestD155_AllEntryPointsLockBeforeJournal(t *testing.T) {
	for _, state := range []string{"missing", "corrupt", "attached"} {
		for _, name := range []string{"recover", "run", "flush", "retain", "apply", "seed"} {
			t.Run(state+"/"+name, func(t *testing.T) {
				lay := testLayout(t.TempDir())
				eng := &Engine{Layout: lay, Sender: HTTPSink{}}
				t.Cleanup(func() { _ = eng.Close() })
				path := filepath.Join(lay.Journal, updatejournal.FileName)
				corrupt := []byte("TEST ONLY corrupt journal; must not be opened under contention")
				if state == "corrupt" {
					if err := os.MkdirAll(lay.Journal, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, corrupt, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				if state == "attached" {
					var err error
					eng.Journal, err = updatejournal.Open(lay.Journal, updatejournal.Options{})
					if err != nil {
						t.Fatal(err)
					}
				}
				journal := eng.Journal
				lk, err := nodelock.TryAcquire(lay.Lock)
				if err != nil {
					t.Fatal(err)
				}
				defer lk.Release()
				calls := map[string]func(context.Context) error{
					"recover": eng.Recover, "run": eng.Run, "flush": eng.Flush, "retain": eng.Retain,
					"apply": func(ctx context.Context) error { return eng.ApplyPending(ctx, "absent-pending.json") },
					"seed":  func(ctx context.Context) error { return eng.Seed(ctx, "absent-manifest.json", "absent-tree") },
				}
				err = calls[name](t.Context())
				var op *OpError
				if !errors.As(err, &op) || op.Code != apierr.UpdateBusy || op.Detail != "lock" || ExitCode(err) != ExitBusy {
					t.Fatalf("%s under live lock: %v, want UPDATE_BUSY: lock / exit 3", name, err)
				}
				if eng.Journal != journal || eng.lockHeld {
					t.Fatal("contended engine attached a journal or claimed the lock")
				}
				if state == "missing" {
					if _, err := os.Stat(lay.Journal); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("contended call created journal directory: %v", err)
					}
				}
				if state == "corrupt" {
					after, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(after, corrupt) {
						t.Fatalf("contended call touched corrupt journal: %q, %v", after, err)
					}
					for _, suffix := range []string{"-wal", "-shm"} {
						if _, err := os.Stat(path + suffix); !errors.Is(err, os.ErrNotExist) {
							t.Fatalf("contended call created %s: %v", suffix, err)
						}
					}
				}
			})
		}
	}
}

func TestD155_FirstBootAndNestedLock(t *testing.T) {
	eng := &Engine{
		Layout: testLayout(t.TempDir()), Sender: HTTPSink{},
		Source: &memSource{err: ErrNoWork},
	}
	defer eng.Close()
	// Retain still has no journal to prune on first boot.
	if err := eng.Retain(t.Context()); err != nil || eng.Journal != nil {
		t.Fatalf("first-boot Retain: journal=%v err=%v", eng.Journal, err)
	}
	if err := eng.Run(t.Context()); err != nil {
		t.Fatalf("first-boot Run with inner Recover: %v", err)
	}
	if eng.Journal == nil {
		t.Fatal("first boot did not open its journal after taking the lock")
	}
	err := eng.withLock(t.Context(), func() error {
		if err := eng.Flush(t.Context()); err != nil {
			return err
		}
		if err := eng.Retain(t.Context()); err != nil {
			return err
		}
		if !eng.lockHeld {
			t.Fatal("nested calls released the parent's lock flag")
		}
		if other, err := nodelock.TryAcquire(eng.Layout.Lock); !errors.Is(err, nodelock.ErrBusy) {
			_ = other.Release()
			t.Fatalf("nested calls released the real lock: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := nodelock.TryAcquire(eng.Layout.Lock)
	if err != nil || eng.lockHeld {
		t.Fatalf("outer call did not release lock: %v", err)
	}
	_ = other.Release()
}
