package enforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Ruleset is the part of an nft -j listing that verification inspects.
type Ruleset struct {
	Chains   map[string]ChainView
	Counters map[string]CounterValue
	Sets     map[string]bool
	Elements map[string][]setElement
	// Tainted is true when a set contains an element this parser cannot
	// accept as a timed host address: a prefix wider than /32, a port range,
	// or any other shape. Verification fails closed on a tainted allow or
	// probe set (§13.9).
	Tainted map[string]bool
	// SetFlags is the flags array nft reported for each set. Absence is
	// empty. Verification rejects a flag the renderer does not emit.
	SetFlags   map[string][]string
	Flowtables []string
	Classes    map[uuid.UUID]uint16
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
	// Parsed match expressions. Verification uses these, not the raw JSON
	// text, so a comment cannot impersonate an interface or a set (§8.2).
	IIF              []string
	OIF              []string
	NegSets          []string
	NegSaddr         []string
	NegDaddr         []string
	SaddrSets        []string
	DaddrSets        []string
	ConcatSets       []string
	Dports           []uint16
	Sports           []uint16
	Saddrs           []netip.Prefix
	Daddrs           []netip.Prefix
	L4Protos         []string
	NFProto          string
	CTEstablished    bool
	CTNotEstablished bool
	MatchMark        bool
	DNAT             bool
	SNAT             string
	// Tainted is true when the rule carries an expression the renderer does
	// not emit: a statement outside the rendered set, more than one verdict
	// token, a verdict token that is not itself a statement, a positive match
	// whose operator is not ==, or an address right-hand side that is neither
	// one prefix nor one named set. Verification fails closed.
	Tainted bool
	// Verdicts is the number of accept and drop tokens anywhere in the
	// expression, including tokens nested inside a verdict map.
	Verdicts int
}

