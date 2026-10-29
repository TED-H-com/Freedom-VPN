package httpapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func principal(role auth.Role) auth.Principal {
	return auth.Principal{Role: role, ID: uuid.New(), NodeID: uuid.New(), IncarnationID: uuid.New(), MFAVerified: true, CSRFToken: "TEST-ONLY-CSRF", AuthenticatedAt: time.Now()}
}
func fixture(t *testing.T, p auth.Principal, agent bool, log *slog.Logger, sets ...any) http.Handler {
	t.Helper()
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	return fixtureConfig(t, p, agent, log, c, sets...)
}
func fixtureConfig(t *testing.T, p auth.Principal, agent bool, log *slog.Logger, c config.Config, sets ...any) http.Handler {
	t.Helper()
	handlers, e := Compose(sets...)
	if e != nil {
		t.Fatal(e)
	}
	h, e := New(handlers, Options{Config: c, AgentListener: agent, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) {
		if p.Role == "" {
			return p, apierr.New(apierr.Unauthenticated, "Sign in required.")
		}
		return p, nil
	}), Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) { return p, nil })})
	if e != nil {
		t.Fatal(e)
	}
	return Transport(log, h)
}
func request(method, path, body string, p auth.Principal) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if method != "GET" {
		r.Header.Set("Idempotency-Key", uuid.NewString())
		r.Header.Set("Origin", "https://portal.example.invalid")
		r.Header.Set("X-CSRF-Token", p.CSRFToken)
	}
	return r
}
func assertStatus(t *testing.T, h http.Handler, r *http.Request, want int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("status=%d want=%d body=%s", w.Code, want, w.Body.String())
	}
	id, e := uuid.Parse(w.Header().Get("X-Request-ID"))
	if e != nil || id.Version() != 7 {
		t.Fatalf("central v7 request ID missing: %s", w.Header())
	}
	return w
}
func TestAT28_RolesAndFallback(t *testing.T) {
	for _, tc := range []struct {
		name   string
		p      auth.Principal
		path   string
		status int
	}{{"anonymous", auth.Principal{}, "/api/v1/admin/nodes", 401}, {"member on admin", principal(auth.Member), "/api/v1/admin/nodes", 403}, {"admin without MFA", func() auth.Principal { p := principal(auth.Admin); p.MFAVerified = false; return p }(), "/api/v1/admin/nodes", 403}, {"unimplemented", principal(auth.Admin), "/api/v1/admin/nodes", 501}, {"member fallback", principal(auth.Member), "/api/v1/member/me", 501}} {
		t.Run(tc.name, func(t *testing.T) {
			assertStatus(t, fixture(t, tc.p, false, nil), request("GET", tc.path, "", tc.p), tc.status)
		})
	}
}
func TestAT28_CertificateIdentityAndForgedHeaders(t *testing.T) {
	p := principal(auth.Agent)
	body := `{"renewal_request_id":"` + uuid.NewString() + `","node_id":"` + p.NodeID.String() + `","node_incarnation_id":"` + p.IncarnationID.String() + `","current_lease_id":null,"applied_config_generation":null}`
	h := fixture(t, p, true, nil)
	r := request("POST", "/api/v1/agent/lease-renewals", body, p)
	r.Header.Set("X-Node-ID", p.NodeID.String())
	r.Header.Set("X-Client-Cert", "TEST ONLY forged certificate")
	assertStatus(t, h, r, 401)
	r = request("POST", "/api/v1/agent/lease-renewals", strings.Replace(body, p.NodeID.String(), uuid.NewString(), 1), p)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	assertStatus(t, h, r, 403)
	r = request("POST", "/api/v1/agent/lease-renewals", body, p)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	r.Header.Set("X-Node-ID", uuid.NewString())
	assertStatus(t, h, r, 501)
	assertStatus(t, fixture(t, p, false, nil), r, 401)
}
func TestAT28_CSRFSchemaSizeAndPreconditions(t *testing.T) {
	p := principal(auth.Admin)
	h := fixture(t, p, false, nil)
	path := "/api/v1/admin/nodes/" + uuid.NewString()
	for _, tc := range []struct {
		name, body string
		status     int
		edit       func(*http.Request)
	}{
		{"csrf", `{"name":"new"}`, 403, func(r *http.Request) { r.Header.Del("X-CSRF-Token") }},
		{"origin", `{"name":"new"}`, 403, func(r *http.Request) { r.Header.Set("Origin", "https://foreign.example.invalid") }},
		{"missing condition", `{"name":"new"}`, 428, func(r *http.Request) {}},
		{"unknown field", `{"name":"new","publication_intent":"published"}`, 422, func(r *http.Request) { r.Header.Set("If-Match", `"1"`) }},
		{"duplicate JSON", `{"name":"first","name":"second"}`, 400, func(r *http.Request) { r.Header.Set("If-Match", `"1"`) }},
		{"missing key", `{"name":"new"}`, 400, func(r *http.Request) { r.Header.Del("Idempotency-Key") }},
		{"valid fallback", `{"name":"new"}`, 501, func(r *http.Request) { r.Header.Set("If-Match", `"1"`) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request("PATCH", path, tc.body, p)
			tc.edit(r)
			assertStatus(t, h, r, tc.status)
		})
	}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.BrowserBodyBytes = 16
	r := request("PATCH", path, `{"name":"too long for configured cap"}`, p)
	r.Header.Set("If-Match", `"1"`)
	assertStatus(t, fixtureConfig(t, p, false, nil, c), r, 413)
}

