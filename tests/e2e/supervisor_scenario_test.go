//go:build e2e

package e2e

import (
	"context"
	"os/exec"
	"strings"
)

// agentSupervisor proves stop/start twenty times without a second fvpn-agent.
// The public commands only toggle pause-agent. The entrypoint loop owns the
// process and agent.pid.
func (r *runner) agentSupervisor(ctx context.Context) {
	rec := Record{ID: "agent-supervisor", Title: "Agent stop and start go through the supervisor", Spec: []string{"§5", "D-053"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() {
		// Leave the acceptance agent running under the supervisor.
		if _, err := r.agentService(ctx, "start", "/usr/local"); err != nil {
			note("restore", "start failed")
		}
		r.record(rec)
	}()
	cycles := 20
	var last string
	for i := 1; i <= cycles; i++ {
		stopped, err := r.agentService(ctx, "stop")
		pids, pidErr := r.agentRunPIDs()
		if err != nil || stopped != "0" || pidErr != nil || len(pids) != 0 {
			note("cycle", itoa(i))
			note("stop_stdout", emptyDash(stopped))
			note("stop_error", boolText(err != nil))
			note("processes_after_stop", itoa(len(pids)))
			return
		}
		started, err := r.agentService(ctx, "start", "/usr/local")
		pids, pidErr = r.agentRunPIDs()
		pidFile := strings.TrimSpace(string(mustDocker(r, "cat", "/var/lib/vpn-node/e2e/agent.pid")))
		if err != nil || pidErr != nil || len(pids) != 1 || started == "" || started == "0" || started != pids[0] || pidFile != started {
			note("cycle", itoa(i))
			note("start_stdout", emptyDash(started))
			note("start_error", boolText(err != nil))
			note("processes_after_start", itoa(len(pids)))
			note("pid_file_matches", boolText(pidFile == started && len(pids) == 1 && pids[0] == started))
			return
		}
		last = started
	}
	again, err := r.agentService(ctx, "start", "/usr/local")
	pids, pidErr := r.agentRunPIDs()
	if err != nil || pidErr != nil || again != last || len(pids) != 1 || pids[0] != last {
		note("idempotent_start", "pid changed")
		note("start_stdout", emptyDash(again))
		note("processes", itoa(len(pids)))
		return
	}
	_, pauseErr := r.lab.docker(r.lab.node, "test", "-e", "/var/lib/vpn-node/e2e/pause-agent")
	note("cycles", itoa(cycles))
	note("final_pid_present", "true")
	note("pause_cleared", boolText(pauseErr != nil))
	note("repeated_start_same_pid", "true")
	if pauseErr != nil {
		rec.Status = StatusPass
	}
}

func (r *runner) agentService(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", r.lab.node, "agent-service.sh"}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// agentRunPIDs lists live fvpn-agent run processes. The walker is fed on stdin
// so the docker exec argv does not contain the process name and count itself.
func (r *runner) agentRunPIDs() ([]string, error) {
	script := `for d in /proc/[0-9]*; do
  [ -d "$d" ] || continue
  [ -r "$d/cmdline" ] || continue
  [ -r "$d/stat" ] || continue
  rest=$(cat "$d/stat") || continue
  rest=${rest#*)}
  rest=${rest# }
  state=${rest%% *}
  [ "$state" = Z ] && continue
  cmd=$(tr '\0' ' ' < "$d/cmdline") || continue
  case "$cmd" in
    *"/fvpn-agent run"|*"/fvpn-agent run "*) printf '%s\n' "${d#/proc/}" ;;
  esac
done
`
	cmd := exec.Command("docker", "exec", "-i", r.lab.node, "sh", "-s")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func mustDocker(r *runner, args ...string) []byte {
	out, err := r.lab.docker(r.lab.node, args...)
	if err != nil {
		return nil
	}
	return out
}
