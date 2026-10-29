//go:build netlab

package netlab

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAT11_IPv6Blocked(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	now := time.Now().UTC()
	m := functionalModel(now, 11)
	applyAt(t, m)
	before := readCounters(t, m)

	startCapture(t, "fv-inet", "fv-ii", "ip6", "/scratch/cap-at11-inet.txt")
	startCapture(t, "fv-lan", "fv-ll", "ip6", "/scratch/cap-at11-lan.txt")
	time.Sleep(300 * time.Millisecond)

	native := runProbe(t, []probeCase{
		{"a1-v6-global-tcp", "fv-a1", "tcp6", "2001:db8:203::10", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-v6-global-icmp", "fv-a1", "icmp6", "2001:db8:203::10", 0, "", "deny", 800, "", "", "", "", 0},
		{"a1-v6-ula", "fv-a1", "icmp6", "fd00:db8:ffff::10", 0, "", "deny", 800, "", "", "", "", 0},
		{"a1-v6-doc", "fv-a1", "icmp6", "2001:db8::10", 0, "", "deny", 800, "", "", "", "", 0},
		{"a1-v6-linklocal", "fv-a1", "icmp6", "fe80::10", 0, "", "deny", 800, "", "", "", "", 0},
		{"a1-v6-mapped-inet", "fv-a1", "tcp6", "::ffff:203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"a1-v6-mapped-lan", "fv-a1", "tcp6", "::ffff:10.1.2.10", 80, "", "deny", 800, "", "", "", "", 0},
	})
	table := printProbe(t, "AT11 native", native)
	requireProbe(t, native)

	dstMAC := ifaceMAC(t, "root", "fv-fh")
	srcMAC := ifaceMAC(t, "fv-fwd", "fv-ff")
	raw := runProbe(t, []probeCase{
		{"raw-global", "fv-fwd", "raw6", "2001:db8:203::10", 9, "", "sent", 800, "fd00:db8:a1::1", srcMAC, dstMAC, "fv-ff", 0},
		{"raw-ula", "fv-fwd", "raw6", "fd00:db8:ffff::10", 9, "", "sent", 800, "fd00:db8:a1::1", srcMAC, dstMAC, "fv-ff", 0},
		{"raw-linklocal", "fv-fwd", "raw6", "fe80::10", 9, "", "sent", 800, "fe80::34", srcMAC, dstMAC, "fv-ff", 0},
		{"raw-mapped-lan", "fv-fwd", "raw6", "::ffff:10.1.2.10", 9, "", "sent", 800, "fd00:db8:a1::1", srcMAC, dstMAC, "fv-ff", 0},
	})
	table += printProbe(t, "AT11 raw6", raw)
	requireProbe(t, raw)
	time.Sleep(300 * time.Millisecond)
	stopCaptures(t)

	inetCap := captureBody(t, "/scratch/cap-at11-inet.txt")
	lanCap := captureBody(t, "/scratch/cap-at11-lan.txt")
	if strings.TrimSpace(inetCap) != "" || strings.TrimSpace(lanCap) != "" {
		t.Fatalf("ipv6 reached a target\ninet:\n%slan:\n%s", inetCap, lanCap)
	}
	after := readCounters(t, m)
	delta := counterPackets(after.Counters, "ctv6") - counterPackets(before.Counters, "ctv6")
	if delta == 0 {
		save(t, "debug.txt", debugPlane())
		t.Fatal("ctv6 did not increase; native ipv6 did not reach the forward hook")
	}
	body := table + fmt.Sprintf("ctv6_packets=%d\ninet_capture_empty=1\nlan_capture_empty=1\nipv6_mode=block\n", delta)
	body += "mapped_v4_note=tcp6 with IPV6_V6ONLY saw no connected session; a mapped address that the kernel rewrites onto the IPv4 path is the IPv4 policy, not an IPv6 bypass.\n"
	save(t, "at11.txt", body)
	save(t, "cap-inet.txt", "# empty\n")
	save(t, "cap-lan.txt", "# empty\n")
	noteSummary(fmt.Sprintf("AT11 ctv6_delta=%d targets_empty=1 mode=block", delta))
}
