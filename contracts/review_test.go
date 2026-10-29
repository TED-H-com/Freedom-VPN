package contracts

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestReviewC02_ArchitectureCoverageIsSemantic(t *testing.T) {
	ex := example{"ReleaseRegisterRequest.two-amd64.json", "ReleaseRegisterRequest", readFile(t, "examples/semantic-invalid/ReleaseRegisterRequest.two-amd64.json")}
	if err := validateComponent(t, ex.component, ex.raw); err != nil {
		t.Fatal(err)
	}
	if err := semanticCheck(t, ex); err != nil {
		t.Fatalf("fixture must have valid signature: %v", err)
	}
	var v struct {
		Manifest struct {
			Signed struct {
				Artifacts []struct {
					Architecture string `json:"architecture"`
				} `json:"artifacts"`
			} `json:"signed"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(ex.raw, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Manifest.Signed.Artifacts) != 2 {
		t.Fatal("fixture must isolate architecture from minItems")
	}
	for _, a := range v.Manifest.Signed.Artifacts {
		if a.Architecture != "amd64" {
			t.Fatal("fixture no longer violates coverage")
		}
	}
	if loadDoc(t).Components.Schemas["ReleaseManifest"].Value.Properties["artifacts"].Value.MinItems != 2 {
		t.Fatal("minItems must remain 2")
	}
}

func TestReviewC05_ExtraSpecMembersStayRequired(t *testing.T) {
	required := map[string][]string{
		"UpdateResult":  {"stage", "recovery_required"},
		"Operation":     {"resource", "error", "request_id", "updated_at", "completed_at"},
		"OperationStep": {"required", "observed_at", "error_code", "message"},
		"UpdateBatch":   {"operation_id", "requires_service_restart", "reason", "created_by", "created_at"},
	}
	for name, fields := range required {
		for _, field := range fields {
			if !slices.Contains(loadDoc(t).Components.Schemas[name].Value.Required, field) {
				t.Errorf("%s.%s relaxed", name, field)
			}
		}
	}
	for _, name := range []string{"UpdateResult", "Operation", "UpdateBatch"} {
		raw := readFile(t, "examples/valid/"+name+".json")
		if err := validateComponent(t, name, raw); err != nil {
			t.Fatal(err)
		}
		for _, field := range required[name] {
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			delete(doc, field)
			invalid, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateComponent(t, name, invalid); err == nil {
				t.Errorf("%s omitted %s accepted", name, field)
			}
		}
	}
}

func TestReviewC06_UsageAcceptsOptionalIdempotencyKey(t *testing.T) {
	op := loadDoc(t).Paths.Value("/agent/usage-batches").Post
	for _, p := range op.Parameters {
		if p.Value.Name == "Idempotency-Key" && p.Value.In == "header" && !p.Value.Required {
			return
		}
	}
	t.Fatal("optional Idempotency-Key missing")
}

func TestReviewC07_ConflictDocumentsAllCodes(t *testing.T) {
	desc := *loadDoc(t).Components.Responses["Conflict"].Value.Description
	for _, code := range []string{"UPDATE_BUSY", "AUTHORIZATION_EXPIRED", "STALE_GENERATION", "LEASE_EXPIRED"} {
		if !strings.Contains(desc, code) {
			t.Errorf("missing %s", code)
		}
	}
}

// These fixtures intentionally pass schema/hash/signature checks. The named
// owning services must reject the domain rule; this suite is not evidence of
// their enforcement (outside the contracts area).
func TestReviewC08_SemanticInvalidFixtures(t *testing.T) {
	var reasons map[string]struct{ Owner, Rule, Note string }
	readJSON(t, "examples/semantic-invalid/reasons.json", &reasons)
	for _, ex := range listExamples(t, "semantic-invalid") {
		t.Run(ex.file, func(t *testing.T) {
			r := reasons[ex.file]
			if r.Owner == "" || r.Rule == "" || r.Note == "" {
				t.Fatal("missing service ownership or rule")
			}
			if err := validateComponent(t, ex.component, ex.raw); err != nil {
				t.Fatalf("must pass schema: %v", err)
			}
			if err := semanticCheck(t, ex); err != nil {
				t.Fatalf("must have valid hash/signature: %v", err)
			}
			if file := jsonSchemaFileFor(ex.component); file != "" {
				if err := validateJSONSchema(t, file, ex.raw); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if len(reasons) != len(listExamples(t, "semantic-invalid")) {
		t.Fatal("missing semantic fixture")
	}
}

func TestReviewC09_ReturnURLRequiresBillingAllowlist(t *testing.T) {
	for _, name := range []string{"CheckoutSessionRequest", "BillingPortalSessionRequest"} {
		raw := readFile(t, "examples/semantic-invalid/"+name+".non-allowlisted.json")
		if err := validateComponent(t, name, raw); err != nil {
			t.Fatal(err)
		}
		var v struct {
			ReturnURL string `json:"return_url"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(v.ReturnURL, "https://unapproved.example.invalid/") {
			t.Fatal("fixture no longer tests the service allowlist")
		}
	}
}

func TestReviewC15_ExpandedCanonicalCorpus(t *testing.T) {
	var v struct{ Cases, Rejects []struct{ Name string } }
	readJSON(t, "vectors/canonical-json.json", &v)
	names := map[string]bool{}
	for _, c := range append(v.Cases, v.Rejects...) {
		names[c.Name] = true
	}
	for _, name := range []string{"nested-duplicate-key", "depth-256", "depth-257", "leading-plus", "positive-exponent", "utf16-differ-only-past-bmp"} {
		if !names[name] {
			t.Errorf("missing %s", name)
		}
	}
	TestCanonicalJSONVectors(t)
}
