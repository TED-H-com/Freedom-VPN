package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
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

// Use the handlers selected by the real service entrypoints, with an explicit
// record time so collection-window assertions do not depend on wall-clock races.
func recoveryServiceLog(t *testing.T, service string, at time.Time, msg string, attrs ...slog.Attr) []byte {
	t.Helper()
	var buf bytes.Buffer
	var handler slog.Handler
	switch service {
	case "agent":
		handler = agent.RedactHandler{Next: slog.NewTextHandler(&buf, nil)}
	case "watchdog":
		handler = slog.NewJSONHandler(&buf, nil)
	default:
		t.Fatalf("unknown test service %q", service)
	}
	record := slog.NewRecord(at, slog.LevelInfo, msg, 0)
	record.AddAttrs(attrs...)
	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSEC02_RecoveryContextAllowlistAndHistoricalBinding(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run, boot := uuid.NewString(), uuid.NewString()
	var agent []byte
	for _, at := range []time.Time{start.Add(-time.Second), start, end.Add(time.Second)} {
		agent = append(agent, recoveryServiceLog(t, "agent", at, "apply",
			slog.String("err", "TC_FOREIGN_FILTER"), slog.String("tc_read_failure", "command_exit"), slog.String("tc_read_context_state_at_return", "deadline"), slog.Int("tc_read_exit_code", 7),
			slog.String("tc_read_interface", "fvup0"), slog.String("tc_read_attachment", "7560:"), slog.String("argv", "TEST_ONLY_PRIVATE_ARGV"), slog.String("stderr", "TEST_ONLY_PRIVATE_STDERR"), slog.String("token", "TEST_ONLY_PRIVATE_TOKEN"),
		)...)
	}
	watchdog := recoveryServiceLog(t, "watchdog", start.Add(time.Second), "baseline decision", slog.String("writer", "watchdog"), slog.String("cause", "heartbeat_stale"), slog.String("boot_id", boot), slog.Uint64("mono_ns", 9007199254740993), slog.Bool("serving", false), slog.Bool("baseline_ok", true))
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

func TestSEC02_RecoveryContextTextHandlerQuotedFields(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	boot := uuid.NewString()
	var input []byte
	for _, msg := range []string{"apply", "baseline decision", "baseline write", "baseline canceled"} {
		input = append(input, recoveryServiceLog(t, "agent", start, msg,
			slog.String("writer", "agent"), slog.String("phase", "end"), slog.String("cause", "policy_expired"), slog.String("boot_id", boot),
			slog.String("err", "serving verification timed out"), slog.Uint64("mono_ns", 9007199254740993), slog.Uint64("generation", ^uint64(0)),
			slog.Bool("serving", false), slog.Bool("baseline_ok", true), slog.Bool("succeeded", true),
			slog.String("TEST_ONLY_PRIVATE key=\"\\\n", "TEST_ONLY_PRIVATE value=\"\\\n\t\x00"),
			slog.String("opaque", `TEST_ONLY_PRIVATE\path`), slog.String("token", "TEST_ONLY_PRIVATE_TOKEN"),
		)...)
	}
	if !bytes.Contains(input, []byte(`msg="baseline decision"`)) || !bytes.Contains(input, []byte(`\x00`)) || bytes.Contains(input, []byte("TEST_ONLY_PRIVATE_TOKEN")) {
		t.Fatal("fixture did not exercise production quoting and redaction")
	}
	original := bytes.Clone(input)
	out := projectRecoveryContext(uuid.NewString(), recoveryTestCommit, start, start, true, true, map[string]recoveryInput{"agent": {Data: input, Available: true}})
	s := out.Streams["agent"]
	if s.Malformed != 0 || s.OutsideWindow != 0 || len(s.Records) != 4 || !bytes.Equal(input, original) {
		t.Fatalf("text input lost or changed: %+v", s)
	}
	for i, msg := range []string{"apply", "baseline decision", "baseline write", "baseline canceled"} {
		row := s.Records[i]
		for key, want := range map[string]any{"event": msg, "writer": "agent", "phase": "end", "cause": "policy_expired", "boot_id": boot, "err": "serving verification timed out", "mono_ns": "9007199254740993", "generation": "18446744073709551615", "serving": false, "baseline_ok": true, "succeeded": true} {
			if row[key] != want {
				t.Fatalf("text field %s: got %v, want %v", key, row[key], want)
			}
		}
	}
	raw, err := json.Marshal(out)
	if err != nil || bytes.Contains(raw, []byte("TEST_ONLY_PRIVATE")) || bytes.Contains(raw, []byte("opaque")) || bytes.Contains(raw, []byte("token")) {
		t.Fatalf("unsafe text projection: %s %v", raw, err)
	}
}

func TestSEC02_RecoveryContextRejectsMalformedText(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := strings.TrimSuffix(string(recoveryServiceLog(t, "agent", start, "apply")), "\n")
	for name, suffix := range map[string]string{
		"duplicate":         " msg=apply",
		"quoted duplicate":  ` "msg"=apply`,
		"unknown duplicate": ` opaque=TEST_ONLY_PRIVATE "opaque"=TEST_ONLY_PRIVATE`,
		"missing equals":    " TEST_ONLY_PRIVATE",
		"empty bare value":  " opaque=",
		"bare equals":       " opaque=TEST_ONLY_PRIVATE=extra",
		"unterminated":      ` opaque="TEST_ONLY_PRIVATE`,
		"invalid escape":    ` opaque="TEST_ONLY_PRIVATE\q"`,
		"trailing junk":     ` opaque="TEST_ONLY_PRIVATE"junk`,
		"raw tab":           " opaque=TEST_ONLY_PRIVATE\tvalue",
	} {
		t.Run(name, func(t *testing.T) {
			out := projectRecoveryContext(uuid.NewString(), recoveryTestCommit, start, start, true, true, map[string]recoveryInput{"agent": {Data: []byte(base + suffix + "\n"), Available: true}})
			s := out.Streams["agent"]
			raw, err := json.Marshal(out)
			if s.Malformed != 1 || len(s.Records) != 0 || err != nil || bytes.Contains(raw, []byte("TEST_ONLY_PRIVATE")) {
				t.Fatalf("malformed input retained or leaked: %s %v", raw, err)
			}
		})
	}
	input := base + ` tc_read_exit_code=256 generation=18446744073709551616 serving="true" mono_ns=1.5` + "\n"
	out := projectRecoveryContext(uuid.NewString(), recoveryTestCommit, start, start, true, true, map[string]recoveryInput{"agent": {Data: []byte(input), Available: true}})
	s := out.Streams["agent"]
	if s.Malformed != 0 || len(s.Records) != 1 || len(s.Records[0]) != 4 {
		t.Fatalf("text format widened field types or bounds: %+v", s)
	}
}

func TestSEC02_RecoveryContextBoundsNewestAndPartialInputs(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var agent []byte
	for i := range recoveryRecordLimit + 20 {
		agent = append(agent, recoveryServiceLog(t, "agent", start.Add(time.Duration(i)*time.Second), "apply", slog.Int("generation", i))...)
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
	oversized := recoveryServiceLog(t, "agent", start, "apply", slog.String("stderr", strings.Repeat("y", recoveryLineLimit+1)))
	large := strings.Repeat("x", recoveryInputLimit+30) + "\n" + string(oversized) + "malformed\n" + string(recoveryServiceLog(t, "agent", start, "apply", slog.String("tc_read_failure", "read_error")))
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
