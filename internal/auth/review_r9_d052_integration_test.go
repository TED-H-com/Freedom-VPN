//go:build integration

package auth

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func reviewActivity(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT pid::text, coalesce(wait_event_type,''), coalesce(wait_event,''), state, left(query, 120) FROM pg_stat_activity WHERE datname = current_database()`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out string
	for rows.Next() {
		var pid, kind, event, state, query string
		if err = rows.Scan(&pid, &kind, &event, &state, &query); err != nil {
			t.Fatal(err)
		}
		out += pid + " " + kind + "/" + event + " " + state + " " + query + "\n"
	}
	return out
}

func reviewRaceCanonical(t *testing.T, role Role, issuerVariants []string, subject string) {
	t.Helper()
	pool := dbtest.New(t)
	store := PostgresStore{Pool: pool}
	ctx := context.Background()
	canon := canonicalIssuer(issuerVariants[0])
	for _, raw := range issuerVariants {
		if canonicalIssuer(raw) != canon {
			t.Fatalf("variant %q canonicalizes to %q, want %q", raw, canonicalIssuer(raw), canon)
		}
	}
	parts := []string{canon, subject}
	if role == Admin {
		parts = []string{"admin", canon, subject}
	}
	scoped, err := json.Marshal(parts)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var key int64
	if err = conn.QueryRow(ctx, `SELECT hashtextextended($1::text, 0)`, string(scoped)).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, key); err != nil {
		t.Fatal(err)
	}
	unlocked := false
	unlock := func() {
		if unlocked {
			return
		}
		unlocked = true
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, key)
	}
	defer unlock()

	start := make(chan struct{})
	var wg sync.WaitGroup
	ids := make(chan uuid.UUID, len(issuerVariants))
	errs := make(chan error, len(issuerVariants))
	now := time.Now().UTC()
	for i, raw := range issuerVariants {
		wg.Add(1)
		go func(i int, raw string) {
			defer wg.Done()
			<-start
			token := randomToken()
			p, loginErr := store.Login(ctx, Identity{
				Issuer: raw, ClientScope: "portal-" + string(rune('a'+i)), Subject: subject,
				Label: "Review", Role: role, MFA: role == Admin, AuthTime: now, Expires: now.Add(time.Hour),
			}, token, "", time.Hour, uuid.New())
			if loginErr != nil {
				errs <- loginErr
				return
			}
			ids <- p.ID
		}(i, raw)
	}
	close(start)

	deadline := time.Now().Add(3 * time.Second)
	waiters := 0
	for time.Now().Before(deadline) {
		if err = pool.QueryRow(ctx, `SELECT count(DISTINCT pid) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid() AND wait_event_type = 'Lock' AND wait_event = 'advisory'`).Scan(&waiters); err != nil {
			t.Fatal(err)
		}
		if waiters >= len(issuerVariants) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	table := "identity_bindings"
	if role == Admin {
		table = "admin_identities"
	}
	var during int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE subject=$1`, subject).Scan(&during); err != nil {
		t.Fatal(err)
	}
	activity := reviewActivity(t, pool)
	unlock()
	wg.Wait()
	close(ids)
	close(errs)
	for loginErr := range errs {
		t.Fatalf("login during canonical lock: %v\n%s", loginErr, activity)
	}
	if waiters < len(issuerVariants) {
		t.Fatalf("expected %d distinct backends waiting on the advisory lock, got %d\n%s", len(issuerVariants), waiters, activity)
	}
	if during != 0 {
		t.Fatalf("TODO(review): %s row existed while the canonical advisory lock was held (lock key differs from lookup): count=%d\n%s", table, during, activity)
	}
	var gotID uuid.UUID
	n := 0
	for id := range ids {
		n++
		if gotID == uuid.Nil {
			gotID = id
		} else if id != gotID {
			t.Fatalf("TODO(review): mixed-case issuers created %s and %s", gotID, id)
		}
	}
	if n != len(issuerVariants) {
		t.Fatalf("logins=%d", n)
	}
	var stored int
	var issuer string
	if err = pool.QueryRow(ctx, `SELECT count(*), min(issuer) FROM `+table+` WHERE subject=$1`, subject).Scan(&stored, &issuer); err != nil {
		t.Fatal(err)
	}
	if stored != 1 || issuer != canon {
		t.Fatalf("TODO(review): stored %d rows issuer %q, canonical %q", stored, issuer, canon)
	}
}

