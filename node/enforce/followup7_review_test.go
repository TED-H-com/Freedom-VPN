package enforce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"testing"
)

// TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding proves that VerifyServing
// accepts rulesets inside table inet fvpn that forward member traffic outside the
// serving model:
//  1. A `ct state related` accept at index 0 of `forward` (ahead of `ip saddr != @allowed_v4 drop`),
//     because mentionsEstablished only checks for "established", not "related".
//  2. A `meta mark` match accept at index 0 of `forward` (ahead of `ip saddr != @allowed_v4 drop`),
//     because rejectBypass only checks rule.SetsMark (writing a mark), not matching a mark.
//  3. An unconditional `counter name "ctbase" accept` placed after `ip saddr != @allowed_v4 drop`,
//     bypassing `@denied_v4`, `@host_v4`, per-device counters, QoS priority, and download `@allowed_v4`.
//  4. Deleting the download `iifname "fv-ih" oifname "fv-fh" ip daddr != @allowed_v4 drop` rule
//     entirely, because checkForwardOrder uses one boolean set by the upload allow-set drop.
//  5. Deleting `@denied_v4` and `@host_v4` drop rules when m.Probe == nil, or tainting
//     `@resolver_v4` with an interval/prefix when m.Probe == nil (verifyResolverSet returns early).
//  6. Adding a `prerouting` nat chain with `dnat` in `inet fvpn` that rewrites destination IP
//     before `forward`, or an `iifname "fv-fh" accept` ahead of the drop in `input`.
//  7. Setting `"flags": ["interval"]` on `allowed_v4` in the JSON listing.
func TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	baseJSON := fixtureServing(m, plan.Assignments)
	if err := VerifyServing(baseJSON, m, plan.Assignments); err != nil {
		t.Fatalf("baseline smallModel listing should pass VerifyServing: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(doc map[string]any)
	}{
		{
			name: "ct_state_related_accept_before_allowed_v4_drop",
			mutate: func(doc map[string]any) {
				prependForwardRule(doc, map[string]any{
					"family": "inet", "table": TableName, "chain": "forward",
					"expr": []any{
						map[string]any{"match": map[string]any{
							"op": "in", "left": map[string]any{"ct": map[string]any{"key": "state"}},
							"right": "related",
						}},
						map[string]any{"counter": "ctbase"},
						map[string]any{"accept": nil},
					},
				})
			},
		},
		{
			name: "meta_mark_match_accept_before_allowed_v4_drop",
			mutate: func(doc map[string]any) {
				prependForwardRule(doc, map[string]any{
					"family": "inet", "table": TableName, "chain": "forward",
					"expr": []any{
						map[string]any{"match": map[string]any{
							"op": "==", "left": map[string]any{"meta": map[string]any{"key": "mark"}},
							"right": 1,
						}},
						map[string]any{"counter": "ctbase"},
						map[string]any{"accept": nil},
					},
				})
			},
		},
		{
			name: "unconditional_accept_after_upload_allow_drop_bypasses_denied_and_qos",
			mutate: func(doc map[string]any) {
				insertAfterUploadAllowDrop(doc, map[string]any{
					"family": "inet", "table": TableName, "chain": "forward",
					"expr": []any{
						map[string]any{"counter": "ctbase"},
						map[string]any{"accept": nil},
					},
				})
			},
		},
		{
			name: "missing_download_allowed_v4_drop_before_established_accept",
			mutate: func(doc map[string]any) {
				removeForwardRuleWhere(doc, func(raw string) bool {
					return strings.Contains(raw, `"daddr"`) && strings.Contains(raw, `"@allowed_v4"`)
				})
			},
		},
		{
			name: "missing_denied_v4_and_host_v4_drops_when_probe_nil",
			mutate: func(doc map[string]any) {
				removeForwardRuleWhere(doc, func(raw string) bool {
					return strings.Contains(raw, `"@denied_v4"`) || strings.Contains(raw, `"@host_v4"`)
				})
			},
		},
		{
			name: "tainted_prefix_in_resolver_v4_when_probe_nil",
			mutate: func(doc map[string]any) {
				replaceSetElem(doc, "resolver_v4", []any{
					map[string]any{"prefix": map[string]any{"addr": "0.0.0.0", "len": 0}},
				})
			},
		},
		{
			name: "prerouting_dnat_chain_rewrites_destination_before_forward",
			mutate: func(doc map[string]any) {
				items := doc["nftables"].([]any)
				items = append(items,
					map[string]any{"chain": map[string]any{
						"family": "inet", "table": TableName, "name": "prerouting",
						"type": "nat", "hook": "prerouting", "prio": -100, "policy": "accept",
					}},
					map[string]any{"rule": map[string]any{
						"family": "inet", "table": TableName, "chain": "prerouting",
						"expr": []any{
							map[string]any{"dnat": map[string]any{"addr": "93.184.216.34"}},
						},
					}},
				)
				doc["nftables"] = items
			},
		},
		{
			name: "input_accept_member_ingress_before_drop",
			mutate: func(doc map[string]any) {
				prependChainRule(doc, "input", map[string]any{
					"family": "inet", "table": TableName, "chain": "input",
					"expr": []any{
						map[string]any{"match": map[string]any{
							"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}},
							"right": m.Datapath.MemberIngress,
						}},
						map[string]any{"accept": nil},
					},
				})
			},
		},
		{
			name: "interval_flag_on_allowed_v4_set",
			mutate: func(doc map[string]any) {
				setFlags(doc, "allowed_v4", []string{"interval", "timeout"})
			},
		},
		{
			name: "duplicate_forward_chain_first_accept_policy_masked_by_second_drop_entry",
			mutate: func(doc map[string]any) {
				items := doc["nftables"].([]any)
				for _, item := range items {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					if c, ok := m["chain"].(map[string]any); ok && c["name"] == "forward" {
						c["policy"] = "accept"
					}
				}
				items = append(items, map[string]any{"chain": map[string]any{
					"family": "ip", "table": TableName, "name": "forward",
					"type": "filter", "hook": "forward", "prio": HookPriorityFilter, "policy": "drop",
				}})
				doc["nftables"] = items
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(baseJSON, &doc); err != nil {
				t.Fatalf("unmarshal baseJSON: %v", err)
			}
			tc.mutate(doc)
			mutated, err := json.Marshal(doc)
			if err != nil {
				t.Fatalf("marshal mutated: %v", err)
			}
			if err := VerifyServing(mutated, m, plan.Assignments); err == nil {
				t.Fatalf("VerifyServing accepted out-of-model ruleset mutation %q", tc.name)
			}
		})
	}
}

