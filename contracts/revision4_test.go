package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// D-048 / §13.10.3: unknown evidence must never become generation zero or a
// fabricated policy list, including after a generated-model round trip.
func TestUPD05_UpdateTargetPolicyEvidencePresence(t *testing.T) {
	schema := loadDoc(t).Components.Schemas["UpdateTarget"].Value
	for _, name := range []string{"policy_generation", "policy_refs"} {
		if slices.Contains(schema.Required, name) || !schema.Properties[name].Value.Nullable {
			t.Fatalf("%s must be optional and nullable", name)
		}
	}
	for _, tc := range []struct {
		file      string
		specified bool
		null      bool
	}{
		{"UpdateBatch.json", false, false},
		{"UpdateBatch.applied-policy.json", true, false},
		{"UpdateBatch.unknown-policy.json", true, true},
	} {
		t.Run(tc.file, func(t *testing.T) {
			raw := readFile(t, "examples/valid/"+tc.file)
			var batch api.UpdateBatch
			if err := json.Unmarshal(raw, &batch); err != nil {
				t.Fatal(err)
			}
			target := batch.Targets[0]
			if target.PolicyGeneration.IsSpecified() != tc.specified || target.PolicyRefs.IsSpecified() != tc.specified ||
				target.PolicyGeneration.IsNull() != tc.null || target.PolicyRefs.IsNull() != tc.null {
				t.Fatal("evidence presence changed")
			}
			generation, genErr := target.PolicyGeneration.Get()
			refs, refsErr := target.PolicyRefs.Get()
			switch {
			case !tc.specified:
				if !errors.Is(genErr, presence.ErrUnspecified) || !errors.Is(refsErr, presence.ErrUnspecified) {
					t.Fatal("absent evidence became a value")
				}
			case tc.null:
				if !errors.Is(genErr, presence.ErrNull) || !errors.Is(refsErr, presence.ErrNull) {
					t.Fatal("null evidence became a value")
				}
			default:
				if genErr != nil || refsErr != nil || generation.String() != "9007199254740993" || len(refs) != 1 || refs[0].String() != "3df64c99-216c-4704-bd07-a4f6fa4dc412" {
					t.Fatal("reported evidence lost precision or changed references")
				}
			}
			encoded, err := json.Marshal(batch)
			if err != nil {
				t.Fatal(err)
			}
			want, err := canonjson.Canonicalize(raw)
			if err != nil {
				t.Fatal(err)
			}
			got, err := canonjson.Canonicalize(encoded)
			if err != nil || string(want) != string(got) {
				t.Fatalf("evidence round trip changed JSON: %s (%v)", encoded, err)
			}
		})
	}
}

func TestUPD05_UpdateTargetPolicyEvidenceBounds(t *testing.T) {
	var batch struct {
		Targets []map[string]any `json:"targets"`
	}
	readJSON(t, "examples/valid/UpdateBatch.json", &batch)
	refs := make([]string, 65)
	for i := range refs {
		refs[i] = fmt.Sprintf("%08x-216c-4704-bd07-a4f6fa4dc412", i)
	}
	for _, tc := range []struct {
		name       string
		generation any
		refs       any
		valid      bool
	}{
		{"known_zero_empty", "0", []string{}, true},
		{"max_uint64", "18446744073709551615", nil, true},
		{"refs_only", nil, []string{"3df64c99-216c-4704-bd07-a4f6fa4dc412"}, true},
		{"number", 1, nil, false},
		{"leading_zero", "01", nil, false},
		{"negative", "-1", nil, false},
		{"too_long", "100000000000000000000", nil, false},
		{"duplicate_refs", nil, []string{"3df64c99-216c-4704-bd07-a4f6fa4dc412", "3df64c99-216c-4704-bd07-a4f6fa4dc412"}, false},
		{"bad_uuid", nil, []string{"invalid"}, false},
		{"max_refs", nil, refs[:64], true},
		{"too_many_refs", nil, refs, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := batch.Targets[0]
			target["policy_generation"], target["policy_refs"] = tc.generation, tc.refs
			raw, err := json.Marshal(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateComponent(t, "UpdateTarget", raw); (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
			if tc.valid {
				var model api.UpdateTarget
				if err := json.Unmarshal(raw, &model); err != nil {
					t.Fatal(err)
				}
				encoded, err := json.Marshal(model)
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				if err := json.Unmarshal(encoded, &got); err != nil {
					t.Fatal(err)
				}
				var want map[string]any
				if err := json.Unmarshal(raw, &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatal("explicit zero, empty list or null lost")
				}
			}
		})
	}
	var model api.UpdateTarget
	if err := json.Unmarshal([]byte(`{"policy_generation":"18446744073709551616"}`), &model); !errors.Is(err, decstr.ErrRange) {
		t.Fatalf("uint64 overflow must fail typed decoding: %v", err)
	}
	// D-049: making target evidence optional must not optionalize result evidence.
	if !slices.Contains(loadDoc(t).Components.Schemas["UpdateResult"].Value.Required, "policy_generation") {
		t.Fatal("UpdateResult.policy_generation must remain required")
	}
}

// Every pre-existing valid example remains accepted; newly added examples are
// included as well so future additions get the same compatibility coverage.
func TestAPI01_Revision4ExistingExamples(t *testing.T) {
	for _, ex := range listExamples(t, "valid") {
		t.Run(ex.file, func(t *testing.T) {
			if err := validateComponent(t, ex.component, ex.raw); err != nil {
				t.Fatal(err)
			}
			if err := semanticCheck(t, ex); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAPI01_DomainRulePositiveRevision(t *testing.T) {
	var request struct {
		Payload struct {
			Rules []map[string]any `json:"rules"`
		} `json:"payload"`
	}
	readJSON(t, "examples/valid/PolicyRevisionCreate.domain-rule-set.json", &request)
	rule := request.Payload.Rules[0]
	for _, revision := range []string{"1", "9007199254740993", "18446744073709551615", "0", "01", "-1", "1.0", "100000000000000000000"} {
		t.Run(revision, func(t *testing.T) {
			rule["revision"] = revision
			raw, err := json.Marshal(rule)
			if err != nil {
				t.Fatal(err)
			}
			valid := revision == "1" || revision == "9007199254740993" || revision == "18446744073709551615"
			if err := validateComponent(t, "DomainRule", raw); (err == nil) != valid {
				t.Fatalf("revision %s: %v", revision, err)
			}
			if valid {
				var model api.DomainRule
				if err := json.Unmarshal(raw, &model); err != nil || model.Revision.String() != revision {
					t.Fatalf("revision changed on decode: %v", err)
				}
			}
		})
	}
	var model api.DomainRule
	if err := json.Unmarshal([]byte(`{"revision":"18446744073709551616"}`), &model); !errors.Is(err, decstr.ErrRange) {
		t.Fatalf("uint64 upper bound must fail typed decoding: %v", err)
	}
}
