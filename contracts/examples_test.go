package contracts

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/TED-H-com/Freedom-VPN/contracts/internal/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// envelopePurpose maps envelope components to their signing purpose.
var envelopePurpose = map[string]envelope.Purpose{
	"ReleaseManifestEnvelope": envelope.PurposeRelease,
	"DesiredStateEnvelope":    envelope.PurposeDesiredState,
	"ServingLeaseEnvelope":    envelope.PurposeLease,
	"UsageCheckpointEnvelope": envelope.PurposeRecovery,
}

func jsonSchemaFileFor(component string) string {
	for _, t := range gen.SchemaTargets {
		if t.Component == component {
			return t.File
		}
	}
	return ""
}

type example struct {
	file      string
	component string
	raw       []byte
}

func listExamples(t *testing.T, dir string) []example {
	entries, err := os.ReadDir(filepath.Join("examples", dir))
	if err != nil {
		t.Fatal(err)
	}
	var out []example
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || strings.HasPrefix(e.Name(), "reasons") {
			continue
		}
		comp, _, _ := strings.Cut(strings.TrimSuffix(e.Name(), ".json"), ".")
		out = append(out, example{e.Name(), comp, readFile(t, filepath.Join("examples", dir, e.Name()))})
	}
	return out
}

func testTrustStore(t *testing.T) (*envelope.TrustStore, time.Time) {
	t.Helper()
	var f struct {
		Comment   string    `json:"_comment"`
		NotBefore time.Time `json:"not_before"`
		NotAfter  time.Time `json:"not_after"`
		VerifyAt  time.Time `json:"verify_at"`
		Keys      []struct {
			KeyID   string `json:"key_id"`
			Purpose string `json:"purpose"`
			Trusted bool   `json:"trusted"`
			PubB64  string `json:"public_key_b64"`
		} `json:"keys"`
	}
	readJSON(t, "vectors/test-keys.json", &f)
	if !strings.HasPrefix(f.Comment, "TEST ONLY - NOT A SECRET") {
		t.Fatal("test keys must be labeled TEST ONLY - NOT A SECRET")
	}
	var keys []envelope.TrustedKey
	for _, k := range f.Keys {
		if !k.Trusted {
			continue
		}
		pub, err := base64.StdEncoding.DecodeString(k.PubB64)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, envelope.TrustedKey{KeyID: k.KeyID, PublicKey: ed25519.PublicKey(pub),
			Purposes: []envelope.Purpose{envelope.Purpose(k.Purpose)}, NotBefore: f.NotBefore, NotAfter: f.NotAfter})
	}
	ts, err := envelope.NewTrustStore(keys...)
	if err != nil {
		t.Fatal(err)
	}
	return ts, f.VerifyAt
}

// semanticCheck verifies hashes and signatures that a schema cannot express.
func semanticCheck(t *testing.T, ex example) error {
	ts, now := testTrustStore(t)
	if p, ok := envelopePurpose[ex.component]; ok {
		_, _, err := envelope.Verify(ex.raw, ts, p, now)
		return err
	}
	switch ex.component {
	case "ReleaseRegisterRequest":
		var r struct {
			Manifest json.RawMessage `json:"manifest"`
		}
		if err := json.Unmarshal(ex.raw, &r); err != nil {
			return err
		}
		_, _, err := envelope.Verify(r.Manifest, ts, envelope.PurposeRelease, now)
		return err
	case "UsageEvent":
		return checkHash(ex.raw, "payload_sha256")
	case "UpdateResult":
		return checkHash(ex.raw, "payload_hash")
	case "UsageBatchRequest":
		var b struct {
			Events []json.RawMessage `json:"events"`
		}
		if err := json.Unmarshal(ex.raw, &b); err != nil {
			return err
		}
		for i, e := range b.Events {
			if err := checkHash(e, "payload_sha256"); err != nil {
				return fmt.Errorf("event %d: %w", i, err)
			}
		}
	}
	return nil
}

var errHashMismatch = errors.New("hash mismatch")

func checkHash(raw []byte, field string) error {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	want, _ := m[field].(string)
	got, err := canonjson.HashHex(raw, field)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%w: %s is %s, computed %s", errHashMismatch, field, want, got)
	}
	return nil
}

