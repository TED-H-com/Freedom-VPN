//go:build e2e

package members

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (l *live) nb(method, path string, body any) response {
	return l.request(nil, method, "https://netbird.example.invalid/api"+path, body, map[string]string{"Authorization": "Token " + l.secret("netbird.token")})
}

type paidSession struct {
	member     *browser
	checkout   string
	start, end time.Time
	snapshot   map[string]any
}
type paymentEvent struct{ Body, Signature string }

func (l *live) paymentHook(path string, body any) response {
	return l.request(nil, "POST", "https://api.stripe.com/e2e/"+path, body, map[string]string{"X-E2E-Secret": l.secret("payment.secret")})
}
func (l *live) checkout(r *record, b *browser, end time.Time) *paidSession {
	headers := commandHeaders()
	body := api.CheckoutSessionRequest{PlanId: l.plan, ReturnUrl: "https://portal.example.invalid:8443/billing/return"}
	first := l.api(b, "POST", "/api/v1/member/checkout-sessions", body, headers)
	replay := l.api(b, "POST", "/api/v1/member/checkout-sessions", body, headers)
	var a, z api.CheckoutSession
	_ = json.Unmarshal(first.body, &a)
	_ = json.Unmarshal(replay.body, &z)
	ok := first.status == 201 && replay.status == 201 && a.CheckoutSessionId != uuid.Nil && a.CheckoutSessionId == z.CheckoutSessionId && a.CheckoutUrl == z.CheckoutUrl
	r.check("checkout command replay", "201 and same hosted session", fmt.Sprintf("first=%d; replay=%d; same_session=%t", first.status, replay.status, ok), "POST /member/checkout-sessions twice with same Idempotency-Key", ok)
	if !ok {
		return nil
	}
	p := &paidSession{member: b, start: time.Now().UTC().Add(-time.Minute).Truncate(time.Second), end: end.UTC().Truncate(time.Second)}
	if l.pool.QueryRow(l.ctx, `SELECT external_session_id FROM billing_sessions WHERE id=$1 AND member_id=$2`, a.CheckoutSessionId, b.member).Scan(&p.checkout) != nil {
		r.block("Checkout lacks provider session projection")
		return nil
	}
	p.snapshot = map[string]any{"checkout_id": p.checkout, "period_start": p.start.Unix(), "period_end": p.end.Unix(), "amount_minor": 1000, "currency": "USD", "status": "active", "paid": true, "cancel_at_period_end": false, "refund_minor": 0}
	return p
}
func (l *live) truth(r *record, p *paidSession) bool {
	got := l.paymentHook("snapshot", p.snapshot)
	r.check("authoritative provider truth", "200; input to provider, not a central invoice fixture", detail(got), "POST authenticated lab payment /e2e/snapshot for actual checkout", got.status == 200)
	return got.status == 200
}
func (l *live) event(r *record, p *paidSession, kind string) paymentEvent {
	got := l.paymentHook("event", map[string]string{"checkout_id": p.checkout, "type": kind})
	var e paymentEvent
	ok := got.status == 200 && json.Unmarshal(got.body, &e) == nil && e.Body != "" && e.Signature != ""
	r.check("provider event "+kind, "200 exact signed event bytes", fmt.Sprintf("HTTP=%d; signed_event_present=%t", got.status, ok), "POST authenticated lab /e2e/event; keep payload and signature in memory", ok)
	return e
}
func (l *live) sendEvent(e paymentEvent) response {
	return l.api(nil, "POST", "/api/v1/webhooks/stripe", []byte(e.Body), map[string]string{"Stripe-Signature": e.Signature})
}
func eventID(e paymentEvent) string {
	var v struct{ ID string }
	_ = json.Unmarshal([]byte(e.Body), &v)
	return v.ID
}
func (l *live) processed(e paymentEvent) bool {
	return l.wait(90*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1 AND processing_status='succeeded'`, eventID(e)) == 1
	})
}
func (l *live) paidReadback(r *record, p *paidSession, state string, refund int) bool {
	var raw []byte
	ok := l.wait(90*time.Second, func() bool {
		var matched bool
		err := l.pool.QueryRow(l.ctx, `SELECT s.last_reconciled_at IS NOT NULL AND s.paid_from=$2 AND s.paid_through=$3 AND e.state=$4 AND (SELECT coalesce(sum(amount_minor),0) FROM billing_records b WHERE b.subscription_id=s.id AND b.record_type='refund')=$5 FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.external_subscription_id=$1`, "sub_"+p.checkout, p.start, p.end, state, refund).Scan(&matched)
		return err == nil && matched
	})
	err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('provider_status',s.provider_status,'cancel_at_period_end',s.cancel_at_period_end,'paid_from',s.paid_from,'paid_through',s.paid_through,'last_reconciled',s.last_reconciled_at IS NOT NULL,'anomalies',s.reconciliation_anomalies,'entitlement',e.state,'revision',e.revision::text,'effective_service_end',e.effective_service_end,'invoices',(SELECT count(*) FROM billing_records b WHERE b.subscription_id=s.id AND b.record_type='paid_invoice'),'refund_total',(SELECT coalesce(sum(amount_minor),0) FROM billing_records b WHERE b.subscription_id=s.id AND b.record_type='refund')) FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.external_subscription_id=$1`, "sub_"+p.checkout).Scan(&raw)
	r.check("reconciled "+state, fmt.Sprintf("one exact paid interval; state=%s; refund=%d", state, refund), string(raw), "poll real subscription, billing_records and entitlements projections", ok && err == nil)
	l.write(r)
	return ok
}
func (l *live) revision(b *browser) string {
	var rev string
	_ = l.pool.QueryRow(l.ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, b.member).Scan(&rev)
	return rev
}
func (l *live) portal(r *record, b *browser) {
	u, _ := url.Parse("https://portal.example.invalid:8443")
	input, _ := json.Marshal(map[string]any{"port": 24443, "cookies": b.client.Jar.Cookies(u)})
	ctx, cancel := context.WithTimeout(l.ctx, 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", filepath.Join(l.root, "deploy/devstack/probe-portal.mjs"))
	cmd.Stdin = bytes.NewReader(input)
	raw, err := cmd.Output()
	var result struct {
		Status  int  `json:"http_status"`
		Visible bool `json:"billing_page_visible"`
		Error   bool `json:"api_error"`
		Pending bool `json:"pending_confirmation"`
	}
	decoded := json.Unmarshal(raw, &result) == nil
	r.check("real Portal billing return", "built billing page renders, pending confirmation; no API error", fmt.Sprintf("probe_ok=%t; HTTP=%d; page=%t; pending=%t; api_error=%t", err == nil, result.Status, result.Visible, result.Pending, result.Error), "shared deploy/devstack/probe-portal.mjs; cookies via stdin, controlled origin only", err == nil && decoded && result.Status == 200 && result.Visible && result.Pending && !result.Error)
}
func (l *live) payment(r *record) {
	if l.plan == uuid.Nil {
		r.block("API-created plan unavailable")
		return
	}
	l.portal(r, l.c)
	p := l.checkout(r, l.c, time.Now().Add(time.Hour))
	if p == nil {
		return
	}
	l.paid = p
	forged := l.api(l.c, "GET", "/billing/return?success=true&paid=true&session_id=TEST_ONLY_forged", nil, nil)
	subscriptions := l.count(`SELECT count(*) FROM subscriptions WHERE member_id=$1`, l.c.member)
	r.check("forged return before provider truth", "200 built Portal; no subscription or live entitlement", fmt.Sprintf("HTTP=%d; subscriptions=%d; state=%s", forged.status, subscriptions, l.entitlementState(l.c)), "GET forged return before snapshot; read product subscriptions and entitlement", forged.status == 200 && subscriptions == 0 && l.entitlementState(l.c) != "active")
	if !l.truth(r, p) {
		return
	}
	paid := l.event(r, p, "invoice.paid")
	bad := paid
	bad.Signature = "t=" + fmt.Sprint(time.Now().Unix()) + ",v1=" + strings.Repeat("0", 64)
	invalid := l.sendEvent(bad)
	r.check("invalid signature", "401; no inbox row", fmt.Sprintf("HTTP=%d; inbox=%d", invalid.status, l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, eventID(paid))), "POST provider's exact bytes with invalid signature", invalid.status == 401 && l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, eventID(paid)) == 0)
	initialRev := l.revision(l.c)
	first := l.sendEvent(paid)
	settled := l.processed(paid)
	r.check("first paid event", "202 then inbox succeeded", fmt.Sprintf("HTTP=%d; reconciled=%t", first.status, settled), "POST exact provider event; poll processing_status", first.status == 202 && settled)
	if !l.paidReadback(r, p, "active", 0) {
		return
	}
	paidRev := l.revision(l.c)
	accepted := 1
	for range 9 {
		reply := l.sendEvent(paid)
		r.check("paced paid replay", "202 once; no rate-limit retry", detail(reply), "wait 2 seconds; POST identical provider bytes once", reply.status == 202)
		if reply.status == 202 {
			accepted++
		}
	}
	invoices := l.count(`SELECT count(*) FROM billing_records WHERE member_id=$1 AND record_type='paid_invoice'`, l.c.member)
	rows := l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, eventID(paid))
	r.check("ten identical paid deliveries", "one inbox/invoice; one revision change then stable", fmt.Sprintf("accepted=%d; inbox=%d; invoices=%d; before=%s; paid=%s; after=%s", accepted, rows, invoices, initialRev, paidRev, l.revision(l.c)), "POST identical exact bytes 10 times; read keys, intervals and revisions", accepted == 10 && rows == 1 && invoices == 1 && initialRev != paidRev && paidRev == l.revision(l.c))
	l.paidReadback(r, p, "active", 0)
	l.write(r)
	l.paymentOrders(r)
	peer := l.native(r, l.c, "member-paid")
	device := l.awaitDevice(peer)
	if device == uuid.Nil {
		r.block("Paid native device admission did not converge")
	} else {
		l.forwardingGate(r)
		l.traceForwarding(r, "member-paid", device)
	}
}

