package nodes

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func updateSummary(ctx context.Context, q Querier, node uuid.UUID) (out presence.Nullable[api.NodeUpdateSummary], err error) {
	out.SetNull()
	var v api.NodeUpdateSummary
	var batch *uuid.UUID
	var observed, result *time.Time
	var code *api.ErrorCode
	var generation string
	err = q.QueryRow(ctx, `SELECT j.id,t.batch_id,j.generation::text,j.state,j.error_code,j.last_result_at,s.last_observed_at FROM fleet_node_state s JOIN update_jobs j ON j.id=s.current_job_id JOIN update_targets t ON t.id=j.target_id WHERE s.node_id=$1`, node).Scan(&v.JobId, &batch, &generation, &v.State, &code, &result, &observed)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	v.Generation, err = decstr.Parse(generation)
	if err != nil {
		return out, err
	}
	if observed != nil {
		*observed = observed.UTC()
	}
	if result != nil {
		*result = result.UTC()
	}
	v.BatchId = presence.FromPointer(batch)
	v.LastErrorCode = presence.FromPointer(code)
	v.LastResultAt = presence.FromPointer(result)
	v.LastObservedAt = presence.FromPointer(observed)
	out.Set(v)
	return
}