// kinErrors flattens kin-openapi errors into "pointer keyword" strings.
func kinErrors(err error) []string {
	var out []string
	var walk func(error)
	walk = func(e error) {
		var me openapi3.MultiError
		if errors.As(e, &me) {
			for _, x := range me {
				walk(x)
			}
			return
		}
		var se *openapi3.SchemaError
		if errors.As(e, &se) {
			// Normalize to the JSON Schema convention used in reasons:
			// kin-openapi reports a missing member at the member's own
			// pointer and an unknown member under "properties".
			p, kw := se.JSONPointer(), se.SchemaField
			if kw == "nullable" {
				kw = "type" // JSON Schema expresses nullability as a type.
			}
			if kw == "required" && len(p) > 0 {
				p = p[:len(p)-1]
			}
			if kw == "properties" && strings.Contains(se.Reason, "unsupported") {
				kw = "additionalProperties"
			}
			out = append(out, pointer(p)+" "+kw)
			if se.Origin != nil {
				walk(se.Origin)
			}
		}
	}
	walk(err)
	return out
}

// jsErrors flattens jsonschema/v6 errors into "pointer keyword" strings.
func jsErrors(err error) []string {
	var out []string
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		kp := e.ErrorKind.KeywordPath()
		kw := ""
		if len(kp) > 0 {
			kw = kp[len(kp)-1]
		}
		out = append(out, pointer(e.InstanceLocation)+" "+kw)
		for _, c := range e.Causes {
			walk(c)
		}
	}
	var ve *jsonschema.ValidationError
	if errors.As(err, &ve) {
		walk(ve)
	}
	return out
}

