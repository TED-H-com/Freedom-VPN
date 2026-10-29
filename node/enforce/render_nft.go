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
	devices, assignments, denied, err := validateServing(m)
	if err != nil {
		return RenderPlan{}, err
	}
	host := mergePrefixes(m.HostV4)
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

	replaceChain(&b, state, "forward", fmt.Sprintf("type filter hook forward priority %d; policy drop;", HookPriorityFilter))
	replaceChain(&b, state, "input", fmt.Sprintf("type filter hook input priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "output", fmt.Sprintf("type filter hook output priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "postrouting", fmt.Sprintf("type nat hook postrouting priority %d; policy accept;", HookPrioritySNAT))

	in := m.Datapath.MemberIngress
	up := m.Datapath.Uplink
	comment := fmt.Sprintf("fvpn_generation=%d fvpn_metering=%s", m.Generation, MeteringVersion)
	// Unreachable marker so the generation is visible without accepting traffic.
	rule(&b, "forward", fmt.Sprintf("ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment %q", comment))

	// IPv6 member forwarding is blocked in v1 (§8.3). The counter is diagnostic.
	rule(&b, "forward", fmt.Sprintf("iifname %q meta nfproto ipv6 counter name %q drop", in, counterV6))
	rule(&b, "forward", fmt.Sprintf("oifname %q meta nfproto ipv6 counter name %q drop", in, counterV6))
	rule(&b, "forward", fmt.Sprintf("iifname %q ct state invalid counter name %q drop", in, counterInvalid))
	rule(&b, "forward", fmt.Sprintf("oifname %q ct state invalid counter name %q drop", in, counterInvalid))

	// Lease and binding before any accept (§8.2, §16.3). An empty allow set
	// drops every upload source.
	rule(&b, "forward", fmt.Sprintf("iifname %q ip saddr != @allowed_v4 counter name %q drop", in, counterUnknown))

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
			rule(&b, "forward", fmt.Sprintf("iifname %q ip saddr %s ip daddr @resolver_v4 tcp dport 53 counter name %q meta priority set %s accept", in, addr, dnsUp, prio))
			rule(&b, "forward", fmt.Sprintf("iifname %q ip saddr %s ip daddr @resolver_v4 udp dport 53 counter name %q meta priority set %s accept", in, addr, dnsUp, prio))
		}
	}
	rule(&b, "forward", fmt.Sprintf("iifname %q ip daddr @resolver_v4 counter name %q drop", in, counterBadDNS))
	rule(&b, "forward", fmt.Sprintf("iifname %q ip daddr @denied_v4 counter name %q drop", in, counterDenied))
	rule(&b, "forward", fmt.Sprintf("iifname %q ip daddr @host_v4 counter name %q drop", in, counterHost))

	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		prio := fmt.Sprintf("0x%08x", ClassID(d.Minor))
		upCounter := usageCounter(dirUpload, d.Binding.DeviceID)
		for _, addr := range d.Addrs {
			rule(&b, "forward", fmt.Sprintf("iifname %q ip saddr %s counter name %q meta priority set %s accept", in, addr, upCounter, prio))
		}
	}

	// Download: the same allow set is checked before established/related
	// traffic is accepted, so a timeout or revocation stops an existing flow
	// (§16.3). Reverse NAT has already restored the member address (§10.2).
	rule(&b, "forward", fmt.Sprintf("oifname %q ip daddr != @allowed_v4 counter name %q drop", in, counterUnkDown))
	rule(&b, "forward", fmt.Sprintf("oifname %q ct state != { established, related } counter name %q drop", in, counterNoDown))
	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		prio := fmt.Sprintf("0x%08x", ClassID(d.Minor))
		dnsDown := usageCounter(dirDNSDownload, d.Binding.DeviceID)
		useDown := usageCounter(dirDownload, d.Binding.DeviceID)
		for _, addr := range d.Addrs {
			rule(&b, "forward", fmt.Sprintf("oifname %q ip saddr @resolver_v4 ip daddr %s tcp sport 53 counter name %q meta priority set %s accept", in, addr, dnsDown, prio))
			rule(&b, "forward", fmt.Sprintf("oifname %q ip saddr @resolver_v4 ip daddr %s udp sport 53 counter name %q meta priority set %s accept", in, addr, dnsDown, prio))
			rule(&b, "forward", fmt.Sprintf("oifname %q ip daddr %s counter name %q meta priority set %s accept", in, addr, useDown, prio))
		}
	}

	// Input drops member-path packets headed at the node. Output stays
	// accept so the control channel is untouched (§8.1, §17.1).
	rule(&b, "input", fmt.Sprintf("iifname %q counter name %q drop", in, counterInput))
	rule(&b, "input", fmt.Sprintf("ip saddr 100.64.0.0/10 counter name %q drop", counterInput))
	rule(&b, "postrouting", fmt.Sprintf("oifname %q ip saddr @allowed_v4 snat ip to %s", up, m.Datapath.SNAT))

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
// left alone (§8.2, §17.1).
func RenderBaseline(dp Datapath, state State) (string, error) {
	if err := validateDatapath(dp); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# inet fvpn boot baseline. Member forwarding is off (§8.2).\n")
	fmt.Fprintf(&b, "add table %s %s\n", TableFamily, TableName)
	for _, name := range []string{counterBase, counterV6, counterInput} {
		if !state.hasCounter(name) {
			fmt.Fprintf(&b, "add counter %s %s %s\n", TableFamily, TableName, name)
		}
	}
	ensureSet(&b, state, "allowed_v4", "type ipv4_addr; flags timeout;")
	if state.hasSet("allowed_v4") {
		fmt.Fprintf(&b, "flush set %s %s allowed_v4\n", TableFamily, TableName)
	}
	replaceChain(&b, state, "forward", fmt.Sprintf("type filter hook forward priority %d; policy drop;", HookPriorityFilter))
	replaceChain(&b, state, "input", fmt.Sprintf("type filter hook input priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "output", fmt.Sprintf("type filter hook output priority %d; policy accept;", HookPriorityFilter))
	replaceChain(&b, state, "postrouting", fmt.Sprintf("type nat hook postrouting priority %d; policy accept;", HookPrioritySNAT))
	comment := "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=" + MeteringVersion
	rule(&b, "forward", fmt.Sprintf("ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment %q", comment))
	rule(&b, "forward", fmt.Sprintf("iifname %q counter name %q drop", dp.MemberIngress, counterBase))
	rule(&b, "forward", fmt.Sprintf("oifname %q counter name %q drop", dp.MemberIngress, counterBase))
	rule(&b, "forward", fmt.Sprintf("iifname %q meta nfproto ipv6 counter name %q drop", dp.MemberIngress, counterV6))
	rule(&b, "forward", fmt.Sprintf("oifname %q meta nfproto ipv6 counter name %q drop", dp.MemberIngress, counterV6))
	rule(&b, "input", fmt.Sprintf("iifname %q counter name %q drop", dp.MemberIngress, counterInput))
	rule(&b, "input", fmt.Sprintf("ip saddr 100.64.0.0/10 counter name %q drop", counterInput))
	return b.String(), nil
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
