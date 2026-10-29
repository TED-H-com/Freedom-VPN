package nodelock

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	if !ok || got.PID != os.Getpid() || got.BootID != readBootID() || got.BootID == "old-boot" {
		t.Fatalf("holder %+v ok %v", got, ok)
	}
}
