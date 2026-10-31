package enforce

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestReviewR14_VerifierModel starts from the serving fixture VerifyServing
// accepts and adds one out-of-model construct per subtest. A subtest that
// expects rejection fails the test when VerifyServing returns nil.
func TestReviewR14_VerifierModel(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	base := fixtureServing(m, plan.Assignments)
	if err := verifyServingRuleset(base, m, plan.Assignments); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	member := m.Datapath.MemberIngress
	uplink := m.Datapath.Uplink

	t.Run("return_verdict", func(t *testing.T) {
		raw := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{map[string]any{"return": nil}},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("return verdict accepted")
		}
	})

	t.Run("policy_accept_on_forward", func(t *testing.T) {
		raw := mutateChainPolicy(t, base, "forward", "accept")
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("forward policy accept accepted")
		}
	})

	t.Run("extra_base_chain_inside_fvpn", func(t *testing.T) {
		raw := appendNft(base, map[string]any{"chain": map[string]any{
			"family": "inet", "table": TableName, "name": "prerouting",
			"type": "filter", "hook": "prerouting", "prio": -200, "policy": "accept",
		}})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("extra base chain inside inet fvpn accepted")
		}
	})

	t.Run("ct_state_established_accept_on_forward", func(t *testing.T) {
		raw := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"ct": map[string]any{"key": "state"}}, "right": "established"}},
				map[string]any{"accept": nil},
			},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("ct state established accept on forward accepted")
		}
	})

	t.Run("renderer_ct_state_invalid_drop", func(t *testing.T) {
		raw := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": member}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": uplink}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"ct": map[string]any{"key": "state"}}, "right": "invalid"}},
				map[string]any{"counter": "ctinv"},
				map[string]any{"drop": nil},
			},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
			t.Fatalf("renderer-shaped ct state invalid drop rejected: %v", err)
		}
	})

	t.Run("renderer_nfproto_ipv6_drop", func(t *testing.T) {
		raw := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": member}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": uplink}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "nfproto"}}, "right": "ipv6"}},
				map[string]any{"counter": "ctv6"},
				map[string]any{"drop": nil},
			},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
			t.Fatalf("renderer-shaped nfproto ipv6 drop rejected: %v", err)
		}
	})

	t.Run("allowed_v4_interval_element", func(t *testing.T) {
		raw := replaceSet(t, base, "allowed_v4", map[string]any{
			"family": "inet", "table": TableName, "name": "allowed_v4",
			"elem": []any{map[string]any{"prefix": map[string]any{"addr": "100.64.0.0", "len": 10}}},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("allowed_v4 prefix element accepted")
		}
	})

	t.Run("allowed_v4_interval_flag", func(t *testing.T) {
		raw := replaceSet(t, base, "allowed_v4", map[string]any{
			"family": "inet", "table": TableName, "name": "allowed_v4",
			"flags": []string{"interval"},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("allowed_v4 interval flag accepted")
		}
	})

	t.Run("allowed_v4_timeout_flag", func(t *testing.T) {
		raw := replaceSet(t, base, "allowed_v4", map[string]any{
			"family": "inet", "table": TableName, "name": "allowed_v4",
			"flags": []string{"timeout"},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
			t.Fatalf("renderer timeout flag rejected: %v", err)
		}
	})

	t.Run("nfproto_ipv6_on_accept", func(t *testing.T) {
		raw := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "nfproto"}}, "right": "ipv6"}},
				map[string]any{"accept": nil},
			},
		})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatal("meta nfproto ipv6 accept accepted")
		}
	})

	t.Run("ip_version_does_not_widen_device_accept", func(t *testing.T) {
		doc := unmarshalRuleset(t, base)
		items := doc["nftables"].([]any)
		for _, item := range items {
			rule, ok := item.(map[string]any)["rule"].(map[string]any)
			if !ok || rule["chain"] != "forward" {
				continue
			}
			expr, _ := rule["expr"].([]any)
			joined, _ := json.Marshal(expr)
			if !strings.Contains(string(joined), `"accept"`) || !strings.Contains(string(joined), `"saddr"`) {
				continue
			}
			rule["expr"] = append([]any{map[string]any{"match": map[string]any{
				"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "version"}}, "right": 4,
			}}}, expr...)
			break
		}
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
			t.Fatalf("extra ip version match on a device accept rejected: %v", err)
		}
	})

	t.Run("named_counter_rebind_does_not_add_an_accept", func(t *testing.T) {
		raw := appendNft(base, map[string]any{"counter": map[string]any{
			"family": "inet", "table": TableName, "name": usageCounter(dirUpload, m.Bindings[0].DeviceID),
			"packets": 999, "bytes": 999,
		}})
		if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
			t.Fatalf("rebinding a named counter's packet count rejected the ruleset: %v", err)
		}
		bare := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"counter": usageCounter(dirUpload, m.Bindings[0].DeviceID)},
				map[string]any{"accept": nil},
			},
		})
		if err := verifyServingRuleset(bare, m, plan.Assignments); err == nil {
			t.Fatal("accept that only names the upload counter was accepted")
		}
	})

	t.Run("second_table_higher_priority_accept", func(t *testing.T) {
		// An accept in an earlier base chain does not stop later base chains.
		// inet fvpn forward policy drop still applies, so this listing is in
		// the verified bytes and is tolerated. The parser must keep the chain
		// rather than drop it.
		raw := appendNft(base, map[string]any{"chain": map[string]any{
			"family": "inet", "table": "not_fvpn", "name": "forward",
			"type": "filter", "hook": "forward", "prio": -200, "policy": "accept",
		}})
		raw = appendNft(raw, map[string]any{"rule": map[string]any{
			"family": "inet", "table": "not_fvpn", "chain": "forward",
			"expr": []any{map[string]any{"accept": nil}},
		}})
		parsed, err := ParseRuleset(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(parsed.Foreign) != 1 || parsed.Foreign[0].Table != "not_fvpn" || parsed.Foreign[0].Hook != "forward" {
			t.Fatalf("foreign accept chain was not kept: %+v", parsed.Foreign)
		}
		if _, merged := parsed.Chains["not_fvpn"]; merged {
			t.Fatal("foreign chain merged into the inet fvpn chain map")
		}
		if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
			t.Fatalf("accept-only table at priority -200 rejected: %v", err)
		}

		// A payload rewrite in an earlier prerouting chain changes the address
		// inet fvpn matches. That listing fails closed.
		mangled := appendNft(base, map[string]any{"chain": map[string]any{
			"family": "inet", "table": "not_fvpn", "name": "pre",
			"type": "filter", "hook": "prerouting", "prio": -200, "policy": "accept",
		}})
		mangled = appendNft(mangled, map[string]any{"rule": map[string]any{
			"family": "inet", "table": "not_fvpn", "chain": "pre",
			"expr": []any{map[string]any{"mangle": map[string]any{
				"key":   map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}},
				"value": "100.64.1.1",
			}}},
		}})
		err = verifyServingRuleset(mangled, m, plan.Assignments)
		if ReasonCode(err) != ReasonIsolation {
			t.Fatalf("ip saddr set: %v", err)
		}
		for _, part := range []string{"table not_fvpn", "family inet", "hook prerouting", "priority -200", "statement mangle"} {
			if !strings.Contains(err.Error(), part) {
				t.Fatalf("verdict %q missing %q", err.Error(), part)
			}
		}
	})
}

