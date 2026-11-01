package agent

// TEST ONLY reference specification for the proposed design in
// docs/node/followup33-intersection-serving-design-2026-11-01.md.
// These types are deliberately not api/enforce models. Nothing in production
// can call this file. Signature checking, normalization of signed times and
// kernel realization are prerequisites, not simulated verification here.

import (
	"fmt"
	"math"
	"net/netip"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

type f33Ceil struct{ Up, Down uint64 }

type f33Binding struct {
	Member, Device, Peer string
	Addresses            []netip.Prefix
	Revision             uint64
	From, Until          time.Duration
}

type f33Member struct {
	// Revision is the proposed signed per-member entitlement witness, NOT
	// MemberRatePolicy.Generation or DesiredState.BindingRevision.
	Revision                      uint64
	ServiceEnd, RateFrom, RateEnd time.Duration
	Ceil                          f33Ceil
}

type f33Policy struct {
	Node, Incarnation, Topology, Isolation, DNSPolicy, Meter string
	Generation, BindingSetRevision, AuthorizationRevision    uint64
	Verified, Complete, Published                            bool
	From, Until                                              time.Duration
	NodeCeil                                                 f33Ceil
	HostReserve                                              uint64
	Allowed, Denied                                          []netip.Prefix
	Resolvers                                                []netip.Addr
	Bindings                                                 []f33Binding
	Members                                                  map[string]f33Member
}

type f33Grant struct {
	Revision             uint64
	LeaseEnd, ServiceEnd time.Duration
}

type f33Lease struct {
	Node, Incarnation, ID             string
	Generation, AuthorizationRevision uint64
	Verified                          bool
	Anchor, NodeEnd, Offline          time.Duration
	Members                           map[string]f33Grant
}

type f33Installed struct {
	Generation uint64
	Verified   bool
	NodeCeil   f33Ceil
	Ceil       map[string]f33Ceil
	Bindings   []f33Binding
	Deadline   map[string]time.Duration
}

type f33Input struct {
	Old, New                                 f33Policy
	Lease                                    f33Lease
	Installed                                f33Installed
	Now                                      time.Duration
	SameBoot, SameProcess, Trusted, ReadyAtE bool
	AtomicPolicy, Frozen                     bool
}

type f33Admitted struct {
	Binding  f33Binding
	Ceil     f33Ceil
	Deadline time.Duration
}

type f33Subset struct {
	NodeCeil        f33Ceil
	Allowed, Denied []netip.Prefix
	Resolvers       []netip.Addr
	Bindings        []f33Admitted
}

// f33Result retains the mathematical candidate for ceiling assertions even
// when it cannot be realized at E. Baseline=true authorizes ZERO bindings.
type f33Result struct {
	Candidate f33Subset
	Baseline  bool
	Reason    string
}

func f33MinCeil(a, b f33Ceil) f33Ceil {
	return f33Ceil{min(a.Up, b.Up), min(a.Down, b.Down)}
}

func f33CeilOK(c f33Ceil) bool { return c.Up > 0 && c.Down > 0 }

func f33Prefixes(ps []netip.Prefix) ([]netip.Prefix, bool) {
	out := slices.Clone(ps)
	for i, p := range out {
		if !p.IsValid() || !p.Addr().Is4() {
			return nil, false
		}
		out[i] = p.Masked()
	}
	slices.SortFunc(out, func(a, b netip.Prefix) int {
		if c := a.Addr().Compare(b.Addr()); c != 0 {
			return c
		}
		return a.Bits() - b.Bits()
	})
	var reduced []netip.Prefix
	for _, p := range out {
		covered := false
		for _, q := range reduced {
			covered = covered || (q.Bits() <= p.Bits() && q.Contains(p.Addr()))
		}
		if !covered {
			reduced = append(reduced, p)
		}
	}
	return reduced, true
}

func f33IntersectPrefixes(a, b []netip.Prefix) []netip.Prefix {
	var out []netip.Prefix
	for _, p := range a {
		for _, q := range b {
			switch {
			case p.Bits() <= q.Bits() && p.Contains(q.Addr()):
				out = append(out, q)
			case q.Bits() <= p.Bits() && q.Contains(p.Addr()):
				out = append(out, p)
			}
		}
	}
	out, _ = f33Prefixes(out)
	return out
}

func f33ValidPolicy(p f33Policy) bool {
	if !p.Verified || !p.Complete || !p.Published || p.Node == "" || p.Incarnation == "" ||
		p.Topology == "" || p.Isolation == "" || p.DNSPolicy == "" || p.Meter == "" ||
		p.Generation == 0 || p.BindingSetRevision == 0 || p.AuthorizationRevision == 0 ||
		p.Until <= p.From || !f33CeilOK(p.NodeCeil) || p.HostReserve == 0 ||
		p.HostReserve >= min(p.NodeCeil.Up, p.NodeCeil.Down) || p.Members == nil {
		return false
	}
	for _, ps := range [][]netip.Prefix{p.Allowed, p.Denied} {
		if _, ok := f33Prefixes(ps); !ok {
			return false
		}
	}
	for _, r := range p.Resolvers {
		if !r.IsValid() || !r.Is4() || r.IsUnspecified() {
			return false
		}
	}
	devices, peers, addresses := map[string]bool{}, map[string]bool{}, map[netip.Addr]bool{}
	for _, b := range p.Bindings {
		if b.Member == "" || b.Device == "" || b.Peer == "" || b.Revision == 0 ||
			b.Until <= b.From || len(b.Addresses) == 0 || devices[b.Device] || peers[b.Peer] {
			return false
		}
		devices[b.Device], peers[b.Peer] = true, true
		for _, a := range b.Addresses {
			if !a.IsValid() || !a.Addr().Is4() || a.Bits() != 32 || a.Addr().IsUnspecified() || addresses[a.Addr()] {
				return false
			}
			addresses[a.Addr()] = true
		}
	}
	for id, m := range p.Members {
		if id == "" || m.Revision == 0 || m.ServiceEnd <= 0 ||
			m.RateEnd <= m.RateFrom || !f33CeilOK(m.Ceil) {
			return false
		}
	}
	return true
}

func f33SameIdentity(a, b f33Binding) bool {
	aa, _ := f33Prefixes(a.Addresses)
	bb, _ := f33Prefixes(b.Addresses)
	return a.Member == b.Member && a.Device == b.Device && a.Peer == b.Peer &&
		a.Revision == b.Revision && slices.Equal(aa, bb)
}

func f33Find(bs []f33Binding, device string) (f33Binding, bool) {
	for _, b := range bs {
		if b.Device == device {
			return b, true
		}
	}
	return f33Binding{}, false
}

func f33SameBindingSet(a, b []f33Binding) bool {
	if len(a) != len(b) {
		return false
	}
	for _, old := range a {
		other, ok := f33Find(b, old.Device)
		if !ok || !f33SameIdentity(old, other) || old.From != other.From || old.Until != other.Until {
			return false
		}
	}
	return true
}

// All signed time fields in this reference are checked, exact durations from
// the OLD lease's signed issued_at, NOT durations from receipt of New. They
// represent timestamps after parsing and overflow rejection by the adapter.
// Installed.Deadline is an absolute same-boot monotonic deadline.
func f33Intersection(in f33Input) f33Result {
	base := func(why string) f33Result { return f33Result{Baseline: true, Reason: why} }
	o, n, l, a := in.Old, in.New, in.Lease, in.Installed
	if in.Frozen || !in.SameBoot || !in.SameProcess || !in.Trusted || !in.ReadyAtE || !in.AtomicPolicy {
		return base("gate")
	}
	if !f33ValidPolicy(o) || !f33ValidPolicy(n) || !l.Verified || !a.Verified || l.ID == "" ||
		l.Anchor <= 0 || in.Now < l.Anchor || l.NodeEnd <= 0 || l.Offline <= 0 {
		return base("provenance")
	}
	if o.Node != n.Node || o.Incarnation != n.Incarnation || l.Node != o.Node || l.Incarnation != o.Incarnation ||
		n.Generation <= o.Generation || l.Generation != o.Generation || a.Generation != o.Generation ||
		l.AuthorizationRevision != o.AuthorizationRevision || n.AuthorizationRevision < o.AuthorizationRevision ||
		n.BindingSetRevision < o.BindingSetRevision {
		return base("identity_or_revision")
	}
	if o.Topology != n.Topology || o.Isolation != n.Isolation || o.DNSPolicy != n.DNSPolicy ||
		o.Meter != n.Meter || o.HostReserve != n.HostReserve {
		return base("global_policy")
	}
	if n.BindingSetRevision == o.BindingSetRevision && !f33SameBindingSet(o.Bindings, n.Bindings) {
		return base("ambiguous_binding_set")
	}
	if !f33CeilOK(a.NodeCeil) || a.NodeCeil.Up > o.NodeCeil.Up || a.NodeCeil.Down > o.NodeCeil.Down {
		return base("old_shape")
	}
	elapsed := in.Now - l.Anchor
	if elapsed < max(o.From, n.From) || elapsed >= min(o.Until, n.Until, l.NodeEnd, l.Offline) {
		return base("global_deadline")
	}
	// Read-back must describe the exact old identities, not just their member
	// IDs. Duplicate/corrupt applied identities invalidate the whole premise.
	ap := o
	ap.Bindings = a.Bindings
	if !f33ValidPolicy(ap) {
		return base("old_bindings")
	}
	for _, b := range a.Bindings {
		old, ok := f33Find(o.Bindings, b.Device)
		if !ok || !f33SameIdentity(old, b) {
			return base("old_bindings")
		}
	}
	allowedOld, _ := f33Prefixes(o.Allowed)
	allowedNew, _ := f33Prefixes(n.Allowed)
	denied, _ := f33Prefixes(append(slices.Clone(o.Denied), n.Denied...))
	c := f33Subset{NodeCeil: f33MinCeil(a.NodeCeil, f33MinCeil(o.NodeCeil, n.NodeCeil)),
		Allowed: f33IntersectPrefixes(allowedOld, allowedNew), Denied: denied}
	for _, r := range o.Resolvers {
		if slices.Contains(n.Resolvers, r) && !slices.Contains(c.Resolvers, r) {
			c.Resolvers = append(c.Resolvers, r)
		}
	}
	slices.SortFunc(c.Resolvers, netip.Addr.Compare)
	if len(c.Allowed) == 0 || len(c.Resolvers) == 0 {
		return base("empty_global_intersection")
	}
	for _, old := range o.Bindings {
		newBinding, present := f33Find(n.Bindings, old.Device)
		installed, installedOK := f33Find(a.Bindings, old.Device)
		om, omOK := o.Members[old.Member]
		nm, nmOK := n.Members[old.Member]
		grant, grantOK := l.Members[old.Member]
		if !present || !installedOK || !f33SameIdentity(old, newBinding) || !f33SameIdentity(old, installed) ||
			!omOK || !nmOK || !grantOK || grant.Revision == 0 ||
			om.Revision != grant.Revision || nm.Revision != grant.Revision {
			continue
		}
		if grant.LeaseEnd <= 0 || grant.ServiceEnd <= 0 {
			continue
		}
		if elapsed < max(old.From, newBinding.From, om.RateFrom, nm.RateFrom) {
			continue
		}
		window := min(l.NodeEnd, l.Offline, o.Until, n.Until, grant.LeaseEnd, grant.ServiceEnd,
			om.ServiceEnd, nm.ServiceEnd, old.Until, newBinding.Until, om.RateEnd, nm.RateEnd)
		if window <= 0 || window > time.Duration(math.MaxInt64)-l.Anchor {
			return base("deadline_overflow")
		}
		deadline := min(l.Anchor+window, a.Deadline[old.Device])
		// Reserve is deducted once for realization; the signed D is retained.
		if deadline <= in.Now || deadline-in.Now <= enforce.DeadlineInstallBound ||
			(deadline-in.Now-enforce.DeadlineInstallBound)/enforce.DeadlineGranularity == 0 {
			continue
		}
		actual, ok := a.Ceil[old.Member]
		if !ok || !f33CeilOK(actual) || actual.Up > min(om.Ceil.Up, o.NodeCeil.Up) || actual.Down > min(om.Ceil.Down, o.NodeCeil.Down) {
			return base("old_shape")
		}
		ceil := f33MinCeil(actual, f33MinCeil(om.Ceil, f33MinCeil(nm.Ceil, c.NodeCeil)))
		binding := old
		binding.Addresses, _ = f33Prefixes(old.Addresses)
		binding.From, binding.Until = max(old.From, newBinding.From), min(old.Until, newBinding.Until)
		c.Bindings = append(c.Bindings, f33Admitted{binding, ceil, deadline})
	}
	// The current kernel model shares a timeout per member. Preserve the
	// earliest device limit; a later full Apply may change this, S cannot.
	memberDeadline := map[string]time.Duration{}
	for _, b := range c.Bindings {
		d, exists := memberDeadline[b.Binding.Member]
		if !exists || b.Deadline < d {
			memberDeadline[b.Binding.Member] = b.Deadline
		}
	}
	for i := range c.Bindings {
		c.Bindings[i].Deadline = memberDeadline[c.Bindings[i].Binding.Member]
	}
	slices.SortFunc(c.Bindings, func(a, b f33Admitted) int {
		if a.Binding.Device < b.Binding.Device {
			return -1
		}
		if a.Binding.Device > b.Binding.Device {
			return 1
		}
		return 0
	})
	out := f33Result{Candidate: c}
	if len(c.Bindings) == 0 {
		out.Baseline, out.Reason = true, "empty_subset"
		return out
	}
	// The initial implementation profile performs no tc mutations at E.
	// A tighter candidate is mathematically safe but cannot be served until
	// it is enforced; using the old larger ceil even briefly would be unsafe.
	if c.NodeCeil != a.NodeCeil {
		out.Baseline, out.Reason = true, "requires_tc"
	}
	for _, b := range c.Bindings {
		if b.Ceil != a.Ceil[b.Binding.Member] {
			out.Baseline, out.Reason = true, "requires_tc"
		}
	}
	return out
}

func f33Fixture() f33Input {
	// All IDs, rates and relative times below are TEST ONLY abstract values.
	p := f33Policy{
		Node: "node", Incarnation: "incarnation", Topology: "path", Isolation: "ipv4-only", DNSPolicy: "dns-content", Meter: "epoch",
		Generation: 4, BindingSetRevision: 7, AuthorizationRevision: 3, Verified: true, Complete: true, Published: true,
		Until: 100 * time.Second, NodeCeil: f33Ceil{1000, 2000}, HostReserve: 10,
		Allowed:   []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.51.100.0/24")},
		Denied:    []netip.Prefix{netip.MustParsePrefix("203.0.113.0/25")},
		Resolvers: []netip.Addr{netip.MustParseAddr("192.0.2.53"), netip.MustParseAddr("198.51.100.53")},
		Members:   map[string]f33Member{},
	}
	l := f33Lease{Node: p.Node, Incarnation: p.Incarnation, ID: "old-lease", Generation: p.Generation,
		AuthorizationRevision: p.AuthorizationRevision, Verified: true, Anchor: time.Second,
		NodeEnd: 100 * time.Second, Offline: 100 * time.Second, Members: map[string]f33Grant{}}
	a := f33Installed{Generation: p.Generation, Verified: true, NodeCeil: p.NodeCeil,
		Ceil: map[string]f33Ceil{}, Deadline: map[string]time.Duration{}}
	for i, member := range []string{"a", "b"} {
		b := f33Binding{Member: member, Device: member, Peer: "peer-" + member, Revision: 9,
			Addresses: []netip.Prefix{netip.MustParsePrefix(fmt.Sprintf("100.64.0.%d/32", i+1))}, Until: 100 * time.Second}
		p.Bindings = append(p.Bindings, b)
		p.Members[member] = f33Member{Revision: 11, ServiceEnd: 100 * time.Second, RateEnd: 100 * time.Second, Ceil: f33Ceil{100, 200}}
		l.Members[member] = f33Grant{Revision: 11, LeaseEnd: 100 * time.Second, ServiceEnd: 100 * time.Second}
		a.Ceil[member], a.Deadline[member] = f33Ceil{100, 200}, 101*time.Second
	}
	a.Bindings = slices.Clone(p.Bindings)
	n := p
	n.Generation, n.BindingSetRevision = 5, 8
	n.Bindings = slices.Clone(p.Bindings)
	n.Members = map[string]f33Member{"a": p.Members["a"], "b": p.Members["b"]}
	return f33Input{Old: p, New: n, Lease: l, Installed: a, Now: 2 * time.Second,
		SameBoot: true, SameProcess: true, Trusted: true, ReadyAtE: true, AtomicPolicy: true}
}

func f33Devices(r f33Result) string {
	if r.Baseline {
		return "baseline"
	}
	var ids string
	for _, b := range r.Candidate.Bindings {
		ids += b.Binding.Device
	}
	return ids
}

// This table is exhaustive over the identity/provenance equivalence classes
// of the reference. Numeric/order domains are enumerated separately below.
func TestF33_ISO02_IntersectionIdentityTable(t *testing.T) {
	tests := []struct {
		name string
		edit func(*f33Input)
		want string
	}{
		{"unchanged", func(*f33Input) {}, "ab"},
		{"remove_device", func(x *f33Input) { x.New.Bindings = x.New.Bindings[1:] }, "b"},
		{"remove_all", func(x *f33Input) { x.New.Bindings = nil }, "baseline"},
		{"node_change", func(x *f33Input) { x.New.Node = "different" }, "baseline"},
		{"incarnation_change", func(x *f33Input) { x.New.Incarnation = "different" }, "baseline"},
		{"lease_node", func(x *f33Input) { x.Lease.Node = "different" }, "baseline"},
		{"lease_incarnation", func(x *f33Input) { x.Lease.Incarnation = "different" }, "baseline"},
		{"lease_generation", func(x *f33Input) { x.Lease.Generation++ }, "baseline"},
		{"lease_authorization", func(x *f33Input) { x.Lease.AuthorizationRevision++ }, "baseline"},
		{"applied_generation", func(x *f33Input) { x.Installed.Generation++ }, "baseline"},
		{"equal_generation", func(x *f33Input) { x.New.Generation = x.Old.Generation }, "baseline"},
		{"generation_rollback", func(x *f33Input) { x.New.Generation = 3 }, "baseline"},
		{"zero_generation", func(x *f33Input) { x.New.Generation = 0 }, "baseline"},
		{"authorization_rollback", func(x *f33Input) { x.New.AuthorizationRevision-- }, "baseline"},
		{"authorization_advance_with_witness", func(x *f33Input) { x.New.AuthorizationRevision++ }, "ab"},
		{"binding_set_rollback", func(x *f33Input) { x.New.BindingSetRevision = 6 }, "baseline"},
		{"same_set_revision_changed_content", func(x *f33Input) { x.New.BindingSetRevision = 7; x.New.Bindings = x.New.Bindings[1:] }, "baseline"},
		{"same_set_revision_same_content", func(x *f33Input) { x.New.BindingSetRevision = 7 }, "ab"},
		{"same_set_revision_reordered", func(x *f33Input) { x.New.BindingSetRevision = 7; slices.Reverse(x.New.Bindings) }, "ab"},
		{"member_reuse", func(x *f33Input) { x.New.Bindings[0].Member = "b" }, "b"},
		{"device_reuse", func(x *f33Input) { x.New.Bindings[0].Device = "c" }, "b"},
		{"peer_reuse", func(x *f33Input) { x.New.Bindings[0].Peer = "new-peer" }, "b"},
		{"address_reuse", func(x *f33Input) {
			x.New.Bindings[0].Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.0.3/32")}
		}, "b"},
		{"address_added_to_device", func(x *f33Input) {
			x.New.Bindings[0].Addresses = append(slices.Clone(x.New.Bindings[0].Addresses), netip.MustParsePrefix("100.64.0.3/32"))
		}, "b"},
		{"binding_revision_advance", func(x *f33Input) { x.New.Bindings[0].Revision++ }, "b"},
		{"binding_revision_rollback", func(x *f33Input) { x.New.Bindings[0].Revision-- }, "b"},
		{"binding_revision_unknown", func(x *f33Input) { x.New.Bindings[0].Revision = 0 }, "baseline"},
		{"new_entitlement", func(x *f33Input) { m := x.New.Members["a"]; m.Revision++; x.New.Members["a"] = m }, "b"},
		{"older_entitlement", func(x *f33Input) { m := x.New.Members["a"]; m.Revision--; x.New.Members["a"] = m }, "b"},
		{"lease_entitlement_mismatch", func(x *f33Input) { m := x.Lease.Members["a"]; m.Revision++; x.Lease.Members["a"] = m }, "b"},
		{"missing_old_member_witness", func(x *f33Input) { delete(x.Old.Members, "a") }, "b"},
		{"missing_new_member_witness", func(x *f33Input) { delete(x.New.Members, "a") }, "b"},
		{"missing_new_witness_document", func(x *f33Input) { x.New.Members = nil }, "baseline"},
		{"missing_old_grant", func(x *f33Input) { delete(x.Lease.Members, "a") }, "b"},
		{"empty_old_lease", func(x *f33Input) { x.Lease.Members = nil }, "baseline"},
		{"not_previously_admitted", func(x *f33Input) { x.Installed.Bindings = x.Installed.Bindings[1:] }, "b"},
		{"applied_identity_disagrees", func(x *f33Input) { x.Installed.Bindings[0].Peer = "other" }, "baseline"},
		{"new_device_same_member", func(x *f33Input) {
			b := x.New.Bindings[0]
			b.Device, b.Peer = "c", "peer-c"
			b.Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.0.3/32")}
			x.New.Bindings = append(x.New.Bindings, b)
		}, "ab"},
		{"new_member", func(x *f33Input) {
			b := x.New.Bindings[0]
			b.Member, b.Device, b.Peer = "c", "c", "peer-c"
			b.Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.0.3/32")}
			x.New.Members["c"] = x.New.Members["a"]
			x.New.Bindings = append(x.New.Bindings, b)
		}, "ab"},
		{"duplicate_device", func(x *f33Input) { x.New.Bindings = append(x.New.Bindings, x.New.Bindings[0]) }, "baseline"},
		{"duplicate_peer", func(x *f33Input) { x.New.Bindings[0].Peer = x.New.Bindings[1].Peer }, "baseline"},
		{"duplicate_address", func(x *f33Input) { x.New.Bindings[0].Addresses = x.New.Bindings[1].Addresses }, "baseline"},
		{"missing_member_identity", func(x *f33Input) { x.New.Bindings[0].Member = "" }, "baseline"},
		{"missing_device_identity", func(x *f33Input) { x.New.Bindings[0].Device = "" }, "baseline"},
		{"missing_peer_identity", func(x *f33Input) { x.New.Bindings[0].Peer = "" }, "baseline"},
		{"non_host_source", func(x *f33Input) {
			x.New.Bindings[0].Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.0.0/24")}
		}, "baseline"},
		{"ipv6_source", func(x *f33Input) {
			x.New.Bindings[0].Addresses = []netip.Prefix{netip.MustParsePrefix("2001:db8::1/128")}
		}, "baseline"},
		{"empty_source", func(x *f33Input) { x.New.Bindings[0].Addresses = nil }, "baseline"},
		{"unknown_policy_semantics", func(x *f33Input) { x.New.Complete = false }, "baseline"},
		{"unverified_old", func(x *f33Input) { x.Old.Verified = false }, "baseline"},
		{"unverified_new", func(x *f33Input) { x.New.Verified = false }, "baseline"},
		{"unverified_lease", func(x *f33Input) { x.Lease.Verified = false }, "baseline"},
		{"unverified_installed", func(x *f33Input) { x.Installed.Verified = false }, "baseline"},
		{"unpublished", func(x *f33Input) { x.New.Published = false }, "baseline"},
		{"freeze", func(x *f33Input) { x.Frozen = true }, "baseline"},
		{"reboot", func(x *f33Input) { x.SameBoot = false }, "baseline"},
		{"same_boot_restart", func(x *f33Input) { x.SameProcess = false }, "baseline"},
		{"clock_anomaly", func(x *f33Input) { x.Trusted = false }, "baseline"},
		{"not_ready_at_E", func(x *f33Input) { x.ReadyAtE = false }, "baseline"},
		{"cannot_atomically_restrict", func(x *f33Input) { x.AtomicPolicy = false }, "baseline"},
		{"topology_change", func(x *f33Input) { x.New.Topology = "other" }, "baseline"},
		{"isolation_change", func(x *f33Input) { x.New.Isolation = "other" }, "baseline"},
		{"dns_document_change", func(x *f33Input) { x.New.DNSPolicy = "other" }, "baseline"},
		{"meter_change", func(x *f33Input) { x.New.Meter = "other" }, "baseline"},
		{"reserve_change", func(x *f33Input) { x.New.HostReserve++ }, "baseline"},
		{"no_resolver_intersection", func(x *f33Input) { x.New.Resolvers = []netip.Addr{netip.MustParseAddr("203.0.113.53")} }, "baseline"},
		{"no_destination_intersection", func(x *f33Input) { x.New.Allowed = []netip.Prefix{netip.MustParsePrefix("203.0.113.0/24")} }, "baseline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := f33Fixture()
			tt.edit(&in)
			before := fmt.Sprintf("%#v", in)
			got := f33Intersection(in)
			if f33Devices(got) != tt.want {
				t.Fatalf("got %s (%s), want %s", f33Devices(got), got.Reason, tt.want)
			}
			if fmt.Sprintf("%#v", in) != before {
				t.Fatal("reference mutated authorization input")
			}
		})
	}
}

