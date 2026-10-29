package enforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Ruleset is the part of an nft -j listing that verification inspects.
type Ruleset struct {
	Chains     map[string]ChainView
	Counters   map[string]CounterValue
	Sets       map[string]bool
	Flowtables []string
	Raw        string
}

// ChainView is one base chain in table order.
type ChainView struct {
	Name    string
	Type    string
	Hook    string
	Policy  string
	Prio    int64
	HasPrio bool
	Rules   []RuleView
}

// RuleView is one rule. Raw is the JSON of its expressions.
type RuleView struct {
	Comment  string
	Raw      string
	Verdict  string
	Counter  string
	SetsMark bool
	Offload  bool
	Prios    []uint32
}

// ParseRuleset reads `nft -j list table inet fvpn`. Numbers above 2^53 stay
// exact because the decoder uses json.Number (§10.4).
func ParseRuleset(raw []byte) (Ruleset, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc struct {
		Nftables []map[string]json.RawMessage `json:"nftables"`
	}
	if err := dec.Decode(&doc); err != nil {
		return Ruleset{}, fmt.Errorf("nft json: %w", err)
	}
	rs := Ruleset{
		Chains:   map[string]ChainView{},
		Counters: map[string]CounterValue{},
		Sets:     map[string]bool{},
		Raw:      string(raw),
	}
	for _, obj := range doc.Nftables {
		switch {
		case hasKey(obj, "flowtable"):
			rs.Flowtables = append(rs.Flowtables, "flowtable")
		case hasKey(obj, "chain"):
			var c struct {
				Table  string      `json:"table"`
				Name   string      `json:"name"`
				Type   string      `json:"type"`
				Hook   string      `json:"hook"`
				Policy string      `json:"policy"`
				Prio   json.Number `json:"prio"`
			}
			if err := json.Unmarshal(obj["chain"], &c); err != nil {
				return Ruleset{}, err
			}
			if c.Table != "" && c.Table != TableName {
				continue
			}
			cv := rs.Chains[c.Name]
			cv.Name = c.Name
			cv.Type = c.Type
			cv.Hook = c.Hook
			cv.Policy = c.Policy
			if c.Prio != "" {
				n, err := c.Prio.Int64()
				if err != nil {
					return Ruleset{}, fmt.Errorf("chain %s priority: %w", c.Name, err)
				}
				cv.Prio = n
				cv.HasPrio = true
			}
			rs.Chains[c.Name] = cv
		case hasKey(obj, "rule"):
			var head struct {
				Table   string          `json:"table"`
				Chain   string          `json:"chain"`
				Comment string          `json:"comment"`
				Expr    json.RawMessage `json:"expr"`
			}
			if err := json.Unmarshal(obj["rule"], &head); err != nil {
				return Ruleset{}, err
			}
			if head.Table != "" && head.Table != TableName {
				continue
			}
			rv := RuleView{Comment: head.Comment, Raw: string(head.Expr)}
			if strings.Contains(rv.Raw, "offload") {
				rv.Offload = true
			}
			if err := inspectExpr(head.Expr, &rv); err != nil {
				return Ruleset{}, err
			}
			ch := rs.Chains[head.Chain]
			ch.Name = head.Chain
			ch.Rules = append(ch.Rules, rv)
			rs.Chains[head.Chain] = ch
		case hasKey(obj, "counter"):
			var c struct {
				Table   string      `json:"table"`
				Name    string      `json:"name"`
				Packets json.Number `json:"packets"`
				Bytes   json.Number `json:"bytes"`
			}
			if err := json.Unmarshal(obj["counter"], &c); err != nil {
				return Ruleset{}, err
			}
			if c.Table != "" && c.Table != TableName {
				continue
			}
			packets, err := parseUintNumber(c.Packets)
			if err != nil {
				return Ruleset{}, fmt.Errorf("counter %s packets: %w", c.Name, err)
			}
			byteCount, err := parseUintNumber(c.Bytes)
			if err != nil {
				return Ruleset{}, fmt.Errorf("counter %s bytes: %w", c.Name, err)
			}
			rs.Counters[c.Name] = CounterValue{Packets: packets, Bytes: byteCount}
		case hasKey(obj, "set"):
			var s struct {
				Table string `json:"table"`
				Name  string `json:"name"`
			}
			if err := json.Unmarshal(obj["set"], &s); err != nil {
				return Ruleset{}, err
			}
			if s.Table != "" && s.Table != TableName {
				continue
			}
			rs.Sets[s.Name] = true
		}
	}
	if strings.Contains(rs.Raw, "\"flowtable\"") && len(rs.Flowtables) == 0 {
		rs.Flowtables = append(rs.Flowtables, "flowtable")
	}
	return rs, nil
}

