//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	t             *testing.T
	r             *httpwire.Runtime
	public, agent http.Handler
	p             auth.Principal
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := dbtest.New(t)
	vault, ca := pkitest.Authority(t)
	identity, e := pki.NewService(pool, ca, vault, pki.TestRenewalOverlap)
	if e != nil {
		t.Fatal(e)
	}
	en, e := enrollment.NewService(pool, ca, identity, vault, vault, enrollment.Config{BootstrapValidity: enrollment.TestBootstrapValidity})
	if e != nil {
		t.Fatal(e)
	}
	runtime := &httpwire.Runtime{Pool: pool, PKI: identity, Enrollment: en, Nodes: nodes.NewService(pool)}
	f := &fixture{t: t, r: runtime, p: auth.Principal{Role: auth.Admin, ID: uuid.New(), MFAVerified: true, AuthenticatedAt: time.Now(), CSRFToken: "test-csrf"}}
	handlers, e := httpapi.Compose(&httpwire.Handlers{Runtime: runtime})
	if e != nil {
		t.Fatal(e)
	}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.AnonymousRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.AgentRate = c.AnonymousRate
	c.BrowserRate = c.AnonymousRate
	c.IngressRate = c.AnonymousRate
	o := httpapi.Options{Config: c, Agents: identity, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return f.p, nil }), Commands: &httpapi.Commands{Pool: pool, Retention: time.Hour}}
	h, e := httpapi.New(handlers, o)
	if e != nil {
		t.Fatal(e)
	}
	f.public = httpapi.Transport(nil, httpwire.RequestContext(h))
	o.AgentListener = true
	h, e = httpapi.New(handlers, o)
	if e != nil {
		t.Fatal(e)
	}
	f.agent = httpapi.Transport(nil, httpwire.RequestContext(h))
	return f
}
func (f *fixture) call(method, path string, body any, cert *x509.Certificate, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	var raw []byte
	if body != nil {
		var e error
		raw, e = json.Marshal(body)
		if e != nil {
			f.t.Fatal(e)
		}
	}
	r := httptest.NewRequest(method, "https://control.example.invalid"+path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://portal.example.invalid")
	r.Header.Set("X-CSRF-Token", f.p.CSRFToken)
	for k, v := range headers {
		if v != "" {
			r.Header.Set(k, v)
		}
	}
	w := httptest.NewRecorder()
	if cert != nil {
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
		f.agent.ServeHTTP(w, r)
	} else {
		f.public.ServeHTTP(w, r)
	}
	return w
}
func decode[T any](t *testing.T, w *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d want %d: %s", w.Code, status, w.Body.String())
	}
	var v T
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func (f *fixture) create(node *uuid.UUID) api.EnrollmentCreated {
	f.t.Helper()
	return decode[api.EnrollmentCreated](f.t, f.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: "installer", NodeId: node}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 201)
}
func (f *fixture) enroll(c api.EnrollmentCreated) (api.EnrollRequest, api.EnrollResponse, *x509.Certificate) {
	f.t.Helper()
	token, e := c.Token.Get()
	if e != nil {
		f.t.Fatal(e)
	}
	r, _ := pkitest.Request(f.t)
	out := decode[api.EnrollResponse](f.t, f.call("POST", "/api/v1/agent/enroll", r, nil, map[string]string{"Authorization": "Enrollment " + token}), 201)
	return r, out, pkitest.Certificate(f.t, out.CertificatePem)
}
func TestAT06_AT07_AT28_HTTPEnrollmentReinstallRevocation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	key := uuid.NewString()
	body := api.EnrollmentCreateRequest{Delivery: "installer"}
	c := decode[api.EnrollmentCreated](t, f.call("POST", "/api/v1/admin/enrollments", body, nil, map[string]string{"Idempotency-Key": key}), 201)
	replay := decode[api.EnrollmentCreated](t, f.call("POST", "/api/v1/admin/enrollments", body, nil, map[string]string{"Idempotency-Key": key}), 201)
	if replay.SecretAvailable || !replay.Token.IsNull() || replay.Enrollment.EnrollmentId != c.Enrollment.EnrollmentId {
		t.Fatal("secret replayed")
	}
	r, out, cert := f.enroll(c)
	token, _ := c.Token.Get()
	again := decode[api.EnrollResponse](t, f.call("POST", "/api/v1/agent/enroll", r, nil, map[string]string{"Authorization": "Enrollment " + token}), 200)
	if again.CertificateSerial != out.CertificateSerial {
		t.Fatal("reissued on retry")
	}
	conflict, _ := pkitest.Request(t)
	w := f.call("POST", "/api/v1/agent/enroll", conflict, nil, map[string]string{"Authorization": "Enrollment " + token})
	if w.Code != 409 {
		t.Fatalf("key race %d %s", w.Code, w.Body.String())
	}
	w = f.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: "installer", NodeId: &out.NodeId}, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
	if w.Code != 409 || !strings.Contains(w.Body.String(), "NODE_INCARNATION_ACTIVE") {
		t.Fatalf("active reinstall %d %s", w.Code, w.Body.String())
	}
	w = f.call("POST", "/api/v1/agent/bootstrap", nil, cert, map[string]string{"X-Node-ID": uuid.NewString()})
	if w.Code != 409 || !strings.Contains(w.Body.String(), "BOOTSTRAP_NOT_READY") {
		t.Fatalf("bootstrap %d %s", w.Code, w.Body.String())
	}
	detail := f.call("GET", "/api/v1/admin/nodes/"+out.NodeId.String(), nil, nil, nil)
	if detail.Code != 200 {
		t.Fatal(detail.Body.String())
	}
	rev := f.call("POST", "/api/v1/admin/nodes/"+out.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY revocation"}, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")})
	if rev.Code != 202 {
		t.Fatalf("revoke %d %s", rev.Code, rev.Body.String())
	}
	if _, e := f.r.PKI.AuthenticateAgent(ctx, cert); e == nil {
		t.Fatal("revoked certificate authenticated")
	}
	// The base migration still prohibits reinstalls. Exercise the proposed domain
	// migration explicitly; embedding it belongs to the database owner.
	if _, e := f.r.Pool.Exec(ctx, migrations.Reinstall); e != nil {
		t.Fatal(e)
	}
	replacement := f.create(&out.NodeId)
	_, newOut, newCert := f.enroll(replacement)
	if newOut.NodeId != out.NodeId || newOut.NodeIncarnationId == out.NodeIncarnationId {
		t.Fatal("reinstall identity")
	}
	if _, e := f.r.PKI.AuthenticateAgent(ctx, newCert); e != nil {
		t.Fatal(e)
	}
}
func TestAT28_HTTPBootstrapExactlyOnce(t *testing.T) {
	f := setup(t)
	c := f.create(nil)
	_, out, cert := f.enroll(c)
	ctx := context.Background()
	var op uuid.UUID
	if e := f.r.Pool.QueryRow(ctx, `SELECT operation_id FROM node_enrollments WHERE id=$1`, c.Enrollment.EnrollmentId).Scan(&op); e != nil {
		t.Fatal(e)
	}
	secret := "TEST ONLY bootstrap " + uuid.NewString()
	e := f.r.Enrollment.StoreBootstrap(ctx, pki.NodeIdentity{NodeID: out.NodeId, IncarnationID: out.NodeIncarnationId}, op, "test-external-key", secret, time.Now().Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	key := uuid.NewString()
	w := f.call("POST", "/api/v1/agent/bootstrap", nil, cert, map[string]string{"Idempotency-Key": key, "X-Node-ID": uuid.NewString()})
	got := decode[api.AgentBootstrap](t, w, 200)
	if got.SetupKey != secret || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("secret delivery")
	}
	for _, k := range []string{key, uuid.NewString(), ""} {
		w = f.call("POST", "/api/v1/agent/bootstrap", nil, cert, map[string]string{"Idempotency-Key": k})
		if w.Code != 410 || !strings.Contains(w.Body.String(), "BOOTSTRAP_CONSUMED") {
			t.Fatalf("consumption %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("secret replay")
		}
	}
	var erased bool
	if e := f.r.Pool.QueryRow(ctx, `SELECT ciphertext IS NULL FROM node_bootstrap_secrets WHERE incarnation_id=$1`, out.NodeIncarnationId).Scan(&erased); e != nil || !erased {
		t.Fatal("ciphertext retained", e)
	}
}
