package nodes

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type operationCaptureKey struct{}

// BeginOperationCapture serializes member fanout without holding a database
// transaction during external read-back. Pins survive retries and process exits.
// The worker must call release, and finish only after every compile succeeds.
func BeginOperationCapture(ctx context.Context, pool *pgxpool.Pool, event db.ClaimedEvent) (context.Context, func(), error) {
	var p struct {
		MemberID    uuid.UUID  `json:"member_id"`
		DeviceID    uuid.UUID  `json:"device_id"`
		OperationID *uuid.UUID `json:"operation_id"`
		Revision    string     `json:"revision"`
	}
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return ctx, nil, err
	}
	// Member suspension payloads also carry operation_id, but link by revision.
	if event.EventType != "device.authorized" && event.EventType != "device.revoked" {
		p.OperationID = nil
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return ctx, nil, err
	}
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,165))`, p.MemberID.String()).Scan(&locked); err != nil {
		_ = conn.Conn().Close(context.Background())
		conn.Release()
		return ctx, nil, err
	}
	if !locked {
		conn.Release()
		return ctx, nil, db.ErrLeaseLost
	}
	release := func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := conn.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended($1,165))`, p.MemberID.String()); err != nil {
			_ = conn.Conn().Close(cleanup)
		}
		conn.Release()
	}
	var ids []uuid.UUID
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT o.id,CASE WHEN o.resource_type='member_device' THEN 'device' ELSE 'member' END,
   CASE WHEN o.resource_type='member_device' THEN o.resource_id ELSE $1 END
   FROM operations o LEFT JOIN entitlement_grants g ON o.resource_type='entitlement_grant' AND g.id=o.resource_id
   WHERE o.kind IN ('member.update','member.suspend','member.unsuspend','member.grant','member.grant_revoke','member.device_authorize','member.device_revoke')
   AND o.status IN ('pending','running','failed','partially_succeeded')
   AND ($2::uuid IS NOT NULL OR o.status IN ('pending','running')) AND o.steps->'nodes'->>'status'<>'succeeded'
   AND (($2::uuid IS NOT NULL AND o.id=$2 AND o.resource_type='member_device' AND o.resource_id=$4) OR
    ($2::uuid IS NULL AND ((o.resource_type='member' AND o.resource_id=$1) OR g.member_id=$1)
     AND o.desired_generation<=$3::numeric AND (o.kind<>'member.grant' OR g.valid_from<=clock_timestamp())))
   ORDER BY o.id`, p.MemberID, p.OperationID, nullableRevision(p.Revision), p.DeviceID)
		if err != nil {
			return err
		}
		type candidate struct {
			ID          uuid.UUID
			SubjectType string
			Subject     uuid.UUID
		}
		candidates, err := pgx.CollectRows(rows, pgx.RowToStructByPos[candidate])
		if err != nil {
			return err
		}
		for _, c := range candidates {
			if _, err = tx.Exec(ctx, `INSERT INTO node_operation_captures(id,operation_id,event_id,subject_type,subject_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT(operation_id) DO NOTHING`, uuid.Must(uuid.NewV7()), c.ID, event.ID, c.SubjectType, c.Subject); err != nil {
				return err
			}
			ids = append(ids, c.ID)
		}
		return nil
	})
	if err != nil {
		release()
		return ctx, nil, err
	}
	return context.WithValue(ctx, operationCaptureKey{}, ids), release, nil
}
func nullableRevision(revision string) any {
	if revision == "" {
		return nil
	}
	return revision
}

// RecordOperationTargetsTx shares the node lock and transaction assigning the
// generation. Even a report before final capture can therefore update its pin.
func RecordOperationTargetsTx(ctx context.Context, tx pgx.Tx, node, incarnation uuid.UUID, generation string, before, after []byte) error {
	ids, _ := ctx.Value(operationCaptureKey{}).([]uuid.UUID)
	if len(ids) == 0 {
		return nil
	}
	if len(before) == 0 {
		before = []byte(`{}`)
	}
	_, err := tx.Exec(ctx, `WITH subjects AS (
  SELECT c.operation_id,
   EXISTS(SELECT 1 FROM jsonb_array_elements(coalesce($6::jsonb->'signed'->'device_bindings','[]')) b
    WHERE (c.subject_type='device' AND b->>'device_id'=c.subject_id::text) OR (c.subject_type='member' AND b->>'member_id'=c.subject_id::text))
   OR (c.subject_type='member' AND EXISTS(SELECT 1 FROM jsonb_array_elements(coalesce($6::jsonb->'signed'->'member_rate_policies','[]')) b WHERE b->>'member_id'=c.subject_id::text)) AS present,
   EXISTS(SELECT 1 FROM jsonb_array_elements(coalesce($5::jsonb->'signed'->'device_bindings','[]')) b
    WHERE (c.subject_type='device' AND b->>'device_id'=c.subject_id::text) OR (c.subject_type='member' AND b->>'member_id'=c.subject_id::text))
   OR (c.subject_type='member' AND EXISTS(SELECT 1 FROM jsonb_array_elements(coalesce($5::jsonb->'signed'->'member_rate_policies','[]')) b WHERE b->>'member_id'=c.subject_id::text)) AS previously_present
  FROM node_operation_captures c WHERE c.operation_id=ANY($1) AND c.captured_at IS NULL)
 INSERT INTO node_operation_targets(operation_id,node_id,incarnation_id,generation,subject_present)
 SELECT s.operation_id,$2,$3,$4::numeric,s.present FROM subjects s
 JOIN nodes n ON n.id=$2 AND n.lifecycle_state='registered' AND n.current_incarnation_id=$3
 JOIN exits e ON e.node_id=n.id WHERE s.present OR s.previously_present
 ON CONFLICT(operation_id,node_id) DO NOTHING`, ids, node, incarnation, generation, before, after)
	return err
}

// FinishOperationCaptureTx is called in the worker's policy receipt transaction.
// An empty captured set is evidence too; an incomplete capture cannot complete.
func FinishOperationCaptureTx(ctx context.Context, tx pgx.Tx, now time.Time) error {
	ids, _ := ctx.Value(operationCaptureKey{}).([]uuid.UUID)
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE node_operation_captures SET captured_at=$2 WHERE operation_id=ANY($1) AND captured_at IS NULL`, ids, now); err != nil {
		return err
	}
	if err := refreshUnconfirmedTargets(ctx, tx, ids, now, now); err != nil {
		return err
	}
	for _, id := range ids {
		if err := completeTargetOperation(ctx, tx, id, now); err != nil {
			return err
		}
	}
	return nil
}

