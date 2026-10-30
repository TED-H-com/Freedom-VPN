//go:build e2e

package members

import (
	"fmt"

	"github.com/google/uuid"
)

// Read immutable evidence from the complete run; never provoke a generation
// change to repair the restart lease.
func (l *live) nodeLeaseTimeline(nodeID uuid.UUID, since string) ([]byte, []byte, bool) {
	var leases, observations []byte
	e1 := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(v ORDER BY at),'[]'::jsonb) FROM (SELECT issued_at at,jsonb_build_object('issued_at',issued_at,'expires_at',expires_at,'generation',policy_generation::text,'member_grants',jsonb_array_length(envelope->'signed'->'members')) v FROM serving_leases WHERE node_id=$1 AND issued_at>=$2::timestamptz-interval '2 minutes') t`, nodeID, since).Scan(&leases)
	e2 := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(v ORDER BY first_at),'[]'::jsonb) FROM (SELECT min(received_at) first_at,jsonb_build_object('first_at',min(received_at),'last_at',max(received_at),'observations',count(*),'generation',generation,'forwarding',forwarding,'reasons',reasons,'safety_status',safety_status,'lease_check',lease_check,'qos_check',qos_check,'source_binding_check',source_binding_check,'errors',errors) v FROM (SELECT received_at,generation,forwarding,reasons,safety_status,lease_check,qos_check,source_binding_check,errors,sum(step) OVER (ORDER BY received_at) AS grp FROM (SELECT received_at,generation,forwarding,reasons,safety_status,lease_check,qos_check,source_binding_check,errors,CASE WHEN lag(generation) OVER (ORDER BY received_at) IS DISTINCT FROM generation OR lag(forwarding) OVER (ORDER BY received_at) IS DISTINCT FROM forwarding OR lag(reasons) OVER (ORDER BY received_at) IS DISTINCT FROM reasons OR lag(safety_status) OVER (ORDER BY received_at) IS DISTINCT FROM safety_status OR lag(lease_check) OVER (ORDER BY received_at) IS DISTINCT FROM lease_check OR lag(qos_check) OVER (ORDER BY received_at) IS DISTINCT FROM qos_check OR lag(source_binding_check) OVER (ORDER BY received_at) IS DISTINCT FROM source_binding_check OR lag(errors) OVER (ORDER BY received_at) IS DISTINCT FROM errors THEN 1 ELSE 0 END AS step FROM (SELECT received_at,report_payload->'applied_config_generation' AS generation,report_payload->'member_forwarding_enabled' AS forwarding,report_payload->'forwarding_block_reasons' AS reasons,report_payload->'safety_status' AS safety_status,(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail') FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='lease' LIMIT 1) AS lease_check,(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail') FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='qos' LIMIT 1) AS qos_check,(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail') FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='source_binding' LIMIT 1) AS source_binding_check,(SELECT coalesce(jsonb_agg(jsonb_build_object('stage',e->'stage','code',e->'code','message',e->'message')),'[]'::jsonb) FROM jsonb_array_elements(coalesce(report_payload->'errors','[]'::jsonb)) e) AS errors FROM node_observations WHERE node_id=$1 AND received_at>=$2::timestamptz) raw) stepped) grouped GROUP BY grp,generation,forwarding,reasons,safety_status,lease_check,qos_check,source_binding_check,errors) t`, nodeID, since).Scan(&observations)
	return leases, observations, e1 == nil && e2 == nil
}

func (l *live) restartLeaseReadback(r *record) {
	inc, err := uuid.Parse(r.Nodes["node2"])
	if err != nil {
		r.block("Second-node incarnation absent from prior record")
		return
	}
	err = l.pool.QueryRow(l.ctx, `SELECT id,current_incarnation_id FROM nodes WHERE current_incarnation_id=$1`, inc).Scan(&l.second.NodeID, &l.second.IncarnationID)
	if err != nil {
		r.block("Second-node identity unavailable")
		return
	}
	if r.Status == "fail" {
		r.PriorChecks = append(r.PriorChecks, r.Steps...)
		r.Steps = nil
		r.Status = "blocked"
		r.block("F4-01: complete AT09 is blocked on the observed post-restart empty-member lease. Primary failed recovery assertion remains in prior_checks and the immutable attempt-2 archive; subsequent independent policy changes cannot make it a pass.")
	}
	leases, observations, ok := l.nodeLeaseTimeline(l.second.NodeID, r.At)
	r.check("immutable restart lease and observation timeline", "retain grant counts, deadlines and observed forwarding without modifying projections", fmt.Sprintf("leases=%s; observations=%s", leases, observations), "SELECT serving_leases and contiguous node_observations for recorded second incarnation since AT09", ok)
	restore := l.useSecond()
	actual := l.counters(uuid.Nil)
	r.check("later independent-generation kernel readback", "record later kernel state without replacing failed recovery probe", fmt.Sprintf("available=%t; generation=%s; member_accept_rules=%d", actual.Available, actual.Generation, actual.MemberAcceptRules), "read only nft table after AT20/21 policy inputs; no recovery action", actual.Available)
	restore()
}
