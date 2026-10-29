//go:build integration

package httpadapter_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// All identities are generated at test time, never fixture credentials.
func testTLSConfig(t *testing.T, c config.Config) config.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "TEST ONLY"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	raw, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	c.TLSCertificateFile, c.AgentCAFile = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "cert.pem")
	c.TLSKeyFile = filepath.Join(dir, "key.pem")
	if err = os.WriteFile(c.TLSCertificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: raw}), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(c.TLSKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600); err != nil {
		t.Fatal(err)
	}
	c.PublicAddress, c.AgentAddress, c.MetricsAddress = "127.0.0.1:0", "127.0.0.1:0", "127.0.0.1:0"
	return c
}

func TestAPI01_CompositionRejectsOverlappingAuthorityAndMissingSettings(t *testing.T) {
	handler := control.Domain{Name: "one", Build: func(context.Context, control.Dependencies) (control.Built, error) {
		return control.Built{Handlers: []any{httpadapter.Handlers{}}}, nil
	}}
	second := handler
	second.Name = "two"
	if _, e := control.Servers(t.Context(), control.Dependencies{}, handler, second); e == nil || !strings.Contains(e.Error(), "duplicate handler") {
		t.Fatal("overlap accepted", e)
	}
	authenticator := auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
		return auth.Principal{}, errors.New("TEST ONLY deny")
	})
	handler.Build = func(context.Context, control.Dependencies) (control.Built, error) {
		return control.Built{Agents: authenticator}, nil
	}
	second.Build = handler.Build
	if _, e := control.Servers(t.Context(), control.Dependencies{}, handler, second); e == nil || !strings.Contains(e.Error(), "Agents") {
		t.Fatal("duplicate authority accepted", e)
	}
	job := worker.Domain{Name: "one", Build: func(context.Context, worker.Dependencies) (worker.Built, error) {
		return worker.Built{Jobs: worker.Registry{"TEST.event": func(context.Context, db.ClaimedEvent) (string, error) { return "test", nil }}}, nil
	}}
	other := job
	other.Name = "two"
	if _, e := worker.Open(t.Context(), worker.Dependencies{}, job, other); e == nil || !strings.Contains(e.Error(), "duplicate worker event") {
		t.Fatal("duplicate event accepted", e)
	}
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing.json")} {
		c := config.Defaults()
		c.MembersConfigFile = path
		if _, e := control.Servers(t.Context(), control.Dependencies{Config: c}); e == nil || !strings.Contains(e.Error(), "members_config_file") {
			t.Fatal("control accepted missing settings", e)
		}
		if _, e := worker.Open(t.Context(), worker.Dependencies{Config: c}); e == nil || !strings.Contains(e.Error(), "members_config_file") {
			t.Fatal("worker accepted missing settings", e)
		}
	}
}

func TestAPI01_CompositionMiddlewareBothListenersInsideTransport(t *testing.T) {
	f := newFixture(t)
	nodes := nodestest.New(t, nodestest.NetBird{})
	f.config.NodesConfigFile, f.config.AgentCAFile = nodes.Path, nodes.Nodes.CACertificateFile
	type orderKey struct{}
	domains := control.Domains()
	// Replace only the test service dependencies; keep the production middleware.
	for i := range domains {
		if domains[i].Name == "members" {
			domains[i].Build = func(_ context.Context, d control.Dependencies) (control.Built, error) {
				return control.Built{Handlers: []any{httpadapter.Handlers{Pool: d.Pool, Services: &f.services.HTTP}}, Middleware: httpadapter.RequestContext}, nil
			}
		}
	}
	first := control.Domain{Name: "TEST-first", Build: func(context.Context, control.Dependencies) (control.Built, error) {
		return control.Built{Middleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if id, ok := apierr.RequestIDFromContext(r.Context()); !ok || id == "" {
					t.Error("middleware outside Transport")
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), orderKey{}, "first")))
			})
		}}, nil
	}}
	second := control.Domain{Name: "TEST-second", Build: func(context.Context, control.Dependencies) (control.Built, error) {
		return control.Built{Middleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Context().Value(orderKey{}) != "first" {
					t.Error("wrong middleware order")
				}
				w.Header().Set("X-Test-Composition", "both")
				next.ServeHTTP(w, r)
			})
		}}, nil
	}}
	app, e := control.Servers(t.Context(), control.Dependencies{Config: f.config, Pool: f.pool, Browser: f.browser}, append([]control.Domain{first, second}, domains...)...)
	if e != nil {
		t.Fatal(e)
	}
	raw := "{ \n \"id\":\"TEST-original\",\"type\":\"invoice.paid\",\"subscription_id\":\"TEST-sub\" }"
	for _, server := range []*http.Server{app.Public, app.Agent} {
		r := httptest.NewRequest("POST", "/api/v1/webhooks/fake", strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Stripe-Signature", f.payment.Sign([]byte(raw)))
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, r)
		if server == app.Public {
			requireStatus(t, w, 202)
		} else {
			requireStatus(t, w, 401)
		}
		if w.Header().Get("X-Test-Composition") != "both" {
			t.Fatal("middleware missing")
		}
	}
	if f.count(`SELECT count(*) FROM inbox_events`) != 1 {
		t.Fatal("signature lost or replay duplicated")
	}
}

