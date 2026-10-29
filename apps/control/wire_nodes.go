package control

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/jackc/pgx/v5/pgxpool"
)

func nodeHandlers(pool *pgxpool.Pool, log *slog.Logger) *httpwire.Handlers {
	return httpwire.Configured(pool, log)
}

// wireNodeServers supplies the direct-certificate authenticator and request-local
// enrollment token before either listener starts. Existing browser/health routes
// remain owned by the server framework. No proxy identity headers are used.
func wireNodeServers(c config.Config, d Dependencies, public, agent *http.Server) error {
	if d.Agents == nil {
		r, e := httpwire.Open(context.Background(), d.Pool, d.Logger)
		if e != nil {
			return e
		}
		d.Agents = r.PKI
	}
	handlers, e := Handlers(d.Pool, d.Logger)
	if e != nil {
		return e
	}
	for _, entry := range []struct {
		server *http.Server
		agent  bool
	}{{public, false}, {agent, true}} {
		api, e := httpapi.New(handlers, httpapi.Options{Config: c, Sessions: d.Browser, Agents: d.Agents, AgentListener: entry.agent, Commands: &httpapi.Commands{Pool: d.Pool, Retention: time.Duration(c.IdempotencySeconds) * time.Second}})
		if e != nil {
			return e
		}
		api = httpapi.Transport(d.Logger, httpwire.RequestContext(api))
		original := entry.server.Handler
		entry.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/") {
				api.ServeHTTP(w, r)
				return
			}
			original.ServeHTTP(w, r)
		})
	}
	return nil
}
