//go:build integration

package entitlement

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"slices"
	"testing"
	"time"
)

func servingFixture(t *testing.T) (*pgxpool.Pool, *Store, uuid.UUID, []uuid.UUID, time.Time) {
	t.Helper()
	p := dbtest.New(t)
	ctx := context.Background()
	m, plan, rev := uuid.New(), uuid.New(), uuid.New()
	groups := []uuid.UUID{uuid.New(), uuid.New()}
	now := time.Now().UTC().Truncate(time.Microsecond)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := p.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id,member_group_ids) VALUES($1,$2)`, m, []uuid.UUID{groups[1], groups[0]})
	exec(`INSERT INTO plans(id,code) VALUES($1,'TEST-ONLY')`, plan)
	exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{"rate_limit":{"down_bps":800000,"up_bps":400000}}')`, rev, plan, groups[:1])
	exec(`INSERT INTO entitlement_grants(member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,'admin_grant',$2,$3,'TEST ONLY', $4,$5)`, m, now.Add(-time.Hour), now.Add(time.Hour), uuid.NewString(), rev)
	store, _ := NewStore(TestConfig())
	store.Now = func() time.Time { return now }
	if err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { _, e := store.Recompute(ctx, tx, m); return e }); err != nil {
		t.Fatal(err)
	}
	return p, store, m, groups, now
}

func TestENT01_D030ServingReaders(t *testing.T) {
	p, _, m, groups, now := servingFixture(t)
	ctx := context.Background()
	all, err := ListServingMembersForGroups(ctx, p, groups, now)
	if err != nil || len(all) != 1 {
		t.Fatal(all, err)
	}
	got := all[0]
	if got.MemberID != m || got.DeviceLimit != 3 || got.PlanRevisionID == nil || got.PlanRateLimit == nil || got.PlanRateLimit.DownBps != 800000 || got.PlanRateLimit.UpBps != 400000 || got.Revision != 1 || len(got.ExitGroupIDs) != 2 {
		t.Fatal(got)
	}
	for _, g := range groups {
		if !slices.Contains(got.ExitGroupIDs, g) {
			t.Fatal("missing union group", g)
		}
	}
	for _, g := range [][]uuid.UUID{nil, {}, {uuid.New()}} {
		rows, e := ListServingMembersForGroups(ctx, p, g, now)
		if e != nil || len(rows) != 0 {
			t.Fatal(rows, e)
		}
	}
	for _, tc := range []struct {
		name, state, member  string
		suspended, projected bool
		offset               time.Duration
		want                 bool
	}{
		{"active", "active", "active", false, false, 0, true},
		{"cancel", "cancel_at_period_end", "active", false, false, 0, true},
		{"grace", "grace", "active", false, false, 0, true},
		{"past_due_with_finite_grace", "past_due", "active", false, false, 0, true},
		{"pending", "pending_payment", "active", false, false, 0, false},
		{"suspended", "suspended", "active", false, false, 0, false},
		{"expired", "expired", "active", false, false, 0, false},
		{"closed", "closed", "active", false, false, 0, false},
		{"member_pending", "active", "pending_payment", false, false, 0, false},
		{"member_closed", "active", "closed", false, false, 0, false},
		{"manual_suspend", "active", "active", true, false, 0, false},
		{"projected_suspend", "active", "active", false, true, 0, false},
		{"before_start", "active", "active", false, false, -2 * time.Hour, false},
		{"at_start", "active", "active", false, false, -time.Hour, true},
		{"at_end", "active", "active", false, false, time.Hour, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, e := p.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(ctx)
			if _, e = tx.Exec(ctx, `UPDATE members SET status=$2,admin_suspended=$3,resource_version=resource_version+1 WHERE id=$1`, m, tc.member, tc.suspended); e != nil {
				t.Fatal(e)
			}
			if _, e = tx.Exec(ctx, `UPDATE entitlements SET state=$2,admin_suspended=$3 WHERE member_id=$1`, m, tc.state, tc.projected); e != nil {
				t.Fatal(e)
			}
			rows, e := ListServingMembersForGroups(ctx, tx, groups, now.Add(tc.offset))
			if e != nil || (len(rows) == 1) != tc.want {
				t.Fatal(rows, e)
			}
			exitGroups, e := ServingExitGroups(ctx, tx, m, now.Add(tc.offset))
			if e != nil || (len(exitGroups) == 2) != tc.want {
				t.Fatal(exitGroups, e)
			}
		})
	}
	if gs, e := ServingExitGroups(ctx, p, uuid.New(), now); e != nil || len(gs) != 0 {
		t.Fatal(gs, e)
	}
}
func TestENT01_RecomputeRepairsStatusWithoutNewRevision(t *testing.T) {
	p, store, m, _, _ := servingFixture(t)
	ctx := context.Background()
	if _, e := p.Exec(ctx, `UPDATE members SET status='pending_payment',resource_version=resource_version+1 WHERE id=$1`, m); e != nil {
		t.Fatal(e)
	}
	if e := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		r, e := store.Recompute(ctx, tx, m)
		if e == nil && (r.Changed || r.Revision != 1) {
			t.Fatal(r)
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	var status string
	var events int
	if e := p.QueryRow(ctx, `SELECT status,(SELECT count(*) FROM outbox_events WHERE event_type='member.entitlement_changed') FROM members WHERE id=$1`, m).Scan(&status, &events); e != nil || status != "active" || events != 1 {
		t.Fatal(status, events, e)
	}
}
