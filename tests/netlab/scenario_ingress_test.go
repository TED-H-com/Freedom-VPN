//go:build netlab

package netlab

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestIngress_InjectedMemberFrames is the devstack follow-up 13 fixture.
// fvpn-netlab inject places one frame on the member interface ingress without
// editing table inet fvpn or a WireGuard peer table.
func TestIngress_InjectedMemberFrames(t *testing.T) {
	skipLab(t)
	cleanupIngressLab(t)
	t.Cleanup(func() { cleanupIngressLab(t) })

	now := time.Now().UTC()
	m := functionalModel(now, 80)
	applyAt(t, m)
	flushConntrack(t)

	rulesBefore := rulesetShape(t)
	wgNodeBefore := wgIdentity(wgDump(t, "fv-fwd"))
	memberBefore := mustExec(t, "ip", "-br", "addr", "show", "dev", "fv-fh")
	routeBefore := mustExec(t, "ip", "route")
	linkBefore := mustExec(t, "ip", "-br", "link", "show", "dev", "fv-fh")

	var non IngressObservation
	uplink := captureDuring(t, "root", "fv-ih", "udp port 39999", func() {
		non = injectAt(t, "fv-fh", "100.64.9.9", "203.0.113.10", "")
	})
	if non.Mechanism != MechanismVethMirred || non.LinkType != LinkEther {
		t.Fatalf("fv-fh transport %s type %d", non.Mechanism, non.LinkType)
	}
	if strings.TrimSpace(uplink) != "" {
		t.Fatalf("non-admitted frame reached fv-ih:\n%s", uplink)
	}
	if delta := counterDelta(non, "ctunku"); delta.Packets < 1 {
		t.Fatalf("ctunku did not move: %+v", delta)
	}
	if delta := counterDelta(non, "ctinv"); delta.Packets != 0 {
		t.Fatalf("ctinv moved: %+v", delta)
	}
	assertDeviceCountersHeld(t, non, "")

	var admitted IngressObservation
	uplink = captureDuring(t, "root", "fv-ih", "udp port 39999", func() {
		admitted = injectAt(t, "fv-fh", "100.64.1.1", "203.0.113.10", "")
	})
	if strings.TrimSpace(uplink) == "" {
		t.Fatal("admitted frame produced no uplink packet")
	}
	upName := usageName("u_", devA1)
	if counterDelta(admitted, upName).Packets < 1 {
		t.Fatalf("%s did not move", upName)
	}
	if counterDelta(admitted, "ctunku").Packets != 0 {
		t.Fatalf("admitted frame hit ctunku: %+v", counterDelta(admitted, "ctunku"))
	}
	assertDeviceCountersHeld(t, admitted, upName, usageName("d_", devA1))

	socketDelta := packetSocketEgress(t)
	if socketDelta != 0 {
		t.Fatalf("AF_PACKET on fv-fh moved ctunku by %d", socketDelta)
	}

	if rulesetShape(t) != rulesBefore {
		t.Fatal("inject rewrote table inet fvpn")
	}
	if wgIdentity(wgDump(t, "fv-fwd")) != wgNodeBefore {
		t.Fatal("inject changed the fv-fwd wireguard peer table")
	}
	if mustExec(t, "ip", "-br", "addr", "show", "dev", "fv-fh") != memberBefore {
		t.Fatal("inject changed fv-fh addresses")
	}
	if mustExec(t, "ip", "route") != routeBefore {
		t.Fatal("inject changed routes")
	}
	if mustExec(t, "ip", "-br", "link", "show", "dev", "fv-fh") != linkBefore {
		t.Fatal("inject changed fv-fh")
	}

	rawTun, rawVeth := rawInterfaceComparison(t)
	wgSep, wgCtunku := separateWireGuardPeer(t)
	if rulesetShape(t) != rulesBefore || wgIdentity(wgDump(t, "fv-fwd")) != wgNodeBefore {
		t.Fatal("mechanism comparison changed fvpn or the node wireguard peers")
	}

	fmt.Printf("SUMMARY INGRESS fv-fh=%s type=%d nonadmitted_ctunku=%d nonadmitted_uplink=0 admitted_%s=%d admitted_uplink=1 afpacket_ctunku=%d raw_tun=%d raw_veth=%d wg_separate=%d wg_ctunku=%d\n",
		non.Mechanism, non.LinkType,
		counterDelta(non, "ctunku").Packets,
		upName, counterDelta(admitted, upName).Packets,
		socketDelta, rawTun, rawVeth, wgSep, wgCtunku)
}

