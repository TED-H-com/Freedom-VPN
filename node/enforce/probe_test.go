package enforce

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRender_ProbeGolden(t *testing.T) {
	m := probeModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "probe-serving.nft", plan.NFT)
	if strings.Contains(plan.NFT, "meta mark") || strings.Contains(plan.NFT, "ct mark") {
		t.Fatal("probe script writes a mark")
	}
	rules := plan.NFT[strings.Index(plan.NFT, "add rule"):]
	v6 := strings.Index(rules, "meta nfproto ipv6")
	dns := strings.Index(rules, "ip saddr @probe_v4 ip daddr @resolver_v4 tcp dport 53")
	deny := strings.Index(rules, "ip daddr @denied_v4")
	host := strings.Index(rules, "ip daddr @host_v4")
	tuple := strings.Index(rules, "ip daddr . meta l4proto . th dport @probe_dst_v4")
	member := strings.Index(rules, `ip saddr 100.64.1.1 counter name "u_`)
	down := strings.Index(rules, `counter name "probe_down" accept`)
	if v6 < 0 || dns < 0 || deny < 0 || host < 0 || tuple < 0 || member < 0 || down < 0 {
		t.Fatalf("missing probe rules:\n%s", plan.NFT)
	}
	if !(v6 < dns && dns < deny && deny < host && host < tuple && tuple < member && member < down) {
		t.Fatalf("probe order v6=%d dns=%d deny=%d host=%d tuple=%d member=%d down=%d", v6, dns, deny, host, tuple, member, down)
	}
	if !strings.Contains(plan.NFT, "100.64.8.1 timeout 30s") || !strings.Contains(plan.NFT, "1.2.3.4 . tcp . 80 timeout 30s") {
		t.Fatal("probe elements missing")
	}
	if !strings.Contains(plan.NFT, `ip saddr @probe_v4 snat ip to 192.0.2.37`) {
		t.Fatal("probe SNAT missing")
	}
	base, err := RenderBaseline(m, State{})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "probe-baseline.nft", base)
	brules := base[strings.Index(base, "add rule"):]
	if strings.Count(brules, " accept") != 4 {
		t.Fatalf("baseline probe accepts:\n%s", base)
	}
	if strings.Contains(base, "ip saddr 100.64.1.1") {
		t.Fatal("baseline probe opened a member accept")
	}
}

func TestRender_ProbeNilRemovesSets(t *testing.T) {
	state := State{Sets: map[string]bool{setProbeV4: true, setProbeDstV4: true, "allowed_v4": true}}
	plan, err := Render(goldenModel(), state, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan.NFT, "@probe_v4") || strings.Contains(plan.NFT, "@probe_dst_v4") {
		t.Fatal("nil probe still matches the probe sets")
	}
	if !strings.Contains(plan.NFT, "delete set inet fvpn probe_v4") || !strings.Contains(plan.NFT, "delete set inet fvpn probe_dst_v4") {
		t.Fatal("nil probe did not delete the probe sets")
	}
	del := strings.LastIndex(plan.NFT, "delete set")
	add := strings.LastIndex(plan.NFT, "add rule")
	if del < add {
		t.Fatal("probe set deleted before the chains were replaced")
	}
	base, err := RenderBaseline(Model{Datapath: goldenModel().Datapath}, state)
	if err != nil {
		t.Fatal(err)
	}
	brules := base[strings.Index(base, "add rule"):]
	if strings.Contains(brules, " accept") || !strings.Contains(base, "delete set inet fvpn probe_v4") {
		t.Fatalf("nil baseline:\n%s", base)
	}
}

