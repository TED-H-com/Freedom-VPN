//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// observeUpdateRecovery records the parent lock and candidate heartbeat without
// acquiring the lock or changing a scenario predicate. The observer is joined
// before return (and on test cleanup); its stop marker belongs to the harness.
func (r *runner) observeUpdateRecovery(ctx context.Context) func() {
	startedAt := time.Now().UTC()
	writeContext := func(completedAt time.Time, started, joined bool, inputs map[string]recoveryInput) {
		value := projectRecoveryContext(r.runID, r.sourceCommit, startedAt, completedAt, started, joined, inputs)
		raw, err := json.MarshalIndent(value, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(r.dir, "update-recovery-context.json"), append(raw, '\n'), 0o644)
		}
		if err != nil {
			r.t.Errorf("write recovery context: %v", err)
		}
	}
	const marker = "/var/lib/vpn-node/e2e/recovery-observer-stop"
	stopPath := filepath.Join(r.lab.state, "node", "var", "e2e", "recovery-observer-stop")
	_ = os.Remove(stopPath)
	cmd := exec.CommandContext(ctx, "docker", "exec", r.lab.node, "sh", "-c", `
while [ ! -e `+marker+` ]; do
  printf 'sample_at '; date -u +%Y-%m-%dT%H:%M:%S.%NZ
  printf 'agent_pid '; cat /var/lib/vpn-node/e2e/agent.pid 2>/dev/null || true
  inode=$(stat -c %i /var/lib/vpn-node/update.lock 2>/dev/null)
  printf 'lock_holder_pids '
  if [ -n "$inode" ]; then
    awk -v inode="$inode" '$2 == "FLOCK" && $6 ~ (":" inode "$") {printf "%s ", $5}' /proc/locks
  fi
  printf '\n'
  if [ -n "$inode" ]; then
    for holder in $(awk -v inode="$inode" '$2 == "FLOCK" && $6 ~ (":" inode "$") {print $5}' /proc/locks); do
      printf 'lock_holder_comm %s ' "$holder"; cat "/proc/$holder/comm" 2>/dev/null || true
    done
  fi
  printf 'agent_executable '; readlink "/proc/$(cat /var/lib/vpn-node/e2e/agent.pid 2>/dev/null)/exe" 2>/dev/null || true
  printf 'forwarding_blocked '; if [ -e /var/lib/vpn-node/update-journal/forwarding.blocked ]; then echo 1; else echo 0; fi
  cat /run/vpn-node/heartbeat 2>/dev/null || true
  printf 'sample_end\n'
  sleep 0.2
done`)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		writeContext(time.Now().UTC(), false, false, nil)
		r.t.Errorf("start recovery observer: %v", err)
		return func() {}
	}
	var once sync.Once
	finish := func() {
		once.Do(func() {
			if err := os.WriteFile(stopPath, []byte("1\n"), 0o644); err != nil {
				r.t.Errorf("stop recovery observer: %v", err)
				_ = cmd.Process.Kill()
			}
			joinErr := cmd.Wait()
			completedAt := time.Now().UTC()
			if joinErr != nil {
				r.t.Errorf("join recovery observer: %v", joinErr)
			}
			inputs := map[string]recoveryInput{"heartbeats": {Data: output.Bytes(), Available: joinErr == nil}}
			_ = os.Remove(stopPath)
			if err := os.WriteFile(filepath.Join(r.dir, "update-recovery-heartbeats.txt"), []byte(Redact(output.String())), 0o644); err != nil {
				r.t.Errorf("write recovery observations: %v", err)
			}
			readCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for _, name := range []string{"agent", "watchdog"} {
				log, err := exec.CommandContext(readCtx, "docker", "exec", r.lab.node, "cat", "/var/lib/vpn-node/e2e/"+name+".log").CombinedOutput()
				inputs[name] = recoveryInput{Data: log, Available: err == nil}
				if err != nil {
					r.t.Errorf("read %s log: %v", name, err)
				}
				if err := os.WriteFile(filepath.Join(r.dir, "update-recovery-"+name+".txt"), []byte(Redact(string(log))), 0o644); err != nil {
					r.t.Errorf("write %s log: %v", name, err)
				}
			}
			inputs["enforcement_timing"] = readUpdateTiming(readCtx, r.lab.node)
			if !inputs["enforcement_timing"].Available {
				r.t.Error("read bounded enforcement timing failed")
			}
			writeContext(completedAt, true, joinErr == nil, inputs)
		})
	}
	r.t.Cleanup(finish)
	return finish
}

