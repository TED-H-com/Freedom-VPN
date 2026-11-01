package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const EventGenerationHoldCheck = "node.generation_hold.check"

// Runs after receipt insertion, under the node/fleet locks. Report replay never
// reaches this path. Older observations cannot reopen or end an incident.
func recordGenerationHoldTx(ctx context.Context, tx pgx.Tx, r api.ObservationReport, now time.Time, current, frozen bool) error {
	if frozen {
		return reconcileGenerationHoldsTx(ctx, tx, r.NodeId, now)
	}
	var id uuid.UUID
	var firstObserved time.Time
	err := tx.QueryRow(ctx, `SELECT id,first_observed_at FROM node_generation_holds
		WHERE node_id=$1 AND incarnation_id=$2 AND ended_at IS NULL FOR UPDATE`, r.NodeId, r.NodeIncarnationId).Scan(&id, &firstObserved)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	leaseGen, known, err := reportedLeaseGeneration(ctx, tx, r)
	if err != nil {
		return err
	}
	applied, appliedErr := r.AppliedConfigGeneration.Get()
	marker := reportsGenerationHold(r)
	if current && !marker && known && appliedErr == nil && applied == leaseGen && id != uuid.Nil {
		if _, err := tx.Exec(ctx, `UPDATE node_generation_holds SET ended_at=$2,end_reason='generation_matched' WHERE id=$1`, id, now); err != nil {
			return err
		}
		return nil
	}
	if marker && known && appliedErr == nil && applied != leaseGen && (current || (id != uuid.Nil && !r.ObservedAt.Before(firstObserved))) {
		if id == uuid.Nil {
			leaseID, _ := r.Lease.LeaseId.Get()
			var revoked bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM serving_lease_members WHERE lease_id=$1)
				AND NOT EXISTS(SELECT 1 FROM serving_lease_members m JOIN members u ON u.id=m.member_id
				WHERE m.lease_id=$1 AND u.status NOT IN ('suspended','closed') AND NOT u.admin_suspended)`, leaseID).Scan(&revoked); err != nil {
				return err
			}
			if revoked {
				return nil
			}
			id = uuid.Must(uuid.NewV7())
			if _, err := tx.Exec(ctx, `INSERT INTO node_generation_holds
				(id,node_id,incarnation_id,first_seen_at,last_seen_at,first_observed_at,latest_report_id)
				VALUES($1,$2,$3,$4,$4,$5,$6)`, id, r.NodeId, r.NodeIncarnationId, now, r.ObservedAt, r.ReportId); err != nil {
				return err
			}
			payload := []byte(`{"schema_version":1}`)
			if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: r.NodeId, AggregateType: "node", AggregateVersion: "1",
				EventType: EventGenerationHoldCheck, SchemaVersion: 1, Payload: payload, AvailableAt: now.Add(generationHoldAlarmAfter)}); err != nil {
				return err
			}
		}
		summary, parsed := parseGenerationHoldSummary(r)
		var steps, elapsed *string
		var alarm *bool
		if parsed {
			s, e := summary.Steps.String(), summary.ElapsedMS.String()
			steps, elapsed, alarm = &s, &e, &summary.Alarm
		}
		valid := parsed && summary.Applied == applied && summary.Lease == leaseGen && summary.Steps > 0
		leaseID, _ := r.Lease.LeaseId.Get()
		if _, err := tx.Exec(ctx, `INSERT INTO node_generation_hold_reports
			(hold_id,node_id,incarnation_id,report_id,boot_id,lease_id,applied_generation,lease_generation,
			reported_steps,reported_elapsed_ms,reported_alarm,summary_valid)
			VALUES($1,$2,$3,$4,$5,$6,$7::text::numeric,$8::text::numeric,$9::text::numeric,$10::text::numeric,$11,$12)`,
			id, r.NodeId, r.NodeIncarnationId, r.ReportId, r.BootId, leaseID, applied.String(), leaseGen.String(), steps, elapsed, alarm, valid); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE node_generation_holds SET last_seen_at=greatest(last_seen_at,$2),
			latest_report_id=CASE WHEN $4 THEN $3 ELSE latest_report_id END WHERE id=$1`, id, now, r.ReportId, current); err != nil {
			return err
		}
	}
	return reconcileGenerationHoldsTx(ctx, tx, r.NodeId, now)
}

// ReconcileGenerationHolds also runs without a heartbeat: the delayed outbox
// event checks the 30s deadline; periodic node reconciliation repairs delayed
// work and closes incidents after revocation/freeze. A nil node means all nodes.
func (s *Service) ReconcileGenerationHolds(ctx context.Context, node uuid.UUID) error {
	return commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return reconcileGenerationHoldsTx(ctx, tx, node, s.now().UTC())
	})
}

