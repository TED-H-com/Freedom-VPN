//go:build integration

package members

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestReviewR17_RevokedGrantIsNotResurrected(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	engine, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	engine.Now = func() time.Time { return now }
	s := Service{Pool: pool, Entitlements: engine}
	member, other, plan, revision, group := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id) VALUES($1),($2)`, member, other)
	exec(`INSERT INTO plans(id,code) VALUES($1,'r17-grant')`, plan)
	exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{}')`, revision, plan, []uuid.UUID{group})
	actor := entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}
	start := now.Add(-time.Hour)
	request := api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: now.Add(24 * time.Hour), Reason: "TEST ONLY finite grant", PlanRevisionId: &revision}
	first, err := s.Grant(ctx, entitlement.Command{Actor: actor, Key: uuid.New()}, member, request)
	if err != nil {
		t.Fatal(err)
	}
	secondReq := request
	secondReq.Reason = "TEST ONLY second grant"
	second, err := s.Grant(ctx, entitlement.Command{Actor: actor, Key: uuid.New()}, other, secondReq)
	if err != nil {
		t.Fatal(err)
	}
	key := uuid.New()
	revoked, err := s.RevokeMemberGrant(ctx, entitlement.Command{Actor: actor, Key: key}, member, first.ResourceID, "TEST ONLY withdrawal")
	if err != nil || revoked.Replayed {
		t.Fatal(revoked, err)
	}
	replay, err := s.RevokeMemberGrant(ctx, entitlement.Command{Actor: actor, Key: key}, member, first.ResourceID, "TEST ONLY withdrawal")
	if err != nil || !replay.Replayed || replay.OperationID == nil || *replay.OperationID != *revoked.OperationID {
		t.Fatal(replay, err)
	}
	// The same key on another member's grant is a different route. It must
	// withdraw that grant rather than replay the first receipt.
	otherRevoke, err := s.RevokeMemberGrant(ctx, entitlement.Command{Actor: actor, Key: key}, other, second.ResourceID, "TEST ONLY other withdrawal")
	if err != nil || otherRevoke.Replayed {
		t.Fatalf("TODO(review): idempotency key reused across grants replayed the first withdrawal (change=%+v err=%v)", otherRevoke, err)
	}
	var live int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM entitlement_grants WHERE id IN ($1,$2) AND revoked_at IS NULL`, first.ResourceID, second.ResourceID).Scan(&live); err != nil || live != 0 {
		t.Fatalf("withdrawn grants still live: %d %v", live, err)
	}
	replacement, err := s.Grant(ctx, entitlement.Command{Actor: actor, Key: uuid.New()}, member, request)
	if err != nil || replacement.ResourceID == first.ResourceID {
		t.Fatal("replacement resurrected the withdrawn row", replacement, err)
	}
	var cleared int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM entitlement_grants WHERE id=$1 AND revoked_at IS NULL`, first.ResourceID).Scan(&cleared); err != nil || cleared != 0 {
		t.Fatal("replacement cleared the withdrawn grant", cleared, err)
	}
	far := request
	far.ValidUntil = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err = s.Grant(ctx, entitlement.Command{Actor: actor, Key: uuid.New()}, member, far); !errors.Is(err, entitlement.ErrInvalid) {
		t.Fatal("year 9999 grant accepted", err)
	}
}
