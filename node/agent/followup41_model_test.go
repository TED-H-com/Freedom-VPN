package agent

import (
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Same generation is deliberately kept throughout: these are renderer inputs,
// not a shortcut based on the desired document's generation or pointer.
func TestISO01_F41_RendererChangesAlwaysApply(t *testing.T) {
	cases := map[string]func(*Decision){
		"binding_address": func(d *Decision) {
			d.Model.Bindings[0].AllowedV4 = []netip.Prefix{netip.MustParsePrefix("100.64.0.19/32")}
		},
		"binding_revision": func(d *Decision) { d.Model.Bindings[0].BindingRevision++ },
		"binding_deadline": func(d *Decision) { d.Model.Bindings[0].ValidUntil = d.Model.Bindings[0].ValidUntil.Add(-time.Minute) },
		"member_deadline":  func(d *Decision) { *d.Model.Leases[0].DeadlineMono -= time.Minute },
		"member_wall_limit": func(d *Decision) {
			d.Model.Leases[0].EffectiveServiceEnd = d.Model.Leases[0].EffectiveServiceEnd.Add(-time.Minute)
		},
		"qos_rate":  func(d *Decision) { d.Model.QoS.Members[0].Up.RateBPS /= 2 },
		"qos_class": func(d *Decision) { d.Model.QoS.Assignments[d.Model.Bindings[0].MemberID]++ },
		"deny_set": func(d *Decision) {
			d.Model.DeniedV4 = append(enforce.SpecDeniedV4(), netip.MustParsePrefix("198.51.100.0/24"))
		},
		"extra_deny":   func(d *Decision) { d.Model.ExtraDeniedV4 = []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")} },
		"host_set":     func(d *Decision) { d.Model.HostV4 = []netip.Prefix{netip.MustParsePrefix("192.0.2.39/32")} },
		"resolver_set": func(d *Decision) { d.Model.Resolvers = []netip.Addr{netip.MustParseAddr("192.0.2.53")} },
		"probe": func(d *Decision) {
			d.Model.Probe = &enforce.ProbeException{PeerID: "TEST-ONLY-probe", SourceV4: netip.MustParsePrefix("100.64.9.9/32"), Destinations: []enforce.ProbeDestination{{AddrV4: netip.MustParsePrefix("203.0.113.9/32"), Proto: "tcp", Port: 443}}, ExpiresAt: d.Model.Now.Add(time.Minute)}
		},
		"datapath":      func(d *Decision) { d.Model.Datapath.SNAT = netip.MustParseAddr("192.0.2.38") },
		"policy_expiry": func(d *Decision) { d.PolicyExpires -= time.Second },
		"clock_anchor":  func(d *Decision) { d.Model.Now = d.Model.Now.Add(time.Second) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			r.clock.Advance(time.Second)
			r.eng.leaseRefreshPending = true
			stepOK(t, r)
			d := Decide(r.eng.input())
			// Clone mutable model fields so the test cannot alter the installed model.
			raw, err := json.Marshal(d.Model)
			if err != nil {
				t.Fatal(err)
			}
			var model enforce.Model
			if err := json.Unmarshal(raw, &model); err != nil {
				t.Fatal(err)
			}
			d.Model = model
			change(&d)
			before := r.plane.applies
			if err := r.eng.enforce(t.Context(), d); err != nil {
				t.Fatal(err)
			}
			if r.plane.applies != before+1 || !r.eng.serving {
				t.Fatalf("model change skipped or failed: applies=%d/%d err=%+v", r.plane.applies, before, r.eng.applyError)
			}
		})
	}
}

func TestMTR03_F41_SkippedTicksKeepCountingIdentity(t *testing.T) {
	r, src, up := f40Live(t)
	r.clock.Advance(time.Second)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	before := r.plane.applies
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	intent, _, err := r.eng.store.Meta(t.Context(), usageIntentKey)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		src.setCounter(up, enforce.CounterValue{Packets: uint64(2 + i), Bytes: uint64(8192 + i*1024)})
		r.clock.Advance(10 * time.Second)
		stepOK(t, r)
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	after, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := r.eng.store.Meta(t.Context(), usageIntentKey)
	if err != nil {
		t.Fatal(err)
	}
	if r.plane.applies != before || after.OpenEpoch != h.OpenEpoch || got != intent {
		t.Fatal("skipping changed apply/epoch/intent")
	}
	// A bound final sample includes every delta while the same rules counted.
	r.eng.handoff(t.Context())
	u := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
	if u.byRevision[1] != 11264 || len(u.gaps) != 0 {
		t.Fatalf("retained samples lost or recounted bytes: %+v", u)
	}
}
