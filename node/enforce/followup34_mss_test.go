package enforce

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Independent fixture uses the JSON read back from nft 1.0.9. No invocation
// of the verifier's expression builder: a renderer/verifier drift must fail.
func f34MSSRule(in, out, field, addr string) map[string]any {
	raw := fmt.Sprintf(`{"rule":{"family":"inet","table":"fvpn","chain":"forward","expr":[
 {"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":%q}},
 {"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":%q}},
 {"match":{"op":"==","left":{"payload":{"protocol":"ip","field":%q}},"right":%q}},
 {"match":{"op":"==","left":{"&":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},
 {"match":{"op":">","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},
 {"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}
 ]}}`, in, out, field, addr)
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		panic(err)
	}
	return obj
}

func f34MSSFixture(raw []byte, m Model) []byte {
	var doc map[string][]map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	var objects []map[string]any
	for _, o := range doc["nftables"] {
		if r, ok := o["rule"].(map[string]any); ok && r["chain"] == "forward" {
			expr, _ := json.Marshal(r["expr"])
			rv := RuleView{Raw: string(expr)}
			if err := inspectExpr(expr, &rv); err != nil {
				panic(err)
			}
			kind := servingAcceptKind(rv, m)
			if kind == "upload" {
				objects = append(objects, f34MSSRule(m.Datapath.MemberIngress, m.Datapath.Uplink, "saddr", rv.Saddrs[0].Addr().String()))
			}
			if kind == "download" {
				objects = append(objects, f34MSSRule(m.Datapath.Uplink, m.Datapath.MemberIngress, "daddr", rv.Daddrs[0].Addr().String()))
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

func TestAT09_D146_MSSRenderParseVerify(t *testing.T) {
	m := smallModel()
	p, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct{ in, out, field, counter string }{{"fv-fh", "fv-ih", "saddr", "u_"}, {"fv-ih", "fv-fh", "daddr", "d_"}} {
		want := fmt.Sprintf("add rule inet fvpn forward iifname %q oifname %q ip %s 100.64.1.1 tcp flags & syn == syn tcp option maxseg size > 1160 tcp option maxseg size set 1160\nadd rule inet fvpn forward iifname %q oifname %q ip %s 100.64.1.1 counter name %q", pair.in, pair.out, pair.field, pair.in, pair.out, pair.field, pair.counter+strings.ReplaceAll(devA1.String(), "-", ""))
		if !strings.Contains(p.NFT, want) {
			t.Fatalf("missing exact adjacent clamp:\n%s", want)
		}
	}
	raw := fixtureServing(m, p.Assignments)
	if err := verifyServingRuleset(raw, m, p.Assignments); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"larger", `1160`, `1200`}, {"smaller", `1160`, `1000`},
		{"raises-small", `"op":"\u003e"`, `"op":"<"`},
		{"syn-match", `"right":"syn"`, `"right":"ack"`},
		{"protocol", `"protocol":"tcp"`, `"protocol":"udp"`},
		{"ipv6", `"protocol":"ip","field":"saddr"`, `"protocol":"ip6","field":"saddr"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Mutate only a clamp, not a preceding admission or deny rule.
			bad := f34EditRule(t, raw, true, func(r map[string]any) {
				b, _ := json.Marshal(r)
				text := string(b)
				old := tc.old
				if tc.name == "ipv6" {
					old = `"field":"saddr","protocol":"ip"`
					tc.replacement = `"field":"saddr","protocol":"ip6"`
				}
				if !strings.Contains(text, old) {
					t.Fatalf("mutation not applied: %s", old)
				}
				if err := json.Unmarshal([]byte(strings.Replace(text, old, tc.replacement, 1)), &r); err != nil {
					t.Fatal(err)
				}
			})
			if err := verifyServingRuleset(bad, m, p.Assignments); err == nil {
				t.Fatal("altered clamp accepted")
			}
		})
	}
	for _, mutation := range []string{"missing", "duplicate", "late", "early", "nftrace", "extra-match", "verdict"} {
		t.Run(mutation, func(t *testing.T) {
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
				if !strings.Contains(string(b), `"tcp option"`) {
					continue
				}
				expr := r["expr"].([]any)
				switch mutation {
				case "missing", "late", "early":
					d["nftables"] = append(d["nftables"][:i], d["nftables"][i+1:]...)
					if mutation == "late" {
						d["nftables"] = append(d["nftables"], o)
					}
					if mutation == "early" {
						d["nftables"] = append([]map[string]any{o}, d["nftables"]...)
					}
				case "duplicate":
					d["nftables"] = append(d["nftables"], o)
				case "nftrace":
					expr[len(expr)-1] = map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "nftrace"}}, "value": 1}}
				case "extra-match":
					r["expr"] = append(expr, matchDport("tcp", 443))
				case "verdict":
					r["expr"] = append(expr, map[string]any{"accept": nil})
				}
				break
			}
			bad, _ := json.Marshal(d)
			if err := verifyServingRuleset(bad, m, p.Assignments); err == nil {
				t.Fatal("unsafe clamp accepted")
			}
		})
	}
}

func f34EditRule(t *testing.T, raw []byte, mss bool, edit func(map[string]any)) []byte {
	t.Helper()
	var d map[string][]map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range d["nftables"] {
		r, ok := o["rule"].(map[string]any)
		if !ok {
			continue
		}
		b, _ := json.Marshal(r)
		match := strings.Contains(string(b), `"tcp option"`)
		if !mss {
			match = strings.Contains(string(b), `"frag-needed"`)
		}
		if match {
			edit(r)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no target rule")
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
