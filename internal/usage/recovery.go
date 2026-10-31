package usage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// RecoveryRequest contains parameters to enter ledger recovery mode (§17.3).
type RecoveryRequest struct {
	NewLedgerEpoch uuid.UUID
	RestoredTo     time.Time
	AffectedSince  time.Time
	Reason         string
	Actor          string
}

// RecoverySignedPayload is the JCS-signed payload stored in usage_recovery_records (§17.3).
type RecoverySignedPayload struct {
	RecoveryID          uuid.UUID `json:"recovery_id"`
	PreviousLedgerEpoch uuid.UUID `json:"previous_ledger_epoch"`
	NewLedgerEpoch      uuid.UUID `json:"new_ledger_epoch"`
	RestoredTo          time.Time `json:"restored_to"`
	AffectedSince       time.Time `json:"affected_since"`
	Reason              string    `json:"reason"`
	Actor               string    `json:"actor"`
	StreamCount         int       `json:"stream_count"`
	CreatedAt           time.Time `json:"created_at"`
}

// RecoveryState describes a started or completed recovery operation (§17.3).
type RecoveryState struct {
	RecoveryID          uuid.UUID         `json:"recovery_id"`
	PreviousLedgerEpoch uuid.UUID         `json:"previous_ledger_epoch"`
	NewLedgerEpoch      uuid.UUID         `json:"new_ledger_epoch"`
	Mode                LedgerMode        `json:"mode"`
	Reason              string            `json:"reason"`
	Actor               string            `json:"actor"`
	SignedRecord        envelope.Envelope `json:"signed_record"`
	RecomputedStreams   int               `json:"recomputed_streams"`
	CreatedAt           time.Time         `json:"created_at"`
	CompletedAt         *time.Time        `json:"completed_at,omitempty"`
}

// FreezeIngest freezes central usage ingestion (§17.3 step 1).
// While frozen, IngestBatch returns 503 DEPENDENCY_UNAVAILABLE and never ACKs.
func (s *Service) FreezeIngest(ctx context.Context, actor, reason string) error {
	if strings.TrimSpace(actor) == "" {
		return apierr.Validation("actor is required", apierr.FieldError{Field: "/actor", Reason: "required"})
	}
	if strings.TrimSpace(reason) == "" {
		return apierr.Validation("reason is required", apierr.FieldError{Field: "/reason", Reason: "required"})
	}
	now := s.now()
	return db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		state, err := s.lockLedgerStateForUpdateInTx(ctx, tx)
		if err != nil {
			return fmt.Errorf("usage: lock ledger state for freeze: %w", err)
		}
		if state.Mode != LedgerModeFrozen {
			if err := holdDesiredStateAfterRestore(ctx, tx, now); err != nil {
				return fmt.Errorf("usage: hold desired state after restore: %w", err)
			}
		}
		tag, err := tx.Exec(ctx, `
			UPDATE usage_ledger_state
			SET mode = $2, mode_reason = $3, changed_by = $4, updated_at = $5
			WHERE id = $1
		`, ledgerStateSingletonID, string(LedgerModeFrozen), reason, actor, now)
		if err != nil {
			return fmt.Errorf("usage: freeze ingest: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("usage: usage_ledger_state singleton missing")
		}
		actID, actType := auditActor(actor)
		afterBytes, _ := json.Marshal(map[string]any{
			"usage_ledger_id": ledgerStateSingletonID,
			"mode":            string(LedgerModeFrozen),
			"reason":          reason,
			"actor":           actor,
		})
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         actID,
			ActorType:     actType,
			Action:        "usage_ledger.freeze",
			Resource:      string(api.ResourceTypeFleetConfig) + ":" + fleetConfigSingletonID,
			Result:        string(api.AuditEventResultSucceeded),
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         afterBytes,
		}); err != nil {
			return fmt.Errorf("usage: append freeze audit: %w", err)
		}
		return nil
	})
}

