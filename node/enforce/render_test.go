package enforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRender_Golden(t *testing.T) {
	plan, err := Render(goldenModel(), State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "serving.nft", plan.NFT)
	assertGolden(t, "serving.tc", plan.TC)
	if strings.Contains(plan.NFT, "meta mark") || strings.Contains(plan.NFT, "ct mark") || strings.Contains(plan.NFT, "flush ruleset") || strings.Contains(plan.NFT, "flowtable") {
		t.Fatalf("serving script uses a forbidden construct:\n%s", plan.NFT)
	}
	if strings.Contains(plan.TC, " fw ") || strings.Contains(plan.TC, "matchall") {
		t.Fatalf("tc script classifies with a filter:\n%s", plan.TC)
	}
	rules := plan.NFT[strings.Index(plan.NFT, "add rule"):]
	denyAt := strings.Index(rules, "ip daddr @denied_v4")
	dnsAt := strings.Index(rules, "tcp dport 53")
	allowDrop := strings.Index(rules, "ip saddr != @allowed_v4")
	usageAt := strings.Index(rules, `ip saddr 100.64.1.1 counter name "u_`)
	if allowDrop < 0 || dnsAt < 0 || denyAt < 0 || usageAt < 0 {
		t.Fatal("missing expected rules")
	}
	// DNS is an exception inside the deny list, so it is accepted before the
	// destination deny. Usage accepts come after that deny (§8.1, §10.2).
	if !(allowDrop < dnsAt && dnsAt < denyAt && denyAt < usageAt) {
		t.Fatalf("rule order allow-drop=%d dns=%d deny=%d usage=%d", allowDrop, dnsAt, denyAt, usageAt)
	}
	if !strings.Contains(plan.NFT, "timeout 3600s") {
		t.Fatal("expected 3600s element timeout")
	}
	base, err := RenderBaseline(goldenModel().Datapath, State{})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "baseline.nft", base)
	for _, line := range strings.Split(base, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "add rule") && strings.Contains(line, " accept") {
			t.Fatalf("baseline rule accepts traffic: %s", line)
		}
	}
}

func TestRender_SecondPassKeepsCounters(t *testing.T) {
	m := goldenModel()
	first, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	state := State{Table: true, Chains: map[string]bool{"forward": true, "input": true, "output": true, "postrouting": true}, Sets: map[string]bool{"allowed_v4": true, "denied_v4": true, "host_v4": true, "resolver_v4": true}, Counters: map[string]CounterValue{}}
	for _, name := range []string{usageCounter(dirUpload, devA1), usageCounter(dirDownload, devA1), "ctunku", "ctv6"} {
		state.Counters[name] = CounterValue{Packets: 4, Bytes: 400}
	}
	leaves := map[uint16]bool{}
	for _, minor := range first.Assignments {
		leaves[minor] = true
	}
	second, err := Render(m, state, TCState{Devices: map[string]TCDev{
		"fv-ih": {RootHTB: true, Leaves: leaves},
		"fv-fh": {RootHTB: true, Leaves: leaves},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(second.NFT, "add counter "+TableFamily+" "+TableName+" "+usageCounter(dirUpload, devA1)) {
		t.Fatal("recreated an existing upload counter")
	}
	if strings.Contains(second.NFT, "delete counter") || strings.Contains(second.NFT, "delete table") {
		t.Fatal("render deletes counters or the table")
	}
	if !strings.Contains(second.NFT, "add counter") {
		t.Fatal("expected new counters for devices that were not in state")
	}
	if strings.Contains(second.TC, "qdisc add") {
		t.Fatal("rate update rebuilt the root qdisc")
	}
	if strings.Contains(second.TC, "class replace") == false {
		t.Fatal("expected in-place class replace")
	}
	if first.Assignments[memberA] != second.Assignments[memberA] {
		t.Fatal("class assignment changed")
	}
}

func TestRender_TimeoutCeilAndExpiry(t *testing.T) {
	m := goldenModel()
	m.Leases[0].TimeoutCeilSeconds = 30
	m.Leases[1].TimeoutCeilSeconds = 30
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan.NFT, "timeout 3600s") || !strings.Contains(plan.NFT, "timeout 30s") {
		t.Fatal("ceil did not cap the kernel timeout")
	}
	// A wall clock stepped backwards must not lengthen the capped timeout.
	m.Now = m.Now.Add(-2 * time.Hour)
	plan, err = Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan.NFT, "timeout 7200s") || strings.Contains(plan.NFT, "timeout 3600s") {
		t.Fatal("clock step lengthened the kernel timeout")
	}
	m = goldenModel()
	m.Bindings[0].ValidUntil = m.Now.Add(-time.Minute)
	plan, err = Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan.NFT, "100.64.1.1 timeout") || strings.Contains(plan.NFT, "ip saddr 100.64.1.1 counter") {
		t.Fatal("expired device is still admitted")
	}
	if !strings.Contains(plan.NFT, "100.64.1.2 timeout") {
		t.Fatal("live device was dropped with the expired one")
	}
}

