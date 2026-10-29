//go:build integration

package devices

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AT02/AT04: finite binding renewal, independent observations and expiry denial
// continue to work under the base F-05/F-14 device guards.
func TestAT04_FiniteBindingRenewalAndExpiredObservation(t *testing.T) {
	s, member, peers := setupDevices(t)
	ctx := context.Background()
	command := deviceCommand("system", member)
	change, err := s.AuthorizeDevice(ctx, command, peers[0])
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.AuthorizeDevice(ctx, command, peers[0])
	if err != nil || !replay.Replayed || replay.DeviceID != change.DeviceID {
		t.Fatal(replay, err)
	}
	var before, renewed time.Time
	if err = s.Pool.QueryRow(ctx, `SELECT valid_until FROM device_bindings WHERE device_id=$1`, change.DeviceID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	now := s.Entitlements.Time()
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var locked uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&locked); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO entitlement_grants(member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) SELECT member_id,'admin_grant',$2,$3,'TEST ONLY renewal','test-admin',plan_revision_id FROM entitlements WHERE member_id=$1`, member, now, before.Add(time.Hour)); err != nil {
			return err
		}
		_, err := s.Entitlements.Recompute(ctx, tx, member)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Pool.QueryRow(ctx, `SELECT valid_until FROM device_bindings WHERE device_id=$1`, change.DeviceID).Scan(&renewed); err != nil || !renewed.Equal(before.Add(time.Hour)) {
		t.Fatal(renewed, err)
	}
	if err = s.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	d, err := s.Get(ctx, command.Actor, change.DeviceID)
	// System actors cannot read member views; use the member's authenticated view.
	if err == nil {
		t.Fatal("system actor gained member view", d)
	}
	d, err = s.Get(ctx, deviceCommand("member", member).Actor, change.DeviceID)
	if err != nil || !d.ObservedAuthorized || d.Version != 1 {
		t.Fatal(d, err)
	}
	s.Entitlements.Now = func() time.Time { return renewed.Add(time.Second) }
	if err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error { _, e := s.Entitlements.Recompute(ctx, tx, member); return e }); err != nil {
		t.Fatal(err)
	}
	if err = s.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	d, err = s.Get(ctx, deviceCommand("member", member).Actor, change.DeviceID)
	if err != nil || d.ObservedAuthorized || d.Version != 1 || d.Status != "authorized" {
		t.Fatal(d, err)
	}
	var incomplete int
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM idempotency_requests WHERE response_status IS NULL OR (resource_id IS NULL AND operation_id IS NULL)`).Scan(&incomplete); err != nil || incomplete != 0 {
		t.Fatal(incomplete, err)
	}
	// Prove the base authorization guard remains in force on expired devices.
	if _, err = s.Pool.Exec(ctx, `UPDATE member_devices SET name='TEST ONLY',resource_version=resource_version+1 WHERE id=$1`, change.DeviceID); err == nil {
		t.Fatal("expired authorized device UPDATE accepted")
	}
}

func TestAT02_BlankPeerIdentifiersDoNotWrite(t *testing.T) {
	s, member, peers := setupDevices(t)
	for _, field := range []string{"deployment", "peer", "scope", "user"} {
		p := peers[0]
		switch field {
		case "deployment":
			p.DeploymentID = " \t"
		case "peer":
			p.ID = " \t"
		case "scope":
			p.OverlayScope = " \t"
		case "user":
			p.ExternalUserID = " \t"
		}
		if _, err := s.AuthorizeDevice(context.Background(), deviceCommand("system", member), p); err != entitlement.ErrInvalid {
			t.Fatal(field, err)
		}
	}
	var n int
	if err := s.Pool.QueryRow(context.Background(), `SELECT count(*) FROM idempotency_requests`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}
