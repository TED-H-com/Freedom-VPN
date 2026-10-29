package nodelock

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTwoProcessesCannotBothHold(t *testing.T) {
	if os.Getenv("NODELOCK_CHILD") == "1" {
		lk, err := TryAcquire(os.Getenv("NODELOCK_PATH"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "child: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(os.Getenv("NODELOCK_READY"), []byte("1"), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "ready: %v\n", err)
			os.Exit(1)
		}
		_, _ = io.Copy(io.Discard, os.Stdin)
		_ = lk.Release()
		os.Exit(0)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "update.lock")
	ready := filepath.Join(dir, "ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestTwoProcessesCannotBothHold$")
	cmd.Env = append(os.Environ(), "NODELOCK_CHILD=1", "NODELOCK_PATH="+path, "NODELOCK_READY="+ready)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, statErr := os.Stat(ready); statErr == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = stdin.Close()
			_ = cmd.Wait()
			t.Fatal("child did not take the lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := TryAcquire(path); !errors.Is(err, ErrBusy) {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatalf("second process acquired the lock: %v", err)
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	lk, err := TryAcquire(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = lk.Release()
}

func TestStaleBootHolderDoesNotBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.lock")
	raw, err := json.Marshal(holder{PID: os.Getpid(), BootID: "old-boot"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	lk, err := TryAcquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	got, ok := readHolder(lk.f)
	if !ok || got.PID != os.Getpid() || got.BootID != readBootID() || got.BootID == "old-boot" || got.StartTime == 0 {
		t.Fatalf("holder %+v ok %v", got, ok)
	}
}

func TestParseStatStartIgnoresCommandParentheses(t *testing.T) {
	// Field 22 is the 20th field after the last ')' of the command name.
	line := "12 (we)ird) name) S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 22222\n"
	n, ok := parseStatStart(line)
	if !ok || n != 22222 {
		t.Fatalf("start %d ok %v", n, ok)
	}
	if got, ok := processStartTime(os.Getpid()); !ok || got <= 0 {
		t.Fatalf("live start %d ok %v", got, ok)
	}
}

func TestHeldLockReportsStaleReasonWithoutStealing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.lock")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	if _, err := f.WriteAt([]byte(`{"pid":4194303,"boot_id":"stale-boot-id"}`), 0); err != nil {
		t.Fatal(err)
	}
	_, errStale := TryAcquire(path)
	if !errors.Is(errStale, ErrBusy) || !strings.Contains(errStale.Error(), "boot id") {
		t.Fatal(errStale)
	}
	body, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(body), "stale-boot-id") {
		t.Fatalf("lock file replaced: %s %v", body, err)
	}

	if err := f.Truncate(0); err != nil {
		t.Fatal(err)
	}
	boot := readBootID()
	reused := fmt.Sprintf(`{"pid":%d,"boot_id":%q}`, os.Getpid(), boot)
	if _, err := f.WriteAt([]byte(reused), 0); err != nil {
		t.Fatal(err)
	}
	_, errReused := TryAcquire(path)
	if !errors.Is(errReused, ErrBusy) || !strings.Contains(errReused.Error(), "start time") {
		t.Fatal(errReused)
	}
	if errStale.Error() == errReused.Error() {
		t.Fatalf("stale reasons are identical: %v", errStale)
	}
	if !holderStale(holder{PID: os.Getpid(), BootID: boot}, boot) {
		t.Fatal("missing start time was treated as the live process")
	}

	start, ok := processStartTime(os.Getpid())
	if !ok {
		t.Fatal("start time")
	}
	if err := f.Truncate(0); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(holder{PID: os.Getpid(), BootID: boot, StartTime: start})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt(raw, 0); err != nil {
		t.Fatal(err)
	}
	_, errLive := TryAcquire(path)
	if !errors.Is(errLive, ErrBusy) || errLive.Error() != ErrBusy.Error() {
		t.Fatalf("live holder: %v", errLive)
	}
}

func TestPIDReuseWithoutHeldLockIsAcquired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.lock")
	raw, err := json.Marshal(holder{PID: os.Getpid(), BootID: readBootID(), StartTime: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	lk, err := TryAcquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	got, ok := readHolder(lk.f)
	if !ok || got.StartTime == 1 || got.StartTime == 0 || got.PID != os.Getpid() || got.BootID != readBootID() {
		t.Fatalf("holder %+v ok %v", got, ok)
	}
}
