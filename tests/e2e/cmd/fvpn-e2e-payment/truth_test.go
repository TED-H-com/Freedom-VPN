package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/google/uuid"
)

type redirectTransport struct {
	base   http.RoundTripper
	origin *url.URL
}

func (t redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme = t.origin.Scheme
	r.URL.Host = t.origin.Host
	return t.base.RoundTrip(r)
}
func TestBIL01_ProviderTruthCheckoutRefundsAndCancellation(t *testing.T) {
	// TEST ONLY: generated keys and an in-process provider, no Internet.
	key, hook, webhook := uuid.NewString(), uuid.NewString(), uuid.NewString()
	s := &server{apiKey: key, hookKey: hook, webhookKey: webhook, sessions: map[string]session{}, truth: map[string]snapshot{}}
	ts := httptest.NewTLSServer(http.HandlerFunc(s.route))
	defer ts.Close()
	dir := t.TempDir()
	for name, value := range map[string]string{"api": key, "webhook": webhook} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	origin, _ := url.Parse(ts.URL)
	client := ts.Client()
	client.Transport = redirectTransport{client.Transport, origin}
	provider, err := billing.NewStripe(billing.StripeConfig{APIKeyFile: filepath.Join(dir, "api"), WebhookSecretFiles: []string{filepath.Join(dir, "webhook")}, Client: client, PortalSessionTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	checkoutID := uuid.New()
	request := billing.CheckoutRequest{ID: checkoutID, PriceID: "TEST_ONLY_price", ReturnURL: "https://portal.example.invalid/billing/return", ExpiresAt: time.Now().Add(time.Hour)}
	checkout, err := provider.CreateCheckout(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := provider.CreateCheckout(ctx, request)
	if err != nil || replay.ID != checkout.ID {
		t.Fatal("checkout replay changed identity")
	}
	request.PriceID = "TEST_ONLY_other"
	if _, err := provider.CreateCheckout(ctx, request); err == nil {
		t.Fatal("idempotency conflict accepted")
	}
	call := func(path, auth string, body any) (int, []byte) {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", ts.URL+path, bytes.NewReader(raw))
		req.Header.Set("X-E2E-Secret", auth)
		res, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable hook")
		}
		return res.StatusCode, data
	}
	snap := snapshot{CheckoutID: checkout.ID, Start: time.Now().Add(-time.Minute).Unix(), End: time.Now().Add(time.Hour).Unix(), Amount: 1000, Currency: "USD", Status: "active", Paid: true}
	if code, _ := call("/e2e/snapshot", key, snap); code != 401 {
		t.Fatal("provider key authorized fixture mutation")
	}
	for _, refund := range []int64{0, 250, 1000} {
		snap.Refund = refund
		snap.Cancel = refund > 0
		if refund == 1000 {
			snap.Status = "canceled"
		}
		if code, _ := call("/e2e/snapshot", hook, snap); code != 200 {
			t.Fatalf("snapshot HTTP %d", code)
		}
		sub, err := provider.FetchSubscription(ctx, "sub_"+checkout.ID)
		if err != nil || sub.CheckoutID != checkoutID || sub.CancelAtPeriodEnd != snap.Cancel || sub.Status != snap.Status {
			t.Fatalf("subscription readback: %v", err)
		}
		invoices, err := provider.FetchInvoices(ctx, sub)
		if err != nil || len(invoices) != 1 {
			t.Fatalf("invoice readback: %v", err)
		}
		inv := invoices[0]
		if inv.AmountMinor != 1000 || inv.Period.Start.Unix() != snap.Start || inv.Period.End.Unix() != snap.End {
			t.Fatal("invoice identity changed")
		}
		var total int64
		for _, item := range inv.Refunds {
			total += item.AmountMinor
		}
		if total != refund {
			t.Fatal("refund truth mismatch")
		}

	}
	for _, kind := range []string{"checkout.session.completed", "invoice.paid", "invoice.payment_failed", "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted", "charge.refunded"} {
		code, raw := call("/e2e/event", hook, map[string]string{"checkout_id": checkout.ID, "type": kind})
		if code != 200 {
			t.Fatalf("event HTTP %d", code)
		}
		var event struct{ Body, Signature string }
		if json.Unmarshal(raw, &event) != nil {
			t.Fatal("event JSON")
		}
		if provider.VerifyWebhook([]byte(event.Body), event.Signature, time.Now()) != nil {
			t.Fatal("event signature")
		}
		parsed, err := provider.ParseEvent([]byte(event.Body))
		if err != nil || parsed.CheckoutID != checkoutID {
			t.Fatalf("event linking: %v", err)
		}
		if provider.VerifyWebhook([]byte(event.Body+" "), event.Signature, time.Now()) == nil {
			t.Fatal("tampered event accepted")
		}
	}
	snap.Refund = 1
	if code, _ := call("/e2e/snapshot", hook, snap); code != 409 {
		t.Fatal("refund regression accepted")
	}
	snap.CheckoutID = "cs_missing"
	if code, _ := call("/e2e/snapshot", hook, snap); code != 404 {
		t.Fatal("orphan truth accepted")
	}
}