// setElement is one element of an nft set. Bounded is false when the element
// has no timeout and no expires value (§13.9).
type setElement struct {
	Addr   netip.Addr
	Proto  string
	Port   uint16
	Concat bool
	// Bounded is false when the element has no timeout and no expires (§13.9).
	// Seconds is max(timeout, expires). Remain is the smaller of the two when
	// both are present, otherwise Seconds. Render uses Remain so a re-render
	// cannot extend the kernel lifetime. Window is expires when both are
	// present, otherwise the single value: verification compares it with the
	// model window. The original timeout is Seconds, compared with the ceiling.
	Bounded bool
	Seconds uint32
	Remain  uint32
	Window  uint32
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
		Elements: map[string][]setElement{},
		Tainted:  map[string]bool{},
		SetFlags: map[string][]string{},
		Raw:      string(raw),
	}
	// Chains are keyed by (family, table, name). A second object with the
	// same identity is an error. A different family is not merged in: an
	// `ip fvpn forward` policy must not hide the `inet` chain's policy.
	seenChain := map[string]struct{}{}
	for _, obj := range doc.Nftables {
		switch {
		case hasKey(obj, "flowtable"):
			rs.Flowtables = append(rs.Flowtables, "flowtable")
		case hasKey(obj, "chain"):
			var c struct {
				Family string      `json:"family"`
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
			fam := c.Family
			if fam == "" {
				fam = TableFamily
			}
			if fam != TableFamily {
				continue
			}
			table := c.Table
			if table == "" {
				table = TableName
			}
			key := fam + "\x00" + table + "\x00" + c.Name
			if _, dup := seenChain[key]; dup {
				return Ruleset{}, fmt.Errorf("duplicate chain %s %s %s", fam, table, c.Name)
			}
			seenChain[key] = struct{}{}
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
				Family  string          `json:"family"`
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
			fam := head.Family
			if fam == "" {
				fam = TableFamily
			}
			if fam != TableFamily {
				continue
			}
			rv := RuleView{Comment: head.Comment, Raw: string(head.Expr)}
			if strings.Contains(rv.Raw, "offload") {
				rv.Offload = true
			}
			if err := inspectExpr(head.Expr, &rv); err != nil {
				return Ruleset{}, err
			}
			if classes := classAssignments(head.Comment); len(classes) > 0 {
				if rs.Classes == nil {
					rs.Classes = map[uuid.UUID]uint16{}
				}
				for id, minor := range classes {
					rs.Classes[id] = minor
				}
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
				Table string          `json:"table"`
				Name  string          `json:"name"`
				Elem  json.RawMessage `json:"elem"`
				Flags json.RawMessage `json:"flags"`
			}
			if err := json.Unmarshal(obj["set"], &s); err != nil {
				return Ruleset{}, err
			}
			if s.Table != "" && s.Table != TableName {
				continue
			}
			flags, err := parseSetFlags(s.Flags)
			if err != nil {
				return Ruleset{}, fmt.Errorf("set %s flags: %w", s.Name, err)
			}
			rs.Sets[s.Name] = true
			if len(flags) > 0 {
				rs.SetFlags[s.Name] = flags
			}
			elems, tainted, err := parseSetElems(s.Elem)
			if err != nil {
				return Ruleset{}, fmt.Errorf("set %s: %w", s.Name, err)
			}
			if tainted {
				rs.Tainted[s.Name] = true
			}
			if len(elems) > 0 {
				rs.Elements[s.Name] = append(rs.Elements[s.Name], elems...)
			}
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
		Table:      true,
		Chains:     map[string]bool{},
		Sets:       map[string]bool{},
		Counters:   rs.Counters,
		Classes:    rs.Classes,
		Generation: forwardGeneration(rs.Chains["forward"].Rules),
	}
	for name, ch := range rs.Chains {
		if ch.Hook != "" || ch.Type != "" || len(ch.Rules) > 0 {
			st.Chains[name] = true
		}
	}
	for name := range rs.Sets {
		st.Sets[name] = true
	}
	st.ProbeRemain = probeSetRemain(rs)
	return st, nil
}

// probeSetRemain is the shortest remaining probe-element lifetime already in
// the kernel. Zero means the probe sets are absent, empty, or unreadable.
func probeSetRemain(rs Ruleset) uint32 {
	var remain uint32
	for _, name := range []string{setProbeV4, setProbeDstV4} {
		if rs.Tainted[name] {
			continue
		}
		for _, el := range rs.Elements[name] {
			r := el.Remain
			if r == 0 {
				r = el.Seconds
			}
			if r == 0 {
				continue
			}
			if remain == 0 || r < remain {
				remain = r
			}
		}
	}
	return remain
}

// classAssignments reads "fvpn_class=<uuid>:<minorhex>" from a rule comment.
// Several members may be comma-separated in one comment; the renderer emits
// one member per comment because nft allows at most 128 bytes.
func classAssignments(comment string) map[uuid.UUID]uint16 {
	const key = "fvpn_class="
	i := strings.Index(comment, key)
	if i < 0 {
		return nil
	}
	rest := comment[i+len(key):]
	if sp := strings.IndexAny(rest, " \t"); sp >= 0 {
		rest = rest[:sp]
	}
	if rest == "" {
		return nil
	}
	out := map[uuid.UUID]uint16{}
	for _, part := range strings.Split(rest, ",") {
		idPart, minorPart, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		id, err := uuid.Parse(idPart)
		if err != nil {
			continue
		}
		n, err := strconv.ParseUint(minorPart, 16, 16)
		if err != nil || n == 0 {
			continue
		}
		out[id] = uint16(n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func forwardGeneration(rules []RuleView) uint64 {
	var max uint64
	seen := false
	for _, rule := range rules {
		n, ok := parseGeneration(rule.Comment)
		if !ok {
			continue
		}
		if !seen || n > max {
			max = n
			seen = true
		}
	}
	return max
}

func parseGeneration(comment string) (uint64, bool) {
	const key = "fvpn_generation="
	i := strings.Index(comment, key)
	if i < 0 {
		return 0, false
	}
	rest := comment[i+len(key):]
	j := 0
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	if j == 0 {
		return 0, false
	}
	n, err := strconv.ParseUint(rest[:j], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
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
	noteStatements(v, rv)
	if rv.Verdicts > 1 {
		rv.Tainted = true
	}
	return nil
}

// renderedStatements are the expression keys Render emits. A direct statement
// with any other key is outside the model. flow and xt are offload bypasses.
var renderedStatements = map[string]bool{
	"match": true, "counter": true, "mangle": true,
	"accept": true, "drop": true, "snat": true,
}

func noteStatements(v any, rv *RuleView) {
	arr, ok := v.([]any)
	if !ok {
		if v != nil {
			rv.Tainted = true
		}
		return
	}
	stmtVerdicts := 0
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok || len(obj) != 1 {
			rv.Tainted = true
			continue
		}
		var key string
		for k := range obj {
			key = k
		}
		switch key {
		case "accept", "drop":
			stmtVerdicts++
		case "flow", "xt":
			rv.Offload = true
			rv.Tainted = true
		default:
			if !renderedStatements[key] {
				rv.Tainted = true
			}
		}
	}
	// A verdict token nested inside a match or a verdict map is not the
	// statement the renderer emits.
	if stmtVerdicts != rv.Verdicts {
		rv.Tainted = true
	}
}

func walkExpr(v any, rv *RuleView) {
	switch t := v.(type) {
	case map[string]any:
		if _, ok := t["accept"]; ok {
			rv.Verdict = "accept"
			rv.Verdicts++
		}
		if _, ok := t["drop"]; ok {
			rv.Verdict = "drop"
			rv.Verdicts++
		}
		if _, ok := t["dnat"]; ok {
			rv.DNAT = true
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
		if match, ok := t["match"]; ok {
			noteMatch(match, rv)
		}
		if snat, ok := t["snat"]; ok {
			noteSNAT(snat, rv)
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

// opIsEq is the positive match the renderer emits. A missing operator is the
// same match: some fixtures omit it, and nft prints "==" when it is present.
func opIsEq(op string) bool {
	return op == "" || op == "=="
}

func noteMatch(v any, rv *RuleView) {
	m, ok := v.(map[string]any)
	if !ok {
		rv.Tainted = true
		return
	}
	op, _ := m["op"].(string)
	left, _ := m["left"].(map[string]any)
	right := m["right"]
	if meta, ok := left["meta"].(map[string]any); ok {
		key, _ := meta["key"].(string)
		name, _ := right.(string)
		switch key {
		case "mark":
			rv.MatchMark = true
		case "iifname", "oifname":
			if !opIsEq(op) || name == "" {
				rv.Tainted = true
				break
			}
			if key == "iifname" {
				rv.IIF = append(rv.IIF, name)
			} else {
				rv.OIF = append(rv.OIF, name)
			}
		case "nfproto":
			if !opIsEq(op) || name == "" {
				rv.Tainted = true
				break
			}
			rv.NFProto = name
		case "l4proto":
			if !opIsEq(op) || name == "" {
				rv.Tainted = true
				break
			}
			rv.L4Protos = append(rv.L4Protos, name)
		}
	}
	if _, ok := left["concat"]; ok {
		if !opIsEq(op) {
			rv.Tainted = true
		} else if name, ok := matchSet(right); ok {
			rv.ConcatSets = append(rv.ConcatSets, name)
		} else {
			rv.Tainted = true
		}
	}
	if payload, ok := left["payload"].(map[string]any); ok {
		field, _ := payload["field"].(string)
		switch field {
		case "saddr", "daddr":
			noteAddrMatch(rv, op, field, right)
		case "dport", "sport":
			if !opIsEq(op) {
				rv.Tainted = true
				break
			}
			if n, ok := asUint(right); ok && n > 0 && n <= 65535 {
				if field == "sport" {
					rv.Sports = append(rv.Sports, uint16(n))
				} else {
					rv.Dports = append(rv.Dports, uint16(n))
				}
			}
			if proto, ok := payload["protocol"].(string); ok && proto != "" {
				rv.L4Protos = append(rv.L4Protos, proto)
			}
		}
	}
	if ct, ok := left["ct"].(map[string]any); ok && ct["key"] == "state" && mentionsEstablished(right) {
		if op == "!=" {
			rv.CTNotEstablished = true
		} else {
			rv.CTEstablished = true
		}
	}
}

// noteAddrMatch records one named set or one prefix. A negated named set is
// the allow-set drop the renderer emits. Anything else, including an
// anonymous set and a negated prefix, taints the rule.
func noteAddrMatch(rv *RuleView, op, field string, right any) {
	if name, ok := namedSetRef(right); ok {
		if op == "!=" {
			rv.NegSets = append(rv.NegSets, name)
			if field == "saddr" {
				rv.NegSaddr = append(rv.NegSaddr, name)
			} else {
				rv.NegDaddr = append(rv.NegDaddr, name)
			}
			return
		}
		if opIsEq(op) {
			if field == "saddr" {
				rv.SaddrSets = append(rv.SaddrSets, name)
			} else {
				rv.DaddrSets = append(rv.DaddrSets, name)
			}
			return
		}
		rv.Tainted = true
		return
	}
	if opIsEq(op) {
		if p, ok := matchPrefix(right); ok {
			if field == "saddr" {
				rv.Saddrs = append(rv.Saddrs, p)
			} else {
				rv.Daddrs = append(rv.Daddrs, p)
			}
			return
		}
	}
	rv.Tainted = true
}

// namedSetRef is one named set. An anonymous set (a list of prefixes or
// addresses) is not a named set.
func namedSetRef(right any) (string, bool) {
	switch t := right.(type) {
	case string:
		if strings.HasPrefix(t, "@") && len(t) > 1 {
			return t[1:], true
		}
	case map[string]any:
		switch set := t["set"].(type) {
		case string:
			if set != "" {
				return strings.TrimPrefix(set, "@"), true
			}
		case []any:
			if len(set) == 1 {
				s, ok := set[0].(string)
				if ok && s != "" && !strings.Contains(s, ".") {
					return strings.TrimPrefix(s, "@"), true
				}
			}
		}
	}
	return "", false
}

func noteSNAT(v any, rv *RuleView) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	if addr, ok := m["addr"].(string); ok && addr != "" {
		rv.SNAT = addr
	}
}

func matchPrefix(right any) (netip.Prefix, bool) {
	switch t := right.(type) {
	case string:
		if strings.HasPrefix(t, "@") {
			return netip.Prefix{}, false
		}
		if p, err := netip.ParsePrefix(t); err == nil && p.Addr().Is4() {
			return p.Masked(), true
		}
		if a, err := netip.ParseAddr(t); err == nil && a.Is4() {
			return netip.PrefixFrom(a, 32), true
		}
	case map[string]any:
		if pref, ok := t["prefix"].(map[string]any); ok {
			return prefixFromMap(pref)
		}
		if _, ok := t["addr"]; ok {
			return prefixFromMap(t)
		}
	}
	return netip.Prefix{}, false
}

func prefixFromMap(m map[string]any) (netip.Prefix, bool) {
	addr, _ := m["addr"].(string)
	a, err := netip.ParseAddr(addr)
	if err != nil || !a.Is4() {
		return netip.Prefix{}, false
	}
	bits := 32
	if n, ok := asUint(m["len"]); ok && n <= 32 {
		bits = int(n)
	}
	p := netip.PrefixFrom(a, bits)
	if !p.IsValid() {
		return netip.Prefix{}, false
	}
	return p.Masked(), true
}

func parseSetFlags(raw json.RawMessage) ([]string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	var one string
	if err := json.Unmarshal(trimmed, &one); err == nil {
		if one == "" {
			return nil, nil
		}
		return []string{one}, nil
	}
	var many []string
	if err := json.Unmarshal(trimmed, &many); err != nil {
		return nil, err
	}
	return many, nil
}

func matchSet(right any) (string, bool) {
	switch t := right.(type) {
	case string:
		if strings.HasPrefix(t, "@") && len(t) > 1 {
			return t[1:], true
		}
	case map[string]any:
		switch set := t["set"].(type) {
		case string:
			if set != "" {
				return strings.TrimPrefix(set, "@"), true
			}
		case []any:
			for _, item := range set {
				if name, ok := matchSet(item); ok {
					return name, true
				}
				if s, ok := item.(string); ok && s != "" && !strings.Contains(s, ".") {
					return strings.TrimPrefix(s, "@"), true
				}
			}
		}
	}
	return "", false
}

func mentionsEstablished(v any) bool {
	switch t := v.(type) {
	case string:
		return t == "established" || t == "related"
	case []any:
		for _, child := range t {
			if mentionsEstablished(child) {
				return true
			}
		}
	case map[string]any:
		for _, child := range t {
			if mentionsEstablished(child) {
				return true
			}
		}
	}
	return false
}

func noteMeta(meta map[string]any, parent map[string]any, rv *RuleView) {
	if meta["key"] == "mark" {
		if _, hasValue := meta["value"]; hasValue || parent["value"] != nil {
			rv.SetsMark = true
		}
	}
}

// parseClassID reads the nft JSON form of meta priority. nft 1.0.9 prints a
// classid as "major:minor", both hexadecimal, for example "1:2dc5".
func parseClassID(s string) (uint64, bool) {
	major, minor, ok := strings.Cut(s, ":")
	if !ok || major == "" || minor == "" || strings.Contains(minor, ":") {
		return 0, false
	}
	maj, err1 := strconv.ParseUint(major, 16, 16)
	min, err2 := strconv.ParseUint(minor, 16, 16)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return (maj << 16) | min, true
}

func asUint(v any) (uint64, bool) {
	switch t := v.(type) {
	case json.Number:
		n, err := strconv.ParseUint(t.String(), 10, 64)
		return n, err == nil
	case string:
		if n, ok := parseClassID(t); ok {
			return n, true
		}
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

func parseSetElems(raw json.RawMessage) ([]setElement, bool, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, false, err
	}
	var out []setElement
	tainted := false
	note := func(item any) {
		el, ok := parseOneElem(item)
		if !ok {
			tainted = true
			return
		}
		out = append(out, el)
	}
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			note(item)
		}
	default:
		note(t)
	}
	return out, tainted, nil
}

func parseOneElem(item any) (setElement, bool) {
	switch t := item.(type) {
	case string:
		return parseElemString(t)
	case map[string]any:
		if inner, ok := t["elem"]; ok {
			return parseOneElem(inner)
		}
		return parseElemObject(t)
	default:
		return setElement{}, false
	}
}

func parseElemObject(m map[string]any) (setElement, bool) {
	bounded, seconds, remain, window := timeoutFromMap(m)
	val, ok := m["val"]
	if !ok {
		if _, has := m["concat"]; has {
			return parseConcatVal(m, bounded, seconds, remain, window)
		}
		return setElement{}, false
	}
	switch t := val.(type) {
	case string:
		el, ok := parseElemString(t)
		if !ok {
			return setElement{}, false
		}
		if bounded {
			el.Bounded = true
			el.Seconds = seconds
			el.Remain = remain
			el.Window = window
		}
		return el, true
	case map[string]any:
		return parseConcatVal(t, bounded, seconds, remain, window)
	case []any:
		return elemFromParts(stringParts(t), bounded, seconds, remain, window)
	default:
		return setElement{}, false
	}
}

func parseConcatVal(m map[string]any, bounded bool, seconds, remain, window uint32) (setElement, bool) {
	raw, ok := m["concat"]
	if !ok {
		return setElement{}, false
	}
	arr, ok := raw.([]any)
	if !ok {
		return setElement{}, false
	}
	return elemFromParts(stringParts(arr), bounded, seconds, remain, window)
}

func stringParts(arr []any) []string {
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		switch t := item.(type) {
		case string:
			out = append(out, t)
		case json.Number:
			out = append(out, t.String())
		default:
			out = append(out, fmt.Sprint(item))
		}
	}
	return out
}

func timeoutFromMap(m map[string]any) (bool, uint32, uint32, uint32) {
	timeout, hasT := asTimeout(m["timeout"])
	expires, hasE := asTimeout(m["expires"])
	if timeout == 0 {
		hasT = false
	}
	if expires == 0 {
		hasE = false
	}
	return combineTimeout(hasT, timeout, hasE, expires)
}

// combineTimeout reports max(timeout, expires) as seconds and the tighter of
// the two as remain. window is expires when both are present, otherwise the
// single value. Remain stays the tighter value so a render ceiling does not
// follow a listing whose expires is longer than its timeout.
func combineTimeout(hasT bool, timeout uint32, hasE bool, expires uint32) (bounded bool, seconds, remain, window uint32) {
	if !hasT && !hasE {
		return false, 0, 0, 0
	}
	seconds = timeout
	if expires > seconds {
		seconds = expires
	}
	remain = seconds
	if hasT && hasE {
		remain = timeout
		if expires < remain {
			remain = expires
		}
	}
	window = timeout
	if hasE {
		window = expires
	}
	return true, seconds, remain, window
}

func asTimeout(v any) (uint32, bool) {
	if v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case json.Number:
		n, err := strconv.ParseUint(t.String(), 10, 32)
		if err != nil {
			return 0, false
		}
		return uint32(n), true
	case string:
		return parseSeconds(t)
	default:
		return 0, false
	}
}

func parseElemString(s string) (setElement, bool) {
	timeout, hasT, s := takeListedDuration(s, "timeout")
	expires, hasE, s := takeListedDuration(s, "expires")
	bounded, seconds, remain, window := combineTimeout(hasT, timeout, hasE, expires)
	return elemFromParts(splitConcat(s), bounded, seconds, remain, window)
}

// takeListedDuration removes one "key <duration>" clause. The duration is
// seconds or an nft span such as 5m or 4m47s. A later clause is left in place.
func takeListedDuration(s, key string) (uint32, bool, string) {
	searchFrom := 0
	for searchFrom < len(s) {
		j := strings.Index(s[searchFrom:], key)
		if j < 0 {
			return 0, false, s
		}
		j += searchFrom
		if j > 0 && s[j-1] != ' ' && s[j-1] != '\t' {
			searchFrom = j + len(key)
			continue
		}
		after := j + len(key)
		if after >= len(s) || (s[after] != ' ' && s[after] != '\t') {
			searchFrom = j + len(key)
			continue
		}
		rest := strings.TrimSpace(s[after:])
		field, _, _ := strings.Cut(rest, " ")
		if i := strings.IndexByte(field, '\t'); i >= 0 {
			field = field[:i]
		}
		n, ok := parseSeconds(field)
		if !ok || n == 0 {
			return 0, false, s
		}
		rel := strings.Index(s[after:], field)
		if rel < 0 {
			return 0, false, s
		}
		end := after + rel + len(field)
		cleaned := strings.TrimSpace(s[:j] + " " + s[end:])
		return n, true, strings.Join(strings.Fields(cleaned), " ")
	}
	return 0, false, s
}

func splitConcat(s string) []string {
	raw := strings.Split(s, " . ")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func elemFromParts(parts []string, bounded bool, seconds, remain, window uint32) (setElement, bool) {
	if len(parts) == 0 {
		return setElement{}, false
	}
	addr, err := netip.ParseAddr(parts[0])
	if err != nil {
		p, err2 := netip.ParsePrefix(parts[0])
		// A CIDR string is a host element only when it is a single address.
		// Anything wider fails the set check (§13.9).
		if err2 != nil || !p.IsValid() || !p.IsSingleIP() {
			return setElement{}, false
		}
		addr = p.Addr()
	}
	if !addr.Is4() {
		return setElement{}, false
	}
	el := setElement{Addr: addr, Bounded: bounded, Seconds: seconds, Remain: remain, Window: window}
	if len(parts) == 1 {
		return el, true
	}
	if len(parts) != 3 {
		return setElement{}, false
	}
	proto, ok := parseL4Proto(parts[1])
	if !ok {
		return setElement{}, false
	}
	port, err := strconv.ParseUint(parts[2], 10, 16)
	if err != nil || port == 0 {
		return setElement{}, false
	}
	el.Proto = proto
	el.Port = uint16(port)
	el.Concat = true
	return el, true
}

func parseL4Proto(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "tcp", "6":
		return "tcp", true
	case "udp", "17":
		return "udp", true
	default:
		return "", false
	}
}

// parseSeconds reads a timeout as a whole number of seconds or as an nft time
// span (30s, 5m, 4m47s, 1h, 1d). Milliseconds are floored. A bare number is
// seconds.
func parseSeconds(s string) (uint32, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if n, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint32(n), true
	}
	var total uint64
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j == i {
			return 0, false
		}
		n, err := strconv.ParseUint(s[i:j], 10, 64)
		if err != nil {
			return 0, false
		}
		i = j
		if i >= len(s) {
			return 0, false
		}
		var mult uint64
		switch {
		case strings.HasPrefix(s[i:], "ms"):
			i += 2
			add := n / 1000
			if add > 0xffffffff || total > 0xffffffff-add {
				return 0, false
			}
			total += add
			continue
		case s[i] == 's':
			mult = 1
		case s[i] == 'm':
			mult = 60
		case s[i] == 'h':
			mult = 3600
		case s[i] == 'd':
			mult = 86400
		default:
			return 0, false
		}
		i++
		if n > 0xffffffff/mult {
			return 0, false
		}
		add := n * mult
		if total > 0xffffffff-add {
			return 0, false
		}
		total += add
	}
	if total == 0 || total > 0xffffffff {
		return 0, false
	}
	return uint32(total), true
}
