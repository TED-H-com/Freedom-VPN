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
