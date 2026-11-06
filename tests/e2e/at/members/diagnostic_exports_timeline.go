//go:build e2e || integration

package members

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type diagnosticQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// This immutable receipt timeline is diagnostic history, not Central's selected
// observation. The ID tiebreaker makes equal receipt times deterministic only.
// Full JSON state includes presence and incarnation: an omitted serving_read is
// distinct from null, and every changed read field starts a new group.
func readNodeLeaseTimeline(ctx context.Context, q diagnosticQuerier, nodeID uuid.UUID, since string) ([]byte, []byte, bool) {
	var leases, observations []byte
	e1 := q.QueryRow(ctx, `SELECT coalesce(jsonb_agg(v ORDER BY at),'[]'::jsonb) FROM
 (SELECT issued_at at,jsonb_build_object('lease_id',id,'issued_at',issued_at,'expires_at',expires_at,
 'generation',policy_generation::text,'member_grants',jsonb_array_length(envelope->'signed'->'members')) v
 FROM serving_leases WHERE node_id=$1 AND issued_at>=$2::timestamptz) t`, nodeID, since).Scan(&leases)
	e2 := q.QueryRow(ctx, `WITH raw AS (
 SELECT id,received_at,jsonb_build_object(
  'node_incarnation_id',incarnation_id,
  'generation',report_payload->'applied_config_generation',
  'forwarding',report_payload->'member_forwarding_enabled',
  'reasons',report_payload->'forwarding_block_reasons',
  'safety_status',report_payload->'safety_status',
  'lease_check',(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail')
    FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='lease' LIMIT 1),
  'qos_check',(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail')
    FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='qos' LIMIT 1),
  'source_binding_check',(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail')
    FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='source_binding' LIMIT 1),
  'errors',(SELECT coalesce(jsonb_agg(jsonb_build_object('stage',e->'stage','code',e->'code','message',e->'message')),'[]'::jsonb)
    FROM jsonb_array_elements(coalesce(report_payload->'errors','[]'::jsonb)) e)
 ) || CASE WHEN report_payload ? 'serving_read'
      THEN jsonb_build_object('serving_read',report_payload->'serving_read') ELSE '{}'::jsonb END AS state
 FROM node_observations WHERE node_id=$1 AND received_at>=$2::timestamptz
), stepped AS (
 SELECT *,row_number() OVER w AS ordinal,
  CASE WHEN lag(state) OVER w IS DISTINCT FROM state THEN 1 ELSE 0 END AS step
 FROM raw WINDOW w AS (ORDER BY received_at,id)
), grouped AS (
 SELECT *,sum(step) OVER (ORDER BY ordinal ROWS UNBOUNDED PRECEDING) AS grp FROM stepped
), compressed AS (
 SELECT min(ordinal) first_ordinal,jsonb_build_object('first_at',min(received_at),
  'last_at',max(received_at),'observations',count(*)) || state AS value
 FROM grouped GROUP BY grp,state
)
SELECT coalesce(jsonb_agg(value ORDER BY first_ordinal),'[]'::jsonb) FROM compressed`, nodeID, since).Scan(&observations)
	return leases, observations, e1 == nil && e2 == nil
}
