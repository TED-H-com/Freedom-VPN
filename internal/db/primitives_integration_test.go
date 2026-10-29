//go:build integration

package db_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func Test15_5_SerializationRetry(t *testing.T) {
	p := dbtest.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var id uuid.UUID
	if err := p.QueryRow(ctx, `INSERT INTO members DEFAULT VALUES RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var calls, ready atomic.Int32
	barrier := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			first := true
			errs <- db.InTx(ctx, p, db.TxOptions{TxOptions: pgx.TxOptions{IsoLevel: pgx.Serializable}}, func(tx pgx.Tx) error {
				calls.Add(1)
				var version string
				if err := tx.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1`, id).Scan(&version); err != nil {
					return err
				}
				if first {
					first = false
					if ready.Add(1) == 2 {
						close(barrier)
					}
					select {
					case <-barrier:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				_, err := tx.Exec(ctx, `UPDATE members SET resource_version=resource_version+1 WHERE id=$1`, id)
				return err
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("expected exactly one serialization retry, calls=%d", calls.Load())
	}
	var version string
	if err := p.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1`, id).Scan(&version); err != nil || version != "3" {
		t.Fatalf("version=%s err=%v", version, err)
	}
}

func Test15_5_DeadlockRetry(t *testing.T) {
	p := dbtest.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	exec(t, p, `INSERT INTO members(id) VALUES($1),($2)`, ids[0], ids[1])
	var ready, calls atomic.Int32
	barrier := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Go(func() {
			first := true
			errs <- db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
				calls.Add(1)
				if _, err := tx.Exec(ctx, `UPDATE members SET resource_version=resource_version+1 WHERE id=$1`, ids[i]); err != nil {
					return err
				}
				if first {
					first = false
					if ready.Add(1) == 2 {
						close(barrier)
					}
					select {
					case <-barrier:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				_, err := tx.Exec(ctx, `UPDATE members SET resource_version=resource_version+1 WHERE id=$1`, ids[1-i])
				return err
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("expected one deadlock retry, calls=%d", calls.Load())
	}
}

func Test15_5_RetryBoundsUniqueAndPanic(t *testing.T) {
	p := dbtest.New(t)
	ctx := context.Background()
	exec(t, p, `INSERT INTO plans(code) VALUES('test')`)
	calls := 0
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		calls++
		_, err := tx.Exec(ctx, `INSERT INTO plans(code) VALUES('test')`)
		return err
	})
	if calls != 1 || !errors.Is(err, db.ErrUniqueViolation) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	var constraint *db.ConstraintError
	if !errors.As(err, &constraint) || constraint.Constraint != "plans_code_key" {
		t.Fatalf("constraint=%v", constraint)
	}
	calls = 0
	err = db.InTx(ctx, p, db.TxOptions{MaxAttempts: 3}, func(tx pgx.Tx) error {
		calls++
		_, err := tx.Exec(ctx, `DO $$ BEGIN RAISE EXCEPTION 'forced serialization failure' USING ERRCODE='40001'; END $$`)
		return err
	})
	sqlError(t, err, "40001", "")
	if calls != 3 {
		t.Fatalf("unbounded retry: %d", calls)
	}
	func() {
		defer func() {
			if recover() != "test panic" {
				t.Error("panic not propagated")
			}
		}()
		_ = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `INSERT INTO plans(code) VALUES('panic')`); err != nil {
				t.Fatal(err)
			}
			panic("test panic")
		})
	}()
	var n int
	if err := p.QueryRow(ctx, `SELECT count(*) FROM plans WHERE code='panic'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("panic committed n=%d err=%v", n, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := db.InTx(canceled, p, db.TxOptions{}, func(pgx.Tx) error { t.Error("called after cancellation"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func Test14_5_VersionAndAtomicAuditOutbox(t *testing.T) {
	p := dbtest.New(t)
	ctx := context.Background()
	id := uuid.New()
	exec(t, p, `INSERT INTO members(id) VALUES($1)`, id)
	sentinel := errors.New("abort local change")
	change := func(tx pgx.Tx) error {
		version, err := db.UpdateWithVersion(ctx, tx, "members", id, "1", map[string]any{"display_name": "Test"})
		if err != nil {
			return err
		}
		if version != "2" {
			t.Errorf("version=%s", version)
		}
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{Actor: "test", ActorType: "admin", Action: "member.update", Resource: id.String(), Result: "accepted", SchemaVersion: 1, Before: []byte(`{}`), After: []byte(`{"display_name":"Test"}`)}); err != nil {
			return err
		}
		return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: id, AggregateType: "member", AggregateVersion: version, EventType: "member.changed", SchemaVersion: 1, Payload: []byte(`{}`)})
	}
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		if err := change(tx); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var n int
	if err := p.QueryRow(ctx, `SELECT (SELECT count(*) FROM audit_events)+(SELECT count(*) FROM outbox_events)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("non-atomic rows=%d err=%v", n, err)
	}
	if err := db.InTx(ctx, p, db.TxOptions{}, change); err != nil {
		t.Fatal(err)
	}
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		_, err := db.UpdateWithVersion(ctx, tx, "members", id, "1", map[string]any{"display_name": "stale"})
		return err
	})
	if !errors.Is(err, db.ErrVersionConflict) {
		t.Fatal(err)
	}
	for _, bad := range []string{`[]`, `null`, `"text"`, `{not-json}`} {
		err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
			return db.AppendAudit(ctx, tx, db.AuditEvent{SchemaVersion: 1, Before: []byte(bad), After: []byte(`{}`)})
		})
		if err == nil {
			t.Errorf("accepted audit %s", bad)
		}
	}
}

func Test14_5_IdempotencyAndConcurrentFirstUse(t *testing.T) {
	p := dbtest.New(t)
	ctx := context.Background()
	scope := db.IdempotencyScope{Principal: "test", Method: "POST", Route: "/api/v1/test", KeyHash: bytes.Repeat([]byte{1}, 32), RequestHash: bytes.Repeat([]byte{2}, 32)}
	var wins atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	ids := make(chan uuid.UUID, 2)
	for range 2 {
		wg.Go(func() {
			errs <- db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
				record, err := db.BeginIdempotent(ctx, tx, scope, 0)
				if err != nil {
					return err
				}
				ids <- record.ID
				if record.New {
					wins.Add(1)
					id := uuid.New()
					if _, err := tx.Exec(ctx, `INSERT INTO members(id) VALUES($1)`, id); err != nil {
						return err
					}
					return db.CompleteIdempotent(ctx, tx, record.ID, db.IdempotencyResult{ResourceType: "member", ResourceID: &id, ResponseStatus: 201})
				}
				if record.ResourceID == nil || record.ResponseStatus == nil || *record.ResponseStatus != 201 {
					return errors.New("replay missing reference/status")
				}
				return nil
			})
		})
	}
	wg.Wait()
	close(errs)
	close(ids)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if wins.Load() != 1 {
		t.Fatalf("winners=%d", wins.Load())
	}
	var recordID uuid.UUID
	for id := range ids {
		if recordID != uuid.Nil && recordID != id {
			t.Fatal("different replay identity")
		}
		recordID = id
	}
	different := scope
	different.RequestHash = bytes.Repeat([]byte{3}, 32)
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { _, err := db.BeginIdempotent(ctx, tx, different, 0); return err })
	if !errors.Is(err, db.ErrIdempotencyConflict) {
		t.Fatalf("different request: %v", err)
	}
	// Expired scope is reusable with a fresh fencing ID, even for a different request.
	exec(t, p, `UPDATE idempotency_requests SET created_at=now()-interval '8 days',expiry=now()-interval '1 day'`)
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		record, err := db.BeginIdempotent(ctx, tx, different, 0)
		if err != nil {
			return err
		}
		if !record.New || record.ID == recordID || record.ResourceID != nil {
			return errors.New("expired key was not reset")
		}
		id := uuid.New()
		return db.CompleteIdempotent(ctx, tx, record.ID, db.IdempotencyResult{ResourceType: "member", ResourceID: &id, ResponseStatus: 201})
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		id := uuid.New()
		return db.CompleteIdempotent(ctx, tx, recordID, db.IdempotencyResult{ResourceID: &id, ResponseStatus: 201})
	})
	if !errors.Is(err, db.ErrIdempotencyConflict) {
		t.Fatalf("stale completion: %v", err)
	}
	// Failed domain work rolls back the key reservation as well.
	scope.KeyHash = bytes.Repeat([]byte{4}, 32)
	abort := errors.New("abort")
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := db.BeginIdempotent(ctx, tx, scope, 0); err != nil {
			return err
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatal(err)
	}
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		r, err := db.BeginIdempotent(ctx, tx, scope, 0)
		if err != nil {
			return err
		}
		if !r.New {
			return errors.New("key survived rollback")
		}
		id := uuid.New()
		return db.CompleteIdempotent(ctx, tx, r.ID, db.IdempotencyResult{ResourceID: &id, ResponseStatus: 201})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func Test14_5_IdempotentInProgressReference(t *testing.T) {
	p := dbtest.New(t)
	ctx := context.Background()
	scope := db.IdempotencyScope{Principal: "test", Method: "POST", Route: "/test", KeyHash: bytes.Repeat([]byte{5}, 32), RequestHash: bytes.Repeat([]byte{6}, 32)}
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		r, err := db.BeginIdempotent(ctx, tx, scope, 0)
		if err != nil {
			return err
		}
		var op uuid.UUID
		if err := tx.QueryRow(ctx, `INSERT INTO operations(kind,principal,resource_type) VALUES('test','test','node') RETURNING id`).Scan(&op); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE idempotency_requests SET operation_id=$1,response_status=202 WHERE id=$2`, op, r.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		r, err := db.BeginIdempotent(ctx, tx, scope, 0)
		if err == nil && (r.New || r.State != "in_progress" || r.OperationID == nil || r.ResponseStatus == nil || *r.ResponseStatus != 202) {
			return errors.New("in-progress replay lost operation")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func Test15_4_OutboxConcurrentClaimsAndLeaseExpiry(t *testing.T) {
	p := dbtest.New(t)
	ctx := context.Background()
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		for range 10 {
			if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.New(), AggregateType: "test", AggregateVersion: "9007199254740993", EventType: "test", SchemaVersion: 1, Payload: []byte(`{}`)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := make(chan []db.ClaimedEvent, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, worker := range []string{"one", "two"} {
		wg.Go(func() { events, err := db.ClaimOutbox(ctx, p, worker, 5, time.Minute); claims <- events; errs <- err })
	}
	wg.Wait()
	close(claims)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[uuid.UUID]bool{}
	var old db.ClaimedEvent
	for events := range claims {
		for _, e := range events {
			if seen[e.ID] {
				t.Fatal("duplicate worker ownership")
			}
			seen[e.ID] = true
			old = e
			if e.AggregateVersion != "9007199254740993" {
				t.Fatal("numeric lost precision")
			}
		}
	}
	if len(seen) != 10 {
		t.Fatalf("claimed=%d", len(seen))
	}
	none, err := db.ClaimOutbox(ctx, p, "third", 10, time.Minute)
	if err != nil || len(none) != 0 {
		t.Fatalf("claimed active leases=%d err=%v", len(none), err)
	}
	exec(t, p, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, old.ID)
	if err := db.CompleteOutbox(ctx, p, old); !errors.Is(err, db.ErrLeaseLost) {
		t.Fatalf("expired lease completed: %v", err)
	}
	events, err := db.ClaimOutbox(ctx, p, old.Worker, 1, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatalf("reclaim: %v %+v", err, events)
	}
	current := events[0]
	if current.LeaseToken == old.LeaseToken || current.Attempt != 2 {
		t.Fatal("lease was not fenced")
	}
	if err := db.CompleteOutbox(ctx, p, old); !errors.Is(err, db.ErrLeaseLost) {
		t.Fatal(err)
	}
	if err := db.RetryOutbox(ctx, p, old, 0); !errors.Is(err, db.ErrLeaseLost) {
		t.Fatal(err)
	}
	if err := db.RetryOutbox(ctx, p, current, time.Hour); err != nil {
		t.Fatal(err)
	}
	none, err = db.ClaimOutbox(ctx, p, "third", 10, time.Minute)
	if err != nil || len(none) != 0 {
		t.Fatalf("ignored backoff: %v", err)
	}
	exec(t, p, `UPDATE outbox_events SET available_at=clock_timestamp()-interval '1 second' WHERE id=$1`, current.ID)
	events, err = db.ClaimOutbox(ctx, p, "third", 1, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatalf("claim after backoff: %v", err)
	}
	if err := db.CompleteOutbox(ctx, p, events[0]); err != nil {
		t.Fatal(err)
	}
	exec(t, p, `UPDATE outbox_events SET available_at=clock_timestamp()-interval '1 second' WHERE id=$1`, current.ID)
	none, err = db.ClaimOutbox(ctx, p, "third", 10, time.Minute)
	if err != nil || len(none) != 0 {
		t.Fatalf("redelivered completed event: %v", err)
	}
}
