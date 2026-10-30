//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func nodeDomains(t *testing.T) (control.Domain, worker.Domain) {
	t.Helper()
	var c control.Domain
	var w worker.Domain
	for _, d := range control.Domains() {
		if d.Name == "nodes" {
			if c.Build != nil {
				t.Fatal("duplicate node control domain")
			}
			c = d
		}
	}
	for _, d := range worker.Domains() {
		if d.Name == "nodes" {
			if w.Build != nil {
				t.Fatal("duplicate node worker domain")
			}
			w = d
		}
	}
	if c.Build == nil || w.Build == nil {
		t.Fatal("production nodes domain missing")
	}
	return c, w
}
func writeFixture(t *testing.T, dir, name string, raw []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if e := os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	return path
}
func jsonFixture(t *testing.T, dir, name string, value any) string {
	t.Helper()
	raw, e := json.Marshal(value)
	if e != nil {
		t.Fatal(e)
	}
	return writeFixture(t, dir, name, raw)
}

// TEST ONLY authority and timing values, all keys generated at test time.
func domainConfig(t *testing.T) (config.Config, config.Nodes) {
	t.Helper()
	dir := t.TempDir()
	fixture := nodestest.New(t, nodestest.NetBird{})
	n := fixture.Nodes
	c := config.Defaults()
	c.NodesConfigFile = fixture.Path
	c.AgentCAFile = n.CACertificateFile
	c.PublicAddress = "127.0.0.1:0"
	c.AgentAddress = "127.0.0.1:0"
	c.MetricsAddress = "127.0.0.1:0"
	c.ShutdownSeconds = 2
	c.PublicOrigin = "https://portal.example.invalid"
	tlsKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "TEST ONLY"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	tlsDER, e := x509.CreateCertificate(rand.Reader, template, template, &tlsKey.PublicKey, tlsKey)
	if e != nil {
		t.Fatal(e)
	}
	private, e := x509.MarshalPKCS8PrivateKey(tlsKey)
	if e != nil {
		t.Fatal(e)
	}
	c.TLSCertificateFile = writeFixture(t, dir, "server.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsDER}))
	c.TLSKeyFile = writeFixture(t, dir, "server-key.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}))
	return c, n
}
func TestAPI01_NodeDomainsRequireExplicitSettings(t *testing.T) {
	c, w := nodeDomains(t)
	good, n := domainConfig(t)
	// A domain may not fall back to process environment when Dependencies disagrees.
	t.Setenv("FVPN_NODES_CONFIG_FILE", good.NodesConfigFile)
	invalid := writeFixture(t, t.TempDir(), "invalid.json", []byte(`{"unrecognized":true}`))
	paths := []string{"", filepath.Join(t.TempDir(), "missing.json"), invalid}
	for _, resolver := range []struct{ v4, v6 []string }{
		{}, {v4: []string{"2001:db8::53"}}, {v6: []string{"192.0.2.53"}}, {v6: []string{"fe80::1%eth0"}},
	} {
		n.DNSResolverIPv4, n.DNSResolverIPv6 = resolver.v4, resolver.v6
		paths = append(paths, jsonFixture(t, t.TempDir(), "bad-dns.json", n))
	}
	for _, path := range paths {
		cfg := config.Defaults()
		cfg.NodesConfigFile = path
		_, ce := c.Build(t.Context(), control.Dependencies{Config: cfg})
		_, we := w.Build(t.Context(), worker.Dependencies{Config: cfg})
		for _, e := range []error{ce, we} {
			var setting *config.SettingError
			if !errors.As(e, &setting) || setting.Name != "nodes_config_file" {
				t.Fatalf("startup setting failure missing: %v", e)
			}
		}
	}
}
func TestAT06_AT28_NodeCompositionUsesMiddlewareAndPKI(t *testing.T) {
	pool := dbtest.New(t)
	cfg, n := domainConfig(t)
	// D-039: exercise only core and nodes through the real composition seam.
	var domains []control.Domain
	for _, domain := range control.Domains() {
		if domain.Name == "control-core" || domain.Name == "nodes" {
			domains = append(domains, domain)
		}
	}
	if len(domains) != 2 {
		t.Fatal("core/node domains missing or duplicated")
	}
	r, e := httpwire.Open(t.Context(), pool, n, nil)
	if e != nil {
		t.Fatal(e)
	}
	token, e := r.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: "installer"})
	if e != nil {
		t.Fatal(e)
	}
	app, e := control.Servers(t.Context(), control.Dependencies{Config: cfg, Pool: pool, Browser: &auth.Browser{}}, domains...)
	if e != nil {
		t.Fatal(e)
	}
	request, _ := pkitest.Request(t)
	raw, e := json.Marshal(request)
	if e != nil {
		t.Fatal(e)
	}
	req := httptest.NewRequest("POST", "/api/v1/agent/enroll", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Enrollment "+mustValue(t, token.Token))
	w := httptest.NewRecorder()
	app.Public.Handler.ServeHTTP(w, req)
	enrolled := decode[api.EnrollResponse](t, w, 201)
	cert := pkitest.Certificate(t, enrolled.CertificatePem)
	for _, identity := range []*x509.Certificate{nil, cert} {
		req := httptest.NewRequest("POST", "/api/v1/agent/bootstrap", nil)
		req.Header.Set("X-Node-ID", uuid.NewString())
		if identity != nil {
			req.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{identity}}}
		}
		w := httptest.NewRecorder()
		app.Agent.Handler.ServeHTTP(w, req)
		want := 401
		if identity != nil {
			want = 409
		}
		if w.Code != want {
			t.Fatalf("authentication/middleware %d want %d: %s", w.Code, want, w.Body.String())
		}
	}
}
func TestAPI01_NodeRunnerSeedsCoalescesAndStops(t *testing.T) {
	pool := dbtest.New(t)
	cfg, _ := domainConfig(t)
	_, domain := nodeDomains(t)
	built, e := domain.Build(t.Context(), worker.Dependencies{Config: cfg, Pool: pool})
	if e != nil {
		t.Fatal(e)
	}
	if len(built.Runners) != 1 || len(built.Jobs) != 11 || built.Jobs["node.publication_changed"] == nil {
		t.Fatal("incomplete node build")
	}
	for _, legacy := range []string{"node.routes.withdraw", "node.peer.withdraw"} {
		if built.Jobs[legacy] != nil {
			t.Fatal("legacy withdrawal handler still registered", legacy)
		}
	}
	var count int
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events`).Scan(&count); e != nil || count != 0 {
		t.Fatal("build performed work", count, e)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 2)
	for range 2 {
		go func() { done <- built.Runners[0](ctx) }()
	}
	waitUntil(t, func() bool {
		if e := pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type='node.reconcile' AND delivered_at IS NULL`).Scan(&count); e != nil {
			t.Fatal(e)
		}
		return count == 1
	})
	cancel()
	for range 2 {
		select {
		case e := <-done:
			if e != nil {
				t.Fatal(e)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("node runner ignored cancellation")
		}
	}
	// The shared dispatcher, rather than a runner, obtains the persisted claim.
	app, e := worker.Open(t.Context(), worker.Dependencies{Config: cfg, Pool: pool}, domain)
	if e != nil {
		t.Fatal(e)
	}
	if worked, e := app.Once(t.Context()); e != nil || !worked {
		t.Fatal("seed not dispatched", worked, e)
	}
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type='node.reconcile' AND delivered_at IS NOT NULL AND delivery_receipt IS NOT NULL`).Scan(&count); e != nil || count != 1 {
		t.Fatal("reconciliation not confirmed", count, e)
	}
}
func waitUntil(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for !ready() {
		select {
		case <-deadline.C:
			t.Fatal("condition timed out")
		case <-tick.C:
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}
