package release

import (
	"bytes"
	"errors"
	"io/fs"
	"sync"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var trustUpdateSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	const id = "urn:fvpn:contracts:schema:release-trust-update-envelope:v1"
	raw, err := fs.ReadFile(contracts.Schemas(), "trust/release-update-envelope.schema.json")
	if err != nil {
		return nil, err
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AssertFormat()
	if err := c.AddResource(id, doc); err != nil {
		return nil, err
	}
	return c.Compile(id)
})

// ValidateTrustUpdateEnvelope uses the embedded trust-update schema, never the
// manifest schema. Canonicalization first rejects duplicate members, trailing
// data, invalid UTF-8 and oversized input (§13.3, §16.2).
func ValidateTrustUpdateEnvelope(raw []byte) error {
	if _, err := canonjson.Canonicalize(raw); err != nil {
		return errors.New("release: invalid trust update JSON")
	}
	s, err := trustUpdateSchema()
	if err != nil {
		return err
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return errors.New("release: invalid trust update JSON")
	}
	if err := s.Validate(doc); err != nil {
		// Do not echo untrusted fields (which might contain private material).
		return errors.New("release: trust update schema validation failed")
	}
	return nil
}
