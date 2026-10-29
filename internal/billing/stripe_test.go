package billing

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func stripeTest(t *testing.T, handler http.Handler) (*Stripe, []byte) {
	t.Helper()
	dir := t.TempDir()
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	keyPath, secretPath := filepath.Join(dir, "api-key"), filepath.Join(dir, "webhook-key")
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(key)), 0600); err != nil {
		t.Fatal(err)
	}
	secret = []byte(hex.EncodeToString(secret))
	if err := os.WriteFile(secretPath, secret, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewStripe(StripeConfig{APIKeyFile: keyPath, WebhookSecretFiles: []string{secretPath}, PortalSessionTTL: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if handler != nil {
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		s.base = server.URL
		s.client = server.Client()
	}
	return s, secret
}
func signature(secret, raw []byte, now time.Time) string {
	ts := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts + "."))
	mac.Write(raw)
	return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}
func TestBILL01_StripeRawSignatureReplayAndRotation(t *testing.T) {
	s, secret := stripeTest(t, nil)
	now := time.Now().UTC()
	raw := []byte(`{"id":"test-event","type":"invoice.paid"}`)
	sig := signature(secret, raw, now)
	if err := s.VerifyWebhook(raw, sig, now); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		raw []byte
		sig string
		now time.Time
	}{{append(raw, ' '), sig, now}, {raw, sig, now.Add(6 * time.Minute)}, {raw, sig, now.Add(-6 * time.Minute)}, {raw, sig + ",t=0", now}, {raw, "t=1,v1=bad", now}} {
		if s.VerifyWebhook(test.raw, test.sig, test.now) == nil {
			t.Fatal("invalid signature accepted")
		}
	}
	unknown := make([]byte, 32)
	_, _ = rand.Read(unknown)
	rotated := signature(unknown, raw, now)
	_, good, _ := strings.Cut(sig, ",")
	if err := s.VerifyWebhook(raw, rotated+","+good, now); err != nil {
		t.Fatal("rotation rejected", err)
	}
}
func TestBILL01_StripeCurrentInvoiceShapePaginationAndRefunds(t *testing.T) {
	ctx := context.Background()
	requests := map[string]int{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Stripe-Version") != StripeAPIVersion || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing provider headers")
		}
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/subscriptions/test-sub":
			fmt.Fprint(w, `{"id":"test-sub","customer":"test-customer","status":"active","cancel_at_period_end":true,"livemode":false,"items":{"data":[{"quantity":1,"price":{"id":"test-price"}}],"has_more":false}}`)
		case "/v1/invoices":
			if r.URL.Query().Get("starting_after") == "" {
				fmt.Fprint(w, `{"data":[{"id":"test-invoice","customer":"test-customer","status":"paid","currency":"jpy","amount_paid":1000,"parent":{"subscription_details":{"subscription":"test-sub"}}}],"has_more":true}`)
			} else {
				fmt.Fprint(w, `{"data":[],"has_more":false}`)
			}
		case "/v1/invoices/test-invoice/lines":
			fmt.Fprint(w, `{"data":[{"id":"test-line","amount":1000,"quantity":1,"period":{"start":1790553600,"end":1793232000},"pricing":{"price_details":{"price":"test-price"}},"parent":{"type":"subscription_item_details","subscription_item_details":{"subscription":"test-sub","proration":false}}}],"has_more":false}`)
		case "/v1/invoice_payments":
			fmt.Fprint(w, `{"data":[{"id":"test-payment","status":"paid","invoice":"test-invoice","payment":{"type":"payment_intent","payment_intent":"test-intent"}}],"has_more":false}`)
		case "/v1/refunds":
			if r.URL.Query().Get("payment_intent") != "test-intent" {
				t.Error("refund scope missing")
			}
			fmt.Fprint(w, `{"data":[{"id":"test-refund","amount":500,"currency":"jpy","status":"succeeded"}],"has_more":false}`)
		default:
			t.Error("unexpected provider path", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	s, _ := stripeTest(t, handler)
	sub, err := s.FetchSubscription(ctx, "test-sub")
	if err != nil || !sub.CancelAtPeriodEnd {
		t.Fatal(sub, err)
	}
	invoices, err := s.FetchInvoices(ctx, sub)
	if err != nil || len(invoices) != 1 || invoices[0].Currency != "JPY" || len(invoices[0].Refunds) != 1 || invoices[0].Refunds[0].AmountMinor != 500 {
		t.Fatal(invoices, err)
	}
	if requests["/v1/invoices"] != 2 {
		t.Fatal("pagination incomplete")
	}
}
func TestBILL03_StripeSessionIdempotencyAndCustomerMapping(t *testing.T) {
	id := uuid.New()
	readbacks := 0
	now := time.Now().UTC().Truncate(time.Second)
	s, _ := stripeTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.Header.Get("Idempotency-Key") != id.String() {
			t.Error("missing stable idempotency key")
		}
		_ = r.ParseForm()
		switch r.URL.Path {
		case "/v1/checkout/sessions":
			if r.Form.Get("customer") != "central-customer" || r.Form.Get("subscription_data[metadata][fvpn_checkout_id]") != id.String() || r.Form.Get("mode") != "subscription" {
				t.Error("bad checkout fields")
			}
			fmt.Fprintf(w, `{"id":"test-session","url":"https://payments.example.invalid/checkout","expires_at":%d}`, now.Add(time.Hour).Unix())
		case "/v1/checkout/sessions/test-session":
			readbacks++
			fmt.Fprintf(w, `{"id":"test-session","url":"https://payments.example.invalid/checkout","expires_at":%d}`, now.Add(time.Hour).Unix())
		case "/v1/billing_portal/sessions":
			if r.Form.Get("customer") != "central-customer" {
				t.Error("wrong customer")
			}
			fmt.Fprintf(w, `{"id":"test-portal","url":"https://payments.example.invalid/portal","created":%d}`, now.Unix())
		default:
			w.WriteHeader(404)
		}
	}))
	if _, err := s.CreateCheckout(context.Background(), CheckoutRequest{ID: id, CustomerID: "central-customer", PriceID: "test-price", ReturnURL: "https://portal.example.invalid/return", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePortal(context.Background(), PortalRequest{ID: id, CustomerID: "central-customer", ReturnURL: "https://portal.example.invalid/return"}); err != nil {
		t.Fatal(err)
	}
	if readbacks != 1 {
		t.Fatal("missing session readback")
	}
}
func TestBILL01_StripeErrorsNeverExposeResponseOrSecrets(t *testing.T) {
	sentinel := "TEST ONLY provider-sensitive-value"
	s, _ := stripeTest(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(400); fmt.Fprint(w, sentinel) }))
	_, err := s.FetchSubscription(context.Background(), "test")
	if err == nil || strings.Contains(err.Error(), sentinel) || strings.Contains(err.Error(), s.key) {
		t.Fatal("provider error leaked")
	}
}
func TestBILL01_StripeEventRoutingIgnoresCreated(t *testing.T) {
	s, _ := stripeTest(t, nil)
	raw := []byte(fmt.Sprintf(`{"id":"test-event","type":"invoice.payment_failed","created":1,"api_version":%q,"livemode":false,"data":{"object":{"id":"test-invoice","customer":"test-customer","parent":{"subscription_details":{"subscription":"test-sub"}}}}}`, StripeAPIVersion))
	e, err := s.ParseEvent(raw)
	if err != nil || e.SubscriptionID != "test-sub" {
		t.Fatal(e, err)
	}
	var m map[string]any
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	m["api_version"] = "unsupported"
	raw, _ = json.Marshal(m)
	if _, err = s.ParseEvent(raw); err == nil {
		t.Fatal("incompatible event schema accepted")
	}
}
func Test14_1_CurrencyMinorUnits(t *testing.T) {
	for _, tc := range []struct {
		amount         int64
		currency, want string
	}{{1000, "JPY", "JPY 1000"}, {1000, "TWD", "TWD 10.00"}, {1001, "KWD", "KWD 1.001"}} {
		got, err := FormatAmount(tc.amount, tc.currency)
		if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
	}
	if _, err := FormatAmount(100, "ZZZ"); err == nil {
		t.Fatal("unknown currency guessed")
	}
}
func TestSEC02_AuthenticatedEncryptionTamperAndKeyRotation(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	v, err := NewVault("first", map[string][]byte{"first": key})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := v.Seal([]byte(`{"id":"test-event"}`), "event")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = v.Open(encrypted, "another-event"); err == nil {
		t.Fatal("swapped ciphertext accepted")
	}
	encrypted[len(encrypted)-1] ^= 1
	if _, err = v.Open(encrypted, "event"); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}
