//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type countingStateProvider struct {
	signing.Provider
	calls int
}

func (p *countingStateProvider) SignEnvelope(ctx context.Context, purpose signing.Purpose, signed any) (envelope.Envelope, error) {
	if purpose == signing.DesiredState {
		p.calls++
	}
	return p.Provider.SignEnvelope(ctx, purpose, signed)
}

func stateCacheFixture(t *testing.T) (*fixture, api.EnrollResponse, api.ObservationReport, *countingStateProvider) {
	t.Helper()
	r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	provider := &countingStateProvider{Provider: r.Provider}
	r.Provider = provider
	f := setupRuntime(t, r)
	release := releasetest.New(t)
	r.Releases.Validator = release.Validator
	if _, err := r.Releases.Register(t.Context(), releasetest.Command(), release.Request(t, release.Manifest)); err != nil {
		t.Fatal(err)
	}
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{release.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	if err := db.InTx(t.Context(), r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return r.Releases.ApplySupportTx(t.Context(), tx, f.p.ID, support, "TEST ONLY restore cache")
	}); err != nil {
		t.Fatal(err)
	}
	f.activate("release_support", uuid.Nil, 1, `{}`)
	validateFollowup28Responses(t, f)
	return f, enrolled.Response, report, provider
}

func TestAT30_D099_HTTPDesiredStateRestoreCache(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ahead     decstr.U64
		olderTime bool
	}{
		{"retained_generation_ahead", 3, false},
		{"fresh_receipt_with_older_node_clock", 3, true},
		{"retained_generation_equals_cache", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, enrolled, report, provider := stateCacheFixture(t)
			r := f.r
			cert := pkitest.Certificate(t, enrolled.CertificatePem)
			const path = "/api/v1/agent/desired-state"
			first := f.call("GET", path, nil, cert, nil)
			cached := decode[api.DesiredStateEnvelope](t, first, http.StatusOK)
			report.AppliedConfigGeneration.Set(cached.Signed.ConfigGeneration)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			snapshot := func() []byte {
				t.Helper()
				var raw []byte
				if err := r.Pool.QueryRow(t.Context(), `SELECT jsonb_build_object('generation',n.desired_config_generation::text,
					'rows',(SELECT count(*) FROM desired_states WHERE node_id=n.id),'envelope',e.envelope,'etag',e.etag)
					FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id WHERE n.id=$1`, report.NodeId).Scan(&raw); err != nil {
					t.Fatal(err)
				}
				return raw
			}
			before, signedBefore := snapshot(), provider.calls
			ledger, err := usage.NewService(usage.Config{Pool: r.Pool})
			if err != nil {
				t.Fatal(err)
			}
			if err = ledger.FreezeIngest(t.Context(), nodes.SystemActorUUID.String(), "TEST ONLY restored central state"); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				w := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": first.Header().Get("ETag")})
				hold := decode[api.ErrorEnvelope](t, w, http.StatusServiceUnavailable)
				if hold.Error.Code != api.ErrorCodeDEPENDENCYUNAVAILABLE || hold.Error.Message != usage.RestoreObservationStatus || !hold.Error.Retryable {
					t.Fatalf("restore hold = %+v", hold.Error)
				}
				if !bytes.Equal(before, snapshot()) || provider.calls != signedBefore {
					t.Fatal("hold changed desired state, envelope or signature")
				}
			}
			var operationID uuid.UUID
			if err = r.Pool.QueryRow(t.Context(), `SELECT id FROM operations WHERE resource_id=$1 AND principal=$2 ORDER BY created_at DESC LIMIT 1`, report.NodeId, usage.RestoreObservationPrincipal).Scan(&operationID); err != nil {
				t.Fatal(err)
			}
			op := decode[api.Operation](t, f.call("GET", "/api/v1/admin/operations/"+operationID.String(), nil, nil, nil), http.StatusOK)
			if op.Status != api.OperationStatusPending || len(op.Steps) != 1 || mustValue(t, op.Steps[0].Message) != usage.RestoreObservationStatus {
				t.Fatalf("restore operation = %+v", op)
			}
			detailPath := "/api/v1/admin/nodes/" + report.NodeId.String()
			detail := decode[api.NodeDetail](t, f.call("GET", detailPath, nil, nil, nil), http.StatusOK)
			held := false
			for _, drift := range detail.Drift {
				if drift.Field == usage.RestoreObservationField && mustValue(t, drift.Observed) == usage.RestoreObservationStatus {
					held = true
				}
			}
			if !held {
				t.Fatal("node detail omitted restore hold")
			}
			retained := cached.Signed.ConfigGeneration + tc.ahead
			report.ReportId = uuid.New()
			if tc.olderTime {
				report.ObservedAt = report.ObservedAt.Add(-time.Hour)
			} else {
				report.ObservedAt = time.Now().UTC()
			}
			report.AppliedConfigGeneration.Set(retained)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			next := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": first.Header().Get("ETag")})
			// Decode even a stale 304 so the regression reports the precise fence.
			if next.Code == http.StatusNotModified {
				t.Fatalf("restored=%s retained=%s HTTP offer=%s (304); want %s", cached.Signed.ConfigGeneration, retained, cached.Signed.ConfigGeneration, retained+1)
			}
			advanced := decode[api.DesiredStateEnvelope](t, next, http.StatusOK)
			if advanced.Signed.ConfigGeneration != retained+1 {
				t.Fatalf("restored=%s retained=%s HTTP offer=%s; want %s", cached.Signed.ConfigGeneration, retained, advanced.Signed.ConfigGeneration, retained+1)
			}
			if provider.calls != signedBefore+1 || reflect.DeepEqual(cached.Signatures, advanced.Signatures) || next.Header().Get("ETag") == first.Header().Get("ETag") {
				t.Fatal("required advance did not produce a fresh signature and ETag")
			}
			trust, err := provider.TrustBundle(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = envelope.Verify(next.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); err != nil {
				t.Fatal(err)
			}
			op = decode[api.Operation](t, f.call("GET", "/api/v1/admin/operations/"+operationID.String(), nil, nil, nil), http.StatusOK)
			if op.Status != api.OperationStatusSucceeded {
				t.Fatal("fresh receipt did not release restore hold")
			}
			// Simulate the node's applied ACK through the production observation
			// endpoint. This proves central projection, not real-device application.
			report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
			report.AppliedConfigGeneration.Set(advanced.Signed.ConfigGeneration)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			detail = decode[api.NodeDetail](t, f.call("GET", detailPath, nil, nil, nil), http.StatusOK)
			if mustValue(t, detail.ConfigGeneration.Applied) != advanced.Signed.ConfigGeneration {
				t.Fatalf("applied ACK = %s; want %s", mustValue(t, detail.ConfigGeneration.Applied), advanced.Signed.ConfigGeneration)
			}
			before = snapshot()
			unchanged := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": next.Header().Get("ETag")})
			if unchanged.Code != http.StatusNotModified || unchanged.Header().Get("ETag") != next.Header().Get("ETag") || provider.calls != signedBefore+1 || !bytes.Equal(before, snapshot()) {
				t.Fatalf("unchanged compile did not reuse advanced cache: status=%d sign calls=%d", unchanged.Code, provider.calls)
			}
			t.Logf("SECOPS-21: restored=%s retained=%s first HTTP offer=%s applied ACK=%s; unchanged=304, no re-sign", cached.Signed.ConfigGeneration, retained, advanced.Signed.ConfigGeneration, mustValue(t, detail.ConfigGeneration.Applied))
		})
	}
}

