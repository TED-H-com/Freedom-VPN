// Package commandtx joins node-domain services to an existing HTTP command
// transaction. Standalone callers retain the same durable service API.
package commandtx

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type key struct{}

func WithTransaction(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, key{}, tx)
}
func Transaction(ctx context.Context) (pgx.Tx, bool) {
	if tx, ok := ctx.Value(key{}).(pgx.Tx); ok {
		return tx, true
	}
	tx, e := httpapi.Transaction(ctx)
	return tx, e == nil
}
func InTx(ctx context.Context, pool *pgxpool.Pool, opts db.TxOptions, fn func(pgx.Tx) error) error {
	run := func(tx pgx.Tx) error {
		if e := checkClaim(ctx, tx); e != nil {
			return e
		}
		// §13.1: take the fleet lock before any node row, including revocation.
		if opts.AccessMode != pgx.ReadOnly {
			if _, e := tx.Exec(ctx, `SELECT id FROM fleet_config FOR UPDATE`); e != nil {
				return e
			}
		}
		return fn(tx)
	}
	if tx, ok := Transaction(ctx); ok {
		return run(tx)
	}
	return db.InTx(ctx, pool, opts, run)
}
func BeginIdempotent(ctx context.Context, tx pgx.Tx, scope db.IdempotencyScope, ttl time.Duration) (db.IdempotencyRecord, error) {
	if _, e := httpapi.Transaction(ctx); e == nil {
		return db.IdempotencyRecord{New: true}, nil
	}
	return db.BeginIdempotent(ctx, tx, scope, ttl)
}
func CompleteIdempotent(ctx context.Context, tx pgx.Tx, id uuid.UUID, result db.IdempotencyResult) error {
	if _, e := httpapi.Transaction(ctx); e == nil {
		return nil
	} // The HTTP adapter records SetResult once.
	return db.CompleteIdempotent(ctx, tx, id, result)
}

type fenceKey struct{}

// WithClaim fences local worker mutations after external read-back. Completion
// remains the dispatcher's responsibility, under the same unexpired claim.
func WithClaim(ctx context.Context, event db.ClaimedEvent) context.Context {
	return context.WithValue(ctx, fenceKey{}, event)
}
func checkClaim(ctx context.Context, tx pgx.Tx) error {
	event, ok := ctx.Value(fenceKey{}).(db.ClaimedEvent)
	if !ok {
		return nil
	}
	var id uuid.UUID
	e := tx.QueryRow(ctx, `SELECT id FROM outbox_events WHERE id=$1 AND lease_owner=$2 AND lease_token=$3 AND lease_until>clock_timestamp() AND delivered_at IS NULL AND dead_lettered_at IS NULL FOR UPDATE`, event.ID, event.Worker, event.LeaseToken).Scan(&id)
	if e == pgx.ErrNoRows {
		return db.ErrLeaseLost
	}
	return e
}
