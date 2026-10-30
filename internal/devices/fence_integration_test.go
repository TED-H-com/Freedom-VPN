//go:build integration

package devices

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

func exitGroup(t *testing.T, ctx context.Context, s *Service) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT exit_group_ids[1] FROM plan_revisions LIMIT 1`).Scan(&id); err != nil || id == uuid.Nil {
		t.Fatal(err, id)
	}
	return id
}

func countWhere(t *testing.T, ctx context.Context, s *Service, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := s.Pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func externalAudits(t *testing.T, ctx context.Context, s *Service, device uuid.UUID) int {
	t.Helper()
	return countWhere(t, ctx, s, `SELECT count(*) FROM audit_events WHERE action=$1 AND resource=$2`, externalWithdrawalAudit, string(api.ResourceTypeMemberDevice)+":"+device.String())
}

func fenceRows(t *testing.T, ctx context.Context, s *Service, device uuid.UUID) int {
	t.Helper()
	return countWhere(t, ctx, s, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, device)
}

func authorizeObserved(t *testing.T, ctx context.Context, s *Service, member uuid.UUID, peer Peer) uuid.UUID {
	t.Helper()
	change, err := s.AuthorizeDevice(ctx, deviceCommand("system", member), peer)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SyncDevice(ctx, change.DeviceID); err != nil {
		t.Fatal(err)
	}
	return change.DeviceID
}

func withdrawPeer(t *testing.T, s *Service, peer Peer) {
	t.Helper()
	peer.Authorized = false
	peer.Groups = nil
	s.Adapter.(*fakeAdapter).setPeer(peer)
}

func TestFence_RestoredObservationDoesNotRegrant(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	withdrawPeer(t, s, peers[0])
	if err := s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	if adapter.grantCount() != before {
		t.Fatalf("restored authorization granted: %d -> %d", before, adapter.grantCount())
	}
	observed, err := adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || observed.Authorized {
		t.Fatal(observed, err)
	}
	if fenceRows(t, ctx, s, id) != 1 || externalAudits(t, ctx, s, id) != 1 {
		t.Fatalf("fence=%d audit=%d", fenceRows(t, ctx, s, id), externalAudits(t, ctx, s, id))
	}
	var peer string
	var groups []string
	if err = s.Pool.QueryRow(ctx, `SELECT after_diff->>'peer_id', ARRAY(SELECT jsonb_array_elements_text(after_diff->'observed_groups')) FROM audit_events WHERE action=$1 AND resource=$2`, externalWithdrawalAudit, string(api.ResourceTypeMemberDevice)+":"+id.String()).Scan(&peer, &groups); err != nil {
		t.Fatal(err)
	}
	if peer != peers[0].ID {
		t.Fatal(peer)
	}
	if err = s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	if externalAudits(t, ctx, s, id) != 1 || adapter.grantCount() != before || fenceRows(t, ctx, s, id) != 1 {
		t.Fatalf("second sync re-audited or granted: audits=%d grants=%d", externalAudits(t, ctx, s, id), adapter.grantCount())
	}
}

func TestFence_TransientWithdrawalDoesNotFence(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	group := exitGroup(t, ctx, s)
	adapter := s.Adapter.(*fakeAdapter)
	withdrawn := peers[0]
	withdrawn.Authorized = false
	confirmed := peers[0]
	confirmed.Authorized = true
	confirmed.Groups = []uuid.UUID{group}
	adapter.mu.Lock()
	adapter.script = []Peer{withdrawn, confirmed}
	adapter.mu.Unlock()
	before := adapter.grantCount()
	if err := s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	if adapter.grantCount() != before+1 || fenceRows(t, ctx, s, id) != 0 || externalAudits(t, ctx, s, id) != 0 {
		t.Fatalf("grants=%d fence=%d audit=%d", adapter.grantCount(), fenceRows(t, ctx, s, id), externalAudits(t, ctx, s, id))
	}
	observed, err := adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || !observed.Authorized {
		t.Fatal(observed, err)
	}
}

func TestFence_FreshGrantAuthorizesUnconfirmedPeer(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	change, err := s.AuthorizeDevice(ctx, deviceCommand("system", member), peers[0])
	if err != nil {
		t.Fatal(err)
	}
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	if err = s.SyncDevice(ctx, change.DeviceID); err != nil {
		t.Fatal(err)
	}
	observed, err := adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || !observed.Authorized || adapter.grantCount() != before+1 {
		t.Fatal(observed, adapter.grantCount(), err)
	}
	if fenceRows(t, ctx, s, change.DeviceID) != 0 || externalAudits(t, ctx, s, change.DeviceID) != 0 {
		t.Fatal("fresh grant was fenced")
	}
}

func TestFence_CentralRevokeStaysWithdrawn(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	if _, err := s.RevokeDevice(ctx, deviceCommand("member", member), id); err != nil {
		t.Fatal(err)
	}
	withdrawPeer(t, s, peers[0])
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	if err := s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	observed, err := adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || observed.Authorized || adapter.grantCount() != before {
		t.Fatal(observed, adapter.grantCount(), err)
	}
	if fenceRows(t, ctx, s, id) != 0 || externalAudits(t, ctx, s, id) != 0 {
		t.Fatalf("central revoke fenced: fence=%d audit=%d", fenceRows(t, ctx, s, id), externalAudits(t, ctx, s, id))
	}
}

func TestFence_ReauthorizationClearsAndGrants(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	withdrawPeer(t, s, peers[0])
	if err := s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	if fenceRows(t, ctx, s, id) != 1 {
		t.Fatal("expected a fence to clear")
	}
	if _, err := s.AuthorizeDevice(ctx, deviceCommand("admin", uuid.New()), peers[0]); err != nil {
		t.Fatal(err)
	}
	if fenceRows(t, ctx, s, id) != 0 {
		t.Fatal("re-authorization left the fence")
	}
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	if err := s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	observed, err := adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || !observed.Authorized || adapter.grantCount() != before+1 || fenceRows(t, ctx, s, id) != 0 {
		t.Fatal(observed, adapter.grantCount(), err)
	}
}

func TestFence_QuotaRevokeAuditStillGrants(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	err := db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return entitlement.Audit(ctx, tx, entitlement.Actor{ID: audit.SystemActor.ID, Role: string(audit.SystemActor.Type)}, "device.revoke", api.ResourceTypeMemberDevice, id, map[string]any{"status": "authorized"}, map[string]any{"status": "pending", "reason": "device_limit_reached"})
	})
	if err != nil {
		t.Fatal(err)
	}
	withdrawPeer(t, s, peers[0])
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	if err = s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	if adapter.grantCount() != before+1 || fenceRows(t, ctx, s, id) != 0 || externalAudits(t, ctx, s, id) != 0 {
		t.Fatalf("quota audit blocked a current authorization: grants=%d fence=%d", adapter.grantCount(), fenceRows(t, ctx, s, id))
	}
}

func TestFence_PermanentRevokeAuditDoesNotRegrant(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	err := db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return entitlement.Audit(ctx, tx, entitlement.Actor{ID: audit.SystemActor.ID, Role: string(audit.SystemActor.Type)}, "device.revoke", api.ResourceTypeMemberDevice, id, map[string]any{"status": "authorized"}, map[string]any{"status": "revoked"})
	})
	if err != nil {
		t.Fatal(err)
	}
	withdrawPeer(t, s, peers[0])
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	if err = s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	if adapter.grantCount() != before || externalAudits(t, ctx, s, id) != 0 || fenceRows(t, ctx, s, id) != 1 {
		t.Fatalf("grants=%d external=%d fence=%d", adapter.grantCount(), externalAudits(t, ctx, s, id), fenceRows(t, ctx, s, id))
	}
	if err = s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	observed, err := adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || observed.Authorized || adapter.grantCount() != before {
		t.Fatal(observed, adapter.grantCount(), err)
	}
}

func TestFence_RestoreReplayLeavesPeerWithdrawn(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	id := authorizeObserved(t, ctx, s, member, peers[0])
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	captureDeviceSnapshot(t, ctx, conn, id)
	if _, err = s.RevokeDevice(ctx, deviceCommand("member", member), id); err != nil {
		t.Fatal(err)
	}
	if err = s.SyncDevice(ctx, id); err != nil {
		t.Fatal(err)
	}
	observed, err := s.Adapter.(*fakeAdapter).ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || observed.Authorized {
		t.Fatal("revoke sync left the peer authorized", observed, err)
	}
	restoreDeviceSnapshot(t, ctx, conn, id)
	var status string
	var observedAuthorized bool
	if err = s.Pool.QueryRow(ctx, `SELECT d.status, o.observed_authorized FROM member_devices d JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.id=$1`, id).Scan(&status, &observedAuthorized); err != nil || status != "authorized" || !observedAuthorized {
		t.Fatal(status, observedAuthorized, err)
	}
	if _, err = s.Pool.Exec(ctx, `UPDATE usage_ledger_state SET mode='frozen', mode_reason='restore drill', changed_by='test' WHERE id=$1`, usageLedgerStateID); err != nil {
		t.Fatal(err)
	}
	reg := prometheus.NewRegistry()
	if err = s.RegisterRestoreFences(reg); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	s.Logger = slog.New(slog.NewTextHandler(&logs, nil))
	adapter := s.Adapter.(*fakeAdapter)
	before := adapter.grantCount()
	if err = s.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	observed, err = adapter.ObservePeer(ctx, peers[0].DeploymentID, peers[0].ID)
	if err != nil || observed.Authorized || adapter.grantCount() != before {
		t.Fatal(observed, adapter.grantCount(), err)
	}
	if fenceRows(t, ctx, s, id) != 1 || externalAudits(t, ctx, s, id) != 1 {
		t.Fatalf("fence=%d audit=%d", fenceRows(t, ctx, s, id), externalAudits(t, ctx, s, id))
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var metric float64
	var found bool
	for _, family := range families {
		if family.GetName() != "fvpn_device_restore_fence_decisions_total" {
			continue
		}
		found = true
		for _, sample := range family.GetMetric() {
			metric += sample.GetCounter().GetValue()
		}
	}
	if !found || metric != 1 || !bytes.Contains(logs.Bytes(), []byte("device authorization unconfirmed during ledger restore")) || !bytes.Contains(logs.Bytes(), []byte(id.String())) {
		t.Fatalf("restore summary missing: found=%v metric=%v log=%s", found, metric, logs.String())
	}
}

func captureDeviceSnapshot(t *testing.T, ctx context.Context, conn *pgxpool.Conn, device uuid.UUID) {
	t.Helper()
	resource := string(api.ResourceTypeMemberDevice) + ":" + device.String()
	statements := []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE t0_member_devices AS SELECT * FROM member_devices WHERE id=$1`, []any{device}},
		{`CREATE TEMP TABLE t0_device_bindings AS SELECT * FROM device_bindings WHERE device_id=$1`, []any{device}},
		{`CREATE TEMP TABLE t0_device_observations AS SELECT * FROM device_authorization_observations WHERE device_id=$1`, []any{device}},
		{`CREATE TEMP TABLE t0_operations AS SELECT * FROM operations WHERE resource_type='member_device' AND resource_id=$1`, []any{device}},
		{`CREATE TEMP TABLE t0_audit_events AS SELECT * FROM audit_events WHERE resource=$1`, []any{resource}},
		{`CREATE TEMP TABLE t0_idempotency AS SELECT * FROM idempotency_requests WHERE resource_id=$1 OR operation_id IN (SELECT id FROM operations WHERE resource_type='member_device' AND resource_id=$1)`, []any{device}},
	}
	for _, step := range statements {
		if _, err := conn.Exec(ctx, step.sql, step.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// restoreDeviceSnapshot replays the captured images. Application triggers forbid
// deleting member_devices and clearing revoked_at; a point-in-time restore
// replaces the images without those triggers. DISABLE TRIGGER USER is the
// table-owner equivalent and leaves foreign keys enforced.
func restoreDeviceSnapshot(t *testing.T, ctx context.Context, conn *pgxpool.Conn, device uuid.UUID) {
	t.Helper()
	resource := string(api.ResourceTypeMemberDevice) + ":" + device.String()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('fvpn.restore','on',true)`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"idempotency_requests", "audit_events", "operations", "device_bindings", "device_authorization_observations", "member_devices"} {
		if _, err = tx.Exec(ctx, `ALTER TABLE `+table+` DISABLE TRIGGER USER`); err != nil {
			t.Fatal(table, err)
		}
	}
	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM idempotency_requests WHERE resource_id=$1 OR operation_id IN (SELECT id FROM operations WHERE resource_type='member_device' AND resource_id=$1)`, []any{device}},
		{`DELETE FROM audit_events WHERE resource=$2 OR operation_id IN (SELECT id FROM operations WHERE resource_type='member_device' AND resource_id=$1)`, []any{device, resource}},
		{`DELETE FROM operations WHERE resource_type='member_device' AND resource_id=$1`, []any{device}},
		{`DELETE FROM device_authorization_observations WHERE device_id=$1`, []any{device}},
		{`DELETE FROM device_bindings WHERE device_id=$1`, []any{device}},
		{`DELETE FROM member_devices WHERE id=$1`, []any{device}},
		{`INSERT INTO member_devices SELECT * FROM t0_member_devices`, nil},
		{`INSERT INTO device_bindings SELECT * FROM t0_device_bindings`, nil},
		{`INSERT INTO device_authorization_observations SELECT * FROM t0_device_observations`, nil},
		{`INSERT INTO operations SELECT * FROM t0_operations`, nil},
		{`INSERT INTO audit_events SELECT * FROM t0_audit_events`, nil},
		{`INSERT INTO idempotency_requests SELECT * FROM t0_idempotency`, nil},
	}
	for _, step := range steps {
		if _, err = tx.Exec(ctx, step.sql, step.args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"idempotency_requests", "audit_events", "operations", "device_bindings", "device_authorization_observations", "member_devices"} {
		if _, err = tx.Exec(ctx, `ALTER TABLE `+table+` ENABLE TRIGGER USER`); err != nil {
			t.Fatal(table, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
