// Package releases implements immutable, signed release registration (§13.3–4).
package releases

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator holds immutable compiled contracts and a separately provisioned trust root.
type Validator struct {
	trust        *envelope.TrustStore
	currentTrust func() (*envelope.TrustStore, error)
	schema       *jsonschema.Schema
	api          *openapi3.T
	origins      map[string]bool
}

func NewValidator(trust *envelope.TrustStore, origins []string) (*Validator, error) {
	if trust == nil || len(origins) == 0 {
		return nil, errors.New("release trust and storage origins are required")
	}
	v := &Validator{trust: trust, origins: map[string]bool{}}
	for _, s := range origins {
		u, e := url.Parse(s)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, errors.New("storage origin must be an HTTPS origin")
		}
		v.origins[strings.ToLower(u.Host)] = true
	}
	raw, e := fs.ReadFile(contracts.Schemas(), "release-manifest-envelope.schema.json")
	if e != nil {
		return nil, e
	}
	doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if e != nil {
		return nil, e
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AssertFormat()
	id := contracts.SchemaID("release-manifest-envelope.schema.json")
	if e = c.AddResource(id, doc); e != nil {
		return nil, e
	}
	v.schema, e = c.Compile(id)
	if e != nil {
		return nil, e
	}
	v.api, e = openapi3.NewLoader().LoadFromData(contracts.OpenAPI())
	return v, e
}

// ValidateBody also rejects duplicate keys before the OpenAPI validator sees them.
func (v *Validator) ValidateBody(component string, raw []byte) error {
	if _, e := canonjson.Canonicalize(raw); e != nil {
		return apierr.New(apierr.ValidationFailed, "Invalid JSON document")
	}
	var value any
	if e := json.Unmarshal(raw, &value); e != nil {
		return apierr.New(apierr.ValidationFailed, "Invalid JSON document")
	}
	s, ok := v.api.Components.Schemas[component]
	if !ok {
		return errors.New("unknown contract component")
	}
	if e := s.Value.VisitJSON(value, openapi3.EnableFormatValidation(), openapi3.WithStringFormatValidator("uuid", openapi3.NewRegexpFormatValidator(openapi3.FormatOfStringForUUIDOfRFC9562))); e != nil {
		return apierr.New(apierr.ValidationFailed, "Document violates the API contract")
	}
	return nil
}

func (v *Validator) AllowURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == "" && v.origins[strings.ToLower(u.Host)]
}

// ValidateManifest checks signatures before interpreting the signed payload. The
// hash binds the whole envelope, matching contracts/vectors (§13.10.2).
func (v *Validator) ValidateManifest(raw []byte, now time.Time) (api.ReleaseManifestEnvelope, string, error) {
	var m api.ReleaseManifestEnvelope
	trust := v.trust
	if v.currentTrust != nil {
		var e error
		trust, e = v.currentTrust()
		if e != nil {
			return m, "", apierr.New(apierr.DependencyUnavailable, "Release trust unavailable")
		}
	}
	_, _, e := envelope.Verify(raw, trust, envelope.PurposeRelease, now)
	if e != nil {
		code := apierr.SignatureInvalid
		if errors.Is(e, envelope.ErrKeyUntrusted) {
			code = apierr.KeyUntrusted
		}
		return m, "", apierr.New(code, "Release signature rejected")
	}
	instance, e := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if e != nil {
		return m, "", e
	}
	if e = v.schema.Validate(instance); e != nil {
		return m, "", apierr.New(apierr.ValidationFailed, "Manifest violates the release schema")
	}
	if e = json.Unmarshal(raw, &m); e != nil {
		return m, "", apierr.New(apierr.ValidationFailed, "Invalid manifest integer")
	}
	found := map[api.Architecture]bool{}
	platforms := map[string]bool{}
	for _, a := range m.Signed.Artifacts {
		if !v.AllowURL(a.Url) {
			return m, "", apierr.New(apierr.ValidationFailed, "Artifact storage origin is not allowed")
		}
		for _, os := range a.OsVersions {
			k := string(a.OsId) + "/" + os + "/" + string(a.Architecture)
			if platforms[k] {
				return m, "", apierr.New(apierr.UnsupportedPlatform, "Duplicate platform artifact")
			}
			platforms[k] = true
		}
		if a.OsId == "ubuntu" && slices.Contains(a.OsVersions, "24.04") {
			found[a.Architecture] = true
		}
	}
	if !found["amd64"] || !found["arm64"] {
		return m, "", apierr.New(apierr.UnsupportedPlatform, "Ubuntu 24.04 artifacts for both architectures are required")
	}
	for _, u := range []string{m.Signed.Evidence.SbomUrl, m.Signed.Evidence.NoticesUrl, m.Signed.Evidence.SourceUrl} {
		if !v.AllowURL(u) {
			return m, "", apierr.New(apierr.ValidationFailed, "Evidence storage origin is not allowed")
		}
	}
	if m.Signed.Compatibility.ControlApiMajor != 1 {
		return m, "", apierr.New(apierr.ControlAPIIncompatible, "Control API major is unsupported")
	}
	if !slices.Contains(m.Signed.Compatibility.PolicySchemaVersions, 1) || !slices.Contains(m.Signed.Compatibility.UsageEventSchemaVersions, 1) {
		return m, "", apierr.New(apierr.SchemaIncompatible, "Required policy or usage schema is unsupported")
	}
	hash, e := canonjson.HashHex(raw)
	return m, hash, e
}

// WithCurrentTrust returns an immutable validator using the current local trust
// authority on every authorization. The callback must not call external services.
func (v *Validator) WithCurrentTrust(source func() (*envelope.TrustStore, error)) *Validator {
	copy := *v
	copy.currentTrust = source
	return &copy
}
