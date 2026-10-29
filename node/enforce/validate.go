package enforce

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"

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

func (d admittedDevice) TimeoutOK() bool { return d.Timeout > 0 }

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
