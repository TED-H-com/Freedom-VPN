// Command fvpn-e2e-idp is the lab OIDC issuer for browser login (§14.2).
// An operator selects the next subject with POST /e2e/assume. The endpoint
// refuses callers that do not present the per-run secret, so a published
// IdP cannot mint an administrator by itself.
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", ":443", "listen address")
	cert := flag.String("cert", "", "TLS certificate")
	key := flag.String("key", "", "TLS private key")
	secretFile := flag.String("assume-secret-file", "", "per-run assume secret")
	issuer := flag.String("issuer", "https://idp.example.invalid", "issuer URL")
	clientID := flag.String("client-id", "portal-e2e", "expected client id")
	clientSecretFile := flag.String("client-secret-file", "", "OIDC client secret")
	redirect := flag.String("redirect", "https://portal.example.invalid:8443/auth/callback", "allowed redirect")
	flag.Parse()
	if *cert == "" || *key == "" || *secretFile == "" || *clientSecretFile == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-idp: cert, key and secret files are required")
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
		assume: assume, key: signer, kid: "e2e-1",
	}
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
	srv := &http.Server{
		Addr: *addr, Handler: mux,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second,
	}
	if err := srv.ListenAndServeTLS(*cert, *key); err != nil {
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
	next         *identity
	codes        map[string]authCode
	key          *rsa.PrivateKey
	kid          string
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
	pub := s.key.PublicKey
	writeJSON(w, http.StatusOK, map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": s.kid,
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}})
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
	s.mu.Lock()
	s.next = &body
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != s.clientID || q.Get("redirect_uri") != s.redirect || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || q.Get("state") == "" || q.Get("nonce") == "" {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	who := s.next
	s.next = nil
	s.mu.Unlock()
	if who == nil {
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
	s.codes[code] = authCode{identity: *who, Nonce: q.Get("nonce"), Challenge: q.Get("code_challenge"), RedirectURI: q.Get("redirect_uri"), Expires: time.Now().Add(5 * time.Minute)}
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
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if !found || time.Now().After(item.Expires) || subtle.ConstantTimeCompare([]byte(challenge), []byte(item.Challenge)) != 1 || r.PostForm.Get("redirect_uri") != item.RedirectURI {
		http.Error(w, "invalid grant\n", http.StatusBadRequest)
		return
	}
	token, err := s.idToken(item)
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

func (s *server) idToken(item authCode) (string, error) {
	now := time.Now().UTC()
	roles := []string{"member"}
	if item.Role == "admin" {
		roles = []string{"admin"}
	}
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": s.kid})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]any{
		"iss": s.issuer, "sub": item.Subject, "aud": s.clientID,
		"iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "auth_time": now.Unix(),
		"nonce": item.Nonce, "amr": []string{"mfa"}, "roles": roles, "name": item.Name,
	})
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
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