func TestRender_Rejects(t *testing.T) {
	m := goldenModel()
	m.IPv6Mode = "allow"
	if _, err := Render(m, State{}, TCState{}); err == nil {
		t.Fatal("v6 allow mode was accepted")
	}
	m = goldenModel()
	m.Datapath.MemberIngress = "eth0"
	if _, err := Render(m, State{}, TCState{}); err == nil {
		t.Fatal("non-fv interface was accepted")
	}
	m = goldenModel()
	m.Bindings[1].AllowedV4 = m.Bindings[0].AllowedV4
	if _, err := Render(m, State{}, TCState{}); err == nil {
		t.Fatal("duplicate address was accepted")
	}
	m = goldenModel()
	m.QoS.Members[0].Up.RateBPS = 40_000_000
	if _, err := Render(m, State{}, TCState{}); err == nil {
		t.Fatal("guarantee sum above the parent was accepted")
	}
	m = goldenModel()
	m.Generation = 0
	if _, err := Render(m, State{}, TCState{}); err == nil {
		t.Fatal("generation 0 was accepted as serving")
	}
}

func TestAssignClasses_StableAndCollision(t *testing.T) {
	a := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	want := PreferredMinor(a)
	var b uuid.UUID
	found := false
	for i := 0; i < 65536; i++ {
		u := a
		u[14] = byte(i)
		u[15] = byte(i >> 8)
		if u != a && PreferredMinor(u) == want {
			b = u
			found = true
			break
		}
	}
	if !found {
		t.Fatal("did not find a colliding member id")
	}
	first, err := AssignClasses([]uuid.UUID{a, b}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first[a] == first[b] {
		t.Fatal("collision was not resolved")
	}
	if first[a] != want && first[b] != want {
		t.Fatal("neither member kept the preferred minor")
	}
	second, err := AssignClasses([]uuid.UUID{b, a}, first)
	if err != nil {
		t.Fatal(err)
	}
	if second[a] != first[a] || second[b] != first[b] {
		t.Fatal("existing assignment was not kept")
	}
	_, err = AssignClasses([]uuid.UUID{a}, map[uuid.UUID]uint16{a: 0x20, b: 0x20})
	if err == nil {
		t.Fatal("duplicate minor was accepted")
	}
}

func TestDetectResets_AndLargeCounter(t *testing.T) {
	raw := []byte(`{"nftables":[{"counter":{"family":"inet","table":"fvpn","name":"u_019a00000000700080000000000000a1","packets":3,"bytes":9007199254740993}}]}`)
	rs, err := ParseRuleset(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := rs.Counters["u_019a00000000700080000000000000a1"]
	if got.Bytes != 9007199254740993 || got.Packets != 3 {
		t.Fatalf("counter parsed as %+v", got)
	}
	resets := DetectResets(map[string]CounterValue{"u_x": {Bytes: 10, Packets: 1}}, map[string]CounterValue{"u_x": {Bytes: 0, Packets: 0}})
	if len(resets) != 1 || resets[0].Reason != ReasonMeterEpoch {
		t.Fatalf("resets %+v", resets)
	}
	gone := DetectResets(map[string]CounterValue{"u_x": {Bytes: 1}}, map[string]CounterValue{})
	if len(gone) != 1 {
		t.Fatal("missing counter was hidden")
	}
}

func TestVerifyServing_AndFailures(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	raw := fixtureServing(m, plan.Assignments)
	if err := VerifyServing(raw, m, plan.Assignments); err != nil {
		t.Fatal(err)
	}
	qos := fixtureQoS(m, plan.Assignments)
	if err := VerifyQoS(qos, m, plan.Assignments); err != nil {
		t.Fatal(err)
	}
	if err := VerifyServing([]byte(`{"nftables":[{"flowtable":{"name":"f"}}]}`), m, plan.Assignments); err == nil || !strings.Contains(err.Error(), string(ReasonFlowtable)) {
		t.Fatalf("flowtable: %v", err)
	}
	badOrder := bytesReplace(raw, []byte(`"drop":null`), []byte(`"accept":null`), 1)
	if err := VerifyServing(badOrder, m, plan.Assignments); err == nil {
		t.Fatal("accept without the allow-set drop was accepted")
	}
	marked := bytesReplace(raw, []byte(`"priority"`), []byte(`"mark"`), 1)
	if err := VerifyServing(marked, m, plan.Assignments); err == nil || !strings.Contains(err.Error(), string(ReasonMark)) {
		t.Fatalf("mark: %v", err)
	}
	if err := EvaluateMeterPath(m.Bindings, []string{"192.0.2.34"}); err == nil || !strings.Contains(err.Error(), string(ReasonMeterPathInvalid)) {
		t.Fatalf("masquerade: %v", err)
	}
	if err := EvaluateMeterPath(m.Bindings, []string{"100.64.1.1"}); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBaseline(t *testing.T) {
	dp := goldenModel().Datapath
	raw := []byte(`{"nftables":[
		{"chain":{"family":"inet","table":"fvpn","name":"forward","type":"filter","hook":"forward","prio":-150,"policy":"drop"}},
		{"rule":{"family":"inet","table":"fvpn","chain":"forward","comment":"fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1","expr":[{"match":{"left":"never","right":"never"}}]}},
		{"rule":{"family":"inet","table":"fvpn","chain":"forward","expr":[{"match":{"left":{"meta":{"key":"iifname"}},"right":"fv-fh"}},{"drop":null}]}},
		{"chain":{"family":"inet","table":"fvpn","name":"input","type":"filter","hook":"input","prio":-150,"policy":"accept"}},
		{"rule":{"family":"inet","table":"fvpn","chain":"input","expr":[{"match":{"left":{"meta":{"key":"iifname"}},"right":"fv-fh"}},{"drop":null}]}},
		{"chain":{"family":"inet","table":"fvpn","name":"output","type":"filter","hook":"output","prio":-150,"policy":"accept"}}
	]}`)
	if err := VerifyBaseline(raw, dp); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBaseline([]byte(`{"nftables":[{"chain":{"table":"fvpn","name":"forward","type":"filter","hook":"forward","prio":0,"policy":"accept"}}]}`), dp); err == nil || !strings.Contains(err.Error(), string(ReasonIsolationBaselineMissing)) {
		t.Fatalf("baseline: %v", err)
	}
}

func TestApply_FailClosed(t *testing.T) {
	m := smallModel()
	f := &fakeExec{}
	listed := 0
	f.handler = func(name string, args []string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")
		if strings.Contains(key, "nft -j list") {
			listed++
			if listed == 1 {
				return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "Error: No such file or directory"}
			}
			return []byte(`{"nftables":[{"table":{"family":"inet","name":"fvpn"}}]}`), nil
		}
		if strings.Contains(key, "tc -j qdisc show") {
			return []byte(`[]`), nil
		}
		if strings.Contains(key, "qdisc del") {
			return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "RTNETLINK answers: No such file or directory"}
		}
		return nil, fmt.Errorf("unexpected %s", key)
	}
	_, err := Apply(context.Background(), f, m, nil)
	if err == nil || !strings.Contains(err.Error(), string(ReasonHookPriority)) && !strings.Contains(err.Error(), string(ReasonMeterPathInvalid)) && !strings.Contains(err.Error(), string(ReasonGenerationMismatch)) {
		t.Fatalf("apply err %v", err)
	}
	if len(f.nft) < 2 || !strings.Contains(f.nft[len(f.nft)-1], "fvpn_baseline=1") {
		t.Fatalf("baseline was not installed after a bad verify:\n%v", f.nft)
	}
}

func TestSamples_Namespaces(t *testing.T) {
	m := smallModel()
	id := devA1
	counters := map[string]CounterValue{
		usageCounter(dirUpload, id):    {Packets: 2, Bytes: 200},
		usageCounter(dirDNSUpload, id): {Packets: 1, Bytes: 70},
	}
	got, err := Samples(m, counters)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	for _, s := range got {
		if s.Direction == dirUpload && s.Namespace != MeteringVersion {
			t.Fatal(s)
		}
		if s.Direction == dirDNSUpload && s.Namespace != DNSNamespace {
			t.Fatal(s)
		}
		if s.BindingRevision != 7 || s.MemberID != memberA {
			t.Fatalf("%+v", s)
		}
	}
}

func TestDeadline_Earliest(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lease := MemberLease{
		MemberID:            memberA,
		EffectiveServiceEnd: now.Add(48 * time.Hour),
		NodeLeaseEnd:        now.Add(10 * time.Hour),
		SignedServerTime:    now,
		MaxOfflineSeconds:   3600,
	}
	got, ok := ForwardingDeadline(lease)
	if !ok || !got.Equal(now.Add(time.Hour)) {
		t.Fatalf("deadline %s ok %v", got, ok)
	}
	sec, admit := elementTimeout(Binding{ValidUntil: now.Add(10 * time.Minute)}, lease, now)
	if !admit || sec != 600 {
		t.Fatalf("binding cap %d admit %v", sec, admit)
	}
}

type fakeExec struct {
	nft     []string
	tc      []string
	handler func(name string, args []string) ([]byte, error)
}

func (f *fakeExec) NftBatch(_ context.Context, script string) error {
	f.nft = append(f.nft, script)
	return nil
}

func (f *fakeExec) TCBatch(_ context.Context, script string) error {
	f.tc = append(f.tc, script)
	return nil
}

func (f *fakeExec) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	return f.handler(name, args)
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if os.Getenv("FVPN_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v\n--- got ---\n%s", path, err, got)
	}
	if string(want) != got {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s", name, got)
	}
}

