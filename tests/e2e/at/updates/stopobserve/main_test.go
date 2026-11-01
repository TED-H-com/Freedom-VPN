//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// Exercise real kernel wait statuses without root, Docker or network access.
// A reaped process cannot be classified; the observer must retain that gap.
func TestUPD05_StopObservationDistinguishesExitKillAndReap(t *testing.T) {
	for _, killed := range []bool{false, true} {
		name := "exit"
		script := "exit 0"
		if killed {
			name, script = "kill", "exec sleep 30"
		}
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", script)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
			if killed {
				if err := cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
			}
			pid := strconv.Itoa(cmd.Process.Pid)
			deadline := time.Now().Add(5 * time.Second)
			var state string
			var status int
			for time.Now().Before(deadline) {
				state, status = procExit(pid)
				if state == "Z" {
					break
				}
				time.Sleep(time.Millisecond)
			}
			want := 0
			if killed {
				want = 9
			}
			if state != "Z" || status != want {
				t.Fatalf("wait status: state=%s status=%d want=%d", state, status, want)
			}
			_ = cmd.Wait()
			if state, status = procExit(pid); state != "unavailable" || status != -1 {
				t.Fatalf("reaped status must stay unknown: %s %d", state, status)
			}
		})
	}
}

// Use the real devstack parent; its child is already reaped before readback.
// Repeated stop and a later launch must never turn the old exit into a new one.
func TestUPD05_ParentReceiptSurvivesReapRepeatAndRestart(t *testing.T) {
	for _, outcome := range []string{"0", "137", "kill"} {
		t.Run(outcome, func(t *testing.T) {
			base := t.TempDir()
			run := func(marker, result string) (identity, <-chan error) {
				done := make(chan error, 1)
				go func() {
					done <- agentwait.Run(base, "/bin/sh", "-c", `while [ ! -f "$1" ]; do sleep 0.01; done; exit "$2"`, "sh", marker, result)
				}()
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) {
					id, err := capture(base)
					if err == nil && !id.AlreadyStopped {
						return id, done
					}
					time.Sleep(time.Millisecond)
				}
				t.Fatal("parent did not publish child identity")
				return identity{}, done
			}
			marker := filepath.Join(base, "finish")
			status := outcome
			if status == "kill" {
				status = "0"
			}
			id, done := run(marker, status)
			if err := retain(base, id); err != nil {
				t.Fatal(err)
			}
			if outcome == "kill" {
				if err := syscall.Kill(id.PID, syscall.SIGKILL); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(marker, nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			first, err := receipt(base, id)
			if err != nil {
				t.Fatal(err)
			}
			var r agentwait.Result
			if err := json.Unmarshal(first, &r); err != nil {
				t.Fatal(err)
			}
			if outcome == "kill" {
				if r.ExitCode != nil || r.Signal == nil || *r.Signal != 9 {
					t.Fatal("signal confused with exit")
				}
			} else {
				want, _ := strconv.Atoi(outcome)
				if r.ExitCode == nil || *r.ExitCode != want || r.Signal != nil {
					t.Fatal("exit confused with signal")
				}
			}
			if err := os.Remove(filepath.Join(base, "agent.pid")); err != nil {
				t.Fatal(err)
			}
			repeat, err := capture(base)
			if err != nil || !repeat.AlreadyStopped || repeat.ID != id.ID {
				t.Fatalf("repeat: %+v %v", repeat, err)
			}
			// A new current ID must not redirect the retained read to another launch.
			next, nextDone := run(filepath.Join(base, "finish-next"), "0")
			if next.ID == id.ID {
				t.Fatal("launch ID reused")
			}
			if err := os.WriteFile(filepath.Join(base, "finish-next"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := <-nextDone; err != nil {
				t.Fatal(err)
			}
			after, err := receipt(base, id)
			if err != nil || !bytes.Equal(first, after) {
				t.Fatal("old receipt changed across restart")
			}
			wrong := id
			wrong.StartTicks += "1"
			if _, err := receipt(base, wrong); err == nil {
				t.Fatal("PID reuse accepted")
			}
			wrong = id
			wrong.ID = next.ID
			if err := validateReceipt(first, wrong); err == nil {
				t.Fatal("wrong launch accepted")
			}
			var fields map[string]any
			_ = json.Unmarshal(first, &fields)
			delete(fields, "wait_status")
			incomplete, _ := json.Marshal(fields)
			if err := validateReceipt(incomplete, id); err == nil {
				t.Fatal("missing raw status inferred zero")
			}
			row := map[string]any{}
			if err := os.WriteFile(filepath.Join(base, "agent-exits", id.ID+".sigkill.json"), []byte(`{"signal":9,"kill_accepted":false}`), 0600); err != nil {
				t.Fatal(err)
			}
			observeReceipt(base, id, row)
			if row["parent_wait_receipt"] == nil || row["sigkill_attempt"] == nil {
				t.Fatal("attempt must be separate from outcome")
			}
			_ = os.Remove(filepath.Join(base, "agent-exits", id.ID+".json"))
			row = map[string]any{}
			observeReceipt(base, id, row)
			if row["receipt_error"] == nil || row["parent_wait_receipt"] != nil {
				t.Fatal("missing receipt inferred from kill attempt")
			}
		})
	}
}

func TestUPD05_SystemdParentWaitidDoesNotInferContainerExit(t *testing.T) {
	id := identity{ID: "launch", PID: 123, StartTicks: "42"}
	for _, tc := range []struct {
		code, status int
		exit, signal any
	}{{1, 0, 0, nil}, {1, 137, 137, nil}, {2, 9, nil, 9}} {
		r, err := systemdReceipt(id, fmt.Sprintf("ExecMainPID=123\nExecMainCode=%d\nExecMainStatus=%d\nExecMainExitTimestamp=Sun 2026-11-01 15:00:00 UTC", tc.code, tc.status))
		if err != nil || r["exit_code"] != tc.exit || r["signal"] != tc.signal {
			t.Fatalf("parent result: %v %v", r, err)
		}
		if _, ok := r["wait_status"]; ok {
			t.Fatal("waitid encoding mislabeled raw wait status")
		}
	}
	if _, err := systemdReceipt(id, "ContainerExitCode=0"); err == nil {
		t.Fatal("container success inferred agent exit")
	}
}
