//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	_ "modernc.org/sqlite"
)

func installerCampaignDirectory() string {
	if os.Getenv("FVPN_E2E_FOLLOWUP19") == "1" {
		return "followup19"
	}
	return "followup18"
}

func TestE2E_Followup19Installer(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP19") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP19=1 with a fresh owned stack")
	}
	t.Setenv("FVPN_E2E_FOLLOWUP18", "1")
	TestE2E_Followup18Installer(t)
}

func commandExit(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	return -1
}

func (r *runner) installerUpgrade19(ctx context.Context, s *Stack, quiescent bool) {
	rec := Record{ID: "installer-journaled-upgrade", Title: "D-125 real serving checkpoint, candidate switch and sealed receipt", Spec: []string{"UPD05", "AT26", "§13.5", "D-125"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	c := s.installerContainer()
	run := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "docker", append([]string{"exec", c}, args...)...).CombinedOutput()
	}
	if quiescent {
		rec.ID = "installer-quiescent-candidate"
		rec.Title = "Isolated candidate and subsequent seed; not serving acceptance"
		if _, e := run("systemctl", "stop", "fvpn-watchdog.service", "fvpn-agent.service"); e != nil {
			note("stop_error", e.Error())
			return
		}
	}
	if e := s.HoldInstallerApply(ctx, true); e != nil {
		note("hold_error", e.Error())
		return
	}
	before, _ := run("readlink", "/opt/vpn-node/current")
	staged, e := s.ProbeInstallerExit(ctx, "installer-node", []string{"upgrade", "--json", "--manifest", "/var/tmp/e2e-rel/newer/manifest.json", "--artifact", "/var/tmp/e2e-rel/newer/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/newer/trust.json"})
	noteJSON(&rec, "stage", staged)
	stagedPointer, _ := run("readlink", "/opt/vpn-node/current")
	stageOK := e == nil && staged.Exit == 0 && bytes.Equal(before, stagedPointer)
	note("stage_kept_pointer", boolText(stageOK))
	pre, _ := run("sh", "-c", `if [ -e /var/lib/vpn-node/state/meter.checkpoint ]; then cat /var/lib/vpn-node/state/meter.checkpoint; else echo absent; fi`)
	note("checkpoint_before", string(pre))
	usageBefore, usageErr := s.UsageLedgerSnapshot(ctx, "installer-node")
	noteJSON(&rec, "usage_before", usageBefore)
	// The serving fixture and staged request are unchanged. This observer reads
	// short-lived files while the real apply runs; it never supplies either file.
	observer := exec.CommandContext(ctx, "docker", "exec", c, "/usr/local/bin/fvpn-e2e-nodeconfig", "upgrade-observe")
	if e = observer.Start(); e != nil {
		note("observer_error", e.Error())
		return
	}
	time.Sleep(100 * time.Millisecond)
	raw, applyErr := run("/opt/vpn-node/current/fvpn-updater", "apply", "--config", "/etc/vpn-node/updater.json", "--pending", "/var/lib/vpn-node/update-staging/pending-release.json")
	note("apply_process", string(raw))
	noteJSON(&rec, "apply_exit", commandExit(applyErr))
	_, _ = run("touch", "/var/lib/vpn-node/e2e/upgrade-observe.done")
	observerErr := observer.Wait()
	trace, _ := run("cat", "/var/lib/vpn-node/e2e/upgrade-observe.jsonl")
	note("handshake_trace", string(trace))
	status, statusErr := run("/opt/vpn-node/current/fvpn-updater", "status", "--config", "/etc/vpn-node/updater.json")
	note("durable_result", string(status))
	checkpoint, cpErr := run("cat", "/var/lib/vpn-node/state/meter.checkpoint")
	note("checkpoint_after", string(checkpoint))
	after, _ := run("readlink", "/opt/vpn-node/current")
	note("current_before", string(before))
	note("current_after", string(after))
	executable, _ := run("sh", "-c", `p=$(systemctl show -p MainPID --value fvpn-agent.service); readlink /proc/$p/exe`)
	note("running_executable", string(executable))
	check, _ := run("/usr/local/bin/fvpn-node-install", "check", "--json")
	note("post_switch_check", string(check))
	usageAfter, _ := s.UsageLedgerSnapshot(ctx, "installer-node")
	noteJSON(&rec, "usage_after", usageAfter)
	_, copyErr := run("/usr/local/bin/fvpn-e2e-nodeconfig", "journal-copy")
	dest := filepath.Join(r.lab.root, ".agent/tmp/followup19", r.runID+"-installer-journal.db")
	receipt, exportErr := s.ExportContainerFile(ctx, "installer-node", "/var/lib/vpn-node/e2e/journal.snapshot.db", dest)
	noteJSON(&rec, "journal_snapshot", receipt)
	logs, _ := run("journalctl", "-u", "fvpn-agent", "-u", "fvpn-updater", "--no-pager", "-n", "160")
	note("agent_updater_journal", string(logs))
	var state struct {
		State  string            `json:"state"`
		Health map[string]string `json:"health"`
	}
	healthy := json.Unmarshal(status, &state) == nil && state.State == "succeeded"
	for _, key := range []string{"processes", "identity_and_state", "internet_only", "dns_policy", "tunnel_probe"} {
		healthy = healthy && state.Health[key] == "passed"
	}
	correlated := false
	var cp map[string]json.RawMessage
	_ = json.Unmarshal(checkpoint, &cp)
	for _, line := range bytes.Split(trace, []byte("\n")) {
		var sample struct {
			At         time.Time         `json:"at"`
			Boot       string            `json:"boot_id"`
			Latch      string            `json:"latch"`
			Checkpoint string            `json:"checkpoint"`
			Notes      map[string]string `json:"wait_notes"`
		}
		if json.Unmarshal(line, &sample) != nil {
			continue
		}
		var seen struct {
			Barrier string `json:"barrier_id"`
		}
		_ = json.Unmarshal([]byte(sample.Checkpoint), &seen)
		for _, n := range sample.Notes {
			var wait struct {
				Boot     string    `json:"boot_id"`
				Identity string    `json:"identity"`
				Started  time.Time `json:"started"`
				Deadline time.Time `json:"deadline"`
			}
			if json.Unmarshal([]byte(n), &wait) == nil && seen.Barrier != "" && seen.Barrier == wait.Identity && sample.Latch == wait.Identity && sample.Boot == wait.Boot && !sample.At.Before(wait.Started) && sample.At.Before(wait.Deadline) {
				correlated = true
				note("correlated_checkpoint", sample.Checkpoint)
			}
		}
	}
	note("correlated_with_original_boot_deadline", boolText(correlated))
	sealedOK := false
	if copyErr == nil && exportErr == nil {
		db, err := sql.Open("sqlite", "file:"+dest+"?mode=ro")
		if err == nil {
			defer db.Close()
			var integrity string
			_ = db.QueryRow("PRAGMA integrity_check").Scan(&integrity)
			note("journal_integrity", integrity)
			rows, err := db.Query("SELECT payload,payload_hash FROM result_outbox ORDER BY seq")
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var body []byte
					var hash string
					if rows.Scan(&body, &hash) != nil {
						continue
					}
					note("sealed_receipt", string(body))
					sum, err := canonjson.HashHex(body, "payload_hash")
					note("receipt_hash_verified", boolText(err == nil && sum == hash))
					var v struct {
						State string                     `json:"state"`
						Usage map[string]json.RawMessage `json:"usage_checkpoint"`
					}
					_ = json.Unmarshal(body, &v)
					match := v.State == "succeeded" && err == nil && sum == hash
					for _, key := range []string{"closed_meter_epoch_id", "last_durable_event_seq", "central_acked_through"} {
						match = match && len(cp[key]) > 0 && bytes.Equal(cp[key], v.Usage[key])
					}
					var newEpoch, oldEpoch string
					_ = json.Unmarshal(v.Usage["new_meter_epoch_id"], &newEpoch)
					_ = json.Unmarshal(cp["closed_meter_epoch_id"], &oldEpoch)
					sealedOK = sealedOK || (match && newEpoch != "" && newEpoch != oldEpoch)
				}
			}
		}
	}
	if state.State == "failed" {
		replay, replayErr := run("/opt/vpn-node/current/fvpn-updater", "apply", "--config", "/etc/vpn-node/updater.json", "--pending", "/var/lib/vpn-node/update-staging/pending-release.failed.json")
		recoverOut, recoverErr := run("/opt/vpn-node/current/fvpn-updater", "recover", "--config", "/etc/vpn-node/updater.json")
		note("failed_apply_replay", string(replay))
		noteJSON(&rec, "failed_apply_replay_exit", commandExit(replayErr))
		note("failed_recover", string(recoverOut))
		noteJSON(&rec, "failed_recover_exit", commandExit(recoverErr))
	}
	note("owner", "updater / agent-core; sealed failure remains failure; no checkpoint supplied")
	if stageOK && applyErr == nil && statusErr == nil && healthy && cpErr == nil && observerErr == nil && correlated && sealedOK && usageErr == nil && !bytes.Equal(before, after) && strings.Contains(string(executable), strings.TrimSpace(string(after))) {
		rec.Status = StatusPass
	}
}

