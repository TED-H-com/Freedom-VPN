//go:build integration

package pki_test

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// §6.1–2 / AT07: exercise the actual goose Up/Down without dropping history or
// weakening certificate/incarnation guards from migration 00110 and the base.
func TestAT07_ReinstallMigrationRoundTrip(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	old, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithExcludeVersions([]int64{111}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = old.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	snapshot := func() (body, metadata string) {
		t.Helper()
		if err := pool.QueryRow(t.Context(), `SELECT pg_get_functiondef(oid),jsonb_build_object('oid',oid,'owner',proowner,'acl',proacl,'security_definer',prosecdef,'settings',proconfig)::text FROM pg_proc WHERE oid='public.pki_guard_terminal()'::regprocedure`).Scan(&body, &metadata); err != nil {
			t.Fatal(err)
		}
		return
	}
	before, metadata := snapshot()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.ApplyVersion(t.Context(), 111, true); err != nil {
		t.Fatal(err)
	}
	up, upMetadata := snapshot()
	if up == before || upMetadata != metadata {
		t.Fatal("Up must replace only the guard body")
	}
	if _, err = provider.ApplyVersion(t.Context(), 111, false); err != nil {
		t.Fatal(err)
	}
	down, downMetadata := snapshot()
	if down != before || downMetadata != metadata {
		t.Fatal("Down did not restore the original guard and privileges")
	}
	if _, err = provider.ApplyVersion(t.Context(), 111, true); err != nil {
		t.Fatal(err)
	}
	again, againMetadata := snapshot()
	if again != up || againMetadata != metadata {
		t.Fatal("reapplied Up differs")
	}
}
