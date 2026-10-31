//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

const (
	reviewDomainX = "198.51.100.10:50000"
	reviewDomainY = "203.0.113.20:50000"
	reviewPrefixX = "198.51.100.0/24"
	reviewPrefixY = "203.0.113.0/24"
)

func reviewRotateLease(t *testing.T, f *fixture) {
	t.Helper()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, err := vault.Rotate(t.Context(), signing.Lease, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
}

func reviewRenew(t *testing.T, f *fixture, enrolled api.EnrollResponse, cert *x509.Certificate, remote string) api.ServingLeaseEnvelope {
	t.Helper()
	body := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolled.NodeId,
		NodeIncarnationId:       enrolled.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	w := f.callFrom("POST", "/api/v1/agent/lease-renewals", body, cert, remote, nil)
	return decode[api.ServingLeaseEnvelope](t, w, http.StatusOK)
}

func reviewLeaseTTL(env api.ServingLeaseEnvelope) time.Duration {
	return env.Signed.ExpiresAt.Sub(env.Signed.IssuedAt)
}

func reviewEndpointCount(t *testing.T, f *fixture, inc uuid.UUID) int {
	t.Helper()
	var n int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_identity_endpoints WHERE incarnation_id=$1`, inc).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func reviewConflictCount(t *testing.T, f *fixture, inc uuid.UUID) int {
	t.Helper()
	var n int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, inc).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func reviewFrozen(t *testing.T, f *fixture, inc uuid.UUID) bool {
	t.Helper()
	var frozen bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1`, inc).Scan(&frozen); err != nil {
		t.Fatal(err)
	}
	return frozen
}

// TestReviewR15_RightmostHopIsTheStoredDomain drives the agent listener with
// deploy's proxy address. nginx appends the client ($proxy_add_x_forwarded_for);
// the leftmost victim address is not the row that ObserveEndpointTx stores.
func TestReviewR15_RightmostHopIsTheStoredDomain(t *testing.T) {
	f := setupWithTrustedProxies(t, []string{"172.28.30.21/32"})
	reviewRotateLease(t, f)
	_, enrolled, cert := f.enroll(f.create(nil))
	body := api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  enrolled.NodeId,
		NodeIncarnationId:       enrolled.NodeIncarnationId,
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	w := f.callFrom("POST", "/api/v1/agent/lease-renewals", body, cert, "172.28.30.21:443", map[string]string{
		"X-Forwarded-For": "198.51.100.10, 203.0.113.44",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("lease status %d: %s", w.Code, w.Body.String())
	}
	var got []byte
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT endpoint_hash FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolled.NodeIncarnationId).Scan(&got); err != nil {
		t.Fatal(err)
	}
	appended := domainSHA256("203.0.113.0/24")
	victim := domainSHA256("198.51.100.0/24")
	if !bytes.Equal(got, appended) {
		t.Fatalf("stored hash is not the proxy-appended /24")
	}
	if bytes.Equal(got, victim) {
		t.Fatal("stored hash is the leftmost client-supplied victim /24")
	}
}

