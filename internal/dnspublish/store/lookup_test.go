package store

import (
	"bytes"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

type documentRow struct {
	id, revision     uuid.UUID
	generation, hash string
	canonical        []byte
}

func (r documentRow) Scan(values ...any) error {
	*values[0].(*uuid.UUID) = r.id
	*values[1].(*uuid.UUID) = r.revision
	*values[2].(*string) = r.generation
	*values[3].(*string) = r.hash
	*values[4].(*[]byte) = r.canonical
	return nil
}

func TestD056_DeliveryReadRejectsCorruptionAndPreservesBytes(t *testing.T) {
	compiled, err := dnspublish.Compile(dnspublish.Input{ProfileCode: "standard", BlockResponse: "nxdomain", BlockedTTLSeconds: 60, Upstreams: []string{"192.0.2.53:53"}, Rules: []dnspublish.Rule{{RuleID: uuid.NewString(), Domain: "a.example.com", Action: "block", Reason: "TEST ONLY <&>"}}})
	if err != nil {
		t.Fatal(err)
	}
	row := documentRow{uuid.New(), uuid.New(), "18446744073709551615", compiled.Hash, compiled.Bytes}
	got, err := readDocument(row)
	if err != nil || !bytes.Equal(got.Canonical, compiled.Bytes) || got.SizeBytes != len(compiled.Bytes) || got.Generation.String() != row.generation || !bytes.Contains(got.Canonical, []byte("<&>")) {
		t.Fatalf("exact read: %v", err)
	}
	for name, mutate := range map[string]func(*documentRow){
		"hash":                func(r *documentRow) { r.hash = "invalid" },
		"newline":             func(r *documentRow) { r.canonical = append(append([]byte{}, r.canonical...), '\n') },
		"html_escaping":       func(r *documentRow) { r.canonical = bytes.ReplaceAll(r.canonical, []byte("<"), []byte(`\u003c`)) },
		"empty":               func(r *documentRow) { r.canonical = nil },
		"oversize":            func(r *documentRow) { r.canonical = make([]byte, canonjson.MaxBytes+1) },
		"generation_overflow": func(r *documentRow) { r.generation = "18446744073709551616" },
		"generation_zero":     func(r *documentRow) { r.generation = "0" },
		"id":                  func(r *documentRow) { r.id = uuid.Nil },
		"revision":            func(r *documentRow) { r.revision = uuid.Nil },
		"privacy": func(r *documentRow) {
			doc := compiled.Document
			doc.Privacy.QueryLogEnabled = true
			r.hash, r.canonical, _ = dnspublish.Hash(doc)
		},
	} {
		t.Run(name, func(t *testing.T) {
			bad := row
			mutate(&bad)
			if _, err := readDocument(bad); err == nil {
				t.Fatal("corrupt stored document delivered")
			}
		})
	}
}
