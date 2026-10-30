package enforce

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/uuid"
)

// VerifyServing checks an nft -j listing against the model that was applied.
// assignments come from the render that was just installed.
func VerifyServing(raw []byte, m Model, assignments map[uuid.UUID]uint16) error {
	rs, err := ParseRuleset(raw)
	if err != nil {
		return checkf(ReasonMeterPathInvalid, "cannot parse nft listing: %v", err)
	}
	if err := verifyForeign(rs, m); err != nil {
		return err
	}
	if err := rejectBypass(rs); err != nil {
		return err
	}
	if err := validateProbe(m); err != nil {
		return checkf(ReasonProbeUnbounded, "%v", err)
	}
	devices, _, _, err := validateServing(m)
	if err != nil {
		return err
	}
	if err := expectChain(rs, "forward", "filter", "forward", "drop", HookPriorityFilter); err != nil {
		return err
	}
	if err := expectChain(rs, "input", "filter", "input", "accept", HookPriorityFilter); err != nil {
		return err
	}
	if err := expectChain(rs, "output", "filter", "output", "accept", HookPriorityFilter); err != nil {
		return err
	}
	if err := expectChain(rs, "postrouting", "nat", "postrouting", "accept", HookPrioritySNAT); err != nil {
		return err
	}
	if err := verifyChainSet(rs); err != nil {
		return err
	}
	if err := verifyModelRules(rs); err != nil {
		return err
	}
	fwd := rs.Chains["forward"]
	if !generationComment(fwd, m.Generation) {
		return checkf(ReasonGenerationMismatch, "forward rules do not carry fvpn_generation=%d", m.Generation)
	}
	if !strings.Contains(rs.Raw, "fvpn_metering="+MeteringVersion) {
		return checkf(ReasonMeterPathInvalid, "ruleset is not %s", MeteringVersion)
	}
	if err := checkForwardOrder(fwd, m); err != nil {
		return err
	}
	if !rulesetBindsIface(rs, m.Datapath.MemberIngress) {
		return checkf(ReasonIsolation, "member ingress %s is not in the ruleset", m.Datapath.MemberIngress)
	}
	snat := rs.Chains["postrouting"]
	if !chainHasSNAT(snat, m.Datapath.SNAT.String()) || !chainHasOIF(snat, m.Datapath.Uplink) {
		return checkf(ReasonMeterPathInvalid, "postrouting does not SNAT %s out %s", m.Datapath.SNAT, m.Datapath.Uplink)
	}
	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		up := usageCounter(dirUpload, d.Binding.DeviceID)
		dn := usageCounter(dirDownload, d.Binding.DeviceID)
		if _, ok := rs.Counters[up]; !ok {
			return checkf(ReasonDeviceClass, "missing upload counter %s", up)
		}
		if _, ok := rs.Counters[dn]; !ok {
			return checkf(ReasonDeviceClass, "missing download counter %s", dn)
		}
		want := ClassID(d.Minor)
		// Upload leaves through the uplink. Download arrives from the uplink.
		if !ruleBinds(fwd, up, want, m.Datapath.MemberIngress, m.Datapath.Uplink) ||
			!ruleBinds(fwd, dn, want, m.Datapath.Uplink, m.Datapath.MemberIngress) {
			return checkf(ReasonQoS, "device %s is not classified to classid 0x%x", d.Binding.DeviceID, want)
		}
		if assignments != nil {
			if got, ok := assignments[d.Binding.MemberID]; !ok || got != d.Minor {
				return checkf(ReasonQoS, "assignment for %s is not minor 0x%x", d.Binding.MemberID, d.Minor)
			}
		}
	}
	if err := verifyDeviceClass(fwd, devices); err != nil {
		return err
	}
	in := rs.Chains["input"]
	if err := verifyInputDrops(in, m.Datapath.MemberIngress); err != nil {
		return err
	}
	if err := verifyAllowedSet(rs); err != nil {
		return err
	}
	if err := verifySetFlags(rs); err != nil {
		return err
	}
	if err := verifyProbe(rs, m, false); err != nil {
		return err
	}
	if len(m.Resolvers) > 0 || (m.Probe != nil && m.Probe.ResolverDNS) {
		if err := verifyResolverSet(rs, m); err != nil {
			return err
		}
	}
	if err := verifyIsolationDrops(fwd, m); err != nil {
		return err
	}
	return nil
}

// verifyAllowedSet fails closed when the member allow set contains a prefix,
// a range, or any element the parser cannot accept as a timed host address.
func verifyAllowedSet(rs Ruleset) error {
	if rs.Tainted["allowed_v4"] {
		return checkf(ReasonIsolation, "allowed_v4 contains a prefix, range, or unknown element")
	}
	return nil
}

