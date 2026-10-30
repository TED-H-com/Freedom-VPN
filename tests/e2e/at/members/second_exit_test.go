//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

// Use the owner's exported hooks; no configuration/identity copying in this harness.
func (l *live) secondExit(r *record) bool {
	if l.second.NodeID != uuid.Nil {
		return true
	}
	if err := l.hook.StartSecondExit(l.ctx); err != nil {
		r.block("F4-02: shared StartSecondExit failed; owner tests/e2e/second_exit.go")
		return false
	}
	second, err := l.hook.EnrollAt(l.ctx, e2e.BrowserSession{Client: l.admin.client, CSRF: l.admin.csrf}, "TEST ONLY second acceptance exit", "node2")
	r.check("second exit enrolled through shared hook", "distinct durable identity and incarnation", fmt.Sprintf("enrolled=%t; distinct=%t", err == nil, second.NodeID != l.node && second.IncarnationID != l.inc), "Stack.StartSecondExit; Stack.EnrollAt(admin, name, node2)", err == nil && second.NodeID != l.node && second.IncarnationID != l.inc)
	if err != nil {
		r.block("F4-02: shared EnrollAt node2 did not complete")
		return false
	}
	l.second = second
	for _, s := range []string{"node2", "node2-netns", "node2-adguard"} {
		out, e := l.docker("inspect", "--format", "{{.Image}}", l.name(s))
		if e == nil {
			l.build.Images[s] = strings.TrimSpace(string(out))
		}
	}
	restore := l.useSecond()
	defer restore()
	return l.publish(r, []uuid.UUID{l.groupA})
}
func (l *live) useSecond() func() {
	node, inc, role := l.node, l.inc, l.nodeRole
	l.node, l.inc, l.nodeRole = l.second.NodeID, l.second.IncarnationID, "node2"
	return func() { l.node, l.inc, l.nodeRole = node, inc, role }
}

