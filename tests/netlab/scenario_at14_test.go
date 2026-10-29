//go:build netlab

package netlab

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func TestAT14_MTR01_MTR02_CountOnce(t *testing.T) {
	skipLab(t)
	quiesceLab(t)
	flushConntrack(t)
	mustExec(t, "ip", "netns", "exec", "fv-a1", "sysctl", "-w", "net.ipv6.conf.fv-wg0.disable_ipv6=1")
	now := time.Now().UTC()
	m := functionalModel(now, 14)
	applyAt(t, m)
	flushConntrack(t)

	var report strings.Builder
	fmt.Fprintf(&report, "metering=%s dns=%s diagnostic=%s\n", enforce.MeteringVersion, enforce.DNSNamespace, enforce.DiagnosticNamespace)
	fmt.Fprintf(&report, "reference=tcpdump -v on member fv-wg0, sum of IP total length (inner). ")
	fmt.Fprintf(&report, "ip -s link on fv-wg0 counts ciphertext on this kernel and is not the inner layer. underlay fv-a1u is not usage.\n")

	udpUp, udpDn := checkTransfer(t, m, &report, "udp", "203.0.113.10", 9000, 400, 1200)
	tcpUp, tcpDn := checkTransfer(t, m, &report, "tcp", "203.0.113.10", 9001, 80, 1000)
	if udpUp > 0.02 || udpDn > 0.02 {
		t.Fatalf("udp deviation upload %.4f download %.4f exceeds 2%%", udpUp, udpDn)
	}
	if tcpUp > 0.02 || tcpDn > 0.02 {
		t.Fatalf("tcp deviation upload %.4f download %.4f exceeds 2%%", tcpUp, tcpDn)
	}

	before := readCounters(t, m)
	dns := runProbe(t, []probeCase{
		{"d1", "fv-a1", "dns", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
		{"d2", "fv-a1", "dns", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
		{"d3", "fv-a1", "dns", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
		{"d4", "fv-a1", "tcp", "198.51.100.53", 53, "", "allow", 800, "", "", "", "", 0},
	})
	requireProbe(t, dns)
	after := readCounters(t, m)
	if sampleBytes(after.Samples, devA1, "upload") != sampleBytes(before.Samples, devA1, "upload") {
		t.Fatal("resolver traffic incremented forward_admission_v1")
	}
	if sampleBytes(after.Samples, devA1, "dns_upload") <= sampleBytes(before.Samples, devA1, "dns_upload") {
		t.Fatal("resolver traffic did not increment dns_service_v1")
	}
	fmt.Fprintf(&report, "dns_upload_delta=%d usage_upload_delta=0\n", sampleBytes(after.Samples, devA1, "dns_upload")-sampleBytes(before.Samples, devA1, "dns_upload"))

	before = after
	deny := runProbe(t, []probeCase{
		{"deny1", "fv-a1", "icmp", "10.1.2.10", 0, "", "deny", 800, "", "", "", "", 0},
		{"deny2", "fv-a1", "tcp", "192.168.10.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"deny3", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 800, "", "", "", "", 0},
	})
	requireProbe(t, deny)
	after = readCounters(t, m)
	if sampleBytes(after.Samples, devA1, "upload") != sampleBytes(before.Samples, devA1, "upload") {
		t.Fatal("denied traffic incremented usage")
	}
	fmt.Fprintf(&report, "deny_ctdeny_delta=%d\n", counterPackets(after.Counters, "ctdeny")-counterPackets(before.Counters, "ctdeny"))

	before = after
	host := mustExec(t, "python3", "-c", "import socket; s=socket.create_connection(('203.0.113.10',80),3); s.sendall(b'GET / HTTP/1.0\\r\\n\\r\\n'); s.recv(64); print('host-ok')")
	if !strings.Contains(host, "host-ok") {
		t.Fatalf("host control transfer failed: %s", host)
	}
	after = readCounters(t, m)
	if sampleBytes(after.Samples, devA1, "upload") != sampleBytes(before.Samples, devA1, "upload") || sampleBytes(after.Samples, devA2, "upload") != sampleBytes(before.Samples, devA2, "upload") {
		t.Fatal("host-originated traffic incremented a member counter")
	}
	fmt.Fprintf(&report, "host_control_usage_delta=0\n")

	prevRaw, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, "/scratch/prev.json", prevRaw)
	name := usageName("u_", devA1)
	mustExec(t, "nft", "reset", "counter", "inet", "fvpn", name)
	resetModel := functionalModel(time.Now().UTC(), 15)
	resetDoc := applyPrev(t, resetModel, "/scratch/prev.json")
	found := false
	for _, r := range resetDoc.Resets {
		if r.Name == name && r.Reason == enforce.ReasonMeterEpoch {
			found = true
		}
	}
	if !found {
		t.Fatalf("counter reset was not reported: %+v", resetDoc.Resets)
	}
	fmt.Fprintf(&report, "reset_reported=%s reason=%s\n", name, enforce.ReasonMeterEpoch)
	save(t, "meter.txt", report.String())
	fmt.Print(report.String())
	noteSummary(fmt.Sprintf("AT14 udp_up_dev=%.4f udp_down_dev=%.4f tcp_up_dev=%.4f tcp_down_dev=%.4f dns_separate=1 reset_reported=1", udpUp, udpDn, tcpUp, tcpDn))
}

func checkTransfer(t *testing.T, m enforce.Model, report *strings.Builder, proto, dst string, port, count, size int) (upDev, dnDev float64) {
	t.Helper()
	// §10.3 counts the inner IP length at the host forward hook. On this
	// kernel, fv-wg0 link stats count WireGuard ciphertext (inner + 36 bytes
	// for a padded UDP datagram: 4 pad-to-16 + 16 header + 16 tag). The
	// member netdev egress and inet output hooks also miss some TCP ACKs.
	// tcpdump -v on the member tunnel device sees the same IP total length
	// the forward counter records.
	path := "/scratch/cap-at14-" + proto + ".txt"
	script := fmt.Sprintf("echo $$ > %s.pid; exec tcpdump -v -n -l -s 128 -i fv-wg0 host 100.64.1.1 > %s 2>%s.err", path, path, path)
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "sh", "-c", script); err != nil {
		t.Fatal(err)
	}
	waitCapture(t, path)
	before := readCounters(t, m)
	wg0 := linkStats(t, "fv-a1", "fv-wg0")
	veth := linkStats(t, "fv-a1", "fv-a1u")
	result := blast(t, "fv-a1", proto, dst, port, count, size)
	time.Sleep(300 * time.Millisecond)
	after := readCounters(t, m)
	wg1 := linkStats(t, "fv-a1", "fv-wg0")
	veth1 := linkStats(t, "fv-a1", "fv-a1u")
	stopOneCapture(path)
	body := readScratch(t, path)
	upInner, dnInner, upPkts, dnPkts := sumInnerLengths(body, "100.64.1.1")
	upNFT := sampleBytes(after.Samples, devA1, "upload") - sampleBytes(before.Samples, devA1, "upload")
	dnNFT := sampleBytes(after.Samples, devA1, "download") - sampleBytes(before.Samples, devA1, "download")
	upWG := wg1.TX - wg0.TX
	dnWG := wg1.RX - wg0.RX
	upUnder := veth1.TX - veth.TX
	upDev = deviation(upNFT, upInner)
	dnDev = deviation(dnNFT, dnInner)
	if sampleBytes(after.Samples, devA2, "upload") != sampleBytes(before.Samples, devA2, "upload") || sampleBytes(after.Samples, devB1, "upload") != sampleBytes(before.Samples, devB1, "upload") {
		t.Fatalf("%s counted on the wrong device", proto)
	}
	if upInner == 0 || dnInner == 0 {
		t.Fatalf("%s inner capture empty up=%d down=%d err=%s", proto, upInner, dnInner, oneLine(execText("sh", "-c", "tail -c 400 "+path+".err || true")))
	}
	if upUnder <= upInner {
		t.Fatalf("%s underlay bytes %d are not above inner bytes %d", proto, upUnder, upInner)
	}
	fmt.Fprintf(report, "%s payload_sent=%d payload_got=%d nft_up=%d inner_up=%d pkts_up=%d dev_up=%.4f nft_down=%d inner_down=%d pkts_down=%d dev_down=%.4f wg_up=%d wg_down=%d underlay_tx=%d\n",
		proto, result.Sent, result.Got, upNFT, upInner, upPkts, upDev, dnNFT, dnInner, dnPkts, dnDev, upWG, dnWG, upUnder)
	if upDev > 0.02 || dnDev > 0.02 {
		t.Fatalf("%s deviation upload %.4f download %.4f exceeds 2%%", proto, upDev, dnDev)
	}
	return upDev, dnDev
}

func waitCapture(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		errText := execText("sh", "-c", "cat "+path+".err 2>/dev/null || true")
		if strings.Contains(errText, "listening") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("tcpdump did not start: %s", oneLine(execText("sh", "-c", "cat "+path+".err 2>/dev/null || true")))
}

var ipTotalLengthRE = regexp.MustCompile(`length (\d+)\)`)

// sumInnerLengths adds IP total lengths from tcpdump -v text. The length is
// on the "IP (" line. The next line that contains ">" names the direction.
func sumInnerLengths(text, overlay string) (up, down uint64, upPkts, dnPkts int) {
	var pending uint64
	var have bool
	for _, line := range strings.Split(text, "\n") {
		if m := ipTotalLengthRE.FindStringSubmatch(line); m != nil && strings.Contains(line, "IP (") {
			n, _ := strconv.ParseUint(m[1], 10, 64)
			if strings.Contains(line, ">") {
				addInner(&up, &down, &upPkts, &dnPkts, line, overlay, n)
				have = false
				continue
			}
			pending = n
			have = true
			continue
		}
		if !have || !strings.Contains(line, ">") {
			continue
		}
		addInner(&up, &down, &upPkts, &dnPkts, line, overlay, pending)
		have = false
	}
	return up, down, upPkts, dnPkts
}

func addInner(up, down *uint64, upPkts, dnPkts *int, line, overlay string, n uint64) {
	left, _, _ := strings.Cut(line, ">")
	if strings.Contains(left, overlay) {
		*up += n
		*upPkts++
	} else if strings.Contains(line, overlay) {
		*down += n
		*dnPkts++
	}
}

func TestSumInnerLengths(t *testing.T) {
	text := "" +
		"12:00:00.000000 IP (tos 0x0, ttl 64, id 1, offset 0, flags [DF], proto UDP (17), length 1228)\n" +
		"    100.64.1.1.53698 > 203.0.113.10.9000: UDP, length 1200\n" +
		"12:00:00.000100 IP (tos 0x0, ttl 64, id 2, offset 0, flags [DF], proto UDP (17), length 1228)\n" +
		"    203.0.113.10.9000 > 100.64.1.1.53698: UDP, length 1200\n" +
		"12:00:00.000200 IP (tos 0x0, ttl 64, id 3, offset 0, flags [DF], proto TCP (6), length 52)\n" +
		"    100.64.1.1.12345 > 203.0.113.10.9001: Flags [.], ack 1, win 502, length 0\n"
	up, down, upPkts, dnPkts := sumInnerLengths(text, "100.64.1.1")
	if up != 1280 || down != 1228 || upPkts != 2 || dnPkts != 1 {
		t.Fatalf("got up=%d (%d) down=%d (%d)", up, upPkts, down, dnPkts)
	}
}

func quiesceLab(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop")
	killPidFile("/scratch/denyloop.pid")
	killPidFile("/scratch/stream-a1.log.pid")
	_, _, _ = dockerExec("sh", "-c", "pids=$(pidof iperf3 || true); if [ -n \"$pids\" ]; then kill $pids; fi")
	_, _, _ = dockerExec("sh", "-c", "pids=$(pidof tcpdump || true); if [ -n \"$pids\" ]; then kill $pids; fi")
	time.Sleep(200 * time.Millisecond)
}

func deviation(got, ref uint64) float64 {
	if ref == 0 {
		return 1
	}
	return math.Abs(float64(got)-float64(ref)) / float64(ref)
}
