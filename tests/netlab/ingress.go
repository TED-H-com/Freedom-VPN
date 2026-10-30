package netlab

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"golang.org/x/sys/unix"
)

const (
	// MechanismVethMirred redirects one ethernet frame from a lab veth onto
	// the member interface ingress. It is the transport for ARPHRD_ETHER.
	MechanismVethMirred = "veth-mirred"
	// MechanismTunMirred writes one raw IPv4 packet into a lab TUN and
	// redirects that ingress onto the member interface. It is the transport
	// for ARPHRD_NONE, which is devstack's fvmb0.
	MechanismTunMirred = "tun-mirred"

	// LinkEther is /sys/class/net/<iface>/type for an ethernet device.
	LinkEther = 1
	// LinkNone is /sys/class/net/<iface>/type for a headerless device
	// (WireGuard, TUN). tcpdump reports that link type as RAW.
	LinkNone = 65534

	// IngressSourcePort is the UDP source port of an injected frame.
	IngressSourcePort = 40000

	injDev  = "fv-inj0"
	injPeer = "fv-inj1"

	ingressSettle = 200 * time.Millisecond
)

// IngressSpec is one IPv4 frame to place on a member interface's ingress.
// Dest is an address beyond the node, so a forwarded frame hits the forward
// chain. Mechanism is empty unless a lab comparison forces veth-mirred or
// tun-mirred; devstack leaves it empty.
type IngressSpec struct {
	MemberIface string
	Source      netip.Addr
	Dest        netip.Addr
	DestPort    uint16
	Mechanism   string
}

// IngressObservation is the read-only named-counter snapshot around one
// inject. Before and After are the absolute nft readings of table inet fvpn.
type IngressObservation struct {
	Mechanism   string                          `json:"mechanism"`
	LinkType    int                             `json:"link_type"`
	MemberIface string                          `json:"member_iface"`
	Source      string                          `json:"source"`
	Dest        string                          `json:"dest"`
	SourcePort  uint16                          `json:"source_port"`
	DestPort    uint16                          `json:"dest_port"`
	Before      map[string]enforce.CounterValue `json:"before"`
	After       map[string]enforce.CounterValue `json:"after"`
}

// InjectMemberIngress runs in the node network namespace. It injects one
// UDP/IPv4 frame onto MemberIface's ingress and returns the named counters
// from before and after that frame.
//
// The frame is not a WireGuard decryption and it does not change a WireGuard
// peer, a NetBird peer, or table inet fvpn. The lab devices it creates
// (fv-inj0, and fv-inj1 for ethernet) are removed before it returns. It needs
// CAP_NET_ADMIN. The ethernet transport also needs CAP_NET_RAW.
//
// Call it from fvpn-netlab inject, executed inside the namespace. On the
// devstack node that is docker exec of this binary with -iface fvmb0. The
// link type selects the transport: 1 is veth-mirred, 65534 is tun-mirred.
func InjectMemberIngress(ctx context.Context, spec IngressSpec) (IngressObservation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var obs IngressObservation
	if err := validateSpec(&spec); err != nil {
		return obs, err
	}
	linkType, err := readLinkType(spec.MemberIface)
	if err != nil {
		return obs, err
	}
	mechanism, err := normalizeMechanism(spec.Mechanism, linkType)
	if err != nil {
		return obs, err
	}
	before, err := readNamedCounters(ctx)
	if err != nil {
		return obs, err
	}
	removeInjector()
	defer removeInjector()

	switch mechanism {
	case MechanismVethMirred:
		err = injectVeth(ctx, spec)
	case MechanismTunMirred:
		err = injectTun(ctx, spec)
	default:
		err = fmt.Errorf("unknown ingress mechanism %q", mechanism)
	}
	if err != nil {
		return obs, err
	}
	if err := waitSettle(ctx); err != nil {
		return obs, err
	}
	after, err := readNamedCounters(ctx)
	if err != nil {
		return obs, err
	}
	obs = IngressObservation{
		Mechanism:   mechanism,
		LinkType:    linkType,
		MemberIface: spec.MemberIface,
		Source:      spec.Source.String(),
		Dest:        spec.Dest.String(),
		SourcePort:  IngressSourcePort,
		DestPort:    spec.DestPort,
		Before:      before,
		After:       after,
	}
	return obs, nil
}

