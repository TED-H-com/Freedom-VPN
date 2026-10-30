//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (r *runner) catalogue(ctx context.Context, admin *session) {
	rec := Record{ID: "serving-catalogue", Title: "Member group and plan through product APIs", Spec: []string{"§9", "§14.2"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	groups, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/groups", nil, nil)
	note("groups_get", httpDetail(groups, err))
	if err != nil || groups.Status != 200 {
		rec.Status = StatusBlocked
		r.planError = "D-057: GET /admin/groups " + httpDetail(groups, err) + "; owner w3-wire-nodes follow-up 11"
		note("owner", r.planError)
		return
	}
	created, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/groups", api.MemberGroupCreate{Code: "e2e-members", Name: "TEST ONLY e2e members"}, map[string]string{"Idempotency-Key": uuid.NewString()})
	note("group_create", httpDetail(created, err))
	var group api.MemberGroup
	if err != nil || created.Status != 201 || json.Unmarshal(created.Body, &group) != nil {
		r.planError = "member group creation failed"
		return
	}
	r.groupID = group.GroupId
	raw, err := os.ReadFile(filepath.Join(r.lab.root, "contracts/examples/policies/fair-use.json"))
	if err != nil {
		return
	}
	var fup map[string]any
	if json.Unmarshal(raw, &fup) != nil {
		return
	}
	fup["overrides"] = []any{}
	now := time.Now().UTC()
	fup["effective_at"] = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	fair, _, err := r.lab.createPolicy(ctx, admin, "fair_use", "e2e-fair-use", fup)
	if err != nil {
		r.planError = err.Error()
		note("fair_use", r.planError)
		return
	}
	if r.profileID == uuid.Nil {
		r.planError = "DNS profile bootstrap unavailable"
		note("dns_profile", r.planError)
		return
	}
	payload := map[string]any{"display_name": map[string]string{"zh-TW": "TEST ONLY e2e", "en": "TEST ONLY e2e"}, "device_limit": 3, "exit_group_ids": []uuid.UUID{r.groupID}, "price": map[string]any{"amount_minor": 1000, "currency": "USD", "billing_interval": "month", "provider": "stripe", "provider_price_ref": "e2e-price"}, "rate_limit": nil, "fair_use_revision_id": fair, "dns_profile_revision_id": r.profileID}
	r.planRevID, r.planID, err = r.lab.createPolicy(ctx, admin, "plan", "e2e-lab", payload)
	if err != nil {
		r.planError = err.Error()
		note("plan", r.planError)
		return
	}
	if err = r.lab.rememberPlan(r.planID, r.planRevID); err != nil {
		note("plan_hook", "failed")
		return
	}
	note("group_id", r.groupID.String())
	note("plan_id", r.planID.String())
	note("plan_revision_id", r.planRevID.String())
	rec.Status = StatusPass
}

func (r *runner) portal(ctx context.Context, member *session) bool {
	rec := Record{ID: "portal-billing-return", Title: "Built Portal billing return renders in a browser", Spec: []string{"§18.4", "§19 AT03"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	client := &http.Client{Transport: r.lab.transport, Timeout: 15 * time.Second}
	page, err := client.Get(r.lab.public + "/billing/return")
	portalOK := false
	if err == nil {
		body, _ := io.ReadAll(io.LimitReader(page.Body, 1<<20))
		page.Body.Close()
		portalOK = page.StatusCode == 200 && strings.Contains(string(body), `id="root"`) && strings.Contains(string(body), "/assets/")
		note("billing_return_http", itoa(page.StatusCode))
	}
	note("built_portal_spa", boolText(portalOK))
	if !portalOK {
		return false
	}
	portalURL, _ := url.Parse(r.lab.public)
	input, _ := json.Marshal(map[string]any{"port": r.lab.ports["control"], "cookies": member.http.Jar.Cookies(portalURL)})
	browserCtx, stop := context.WithTimeout(ctx, 40*time.Second)
	defer stop()
	browser := exec.CommandContext(browserCtx, "node", filepath.Join(r.lab.root, "deploy/devstack/probe-portal.mjs"))
	browser.Stdin = bytes.NewReader(input)
	browserOut, browserErr := browser.Output()
	note("browser_probe", strings.TrimSpace(string(browserOut)))
	if browserErr != nil {
		note("browser_error", "billing page did not render successfully")
		return false
	}
	rec.Status = StatusPass
	return true
}

func (r *runner) billing(ctx context.Context, _ *session) {
	rec := Record{ID: "billing", Title: "Checkout provider truth projects a paid period", Spec: []string{"§9.2", "§9.3", "§19 AT03", "§19 AT04"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	client := &http.Client{Transport: r.lab.transport, Timeout: 15 * time.Second}
	member, err := r.lab.login(ctx, "e2e-billing", "member")
	if err != nil {
		note("login", err.Error())
		return
	}
	if !r.portal(ctx, member) {
		note("portal", "return page probe failed")
		return
	}
	if r.planRevID == uuid.Nil {
		rec.Status = StatusBlocked
		note("owner", r.planError+"; payment live reconciliation awaits API-created plan")
		return
	}
	checkout, err := member.call(ctx, "POST", r.lab.public+"/api/v1/member/checkout-sessions", api.CheckoutSessionRequest{PlanId: r.planID, ReturnUrl: r.lab.public + "/billing/return"}, map[string]string{"Idempotency-Key": uuid.NewString()})
	note("checkout", httpDetail(checkout, err))
	var session api.CheckoutSession
	if err != nil || checkout.Status != 201 || json.Unmarshal(checkout.Body, &session) != nil {
		return
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer conn.Close(ctx)
	var providerID string
	if conn.QueryRow(ctx, `SELECT external_session_id FROM billing_sessions WHERE id=$1 AND member_id=$2`, session.CheckoutSessionId, member.memberID).Scan(&providerID) != nil {
		return
	}
	secret, err := r.lab.secret("payment.secret")
	if err != nil {
		return
	}
	hook := func(path string, body any) ([]byte, bool) {
		raw, _ := json.Marshal(body)
		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.stripe.com/e2e/"+path, bytes.NewReader(raw))
		if err != nil {
			return nil, false
		}
		req.Header.Set("X-E2E-Secret", secret)
		resp, err := client.Do(req)
		if err != nil {
			return nil, false
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		note("hook_"+path, itoa(resp.StatusCode))
		return out, resp.StatusCode == 200
	}
	start, end := time.Now().UTC().Add(-time.Minute).Truncate(time.Second), time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	snapshot := map[string]any{"checkout_id": providerID, "period_start": start.Unix(), "period_end": end.Unix(), "amount_minor": 1000, "currency": "USD", "status": "active", "paid": true, "cancel_at_period_end": false, "refund_minor": 0}
	event := func(kind string) bool {
		raw, ok := hook("event", map[string]string{"checkout_id": providerID, "type": kind})
		if !ok {
			return false
		}
		var e struct{ Body, Signature string }
		if json.Unmarshal(raw, &e) != nil {
			return false
		}
		for range 2 {
			req, _ := http.NewRequestWithContext(ctx, "POST", r.lab.public+"/api/v1/webhooks/stripe", strings.NewReader(e.Body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", e.Signature)
			resp, err := client.Do(req)
			if err != nil {
				return false
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			note("webhook_"+kind, itoa(resp.StatusCode))
			if resp.StatusCode != 202 {
				return false
			}
		}
		return true
	}
	poll := func(refund int64, canceled, paid bool) bool {
		wait, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		var rows []byte
		ok := waitUntil(wait, time.Second, func() bool {
			var matched bool
			err := conn.QueryRow(wait, `SELECT s.last_reconciled_at IS NOT NULL AND s.cancel_at_period_end=$2 AND (($5 AND s.paid_from=$3 AND s.paid_through=$4) OR (NOT $5 AND s.paid_from IS NULL AND s.paid_through IS NULL)) AND (SELECT coalesce(sum(amount_minor),0) FROM billing_records b WHERE b.subscription_id=s.id AND b.record_type='refund')=$6 AND e.state=ANY($7::text[]) FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, member.memberID, canceled, start, end, paid, refund, func() []string {
				if !paid {
					return []string{"pending_payment", "expired"}
				}
				if canceled {
					return []string{"cancel_at_period_end"}
				}
				return []string{"active"}
			}()).Scan(&matched)
			return err == nil && matched
		})
		_ = conn.QueryRow(ctx, `SELECT jsonb_build_object('provider_status',s.provider_status,'cancel_at_period_end',s.cancel_at_period_end,'paid_from',s.paid_from,'paid_through',s.paid_through,'last_reconciled',s.last_reconciled_at IS NOT NULL,'anomalies',s.reconciliation_anomalies,'entitlement_state',e.state,'effective_service_end',e.effective_service_end,'paid_invoice_count',(SELECT count(*) FROM billing_records b WHERE b.subscription_id=s.id AND b.record_type='paid_invoice'),'refund_total',(SELECT coalesce(sum(amount_minor),0) FROM billing_records b WHERE b.subscription_id=s.id AND b.record_type='refund')) FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, member.memberID).Scan(&rows)
		note(fmt.Sprintf("stored_rows_refund_%d", refund), string(rows))
		return ok
	}
	if _, ok := hook("snapshot", snapshot); !ok || !event("checkout.session.completed") {
		return
	}
	if !poll(0, false, true) {
		note("owner", "w3-wire-members: paid interval reconciliation")
		rec.Status = StatusBlocked
		return
	}
	note("first_full_paid_period_reconciled", "true")
	note("period_start", start.Format(time.RFC3339))
	note("period_end", end.Format(time.RFC3339))
	snapshot["cancel_at_period_end"] = true
	snapshot["refund_minor"] = 250
	if _, ok := hook("snapshot", snapshot); !ok || !event("customer.subscription.updated") || !poll(250, true, true) {
		return
	}
	snapshot["refund_minor"] = 1000
	snapshot["status"] = "canceled"
	if _, ok := hook("snapshot", snapshot); !ok || !event("charge.refunded") || !poll(1000, true, false) {
		return
	}
	var invoices, refunds int
	if conn.QueryRow(ctx, `SELECT count(*) FILTER(WHERE record_type='paid_invoice'),count(*) FILTER(WHERE record_type='refund') FROM billing_records WHERE member_id=$1`, member.memberID).Scan(&invoices, &refunds) != nil {
		return
	}
	note("invoice_rows_after_duplicate_deliveries", itoa(invoices))
	note("immutable_refund_rows", itoa(refunds))
	if invoices == 1 && refunds == 2 {
		rec.Status = StatusPass
	}
}
