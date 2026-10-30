package enforce

import (
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
	if err := VerifyServing(base, m, plan.Assignments); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	member := m.Datapath.MemberIngress
	uplink := m.Datapath.Uplink

	t.Run("return_verdict", func(t *testing.T) {
		raw := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{map[string]any{"return": nil}},
		})
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
			t.Fatal("return verdict accepted")
		}
	})

	t.Run("policy_accept_on_forward", func(t *testing.T) {
		raw := mutateChainPolicy(t, base, "forward", "accept")
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
			t.Fatal("forward policy accept accepted")
		}
	})

	t.Run("extra_base_chain_inside_fvpn", func(t *testing.T) {
		raw := appendNft(base, map[string]any{"chain": map[string]any{
			"family": "inet", "table": TableName, "name": "prerouting",
			"type": "filter", "hook": "prerouting", "prio": -200, "policy": "accept",
		}})
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
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
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
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
		if err := VerifyServing(raw, m, plan.Assignments); err != nil {
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
		if err := VerifyServing(raw, m, plan.Assignments); err != nil {
			t.Fatalf("renderer-shaped nfproto ipv6 drop rejected: %v", err)
		}
	})

	t.Run("allowed_v4_interval_element", func(t *testing.T) {
		raw := replaceSet(t, base, "allowed_v4", map[string]any{
			"family": "inet", "table": TableName, "name": "allowed_v4",
			"elem": []any{map[string]any{"prefix": map[string]any{"addr": "100.64.0.0", "len": 10}}},
		})
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
			t.Fatal("allowed_v4 prefix element accepted")
		}
	})

	t.Run("allowed_v4_interval_flag", func(t *testing.T) {
		raw := replaceSet(t, base, "allowed_v4", map[string]any{
			"family": "inet", "table": TableName, "name": "allowed_v4",
			"flags": []string{"interval"},
		})
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
			t.Fatal("allowed_v4 interval flag accepted")
		}
	})

	t.Run("allowed_v4_timeout_flag", func(t *testing.T) {
		raw := replaceSet(t, base, "allowed_v4", map[string]any{
			"family": "inet", "table": TableName, "name": "allowed_v4",
			"flags": []string{"timeout"},
		})
		if err := VerifyServing(raw, m, plan.Assignments); err != nil {
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
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
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
		if err := VerifyServing(raw, m, plan.Assignments); err != nil {
			t.Fatalf("extra ip version match on a device accept rejected: %v", err)
		}
	})

	t.Run("named_counter_rebind_does_not_add_an_accept", func(t *testing.T) {
		raw := appendNft(base, map[string]any{"counter": map[string]any{
			"family": "inet", "table": TableName, "name": usageCounter(dirUpload, m.Bindings[0].DeviceID),
			"packets": 999, "bytes": 999,
		}})
		if err := VerifyServing(raw, m, plan.Assignments); err != nil {
			t.Fatalf("rebinding a named counter's packet count rejected the ruleset: %v", err)
		}
		bare := prependRule(t, base, "forward", map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"counter": usageCounter(dirUpload, m.Bindings[0].DeviceID)},
				map[string]any{"accept": nil},
			},
		})
		if err := VerifyServing(bare, m, plan.Assignments); err == nil {
			t.Fatal("accept that only names the upload counter was accepted")
		}
	})

	t.Run("second_table_higher_priority_accept", func(t *testing.T) {
		// Read-back is `nft -j list table inet fvpn` (apply.go). A second
		// table is absent from those bytes. This listing includes it anyway:
		// the parser drops every object whose table is not fvpn, so a
		// higher-priority accept hook is invisible when it is present too.
		raw := appendNft(base, map[string]any{"chain": map[string]any{
			"family": "inet", "table": "not_fvpn", "name": "forward",
			"type": "filter", "hook": "forward", "prio": -200, "policy": "accept",
		}})
		raw = appendNft(raw, map[string]any{"rule": map[string]any{
			"family": "inet", "table": "not_fvpn", "chain": "forward",
			"expr": []any{map[string]any{"accept": nil}},
		}})
		if err := VerifyServing(raw, m, plan.Assignments); err == nil {
			t.Errorf("TODO(review): VerifyServing accepted inet table not_fvpn with a forward hook at priority -200 and policy accept")
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
