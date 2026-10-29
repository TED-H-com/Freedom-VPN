package billing

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// FakeProvider is a sandbox only. It has no real network or card-processing
// ability. Its webhook secret is random per instance, never a committed key.
type FakeProvider struct {
	mu            sync.Mutex
	secret        []byte
	subscriptions map[string]Subscription
	invoices      map[string][]Invoice
	sessions      map[string]HostedSession
	now           func() time.Time
}

func NewFakeProvider() *FakeProvider {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("sandbox random source unavailable")
	}
	return &FakeProvider{secret: secret, subscriptions: map[string]Subscription{}, invoices: map[string][]Invoice{}, sessions: map[string]HostedSession{}, now: time.Now}
}
func (f *FakeProvider) Name() string { return "fake" }
func (f *FakeProvider) SetState(s Subscription, invoices []Invoice) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subscriptions[s.ID] = s
	f.invoices[s.ID] = cloneInvoices(invoices)
}
func cloneInvoices(in []Invoice) []Invoice {
	out := append([]Invoice{}, in...)
	for i := range out {
		out[i].Refunds = append([]Refund{}, in[i].Refunds...)
	}
	return out
}
func (f *FakeProvider) FetchSubscription(ctx context.Context, id string) (Subscription, error) {
	if err := ctx.Err(); err != nil {
		return Subscription{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.subscriptions[id]
	if !ok {
		return s, ErrProvider
	}
	return s, nil
}
func (f *FakeProvider) FetchInvoices(ctx context.Context, s Subscription) ([]Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneInvoices(f.invoices[s.ID]), nil
}
func (f *FakeProvider) Sign(raw []byte) string {
	h := hmac.New(sha256.New, f.secret)
	h.Write(raw)
	return hex.EncodeToString(h.Sum(nil))
}
func (f *FakeProvider) VerifyWebhook(raw []byte, sig string, _ time.Time) error {
	decoded, err := hex.DecodeString(sig)
	if err != nil {
		return ErrSignature
	}
	expected, _ := hex.DecodeString(f.Sign(raw))
	if !hmac.Equal(expected, decoded) {
		return ErrSignature
	}
	return nil
}
func (f *FakeProvider) ParseEvent(raw []byte) (Event, error) {
	var e Event
	if json.Unmarshal(raw, &e) != nil || e.ID == "" || e.Type == "" {
		return e, ErrPayload
	}
	return e, nil
}
func (f *FakeProvider) session(ctx context.Context, id, kind string, expiry time.Time) (HostedSession, error) {
	if err := ctx.Err(); err != nil {
		return HostedSession{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.sessions[id]; ok {
		return s, nil
	}
	s := HostedSession{ID: "fake-" + id, URL: "https://payments.example.invalid/" + kind + "/" + id, ExpiresAt: expiry}
	f.sessions[id] = s
	return s, nil
}
func (f *FakeProvider) CreateCheckout(ctx context.Context, r CheckoutRequest) (HostedSession, error) {
	return f.session(ctx, r.ID.String(), "checkout", r.ExpiresAt)
}
func (f *FakeProvider) CreatePortal(ctx context.Context, r PortalRequest) (HostedSession, error) {
	if r.CustomerID == "" {
		return HostedSession{}, ErrPayload
	}
	return f.session(ctx, r.ID.String(), "portal", f.now().UTC().Add(5*time.Minute))
}
