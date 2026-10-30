//go:build netlab

package at

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/at/isolation"
)

// fastloopSource is a scratch-only probe. tests/netlab/scripts/denyloop.py
// sleeps between waves, so it cannot hold ten attempts a second (§19 AT10).
const fastloopSource = `#!/usr/bin/env python3
import json, os, socket, time, sys
from concurrent.futures import ThreadPoolExecutor

stop, stat = sys.argv[1], sys.argv[2]
dests = json.loads(sys.argv[3])

def one(item):
    sock = socket.socket()
    sock.settimeout(0.08)
    try:
        sock.connect((item[0], int(item[1])))
        return item[2]
    except OSError:
        return ""
    finally:
        sock.close()

pool = ThreadPoolExecutor(max_workers=8)
start = time.monotonic()
attempts = 0
allows = []
while not os.path.exists(stop):
    futs = [pool.submit(one, item) for item in dests]
    for fut in futs:
        attempts += 1
        hit = fut.result()
        if hit:
            allows.append(hit)
    elapsed = max(time.monotonic() - start, 0.001)
    json.dump({"attempts": attempts, "seconds": round(elapsed, 3), "allows": allows[-8:]}, open(stat, "w"))
pool.shutdown(wait=False)
`

func TestAT10_HostAndUpstream(t *testing.T) {
	skipLab(t)
	rec := labMeta("generation 3000-3020")
	rec.ID = "lab-AT10"
	rec.Title = "Mechanism: host and upstream isolation during apply and interface change"
	rec.Spec = []string{"§19 AT10", "ISO01", "ISO02"}
	rec.Status = isolation.StatusFail
	rec.Expected = "Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow."
	rec.Actual = "not finished"
	t.Cleanup(func() {
		_, _, _ = dockerExec("sh", "-c", "touch /scratch/fastloop.stop; sleep 0.3; pids=$(pidof tcpdump || true); if [ -n \"$pids\" ]; then kill $pids; fi")
		_, _, _ = dockerExec("ip", "link", "del", "fv-px")
		finish(t, rec)
	})
	flush(t)

	m := labModel(3000, allBinds(), nil)
	doc := applyModel(t, m)
	if doc.Admitted != 3 {
		fail(t, &rec, "admitted %d, want 3", doc.Admitted)
	}
	rules := ruleset(t)
	if !strings.Contains(rules, "policy drop") || !strings.Contains(rules, "fvpn_generation=3000") {
		fail(t, &rec, "ruleset missing drop policy or generation 3000")
	}
	_ = isolation.WriteExtra(evidenceDir(), "lab-AT10-ruleset.nft", rules)

	startCapture(t, "fv-lan", "fv-ll", "ip and (dst host 10.1.2.10 or dst host 192.168.10.10 or dst host 172.16.5.10 or dst host 169.254.169.254 or dst host 192.0.2.200)", "/scratch/cap-at10-lan.txt")
	startCapture(t, "fv-a2", "fv-wg0", "dst port 8080", "/scratch/cap-at10-peer.txt")
	startCapture(t, "root", "lo", "host 192.0.2.33 or host 192.0.2.37", "/scratch/cap-at10-host.txt")
	time.Sleep(200 * time.Millisecond)

	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/stream-at10.log"); err != nil {
		fail(t, &rec, "stream: %v", err)
	}
	if !waitLog(t, "/scratch/stream-at10.log", "ok", 8*time.Second) {
		fail(t, &rec, "allowed stream did not become healthy")
	}

	dests := [][]any{
		{"10.1.2.10", 80, "lan10"},
		{"192.168.10.10", 80, "lan192"},
		{"172.16.5.10", 80, "lan172"},
		{"169.254.169.254", 80, "metadata"},
		{"192.0.2.200", 80, "public-prefix"},
		{"192.0.2.33", 2222, "ssh"},
		{"100.64.1.2", 8080, "peer"},
	}
	body, _ := json.Marshal(dests)
	writeScratch(t, "/scratch/fastloop.py", []byte(fastloopSource))
	_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/fastloop.stop /scratch/fastloop.json")
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/scratch/fastloop.py", "/scratch/fastloop.stop", "/scratch/fastloop.json", string(body)); err != nil {
		fail(t, &rec, "fastloop: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	flap(t, &rec)
	for i := 1; i <= 20; i++ {
		gen := uint64(3000 + i)
		if applyModel(t, labModel(gen, allBinds(), nil)).Admitted != 3 {
			fail(t, &rec, "generation %d admitted the wrong set", gen)
		}
	}
	if strings.Contains(readScratch(t, "/scratch/stream-at10.log"), "fail") {
		fail(t, &rec, "allowed Internet flow failed during regeneration")
	}
	_, _, _ = dockerExec("sh", "-c", "touch /scratch/fastloop.stop")
	time.Sleep(400 * time.Millisecond)
	statRaw := readScratch(t, "/scratch/fastloop.json")
	var stat struct {
		Attempts int      `json:"attempts"`
		Seconds  float64  `json:"seconds"`
		Allows   []string `json:"allows"`
	}
	if json.Unmarshal([]byte(statRaw), &stat) != nil || stat.Seconds <= 0 {
		fail(t, &rec, "fastloop stats missing: %s", clipOne(statRaw))
	}
	rate := float64(stat.Attempts) / stat.Seconds
	if len(stat.Allows) > 0 {
		fail(t, &rec, "continuous probe saw an allow: %s", strings.Join(stat.Allows, ","))
	}
	if rate < 10 {
		fail(t, &rec, "continuous probe rate %.2f/s is below 10 (attempts=%d seconds=%.3f)", rate, stat.Attempts, stat.Seconds)
	}
	stopCaptures()
	for _, path := range []string{"/scratch/cap-at10-lan.txt", "/scratch/cap-at10-peer.txt", "/scratch/cap-at10-host.txt"} {
		if strings.TrimSpace(captureBody(t, path)) != "" {
			fail(t, &rec, "denied packet reached %s", path)
		}
	}
	matrix := runProbe(t, []probeCase{
		{"lan10", "fv-a1", "tcp", "10.1.2.10", 80, "", "deny", 400, "", "", "", ""},
		{"lan192", "fv-a1", "tcp", "192.168.10.10", 80, "", "deny", 400, "", "", "", ""},
		{"lan172", "fv-a1", "tcp", "172.16.5.10", 80, "", "deny", 400, "", "", "", ""},
		{"cgnat", "fv-a1", "tcp", "100.64.5.5", 80, "", "deny", 400, "", "", "", ""},
		{"meta", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 400, "", "", "", ""},
		{"pub", "fv-a1", "tcp", "192.0.2.200", 80, "", "deny", 400, "", "", "", ""},
		{"ssh", "fv-a1", "tcp", "192.0.2.33", 2222, "", "deny", 400, "", "", "", ""},
		{"api", "fv-a1", "tcp", "192.0.2.37", 8443, "", "deny", 400, "", "", "", ""},
		{"inet", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", ""},
	})
	text := matrixText(matrix)
	_ = isolation.WriteExtra(evidenceDir(), "lab-AT10-matrix.txt", text)
	if bad := mismatches(matrix); bad != "" {
		fail(t, &rec, "matrix:\n%s", bad)
	}
	rec.Status = isolation.StatusPass
	rec.Actual = fmt.Sprintf("20 regenerations, interface flap, rate %.1f attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.", rate)
	rec.Observations = []isolation.Observation{
		{Name: "attempts", Detail: fmt.Sprintf("%d", stat.Attempts)},
		{Name: "seconds", Detail: fmt.Sprintf("%.3f", stat.Seconds)},
		{Name: "rate_per_s", Detail: fmt.Sprintf("%.2f", rate)},
		{Name: "regenerations", Detail: "20"},
		{Name: "flap", Detail: "veth fv-px 198.18.0.0/30 inside the lab container"},
		{Name: "internet", Detail: "203.0.113.10:80 allowed"},
		{Name: "scope", Detail: "mechanism lab; product status is AT10.json"},
	}
}

func TestAT11_IPv6Metadata(t *testing.T) {
	skipLab(t)
	rec := labMeta("generation 11")
	rec.ID = "lab-AT11"
	rec.Title = "Mechanism: IPv6 block mode, metadata and special addresses"
	rec.Spec = []string{"§19 AT11", "ISO01", "ISO03", "§8.3"}
	rec.Status = isolation.StatusFail
	rec.Expected = "IPv6 mode block drops member IPv6, including ULA, link-local, loopback and IPv4-mapped. Metadata stays unreachable. There is no route mode in v1."
	rec.Actual = "not finished"
	t.Cleanup(func() {
		stopCaptures()
		finish(t, rec)
	})
	flush(t)
	m := labModel(11, allBinds(), nil)
	applyModel(t, m)
	before := counterPackets(t, m, "ctv6")
	startCapture(t, "fv-inet", "fv-ii", "ip6", "/scratch/cap-at11-inet.txt")
	startCapture(t, "fv-lan", "fv-ll", "ip6", "/scratch/cap-at11-lan.txt")
	time.Sleep(200 * time.Millisecond)

	native := runProbe(t, []probeCase{
		{"v6-global-tcp", "fv-a1", "tcp6", "2001:db8:203::10", 80, "", "deny", 800, "", "", "", ""},
		{"v6-global-icmp", "fv-a1", "icmp6", "2001:db8:203::10", 0, "", "deny", 800, "", "", "", ""},
		{"v6-ula", "fv-a1", "icmp6", "fd00:db8:ffff::10", 0, "", "deny", 800, "", "", "", ""},
		{"v6-doc", "fv-a1", "icmp6", "2001:db8::10", 0, "", "deny", 800, "", "", "", ""},
		{"v6-linklocal", "fv-a1", "icmp6", "fe80::10", 0, "", "deny", 800, "", "", "", ""},
		{"v6-mapped-inet", "fv-a1", "tcp6", "::ffff:203.0.113.10", 80, "", "deny", 800, "", "", "", ""},
		{"v6-mapped-lan", "fv-a1", "tcp6", "::ffff:10.1.2.10", 80, "", "deny", 800, "", "", "", ""},
		{"v4-metadata", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 400, "", "", "", ""},
		{"v4-loopback-connect", "fv-a1", "tcp", "127.0.0.1", 2222, "", "deny", 400, "", "", "", ""},
	})
	dstMAC := ifaceMAC(t, "root", "fv-fh")
	srcMAC := ifaceMAC(t, "fv-fwd", "fv-ff")
	raw := runProbe(t, []probeCase{
		{"raw-global", "fv-fwd", "raw6", "2001:db8:203::10", 9, "", "sent", 800, "fd00:db8:a1::1", srcMAC, dstMAC, "fv-ff"},
		{"raw-ula", "fv-fwd", "raw6", "fd00:db8:ffff::10", 9, "", "sent", 800, "fd00:db8:a1::1", srcMAC, dstMAC, "fv-ff"},
		{"raw-linklocal", "fv-fwd", "raw6", "fe80::10", 9, "", "sent", 800, "fe80::34", srcMAC, dstMAC, "fv-ff"},
		{"raw-mapped", "fv-fwd", "raw6", "::ffff:10.1.2.10", 9, "", "sent", 800, "fd00:db8:a1::1", srcMAC, dstMAC, "fv-ff"},
		{"raw-loopback", "fv-fwd", "raw4", "127.0.0.1", 9, "", "sent", 800, "100.64.1.1", srcMAC, dstMAC, "fv-ff"},
	})
	time.Sleep(300 * time.Millisecond)
	stopCaptures()
	inetCap := strings.TrimSpace(captureBody(t, "/scratch/cap-at11-inet.txt"))
	lanCap := strings.TrimSpace(captureBody(t, "/scratch/cap-at11-lan.txt"))
	if inetCap != "" || lanCap != "" {
		fail(t, &rec, "ipv6 reached a target")
	}
	delta := counterPackets(t, m, "ctv6") - before
	if delta == 0 {
		fail(t, &rec, "ctv6 did not increase")
	}
	if bad := mismatches(append(native, raw...)); bad != "" {
		fail(t, &rec, "probes:\n%s", bad)
	}
	coverage := "ipv6_mode=block\nroute_mode=not_implemented_v1\nctv6_delta=" + fmt.Sprintf("%d", delta) + "\ninet_capture=empty\nlan_capture=empty\nula=deny\nlink_local=deny\nloopback_connect=deny\nloopback_raw=sent_and_not_delivered\nmetadata=deny\nipv4_mapped=deny\n"
	_ = isolation.WriteExtra(evidenceDir(), "lab-AT11-coverage.txt", coverage+matrixText(append(native, raw...)))
	rec.Status = isolation.StatusPass
	rec.Actual = fmt.Sprintf("mode block, ctv6 +%d, special-address probes denied, target captures empty. Route mode was not claimed (§8.3).", delta)
	rec.Observations = []isolation.Observation{
		{Name: "ipv6_mode", Detail: string(enforce.IPv6Block)},
		{Name: "ctv6_delta", Detail: fmt.Sprintf("%d", delta)},
		{Name: "route_mode", Detail: "not implemented in v1"},
		{Name: "scope", Detail: "mechanism lab; product status is AT11.json"},
	}
}

func TestAT12_CrossMemberDNS(t *testing.T) {
	skipLab(t)
	rec := labMeta("generation 12")
	rec.ID = "lab-AT12"
	rec.Title = "Mechanism: cross-member, management plane and DNS exceptions"
	rec.Spec = []string{"§19 AT12", "ISO01", "ISO04", "DNS03"}
	rec.Status = isolation.StatusFail
	rec.Expected = "Only TCP and UDP 53 to the designated resolver succeed. Peer, SSH, agent API, arbitrary overlay and a spoofed source are denied."
	rec.Actual = "not finished"
	var all []probeResult
	t.Cleanup(func() {
		stopCaptures()
		_, _, _ = dockerExec("ip", "netns", "exec", "fv-fwd", "ip", "addr", "del", "100.64.7.7/32", "dev", "fv-ff")
		if err := isolation.WriteExtra(evidenceDir(), "lab-AT12-matrix.txt", matrixText(all)); err != nil {
			t.Errorf("matrix evidence: %v", err)
		}
		finish(t, rec)
	})
	flush(t)
	_, _, _ = dockerExec("ip", "netns", "exec", "fv-fwd", "ip", "addr", "del", "100.64.7.7/32", "dev", "fv-ff")
	mustExec(t, "ip", "netns", "exec", "fv-fwd", "ip", "addr", "add", "100.64.7.7/32", "dev", "fv-ff")
	m := labModel(12, allBinds(), nil)
	if applyModel(t, m).Admitted != 3 {
		fail(t, &rec, "admitted set")
	}
	// Host-originated UDP controls must arrive at each peer's ingress capture.
	// Stop them before the member deny matrix; they are not member forwarding.
	var captureEvidence strings.Builder
	for _, peer := range []struct{ ns, ip string }{{"fv-a1", "100.64.1.1"}, {"fv-a2", "100.64.1.2"}, {"fv-b1", "100.64.2.1"}} {
		path := "/scratch/cap-at12-control-" + peer.ns + ".txt"
		startCapture(t, peer.ns, "fv-wg0", "udp dst port 8080", path)
		mustExec(t, "python3", "-c", "import socket; s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM); s.sendto(b'control',('"+peer.ip+"',8080)); s.close()")
		if !waitLog(t, path, peer.ip+".8080", 3*time.Second) {
			fail(t, &rec, "ingress capture positive control missing for %s", peer.ns)
		}
		stopCaptures()
		fmt.Fprintf(&captureEvidence, "positive control %s direction=in:\n%s\n", peer.ns, captureBody(t, path))
	}
	t.Cleanup(func() {
		if err := isolation.WriteExtra(evidenceDir(), "lab-AT12-captures.txt", captureEvidence.String()); err != nil {
			t.Errorf("capture evidence: %v", err)
		}
	})
	startCapture(t, "fv-lan", "fv-ll", "ip", "/scratch/cap-at12-lan.txt")
	startCapture(t, "fv-a1", "fv-wg0", "dst port 8080", "/scratch/cap-at12-a1.txt")
	startCapture(t, "fv-a2", "fv-wg0", "dst port 8080", "/scratch/cap-at12-a2.txt")
	startCapture(t, "fv-b1", "fv-wg0", "dst port 8080", "/scratch/cap-at12-b1.txt")
	startCapture(t, "fv-inet", "fv-ii", "(dst host 198.51.100.53 and dst port 8080) or (dst host 203.0.113.10 and (tcp dst port 80 or icmp))", "/scratch/cap-at12-inet.txt")
	// Retain the old filter and an explicit outbound view to explain the old
	// false positive without weakening the target-ingress assertions.
	startDirectionalCapture(t, "fv-a2", "fv-wg0", "inout", "dst port 8080", "/scratch/cap-at12-a2-both.txt")
	startDirectionalCapture(t, "fv-a2", "fv-wg0", "out", "dst port 8080", "/scratch/cap-at12-a2-out.txt")
	denied := runProbe(t, []probeCase{
		{"a1-to-a2", "fv-a1", "tcp", "100.64.1.2", 8080, "", "deny", 800, "", "", "", ""},
		{"a1-to-b1", "fv-a1", "tcp", "100.64.2.1", 8080, "", "deny", 800, "", "", "", ""},
		{"a2-to-a1", "fv-a2", "tcp", "100.64.1.1", 8080, "", "deny", 800, "", "", "", ""},
		{"overlay", "fv-a1", "tcp", "100.64.5.5", 80, "", "deny", 800, "", "", "", ""},
		{"ssh", "fv-a1", "tcp", "192.0.2.33", 2222, "", "deny", 800, "", "", "", ""},
		{"api", "fv-a1", "tcp", "192.0.2.37", 8443, "", "deny", 800, "", "", "", ""},
		{"resolver-http", "fv-a1", "tcp", "198.51.100.53", 8080, "", "deny", 800, "", "", "", ""},
		{"rogue", "fv-rogue", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", ""},
		{"spoof", "fv-fwd", "icmp", "203.0.113.10", 0, "100.64.7.7", "deny", 800, "", "", "", ""},
	})
	all = append(all, denied...)
	stopCaptures()
	var reached []string
	for _, target := range []string{"lan", "a1", "a2", "b1", "inet"} {
		body := captureBody(t, "/scratch/cap-at12-"+target+".txt")
		fmt.Fprintf(&captureEvidence, "deny matrix %s direction=in packets=%d:\n%s\n", target, len(strings.FieldsFunc(strings.TrimSpace(body), func(r rune) bool { return r == '\n' })), body)
		if strings.TrimSpace(body) != "" {
			reached = append(reached, target)
		}
	}
	both := captureBody(t, "/scratch/cap-at12-a2-both.txt")
	outbound := captureBody(t, "/scratch/cap-at12-a2-out.txt")
	fmt.Fprintf(&captureEvidence, "a2 direction=inout (original filter):\n%s\na2 direction=out:\n%s\n", both, outbound)
	if len(reached) > 0 {
		fail(t, &rec, "denied packets reached target ingress: %s", strings.Join(reached, ", "))
	}
	if !strings.Contains(outbound, " > 100.64.1.1.8080:") || strings.TrimSpace(both) != strings.TrimSpace(outbound) {
		fail(t, &rec, "a2 bidirectional capture was not explained by the outbound probe; see lab-AT12-captures.txt")
	}
	dns := runProbe(t, []probeCase{
		{"dns-udp", "fv-a1", "dns", "198.51.100.53", 53, "", "allow", 800, "", "", "", ""},
		{"dns-tcp", "fv-a1", "tcp", "198.51.100.53", 53, "", "allow", 800, "", "", "", ""},
	})
	all = append(all, dns...)
	allow := runProbe(t, []probeCase{
		{"a1-inet", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", ""},
		{"a2-inet", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", ""},
		{"b1-inet", "fv-b1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", ""},
	})
	all = append(all, allow...)
	local := mustExec(t, "python3", "-c", "import socket; s=socket.create_connection(('127.0.0.1',2222),2); s.close(); print('local-ok')")
	if !strings.Contains(local, "local-ok") {
		fail(t, &rec, "host listener on 127.0.0.1:2222 failed")
	}
	if bad := mismatches(all); bad != "" {
		fail(t, &rec, "matrix:\n%s", bad)
	}
	rec.Status = isolation.StatusPass
	rec.Actual = "Two members and two devices of member A: peer, management, overlay, rogue and spoofed source denied. TCP and UDP 53 to 198.51.100.53 allowed. Host 127.0.0.1:2222 stayed up. Mechanism lab only."
	rec.Observations = []isolation.Observation{
		{Name: "capture_direction", Detail: "a2 inout equals out (a2-to-a1 SYN); a1/a2/b1/lan/inet ingress empty; three peer ingress positive controls observed; see lab-AT12-captures.txt"},
		{Name: "resolver", Detail: "198.51.100.53 tcp/53 udp/53 allow; tcp/8080 deny"},
		{Name: "spoof_source", Detail: "100.64.7.7 denied"},
		{Name: "members", Detail: "fv-a1 fv-a2 fv-b1"},
		{Name: "scope", Detail: "mechanism lab; product status is AT12.json"},
	}
}

func TestAT13_RevokeAndCrash(t *testing.T) {
	skipLab(t)
	rec := labMeta("generation 4000-4100")
	rec.ID = "lab-AT13"
	rec.Title = "Mechanism: revocation, agent SIGKILL and baseline"
	rec.Spec = []string{"§19 AT13", "ISO02", "ENT02", "SEC01"}
	rec.Status = isolation.StatusFail
	rec.Expected = "An established flow dies on revoke, conntrack for that source is removed, SIGKILL leaves the ruleset, and baseline keeps member forwarding off."
	rec.Actual = "not finished"
	t.Cleanup(func() {
		_, _, _ = dockerExec("sh", "-c", "pids=$(pidof fvpn-netlab || true); if [ -n \"$pids\" ]; then kill -9 $pids; fi")
		finish(t, rec)
	})
	flush(t)
	initial := labModel(4000, allBinds(), nil)
	applyModel(t, initial)
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/stream-at13.log"); err != nil {
		fail(t, &rec, "stream: %v", err)
	}
	if !waitLog(t, "/scratch/stream-at13.log", "ok", 8*time.Second) {
		fail(t, &rec, "stream did not establish")
	}
	ctBefore, ctErr, _ := dockerExec("conntrack", "-L")
	if !strings.Contains(ctBefore+ctErr, "100.64.1.1") {
		fail(t, &rec, "no conntrack entry for 100.64.1.1 before revoke")
	}
	revokedAt := time.Now()
	revoked := labModel(4001, []bind{
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, []string{"100.64.1.1"})
	applyModel(t, revoked)
	if !waitLog(t, "/scratch/stream-at13.log", "fail", 6*time.Second) {
		fail(t, &rec, "revoked flow kept succeeding")
	}
	ctAfter, ctAfterErr, _ := dockerExec("conntrack", "-L", "-s", "100.64.1.1")
	if strings.Contains(ctAfter+ctAfterErr, "100.64.1.1") {
		fail(t, &rec, "conntrack for the revoked source survived")
	}
	still := runProbe(t, []probeCase{
		{"a1-revoked", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", ""},
		{"a2-remains", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", ""},
	})
	if bad := mismatches(still); bad != "" {
		fail(t, &rec, "after revoke:\n%s", bad)
	}

	hold := labModel(4100, []bind{
		{memberA, devA2, "peer-a2", "100.64.1.2"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, []string{"100.64.1.1"})
	raw, _ := json.Marshal(hold)
	writeScratch(t, "/scratch/hold.json", raw)
	_, _, _ = dockerExec("rm", "-f", "/scratch/hold.ready")
	if err := dockerDetach("fvpn-netlab", "hold", "-f", "/scratch/hold.json"); err != nil {
		fail(t, &rec, "hold: %v", err)
	}
	if !waitLog(t, "/scratch/hold.ready", "ok", 10*time.Second) {
		fail(t, &rec, "hold did not become ready")
	}
	mustExec(t, "sh", "-c", "kill -9 $(pidof fvpn-netlab)")
	time.Sleep(200 * time.Millisecond)
	alive, _, _ := dockerExec("sh", "-c", "pidof fvpn-netlab || true")
	if strings.TrimSpace(alive) != "" {
		fail(t, &rec, "fvpn-netlab still running after SIGKILL")
	}
	rules := ruleset(t)
	if !strings.Contains(rules, "fvpn_generation=4100") || !strings.Contains(rules, "policy drop") {
		fail(t, &rec, "SIGKILL removed the ruleset")
	}
	afterKill := runProbe(t, []probeCase{
		{"a1-stays", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", ""},
		{"lan-stays", "fv-a2", "tcp", "10.1.2.10", 80, "", "deny", 800, "", "", "", ""},
		{"a2-stays", "fv-a2", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", ""},
	})
	if bad := mismatches(afterKill); bad != "" {
		fail(t, &rec, "after SIGKILL:\n%s", bad)
	}
	applyBaseline(t)
	base := runProbe(t, []probeCase{
		{"base-a1", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", ""},
		{"base-a2", "fv-a2", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", ""},
	})
	if bad := mismatches(base); bad != "" {
		fail(t, &rec, "baseline:\n%s", bad)
	}
	baseRules := ruleset(t)
	if !strings.Contains(baseRules, "fvpn_generation=0") {
		fail(t, &rec, "baseline generation was not 0")
	}
	local := mustExec(t, "python3", "-c", "import socket; s=socket.create_connection(('127.0.0.1',2222),2); s.close(); print('local-ok')")
	if !strings.Contains(local, "local-ok") {
		fail(t, &rec, "baseline broke the host listener")
	}
	timeline := fmt.Sprintf("establish=ok\nrevoke_at=%s\nconntrack_cleared=yes\nsigkill_generation=4100\nbaseline_generation=0\nhost_listener=up\nnetbird_restart=not_in_this_container\nnode_reboot=not_in_this_container\nupdater_switch=not_in_this_container\n", revokedAt.UTC().Format(time.RFC3339))
	_ = isolation.WriteExtra(evidenceDir(), "lab-AT13-timeline.txt", timeline)
	_ = isolation.WriteExtra(evidenceDir(), "lab-AT13-ruleset.nft", rules)
	rec.Status = isolation.StatusPass
	rec.Actual = "Revoke stopped the live flow and cleared its conntrack. SIGKILL left generation 4100. Baseline returned generation 0 and denied both members while 127.0.0.1:2222 stayed up. NetBird restart, node reboot and the updater switch are not in this container."
	rec.Observations = []isolation.Observation{
		{Name: "conntrack", Detail: "100.64.1.1 removed after revoke"},
		{Name: "sigkill", Detail: "generation 4100 remained"},
		{Name: "baseline", Detail: "generation 0, member forwarding denied"},
		{Name: "not_exercised", Detail: "NetBird restart, node reboot, updater switch"},
		{Name: "scope", Detail: "mechanism lab; product status is AT13.json"},
	}
}

func fail(t *testing.T, rec *isolation.Record, format string, args ...any) {
	t.Helper()
	rec.Status = isolation.StatusFail
	rec.Actual = fmt.Sprintf(format, args...)
	t.Fatal(rec.Actual)
}

func mismatches(results []probeResult) string {
	var bad []string
	for _, r := range results {
		if !r.OK {
			bad = append(bad, fmt.Sprintf("%s expect %s observed %s", r.Name, r.Expect, r.Observed))
		}
	}
	return strings.Join(bad, "\n")
}

func flush(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("conntrack", "-F")
}

func waitLog(t *testing.T, path, needle string, limit time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if scratchExists(path) && strings.Contains(readScratch(t, path), needle) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func flap(t *testing.T, rec *isolation.Record) {
	t.Helper()
	_, _, _ = dockerExec("ip", "link", "del", "fv-px")
	mustExec(t, "ip", "link", "add", "fv-px", "type", "veth", "peer", "name", "fv-py")
	mustExec(t, "ip", "addr", "add", "198.18.0.1/30", "dev", "fv-px")
	mustExec(t, "ip", "link", "set", "fv-px", "up")
	mustExec(t, "ip", "link", "set", "fv-py", "up")
	mustExec(t, "ip", "addr", "add", "198.18.0.2/30", "dev", "fv-py")
	mustExec(t, "ip", "addr", "add", "198.18.0.10/32", "dev", "fv-py")
	mustExec(t, "ip", "route", "add", "198.18.0.10/32", "via", "198.18.0.2", "dev", "fv-px")
	startCapture(t, "root", "fv-py", "ip and host 198.18.0.10", "/scratch/cap-at10-flap.txt")
	time.Sleep(200 * time.Millisecond)
	res := runProbe(t, []probeCase{{"flap", "fv-a1", "tcp", "198.18.0.10", 80, "", "deny", 800, "", "", "", ""}})
	if bad := mismatches(res); bad != "" {
		fail(t, rec, "flap probe: %s", bad)
	}
	stopOne("/scratch/cap-at10-flap.txt")
	time.Sleep(200 * time.Millisecond)
	if strings.TrimSpace(captureBody(t, "/scratch/cap-at10-flap.txt")) != "" {
		fail(t, rec, "flapped interface received a member packet")
	}
	mustExec(t, "ip", "link", "del", "fv-px")
}

func stopOne(path string) {
	_, _, _ = dockerExec("sh", "-c", "if [ -s "+path+".pid ]; then kill $(cat "+path+".pid) || true; fi")
}

func clipOne(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 180 {
		return s[:180]
	}
	return s
}
