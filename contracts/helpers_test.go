package contracts

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	docOnce sync.Once
	docVal  *openapi3.T
	docErr  error
)

// loadDoc parses and validates openapi.yaml once.
func loadDoc(t *testing.T) *openapi3.T {
	t.Helper()
	docOnce.Do(func() {
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = false
		docVal, docErr = loader.LoadFromData(OpenAPI())
		if docErr == nil {
			docErr = docVal.Validate(context.Background(), openapi3.EnableSchemaFormatValidation())
		}
	})
	if docErr != nil {
		t.Fatalf("openapi.yaml: %v", docErr)
	}
	return docVal
}

// visitOpts are the options used to validate JSON values against
// components: all errors, formats asserted, RFC 9562 UUIDs.
func visitOpts() []openapi3.SchemaValidationOption {
	return []openapi3.SchemaValidationOption{
		openapi3.MultiErrors(),
		openapi3.EnableFormatValidation(),
		openapi3.WithStringFormatValidator("uuid", openapi3.NewRegexpFormatValidator(openapi3.FormatOfStringForUUIDOfRFC9562)),
	}
}

// validateComponent validates raw JSON against an OpenAPI component.
func validateComponent(t *testing.T, component string, raw []byte) error {
	t.Helper()
	doc := loadDoc(t)
	ref, ok := doc.Components.Schemas[component]
	if !ok {
		t.Fatalf("no component %q", component)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("%s: invalid JSON: %v", component, err)
	}
	return ref.Value.VisitJSON(v, visitOpts()...)
}

var (
	jsOnce    sync.Once
	jsSchemas map[string]*jsonschema.Schema
	jsErr     error
)

// jsonSchemas compiles every committed JSON Schema with format assertion.
func jsonSchemas(t *testing.T) map[string]*jsonschema.Schema {
	t.Helper()
	jsOnce.Do(func() {
		jsSchemas = map[string]*jsonschema.Schema{}
		c := jsonschema.NewCompiler()
		c.DefaultDraft(jsonschema.Draft2020)
		c.AssertFormat()
		ids := map[string]string{}
		jsErr = fs.WalkDir(Schemas(), ".", func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() {
				return nil
			}
			f, err := Schemas().Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			doc, err := jsonschema.UnmarshalJSON(f)
			if err != nil {
				return err
			}
			id := doc.(map[string]any)["$id"].(string)
			ids[path] = id
			return c.AddResource(id, doc)
		})
		if jsErr != nil {
			return
		}
		for file, id := range ids {
			s, err := c.Compile(id)
			if err != nil {
				jsErr = err
				return
			}
			jsSchemas[file] = s
		}
	})
	if jsErr != nil {
		t.Fatalf("json schemas: %v", jsErr)
	}
	return jsSchemas
}

func validateJSONSchema(t *testing.T, file string, raw []byte) error {
	t.Helper()
	s, ok := jsonSchemas(t)[file]
	if !ok {
		t.Fatalf("no JSON schema %s", file)
	}
	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("%s: invalid JSON: %v", file, err)
	}
	return s.Validate(inst)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := json.Unmarshal(readFile(t, path), v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
