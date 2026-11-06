package e2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const recoveryTestCommit = "1111111111111111111111111111111111111111" // TEST ONLY

func recoveryHeartbeat(at time.Time, boot string) string {
	return fmt.Sprintf("sample_at %s\nagent_pid 17\nlock_holder_pids 18 19 \nagent_executable TEST_ONLY_PRIVATE_PATH\nboot_id %s\nmono_ns 9007199254740993\nwall_unix_ns 1790000000000000000\npolicy_expires_mono_ns 0\ngeneration 100\nserving 0\nbaseline_ok 1\nreason policy_missing\nforwarding_blocked 1\nsample_end\n", at.Format(time.RFC3339Nano), boot)
}

func recoveryLog(at time.Time, fields map[string]any) []byte {
	fields["time"] = at.Format(time.RFC3339Nano)
	raw, _ := json.Marshal(fields)
	return append(raw, '\n')
}

func TestSEC02_RecoveryContextAllowlistAndHistoricalBinding(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run, boot := uuid.NewString(), uuid.NewString()
	var agent []byte
	for _, at := range []time.Time{start.Add(-time.Second), start, end.Add(time.Second)} {
		agent = append(agent, recoveryLog(at, map[string]any{
			"msg": "apply", "err": "TC_FOREIGN_FILTER", "tc_read_failure": "command_exit", "tc_read_context_state_at_return": "deadline", "tc_read_exit_code": 7,
			"tc_read_interface": "fvup0", "tc_read_attachment": "7560:", "argv": "TEST_ONLY_PRIVATE_ARGV", "stderr": "TEST_ONLY_PRIVATE_STDERR", "token": "TEST_ONLY_PRIVATE_TOKEN",
		})...)
	}
	watchdog := recoveryLog(start.Add(time.Second), map[string]any{"msg": "baseline decision", "writer": "watchdog", "cause": "heartbeat_stale", "boot_id": boot, "mono_ns": uint64(9007199254740993), "serving": false, "baseline_ok": true})
	inputs := map[string]recoveryInput{
		"heartbeats": {Data: []byte(recoveryHeartbeat(start, boot)), Available: true},
		"agent":      {Data: agent, Available: true}, "watchdog": {Data: watchdog, Available: true},
	}
	out := projectRecoveryContext(run, recoveryTestCommit, start, end, true, true, inputs)
	if !out.BindingAvailable || out.RunID != run || out.Source != recoveryTestCommit || !out.WindowAvailable || !out.Started || !out.Joined {
		t.Fatalf("missing actual run binding: %+v", out)
	}
	a := out.Streams["agent"]
	if len(a.Records) != 3 || a.OutsideWindow != 2 {
		t.Fatalf("historical context discarded: %+v", a)
	}
	for i, position := range []string{"before", "within", "after"} {
		row := a.Records[i]
		if row["window_position"] != position || row["inside_collection_window"] != (i == 1) || row["tc_read_context_state_at_return"] != "deadline" || row["tc_read_failure"] != "command_exit" || row["tc_read_exit_code"] != uint64(7) {
			t.Fatalf("ambiguous event: %+v", row)
		}
	}
	h := out.Streams["heartbeats"].Records[0]
	if h["heartbeat_fields_available"] != true || h["mono_ns"] != "9007199254740993" || h["serving"] != false || h["forwarding_blocked"] != true {
		t.Fatalf("heartbeat fields: %+v", h)
	}
	w := out.Streams["watchdog"].Records[0]
	if w["boot_id"] != boot || w["mono_ns"] != "9007199254740993" {
		t.Fatalf("boot/time provenance lost: %+v", w)
	}
	raw, err := json.Marshal(out)
	if err != nil || strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
		t.Fatalf("unsafe output: %s %v", raw, err)
	}
	if !strings.Contains(out.Scope, "not later log reads") || !strings.Contains(out.Scope, "same-launch") {
		t.Fatal("missing observation scope")
	}
}

