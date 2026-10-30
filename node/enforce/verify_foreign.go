package enforce

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// verifyForeign fails closed when a base chain outside inet fvpn can change
// the packet that table matches, or can drop a packet that table must
// forward. An accept in one base chain does not stop later base chains of
// the same hook, so a chain that only matches, counts, logs, accepts, or
// continues is tolerated at any priority. input and output are tolerated
// because forwarded member traffic does not cross them. The agent still
// does not write these chains.
func verifyForeign(rs Ruleset, m Model) error {
	byKey := make(map[string]*ForeignChain, len(rs.Foreign))
	for i := range rs.Foreign {
		c := &rs.Foreign[i]
		byKey[foreignID(c.Family, c.Table, c.Name)] = c
	}
	for i := range rs.Foreign {
		c := &rs.Foreign[i]
		cross, unknown := hookCrosses(c.Family, c.Hook, c.Dev, m)
		if !cross {
			continue
		}
		if unknown {
			return foreignVerdict(c.Table, c.Family, c.Hook, c.HasPrio, c.Prio, c.Name, "hook", ReasonIsolation)
		}
		if err := checkForeignChain(c, byKey); err != nil {
			return err
		}
	}
	for _, ft := range rs.ForeignFlows {
		if !flowCrosses(ft, m) {
			continue
		}
		return foreignVerdict(ft.Table, ft.Family, ft.Hook, ft.HasPrio, ft.Prio, ft.Name, "flow", ReasonFlowtable)
	}
	return nil
}

func checkForeignChain(c *ForeignChain, byKey map[string]*ForeignChain) error {
	if c.BadDev {
		return foreignVerdict(c.Table, c.Family, c.Hook, c.HasPrio, c.Prio, c.Name, "device", ReasonIsolation)
	}
	switch c.Policy {
	case "", "accept":
	case "drop", "reject":
		return foreignVerdict(c.Table, c.Family, c.Hook, c.HasPrio, c.Prio, c.Name, c.Policy, ReasonIsolation)
	default:
		return foreignVerdict(c.Table, c.Family, c.Hook, c.HasPrio, c.Prio, c.Name, "policy", ReasonIsolation)
	}
	if !c.HasPrio {
		return foreignVerdict(c.Table, c.Family, c.Hook, false, 0, c.Name, "priority", ReasonIsolation)
	}
	return walkForeign(c, c, byKey, map[string]bool{})
}

func walkForeign(base, ch *ForeignChain, byKey map[string]*ForeignChain, stack map[string]bool) error {
	id := foreignID(ch.Family, ch.Table, ch.Name)
	if stack[id] {
		return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, "jump", ReasonIsolation)
	}
	stack[id] = true
	defer delete(stack, id)
	for _, rule := range ch.Rules {
		for _, st := range rule.Stmts {
			if err := foreignStmt(base, ch, st, byKey, stack); err != nil {
				return err
			}
		}
	}
	return nil
}

func foreignStmt(base, ch *ForeignChain, st ForeignStmt, byKey map[string]*ForeignChain, stack map[string]bool) error {
	switch st.Key {
	case "match", "counter", "log", "accept", "continue":
		return nil
	case "jump", "goto":
		if st.Target == "" {
			return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, st.Key, ReasonIsolation)
		}
		tgt, ok := byKey[foreignID(ch.Family, ch.Table, st.Target)]
		if !ok {
			return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, st.Key, ReasonIsolation)
		}
		return walkForeign(base, tgt, byKey, stack)
	case "mangle":
		if mangleSetsMark(st.Body) {
			return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, "mark", ReasonMark)
		}
		return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, "mangle", ReasonIsolation)
	case "flow", "xt":
		return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, st.Key, ReasonFlowtable)
	default:
		if st.Key == "" {
			st.Key = "unparsed"
		}
		return foreignVerdict(base.Table, base.Family, base.Hook, base.HasPrio, base.Prio, base.Name, st.Key, ReasonIsolation)
	}
}

func foreignVerdict(table, family, hook string, hasPrio bool, prio int64, chain, statement string, reason Reason) error {
	p := "missing"
	if hasPrio {
		p = strconv.FormatInt(prio, 10)
	}
	if hook == "" {
		hook = "missing"
	}
	return checkf(reason, "table %s family %s hook %s priority %s chain %s statement %s", table, family, hook, p, chain, statement)
}

func foreignID(family, table, name string) string {
	return family + "\x00" + table + "\x00" + name
}

// hookCrosses reports whether a base chain of this hook can see forwarded
// member traffic. unknown is true for a hook this model does not classify.
// input and output are host-local. ingress and egress cross only when the
// chain names no device or names the member ingress or the uplink.
func hookCrosses(family, hook string, dev []string, m Model) (cross, unknown bool) {
	if family == "arp" || hook == "" {
		return false, false
	}
	switch hook {
	case "input", "output":
		return false, false
	case "prerouting", "forward", "postrouting":
		return true, false
	case "ingress", "egress":
		if len(dev) == 0 || devHits(dev, m) {
			return true, false
		}
		return false, false
	default:
		return true, true
	}
}

func flowCrosses(ft ForeignFlow, m Model) bool {
	if ft.Family == "arp" {
		return false
	}
	if ft.BadDev || len(ft.Dev) == 0 || devHits(ft.Dev, m) {
		return true
	}
	return false
}

func devHits(dev []string, m Model) bool {
	for _, d := range dev {
		if d == "" || d == m.Datapath.MemberIngress || d == m.Datapath.Uplink {
			return true
		}
	}
	return false
}

func mangleSetsMark(body json.RawMessage) bool {
	if len(bytes.TrimSpace(body)) == 0 || string(bytes.TrimSpace(body)) == "null" {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return false
	}
	return jsonSetsMark(v)
}

func jsonSetsMark(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		if meta, ok := t["meta"].(map[string]any); ok && meta["key"] == "mark" {
			return true
		}
		if ct, ok := t["ct"].(map[string]any); ok && ct["key"] == "mark" {
			return true
		}
		for _, child := range t {
			if jsonSetsMark(child) {
				return true
			}
		}
	case []any:
		for _, child := range t {
			if jsonSetsMark(child) {
				return true
			}
		}
	}
	return false
}