// TestReviewR15_EndpointWriteRequiresTheNodesLeaf is the authenticator gate.
// A request with no verified leaf, a revoked leaf, and another node's leaf
// do not insert node_identity_endpoints for the victim node.
func TestReviewR15_EndpointWriteRequiresTheNodesLeaf(t *testing.T) {
	f := setup(t)
	reviewRotateLease(t, f)
	_, victim, victimCert := f.enroll(f.create(nil))
	_, other, otherCert := f.enroll(f.create(nil))
	if reviewEndpointCount(t, f, victim.NodeIncarnationId) != 0 {
		t.Fatal("enroll wrote an endpoint row")
	}

	r := httptest.NewRequest(http.MethodPost, "https://control.example.invalid/api/v1/agent/lease-renewals", bytes.NewReader([]byte(`{}`)))
	r.RemoteAddr = reviewDomainY
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.agent.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no leaf: status %d, want 401", w.Code)
	}
	if reviewEndpointCount(t, f, victim.NodeIncarnationId) != 0 {
		t.Fatal("a request with no leaf wrote the victim endpoint")
	}

	// Another node's leaf authenticates as that node. The body names the victim.
	spoof := api.LeaseRenewalRequest{
		RenewalRequestId: uuid.New(), NodeId: victim.NodeId, NodeIncarnationId: victim.NodeIncarnationId,
		CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	w = f.callFrom("POST", "/api/v1/agent/lease-renewals", spoof, otherCert, reviewDomainY, nil)
	if w.Code != http.StatusOK && w.Code != http.StatusConflict && w.Code != http.StatusForbidden {
		// RenewTx rejects an incarnation that is not the authenticated one.
		// Either a 4xx after the other node's observe, or 200 for the other node, is acceptable.
		// A 200 would be the other node's lease. The victim row must stay empty.
		t.Logf("other-leaf lease status %d: %s", w.Code, w.Body.String())
	}
	if reviewEndpointCount(t, f, victim.NodeIncarnationId) != 0 {
		t.Fatal("another node's leaf wrote the victim endpoint")
	}
	// A body that names a different node fails inside the same transaction as
	// ObserveEndpointTx, so that request does not commit either row.
	if w.Code == http.StatusOK {
		if reviewEndpointCount(t, f, other.NodeIncarnationId) != 1 {
			t.Fatalf("other node endpoint rows = %d, want 1 (the cert, not the body, selects the row)", reviewEndpointCount(t, f, other.NodeIncarnationId))
		}
	} else if n := reviewEndpointCount(t, f, other.NodeIncarnationId); n != 0 {
		t.Fatalf("rejected other-leaf request status %d still wrote %d endpoint rows", w.Code, n)
	} else {
		t.Logf("other-leaf request naming the victim returned %d and committed no endpoint row", w.Code)
	}
	reviewRenew(t, f, other, otherCert, reviewDomainY)
	if reviewEndpointCount(t, f, other.NodeIncarnationId) != 1 || reviewEndpointCount(t, f, victim.NodeIncarnationId) != 0 {
		t.Fatal("the other node's own renewal did not write only its own row")
	}

	reviewRenew(t, f, victim, victimCert, reviewDomainX)
	if reviewEndpointCount(t, f, victim.NodeIncarnationId) != 1 {
		t.Fatal("victim leaf did not write its own row")
	}
	before := reviewEndpointCount(t, f, victim.NodeIncarnationId)
	detail := f.call("GET", "/api/v1/admin/nodes/"+victim.NodeId.String(), nil, nil, nil)
	rev := f.call("POST", "/api/v1/admin/nodes/"+victim.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY review revoke"}, nil, map[string]string{
		"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag"),
	})
	if rev.Code != http.StatusAccepted {
		t.Fatalf("revoke %d %s", rev.Code, rev.Body.String())
	}
	w = f.callFrom("POST", "/api/v1/agent/lease-renewals", api.LeaseRenewalRequest{
		RenewalRequestId: uuid.New(), NodeId: victim.NodeId, NodeIncarnationId: victim.NodeIncarnationId,
		CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}, victimCert, reviewDomainY, nil)
	if w.Code == http.StatusOK {
		t.Fatalf("revoked leaf renewed: %s", w.Body.String())
	}
	if reviewEndpointCount(t, f, victim.NodeIncarnationId) != before {
		t.Fatalf("revoked leaf changed endpoint rows from %d to %d", before, reviewEndpointCount(t, f, victim.NodeIncarnationId))
	}
}

