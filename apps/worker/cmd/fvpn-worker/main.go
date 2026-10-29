package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if run(log) != nil {
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
	metrics := prometheus.NewRegistry()
	d, e := worker.New(pool, worker.Jobs(pool), worker.Config{Concurrency: c.WorkerConcurrency, Attempts: c.WorkerAttempts, Lease: time.Duration(c.WorkerLeaseSeconds) * time.Second, JobTimeout: time.Duration(c.WorkerJobSeconds) * time.Second, Poll: time.Duration(c.WorkerPollSeconds) * time.Second, Backoff: time.Duration(c.WorkerBackoffSeconds) * time.Second, MaxBackoff: time.Duration(c.WorkerBackoffMaxSeconds) * time.Second}, metrics, log)
	if e != nil {
		return e
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(metrics, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		probe, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if pool.Ping(probe) != nil {
			apierr.Write(w, r, apierr.New(apierr.DependencyUnavailable, "Database unavailable."))
			return
		}
		w.WriteHeader(200)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.New(apierr.ResourceNotFound, "Route not found."))
	})
	server := &http.Server{Addr: c.MetricsAddress, Handler: httpapi.Transport(log, mux), ReadHeaderTimeout: time.Duration(c.ReadTimeoutSeconds) * time.Second, ReadTimeout: time.Duration(c.ReadTimeoutSeconds) * time.Second, WriteTimeout: time.Duration(c.WriteTimeoutSeconds) * time.Second, IdleTimeout: time.Duration(c.IdleTimeoutSeconds) * time.Second}
	workerDone := make(chan struct{})
	workerError := make(chan error, 1)
	listenerError := make(chan error, 1)
	go func() { workerError <- d.Run(ctx); close(workerDone) }()
	go func() { listenerError <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
	case e = <-listenerError:
	case <-workerDone:
		e = <-workerError
	}
	stop()
	shutdown, cancel := context.WithTimeout(context.Background(), time.Duration(c.ShutdownSeconds)*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		_ = server.Close()
	}
	select {
	case <-workerDone:
	case <-shutdown.Done():
		return shutdown.Err()
	}
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
