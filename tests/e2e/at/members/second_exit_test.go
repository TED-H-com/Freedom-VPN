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
	device := l.awaitDevice(l.nativePeer)
	l.forwardingGate(r)
	l.traceForwarding(r, "member", device)
	restore = l.useSecond()
	l.traceForwarding(r, "member", device)
	// Observe native failure behavior; do not presume automatic failover.
	_, stop := l.docker("stop", "--time", "5", l.name("node2"))
	time.Sleep(3 * time.Second)
	raw, e := l.inside("member", "/tmp/at-members-probe", "transfer", uuid.NewString())
	var v map[string]any
	_ = json.Unmarshal(raw, &v)
	safe, _ := json.Marshal(map[string]any{"probe": v, "route": l.routeObservation("member")})
	r.check("selected exit offline behavior", "record bounded origin result and actual route", string(safe), "stop own node2; bounded HTTPS; ip route get; restart same durable identity", stop == nil && e == nil)
	_, resume := l.docker("start", l.name("node2"))
	r.check("second exit resume", "start succeeds", fmt.Sprintf("started=%t", resume == nil), "docker start assigned node2", resume == nil)
	l.forwardingGate(r)
	l.traceForwarding(r, "member", device)
	l.resolverEvidence(r)
	restore()
	l.traceForwarding(r, "member", device)
	r.block("F4-03: two real exits and selected-node SNAT/counts are measured. Direct/forced relay, UDP/QUIC and PMTU baseline fixtures, four native platforms and a physical second NAT still need operator runs; see OPERATORS.md. No fastest-exit claim.")
}