// VerifyBaseline checks the fail-closed boot ruleset (§8.2, §17.1). A non-nil
// Probe is the only forward accept that is allowed.
func VerifyBaseline(raw []byte, m Model) error {
	if err := validateDatapath(m.Datapath); err != nil {
		return err
	}
	if err := validateProbe(m); err != nil {
		return checkf(ReasonProbeUnbounded, "%v", err)
	}
	rs, err := ParseRuleset(raw)
	if err != nil {
		return checkf(ReasonIsolationBaselineMissing, "cannot parse nft listing: %v", err)
	}
	if err := rejectBypass(rs); err != nil {
		return &CheckError{Reason: ReasonIsolationBaselineMissing, Detail: err.Error()}
	}
	if err := verifyModelRules(rs); err != nil {
		return err
	}
	fwd, ok := rs.Chains["forward"]
	if !ok || fwd.Policy != "drop" || fwd.Hook != "forward" || !fwd.HasPrio || fwd.Prio != HookPriorityFilter {
		return checkf(ReasonIsolationBaselineMissing, "forward chain is not the drop baseline")
	}
	if err := verifyAllowedSet(rs); err != nil {
		return err
	}
	if err := verifyProbe(rs, m, true); err != nil {
		if m.Probe == nil {
			if ce, ok := err.(*CheckError); ok && ce.Reason == ReasonProbeUnbounded {
				return err
			}
			return &CheckError{Reason: ReasonIsolationBaselineMissing, Detail: err.Error()}
		}
		return err
	}
	if !strings.Contains(rs.Raw, "fvpn_baseline=1") || !strings.Contains(rs.Raw, "fvpn_generation=0") {
		return checkf(ReasonIsolationBaselineMissing, "baseline generation marker is missing")
	}
	if !hasDropOnIngress(rs.Chains["input"], m.Datapath.MemberIngress) {
		return checkf(ReasonIsolationBaselineMissing, "input does not drop %s", m.Datapath.MemberIngress)
	}
	out, ok := rs.Chains["output"]
	if !ok || out.Policy != "accept" {
		return checkf(ReasonIsolationBaselineMissing, "output chain is not accept")
	}
	return nil
}

// verifyProbe checks the synthetic-probe sets, element timeouts, and accepts.
// baseline is true when any other forward accept is a failure.
func verifyProbe(rs Ruleset, m Model, baseline bool) error {
	fwd := rs.Chains["forward"]
	if m.Probe == nil {
		for _, rule := range fwd.Rules {
			if rule.Verdict != "accept" {
				continue
			}
			if rule.Counter == counterProbeUp || rule.Counter == counterProbeDown {
				return checkf(ReasonIsolation, "probe accept is present without a probe exception")
			}
			if baseline {
				return checkf(ReasonIsolationBaselineMissing, "baseline forward chain contains an accept")
			}
			if servingAcceptKind(rule, m) == "" {
				return checkf(ReasonIsolation, "forward accept is outside the serving model")
			}
		}
		if rs.Tainted[setProbeV4] || rs.Tainted[setProbeDstV4] {
			return checkf(ReasonProbeUnbounded, "probe set contains a prefix, range, or unknown element")
		}
		if len(rs.Elements[setProbeV4]) > 0 || len(rs.Elements[setProbeDstV4]) > 0 {
			return checkf(ReasonIsolation, "probe set is populated without a probe exception")
		}
		return nil
	}
	if err := verifyProbeElements(rs, m); err != nil {
		return err
	}
	if m.Probe.ResolverDNS {
		if err := verifyResolverSet(rs, m); err != nil {
			return err
		}
	}
	if _, ok := rs.Counters[counterProbeUp]; !ok {
		return checkf(ReasonIsolation, "missing counter %s", counterProbeUp)
	}
	if _, ok := rs.Counters[counterProbeDown]; !ok {
		return checkf(ReasonIsolation, "missing counter %s", counterProbeDown)
	}
	var tuple, dns, down, other, dnsTCP, dnsUDP int
	var v6Idx, denyIdx, hostIdx, tupleIdx, dnsIdx, downIdx, memberIdx = -1, -1, -1, -1, -1, -1, -1
	for i, rule := range fwd.Rules {
		if rule.Verdict == "drop" && rule.NFProto == "ipv6" && v6Idx < 0 {
			v6Idx = i
		}
		if rule.Verdict == "drop" && hasName(rule.DaddrSets, "denied_v4") {
			denyIdx = i
		}
		if rule.Verdict == "drop" && hasName(rule.DaddrSets, "host_v4") {
			hostIdx = i
		}
		kind := classifyProbeRule(rule, m)
		switch kind {
		case "tuple":
			tuple++
			tupleIdx = i
		case "dns":
			dns++
			if dnsIdx < 0 {
				dnsIdx = i
			}
			if hasProto(rule, "tcp") {
				dnsTCP++
			}
			if hasProto(rule, "udp") {
				dnsUDP++
			}
		case "down":
			down++
			downIdx = i
		default:
			if rule.Verdict == "accept" {
				// A probe-source accept that is not the port-53 pair or the
				// destination tuple would outlive the exception's shape.
				// Serving and baseline both fail closed (§13.9).
				if probeScopedAccept(rule) {
					return checkf(ReasonIsolation, "probe source accept is wider than the DNS or tuple exception")
				}
				if !baseline && servingAcceptKind(rule, m) != "" {
					if strings.HasPrefix(rule.Counter, "u_") && memberIdx < 0 {
						memberIdx = i
					}
					break
				}
				other++
				if strings.HasPrefix(rule.Counter, "u_") && memberIdx < 0 {
					memberIdx = i
				}
			}
		}
	}
	wantTuple, wantDNS, wantDown := expectedProbeAccepts(m.Probe)
	if tuple != wantTuple || dns != wantDNS || down != wantDown {
		return checkf(ReasonIsolation, "probe accepts tuple=%d dns=%d down=%d, want %d/%d/%d", tuple, dns, down, wantTuple, wantDNS, wantDown)
	}
	if wantDNS > 0 && (dnsTCP != 1 || dnsUDP != 1) {
		return checkf(ReasonIsolation, "probe DNS accept is not one tcp and one udp port 53")
	}
	if other > 0 {
		if baseline {
			return checkf(ReasonIsolationBaselineMissing, "baseline forward chain contains an accept that is not the probe")
		}
		return checkf(ReasonIsolation, "forward accept is outside the serving model")
	}
	if wantTuple > 0 && (v6Idx < 0 || denyIdx < 0 || hostIdx < 0 || !(v6Idx < tupleIdx && denyIdx < tupleIdx && hostIdx < tupleIdx)) {
		return checkf(ReasonIsolation, "probe upload accept is not behind the v6, deny, and host drops")
	}
	if wantTuple > 0 && memberIdx >= 0 && tupleIdx > memberIdx {
		return checkf(ReasonIsolation, "probe upload accept is after a member accept")
	}
	if wantDNS > 0 && (v6Idx < 0 || dnsIdx < 0 || v6Idx > dnsIdx) {
		return checkf(ReasonIsolation, "probe DNS accept is not behind the v6 drop")
	}
	if wantDNS > 0 && (dnsIdx < 0 || denyIdx < 0 || dnsIdx > denyIdx) {
		return checkf(ReasonIsolation, "probe DNS accept is not before the deny drop")
	}
	if downIdx < 0 || (denyIdx >= 0 && downIdx < denyIdx) {
		return checkf(ReasonIsolation, "probe download accept is before the deny drop")
	}
	if !hasProbeSNAT(rs.Chains["postrouting"], m) {
		return checkf(ReasonMeterPathInvalid, "postrouting does not SNAT the probe source out %s", m.Datapath.Uplink)
	}
	return nil
}

