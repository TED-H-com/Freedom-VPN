//go:build integration

package leases

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TEST ONLY - NOT A SECRET
func testLeaseSignerAndTrustStore(t *testing.T, now time.Time) (Signer, *envelope.TrustStore) {
	t.Helper()
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - lease-key-1"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)

	signer := desiredstate.NewKeySigner(desiredstate.SigningKey{
		KeyID:      "lease-key-test-1",
		Purpose:    envelope.PurposeLease,
		PrivateKey: priv,
	})
	ts, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID:     "lease-key-test-1",
		PublicKey: pub,
		Purposes:  []envelope.Purpose{envelope.PurposeLease},
		NotBefore: now.Add(-24 * time.Hour),
		NotAfter:  now.Add(365 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("NewTrustStore: %v", err)
	}
	return signer, ts
}

func seedLeaseTestNode(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	groupID uuid.UUID,
	intent api.PublicationIntent,
) (uuid.UUID, uuid.UUID) {
	t.Helper()
	nodeID := uuid.New()
	incID := uuid.New()
	exitID := uuid.New()
	fp := sha256.Sum256([]byte("pubkey-" + nodeID.String()))

	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes (
			id, lifecycle_state, provisioning_state,
			desired_config_generation, applied_config_generation,
			control_status, data_health, safety_status
		) VALUES (
			$1, 'registered', 'ready',
			3, 3,
			'online', 'healthy', 'passed'
		)`, nodeID); err != nil {
		t.Fatalf("insert node: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO node_incarnations (
			id, node_id, public_key_fingerprint, deployment_id, peer_id, os, os_version, arch
		) VALUES (
			$1, $2, $3, 'nb-main', $4, 'ubuntu', '24.04', 'amd64'
		)`, incID, nodeID, fp[:], "peer-"+nodeID.String()[:8]); err != nil {
		t.Fatalf("insert incarnation: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $1 WHERE id = $2`, incID, nodeID); err != nil {
		t.Fatalf("update current_incarnation_id: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO exits (
			id, node_id, display_name, region, publication_intent, publication_observed
		) VALUES (
			$1, $2, 'tw-taipei-lease', 'tw-tpe', $3, 'active'
		)`, exitID, nodeID, string(intent)); err != nil {
		t.Fatalf("insert exit: %v", err)
	}
	if groupID != uuid.Nil {
		if _, err := pool.Exec(ctx, `INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, groupID); err != nil {
			t.Fatalf("insert exit_group: %v", err)
		}
	}
	return nodeID, incID
}

