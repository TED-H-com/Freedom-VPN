package enforce

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Regression for review r20 H-2. Before-fail evidence retains the original
// proof. Replace complete fixture sets to keep each listing unambiguous.
func TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets(t *testing.T) {
	t.Run("serving_emptied_and_timed_deny_and_host_sets", func(t *testing.T) {
		m := smallModel()
		m.ExtraDeniedV4 = []netip.Prefix{mustPrefix("192.0.2.128/25"), mustPrefix("198.51.100.0/24")}
		m.HostV4 = []netip.Prefix{mustPrefix("192.0.2.33/32"), mustPrefix("192.0.2.37/32")}
		plan, err := Render(m, State{}, TCState{})
		if err != nil {
			t.Fatal(err)
		}
		raw := fixtureServingWithDenyAndHost(m, plan.Assignments,
			map[string]any{
				"family": "inet",
				"table":  "fvpn",
				"name":   "denied_v4",
				"type":   "ipv4_addr",
				"flags":  []any{"interval", "timeout"},
				// Empty elem: missing all SpecDeniedV4() and ExtraDeniedV4 prefixes,
				// plus carries an illegal timeout flag.
			},
			map[string]any{
				"family": "inet",
				"table":  "fvpn",
				"name":   "host_v4",
				"type":   "ipv4_addr",
				"flags":  []any{"interval", "timeout"},
				// Empty elem: missing HostV4 and Datapath.SNAT/32.
			},
		)
		if err := verifyServingRuleset(raw, m, plan.Assignments); err == nil {
			t.Fatalf("H-2: verifyServingRuleset accepted emptied @denied_v4 and @host_v4 with timeout flags against a model requiring %d extra-denied and %d host prefixes",
				len(m.ExtraDeniedV4), len(m.HostV4))
		}
	})

	t.Run("probe_baseline_emptied_deny_host_and_nonempty_allowed_v4", func(t *testing.T) {
		m := smallModel()
		m.Probe = probeModel().Probe
		m.ExtraDeniedV4 = []netip.Prefix{mustPrefix("192.0.2.128/25")}
		m.HostV4 = []netip.Prefix{mustPrefix("192.0.2.33/32"), mustPrefix("192.0.2.37/32")}
		raw := probeBaselineFixture(m, true)
		raw = f27ReplaceSet(raw, map[string]any{
			"family": "inet",
			"table":  "fvpn",
			"name":   "allowed_v4",
			"type":   "ipv4_addr",
			"flags":  []any{"timeout"},
			"elem":   []any{elemVal("100.64.1.1", map[string]any{"timeout": 300, "expires": 299})},
		})
		raw = f27ReplaceSet(raw, map[string]any{
			"family": "inet",
			"table":  "fvpn",
			"name":   "denied_v4",
			"type":   "ipv4_addr",
			"flags":  []any{"interval", "timeout"},
		})
		raw = f27ReplaceSet(raw, map[string]any{
			"family": "inet",
			"table":  "fvpn",
			"name":   "host_v4",
			"type":   "ipv4_addr",
			"flags":  []any{"interval", "timeout"},
		})
		if err := VerifyBaseline(raw, m); err == nil {
			t.Fatal("H-2: VerifyBaseline accepted a probe baseline with emptied @denied_v4/@host_v4 and a non-empty @allowed_v4")
		}
	})
}

func TestR20_Held_VerifyServingCatchesResolverBindingQoSAndSNATDrift(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	baseRaw := fixtureServing(m, plan.Assignments)
	if err := verifyServingRuleset(baseRaw, m, plan.Assignments); err != nil {
		t.Fatalf("clean fixture failed: %v", err)
	}

	// 1. Resolver drift (missing or extra resolver IP) is caught by verifyResolverSet.
	mChangedResolver := m
	mChangedResolver.Resolvers = []netip.Addr{mustAddr("198.51.100.53"), mustAddr("198.51.100.54")}
	if err := verifyServingRuleset(baseRaw, mChangedResolver, plan.Assignments); err == nil || !strings.Contains(err.Error(), "resolver_v4 is not exactly the configured resolvers") {
		t.Fatalf("expected resolver_v4 mismatch error, got %v", err)
	}

	// 2. Generation drift is caught by verifyGeneration.
	mChangedGen := m
	mChangedGen.Generation = m.Generation + 1
	if err := verifyServingRuleset(baseRaw, mChangedGen, plan.Assignments); err == nil || !strings.Contains(err.Error(), string(ReasonGenerationMismatch)) {
		t.Fatalf("expected generation_mismatch, got %v", err)
	}

	// 3. Datapath SNAT drift is caught by verifyPostrouting.
	mChangedSNAT := m
	mChangedSNAT.Datapath.SNAT = mustAddr("192.0.2.99")
	if err := verifyServingRuleset(baseRaw, mChangedSNAT, plan.Assignments); err == nil || !strings.Contains(err.Error(), "postrouting") {
		t.Fatalf("expected postrouting error on SNAT drift, got %v", err)
	}

	// 4. Binding address / counter drift is caught by verifyForward and verifyAllowedElements.
	mChangedBinding := m
	mChangedBinding.Bindings = append([]Binding(nil), m.Bindings...)
	mChangedBinding.Bindings[0].AllowedV4 = []netip.Prefix{mustPrefix("100.64.1.99/32")}
	if err := verifyServingRuleset(baseRaw, mChangedBinding, plan.Assignments); err == nil {
		t.Fatal("expected verifyServingRuleset to reject binding address drift")
	}

	// 5. Lease / ValidUntil tightening is caught by verifyAllowedLifetimes when kernel expires exceeds remaining lifetime + slack.
	setRaw := []byte("table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout elements = { 100.64.1.1 timeout 3600s expires 1800s } } }")
	mTightLease := m
	mTightLease.Leases = append([]MemberLease(nil), m.Leases...)
	mTightLease.Leases[0].TimeoutCeilSeconds = 60
	if _, err := verifyAllowedLifetimes(setRaw, mTightLease, 0); err == nil {
		t.Fatal("expected verifyAllowedLifetimes to reject kernel element expires > tightened lease ceil")
	}

	// 6. QoS rate drift is caught by VerifyQoS.
	qos := fixtureQoS(m, plan.Assignments)
	mChangedQoS := m
	mChangedQoS.QoS.Members = append([]MemberQoS(nil), m.QoS.Members...)
	mChangedQoS.QoS.Members[0].Up = Rate{RateBPS: 10_000_000, CeilBPS: 10_000_000}
	if err := VerifyQoS(qos, mChangedQoS, plan.Assignments); err == nil || !strings.Contains(err.Error(), string(ReasonQoS)) {
		t.Fatalf("expected VerifyQoS to reject member ceil drift, got %v", err)
	}
}

func fixtureServingWithDenyAndHost(m Model, assignments map[uuid.UUID]uint16, deniedSet, hostSet map[string]any) []byte {
	raw := fixtureServing(m, assignments)
	return f27ReplaceSet(f27ReplaceSet(raw, deniedSet), hostSet)
}
