//go:build integration

package httpadapter_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const testOrigin = "https://portal.example.invalid"

type browserSession struct {
	id     uuid.UUID
	cookie *http.Cookie
	csrf   string
}
type fixture struct {
	router                routers.Router
	browser               *auth.Browser
	t                     *testing.T
	pool                  *pgxpool.Pool
	h                     http.Handler
	auth                  http.Handler
	services              *membersruntime.Services
	payment               *billing.FakeProvider
	probe                 *probePayment
	now                   time.Time
	plan, revision, group uuid.UUID
	idp                   *testIDP
	netbird               *netbirdFake
	dispatcher            *worker.Application
	config                config.Config
	app                   *control.Application
	logs                  bytes.Buffer
	identityMetrics       *prometheus.Registry
}
type probePayment struct {
	*billing.FakeProvider
	pool       *pgxpool.Pool
	t          *testing.T
	failPortal bool
	calls      int
}

func (p *probePayment) CreatePortal(ctx context.Context, r billing.PortalRequest) (billing.HostedSession, error) {
	p.calls++
	var n int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction'`).Scan(&n); err != nil || n != 0 {
		p.t.Errorf("provider called with open request transaction: %d %v", n, err)
	}
	if p.failPortal {
		return billing.HostedSession{}, billing.ErrProvider
	}
	return p.FakeProvider.CreatePortal(ctx, r)
}
func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := dbtest.New(t)
	f := &fixture{t: t, pool: pool, now: time.Now().UTC().Truncate(time.Second), plan: uuid.New(), revision: uuid.New(), group: uuid.New()}
	engine, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	engine.Now = func() time.Time { return f.now }
	f.payment = billing.NewFakeProvider()
	f.probe = &probePayment{FakeProvider: f.payment, pool: pool, t: t}
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		t.Fatal(err)
	}
	vault, err := billing.NewVault("TEST-ONLY", map[string][]byte{"TEST-ONLY": key})
	clear(key)
	if err != nil {
		t.Fatal(err)
	}
	pay, err := billing.NewService(pool, f.probe, vault, engine, billing.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	fakeNB, nb := newNetBird(t)
	f.netbird = fakeNB
	bindings := devices.NetBirdBindings{Pool: pool}
	client, err := netbird.New(netbird.Config{BaseURL: nb.URL + "/api", DeploymentID: "TEST-ONLY", Token: testToken{}, Bindings: bindings, AllowInsecureHTTP: true, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	adapter := &devices.NetBirdAdapter{Pool: pool, Client: client, Bindings: bindings, DeploymentID: "TEST-ONLY", OverlayScope: "TEST-ONLY"}
	f.services = &membersruntime.Services{HTTP: httpadapter.Services{Members: &members.Service{Pool: pool, Entitlements: engine}, Devices: &devices.Service{Pool: pool, Entitlements: engine, Adapter: adapter}, Billing: pay}, Adapter: adapter, Compiler: desiredstate.NewCompiler(pool, nil), ReconcileInterval: time.Hour, DeviceInterval: time.Second}
	f.exec(`INSERT INTO member_groups(id,code,name) VALUES($1,'test-members','TEST ONLY')`, f.group)
	f.exec(`INSERT INTO plans(id,code) VALUES($1,'test-plan')`, f.plan)
	f.exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,provider,provider_price_ref,schema_version,payload) VALUES($1,$2,1,3,$3,'fake','TEST-ONLY-price',1,'{"display_name":{"zh-TW":"測試方案"}}')`, f.revision, f.plan, []uuid.UUID{f.group})
	f.exec(`UPDATE plans SET active_revision_id=$2,resource_version=resource_version+1 WHERE id=$1`, f.plan, f.revision)
	f.idp = newIDP(t)
	f.identityMetrics = prometheus.NewRegistry()
	adapter.Identity, err = devices.NewIdentityDiscovery(f.idp.server.URL, f.identityMetrics, slog.New(slog.NewTextHandler(&f.logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	browser, err := auth.NewBrowser(t.Context(), auth.BrowserConfig{Issuer: f.idp.server.URL, ClientID: "TEST-ONLY-portal", Origin: testOrigin, AdminClaim: "roles", AdminValue: "admin", MFAAMR: []string{"mfa"}, Idle: time.Hour, Absolute: 2 * time.Hour, LoginTTL: time.Minute, RecentAuth: time.Hour}, auth.PostgresStore{Pool: pool}, f.idp.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	f.browser = browser
	spec, e := gen.GetSpec()
	if e != nil {
		t.Fatal(e)
	}
	spec.Servers = nil
	f.router, e = legacy.NewRouter(spec)
	if e != nil {
		t.Fatal(e)
	}
	c := config.Defaults()
	c.PublicOrigin = testOrigin
	c.OIDCIssuer = f.idp.server.URL
	c.BrowserRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.IngressRate = c.BrowserRate
	c.AnonymousRate = c.BrowserRate
	c.WorkerAttempts = 3
	c = testTLSConfig(t, c)
	f.config = c
	f.compose(c)
	f.dispatcher, err = worker.Open(t.Context(), worker.Dependencies{Config: c, Pool: pool}, worker.Domain{Name: "members", Build: func(context.Context, worker.Dependencies) (worker.Built, error) {
		return worker.Built{Jobs: worker.MemberJobs(), Context: func(ctx context.Context) context.Context { return worker.WithMembers(ctx, f.services) }}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.t.Context(), sql, args...); err != nil {
		f.t.Fatal(err)
	}
}
func (f *fixture) count(sql string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.pool.QueryRow(f.t.Context(), sql, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}
func (f *fixture) call(s browserSession, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", testOrigin)
	r.Header.Set("X-CSRF-Token", s.csrf)
	if s.cookie != nil {
		r.AddCookie(s.cookie)
	}
	if method != "GET" {
		r.Header.Set("Idempotency-Key", uuid.NewString())
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	if w.Code >= 200 && w.Code < 300 {
		clone := r.Clone(r.Context())
		u := *r.URL
		clone.URL = &u
		clone.URL.Path = strings.TrimPrefix(clone.URL.Path, "/api/v1")
		route, params, e := f.router.FindRoute(clone)
		if e != nil {
			f.t.Fatal(e)
		}
		e = openapi3filter.ValidateResponse(r.Context(), &openapi3filter.ResponseValidationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: clone, Route: route, PathParams: params}, Status: w.Code, Header: w.Header(), Body: io.NopCloser(bytes.NewReader(w.Body.Bytes())), Options: &openapi3filter.Options{IncludeResponseStatus: true}})
		if e != nil {
			f.t.Errorf("response violates contract for %s: %v", path, e)
		}
	}
	if w.Header().Get("X-Request-ID") == "" || w.Header().Get("Cache-Control") != "no-store" {
		f.t.Fatal("missing security headers")
	}
	return w
}
func requireStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP %d want %d: %s", w.Code, status, w.Body.String())
	}
}
func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func (f *fixture) drain() {
	f.t.Helper()
	ctx := f.t.Context()
	for range 100 {
		worked, e := f.dispatcher.Once(ctx)
		if e != nil {
			f.t.Fatal(e)
		}
		if !worked {
			break
		}
	}
	if n := f.count(`SELECT count(*) FROM outbox_events WHERE delivered_at IS NULL`); n != 0 {
		var typ, code string
		_ = f.pool.QueryRow(f.t.Context(), `SELECT event_type,coalesce(last_error_code,'') FROM outbox_events WHERE delivered_at IS NULL ORDER BY created_at LIMIT 1`).Scan(&typ, &code)
		f.t.Fatalf("undelivered work %d (%s %s)", n, typ, code)
	}
}

// Seed native-provider inputs only. Discovery owns both binding projections (D-050).
func (f *fixture) seedPeer(s browserSession, peer, ip string) {
	f.t.Helper()
	var subject string
	if err := f.pool.QueryRow(f.t.Context(), `SELECT i.subject FROM identity_bindings i JOIN members m ON m.canonical_identity_ref=i.id WHERE m.id=$1`, s.id).Scan(&subject); err != nil {
		f.t.Fatal(err)
	}
	f.seedNative(subject, peer, ip, false, false)
}
func (f *fixture) seedNative(user, peer, ip string, service, blocked bool) {
	f.netbird.mu.Lock()
	defer f.netbird.mu.Unlock()
	f.netbird.objects["users"][user] = map[string]any{"id": user, "auto_groups": []string{}, "is_service_user": service, "is_blocked": blocked}
	f.netbird.objects["peers"][peer] = map[string]any{"id": peer, "name": "TEST ONLY device", "user_id": user, "ip": ip, "ephemeral": false}
}

// Real OIDC signature/PKCE, real PostgreSQL sessions, real CSRF middleware.
type testIDP struct {
	server                          *httptest.Server
	key                             *rsa.PrivateKey
	mu                              sync.Mutex
	subject, nonce, challenge, role string
}

func newIDP(t *testing.T, tlsMode ...bool) *testIDP {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	p := &testIDP{key: key}
	p.server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		defer p.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": p.server.URL, "authorization_endpoint": p.server.URL + "/authorize", "token_endpoint": p.server.URL + "/token", "jwks_uri": p.server.URL + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "TEST-ONLY", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
		case "/token":
			_ = r.ParseForm()
			hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(hash[:]) != p.challenge {
				w.WriteHeader(400)
				return
			}
			now := time.Now()
			claims := map[string]any{"iss": p.server.URL, "aud": "TEST-ONLY-portal", "sub": p.subject, "nonce": p.nonce, "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "auth_time": now.Unix(), "name": "TEST ONLY member", "amr": []string{"mfa"}, "roles": []string{p.role}}
			b, _ := json.Marshal(claims)
			unsigned := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"TEST-ONLY"}`)) + "." + base64.RawURLEncoding.EncodeToString(b)
			digest := sha256.Sum256([]byte(unsigned))
			signature, e := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
			if e != nil {
				t.Error(e)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "TEST ONLY transient access token", "token_type": "Bearer", "id_token": unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)})
		default:
			w.WriteHeader(404)
		}
	}))
	if len(tlsMode) > 0 && tlsMode[0] {
		p.server.StartTLS()
	} else {
		p.server.Start()
	}
	t.Cleanup(p.server.Close)
	return p
}
func (f *fixture) login(subject, role string) browserSession {
	f.t.Helper()
	w := httptest.NewRecorder()
	f.auth.ServeHTTP(w, httptest.NewRequest("GET", "/auth/login", nil))
	requireStatus(f.t, w, 302)
	target, e := url.Parse(w.Header().Get("Location"))
	if e != nil {
		f.t.Fatal(e)
	}
	f.idp.mu.Lock()
	f.idp.subject = subject
	f.idp.role = role
	f.idp.nonce = target.Query().Get("nonce")
	f.idp.challenge = target.Query().Get("code_challenge")
	f.idp.mu.Unlock()
	callback := httptest.NewRequest("GET", "/auth/callback?code=TEST-ONLY&state="+url.QueryEscape(target.Query().Get("state")), nil)
	for _, c := range w.Result().Cookies() {
		callback.AddCookie(c)
	}
	w = httptest.NewRecorder()
	f.auth.ServeHTTP(w, callback)
	requireStatus(f.t, w, 303)
	cookieName := auth.MemberSessionCookie
	identityQuery := `SELECT member_id FROM identity_bindings WHERE issuer=$1 AND client_scope='TEST-ONLY-portal' AND subject=$2`
	if role == "admin" {
		cookieName = auth.AdminSessionCookie
		identityQuery = `SELECT id FROM admin_identities WHERE issuer=$1 AND client_scope='TEST-ONLY-portal' AND subject=$2`
	}
	var s browserSession
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			s.cookie = c
		}
	}
	if s.cookie == nil {
		f.t.Fatal("missing session cookie")
	}
	r := httptest.NewRequest("GET", "/auth/session", nil)
	r.AddCookie(s.cookie)
	w = httptest.NewRecorder()
	f.auth.ServeHTTP(w, r)
	requireStatus(f.t, w, 200)
	s.csrf = w.Header().Get("X-CSRF-Token")
	if e = f.pool.QueryRow(f.t.Context(), identityQuery, f.idp.server.URL, subject).Scan(&s.id); e != nil {
		f.t.Fatal(e)
	}
	return s
}

