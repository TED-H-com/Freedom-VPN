// Command fvpn-e2e-control composes real control with the TEST ONLY D-087 seam.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		var setting interface{ Setting() string }
		if errors.As(err, &setting) {
			log.Error("invalid startup setting", "setting", setting.Setting())
		}
		log.Error("control stopped", "error_code", "STARTUP_OR_RUNTIME_FAILURE")
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	file := flag.String("config", "", "optional JSON configuration")
	migrate := flag.Bool("migrate-on-start", false, "apply embedded migrations before serving")
	command := flag.String("fixture-command", "", "TEST ONLY: false, true, evidence")
	flag.Parse()
	if *command != "" {
		return granttrace.Command(*command, os.Stdout)
	}
	c, e := config.Load(*file)
	if e != nil {
		return e
	}
	if *migrate {
		c.MigrateOnStart = true
	}
	if e = c.ValidateControl(); e != nil {
		return e
	}
	log.Info("effective configuration", "config", c.Effective())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn, e := config.ReadSecret(c.DatabaseURLFile)
	if e != nil {
		return e
	}
	pool, e := db.Open(ctx, dsn, db.OpenOptions{})
	if e != nil {
		return e
	}
	defer pool.Close()
	if c.MigrateOnStart {
		if e = db.Migrate(ctx, pool); e != nil {
			return e
		}
	}
	secret := ""
	if c.OIDCSecretFile != "" {
		secret, e = config.ReadSecret(c.OIDCSecretFile)
		if e != nil {
			return e
		}
	}
	store := auth.PostgresStore{Pool: pool}
	browser, e := auth.NewBrowser(ctx, auth.BrowserConfig{Issuer: c.OIDCIssuer, ClientID: c.OIDCClientID, ClientSecret: secret, Origin: c.PublicOrigin, AdminClaim: c.AdminClaim, AdminValue: c.AdminValue, MFAAMR: c.MFAAMR, MFAACR: c.MFAACR, Idle: time.Duration(c.SessionIdleSeconds) * time.Second, Absolute: time.Duration(c.SessionAbsoluteSeconds) * time.Second, LoginTTL: time.Duration(c.LoginTTLSeconds) * time.Second, RecentAuth: time.Duration(c.RecentAuthSeconds) * time.Second, SubjectPerMinute: c.LoginSubjectPerMinute, MaxRatePrincipals: c.MaxRatePrincipals}, store, nil)
	if e != nil {
		return e
	}
	grants := nodestest.NewMemberGrantEligibility()
	trace := &granttrace.Recorder{}
	domains := control.Domains()
	for i := range domains {
		if domains[i].Name == "nodes" {
			domains[i].Build = func(ctx context.Context, d control.Dependencies) (control.Built, error) {
				cfg, err := config.LoadNodes(d.Config.NodesConfigFile)
				if err != nil {
					return control.Built{}, err
				}
				runtime, err := httpwire.Open(ctx, d.Pool, cfg, d.Logger, httpwire.WithTestOnlyGrantEligibilityReaderFactory(grants.ReaderFactory))
				if err != nil {
					return control.Built{}, err
				}
				return control.Built{Handlers: []any{&httpwire.Handlers{Runtime: runtime}}, Agents: runtime.PKI, Middleware: httpwire.RequestContext}, nil
			}
		}
	}
	app, e := control.Servers(ctx, control.Dependencies{Config: c, Pool: pool, Browser: browser, Logger: log}, domains...)
	if e != nil {
		return e
	}
	app.Agent.Handler = trace.Wrap(app.Agent.Handler)
	closeFixture, e := trace.Listen(grants.SetMemberGrantsReady)
	if e != nil {
		return e
	}
	defer closeFixture()
	return control.Run(ctx, app)
}
