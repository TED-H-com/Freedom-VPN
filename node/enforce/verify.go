package enforce

import (
	"fmt"
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
	if err := rejectBypass(rs); err != nil {
		return err
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
	fwd := rs.Chains["forward"]
	if !generationComment(fwd, m.Generation) {
		return checkf(ReasonGenerationMismatch, "forward rules do not carry fvpn_generation=%d", m.Generation)
	}
	if !strings.Contains(rs.Raw, "fvpn_metering="+MeteringVersion) {
		return checkf(ReasonMeterPathInvalid, "ruleset is not %s", MeteringVersion)
	}
	if err := checkForwardOrder(fwd); err != nil {
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
	in := rs.Chains["input"]
	if !hasDropOnIngress(in, m.Datapath.MemberIngress) {
		return checkf(ReasonIsolation, "input chain does not drop member ingress %s", m.Datapath.MemberIngress)
	}
	if err := verifyProbe(rs, m, false); err != nil {
		return err
	}
	return nil
}

// VerifyBaseline checks the fail-closed boot ruleset (§8.2, §17.1). A non-nil
// Probe is the only forward accept that is allowed.
func VerifyBaseline(raw []byte, m Model) error {
	if err := validateDatapath(m.Datapath); err != nil {
		return err
	}
	if m.Probe != nil {
		if err := validateProbe(m); err != nil {
			return err
		}
	}
	rs, err := ParseRuleset(raw)
	if err != nil {
		return checkf(ReasonIsolationBaselineMissing, "cannot parse nft listing: %v", err)
	}
	if err := rejectBypass(rs); err != nil {
		return &CheckError{Reason: ReasonIsolationBaselineMissing, Detail: err.Error()}
	}
	fwd, ok := rs.Chains["forward"]
	if !ok || fwd.Policy != "drop" || fwd.Hook != "forward" || !fwd.HasPrio || fwd.Prio != HookPriorityFilter {
		return checkf(ReasonIsolationBaselineMissing, "forward chain is not the drop baseline")
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
			if rule.Verdict == "accept" {
				if baseline {
					return checkf(ReasonIsolationBaselineMissing, "baseline forward chain contains an accept")
				}
				if rule.Counter == counterProbeUp || rule.Counter == counterProbeDown {
					return checkf(ReasonIsolation, "probe accept is present without a probe exception")
				}
			}
		}
		if len(rs.Elements[setProbeV4]) > 0 || len(rs.Elements[setProbeDstV4]) > 0 {
			return checkf(ReasonIsolation, "probe set is populated without a probe exception")
		}
		return nil
	}
	if err := verifyProbeElements(rs, m.Probe); err != nil {
		return err
	}
	if _, ok := rs.Counters[counterProbeUp]; !ok {
		return checkf(ReasonIsolation, "missing counter %s", counterProbeUp)
	}
	if _, ok := rs.Counters[counterProbeDown]; !ok {
		return checkf(ReasonIsolation, "missing counter %s", counterProbeDown)
	}
	var tuple, dns, down, other int
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
		case "down":
			down++
			downIdx = i
		default:
			if rule.Verdict == "accept" {
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
	if baseline && other > 0 {
		return checkf(ReasonIsolationBaselineMissing, "baseline forward chain contains an accept that is not the probe")
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
	if rule.Counter == counterProbeDown && hasName(rule.IIF, up) && hasName(rule.OIF, in) && hasName(rule.DaddrSets, setProbeV4) && rule.CTEstablished {
		return "down"
	}
	if rule.Counter != counterProbeUp || !hasName(rule.IIF, in) || !hasName(rule.OIF, up) || !hasName(rule.SaddrSets, setProbeV4) {
		return ""
	}
	if hasName(rule.ConcatSets, setProbeDstV4) {
		return "tuple"
	}
	if hasName(rule.DaddrSets, "resolver_v4") && hasPort(rule, 53) && (hasProto(rule, "tcp") || hasProto(rule, "udp")) {
		return "dns"
	}
	return ""
}

func verifyProbeElements(rs Ruleset, p *ProbeException) error {
	src := rs.Elements[setProbeV4]
	if len(src) != 1 || src[0].Concat || src[0].Addr != p.SourceV4.Addr() {
		return checkf(ReasonIsolation, "probe_v4 is not exactly the probe source")
	}
	if err := boundedElement(src[0], setProbeV4); err != nil {
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
		if err := boundedElement(el, setProbeDstV4); err != nil {
			return err
		}
		key := el.Addr.String() + "/" + el.Proto + "/" + fmt.Sprint(el.Port)
		if _, ok := want[key]; !ok {
			return checkf(ReasonIsolation, "probe_dst_v4 has unexpected %s", key)
		}
	}
	return nil
}

func boundedElement(el setElement, set string) error {
	if !el.Bounded || el.Seconds == 0 || el.Seconds > ProbeMaxSeconds {
		return checkf(ReasonProbeUnbounded, "%s element %s has no kernel timeout within %ds", set, el.Addr, ProbeMaxSeconds)
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

func hasPort(rule RuleView, port uint16) bool {
	for _, p := range rule.Dports {
		if p == port {
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
	if len(rs.Flowtables) > 0 || strings.Contains(rs.Raw, "\"flowtable\"") {
		return checkf(ReasonFlowtable, "flowtable would bypass the accounting hook")
	}
	for _, ch := range rs.Chains {
		for _, rule := range ch.Rules {
			if rule.Offload {
				return checkf(ReasonFlowtable, "rule offload would bypass the accounting hook")
			}
			if rule.SetsMark {
				return checkf(ReasonMark, "rule writes skb mark or conntrack mark")
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

func checkForwardOrder(ch ChainView) error {
	seenAllowDrop := false
	for _, rule := range ch.Rules {
		if rule.Verdict == "drop" && negatesSet(rule, "allowed_v4") {
			seenAllowDrop = true
		}
		if rule.Verdict == "accept" && rule.CTEstablished && !seenAllowDrop {
			return checkf(ReasonCounterOrder, "established accept is before the allow-set drop")
		}
		if rule.Verdict == "accept" && rule.Counter == "" {
			return checkf(ReasonCounterOrder, "accept rule has no named counter")
		}
	}
	if !seenAllowDrop {
		return checkf(ReasonCounterOrder, "no drop of sources outside the allow set before accept")
	}
	return nil
}

func negatesSet(rule RuleView, name string) bool {
	for _, set := range rule.NegSets {
		if set == name {
			return true
		}
	}
	return false
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
