// Package gen derives the committed JSON Schemas, test vectors and signed
// examples from contracts/openapi.yaml and the TEST ONLY seeds. It is run by
// `make -f mk/contracts.mk contracts-generate` through
// contracts/internal/cmd/contractsgen, and the contracts tests fail when the
// committed files differ from its output.
package gen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaTarget maps a JSON Schema file to the OpenAPI component it is
// derived from.
type SchemaTarget struct {
	File      string // file name under contracts/schemas
	Component string // OpenAPI component (and JSON Schema title)
	Section   string // spec section
}

// SchemaTargets lists the JSON Schema 2020-12 documents for signed or hashed
// documents that are also used outside HTTP bodies (AGENTS.md §3).
var SchemaTargets = []SchemaTarget{
	{"release-manifest-envelope.schema.json", "ReleaseManifestEnvelope", "§13.3"},
	{"desired-state-envelope.schema.json", "DesiredStateEnvelope", "§14.7"},
	{"serving-lease-envelope.schema.json", "ServingLeaseEnvelope", "§16.3"},
	{"usage-checkpoint-envelope.schema.json", "UsageCheckpointEnvelope", "§17.3"},
	{"usage-event.schema.json", "UsageEvent", "§10.5"},
}

const componentPrefix = "#/components/schemas/"

// LoadComponents parses the OpenAPI YAML and returns components.schemas.
func LoadComponents(openapiYAML []byte) (map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(openapiYAML, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi: %w", err)
	}
	comps, _ := doc["components"].(map[string]any)
	schemas, _ := comps["schemas"].(map[string]any)
	if schemas == nil {
		return nil, fmt.Errorf("openapi: no components.schemas")
	}
	return schemas, nil
}

// JSONSchemas returns the generated JSON Schema files keyed by file name.
func JSONSchemas(openapiYAML []byte) (map[string][]byte, error) {
	comps, err := LoadComponents(openapiYAML)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, t := range SchemaTargets {
		b, err := jsonSchemaFor(comps, t)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.File, err)
		}
		out[t.File] = b
	}
	return out, nil
}

func jsonSchemaFor(comps map[string]any, t SchemaTarget) ([]byte, error) {
	root, ok := comps[t.Component].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("component %s not found", t.Component)
	}
	deps := map[string]bool{}
	if err := collectRefs(comps, root, deps); err != nil {
		return nil, err
	}
	conv, err := convert(root)
	if err != nil {
		return nil, err
	}
	doc := conv.(map[string]any)
	doc["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	doc["$id"] = "urn:fvpn:contracts:schema:" + strings.TrimSuffix(t.File, ".schema.json") + ":v1"
	doc["title"] = t.Component
	doc["$comment"] = fmt.Sprintf("Generated from contracts/openapi.yaml component %s (%s) by contracts/internal/cmd/contractsgen. DO NOT EDIT.", t.Component, t.Section)
	if len(deps) > 0 {
		defs := map[string]any{}
		for name := range deps {
			c, err := convert(comps[name])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			cm := c.(map[string]any)
			cm["title"] = name
			defs[name] = cm
		}
		doc["$defs"] = defs
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func collectRefs(comps map[string]any, n any, seen map[string]bool) error {
	switch v := n.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			name, ok := strings.CutPrefix(ref, componentPrefix)
			if !ok {
				return fmt.Errorf("unsupported $ref %q", ref)
			}
			if seen[name] {
				return nil
			}
			c, ok := comps[name]
			if !ok {
				return fmt.Errorf("unresolved $ref %q", ref)
			}
			seen[name] = true
			return collectRefs(comps, c, seen)
		}
		keys := sortedKeys(v)
		for _, k := range keys {
			if strings.HasPrefix(k, "x-") || k == "discriminator" {
				continue
			}
			if err := collectRefs(comps, v[k], seen); err != nil {
				return err
			}
		}
	case []any:
		for _, e := range v {
			if err := collectRefs(comps, e, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// convert maps an OpenAPI 3.0 schema object to JSON Schema 2020-12:
// nullable becomes a "null" type (or an anyOf with {"type":"null"} for
// composed schemas), $ref targets move to $defs, `example` becomes
// `examples`, and x- extensions and discriminator are dropped.
func convert(n any) (any, error) {
	switch v := n.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			name, ok := strings.CutPrefix(ref, componentPrefix)
			if !ok {
				return nil, fmt.Errorf("unsupported $ref %q", ref)
			}
			return map[string]any{"$ref": "#/$defs/" + name}, nil
		}
		out := map[string]any{}
		for _, k := range sortedKeys(v) {
			if strings.HasPrefix(k, "x-") {
				continue
			}
			switch k {
			case "nullable", "discriminator", "example", "externalDocs", "xml":
				continue
			case "properties", "patternProperties", "$defs":
				props := map[string]any{}
				pm, _ := v[k].(map[string]any)
				for _, pk := range sortedKeys(pm) {
					c, err := convert(pm[pk])
					if err != nil {
						return nil, fmt.Errorf("%s: %w", pk, err)
					}
					props[pk] = c
				}
				out[k] = props
			default:
				c, err := convert(v[k])
				if err != nil {
					return nil, fmt.Errorf("%s: %w", k, err)
				}
				out[k] = c
			}
		}
		if ex, ok := v["example"]; ok {
			out["examples"] = []any{ex}
		}
		if nullable, _ := v["nullable"].(bool); nullable {
			_, hasAll := v["allOf"]
			_, hasOne := v["oneOf"]
			_, hasAny := v["anyOf"]
			if hasAll || hasOne || hasAny {
				outer := map[string]any{}
				if d, ok := out["description"]; ok {
					outer["description"] = d
					delete(out, "description")
				}
				outer["anyOf"] = []any{out, map[string]any{"type": "null"}}
				return outer, nil
			}
			t, ok := out["type"].(string)
			if !ok {
				return nil, fmt.Errorf("nullable schema without a single type")
			}
			out["type"] = []any{t, "null"}
			if enum, ok := out["enum"].([]any); ok {
				hasNull := false
				for _, e := range enum {
					if e == nil {
						hasNull = true
					}
				}
				if !hasNull {
					out["enum"] = append(append([]any{}, enum...), nil)
				}
			}
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			c, err := convert(e)
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	default:
		return v, nil
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
