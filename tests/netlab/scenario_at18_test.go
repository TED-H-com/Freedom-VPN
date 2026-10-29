//go:build netlab

package netlab

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func TestAT18_FUP02_QoS(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	restartIperf(t)
	warm, measure, reps := windows()
	seconds := int((warm + measure + 3*time.Second) / time.Second)
	var report strings.Builder
	fmt.Fprintf(&report, "TEST ONLY R=%.0fMbit C=%.0fMbit warm=%s measure=%s reps=%d\n", float64(testRateBPS)/1e6, float64(testCapBPS)/1e6, warm, measure, reps)

	shared := deviceModel(time.Now().UTC(), 18, []idBind{
		{memberA, devA1, "peer-a1", "100.64.1.1"},
		{memberA, devA2, "peer-a2", "100.64.1.2"},
	}, sharedQoS(), 48*time.Hour)
	doc := applyAt(t, shared)
	minorA := minorOf(t, doc, memberA)

	for rep := 1; rep <= reps; rep++ {
		up := measureClass(t, "fv-ih", minorA, testRateBPS, warm, measure, func() {
			launchIperf(t, "fv-a1", 5201, seconds, "/scratch/iperf-a1.pid")
			launchIperf(t, "fv-a2", 5202, seconds, "/scratch/iperf-a2.pid")
		})
		fmt.Fprintf(&report, "shared_upload rep=%d mbps=%.2f ratio=%.3f drops=%d cpu=%.1f host_mbps=%.2f\n", rep, up.mbps, up.ratio, up.drops, up.cpu, up.hostMbps)
		assertRatio(t, "shared upload", up.ratio, 0.85, 1.10)
		requireIperfBytes(t, "/scratch/iperf-a1.pid.log")
		requireIperfBytes(t, "/scratch/iperf-a2.pid.log")
	}
	for rep := 1; rep <= reps; rep++ {
		down := measureClass(t, "fv-fh", minorA, testRateBPS, warm, measure, func() {
			launchIperf(t, "fv-a1", 5203, seconds, "/scratch/iperf-a1.pid", "-R")
			launchIperf(t, "fv-a2", 5204, seconds, "/scratch/iperf-a2.pid", "-R")
		})
		fmt.Fprintf(&report, "shared_download rep=%d mbps=%.2f ratio=%.3f drops=%d cpu=%.1f\n", rep, down.mbps, down.ratio, down.drops, down.cpu)
		assertRatio(t, "shared download", down.ratio, 0.85, 1.10)
	}
	for rep := 1; rep <= reps; rep++ {
		up, down := measureIfacePair(t, "fv-ih", "fv-fh", minorA, testRateBPS, warm, measure, func() {
			launchIperf(t, "fv-a1", 5201, seconds, "/scratch/iperf-a1.pid")
			launchIperf(t, "fv-a2", 5202, seconds, "/scratch/iperf-a2.pid")
			launchIperf(t, "fv-a1", 5203, seconds, "/scratch/iperf-a1b.pid", "-R")
			launchIperf(t, "fv-a2", 5204, seconds, "/scratch/iperf-a2b.pid", "-R")
		})
		fmt.Fprintf(&report, "shared_both rep=%d up_mbps=%.2f up_ratio=%.3f down_mbps=%.2f down_ratio=%.3f\n", rep, up.mbps, up.ratio, down.mbps, down.ratio)
		assertRatio(t, "both upload", up.ratio, 0.85, 1.10)
		assertRatio(t, "both download", down.ratio, 0.85, 1.10)
	}

	cont := deviceModel(time.Now().UTC(), 19, allBinds(), contendQoS(), 48*time.Hour)
	cdoc := applyAt(t, cont)
	minorA = minorOf(t, cdoc, memberA)
	minorB := minorOf(t, cdoc, memberB)
	for rep := 1; rep <= reps; rep++ {
		a, b := measurePair(t, "fv-ih", minorA, minorB, warm, measure, func() {
			launchIperf(t, "fv-a1", 5201, seconds, "/scratch/iperf-a1.pid", "-P", "8")
			launchIperf(t, "fv-b1", 5202, seconds, "/scratch/iperf-b1.pid", "-P", "1")
		})
		sum := a.ratio + b.ratio
		fmt.Fprintf(&report, "contend rep=%d a=%.3f b=%.3f sum=%.3f drops=%d cpu=%.1f\n", rep, a.ratio, b.ratio, sum, a.drops+b.drops, a.cpu)
		assertRatio(t, "member A share of C", a.ratio, 0.35, 0.65)
		assertRatio(t, "member B share of C", b.ratio, 0.35, 0.65)
		assertRatio(t, "total of C", sum, 0.85, 1.10)
	}

	beforeBytes := classStats(t, "fv-ih")[classHandle(minorA)].Bytes
	changed := cont
	changed.Generation = 20
	changed.Now = time.Now().UTC()
	changed.QoS.Assignments = nil
	for i := range changed.QoS.Members {
		if changed.QoS.Members[i].MemberID == memberA {
			changed.QoS.Members[i].Up = enforce.Rate{RateBPS: 10_000_000, CeilBPS: 10_000_000}
			changed.QoS.Members[i].Down = enforce.Rate{RateBPS: 10_000_000, CeilBPS: 10_000_000}
		}
		if changed.QoS.Members[i].MemberID == memberB {
			changed.QoS.Members[i].Up = enforce.Rate{RateBPS: 8_000_000, CeilBPS: 8_000_000}
			changed.QoS.Members[i].Down = enforce.Rate{RateBPS: 8_000_000, CeilBPS: 8_000_000}
		}
	}
	// 1 + 10 + 8 = 19 <= parent 20.
	changedDoc := applyAt(t, changed)
	if minorOf(t, changedDoc, memberA) != minorA || minorOf(t, changedDoc, memberB) != minorB {
		t.Fatal("rate change moved a class minor")
	}
	afterBytes := classStats(t, "fv-ih")[classHandle(minorA)].Bytes
	if afterBytes < beforeBytes {
		t.Fatalf("class counters reset across rate change (%d -> %d)", beforeBytes, afterBytes)
	}
	text := strings.ToLower(mustExec(t, "tc", "class", "show", "dev", "fv-ih"))
	if !strings.Contains(text, "10mbit") && !strings.Contains(text, "10000kbit") {
		t.Fatalf("new rate is not installed:\n%s", text)
	}
	fmt.Fprintf(&report, "rate_change minor_a=%s minor_b=%s bytes_kept=%d\n", classHandle(minorA), classHandle(minorB), afterBytes)
	save(t, "qos.txt", report.String())
	save(t, "tc-ih.txt", mustExec(t, "tc", "-s", "class", "show", "dev", "fv-ih"))
	save(t, "tc-fh.txt", mustExec(t, "tc", "-s", "class", "show", "dev", "fv-fh"))
	fmt.Print(report.String())
	noteSummary("AT18 qos windows passed R=" + strconv.FormatUint(testRateBPS/1_000_000, 10) + "Mbit")
}

