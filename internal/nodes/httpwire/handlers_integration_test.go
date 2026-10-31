//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
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
	return setupRuntime(t, runtime)
}

func setupRuntime(t *testing.T, runtime *httpwire.Runtime) *fixture {
	t.Helper()
	pool, identity := runtime.Pool, runtime.PKI
	f := &fixture{t: t, r: runtime, p: auth.Principal{Role: auth.Admin, ID: uuid.New(), MFAVerified: true, AuthenticatedAt: time.Now(), CSRFToken: "test-csrf"}}
	handlers, e := httpapi.Compose(operations.Store{Pool: pool}, &audit.Store{Pool: pool}, &httpwire.Handlers{Runtime: runtime})
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
	// TEST ONLY below-floor issuer fixture: D-065 PUT has a 60 s lab floor,
	// so this 2 s window deliberately exercises the issuer beneath the API range.
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

func TestENR02_EnrollmentTokenSha256HTTPAndContractShape(t *testing.T) {
	// 1. Confirm contract example shapes for Enrollment, EnrollmentCreated, and EnrollmentCreated.replay.
	for _, tc := range []struct {
		path   string
		replay bool
	}{
		{path: "../../../contracts/examples/valid/EnrollmentCreated.json", replay: false},
		{path: "../../../contracts/examples/valid/EnrollmentCreated.replay.json", replay: true},
	} {
		raw, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		var ex api.EnrollmentCreated
		if err := json.Unmarshal(raw, &ex); err != nil {
			t.Fatalf("unmarshal %s: %v", tc.path, err)
		}
		if ex.Enrollment.TokenSha256 == nil || len(*ex.Enrollment.TokenSha256) != 64 {
			t.Fatalf("%s: expected 64-char token_sha256, got %v", tc.path, ex.Enrollment.TokenSha256)
		}
		if !tc.replay {
			if tok, err := ex.Token.Get(); err != nil || tok == "" {
				t.Fatalf("%s: expected non-null token: %v", tc.path, err)
			}
		} else if !ex.Token.IsNull() || ex.SecretAvailable {
			t.Fatalf("%s: replay example must have token=null and secret_available=false", tc.path)
		}
	}
	rawEnr, err := os.ReadFile("../../../contracts/examples/valid/Enrollment.json")
	if err != nil {
		t.Fatalf("read Enrollment.json: %v", err)
	}
	var exEnr api.Enrollment
	if err := json.Unmarshal(rawEnr, &exEnr); err != nil {
		t.Fatalf("unmarshal Enrollment.json: %v", err)
	}
	if exEnr.TokenSha256 == nil || len(*exEnr.TokenSha256) != 64 {
		t.Fatalf("Enrollment.json: expected 64-char token_sha256, got %v", exEnr.TokenSha256)
	}

	// 2. Assert HTTP handler responses for createEnrollment (create + replay), getEnrollment, revokeEnrollment, and audit.
	f := setup(t)
	idemKey := uuid.NewString()
	body := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller}

	wCreate := f.call("POST", "/api/v1/admin/enrollments", body, nil, map[string]string{"Idempotency-Key": idemKey})
	if got := wCreate.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("create Cache-Control = %q, want no-store", got)
	}
	created := decode[api.EnrollmentCreated](t, wCreate, 201)
	token, err := created.Token.Get()
	if err != nil || !created.SecretAvailable {
		t.Fatalf("create token missing: secret_available=%v err=%v", created.SecretAvailable, err)
	}
	sum := sha256.Sum256([]byte(token))
	wantDigest := hex.EncodeToString(sum[:])
	if created.Enrollment.TokenSha256 == nil || *created.Enrollment.TokenSha256 != wantDigest {
		t.Fatalf("create token_sha256 = %v, want %q", created.Enrollment.TokenSha256, wantDigest)
	}

	// Idempotent replay: plaintext token is withheld (null), token_sha256 remains identical.
	replayed := decode[api.EnrollmentCreated](t, f.call("POST", "/api/v1/admin/enrollments", body, nil, map[string]string{"Idempotency-Key": idemKey}), 201)
	if replayed.SecretAvailable || !replayed.Token.IsNull() {
		t.Fatalf("replay leaked secret: secret_available=%v token=%v", replayed.SecretAvailable, replayed.Token)
	}
	if replayed.Enrollment.TokenSha256 == nil || *replayed.Enrollment.TokenSha256 != wantDigest {
		t.Fatalf("replay token_sha256 = %v, want %q", replayed.Enrollment.TokenSha256, wantDigest)
	}

	// GET /api/v1/admin/enrollments/{enrollment_id}
	enrPath := "/api/v1/admin/enrollments/" + created.Enrollment.EnrollmentId.String()
	fetched := decode[api.Enrollment](t, f.call("GET", enrPath, nil, nil, nil), 200)
	if fetched.TokenSha256 == nil || *fetched.TokenSha256 != wantDigest {
		t.Fatalf("getEnrollment token_sha256 = %v, want %q", fetched.TokenSha256, wantDigest)
	}

	// POST /api/v1/admin/enrollments/{enrollment_id}/revoke
	revoked := decode[api.Enrollment](t, f.call("POST", enrPath+"/revoke", api.ReasonRequest{Reason: "TEST ONLY revoke unused token"}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 200)
	if revoked.State != api.EnrollmentStateRevoked || revoked.TokenSha256 == nil || *revoked.TokenSha256 != wantDigest {
		t.Fatalf("revokeEnrollment state=%q token_sha256=%v, want revoked / %q", revoked.State, revoked.TokenSha256, wantDigest)
	}

	// GET /api/v1/admin/audit-events confirms changes.after.token_sha256 is preserved (not redacted).
	auditList := decode[api.AuditEventList](t, f.call("GET", "/api/v1/admin/audit-events?resource_type=enrollment&resource_id="+created.Enrollment.EnrollmentId.String(), nil, nil, nil), 200)
	var createdAudit *api.AuditEvent
	for i := range auditList.Items {
		if auditList.Items[i].Action == "enrollment.created" {
			createdAudit = &auditList.Items[i]
			break
		}
	}
	if createdAudit == nil {
		t.Fatalf("expected enrollment.created audit event in %+v", auditList.Items)
	}
	afterMap, err := createdAudit.Changes.After.Get()
	if err != nil {
		t.Fatalf("audit changes.after: %v", err)
	}
	if got := afterMap["token_sha256"]; got != wantDigest {
		t.Fatalf("audit changes.after.token_sha256 = %v, want %q", got, wantDigest)
	}
}

