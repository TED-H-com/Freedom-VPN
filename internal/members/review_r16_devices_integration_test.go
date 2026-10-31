//go:build integration

package members

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/google/uuid"
)

func reviewDevices(t *testing.T, limit int) (*devices.Service, uuid.UUID, time.Time) {
	t.Helper()
	pool := dbtest.New(t)
	ctx := t.Context()
	member, identity, plan, revision, group := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id, email, display_name, status) VALUES($1, $2, $2, 'active')`, member, member.String()+"@example.invalid")
	exec(`INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,'https://id.example.invalid','portal',$3,'sub','oidc',$4)`, identity, member, "netbird-"+member.String()[:8], now)
	exec(`INSERT INTO netbird_user_bindings(id,member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,$2,'test',$3,$4,'shared_issuer_public_subject:https://id.example.invalid',1,$5)`, uuid.New(), member, "netbird-"+member.String()[:8], identity, now)
	exec(`INSERT INTO member_groups(id,code,name) VALUES($1,$2,$2)`, group, "r16-"+group.String()[:8])
	exec(`INSERT INTO plans(id,code) VALUES($1,$2)`, plan, "r16-"+plan.String()[:8])
	exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,$3,$4,1,'{}')`, revision, plan, limit, []uuid.UUID{group})
	exec(`INSERT INTO entitlement_grants(id,member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,$2,'admin_grant',$3,$4,'TEST ONLY finite grant','test-admin',$5)`, uuid.New(), member, now.Add(-time.Hour), now.Add(time.Hour), revision)
	engine, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	engine.Now = func() time.Time { return now }
	return &devices.Service{Pool: pool, Entitlements: engine}, member, now
}

func reviewPending(t *testing.T, s *devices.Service, member uuid.UUID, now time.Time, ip string) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	peer := "peer-" + id.String()
	ctx := t.Context()
	if _, err := s.Pool.Exec(ctx, `INSERT INTO member_devices(id,member_id,deployment_id,peer_id,name,platform,status) VALUES($1,$2,'test',$3,$4,'linux','pending')`, id, member, peer, "dev-"+peer); err != nil {
		t.Fatal(err)
	}
	// A closed historical binding lets promotePending open the next interval.
	// Two pending devices with no binding both roll back as ErrConflict.
	if _, err := s.Pool.Exec(ctx, `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,'test',$3,1,$4,$5,$6)`, id, member, ip, now.Add(-2*time.Hour), now.Add(-time.Hour), peer); err != nil {
		t.Fatal(err)
	}
	return id
}

func reviewAdmin() entitlement.Command {
	return entitlement.Command{Actor: entitlement.Actor{Role: "admin", ID: uuid.New(), RequestID: uuid.New()}, Key: uuid.New()}
}

// TestReviewR16_AdminDeviceCursorIsStable walks the admin keyset while a
// device is authorized and a newer device is inserted. Production device ids
// are UUIDv7, so a device created during the walk sorts after the cursor.
func TestReviewR16_AdminDeviceCursorIsStable(t *testing.T) {
	s, member, now := reviewDevices(t, 3)
	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = reviewPending(t, s, member, now, "100.64.1."+string(rune('1'+i)))
		time.Sleep(2 * time.Millisecond)
	}
	seen := map[uuid.UUID]int{}
	var extra uuid.UUID
	cursor := ""
	actor := reviewAdmin().Actor
	for pageN := 0; pageN < 20; pageN++ {
		page, err := s.ListAdmin(t.Context(), actor, devices.AdminQuery{Member: &member, Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range page.Items {
			seen[d.ID]++
		}
		if pageN == 0 {
			if _, err := s.AuthorizeMemberDevice(t.Context(), reviewAdmin(), member, ids[0]); err != nil {
				t.Fatal(err)
			}
			time.Sleep(5 * time.Millisecond)
			extra = reviewPending(t, s, member, now, "100.64.1.9")
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	for _, id := range ids {
		if seen[id] != 1 {
			t.Fatalf("TODO(review): admin device %s appeared %d times under a concurrent authorization", id, seen[id])
		}
	}
	if seen[extra] != 1 {
		t.Fatalf("TODO(review): device inserted during the walk appeared %d times", seen[extra])
	}
}

// TestReviewR16_ConcurrentAuthorizeHitsDeviceLimit is the production admin
// path with two pending devices and device_limit 1. Re-grant of the winner
// after a fence returns the same operation.
func TestReviewR16_ConcurrentAuthorizeHitsDeviceLimit(t *testing.T) {
	s, member, now := reviewDevices(t, 1)
	a := reviewPending(t, s, member, now, "100.64.2.1")
	b := reviewPending(t, s, member, now, "100.64.2.2")
	start := make(chan struct{})
	type outcome struct {
		id     uuid.UUID
		change devices.Change
		err    error
	}
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, id := range []uuid.UUID{a, b} {
		wg.Go(func() {
			<-start
			c, err := s.AuthorizeMemberDevice(t.Context(), reviewAdmin(), member, id)
			results <- outcome{id, c, err}
		})
	}
	close(start)
	wg.Wait()
	close(results)
	var winner outcome
	success, limited := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			winner = result
			success++
		case errors.Is(result.err, devices.ErrLimit):
			limited++
		default:
			t.Fatal(result.err)
		}
	}
	if success != 1 || limited != 1 {
		t.Fatalf("TODO(review): concurrent admin authorize success=%d limit=%d; device_limit 1 must leave one 409", success, limited)
	}
	var authorized int
	if err := s.Pool.QueryRow(t.Context(), `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, member).Scan(&authorized); err != nil || authorized != 1 {
		t.Fatalf("authorized count = %d (%v)", authorized, err)
	}
	if _, err := s.Pool.Exec(t.Context(), `INSERT INTO device_authorization_fences(device_id, fenced_at) VALUES($1, $2)`, winner.id, now); err != nil {
		t.Fatal(err)
	}
	again, err := s.AuthorizeMemberDevice(t.Context(), reviewAdmin(), member, winner.id)
	if err != nil || winner.change.OperationID == nil || again.OperationID == nil || *again.OperationID != *winner.change.OperationID {
		t.Fatalf("re-grant of a fenced authorized device = %+v %v, first operation %v", again, err, winner.change.OperationID)
	}
	var ops int
	if err := s.Pool.QueryRow(t.Context(), `SELECT count(*) FROM operations WHERE resource_id=$1`, winner.id).Scan(&ops); err != nil || ops != 1 {
		t.Fatalf("re-grant created another operation: %d %v", ops, err)
	}
}
