//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func setupWithTrustedProxies(t *testing.T, trustedProxies []string) *fixture {
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
	handlers, e := httpapi.Compose(operations.Store{Pool: pool}, &audit.Store{Pool: pool}, &httpwire.Handlers{Runtime: runtime})
	if e != nil {
		t.Fatal(e)
	}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.TrustedProxies = trustedProxies
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

func (f *fixture) callFrom(method, path string, body any, cert *x509.Certificate, remoteAddr string, headers map[string]string) *httptest.ResponseRecorder {
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
	if remoteAddr != "" {
		r.RemoteAddr = remoteAddr
	}
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

func loadObservationReport(t *testing.T, nodeID, incarnationID, releaseID uuid.UUID, observedAt time.Time) api.ObservationReport {
	t.Helper()
	raw, err := os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	report.NodeId = nodeID
	report.NodeIncarnationId = incarnationID
	report.ReportId = uuid.New()
	report.ObservedAt = observedAt.UTC()
	report.CurrentReleaseId.Set(releaseID)
	report.LastGoodReleaseId.Set(releaseID)
	report.Update.JobId.SetNull()
	report.Update.Generation.SetNull()
	report.MemberForwardingEnabled = false
	report.ProvisioningState = api.ProvisioningStateApplying
	report.SafetyStatus = api.SafetyStatusPassed
	report.Netbird.PeerId.SetNull()
	report.Lease.LeaseId.SetNull()
	return report
}

func seedApprovedRelease(t *testing.T, f *fixture) uuid.UUID {
	t.Helper()
	ctx := t.Context()
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	fleet, err := fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	f.r.Fleet = fleet
	var registration api.ReleaseRegisterRequest
	if err = json.Unmarshal(rf.Request(t, rf.Manifest), &registration); err != nil {
		t.Fatal(err)
	}
	decode[api.Release](t, f.call("POST", "/api/v1/admin/releases", registration, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 201)
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	definition, revision := uuid.New(), uuid.New()
	if err = db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if e := f.r.Releases.ApplySupportTx(ctx, tx, f.p.ID, support, "TEST ONLY support"); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `INSERT INTO policy_definitions(id,type,code) VALUES($1,'release_support','TEST ONLY support')`, definition); e != nil {
			return e
		}
		hash, _ := canonjson.Hash([]byte(`{}`))
		if _, e := tx.Exec(ctx, `INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,'release_support',1,1,'{}',$3,'TEST ONLY key','TEST ONLY actor')`, revision, definition, hash[:]); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE policy_definitions SET active_revision_id=$2,resource_version=resource_version+1 WHERE id=$1`, definition, revision)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	return rf.Manifest.ReleaseId
}

func domainSHA256(domain string) []byte {
	sum := sha256.Sum256([]byte(domain))
	return sum[:]
}

func TestD072_ClonedIdentityConflictRoamingFreezeAuditAndReenroll(t *testing.T) {
	f := setupWithTrustedProxies(t, []string{"192.0.2.0/24", "2001:db8:feed::/48"})
	ctx := t.Context()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, err := vault.Rotate(ctx, signing.Lease, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Rotate(ctx, signing.DesiredState, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	releaseID := seedApprovedRelease(t, f)
	f.r.Config.DesiredStateSeconds = 3600
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	f.r.Config.DNSResolverIPv6 = []string{"2001:db8::53"}

	now := time.Now().UTC().Truncate(time.Second)
	f.r.Enrollment.SetClock(func() time.Time { return now })

	// -------------------------------------------------------------------------
	// (g) GET /api/v1/agent/desired-state does not touch node_identity_endpoints
	// or node_identity_conflicts.
	// -------------------------------------------------------------------------
	_, enrolledG, certG := f.enroll(f.create(nil))
	wDesired := f.callFrom("GET", "/api/v1/agent/desired-state", nil, certG, "198.51.100.10:50000", nil)
	if wDesired.Code != http.StatusOK {
		t.Fatalf("GET desired-state status = %d: %s", wDesired.Code, wDesired.Body.String())
	}
	var endpointRows, conflictRows int
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledG.NodeIncarnationId).Scan(&endpointRows); err != nil {
		t.Fatal(err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledG.NodeIncarnationId).Scan(&conflictRows); err != nil {
		t.Fatal(err)
	}
	if endpointRows != 0 || conflictRows != 0 {
		t.Fatalf("GET desired-state touched identity tables: endpoints=%d conflicts=%d", endpointRows, conflictRows)
	}

	// -------------------------------------------------------------------------
	// (c) Same /24 (IPv4) and same /48 (IPv6) with different host addresses are
	// treated as one compatibility domain with no conflict.
	// -------------------------------------------------------------------------
	_, enrolledC4, certC4 := f.enroll(f.create(nil))
	for _, addr := range []string{"198.51.100.11:41001", "198.51.100.222:41002", "198.51.100.5:41003"} {
		now = now.Add(5 * time.Second)
		req := api.LeaseRenewalRequest{
			RenewalRequestId:        uuid.New(),
			NodeId:                  enrolledC4.NodeId,
			NodeIncarnationId:       enrolledC4.NodeIncarnationId,
			CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
			AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
		}
		decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", req, certC4, addr, nil), http.StatusOK)
	}
	var c4Hash []byte
	var c4Frozen bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT endpoint_hash, mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledC4.NodeIncarnationId).Scan(&c4Hash, &c4Frozen); err != nil {
		t.Fatal(err)
	}
	if c4Frozen || !bytes.Equal(c4Hash, domainSHA256("198.51.100.0/24")) {
		t.Fatalf("IPv4 /24 domain mismatch: frozen=%v hash=%x", c4Frozen, c4Hash)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledC4.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 0 {
		t.Fatalf("IPv4 same /24 recorded conflict: count=%d err=%v", conflictRows, err)
	}

	_, enrolledC6, certC6 := f.enroll(f.create(nil))
	for _, addr := range []string{"[2001:db8:abcd:0001::10]:41001", "[2001:db8:abcd:ffff::99]:41002", "[2001:db8:abcd:1234:5678::1]:41003"} {
		now = now.Add(5 * time.Second)
		req := api.LeaseRenewalRequest{
			RenewalRequestId:        uuid.New(),
			NodeId:                  enrolledC6.NodeId,
			NodeIncarnationId:       enrolledC6.NodeIncarnationId,
			CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
			AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
		}
		decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", req, certC6, addr, nil), http.StatusOK)
	}
	var c6Hash []byte
	var c6Frozen bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT endpoint_hash, mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledC6.NodeIncarnationId).Scan(&c6Hash, &c6Frozen); err != nil {
		t.Fatal(err)
	}
	if c6Frozen || !bytes.Equal(c6Hash, domainSHA256("2001:db8:abcd::/48")) {
		t.Fatalf("IPv6 /48 domain mismatch: frozen=%v hash=%x", c6Frozen, c6Hash)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledC6.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 0 {
		t.Fatalf("IPv6 same /48 recorded conflict: count=%d err=%v", conflictRows, err)
	}

	// -------------------------------------------------------------------------
	// (d) Behind a configured trusted_proxies address, X-Forwarded-For is the
	// domain; from an unknown source, X-Forwarded-For is ignored and the peer
	// address is the domain.
	// -------------------------------------------------------------------------
	_, enrolledProxy, certProxy := f.enroll(f.create(nil))
	leaseReqProxy := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolledProxy.NodeId,
		NodeIncarnationId:       enrolledProxy.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseReqProxy, certProxy, "192.0.2.10:443", map[string]string{
		"X-Forwarded-For": "203.0.113.77, 192.0.2.11",
	}), http.StatusOK)
	var proxyHash []byte
	if err := f.r.Pool.QueryRow(ctx, `SELECT endpoint_hash FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledProxy.NodeIncarnationId).Scan(&proxyHash); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(proxyHash, domainSHA256("203.0.113.0/24")) {
		t.Fatalf("trusted proxy X-Forwarded-For not honoured: got %x want %x", proxyHash, domainSHA256("203.0.113.0/24"))
	}

	_, enrolledUntrusted, certUntrusted := f.enroll(f.create(nil))
	leaseReqUntrusted := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolledUntrusted.NodeId,
		NodeIncarnationId:       enrolledUntrusted.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseReqUntrusted, certUntrusted, "198.51.100.88:443", map[string]string{
		"X-Forwarded-For": "203.0.113.77",
	}), http.StatusOK)
	var untrustedHash []byte
	if err := f.r.Pool.QueryRow(ctx, `SELECT endpoint_hash FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledUntrusted.NodeIncarnationId).Scan(&untrustedHash); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(untrustedHash, domainSHA256("198.51.100.0/24")) {
		t.Fatalf("untrusted peer X-Forwarded-For was not ignored: got %x want %x", untrustedHash, domainSHA256("198.51.100.0/24"))
	}

	// -------------------------------------------------------------------------
	// (b) Single switch X -> Y is roaming: no conflict, X remains mapped while
	// inside the active window, and promotes to Y after X has been quiet for a
	// full activeWindow.
	// -------------------------------------------------------------------------
	_, enrolledRoam, certRoam := f.enroll(f.create(nil))
	domainX := "198.51.100.10:50000"
	domainY := "203.0.113.20:50000"

	// Initial call from X at t0.
	roamLease := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolledRoam.NodeId,
		NodeIncarnationId:       enrolledRoam.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", roamLease, certRoam, domainX, nil), http.StatusOK)

	// Switch to Y 30s later (inside the 120s activeWindow): X stays mapped, Y is candidate, 0 conflicts.
	now = now.Add(30 * time.Second)
	roamObs1 := loadObservationReport(t, enrolledRoam.NodeId, enrolledRoam.NodeIncarnationId, releaseID, now)
	decode[api.ObservationAccepted](t, f.callFrom("POST", "/api/v1/agent/observations", roamObs1, certRoam, domainY, nil), http.StatusAccepted)

	var mappedHash []byte
	var roamFrozen bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT endpoint_hash, mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledRoam.NodeIncarnationId).Scan(&mappedHash, &roamFrozen); err != nil {
		t.Fatal(err)
	}
	if roamFrozen || !bytes.Equal(mappedHash, domainSHA256("198.51.100.0/24")) {
		t.Fatalf("during roaming window expected X to remain mapped and unfrozen: frozen=%v hash=%x", roamFrozen, mappedHash)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledRoam.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 0 {
		t.Fatalf("roaming switch recorded conflict: count=%d err=%v", conflictRows, err)
	}

	// After X has been quiet for > 120s (e.g. +100s, total 130s since X), Y promotes to the mapping.
	now = now.Add(100 * time.Second)
	roamObs2 := loadObservationReport(t, enrolledRoam.NodeId, enrolledRoam.NodeIncarnationId, releaseID, now)
	decode[api.ObservationAccepted](t, f.callFrom("POST", "/api/v1/agent/observations", roamObs2, certRoam, domainY, nil), http.StatusAccepted)

	if err := f.r.Pool.QueryRow(ctx, `SELECT endpoint_hash, mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledRoam.NodeIncarnationId).Scan(&mappedHash, &roamFrozen); err != nil {
		t.Fatal(err)
	}
	if roamFrozen || !bytes.Equal(mappedHash, domainSHA256("203.0.113.0/24")) {
		t.Fatalf("after X quiet for full window expected Y promoted and unfrozen: frozen=%v hash=%x", roamFrozen, mappedHash)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledRoam.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 0 {
		t.Fatalf("promoted roaming recorded conflict: count=%d err=%v", conflictRows, err)
	}

	// And if X is seen again within the window while Y is now mapped (Y and X
	// both seen within one window twice), it freezes as a conflict.
	now = now.Add(20 * time.Second)
	roamLease.RenewalRequestId = uuid.New()
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", roamLease, certRoam, domainX, nil), http.StatusOK)
	if err := f.r.Pool.QueryRow(ctx, `SELECT mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledRoam.NodeIncarnationId).Scan(&roamFrozen); err != nil || !roamFrozen {
		t.Fatalf("expected second window seeing {Y, X} to freeze mapping: frozen=%v err=%v", roamFrozen, err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledRoam.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 1 {
		t.Fatalf("expected 1 conflict row after {Y, X} seen in two windows: count=%d err=%v", conflictRows, err)
	}

	// -------------------------------------------------------------------------
	// (a) Interleaved X -> Y -> X inside the window across lease-renewal and
	// observation calls -> 1 conflict row, mapping_frozen=true, 1
	// node.identity.conflict audit (incarnation_id, conflict_count, observed_at
	// only), all three requests return 2xx, next observation projects
	// provisioning_state=failed, and later requests while frozen emit no second
	// audit or conflict row.
	// -------------------------------------------------------------------------
	_, enrolledA, certA := f.enroll(f.create(nil))

	// Pre-bind a known peer_id on enrolledA so (e) can also verify it is never overwritten after freeze.
	if _, err := f.r.Pool.Exec(ctx, `UPDATE node_incarnations SET deployment_id='netbird-prod', peer_id='peer-original-bound' WHERE id=$1`, enrolledA.NodeIncarnationId); err != nil {
		t.Fatal(err)
	}

	// Step 1: X lease renewal -> 200 OK.
	now = now.Add(10 * time.Second)
	leaseA1 := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolledA.NodeId,
		NodeIncarnationId:       enrolledA.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseA1, certA, domainX, nil), http.StatusOK)

	// Step 2: Y observation -> 202 Accepted (roaming candidate).
	now = now.Add(15 * time.Second)
	obsA2 := loadObservationReport(t, enrolledA.NodeId, enrolledA.NodeIncarnationId, releaseID, now)
	decode[api.ObservationAccepted](t, f.callFrom("POST", "/api/v1/agent/observations", obsA2, certA, domainY, nil), http.StatusAccepted)

	// Step 3: X lease renewal inside window -> 200 OK, triggers conflict freeze and 1 audit event.
	now = now.Add(15 * time.Second)
	conflictObservedAt := now
	leaseA3 := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolledA.NodeId,
		NodeIncarnationId:       enrolledA.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseA3, certA, domainX, nil), http.StatusOK)

	var aFrozen bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledA.NodeIncarnationId).Scan(&aFrozen); err != nil || !aFrozen {
		t.Fatalf("expected mapping_frozen=true after X->Y->X: frozen=%v err=%v", aFrozen, err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledA.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 1 {
		t.Fatalf("expected 1 conflict row after X->Y->X: count=%d err=%v", conflictRows, err)
	}

	// Next observation (claiming ready and forwarding=true) returns 202 Accepted,
	// projects provisioning_state=failed, safety_status=failed, forwarding=false,
	// and emits no second conflict row or audit event.
	now = now.Add(10 * time.Second)
	obsA4 := loadObservationReport(t, enrolledA.NodeId, enrolledA.NodeIncarnationId, releaseID, now)
	obsA4.ProvisioningState = api.ProvisioningStateReady
	obsA4.SafetyStatus = api.SafetyStatusPassed
	obsA4.MemberForwardingEnabled = true
	obsA4.Netbird.PeerId.Set("peer-attacker-clone")
	decode[api.ObservationAccepted](t, f.callFrom("POST", "/api/v1/agent/observations", obsA4, certA, domainY, nil), http.StatusAccepted)

	var provState, safetyStatus string
	var forwardingEnabled bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT provisioning_state, safety_status, member_forwarding_observed FROM nodes WHERE id=$1`, enrolledA.NodeId).Scan(&provState, &safetyStatus, &forwardingEnabled); err != nil {
		t.Fatal(err)
	}
	if provState != "failed" || safetyStatus != "failed" || forwardingEnabled {
		t.Fatalf("frozen incarnation observation projection = (%q, %q, %v), want (failed, failed, false)", provState, safetyStatus, forwardingEnabled)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledA.NodeIncarnationId).Scan(&conflictRows); err != nil || conflictRows != 1 {
		t.Fatalf("expected still 1 conflict row after post-freeze observation: count=%d err=%v", conflictRows, err)
	}

	// Verify audit event: exactly 1 node.identity.conflict event on resource node:<NodeId>
	// with only incarnation_id, conflict_count, observed_at in changes.after (no hashes, IPs, or domains).
	auditList := decode[api.AuditEventList](t, f.call("GET", "/api/v1/admin/audit-events?resource_type=node&resource_id="+enrolledA.NodeId.String(), nil, nil, nil), http.StatusOK)
	var conflictAudits []api.AuditEvent
	for _, ev := range auditList.Items {
		if ev.Action == "node.identity.conflict" {
			conflictAudits = append(conflictAudits, ev)
		}
	}
	if len(conflictAudits) != 1 {
		t.Fatalf("expected exactly 1 node.identity.conflict audit event, got %d (%+v)", len(conflictAudits), auditList.Items)
	}
	conflictAudit := conflictAudits[0]
	if conflictAudit.Resource.Type != api.ResourceTypeNode || conflictAudit.Resource.Id != enrolledA.NodeId {
		t.Fatalf("conflict audit resource = %+v, want node:%s", conflictAudit.Resource, enrolledA.NodeId)
	}
	afterMap, err := conflictAudit.Changes.After.Get()
	if err != nil {
		t.Fatalf("conflict audit changes.after: %v", err)
	}
	if len(afterMap) != 3 ||
		afterMap["incarnation_id"] != enrolledA.NodeIncarnationId.String() ||
		afterMap["conflict_count"] != float64(1) ||
		afterMap["observed_at"] != conflictObservedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected conflict audit changes.after = %+v", afterMap)
	}
	rawAuditBytes, _ := json.Marshal(conflictAudit)
	for _, forbidden := range []string{"198.51.100", "203.0.113", "endpoint_hash", "previous_hash", "conflicting_hash"} {
		if strings.Contains(string(rawAuditBytes), forbidden) {
			t.Fatalf("conflict audit leaked forbidden string %q: %s", forbidden, string(rawAuditBytes))
		}
	}

	// -------------------------------------------------------------------------
	// (e) While frozen, worker peer observe / discovery and observation paths
	// leave the incarnation's external identity binding untouched.
	// -------------------------------------------------------------------------
	var boundPeer string
	if err := f.r.Pool.QueryRow(ctx, `SELECT peer_id FROM node_incarnations WHERE id=$1`, enrolledA.NodeIncarnationId).Scan(&boundPeer); err != nil {
		t.Fatal(err)
	}
	if boundPeer != "peer-original-bound" {
		t.Fatalf("observation path mutated peer_id while frozen: got %q want peer-original-bound", boundPeer)
	}

	// Prove netbird.Client.EnsureNodePeer and DiscoverNodePeer skip binding writes
	// when the incarnation is frozen under WithDiscoveryFence.
	nbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/peers/peer-attacker-clone":
			_, _ = w.Write([]byte(`{"id":"peer-attacker-clone","ip":"100.64.0.99"}`))
		case "/api/peers":
			_, _ = w.Write([]byte(`[{"id":"peer-attacker-clone","ip":"100.64.0.99"}]`))
		case "/api/setup-keys/setup-key-1":
			_, _ = w.Write([]byte(`{"id":"setup-key-1","name":"fvpn-node-setup"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer nbServer.Close()

	nbClient, err := netbird.New(netbird.Config{
		BaseURL:           nbServer.URL + "/api",
		DeploymentID:      "netbird-prod",
		Token:             netbird.TokenFile(writeFixture(t, t.TempDir(), "token", []byte("TEST ONLY token"))),
		Bindings:          nodes.NetBirdBindings{Pool: f.r.Pool},
		Timeout:           5 * time.Second,
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	errFrozenFence := errors.New("incarnation frozen")
	fencedCtx := netbird.WithDiscoveryFence(ctx, func(ctx context.Context) error {
		var active bool
		if e := f.r.Pool.QueryRow(ctx, `SELECT n.current_incarnation_id=$2 AND n.lifecycle_state='registered' AND i.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=i.id AND x.mapping_frozen) FROM nodes n JOIN node_incarnations i ON i.id=n.current_incarnation_id WHERE n.id=$1`, enrolledA.NodeId, enrolledA.NodeIncarnationId).Scan(&active); e != nil {
			return e
		}
		if !active {
			return errFrozenFence
		}
		return nil
	})
	if _, err := nbClient.EnsureNodePeer(fencedCtx, netbird.NodePeer{
		NodeID:        enrolledA.NodeId.String(),
		IncarnationID: enrolledA.NodeIncarnationId.String(),
		PeerID:        "peer-attacker-clone",
	}); !errors.Is(err, errFrozenFence) {
		t.Fatalf("EnsureNodePeer under frozen fence err = %v, want %v", err, errFrozenFence)
	}
	if _, err := nbClient.DiscoverNodePeer(fencedCtx, uuid.NewString(), netbird.NodePeer{
		NodeID:        enrolledA.NodeId.String(),
		IncarnationID: enrolledA.NodeIncarnationId.String(),
	}); !errors.Is(err, errFrozenFence) {
		t.Fatalf("DiscoverNodePeer under frozen fence err = %v, want %v", err, errFrozenFence)
	}
	_, bound, err := (nodes.NetBirdBindings{Pool: f.r.Pool}).Get(ctx, "netbird-prod", "peers", "node:"+enrolledA.NodeId.String()+":"+enrolledA.NodeIncarnationId.String())
	if err != nil || bound {
		t.Fatalf("frozen incarnation wrote external_bindings row: bound=%v err=%v", bound, err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT peer_id FROM node_incarnations WHERE id=$1`, enrolledA.NodeIncarnationId).Scan(&boundPeer); err != nil || boundPeer != "peer-original-bound" {
		t.Fatalf("peer_id mutated while frozen: got %q err=%v", boundPeer, err)
	}

	// -------------------------------------------------------------------------
	// (f) Revoke + re-enroll produces a new unfrozen incarnation while old
	// conflict rows remain attached to the retired incarnation.
	// -------------------------------------------------------------------------
	detail := f.call("GET", "/api/v1/admin/nodes/"+enrolledA.NodeId.String(), nil, nil, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("GET node detail: %d %s", detail.Code, detail.Body.String())
	}
	rev := f.call("POST", "/api/v1/admin/nodes/"+enrolledA.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY cloned identity revoke"}, nil, map[string]string{
		"Idempotency-Key": uuid.NewString(),
		"If-Match":        detail.Header().Get("ETag"),
	})
	if rev.Code != http.StatusAccepted {
		t.Fatalf("revoke frozen node: %d %s", rev.Code, rev.Body.String())
	}

	reEnrollment := f.create(&enrolledA.NodeId)
	_, reEnrolled, reCert := f.enroll(reEnrollment)
	if reEnrolled.NodeId != enrolledA.NodeId || reEnrolled.NodeIncarnationId == enrolledA.NodeIncarnationId {
		t.Fatalf("re-enroll identity mismatch: got %+v", reEnrolled)
	}

	// New incarnation initially has no node_identity_endpoints row and 0 conflicts,
	// while the retired incarnation retains its frozen row and conflict evidence.
	var newEndpointRows, newConflictRows, oldConflictRows int
	var oldFrozen bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_endpoints WHERE incarnation_id=$1`, reEnrolled.NodeIncarnationId).Scan(&newEndpointRows); err != nil || newEndpointRows != 0 {
		t.Fatalf("new incarnation should have 0 endpoint rows right after enroll, got %d (err=%v)", newEndpointRows, err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, reEnrolled.NodeIncarnationId).Scan(&newConflictRows); err != nil || newConflictRows != 0 {
		t.Fatalf("new incarnation should have 0 conflict rows, got %d (err=%v)", newConflictRows, err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolledA.NodeIncarnationId).Scan(&oldFrozen); err != nil || !oldFrozen {
		t.Fatalf("retired incarnation lost mapping_frozen: frozen=%v err=%v", oldFrozen, err)
	}
	if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, enrolledA.NodeIncarnationId).Scan(&oldConflictRows); err != nil || oldConflictRows != 1 {
		t.Fatalf("retired incarnation lost conflict rows: count=%d err=%v", oldConflictRows, err)
	}

	// New incarnation can heartbeat and renew normally without being frozen.
	now = now.Add(10 * time.Second)
	reObs := loadObservationReport(t, reEnrolled.NodeId, reEnrolled.NodeIncarnationId, releaseID, now)
	decode[api.ObservationAccepted](t, f.callFrom("POST", "/api/v1/agent/observations", reObs, reCert, domainY, nil), http.StatusAccepted)
	var newFrozen bool
	if err := f.r.Pool.QueryRow(ctx, `SELECT mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, reEnrolled.NodeIncarnationId).Scan(&newFrozen); err != nil || newFrozen {
		t.Fatalf("new incarnation should be unfrozen after observation: frozen=%v err=%v", newFrozen, err)
	}
}

