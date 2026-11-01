//go:build installer

package installertest

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Observe the installed binary, without replacing ExecStart or its signal
// handler. strace adds measurement overhead; this is not a worst-case test.
func testAT07AgentStop(t *testing.T, c string, pair bool) {
	t.Helper()
	const agent = "fvpn-agent.service"
	const watchdog = "fvpn-watchdog.service"
	if got := strings.TrimSpace(mustExec(t, c, "systemctl", "show", agent, "-p", "TimeoutStopUSec", "--value")); got != "15s" {
		t.Fatalf("installed TimeoutStopUSec=%q", got)
	}
	pid := strings.TrimSpace(mustExec(t, c, "systemctl", "show", agent, "-p", "MainPID", "--value"))
	if id, err := strconv.Atoi(pid); err != nil || id <= 1 {
		t.Fatalf("agent is not running: %q", pid)
	}
	tracePath := "/var/tmp/fvpn-agent-stop.trace"
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	trace := exec.CommandContext(ctx, "docker", "exec", c, "strace", "-f", "-ttt", "-e", "trace=exit_group", "-e", "signal=SIGTERM", "-p", pid, "-o", tracePath)
	var diagnostics strings.Builder
	trace.Stdout, trace.Stderr = &diagnostics, &diagnostics
	if err := trace.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		cancel()
		if !waited {
			_ = trace.Wait()
		}
	}()
	attached := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		status := mustExec(t, c, "cat", "/proc/"+pid+"/status")
		for _, line := range strings.Split(status, "\n") {
			if strings.HasPrefix(line, "TracerPid:") && strings.TrimSpace(strings.TrimPrefix(line, "TracerPid:")) != "0" {
				attached = true
			}
		}
		if attached {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !attached {
		t.Fatal("strace did not attach to the installed agent")
	}
	units := []string{agent}
	if pair {
		units = append(units, watchdog)
	}
	// Clock readings surround systemctl inside the container, excluding docker
	// exec startup. No simulated sleep or signal handler supplies the timing.
	args := append([]string{"sh", "-c", `start=$(date +%s%N)
systemctl stop "$@"
code=$?
end=$(date +%s%N)
printf '%s %s\n' "$start" "$end"
exit "$code"`, "stop-timing"}, units...)
	raw := strings.Fields(mustExec(t, c, args...))
	if len(raw) != 2 {
		t.Fatalf("invalid stop timing: %v", raw)
	}
	started, err := strconv.ParseInt(raw[0], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	ended, err := strconv.ParseInt(raw[1], 10, 64)
	if err != nil || ended < started {
		t.Fatalf("stop clock: %v %v", raw, err)
	}
	err = trace.Wait()
	waited = true
	if err != nil {
		t.Fatalf("strace: %v %s", err, diagnostics.String())
	}
	observed := mustExec(t, c, "cat", tracePath)
	signal := regexp.MustCompile(`(?m)^\d+\s+([0-9]+\.[0-9]+)\s+--- SIGTERM`).FindStringSubmatch(observed)
	exit := regexp.MustCompile(`(?m)^\d+\s+([0-9]+\.[0-9]+)\s+exit_group\(0\)`).FindStringSubmatch(observed)
	if len(signal) != 2 || len(exit) != 2 {
		t.Fatalf("missing real SIGTERM or successful exit_group observation: %s", observed)
	}
	signalAt, err := time.ParseDuration(signal[1] + "s")
	if err != nil {
		t.Fatal(err)
	}
	exitAt, err := time.ParseDuration(exit[1] + "s")
	if err != nil || exitAt < signalAt {
		t.Fatalf("trace clock: %v %v %v", signal, exit, err)
	}
	for _, unit := range units {
		got := strings.TrimSpace(mustExec(t, c, "systemctl", "show", unit, "-p", "Result", "--value"))
		if got != "success" {
			t.Fatalf("%s stop result %s", unit, got)
		}
	}
	if pair {
		stamp := func(unit, property string) uint64 {
			t.Helper()
			value := strings.TrimSpace(mustExec(t, c, "systemctl", "show", unit, "-p", property, "--value"))
			n, err := strconv.ParseUint(value, 10, 64)
			if err != nil || n == 0 {
				t.Fatalf("%s %s: %q %v", unit, property, value, err)
			}
			return n
		}
		agentExit := stamp(agent, "ExecMainExitTimestampMonotonic")
		watchdogStop := stamp(watchdog, "ActiveExitTimestampMonotonic")
		if watchdogStop < agentExit {
			t.Fatalf("watchdog stopped before agent exit: %d < %d", watchdogStop, agentExit)
		}
		t.Logf("joint stop ordering: watchdog stop began %s after agent exit", time.Duration(watchdogStop-agentExit)*time.Microsecond)
	} else {
		mustExec(t, c, "systemctl", "is-active", watchdog)
	}
	t.Logf("AT07 real installed agent stop (pair=%t): SIGTERM -> exit_group(0)=%s; systemctl stop return=%s; strace attached, wall-clock observations, enrollment-only fixture", pair, exitAt-signalAt, time.Duration(ended-started))
	if pair {
		mustExec(t, c, "systemctl", "start", watchdog, agent)
	}
	mustExec(t, c, "rm", tracePath)
}