func expectedProbeAccepts(p *ProbeException) (tuple, dns, down int) {
	if len(p.Destinations) > 0 {
		tuple = 1
	}
	if p.ResolverDNS {
		dns = 2
	}
	down = 1
	return tuple, dns, down
}

func classifyProbeRule(rule RuleView, m Model) string {
	in := m.Datapath.MemberIngress
	up := m.Datapath.Uplink
	if rule.Verdict != "accept" {
		return ""
	}
	if rule.Counter == counterProbeDown && exactNames(rule.IIF, up) && exactNames(rule.OIF, in) && exactNames(rule.DaddrSets, setProbeV4) && len(rule.SaddrSets) == 0 && len(rule.ConcatSets) == 0 && len(rule.Dports) == 0 && rule.CTEstablished {
		return "down"
	}
	if rule.Counter != counterProbeUp || !exactNames(rule.IIF, in) || !exactNames(rule.OIF, up) || !exactNames(rule.SaddrSets, setProbeV4) || rule.CTEstablished {
		return ""
	}
	if exactNames(rule.ConcatSets, setProbeDstV4) && len(rule.DaddrSets) == 0 && len(rule.Dports) == 0 && len(rule.L4Protos) == 0 {
		return "tuple"
	}
	// One L4 protocol, one destination port, one destination set. Anything
	// else is not the resolver carve-out and must not count as it.
	if exactNames(rule.DaddrSets, "resolver_v4") && len(rule.ConcatSets) == 0 && len(rule.Dports) == 1 && rule.Dports[0] == 53 && len(rule.L4Protos) == 1 && (rule.L4Protos[0] == "tcp" || rule.L4Protos[0] == "udp") {
		return "dns"
	}
	return ""
}

// probeScopedAccept is an accept that names the probe source, the probe
// destination tuple, or the probe address and was not classified above.
func probeScopedAccept(rule RuleView) bool {
	return hasName(rule.SaddrSets, setProbeV4) || hasName(rule.DaddrSets, setProbeV4) || hasName(rule.ConcatSets, setProbeV4) || hasName(rule.ConcatSets, setProbeDstV4)
}

func exactNames(names []string, want string) bool {
	return len(names) == 1 && names[0] == want
}

