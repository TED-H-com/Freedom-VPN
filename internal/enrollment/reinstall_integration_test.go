//go:build integration

package enrollment_test

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestAT07_AuthorizedReinstallPreservesNodeAndRevokedHistory(t *testing.T) {
	f := setup(t) // Includes real migration 00111; no test-only DDL.
	ctx := t.Context()
	missing := uuid.New()
	_, err := f.s.CreateEnrollment(ctx, command(), api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &missing})
	expectCode(t, err, apierr.ResourceNotFound)
	old, oldRequest, initial, oldCert := enroll(t, f)
	node := initial.Response.NodeId
	reinstall := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &node}
	_, err = f.s.CreateEnrollment(ctx, command(), reinstall)
	expectCode(t, err, apierr.NodeIncarnationActive)
	version := decstr.U64(2)
	if _, err = f.s.RevokeNode(ctx, command(), node, &version, "TEST ONLY authorized reinstall"); err != nil {
		t.Fatal(err)
	}
	var before string
	if err = f.pool.QueryRow(ctx, `SELECT to_jsonb(i)::text FROM node_incarnations i WHERE id=$1`, initial.Response.NodeIncarnationId).Scan(&before); err != nil {
		t.Fatal(err)
	}
	cmd := command()
	created, err := f.s.CreateEnrollment(ctx, cmd, reinstall)
	if err != nil || created.Enrollment.NodeId != node {
		t.Fatal("reinstall did not bind to existing node", err)
	}
	replay, err := f.s.CreateEnrollment(ctx, cmd, reinstall)
	if err != nil || replay.Enrollment.EnrollmentId != created.Enrollment.EnrollmentId || replay.SecretAvailable || !replay.Token.IsNull() {
		t.Fatal("reinstall creation replay changed binding or exposed token", err)
	}
	var stillRevoked bool
	if err = f.pool.QueryRow(ctx, `SELECT lifecycle_state='revoked' AND current_incarnation_id=$2 AND resource_version=3 FROM nodes WHERE id=$1`, node, initial.Response.NodeIncarnationId).Scan(&stillRevoked); err != nil || !stillRevoked {
		t.Fatal("creating enrollment cleared the old lifecycle", err)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE nodes SET lifecycle_state='registered',resource_version=resource_version+1 WHERE id=$1`, node); err == nil {
		t.Fatal("bare lifecycle update revived revoked node")
	}
	request, _ := pkitest.Request(t)
	replacement, err := f.s.Enroll(ctx, nullableValue(created.Token), request)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Response.NodeId != node || replacement.Response.NodeIncarnationId == initial.Response.NodeIncarnationId || replacement.Response.LifecycleState != api.LifecycleStateRegistered {
		t.Fatal("reinstall changed stable node or reused old incarnation")
	}
	if count(t, f, "nodes") != 1 || count(t, f, "exits") != 1 || count(t, f, "node_incarnations") != 2 || count(t, f, "node_certificates") != 2 {
		t.Fatal("reinstall duplicated registry or lost identity history")
	}
	newCert := pkitest.Certificate(t, replacement.Response.CertificatePem)
	principal, err := f.auth.AuthenticateAgent(ctx, newCert)
	if err != nil || principal.NodeID != node || principal.IncarnationID != request.NodeIncarnationId {
		t.Fatal("new incarnation cannot use control authentication", err)
	}
	if _, err = f.auth.Certificate(auth.WithPrincipal(ctx, principal)); err != nil {
		t.Fatal("new certificate unavailable through framework adapter", err)
	}
	_, err = f.auth.Authenticate(ctx, oldCert)
	expectCode(t, err, apierr.IdentityRevoked)
	_, err = f.s.Enroll(ctx, nullableValue(old.Token), oldRequest)
	expectCode(t, err, apierr.IdentityRevoked)
	var after string
	if err = f.pool.QueryRow(ctx, `SELECT to_jsonb(i)::text FROM node_incarnations i WHERE id=$1`, initial.Response.NodeIncarnationId).Scan(&after); err != nil || after != before {
		t.Fatal("reinstall rewrote revoked incarnation evidence", err)
	}
	repeated, err := f.s.Enroll(ctx, nullableValue(created.Token), request)
	if err != nil || !repeated.Replay || repeated.Response != replacement.Response || repeated.OperationID != replacement.OperationID {
		t.Fatal("lost reinstall response duplicated work", err)
	}
	version = 4
	if _, err = f.s.RetireNode(ctx, command(), node, &version, "TEST ONLY retirement"); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.CreateEnrollment(ctx, command(), reinstall)
	expectCode(t, err, apierr.NodeRetired)
	_, err = f.auth.Authenticate(ctx, newCert)
	expectCode(t, err, apierr.IdentityRevoked)
	if _, err = f.pool.Exec(ctx, `UPDATE nodes SET lifecycle_state='revoked',resource_version=resource_version+1 WHERE id=$1`, node); err == nil {
		t.Fatal("retirement could be routed through revoked to permit reinstall")
	}
}

func TestAT07_ReinstallRechecksIncarnationAfterTokenCreation(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	_, _, initial, oldCert := enroll(t, f)
	// Incarnation revocation can precede a registry lifecycle update. The
	// authorization condition is the actual identity's revocation evidence.
	exec(t, f, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, initial.Response.NodeIncarnationId)
	reinstall := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &initial.Response.NodeId}
	first, err := f.s.CreateEnrollment(ctx, command(), reinstall)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.s.CreateEnrollment(ctx, command(), reinstall)
	if err != nil {
		t.Fatal(err)
	}
	request, _ := pkitest.Request(t)
	replacement, err := f.s.Enroll(ctx, nullableValue(first.Token), request)
	if err != nil || replacement.Response.NodeId != initial.Response.NodeId {
		t.Fatal("revoked current incarnation could not be replaced", err)
	}
	otherRequest, _ := pkitest.Request(t)
	_, err = f.s.Enroll(ctx, nullableValue(second.Token), otherRequest)
	expectCode(t, err, apierr.NodeIncarnationActive)
	if count(t, f, "nodes") != 1 || count(t, f, "node_incarnations") != 2 {
		t.Fatal("second authorized token replaced an active incarnation")
	}
	_, err = f.auth.Authenticate(ctx, oldCert)
	expectCode(t, err, apierr.IdentityRevoked)
}