// TestReviewR12_Q2_RevokedDeviceCountersLeakIntoSamplesAndConntrackDropsCounters
// proves two revocation accounting/cleanup defects in node/enforce/apply.go:
//  1. Because Render never deletes named counters (`u_<dev>`, `d_<dev>`, `nu_<dev>`, `nd_<dev>`)
//     from `inet fvpn`, removing a device from m.Bindings leaves its counters in `rs.Counters`.
//     Samples(m, rs.Counters) looks up `b := byID[id.String()]` without checking `, ok`,
//     so it emits Sample entries with MemberID == uuid.Nil, PeerID == "", BindingRevision == 0
//     carrying the revoked device's cumulative packet and byte counts on every subsequent Apply.
//  2. When conntrack -D fails after the serving ruleset was installed and verified,
//     Apply returns ApplyResult{Verified: true} BEFORE populating Counters and Samples,
//     leaving ApplyResult.Counters and ApplyResult.Samples nil.
func TestReviewR12_Q2_RevokedDeviceCountersLeakIntoSamplesAndConntrackDropsCounters(t *testing.T) {
	m := goldenModel()
	revokedDev := m.Bindings[1].DeviceID
	lingering := map[string]CounterValue{
		usageCounter(dirUpload, m.Bindings[0].DeviceID):   {Packets: 10, Bytes: 1000},
		usageCounter(dirDownload, m.Bindings[0].DeviceID): {Packets: 20, Bytes: 2000},
		usageCounter(dirUpload, revokedDev):               {Packets: 99, Bytes: 99999},
		usageCounter(dirDownload, revokedDev):             {Packets: 88, Bytes: 88888},
		"nu_" + revokedDev.String():                       {Packets: 77, Bytes: 77777},
		"nd_" + revokedDev.String():                       {Packets: 66, Bytes: 66666},
	}

	// Revoke device 2 from the active bindings.
	mAfter := m
	mAfter.Bindings = m.Bindings[:1]
	samples, err := Samples(mAfter, lingering)
	if err != nil {
		t.Fatalf("Samples: %v", err)
	}
	for _, s := range samples {
		if s.DeviceID == revokedDev || s.BindingRevision == 0 {
			t.Errorf("Samples emitted lingering counter for revoked device %s with MemberID=%s BindingRevision=%d Bytes=%d",
				s.DeviceID, s.MemberID, s.BindingRevision, s.Bytes)
		}
	}

	// Now prove that when conntrack -D fails after VerifyServing and VerifyQoS pass,
	// Apply returns Verified=true with empty Counters and Samples.
	sm := smallModel()
	sm.Revoke = []netip.Addr{mustAddr("100.64.1.1")}
	f := servingExec(sm)
	f.handler = conntrackHandler(sm, f, func(key string) ([]byte, error) {
		return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "conntrack v1.4.8: operation failed"}
	})
	res, err := Apply(context.Background(), f, sm, nil)
	var ctErr *ConntrackDeleteError
	if !errors.As(err, &ctErr) || !res.Verified {
		t.Fatalf("expected ConntrackDeleteError with Verified=true, got Verified=%v err=%v", res.Verified, err)
	}
	if len(res.Counters) == 0 || len(res.Samples) == 0 {
		t.Errorf("Apply returned Verified=true with empty Counters (%d) and Samples (%d) on ConntrackDeleteError", len(res.Counters), len(res.Samples))
	}
}

