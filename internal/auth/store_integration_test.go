//go:build integration

package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestID01_ScopedIdentitySessionRotationAndExpiry(t *testing.T) {
	pool := dbtest.New(t)
	store := PostgresStore{Pool: pool}
	identity := Identity{Issuer: "https://idp.example.invalid", ClientScope: "portal", Subject: "opaque-subject", Label: "Test Member", Role: Member, AuthTime: time.Now().UTC(), Expires: time.Now().Add(time.Hour)}
	var wg sync.WaitGroup
	ids := make(chan uuid.UUID, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, e := store.Login(t.Context(), identity, randomToken(), "", time.Hour, uuid.New())
			if e != nil {
				t.Error(e)
				return
			}
			ids <- p.ID
		}()
	}
	wg.Wait()
	close(ids)
	var id uuid.UUID
	for v := range ids {
		if id == uuid.Nil {
			id = v
		}
		if id != v {
			t.Fatal("duplicate first-login members")
		}
	}
	var members, audits int
	if e := pool.QueryRow(t.Context(), `SELECT count(*) FROM members WHERE status='pending_payment'`).Scan(&members); e != nil {
		t.Fatal(e)
	}
	if e := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='member.register'`).Scan(&audits); e != nil {
		t.Fatal(e)
	}
	if members != 1 || audits != 1 {
		t.Fatal(members, audits)
	}
	// §15.1 / F-14: setting the canonical identity is a versioned mutation.
	var version string
	var bound bool
	if e := pool.QueryRow(t.Context(), `SELECT resource_version::text,canonical_identity_ref IS NOT NULL FROM members WHERE id=$1`, id).Scan(&version, &bound); e != nil || version != "2" || !bound {
		t.Fatalf("canonical identity/version: %s %v %v", version, bound, e)
	}
	if _, e := store.Login(t.Context(), identity, "old", "", time.Hour, uuid.New()); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Login(t.Context(), identity, "new", "old", time.Hour, uuid.New()); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Session(t.Context(), "old", time.Minute); e == nil {
		t.Fatal("old session survived rotation")
	}
	if _, e := store.Session(t.Context(), "new", time.Minute); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(t.Context(), `UPDATE browser_sessions SET last_seen_at=clock_timestamp()-interval '1 hour' WHERE token_hash=$1`, digest("new")); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Session(t.Context(), "new", time.Minute); e == nil {
		t.Fatal("idle expiry not enforced")
	}
	identity.ClientScope = "other-client"
	p, e := store.Login(t.Context(), identity, "other", "", time.Hour, uuid.New())
	if e != nil || p.ID != id {
		t.Fatalf("client_scope rotation created duplicate member: got %v want %v (err=%v)", p.ID, id, e)
	}
	var bindings int
	if e := pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_bindings WHERE issuer=$1 AND subject=$2`, identity.Issuer, identity.Subject).Scan(&bindings); e != nil || bindings != 1 {
		t.Fatalf("expected 1 identity_binding after client_scope rotation, got %d (err=%v)", bindings, e)
	}
	if _, e := pool.Exec(t.Context(), `UPDATE browser_sessions SET created_at=clock_timestamp()-interval '2 hours',absolute_expires_at=clock_timestamp()-interval '1 hour' WHERE token_hash=$1`, digest("other")); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Session(t.Context(), "other", time.Hour); e == nil {
		t.Fatal("absolute expiry not enforced")
	}
	flow := Flow{State: "state", Browser: "browser", Nonce: "nonce", Verifier: "verifier", Expires: time.Now().Add(time.Minute)}
	if e := store.SaveFlow(t.Context(), flow); e != nil {
		t.Fatal(e)
	}
	if _, e := store.ConsumeFlow(t.Context(), "state", "wrong"); e == nil {
		t.Fatal("flow not browser bound")
	}
	if _, e := store.ConsumeFlow(t.Context(), "state", "browser"); e != nil {
		t.Fatal(e)
	}
	if _, e := store.ConsumeFlow(t.Context(), "state", "browser"); e == nil {
		t.Fatal("flow replay")
	}
}

