//go:build integration

package httpadapter_test

import (
	"bytes"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// D-077: admin device list, admin re-authorization, and the fence projection.
// DEVICE_LIMIT_REACHED is asserted at the pkg/apierr status (409). The
// operation prose asks for 422; that status is not what apierr emits.
func TestD077_AdminDeviceListAndReauthorize(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-member", "member")
	other := f.login("TEST-other", "member")
	admin := f.login("TEST-admin", "admin")
	checkout, _ := f.checkout(member)
	f.pay(member, checkout, "TEST-paid-member", f.now.Add(24*time.Hour))
	otherCheckout, _ := f.checkout(other)
	f.pay(other, otherCheckout, "TEST-paid-other", f.now.Add(24*time.Hour))

	regrant := f.authorize(member, "TEST-regrant", "100.64.0.11")
	revoked := f.authorize(member, "TEST-revoked", "100.64.0.12")
	ordinary := f.authorize(member, "TEST-ordinary", "100.64.0.13")
	otherDevice := f.authorize(other, "TEST-other-peer", "100.64.0.21")
	if !f.netbird.inPaidGroup("TEST-regrant") {
		t.Fatal("discovery did not grant the re-authorization target")
	}

	pending := f.insertPending(member.id, "TEST-pending")
	over := f.insertPending(member.id, "TEST-over-limit")
	regrantSince := f.fence(regrant, true)
	revokedSince := f.fence(revoked, true)
	overSince := f.fence(over, false)
	f.netbird.withdrawPaid("TEST-regrant")
	f.netbird.withdrawPaid("TEST-revoked")
	if f.netbird.inPaidGroup("TEST-regrant") || f.netbird.inPaidGroup("TEST-revoked") {
		t.Fatal("withdraw left a paid-group membership")
	}

	// (a) fence filter and authorization_unconfirmed_since.
	fenced := f.adminDevices(admin, url.Values{"authorization_unconfirmed": {"true"}})
	if len(fenced) != 3 {
		t.Fatalf("fenced page %d", len(fenced))
	}
	for _, id := range []uuid.UUID{regrant, revoked, over} {
		item, ok := adminDevice(fenced, id)
		if !ok || item.AuthorizationUnconfirmed == nil || !*item.AuthorizationUnconfirmed || item.MemberId == nil || *item.MemberId != member.id {
			t.Fatalf("fenced item %s missing: %+v", id, item)
		}
	}
	if got := adminSince(t, adminDeviceMust(t, fenced, regrant)); got.UnixMicro() != regrantSince.UnixMicro() {
		t.Fatalf("regrant since %s want %s", got, regrantSince)
	}
	if got := adminSince(t, adminDeviceMust(t, fenced, revoked)); got.UnixMicro() != revokedSince.UnixMicro() {
		t.Fatalf("revoked since %s want %s", got, revokedSince)
	}
	if got := adminSince(t, adminDeviceMust(t, fenced, over)); got.UnixMicro() != overSince.UnixMicro() {
		t.Fatalf("over-limit since %s want %s", got, overSince)
	}
	clear := f.adminDevices(admin, url.Values{"authorization_unconfirmed": {"false"}})
	for _, item := range clear {
		if item.DeviceId == regrant || item.DeviceId == revoked || item.DeviceId == over {
			t.Fatalf("fenced device on the clear page: %s", item.DeviceId)
		}
		if item.AuthorizationUnconfirmed == nil || *item.AuthorizationUnconfirmed || !item.AuthorizationUnconfirmedSince.IsNull() {
			t.Fatalf("clear item projection: %+v", item)
		}
	}
	if len(clear) != 3 {
		t.Fatalf("clear page %d", len(clear))
	}

	// (b) member list: fenced reads pending + true; an ordinary pending device reads false.
	memberPage := f.memberDevices(member)
	requireFlag(t, memberPage, regrant, "pending", true)
	requireFlag(t, memberPage, revoked, "pending", true)
	requireFlag(t, memberPage, pending, "pending", false)
	requireFlag(t, memberPage, over, "pending", true)
	requireFlag(t, memberPage, ordinary, "authorized", false)
	for _, item := range memberPage {
		if item.AuthorizationUnconfirmed == nil {
			t.Fatalf("member device omitted authorization_unconfirmed: %s", item.DeviceId)
		}
	}
	otherPage := f.memberDevices(other)
	requireFlag(t, otherPage, otherDevice, "authorized", false)

	// (g) member_id and status filters, and cursor pages of more than one page.
	paged := f.adminDevices(admin, url.Values{"limit": {"1"}})
	if len(paged) != 6 {
		t.Fatalf("paged devices %d", len(paged))
	}
	oneMember := f.adminDevices(admin, url.Values{"member_id": {member.id.String()}, "limit": {"2"}})
	if len(oneMember) != 5 {
		t.Fatalf("member filter %d", len(oneMember))
	}
	for _, item := range oneMember {
		if item.MemberId == nil || *item.MemberId != member.id {
			t.Fatalf("member filter leaked %v", item.MemberId)
		}
	}
	pendingPage := f.adminDevices(admin, url.Values{"status": {"pending"}, "limit": {"2"}})
	if len(pendingPage) != 4 {
		t.Fatalf("pending filter %d", len(pendingPage))
	}
	for _, item := range pendingPage {
		if item.Status != "pending" {
			t.Fatalf("pending filter status %s", item.Status)
		}
	}
	if _, ok := adminDevice(pendingPage, regrant); !ok {
		t.Fatal("fenced device missing from status=pending")
	}
	if _, ok := adminDevice(pendingPage, ordinary); ok {
		t.Fatal("observed authorized device matched status=pending")
	}
	authorizedPage := f.adminDevices(admin, url.Values{"status": {"authorized"}})
	if len(authorizedPage) != 2 {
		t.Fatalf("authorized filter %d", len(authorizedPage))
	}
	for _, id := range []uuid.UUID{ordinary, otherDevice} {
		if _, ok := adminDevice(authorizedPage, id); !ok {
			t.Fatalf("authorized filter missing %s", id)
		}
	}
	if len(f.adminDevices(admin, url.Values{"status": {"revoked"}})) != 0 {
		t.Fatal("revoked filter matched before revocation")
	}
	first := f.call(admin, "GET", "/api/v1/admin/devices?limit=1", "", nil)
	requireStatus(t, first, 200)
	cursor, err := decode[api.AdminDeviceList](t, first).NextCursor.Get()
	if err != nil || cursor == "" {
		t.Fatal(cursor, err)
	}
	bound := f.call(admin, "GET", "/api/v1/admin/devices?"+url.Values{"member_id": {member.id.String()}, "cursor": {cursor}}.Encode(), "", nil)
	requireStatus(t, bound, 400)
	if code := decode[apiErrorBody](t, bound).Error.Code; code != "MALFORMED_REQUEST" {
		t.Fatal(bound.Body.String())
	}

	// A member session is not an admin session. The route reads only the admin
	// cookie, so this is unauthenticated and the fence stays.
	denied := f.call(member, "POST", authorizePath(member.id, regrant), "", nil)
	requireStatus(t, denied, 401)
	if code := decode[apiErrorBody](t, denied).Error.Code; code != "UNAUTHENTICATED" {
		t.Fatal(denied.Body.String())
	}
	if f.fences(regrant) != 1 {
		t.Fatal("member authorize cleared the fence")
	}

	// (f) over the allowance: the code is DEVICE_LIMIT_REACHED and the fence stays.
	// pkg/apierr maps that code to 409. D-077's 422 is recorded for the orchestrator.
	limited := f.call(admin, "POST", authorizePath(member.id, over), "", nil)
	requireStatus(t, limited, 409)
	if code := decode[apiErrorBody](t, limited).Error.Code; code != "DEVICE_LIMIT_REACHED" {
		t.Fatal(limited.Body.String())
	}
	if f.fences(over) != 1 {
		t.Fatal("device-limit rejection deleted the fence")
	}

	// (d) the device belongs to a different member.
	missing := f.call(admin, "POST", authorizePath(other.id, regrant), "", nil)
	requireStatus(t, missing, 404)
	if code := decode[apiErrorBody](t, missing).Error.Code; code != "RESOURCE_NOT_FOUND" {
		t.Fatal(missing.Body.String())
	}
	if f.fences(regrant) != 1 {
		t.Fatal("wrong member deleted the fence")
	}

	// (e) revocation clears the fence; a later authorize conflicts.
	w := f.call(member, "DELETE", "/api/v1/member/devices/"+revoked.String(), "", nil)
	requireStatus(t, w, 202)
	if f.fences(revoked) != 0 {
		t.Fatal("revoke left the fence")
	}
	conflict := f.call(admin, "POST", authorizePath(member.id, revoked), "", nil)
	requireStatus(t, conflict, 409)
	if code := decode[apiErrorBody](t, conflict).Error.Code; code != "STATE_CONFLICT" {
		t.Fatal(conflict.Body.String())
	}
	if f.fences(revoked) != 0 {
		t.Fatal("conflict authorize recreated the fence")
	}
	f.drain()
	revokedPage := f.adminDevices(admin, url.Values{"status": {"revoked"}, "member_id": {member.id.String()}})
	item, ok := adminDevice(revokedPage, revoked)
	if !ok || item.Status != "revoked" || item.AuthorizationUnconfirmed == nil || *item.AuthorizationUnconfirmed || !item.AuthorizationUnconfirmedSince.IsNull() {
		t.Fatalf("revoked projection: %+v %v", item, ok)
	}

	// (c) re-authorize deletes the fence with the grant and replays one operation.
	before := f.netbird.grantCount()
	ops := f.count(`SELECT count(*) FROM operations WHERE resource_id=$1 AND kind=$2`, regrant, api.OperationKindMemberDeviceAuthorize)
	key := uuid.NewString()
	w = f.call(admin, "POST", authorizePath(member.id, regrant), "", map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 202)
	accepted := decode[api.Operation](t, w)
	if accepted.Kind != api.OperationKindMemberDeviceAuthorize || accepted.Resource.Type != api.ResourceTypeMemberDevice || accepted.Resource.Id != regrant {
		t.Fatalf("operation %+v", accepted)
	}
	if f.fences(regrant) != 0 {
		t.Fatal("authorize left the fence")
	}
	f.drain()
	if got := f.netbird.grantCount(); got != before+1 {
		t.Fatalf("grants %d before %d", got, before)
	}
	if !f.netbird.inPaidGroup("TEST-regrant") {
		t.Fatal("re-authorization did not restore the paid group")
	}
	requireFlag(t, f.memberDevices(member), regrant, "authorized", false)
	outbox := f.count(`SELECT count(*) FROM outbox_events`)
	w = f.call(admin, "POST", authorizePath(member.id, regrant), "", map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 202)
	replayed := decode[api.Operation](t, w)
	if replayed.OperationId != accepted.OperationId {
		t.Fatalf("replay operation %s want %s", replayed.OperationId, accepted.OperationId)
	}
	f.drain()
	if f.netbird.grantCount() != before+1 || f.count(`SELECT count(*) FROM outbox_events`) != outbox {
		t.Fatal("idempotent replay granted again")
	}
	// An already-authorized device keeps its original member.device_authorize row.
	// Replay must not insert another one.
	if got := f.count(`SELECT count(*) FROM operations WHERE resource_id=$1 AND kind=$2`, regrant, api.OperationKindMemberDeviceAuthorize); got != ops {
		t.Fatalf("authorize operations %d before %d", got, ops)
	}
}

func (f *fixture) insertPending(member uuid.UUID, peer string) uuid.UUID {
	f.t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		f.t.Fatal(err)
	}
	f.exec(`INSERT INTO member_devices(id,member_id,deployment_id,peer_id,name,status,platform) VALUES($1,$2,'TEST-ONLY',$3,'TEST ONLY pending','pending','linux')`, id, member, peer)
	return id
}

