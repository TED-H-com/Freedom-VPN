//go:build integration

package db_test

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func Test15_5_MigrationsRoundTrip(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatalf("initial Up: %v", err)
	}
	if _, err := provider.DownTo(t.Context(), migrationDownFloor); err != nil {
		t.Fatalf("DownTo(%d): %v", migrationDownFloor, err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatalf("Up after DownTo(%d): %v", migrationDownFloor, err)
	}
}