// verifyResolverSet requires @resolver_v4 to be exactly the configured
// resolver addresses. A prefix, a range, or an extra address fails closed.
// The detail does not name an address. Callers decide when the hole is open:
// serving checks it whenever the model lists resolvers, not only for a probe.
func verifyResolverSet(rs Ruleset, m Model) error {
	if rs.Tainted["resolver_v4"] {
		return checkf(ReasonIsolation, "resolver_v4 contains a prefix, range, or unknown element")
	}
	want := map[netip.Addr]struct{}{}
	for _, addr := range m.Resolvers {
		if !addr.IsValid() || !addr.Is4() {
			return checkf(ReasonIsolation, "resolver_v4 is not exactly the configured resolvers")
		}
		want[addr] = struct{}{}
	}
	got := rs.Elements["resolver_v4"]
	if len(got) != len(want) {
		return checkf(ReasonIsolation, "resolver_v4 is not exactly the configured resolvers")
	}
	seen := map[netip.Addr]struct{}{}
	for _, el := range got {
		if el.Concat || el.Port != 0 || el.Proto != "" {
			return checkf(ReasonIsolation, "resolver_v4 is not exactly the configured resolvers")
		}
		if _, ok := want[el.Addr]; !ok {
			return checkf(ReasonIsolation, "resolver_v4 is not exactly the configured resolvers")
		}
		if _, dup := seen[el.Addr]; dup {
			return checkf(ReasonIsolation, "resolver_v4 is not exactly the configured resolvers")
		}
		seen[el.Addr] = struct{}{}
	}
	return nil
}

func verifyProbeElements(rs Ruleset, m Model) error {
	p := m.Probe
	if rs.Tainted[setProbeV4] || rs.Tainted[setProbeDstV4] {
		return checkf(ReasonProbeUnbounded, "probe set contains a prefix, range, or unknown element")
	}
	limit, ok := probeTimeout(p.ExpiresAt, m.Now)
	if !ok || limit > ProbeMaxSeconds {
		return checkf(ReasonProbeUnbounded, "probe expiration is not admissible")
	}
	src := rs.Elements[setProbeV4]
	if len(src) != 1 || src[0].Concat || src[0].Addr != p.SourceV4.Addr() {
		return checkf(ReasonIsolation, "probe_v4 is not exactly the probe source")
	}
	if err := boundedElement(src[0], setProbeV4, limit); err != nil {
		return err
	}
	want := map[string]ProbeDestination{}
	for _, d := range p.Destinations {
		key := d.AddrV4.Addr().String() + "/" + d.Proto + "/" + fmt.Sprint(d.Port)
		want[key] = d
	}
	got := rs.Elements[setProbeDstV4]
	if len(want) == 0 {
		if len(got) > 0 {
			return checkf(ReasonIsolation, "probe_dst_v4 is populated without destinations")
		}
		return nil
	}
	if len(got) != len(want) {
		return checkf(ReasonIsolation, "probe_dst_v4 has %d elements, want %d", len(got), len(want))
	}
	for _, el := range got {
		if err := boundedElement(el, setProbeDstV4, limit); err != nil {
			return err
		}
		key := el.Addr.String() + "/" + el.Proto + "/" + fmt.Sprint(el.Port)
		if _, ok := want[key]; !ok {
			return checkf(ReasonIsolation, "probe_dst_v4 has unexpected %s", key)
		}
	}
	return nil
}

// boundedElement accepts a probe element when it has a kernel timeout, the
// original timeout (max of timeout and expires) is within ProbeMaxSeconds,
// and the remaining lifetime is within limit+1. limit is the model window.
// The extra second covers the floor in probeTimeout and the tick between the
// listing and Now. In the kernel, timeout does not shrink and expires does.
func boundedElement(el setElement, set string, limit uint32) error {
	if !el.Bounded || el.Seconds == 0 {
		return checkf(ReasonProbeUnbounded, "%s element %s has no kernel timeout within ceiling %ds", set, el.Addr, ProbeMaxSeconds)
	}
	if el.Seconds > ProbeMaxSeconds {
		return checkf(ReasonProbeUnbounded, "%s element %s original timeout %ds exceeds ceiling %ds", set, el.Addr, el.Seconds, ProbeMaxSeconds)
	}
	bound := limit + 1
	if el.Window == 0 || el.Window > bound {
		return checkf(ReasonProbeUnbounded, "%s element %s remaining %ds exceeds bound %ds", set, el.Addr, el.Window, bound)
	}
	return nil
}

func hasProbeSNAT(ch ChainView, m Model) bool {
	for _, rule := range ch.Rules {
		if rule.SNAT == m.Datapath.SNAT.String() && hasName(rule.SaddrSets, setProbeV4) && hasName(rule.OIF, m.Datapath.Uplink) {
			return true
		}
	}
	return false
}

func hasProto(rule RuleView, proto string) bool {
	for _, p := range rule.L4Protos {
		if p == proto {
			return true
		}
	}
	return false
}

