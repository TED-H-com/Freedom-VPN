package enforce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func requireCheck(t *testing.T, err error, want Reason) {
	t.Helper()
	var ce *CheckError
	if !errors.As(err, &ce) || ce.Reason != want {
		t.Fatalf("got %v, want reason %s", err, want)
	}
}

// TestReviewR13_Q1_VerifierAcceptsOutOfModelRulesetsAndQoS proves that after the
// seven rules in follow-up 7 (#172), VerifyServing, VerifyBaseline, and VerifyQoS
// still accept rulesets and tc trees that forward member traffic outside the model.
func TestReviewR13_Q1_VerifierAcceptsOutOfModelRulesetsAndQoS(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	baseJSON := fixtureServing(m, plan.Assignments)
	if err := VerifyServing(baseJSON, m, plan.Assignments); err != nil {
		t.Fatalf("baseline smallModel listing should pass VerifyServing: %v", err)
	}

	t.Run("jump_or_goto_into_unchecked_chain", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		// 1. VerifyServing checks expectChain("output", ...) header attributes,
		//    but NEVER inspects rs.Chains["output"].Rules!
		// 2. walkExpr (parse_nft.go:421-468) only records "accept" and "drop"
		//    into rule.Verdict, ignoring "jump" and "goto" (rule.Verdict == "").
		// 3. Prepending `iifname "fv-fh" jump output` (or `goto unlisted` where
		//    the target chain is jumped to at index 0 of `forward`) leaves
		//    rule.Verdict == "" in `forward`, bypassing every drop rule in `forward`.
		prependChainRule(doc, "output", map[string]any{
			"family": "inet", "table": TableName, "chain": "output",
			"expr": []any{map[string]any{"accept": nil}},
		})
		prependForwardRule(doc, map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"match": map[string]any{
					"op":    "==",
					"left":  map[string]any{"meta": map[string]any{"key": "iifname"}},
					"right": m.Datapath.MemberIngress,
				}},
				map[string]any{"jump": map[string]any{"target": "output"}},
			},
		})
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonIsolation)
	})

	t.Run("vmap_verdict_map_before_drops", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		// walkExpr ignores `vmap` structure and overwrites rv.Verdict with the last
		// nested verdict token ("drop"), or leaves rv.Verdict == "" when referencing
		// a named verdict map `@rogue_vmap`.
		prependForwardRule(doc, map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"vmap": map[string]any{
					"key": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "daddr"}},
					"data": map[string]any{"set": []any{
						[]any{map[string]any{"prefix": map[string]any{"addr": "0.0.0.0", "len": 0}}, map[string]any{"accept": nil}},
						[]any{"192.0.2.254", map[string]any{"drop": nil}},
					}},
				}},
			},
		})
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonIsolation)
	})

	t.Run("serving_accept_with_negated_dport_and_anonymous_set_or_wrong_prio", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		devID := m.Bindings[0].DeviceID
		minor := plan.Assignments[m.Bindings[0].MemberID]
		// 1. noteMatch (parse_nft.go:514-521, 558-566) ignores `op != "=="` on
		//    dport/sport and iifname/oifname, so `tcp dport != 53` records Dports=[53]
		//    and is classified as "dns-up", opening all non-53 TCP ports to
		//    @resolver_v4 before `ctbadns` drop!
		// 2. matchPrefix (parse_nft.go:595-615) ignores anonymous sets `{"set": [...]}`
		//    containing IP prefixes/addresses, leaving rule.Saddrs empty, and
		//    hostOrEmpty(rule.Saddrs) (verify.go:714) returns true when len == 0!
		//    Simultaneously, servingAcceptKind (verify.go:658) only checks
		//    `len(rule.Prios) == 1` without checking which HTB class priority is set.
		dnsUp := usageCounter(dirDNSUpload, devID)
		up := usageCounter(dirUpload, devID)
		doc["nftables"] = append(doc["nftables"].([]any),
			map[string]any{"counter": map[string]any{"family": "inet", "table": "fvpn", "name": dnsUp, "packets": 1, "bytes": 100}},
		)
		insertAfterUploadAllowDrop(doc, map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.Uplink}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}}, "right": "100.64.1.1"}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}},
				map[string]any{"match": map[string]any{"op": "!=", "left": map[string]any{"payload": map[string]any{"protocol": "tcp", "field": "dport"}}, "right": 53}},
				map[string]any{"counter": map[string]any{"name": dnsUp}},
				map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": ClassID(minor)}},
				map[string]any{"accept": nil},
			},
		})
		// Also append an upload rule matching an anonymous set `{ 0.0.0.0/0, 100.64.99.99 }`
		// with wrong HTB priority 0x0001fffe (1:fffe) and negated iifname/oifname.
		doc["nftables"] = append(doc["nftables"].([]any), map[string]any{
			"rule": map[string]any{
				"family": "inet", "table": TableName, "chain": "forward",
				"expr": []any{
					map[string]any{"match": map[string]any{"op": "!=", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
					map[string]any{"match": map[string]any{"op": "!=", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.Uplink}},
					map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}}, "right": map[string]any{"set": []any{map[string]any{"prefix": map[string]any{"addr": "0.0.0.0", "len": 0}}, "100.64.99.99"}}}},
					map[string]any{"counter": map[string]any{"name": up}},
					map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": uint64(0x0001fffe)}},
					map[string]any{"accept": nil},
				},
			},
		})
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonIsolation)
	})

	t.Run("comments_and_line_continuations_in_parsed_tc_class_text", func(t *testing.T) {
		// ParseClassJSON (parse_tc.go:65-67, 93-124) falls back to parseClassText
		// when tc output does not start with '[' or '{'.
		// In parseClassText:
		// 1. Inline `#` comments on a `class htb ...` line are tokenized by
		//    strings.Fields(line), and the loop overwrites `rate` and `ceil` with
		//    the last occurrence on the line (`# rate 25Mbit ceil 50Mbit`), hiding
		//    an actual 1Kbit throttle!
		// 2. Backslash `\` line continuations split the class line so any tokens
		//    on the continuation line (which does not start with "class htb ") are
		//    silently ignored.
		qos := fixtureQoS(m, plan.Assignments)
		minor := plan.Assignments[m.Bindings[0].MemberID]
		textClasses := strings.Join([]string{
			fmt.Sprintf("class htb 1:1 root rate %dbit ceil %dbit", m.QoS.Up.RateBPS, m.QoS.Up.CeilBPS),
			fmt.Sprintf("class htb 1:fffe parent 1:1 rate %dbit ceil %dbit", m.QoS.HostUp.RateBPS, m.QoS.HostUp.CeilBPS),
			fmt.Sprintf("class htb 1:%x parent 1:1 rate 1000bit ceil 1000bit \\ # rate %dbit ceil %dbit", minor, m.QoS.Members[0].Up.RateBPS, m.QoS.Members[0].Up.CeilBPS),
			"  rate 999999999bit ceil 999999999bit",
		}, "\n")
		upPair := qos[m.Datapath.Uplink]
		upPair[1] = []byte(textClasses)
		qos[m.Datapath.Uplink] = upPair
		requireCheck(t, VerifyQoS(qos, m, plan.Assignments), ReasonQoS)
	})

	t.Run("meta_l4proto_narrowing_on_isolation_drops", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		// Narrowing @denied_v4, @host_v4, and @resolver_v4 drop rules with
		// `meta l4proto icmp` leaves TCP and UDP to denied/host/resolver prefixes
		// completely open while satisfying verifyIsolationDrops.
		for _, item := range doc["nftables"].([]any) {
			rm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			r, ok := rm["rule"].(map[string]any)
			if !ok || r["chain"] != "forward" {
				continue
			}
			raw, _ := json.Marshal(r)
			if strings.Contains(string(raw), `"@denied_v4"`) || strings.Contains(string(raw), `"@host_v4"`) || strings.Contains(string(raw), `"@resolver_v4"`) {
				expr := r["expr"].([]any)
				expr = append([]any{
					map[string]any{"match": map[string]any{
						"op":    "==",
						"left":  map[string]any{"meta": map[string]any{"key": "l4proto"}},
						"right": "icmp",
					}},
				}, expr...)
				r["expr"] = expr
			}
		}
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonIsolation)
	})

	t.Run("flow_add_offload_bypass_in_json_ruleset", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		// rejectBypass (verify.go:532-548) only checks case-sensitive substring
		// `"flowtable"` in rs.Raw and `"offload"` in rv.Raw, while walkExpr does
		// not inspect `{"flow": ...}` or `{"xt": {"name": "FLOWOFFLOAD"}}` expressions.
		prependForwardRule(doc, map[string]any{
			"family": "inet", "table": TableName, "chain": "forward",
			"expr": []any{
				map[string]any{"flow": map[string]any{"op": "add", "table": "@fastpath"}},
				map[string]any{"xt": map[string]any{"type": "target", "name": "FLOWOFFLOAD"}},
			},
		})
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonFlowtable)
	})

	t.Run("meta_mark_match_disables_isolation_drop_rules", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		// rejectBypass checks rule.SetsMark, and servingAcceptKind checks
		// rule.MatchMark on accept rules, but nothing checks rule.MatchMark on
		// drop rules! Conditioning @denied_v4, @host_v4, @resolver_v4, and
		// != @allowed_v4 drop rules on `meta mark == 0xdeadbeef` disables all of
		// them for normal unmarked packets.
		for _, item := range doc["nftables"].([]any) {
			rm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			r, ok := rm["rule"].(map[string]any)
			if !ok || r["chain"] != "forward" {
				continue
			}
			raw, _ := json.Marshal(r)
			if strings.Contains(string(raw), `"drop"`) && (strings.Contains(string(raw), `"@allowed_v4"`) || strings.Contains(string(raw), `"@denied_v4"`) || strings.Contains(string(raw), `"@host_v4"`) || strings.Contains(string(raw), `"@resolver_v4"`)) {
				expr := r["expr"].([]any)
				expr = append([]any{
					map[string]any{"match": map[string]any{
						"op":    "==",
						"left":  map[string]any{"meta": map[string]any{"key": "mark"}},
						"right": 0xdeadbeef,
					}},
				}, expr...)
				r["expr"] = expr
			}
		}
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonIsolation)
	})

	t.Run("qos_wrong_fq_codel_parent_major", func(t *testing.T) {
		// In VerifyQoS, attach fq_codel to parent "99:<minor>" instead of "1:<minor>".
		// splitHandle(parent) in ParseQdiscJSON (parse_tc.go:46) discards the major
		// handle and only records `view.Leaves[minor] = true`.
		qos := fixtureQoS(m, plan.Assignments)
		minor := plan.Assignments[m.Bindings[0].MemberID]
		oldParent := fmt.Sprintf(`"parent":"%s"`, classHandle(minor))
		newParent := fmt.Sprintf(`"parent":"99:%x"`, minor)
		for iface, pair := range qos {
			pair[0] = bytesReplace(pair[0], []byte(oldParent), []byte(newParent), 1)
			qos[iface] = pair
		}
		requireCheck(t, VerifyQoS(qos, m, plan.Assignments), ReasonQoS)
	})

	t.Run("upload_priority_is_not_the_assigned_class", func(t *testing.T) {
		// The bundled case above is tainted before the class check. This one
		// keeps a correct upload rule and adds a second accept for the same
		// counter at the host default class, so ruleBinds still matches.
		var doc map[string]any
		if err := json.Unmarshal(baseJSON, &doc); err != nil {
			t.Fatal(err)
		}
		devID := m.Bindings[0].DeviceID
		addr := m.Bindings[0].AllowedV4[0].Addr().String()
		up := usageCounter(dirUpload, devID)
		doc["nftables"] = append(doc["nftables"].([]any), map[string]any{
			"rule": map[string]any{
				"family": "inet", "table": TableName, "chain": "forward",
				"expr": []any{
					map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
					map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.Uplink}},
					map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}}, "right": addr}},
					map[string]any{"counter": map[string]any{"name": up}},
					map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": uint64(0x0001fffe)}},
					map[string]any{"accept": nil},
				},
			},
		})
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		requireCheck(t, VerifyServing(mutated, m, plan.Assignments), ReasonQoS)
	})
}