func goldenModel() Model {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return Model{
		Generation: 7,
		IPv6Mode:   IPv6Block,
		Now:        now,
		Datapath:   Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: mustAddr("192.0.2.37")},
		Bindings: []Binding{
			bind(memberA, devA1, "peer-a1", "100.64.1.1", now),
			bind(memberA, devA2, "peer-a2", "100.64.1.2", now),
			bind(memberB, devB1, "peer-b1", "100.64.2.1", now),
		},
		Leases:        []MemberLease{lease(memberA, now), lease(memberB, now)},
		Resolvers:     []netip.Addr{mustAddr("198.51.100.53")},
		ExtraDeniedV4: []netip.Prefix{mustPrefix("192.0.2.128/25")},
		HostV4:        []netip.Prefix{mustPrefix("192.0.2.33/32"), mustPrefix("192.0.2.37/32")},
		QoS: QoS{
			Up:       Rate{RateBPS: 40_000_000, CeilBPS: 40_000_000},
			Down:     Rate{RateBPS: 40_000_000, CeilBPS: 40_000_000},
			HostUp:   Rate{RateBPS: 1_000_000, CeilBPS: 40_000_000},
			HostDown: Rate{RateBPS: 1_000_000, CeilBPS: 40_000_000},
			Members: []MemberQoS{
				{MemberID: memberA, Up: Rate{20_000_000, 20_000_000}, Down: Rate{20_000_000, 20_000_000}},
				{MemberID: memberB, Up: Rate{10_000_000, 20_000_000}, Down: Rate{10_000_000, 20_000_000}},
			},
		},
	}
}

