package enforce

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func ReasonCode(err error) Reason {
	var ce *CheckError
	if errors.As(err, &ce) && ce != nil {
		return ce.Reason
	}
	return ""
}

func TestReviewD063_VerifyBaseline_PrefixAndRangeElementsBypass(t *testing.T) {
	// TODO(review): parse_nft.go strips CIDR prefix lengths in elemFromParts and
	// silently drops nft -j prefix/range element objects in parseSetElems instead
	// of failing VerifyBaseline/VerifyServing with ReasonProbeUnbounded.
	t.Run("string_cidr_prefix_in_probe_sets", func(t *testing.T) {
		m := smallModel()
		m.Probe = probeModel().Probe
		raw := probeBaselineWithCustomElems(
			m,
			[]any{elemVal("100.64.8.1/24", map[string]any{"timeout": 30, "expires": 29})},
			[]any{elemVal("1.2.3.4/16 . tcp . 80", map[string]any{"timeout": 30, "expires": 29})},
		)
		err := VerifyBaseline(raw, m)
		if ReasonCode(err) != ReasonProbeUnbounded {
			t.Fatalf("VerifyBaseline accepted non-/32 CIDR elements in probe_v4 and probe_dst_v4: err=%v (reason=%q)", err, ReasonCode(err))
		}
	})

	t.Run("nft_json_prefix_and_range_objects_silently_dropped", func(t *testing.T) {
		m := smallModel()
		m.Probe = probeModel().Probe
		timeout := map[string]any{"timeout": 30, "expires": 29}
		raw := probeBaselineWithCustomElems(
			m,
			[]any{
				elemVal("100.64.8.1", timeout),
				elemVal(map[string]any{"prefix": map[string]any{"addr": "100.64.0.0", "len": 10}}, timeout),
			},
			[]any{
				elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, timeout),
				elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", map[string]any{"range": []any{1, 65535}}}}, timeout),
			},
		)
		err := VerifyBaseline(raw, m)
		if ReasonCode(err) != ReasonProbeUnbounded {
			t.Fatalf("VerifyBaseline silently ignored prefix/range objects in probe sets: err=%v (reason=%q)", err, ReasonCode(err))
		}
	})

	t.Run("expires_exceeding_max_or_model_window_ignored", func(t *testing.T) {
		m := smallModel()
		m.Probe = probeModel().Probe
		m.Probe.ExpiresAt = m.Now.Add(5 * time.Second)
		raw := probeBaselineWithCustomElems(
			m,
			[]any{elemVal("100.64.8.1", map[string]any{"timeout": 30, "expires": ProbeMaxSeconds + 600})},
			[]any{elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, map[string]any{"timeout": 300, "expires": 299})},
		)
		err := VerifyBaseline(raw, m)
		if ReasonCode(err) != ReasonProbeUnbounded {
			t.Fatalf("VerifyBaseline accepted element expires=%d and timeout=300s for a 5s probe: err=%v (reason=%q)", ProbeMaxSeconds+600, err, ReasonCode(err))
		}
	})
}

func TestReviewD063_VerifyBaseline_ModelBoundsReturnPlainErrorNotProbeUnbounded(t *testing.T) {
	// TODO(review): VerifyBaseline delegates model-level probe checks to
	// validateProbe(m), which returns a plain fmt.Errorf without wrapping in
	// *CheckError{Reason: ReasonProbeUnbounded}.
	m := smallModel()
	m.Probe = probeModel().Probe
	raw := probeBaselineFixture(m, true)

	cases := []struct {
		name string
		edit func(*Model)
	}{
		{"wide_source", func(m *Model) { m.Probe.SourceV4 = mustPrefix("100.64.8.0/24") }},
		{"wide_dest", func(m *Model) { m.Probe.Destinations[0].AddrV4 = mustPrefix("1.2.3.0/24") }},
		{"port_zero", func(m *Model) { m.Probe.Destinations[0].Port = 0 }},
		{"bad_proto", func(m *Model) { m.Probe.Destinations[0].Proto = "icmp" }},
		{"past_expiry", func(m *Model) { m.Probe.ExpiresAt = m.Now.Add(-time.Second) }},
		{"excess_expiry", func(m *Model) {
			m.Probe.ExpiresAt = m.Now.Add(time.Duration(ProbeMaxSeconds)*time.Second + time.Second)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := m
			cp := *m.Probe
			cp.Destinations = append([]ProbeDestination(nil), m.Probe.Destinations...)
			bad.Probe = &cp
			tc.edit(&bad)
			err := VerifyBaseline(raw, bad)
			if ReasonCode(err) != ReasonProbeUnbounded {
				t.Fatalf("VerifyBaseline(%s) returned reason=%q err=%v, want %q", tc.name, ReasonCode(err), err, ReasonProbeUnbounded)
			}
		})
	}
}

