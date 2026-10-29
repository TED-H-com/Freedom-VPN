// Package billing keeps provider-specific payment facts outside the entitlement
// engine. Hosted checkout and return pages are never evidence of payment (§9.3).
package billing

import (
	"context"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/google/uuid"
	"time"
)

var (
	ErrSignature   = errors.New("invalid webhook signature")
	ErrPayload     = errors.New("invalid provider payload")
	ErrProvider    = errors.New("payment provider unavailable")
	ErrUnsupported = errors.New("unsupported provider event")
	ErrBusy        = errors.New("billing work is already claimed")
	ErrReturnURL   = errors.New("return URL is not allowed")
	ErrAnomaly     = errors.New("billing reconciliation anomaly")
)

type Event struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	CustomerID     string    `json:"customer_id,omitempty"`
	InvoiceID      string    `json:"invoice_id,omitempty"`
	CheckoutID     uuid.UUID `json:"checkout_id,omitempty"`
}
type Subscription struct {
	ID                string    `json:"id"`
	CustomerID        string    `json:"customer_id"`
	Status            string    `json:"status"`
	PriceID           string    `json:"price_id"`
	CancelAtPeriodEnd bool      `json:"cancel_at_period_end"`
	CheckoutID        uuid.UUID `json:"checkout_id"`
}
type Refund struct {
	ID          string `json:"id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}
type Invoice struct {
	ID             string             `json:"id"`
	SubscriptionID string             `json:"subscription_id"`
	CustomerID     string             `json:"customer_id"`
	Status         string             `json:"status"`
	AmountMinor    int64              `json:"amount_minor"`
	Currency       string             `json:"currency"`
	Period         entitlement.Period `json:"period"`
	Refunds        []Refund           `json:"refunds"`
}
type CheckoutRequest struct {
	ID, MemberID, PlanRevisionID   uuid.UUID
	CustomerID, PriceID, ReturnURL string
	ExpiresAt                      time.Time
}
type PortalRequest struct {
	ID                    uuid.UUID
	CustomerID, ReturnURL string
}
type HostedSession struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
type Provider interface {
	Name() string
	CreateCheckout(context.Context, CheckoutRequest) (HostedSession, error)
	CreatePortal(context.Context, PortalRequest) (HostedSession, error)
	FetchSubscription(context.Context, string) (Subscription, error)
	FetchInvoices(context.Context, Subscription) ([]Invoice, error)
	VerifyWebhook([]byte, string, time.Time) error
	ParseEvent([]byte) (Event, error)
}
type RefundPolicy string

const (
	RevokeRefundedPeriod RefundPolicy = "revoke_original_period"
	KeepRefundedPeriod   RefundPolicy = "keep_original_period"
)

type Config struct {
	FullRefundPolicy RefundPolicy
	ReturnURLs       []string
	LeaseDuration    time.Duration
	MaxWebhookBytes  int
}

func TestConfig() Config {
	return Config{FullRefundPolicy: RevokeRefundedPeriod, ReturnURLs: []string{"https://portal.example.invalid/billing/return"}, LeaseDuration: 2 * time.Minute, MaxWebhookBytes: 1 << 20}
}