func TestReviewR9_CanonicalLockMatchesLookup(t *testing.T) {
	variants := []string{
		"https://idp.example.invalid/dex",
		"https://IDP.Example.Invalid/dex",
		"https://idp.example.invalid:443/dex",
		"https://idp.example.invalid/dex/",
		"https://IDP.example.invalid:443/dex/",
		"https://Idp.Example.Invalid/dex",
		"https://idp.example.invalid/dex///",
		"https://IDP.EXAMPLE.INVALID/dex",
	}
	t.Run("member", func(t *testing.T) {
		reviewRaceCanonical(t, Member, variants, "review-member-"+uuid.NewString())
	})
	t.Run("admin", func(t *testing.T) {
		reviewRaceCanonical(t, Admin, variants, "review-admin-"+uuid.NewString())
	})
}

func TestReviewR9_AdminOldSessionSurvivesClientScopeRotation(t *testing.T) {
	pool := dbtest.New(t)
	store := PostgresStore{Pool: pool}
	ctx := context.Background()
	now := time.Now().UTC()
	subject := "review-admin-session-" + uuid.NewString()
	oldToken := randomToken()
	first, err := store.Login(ctx, Identity{
		Issuer: "https://IDP.Example.Invalid:443/dex/", ClientScope: "portal-v1", Subject: subject,
		Label: "Review Admin", Role: Admin, MFA: true, AuthTime: now, Expires: now.Add(time.Hour),
	}, oldToken, "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Login(ctx, Identity{
		Issuer: "https://idp.example.invalid/dex", ClientScope: "portal-v2", Subject: subject,
		Label: "Review Admin", Role: Admin, MFA: true, AuthTime: now, Expires: now.Add(time.Hour),
	}, randomToken(), "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("TODO(review): client scope rotation created admin %s and %s", first.ID, second.ID)
	}
	session, err := store.Session(ctx, oldToken, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if session.ID != first.ID || session.Role != Admin {
		t.Fatalf("old admin session after scope rotation: %+v", session)
	}
	var sessions int
	var scope, issuer string
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM admin_sessions WHERE admin_id=$1`, first.ID).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*), min(client_scope), min(issuer) FROM admin_identities WHERE subject=$1`, subject).Scan(new(int), &scope, &issuer); err != nil {
		t.Fatal(err)
	}
	var identities int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM admin_identities WHERE subject=$1`, subject).Scan(&identities); err != nil {
		t.Fatal(err)
	}
	if identities != 1 || scope != "portal-v2" || issuer != canonicalIssuer("https://idp.example.invalid/dex") || sessions != 2 {
		t.Fatalf("identities=%d scope=%s issuer=%s sessions=%d", identities, scope, issuer, sessions)
	}
}

func TestReviewR9_DistinctIssuersStayDistinctMembers(t *testing.T) {
	pool := dbtest.New(t)
	store := PostgresStore{Pool: pool}
	ctx := context.Background()
	now := time.Now().UTC()
	subject := "review-split-" + uuid.NewString()
	issuers := []string{"https://idp.example.invalid/dex", "https://other.example.invalid/dex", "http://idp.example.invalid/dex"}
	seen := map[uuid.UUID]string{}
	for _, issuer := range issuers {
		p, err := store.Login(ctx, Identity{
			Issuer: issuer, ClientScope: "portal-v1", Subject: subject, Label: "Review", Role: Member,
			AuthTime: now, Expires: now.Add(time.Hour),
		}, randomToken(), "", time.Hour, uuid.New())
		if err != nil {
			t.Fatal(issuer, err)
		}
		if prev, ok := seen[p.ID]; ok {
			t.Fatalf("TODO(review): %q and %q collapsed to member %s", prev, issuer, p.ID)
		}
		seen[p.ID] = issuer
	}
	if got := len(seen); got != len(issuers) {
		t.Fatal(got)
	}
}
