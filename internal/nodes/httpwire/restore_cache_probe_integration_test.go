//go:build integration

package httpwire_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// TestAT30_D095_HTTPRestoreCacheProbe promotes the policies follow-up 7 probe
// into a compiler + HTTP regression after D-099. dbtest (via stateCacheFixture)
// isolates each TEST ONLY restored snapshot. Observations and the applied ACK
// are simulated; this is not a PITR drill or a real node-local store test.
func TestAT30_D095_HTTPRestoreCacheProbe(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		desired, maxStored, retained decstr.U64
		completeBeforeObservation    bool
	}{
		{"original_probe", 2, 2, 5, false},
		{"secops21_generations", 11, 13, 14, false},
		{"complete_before_fresh_observation", 11, 13, 14, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, enrolled, report, provider := stateCacheFixture(t)
			r := f.r
			cert := pkitest.Certificate(t, enrolled.CertificatePem)
			const path = "/api/v1/agent/desired-state"
			// TEST ONLY snapshot setup: create a correctly signed cached offer at
			// the restored generation, with optionally higher stored history.
			f.sql(`UPDATE nodes SET desired_config_generation=$2::numeric,resource_version=resource_version+1 WHERE id=$1`, report.NodeId, (tc.desired - 1).String())
			first := f.call("GET", path, nil, cert, nil)
			cached := decode[api.DesiredStateEnvelope](t, first, http.StatusOK)
			if cached.Signed.ConfigGeneration != tc.desired {
				t.Fatalf("restored cache generation = %s; want %s", cached.Signed.ConfigGeneration, tc.desired)
			}
			if tc.maxStored > tc.desired {
				f.sql(`INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at)
					SELECT node_id,incarnation_id,$2::numeric,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at
					FROM desired_states WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, report.NodeId, tc.maxStored.String())
			}
			report.AppliedConfigGeneration.Set(tc.desired)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)

			type stateSnapshot struct {
				desired, maxStored, etag string
				rows                     int
				raw                      []byte
			}
			snapshot := func() stateSnapshot {
				t.Helper()
				var s stateSnapshot
				if err := r.Pool.QueryRow(t.Context(), `SELECT n.desired_config_generation::text,
					(SELECT max(generation)::text FROM desired_states WHERE node_id=n.id),
					(SELECT count(*) FROM desired_states WHERE node_id=n.id),e.envelope,e.etag
					FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id
					WHERE n.id=$1`, report.NodeId).Scan(&s.desired, &s.maxStored, &s.rows, &s.raw, &s.etag); err != nil {
					t.Fatal(err)
				}
				return s
			}
			before, signedBefore := snapshot(), provider.calls
			if before.desired != tc.desired.String() || before.maxStored != tc.maxStored.String() || before.etag == "" {
				t.Fatal("restored snapshot does not match the requested generation fences")
			}
			_, key, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			ledger, err := usage.NewService(usage.Config{Pool: r.Pool,
				LedgerEpochSigner: usage.NewLedgerEpochSigner("test-restore-epoch", key)})
			if err != nil {
				t.Fatal(err)
			}
			// Offline ordering: install the freeze on the restored database before
			// any post-restore HTTP compilation can run. No worker is running here.
			if err = ledger.FreezeIngest(t.Context(), nodes.SystemActorUUID.String(), "TEST ONLY restored central state"); err != nil {
				t.Fatal(err)
			}
			assertHeld := func() {
				t.Helper()
				w := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": before.etag})
				hold := decode[api.ErrorEnvelope](t, w, http.StatusServiceUnavailable)
				if hold.Error.Code != api.ErrorCodeDEPENDENCYUNAVAILABLE || hold.Error.Message != "pending fresh observation after restore" || !hold.Error.Retryable {
					t.Fatalf("restore hold = %+v", hold.Error)
				}
				if !reflect.DeepEqual(before, snapshot()) || provider.calls != signedBefore {
					t.Fatal("held HTTP request changed generation, stored history, envelope, signature or ETag")
				}
			}
			assertHeld()
			if _, err = ledger.BeginRecovery(t.Context(), usage.RecoveryRequest{
				Actor: nodes.SystemActorUUID.String(), Reason: "TEST ONLY restore", RestoredTo: time.Now().UTC().Add(-time.Hour),
			}); err != nil {
				t.Fatal(err)
			}
			if tc.completeBeforeObservation {
				if err = ledger.CompleteRecovery(t.Context(), nodes.SystemActorUUID.String()); err != nil {
					t.Fatal(err)
				}
			}
			assertHeld()
			// Replaying the old report after freeze must not establish freshness.
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			assertHeld()

			report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
			report.AppliedConfigGeneration.Set(tc.retained)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			next := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": before.etag})
			if next.Code == http.StatusNotModified {
				t.Fatalf("restored=%s stored=%s retained=%s first HTTP offer=%s (304); want %s", tc.desired, tc.maxStored, tc.retained, cached.Signed.ConfigGeneration, tc.retained+1)
			}
			advanced := decode[api.DesiredStateEnvelope](t, next, http.StatusOK)
			if advanced.Signed.ConfigGeneration != tc.retained+1 {
				t.Fatalf("restored=%s stored=%s retained=%s first HTTP offer=%s; want %s", tc.desired, tc.maxStored, tc.retained, advanced.Signed.ConfigGeneration, tc.retained+1)
			}
			if provider.calls != signedBefore+1 || reflect.DeepEqual(cached.Signatures, advanced.Signatures) || next.Header().Get("ETag") == "" || next.Header().Get("ETag") == before.etag {
				t.Fatal("first post-restore offer lacks a new signature or ETag")
			}
			trust, err := provider.TrustBundle(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = envelope.Verify(next.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); err != nil {
				t.Fatal(err)
			}
			after := snapshot()
			var stored api.DesiredStateEnvelope
			if err = json.Unmarshal(after.raw, &stored); err != nil {
				t.Fatal(err)
			}
			if after.desired != advanced.Signed.ConfigGeneration.String() || after.maxStored != after.desired || after.rows != before.rows+1 || after.etag != next.Header().Get("ETag") || !reflect.DeepEqual(stored, advanced) {
				t.Fatal("first HTTP offer does not match the single newly persisted central envelope")
			}
			detailPath := "/api/v1/admin/nodes/" + report.NodeId.String()
			detail := decode[api.NodeDetail](t, f.call("GET", detailPath, nil, nil, nil), http.StatusOK)
			if mustValue(t, detail.ConfigGeneration.Applied) != tc.retained {
				t.Fatal("issuing the offer must not claim an applied ACK")
			}
			var pending int
			if err = r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM operations WHERE resource_id=$1 AND principal=$2 AND status<>'succeeded'`,
				report.NodeId, usage.RestoreObservationPrincipal).Scan(&pending); err != nil || pending != 0 {
				t.Fatalf("restore receipt operation remains pending: %d, %v", pending, err)
			}

			// Simulated applied ACK through the production HTTP observation writer;
			// live SECOPS-21 still needs matching retained-node SQLite evidence.
			report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
			report.AppliedConfigGeneration.Set(advanced.Signed.ConfigGeneration)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
			detail = decode[api.NodeDetail](t, f.call("GET", detailPath, nil, nil, nil), http.StatusOK)
			if mustValue(t, detail.ConfigGeneration.Applied) != advanced.Signed.ConfigGeneration {
				t.Fatalf("simulated applied ACK = %s; want %s", mustValue(t, detail.ConfigGeneration.Applied), advanced.Signed.ConfigGeneration)
			}
			unchanged := f.call("GET", path, nil, cert, map[string]string{"If-None-Match": after.etag})
			if unchanged.Code != http.StatusNotModified || unchanged.Header().Get("ETag") != after.etag || provider.calls != signedBefore+1 || !reflect.DeepEqual(after, snapshot()) {
				t.Fatalf("ordinary unchanged request failed to reuse advanced cache: status=%d signs=%d", unchanged.Code, provider.calls)
			}
			t.Logf("SECOPS-21: restored=%s stored=%s retained=%s; hold=503 DEPENDENCY_UNAVAILABLE; first HTTP offer=%s with new verified signature/ETag; simulated applied ACK=%s; unchanged=304",
				tc.desired, tc.maxStored, tc.retained, advanced.Signed.ConfigGeneration, mustValue(t, detail.ConfigGeneration.Applied))
		})
	}
}
