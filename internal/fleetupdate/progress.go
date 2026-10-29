package fleetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CountStates is the single counting definition (§13.12). HTTP folds both
// cancellation reasons into cancelled and superseded_incarnation into superseded.
func CountStates(states []string) (api.UpdateBatchCounts, api.UpdateBatchState, error) {
	var c api.UpdateBatchCounts
	for _, state := range states {
		switch state {
		case "queued":
			c.Queued++
		case "waiting_online":
			c.WaitingOnline++
		case "downloading":
			c.Downloading++
		case "verifying":
			c.Verifying++
		case "applying":
			c.Applying++
		case "checking":
			c.Checking++
		case "succeeded":
			c.Succeeded++
		case "already_current":
			c.AlreadyCurrent++
		case "failed":
			c.Failed++
		case "rolled_back":
			c.RolledBack++
		case "superseded", "superseded_incarnation":
			c.Superseded++
		case "cancelled_revoked", "cancelled_retired":
			c.Cancelled++
		default:
			return c, "", fmt.Errorf("unknown update state %q", state)
		}
	}
	if c.WaitingOnline > 0 {
		return c, "waiting", nil
	}
	if c.Queued+c.Downloading+c.Verifying+c.Applying+c.Checking > 0 {
		return c, "running", nil
	}
	if c.Failed+c.RolledBack+c.Superseded+c.Cancelled > 0 {
		return c, "completed_with_exceptions", nil
	}
	return c, "completed", nil
}
func batchCounts(ctx context.Context, tx pgx.Tx, id uuid.UUID) (api.UpdateBatchCounts, api.UpdateBatchState, error) {
	rows, e := tx.Query(ctx, `SELECT j.state FROM update_targets t JOIN LATERAL (SELECT state FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true WHERE t.batch_id=$1`, id)
	if e != nil {
		return api.UpdateBatchCounts{}, "", e
	}
	states, e := pgx.CollectRows(rows, pgx.RowTo[string])
	if e != nil {
		return api.UpdateBatchCounts{}, "", e
	}
	var count int
	if e = tx.QueryRow(ctx, `SELECT target_count FROM update_batches WHERE id=$1`, id).Scan(&count); e != nil {
		return api.UpdateBatchCounts{}, "", e
	}
	if len(states) != count {
		return api.UpdateBatchCounts{}, "", errors.New("batch snapshot lacks a target job")
	}
	return CountStates(states)
}
func recomputeAll(ctx context.Context, tx pgx.Tx) error {
	rows, e := tx.Query(ctx, `SELECT id FROM update_batches ORDER BY id`)
	if e != nil {
		return e
	}
	ids, e := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if e != nil {
		return e
	}
	for _, id := range ids {
		counts, state, e := batchCounts(ctx, tx, id)
		if e != nil {
			return e
		}
		raw, e := json.Marshal(counts)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO update_batch_progress(batch_id,state,counts) VALUES($1,$2,$3) ON CONFLICT(batch_id) DO UPDATE SET state=excluded.state,counts=excluded.counts,updated_at=clock_timestamp() WHERE (update_batch_progress.state,update_batch_progress.counts) IS DISTINCT FROM (excluded.state,excluded.counts)`, id, state, raw)
		if e != nil {
			return e
		}
		var op uuid.UUID
		var status string
		if e = tx.QueryRow(ctx, `SELECT o.id,o.status FROM operations o JOIN update_batches b ON b.operation_id=o.id WHERE b.id=$1`, id).Scan(&op, &status); e != nil {
			return e
		}
		if status == "pending" || status == "failed" || status == "partially_succeeded" {
			if _, e = tx.Exec(ctx, `UPDATE operations SET status='running',completed_at=NULL WHERE id=$1`, op); e != nil {
				return e
			}
		}
		if state == "completed" || state == "completed_with_exceptions" {
			next := "succeeded"
			if state == "completed_with_exceptions" {
				next = "partially_succeeded"
			}
			_, e = tx.Exec(ctx, `UPDATE operations SET status=$2,completed_at=clock_timestamp(),steps='{"targets":{"required":true,"status":"succeeded","observed_ref":"batch"}}',observed_refs=jsonb_build_object('batch',$3::jsonb) WHERE id=$1 AND status='running'`, op, next, raw)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (s *Service) GetBatch(ctx context.Context, id uuid.UUID, cursor string, limit int) (api.UpdateBatch, error) {
	var b api.UpdateBatch
	after, e := releases.ParseCursor(cursor)
	if e != nil {
		return b, e
	}
	limit, e = releases.PageLimit(limit)
	if e != nil {
		return b, e
	}
	e = db.InTx(ctx, s.pool, db.TxOptions{TxOptions: pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx) error {
		b = api.UpdateBatch{Targets: []api.UpdateTarget{}}
		e := tx.QueryRow(ctx, `SELECT b.id,b.operation_id,b.target_release_id,b.target_count,b.created_at,b.created_by::uuid,b.reason,(r.manifest->'signed'->'compatibility'->>'requires_service_restart')::boolean,coalesce(p.updated_at,b.created_at) FROM update_batches b JOIN releases r ON r.id=b.target_release_id LEFT JOIN update_batch_progress p ON p.batch_id=b.id WHERE b.id=$1`, id).Scan(&b.BatchId, &b.OperationId, &b.ReleaseId, &b.TargetCount, &b.CreatedAt, &b.CreatedBy, &b.Reason, &b.RequiresServiceRestart, &b.UpdatedAt)
		if errors.Is(e, pgx.ErrNoRows) {
			return apierr.New(apierr.ResourceNotFound, "Update batch does not exist")
		}
		if e != nil {
			return e
		}
		b.Counts, b.State, e = batchCounts(ctx, tx, id)
		if e != nil {
			return e
		}
		b.TargetsUrl = "/api/v1/admin/update-batches/" + id.String()
		rows, e := tx.Query(ctx, `SELECT t.id,t.node_id,t.incarnation_at_snapshot,t.execution_incarnation_id,t.target_generation::text,t.online_at_snapshot,n.current_release_id,s.last_observed_at,j.last_result_at,
 j.id,j.attempt,j.state,j.error_code,j.stage,j.started_at,j.completed_at,j.recovery_required
 FROM update_targets t JOIN nodes n ON n.id=t.node_id LEFT JOIN fleet_node_state s ON s.node_id=n.id JOIN LATERAL (SELECT * FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true WHERE t.batch_id=$1 AND t.id>$2 ORDER BY t.id LIMIT $3`, id, after, limit+1)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var t api.UpdateTarget
			var j api.UpdateJobSummary
			var gen string
			if e = rows.Scan(&t.TargetId, &t.NodeId, &t.IncarnationAtSnapshot, &t.ExecutionIncarnationId, &gen, &t.OnlineAtSnapshot, &t.CurrentReleaseId, &t.LastObservedAt, &t.LastResultAt, &j.JobId, &j.Attempt, &j.State, &j.ErrorCode, &j.Stage, &j.StartedAt, &j.FinishedAt, &j.RecoveryRequired); e != nil {
				return e
			}
			if len(b.Targets) == limit {
				c := releases.Cursor(b.Targets[len(b.Targets)-1].TargetId)
				b.NextCursor = &c
				break
			}
			t.Generation, e = decstr.Parse(gen)
			if e != nil {
				return e
			}
			t.Attempts = j.Attempt
			t.State = j.State
			t.CurrentJob = &j
			t.Retryable = retryable(string(j.State), j.ErrorCode) && j.Attempt < 1000
			b.Targets = append(b.Targets, t)
		}
		return rows.Err()
	})
	return b, e
}

// TargetEvidence exposes fields not yet present in the generated HTTP response.
type TargetEvidence struct {
	NeedsAttention, Unconfirmed  bool
	LastResultAt, LastObservedAt *time.Time
}

func (s *Service) Evidence(ctx context.Context, target uuid.UUID) (TargetEvidence, error) {
	var e TargetEvidence
	err := s.pool.QueryRow(ctx, `SELECT j.needs_attention,s.last_observed_at IS NULL OR coalesce(j.last_result_at>s.received_at,false),j.last_result_at,s.last_observed_at FROM update_targets t LEFT JOIN fleet_node_state s ON s.node_id=t.node_id JOIN LATERAL (SELECT * FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true WHERE t.id=$1`, target).Scan(&e.NeedsAttention, &e.Unconfirmed, &e.LastResultAt, &e.LastObservedAt)
	return e, err
}

// Reconcile is a worker step; watchdog attention never fabricates failure or
// success. Heartbeats/results remain the only actual-version evidence (§13.8).
func (s *Service) Reconcile(ctx context.Context) error {
	return db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if e := s.lock(ctx, tx); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE update_jobs SET needs_attention=true WHERE state IN ('applying','checking') AND phase_since<clock_timestamp()-$1::bigint*interval '1 millisecond'`, s.config.StuckWindow.Milliseconds())
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE update_jobs j SET state='waiting_online' WHERE j.state='queued' AND NOT EXISTS(SELECT 1 FROM fleet_node_state s WHERE s.node_id=j.node_id AND s.observation_incarnation_id=j.execution_incarnation_id AND s.received_at>clock_timestamp()-$1::bigint*interval '1 millisecond') AND NOT EXISTS(SELECT 1 FROM node_observations o WHERE o.node_id=j.node_id AND o.incarnation_id=j.execution_incarnation_id AND o.control_status='online' AND o.received_at>clock_timestamp()-$1::bigint*interval '1 millisecond')`, s.config.OnlineWindow.Milliseconds())
		if e != nil {
			return e
		}
		return recomputeAll(ctx, tx)
	})
}
func retryable(state string, code *api.ErrorCode) bool {
	return state == "failed" && code != nil && apierr.Code(*code).DefaultRetryable()
}
