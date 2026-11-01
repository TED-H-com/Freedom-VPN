//go:build linux

package main

import (
	"os/exec"
	"strconv"
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