func injectAt(t *testing.T, iface, src, dst, mechanism string) IngressObservation {
	t.Helper()
	args := []string{"fvpn-netlab", "inject", "-iface", iface, "-src", src, "-dst", dst, "-port", "39999"}
	if mechanism != "" {
		args = append(args, "-mechanism", mechanism)
	}
	out, errOut, err := dockerExec(args...)
	if err != nil {
		t.Fatalf("inject %s %s: %v\n%s%s", iface, mechanism, err, out, errOut)
	}
	var obs IngressObservation
	if jerr := json.Unmarshal([]byte(out), &obs); jerr != nil {
		t.Fatalf("inject json: %v\n%s", jerr, out)
	}
	return obs
}

type pktDelta struct {
	Packets uint64
	Bytes   uint64
}

func counterDelta(obs IngressObservation, name string) pktDelta {
	before := obs.Before[name]
	after := obs.After[name]
	return pktDelta{Packets: after.Packets - before.Packets, Bytes: after.Bytes - before.Bytes}
}

func assertDeviceCountersHeld(t *testing.T, obs IngressObservation, allow ...string) {
	t.Helper()
	allowed := map[string]bool{}
	for _, name := range allow {
		allowed[name] = true
	}
	for name, after := range obs.After {
		if allowed[name] {
			continue
		}
		if !strings.HasPrefix(name, "u_") && !strings.HasPrefix(name, "d_") && !strings.HasPrefix(name, "nu_") && !strings.HasPrefix(name, "nd_") {
			continue
		}
		before := obs.Before[name]
		if after.Packets != before.Packets || after.Bytes != before.Bytes {
			t.Fatalf("counter %s moved from %+v to %+v", name, before, after)
		}
	}
}

func captureDuring(t *testing.T, ns, iface, filter string, fn func()) string {
	t.Helper()
	path := "/scratch/cap-ingress.txt"
	_, _, _ = dockerExec("rm", "-f", path, path+".err", path+".pid")
	startCapture(t, ns, iface, filter, path)
	deadline := time.Now().Add(2 * time.Second)
	for {
		errText := execText("sh", "-c", "cat "+path+".err 2>/dev/null || true")
		if strings.Contains(errText, "listening") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("capture %s did not start: %s", path, oneLine(errText))
		}
		time.Sleep(50 * time.Millisecond)
	}
	fn()
	time.Sleep(200 * time.Millisecond)
	stopCaptures(t)
	return captureBody(t, path)
}

func packetSocketEgress(t *testing.T) uint64 {
	t.Helper()
	before := namedTablePackets(t, "inet", "fvpn", "ctunku")
	mac := ifaceMAC(t, "root", "fv-fh")
	hw, err := net.ParseMAC(mac)
	if err != nil {
		t.Fatal(err)
	}
	bogus, err := net.ParseMAC("02:ff:00:00:00:01")
	if err != nil {
		t.Fatal(err)
	}
	ip := udpPacket(mustIP("100.64.9.9"), mustIP("203.0.113.10"), IngressSourcePort, 39999, []byte("x"))
	frame := ethernetFrame(bogus, hw, ip)
	script := `import socket,sys
frame=bytes.fromhex(sys.argv[1])
s=socket.socket(socket.AF_PACKET, socket.SOCK_RAW)
s.bind(("fv-fh", 0))
print(s.send(frame))
`
	body := captureDuring(t, "fv-fwd", "fv-ff", "ip src 100.64.9.9 and udp port 39999", func() {
		out, errOut, err := dockerExec("python3", "-c", script, hex.EncodeToString(frame))
		if err != nil {
			t.Fatalf("af_packet: %v\n%s%s", err, out, errOut)
		}
	})
	if strings.TrimSpace(body) == "" {
		t.Fatal("AF_PACKET send on fv-fh did not emerge on fv-ff")
	}
	after := namedTablePackets(t, "inet", "fvpn", "ctunku")
	if after < before {
		t.Fatalf("ctunku went backwards %d -> %d", before, after)
	}
	return after - before
}

