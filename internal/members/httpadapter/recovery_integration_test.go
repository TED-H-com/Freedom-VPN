//go:build integration

package httpadapter_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestAPI01_ConcurrentHTTPReplayAndRealSessionRateLimit(t *testing.T) {
	f := newFixture(t)
	s := f.login("TEST-member", "member")
	key := uuid.NewString()
	body := fmt.Sprintf(`{"plan_id":%q,"return_url":"https://portal.example.invalid/billing/return"}`, f.plan)
	var wg sync.WaitGroup
	ids := make(chan uuid.UUID, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := f.call(s, "POST", "/api/v1/member/checkout-sessions", body, map[string]string{"Idempotency-Key": key})
			if w.Code != 202 {
				t.Errorf("concurrent command HTTP %d", w.Code)
				return
			}
			ids <- decode[api.Operation](t, w).OperationId
		}()
	}
	wg.Wait()
	close(ids)
	first := uuid.Nil
	for id := range ids {
		if first == uuid.Nil {
			first = id
		}
		if id != first {
			t.Fatal("duplicate concurrent operation")
		}
	}
	if f.count(`SELECT count(*) FROM billing_sessions`) != 1 || f.count(`SELECT count(*) FROM operations`) != 1 || f.count(`SELECT count(*) FROM idempotency_requests`) != 1 {
		t.Fatal("nested idempotency or duplicate commands")
	}
	c := f.config
	c.BrowserRate = config.Bucket{PerSecond: 0.001, Burst: 1}
	f.compose(c)
	requireStatus(t, f.call(s, "GET", "/api/v1/member/me", "", nil), 200)
	w := f.call(s, "GET", "/api/v1/member/me", "", nil)
	requireStatus(t, w, 429)
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing rate retry hint")
	}
}
func TestMEM03_NetBirdLostCreateIdentityDriftAndPersistentBindings(t *testing.T) {
	f := newFixture(t)
	s := f.login("TEST-member", "member")
	checkout, _ := f.checkout(s)
	f.pay(s, checkout, "TEST-paid", f.now.AddDate(0, 0, 1))
	f.bind(s, "TEST-user-"+s.id.String(), "TEST-peer", "100.64.0.11")
	change, e := f.services.HTTP.Devices.ObserveAndAuthorize(t.Context(), entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}, "TEST-ONLY", "TEST-peer")
	if e != nil {
		t.Fatal(e)
	}
	f.netbird.mu.Lock()
	f.netbird.loseCreate = true
	f.netbird.mu.Unlock()
	if e = f.services.HTTP.Devices.SyncDevice(t.Context(), change.DeviceID); e == nil {
		t.Fatal("lost create response counted as success")
	}
	// A fresh BindingStore instance recovers the same external group by stable
	// name; process-local caches cannot provide this guarantee.
	f.services.Adapter.Bindings = devices.NetBirdBindings{Pool: f.pool}
	if e = f.services.HTTP.Devices.SyncDevice(t.Context(), change.DeviceID); e != nil {
		t.Fatal(e)
	}
	f.netbird.mu.Lock()
	groups, writes := len(f.netbird.objects["groups"]), f.netbird.writes
	f.netbird.mu.Unlock()
	if groups != 2 {
		t.Fatal("duplicate paid group after lost create", groups)
	}
	if e = f.services.HTTP.Devices.SyncDevice(t.Context(), change.DeviceID); e != nil {
		t.Fatal(e)
	}
	f.netbird.mu.Lock()
	if f.netbird.writes != writes {
		t.Error("repeated sync changed upstream state")
	}
	f.netbird.objects["peers"]["TEST-peer"]["user_id"] = "TEST-foreign-user"
	f.netbird.mu.Unlock()
	if e = f.services.HTTP.Devices.SyncDevice(t.Context(), change.DeviceID); e == nil {
		t.Fatal("changed peer owner accepted")
	}
	peer, e := f.services.Adapter.ObservePeer(t.Context(), "TEST-ONLY", "TEST-peer")
	if e != nil || peer.Authorized {
		t.Fatal("identity drift did not withdraw access", e)
	}
	if f.count(`SELECT count(*) FROM device_authorization_observations WHERE device_id=$1 AND observed_authorized`, change.DeviceID) != 0 {
		t.Fatal("identity drift recorded authorization")
	}
}
func TestAPI01_MemberMetadataETagFilteredPagesAndGrantRollback(t *testing.T) {
	f := newFixture(t)
	a := f.login("TEST-admin", "admin")
	s := f.login("TEST-member", "member")
	path := "/api/v1/admin/members/" + s.id.String()
	w := f.call(a, "GET", path, "", nil)
	requireStatus(t, w, 200)
	old := w.Header().Get("ETag")
	key := uuid.NewString()
	w = f.call(a, "PATCH", path, `{"display_name":"TEST ONLY renamed"}`, map[string]string{"If-Match": old, "Idempotency-Key": key})
	requireStatus(t, w, 200)
	if w.Header().Get("ETag") == old {
		t.Fatal("metadata version did not advance")
	}
	w = f.call(a, "PATCH", path, `{"display_name":"TEST ONLY renamed"}`, map[string]string{"If-Match": old, "Idempotency-Key": key})
	requireStatus(t, w, 200)
	w = f.call(a, "GET", "/api/v1/admin/members?limit=1&entitlement_state=pending_payment", "", nil)
	requireStatus(t, w, 200)
	page := decode[api.MemberList](t, w)
	if len(page.Items) != 1 || page.NextCursor.IsNull() {
		t.Fatal("filtered pagination missing")
	}
	cursor, _ := page.NextCursor.Get()
	w = f.call(a, "GET", "/api/v1/admin/members?limit=1&entitlement_state=pending_payment&cursor="+cursor, "", nil)
	requireStatus(t, w, 200)
	next := decode[api.MemberList](t, w)
	if len(next.Items) != 1 || next.Items[0].MemberId == page.Items[0].MemberId {
		t.Fatal("page skipped/duplicated member")
	}
	requireStatus(t, f.call(a, "GET", "/api/v1/admin/members?limit=1&entitlement_state=active&cursor="+cursor, "", nil), 400)
	before := f.count(`SELECT count(*) FROM idempotency_requests`)
	requireStatus(t, f.call(a, "POST", path+"/grants", fmt.Sprintf(`{"source":"trial","plan_revision_id":%q,"valid_until":"9999-01-01T00:00:00Z","reason":"TEST ONLY"}`, f.revision), nil), 422)
	if f.count(`SELECT count(*) FROM idempotency_requests`) != before || f.count(`SELECT count(*) FROM entitlement_grants`) != 0 {
		t.Fatal("failed grant left committed work")
	}
}
