package enforce

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Interface names we create start with fv (AGENTS.md §4). A hyphen may follow
// immediately, as in fv-fh. The total length stays inside IFNAMSIZ (15).
var ifNameRE = regexp.MustCompile(`^fv[a-z0-9_-]{1,13}$`)

func validateIfName(name string) error {
	if !ifNameRE.MatchString(name) {
		return fmt.Errorf("interface %q must match %s", name, ifNameRE.String())
	}
	return nil
}

func validateDatapath(dp Datapath) error {
	if err := validateIfName(dp.MemberIngress); err != nil {
		return err
	}
	if err := validateIfName(dp.Uplink); err != nil {
		return err
	}
	if dp.MemberIngress == dp.Uplink {
		return fmt.Errorf("member ingress and uplink must be different interfaces")
	}
	if !dp.SNAT.IsValid() || !dp.SNAT.Is4() || dp.SNAT.IsUnspecified() || dp.SNAT.IsMulticast() || dp.SNAT.IsLoopback() {
		return fmt.Errorf("snat address %s is not a usable IPv4 address", dp.SNAT)
	}
	return nil
}

type admittedDevice struct {
	Binding Binding
	Lease   MemberLease
	Addrs   []netip.Addr
	Timeout uint32
	Minor   uint16
}

func validateServing(m Model) ([]admittedDevice, map[uuid.UUID]uint16, []netip.Prefix, error) {
	if m.Generation == 0 {
		return nil, nil, nil, fmt.Errorf("serving generation must be non-zero; 0 is the boot baseline")
	}
	if m.IPv6Mode != IPv6Block {
		return nil, nil, nil, fmt.Errorf("ipv6 mode %q is not implemented; v1 only supports %q (§8.3)", m.IPv6Mode, IPv6Block)
	}
	if m.Now.IsZero() {
		return nil, nil, nil, fmt.Errorf("model.now is required")
	}
	if err := validateDatapath(m.Datapath); err != nil {
		return nil, nil, nil, err
	}
	// Reject non-IPv4 set members before any tc or nft batch. An IPv6 literal
	// in an ipv4_addr set fails nft only after the qdisc has already changed.
	if err := validateIPv4Sets(m); err != nil {
		return nil, nil, nil, err
	}
	if err := validateProbe(m); err != nil {
		return nil, nil, nil, err
	}
	denied, err := m.denyV4()
	if err != nil {
		return nil, nil, nil, err
	}
	leases := map[uuid.UUID]MemberLease{}
	for _, l := range m.Leases {
		if _, ok := leases[l.MemberID]; ok {
			return nil, nil, nil, fmt.Errorf("duplicate lease for %s", l.MemberID)
		}
		if _, ok := ForwardingDeadline(l); !ok {
			return nil, nil, nil, fmt.Errorf("lease for %s is incomplete", l.MemberID)
		}
		leases[l.MemberID] = l
	}
	seenIP := map[netip.Addr]uuid.UUID{}
	seenDev := map[uuid.UUID]bool{}
	var devices []admittedDevice
	membersNeeded := map[uuid.UUID]bool{}
	for _, b := range m.Bindings {
		if b.MemberID == uuid.Nil || b.DeviceID == uuid.Nil {
			return nil, nil, nil, fmt.Errorf("binding is missing member_id or device_id")
		}
		if b.PeerID == "" || len(b.PeerID) > 128 || strings.ContainsAny(b.PeerID, "\r\n") {
			return nil, nil, nil, fmt.Errorf("peer_id for %s is empty or unsafe", b.DeviceID)
		}
		if seenDev[b.DeviceID] {
			return nil, nil, nil, fmt.Errorf("duplicate device %s", b.DeviceID)
		}
		seenDev[b.DeviceID] = true
		if len(b.AllowedV6) > 0 && m.IPv6Mode == IPv6Block {
			// Recorded on the binding for the agent, but v1 does not admit v6.
		}
		lease, ok := leases[b.MemberID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("device %s has no member lease", b.DeviceID)
		}
		var addrs []netip.Addr
		for _, p := range b.AllowedV4 {
			if !p.IsValid() || !p.Addr().Is4() || p.Bits() != 32 {
				return nil, nil, nil, fmt.Errorf("device %s allowed ipv4 %s is not a /32", b.DeviceID, p)
			}
			addr := p.Addr()
			if addr.IsUnspecified() || addr.IsMulticast() || addr.IsLoopback() {
				return nil, nil, nil, fmt.Errorf("device %s allowed address %s is not usable", b.DeviceID, addr)
			}
			if other, dup := seenIP[addr]; dup {
				return nil, nil, nil, fmt.Errorf("address %s is bound to both %s and %s", addr, other, b.DeviceID)
			}
			seenIP[addr] = b.DeviceID
			addrs = append(addrs, addr)
		}
		if len(addrs) == 0 {
			return nil, nil, nil, fmt.Errorf("device %s has no allowed ipv4", b.DeviceID)
		}
		sortAddrs(addrs)
		timeout, admit := elementTimeout(b, lease, m.Now)
		dev := admittedDevice{Binding: b, Lease: lease, Addrs: addrs, Timeout: timeout}
		if admit {
			devices = append(devices, dev)
			membersNeeded[b.MemberID] = true
		} else {
			// Expired or not-yet-valid devices are omitted from the allow set.
			// They still need a class only while admitted. Counters stay.
			devices = append(devices, dev)
		}
	}
	if err := validateQoS(m.QoS, membersNeeded); err != nil {
		return nil, nil, nil, err
	}
	var memberIDs []uuid.UUID
	for _, mem := range m.QoS.Members {
		memberIDs = append(memberIDs, mem.MemberID)
	}
	assignments, err := AssignClasses(memberIDs, m.QoS.Assignments)
	if err != nil {
		return nil, nil, nil, err
	}
	for i := range devices {
		if !devices[i].TimeoutOK() {
			continue
		}
		minor, ok := assignments[devices[i].Binding.MemberID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("admitted member %s has no qos class", devices[i].Binding.MemberID)
		}
		devices[i].Minor = minor
	}
	sortDevices(devices)
	return devices, assignments, denied, nil
}