func pointer(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return "/" + strings.Join(parts, "/")
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestAPI01_ValidExamples(t *testing.T) {
	for _, ex := range listExamples(t, "valid") {
		t.Run(ex.file, func(t *testing.T) {
			if err := validateComponent(t, ex.component, ex.raw); err != nil {
				t.Fatalf("OpenAPI %s: %v", ex.component, err)
			}
			if f := jsonSchemaFileFor(ex.component); f != "" {
				if err := validateJSONSchema(t, f, ex.raw); err != nil {
					t.Fatalf("JSON Schema %s: %v", f, err)
				}
			}
			if err := semanticCheck(t, ex); err != nil {
				t.Fatalf("semantic check: %v", err)
			}
		})
	}
}

func loadReasons(t *testing.T) map[string]gen.Reason {
	reasons := map[string]gen.Reason{}
	for _, f := range []string{"reasons.json", "reasons.generated.json"} {
		var m map[string]gen.Reason
		readJSON(t, filepath.Join("examples/invalid", f), &m)
		for k, v := range m {
			if _, dup := reasons[k]; dup {
				t.Fatalf("duplicate reason for %s", k)
			}
			reasons[k] = v
		}
	}
	return reasons
}

func TestAPI01_InvalidExamplesFailForTheExpectedReason(t *testing.T) {
	reasons := loadReasons(t)
	seen := map[string]bool{}
	for _, ex := range listExamples(t, "invalid") {
		t.Run(ex.file, func(t *testing.T) {
			seen[ex.file] = true
			r, ok := reasons[ex.file]
			if !ok {
				t.Fatalf("no expected reason in examples/invalid/reasons*.json")
			}
			if r.Note == "" {
				t.Error("reason needs a note")
			}
			jsFile := jsonSchemaFileFor(ex.component)
			switch r.Kind {
			case "schema":
				want := r.Pointer + " " + r.Keyword
				err := validateComponent(t, ex.component, ex.raw)
				if err == nil {
					t.Fatalf("OpenAPI %s accepted the invalid example", ex.component)
				}
				if got := kinErrors(err); !contains(got, want) {
					t.Fatalf("OpenAPI errors %q do not include %q", got, want)
				}
				if jsFile != "" {
					err := validateJSONSchema(t, jsFile, ex.raw)
					if err == nil {
						t.Fatalf("JSON Schema %s accepted the invalid example", jsFile)
					}
					if got := jsErrors(err); !contains(got, want) {
						t.Fatalf("JSON Schema errors %q do not include %q", got, want)
					}
				}
			case "payload_hash", "signature":
				// Schema-valid, semantically invalid.
				if err := validateComponent(t, ex.component, ex.raw); err != nil {
					t.Fatalf("must be schema-valid: %v", err)
				}
				if jsFile != "" {
					if err := validateJSONSchema(t, jsFile, ex.raw); err != nil {
						t.Fatalf("must be JSON-Schema-valid: %v", err)
					}
				}
				err := semanticCheck(t, ex)
				if err == nil {
					t.Fatal("semantic check accepted the invalid example")
				}
				if r.Kind == "payload_hash" && !errors.Is(err, errHashMismatch) {
					t.Fatalf("want hash mismatch, got %v", err)
				}
				if r.Kind == "signature" {
					var ee *envelope.Error
					if !errors.As(err, &ee) || ee.Code != r.Code {
						t.Fatalf("want %s, got %v", r.Code, err)
					}
				}
			default:
				t.Fatalf("unknown reason kind %q", r.Kind)
			}
		})
	}
	for f := range reasons {
		if !seen[f] {
			t.Errorf("reason for missing file %s", f)
		}
	}
}

// TestAPI01_ExampleCoverage requires a valid example for every request and
// 2xx response body, and an invalid example for every request body.
func TestAPI01_ExampleCoverage(t *testing.T) {
	doc := loadDoc(t)
	valid, invalid := map[string]bool{}, map[string]bool{}
	for _, ex := range listExamples(t, "valid") {
		if _, ok := doc.Components.Schemas[ex.component]; !ok {
			t.Errorf("valid/%s: unknown component", ex.file)
		}
		valid[ex.component] = true
	}
	for _, ex := range listExamples(t, "invalid") {
		if _, ok := doc.Components.Schemas[ex.component]; !ok {
			t.Errorf("invalid/%s: unknown component", ex.file)
		}
		invalid[ex.component] = true
	}
	name := func(ref string) string { return strings.TrimPrefix(ref, "#/components/schemas/") }
	var missing []string
	for _, o := range operations(t) {
		if o.op.RequestBody != nil && o.path != "/webhooks/{provider}" {
			n := name(o.op.RequestBody.Value.Content.Get("application/json").Schema.Ref)
			if !valid[n] || !invalid[n] {
				missing = append(missing, fmt.Sprintf("%s request %s (valid=%v invalid=%v)", o.key, n, valid[n], invalid[n]))
			}
		}
		for code, r := range o.op.Responses.Map() {
			mt := r.Value.Content.Get("application/json")
			if mt == nil || (code[0] != '2' && code != "412") {
				continue
			}
			if n := name(mt.Schema.Ref); !valid[n] {
				missing = append(missing, fmt.Sprintf("%s %s response %s", o.key, code, n))
			}
		}
	}
	// Every policy type has a valid create example.
	for _, pt := range []string{"plan", "fair-use", "dns-profile", "domain-rule-set", "threat-feed", "release-support"} {
		if _, err := os.Stat("examples/valid/PolicyRevisionCreate." + pt + ".json"); err != nil {
			missing = append(missing, "PolicyRevisionCreate."+pt)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("missing examples:\n%s", strings.Join(missing, "\n"))
	}
}

// TestJSONSchemas_ComparatorDetectsDrift proves the consistency check is not
// vacuous: a one-field change must be reported.
func TestJSONSchemas_ComparatorDetectsDrift(t *testing.T) {
	comps, err := gen.LoadComponents(OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	var js map[string]any
	readJSON(t, "schemas/usage-event.schema.json", &js)
	defs := js["$defs"].(map[string]any)
	rec := defs["UsageRecord"].(map[string]any)["properties"].(map[string]any)
	rec["upload_bytes"].(map[string]any)["maxLength"] = 21.0
	jn := &normalizer{resolve: func(ref string) (map[string]any, error) {
		return defs[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any), nil
	}}
	on := &normalizer{resolve: func(ref string) (map[string]any, error) {
		return comps[strings.TrimPrefix(ref, "#/components/schemas/")].(map[string]any), nil
	}}
	a, _ := on.norm(comps["UsageEvent"].(map[string]any))
	b, _ := jn.norm(js)
	if fmt.Sprint(a) == fmt.Sprint(b) {
		t.Fatal("comparator missed a maxLength change")
	}
}
