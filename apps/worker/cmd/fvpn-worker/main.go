package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		var setting interface{ Setting() string }
		if errors.As(err, &setting) {
			log.Error("invalid startup setting", "setting", setting.Setting())
		}
		log.Error("worker stopped", "error_code", "STARTUP_OR_RUNTIME_FAILURE")
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	file := flag.String("config", "", "optional JSON configuration")
	flag.Parse()
	c, e := config.Load(*file)
	if e != nil {
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
	app, e := worker.Open(ctx, worker.Dependencies{Config: c, Pool: pool, Logger: log})
	if e != nil {
		return e
	}
	return app.Run(ctx)
}
