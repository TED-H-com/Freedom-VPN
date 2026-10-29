//go:build integration

package desiredstate

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TEST ONLY - NOT A SECRET
func testDesiredStateSignerAndTrustStore(t *testing.T, now time.Time) (Signer, *envelope.TrustStore) {
	t.Helper()
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - desired-state-key-1"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)

	signer := NewKeySigner(SigningKey{
		KeyID:      "ds-key-test-1",
		Purpose:    envelope.PurposeDesiredState,
		PrivateKey: priv,
	})
	ts, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID:     "ds-key-test-1",
		PublicKey: pub,
		Purposes:  []envelope.Purpose{envelope.PurposeDesiredState},
		NotBefore: now.Add(-24 * time.Hour),
		NotAfter:  now.Add(365 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("NewTrustStore: %v", err)
	}
	return signer, ts
}

func seedCompilerTestNode(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	groupID uuid.UUID,
	intent api.PublicationIntent,
	cap *api.NodeCapacity,
) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	relID := uuid.New()
	manifestHash := sha256.Sum256([]byte("manifest-" + relID.String()))
	sig := make([]byte, 64)
	if _, err := pool.Exec(ctx, `
		INSERT INTO releases (
			id, version, release_sequence, manifest_schema_version,
			manifest, manifest_hash, signature, signing_key_ref, component_versions
		) VALUES (
			$1, $2, 1, 1,
			'{"schema_version":1}'::jsonb, $3, $4, 'rel-key-1', '{"fvpn_agent":"1.0.0"}'::jsonb
		)`, relID, "2026.10.29-"+relID.String()[:8], manifestHash[:], sig); err != nil {
		t.Fatalf("insert release: %v", err)
	}

	nodeID := uuid.New()
	incID := uuid.New()
	exitID := uuid.New()
	fp := sha256.Sum256([]byte("pubkey-" + nodeID.String()))
	var capBytes []byte
	if cap != nil {
		capBytes, _ = json.Marshal(cap)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes (
			id, lifecycle_state, provisioning_state,
			desired_config_generation, applied_config_generation,
			desired_release_id, current_release_id, last_good_release_id,
			control_status, data_health, safety_status, capacity
		) VALUES (
			$1, 'registered', 'ready',
			0, 0,
			$2, $2, $2,
			'online', 'healthy', 'passed', $3
		)`, nodeID, relID, capBytes); err != nil {
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
			$1, $2, 'tw-taipei-ds', 'tw-tpe', $3, 'active'
		)`, exitID, nodeID, string(intent)); err != nil {
		t.Fatalf("insert exit: %v", err)
	}
	if groupID != uuid.Nil {
		if _, err := pool.Exec(ctx, `INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, groupID); err != nil {
			t.Fatalf("insert exit_group: %v", err)
		}
	}
	return nodeID, incID, exitID, relID
}

func TestCompiler_MonotonicGenerationRevocationSuspensionGroupFilterAndRates(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)

	grpA := uuid.New()
	grpB := uuid.New()
	for _, g := range []struct {
		id   uuid.UUID
		code string
	}{{grpA, "grp-a"}, {grpB, "grp-b"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, $2, $3)`, g.id, g.code, g.code); err != nil {
			t.Fatalf("insert group: %v", err)
		}
	}

	capDown := int64(40_000_000)
	capUp := int64(20_000_000)
	nodeCap := &api.NodeCapacity{
		DownBps:          1_000_000_000,
		UpBps:            1_000_000_000,
		MemberDownBpsCap: &capDown,
		MemberUpBpsCap:   &capUp,
	}
	nodeID, incID, _, relID := seedCompilerTestNode(t, ctx, pool, grpA, api.PublicationIntentPublished, nodeCap)

	signer, ts := testDesiredStateSignerAndTrustStore(t, now)
	entReader := NewFakeEntitlementReader()

	fupRevID := uuid.New()
	dnsRevID := uuid.New()
	relPolRevID := uuid.New()
	polReader := NewFakePolicyReader(PolicySnapshot{
		AuthorizationGeneration:  4,
		FairUseGeneration:        3,
		DNSGeneration:            5,
		DNSResolverIPv4:          []string{"100.64.0.53"},
		DNSResolverIPv6:          []string{"2001:db8::53"},
		DesiredReleaseID:         &relID,
		SupportedReleasePolicyID: relPolRevID,
		ServingReleaseIDs:        []uuid.UUID{relID},
		PolicyRefs:               []uuid.UUID{fupRevID, dnsRevID, relPolRevID},
		AccountingPeriod:         DefaultAccountingPeriod(now),
	})

	member1ID := uuid.New()
	dev1AuthID := uuid.New()
	dev1RevokedID := uuid.New()
	member2OtherGroupID := uuid.New()
	dev2ID := uuid.New()
	member3SuspendedID := uuid.New()
	dev3ID := uuid.New()
	clampedAt := now

	entReader.SetMembers(
		MemberEntitlement{
			MemberID:            member1ID,
			GroupIDs:            []uuid.UUID{grpA},
			Suspended:           false,
			ValidFrom:           now.Add(-24 * time.Hour),
			EffectiveServiceEnd: now.Add(7 * 24 * time.Hour),
			PlanDownBps:         100_000_000,
			PlanUpBps:           100_000_000,
			FairUseTierDownBps:  50_000_000,
			FairUseTierUpBps:    25_000_000,
			TierIndex:           1,
			Priority:            2,
			FairUsePolicyGen:    3,
			Mode:                api.MemberRatePolicyModeEnforce,
			Reason:              api.MemberRatePolicyReasonFairUseTier,
			Bindings: []MemberDeviceBinding{
				{
					DeviceID:        dev1AuthID,
					NetbirdPeerID:   "nb-peer-dev1",
					AllowedIPv4:     []string{"100.64.0.10/32"},
					AllowedIPv6:     []string{},
					BindingRevision: 4,
					ValidFrom:       now.Add(-1 * time.Hour),
					ValidUntil:      now.Add(7 * 24 * time.Hour),
					Revoked:         false,
				},
				{
					DeviceID:        dev1RevokedID,
					NetbirdPeerID:   "nb-peer-revoked",
					AllowedIPv4:     []string{"100.64.0.11/32"},
					AllowedIPv6:     []string{},
					BindingRevision: 5,
					ValidFrom:       now.Add(-1 * time.Hour),
					ValidUntil:      now.Add(7 * 24 * time.Hour),
					Revoked:         true, // Must be omitted immediately!
				},
			},
		},
		MemberEntitlement{
			MemberID:            member2OtherGroupID,
			GroupIDs:            []uuid.UUID{grpB}, // Not in grpA -> must be excluded!
			ValidFrom:           now.Add(-24 * time.Hour),
			EffectiveServiceEnd: now.Add(7 * 24 * time.Hour),
			PlanDownBps:         100_000_000,
			PlanUpBps:           100_000_000,
			Bindings: []MemberDeviceBinding{
				{
					DeviceID:        dev2ID,
					NetbirdPeerID:   "nb-peer-dev2",
					AllowedIPv4:     []string{"100.64.0.20/32"},
					BindingRevision: 1,
					ValidFrom:       now.Add(-1 * time.Hour),
					ValidUntil:      now.Add(7 * 24 * time.Hour),
				},
			},
		},
		MemberEntitlement{
			MemberID:            member3SuspendedID,
			GroupIDs:            []uuid.UUID{grpA},
			Suspended:           true, // Suspended -> valid_until clamped to <= clampedAt!
			ClampedUntil:        &clampedAt,
			ValidFrom:           now.Add(-24 * time.Hour),
			EffectiveServiceEnd: now.Add(7 * 24 * time.Hour),
			PlanDownBps:         100_000_000,
			PlanUpBps:           100_000_000,
			Bindings: []MemberDeviceBinding{
				{
					DeviceID:        dev3ID,
					NetbirdPeerID:   "nb-peer-dev3",
					AllowedIPv4:     []string{"100.64.0.30/32"},
					BindingRevision: 2,
					ValidFrom:       now.Add(-1 * time.Hour),
					ValidUntil:      now.Add(7 * 24 * time.Hour),
				},
			},
		},
	)

	compiler := NewCompiler(
		pool,
		signer,
		WithEntitlementReader(entReader),
		WithPolicyReader(polReader),
		WithClock(func() time.Time { return now }),
	)

	// First compile -> generation 1.
	res1, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatalf("Compile #1: %v", err)
	}
	if res1.Generation != 1 {
		t.Fatalf("res1.Generation = %d, want 1", res1.Generation)
	}

	// Verify signature over canonical JSON with pkg/envelope.Verify.
	envRaw, err := json.Marshal(res1.Envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if _, keyID, err := envelope.Verify(envRaw, ts, envelope.PurposeDesiredState, now); err != nil || keyID != "ds-key-test-1" {
		t.Fatalf("envelope.Verify = (key=%q, err=%v), want (ds-key-test-1, nil)", keyID, err)
	}

	signed := res1.Envelope.Signed
	if signed.NodeId != nodeID || signed.NodeIncarnationId != incID {
		t.Errorf("signed identity = (%s, %s), want (%s, %s)", signed.NodeId, signed.NodeIncarnationId, nodeID, incID)
	}
	if signed.PublicationIntent != api.PublicationIntentPublished {
		t.Errorf("PublicationIntent = %s, want published", signed.PublicationIntent)
	}

	// Verify device bindings:
	// - dev1AuthID present
	// - dev1RevokedID omitted
	// - dev2ID (grpB) omitted
	// - dev3ID (suspended member) has valid_until clamped to <= now
	foundDev1 := false
	foundDev3 := false
	for _, b := range signed.DeviceBindings {
		if b.DeviceId == dev1RevokedID {
			t.Errorf("revoked device %s must not appear in device_bindings", dev1RevokedID)
		}
		if b.DeviceId == dev2ID {
			t.Errorf("member device %s outside exit's member_group_ids must not appear in device_bindings", dev2ID)
		}
		if b.DeviceId == dev3ID {
			foundDev3 = true
			if b.ValidUntil.After(now) {
				t.Errorf("suspended member device %s has ValidUntil=%v > now=%v", dev3ID, b.ValidUntil, now)
			}
		}
		if b.DeviceId == dev1AuthID {
			foundDev1 = true
			if uint64(b.BindingRevision) != 4 || b.PeerId != "nb-peer-dev1" {
				t.Errorf("unexpected dev1 binding: %+v", b)
			}
		}
	}
	if !foundDev1 {
		t.Fatalf("expected authorized device %s in device_bindings", dev1AuthID)
	}
	if !foundDev3 {
		t.Fatalf("expected suspended member device %s with clamped ValidUntil in device_bindings", dev3ID)
	}

	// Verify effective_rate = min(100M, 50M, 40M) = 40M down and min(100M, 25M, 20M) = 20M up.
	foundPolicy := false
	for _, mp := range signed.MemberRatePolicies {
		if mp.MemberId == member1ID {
			foundPolicy = true
			if mp.DownBps == nil || *mp.DownBps != 40_000_000 || mp.UpBps == nil || *mp.UpBps != 20_000_000 {
				t.Errorf("member1 EffectiveRate = (down=%v, up=%v), want (40_000_000, 20_000_000)", mp.DownBps, mp.UpBps)
			}
		}
	}
	if !foundPolicy {
		t.Fatalf("expected member1 in MemberRatePolicies")
	}

	// Second compile -> generation monotonically increases to 2.
	res2, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatalf("Compile #2: %v", err)
	}
	if res2.Generation != 2 {
		t.Fatalf("res2.Generation = %d, want 2", res2.Generation)
	}
}