type panicSet struct{}

func (panicSet) ListNodes(context.Context, gen.ListNodesRequestObject) (gen.ListNodesResponseObject, error) {
	panic("TEST ONLY internal SQL token")
}

type errorSet struct{}

func (errorSet) ListNodes(context.Context, gen.ListNodesRequestObject) (gen.ListNodesResponseObject, error) {
	return nil, errors.New("TEST ONLY internal SQL token")
}
func TestAT28_ErrorAndLogRedaction(t *testing.T) {
	p := principal(auth.Admin)
	for _, set := range []any{panicSet{}, errorSet{}} {
		var logs bytes.Buffer
		h := fixture(t, p, false, slog.New(slog.NewJSONHandler(&logs, nil)), set)
		r := request("GET", "/api/v1/admin/nodes?untrusted=TEST-ONLY-QUERY", "", p)
		secrets := []string{"TEST-ONLY-AUTH", "TEST-ONLY-COOKIE", "TEST-ONLY-SETUP", "TEST-ONLY-SIGNATURE", "TEST-ONLY-TRACE"}
		for i, key := range []string{"Authorization", "Cookie", "X-Setup-Key", "X-Webhook-Signature", "X-Request-ID"} {
			r.Header.Set(key, secrets[i])
		}
		w := assertStatus(t, h, r, 500)
		for _, v := range append(secrets, "TEST ONLY internal SQL token", "TEST-ONLY-QUERY") {
			if strings.Contains(logs.String(), v) || strings.Contains(w.Body.String(), v) {
				t.Fatalf("leaked %q", v)
			}
		}
		if w.Header().Get("X-Request-ID") == r.Header.Get("X-Request-ID") {
			t.Fatal("trusted client request ID")
		}
	}
	// Even secret response headers are never passed to the access logger.
	var logs bytes.Buffer
	h := Transport(slog.New(slog.NewJSONHandler(&logs, nil)), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "TEST-ONLY-RESPONSE-SECRET")
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if strings.Contains(logs.String(), "TEST-ONLY-RESPONSE-SECRET") {
		t.Fatal("response secret logged")
	}
}
func TestAPI01_ETag(t *testing.T) {
	if CheckIfMatch("", 1).(*apierr.Error).HTTPStatus() != 428 {
		t.Fatal("missing")
	}
	e := CheckIfMatch(`"2"`, 1).(*apierr.Error)
	if e.HTTPStatus() != 412 || e.Details["current_resource_version"] != decstr.U64(1) {
		t.Fatal(e)
	}
	if CheckIfMatch(`"1"`, 1) != nil {
		t.Fatal("match")
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("If-None-Match", `W/"1", "2"`)
	w := httptest.NewRecorder()
	if !NotModified(w, r, ETag(1)) || w.Code != 304 {
		t.Fatal("not modified")
	}
}
func TestAPI01_RateLimit(t *testing.T) {
	p := principal(auth.Member)
	c := config.Defaults()
	c.BrowserRate = config.Bucket{PerSecond: .001, Burst: 1}
	h := fixtureConfig(t, p, false, nil, c)
	assertStatus(t, h, request("GET", "/api/v1/member/me", "", p), 501)
	w := assertStatus(t, h, request("GET", "/api/v1/member/me", "", p), 429)
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}
func TestAPI01_CompositionRejectsDuplicates(t *testing.T) {
	if _, e := Compose(panicSet{}, errorSet{}); e == nil {
		t.Fatal("duplicate route accepted")
	}
}

func TestENR02_EnrollmentReplayStripsSecret(t *testing.T) {
	token := "TEST ONLY one-time token"
	response := gen.CreateEnrollment201JSONResponse{}
	response.Body.SecretAvailable = true
	response.Body.Token.Set(token)
	v, e := sanitizeReplay("CreateEnrollment", response)
	if e != nil {
		t.Fatal(e)
	}
	safe := v.(gen.CreateEnrollment201JSONResponse)
	if safe.Body.SecretAvailable || !safe.Body.Token.IsNull() {
		t.Fatal("secret replayed")
	}
	if !response.Body.Token.IsSpecified() || response.Body.Token.IsNull() {
		t.Fatal("original response mutated")
	}
}

type webhookSet struct{ raw []byte }

func (s *webhookSet) ReceiveProviderWebhook(ctx context.Context, _ gen.ReceiveProviderWebhookRequestObject) (gen.ReceiveProviderWebhookResponseObject, error) {
	s.raw = RawBody(ctx)
	return nil, errors.New("test hook stops before acknowledgment")
}
func TestAPI01_RawWebhookAndUnsupportedMethod(t *testing.T) {
	p := principal(auth.Admin)
	s := &webhookSet{}
	raw := " {\n \"event\": \"TEST ONLY signed payload\"\n}\n"
	h := fixture(t, p, false, nil, s)
	r := request("POST", "/api/v1/webhooks/example", raw, p)
	r.Header.Del("Idempotency-Key")
	assertStatus(t, h, r, 500)
	if string(s.raw) != raw {
		t.Fatal("raw signature bytes changed")
	}
	r = request("POST", "/api/v1/admin/nodes", "", p)
	assertStatus(t, h, r, 405)
}

// D-015: the composition remains transitional, but transport must enforce the
// secret-delivery route's mTLS, optional idempotency header and empty body.
func TestAT28_BootstrapTransport(t *testing.T) {
	p := principal(auth.Agent)
	h := fixture(t, p, true, nil)
	for _, tc := range []struct {
		name, body, key string
		status          int
	}{
		{"without optional key", "", "", 501},
		{"with optional key", "", uuid.NewString(), 501},
		{"invalid optional key", "", "invalid", 400},
		{"object body", "{}", "", 400},
		{"null body", "null", "", 400},
		{"whitespace body", " ", "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request("POST", "/api/v1/agent/bootstrap", tc.body, p)
			r.Header.Del("Idempotency-Key")
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			r.ContentLength = -1 // Also reject bodies without a declared length.
			w := assertStatus(t, h, r, tc.status)
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("bootstrap response must not be cached")
			}
		})
	}
	r := request("POST", "/api/v1/agent/bootstrap", "", p)
	assertStatus(t, h, r, 401)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	assertStatus(t, fixture(t, p, false, nil), r, 401)
}

func TestReviewC12_TransportRefusesSecondError(t *testing.T) {
	h := Transport(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":true}`))
		WriteError(w, r, errNotImplemented)
		apierr.Write(w, r, errors.New("late middleware error"))
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/test", nil))
	if w.Code != http.StatusAccepted || w.Body.String() != `{"accepted":true}` {
		t.Fatalf("response overwritten or appended: %d %s", w.Code, w.Body.String())
	}
}

func TestAPI01_UsageOptionalIdempotencyKey(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/UsageBatchRequest.json")
	if err != nil {
		t.Fatal(err)
	}
	var body gen.UsageBatchRequest
	if err = json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	p := principal(auth.Agent)
	p.NodeID, p.IncarnationID = body.NodeId, body.NodeIncarnationId
	h := fixture(t, p, true, nil)
	for _, tc := range []struct {
		key    string
		status int
	}{{"", 501}, {uuid.NewString(), 501}, {"invalid", 400}} {
		r := request("POST", "/api/v1/agent/usage-batches", string(raw), p)
		r.Header.Del("Idempotency-Key")
		if tc.key != "" {
			r.Header.Set("Idempotency-Key", tc.key)
		}
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		assertStatus(t, h, r, tc.status)
	}
}
