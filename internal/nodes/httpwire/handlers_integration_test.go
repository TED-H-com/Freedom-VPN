//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fixture struct {
	t             *testing.T
	r             *httpwire.Runtime
	public, agent http.Handler
	p             auth.Principal
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := dbtest.New(t)
	vault, ca := pkitest.Authority(t)
	identity, e := pki.NewService(pool, ca, vault, pki.TestRenewalOverlap)
	if e != nil {
		t.Fatal(e)
	}
	en, e := enrollment.NewService(pool, ca, identity, vault, vault, enrollment.Config{BootstrapValidity: enrollment.TestBootstrapValidity})
	if e != nil {
		t.Fatal(e)
	}
	runtime := &httpwire.Runtime{Pool: pool, PKI: identity, Enrollment: en, Nodes: nodes.NewService(pool), Provider: vault}
	f := &fixture{t: t, r: runtime, p: auth.Principal{Role: auth.Admin, ID: uuid.New(), MFAVerified: true, AuthenticatedAt: time.Now(), CSRFToken: "test-csrf"}}
	handlers, e := httpapi.Compose(operations.Store{Pool: pool}, &httpwire.Handlers{Runtime: runtime})
	if e != nil {
		t.Fatal(e)
	}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.AnonymousRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.AgentRate = c.AnonymousRate
	c.BrowserRate = c.AnonymousRate
	c.IngressRate = c.AnonymousRate
	o := httpapi.Options{Config: c, Agents: identity, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return f.p, nil }), Commands: &httpapi.Commands{Pool: pool, Retention: time.Hour}}
	h, e := httpapi.New(handlers, o)
	if e != nil {
		t.Fatal(e)
	}
	f.public = httpapi.Transport(nil, httpwire.RequestContext(h))
	o.AgentListener = true
	h, e = httpapi.New(handlers, o)
	if e != nil {
		t.Fatal(e)
	}
	f.agent = httpapi.Transport(nil, httpwire.RequestContext(h))
	return f
}
func (f *fixture) call(method, path string, body any, cert *x509.Certificate, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	var raw []byte
	if body != nil {
		var e error
		raw, e = json.Marshal(body)
		if e != nil {
			f.t.Fatal(e)
		}
	}
	r := httptest.NewRequest(method, "https://control.example.invalid"+path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://portal.example.invalid")
	r.Header.Set("X-CSRF-Token", f.p.CSRFToken)
	for k, v := range headers {
		if v != "" {
			r.Header.Set(k, v)
		}
	}
	w := httptest.NewRecorder()
	if cert != nil {
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
		f.agent.ServeHTTP(w, r)
	} else {
		f.public.ServeHTTP(w, r)
	}
	return w
}
func decode[T any](t *testing.T, w *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d want %d: %s", w.Code, status, w.Body.String())
	}
	var v T
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func (f *fixture) create(node *uuid.UUID) api.EnrollmentCreated {
	f.t.Helper()
	return decode[api.EnrollmentCreated](f.t, f.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: "installer", NodeId: node}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 201)
}
func (f *fixture) enroll(c api.EnrollmentCreated) (api.EnrollRequest, api.EnrollResponse, *x509.Certificate) {
	f.t.Helper()
	token, e := c.Token.Get()
	if e != nil {
		f.t.Fatal(e)
	}
	r, _ := pkitest.Request(f.t)
	out := decode[api.EnrollResponse](f.t, f.call("POST", "/api/v1/agent/enroll", r, nil, map[string]string{"Authorization": "Enrollment " + token}), 201)
	return r, out, pkitest.Certificate(f.t, out.CertificatePem)
}
func TestAT06_AT07_AT28_HTTPEnrollmentReinstallRevocation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	key := uuid.NewString()
	body := api.EnrollmentCreateRequest{Delivery: "installer"}
	c := decode[api.EnrollmentCreated](t, f.call("POST", "/api/v1/admin/enrollments", body, nil, map[string]string{"Idempotency-Key": key}), 201)
	replay := decode[api.EnrollmentCreated](t, f.call("POST", "/api/v1/admin/enrollments", body, nil, map[string]string{"Idempotency-Key": key}), 201)
	if replay.SecretAvailable || !replay.Token.IsNull() || replay.Enrollment.EnrollmentId != c.Enrollment.EnrollmentId {
		t.Fatal("secret replayed")
	}
	r, out, cert := f.enroll(c)
	token, _ := c.Token.Get()
	again := decode[api.EnrollResponse](t, f.call("POST", "/api/v1/agent/enroll", r, nil, map[string]string{"Authorization": "Enrollment " + token}), 200)
	if again.CertificateSerial != out.CertificateSerial {
		t.Fatal("reissued on retry")
	}
	conflict, _ := pkitest.Request(t)
	w := f.call("POST", "/api/v1/agent/enroll", conflict, nil, map[string]string{"Authorization": "Enrollment " + token})
	if w.Code != 409 {
		t.Fatalf("key race %d %s", w.Code, w.Body.String())
	}
	w = f.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: "installer", NodeId: &out.NodeId}, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
	if w.Code != 409 || !strings.Contains(w.Body.String(), "NODE_INCARNATION_ACTIVE") {
		t.Fatalf("active reinstall %d %s", w.Code, w.Body.String())
	}
	w = f.call("POST", "/api/v1/agent/bootstrap", nil, cert, map[string]string{"X-Node-ID": uuid.NewString()})
	if w.Code != 409 || !strings.Contains(w.Body.String(), "BOOTSTRAP_NOT_READY") {
		t.Fatalf("bootstrap %d %s", w.Code, w.Body.String())
	}
	detail := f.call("GET", "/api/v1/admin/nodes/"+out.NodeId.String(), nil, nil, nil)
	if detail.Code != 200 {
		t.Fatal(detail.Body.String())
	}
	rev := f.call("POST", "/api/v1/admin/nodes/"+out.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY revocation"}, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")})
	if rev.Code != 202 {
		t.Fatalf("revoke %d %s", rev.Code, rev.Body.String())
	}
	if _, e := f.r.PKI.AuthenticateAgent(ctx, cert); e == nil {
		t.Fatal("revoked certificate authenticated")
	}
	replacement := f.create(&out.NodeId)
	_, newOut, newCert := f.enroll(replacement)
	if newOut.NodeId != out.NodeId || newOut.NodeIncarnationId == out.NodeIncarnationId {
		t.Fatal("reinstall identity")
	}
	if _, e := f.r.PKI.AuthenticateAgent(ctx, newCert); e != nil {
		t.Fatal(e)
	}
}
func TestAT28_HTTPBootstrapExactlyOnce(t *testing.T) {
	f := setup(t)
	c := f.create(nil)
	_, out, cert := f.enroll(c)
	ctx := context.Background()
	var op uuid.UUID
	if e := f.r.Pool.QueryRow(ctx, `SELECT operation_id FROM node_enrollments WHERE id=$1`, c.Enrollment.EnrollmentId).Scan(&op); e != nil {
		t.Fatal(e)
	}
	secret := "TEST ONLY bootstrap " + uuid.NewString()
	e := f.r.Enrollment.StoreBootstrap(ctx, pki.NodeIdentity{NodeID: out.NodeId, IncarnationID: out.NodeIncarnationId}, op, "test-external-key", secret, time.Now().Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	key := uuid.NewString()
	w := f.call("POST", "/api/v1/agent/bootstrap", nil, cert, map[string]string{"Idempotency-Key": key, "X-Node-ID": uuid.NewString()})
	got := decode[api.AgentBootstrap](t, w, 200)
	if got.SetupKey != secret || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("secret delivery")
	}
	for _, k := range []string{key, uuid.NewString(), ""} {
		w = f.call("POST", "/api/v1/agent/bootstrap", nil, cert, map[string]string{"Idempotency-Key": k})
		if w.Code != 410 || !strings.Contains(w.Body.String(), "BOOTSTRAP_CONSUMED") {
			t.Fatalf("consumption %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("secret replay")
		}
	}
	var erased bool
	if e := f.r.Pool.QueryRow(ctx, `SELECT ciphertext IS NULL FROM node_bootstrap_secrets WHERE incarnation_id=$1`, out.NodeIncarnationId).Scan(&erased); e != nil || !erased {
		t.Fatal("ciphertext retained", e)
	}
}

func TestAT23_AT24_AT26_HTTPBatchSnapshotAndResultEvidence(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	fleet, e := fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if e != nil {
		t.Fatal(e)
	}
	f.r.Fleet = fleet
	f.r.Nodes = nodes.NewService(f.r.Pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, r api.ObservationReport) error {
		return fleet.ObserveReportTx(ctx, tx, fleetupdate.Identity{NodeID: r.NodeId, IncarnationID: r.NodeIncarnationId}, r)
	}))
	var registration api.ReleaseRegisterRequest
	if e = json.Unmarshal(rf.Request(t, rf.Manifest), &registration); e != nil {
		t.Fatal(e)
	}
	decode[api.Release](t, f.call("POST", "/api/v1/admin/releases", registration, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 201)
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	if e = db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return f.r.Releases.ApplySupportTx(ctx, tx, f.p.ID, support, "TEST ONLY support")
	}); e != nil {
		t.Fatal(e)
	}
	c := f.create(nil)
	_, enrolled, cert := f.enroll(c)
	pending := f.create(nil)
	var g string
	if e = f.r.Pool.QueryRow(ctx, `SELECT generation::text FROM fleet_config`).Scan(&g); e != nil {
		t.Fatal(e)
	}
	generation, e := decstr.Parse(g)
	if e != nil {
		t.Fatal(e)
	}
	req := api.UpdateBatchCreateRequest{ReleaseId: rf.Manifest.ReleaseId, Scope: "all_nodes", ExpectedGeneration: generation, SetFleetDefault: true, Reason: "TEST ONLY fleet update"}
	key := uuid.NewString()
	batch := decode[api.UpdateBatchAccepted](t, f.call("POST", "/api/v1/admin/update-batches", req, nil, map[string]string{"Idempotency-Key": key}), 202)
	replay := decode[api.UpdateBatchAccepted](t, f.call("POST", "/api/v1/admin/update-batches", req, nil, map[string]string{"Idempotency-Key": key}), 202)
	if replay.BatchId != batch.BatchId || batch.TargetCount != 2 {
		t.Fatal("snapshot/replay changed")
	}
	req.Reason = "TEST ONLY changed"
	if w := f.call("POST", "/api/v1/admin/update-batches", req, nil, map[string]string{"Idempotency-Key": key}); w.Code != 409 {
		t.Fatalf("idempotency conflict %d", w.Code)
	}
	later := f.create(nil)
	_, after, afterCert := f.enroll(later)
	_ = afterCert
	_, bound, boundCert := f.enroll(pending)
	_ = boundCert
	work, e := fleet.NodeUpdateIntent(ctx, fleetupdate.Identity{NodeID: enrolled.NodeId, IncarnationID: enrolled.NodeIncarnationId})
	if e != nil || work == nil {
		t.Fatal(e)
	}
	if _, e = fleet.NodeUpdateIntent(ctx, fleetupdate.Identity{NodeID: bound.NodeId, IncarnationID: bound.NodeIncarnationId}); e != nil {
		t.Fatal(e)
	}
	if _, e = fleet.NodeUpdateIntent(ctx, fleetupdate.Identity{NodeID: after.NodeId, IncarnationID: after.NodeIncarnationId}); e != nil {
		t.Fatal(e)
	}
	view := decode[api.UpdateBatch](t, f.call("GET", "/api/v1/admin/update-batches/"+batch.BatchId.String(), nil, nil, nil), 200)
	if view.TargetCount != 2 || len(view.Targets) != 2 {
		t.Fatal("snapshot enlarged")
	}
	raw, e := os.ReadFile("../../../contracts/examples/valid/UpdateResult.json")
	if e != nil {
		t.Fatal(e)
	}
	var result api.UpdateResult
	if e = json.Unmarshal(raw, &result); e != nil {
		t.Fatal(e)
	}
	// D-048: accepted evidence must survive a newer desired state.
	policyGeneration := decstr.MustParse("9007199254740993")
	appliedPolicy, newerPolicy := uuid.New(), uuid.New()
	for offset, ref := range []uuid.UUID{appliedPolicy, newerPolicy} {
		f.sql(`INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at) VALUES($1,$2,$3::numeric,$4,0,$5,$6,1,clock_timestamp())`, enrolled.NodeId, enrolled.NodeIncarnationId, (policyGeneration + decstr.U64(offset)).String(), []uuid.UUID{ref}, rf.Manifest.ReleaseId, make([]byte, 32))
	}
	result.PolicyGeneration = policyGeneration
	intent := work.Intent
	result.ResultId = uuid.New()
	result.NodeId = intent.NodeId
	result.IncarnationId = intent.IncarnationId
	result.BatchId = intent.BatchId
	result.TargetId = intent.TargetId
	result.JobId = intent.JobId
	result.Attempt = intent.Attempt
	result.Generation = intent.Generation
	result.RequestedReleaseId = intent.ReleaseId
	result.CurrentReleaseId.Set(intent.ReleaseId)
	result.PreviousReleaseId.SetNull()
	result.ManifestSha256 = intent.ManifestSha256
	result.State = "already_current"
	result.PublicationIntent = intent.PublicationIntent
	result.MemberForwardingEnabled = false
	result.StartedAt = time.Now().UTC()
	result.FinishedAt.Set(time.Now().UTC())
	result.RecoveryRequired = presence.NewRequired(false)
	hash := func() {
		b, e := json.Marshal(result)
		if e != nil {
			t.Fatal(e)
		}
		result.PayloadHash, e = canonjson.HashHex(b, "payload_hash")
		if e != nil {
			t.Fatal(e)
		}
	}
	hash()
	resultKey := map[string]string{"Idempotency-Key": uuid.NewString()}
	firstResult := f.call("POST", "/api/v1/agent/update-results", result, cert, resultKey)
	ack := decode[api.UpdateResultAck](t, firstResult, 202)
	keyedResult := f.call("POST", "/api/v1/agent/update-results", result, cert, resultKey)
	decode[api.UpdateResultAck](t, keyedResult, 202)
	if !bytes.Equal(firstResult.Body.Bytes(), keyedResult.Body.Bytes()) {
		t.Fatal("keyed result replay changed receipt")
	}
	var resultRows int
	if e = f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM update_results WHERE result_id=$1`, result.ResultId).Scan(&resultRows); e != nil || resultRows != 1 {
		t.Fatal("keyed result replay wrote another row", resultRows, e)
	}
	again := decode[api.UpdateResultAck](t, f.call("POST", "/api/v1/agent/update-results", result, cert, nil), 202)
	if !ack.ReceivedAt.Equal(again.ReceivedAt) {
		t.Fatal("result receipt changed")
	}
	result.BootId = uuid.New()
	hash()
	if w := f.call("POST", "/api/v1/agent/update-results", result, cert, nil); w.Code != 409 || !strings.Contains(w.Body.String(), "ACK_CONFLICT") {
		t.Fatalf("result conflict %d %s", w.Code, w.Body.String())
	}
	raw, e = os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
	if e != nil {
		t.Fatal(e)
	}
	var report api.ObservationReport
	if e = json.Unmarshal(raw, &report); e != nil {
		t.Fatal(e)
	}
	report.NodeId = enrolled.NodeId
	report.NodeIncarnationId = enrolled.NodeIncarnationId
	report.ReportId = uuid.New()
	report.ObservedAt = time.Now().UTC()
	report.CurrentReleaseId.Set(intent.ReleaseId)
	report.LastGoodReleaseId.Set(intent.ReleaseId)
	report.Update.JobId.Set(intent.JobId)
	report.Update.Generation.Set(intent.Generation)
	report.MemberForwardingEnabled = false
	report.Netbird.PeerId.SetNull()
	report.Lease.LeaseId.SetNull()
	observationKey := map[string]string{"Idempotency-Key": uuid.NewString()}
	firstObservation := f.call("POST", "/api/v1/agent/observations", report, cert, observationKey)
	observed := decode[api.ObservationAccepted](t, firstObservation, 202)
	keyedObservation := f.call("POST", "/api/v1/agent/observations", report, cert, observationKey)
	decode[api.ObservationAccepted](t, keyedObservation, 202)
	if !bytes.Equal(firstObservation.Body.Bytes(), keyedObservation.Body.Bytes()) {
		t.Fatal("keyed observation replay changed receipt")
	}
	var observationRows int
	if e = f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&observationRows); e != nil || observationRows != 1 {
		t.Fatal("keyed observation replay wrote another row", observationRows, e)
	}
	duplicate := decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
	if !observed.ReceivedAt.Equal(duplicate.ReceivedAt) {
		t.Fatal("observation dedupe")
	}
	evidence, e := fleet.Evidence(ctx, mustValue(t, intent.TargetId))
	if e != nil || evidence.Unconfirmed || evidence.NeedsAttention {
		t.Fatalf("not confirmed: %+v %v", evidence, e)
	}
	view = decode[api.UpdateBatch](t, f.call("GET", "/api/v1/admin/update-batches/"+batch.BatchId.String(), nil, nil, nil), 200)
	rawPage, e := json.Marshal(view)
	if e != nil {
		t.Fatal(e)
	}
	if e = rf.Validator.ValidateBody("UpdateBatch", rawPage); e != nil {
		t.Fatal(e)
	}
	for _, target := range view.Targets {
		if target.NodeId == enrolled.NodeId {
			refs := mustValue(t, target.PolicyRefs)
			if mustValue(t, target.PolicyGeneration) != policyGeneration || len(refs) != 1 || refs[0] != appliedPolicy {
				t.Fatal("accepted policy evidence missing or replaced", target)
			}
		} else if !target.PolicyGeneration.IsSpecified() || !target.PolicyGeneration.IsNull() || !target.PolicyRefs.IsSpecified() || !target.PolicyRefs.IsNull() {
			t.Fatal("unknown policy evidence must be null", target)
		}
		if target.NodeId == enrolled.NodeId && (target.Unconfirmed != evidence.Unconfirmed || target.NeedsAttention != evidence.NeedsAttention) {
			t.Fatal("mapping drift")
		}
	}
	report.SafetyStatus = "failed"
	if w := f.call("POST", "/api/v1/agent/observations", report, cert, nil); w.Code != 409 {
		t.Fatalf("observation conflict %d %s", w.Code, w.Body.String())
	}
}
func mustValue[T any](t *testing.T, v interface{ Get() (T, error) }) T {
	t.Helper()
	x, e := v.Get()
	if e != nil {
		t.Fatal(e)
	}
	return x
}

func TestAT29_HTTPSignedLeaseWindowAndReplay(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, e := vault.Rotate(ctx, signing.Lease, 48*time.Hour, time.Hour); e != nil {
		t.Fatal(e)
	}
	_, enrolled, cert := f.enroll(f.create(nil))
	r := api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId, CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()}
	key := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call("POST", "/api/v1/agent/lease-renewals", r, cert, key)
	a := decode[api.ServingLeaseEnvelope](t, first, 200)
	replayed := f.call("POST", "/api/v1/agent/lease-renewals", r, cert, key)
	decode[api.ServingLeaseEnvelope](t, replayed, 200)
	if !bytes.Equal(first.Body.Bytes(), replayed.Body.Bytes()) {
		t.Fatal("keyed lease replay changed envelope or signature")
	}
	var leaseRows int
	if e := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM serving_leases WHERE renewal_request_id=$1`, r.RenewalRequestId).Scan(&leaseRows); e != nil || leaseRows != 1 {
		t.Fatal("keyed lease replay wrote another row", leaseRows, e)
	}
	b := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", r, cert, nil), 200)
	if a.Signed.LeaseId != b.Signed.LeaseId || !a.Signed.ExpiresAt.Equal(b.Signed.ExpiresAt) || a.Signed.ExpiresAt.Sub(a.Signed.IssuedAt) != 24*time.Hour || len(a.Signed.Members) != 0 {
		t.Fatal("lease replay or deadline changed")
	}
	trust, e := vault.TrustBundle(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(a)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = envelope.Verify(raw, trust, envelope.PurposeLease, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e = f.r.Pool.Exec(ctx, `UPDATE fleet_config SET max_offline_lease_seconds=2,resource_version=resource_version+1`); e != nil {
		t.Fatal(e)
	}
	r.RenewalRequestId = uuid.New()
	r.CurrentLeaseId.Set(a.Signed.LeaseId)
	bounded := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", r, cert, nil), 200)
	if bounded.Signed.ExpiresAt.Sub(bounded.Signed.IssuedAt) != 2*time.Second {
		t.Fatal("lease maximum ignored")
	}
	if _, e = f.r.Pool.Exec(ctx, `UPDATE node_certificates SET usable_until=clock_timestamp()-interval '1 second' WHERE incarnation_id=$1`, enrolled.NodeIncarnationId); e != nil {
		t.Fatal(e)
	}
	if w := f.call("POST", "/api/v1/agent/lease-renewals", r, cert, map[string]string{"X-Node-ID": enrolled.NodeId.String()}); w.Code != 403 {
		t.Fatalf("expired certificate reached handler %d %s", w.Code, w.Body.String())
	}
}

func TestAPI01_HTTPControlDesiredStateSignatureAndETag(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, e := vault.Rotate(ctx, signing.DesiredState, 48*time.Hour, time.Hour); e != nil {
		t.Fatal(e)
	}
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	var e error
	f.r.Fleet, e = fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if e != nil {
		t.Fatal(e)
	}
	f.r.Config.DesiredStateSeconds = 3600
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	f.r.Config.DNSResolverIPv6 = []string{"2001:db8::53"}
	definition, revision := uuid.New(), uuid.New()
	e = db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `INSERT INTO policy_definitions(id,type,code) VALUES($1,'release_support','TEST ONLY empty support')`, definition); e != nil {
			return e
		}
		hash, _ := canonjson.Hash([]byte(`{}`))
		if _, e := tx.Exec(ctx, `INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,'release_support',1,1,'{}',$3,'TEST ONLY key','TEST ONLY actor')`, revision, definition, hash[:]); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE policy_definitions SET active_revision_id=$2,resource_version=resource_version+1 WHERE id=$1`, definition, revision)
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	_, enrolled, cert := f.enroll(f.create(nil))
	w := f.call("GET", "/api/v1/agent/desired-state", nil, cert, map[string]string{"X-Node-ID": uuid.NewString()})
	env := decode[api.DesiredStateEnvelope](t, w, 200)
	if env.Signed.NodeId != enrolled.NodeId || len(env.Signed.DeviceBindings) != 0 || len(env.Signed.MemberRatePolicies) != 0 || len(env.Signed.ServingReleaseIds) != 0 {
		t.Fatal("staging grants or spoofed identity")
	}
	if len(env.Signed.Dns.ResolverIpv4) != 1 || env.Signed.Dns.ResolverIpv4[0] != "100.64.0.53" || len(env.Signed.Dns.ResolverIpv6) != 1 || env.Signed.Dns.ResolverIpv6[0] != "2001:db8::53" {
		t.Fatal("configured DNS resolvers missing from signed state")
	}
	trust, e := vault.TrustBundle(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = envelope.Verify(w.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); e != nil {
		t.Fatal(e)
	}
	cached := f.call("GET", "/api/v1/agent/desired-state", nil, cert, map[string]string{"If-None-Match": w.Header().Get("ETag")})
	if cached.Code != 304 {
		t.Fatalf("cache %d %s", cached.Code, cached.Body.String())
	}
	// Deployment changes must invalidate a cached envelope even when no policy
	// revision or node resource version changed (§8.1, §12.2).
	for _, ipv6 := range []bool{true, false} {
		if ipv6 {
			f.r.Config.DNSResolverIPv6 = []string{"2001:db8::54"}
		} else {
			f.r.Config.DNSResolverIPv4 = []string{"100.64.0.54"}
		}
		changed := f.call("GET", "/api/v1/agent/desired-state", nil, cert, map[string]string{"If-None-Match": w.Header().Get("ETag")})
		next := decode[api.DesiredStateEnvelope](t, changed, 200)
		if next.Signed.Dns.ResolverIpv4[0] != f.r.Config.DNSResolverIPv4[0] || next.Signed.Dns.ResolverIpv6[0] != f.r.Config.DNSResolverIPv6[0] || changed.Header().Get("ETag") == w.Header().Get("ETag") {
			t.Fatal("DNS change reused stale desired state")
		}
		if _, _, e = envelope.Verify(changed.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); e != nil {
			t.Fatal(e)
		}
		w = changed
	}

}

func TestENR02_HTTPCertificateRenewalAndIdentityChecks(t *testing.T) {
	f := setup(t)
	_, enrolled, old := f.enroll(f.create(nil))
	_, csr := pkitest.CSR(t)
	r := api.CertificateRenewRequest{NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId, RenewalRequestId: uuid.New(), CsrPem: csr}
	key := uuid.NewString()
	headers := map[string]string{"Idempotency-Key": key, "X-Node-ID": uuid.NewString()}
	a := decode[api.CertificateRenewResponse](t, f.call("POST", "/api/v1/agent/certificates/renew", r, old, headers), 201)
	b := decode[api.CertificateRenewResponse](t, f.call("POST", "/api/v1/agent/certificates/renew", r, old, headers), 201)
	if a.CertificateSerial != b.CertificateSerial || a.CertificateSerial == enrolled.CertificateSerial {
		t.Fatal("certificate renewal replay changed identity")
	}
	fresh := pkitest.Certificate(t, a.CertificatePem)
	if _, e := f.r.PKI.AuthenticateAgent(context.Background(), fresh); e != nil {
		t.Fatal(e)
	}
	r.NodeId = uuid.New()
	if w := f.call("POST", "/api/v1/agent/certificates/renew", r, fresh, nil); w.Code != 403 {
		t.Fatalf("body identity accepted: %d %s", w.Code, w.Body.String())
	}
	if _, e := f.r.Pool.Exec(context.Background(), `UPDATE node_certificates SET usable_until=clock_timestamp()-interval '1 second' WHERE incarnation_id=$1 AND serial=$2`, enrolled.NodeIncarnationId, enrolled.CertificateSerial); e != nil {
		t.Fatal(e)
	}
	if w := f.call("POST", "/api/v1/agent/bootstrap", nil, old, headers); w.Code != 403 {
		t.Fatalf("expired overlap accepted: %d %s", w.Code, w.Body.String())
	}
	if w := f.call("POST", "/api/v1/agent/bootstrap", nil, fresh, nil); w.Code != 409 {
		t.Fatalf("new certificate rejected: %d %s", w.Code, w.Body.String())
	}
}

func TestAT07_HTTPRetireAndPreconditions(t *testing.T) {
	f := setup(t)
	_, enrolled, cert := f.enroll(f.create(nil))
	path := "/api/v1/admin/nodes/" + enrolled.NodeId.String()
	for _, test := range []struct {
		tag    string
		status int
	}{{"", 428}, {`"0"`, 412}} {
		w := f.call("DELETE", path, nil, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": test.tag})
		if w.Code != test.status {
			t.Fatalf("precondition %d want %d: %s", w.Code, test.status, w.Body.String())
		}
	}
	detail := f.call("GET", path, nil, nil, nil)
	w := f.call("DELETE", path, nil, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")})
	if w.Code != 202 {
		t.Fatalf("retire %d %s", w.Code, w.Body.String())
	}
	if w := f.call("POST", "/api/v1/agent/bootstrap", nil, cert, nil); w.Code != 403 {
		t.Fatalf("retired identity accepted: %d", w.Code)
	}
	w = f.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: "installer", NodeId: &enrolled.NodeId}, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
	if w.Code != 409 || !strings.Contains(w.Body.String(), "NODE_RETIRED") {
		t.Fatalf("retired reinstall: %d %s", w.Code, w.Body.String())
	}
}
