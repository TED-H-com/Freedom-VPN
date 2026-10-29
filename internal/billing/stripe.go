package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/google/uuid"
)

// StripeAPIVersion is pinned to the stable API documented at implementation.
// Configure snapshot webhook destinations to the same version; see docs/billing.
const StripeAPIVersion = "2026-09-30.endive"

type StripeConfig struct {
	APIKeyFile         string
	WebhookSecretFiles []string
	Client             *http.Client
	LiveMode           bool
	PortalSessionTTL   time.Duration
}
type Stripe struct {
	key       string
	secrets   [][]byte
	client    *http.Client
	base      string
	live      bool
	portalTTL time.Duration
	now       func() time.Time
}

func NewStripe(c StripeConfig) (*Stripe, error) {
	read := func(path string) ([]byte, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, ErrProvider
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, 4097))
		if err != nil || len(b) > 4096 || len(strings.TrimSpace(string(b))) == 0 {
			return nil, ErrProvider
		}
		return []byte(strings.TrimSpace(string(b))), nil
	}
	key, err := read(c.APIKeyFile)
	if err != nil {
		return nil, err
	}
	s := &Stripe{key: string(key), base: "https://api.stripe.com", live: c.LiveMode, portalTTL: c.PortalSessionTTL, now: time.Now}
	clear(key)
	if len(c.WebhookSecretFiles) == 0 || c.PortalSessionTTL <= 0 || c.PortalSessionTTL > 5*time.Minute {
		return nil, ErrPayload
	}
	for _, path := range c.WebhookSecretFiles {
		secret, e := read(path)
		if e != nil {
			return nil, e
		}
		s.secrets = append(s.secrets, secret)
	}
	client := http.Client{Timeout: 15 * time.Second}
	if c.Client != nil {
		client = *c.Client
		if client.Timeout <= 0 {
			client.Timeout = 15 * time.Second
		}
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	s.client = &client
	return s, nil
}
func (s *Stripe) Name() string { return "stripe" }
func (s *Stripe) request(ctx context.Context, method, path string, values url.Values, key string, out any) error {
	var body io.Reader
	endpoint := s.base + path
	if method == http.MethodGet {
		if len(values) > 0 {
			endpoint += "?" + values.Encode()
		}
	} else {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return ErrProvider
	}
	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("Stripe-Version", StripeAPIVersion)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return ErrProvider
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ErrProvider
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20+1))
	if err != nil || len(raw) > 4<<20 || json.Unmarshal(raw, out) != nil {
		return ErrPayload
	}
	return nil
}

