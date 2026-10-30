//go:build integration

package httpadapter_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReview_F01_PlanDowngradeAndUnsuspendExceedDeviceSlots proves that:
//  1. When a member with 3 authorized devices on a 3-slot plan transitions to a
//     1-slot plan revision, Entitlements.Recompute, SyncMember/SyncDevice, and
//     GET /api/v1/member/devices keep all 3 devices authorized (AuthorizedCount=3 > DeviceLimit=1).
//  2. When that member is suspended (withdrawing NetBird access for all 3 devices)
//     and then unsuspended while on the 1-slot plan, SyncMember re-authorizes all 3
//     devices in NetBird without enforcing DeviceLimit=1.
func TestReview_F01_PlanDowngradeAndUnsuspendExceedDeviceSlots(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	member := f.login("TEST-f01-member", "member")
	admin := f.login("TEST-f01-admin", "admin")

	// Seed a 1-slot plan revision alongside the fixture's 3-slot revision.
	rev1Slot := uuid.New()
	f.exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,provider,provider_price_ref,schema_version,payload) VALUES($1,$2,2,1,$3,'fake','TEST-ONLY-price-1slot',1,'{"display_name":{"zh-TW":"單裝置方案"}}')`,
		rev1Slot, f.plan, []uuid.UUID{f.group})

	memberPath := "/api/v1/admin/members/" + member.id.String()
	// Grant 3-slot plan first.
	grant3Body := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY 3-slot"}`,
		f.revision, f.now.Add(2*time.Hour).Format(time.RFC3339))
	w := f.call(admin, "POST", memberPath+"/grants", grant3Body, nil)
	requireStatus(t, w, 202)
	grant3ID := decode[api.Operation](t, w).Resource.Id
	f.drain()

	// Admit and sync 3 devices for the member.
	f.authorize(member, "TEST-f01-peer-1", "100.64.0.31")
	f.authorize(member, "TEST-f01-peer-2", "100.64.0.32")
	f.authorize(member, "TEST-f01-peer-3", "100.64.0.33")

	// Revoke the 3-slot grant and immediately issue the 1-slot grant so the
	// member's active entitlement downgrades from device_limit=3 to device_limit=1.
	cmd := entitlement.Command{Actor: entitlement.Actor{ID: admin.id, Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
	if err := f.services.HTTP.Members.RevokeGrant(ctx, cmd, member.id, grant3ID, "TEST ONLY downgrade to 1 slot"); err != nil {
		t.Fatal(err)
	}
	grant1Body := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY 1-slot"}`,
		rev1Slot, f.now.Add(2*time.Hour).Format(time.RFC3339))
	requireStatus(t, f.call(admin, "POST", memberPath+"/grants", grant1Body, nil), 202)
	f.drain()

	w = f.call(member, "GET", "/api/v1/member/devices", "", nil)
	requireStatus(t, w, 200)
	devList := decode[api.MemberDeviceList](t, w)
	limit, _ := devList.DeviceLimit.Get()
	if devList.AuthorizedCount > limit {
		t.Errorf("[F-01]: after plan downgrade to 1 slot, GET /api/v1/member/devices reports authorized_count=%d > device_limit=%d",
			devList.AuthorizedCount, limit)
	}

	upstreamAuthCount := 0
	for _, pid := range []string{"TEST-f01-peer-1", "TEST-f01-peer-2", "TEST-f01-peer-3"} {
		p, err := f.services.Adapter.ObservePeer(ctx, "TEST-ONLY", pid)
		if err != nil {
			t.Fatal(err)
		}
		if p.Authorized {
			upstreamAuthCount++
		}
	}
	if upstreamAuthCount > 1 {
		t.Errorf("[F-01]: after plan downgrade to 1 slot and worker drain, %d peers remain authorized in NetBird; want <= 1",
			upstreamAuthCount)
	}

	// Suspend the member (withdrawing all 3 peers), then unsuspend while still on the 1-slot plan.
	w = f.call(admin, "GET", memberPath, "", nil)
	requireStatus(t, w, 200)
	requireStatus(t, f.call(admin, "PATCH", memberPath, `{"admin_suspended":true,"suspension_reason":"TEST ONLY suspend"}`, map[string]string{"If-Match": w.Header().Get("ETag")}), 202)
	f.drain()

	w = f.call(admin, "GET", memberPath, "", nil)
	requireStatus(t, w, 200)
	requireStatus(t, f.call(admin, "PATCH", memberPath, `{"admin_suspended":false}`, map[string]string{"If-Match": w.Header().Get("ETag")}), 202)
	f.drain()

	reAdmittedCount := 0
	for _, pid := range []string{"TEST-f01-peer-1", "TEST-f01-peer-2", "TEST-f01-peer-3"} {
		p, err := f.services.Adapter.ObservePeer(ctx, "TEST-ONLY", pid)
		if err != nil {
			t.Fatal(err)
		}
		if p.Authorized {
			reAdmittedCount++
		}
	}
	if reAdmittedCount > 1 {
		t.Errorf("[F-01]: re-admission after suspension on 1-slot plan re-authorized %d peers in NetBird; want <= 1",
			reAdmittedCount)
	}
}

