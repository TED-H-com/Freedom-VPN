package usage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// BatchResult wraps api.UsageBatchResult with MaxSeenSeq as required by Deliverable 1.
type BatchResult struct {
	api.UsageBatchResult
	MaxSeenSeq decstr.U64 `json:"max_seen_seq"`
}

// IngestBatch validates and ingests a batch of usage events from an authenticated node (§10.6).
//
// Whole-batch preconditions (mTLS identity, node lifecycle, ledger freeze, ledger_epoch)
// are checked before any database mutation. Each event in the batch is then processed
// in its own atomic transaction so that a conflict or 422 rejection on one event never
// rolls back or deletes other valid events in the batch.
func (s *Service) IngestBatch(
	ctx context.Context,
	nodeIdentity NodeIdentity,
	batch api.UsageBatchRequest,
) (BatchResult, error) {
	if s.pool == nil {
		return BatchResult{}, apierr.New(apierr.DependencyUnavailable, "usage ledger database unavailable")
	}
	if nodeIdentity.NodeID == uuid.Nil || nodeIdentity.NodeIncarnationID == uuid.Nil {
		return BatchResult{}, apierr.New(apierr.Unauthenticated, "missing authenticated node mTLS identity")
	}

	// Whole-batch mTLS identity check (§10.6, §14.4).
	if batch.NodeId != nodeIdentity.NodeID || batch.NodeIncarnationId != nodeIdentity.NodeIncarnationID {
		return BatchResult{}, apierr.New(apierr.IncarnationMismatch, "batch node_id or node_incarnation_id does not match mTLS certificate identity")
	}
	for i, ev := range batch.Events {
		if ev.NodeId != nodeIdentity.NodeID || ev.NodeIncarnationId != nodeIdentity.NodeIncarnationID {
			return BatchResult{}, apierr.Newf(apierr.IncarnationMismatch, "event[%d] node_id or node_incarnation_id does not match mTLS certificate identity", i)
		}
	}

	if len(batch.Events) > MaxBatchEvents {
		return BatchResult{}, apierr.Newf(apierr.PayloadTooLarge, "batch exceeds maximum of %d events", MaxBatchEvents)
	}

	if err := s.checkNodeIncarnation(ctx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID); err != nil {
		return BatchResult{}, err
	}

	state, err := s.GetLedgerState(ctx)
	if err != nil {
		return BatchResult{}, err
	}
	if state.Mode == LedgerModeFrozen {
		return BatchResult{}, apierr.New(apierr.DependencyUnavailable, "usage ledger is frozen for recovery; do not advance ACK")
	}
	if batch.LedgerEpoch != nil {
		if *batch.LedgerEpoch != state.LedgerEpoch {
			return BatchResult{}, apierr.New(apierr.LedgerEpochMismatch, "batch ledger_epoch does not match current central ledger_epoch").
				WithDetail("current_ledger_epoch", state.LedgerEpoch.String())
		}
	} else if state.ActiveRecoveryID != nil {
		return BatchResult{}, apierr.New(apierr.LedgerEpochMismatch, "batch ledger_epoch is required after central recovery; fetch GET /agent/usage-checkpoint first").
			WithDetail("current_ledger_epoch", state.LedgerEpoch.String())
	}

	receivedAt := s.now()

	// Process replay_unavailable ranges if present (§10.8 step 4, §17.3 step 9).
	if batch.ReplayUnavailable != nil && len(*batch.ReplayUnavailable) > 0 {
		if err := s.recordReplayUnavailable(ctx, nodeIdentity, state.LedgerEpoch, *batch.ReplayUnavailable, receivedAt); err != nil {
			return BatchResult{}, err
		}
	}

	results := make([]api.UsageEventResult, len(batch.Events))
	for i, ev := range batch.Events {
		res, err := s.ingestSingleEvent(ctx, nodeIdentity, state, i, ev, receivedAt)
		if err != nil {
			return BatchResult{}, err
		}
		results[i] = res
	}

	// Read final committed watermark and blocked_seq for the stream.
	var (
		finalCP    checkpointRow
		blockedSeq *decstr.U64
	)
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := ensureAndLockCheckpointInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, state.LedgerEpoch, receivedAt); err != nil {
			return err
		}
		cp, err := recomputeStreamWatermarkWithConflictsInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, false, receivedAt)
		if err != nil {
			return err
		}
		finalCP = cp
		bSeq, err := computeBlockedSeqInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, cp.highestContiguous)
		if err != nil {
			return err
		}
		blockedSeq = bSeq
		return nil
	})
	if err != nil {
		return BatchResult{}, apierr.Wrap(err, apierr.DependencyUnavailable, "usage ledger database unavailable")
	}

	return BatchResult{
		UsageBatchResult: api.UsageBatchResult{
			NodeId:               nodeIdentity.NodeID,
			NodeIncarnationId:    nodeIdentity.NodeIncarnationID,
			LedgerEpoch:          state.LedgerEpoch,
			HighestContiguousSeq: decstr.U64(finalCP.highestContiguous),
			BlockedSeq:           blockedSeq,
			Results:              results,
			ReceivedAt:           receivedAt,
		},
		MaxSeenSeq: decstr.U64(finalCP.maxSeen),
	}, nil
}

