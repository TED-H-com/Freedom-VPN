package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

func TestD155_RecoverCLIExitAndStderrUnderHeldLock(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "fvpn-updater")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build updater: %v\n%s", err, out)
	}
	trust := filepath.Join(root, "trust.json")
	writeCLITrust(t, trust)
	lay := updater.Layout{
		Releases: filepath.Join(root, "releases"), Current: filepath.Join(root, "current"),
		Identity: filepath.Join(root, "identity"), State: filepath.Join(root, "state"),
		Journal: filepath.Join(root, "journal"), Lock: filepath.Join(root, "update.lock"),
		Staging: filepath.Join(root, "staging"), Trust: trust,
	}
	cfg := filepath.Join(root, "updater.json")
	writeCLIConfig(t, cfg, lay, trust, filepath.Join(root, "desired.json"), uuid.Nil)
	lk, err := nodelock.TryAcquire(lay.Lock)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	raw, err := os.ReadFile(lay.Lock)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the parent's real flock while changing only its recorded metadata.
	// Each child must distinguish live contention from stale diagnostics.
	for _, tc := range []struct {
		name, field string
		value       any
		detail      string
		want        int
	}{
		{name: "live same boot", detail: "lock", want: updater.ExitBusy},
		{"stale boot", "boot_id", uuid.NewString(), "lock boot id", 1},
		{"invalid pid", "pid", 0, "lock pid", 1},
		{"absent process", "pid", 1 << 30, "lock process", 1},
		{"reused pid", "start_time", 0, "lock start time", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var holder map[string]json.RawMessage
			if err := json.Unmarshal(raw, &holder); err != nil {
				t.Fatal(err)
			}
			if tc.field != "" {
				value, err := json.Marshal(tc.value)
				if err != nil {
					t.Fatal(err)
				}
				holder[tc.field] = value
			}
			body, err := json.Marshal(holder)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(lay.Lock, body, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(t.Context(), bin, "recover", "-config", cfg)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != tc.want {
				t.Fatalf("recover: %v, want exit %d; stderr=%s", err, tc.want, &stderr)
			}
			if want := "fvpn-updater: updater: UPDATE_BUSY: " + tc.detail + "\n"; stderr.String() != want || stdout.Len() != 0 {
				t.Fatalf("stdout=%q stderr=%q, want empty stdout and %q", &stdout, &stderr, want)
			}
		})
	}
}

func TestD155_StatusRemainsReadOnlyUnderHeldLock(t *testing.T) {
	lay, cfg, _, _, seedArgs := seedCLIInputs(t)
	var before, stderr bytes.Buffer
	if code := run(seedArgs, &before, &stderr); code != 0 {
		t.Fatalf("seed: exit %d, %s", code, &stderr)
	}
	before.Reset()
	args := []string{"status", "-config", cfg}
	if code := run(args, &before, &stderr); code != 0 {
		t.Fatalf("status before lock: exit %d, %s", code, &stderr)
	}
	lk, err := nodelock.TryAcquire(lay.Lock)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	for _, args := range [][]string{args, {"status", "-journal", filepath.Join(lay.Journal, updatejournal.FileName)}} {
		var during, stderr bytes.Buffer
		if code := run(args, &during, &stderr); code != 0 || stderr.Len() != 0 || !bytes.Equal(before.Bytes(), during.Bytes()) {
			t.Fatalf("status under lock: exit %d, stdout=%s stderr=%s; want %s", code, &during, &stderr, &before)
		}
	}
	missing := filepath.Join(lay.Journal, "missing", updatejournal.FileName)
	var out bytes.Buffer
	if code := run([]string{"status", "-journal", missing}, &out, &stderr); code != 1 {
		t.Fatalf("status of absent journal under lock: exit %d, want 1", code)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("status created missing journal directory: %v", err)
	}
}
