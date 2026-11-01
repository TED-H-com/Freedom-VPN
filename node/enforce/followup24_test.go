package enforce

import (
	"encoding/json"
	"strings"
	"testing"
)

// AT13/ISO02: a live conntrack entry cannot bypass current remote restrictions.
func TestAT13_DownloadSourceVerification(t *testing.T) {
	for _, mode := range []string{"member", "probe-serving", "probe-baseline"} {
		t.Run(mode, func(t *testing.T) {
			m := smallModel()
			if mode != "member" {
				m.Probe = probeModel().Probe
			}
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				t.Fatal(err)
			}
			raw := fixtureServing(m, plan.Assignments)
			verify := func(b []byte) error { return verifyServingRuleset(b, m, plan.Assignments) }
			if mode == "probe-serving" {
				raw = probeServingFixture(m, plan.Assignments, true)
			} else if mode == "probe-baseline" {
				raw = probeBaselineFixture(m, true)
				verify = func(b []byte) error { return VerifyBaseline(b, m) }
			}
			if err := verify(raw); err != nil {
				t.Fatalf("positive control: %v", err)
			}
			for _, set := range []string{"denied_v4", "host_v4", "resolver_v4"} {
				for _, mutation := range []string{"missing", "late", "wrong-interface", "extra-destination", "ct-invalid", "wrong-counter"} {
					t.Run(set+"/"+mutation, func(t *testing.T) {
						bad := mutateDownloadGuard(t, raw, set, mutation)
						if err := verify(bad); ReasonCode(err) != ReasonIsolation {
							t.Fatalf("unsafe download guard accepted: %v", err)
						}
					})
				}
			}
		})
	}
}

func mutateDownloadGuard(t *testing.T, raw []byte, set, mutation string) []byte {
	t.Helper()
	var doc struct {
		Nftables []map[string]any `json:"nftables"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	found := false
	for i, obj := range doc.Nftables {
		r, ok := obj["rule"].(map[string]any)
		if !ok {
			continue
		}
		b, _ := json.Marshal(r)
		if !strings.Contains(string(b), `"name":"ctdenyd"`) || !strings.Contains(string(b), `"@`+set+`"`) {
			continue
		}
		found = true
		expr := r["expr"].([]any)
		switch mutation {
		case "missing", "late":
			doc.Nftables = append(doc.Nftables[:i], doc.Nftables[i+1:]...)
			if mutation == "late" {
				doc.Nftables = append(doc.Nftables, obj)
			}
		case "wrong-interface":
			expr[0] = matchMetaIface("iifname", "fv-wrong")
		case "extra-destination":
			r["expr"] = append([]any{matchPayload("==", "daddr", "100.64.1.99")}, expr...)
		case "ct-invalid":
			r["expr"] = append([]any{matchCTState("==", "invalid")}, expr...)
		case "wrong-counter":
			expr[3] = namedCounter(counterDenied)
		}
		break
	}
	if !found {
		t.Fatal("fixture lacks source guard")
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMTR05_DownloadDenialIsDiagnostic(t *testing.T) {
	m := smallModel()
	samples, err := Samples(m, map[string]CounterValue{counterDeniedDown: {Packets: 4, Bytes: 116}})
	if err != nil || len(samples) != 0 || !diagnosticNamespace(counterDeniedDown) {
		t.Fatalf("download denial must be excluded from usage: samples=%+v err=%v", samples, err)
	}
}
