//go:build e2e

package members

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
)

func (l *live) nb(method, path string, body any) response {
	return l.request(nil, method, "https://netbird.example.invalid/api"+path, body, map[string]string{"Authorization": "Token " + l.secret("netbird.token")})
}
func (l *live) webhook(id, kind string, valid bool) response {
	object := map[string]any{"id": "in_TEST_ONLY_" + l.run, "customer": "cus_TEST_ONLY_" + l.run, "parent": map[string]any{"subscription_details": map[string]string{"subscription": "sub_TEST_ONLY_" + l.run}}}
	if strings.HasPrefix(kind, "customer.subscription.") {
		object = map[string]any{"id": "sub_TEST_ONLY_" + l.run, "customer": "cus_TEST_ONLY_" + l.run}
	}
	raw, _ := json.Marshal(map[string]any{"id": id, "type": kind, "created": 1, "api_version": billing.StripeAPIVersion, "livemode": false, "data": map[string]any{"object": object}})
	stamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(l.secret("webhook.secret")))
	_, _ = mac.Write([]byte(stamp + "."))
	_, _ = mac.Write(raw)
	sig := hex.EncodeToString(mac.Sum(nil))
	if !valid {
		sig = strings.Repeat("0", 64)
	}
	return l.api(nil, "POST", "/api/v1/webhooks/stripe", raw, map[string]string{"Stripe-Signature": "t=" + stamp + ",v1=" + sig})
}
func (l *live) payment(r *record) {
	headers := commandHeaders()
	body := map[string]any{"plan_id": l.plan, "return_url": "https://portal.example.invalid:8443/billing/return"}
	checkout := l.api(l.c, "POST", "/api/v1/member/checkout-sessions", body, headers)
	replay := l.api(l.c, "POST", "/api/v1/member/checkout-sessions", body, headers)
	var first, again struct {
		URL string `json:"checkout_url"`
	}
	_ = json.Unmarshal(checkout.body, &first)
	_ = json.Unmarshal(replay.body, &again)
	r.check("real hosted checkout replay", "201 with hosted URL; same command replays", fmt.Sprintf("first=%d; replay=%d; same_url=%t", checkout.status, replay.status, first.URL != "" && first.URL == again.URL), "POST /member/checkout-sessions twice with same Idempotency-Key", checkout.status == 201 && replay.status == 201 && first.URL != "" && first.URL == again.URL)
	before := l.count(`SELECT count(*) FROM subscriptions WHERE member_id=$1`, l.c.member)
	forged := l.api(l.c, "GET", "/billing/return?success=true&paid=true&session_id=TEST_ONLY_forged", nil, nil)
	after := l.count(`SELECT count(*) FROM subscriptions WHERE member_id=$1`, l.c.member)
	r.check("forged success return", "no subscription created", fmt.Sprintf("return_HTTP=%d; before=%d; after=%d", forged.status, before, after), "GET forged billing return; SELECT member subscriptions", before == 0 && after == 0)
	if forged.status == 404 {
		r.block("The running control origin returns 404 for /billing/return; no rendered Portal return page is served. No-rights side effect is observed, but the frontend-return journey is unavailable.")
	}
	badID := "evt_TEST_ONLY_bad_" + l.run
	bad := l.webhook(badID, "invoice.paid", false)
	badRows := l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, badID)
	r.check("invalid signature", "401, no inbox row", fmt.Sprintf("%s; inbox_rows=%d", detail(bad), badRows), "POST exact webhook bytes with invalid Stripe-Signature", bad.status == 401 && badRows == 0)
	var revBefore string
	_ = l.pool.QueryRow(l.ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, l.c.member).Scan(&revBefore)
	id := "evt_TEST_ONLY_replay_" + l.run
	accepted := 0
	for range 10 {
		w := l.webhook(id, "invoice.paid", true)
		if w.status == 202 || w.status == 200 {
			accepted++
		}
	}
	rows := l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, id)
	r.check("ten deliveries", "one durable inbox key", fmt.Sprintf("accepted=%d/10; inbox_rows=%d", accepted, rows), "POST same paid event ten times; SELECT source/external_event_id uniqueness", accepted == 10 && rows == 1)
	orders := [][3]string{{"created", "paid", "failed"}, {"created", "failed", "paid"}, {"paid", "created", "failed"}, {"paid", "failed", "created"}, {"failed", "created", "paid"}, {"failed", "paid", "created"}}
	kinds := map[string]string{"created": "customer.subscription.created", "paid": "invoice.paid", "failed": "invoice.payment_failed"}
	count := 0
	for i, order := range orders {
		for _, kind := range order {
			w := l.webhook(fmt.Sprintf("evt_TEST_ONLY_order_%s_%d_%s", l.run, i, kind), kinds[kind], true)
			if w.status == 202 || w.status == 200 {
				count++
			}
		}
	}
	r.check("all six delivery orders", "18 authenticated inbox deliveries", fmt.Sprintf("accepted=%d/18", count), "POST created/paid/failed in every permutation", count == 18)
	// The authoritative GET must work before a delivered event can mean paid.
	snapshot := l.request(nil, "GET", "https://api.stripe.com/v1/subscriptions/sub_TEST_ONLY_"+l.run, nil, map[string]string{"Authorization": "Bearer " + l.secret("stripe.key")})
	r.check("authoritative provider precondition", "record snapshot availability", detail(snapshot), "GET controlled payment provider /v1/subscriptions/{id}", snapshot.status != 0)
	var revAfter string
	_ = l.pool.QueryRow(l.ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, l.c.member).Scan(&revAfter)
	r.check("pending entitlement revision", "unreconciled deliveries do not extend interval", fmt.Sprintf("before=%s; after=%s", revBefore, revAfter), "SELECT entitlements.revision before/after webhook sequence", revBefore == revAfter)
	state := l.entitlementState(l.c)
	r.check("unreconciled event cannot grant", "pending member has no live entitlement", state, "GET /member/me; SELECT effective entitlement", state == "pending_payment" || state == "expired" || state == "none")
	if snapshot.status == 404 {
		r.block("Controlled payment provider returns 404 for the authoritative subscription snapshot and empty invoice/refund lists. Inbox replay/order is exercised; no paid subscription can reconcile, so one paid interval, revision stability after successful reconciliation, refunds and gateway authorization remain blocked. Add mutable snapshots to fvpn-e2e-payment, not fake production entitlements.")
	}
}
func (l *live) entitlementState(b *browser) string {
	var state string
	err := l.pool.QueryRow(l.ctx, `SELECT state FROM entitlements WHERE member_id=$1`, b.member).Scan(&state)
	if err != nil {
		return "none"
	}
	return state
}
func (l *live) entitlements(r *record) {
	expiry := time.Now().UTC().Add(12 * time.Second)
	g := l.grant(l.c, expiry)
	r.check("finite short entitlement", "202; active before real clock expiry", detail(g)+"; state="+l.entitlementState(l.c), "POST finite TEST ONLY grant, 12 seconds", g.status == 202 && l.entitlementState(l.c) == "active")
	expired := l.wait(90*time.Second, func() bool { return l.entitlementState(l.c) == "expired" })
	r.check("real expiry boundary", "effective interval ends without extending", fmt.Sprintf("expired=%t; elapsed_after_end_seconds=%.0f", expired, time.Since(expiry).Seconds()), "wait on real UTC clock; poll product entitlements.state projection", expired)
	suspend := l.patchMember(l.c, map[string]any{"admin_suspended": true, "suspension_reason": "TEST ONLY acceptance"})
	r.check("manual suspension", "202", detail(suspend), "PATCH member admin_suspended=true with If-Match", suspend.status == 202)
	grant := l.grant(l.c, time.Now().UTC().Add(2*time.Hour))
	state := l.entitlementState(l.c)
	r.check("grant does not clear suspension", "202; remains suspended", detail(grant)+"; state="+state, "POST second finite grant; read effective entitlement", grant.status == 202 && state == "suspended")
	screen := l.api(l.c, "GET", "/api/v1/member/me", nil, nil)
	r.check("member view model", "200 with admin_suspended=true", fmt.Sprintf("HTTP=%d; suspended_flag=%t", screen.status, strings.Contains(string(screen.body), `"admin_suspended":true`)), "GET /member/me after suspend", screen.status == 200 && strings.Contains(string(screen.body), `"admin_suspended":true`))
	for _, kind := range []string{"customer.subscription.updated", "customer.subscription.deleted", "charge.refunded", "invoice.paid"} {
		event := l.webhook("evt_TEST_ONLY_lifecycle_"+kind+"_"+l.run, kind, true)
		r.check("lifecycle inbox "+kind, "authenticated lifecycle event accepted, no payment truth inferred", detail(event), "POST exact signed lifecycle event", event.status == 202 || event.status == 200)
	}
	for _, path := range []string{"/v1/subscriptions/sub_TEST_ONLY_" + l.run, "/v1/invoices", "/v1/refunds"} {
		snapshot := l.request(nil, "GET", "https://api.stripe.com"+path, nil, map[string]string{"Authorization": "Bearer " + l.secret("stripe.key")})
		var list struct {
			Data []json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(snapshot.body, &list)
		r.check("lifecycle snapshot "+strings.Split(path, "/")[2], "record actual authoritative fixture availability", fmt.Sprintf("HTTP=%d; objects=%d", snapshot.status, len(list.Data)), "GET controlled provider authoritative snapshot/list", snapshot.status != 0)
	}
	r.block("No successful paid snapshot (AT03) or serving data path (AT08; native admission is now attempted in AT01/AT02). A grant surviving suspension is not proof of successful-payment precedence. Existing TCP/UDP stop latency, cancellation/refund timing, every-node enforcement and rendered member screen need those prerequisites.")
}
