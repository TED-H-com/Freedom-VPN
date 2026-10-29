// Package worker dispatches durable, fenced, at-least-once work (§14.5, §17.1).
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"time"

	centralconfig "github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler must honor cancellation and make retries safe with stable external IDs,
// domain uniqueness and read-back. Receipt references already persisted evidence.
type Handler func(context.Context, db.ClaimedEvent) (receipt string, err error)
type Registry map[string]Handler

func (r Registry) Register(event string, h Handler) {
	if event == "" || h == nil || r[event] != nil {
		panic("invalid or duplicate worker handler")
	}
	r[event] = h
}

type Config struct {
	Concurrency, Attempts                        int
	Lease, JobTimeout, Poll, Backoff, MaxBackoff time.Duration
}
type Dispatcher struct {
	pool     *pgxpool.Pool
	registry Registry
	config   Config
	name     string
	log      *slog.Logger
	results  *prometheus.CounterVec
	active   prometheus.Gauge
	duration prometheus.Histogram
	dead     prometheus.Gauge
}

func New(pool *pgxpool.Pool, r Registry, c Config, metrics prometheus.Registerer, log *slog.Logger) (*Dispatcher, error) {
	if pool == nil || c.Concurrency < 1 || c.Concurrency > 1000 || c.Attempts < 1 || c.Lease <= c.JobTimeout || c.JobTimeout <= 0 || c.Poll <= 0 || c.Backoff <= 0 || c.MaxBackoff < c.Backoff || c.MaxBackoff > 24*time.Hour || c.Lease > 24*time.Hour {
		return nil, errors.New("invalid worker configuration")
	}
	registry := Registry{}
	for k, v := range r {
		if k == "" || v == nil {
			return nil, errors.New("invalid worker handler")
		}
		registry[k] = v
	}
	d := &Dispatcher{pool: pool, registry: registry, config: c, name: uuid.Must(uuid.NewV7()).String(), log: log,
		results:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "fvpn_worker_events_total", Help: "Outbox dispatch outcomes."}, []string{"result"}),
		active:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "fvpn_worker_active", Help: "Active dispatch jobs."}),
		duration: prometheus.NewHistogram(prometheus.HistogramOpts{Name: "fvpn_worker_job_seconds", Help: "Dispatch time including read-back."}),
		dead:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "fvpn_worker_dead_letters", Help: "Persisted dead-letter count."})}
	if metrics != nil {
		for _, m := range []prometheus.Collector{d.results, d.active, d.duration, d.dead} {
			if e := metrics.Register(m); e != nil {
				return nil, e
			}
		}
	}
	return d, nil
}
func (d *Dispatcher) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	for i := 0; i < d.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				worked, e := d.Once(ctx)
				if e != nil && ctx.Err() == nil && d.log != nil {
					d.log.Error("outbox dispatch failed", "error_code", "DISPATCH_FAILED")
				}
				if !worked || e != nil {
					t := time.NewTimer(d.config.Poll)
					select {
					case <-ctx.Done():
						t.Stop()
						return
					case <-t.C:
					}
				}
			}
		}()
	}
	wg.Wait()
	return nil
}
func (d *Dispatcher) Once(ctx context.Context) (bool, error) {
	events, e := db.ClaimOutbox(ctx, d.pool, d.name, 1, d.config.Lease)
	if e != nil {
		return false, e
	}
	if len(events) == 0 {
		var n int
		if e = d.pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE dead_lettered_at IS NOT NULL`).Scan(&n); e == nil {
			d.dead.Set(float64(n))
		}
		return false, e
	}
	event := events[0]
	d.active.Inc()
	defer d.active.Dec()
	start := time.Now()
	defer func() { d.duration.Observe(time.Since(start).Seconds()) }()
	if event.Attempt > d.config.Attempts {
		return true, d.deadLetter(ctx, event, "ATTEMPTS_EXHAUSTED")
	}
	handler := d.registry[event.EventType]
	code := "HANDLER_FAILED"
	var receipt string
	if handler == nil {
		e = errors.New("unregistered event")
		code = "HANDLER_MISSING"
	} else {
		jobCtx, cancel := context.WithTimeout(ctx, d.config.JobTimeout)
		receipt, e = invoke(jobCtx, handler, event)
		if e == nil {
			e = jobCtx.Err()
		}
		cancel()
	}
	if e == nil && receipt != "" {
		e = d.complete(ctx, event, receipt)
		if e == nil {
			d.results.WithLabelValues("delivered").Inc()
		}
		return true, e
	}
	// Canceled/uncertain work is retried under its original event ID. Lease expiry
	// recovers interrupted processes; no success is invented during shutdown.
	if ctx.Err() != nil {
		return true, ctx.Err()
	}
	if event.Attempt >= d.config.Attempts {
		return true, d.deadLetter(ctx, event, code)
	}
	delay := d.config.Backoff
	for i := 1; i < event.Attempt && delay < d.config.MaxBackoff; i++ {
		delay = min(delay*2, d.config.MaxBackoff)
	}
	delay = delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
	e = db.RetryOutbox(ctx, d.pool, event, delay)
	if e == nil {
		d.results.WithLabelValues("retry").Inc()
	}
	return true, e
}
func invoke(ctx context.Context, h Handler, e db.ClaimedEvent) (receipt string, err error) {
	defer func() {
		if recover() != nil {
			receipt = ""
			err = errors.New("handler panic")
		}
	}()
	return h(ctx, e)
}
func (d *Dispatcher) deadLetter(ctx context.Context, event db.ClaimedEvent, code string) error {
	e := db.InTx(ctx, d.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, `UPDATE outbox_events SET dead_lettered_at=clock_timestamp(),last_error_code=$4,available_at='infinity',lease_owner=NULL,lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_owner=$2 AND lease_token=$3 AND lease_until>clock_timestamp() AND delivered_at IS NULL AND dead_lettered_at IS NULL`, event.ID, event.Worker, event.LeaseToken, code)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return db.ErrLeaseLost
		}
		return db.AppendAudit(ctx, tx, db.AuditEvent{Actor: "system", ActorType: "system", Action: "outbox.dead_letter", Resource: event.AggregateType + ":" + event.AggregateID.String(), Result: "failed", SchemaVersion: 1, Before: []byte(`{}`), After: []byte(`{"dead_lettered":true}`)})
	})
	if e == nil {
		d.results.WithLabelValues("dead_letter").Inc()
		d.dead.Inc()
		if d.log != nil {
			d.log.Warn("outbox dead letter", "event_id", event.ID, "attempt", event.Attempt, "error_code", code)
		}
	}
	return e
}

func (d *Dispatcher) complete(ctx context.Context, event db.ClaimedEvent, receipt string) error {
	if len(receipt) > 512 {
		return errors.New("invalid evidence reference")
	}
	tag, e := d.pool.Exec(ctx, `UPDATE outbox_events SET delivered_at=clock_timestamp(),delivery_receipt=$4,lease_owner=NULL,lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_owner=$2 AND lease_token=$3 AND lease_until>clock_timestamp() AND delivered_at IS NULL AND dead_lettered_at IS NULL`, event.ID, event.Worker, event.LeaseToken, receipt)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return db.ErrLeaseLost
	}
	return nil
}

// Dependencies belong to the process. Domain builds must not start goroutines,
// read flags/environment, or open pools. Context only adds immutable values.
type Dependencies struct {
	Config  centralconfig.Config
	Pool    *pgxpool.Pool
	Metrics *prometheus.Registry
	Logger  *slog.Logger
}
type Domain struct {
	Name  string
	Build func(context.Context, Dependencies) (Built, error)
}
type Built struct {
	Jobs    Registry
	Context func(context.Context) context.Context
	Runners []func(context.Context) error
}
type Application struct {
	dispatcher *Dispatcher
	contexts   []func(context.Context) context.Context
	runners    []func(context.Context) error
	server     *http.Server
	shutdown   time.Duration
}

// Open validates and merges all domain registrations before any work is claimed.
// With no explicit domains it uses the production Domains list.
func Open(ctx context.Context, d Dependencies, domains ...Domain) (*Application, error) {
	if d.Metrics == nil {
		d.Metrics = prometheus.NewRegistry()
	}
	if domains == nil {
		domains = Domains()
	}
	names := map[string]bool{}
	registry := Registry{}
	app := &Application{shutdown: time.Duration(d.Config.ShutdownSeconds) * time.Second}
	built := make([]Built, 0, len(domains))
	for _, domain := range domains {
		if domain.Name == "" || domain.Build == nil || names[domain.Name] {
			return nil, errors.New("invalid or duplicate worker domain")
		}
		names[domain.Name] = true
		result, e := domain.Build(ctx, d)
		if e != nil {
			return nil, fmt.Errorf("worker domain %s: %w", domain.Name, e)
		}
		for name, handler := range result.Jobs {
			if name == "" || handler == nil {
				return nil, fmt.Errorf("worker domain %s: invalid handler", domain.Name)
			}
			if registry[name] != nil {
				return nil, fmt.Errorf("duplicate worker event: %s", name)
			}
			registry[name] = handler
		}
		for _, run := range result.Runners {
			if run == nil {
				return nil, fmt.Errorf("worker domain %s: nil runner", domain.Name)
			}
		}
		if result.Context != nil {
			app.contexts = append(app.contexts, result.Context)
		}
		app.runners = append(app.runners, result.Runners...)
		built = append(built, result)
	}
	c := d.Config
	if app.shutdown <= 0 {
		return nil, errors.New("shutdown_seconds required")
	}
	dispatcher, e := New(d.Pool, registry, Config{Concurrency: c.WorkerConcurrency, Attempts: c.WorkerAttempts, Lease: time.Duration(c.WorkerLeaseSeconds) * time.Second, JobTimeout: time.Duration(c.WorkerJobSeconds) * time.Second, Poll: time.Duration(c.WorkerPollSeconds) * time.Second, Backoff: time.Duration(c.WorkerBackoffSeconds) * time.Second, MaxBackoff: time.Duration(c.WorkerBackoffMaxSeconds) * time.Second}, d.Metrics, d.Logger)
	if e != nil {
		return nil, e
	}
	app.dispatcher = dispatcher
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(d.Metrics, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		probe, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if d.Pool.Ping(probe) != nil {
			apierr.Write(w, r, apierr.New(apierr.DependencyUnavailable, "Database unavailable."))
			return
		}
		w.WriteHeader(200)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.New(apierr.ResourceNotFound, "Route not found."))
	})
	app.server = &http.Server{Addr: c.MetricsAddress, Handler: httpapi.Transport(d.Logger, mux), ReadHeaderTimeout: time.Duration(c.ReadTimeoutSeconds) * time.Second, ReadTimeout: time.Duration(c.ReadTimeoutSeconds) * time.Second, WriteTimeout: time.Duration(c.WriteTimeoutSeconds) * time.Second, IdleTimeout: time.Duration(c.IdleTimeoutSeconds) * time.Second}
	if d.Logger != nil {
		for i, domain := range domains {
			events := make([]string, 0, len(built[i].Jobs))
			for name := range built[i].Jobs {
				events = append(events, name)
			}
			slices.Sort(events)
			d.Logger.Info("worker domain built", "domain", domain.Name, "events", events, "runners", len(built[i].Runners))
		}
	}
	return app, nil
}
func (a *Application) context(ctx context.Context) context.Context {
	for _, values := range a.contexts {
		ctx = values(ctx)
	}
	return ctx
}

// Once dispatches one job through the same context composition as Run.
func (a *Application) Once(ctx context.Context) (bool, error) {
	return a.dispatcher.Once(a.context(ctx))
}

func (a *Application) Run(ctx context.Context) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	ctx = a.context(ctx)
	results := make(chan error, len(a.runners)+2)
	go func() { results <- a.dispatcher.Run(ctx) }()
	go func() {
		e := a.server.ListenAndServe()
		if errors.Is(e, http.ErrServerClosed) {
			e = nil
		}
		results <- e
	}()
	for _, run := range a.runners {
		go func() { results <- run(ctx) }()
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
	shutdown, cancel := context.WithTimeout(context.Background(), a.shutdown)
	defer cancel()
	if e := a.server.Shutdown(shutdown); e != nil {
		_ = a.server.Close()
		err = errors.Join(err, e)
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
