//go:build integration

package httpwire_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestENR02_HTTPEnrollmentNextContractAndReplay(t *testing.T) {
	f := setup(t)
	validateFollowup28Responses(t, f)
	created := f.create(nil)
	request, first, cert := f.enroll(created)
	if first.Next != api.EnrollResponseNextApiv1agentdesiredState {
		t.Fatalf("enrollment next = %q", first.Next)
	}
	replay := decode[api.EnrollResponse](t, f.call("POST", "/api/v1/agent/enroll", request, nil,
		map[string]string{"Authorization": "Enrollment " + mustValue(t, created.Token)}), http.StatusOK)
	if replay.Next != first.Next || replay.CertificateSerial != first.CertificateSerial || replay.CertificatePem != first.CertificatePem {
		t.Fatal("enrollment replay changed next or reissued the certificate")
	}
	// The independent one-time mTLS bootstrap endpoint still reports its
	// provisioning prerequisite; next is the contract's desired-state link.
	failed := decode[api.ErrorEnvelope](t, f.call("POST", "/api/v1/agent/bootstrap", nil, cert, nil), http.StatusConflict)
	if failed.Error.Code != api.ErrorCodeBOOTSTRAPNOTREADY {
		t.Fatalf("bootstrap prerequisite = %s", failed.Error.Code)
	}
}

func TestD089_HTTPInstallerResultKeyedReplay(t *testing.T) {
	f := setup(t)
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	var err error
	f.r.Fleet, err = fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.r.Releases.Register(t.Context(), releasetest.Command(), rf.Request(t, rf.Manifest)); err != nil {
		t.Fatal(err)
	}
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	if err = db.InTx(t.Context(), f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return f.r.Releases.ApplySupportTx(t.Context(), tx, f.p.ID, support, "TEST ONLY installer receipt")
	}); err != nil {
		t.Fatal(err)
	}
	validateFollowup28Responses(t, f)
	_, enrolled, cert := f.enroll(f.create(nil))
	raw, err := os.ReadFile("../../../contracts/examples/valid/UpdateResult.installer.json")
	if err != nil {
		t.Fatal(err)
	}
	var body api.UpdateResult
	if err = json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	body.NodeId, body.IncarnationId = enrolled.NodeId, enrolled.NodeIncarnationId
	body.JobId, body.ResultId = uuid.New(), uuid.New()
	body.RequestedReleaseId = rf.Manifest.ReleaseId
	body.CurrentReleaseId.Set(rf.Manifest.ReleaseId)
	body.PreviousReleaseId.SetNull()
	body.PublicationIntent = enrolled.PublicationIntent
	body.StateSchemaVersion = rf.Manifest.Compatibility.StateSchemaTo
	body.StartedAt = time.Now().UTC()
	body.FinishedAt.Set(body.StartedAt)
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT encode(manifest_hash,'hex') FROM releases WHERE id=$1`, rf.Manifest.ReleaseId).Scan(&body.ManifestSha256); err != nil {
		t.Fatal(err)
	}
	hash := func(body api.UpdateResult) api.UpdateResult {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		body.PayloadHash, err = canonjson.HashHex(raw, "payload_hash")
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	body = hash(body)
	// UUID text is case-insensitive. Preserve an uppercase result_id in the
	// original payload to prove receipt binding uses UUID identity, not text.
	raw, err = json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	wire["result_id"], err = json.Marshal(strings.ToUpper(body.ResultId.String()))
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	body.PayloadHash, err = canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		t.Fatal(err)
	}
	wire["payload_hash"], err = json.Marshal(body.PayloadHash)
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call("POST", "/api/v1/agent/update-results", wire, cert, headers)
	ack := decode[api.UpdateResultAck](t, first, http.StatusAccepted)
	if !ack.Accepted || ack.ResultId != body.ResultId || !ack.AppliedToCurrentObservation {
		t.Fatalf("installer ACK = %+v", ack)
	}
	// A new HTTP stack proves replay uses the durable receipt. Removing target
	// approval must not turn replay into a second eligibility decision.
	support.ApprovedReleaseIds = []uuid.UUID{}
	if err = db.InTx(t.Context(), f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return f.r.Releases.ApplySupportTx(t.Context(), tx, f.p.ID, support, "TEST ONLY remove target approval after receipt")
	}); err != nil {
		t.Fatal(err)
	}
	f = setupRuntime(t, f.r)
	validateFollowup28Responses(t, f)
	for _, key := range []map[string]string{headers, {"Idempotency-Key": uuid.NewString()}} {
		replay := f.call("POST", "/api/v1/agent/update-results", wire, cert, key)
		decode[api.UpdateResultAck](t, replay, http.StatusAccepted)
		if !bytes.Equal(first.Body.Bytes(), replay.Body.Bytes()) {
			t.Fatal("installer replay changed original ACK")
		}
	}
	for _, change := range []func(*api.UpdateResult){
		func(r *api.UpdateResult) { r.BootId = uuid.New() },
		func(r *api.UpdateResult) { r.ResultId = uuid.New() },
	} {
		changed := body
		change(&changed)
		changed = hash(changed)
		// A distinct transport key reaches fleetupdate's existing immutable-job
		// conflict check. Reusing the original key is an HTTP command conflict.
		for _, tc := range []struct {
			key  map[string]string
			code api.ErrorCode
		}{
			{map[string]string{"Idempotency-Key": uuid.NewString()}, api.ErrorCodeACKCONFLICT},
			{headers, api.ErrorCodeIDEMPOTENCYCONFLICT},
		} {
			out := decode[api.ErrorEnvelope](t, f.call("POST", "/api/v1/agent/update-results", changed, cert, tc.key), http.StatusConflict)
			if out.Error.Code != tc.code {
				t.Fatalf("conflict = %s; want %s", out.Error.Code, tc.code)
			}
		}
	}
	var receipts, central, audits int
	var stored []byte
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT payload,
		(SELECT count(*) FROM installer_update_results), (SELECT count(*) FROM update_results),
		(SELECT count(*) FROM audit_events WHERE action='update_result.accept' AND after_diff->>'source'='installer')
		FROM installer_update_results WHERE node_id=$1 AND incarnation_id=$2 AND job_id=$3`, body.NodeId, body.IncarnationId, body.JobId).Scan(&stored, &receipts, &central, &audits); err != nil {
		t.Fatal(err)
	}
	var saved api.UpdateResult
	if err = json.Unmarshal(stored, &saved); err != nil || receipts != 1 || central != 0 || audits != 1 || saved.PayloadHash != body.PayloadHash || saved.ResultId != body.ResultId {
		t.Fatalf("receipt changed: installer=%d central=%d audits=%d err=%v", receipts, central, audits, err)
	}
}