func smallModel() Model {
	m := goldenModel()
	m.Generation = 3
	m.Bindings = m.Bindings[:1]
	m.Leases = m.Leases[:1]
	m.QoS.Members = m.QoS.Members[:1]
	return m
}

func fixtureServing(m Model, assignments map[uuid.UUID]uint16) []byte {
	minor := assignments[m.Bindings[0].MemberID]
	classid := ClassID(minor)
	up := usageCounter(dirUpload, m.Bindings[0].DeviceID)
	dn := usageCounter(dirDownload, m.Bindings[0].DeviceID)
	doc := map[string]any{
		"nftables": []any{
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "forward", "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "comment": fmt.Sprintf("fvpn_generation=%d fvpn_metering=%s", m.Generation, MeteringVersion), "expr": []any{map[string]any{"match": map[string]any{"left": "never", "right": "never"}}}}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "expr": []any{
				map[string]any{"match": map[string]any{"op": "!=", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}},
				map[string]any{"counter": map[string]any{"name": "ctunku"}},
				map[string]any{"drop": nil},
			}}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"counter": map[string]any{"name": up}},
				map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": classid}},
				map[string]any{"accept": nil},
			}}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"counter": map[string]any{"name": dn}},
				map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": classid}},
				map[string]any{"accept": nil},
			}}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "input", "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "input", "expr": []any{
				map[string]any{"match": map[string]any{"left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"drop": nil},
			}}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "output", "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "postrouting", "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "postrouting", "expr": []any{
				map[string]any{"match": map[string]any{"left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.Uplink}},
				map[string]any{"snat": map[string]any{"addr": m.Datapath.SNAT.String()}},
			}}},
			map[string]any{"counter": map[string]any{"family": "inet", "table": "fvpn", "name": up, "packets": 1, "bytes": 100}},
			map[string]any{"counter": map[string]any{"family": "inet", "table": "fvpn", "name": dn, "packets": 1, "bytes": 100}},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return raw
}