// TestReviewR15_FrozenIncarnationStillReceivesAGrantLease alternates two
// domains inside IdentityActiveWindow. The detecting renewal runs
// ObserveEndpointTx and then RenewTx in one transaction. Freeze does not
// change provisioning_state, so a published ready exit still receives member
// grants for DefaultMaxOfflineLeaseSeconds. The following observation forces
// provisioning and safety to failed and does not shorten that lease.
func TestReviewR15_FrozenIncarnationStillReceivesAGrantLease(t *testing.T) {
	f, enrolled, cert, _, _ := servingFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.r.Enrollment.SetClock(func() time.Time { return now })

	first := reviewRenew(t, f, enrolled, cert, reviewDomainX)
	if len(first.Signed.Members) == 0 {
		t.Fatalf("setup: published ready exit minted %d member grants; the freeze comparison needs a grant-bearing lease", len(first.Signed.Members))
	}
	firstTTL := reviewLeaseTTL(first)
	if firstTTL < 23*time.Hour {
		t.Fatalf("setup: first lease ttl %s, want about %s", firstTTL, time.Duration(leases.DefaultMaxOfflineLeaseSeconds)*time.Second)
	}

	second := reviewRenew(t, f, enrolled, cert, reviewDomainY)
	if reviewFrozen(t, f, enrolled.NodeIncarnationId) {
		t.Fatal("a single X to Y switch froze the incarnation; that is roaming, not a conflict")
	}
	if len(second.Signed.Members) == 0 {
		t.Fatal("roaming renewal dropped member grants")
	}

	detecting := reviewRenew(t, f, enrolled, cert, reviewDomainX)
	if !reviewFrozen(t, f, enrolled.NodeIncarnationId) {
		t.Fatal("X to Y to X inside the window did not freeze")
	}
	if reviewConflictCount(t, f, enrolled.NodeIncarnationId) != 1 {
		t.Fatalf("conflicts = %d, want 1", reviewConflictCount(t, f, enrolled.NodeIncarnationId))
	}
	detectTTL := reviewLeaseTTL(detecting)
	var storedExpiry time.Time
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT expires_at FROM serving_leases WHERE node_id=$1 AND issued_at=$2`, enrolled.NodeId, detecting.Signed.IssuedAt).Scan(&storedExpiry); err != nil {
		t.Fatal(err)
	}
	if storedExpiry.Sub(detecting.Signed.ExpiresAt).Abs() > time.Microsecond {
		t.Fatalf("stored expiry %s, envelope %s", storedExpiry, detecting.Signed.ExpiresAt)
	}

	reviewSubmitReady(t, f, enrolled, cert)
	var provisioning, safety string
	var forwarding bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT provisioning_state, safety_status, member_forwarding_observed FROM nodes WHERE id=$1`, enrolled.NodeId).Scan(&provisioning, &safety, &forwarding); err != nil {
		t.Fatal(err)
	}
	var expiryAfter time.Time
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT expires_at FROM serving_leases WHERE node_id=$1 AND issued_at=$2`, enrolled.NodeId, detecting.Signed.IssuedAt).Scan(&expiryAfter); err != nil {
		t.Fatal(err)
	}

	later := reviewRenew(t, f, enrolled, cert, reviewDomainX)
	laterTTL := reviewLeaseTTL(later)

	if len(detecting.Signed.Members) > 0 && detectTTL >= 23*time.Hour && expiryAfter.Equal(storedExpiry) {
		t.Fatalf("TODO(review): the renewal that froze incarnation %s still minted %d member grants for %s (DefaultMaxOfflineLeaseSeconds=%d). ObserveEndpointTx runs before RenewTx and recordIdentityConflict does not change provisioning_state, so includeMembers stayed true. The next observation set provisioning=%s safety=%s forwarding=%v and left serving_leases.expires_at at %s. A later renewal ttl is %s with %d members. The agent renews a grant-bearing lease only inside RenewLeadSeconds (default 120) of its deadline, so this lease keeps member forwarding until about %s after the detecting renewal. Recovery is revoke and re-enroll.",
			enrolled.NodeIncarnationId, len(detecting.Signed.Members), detectTTL, leases.DefaultMaxOfflineLeaseSeconds,
			provisioning, safety, forwarding, expiryAfter.UTC().Format(time.RFC3339), laterTTL, len(later.Signed.Members),
			detectTTL-120*time.Second)
	}
	if provisioning != "failed" || safety != "failed" || forwarding {
		t.Fatalf("frozen observation left provisioning=%s safety=%s forwarding=%v", provisioning, safety, forwarding)
	}
	if len(detecting.Signed.Members) != 0 || detectTTL > time.Minute {
		t.Fatalf("detecting lease members=%d ttl=%s; expected the grantless backstop once this stops being a finding", len(detecting.Signed.Members), detectTTL)
	}
}

// TestReviewR15_MultihomedNodeFreezesItself is the same interleaving path a
// clone uses. Two uplinks inside IdentityActiveWindow freeze the incarnation.
// The operator cost is revoke and re-enroll. This branch's nodes guide calls
// that a cloned private key.
func TestReviewR15_MultihomedNodeFreezesItself(t *testing.T) {
	f := setup(t)
	reviewRotateLease(t, f)
	_, enrolled, cert := f.enroll(f.create(nil))
	now := time.Now().UTC().Truncate(time.Second)
	f.r.Enrollment.SetClock(func() time.Time { return now })
	reviewRenew(t, f, enrolled, cert, reviewDomainX)
	reviewRenew(t, f, enrolled, cert, reviewDomainY)
	reviewRenew(t, f, enrolled, cert, reviewDomainX)
	if !reviewFrozen(t, f, enrolled.NodeIncarnationId) {
		t.Fatal("alternating two uplinks inside the window did not freeze")
	}
	if enrollment.DefaultIdentityActiveWindow != 120*time.Second {
		t.Fatalf("window = %s", enrollment.DefaultIdentityActiveWindow)
	}
	detail := decode[api.NodeDetail](t, f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil), 200)
	conflict, err := detail.IdentityConflict.Get()
	if err != nil || !conflict.MappingFrozen || conflict.ConflictCount != 1 {
		t.Fatalf("operator identity_conflict = %+v err=%v", conflict, err)
	}
}

// TestReviewR15_QuietWindowPromotesWithoutAnAudit is roaming. Y is promoted
// after X has been quiet for the window. No conflict row and no resource
// version bump. Months of daily roaming stay one row per incarnation. Revoke
// leaves that row on the retired incarnation.
func TestReviewR15_QuietWindowPromotesWithoutAnAudit(t *testing.T) {
	f := setup(t)
	reviewRotateLease(t, f)
	_, enrolled, cert := f.enroll(f.create(nil))
	now := time.Now().UTC().Truncate(time.Second)
	f.r.Enrollment.SetClock(func() time.Time { return now })
	reviewRenew(t, f, enrolled, cert, reviewDomainX)
	var version int64
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT resource_version FROM nodes WHERE id=$1`, enrolled.NodeId).Scan(&version); err != nil {
		t.Fatal(err)
	}
	now = now.Add(enrollment.DefaultIdentityActiveWindow + time.Second)
	f.r.Enrollment.SetClock(func() time.Time { return now })
	reviewRenew(t, f, enrolled, cert, reviewDomainY)
	var got []byte
	var versionAfter int64
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT endpoint_hash, (SELECT resource_version FROM nodes WHERE id=$2) FROM node_identity_endpoints WHERE incarnation_id=$1`, enrolled.NodeIncarnationId, enrolled.NodeId).Scan(&got, &versionAfter); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, domainSHA256(reviewPrefixY)) {
		t.Fatal("quiet window did not promote Y")
	}
	if reviewConflictCount(t, f, enrolled.NodeIncarnationId) != 0 || reviewFrozen(t, f, enrolled.NodeIncarnationId) {
		t.Fatal("roaming promotion wrote a conflict or froze")
	}
	var audits int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='node.identity.conflict' AND resource=$1`, "node:"+enrolled.NodeId.String()).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 0 {
		t.Fatalf("promotion wrote %d node.identity.conflict audits", audits)
	}
	if versionAfter != version {
		t.Fatalf("promotion bumped resource_version from %d to %d", version, versionAfter)
	}

	// LeaseRate's default burst is 5. Two further promotions stay inside it
	// and still cross the window twice. The row count, not the sample count,
	// is the bound.
	for i := 0; i < 2; i++ {
		now = now.Add(enrollment.DefaultIdentityActiveWindow + time.Second)
		f.r.Enrollment.SetClock(func() time.Time { return now })
		remote := reviewDomainX
		if i%2 == 0 {
			remote = reviewDomainY
		}
		reviewRenew(t, f, enrolled, cert, remote)
	}
	if reviewEndpointCount(t, f, enrolled.NodeIncarnationId) != 1 {
		t.Fatalf("daily roaming grew endpoint rows to %d, want 1", reviewEndpointCount(t, f, enrolled.NodeIncarnationId))
	}

	detail := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	rev := f.call("POST", "/api/v1/admin/nodes/"+enrolled.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY roaming revoke"}, nil, map[string]string{
		"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag"),
	})
	if rev.Code != http.StatusAccepted {
		t.Fatalf("revoke %d %s", rev.Code, rev.Body.String())
	}
	if reviewEndpointCount(t, f, enrolled.NodeIncarnationId) != 1 {
		t.Fatal("revoke deleted the retired incarnation's endpoint row")
	}
	_, replaced, _ := f.enroll(f.create(&enrolled.NodeId))
	if replaced.NodeIncarnationId == enrolled.NodeIncarnationId {
		t.Fatal("re-enroll kept the incarnation")
	}
	if reviewEndpointCount(t, f, replaced.NodeIncarnationId) != 0 {
		t.Fatal("the new incarnation inherited an endpoint row")
	}
	var inherited bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT coalesce(mapping_frozen, false) FROM node_identity_endpoints WHERE incarnation_id=$1`, replaced.NodeIncarnationId).Scan(&inherited); err == nil && inherited {
		t.Fatal("new incarnation is frozen")
	}
}

func reviewSubmitReady(t *testing.T, f *fixture, enrolled api.EnrollResponse, cert *x509.Certificate) {
	t.Helper()
	state := decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), 200)
	raw, err := os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	report.NodeId, report.NodeIncarnationId, report.ReportId = enrolled.NodeId, enrolled.NodeIncarnationId, uuid.New()
	report.ObservedAt = time.Now().UTC()
	report.AppliedConfigGeneration.Set(state.Signed.ConfigGeneration)
	report.CurrentReleaseId.SetNull()
	report.LastGoodReleaseId.SetNull()
	report.Lease.LeaseId.SetNull()
	report.Netbird.PeerId.SetNull()
	report.MemberForwardingEnabled = true
	report.ProvisioningState = api.ProvisioningStateReady
	report.SafetyStatus = api.SafetyStatusPassed
	for i := range report.Checks {
		report.Checks[i].Status = api.HealthCheckStatusPassed
		report.Checks[i].ObservedAt = report.ObservedAt
	}
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
}
