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
	timingStack, timingErr := l.hook.MemberTransportStack(l.name("member"), "node2")
	if timingErr != nil || timingStack.EnableEnforcementTiming(l.ctx) != nil {
		r.block("Second exit timing unavailable before enrollment")
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
	defer l.timedSection("selected-exit-restart")()
	// The full assembled campaign can outlast a member browser idle session.
	// Reauthenticate through OIDC; do not lengthen the product timeout.
	reauthStarted := time.Now().UTC()
	l.a = l.login("AT-member-A-"+l.run, "member")
	r.jsonCheck("AT09 member OIDC reauthentication", "fresh authenticated member session before member endpoints", map[string]any{"started_at": reauthStarted, "completed_at": time.Now().UTC(), "member_id": l.a.member}, "real OIDC redirects and GET auth/session validated by login", true)
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
	leaseBefore := l.storedLeaseReadback("node2", false)
	stopStarted := time.Now().UTC()
	_, stop := l.docker("stop", "--time", "5", l.name("node2"), l.name("node2-adguard"), l.name("node2-netns"))
	stopEnded := time.Now().UTC()
	leaseStopped := l.storedLeaseReadback("node2", true)
	l.recordAgentTransition(r, "selected namespace stop", stopStarted, stopEnded, leaseBefore, leaseStopped, stop)
	time.Sleep(3 * time.Second)
	probeID := uuid.NewString()
	l.request(nil, "PUT", "https://releases.example.invalid/e2e/traffic", map[string]string{"id": probeID}, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	peerBefore := l.transferPeerViews("member")
	raw, e := l.inside("member", "/tmp/at-members-probe", "transfer", probeID)
	peerAfter := l.transferPeerViews("member")
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
	safe, _ := json.Marshal(map[string]any{"peer_status": peerBefore, "peer_status_after": peerAfter, "probe": v, "route": l.routeObservation("member"), "origin_requests": received.Requests, "origin_node": source})
	r.check("selected exit offline behavior", "record bounded origin result and actual route", string(safe), "stop node2, its resolver and namespace holder; bounded HTTPS; authenticated origin arrival; restart same durable identity", stop == nil && e == nil)
	// Keep the original stop/offline probe sequence, then let the shared hook
	// restart the already-stopped namespace and retain the entire 90-second
	// window. Observe counted recovery concurrently so collection does not
	// postpone the first traffic probe until after the acceptance deadline.
	resumedAt := time.Now().UTC()
	type restartResult struct {
		evidence e2e.NamespaceRestartEvidence
		err      error
	}
	restartDone := make(chan restartResult, 1)
	nodeID := l.node
	go func() {
		evidence, err := l.hook.RestartNodeNamespace90(l.ctx, "node2", nodeID)
		restartDone <- restartResult{evidence, err}
	}()
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
	capture := <-restartDone
	l.recordAgentTransition(r, "selected namespace restart and 90-second observation", resumedAt, time.Now().UTC(), leaseStopped, l.storedLeaseReadback("node2", false), capture.err)
	r.jsonCheck("second exit complete 90-second restart capture", "full window, unchanged generation; boot/final nft/tc, heartbeats, observations and agent log retained", map[string]any{"capture": capture.evidence, "error": safeError(capture.err)}, "RestartNodeNamespace90; no state, topology or generation repair", capture.err == nil && capture.evidence.GenerationBefore == desiredBefore.String() && capture.evidence.GenerationAfter == desiredBefore.String() && capture.evidence.Finished.Sub(capture.evidence.Started) >= 90*time.Second && len(capture.evidence.Samples) > 0)
	if !capture.evidence.Started.IsZero() {
		resumedAt = capture.evidence.Started
	}
	if capture.evidence.ServingAt != nil {
		readyAt = *capture.evidence.ServingAt
	}
	l.recordRestartRecovery(r, desiredBefore, appliedBefore, resumedAt, readyAt, firstByteAt, provisionalAdmin, counted)
	l.resolverEvidence(r)
	restore()
	l.traceForwarding(r, "member", device)
	l.nativeMatrix(r)
}

func (l *live) adminNodeSummary() map[string]any {
	got := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
	var node api.NodeDetail
	if got.status != 200 || json.Unmarshal(got.body, &node) != nil {
		return nil
	}
	return summarizeNodeDetail(node)
}

func (l *live) recordRestartRecovery(r *record, desiredBefore, appliedBefore decstr.U64, resumedAt, readyAt, firstByteAt time.Time, provisionalAdmin map[string]any, counted bool) {
	afterResp := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
	var afterNode api.NodeDetail
	decoded := afterResp.status == 200 && json.Unmarshal(afterResp.body, &afterNode) == nil
	desiredAfter := afterNode.ConfigGeneration.Desired
	appliedAfter, _ := afterNode.ConfigGeneration.Applied.Get()
	fwdAfter, _ := afterNode.MemberForwardingObserved.Get()
	sameGeneration := decoded && desiredBefore != 0 && desiredAfter == desiredBefore && appliedAfter == desiredBefore && appliedBefore == desiredBefore
	leases, observations, queryOK := l.nodeLeaseTimeline(l.node, resumedAt.Format(time.RFC3339Nano))
	var leaseRows []restartLease
	queryOK = queryOK && json.Unmarshal(leases, &leaseRows) == nil
	assessment, observationOK := l.verifyProvisionalObservations(r, resumedAt, desiredBefore.String(), provisionalAdmin, leaseRows, queryOK)
	queryOK = queryOK && observationOK
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
		"restart_assessment":         assessment,
		"leases":                     json.RawMessage(leases),
		"observations":               json.RawMessage(observations),
	})
	r.check(
		"same-generation D-067 restart recovery and lease sequence",
		"forwarding restored on unchanged desired generation within bounded schedule; grant-less then populated, or populated from first renewal; counted fvmb0 transfer",
		string(summary),
		"GET /admin/nodes/{node2} before/during/after restart; SELECT serving_leases and contiguous node_observations; verify counted member transfer",
		sameGeneration && fwdAfter && postRestartPopulated && assessment.SequenceOK && counted && !firstByteAt.IsZero() && firstByteSeconds > 0 && firstByteSeconds <= 90 && queryOK,
	)
	r.jsonCheck("restart without installed foreign filter reports no TC_FOREIGN_FILTER", "no foreign filter report in a recreated namespace with no injected tc object", assessment, "all post-restart reports; namespace recreation removes prior tc objects", queryOK && assessment.NoForeignFilter)
}