func (s *Service) ingestSingleEvent(
	ctx context.Context,
	nodeIdentity NodeIdentity,
	initialState LedgerState,
	eventIdx int,
	ev api.UsageEvent,
	receivedAt time.Time,
) (api.UsageEventResult, error) {
	seq := uint64(ev.EventSeq)
	fieldErrs, computedHash, computedHex, canonPayload := validateEventFields(eventIdx, ev)

	var out api.UsageEventResult
	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		state, err := s.getLedgerStateInTx(ctx, tx)
		if err != nil {
			return err
		}
		if state.Mode == LedgerModeFrozen {
			return apierr.New(apierr.DependencyUnavailable, "usage ledger is frozen for recovery")
		}
		if state.LedgerEpoch != initialState.LedgerEpoch {
			return apierr.New(apierr.LedgerEpochMismatch, "ledger_epoch changed during batch ingestion").
				WithDetail("current_ledger_epoch", state.LedgerEpoch.String())
		}

		if _, err := ensureAndLockCheckpointInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, state.LedgerEpoch, receivedAt); err != nil {
			return err
		}

		// 1. If seq > 0, check whether (node_id, incarnation_id, event_seq) already exists in usage_events.
		if seq > 0 {
			var (
				existingID   uuid.UUID
				existingHash []byte
			)
			err := tx.QueryRow(ctx, `
				SELECT id, payload_hash
				FROM usage_events
				WHERE node_id = $1 AND incarnation_id = $2 AND event_seq = $3::numeric
			`, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, formatUint64(seq)).Scan(&existingID, &existingHash)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("usage: lookup existing event: %w", err)
			}
			if err == nil {
				// Event key already exists in usage_events!
				if len(fieldErrs) == 0 && bytes.Equal(existingHash, computedHash[:]) {
					// Idempotent duplicate resend with identical canonical payload hash (§10.6).
					if _, err := recomputeStreamWatermarkWithConflictsInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, false, receivedAt); err != nil {
						return err
					}
					out = api.UsageEventResult{
						EventSeq: ev.EventSeq,
						Status:   api.UsageEventResultStatusDuplicate,
					}
					return nil
				}

				// Same (node_id, incarnation_id, event_seq) with a different hash -> USAGE_EVENT_CONFLICT.
				conflictHash := computedHash
				conflictPayload := canonPayload
				if len(conflictPayload) == 0 {
					raw, _ := json.Marshal(ev)
					conflictPayload = raw
					conflictHash = sha256.Sum256(raw)
				}
				if bytes.Equal(existingHash, conflictHash[:]) {
					// Defensive fallback if raw happened to hash identically despite fieldErrs.
					tagged := append([]byte("conflict:"), conflictPayload...)
					conflictHash = sha256.Sum256(tagged)
				}
				if err := recordConflictInTx(
					ctx, tx,
					nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID,
					seq, existingID, existingHash,
					conflictHash[:], conflictPayload,
					state.LedgerEpoch, receivedAt,
				); err != nil {
					return err
				}
				out = makeErrorResult(ev.EventSeq, api.UsageEventResultStatusConflict, apierr.UsageEventConflict, "Same event_seq with a different payload_sha256.", nil)
				return nil
			}
		}

		// 2. If structural/hash validation failed, record rejection and return rejected (422 semantics for this event).
		if len(fieldErrs) > 0 {
			if err := recordRejectionInTx(
				ctx, tx,
				nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID,
				seq, canonPayload, ev,
				string(apierr.ValidationFailed), fieldErrs,
				state.LedgerEpoch, receivedAt,
			); err != nil {
				return err
			}
			out = makeErrorResult(ev.EventSeq, api.UsageEventResultStatusRejected, apierr.ValidationFailed, "Event failed validation.", fieldErrs)
			return nil
		}

		// 3. Validate historical device_bindings for every record (§10.2, §15.2).
		bindingErrs, err := validateEventBindingsInTx(ctx, tx, eventIdx, ev)
		if err != nil {
			return err
		}
		if len(bindingErrs) > 0 {
			if err := recordRejectionInTx(
				ctx, tx,
				nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID,
				seq, canonPayload, ev,
				string(apierr.ValidationFailed), bindingErrs,
				state.LedgerEpoch, receivedAt,
			); err != nil {
				return err
			}
			out = makeErrorResult(ev.EventSeq, api.UsageEventResultStatusRejected, apierr.ValidationFailed, "Event record failed historical device_bindings validation.", bindingErrs)
			return nil
		}

		// 4. Insert usage_events, usage_event_payloads, usage_samples, projection_receipts,
		// usage_rollups, usage_rollup_details, and usage_checkpoints in one atomic transaction (§10.6).
		eventID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("usage: generate event_id: %w", err)
		}
		var qualityFlags []string
		if ev.Quality != api.UsageQualityComplete {
			qualityFlags = []string{string(ev.Quality)}
		} else {
			qualityFlags = []string{}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO usage_events (
				id, node_id, incarnation_id, event_seq, schema_version,
				payload_hash, record_count, received_at, quality_flags,
				kind, meter_epoch_id, metering_version, quality,
				interval_start, interval_end, ledger_epoch
			) VALUES (
				$1, $2, $3, $4::numeric, $5,
				$6, $7, $8, $9,
				'event', $10, $11, $12,
				$13, $14, $15
			)
		`,
			eventID,
			nodeIdentity.NodeID,
			nodeIdentity.NodeIncarnationID,
			formatUint64(seq),
			int(ev.SchemaVersion),
			computedHash[:],
			len(ev.Records),
			receivedAt,
			qualityFlags,
			ev.MeterEpochId,
			string(ev.MeteringVersion),
			string(ev.Quality),
			ev.IntervalStart.UTC(),
			ev.IntervalEnd.UTC(),
			state.LedgerEpoch,
		)
		if err != nil {
			return fmt.Errorf("usage: insert usage_events: %w", err)
		}

		payloadID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("usage: generate payload_id: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO usage_event_payloads (
				id, event_id, payload, created_at
			) VALUES ($1, $2, $3, $4)
		`, payloadID, eventID, canonPayload, receivedAt)
		if err != nil {
			return fmt.Errorf("usage: insert usage_event_payloads: %w", err)
		}

		elapsedNs := uint64(ev.IntervalEnd.UTC().Sub(ev.IntervalStart.UTC()).Nanoseconds())
		clockQuality := "trusted"
		if ev.Gap != nil && ev.Gap.Reason == api.UsageGapReasonClockAnomaly {
			clockQuality = "anomaly"
		}

		samplesToProject := make([]sampleForProjection, 0, len(ev.Records))
		for rIdx, rec := range ev.Records {
			sampleID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("usage: generate sample_id: %w", err)
			}
			upBytes := uint64(rec.UploadBytes)
			downBytes := uint64(rec.DownloadBytes)
			bindRev := uint64(rec.BindingRevision)

			_, err = tx.Exec(ctx, `
				INSERT INTO usage_samples (
					id, event_id, record_index, member_id, device_id,
					binding_revision, meter_epoch_id, interval_start, interval_end,
					upload_bytes, download_bytes, metering_version,
					monotonic_elapsed_ns, clock_quality, quality_flags
				) VALUES (
					$1, $2, $3, $4, $5,
					$6::numeric, $7, $8, $9,
					$10::numeric, $11::numeric, $12,
					$13::numeric, $14, $15
				)
			`,
				sampleID,
				eventID,
				rIdx,
				rec.MemberId,
				rec.DeviceId,
				decstr.U64(bindRev).String(),
				ev.MeterEpochId,
				ev.IntervalStart.UTC(),
				ev.IntervalEnd.UTC(),
				decstr.U64(upBytes).String(),
				decstr.U64(downBytes).String(),
				string(ev.MeteringVersion),
				decstr.U64(elapsedNs).String(),
				clockQuality,
				qualityFlags,
			)
			if err != nil {
				return fmt.Errorf("usage: insert usage_samples[%d]: %w", rIdx, err)
			}

			samplesToProject = append(samplesToProject, sampleForProjection{
				SampleID:        sampleID,
				EventID:         eventID,
				NodeID:          nodeIdentity.NodeID,
				IncarnationID:   nodeIdentity.NodeIncarnationID,
				MemberID:        rec.MemberId,
				DeviceID:        rec.DeviceId,
				IntervalStart:   ev.IntervalStart.UTC(),
				IntervalEnd:     ev.IntervalEnd.UTC(),
				UploadBytes:     upBytes,
				DownloadBytes:   downBytes,
				MeteringVersion: string(ev.MeteringVersion),
				Quality:         string(ev.Quality),
				ReceivedAt:      receivedAt,
			})
		}

		reconciling := state.Mode == LedgerModeReconciling
		if len(samplesToProject) > 0 {
			if err := projectSamplesInTx(ctx, tx, state.ActiveProjectionGeneration, reconciling, samplesToProject); err != nil {
				return err
			}
		}

		if ev.Quality == api.UsageQualityGap {
			gapID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("usage: generate gap_id: %w", err)
			}
			gapReason := "unknown"
			if ev.Gap != nil && ev.Gap.Reason != "" {
				gapReason = string(ev.Gap.Reason)
			}
			affectedPeriods := MonthsOverlapping(ev.IntervalStart.UTC(), ev.IntervalEnd.UTC())
			_, err = tx.Exec(ctx, `
				INSERT INTO usage_ledger_gaps (
					id, node_id, incarnation_id, kind,
					start_seq, end_seq, reason, ledger_epoch,
					affected_from, affected_until, affected_periods,
					source_event_id, recorded_at
				) VALUES (
					$1, $2, $3, 'node_reported',
					$4::numeric, $4::numeric, $5, $6,
					$7, $8, $9,
					$10, $11
				)
				ON CONFLICT (node_id, incarnation_id, kind, start_seq) DO NOTHING
			`,
				gapID,
				nodeIdentity.NodeID,
				nodeIdentity.NodeIncarnationID,
				formatUint64(seq),
				gapReason,
				state.LedgerEpoch,
				ev.IntervalStart.UTC(),
				ev.IntervalEnd.UTC(),
				affectedPeriods,
				eventID,
				receivedAt,
			)
			if err != nil {
				return fmt.Errorf("usage: insert node_reported gap: %w", err)
			}

			if err := markGapOnRollupsInTx(
				ctx, tx,
				state.ActiveProjectionGeneration,
				reconciling,
				eventID,
				nodeIdentity.NodeID,
				nodeIdentity.NodeIncarnationID,
				string(ev.MeteringVersion),
				ev.IntervalStart.UTC(),
				ev.IntervalEnd.UTC(),
				receivedAt,
			); err != nil {
				return err
			}
		}

		if _, err := recomputeStreamWatermarkWithConflictsInTx(ctx, tx, nodeIdentity.NodeID, nodeIdentity.NodeIncarnationID, false, receivedAt); err != nil {
			return err
		}

		// Touch unused computedHex to satisfy compiler if needed.
		_ = computedHex

		out = api.UsageEventResult{
			EventSeq: ev.EventSeq,
			Status:   api.UsageEventResultStatusAccepted,
		}
		return nil
	})
	if err != nil {
		var ae *apierr.Error
		if errors.As(err, &ae) {
			return api.UsageEventResult{}, ae
		}
		return api.UsageEventResult{}, apierr.Wrap(err, apierr.DependencyUnavailable, "usage ledger database unavailable")
	}
	return out, nil
}

