package dnspolicy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestD058_NullAttributionSourceRoundTrip(t *testing.T) {
	source := ListSource{SourceID: "test-feed", Version: "1", License: "TEST ONLY",
		LicenseStatus: LicenseCommercialApproved, PermittedUse: "TEST ONLY commercial approval",
		DistributionNotes: "TEST ONLY server-side distribution", URL: "https://example.com/feed"}
	if err := source.ValidateMetadata(); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil || string(fields["attribution_notes"]) != "null" {
		t.Fatalf("attribution not null: %v", err)
	}
	var decoded ListSource
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.AttributionNotes != nil {
		t.Fatalf("null attribution round trip: %v", err)
	}
	source.AttributionNotes = new("TEST ONLY attribution")
	clone := source.Clone()
	*source.AttributionNotes = "changed"
	if *clone.AttributionNotes != "TEST ONLY attribution" {
		t.Fatal("source clone shares attribution pointer")
	}
	source.AttributionNotes = new("")
	if err := source.ValidateMetadata(); err == nil {
		t.Fatal("explicit blank attribution must remain invalid")
	}
}

func TestB9_AttributionNoteVector(t *testing.T) {
	base := ListSource{SourceID: "test-feed", Version: "1", License: "TEST ONLY",
		LicenseStatus: LicenseCommercialApproved, PermittedUse: "TEST ONLY commercial approval",
		DistributionNotes: "TEST ONLY server-side distribution", URL: "https://example.com/feed"}
	encoded, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	valid, err := json.Marshal("TEST ONLY attribution")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		value  json.RawMessage
		accept bool
	}{
		{name: "null", value: json.RawMessage("null"), accept: true},
		{name: "missing", accept: true},
		{name: "empty", value: json.RawMessage(`""`), accept: false},
		{name: "whitespace", value: json.RawMessage(`"   "`), accept: false},
		{name: "valid", value: valid, accept: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := make(map[string]json.RawMessage, len(object))
			for key, value := range object {
				fields[key] = value
			}
			if tc.name == "missing" {
				delete(fields, "attribution_notes")
			} else {
				fields["attribution_notes"] = tc.value
			}
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			var source ListSource
			if err := json.Unmarshal(raw, &source); err != nil {
				t.Fatal(err)
			}
			err = source.ValidateMetadata()
			if tc.accept {
				if err != nil {
					t.Fatal(err)
				}
				if tc.name != "valid" && source.AttributionNotes != nil {
					t.Fatal("null and omitted notes must decode to nil")
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Code != ErrCodeSourceLicenseUnapproved || ve.Field != "attribution_notes" || ve.Message != "attribution_notes must be null or a non-blank note" {
				t.Fatalf("attribution error = %v", err)
			}
		})
	}
}