func TestID01_LoginFlowRoleMigrationAndBackwardCompatibility(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	ctx := t.Context()
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	preProvider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithExcludeVersions([]int64{185}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := preProvider.Up(ctx); err != nil {
		t.Fatal(err)
	}

	schemaSnapshot := func() string {
		t.Helper()
		var snap string
		if err := pool.QueryRow(ctx, `
			SELECT jsonb_build_object(
				'columns', COALESCE((
					SELECT jsonb_agg(jsonb_build_object(
						'name', a.attname,
						'type', format_type(a.atttypid, a.atttypmod),
						'not_null', a.attnotnull
					) ORDER BY a.attname)
					FROM pg_attribute a
					WHERE a.attrelid = 'public.oidc_login_flows'::regclass
					  AND a.attnum > 0 AND NOT a.attisdropped
				), '[]'::jsonb),
				'constraints', COALESCE((
					SELECT jsonb_agg(jsonb_build_object(
						'type', c.contype,
						'def', pg_get_constraintdef(c.oid)
					) ORDER BY pg_get_constraintdef(c.oid))
					FROM pg_constraint c
					WHERE c.conrelid = 'public.oidc_login_flows'::regclass
				), '[]'::jsonb)
			)::text`).Scan(&snap); err != nil {
			t.Fatal(err)
		}
		return snap
	}
	beforeSchema := schemaSnapshot()

	exp := time.Now().UTC().Add(5 * time.Minute)
	// Begin login flows BEFORE migration 00185 (legacy prefix shape).
	for _, tc := range []struct {
		state, browser, nonce, verifier string
	}{
		{"pre-admin", "b-admin", "n-admin", "admin:pkce-verifier-admin"},
		{"pre-member", "b-member", "n-member", "member:pkce-verifier-member"},
		{"pre-unset", "b-unset", "n-unset", "pkce-verifier-unset"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO oidc_login_flows(state_hash,browser_hash,nonce,verifier,expires_at,id) VALUES($1,$2,$3,$4,$5,$6)`,
			digest(tc.state), digest(tc.browser), tc.nonce, tc.verifier, exp, uuid.Must(uuid.NewV7())); err != nil {
			t.Fatal(err)
		}
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.ApplyVersion(ctx, 185, true); err != nil {
		t.Fatal(err)
	}
	upSchema := schemaSnapshot()
	if upSchema == beforeSchema {
		t.Fatal("migration 00185 did not alter oidc_login_flows schema")
	}

	// Verify backfill stripped the prefix and populated role.
	for _, tc := range []struct {
		state, wantVerifier string
		wantRole            *string
	}{
		{"pre-admin", "pkce-verifier-admin", strPtr("admin")},
		{"pre-member", "pkce-verifier-member", strPtr("member")},
		{"pre-unset", "pkce-verifier-unset", nil},
	} {
		var gotVerifier string
		var gotRole *string
		if err := pool.QueryRow(ctx, `SELECT verifier, role FROM oidc_login_flows WHERE state_hash=$1`, digest(tc.state)).Scan(&gotVerifier, &gotRole); err != nil {
			t.Fatal(err)
		}
		if gotVerifier != tc.wantVerifier || (gotRole == nil) != (tc.wantRole == nil) || (gotRole != nil && *gotRole != *tc.wantRole) {
			t.Fatalf("backfill for %s: verifier=%q role=%v, want verifier=%q role=%v", tc.state, gotVerifier, gotRole, tc.wantVerifier, tc.wantRole)
		}
	}

	store := PostgresStore{Pool: pool}
	// Consume AFTER migration 00185 for flows begun BEFORE the migration.
	for _, tc := range []struct {
		state, browser, wantVerifier string
		wantRole                     Role
	}{
		{"pre-admin", "b-admin", "pkce-verifier-admin", Admin},
		{"pre-member", "b-member", "pkce-verifier-member", Member},
		{"pre-unset", "b-unset", "pkce-verifier-unset", ""},
	} {
		got, err := store.ConsumeFlow(ctx, tc.state, tc.browser)
		if err != nil {
			t.Fatalf("ConsumeFlow(%s): %v", tc.state, err)
		}
		if got.Verifier != tc.wantVerifier || got.Role != tc.wantRole {
			t.Fatalf("ConsumeFlow(%s) = (verifier=%q, role=%q), want (%q, %q)", tc.state, got.Verifier, got.Role, tc.wantVerifier, tc.wantRole)
		}
	}

	// Reader accepts legacy prefix shape for one release even when role column is NULL.
	for _, tc := range []struct {
		state, browser, rawVerifier, wantVerifier string
		wantRole                                  Role
	}{
		{"legacy-admin", "b-la", "admin:ver-la", "ver-la", Admin},
		{"legacy-member", "b-lm", "member:ver-lm", "ver-lm", Member},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO oidc_login_flows(state_hash,browser_hash,nonce,verifier,role,expires_at,id) VALUES($1,$2,'n',$3,NULL,$4,$5)`,
			digest(tc.state), digest(tc.browser), tc.rawVerifier, exp, uuid.Must(uuid.NewV7())); err != nil {
			t.Fatal(err)
		}
		got, err := store.ConsumeFlow(ctx, tc.state, tc.browser)
		if err != nil || got.Verifier != tc.wantVerifier || got.Role != tc.wantRole {
			t.Fatalf("legacy ConsumeFlow(%s) = (%+v, %v), want verifier=%q role=%q", tc.state, got, err, tc.wantVerifier, tc.wantRole)
		}
	}

	// Writer uses the role column only (never prefixing verifier) and enforces the role check constraint.
	if err := store.SaveFlow(ctx, Flow{State: "bad-role", Browser: "b", Nonce: "n", Verifier: "v", Role: Role("invalid"), Expires: exp}); err == nil {
		t.Fatal("expected CHECK constraint failure for invalid role")
	}
	for _, tc := range []struct {
		state, browser, verifier string
		role                     Role
	}{
		{"post-admin", "b-pa", "ver-pa", Admin},
		{"post-member", "b-pm", "ver-pm", Member},
		{"post-unset", "b-pu", "ver-pu", ""},
	} {
		if err := store.SaveFlow(ctx, Flow{State: tc.state, Browser: tc.browser, Nonce: "n", Verifier: tc.verifier, Role: tc.role, Expires: exp}); err != nil {
			t.Fatal(err)
		}
		var rawVerifier string
		var rawRole *string
		if err := pool.QueryRow(ctx, `SELECT verifier, role FROM oidc_login_flows WHERE state_hash=$1`, digest(tc.state)).Scan(&rawVerifier, &rawRole); err != nil {
			t.Fatal(err)
		}
		if rawVerifier != tc.verifier {
			t.Fatalf("SaveFlow prefixed verifier: got %q, want %q", rawVerifier, tc.verifier)
		}
		if (tc.role == "" && rawRole != nil) || (tc.role != "" && (rawRole == nil || *rawRole != string(tc.role))) {
			t.Fatalf("SaveFlow role column = %v, want %q", rawRole, tc.role)
		}
	}

	// Down migration restores the legacy prefix and matches the pre-00185 schema without drift.
	if _, err := provider.ApplyVersion(ctx, 185, false); err != nil {
		t.Fatal(err)
	}
	downSchema := schemaSnapshot()
	if downSchema != beforeSchema {
		t.Fatalf("migration 00185 Down schema drift:\ngot:  %s\nwant: %s", downSchema, beforeSchema)
	}
	for _, tc := range []struct {
		state, wantVerifier string
	}{
		{"post-admin", "admin:ver-pa"},
		{"post-member", "member:ver-pm"},
		{"post-unset", "ver-pu"},
	} {
		var gotVerifier string
		if err := pool.QueryRow(ctx, `SELECT verifier FROM oidc_login_flows WHERE state_hash=$1`, digest(tc.state)).Scan(&gotVerifier); err != nil {
			t.Fatal(err)
		}
		if gotVerifier != tc.wantVerifier {
			t.Fatalf("after Down %s verifier = %q, want %q", tc.state, gotVerifier, tc.wantVerifier)
		}
	}

	// Re-applying Up restores the column and strips the prefix cleanly.
	if _, err := provider.ApplyVersion(ctx, 185, true); err != nil {
		t.Fatal(err)
	}
	if reUpSchema := schemaSnapshot(); reUpSchema != upSchema {
		t.Fatalf("re-applied Up schema drift:\ngot:  %s\nwant: %s", reUpSchema, upSchema)
	}
	for _, tc := range []struct {
		state, browser, wantVerifier string
		wantRole                     Role
	}{
		{"post-admin", "b-pa", "ver-pa", Admin},
		{"post-member", "b-pm", "ver-pm", Member},
		{"post-unset", "b-pu", "ver-pu", ""},
	} {
		got, err := store.ConsumeFlow(ctx, tc.state, tc.browser)
		if err != nil || got.Verifier != tc.wantVerifier || got.Role != tc.wantRole {
			t.Fatalf("after re-Up ConsumeFlow(%s) = (%+v, %v)", tc.state, got, err)
		}
	}
}

func strPtr(s string) *string { return &s }
