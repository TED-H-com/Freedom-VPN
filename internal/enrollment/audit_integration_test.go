//go:build integration

package enrollment_test

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestD029_EnrollmentAuditAndOperationsRoundTrip(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	created, _, enrolled, cert := enroll(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.s.StoreBootstrap(ctx, id, enrolled.OperationID, "test-audit-key", "TEST ONLY audit bootstrap", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.TakeBootstrap(ctx, cert); err != nil {
		t.Fatal(err)
	}
	unused := create(t, f)
	if _, err = f.s.RevokeEnrollment(ctx, command(), unused.Enrollment.EnrollmentId, "TEST ONLY unused enrollment"); err != nil {
		t.Fatal(err)
	}
	version := decstr.U64(2)
	revocation, err := f.s.RevokeNode(ctx, command(), id.NodeID, &version, "TEST ONLY revoke")
	if err != nil {
		t.Fatal(err)
	}
	retirement, err := f.s.RetireNode(ctx, command(), id.NodeID, &revocation.ResourceVersion, "TEST ONLY retire")
	if err != nil {
		t.Fatal(err)
	}
	store := audit.Store{Pool: f.pool}
	page, err := store.List(ctx, api.ListAuditEventsParams{})
	if err != nil || len(page.Items) != 7 || store.AnomalyCount() != 0 || count(t, f, "audit_events") != len(page.Items) {
		t.Fatalf("audit round-trip: rows=%d anomalies=%d err=%v", len(page.Items), store.AnomalyCount(), err)
	}
	resources := map[string]api.ResourceRef{
		"enrollment.consumed": {Type: api.ResourceTypeEnrollment, Id: created.Enrollment.EnrollmentId},
		"enrollment.revoked":  {Type: api.ResourceTypeEnrollment, Id: unused.Enrollment.EnrollmentId},
		"bootstrap.claimed":   {Type: api.ResourceTypeNode, Id: id.NodeID},
		"node.revoked":        {Type: api.ResourceTypeNode, Id: id.NodeID},
		"node.retired":        {Type: api.ResourceTypeNode, Id: id.NodeID},
	}
	for _, event := range page.Items {
		if nullableValue(event.Actor.Id) == uuid.Nil || event.Result != api.AuditEventResultSucceeded || nullableValue(event.RequestId) == uuid.Nil {
			t.Fatal("audit identity/result missing")
		}
		if event.Action == "enrollment.created" {
			if event.Resource.Type != api.ResourceTypeEnrollment || (event.Resource.Id != created.Enrollment.EnrollmentId && event.Resource.Id != unused.Enrollment.EnrollmentId) {
				t.Fatal("wrong enrollment creation resource")
			}
		} else if event.Resource != resources[event.Action] {
			t.Fatal("wrong audit resource", event.Action)
		}
		if event.Action == "enrollment.consumed" || event.Action == "bootstrap.claimed" {
			if event.Actor.Type != api.AuditEventActorTypeNode || nullableValue(event.Actor.Id) != id.NodeID {
				t.Fatal("node actor lost stable UUID")
			}
		} else if event.Actor.Type != api.AuditEventActorTypeAdmin {
			t.Fatal("wrong administrator actor type")
		}
	}
	nodeType := api.ResourceTypeNode
	filtered, err := store.List(ctx, api.ListAuditEventsParams{ResourceType: &nodeType, ResourceId: &id.NodeID})
	if err != nil || len(filtered.Items) != 3 || store.AnomalyCount() != 0 {
		t.Fatal("node audit SQL filters lost bootstrap/revocation/retirement", err)
	}
	ops := operations.Store{Pool: f.pool}
	for operation, kind := range map[uuid.UUID]api.OperationKind{enrolled.OperationID: api.OperationKindNodeEnroll, revocation.OperationID: api.OperationKindNodeRevoke, retirement.OperationID: api.OperationKindNodeRetire} {
		op, err := ops.Get(ctx, operation, auth.Principal{Role: auth.Admin, ID: uuid.New()})
		if err != nil || op.Kind != kind || !op.Kind.Valid() || op.Resource.Type != api.ResourceTypeNode || op.Resource.Id != id.NodeID || op.Status != api.OperationStatusPending || op.RequestId == uuid.Nil {
			t.Fatal("operation does not round-trip through the contract reader", err)
		}
		for _, event := range page.Items {
			if event.Action != "bootstrap.claimed" && !event.OperationId.IsNull() && nullableValue(event.OperationId) == operation && nullableValue(event.RequestId) != op.RequestId {
				t.Fatal("operation and command audit lost request correlation")
			}
		}
	}
}

func TestD029_InvalidAdministratorCannotPersistRevocation(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	_, _, enrolled, cert := enroll(t, f)
	version := decstr.U64(2)
	for _, actor := range []string{"test-admin", "admin:" + uuid.Nil.String(), "member:" + uuid.NewString(), "node:" + uuid.NewString()} {
		cmd := command()
		cmd.Principal = actor
		_, err := f.s.RevokeNode(ctx, cmd, enrolled.Response.NodeId, &version, "TEST ONLY invalid principal")
		expectCode(t, err, apierr.ValidationFailed)
		if _, err = f.auth.Authenticate(ctx, cert); err != nil {
			t.Fatal("invalid principal persisted a revocation tombstone", err)
		}
	}
}
