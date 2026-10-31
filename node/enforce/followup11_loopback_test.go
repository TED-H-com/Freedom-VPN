package enforce

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// dockerDNSListing is `nft -j list ruleset` from container
// fvpn-w1-netlab-dnsprobe on Docker network fvpn-w1-netlab-dns. It is the
// embedded DNS NAT AT10 recorded on 2026-10-31: OUTPUT and POSTROUTING jump
// into DOCKER_OUTPUT and DOCKER_POSTROUTING, and every rule matches
// 127.0.0.11. The tcp and udp sports are the DNS proxy ports from that dump.
const dockerDNSListing = `{"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "ip", "name": "nat", "handle": 1}}, {"chain": {"family": "ip", "table": "nat", "name": "DOCKER_OUTPUT", "handle": 1}}, {"chain": {"family": "ip", "table": "nat", "name": "OUTPUT", "handle": 2, "type": "nat", "hook": "output", "prio": -100, "policy": "accept"}}, {"chain": {"family": "ip", "table": "nat", "name": "DOCKER_POSTROUTING", "handle": 4}}, {"chain": {"family": "ip", "table": "nat", "name": "POSTROUTING", "handle": 5, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "ip", "table": "nat", "chain": "DOCKER_OUTPUT", "handle": 9, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "127.0.0.11"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 53}}, {"counter": {"packets": 0, "bytes": 0}}, {"xt": {"type": "target", "name": "DNAT"}}]}}, {"rule": {"family": "ip", "table": "nat", "chain": "DOCKER_OUTPUT", "handle": 7, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "127.0.0.11"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "dport"}}, "right": 53}}, {"counter": {"packets": 0, "bytes": 0}}, {"xt": {"type": "target", "name": "DNAT"}}]}}, {"rule": {"family": "ip", "table": "nat", "chain": "OUTPUT", "handle": 3, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "127.0.0.11"}}, {"counter": {"packets": 0, "bytes": 0}}, {"jump": {"target": "DOCKER_OUTPUT"}}]}}, {"rule": {"family": "ip", "table": "nat", "chain": "DOCKER_POSTROUTING", "handle": 10, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "127.0.0.11"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "sport"}}, "right": 46383}}, {"counter": {"packets": 0, "bytes": 0}}, {"xt": {"type": "target", "name": "SNAT"}}]}}, {"rule": {"family": "ip", "table": "nat", "chain": "DOCKER_POSTROUTING", "handle": 8, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "127.0.0.11"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "sport"}}, "right": 33185}}, {"counter": {"packets": 0, "bytes": 0}}, {"xt": {"type": "target", "name": "SNAT"}}]}}, {"rule": {"family": "ip", "table": "nat", "chain": "POSTROUTING", "handle": 6, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "127.0.0.11"}}, {"counter": {"packets": 0, "bytes": 0}}, {"jump": {"target": "DOCKER_POSTROUTING"}}]}}]}`

func TestFollowup11_DockerDNSLoopbackScoped(t *testing.T) {
	raw, m, assign := servingPlus(t, nil)
	raw = appendRawRules(t, raw, []byte(dockerDNSListing))
	if err := verifyServingRuleset(raw, m, assign); err != nil {
		t.Fatalf("embedded DNS NAT rejected: %v", err)
	}
	ev := evidenceFrom(t, raw)
	want := []struct {
		chain, hook, match string
		handle             uint64
	}{
		{"DOCKER_OUTPUT", "output", "ip daddr 127.0.0.11", 9},
		{"DOCKER_OUTPUT", "output", "ip daddr 127.0.0.11", 7},
		{"OUTPUT", "output", "ip daddr 127.0.0.11", 3},
		{"DOCKER_POSTROUTING", "postrouting", "ip saddr 127.0.0.11", 10},
		{"DOCKER_POSTROUTING", "postrouting", "ip saddr 127.0.0.11", 8},
		{"POSTROUTING", "postrouting", "ip daddr 127.0.0.11", 6},
	}
	if len(ev) != len(want) {
		t.Fatalf("evidence %d: %+v", len(ev), ev)
	}
	for i, w := range want {
		got := ev[i]
		if got.Class != LoopbackScopedClass || got.Family != "ip" || got.Table != "nat" || got.Chain != w.chain || got.Hook != w.hook || got.Match != w.match || !got.HasHandle || got.Handle != w.handle {
			t.Fatalf("evidence[%d] = %+v, want chain %s hook %s handle %d match %s", i, got, w.chain, w.hook, w.handle, w.match)
		}
	}

	t.Run("renamed_chain", func(t *testing.T) {
		raw, m, assign := servingPlus(t, append(crossedChain("ip", "other", "post", "postrouting", 100),
			ruleObj("ip", "other", "post", 4, []any{
				ipMatch("ip", "daddr", "127.0.0.11"),
				xtTarget("DNAT"),
			}),
		))
		if err := verifyServingRuleset(raw, m, assign); err != nil {
			t.Fatalf("renamed chain rejected: %v", err)
		}
		ev := evidenceFrom(t, raw)
		if len(ev) != 1 || ev[0].Table != "other" || ev[0].Chain != "post" || ev[0].Class != LoopbackScopedClass || ev[0].Match != "ip daddr 127.0.0.11" {
			t.Fatalf("evidence %+v", ev)
		}
	})
}

