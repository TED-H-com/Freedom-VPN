package enforce

import (
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"
)

// TEST ONLY. Positive control and six isolated mutations use the same live
// read-back entry point as Apply and VerifyServing, without changing any rule.
func TestISO01_F27_LiveSetDrift(t *testing.T) {
	m := smallModel()
	m.DeniedV4 = []netip.Prefix{mustPrefix("192.0.2.0/25"), mustPrefix("198.51.100.0/24")}
	m.HostV4 = []netip.Prefix{mustPrefix("203.0.113.4/32"), mustPrefix("203.0.113.8/32")}
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	raw := f27StaticSets(fixtureServing(m, plan.Assignments), m)
	if err := verifyServingRuleset(raw, m, plan.Assignments); err != nil {
		t.Fatal("positive control:", err)
	}
	for _, name := range []string{"denied_v4", "host_v4"} {
		for _, change := range []string{"add", "remove", "replace"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				changed := f27ChangeSet(raw, name, func(set map[string]any) {
					elems := set["elem"].([]any)
					switch change {
					case "add":
						elems = append(elems, "203.0.113.99")
					case "remove":
						elems = elems[1:]
					case "replace":
						elems[0] = "203.0.113.99"
					}
					set["elem"] = elems
				})
				err := verifyServingRuleset(changed, m, plan.Assignments)
				var check *CheckError
				if !errors.As(err, &check) || check.Reason != ReasonIsolation {
					t.Fatalf("live %s %s was not rejected as isolation: %v", name, change, err)
				}
			})
		}
	}
}

// Fixtures use nft's JSON prefix/address shapes, independently of the parser.
func f27StaticSets(raw []byte, m Model) []byte {
	denied := m.DeniedV4
	if denied == nil {
		denied = SpecDeniedV4()
	}
	denied = append(append([]netip.Prefix{}, denied...), m.ExtraDeniedV4...)
	host := append(append([]netip.Prefix{}, m.HostV4...), netip.PrefixFrom(m.Datapath.SNAT, 32))
	for name, prefixes := range map[string][]netip.Prefix{"denied_v4": denied, "host_v4": host} {
		var elems []any
		for _, p := range prefixes {
			if p.Bits() == 32 {
				elems = append(elems, p.Addr().String())
			} else {
				elems = append(elems, map[string]any{"prefix": map[string]any{"addr": p.Masked().Addr().String(), "len": p.Bits()}})
			}
		}
		raw = f27ChangeSet(raw, name, func(set map[string]any) {
			set["type"], set["flags"], set["elem"] = "ipv4_addr", []any{"interval"}, elems
		})
	}
	return raw
}

func f27ChangeSet(raw []byte, name string, change func(map[string]any)) []byte {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	items := doc["nftables"].([]any)
	found := false
	for _, item := range items {
		if set, ok := item.(map[string]any)["set"].(map[string]any); ok && set["name"] == name {
			change(set)
			found = true
		}
	}
	if !found {
		set := map[string]any{"family": "inet", "table": "fvpn", "name": name}
		change(set)
		items = append(items, map[string]any{"set": set})
	}
	doc["nftables"] = items
	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return out
}

func f27ReplaceSet(raw []byte, replacement map[string]any) []byte {
	return f27ChangeSet(raw, replacement["name"].(string), func(set map[string]any) {
		clear(set)
		for key, value := range replacement {
			set[key] = value
		}
	})
}

func TestISO01_F27_IntervalShapesAndTaint(t *testing.T) {
	for _, path := range []string{"serving", "serving_probe", "baseline_probe"} {
		t.Run(path, func(t *testing.T) {
			m := smallModel()
			if path != "serving" {
				m.Probe = probeModel().Probe
			}
			m.DeniedV4 = []netip.Prefix{mustPrefix("192.0.2.0/25")}
			m.ExtraDeniedV4 = []netip.Prefix{mustPrefix("192.0.2.128/25"), mustPrefix("192.0.2.64/26")}
			m.HostV4 = []netip.Prefix{mustPrefix("192.0.2.32/30"), mustPrefix("192.0.2.36/32")}
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				t.Fatal(err)
			}
			raw := fixtureServing(m, plan.Assignments)
			if path == "serving_probe" {
				raw = probeServingFixture(m, plan.Assignments, true)
			}
			if path == "baseline_probe" {
				raw = probeBaselineFixture(m, true)
			}
			verify := func(raw []byte) error {
				if path == "baseline_probe" {
					return VerifyBaseline(raw, m)
				}
				return verifyServingRuleset(raw, m, plan.Assignments)
			}
			if err := verify(raw); err != nil {
				t.Fatal("positive control:", err)
			}
			// .32-.37 includes the implicit SNAT .37; no HostV4 entry duplicates it.
			for name, span := range map[string][]string{"denied_v4": {"192.0.2.0", "192.0.2.255"}, "host_v4": {"192.0.2.32", "192.0.2.37"}} {
				merged := f27ChangeSet(raw, name, func(s map[string]any) { s["elem"] = []any{map[string]any{"range": span}} })
				if err := verify(merged); err != nil {
					t.Fatalf("equivalent %s ranges: %v", name, err)
				}
				for change, mutate := range map[string]func(map[string]any){
					"missing":          func(s map[string]any) { s["name"] = "unreferenced_v4" },
					"empty":            func(s map[string]any) { delete(s, "elem") },
					"malformed_flags":  func(s map[string]any) { s["flags"] = 42 },
					"no_interval":      func(s map[string]any) { delete(s, "flags") },
					"timeout_flag":     func(s map[string]any) { s["flags"] = []any{"interval", "timeout"} },
					"set_timeout_zero": func(s map[string]any) { s["timeout"] = 0 },
					"wrong_type":       func(s map[string]any) { s["type"] = "ipv6_addr" },
					"expires": func(s map[string]any) {
						s["elem"] = []any{map[string]any{"elem": map[string]any{"val": map[string]any{"range": span}, "expires": 1}}}
					},
					"timeout_zero": func(s map[string]any) {
						s["elem"] = []any{map[string]any{"val": map[string]any{"range": span}, "timeout": 0}}
					},
					"unknown": func(s map[string]any) { s["elem"] = []any{map[string]any{"unknown": span}} },
					"add":     func(s map[string]any) { s["elem"] = append(s["elem"].([]any), "203.0.113.99") },
				} {
					t.Run(name+"/"+change, func(t *testing.T) {
						err := verify(f27ChangeSet(merged, name, mutate))
						var check *CheckError
						if !errors.As(err, &check) || check.Reason != ReasonIsolation {
							t.Fatalf("want isolation, got %v", err)
						}
					})
				}
			}
		})
	}
}

