package nodes

import (
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func TestD056_ResolverDocumentExactCanonicalMetadata(t *testing.T) {
	raw := []byte(`{"rules":["<&>"],"schema_version":1}`)
	hash, err := canonjson.HashHex(raw)
	if err != nil {
		t.Fatal(err)
	}
	document := ResolverDocument{Canonical: raw, Reference: api.ResolverDocumentRef{DocumentId: uuid.New(), ProfileRevisionId: uuid.New(), Generation: 3, SizeBytes: len(raw), ContentSha256: hash}}
	if err = ValidateResolverDocument(document); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*ResolverDocument)
	}{
		{"empty", func(d *ResolverDocument) { d.Canonical = nil }},
		{"oversized", func(d *ResolverDocument) {
			d.Canonical = []byte(`"` + strings.Repeat("x", canonjson.MaxBytes) + `"`)
			d.Reference.SizeBytes = len(d.Canonical)
		}},
		{"different bytes", func(d *ResolverDocument) { d.Canonical = []byte(`{"rules":["xxx"],"schema_version":1}`) }},
		{"noncanonical newline", func(d *ResolverDocument) {
			d.Canonical = append([]byte{}, append(d.Canonical, '\n')...)
			d.Reference.SizeBytes++
		}},
		{"wrong length", func(d *ResolverDocument) { d.Reference.SizeBytes++ }},
		{"unknown generation", func(d *ResolverDocument) { d.Reference.Generation = 0 }},
		{"unknown document", func(d *ResolverDocument) { d.Reference.DocumentId = uuid.Nil }},
		{"unknown revision", func(d *ResolverDocument) { d.Reference.ProfileRevisionId = uuid.Nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := document
			tc.change(&d)
			if ValidateResolverDocument(d) == nil {
				t.Fatal("invalid document accepted")
			}
		})
	}
}
