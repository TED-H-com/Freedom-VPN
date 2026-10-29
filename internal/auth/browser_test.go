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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type memoryStore struct {
	flows      map[string]Flow
	sessions   map[string]Principal
	identities []Identity
}

func (s *memoryStore) SaveFlow(_ context.Context, f Flow) error { s.flows[f.State] = f; return nil }
func (s *memoryStore) ConsumeFlow(_ context.Context, state, browser string) (Flow, error) {
	f, ok := s.flows[state]
	if !ok || f.Browser != browser || !f.Expires.After(time.Now()) {
		return Flow{}, pgx.ErrNoRows
	}
	delete(s.flows, state)
	return f, nil
}
func (s *memoryStore) Login(_ context.Context, i Identity, token, old string, _ time.Duration, _ uuid.UUID) (Principal, error) {
	delete(s.sessions, old)
	p := Principal{Role: i.Role, ID: uuid.New(), DisplayLabel: i.Label, MFAVerified: i.MFA, AuthenticatedAt: i.AuthTime}
	s.sessions[token] = p
	s.identities = append(s.identities, i)
	return p, nil
}
func (s *memoryStore) Session(_ context.Context, token string, _ time.Duration) (Principal, error) {
	p, ok := s.sessions[token]
	if !ok {
		return p, pgx.ErrNoRows
	}
	return p, nil
}
func (s *memoryStore) Logout(_ context.Context, token string) error {
	delete(s.sessions, token)
	return nil
}
func TestAT28_OIDCVerificationAndSessions(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"member", "admin", "no-mfa", "issuer", "audience", "expired", "signature", "nonce", "state", "pkce", "future-auth"} {
		t.Run(mode, func(t *testing.T) {
			var issuer, nonce, challenge string
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/.well-known/openid-configuration":
					_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}})
				case "/jwks":
					_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "TEST-ONLY", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
				case "/token":
					_ = r.ParseForm()
					h := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
					if base64.RawURLEncoding.EncodeToString(h[:]) != challenge || mode == "pkce" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
						return
					}
					now := time.Now()
					claims := map[string]any{"iss": issuer, "aud": "test-client", "sub": "opaque-subject", "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "auth_time": now.Unix(), "nonce": nonce, "name": "Test Member", "amr": []string{"mfa"}}
					switch mode {
					case "admin", "no-mfa", "future-auth":
						claims["roles"] = []string{"admin"}
					}
					switch mode {
					case "no-mfa":
						claims["amr"] = []string{"pwd"}
					case "issuer":
						claims["iss"] = "https://wrong.example.invalid"
					case "audience":
						claims["aud"] = "other-client"
					case "expired":
						claims["exp"] = now.Add(-time.Hour).Unix()
					case "nonce":
						claims["nonce"] = "wrong-nonce"
					case "future-auth":
						claims["auth_time"] = now.Add(time.Hour).Unix()
					}
					header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"TEST-ONLY"}`))
					body, _ := json.Marshal(claims)
					unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
					digest := sha256.Sum256([]byte(unsigned))
					signature, e := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
					if e != nil {
						t.Error(e)
						return
					}
					if mode == "signature" {
						signature[0] ^= 0xff
					}
					jwt := unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
					_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "TEST-ONLY-TOKEN", "token_type": "Bearer", "id_token": jwt})
				default:
					w.WriteHeader(404)
				}
			}))
			defer provider.Close()
			issuer = provider.URL
			store := &memoryStore{flows: map[string]Flow{}, sessions: map[string]Principal{"old-session": {Role: Member, ID: uuid.New()}}}
			b, e := NewBrowser(context.Background(), BrowserConfig{Issuer: issuer, ClientID: "test-client", Origin: "https://portal.example.invalid", AdminClaim: "roles", AdminValue: "admin", MFAAMR: []string{"mfa"}, Idle: time.Minute, Absolute: time.Hour, LoginTTL: time.Minute, RecentAuth: time.Minute}, store, provider.Client())
			if e != nil {
				t.Fatal(e)
			}
			mux := http.NewServeMux()
			b.Register(mux)
			login := httptest.NewRecorder()
			mux.ServeHTTP(login, httptest.NewRequest("GET", "/auth/login", nil))
			if login.Code != 302 {
				t.Fatal(login.Code)
			}
			target, e := url.Parse(login.Header().Get("Location"))
			if e != nil {
				t.Fatal(e)
			}
			nonce = target.Query().Get("nonce")
			challenge = target.Query().Get("code_challenge")
			if target.Query().Get("code_challenge_method") != "S256" || nonce == "" {
				t.Fatal("PKCE/nonce missing")
			}
			state := target.Query().Get("state")
			if mode == "state" {
				state = "wrong-state"
			}
			callback := httptest.NewRequest("GET", "/auth/callback?code=test-code&state="+state, nil)
			for _, c := range login.Result().Cookies() {
				if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
					t.Fatal("insecure flow cookie")
				}
				callback.AddCookie(c)
			}
			callback.AddCookie(&http.Cookie{Name: SessionCookie, Value: "old-session"})
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, callback)
			want := 401
			if mode == "no-mfa" {
				want = 403
			}
			if mode == "member" || mode == "admin" {
				want = 303
			}
			if w.Code != want {
				t.Fatalf("callback %d want %d: %s", w.Code, want, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cacheable callback")
			}
			if want != 303 {
				if len(store.identities) != 0 {
					t.Fatal("invalid token created identity")
				}
				return
			}
			if _, ok := store.sessions["old-session"]; ok {
				t.Fatal("session not rotated")
			}
			var sessionCookie *http.Cookie
			for _, c := range w.Result().Cookies() {
				if c.Name == SessionCookie {
					sessionCookie = c
				}
			}
			if sessionCookie == nil || !sessionCookie.Secure || !sessionCookie.HttpOnly {
				t.Fatal("session cookie")
			}
			r := httptest.NewRequest("GET", "/auth/session", nil)
			r.AddCookie(sessionCookie)
			bootstrap := httptest.NewRecorder()
			mux.ServeHTTP(bootstrap, r)
			if bootstrap.Code != 200 || bootstrap.Header().Get("Cache-Control") != "no-store" || bootstrap.Header().Get("X-CSRF-Token") == "" {
				t.Fatal("bootstrap failed")
			}
			var data map[string]any
			if json.Unmarshal(bootstrap.Body.Bytes(), &data) != nil || data["authenticated"] != true || data["role"] != mode {
				t.Fatal(bootstrap.Body.String())
			}
			if mode == "admin" && data["member_id"] != nil {
				t.Fatal("admin contains member ID")
			}
			if mode == "member" && data["member_id"] == nil {
				t.Fatal("missing member ID")
			}
			replay := httptest.NewRecorder()
			mux.ServeHTTP(replay, callback)
			if replay.Code != 401 {
				t.Fatal("state replay accepted")
			}
			logout := httptest.NewRequest("POST", "/auth/logout", nil)
			logout.AddCookie(sessionCookie)
			logout.Header.Set("Origin", "https://portal.example.invalid")
			logout.Header.Set("X-CSRF-Token", bootstrap.Header().Get("X-CSRF-Token"))
			out := httptest.NewRecorder()
			mux.ServeHTTP(out, logout)
			if out.Code != 204 {
				t.Fatal("logout", out.Code)
			}
			out = httptest.NewRecorder()
			mux.ServeHTTP(out, r)
			if !strings.Contains(out.Body.String(), `"authenticated":false`) {
				t.Fatal("session survived logout")
			}
		})
	}
}
