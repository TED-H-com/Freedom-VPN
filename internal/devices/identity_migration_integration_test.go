//go:build integration

package devices

import (
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestD052_F04MigrationRefusesHistoricalDuplicateIdentities(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.UpTo(t.Context(), 181); err != nil {
		t.Fatal(err)
	}
	// Negative migration input only: conflicting historical rows, never a silent
	// member merge or a positively seeded authorization projection (D-050).
	for _, scope := range []string{"TEST-old-client", "TEST-new-client"} {
		id := uuid.New()
		if _, err = pool.Exec(t.Context(), `INSERT INTO members(id) VALUES($1)`, id); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(t.Context(), `INSERT INTO identity_bindings(member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,'https://issuer.example.invalid',$2,'TEST-private-subject','sub','oidc',clock_timestamp())`, id, scope); err != nil {
			t.Fatal(err)
		}
	}
	err = db.Migrate(t.Context(), pool)
	if err == nil || !strings.Contains(err.Error(), "duplicate issuer/subject groups: 1") || strings.Contains(err.Error(), "TEST-private-subject") {
		t.Fatal("migration did not safely report duplicate count", err)
	}
	var count int
	if err = pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_bindings`).Scan(&count); err != nil || count != 2 {
		t.Fatal("migration rewrote historical identities", count, err)
	}
}

func TestD052_F04MigrationEnforcesPublicSubjectUniqueness(t *testing.T) {
	pool := dbtest.New(t)
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM pg_constraint WHERE conrelid='identity_bindings'::regclass AND conname='identity_bindings_issuer_subject_key' AND contype='u'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("shared subject unique constraint missing", err)
	}
}
