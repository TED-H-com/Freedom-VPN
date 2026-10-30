// Command fvpn-e2e-idp is the lab helper for browser login (§14.2).
//
// POST /e2e/assume stores one identity per caller. The ticket is the
// e2e_login cookie and the X-E2E-Ticket header. Authorize reads that
// ticket and does not keep a process-wide next subject (SECOPS-06).
//
// When -dex-upstream is set, POST /oauth2/token and GET /oauth2/keys adapt
// the embedded Dex issuer for the portal client. Dex stays the
// authorization server. The adapter re-signs only the portal ID token,
// keeps iss and sub, and adds TEST ONLY roles and amr. Every other grant,
// including the CLI device-code grant, is proxied unchanged.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", ":443", "TLS listen address")
	httpAddr := flag.String("http-addr", "", "optional plain HTTP listen address")
	cert := flag.String("cert", "", "TLS certificate")
	key := flag.String("key", "", "TLS private key")
	secretFile := flag.String("assume-secret-file", "", "per-run assume secret")
	issuer := flag.String("issuer", "https://idp.example.invalid", "standalone issuer URL")
	clientID := flag.String("client-id", "portal-e2e", "standalone client id")
	clientSecretFile := flag.String("client-secret-file", "", "OIDC client secret")
	redirect := flag.String("redirect", "https://portal.example.invalid:8443/auth/callback", "portal redirect")
	dexUpstream := flag.String("dex-upstream", "", "embedded Dex base URL, for example http://netbird:8080")
	embedIssuer := flag.String("embed-issuer", "", "embedded issuer string copied onto portal ID tokens")
	portalClient := flag.String("portal-client-id", "", "public client id the portal uses")
	dexHost := flag.String("dex-host", "netbird.example.invalid", "Host header for Dex upstream requests")
	adminEmail := flag.String("admin-email", "admin@example.invalid", "profile value that receives the admin role")
	adminName := flag.String("admin-name", "E2E Admin", "profile name that receives the admin role")
	flag.Parse()
	if *cert == "" || *key == "" || *secretFile == "" || *clientSecretFile == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: cert, key and secret files are required")
		os.Exit(2)
	}
	if *dexUpstream != "" && (*embedIssuer == "" || *portalClient == "") {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: dex upstream requires embed issuer and portal client id")
		os.Exit(2)
	}
	assume, err := readTrim(*secretFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: assume secret")
		os.Exit(1)
	}
	clientSecret, err := readTrim(*clientSecretFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: client secret")
		os.Exit(1)
	}
	signer, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: key generation failed")
		os.Exit(1)
	}
	s := &server{
		issuer: *issuer, clientID: *clientID, clientSecret: clientSecret, redirect: *redirect,
		assume: assume, key: signer, kid: "e2e-1", portalKid: "e2e-portal",
		dexUpstream: strings.TrimRight(*dexUpstream, "/"), embedIssuer: *embedIssuer,
		portalClient: *portalClient, dexHost: *dexHost, adminEmail: *adminEmail, adminName: *adminName,
		dexClient: &http.Client{Timeout: 10 * time.Second},
	}
	tlsSrv := &http.Server{
		Addr: *addr, Handler: s.routes(),
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second,
	}
	if *httpAddr != "" {
		plain := &http.Server{
			Addr: *httpAddr, Handler: s.routes(),
			ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second,
		}
		go func() {
			if err := plain.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: http listener stopped")
				os.Exit(1)
			}
		}()
	}
	if err := tlsSrv.ListenAndServeTLS(*cert, *key); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: stopped")
		os.Exit(1)
	}
}

type identity struct {
	Subject string
	Role    string
	Name    string
}

type authCode struct {
	identity
	Nonce       string
	Challenge   string
	RedirectURI string
	Expires     time.Time
}

type server struct {
	mu           sync.Mutex
	issuer       string
	clientID     string
	clientSecret string
	redirect     string
	assume       string
	sessions     map[string]identity
	codes        map[string]authCode
	key          *rsa.PrivateKey
	kid          string
	portalKid    string
	dexUpstream  string
	embedIssuer  string
	portalClient string
	dexHost      string
	adminEmail   string
	adminName    string
	dexClient    *http.Client
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /.well-known/openid-configuration", s.discovery)
	mux.HandleFunc("GET /jwks", s.jwks)
	mux.HandleFunc("GET /authorize", s.authorize)
	mux.HandleFunc("POST /token", s.token)
	mux.HandleFunc("POST /e2e/assume", s.assumeNext)
	if s.dexUpstream != "" {
		mux.HandleFunc("POST /oauth2/token", s.oauthToken)
		mux.HandleFunc("GET /oauth2/keys", s.oauthKeys)
	}
	return mux
}