func (l *live) paymentOrders(r *record) {
	seenMembers := map[string]bool{}
	orders := [][3]string{{"created", "paid", "failed"}, {"created", "failed", "paid"}, {"paid", "created", "failed"}, {"paid", "failed", "created"}, {"failed", "created", "paid"}, {"failed", "paid", "created"}}
	for i, order := range orders {
		b := l.login(fmt.Sprintf("AT-order-%s-%d", l.run, i), "member")
		r.check("independent ordering identity", "one new member per permutation", fmt.Sprintf("unique=%t", !seenMembers[b.member]), "shared IdP login with short fixture key", !seenMembers[b.member])
		if seenMembers[b.member] {
			r.block("Harness identity collision; stop ordering matrix without modifying projections")
			return
		}
		seenMembers[b.member] = true
		p := l.checkout(r, b, time.Now().Add(time.Hour))
		if p == nil || !l.truth(r, p) {
			return
		}
		invoice := l.event(r, p, "invoice.paid")
		created := l.event(r, p, "customer.subscription.created")
		failed := l.event(r, p, "invoice.payment_failed")
		events := map[string]paymentEvent{"created": created, "paid": invoice, "failed": failed}
		delivered, processed := 0, 0
		for _, kind := range order {
			if l.sendEvent(events[kind]).status == 202 {
				delivered++
			}
			if l.processed(events[kind]) {
				processed++
			}
		}
		good := l.paidReadback(r, p, "active", 0)
		rows := l.count(`SELECT count(*) FROM billing_records WHERE member_id=$1 AND record_type='paid_invoice'`, b.member)
		r.check("delivery order "+strings.Join(order[:], "/"), "each event processed against same authoritative paid truth; one exact interval", fmt.Sprintf("delivered=%d; processed=%d; invoices=%d; period_matches=%t", delivered, processed, rows, good), "fresh checkout/subject per order; authenticated provider hooks for all three event types; read paid interval", delivered == 3 && processed == 3 && rows == 1 && good)
		l.write(r)
	}
}
func (l *live) entitlementState(b *browser) string {
	var state string
	if l.pool.QueryRow(l.ctx, `SELECT state FROM entitlements WHERE member_id=$1`, b.member).Scan(&state) != nil {
		return "none"
	}
	return state
}
func (l *live) entitlements(r *record) {
	if l.paid == nil {
		r.block("AT03 paid session unavailable")
		return
	}
	p := l.paid
	// §9.2–9.3: cancellation preserves paid history, full refund ends effective service.
	p.snapshot["cancel_at_period_end"] = true
	p.snapshot["refund_minor"] = 250
	if l.truth(r, p) {
		ev := l.event(r, p, "customer.subscription.updated")
		got := l.sendEvent(ev)
		r.check("cancel and partial refund event", "202 then reconciled", detail(got), "update provider truth; POST signed event", got.status == 202 && l.processed(ev))
		l.paidReadback(r, p, "cancel_at_period_end", 250)
	}
	// Two separate native devices keep a persistent TCP connection on each exit.
	var secondMonitor *flowMonitor
	if l.secondExit(r) {
		peer := l.native(r, l.c, "member-paid2")
		device := l.awaitDevice(peer)
		restore := l.useSecond()
		l.forwardingGate(r)
		if device != uuid.Nil && l.traceForwarding(r, "member-paid2", device) {
			secondMonitor = l.startFlow(r, "member-paid2")
		}
		restore()
	}
	var paidDevice uuid.UUID
	_ = l.pool.QueryRow(l.ctx, `SELECT id FROM member_devices WHERE member_id=$1 AND status='authorized' ORDER BY created_at LIMIT 1`, l.c.member).Scan(&paidDevice)
	l.forwardingGate(r)
	l.traceForwarding(r, "member-paid", paidDevice)
	monitor := l.startFlow(r, "member-paid")
	refundAt := time.Now().UTC()
	p.snapshot["refund_minor"] = 1000
	p.snapshot["status"] = "canceled"
	if l.truth(r, p) {
		ev := l.event(r, p, "charge.refunded")
		got := l.sendEvent(ev)
		r.check("full refund event", "202 then reconciled", detail(got), "POST full refund truth and exact signed event", got.status == 202 && l.processed(ev))
		l.paidReadback(r, p, "expired", 1000)
		invoices := l.count(`SELECT count(*) FROM billing_records WHERE member_id=$1 AND record_type='paid_invoice'`, l.c.member)
		refunds := l.count(`SELECT count(*) FROM billing_records WHERE member_id=$1 AND record_type='refund'`, l.c.member)
		r.check("refund audit history", "one invoice retained; two refund deltas", fmt.Sprintf("invoices=%d; refunds=%d", invoices, refunds), "read immutable billing_records after cumulative 250/1000 refund", invoices == 1 && refunds == 2)
		deleted := l.event(r, p, "customer.subscription.deleted")
		got = l.sendEvent(deleted)
		r.check("cancellation deletion replay", "202; reconciled without restoring service", detail(got), "POST customer.subscription.deleted after canceled truth", got.status == 202 && l.processed(deleted) && l.entitlementState(l.c) == "expired")
	}
	l.finishFlow(r, monitor, refundAt)
	l.finishFlow(r, secondMonitor, refundAt)
	l.traceForwarding(r, "member-paid", uuid.Nil)
	if l.second.NodeID != uuid.Nil {
		restore := l.useSecond()
		l.traceForwarding(r, "member-paid2", uuid.Nil)
		l.publicationCommand(r, "disabled", []uuid.UUID{l.groupA})
		restore()
	}
	// Expiry runs on a fresh, paid subscription so no grant masks the boundary.
	expiryMember := l.login("AT-expiry-"+l.run, "member")
	ep := l.checkout(r, expiryMember, time.Now().Add(45*time.Second))
	if ep != nil && l.truth(r, ep) {
		ev := l.event(r, ep, "checkout.session.completed")
		got := l.sendEvent(ev)
		r.check("short real paid period", "202 then active", detail(got), "actual checkout and provider paid interval ending in 45 seconds", got.status == 202 && l.processed(ev))
		l.paidReadback(r, ep, "active", 0)
		expired := l.wait(100*time.Second, func() bool { return l.entitlementState(expiryMember) == "expired" })
		r.check("real paid expiry", "product projection expires at real UTC boundary", fmt.Sprintf("expired=%t; seconds_after_end=%.0f", expired, time.Since(ep.end).Seconds()), "poll entitlement.state without changing any clock", expired)
	}
	suspended := l.login("AT-suspend-"+l.run, "member")
	got := l.patchMember(suspended, map[string]any{"admin_suspended": true, "suspension_reason": "TEST ONLY successful payment precedence"})
	r.check("manual suspension before payment", "202", detail(got), "PATCH member with If-Match", got.status == 202)
	sp := l.checkout(r, suspended, time.Now().Add(time.Hour))
	if sp != nil && l.truth(r, sp) {
		ev := l.event(r, sp, "invoice.paid")
		paid := l.sendEvent(ev)
		r.check("successful payment while suspended", "202 then reconciled", detail(paid), "POST real paid event after manual suspension", paid.status == 202 && l.processed(ev))
		l.paidReadback(r, sp, "suspended", 0)
		screen := l.api(suspended, "GET", "/api/v1/member/me", nil, nil)
		r.check("member screen state", "200; admin_suspended true despite paid interval", fmt.Sprintf("HTTP=%d; suspended=%t", screen.status, strings.Contains(string(screen.body), `"admin_suspended":true`)), "GET /member/me after successful paid reconciliation", screen.status == 200 && strings.Contains(string(screen.body), `"admin_suspended":true`))
	}
	r.block("F4-04: persistent TCP refund withdrawal is attempted on both real exits. The shared UDP/4443 hook is an authenticated updater-only probe, not a member echo service; do not count it as member traffic. Member UDP plus expiry/suspension flow timing remains unproved. Owner: w3-devstack-e2e traffic fixtures.")
	r.block("Paid periods, cancellation/refunds, real expiry and suspension are control-plane checks. All-event TCP/UDP stop latency and rendered suspended/expired member screens still need the remaining flow and platform fixtures; OPERATORS.md retains the exact scripts.")
}