func sharedQoS() enforce.QoS {
	parent := enforce.Rate{RateBPS: 80_000_000, CeilBPS: 80_000_000}
	host := enforce.Rate{RateBPS: 1_000_000, CeilBPS: 80_000_000}
	member := enforce.Rate{RateBPS: testRateBPS, CeilBPS: testRateBPS}
	return enforce.QoS{Up: parent, Down: parent, HostUp: host, HostDown: host, Members: []enforce.MemberQoS{{MemberID: memberA, Up: member, Down: member}}}
}

func contendQoS() enforce.QoS {
	parent := enforce.Rate{RateBPS: testCapBPS, CeilBPS: testCapBPS}
	host := enforce.Rate{RateBPS: 1_000_000, CeilBPS: testCapBPS}
	share := enforce.Rate{RateBPS: 9_000_000, CeilBPS: testCapBPS}
	return enforce.QoS{
		Up: parent, Down: parent, HostUp: host, HostDown: host,
		Members: []enforce.MemberQoS{
			{MemberID: memberA, Up: share, Down: share},
			{MemberID: memberB, Up: share, Down: share},
		},
	}
}

type bandResult struct {
	mbps     float64
	ratio    float64
	drops    uint64
	cpu      float64
	elapsed  float64
	hostMbps float64
}

func measureClass(t *testing.T, dev string, minor uint16, target uint64, warm, measure time.Duration, start func()) bandResult {
	t.Helper()
	start()
	time.Sleep(warm)
	t0 := time.Now()
	idle1, total1 := cpuJiffies()
	before := classStats(t, dev)
	time.Sleep(measure)
	elapsed := time.Since(t0).Seconds()
	idle2, total2 := cpuJiffies()
	after := classStats(t, dev)
	time.Sleep(4 * time.Second)
	h := classHandle(minor)
	b0, b1 := before[h], after[h]
	host0, host1 := before["1:fffe"], after["1:fffe"]
	mbps := float64(b1.Bytes-b0.Bytes) * 8 / elapsed / 1e6
	cpu := 0.0
	if total2 > total1 {
		cpu = 100 * (1 - float64(idle2-idle1)/float64(total2-total1))
	}
	if b1.Bytes < b0.Bytes {
		t.Fatalf("class %s bytes went backwards", h)
	}
	if (b1.Bytes - b0.Bytes) < (host1.Bytes - host0.Bytes) {
		t.Fatalf("member class %s carried less than the host class; classification missed", h)
	}
	return bandResult{
		mbps:     mbps,
		ratio:    mbps / (float64(target) / 1e6),
		drops:    b1.Dropped - b0.Dropped,
		cpu:      cpu,
		elapsed:  elapsed,
		hostMbps: float64(host1.Bytes-host0.Bytes) * 8 / elapsed / 1e6,
	}
}

