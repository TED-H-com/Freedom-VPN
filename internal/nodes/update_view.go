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
	return ReadUpdateSummary(ctx, q, node)
}

// ReadUpdateSummary reads the same projection used by node detail and list.
// Writers use it in their transaction to advance the node ETag only when the
// visible summary changes, including central-job precedence and incarnation.
func ReadUpdateSummary(ctx context.Context, q Querier, node uuid.UUID) (out presence.Nullable[api.NodeUpdateSummary], err error) {
	out.SetNull()
	var v api.NodeUpdateSummary
	var batch *uuid.UUID
	var observed, result *time.Time
	var code *api.ErrorCode
	var generation string
	var source api.UpdateResultSource
	// D-097: a current central job always wins. Without one, project only the
	// newest accepted receipt belonging to the node's current incarnation.
	err = q.QueryRow(ctx, `SELECT j.id,t.batch_id,j.generation::text,j.state,j.error_code,j.last_result_at,s.last_observed_at,'central'
 FROM fleet_node_state s JOIN update_jobs j ON j.id=s.current_job_id JOIN update_targets t ON t.id=j.target_id WHERE s.node_id=$1
 UNION ALL
 SELECT r.job_id,(r.payload->>'batch_id')::uuid,r.payload->>'generation',r.payload->>'state',r.payload->>'error_code',r.received_at,
 CASE WHEN r.source='central' THEN s.last_observed_at END,r.source
 FROM nodes n LEFT JOIN fleet_node_state s ON s.node_id=n.id
 CROSS JOIN LATERAL (
  SELECT e.job_id,e.payload,e.received_at,'central' AS source FROM update_results e
   WHERE e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id AND e.ack->>'accepted'='true'
  UNION ALL
  SELECT e.job_id,e.payload,e.received_at,'installer' AS source FROM installer_update_results e
   WHERE e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id
  ORDER BY received_at DESC,job_id DESC LIMIT 1
 ) r WHERE n.id=$1 AND s.current_job_id IS NULL`, node).Scan(&v.JobId, &batch, &generation, &v.State, &code, &result, &observed, &source)
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
	v.Source = &source
	v.LastErrorCode = presence.FromPointer(code)
	v.LastResultAt = presence.FromPointer(result)
	v.LastObservedAt = presence.FromPointer(observed)
	out.Set(v)
	return
}
