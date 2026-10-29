package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// checkpointRow represents a locked row in usage_checkpoints.
type checkpointRow struct {
	id                uuid.UUID
	nodeID            uuid.UUID
	incarnationID     uuid.UUID
	ledgerEpoch       uuid.UUID
	highestContiguous uint64
	maxSeen           uint64
	missingRanges     []SeqRange
	prefixProof       []byte
	updatedAt         time.Time
}

// ensureAndLockCheckpointInTx returns the stream's usage_checkpoints row locked with FOR UPDATE,
// creating an initial row matching committed usage_events receipts if none exists yet (F-04).
func ensureAndLockCheckpointInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	ledgerEpoch uuid.UUID,
	now time.Time,
) (checkpointRow, error) {
	cp, err := selectCheckpointForUpdateInTx(ctx, tx, nodeID, incarnationID)
	if err == nil {
		return cp, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return checkpointRow{}, fmt.Errorf("usage: lock checkpoint row: %w", err)
	}

	// Serialize stream checkpoint creation on the parent node_incarnations row (F-13 lock order).
	var incExists int
	if err := tx.QueryRow(ctx, `
		SELECT 1 FROM node_incarnations
		WHERE node_id = $1 AND id = $2
		FOR NO KEY UPDATE
	`, nodeID, incarnationID).Scan(&incExists); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return checkpointRow{}, fmt.Errorf("usage: lock node incarnation for checkpoint init: %w", err)
	}

	// Re-check after acquiring the node_incarnations lock: if a concurrent transaction
	// already created usage_checkpoints, lock and return it without running BEFORE INSERT triggers.
	cp, err = selectCheckpointForUpdateInTx(ctx, tx, nodeID, incarnationID)
	if err == nil {
		return cp, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return checkpointRow{}, fmt.Errorf("usage: lock checkpoint row after incarnation lock: %w", err)
	}

	newHCS, newMax, missing, proof, err := computeWatermarkFromEventsInTx(ctx, tx, nodeID, incarnationID, 0, 0, nil)
	if err != nil {
		return checkpointRow{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO usage_checkpoints (
			node_id, incarnation_id, highest_contiguous_seq, max_seen_seq,
			ledger_epoch, prefix_proof, updated_at
		) VALUES ($1, $2, $3::numeric, $4::numeric, $5, $6, $7)
		ON CONFLICT (node_id, incarnation_id) DO NOTHING
	`, nodeID, incarnationID, formatUint64(newHCS), formatUint64(newMax), ledgerEpoch, proof, now)
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: ensure checkpoint row: %w", err)
	}

	cp, err = selectCheckpointForUpdateInTx(ctx, tx, nodeID, incarnationID)
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: lock checkpoint row after insert: %w", err)
	}
	cp.missingRanges = missing
	return cp, nil
}

func selectCheckpointForUpdateInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
) (checkpointRow, error) {
	var (
		cp      checkpointRow
		hcsText string
		maxText string
	)
	err := tx.QueryRow(ctx, `
		SELECT id, node_id, incarnation_id, ledger_epoch,
		       highest_contiguous_seq::text, max_seen_seq::text,
		       COALESCE(prefix_proof, ''::bytea), updated_at
		FROM usage_checkpoints
		WHERE node_id = $1 AND incarnation_id = $2
		FOR UPDATE
	`, nodeID, incarnationID).Scan(
		&cp.id,
		&cp.nodeID,
		&cp.incarnationID,
		&cp.ledgerEpoch,
		&hcsText,
		&maxText,
		&cp.prefixProof,
		&cp.updatedAt,
	)
	if err != nil {
		return checkpointRow{}, err
	}
	hcs, err := parseNumericUint64(hcsText)
	if err != nil {
		return checkpointRow{}, err
	}
	mx, err := parseNumericUint64(maxText)
	if err != nil {
		return checkpointRow{}, err
	}
	cp.highestContiguous = hcs
	cp.maxSeen = mx
	cp.updatedAt = cp.updatedAt.UTC()
	return cp, nil
}

// computeWatermarkFromEventsInTx computes the exact contiguous watermark, max_seen_seq,
// missing_ranges, and prefix_proof from committed usage_events rows above baseHCS (F-04).
func computeWatermarkFromEventsInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	baseHCS uint64,
	baseMax uint64,
	baseProof []byte,
) (uint64, uint64, []SeqRange, []byte, error) {
	rows, err := tx.Query(ctx, `
		SELECT event_seq::text, payload_hash
		FROM usage_events
		WHERE node_id = $1 AND incarnation_id = $2 AND event_seq > $3::numeric
		ORDER BY event_seq ASC
	`, nodeID, incarnationID, formatUint64(baseHCS))
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("usage: query committed events above hcs: %w", err)
	}
	type seqHash struct {
		seq  uint64
		hash []byte
	}
	var above []seqHash
	var committedSeqs []uint64
	newMax := baseMax
	for rows.Next() {
		var (
			sText string
			h     []byte
		)
		if err := rows.Scan(&sText, &h); err != nil {
			rows.Close()
			return 0, 0, nil, nil, fmt.Errorf("usage: scan event seq and hash: %w", err)
		}
		seq, err := parseNumericUint64(sText)
		if err != nil {
			rows.Close()
			return 0, 0, nil, nil, err
		}
		above = append(above, seqHash{seq: seq, hash: h})
		committedSeqs = append(committedSeqs, seq)
		if seq > newMax {
			newMax = seq
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, nil, nil, fmt.Errorf("usage: iterate committed events: %w", err)
	}

	proof := append([]byte(nil), baseProof...)
	newHCS := AdvanceContiguous(baseHCS, committedSeqs)
	for _, sh := range above {
		if sh.seq <= baseHCS {
			continue
		}
		if sh.seq > newHCS {
			break
		}
		proof = ExtendPrefixProof(proof, sh.seq, sh.hash)
	}
	missing := MissingRanges(newHCS, committedSeqs)
	return newHCS, newMax, missing, proof, nil
}

// recomputeStreamWatermarkWithConflictsInTx recomputes highest_contiguous_seq,
// max_seen_seq, missing_ranges, and prefix_proof from committed usage_events rows (F-04).
// Within one ledger_epoch, highest_contiguous_seq and max_seen_seq never regress.
func recomputeStreamWatermarkWithConflictsInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	forceFromZero bool,
	now time.Time,
) (checkpointRow, error) {
	cp, err := selectCheckpointForUpdateInTx(ctx, tx, nodeID, incarnationID)
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: lock checkpoint for recompute: %w", err)
	}
	baseHCS := cp.highestContiguous
	baseMax := cp.maxSeen
	baseProof := cp.prefixProof
	if forceFromZero {
		baseHCS = 0
		baseMax = 0
		baseProof = nil
	}

	newHCS, newMax, missing, proof, err := computeWatermarkFromEventsInTx(ctx, tx, nodeID, incarnationID, baseHCS, baseMax, baseProof)
	if err != nil {
		return checkpointRow{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE usage_checkpoints
		SET highest_contiguous_seq = $3::numeric,
		    max_seen_seq = $4::numeric,
		    prefix_proof = $5,
		    updated_at = $6
		WHERE node_id = $1 AND incarnation_id = $2
	`, nodeID, incarnationID, formatUint64(newHCS), formatUint64(newMax), proof, now)
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: update usage_checkpoints: %w", err)
	}

	cp.highestContiguous = newHCS
	cp.maxSeen = newMax
	cp.missingRanges = missing
	cp.prefixProof = proof
	cp.updatedAt = now.UTC()
	return cp, nil
}

