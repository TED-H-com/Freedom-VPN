package enforce

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// TEST ONLY synthetic listing. No bytes or field values are taken from the
// private F13-01 sample. fvfixture is independent of the live lab chains too.
func f23Chain() map[string]any {
	return map[string]any{"family": TableFamily, "table": TableName, "name": "fvfixture", "handle": 17,
		"type": "filter", "hook": "forward", "prio": 0, "policy": "drop", "comment": "TEST ONLY"}
}

func f23Listing(t *testing.T, objects ...map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"nftables": objects})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestF13_01_DuplicateChainKind(t *testing.T) {
	for _, field := range []string{"identical", "handle", "policy", "hook", "prio", "type", "comment", "unknown_field"} {
		t.Run(field, func(t *testing.T) {
			first, second := f23Chain(), f23Chain()
			switch field {
			case "identical":
			case "handle", "prio":
				second[field] = 33
			default:
				second[field] = "TEST ONLY changed"
			}
			raw := f23Listing(t, map[string]any{"chain": first}, map[string]any{"chain": second})
			if !json.Valid(raw) {
				t.Fatal("fixture is not valid JSON")
			}
			rs, rulesetErr := ParseRuleset(raw)
			st, stateErr := ParseState(raw)
			if !reflect.ValueOf(rs).IsZero() || !reflect.ValueOf(st).IsZero() {
				t.Fatal("ambiguous listing returned partial state")
			}
			for _, err := range []error{rulesetErr, stateErr} {
				var duplicate *DuplicateChainError
				if !errors.As(fmt.Errorf("wrapped: %w", err), &duplicate) {
					t.Fatalf("want duplicate_chain kind, got %T: %v", err, err)
				}
				if err.Error() != "duplicate chain inet fvpn fvfixture" {
					t.Fatalf("existing diagnostic changed: %v", err)
				}
			}
			// Existing verifier reasons and failure behavior are preserved. No
			// duplicate collapsing or automatic retry is introduced.
			requireCheck(t, verifyServingRuleset(raw, smallModel(), nil), ReasonMeterPathInvalid)
			requireCheck(t, VerifyBaseline(raw, smallModel()), ReasonIsolationBaselineMissing)
		})
	}
}

func TestF13_01_UniqueChainAndOtherParseErrors(t *testing.T) {
	for _, field := range []string{"single", "family", "table", "name"} {
		t.Run(field, func(t *testing.T) {
			objects := []map[string]any{{"chain": f23Chain()}}
			if field != "single" {
				second := f23Chain()
				second[field] = "fvother"
				if field == "family" {
					second[field] = "ip"
				}
				objects = append(objects, map[string]any{"chain": second})
			}
			raw := f23Listing(t, objects...)
			if _, err := ParseRuleset(raw); err != nil {
				t.Fatal(err)
			}
			if _, err := ParseState(raw); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, raw := range []string{`{"nftables":[`, `{"nftables":[{"chain":{"name":17}}]}`} {
		_, err := ParseState([]byte(raw))
		var duplicate *DuplicateChainError
		if err == nil || errors.As(err, &duplicate) {
			t.Fatalf("malformed listing confused with duplicate: %v", err)
		}
	}
}