func validateSpec(spec *IngressSpec) error {
	if !validIface(spec.MemberIface) {
		return fmt.Errorf("member interface %q is not a device name", spec.MemberIface)
	}
	if spec.MemberIface == injDev || spec.MemberIface == injPeer {
		return fmt.Errorf("member interface %s is reserved for the injector", spec.MemberIface)
	}
	if !spec.Source.Is4() || !spec.Dest.Is4() {
		return fmt.Errorf("source and dest must be IPv4")
	}
	if spec.DestPort == 0 {
		spec.DestPort = 9
	}
	return nil
}

func validIface(name string) bool {
	if name == "" || len(name) > 15 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func normalizeMechanism(requested string, linkType int) (string, error) {
	switch requested {
	case "":
		return mechanismFor(linkType)
	case MechanismVethMirred, MechanismTunMirred:
		return requested, nil
	default:
		return "", fmt.Errorf("unknown ingress mechanism %q", requested)
	}
}

func mechanismFor(linkType int) (string, error) {
	switch linkType {
	case LinkEther:
		return MechanismVethMirred, nil
	case LinkNone:
		return MechanismTunMirred, nil
	default:
		return "", fmt.Errorf("member interface type %d has no ingress transport", linkType)
	}
}

func injectVeth(ctx context.Context, spec IngressSpec) error {
	loadRedirectModules(ctx)
	if err := run(ctx, "ip", "link", "add", injDev, "type", "veth", "peer", "name", injPeer); err != nil {
		return err
	}
	if err := run(ctx, "ip", "link", "set", injDev, "up"); err != nil {
		return err
	}
	if err := run(ctx, "ip", "link", "set", injPeer, "up"); err != nil {
		return err
	}
	// ethtool is absent on the devstack node image. This transport is the
	// ethernet one; a missing ethtool leaves the veth defaults.
	_ = run(ctx, "ethtool", "-K", injDev, "gro", "off", "gso", "off", "tso", "off")
	if err := installRedirect(ctx, spec.MemberIface); err != nil {
		return err
	}
	dst, err := ifaceHardware(injDev)
	if err != nil {
		return err
	}
	src, err := ifaceHardware(injPeer)
	if err != nil {
		return err
	}
	pkt := udpPacket(spec.Source, spec.Dest, IngressSourcePort, spec.DestPort, []byte("x"))
	return sendEthernet(injPeer, ethernetFrame(dst, src, pkt))
}

func injectTun(ctx context.Context, spec IngressSpec) error {
	loadRedirectModules(ctx)
	if err := run(ctx, "ip", "tuntap", "add", "mode", "tun", "name", injDev); err != nil {
		return err
	}
	if err := run(ctx, "ip", "link", "set", injDev, "up"); err != nil {
		return err
	}
	if err := installRedirect(ctx, spec.MemberIface); err != nil {
		return err
	}
	pkt := udpPacket(spec.Source, spec.Dest, IngressSourcePort, spec.DestPort, []byte("x"))
	return writeTun(injDev, pkt)
}

func loadRedirectModules(ctx context.Context) {
	// The devstack and lab images do not ship these modules. On a kernel
	// where they are already built in or loaded, modprobe failing is fine.
	for _, name := range []string{"sch_ingress", "act_mirred", "cls_matchall"} {
		_ = run(ctx, "modprobe", name)
	}
}

func installRedirect(ctx context.Context, member string) error {
	if err := run(ctx, "tc", "qdisc", "add", "dev", injDev, "clsact"); err != nil {
		return err
	}
	return run(ctx, "tc", "filter", "add", "dev", injDev, "ingress", "matchall",
		"action", "mirred", "ingress", "redirect", "dev", member)
}

func removeInjector() {
	ctx := context.Background()
	_ = run(ctx, "ip", "link", "del", injDev)
}

func readLinkType(iface string) (int, error) {
	raw, err := os.ReadFile("/sys/class/net/" + iface + "/type")
	if err != nil {
		return 0, fmt.Errorf("member interface %s: %w", iface, err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return 0, fmt.Errorf("member interface %s type: %w", iface, err)
	}
	return n, nil
}

func ifaceHardware(name string) (net.HardwareAddr, error) {
	raw, err := os.ReadFile("/sys/class/net/" + name + "/address")
	if err != nil {
		return nil, err
	}
	mac, err := net.ParseMAC(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("%s address: %w", name, err)
	}
	return mac, nil
}

func readNamedCounters(ctx context.Context) (map[string]enforce.CounterValue, error) {
	cmd := exec.CommandContext(ctx, "nft", "-j", "list", "table", enforce.TableFamily, enforce.TableName)
	out, err := cmd.Output()
	if err != nil {
		msg := ""
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			msg = strings.TrimSpace(string(exit.Stderr))
		}
		return nil, fmt.Errorf("nft list: %w: %s", err, msg)
	}
	rs, err := enforce.ParseRuleset(out)
	if err != nil {
		return nil, err
	}
	if rs.Counters == nil {
		rs.Counters = map[string]enforce.CounterValue{}
	}
	return rs.Counters, nil
}

func run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func waitSettle(ctx context.Context) error {
	timer := time.NewTimer(ingressSettle)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func ethernetFrame(dst, src net.HardwareAddr, ip []byte) []byte {
	frame := make([]byte, 14+len(ip))
	copy(frame[0:6], dst)
	copy(frame[6:12], src)
	frame[12] = 0x08
	frame[13] = 0x00
	copy(frame[14:], ip)
	if len(frame) < 60 {
		padded := make([]byte, 60)
		copy(padded, frame)
		frame = padded
	}
	return frame
}

func sendEthernet(iface string, frame []byte) error {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return err
	}
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	sa := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ALL),
		Ifindex:  ifi.Index,
	}
	if err := unix.Bind(fd, sa); err != nil {
		return err
	}
	if err := unix.Sendto(fd, frame, 0, sa); err != nil {
		return err
	}
	return nil
}

