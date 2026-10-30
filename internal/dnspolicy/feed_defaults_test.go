package dnspolicy

import (
	"encoding/json"
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