// BeginRecovery creates a new ledger_epoch with a signed recovery record,
// recomputes every stream's watermark and missing ranges from committed rows,
// marks affected rollups reconciliation_in_progress, and switches the ledger
// into reconciling mode so nodes can pull checkpoints and replay (§17.3 steps 2-6).
func (s *Service) BeginRecovery(ctx context.Context, req RecoveryRequest) (RecoveryState, error) {
	epochSigner := s.ledgerEpochSigner
	if epochSigner == nil {
		epochSigner = s.signer
	}
	if epochSigner == nil {
		return RecoveryState{}, fmt.Errorf("usage: ledger-epoch signer is not configured")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return RecoveryState{}, apierr.Validation("reason is required", apierr.FieldError{Field: "/reason", Reason: "required"})
	}
	if strings.TrimSpace(req.Actor) == "" {
		return RecoveryState{}, apierr.Validation("actor is required", apierr.FieldError{Field: "/actor", Reason: "required"})
	}

	now := s.now()
	restoredTo := req.RestoredTo.UTC()
	if restoredTo.IsZero() {
		restoredTo = now
	}
	affectedSince := req.AffectedSince.UTC()
	if affectedSince.IsZero() {
		affectedSince = MonthOf(restoredTo).Start
	}
	if affectedSince.After(restoredTo) {
		return RecoveryState{}, apierr.Validation("affected_since must not be after restored_to", apierr.FieldError{Field: "/affected_since", Reason: "interval_order"})
	}

	recoveryID, err := uuid.NewV7()
	if err != nil {
		return RecoveryState{}, fmt.Errorf("usage: generate recovery uuid: %w", err)
	}
	newEpoch := req.NewLedgerEpoch
	if newEpoch == uuid.Nil {
		newEpoch, err = uuid.NewV7()
		if err != nil {
			return RecoveryState{}, fmt.Errorf("usage: generate new ledger_epoch uuid: %w", err)
		}
	}

	var result RecoveryState
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		state, err := s.lockLedgerStateForUpdateInTx(ctx, tx)
		if err != nil {
			return fmt.Errorf("usage: lock ledger state: %w", err)
		}
		prevEpoch := state.LedgerEpoch
		if newEpoch == prevEpoch {
			return apierr.Validation("new_ledger_epoch must differ from current ledger_epoch", apierr.FieldError{Field: "/new_ledger_epoch", Reason: "conflict"})
		}
		// Direct BeginRecovery must install the same D-095 hold as FreezeIngest.
		// The normal freeze -> begin sequence retains the original freeze cutoff.
		if state.Mode != LedgerModeFrozen {
			if err := holdDesiredStateAfterRestore(ctx, tx, now); err != nil {
				return fmt.Errorf("usage: hold desired state after restore: %w", err)
			}
		}

		// Discover all node incarnation streams from checkpoints, events, and node_incarnations.
		type streamKey struct {
			nodeID        uuid.UUID
			incarnationID uuid.UUID
		}
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT node_id, incarnation_id FROM usage_checkpoints
			UNION
			SELECT DISTINCT node_id, incarnation_id FROM usage_events
			ORDER BY node_id, incarnation_id
		`)
		if err != nil {
			return fmt.Errorf("usage: list streams for recovery: %w", err)
		}
		var streams []streamKey
		for rows.Next() {
			var sk streamKey
			if err := rows.Scan(&sk.nodeID, &sk.incarnationID); err != nil {
				rows.Close()
				return fmt.Errorf("usage: scan stream key: %w", err)
			}
			streams = append(streams, sk)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("usage: iterate streams: %w", err)
		}

		payload := RecoverySignedPayload{
			RecoveryID:          recoveryID,
			PreviousLedgerEpoch: prevEpoch,
			NewLedgerEpoch:      newEpoch,
			RestoredTo:          restoredTo,
			AffectedSince:       affectedSince,
			Reason:              req.Reason,
			Actor:               req.Actor,
			StreamCount:         len(streams),
			CreatedAt:           now,
		}
		rawEnv, err := epochSigner.Sign(envelope.PurposeLedgerEpoch, payload)
		if err != nil {
			return fmt.Errorf("usage: sign recovery record: %w", err)
		}
		if len(rawEnv.Signatures) == 0 {
			return fmt.Errorf("usage: recovery signer returned no signatures")
		}
		envBytes, err := rawEnv.MarshalCanonical()
		if err != nil {
			return fmt.Errorf("usage: marshal canonical recovery envelope: %w", err)
		}
		recSum := sha256.Sum256(envBytes)

		_, err = tx.Exec(ctx, `
			INSERT INTO usage_recovery_records (
				id, ledger_epoch, previous_epoch, restored_to, affected_since,
				reason, actor, signed_envelope, signing_key_id, record_sha256,
				stream_count, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'reconciling', $12)
		`,
			recoveryID,
			newEpoch,
			prevEpoch,
			restoredTo,
			affectedSince,
			req.Reason,
			req.Actor,
			envBytes,
			rawEnv.Signatures[0].KeyID,
			recSum[:],
			len(streams),
			now,
		)
		if err != nil {
			return fmt.Errorf("usage: insert usage_recovery_records: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO usage_ledger_epochs (
				id, previous_epoch, recovery_record_id, created_at
			) VALUES ($1, $2, $3, $4)
		`, newEpoch, prevEpoch, recoveryID, now)
		if err != nil {
			return fmt.Errorf("usage: insert usage_ledger_epochs: %w", err)
		}

		// Reset each stream checkpoint into newEpoch and recompute watermarks & gaps
		// directly from committed usage_events rows in one statement per stream (§17.3 step 3, F-04).
		for _, sk := range streams {
			if _, err := resetStreamCheckpointForNewEpochInTx(ctx, tx, sk.nodeID, sk.incarnationID, newEpoch, now); err != nil {
				return fmt.Errorf("usage: recompute stream %s/%s: %w", sk.nodeID, sk.incarnationID, err)
			}
		}

		// Mark affected rollups with reconciliation_in_progress = TRUE (§17.3 step 4).
		genStr := formatUint64(state.ActiveProjectionGeneration)
		_, err = tx.Exec(ctx, `
			UPDATE usage_rollups
			SET reconciliation_in_progress = TRUE
			WHERE projection_generation = $1::numeric
			  AND period_end > $2
		`, genStr, affectedSince)
		if err != nil {
			return fmt.Errorf("usage: mark rollups reconciliation_in_progress: %w", err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE usage_ledger_state
			SET current_epoch = $2,
			    mode = $3,
			    mode_reason = $4,
			    changed_by = $5,
			    updated_at = $6
			WHERE id = $1
		`, ledgerStateSingletonID, newEpoch, string(LedgerModeReconciling), req.Reason, req.Actor, now)
		if err != nil {
			return fmt.Errorf("usage: update usage_ledger_state for recovery: %w", err)
		}

		actID, actType := auditActor(req.Actor)
		afterBytes, _ := json.Marshal(map[string]any{
			"usage_ledger_id":       ledgerStateSingletonID,
			"recovery_id":           recoveryID.String(),
			"previous_ledger_epoch": prevEpoch.String(),
			"new_ledger_epoch":      newEpoch.String(),
			"mode":                  string(LedgerModeReconciling),
			"reason":                req.Reason,
			"actor":                 req.Actor,
			"recomputed_streams":    len(streams),
		})
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         actID,
			ActorType:     actType,
			Action:        "usage_ledger.begin_recovery",
			Resource:      string(api.ResourceTypeFleetConfig) + ":" + fleetConfigSingletonID,
			Result:        string(api.AuditEventResultSucceeded),
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         afterBytes,
		}); err != nil {
			return fmt.Errorf("usage: append begin_recovery audit: %w", err)
		}

		result = RecoveryState{
			RecoveryID:          recoveryID,
			PreviousLedgerEpoch: prevEpoch,
			NewLedgerEpoch:      newEpoch,
			Mode:                LedgerModeReconciling,
			Reason:              req.Reason,
			Actor:               req.Actor,
			SignedRecord:        rawEnv,
			RecomputedStreams:   len(streams),
			CreatedAt:           now,
		}
		return nil
	})
	if err != nil {
		return RecoveryState{}, err
	}
	return result, nil
}

// CompleteRecovery exits reconciling mode after replay/gap recording finishes (§17.3).
func (s *Service) CompleteRecovery(ctx context.Context, actor string) error {
	if strings.TrimSpace(actor) == "" {
		return apierr.Validation("actor is required", apierr.FieldError{Field: "/actor", Reason: "required"})
	}
	now := s.now()
	return db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		state, err := s.lockLedgerStateForUpdateInTx(ctx, tx)
		if err != nil {
			return err
		}

		if state.ActiveRecoveryID != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE usage_recovery_records
				SET status = 'completed', completed_at = $2, completed_by = $3
				WHERE id = $1 AND status <> 'completed'
			`, *state.ActiveRecoveryID, now, actor); err != nil {
				return fmt.Errorf("usage: complete recovery record: %w", err)
			}
		}

		genStr := formatUint64(state.ActiveProjectionGeneration)
		if _, err := tx.Exec(ctx, `
			UPDATE usage_rollups
			SET reconciliation_in_progress = FALSE
			WHERE projection_generation = $1::numeric AND reconciliation_in_progress = TRUE
		`, genStr); err != nil {
			return fmt.Errorf("usage: clear reconciliation_in_progress on rollups: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE usage_ledger_state
			SET mode = $2, mode_reason = 'recovery_completed', changed_by = $3, updated_at = $4
			WHERE id = $1
		`, ledgerStateSingletonID, string(LedgerModeNormal), actor, now); err != nil {
			return fmt.Errorf("usage: set ledger mode normal: %w", err)
		}
		actID, actType := auditActor(actor)
		afterMap := map[string]any{
			"usage_ledger_id": ledgerStateSingletonID,
			"mode":            string(LedgerModeNormal),
			"ledger_epoch":    state.LedgerEpoch.String(),
			"actor":           actor,
		}
		if state.ActiveRecoveryID != nil {
			afterMap["recovery_id"] = state.ActiveRecoveryID.String()
		}
		afterBytes, _ := json.Marshal(afterMap)
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         actID,
			ActorType:     actType,
			Action:        "usage_ledger.complete_recovery",
			Resource:      string(api.ResourceTypeFleetConfig) + ":" + fleetConfigSingletonID,
			Result:        string(api.AuditEventResultSucceeded),
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         afterBytes,
		}); err != nil {
			return fmt.Errorf("usage: append complete_recovery audit: %w", err)
		}
		return nil
	})
}

// RecordReplayUnavailable records unrecoverable sequence ranges reported by a node
// during recovery or local retention expiry (§10.8 step 4, §17.3 step 9).
// It never invents bytes or overwrites the ledger with monthly totals; it inserts
// zero-byte tombstone rows so the contiguous watermark can advance past the permanently
// lost sequences, records the gap and affected months in usage_ledger_gaps, and marks
// affected member rollups with gap = TRUE.
func (s *Service) RecordReplayUnavailable(
	ctx context.Context,
	nodeIdentity NodeIdentity,
	ledgerEpoch uuid.UUID,
	ranges []api.ReplayUnavailableRange,
) error {
	if len(ranges) == 0 {
		return nil
	}
	state, err := s.GetLedgerState(ctx)
	if err != nil {
		return err
	}
	if state.Mode == LedgerModeFrozen {
		return apierr.New(apierr.DependencyUnavailable, "usage ledger is frozen for recovery")
	}
	if state.Mode != LedgerModeReconciling {
		return apierr.New(apierr.StateConflict, "replay_unavailable is only accepted while ledger mode is reconciling").
			WithDetail("ledger_mode", string(state.Mode))
	}
	if ledgerEpoch != state.LedgerEpoch {
		return apierr.New(apierr.LedgerEpochMismatch, "batch ledger_epoch does not match current central ledger_epoch").
			WithDetail("current_ledger_epoch", state.LedgerEpoch.String())
	}
	if err := s.checkNodeIncarnation(ctx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID); err != nil {
		return err
	}
	return s.recordReplayUnavailable(ctx, nodeIdentity, ledgerEpoch, ranges, s.now())
}

// maxReplayUnavailableSpan caps how many tombstone sequence rows a single range can synthesize.
const maxReplayUnavailableSpan = 10000

func (s *Service) recordReplayUnavailable(
	ctx context.Context,
	nodeIdentity NodeIdentity,
	ledgerEpoch uuid.UUID,
	ranges []api.ReplayUnavailableRange,
	receivedAt time.Time,
) error {
	type parsedRange struct {
		start  uint64
		end    uint64
		reason string
	}
	parsed := make([]parsedRange, 0, len(ranges))
	for i, r := range ranges {
		startSeq := uint64(r.StartSeq)
		endSeq := uint64(r.EndSeq)
		if startSeq == 0 {
			return apierr.Validation(
				"replay_unavailable.start_seq must be >= 1",
				apierr.FieldError{Field: fmt.Sprintf("/replay_unavailable/%d/start_seq", i), Reason: "positive_uint64"},
			)
		}
		if endSeq < startSeq {
			return apierr.Validation(
				"replay_unavailable.end_seq must be >= start_seq",
				apierr.FieldError{Field: fmt.Sprintf("/replay_unavailable/%d/end_seq", i), Reason: "range_order"},
			)
		}
		if endSeq-startSeq+1 > maxReplayUnavailableSpan {
			return apierr.Validation(
				"replay_unavailable range span exceeds limit",
				apierr.FieldError{Field: fmt.Sprintf("/replay_unavailable/%d/end_seq", i), Reason: "range_too_large"},
			)
		}
		if !r.Reason.Valid() {
			return apierr.Validation(
				"invalid replay_unavailable.reason",
				apierr.FieldError{Field: fmt.Sprintf("/replay_unavailable/%d/reason", i), Reason: "enum"},
			)
		}
		parsed = append(parsed, parsedRange{
			start:  startSeq,
			end:    endSeq,
			reason: string(r.Reason),
		})
	}

	return db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		preState, err := s.getLedgerStateInTx(ctx, tx)
		if err != nil {
			return err
		}
		if preState.Mode == LedgerModeFrozen {
			return apierr.New(apierr.DependencyUnavailable, "usage ledger is frozen for recovery")
		}
		if preState.Mode != LedgerModeReconciling {
			return apierr.New(apierr.StateConflict, "replay_unavailable is only accepted while ledger mode is reconciling").
				WithDetail("ledger_mode", string(preState.Mode))
		}
		if preState.LedgerEpoch != ledgerEpoch {
			return apierr.New(apierr.LedgerEpochMismatch, "ledger_epoch changed during request").
				WithDetail("current_ledger_epoch", preState.LedgerEpoch.String())
		}

		if _, err := ensureAndLockCheckpointInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, ledgerEpoch, receivedAt); err != nil {
			return err
		}

		state, err := s.lockLedgerStateForShareInTx(ctx, tx)
		if err != nil {
			return err
		}
		if state.Mode == LedgerModeFrozen {
			return apierr.New(apierr.DependencyUnavailable, "usage ledger is frozen for recovery")
		}
		if state.Mode != LedgerModeReconciling {
			return apierr.New(apierr.StateConflict, "replay_unavailable is only accepted while ledger mode is reconciling").
				WithDetail("ledger_mode", string(state.Mode))
		}
		if state.LedgerEpoch != ledgerEpoch {
			return apierr.New(apierr.LedgerEpochMismatch, "ledger_epoch changed during request").
				WithDetail("current_ledger_epoch", state.LedgerEpoch.String())
		}

		for _, r := range parsed {
			affectedFrom, affectedUntil, err := inferGapWindowInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, r.start, r.end, receivedAt)
			if err != nil {
				return err
			}
			affectedPeriods := MonthsOverlapping(affectedFrom, affectedUntil)

			// Find already committed sequences in [r.start, r.end] so we never overwrite real events.
			rows, err := tx.Query(ctx, `
				SELECT event_seq::text
				FROM usage_events
				WHERE node_id = $1
				  AND incarnation_id = $2
				  AND event_seq >= $3::numeric
				  AND event_seq <= $4::numeric
			`, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, formatUint64(r.start), formatUint64(r.end))
			if err != nil {
				return fmt.Errorf("usage: query existing events in replay_unavailable range: %w", err)
			}
			existing := make(map[uint64]bool)
			for rows.Next() {
				var sText string
				if err := rows.Scan(&sText); err != nil {
					rows.Close()
					return fmt.Errorf("usage: scan existing seq: %w", err)
				}
				seq, err := parseNumericUint64(sText)
				if err != nil {
					rows.Close()
					return err
				}
				existing[seq] = true
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return fmt.Errorf("usage: iterate existing seqs: %w", err)
			}

			var firstTombstoneID *uuid.UUID
			for seq := r.start; seq <= r.end; seq++ {
				if existing[seq] {
					continue
				}
				eventID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("usage: generate tombstone event_id: %w", err)
				}
				if firstTombstoneID == nil {
					idCopy := eventID
					firstTombstoneID = &idCopy
				}
				hashBytes, err := computeTombstoneHash(nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, seq, affectedFrom, affectedUntil)
				if err != nil {
					return fmt.Errorf("usage: compute tombstone hash: %w", err)
				}
				_, err = tx.Exec(ctx, `
					INSERT INTO usage_events (
						id, node_id, incarnation_id, event_seq, schema_version,
						payload_hash, record_count, received_at, quality_flags,
						kind, metering_version, quality,
						interval_start, interval_end, ledger_epoch
					) VALUES (
						$1, $2, $3, $4::numeric, 1,
						$5, 0, $6, ARRAY['gap','replay_unavailable'],
						'replay_unavailable', $7, 'gap',
						$8, $9, $10
					)
				`,
					eventID,
					nodeIdentity.NodeID,
					nodeIdentity.NodeIncarnationID,
					formatUint64(seq),
					hashBytes[:],
					receivedAt,
					MeteringVersionForwardAdmissionV1,
					affectedFrom,
					affectedUntil,
					ledgerEpoch,
				)
				if err != nil {
					return fmt.Errorf("usage: insert replay_unavailable tombstone seq %d: %w", seq, err)
				}
			}

			gapID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("usage: generate gap_id: %w", err)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO usage_ledger_gaps (
					id, node_id, incarnation_id, kind,
					start_seq, end_seq, reason, ledger_epoch,
					affected_from, affected_until, affected_periods,
					source_event_id, recorded_at
				) VALUES (
					$1, $2, $3, 'replay_unavailable',
					$4::numeric, $5::numeric, $6, $7,
					$8, $9, $10,
					$11, $12
				)
				ON CONFLICT (node_id, incarnation_id, kind, start_seq) DO NOTHING
			`,
				gapID,
				nodeIdentity.NodeID,
				nodeIdentity.NodeIncarnationID,
				formatUint64(r.start),
				formatUint64(r.end),
				r.reason,
				ledgerEpoch,
				affectedFrom,
				affectedUntil,
				affectedPeriods,
				firstTombstoneID,
				receivedAt,
			)
			if err != nil {
				return fmt.Errorf("usage: insert usage_ledger_gaps: %w", err)
			}

			if err := markReplayUnavailableOnRollupsInTx(ctx, tx, state.ActiveProjectionGeneration, nodeIdentity.NodeID, affectedFrom, affectedUntil, affectedPeriods, receivedAt); err != nil {
				return err
			}
		}

		if _, err := recomputeStreamWatermarkWithConflictsInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, false, receivedAt); err != nil {
			return err
		}
		return nil
	})
}

func inferGapWindowInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	startSeq uint64,
	endSeq uint64,
	fallbackNow time.Time,
) (time.Time, time.Time, error) {
	var prevEnd *time.Time
	err := tx.QueryRow(ctx, `
		SELECT MAX(interval_end)
		FROM usage_events
		WHERE node_id = $1
		  AND incarnation_id = $2
		  AND event_seq < $3::numeric
		  AND interval_end IS NOT NULL
	`, nodeID, incarnationID, formatUint64(startSeq)).Scan(&prevEnd)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("usage: query previous event interval_end: %w", err)
	}

	var nextStart *time.Time
	err = tx.QueryRow(ctx, `
		SELECT MIN(interval_start)
		FROM usage_events
		WHERE node_id = $1
		  AND incarnation_id = $2
		  AND event_seq > $3::numeric
		  AND interval_start IS NOT NULL
	`, nodeID, incarnationID, formatUint64(endSeq)).Scan(&nextStart)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("usage: query next event interval_start: %w", err)
	}

	var from, until time.Time
	switch {
	case prevEnd != nil && nextStart != nil && nextStart.After(*prevEnd):
		from = prevEnd.UTC()
		until = nextStart.UTC()
	case prevEnd != nil:
		from = prevEnd.UTC()
		until = from.Add(time.Minute)
		if fallbackNow.After(from) {
			until = fallbackNow.UTC()
		}
	case nextStart != nil:
		until = nextStart.UTC()
		from = until.Add(-time.Minute)
	default:
		until = fallbackNow.UTC()
		from = until.Add(-time.Minute)
	}
	if !until.After(from) {
		until = from.Add(time.Second)
	}
	return from, until, nil
}

