//go:build integration

package devices

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"testing"
)

func TestMEM02_D029DeviceReplayOperationsAndAudit(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	c, err := s.AuthorizeDevice(ctx, deviceCommand("system", member), peers[0])
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.AuthorizeDevice(ctx, deviceCommand("system", member), peers[0])
	if err != nil || again.OperationID == nil || *again.OperationID != *c.OperationID {
		t.Fatal(again, err)
	}
	p := auth.Principal{Role: auth.Member, ID: member}
	read := func(id uuid.UUID) api.Operation {
		t.Helper()
		op, e := (operations.Store{Pool: s.Pool}).Get(ctx, id, p)
		if e != nil {
			t.Fatal(e)
		}
		if op.Resource.Type != api.ResourceTypeMemberDevice || !op.Kind.Valid() {
			t.Fatal(op)
		}
		return op
	}
	if op := read(*c.OperationID); op.Kind != api.OperationKindMemberDeviceAuthorize {
		t.Fatal(op)
	}
	adapter := s.Adapter.(*fakeAdapter)
	adapter.ignoreWrites = true
	if err = s.SyncDevice(ctx, c.DeviceID); err == nil {
		t.Fatal("unobserved sync succeeded")
	}
	op := read(*c.OperationID)
	if op.Kind != api.OperationKindMemberDeviceAuthorize || op.Error.IsNull() || op.Status != api.OperationStatusFailed {
		t.Fatal("failed step invisible", op)
	}
	var code, message string
	if err = s.Pool.QueryRow(ctx, `SELECT error_code,error_message FROM operations WHERE id=$1`, c.OperationID).Scan(&code, &message); err != nil || code != "DEPENDENCY_UNAVAILABLE" || message == "" {
		t.Fatal(code, message, err)
	}
	adapter.ignoreWrites = false
	if err = s.SyncDevice(ctx, c.DeviceID); err != nil {
		t.Fatal(err)
	}
	op = read(*c.OperationID)
	if op.Kind != api.OperationKindMemberDeviceAuthorize || op.Status != api.OperationStatusRunning || !op.Error.IsNull() {
		t.Fatal(op)
	}
	revoked, err := s.RevokeDevice(ctx, deviceCommand("admin", uuid.New()), c.DeviceID)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := s.RevokeDevice(ctx, deviceCommand("member", member), c.DeviceID)
	if err != nil || repeated.OperationID == nil || *repeated.OperationID != *revoked.OperationID {
		t.Fatal(repeated, err)
	}
	if op = read(*revoked.OperationID); op.Kind != api.OperationKindMemberDeviceRevoke {
		t.Fatal(op)
	}
	// Legacy closed rows have no prior device operation: create a pending
	// observation/withdrawal operation rather than an empty 202 or a false success.
	legacy := uuid.New()
	if _, err = s.Pool.Exec(ctx, `INSERT INTO member_devices(id,member_id,deployment_id,peer_id,status,closed_at) VALUES($1,$2,'test','legacy','closed',clock_timestamp())`, legacy, member); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.RevokeDevice(ctx, deviceCommand("member", member), legacy)
	if err != nil || recovered.OperationID == nil {
		t.Fatal(recovered, err)
	}
	if op = read(*recovered.OperationID); op.Status != api.OperationStatusPending {
		t.Fatal(op)
	}
	store := audit.Store{Pool: s.Pool}
	page, err := store.List(ctx, api.ListAuditEventsParams{})
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&count); err != nil || len(page.Items) != count || store.AnomalyCount() != 0 {
		t.Fatal(count, len(page.Items), store.AnomalyCount(), err)
	}
}