// VerifyQoS checks HTB parents, member leaves and fq_codel on both directions
// (§11.4). devJSON maps an interface name to the pair (qdisc JSON, class JSON).
func VerifyQoS(devJSON map[string][2][]byte, m Model, assignments map[uuid.UUID]uint16) error {
	if _, _, _, err := validateServing(m); err != nil {
		return err
	}
	sides := []struct {
		iface  string
		parent Rate
		host   Rate
		pick   func(MemberQoS) Rate
	}{
		{m.Datapath.Uplink, m.QoS.Up, m.QoS.HostUp, func(q MemberQoS) Rate { return q.Up }},
		{m.Datapath.MemberIngress, m.QoS.Down, m.QoS.HostDown, func(q MemberQoS) Rate { return q.Down }},
	}
	for _, s := range sides {
		pair, ok := devJSON[s.iface]
		if !ok {
			return checkf(ReasonQoS, "no tc listing for %s", s.iface)
		}
		view, err := ParseQdiscJSON(pair[0])
		if err != nil {
			return checkf(ReasonQoS, "%s qdisc: %v", s.iface, err)
		}
		view, err = ParseClassJSON(pair[1], view)
		if err != nil {
			return checkf(ReasonQoS, "%s class: %v", s.iface, err)
		}
		if !view.RootHTB {
			return checkf(ReasonQoS, "%s has no root HTB", s.iface)
		}
		if err := expectClass(view, s.iface, classHandle(1), s.parent); err != nil {
			return err
		}
		if err := expectClass(view, s.iface, classHandle(HTBDefaultMinor), s.host); err != nil {
			return err
		}
		for _, mem := range m.QoS.Members {
			minor, ok := assignments[mem.MemberID]
			if !ok {
				return checkf(ReasonQoS, "no class minor for %s", mem.MemberID)
			}
			if err := expectClass(view, s.iface, classHandle(minor), s.pick(mem)); err != nil {
				return err
			}
			if !view.Leaves[minor] {
				return checkf(ReasonQoS, "%s class %s has no fq_codel leaf", s.iface, classHandle(minor))
			}
		}
	}
	return nil
}

