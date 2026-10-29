package fleetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RetryAccepted struct {
	OperationID uuid.UUID
	TargetCount int
}

// Retry creates only attempts of the still-current target; retrying an old batch
// cannot supersede a newer desired release. HTTP adapter resolves OperationID.
func (s *Service) Retry(ctx context.Context, c releases.Command, batch uuid.UUID, r api.UpdateBatchRetryRequest) (RetryAccepted, error) {
	var out RetryAccepted
	raw, e := json.Marshal(r)
	if e != nil {
		return out, e
	}
	if e = s.releases.Validator.ValidateBody("UpdateBatchRetryRequest", raw); e != nil {
		return out, e
	}
	e = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		out = RetryAccepted{}
		rec, e := releases.Begin(ctx, tx, c, "/api/v1/admin/update-batches/"+batch.String()+"/retry", raw)
		if e != nil {
			return e
		}
		if !rec.New {
			out.OperationID = *rec.OperationID
			return tx.QueryRow(ctx, `SELECT count(*) FROM update_retry_jobs WHERE operation_id=$1`, out.OperationID).Scan(&out.TargetCount)
		}
		if e = s.lock(ctx, tx); e != nil {
			return e
		}
		var release uuid.UUID
		e = tx.QueryRow(ctx, `SELECT target_release_id FROM update_batches WHERE id=$1`, batch).Scan(&release)
		if errors.Is(e, pgx.ErrNoRows) {
			return apierr.New(apierr.ResourceNotFound, "Update batch does not exist")
		}
		if e != nil {
			return e
		}
		if _, _, _, e = s.releases.Eligible(ctx, tx, release); e != nil {
			return e
		}
		rows, e := tx.Query(ctx, `SELECT t.id,j.id,t.target_state FROM update_targets t LEFT JOIN LATERAL (SELECT id FROM update_jobs WHERE target_id=t.id ORDER BY attempt DESC LIMIT 1) j ON true WHERE t.batch_id=$1 ORDER BY t.id`, batch)
		if e != nil {
			return e
		}
		type pair struct {
			target uuid.UUID
			job    *uuid.UUID
			state  string
		}
		var all []pair
		for rows.Next() {
			var p pair
			if e = rows.Scan(&p.target, &p.job, &p.state); e != nil {
				rows.Close()
				return e
			}
			all = append(all, p)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		found := map[uuid.UUID]bool{}
		type candidate struct {
			node node
			job  job
		}
		var candidates []candidate
		for _, p := range all {
			if r.TargetIds != nil && !slices.Contains(*r.TargetIds, p.target) {
				continue
			}
			found[p.target] = true
			if p.job == nil || p.state != "active" {
				if r.TargetIds != nil {
					return apierr.New(apierr.StateConflict, "Selected target has no retryable job")
				}
				continue
			}
			j, e := readJob(ctx, tx, *p.job)
			if e != nil {
				return e
			}
			n, e := s.readNode(ctx, tx, j.nodeID)
			if e != nil {
				return e
			}
			var code *api.ErrorCode
			if e = tx.QueryRow(ctx, `SELECT error_code FROM update_jobs WHERE id=$1`, j.id).Scan(&code); e != nil {
				return e
			}
			allowed := retryable(j.state, code) && j.attempt < 1000 && n.lifecycle != "revoked" && n.lifecycle != "retired" && n.job != nil && *n.job == j.id && equalID(n.incarnation, j.incarnation) && n.generation == j.generation
			if !allowed {
				if r.TargetIds != nil {
					return apierr.New(apierr.StateConflict, "Selected target is not retryable or has newer work")
				}
				continue
			}
			candidates = append(candidates, candidate{n, j})
		}
		if r.TargetIds != nil {
			for _, id := range *r.TargetIds {
				if !found[id] {
					return apierr.New(apierr.ResourceNotFound, "Retry target is outside this batch")
				}
			}
		}
		if len(candidates) == 0 {
			return apierr.New(apierr.StateConflict, "No retryable current failed targets")
		}
		op, e := operations.Create(ctx, tx, operations.CreateRequest{
			Kind: api.OperationKindUpdateBatchRetry, Principal: auth.Principal{Role: auth.Admin, ID: c.ActorID},
			Resource: api.ResourceRef{Type: api.ResourceTypeUpdateBatch, Id: batch}, RequestID: commandtx.RequestID(ctx),
			Steps: map[string]operations.Step{"attempts": {Required: true}},
		})
		if e != nil {
			return e
		}
		out = RetryAccepted{OperationID: op, TargetCount: len(candidates)}
		for _, x := range candidates {
			if e = s.newJob(ctx, tx, x.node, x.job.target, release, x.job.generation, x.job.attempt+1); e != nil {
				return e
			}
			_, e = tx.Exec(ctx, `INSERT INTO update_retry_jobs(operation_id,job_id) SELECT $1,current_job_id FROM fleet_node_state WHERE node_id=$2`, op, x.node.id)
			if e != nil {
				return e
			}
		}
		if e = notify(ctx, tx, batch, "fleet.update_retry"); e != nil {
			return e
		}
		if e = releases.Audit(ctx, tx, c.ActorID, api.AuditEventActorTypeAdmin, "update_batch.retry", api.ResourceRef{Type: api.ResourceTypeUpdateBatch, Id: batch}, &op, api.AuditEventResultSucceeded, map[string]any{"reason": r.Reason, "target_count": len(candidates)}); e != nil {
			return e
		}
		if e = recomputeAll(ctx, tx); e != nil {
			return e
		}
		return commandtx.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{ResourceType: "update_batch", ResourceID: &batch, OperationID: &op, ResponseStatus: 202})
	})
	return out, e
}
