//go:build integration

package httpadapter_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// §9, §14.3, §15.4: exercise all member audit writers through the composed HTTP
// server and its real workers, then read the entire stream through control-core.
// Grant withdrawal uses the revision 14 route, including same-key replay.
func TestAPI01_R3MemberAuditLifecycle(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	member := f.login("TEST-member-audit", "member")
	admin := f.login("TEST-admin-audit", "admin")
	path := "/api/v1/admin/members/" + member.id.String()
	grantBody := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY audit lifecycle"}`, f.revision, f.now.Add(time.Hour).Format(time.RFC3339))
	w := f.call(admin, "POST", path+"/grants", grantBody, nil)
	requireStatus(t, w, 202)
	grantOp := decode[api.Operation](t, w)
	if grantOp.Kind != api.OperationKindMemberGrant || grantOp.Resource.Type != api.ResourceTypeEntitlementGrant {
		t.Fatal(grantOp)
	}
	f.drain()
	device := f.authorize(member, "TEST-audit-peer", "100.64.0.20")
	var authorizeID uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT id FROM operations WHERE resource_type=$1 AND resource_id=$2 AND kind=$3`, api.ResourceTypeMemberDevice, device, api.OperationKindMemberDeviceAuthorize).Scan(&authorizeID); err != nil {
		t.Fatal(err)
	}
	w = f.call(member, "GET", "/api/v1/member/operations/"+authorizeID.String(), "", nil)
	requireStatus(t, w, 200)
	authorize := decode[api.Operation](t, w)
	if authorize.Kind != api.OperationKindMemberDeviceAuthorize || authorize.Resource.Type != api.ResourceTypeMemberDevice {
		t.Fatal(authorize)
	}
	observed := false
	for _, step := range authorize.Steps {
		if step.Name == "netbird" && step.Status == api.OperationStepStatusSucceeded {
			observed = true
		}
	}
	if !observed {
		t.Fatal("authorization missing successful read-back step")
	}

	for _, suspended := range []bool{true, false} {
		w = f.call(admin, "GET", path, "", nil)
		requireStatus(t, w, 200)
		body := fmt.Sprintf(`{"admin_suspended":%t,"suspension_reason":"TEST ONLY audit lifecycle"}`, suspended)
		w = f.call(admin, "PATCH", path, body, map[string]string{"If-Match": w.Header().Get("ETag")})
		requireStatus(t, w, 202)
		want := api.OperationKindMemberUnsuspend
		if suspended {
			want = api.OperationKindMemberSuspend
		}
		if op := decode[api.Operation](t, w); op.Kind != want || op.Resource.Type != api.ResourceTypeMember {
			t.Fatal(op)
		}
		f.drain()
	}
	w = f.call(member, "DELETE", "/api/v1/member/devices/"+device.String(), "", nil)
	requireStatus(t, w, 202)
	if op := decode[api.Operation](t, w); op.Kind != api.OperationKindMemberDeviceRevoke || op.Resource.Type != api.ResourceTypeMemberDevice {
		t.Fatal(op)
	}
	f.drain()

	key := uuid.NewString()
	for range 2 {
		w = f.call(admin, "POST", path+"/grants/"+grantOp.Resource.Id.String()+"/revoke", `{"reason":"TEST ONLY audit lifecycle"}`, map[string]string{"Idempotency-Key": key})
		requireStatus(t, w, 202)
	}
	f.drain()
	var revokeID uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT id FROM operations WHERE resource_type=$1 AND resource_id=$2 AND kind=$3`, api.ResourceTypeEntitlementGrant, grantOp.Resource.Id, api.OperationKindMemberGrantRevoke).Scan(&revokeID); err != nil {
		t.Fatal(err)
	}
	revoke, err := (operations.Store{Pool: f.pool}).Get(ctx, revokeID, auth.Principal{Role: auth.Admin, ID: admin.id})
	if err != nil || revoke.Kind != api.OperationKindMemberGrantRevoke || revoke.Resource.Type != api.ResourceTypeEntitlementGrant {
		t.Fatal(revoke, err)
	}

	checkout, _ := f.checkout(member) // Includes a replay of the synchronous 201.
	f.pay(member, checkout, "TEST-audit-payment", f.now.Add(time.Hour))
	// Portal sessions use the same reservation/completion writer.
	w = f.call(member, "POST", "/api/v1/member/billing-portal-sessions", `{"return_url":"https://portal.example.invalid/billing/return"}`, nil)
	requireStatus(t, w, 201)
	var portal uuid.UUID
	if err = f.pool.QueryRow(ctx, `SELECT id FROM billing_sessions WHERE member_id=$1 AND kind='portal'`, member.id).Scan(&portal); err != nil {
		t.Fatal(err)
	}

	events := readMemberAudit(t, f)
	for _, want := range []struct {
		action   string
		resource api.ResourceType
		id       uuid.UUID
		count    int
	}{
		{"member.register", api.ResourceTypeMember, member.id, 1},
		{"member.grant", api.ResourceTypeEntitlementGrant, grantOp.Resource.Id, 1},
		{"member.grant_revoke", api.ResourceTypeEntitlementGrant, grantOp.Resource.Id, 1},
		{"device.authorize", api.ResourceTypeMemberDevice, device, 1},
		{"device.revoke", api.ResourceTypeMemberDevice, device, 1},
		{"member.patch", api.ResourceTypeMember, member.id, 2},
		{"billing.checkout", api.ResourceTypeCheckoutSession, checkout, 1},
		{"billing.checkout_completed", api.ResourceTypeCheckoutSession, checkout, 1},
		{"billing.portal", api.ResourceTypeCheckoutSession, portal, 1},
		{"billing.portal_completed", api.ResourceTypeCheckoutSession, portal, 1},
		{"billing.reconciled", api.ResourceTypeMember, member.id, 1},
	} {
		count := 0
		for _, event := range events {
			if event.Action == want.action && event.Resource.Id == want.id {
				if event.Resource.Type != want.resource || event.Result != api.AuditEventResultSucceeded {
					t.Fatal(event)
				}
				count++
			}
		}
		if count != want.count {
			t.Errorf("%s: got %d audit events, want %d", want.action, count, want.count)
		}
	}
	for _, action := range []string{"member.entitlement_changed", "netbird.observed", "member.policy_compiled"} {
		found := false
		for _, event := range events {
			found = found || event.Action == action
		}
		if !found {
			t.Errorf("missing worker audit: %s", action)
		}
	}
}