// VerifyWebhook implements Stripe's raw-body HMAC protocol, which is distinct
// from the platform's Ed25519 pkg/envelope protocol (§9.3, source S19).
func (s *Stripe) VerifyWebhook(raw []byte, header string, now time.Time) error {
	if len(header) > 8192 {
		return ErrSignature
	}
	var timestamp string
	var signatures [][]byte
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			return ErrSignature
		}
		switch key {
		case "t":
			if timestamp != "" {
				return ErrSignature
			}
			timestamp = value
		case "v1":
			b, err := hex.DecodeString(value)
			if err != nil || len(b) != sha256.Size {
				return ErrSignature
			}
			signatures = append(signatures, b)
		}
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || len(signatures) == 0 {
		return ErrSignature
	}
	signedAt := time.Unix(seconds, 0)
	delta := now.Sub(signedAt)
	if delta > 5*time.Minute || delta < -5*time.Minute {
		return ErrSignature
	}
	valid := false
	for _, secret := range s.secrets {
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(timestamp + "."))
		mac.Write(raw)
		expected := mac.Sum(nil)
		for _, sig := range signatures {
			valid = hmac.Equal(expected, sig) || valid
		}
	}
	if !valid {
		return ErrSignature
	}
	return nil
}
func (s *Stripe) ParseEvent(raw []byte) (Event, error) {
	var wire struct {
		ID, Type   string
		APIVersion string `json:"api_version"`
		LiveMode   bool   `json:"livemode"`
		Data       struct{ Object json.RawMessage }
	}
	var out Event
	if json.Unmarshal(raw, &wire) != nil || !validExternal(wire.ID) || wire.Type == "" || wire.LiveMode != s.live {
		return out, ErrPayload
	}
	out.ID = wire.ID
	out.Type = wire.Type
	relevant := strings.HasPrefix(wire.Type, "customer.subscription.") || strings.HasPrefix(wire.Type, "invoice.") || wire.Type == "checkout.session.completed" || wire.Type == "charge.refunded"
	if !relevant {
		return out, nil
	}
	if wire.APIVersion != StripeAPIVersion {
		return out, ErrPayload
	}
	var object struct {
		ID              string
		Customer        string
		Subscription    string
		ClientReference string `json:"client_reference_id"`
		Parent          struct {
			SubscriptionDetails struct{ Subscription string } `json:"subscription_details"`
		}
		Metadata map[string]string
	}
	if json.Unmarshal(wire.Data.Object, &object) != nil {
		return out, ErrPayload
	}
	out.CustomerID = object.Customer
	switch {
	case strings.HasPrefix(wire.Type, "customer.subscription."):
		out.SubscriptionID = object.ID
	case strings.HasPrefix(wire.Type, "invoice."):
		out.InvoiceID = object.ID
		out.SubscriptionID = object.Parent.SubscriptionDetails.Subscription
	case wire.Type == "checkout.session.completed":
		out.SubscriptionID = object.Subscription
	}
	if id := object.Metadata["fvpn_checkout_id"]; id != "" {
		parsed, e := uuid.Parse(id)
		if e != nil {
			return out, ErrPayload
		}
		out.CheckoutID = parsed
	}
	return out, nil
}
func (s *Stripe) CreateCheckout(ctx context.Context, r CheckoutRequest) (HostedSession, error) {
	var wire struct {
		ID, URL   string
		ExpiresAt int64 `json:"expires_at"`
	}
	values := url.Values{"mode": {"subscription"}, "success_url": {r.ReturnURL}, "cancel_url": {r.ReturnURL}, "line_items[0][price]": {r.PriceID}, "line_items[0][quantity]": {"1"}, "client_reference_id": {r.ID.String()}, "subscription_data[metadata][fvpn_checkout_id]": {r.ID.String()}, "metadata[fvpn_checkout_id]": {r.ID.String()}, "expires_at": {strconv.FormatInt(r.ExpiresAt.Unix(), 10)}}
	if r.CustomerID != "" {
		values.Set("customer", r.CustomerID)
	}
	if err := s.request(ctx, http.MethodPost, "/v1/checkout/sessions", values, r.ID.String(), &wire); err != nil {
		return HostedSession{}, err
	}
	// Read back the stable external session before declaring the operation done.
	if err := s.request(ctx, http.MethodGet, "/v1/checkout/sessions/"+url.PathEscape(wire.ID), nil, "", &wire); err != nil {
		return HostedSession{}, err
	}
	return HostedSession{ID: wire.ID, URL: wire.URL, ExpiresAt: time.Unix(wire.ExpiresAt, 0).UTC()}, nil
}
func (s *Stripe) CreatePortal(ctx context.Context, r PortalRequest) (HostedSession, error) {
	var wire struct {
		ID, URL string
		Created int64
	}
	if r.CustomerID == "" {
		return HostedSession{}, ErrPayload
	}
	if err := s.request(ctx, http.MethodPost, "/v1/billing_portal/sessions", url.Values{"customer": {r.CustomerID}, "return_url": {r.ReturnURL}}, r.ID.String(), &wire); err != nil {
		return HostedSession{}, err
	}
	// Portal sessions have no retrieval endpoint or exposed expiry. The response
	// is authoritative evidence; the configured local TTL caps URL presentation.
	created := time.Unix(wire.Created, 0).UTC()
	if wire.Created <= 0 {
		created = s.now().UTC()
	}
	return HostedSession{ID: wire.ID, URL: wire.URL, ExpiresAt: created.Add(s.portalTTL)}, nil
}
func (s *Stripe) FetchSubscription(ctx context.Context, id string) (Subscription, error) {
	var wire struct {
		ID, Customer, Status string
		Cancel               bool `json:"cancel_at_period_end"`
		Live                 bool `json:"livemode"`
		Metadata             map[string]string
		Items                struct {
			HasMore bool `json:"has_more"`
			Data    []struct {
				Price    struct{ ID string }
				Quantity int
			}
		}
	}
	if err := s.request(ctx, http.MethodGet, "/v1/subscriptions/"+url.PathEscape(id), nil, "", &wire); err != nil {
		return Subscription{}, err
	}
	if wire.ID != id || wire.Live != s.live || wire.Items.HasMore || len(wire.Items.Data) != 1 || wire.Items.Data[0].Quantity != 1 {
		return Subscription{}, ErrAnomaly
	}
	out := Subscription{ID: wire.ID, CustomerID: wire.Customer, Status: wire.Status, PriceID: wire.Items.Data[0].Price.ID, CancelAtPeriodEnd: wire.Cancel}
	if ref := wire.Metadata["fvpn_checkout_id"]; ref != "" {
		var err error
		out.CheckoutID, err = uuid.Parse(ref)
		if err != nil {
			return out, ErrAnomaly
		}
	}
	return out, nil
}
func stripeList[T any](ctx context.Context, s *Stripe, path string, values url.Values) ([]T, error) {
	if values == nil {
		values = url.Values{}
	}
	values.Set("limit", "100")
	var out []T
	seen := map[string]bool{}
	for range 100 {
		var page struct {
			Data    []json.RawMessage
			HasMore bool `json:"has_more"`
		}
		if err := s.request(ctx, http.MethodGet, path, values, "", &page); err != nil {
			return nil, err
		}
		last := ""
		for _, raw := range page.Data {
			var id struct{ ID string }
			if json.Unmarshal(raw, &id) != nil || !validExternal(id.ID) || seen[id.ID] {
				return nil, ErrPayload
			}
			seen[id.ID] = true
			last = id.ID
			var value T
			if json.Unmarshal(raw, &value) != nil {
				return nil, ErrPayload
			}
			out = append(out, value)
		}
		if !page.HasMore {
			return out, nil
		}
		if last == "" {
			return nil, ErrPayload
		}
		values.Set("starting_after", last)
	}
	return nil, ErrAnomaly
}

