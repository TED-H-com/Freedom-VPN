package enforce

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func f34PMTUFixture(raw []byte, m Model) []byte {
	var doc map[string][]map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	devices, _, _, err := validateServing(m)
	if err != nil {
		panic(err)
	}
	var added []map[string]any
	for _, d := range devices {
		if d.TimeoutOK() {
			for _, addr := range d.Addrs {
				for _, set := range []string{"resolver_v4", "denied_v4", "host_v4"} {
					// Independent live nft JSON shape; deliberately use the string class ID.
					text := fmt.Sprintf(`{"rule":{"family":"inet","table":"fvpn","chain":"forward","expr":[
   {"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":%q}},
   {"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":%q}},
   {"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":%q}},
   {"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":%q}},
   {"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},
   {"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},
   {"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},
   {"counter":%q},{"mangle":{"key":{"meta":{"key":"priority"}},"value":%q}},{"accept":null}
  ]}}`, m.Datapath.Uplink, m.Datapath.MemberIngress, "@"+set, addr, usageCounter(dirDownload, d.Binding.DeviceID), fmt.Sprintf("1:%x", d.Minor))
					var obj map[string]any
					if err := json.Unmarshal([]byte(text), &obj); err != nil {
						panic(err)
					}
					added = append(added, obj)
				}
			}
		}
	}
	var objects []map[string]any
	inserted := false
	for _, o := range doc["nftables"] {
		if r, ok := o["rule"].(map[string]any); ok && !inserted {
			b, _ := json.Marshal(r)
			if strings.Contains(string(b), `"name":"ctdenyd"`) {
				objects = append(objects, added...)
				inserted = true
			}
		}
		objects = append(objects, o)
	}
	doc["nftables"] = objects
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return b
}

func TestAT13_D147_RenderParseVerify(t *testing.T) {
	for _, probe := range []bool{false, true} {
		m := smallModel()
		if probe {
			m.Probe = probeModel().Probe
		}
		p, err := Render(m, State{}, TCState{})
		if err != nil {
			t.Fatal(err)
		}
		raw := fixtureServing(m, p.Assignments)
		if probe {
			raw = probeServingFixture(m, p.Assignments, true)
		}
		if err := verifyServingRuleset(raw, m, p.Assignments); err != nil {
			t.Fatal(err)
		}
		for _, set := range []string{"resolver_v4", "denied_v4", "host_v4"} {
			want := fmt.Sprintf(`iifname "fv-ih" oifname "fv-fh" ip saddr @%s ip daddr 100.64.1.1 ct state related icmp type destination-unreachable icmp code frag-needed counter name "d_019a00000000700080000000000000a1" meta priority set 0x%08x accept`, set, ClassID(p.Assignments[memberA]))
			if strings.Count(p.NFT, want) != 1 {
				t.Fatalf("missing exact PMTU accept %s", want)
			}
			idx := strings.Index(p.NFT, want)
			if idx < strings.Index(p.NFT, `ct state != { established, related }`) || idx > strings.Index(p.NFT, `ip saddr @resolver_v4 counter name "ctdenyd" drop`) {
				t.Fatal("PMTU placement")
			}
		}
		for _, tc := range []struct {
			name  string
			index int
			field string
			value any
		}{
			{"time-exceeded", 5, "right", "time-exceeded"}, {"other-code", 6, "right", 3},
			{"established", 4, "right", "established"}, {"not-related", 4, "op", "!="},
			{"non-member", 3, "right", "100.64.9.99"}, {"ipv6", -1, "", nil},
			{"wrong-interface", 0, "right", "fv-other"}, {"wrong-source-set", 2, "right", "@allowed_v4"},
			{"counter", 7, "", nil}, {"priority", 8, "", nil},
		} {
			t.Run(fmt.Sprintf("probe=%t/%s", probe, tc.name), func(t *testing.T) {
				bad := f34EditRule(t, raw, false, func(r map[string]any) {
					e := r["expr"].([]any)
					switch tc.name {
					case "ipv6":
						e[3].(map[string]any)["match"].(map[string]any)["left"].(map[string]any)["payload"].(map[string]any)["protocol"] = "ip6"
					case "counter":
						e[7] = map[string]any{"counter": "ctdenyd"}
					case "priority":
						e[8].(map[string]any)["mangle"].(map[string]any)["value"] = "1:fffe"
					default:
						e[tc.index].(map[string]any)["match"].(map[string]any)[tc.field] = tc.value
					}
				})
				if err := verifyServingRuleset(bad, m, p.Assignments); err == nil {
					t.Fatal("altered PMTU rule accepted")
				}
			})
		}
		for _, mutation := range []string{"missing", "early", "late", "duplicate", "extra-match", "no-related", "no-icmp", "nftrace"} {
			t.Run(fmt.Sprintf("probe=%t/%s", probe, mutation), func(t *testing.T) {
				var d map[string][]map[string]any
				if err := json.Unmarshal(raw, &d); err != nil {
					t.Fatal(err)
				}
				for i, o := range d["nftables"] {
					r, ok := o["rule"].(map[string]any)
					if !ok {
						continue
					}
					b, _ := json.Marshal(r)
					if !strings.Contains(string(b), `"frag-needed"`) {
						continue
					}
					e := r["expr"].([]any)
					switch mutation {
					case "missing", "early", "late":
						d["nftables"] = append(d["nftables"][:i], d["nftables"][i+1:]...)
						if mutation == "early" {
							d["nftables"] = append([]map[string]any{o}, d["nftables"]...)
						}
						if mutation == "late" {
							d["nftables"] = append(d["nftables"], o)
						}
					case "duplicate":
						d["nftables"] = append(d["nftables"], o)
					case "extra-match":
						r["expr"] = append([]any{matchMeta("nfproto", "ipv4")}, e...)
					case "no-related":
						r["expr"] = append(e[:4], e[5:]...)
					case "no-icmp":
						r["expr"] = append(e[:5], e[7:]...)
					case "nftrace":
						r["expr"] = append([]any{map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "nftrace"}}, "value": 1}}}, e...)
					}
					break
				}
				b, _ := json.Marshal(d)
				if err := verifyServingRuleset(b, m, p.Assignments); err == nil {
					t.Fatal("unsafe PMTU rule accepted")
				}
			})
		}
	}
}
