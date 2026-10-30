//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"time"
)

func (r *runner) secondExit(ctx context.Context) {
	rec := Record{ID: "second-exit", Title: "Second real exit enrollment and independent resolver", Spec: []string{"§6", "§19 AT09", "F3-03"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	s := &Stack{lab: r.lab}
	if err := s.StartSecondExit(ctx); err != nil {
		note("start", err.Error())
		return
	}
	admin := r.mustAdmin(ctx)
	if admin == nil {
		note("login", "failed")
		return
	}
	node, err := s.EnrollAt(ctx, BrowserSession{Client: admin.http, CSRF: admin.csrf}, "TEST ONLY second real exit", "node2")
	if err != nil {
		note("enroll", err.Error())
		return
	}
	note("distinct_identity", boolText(node.NodeID != r.nodeID && node.IncarnationID != r.incID))
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var detail map[string]json.RawMessage
	ready := waitUntil(wait, time.Second, func() bool {
		res, e := s.Call(wait, BrowserSession{Client: admin.http, CSRF: admin.csrf}, "GET", "/api/v1/admin/nodes/"+node.NodeID.String(), nil, nil)
		if e != nil || res.Status != 200 || json.Unmarshal(res.Body, &detail) != nil {
			return false
		}
		return string(detail["provisioning_state"]) == `"ready"`
	})
	for _, key := range []string{"provisioning_state", "safety_status", "data_health", "applied_config_generation"} {
		note(key, string(detail[key]))
	}
	raw, e := r.lab.docker(r.lab.prefix+"-node2", "nft", "-j", "list", "table", "inet", "fvpn")
	note("independent_nft_table", boolText(e == nil && len(raw) > 0))
	note("scope", "real second node hook; publication and simultaneous two-exit traffic remain harness scenarios, physical topology and boot flavours need operator runs")
	if ready && node.NodeID != r.nodeID && node.IncarnationID != r.incID && e == nil {
		rec.Status = StatusPass
	} else {
		note("owner", "w3-devstack-e2e hook / w2-agent-core readiness; see exact state")
	}
}
