package enforce

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// exactRuleExpressions compares the entire statement sequence, not a subset
// of parsed predicates. Object key order is immaterial; extra fields are not.
func exactRuleExpressions(raw, expected string) bool {
	var got, want any
	if json.Unmarshal([]byte(raw), &got) != nil || json.Unmarshal([]byte(expected), &want) != nil {
		return false
	}
	return reflect.DeepEqual(got, want)
}

func mssExpressions(in, out, field, addr string) string {
	return fmt.Sprintf(`[
 {"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":%q}},
 {"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":%q}},
 {"match":{"op":"==","left":{"payload":{"protocol":"ip","field":%q}},"right":%q}},
 {"match":{"op":"==","left":{"&":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},
 {"match":{"op":">","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":%d}},
 {"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":%d}}
 ]`, in, out, field, addr, memberTCPMSS, memberTCPMSS)
}

func verifyMSSClamps(rs Ruleset, m Model, devices []admittedDevice) error {
	expected := 0
	for _, d := range devices {
		if d.TimeoutOK() {
			expected += 2 * len(d.Addrs)
		}
	}
	found := 0
	rules := rs.Chains["forward"].Rules
	seen := map[string]bool{}
	for i, r := range rules {
		if !r.TCPMSS {
			continue
		}
		if i+1 >= len(rules) {
			return checkf(ReasonIsolation, "MSS clamp has no following member accept")
		}
		next := rules[i+1]
		kind := servingAcceptKind(next, m)
		in, out, field, addrs := m.Datapath.MemberIngress, m.Datapath.Uplink, "saddr", next.Saddrs
		if kind == "download" {
			in, out, field, addrs = out, in, "daddr", next.Daddrs
		}
		if (kind != "upload" && kind != "download") || !oneHost(addrs) {
			return checkf(ReasonIsolation, "MSS clamp does not immediately precede a general member accept")
		}
		key := kind + ":" + addrs[0].String()
		if seen[key] || !exactRuleExpressions(r.Raw, mssExpressions(in, out, field, addrs[0].Addr().String())) {
			return checkf(ReasonIsolation, "MSS clamp differs from the rendered shape")
		}
		seen[key] = true
		found++
	}
	if found != expected {
		return checkf(ReasonIsolation, "MSS clamps=%d want %d", found, expected)
	}
	return nil
}
