package e2e

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

const recoveryInputLimit = 1 << 20
const recoveryLineLimit = 16 << 10
const recoveryRecordLimit = 256

type recoveryInput struct {
	Data      []byte
	Available bool
}

type recoveryStream struct {
	Available      bool             `json:"available"`
	InputBytes     int              `json:"input_bytes"`
	ScannedBytes   int              `json:"scanned_bytes"`
	Truncated      bool             `json:"truncated"`
	Malformed      int              `json:"malformed_records"`
	Incomplete     int              `json:"incomplete_records"`
	OutsideWindow  int              `json:"outside_window_records"`
	DroppedRecords int              `json:"dropped_records"`
	Records        []map[string]any `json:"records"`
}

type recoveryContext struct {
	RunID            string                    `json:"run_id"`
	Source           string                    `json:"source_commit"`
	BindingAvailable bool                      `json:"binding_available"`
	StartedAt        time.Time                 `json:"observer_started_at"`
	CompletedAt      time.Time                 `json:"observer_finished_at"`
	WindowAvailable  bool                      `json:"window_available"`
	Started          bool                      `json:"observer_started"`
	Joined           bool                      `json:"observer_join_ok"`
	InputLimit       int                       `json:"input_limit_bytes"`
	LineLimit        int                       `json:"line_limit_bytes"`
	RecordLimit      int                       `json:"record_limit_per_stream"`
	Streams          map[string]recoveryStream `json:"streams"`
	Scope            string                    `json:"scope"`
}

var recoveryCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)
var recoveryInterface = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,15}$`)
var recoveryAttachment = regexp.MustCompile(`^[0-9a-fA-F]{1,4}:[0-9a-fA-F]{0,4}$`)

// projectRecoveryContext retains only fixed numeric/enum metadata from already
// collected bytes. It does not re-read state or change any scenario verdict.
// Log timestamps establish a collection window, never a same-launch attestation.
func projectRecoveryContext(runID, source string, start, end time.Time, started, joined bool, inputs map[string]recoveryInput) recoveryContext {
	out := recoveryContext{StartedAt: start, CompletedAt: end, Started: started, Joined: joined,
		InputLimit: recoveryInputLimit, LineLimit: recoveryLineLimit, RecordLimit: recoveryRecordLimit,
		Streams: map[string]recoveryStream{}, Scope: "Diagnostic collection only; window bounds cover the observer, not later log reads. Records outside that window are historical or subsequent context. Partial inputs and omitted records cannot establish success or same-launch identity; context state is observed, not a proven command failure cause."}
	if id, err := uuid.Parse(runID); err == nil && id != uuid.Nil && recoveryCommit.MatchString(source) {
		out.RunID, out.Source, out.BindingAvailable = id.String(), source, true
	}
	out.WindowAvailable = !start.IsZero() && !end.Before(start)
	for _, name := range []string{"heartbeats", "agent", "watchdog"} {
		in := inputs[name]
		s := recoveryStream{Available: in.Available, InputBytes: len(in.Data), Records: []map[string]any{}}
		raw := in.Data
		if len(raw) > recoveryInputLimit {
			// The useful failure is near the end of an old agent log. Retain
			// a bounded suffix, discard its first partial line/sample explicitly.
			raw = raw[len(raw)-recoveryInputLimit:]
			s.Truncated = true
			if i := bytes.IndexByte(raw, '\n'); i >= 0 {
				raw = raw[i+1:]
			} else {
				raw = nil
			}
			s.Incomplete++
		}
		s.ScannedBytes = len(raw)
		appendRecord := func(row map[string]any) {
			stamp, ok := row["time"].(string)
			at, err := time.Parse(time.RFC3339Nano, stamp)
			if !ok || err != nil {
				s.Malformed++
				return
			}
			// Preserve the event instant in a bounded canonical spelling. Old
			// agent logs may contain the startup failure preceding this observer.
			row["time"] = at.UTC().Format(time.RFC3339Nano)
			inside := out.WindowAvailable && !at.Before(start) && !at.After(end)
			row["inside_collection_window"] = inside
			position := "within"
			switch {
			case !out.WindowAvailable:
				position = "unavailable"
			case at.Before(start):
				position = "before"
			case at.After(end):
				position = "after"
			}
			row["window_position"] = position
			if !inside {
				s.OutsideWindow++
			}
			if len(s.Records) == recoveryRecordLimit {
				s.Truncated = true
				s.DroppedRecords++
				copy(s.Records, s.Records[1:])
				s.Records = s.Records[:recoveryRecordLimit-1]
			}
			s.Records = append(s.Records, row)
		}
		var sample map[string]any
		for len(raw) > 0 {
			line, rest, _ := bytes.Cut(raw, []byte{'\n'})
			raw = rest
			if len(line) == 0 {
				continue
			}
			if len(line) > recoveryLineLimit {
				s.Malformed++
				s.Truncated = true
				continue
			}
			if name == "heartbeats" {
				text := string(line)
				if strings.HasPrefix(text, "sample_at ") {
					if sample != nil {
						s.Incomplete++
					}
					sample = map[string]any{"time": strings.TrimPrefix(text, "sample_at ")}
				} else if text == "sample_end" {
					if sample == nil {
						s.Incomplete++
					} else {
						complete := true
						for _, key := range []string{"boot_id", "mono_ns", "wall_unix_ns", "policy_expires_mono_ns", "generation", "serving", "baseline_ok", "reason"} {
							if _, ok := sample[key]; !ok {
								complete = false
							}
						}
						sample["heartbeat_fields_available"] = complete
						if !complete {
							s.Incomplete++
						}
						appendRecord(sample)
					}
					sample = nil
				} else if sample != nil {
					key, value, _ := strings.Cut(text, " ")
					projectHeartbeatField(sample, key, value)
				}
				continue
			}
			fields, ok := decodeRecoveryLog(line)
			if !ok {
				s.Malformed++
				continue
			}
			row := projectRecoveryLog(fields)
			if row != nil {
				appendRecord(row)
			}
		}
		if sample != nil {
			s.Incomplete++
		}
		out.Streams[name] = s
	}
	return out
}

// The agent uses slog.TextHandler; watchdog uses slog.JSONHandler. Normalize
// their fields before applying the same allowlist. TextHandler uses Go quoting,
// including escapes that JSON does not accept, and bare numeric/bool values do
// not retain their original Go type. Never include raw input in parse errors.
func decodeRecoveryLog(line []byte) (map[string]json.RawMessage, bool) {
	if bytes.HasPrefix(bytes.TrimSpace(line), []byte{'{'}) {
		var fields map[string]json.RawMessage
		err := json.Unmarshal(line, &fields)
		return fields, err == nil && fields != nil
	}
	fields := map[string]json.RawMessage{}
	text := string(line)
	for len(text) > 0 {
		key, n, _, ok := recoveryTextToken(text, '=')
		if !ok || n == len(text) || text[n] != '=' {
			return nil, false
		}
		if _, duplicate := fields[key]; duplicate {
			return nil, false
		}
		text = text[n+1:]
		value, n, quoted, ok := recoveryTextToken(text, ' ')
		if !ok || (n < len(text) && text[n] != ' ') {
			return nil, false
		}
		raw, _ := json.Marshal(value)
		if !quoted {
			number, err := strconv.ParseUint(value, 10, 64)
			if (err == nil && strconv.FormatUint(number, 10) == value) || value == "true" || value == "false" {
				raw = []byte(value)
			}
		}
		fields[key] = raw
		text = text[n:]
		if len(text) > 0 {
			text = text[1:]
			if len(text) == 0 {
				return nil, false
			}
		}
	}
	return fields, len(fields) > 0
}

// Read one TextHandler key or value without splitting quoted whitespace or
// treating a bare backslash (which TextHandler permits) as an escape.
func recoveryTextToken(text string, separator byte) (value string, n int, quoted, ok bool) {
	if len(text) == 0 {
		return "", 0, false, false
	}
	if text[0] == '"' {
		for i := 1; i < len(text); i++ {
			switch text[i] {
			case '\\':
				i++
			case '"':
				value, err := strconv.Unquote(text[:i+1])
				return value, i + 1, true, err == nil
			}
		}
		return "", 0, true, false
	}
	for i, r := range text {
		if r == rune(separator) {
			return text[:i], i, false, i > 0
		}
		if r == '"' || r == '=' || r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return "", 0, false, false
		}
	}
	return text, len(text), false, true
}

func recoveryReason(value string) bool {
	if api.ForwardingBlockReason(value).Valid() {
		return true
	}
	if code, ok := strings.CutPrefix(value, "verification_"); ok && api.ErrorCode(code).Valid() {
		return true
	}
	switch value {
	case "serving", "identity_frozen", "member_path_unconfigured", "adoption_pending", "clock_unavailable", "heartbeat_unreadable", "boot_changed", "heartbeat_in_future", "heartbeat_stale", "baseline_unverified", "policy_expired", "serving_without_deadline", "baseline_unreadable", "baseline_missing", "heartbeat_refreshed", "apply_rollback", "apply_quarantine", "identity_revoked", "updater_barrier", "updater_recovery", "conntrack_blocked", "shutdown", "policy_deadline", "non_serving", "probe_install", "probe_clear", "foreign_tc_quarantine", "boot_unverified":
		return true
	}
	return false
}

func projectHeartbeatField(row map[string]any, key, value string) {
	switch key {
	case "agent_pid":
		if n, err := strconv.ParseUint(value, 10, 31); err == nil && n > 0 {
			row[key] = n
		}
	case "lock_holder_pids":
		fields := strings.Fields(value)
		if len(fields) > 8 {
			return
		}
		pids := []uint64{}
		for _, text := range fields {
			n, err := strconv.ParseUint(text, 10, 31)
			if err != nil || n == 0 {
				return
			}
			pids = append(pids, n)
		}
		row[key] = pids
	case "boot_id":
		if id, err := uuid.Parse(value); err == nil && id != uuid.Nil {
			row[key] = id.String()
		}
	case "mono_ns", "wall_unix_ns", "policy_expires_mono_ns", "generation":
		if n, err := strconv.ParseUint(value, 10, 64); err == nil {
			row[key] = strconv.FormatUint(n, 10)
		}
	case "baseline_ok", "serving", "forwarding_blocked":
		if value == "0" || value == "1" {
			row[key] = value == "1"
		}
	case "reason":
		if recoveryReason(value) {
			row[key] = value
		}
	}
}

func projectRecoveryLog(fields map[string]json.RawMessage) map[string]any {
	text := func(key string) string { var s string; _ = json.Unmarshal(fields[key], &s); return s }
	msg := text("msg")
	switch msg {
	case "apply", "baseline decision", "baseline write", "baseline canceled":
	default:
		return nil
	}
	row := map[string]any{"time": text("time"), "event": msg}
	projectHeartbeatField(row, "boot_id", text("boot_id"))
	for _, key := range []string{"writer", "phase", "cause", "err", "tc_read_failure", "tc_read_context_state_at_return", "tc_read_interface", "tc_read_attachment"} {
		s := text(key)
		ok := false
		switch key {
		case "writer":
			ok = s == "agent" || s == "agent_apply" || s == "watchdog"
		case "phase":
			ok = s == "begin" || s == "end"
		case "cause":
			ok = recoveryReason(s)
		case "err":
			ok = api.ErrorCode(s).Valid() || s == "apply failed" || s == "serving verification timed out" || s == "serving verification canceled"
		case "tc_read_failure":
			ok = s == "deadline" || s == "canceled" || s == "command_exit" || s == "read_error"
		case "tc_read_context_state_at_return":
			ok = s == "none" || s == "deadline" || s == "canceled"
		case "tc_read_interface":
			ok = recoveryInterface.MatchString(s) && Redact(s) == s && !strings.HasPrefix(s, "github_pat_")
		case "tc_read_attachment":
			ok = recoveryAttachment.MatchString(s) || s == "qdisc" || s == "class" || s == "root" || s == "ingress" || s == "egress"
		}
		if ok {
			row[key] = s
		}
	}
	for _, key := range []string{"mono_ns", "heartbeat_mono_ns", "policy_expires_mono_ns", "generation", "start_mono_ns", "end_mono_ns", "tc_read_exit_code"} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(string(raw), 10, 64)
		if err == nil {
			if key == "tc_read_exit_code" {
				if n <= 255 {
					row[key] = n
				}
			} else {
				row[key] = strconv.FormatUint(n, 10)
			}
		}
	}
	for _, key := range []string{"serving", "baseline_ok", "succeeded"} {
		if raw, ok := fields[key]; ok && (string(raw) == "true" || string(raw) == "false") {
			row[key] = string(raw) == "true"
		}
	}
	return row
}
