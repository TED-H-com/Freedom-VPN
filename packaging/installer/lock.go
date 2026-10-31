package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// errBusy is UPDATE_BUSY (§13.5, §13.11). packaging/installer cannot import
// node/internal/nodelock. The flock, the path and the holder JSON match that
// package: LOCK_EX|LOCK_NB, no unlink on contention, holder written only
// after the lock is held.
var errBusy = errors.New("UPDATE_BUSY: node update lock is held by another process")

type nodeLock struct {
	f *os.File
}

type lockHolder struct {
	PID       int    `json:"pid"`
	BootID    string `json:"boot_id"`
	StartTime int64  `json:"start_time"`
}

func tryLock(path string) (*nodeLock, error) {
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
			return nil, errBusy
		}
		return nil, fmt.Errorf("flock: %w", err)
	}
	if err := writeLockHolder(f); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, err
	}
	return &nodeLock{f: f}, nil
}

func (l *nodeLock) Release() error {
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

func writeLockHolder(f *os.File) error {
	start, ok := processStartTime(os.Getpid())
	if !ok {
		return errors.New("installer: process start time")
	}
	raw, err := json.Marshal(lockHolder{PID: os.Getpid(), BootID: readBootFile(), StartTime: start})
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

func readBootFile() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func processStartTime(pid int) (int64, bool) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}
	s := string(b)
	i := strings.LastIndex(s, ")")
	if i < 0 || i+2 >= len(s) {
		return 0, false
	}
	fields := strings.Fields(s[i+2:])
	if len(fields) <= 19 {
		return 0, false
	}
	n, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
