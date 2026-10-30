//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	ready := l.wait(90*time.Second, func() bool {
		var observed bool
		err := l.pool.QueryRow(l.ctx, `SELECT received_at>$2 AND (report_payload->>'member_forwarding_enabled')::boolean FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC LIMIT 1`, l.node, resumedAt).Scan(&observed)
		return err == nil && observed && l.counters(device).MemberAcceptRules > 0
	})
	r.check("second exit fresh serving after restart", "new accepted serving observation and nonempty kernel rules", fmt.Sprintf("ready=%t; restart_at=%s", ready, resumedAt.Format(time.RFC3339Nano)), "poll post-restart node observation and kernel rules; never trust stale pre-restart central row", ready)
	l.forwardingGate(r)
	l.traceForwarding(r, "member", device)
	l.resolverEvidence(r)
	restore()
	l.traceForwarding(r, "member", device)
	r.block("F4-03: two real exits and selected-node SNAT/counts are measured. Direct/forced relay, UDP/QUIC and PMTU baseline fixtures, four native platforms and a physical second NAT still need operator runs; see OPERATORS.md. No fastest-exit claim.")
}