func validateEventFields(eventIdx int, ev api.UsageEvent) ([]api.FieldError, [32]byte, string, []byte) {
	var errs []api.FieldError
	prefix := fmt.Sprintf("/events/%d", eventIdx)

	if uint64(ev.EventSeq) == 0 {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/event_seq",
			Reason: "positive_uint64",
		})
	}
	if !ev.SchemaVersion.Valid() {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/schema_version",
			Reason: "enum",
		})
	}
	if ev.MeterEpochId == uuid.Nil {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/meter_epoch_id",
			Reason: "required",
		})
	}
	if string(ev.MeteringVersion) != MeteringVersionForwardAdmissionV1 {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/metering_version",
			Reason: "enum",
		})
	}
	if !ev.Quality.Valid() {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/quality",
			Reason: "enum",
		})
	}
	if ev.Quality == api.UsageQualityGap && ev.Gap == nil {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/gap",
			Reason: "required",
		})
	}
	if ev.Quality != api.UsageQualityGap && ev.Gap != nil {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/gap",
			Reason: "forbidden",
		})
	}
	if ev.Gap != nil && !ev.Gap.Reason.Valid() {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/gap/reason",
			Reason: "enum",
		})
	}
	if ev.IntervalStart.IsZero() {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/interval_start",
			Reason: "required",
		})
	}
	if ev.IntervalEnd.IsZero() {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/interval_end",
			Reason: "required",
		})
	}
	if !ev.IntervalStart.IsZero() && !ev.IntervalEnd.IsZero() && !ev.IntervalEnd.After(ev.IntervalStart) {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/interval_end",
			Reason: "interval_order",
		})
	}

	seenDevice := make(map[uuid.UUID]bool, len(ev.Records))
	for rIdx, rec := range ev.Records {
		rPrefix := fmt.Sprintf("%s/records/%d", prefix, rIdx)
		if rec.MemberId == uuid.Nil {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/member_id",
				Reason: "required",
			})
		}
		if rec.DeviceId == uuid.Nil {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/device_id",
				Reason: "required",
			})
		} else if seenDevice[rec.DeviceId] {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/device_id",
				Reason: "duplicate_device",
			})
		} else {
			seenDevice[rec.DeviceId] = true
		}
		if uint64(rec.BindingRevision) == 0 {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/binding_revision",
				Reason: "positive_uint64",
			})
		}
		p, err := ParseMonth(rec.PeriodId)
		if err != nil {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/period_id",
				Reason: "format",
			})
		} else if !ev.IntervalStart.IsZero() && !ev.IntervalEnd.IsZero() && ev.IntervalEnd.After(ev.IntervalStart) {
			// Ensure period_id overlaps [interval_start, interval_end).
			if !p.Start.Before(ev.IntervalEnd.UTC()) || !p.End.After(ev.IntervalStart.UTC()) {
				errs = append(errs, api.FieldError{
					Field:  rPrefix + "/period_id",
					Reason: "period_mismatch",
				})
			}
		}
	}

	hashBytes, hashHex, canonPayload, err := ComputeEventHash(ev)
	if err != nil {
		errs = append(errs, api.FieldError{
			Field:  prefix,
			Reason: "canonical_json",
		})
		return errs, [32]byte{}, "", nil
	}
	if !strings.EqualFold(ev.PayloadSha256, hashHex) || len(ev.PayloadSha256) != 64 {
		errs = append(errs, api.FieldError{
			Field:  prefix + "/payload_sha256",
			Reason: "hash_mismatch",
		})
	}
	return errs, hashBytes, hashHex, canonPayload
}

