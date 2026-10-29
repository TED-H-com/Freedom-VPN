// Package nodelock is the node-wide exclusive update lock of §13.5: the agent,
// the manual installer and the updater take the same flock before they change
// the version pointer, the state schema or product services. Contention is
// reported as busy (UPDATE_BUSY), never waited on forever.
package nodelock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// DefaultPath is the default lock file (§4 node paths).
const DefaultPath = "/var/lib/vpn-node/update.lock"

// ErrBusy means another process holds the lock (UPDATE_BUSY, §13.11).
var ErrBusy = errors.New("UPDATE_BUSY: node update lock is held by another process")

// Lock is a held exclusive flock. The descriptor stays open until Release.
type Lock struct {
	f *os.File
}

// holder is the metadata written only after flock succeeds (D-034d).
type holder struct {
	PID    int    `json:"pid"`
	BootID string `json:"boot_id"`
}

// TryAcquire takes LOCK_EX|LOCK_NB before it writes holder metadata. It
// returns ErrBusy when the lock is held elsewhere and does not remove the
// file. A crashed holder is released by the kernel when its descriptor
// closes. Stale detection compares the recorded boot id and pid; a stale
// record does not justify deleting a file another process still has locked.
func TryAcquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			h, ok := readHolder(f)
			_ = f.Close()
			if ok && holderStale(h, readBootID()) {
				// The record is stale, but flock is still held. Do not unlink.
				return nil, fmt.Errorf("%w", ErrBusy)
			}
			return nil, ErrBusy
		}
		_ = f.Close()
		return nil, fmt.Errorf("flock: %w", err)
	}
	if err := writeHolder(f); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, err
	}
	return &Lock{f: f}, nil
}

func readBootID() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readHolder(f *os.File) (holder, bool) {
	st, err := f.Stat()
	if err != nil || st.Size() == 0 || st.Size() > 1<<20 {
		return holder{}, false
	}
	buf := make([]byte, st.Size())
	n, err := f.ReadAt(buf, 0)
	if err != nil || n == 0 {
		return holder{}, false
	}
	var h holder
	if json.Unmarshal(buf[:n], &h) != nil {
		return holder{}, false
	}
	return h, true
}

func writeHolder(f *os.File) error {
	raw, err := json.Marshal(holder{PID: os.Getpid(), BootID: readBootID()})
	if err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.WriteAt(raw, 0); err != nil {
		return err
	}
	return f.Sync()
}

// holderStale reports whether a recorded holder cannot still be this boot's
// live process. A different boot id is stale even when the pid number has
// been reused. A dead pid is stale on the same boot.
func holderStale(h holder, boot string) bool {
	if h.BootID != boot {
		return true
	}
	if h.PID <= 0 {
		return true
	}
	return syscall.Kill(h.PID, 0) != nil
}

// Acquire retries TryAcquire with a fixed interval until attempts run out or
// ctx ends (bounded wait, §13.11 UPDATE_BUSY).
func Acquire(ctx context.Context, path string, attempts int, interval time.Duration) (*Lock, error) {
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		l, err := TryAcquire(path)
		if err == nil {
			return l, nil
		}
		last = err
		if !errors.Is(err, ErrBusy) {
			return nil, err
		}
		if i == attempts-1 {
			break
		}
		t := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
	return nil, last
}

// Release drops the lock. It is safe to call more than once.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	cerr := l.f.Close()
	l.f = nil
	if err != nil {
		return err
	}
	return cerr
}
