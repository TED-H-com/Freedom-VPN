//go:build integration

package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
)

// TestReview_F11_CrossPrincipalSessionOverwriteUnauditedAndAdminClaimBlocksMemberBinding
// proves two session/identity defects in Browser + PostgresStore:
//  1. When Member B logs in on a browser carrying Member A's __Host-fvpn-member cookie,
//     PostgresStore.Login deletes Member A's session row without emitting a
//     session.logout audit event for Member A (D-029, §16.4).
//  2. When a dual-role user (carrying the admin role claim in Dex) logs in via
//     /auth/login?return_to=/member, Browser.callback unconditionally forces
//     Role=Admin, so PostgresStore.Login writes only to admin_identities and
//     never creates a members or identity_bindings row for D-052 (ID01, §9.1).
func TestReview_F11_CrossPrincipalSessionOverwriteUnauditedAndAdminClaimBlocksMemberBinding(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	store := PostgresStore{Pool: pool}

	t.Run("cross_member_session_overwrite_misses_logout_audit", func(t *testing.T) {
		now := time.Now().UTC()
		idA := Identity{
			Issuer:      "https://idp.example.invalid",
			ClientScope: "portal",
			Subject:     "subject-member-a",
			Label:       "Member A",
			Role:        Member,
			AuthTime:    now,
			Expires:     now.Add(time.Hour),
		}
		idB := Identity{
			Issuer:      "https://idp.example.invalid",
			ClientScope: "portal",
			Subject:     "subject-member-b",
			Label:       "Member B",
			Role:        Member,
			AuthTime:    now,
			Expires:     now.Add(time.Hour),
		}

		tokenA := randomToken()
		principalA, err := store.Login(ctx, idA, tokenA, "", time.Hour, uuid.New())
		if err != nil {
			t.Fatal(err)
		}

		// Member B logs in on the same browser while tokenA is still in the cookie jar.
		tokenB := randomToken()
		principalB, err := store.Login(ctx, idB, tokenB, tokenA, time.Hour, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if principalA.ID == principalB.ID {
			t.Fatal("distinct subjects mapped to same member")
		}

		// Member A's session was deleted by Member B's login.
		if _, err := store.Session(ctx, tokenA, time.Minute); err == nil {
			t.Fatal("Member A session survived overwrite")
		}

		var logoutAudits int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='session.logout' AND actor=$1 AND after_diff->>'reason'='superseded'`, principalA.ID.String()).Scan(&logoutAudits); err != nil {
			t.Fatal(err)
		}
		if logoutAudits != 1 {
			t.Errorf("[F-11]: Member A session was terminated by Member B login on shared browser, but session.logout (reason=superseded) audit count for Member A = %d; want 1", logoutAudits)
		}
	})

	t.Run("admin_claim_forces_admin_role_and_prevents_d052_member_binding", func(t *testing.T) {
		// TEST ONLY - NOT A SECRET: ephemeral RSA key generated at test time.
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		var issuer, nonce, challenge string
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"issuer":                                issuer,
					"authorization_endpoint":                issuer + "/authorize",
					"token_endpoint":                        issuer + "/token",
					"jwks_uri":                              issuer + "/jwks",
					"id_token_signing_alg_values_supported": []string{"RS256"},
				})
			case "/jwks":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"keys": []any{map[string]any{
						"kty": "RSA",
						"alg": "RS256",
						"use": "sig",
						"kid": "TEST-ONLY",
						"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
						"e":   "AQAB",
					}},
				})
			case "/token":
				_ = r.ParseForm()
				sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
				if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				now := time.Now()
				claims := map[string]any{
					"iss":       issuer,
					"aud":       "portal-client",
					"sub":       "dual-role-dex-subject",
					"exp":       now.Add(time.Hour).Unix(),
					"iat":       now.Unix(),
					"auth_time": now.Unix(),
					"nonce":     nonce,
					"name":      "TEST ONLY Operator",
					"roles":     []string{"admin", "member"},
					"amr":       []string{"mfa"},
				}
				header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"TEST-ONLY"}`))
				body, _ := json.Marshal(claims)
				unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
				digest := sha256.Sum256([]byte(unsigned))
				sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
				jwt := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "TEST-ONLY-TOKEN",
					"token_type":   "Bearer",
					"id_token":     jwt,
				})
			}
		}))
		defer provider.Close()
		issuer = provider.URL

		b, err := NewBrowser(context.Background(), BrowserConfig{
			Issuer:     issuer,
			ClientID:   "portal-client",
			Origin:     "https://portal.example.invalid",
			AdminClaim: "roles",
			AdminValue: "admin",
			MFAAMR:     []string{"mfa"},
			Idle:       time.Minute,
			Absolute:   time.Hour,
			LoginTTL:   time.Minute,
			RecentAuth: time.Minute,
		}, store, provider.Client())
		if err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		b.Register(mux)

		loginRec := httptest.NewRecorder()
		mux.ServeHTTP(loginRec, httptest.NewRequest("GET", "/auth/login?role=member&return_to=/member/devices", nil))
		target, err := url.Parse(loginRec.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		nonce = target.Query().Get("nonce")
		challenge = target.Query().Get("code_challenge")
		state := target.Query().Get("state")

		cbReq := httptest.NewRequest("GET", "/auth/callback?code=test-code&state="+state, nil)
		for _, c := range loginRec.Result().Cookies() {
			cbReq.AddCookie(c)
		}
		cbRec := httptest.NewRecorder()
		mux.ServeHTTP(cbRec, cbReq)
		if cbRec.Code != http.StatusSeeOther {
			t.Fatalf("callback HTTP %d: %s", cbRec.Code, cbRec.Body.String())
		}

		var bindings int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM identity_bindings WHERE issuer=$1 AND subject='dual-role-dex-subject'`, issuer).Scan(&bindings); err != nil {
			t.Fatal(err)
		}
		if bindings != 1 {
			t.Errorf("[F-11]: dual-role user requesting /auth/login?role=member&return_to=/member/devices received %d identity_bindings rows (forced into Admin role); want 1 member identity_binding for D-052", bindings)
		}
	})
}

func TestReview_F04_CanonicalIssuerSubjectAcrossClientScopesAndSchemas(t *testing.T) {
	schemas := []struct {
		name                string
		issuerSubjectUnique bool
		mutableClientScope  bool
		expectUpdatedScope  bool
	}{
		{
			name:               "current_schema",
			expectUpdatedScope: true,
		},
		{
			name:                "migration_00183_issuer_subject_unique",
			issuerSubjectUnique: true,
			expectUpdatedScope:  true,
		},
		{
			name:                "migration_00183_with_mutable_client_scope",
			issuerSubjectUnique: true,
			mutableClientScope:  true,
			expectUpdatedScope:  true,
		},
	}

	for _, sc := range schemas {
		t.Run(sc.name, func(t *testing.T) {
			pool := dbtest.New(t)
			ctx := t.Context()
			// Migration 00183 creates identity_bindings_issuer_subject_key.
			// When the catalog already has it, exercise that schema and do not ALTER again.
			if sc.issuerSubjectUnique {
				var exists bool
				if err := pool.QueryRow(ctx, `SELECT EXISTS (
					SELECT 1 FROM pg_constraint
					WHERE conname = 'identity_bindings_issuer_subject_key'
					  AND conrelid = 'identity_bindings'::regclass
				)`).Scan(&exists); err != nil {
					t.Fatal(err)
				}
				if !exists {
					if _, err := pool.Exec(ctx, `ALTER TABLE identity_bindings ADD CONSTRAINT identity_bindings_issuer_subject_key UNIQUE (issuer, subject)`); err != nil {
						t.Fatal(err)
					}
				}
			}
			if sc.mutableClientScope {
				if _, err := pool.Exec(ctx, `
					DROP TRIGGER identity_bindings_identity ON identity_bindings;
					CREATE TRIGGER identity_bindings_identity BEFORE UPDATE ON identity_bindings FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','issuer','subject');
				`); err != nil {
					t.Fatal(err)
				}
			}
			store := PostgresStore{Pool: pool}
			now := time.Now().UTC()

			// 1. Client-scope rotation and issuer canonicalization keep the same member and leave 1 binding.
			sub := "f04-member-subject-" + sc.name
			idV1 := Identity{
				Issuer:      "https://idp.example.invalid/dex",
				ClientScope: "portal-v1",
				Subject:     sub,
				Label:       "F04 Member",
				Role:        Member,
				AuthTime:    now,
				Expires:     now.Add(time.Hour),
			}
			p1, err := store.Login(ctx, idV1, randomToken(), "", time.Hour, uuid.New())
			if err != nil {
				t.Fatal(err)
			}

			idV2 := idV1
			idV2.Issuer = "https://IDP.Example.Invalid:443/dex/"
			idV2.ClientScope = "portal-v2"
			p2, err := store.Login(ctx, idV2, randomToken(), "", time.Hour, uuid.New())
			if err != nil {
				t.Fatal(err)
			}
			if p2.ID != p1.ID {
				t.Fatalf("client_scope rotation created second member: got %s, want %s", p2.ID, p1.ID)
			}

			// Second login from the rotated client must keep the member/scope and must NOT append a second identity_binding.proof_updated audit event.
			p3, err := store.Login(ctx, idV2, randomToken(), "", time.Hour, uuid.New())
			if err != nil {
				t.Fatal(err)
			}
			if p3.ID != p1.ID {
				t.Fatalf("second rotated-client login changed member: got %s, want %s", p3.ID, p1.ID)
			}

			var bindingCount int
			var recordedScope string
			if err := pool.QueryRow(ctx, `SELECT count(*), min(client_scope) FROM identity_bindings WHERE subject=$1`, sub).Scan(&bindingCount, &recordedScope); err != nil {
				t.Fatal(err)
			}
			if bindingCount != 1 {
				t.Fatalf("expected 1 identity_binding for subject %q, got %d", sub, bindingCount)
			}
			if sc.expectUpdatedScope && recordedScope != "portal-v2" {
				t.Fatalf("expected client_scope updated to portal-v2, got %q", recordedScope)
			}

			// 2. Audit event on reused binding with rotated client_scope is emitted once for the rotation and stays on D-029 allowlist without raw subject/issuer.
			var proofAudits int
			var beforeRaw, afterRaw string
			if err := pool.QueryRow(ctx, `SELECT count(*), coalesce(min(before_diff::text),''), coalesce(min(after_diff::text),'') FROM audit_events WHERE action='identity_binding.proof_updated' AND actor=$1 AND resource=$2 AND result='succeeded' AND after_diff->>'reason'='proof_updated' AND coalesce(after_diff->>'identity_binding_id','') <> ''`, p1.ID.String(), "member:"+p1.ID.String()).Scan(&proofAudits, &beforeRaw, &afterRaw); err != nil {
				t.Fatal(err)
			}
			if proofAudits != 1 {
				t.Fatalf("expected 1 identity_binding.proof_updated audit event after two logins from rotated client, got %d", proofAudits)
			}
			t.Logf("schema=%s recorded_scope=%s proof_updated_audits=%d", sc.name, recordedScope, proofAudits)
			var afterMap map[string]any
			if err := json.Unmarshal([]byte(afterRaw), &afterMap); err != nil {
				t.Fatal(err)
			}
			for k := range afterMap {
				if k != "identity_binding_id" && k != "reason" {
					t.Fatalf("unexpected non-allowlisted key %q in identity_binding.proof_updated after_diff: %s", k, afterRaw)
				}
			}
			combinedAudit := beforeRaw + " " + afterRaw
			if strings.Contains(combinedAudit, sub) || strings.Contains(combinedAudit, "idp.example.invalid") || strings.Contains(combinedAudit, "portal-v") {
				t.Fatalf("identity_binding.proof_updated audit leaked raw identity fields: %s", combinedAudit)
			}

			// 3. Racing first-time logins across client scopes yield one member and one binding.
			raceSub := "f04-race-subject-" + sc.name
			const racers = 8
			start := make(chan struct{})
			var wg sync.WaitGroup
			memberIDs := make(chan uuid.UUID, racers)
			errs := make(chan error, racers)
			for i := 0; i < racers; i++ {
				scope := "portal-v1"
				if i%2 == 1 {
					scope = "portal-v2"
				}
				wg.Add(1)
				go func(clientScope string) {
					defer wg.Done()
					<-start
					p, err := store.Login(ctx, Identity{
						Issuer:      "https://idp.example.invalid/dex",
						ClientScope: clientScope,
						Subject:     raceSub,
						Label:       "Race Member",
						Role:        Member,
						AuthTime:    now,
						Expires:     now.Add(time.Hour),
					}, randomToken(), "", time.Hour, uuid.New())
					if err != nil {
						errs <- err
						return
					}
					memberIDs <- p.ID
				}(scope)
			}
			close(start)
			wg.Wait()
			close(memberIDs)
			close(errs)
			for err := range errs {
				t.Fatalf("concurrent login error: %v", err)
			}
			var raceMemberID uuid.UUID
			for id := range memberIDs {
				if raceMemberID == uuid.Nil {
					raceMemberID = id
				} else if id != raceMemberID {
					t.Fatalf("racing first logins created distinct members: %s vs %s", raceMemberID, id)
				}
			}
			var raceBindings int
			var raceScope string
			if err := pool.QueryRow(ctx, `SELECT count(*), min(client_scope) FROM identity_bindings WHERE subject=$1`, raceSub).Scan(&raceBindings, &raceScope); err != nil {
				t.Fatal(err)
			}
			if raceBindings != 1 {
				t.Fatalf("expected 1 binding after racing first logins, got %d", raceBindings)
			}

			// Also verify the unique-violation (23505) savepoint retry path directly when an insert collides with the existing unique key.
			collisionScope := raceScope
			if sc.issuerSubjectUnique {
				collisionScope = "portal-v3"
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			recoveredID, created, err := insertOrSelectMemberBinding(ctx, tx, Identity{
				Issuer:      "https://idp.example.invalid/dex",
				ClientScope: collisionScope,
				Subject:     raceSub,
				Label:       "Race Member",
				Role:        Member,
			}, "https://idp.example.invalid/dex", uuid.New())
			if err != nil {
				t.Fatalf("insertOrSelectMemberBinding on existing unique key failed: %v", err)
			}
			if created || recoveredID != raceMemberID {
				t.Fatalf("insertOrSelectMemberBinding = (%s, created=%v), want (%s, created=false)", recoveredID, created, raceMemberID)
			}

			// 4. Admin identities also resolve by canonical (issuer, subject) across client_scope rotation.
			adminSub := "f04-admin-subject-" + sc.name
			admin1, err := store.Login(ctx, Identity{
				Issuer:      "https://idp.example.invalid/dex",
				ClientScope: "portal-v1",
				Subject:     adminSub,
				Label:       "F04 Admin",
				Role:        Admin,
				MFA:         true,
				AuthTime:    now,
				Expires:     now.Add(time.Hour),
			}, randomToken(), "", time.Hour, uuid.New())
			if err != nil {
				t.Fatal(err)
			}
			admin2, err := store.Login(ctx, Identity{
				Issuer:      "https://IDP.Example.Invalid:443/dex/",
				ClientScope: "portal-v2",
				Subject:     adminSub,
				Label:       "F04 Admin",
				Role:        Admin,
				MFA:         true,
				AuthTime:    now,
				Expires:     now.Add(time.Hour),
			}, randomToken(), "", time.Hour, uuid.New())
			if err != nil {
				t.Fatal(err)
			}
			if admin2.ID != admin1.ID {
				t.Fatalf("admin client_scope rotation created second admin identity: %s vs %s", admin2.ID, admin1.ID)
			}
			var adminCount int
			var adminScope string
			if err := pool.QueryRow(ctx, `SELECT count(*), min(client_scope) FROM admin_identities WHERE subject=$1`, adminSub).Scan(&adminCount, &adminScope); err != nil {
				t.Fatal(err)
			}
			if adminCount != 1 || adminScope != "portal-v2" {
				t.Fatalf("admin_identities count=%d scope=%q, want count=1 scope=portal-v2", adminCount, adminScope)
			}
		})
	}
}