type stripeInvoice struct {
	ID, Customer, Status, Currency string
	AmountPaid                     int64 `json:"amount_paid"`
	Parent                         struct {
		SubscriptionDetails struct{ Subscription string } `json:"subscription_details"`
	}
}
type stripeLine struct {
	ID       string
	Amount   int64
	Quantity int
	Period   struct{ Start, End int64 }
	Pricing  struct {
		PriceDetails struct{ Price string } `json:"price_details"`
	}
	Parent struct {
		Type                    string
		SubscriptionItemDetails struct {
			Subscription string
			Proration    bool
		} `json:"subscription_item_details"`
	}
}
type stripePayment struct {
	ID, Status, Invoice string
	Payment             struct {
		Type, Charge  string
		PaymentIntent string `json:"payment_intent"`
	}
}

func (s *Stripe) FetchInvoices(ctx context.Context, sub Subscription) ([]Invoice, error) {
	invoices, err := stripeList[stripeInvoice](ctx, s, "/v1/invoices", url.Values{"subscription": {sub.ID}, "status": {"paid"}})
	if err != nil {
		return nil, err
	}
	out := []Invoice{}
	for _, wire := range invoices {
		if wire.Parent.SubscriptionDetails.Subscription != sub.ID || wire.Customer != sub.CustomerID || wire.Status != "paid" {
			return nil, ErrAnomaly
		}
		lines, e := stripeList[stripeLine](ctx, s, "/v1/invoices/"+url.PathEscape(wire.ID)+"/lines", nil)
		if e != nil {
			return nil, e
		}
		// v1 supports one recurring plan item. Mixed products/prorations require an
		// explicit allocation policy; reject them instead of inventing paid time.
		if len(lines) != 1 {
			return nil, ErrAnomaly
		}
		line := lines[0]
		if line.Parent.Type != "subscription_item_details" || line.Parent.SubscriptionItemDetails.Subscription != sub.ID || line.Parent.SubscriptionItemDetails.Proration || line.Pricing.PriceDetails.Price != sub.PriceID || line.Quantity != 1 || line.Amount < 0 {
			return nil, ErrAnomaly
		}
		invoice := Invoice{ID: wire.ID, SubscriptionID: sub.ID, CustomerID: sub.CustomerID, Status: wire.Status, AmountMinor: wire.AmountPaid, Currency: strings.ToUpper(wire.Currency), Period: entitlement.Period{Start: time.Unix(line.Period.Start, 0).UTC(), End: time.Unix(line.Period.End, 0).UTC()}, Refunds: []Refund{}}
		payments, e := stripeList[stripePayment](ctx, s, "/v1/invoice_payments", url.Values{"invoice": {wire.ID}, "status": {"paid"}})
		if e != nil {
			return nil, e
		}
		for _, payment := range payments {
			if payment.Invoice != wire.ID || payment.Status != "paid" {
				return nil, ErrAnomaly
			}
			values := url.Values{}
			switch payment.Payment.Type {
			case "payment_intent":
				if payment.Payment.PaymentIntent == "" {
					return nil, ErrAnomaly
				}
				values.Set("payment_intent", payment.Payment.PaymentIntent)
			case "charge":
				if payment.Payment.Charge == "" {
					return nil, ErrAnomaly
				}
				values.Set("charge", payment.Payment.Charge)
			default:
				return nil, ErrAnomaly
			}
			// Shared payments cannot allocate a refund to this invoice safely.
			linked, e := stripeList[stripePayment](ctx, s, "/v1/invoice_payments", url.Values{"payment[type]": {payment.Payment.Type}, "payment[" + payment.Payment.Type + "]": {values.Get(payment.Payment.Type)}, "status": {"paid"}})
			if e != nil {
				return nil, e
			}
			if len(linked) != 1 || linked[0].Invoice != wire.ID {
				return nil, ErrAnomaly
			}
			refunds, e := stripeList[struct {
				ID, Currency, Status string
				Amount               int64
			}](ctx, s, "/v1/refunds", values)
			if e != nil {
				return nil, e
			}
			for _, r := range refunds {
				invoice.Refunds = append(invoice.Refunds, Refund{ID: r.ID, AmountMinor: r.Amount, Currency: strings.ToUpper(r.Currency), Status: r.Status})
			}
		}
		out = append(out, invoice)
	}
	return out, nil
}
