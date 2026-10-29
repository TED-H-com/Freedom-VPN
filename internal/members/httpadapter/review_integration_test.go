//go:build integration

package httpadapter_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

type failingCheckout struct {
	billing.Provider
	mu    sync.Mutex
	calls int
	check func(context.Context)
}

func (p *failingCheckout) CreateCheckout(ctx context.Context, r billing.CheckoutRequest) (billing.HostedSession, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	p.check(ctx)
	if p.calls == 1 {
		return billing.HostedSession{}, errors.New("TEST ONLY provider failure must not escape")
	}
	return p.Provider.CreateCheckout(ctx, r)
}
func TestBILL03_D031SynchronousCheckoutFailureAndRetry(t *testing.T) {
	f := newFixture(t)
	s := f.login("TEST-member", "member")
	provider := &failingCheckout{Provider: f.services.HTTP.Billing.Provider}
	provider.check = func(ctx context.Context) {
		// The reservation is durable and the member row lock is free during I/O.
		tx, e := f.services.HTTP.Members.Pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		if _, e = tx.Exec(ctx, `SELECT id FROM members WHERE id=$1 FOR UPDATE NOWAIT`, s.id); e != nil {
			t.Fatal("provider called inside member transaction", e)
		}
		var n int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM billing_sessions WHERE member_id=$1`, s.id).Scan(&n); e != nil || n != 1 {
			t.Fatal("reservation not durable", n, e)
		}
	}
	f.services.HTTP.Billing.Provider = provider
	key := uuid.NewString()
	body := fmt.Sprintf(`{"plan_id":%q,"return_url":"https://portal.example.invalid/billing/return"}`, f.plan)
	w := f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 503)
	var failure struct {
		Error struct {
			Code      string
			Retryable bool
			Message   string
		}
	}
	failure = decode[struct {
		Error struct {
			Code      string
			Retryable bool
			Message   string
		}
	}](t, w)
	if failure.Error.Code != "DEPENDENCY_UNAVAILABLE" || !failure.Error.Retryable {
		t.Fatal(failure)
	}
	var session, op uuid.UUID
	var state, code, message, stepCode string
	p := f.services.HTTP.Members.Pool
	if e := p.QueryRow(t.Context(), `SELECT b.id,b.operation_id,b.fulfillment_state,o.error_code,o.error_message,o.steps->'provider'->>'error_code' FROM billing_sessions b JOIN operations o ON o.id=b.operation_id WHERE b.member_id=$1`, s.id).Scan(&session, &op, &state, &code, &message, &stepCode); e != nil {
		t.Fatal(e)
	}
	if state != "failed" || code != "DEPENDENCY_UNAVAILABLE" || stepCode != code || message != "External dependency is unavailable." {
		t.Fatal(state, code, message, stepCode)
	}
	observed, e := (operations.Store{Pool: p}).Get(t.Context(), op, auth.Principal{Role: auth.Member, ID: s.id})
	if e != nil || observed.Status != api.OperationStatusFailed || observed.Error.IsNull() {
		t.Fatal(observed, e)
	}
	w = f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 201)
	result := decode[api.CheckoutSession](t, w)
	if result.CheckoutSessionId != session || result.CheckoutUrl == "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(result)
	}
	w = f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 201)
	if again := decode[api.CheckoutSession](t, w); again != result || provider.calls != 2 {
		t.Fatal(again, provider.calls)
	}
	if f.count(`SELECT count(*) FROM billing_sessions`) != 1 || f.count(`SELECT count(*) FROM idempotency_requests`) != 1 || f.count(`SELECT count(*) FROM outbox_events WHERE event_type='billing.session_requested'`) != 0 || f.count(`SELECT count(*) FROM entitlements`) != 0 {
		t.Fatal("checkout duplicated work or granted service")
	}
	// D-031 replays retain the originally stored 201 and expiry, even after
	// the hosted link has expired; a new session requires a new command key.
	f.services.HTTP.Billing.Entitlements.Now = func() time.Time { return result.ExpiresAt.Add(time.Hour) }
	w = f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 201)
	if again := decode[api.CheckoutSession](t, w); again != result || provider.calls != 2 {
		t.Fatal(again, provider.calls)
	}
	if f.count(`SELECT count(*) FROM operations WHERE error_code IS NOT NULL OR error_message IS NOT NULL`) != 0 {
		t.Fatal("recovered operation kept stale error")
	}
}