func TestENR03_ListNodeIncarnationsHTTPReadModelAndRetainedUsage(t *testing.T) {
	// 1. Confirm contract example shape for NodeIncarnationList.json.
	rawExample, err := os.ReadFile("../../../contracts/examples/valid/NodeIncarnationList.json")
	if err != nil {
		t.Fatalf("read NodeIncarnationList.json: %v", err)
	}
	var exList api.NodeIncarnationList
	if err := json.Unmarshal(rawExample, &exList); err != nil {
		t.Fatalf("unmarshal NodeIncarnationList.json: %v", err)
	}
	if len(exList.Items) != 2 || !exList.Items[0].Current || exList.Items[1].Current {
		t.Fatalf("unexpected NodeIncarnationList.json example items: %+v", exList.Items)
	}

	f := setup(t)
	ctx := t.Context()

	// 2. RBAC checks: non-admin / member -> 403.
	adminPrincipal := f.p
	f.p = auth.Principal{Role: auth.Member, ID: uuid.New(), AuthenticatedAt: time.Now()}
	if w := f.call("GET", "/api/v1/admin/nodes/"+uuid.NewString()+"/incarnations", nil, nil, nil); w.Code != 403 {
		t.Fatalf("member status = %d, want 403", w.Code)
	}
	f.p = auth.Principal{Role: auth.Admin, ID: uuid.New(), MFAVerified: false, AuthenticatedAt: time.Now()}
	if w := f.call("GET", "/api/v1/admin/nodes/"+uuid.NewString()+"/incarnations", nil, nil, nil); w.Code != 403 {
		t.Fatalf("admin without MFA status = %d, want 403", w.Code)
	}
	f.p = adminPrincipal

	// 3. Unknown node -> 404 RESOURCE_NOT_FOUND.
	w404 := f.call("GET", "/api/v1/admin/nodes/"+uuid.NewString()+"/incarnations", nil, nil, nil)
	if w404.Code != 404 || !strings.Contains(w404.Body.String(), "RESOURCE_NOT_FOUND") {
		t.Fatalf("unknown node = %d %s, want 404 RESOURCE_NOT_FOUND", w404.Code, w404.Body.String())
	}

	// 4. Create enrollment (creates nodes row before any incarnation is bound).
	// Known node with no incarnations returns 200 with empty list and null next_cursor.
	enr1 := f.create(nil)
	nodeID := enr1.Enrollment.NodeId
	incPath := "/api/v1/admin/nodes/" + nodeID.String() + "/incarnations"

	emptyList := decode[api.NodeIncarnationList](t, f.call("GET", incPath, nil, nil, nil), 200)
	if len(emptyList.Items) != 0 || !emptyList.NextCursor.IsNull() {
		t.Fatalf("unconsumed node incarnations = %+v, want empty items and null next_cursor", emptyList)
	}

	// 5. Enroll incarnation 1.
	req1, enrolled1, _ := f.enroll(enr1)
	csr1, _, err := enrollment.VerifyProof(req1)
	if err != nil {
		t.Fatalf("VerifyProof 1: %v", err)
	}
	wantFP1 := hex.EncodeToString(pki.Fingerprint(csr1))
	inc1 := enrolled1.NodeIncarnationId

	// Prepare a member + grant + authorized device through domain services so usageSvc.IngestBatch can validate device_bindings.
	now := time.Now().UTC().Truncate(time.Second)
	periodID := now.Format("2006-01")
	var grp api.MemberGroup
	if err := db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var gErr error
		grp, gErr = f.r.Nodes.CreateGroupTx(ctx, tx, f.p.Scope(), nil, api.MemberGroupCreate{
			Code: "grp-enr03",
			Name: "ENR03 Group",
		})
		return gErr
	}); err != nil {
		t.Fatalf("CreateGroupTx: %v", err)
	}
	memberID := uuid.New()
	identityID := uuid.New()
	planID := uuid.New()
	planRevID := uuid.New()
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO identity_bindings (id, member_id, issuer, client_scope, subject, stable_claim_ref, binding_method, verified_at) VALUES ($1, $2, 'https://id.example.invalid', 'portal', 'TEST ONLY enr03 user', 'sub', 'oidc', $3)`, identityID, memberID, now); err != nil {
		t.Fatalf("insert identity_binding: %v", err)
	}
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO netbird_user_bindings (id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at) VALUES ($1, $2, 'netbird-prod', 'TEST ONLY enr03 user', $3, 'shared_issuer_public_subject:https://id.example.invalid', 1, $4)`, uuid.New(), memberID, identityID, now); err != nil {
		t.Fatalf("insert netbird_user_binding: %v", err)
	}
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, 'plan-enr03')`, planID); err != nil {
		t.Fatalf("insert plan: %v", err)
	}
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, []uuid.UUID{grp.GroupId}); err != nil {
		t.Fatalf("insert plan_revision: %v", err)
	}

	entStore, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatalf("entitlement.NewStore: %v", err)
	}
	entStore.Now = func() time.Time { return now }
	membersSvc := &members.Service{Pool: f.r.Pool, Entitlements: entStore}
	devicesSvc := &devices.Service{Pool: f.r.Pool, Entitlements: entStore}

	validFrom := now.Add(-2 * time.Hour)
	validUntil := now.Add(24 * time.Hour)
	if _, err := membersSvc.Grant(ctx, entitlement.Command{
		Actor: entitlement.Actor{Role: "admin", ID: f.p.ID, RequestID: uuid.New()},
		Key:   uuid.New(),
	}, memberID, api.GrantCreateRequest{
		Source:         api.GrantSourceAdminGrant,
		ValidFrom:      &validFrom,
		ValidUntil:     validUntil,
		Reason:         "TEST ONLY ENR03 grant",
		PlanRevisionId: &planRevID,
	}); err != nil {
		t.Fatalf("members.Grant: %v", err)
	}
	devOut, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{
		Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()},
		Key:   uuid.New(),
	}, devices.Peer{
		DeploymentID:   "netbird-prod",
		ID:             "TEST ONLY enr03 peer",
		ExternalUserID: "TEST ONLY enr03 user",
		OverlayScope:   "TEST ONLY",
		OverlayIP:      netip.MustParseAddr("100.64.0.42"),
		Platform:       "linux",
	})
	if err != nil {
		t.Fatalf("AuthorizeDevice: %v", err)
	}
	var bindingRev decstr.U64
	if err := f.r.Pool.QueryRow(ctx, `SELECT binding_revision::text FROM device_bindings WHERE device_id = $1 ORDER BY binding_revision DESC LIMIT 1`, devOut.DeviceID).Scan(&bindingRev); err != nil {
		t.Fatalf("query binding_revision: %v", err)
	}

	usageNow := now
	usageSvc, err := usage.NewService(usage.Config{Pool: f.r.Pool, Clock: func() time.Time { return usageNow }})
	if err != nil {
		t.Fatalf("usage.NewService: %v", err)
	}

	// 6. Checkpoint with no samples -> upload_bytes="0", download_bytes="0", and through set.
	if _, err := usageSvc.IngestBatch(ctx, usage.NodeIdentity{NodeID: nodeID, NodeIncarnationID: inc1}, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: inc1,
		Events:            []api.UsageEvent{},
	}); err != nil {
		t.Fatalf("IngestBatch empty: %v", err)
	}
	var wantZeroThrough time.Time
	if err := f.r.Pool.QueryRow(ctx, `SELECT updated_at FROM usage_checkpoints WHERE node_id = $1 AND incarnation_id = $2`, nodeID, inc1).Scan(&wantZeroThrough); err != nil {
		t.Fatalf("query zero-sample checkpoint updated_at: %v", err)
	}
	zeroSampleList := decode[api.NodeIncarnationList](t, f.call("GET", incPath, nil, nil, nil), 200)
	if len(zeroSampleList.Items) != 1 {
		t.Fatalf("expected 1 incarnation, got %d", len(zeroSampleList.Items))
	}
	zItem := zeroSampleList.Items[0]
	zThrough, err := zItem.Usage.Through.Get()
	if err != nil || zItem.Usage.UploadBytes.String() != "0" || zItem.Usage.DownloadBytes.String() != "0" {
		t.Fatalf("checkpoint with zero samples usage = %+v (err=%v), want 0/0 with non-null through", zItem.Usage, err)
	}
	if !zThrough.Equal(wantZeroThrough.UTC()) {
		t.Fatalf("through = %v, want %v", zThrough, wantZeroThrough.UTC())
	}

	// 7. Ingest usage on incarnation 1:
	// - Accepted event 1: upload_bytes = 9007199254740993 (2^53 + 1), download_bytes = 1024
	// - Accepted event 2: upload_bytes = 1000, download_bytes = 2048
	//   -> Sum on inc1: upload_bytes = "9007199254741993" (> 2^53), download_bytes = "3072"
	// - Conflicting event on seq 2 (upload_bytes = 999999) -> stored in usage_event_conflicts, ignored by read model
	// - Rejected event on seq 3 (upload_bytes = 888888, bad hash) -> stored in usage_event_rejections, ignored by read model
	makeEvent := func(incID uuid.UUID, seq uint64, start, end time.Time, up, down uint64) api.UsageEvent {
		ev := api.UsageEvent{
			NodeId:            nodeID,
			NodeIncarnationId: incID,
			EventSeq:          decstr.U64(seq),
			SchemaVersion:     api.UsageEventSchemaVersionN1,
			MeterEpochId:      uuid.MustParse("3e6c7b40-9651-4e7b-b724-0949fb7ab3dc"),
			MeteringVersion:   api.MeteringVersionForwardAdmissionV1,
			Quality:           api.UsageQualityComplete,
			IntervalStart:     start,
			IntervalEnd:       end,
			Records: []api.UsageRecord{
				{
					MemberId:        memberID,
					DeviceId:        devOut.DeviceID,
					BindingRevision: bindingRev,
					PeriodId:        periodID,
					UploadBytes:     decstr.U64(up),
					DownloadBytes:   decstr.U64(down),
				},
			},
		}
		_, hashHex, _, err := usage.ComputeEventHash(ev)
		if err != nil {
			t.Fatalf("ComputeEventHash: %v", err)
		}
		ev.PayloadSha256 = hashHex
		return ev
	}

	usageNow = now.Add(5 * time.Minute)
	ev1 := makeEvent(inc1, 1, now.Add(1*time.Minute), now.Add(2*time.Minute), 9007199254740993, 1024)
	ev2 := makeEvent(inc1, 2, now.Add(2*time.Minute), now.Add(3*time.Minute), 1000, 2048)
	ev2Conflict := makeEvent(inc1, 2, now.Add(2*time.Minute), now.Add(3*time.Minute), 999999, 999999)
	ev3Rejected := makeEvent(inc1, 3, now.Add(3*time.Minute), now.Add(4*time.Minute), 888888, 888888)
	ev3Rejected.PayloadSha256 = strings.Repeat("0", 64)

	batchRes1, err := usageSvc.IngestBatch(ctx, usage.NodeIdentity{NodeID: nodeID, NodeIncarnationID: inc1}, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: inc1,
		Events:            []api.UsageEvent{ev1, ev2, ev2Conflict, ev3Rejected},
	})
	if err != nil {
		t.Fatalf("IngestBatch inc1: %v", err)
	}
	if len(batchRes1.Results) != 4 ||
		batchRes1.Results[0].Status != api.UsageEventResultStatusAccepted ||
		batchRes1.Results[1].Status != api.UsageEventResultStatusAccepted ||
		batchRes1.Results[2].Status != api.UsageEventResultStatusConflict ||
		batchRes1.Results[3].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("unexpected batchRes1 statuses: %+v", batchRes1.Results)
	}
	var inc1Through time.Time
	if err := f.r.Pool.QueryRow(ctx, `SELECT updated_at FROM usage_checkpoints WHERE node_id = $1 AND incarnation_id = $2`, nodeID, inc1).Scan(&inc1Through); err != nil {
		t.Fatalf("query inc1 checkpoint updated_at: %v", err)
	}
	inc1Through = inc1Through.UTC()

	// 8. Revoke the node (before reinstalling) and verify inc1 is retired with retire_reason="revocation".
	detail1 := f.call("GET", "/api/v1/admin/nodes/"+nodeID.String(), nil, nil, nil)
	if detail1.Code != 200 {
		t.Fatalf("getNode 1: %d %s", detail1.Code, detail1.Body.String())
	}
	wRev := f.call("POST", "/api/v1/admin/nodes/"+nodeID.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY revoke before reinstall"}, nil, map[string]string{
		"Idempotency-Key": uuid.NewString(),
		"If-Match":        detail1.Header().Get("ETag"),
	})
	if wRev.Code != 202 {
		t.Fatalf("revoke node: %d %s", wRev.Code, wRev.Body.String())
	}

	revokedList := decode[api.NodeIncarnationList](t, f.call("GET", incPath, nil, nil, nil), 200)
	if len(revokedList.Items) != 1 {
		t.Fatalf("expected 1 incarnation after revoke, got %d", len(revokedList.Items))
	}
	revItem := revokedList.Items[0]
	revReason, err := revItem.RetireReason.Get()
	if err != nil || revItem.State != api.NodeIncarnationStateRetired || revReason != api.NodeIncarnationRetireReasonRevocation || revItem.RetiredAt.IsNull() {
		t.Fatalf("after revoke item = %+v, want state=retired, retire_reason=revocation, non-null retired_at", revItem)
	}
	if revItem.Usage.UploadBytes.String() != "9007199254741993" || revItem.Usage.DownloadBytes.String() != "3072" {
		t.Fatalf("after revoke usage = %+v, want 9007199254741993 / 3072", revItem.Usage)
	}

	// 9. Guarded reinstall path: create replacement enrollment for nodeID and enroll incarnation 2.
	enr2 := f.create(&nodeID)
	req2, enrolled2, _ := f.enroll(enr2)
	csr2, _, err := enrollment.VerifyProof(req2)
	if err != nil {
		t.Fatalf("VerifyProof 2: %v", err)
	}
	wantFP2 := hex.EncodeToString(pki.Fingerprint(csr2))
	inc2 := enrolled2.NodeIncarnationId
	if inc2 == inc1 {
		t.Fatal("expected distinct incarnation ID after reinstall")
	}

	// Record usage on incarnation 2 without a usage_checkpoints row so the join proves:
	// two incarnations, usage on both, a checkpoint on one (inc1), and a sum above 2^53 (on inc1).
	ledgerState, err := usageSvc.GetLedgerState(ctx)
	if err != nil {
		t.Fatalf("GetLedgerState: %v", err)
	}
	insertInc2Event := func(seq uint64, start, end time.Time, up, down uint64) {
		ev := makeEvent(inc2, seq, start, end, up, down)
		hashBytes, _, canonPayload, err := usage.ComputeEventHash(ev)
		if err != nil {
			t.Fatalf("ComputeEventHash inc2: %v", err)
		}
		evID := uuid.New()
		if err := db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `
				INSERT INTO usage_events (
					id, node_id, incarnation_id, event_seq, schema_version,
					payload_hash, record_count, received_at, quality_flags,
					kind, meter_epoch_id, metering_version, quality,
					interval_start, interval_end, ledger_epoch
				) VALUES (
					$1, $2, $3, $4::numeric, 1,
					$5, 1, $6, '{}',
					'event', $7, $8, 'complete',
					$9, $10, $11
				)`,
				evID, nodeID, inc2, decstr.U64(seq).String(),
				hashBytes[:], now.Add(15*time.Minute),
				ev.MeterEpochId, string(ev.MeteringVersion),
				start, end, ledgerState.LedgerEpoch,
			); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO usage_event_payloads (id, event_id, payload, created_at)
				VALUES ($1, $2, $3, $4)`,
				uuid.New(), evID, canonPayload, now.Add(15*time.Minute),
			); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO usage_samples (
					id, event_id, record_index, member_id, device_id,
					binding_revision, meter_epoch_id, interval_start, interval_end,
					upload_bytes, download_bytes, metering_version,
					monotonic_elapsed_ns, clock_quality, quality_flags
				) VALUES (
					$1, $2, 0, $3, $4,
					$5::numeric, $6, $7, $8,
					$9::numeric, $10::numeric, $11,
					60000000000, 'trusted', '{}'
				)`,
				uuid.New(), evID, memberID, devOut.DeviceID,
				bindingRev.String(), ev.MeterEpochId, start, end,
				decstr.U64(up).String(), decstr.U64(down).String(), string(ev.MeteringVersion),
			)
			return err
		}); err != nil {
			t.Fatalf("insert usage_events/usage_samples inc2 seq %d: %v", seq, err)
		}
	}
	insertInc2Event(1, now.Add(5*time.Minute), now.Add(6*time.Minute), 4096, 8192)

	bothList := decode[api.NodeIncarnationList](t, f.call("GET", incPath, nil, nil, nil), 200)
	if len(bothList.Items) != 2 || !bothList.NextCursor.IsNull() {
		t.Fatalf("expected 2 incarnations and null next_cursor, got %+v", bothList)
	}

	// Newest first: items[0] is inc2 (current, active), items[1] is inc1 (retired, reinstall).
	got2 := bothList.Items[0]
	got1 := bothList.Items[1]

	if got2.IncarnationId != inc2 || got2.EnrollmentId != enr2.Enrollment.EnrollmentId {
		t.Fatalf("items[0] identity = %+v, want inc2=%s enr2=%s", got2, inc2, enr2.Enrollment.EnrollmentId)
	}
	if !got2.Current || got2.State != api.NodeIncarnationStateActive || !got2.RetireReason.IsNull() || !got2.RetiredAt.IsNull() {
		t.Fatalf("items[0] lifecycle = %+v, want current=true state=active retire_reason=null retired_at=null", got2)
	}
	if got2.BoundPublicKeySha256 != wantFP2 {
		t.Fatalf("items[0] bound_public_key_sha256 = %q, want %q", got2.BoundPublicKeySha256, wantFP2)
	}
	if got2.Usage.UploadBytes.String() != "4096" || got2.Usage.DownloadBytes.String() != "8192" || !got2.Usage.Through.IsNull() {
		t.Fatalf("items[0] usage = %+v, want upload=4096 download=8192 through=null", got2.Usage)
	}

	if got1.IncarnationId != inc1 || got1.EnrollmentId != enr1.Enrollment.EnrollmentId {
		t.Fatalf("items[1] identity = %+v, want inc1=%s enr1=%s", got1, inc1, enr1.Enrollment.EnrollmentId)
	}
	got1Reason, err := got1.RetireReason.Get()
	if err != nil || got1.Current || got1.State != api.NodeIncarnationStateRetired || got1Reason != api.NodeIncarnationRetireReasonReinstall || got1.RetiredAt.IsNull() {
		t.Fatalf("items[1] lifecycle = %+v, want current=false state=retired retire_reason=reinstall", got1)
	}
	if got1.BoundPublicKeySha256 != wantFP1 {
		t.Fatalf("items[1] bound_public_key_sha256 = %q, want %q", got1.BoundPublicKeySha256, wantFP1)
	}
	got1Through, err := got1.Usage.Through.Get()
	if err != nil || got1.Usage.UploadBytes.String() != "9007199254741993" || got1.Usage.DownloadBytes.String() != "3072" || !got1Through.Equal(inc1Through) {
		t.Fatalf("items[1] retained usage = %+v, want upload=9007199254741993 download=3072 through=%v", got1.Usage, inc1Through)
	}

	// 10. Pagination: limit=1 returns page 1 (inc2) with next_cursor, then page 2 (inc1) with null next_cursor.
	page1 := decode[api.NodeIncarnationList](t, f.call("GET", incPath+"?limit=1", nil, nil, nil), 200)
	p1Cursor, err := page1.NextCursor.Get()
	if err != nil || len(page1.Items) != 1 || page1.Items[0].IncarnationId != inc2 {
		t.Fatalf("page1 = %+v, want [inc2] with next_cursor", page1)
	}
	page2 := decode[api.NodeIncarnationList](t, f.call("GET", incPath+"?limit=1&cursor="+p1Cursor, nil, nil, nil), 200)
	if len(page2.Items) != 1 || page2.Items[0].IncarnationId != inc1 || !page2.NextCursor.IsNull() {
		t.Fatalf("page2 = %+v, want [inc1] with null next_cursor", page2)
	}
	if wBadCur := f.call("GET", incPath+"?cursor=not-a-valid-cursor", nil, nil, nil); wBadCur.Code != 400 {
		t.Fatalf("malformed cursor status = %d, want 400", wBadCur.Code)
	}

	// 11. Sum above 2^64-1 fails closed (500 INTERNAL_ERROR), never returning a truncated total.
	insertInc2Event(2, now.Add(6*time.Minute), now.Add(7*time.Minute), ^uint64(0), 1)
	wOverflow := f.call("GET", incPath, nil, nil, nil)
	if wOverflow.Code != 500 || !strings.Contains(wOverflow.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("overflow sum status = %d %s, want 500 INTERNAL_ERROR", wOverflow.Code, wOverflow.Body.String())
	}
}

func TestSECOPS19_D077_HTTPNodeDetailUsageOutboxETagAndDomainColumn(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	enr1 := f.create(nil)
	_, enrolled1, cert1 := f.enroll(enr1)
	nodeID := enrolled1.NodeId
	inc1 := enrolled1.NodeIncarnationId
	nodePath := "/api/v1/admin/nodes/" + nodeID.String()

	// 1. Initial GET before any observation returns usage_outbox: null (never absent) and strong ETag.
	w0 := f.call("GET", nodePath, nil, nil, nil)
	if w0.Code != 200 {
		t.Fatalf("GET initial node: %d %s", w0.Code, w0.Body.String())
	}
	etag0 := w0.Header().Get("ETag")
	if etag0 == "" {
		t.Fatal("expected non-empty ETag header on initial GET")
	}
	var raw0 map[string]json.RawMessage
	if err := json.Unmarshal(w0.Body.Bytes(), &raw0); err != nil {
		t.Fatalf("unmarshal initial raw JSON: %v", err)
	}
	if val, ok := raw0["usage_outbox"]; !ok || string(val) != "null" {
		t.Fatalf("initial usage_outbox raw JSON = (%v, %s), want (true, null)", ok, string(val))
	}
	// If-None-Match with etag0 returns 304 Not Modified.
	w304 := f.call("GET", nodePath, nil, nil, map[string]string{"If-None-Match": etag0})
	if w304.Code != http.StatusNotModified || w304.Header().Get("ETag") != etag0 {
		t.Fatalf("If-None-Match initial = %d (ETag=%s), want 304 (ETag=%s)", w304.Code, w304.Header().Get("ETag"), etag0)
	}

	// 2. Load valid observation example and submit with a populated usage block over mTLS HTTP.
	rawExample, err := os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		t.Fatalf("read ObservationReport.json: %v", err)
	}
	var rep1 api.ObservationReport
	if err := json.Unmarshal(rawExample, &rep1); err != nil {
		t.Fatalf("unmarshal ObservationReport.json: %v", err)
	}
	rep1.NodeId = nodeID
	rep1.NodeIncarnationId = inc1
	rep1.ReportId = uuid.New()
	rep1.ObservedAt = time.Now().UTC().Truncate(time.Microsecond)
	rep1.CurrentReleaseId.SetNull()
	rep1.LastGoodReleaseId.SetNull()
	for i := range rep1.Checks {
		rep1.Checks[i].ObservedAt = rep1.ObservedAt
	}
	blockedSeq := decstr.U64(12059)
	rep1.Usage.BlockedSeq.Set(blockedSeq)

	wObs1 := f.call("POST", "/api/v1/agent/observations", rep1, cert1, nil)
	ack1 := decode[api.ObservationAccepted](t, wObs1, http.StatusAccepted)

	// Old ETag now misses (200 OK) and returns new ETag + populated usage_outbox.
	w1 := f.call("GET", nodePath, nil, nil, map[string]string{"If-None-Match": etag0})
	if w1.Code != http.StatusOK {
		t.Fatalf("GET after rep1 with old ETag = %d %s, want 200", w1.Code, w1.Body.String())
	}
	etag1 := w1.Header().Get("ETag")
	if etag1 == "" || etag1 == etag0 {
		t.Fatalf("ETag after rep1 = %q, want distinct from %q", etag1, etag0)
	}
	detail1 := decode[api.NodeDetail](t, w1, http.StatusOK)
	u1, err := detail1.UsageOutbox.Get()
	if err != nil {
		t.Fatalf("detail1.UsageOutbox.Get(): %v", err)
	}
	if !u1.ObservedAt.Equal(ack1.ReceivedAt) {
		t.Fatalf("u1.ObservedAt = %v, want ack1.ReceivedAt %v", u1.ObservedAt, ack1.ReceivedAt)
	}
	if gotBlocked, err := u1.BlockedSeq.Get(); err != nil || gotBlocked != blockedSeq {
		t.Fatalf("u1.BlockedSeq = (%v, %v), want %v", gotBlocked, err, blockedSeq)
	}
	if u1.LastDurableEventSeq != rep1.Usage.LastDurableEventSeq ||
		u1.CentralAckedThrough != rep1.Usage.CentralAckedThrough ||
		u1.UnackedEventCount != rep1.Usage.UnackedEventCount ||
		u1.UnackedPayloadBytes != rep1.Usage.UnackedPayloadBytes ||
		u1.AckedArchiveBytes != rep1.Usage.AckedArchiveBytes {
		t.Fatalf("u1 = %+v, want matching rep1.Usage %+v", u1, rep1.Usage)
	}

	// If-None-Match with etag1 returns 304 Not Modified.
	if w := f.call("GET", nodePath, nil, nil, map[string]string{"If-None-Match": etag1}); w.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match etag1 = %d, want 304", w.Code)
	}

	// 3. Submit a later observation without a usage block over HTTP so the latest
	// node_observations.report_payload has no "usage" key, and purge audit_events for this node.
	// Verifies both (b) report without a usage block leaves the previous projection intact and
	// (D-075) GET /admin/nodes/{id} reads from node_incarnations rather than audit_events or
	// the latest node_observations.report_payload.
	rep2 := rep1
	rep2.ReportId = uuid.New()
	rep2.ObservedAt = rep1.ObservedAt.Add(5 * time.Second)
	for i := range rep2.Checks {
		rep2.Checks[i].ObservedAt = rep2.ObservedAt
	}
	rep2Bytes, err := json.Marshal(rep2)
	if err != nil {
		t.Fatalf("marshal rep2: %v", err)
	}
	var rep2Map map[string]json.RawMessage
	if err := json.Unmarshal(rep2Bytes, &rep2Map); err != nil {
		t.Fatalf("unmarshal rep2Map: %v", err)
	}
	delete(rep2Map, "usage")
	var rep2NoUsage api.ObservationReport
	if err := json.Unmarshal(func() []byte { b, _ := json.Marshal(rep2Map); return b }(), &rep2NoUsage); err != nil {
		t.Fatalf("unmarshal rep2NoUsage: %v", err)
	}
	if _, err := f.r.Nodes.SubmitObservation(ctx, nodeID, inc1, rep2NoUsage); err != nil {
		t.Fatalf("SubmitObservation rep2NoUsage: %v", err)
	}

	var latestHasUsage bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT report_payload ? 'usage' FROM node_observations WHERE report_id = $1`, rep2.ReportId).Scan(&latestHasUsage); err != nil || latestHasUsage {
		t.Fatalf("latest observation report_payload ? 'usage' = (%v, %v), want (false, nil)", latestHasUsage, err)
	}

	if err := db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL fvpn.retention_purge = 'on'`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM audit_events WHERE resource = $1`, "node:"+nodeID.String()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("purge audit_events: %v", err)
	}

	w2 := f.call("GET", nodePath, nil, nil, map[string]string{"If-None-Match": etag1})
	if w2.Code != http.StatusOK {
		t.Fatalf("GET after rep2NoUsage with etag1 = %d %s, want 200", w2.Code, w2.Body.String())
	}
	if etag2 := w2.Header().Get("ETag"); etag2 == "" || etag2 == etag1 {
		t.Fatalf("ETag after rep2NoUsage = %q, want distinct from %q", etag2, etag1)
	}
	detailAfterPurge := decode[api.NodeDetail](t, w2, http.StatusOK)
	uAfterPurge, err := detailAfterPurge.UsageOutbox.Get()
	if err != nil || !uAfterPurge.ObservedAt.Equal(ack1.ReceivedAt) || uAfterPurge.LastDurableEventSeq != rep1.Usage.LastDurableEventSeq {
		t.Fatalf("after rep2NoUsage and purging audit_events, UsageOutbox = (%+v, %v), want preserved domain projection from rep1", uAfterPurge, err)
	}
}
