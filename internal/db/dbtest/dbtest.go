// Package dbtest creates isolated PostgreSQL databases for integration tests.
package dbtest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New creates and migrates a fresh fvpn_<random> database. Cleanup uses an
// independent context so a failed/canceled test still drops its own database.
// It never changes cluster roles or other databases. No DSN is printed.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return newDatabase(t, true)
}

// NewUnmigrated provides the same isolation and cleanup for concurrent bootstrap
// tests, which must exercise Migrate before any schema exists.
func NewUnmigrated(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return newDatabase(t, false)
}

func newDatabase(t testing.TB, migrate bool) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("FVPN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("integration test requires FVPN_TEST_DATABASE_URL (PostgreSQL 17 admin DSN)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, err := db.Open(ctx, dsn, db.OpenOptions{MaxConns: 2})
	if err != nil {
		t.Fatal("open test database admin connection:", err)
	}
	name := "fvpn_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		admin.Close()
		t.Fatal("create isolated database failed")
	}
	t.Logf("isolated database: %s", name)
	var pool *pgxpool.Pool
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		if _, err := admin.Exec(cleanup, "DROP DATABASE "+quoted+" WITH (FORCE)"); err != nil {
			t.Error("drop isolated test database failed:", name)
		}
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid test database config")
	}
	cfg.ConnConfig.Database = name
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["timezone"] = "UTC"
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "30000"
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "10000"
	cfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "30000"
	cfg.MaxConns = 10
	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("open isolated database failed")
	}
	if migrate {
		if err = db.Migrate(ctx, pool); err != nil {
			t.Fatalf("migrate isolated database: %v", err)
		}
	}
	return pool
}
