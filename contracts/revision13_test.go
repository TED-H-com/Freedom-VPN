package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestAPI01_D088ObservationAcceptedFreezeSignal(t *testing.T) {
	schema := loadDoc(t).Components.Schemas["ObservationAccepted"].Value
	if !slices.Equal(schema.Required, []string{"report_id", "accepted", "received_at"}) {
		t.Fatal("the existing required observation ACK members must remain unchanged")
	}
	for _, value := range []string{"absent", "true", "false", "null", `"true"`, "1"} {
		t.Run(value, func(t *testing.T) {
			var body map[string]json.RawMessage
			readJSON(t, "examples/valid/ObservationAccepted.json", &body)
			if value != "absent" {
				body["identity_frozen"] = json.RawMessage(value)
			}
			raw := mustMarshal(t, body)
			valid := value == "absent" || value == "true" || value == "false"
			if err := validateComponent(t, "ObservationAccepted", raw); (err == nil) != valid {
				t.Fatalf("valid=%v: %v", valid, err)
			}
			if valid {
				var ack api.ObservationAccepted
				assertRevision13RoundTrip(t, raw, &ack)
				if (ack.IdentityFrozen == nil) != (value == "absent") {
					t.Fatal("freeze-signal presence changed")
				}
			}
		})
	}
}

func TestUPD05_D089InstallerSourceAndPayloadHash(t *testing.T) {
	doc := loadDoc(t)
	if doc.Info.Version != "1.9.0" || len(operations(t)) != 49 {
		t.Fatal("revision 13 must be API 1.9.0 with 49 operations")
	}
	post := doc.Paths.Value("/agent/update-results").Post
	if post.RequestBody.Value.Content["application/json"].Schema.Ref != "#/components/schemas/UpdateResult" || post.Responses.Value("404") == nil {
		t.Fatal("submitUpdateResult must use UpdateResult and document unknown-job 404")
	}
	for _, name := range []string{"UpdateResult", "UpdateJobSummary", "UpdateTarget", "NodeUpdateSummary"} {
		schema := doc.Components.Schemas[name].Value
		source := schema.Properties["source"].Value
		if contains(schema.Required, "source") || source.Nullable || source.Default != "central" || !slices.Equal(source.Enum, []any{"central", "installer"}) {
			t.Fatalf("%s source must be optional, non-nullable, central/installer, default central", name)
		}
		for _, value := range []any{"central", "installer", "unknown", nil, true} {
			valid := value == "central" || value == "installer"
			if err := source.VisitJSON(value, visitOpts()...); (err == nil) != valid {
				t.Fatalf("%s source %v: valid=%v: %v", name, value, valid, err)
			}
		}
	}
	for _, file := range []string{"UpdateResult.json", "UpdateResult.installer.json"} {
		raw := readFile(t, "examples/valid/"+file)
		if err := validateComponent(t, "UpdateResult", raw); err != nil {
			t.Fatal(err)
		}
		var result api.UpdateResult
		encoded := assertRevision13RoundTrip(t, raw, &result)
		if err := checkHash(encoded, "payload_hash"); err != nil {
			t.Fatal(err)
		}
		if file == "UpdateResult.json" {
			if result.Source != nil {
				t.Fatal("default source must not be inserted into legacy hashed payloads")
			}
			continue
		}
		if result.Source == nil || *result.Source != api.UpdateResultSourceInstaller || result.Attempt != 1 || result.JobId.Version() != 5 || !result.BatchId.IsNull() || !result.TargetId.IsNull() {
			t.Fatal("installer fixture must carry source installer, UUIDv5, attempt 1 and no batch/target")
		}
		tampered := bytes.Replace(raw, []byte(`"installer"`), []byte(`"central"`), 1)
		if err := validateComponent(t, "UpdateResult", tampered); err != nil {
			t.Fatal(err)
		}
		if err := checkHash(tampered, "payload_hash"); !errors.Is(err, errHashMismatch) {
			t.Fatalf("changing source without rehashing must fail: %v", err)
		}
	}
}

func TestISO01_D085ForeignTCFilterObservation(t *testing.T) {
	spec, ok := apierr.Lookup(apierr.TCForeignFilter)
	health, _ := apierr.Lookup(apierr.HealthCheckFailed)
	if !ok || spec.HTTPStatus != health.HTTPStatus || spec.Retryable != health.Retryable || !spec.NodeLocal || !api.ErrorCodeTCFOREIGNFILTER.Valid() {
		t.Fatal("TC_FOREIGN_FILTER must retain HEALTH_CHECK_FAILED's HTTP mapping and node-local classification")
	}
	var body map[string]any
	readJSON(t, "examples/valid/ObservationReport.json", &body)
	check := map[string]any{"name": "qos", "status": "failed", "observed_at": "2026-10-28T15:00:30Z", "error_code": "TC_FOREIGN_FILTER"}
	reportedError := map[string]any{"code": "TC_FOREIGN_FILTER", "stage": "verify", "message": "Serving verification found a tc filter on the member path; the node withdrew serving and installed the baseline.", "observed_at": "2026-10-28T15:00:30Z"}
	body["checks"] = []any{check}
	body["errors"] = []any{reportedError}
	raw := mustMarshal(t, body)
	if err := validateComponent(t, "ObservationReport", raw); err != nil {
		t.Fatal(err)
	}
	assertObservationRoundTrip(t, raw)
	for _, field := range []struct {
		object map[string]any
		key    string
	}{{check, "error_code"}, {reportedError, "code"}} {
		field.object[field.key] = "TC_UNKNOWN_FILTER"
		if err := validateComponent(t, "ObservationReport", mustMarshal(t, body)); err == nil {
			t.Fatalf("unknown observation %s must fail", field.key)
		}
		field.object[field.key] = "TC_FOREIGN_FILTER"
	}
}

func assertRevision13RoundTrip(t *testing.T, raw []byte, model any) []byte {
	t.Helper()
	if err := json.Unmarshal(raw, model); err != nil {
		t.Fatal(err)
	}
	encoded := mustMarshal(t, model)
	want, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := canonjson.Canonicalize(encoded)
	if err != nil || !bytes.Equal(want, got) {
		t.Fatalf("round trip changed JSON: got %s, want %s (err %v)", got, want, err)
	}
	return encoded
}