func TestRender_ProbeRejects(t *testing.T) {
	now := goldenModel().Now
	cases := []struct {
		name string
		edit func(*Model)
	}{
		{"wide source", func(m *Model) { m.Probe.SourceV4 = mustPrefix("100.64.8.0/24") }},
		{"wide dest", func(m *Model) { m.Probe.Destinations[0].AddrV4 = mustPrefix("1.2.3.0/24") }},
		{"rfc1918", func(m *Model) { m.Probe.Destinations[0].AddrV4 = mustPrefix("10.1.1.1/32") }},
		{"link local", func(m *Model) { m.Probe.Destinations[0].AddrV4 = mustPrefix("169.254.1.50/32") }},
		{"metadata", func(m *Model) { m.Probe.Destinations[0].AddrV4 = mustPrefix("169.254.169.254/32") }},
		{"doc range", func(m *Model) { m.Probe.Destinations[0].AddrV4 = mustPrefix("203.0.113.10/32") }},
		{"host", func(m *Model) {
			m.Probe.Destinations[0].AddrV4 = mustPrefix("192.0.2.37/32")
			m.Probe.Destinations[0].Port = 9
		}},
		{"admin", func(m *Model) {
			m.Probe.Destinations[0].AddrV4 = mustPrefix("192.0.2.37/32")
			m.Probe.Destinations[0].Port = 443
		}},
		{"proto", func(m *Model) { m.Probe.Destinations[0].Proto = "icmp" }},
		{"port", func(m *Model) { m.Probe.Destinations[0].Port = 0 }},
		{"empty", func(m *Model) {
			m.Probe.Destinations = nil
			m.Probe.ResolverDNS = false
		}},
		{"past", func(m *Model) { m.Probe.ExpiresAt = now.Add(-time.Second) }},
		{"subsecond", func(m *Model) { m.Probe.ExpiresAt = now.Add(500 * time.Millisecond) }},
		{"too long", func(m *Model) {
			m.Probe.ExpiresAt = now.Add(time.Duration(ProbeMaxSeconds)*time.Second + time.Nanosecond)
		}},
		{"no now", func(m *Model) { m.Now = time.Time{} }},
		{"peer", func(m *Model) { m.Probe.PeerID = "" }},
		{"member peer", func(m *Model) { m.Probe.PeerID = "peer-a1" }},
		{"member addr", func(m *Model) { m.Probe.SourceV4 = mustPrefix("100.64.1.1/32") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := probeModel()
			tc.edit(&m)
			if _, err := Render(m, State{}, TCState{}); err == nil {
				t.Fatal("serving accepted the probe")
			}
			if _, err := RenderBaseline(m, State{}); err == nil {
				t.Fatal("baseline accepted the probe")
			}
		})
	}
	m := probeModel()
	m.Probe.Destinations[0].Port = 443
	if _, err := Render(m, State{}, TCState{}); err != nil {
		t.Fatal(err)
	}
	m = probeModel()
	m.Probe.ExpiresAt = m.Now.Add(time.Duration(ProbeMaxSeconds) * time.Second)
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.NFT, "100.64.8.1 timeout 300s") {
		t.Fatal("300s probe was not admitted")
	}
	m = probeModel()
	m.DeniedV4 = Without(SpecDeniedV4(), mustPrefix("203.0.113.0/24"))
	m.Probe.Destinations[0].AddrV4 = mustPrefix("203.0.113.10/32")
	plan, err = Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.NFT, "203.0.113.10 . tcp . 80") {
		t.Fatal("lab internet destination was rejected")
	}
}

