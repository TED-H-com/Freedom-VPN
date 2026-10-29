//go:build integration

package members

import (
	"context"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestMEM01_VersionsPaginationSuspensionAndTombstone(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	engine, _ := entitlement.NewStore(entitlement.TestConfig())
	s := Service{Pool: pool, Entitlements: engine}
	actor := entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, id := range ids {
		if _, err := pool.Exec(ctx, `INSERT INTO members(id) VALUES($1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.List(ctx, actor, "", 2)
	if err != nil || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatal(first, err)
	}
	second, err := s.List(ctx, actor, first.NextCursor, 2)
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatal(second, err)
	}
	c := entitlement.Command{Actor: actor, Key: uuid.New()}
	name := "TEST ONLY display"
	version := decstr.U64(1)
	if _, err = s.Patch(ctx, c, ids[0], nil, api.MemberPatch{DisplayName: &name}); !errors.Is(err, entitlement.ErrPreconditionRequired) {
		t.Fatal(err)
	}
	if _, err = s.Patch(ctx, c, ids[0], &version, api.MemberPatch{DisplayName: &name}); err != nil {
		t.Fatal(err)
	}
	if replay, err := s.Patch(ctx, c, ids[0], &version, api.MemberPatch{DisplayName: &name}); err != nil || !replay.Replayed {
		t.Fatal(replay, err)
	}
	c.Key = uuid.New()
	if _, err = s.Patch(ctx, c, ids[0], &version, api.MemberPatch{DisplayName: &name}); !errors.Is(err, db.ErrVersionConflict) {
		t.Fatal(err)
	}
	member, err := s.Get(ctx, actor, ids[0])
	if err != nil || member.ResourceVersion != 2 {
		t.Fatal(member, err)
	}
	version = member.ResourceVersion
	suspended, reason := true, "TEST ONLY suspension"
	c.Key = uuid.New()
	if _, err = s.Patch(ctx, c, ids[0], &version, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason}); err != nil {
		t.Fatal(err)
	}
	member, err = s.Get(ctx, actor, ids[0])
	if err != nil || !member.AdminSuspended || member.Entitlement.State != "suspended" {
		t.Fatal(member, err)
	}
	suspended = false
	version = member.ResourceVersion
	c.Key = uuid.New()
	if _, err = s.Patch(ctx, c, ids[0], &version, api.MemberPatch{AdminSuspended: &suspended}); err != nil {
		t.Fatal(err)
	}
	member, err = s.Get(ctx, actor, ids[0])
	if err != nil || member.AdminSuspended {
		t.Fatal(member, err)
	}
	version = member.ResourceVersion
	c.Key = uuid.New()
	if _, err = s.Close(ctx, c, ids[0], &version, "TEST ONLY closure"); err != nil {
		t.Fatal(err)
	}
	member, err = s.Get(ctx, actor, ids[0])
	if err != nil || member.Status != "closed" || member.ClosedAt == nil || member.Entitlement.State != "closed" {
		t.Fatal(member, err)
	}
	if _, err = s.Get(ctx, entitlement.Actor{ID: ids[1], Role: "member"}, ids[0]); !errors.Is(err, db.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestAT04_FiniteFutureGrantActivationRevokeAndAudit(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	engine, _ := entitlement.NewStore(entitlement.TestConfig())
	engine.Now = func() time.Time { return now }
	s := Service{Pool: pool, Entitlements: engine}
	member, plan, revision, group := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id) VALUES($1)`, member)
	exec(`INSERT INTO plans(id,code) VALUES($1,'test')`, plan)
	exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{}')`, revision, plan, []uuid.UUID{group})
	actor := entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}
	c := entitlement.Command{Actor: actor, Key: uuid.New()}
	start := now.Add(time.Hour)
	request := api.GrantCreateRequest{Source: api.GrantSourceTrial, ValidFrom: &start, ValidUntil: start.Add(time.Hour), Reason: "TEST ONLY finite trial", PlanRevisionId: &revision}
	change, err := s.Grant(ctx, c, member, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Grant(ctx, c, member, request)
	if err != nil || !replay.Replayed || replay.ResourceID != change.ResourceID {
		t.Fatal(replay, err)
	}
	before, err := s.Get(ctx, actor, member)
	if err != nil || before.Entitlement.Allows(now) || len(before.Grants) != 1 {
		t.Fatal(before, err)
	}
	now = start
	if _, err = engine.Sweep(ctx, pool, 100); err != nil {
		t.Fatal(err)
	}
	active, err := s.Get(ctx, actor, member)
	if err != nil || !active.Entitlement.Allows(now) {
		t.Fatal(active, err)
	}
	op, err := (operations.Store{Pool: pool}).Get(ctx, *change.OperationID, auth.Principal{Role: auth.Admin, ID: actor.ID})
	if err != nil || op.Kind != api.OperationKindMemberGrant || op.Resource.Type != api.ResourceTypeEntitlementGrant || op.Resource.Id != change.ResourceID {
		t.Fatal(op, err)
	}
	flag := false
	for range 2 {
		c.Key = uuid.New()
		if _, err = s.Patch(ctx, c, member, &active.ResourceVersion, api.MemberPatch{AdminSuspended: &flag}); err != nil {
			t.Fatal(err)
		}
		active, err = s.Get(ctx, actor, member)
		if err != nil || active.Status != "active" || active.Entitlement.State != "active" {
			t.Fatal(active, err)
		}
	}
	var desired decstr.U64
	if err = pool.QueryRow(ctx, `SELECT desired_generation::text FROM operations WHERE id=$1`, change.OperationID).Scan(&desired); err != nil || desired != active.Entitlement.Revision {
		t.Fatal(desired, err)
	}
	c.Key = uuid.New()
	if err = s.RevokeGrant(ctx, c, member, change.ResourceID, "TEST ONLY revocation"); err != nil {
		t.Fatal(err)
	}
	var revokeID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM operations WHERE resource_type=$1 AND resource_id=$2 AND kind=$3`, api.ResourceTypeEntitlementGrant, change.ResourceID, api.OperationKindMemberGrantRevoke).Scan(&revokeID); err != nil {
		t.Fatal(err)
	}
	op, err = (operations.Store{Pool: pool}).Get(ctx, revokeID, auth.Principal{Role: auth.Admin, ID: actor.ID})
	if err != nil || !op.Kind.Valid() || op.Kind != api.OperationKindMemberGrantRevoke || !op.Resource.Type.Valid() || op.Resource.Type != api.ResourceTypeEntitlementGrant || op.Resource.Id != change.ResourceID {
		t.Fatal(op, err)
	}
	revoked, err := s.Get(ctx, actor, member)
	if err != nil || revoked.Entitlement.Allows(now) {
		t.Fatal(revoked, err)
	}
	request.ValidUntil = now.Add(100 * 365 * 24 * time.Hour)
	c.Key = uuid.New()
	if _, err = s.Grant(ctx, c, member, request); !errors.Is(err, entitlement.ErrInvalid) {
		t.Fatal("far future grant accepted", err)
	}
	auditStore := audit.Store{Pool: pool}
	page, err := auditStore.List(ctx, api.ListAuditEventsParams{})
	if err != nil || auditStore.AnomalyCount() != 0 {
		t.Fatal(page, err)
	}
	for _, event := range page.Items {
		if event.Action == "member.grant" || event.Action == "grant.revoke" {
			if event.Resource.Type != api.ResourceTypeEntitlementGrant || event.Resource.Id != change.ResourceID {
				t.Fatal(event)
			}
		}
	}
	var n int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('member.grant','grant.revoke')`).Scan(&n); err != nil || n != 2 {
		t.Fatal(n, err)
	}
}
