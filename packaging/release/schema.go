package release

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"sync"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const manifestSchemaID = "urn:fvpn:contracts:schema:release-manifest-envelope:v1"

// dummySignature is 64 zero bytes, a syntactically valid signature used only
// to validate the signed object against the envelope schema. It is not a
// trusted signature.
var dummySignature = base64.StdEncoding.EncodeToString(make([]byte, 64))

var (
	schemaOnce sync.Once
	schemaVal  *jsonschema.Schema
	schemaErr  error
)

func manifestSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		raw, err := fs.ReadFile(contracts.Schemas(), "release-manifest-envelope.schema.json")
		if err != nil {
			schemaErr = err
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		c.DefaultDraft(jsonschema.Draft2020)
		c.AssertFormat()
		if err := c.AddResource(manifestSchemaID, doc); err != nil {
			schemaErr = err
			return
		}
		schemaVal, schemaErr = c.Compile(manifestSchemaID)
	})
	return schemaVal, schemaErr
}

// ValidateEnvelope checks a release manifest envelope against the committed
// JSON Schema (§13.3). Schema validity is not signature validity.
func ValidateEnvelope(raw []byte) error {
	s, err := manifestSchema()
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("release: manifest JSON: %w", err)
	}
	if err := s.Validate(inst); err != nil {
		return fmt.Errorf("release: manifest schema: %w", err)
	}
	return nil
}

// ValidateSigned checks the signed object by wrapping it in a syntactically
// valid envelope. The wrapper signature is not verified here.
func ValidateSigned(signed []byte) error {
	var obj any
	dec := json.NewDecoder(bytes.NewReader(signed))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return fmt.Errorf("release: signed JSON: %w", err)
	}
	wrap := map[string]any{
		"signed": obj,
		"signatures": []any{map[string]any{
			"key_id":        "schema-check-placeholder",
			"algorithm":     "ed25519",
			"signature_b64": dummySignature,
		}},
	}
	raw, err := json.Marshal(wrap)
	if err != nil {
		return err
	}
	return ValidateEnvelope(raw)
}