func TestSamples_ProbeNotUsage(t *testing.T) {
	got, err := Samples(smallModel(), map[string]CounterValue{
		counterProbeUp:   {Packets: 4, Bytes: 400},
		counterProbeDown: {Packets: 3, Bytes: 300},
		counterDenied:    {Packets: 1, Bytes: 40},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("probe counters entered usage: %+v", got)
	}
}

func TestVerify_Probe(t *testing.T) {
	m := smallModel()
	m.Probe = probeModel().Probe
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	raw := probeServingFixture(m, plan.Assignments, true)
	if err := VerifyServing(raw, m, plan.Assignments); err != nil {
		t.Fatal(err)
	}
	base := probeBaselineFixture(m, true)
	if err := VerifyBaseline(base, m); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBaseline(probeBaselineFixture(m, false), m); err == nil || !strings.Contains(err.Error(), string(ReasonProbeUnbounded)) {
		t.Fatalf("unbounded: %v", err)
	}
	extra := appendForwardAccept(base)
	if err := VerifyBaseline(extra, m); err == nil || !strings.Contains(err.Error(), string(ReasonIsolationBaselineMissing)) {
		t.Fatalf("extra accept: %v", err)
	}
	plain := smallModel()
	if err := VerifyServing(raw, plain, plan.Assignments); err == nil {
		t.Fatal("probe rules matched a model with no probe")
	}
}

func probeModel() Model {
	m := goldenModel()
	m.Probe = &ProbeException{
		PeerID:   "peer-probe",
		SourceV4: mustPrefix("100.64.8.1/32"),
		Destinations: []ProbeDestination{{
			AddrV4: mustPrefix("1.2.3.4/32"),
			Proto:  "tcp",
			Port:   80,
		}},
		ResolverDNS: true,
		ExpiresAt:   m.Now.Add(30 * time.Second),
	}
	return m
}

func probeServingFixture(m Model, assignments map[uuid.UUID]uint16, bounded bool) []byte {
	minor := assignments[m.Bindings[0].MemberID]
	classid := ClassID(minor)
	up := usageCounter(dirUpload, m.Bindings[0].DeviceID)
	dn := usageCounter(dirDownload, m.Bindings[0].DeviceID)
	in := m.Datapath.MemberIngress
	ul := m.Datapath.Uplink
	timeout := map[string]any{"timeout": 30, "expires": 29}
	if !bounded {
		timeout = map[string]any{}
	}
	doc := map[string]any{
		"nftables": []any{
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "forward", "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "comment": fmt.Sprintf("fvpn_generation=%d fvpn_metering=%s", m.Generation, MeteringVersion), "expr": []any{map[string]any{"match": map[string]any{"left": "never", "right": "never"}}}}},
			ruleExpr("forward", matchMeta("nfproto", "ipv6"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchPayload("!=", "saddr", "@allowed_v4"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), matchPayload("==", "daddr", "@resolver_v4"), matchDport("tcp", 53), namedCounter(counterProbeUp), map[string]any{"accept": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), matchPayload("==", "daddr", "@resolver_v4"), matchDport("udp", 53), namedCounter(counterProbeUp), map[string]any{"accept": nil}),
			ruleExpr("forward", matchPayload("==", "daddr", "@denied_v4"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchPayload("==", "daddr", "@host_v4"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), matchConcat("@probe_dst_v4"), namedCounter(counterProbeUp), map[string]any{"accept": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), namedCounter(up), manglePriority(classid), map[string]any{"accept": nil}),
			ruleExpr("forward", matchMetaIface("iifname", ul), matchMetaIface("oifname", in), matchPayload("==", "daddr", "@probe_v4"), matchEstablished(), namedCounter(counterProbeDown), map[string]any{"accept": nil}),
			ruleExpr("forward", matchMetaIface("iifname", ul), matchMetaIface("oifname", in), namedCounter(dn), manglePriority(classid), map[string]any{"accept": nil}),
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "input", "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}},
			ruleExpr("input", matchMetaIface("iifname", in), map[string]any{"drop": nil}),
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "output", "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "postrouting", "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}},
			ruleExpr("postrouting", matchMetaIface("oifname", ul), map[string]any{"snat": map[string]any{"addr": m.Datapath.SNAT.String()}}),
			ruleExpr("postrouting", matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), map[string]any{"snat": map[string]any{"addr": m.Datapath.SNAT.String()}}),
			map[string]any{"counter": map[string]any{"table": "fvpn", "name": up, "packets": 1, "bytes": 100}},
			map[string]any{"counter": map[string]any{"table": "fvpn", "name": dn, "packets": 1, "bytes": 100}},
			map[string]any{"counter": map[string]any{"table": "fvpn", "name": counterProbeUp, "packets": 0, "bytes": 0}},
			map[string]any{"counter": map[string]any{"table": "fvpn", "name": counterProbeDown, "packets": 0, "bytes": 0}},
			map[string]any{"set": map[string]any{"table": "fvpn", "name": "probe_v4", "elem": []any{elemVal("100.64.8.1", timeout)}}},
			map[string]any{"set": map[string]any{"table": "fvpn", "name": "probe_dst_v4", "elem": []any{elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, timeout)}}},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return raw
}

func probeBaselineFixture(m Model, bounded bool) []byte {
	in := m.Datapath.MemberIngress
	ul := m.Datapath.Uplink
	timeout := map[string]any{"timeout": 30, "expires": 29}
	if !bounded {
		timeout = map[string]any{}
	}
	doc := map[string]any{
		"nftables": []any{
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "forward", "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=" + MeteringVersion, "expr": []any{map[string]any{"match": map[string]any{"left": "never", "right": "never"}}}}},
			ruleExpr("forward", matchMeta("nfproto", "ipv6"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), matchPayload("==", "daddr", "@resolver_v4"), matchDport("tcp", 53), namedCounter(counterProbeUp), map[string]any{"accept": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), matchPayload("==", "daddr", "@resolver_v4"), matchDport("udp", 53), namedCounter(counterProbeUp), map[string]any{"accept": nil}),
			ruleExpr("forward", matchPayload("==", "daddr", "@denied_v4"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchPayload("==", "daddr", "@host_v4"), map[string]any{"drop": nil}),
			ruleExpr("forward", matchMetaIface("iifname", in), matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), matchConcat("@probe_dst_v4"), namedCounter(counterProbeUp), map[string]any{"accept": nil}),
			ruleExpr("forward", matchMetaIface("iifname", ul), matchMetaIface("oifname", in), matchPayload("==", "daddr", "@probe_v4"), matchEstablished(), namedCounter(counterProbeDown), map[string]any{"accept": nil}),
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "input", "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}},
			ruleExpr("input", matchMetaIface("iifname", in), map[string]any{"drop": nil}),
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "output", "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "postrouting", "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}},
			ruleExpr("postrouting", matchMetaIface("oifname", ul), matchPayload("==", "saddr", "@probe_v4"), map[string]any{"snat": map[string]any{"addr": m.Datapath.SNAT.String()}}),
			map[string]any{"counter": map[string]any{"table": "fvpn", "name": counterProbeUp, "packets": 0, "bytes": 0}},
			map[string]any{"counter": map[string]any{"table": "fvpn", "name": counterProbeDown, "packets": 0, "bytes": 0}},
			map[string]any{"set": map[string]any{"table": "fvpn", "name": "probe_v4", "elem": []any{elemVal("100.64.8.1", timeout)}}},
			map[string]any{"set": map[string]any{"table": "fvpn", "name": "probe_dst_v4", "elem": []any{elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, timeout)}}},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return raw
}