func TestFollowup11_NonLoopbackMasquerade(t *testing.T) {
	nonLoop := ipMatch("ip", "saddr", map[string]any{"prefix": map[string]any{"addr": "172.20.0.0", "len": 16}})

	t.Run("base_chain_xt_snat", func(t *testing.T) {
		raw, m, assign := servingPlus(t, nil)
		raw = appendRawRules(t, raw, []byte(dockerDNSListing))
		raw = appendNft(raw, ruleObj("ip", "nat", "POSTROUTING", 30, []any{nonLoop, xtTarget("SNAT")}))
		requireBypass(t, raw, m, assign, "table nat", "family ip", "hook postrouting", "priority 100", "chain POSTROUTING", "statement xt", "handle 30")
	})

	t.Run("jumped_chain_masquerade", func(t *testing.T) {
		raw, m, assign := servingPlus(t, nil)
		raw = appendRawRules(t, raw, []byte(dockerDNSListing))
		raw = appendNft(raw, ruleObj("ip", "nat", "DOCKER_POSTROUTING", 31, []any{nonLoop, xtTarget("MASQUERADE")}))
		requireBypass(t, raw, m, assign, "hook postrouting", "statement xt", "handle 31")
	})
}

func TestFollowup11_XTMatchDoesNotProve(t *testing.T) {
	raw, m, assign := servingPlus(t, append(crossedChain("ip", "nat", "POSTROUTING", "postrouting", 100),
		ruleObj("ip", "nat", "POSTROUTING", 5, []any{
			map[string]any{"xt": map[string]any{"type": "match", "name": "addrtype", "daddr": "127.0.0.11"}},
			xtTarget("SNAT"),
		}),
	))
	requireBypass(t, raw, m, assign, "statement xt", "handle 5")
	if ev := evidenceFrom(t, raw); len(ev) != 0 {
		t.Fatalf("xt match proved scope: %+v", ev)
	}
}

func TestFollowup11_ScopedBesideUnscoped(t *testing.T) {
	raw, m, assign := servingPlus(t, append(crossedChain("ip", "fvobs", "post", "postrouting", 100),
		ruleObj("ip", "fvobs", "post", 11, []any{
			ipMatch("ip", "daddr", "127.0.0.11"),
			xtTarget("SNAT"),
		}),
		ruleObj("ip", "fvobs", "post", 22, []any{
			ipMatch("ip", "saddr", map[string]any{"prefix": map[string]any{"addr": "10.1.2.0", "len": 24}}),
			xtTarget("SNAT"),
		}),
	))
	err := requireBypass(t, raw, m, assign, "table fvobs", "hook postrouting", "chain post", "statement xt", "handle 22")
	if strings.Contains(err, "handle 11") {
		t.Fatalf("verdict named the scoped rule: %s", err)
	}
}

