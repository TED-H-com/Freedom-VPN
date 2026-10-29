package db

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxOptions bounds retries, including the first attempt. Defaults: read committed,
// four attempts, full jitter with a 10ms base and a 250ms maximum delay.
type TxOptions struct {
	pgx.TxOptions
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// InTx rolls back on errors and panics. Only 40001 and 40P01 are retryable,
// including errors returned by Commit; uncertain network outcomes are not.
// fn may run more than once and must contain only transactional database work.
func InTx(ctx context.Context, pool *pgxpool.Pool, opts TxOptions, fn func(pgx.Tx) error) error {
	if opts.MaxAttempts == 0 {
		opts.MaxAttempts = 4
	}
	if opts.BaseDelay == 0 {
		opts.BaseDelay = 10 * time.Millisecond
	}
	if opts.MaxDelay == 0 {
		opts.MaxDelay = 250 * time.Millisecond
	}
	if opts.MaxAttempts < 1 || opts.MaxAttempts > 10 || opts.BaseDelay < 0 || opts.MaxDelay < opts.BaseDelay || opts.MaxDelay > time.Minute || fn == nil {
		return errors.New("invalid transaction options")
	}
	delay := opts.BaseDelay
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := txAttempt(ctx, pool, opts.TxOptions, fn)
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || (pgErr.Code != "40001" && pgErr.Code != "40P01") || attempt >= opts.MaxAttempts {
			return MapError(err)
		}
		timer := time.NewTimer(time.Duration(rand.Int64N(int64(delay) + 1)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, opts.MaxDelay)
	}
}

func txAttempt(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		// Rollback must still work when a request was canceled or fn panicked.
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
