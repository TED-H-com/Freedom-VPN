package enforce

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// RenderPlan is one atomic nft transaction plus the tc batch that matches it.
type RenderPlan struct {
	NFT         string
	TC          string
	Assignments map[uuid.UUID]uint16
	// Timeouts is the kernel allow-set lifetime per admitted device, in seconds.
	Timeouts map[string]uint32
	// Finals are last readings for device counters this plan stops referencing.
	// The caller persists them before it considers the device gone (§10.7).
	Finals   []Sample
	Admitted int
}

// Render builds the serving transaction. Existing counter objects are not
// recreated. The script is one nft transaction: readers see either the old
// ruleset or the new one, never an allow-all gap (§8.2).
func Render(m Model, state State, tcState TCState) (RenderPlan, error) {
	m = reserveUnmappedClasses(m, tcState)
	devices, assignments, denied, err := validateServing(m)
	if err != nil {
		return RenderPlan{}, err
	}
	host := mergePrefixes(m.HostV4)
	if m.Probe != nil && m.Datapath.SNAT.Is4() {
		// The node's SNAT address is host material even when HostV4 omitted it.
		host = mergePrefixes(append(host, netip.PrefixFrom(m.Datapath.SNAT, 32)))
	}
	probeSec, holdProbe, err := probeSeconds(m, state)
	if err != nil {
		return RenderPlan{}, err
	}
	resolvers := append([]netip.Addr{}, m.Resolvers...)
	sort.Slice(resolvers, func(i, j int) bool { return resolvers[i].Compare(resolvers[j]) < 0 })

	var b strings.Builder
	fmt.Fprintf(&b, "# inet fvpn generation %d %s\n", m.Generation, MeteringVersion)
	b.WriteString("# §8.2 atomic transaction. Counters that already exist are not recreated (§10.7).\n")
	fmt.Fprintf(&b, "add table %s %s\n", TableFamily, TableName)

	wantCounters := map[string]bool{}
	for _, name := range diagnosticCounters() {
		wantCounters[name] = true
	}
	for _, d := range devices {
		wantCounters[usageCounter(dirUpload, d.Binding.DeviceID)] = true
		wantCounters[usageCounter(dirDownload, d.Binding.DeviceID)] = true
		wantCounters[usageCounter(dirDNSUpload, d.Binding.DeviceID)] = true
		wantCounters[usageCounter(dirDNSDownload, d.Binding.DeviceID)] = true
	}
	if m.Probe != nil {
		wantCounters[counterProbeUp] = true
		wantCounters[counterProbeDown] = true
	}
	names := sortedKeys(wantCounters)
	for _, name := range names {
		if state.hasCounter(name) {
			continue
		}
		fmt.Fprintf(&b, "add counter %s %s %s\n", TableFamily, TableName, name)
	}

	ensureSet(&b, state, "allowed_v4", "type ipv4_addr; flags timeout;")
	ensureSet(&b, state, "denied_v4", "type ipv4_addr; flags interval; auto-merge;")
	ensureSet(&b, state, "host_v4", "type ipv4_addr; flags interval; auto-merge;")
	ensureSet(&b, state, "resolver_v4", "type ipv4_addr;")
	if m.Probe != nil {
		ensureSet(&b, state, setProbeV4, "type ipv4_addr; flags timeout;")
		if len(m.Probe.Destinations) > 0 {
			ensureSet(&b, state, setProbeDstV4, "type ipv4_addr . inet_proto . inet_service; flags timeout;")
		}
	}

	flushOrFill := func(set string, body string) {
		if state.hasSet(set) {
			fmt.Fprintf(&b, "flush set %s %s %s\n", TableFamily, TableName, set)
		}
		if body != "" {
			fmt.Fprintf(&b, "add element %s %s %s { %s }\n", TableFamily, TableName, set, body)
		}
	}
	flushOrFill("allowed_v4", allowedElements(devices))
	flushOrFill("denied_v4", prefixList(denied))
	flushOrFill("host_v4", prefixList(host))
	flushOrFill("resolver_v4", addrList(resolvers))
	if m.Probe != nil && !holdProbe {
		flushOrFill(setProbeV4, probeSourceElement(m.Probe.SourceV4.Addr(), probeSec))
		if len(m.Probe.Destinations) > 0 {
			flushOrFill(setProbeDstV4, probeDestElements(m.Probe.Destinations, probeSec))
		}
	}

	replaceChain(&b, state, "forward", fmt.Sprintf("type filter hook forward priority %d; policy drop;", HookPriorityFilter))
	replaceChain(&b, state, "input", fmt.Sprintf("type filter hook input priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "output", fmt.Sprintf("type filter hook output priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "postrouting", fmt.Sprintf("type nat hook postrouting priority %d; policy accept;", HookPrioritySNAT))

	in := m.Datapath.MemberIngress
	up := m.Datapath.Uplink
	// Unreachable markers. nft comments are at most 128 bytes, so the class
	// map is one rule per member rather than one packed comment (§8.2, §11.4).
	if err := markerRule(&b, "forward", fmt.Sprintf("fvpn_generation=%d fvpn_metering=%s", m.Generation, MeteringVersion)); err != nil {
		return RenderPlan{}, err
	}
	for _, id := range sortedIDs(assignments) {
		if err := markerRule(&b, "forward", fmt.Sprintf("fvpn_class=%s:%x", id, assignments[id])); err != nil {
			return RenderPlan{}, err
		}
	}

	// IPv6 member forwarding is blocked in v1 (§8.3). The counter is diagnostic.
	// Upload matches bind the uplink; download matches bind the uplink as ingress.
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q meta nfproto ipv6 counter name %q drop", in, up, counterV6))
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q meta nfproto ipv6 counter name %q drop", up, in, counterV6))
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ct state invalid counter name %q drop", in, up, counterInvalid))
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ct state invalid counter name %q drop", up, in, counterInvalid))

	// Lease and binding before any accept (§8.2, §16.3). An empty allow set
	// drops every upload source. A live probe source is not a member, so the
	// drop exempts @probe_v4 only while the exception is installed (§13.9).
	if m.Probe != nil {
		rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr != @allowed_v4 ip saddr != @%s counter name %q drop", in, up, setProbeV4, counterUnknown))
	} else {
		rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr != @allowed_v4 counter name %q drop", in, up, counterUnknown))
	}

	// DNS exception before the destination deny, because the resolver prefix
	// is itself inside the spec deny list (§8.1, §10.1). These counters are
	// dns_service_v1, not forward_admission_v1.
	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		prio := fmt.Sprintf("0x%08x", ClassID(d.Minor))
		dnsUp := usageCounter(dirDNSUpload, d.Binding.DeviceID)
		for _, addr := range d.Addrs {
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr %s ip daddr @resolver_v4 tcp dport 53 counter name %q meta priority set %s accept", in, up, addr, dnsUp, prio))
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr %s ip daddr @resolver_v4 udp dport 53 counter name %q meta priority set %s accept", in, up, addr, dnsUp, prio))
		}
	}
	// Probe DNS is the same hole as member DNS: the resolver prefix is inside
	// the deny list, and only dport 53 is accepted. The source element timeout
	// closes the hole (§13.9).
	if m.Probe != nil && m.Probe.ResolverDNS {
		writeProbeDNS(&b, in, up)
	}
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr @resolver_v4 counter name %q drop", in, up, counterBadDNS))
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr @denied_v4 counter name %q drop", in, up, counterDenied))
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr @host_v4 counter name %q drop", in, up, counterHost))
	// General probe accepts sit behind the deny and host drops, and before
	// any member accept. A tuple that is not in the set falls through.
	if m.Probe != nil && len(m.Probe.Destinations) > 0 {
		writeProbeTuple(&b, in, up)
	}

	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		prio := fmt.Sprintf("0x%08x", ClassID(d.Minor))
		upCounter := usageCounter(dirUpload, d.Binding.DeviceID)
		for _, addr := range d.Addrs {
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr %s counter name %q meta priority set %s accept", in, up, addr, upCounter, prio))
		}
	}

	// Download: the same allow set is checked before established/related
	// traffic is accepted, so a timeout or revocation stops an existing flow
	// (§16.3). Reverse NAT has already restored the member address (§10.2).
	if m.Probe != nil {
		rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr != @allowed_v4 ip daddr != @%s counter name %q drop", up, in, setProbeV4, counterUnkDown))
	} else {
		rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr != @allowed_v4 counter name %q drop", up, in, counterUnkDown))
	}
	rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ct state != { established, related } counter name %q drop", up, in, counterNoDown))
	if m.Probe != nil {
		writeProbeDown(&b, in, up)
	}
	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		prio := fmt.Sprintf("0x%08x", ClassID(d.Minor))
		dnsDown := usageCounter(dirDNSDownload, d.Binding.DeviceID)
		useDown := usageCounter(dirDownload, d.Binding.DeviceID)
		for _, addr := range d.Addrs {
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr @resolver_v4 ip daddr %s tcp sport 53 counter name %q meta priority set %s accept", up, in, addr, dnsDown, prio))
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr @resolver_v4 ip daddr %s udp sport 53 counter name %q meta priority set %s accept", up, in, addr, dnsDown, prio))
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr %s counter name %q meta priority set %s accept", up, in, addr, useDown, prio))
		}
	}

	// Input drops member-path packets headed at the node. Output stays
	// accept so the control channel is untouched (§8.1, §17.1).
	rule(&b, "input", fmt.Sprintf("iifname %q counter name %q drop", in, counterInput))
	rule(&b, "input", fmt.Sprintf("ip saddr 100.64.0.0/10 counter name %q drop", counterInput))
	rule(&b, "postrouting", fmt.Sprintf("oifname %q ip saddr @allowed_v4 snat ip to %s", up, m.Datapath.SNAT))
	if m.Probe != nil {
		writeProbeSNAT(&b, up, m.Datapath.SNAT.String())
	}
	// Chains are already replaced, so a deleted probe set is no longer referenced.
	writeProbeSetCleanup(&b, state, m.Probe)

	tc, err := renderTC(m, assignments, tcState)
	if err != nil {
		return RenderPlan{}, err
	}
	plan := RenderPlan{
		NFT:         b.String(),
		TC:          tc,
		Assignments: assignments,
		Timeouts:    map[string]uint32{},
	}
	admitted := 0
	for _, d := range devices {
		if d.TimeoutOK() {
			admitted++
			plan.Timeouts[d.Binding.DeviceID.String()] = d.Timeout
		}
	}
	plan.Admitted = admitted
	plan.Finals = retiredSamples(m, state, devices)
	return plan, nil
}

