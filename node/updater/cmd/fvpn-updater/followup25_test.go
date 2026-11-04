package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

func TestD158_CLIWaitRecordAndExactErrorLine(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "fvpn-updater")
	if out, err := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
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
	for _, tc := range []struct {
		name, detail string
		releaseAfter time.Duration
		wantExit     int
	}{
		{name: "uncontended"},
		{name: "released", releaseAfter: 600 * time.Millisecond},
		{name: "real default budget", wantExit: 3},
		{name: "stale boot", detail: "boot id", wantExit: 1},
		{name: "stale pid", detail: "pid", wantExit: 1},
		{name: "stale process", detail: "process", wantExit: 1},
		{name: "stale start", detail: "start time", wantExit: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var release func()
			if tc.name != "uncontended" {
				lk, err := nodelock.TryAcquire(lay.Lock)
				if err != nil {
					t.Fatal(err)
				}
				var once sync.Once
				release = func() { once.Do(func() { _ = lk.Release() }) }
				defer release()
				if tc.detail != "" {
					b, err := os.ReadFile(lay.Lock)
					if err != nil {
						t.Fatal(err)
					}
					var h map[string]any
					if err := json.Unmarshal(b, &h); err != nil {
						t.Fatal(err)
					}
					switch tc.detail {
					case "boot id":
						h["boot_id"] = "TEST ONLY different boot"
					case "pid":
						h["pid"] = 0
					case "process":
						h["pid"] = 1 << 30
					case "start time":
						h["start_time"] = 0
					}
					b, err = json.Marshal(h)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(lay.Lock, b, 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			cmd := exec.CommandContext(t.Context(), bin, "run", "-config", cfg)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			started := time.Now()
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if tc.releaseAfter != 0 {
				timer := time.AfterFunc(tc.releaseAfter, release)
				defer timer.Stop()
			}
			err := cmd.Wait()
			elapsed := time.Since(started)
			code := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			t.Logf("elapsed=%s exit=%d stderr=%s", elapsed, code, &stderr)
			if code != tc.wantExit || stdout.Len() != 0 {
				t.Fatalf("exit=%d stdout=%q stderr=%s", code, &stdout, &stderr)
			}
			if tc.name == "uncontended" {
				if stderr.Len() != 0 {
					t.Fatalf("uncontended stderr changed: %s", &stderr)
				}
				return
			}
			if tc.name == "real default budget" && (elapsed < 10*time.Second || elapsed > 12*time.Second) {
				t.Fatalf("default budget elapsed=%s", elapsed)
			}
			if tc.detail != "" && (elapsed < 80*time.Millisecond || elapsed > time.Second) {
				t.Fatalf("stale window elapsed=%s", elapsed)
			}
			line, remaining, ok := strings.Cut(stderr.String(), "\n")
			var record map[string]any
			if !ok || json.Unmarshal([]byte(line), &record) != nil {
				t.Fatalf("missing JSON wait record: %s", &stderr)
			}
			waited, ok := record["waited_ms"].(float64)
			if !ok || waited < 0 || waited > float64(elapsed.Milliseconds()) || record["msg"] != "update lock wait" || record["command"] != "run" || record["budget_ms"] != float64(10000) || record["acquired"] != (code == 0) {
				t.Fatalf("wrong record: %v", record)
			}
			if tc.name == "real default budget" && waited < 10000 {
				t.Fatalf("budget returned early: %v", record)
			}
			wantError := ""
			if code != 0 {
				wantError = "fvpn-updater: updater: UPDATE_BUSY: lock"
				if tc.detail != "" {
					wantError += " " + tc.detail
				}
				wantError += "\n"
			}
			if remaining != wantError {
				t.Fatalf("error line changed or extra records: %q, want %q", remaining, wantError)
			}
		})
	}
}
