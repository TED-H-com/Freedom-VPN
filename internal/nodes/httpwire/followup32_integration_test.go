//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Both ingestion entrypoints use the unmodified capture, including identity,
// timestamps, release, TC digest, reasons, check codes and stage errors.
func TestD126_MixedObservationReceiptAndProjections(t *testing.T) {
	for _, entrypoint := range []string{"SubmitObservation", "HTTP"} {
		t.Run(entrypoint, func(t *testing.T) {
			raw, err := os.ReadFile("../testdata/d126-mixed-observation.json")
			if err != nil {
				t.Fatal(err)
			}
			var report api.ObservationReport
			if err := json.Unmarshal(raw, &report); err != nil {
				t.Fatal(err)
			}
			f, cert := mixedObservationFixture(t, report)
			validateFollowup28Responses(t, f)
			ctx := t.Context()
			var pending []uuid.UUID
			for _, kind := range []string{nodes.OpKindPublish, nodes.OpKindUpdate} {
				if err := db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
					generation := uint64(mustValue(t, report.AppliedConfigGeneration))
					op, err := nodes.CreateOperation(ctx, tx, nodes.CreateOperationParams{
						Kind: kind, ResourceID: report.NodeId, Principal: f.p.Scope(), DesiredGeneration: &generation,
					})
					pending = append(pending, op.OperationId)
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			key := uuid.NewString()
			var ack api.ObservationAccepted
			if entrypoint == "HTTP" {
				ack = decode[api.ObservationAccepted](t, postMixedObservation(f, raw, cert, key), http.StatusAccepted)
			} else {
				ack, err = f.r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
				if err != nil {
					t.Fatal(err)
				}
			}
			if ack.Accepted != api.ObservationAcceptedAcceptedTrue || ack.ReportId != report.ReportId {
				t.Fatalf("report was not accepted: %+v", ack)
			}

			var stored, hash, checks []byte
			var equalJSON bool
			var safety, health, provisioning string
			var forwarding bool
			if err := f.r.Pool.QueryRow(ctx, `SELECT report_payload, payload_hash, checks,
				report_payload=$2::jsonb, safety_status, data_health, provisioning_state, member_forwarding_enabled
				FROM node_observations WHERE report_id=$1`, report.ReportId, raw).
				Scan(&stored, &hash, &checks, &equalJSON, &safety, &health, &provisioning, &forwarding); err != nil {
				t.Fatal(err)
			}
			// JSONB does not preserve whitespace/key order. Compare every value,
			// including array order and exact strings, and the canonical bytes/hash.
			want, err := canonjson.Canonicalize(raw)
			if err != nil {
				t.Fatal(err)
			}
			got, err := canonjson.Canonicalize(stored)
			digest := sha256.Sum256(want)
			if err != nil || !equalJSON || !bytes.Equal(want, got) || !bytes.Equal(hash, digest[:]) {
				t.Fatalf("stored report values or hash changed: %v", err)
			}
			if safety != "failed" || health != "failed" || provisioning != "applying" || forwarding {
				t.Fatalf("failure became applied/serving: %s/%s/%s/%v", safety, health, provisioning, forwarding)
			}
			var storedChecks map[api.NodeCheckName]struct {
				Status api.HealthCheckStatus `json:"status"`
				Code   api.ErrorCode         `json:"error_code"`
			}
			if err := json.Unmarshal(checks, &storedChecks); err != nil {
				t.Fatal(err)
			}
			for _, name := range []api.NodeCheckName{api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
				if check := storedChecks[name]; check.Status != api.HealthCheckStatusFailed || check.Code != api.ErrorCodeTCFOREIGNFILTER {
					t.Fatalf("stored %s check was normalized: %+v", name, check)
				}
			}

			path := "/api/v1/admin/nodes/" + report.NodeId.String()
			beforeReplay := f.call("GET", path, nil, nil, nil)
			detail := decode[api.NodeDetail](t, beforeReplay, http.StatusOK)
			domain, _, err := f.r.Nodes.GetNodeWithDNSEvidence(ctx, report.NodeId)
			if err != nil {
				t.Fatal(err)
			}
			for _, view := range []api.NodeDetail{domain, detail} {
				if !reflect.DeepEqual(view.Checks, report.Checks) || !reflect.DeepEqual(view.RecentErrors, report.Errors) ||
					!reflect.DeepEqual(view.DnsPolicyEvidence, report.DnsPolicyEvidence) {
					t.Fatal("D-093 node/portal view changed checks, stage causes or DNS evidence")
				}
				if view.SafetyStatus != api.SafetyStatusFailed || view.DataHealth != api.DataHealthFailed ||
					view.ProvisioningState != api.ProvisioningStateApplying || mustValue(t, view.MemberForwardingObserved) ||
					view.PublicationObserved != api.PublicationObservedWithdrawn {
					t.Fatalf("node view treats failure as applied/serving: %+v", view)
				}
				// The reported generation remains a claim/high-water mark; it is
				// deliberately not rewritten to imply an applied-policy ACK.
				if mustValue(t, view.ConfigGeneration.Applied) != mustValue(t, report.AppliedConfigGeneration) {
					t.Fatal("reported generation claim changed")
				}
			}
			for _, id := range pending {
				op := decode[api.Operation](t, f.call("GET", "/api/v1/admin/operations/"+id.String(), nil, nil, nil), http.StatusOK)
				if op.Status != api.OperationStatusPending || !op.ObservedGeneration.IsNull() || !op.Error.IsNull() {
					t.Fatalf("receipt became an applied operation ACK: %+v", op)
				}
				for _, step := range op.Steps {
					if step.Status != api.OperationStepStatusPending || !step.ObservedAt.IsNull() {
						t.Fatalf("receipt completed a step: %+v", step)
					}
				}
			}
			for _, plain := range []bool{false, true} {
				candidate := report
				if plain {
					candidate.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed}
				}
				cause, err := nodes.PublicationDataPathFailure(ctx, f.r.Pool, candidate, 30, report.ObservedAt)
				if err != nil || cause != "check_failed" {
					t.Fatalf("plain=%v publication failure=%q: %v", plain, cause, err)
				}
			}

			// Rebuild the HTTP stack, then cover transport-key replay, a new key,
			// and an unkeyed report_id replay. All must return the stored receipt.
			f = setupRuntime(t, f.r)
			validateFollowup28Responses(t, f)
			for _, replayKey := range []string{key, uuid.NewString(), ""} {
				replay := decode[api.ObservationAccepted](t, postMixedObservation(f, raw, cert, replayKey), http.StatusAccepted)
				if !reflect.DeepEqual(replay, ack) {
					t.Fatalf("replay changed stored ACK: %+v != %+v", replay, ack)
				}
			}
			replay, err := f.r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
			if err != nil || !reflect.DeepEqual(replay, ack) {
				t.Fatalf("domain replay changed stored ACK: %+v %v", replay, err)
			}
			afterReplay := f.call("GET", path, nil, nil, nil)
			after := decode[api.NodeDetail](t, afterReplay, http.StatusOK)
			if !reflect.DeepEqual(after.Checks, detail.Checks) || !reflect.DeepEqual(after.RecentErrors, detail.RecentErrors) ||
				after.SafetyStatus != detail.SafetyStatus || after.ProvisioningState != detail.ProvisioningState ||
				!reflect.DeepEqual(after.ConfigGeneration, detail.ConfigGeneration) {
				t.Fatal("replay rewrote node projections")
			}
			// Domain ingestion has no transport endpoint. Its first HTTP replay
			// legitimately creates identity_conflict metadata; the HTTP-first
			// case must preserve the entire body and ETag as well.
			if entrypoint == "HTTP" && (!bytes.Equal(beforeReplay.Body.Bytes(), afterReplay.Body.Bytes()) || beforeReplay.Header().Get("ETag") != afterReplay.Header().Get("ETag")) {
				t.Fatal("HTTP replay changed body or ETag")
			}
			// Removing one cumulative reason changes the payload and must still
			// conflict under the same report_id; acceptance is not weaker validation.
			changed := report
			changed.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed}
			conflict := decode[api.ErrorEnvelope](t, f.call("POST", "/api/v1/agent/observations", changed, cert, nil), http.StatusConflict)
			if conflict.Error.Code != api.ErrorCodeACKCONFLICT {
				t.Fatalf("changed report replay = %s", conflict.Error.Code)
			}
			var count int
			if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&count); err != nil || count != 1 {
				t.Fatalf("receipt count=%d: %v", count, err)
			}

			// Operations expose their own step/final error codes, not a copy of
			// the observation's arrays. Verify both stored causes through the real
			// failure writer and both existing operation projections (D-093).
			for _, index := range []int{0, 2} {
				code := report.Errors[index].Code
				var id uuid.UUID
				if err := db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
					op, err := nodes.CreateOperation(ctx, tx, nodes.CreateOperationParams{Kind: nodes.OpKindUpdate, ResourceID: report.NodeId, Principal: f.p.Scope()})
					if err != nil {
						return err
					}
					id = op.OperationId
					return operations.Transition(ctx, tx, id, nodes.StepAgentPolicy, api.OperationStepStatusFailed, nil, &code, nil)
				}); err != nil {
					t.Fatal(err)
				}
				domain, err := f.r.Nodes.GetOperation(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				wire := decode[api.Operation](t, f.call("GET", "/api/v1/admin/operations/"+id.String(), nil, nil, nil), http.StatusOK)
				for _, op := range []api.Operation{domain, wire} {
					if op.Status != api.OperationStatusFailed || mustValue(t, op.Error).Code != code || len(op.Steps) != 1 || mustValue(t, op.Steps[0].ErrorCode) != code {
						t.Fatalf("operation replaced %s cause: %+v", code, op)
					}
				}
			}
		})
	}
}

