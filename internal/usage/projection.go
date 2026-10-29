package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
)

// sampleForProjection holds the committed sample fields needed by the projector.
type sampleForProjection struct {
	SampleID        uuid.UUID
	EventID         uuid.UUID
	NodeID          uuid.UUID
	IncarnationID   uuid.UUID
	MemberID        uuid.UUID
	DeviceID        uuid.UUID
	IntervalStart   time.Time
	IntervalEnd     time.Time
	UploadBytes     uint64
	DownloadBytes   uint64
	MeteringVersion string
	Quality         string // complete, boundary_estimated, gap
	ReceivedAt      time.Time
}

// projectSamplesInTx projects a slice of samples into usage_rollups and
// usage_rollup_details for the given generation inside tx (§10.9, §15.4).
// Each (sample_id, projection_name, generation, period) receipt is inserted with
// ON CONFLICT DO NOTHING and the rollup increment only happens when the receipt
// is newly inserted, guaranteeing idempotency across reruns.
func projectSamplesInTx(
	ctx context.Context,
	tx pgx.Tx,
	generation uint64,
	reconciling bool,
	samples []sampleForProjection,
) error {
	genStr := formatUint64(generation)
	for _, smp := range samples {
		for _, gran := range []Granularity{GranularityMonth, GranularityDay} {
			segments, err := Split(smp.IntervalStart, smp.IntervalEnd, smp.UploadBytes, smp.DownloadBytes, gran)
			if err != nil {
				return fmt.Errorf("usage: split sample %s (%s): %w", smp.SampleID, gran, err)
			}

			for _, seg := range segments {
				receiptID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("usage: generate receipt id: %w", err)
				}
				tag, err := tx.Exec(ctx, `
					INSERT INTO projection_receipts (
						id, sample_id, projection_name, generation, period, applied_at
					) VALUES ($1, $2, $3, $4::numeric, $5, $6)
					ON CONFLICT (sample_id, projection_name, generation, period) DO NOTHING
				`,
					receiptID,
					smp.SampleID,
					ProjectionNameMemberUsage,
					genStr,
					seg.ID,
					smp.ReceivedAt,
				)
				if err != nil {
					return fmt.Errorf("usage: insert projection receipt: %w", err)
				}
				if tag.RowsAffected() == 0 {
					continue
				}

				isEstimated := seg.Estimated || smp.Quality == "boundary_estimated"
				isGap := smp.Quality == "gap"
				upStr := formatUint64(seg.Upload)
				downStr := formatUint64(seg.Download)

				rollupID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("usage: generate rollup id: %w", err)
				}
				_, err = tx.Exec(ctx, `
					INSERT INTO usage_rollups (
						id, member_id, period, granularity, projection_generation,
						metering_version, upload_bytes, download_bytes,
						applied_event_checkpoint, last_observed_at, aggregate_as_of,
						period_start, period_end, estimated, gap, reconciliation_in_progress
					) VALUES (
						$1, $2, $3, $4, $5::numeric,
						$6, $7::numeric, $8::numeric,
						$9, $10, $11,
						$12, $13, $14, $15, $16
					)
					ON CONFLICT (member_id, period, granularity, projection_generation)
					DO UPDATE SET
						upload_bytes = usage_rollups.upload_bytes + EXCLUDED.upload_bytes,
						download_bytes = usage_rollups.download_bytes + EXCLUDED.download_bytes,
						applied_event_checkpoint = EXCLUDED.applied_event_checkpoint,
						last_observed_at = GREATEST(usage_rollups.last_observed_at, EXCLUDED.last_observed_at),
						aggregate_as_of = GREATEST(usage_rollups.aggregate_as_of, EXCLUDED.aggregate_as_of),
						estimated = usage_rollups.estimated OR EXCLUDED.estimated,
						gap = usage_rollups.gap OR EXCLUDED.gap,
						reconciliation_in_progress = usage_rollups.reconciliation_in_progress OR EXCLUDED.reconciliation_in_progress
				`,
					rollupID,
					smp.MemberID,
					seg.ID,
					string(gran),
					genStr,
					smp.MeteringVersion,
					upStr,
					downStr,
					smp.EventID,
					smp.IntervalEnd.UTC(),
					smp.ReceivedAt.UTC(),
					seg.Start,
					seg.End,
					isEstimated,
					isGap,
					reconciling,
				)
				if err != nil {
					return fmt.Errorf("usage: upsert usage_rollups: %w", err)
				}

				detailID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("usage: generate rollup detail id: %w", err)
				}
				_, err = tx.Exec(ctx, `
					INSERT INTO usage_rollup_details (
						id, projection_generation, granularity, period,
						member_id, node_id, incarnation_id, device_id,
						metering_version, upload_bytes, download_bytes,
						last_observed_at, period_start, period_end,
						estimated, gap, updated_at
					) VALUES (
						$1, $2::numeric, $3, $4,
						$5, $6, $7, $8,
						$9, $10::numeric, $11::numeric,
						$12, $13, $14,
						$15, $16, $17
					)
					ON CONFLICT (projection_generation, granularity, period, member_id, node_id, incarnation_id, device_id)
					DO UPDATE SET
						upload_bytes = usage_rollup_details.upload_bytes + EXCLUDED.upload_bytes,
						download_bytes = usage_rollup_details.download_bytes + EXCLUDED.download_bytes,
						last_observed_at = GREATEST(usage_rollup_details.last_observed_at, EXCLUDED.last_observed_at),
						estimated = usage_rollup_details.estimated OR EXCLUDED.estimated,
						gap = usage_rollup_details.gap OR EXCLUDED.gap,
						updated_at = EXCLUDED.updated_at
				`,
					detailID,
					genStr,
					string(gran),
					seg.ID,
					smp.MemberID,
					smp.NodeID,
					smp.IncarnationID,
					smp.DeviceID,
					smp.MeteringVersion,
					upStr,
					downStr,
					smp.IntervalEnd.UTC(),
					seg.Start,
					seg.End,
					isEstimated,
					isGap,
					smp.ReceivedAt.UTC(),
				)
				if err != nil {
					return fmt.Errorf("usage: upsert usage_rollup_details: %w", err)
				}
			}
		}
	}
	return nil
}