// TestReviewR12_Q3_ServingWithoutProbeOmitsSNATFromHostV4 proves that in Render
// (render_nft.go:35), m.Datapath.SNAT is only merged into @host_v4 when m.Probe != nil.
// When m.Probe == nil and m.HostV4 is empty, @host_v4 is rendered empty.
func TestReviewR12_Q3_ServingWithoutProbeOmitsSNATFromHostV4(t *testing.T) {
	m := goldenModel()
	m.HostV4 = nil
	m.Probe = nil
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	wantSNAT := m.Datapath.SNAT.String() + "/32"
	if !strings.Contains(plan.NFT, "add element inet fvpn host_v4 { "+wantSNAT+" }") {
		t.Fatalf("serving ruleset without Probe omitted SNAT %s from @host_v4", wantSNAT)
	}
}

// TestReviewR12_Q2_Q3_RevocationOrderingAndDenyBoundariesHeld verifies the parts
// of Questions 1, 2 and 3 that held in the renderer and golden rulesets:
//   - In both serving.nft and probe-serving.nft, `ip saddr != @allowed_v4 drop` precedes
//     every upload/DNS/probe accept, and `ip daddr != @allowed_v4 drop` precedes `probe_down`
//     and every download established/related accept.
//   - Member destination 192.0.0.9/32 and 192.0.0.10/32 are covered by parent 192.0.0.0/24
//     in SpecDeniedV4(), while 192.88.99.0/24 is denied and the three AS112/AMT blocks
//     (192.31.196.0/24, 192.52.193.0/24, 192.175.48.0/24) are not in SpecDeniedV4().
//   - Resolver IP in @resolver_v4 is accepted only on TCP/UDP port 53 and dropped on all
//     other ports by the immediately following `ip daddr @resolver_v4 counter name "ctbadns" drop`.
func TestReviewR12_Q2_Q3_RevocationOrderingAndDenyBoundariesHeld(t *testing.T) {
	for _, golden := range []string{"testdata/serving.nft", "testdata/probe-serving.nft"} {
		rawBytes, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("read %s: %v", golden, err)
		}
		lines := strings.Split(string(rawBytes), "\n")
		upAllowDrop := -1
		downAllowDrop := -1
		firstUploadAccept := -1
		firstDownloadAccept := -1
		probeDownAccept := -1
		dnsTCPAccept := -1
		dnsUDPAccept := -1
		dnsNonPort53Drop := -1

		for i, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "add rule inet fvpn forward ") {
				continue
			}
			if strings.Contains(line, `iifname "fv-fh" oifname "fv-ih" ip saddr != @allowed_v4`) && strings.Contains(line, `counter name "ctunku" drop`) {
				upAllowDrop = i
			}
			if strings.Contains(line, `iifname "fv-ih" oifname "fv-fh" ip daddr != @allowed_v4`) && strings.Contains(line, `counter name "ctunkd" drop`) {
				downAllowDrop = i
			}
			if strings.Contains(line, `ip daddr @resolver_v4 tcp dport 53 counter name "nu_`) {
				dnsTCPAccept = i
			}
			if strings.Contains(line, `ip daddr @resolver_v4 udp dport 53 counter name "nu_`) {
				dnsUDPAccept = i
			}
			if strings.Contains(line, `ip daddr @resolver_v4 counter name "ctbadns" drop`) {
				dnsNonPort53Drop = i
			}
			if strings.HasSuffix(strings.TrimSpace(line), "accept") {
				if strings.Contains(line, `iifname "fv-fh"`) && firstUploadAccept < 0 {
					firstUploadAccept = i
				}
				if strings.Contains(line, `iifname "fv-ih"`) && firstDownloadAccept < 0 {
					firstDownloadAccept = i
				}
				if strings.Contains(line, `counter name "probe_down" accept`) {
					probeDownAccept = i
				}
			}
		}
		if upAllowDrop < 0 || firstUploadAccept < 0 || upAllowDrop >= firstUploadAccept {
			t.Fatalf("%s: upload allow-set drop (%d) does not precede first upload accept (%d)", golden, upAllowDrop, firstUploadAccept)
		}
		if downAllowDrop < 0 || firstDownloadAccept < 0 || downAllowDrop >= firstDownloadAccept {
			t.Fatalf("%s: download allow-set drop (%d) does not precede first download accept (%d)", golden, downAllowDrop, firstDownloadAccept)
		}
		if strings.Contains(golden, "probe") && (probeDownAccept < 0 || downAllowDrop >= probeDownAccept) {
			t.Fatalf("%s: download allow-set drop (%d) does not precede probe_down accept (%d)", golden, downAllowDrop, probeDownAccept)
		}
		if dnsTCPAccept < 0 || dnsUDPAccept < 0 || dnsNonPort53Drop < 0 || !(dnsTCPAccept < dnsNonPort53Drop && dnsUDPAccept < dnsNonPort53Drop) {
			t.Fatalf("%s: resolver port-53 accepts (%d,%d) not followed by ctbadns drop (%d)", golden, dnsTCPAccept, dnsUDPAccept, dnsNonPort53Drop)
		}
	}

	denied := SpecDeniedV4()
	contains := func(ip string) bool {
		addr := netip.MustParseAddr(ip)
		for _, p := range denied {
			if p.Contains(addr) {
				return true
			}
		}
		return false
	}
	for _, anycast := range []string{"192.0.0.9", "192.0.0.10", "192.88.99.1"} {
		if !contains(anycast) {
			t.Fatalf("expected %s to be covered by SpecDeniedV4()", anycast)
		}
	}
	for _, allowed := range []string{"192.31.196.1", "192.52.193.1", "192.175.48.1"} {
		if contains(allowed) {
			t.Fatalf("expected AS112/AMT address %s to be permitted by SpecDeniedV4()", allowed)
		}
	}
}

