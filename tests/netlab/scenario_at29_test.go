//go:build netlab

package netlab

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAT29_LeaseTimeout(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	now := time.Now().UTC()
	m := functionalModel(now, 29)
	for i := range m.Leases {
		// Earliest deadline is the service end. The offline lease is longer,
		// so the kernel timeout is the service end (§16.3).
		m.Leases[i].EffectiveServiceEnd = now.Add(8 * time.Second)
		m.Leases[i].NodeLeaseEnd = now.Add(time.Hour)
		m.Leases[i].SignedServerTime = now
		m.Leases[i].MaxOfflineSeconds = 3600
	}
	for i := range m.Bindings {
		m.Bindings[i].ValidUntil = now.Add(time.Hour)
	}
	m.Now = now
	applyStarted := time.Now()
	doc := applyAt(t, m)
	if doc.Admitted != 3 {
		t.Fatalf("admitted %d, want 3", doc.Admitted)
	}
	// The one-shot apply has exited. Nothing is refreshing the lease.
	time.Sleep(time.Second)
	early := runProbe(t, []probeCase{{"early", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}})
	requireProbe(t, early)

	var stopped time.Duration
	deadline := applyStarted.Add(14 * time.Second)
	for time.Now().Before(deadline) {
		res := runProbe(t, []probeCase{{"later", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 500, "", "", "", "", 0}})
		if len(res) == 1 && !res[0].OK {
			stopped = time.Since(applyStarted)
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if stopped == 0 {
		save(t, "debug.txt", debugPlane())
		t.Fatal("forwarding did not stop after the kernel timeout")
	}
	if stopped < 4*time.Second {
		t.Fatalf("forwarding stopped too soon (%s); the allow element was not installed", stopped)
	}
	still := runProbe(t, []probeCase{{"stays", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0}})
	requireProbe(t, still)

	clockNote := probeTimeNamespace(t)
	resumed := functionalModel(time.Now().UTC(), 30)
	applyAt(t, resumed)
	again := runProbe(t, []probeCase{{"resume", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}})
	requireProbe(t, again)
	body := fmt.Sprintf("stop_after=%s admitted=%d\n%s\n", stopped.Round(time.Millisecond), doc.Admitted, clockNote)
	save(t, "lease.txt", body)
	fmt.Print(body)
	noteSummary(fmt.Sprintf("AT29 stop_after=%s resume=1", stopped.Round(time.Millisecond)))
}

func probeTimeNamespace(t *testing.T) string {
	t.Helper()
	before := strings.TrimSpace(mustExec(t, "date", "+%s"))
	out, errOut, err := dockerExec("unshare", "--time", "--fork", "sh", "-c", "date -s @1577836800 >/dev/null && date -u +%Y-%m-%dT%H:%M:%SZ")
	after := strings.TrimSpace(mustExec(t, "date", "+%s"))
	if err != nil {
		return "clock_step=skipped reason=" + oneLine(errOut+" "+err.Error()) + " container_before=" + before + " container_after=" + after
	}
	b, _ := parseSec(before)
	a, _ := parseSec(after)
	if a+30 < b {
		// The child namespace was not isolated. Put the container clock back
		// immediately. This is the container's time namespace, which on Docker
		// is the host clock, so this restore is mandatory.
		_, _, _ = dockerExec("date", "-s", "@"+before)
		return "clock_step=UNSAFE restored=1 before=" + before + " after=" + after
	}
	return "clock_step=isolated child=" + strings.TrimSpace(out) + " container_before=" + before + " container_after=" + after + " note=nft timeout follows jiffies, not the wall clock"
}

func parseSec(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscan(s, &n)
	return n, err
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 180 {
		return s[:180]
	}
	return s
}
