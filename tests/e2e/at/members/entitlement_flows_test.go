//go:build e2e

package members

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// §19 AT04: enroll real paid devices on both exits before a rights change.
// Delays use the real clock and never update an entitlement projection.
func (l *live) paidPair(r *record, b *browser, label string) []string {
	if l.second.NodeID == uuid.Nil {
		r.block("Second exit absent for " + label + " TCP cohort")
		return nil
	}
	services := []string{"member-" + label + "1", "member-" + label + "2"}
	for i, s := range services {
		peer := l.native(r, b, s)
		device := l.awaitDevice(peer)
		restore := func() {}
		if i == 1 {
			restore = l.useSecond()
		}
		l.forwardingGate(r)
		if device == uuid.Nil || !l.traceForwarding(r, s, device) {
			restore()
			r.block(label + " lacks a positive paid flow on both nodes")
			return nil
		}
		restore()
	}
	return services
}
func (l *live) pairMonitors(r *record, services []string) []*flowMonitor {
	var monitors []*flowMonitor
	for _, s := range services {
		monitors = append(monitors, l.startFlow(r, s))
	}
	return monitors
}
func (l *live) endPair(r *record, services []string, monitors []*flowMonitor, event time.Time) {
	for _, m := range monitors {
		l.finishFlow(r, m, event)
	}
	for i, s := range services {
		restore := func() {}
		if i == 1 {
			restore = l.useSecond()
		}
		l.traceForwarding(r, s, uuid.Nil)
		restore()
	}
}
func (l *live) suspendedFlows(r *record) {
	b := l.login("AT-suspend-live-"+l.run, "member")
	p := l.checkout(r, b, time.Now().Add(time.Hour))
	if p == nil || !l.truth(r, p) {
		return
	}
	event := l.event(r, p, "invoice.paid")
	paid := l.sendEvent(event)
	r.check("suspension flow paid precondition", "202 and active", detail(paid), "checkout/provider paid truth before native admission", paid.status == 202 && l.processed(event) && l.paidReadback(r, p, "active", 0))
	services := l.paidPair(r, b, "suspend")
	monitors := l.pairMonitors(r, services)
	at := time.Now().UTC()
	suspended := l.patchMember(r, b, map[string]any{"admin_suspended": true, "suspension_reason": "TEST ONLY live flow withdrawal"})
	r.check("live suspension command", "202", detail(suspended), "PATCH member with live TCP connections on both nodes", suspended.status == 202)
	again := l.event(r, p, "invoice.paid")
	delivery := l.sendEvent(again)
	r.check("paid event after live suspension", "202, remains suspended", fmt.Sprintf("HTTP=%d; reconciled=%t", delivery.status, l.processed(again)), "deliver authoritative successful payment after suspension", delivery.status == 202 && l.entitlementState(b) == "suspended")
	l.endPair(r, services, monitors, at)
}