// TestReviewR13_Q1_Q2_WhatHeld verifies the Q1 and Q2 properties that held in
// node/enforce:
//   - Apply loads state with `nft -j list table inet fvpn`, then reads the
//     whole ruleset with `nft -j list ruleset` for VerifyServing. It installs
//     nftables before VerifyServing, applyQoS, VerifyQoS, and deleteConntrack.
//   - In the verified serving ruleset, a revoked IPv4 address removed from @allowed_v4
//     and device rules is dropped before any accept or established/related rule in
//     upload, download/reply, and input directions.
//   - Samples keys counters by DeviceID, so a different DeviceID assigned the same
//     IPv4 does not inherit the old DeviceID's nftables counter objects.
func TestReviewR13_Q1_Q2_WhatHeld(t *testing.T) {
	m := smallModel()
	m.Revoke = []netip.Addr{netip.MustParseAddr("100.64.9.9")}
	f := servingExec(m)
	var cmds []string
	conntrackH := conntrackHandler(m, f, func(string) ([]byte, error) { return nil, nil })
	f.handler = func(name string, args []string) ([]byte, error) {
		cmds = append(cmds, name+" "+strings.Join(args, " "))
		return conntrackH(name, args)
	}
	res, err := Apply(context.Background(), f, m, nil)
	if err != nil || !res.Verified {
		t.Fatalf("Apply: verified=%v err=%v", res.Verified, err)
	}

	if len(f.nft) != 1 || len(f.tc) != 1 {
		t.Fatalf("expected 1 nft batch and 1 tc batch, got nft=%d tc=%d", len(f.nft), len(f.tc))
	}
	hasTableList := false
	hasRuleset := false
	for _, c := range cmds {
		if c == "nft -j list table inet fvpn" {
			hasTableList = true
		}
		if c == "nft -j list ruleset" {
			hasRuleset = true
		}
	}
	if !hasTableList || !hasRuleset {
		t.Fatalf("expected loadState `nft -j list table inet fvpn` and verify `nft -j list ruleset`, got %v", cmds)
	}
	lastTwo := cmds[len(cmds)-2:]
	if lastTwo[0] != "conntrack -D -s 100.64.9.9" || lastTwo[1] != "conntrack -D -d 100.64.9.9" {
		t.Fatalf("expected conntrack deletes after QoS verification, got %v", lastTwo)
	}

	// Verify that when IP 100.64.1.1 is reassigned from dev1 to dev2, Samples
	// does not attribute dev1's lingering nftables counters to dev2.
	dev1 := m.Bindings[0].DeviceID
	dev2 := uuid.MustParse("00000000-0000-7000-8000-000000000099")
	mReassigned := m
	b2 := m.Bindings[0]
	b2.DeviceID = dev2
	b2.MemberID = uuid.MustParse("00000000-0000-7000-8000-000000000088")
	mReassigned.Bindings = []Binding{b2}

	counters := map[string]CounterValue{
		usageCounter(dirUpload, dev1):   {Packets: 500, Bytes: 500000},
		usageCounter(dirDownload, dev1): {Packets: 600, Bytes: 600000},
		usageCounter(dirUpload, dev2):   {Packets: 1, Bytes: 100},
		usageCounter(dirDownload, dev2): {Packets: 2, Bytes: 200},
	}
	samples, err := Samples(mReassigned, counters)
	if err != nil {
		t.Fatalf("Samples: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples for dev2 only, got %d: %+v", len(samples), samples)
	}
	for _, s := range samples {
		if s.DeviceID != dev2 || s.Bytes > 200 {
			t.Fatalf("unexpected sample attribution: %+v", s)
		}
	}
}