// EvaluateMeterPath reports METER_PATH_INVALID when packets on the member
// ingress do not belong to a current binding. That is the fail-closed
// reaction to a routing namespace that masqueraded the member address away
// (§10.2). An empty observation is not a failure.
func EvaluateMeterPath(bindings []Binding, ingressSources []string) error {
	if len(ingressSources) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, b := range bindings {
		for _, p := range b.AllowedV4 {
			if p.IsValid() && p.Bits() == 32 {
				allowed[p.Addr().String()] = true
			}
		}
	}
	if len(allowed) == 0 {
		return checkf(ReasonMeterPathInvalid, "ingress traffic with no device bindings")
	}
	var unknown []string
	for _, src := range ingressSources {
		if !allowed[src] {
			unknown = append(unknown, src)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	return checkf(ReasonMeterPathInvalid, "ingress sources %s match no device binding; refusing to guess", strings.Join(unknown, ","))
}

func rejectBypass(rs Ruleset) error {
	if len(rs.Flowtables) > 0 {
		return checkf(ReasonFlowtable, "flowtable would bypass the accounting hook")
	}
	for _, ch := range rs.Chains {
		for _, rule := range ch.Rules {
			if rule.Offload || strings.Contains(rule.Raw, "\"flowtable\"") {
				return checkf(ReasonFlowtable, "rule offload would bypass the accounting hook")
			}
			if rule.SetsMark {
				return checkf(ReasonMark, "rule writes skb mark or conntrack mark")
			}
			if rule.DNAT {
				return checkf(ReasonIsolation, "rule rewrites the destination address")
			}
		}
	}
	if strings.Contains(rs.Raw, "flush ruleset") {
		return checkf(ReasonIsolation, "ruleset contains flush ruleset")
	}
	return nil
}

func expectChain(rs Ruleset, name, typ, hook, policy string, prio int64) error {
	ch, ok := rs.Chains[name]
	if !ok || ch.Hook == "" {
		return checkf(ReasonHookPriority, "chain %s is missing", name)
	}
	if ch.Type != typ || ch.Hook != hook || ch.Policy != policy || !ch.HasPrio || ch.Prio != prio {
		return checkf(ReasonHookPriority, "chain %s is %s hook %s prio %d policy %s, want %s hook %s prio %d policy %s",
			name, ch.Type, ch.Hook, ch.Prio, ch.Policy, typ, hook, prio, policy)
	}
	return nil
}

func generationComment(ch ChainView, generation uint64) bool {
	want := fmt.Sprintf("fvpn_generation=%d", generation)
	for _, rule := range ch.Rules {
		rest, ok := strings.CutPrefix(rule.Comment, want)
		if !ok {
			idx := strings.Index(rule.Comment, want)
			if idx < 0 {
				continue
			}
			rest = rule.Comment[idx+len(want):]
		}
		if rest == "" || rest[0] < '0' || rest[0] > '9' {
			return true
		}
	}
	return false
}

func checkForwardOrder(ch ChainView, m Model) error {
	seenUp := false
	seenDown := false
	seenNotEst := false
	for _, rule := range ch.Rules {
		if rule.Verdict == "drop" && hasName(rule.NegSaddr, "allowed_v4") {
			seenUp = true
		}
		if rule.Verdict == "drop" && hasName(rule.NegDaddr, "allowed_v4") {
			seenDown = true
		}
		if rule.Verdict == "drop" && rule.CTNotEstablished {
			seenNotEst = true
		}
		if rule.Verdict != "accept" {
			continue
		}
		if rule.Counter == "" {
			return checkf(ReasonCounterOrder, "accept rule has no named counter")
		}
		upload, download := acceptDirections(rule, m)
		if upload && !seenUp {
			return checkf(ReasonCounterOrder, "upload accept is before the allow-set drop")
		}
		if download && !seenDown {
			return checkf(ReasonCounterOrder, "download accept is before the allow-set drop")
		}
		if strings.HasPrefix(rule.Counter, "d_") && !strings.HasPrefix(rule.Counter, "nd_") && !seenNotEst {
			return checkf(ReasonCounterOrder, "download accept is before the non-established drop")
		}
	}
	if !seenUp || !seenDown {
		return checkf(ReasonCounterOrder, "allow-set drop is missing for upload or download")
	}
	return nil
}

func acceptDirections(rule RuleView, m Model) (upload, download bool) {
	onMember := hasName(rule.IIF, m.Datapath.MemberIngress)
	onUplink := hasName(rule.IIF, m.Datapath.Uplink)
	if !onMember && !onUplink {
		return true, true
	}
	return onMember, onUplink
}

// servingAcceptKind classifies a forward accept Render emits for a member.
// Probe accepts are classified separately. Anything else is outside the model.
func servingAcceptKind(rule RuleView, m Model) string {
	if rule.Verdict != "accept" || rule.SetsMark || rule.MatchMark || rule.Offload || rule.DNAT || rule.CTEstablished {
		return ""
	}
	if rule.NFProto == "ipv6" || len(rule.Prios) != 1 || len(rule.NegSets) != 0 || len(rule.ConcatSets) != 0 {
		return ""
	}
	in := m.Datapath.MemberIngress
	up := m.Datapath.Uplink
	switch {
	case deviceCounter(rule.Counter, "nu_") && exactNames(rule.IIF, in) && exactNames(rule.OIF, up) &&
		exactNames(rule.DaddrSets, "resolver_v4") && len(rule.SaddrSets) == 0 &&
		len(rule.Dports) == 1 && rule.Dports[0] == 53 && len(rule.Sports) == 0 &&
		len(rule.L4Protos) == 1 && (rule.L4Protos[0] == "tcp" || rule.L4Protos[0] == "udp") &&
		oneHost(rule.Saddrs) && len(rule.Daddrs) == 0:
		return "dns-up"
	case deviceCounter(rule.Counter, "nd_") && exactNames(rule.IIF, up) && exactNames(rule.OIF, in) &&
		exactNames(rule.SaddrSets, "resolver_v4") && len(rule.DaddrSets) == 0 &&
		len(rule.Sports) == 1 && rule.Sports[0] == 53 && len(rule.Dports) == 0 &&
		len(rule.L4Protos) == 1 && (rule.L4Protos[0] == "tcp" || rule.L4Protos[0] == "udp") &&
		len(rule.Saddrs) == 0 && oneHost(rule.Daddrs):
		return "dns-down"
	case deviceCounter(rule.Counter, "u_") && exactNames(rule.IIF, in) && exactNames(rule.OIF, up) &&
		len(rule.SaddrSets) == 0 && len(rule.DaddrSets) == 0 &&
		len(rule.Dports) == 0 && len(rule.Sports) == 0 && len(rule.L4Protos) == 0 &&
		oneHost(rule.Saddrs) && len(rule.Daddrs) == 0:
		return "upload"
	case deviceCounter(rule.Counter, "d_") && exactNames(rule.IIF, up) && exactNames(rule.OIF, in) &&
		len(rule.SaddrSets) == 0 && len(rule.DaddrSets) == 0 &&
		len(rule.Dports) == 0 && len(rule.Sports) == 0 && len(rule.L4Protos) == 0 &&
		len(rule.Saddrs) == 0 && oneHost(rule.Daddrs):
		return "download"
	default:
		return ""
	}
}

func deviceCounter(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	_, err := uuid.Parse(name[len(prefix):])
	return err == nil
}

func oneHost(ps []netip.Prefix) bool {
	return len(ps) == 1 && ps[0].Bits() == 32 && ps[0].Addr().Is4()
}

// verifyModelRules requires the rendered shape on every chain. The generation
// and class markers are the only rules without a verdict. Output stays empty:
// a jump into it would otherwise leave forward.
func verifyModelRules(rs Ruleset) error {
	if len(rs.Chains["output"].Rules) > 0 {
		return checkf(ReasonIsolation, "output chain is not empty")
	}
	for _, name := range []string{"forward", "input", "postrouting"} {
		for _, rule := range rs.Chains[name].Rules {
			if err := verifyRuleShape(name, rule); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyRuleShape(chain string, rule RuleView) error {
	if rule.Tainted || rule.Verdicts > 1 {
		return checkf(ReasonIsolation, "rule uses an expression the renderer does not emit")
	}
	if isMarker(rule) {
		return nil
	}
	switch chain {
	case "forward":
		if rule.Verdict != "accept" && rule.Verdict != "drop" {
			return checkf(ReasonIsolation, "forward rule has no verdict")
		}
	case "input":
		if rule.Verdict != "drop" {
			return checkf(ReasonIsolation, "input rule is not a drop")
		}
	case "postrouting":
		if rule.Verdict != "" || rule.SNAT == "" {
			return checkf(ReasonIsolation, "postrouting rule is not SNAT")
		}
	}
	if rule.Verdict == "drop" && (rule.MatchMark || len(rule.L4Protos) > 0 || len(rule.Dports) > 0 || len(rule.Sports) > 0) {
		return checkf(ReasonIsolation, "drop rule carries a match the renderer does not emit")
	}
	return nil
}

func isMarker(rule RuleView) bool {
	if rule.Verdict != "" || rule.SNAT != "" || rule.Verdicts != 0 || rule.Tainted {
		return false
	}
	return strings.Contains(rule.Comment, "fvpn_generation=") || strings.Contains(rule.Comment, "fvpn_class=")
}

// verifyDeviceClass requires every device accept to name exactly one admitted
// address and that device's HTB class. One matching rule is not enough: an
// extra accept for the same counter at the host default class still fails.
func verifyDeviceClass(fwd ChainView, devices []admittedDevice) error {
	byID := map[uuid.UUID]admittedDevice{}
	for _, d := range devices {
		if d.TimeoutOK() {
			byID[d.Binding.DeviceID] = d
		}
	}
	for _, rule := range fwd.Rules {
		if rule.Verdict != "accept" {
			continue
		}
		id, _, _, ok := parseDeviceCounter(rule.Counter)
		if !ok {
			continue
		}
		d, known := byID[id]
		got, other, dirOK := deviceHostSide(rule)
		if !known || !dirOK {
			return checkf(ReasonQoS, "device rule priority is not the assigned class")
		}
		if len(other) != 0 || !oneHost(got) || !addrIn(got[0].Addr(), d.Addrs) || len(rule.Prios) != 1 || rule.Prios[0] != ClassID(d.Minor) {
			return checkf(ReasonQoS, "device rule priority is not the assigned class")
		}
	}
	return nil
}

func deviceHostSide(rule RuleView) (got, other []netip.Prefix, ok bool) {
	switch {
	case strings.HasPrefix(rule.Counter, "nu_"), strings.HasPrefix(rule.Counter, "u_"):
		return rule.Saddrs, rule.Daddrs, true
	case strings.HasPrefix(rule.Counter, "nd_"), strings.HasPrefix(rule.Counter, "d_"):
		return rule.Daddrs, rule.Saddrs, true
	default:
		return nil, nil, false
	}
}

func addrIn(addr netip.Addr, addrs []netip.Addr) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}

// verifyChainSet allows only the base chains Render emits. A regular chain
// or a nat prerouting hook inside inet fvpn is rejected.
func verifyChainSet(rs Ruleset) error {
	allowed := map[string]bool{"forward": true, "input": true, "output": true, "postrouting": true}
	if len(rs.Chains) != len(allowed) {
		return checkf(ReasonHookPriority, "ruleset has %d chains, want the four rendered chains", len(rs.Chains))
	}
	for name := range rs.Chains {
		if !allowed[name] {
			return checkf(ReasonHookPriority, "chain %s is not part of the rendered ruleset", name)
		}
	}
	return nil
}

// verifySetFlags rejects a flag the renderer does not emit on the sets that
// must stay host addresses. denied_v4 and host_v4 are interval sets on purpose.
func verifySetFlags(rs Ruleset) error {
	allowed := map[string]map[string]bool{
		"allowed_v4":  {"timeout": true},
		setProbeV4:    {"timeout": true},
		setProbeDstV4: {"timeout": true},
		"resolver_v4": {},
	}
	for name, okFlag := range allowed {
		for _, flag := range rs.SetFlags[name] {
			if !okFlag[flag] {
				return checkf(ReasonIsolation, "set %s has a flag the renderer does not emit", name)
			}
		}
	}
	return nil
}

var cgnatV4 = netip.MustParsePrefix("100.64.0.0/10")

// verifyInputDrops requires both input drops, and no accept of member-ingress
// or CGNAT traffic before the later of the two.
func verifyInputDrops(ch ChainView, ifname string) error {
	iifIdx, cgnatIdx := -1, -1
	for i, rule := range ch.Rules {
		if rule.Verdict == "drop" && hasName(rule.IIF, ifname) && iifIdx < 0 {
			iifIdx = i
		}
		if rule.Verdict == "drop" && coversPrefix(rule.Saddrs, cgnatV4) && cgnatIdx < 0 {
			cgnatIdx = i
		}
	}
	if iifIdx < 0 {
		return checkf(ReasonIsolation, "input chain does not drop member ingress %s", ifname)
	}
	if cgnatIdx < 0 {
		return checkf(ReasonIsolation, "input chain does not drop 100.64.0.0/10")
	}
	need := iifIdx
	if cgnatIdx > need {
		need = cgnatIdx
	}
	for i, rule := range ch.Rules {
		if rule.Verdict != "accept" || i >= need {
			continue
		}
		if hasName(rule.IIF, ifname) || coversPrefix(rule.Saddrs, cgnatV4) || (len(rule.IIF) == 0 && len(rule.Saddrs) == 0) {
			return checkf(ReasonIsolation, "input accept precedes the member and CGNAT drops")
		}
	}
	return nil
}

func coversPrefix(got []netip.Prefix, want netip.Prefix) bool {
	for _, p := range got {
		if p.Bits() == want.Bits() && p.Addr() == want.Addr() {
			return true
		}
	}
	return false
}

// verifyIsolationDrops requires the deny, host, and non-53 resolver drops
// whenever the model has bindings or resolvers, including when Probe is nil.
func verifyIsolationDrops(ch ChainView, m Model) error {
	if len(m.Bindings) == 0 && len(m.Resolvers) == 0 {
		return nil
	}
	denyIdx, hostIdx, resIdx := -1, -1, -1
	for i, rule := range ch.Rules {
		if rule.Verdict != "drop" {
			continue
		}
		if hasName(rule.DaddrSets, "denied_v4") && denyIdx < 0 {
			denyIdx = i
		}
		if hasName(rule.DaddrSets, "host_v4") && hostIdx < 0 {
			hostIdx = i
		}
		if hasName(rule.DaddrSets, "resolver_v4") && resIdx < 0 {
			resIdx = i
		}
	}
	if denyIdx < 0 {
		return checkf(ReasonIsolation, "forward chain does not drop @denied_v4")
	}
	if hostIdx < 0 {
		return checkf(ReasonIsolation, "forward chain does not drop @host_v4")
	}
	if len(m.Resolvers) > 0 || (m.Probe != nil && m.Probe.ResolverDNS) {
		if resIdx < 0 {
			return checkf(ReasonIsolation, "forward chain does not drop non-53 resolver traffic")
		}
		if resIdx > denyIdx {
			return checkf(ReasonIsolation, "resolver drop is not before the deny drop")
		}
		for i, rule := range ch.Rules {
			if rule.Verdict != "accept" || i <= resIdx {
				continue
			}
			if strings.HasPrefix(rule.Counter, "nu_") || (rule.Counter == counterProbeUp && hasName(rule.DaddrSets, "resolver_v4")) {
				return checkf(ReasonIsolation, "DNS accept is not before the resolver drop")
			}
		}
	}
	for i, rule := range ch.Rules {
		if rule.Verdict == "accept" && strings.HasPrefix(rule.Counter, "u_") && !strings.HasPrefix(rule.Counter, "nu_") {
			if i < denyIdx || i < hostIdx {
				return checkf(ReasonIsolation, "member upload accept is not behind the deny and host drops")
			}
		}
	}
	return nil
}

func chainHasOIF(ch ChainView, name string) bool {
	for _, rule := range ch.Rules {
		if hasName(rule.OIF, name) {
			return true
		}
	}
	return false
}

func chainHasSNAT(ch ChainView, addr string) bool {
	for _, rule := range ch.Rules {
		if rule.SNAT == addr {
			return true
		}
	}
	return false
}

func rulesetBindsIface(rs Ruleset, name string) bool {
	for _, ch := range rs.Chains {
		for _, rule := range ch.Rules {
			if hasName(rule.IIF, name) || hasName(rule.OIF, name) {
				return true
			}
		}
	}
	return false
}

func ruleBinds(ch ChainView, counter string, classid uint32, iif, oif string) bool {
	for _, rule := range ch.Rules {
		if rule.Counter != counter || rule.Verdict != "accept" {
			continue
		}
		if !hasPrio(rule, classid) || !hasName(rule.IIF, iif) || !hasName(rule.OIF, oif) {
			continue
		}
		return true
	}
	return false
}

func hasPrio(rule RuleView, classid uint32) bool {
	for _, p := range rule.Prios {
		if p == classid {
			return true
		}
	}
	return false
}

func hasName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func hasDropOnIngress(ch ChainView, ifname string) bool {
	for _, rule := range ch.Rules {
		if rule.Verdict == "drop" && hasName(rule.IIF, ifname) {
			return true
		}
	}
	return false
}

func expectClass(view QdiscView, iface, handle string, want Rate) error {
	got, ok := view.Classes[handle]
	if !ok || !got.Found {
		return checkf(ReasonQoS, "%s class %s is missing", iface, handle)
	}
	if !rateClose(got.RateBPS, want.RateBPS) || !rateClose(got.CeilBPS, want.CeilBPS) {
		return checkf(ReasonQoS, "%s class %s rate %d/%d, want %d/%d", iface, handle, got.RateBPS, got.CeilBPS, want.RateBPS, want.CeilBPS)
	}
	return nil
}

func rateClose(got, want uint64) bool {
	if got == want {
		return true
	}
	if want == 0 {
		return false
	}
	diff := got
	if got > want {
		diff = got - want
	} else {
		diff = want - got
	}
	return diff*100 <= want
}