func appendForwardAccept(raw []byte) []byte {
	var doc struct {
		Nftables []json.RawMessage `json:"nftables"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	doc.Nftables = append(doc.Nftables, json.RawMessage(`{"rule":{"family":"inet","table":"fvpn","chain":"forward","expr":[{"counter":{"name":"ctbase"}},{"accept":null}]}}`))
	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return out
}

func ruleExpr(chain string, expr ...any) map[string]any {
	return map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": chain, "expr": expr}}
}

func matchMeta(key, val string) map[string]any {
	return map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": key}}, "right": val}}
}

func matchMetaIface(key, val string) map[string]any { return matchMeta(key, val) }

func matchPayload(op, field, right string) map[string]any {
	return map[string]any{"match": map[string]any{"op": op, "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": field}}, "right": right}}
}

func matchDport(proto string, port int) map[string]any {
	return map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"payload": map[string]any{"protocol": proto, "field": "dport"}}, "right": port}}
}

func matchConcat(set string) map[string]any {
	return map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"concat": []any{}}, "right": set}}
}

func matchEstablished() map[string]any {
	return map[string]any{"match": map[string]any{"op": "in", "left": map[string]any{"ct": map[string]any{"key": "state"}}, "right": []any{"established", "related"}}}
}

func namedCounter(name string) map[string]any {
	return map[string]any{"counter": map[string]any{"name": name}}
}

func manglePriority(classid uint32) map[string]any {
	return map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": classid}}
}

func elemVal(val any, extra map[string]any) map[string]any {
	body := map[string]any{"val": val}
	for k, v := range extra {
		body[k] = v
	}
	return map[string]any{"elem": body}
}