func fixtureQoS(m Model, assignments map[uuid.UUID]uint16) map[string][2][]byte {
	minor := assignments[m.Bindings[0].MemberID]
	handle := classHandle(minor)
	q := fmt.Sprintf(`[{"kind":"htb","handle":"1:","root":true,"options":{"default":"fffe"}},{"kind":"fq_codel","handle":"%x:","parent":"%s"}]`, minor, handle)
	c := fmt.Sprintf(`[{"kind":"htb","handle":"1:1","options":{"rate":"40mbit","ceil":"40mbit"}},{"kind":"htb","handle":"1:fffe","options":{"rate":"1mbit","ceil":"40mbit"}},{"kind":"htb","handle":"%s","options":{"rate":"20mbit","ceil":"20mbit"}}]`, handle)
	pair := [2][]byte{[]byte(q), []byte(c)}
	return map[string][2][]byte{m.Datapath.Uplink: pair, m.Datapath.MemberIngress: pair}
}

func bytesReplace(raw []byte, old, new []byte, n int) []byte {
	return []byte(strings.Replace(string(raw), string(old), string(new), n))
}

var (
	memberA = uuid.MustParse("019a0000-0000-7000-8000-00000000000a")
	memberB = uuid.MustParse("019a0000-0000-7000-8000-00000000000b")
	devA1   = uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	devA2   = uuid.MustParse("019a0000-0000-7000-8000-0000000000a2")
	devB1   = uuid.MustParse("019a0000-0000-7000-8000-0000000000b1")
)

func mustAddr(s string) netip.Addr { return netip.MustParseAddr(s) }

func mustPrefix(s string) netip.Prefix { return netip.MustParsePrefix(s) }

func bind(member, device uuid.UUID, peer, ip string, now time.Time) Binding {
	return Binding{
		MemberID:        member,
		DeviceID:        device,
		PeerID:          peer,
		AllowedV4:       []netip.Prefix{mustPrefix(ip + "/32")},
		BindingRevision: 7,
		ValidFrom:       now.Add(-time.Minute),
		ValidUntil:      now.Add(24 * time.Hour),
	}
}

func lease(id uuid.UUID, now time.Time) MemberLease {
	return MemberLease{
		MemberID:            id,
		EffectiveServiceEnd: now.Add(48 * time.Hour),
		NodeLeaseEnd:        now.Add(48 * time.Hour),
		SignedServerTime:    now,
		MaxOfflineSeconds:   3600,
	}
}