func TestSEC02_RecoveryContextBoundsNewestAndPartialInputs(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var agent []byte
	for i := range recoveryRecordLimit + 20 {
		agent = append(agent, recoveryLog(start.Add(time.Duration(i)*time.Second), map[string]any{"msg": "apply", "generation": i})...)
	}
	out := projectRecoveryContext(uuid.NewString(), recoveryTestCommit, start, start.Add(time.Hour), true, false, map[string]recoveryInput{
		"agent":      {Data: agent, Available: false},
		"heartbeats": {Data: []byte("sample_end\nsample_at " + start.Format(time.RFC3339Nano) + "\nagent_pid 17\nsample_end\nsample_at incomplete\n"), Available: false},
	})
	a := out.Streams["agent"]
	if a.Available || !a.Truncated || a.DroppedRecords != 20 || len(a.Records) != recoveryRecordLimit || a.Records[0]["generation"] != "20" || a.Records[len(a.Records)-1]["generation"] != fmt.Sprint(recoveryRecordLimit+19) {
		t.Fatalf("latest bounded context lost: %+v", a)
	}
	h := out.Streams["heartbeats"]
	if h.Incomplete != 3 || len(h.Records) != 1 || h.Records[0]["heartbeat_fields_available"] != false || out.Streams["watchdog"].Available {
		t.Fatalf("partial input presented as complete: %+v", out.Streams)
	}
	large := strings.Repeat("x", recoveryInputLimit+30) + "\n" + strings.Repeat("y", recoveryLineLimit+1) + "\nmalformed\n" + string(recoveryLog(start, map[string]any{"msg": "apply", "tc_read_failure": "read_error"}))
	out = projectRecoveryContext(uuid.NewString(), recoveryTestCommit, start, start, true, true, map[string]recoveryInput{"agent": {Data: []byte(large), Available: true}})
	a = out.Streams["agent"]
	if a.ScannedBytes > recoveryInputLimit || !a.Truncated || a.Malformed != 2 || a.Incomplete != 1 || len(a.Records) != 1 || a.Records[0]["tc_read_failure"] != "read_error" {
		t.Fatalf("scan cap/partial status: %+v", a)
	}
	// A pathological fractional second must not amplify the bounded projection.
	stamp := "2026-01-01T00:00:00." + strings.Repeat("0", 12000) + "Z"
	line := fmt.Sprintf("{\"time\":%q,\"msg\":\"apply\"}\n", stamp)
	out = projectRecoveryContext(uuid.NewString(), recoveryTestCommit, start, start, true, true, map[string]recoveryInput{"agent": {Data: []byte(strings.Repeat(line, 100)), Available: true}})
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil || len(raw) > 256<<10 {
		t.Fatalf("unbounded timestamp/output: bytes=%d err=%v", len(raw), err)
	}
}

func TestSEC02_RecoveryContextRejectsUnsafeFieldsAndMissingBinding(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	unsafe := recoveryLog(start, map[string]any{"msg": "apply", "err": "TEST_ONLY_PRIVATE", "tc_read_failure": "TEST_ONLY_PRIVATE", "tc_read_context_state_at_return": "TEST_ONLY_PRIVATE", "tc_read_interface": "unsafe/interface", "tc_read_attachment": "unsafe selector", "tc_read_exit_code": -1, "mono_ns": 1.5, "serving": "false", "boot_id": "invalid"})
	out := projectRecoveryContext("TEST_ONLY_PRIVATE", "TEST_ONLY_PRIVATE", time.Time{}, start, false, false, map[string]recoveryInput{"agent": {Data: unsafe}})
	if out.BindingAvailable || out.RunID != "" || out.Source != "" || out.WindowAvailable || out.Started || out.Joined {
		t.Fatalf("invented observation binding: %+v", out)
	}
	row := out.Streams["agent"].Records[0]
	if len(row) != 4 || row["window_position"] != "unavailable" {
		t.Fatalf("unsafe fields retained: %+v", row)
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "PRIVATE") || strings.Contains(string(raw), "unsafe") {
		t.Fatalf("unsafe input leaked: %s", raw)
	}
	for _, value := range []string{"18446744073709551616", "-1", "TEST_ONLY_PRIVATE"} {
		row := map[string]any{}
		projectHeartbeatField(row, "generation", value)
		projectHeartbeatField(row, "lock_holder_pids", value)
		projectHeartbeatField(row, "reason", value)
		if len(row) != 0 {
			t.Fatalf("accepted invalid heartbeat fields: %+v", row)
		}
	}
}
