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
)

// TEST ONLY: used at the already authorized AT02 service stop. Keep the pause
// marker until the original Service(start) resumes the same selected release.
// The supervisor may also signal; the interval begins before either can do so.
func observeStop() {
	const base = "/var/lib/vpn-node/e2e"
	out := map[string]any{"available": false, "control_io_after_cancel": "not observable here"}
	defer func() { _ = json.NewEncoder(os.Stdout).Encode(out) }()
	pidBytes, err := os.ReadFile(base + "/agent.pid")
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil || pid <= 1 {
		return
	}
	id, err := os.ReadFile(base + "/agent-process-id")
	if err != nil {
		return
	}
	launch := strings.TrimSpace(string(id))
	if filepath.Base(launch) != launch || launch == "." || launch == "" {
		return
	}
	exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil || filepath.Base(exe) != "fvpn-agent" {
		return
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return
	}
	fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+1:])
	if len(fields) < 20 {
		return
	}
	out["pid"], out["process_id"], out["start_ticks"] = pid, launch, fields[19]
	started := time.Now()
	out["signal_request_started_at"] = started.UTC()
	if err := os.WriteFile(base+"/pause-agent", nil, 0600); err != nil {
		return
	}
	signalAt := time.Now()
	err = syscall.Kill(pid, syscall.SIGTERM)
	out["sigterm_requested_at"], out["sigterm_accepted"] = signalAt.UTC(), err == nil
	for time.Since(started) < 15*time.Second {
		raw, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		exited := os.IsNotExist(e)
		if e == nil {
			f := strings.Fields(string(raw)[strings.LastIndex(string(raw), ")")+1:])
			exited = len(f) > 19 && (f[0] == "Z" || f[19] != fields[19])
		}
		if exited {
			elapsed := time.Since(started)
			out["exit_observed_at"], out["request_to_exit_upper_bound_ns"] = time.Now().UTC(), fmt.Sprint(elapsed.Nanoseconds())
			out["within_13s"], out["available"] = elapsed <= 13*time.Second, true
			break
		}
		time.Sleep(time.Millisecond)
	}
	// Persisted parent wait distinguishes a cooperative exit from supervisor KILL.
	for i := 0; i < 200; i++ {
		raw, e := os.ReadFile(base + "/agent-exits/" + launch + ".json")
		if e == nil && json.Valid(raw) {
			var receipt struct {
				PID        int    `json:"pid"`
				StartTicks string `json:"start_ticks"`
				ExitCode   *int   `json:"exit_code"`
				Signal     *int   `json:"signal"`
			}
			if json.Unmarshal(raw, &receipt) == nil {
				out["cooperative_exit"] = receipt.PID == pid && receipt.StartTicks == fields[19] && receipt.Signal == nil && receipt.ExitCode != nil && *receipt.ExitCode == 0
			}
			out["parent_wait_receipt"] = json.RawMessage(raw)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	out["scope"] = "pause/SIGTERM request to sampled process exit upper bound; supervisor may also signal; parent wait distinguishes actual termination; no kernel-exit instant or hard scheduling guarantee"
}
