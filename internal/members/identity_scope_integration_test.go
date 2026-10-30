//go:build integration

package members

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func scopeIdentity(subject string) auth.Identity {
	now := time.Now().UTC()
	return auth.Identity{
		Issuer: "https://issuer.example.invalid", ClientScope: "TEST ONLY portal-v1",
		Subject: subject, Label: "TEST ONLY member", Role: auth.Member,
		AuthTime: now, Expires: now.Add(time.Hour),
	}
}

// ID01 / D-052 / §15.2, §16.4: the current schema persists rotated scope through
// real Login, preserving identity and avoiding a proof-update audit on every login.
func TestID01_D052ClientScopeRotation(t *testing.T) {
	pool := dbtest.New(t)
	store := auth.PostgresStore{Pool: pool}
	identity := scopeIdentity("TEST ONLY public subject")
	login := func() uuid.UUID {
		t.Helper()
		principal, err := store.Login(t.Context(), identity, uuid.NewString(), "", time.Hour, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		return principal.ID
	}
	member := login()
	var binding uuid.UUID
	if err := pool.QueryRow(t.Context(), `SELECT id FROM identity_bindings WHERE member_id=$1`, member).Scan(&binding); err != nil {
		t.Fatal(err)
	}
	identity.ClientScope = "TEST ONLY portal-v2"
	identity.Issuer = "https://ISSUER.example.invalid:443/"
	for range 3 {
		if got := login(); got != member {
			t.Fatalf("rotation changed member: got %s want %s", got, member)
		}
		var count int
		var scope string
		if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_bindings`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("binding count=%d err=%v", count, err)
		}
		if err := pool.QueryRow(t.Context(), `SELECT client_scope FROM identity_bindings WHERE id=$1 AND member_id=$2 AND issuer=$3 AND subject=$4`, binding, member, "https://issuer.example.invalid", identity.Subject).Scan(&scope); err != nil || scope != identity.ClientScope {
			t.Fatalf("stored scope=%q want=%q err=%v", scope, identity.ClientScope, err)
		}
		if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='identity_binding.proof_updated' AND resource=$1`, "member:"+member.String()).Scan(&count); err != nil || count != 1 {
			t.Fatalf("proof-update audit count=%d want=1 err=%v", count, err)
		}
	}
	t.Log("current schema: rotated scope stored; same member and binding; three logins produced one proof-update audit")
}

// §15.2, §15.5 / D-052: exercise the actual embedded migration, including Down,
// catalog restoration and preservation of identity fields and existing row data.
func Test15_5_IdentityScopeMigrationRoundTrip(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	previous, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithExcludeVersions([]int64{186}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := previous.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	store := auth.PostgresStore{Pool: pool}
	member, err := store.Login(t.Context(), scopeIdentity("TEST ONLY public subject"), uuid.NewString(), "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.Login(t.Context(), scopeIdentity("TEST ONLY other subject"), uuid.NewString(), "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func() (trigger, rows string) {
		t.Helper()
		if err := pool.QueryRow(t.Context(), `SELECT pg_get_triggerdef(oid) || ' enabled=' || tgenabled::text FROM pg_trigger WHERE tgrelid='identity_bindings'::regclass AND tgname='identity_bindings_identity'`).Scan(&trigger); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(t.Context(), `SELECT jsonb_agg(to_jsonb(b) ORDER BY b.id)::text FROM identity_bindings b`).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return
	}
	apply := func(up bool) {
		t.Helper()
		_, before := snapshot()
		if _, err := provider.ApplyVersion(t.Context(), 186, up); err != nil {
			t.Fatal(err)
		}
		_, after := snapshot()
		if after != before {
			t.Fatal("migration changed identity binding data")
		}
	}
	reject := func(t *testing.T, field, query string, value any) {
		t.Helper()
		_, before := snapshot()
		_, err := pool.Exec(t.Context(), query, member.ID, value)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.Message != "immutable identity field: "+field {
			t.Fatalf("%s mutation: want immutable-field trigger rejection, got %v", field, err)
		}
		_, after := snapshot()
		if after != before {
			t.Fatalf("rejected %s mutation changed stored data", field)
		}
	}
	const updateScope = `UPDATE identity_bindings SET client_scope=$2 WHERE member_id=$1`
	update := func(scope string) {
		t.Helper()
		tag, err := pool.Exec(t.Context(), updateScope, member.ID, scope)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("scope update: rows=%d err=%v", tag.RowsAffected(), err)
		}
		var stored string
		if err := pool.QueryRow(t.Context(), `SELECT client_scope FROM identity_bindings WHERE member_id=$1`, member.ID).Scan(&stored); err != nil || stored != scope {
			t.Fatalf("scope update: stored=%q want=%q err=%v", stored, scope, err)
		}
	}
	before, _ := snapshot()
	reject(t, "client_scope", updateScope, "TEST ONLY portal-v2")
	apply(true)
	after, _ := snapshot()
	if after == before {
		t.Fatal("Up did not change the identity trigger")
	}
	update("TEST ONLY portal-v2")
	for _, tc := range []struct {
		field, query string
		value        any
	}{
		{"id", `UPDATE identity_bindings SET id=$2 WHERE member_id=$1`, uuid.New()},
		{"issuer", `UPDATE identity_bindings SET issuer=$2 WHERE member_id=$1`, "https://other.example.invalid"},
		{"subject", `UPDATE identity_bindings SET subject=$2 WHERE member_id=$1`, "TEST ONLY different subject"},
		{"member_id", `UPDATE identity_bindings SET member_id=$2 WHERE member_id=$1`, other.ID},
	} {
		t.Run(tc.field+"_immutable", func(t *testing.T) {
			reject(t, tc.field, tc.query, tc.value)
		})
	}
	apply(false)
	if restored, _ := snapshot(); restored != before {
		t.Fatal("Down did not restore the original trigger catalog definition")
	}
	reject(t, "client_scope", updateScope, "TEST ONLY portal-v3")
	apply(true)
	if reapplied, _ := snapshot(); reapplied != after {
		t.Fatal("reapplied Up differs from initial Up")
	}
	update("TEST ONLY portal-v3")
	t.Log("00186 Up/Down/Up: catalog restored; migration preserved rows; client_scope mutable only after Up; id/member_id/issuer/subject immutable")
}
