package usagestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// ApplyCheckpoint verifies a signed usage checkpoint and adopts it (§10.8,
// §17.3). A lower watermark is accepted only for a ledger epoch this node
// has not used before. The same epoch never moves the ACK backwards. Old
// epoch rows stay for audit. Events above the new watermark return to the
// outbox with their original payloads. Ranges this node cannot replay are
// recorded as replay_unavailable.
func (s *Store) ApplyCheckpoint(ctx context.Context, raw []byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.trust == nil {
		return ErrNoTrust
	}
	if now.IsZero() {
		now = s.now()
	}
	signed, _, err := envelope.Verify(raw, s.trust, envelope.PurposeRecovery, now)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadCheckpoint, err)
	}
	var cp api.UsageCheckpoint
	if err := json.Unmarshal(signed, &cp); err != nil {
		return fmt.Errorf("%w: %v", ErrBadCheckpoint, err)
	}
	if cp.NodeId != s.nodeID || cp.NodeIncarnationId != s.incID {
		return fmt.Errorf("%w: checkpoint is for another node incarnation", ErrBadCheckpoint)
	}
	if cp.LedgerEpoch == uuid.Nil || cp.IssuedAt.IsZero() {
		return fmt.Errorf("%w: checkpoint is missing ledger_epoch or issued_at", ErrBadCheckpoint)
	}
	highest := uint64(cp.HighestContiguousSeq)
	maxSeen := uint64(cp.MaxSeenSeq)
	if maxSeen < highest || highest >= 1<<63 || maxSeen >= 1<<63 {
		return fmt.Errorf("%w: checkpoint sequence is inconsistent", ErrBadCheckpoint)
	}
	for _, r := range cp.MissingRanges {
		if uint64(r.StartSeq) == 0 || uint64(r.EndSeq) < uint64(r.StartSeq) || uint64(r.EndSeq) >= 1<<63 {
			return fmt.Errorf("%w: missing range is not an inclusive span", ErrBadCheckpoint)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	h, err := s.headTx(ctx, tx)
	if err != nil {
		return err
	}
	var known int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_epochs WHERE ledger_epoch = ?`, cp.LedgerEpoch.String()).Scan(&known); err != nil {
		return err
	}
	newEpoch := cp.LedgerEpoch != h.LedgerEpoch
	if !newEpoch {
		if highest < h.AckedThrough {
			return ErrAckRegress
		}
	} else if h.LedgerEpoch != uuid.Nil && known > 0 {
		// A previously accepted epoch is not a new baseline, even when
		// the signature still verifies (§17.3 step 5).
		return ErrEpochReplay
	}
	if newEpoch && highest < h.AckedThrough {
		if _, err := tx.ExecContext(ctx, `UPDATE events SET state = 'outbox', acked_at = NULL, acked_epoch = NULL
			WHERE state = 'archived' AND seq > ?`, int64(highest)); err != nil {
			return err
		}
	}
	if highest > h.AckedThrough {
		if _, err := tx.ExecContext(ctx, `UPDATE events SET state = 'archived', acked_at = ?, acked_epoch = ?
			WHERE state = 'outbox' AND seq <= ?`,
			s.ts(now), cp.LedgerEpoch.String(), int64(highest)); err != nil {
			return err
		}
	}
	next := h.NextSeq
	if maxSeen+1 > next {
		next = maxSeen + 1
	}
	if _, err := tx.ExecContext(ctx, `UPDATE meta SET ledger_epoch = ?, acked_through = ?, blocked_seq = NULL, next_seq = ? WHERE id = 1`,
		cp.LedgerEpoch.String(), int64(highest), int64(next)); err != nil {
		return err
	}
	if h.LedgerEpoch != uuid.Nil && newEpoch {
		if _, err := tx.ExecContext(ctx, `UPDATE ledger_epochs SET superseded_at = ? WHERE ledger_epoch = ? AND superseded_at IS NULL`,
			s.ts(now), h.LedgerEpoch.String()); err != nil {
			return err
		}
	}
	var recovery any
	if id, err := cp.RecoveryRecordId.Get(); err == nil && id != uuid.Nil {
		recovery = id.String()
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_epochs (ledger_epoch, accepted_at, acked_through, recovery_record_id, checkpoint_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(ledger_epoch) DO UPDATE SET acked_through = excluded.acked_through,
			recovery_record_id = excluded.recovery_record_id,
			checkpoint_json = excluded.checkpoint_json,
			superseded_at = NULL`,
		cp.LedgerEpoch.String(), s.ts(now), int64(highest), recovery, raw); err != nil {
		return err
	}
	if err := s.recordUnavailable(ctx, tx, cp, highest); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) recordUnavailable(ctx context.Context, tx *sql.Tx, cp api.UsageCheckpoint, highest uint64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM replay_unavailable`); err != nil {
		return err
	}
	have := map[uint64]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT seq FROM events`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			rows.Close()
			return err
		}
		if seq > 0 {
			have[uint64(seq)] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var purged int64
	if err := tx.QueryRowContext(ctx, `SELECT purged_through FROM meta WHERE id = 1`).Scan(&purged); err != nil {
		return err
	}
	for _, mr := range cp.MissingRanges {
		start, end := uint64(mr.StartSeq), uint64(mr.EndSeq)
		if end < highest {
			// Inside the contiguous prefix the central ledger already
			// has these events. Nothing to replay.
			continue
		}
		if start <= highest {
			start = highest + 1
		}
		if start == 0 || end < start {
			continue
		}
		if end-start > 10_000_000 {
			return fmt.Errorf("%w: missing range is too large to account", ErrBadCheckpoint)
		}
		var hs, he uint64
		open := false
		flush := func() error {
			if !open {
				return nil
			}
			insert := func(from, to uint64, reason api.ReplayUnavailableRangeReason) error {
				_, err := tx.ExecContext(ctx, `INSERT INTO replay_unavailable (start_seq, end_seq, reason) VALUES (?, ?, ?)`,
					int64(from), int64(to), string(reason))
				return err
			}
			switch {
			case purged > 0 && int64(hs) <= purged && int64(he) > purged:
				if err := insert(hs, uint64(purged), api.ReplayUnavailableRangeReasonRetentionExpired); err != nil {
					return err
				}
				if err := insert(uint64(purged)+1, he, api.ReplayUnavailableRangeReasonLocalDataLost); err != nil {
					return err
				}
			case purged > 0 && int64(he) <= purged:
				if err := insert(hs, he, api.ReplayUnavailableRangeReasonRetentionExpired); err != nil {
					return err
				}
			default:
				if err := insert(hs, he, api.ReplayUnavailableRangeReasonLocalDataLost); err != nil {
					return err
				}
			}
			open = false
			return nil
		}
		for seq := start; seq <= end; seq++ {
			if have[seq] {
				if err := flush(); err != nil {
					return err
				}
				continue
			}
			if !open {
				hs, he = seq, seq
				open = true
				continue
			}
			he = seq
		}
		if err := flush(); err != nil {
			return err
		}
	}
	return nil
}

// LedgerEpochs returns the accepted epochs, oldest acceptance first. Rows
// are kept after a newer epoch supersedes them (§17.3 step 5).
func (s *Store) LedgerEpochs(ctx context.Context) ([]uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT ledger_epoch FROM ledger_epochs ORDER BY accepted_at, ledger_epoch`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// AckedThrough is the contiguous watermark of the current ledger epoch.
func (s *Store) AckedThrough(ctx context.Context) (uint64, error) {
	h, err := s.Head(ctx)
	if err != nil {
		return 0, err
	}
	return h.AckedThrough, nil
}

// CountEvents reports outbox and archived rows. Tests use it as the
// crash-consistency oracle.
func (s *Store) CountEvents(ctx context.Context) (outbox, archived int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(CASE WHEN state = 'outbox' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN state = 'archived' THEN 1 ELSE 0 END), 0)
		FROM events`).Scan(&outbox, &archived)
	return outbox, archived, err
}