func TestF33_FUP02_FUP04_IntersectionCeilings(t *testing.T) {
	// All 3^4 combinations of tighter/equal/looser policy ceilings. Testing
	// directions independently detects the r18 L-2 upload-only mistake.
	for _, mu := range []uint64{99, 100, 101} {
		for _, md := range []uint64{199, 200, 201} {
			for _, nu := range []uint64{999, 1000, 1001} {
				for _, nd := range []uint64{1999, 2000, 2001} {
					t.Run(fmt.Sprintf("%d_%d_%d_%d", mu, md, nu, nd), func(t *testing.T) {
						x := f33Fixture()
						m := x.New.Members["a"]
						m.Ceil = f33Ceil{mu, md}
						x.New.Members["a"], x.New.NodeCeil = m, f33Ceil{nu, nd}
						r := f33Intersection(x)
						if len(r.Candidate.Bindings) != 2 || r.Candidate.Bindings[0].Ceil != (f33Ceil{min(mu, 100), min(md, 200)}) ||
							r.Candidate.NodeCeil != (f33Ceil{min(nu, 1000), min(nd, 2000)}) {
							t.Fatalf("incorrect stricter ceiling: %+v", r)
						}
						if r.Baseline != (mu < 100 || md < 200 || nu < 1000 || nd < 2000) {
							t.Fatalf("must baseline when tc must change: %+v", r)
						}
					})
				}
			}
		}
	}
	for _, target := range []string{"old_node", "new_node", "old_member", "new_member", "installed_node", "installed_member"} {
		for _, direction := range []string{"up", "down"} {
			t.Run("zero_"+target+"_"+direction, func(t *testing.T) {
				x := f33Fixture()
				zero := func(c f33Ceil) f33Ceil {
					if direction == "up" {
						c.Up = 0
					} else {
						c.Down = 0
					}
					return c
				}
				switch target {
				case "old_node":
					x.Old.NodeCeil = zero(x.Old.NodeCeil)
				case "new_node":
					x.New.NodeCeil = zero(x.New.NodeCeil)
				case "installed_node":
					x.Installed.NodeCeil = zero(x.Installed.NodeCeil)
				case "installed_member":
					x.Installed.Ceil["a"] = zero(x.Installed.Ceil["a"])
				case "old_member":
					m := x.Old.Members["a"]
					m.Ceil = zero(m.Ceil)
					x.Old.Members["a"] = m
				case "new_member":
					m := x.New.Members["a"]
					m.Ceil = zero(m.Ceil)
					x.New.Members["a"] = m
				}
				if !f33Intersection(x).Baseline {
					t.Fatal("zero ceiling admitted")
				}
			})
		}
	}
	x := f33Fixture()
	x.Installed.Ceil["a"] = f33Ceil{50, 60}
	m := x.New.Members["a"]
	m.Ceil = f33Ceil{80, 90}
	x.New.Members["a"] = m
	r := f33Intersection(x)
	if r.Baseline || r.Candidate.Bindings[0].Ceil != (f33Ceil{50, 60}) {
		t.Fatal("lost stricter installed ceiling")
	}
	x = f33Fixture()
	x.New.NodeCeil = f33Ceil{80, 150}
	r = f33Intersection(x)
	if !r.Baseline || len(r.Candidate.Bindings) != 2 || r.Candidate.Bindings[0].Ceil != (f33Ceil{80, 150}) || r.Candidate.Bindings[1].Ceil != (f33Ceil{80, 150}) {
		t.Fatal("node ceiling did not cap every member in both directions")
	}
}

