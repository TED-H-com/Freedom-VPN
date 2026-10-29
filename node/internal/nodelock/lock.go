// Package nodelock is the node-wide exclusive update lock of §13.5: the agent,
// the manual installer and the updater take the same flock before they change
// the version pointer, the state schema or product services. Contention is
// reported as busy (UPDATE_BUSY), never waited on forever.
package nodelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// DefaultPath is the default lock file (§4 node paths).
const DefaultPath = "/var/lib/vpn-node/update.lock"

// ErrBusy means another process holds the lock (UPDATE_BUSY, §13.11).
var ErrBusy = errors.New("UPDATE_BUSY: node update lock is held by another process")

// Lock is a held exclusive flock. The zero value is not usable.
type Lock struct {
	f *os.File
}

// TryAcquire takes the lock without blocking. It returns ErrBusy when the
// lock is held elsewhere. The lock is released by Release or when the process
// exits, so a crashed holder never wedges the node.
func TryAcquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, fmt.Errorf("flock: %w", err)
	}
	return &Lock{f: f}, nil
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
