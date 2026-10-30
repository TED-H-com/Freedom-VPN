//go:build integration

package devices

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestD052_F01DeterministicDowngradeAndBindingReadmission(t *testing.T) {
	const issuer = "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()
	member := env.loginMember(issuer, "portal", "TEST-rank-subject")
	originalGrant := env.grantMember(member, env.revision, 2*time.Hour)
	for i, ip := range []string{"100.64.0.121", "100.64.0.122", "100.64.0.123"} {
		env.addUpstreamPeer("TEST-rank-subject", "TEST-peer-"+string(rune('a'+i)), ip)
	}
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	var oldest uuid.UUID
	if err := env.pool.QueryRow(ctx, `SELECT id FROM member_devices WHERE member_id=$1 ORDER BY authorized_at ASC,id ASC LIMIT 1`, member).Scan(&oldest); err != nil {
		t.Fatal(err)
	}
	low := uuid.New()
	if _, err := env.pool.Exec(ctx, `INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,2,1,$3,1,'{}')`, low, env.plan, []uuid.UUID{env.group}); err != nil {
		t.Fatal(err)
	}
	// Time progresses between durable intervals; historical ownership is retained.
	env.now = env.now.Add(time.Microsecond)
	env.engine.Now = func() time.Time { return env.now }
	if err := env.members.RevokeGrant(ctx, deviceCommand("admin", uuid.New()), member, originalGrant, "TEST ONLY downgrade"); err != nil {
		t.Fatal(err)
	}
	lowGrant := env.grantMember(member, low, time.Hour)
	var survivor uuid.UUID
	if err := env.pool.QueryRow(ctx, `SELECT id FROM member_devices WHERE member_id=$1 AND status='authorized'`, member).Scan(&survivor); err != nil || survivor != oldest {
		t.Fatal("oldest device lost its slot", err)
	}
	var count int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, member).Scan(&count); err != nil || count != 1 {
		t.Fatal("downgrade quota", count, err)
	}
	if err := env.devices.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	env.now = env.now.Add(time.Microsecond)
	if err := env.members.RevokeGrant(ctx, deviceCommand("admin", uuid.New()), member, lowGrant, "TEST ONLY upgrade"); err != nil {
		t.Fatal(err)
	}
	env.grantMember(member, env.revision, time.Hour)
	if err := env.devices.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, member).Scan(&count); err != nil || count != 3 {
		t.Fatal("restored slots not admitted", count, err)
	}
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE member_id=$1 AND binding_revision=1 AND valid_until<=$2`, member, env.now).Scan(&count); err != nil || count != 3 {
		t.Fatal("historical intervals reopened", count, err)
	}
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE member_id=$1 AND binding_revision>1 AND valid_from<=$2 AND valid_until>$2`, member, env.now).Scan(&count); err != nil || count != 3 {
		t.Fatal("new binding revisions missing", count, err)
	}
	var before, after int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE member_id=$1`, member).Scan(&before); err != nil {
		t.Fatal(err)
	}
	// Existing reconciliation remains idempotent after interval restoration.
	env.grantMember(member, env.revision, 30*time.Minute)
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE member_id=$1`, member).Scan(&after); err != nil || after != before {
		t.Fatal("recompute duplicated intervals", before, after, err)
	}
}

func TestD052_F06OverlapIsConflictNotQuota(t *testing.T) {
	const issuer = "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()
	a := env.loginMember(issuer, "portal", "TEST-owner-a")
	b := env.loginMember(issuer, "portal", "TEST-owner-b")
	for _, m := range []uuid.UUID{a, b} {
		env.grantMember(m, env.revision, time.Hour)
	}
	env.addUpstreamPeer("TEST-owner-a", "TEST-peer-a", "100.64.0.124")
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	env.addUpstreamPeer("TEST-owner-b", "TEST-peer-b", "100.64.0.124")
	if _, err := env.adapter.bindUser(ctx, "TEST-owner-b"); err != nil {
		t.Fatal(err)
	}
	_, err := env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", b), "test-deploy", "TEST-peer-b")
	if !errors.Is(err, entitlement.ErrConflict) || errors.Is(err, ErrLimit) {
		t.Fatal("overlap reported as quota", err)
	}
}

func TestD052_F08SQLRejectsFalseProofWithoutAdapterGuard(t *testing.T) {
	env := newD052Env(t, "https://issuer.example.invalid", 3)
	ctx := t.Context()
	member := env.loginMember("https://issuer.example.invalid", "portal", "TEST-proof-subject")
	env.grantMember(member, env.revision, time.Hour)
	env.addUpstreamPeer("TEST-proof-subject", "TEST-peer", "100.64.0.125")
	if _, err := env.adapter.bindUser(ctx, "TEST-proof-subject"); err != nil {
		t.Fatal(err)
	}
	peer, err := env.adapter.ObservePeer(ctx, "test-deploy", "TEST-peer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = env.pool.Exec(ctx, `UPDATE netbird_user_bindings SET mapping_proof='TEST ONLY incorrect proof' WHERE member_id=$1`, member); err != nil {
		t.Fatal(err)
	}
	if _, err = env.devices.AuthorizeDevice(ctx, deviceCommand("system", member), peer); !errors.Is(err, ErrUnmapped) {
		t.Fatal("SQL trusted a stale adapter observation", err)
	}
	var auditCount int
	if err = env.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorize' AND result=$1`, api.AuditEventResultDenied).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatal("denial not committed", err)
	}
}