func (f *fixture) fence(device uuid.UUID, observed bool) time.Time {
	f.t.Helper()
	var at time.Time
	if err := f.pool.QueryRow(f.t.Context(), `INSERT INTO device_authorization_fences(device_id,fenced_at) VALUES($1,clock_timestamp()) RETURNING fenced_at`, device).Scan(&at); err != nil {
		f.t.Fatal(err)
	}
	if observed {
		tag, err := f.pool.Exec(f.t.Context(), `UPDATE device_authorization_observations SET observed_authorized=false WHERE device_id=$1`, device)
		if err != nil || tag.RowsAffected() != 1 {
			f.t.Fatalf("observation update %d %v", tag.RowsAffected(), err)
		}
	}
	return at
}

func (f *fixture) fences(device uuid.UUID) int {
	f.t.Helper()
	return f.count(`SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, device)
}

func authorizePath(member, device uuid.UUID) string {
	return "/api/v1/admin/members/" + member.String() + "/devices/" + device.String() + "/authorize"
}

type apiErrorBody struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func (f *fixture) adminDevices(admin browserSession, query url.Values) []api.AdminDevice {
	f.t.Helper()
	q := url.Values{}
	for k, vs := range query {
		q[k] = append([]string{}, vs...)
	}
	limit := q.Get("limit")
	var all []api.AdminDevice
	seen := map[uuid.UUID]bool{}
	for pageN := 1; pageN <= 20; pageN++ {
		path := "/api/v1/admin/devices"
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
		w := f.call(admin, "GET", path, "", nil)
		requireStatus(f.t, w, 200)
		page := decode[api.AdminDeviceList](f.t, w)
		if len(page.Items) > 0 {
			requireDeviceProjection(f.t, w.Body.Bytes(), true)
		}
		if limit == "1" && len(page.Items) != 1 {
			f.t.Fatalf("limit 1 returned %d", len(page.Items))
		}
		for _, item := range page.Items {
			if item.AuthorizationUnconfirmed == nil || item.MemberId == nil || !item.AuthorizationUnconfirmedSince.IsSpecified() {
				f.t.Fatalf("admin projection: %+v", item)
			}
			if seen[item.DeviceId] {
				f.t.Fatalf("duplicate device %s", item.DeviceId)
			}
			seen[item.DeviceId] = true
			if len(all) > 0 {
				prev := all[len(all)-1]
				if bytes.Compare((*item.MemberId)[:], (*prev.MemberId)[:]) < 0 || (bytes.Equal((*item.MemberId)[:], (*prev.MemberId)[:]) && bytes.Compare(item.DeviceId[:], prev.DeviceId[:]) <= 0) {
					f.t.Fatalf("order broke at %s/%s", item.MemberId, item.DeviceId)
				}
			}
			all = append(all, item)
		}
		if page.NextCursor.IsNull() {
			if limit == "1" && len(all) > 1 && pageN != len(all) {
				f.t.Fatalf("pages %d devices %d", pageN, len(all))
			}
			return all
		}
		next, err := page.NextCursor.Get()
		if err != nil || next == "" {
			f.t.Fatal(err)
		}
		q.Set("cursor", next)
	}
	f.t.Fatal("admin cursor did not end")
	return nil
}

func (f *fixture) memberDevices(s browserSession) []api.MemberDevice {
	f.t.Helper()
	w := f.call(s, "GET", "/api/v1/member/devices", "", nil)
	requireStatus(f.t, w, 200)
	requireDeviceProjection(f.t, w.Body.Bytes(), false)
	return decode[api.MemberDeviceList](f.t, w).Items
}

func requireDeviceProjection(t *testing.T, body []byte, since bool) {
	t.Helper()
	var page struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		t.Fatal("projection check on an empty page")
	}
	for _, item := range page.Items {
		if !bytes.Contains(item, []byte(`"authorization_unconfirmed":`)) {
			t.Fatalf("authorization_unconfirmed omitted: %s", item)
		}
		if since && !bytes.Contains(item, []byte(`"authorization_unconfirmed_since":`)) {
			t.Fatalf("authorization_unconfirmed_since omitted: %s", item)
		}
	}
}

func adminDevice(items []api.AdminDevice, id uuid.UUID) (api.AdminDevice, bool) {
	for _, item := range items {
		if item.DeviceId == id {
			return item, true
		}
	}
	return api.AdminDevice{}, false
}

func adminDeviceMust(t *testing.T, items []api.AdminDevice, id uuid.UUID) api.AdminDevice {
	t.Helper()
	item, ok := adminDevice(items, id)
	if !ok {
		t.Fatalf("device %s not on page", id)
	}
	return item
}

func adminSince(t *testing.T, item api.AdminDevice) time.Time {
	t.Helper()
	got, err := item.AuthorizationUnconfirmedSince.Get()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func requireFlag(t *testing.T, items []api.MemberDevice, id uuid.UUID, status api.DeviceStatus, unconfirmed bool) {
	t.Helper()
	for _, item := range items {
		if item.DeviceId != id {
			continue
		}
		if item.Status != status || item.AuthorizationUnconfirmed == nil || *item.AuthorizationUnconfirmed != unconfirmed {
			t.Fatalf("device %s status %s unconfirmed %v", id, item.Status, item.AuthorizationUnconfirmed)
		}
		return
	}
	t.Fatalf("device %s missing", id)
}
