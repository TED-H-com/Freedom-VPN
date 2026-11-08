package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// A complete Apply contains many command/resource observations. Give its line
// more room than ordinary logs, while bounding both the container read and the
// public projection. No raw timing stream or resource text is exported.
const updateTimingInputLimit = 4 << 20
const updateTimingLineLimit = 1 << 20
const updateTimingRecordLimit = 64
const updateTimingCommandLimit = 8
const updateTimingPath = "/var/lib/vpn-node/e2e/enforcement-timing.jsonl"

type recoveryTimingStream struct {
	recoveryStream
	InputLimit      int    `json:"input_limit_bytes"`
	LineLimit       int    `json:"line_limit_bytes"`
	RecordLimit     int    `json:"record_limit"`
	CommandLimit    int    `json:"command_limit_per_record"`
	DroppedCommands int    `json:"dropped_commands"`
	Scope           string `json:"scope"`
}

type timingOutput struct {
	data     []byte
	overflow bool
}

func (w *timingOutput) Write(p []byte) (int, error) {
	n := len(p)
	room := updateTimingInputLimit + 1 - len(w.data)
	if len(p) > room {
		p = p[:room]
		w.overflow = true
	}
	w.data = append(w.data, p...)
	return n, nil
}

// tail bounds the subprocess output; the writer independently caps retained
// memory even if the command misbehaves. stderr is never parsed as evidence.
func readUpdateTiming(ctx context.Context, node string) recoveryInput {
	cmd := exec.CommandContext(ctx, "docker", "exec", node, "tail", "-c", strconv.Itoa(updateTimingInputLimit+1), updateTimingPath)
	var out timingOutput
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	err := cmd.Run()
	return recoveryInput{Data: out.data, Available: err == nil && !out.overflow}
}

