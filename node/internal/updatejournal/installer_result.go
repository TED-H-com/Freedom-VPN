package updatejournal

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ReplaceInstallerResult atomically retains the legacy seal as retired evidence
// and enqueues its D-114 replacement. The original result ID remains present so
// recovery cannot mint the old receipt again. Acknowledged history is untouched.
// A preexisting replacement identity is a conflict, never an overwrite.
func (j *Journal) ReplaceInstallerResult(ctx context.Context, old, next OutboxEntry) (OutboxEntry, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return old, err
	}
	defer func() { _ = tx.Rollback() }()
	stored, err := scanOutbox(tx.QueryRowContext(ctx, outboxSelect+` WHERE result_id = ?`, old.ResultID))
	if err != nil {
		return old, err
	}
	if stored.Status != OutboxPending || stored.PayloadHash != old.PayloadHash {
		return old, ErrResultConflict
	}
	var present int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM result_outbox WHERE job_id = ?
		UNION ALL SELECT 1 FROM result_history WHERE job_id = ?`, next.JobID, next.JobID).Scan(&present)
	if err == nil {
		return old, ErrResultConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return old, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO result_outbox
		(result_id, job_id, attempt, generation, state, payload, payload_hash, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?)`, next.ResultID, next.JobID, next.Attempt,
		next.Generation.String(), next.State, next.Payload, next.PayloadHash, stored.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return old, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE result_outbox SET status = 'rejected',
		last_error = 'installer_job_id_rederived', retry_not_before = NULL WHERE result_id = ?`, old.ResultID); err != nil {
		return old, err
	}
	// Bounded audit link in the existing journal, without changing schema.
	if err := setMeta(ctx, tx, "installer_result_replacement/"+old.ResultID, next.ResultID); err != nil {
		return old, err
	}
	next, err = scanOutbox(tx.QueryRowContext(ctx, outboxSelect+` WHERE result_id = ?`, next.ResultID))
	if err != nil {
		return old, err
	}
	return next, tx.Commit()
}