// markGapOnRollupsInTx marks rollups overlapping [intervalStart, intervalEnd) for
// members who had active bindings on the node (or existing rollups in the period)
// with gap = TRUE when a node reports a gap event (§10.7).
func markGapOnRollupsInTx(
	ctx context.Context,
	tx pgx.Tx,
	generation uint64,
	reconciling bool,
	eventID uuid.UUID,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	meteringVersion string,
	intervalStart time.Time,
	intervalEnd time.Time,
	receivedAt time.Time,
) error {
	genStr := formatUint64(generation)

	// Find members who have usage on this node or had an active device binding overlapping the gap window.
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT member_id FROM (
			SELECT member_id
			FROM usage_rollup_details
			WHERE projection_generation = $1::numeric AND node_id = $2
			UNION
			SELECT member_id
			FROM device_bindings
			WHERE valid_from < $4 AND valid_until > $3
		) m
	`, genStr, nodeID, intervalStart.UTC(), intervalEnd.UTC())
	if err != nil {
		return fmt.Errorf("usage: query members for gap event: %w", err)
	}
	var memberIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("usage: scan member for gap event: %w", err)
		}
		memberIDs = append(memberIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("usage: iterate members for gap event: %w", err)
	}

	for _, gran := range []Granularity{GranularityMonth, GranularityDay} {
		segments, err := Split(intervalStart, intervalEnd, 0, 0, gran)
		if err != nil {
			return fmt.Errorf("usage: split gap interval (%s): %w", gran, err)
		}
		for _, seg := range segments {
			for _, memberID := range memberIDs {
				rollupID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("usage: generate gap rollup id: %w", err)
				}
				_, err = tx.Exec(ctx, `
					INSERT INTO usage_rollups (
						id, member_id, period, granularity, projection_generation,
						metering_version, upload_bytes, download_bytes,
						applied_event_checkpoint, last_observed_at, aggregate_as_of,
						period_start, period_end, estimated, gap, reconciliation_in_progress
					) VALUES (
						$1, $2, $3, $4, $5::numeric,
						$6, 0, 0,
						$7, $8, $9,
						$10, $11, FALSE, TRUE, $12
					)
					ON CONFLICT (member_id, period, granularity, projection_generation)
					DO UPDATE SET
						gap = TRUE,
						last_observed_at = GREATEST(usage_rollups.last_observed_at, EXCLUDED.last_observed_at),
						aggregate_as_of = GREATEST(usage_rollups.aggregate_as_of, EXCLUDED.aggregate_as_of),
						reconciliation_in_progress = usage_rollups.reconciliation_in_progress OR EXCLUDED.reconciliation_in_progress
				`,
					rollupID,
					memberID,
					seg.ID,
					string(gran),
					genStr,
					meteringVersion,
					eventID,
					intervalEnd.UTC(),
					receivedAt.UTC(),
					seg.Start,
					seg.End,
					reconciling,
				)
				if err != nil {
					return fmt.Errorf("usage: mark gap on usage_rollups: %w", err)
				}
			}
		}
	}
	return nil
}

// RebuildProjectionResult summarizes a completed projection generation rebuild (§10.9, §15.4).
type RebuildProjectionResult struct {
	PreviousGeneration uint64    `json:"previous_generation"`
	NewGeneration      uint64    `json:"new_generation"`
	ProjectedSamples   int       `json:"projected_samples"`
	VerifiedAt         time.Time `json:"verified_at"`
}

// RebuildProjection writes a fresh projection_generation from committed usage_samples
// and usage_events, verifies that the new generation's byte totals conserve the raw
// sample sums, and atomically switches the active read pointer to the new generation
// without mutating or re-adding into the old generation (§10.9, §15.4).
func (s *Service) RebuildProjection(ctx context.Context) (RebuildProjectionResult, error) {
	var out RebuildProjectionResult
	now := s.now()

	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		state, err := s.getLedgerStateInTx(ctx, tx)
		if err != nil {
			return err
		}
		reconciling := state.Mode == LedgerModeReconciling

		var oldGenID uuid.UUID
		var oldGenText string
		err = tx.QueryRow(ctx, `
			SELECT id, generation::text
			FROM usage_projection_generations
			WHERE projection_name = $1 AND state = 'active'
			FOR UPDATE
		`, ProjectionNameMemberUsage).Scan(&oldGenID, &oldGenText)
		if err != nil {
			return fmt.Errorf("usage: lock active projection generation: %w", err)
		}
		oldGen, err := parseNumericUint64(oldGenText)
		if err != nil {
			return err
		}

		var maxGenText string
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(generation), 0)::text
			FROM usage_projection_generations
			WHERE projection_name = $1
		`, ProjectionNameMemberUsage).Scan(&maxGenText)
		if err != nil {
			return fmt.Errorf("usage: query max projection generation: %w", err)
		}
		maxGen, err := parseNumericUint64(maxGenText)
		if err != nil {
			return err
		}
		newGen := maxGen + 1
		newGenStr := formatUint64(newGen)

		newGenID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("usage: generate projection generation id: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO usage_projection_generations (
				id, projection_name, generation, state, started_at
			) VALUES ($1, $2, $3::numeric, 'building', $4)
		`, newGenID, ProjectionNameMemberUsage, newGenStr, now)
		if err != nil {
			return fmt.Errorf("usage: insert building projection generation: %w", err)
		}

		// Load all committed usage_samples in deterministic order.
		rows, err := tx.Query(ctx, `
			SELECT s.id,
			       s.event_id,
			       e.node_id,
			       e.incarnation_id,
			       s.member_id,
			       s.device_id,
			       s.interval_start,
			       s.interval_end,
			       s.upload_bytes::text,
			       s.download_bytes::text,
			       s.metering_version,
			       COALESCE(e.quality, 'complete'),
			       e.received_at
			FROM usage_samples s
			JOIN usage_events e ON e.id = s.event_id
			ORDER BY e.received_at ASC, e.id ASC, s.record_index ASC
		`)
		if err != nil {
			return fmt.Errorf("usage: query samples for rebuild: %w", err)
		}
		var samples []sampleForProjection
		for rows.Next() {
			var (
				smp            sampleForProjection
				upText, dnText string
			)
			if err := rows.Scan(
				&smp.SampleID,
				&smp.EventID,
				&smp.NodeID,
				&smp.IncarnationID,
				&smp.MemberID,
				&smp.DeviceID,
				&smp.IntervalStart,
				&smp.IntervalEnd,
				&upText,
				&dnText,
				&smp.MeteringVersion,
				&smp.Quality,
				&smp.ReceivedAt,
			); err != nil {
				rows.Close()
				return fmt.Errorf("usage: scan sample for rebuild: %w", err)
			}
			u, err := parseNumericUint64(upText)
			if err != nil {
				rows.Close()
				return err
			}
			d, err := parseNumericUint64(dnText)
			if err != nil {
				rows.Close()
				return err
			}
			smp.UploadBytes = u
			smp.DownloadBytes = d
			samples = append(samples, smp)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("usage: iterate samples for rebuild: %w", err)
		}

		if err := projectSamplesInTx(ctx, tx, newGen, reconciling, samples); err != nil {
			return err
		}

		// Replay node-reported gap events onto the new generation's rollups.
		gapRows, err := tx.Query(ctx, `
			SELECT id, node_id, incarnation_id,
			       COALESCE(metering_version, $1),
			       interval_start, interval_end, received_at
			FROM usage_events
			WHERE quality = 'gap'
			  AND kind = 'event'
			  AND interval_start IS NOT NULL
			  AND interval_end IS NOT NULL
			ORDER BY received_at ASC, id ASC
		`, MeteringVersionForwardAdmissionV1)
		if err != nil {
			return fmt.Errorf("usage: query gap events for rebuild: %w", err)
		}
		type gapEvt struct {
			id            uuid.UUID
			nodeID        uuid.UUID
			incarnationID uuid.UUID
			meteringVer   string
			start, end    time.Time
			receivedAt    time.Time
		}
		var gapEvts []gapEvt
		for gapRows.Next() {
			var ge gapEvt
			if err := gapRows.Scan(&ge.id, &ge.nodeID, &ge.incarnationID, &ge.meteringVer, &ge.start, &ge.end, &ge.receivedAt); err != nil {
				gapRows.Close()
				return fmt.Errorf("usage: scan gap event for rebuild: %w", err)
			}
			gapEvts = append(gapEvts, ge)
		}
		gapRows.Close()
		if err := gapRows.Err(); err != nil {
			return fmt.Errorf("usage: iterate gap events for rebuild: %w", err)
		}
		for _, ge := range gapEvts {
			if err := markGapOnRollupsInTx(ctx, tx, newGen, reconciling, ge.id, ge.nodeID, ge.incarnationID, ge.meteringVer, ge.start, ge.end, ge.receivedAt); err != nil {
				return err
			}
		}

		// Replay recorded replay_unavailable gaps onto the new generation's rollups.
		unavailRows, err := tx.Query(ctx, `
			SELECT node_id, affected_from, affected_until, affected_periods
			FROM usage_ledger_gaps
			WHERE kind = 'replay_unavailable'
		`)
		if err != nil {
			return fmt.Errorf("usage: query ledger gaps for rebuild: %w", err)
		}
		type unavailGap struct {
			nodeID   uuid.UUID
			from, to time.Time
			periods  []string
		}
		var unavailGaps []unavailGap
		for unavailRows.Next() {
			var ug unavailGap
			if err := unavailRows.Scan(&ug.nodeID, &ug.from, &ug.to, &ug.periods); err != nil {
				unavailRows.Close()
				return fmt.Errorf("usage: scan ledger gap for rebuild: %w", err)
			}
			unavailGaps = append(unavailGaps, ug)
		}
		unavailRows.Close()
		if err := unavailRows.Err(); err != nil {
			return fmt.Errorf("usage: iterate ledger gaps for rebuild: %w", err)
		}
		for _, ug := range unavailGaps {
			if err := markReplayUnavailableOnRollupsInTx(ctx, tx, newGen, ug.nodeID, ug.from, ug.to, ug.periods, now); err != nil {
				return err
			}
		}

		// Verify conservation: total bytes across new generation's month rollups and day rollups
		// must equal the exact sum of upload_bytes and download_bytes across all usage_samples.
		var sampleUp, sampleDown string
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(upload_bytes), 0)::text,
			       COALESCE(SUM(download_bytes), 0)::text
			FROM usage_samples
		`).Scan(&sampleUp, &sampleDown)
		if err != nil {
			return fmt.Errorf("usage: sum usage_samples for verification: %w", err)
		}

		for _, gran := range []Granularity{GranularityMonth, GranularityDay} {
			var rollupUp, rollupDown string
			err = tx.QueryRow(ctx, `
				SELECT COALESCE(SUM(upload_bytes), 0)::text,
				       COALESCE(SUM(download_bytes), 0)::text
				FROM usage_rollups
				WHERE projection_generation = $1::numeric AND granularity = $2
			`, newGenStr, string(gran)).Scan(&rollupUp, &rollupDown)
			if err != nil {
				return fmt.Errorf("usage: sum usage_rollups (%s) for verification: %w", gran, err)
			}
			if rollupUp != sampleUp || rollupDown != sampleDown {
				return fmt.Errorf("usage: projection rebuild verification failed for %s: rollup=(%s,%s) != samples=(%s,%s)",
					gran, rollupUp, rollupDown, sampleUp, sampleDown)
			}
		}

		verificationJSON, err := json.Marshal(map[string]any{
			"sample_count":   len(samples),
			"upload_bytes":   sampleUp,
			"download_bytes": sampleDown,
			"verified_at":    now.Format(time.RFC3339Nano),
		})
		if err != nil {
			return fmt.Errorf("usage: marshal verification json: %w", err)
		}

		// Switch active read pointer: mark oldGen superseded and newGen active.
		_, err = tx.Exec(ctx, `
			UPDATE usage_projection_generations
			SET state = 'superseded', superseded_at = $2
			WHERE id = $1
		`, oldGenID, now)
		if err != nil {
			return fmt.Errorf("usage: supersede old projection generation: %w", err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE usage_projection_generations
			SET state = 'active', activated_at = $2, verification = $3
			WHERE id = $1
		`, newGenID, now, verificationJSON)
		if err != nil {
			return fmt.Errorf("usage: activate new projection generation: %w", err)
		}

		out = RebuildProjectionResult{
			PreviousGeneration: oldGen,
			NewGeneration:      newGen,
			ProjectedSamples:   len(samples),
			VerifiedAt:         now,
		}
		return nil
	})
	if err != nil {
		return RebuildProjectionResult{}, err
	}
	return out, nil
}
