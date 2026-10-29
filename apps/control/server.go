// Package control wires public and machine listeners without trusting proxy
// identity headers. The PKI domain supplies Dependencies.Agents.
package control

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Pool    *pgxpool.Pool
	Browser *auth.Browser
	Agents  auth.AgentAuthenticator
	Logger  *slog.Logger
}

func Servers(c config.Config, d Dependencies) (public, agent *http.Server, err error) {
	handlers, e := Handlers(d.Pool, d.Logger)
	if e != nil {
		return nil, nil, e
	}
	commands := &httpapi.Commands{Pool: d.Pool, Retention: time.Duration(c.IdempotencySeconds) * time.Second}
	base := httpapi.Options{Config: c, Sessions: d.Browser, Agents: d.Agents, Commands: commands}
	publicAPI, e := httpapi.New(handlers, base)
	if e != nil {
		return nil, nil, e
	}
	base.AgentListener = true
	agentAPI, e := httpapi.New(handlers, base)
	if e != nil {
		return nil, nil, e
	}
	certificate, e := tls.LoadX509KeyPair(c.TLSCertificateFile, c.TLSKeyFile)
	if e != nil {
		return nil, nil, errors.New("cannot load server TLS identity")
	}
	ca, e := os.ReadFile(c.AgentCAFile)
	if e != nil {
		return nil, nil, errors.New("cannot read agent CA")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, nil, errors.New("invalid agent CA")
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
	public = makeServer(c.PublicAddress, root(publicAPI, false), &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}})
	agent = makeServer(c.AgentAddress, root(agentAPI, true), AgentTLSConfig(certificate, roots))
	return
}

// TLS handshake errors may contain hostile certificate fields; discard the
// standard library's free-form log. Access/security outcomes use structured logs.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func Run(ctx context.Context, c config.Config, servers ...*http.Server) error {
	results := make(chan error, len(servers))
	for _, s := range servers {
		go func() {
			e := s.ListenAndServeTLS("", "")
			if errors.Is(e, http.ErrServerClosed) {
				e = nil
			}
			results <- e
		}()
	}
	var err error
	select {
	case <-ctx.Done():
	case err = <-results:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), time.Duration(c.ShutdownSeconds)*time.Second)
	defer cancel()
	for _, s := range servers {
		if e := s.Shutdown(shutdown); e != nil {
			_ = s.Close()
			if err == nil {
				err = e
			}
		}
	}
	return err
}

// AgentTLSConfig requires direct TLS verification; proxy identity headers do not
// participate in the trust boundary. Runtime revocation remains the PKI hook.
func AgentTLSConfig(certificate tls.Certificate, roots *x509.CertPool) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: roots}
}
