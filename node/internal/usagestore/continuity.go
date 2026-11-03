package usagestore

import (
	"context"

	"github.com/google/uuid"
)

// SameCountingInterval proves that an admission intent for from still describes
// to. Epoch rows are append-only; rowid orders their insertion (also used by
// ClosedCheckpoint), not wall time or UUID order. Every intervening close/open
// must be a same-boot counter reset. A final sample, fresh baseline, missing
// origin or reboot breaks the proof. No ledger state is changed during adoption.
func (s *Store) SameCountingInterval(ctx context.Context, from, to uuid.UUID, boot string) (bool, error) {
	if from == uuid.Nil || to == uuid.Nil || boot == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var ok bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM epochs AS first, epochs AS last
		WHERE first.epoch_id = ? AND last.epoch_id = ?
		AND first.rowid <= last.rowid AND last.closed_at IS NULL
		AND NOT EXISTS (
			SELECT 1 FROM epochs AS e WHERE e.rowid BETWEEN first.rowid AND last.rowid
			AND (e.boot_id != ?
				OR (e.rowid > first.rowid AND e.open_reason != 'counter_discontinuity')
				OR (e.rowid < last.rowid AND
					(e.closed_at IS NULL OR COALESCE(e.close_reason, '') != 'counter_discontinuity')))
		))`, from.String(), to.String(), boot).Scan(&ok)
	return ok, err
}