func TestFollowup11_JumpToScopedChain(t *testing.T) {
	raw, m, assign := servingPlus(t, append(crossedChain("ip", "nat", "POSTROUTING", "postrouting", 100),
		ruleObj("ip", "nat", "POSTROUTING", 1, []any{
			map[string]any{"jump": map[string]any{"target": "INNER"}},
		}),
		map[string]any{"chain": map[string]any{"family": "ip", "table": "nat", "name": "INNER"}},
		ruleObj("ip", "nat", "INNER", 2, []any{
			ipMatch("ip", "saddr", "127.0.0.11"),
			xtTarget("SNAT"),
		}),
	))
	if err := verifyServingRuleset(raw, m, assign); err != nil {
		t.Fatalf("jump into a loopback-scoped chain rejected: %v", err)
	}
	ev := evidenceFrom(t, raw)
	if len(ev) != 1 || ev[0].Chain != "INNER" || ev[0].Hook != "postrouting" || ev[0].Match != "ip saddr 127.0.0.11" || !ev[0].HasHandle || ev[0].Handle != 2 || ev[0].Class != LoopbackScopedClass {
		t.Fatalf("evidence %+v", ev)
	}
}

func TestFollowup11_PrefixBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		family string
		expr   []any
		want   string
	}{
		{name: "v4_prefix_8", family: "ip", want: "ip daddr 127.0.0.0/8", expr: xtRule(ipMatch("ip", "daddr", prefix("127.0.0.0", 8)))},
		{name: "v4_host", family: "ip", want: "ip daddr 127.0.0.11", expr: xtRule(ipMatch("ip", "daddr", "127.0.0.11"))},
		{name: "v4_narrower_prefix", family: "ip", want: "ip daddr 127.1.0.0/16", expr: xtRule(ipMatch("ip", "daddr", prefix("127.1.0.0", 16)))},
		{name: "v6_loopback", family: "ip6", want: "ip6 saddr ::1", expr: xtRule(ipMatch("ip6", "saddr", "::1"))},
		{name: "v6_prefix_128", family: "ip6", want: "ip6 daddr ::1/128", expr: xtRule(ipMatch("ip6", "daddr", prefix("::1", 128)))},
		{name: "v4_range", family: "ip", want: "ip saddr 127.0.0.1-127.0.0.10", expr: xtRule(ipMatch("ip", "saddr", map[string]any{"range": []any{"127.0.0.1", "127.0.0.10"}}))},
		{name: "v4_range_object", family: "ip", want: "ip saddr 127.0.0.1-127.0.0.10", expr: xtRule(ipMatch("ip", "saddr", map[string]any{"range": map[string]any{"start": "127.0.0.1", "end": "127.0.0.10"}}))},
		{name: "v4_anonymous_set", family: "ip", want: "ip daddr {127.0.0.1, 127.0.0.11}", expr: xtRule(ipMatch("ip", "daddr", map[string]any{"set": []any{"127.0.0.1", "127.0.0.11"}}))},
		{name: "v4_set_of_prefixes", family: "ip", want: "ip daddr {127.0.0.0/8}", expr: xtRule(ipMatch("ip", "daddr", map[string]any{"set": []any{prefix("127.0.0.0", 8)}}))},
		{name: "op_omitted", family: "ip", want: "ip daddr 127.0.0.11", expr: xtRule(map[string]any{"match": map[string]any{"left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "daddr"}}, "right": "127.0.0.11"}})},
		{name: "l4proto_and_loopback", family: "ip", want: "ip daddr 127.0.0.11", expr: xtRule(
			map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "l4proto"}}, "right": "udp"}},
			ipMatch("ip", "daddr", "127.0.0.11"),
		)},
		{name: "port_and_loopback", family: "ip", want: "ip daddr 127.0.0.11", expr: xtRule(
			ipMatch("ip", "daddr", "127.0.0.11"),
			map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": "tcp", "field": "dport"}}, "right": 53}},
		)},
		{name: "v4_prefix_7", family: "ip", expr: xtRule(ipMatch("ip", "daddr", prefix("126.0.0.0", 7)))},
		{name: "v4_prefix_7_unmasked", family: "ip", expr: xtRule(ipMatch("ip", "daddr", prefix("127.0.0.0", 7)))},
		{name: "v4_default", family: "ip", expr: xtRule(ipMatch("ip", "saddr", prefix("0.0.0.0", 0)))},
		{name: "v4_range_outside", family: "ip", expr: xtRule(ipMatch("ip", "saddr", map[string]any{"range": []any{"126.255.255.255", "127.0.0.1"}}))},
		{name: "v4_set_mixed", family: "ip", expr: xtRule(ipMatch("ip", "daddr", map[string]any{"set": []any{"127.0.0.1", "10.0.0.1"}}))},
		{name: "v4_named_set", family: "ip", expr: xtRule(ipMatch("ip", "saddr", "@named"))},
		{name: "v4_named_set_object", family: "ip", expr: xtRule(ipMatch("ip", "saddr", map[string]any{"set": map[string]any{"name": "named"}}))},
		{name: "v4_not_equal", family: "ip", expr: xtRule(map[string]any{"match": map[string]any{"op": "!=", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "daddr"}}, "right": "127.0.0.11"}})},
		{name: "v6_prefix_127", family: "ip6", expr: xtRule(ipMatch("ip6", "daddr", prefix("::", 127)))},
		{name: "l4proto_alone", family: "ip", expr: xtRule(map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "l4proto"}}, "right": "udp"}})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, m, assign := servingPlus(t, append(crossedChain(tc.family, "nat", "POSTROUTING", "postrouting", 100),
				ruleObj(tc.family, "nat", "POSTROUTING", 1, tc.expr),
			))
			ev := evidenceFrom(t, raw)
			if tc.want == "" {
				requireBypass(t, raw, m, assign, "statement xt", "handle 1")
				if len(ev) != 0 {
					t.Fatalf("proved scope: %+v", ev)
				}
				return
			}
			if err := verifyServingRuleset(raw, m, assign); err != nil {
				t.Fatalf("rejected: %v", err)
			}
			if len(ev) != 1 || ev[0].Class != LoopbackScopedClass || ev[0].Match != tc.want || ev[0].Hook != "postrouting" || ev[0].Family != tc.family {
				t.Fatalf("evidence %+v, want %s", ev, tc.want)
			}
		})
	}
}

