package commandtx

import (
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CompileFenceNamespace ("FVPN") and CompileFencePurpose name the D-167
// database-wide compile/capture fence. PostgreSQL's two-int advisory namespace
// is disjoint from the bigint hash locks used for member fanout and receipts.
const CompileFenceNamespace = 0x4656504e
const CompileFencePurpose = 167

// SharedCompileFence must be the transaction's first lock. Hold it through
// commit, including signing and all desired-state/target/removal writes.
func SharedCompileFence(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared($1::int,$2::int)`, CompileFenceNamespace, CompileFencePurpose)
	return err
}

// ExclusiveCompileFence drains in-flight compiles and prevents new ones until
// commit. Capture uses it both before recipient selection and at finalization.
func ExclusiveCompileFence(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1::int,$2::int)`, CompileFenceNamespace, CompileFencePurpose)
	return err
}

// RequireCompileFence checks the actual backend lock, not a context marker that
// could accidentally be carried into a different transaction or connection.
func RequireCompileFence(ctx context.Context, tx pgx.Tx) error {
	var held bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory'
  AND pid=pg_backend_pid() AND database=(SELECT oid FROM pg_database WHERE datname=current_database())
  AND classid=$1::oid AND objid=$2::oid AND objsubid=2 AND granted AND mode IN ('ShareLock','ExclusiveLock'))`, CompileFenceNamespace, CompileFencePurpose).Scan(&held)
	if err != nil {
		return err
	}
	if !held {
		return errors.New("desired state compile fence is required")
	}
	return nil
}

// InCompileTx takes the shared fence before the ordinary command transaction's
// claim and fleet locks. Nested callers must enter through this wrapper before
// taking their own locks; acquiring the same fence again is harmless.
func InCompileTx(ctx context.Context, pool *pgxpool.Pool, opts db.TxOptions, fn func(pgx.Tx) error) error {
	run := func(tx pgx.Tx) error {
		if err := SharedCompileFence(ctx, tx); err != nil {
			return err
		}
		return InTx(WithTransaction(ctx, tx), pool, opts, fn)
	}
	if tx, ok := Transaction(ctx); ok {
		return run(tx)
	}
	return db.InTx(ctx, pool, opts, run)
}
