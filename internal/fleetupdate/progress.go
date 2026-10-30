package fleetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
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

// An unbound enrolling target still counts exactly once (§13.12). A newer
// desired generation supersedes it even though it has no execution job yet.
const targetStateSQL = `CASE WHEN t.target_state<>'active' THEN t.target_state
 WHEN j.id IS NOT NULL THEN j.state
 WHEN t.target_generation<n.update_generation THEN 'superseded'
 ELSE 'waiting_online' END`

func batchCounts(ctx context.Context, tx pgx.Tx, id uuid.UUID) (api.UpdateBatchCounts, api.UpdateBatchState, error) {
	rows, e := tx.Query(ctx, `SELECT `+targetStateSQL+` FROM update_targets t JOIN nodes n ON n.id=t.node_id LEFT JOIN LATERAL (SELECT id,state FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true WHERE t.batch_id=$1`, id)
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
		return api.UpdateBatchCounts{}, "", errors.New("batch snapshot target count changed")
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
		if e = tx.QueryRow(ctx, `SELECT operation_id FROM update_batches WHERE id=$1`, id).Scan(&op); e != nil {
			return e
		}
		if e = projectOperation(ctx, tx, op, "targets", state); e != nil {
			return e
		}
	}
	return recomputeRetries(ctx, tx)
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
	e = commandtx.InTx(ctx, s.pool, db.TxOptions{TxOptions: pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx) error {
		b = api.UpdateBatch{Targets: []api.UpdateTarget{},
			NextCursor: presence.NewNullNullable[string]()}
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
		b.CreatedAt = b.CreatedAt.UTC()
		b.UpdatedAt = b.UpdatedAt.UTC()
		b.TargetsUrl = "/api/v1/admin/update-batches/" + id.String()
		rows, e := tx.Query(ctx, `SELECT t.id,t.node_id,x.display_name,t.incarnation_at_snapshot,t.execution_incarnation_id,t.target_generation::text,t.online_at_snapshot,n.current_release_id,s.last_observed_at,j.last_result_at,
 `+targetStateSQL+`,j.id,coalesce(j.attempt,0),coalesce(j.state,'waiting_online'),j.error_code,j.stage,j.started_at,j.completed_at,coalesce(j.recovery_required,false),`+needsAttentionSQL+`,`+unconfirmedSQL+`,r.policy_generation::text,r.policy_refs,current_result.payload
 FROM update_targets t JOIN nodes n ON n.id=t.node_id LEFT JOIN exits x ON x.node_id=n.id LEFT JOIN fleet_node_state s ON s.node_id=n.id LEFT JOIN LATERAL (SELECT * FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true LEFT JOIN update_results r ON r.result_id=(j.observed_evidence->>'result_id')::uuid
 LEFT JOIN LATERAL (
  SELECT payload FROM update_results WHERE job_id=j.id
   AND ack->>'accepted'='true' AND ack->>'accepted_as_history'='false'
  ORDER BY received_at DESC,result_id DESC LIMIT 1
 ) current_result ON true
 WHERE t.batch_id=$1 AND t.id>$2 ORDER BY t.id LIMIT $3`, id, after, limit+1)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var t api.UpdateTarget
			var j api.UpdateJobSummary
			var jobID, incarnationAtSnapshot, executionIncarnation, currentRelease *uuid.UUID
			var lastObserved, lastResult, startedAt, finishedAt *time.Time
			var errorCode *api.ErrorCode
			var stage *api.UpdateStage
			var recoveryRequired bool
			var policyGeneration *string
			var policyRefs []uuid.UUID
			var resultPayload []byte
			var gen string
			var nodeName *string
			if e = rows.Scan(&t.TargetId, &t.NodeId, &nodeName, &incarnationAtSnapshot, &executionIncarnation, &gen, &t.OnlineAtSnapshot, &currentRelease, &lastObserved, &lastResult, &t.State, &jobID, &j.Attempt, &j.State, &errorCode, &stage, &startedAt, &finishedAt, &recoveryRequired, &t.NeedsAttention, &t.Unconfirmed, &policyGeneration, &policyRefs, &resultPayload); e != nil {
				return e
			}
			if len(b.Targets) == limit {
				c := releases.Cursor(b.Targets[len(b.Targets)-1].TargetId)
				b.NextCursor.Set(c)
				break
			}
			// Revision 7 / §13.8: an old applying job can finish its ledger as
			// history, replacing observed_evidence. Select current receipts by
			// their immutable ACK instead, retaining earlier current evidence.
			// applied_to_current_observation may be false after a heartbeat;
			// that does not turn an accepted current result into history.
			if e = setTargetResultEvidence(&t, resultPayload); e != nil {
				return e
			}
			// D-048: use the accepted result's immutable evidence, never current
			// desired state. The page join batches the same lookup as Evidence.
			t.PolicyGeneration.SetNull()
			t.PolicyRefs.SetNull()
			if policyGeneration != nil {
				generation, err := decstr.Parse(*policyGeneration)
				if err != nil {
					return err
				}
				t.PolicyGeneration.Set(generation)
			}
			if policyRefs != nil {
				t.PolicyRefs.Set(policyRefs)
			}
			t.Generation, e = decstr.Parse(gen)
			if e != nil {
				return e
			}
			utcTime(lastResult)
			utcTime(lastObserved)
			utcTime(startedAt)
			utcTime(finishedAt)
			t.IncarnationAtSnapshot = presence.FromPointer(incarnationAtSnapshot)
			t.ExecutionIncarnationId = presence.FromPointer(executionIncarnation)
			t.CurrentReleaseId = presence.FromPointer(currentRelease)
			t.LastObservedAt = presence.FromPointer(lastObserved)
			t.LastResultAt = presence.FromPointer(lastResult)
			t.NodeName = presence.FromPointer(nodeName)
			t.CurrentJob.SetNull()
			j.ErrorCode = presence.FromPointer(errorCode)
			j.Stage = presence.FromPointer(stage)
			j.StartedAt = presence.FromPointer(startedAt)
			j.FinishedAt = presence.FromPointer(finishedAt)
			j.RecoveryRequired = presence.NewRequired(recoveryRequired)

			t.Attempts = j.Attempt
			if jobID != nil {
				j.JobId = *jobID
				t.CurrentJob.Set(j)
			}
			t.Retryable = retryable(string(t.State), errorCode) && j.Attempt < 1000
			b.Targets = append(b.Targets, t)
		}
		return rows.Err()
	})
	return b, e
}

