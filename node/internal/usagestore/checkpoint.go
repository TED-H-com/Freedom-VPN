package usagestore

import (
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// MeterCheckpoint is the existing node/updater.FileMeter JSON format. It is
// local barrier evidence, distinct from a central signed recovery checkpoint.
type MeterCheckpoint struct {
	ClosedMeterEpochID  string     `json:"closed_meter_epoch_id"`
	LastDurableEventSeq decstr.U64 `json:"last_durable_event_seq"`
	CentralAckedThrough decstr.U64 `json:"central_acked_through"`
	NewMeterEpochID     string     `json:"new_meter_epoch_id"`
}

// ClosedCheckpoint reads the epoch close and both sequence watermarks in one
// snapshot. It never generates an epoch, event, sequence, or central ACK.
func (s *Store) ClosedCheckpoint(ctx context.Context) (MeterCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MeterCheckpoint{}, err
	}
	defer func() { _ = tx.Rollback() }()
	h, err := s.headTx(ctx, tx)
	if err != nil {
		return MeterCheckpoint{}, err
	}
	if h.OpenEpoch != uuid.Nil || h.LastCloseReason != "final_sample" || h.NextSeq == 0 {
		return MeterCheckpoint{}, errors.New("usagestore: no completed meter barrier")
	}
	var id string
	if err := tx.QueryRowContext(ctx, `SELECT epoch_id FROM epochs WHERE closed_at IS NOT NULL AND close_reason = 'final_sample' ORDER BY rowid DESC LIMIT 1`).Scan(&id); err != nil {
		return MeterCheckpoint{}, err
	}
	if parsed, err := uuid.Parse(id); err != nil || parsed == uuid.Nil {
		return MeterCheckpoint{}, errors.New("usagestore: invalid closed epoch")
	}
	return MeterCheckpoint{ClosedMeterEpochID: id, LastDurableEventSeq: decstr.U64(h.NextSeq - 1), CentralAckedThrough: decstr.U64(h.AckedThrough)}, nil
}
