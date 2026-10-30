//go:build e2e

package members

import (
	"fmt"

	"github.com/google/uuid"
)

// Read immutable evidence from the complete run; never provoke a generation
// change to repair the empty restart lease. AT20's independent policy command
// already occurred and its later recovery cannot pass the earlier AT09 failure.
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
	var leases, observations []byte
	e1 := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(v ORDER BY at),'[]'::jsonb) FROM (SELECT issued_at at,jsonb_build_object('issued_at',issued_at,'expires_at',expires_at,'generation',policy_generation::text,'member_grants',jsonb_array_length(envelope->'signed'->'members')) v FROM serving_leases WHERE node_id=$1 AND issued_at>=$2::timestamptz-interval '2 minutes') t`, l.second.NodeID, r.At).Scan(&leases)
	e2 := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(v ORDER BY first_at),'[]'::jsonb) FROM (SELECT min(received_at) first_at,jsonb_build_object('first_at',min(received_at),'last_at',max(received_at),'observations',count(*),'generation',report_payload->'applied_config_generation','forwarding',report_payload->'member_forwarding_enabled','reasons',report_payload->'forwarding_block_reasons') v FROM node_observations WHERE node_id=$1 AND received_at>=$2::timestamptz GROUP BY report_payload->'applied_config_generation',report_payload->'member_forwarding_enabled',report_payload->'forwarding_block_reasons') t`, l.second.NodeID, r.At).Scan(&observations)
	r.check("immutable restart lease and observation timeline", "retain grant counts, deadlines and observed forwarding without modifying projections", fmt.Sprintf("leases=%s; observations=%s", leases, observations), "SELECT serving_leases and node_observations for recorded second incarnation since AT09", e1 == nil && e2 == nil)
	restore := l.useSecond()
	actual := l.counters(uuid.Nil)
	r.check("later independent-generation kernel readback", "record later kernel state without replacing failed recovery probe", fmt.Sprintf("available=%t; generation=%s; member_accept_rules=%d", actual.Available, actual.Generation, actual.MemberAcceptRules), "read only nft table after AT20/21 policy inputs; no recovery action", actual.Available)
	restore()
}
