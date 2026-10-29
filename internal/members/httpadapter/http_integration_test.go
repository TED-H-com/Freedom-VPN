//go:build integration

package httpadapter_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (f *fixture) checkout(s browserSession) (uuid.UUID, string) {
	f.t.Helper()
	key := uuid.NewString()
	body := fmt.Sprintf(`{"plan_id":%q,"return_url":"https://portal.example.invalid/billing/return"}`, f.plan)
	w := f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
	requireStatus(f.t, w, 201)
	first := decode[api.CheckoutSession](f.t, w)
	if first.CheckoutUrl == "" {
		f.t.Fatal("missing synchronous checkout URL")
	}
	w = f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
	requireStatus(f.t, w, 201)
	session := decode[api.CheckoutSession](f.t, w)
	if session.CheckoutUrl == "" || session != first {
		f.t.Fatal("missing hosted checkout")
	}
	return session.CheckoutSessionId, key
}
func (f *fixture) pay(s browserSession, checkout uuid.UUID, event string, end time.Time) {
	f.t.Helper()
	sub := billing.Subscription{ID: "TEST-sub-" + s.id.String(), CustomerID: "TEST-customer-" + s.id.String(), Status: "active", PriceID: "TEST-ONLY-price", CheckoutID: checkout}
	f.payment.SetState(sub, []billing.Invoice{{ID: "TEST-invoice-" + s.id.String(), SubscriptionID: sub.ID, CustomerID: sub.CustomerID, Status: "paid", AmountMinor: 100, Currency: "USD", Period: entitlement.Period{Start: f.now.Add(-time.Hour), End: end}}})
	// Whitespace is deliberately significant to the raw-byte signature.
	raw := fmt.Sprintf("{ \n \"id\": %q, \"type\":\"invoice.paid\", \"subscription_id\":%q }", event, sub.ID)
	w := f.call(browserSession{}, "POST", "/api/v1/webhooks/fake", raw, map[string]string{"Stripe-Signature": f.payment.Sign([]byte(raw)), "Idempotency-Key": ""})
	requireStatus(f.t, w, 202)
	f.drain()
}
func (f *fixture) authorize(s browserSession, peer, ip string) uuid.UUID {
	f.t.Helper()
	f.bind(s, "TEST-user-"+s.id.String(), peer, ip)
	if e := f.services.Adapter.Discover(f.t.Context(), f.services.HTTP.Devices); e != nil {
		f.t.Fatal(e)
	}
	f.drain()
	var id uuid.UUID
	if e := f.pool.QueryRow(f.t.Context(), `SELECT id FROM member_devices WHERE member_id=$1 AND peer_id=$2`, s.id, peer).Scan(&id); e != nil {
		f.t.Fatal(e)
	}
	return id
}
func TestAT01_AT03_LoginCheckoutWebhookDeviceAuthorizationHTTP(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-member", "member")
	w := f.call(member, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 200)
	self := decode[api.MemberSelf](t, w)
	if self.Entitlement.State != "pending_payment" || !self.Entitlement.EffectiveServiceEnd.IsNull() {
		t.Fatal("first login granted service")
	}
	checkout, key := f.checkout(member)
	w = f.call(member, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 200)
	if decode[api.MemberSelf](t, w).Entitlement.State != "pending_payment" {
		t.Fatal("checkout granted service")
	}
	w = f.call(member, "POST", "/api/v1/member/checkout-sessions", fmt.Sprintf(`{"plan_id":%q,"return_url":"https://evil.example.invalid/return"}`, f.plan), nil)
	requireStatus(t, w, 422)
	w = f.call(member, "POST", "/api/v1/member/checkout-sessions", fmt.Sprintf(`{"plan_id":%q,"return_url":"https://portal.example.invalid/billing/return"}`, uuid.New()), map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 409)
	f.pay(member, checkout, "TEST-event-paid", f.now.Add(24*time.Hour))
	w = f.call(member, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 200)
	self = decode[api.MemberSelf](t, w)
	if self.Entitlement.State != "active" {
		t.Fatal(self.Entitlement.State)
	}
	device := f.authorize(member, "TEST-peer", "100.64.0.11")
	w = f.call(member, "GET", "/api/v1/member/devices", "", nil)
	requireStatus(t, w, 200)
	list := decode[api.MemberDeviceList](t, w)
	if list.AuthorizedCount != 1 || len(list.Items) != 1 || list.Items[0].DeviceId != device || list.Items[0].Status != "authorized" {
		t.Fatal(list)
	}
	var opID uuid.UUID
	if e := f.pool.QueryRow(t.Context(), `SELECT id FROM operations WHERE resource_id=$1`, device).Scan(&opID); e != nil {
		t.Fatal(e)
	}
	w = f.call(member, "GET", "/api/v1/member/operations/"+opID.String(), "", nil)
	requireStatus(t, w, 200)
	op := decode[api.Operation](t, w)
	if op.Status != "running" {
		t.Fatal("node ACK was fabricated", op.Status)
	}
	for _, step := range op.Steps {
		if step.Name == "netbird" && (step.Status != "succeeded" || step.ObservedAt.IsNull()) {
			t.Fatal("missing NetBird evidence")
		}
		if step.Name == "nodes" && step.Status != "pending" {
			t.Fatal("node ACK fabricated")
		}
	}
	f.netbird.mu.Lock()
	foreign := f.netbird.objects["groups"]["TEST-foreign"]["peers"]
	f.netbird.mu.Unlock()
	b, _ := json.Marshal(foreign)
	if string(b) != `["TEST-peer"]` {
		t.Fatal("unmanaged membership changed")
	}
	if f.count(`SELECT count(*) FROM outbox_events WHERE attempt<>1`) != 0 {
		t.Fatal("normal flow required a retry")
	}
	if f.count(`SELECT count(*) FROM idempotency_requests WHERE state<>'completed' OR response_status IS NULL`) != 0 {
		t.Fatal("incomplete command reservation")
	}
	for _, secret := range []string{"Stripe-Signature", "synthetic token", "access_token", "payments.example.invalid"} {
		if strings.Contains(f.logs.String(), secret) {
			t.Fatal("secret-bearing log field", secret)
		}
	}
}
func TestAT03_WebhookRawBytesReplaySignatureFailureAndPortal(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-member", "member")
	checkout, _ := f.checkout(member)
	f.pay(member, checkout, "TEST-event-paid", f.now.Add(time.Hour))
	raw := fmt.Sprintf("{ \n \"id\": %q, \"type\":\"invoice.paid\", \"subscription_id\":%q }", "TEST-event-paid", "TEST-sub-"+member.id.String())
	for range 10 {
		w := f.call(browserSession{}, "POST", "/api/v1/webhooks/fake", raw, map[string]string{"Stripe-Signature": f.payment.Sign([]byte(raw)), "Idempotency-Key": ""})
		requireStatus(t, w, 202)
		if !decode[api.WebhookAck](t, w).Duplicate {
			t.Fatal("replay not deduplicated")
		}
	}
	for _, signature := range []string{"", "TEST-invalid", f.payment.Sign([]byte(strings.ReplaceAll(raw, " ", "")))} {
		w := f.call(browserSession{}, "POST", "/api/v1/webhooks/fake", raw, map[string]string{"Stripe-Signature": signature, "Idempotency-Key": ""})
		requireStatus(t, w, 401)
		if !strings.Contains(w.Body.String(), "WEBHOOK_SIGNATURE_INVALID") {
			t.Fatal(w.Body.String())
		}
	}
	if f.count(`SELECT count(*) FROM inbox_events`) != 1 || f.count(`SELECT count(*) FROM billing_records`) != 1 {
		t.Fatal("duplicate billing evidence")
	}
	f.probe.failPortal = true
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	body := `{"return_url":"https://portal.example.invalid/billing/return"}`
	w := f.call(member, "POST", "/api/v1/member/billing-portal-sessions", body, headers)
	requireStatus(t, w, 503)
	if f.count(`SELECT count(*) FROM billing_sessions WHERE kind='portal'`) != 1 {
		t.Fatal("portal intent was not committed before I/O")
	}
	f.probe.failPortal = false
	w = f.call(member, "POST", "/api/v1/member/billing-portal-sessions", body, headers)
	requireStatus(t, w, 201)
	portal := decode[api.BillingPortalSession](t, w)
	w = f.call(member, "POST", "/api/v1/member/billing-portal-sessions", body, headers)
	requireStatus(t, w, 201)
	if decode[api.BillingPortalSession](t, w).PortalUrl != portal.PortalUrl || f.probe.calls != 2 {
		t.Fatal("portal replay created another provider session")
	}
	f.drain()
}
func TestAT02_DeviceLimitRevokeOwnershipReadbackHTTP(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-member", "member")
	other := f.login("TEST-other", "member")
	checkout, _ := f.checkout(member)
	f.pay(member, checkout, "TEST-paid", f.now.Add(time.Hour))
	first := f.authorize(member, "TEST-peer", "100.64.0.11")
	f.authorize(member, "TEST-peer-2", "100.64.0.12")
	f.bind(member, "TEST-user-"+member.id.String(), "TEST-peer-3", "100.64.0.13")
	f.bind(member, "TEST-user-"+member.id.String(), "TEST-peer-4", "100.64.0.14")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, peer := range []string{"TEST-peer-3", "TEST-peer-4"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := f.services.HTTP.Devices.ObserveAndAuthorize(t.Context(), entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}, "TEST-ONLY", peer)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	successes, failures := 0, 0
	for e := range results {
		if e == nil {
			successes++
		} else if e.Error() == "device limit reached" {
			failures++
		} else {
			t.Fatal(e)
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatal(successes, failures)
	}
	f.drain()
	path := "/api/v1/member/devices/" + first.String()
	requireStatus(t, f.call(other, "DELETE", path, "", nil), 404)
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	w := f.call(member, "DELETE", path, "", headers)
	requireStatus(t, w, 202)
	op := decode[api.Operation](t, w)
	if op.Status != "pending" {
		t.Fatal("revocation completed before observation")
	}
	f.netbird.mu.Lock()
	f.netbird.stale = true
	f.netbird.mu.Unlock()
	if e := f.services.HTTP.Devices.SyncDevice(t.Context(), first); e == nil {
		t.Fatal("stale NetBird read-back completed revocation")
	}
	w = f.call(member, "GET", "/api/v1/member/operations/"+op.OperationId.String(), "", nil)
	requireStatus(t, w, 200)
	if failure := decode[api.Operation](t, w); failure.Status != api.OperationStatusFailed || failure.Error.IsNull() {
		t.Fatal("failed readback did not expose operation failure")
	}
	f.netbird.mu.Lock()
	f.netbird.stale = false
	f.netbird.mu.Unlock()
	f.drain()
	w = f.call(member, "DELETE", path, "", headers)
	requireStatus(t, w, 202)
	if decode[api.Operation](t, w).OperationId != op.OperationId {
		t.Fatal("revocation replay changed operation")
	}
	w = f.call(member, "DELETE", path, "", nil)
	requireStatus(t, w, 202)
	if decode[api.Operation](t, w).OperationId != op.OperationId {
		t.Fatal("new key duplicated terminal revoke")
	}
	requireStatus(t, f.call(other, "GET", "/api/v1/member/operations/"+op.OperationId.String(), "", nil), 404)
	if e := f.services.Adapter.Discover(t.Context(), f.services.HTTP.Devices); e != nil {
		t.Fatal(e)
	}
	f.drain()
	if f.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, member.id) != 3 {
		t.Fatal("revocation did not free slot")
	}
}
func TestAT04_SuspensionGrantGraceAndHTTPPreconditions(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-member", "member")
	admin := f.login("TEST-admin", "admin")
	checkout, _ := f.checkout(member)
	f.pay(member, checkout, "TEST-paid", f.now.Add(time.Hour))
	device := f.authorize(member, "TEST-peer", "100.64.0.11")
	path := "/api/v1/admin/members/" + member.id.String()
	w := f.call(admin, "GET", path, "", nil)
	requireStatus(t, w, 200)
	etag := w.Header().Get("ETag")
	body := `{"admin_suspended":true,"suspension_reason":"TEST ONLY suspension"}`
	requireStatus(t, f.call(admin, "PATCH", path, body, map[string]string{"If-Match": etag, "X-CSRF-Token": "bad"}), 403)
	requireStatus(t, f.call(admin, "PATCH", path, body, map[string]string{"If-Match": etag, "Idempotency-Key": ""}), 400)
	requireStatus(t, f.call(admin, "PATCH", path, body, nil), 428)
	requireStatus(t, f.call(admin, "PATCH", path, body, map[string]string{"If-Match": `"999"`}), 412)
	requireStatus(t, f.call(admin, "PATCH", path, `{"admin_suspended":true,"suspension_reason":"TEST ONLY","unknown":true}`, map[string]string{"If-Match": etag}), 422)
	key := uuid.NewString()
	w = f.call(admin, "PATCH", path, body, map[string]string{"If-Match": etag, "Idempotency-Key": key})
	requireStatus(t, w, 202)
	op := decode[api.Operation](t, w)
	w = f.call(admin, "PATCH", path, body, map[string]string{"If-Match": etag, "Idempotency-Key": key})
	requireStatus(t, w, 202)
	if decode[api.Operation](t, w).OperationId != op.OperationId {
		t.Fatal("suspension replay")
	}
	f.drain()
	f.pay(member, checkout, "TEST-paid-after-suspension", f.now.Add(time.Hour))
	w = f.call(member, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 200)
	if decode[api.MemberSelf](t, w).Entitlement.State != "suspended" {
		t.Fatal("payment lifted suspension")
	}
	observed, e := f.services.Adapter.ObservePeer(t.Context(), "TEST-ONLY", "TEST-peer")
	if e != nil || observed.Authorized {
		t.Fatal("suspended peer retained access", e)
	}
	w = f.call(admin, "GET", path, "", nil)
	requireStatus(t, w, 200)
	etag = w.Header().Get("ETag")
	requireStatus(t, f.call(admin, "PATCH", path, `{"admin_suspended":false}`, map[string]string{"If-Match": etag}), 202)
	f.drain()
	sub := billing.Subscription{ID: "TEST-sub-" + member.id.String(), CustomerID: "TEST-customer-" + member.id.String(), PriceID: "TEST-ONLY-price", CheckoutID: checkout, Status: "past_due"}
	f.payment.SetState(sub, []billing.Invoice{{ID: "TEST-invoice-" + member.id.String(), SubscriptionID: sub.ID, CustomerID: sub.CustomerID, Status: "paid", AmountMinor: 100, Currency: "USD", Period: entitlement.Period{Start: f.now.Add(-time.Hour), End: f.now.Add(time.Hour)}}})
	f.now = f.now.Add(2 * time.Hour)
	raw, _ := json.Marshal(billing.Event{ID: "TEST-past-due", Type: "invoice.failed", SubscriptionID: sub.ID})
	requireStatus(t, f.call(browserSession{}, "POST", "/api/v1/webhooks/fake", string(raw), map[string]string{"Stripe-Signature": f.payment.Sign(raw), "Idempotency-Key": ""}), 202)
	f.drain()
	w = f.call(member, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 200)
	if decode[api.MemberSelf](t, w).Entitlement.State != "grace" {
		t.Fatal("missing finite grace")
	}
	f.now = f.now.Add(73 * time.Hour)
	if _, e = f.services.HTTP.Members.Entitlements.Sweep(t.Context(), f.pool, 100); e != nil {
		t.Fatal(e)
	}
	f.drain()
	if f.count(`SELECT count(*) FROM device_authorization_observations WHERE device_id=$1 AND observed_authorized`, device) != 0 {
		t.Fatal("grace expiry did not withdraw")
	}
	grant := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY grant"}`, f.revision, f.now.Add(time.Hour).Format(time.RFC3339))
	w = f.call(admin, "POST", path+"/grants", grant, nil)
	requireStatus(t, w, 202)
	f.drain()
	requireStatus(t, f.call(admin, "POST", path+"/grants", fmt.Sprintf(`{"source":"trial","plan_revision_id":%q,"valid_until":"9999-01-01T00:00:00Z","reason":"TEST ONLY"}`, f.revision), nil), 422)
}
func TestAPI01_MemberJobsPreserveUnrelatedOutboxAndDeadLetter(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	if e := db.InTx(ctx, f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		raw, _ := canonjson.Marshal(map[string]any{"inbox_id": uuid.New()})
		if e := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.New(), AggregateType: "node", AggregateVersion: "1", EventType: "TEST.other_area", SchemaVersion: 1, Payload: []byte(`{}`)}); e != nil {
			return e
		}
		return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.New(), AggregateType: "billing_inbox", AggregateVersion: "1", EventType: "billing.inbox_received", SchemaVersion: 999, Payload: raw})
	}); e != nil {
		t.Fatal(e)
	}
	for range 8 {
		_, e := f.dispatcher.Once(ctx)
		if e != nil {
			t.Fatal(e)
		}
		f.exec(`UPDATE outbox_events SET available_at=clock_timestamp() WHERE delivered_at IS NULL AND dead_lettered_at IS NULL`)
	}
	if f.count(`SELECT count(*) FROM outbox_events WHERE delivered_at IS NOT NULL`) != 0 || f.count(`SELECT count(*) FROM outbox_events WHERE dead_lettered_at IS NOT NULL`) != 2 {
		t.Fatal("unknown or malformed work acknowledged")
	}
}