func observeOperationTargets(ctx context.Context, tx pgx.Tx, node uuid.UUID, report api.ObservationReport, safety api.SafetyStatus, forwarding bool, now time.Time, observationRef string) error {
	generation, _ := report.AppliedConfigGeneration.Get()
	_, hasIso := hasIsolationApplyError(report)
	_, pending := isPendingIsolationApplyError(report, forwarding)
	clean := safety == api.SafetyStatusPassed && report.ProvisioningState == api.ProvisioningStateReady && !hasIso
	internetOnly := false
	for _, ch := range report.Checks {
		if ch.Name == api.NodeCheckNameInternetOnly && ch.Status == api.HealthCheckStatusPassed {
			internetOnly = true
		}
	}
	closed := isFailClosedIsolationApplyError(report, safety, forwarding) || (!forwarding && !pending && !hasIso && internetOnly)
	rows, err := tx.Query(ctx, `SELECT operation_id,subject_present,coalesce(error_code,''),outcome,generation<=$2::numeric AND compiled_at<=$3 FROM node_operation_targets
  WHERE node_id=$1 ORDER BY operation_id FOR UPDATE`, node, generation.String(), now)
	if err != nil {
		return err
	}
	type target struct {
		Op                     uuid.UUID
		Present                bool
		PreviousError          string
		Outcome                string
		GenerationAcknowledged bool
	}
	targets, err := pgx.CollectRows(rows, pgx.RowToStructByPos[target])
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.Op)
		if target.Outcome == "acknowledged" || !target.GenerationAcknowledged {
			continue
		}
		ref := observationRef
		resolution, reason := "conntrack_delete_succeeded", "clean_ready"
		if !clean {
			resolution, reason = "node_failed_closed", "node_failed_closed"
		}
		if clean || (closed && !target.Present) {
			if _, err = tx.Exec(ctx, `UPDATE node_operation_targets SET outcome='acknowledged',reason=$3,error_code=NULL,error_message=NULL,observed_at=$4,report_id=$5,acknowledged_incarnation_id=$6,observed_ref=$7 WHERE operation_id=$1 AND node_id=$2`, target.Op, node, reason, now, report.ReportId, report.NodeIncarnationId, ref); err != nil {
				return err
			}
			if err = targetAudit(ctx, tx, target.Op, node, "node.operation_target_acknowledged", "succeeded", map[string]any{"outcome": "acknowledged", "report_id": report.ReportId, "applied_config_generation": generation.String(), "incarnation_id": report.NodeIncarnationId, "previous_outcome": target.Outcome}); err != nil {
				return err
			}
			if target.PreviousError != "" || !clean {
				if err = targetAudit(ctx, tx, target.Op, node, "node.revoke_isolation_resolved", "succeeded", map[string]any{"resolution": resolution, "report_id": report.ReportId, "applied_config_generation": generation.String()}); err != nil {
					return err
				}
			}
		} else if pending {
			if _, err = tx.Exec(ctx, `UPDATE node_operation_targets SET outcome='pending',reason='conntrack_delete_pending',error_code='ISOLATION_POLICY_INVALID',error_message='member_forwarding_unconfirmed',observed_at=$3,report_id=$4,observed_ref=$5 WHERE operation_id=$1 AND node_id=$2`, target.Op, node, now, report.ReportId, ref); err != nil {
				return err
			}
			if target.PreviousError == "" {
				if err = targetAudit(ctx, tx, target.Op, node, "node.revoke_isolation_pending", "failed", map[string]any{"nodes_step_status": "pending", "error_code": "ISOLATION_POLICY_INVALID", "reason": "conntrack_delete_pending", "report_id": report.ReportId, "applied_config_generation": generation.String()}); err != nil {
					return err
				}
			}
		}
	}
	// Only nonblocking target locks are taken here, before any operation lock.
	// Another reporter holding its target will perform its own final re-read.
	if err = refreshUnconfirmedTargets(ctx, tx, ids, now, report.ObservedAt.UTC()); err != nil {
		return err
	}
	for _, id := range ids {
		if err = completeTargetOperation(ctx, tx, id, now); err != nil {
			return err
		}
	}
	return nil
}

