package enforce

import (
	"net/netip"
	"sort"
)

// SpecDeniedV4 is the v1 destination deny list (§8.1). It is every IANA
// IPv4 special-purpose block whose Globally Reachable cell is not True,
// plus multicast 224.0.0.0/4. 192.88.99.0/24 (deprecated 6to4 relay anycast,
// RFC 7526) is included: that cell is blank because the allocation ended in
// 2015. 192.31.196.0/24, 192.52.193.0/24 and 192.175.48.0/24 are globally
// reachable and are not listed. The globally reachable anycasts 192.0.0.9/32
// and 192.0.0.10/32 stay covered by their parent 192.0.0.0/24, which is not
// globally reachable; this list does not punch that hole. A lab that uses
// one of these ranges as its simulated Internet must pass an explicit
// DeniedV4 that omits only that simulated range. Production callers leave
// DeniedV4 nil and get this list.
func SpecDeniedV4() []netip.Prefix {
	return mustPrefixes(
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
	)
}

// Without returns a copy of list that does not contain any prefix equal to
// one in drop. It does not punch holes inside a wider prefix.
func Without(list []netip.Prefix, drop ...netip.Prefix) []netip.Prefix {
	skip := map[netip.Prefix]bool{}
	for _, p := range drop {
		skip[p.Masked()] = true
	}
	out := make([]netip.Prefix, 0, len(list))
	for _, p := range list {
		p = p.Masked()
		if !skip[p] {
			out = append(out, p)
		}
	}
	return out
}

func mustPrefixes(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, netip.MustParsePrefix(c))
	}
	return out
}

func mergePrefixes(in []netip.Prefix) []netip.Prefix {
	seen := map[netip.Prefix]bool{}
	out := make([]netip.Prefix, 0, len(in))
	for _, p := range in {
		if !p.IsValid() {
			continue
		}
		p = p.Masked()
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return out
}