func (l *live) verifyProvisionalObservations(r *record, resumedAt time.Time, expectedGen string, provisionalAdmin map[string]any, leases []restartLease, leasesAvailable bool) (restartAssessment, bool) {
	rows, err := l.pool.Query(l.ctx, `SELECT received_at,report_payload FROM node_observations WHERE node_id=$1 AND received_at>=$2 ORDER BY received_at`, l.node, resumedAt)
	queryOK := err == nil && leasesAvailable
	observations := []restartObservation{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var row restartObservation
			var raw []byte
			if rows.Scan(&row.ReceivedAt, &raw) != nil || json.Unmarshal(raw, &row.Report) != nil {
				queryOK = false
				continue
			}
			observations = append(observations, row)
		}
		queryOK = queryOK && rows.Err() == nil
	}
	assessment := assessRestart(resumedAt, expectedGen, leases, observations)
	actual := map[string]any{"assessment": assessment, "provisional_admin_node": provisionalAdmin, "query_ok": queryOK}
	const name = "D-067 revision 11 provisional lease observation"
	if queryOK && assessment.D067Status == "blocked" {
		raw, _ := json.Marshal(actual)
		r.Steps = append(r.Steps, step{Name: name, Status: "blocked", Expected: "grant-less lease exercises D-067 revision 11 with D-126", Actual: string(raw), Command: "join post-restart reports to their held lease IDs", At: time.Now().UTC().Format(time.RFC3339)})
		r.block("D-067 restart subcheck not exercised: no grant-less lease after restart; populated publication still exercises D-067")
	} else {
		r.jsonCheck(name, "under a grant-less lease: awaiting_member_grants, D-126 check precedence and exactly one provisional stage=lease error", actual, "join post-restart report lease IDs; populated reports use safety checks", queryOK && assessment.D067Status == "pass")
	}
	return assessment, queryOK
}