// refreshUnconfirmedTargets never waits on another reporter's target while
// holding this reporter's target. All operation locks follow these row writes.
func refreshUnconfirmedTargets(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, now, observedNow time.Time) error {
	rows, err := tx.Query(ctx, `SELECT t.operation_id,t.node_id,t.outcome,coalesce(t.reason,''),
  CASE WHEN n.lifecycle_state IN ('revoked','retired') THEN 'node_'||n.lifecycle_state
   WHEN n.last_observed_at IS NULL OR n.last_observed_at<$3::timestamptz-interval '90 seconds' THEN 'observation_stale'
   WHEN n.last_seen_at IS NULL OR n.last_seen_at<$2::timestamptz-interval '90 seconds' THEN 'contact_stale'
   WHEN n.control_status<>'online' THEN 'node_offline' ELSE '' END
  FROM node_operation_targets t JOIN nodes n ON n.id=t.node_id
  WHERE t.operation_id=ANY($1) AND t.outcome<>'acknowledged'
  ORDER BY t.operation_id,t.node_id FOR UPDATE OF t SKIP LOCKED`, ids, now, observedNow)
	if err != nil {
		return err
	}
	type row struct {
		Op, Node                   uuid.UUID
		Outcome, OldReason, Reason string
	}
	targets, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
	if err != nil {
		return err
	}
	for _, t := range targets {
		if t.Reason != "" {
			if t.Outcome == "unconfirmed" && t.OldReason == t.Reason {
				continue
			}
			if _, err = tx.Exec(ctx, `UPDATE node_operation_targets SET outcome='unconfirmed',reason=$3 WHERE operation_id=$1 AND node_id=$2`, t.Op, t.Node, t.Reason); err != nil {
				return err
			}
			if err = targetAudit(ctx, tx, t.Op, t.Node, "node.operation_target_unconfirmed", "succeeded", map[string]any{"outcome": "unconfirmed", "reason": t.Reason}); err != nil {
				return err
			}
		} else if t.Outcome == "unconfirmed" {
			if _, err = tx.Exec(ctx, `UPDATE node_operation_targets SET outcome='pending',reason=CASE WHEN error_code IS NOT NULL THEN 'conntrack_delete_pending' END WHERE operation_id=$1 AND node_id=$2`, t.Op, t.Node); err != nil {
				return err
			}
		}
	}
	return nil
}