func validateEventBindingsInTx(
	ctx context.Context,
	tx pgx.Tx,
	eventIdx int,
	ev api.UsageEvent,
) ([]api.FieldError, error) {
	var errs []api.FieldError
	prefix := fmt.Sprintf("/events/%d", eventIdx)
	iStart := ev.IntervalStart.UTC()
	iEnd := ev.IntervalEnd.UTC()

	for rIdx, rec := range ev.Records {
		rPrefix := fmt.Sprintf("%s/records/%d", prefix, rIdx)
		var (
			boundMemberID uuid.UUID
			validFrom     time.Time
			validUntil    time.Time
		)
		err := tx.QueryRow(ctx, `
			SELECT member_id, valid_from, valid_until
			FROM device_bindings
			WHERE device_id = $1 AND binding_revision = $2::numeric
			FOR SHARE
		`, rec.DeviceId, rec.BindingRevision.String()).Scan(&boundMemberID, &validFrom, &validUntil)
		if errors.Is(err, pgx.ErrNoRows) {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/binding_revision",
				Reason: "binding_not_found",
			})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("usage: query device_bindings: %w", err)
		}
		if boundMemberID != rec.MemberId {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/member_id",
				Reason: "binding_member_mismatch",
			})
		}
		if iStart.Before(validFrom.UTC()) || iEnd.After(validUntil.UTC()) {
			errs = append(errs, api.FieldError{
				Field:  rPrefix + "/binding_revision",
				Reason: "interval_outside_binding_window",
			})
		}
	}
	return errs, nil
}

func recordConflictInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	seq uint64,
	existingEventID uuid.UUID,
	existingHash []byte,
	conflictingHash []byte,
	conflictingPayload []byte,
	ledgerEpoch uuid.UUID,
	now time.Time,
) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("usage: generate conflict id: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO usage_event_conflicts (
			id, node_id, incarnation_id, event_seq,
			existing_event_id, existing_payload_hash,
			conflicting_payload_hash, conflicting_payload,
			ledger_epoch, first_seen_at, last_seen_at, occurrences
		) VALUES (
			$1, $2, $3, $4::numeric,
			$5, $6,
			$7, $8,
			$9, $10, $10, 1
		)
		ON CONFLICT (node_id, incarnation_id, event_seq, conflicting_payload_hash)
		DO UPDATE SET
			last_seen_at = EXCLUDED.last_seen_at,
			occurrences = usage_event_conflicts.occurrences + 1
	`,
		id,
		nodeID,
		incarnationID,
		formatUint64(seq),
		existingEventID,
		existingHash,
		conflictingHash,
		conflictingPayload,
		ledgerEpoch,
		now,
	)
	if err != nil {
		return fmt.Errorf("usage: record usage_event_conflicts: %w", err)
	}
	return nil
}

func recordRejectionInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	seq uint64,
	canonPayload []byte,
	ev api.UsageEvent,
	errorCode string,
	fields []api.FieldError,
	ledgerEpoch uuid.UUID,
	now time.Time,
) error {
	payload := canonPayload
	if len(payload) == 0 {
		raw, err := json.Marshal(ev)
		if err != nil {
			payload = []byte("{}")
		} else {
			payload = raw
		}
	}
	contentSum := sha256.Sum256(payload)
	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("usage: marshal rejection fields: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("usage: generate rejection id: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO usage_event_rejections (
			id, node_id, incarnation_id, event_seq,
			content_sha256, error_code, fields, payload,
			ledger_epoch, first_seen_at, last_seen_at, occurrences
		) VALUES (
			$1, $2, $3, $4::numeric,
			$5, $6, $7, $8,
			$9, $10, $10, 1
		)
		ON CONFLICT (node_id, incarnation_id, event_seq, content_sha256)
		DO UPDATE SET
			error_code = EXCLUDED.error_code,
			fields = EXCLUDED.fields,
			last_seen_at = EXCLUDED.last_seen_at,
			occurrences = usage_event_rejections.occurrences + 1
	`,
		id,
		nodeID,
		incarnationID,
		formatUint64(seq),
		contentSum[:],
		errorCode,
		fieldsJSON,
		payload,
		ledgerEpoch,
		now,
	)
	if err != nil {
		return fmt.Errorf("usage: record usage_event_rejections: %w", err)
	}
	return nil
}