// Snapshot all retained invocations, so update-bad also contains update-good's
// complete output. This is observation only and does not alter either predicate.
func (r *runner) retainUpdaterInvocations(rec *Record) {
	raw, err := r.lab.docker(r.lab.node, "sh", "-c", "test -d /var/lib/vpn-node/e2e/updater-invocations || exit 0; tar -C /var/lib/vpn-node/e2e -cf - updater-invocations")
	if err != nil {
		r.t.Errorf("read updater invocations: %v", err)
		return
	}
	rows, err := ReadUpdaterInvocations(bytes.NewReader(raw))
	if err != nil {
		r.t.Errorf("parse updater invocations: %v", err)
		return
	}
	for i := range rows {
		rows[i].Output = Redact(rows[i].Output)
	}
	data, err := json.Marshal(rows)
	if err != nil {
		r.t.Error(err)
		return
	}
	rec.Observations = append(rec.Observations, Observation{Name: "updater_invocations", Detail: string(data)})
}

func TestSEC02_RecoveryTimingConfigReachesBootAndCandidates(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "node", "etc")
	base := filepath.Join(root, "node", "var", "e2e")
	for _, dir := range []string{etc, base} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	runPath := filepath.Join(etc, "agent-run.json")
	initial := []byte(`{"test_mode":true,"agent_release":"test-initial","control_url":"https://example.invalid","opaque":"TEST_ONLY_PRIVATE"}`)
	for _, name := range []string{"agent-run.json", "agent-enroll.json"} {
		if err := os.WriteFile(filepath.Join(etc, name), initial, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Only the existing opt-in's exact read/check/write calls are accepted.
	// No real Docker executable or running service participates in this test.
	script := `#!/bin/sh
set -eu
if [ "$#" = 6 ] && [ "$1" = exec ] && [ "$2" = fvpn-timing-test-node ] && [ "$3" = test ] && [ "$4" = '!' ] && [ "$5" = -e ] && [ "$6" = /var/lib/vpn-node/identity/node.json ]; then
  [ "${FVPN_TEST_TIMING_ENROLLED:-}" != 1 ]
elif [ "$#" = 4 ] && [ "$1" = exec ] && [ "$2" = fvpn-timing-test-node ] && [ "$3" = cat ] && [ "$4" = /etc/vpn-node/agent-run.json ]; then
  cat "$FVPN_TEST_TIMING_CONFIG"
elif [ "$#" = 9 ] && [ "$1" = exec ] && [ "$2" = -i ] && [ "$3" = fvpn-timing-test-node ] && [ "$4" = sh ] && [ "$5" = -c ] && [ "$7" = sh ] && [ "$8" = /etc/vpn-node/agent-run.json ] && [ "$9" = 600 ]; then
  cat > "$FVPN_TEST_TIMING_CONFIG"
  chmod 600 "$FVPN_TEST_TIMING_CONFIG"
else
  exit 91
fi
`
	if err := os.WriteFile(filepath.Join(root, "docker"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FVPN_TEST_TIMING_CONFIG", runPath)
	t.Setenv("FVPN_TEST_TIMING_ENROLLED", "")
	l := &lab{node: "fvpn-timing-test-node", state: root}
	s := &Stack{lab: l}
	if err := s.EnableEnforcementTiming(context.Background()); err != nil {
		t.Fatal(err)
	}
	check := func(path, release, version string) {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg["test_timing_path"] != updateTimingPath || cfg["test_mode"] != true || cfg["opaque"] != "TEST_ONLY_PRIVATE" || cfg["control_url"] != "https://example.invalid" || cfg["agent_release"] != version {
			t.Fatalf("timing or existing runtime configuration lost at %s", filepath.Base(path))
		}
		if release != "" && cfg["release_id"] != release {
			t.Fatal("release binding changed")
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("config mode: %v %v", info, err)
		}
	}
	check(runPath, "", "test-initial")
	// Serving preparation copies the enabled runtime config into every release,
	// including boot. Exercise the actual helpers both before and after boot ID
	// binding; update/restart selects these existing copies without rewriting.
	boot, candidate := uuid.New(), uuid.New()
	if err := l.candidateConfig(boot, "test-boot"); err != nil {
		t.Fatal(err)
	}
	if err := l.setBootRelease(boot); err != nil {
		t.Fatal(err)
	}
	if err := l.candidateConfig(candidate, "test-candidate"); err != nil {
		t.Fatal(err)
	}
	check(runPath, boot.String(), "test-initial")
	check(filepath.Join(base, "agent-"+boot.String()+".json"), boot.String(), "test-boot")
	check(filepath.Join(base, "agent-"+candidate.String()+".json"), candidate.String(), "test-candidate")
	for _, testMode := range []bool{true, false} {
		if testMode {
			t.Setenv("FVPN_TEST_TIMING_ENROLLED", "1")
		} else {
			t.Setenv("FVPN_TEST_TIMING_ENROLLED", "")
			if err := os.WriteFile(runPath, []byte(`{"test_mode":false}`), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		before, err := os.ReadFile(runPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.EnableEnforcementTiming(context.Background()); err == nil {
			t.Fatal("unsafe opt-in accepted")
		}
		after, err := os.ReadFile(runPath)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("rejected opt-in changed runtime config")
		}
	}
}