func prependRule(t *testing.T, raw []byte, chain string, rule map[string]any) []byte {
	t.Helper()
	doc := unmarshalRuleset(t, raw)
	prependChainRule(doc, chain, rule)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func unmarshalRuleset(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func mutateChainPolicy(t *testing.T, raw []byte, name, policy string) []byte {
	t.Helper()
	doc := unmarshalRuleset(t, raw)
	for _, item := range doc["nftables"].([]any) {
		ch, ok := item.(map[string]any)["chain"].(map[string]any)
		if ok && ch["name"] == name && ch["table"] == TableName {
			ch["policy"] = policy
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func replaceSet(t *testing.T, raw []byte, name string, set map[string]any) []byte {
	t.Helper()
	doc := unmarshalRuleset(t, raw)
	items := doc["nftables"].([]any)
	replaced := false
	for i, item := range items {
		obj, ok := item.(map[string]any)["set"].(map[string]any)
		if ok && obj["name"] == name {
			items[i] = map[string]any{"set": set}
			replaced = true
		}
	}
	if !replaced {
		t.Fatalf("set %s not in fixture", name)
	}
	doc["nftables"] = items
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestFollowup10_ForeignChains is the tolerance rule for base chains outside
// inet fvpn. Safe statements and hooks member traffic does not cross are
// accepted. A statement that can change what inet fvpn sees or decides is
// rejected with table, family, hook, priority, and statement.
func TestFollowup10_ForeignChains(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	base := fixtureServing(m, plan.Assignments)
	accept := []any{map[string]any{"accept": nil}}
	counterLog := []any{
		map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
		map[string]any{"counter": "ctobs"},
		map[string]any{"log": map[string]any{"prefix": "fvpn-obs"}},
		map[string]any{"continue": nil},
	}
	saddr := []any{map[string]any{"mangle": map[string]any{
		"key":   map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}},
		"value": "100.64.1.1",
	}}}
	metaMark := []any{map[string]any{"mangle": map[string]any{
		"key":   map[string]any{"meta": map[string]any{"key": "mark"}},
		"value": 1,
	}}}
	ctMark := []any{map[string]any{"mangle": map[string]any{
		"key":   map[string]any{"ct": map[string]any{"key": "mark"}},
		"value": 1,
	}}}

	cases := []struct {
		name    string
		objects []map[string]any
		reason  Reason
		parts   []string
	}{
		{
			name:    "counter_log_match_continue",
			objects: foreignObjects("inet", "fvobs", "early", "forward", -200, "accept", "", counterLog),
		},
		{
			name:    "input_drop_on_host_port",
			objects: append(foreignObjects("inet", "fvobs", "in", "input", 0, "drop", "", []any{map[string]any{"drop": nil}}), foreignObjects("ip", "filter", "out", "output", 0, "accept", "", saddr)...),
		},
		{
			name:    "netdev_mangle_on_other_device",
			objects: foreignObjects("netdev", "fvobs", "ing", "ingress", -200, "accept", "fv-other", saddr),
		},
		{
			name: "jump_to_accept_chain",
			objects: append(
				foreignObjects("inet", "fvobs", "early", "prerouting", -200, "accept", "", []any{map[string]any{"jump": map[string]any{"target": "ok"}}}),
				foreignObjects("inet", "fvobs", "ok", "", 0, "", "", accept)...,
			),
		},
		{
			name:    "later_forward_drop",
			objects: foreignObjects("inet", "fvobs", "late", "forward", 0, "accept", "", []any{map[string]any{"drop": nil}}),
			reason:  ReasonIsolation,
			parts:   []string{"table fvobs", "family inet", "hook forward", "priority 0", "statement drop"},
		},
		{
			name:    "policy_drop",
			objects: foreignObjects("ip6", "fvobs", "fwd", "forward", -200, "drop", "", nil),
			reason:  ReasonIsolation,
			parts:   []string{"table fvobs", "family ip6", "hook forward", "priority -200", "statement drop"},
		},
		{
			name:    "meta_mark_set",
			objects: foreignObjects("inet", "fvobs", "early", "forward", -200, "accept", "", metaMark),
			reason:  ReasonMark,
			parts:   []string{"table fvobs", "family inet", "hook forward", "priority -200", "statement mark"},
		},
		{
			name:    "ct_mark_set",
			objects: foreignObjects("inet", "fvobs", "early", "postrouting", 50, "accept", "", ctMark),
			reason:  ReasonMark,
			parts:   []string{"hook postrouting", "priority 50", "statement mark"},
		},
		{
			name:    "notrack",
			objects: foreignObjects("inet", "fvobs", "early", "prerouting", -300, "accept", "", []any{map[string]any{"notrack": nil}}),
			reason:  ReasonIsolation,
			parts:   []string{"statement notrack", "priority -300"},
		},
		{
			name:    "queue",
			objects: foreignObjects("bridge", "fvobs", "early", "forward", -200, "accept", "", []any{map[string]any{"queue": map[string]any{"num": 0}}}),
			reason:  ReasonIsolation,
			parts:   []string{"family bridge", "statement queue"},
		},
		{
			name:    "snat",
			objects: foreignObjects("ip", "fvobs", "post", "postrouting", 80, "accept", "", []any{map[string]any{"snat": map[string]any{"addr": "192.0.2.37"}}}),
			reason:  ReasonIsolation,
			parts:   []string{"family ip", "statement snat"},
		},
		{
			name:    "flow_offload",
			objects: foreignObjects("inet", "fvobs", "early", "forward", -200, "accept", "", []any{map[string]any{"flow": map[string]any{"op": "add", "flowtable": "@fast"}}}),
			reason:  ReasonFlowtable,
			parts:   []string{"statement flow"},
		},
		{
			name: "jump_into_mangle",
			objects: append(
				foreignObjects("inet", "fvobs", "early", "prerouting", -200, "accept", "", []any{map[string]any{"goto": map[string]any{"target": "rewritten"}}}),
				foreignObjects("inet", "fvobs", "rewritten", "", 0, "", "", saddr)...,
			),
			reason: ReasonIsolation,
			parts:  []string{"hook prerouting", "priority -200", "statement mangle"},
		},
		{
			name:    "jump_target_missing",
			objects: foreignObjects("inet", "fvobs", "early", "forward", -200, "accept", "", []any{map[string]any{"jump": map[string]any{"target": "missing"}}}),
			reason:  ReasonIsolation,
			parts:   []string{"statement jump"},
		},
		{
			name: "jump_cycle",
			objects: append(
				foreignObjects("inet", "fvobs", "early", "forward", -200, "accept", "", []any{map[string]any{"jump": map[string]any{"target": "loop"}}}),
				foreignObjects("inet", "fvobs", "loop", "", 0, "", "", []any{map[string]any{"jump": map[string]any{"target": "early"}}})...,
			),
			reason: ReasonIsolation,
			parts:  []string{"statement jump"},
		},
		{
			name:    "netdev_mangle_on_member",
			objects: foreignObjects("netdev", "fvobs", "ing", "ingress", -200, "accept", m.Datapath.MemberIngress, saddr),
			reason:  ReasonIsolation,
			parts:   []string{"family netdev", "hook ingress", "statement mangle"},
		},
		{
			name:    "netdev_mangle_on_uplink_egress",
			objects: foreignObjects("netdev", "fvobs", "eg", "egress", 0, "accept", m.Datapath.Uplink, saddr),
			reason:  ReasonIsolation,
			parts:   []string{"hook egress", "statement mangle"},
		},
		{
			name:    "missing_priority",
			objects: foreignChainNoPrio("inet", "fvobs", "early", "forward", "accept", accept),
			reason:  ReasonIsolation,
			parts:   []string{"priority missing", "statement priority"},
		},
		{
			name:    "arp_drop_tolerated",
			objects: foreignObjects("arp", "fvobs", "fwd", "forward", -200, "drop", "", []any{map[string]any{"drop": nil}}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := base
			for _, obj := range tc.objects {
				raw = appendNft(raw, obj)
			}
			err := verifyServingRuleset(raw, m, plan.Assignments)
			if tc.reason == "" {
				if err != nil {
					t.Fatalf("tolerated chain rejected: %v", err)
				}
				return
			}
			if ReasonCode(err) != tc.reason {
				t.Fatalf("got %v, want %s", err, tc.reason)
			}
			for _, part := range tc.parts {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("verdict %q missing %q", err.Error(), part)
				}
			}
		})
	}
}

func TestApply_ForeignMangleInstallsBaselineInsideFvpn(t *testing.T) {
	m := smallModel()
	f := servingExec(m)
	inner := conntrackHandler(m, f, func(string) ([]byte, error) { return nil, nil })
	f.handler = func(name string, args []string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")
		if key == "nft -j list ruleset" {
			f.phase = "qos"
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				return nil, err
			}
			raw := fixtureServing(m, plan.Assignments)
			for _, obj := range foreignObjects("inet", "not_fvpn", "pre", "prerouting", -200, "accept", "", []any{
				map[string]any{"mangle": map[string]any{
					"key":   map[string]any{"payload": map[string]any{"protocol": "ip6", "field": "saddr"}},
					"value": "100.64.1.1",
				}},
			}) {
				raw = appendNft(raw, obj)
			}
			return raw, nil
		}
		return inner(name, args)
	}
	res, err := Apply(context.Background(), f, m, nil)
	if res.Verified || ReasonCode(err) != ReasonIsolation {
		t.Fatalf("verified=%v err=%v", res.Verified, err)
	}
	for _, part := range []string{"table not_fvpn", "family inet", "hook prerouting", "priority -200", "statement mangle"} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("verdict %q missing %q", err.Error(), part)
		}
	}
	if len(f.nft) < 2 || !strings.Contains(f.nft[len(f.nft)-1], "fvpn_baseline=1") {
		t.Fatal("baseline was not installed")
	}
	for _, script := range f.nft {
		if strings.Contains(script, "not_fvpn") {
			t.Fatal("nft batch names the foreign table")
		}
	}
}

func foreignObjects(family, table, name, hook string, prio int, policy, dev string, expr []any) []map[string]any {
	ch := map[string]any{
		"family": family, "table": table, "name": name, "policy": policy,
	}
	if hook != "" {
		ch["type"] = "filter"
		ch["hook"] = hook
		ch["prio"] = prio
	}
	if dev != "" {
		ch["dev"] = dev
	}
	out := []map[string]any{{"chain": ch}}
	if expr != nil {
		out = append(out, map[string]any{"rule": map[string]any{
			"family": family, "table": table, "chain": name, "expr": expr,
		}})
	}
	return out
}

func foreignChainNoPrio(family, table, name, hook, policy string, expr []any) []map[string]any {
	ch := map[string]any{
		"family": family, "table": table, "name": name,
		"type": "filter", "hook": hook, "policy": policy,
	}
	return []map[string]any{
		{"chain": ch},
		{"rule": map[string]any{"family": family, "table": table, "chain": name, "expr": expr}},
	}
}