func TestBILL01_WorkerReconciliationRepairsMissedWebhook(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-member", "member")
	checkout, _ := f.checkout(member)
	paidEnd := f.now.Add(time.Hour)
	f.pay(member, checkout, "TEST-initial", paidEnd)
	sub := billing.Subscription{ID: "TEST-sub-" + member.id.String(), CustomerID: "TEST-customer-" + member.id.String(), Status: "active", PriceID: "TEST-ONLY-price", CheckoutID: checkout}
	f.payment.SetState(sub, []billing.Invoice{
		{ID: "TEST-invoice-" + member.id.String(), SubscriptionID: sub.ID, CustomerID: sub.CustomerID, Status: "paid", AmountMinor: 100, Currency: "USD", Period: entitlement.Period{Start: f.now.Add(-time.Hour), End: paidEnd}},
		{ID: "TEST-renewal", SubscriptionID: sub.ID, CustomerID: sub.CustomerID, Status: "paid", AmountMinor: 100, Currency: "USD", Period: entitlement.Period{Start: paidEnd, End: paidEnd.Add(time.Hour)}},
	})
	run := uuid.New()
	for range 2 {
		e := db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			raw, _ := canonjson.Marshal(map[string]any{"run_id": run, "after": uuid.Nil})
			return db.EnqueueOutbox(t.Context(), tx, db.OutboxEvent{AggregateID: run, AggregateType: "member_maintenance", AggregateVersion: "1", EventType: "billing.reconcile", SchemaVersion: 1, Payload: raw})
		})
		if e != nil {
			t.Fatal(e)
		}
		f.drain()
	}
	w := f.call(member, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 200)
	self := decode[api.MemberSelf](t, w)
	end, e := self.Entitlement.PaidThrough.Get()
	if e != nil || !end.Equal(paidEnd.Add(time.Hour)) {
		t.Fatal("missed webhook not repaired", end, e)
	}
	if f.count(`SELECT count(*) FROM billing_records`) != 2 || f.count(`SELECT count(*) FROM inbox_events WHERE event_type='reconciliation'`) != 1 {
		t.Fatal("reconciliation duplicated durable work")
	}
}