func TestISO01_F27_BareBaseline(t *testing.T) {
	m := Model{Datapath: smallModel().Datapath}
	raw := fixtureBaseline()
	if err := VerifyBaseline(raw, m); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"denied_v4", "host_v4"} {
		unexpected := f27ChangeSet(raw, name, func(s map[string]any) { s["flags"] = []any{"interval"}; s["type"] = "ipv4_addr" })
		if err := VerifyBaseline(unexpected, m); err == nil {
			t.Fatal("unexpected empty set accepted", name)
		}
	}
	for _, elems := range []any{[]any{"100.64.1.1"}, []any{map[string]any{"prefix": map[string]any{"addr": "100.64.0.0", "len": 10}}}, []any{map[string]any{"unknown": nil}}} {
		changed := f27ChangeSet(raw, "allowed_v4", func(s map[string]any) { s["elem"] = elems })
		if err := VerifyBaseline(changed, m); err == nil {
			t.Fatal("nonempty/tainted baseline allowed_v4 accepted")
		}
	}
	state := State{Sets: map[string]bool{"denied_v4": true, "host_v4": true}}
	script, err := RenderBaseline(m, state)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"denied_v4", "host_v4"} {
		if !strings.Contains(script, "delete set inet fvpn "+name) {
			t.Fatal("bare transition retains", name)
		}
	}
}

func TestISO01_F27_IntervalBoundaries(t *testing.T) {
	for _, raw := range []string{
		`[{"prefix":{"addr":"0.0.0.0","len":0}}]`,
		`[{"prefix":{"addr":"0.0.0.0","len":1}},{"prefix":{"addr":"128.0.0.0","len":1}}]`,
		`[{"range":["0.0.0.0","255.255.255.254"]},"255.255.255.255"]`,
	} {
		got, bad := parseIntervalElems([]byte(raw))
		if bad || len(got) != 1 || got[0] != (ipv4Range{0, ^uint32(0)}) {
			t.Fatal("full IPv4 union", raw, got, bad)
		}
	}
	for _, raw := range []string{
		`[{"prefix":{"addr":"192.0.2.0","len":33}}]`,
		`[{"prefix":{"addr":"192.0.2.0","len":null}}]`,
		`[{"prefix":{"addr":"192.0.2.0","len":24.5}}]`,
		`[{"range":["192.0.2.2","192.0.2.1"]}]`,
		`[{"range":["192.0.2.1"]}]`,
		`["::ffff:192.0.2.1"]`, `["2001:db8::1"]`,
		`[{"elem":{"val":"192.0.2.1","expires":null}}]`,
		`[{"elem":"192.0.2.1","timeout":0}]`,
		`["192.0.2.1 timeout 1s"]`, `[{"concat":["192.0.2.1",80]}]`,
	} {
		if _, bad := parseIntervalElems([]byte(raw)); !bad {
			t.Fatal("accepted tainted interval", raw)
		}
	}
}

func TestISO01_F27_ProbeBaselineAdmissionMustBeEmpty(t *testing.T) {
	m := smallModel()
	m.Probe = probeModel().Probe
	raw := probeBaselineFixture(m, true)
	if err := VerifyBaseline(raw, m); err != nil {
		t.Fatal(err)
	}
	for name, element := range map[string]any{
		"timed_host": elemVal("100.64.1.1", map[string]any{"timeout": 30, "expires": 29}),
		"prefix":     map[string]any{"prefix": map[string]any{"addr": "100.64.0.0", "len": 10}},
		"unknown":    map[string]any{"unexpected": nil},
	} {
		t.Run(name, func(t *testing.T) {
			changed := f27ChangeSet(raw, "allowed_v4", func(s map[string]any) { s["elem"] = []any{element} })
			var check *CheckError
			if err := VerifyBaseline(changed, m); !errors.As(err, &check) || check.Reason != ReasonIsolation {
				t.Fatalf("baseline admission: %v", err)
			}
		})
	}
}
