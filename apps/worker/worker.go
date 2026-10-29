// Package worker dispatches durable, fenced, at-least-once work (§14.5, §17.1).
package worker

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
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