func projectUpdateTiming(in recoveryInput, start, end time.Time) recoveryTimingStream {
	out := recoveryTimingStream{recoveryStream: recoveryStream{Available: in.Available, InputBytes: len(in.Data), Records: []map[string]any{}}, InputLimit: updateTimingInputLimit, LineLimit: updateTimingLineLimit, RecordLimit: updateTimingRecordLimit, CommandLimit: updateTimingCommandLimit,
		Scope: "Bounded suffix of TEST ONLY producer measurements. Input bytes count the bounded read, not the full file. Non-Apply records are intentionally omitted; projected sequence gaps alone do not prove lost producer records. Only Apply shape/install command observations are projected. Producer run/boot/sequence and clock samples must be joined before comparisons; truncation does not preserve a complete sequence. Zero operation stamps mean unavailable; zero or unordered clock samples cannot establish timing. Command stamps bracket userspace observations, not kernel events or a proven timeout cause."}
	raw := in.Data
	if len(raw) > updateTimingInputLimit {
		raw = raw[len(raw)-updateTimingInputLimit:]
		out.Truncated = true
		out.Incomplete++
		if i := bytes.IndexByte(raw, '\n'); i >= 0 {
			raw = raw[i+1:]
		} else {
			raw = nil
		}
	}
	out.ScannedBytes = len(raw)
	for len(raw) > 0 {
		line, rest, complete := bytes.Cut(raw, []byte{'\n'})
		raw = rest
		if !complete {
			out.Incomplete++
			break
		}
		if len(line) == 0 {
			continue
		}
		if len(line) > updateTimingLineLimit {
			out.Malformed++
			out.Truncated = true
			continue
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(line, &fields) != nil || fields == nil {
			out.Malformed++
			continue
		}
		source, ok := timingString(fields["source"])
		if !ok {
			out.Incomplete++
			continue
		}
		if source != "apply" {
			continue
		}
		row, dropped, ok := projectTimingRow(fields)
		if !ok {
			out.Incomplete++
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, row["time"].(string))
		position := "within"
		switch {
		case start.IsZero() || end.Before(start):
			position = "unavailable"
		case at.Before(start):
			position = "before"
		case at.After(end):
			position = "after"
		}
		row["window_position"], row["inside_collection_window"] = position, position == "within"
		if position != "within" {
			out.OutsideWindow++
		}
		out.DroppedCommands += dropped
		if dropped > 0 {
			out.Truncated = true
		}
		if len(out.Records) == updateTimingRecordLimit {
			out.DroppedRecords++
			out.Truncated = true
			copy(out.Records, out.Records[1:])
			out.Records = out.Records[:updateTimingRecordLimit-1]
		}
		out.Records = append(out.Records, row)
	}
	return out
}

type timingFields map[string]json.RawMessage

func timingObject(raw json.RawMessage) (timingFields, bool) {
	var v timingFields
	err := json.Unmarshal(raw, &v)
	return v, err == nil && v != nil
}
func timingString(raw json.RawMessage) (string, bool) {
	var v string
	err := json.Unmarshal(raw, &v)
	return v, err == nil && len(raw) > 0 && raw[0] == '"'
}
func timingBool(raw json.RawMessage) (any, bool) {
	if bytes.Equal(raw, []byte("true")) {
		return true, true
	}
	if bytes.Equal(raw, []byte("false")) {
		return false, true
	}
	return nil, false
}
func timingDecimal(raw json.RawMessage, unsigned bool) (any, bool) {
	v, ok := timingString(raw)
	if !ok {
		return nil, false
	}
	if unsigned {
		n, e := strconv.ParseUint(v, 10, 64)
		return v, e == nil && strconv.FormatUint(n, 10) == v
	}
	n, e := strconv.ParseInt(v, 10, 64)
	return v, e == nil && n >= 0 && strconv.FormatInt(n, 10) == v
}
func timingInt(raw json.RawMessage, low, high int64) (any, bool) {
	n, e := strconv.ParseInt(string(raw), 10, 64)
	return n, e == nil && n >= low && n <= high && strconv.FormatInt(n, 10) == string(raw)
}
func timingEnum(raw json.RawMessage, values ...string) (any, bool) {
	v, ok := timingString(raw)
	if ok {
		for _, want := range values {
			if v == want {
				return v, true
			}
		}
	}
	return nil, false
}
func timingUUID(raw json.RawMessage) (any, bool) {
	s, ok := timingString(raw)
	id, e := uuid.Parse(s)
	return s, ok && e == nil && id != uuid.Nil && id.String() == s
}
func timingStamp(raw json.RawMessage) (any, bool) {
	f, ok := timingObject(raw)
	if !ok {
		return nil, false
	}
	ns, nok := timingDecimal(f["ns"], false)
	available, aok := timingBool(f["available"])
	return map[string]any{"ns": ns, "available": available}, nok && aok
}
func timingBracket(raw json.RawMessage) (any, bool) {
	f, ok := timingObject(raw)
	if !ok {
		return nil, false
	}
	before, bok := timingStamp(f["before"])
	after, aok := timingStamp(f["after"])
	return map[string]any{"before": before, "after": after}, bok && aok
}

func projectTimingRow(f timingFields) (map[string]any, int, bool) {
	out := map[string]any{"source": "apply"}
	good := true
	set := func(key string, value any, ok bool) {
		if !ok {
			good = false
			return
		}
		out[key] = value
	}
	for _, key := range []string{"run_id", "boot_id"} {
		v, ok := timingUUID(f[key])
		set(key, v, ok)
	}
	v, ok := timingDecimal(f["sequence"], true)
	set("sequence", v, ok && v != "0")
	anchor, aok := timingObject(f["anchor"])
	if !aok {
		return nil, 0, false
	}
	a := map[string]any{}
	for _, key := range []string{"mono_before_ns", "mono_after_ns"} {
		v, ok := timingDecimal(anchor[key], false)
		if !ok {
			good = false
		}
		a[key] = v
	}
	wall, wok := timingString(anchor["wall"])
	at, err := time.Parse(time.RFC3339Nano, wall)
	if !wok || err != nil || at.IsZero() {
		return nil, 0, false
	}
	a["wall"] = at.UTC().Format(time.RFC3339Nano)
	out["anchor"] = a
	out["time"] = a["wall"]
	t, tok := timingObject(f["timing"])
	if !tok {
		return nil, 0, false
	}
	for _, key := range []string{"version"} {
		v, ok := timingInt(t[key], 1, 1)
		set(key, v, ok)
	}
	v, ok = timingEnum(t["clock"], "CLOCK_MONOTONIC")
	set("clock", v, ok)
	v, ok = timingEnum(t["operation"], "apply")
	set("operation", v, ok)
	v, ok = timingDecimal(t["candidate_generation"], true)
	set("candidate_generation", v, ok)
	for _, key := range []string{"start_ns", "end_ns"} {
		v, ok := timingDecimal(t[key], false)
		set(key, v, ok)
	}
	for _, key := range []string{"succeeded", "verified"} {
		v, ok := timingBool(t[key])
		set(key, v, ok)
	}
	d, dok := timingObject(t["deadline"])
	if !dok {
		return nil, 0, false
	}
	deadline := map[string]any{}
	for _, key := range []string{"granularity_ns", "install_bound_ns", "apply_start_ns", "render_start_ns", "install_start_ns", "installed_by_ns", "release_start_ns", "release_end_ns"} {
		v, ok := timingDecimal(d[key], false)
		if !ok {
			good = false
		}
		deadline[key] = v
	}
	for _, key := range []string{"install_ok", "guard_released"} {
		v, ok := timingBool(d[key])
		if !ok {
			good = false
		}
		deadline[key] = v
	}
	out["deadline"] = deadline
	commands := []map[string]any{}
	dropped := 0
	var rawCommands []json.RawMessage
	// Commands are optional in old/disabled measurements; absence remains explicit.
	if raw, exists := t["commands"]; exists {
		if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &rawCommands) != nil {
			return nil, 0, false
		}
		out["commands_available"] = true
		for _, raw := range rawCommands {
			command, cok := timingObject(raw)
			if !cok {
				return nil, 0, false
			}
			phase, pok := timingEnum(command["phase"], "prepare", "shape", "install", "retire", "release", "verify", "rollback", "revoke", "quarantine")
			if !pok {
				return nil, 0, false
			}
			if phase != "shape" && phase != "install" {
				continue
			}
			if len(commands) == updateTimingCommandLimit {
				dropped++
				continue
			}
			row, ok := projectTimingCommand(command)
			if !ok {
				return nil, 0, false
			}
			commands = append(commands, row)
		}
	} else {
		out["commands_available"] = false
	}
	out["commands"], out["dropped_commands"] = commands, dropped
	return out, dropped, good
}