// resetStreamCheckpointForNewEpochInTx recomputes the stream's watermark from 0
// over committed usage_events and atomically writes the new ledger_epoch and
// exact watermark into usage_checkpoints in a single statement (F-04).
func resetStreamCheckpointForNewEpochInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	newEpoch uuid.UUID,
	now time.Time,
) (checkpointRow, error) {
	newHCS, newMax, missing, proof, err := computeWatermarkFromEventsInTx(ctx, tx, nodeID, incarnationID, 0, 0, nil)
	if err != nil {
		return checkpointRow{}, err
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO usage_checkpoints (
			node_id, incarnation_id, highest_contiguous_seq, max_seen_seq,
			ledger_epoch, prefix_proof, updated_at
		) VALUES ($1, $2, $3::numeric, $4::numeric, $5, $6, $7)
		ON CONFLICT (node_id, incarnation_id) DO UPDATE SET
			ledger_epoch = EXCLUDED.ledger_epoch,
			highest_contiguous_seq = EXCLUDED.highest_contiguous_seq,
			max_seen_seq = EXCLUDED.max_seen_seq,
			prefix_proof = EXCLUDED.prefix_proof,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`, nodeID, incarnationID, formatUint64(newHCS), formatUint64(newMax), newEpoch, proof, now).Scan(&id)
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: reset stream checkpoint for new epoch: %w", err)
	}

	return checkpointRow{
		id:                id,
		nodeID:            nodeID,
		incarnationID:     incarnationID,
		ledgerEpoch:       newEpoch,
		highestContiguous: newHCS,
		maxSeen:           newMax,
		missingRanges:     missing,
		prefixProof:       proof,
		updatedAt:         now.UTC(),
	}, nil
}

// GetCheckpoint returns the signed central usage checkpoint for a node incarnation stream (§17.3).
// It derives highest_contiguous_seq, max_seen_seq, and missing_ranges directly from committed
// usage_events rows and signs the checkpoint with the recovery-purpose signer.
func (s *Service) GetCheckpoint(ctx context.Context, nodeIdentity NodeIdentity) (api.UsageCheckpointEnvelope, error) {
	if s.signer == nil {
		return api.UsageCheckpointEnvelope{}, errors.New("usage: recovery signer is not configured")
	}
	if err := s.checkNodeIncarnation(ctx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID); err != nil {
		return api.UsageCheckpointEnvelope{}, err
	}

	now := s.now()
	var signedCP api.UsageCheckpoint

	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		state, err := s.getLedgerStateInTx(ctx, tx)
		if err != nil {
			return err
		}

		if _, err := ensureAndLockCheckpointInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, state.LedgerEpoch, now); err != nil {
			return err
		}

		cp, err := recomputeStreamWatermarkWithConflictsInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, false, now)
		if err != nil {
			return err
		}

		signedCP = api.UsageCheckpoint{
			NodeId:               nodeIdentity.NodeID,
			NodeIncarnationId:    nodeIdentity.NodeIncarnationID,
			LedgerEpoch:          state.LedgerEpoch,
			RecoveryRecordId:     presence.FromPointer(state.ActiveRecoveryID),
			HighestContiguousSeq: decstr.U64(cp.highestContiguous),
			MaxSeenSeq:           decstr.U64(cp.maxSeen),
			MissingRanges:        toAPIRanges(cp.missingRanges),
			IssuedAt:             now,
		}
		return nil
	})
	if err != nil {
		return api.UsageCheckpointEnvelope{}, err
	}

	rawEnv, err := s.signer.Sign(envelope.PurposeRecovery, signedCP)
	if err != nil {
		return api.UsageCheckpointEnvelope{}, fmt.Errorf("usage: sign checkpoint: %w", err)
	}

	return api.UsageCheckpointEnvelope{
		Signed:     signedCP,
		Signatures: toAPISignatures(rawEnv.Signatures),
	}, nil
}
