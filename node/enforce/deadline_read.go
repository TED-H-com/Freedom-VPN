package enforce

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DeadlineReadEvidence is the extra precision read required by nft 1.0.9:
// JSON rounds timeout/expires down to seconds. No packet selectors are retained.
type DeadlineReadEvidence struct {
	Start    time.Duration `json:"start_ns,string"`
	End      time.Duration `json:"end_ns,string"`
	Elements int           `json:"elements"`
}

type allowedLifetime struct {
	addr    netip.Addr
	timeout time.Duration
	expires time.Duration
}

var allowedListing = regexp.MustCompile(`(?s)^\s*table inet fvpn\s*\{\s*set allowed_v4\s*\{\s*type ipv4_addr\s*flags timeout\s*(?:elements\s*=\s*\{([^{}]*)\}\s*)?\}\s*\}\s*$`)

func parseAllowedLifetimes(raw []byte) ([]allowedLifetime, error) {
	match := allowedListing.FindSubmatch(raw)
	if match == nil {
		return nil, fmt.Errorf("invalid allowed_v4 precision listing")
	}
	body := strings.TrimSpace(string(match[1]))
	if body == "" {
		return nil, nil
	}
	var out []allowedLifetime
	seen := map[netip.Addr]bool{}
	for _, row := range strings.Split(body, ",") {
		f := strings.Fields(row)
		if (len(f) != 3 && len(f) != 5) || f[1] != "timeout" {
			return nil, fmt.Errorf("invalid timed element")
		}
		addr, err := netip.ParseAddr(f[0])
		if err != nil || !addr.Is4() || seen[addr] {
			return nil, fmt.Errorf("invalid or duplicate timed host")
		}
		seen[addr] = true
		timeout, ok := parseNftDuration(f[2])
		if !ok || timeout <= 0 {
			return nil, fmt.Errorf("unbounded timed host")
		}
		expires := timeout
		if len(f) == 5 {
			if f[3] != "expires" {
				return nil, fmt.Errorf("invalid expiry")
			}
			expires, ok = parseNftDuration(f[4])
			if !ok {
				return nil, fmt.Errorf("invalid expiry duration")
			}
		}
		out = append(out, allowedLifetime{addr, timeout, expires})
	}
	return out, nil
}

// parseNftDuration keeps integer milliseconds, including rollover such as
// 1s999ms. It rejects sub-millisecond input, fractional numbers and overflow.
func parseNftDuration(s string) (time.Duration, bool) {
	var total uint64
	const maxMillis = uint64((1<<63 - 1) / int64(time.Millisecond))
	for len(s) > 0 {
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == 0 {
			return 0, false
		}
		n, err := strconv.ParseUint(s[:i], 10, 64)
		if err != nil {
			return 0, false
		}
		s = s[i:]
		var scale uint64
		switch {
		case strings.HasPrefix(s, "ms"):
			scale, s = 1, s[2:]
		case strings.HasPrefix(s, "s"):
			scale, s = 1000, s[1:]
		case strings.HasPrefix(s, "m"):
			scale, s = 60000, s[1:]
		case strings.HasPrefix(s, "h"):
			scale, s = 3600000, s[1:]
		case strings.HasPrefix(s, "d"):
			scale, s = 86400000, s[1:]
		default:
			return 0, false
		}
		if n > (maxMillis-total)/scale {
			return 0, false
		}
		total += n * scale
	}
	return time.Duration(total) * time.Millisecond, true
}

func verifyAllowedLifetimes(raw []byte, m Model, readStart time.Duration) (int, error) {
	elements, err := parseAllowedLifetimes(raw)
	if err != nil {
		return 0, checkf(ReasonIsolation, "%v", err)
	}
	limits := map[netip.Addr][2]time.Duration{}
	for _, binding := range m.Bindings {
		for _, lease := range m.Leases {
			if binding.MemberID != lease.MemberID {
				continue
			}
			original, _ := memberTimeout(binding, lease, m)
			current := m
			if m.NowMono > 0 {
				if readStart < m.NowMono {
					return 0, checkf(ReasonIsolation, "monotonic read precedes model")
				}
				current.Now = m.Now.Add(readStart - m.NowMono)
			}
			current.NowMono = readStart
			remaining, _ := memberTimeout(binding, lease, current)
			for _, addr := range binding.AllowedV4 {
				limits[addr.Addr()] = [2]time.Duration{original, remaining}
			}
		}
	}
	for _, el := range elements {
		limit, ok := limits[el.addr]
		if !ok || el.timeout > limit[0] || el.expires > limit[1] || el.expires > el.timeout {
			return len(elements), checkf(ReasonIsolation, "allowed_v4 lifetime exceeds model deadline")
		}
	}
	return len(elements), nil
}

func readAllowedLifetimes(ctx context.Context, ex Executor, m Model) (ev DeadlineReadEvidence, err error) {
	ev.Start, err = MonotonicNow()
	if err != nil {
		return ev, err
	}
	raw, err := ex.Output(ctx, "nft", "list", "set", "inet", "fvpn", "allowed_v4")
	ev.End, _ = MonotonicNow()
	if err != nil {
		return ev, checkf(ReasonIsolation, "allowed_v4 precision read failed")
	}
	ev.Elements, err = verifyAllowedLifetimes(raw, m, ev.Start)
	return ev, err
}