func TestFollowup7_ServingResolverNon53Drop(t *testing.T) {
	m := smallModel()
	m.Probe = nil
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	drop := `ip daddr @resolver_v4 counter name "ctbadns" drop`
	deny := `ip daddr @denied_v4`
	dropAt := strings.Index(plan.NFT, drop)
	denyAt := strings.Index(plan.NFT, deny)
	if dropAt < 0 || denyAt < 0 || dropAt > denyAt {
		t.Fatalf("serving resolver drop %d is not before the deny drop %d", dropAt, denyAt)
	}
	raw := fixtureServing(m, plan.Assignments)
	if err := VerifyServing(raw, m, plan.Assignments); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	removeForwardRuleWhere(doc, func(s string) bool {
		return strings.Contains(s, `"@resolver_v4"`) && strings.Contains(s, `"drop"`)
	})
	mutated, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyServing(mutated, m, plan.Assignments); err == nil {
		t.Fatal("VerifyServing accepted a serving ruleset with no non-53 resolver drop")
	}
}

func prependForwardRule(doc map[string]any, rule map[string]any) {
	prependChainRule(doc, "forward", rule)
}

func prependChainRule(doc map[string]any, chain string, rule map[string]any) {
	items := doc["nftables"].([]any)
	var out []any
	inserted := false
	for _, item := range items {
		m, ok := item.(map[string]any)
		if ok && !inserted {
			if r, ok := m["rule"].(map[string]any); ok && r["chain"] == chain {
				out = append(out, map[string]any{"rule": rule})
				inserted = true
			}
		}
		out = append(out, item)
	}
	if !inserted {
		out = append(out, map[string]any{"rule": rule})
	}
	doc["nftables"] = out
}

func insertAfterUploadAllowDrop(doc map[string]any, rule map[string]any) {
	items := doc["nftables"].([]any)
	var out []any
	inserted := false
	for _, item := range items {
		out = append(out, item)
		if !inserted {
			raw, _ := json.Marshal(item)
			if strings.Contains(string(raw), `"saddr"`) && strings.Contains(string(raw), `"@allowed_v4"`) {
				out = append(out, map[string]any{"rule": rule})
				inserted = true
			}
		}
	}
	doc["nftables"] = out
}

func removeForwardRuleWhere(doc map[string]any, pred func(raw string) bool) {
	items := doc["nftables"].([]any)
	var out []any
	for _, item := range items {
		raw, _ := json.Marshal(item)
		if strings.Contains(string(raw), `"rule"`) && pred(string(raw)) {
			continue
		}
		out = append(out, item)
	}
	doc["nftables"] = out
}

func replaceSetElem(doc map[string]any, setName string, elem any) {
	items := doc["nftables"].([]any)
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := m["set"].(map[string]any); ok && s["name"] == setName {
			s["elem"] = elem
		}
	}
}

func setFlags(doc map[string]any, setName string, flags []string) {
	items := doc["nftables"].([]any)
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := m["set"].(map[string]any); ok && s["name"] == setName {
			s["flags"] = flags
		}
	}
}