func TestAPI01_CompositionWorkerContextAndRunnerSupervision(t *testing.T) {
	f := newFixture(t)
	type valueKey struct{}
	id := uuid.New()
	if e := db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return db.EnqueueOutbox(t.Context(), tx, db.OutboxEvent{ID: id, AggregateID: uuid.New(), AggregateType: "TEST ONLY", AggregateVersion: "1", EventType: "TEST.context", SchemaVersion: 1, Payload: []byte(`{}`)})
	}); e != nil {
		t.Fatal(e)
	}
	observed := false
	jobs := worker.Domain{Name: "TEST-context", Build: func(_ context.Context, d worker.Dependencies) (worker.Built, error) {
		if d.Metrics == nil || d.Pool != f.pool || d.Config.WorkerAttempts != f.config.WorkerAttempts {
			return worker.Built{}, errors.New("incomplete worker build dependencies")
		}
		return worker.Built{Context: func(ctx context.Context) context.Context { return context.WithValue(ctx, valueKey{}, "TEST ONLY") }, Jobs: worker.Registry{"TEST.context": func(ctx context.Context, e db.ClaimedEvent) (string, error) {
			if ctx.Value(valueKey{}) != "TEST ONLY" || e.ID != id {
				return "", errors.New("missing worker context")
			}
			observed = true
			return "outbox_event:" + id.String(), nil
		}}}, nil
	}}
	app, e := worker.Open(t.Context(), worker.Dependencies{Config: f.config, Pool: f.pool}, jobs)
	if e != nil {
		t.Fatal(e)
	}
	if worked, e := app.Once(t.Context()); e != nil || !worked || !observed {
		t.Fatal("context did not reach handler", worked, e)
	}
	if f.count(`SELECT count(*) FROM outbox_events WHERE id=$1 AND delivery_receipt IS NOT NULL`, id) != 1 {
		t.Fatal("missing fenced receipt")
	}
	for _, process := range []string{"control", "worker"} {
		t.Run(process, func(t *testing.T) {
			started, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			sentinel := errors.New("TEST ONLY runner failed")
			runners := []func(context.Context) error{
				func(context.Context) error { <-started; return sentinel },
				func(ctx context.Context) error { close(started); <-ctx.Done(); close(cancelled); <-release; return nil },
			}
			var run func(context.Context) error
			if process == "control" {
				a, e := control.Servers(t.Context(), control.Dependencies{Config: f.config, Pool: f.pool, Browser: f.browser}, control.Domain{Name: "TEST-runners", Build: func(context.Context, control.Dependencies) (control.Built, error) {
					return control.Built{Runners: runners}, nil
				}})
				if e != nil {
					t.Fatal(e)
				}
				run = func(ctx context.Context) error { return control.Run(ctx, a) }
			} else {
				a, e := worker.Open(t.Context(), worker.Dependencies{Config: f.config, Pool: f.pool}, worker.Domain{Name: "TEST-runners", Build: func(context.Context, worker.Dependencies) (worker.Built, error) {
					return worker.Built{Runners: runners}, nil
				}})
				if e != nil {
					t.Fatal(e)
				}
				run = a.Run
			}
			done := make(chan error, 1)
			go func() { done <- run(t.Context()) }()
			select {
			case <-cancelled:
			case <-time.After(5 * time.Second):
				close(release)
				t.Fatal("runner failure did not cancel siblings")
			}
			select {
			case e := <-done:
				close(release)
				t.Fatal("did not wait for runner", e)
			case <-time.After(20 * time.Millisecond):
			}
			close(release)
			select {
			case e := <-done:
				if !errors.Is(e, sentinel) {
					t.Fatal("lost runner failure", e)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("shutdown did not finish")
			}
		})
	}
}

func TestAPI01_CompositionBoundsRunnerShutdown(t *testing.T) {
	f := newFixture(t)
	c := f.config
	c.ShutdownSeconds = 1
	for _, process := range []string{"control", "worker"} {
		t.Run(process, func(t *testing.T) {
			started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			runner := func(context.Context) error { close(started); <-release; close(finished); return nil }
			var run func(context.Context) error
			if process == "control" {
				app, e := control.Servers(t.Context(), control.Dependencies{Config: c, Pool: f.pool, Browser: f.browser}, control.Domain{Name: "TEST-timeout", Build: func(context.Context, control.Dependencies) (control.Built, error) {
					return control.Built{Runners: []func(context.Context) error{runner}}, nil
				}})
				if e != nil {
					t.Fatal(e)
				}
				run = func(ctx context.Context) error { return control.Run(ctx, app) }
			} else {
				app, e := worker.Open(t.Context(), worker.Dependencies{Config: c, Pool: f.pool}, worker.Domain{Name: "TEST-timeout", Build: func(context.Context, worker.Dependencies) (worker.Built, error) {
					return worker.Built{Runners: []func(context.Context) error{runner}}, nil
				}})
				if e != nil {
					t.Fatal(e)
				}
				run = app.Run
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- run(ctx) }()
			<-started
			cancel()
			select {
			case e := <-done:
				if !errors.Is(e, context.DeadlineExceeded) {
					t.Error("missing shutdown timeout", e)
				}
			case <-time.After(5 * time.Second):
				t.Error("shutdown exceeded configured bound")
			}
			close(release)
			<-finished
		})
	}
}
