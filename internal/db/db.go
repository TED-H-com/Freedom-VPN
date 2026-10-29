// Package db provides transaction primitives for the boundaries in §15.5.
// Callers must not perform network side effects inside retryable transactions.
package db

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenOptions bounds connection and server work. Zero values select defaults.
type OpenOptions struct {
	MaxConns               int32
	ConnectTimeout         time.Duration
	StatementTimeout       time.Duration
	LockTimeout            time.Duration
	IdleTransactionTimeout time.Duration
}

// Open validates configuration and pings before returning. Errors intentionally
// omit driver connection strings, which may contain credentials.
func Open(ctx context.Context, dsn string, opts OpenOptions) (*pgxpool.Pool, error) {
	if opts.MaxConns < 0 || opts.ConnectTimeout < 0 || opts.StatementTimeout < 0 || opts.LockTimeout < 0 || opts.IdleTransactionTimeout < 0 {
		return nil, errors.New("invalid database options")
	}
	if opts.MaxConns == 0 {
		opts.MaxConns = 10
	}
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = 5 * time.Second
	}
	if opts.StatementTimeout == 0 {
		opts.StatementTimeout = 30 * time.Second
	}
	if opts.LockTimeout == 0 {
		opts.LockTimeout = 5 * time.Second
	}
	if opts.IdleTransactionTimeout == 0 {
		opts.IdleTransactionTimeout = 30 * time.Second
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	cfg.MaxConns = opts.MaxConns
	cfg.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnLifetimeJitter = 5 * time.Minute
	for key, value := range map[string]time.Duration{"statement_timeout": opts.StatementTimeout, "lock_timeout": opts.LockTimeout, "idle_in_transaction_session_timeout": opts.IdleTransactionTimeout} {
		cfg.ConnConfig.RuntimeParams[key] = strconv.FormatInt(max(1, value.Milliseconds()), 10)
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "UTC"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("cannot create database pool")
	}
	pingCtx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
	defer cancel()
	if err = pool.Ping(pingCtx); err != nil {
		pool.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("database connection failed")
	}
	return pool, nil
}