func rawInterfaceComparison(t *testing.T) (tunDelta, vethDelta uint64) {
	t.Helper()
	cleanupIngressLab(t)
	bashLab(t, `
set -euo pipefail
WG=/usr/local/bin/wg
umask 077
mkdir -p /scratch/keys
$WG genkey > /scratch/keys/ingress-raw
ip link add fv-raw0 type wireguard
$WG set fv-raw0 private-key /scratch/keys/ingress-raw
ip link set fv-raw0 up
sysctl -w net.ipv4.conf.fv-raw0.rp_filter=0 net.ipv4.conf.fv-raw0.forwarding=1 >/dev/null
`)
	linkType := strings.TrimSpace(mustExec(t, "cat", "/sys/class/net/fv-raw0/type"))
	if linkType != strconv.Itoa(LinkNone) {
		t.Fatalf("fv-raw0 type %s", linkType)
	}
	ctBefore := namedTablePackets(t, "inet", "fvpn", "ctunku")
	armSideCounter(t, "fv-raw0", "rawin")
	veth := injectAt(t, "fv-raw0", "100.64.9.9", "203.0.113.10", MechanismVethMirred)
	if veth.Mechanism != MechanismVethMirred || veth.LinkType != LinkNone {
		t.Fatalf("forced veth observation mechanism %s type %d", veth.Mechanism, veth.LinkType)
	}
	vethDelta = sideCounter(t, "rawin")
	armSideCounter(t, "fv-raw0", "rawin")
	tun := injectAt(t, "fv-raw0", "100.64.9.9", "203.0.113.10", "")
	if tun.Mechanism != MechanismTunMirred || tun.LinkType != LinkNone {
		t.Fatalf("raw auto transport %s type %d", tun.Mechanism, tun.LinkType)
	}
	tunDelta = sideCounter(t, "rawin")
	ctAfter := namedTablePackets(t, "inet", "fvpn", "ctunku")
	if tunDelta < 1 {
		t.Fatal("tun-mirred onto a raw interface did not reach the forward hook")
	}
	if vethDelta < 1 {
		t.Fatal("veth-mirred onto a raw interface did not deliver the injected tuple")
	}
	if ctAfter != ctBefore {
		t.Fatalf("raw inject moved ctunku %d -> %d", ctBefore, ctAfter)
	}
	if counterDelta(tun, "ctunku").Packets != 0 || counterDelta(veth, "ctunku").Packets != 0 {
		t.Fatal("raw inject observation shows ctunku moving")
	}
	return tunDelta, vethDelta
}

func armSideCounter(t *testing.T, iface, name string) {
	t.Helper()
	bashLab(t, fmt.Sprintf(`
set -euo pipefail
nft delete table inet fvpinj 2>/dev/null || true
nft add table inet fvpinj
nft add counter inet fvpinj %s
nft add chain inet fvpinj pre '{ type filter hook forward priority -160 ; policy accept ; }'
nft add rule inet fvpinj pre iifname "%s" ip saddr 100.64.9.9 ip daddr 203.0.113.10 udp dport 39999 counter name "%s"
`, name, iface, name))
}

func separateWireGuardPeer(t *testing.T) (ingressDelta, ctunkuDelta uint64) {
	t.Helper()
	cleanupIngressLab(t)
	ctBefore := namedTablePackets(t, "inet", "fvpn", "ctunku")
	bashLab(t, `
set -euo pipefail
WG=/usr/local/bin/wg
umask 077
mkdir -p /scratch/keys
$WG genkey | tee /scratch/keys/ingress-node | $WG pubkey > /scratch/keys/ingress-node.pub
$WG genkey | tee /scratch/keys/ingress-peer | $WG pubkey > /scratch/keys/ingress-peer.pub
ip netns add fv-wgp
ip link add fv-wgu type veth peer name fv-wgg
ip addr add 192.0.2.80/30 dev fv-wgu
ip link set fv-wgu up
ip link set fv-wgg netns fv-wgp
ip netns exec fv-wgp ip addr add 192.0.2.81/30 dev fv-wgg
ip netns exec fv-wgp ip link set fv-wgg up
ip netns exec fv-wgp ip link set lo up
ip link add fv-lab0 type wireguard
ip netns exec fv-wgp ip link add fv-wgp0 type wireguard
$WG set fv-lab0 private-key /scratch/keys/ingress-node listen-port 51829 peer "$(cat /scratch/keys/ingress-peer.pub)" allowed-ips 100.64.50.2/32
ip addr add 100.64.50.1/32 dev fv-lab0
ip link set fv-lab0 up
sysctl -w net.ipv4.conf.fv-lab0.rp_filter=0 net.ipv4.conf.fv-lab0.forwarding=1 >/dev/null
ip netns exec fv-wgp $WG set fv-wgp0 private-key /scratch/keys/ingress-peer peer "$(cat /scratch/keys/ingress-node.pub)" endpoint 192.0.2.80:51829 allowed-ips 0.0.0.0/0
ip netns exec fv-wgp ip addr add 100.64.50.2/32 dev fv-wgp0
ip netns exec fv-wgp ip link set fv-wgp0 up
ip netns exec fv-wgp ip route replace 203.0.113.10/32 dev fv-wgp0
ip netns exec fv-wgp sysctl -w net.ipv4.conf.all.rp_filter=0 net.ipv4.conf.fv-wgp0.rp_filter=0 >/dev/null
nft add table inet fvpinj
nft add counter inet fvpinj wgin
nft add chain inet fvpinj pre '{ type filter hook forward priority -160 ; policy accept ; }'
nft add rule inet fvpinj pre iifname "fv-lab0" ip saddr 100.64.50.2 ip daddr 203.0.113.10 udp dport 39999 counter name "wgin"
`)
	bashLab(t, `
set -euo pipefail
ip netns exec fv-wgp bash -c 'echo -n x >/dev/udp/203.0.113.10/39999'
ok=0
for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  if /usr/local/bin/wg show fv-lab0 latest-handshakes | awk '$2 > 0 { found=1 } END { exit !found }'; then
    ok=1
    break
  fi
  sleep 0.2
done
if [ "$ok" != 1 ]; then
  echo '--- node ---' >&2
  /usr/local/bin/wg show fv-lab0 >&2 || true
  echo '--- peer ---' >&2
  ip netns exec fv-wgp /usr/local/bin/wg show >&2 || true
  exit 1
fi
ip netns exec fv-wgp bash -c 'echo -n x >/dev/udp/203.0.113.10/39999'
`)
	time.Sleep(300 * time.Millisecond)
	ingressDelta = sideCounter(t, "wgin")
	ctAfter := namedTablePackets(t, "inet", "fvpn", "ctunku")
	if ctAfter < ctBefore {
		t.Fatalf("ctunku went backwards %d -> %d", ctBefore, ctAfter)
	}
	ctunkuDelta = ctAfter - ctBefore
	if ingressDelta < 1 {
		dump := execText("/usr/local/bin/wg", "show", "fv-lab0")
		t.Fatalf("separate wireguard peer delivered no forward packet on fv-lab0\n%s", dump)
	}
	if ctunkuDelta != 0 {
		t.Fatalf("separate wireguard peer moved fv-fh ctunku by %d", ctunkuDelta)
	}
	return ingressDelta, ctunkuDelta
}