func markReplayUnavailableOnRollupsInTx(
	ctx context.Context,
	tx pgx.Tx,
	generation uint64,
	nodeID uuid.UUID,
	affectedFrom time.Time,
	affectedUntil time.Time,
	affectedPeriods []string,
	now time.Time,
) error {
	genStr := formatUint64(generation)
	_ = affectedFrom
	_ = affectedUntil
	_ = now
	for _, m := range affectedPeriods {
		_, err := tx.Exec(ctx, `
			UPDATE usage_rollups r
			SET gap = TRUE
			WHERE r.projection_generation = $1::numeric
			  AND (r.period = $2 OR r.period LIKE $2 || '-%')
			  AND EXISTS (
			      SELECT 1 FROM usage_rollup_details d
			      WHERE d.projection_generation = r.projection_generation
			        AND d.member_id = r.member_id
			        AND d.node_id = $3
			  )
		`, genStr, m, nodeID)
		if err != nil {
			return fmt.Errorf("usage: mark replay_unavailable gap on rollups for month %s: %w", m, err)
		}
	}
	return nil
}

const systemActorUUID = "00000000-0000-0000-0000-000000000003"

func auditActor(actor string) (string, string) {
	trimmed := strings.TrimSpace(actor)
	if id, err := uuid.Parse(trimmed); err == nil {
		if id.String() == systemActorUUID || id == uuid.Nil {
			return systemActorUUID, string(api.AuditEventActorTypeSystem)
		}
		return id.String(), string(api.AuditEventActorTypeAdmin)
	}
	return systemActorUUID, string(api.AuditEventActorTypeSystem)
}