// HTTP double uses the same management response shapes as internal/netbird's
// private test double; exercises the real client and persistent binding store.
type testToken struct{}

func (testToken) Token(context.Context) (string, error) { return "TEST ONLY synthetic token", nil }

type netbirdFake struct {
	loseCreate     bool
	mu             sync.Mutex
	objects        map[string]map[string]map[string]any
	serial, writes int
	// grants counts peers newly added to an fvpn-paid-* group. A repeat PUT
	// that leaves the peer in place does not increment it.
	grants      int
	stale       bool
	onUsersRead func() error
}

func newNetBird(t *testing.T) (*netbirdFake, *httptest.Server) {
	f := &netbirdFake{objects: map[string]map[string]map[string]any{}}
	for _, kind := range []string{"groups", "peers", "users", "routes", "policies", "dns/nameservers"} {
		f.objects[kind] = map[string]map[string]any{}
	}
	f.objects["groups"]["TEST-foreign"] = map[string]any{"id": "TEST-foreign", "name": "Unmanaged", "peers": []string{"TEST-peer"}, "resources": []any{}}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	return f, server
}
func (f *netbirdFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Token TEST ONLY synthetic token" {
		w.WriteHeader(401)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	kind, id, _ := strings.Cut(path, "/")
	if kind == "dns" {
		kind = path
		id = ""
	}
	bucket, ok := f.objects[kind]
	if !ok {
		w.WriteHeader(404)
		return
	}
	respond := func(obj map[string]any) {
		if kind != "groups" {
			_ = json.NewEncoder(w).Encode(obj)
			return
		}
		copy := map[string]any{}
		for k, v := range obj {
			copy[k] = v
		}
		peers := []map[string]any{}
		b, _ := json.Marshal(obj["peers"])
		var ids []string
		_ = json.Unmarshal(b, &ids)
		for _, peer := range ids {
			peers = append(peers, map[string]any{"id": peer})
		}
		copy["peers"] = peers
		_ = json.NewEncoder(w).Encode(copy)
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		if kind == "users" && f.onUsersRead != nil {
			if err := f.onUsersRead(); err != nil {
				w.WriteHeader(503)
				return
			}
		}
		if id != "" {
			obj, ok := bucket[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			respond(obj)
		} else {
			_, _ = w.Write([]byte("["))
			first := true
			for _, obj := range bucket {
				if !first {
					_, _ = w.Write([]byte(","))
				}
				first = false
				respond(obj)
			}
			_, _ = w.Write([]byte("]"))
		}
		return
	}
	if r.Method == "DELETE" {
		delete(bucket, id)
		w.WriteHeader(204)
		return
	}
	var obj map[string]any
	if json.NewDecoder(r.Body).Decode(&obj) != nil {
		w.WriteHeader(400)
		return
	}
	f.writes++
	if kind == "groups" {
		name, _ := obj["name"].(string)
		if strings.HasPrefix(name, "fvpn-paid-") {
			prev := map[string]bool{}
			if old, ok := bucket[id]; ok {
				for _, peer := range peerList(old["peers"]) {
					prev[peer] = true
				}
			}
			for _, peer := range peerList(obj["peers"]) {
				if peer != "" && !prev[peer] {
					f.grants++
				}
			}
		}
	}
	if id == "" {
		f.serial++
		id = fmt.Sprintf("TEST-created-%d", f.serial)
	}
	obj["id"] = id
	if !f.stale {
		bucket[id] = obj
	}
	if f.loseCreate && r.Method == "POST" {
		f.loseCreate = false
		w.WriteHeader(503)
		return
	}
	respond(obj)
}

func (f *netbirdFake) grantCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.grants
}