func mixedObservationFixture(t *testing.T, report api.ObservationReport) (*fixture, *x509.Certificate) {
	t.Helper()
	ctx := t.Context()
	cfg := nodestest.New(t, nodestest.NetBird{})
	runtime, err := httpwire.Open(ctx, dbtest.New(t), cfg.Nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := setupRuntime(t, runtime)
	// Seed only the identity/release prerequisites to retain every captured
	// report byte. Enrollment and certificate authentication still run normally.
	if _, err := runtime.Pool.Exec(ctx, `INSERT INTO nodes(id) VALUES($1)`, report.NodeId); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Pool.Exec(ctx, `INSERT INTO exits(node_id,display_name,region,publication_intent,publication_observed)
		VALUES($1,'TEST ONLY D-126','TEST ONLY','published','withdrawn')`, report.NodeId); err != nil {
		t.Fatal(err)
	}
	created := f.create(&report.NodeId)
	request, key := pkitest.Request(t)
	request.NodeIncarnationId = report.NodeIncarnationId
	pkitest.SignRequest(t, &request, key)
	enrolled := decode[api.EnrollResponse](t, f.call("POST", "/api/v1/agent/enroll", request, nil,
		map[string]string{"Authorization": "Enrollment " + mustValue(t, created.Token)}), http.StatusCreated)
	if _, err := runtime.Pool.Exec(ctx, `INSERT INTO releases(id,version,release_sequence,manifest_schema_version,manifest,manifest_hash,signature,signing_key_ref,component_versions)
		VALUES($1,'TEST ONLY D-126',1,1,'{}',decode(repeat('00',32),'hex'),decode(repeat('00',64),'hex'),'TEST ONLY','{}')`, mustValue(t, report.CurrentReleaseId)); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Pool.Exec(ctx, `UPDATE nodes SET desired_config_generation=30,resource_version=resource_version+1 WHERE id=$1`, report.NodeId); err != nil {
		t.Fatal(err)
	}
	return f, pkitest.Certificate(t, enrolled.CertificatePem)
}

// fixture.call marshals JSON and would compact RawMessage. This helper sends
// the captured file verbatim through authentication, schema checks and commands.
func postMixedObservation(f *fixture, raw []byte, cert *x509.Certificate, key string) *httptest.ResponseRecorder {
	f.t.Helper()
	r := httptest.NewRequest("POST", "https://control.example.invalid/api/v1/agent/observations", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
	w := httptest.NewRecorder()
	f.agent.ServeHTTP(w, r)
	return w
}
