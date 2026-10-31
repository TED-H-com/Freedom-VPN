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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
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
// StartTime is field 22 of /proc/<pid>/stat, in clock ticks since boot.
type holder struct {
	PID       int    `json:"pid"`
	BootID    string `json:"boot_id"`
	StartTime int64  `json:"start_time"`
}

// TryAcquire takes LOCK_EX|LOCK_NB before it writes holder metadata. It
// returns ErrBusy when the lock is held elsewhere and does not remove the
// file. A crashed holder is released by the kernel when its descriptor
// closes. Stale detection compares the recorded boot id, pid and process
// start time. A stale record does not justify deleting a file another
// process still has locked: the error names the reason and the flock stays.
func TryAcquire(path string) (*Lock, error) {
	return tryAcquire(path, "", "/proc/sys/kernel/random/boot_id")
}

func tryAcquire(path, fallback, bootPath string) (*Lock, error) {
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
			if ok {
				if reason, stale := holderStaleReason(h, bootID(bootPath, fallback)); stale {
					// The record is stale, but flock is still held. Do not unlink.
					return nil, fmt.Errorf("%w: %s", ErrBusy, reason)
				}
			}
			return nil, ErrBusy
		}
		_ = f.Close()
		return nil, fmt.Errorf("flock: %w", err)
	}
	if err := writeHolder(f, bootID(bootPath, fallback)); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, err
	}
	return &Lock{f: f}, nil
}

func readBootID() string {
	return bootID("/proc/sys/kernel/random/boot_id", "")
}

func bootID(path, fallback string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	id, err := uuid.Parse(strings.TrimSpace(string(b)))
	if err != nil || id == uuid.Nil {
		return fallback
	}
	return id.String()
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

func writeHolder(f *os.File, boot string) error {
	start, ok := processStartTime(os.Getpid())
	if !ok {
		return errors.New("nodelock: process start time")
	}
	raw, err := json.Marshal(holder{PID: os.Getpid(), BootID: boot, StartTime: start})
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
// been reused. A dead pid is stale on the same boot. The same pid is stale
// when its /proc start time differs, which is how a reused pid is told apart
// from the process that took the lock.
func holderStale(h holder, boot string) bool {
	_, stale := holderStaleReason(h, boot)
	return stale
}

// holderStaleReason names why a record is stale: "boot id", "pid",
// "process" or "start time". A live flock holder is not stale.
func holderStaleReason(h holder, boot string) (string, bool) {
	if h.BootID != boot {
		return "boot id", true
	}
	if h.PID <= 0 {
		return "pid", true
	}
	if syscall.Kill(h.PID, 0) != nil {
		return "process", true
	}
	start, ok := processStartTime(h.PID)
	if !ok {
		return "process", true
	}
	if h.StartTime == 0 || h.StartTime != start {
		return "start time", true
	}
	return "", false
}

// processStartTime reads field 22 of /proc/<pid>/stat (starttime, clock ticks
// since boot). The command name is in parentheses and may itself contain
// spaces and parentheses, so the fields are counted after the last ')'.
func processStartTime(pid int) (int64, bool) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}
	return parseStatStart(string(b))
}

func parseStatStart(s string) (int64, bool) {
	i := strings.LastIndex(s, ")")
	if i < 0 || i+2 >= len(s) {
		return 0, false
	}
	fields := strings.Fields(s[i+2:])
	// Field 22 is index 19 of the fields that follow the command name.
	if len(fields) <= 19 {
		return 0, false
	}
	n, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// Acquire retries TryAcquire with a fixed interval until attempts run out or
// ctx ends (bounded wait, §13.11 UPDATE_BUSY).
func Acquire(ctx context.Context, path string, attempts int, interval time.Duration) (*Lock, error) {
	return AcquireWithBootID(ctx, path, attempts, interval, "")
}

// AcquireWithBootID uses fallback only if the kernel boot id is unreadable.
// It otherwise has the same bounded contention behavior as Acquire.
func AcquireWithBootID(ctx context.Context, path string, attempts int, interval time.Duration, fallback string) (*Lock, error) {
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		l, err := tryAcquire(path, fallback, "/proc/sys/kernel/random/boot_id")
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