// CloseInactiveGenerationHoldsTx joins terminal identity/member changes to the
// existing command transaction. Callers hold the fleet lock before node locks.
func CloseInactiveGenerationHoldsTx(ctx context.Context, tx pgx.Tx, node uuid.UUID, now time.Time) error {
	// A single member's removal must not reset an incident affecting survivors.
	// Empty grant sets do not manufacture revocation evidence.
	_, err := tx.Exec(ctx, `WITH endings AS (
		SELECT h.id, CASE
		WHEN n.current_incarnation_id IS DISTINCT FROM h.incarnation_id OR n.lifecycle_state<>'registered' OR i.revoked_at IS NOT NULL THEN 'identity_inactive'
		WHEN EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=h.incarnation_id AND x.mapping_frozen) THEN 'identity_frozen'
		WHEN EXISTS(SELECT 1 FROM serving_lease_members m WHERE m.lease_id=r.lease_id)
		 AND NOT EXISTS(SELECT 1 FROM serving_lease_members m JOIN members u ON u.id=m.member_id WHERE m.lease_id=r.lease_id AND u.status NOT IN ('suspended','closed') AND NOT u.admin_suspended) THEN 'members_revoked'
		END AS reason
		FROM node_generation_holds h JOIN nodes n ON n.id=h.node_id JOIN node_incarnations i ON i.id=h.incarnation_id
		JOIN node_generation_hold_reports r ON r.node_id=h.node_id AND r.incarnation_id=h.incarnation_id AND r.report_id=h.latest_report_id
		WHERE h.ended_at IS NULL AND ($1::uuid='00000000-0000-0000-0000-000000000000' OR h.node_id=$1))
		UPDATE node_generation_holds h SET ended_at=$2,end_reason=e.reason FROM endings e WHERE h.id=e.id AND e.reason IS NOT NULL`, node, now)
	return err
}

func reconcileGenerationHoldsTx(ctx context.Context, tx pgx.Tx, node uuid.UUID, now time.Time) error {
	if err := CloseInactiveGenerationHoldsTx(ctx, tx, node, now); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT h.id,h.node_id,h.incarnation_id,h.first_seen_at,h.last_seen_at,
		r.report_id,r.boot_id,r.lease_id,r.applied_generation::text,r.lease_generation::text,
		r.reported_steps::text,r.reported_elapsed_ms::text,r.reported_alarm,r.summary_valid,alarm.report_id
		FROM node_generation_holds h JOIN node_generation_hold_reports r
		ON r.node_id=h.node_id AND r.incarnation_id=h.incarnation_id AND r.report_id=h.latest_report_id
		LEFT JOIN LATERAL (SELECT report_id FROM node_generation_hold_reports
		 WHERE hold_id=h.id AND summary_valid AND reported_alarm ORDER BY report_id LIMIT 1) alarm ON true
		WHERE h.ended_at IS NULL AND h.escalated_at IS NULL
		AND ($1::uuid='00000000-0000-0000-0000-000000000000' OR h.node_id=$1)
		AND (h.first_seen_at <= $2 OR alarm.report_id IS NOT NULL) FOR UPDATE OF h`, node, now.Add(-generationHoldAlarmAfter))
	if err != nil {
		return err
	}
	var alarms []operations.GenerationHoldAlarm
	for rows.Next() {
		var a operations.GenerationHoldAlarm
		if err := rows.Scan(&a.IncidentID, &a.NodeID, &a.IncarnationID, &a.FirstSeenAt, &a.LastSeenAt,
			&a.ReportID, &a.BootID, &a.LeaseID, &a.AppliedGeneration, &a.LeaseGeneration,
			&a.ReportedSteps, &a.ReportedElapsedMS, &a.ReportedAlarm, &a.SummaryValid, &a.AlarmReportID); err != nil {
			rows.Close()
			return err
		}
		alarms = append(alarms, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, a := range alarms {
		if err := operations.RecordGenerationHoldAlarm(ctx, tx, a); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE node_generation_holds SET escalated_at=$2 WHERE id=$1`, a.IncidentID, now); err != nil {
			return err
		}
	}
	return nil
}

// ValidateGenerationHoldEvent is shared with the worker's durable event handler.
func ValidateGenerationHoldEvent(event db.ClaimedEvent) error {
	var payload struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return errors.New("invalid generation hold event")
	}
	if payload.SchemaVersion != 1 || event.AggregateType != "node" || event.AggregateID == uuid.Nil {
		return errors.New("invalid generation hold event")
	}
	return nil
}