func projectTimingCommand(f timingFields) (map[string]any, bool) {
	out := map[string]any{}
	good := true
	set := func(k string, v any, ok bool) {
		if !ok {
			good = false
			return
		}
		out[k] = v
	}
	v, ok := timingEnum(f["kind"], "nft", "tc", "conntrack", "ip", "other")
	set("kind", v, ok)
	v, ok = timingEnum(f["phase"], "shape", "install")
	set("phase", v, ok)
	v, ok = timingBracket(f["context_deadline"])
	set("context_deadline", v, ok)
	for _, k := range []string{"start_attempt", "start_return", "child_started_by", "child_start_exact", "cancel_observed", "wait_return"} {
		v, ok := timingStamp(f[k])
		set(k, v, ok)
	}
	for _, k := range []string{"context_before", "context_after", "context_at_cancel"} {
		v, ok := timingEnum(f[k], "none", "canceled", "deadline_exceeded", "other", "unavailable")
		set(k, v, ok)
	}
	for _, k := range []string{"started", "ok", "exit_available", "signal_available", "cpu_available", "switches_available"} {
		v, ok := timingBool(f[k])
		set(k, v, ok)
	}
	v, ok = timingInt(f["exit_code"], -1, 255)
	set("exit_code", v, ok)
	v, ok = timingInt(f["signal"], 0, 64)
	set("signal", v, ok)
	for _, k := range []string{"user_cpu_ns", "system_cpu_ns", "voluntary_switches", "involuntary_switches"} {
		v, ok := timingDecimal(f[k], false)
		set(k, v, ok)
	}
	return out, good
}
