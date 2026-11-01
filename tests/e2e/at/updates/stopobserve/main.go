// Command stopobserve is a TEST ONLY observer installed in owned lab containers.
// It delegates the original service command without changing arguments or limits.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const base = "/var/lib/vpn-node/e2e"

const evidence = "/var/lib/vpn-node/e2e/stop-observation.jsonl"

func lease() map[string]any {
	out := map[string]any{"present": false}
	db, err := sql.Open("sqlite", "file:/var/lib/vpn-node/state/agent.db?mode=ro&_pragma=busy_timeout(100)")
	if err != nil {
		out["read_error"] = true
		return out
	}
	defer db.Close()
	var id, expiry string
	err = db.QueryRow("SELECT lease_id, expires_at FROM lease WHERE id=1").Scan(&id, &expiry)
	if err == nil {
		out["present"], out["lease_id"], out["expires_at"] = true, id, expiry
	} else if err != sql.ErrNoRows {
		out["read_error"] = true
	}
	return out
}

func transaction() map[string]any {
	out := map[string]any{}
	db, err := sql.Open("sqlite", "file:/var/lib/vpn-node/update-journal/journal.db?mode=ro&_pragma=busy_timeout(100)")
	if err != nil {
		return out
	}
	defer db.Close()
	var id, stage string
	if db.QueryRow("SELECT txn_id, stage FROM txns ORDER BY updated_at DESC LIMIT 1").Scan(&id, &stage) == nil {
		out["txn_id"], out["stage"] = id, stage
	}
	return out
}

func procExit(pid string) (string, int) {
	b, err := os.ReadFile("/proc/" + pid + "/stat")
	if err != nil {
		return "unavailable", -1
	}
	end := strings.LastIndex(string(b), ")")
	if end < 0 {
		return "unavailable", -1
	}
	f := strings.Fields(string(b)[end+1:])
	if len(f) < 50 {
		return "unavailable", -1
	}
	code, err := strconv.Atoi(f[49])
	if err != nil {
		return "unavailable", -1
	}
	return f[0], code
}

func systemd() string {
	b, _ := exec.Command("/usr/bin/systemctl", "show", "fvpn-agent.service", "-p", "ExecMainCode", "-p", "ExecMainStatus", "-p", "Result", "-p", "TimeoutStopUSec", "-p", "MainPID", "-p", "InvocationID", "-p", "ExecMainPID", "-p", "ExecMainExitTimestamp").Output()
	return strings.TrimSpace(string(b))
}

func main() {
	if len(os.Args) < 3 {
		os.Exit(64)
	}
	target, args := os.Args[1], os.Args[2:]
	parent, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", os.Getppid()))
	observe := strings.HasSuffix(parent, "/fvpn-updater") && (args[0] == "stop" || args[0] == "start")
	if !observe {
		c := exec.Command(target, args...)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				os.Exit(e.ExitCode())
			}
			os.Exit(127)
		}
		return
	}
	row := map[string]any{"command": append([]string{target}, args...), "parent": filepath.Base(parent), "action": args[0], "transaction": transaction(), "lease_before": lease()}
	isSystemd := target == "/usr/bin/systemctl"
	var pid string
	var id identity
	var identityErr error
	if isSystemd {
		b, _ := exec.Command(target, "show", "-p", "MainPID", "--value", "fvpn-agent.service").Output()
		pid = strings.TrimSpace(string(b))
		before := systemd()
		row["systemd_before"] = before
		if args[0] == "stop" && pid != "" && pid != "0" {
			id.PID, _ = strconv.Atoi(pid)
			raw, e := os.ReadFile("/proc/" + pid + "/stat")
			if e == nil {
				id.StartTicks, e = startTicks(raw)
			}
			identityErr = e
			for _, line := range strings.Split(before, "\n") {
				if v, ok := strings.CutPrefix(line, "InvocationID="); ok {
					id.ID = v
				}
			}
		} else {
			id, identityErr = retained(base)
			id.AlreadyStopped = args[0] == "stop"
		}
	} else {
		b, _ := os.ReadFile("/var/lib/vpn-node/e2e/agent.pid")
		pid = strings.TrimSpace(string(b))
		if args[0] == "stop" {
			id, identityErr = capture(base)
		} else {
			id, identityErr = retained(base)
		}
	}
	row["agent_identity_before"] = id
	if identityErr != nil {
		row["identity_error"] = identityErr.Error()
	}
	if args[0] == "stop" && identityErr == nil {
		if err := retain(base, id); err != nil {
			row["identity_error"] = "identity retention failed"
		}
	}
	row["agent_pid_before"] = pid
	done := make(chan struct{})
	observed := make(chan int, 1)
	go func() {
		code := -1
		if isSystemd && pid != "" && pid != "0" {
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				state, status := procExit(pid)
				if state == "Z" {
					code = status
					break
				}
				if state == "unavailable" {
					break
				}
				select {
				case <-done:
					observed <- code
					return
				case <-ticker.C:
				}
			}
		}
		observed <- code
	}()
	started := time.Now()
	row["started_at"] = started.UTC().Format(time.RFC3339Nano)
	c := exec.Command(target, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := c.Run()
	elapsed := time.Since(started)
	close(done)
	row["duration_ns"] = elapsed.Nanoseconds()
	row["finished_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	code := 0
	if err != nil {
		code = 127
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		}
	}
	row["exit_code"] = code
	row["sampled_raw_wait_status"] = <-observed
	row["lease_after"] = lease()
	if isSystemd {
		after := systemd()
		row["systemd_after"] = after
		if args[0] == "stop" {
			r, e := systemdReceipt(id, after)
			if e != nil {
				row["receipt_error"] = e.Error()
			} else {
				row["systemd_parent_wait_receipt"] = r
				raw, _ := json.Marshal(r)
				if e := os.WriteFile(filepath.Join(base, "stop-observer-systemd-receipt.json"), raw, 0600); e != nil {
					row["receipt_error"] = "systemd receipt retention failed"
				}
			}
		} else {
			raw, e := os.ReadFile(filepath.Join(base, "stop-observer-systemd-receipt.json"))
			if e == nil && json.Valid(raw) {
				row["retained_systemd_parent_wait_receipt"] = json.RawMessage(raw)
			}
		}
	} else if identityErr == nil {
		observeReceipt(base, id, row)
	}
	f, e := os.OpenFile(evidence, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		fmt.Fprintln(os.Stderr, "TEST ONLY stop observation unavailable")
		os.Exit(125)
	}
	e = json.NewEncoder(f).Encode(row)
	_ = f.Close()
	if e != nil {
		os.Exit(125)
	}
	if code < 0 {
		code = 128
	}
	os.Exit(code)
}
