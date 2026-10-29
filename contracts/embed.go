// Package contracts embeds the Freedom VPN API contract so that services and
// tests can validate against the exact committed documents.
//
//   - openapi.yaml: OpenAPI 3.0.3, the single source of truth for HTTP bodies
//     (Project Spec §14).
//
//   - schemas/*.schema.json: JSON Schema 2020-12 for signed or hashed
//     documents (release manifest, desired state, serving lease, usage
//     checkpoint envelopes and the usage event), generated from openapi.yaml.
//
//   - schemas/policies: byte-for-byte owner payload schema snapshots.
//
//   - schemas/trust: standalone release trust rotation schema (§13.4).
//
// See docs/contracts/README.md for conventions.
package contracts

import (
	"embed"
	"io/fs"
)

//go:embed openapi.yaml schemas
var files embed.FS

// OpenAPI returns the OpenAPI document.
func OpenAPI() []byte {
	b, err := files.ReadFile("openapi.yaml")
	if err != nil {
		panic(err) // embedded at build time
	}
	return b
}

// Schemas returns the JSON Schema files, rooted at the schemas directory.
func Schemas() fs.FS {
	sub, err := fs.Sub(files, "schemas")
	if err != nil {
		panic(err)
	}
	return sub
}

// SchemaID returns the $id of a generated JSON Schema file name.
func SchemaID(file string) string {
	const suffix = ".schema.json"
	if len(file) > len(suffix) && file[len(file)-len(suffix):] == suffix {
		file = file[:len(file)-len(suffix)]
	}
	return "urn:fvpn:contracts:schema:" + file + ":v1"
}