func TestF33_ENT02_IntersectionDeadlines(t *testing.T) {
	limits := []struct {
		name string
		set  func(*f33Input, time.Duration)
	}{
		{"old_node_lease", func(x *f33Input, v time.Duration) { x.Lease.NodeEnd = v }},
		{"old_offline", func(x *f33Input, v time.Duration) { x.Lease.Offline = v }},
		{"old_desired", func(x *f33Input, v time.Duration) { x.Old.Until = v }},
		{"new_desired", func(x *f33Input, v time.Duration) { x.New.Until = v }},
		{"old_member_lease", func(x *f33Input, v time.Duration) {
			m := x.Lease.Members["a"]
			m.LeaseEnd = v
			x.Lease.Members["a"] = m
		}},
		{"old_lease_service", func(x *f33Input, v time.Duration) {
			m := x.Lease.Members["a"]
			m.ServiceEnd = v
			x.Lease.Members["a"] = m
		}},
		{"old_desired_service", func(x *f33Input, v time.Duration) { m := x.Old.Members["a"]; m.ServiceEnd = v; x.Old.Members["a"] = m }},
		{"new_desired_service", func(x *f33Input, v time.Duration) { m := x.New.Members["a"]; m.ServiceEnd = v; x.New.Members["a"] = m }},
		{"old_binding", func(x *f33Input, v time.Duration) { x.Old.Bindings[0].Until = v }},
		{"new_binding", func(x *f33Input, v time.Duration) { x.New.Bindings[0].Until = v }},
		{"old_rate", func(x *f33Input, v time.Duration) { m := x.Old.Members["a"]; m.RateEnd = v; x.Old.Members["a"] = m }},
		{"new_rate", func(x *f33Input, v time.Duration) { m := x.New.Members["a"]; m.RateEnd = v; x.New.Members["a"] = m }},
		{"retained_D", func(x *f33Input, v time.Duration) { x.Installed.Deadline["a"] = x.Lease.Anchor + v }},
	}
	for _, limit := range limits {
		for _, v := range []time.Duration{-time.Second, 0, time.Second, 1100 * time.Millisecond, 1100*time.Millisecond + time.Nanosecond, 1101 * time.Millisecond, 20 * time.Second, 100 * time.Second, 200 * time.Second} {
			t.Run(fmt.Sprintf("%s_%s", limit.name, v), func(t *testing.T) {
				x := f33Fixture()
				limit.set(&x, v)
				r := f33Intersection(x)
				admitted := false
				if !r.Baseline {
					for _, b := range r.Candidate.Bindings {
						if b.Binding.Device != "a" {
							continue
						}
						admitted = true
						if want := time.Second + min(v, 100*time.Second); b.Deadline != want {
							t.Fatalf("deadline %s want %s", b.Deadline, want)
						}
					}
				}
				if admitted != (v >= 1101*time.Millisecond) {
					t.Fatalf("admitted=%t result=%+v", admitted, r)
				}
			})
		}
	}
	for _, which := range []string{"old_desired", "new_desired", "old_binding", "new_binding", "old_rate", "new_rate"} {
		t.Run("not_yet_valid_"+which, func(t *testing.T) {
			x := f33Fixture()
			switch which {
			case "old_desired":
				x.Old.From = 3 * time.Second
			case "new_desired":
				x.New.From = 3 * time.Second
			case "old_binding":
				x.Old.Bindings[0].From = 3 * time.Second
			case "new_binding":
				x.New.Bindings[0].From = 3 * time.Second
			case "old_rate":
				m := x.Old.Members["a"]
				m.RateFrom = 3 * time.Second
				x.Old.Members["a"] = m
			case "new_rate":
				m := x.New.Members["a"]
				m.RateFrom = 3 * time.Second
				x.New.Members["a"] = m
			}
			r := f33Intersection(x)
			if f33Devices(r) == "ab" {
				t.Fatal("future validity admitted")
			}
		})
	}
	for _, now := range []time.Duration{0, time.Second, 90 * time.Second, 101 * time.Second} {
		x := f33Fixture()
		x.Now = now
		r := f33Intersection(x)
		if now < x.Lease.Anchor || now >= 101*time.Second {
			if !r.Baseline {
				t.Fatal("invalid monotonic time admitted")
			}
		} else if r.Baseline || r.Candidate.Bindings[0].Deadline != 101*time.Second {
			t.Fatal("receipt or replay reanchored deadline")
		}
	}
	x := f33Fixture()
	x.Lease.Anchor = time.Duration(math.MaxInt64) - time.Second
	x.Now = x.Lease.Anchor
	if !f33Intersection(x).Baseline {
		t.Fatal("overflow admitted")
	}
	if enforce.DeadlineInstallBound != 100*time.Millisecond || enforce.DeadlineGranularity != time.Millisecond {
		t.Fatal("reference assumes unchanged D-134 reserve/granularity")
	}
}

