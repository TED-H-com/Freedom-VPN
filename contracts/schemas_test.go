package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/TED-H-com/Freedom-VPN/contracts/internal/gen"
)

// normalizer reduces an OpenAPI 3.0 or JSON Schema 2020-12 schema to a
// dialect-neutral structure so the two can be compared: references are
// resolved, allOf is merged, nullability becomes a "null" type, enums lose
// their null member and annotations (description, title, examples) are
// dropped. It is written independently of contracts/internal/gen, so it
// checks the generator rather than repeating it.
type normalizer struct {
	resolve func(ref string) (map[string]any, error)
	depth   int
}

var structuralKeys = []string{"pattern", "format", "minLength", "maxLength", "minimum", "maximum",
	"minItems", "maxItems", "uniqueItems", "minProperties", "maxProperties", "const", "default"}

func (n *normalizer) norm(s map[string]any) (map[string]any, error) {
	n.depth++
	defer func() { n.depth-- }()
	if n.depth > 64 {
		return nil, fmt.Errorf("schema nesting too deep (cycle?)")
	}
	if ref, ok := s["$ref"].(string); ok {
		target, err := n.resolve(ref)
		if err != nil {
			return nil, err
		}
		return n.norm(target)
	}
	if anyOf, ok := s["anyOf"].([]any); ok {
		var nonNull []any
		nullable := false
		for _, b := range anyOf {
			bm := b.(map[string]any)
			if t, ok := bm["type"].(string); ok && t == "null" && len(bm) == 1 {
				nullable = true
				continue
			}
			nonNull = append(nonNull, b)
		}
		if nullable && len(nonNull) == 1 {
			out, err := n.norm(nonNull[0].(map[string]any))
			if err != nil {
				return nil, err
			}
			out["types"] = addType(out["types"], "null")
			return out, nil
		}
		return nil, fmt.Errorf("unsupported anyOf")
	}
	out := map[string]any{}
	types := []string{}
	switch t := s["type"].(type) {
	case string:
		types = append(types, t)
	case []any:
		for _, x := range t {
			types = append(types, x.(string))
		}
	}
	if nb, _ := s["nullable"].(bool); nb {
		types = append(types, "null")
	}
	if len(types) > 0 {
		sort.Strings(types)
		out["types"] = types
	}
	for _, k := range structuralKeys {
		if v, ok := s[k]; ok {
			out[k] = numberize(v)
		}
	}
	if req, ok := s["required"].([]any); ok {
		out["required"] = sortedStrings(req)
	}
	if enum, ok := s["enum"].([]any); ok {
		var vals []string
		for _, e := range enum {
			if e != nil {
				vals = append(vals, fmt.Sprint(e))
			}
		}
		sort.Strings(vals)
		out["enum"] = vals
	}
	if props, ok := s["properties"].(map[string]any); ok {
		np := map[string]any{}
		for k, v := range props {
			c, err := n.norm(v.(map[string]any))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			np[k] = c
		}
		out["properties"] = np
	}
	if items, ok := s["items"].(map[string]any); ok {
		c, err := n.norm(items)
		if err != nil {
			return nil, fmt.Errorf("items: %w", err)
		}
		out["items"] = c
	}
	switch ap := s["additionalProperties"].(type) {
	case bool:
		out["additionalProperties"] = ap
	case map[string]any:
		c, err := n.norm(ap)
		if err != nil {
			return nil, err
		}
		out["additionalProperties"] = c
	}
	if oneOf, ok := s["oneOf"].([]any); ok {
		var list []any
		for _, b := range oneOf {
			c, err := n.norm(b.(map[string]any))
			if err != nil {
				return nil, err
			}
			list = append(list, c)
		}
		out["oneOf"] = list
	}
	if allOf, ok := s["allOf"].([]any); ok {
		for _, b := range allOf {
			c, err := n.norm(b.(map[string]any))
			if err != nil {
				return nil, err
			}
			merge(out, c)
		}
	}
	return out, nil
}