// withdrawPaid removes a peer from paid groups without an HTTP write, so the
// grant counter stays put. The next authorization PUT is then observable.
func (f *netbirdFake) withdrawPaid(peer string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, obj := range f.objects["groups"] {
		name, _ := obj["name"].(string)
		if !strings.HasPrefix(name, "fvpn-paid-") {
			continue
		}
		kept := []string{}
		for _, id := range peerList(obj["peers"]) {
			if id != peer {
				kept = append(kept, id)
			}
		}
		obj["peers"] = kept
	}
}

func (f *netbirdFake) inPaidGroup(peer string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, obj := range f.objects["groups"] {
		name, _ := obj["name"].(string)
		if !strings.HasPrefix(name, "fvpn-paid-") {
			continue
		}
		for _, id := range peerList(obj["peers"]) {
			if id == peer {
				return true
			}
		}
	}
	return false
}

func peerList(v any) []string {
	switch p := v.(type) {
	case nil:
		return nil
	case []string:
		return append([]string{}, p...)
	case []any:
		out := make([]string, 0, len(p))
		for _, item := range p {
			switch x := item.(type) {
			case string:
				out = append(out, x)
			case map[string]any:
				if id, ok := x["id"].(string); ok {
					out = append(out, id)
				}
			}
		}
		return out
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var ids []string
		if json.Unmarshal(b, &ids) == nil && (len(ids) > 0 || string(b) == "[]" || string(b) == "null") {
			return ids
		}
		var objs []map[string]any
		if json.Unmarshal(b, &objs) != nil {
			return nil
		}
		out := make([]string, 0, len(objs))
		for _, o := range objs {
			if id, ok := o["id"].(string); ok {
				out = append(out, id)
			}
		}
		return out
	}
}

// Test dependencies enter through the generic build seam; listeners, middleware
// and strict composition are identical to the binary's Servers path.
func (f *fixture) compose(c config.Config) {
	f.t.Helper()
	// D-039: member integration tests compose only core and the member domain.
	// Select core explicitly because the production constructor is private.
	var domains []control.Domain
	for _, domain := range control.Domains() {
		if domain.Name == "control-core" {
			domains = append(domains, domain)
		}
	}
	if len(domains) != 1 {
		f.t.Fatal("control core domain missing or duplicated")
	}
	domains = append(domains, control.Domain{Name: "members", Build: func(_ context.Context, d control.Dependencies) (control.Built, error) {
		return control.Built{Handlers: []any{httpadapter.Handlers{Pool: d.Pool, Services: &f.services.HTTP}}, Middleware: httpadapter.RequestContext}, nil
	}})
	app, e := control.Servers(f.t.Context(), control.Dependencies{Config: c, Pool: f.pool, Browser: f.browser, Logger: slog.New(slog.NewTextHandler(&f.logs, nil))}, domains...)
	if e != nil {
		f.t.Fatal(e)
	}
	f.app = app
	f.h, f.auth = app.Public.Handler, app.Public.Handler
}