// ParseState extracts the objects a later render must preserve.
func ParseState(raw []byte) (State, error) {
	rs, err := ParseRuleset(raw)
	if err != nil {
		return State{}, err
	}
	st := State{
		Table:    true,
		Chains:   map[string]bool{},
		Sets:     map[string]bool{},
		Counters: rs.Counters,
	}
	for name, ch := range rs.Chains {
		if ch.Hook != "" || ch.Type != "" || len(ch.Rules) > 0 {
			st.Chains[name] = true
		}
	}
	for name := range rs.Sets {
		st.Sets[name] = true
	}
	return st, nil
}

func hasKey(obj map[string]json.RawMessage, key string) bool {
	_, ok := obj[key]
	return ok
}

func parseUintNumber(n json.Number) (uint64, error) {
	if n == "" {
		return 0, nil
	}
	return strconv.ParseUint(n.String(), 10, 64)
}

func inspectExpr(raw json.RawMessage, rv *RuleView) error {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return err
	}
	walkExpr(v, rv)
	return nil
}

func walkExpr(v any, rv *RuleView) {
	switch t := v.(type) {
	case map[string]any:
		if _, ok := t["accept"]; ok {
			rv.Verdict = "accept"
		}
		if _, ok := t["drop"]; ok {
			rv.Verdict = "drop"
		}
		if c, ok := t["counter"]; ok {
			switch cv := c.(type) {
			case map[string]any:
				if name, ok := cv["name"].(string); ok {
					rv.Counter = name
				}
			case string:
				rv.Counter = cv
			}
		}
		if m, ok := t["mangle"]; ok {
			noteMangle(m, rv)
		}
		if meta, ok := t["meta"].(map[string]any); ok {
			noteMeta(meta, t, rv)
		}
		if ct, ok := t["ct"]; ok {
			blob := fmt.Sprint(ct)
			if strings.Contains(blob, "mark") {
				rv.SetsMark = true
			}
		}
		for _, child := range t {
			walkExpr(child, rv)
		}
	case []any:
		for _, child := range t {
			walkExpr(child, rv)
		}
	}
}

func noteMangle(v any, rv *RuleView) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	key := m["key"]
	if key == nil {
		key = m["left"]
	}
	val := m["value"]
	if val == nil {
		val = m["right"]
	}
	meta, _ := key.(map[string]any)
	inner, _ := meta["meta"].(map[string]any)
	if inner == nil {
		if meta["key"] == "priority" || meta["key"] == "mark" {
			inner = meta
		}
	}
	if inner["key"] == "mark" {
		rv.SetsMark = true
	}
	if inner["key"] == "priority" {
		if n, ok := asUint(val); ok {
			rv.Prios = append(rv.Prios, uint32(n))
		}
	}
}

func noteMeta(meta map[string]any, parent map[string]any, rv *RuleView) {
	if meta["key"] == "mark" {
		if _, hasValue := meta["value"]; hasValue || parent["value"] != nil {
			rv.SetsMark = true
		}
	}
}

func asUint(v any) (uint64, bool) {
	switch t := v.(type) {
	case json.Number:
		n, err := strconv.ParseUint(t.String(), 10, 64)
		return n, err == nil
	case string:
		n, err := strconv.ParseUint(strings.TrimPrefix(t, "0x"), 16, 64)
		if err == nil && strings.HasPrefix(t, "0x") {
			return n, true
		}
		n, err = strconv.ParseUint(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
