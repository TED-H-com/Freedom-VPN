package agentwait

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestAT23_WaitStatusSurvivesReap(t *testing.T) {
	base := t.TempDir()
	for _, row := range []struct {
		name, command string
		code, signal  int
	}{
		{"zero", "exit 0", 0, 0},
		{"exit137", "exit 137", 137, 0},
		{"sigkill", "kill -KILL $$", 0, 9},
	} {
		t.Run(row.name, func(t *testing.T) {
			if err := Run(base, "/bin/sh", "-c", row.command); err != nil {
				t.Fatal(err)
			}
			id, err := os.ReadFile(filepath.Join(base, "agent-process-id"))
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(base, "agent-exits", strings.TrimSpace(string(id))+".json")
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var r Result
			if err := json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
			if r.StartTicks == "" || r.WaitObservedAt.Before(r.StartedAt) {
				t.Fatalf("bad identity/time: %+v", r)
			}
			if _, err := os.Stat("/proc/" + strconv.Itoa(r.PID)); !os.IsNotExist(err) {
				t.Fatal("child not reaped")
			}
			if row.signal == 0 {
				if r.Signal != nil || r.ExitCode == nil || *r.ExitCode != row.code || r.WaitStatus != uint32(row.code<<8) {
					t.Fatalf("bad exit: %+v", r)
				}
			} else if r.ExitCode != nil || r.Signal == nil || *r.Signal != row.signal || r.WaitStatus != uint32(syscall.SIGKILL) {
				t.Fatalf("bad signal: %+v", r)
			}
			info, err := os.Stat(file)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("receipt permissions")
			}
		})
	}
	files, _ := filepath.Glob(filepath.Join(base, "agent-exits", "*.json"))
	if len(files) != 3 {
		t.Fatal("restart replaced a receipt")
	}
}
