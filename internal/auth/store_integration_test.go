//go:build integration

package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
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
	if e != nil || p.ID == id {
		t.Fatal("cross-client identity merged", e)
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
