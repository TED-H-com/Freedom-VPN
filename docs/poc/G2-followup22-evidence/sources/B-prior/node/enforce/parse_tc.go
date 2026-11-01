package enforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// QdiscView is the HTB and fq_codel state on one interface.
type QdiscView struct {
	RootHTB bool
	Classes map[string]ClassView
	Leaves  map[uint16]bool
}

// ClassView is one HTB class. RateBPS is the guaranteed rate.
type ClassView struct {
	RateBPS uint64
	CeilBPS uint64
	Found   bool
}

// ParseQdiscJSON reads `tc -j qdisc show dev IFACE`.
func ParseQdiscJSON(raw []byte) (QdiscView, error) {
	view := QdiscView{Classes: map[string]ClassView{}, Leaves: map[uint16]bool{}}
	if len(bytes.TrimSpace(raw)) == 0 {
		return view, nil
	}
	var arr []map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&arr); err != nil {
		return view, fmt.Errorf("tc qdisc json: %w", err)
	}
	for _, obj := range arr {
		kind, _ := obj["kind"].(string)
		handle, _ := obj["handle"].(string)
		root, _ := obj["root"].(bool)
		parent, _ := obj["parent"].(string)
		if kind == "htb" && (root || parent == "") && strings.HasPrefix(handle, "1:") {
			view.RootHTB = true
		}
		if kind == "fq_codel" {
			major, minor, ok := splitHandle(parent)
			// The renderer attaches fq_codel under HTB major 1. A leaf whose
			// parent major is anything else is not the member class.
			if ok && uint32(major) == HTBMajor {
				view.Leaves[minor] = true
			}
		}
	}
	return view, nil
}

// ParseClassJSON reads `tc -j class show dev IFACE` and merges it into view.
// iproute2 on Ubuntu 24.04 prints class listings as text even with -j, while
// qdisc listings are JSON. Both forms are accepted.
func ParseClassJSON(raw []byte, view QdiscView) (QdiscView, error) {
	if view.Classes == nil {
		view.Classes = map[string]ClassView{}
	}
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 {
		return view, nil
	}
	if trim[0] != '[' && trim[0] != '{' {
		return parseClassText(trim, view)
	}
	var arr []map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&arr); err != nil {
		return view, fmt.Errorf("tc class json: %w", err)
	}
	for _, obj := range arr {
		kind, _ := obj["kind"].(string)
		if kind != "" && kind != "htb" {
			continue
		}
		handle, _ := obj["handle"].(string)
		if handle == "" {
			handle, _ = obj["class"].(string)
		}
		if handle == "" {
			continue
		}
		opts, _ := obj["options"].(map[string]any)
		rate, ceil := ratePair(opts)
		view.Classes[handle] = ClassView{RateBPS: rate, CeilBPS: ceil, Found: true}
	}
	return view, nil
}

func parseClassText(raw []byte, view QdiscView) (QdiscView, error) {
	// A comment or a backslash continuation can hide the real rate behind a
	// later token. Real `tc class show` output has neither.
	if bytes.Contains(raw, []byte("#")) || bytes.Contains(raw, []byte("\\")) {
		return view, fmt.Errorf("tc class text contains a comment or continuation")
	}
	found := 0
	seen := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "class htb ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		handle := fields[2]
		if seen[handle] {
			return view, fmt.Errorf("tc class %s is duplicated", handle)
		}
		var rate, ceil uint64
		var rateN, ceilN int
		var haveRate, haveCeil bool
		for i := 0; i+1 < len(fields); i++ {
			switch fields[i] {
			case "rate":
				rateN++
				rate, haveRate = parseRateString(fields[i+1])
			case "ceil":
				ceilN++
				ceil, haveCeil = parseRateString(fields[i+1])
			}
		}
		if rateN != 1 || ceilN != 1 || !haveRate || !haveCeil {
			return view, fmt.Errorf("tc class %s has no single rate and ceil", handle)
		}
		seen[handle] = true
		view.Classes[handle] = ClassView{RateBPS: rate, CeilBPS: ceil, Found: true}
		found++
	}
	if found == 0 {
		return view, fmt.Errorf("tc class text: no htb classes")
	}
	return view, nil
}

func ratePair(opts map[string]any) (rate, ceil uint64) {
	if opts == nil {
		return 0, 0
	}
	rate = firstRate(opts, "rate", "rate_bps", "rate64")
	ceil = firstRate(opts, "ceil", "ceil_bps", "ceil64")
	if rate == 0 {
		if b, ok := asUint(opts["rate_bytes"]); ok {
			rate = b * 8
		}
	}
	if ceil == 0 {
		if b, ok := asUint(opts["ceil_bytes"]); ok {
			ceil = b * 8
		}
	}
	return rate, ceil
}

func firstRate(opts map[string]any, keys ...string) uint64 {
	for _, k := range keys {
		if v, ok := opts[k]; ok {
			if n, ok := parseRateValue(v); ok {
				return n
			}
		}
	}
	return 0
}

func parseRateValue(v any) (uint64, bool) {
	switch t := v.(type) {
	case json.Number:
		n, err := strconv.ParseUint(t.String(), 10, 64)
		return n, err == nil
	case string:
		return parseRateString(t)
	default:
		return 0, false
	}
}

func parseRateString(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if n, err := strconv.ParseUint(s, 10, 64); err == nil {
		return n, true
	}
	lower := strings.ToLower(s)
	mult := uint64(1)
	for _, suf := range []struct {
		s string
		m uint64
	}{
		{"gbit", 1_000_000_000},
		{"mbit", 1_000_000},
		{"kbit", 1_000},
		{"bit", 1},
		{"gibit", 1_000_000_000},
		{"bps", 1},
	} {
		if strings.HasSuffix(lower, suf.s) {
			mult = suf.m
			lower = strings.TrimSuffix(lower, suf.s)
			break
		}
	}
	lower = strings.TrimSpace(lower)
	n, err := strconv.ParseFloat(lower, 64)
	if err != nil {
		return 0, false
	}
	return uint64(n * float64(mult)), true
}

func splitHandle(h string) (major, minor uint16, ok bool) {
	h = strings.TrimSuffix(h, ":")
	parts := strings.Split(h, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, 0, false
	}
	maj, err1 := strconv.ParseUint(parts[0], 16, 16)
	min, err2 := strconv.ParseUint(parts[1], 16, 16)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return uint16(maj), uint16(min), true
}
