package entitlement

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestAT04_StateTable(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	p := Plan{ID: uuid.New(), DeviceLimit: 3, ExitGroups: []uuid.UUID{uuid.New()}}
	paid := func() Input {
		return Input{Now: now, MemberStatus: "active", Plan: &p, Config: TestConfig(), Subscription: &Subscription{ID: uuid.New(), ProviderStatus: "active", PaidFrom: &start, PaidThrough: &end, ServicePeriods: []Period{{start, end}}}}
	}
	tests := []struct {
		name, state string
		change      func(*Input)
		allow       bool
	}{
		{"active", "active", func(i *Input) {}, true},
		{"pending_payment", "pending_payment", func(i *Input) { i.Subscription = nil }, false},
		{"suspended_then_paid", "suspended", func(i *Input) { i.AdminSuspended = true }, false},
		{"closed", "closed", func(i *Input) { i.MemberStatus = "closed" }, false},
		{"period_end_cancel", "cancel_at_period_end", func(i *Input) { i.Subscription.CancelAtPeriodEnd = true }, true},
		{"cancel_expired", "expired", func(i *Input) { i.Now = end; i.Subscription.CancelAtPeriodEnd = true }, false},
		{"expired", "expired", func(i *Input) { i.Now = end }, false},
		{"past_due_no_period", "past_due", func(i *Input) { i.Subscription.ProviderStatus = "past_due"; i.Subscription.ServicePeriods = nil }, false},
		{"grace", "grace", func(i *Input) {
			i.Now = end
			i.Subscription.ProviderStatus = "past_due"
			i.Subscription.GraceFrom = &end
			until := end.Add(time.Hour)
			i.Subscription.GraceUntil = &until
			i.Subscription.GraceReason = "TEST ONLY"
		}, true},
		{"finite_grant", "active", func(i *Input) {
			i.Subscription = nil
			i.Grants = []Grant{{ID: uuid.New(), Plan: p, Start: start, End: end, Reason: "TEST ONLY", Source: "admin_grant"}}
		}, true},
		{"full_refund", "expired", func(i *Input) { i.Subscription.ServicePeriods = nil }, false},
		{"partial_refund_retains_period", "active", func(i *Input) {}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := paid()
			tc.change(&in)
			r, err := Compute(in)
			if err != nil || r.State != tc.state || r.Allows(in.Now) != tc.allow {
				t.Fatalf("result=%+v err=%v", r, err)
			}
			if in.Subscription != nil && r.PaidThrough != nil && !r.PaidThrough.Equal(*in.Subscription.PaidThrough) {
				t.Fatal("billing fact mutated")
			}
		})
	}
}
func TestENT01_NoGapBridgeOrInfiniteGrant(t *testing.T) {
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	p := Plan{ID: uuid.New(), DeviceLimit: 3, ExitGroups: []uuid.UUID{uuid.New()}}
	in := Input{Now: now, MemberStatus: "active", Plan: &p, Config: TestConfig(), Grants: []Grant{{ID: uuid.New(), Plan: p, Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Reason: "TEST ONLY", Source: "trial"}}}
	r, err := Compute(in)
	if err != nil || r.Allows(now) {
		t.Fatalf("future grant: %+v %v", r, err)
	}
	in.Grants[0].End = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err = Compute(in); err == nil {
		t.Fatal("unbounded grant accepted")
	}
}
