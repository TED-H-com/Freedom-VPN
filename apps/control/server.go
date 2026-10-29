// Package control wires public and machine listeners without trusting proxy
// identity headers. Domains supply handlers, identity verification and runners.
package control

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Dependencies are supplied once by the binary; domains never open another pool.
type Dependencies struct {
	Config  config.Config
	Pool    *pgxpool.Pool
	Browser *auth.Browser
	Logger  *slog.Logger
}
type Domain struct {
	Name  string
	Build func(context.Context, Dependencies) (Built, error)
}
type Built struct {
	Handlers   []any
	Agents     auth.AgentAuthenticator
	Middleware func(http.Handler) http.Handler
	Runners    []func(context.Context) error
}

// Application owns the two listeners and the lifecycle of all domain runners.
// Only the composition layer may construct or wrap the listeners' handlers.
type Application struct {
	Public, Agent *http.Server
	runners       []func(context.Context) error
	shutdown      time.Duration
}

// Servers builds the default Domains list, or an explicit list for composition
// tests. Middleware runs in registration order, inside httpapi.Transport, on
// both listeners. Nothing listens or starts a goroutine until Run.
func Servers(ctx context.Context, d Dependencies, domains ...Domain) (*Application, error) {
	if domains == nil {
		domains = Domains()
	}
	var sets []any
	var agents auth.AgentAuthenticator
	var middleware []func(http.Handler) http.Handler
	var runners []func(context.Context) error
	built := make([]Built, 0, len(domains))
	names := map[string]bool{}
	for _, domain := range domains {
		if domain.Name == "" || domain.Build == nil || names[domain.Name] {
			return nil, errors.New("invalid or duplicate control domain")
		}
		names[domain.Name] = true
		result, e := domain.Build(ctx, d)
		if e != nil {
			return nil, fmt.Errorf("control domain %s: %w", domain.Name, e)
		}
		if result.Agents != nil {
			if agents != nil {
				return nil, errors.New("multiple control domains provide Agents")
			}
			agents = result.Agents
		}
		for _, run := range result.Runners {
			if run == nil {
				return nil, fmt.Errorf("control domain %s: nil runner", domain.Name)
			}
		}
		sets = append(sets, result.Handlers...)
		if result.Middleware != nil {
			middleware = append(middleware, result.Middleware)
		}
		runners = append(runners, result.Runners...)
		built = append(built, result)
	}
	handlers, e := httpapi.Compose(sets...)
	if e != nil {
		return nil, e
	}
	c := d.Config
	if d.Pool == nil || d.Browser == nil || c.ShutdownSeconds < 1 {
		return nil, errors.New("control database, browser and shutdown_seconds required")
	}
	wrap := func(h http.Handler) http.Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			h = middleware[i](h)
		}
		return h
	}
	commands := &httpapi.Commands{Pool: d.Pool, Retention: time.Duration(c.IdempotencySeconds) * time.Second}
	base := httpapi.Options{Config: c, Sessions: d.Browser, Agents: agents, Commands: commands}
	publicAPI, e := httpapi.New(handlers, base)
	if e != nil {
		return nil, e
	}
	base.AgentListener = true
	agentAPI, e := httpapi.New(handlers, base)
	if e != nil {
		return nil, e
	}
	certificate, e := tls.LoadX509KeyPair(c.TLSCertificateFile, c.TLSKeyFile)
	if e != nil {
		return nil, errors.New("cannot load server TLS identity")
	}
	ca, e := os.ReadFile(c.AgentCAFile)
	if e != nil {
		return nil, errors.New("cannot read agent CA")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, errors.New("invalid agent CA")
	}
	authMux := http.NewServeMux()
	d.Browser.Register(authMux)
	authRoutes := httpapi.BrowserRoutes(c, authMux)
	root := func(api http.Handler, isAgent bool) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			switch r.URL.Path {
			case "/healthz":
				if r.Method != "GET" {
					apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "GET required."))
					return
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte("ok\n"))
				return
			case "/readyz":
				if r.Method != "GET" {
					apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "GET required."))
					return
				}
				ctx, cancel := context.WithTimeout(r.Context(), time.Second)
				defer cancel()
				if e := d.Pool.Ping(ctx); e != nil {
					apierr.Write(w, r, apierr.New(apierr.DependencyUnavailable, "Database unavailable."))
					return
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte("ready\n"))
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/v1/") {
				api.ServeHTTP(w, r)
				return
			}
			if !isAgent {
				method, ok := map[string]string{"/auth/login": "GET", "/auth/callback": "GET", "/auth/session": "GET", "/auth/logout": "POST"}[r.URL.Path]
				if ok {
					if r.Method != method {
						apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
						return
					}
					authRoutes.ServeHTTP(w, r)
					return
				}
			}
			apierr.Write(w, r, apierr.New(apierr.ResourceNotFound, "Route not found."))
		})
	}
	makeServer := func(address string, h http.Handler, t *tls.Config) *http.Server {
		return &http.Server{Addr: address, Handler: httpapi.Transport(d.Logger, h), TLSConfig: t, ReadHeaderTimeout: time.Duration(c.ReadTimeoutSeconds) * time.Second, ReadTimeout: time.Duration(c.ReadTimeoutSeconds) * time.Second, WriteTimeout: time.Duration(c.WriteTimeoutSeconds) * time.Second, IdleTimeout: time.Duration(c.IdleTimeoutSeconds) * time.Second, MaxHeaderBytes: 32 << 10, ErrorLog: slog.NewLogLogger(slog.NewTextHandler(discardWriter{}, nil), slog.LevelError)}
	}
	public := makeServer(c.PublicAddress, root(wrap(publicAPI), false), &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}})
	agent := makeServer(c.AgentAddress, root(wrap(agentAPI), true), AgentTLSConfig(certificate, roots))
	if d.Logger != nil {
		for i, domain := range domains {
			d.Logger.Info("control domain built", "domain", domain.Name, "operations", operationNames(built[i].Handlers), "agent_authenticator", built[i].Agents != nil, "runners", len(built[i].Runners))
		}
	}
	return &Application{Public: public, Agent: agent, runners: runners, shutdown: time.Duration(c.ShutdownSeconds) * time.Second}, nil
}

