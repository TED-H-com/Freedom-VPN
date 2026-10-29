package contracts

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// The task requires preserving the owner's schemas, including known limitations.
func TestAPI01_PromotedPolicyPayloads(t *testing.T) {
	for _, kind := range []string{"plan", "fair-use", "dns-profile", "domain-rule-set", "threat-feed", "release-support"} {
		t.Run(kind, func(t *testing.T) {
			file := kind + "-payload.schema.json"
			owner := readFile(t, "../internal/policies/schemas/"+file)
			promoted := readFile(t, "schemas/policies/"+file)
			embedded, err := fs.ReadFile(Schemas(), "policies/"+file)
			if err != nil || !bytes.Equal(owner, promoted) || !bytes.Equal(embedded, promoted) {
				t.Fatal("promotion or embedding changed owner bytes", err)
			}
			raw := readFile(t, "examples/policies/"+kind+".json")
			if err := validateJSONSchema(t, "policies/"+file, raw); err != nil {
				t.Fatal(err)
			}
			// These are the payloads used by the owner's TestContractExamplesValidate.
			var source struct {
				Payload json.RawMessage `json:"payload"`
			}
			readJSON(t, "examples/valid/PolicyRevisionCreate."+kind+".json", &source)
			var got, want map[string]any
			_ = json.Unmarshal(raw, &got)
			_ = json.Unmarshal(source.Payload, &want)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if !bytes.Equal(gotJSON, wantJSON) {
				t.Fatal("example differs from owner test payload")
			}
			got["unknown_member"] = true
			bad, _ := json.Marshal(got)
			if err := validateJSONSchema(t, "policies/"+file, bad); err == nil {
				t.Fatal("unknown field accepted")
			}
			if err := validateJSONSchema(t, "policies/"+file, []byte(`{}`)); err == nil {
				t.Fatal("missing required payload accepted")
			}
		})
	}
}

func TestUPD02_ReleaseTrustUpdateSchema(t *testing.T) {
	const schema = "trust/release-update-envelope.schema.json"
	raw := readFile(t, "examples/trust/valid/release-update.json")
	if err := validateJSONSchema(t, schema, raw); err != nil {
		t.Fatal(err)
	}
	trust, now := testTrustStore(t)
	if _, _, err := envelope.Verify(raw, trust, envelope.PurposeRelease, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := envelope.Verify(raw, trust, envelope.PurposeDesiredState, now); err == nil {
		t.Fatal("trust update accepted for wrong purpose")
	}
	files, err := filepath.Glob("examples/trust/invalid/*.json")
	if err != nil || len(files) != 6 {
		t.Fatal("missing negative fixtures", err)
	}
	for _, file := range files {
		if err := validateJSONSchema(t, schema, readFile(t, file)); err == nil {
			t.Errorf("accepted %s", file)
		}
	}
	// Verification must use existing trust, never a key advertised in the update.
	var env envelope.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	env.Signatures[0].KeyID = "untrusted-test-key"
	bad, _ := json.Marshal(env)
	if _, _, err := envelope.Verify(bad, trust, envelope.PurposeRelease, now); err == nil {
		t.Fatal("update supplied its own trust")
	}
}
