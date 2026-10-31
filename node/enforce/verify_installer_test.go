package enforce

import (
	"fmt"
	"strings"
	"testing"
)

// D-116: the installer owns a separate, unreachable marker table. Parse the
// complete listing; never construct a Ruleset that has already lost that table.
func TestD116_InstallerMarkerScope(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	good := fixtureServing(m, plan.Assignments)
	broken := mutateChainPolicy(t, good, "forward", "accept")
	table := map[string]any{"table": map[string]any{"family": "inet", "name": "fvpn_install", "comment": "managed-by=fvpn-node-install"}}
	owner := map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn_install", "name": "owner", "comment": "managed-by=fvpn-node-install"}}
	// Anonymous counter and drop reference no other object. With no hook or
	// incoming jump this rule cannot see a packet; adding a hook must reject it.
	rule := map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn_install", "chain": "owner", "expr": []any{map[string]any{"counter": map[string]any{"packets": 0, "bytes": 0}}, map[string]any{"drop": nil}}}}
	for _, tc := range []struct {
		name    string
		objects []map[string]any
		foreign int
	}{
		{"table_comment", []map[string]any{table}, 0},
		{"regular_chain", []map[string]any{table, owner}, 1},
		{"unreachable_rules", []map[string]any{table, owner, rule}, 1},
		{"owned_table_comment", []map[string]any{{"table": map[string]any{"family": "inet", "name": "fvpn", "comment": "managed-by=fvpn-node-install"}}}, 0},
	} {
		for _, baseline := range []struct {
			name   string
			raw    []byte
			reason Reason
		}{{"green", good, ""}, {"broken", broken, ReasonHookPriority}} {
			t.Run(tc.name+"/"+baseline.name, func(t *testing.T) {
				want := verifyServingRuleset(baseline.raw, m, plan.Assignments)
				if ReasonCode(want) != baseline.reason {
					t.Fatalf("control: %v", want)
				}
				raw := baseline.raw
				for _, obj := range tc.objects {
					raw = appendNft(raw, obj)
				}
				rs, err := ParseRuleset(raw)
				if err != nil {
					t.Fatal(err)
				}
				if len(rs.Chains) != 4 || len(rs.Foreign) != tc.foreign {
					t.Fatalf("scope: owned=%d foreign=%d", len(rs.Chains), len(rs.Foreign))
				}
				if err := verifyForeign(rs, m); err != nil {
					t.Fatalf("unhooked marker: %v", err)
				}
				got := verifyServingRuleset(raw, m, plan.Assignments)
				if fmt.Sprint(got) != fmt.Sprint(want) {
					t.Fatalf("changed verdict: %v -> %v", want, got)
				}
			})
		}
	}
	t.Run("hooked_drop", func(t *testing.T) {
		hooked := map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn_install", "name": "owner", "type": "filter", "hook": "forward", "prio": 0, "policy": "accept", "comment": "managed-by=fvpn-node-install"}}
		for _, base := range [][]byte{good, broken} {
			raw := appendNft(appendNft(appendNft(base, table), hooked), rule)
			rs, err := ParseRuleset(raw)
			if err != nil {
				t.Fatal(err)
			}
			for _, err := range []error{verifyForeign(rs, m), verifyServingRuleset(raw, m, plan.Assignments)} {
				if ReasonCode(err) != ReasonIsolation || !strings.Contains(err.Error(), "table fvpn_install family inet hook forward priority 0 chain owner statement drop") {
					t.Fatalf("hooked control: %v", err)
				}
			}
		}
	})
	t.Run("owner_inside_fvpn", func(t *testing.T) {
		raw := appendNft(good, map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "owner", "comment": "managed-by=fvpn-node-install"}})
		rs, err := ParseRuleset(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range []error{verifyChainSet(rs), verifyServingRuleset(raw, m, plan.Assignments)} {
			if ReasonCode(err) != ReasonHookPriority {
				t.Fatalf("extra owner: %v", err)
			}
		}
	})
}