func TestReviewD063_RenderBaseline_DestinationCollidesWithMemberAndUnboundedCount(t *testing.T) {
	// TODO(review): validateProbeDestination does not check d.AddrV4 against
	// m.Bindings[i].AllowedV4, and validateProbe has no cap on len(p.Destinations).
	t.Run("destination_equal_to_member_binding_allowed", func(t *testing.T) {
		m := probeModel()
		m.DeniedV4 = Without(SpecDeniedV4(), mustPrefix("100.64.0.0/10"))
		m.Probe.Destinations[0].AddrV4 = m.Bindings[0].AllowedV4[0]
		if _, err := Render(m, State{}, TCState{}); err == nil {
			t.Fatalf("Render accepted probe destination %s equal to member binding %s", m.Probe.Destinations[0].AddrV4, m.Bindings[0].AllowedV4[0])
		}
	})

	t.Run("unbounded_destination_count", func(t *testing.T) {
		m := probeModel()
		dests := make([]ProbeDestination, 0, 256)
		for i := 1; i <= 256; i++ {
			dests = append(dests, ProbeDestination{
				AddrV4: mustPrefix(fmt.Sprintf("1.2.3.%d/32", (i%250)+1)),
				Proto:  "tcp",
				Port:   uint16(1000 + i),
			})
		}
		m.Probe.Destinations = dests
		if _, err := RenderBaseline(m, State{}); err == nil {
			t.Fatalf("RenderBaseline accepted %d probe destinations without an upper bound", len(dests))
		}
	})
}

func TestReviewD063_OverlappingProbeResetsKernelTimeoutPastMax(t *testing.T) {
	// TODO(review): ProbeException has no TimeoutCeilSeconds equivalent (unlike
	// MemberLease.TimeoutCeilSeconds); a second render while a probe is active
	// flushes @probe_v4/@probe_dst_v4 and resets the kernel timeout to another
	// full ProbeMaxSeconds window from the new m.Now.
	t0 := goldenModel().Now
	m1 := probeModel()
	m1.Now = t0
	m1.Probe.ExpiresAt = t0.Add(time.Duration(ProbeMaxSeconds) * time.Second)
	s1, err := RenderBaseline(m1, State{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s1, "100.64.8.1 timeout 300s") {
		t.Fatalf("first render missing 300s timeout:\n%s", s1)
	}

	// 200s later, an overlapping probe is rendered while probe_v4 is already live in State.
	m2 := m1
	cp := *m1.Probe
	m2.Probe = &cp
	m2.Now = t0.Add(200 * time.Second)
	m2.Probe.ExpiresAt = m2.Now.Add(time.Duration(ProbeMaxSeconds) * time.Second)
	s2, err := RenderBaseline(m2, State{Sets: map[string]bool{setProbeV4: true, setProbeDstV4: true}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s2, "100.64.8.1 timeout 300s") {
		t.Fatalf("overlapping render reset probe_v4 timeout to 300s at t0+200s (total lifetime 500s > ProbeMaxSeconds=%ds)", ProbeMaxSeconds)
	}
}

func TestReviewD063_Held_ProbeBoundsAndMemberIsolation(t *testing.T) {
	m := probeModel()
	base, err := RenderBaseline(m, State{})
	if err != nil {
		t.Fatal(err)
	}
	// While forwarding is blocked (RenderBaseline), forward policy is drop and
	// every accept rule is gated on @probe_v4 containing only 100.64.8.1/32.
	// A real member address (100.64.1.1) targeting the probe destination (1.2.3.4:80)
	// cannot match @probe_v4.
	if !strings.Contains(base, "type filter hook forward priority -150; policy drop;") {
		t.Fatalf("baseline forward chain does not default to drop:\n%s", base)
	}
	for _, line := range strings.Split(base, "\n") {
		if !strings.Contains(line, "add rule inet fvpn forward") || !strings.Contains(line, " accept") {
			continue
		}
		if !strings.Contains(line, "@probe_v4") {
			t.Fatalf("baseline forward accept rule is not scoped to @probe_v4: %s", line)
		}
	}
	if strings.Contains(base, "100.64.1.1") {
		t.Fatalf("baseline ruleset contains member address 100.64.1.1:\n%s", base)
	}

	// VerifyBaseline catches unbounded timeout, excess expires, and unexpected set elements
	// with ReasonProbeUnbounded.
	sm := smallModel()
	sm.Probe = probeModel().Probe
	overTimeout := probeBaselineWithCustomElems(
		sm,
		[]any{elemVal("100.64.8.1", map[string]any{"timeout": ProbeMaxSeconds + 1, "expires": 30})},
		[]any{elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, map[string]any{"timeout": 30, "expires": 29})},
	)
	if err := VerifyBaseline(overTimeout, sm); ReasonCode(err) != ReasonProbeUnbounded {
		t.Fatalf("over-max timeout reason=%q err=%v", ReasonCode(err), err)
	}
	extraMemberElem := probeBaselineWithCustomElems(
		sm,
		[]any{
			elemVal("100.64.8.1", map[string]any{"timeout": 30, "expires": 29}),
			elemVal("100.64.1.1", map[string]any{"timeout": 30, "expires": 29}),
		},
		[]any{elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, map[string]any{"timeout": 30, "expires": 29})},
	)
	if err := VerifyBaseline(extraMemberElem, sm); ReasonCode(err) != ReasonIsolation {
		t.Fatalf("extra member element in probe_v4 reason=%q err=%v", ReasonCode(err), err)
	}
}