func assertIdentityConflictJSONKeys(t *testing.T, rawNodeDetail []byte, wantNull bool) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(rawNodeDetail, &doc); err != nil {
		t.Fatalf("unmarshal NodeDetail JSON: %v", err)
	}
	rawVal, present := doc["identity_conflict"]
	if !present {
		t.Fatalf("NodeDetail JSON is missing required projected field identity_conflict: %s", string(rawNodeDetail))
	}
	if wantNull {
		if rawVal != nil {
			t.Fatalf("identity_conflict = %#v, want null", rawVal)
		}
		return
	}
	obj, ok := rawVal.(map[string]any)
	if !ok {
		t.Fatalf("identity_conflict = %#v, want JSON object", rawVal)
	}
	if len(obj) != 3 {
		t.Fatalf("identity_conflict JSON object must have exactly 3 keys, got %d: %#v", len(obj), obj)
	}
	for _, k := range []string{"mapping_frozen", "conflict_count", "last_conflict_at"} {
		if _, exists := obj[k]; !exists {
			t.Fatalf("identity_conflict JSON object missing key %q: %#v", k, obj)
		}
	}
	for _, forbidden := range []string{"198.51.100", "203.0.113", "endpoint_hash", "previous_hash", "conflicting_hash"} {
		if strings.Contains(string(rawNodeDetail), forbidden) {
			t.Fatalf("NodeDetail JSON leaked forbidden string %q: %s", forbidden, string(rawNodeDetail))
		}
	}
}