func TestF33_ISO01_ISO04_IntersectionSets(t *testing.T) {
	x := f33Fixture()
	x.New.Allowed = []netip.Prefix{netip.MustParsePrefix("192.0.2.128/25"), netip.MustParsePrefix("203.0.113.0/24")}
	x.New.Denied = []netip.Prefix{netip.MustParsePrefix("192.0.2.192/26")}
	x.New.Resolvers = []netip.Addr{netip.MustParseAddr("192.0.2.53"), netip.MustParseAddr("203.0.113.53")}
	r := f33Intersection(x)
	if r.Baseline || !slices.Equal(r.Candidate.Allowed, []netip.Prefix{netip.MustParsePrefix("192.0.2.128/25")}) ||
		!slices.Equal(r.Candidate.Denied, []netip.Prefix{netip.MustParsePrefix("192.0.2.192/26"), netip.MustParsePrefix("203.0.113.0/25")}) ||
		!slices.Equal(r.Candidate.Resolvers, []netip.Addr{netip.MustParseAddr("192.0.2.53")}) {
		t.Fatalf("incorrect set algebra: %+v", r)
	}
	// Exhaust all 256 addresses in each documentation /24. Denial always wins
	// for ordinary forwarding; DNS is a separately scoped exact service rule.
	allows := func(ps []netip.Prefix, a netip.Addr) bool {
		for _, p := range ps {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	for _, network := range []string{"192.0.2.0", "198.51.100.0", "203.0.113.0"} {
		a := netip.MustParseAddr(network)
		for i := 0; i < 256; i++ {
			got := allows(r.Candidate.Allowed, a) && !allows(r.Candidate.Denied, a)
			want := allows(x.Old.Allowed, a) && allows(x.New.Allowed, a) && !allows(x.Old.Denied, a) && !allows(x.New.Denied, a)
			if got != want {
				t.Fatalf("incorrect destination meet at %s", a)
			}
			a = a.Next()
		}
	}
	for _, mask := range []int{0, 24, 25, 32} {
		for _, other := range []int{0, 24, 25, 32} {
			p := netip.PrefixFrom(netip.MustParseAddr("192.0.2.128"), mask).Masked()
			q := netip.PrefixFrom(netip.MustParseAddr("192.0.2.128"), other).Masked()
			got := f33IntersectPrefixes([]netip.Prefix{p}, []netip.Prefix{q})
			want := netip.PrefixFrom(netip.MustParseAddr("192.0.2.128"), max(mask, other)).Masked()
			if !slices.Equal(got, []netip.Prefix{want}) {
				t.Fatal("prefix intersection used string equality")
			}
		}
	}
}

func TestF33_ENT01_IntersectionIdentityConjunction(t *testing.T) {
	// Exhaust all 256 combinations, so one matching field cannot compensate
	// for a mismatched identity/revision or an absent old authorization source.
	for mask := 0; mask < 256; mask++ {
		t.Run(fmt.Sprintf("mask_%08b", mask), func(t *testing.T) {
			x := f33Fixture()
			if mask&1 != 0 {
				x.New.Bindings[0].Member = "c"
			}
			if mask&2 != 0 {
				x.New.Bindings[0].Device = "c"
			}
			if mask&4 != 0 {
				x.New.Bindings[0].Peer = "peer-c"
			}
			if mask&8 != 0 {
				x.New.Bindings[0].Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.0.3/32")}
			}
			if mask&16 != 0 {
				x.New.Bindings[0].Revision++
			}
			if mask&32 != 0 {
				m := x.New.Members["a"]
				m.Revision++
				x.New.Members["a"] = m
			}
			if mask&64 != 0 {
				delete(x.Lease.Members, "a")
			}
			if mask&128 != 0 {
				x.Installed.Bindings = x.Installed.Bindings[1:]
			}
			want := "b"
			if mask == 0 {
				want = "ab"
			}
			if r := f33Intersection(x); f33Devices(r) != want {
				t.Fatalf("got %s (%s), want %s", f33Devices(r), r.Reason, want)
			}
		})
	}
}

func TestF33_ENT02_IntersectionMultipleDevicesAndOrder(t *testing.T) {
	x := f33Fixture()
	// Two previously admitted devices of a share its class and earliest D.
	c := x.Old.Bindings[0]
	c.Device, c.Peer = "c", "peer-c"
	c.Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.0.3/32")}
	x.Old.Bindings = append(x.Old.Bindings, c)
	x.New.Bindings = append(x.New.Bindings, c)
	x.Installed.Bindings = append(x.Installed.Bindings, c)
	x.Installed.Deadline["c"] = 101 * time.Second
	x.New.Bindings[2].Until = 20 * time.Second
	r := f33Intersection(x)
	if f33Devices(r) != "abc" {
		t.Fatalf("multi-device subset: %+v", r)
	}
	for _, b := range r.Candidate.Bindings {
		if b.Binding.Member == "a" && (b.Deadline != 21*time.Second || b.Ceil != (f33Ceil{100, 200})) {
			t.Fatal("member timeout/class not shared conservatively")
		}
	}
	slices.Reverse(x.Old.Bindings)
	slices.Reverse(x.New.Bindings)
	slices.Reverse(x.Installed.Bindings)
	if next := f33Intersection(x); !reflect.DeepEqual(next, r) {
		t.Fatal("iteration order changed subset")
	}

	// Address-set order is immaterial; removing any address excludes the
	// entire changed device, even if another old address remains.
	x = f33Fixture()
	addresses := []netip.Prefix{netip.MustParsePrefix("100.64.0.1/32"), netip.MustParsePrefix("100.64.0.3/32")}
	x.Old.Bindings[0].Addresses, x.Installed.Bindings[0].Addresses = addresses, addresses
	x.New.Bindings[0].Addresses = []netip.Prefix{addresses[1], addresses[0]}
	if f33Devices(f33Intersection(x)) != "ab" {
		t.Fatal("address ordering changed identity")
	}
	x.New.Bindings[0].Addresses = x.New.Bindings[0].Addresses[:1]
	if f33Devices(f33Intersection(x)) != "b" {
		t.Fatal("changed address set was admitted")
	}
}

func TestF33_ISO02_IntersectionAmbiguityAndBounds(t *testing.T) {
	tests := []struct {
		name string
		edit func(*f33Input)
	}{
		{"missing_lease_id", func(x *f33Input) { x.Lease.ID = "" }},
		{"missing_anchor", func(x *f33Input) { x.Lease.Anchor = 0 }},
		{"missing_old_witness_document", func(x *f33Input) { x.Old.Members = nil }},
		{"empty_new_witness", func(x *f33Input) { x.New.Members = map[string]f33Member{} }},
		{"zero_entitlement_revision", func(x *f33Input) { m := x.New.Members["a"]; m.Revision = 0; x.New.Members["a"] = m }},
		{"duplicate_old_device", func(x *f33Input) { x.Old.Bindings = append(x.Old.Bindings, x.Old.Bindings[0]) }},
		{"duplicate_applied_device", func(x *f33Input) { x.Installed.Bindings = append(x.Installed.Bindings, x.Installed.Bindings[0]) }},
		{"unknown_rate_mode_or_selection", func(x *f33Input) { x.New.Complete = false }},
		{"zero_host_reserve", func(x *f33Input) { x.New.HostReserve = 0 }},
		{"host_reserve_exhausts_capacity", func(x *f33Input) { x.New.HostReserve = 1000 }},
		{"old_node_ceil_violation", func(x *f33Input) { x.Installed.NodeCeil.Up++ }},
		{"old_member_ceil_violation", func(x *f33Input) { x.Installed.Ceil["a"] = f33Ceil{101, 200} }},
		{"missing_old_shape", func(x *f33Input) { delete(x.Installed.Ceil, "a") }},
		{"unknown_destination", func(x *f33Input) { x.New.Allowed = []netip.Prefix{{}} }},
		{"ipv6_denial_unsupported", func(x *f33Input) { x.New.Denied = []netip.Prefix{netip.MustParsePrefix("2001:db8::/32")} }},
		{"invalid_resolver", func(x *f33Input) { x.New.Resolvers = []netip.Addr{{}} }},
		{"ipv6_resolver_unsupported", func(x *f33Input) { x.New.Resolvers = []netip.Addr{netip.MustParseAddr("2001:db8::53")} }},
		{"empty_destinations", func(x *f33Input) { x.New.Allowed = nil }},
		{"empty_resolvers", func(x *f33Input) { x.New.Resolvers = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := f33Fixture()
			tt.edit(&x)
			if !f33Intersection(x).Baseline {
				t.Fatal("ambiguity admitted")
			}
		})
	}

	// Timestamps before old issued_at are normal; only future starts block.
	x := f33Fixture()
	x.Old.From, x.New.From = -time.Hour, -time.Minute
	x.Old.Bindings[0].From, x.New.Bindings[0].From = -time.Hour, -time.Minute
	m := x.Old.Members["a"]
	m.RateFrom = -time.Hour
	x.Old.Members["a"] = m
	m = x.New.Members["a"]
	m.RateFrom = -time.Minute
	x.New.Members["a"] = m
	if f33Devices(f33Intersection(x)) != "ab" {
		t.Fatal("valid start before anchor rejected")
	}

	// Decimal wire parsing is outside this normalized reference; integers
	// above JavaScript's safe range still compare exactly here.
	x = f33Fixture()
	x.Old.Generation, x.Lease.Generation, x.Installed.Generation = math.MaxUint64-1, math.MaxUint64-1, math.MaxUint64-1
	x.New.Generation = math.MaxUint64
	x.Old.Bindings[0].Revision, x.New.Bindings[0].Revision, x.Installed.Bindings[0].Revision = 1<<53, 1<<53, 1<<53
	if f33Devices(f33Intersection(x)) != "ab" {
		t.Fatal("wide identity narrowed")
	}
	x.New.Bindings[0].Revision++
	if f33Devices(f33Intersection(x)) != "b" {
		t.Fatal("adjacent wide revisions compared equal")
	}
	if f33MinCeil(f33Ceil{math.MaxUint64, math.MaxUint64 - 1}, f33Ceil{math.MaxUint64 - 1, math.MaxUint64}) != (f33Ceil{math.MaxUint64 - 1, math.MaxUint64 - 1}) {
		t.Fatal("wide ceilings lost precision")
	}
}
