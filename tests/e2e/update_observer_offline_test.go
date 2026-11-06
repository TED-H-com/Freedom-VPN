//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This test supplies the docker executable itself. It never contacts a daemon.
// Failed collection runs in a child so the observer's real t.Errorf paths are
// checked without suppressing them or changing production collection behavior.
func TestAT19_RecoveryObserverOfflineArtifact(t *testing.T) {
	for _, mode := range []string{"success", "start_failure", "join_failure", "read_failure"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAT19_RecoveryObserverChild$")
			cmd.Env = append(os.Environ(), "FVPN_RECOVERY_CHILD=scenario", "FVPN_RECOVERY_MODE="+mode, "FVPN_RECOVERY_ROOT="+root)
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil || (err == nil) != (mode == "success") {
				t.Fatalf("child status: %v context=%v\n%s", err, ctx.Err(), output)
			}
			raw, err := os.ReadFile(filepath.Join(root, "out", "update-recovery-context.json"))
			if err != nil {
				t.Fatal(err)
			}
			var result recoveryContext
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			binding, err := os.ReadFile(filepath.Join(root, "run-id"))
			if err != nil {
				t.Fatal(err)
			}
			if !result.BindingAvailable || result.RunID != string(binding) || result.Source != recoveryTestCommit || strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
				t.Fatalf("invalid or unsafe artifact: %s", raw)
			}
			if mode == "start_failure" {
				if result.Started || result.Joined || result.Streams["agent"].Available || len(result.Streams["heartbeats"].Records) != 0 {
					t.Fatalf("invented observation: %s", raw)
				}
				return
			}
			if !result.Started || result.Joined != (mode != "join_failure") || result.Streams["heartbeats"].Available != result.Joined || len(result.Streams["heartbeats"].Records) != 1 {
				t.Fatalf("observer join/partial status: %s", raw)
			}
			a, w := result.Streams["agent"], result.Streams["watchdog"]
			if !a.Available || len(a.Records) != 1 || a.Records[0]["tc_read_failure"] != "command_exit" || a.Records[0]["window_position"] != "before" || w.Available != (mode != "read_failure") || len(w.Records) != 1 || w.Records[0]["window_position"] != "after" {
				t.Fatalf("actual log buffers not projected: %s", raw)
			}
			for _, name := range []string{"heartbeats", "agent", "watchdog"} {
				if _, err := os.Stat(filepath.Join(root, "out", "update-recovery-"+name+".txt")); err != nil {
					t.Fatalf("existing TXT missing: %v", err)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "state", "node", "var", "e2e", "recovery-observer-stop")); !os.IsNotExist(err) {
				t.Fatalf("observer marker not removed: %v", err)
			}
		})
	}
}

func TestAT19_RecoveryObserverChild(t *testing.T) {
	role := os.Getenv("FVPN_RECOVERY_CHILD")
	if role == "" {
		return
	}
	root, mode := os.Getenv("FVPN_RECOVERY_ROOT"), os.Getenv("FVPN_RECOVERY_MODE")
	if role == "docker" {
		var args []string
		for i, arg := range os.Args {
			if arg == "--" {
				args = os.Args[i+1:]
				break
			}
		}
		if len(args) < 4 || args[0] != "exec" || args[1] != "fvpn-offline-node" {
			os.Exit(80)
		}
		if args[2] == "sh" {
			fmt.Print(recoveryHeartbeat(time.Now().UTC(), uuid.NewString()))
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := os.Stat(filepath.Join(root, "state", "node", "var", "e2e", "recovery-observer-stop")); err == nil {
					if mode == "join_failure" {
						os.Exit(4)
					}
					os.Exit(0)
				}
				time.Sleep(5 * time.Millisecond)
			}
			os.Exit(81)
		}
		if args[2] != "cat" {
			os.Exit(82)
		}
		if filepath.Base(args[3]) == "agent.log" {
			raw, err := os.ReadFile(filepath.Join(root, "agent.log"))
			if err != nil {
				os.Exit(83)
			}
			_, _ = os.Stdout.Write(raw)
		} else if filepath.Base(args[3]) == "watchdog.log" {
			_, _ = os.Stdout.Write(recoveryLog(time.Now().UTC(), map[string]any{"msg": "baseline write", "writer": "watchdog", "phase": "end", "cause": "heartbeat_stale", "succeeded": true}))
			if mode == "read_failure" {
				fmt.Fprintln(os.Stderr, "TEST_ONLY_PRIVATE_STDERR")
				os.Exit(9)
			}
		} else {
			os.Exit(84)
		}
		os.Exit(0)
	}
	if role != "scenario" {
		t.Fatal("unexpected child role")
	}
	for _, dir := range []string{"bin", "out", filepath.Join("state", "node", "var", "e2e")} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if mode != "start_failure" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		quoted := "'" + strings.ReplaceAll(executable, "'", "'\"'\"'") + "'"
		script := "#!/bin/sh\nFVPN_RECOVERY_CHILD=docker\nexport FVPN_RECOVERY_CHILD\nexec " + quoted + " -test.run=^TestAT19_RecoveryObserverChild$ -- \"$@\"\n"
		if err := os.WriteFile(filepath.Join(root, "bin", "docker"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", filepath.Join(root, "bin"))
	runID := uuid.NewString()
	if err := os.WriteFile(filepath.Join(root, "run-id"), []byte(runID), 0o600); err != nil {
		t.Fatal(err)
	}
	log := recoveryLog(time.Now().Add(-time.Second), map[string]any{"msg": "apply", "err": "TC_FOREIGN_FILTER", "tc_read_failure": "command_exit", "tc_read_context_state_at_return": "deadline", "tc_read_interface": "fvup0", "tc_read_attachment": "7560:", "stderr": "TEST_ONLY_PRIVATE_STDERR"})
	if err := os.WriteFile(filepath.Join(root, "agent.log"), log, 0o600); err != nil {
		t.Fatal(err)
	}
	r := &runner{t: t, lab: &lab{state: filepath.Join(root, "state"), node: "fvpn-offline-node"}, dir: filepath.Join(root, "out"), runID: runID, sourceCommit: recoveryTestCommit}
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	finish := r.observeUpdateRecovery(ctx)
	finish()
	before, err := os.ReadFile(filepath.Join(r.dir, "update-recovery-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	finish()
	after, err := os.ReadFile(filepath.Join(r.dir, "update-recovery-context.json"))
	if err != nil || string(before) != string(after) {
		t.Fatalf("finish was not idempotent: %v", err)
	}
}
