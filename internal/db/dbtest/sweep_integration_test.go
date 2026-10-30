//go:build integration

package dbtest

import (
	"bytes"
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

func TestSweep_DropsOnlyIdleOldIsolatedDatabases(t *testing.T) {
	pool := openAdmin(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Anchor is two hours ahead of the wall clock. Fixtures encoded against it
	// are not older than one hour of wall time, so another process sweeping
	// with the real clock leaves them alone. This sweep judges age from anchor.
	anchor := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	oldName := mustDatabaseName(t, anchor.Add(-2*time.Hour))
	youngName := mustDatabaseName(t, anchor.Add(-30*time.Minute))
	busyName := mustDatabaseName(t, anchor.Add(-2*time.Hour))
	legacyIdle := "fvpn_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	legacyBusy := "fvpn_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	foreign := "fvpn_do_not_touch_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	allow := map[string]struct{}{}
	for _, name := range []string{oldName, youngName, busyName, legacyIdle, legacyBusy, foreign} {
		createDatabase(t, pool, name)
		allow[name] = struct{}{}
	}
	holdConnection(t, pool, busyName)
	holdConnection(t, pool, legacyBusy)

	var adminName string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&adminName); err != nil {
		t.Fatal(err)
	}
	protected := []string{adminName, "template0", "template1", youngName, busyName, legacyIdle, legacyBusy, foreign}

	var log bytes.Buffer
	previousLog := sweepLog
	sweepLog = &log
	t.Cleanup(func() { sweepLog = previousLog })

	t.Setenv("FVPN_TEST_DB_SWEEP_LEGACY", "")
	dropped, err := sweep(ctx, pool, time.Hour, anchor, allow)
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped) != 1 || dropped[0] != oldName {
		t.Fatalf("dropped %v, want %s", dropped, oldName)
	}
	if databaseExists(t, pool, oldName) {
		t.Fatal("old database still exists")
	}
	for _, name := range protected {
		if !databaseExists(t, pool, name) {
			t.Fatalf("sweep removed %s", name)
		}
	}

	t.Setenv("FVPN_TEST_DB_SWEEP_LEGACY", "1")
	dropped, err = sweep(ctx, pool, time.Hour, anchor, allow)
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped) != 1 || dropped[0] != legacyIdle {
		t.Fatalf("legacy dropped %v, want %s", dropped, legacyIdle)
	}
	if databaseExists(t, pool, legacyIdle) {
		t.Fatal("idle legacy database still exists")
	}
	for _, name := range []string{adminName, "template0", "template1", youngName, busyName, legacyBusy, foreign} {
		if !databaseExists(t, pool, name) {
			t.Fatalf("legacy sweep removed %s", name)
		}
	}
	wantLog := "dbtest sweep: dropped=1 skipped_busy=1\ndbtest sweep: dropped=1 skipped_busy=2\n"
	if log.String() != wantLog {
		t.Fatalf("sweep log %q", log.String())
	}
}

func openAdmin(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("FVPN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("integration test requires FVPN_TEST_DATABASE_URL (PostgreSQL 17 admin DSN)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, dsn, db.OpenOptions{MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func mustDatabaseName(t *testing.T, created time.Time) string {
	t.Helper()
	name, err := databaseName(created)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func createDatabase(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	quoted := pgx.Identifier{name}.Sanitize()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cctx, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		_, _ = pool.Exec(cctx, "DROP DATABASE IF EXISTS "+quoted+" WITH (FORCE)")
	})
}

func holdConnection(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	cfg := pool.Config().ConnConfig.Copy()
	cfg.Database = name
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	var one int
	if err := conn.QueryRow(context.Background(), "SELECT 1").Scan(&one); err != nil {
		t.Fatal(err)
	}
}

func databaseExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var found bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&found); err != nil {
		t.Fatal(err)
	}
	return found
}
