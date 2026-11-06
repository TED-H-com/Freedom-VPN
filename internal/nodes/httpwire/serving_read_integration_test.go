//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestD164_HTTPServingReadAcceptanceDetailAndMutation(t *testing.T) {
	r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, r)
	validateFollowup28Responses(t, f)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	initial := f.call(http.MethodGet, path, nil, nil, nil)
	d := decode[api.NodeDetail](t, initial, http.StatusOK)
	if !d.ServingRead.IsSpecified() || !d.ServingRead.IsNull() {
		t.Fatal("missing null initial diagnostic")
	}
	// A later actual read timestamp is legal; false is retained diagnostic data.
	report.ServingRead.Set(api.ServingReadEvidence{ReadAt: report.ObservedAt.Add(time.Hour), Source: api.ServingReadEvidenceSourceApply, CandidateGeneration: decstr.U64(^uint64(0)), Verified: false, Diagnostic: strings.Repeat("界", 500)})
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := decode[api.ObservationAccepted](t, f.call(http.MethodPost, "/api/v1/agent/observations", report, cert, headers), http.StatusAccepted)
	requireSameReceipt(t, first, decode[api.ObservationAccepted](t, f.call(http.MethodPost, "/api/v1/agent/observations", report, cert, headers), http.StatusAccepted))
	requireSameReceipt(t, first, decode[api.ObservationAccepted](t, f.call(http.MethodPost, "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted))
	w := f.call(http.MethodGet, path, nil, nil, nil)
	d = decode[api.NodeDetail](t, w, http.StatusOK)
	if !reflect.DeepEqual(d.ServingRead, report.ServingRead) || w.Header().Get("ETag") == initial.Header().Get("ETag") {
		t.Fatal("GET diagnostic/version mismatch")
	}
	var payload, hash []byte
	if e := r.Pool.QueryRow(t.Context(), `SELECT report_payload,payload_hash FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&payload, &hash); e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(report)
	if e != nil {
		t.Fatal(e)
	}
	want, e := canonjson.Canonicalize(raw)
	if e != nil {
		t.Fatal(e)
	}
	stored, e := canonjson.Canonicalize(payload)
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(want)
	if !bytes.Equal(stored, want) || !bytes.Equal(hash, sum[:]) {
		t.Fatal("HTTP report values/hash changed")
	}
	conditional := f.call(http.MethodGet, path, nil, nil, map[string]string{"If-None-Match": w.Header().Get("ETag")})
	if conditional.Code != http.StatusNotModified {
		t.Fatal("strong ETag conditional GET", conditional.Code)
	}
	// Mutation responses and their idempotent projection use GetNodeTx/GetNode,
	// not only the GET wrapper that also attaches DNS evidence.
	mutationHeaders := map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": w.Header().Get("ETag")}
	patch := map[string]any{"notes": "TEST ONLY diagnostic projection"}
	updated := decode[api.NodeDetail](t, f.call(http.MethodPatch, path, patch, nil, mutationHeaders), http.StatusOK)
	if !reflect.DeepEqual(updated.ServingRead, report.ServingRead) {
		t.Fatal("mutation omitted diagnostic")
	}
	updated = decode[api.NodeDetail](t, f.call(http.MethodPatch, path, patch, nil, mutationHeaders), http.StatusOK)
	if !reflect.DeepEqual(updated.ServingRead, report.ServingRead) {
		t.Fatal("mutation replay omitted diagnostic")
	}
	// Advancing to a no-read observation clears availability and invalidates GET.
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(time.Second)
	report.ServingRead.SetNull()
	f = setupRuntime(t, r)
	validateFollowup28Responses(t, f)
	decode[api.ObservationAccepted](t, f.call(http.MethodPost, "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
	w = f.call(http.MethodGet, path, nil, nil, map[string]string{"If-None-Match": w.Header().Get("ETag")})
	d = decode[api.NodeDetail](t, w, http.StatusOK)
	if !d.ServingRead.IsNull() {
		t.Fatal("selected null recovered older read")
	}
	// The same report ID cannot change null into absence, even though both
	// project as null. Exercise the actual authenticated HTTP conflict boundary.
	report.ServingRead = api.ObservationReport{}.ServingRead
	conflict := f.call(http.MethodPost, "/api/v1/agent/observations", report, cert, nil)
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if e = json.Unmarshal(conflict.Body.Bytes(), &failure); e != nil || conflict.Code != http.StatusConflict || failure.Error.Code != "ACK_CONFLICT" {
		t.Fatalf("wrong null/absent conflict: %d %s %v", conflict.Code, conflict.Body.String(), e)
	}
	// A second valid source/outcome retains a decimal generation above 2^53,
	// then an omitted field clears it without recovering the earlier value.
	for _, withRead := range []bool{true, false} {
		report.ReportId = uuid.New()
		report.ObservedAt = report.ObservedAt.Add(time.Second)
		report.ServingRead = api.ObservationReport{}.ServingRead
		if withRead {
			report.ServingRead.Set(api.ServingReadEvidence{ReadAt: report.ObservedAt, Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: decstr.U64(9007199254740993), Verified: true, Diagnostic: "TEST ONLY opaque text {"})
		}
		f = setupRuntime(t, r)
		validateFollowup28Responses(t, f)
		decode[api.ObservationAccepted](t, f.call(http.MethodPost, "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
		d = decode[api.NodeDetail](t, f.call(http.MethodGet, path, nil, nil, nil), http.StatusOK)
		if withRead && !reflect.DeepEqual(d.ServingRead, report.ServingRead) {
			t.Fatal("verified live read lost decimal generation")
		}
		if !withRead && (!d.ServingRead.IsSpecified() || !d.ServingRead.IsNull()) {
			t.Fatal("selected omission recovered older read")
		}
	}
}

func TestD164_HTTPInvalidServingReadDoesNotPersist(t *testing.T) {
	r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	for name, change := range map[string]func(map[string]any){
		"source":             func(v map[string]any) { v["source"] = "invented" },
		"source_null":        func(v map[string]any) { v["source"] = nil },
		"missing_verdict":    func(v map[string]any) { delete(v, "verified") },
		"integer_generation": func(v map[string]any) { v["candidate_generation"] = 1 },
		"uint64_overflow":    func(v map[string]any) { v["candidate_generation"] = "18446744073709551616" },
		"too_long":           func(v map[string]any) { v["diagnostic"] = strings.Repeat("界", 501) },
		"offset_time":        func(v map[string]any) { v["read_at"] = "2026-11-06T00:00:00+00:00" },
		"invalid_calendar":   func(v map[string]any) { v["read_at"] = "2026-02-30T00:00:00Z" },
		"extra_field":        func(v map[string]any) { v["current_permission"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			raw, e := json.Marshal(report)
			if e != nil {
				t.Fatal(e)
			}
			var body map[string]any
			if e = json.Unmarshal(raw, &body); e != nil {
				t.Fatal(e)
			}
			body["report_id"] = uuid.NewString()
			value := map[string]any{"read_at": "2026-11-06T00:00:00Z", "source": "live", "candidate_generation": "9007199254740993", "verified": false, "diagnostic": "TEST ONLY {"}
			change(value)
			body["serving_read"] = value
			f := setupRuntime(t, r)
			w := f.call(http.MethodPost, "/api/v1/agent/observations", body, cert, nil)
			expectedStatus, expectedCode := http.StatusUnprocessableEntity, "VALIDATION_FAILED"
			if name == "uint64_overflow" || name == "invalid_calendar" {
				expectedStatus, expectedCode = http.StatusBadRequest, "MALFORMED_REQUEST"
			}
			var failure struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if e = json.Unmarshal(w.Body.Bytes(), &failure); e != nil {
				t.Fatal(e)
			}
			if w.Code != expectedStatus || failure.Error.Code != expectedCode {
				t.Fatalf("wrong validation rejection: %d %s", w.Code, w.Body.String())
			}
			var count int
			if e = r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_observations WHERE report_id=$1`, body["report_id"]).Scan(&count); e != nil || count != 0 {
				t.Fatal("invalid report persisted", e)
			}
		})
	}
}