func (s *server) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/authorize",
		"token_endpoint":                        s.issuer + "/token",
		"jwks_uri":                              s.issuer + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "nonce", "auth_time", "amr", "roles", "name"},
	})
}

func (s *server) jwks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"keys": []any{s.publicJWK(s.kid)}})
}

func (s *server) assumeNext(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-E2E-Secret")), []byte(s.assume)) != 1 {
		http.Error(w, "unauthorized\n", http.StatusUnauthorized)
		return
	}
	var body identity
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body) != nil || body.Subject == "" || (body.Role != "admin" && body.Role != "member") {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		body.Name = body.Role
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		http.Error(w, "unavailable\n", http.StatusServiceUnavailable)
		return
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)
	s.mu.Lock()
	if s.sessions == nil {
		s.sessions = map[string]identity{}
	}
	if len(s.sessions) >= 256 {
		s.mu.Unlock()
		http.Error(w, "too many pending logins\n", http.StatusTooManyRequests)
		return
	}
	s.sessions[ticket] = body
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: "e2e_login", Value: ticket, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("X-E2E-Ticket", ticket)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != s.clientID || q.Get("redirect_uri") != s.redirect || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || q.Get("state") == "" || q.Get("nonce") == "" {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	ticket := r.Header.Get("X-E2E-Ticket")
	if c, err := r.Cookie("e2e_login"); err == nil && c.Value != "" && ticket == "" {
		ticket = c.Value
	}
	s.mu.Lock()
	who, ok := s.sessions[ticket]
	if ok {
		delete(s.sessions, ticket)
	}
	s.mu.Unlock()
	if !ok {
		http.Error(w, "no subject assumed\n", http.StatusBadRequest)
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		http.Error(w, "unavailable\n", http.StatusServiceUnavailable)
		return
	}
	code := base64.RawURLEncoding.EncodeToString(raw)
	s.mu.Lock()
	if s.codes == nil {
		s.codes = map[string]authCode{}
	}
	s.codes[code] = authCode{identity: who, Nonce: q.Get("nonce"), Challenge: q.Get("code_challenge"), RedirectURI: q.Get("redirect_uri"), Expires: time.Now().Add(5 * time.Minute)}
	s.mu.Unlock()
	http.Redirect(w, r, s.redirect+"?code="+code+"&state="+q.Get("state"), http.StatusFound)
}

func (s *server) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	if id != s.clientID || subtle.ConstantTimeCompare([]byte(secret), []byte(s.clientSecret)) != 1 {
		http.Error(w, "unauthorized\n", http.StatusUnauthorized)
		return
	}
	code := r.PostForm.Get("code")
	s.mu.Lock()
	item, found := s.codes[code]
	if found {
		delete(s.codes, code)
	}
	s.mu.Unlock()
	sum := sha256Sum(r.PostForm.Get("code_verifier"))
	if !found || time.Now().After(item.Expires) || subtle.ConstantTimeCompare([]byte(sum), []byte(item.Challenge)) != 1 || r.PostForm.Get("redirect_uri") != item.RedirectURI {
		http.Error(w, "invalid grant\n", http.StatusBadRequest)
		return
	}
	token, err := s.standaloneToken(item)
	if err != nil {
		http.Error(w, "unavailable\n", http.StatusServiceUnavailable)
		return
	}
	access := make([]byte, 16)
	_, _ = rand.Read(access)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": base64.RawURLEncoding.EncodeToString(access),
		"token_type":   "Bearer",
		"expires_in":   300,
		"id_token":     token,
	})
}

func (s *server) standaloneToken(item authCode) (string, error) {
	now := time.Now().UTC()
	roles := []string{"member"}
	if item.Role == "admin" {
		roles = []string{"admin"}
	}
	return signRS256(s.key, s.kid, map[string]any{
		"iss": s.issuer, "sub": item.Subject, "aud": s.clientID,
		"iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "auth_time": now.Unix(),
		"nonce": item.Nonce, "amr": []string{"mfa"}, "roles": roles, "name": item.Name,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", errors.New("empty")
	}
	return s, nil
}