func measureIfacePair(t *testing.T, upDev, downDev string, minor uint16, target uint64, warm, measure time.Duration, start func()) (up, down bandResult) {
	t.Helper()
	start()
	time.Sleep(warm)
	t0 := time.Now()
	beforeUp := classStats(t, upDev)
	beforeDown := classStats(t, downDev)
	time.Sleep(measure)
	elapsed := time.Since(t0).Seconds()
	afterUp := classStats(t, upDev)
	afterDown := classStats(t, downDev)
	time.Sleep(4 * time.Second)
	return bandAgainst(beforeUp, afterUp, minor, target, elapsed), bandAgainst(beforeDown, afterDown, minor, target, elapsed)
}

func bandAgainst(before, after map[string]classStat, minor uint16, target uint64, elapsed float64) bandResult {
	h := classHandle(minor)
	b0, b1 := before[h], after[h]
	mbps := float64(b1.Bytes-b0.Bytes) * 8 / elapsed / 1e6
	drops := uint64(0)
	if b1.Dropped >= b0.Dropped {
		drops = b1.Dropped - b0.Dropped
	}
	return bandResult{mbps: mbps, ratio: mbps / (float64(target) / 1e6), drops: drops, elapsed: elapsed}
}

func measurePair(t *testing.T, dev string, minorA, minorB uint16, warm, measure time.Duration, start func()) (a, b bandResult) {
	t.Helper()
	start()
	time.Sleep(warm)
	t0 := time.Now()
	idle1, total1 := cpuJiffies()
	before := classStats(t, dev)
	time.Sleep(measure)
	elapsed := time.Since(t0).Seconds()
	idle2, total2 := cpuJiffies()
	after := classStats(t, dev)
	time.Sleep(4 * time.Second)
	cpu := 0.0
	if total2 > total1 {
		cpu = 100 * (1 - float64(idle2-idle1)/float64(total2-total1))
	}
	return bandFrom(before, after, minorA, elapsed, cpu), bandFrom(before, after, minorB, elapsed, cpu)
}

func bandFrom(before, after map[string]classStat, minor uint16, elapsed, cpu float64) bandResult {
	h := classHandle(minor)
	b0, b1 := before[h], after[h]
	mbps := float64(b1.Bytes-b0.Bytes) * 8 / elapsed / 1e6
	drops := uint64(0)
	if b1.Dropped >= b0.Dropped {
		drops = b1.Dropped - b0.Dropped
	}
	return bandResult{mbps: mbps, ratio: mbps / (float64(testCapBPS) / 1e6), drops: drops, cpu: cpu, elapsed: elapsed}
}

func assertRatio(t *testing.T, name string, ratio, low, high float64) {
	t.Helper()
	if ratio < low || ratio > high {
		t.Fatalf("%s ratio %.3f outside %.2f-%.2f", name, ratio, low, high)
	}
}

func launchIperf(t *testing.T, ns string, port, seconds int, pid string, extra ...string) {
	t.Helper()
	args := []string{"iperf3", "-c", "203.0.113.10", "-p", strconv.Itoa(port), "-t", strconv.Itoa(seconds), "-J"}
	args = append(args, extra...)
	script := fmt.Sprintf("echo $$ > %s; exec %s > %s.log 2>&1", pid, strings.Join(args, " "), pid)
	if err := dockerDetach("ip", "netns", "exec", ns, "sh", "-c", script); err != nil {
		t.Fatal(err)
	}
}

func restartIperf(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("ip", "netns", "exec", "fv-inet", "sh", "-c", "pids=$(pidof iperf3 || true); if [ -n \"$pids\" ]; then kill $pids || true; fi")
	time.Sleep(200 * time.Millisecond)
	for _, port := range []int{5201, 5202, 5203, 5204} {
		mustExec(t, "ip", "netns", "exec", "fv-inet", "iperf3", "-s", "-p", strconv.Itoa(port), "-D")
	}
}

func requireIperfBytes(t *testing.T, path string) {
	t.Helper()
	body := readScratch(t, path)
	if !strings.Contains(body, `"bytes"`) {
		t.Fatalf("iperf log %s has no byte count:\n%s", path, body)
	}
}
