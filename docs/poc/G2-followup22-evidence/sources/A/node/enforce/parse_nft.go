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
	// Foreign is every chain whose family or table is not inet fvpn, in
	// listing order. VerifyServing fails closed when one of these can change
	// the packet inet fvpn matches or a verdict inet fvpn already made.
	// Counters and sets from other tables stay out of Counters and Sets.
	Foreign []ForeignChain
	// ForeignFlows is every flowtable outside inet fvpn.
	ForeignFlows []ForeignFlow
}

// ForeignChain is one chain outside table inet fvpn.
type ForeignChain struct {
	Family  string
	Table   string
	Name    string
	Type    string
	Hook    string
	Policy  string
	Dev     []string
	Prio    int64
	HasPrio bool
	BadDev  bool
	Rules   []ForeignRule
}

// ForeignRule is one rule outside inet fvpn. Stmts are the nft JSON
// statement keys in order. Handle is set when the listing carried a handle.
type ForeignRule struct {
	Stmts     []ForeignStmt
	Handle    uint64
	HasHandle bool
}

// ForeignStmt is one nft statement. Target is set for jump and goto.
type ForeignStmt struct {
	Key    string
	Body   json.RawMessage
	Target string
}

// ForeignFlow is a flowtable outside inet fvpn.
type ForeignFlow struct {
	Family  string
	Table   string
	Name    string
	Hook    string
	Dev     []string
	Prio    int64
	HasPrio bool
	BadDev  bool
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

// ParseRuleset reads an nft -j listing. Apply verifies `nft -j list ruleset`
// so every table is in the bytes. The inet fvpn model is still keyed by
// family, table, and name: an ip fvpn chain or a chain in another table is
// recorded on Foreign and is not merged into Chains. Numbers above 2^53 stay
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
	// Chains are keyed by (family, table, name). A second inet fvpn object
	// with the same identity is an error. A different family or table is
	// not merged in: an `ip fvpn forward` policy must not hide the `inet`
	// chain's policy, and it is kept on Foreign for VerifyServing.
	seenChain := map[string]struct{}{}
	var foreign foreignBuild
	for _, obj := range doc.Nftables {
		switch {
		case hasKey(obj, "flowtable"):
			if err := noteFlowtable(&rs, obj["flowtable"]); err != nil {
				return Ruleset{}, err
			}
		case hasKey(obj, "chain"):
			if err := noteChain(&rs, &foreign, seenChain, obj["chain"]); err != nil {
				return Ruleset{}, err
			}
		case hasKey(obj, "rule"):
			if err := noteRule(&rs, &foreign, obj["rule"]); err != nil {
				return Ruleset{}, err
			}
		case hasKey(obj, "counter"):
			if foreign, err := foreignObject(obj["counter"]); err != nil {
				return Ruleset{}, err
			} else if foreign {
				continue
			}
			var c struct {
				Table   string      `json:"table"`
				Name    string      `json:"name"`
				Packets json.Number `json:"packets"`
				Bytes   json.Number `json:"bytes"`
			}
			if err := json.Unmarshal(obj["counter"], &c); err != nil {
				return Ruleset{}, err
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
			if foreign, err := foreignObject(obj["set"]); err != nil {
				return Ruleset{}, err
			} else if foreign {
				continue
			}
			var s struct {
				Table string          `json:"table"`
				Name  string          `json:"name"`
				Elem  json.RawMessage `json:"elem"`
				Flags json.RawMessage `json:"flags"`
			}
			if err := json.Unmarshal(obj["set"], &s); err != nil {
				return Ruleset{}, err
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
	rs.Foreign = foreign.done()
	if strings.Contains(rs.Raw, "\"flowtable\"") && len(rs.Flowtables) == 0 && !foreignExplainsFlowtable(rs) {
		rs.Flowtables = append(rs.Flowtables, "flowtable")
	}
	return rs, nil
}

// foreignObject reports whether a JSON object lives outside inet fvpn.
// An empty family or table is inet fvpn, matching a listing of that table alone.
func foreignObject(raw json.RawMessage) (bool, error) {
	fam, table, err := objectIdentity(raw)
	if err != nil {
		return false, err
	}
	return fam != TableFamily || table != TableName, nil
}

func objectIdentity(raw json.RawMessage) (family, table string, err error) {
	var head struct {
		Family string `json:"family"`
		Table  string `json:"table"`
	}
	if len(raw) != 0 {
		if err = json.Unmarshal(raw, &head); err != nil {
			return "", "", err
		}
	}
	family = head.Family
	if family == "" {
		family = TableFamily
	}
	table = head.Table
	if table == "" {
		table = TableName
	}
	return family, table, nil
}

type foreignBuild struct {
	order []string
	by    map[string]*ForeignChain
}

func (b *foreignBuild) get(fam, table, name string) *ForeignChain {
	if b.by == nil {
		b.by = map[string]*ForeignChain{}
	}
	key := fam + "\x00" + table + "\x00" + name
	if c, ok := b.by[key]; ok {
		return c
	}
	c := &ForeignChain{Family: fam, Table: table, Name: name}
	b.by[key] = c
	b.order = append(b.order, key)
	return c
}

func (b *foreignBuild) done() []ForeignChain {
	if len(b.order) == 0 {
		return nil
	}
	out := make([]ForeignChain, 0, len(b.order))
	for _, key := range b.order {
		out = append(out, *b.by[key])
	}
	return out
}

func noteFlowtable(rs *Ruleset, raw json.RawMessage) error {
	fam, table, err := objectIdentity(raw)
	if err != nil {
		return err
	}
	if fam == TableFamily && table == TableName {
		rs.Flowtables = append(rs.Flowtables, "flowtable")
		return nil
	}
	var c struct {
		Name string          `json:"name"`
		Hook string          `json:"hook"`
		Prio json.RawMessage `json:"prio"`
		Dev  json.RawMessage `json:"dev"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	ft := ForeignFlow{Family: fam, Table: table, Name: c.Name, Hook: c.Hook}
	if prio, has, bad := parsePrio(c.Prio); has && !bad {
		ft.Prio = prio
		ft.HasPrio = true
	}
	devs, badDev := parseDev(c.Dev)
	ft.Dev = devs
	ft.BadDev = badDev
	rs.ForeignFlows = append(rs.ForeignFlows, ft)
	return nil
}

func noteChain(rs *Ruleset, foreign *foreignBuild, seen map[string]struct{}, raw json.RawMessage) error {
	var c struct {
		Family string          `json:"family"`
		Table  string          `json:"table"`
		Name   string          `json:"name"`
		Type   string          `json:"type"`
		Hook   string          `json:"hook"`
		Policy string          `json:"policy"`
		Prio   json.RawMessage `json:"prio"`
		Dev    json.RawMessage `json:"dev"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	fam := c.Family
	if fam == "" {
		fam = TableFamily
	}
	table := c.Table
	if table == "" {
		table = TableName
	}
	if fam != TableFamily || table != TableName {
		ch := foreign.get(fam, table, c.Name)
		if c.Type != "" {
			ch.Type = c.Type
		}
		if c.Hook != "" {
			ch.Hook = c.Hook
		}
		if c.Policy != "" {
			ch.Policy = c.Policy
		}
		if prio, has, bad := parsePrio(c.Prio); bad {
			ch.HasPrio = false
		} else if has {
			ch.Prio = prio
			ch.HasPrio = true
		}
		if len(c.Dev) != 0 {
			devs, badDev := parseDev(c.Dev)
			ch.Dev = devs
			ch.BadDev = badDev
		}
		return nil
	}
	key := fam + "\x00" + table + "\x00" + c.Name
	if _, dup := seen[key]; dup {
		return fmt.Errorf("duplicate chain %s %s %s", fam, table, c.Name)
	}
	seen[key] = struct{}{}
	cv := rs.Chains[c.Name]
	cv.Name = c.Name
	cv.Type = c.Type
	cv.Hook = c.Hook
	cv.Policy = c.Policy
	if prio, has, bad := parsePrio(c.Prio); bad {
		return fmt.Errorf("chain %s priority: %s", c.Name, bytes.TrimSpace(c.Prio))
	} else if has {
		cv.Prio = prio
		cv.HasPrio = true
	}
	rs.Chains[c.Name] = cv
	return nil
}

func noteRule(rs *Ruleset, foreign *foreignBuild, raw json.RawMessage) error {
	var head struct {
		Family  string          `json:"family"`
		Table   string          `json:"table"`
		Chain   string          `json:"chain"`
		Comment string          `json:"comment"`
		Handle  *uint64         `json:"handle"`
		Expr    json.RawMessage `json:"expr"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return err
	}
	fam := head.Family
	if fam == "" {
		fam = TableFamily
	}
	table := head.Table
	if table == "" {
		table = TableName
	}
	if fam != TableFamily || table != TableName {
		ch := foreign.get(fam, table, head.Chain)
		rule := ForeignRule{Stmts: foreignStmts(head.Expr)}
		if head.Handle != nil {
			rule.Handle = *head.Handle
			rule.HasHandle = true
		}
		ch.Rules = append(ch.Rules, rule)
		return nil
	}
	rv := RuleView{Comment: head.Comment, Raw: string(head.Expr)}
	if strings.Contains(rv.Raw, "offload") {
		rv.Offload = true
	}
	if err := inspectExpr(head.Expr, &rv); err != nil {
		return err
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
	return nil
}

func foreignStmts(expr json.RawMessage) []ForeignStmt {
	if len(expr) == 0 || string(bytes.TrimSpace(expr)) == "null" {
		return nil
	}
	var arr []map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(expr))
	dec.UseNumber()
	if err := dec.Decode(&arr); err != nil {
		return []ForeignStmt{{Key: "unparsed"}}
	}
	out := make([]ForeignStmt, 0, len(arr))
	for _, obj := range arr {
		if len(obj) != 1 {
			out = append(out, ForeignStmt{Key: "unparsed"})
			continue
		}
		var key string
		var body json.RawMessage
		for k, v := range obj {
			key = k
			body = v
		}
		st := ForeignStmt{Key: key, Body: body}
		if key == "jump" || key == "goto" {
			st.Target = jumpTarget(body)
		}
		out = append(out, st)
	}
	return out
}

func jumpTarget(body json.RawMessage) string {
	var obj struct {
		Target string `json:"target"`
	}
	if json.Unmarshal(body, &obj) == nil && obj.Target != "" {
		return obj.Target
	}
	var s string
	if json.Unmarshal(body, &s) == nil {
		return s
	}
	return ""
}

// parsePrio reads an nft JSON priority. nft emits a number. has is false when
// the field is absent. bad is true when the field is present and not an integer.
func parsePrio(raw json.RawMessage) (prio int64, has, bad bool) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return 0, false, false
	}
	var num json.Number
	if err := json.Unmarshal(raw, &num); err == nil {
		n, err := num.Int64()
		if err != nil {
			return 0, false, true
		}
		return n, true, false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false, true
		}
		return n, true, false
	}
	return 0, false, true
}

// parseDev reads an nft device or device list. bad is true when the field is
// present and is neither a string nor an array of strings.
func parseDev(raw json.RawMessage) (devs []string, bad bool) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, false
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return []string{one}, false
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		return many, false
	}
	return nil, true
}

func foreignExplainsFlowtable(rs Ruleset) bool {
	if len(rs.ForeignFlows) > 0 {
		return true
	}
	for _, c := range rs.Foreign {
		for _, rule := range c.Rules {
			for _, st := range rule.Stmts {
				if st.Key == "flow" || st.Key == "xt" || bytes.Contains(st.Body, []byte(`"flowtable"`)) {
					return true
				}
			}
		}
	}
	return false
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
