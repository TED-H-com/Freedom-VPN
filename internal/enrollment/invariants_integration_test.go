//go:build integration

package enrollment_test

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestF15_InvalidIdentifiersCannotRevokeOrPersist(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	_, _, enrolled, cert := enroll(t, f)
	unused := create(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	version := decstr.U64(2)
	for _, principal := range []string{"", " \t\n", "\u2003", "invalid\x00actor", "invalid\xffactor"} {
		cmd := enrollment.Command{Principal: principal, Key: uuid.New()}
		_, err = f.s.CreateEnrollment(ctx, cmd, api.EnrollmentCreateRequest{Delivery: "installer"})
		expectCode(t, err, apierr.ValidationFailed)
		_, err = f.s.RevokeEnrollment(ctx, cmd, unused.Enrollment.EnrollmentId, "TEST ONLY invalid actor")
		expectCode(t, err, apierr.ValidationFailed)
		_, err = f.s.RevokeNode(ctx, cmd, id.NodeID, &version, "TEST ONLY invalid actor")
		expectCode(t, err, apierr.ValidationFailed)
		// The revocation journal is outside the DB transaction. Validation must
		// precede it, rather than relying on AppendAudit to roll back the DB.
		if _, err = f.auth.Authenticate(ctx, cert); err != nil {
			t.Fatal("invalid actor permanently denied the valid identity", err)
		}
		err = f.s.StoreBootstrap(ctx, id, enrolled.OperationID, principal, "TEST ONLY setup value", time.Now().Add(time.Minute))
		expectCode(t, err, apierr.ValidationFailed)
		_, err = f.s.ObserveEndpoint(ctx, cert, principal, time.Minute)
		expectCode(t, err, apierr.ValidationFailed)
	}
	state, err := f.s.GetEnrollment(ctx, unused.Enrollment.EnrollmentId)
	if err != nil || state.State != "pending" {
		t.Fatal("invalid command changed enrollment", err)
	}
	if count(t, f, "idempotency_requests") != 2 || count(t, f, "operations") != 1 || count(t, f, "outbox_events") != 1 || count(t, f, "node_bootstrap_secrets") != 0 || count(t, f, "node_identity_endpoints") != 0 {
		t.Fatal("invalid identifiers left persisted work or reservations")
	}
}

func TestF06F10F13_ServiceWritesPreserveCommittedEvidence(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	created, request, enrolled, cert := enroll(t, f)
	certificateSnapshot := func() string {
		t.Helper()
		var snapshot string
		// Only the separately tracked effective overlap may change on renewal.
		if err := f.pool.QueryRow(ctx, `SELECT (to_jsonb(c)-'usable_until')::text FROM node_certificates c WHERE serial=$1`, enrolled.Response.CertificateSerial).Scan(&snapshot); err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	originalCertificate := certificateSnapshot()
	_, csr := pkitest.CSR(t)
	renewed, err := f.auth.Renew(ctx, cert, api.CertificateRenewRequest{NodeId: enrolled.Response.NodeId, NodeIncarnationId: enrolled.Response.NodeIncarnationId, RenewalRequestId: uuid.New(), CsrPem: csr})
	if err != nil {
		t.Fatal(err)
	}
	if certificateSnapshot() != originalCertificate {
		t.Fatal("renewal rewrote original certificate evidence")
	}
	// Simulate worker read-back using explicitly TEST ONLY evidence. Replays
	// and later node revocation must leave the succeeded provision row frozen.
	exec(t, f, `UPDATE operations SET status='running' WHERE id=$1`, enrolled.OperationID)
	exec(t, f, `UPDATE operations SET status='succeeded',completed_at=clock_timestamp(),
 steps=(SELECT jsonb_object_agg(key,value || '{"status":"succeeded","observed_ref":"test_readback"}'::jsonb) FROM jsonb_each(steps)),
 observed_refs='{"test_readback":{"label":"TEST ONLY worker evidence"}}' WHERE id=$1`, enrolled.OperationID)
	var frozenOperation string
	if err = f.pool.QueryRow(ctx, `SELECT to_jsonb(o)::text FROM operations o WHERE id=$1`, enrolled.OperationID).Scan(&frozenOperation); err != nil {
		t.Fatal(err)
	}
	replayed, err := f.s.Enroll(ctx, nullableValue(created.Token), request)
	if err != nil || !replayed.Replay || replayed.OperationID != enrolled.OperationID {
		t.Fatal("completed provisioning operation could not be replayed", err)
	}
	// Earlier certificate-level revocation must survive later node revocation.
	exec(t, f, `UPDATE node_certificates SET revoked_at=clock_timestamp() WHERE serial=$1`, enrolled.Response.CertificateSerial)
	revokedCertificate := certificateSnapshot()
	exec(t, f, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, enrolled.Response.NodeIncarnationId)
	var revokedIncarnation string
	if err = f.pool.QueryRow(ctx, `SELECT to_jsonb(i)::text FROM node_incarnations i WHERE id=$1`, enrolled.Response.NodeIncarnationId).Scan(&revokedIncarnation); err != nil {
		t.Fatal(err)
	}

	unused := create(t, f)
	revokeEnrollmentCommand := command()
	if _, err = f.s.RevokeEnrollment(ctx, revokeEnrollmentCommand, unused.Enrollment.EnrollmentId, "TEST ONLY unused token"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.RevokeEnrollment(ctx, revokeEnrollmentCommand, unused.Enrollment.EnrollmentId, "TEST ONLY unused token"); err != nil {
		t.Fatal(err)
	}
	version := decstr.U64(2)
	revokeNodeCommand := command()
	result, err := f.s.RevokeNode(ctx, revokeNodeCommand, enrolled.Response.NodeId, &version, "TEST ONLY identity withdrawal")
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceVersion != 3 {
		t.Fatal("node update did not advance resource version")
	}
	if _, err = f.s.RevokeNode(ctx, revokeNodeCommand, enrolled.Response.NodeId, &version, "TEST ONLY identity withdrawal"); err != nil {
		t.Fatal(err)
	}
	if certificateSnapshot() != revokedCertificate {
		t.Fatal("node revocation rewrote historical certificate or revocation timestamp")
	}
	var afterIncarnation string
	if err = f.pool.QueryRow(ctx, `SELECT to_jsonb(i)::text FROM node_incarnations i WHERE id=$1`, enrolled.Response.NodeIncarnationId).Scan(&afterIncarnation); err != nil || afterIncarnation != revokedIncarnation {
		t.Fatal("node revocation rewrote incarnation identity or revocation timestamp", err)
	}
	_, err = f.auth.Authenticate(ctx, pkitest.Certificate(t, renewed.CertificatePem))
	expectCode(t, err, apierr.IdentityRevoked)
	var afterOperation string
	if err = f.pool.QueryRow(ctx, `SELECT to_jsonb(o)::text FROM operations o WHERE id=$1`, enrolled.OperationID).Scan(&afterOperation); err != nil || afterOperation != frozenOperation {
		t.Fatal("terminal provisioning evidence changed", err)
	}
	var complete, pending bool
	if err = f.pool.QueryRow(ctx, `SELECT count(*)=4 AND bool_and(state='completed' AND resource_id IS NOT NULL AND response_status IS NOT NULL)
 AND count(*) FILTER(WHERE response_status=201)=2 AND count(*) FILTER(WHERE response_status=200)=1
 AND count(*) FILTER(WHERE response_status=202 AND operation_id=$1)=1 FROM idempotency_requests`, result.OperationID).Scan(&complete); err != nil || !complete {
		t.Fatal("command did not commit complete replay references/status", err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT status='pending' AND steps<>'{}'::jsonb
	AND NOT EXISTS(SELECT 1 FROM jsonb_each(steps) WHERE value->>'required' IS DISTINCT FROM 'true' OR value->>'status' IS DISTINCT FROM 'pending')
 FROM operations WHERE id=$1`, result.OperationID).Scan(&pending); err != nil || !pending {
		t.Fatal("withdrawal operation does not start pending with required steps", err)
	}
}