// TestReview_F03_D052ProofRevisionAdvanceBricksDeviceSync proves that when an
// entitled member has synced a device at netbird_user_bindings.revision=1 (recording
// DesiredHash=hash([identityBindingID,"1",userID]) in external_bindings) and
// netbird_user_bindings.revision subsequently advances to 2 via D-052 proof update,
// every subsequent SyncDevice call for existing or new devices fails permanently
// with netbird: conflict.
func TestReview_F03_D052ProofRevisionAdvanceBricksDeviceSync(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	member := f.login("TEST-f03-member", "member")
	admin := f.login("TEST-f03-admin", "admin")

	memberPath := "/api/v1/admin/members/" + member.id.String()
	grantBody := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY f03"}`,
		f.revision, f.now.Add(time.Hour).Format(time.RFC3339))
	requireStatus(t, f.call(admin, "POST", memberPath+"/grants", grantBody, nil), 202)
	f.drain()

	// Admit and sync first device at netbird_user_bindings.revision=1.
	dev1 := f.authorize(member, "TEST-f03-peer-1", "100.64.0.61")

	// Trigger the D-052 proof_updated path (same path exercised in TestAT01_D052ChangedProofAdvancesRevisionOnce)
	// which advances netbird_user_bindings.revision from 1 to 2.
	f.exec(`UPDATE netbird_user_bindings SET mapping_proof='legacy-proof' WHERE member_id=$1`, member.id)
	if err := f.services.Adapter.Discover(ctx, f.services.HTTP.Devices); err != nil {
		t.Fatal(err)
	}
	var rev int
	if err := f.pool.QueryRow(ctx, `SELECT revision FROM netbird_user_bindings WHERE member_id=$1`, member.id).Scan(&rev); err != nil || rev != 2 {
		t.Fatalf("expected netbird_user_bindings.revision=2, got %d (err=%v)", rev, err)
	}

	// Now re-syncing dev1 or syncing a newly admitted dev2 fails with netbird: conflict!
	if err := f.services.HTTP.Devices.SyncDevice(ctx, dev1); err != nil {
		t.Errorf("[F-03]: SyncDevice for existing device failed after netbird_user_bindings.revision advanced to 2: %v", err)
	}

	f.seedPeer(member, "TEST-f03-peer-2", "100.64.0.62")
	if err := f.services.Adapter.Discover(ctx, f.services.HTTP.Devices); err != nil {
		t.Fatal(err)
	}
	var dev2 uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT id FROM member_devices WHERE member_id=$1 AND peer_id='TEST-f03-peer-2'`, member.id).Scan(&dev2); err != nil {
		t.Fatal(err)
	}
	if err := f.services.HTTP.Devices.SyncDevice(ctx, dev2); err != nil {
		t.Errorf("[F-03]: SyncDevice for new device failed after netbird_user_bindings.revision advanced to 2: %v", err)
	}
}

// TestReview_F07_D029FailedAdmissionsUnauditedAndUnallowlistedAuditPayloads proves:
//  1. Failed device admissions (ErrLimit, ErrEntitlement, ErrUnmapped) emit zero
//     audit_events rows (violating Question 5: "failed admissions are audited with a fixed reason").
//  2. entitlement.Audit bypasses audit.Diff allowlists and writes MemberPatch
//     (display_name, notes, suspension_reason) and grant/revoke reason strings
//     verbatim into audit_events.after_diff (allowing email, raw subject, and IP
//     leakage into audit payloads), while leaving formal column audit_events.operation_id NULL.
func TestReview_F07_D029FailedAdmissionsUnauditedAndUnallowlistedAuditPayloads(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	member := f.login("TEST-f07-member", "member")
	unentitled := f.login("TEST-f07-unentitled", "member")
	admin := f.login("TEST-f07-admin", "admin")

	memberPath := "/api/v1/admin/members/" + member.id.String()
	grantBody := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY grant"}`,
		f.revision, f.now.Add(time.Hour).Format(time.RFC3339))
	requireStatus(t, f.call(admin, "POST", memberPath+"/grants", grantBody, nil), 202)
	f.drain()

	// Fill all 3 slots for member.
	f.authorize(member, "TEST-f07-peer-1", "100.64.0.71")
	f.authorize(member, "TEST-f07-peer-2", "100.64.0.72")
	f.authorize(member, "TEST-f07-peer-3", "100.64.0.73")

	beforeAuditCount := f.count(`SELECT count(*) FROM audit_events`)

	// Trigger failed admissions:
	// (a) 4th device for member -> ErrLimit
	f.seedPeer(member, "TEST-f07-peer-4-overlimit", "100.64.0.74")
	cmd := entitlement.Command{Actor: entitlement.Actor{ID: member.id, Role: "system", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := f.services.HTTP.Devices.ObserveAndAuthorize(ctx, cmd, "TEST-ONLY", "TEST-f07-peer-4-overlimit"); !errors.Is(err, devices.ErrLimit) {
		t.Fatalf("expected ErrLimit, got %v", err)
	}
	// (b) Device for unentitled member -> ErrEntitlement
	f.seedPeer(unentitled, "TEST-f07-peer-unentitled", "100.64.0.75")
	if err := f.services.Adapter.Discover(ctx, f.services.HTTP.Devices); err != nil {
		t.Fatal(err)
	}
	cmd2 := entitlement.Command{Actor: entitlement.Actor{ID: unentitled.id, Role: "system", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := f.services.HTTP.Devices.ObserveAndAuthorize(ctx, cmd2, "TEST-ONLY", "TEST-f07-peer-unentitled"); !errors.Is(err, devices.ErrEntitlement) {
		t.Fatalf("expected ErrEntitlement, got %v", err)
	}
	// (c) Unmapped native user peer -> ErrUnmapped
	f.seedNative("TEST-f07-unmapped-subject", "TEST-f07-peer-unmapped", "100.64.0.76", false, false)
	cmd3 := entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := f.services.HTTP.Devices.ObserveAndAuthorize(ctx, cmd3, "TEST-ONLY", "TEST-f07-peer-unmapped"); !errors.Is(err, devices.ErrUnmapped) {
		t.Fatalf("expected ErrUnmapped, got %v", err)
	}

	afterAuditCount := f.count(`SELECT count(*) FROM audit_events`)
	var deniedDeviceAudits int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE result='denied'`).Scan(&deniedDeviceAudits)
	if afterAuditCount == beforeAuditCount || deniedDeviceAudits == 0 {
		t.Errorf("[F-07]: failed device admissions (ErrLimit, ErrEntitlement, ErrUnmapped) produced %d new audit events (%d denied); want failed admissions audited with fixed reason (D-029)",
			afterAuditCount-beforeAuditCount, deniedDeviceAudits)
	}

	// Now patch member metadata with free-text fields containing email, raw subject, and IP.
	w := f.call(admin, "GET", memberPath, "", nil)
	requireStatus(t, w, 200)
	patchBody := `{"display_name":"leak-user@example.invalid","notes":"sub=TEST-f07-member ip=198.51.100.99","admin_suspended":true,"suspension_reason":"contact leak-user@example.invalid from 198.51.100.99"}`
	w = f.call(admin, "PATCH", memberPath, patchBody, map[string]string{"If-Match": w.Header().Get("ETag")})
	requireStatus(t, w, 202)

	store := audit.Store{Pool: f.pool}
	page, err := store.List(ctx, api.ListAuditEventsParams{})
	if err != nil {
		t.Fatal(err)
	}
	denialReasons := map[string]bool{}
	for _, ev := range page.Items {
		raw, _ := json.Marshal(ev)
		for _, forbidden := range []string{"leak-user@example.invalid", "TEST-f07-member", "198.51.100.99"} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("[F-07]: audit event %s (%s) leaked forbidden PII/subject/IP %q in payload because entitlement.Audit bypasses audit.Diff allowlists",
					ev.Action, ev.AuditEventId, forbidden)
			}
		}
		// A denial creates no asynchronous operation. The original proof's blanket
		// non-null check applied to these new rows too; require correlation for
		// accepted commands and explicitly forbid fabricated operations on denials.
		if ev.Action == "device.authorize" && ev.Result == api.AuditEventResultDenied {
			if !ev.OperationId.IsNull() {
				t.Fatal("denied admission fabricated an operation")
			}
			diff, _ := ev.Changes.After.Get()
			reason, _ := diff["reason"].(string)
			switch reason {
			case "device_limit_reached", "entitlement_inactive", "identity_unmapped":
				denialReasons[reason] = true
			default:
				t.Fatalf("unbounded denial reason %q", reason)
			}
		}

		if (ev.Action == "device.authorize" || ev.Action == "member.grant" || ev.Action == "member.patch") && ev.Result == api.AuditEventResultSucceeded && ev.OperationId.IsNull() {
			t.Errorf("[F-07]: audit event %s (%s) has formal OperationId=null (operation_id was only written into after_diff JSON)",
				ev.Action, ev.AuditEventId)
		}
	}
	if len(denialReasons) != 3 || store.AnomalyCount() != 0 {
		t.Fatal("missing denied evidence or audit anomalies", denialReasons, store.AnomalyCount())
	}

}

// TestReview_Held_CrossMemberIsolationAnd404Parity verifies §7 isolation: a member
// never learns another member's device IDs, subjects, or group IDs through any list,
// error message, or 404/409 status difference.
func TestReview_Held_CrossMemberIsolationAnd404Parity(t *testing.T) {
	f := newFixture(t)
	memberA := f.login("TEST-iso-member-a", "member")
	memberB := f.login("TEST-iso-member-b", "member")
	admin := f.login("TEST-iso-admin", "admin")

	for _, m := range []browserSession{memberA, memberB} {
		path := "/api/v1/admin/members/" + m.id.String()
		grantBody := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_until":%q,"reason":"TEST ONLY iso"}`,
			f.revision, f.now.Add(time.Hour).Format(time.RFC3339))
		requireStatus(t, f.call(admin, "POST", path+"/grants", grantBody, nil), 202)
	}
	f.drain()

	devA := f.authorize(memberA, "TEST-iso-peer-a", "100.64.0.81")
	devBActive := f.authorize(memberB, "TEST-iso-peer-b1", "100.64.0.82")
	devBRevoked := f.authorize(memberB, "TEST-iso-peer-b2", "100.64.0.83")

	wRev := f.call(memberB, "DELETE", "/api/v1/member/devices/"+devBRevoked.String(), "", nil)
	requireStatus(t, wRev, 202)
	opBRevoke := decode[api.Operation](t, wRev)
	f.drain()

	// 1. Member A's /member/me and /member/devices must never contain Member B's IDs, subjects, or group UUIDs.
	for _, endpoint := range []string{"/api/v1/member/me", "/api/v1/member/devices"} {
		w := f.call(memberA, "GET", endpoint, "", nil)
		requireStatus(t, w, 200)
		body := w.Body.String()
		for _, secret := range []string{
			memberB.id.String(),
			devBActive.String(),
			devBRevoked.String(),
			"TEST-iso-member-b",
			"TEST-iso-peer-b1",
			f.group.String(),
		} {
			if strings.Contains(body, secret) {
				t.Fatalf("GET %s leaked %q in body: %s", endpoint, secret, body)
			}
		}
		if endpoint == "/api/v1/member/devices" && !strings.Contains(body, devA.String()) {
			t.Fatalf("GET %s missing own device %s", endpoint, devA)
		}
	}

	// 2. DELETE /api/v1/member/devices/{id} as Member A must return identical 404
	// for Member B's active device, Member B's already-revoked device, and a random UUID.
	randomID := uuid.New()
	var baselineCode, baselineMsg string
	for i, targetID := range []uuid.UUID{randomID, devBActive, devBRevoked} {
		w := f.call(memberA, "DELETE", "/api/v1/member/devices/"+targetID.String(), "", nil)
		requireStatus(t, w, 404)
		var env api.ErrorEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			baselineCode = string(env.Error.Code)
			baselineMsg = env.Error.Message
		} else if string(env.Error.Code) != baselineCode || env.Error.Message != baselineMsg {
			t.Fatalf("404 oracle difference for target %s: got (%s, %q), want (%s, %q)",
				targetID, env.Error.Code, env.Error.Message, baselineCode, baselineMsg)
		}
	}

	// 3. GET /api/v1/member/operations/{id} as Member A must return identical 404
	// for Member B's operation and a random UUID.
	for _, opID := range []uuid.UUID{uuid.New(), opBRevoke.OperationId} {
		w := f.call(memberA, "GET", "/api/v1/member/operations/"+opID.String(), "", nil)
		requireStatus(t, w, 404)
	}
}
