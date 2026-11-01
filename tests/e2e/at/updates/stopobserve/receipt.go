package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait"
	"github.com/google/uuid"
)

type identity struct {
	ID             string `json:"id"`
	PID            int    `json:"pid"`
	StartTicks     string `json:"start_ticks"`
	AlreadyStopped bool   `json:"already_stopped"`
}

func startTicks(raw []byte) (string, error) {
	end := strings.LastIndexByte(string(raw), ')')
	if end < 0 {
		return "", fmt.Errorf("missing process stat")
	}
	fields := strings.Fields(string(raw[end+1:]))
	if len(fields) < 20 {
		return "", fmt.Errorf("short process stat")
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", fmt.Errorf("invalid start ticks")
	}
	return fields[19], nil
}

// Retain ID first, then PID and kernel start ticks, before the original stop.
// An idempotent stop may reuse only the same launch's previously captured identity.
func capture(base string) (identity, error) {
	var out identity
	raw, err := os.ReadFile(filepath.Join(base, "agent-process-id"))
	if err != nil {
		return out, fmt.Errorf("launch ID unavailable")
	}
	out.ID = strings.TrimSpace(string(raw))
	if _, err := uuid.Parse(out.ID); err != nil {
		return out, fmt.Errorf("invalid launch ID")
	}
	raw, err = os.ReadFile(filepath.Join(base, "agent.pid"))
	if err != nil {
		previous, e := retained(base)
		if e != nil || previous.ID != out.ID {
			return out, fmt.Errorf("pre-stop PID unavailable")
		}
		previous.AlreadyStopped = true
		if _, e := receipt(base, previous); e != nil {
			return out, e
		}
		return previous, nil
	}
	out.PID, err = strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || out.PID <= 1 {
		return out, fmt.Errorf("invalid pre-stop PID")
	}
	raw, err = os.ReadFile(fmt.Sprintf("/proc/%d/stat", out.PID))
	if err != nil {
		return out, fmt.Errorf("pre-stop process unavailable")
	}
	out.StartTicks, err = startTicks(raw)
	return out, err
}

func retain(base string, id identity) error {
	raw, err := json.Marshal(id)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(base, "stop-observer-identity.json"), raw, 0600)
}
func retained(base string) (identity, error) {
	var id identity
	raw, err := os.ReadFile(filepath.Join(base, "stop-observer-identity.json"))
	if err == nil {
		err = json.Unmarshal(raw, &id)
	}
	return id, err
}

func validateReceipt(raw []byte, id identity) error {
	var r agentwait.Result
	// Required fields cannot silently acquire zero values (especially exit 0).
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return fmt.Errorf("invalid wait receipt")
	}
	for _, key := range []string{"id", "pid", "start_ticks", "wait_status", "exit_code", "signal", "wait_observed_at", "core_dump"} {
		if _, ok := fields[key]; !ok {
			return fmt.Errorf("incomplete wait receipt")
		}
	}
	if json.Unmarshal(raw, &r) != nil || r.ID != id.ID || r.PID != id.PID || r.StartTicks != id.StartTicks || id.StartTicks == "" {
		return fmt.Errorf("wait receipt identity mismatch")
	}
	if r.WaitObservedAt.IsZero() || !strings.HasSuffix(strings.Trim(string(fields["wait_observed_at"]), "\""), "Z") {
		return fmt.Errorf("wait observation time unavailable or not UTC")
	}
	status := syscall.WaitStatus(r.WaitStatus)
	if status.Exited() {
		if r.ExitCode == nil || r.Signal != nil || *r.ExitCode != status.ExitStatus() {
			return fmt.Errorf("exit receipt mismatch")
		}
	} else if status.Signaled() {
		if r.Signal == nil || r.ExitCode != nil || *r.Signal != int(status.Signal()) {
			return fmt.Errorf("signal receipt mismatch")
		}
	} else {
		return fmt.Errorf("nonterminal wait receipt")
	}
	if status.CoreDump() != r.CoreDump {
		return fmt.Errorf("core dump receipt mismatch")
	}
	return nil
}

func receipt(base string, id identity) (json.RawMessage, error) {
	if _, err := uuid.Parse(id.ID); err != nil {
		return nil, fmt.Errorf("invalid retained ID")
	}
	raw, err := os.ReadFile(filepath.Join(base, "agent-exits", id.ID+".json"))
	if err != nil {
		return nil, fmt.Errorf("wait receipt unavailable")
	}
	if err := validateReceipt(raw, id); err != nil {
		return nil, err
	}
	return raw, nil
}

func observeReceipt(base string, id identity, row map[string]any) {
	raw, err := receipt(base, id)
	if err != nil {
		row["receipt_error"] = err.Error()
	} else {
		row["parent_wait_receipt"] = raw
	}
	// This separate record is an attempted SIGKILL, never the termination result.
	raw, err = os.ReadFile(filepath.Join(base, "agent-exits", id.ID+".sigkill.json"))
	if err == nil && json.Valid(raw) {
		row["sigkill_attempt"] = json.RawMessage(raw)
	}
}

// Systemd is the installer agent's parent. Export its waitid code/status without
// relabeling the encoded equivalent as a raw wait(2) status. The independent
// /proc sample, when available, remains explicitly separate.
func systemdReceipt(before identity, properties string) (map[string]any, error) {
	p := map[string]string{}
	for _, line := range strings.Split(properties, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			p[k] = v
		}
	}
	pid, _ := strconv.Atoi(p["ExecMainPID"])
	if pid != before.PID || before.ID == "" || before.StartTicks == "" {
		return nil, fmt.Errorf("systemd parent identity unavailable")
	}
	code, e1 := strconv.Atoi(p["ExecMainCode"])
	status, e2 := strconv.Atoi(p["ExecMainStatus"])
	at, e3 := time.Parse("Mon 2006-01-02 15:04:05 MST", p["ExecMainExitTimestamp"])
	if e1 != nil || e2 != nil || e3 != nil || at.IsZero() {
		return nil, fmt.Errorf("systemd parent wait unavailable")
	}
	r := map[string]any{"id": before.ID, "pid": pid, "start_ticks": before.StartTicks, "waitid_code": code, "waitid_status": status, "wait_observed_at": at.UTC().Format(time.RFC3339Nano), "timestamp_precision": "1 second (systemctl show)", "exit_code": nil, "signal": nil, "source": "systemd parent waitid"}
	switch code {
	case 1:
		r["exit_code"] = status
		r["encoded_wait_status"] = status << 8
	case 2, 3:
		r["signal"] = status
		raw := status
		if code == 3 {
			raw |= 128
		}
		r["encoded_wait_status"] = raw
	default:
		return nil, fmt.Errorf("systemd wait not terminal")
	}
	return r, nil
}
