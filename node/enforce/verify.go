package enforce

import (
	"fmt"
	"strconv"
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
	if !strings.Contains(rs.Raw, m.Datapath.MemberIngress) {
		return checkf(ReasonIsolation, "member ingress %s is not in the ruleset", m.Datapath.MemberIngress)
	}
	snat := rs.Chains["postrouting"]
	if !chainMentions(snat, m.Datapath.SNAT.String()) || !chainMentions(snat, m.Datapath.Uplink) {
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
		if !ruleHasCounterAndPriority(fwd, up, want) || !ruleHasCounterAndPriority(fwd, dn, want) {
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
	return nil
}

// VerifyBaseline checks the fail-closed boot ruleset (§8.2, §17.1).
func VerifyBaseline(raw []byte, dp Datapath) error {
	if err := validateDatapath(dp); err != nil {
		return err
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
	for _, rule := range fwd.Rules {
		if rule.Verdict == "accept" {
			return checkf(ReasonIsolationBaselineMissing, "baseline forward chain contains an accept")
		}
	}
	if !strings.Contains(rs.Raw, "fvpn_baseline=1") || !strings.Contains(rs.Raw, "fvpn_generation=0") {
		return checkf(ReasonIsolationBaselineMissing, "baseline generation marker is missing")
	}
	if !hasDropOnIngress(rs.Chains["input"], dp.MemberIngress) {
		return checkf(ReasonIsolationBaselineMissing, "input does not drop %s", dp.MemberIngress)
	}
	out, ok := rs.Chains["output"]
	if !ok || out.Policy != "accept" {
		return checkf(ReasonIsolationBaselineMissing, "output chain is not accept")
	}
	return nil
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
		if rule.Verdict == "drop" && strings.Contains(rule.Raw, "allowed_v4") {
			seenAllowDrop = true
		}
		if rule.Verdict == "accept" && strings.Contains(rule.Raw, "established") && !seenAllowDrop {
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

func chainMentions(ch ChainView, needle string) bool {
	for _, rule := range ch.Rules {
		if strings.Contains(rule.Raw, needle) || strings.Contains(rule.Comment, needle) {
			return true
		}
	}
	return false
}

func ruleHasCounterAndPriority(ch ChainView, counter string, classid uint32) bool {
	dec := strconv.FormatUint(uint64(classid), 10)
	hex := fmt.Sprintf("0x%x", classid)
	for _, rule := range ch.Rules {
		if rule.Counter != counter || rule.Verdict != "accept" {
			continue
		}
		for _, p := range rule.Prios {
			if p == classid {
				return true
			}
		}
		if strings.Contains(rule.Raw, dec) || strings.Contains(strings.ToLower(rule.Raw), hex) {
			return true
		}
	}
	return false
}

func hasDropOnIngress(ch ChainView, ifname string) bool {
	for _, rule := range ch.Rules {
		if rule.Verdict == "drop" && strings.Contains(rule.Raw, ifname) {
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