func validateIPv4Sets(m Model) error {
	for _, addr := range m.Resolvers {
		if !addr.IsValid() || !addr.Is4() {
			return fmt.Errorf("resolver %s is not an ipv4 address", addr)
		}
	}
	for _, addr := range m.Revoke {
		if !addr.IsValid() || !addr.Is4() {
			return fmt.Errorf("revoke address %s is not an ipv4 address", addr)
		}
	}
	if err := validateV4Prefixes(m.DeniedV4, "denied_v4"); err != nil {
		return err
	}
	if err := validateV4Prefixes(m.ExtraDeniedV4, "extra_denied_v4"); err != nil {
		return err
	}
	return validateV4Prefixes(m.HostV4, "host_v4")
}

func validateV4Prefixes(ps []netip.Prefix, name string) error {
	for _, p := range ps {
		if !p.IsValid() || !p.Addr().Is4() {
			return fmt.Errorf("%s prefix %s is not an ipv4 prefix", name, p)
		}
	}
	return nil
}

func (d admittedDevice) TimeoutOK() bool { return d.Timeout > 0 }

// validateProbe checks the synthetic-probe exception (§13.9). A nil probe is
// the removal path and is valid. Validation runs before any tc or nft batch.
func validateProbe(m Model) error {
	p := m.Probe
	if p == nil {
		return nil
	}
	if p.PeerID == "" || len(p.PeerID) > 128 || strings.ContainsAny(p.PeerID, "\r\n") {
		return fmt.Errorf("probe peer_id is empty or unsafe")
	}
	if !p.SourceV4.IsValid() || !p.SourceV4.Addr().Is4() || p.SourceV4.Bits() != 32 {
		return fmt.Errorf("probe source %s is not an ipv4 /32", p.SourceV4)
	}
	src := p.SourceV4.Addr()
	if src.IsUnspecified() || src.IsMulticast() || src.IsLoopback() || src.IsLinkLocalUnicast() {
		return fmt.Errorf("probe source %s is not usable", src)
	}
	if nodeLocalAddr(m, src) {
		return fmt.Errorf("probe source %s is a node address", src)
	}
	for _, b := range m.Bindings {
		if b.PeerID != "" && b.PeerID == p.PeerID {
			return fmt.Errorf("probe peer %s is a member device", p.PeerID)
		}
		for _, allowed := range b.AllowedV4 {
			if allowed.IsValid() && allowed.Contains(src) {
				return fmt.Errorf("probe source %s collides with member binding %s", src, b.DeviceID)
			}
		}
	}
	if len(p.Destinations) == 0 && !p.ResolverDNS {
		return fmt.Errorf("probe has no destinations and resolver DNS is off")
	}
	if p.ResolverDNS {
		if len(m.Resolvers) == 0 {
			return fmt.Errorf("probe resolver DNS requires at least one ipv4 resolver")
		}
		for _, addr := range m.Resolvers {
			if !addr.IsValid() || !addr.Is4() {
				return fmt.Errorf("probe resolver %s is not an ipv4 address", addr)
			}
		}
	}
	denied, err := m.denyV4()
	if err != nil {
		return err
	}
	for _, d := range p.Destinations {
		if err := validateProbeDestination(m, d, denied); err != nil {
			return err
		}
	}
	if m.Now.IsZero() {
		return fmt.Errorf("model.now is required to evaluate the probe expiration")
	}
	remain := p.ExpiresAt.Sub(m.Now)
	if remain <= 0 {
		return fmt.Errorf("probe ExpiresAt is in the past")
	}
	if remain > time.Duration(ProbeMaxSeconds)*time.Second {
		return fmt.Errorf("probe ExpiresAt is more than %d seconds ahead", ProbeMaxSeconds)
	}
	if _, ok := probeTimeout(p.ExpiresAt, m.Now); !ok {
		return fmt.Errorf("probe ExpiresAt is less than one second ahead")
	}
	return nil
}