func (l *live) twoExits(r *record) {
	if !l.secondExit(r) {
		return
	}
	l.publish(r, []uuid.UUID{l.groupA})
	restore := l.useSecond()
	l.publish(r, []uuid.UUID{l.groupA})
	l.forwardingGate(r)
	restore()
	l.routeReadback(r)
	exits := l.api(l.a, "GET", "/api/v1/member/exits", nil, nil)
	var visible struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(exits.body, &visible)
	r.check("member sees both published exits", "200 and two exits", fmt.Sprintf("HTTP=%d; exits=%d", exits.status, len(visible.Items)), "GET /member/exits through member session", exits.status == 200 && len(visible.Items) == 2)
	device := l.awaitDevice(l.nativePeer)
	l.forwardingGate(r)
	l.traceForwarding(r, "member", device)
	restore = l.useSecond()
	l.traceForwarding(r, "member", device)
	beforeResp := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
	var beforeNode api.NodeDetail
	_ = json.Unmarshal(beforeResp.body, &beforeNode)
	desiredBefore := beforeNode.ConfigGeneration.Desired
	appliedBefore, _ := beforeNode.ConfigGeneration.Applied.Get()
	// Observe native failure behavior; do not presume automatic failover.
	// Stop all holders of the exit namespace: stopping only the agent
	// leaves kernel forwarding alive in node2-netns/adguard.
	_, stop := l.docker("stop", "--time", "5", l.name("node2"), l.name("node2-adguard"), l.name("node2-netns"))
	time.Sleep(3 * time.Second)
	probeID := uuid.NewString()
	l.request(nil, "PUT", "https://releases.example.invalid/e2e/traffic", map[string]string{"id": probeID}, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	raw, e := l.inside("member", "/tmp/at-members-probe", "transfer", probeID)
	arrival := l.request(nil, "GET", "https://releases.example.invalid/e2e/traffic", nil, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	var received struct {
		Requests int    `json:"requests"`
		Source   string `json:"source_address"`
	}
	_ = json.Unmarshal(arrival.body, &received)
	source := "none"
	if received.Requests > 0 {
		source = "unexpected"
		if received.Source == "192.0.4.20" {
			source = "primary"
		}
		if received.Source == "192.0.4.22" {
			source = "second"
		}
	}
	var v map[string]any
	_ = json.Unmarshal(raw, &v)
	safe, _ := json.Marshal(map[string]any{"probe": v, "route": l.routeObservation("member"), "origin_requests": received.Requests, "origin_node": source})
	r.check("selected exit offline behavior", "record bounded origin result and actual route", string(safe), "stop node2, its resolver and namespace holder; bounded HTTPS; authenticated origin arrival; restart same durable identity", stop == nil && e == nil)
	resumedAt := time.Now().UTC()
	_, nsErr := l.docker("start", l.name("node2-netns"))
	_, dnsErr := l.docker("start", l.name("node2-adguard"))
	_, resume := l.docker("start", l.name("node2"))
	r.check("second exit resume", "start succeeds", fmt.Sprintf("started=%t", resume == nil && nsErr == nil && dnsErr == nil), "docker start assigned node2", resume == nil && nsErr == nil && dnsErr == nil)
	var readyAt time.Time
	var provisionalAdmin map[string]any
	ready := l.wait(90*time.Second, func() bool {
		var observed, forwarding bool
		err := l.pool.QueryRow(l.ctx, `SELECT received_at>$2,(report_payload->>'member_forwarding_enabled')::boolean FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC LIMIT 1`, l.node, resumedAt).Scan(&observed, &forwarding)
		if err == nil && observed && !forwarding && provisionalAdmin == nil {
			provisionalAdmin = l.adminNodeSummary()
		}
		if err == nil && observed && forwarding && l.counters(device).MemberAcceptRules > 0 {
			readyAt = time.Now().UTC()
			return true
		}
		return false
	})
	r.check("second exit fresh serving after restart", "new accepted serving observation and nonempty kernel rules", fmt.Sprintf("ready=%t; restart_at=%s; ready_at=%s; ready_seconds=%.3f", ready, resumedAt.Format(time.RFC3339Nano), readyAt.Format(time.RFC3339Nano), readyAt.Sub(resumedAt).Seconds()), "poll post-restart node observation and kernel rules; never trust stale pre-restart central row", ready)
	l.forwardingGate(r)
	l.lastFirstByteAt = time.Time{}
	counted := l.traceForwarding(r, "member", device)
	firstByteAt := l.lastFirstByteAt
	l.recordRestartRecovery(r, desiredBefore, appliedBefore, resumedAt, readyAt, firstByteAt, provisionalAdmin, counted)
	l.resolverEvidence(r)
	restore()
	l.traceForwarding(r, "member", device)
	r.block("F4-03: two real exits and selected-node SNAT/counts are measured. Direct/forced relay, UDP/QUIC and PMTU baseline fixtures, four native platforms and a physical second NAT still need operator runs; see OPERATORS.md. No fastest-exit claim.")
}

func (l *live) adminNodeSummary() map[string]any {
	got := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
	var node api.NodeDetail
	if got.status != 200 || json.Unmarshal(got.body, &node) != nil {
		return nil
	}
	return summarizeNodeDetail(node)
}

func summarizeNodeDetail(node api.NodeDetail) map[string]any {
	applied, _ := node.ConfigGeneration.Applied.Get()
	forwarding, _ := node.MemberForwardingObserved.Get()
	var leaseCheck map[string]any
	for _, c := range node.Checks {
		if c.Name == api.NodeCheckNameLease {
			code, _ := c.ErrorCode.Get()
			detail := ""
			if c.Detail != nil {
				detail = *c.Detail
			}
			leaseCheck = map[string]any{"status": c.Status, "error_code": code, "detail": detail, "observed_at": c.ObservedAt}
			break
		}
	}
	return map[string]any{
		"desired_generation":         node.ConfigGeneration.Desired,
		"applied_generation":         applied,
		"provisioning_state":         node.ProvisioningState,
		"publication_intent":         node.PublicationIntent,
		"publication_observed":       node.PublicationObserved,
		"safety_status":              node.SafetyStatus,
		"member_forwarding_observed": forwarding,
		"lease_check":                leaseCheck,
	}
}

func (l *live) recordRestartRecovery(r *record, desiredBefore, appliedBefore decstr.U64, resumedAt, readyAt, firstByteAt time.Time, provisionalAdmin map[string]any, counted bool) {
	afterResp := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
	var afterNode api.NodeDetail
	decoded := afterResp.status == 200 && json.Unmarshal(afterResp.body, &afterNode) == nil
	desiredAfter := afterNode.ConfigGeneration.Desired
	appliedAfter, _ := afterNode.ConfigGeneration.Applied.Get()
	fwdAfter, _ := afterNode.MemberForwardingObserved.Get()
	sameGeneration := decoded && desiredBefore != 0 && desiredAfter == desiredBefore && appliedAfter == desiredBefore && appliedBefore == desiredBefore
	leases, observations, queryOK := l.nodeLeaseTimeline(l.node, r.At)
	var leaseRows []struct {
		IssuedAt     time.Time `json:"issued_at"`
		Generation   string    `json:"generation"`
		MemberGrants int       `json:"member_grants"`
	}
	_ = json.Unmarshal(leases, &leaseRows)
	postRestartPopulated := false
	for _, row := range leaseRows {
		if !row.IssuedAt.Before(resumedAt) && row.Generation == desiredBefore.String() && row.MemberGrants > 0 {
			postRestartPopulated = true
		}
	}
	firstByteSeconds := 0.0
	if !firstByteAt.IsZero() {
		firstByteSeconds = firstByteAt.Sub(resumedAt).Seconds()
	}
	readySeconds := 0.0
	if !readyAt.IsZero() {
		readySeconds = readyAt.Sub(resumedAt).Seconds()
	}
	summary, _ := json.Marshal(map[string]any{
		"desired_before":             desiredBefore,
		"applied_before":             appliedBefore,
		"desired_after":              desiredAfter,
		"applied_after":              appliedAfter,
		"same_desired_generation":    sameGeneration,
		"resumed_at":                 resumedAt.Format(time.RFC3339Nano),
		"serving_ready_at":           readyAt.Format(time.RFC3339Nano),
		"serving_ready_seconds":      readySeconds,
		"first_counted_byte_at":      firstByteAt.Format(time.RFC3339Nano),
		"first_counted_byte_seconds": firstByteSeconds,
		"provisional_admin_node":     provisionalAdmin,
		"recovered_admin_node":       summarizeNodeDetail(afterNode),
		"leases":                     json.RawMessage(leases),
		"observations":               json.RawMessage(observations),
	})
	r.check(
		"same-generation D-067 restart recovery and lease sequence",
		"forwarding restored on unchanged desired generation within bounded schedule; provisional renewal yields populated lease and counted fvmb0 transfer",
		string(summary),
		"GET /admin/nodes/{node2} before/during/after restart; SELECT serving_leases and contiguous node_observations; verify counted member transfer",
		sameGeneration && fwdAfter && postRestartPopulated && counted && !firstByteAt.IsZero() && firstByteSeconds > 0 && firstByteSeconds <= 90 && queryOK,
	)
}