func TestAT29_LeaseMathReplayAndEntitlementImmutability(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

	grpID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'grp-at29', 'grp-at29')`, grpID); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	nodeID, incID := seedLeaseTestNode(t, ctx, pool, grpID, api.PublicationIntentPublished)

	// Seed 3 members in members + entitlements + assignments + member_devices:
	// - memberSooner: effective_service_end = now + 2h (< 24h)
	// - memberLater:  effective_service_end = now + 10d (> 24h)
	// - memberSuspended: admin_suspended = true (must be excluded)
	memberSooner := uuid.New()
	memberLater := uuid.New()
	memberSuspended := uuid.New()
	soonerEnd := now.Add(2 * time.Hour)
	laterEnd := now.Add(10 * 24 * time.Hour)
	validFrom := now.Add(-24 * time.Hour)

	seedMember := func(mid uuid.UUID, email string, end time.Time, rev int, suspended bool) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO members (id, email, display_name, status)
			VALUES ($1, $2, $2, 'active')`, mid, email); err != nil {
			t.Fatalf("insert member %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO entitlements (member_id, revision, device_limit, valid_from, effective_service_end, admin_suspended, computed_at)
			VALUES ($1, $2, 3, $3, $4, $5, $6)`, mid, rev, validFrom, end, suspended, now.Add(-1*time.Hour)); err != nil {
			t.Fatalf("insert entitlement %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO assignments (member_id, group_id, valid_from, valid_until)
			VALUES ($1, $2, $3, $4)`, mid, grpID, validFrom, end); err != nil {
			t.Fatalf("insert assignment %s: %v", email, err)
		}
		devID := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO member_devices (id, member_id, deployment_id, peer_id, name, status, authorized_at)
			VALUES ($1, $2, 'nb-main', $3, 'laptop', 'authorized', $4)`, devID, mid, "peer-"+mid.String()[:8], validFrom); err != nil {
			t.Fatalf("insert device %s: %v", email, err)
		}
	}

	seedMember(memberSooner, "sooner@example.com", soonerEnd, 4, false)
	seedMember(memberLater, "later@example.com", laterEnd, 5, false)
	seedMember(memberSuspended, "suspended@example.com", laterEnd, 6, true)

	signer, ts := testLeaseSignerAndTrustStore(t, now)
	svc := NewService(pool, signer, WithClock(func() time.Time { return now }))

	reqID := uuid.New()
	cfgGen := nodes.ParseU64(3)
	env1, err := svc.Renew(ctx, nodeID, incID, api.LeaseRenewalRequest{
		RenewalRequestId:        reqID,
		NodeId:                  nodeID,
		NodeIncarnationId:       incID,
		AppliedConfigGeneration: &cfgGen,
	})
	if err != nil {
		t.Fatalf("Renew initial: %v", err)
	}

	// Verify signature with pkg/envelope.Verify for PurposeLease.
	raw1, err := json.Marshal(env1)
	if err != nil {
		t.Fatalf("marshal lease envelope: %v", err)
	}
	if _, keyID, err := envelope.Verify(raw1, ts, envelope.PurposeLease, now); err != nil || keyID != "lease-key-test-1" {
		t.Fatalf("envelope.Verify = (key=%q, err=%v), want (lease-key-test-1, nil)", keyID, err)
	}

	wantNodeEnd := now.Add(24 * time.Hour)
	if !env1.Signed.IssuedAt.Equal(now) || !env1.Signed.ExpiresAt.Equal(wantNodeEnd) {
		t.Fatalf("node lease window = [%v, %v), want [%v, %v)", env1.Signed.IssuedAt, env1.Signed.ExpiresAt, now, wantNodeEnd)
	}
	if len(env1.Signed.Members) != 2 {
		t.Fatalf("len(Members) = %d, want 2 (suspended member excluded)", len(env1.Signed.Members))
	}

	for _, m := range env1.Signed.Members {
		switch m.MemberId {
		case memberSooner:
			if !m.LeaseEnd.Equal(soonerEnd) {
				t.Errorf("memberSooner LeaseEnd = %v, want %v (clamped to effective_service_end)", m.LeaseEnd, soonerEnd)
			}
		case memberLater:
			if !m.LeaseEnd.Equal(wantNodeEnd) {
				t.Errorf("memberLater LeaseEnd = %v, want %v (clamped to 24h max_offline_lease)", m.LeaseEnd, wantNodeEnd)
			}
		default:
			t.Errorf("unexpected member %s in lease grants", m.MemberId)
		}
	}

	// Verify entitlements table rows were NOT modified by lease issuance (§16.3).
	var checkEnd time.Time
	var checkRev int
	if err := pool.QueryRow(ctx, `SELECT effective_service_end, revision::int FROM entitlements WHERE member_id = $1`, memberSooner).Scan(&checkEnd, &checkRev); err != nil {
		t.Fatalf("query entitlement: %v", err)
	}
	if !checkEnd.Equal(soonerEnd) || checkRev != 4 {
		t.Fatalf("entitlement row modified: got (%v, rev=%d), want (%v, rev=4)", checkEnd, checkRev, soonerEnd)
	}

	// Advance clock by 30 minutes and replay the exact same renewal_request_id.
	// Must return the exact same signed lease and expires_at without restarting TTL!
	now = now.Add(30 * time.Minute)
	envReplay, err := svc.Renew(ctx, nodeID, incID, api.LeaseRenewalRequest{
		RenewalRequestId:        reqID,
		NodeId:                  nodeID,
		NodeIncarnationId:       incID,
		AppliedConfigGeneration: &cfgGen,
	})
	if err != nil {
		t.Fatalf("Renew replay: %v", err)
	}
	if envReplay.Signed.LeaseId != env1.Signed.LeaseId {
		t.Errorf("replay LeaseId = %s, want %s", envReplay.Signed.LeaseId, env1.Signed.LeaseId)
	}
	if !envReplay.Signed.ExpiresAt.Equal(env1.Signed.ExpiresAt) || !envReplay.Signed.IssuedAt.Equal(env1.Signed.IssuedAt) {
		t.Errorf("replay restarted TTL: got [%v, %v), want [%v, %v)",
			envReplay.Signed.IssuedAt, envReplay.Signed.ExpiresAt, env1.Signed.IssuedAt, env1.Signed.ExpiresAt)
	}
}

func TestLeases_UnpublishedDisabledAndRevokedNode(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 13, 0, 0, 0, time.UTC)

	grpID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'grp-unpub', 'grp-unpub')`, grpID); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	unpubNodeID, unpubIncID := seedLeaseTestNode(t, ctx, pool, grpID, api.PublicationIntentUnpublished)
	disabledNodeID, disabledIncID := seedLeaseTestNode(t, ctx, pool, grpID, api.PublicationIntentDisabled)

	signer, _ := testLeaseSignerAndTrustStore(t, now)
	fakeEnt := NewFakeEntitlementReader(MemberLeaseInput{
		MemberID:            uuid.New(),
		GroupIDs:            []uuid.UUID{grpID},
		EntitlementRevision: 1,
		ValidFrom:           now.Add(-1 * time.Hour),
		EffectiveServiceEnd: now.Add(24 * time.Hour),
		DeviceIDs:           []uuid.UUID{uuid.New()},
	})
	svc := NewService(pool, signer, WithEntitlementReader(fakeEnt), WithClock(func() time.Time { return now }))

	// 1. Unpublished node receives 0 member grants.
	envUnpub, err := svc.Renew(ctx, unpubNodeID, unpubIncID, api.LeaseRenewalRequest{
		RenewalRequestId:  uuid.New(),
		NodeId:            unpubNodeID,
		NodeIncarnationId: unpubIncID,
	})
	if err != nil {
		t.Fatalf("Renew(unpublished): %v", err)
	}
	if len(envUnpub.Signed.Members) != 0 {
		t.Fatalf("unpublished node got %d member grants, want 0", len(envUnpub.Signed.Members))
	}

	// 2. Disabled node receives 0 member grants.
	envDis, err := svc.Renew(ctx, disabledNodeID, disabledIncID, api.LeaseRenewalRequest{
		RenewalRequestId:  uuid.New(),
		NodeId:            disabledNodeID,
		NodeIncarnationId: disabledIncID,
	})
	if err != nil {
		t.Fatalf("Renew(disabled): %v", err)
	}
	if len(envDis.Signed.Members) != 0 {
		t.Fatalf("disabled node got %d member grants, want 0", len(envDis.Signed.Members))
	}

	// 3. Mismatched incarnation -> 403 INCARNATION_MISMATCH.
	_, err = svc.Renew(ctx, unpubNodeID, uuid.New(), api.LeaseRenewalRequest{
		RenewalRequestId:  uuid.New(),
		NodeId:            unpubNodeID,
		NodeIncarnationId: unpubIncID,
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.IncarnationMismatch || ae.HTTPStatus() != 403 {
		t.Fatalf("mismatched incarnation err = %v, want 403 INCARNATION_MISMATCH", err)
	}

	// 4. Revoked node -> 403 IDENTITY_REVOKED.
	if _, err := pool.Exec(ctx, `UPDATE nodes SET lifecycle_state = 'revoked' WHERE id = $1`, unpubNodeID); err != nil {
		t.Fatalf("revoke node: %v", err)
	}
	_, err = svc.Renew(ctx, unpubNodeID, unpubIncID, api.LeaseRenewalRequest{
		RenewalRequestId:  uuid.New(),
		NodeId:            unpubNodeID,
		NodeIncarnationId: unpubIncID,
	})
	if !errors.As(err, &ae) || ae.Code != apierr.IdentityRevoked || ae.HTTPStatus() != 403 {
		t.Fatalf("revoked node err = %v, want 403 IDENTITY_REVOKED", err)
	}
}
