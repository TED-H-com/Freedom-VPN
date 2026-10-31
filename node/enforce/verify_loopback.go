package enforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
)

// LoopbackScopedClass is the read-back class of a foreign rule whose native
// match restricts the packet to loopback. Forwarded member traffic does not
// carry those addresses.
const LoopbackScopedClass = "loopback_scoped"

// ForeignEvidence is one loopback-scoped rule from an nft -j ruleset.
// Match is the native expression that proves the scope, for example
// "ip saddr 127.0.0.11".
type ForeignEvidence struct {
	Class     string
	Table     string
	Family    string
	Chain     string
	Hook      string
	Handle    uint64
	HasHandle bool
	Match     string
}

var (
	loopbackV4 = netip.MustParsePrefix("127.0.0.0/8")
	loopbackV6 = netip.MustParseAddr("::1")
)

// LoopbackEvidence lists every foreign rule whose native match proves the
// packet is restricted to loopback. VerifyServing uses the same proof and
// does not fail closed on those rules. Hook is the base-chain hook that
// reaches the rule; a regular chain takes the hook of the chain that jumps
// to it.
func LoopbackEvidence(rs Ruleset) []ForeignEvidence {
	var out []ForeignEvidence
	for i := range rs.Foreign {
		ch := &rs.Foreign[i]
		hook := foreignHook(rs.Foreign, ch)
		for _, rule := range ch.Rules {
			match, ok := loopbackProof(rule)
			if !ok {
				continue
			}
			out = append(out, ForeignEvidence{
				Class:     LoopbackScopedClass,
				Table:     ch.Table,
				Family:    ch.Family,
				Chain:     ch.Name,
				Hook:      hook,
				Handle:    rule.Handle,
				HasHandle: rule.HasHandle,
				Match:     match,
			})
		}
	}
	return out
}

func foreignHook(chains []ForeignChain, ch *ForeignChain) string {
	if ch.Hook != "" {
		return ch.Hook
	}
	return hookFrom(chains, ch.Family, ch.Table, ch.Name, map[string]bool{})
}

func hookFrom(chains []ForeignChain, family, table, name string, seen map[string]bool) string {
	id := foreignID(family, table, name)
	if seen[id] {
		return ""
	}
	seen[id] = true
	for i := range chains {
		other := &chains[i]
		if other.Family != family || other.Table != table || other.Name == name {
			continue
		}
		if !chainJumpsTo(other, name) {
			continue
		}
		if other.Hook != "" {
			return other.Hook
		}
		if h := hookFrom(chains, family, table, other.Name, seen); h != "" {
			return h
		}
	}
	return ""
}

func chainJumpsTo(ch *ForeignChain, name string) bool {
	for _, rule := range ch.Rules {
		for _, st := range rule.Stmts {
			if (st.Key == "jump" || st.Key == "goto") && st.Target == name {
				return true
			}
		}
	}
	return false
}

// loopbackProof reports the first native match that restricts this rule to
// loopback. An xt match is opaque and is ignored. meta l4proto and port
// matches are ignored: they neither prove scope nor cancel a proof.
func loopbackProof(rule ForeignRule) (string, bool) {
	for _, st := range rule.Stmts {
		if st.Key != "match" {
			continue
		}
		if match, ok := provingMatch(st.Body); ok {
			return match, true
		}
	}
	return "", false
}

func provingMatch(body json.RawMessage) (string, bool) {
	var m struct {
		Op    string          `json:"op"`
		Left  json.RawMessage `json:"left"`
		Right json.RawMessage `json:"right"`
	}
	if json.Unmarshal(body, &m) != nil || !opIsEq(m.Op) {
		return "", false
	}
	var left struct {
		Payload struct {
			Protocol string `json:"protocol"`
			Field    string `json:"field"`
		} `json:"payload"`
	}
	if json.Unmarshal(m.Left, &left) != nil {
		return "", false
	}
	proto := left.Payload.Protocol
	field := left.Payload.Field
	if (proto != "ip" && proto != "ip6") || (field != "saddr" && field != "daddr") {
		return "", false
	}
	desc, ok := loopbackRight(proto, m.Right)
	if !ok {
		return "", false
	}
	return proto + " " + field + " " + desc, true
}

