//go:build integration

package devices

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/google/uuid"
)

type fakeAdapter struct {
	mu           sync.Mutex
	peers        map[string]Peer
	ignoreWrites bool
}

func (f *fakeAdapter) ObservePeer(_ context.Context, _, peer string) (Peer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.peers[peer]
	if !ok {
		return p, db.ErrNotFound
	}
	return p, nil
}
func (f *fakeAdapter) SetAuthorization(_ context.Context, _, peer string, allow bool, groups []uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ignoreWrites {
		return nil
	}
	p, ok := f.peers[peer]
	if !ok {
		return db.ErrNotFound
	}
	p.Authorized = allow
	p.Groups = append([]uuid.UUID{}, groups...)
	f.peers[peer] = p
	return nil
}
func deviceCommand(role string, id uuid.UUID) entitlement.Command {
	return entitlement.Command{Actor: entitlement.Actor{Role: role, ID: id, RequestID: uuid.New()}, Key: uuid.New()}
}
func setupDevices(t *testing.T) (*Service, uuid.UUID, []Peer) {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()
	member, identity, plan, revision, group := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id) VALUES($1)`, member)
	exec(`INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,'https://id.example.invalid','portal','portal-subject','verified-shared-claim','verified-link',$3)`, identity, member, now)
	exec(`INSERT INTO netbird_user_bindings(id,member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,$2,'test','netbird-user',$3,'TEST ONLY verified cross-client proof',1,$4)`, uuid.New(), member, identity, now)
	exec(`INSERT INTO member_groups(id,code,name) VALUES($1,'test','TEST ONLY')`, group)
	exec(`INSERT INTO plans(id,code) VALUES($1,'test')`, plan)
	exec(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,1,$3,1,'{}')`, revision, plan, []uuid.UUID{group})
	exec(`INSERT INTO entitlement_grants(id,member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,$2,'admin_grant',$3,$4,'TEST ONLY finite grant','test-admin',$5)`, uuid.New(), member, now.Add(-time.Hour), now.Add(time.Hour), revision)
	engine, _ := entitlement.NewStore(entitlement.TestConfig())
	engine.Now = func() time.Time { return now }
	peers := []Peer{{DeploymentID: "test", ID: "peer-a", ExternalUserID: "netbird-user", OverlayScope: "test", OverlayIP: netip.MustParseAddr("100.64.0.1"), Platform: "linux"}, {DeploymentID: "test", ID: "peer-b", ExternalUserID: "netbird-user", OverlayScope: "test", OverlayIP: netip.MustParseAddr("100.64.0.2"), Platform: "android"}}
	adapter := &fakeAdapter{peers: map[string]Peer{peers[0].ID: peers[0], peers[1].ID: peers[1]}}
	return &Service{Pool: pool, Entitlements: engine, Adapter: adapter}, member, peers
}
func TestAT02_LastSlotRaceIsolationAndRevoke(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	start := make(chan struct{})
	type outcome struct {
		change Change
		err    error
	}
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Go(func() {
			<-start
			c, e := s.AuthorizeDevice(ctx, deviceCommand("system", member), peer)
			results <- outcome{c, e}
		})
	}
	close(start)
	wg.Wait()
	close(results)
	var winner Change
	success, limited := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			winner = result.change
			success++
		case errors.Is(result.err, ErrLimit):
			limited++
		default:
			t.Fatal(result.err)
		}
	}
	if success != 1 || limited != 1 {
		t.Fatalf("success=%d limit=%d", success, limited)
	}
	foreign := deviceCommand("member", uuid.New())
	if _, err := s.Get(ctx, foreign.Actor, winner.DeviceID); !errors.Is(err, db.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.RevokeDevice(ctx, foreign, winner.DeviceID); !errors.Is(err, db.ErrNotFound) {
		t.Fatal(err)
	}
	page, err := s.List(ctx, deviceCommand("member", member).Actor, member, "", 50)
	if err != nil || len(page.Items) != 1 || page.AuthorizedCount != 1 {
		t.Fatal(page, err)
	}
	if err = s.SyncDevice(ctx, winner.DeviceID); err != nil {
		t.Fatal(err)
	}
	own := deviceCommand("member", member)
	if _, err = s.RevokeDevice(ctx, own, winner.DeviceID); err != nil {
		t.Fatal(err)
	}
	if err = s.SyncDevice(ctx, winner.DeviceID); err != nil {
		t.Fatal(err)
	}
	var winnerPeer string
	if err = s.Pool.QueryRow(ctx, `SELECT peer_id FROM member_devices WHERE id=$1`, winner.DeviceID).Scan(&winnerPeer); err != nil {
		t.Fatal(err)
	}
	for _, peer := range peers {
		if peer.ID != winnerPeer {
			if _, err = s.AuthorizeDevice(ctx, deviceCommand("system", member), peer); err != nil {
				t.Fatal(err)
			}
		}
	}
	var open int
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE device_id=$1 AND valid_until='infinity'`, winner.DeviceID).Scan(&open); err != nil || open != 0 {
		t.Fatal(open, err)
	}
	var events int
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type IN ('device.authorized','device.revoked')`).Scan(&events); err != nil || events != 3 {
		t.Fatal(events, err)
	}
}
func TestAT01_VerifiedCrossClientMappingAndReadBack(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	unknown := peers[0]
	unknown.ExternalUserID = "unmapped-user"
	if _, err := s.AuthorizeDevice(ctx, deviceCommand("system", member), unknown); !errors.Is(err, ErrUnmapped) {
		t.Fatal(err)
	}
	unknown.ExternalUserID = ""
	if _, err := s.AuthorizeDevice(ctx, deviceCommand("system", member), unknown); !errors.Is(err, ErrUnmapped) {
		t.Fatal(err)
	}
	change, err := s.ObserveAndAuthorize(ctx, deviceCommand("system", member), peers[0].DeploymentID, peers[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	adapter := s.Adapter.(*fakeAdapter)
	adapter.ignoreWrites = true
	if err = s.SyncDevice(ctx, change.DeviceID); !errors.Is(err, entitlement.ErrConflict) {
		t.Fatal("missing read-back accepted", err)
	}
	d, err := s.Get(ctx, deviceCommand("member", member).Actor, change.DeviceID)
	if err != nil || d.ObservedAuthorized {
		t.Fatal(d, err)
	}
	adapter.ignoreWrites = false
	if err = s.SyncDevice(ctx, change.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pool.Exec(ctx, `UPDATE members SET admin_suspended=true WHERE id=$1`, member); err != nil {
		t.Fatal(err)
	}
	// The worker reads a recomputed entitlement, never old event authorization.
	if _, err = s.RevokeDevice(ctx, deviceCommand("admin", uuid.New()), change.DeviceID); err != nil {
		t.Fatal(err)
	}
	if err = s.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	p, _ := adapter.ObservePeer(ctx, "test", peers[0].ID)
	if p.Authorized {
		t.Fatal("revoked peer remains authorized")
	}
}