func validateProbeDestination(m Model, d ProbeDestination, denied []netip.Prefix) error {
	if !d.AddrV4.IsValid() || !d.AddrV4.Addr().Is4() || d.AddrV4.Bits() != 32 {
		return fmt.Errorf("probe destination %s is not an ipv4 /32", d.AddrV4)
	}
	if d.Proto != "tcp" && d.Proto != "udp" {
		return fmt.Errorf("probe destination proto %q is not tcp or udp", d.Proto)
	}
	if d.Port == 0 {
		return fmt.Errorf("probe destination port must be 1-65535")
	}
	addr := d.AddrV4.Addr()
	if addr.IsUnspecified() || addr.IsMulticast() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return fmt.Errorf("probe destination %s is inside host or denied material", addr)
	}
	local := nodeLocalAddr(m, addr)
	if local && probeAdminPort(d.Port) {
		return fmt.Errorf("probe destination %s port %d is a node admin port", addr, d.Port)
	}
	if local || addrInPrefixes(addr, denied) || addrInPrefixes(addr, m.HostV4) {
		return fmt.Errorf("probe destination %s is inside host or denied material", addr)
	}
	return nil
}

func nodeLocalAddr(m Model, addr netip.Addr) bool {
	if m.Datapath.SNAT.IsValid() && addr == m.Datapath.SNAT {
		return true
	}
	return addrInPrefixes(addr, m.HostV4)
}

func addrInPrefixes(addr netip.Addr, ps []netip.Prefix) bool {
	for _, p := range ps {
		if p.IsValid() && p.Contains(addr) {
			return true
		}
	}
	return false
}

func probeAdminPort(port uint16) bool {
	switch port {
	case 22, 80, 443, 3000, 8080, 9091:
		return true
	default:
		return false
	}
}

func validateQoS(q QoS, needed map[uuid.UUID]bool) error {
	if err := validateParent(q.Up, "qos.up"); err != nil {
		return err
	}
	if err := validateParent(q.Down, "qos.down"); err != nil {
		return err
	}
	if err := validateHostRate(q.HostUp, q.Up, "qos.host_up"); err != nil {
		return err
	}
	if err := validateHostRate(q.HostDown, q.Down, "qos.host_down"); err != nil {
		return err
	}
	seen := map[uuid.UUID]bool{}
	var sumUp, sumDown uint64
	sumUp += q.HostUp.RateBPS
	sumDown += q.HostDown.RateBPS
	for _, mem := range q.Members {
		if mem.MemberID == uuid.Nil {
			return fmt.Errorf("qos member id is nil")
		}
		if seen[mem.MemberID] {
			return fmt.Errorf("duplicate qos member %s", mem.MemberID)
		}
		seen[mem.MemberID] = true
		if err := validateClassRate(mem.Up, q.Up, "up "+mem.MemberID.String()); err != nil {
			return err
		}
		if err := validateClassRate(mem.Down, q.Down, "down "+mem.MemberID.String()); err != nil {
			return err
		}
		sumUp += mem.Up.RateBPS
		sumDown += mem.Down.RateBPS
	}
	if sumUp > q.Up.RateBPS {
		return fmt.Errorf("sum of upload guarantees %d exceeds parent %d (§11.4)", sumUp, q.Up.RateBPS)
	}
	if sumDown > q.Down.RateBPS {
		return fmt.Errorf("sum of download guarantees %d exceeds parent %d (§11.4)", sumDown, q.Down.RateBPS)
	}
	for id := range needed {
		if !seen[id] {
			return fmt.Errorf("admitted member %s has no qos class", id)
		}
	}
	return nil
}

func validateParent(r Rate, name string) error {
	if r.RateBPS == 0 || r.CeilBPS == 0 {
		return fmt.Errorf("%s rate and ceil must be non-zero", name)
	}
	if r.RateBPS > r.CeilBPS {
		return fmt.Errorf("%s rate %d exceeds ceil %d", name, r.RateBPS, r.CeilBPS)
	}
	return nil
}

func validateHostRate(host, parent Rate, name string) error {
	if host.RateBPS == 0 && host.CeilBPS == 0 {
		return fmt.Errorf("%s must set a host class so unclassified traffic is not member traffic (§11.4)", name)
	}
	return validateClassRate(host, parent, name)
}

func validateClassRate(child, parent Rate, name string) error {
	if child.RateBPS == 0 || child.CeilBPS == 0 {
		return fmt.Errorf("%s rate and ceil must be non-zero", name)
	}
	if child.RateBPS > child.CeilBPS {
		return fmt.Errorf("%s rate exceeds ceil", name)
	}
	if child.CeilBPS > parent.CeilBPS {
		return fmt.Errorf("%s ceil exceeds parent ceil", name)
	}
	return nil
}

func sortAddrs(a []netip.Addr) {
	for i := 1; i < len(a); i++ {
		j := i
		for j > 0 && a[j].Compare(a[j-1]) < 0 {
			a[j], a[j-1] = a[j-1], a[j]
			j--
		}
	}
}

func sortDevices(d []admittedDevice) {
	for i := 1; i < len(d); i++ {
		j := i
		for j > 0 && d[j].Binding.DeviceID.String() < d[j-1].Binding.DeviceID.String() {
			d[j], d[j-1] = d[j-1], d[j]
			j--
		}
	}
}
