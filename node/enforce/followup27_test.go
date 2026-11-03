package enforce

import (
	"encoding/json"
	"errors"
	"net/netip"
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
