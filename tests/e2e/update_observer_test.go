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
	"time"
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
