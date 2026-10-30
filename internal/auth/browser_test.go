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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func counterValue(t *testing.T, reg *prometheus.Registry, class, limiter string) float64 {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() != "fvpn_control_rate_limited_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			var gotClass, gotLimiter string
			for _, lp := range m.GetLabel() {
				switch lp.GetName() {
				case "class":
					gotClass = lp.GetValue()
				case "limiter":
					gotLimiter = lp.GetValue()
				}
			}
			if gotClass == class && gotLimiter == limiter {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

type memoryStore struct {
	mu         sync.Mutex
	flows      map[string]Flow
	sessions   map[string]Principal
	identities []Identity
	rejections []string
}

func (s *memoryStore) SaveFlow(_ context.Context, f Flow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flows[f.State] = f
	return nil
}
func (s *memoryStore) AuditLoginRejection(_ context.Context, reason string, _ uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rejections = append(s.rejections, reason)
	return nil
}
func (s *memoryStore) ConsumeFlow(_ context.Context, state, browser string) (Flow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flows[state]
	if !ok || f.Browser != browser || !f.Expires.After(time.Now()) {
		return Flow{}, pgx.ErrNoRows
	}
	delete(s.flows, state)
	return f, nil
}
func (s *memoryStore) Login(_ context.Context, i Identity, token, old string, _ time.Duration, _ uuid.UUID) (Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, old)
	p := Principal{Role: i.Role, ID: uuid.New(), DisplayLabel: i.Label, MFAVerified: i.MFA, AuthenticatedAt: i.AuthTime}
	s.sessions[token] = p
	s.identities = append(s.identities, i)
	return p, nil
}
func (s *memoryStore) Session(_ context.Context, token string, _ time.Duration) (Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.sessions[token]
	if !ok {
		return p, pgx.ErrNoRows
	}
	return p, nil
}
func (s *memoryStore) Logout(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
			oldRole := Member
			if mode == "admin" {
				oldRole = Admin
			}
			callback.AddCookie(&http.Cookie{Name: cookieNameForRole(oldRole), Value: "old-session"})
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
				if c.Name == cookieNameForRole(oldRole) {
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

func TestSEC05_SubjectLoginLimitAndRequestedRole(t *testing.T) {
	// TEST ONLY - NOT A SECRET: ephemeral RSA key generated at test time.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	var tokenMu sync.Mutex
	noncesByCode := map[string]string{}
	challengesByCode := map[string]string{}
	subjectsByCode := map[string]string{}
	rolesByCode := map[string][]string{}

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
			code := r.Form.Get("code")
			tokenMu.Lock()
			nonce := noncesByCode[code]
			challenge := challengesByCode[code]
			sub := subjectsByCode[code]
			roles := rolesByCode[code]
			tokenMu.Unlock()
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			now := time.Now()
			claims := map[string]any{
				"iss":       issuer,
				"aud":       "portal-client",
				"sub":       sub,
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"auth_time": now.Unix(),
				"nonce":     nonce,
				"name":      "TEST ONLY Member",
				"amr":       []string{"mfa"},
				"roles":     roles,
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

	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fvpn_control_rate_limited_total",
		Help: "TEST ONLY rate limited counter.",
	}, []string{"class", "limiter"})
	if err := reg.Register(counter); err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{flows: map[string]Flow{}, sessions: map[string]Principal{}}
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
		// Default SubjectPerMinute (10)
		RateLimited: counter,
	}, store, provider.Client())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	b.Register(mux)

	completeLogin := func(loginPath, subject string, roles []string) *httptest.ResponseRecorder {
		loginRec := httptest.NewRecorder()
		mux.ServeHTTP(loginRec, httptest.NewRequest("GET", loginPath, nil))
		if loginRec.Code != http.StatusFound {
			t.Fatalf("login(%s) status = %d", loginPath, loginRec.Code)
		}
		target, err := url.Parse(loginRec.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		code := uuid.NewString()
		tokenMu.Lock()
		noncesByCode[code] = target.Query().Get("nonce")
		challengesByCode[code] = target.Query().Get("code_challenge")
		subjectsByCode[code] = subject
		rolesByCode[code] = roles
		tokenMu.Unlock()

		cbReq := httptest.NewRequest("GET", "/auth/callback?code="+code+"&state="+target.Query().Get("state"), nil)
		for _, c := range loginRec.Result().Cookies() {
			cbReq.AddCookie(c)
		}
		cbRec := httptest.NewRecorder()
		mux.ServeHTTP(cbRec, cbReq)
		return cbRec
	}

	// F-11: requesting admin role without the admin claim is refused with 403 Forbidden.
	if rec := completeLogin("/auth/login?role=admin", "member-only-subject", []string{"member"}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when admin role requested without claim, got %d: %s", rec.Code, rec.Body.String())
	}

	// D-060: 10 completions for one subject succeed; the 11th is refused with 429 + Retry-After,
	// while another subject from the same address still succeeds.
	for i := 1; i <= 10; i++ {
		if rec := completeLogin("/auth/login?role=member", "subject-repeat", []string{"member"}); rec.Code != http.StatusSeeOther {
			t.Fatalf("completion %d for subject-repeat failed with %d: %s", i, rec.Code, rec.Body.String())
		}
	}
	eleventh := completeLogin("/auth/login?role=member", "subject-repeat", []string{"member"})
	if eleventh.Code != http.StatusTooManyRequests || eleventh.Header().Get("Retry-After") == "" || !strings.Contains(eleventh.Body.String(), `"code":"RATE_LIMITED"`) {
		t.Fatalf("expected 11th completion for subject-repeat to return 429 with Retry-After and RATE_LIMITED, got %d headers=%v body=%s", eleventh.Code, eleventh.Header(), eleventh.Body.String())
	}
	if got := counterValue(t, reg, "login", "subject"); got != 1 {
		t.Fatalf("rate_limited_total{class=login,limiter=subject} = %v, want 1", got)
	}
	if other := completeLogin("/auth/login?role=member", "subject-other", []string{"member"}); other.Code != http.StatusSeeOther {
		t.Fatalf("other subject should proceed while subject-repeat is rate-limited, got %d: %s", other.Code, other.Body.String())
	}
}
