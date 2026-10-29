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
// creating an initial row at (0, 0) if none exists yet.
func ensureAndLockCheckpointInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	ledgerEpoch uuid.UUID,
	now time.Time,
) (checkpointRow, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO usage_checkpoints (
			node_id, incarnation_id, highest_contiguous_seq, max_seen_seq,
			ledger_epoch, prefix_proof, updated_at
		) VALUES ($1, $2, 0, 0, $3, ''::bytea, $4)
		ON CONFLICT (node_id, incarnation_id) DO NOTHING
	`, nodeID, incarnationID, ledgerEpoch, now)
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: ensure checkpoint row: %w", err)
	}

	var (
		cp      checkpointRow
		hcsText string
		maxText string
	)
	err = tx.QueryRow(ctx, `
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
		return checkpointRow{}, fmt.Errorf("usage: lock checkpoint row: %w", err)
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

// recomputeStreamWatermarkWithConflictsInTx recomputes highest_contiguous_seq,
// max_seen_seq, missing_ranges, and prefix_proof from committed usage_events rows,
// stopping contiguous advancement before any unacknowledged conflicted sequence.
// Within one ledger_epoch, highest_contiguous_seq and max_seen_seq never regress.
func recomputeStreamWatermarkWithConflictsInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	forceFromZero bool,
	now time.Time,
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
		return checkpointRow{}, fmt.Errorf("usage: lock checkpoint for recompute: %w", err)
	}
	baseHCS, err := parseNumericUint64(hcsText)
	if err != nil {
		return checkpointRow{}, err
	}
	baseMax, err := parseNumericUint64(maxText)
	if err != nil {
		return checkpointRow{}, err
	}
	proof := cp.prefixProof
	if forceFromZero {
		baseHCS = 0
		baseMax = 0
		proof = []byte{}
	}

	// Find any conflicted sequences above baseHCS so contiguous ACK stops before them.
	conflictSeqs := make(map[uint64]bool)
	cRows, err := tx.Query(ctx, `
		SELECT DISTINCT event_seq::text
		FROM usage_event_conflicts
		WHERE node_id = $1 AND incarnation_id = $2 AND event_seq > $3::numeric
	`, nodeID, incarnationID, formatUint64(baseHCS))
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: query conflicts for watermark: %w", err)
	}
	for cRows.Next() {
		var sText string
		if err := cRows.Scan(&sText); err != nil {
			cRows.Close()
			return checkpointRow{}, err
		}
		seq, err := parseNumericUint64(sText)
		if err != nil {
			cRows.Close()
			return checkpointRow{}, err
		}
		conflictSeqs[seq] = true
	}
	cRows.Close()
	if err := cRows.Err(); err != nil {
		return checkpointRow{}, err
	}

	// Read all committed events above baseHCS ordered by event_seq ASC.
	rows, err := tx.Query(ctx, `
		SELECT event_seq::text, payload_hash
		FROM usage_events
		WHERE node_id = $1 AND incarnation_id = $2 AND event_seq > $3::numeric
		ORDER BY event_seq ASC
	`, nodeID, incarnationID, formatUint64(baseHCS))
	if err != nil {
		return checkpointRow{}, fmt.Errorf("usage: query committed events above hcs: %w", err)
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
			return checkpointRow{}, fmt.Errorf("usage: scan event seq and hash: %w", err)
		}
		seq, err := parseNumericUint64(sText)
		if err != nil {
			rows.Close()
			return checkpointRow{}, err
		}
		above = append(above, seqHash{seq: seq, hash: h})
		if !conflictSeqs[seq] {
			committedSeqs = append(committedSeqs, seq)
		}
		if seq > newMax {
			newMax = seq
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return checkpointRow{}, fmt.Errorf("usage: iterate committed events: %w", err)
	}

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

	// For missing ranges between newHCS and newMax, only non-conflicted committed events
	// count as present; a conflicted sequence above newHCS remains in missing_ranges/blocked_seq.
	missing := MissingRanges(newHCS, committedSeqs)
	// Also if newMax > highest non-conflicted seq, include any trailing gap up to newMax.
	if newMax > newHCS {
		var maxNonConflicted uint64
		if len(committedSeqs) > 0 {
			maxNonConflicted = committedSeqs[len(committedSeqs)-1]
		} else {
			maxNonConflicted = newHCS
		}
		if newMax > maxNonConflicted {
			missing = append(missing, SeqRange{Start: maxNonConflicted + 1, End: newMax})
		}
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
			RecoveryRecordId:     state.ActiveRecoveryID,
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
