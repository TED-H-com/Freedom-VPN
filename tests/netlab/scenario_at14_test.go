//go:build netlab

package netlab

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func TestAT14_MTR01_MTR02_CountOnce(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	mustExec(t, "ip", "netns", "exec", "fv-a1", "sysctl", "-w", "net.ipv6.conf.fv-wg0.disable_ipv6=1")
	now := time.Now().UTC()
	m := functionalModel(now, 14)
	applyAt(t, m)
	flushConntrack(t)

	var report strings.Builder
	fmt.Fprintf(&report, "metering=%s dns=%s diagnostic=%s\n", enforce.MeteringVersion, enforce.DNSNamespace, enforce.DiagnosticNamespace)
	fmt.Fprintf(&report, "reference=member fv-wg0 inner stats64; underlay fv-a1u is not usage\n")

	upDev, dnDev := checkTransfer(t, m, &report, "udp", "203.0.113.10", 9000, 400, 1200)
	checkTransfer(t, m, &report, "tcp", "203.0.113.10", 9001, 80, 1000)
	if upDev > 0.02 || dnDev > 0.02 {
		t.Fatalf("udp deviation upload %.4f download %.4f exceeds 2%%", upDev, dnDev)
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
	noteSummary(fmt.Sprintf("AT14 udp_up_dev=%.4f udp_down_dev=%.4f dns_separate=1 reset_reported=1", upDev, dnDev))
}

func checkTransfer(t *testing.T, m enforce.Model, report *strings.Builder, proto, dst string, port, count, size int) (upDev, dnDev float64) {
	t.Helper()
	before := readCounters(t, m)
	wg0 := linkStats(t, "fv-a1", "fv-wg0")
	veth := linkStats(t, "fv-a1", "fv-a1u")
	result := blast(t, "fv-a1", proto, dst, port, count, size)
	after := readCounters(t, m)
	wg1 := linkStats(t, "fv-a1", "fv-wg0")
	veth1 := linkStats(t, "fv-a1", "fv-a1u")
	upNFT := sampleBytes(after.Samples, devA1, "upload") - sampleBytes(before.Samples, devA1, "upload")
	dnNFT := sampleBytes(after.Samples, devA1, "download") - sampleBytes(before.Samples, devA1, "download")
	upWG := wg1.TX - wg0.TX
	dnWG := wg1.RX - wg0.RX
	upUnder := veth1.TX - veth.TX
	upDev = deviation(upNFT, upWG)
	dnDev = deviation(dnNFT, dnWG)
	if sampleBytes(after.Samples, devA2, "upload") != sampleBytes(before.Samples, devA2, "upload") || sampleBytes(after.Samples, devB1, "upload") != sampleBytes(before.Samples, devB1, "upload") {
		t.Fatalf("%s counted on the wrong device", proto)
	}
	if upUnder <= upWG {
		t.Fatalf("%s underlay bytes %d are not above inner bytes %d", proto, upUnder, upWG)
	}
	fmt.Fprintf(report, "%s payload_sent=%d payload_got=%d nft_up=%d wg_up=%d dev_up=%.4f nft_down=%d wg_down=%d dev_down=%.4f underlay_tx=%d\n",
		proto, result.Sent, result.Got, upNFT, upWG, upDev, dnNFT, dnWG, dnDev, upUnder)
	if proto == "tcp" && (upDev > 0.02 || dnDev > 0.02) {
		t.Fatalf("tcp deviation upload %.4f download %.4f exceeds 2%%", upDev, dnDev)
	}
	return upDev, dnDev
}

func deviation(got, ref uint64) float64 {
	if ref == 0 {
		return 1
	}
	return math.Abs(float64(got)-float64(ref)) / float64(ref)
}
