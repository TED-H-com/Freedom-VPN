//go:build integration

package devices

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// gateAdapter blocks the first withdrawal so a second SyncMember overlaps the
// member lock. It counts terminal authorization writes per peer.
type gateAdapter struct {
	base    *fakeAdapter
	mu      sync.Mutex
	calls   map[string]int
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	fail    error
}

func (g *gateAdapter) ObservePeer(ctx context.Context, deployment, peer string) (Peer, error) {
	return g.base.ObservePeer(ctx, deployment, peer)
}

func (g *gateAdapter) SetAuthorization(ctx context.Context, deployment, peer string, allow bool, groups []uuid.UUID) error {
	if g.fail != nil {
		return g.fail
	}
	if !allow {
		g.mu.Lock()
		g.calls[peer]++
		g.mu.Unlock()
		g.once.Do(func() { close(g.entered) })
		select {
		case <-g.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return g.base.SetAuthorization(ctx, deployment, peer, allow, groups)
}

func (g *gateAdapter) open() {
	g.once.Do(func() { close(g.entered) })
	select {
	case <-g.release:
	default:
		close(g.release)
	}
}

type suspendFixture struct {
	svc    *Service
	member uuid.UUID
	peers  []Peer
	gate   *gateAdapter
	op     uuid.UUID
}

func newSuspendFixture(t *testing.T) suspendFixture {
	t.Helper()
	svc, member, peers := setupDevices(t)
	ctx := t.Context()
	base := svc.Adapter.(*fakeAdapter)
	gate := &gateAdapter{base: base, calls: map[string]int{}, entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(gate.open)
	third := Peer{DeploymentID: "test", ID: "peer-c", ExternalUserID: "netbird-user", OverlayScope: "test", OverlayIP: netip.MustParseAddr("100.64.0.3"), Platform: "linux"}
	base.mu.Lock()
	base.peers[third.ID] = third
	base.mu.Unlock()
	peers = append(peers, third)
	var plan uuid.UUID
	var groups []uuid.UUID
	if err := svc.Pool.QueryRow(ctx, `SELECT plan_id,exit_group_ids FROM plan_revisions`).Scan(&plan, &groups); err != nil {
		t.Fatal(err)
	}
	revision := uuid.New()
	if _, err := svc.Pool.Exec(ctx, `INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,2,3,$3,1,'{}')`, revision, plan, groups); err != nil {
		t.Fatal(err)
	}
	// plan_revisions are immutable. Point the open grant at a 3-device revision
	// before admission; AuthorizeDevice recomputes from that grant.
	if _, err := svc.Pool.Exec(ctx, `UPDATE entitlement_grants SET plan_revision_id=$1 WHERE member_id=$2`, revision, member); err != nil {
		t.Fatal(err)
	}
	svc.Adapter = gate
	for _, peer := range peers {
		if _, err := svc.AuthorizeDevice(ctx, deviceCommand("system", member), peer); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	for _, peer := range peers {
		observed, err := base.ObservePeer(ctx, "test", peer.ID)
		if err != nil || !observed.Authorized {
			t.Fatal("device not authorized before suspend", peer.ID, err)
		}
	}
	var changed, suspended int
	if err := svc.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE event_type='member.entitlement_changed'), count(*) FILTER (WHERE event_type='member.suspended') FROM outbox_events`).Scan(&changed, &suspended); err != nil {
		t.Fatal(err)
	}
	var version decstr.U64
	if err := svc.Pool.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1`, member).Scan(&version); err != nil {
		t.Fatal(err)
	}
	on, reason := true, "TEST ONLY contention suspend"
	cmd := entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := (&members.Service{Pool: svc.Pool, Entitlements: svc.Entitlements}).Patch(ctx, cmd, member, &version, api.MemberPatch{AdminSuspended: &on, SuspensionReason: &reason}); err != nil {
		t.Fatal(err)
	}
	var changedAfter, suspendedAfter int
	if err := svc.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE event_type='member.entitlement_changed'), count(*) FILTER (WHERE event_type='member.suspended') FROM outbox_events`).Scan(&changedAfter, &suspendedAfter); err != nil {
		t.Fatal(err)
	}
	if changedAfter != changed+1 || suspendedAfter != suspended+1 {
		t.Fatalf("suspend outbox = entitlement_changed %d->%d suspended %d->%d, want +1 and +1", changed, changedAfter, suspended, suspendedAfter)
	}
	var op uuid.UUID
	var status, step string
	if err := svc.Pool.QueryRow(ctx, `SELECT id,status,steps->'netbird'->>'status' FROM operations WHERE kind=$1 AND resource_id=$2`, api.OperationKindMemberSuspend, member).Scan(&op, &status, &step); err != nil {
		t.Fatal(err)
	}
	if status != string(api.OperationStatusPending) || step != string(api.OperationStepStatusPending) {
		t.Fatalf("suspend operation before sync = %s/%s", status, step)
	}
	return suspendFixture{svc: svc, member: member, peers: peers, gate: gate, op: op}
}

func operationState(t *testing.T, f suspendFixture) (status, code, step, stepCode string) {
	t.Helper()
	var errorCode, netbirdCode *string
	if err := f.svc.Pool.QueryRow(t.Context(), `SELECT status,error_code,steps->'netbird'->>'status',steps->'netbird'->>'error_code' FROM operations WHERE id=$1`, f.op).Scan(&status, &errorCode, &step, &netbirdCode); err != nil {
		t.Fatal(err)
	}
	if errorCode != nil {
		code = *errorCode
	}
	if netbirdCode != nil {
		stepCode = *netbirdCode
	}
	return status, code, step, stepCode
}

func TestSyncMemberContentionDoesNotFailSuspend(t *testing.T) {
	f := newSuspendFixture(t)
	ctx := t.Context()
	stop := make(chan struct{})
	done := make(chan struct{})
	var pollMu sync.Mutex
	failed := false
	polls := 0
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			var status, step string
			err := f.svc.Pool.QueryRow(context.Background(), `SELECT status,steps->'netbird'->>'status' FROM operations WHERE id=$1`, f.op).Scan(&status, &step)
			if err != nil {
				continue
			}
			pollMu.Lock()
			polls++
			if status == string(api.OperationStatusFailed) || step == string(api.OperationStepStatusFailed) {
				failed = true
			}
			pollMu.Unlock()
		}
	}()
	var stopOnce sync.Once
	halt := func() {
		stopOnce.Do(func() { close(stop) })
		<-done
	}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errs <- f.svc.SyncMember(ctx, f.member)
	}()
	t.Cleanup(func() {
		f.gate.open()
		wg.Wait()
		halt()
	})
	select {
	case <-f.gate.entered:
	case err := <-errs:
		t.Fatalf("SyncMember finished before the member lock was held: %v", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		errs <- f.svc.SyncMember(ctx, f.member)
	}()
	loser := <-errs
	if !errors.Is(loser, db.ErrLeaseLost) {
		t.Fatalf("overlapping SyncMember = %v, want lease lost", loser)
	}
	for range 20 {
		status, code, step, _ := operationState(t, f)
		if status == string(api.OperationStatusFailed) || step == string(api.OperationStepStatusFailed) || code != "" {
			t.Fatalf("operation failed while the other worker held the member lock: status=%s step=%s code=%s", status, step, code)
		}
	}
	f.gate.open()
	winner := <-errs
	halt()
	if winner != nil {
		t.Fatal(winner)
	}
	pollMu.Lock()
	seen, bad := polls, failed
	pollMu.Unlock()
	if bad || seen == 0 {
		t.Fatalf("polls=%d sawFailed=%v", seen, bad)
	}
	status, code, step, stepCode := operationState(t, f)
	// nodes stays pending until a node observation, so the operation is running
	// with the netbird step succeeded. It must not be failed.
	if status != string(api.OperationStatusRunning) || step != string(api.OperationStepStatusSucceeded) || code != "" || stepCode != "" {
		t.Fatalf("after contention status=%s code=%s netbird=%s netbird_code=%s, want running/succeeded", status, code, step, stepCode)
	}
	for _, peer := range f.peers {
		f.gate.mu.Lock()
		n := f.gate.calls[peer.ID]
		f.gate.mu.Unlock()
		observed, err := f.gate.base.ObservePeer(ctx, "test", peer.ID)
		if err != nil || observed.Authorized || n != 1 {
			t.Fatalf("peer %s final authorized=%v writes=%d err=%v, want one withdrawn state", peer.ID, observed.Authorized, n, err)
		}
	}
}

func TestSyncMemberAdapterFailureFailsSuspend(t *testing.T) {
	f := newSuspendFixture(t)
	ctx := t.Context()
	f.gate.fail = errors.New("TEST ONLY upstream")
	f.gate.open()
	if err := f.svc.SyncMember(ctx, f.member); err == nil {
		t.Fatal("upstream failure was ignored")
	}
	status, code, step, stepCode := operationState(t, f)
	if status != string(api.OperationStatusFailed) || code != "DEPENDENCY_UNAVAILABLE" || step != string(api.OperationStepStatusFailed) || stepCode != "DEPENDENCY_UNAVAILABLE" {
		t.Fatalf("genuine failure status=%s code=%s netbird=%s netbird_code=%s", status, code, step, stepCode)
	}
	f.gate.fail = nil
	if err := f.svc.SyncMember(ctx, f.member); err != nil {
		t.Fatal(err)
	}
	status, code, step, stepCode = operationState(t, f)
	if status != string(api.OperationStatusRunning) || code != "" || step != string(api.OperationStepStatusSucceeded) || stepCode != "" {
		t.Fatalf("recovered status=%s code=%s netbird=%s netbird_code=%s", status, code, step, stepCode)
	}
	for _, peer := range f.peers {
		observed, err := f.gate.base.ObservePeer(ctx, "test", peer.ID)
		if err != nil || observed.Authorized {
			t.Fatalf("peer %s still authorized after recovery (%v)", peer.ID, err)
		}
	}
}