func TestAT30_D099_HTTPDesiredStateGenerationFences(t *testing.T) {
	for _, fence := range []string{"applied_generation", "stored_generation"} {
		t.Run(fence, func(t *testing.T) {
			f, enrolled, report, provider := stateCacheFixture(t)
			cert := pkitest.Certificate(t, enrolled.CertificatePem)
			const path = "/api/v1/agent/desired-state"
			first := f.call("GET", path, nil, cert, nil)
			cached := decode[api.DesiredStateEnvelope](t, first, http.StatusOK)
			retained := cached.Signed.ConfigGeneration + 3
			signedBefore := provider.calls
			if fence == "applied_generation" {
				report.AppliedConfigGeneration.Set(retained)
				decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			} else {
				// TEST ONLY restored snapshot: desired/envelope are behind stored
				// history. No restore operation exists to invalidate this cache.
				f.sql(`INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at)
					SELECT node_id,incarnation_id,$2::numeric,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at
					FROM desired_states WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, enrolled.NodeId, retained.String())
			}
			next := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": first.Header().Get("ETag")})
			advanced := decode[api.DesiredStateEnvelope](t, next, http.StatusOK)
			if advanced.Signed.ConfigGeneration != retained+1 || provider.calls != signedBefore+1 || reflect.DeepEqual(advanced.Signatures, cached.Signatures) {
				t.Fatalf("%s fence: cached=%s fence=%s offer=%s signs=%d", fence, cached.Signed.ConfigGeneration, retained, advanced.Signed.ConfigGeneration, provider.calls)
			}
			unchanged := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": next.Header().Get("ETag")})
			if unchanged.Code != http.StatusNotModified || provider.calls != signedBefore+1 {
				t.Fatalf("advanced cache not reused: status=%d signs=%d", unchanged.Code, provider.calls)
			}
		})
	}
}