func merge(dst, src map[string]any) {
	for k, v := range src {
		switch k {
		case "properties":
			dp, _ := dst[k].(map[string]any)
			if dp == nil {
				dp = map[string]any{}
			}
			for pk, pv := range v.(map[string]any) {
				dp[pk] = pv
			}
			dst[k] = dp
		case "required":
			set := map[string]bool{}
			for _, r := range append(toStrings(dst[k]), v.([]string)...) {
				set[r] = true
			}
			var all []string
			for r := range set {
				all = append(all, r)
			}
			sort.Strings(all)
			dst[k] = all
		case "types":
			if dt, ok := dst[k]; ok {
				// A nullable wrapper keeps "null"; the base type comes from both.
				dst[k] = unionTypes(dt.([]string), v.([]string))
			} else {
				dst[k] = v
			}
		default:
			if _, exists := dst[k]; !exists {
				dst[k] = v
			}
		}
	}
}

func unionTypes(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range append(append([]string{}, a...), b...) {
		set[x] = true
	}
	var out []string
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func addType(v any, t string) []string {
	ts, _ := v.([]string)
	return unionTypes(ts, []string{t})
}

func toStrings(v any) []string {
	s, _ := v.([]string)
	return s
}

func sortedStrings(v []any) []string {
	out := make([]string, len(v))
	for i, x := range v {
		out[i] = x.(string)
	}
	sort.Strings(out)
	return out
}

func numberize(v any) any {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	}
	return v
}

// TestJSONSchemas_ConsistentWithOpenAPI asserts that every JSON Schema in
// contracts/schemas is structurally identical to the OpenAPI component of
// the same name (its title).
func TestJSONSchemas_ConsistentWithOpenAPI(t *testing.T) {
	comps, err := gen.LoadComponents(OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir("schemas")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(gen.SchemaTargets) {
		t.Fatalf("schemas/ has %d files, want %d", len(entries), len(gen.SchemaTargets))
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			var js map[string]any
			readJSON(t, filepath.Join("schemas", e.Name()), &js)
			if js["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
				t.Fatalf("$schema = %v", js["$schema"])
			}
			if js["$id"] != SchemaID(e.Name()) {
				t.Fatalf("$id = %v", js["$id"])
			}
			title, _ := js["title"].(string)
			comp, ok := comps[title].(map[string]any)
			if !ok {
				t.Fatalf("no OpenAPI component named %q", title)
			}
			defs, _ := js["$defs"].(map[string]any)
			jn := &normalizer{resolve: func(ref string) (map[string]any, error) {
				name, ok := strings.CutPrefix(ref, "#/$defs/")
				if !ok || defs[name] == nil {
					return nil, fmt.Errorf("bad JSON Schema ref %q", ref)
				}
				return defs[name].(map[string]any), nil
			}}
			on := &normalizer{resolve: func(ref string) (map[string]any, error) {
				name, ok := strings.CutPrefix(ref, "#/components/schemas/")
				if !ok || comps[name] == nil {
					return nil, fmt.Errorf("bad OpenAPI ref %q", ref)
				}
				return comps[name].(map[string]any), nil
			}}
			a, err := on.norm(comp)
			if err != nil {
				t.Fatalf("normalize OpenAPI: %v", err)
			}
			b, err := jn.norm(js)
			if err != nil {
				t.Fatalf("normalize JSON Schema: %v", err)
			}
			if diff := cmp.Diff(a, b); diff != "" {
				t.Fatalf("JSON Schema differs from OpenAPI component %s (-openapi +jsonschema):\n%s", title, diff)
			}
		})
	}
}

// TestGeneratedFilesUpToDate fails when schemas, vectors or generated
// examples differ from `go run ./contracts/internal/cmd/contractsgen`.
func TestGeneratedFilesUpToDate(t *testing.T) {
	files, err := gen.Files(OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.FromSlash(name))
		if err != nil {
			t.Errorf("%s: %v (run make -f mk/contracts.mk contracts-generate)", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s is stale; run make -f mk/contracts.mk contracts-generate", name)
		}
	}
}

// TestEmbeddedFilesMatchDisk guards the go:embed patterns.
func TestEmbeddedFilesMatchDisk(t *testing.T) {
	if !bytes.Equal(OpenAPI(), readFile(t, "openapi.yaml")) {
		t.Fatal("embedded openapi.yaml differs")
	}
	for _, target := range gen.SchemaTargets {
		f, err := Schemas().Open(target.File)
		if err != nil {
			t.Fatalf("embedded %s: %v", target.File, err)
		}
		var v any
		if err := json.NewDecoder(f).Decode(&v); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	if SchemaID("usage-event.schema.json") != "urn:fvpn:contracts:schema:usage-event:v1" || SchemaID("x") != "urn:fvpn:contracts:schema:x:v1" {
		t.Fatal("SchemaID")
	}
}
