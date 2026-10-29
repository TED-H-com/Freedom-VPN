//go:build integration

package billing

import (
	"context"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestMEM03_MigrateExistingSuspensionPreservesBaseGuards(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err = provider.UpTo(ctx, 5); err != nil {
		t.Fatal(err)
	}
	var before string
	if err = pool.QueryRow(ctx, `SELECT pg_get_functiondef('check_member_lease()'::regprocedure)`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	member := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO members(id,status) VALUES($1,'suspended')`, member); err != nil {
		t.Fatal(err)
	}
	if _, err = provider.UpTo(ctx, 120); err != nil {
		t.Fatal(err)
	}
	var version decstr.U64
	var suspended bool
	var reason, after string
	if err = pool.QueryRow(ctx, `SELECT resource_version::text,admin_suspended,suspension_reason FROM members WHERE id=$1`, member).Scan(&version, &suspended, &reason); err != nil {
		t.Fatal(err)
	}
	if version != 2 || !suspended || reason == "" {
		t.Fatal("lost suspension or version", version, suspended)
	}
	if err = pool.QueryRow(ctx, `SELECT pg_get_functiondef('check_member_lease()'::regprocedure)`).Scan(&after); err != nil || before != after {
		t.Fatal("base lease guard changed", err)
	}
}
