package contracts

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestUPD05_Revision7VersionAndUpdateTargetEvidence(t *testing.T) {
	doc := loadDoc(t)
	// Revision 9 advances info.version to 1.5.0. This test still pins revision 7's decisions.
	if doc.Info.Version != "1.5.0" {
		t.Fatal("API version must be 1.5.0")
	}
	schema := doc.Components.Schemas["UpdateTarget"].Value
	for _, name := range []string{"previous_release_id", "health", "candidate_health"} {
		prop := schema.Properties[name].Value
		if prop == nil || !prop.Nullable || contains(schema.Required, name) {
			t.Fatalf("%s must be optional and nullable", name)
		}
		if !strings.Contains(prop.Description, "latest current") || !strings.Contains(prop.Description, "superseded generation") {
			t.Fatalf("%s must name the current result and exclude history results", name)
		}
	}
	if schema.Properties["previous_release_id"].Value.Format != "uuid" {
		t.Fatal("previous_release_id must be a uuid")
	}
	for _, name := range []string{"health", "candidate_health"} {
		if schema.Properties[name].Value.AllOf[0].Ref != "#/components/schemas/UpdateHealth" {
			t.Fatalf("%s must be UpdateHealth", name)
		}
	}
	result := doc.Components.Schemas["UpdateResult"].Value
	if contains(result.Required, "candidate_health") || result.Properties["health"] == nil {
		t.Fatal("UpdateResult must be unchanged")
	}
	if doc.Components.Schemas["UpdateHealth"].Value.Properties["tunnel_probe"] == nil {
		t.Fatal("UpdateHealth must be unchanged")
	}
}

func TestUPD05_Revision7UpdateTargetExamplesRoundTrip(t *testing.T) {
	raw := readFile(t, "examples/valid/UpdateTarget.rolled-back.json")
	if err := validateComponent(t, "UpdateTarget", raw); err != nil {
		t.Fatal(err)
	}
	var rolled api.UpdateTarget
	if err := json.Unmarshal(raw, &rolled); err != nil {
		t.Fatal(err)
	}
	if rolled.State != api.UpdateJobStateRolledBack {
		t.Fatal("rolled-back example must use state rolled_back")
	}
	previous, err := rolled.PreviousReleaseId.Get()
	if err != nil || previous.String() != "6d2797d0-b9a5-4724-96af-9b3039d351b1" {
		t.Fatal("previous_release_id must be the candidate")
	}
	current, err := rolled.CurrentReleaseId.Get()
	if err != nil || current.String() == previous.String() || current.String() != "cdb25ffa-5395-400c-ad3e-b32899f14bb8" {
		t.Fatal("current_release_id must be the restored release")
	}
	health, err := rolled.Health.Get()
	if err != nil || !healthPassed(health) {
		t.Fatal("health must be the restored release's checks")
	}
	candidate, err := rolled.CandidateHealth.Get()
	if err != nil || candidate.TunnelProbe != api.HealthCheckStatusFailed || candidate.Processes != api.HealthCheckStatusPassed {
		t.Fatal("candidate_health must keep the candidate failure")
	}
	assertTargetRoundTrip(t, raw)

	nullRaw := readFile(t, "examples/valid/UpdateTarget.null-evidence.json")
	if err := validateComponent(t, "UpdateTarget", nullRaw); err != nil {
		t.Fatal(err)
	}
	var empty api.UpdateTarget
	if err := json.Unmarshal(nullRaw, &empty); err != nil {
		t.Fatal(err)
	}
	if !empty.PreviousReleaseId.IsNull() || !empty.Health.IsNull() || !empty.CandidateHealth.IsNull() {
		t.Fatal("explicit null evidence was lost")
	}
	assertTargetRoundTrip(t, nullRaw)

	batch := readFile(t, "examples/valid/UpdateBatch.rolled-back.json")
	if err := validateComponent(t, "UpdateBatch", batch); err != nil {
		t.Fatal(err)
	}
	var decoded api.UpdateBatch
	if err := json.Unmarshal(batch, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Targets) != 1 || decoded.Targets[0].State != api.UpdateJobStateRolledBack {
		t.Fatal("batch detail must carry the rolled-back target")
	}
	assertTargetRoundTrip(t, mustMarshal(t, decoded.Targets[0]))

	// candidate_health on a non-rolled-back state is still schema-valid.
	// The rule is semantic: the fleet-update projector (follow-up 5) copies
	// candidate_health only from a rolled_back current result. This schema
	// does not enforce it.
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	body["state"] = "succeeded"
	succeeded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateComponent(t, "UpdateTarget", succeeded); err != nil {
		t.Fatal(err)
	}
	var other api.UpdateTarget
	if err := json.Unmarshal(succeeded, &other); err != nil {
		t.Fatal(err)
	}
	if other.State == api.UpdateJobStateRolledBack || !other.CandidateHealth.IsSpecified() || other.CandidateHealth.IsNull() {
		t.Fatal("non-rolled-back candidate_health must remain present for the schema")
	}
}

func assertTargetRoundTrip(t *testing.T, raw []byte) {
	t.Helper()
	var model api.UpdateTarget
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := canonjson.Canonicalize(encoded)
	if err != nil || !bytes.Equal(want, got) {
		t.Fatalf("round trip changed JSON\n got %s\nwant %s", got, want)
	}
}