func TestD072_NodeDetailIdentityConflictProjectionAndETag(t *testing.T) {
	f := setupWithTrustedProxies(t, []string{"192.0.2.0/24"})
	ctx := t.Context()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, err := vault.Rotate(ctx, signing.Lease, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Rotate(ctx, signing.DesiredState, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	releaseID := seedApprovedRelease(t, f)
	f.r.Config.DesiredStateSeconds = 3600
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	f.r.Config.DNSResolverIPv6 = []string{"2001:db8::53"}

	now := time.Now().UTC().Truncate(time.Second)
	f.r.Enrollment.SetClock(func() time.Time { return now })

	domainX := "198.51.100.10:50000"
	domainY := "203.0.113.20:50000"

	// -------------------------------------------------------------------------
	// (d) Node with no current incarnation -> identity_conflict is null.
	// -------------------------------------------------------------------------
	noIncNodeID := uuid.New()
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO nodes (id, lifecycle_state, provisioning_state) VALUES ($1, 'enrolling', 'pending')`, noIncNodeID); err != nil {
		t.Fatalf("insert enrolling node without current_incarnation_id: %v", err)
	}
	wNoInc := f.call("GET", "/api/v1/admin/nodes/"+noIncNodeID.String(), nil, nil, nil)
	detailNoInc := decode[api.NodeDetail](t, wNoInc, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wNoInc.Body.Bytes(), true)
	if !detailNoInc.IdentityConflict.IsSpecified() || !detailNoInc.IdentityConflict.IsNull() {
		t.Fatalf("node with no current incarnation: IdentityConflict = %+v, want specified null", detailNoInc.IdentityConflict)
	}

	// Enrolled node before any endpoint is observed -> also null.
	_, enrolled, cert := f.enroll(f.create(nil))
	wBeforeObs := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	detailBeforeObs := decode[api.NodeDetail](t, wBeforeObs, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wBeforeObs.Body.Bytes(), true)
	if !detailBeforeObs.IdentityConflict.IsSpecified() || !detailBeforeObs.IdentityConflict.IsNull() {
		t.Fatalf("enrolled node before first endpoint observation: IdentityConflict = %+v, want specified null", detailBeforeObs.IdentityConflict)
	}
	etagBeforeObs := wBeforeObs.Header().Get("ETag")
	if etagBeforeObs == "" {
		t.Fatal("expected ETag on GetNode response")
	}
	if w304 := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, map[string]string{"If-None-Match": etagBeforeObs}); w304.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match %s before change = %d, want 304", etagBeforeObs, w304.Code)
	}

	// -------------------------------------------------------------------------
	// (a) Enrolled node, no conflict -> {mapping_frozen: false, conflict_count: 0, last_conflict_at: null}.
	// -------------------------------------------------------------------------
	now = now.Add(5 * time.Second)
	leaseReq1 := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolled.NodeId,
		NodeIncarnationId:       enrolled.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseReq1, cert, domainX, nil), http.StatusOK)

	wNoConflict := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	detailNoConflict := decode[api.NodeDetail](t, wNoConflict, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wNoConflict.Body.Bytes(), false)
	icA, err := detailNoConflict.IdentityConflict.Get()
	if err != nil {
		t.Fatalf("expected non-null identity_conflict in (a): %v", err)
	}
	if icA.MappingFrozen || icA.ConflictCount != 0 || !icA.LastConflictAt.IsNull() {
		t.Fatalf("(a) identity_conflict = %+v, want {false, 0, null}", icA)
	}
	etagA := wNoConflict.Header().Get("ETag")
	if etagA == "" || etagA == etagBeforeObs {
		t.Fatalf("expected ETag to change when first endpoint row created: before=%q after=%q", etagBeforeObs, etagA)
	}
	if w304 := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, map[string]string{"If-None-Match": etagA}); w304.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match %s in (a) = %d, want 304", etagA, w304.Code)
	}

	// Verify listNodes / NodeSummary is unchanged and does not project identity_conflict.
	wList := f.call("GET", "/api/v1/admin/nodes", nil, nil, nil)
	if wList.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/admin/nodes = %d", wList.Code)
	}
	if strings.Contains(wList.Body.String(), "identity_conflict") {
		t.Fatalf("listNodes / NodeSummary must not contain identity_conflict: %s", wList.Body.String())
	}

	// -------------------------------------------------------------------------
	// (b) & (e) After an interleaving conflict -> {true, 1, <ts>}, ETag changes
	// across (a)->(b), 304 otherwise; and a second conflict row -> {true, 2, <newer ts>}.
	// -------------------------------------------------------------------------
	now = now.Add(10 * time.Second)
	leaseReq2 := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolled.NodeId,
		NodeIncarnationId:       enrolled.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseReq2, cert, domainY, nil), http.StatusOK)

	now = now.Add(10 * time.Second)
	conflict1At := now
	leaseReq3 := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolled.NodeId,
		NodeIncarnationId:       enrolled.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	decode[api.ServingLeaseEnvelope](t, f.callFrom("POST", "/api/v1/agent/lease-renewals", leaseReq3, cert, domainX, nil), http.StatusOK)

	// Old ETag from (a) must not return 304 now that conflict was recorded.
	wConflict1 := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, map[string]string{"If-None-Match": etagA})
	detailConflict1 := decode[api.NodeDetail](t, wConflict1, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wConflict1.Body.Bytes(), false)
	etagB1 := wConflict1.Header().Get("ETag")
	if etagB1 == "" || etagB1 == etagA {
		t.Fatalf("(e) expected ETag to change across (a)->(b): etagA=%q etagB1=%q", etagA, etagB1)
	}
	if w304 := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, map[string]string{"If-None-Match": etagB1}); w304.Code != http.StatusNotModified {
		t.Fatalf("(e) If-None-Match %s after conflict 1 = %d, want 304", etagB1, w304.Code)
	}
	icB1, err := detailConflict1.IdentityConflict.Get()
	if err != nil {
		t.Fatalf("expected non-null identity_conflict in (b): %v", err)
	}
	lastAt1, err := icB1.LastConflictAt.Get()
	if err != nil || !icB1.MappingFrozen || icB1.ConflictCount != 1 || !lastAt1.Equal(conflict1At) {
		t.Fatalf("(b) after 1st conflict: got %+v (last=%v err=%v), want {true, 1, %v}", icB1, lastAt1, err, conflict1At)
	}
	// Deliverable 3: no drift entry for identity_conflict.
	for _, d := range detailConflict1.Drift {
		if strings.Contains(d.Field, "identity") || strings.Contains(d.Field, "conflict") {
			t.Fatalf("unexpected drift entry for identity_conflict: %+v", d)
		}
	}

	// Second conflict row with a newer timestamp -> {true, 2, <newer ts>}.
	conflict2At := conflict1At.Add(25 * time.Second)
	if _, err := f.r.Pool.Exec(ctx, `INSERT INTO node_identity_conflicts (id, incarnation_id, previous_hash, conflicting_hash, observed_at) VALUES ($1, $2, $3, $4, $5)`,
		uuid.Must(uuid.NewV7()), enrolled.NodeIncarnationId, domainSHA256("198.51.100.0/24"), domainSHA256("192.0.2.0/24"), conflict2At); err != nil {
		t.Fatalf("insert second conflict row: %v", err)
	}
	if _, err := f.r.Pool.Exec(ctx, `UPDATE nodes SET resource_version = resource_version + 1 WHERE id = $1`, enrolled.NodeId); err != nil {
		t.Fatalf("bump resource_version for second conflict row: %v", err)
	}

	wConflict2 := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, map[string]string{"If-None-Match": etagB1})
	detailConflict2 := decode[api.NodeDetail](t, wConflict2, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wConflict2.Body.Bytes(), false)
	etagB2 := wConflict2.Header().Get("ETag")
	if etagB2 == "" || etagB2 == etagB1 {
		t.Fatalf("expected ETag to change on second conflict row: etagB1=%q etagB2=%q", etagB1, etagB2)
	}
	icB2, err := detailConflict2.IdentityConflict.Get()
	if err != nil {
		t.Fatalf("expected non-null identity_conflict after 2nd conflict: %v", err)
	}
	lastAt2, err := icB2.LastConflictAt.Get()
	if err != nil || !icB2.MappingFrozen || icB2.ConflictCount != 2 || !lastAt2.Equal(conflict2At) {
		t.Fatalf("(b) after 2nd conflict: got %+v (last=%v err=%v), want {true, 2, %v}", icB2, lastAt2, err, conflict2At)
	}

	// -------------------------------------------------------------------------
	// (c) After revokeNode + re-enroll -> new incarnation projects {false, 0, null}
	// (once observed; null before first endpoint observation) and the retired
	// incarnation's rows are not counted.
	// -------------------------------------------------------------------------
	revResp := f.call("POST", "/api/v1/admin/nodes/"+enrolled.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY revoke conflicted incarnation"}, nil, map[string]string{
		"Idempotency-Key": uuid.NewString(),
		"If-Match":        etagB2,
	})
	if revResp.Code != http.StatusAccepted {
		t.Fatalf("revokeNode: %d %s", revResp.Code, revResp.Body.String())
	}

	_, reEnrolled, reCert := f.enroll(f.create(&enrolled.NodeId))

	// Before first endpoint observation on the new incarnation -> null (retired incarnation's rows are ignored).
	wReBeforeObs := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	detailReBeforeObs := decode[api.NodeDetail](t, wReBeforeObs, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wReBeforeObs.Body.Bytes(), true)
	if !detailReBeforeObs.IdentityConflict.IsSpecified() || !detailReBeforeObs.IdentityConflict.IsNull() {
		t.Fatalf("re-enrolled node before first endpoint observation: IdentityConflict = %+v, want specified null", detailReBeforeObs.IdentityConflict)
	}

	// After first endpoint observation on the new incarnation -> {false, 0, null} (retired incarnation's 2 conflict rows not counted).
	now = conflict2At.Add(10 * time.Second)
	reObs := loadObservationReport(t, reEnrolled.NodeId, reEnrolled.NodeIncarnationId, releaseID, now)
	decode[api.ObservationAccepted](t, f.callFrom("POST", "/api/v1/agent/observations", reObs, reCert, domainY, nil), http.StatusAccepted)

	wReAfterObs := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	detailReAfterObs := decode[api.NodeDetail](t, wReAfterObs, http.StatusOK)
	assertIdentityConflictJSONKeys(t, wReAfterObs.Body.Bytes(), false)
	icC, err := detailReAfterObs.IdentityConflict.Get()
	if err != nil {
		t.Fatalf("expected non-null identity_conflict in (c): %v", err)
	}
	if icC.MappingFrozen || icC.ConflictCount != 0 || !icC.LastConflictAt.IsNull() {
		t.Fatalf("(c) after revokeNode + re-enroll: got %+v, want {false, 0, null}", icC)
	}
}