func targetAudit(ctx context.Context, tx pgx.Tx, op, node uuid.UUID, action, result string, after map[string]any) error {
	var resource api.ResourceRef
	var request *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT resource_type,resource_id,request_id FROM operations WHERE id=$1`, op).Scan(&resource.Type, &resource.Id, &request); err != nil {
		return err
	}
	actor := audit.Actor{Type: api.AuditEventActorTypeSystem, ID: SystemActorUUID}
	if node != uuid.Nil {
		after["node_id"] = node
		actor = audit.Actor{Type: api.AuditEventActorTypeNode, ID: node}
	}
	fields := make([]string, 0, len(after))
	for key := range after {
		fields = append(fields, key)
	}
	event, err := audit.NewEvent(audit.Entry{Actor: actor, Action: action, Resource: resource, OperationID: &op, RequestID: request, Result: api.AuditEventResult(result), Before: map[string]any{}, After: after, Fields: fields})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}

func completeTargetOperation(ctx context.Context, tx pgx.Tx, id uuid.UUID, now time.Time) error {
	var status, step string
	var capture uuid.UUID
	var captured *time.Time
	// Mark target rows first, then lock and re-read. READ COMMITTED sees the
	// other last reporter's commit after waiting for this operation lock.
	if err := tx.QueryRow(ctx, `SELECT o.status,o.steps->'nodes'->>'status',c.ID,c.captured_at FROM operations o JOIN node_operation_captures c ON c.operation_id=o.id WHERE o.id=$1 FOR UPDATE OF o`, id).Scan(&status, &step, &capture, &captured); err != nil {
		return err
	}
	if captured == nil || status == "succeeded" || status == "superseded" || step == "succeeded" {
		return nil
	}
	var outcomes []byte
	var waiting, pendingIsolation bool
	if err := tx.QueryRow(ctx, `SELECT coalesce(jsonb_agg(jsonb_build_object('node_id',node_id,'incarnation_id',incarnation_id,'generation',generation::text,'subject_present',subject_present,'outcome',outcome,'reason',reason,'error_code',error_code,'error_message',error_message,'observed_at',observed_at,'report_id',report_id,'acknowledged_incarnation_id',acknowledged_incarnation_id,'observed_ref',observed_ref) ORDER BY node_id),'[]'),coalesce(bool_or(outcome='pending'),false),coalesce(bool_or(outcome='pending' AND error_code='ISOLATION_POLICY_INVALID'),false) FROM node_operation_targets WHERE operation_id=$1`, id).Scan(&outcomes, &waiting, &pendingIsolation); err != nil {
		return err
	}
	if pendingIsolation {
		_, err := tx.Exec(ctx, `UPDATE operations SET steps=jsonb_set(steps,'{nodes}',(steps->'nodes')||jsonb_build_object('status','pending','error_code','ISOLATION_POLICY_INVALID','error_message','member_forwarding_unconfirmed','reason','conntrack_delete_pending','observed_at',$2::timestamptz,'updated_at',$2::timestamptz)), observed_refs=coalesce(observed_refs,'{}')||jsonb_build_object('nodes_isolation_pending',jsonb_build_object('reference',$3::text,'observed_at',$2::timestamptz)),updated_at=$2,resource_version=coalesce(resource_version,0)+1 WHERE id=$1`, id, now, "node_operation_capture:"+capture.String())
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE operations SET steps=jsonb_set(steps,'{nodes}',(steps->'nodes')-'error_code'-'error_message'-'reason'),observed_refs=coalesce(observed_refs,'{}')-'nodes_isolation_pending' WHERE id=$1`, id); err != nil {
		return err
	}
	if waiting {
		return nil
	}
	if step == "failed" {
		if err := entitlement.TransitionOperation(ctx, tx, id, "nodes", api.OperationStepStatusRunning, nil); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE node_operation_captures SET completed_at=$2,outcomes=$3 WHERE operation_id=$1`, id, now, outcomes); err != nil {
		return err
	}
	if err := targetAudit(ctx, tx, id, uuid.Nil, "member.nodes_completed", "succeeded", map[string]any{"capture_id": capture, "targets": json.RawMessage(outcomes)}); err != nil {
		return err
	}

	return entitlement.TransitionOperation(ctx, tx, id, "nodes", api.OperationStepStatusSucceeded, &operations.Evidence{Reference: "node_operation_capture:" + capture.String(), ObservedAt: now})
}

func sweepOperationTargets(ctx context.Context, tx pgx.Tx, now time.Time) error {
	rows, err := tx.Query(ctx, `SELECT c.operation_id FROM node_operation_captures c JOIN operations o ON o.id=c.operation_id WHERE c.captured_at IS NOT NULL AND o.steps->'nodes'->>'status'<>'succeeded' AND o.status<>'superseded' ORDER BY c.operation_id`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	if err = refreshUnconfirmedTargets(ctx, tx, ids, now, now); err != nil {
		return err
	}
	for _, id := range ids {
		if err = completeTargetOperation(ctx, tx, id, now); err != nil {
			return err
		}
	}
	return nil
}