func loopbackRight(family string, raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return loopbackString(family, s)
	}
	if desc, ok := loopbackPrefix(family, raw); ok {
		return desc, true
	}
	if desc, ok := loopbackRange(family, raw); ok {
		return desc, true
	}
	if desc, ok := loopbackSet(family, raw); ok {
		return desc, true
	}
	return "", false
}

func loopbackString(family, s string) (string, bool) {
	if s == "" || strings.HasPrefix(s, "@") {
		return "", false
	}
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil || !prefixInside(family, p) {
			return "", false
		}
		return p.Masked().String(), true
	}
	addr, err := netip.ParseAddr(s)
	if err != nil || !addrInside(family, addr) {
		return "", false
	}
	return addr.String(), true
}

func loopbackPrefix(family string, raw json.RawMessage) (string, bool) {
	var body struct {
		Prefix *struct {
			Addr string `json:"addr"`
			Len  *int   `json:"len"`
		} `json:"prefix"`
	}
	if json.Unmarshal(raw, &body) != nil || body.Prefix == nil || body.Prefix.Len == nil || body.Prefix.Addr == "" {
		return "", false
	}
	p, err := netip.ParsePrefix(fmt.Sprintf("%s/%d", body.Prefix.Addr, *body.Prefix.Len))
	if err != nil || !prefixInside(family, p) {
		return "", false
	}
	return p.Masked().String(), true
}

func loopbackRange(family string, raw json.RawMessage) (string, bool) {
	var body struct {
		Range json.RawMessage `json:"range"`
	}
	if json.Unmarshal(raw, &body) != nil || len(bytes.TrimSpace(body.Range)) == 0 {
		return "", false
	}
	var pair []string
	if json.Unmarshal(body.Range, &pair) == nil && len(pair) == 2 {
		return rangeEnds(family, pair[0], pair[1])
	}
	var obj struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if json.Unmarshal(body.Range, &obj) == nil && obj.Start != "" && obj.End != "" {
		return rangeEnds(family, obj.Start, obj.End)
	}
	return "", false
}

func rangeEnds(family, start, end string) (string, bool) {
	a, aerr := netip.ParseAddr(start)
	b, berr := netip.ParseAddr(end)
	if aerr != nil || berr != nil || !addrInside(family, a) || !addrInside(family, b) || a.Compare(b) > 0 {
		return "", false
	}
	return a.String() + "-" + b.String(), true
}

func loopbackSet(family string, raw json.RawMessage) (string, bool) {
	var body struct {
		Set json.RawMessage `json:"set"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return "", false
	}
	elemsRaw := bytes.TrimSpace(body.Set)
	if len(elemsRaw) == 0 || elemsRaw[0] != '[' {
		return "", false
	}
	var elems []json.RawMessage
	if json.Unmarshal(elemsRaw, &elems) != nil || len(elems) == 0 {
		return "", false
	}
	parts := make([]string, 0, len(elems))
	for _, el := range elems {
		desc, ok := loopbackRight(family, el)
		if !ok {
			return "", false
		}
		parts = append(parts, desc)
	}
	return "{" + strings.Join(parts, ", ") + "}", true
}

func addrInside(family string, addr netip.Addr) bool {
	switch family {
	case "ip":
		return addr.Is4() && loopbackV4.Contains(addr)
	case "ip6":
		return addr.Is6() && !addr.Is4In6() && addr == loopbackV6
	default:
		return false
	}
}

func prefixInside(family string, p netip.Prefix) bool {
	if !p.IsValid() {
		return false
	}
	p = p.Masked()
	switch family {
	case "ip":
		if !p.Addr().Is4() || p.Bits() < loopbackV4.Bits() {
			return false
		}
		return loopbackV4.Contains(p.Addr())
	case "ip6":
		return p.Addr().Is6() && !p.Addr().Is4In6() && p.Bits() == 128 && p.Addr() == loopbackV6
	default:
		return false
	}
}
