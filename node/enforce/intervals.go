package enforce

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/netip"
	"slices"
	"sort"
)

// ipv4Range is closed at both ends. Compare coverage, not prefix spelling:
// nft interval/auto-merge can combine overlapping and adjacent model prefixes.
type ipv4Range struct{ first, last uint32 }

func prefixRange(p netip.Prefix) (ipv4Range, bool) {
	if !p.IsValid() || !p.Addr().Is4() {
		return ipv4Range{}, false
	}
	first := ipv4Number(p.Masked().Addr())
	return ipv4Range{first, first | (^uint32(0) >> p.Bits())}, true
}

func ipv4Number(a netip.Addr) uint32 {
	b := a.As4()
	return binary.BigEndian.Uint32(b[:])
}

func canonicalRanges(ranges []ipv4Range) []ipv4Range {
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].first != ranges[j].first {
			return ranges[i].first < ranges[j].first
		}
		return ranges[i].last < ranges[j].last
	})
	out := ranges[:0]
	for _, r := range ranges {
		if len(out) != 0 && uint64(r.first) <= uint64(out[len(out)-1].last)+1 {
			out[len(out)-1].last = max(out[len(out)-1].last, r.last)
		} else {
			out = append(out, r)
		}
	}
	return out
}

// Only these static interval sets get this parser. Allow, resolver and probe
// elements retain their existing host/tuple and lifetime validation.
func parseIntervalElems(raw json.RawMessage) ([]ipv4Range, bool) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false
	}
	var elems []json.RawMessage
	if json.Unmarshal(raw, &elems) != nil {
		return nil, true
	}
	out := make([]ipv4Range, 0, len(elems))
	for _, elem := range elems {
		r, ok := parseIntervalElem(elem)
		if !ok {
			return nil, true
		}
		out = append(out, r)
	}
	return canonicalRanges(out), false
}

func parseIntervalElem(raw json.RawMessage) (ipv4Range, bool) {
	var addr string
	if json.Unmarshal(raw, &addr) == nil {
		a, err := netip.ParseAddr(addr)
		if err == nil && a.Is4() {
			n := ipv4Number(a)
			return ipv4Range{n, n}, true
		}
		// Some emulators emit /32 strings; wider prefixes use nft's prefix object.
		p, err := netip.ParsePrefix(addr)
		if err == nil && p.Bits() == 32 {
			return prefixRange(p)
		}
		return ipv4Range{}, false
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil || len(obj) != 1 {
		return ipv4Range{}, false
	}
	// Accept only unadorned value wrappers: timeout/expires (even zero/null),
	// comments, expressions and other metadata were not rendered and taint it.
	for _, key := range []string{"elem", "val"} {
		if inner, ok := obj[key]; ok {
			return parseIntervalElem(inner)
		}
	}
	if raw, ok := obj["prefix"]; ok {
		var p map[string]json.RawMessage
		if json.Unmarshal(raw, &p) != nil || len(p) != 2 {
			return ipv4Range{}, false
		}
		var addr string
		var bits int
		if json.Unmarshal(p["addr"], &addr) != nil || json.Unmarshal(p["len"], &bits) != nil || bytes.Equal(p["len"], []byte("null")) {
			return ipv4Range{}, false
		}
		a, err := netip.ParseAddr(addr)
		if err != nil || !a.Is4() || bits < 0 || bits > 32 {
			return ipv4Range{}, false
		}
		return prefixRange(netip.PrefixFrom(a, bits))
	}
	if raw, ok := obj["range"]; ok {
		var pair []string
		if json.Unmarshal(raw, &pair) != nil || len(pair) != 2 {
			return ipv4Range{}, false
		}
		a, errA := netip.ParseAddr(pair[0])
		b, errB := netip.ParseAddr(pair[1])
		if errA != nil || errB != nil || !a.Is4() || !b.Is4() || a.Compare(b) > 0 {
			return ipv4Range{}, false
		}
		return ipv4Range{ipv4Number(a), ipv4Number(b)}, true
	}
	return ipv4Range{}, false
}

func verifyIsolationSets(rs Ruleset, m Model, present bool) error {
	if !present {
		if rs.Sets["denied_v4"] || rs.Sets["host_v4"] {
			return checkf(ReasonIsolation, "bare baseline contains a denial set")
		}
		return nil
	}
	// Baselines do not use validateServing; reject invalid model input here too.
	for _, ps := range [][]netip.Prefix{m.DeniedV4, m.ExtraDeniedV4, m.HostV4} {
		if err := validateV4Prefixes(ps, "isolation"); err != nil {
			return checkf(ReasonIsolation, "invalid isolation model")
		}
	}
	denied, err := m.denyV4()
	if err != nil {
		return checkf(ReasonIsolation, "invalid isolation model")
	}
	host := append(append([]netip.Prefix{}, m.HostV4...), netip.PrefixFrom(m.Datapath.SNAT, 32))
	for _, set := range []struct {
		name     string
		prefixes []netip.Prefix
	}{{"denied_v4", denied}, {"host_v4", host}} {
		if !rs.Sets[set.name] || rs.Tainted[set.name] {
			return checkf(ReasonIsolation, "set %s is missing or tainted", set.name)
		}
		expected := make([]ipv4Range, 0, len(set.prefixes))
		for _, p := range set.prefixes {
			r, ok := prefixRange(p)
			if !ok {
				return checkf(ReasonIsolation, "invalid isolation model")
			}
			expected = append(expected, r)
		}
		if !slices.Equal(canonicalRanges(expected), rs.intervals[set.name]) {
			return checkf(ReasonIsolation, "set %s coverage differs from the model", set.name)
		}
	}
	return nil
}