func TestCompiler_ReSignETag304AndIdentityMismatch(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)

	grpID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'grp-resign', 'grp-resign')`, grpID); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	nodeID, incID, _, _ := seedCompilerTestNode(t, ctx, pool, grpID, api.PublicationIntentPublished, nil)

	signer, _ := testDesiredStateSignerAndTrustStore(t, now)
	compiler := NewCompiler(
		pool,
		signer,
		WithEntitlementReader(NewFakeEntitlementReader()),
		WithPolicyReader(NewFakePolicyReader(PolicySnapshot{})),
		WithClock(func() time.Time { return now }),
	)

	// 1. Initial GetDesiredState compiles generation 1 on the fly if none exists yet.
	get1, err := compiler.GetDesiredState(ctx, nodeID, incID, "")
	if err != nil {
		t.Fatalf("GetDesiredState initial: %v", err)
	}
	if get1.NotModified || get1.Envelope == nil || get1.ETag == "" {
		t.Fatalf("unexpected GetDesiredState initial: %+v", get1)
	}
	if uint64(get1.Envelope.Signed.ConfigGeneration) != 1 {
		t.Fatalf("ConfigGeneration = %s, want 1", get1.Envelope.Signed.ConfigGeneration)
	}

	// 2. GetDesiredState with matching If-None-Match returns NotModified (304).
	get304, err := compiler.GetDesiredState(ctx, nodeID, incID, get1.ETag)
	if err != nil {
		t.Fatalf("GetDesiredState(If-None-Match): %v", err)
	}
	if !get304.NotModified {
		t.Fatalf("expected NotModified=true when If-None-Match matches %q", get1.ETag)
	}

	// 3. Advance clock and call ReSign -> generation stays 1, but validity window, signature, and ETag change (§14.7)!
	now = now.Add(2 * time.Hour)
	reSigned, err := compiler.ReSign(ctx, nodeID)
	if err != nil {
		t.Fatalf("ReSign: %v", err)
	}
	if reSigned.Generation != 1 {
		t.Fatalf("ReSign changed generation to %d, want 1", reSigned.Generation)
	}
	if reSigned.ETag == get1.ETag {
		t.Fatalf("expected ReSign to change ETag even when generation stays 1, got %q", reSigned.ETag)
	}

	// Old ETag no longer returns 304; new ETag returns 304.
	getAfterReSign, err := compiler.GetDesiredState(ctx, nodeID, incID, get1.ETag)
	if err != nil {
		t.Fatalf("GetDesiredState with old ETag: %v", err)
	}
	if getAfterReSign.NotModified || getAfterReSign.ETag != reSigned.ETag {
		t.Fatalf("expected modified response with new ETag %q, got %+v", reSigned.ETag, getAfterReSign)
	}
	getNew304, err := compiler.GetDesiredState(ctx, nodeID, incID, reSigned.ETag)
	if err != nil || !getNew304.NotModified {
		t.Fatalf("expected 304 with new ETag, got (%+v, %v)", getNew304, err)
	}

	// 4. Identity mismatch (wrong incarnation_id) -> 403 INCARNATION_MISMATCH.
	_, err = compiler.GetDesiredState(ctx, nodeID, uuid.New(), "")
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.IncarnationMismatch || ae.HTTPStatus() != 403 {
		t.Fatalf("wrong incarnation err = %v, want 403 INCARNATION_MISMATCH", err)
	}

	// 5. Revoked node -> 403 IDENTITY_REVOKED.
	if _, err := pool.Exec(ctx, `UPDATE nodes SET lifecycle_state = 'revoked' WHERE id = $1`, nodeID); err != nil {
		t.Fatalf("revoke node: %v", err)
	}
	_, err = compiler.GetDesiredState(ctx, nodeID, incID, "")
	if !errors.As(err, &ae) || ae.Code != apierr.IdentityRevoked || ae.HTTPStatus() != 403 {
		t.Fatalf("revoked node err = %v, want 403 IDENTITY_REVOKED", err)
	}
}

func TestCompiler_RecompileBatchCoalescesBurst(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 16, 0, 0, 0, time.UTC)

	grpID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'grp-burst', 'grp-burst')`, grpID); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	nodeID, _, _, _ := seedCompilerTestNode(t, ctx, pool, grpID, api.PublicationIntentPublished, nil)

	signer, _ := testDesiredStateSignerAndTrustStore(t, now)
	compiler := NewCompiler(
		pool,
		signer,
		WithEntitlementReader(NewFakeEntitlementReader()),
		WithPolicyReader(NewFakePolicyReader(PolicySnapshot{})),
		WithClock(func() time.Time { return now }),
	)

	nodePayload, _ := json.Marshal(map[string]any{"node_id": nodeID.String()})
	groupPayload, _ := json.Marshal(map[string]any{"group_ids": []string{grpID.String()}})
	fleetPayload, _ := json.Marshal(map[string]any{"reason": "fleet-wide"})

	burst := []db.ClaimedEvent{
		{ID: uuid.New(), AggregateID: nodeID, EventType: EventNodeUpdated, Payload: nodePayload},
		{ID: uuid.New(), AggregateID: nodeID, EventType: EventNodePublicationChanged, Payload: nodePayload},
		{ID: uuid.New(), AggregateID: uuid.New(), EventType: EventMemberEntitlementChanged, Payload: groupPayload},
		{ID: uuid.New(), AggregateID: uuid.New(), EventType: EventDeviceAuthorized, Payload: groupPayload},
		{ID: uuid.New(), AggregateID: uuid.New(), EventType: EventDeviceRevoked, Payload: groupPayload},
		{ID: uuid.New(), AggregateID: uuid.New(), EventType: EventMemberSuspended, Payload: groupPayload},
		{ID: uuid.New(), AggregateID: uuid.New(), EventType: EventPolicyActivated, Payload: fleetPayload},
		{ID: uuid.New(), AggregateID: uuid.New(), EventType: EventReleaseTargetChanged, Payload: fleetPayload},
	}

	results, err := compiler.RecompileBatch(ctx, burst)
	if err != nil {
		t.Fatalf("RecompileBatch: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("RecompileBatch compiled %d times, want 1 coalesced compile for node %s", len(results), nodeID)
	}
	if results[0].Envelope.Signed.NodeId != nodeID || results[0].Generation != 1 {
		t.Fatalf("RecompileBatch result = %+v, want NodeID=%s Generation=1", results[0], nodeID)
	}
}