// TLS handshake errors may contain hostile certificate fields; discard the
// standard library's free-form log. Access/security outcomes use structured logs.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// Run cancels all runners when a listener or runner fails, then waits for both
// HTTP drain and runner completion under one shutdown deadline.
func Run(ctx context.Context, app *Application) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	servers := []*http.Server{app.Public, app.Agent}
	results := make(chan error, len(servers)+len(app.runners))
	launch := func(run func(context.Context) error) { go func() { results <- run(ctx) }() }
	for _, s := range servers {
		launch(func(context.Context) error {
			e := s.ListenAndServeTLS("", "")
			if errors.Is(e, http.ErrServerClosed) {
				return nil
			}
			return e
		})
	}
	for _, run := range app.runners {
		launch(run)
	}
	remaining := cap(results)
	var err error
	select {
	case <-ctx.Done():
	case err = <-results:
		remaining--
	}
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		err = nil
	}
	stop()
	shutdown, cancel := context.WithTimeout(context.Background(), app.shutdown)
	defer cancel()
	drained := make(chan error, len(servers))
	for _, s := range servers {
		go func() {
			e := s.Shutdown(shutdown)
			if e != nil {
				_ = s.Close()
			}
			drained <- e
		}()
	}
	for range servers {
		if e := <-drained; e != nil {
			err = errors.Join(err, e)
		}
	}
	for remaining > 0 {
		select {
		case e := <-results:
			remaining--
			if e != nil && !errors.Is(e, context.Canceled) {
				err = errors.Join(err, e)
			}
		case <-shutdown.Done():
			return errors.Join(err, shutdown.Err())
		}
	}
	return err
}

func operationNames(sets []any) []string {
	names := []string{}
	contract := reflect.TypeFor[gen.StrictServerInterface]()
	for i := 0; i < contract.NumMethod(); i++ {
		method := contract.Method(i)
		for _, set := range sets {
			actual := reflect.ValueOf(set).MethodByName(method.Name)
			if actual.IsValid() && actual.Type() == method.Type {
				names = append(names, method.Name)
			}
		}
	}
	return names
}

func core() Domain {
	return Domain{Name: "control-core", Build: func(_ context.Context, d Dependencies) (Built, error) {
		cleanup := func(ctx context.Context) error {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					probe, cancel := context.WithTimeout(ctx, 10*time.Second)
					authErr := (auth.PostgresStore{Pool: d.Pool}).Cleanup(probe, time.Duration(d.Config.SessionIdleSeconds)*time.Second)
					commandErr := (&httpapi.Commands{Pool: d.Pool}).Cleanup(probe)
					cancel()
					if ctx.Err() != nil {
						return nil
					}
					if (authErr != nil || commandErr != nil) && d.Logger != nil {
						d.Logger.Warn("authentication or idempotency cleanup failed", "error_code", "CLEANUP_FAILED")
					}
				}
			}
		}
		return Built{Handlers: []any{operations.Store{Pool: d.Pool}, &audit.Store{Pool: d.Pool, Logger: d.Logger}}, Runners: []func(context.Context) error{cleanup}}, nil
	}}
}

// AgentTLSConfig requires direct TLS verification; proxy identity headers do not
// participate in the trust boundary. Runtime revocation remains the PKI hook.
func AgentTLSConfig(certificate tls.Certificate, roots *x509.CertPool) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: roots}
}