func servingPlus(t *testing.T, objects []map[string]any) ([]byte, Model, map[uuid.UUID]uint16) {
	t.Helper()
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	raw := fixtureServing(m, plan.Assignments)
	for _, obj := range objects {
		raw = appendNft(raw, obj)
	}
	return raw, m, plan.Assignments
}

func evidenceFrom(t *testing.T, raw []byte) []ForeignEvidence {
	t.Helper()
	rs, err := ParseRuleset(raw)
	if err != nil {
		t.Fatal(err)
	}
	return LoopbackEvidence(rs)
}

func requireBypass(t *testing.T, raw []byte, m Model, assign map[uuid.UUID]uint16, parts ...string) string {
	t.Helper()
	err := verifyServingRuleset(raw, m, assign)
	if ReasonCode(err) != ReasonFlowtable {
		t.Fatalf("got %v, want FLOWTABLE_BYPASS", err)
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("verdict %q missing %q", err.Error(), part)
		}
	}
	return err.Error()
}

func appendRawRules(t *testing.T, base, extra []byte) []byte {
	t.Helper()
	var doc struct {
		Nftables []json.RawMessage `json:"nftables"`
	}
	if err := json.Unmarshal(base, &doc); err != nil {
		t.Fatal(err)
	}
	var more struct {
		Nftables []json.RawMessage `json:"nftables"`
	}
	if err := json.Unmarshal(extra, &more); err != nil {
		t.Fatal(err)
	}
	doc.Nftables = append(doc.Nftables, more.Nftables...)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func crossedChain(family, table, name, hook string, prio int) []map[string]any {
	return []map[string]any{{"chain": map[string]any{
		"family": family, "table": table, "name": name,
		"type": "nat", "hook": hook, "prio": prio, "policy": "accept",
	}}}
}

func ruleObj(family, table, chain string, handle int, expr []any) map[string]any {
	return map[string]any{"rule": map[string]any{
		"family": family, "table": table, "chain": chain, "handle": handle, "expr": expr,
	}}
}

func ipMatch(proto, field string, right any) map[string]any {
	return map[string]any{"match": map[string]any{
		"op":    "==",
		"left":  map[string]any{"payload": map[string]any{"protocol": proto, "field": field}},
		"right": right,
	}}
}

func xtTarget(name string) map[string]any {
	return map[string]any{"xt": map[string]any{"type": "target", "name": name}}
}

func xtRule(stmts ...any) []any {
	return append(stmts, xtTarget("SNAT"))
}

func prefix(addr string, length int) map[string]any {
	return map[string]any{"prefix": map[string]any{"addr": addr, "len": length}}
}
