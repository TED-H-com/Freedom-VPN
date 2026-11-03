package enforce

import (
	"encoding/json"
	"fmt"
)

func pmtuExpressions(in, out, set, addr, counter string, priority uint32) string {
	return fmt.Sprintf(`[
 {"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":%q}},
 {"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":%q}},
 {"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":%q}},
 {"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":%q}},
 {"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},
 {"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},
 {"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},
 {"counter":%q},
 {"mangle":{"key":{"meta":{"key":"priority"}},"value":%d}},
 {"accept":null}
 ]`, in, out, "@"+set, addr, counter, priority)
}

// nft emits class IDs as hexadecimal "major:minor" or a JSON integer. Counter
// references have string and {name:...} forms. Normalize only these exact forms;
// preserve every other key, operator, match and statement for full comparison.
func normalizedPMTU(raw string) string {
	var expr []map[string]any
	if json.Unmarshal([]byte(raw), &expr) != nil {
		return ""
	}
	for _, st := range expr {
		if c, ok := st["counter"].(map[string]any); ok && len(c) == 1 {
			if n, ok := c["name"].(string); ok {
				st["counter"] = n
			}
		}
		if m, ok := st["mangle"].(map[string]any); ok {
			if value, ok := m["value"].(string); ok {
				if n, ok := parseClassID(value); ok {
					m["value"] = n
				}
			}
		}
	}
	b, err := json.Marshal(expr)
	if err != nil {
		return ""
	}
	return string(b)
}

func pmtuSource(r RuleView, m Model) string {
	if r.Tainted || r.Verdict != "accept" || !r.ICMPMatch || !oneHost(r.Daddrs) || len(r.Prios) != 1 || !deviceCounter(r.Counter, "d_") {
		return ""
	}
	for _, set := range []string{"resolver_v4", "denied_v4", "host_v4"} {
		want := pmtuExpressions(m.Datapath.Uplink, m.Datapath.MemberIngress, set, r.Daddrs[0].Addr().String(), r.Counter, r.Prios[0])
		if exactRuleExpressions(normalizedPMTU(r.Raw), want) {
			return set
		}
	}
	return ""
}

func verifyPMTUExceptions(rs Ruleset, m Model, devices []admittedDevice) error {
	var want []string
	for _, d := range devices {
		if !d.TimeoutOK() {
			continue
		}
		for _, addr := range d.Addrs {
			for _, set := range []string{"resolver_v4", "denied_v4", "host_v4"} {
				want = append(want, pmtuExpressions(m.Datapath.Uplink, m.Datapath.MemberIngress, set, addr.String(), usageCounter(dirDownload, d.Binding.DeviceID), ClassID(d.Minor)))
			}
		}
	}
	rules := rs.Chains["forward"].Rules
	firstDeny, found := -1, 0
	for i, r := range rules {
		if firstDeny < 0 && downloadSourceDrop(r, m, "resolver_v4") {
			firstDeny = i
		}
		if pmtuSource(r, m) != "" {
			found++
		}
	}
	if found != len(want) || firstDeny < len(want) {
		return checkf(ReasonIsolation, "PMTU exceptions=%d want %d before source denials", found, len(want))
	}
	for i, w := range want {
		if !exactRuleExpressions(normalizedPMTU(rules[firstDeny-len(want)+i].Raw), w) {
			return checkf(ReasonIsolation, "PMTU exception differs in match, meter, classification or placement")
		}
	}
	return nil
}
