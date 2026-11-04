package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
)

func f49RecoverCommand(t *testing.T, r *rig, body string) string {
	t.Helper()
	dir := t.TempDir()
	count := filepath.Join(dir, "calls")
	script := filepath.Join(dir, "recover")
	config := filepath.Join(dir, "updater.json")
	if err := os.WriteFile(config, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte(fmt.Sprintf("#!/bin/sh\nset -eu\necho call >> %q\n%s\n", count, body)), 0700); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.UpdaterRecover, r.eng.cfg.UpdaterConfig = script, config
	r.eng.recovered = false
	return count
}

func f49Calls(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(raw), "call\n")
}

func TestD155_OldUpdaterDefersBeforeSpawn(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	// Pre-exit-3 updater: contention would be exit 1. A parent held through
	// candidate startup is detected without executing that legacy failure.
	count := f49RecoverCommand(t, r, fmt.Sprintf("exec 9>>%q\nflock -n 9 || exit 1\nexit 0", r.eng.cfg.LockPath))
	lock, err := nodelock.Acquire(context.Background(), r.eng.cfg.LockPath, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	for range 3 {
		if err := r.eng.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
		if r.eng.serving || r.eng.recovered || !r.eng.recoverBlocked {
			t.Fatal("live parent latched or served")
		}
	}
	if f49Calls(t, count) != 0 {
		t.Fatal("spawned legacy updater under parent lock")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.recovered || r.eng.recoverBlocked {
		t.Fatal("legacy updater did not recover after release")
	}
	if f49Calls(t, count) != 1 {
		t.Fatal("wrong recovery count")
	}
}

func TestD155_RecoverExitAndRetry(t *testing.T) {
	for _, code := range []int{0, 1, 2, 3} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			r := newRig(t)
			count := f49RecoverCommand(t, r, fmt.Sprintf("exit %d", code))
			r.eng.recoverOnce(context.Background())
			if r.eng.recovered != (code != 3) || r.eng.recoverBlocked != (code != 0) {
				t.Fatalf("exit %d: recovered=%v blocked=%v", code, r.eng.recovered, r.eng.recoverBlocked)
			}
			for range 10 {
				r.eng.recoverOnce(context.Background())
			}
			r.clock.Advance(recoverRetryInterval - time.Nanosecond)
			r.eng.recoverOnce(context.Background())
			if f49Calls(t, count) != 1 {
				t.Fatal("retry storm")
			}
			r.clock.Advance(time.Nanosecond)
			r.eng.recoverOnce(context.Background())
			want := 1
			if code == 3 {
				want = 2
			}
			if f49Calls(t, count) != want {
				t.Fatal("incorrect retry/terminal result")
			}
		})
	}
}

func TestD155_StaleHolderIsTerminal(t *testing.T) {
	for _, reason := range []string{"boot id", "pid", "process", "start time"} {
		t.Run(reason, func(t *testing.T) {
			r := newRig(t)
			count := f49RecoverCommand(t, r, "exit 1")
			lock, err := nodelock.Acquire(context.Background(), r.eng.cfg.LockPath, 1, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			raw, err := os.ReadFile(r.eng.cfg.LockPath)
			if err != nil {
				t.Fatal(err)
			}
			var h struct {
				PID       int    `json:"pid"`
				BootID    string `json:"boot_id"`
				StartTime int64  `json:"start_time"`
			}
			if err := json.Unmarshal(raw, &h); err != nil {
				t.Fatal(err)
			}
			switch reason {
			case "boot id":
				h.BootID = "00000000-0000-4000-8000-000000000001"
			case "pid":
				h.PID = 0
			case "process":
				h.PID = 2147483647
			case "start time":
				h.StartTime++
			}
			raw, _ = json.Marshal(h)
			if err := os.WriteFile(r.eng.cfg.LockPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if r.eng.updateLockHeld() {
				t.Fatal("stale holder counted as live")
			}
			r.eng.recoverOnce(context.Background())
			r.clock.Advance(2 * time.Second)
			r.eng.recoverOnce(context.Background())
			if !r.eng.recovered || !r.eng.recoverBlocked || f49Calls(t, count) != 1 {
				t.Fatal("stale holder deferred forever")
			}
		})
	}
}

func TestD155_MissingConfigTerminal(t *testing.T) {
	r := newRig(t)
	count := f49RecoverCommand(t, r, "exit 0")
	r.eng.cfg.UpdaterConfig = ""
	r.eng.recoverOnce(context.Background())
	if !r.eng.recovered || !r.eng.recoverBlocked || f49Calls(t, count) != 0 {
		t.Fatal("missing config not terminal")
	}
}
