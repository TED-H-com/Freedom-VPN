package control

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type boundListener struct{ name, address string }
type listenerLog struct {
	slog.Handler
	bound chan boundListener
}

func (h listenerLog) Handle(ctx context.Context, record slog.Record) error {
	if record.Message == "control listener started" {
		var listener boundListener
		record.Attrs(func(a slog.Attr) bool {
			switch a.Key {
			case "listener":
				listener.name = a.Value.String()
			case "address":
				listener.address = a.Value.String()
			}
			return true
		})
		h.bound <- listener
	}
	return h.Handler.Handle(ctx, record)
}

func metricsConfig(t *testing.T) (config.Config, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.PublicAddress, c.AgentAddress, c.MetricsAddress = "127.0.0.1:0", "127.0.0.1:0", "127.0.0.1:0"
	c.ShutdownSeconds = 2
	dir := t.TempDir()
	c.TLSCertificateFile, c.TLSKeyFile = filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	c.AgentCAFile = c.TLSCertificateFile
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(c.TLSCertificateFile, cert, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.TLSKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), 0600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(cert) {
		t.Fatal("invalid test certificate")
	}
	return c, roots
}

func TestAPI01_ControlMetricsListenerLifecycle(t *testing.T) {
	c, roots := metricsConfig(t)
	registry := prometheus.NewRegistry()
	logs := listenerLog{Handler: slog.NewTextHandler(io.Discard, nil), bound: make(chan boundListener, 3)}
	app, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: &auth.Browser{}, Metrics: registry, Logger: slog.New(logs)}, Domain{Name: "TEST-metrics", Build: func(_ context.Context, d Dependencies) (Built, error) {
		if d.Metrics != registry {
			t.Fatal("domain received a different registry")
		}
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "fvpn_test_domain_value", Help: "TEST ONLY domain collector."})
		gauge.Set(7)
		return Built{}, d.Metrics.Register(gauge)
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	var runErr error
	go func() { runErr = Run(ctx, app); close(done) }()
	stop := func() {
		cancel()
		select {
		case <-done:
			if runErr != nil {
				t.Error(runErr)
			}
		case <-time.After(5 * time.Second):
			t.Error("control listeners did not stop")
		}
	}
	t.Cleanup(stop)
	addresses := map[string]string{}
	for len(addresses) < 3 {
		select {
		case bound := <-logs.bound:
			addresses[bound.name] = bound.address
		case <-done:
			t.Fatal("application stopped before binding listeners", runErr)
		case <-time.After(5 * time.Second):
			t.Fatal("missing bound listener log")
		}
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	response, err := client.Get("https://" + addresses["public"] + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal("public listener unavailable", response.Status)
	}
	response, err = client.Get("http://" + addresses["metrics"] + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.StatusCode != http.StatusOK || err != nil || contentType != "text/plain" {
		t.Fatal("invalid metrics response", response.Status, response.Header, err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []string{"fvpn_test_domain_value 7", `fvpn_control_http_requests_total{route="GET /healthz",status="200"} 1`} {
		if !strings.Contains(string(body), sample) {
			t.Fatalf("missing metric %q in %s", sample, body)
		}
	}
	stop()
	for name, address := range addresses {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = conn.Close()
			t.Errorf("%s listener still accepts connections", name)
		}
	}
}

func TestSEC02_ControlMetricsLoopbackAndBindFailure(t *testing.T) {
	c, _ := metricsConfig(t)
	for _, address := range []string{"0.0.0.0:9091", "[::]:9091", "192.0.2.1:9091", ":9091", "invalid"} {
		c.MetricsAddress = address
		if _, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: &auth.Browser{}}, []Domain{}...); err == nil || !strings.Contains(err.Error(), "metrics_address") {
			t.Fatalf("unsafe metrics address %q: %v", address, err)
		}
	}
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	c.MetricsAddress = busy.Addr().String()
	var started atomic.Bool
	app, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: &auth.Browser{}}, Domain{Name: "TEST-runner", Build: func(context.Context, Dependencies) (Built, error) {
		return Built{Runners: []func(context.Context) error{func(context.Context) error { started.Store(true); return nil }}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(t.Context(), app); err == nil || started.Load() {
		t.Fatal("metrics bind failure must prevent runner startup", err, started.Load())
	}
}

type memAuthStore struct {
	mu       sync.Mutex
	flows    map[string]auth.Flow
	sessions map[string]auth.Principal
}

func (s *memAuthStore) SaveFlow(_ context.Context, f auth.Flow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flows[f.State] = f
	return nil
}
func (s *memAuthStore) ConsumeFlow(_ context.Context, state, browser string) (auth.Flow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flows[state]
	if !ok || f.Browser != browser || !f.Expires.After(time.Now()) {
		return auth.Flow{}, pgx.ErrNoRows
	}
	delete(s.flows, state)
	return f, nil
}
func (s *memAuthStore) Login(_ context.Context, i auth.Identity, token, old string, _ time.Duration, _ uuid.UUID) (auth.Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, old)
	p := auth.Principal{Role: i.Role, ID: uuid.New(), DisplayLabel: i.Label, MFAVerified: i.MFA, AuthenticatedAt: i.AuthTime}
	s.sessions[token] = p
	return p, nil
}
func (s *memAuthStore) Session(_ context.Context, token string, _ time.Duration) (auth.Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.sessions[token]
	if !ok {
		return p, pgx.ErrNoRows
	}
	return p, nil
}
func (s *memAuthStore) Logout(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

func TestAT31_D060_ComposedConcurrentSignInAndSubjectLimiter(t *testing.T) {
	// TEST ONLY - NOT A SECRET: ephemeral RSA key generated at test time.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	var idpMu sync.Mutex
	noncesByCode := map[string]string{}
	challengesByCode := map[string]string{}
	subjectsByCode := map[string]string{}

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					"n":   base64.RawURLEncoding.EncodeToString(rsaKey.N.Bytes()),
					"e":   "AQAB",
				}},
			})
		case "/token":
			_ = r.ParseForm()
			code := r.Form.Get("code")
			idpMu.Lock()
			nonce := noncesByCode[code]
			challenge := challengesByCode[code]
			sub := subjectsByCode[code]
			idpMu.Unlock()
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
			}
			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"TEST-ONLY"}`))
			body, _ := json.Marshal(claims)
			unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
			digest := sha256.Sum256([]byte(unsigned))
			sig, _ := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
			jwt := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "TEST-ONLY-TOKEN",
				"token_type":   "Bearer",
				"id_token":     jwt,
			})
		}
	}))
	defer idp.Close()
	issuer = idp.URL

	c, _ := metricsConfig(t)
	// Use the default Burst=40 with a slow refill so the 41st login request is
	// deterministically inside one refill window even on loaded CI hosts.
	c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: config.Defaults().LoginRate.Burst}
	store := &memAuthStore{flows: map[string]auth.Flow{}, sessions: map[string]auth.Principal{}}
	browser, err := auth.NewBrowser(t.Context(), auth.BrowserConfig{
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
	}, store, idp.Client())
	if err != nil {
		t.Fatal(err)
	}
	registry := prometheus.NewRegistry()
	app, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: browser, Metrics: registry}, core())
	if err != nil {
		t.Fatal(err)
	}

	doSignIn := func(subject string) (int, http.Header, string) {
		loginReq := httptest.NewRequest("GET", "/auth/login", nil)
		loginReq.RemoteAddr = "198.51.100.25:50000"
		loginRec := httptest.NewRecorder()
		app.Public.Handler.ServeHTTP(loginRec, loginReq)
		if loginRec.Code != http.StatusFound {
			return loginRec.Code, loginRec.Header(), loginRec.Body.String()
		}
		target, err := url.Parse(loginRec.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		code := uuid.NewString()
		idpMu.Lock()
		noncesByCode[code] = target.Query().Get("nonce")
		challengesByCode[code] = target.Query().Get("code_challenge")
		subjectsByCode[code] = subject
		idpMu.Unlock()

		cbReq := httptest.NewRequest("GET", "/auth/callback?code="+code+"&state="+target.Query().Get("state"), nil)
		cbReq.RemoteAddr = "198.51.100.25:50000"
		for _, ck := range loginRec.Result().Cookies() {
			cbReq.AddCookie(ck)
		}
		cbRec := httptest.NewRecorder()
		app.Public.Handler.ServeHTTP(cbRec, cbReq)
		if cbRec.Code != http.StatusSeeOther {
			return cbRec.Code, cbRec.Header(), cbRec.Body.String()
		}
		sessReq := httptest.NewRequest("GET", "/auth/session?role=member", nil)
		sessReq.RemoteAddr = "198.51.100.25:50000"
		for _, ck := range cbRec.Result().Cookies() {
			if ck.Value != "" {
				sessReq.AddCookie(ck)
			}
		}
		sessRec := httptest.NewRecorder()
		app.Public.Handler.ServeHTTP(sessRec, sessReq)
		return sessRec.Code, sessRec.Header(), sessRec.Body.String()
	}

	// 1. 20 concurrent sign-ins from one source address all complete.
	const concurrent = 20
	barrier := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, concurrent)
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			status, _, body := doSignIn(fmt.Sprintf("member-%02d", idx))
			if status != http.StatusOK || !strings.Contains(body, `"authenticated":true`) {
				errs <- fmt.Errorf("concurrent sign-in %d failed: status=%d body=%s", idx, status, body)
			}
		}(i)
	}
	close(barrier)
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}

	// 2. The 41st login/callback request inside one refill window is refused with 429 and Retry-After.
	req41 := httptest.NewRequest("GET", "/auth/login", nil)
	req41.RemoteAddr = "198.51.100.25:50000"
	rec41 := httptest.NewRecorder()
	app.Public.Handler.ServeHTTP(rec41, req41)
	if rec41.Code != http.StatusTooManyRequests || rec41.Header().Get("Retry-After") == "" || !strings.Contains(rec41.Body.String(), `"code":"RATE_LIMITED"`) {
		t.Fatalf("41st login request: status=%d headers=%v body=%s", rec41.Code, rec41.Header(), rec41.Body.String())
	}

	// 3. Raise LoginRate as in a lab and verify the per-subject limit (10/min) refuses
	// the 11th completion for one subject while another subject from the same address proceeds.
	c2 := c
	c2.LoginRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	app2, err := Servers(t.Context(), Dependencies{Config: c2, Pool: &pgxpool.Pool{}, Browser: browser, Metrics: prometheus.NewRegistry()}, core())
	if err != nil {
		t.Fatal(err)
	}
	// Reset rateLimited counter back to the first registry so both address and subject increments are scraped together.
	browser.SetRateLimit(c2.LoginSubjectPerMinute, c2.MaxRatePrincipals, nil)
	doSignInApp2 := func(subject string) (int, http.Header, string) {
		loginReq := httptest.NewRequest("GET", "/auth/login", nil)
		loginReq.RemoteAddr = "198.51.100.25:50000"
		loginRec := httptest.NewRecorder()
		app2.Public.Handler.ServeHTTP(loginRec, loginReq)
		if loginRec.Code != http.StatusFound {
			return loginRec.Code, loginRec.Header(), loginRec.Body.String()
		}
		target, err := url.Parse(loginRec.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		code := uuid.NewString()
		idpMu.Lock()
		noncesByCode[code] = target.Query().Get("nonce")
		challengesByCode[code] = target.Query().Get("code_challenge")
		subjectsByCode[code] = subject
		idpMu.Unlock()

		cbReq := httptest.NewRequest("GET", "/auth/callback?code="+code+"&state="+target.Query().Get("state"), nil)
		cbReq.RemoteAddr = "198.51.100.25:50000"
		for _, ck := range loginRec.Result().Cookies() {
			cbReq.AddCookie(ck)
		}
		cbRec := httptest.NewRecorder()
		app2.Public.Handler.ServeHTTP(cbRec, cbReq)
		return cbRec.Code, cbRec.Header(), cbRec.Body.String()
	}

	for i := 1; i <= 10; i++ {
		if status, _, body := doSignInApp2("subject-burst"); status != http.StatusSeeOther {
			t.Fatalf("completion %d for subject-burst failed: status=%d body=%s", i, status, body)
		}
	}
	status11, hdr11, body11 := doSignInApp2("subject-burst")
	if status11 != http.StatusTooManyRequests || hdr11.Get("Retry-After") == "" || !strings.Contains(body11, `"code":"RATE_LIMITED"`) {
		t.Fatalf("11th completion for subject-burst: status=%d headers=%v body=%s", status11, hdr11, body11)
	}
	if statusOther, _, bodyOther := doSignInApp2("subject-neighbor"); statusOther != http.StatusSeeOther {
		t.Fatalf("other subject from same address failed: status=%d body=%s", statusOther, bodyOther)
	}

	// 4. Verify metrics scrape on both registries.
	metricsRec1 := httptest.NewRecorder()
	app.Metrics.Handler.ServeHTTP(metricsRec1, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(metricsRec1.Body.String(), `fvpn_control_rate_limited_total{class="login",limiter="address"} 1`) {
		t.Fatalf("missing address rate limit metric in:\n%s", metricsRec1.Body.String())
	}
	metricsRec2 := httptest.NewRecorder()
	app2.Metrics.Handler.ServeHTTP(metricsRec2, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(metricsRec2.Body.String(), `fvpn_control_rate_limited_total{class="login",limiter="subject"} 1`) {
		t.Fatalf("missing subject rate limit metric in:\n%s", metricsRec2.Body.String())
	}
}
