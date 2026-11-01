//go:build e2e

package e2e

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GenerationHoldSnapshot reads one consistent content-free central snapshot.
// Accepted reports are selected by the same node/incarnation/report keys as the
// product. No report messages, signatures, private keys or policy bodies escape.
func (s *Stack) GenerationHoldSnapshot(ctx context.Context, node uuid.UUID) (json.RawMessage, error) {
	db, err := s.lab.db(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close(ctx)
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var out json.RawMessage
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object(
 'sampled_at',clock_timestamp(),'node_id',$1::uuid,
 'incidents',coalesce((SELECT jsonb_agg(to_jsonb(h) ORDER BY h.first_seen_at,h.id) FROM node_generation_holds h WHERE h.node_id=$1),'[]'),
 'samples',coalesce((SELECT jsonb_agg((to_jsonb(r)-'applied_generation'-'lease_generation'-'reported_steps'-'reported_elapsed_ms') || jsonb_build_object('applied_generation',r.applied_generation::text,'lease_generation',r.lease_generation::text,'reported_steps',r.reported_steps::text,'reported_elapsed_ms',r.reported_elapsed_ms::text,'received_at',o.received_at,'observed_at',o.observed_at) ORDER BY o.received_at,r.report_id) FROM node_generation_hold_reports r JOIN node_observations o USING(node_id,incarnation_id,report_id) WHERE r.node_id=$1),'[]'),
 'reports',coalesce((SELECT jsonb_agg(jsonb_build_object('report_id',o.report_id,'incarnation_id',o.incarnation_id,'boot_id',o.report_payload->'boot_id','received_at',o.received_at,'observed_at',o.observed_at,'applied_generation',o.applied_generation::text,'lease_id',o.report_payload->'lease'->'lease_id','member_forwarding_enabled',o.member_forwarding_enabled,'forwarding_block_reasons',o.report_payload->'forwarding_block_reasons','checks',(SELECT jsonb_agg(jsonb_build_object('name',c->'name','status',c->'status','error_code',c->'error_code')) FROM jsonb_array_elements(o.report_payload->'checks') c),'errors',(SELECT jsonb_agg(jsonb_build_object('code',e->'code','stage',e->'stage','observed_at',e->'observed_at')) FROM jsonb_array_elements(o.report_payload->'errors') e)) ORDER BY o.received_at,o.id) FROM node_observations o WHERE o.node_id=$1),'[]'),
 'leases',coalesce((SELECT jsonb_agg(jsonb_build_object('lease_id',s.id,'incarnation_id',s.incarnation_id,'generation',s.policy_generation::text,'issued_at',s.issued_at,'expires_at',s.expires_at,'members',coalesce((SELECT jsonb_agg(jsonb_build_object('member_id',m.member_id,'valid_from',m.valid_from,'expires_at',m.expires_at) ORDER BY m.member_id) FROM serving_lease_members m WHERE m.lease_id=s.id),'[]')) ORDER BY s.issued_at,s.id) FROM serving_leases s WHERE s.node_id=$1),'[]'),
 'audits',coalesce((SELECT jsonb_agg(jsonb_build_object('id',a.id,'occurred_at',a.occurred_at,'action',a.action,'resource',a.resource,'result',a.result,'evidence',a.after_diff) ORDER BY a.occurred_at,a.id) FROM audit_events a WHERE a.action='node.generation_hold.escalated' AND a.resource='node:'||$1::text),'[]'))`, node).Scan(&out)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}
