//go:build integration

package db_test

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// D-035 / §15.4, §16.4: upgrade an existing schema, read the actual purge row,
// and prove Down restores the prior body without changing the privilege boundary.
func TestReviewD035_PurgeAuditMigrationRoundTrip(t *testing.T) {
	p := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(p)
	defer sqlDB.Close()
	old, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithExcludeVersions([]int64{6}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	snapshot := func() (body, metadata string) {
		t.Helper()
		if err := p.QueryRow(t.Context(), `SELECT pg_get_functiondef(oid),
   jsonb_build_object('oid',oid,'owner',proowner,'acl',proacl,'security_definer',prosecdef,'settings',proconfig)::text
   FROM pg_proc WHERE oid='public.purge_retained(text,timestamptz,text)'::regprocedure`).Scan(&body, &metadata); err != nil {
			t.Fatal(err)
		}
		return
	}
	previousBody, previousMetadata := snapshot()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	// Other domain migrations already have higher versions. Apply only the
	// requested backfilled version; do not roll back or rerun domain migrations.
	if _, err := provider.ApplyVersion(t.Context(), 6, true); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	canonicalBody, metadata := snapshot()
	if canonicalBody == previousBody || metadata != previousMetadata {
		t.Fatal("migration did not replace only the body")
	}
	exec(t, p, `INSERT INTO audit_events(actor,actor_type,action,resource,result,occurred_at)
  VALUES('00000000-0000-0000-0000-000000000000','system','retention.test','fleet_config:00000000-0000-0000-0000-000000000001','succeeded',now()-interval '200 days')`)
	// Invalid cutoffs neither delete the expired row nor append an audit event.
	_, err = p.Exec(t.Context(), `SELECT purge_retained('audit_events',now(),'too early')`)
	sqlError(t, err, "22023", "")
	reviewCount(t, p, `SELECT count(*) FROM audit_events`, 1)
	reviewCount(t, p, `SELECT purge_retained('audit_events',now()-interval '181 days','D-035 retention test')`, 1)
	reviewRetentionAudit(t, p, "audit_events", "1")
	reviewCount(t, p, `SELECT count(*) FROM audit_events WHERE actor='00000000-0000-0000-0000-000000000000'
  AND actor_type='system' AND resource='fleet_config:00000000-0000-0000-0000-000000000001'
  AND action='retention.purge' AND result='succeeded'`, 1)
	// Canonical storage must also work with the reader's SQL filters.
	store := audit.Store{Pool: p}
	actor, resource := uuid.Nil, uuid.MustParse("00000000-0000-0000-0000-000000000001")
	typ := api.ResourceTypeFleetConfig
	page, err := store.List(t.Context(), api.ListAuditEventsParams{ActorId: &actor, ResourceType: &typ, ResourceId: &resource})
	if err != nil || len(page.Items) != 1 || store.AnomalyCount() != 0 {
		t.Fatalf("filtered purge audit: items=%d anomalies=%d err=%v", len(page.Items), store.AnomalyCount(), err)
	}
	if _, err := provider.ApplyVersion(t.Context(), 6, false); err != nil {
		t.Fatal(err)
	}
	restoredBody, metadata := snapshot()
	if restoredBody != previousBody || metadata != previousMetadata {
		t.Fatal("Down did not restore the original body and privileges")
	}
	if _, err := provider.ApplyVersion(t.Context(), 6, true); err != nil {
		t.Fatal(err)
	}
	reappliedBody, metadata := snapshot()
	if reappliedBody != canonicalBody || metadata != previousMetadata {
		t.Fatal("reapplied Up differs from initial Up")
	}
	// Migration/rollback never mutates the already committed audit evidence.
	reviewRetentionAudit(t, p, "audit_events", "1")
}