func sideCounter(t *testing.T, name string) uint64 {
	t.Helper()
	return namedTablePackets(t, "inet", "fvpinj", name)
}

func namedTablePackets(t *testing.T, family, table, name string) uint64 {
	t.Helper()
	out, errOut, err := dockerExec("nft", "list", "counter", family, table, name)
	if err != nil {
		t.Fatalf("nft list counter %s %s %s: %v\n%s%s", family, table, name, err, out, errOut)
	}
	m := regexp.MustCompile(`packets\s+(\d+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("counter %s has no packets line:\n%s", name, out)
	}
	n, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func rulesetShape(t *testing.T) string {
	t.Helper()
	return mustExec(t, "nft", "-s", "list", "table", "inet", "fvpn")
}

// wgIdentity keeps the interface and the peer keys, endpoints, and allowed
// addresses. Handshake times and transfer counters move when a decrypted
// packet flows, and those counters are not the peer table.
func wgIdentity(dump string) string {
	var b strings.Builder
	for _, line := range strings.Split(dump, "\n") {
		f := strings.Fields(line)
		if len(f) >= 9 {
			fmt.Fprintf(&b, "%s %s %s %s %s %s\n", f[0], f[1], f[2], f[3], f[4], f[8])
			continue
		}
		if len(f) > 0 {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func wgDump(t *testing.T, ns string) string {
	t.Helper()
	args := []string{"/usr/local/bin/wg", "show", "all", "dump"}
	if ns != "" && ns != "root" {
		args = append([]string{"ip", "netns", "exec", ns}, args...)
	}
	out, errOut, err := dockerExec(args...)
	if err != nil {
		t.Fatalf("wg show %s: %v\n%s%s", ns, err, out, errOut)
	}
	return out
}

func bashLab(t *testing.T, script string) string {
	t.Helper()
	out, errOut, err := dockerExecStdin([]string{"bash", "-s"}, []byte(script))
	if err != nil {
		t.Fatalf("bash: %v\n%s%s\n-- script --\n%s", err, out, errOut, script)
	}
	return out
}

func cleanupIngressLab(t *testing.T) {
	t.Helper()
	script := `
nft delete table inet fvpinj 2>/dev/null || true
ip link del fv-inj0 2>/dev/null || true
ip link del fv-raw0 2>/dev/null || true
ip link del fv-lab0 2>/dev/null || true
ip netns del fv-wgp 2>/dev/null || true
ip link del fv-wgu 2>/dev/null || true
`
	_, _, _ = dockerExecStdin([]string{"bash", "-s"}, []byte(script))
}