// Evidence-only SQLite source snapshot, including WAL. It never adopts a stream.
func (r *runner) usageCopy19(ctx context.Context, s *Stack, label string) (string, error) {
	_, err := r.lab.docker(s.installerContainer(), "/usr/local/bin/fvpn-e2e-nodeconfig", "usage-copy")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(r.lab.root, ".agent/tmp/followup19", r.runID+"-usage-"+label+".db")
	_, err = s.ExportContainerFile(ctx, "installer-node", "/var/lib/vpn-node/e2e/usage.snapshot.db", dest)
	return dest, err
}

func retainedUsage19(before, after, origin string) (bool, error) {
	a, e := sql.Open("sqlite", "file:"+before+"?mode=ro")
	if e != nil {
		return false, e
	}
	defer a.Close()
	b, e := sql.Open("sqlite", "file:"+after+"?mode=ro")
	if e != nil {
		return false, e
	}
	defer b.Close()
	var ack, next int64
	if e = a.QueryRow("SELECT acked_through,next_seq FROM meta WHERE incarnation_id=?", origin).Scan(&ack, &next); e != nil {
		return false, e
	}
	var retiredAck, last int64
	if e = b.QueryRow("SELECT acked_through,last_seq FROM retired_streams WHERE incarnation_id=?", origin).Scan(&retiredAck, &last); e != nil {
		return false, e
	}
	if retiredAck < ack || last < next-1 {
		return false, fmt.Errorf("origin watermarks regressed")
	}
	rows, e := a.Query("SELECT seq,payload,payload_sha256 FROM events")
	if e != nil {
		return false, e
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var seq int64
		var payload, retained []byte
		var hash, retainedHash string
		if e = rows.Scan(&seq, &payload, &hash); e != nil {
			return false, e
		}
		if e = b.QueryRow("SELECT payload,payload_sha256 FROM retired_events WHERE incarnation_id=? AND seq=?", origin, seq).Scan(&retained, &retainedHash); e != nil {
			return false, e
		}
		if !bytes.Equal(payload, retained) || hash != retainedHash {
			return false, fmt.Errorf("origin canonical event changed")
		}
		count++
	}
	return count > 0, rows.Err()
}
