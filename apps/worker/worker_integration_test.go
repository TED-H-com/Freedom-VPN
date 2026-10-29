//go:build integration

package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

func enqueue(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	e := db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return db.EnqueueOutbox(t.Context(), tx, db.OutboxEvent{ID: id, AggregateID: uuid.New(), AggregateType: "node", AggregateVersion: "1", EventType: "test.readback", SchemaVersion: 1, Payload: []byte(`{}`)})
	})
	if e != nil {
		t.Fatal(e)
	}
	return id
}
func dispatcher(t *testing.T, pool *pgxpool.Pool, h Handler) *Dispatcher {
	t.Helper()
	d, e := New(pool, Registry{"test.readback": h}, Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Second}, prometheus.NewRegistry(), nil)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestAPI01_WorkerDisjointClaimsExpiryAndDeadLetters(t *testing.T) {
	pool := dbtest.New(t)
	id := enqueue(t, pool)
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	h := func(context.Context, db.ClaimedEvent) (string, error) {
		calls.Add(1)
		close(started)
		<-release
		return "observation:" + uuid.NewString(), nil
	}
	first, second := dispatcher(t, pool, h), dispatcher(t, pool, h)
	done := make(chan error, 1)
	go func() { _, e := first.Once(t.Context()); done <- e }()
	<-started
	if worked, e := second.Once(t.Context()); e != nil || worked {
		t.Fatal("duplicate lease", worked, e)
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate delivery")
	}
	var receipt *string
	if e := pool.QueryRow(t.Context(), `SELECT delivery_receipt FROM outbox_events WHERE id=$1`, id).Scan(&receipt); e != nil || receipt == nil {
		t.Fatal("read-back reference lost", e)
	}
	id = enqueue(t, pool)
	claims, e := db.ClaimOutbox(t.Context(), pool, "crashed-worker", 1, time.Minute)
	if e != nil || len(claims) != 1 {
		t.Fatal(e)
	}
	if _, e = pool.Exec(t.Context(), `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	recovered := dispatcher(t, pool, func(_ context.Context, event db.ClaimedEvent) (string, error) {
		if event.Attempt != 2 {
			t.Error("wrong attempt", event.Attempt)
		}
		return "observation:" + uuid.NewString(), nil
	})
	if worked, e := recovered.Once(t.Context()); e != nil || !worked {
		t.Fatal(e)
	}
	if e := db.CompleteOutbox(t.Context(), pool, claims[0]); !errors.Is(e, db.ErrLeaseLost) {
		t.Fatal("stale worker completed", e)
	}
	id = enqueue(t, pool)
	var failed atomic.Int32
	bad := dispatcher(t, pool, func(context.Context, db.ClaimedEvent) (string, error) {
		failed.Add(1)
		return "", errors.New("TEST ONLY secret provider error")
	})
	for i := 0; i < 3; i++ {
		if worked, e := bad.Once(t.Context()); e != nil || !worked {
			t.Fatal(e)
		}
		if i < 2 {
			if _, e := pool.Exec(t.Context(), `UPDATE outbox_events SET available_at=clock_timestamp() WHERE id=$1`, id); e != nil {
				t.Fatal(e)
			}
		}
	}
	if failed.Load() != 3 {
		t.Fatal("attempt budget", failed.Load())
	}
	if worked, e := bad.Once(t.Context()); e != nil || worked {
		t.Fatal("dead letter reclaimed", e)
	}
	var dead bool
	var code string
	e = pool.QueryRow(t.Context(), `SELECT dead_lettered_at IS NOT NULL AND delivered_at IS NULL,last_error_code FROM outbox_events WHERE id=$1`, id).Scan(&dead, &code)
	if e != nil || !dead || code != "HANDLER_FAILED" {
		t.Fatal(dead, code, e)
	}
	var audits int
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='outbox.dead_letter'`).Scan(&audits); e != nil || audits != 1 {
		t.Fatal("dead letter not visible", e)
	}
}