func writeTun(name string, pkt []byte) error {
	fd, err := unix.Open("/dev/net/tun", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	var ifr struct {
		Name  [unix.IFNAMSIZ]byte
		Flags uint16
		_     [22]byte
	}
	copy(ifr.Name[:], name)
	ifr.Flags = unix.IFF_TUN | unix.IFF_NO_PI
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		return os.NewSyscallError("ioctl TUNSETIFF", errno)
	}
	n, err := unix.Write(fd, pkt)
	if err != nil {
		return err
	}
	if n != len(pkt) {
		return fmt.Errorf("tun write wrote %d of %d", n, len(pkt))
	}
	return nil
}

func htons(v uint16) uint16 {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	return binary.NativeEndian.Uint16(buf[:])
}

// udpPacket is an IPv4 UDP datagram. The UDP checksum is 0, which IPv4 treats
// as "no checksum". The IPv4 header checksum is computed.
func udpPacket(src, dst netip.Addr, sport, dport uint16, payload []byte) []byte {
	udpLen := 8 + len(payload)
	total := 20 + udpLen
	pkt := make([]byte, total)
	pkt[0] = 0x45
	binary.BigEndian.PutUint16(pkt[2:4], uint16(total))
	pkt[8] = 64
	pkt[9] = 17
	copy(pkt[12:16], src.AsSlice())
	copy(pkt[16:20], dst.AsSlice())
	binary.BigEndian.PutUint16(pkt[10:12], ipChecksum(pkt[:20]))
	binary.BigEndian.PutUint16(pkt[20:22], sport)
	binary.BigEndian.PutUint16(pkt[22:24], dport)
	binary.BigEndian.PutUint16(pkt[24:26], uint16(udpLen))
	copy(pkt[28:], payload)
	return pkt
}

func ipChecksum(header []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(header); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