func TestD164_HTTPCorruptServingReadIsInternalError(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	report.ServingRead.Set(api.ServingReadEvidence{ReadAt: report.ObservedAt, Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: decstr.U64(1), Verified: false, Diagnostic: "TEST ONLY"})
	if _, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	// Fault injection belongs only to this disposable test database. Original
	// immutable observation rows and their hashes are never changed.
	if _, err := r.Pool.Exec(t.Context(), `ALTER TABLE nodes DROP CONSTRAINT nodes_serving_read_shape`); err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]string{"source": "invented", "diagnostic": strings.Repeat("界", 501)} {
		t.Run(field, func(t *testing.T) {
			evidence, err := report.ServingRead.Get()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(evidence)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.Pool.Exec(t.Context(), `UPDATE nodes SET serving_read=jsonb_set($2::jsonb,ARRAY[$3::text],to_jsonb($4::text)),resource_version=resource_version+1 WHERE id=$1`, report.NodeId, raw, field, value); err != nil {
				t.Fatal(err)
			}
			f := setupRuntime(t, r)
			w := f.call(http.MethodGet, "/api/v1/admin/nodes/"+report.NodeId.String(), nil, nil, nil)
			var failure struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &failure); err != nil || w.Code != http.StatusInternalServerError || failure.Error.Code != "INTERNAL_ERROR" {
				t.Fatalf("corruption did not fail internally: %d %s %v", w.Code, w.Body.String(), err)
			}
		})
	}
}
