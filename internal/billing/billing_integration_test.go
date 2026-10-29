//go:build integration

package billing

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fixture struct {
	s                           *Service
	fake                        *FakeProvider
	member, plan, revision, sub uuid.UUID
	now                         time.Time
	subscription                Subscription
	invoice                     Invoice
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	f := fixture{member: uuid.New(), plan: uuid.New(), revision: uuid.New(), sub: uuid.New(), now: now}
	group := uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id) VALUES($1)`, f.member)
	exec(`INSERT INTO member_groups(id,code,name) VALUES($1,'test','TEST ONLY')`, group)
	exec(`INSERT INTO plans(id,code) VALUES($1,'test')`, f.plan)
	exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,provider,provider_price_ref,schema_version,payload) VALUES($1,$2,1,3,$3,'fake','test-price',1,'{}')`, f.revision, f.plan, []uuid.UUID{group})
	exec(`UPDATE plans SET active_revision_id=$2 WHERE id=$1`, f.plan, f.revision)
	exec(`INSERT INTO billing_customers(id,member_id,provider,external_customer_id) VALUES($1,$2,'fake','test-customer')`, uuid.New(), f.member)
	exec(`INSERT INTO subscriptions(id,member_id,provider,external_subscription_id,external_customer_id,plan_revision_id,provider_status,is_active) VALUES($1,$2,'fake','test-subscription','test-customer',$3,'incomplete',true)`, f.sub, f.member, f.revision)
	engine, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	engine.Now = func() time.Time { return now }
	f.fake = NewFakeProvider()
	f.fake.now = func() time.Time { return now }
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		t.Fatal(err)
	}
	vault, err := NewVault("test", map[string][]byte{"test": key})
	if err != nil {
		t.Fatal(err)
	}
	clear(key)
	f.s, err = NewService(pool, f.fake, vault, engine, TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	f.subscription = Subscription{ID: "test-subscription", CustomerID: "test-customer", PriceID: "test-price", Status: "active"}
	f.invoice = Invoice{ID: "test-invoice", SubscriptionID: f.subscription.ID, CustomerID: f.subscription.CustomerID, Status: "paid", AmountMinor: 1000, Currency: "TWD", Period: entitlement.Period{Start: now.Add(-24 * time.Hour), End: now.Add(29 * 24 * time.Hour)}}
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	return f
}
func command(role string, id uuid.UUID) entitlement.Command {
	return entitlement.Command{Actor: entitlement.Actor{ID: id, Role: role, RequestID: uuid.New()}, Key: uuid.New()}
}
func (f fixture) deliver(t *testing.T, event Event) {
	t.Helper()
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Ingest(context.Background(), raw, f.fake.Sign(raw)); err != nil {
		t.Fatal(err)
	}
	worked, err := f.s.ProcessOne(context.Background())
	if err != nil || !worked {
		t.Fatalf("process worked=%v err=%v", worked, err)
	}
}
func (f fixture) projection(t *testing.T) entitlement.Projection {
	t.Helper()
	var p entitlement.Projection
	var raw []byte
	if err := f.s.Pool.QueryRow(context.Background(), `SELECT projection,revision::text FROM entitlements WHERE member_id=$1`, f.member).Scan(&raw, &p.Revision); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &p.Result); err != nil {
		t.Fatal(err)
	}
	return p
}
func (f fixture) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := f.s.Pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func TestAT03_DuplicateTenAndEveryEventOrder(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	event := Event{ID: "test-paid", Type: "invoice.paid", SubscriptionID: f.subscription.ID}
	raw, _ := json.Marshal(event)
	for range 10 {
		if _, err := f.s.Ingest(ctx, raw, f.fake.Sign(raw)); err != nil {
			t.Fatal(err)
		}
	}
	if f.count(t, "inbox_events") != 1 {
		t.Fatal("inbox duplicate")
	}
	if _, err := f.s.ProcessOne(ctx); err != nil {
		t.Fatal(err)
	}
	initial := f.projection(t)
	orders := [][]string{{"created", "paid", "failed"}, {"created", "failed", "paid"}, {"paid", "created", "failed"}, {"paid", "failed", "created"}, {"failed", "created", "paid"}, {"failed", "paid", "created"}}
	for i, order := range orders {
		for j, kind := range order {
			f.deliver(t, Event{ID: fmt.Sprintf("test-%d-%d", i, j), Type: "invoice." + kind, SubscriptionID: f.subscription.ID})
		}
	}
	p := f.projection(t)
	if p.Revision != initial.Revision || p.State != "active" || !p.PaidThrough.Equal(f.invoice.Period.End) || f.count(t, "billing_records") != 1 {
		t.Fatalf("duplicate extension %+v", p)
	}
	if f.count(t, "entitlement_history") != 1 {
		t.Fatal("unchanged projection revised")
	}
	var ciphertext []byte
	if err := f.s.Pool.QueryRow(ctx, `SELECT encrypted_payload FROM inbox_events WHERE external_event_id=$1`, event.ID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, raw) || bytes.Contains(ciphertext, []byte(event.ID)) {
		t.Fatal("plaintext persisted")
	}
	plaintext, err := f.s.Vault.Open(ciphertext, aad("fake", event.ID))
	if err != nil || !bytes.Equal(raw, plaintext) {
		t.Fatal("raw payload lost")
	}
}
func TestAT03_ForgedReturnInvalidSignatureAndCardMinimization(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := command("member", f.member)
	intent, err := f.s.Checkout(ctx, c, f.plan, f.s.Config.ReturnURLs[0])
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.s.Checkout(ctx, c, f.plan, f.s.Config.ReturnURLs[0])
	if err != nil || replay.ID != intent.ID {
		t.Fatal("session duplicated", err)
	}
	if err = f.s.FulfillSession(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	if err = f.s.FulfillSession(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.SessionResult(ctx, c.Actor, intent.ID); err != nil {
		t.Fatal(err)
	}
	if f.count(t, "entitlements") != 0 || f.count(t, "billing_records") != 0 {
		t.Fatal("checkout granted entitlement")
	}
	if _, err = f.s.Checkout(ctx, command("member", f.member), f.plan, "https://portal.example.invalid.evil.invalid/billing/return"); !errors.Is(err, ErrReturnURL) {
		t.Fatal(err)
	}
	if _, err = f.s.Checkout(ctx, command("member", f.member), uuid.New(), f.s.Config.ReturnURLs[0]); !errors.Is(err, db.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = f.s.SessionResult(ctx, command("member", uuid.New()).Actor, intent.ID); !errors.Is(err, db.ErrNotFound) {
		t.Fatal(err)
	}
	raw := []byte(`{"id":"forged","type":"invoice.paid","subscription_id":"test-subscription"}`)
	if _, err = f.s.Ingest(ctx, raw, "invalid"); !errors.Is(err, ErrSignature) {
		t.Fatal(err)
	}
	card := []byte(`{"id":"unsafe","type":"invoice.paid","card":{"number":"TEST ONLY synthetic card data"}}`)
	if _, err = f.s.Ingest(ctx, card, f.fake.Sign(card)); !errors.Is(err, ErrPayload) {
		t.Fatal(err)
	}
	if f.count(t, "inbox_events") != 0 {
		t.Fatal("invalid event persisted")
	}
	portal, err := f.s.Portal(ctx, command("member", f.member), f.s.Config.ReturnURLs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = f.s.FulfillSession(ctx, portal.ID); err != nil {
		t.Fatal(err)
	}
}
func TestAT03_LateFailureCannotOverwriteRenewal(t *testing.T) {
	f := setup(t)
	f.deliver(t, Event{ID: "old-paid", Type: "invoice.paid", SubscriptionID: f.subscription.ID})
	renewal := f.invoice
	renewal.ID = "test-renewal"
	renewal.Period.Start = f.invoice.Period.End
	renewal.Period.End = renewal.Period.Start.Add(30 * 24 * time.Hour)
	f.fake.SetState(f.subscription, []Invoice{renewal, f.invoice})
	f.deliver(t, Event{ID: "renewal-paid", Type: "invoice.paid", SubscriptionID: f.subscription.ID})
	renewed := f.projection(t)
	if !renewed.PaidThrough.Equal(renewal.Period.End) {
		t.Fatal("renewal not projected")
	}
	f.deliver(t, Event{ID: "late-old-failure", Type: "invoice.payment_failed", SubscriptionID: f.subscription.ID})
	p := f.projection(t)
	if p.Revision != renewed.Revision || !p.PaidThrough.Equal(renewal.Period.End) {
		t.Fatal("late failure regressed paid period")
	}
}
func TestAT04_CancelRefundRenewSuspendAndExactlyOneOutbox(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.subscription.CancelAtPeriodEnd = true
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	f.deliver(t, Event{ID: "cancel", Type: "customer.subscription.updated", SubscriptionID: f.subscription.ID})
	if p := f.projection(t); p.State != "cancel_at_period_end" || !p.Allows(f.now) {
		t.Fatal(p)
	}
	f.invoice.Refunds = []Refund{{ID: "partial", AmountMinor: 200, Currency: "TWD", Status: "succeeded"}}
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	f.deliver(t, Event{ID: "partial-event", Type: "charge.refunded", CustomerID: f.subscription.CustomerID})
	if p := f.projection(t); !p.Allows(f.now) || !p.PaidThrough.Equal(f.invoice.Period.End) {
		t.Fatal(p)
	}
	f.invoice.Refunds = append(f.invoice.Refunds, Refund{ID: "remaining", AmountMinor: 800, Currency: "TWD", Status: "succeeded"})
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	f.deliver(t, Event{ID: "full-event", Type: "charge.refunded", CustomerID: f.subscription.CustomerID})
	if p := f.projection(t); p.State != "expired" || p.Allows(f.now) || !p.PaidThrough.Equal(f.invoice.Period.End) {
		t.Fatal(p)
	}
	f.subscription.CancelAtPeriodEnd = false
	renewal := f.invoice
	renewal.ID = "replacement"
	renewal.Refunds = nil
	renewal.Period.End = renewal.Period.End.Add(30 * 24 * time.Hour)
	f.fake.SetState(f.subscription, []Invoice{f.invoice, renewal})
	f.deliver(t, Event{ID: "renewal", Type: "invoice.paid", SubscriptionID: f.subscription.ID})
	m := members.Service{Pool: f.s.Pool, Entitlements: f.s.Entitlements}
	var version decstr.U64
	if err := f.s.Pool.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1`, f.member).Scan(&version); err != nil {
		t.Fatal(err)
	}
	suspended, reason := true, "TEST ONLY administrative suspension"
	if _, err := m.Patch(ctx, command("admin", uuid.New()), f.member, &version, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason}); err != nil {
		t.Fatal(err)
	}
	f.deliver(t, Event{ID: "paid-after-suspension", Type: "invoice.paid", SubscriptionID: f.subscription.ID})
	if p := f.projection(t); p.State != "suspended" || p.Allows(f.now) {
		t.Fatal(p)
	}
	var events, history int
	if err := f.s.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM outbox_events WHERE event_type='member.entitlement_changed'),(SELECT count(*) FROM entitlement_history)`).Scan(&events, &history); err != nil || events != history {
		t.Fatalf("events=%d history=%d err=%v", events, history, err)
	}
}
func TestBILL01_ReconciliationRepairsDroppedWebhookAndGraceExpires(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	run := uuid.New()
	for range 2 {
		if _, count, err := f.s.Reconcile(ctx, run, uuid.Nil, 100); err != nil || count != 1 {
			t.Fatal(count, err)
		}
	}
	if f.count(t, "inbox_events") != 1 {
		t.Fatal("reconciliation retry duplicated")
	}
	if _, err := f.s.ProcessOne(ctx); err != nil {
		t.Fatal(err)
	}
	if p := f.projection(t); p.State != "active" {
		t.Fatal(p)
	}
	end := f.invoice.Period.End
	f.s.Entitlements.Now = func() time.Time { return end }
	f.subscription.Status = "past_due"
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	f.deliver(t, Event{ID: "past-due", Type: "invoice.payment_failed", SubscriptionID: f.subscription.ID})
	p := f.projection(t)
	if p.State != "grace" || p.GraceUntil == nil || !p.PaidThrough.Equal(end) {
		t.Fatal(p)
	}
	expiry := *p.GraceUntil
	f.s.Entitlements.Now = func() time.Time { return expiry }
	if _, err := f.s.Entitlements.Sweep(ctx, f.s.Pool, 100); err != nil {
		t.Fatal(err)
	}
	if p = f.projection(t); p.Allows(expiry) || p.State != "expired" {
		t.Fatal(p)
	}
}
func TestBILL01_FencingAndTransactionalRollback(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	raw, _ := json.Marshal(Event{ID: "event", Type: "paid", SubscriptionID: f.subscription.ID})
	insert, err := f.s.Ingest(ctx, raw, f.fake.Sign(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Pool.Exec(ctx, `UPDATE subscriptions SET sync_token=$2,sync_until=clock_timestamp()+interval '1 minute' WHERE id=$1`, f.sub, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if err = f.s.syncSubscription(ctx, f.sub, insert.ID); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if f.count(t, "billing_records") != 0 {
		t.Fatal("busy sync mutated ledger")
	}
	// Rollback includes history, policy outbox and audit.
	sentinel := errors.New("TEST ONLY rollback")
	err = db.InTx(ctx, f.s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := f.s.Entitlements.Recompute(ctx, tx, f.member); e != nil {
			return e
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if f.count(t, "entitlement_history") != 0 || f.count(t, "outbox_events") != 0 {
		t.Fatal("partial commit")
	}
}

func TestBILL01_NewSubscriptionRequiresServerCheckoutProof(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	member := uuid.New()
	if _, err := f.s.Pool.Exec(ctx, `INSERT INTO members(id) VALUES($1)`, member); err != nil {
		t.Fatal(err)
	}
	sub := Subscription{ID: "new-subscription", CustomerID: "new-customer", PriceID: "test-price", Status: "active"}
	if _, err := f.s.bindSubscription(ctx, sub); !errors.Is(err, ErrAnomaly) {
		t.Fatal("unproved subscription accepted", err)
	}
	intent, err := f.s.Checkout(ctx, command("member", member), f.plan, f.s.Config.ReturnURLs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = f.s.FulfillSession(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	sub.CheckoutID = intent.ID
	invoice := f.invoice
	invoice.ID = "new-invoice"
	invoice.SubscriptionID = sub.ID
	invoice.CustomerID = sub.CustomerID
	f.fake.SetState(sub, []Invoice{invoice})
	f.deliver(t, Event{ID: "new-subscription-event", Type: "invoice.paid", SubscriptionID: sub.ID})
	var state, customer string
	if err = f.s.Pool.QueryRow(ctx, `SELECT e.state,c.external_customer_id FROM entitlements e JOIN billing_customers c ON c.member_id=e.member_id WHERE e.member_id=$1`, member).Scan(&state, &customer); err != nil || state != "active" || customer != sub.CustomerID {
		t.Fatal(state, customer, err)
	}
}

type delayedProvider struct {
	Provider
	mu               sync.Mutex
	first            bool
	started, release chan struct{}
}

func (p *delayedProvider) FetchInvoices(ctx context.Context, s Subscription) ([]Invoice, error) {
	invoices, err := p.Provider.FetchInvoices(ctx, s)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	first := !p.first
	p.first = true
	p.mu.Unlock()
	if first {
		close(p.started)
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return invoices, nil
}
func TestBILL01_StaleProviderReadCannotCommitAfterLeaseReplacement(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, _ := json.Marshal(Event{ID: "race-event", Type: "paid", SubscriptionID: f.subscription.ID})
	event, err := f.s.Ingest(ctx, raw, f.fake.Sign(raw))
	if err != nil {
		t.Fatal(err)
	}
	provider := &delayedProvider{Provider: f.fake, started: make(chan struct{}), release: make(chan struct{})}
	f.s.Provider = provider
	done := make(chan error, 1)
	go func() { done <- f.s.syncSubscription(ctx, f.sub, event.ID) }()
	select {
	case <-provider.started:
	case <-ctx.Done():
		t.Fatal("worker did not reach provider")
	}
	if _, err = f.s.Pool.Exec(ctx, `UPDATE subscriptions SET sync_until=clock_timestamp()-interval '1 second' WHERE id=$1`, f.sub); err != nil {
		t.Fatal(err)
	}
	renewed := f.invoice
	renewed.ID = "race-renewal"
	renewed.Period.End = renewed.Period.End.Add(30 * 24 * time.Hour)
	f.fake.SetState(f.subscription, []Invoice{f.invoice, renewed})
	if err = f.s.syncSubscription(ctx, f.sub, event.ID); err != nil {
		t.Fatal(err)
	}
	p := f.projection(t)
	close(provider.release)
	if err = <-done; !errors.Is(err, db.ErrLeaseLost) {
		t.Fatal("stale writer was not fenced", err)
	}
	after := f.projection(t)
	if after.Revision != p.Revision || !after.PaidThrough.Equal(renewed.Period.End) {
		t.Fatal("stale provider read won")
	}
}
func TestAT04_KeepFullRefundPolicyAndConflictingInvoiceAnomaly(t *testing.T) {
	f := setup(t)
	f.s.Config.FullRefundPolicy = KeepRefundedPeriod
	f.invoice.Refunds = []Refund{{ID: "full", AmountMinor: f.invoice.AmountMinor, Currency: f.invoice.Currency, Status: "succeeded"}}
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	f.deliver(t, Event{ID: "full-kept", Type: "paid", SubscriptionID: f.subscription.ID})
	if p := f.projection(t); !p.Allows(f.now) {
		t.Fatal(p)
	}
	f.invoice.AmountMinor++
	f.fake.SetState(f.subscription, []Invoice{f.invoice})
	raw, _ := json.Marshal(Event{ID: "conflicting-invoice", Type: "paid", SubscriptionID: f.subscription.ID})
	if _, err := f.s.Ingest(context.Background(), raw, f.fake.Sign(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.ProcessOne(context.Background()); !errors.Is(err, ErrAnomaly) {
		t.Fatal(err)
	}
	var anomalies []string
	if err := f.s.Pool.QueryRow(context.Background(), `SELECT reconciliation_anomalies FROM subscriptions WHERE id=$1`, f.sub).Scan(&anomalies); err != nil || len(anomalies) != 1 {
		t.Fatal(anomalies, err)
	}
}