func setTargetResultEvidence(target *api.UpdateTarget, raw []byte) error {
	var evidence struct {
		State             api.UpdateResultState `json:"state"`
		PreviousReleaseID *uuid.UUID            `json:"previous_release_id"`
		Health            *api.UpdateHealth     `json:"health"`
		CandidateHealth   *api.UpdateHealth     `json:"candidate_health"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &evidence); err != nil {
			return err
		}
	}
	// Revision 7 requires explicit nulls even when no current receipt exists.
	target.PreviousReleaseId = presence.FromPointer(evidence.PreviousReleaseID)
	target.Health = presence.FromPointer(evidence.Health)
	target.CandidateHealth.SetNull()
	if evidence.State == api.UpdateResultStateRolledBack {
		target.CandidateHealth = presence.FromPointer(evidence.CandidateHealth)
	}
	return nil
}

// Shared by response mapping, evidence queries and node summaries. Receipt time
// must be strictly after authorization; another incarnation never confirms work.
const needsAttentionSQL = `coalesce(j.needs_attention,false) OR coalesce(j.state IN ('failed','rolled_back','blocked'),false) OR coalesce(j.state='already_current' AND n.current_release_id IS DISTINCT FROM j.release_id,false)`

// D-062: a generationless receipt has no observation incarnation to confirm
// update work. This nullable marker survives raw-heartbeat retention; the numeric
// cache remains a separate monotonic high-water mark for allocation and ordering.
const observedGenerationSQL = `CASE WHEN s.observation_incarnation_id IS NULL THEN NULL
 ELSE s.observed_update_generation END`
const unconfirmedSQL = `j.id IS NULL OR j.issued_at IS NULL OR s.last_observed_at IS NULL
 OR s.observation_incarnation_id IS DISTINCT FROM j.execution_incarnation_id
 OR (` + observedGenerationSQL + `) IS DISTINCT FROM j.generation
 OR coalesce(s.received_at<=j.issued_at,true)
 OR coalesce(j.last_result_at>s.received_at,false)
 OR j.last_observed_at IS DISTINCT FROM s.last_observed_at`

// TargetEvidence exposes the stored watchdog and observation evidence.
type TargetEvidence struct {
	NeedsAttention, Unconfirmed  bool
	LastResultAt, LastObservedAt *time.Time
	// Applied configuration evidence is independent of the update generation.
	// UpdateTarget exposes the same accepted-result evidence (D-048).
	PolicyGeneration *decstr.U64
	PolicyRefs       []uuid.UUID
}

func (s *Service) Evidence(ctx context.Context, target uuid.UUID) (TargetEvidence, error) {
	var e TargetEvidence
	var policyGeneration *string
	err := s.pool.QueryRow(ctx, `SELECT `+needsAttentionSQL+`,`+unconfirmedSQL+`,j.last_result_at,s.last_observed_at,r.policy_generation::text,r.policy_refs
 FROM update_targets t JOIN nodes n ON n.id=t.node_id LEFT JOIN fleet_node_state s ON s.node_id=t.node_id
 LEFT JOIN LATERAL (SELECT * FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true
 LEFT JOIN update_results r ON r.result_id=(j.observed_evidence->>'result_id')::uuid
 WHERE t.id=$1`, target).Scan(&e.NeedsAttention, &e.Unconfirmed, &e.LastResultAt, &e.LastObservedAt, &policyGeneration, &e.PolicyRefs)
	if err == nil && policyGeneration != nil {
		generation, parseErr := decstr.Parse(*policyGeneration)
		if parseErr != nil {
			return e, parseErr
		}
		e.PolicyGeneration = &generation
	}
	return e, err
}

// Reconcile is a worker step; watchdog attention never fabricates failure or
// success. Heartbeats/results remain the only actual-version evidence (§13.8).
func (s *Service) Reconcile(ctx context.Context) error {
	return commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
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

func utcTime(t *time.Time) {
	if t != nil {
		*t = t.UTC()
	}
}

func recomputeRetries(ctx context.Context, tx pgx.Tx) error {
	rows, e := tx.Query(ctx, `SELECT DISTINCT o.id FROM operations o JOIN update_retry_jobs r ON r.operation_id=o.id WHERE o.status IN ('pending','running') ORDER BY o.id`)
	if e != nil {
		return e
	}
	ids, e := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if e != nil {
		return e
	}
	for _, id := range ids {
		rows, e := tx.Query(ctx, `SELECT j.state FROM update_retry_jobs r JOIN update_jobs j ON j.id=r.job_id WHERE r.operation_id=$1`, id)
		if e != nil {
			return e
		}
		states, e := pgx.CollectRows(rows, pgx.RowTo[string])
		if e != nil {
			return e
		}
		_, state, e := CountStates(states)
		if e != nil {
			return e
		}
		if e = projectOperation(ctx, tx, id, "attempts", state); e != nil {
			return e
		}
	}
	return nil
}