// RenderBaseline is the fail-closed boot ruleset. Member forwarding is off.
// Control traffic (output, and input that is not from the member path) is
// left alone (§8.2, §17.1). A non-nil Probe is the only forward accept, so a
// health check can use the candidate path while members stay blocked (§13.9).
func RenderBaseline(m Model, state State) (string, error) {
	if err := validateDatapath(m.Datapath); err != nil {
		return "", err
	}
	if m.Probe != nil {
		if err := validateProbe(m); err != nil {
			return "", err
		}
	}
	probeSec, holdProbe, err := probeSeconds(m, state)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# inet fvpn boot baseline. Member forwarding is off (§8.2).\n")
	fmt.Fprintf(&b, "add table %s %s\n", TableFamily, TableName)
	names := []string{counterBase, counterV6, counterInput}
	if m.Probe != nil {
		names = append(names, counterInvalid, counterDenied, counterHost, counterProbeUp, counterProbeDown)
		if m.Probe.ResolverDNS {
			names = append(names, counterBadDNS)
		}
	}
	for _, name := range names {
		if !state.hasCounter(name) {
			fmt.Fprintf(&b, "add counter %s %s %s\n", TableFamily, TableName, name)
		}
	}
	ensureSet(&b, state, "allowed_v4", "type ipv4_addr; flags timeout;")
	if state.hasSet("allowed_v4") {
		fmt.Fprintf(&b, "flush set %s %s allowed_v4\n", TableFamily, TableName)
	}
	if m.Probe != nil {
		denied, err := m.denyV4()
		if err != nil {
			return "", err
		}
		host := mergePrefixes(m.HostV4)
		if m.Datapath.SNAT.Is4() {
			host = mergePrefixes(append(host, netip.PrefixFrom(m.Datapath.SNAT, 32)))
		}
		resolvers := append([]netip.Addr{}, m.Resolvers...)
		sort.Slice(resolvers, func(i, j int) bool { return resolvers[i].Compare(resolvers[j]) < 0 })
		ensureSet(&b, state, "denied_v4", "type ipv4_addr; flags interval; auto-merge;")
		ensureSet(&b, state, "host_v4", "type ipv4_addr; flags interval; auto-merge;")
		ensureSet(&b, state, setProbeV4, "type ipv4_addr; flags timeout;")
		if m.Probe.ResolverDNS {
			ensureSet(&b, state, "resolver_v4", "type ipv4_addr;")
		}
		if len(m.Probe.Destinations) > 0 {
			ensureSet(&b, state, setProbeDstV4, "type ipv4_addr . inet_proto . inet_service; flags timeout;")
		}
		flushFill := func(set, body string) {
			if state.hasSet(set) {
				fmt.Fprintf(&b, "flush set %s %s %s\n", TableFamily, TableName, set)
			}
			if body != "" {
				fmt.Fprintf(&b, "add element %s %s %s { %s }\n", TableFamily, TableName, set, body)
			}
		}
		flushFill("denied_v4", prefixList(denied))
		flushFill("host_v4", prefixList(host))
		if m.Probe.ResolverDNS {
			flushFill("resolver_v4", addrList(resolvers))
		}
		if !holdProbe {
			flushFill(setProbeV4, probeSourceElement(m.Probe.SourceV4.Addr(), probeSec))
			if len(m.Probe.Destinations) > 0 {
				flushFill(setProbeDstV4, probeDestElements(m.Probe.Destinations, probeSec))
			}
		}
	}
	replaceChain(&b, state, "forward", fmt.Sprintf("type filter hook forward priority %d; policy drop;", HookPriorityFilter))
	replaceChain(&b, state, "input", fmt.Sprintf("type filter hook input priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "output", fmt.Sprintf("type filter hook output priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "postrouting", fmt.Sprintf("type nat hook postrouting priority %d; policy accept;", HookPrioritySNAT))
	if err := markerRule(&b, "forward", "fvpn_generation=0 fvpn_baseline=1 fvpn_metering="+MeteringVersion); err != nil {
		return "", err
	}
	in := m.Datapath.MemberIngress
	up := m.Datapath.Uplink
	if m.Probe != nil {
		// Drops stay unbound to the uplink so the baseline does not narrow.
		// The probe accepts name both interfaces. DNS is before the deny drop
		// because the resolver prefix is inside that set. The tuple accept is
		// after the deny and host drops (§13.9).
		rule(&b, "forward", fmt.Sprintf("iifname %q meta nfproto ipv6 counter name %q drop", in, counterV6))
		rule(&b, "forward", fmt.Sprintf("oifname %q meta nfproto ipv6 counter name %q drop", in, counterV6))
		rule(&b, "forward", fmt.Sprintf("iifname %q ct state invalid counter name %q drop", in, counterInvalid))
		rule(&b, "forward", fmt.Sprintf("oifname %q ct state invalid counter name %q drop", in, counterInvalid))
		if m.Probe.ResolverDNS {
			writeProbeDNS(&b, in, up)
			rule(&b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr @resolver_v4 counter name %q drop", in, up, counterBadDNS))
		}
		rule(&b, "forward", fmt.Sprintf("iifname %q ip daddr @denied_v4 counter name %q drop", in, counterDenied))
		rule(&b, "forward", fmt.Sprintf("iifname %q ip daddr @host_v4 counter name %q drop", in, counterHost))
		if len(m.Probe.Destinations) > 0 {
			writeProbeTuple(&b, in, up)
		}
		writeProbeDown(&b, in, up)
	}
	rule(&b, "forward", fmt.Sprintf("iifname %q counter name %q drop", in, counterBase))
	rule(&b, "forward", fmt.Sprintf("oifname %q counter name %q drop", in, counterBase))
	if m.Probe == nil {
		rule(&b, "forward", fmt.Sprintf("iifname %q meta nfproto ipv6 counter name %q drop", in, counterV6))
		rule(&b, "forward", fmt.Sprintf("oifname %q meta nfproto ipv6 counter name %q drop", in, counterV6))
	}
	rule(&b, "input", fmt.Sprintf("iifname %q counter name %q drop", in, counterInput))
	rule(&b, "input", fmt.Sprintf("ip saddr 100.64.0.0/10 counter name %q drop", counterInput))
	if m.Probe != nil {
		writeProbeSNAT(&b, up, m.Datapath.SNAT.String())
	}
	writeProbeSetCleanup(&b, state, m.Probe)
	return b.String(), nil
}

const (
	setProbeV4    = "probe_v4"
	setProbeDstV4 = "probe_dst_v4"
)

// probeSeconds is the kernel timeout for this render. hold is true when the
// probe sets are already installed and this render must not replace their
// elements: a fresh ProbeMaxSeconds window would extend the original
// ExpiresAt, and State has no remaining lifetime to cap against (§13.9).
func probeSeconds(m Model, state State) (sec uint32, hold bool, err error) {
	if m.Probe == nil {
		return 0, false, nil
	}
	got, ok := probeTimeout(m.Probe.ExpiresAt, m.Now)
	if !ok || got > ProbeMaxSeconds {
		return 0, false, fmt.Errorf("probe expiration is not admissible")
	}
	sec = got
	if m.Probe.TimeoutCeilSeconds > 0 && uint64(sec) > m.Probe.TimeoutCeilSeconds {
		sec = uint32(m.Probe.TimeoutCeilSeconds)
	}
	live := state.hasSet(setProbeV4) || state.hasSet(setProbeDstV4)
	if live && state.ProbeRemain > 0 && sec > state.ProbeRemain {
		sec = state.ProbeRemain
	}
	if sec == 0 {
		return 0, false, fmt.Errorf("probe expiration is not admissible")
	}
	// Sets already exist and nothing records how much of the original window
	// is left. Do not flush them and install another full ProbeMaxSeconds.
	hold = live && state.ProbeRemain == 0 && m.Probe.TimeoutCeilSeconds == 0 && sec >= ProbeMaxSeconds
	return sec, hold, nil
}

func writeProbeDNS(b *strings.Builder, in, up string) {
	rule(b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr @%s ip daddr @resolver_v4 tcp dport 53 counter name %q accept", in, up, setProbeV4, counterProbeUp))
	rule(b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr @%s ip daddr @resolver_v4 udp dport 53 counter name %q accept", in, up, setProbeV4, counterProbeUp))
}

func writeProbeTuple(b *strings.Builder, in, up string) {
	rule(b, "forward", fmt.Sprintf("iifname %q oifname %q ip saddr @%s ip daddr . meta l4proto . th dport @%s counter name %q accept", in, up, setProbeV4, setProbeDstV4, counterProbeUp))
}

func writeProbeDown(b *strings.Builder, in, up string) {
	rule(b, "forward", fmt.Sprintf("iifname %q oifname %q ip daddr @%s ct state established,related counter name %q accept", up, in, setProbeV4, counterProbeDown))
}

func writeProbeSNAT(b *strings.Builder, up, snat string) {
	rule(b, "postrouting", fmt.Sprintf("oifname %q ip saddr @%s snat ip to %s", up, setProbeV4, snat))
}

func writeProbeSetCleanup(b *strings.Builder, state State, p *ProbeException) {
	if p == nil {
		deleteSet(b, state, setProbeV4)
	}
	if p == nil || len(p.Destinations) == 0 {
		deleteSet(b, state, setProbeDstV4)
	}
}

func deleteSet(b *strings.Builder, state State, name string) {
	if state.hasSet(name) {
		fmt.Fprintf(b, "delete set %s %s %s\n", TableFamily, TableName, name)
	}
}

func probeSourceElement(addr netip.Addr, sec uint32) string {
	return fmt.Sprintf("%s timeout %ds", addr, sec)
}

func probeDestElements(dsts []ProbeDestination, sec uint32) string {
	dsts = append([]ProbeDestination(nil), dsts...)
	sort.Slice(dsts, func(i, j int) bool {
		if c := dsts[i].AddrV4.Addr().Compare(dsts[j].AddrV4.Addr()); c != 0 {
			return c < 0
		}
		if dsts[i].Proto != dsts[j].Proto {
			return dsts[i].Proto < dsts[j].Proto
		}
		return dsts[i].Port < dsts[j].Port
	})
	var parts []string
	seen := map[string]bool{}
	for _, d := range dsts {
		key := fmt.Sprintf("%s . %s . %d", d.AddrV4.Addr(), d.Proto, d.Port)
		if seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, fmt.Sprintf("%s timeout %ds", key, sec))
	}
	return strings.Join(parts, ", ")
}

func diagnosticCounters() []string {
	return []string{
		counterV6, counterUnknown, counterUnkDown, counterDenied, counterHost,
		counterBadDNS, counterInvalid, counterInput, counterBase, counterNoDown,
	}
}

func ensureSet(b *strings.Builder, state State, name, spec string) {
	if state.hasSet(name) {
		return
	}
	fmt.Fprintf(b, "add set %s %s %s { %s }\n", TableFamily, TableName, name, spec)
}

func replaceChain(b *strings.Builder, state State, name, spec string) {
	if state.hasChain(name) {
		fmt.Fprintf(b, "delete chain %s %s %s\n", TableFamily, TableName, name)
	}
	fmt.Fprintf(b, "add chain %s %s %s { %s }\n", TableFamily, TableName, name, spec)
}

func rule(b *strings.Builder, chain, body string) {
	fmt.Fprintf(b, "add rule %s %s %s %s\n", TableFamily, TableName, chain, body)
}

// nftCommentMax is the userspace limit (nft 1.0.9: "128 characters maximum").
const nftCommentMax = 128

func markerRule(b *strings.Builder, chain, comment string) error {
	if len(comment) > nftCommentMax {
		return fmt.Errorf("nft comment is %d bytes, maximum is %d", len(comment), nftCommentMax)
	}
	rule(b, chain, fmt.Sprintf("ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment %q", comment))
	return nil
}

func sortedIDs(assignments map[uuid.UUID]uint16) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(assignments))
	for id := range assignments {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func allowedElements(devices []admittedDevice) string {
	var parts []string
	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		for _, addr := range d.Addrs {
			parts = append(parts, fmt.Sprintf("%s timeout %ds", addr, d.Timeout))
		}
	}
	return strings.Join(parts, ", ")
}

func prefixList(p []netip.Prefix) string {
	parts := make([]string, len(p))
	for i, x := range p {
		parts[i] = x.String()
	}
	return strings.Join(parts, ", ")
}

func addrList(a []netip.Addr) string {
	parts := make([]string, len(a))
	for i, x := range a {
		parts[i] = x.String()
	}
	return strings.Join(parts, ", ")
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