// computeBlockedSeqInTx finds the lowest sequence above highest_contiguous_seq
// that is in conflict or rejected (and not resolved by an accepted non-conflicted event)
// and therefore blocks contiguous ACK progress (§10.6).
func computeBlockedSeqInTx(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	highestContiguous uint64,
) (*decstr.U64, error) {
	var minSeqText *string
	err := tx.QueryRow(ctx, `
		SELECT MIN(seq)::text FROM (
			SELECT c.event_seq AS seq
			FROM usage_event_conflicts c
			WHERE c.node_id = $1 AND c.incarnation_id = $2 AND c.event_seq > $3::numeric
			UNION
			SELECT r.event_seq AS seq
			FROM usage_event_rejections r
			WHERE r.node_id = $1 AND r.incarnation_id = $2 AND r.event_seq > $3::numeric
			  AND NOT EXISTS (
			      SELECT 1 FROM usage_events e
			      WHERE e.node_id = r.node_id
			        AND e.incarnation_id = r.incarnation_id
			        AND e.event_seq = r.event_seq
			  )
		) blocked
	`, nodeID, incarnationID, formatUint64(highestContiguous)).Scan(&minSeqText)
	if err != nil || minSeqText == nil {
		return nil, err
	}
	seq, err := parseNumericUint64(*minSeqText)
	if err != nil {
		return nil, err
	}
	u := decstr.U64(seq)
	return &u, nil
}

func makeErrorResult(
	seq decstr.U64,
	status api.UsageEventResultStatus,
	code apierr.Code,
	msg string,
	fields []api.FieldError,
) api.UsageEventResult {
	if fields == nil {
		fields = []api.FieldError{}
	}
	return api.UsageEventResult{
		EventSeq: seq,
		Status:   status,
		Error: &struct {
			Code    api.ErrorCode    `json:"code"`
			Fields  []api.FieldError `json:"fields"`
			Message string           `json:"message"`
		}{
			Code:    api.ErrorCode(code),
			Fields:  fields,
			Message: msg,
		},
	}
}
