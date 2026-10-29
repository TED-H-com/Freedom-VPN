package main

import (
	"context"
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
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if run(log) != nil {
		log.Error("control stopped", "error_code", "STARTUP_OR_RUNTIME_FAILURE")
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	file := flag.String("config", "", "optional JSON configuration")
	migrate := flag.Bool("migrate-on-start", false, "apply embedded migrations before serving")
	flag.Parse()
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
	browser, e := auth.NewBrowser(ctx, auth.BrowserConfig{Issuer: c.OIDCIssuer, ClientID: c.OIDCClientID, ClientSecret: secret, Origin: c.PublicOrigin, AdminClaim: c.AdminClaim, AdminValue: c.AdminValue, MFAAMR: c.MFAAMR, MFAACR: c.MFAACR, Idle: time.Duration(c.SessionIdleSeconds) * time.Second, Absolute: time.Duration(c.SessionAbsoluteSeconds) * time.Second, LoginTTL: time.Duration(c.LoginTTLSeconds) * time.Second, RecentAuth: time.Duration(c.RecentAuthSeconds) * time.Second}, store, nil)
	if e != nil {
		return e
	}
	// The PKI task wires its database-backed AgentAuthenticator here. nil is a
	// fail-closed boundary: even a CA-valid certificate grants no API authority.
	public, agent, e := control.Servers(c, control.Dependencies{Pool: pool, Browser: browser, Logger: log})
	if e != nil {
		return e
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup, cancel := context.WithTimeout(ctx, 10*time.Second)
				if store.Cleanup(cleanup, time.Duration(c.SessionIdleSeconds)*time.Second) != nil {
					log.Warn("authentication cleanup failed", "error_code", "CLEANUP_FAILED")
				}
				if (&httpapi.Commands{Pool: pool}).Cleanup(cleanup) != nil {
					log.Warn("idempotency cleanup failed", "error_code", "CLEANUP_FAILED")
				}
				cancel()
			}
		}
	}()
	return control.Run(ctx, c, public, agent)
}
