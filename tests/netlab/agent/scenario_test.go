//go:build netlab

package agentlab

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestLab01_ForwardOnlyAfterLease(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	admin(t, "/hold", "")
	defer admin(t, "/release", "")

	startAgent(t)
	defer stopAgent()
	// Boot has already installed the baseline. The lease call is held, so
	// this step cannot enable forwarding (§16.3).
	if !waitProbe(t, "deny", 12*time.Second) {
		t.Fatalf("forwarded before a lease:\n%s", debugPlane(t))
	}
	admin(t, "/release", "")
	if !waitProbe(t, "allow", 20*time.Second) {
		t.Fatalf("did not forward after a valid lease:\n%s", debugPlane(t))
	}
	if !scratchExists(labDir + "/identity/setup.key") {
		t.Fatal("setup key was not stored")
	}
	mode := strings.Fields(mustExec(t, "stat", "-c", "%a", labDir+"/identity/setup.key"))
	if len(mode) != 1 || mode[0] != "600" {
		t.Fatalf("setup key mode %v, want 600", mode)
	}
	assertNoSecrets(t)
	fmt.Println("SUMMARY AT-agent forward-after-lease=1")
}

func TestLab02_WatchdogEnforces(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	startWatch(t)
	defer stopWatch()
	startAgent(t)
	if !waitProbe(t, "allow", 20*time.Second) {
		t.Fatalf("agent did not start forwarding:\n%s", debugPlane(t))
	}
	// SIGKILL, not SIGTERM. A graceful stop is itself a hand-off and
	// installs the baseline; this test needs the process to disappear
	// so the watchdog is what fail-closes.
	crashProc("fvpn-agent", labDir+"/agent.pid")
	// Stale heartbeat is 3s. The watchdog, not a new control call, installs
	// the baseline (§16.3).
	if !waitProbe(t, "deny", 12*time.Second) {
		t.Fatalf("watchdog left forwarding on:\n%s", debugPlane(t))
	}
	fmt.Println("SUMMARY AT-agent watchdog-enforces=1")
}

func TestLab03_KernelTimeout(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":8}`)
	started := time.Now()
	startAgent(t)
	if !waitProbe(t, "allow", 15*time.Second) {
		stopAgent()
		t.Fatalf("short lease did not forward:\n%s", debugPlane(t))
	}
	// Crash both processes. SIGTERM is a graceful hand-off: Close installs
	// the fail-closed baseline before the kernel element times out.
	// SIGKILL leaves the allow element, so the kernel timeout is the deadline.
	crashProc("fvpn-watchdog", labDir+"/watchdog.pid")
	crashProc("fvpn-agent", labDir+"/agent.pid")
	var stopped time.Duration
	deadline := started.Add(14 * time.Second)
	for time.Now().Before(deadline) {
		if ok, _ := probeOK(t, "deny"); ok {
			stopped = time.Since(started)
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if stopped == 0 {
		t.Fatalf("forwarding did not stop after the kernel timeout:\n%s", debugPlane(t))
	}
	if stopped < 4*time.Second {
		t.Fatalf("forwarding stopped too soon (%s); the allow element was not installed", stopped)
	}
	fmt.Printf("SUMMARY AT29 stop_after=%s\n", stopped.Round(time.Millisecond))
}

// crashProc sends SIGKILL and does not SIGTERM first, so the process cannot
// run its hand-off. stopProc stays the graceful stop used by the other labs.
func crashProc(name, pidFile string) {
	_, _, _ = dockerExec("sh", "-c", fmt.Sprintf(
		"if [ -f %s ]; then kill -9 $(cat %s) 2>/dev/null || true; rm -f %s; fi; pkill -9 -x %s 2>/dev/null || true",
		pidFile, pidFile, pidFile, name,
	))
}

func TestLab04_RestartWaitsForLease(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	startAgent(t)
	if !waitProbe(t, "allow", 20*time.Second) {
		stopAgent()
		t.Fatalf("precondition: not forwarding:\n%s", debugPlane(t))
	}
	stopAgent()
	admin(t, "/hold", "")
	defer admin(t, "/release", "")
	startAgent(t)
	defer stopAgent()
	// The new process installs the baseline before it accepts a lease.
	if !waitProbe(t, "deny", 12*time.Second) {
		t.Fatalf("restart forwarded before a lease:\n%s", debugPlane(t))
	}
	admin(t, "/release", "")
	if !waitProbe(t, "allow", 20*time.Second) {
		t.Fatalf("restart did not forward after the new lease:\n%s", debugPlane(t))
	}
	fmt.Println("SUMMARY AT-agent restart-waits=1")
}

func TestLab05_NoAllowAllOnUpdate(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	startAgent(t)
	defer stopAgent()
	if !waitProbe(t, "allow", 20*time.Second) {
		t.Fatalf("precondition: not forwarding:\n%s", debugPlane(t))
	}
	_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/denyloop.stop /scratch/deny-hits.log "+labDir+"/deny-cases.json")
	deny := []probeCase{
		{"lan10", "fv-a1", "tcp", "10.1.2.10", 80, "", "deny", 400, "", "", "", "", 0},
		{"lan192", "fv-a1", "tcp", "192.168.10.10", 80, "", "deny", 400, "", "", "", "", 0},
		{"meta", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 400, "", "", "", "", 0},
		{"pubpriv", "fv-a1", "tcp", "192.0.2.200", 80, "", "deny", 400, "", "", "", "", 0},
		{"rogue", "fv-rogue", "tcp", "203.0.113.10", 80, "", "deny", 400, "", "", "", "", 0},
	}
	body, err := json.Marshal(map[string]any{"cases": deny})
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/deny-cases.json", body)
	if err := dockerDetach("python3", "/opt/netlab/scripts/denyloop.py", "/scratch/deny-cases.json"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop; if [ -f /scratch/denyloop.pid ]; then kill $(cat /scratch/denyloop.pid) 2>/dev/null || true; fi")
	}()
	for i := 0; i < 3; i++ {
		admin(t, "/bump", "")
		time.Sleep(1500 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop")
	time.Sleep(500 * time.Millisecond)
	if scratchExists("/scratch/deny-hits.log") {
		hits := readScratch(t, "/scratch/deny-hits.log")
		if strings.TrimSpace(hits) != "" {
			t.Fatalf("allow-all window during desired-state update:\n%s", hits)
		}
	}
	if ok, obs := probeOK(t, "allow"); !ok {
		t.Fatalf("forwarding dropped across the update (%s):\n%s", obs, debugPlane(t))
	}
	lan := runProbe(t, []probeCase{{"lan", "fv-a1", "tcp", "10.1.2.10", 80, "", "deny", 800, "", "", "", "", 0}})
	if len(lan) != 1 || !lan[0].OK {
		t.Fatalf("private destination was allowed: %+v", lan)
	}
	fmt.Println("SUMMARY AT13 AT29 no-allow-all=1")
}

func TestLab06_RevokedStopsForwarding(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	startAgent(t)
	defer stopAgent()
	if !waitProbe(t, "allow", 20*time.Second) {
		t.Fatalf("precondition: not forwarding:\n%s", debugPlane(t))
	}
	admin(t, "/revoke", "")
	if !waitProbe(t, "deny", 15*time.Second) {
		t.Fatalf("revoked identity kept forwarding:\n%s", debugPlane(t))
	}
	assertNoSecrets(t)
	fmt.Println("SUMMARY AT-agent revoked-stops=1")
}
