//go:build integration

package httpwire_test

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
)

func TestD056_D058_HTTPObservationDocumentEvidenceAndCheckDetail(t *testing.T) {
	runtime, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	type detail struct {
		api.NodeDetail
		Evidence presence.Nullable[api.DnsPolicyEvidence] `json:"dns_policy_evidence"`
	}
	before := decode[detail](t, f.call("GET", path, nil, nil, nil), http.StatusOK)
	if !before.Evidence.IsNull() {
		t.Fatal("unknown DNS evidence fabricated")
	}
	raw, err := os.ReadFile("../../../contracts/examples/valid/ObservationReport.resolver-document.json")
	if err != nil {
		t.Fatal(err)
	}
	var example api.ObservationReport
	if err = json.Unmarshal(raw, &example); err != nil {
		t.Fatal(err)
	}
	report.DnsPolicyEvidence = example.DnsPolicyEvidence
	message := "TEST ONLY baseline rules installed; generation acknowledged"
	report.Checks[0].Detail = &message
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call("POST", "/api/v1/agent/observations", report, cert, headers)
	decode[api.ObservationAccepted](t, first, http.StatusAccepted)
	replay := f.call("POST", "/api/v1/agent/observations", report, cert, headers)
	decode[api.ObservationAccepted](t, replay, http.StatusAccepted)
	if first.Body.String() != replay.Body.String() {
		t.Fatal("observation receipt replay changed")
	}
	w := f.call("GET", path, nil, nil, nil)
	got := decode[detail](t, w, http.StatusOK)
	if !reflect.DeepEqual(got.Evidence, report.DnsPolicyEvidence) || got.Checks[0].Detail == nil || *got.Checks[0].Detail != message {
		t.Fatalf("evidence/detail missing: %s", w.Body.String())
	}
	// The additive evidence projection remains valid against revision 5's open
	// NodeDetail schema while the typed generated field is coordinated upstream.
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var body any
	if err = json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if err = spec.Components.Schemas["NodeDetail"].Value.VisitJSON(body); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	var count int
	if err = runtime.Pool.QueryRow(t.Context(), `SELECT report_payload FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	var immutable api.ObservationReport
	if err = json.Unmarshal(stored, &immutable); err != nil || !reflect.DeepEqual(immutable.DnsPolicyEvidence, report.DnsPolicyEvidence) || *immutable.Checks[0].Detail != message {
		t.Fatal("immutable evidence changed", err)
	}
	if err = runtime.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate immutable report", err)
	}
	// Late reports cannot replace current evidence, and a newer report with no
	// evidence reports unknown rather than retaining an older applied document.
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(-time.Second)
	report.DnsPolicyEvidence.SetNull()
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
	got = decode[detail](t, f.call("GET", path, nil, nil, nil), http.StatusOK)
	if !reflect.DeepEqual(got.Evidence, example.DnsPolicyEvidence) {
		t.Fatal("stale report replaced evidence")
	}
	report.ReportId = uuid.New()
	report.ObservedAt = time.Now().UTC()
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
	got = decode[detail](t, f.call("GET", path, nil, nil, nil), http.StatusOK)
	if !got.Evidence.IsNull() {
		t.Fatal("unknown evidence replaced with last known policy")
	}
}