func TestReviewD063_Held_ProbeCountersIsolatedFromMemberUsage(t *testing.T) {
	m := smallModel()
	m.Probe = probeModel().Probe
	dev := m.Bindings[0].DeviceID
	upName := usageCounter(dirUpload, dev)
	downName := usageCounter(dirDownload, dev)

	prev := map[string]CounterValue{
		upName:           {Packets: 10, Bytes: 1000},
		downName:         {Packets: 20, Bytes: 2000},
		counterProbeUp:   {Packets: 500, Bytes: 50000},
		counterProbeDown: {Packets: 600, Bytes: 60000},
	}
	curr := map[string]CounterValue{
		upName:           {Packets: 15, Bytes: 1500},
		downName:         {Packets: 25, Bytes: 2500},
		counterProbeUp:   {Packets: 0, Bytes: 0}, // probe counter reset on clear
		counterProbeDown: {Packets: 0, Bytes: 0},
	}
	resets := DetectMeterResets(
		map[string]CounterValue{
			upName:   prev[upName],
			downName: prev[downName],
		},
		State{
			Table:      true,
			Chains:     map[string]bool{"forward": true},
			Generation: m.Generation,
			Counters:   curr,
		},
	)
	if len(resets) != 0 {
		t.Fatalf("probe counter reset triggered member reset detection: %+v", resets)
	}
	samples, err := Samples(m, curr)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 directional member samples, got %+v", samples)
	}
	for _, s := range samples {
		if s.DeviceID != dev {
			t.Fatalf("unexpected device in samples: %+v", s)
		}
		if s.Direction == dirUpload && (s.Packets != 15 || s.Bytes != 1500) {
			t.Fatalf("upload sample altered by probe counters: %+v", s)
		}
		if s.Direction == dirDownload && (s.Packets != 25 || s.Bytes != 2500) {
			t.Fatalf("download sample altered by probe counters: %+v", s)
		}
	}
}

func probeBaselineWithCustomElems(m Model, probeV4Elems, probeDstElems []any) []byte {
	in := m.Datapath.MemberIngress
	ul := m.Datapath.Uplink
	if m.Datapath.SNAT == (netip.Addr{}) {
		m.Datapath.SNAT = netip.MustParseAddr("192.0.2.37")
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
			map[string]any{"set": map[string]any{"table": "fvpn", "name": "probe_v4", "elem": probeV4Elems}},
			map[string]any{"set": map[string]any{"table": "fvpn", "name": "probe_dst_v4", "elem": probeDstElems}},
			resolverSetFixture(m),
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return raw
}