func readMemberAudit(t *testing.T, f *fixture) []api.AuditEvent {
	t.Helper()
	store := audit.Store{Pool: f.pool}
	limit := 3 // Force cursor pagination across all writers.
	params := api.ListAuditEventsParams{Limit: &limit}
	var events []api.AuditEvent
	for {
		page, err := store.List(t.Context(), params)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, page.Items...)
		if page.NextCursor.IsNull() {
			break
		}
		next, err := page.NextCursor.Get()
		if err != nil {
			t.Fatal(err)
		}
		params.Cursor = &next
	}
	if count := f.count(`SELECT count(*) FROM audit_events`); count != len(events) || store.AnomalyCount() != 0 {
		t.Fatalf("stored=%d returned=%d anomalies=%d", count, len(events), store.AnomalyCount())
	}
	for _, event := range events {
		actorID, err := event.Actor.Id.Get()
		if err != nil || actorID == uuid.Nil || event.Resource.Id == uuid.Nil || !event.Actor.Type.Valid() || !event.Resource.Type.Valid() || !event.Result.Valid() {
			t.Fatal(event)
		}
		var resource, actor string
		if err := f.pool.QueryRow(t.Context(), `SELECT resource,actor FROM audit_events WHERE id=$1`, event.AuditEventId).Scan(&resource, &actor); err != nil {
			t.Fatal(err)
		}
		if resource != string(event.Resource.Type)+":"+event.Resource.Id.String() || actor != actorID.String() {
			t.Fatal("noncanonical persisted audit identity", event.AuditEventId)
		}
		raw, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"https://", "TEST-ONLY-price", "TEST-customer-", "Stripe-Signature"} {
			if strings.Contains(string(raw), secret) {
				t.Fatal("provider data leaked into audit", event.AuditEventId)
			}
		}
	}
	t.Logf("audit read-back: %d/%d rows, zero anomalies", len(events), len(events))
	return events
}
