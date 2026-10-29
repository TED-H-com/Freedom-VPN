//go:build integration

package desiredstate

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
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
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $1, resource_version = resource_version + 1 WHERE id = $2`, incID, nodeID); err != nil {
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
		MemberDownBpsCap: presence.NewNullableWithValue(capDown),
		MemberUpBpsCap:   presence.NewNullableWithValue(capUp),
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
	if !signed.Update.IsNull() {
		t.Fatal("desired-state without an update must sign an explicit null")
	}
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
			if (!mp.DownBps.IsSpecified() || mp.DownBps.IsNull()) || nullableValue(t, mp.DownBps) != 40_000_000 || (!mp.UpBps.IsSpecified() || mp.UpBps.IsNull()) || nullableValue(t, mp.UpBps) != 20_000_000 {
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

	// 5. Revoked node -> 403 IDENTITY_REVOKED for both GetDesiredState and ReSign.
	if _, err := pool.Exec(ctx, `UPDATE nodes SET lifecycle_state = 'revoked', resource_version = resource_version + 1 WHERE id = $1`, nodeID); err != nil {
		t.Fatalf("revoke node: %v", err)
	}
	_, err = compiler.GetDesiredState(ctx, nodeID, incID, "")
	if !errors.As(err, &ae) || ae.Code != apierr.IdentityRevoked || ae.HTTPStatus() != 403 {
		t.Fatalf("revoked node GetDesiredState err = %v, want 403 IDENTITY_REVOKED", err)
	}
	_, err = compiler.ReSign(ctx, nodeID)
	if !errors.As(err, &ae) || ae.Code != apierr.IdentityRevoked || ae.HTTPStatus() != 403 {
		t.Fatalf("revoked node ReSign err = %v, want 403 IDENTITY_REVOKED", err)
	}
}

func TestCompiler_DBReaders_ReleaseNull_FUPShadowEnforce_DNSProfile_AndMissingPlanRate(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)

	grpID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'grp-db', 'grp-db')`, grpID); err != nil {
		t.Fatalf("insert group: %v", err)
	}

	capDown := int64(60_000_000)
	capUp := int64(30_000_000)
	nodeCap := &api.NodeCapacity{
		DownBps:          1_000_000_000,
		UpBps:            1_000_000_000,
		MemberDownBpsCap: presence.NewNullableWithValue(capDown),
		MemberUpBpsCap:   presence.NewNullableWithValue(capUp),
	}
	capBytes, _ := json.Marshal(nodeCap)

	nodeID := uuid.New()
	incID := uuid.New()
	exitID := uuid.New()
	fp := sha256.Sum256([]byte("pubkey-" + nodeID.String()))

	// 1. Insert node with NO desired_release_id (and fleet_config has no desired_release_id).
	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes (
			id, lifecycle_state, provisioning_state,
			desired_config_generation, applied_config_generation,
			control_status, data_health, safety_status, capacity
		) VALUES (
			$1, 'registered', 'ready',
			0, 0,
			'online', 'healthy', 'passed', $2
		)`, nodeID, capBytes); err != nil {
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
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $1, resource_version = resource_version + 1 WHERE id = $2`, incID, nodeID); err != nil {
		t.Fatalf("update current_incarnation_id: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO exits (
			id, node_id, display_name, region, publication_intent, publication_observed
		) VALUES (
			$1, $2, 'tw-taipei-db', 'tw-tpe', 'published', 'active'
		)`, exitID, nodeID); err != nil {
		t.Fatalf("insert exit: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, grpID); err != nil {
		t.Fatalf("insert exit_group: %v", err)
	}

	signer, ts := testDesiredStateSignerAndTrustStore(t, now)
	// Use default DBEntitlementReader and DBPolicyReader (no fakes) with configured deployment DNS resolvers.
	compiler := NewCompiler(
		pool,
		signer,
		WithClock(func() time.Time { return now }),
		WithDNSResolvers([]string{"192.0.2.53"}, []string{"2001:db8::53"}),
	)

	// Compile #1: no release target -> must succeed with DesiredReleaseId = null.
	resNoRel, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatalf("Compile without release target: %v", err)
	}
	if !resNoRel.Envelope.Signed.DesiredReleaseId.IsNull() {
		t.Fatalf("expected DesiredReleaseId to be null when no release target exists, got %v", resNoRel.Envelope.Signed.DesiredReleaseId)
	}

	// 2. Seed release, dns_profile, fair_use policy revisions, plans, members, devices, and fup_member_states.
	relID := uuid.New()
	manifestHash := sha256.Sum256([]byte("manifest-" + relID.String()))
	if _, err := pool.Exec(ctx, `
		INSERT INTO releases (
			id, version, release_sequence, manifest_schema_version,
			manifest, manifest_hash, signature, signing_key_ref, component_versions
		) VALUES (
			$1, $2, 1, 1,
			'{"schema_version":1}'::jsonb, $3, $4, 'rel-key-1', '{"fvpn_agent":"1.0.0"}'::jsonb
		)`, relID, "2026.10.28-"+relID.String()[:8], manifestHash[:], make([]byte, 64)); err != nil {
		t.Fatalf("insert release: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE nodes SET desired_release_id = $2, resource_version = resource_version + 1 WHERE id = $1`, nodeID, relID); err != nil {
		t.Fatalf("set node desired_release_id: %v", err)
	}

	insertActivePolicy := func(typ string, rev int, payload string) uuid.UUID {
		defID := uuid.New()
		revID := uuid.New()
		sum := sha256.Sum256([]byte(payload))
		if _, err := pool.Exec(ctx, `INSERT INTO policy_definitions (id, type, code, generation) VALUES ($1, $2, $3, $4)`, defID, typ, typ+"-"+defID.String()[:8], rev); err != nil {
			t.Fatalf("insert policy_definition %s: %v", typ, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO policy_revisions (id, definition_id, type, revision, schema_version, payload, payload_hash, signing_key_ref, created_by)
			VALUES ($1, $2, $3, $4, 1, $5::jsonb, $6, 'key-1', 'admin')`,
			revID, defID, typ, rev, payload, sum[:]); err != nil {
			t.Fatalf("insert policy_revision %s: %v", typ, err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE policy_definitions
			SET active_revision_id = $2, resource_version = resource_version + 1
			WHERE id = $1`, defID, revID); err != nil {
			t.Fatalf("activate policy_definition %s: %v", typ, err)
		}
		return revID
	}

	// Payload includes a decoy resolver_ipv4 to verify resolver addresses come only from compiler config.
	dnsRevID := insertActivePolicy(string(api.PolicyTypeDnsProfile), 9, `{"resolver_ipv4":["198.51.100.53"],"resolver_ipv6":[]}`)
	fupEnforceRev := insertActivePolicy(string(api.PolicyTypeFairUse), 6, `{"mode":"enforce","effective_at":"2026-10-01T00:00:00Z"}`)

	// Before any published head exists in dns_publication_heads, DnsGeneration must be 0 even with an active dns_profile revision.
	compiledBeforeDNSHead, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatalf("Compile before dns_publication_heads: %v", err)
	}
	if compiledBeforeDNSHead.Envelope.Signed.PolicyRefs.DnsGeneration != 0 {
		t.Fatalf("DnsGeneration without dns_publication_heads = %s, want 0", compiledBeforeDNSHead.Envelope.Signed.PolicyRefs.DnsGeneration)
	}

	// Publish an active head in dns_publication_heads at policy_generation = 9.
	var dnsDefID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT definition_id FROM policy_revisions WHERE id = $1`, dnsRevID).Scan(&dnsDefID); err != nil {
		t.Fatalf("select dns definition_id: %v", err)
	}
	dnsDocBytes := []byte(`{"schema_version":1}`)
	dnsDocSum := fmt.Sprintf("%x", sha256.Sum256(dnsDocBytes))
	if _, err := pool.Exec(ctx, `
		INSERT INTO dns_resolver_documents (
			profile_revision_id, definition_id, profile_code, policy_generation, document_sha256, document, canonical_bytes
		) VALUES ($1, $2, 'dns-main', 9, $3, $4::jsonb, $5)`,
		dnsRevID, dnsDefID, dnsDocSum, string(dnsDocBytes), dnsDocBytes); err != nil {
		t.Fatalf("insert dns_resolver_documents: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO dns_publication_heads (
			definition_id, profile_revision_id, policy_generation, document_sha256, status
		) VALUES ($1, $2, 9, $3, 'active')`,
		dnsDefID, dnsRevID, dnsDocSum); err != nil {
		t.Fatalf("insert dns_publication_heads: %v", err)
	}

	// Also insert a future fair_use revision (effective_at > now) on a second definition for before-effective_at testing.
	futureDefID := uuid.New()
	futureRevID := uuid.New()
	futurePayload := `{"mode":"enforce","effective_at":"2026-11-01T00:00:00Z"}`
	futureSum := sha256.Sum256([]byte(futurePayload))
	if _, err := pool.Exec(ctx, `INSERT INTO policy_definitions (id, type, code, generation) VALUES ($1, 'fair_use', $2, 5)`, futureDefID, "fup-future-"+futureDefID.String()[:8]); err != nil {
		t.Fatalf("insert future fup def: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO policy_revisions (id, definition_id, type, revision, schema_version, payload, payload_hash, signing_key_ref, created_by)
		VALUES ($1, $2, 'fair_use', 5, 1, $3::jsonb, $4, 'key-1', 'admin')`,
		futureRevID, futureDefID, futurePayload, futureSum[:]); err != nil {
		t.Fatalf("insert future fup rev: %v", err)
	}

	seedPlanRevision := func(code string, payload string) uuid.UUID {
		planID := uuid.New()
		planRevID := uuid.New()
		if _, err := pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, $2)`, planID, code+"-"+planID.String()[:8]); err != nil {
			t.Fatalf("insert plan %s: %v", code, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload)
			VALUES ($1, $2, 1, 3, $3, 1, $4::jsonb)`,
			planRevID, planID, []uuid.UUID{grpID}, payload); err != nil {
			t.Fatalf("insert plan_revision %s: %v", code, err)
		}
		return planRevID
	}

	planWithRate := seedPlanRevision("plan-100m", `{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}`)
	planWithoutRate := seedPlanRevision("plan-norate", `{}`)

	seedMemberWithBinding := func(email string, planRevID uuid.UUID, ip string) uuid.UUID {
		mid := uuid.New()
		devID := uuid.New()
		validFrom := now.Add(-24 * time.Hour)
		validUntil := now.Add(14 * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO members (id, email, display_name, status) VALUES ($1, $2, $2, 'active')`, mid, email); err != nil {
			t.Fatalf("insert member %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO entitlements (
				member_id, revision, state, plan_revision_id, device_limit,
				valid_from, effective_service_end, admin_suspended, projection
			) VALUES (
				$1, 2, 'active', $2, 3,
				$3, $4, false, jsonb_build_object('exit_group_ids', to_jsonb(ARRAY[$5::text]))
			)`, mid, planRevID, validFrom, validUntil, grpID.String()); err != nil {
			t.Fatalf("insert entitlement %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO member_devices (id, member_id, deployment_id, peer_id, status, authorized_at)
			VALUES ($1, $2, 'nb-main', $3, 'authorized', $4)`, devID, mid, "peer-"+devID.String()[:8], validFrom); err != nil {
			t.Fatalf("insert device %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO device_bindings (device_id, member_id, overlay_scope, overlay_ip, binding_revision, valid_from, valid_until, peer_ref)
			VALUES ($1, $2, 'overlay-main', $3, 1, $4, $5, $6)`,
			devID, mid, ip, validFrom, validUntil, "peer-"+devID.String()[:8]); err != nil {
			t.Fatalf("insert binding %s: %v", email, err)
		}
		return mid
	}

	memberShadow := seedMemberWithBinding("shadow@example.com", planWithRate, "100.64.0.41")
	memberBeforeEffective := seedMemberWithBinding("before-eff@example.com", planWithRate, "100.64.0.42")
	memberEnforce := seedMemberWithBinding("enforce@example.com", planWithRate, "100.64.0.43")
	memberMissingRate := seedMemberWithBinding("norate@example.com", planWithoutRate, "100.64.0.44")

	// Seed fup_member_states for period 2026-10:
	// - memberShadow: mode=shadow, would_apply_down_bps=1000, generation=4 -> must compile mode=shadow with null tier rate (min(100M, 60M) = 60M).
	if _, err := pool.Exec(ctx, `
		INSERT INTO fup_member_states (
			member_id, period_id, period_start, period_end, policy_revision_id,
			mode, would_apply_tier_index, would_apply_down_bps, would_apply_up_bps,
			would_apply_priority, would_apply_reason, generation, aggregate_as_of
		) VALUES (
			$1, '2026-10', '2026-10-01T00:00:00Z', '2026-11-01T00:00:00Z', $2,
			'shadow', 2, 1000, 1000,
			3, 'fair_use_tier', 4, $3
		)`, memberShadow, fupEnforceRev, now); err != nil {
		t.Fatalf("insert fup shadow state: %v", err)
	}

	// - memberBeforeEffective: mode=enforce on futureRevID (effective_at = 2026-11-01 > now) -> must compile mode=shadow with null tier rate.
	if _, err := pool.Exec(ctx, `
		INSERT INTO fup_member_states (
			member_id, period_id, period_start, period_end, policy_revision_id,
			mode, would_apply_tier_index, would_apply_down_bps, would_apply_up_bps,
			would_apply_priority, would_apply_reason,
			enforced_state, enforced_tier_index, enforced_down_bps, enforced_up_bps,
			enforced_priority, enforced_reason, generation, aggregate_as_of
		) VALUES (
			$1, '2026-10', '2026-10-01T00:00:00Z', '2026-11-01T00:00:00Z', $2,
			'enforce', 1, 5000000, 2000000,
			2, 'fair_use_tier',
			'adjusted', 1, 5000000, 2000000,
			2, 'fair_use_tier', 5, $3
		)`, memberBeforeEffective, futureRevID, now); err != nil {
		t.Fatalf("insert fup before-effective state: %v", err)
	}

	// - memberEnforce: mode=enforce on fupEnforceRev (effective_at <= now), enforced_down_bps=15M, enforced_up_bps=8M, generation=7.
	if _, err := pool.Exec(ctx, `
		INSERT INTO fup_member_states (
			member_id, period_id, period_start, period_end, policy_revision_id,
			mode, would_apply_tier_index, would_apply_down_bps, would_apply_up_bps,
			would_apply_priority, would_apply_reason,
			enforced_state, enforced_tier_index, enforced_down_bps, enforced_up_bps,
			enforced_priority, enforced_reason, generation, aggregate_as_of
		) VALUES (
			$1, '2026-10', '2026-10-01T00:00:00Z', '2026-11-01T00:00:00Z', $2,
			'enforce', 2, 15000000, 8000000,
			4, 'fair_use_tier',
			'adjusted', 2, 15000000, 8000000,
			4, 'fair_use_tier', 7, $3
		)`, memberEnforce, fupEnforceRev, now); err != nil {
		t.Fatalf("insert fup enforce state: %v", err)
	}

	compiled, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatalf("Compile with DB readers: %v", err)
	}

	rawEnv, err := json.Marshal(compiled.Envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if _, _, err := envelope.Verify(rawEnv, ts, envelope.PurposeDesiredState, now); err != nil {
		t.Fatalf("envelope.Verify: %v", err)
	}

	signed := compiled.Envelope.Signed
	if gotRel, err := signed.DesiredReleaseId.Get(); err != nil || gotRel != relID {
		t.Fatalf("DesiredReleaseId = (%v, %v), want %s", gotRel, err, relID)
	}
	if signed.PolicyRefs.DnsGeneration != 9 {
		t.Errorf("DnsGeneration = %s, want 9 (active dns_profile revision)", signed.PolicyRefs.DnsGeneration)
	}
	if signed.PolicyRefs.FupGeneration != 6 {
		t.Errorf("FupGeneration = %s, want 6 (active fair_use revision)", signed.PolicyRefs.FupGeneration)
	}
	if len(signed.Dns.ResolverIpv4) != 1 || signed.Dns.ResolverIpv4[0] != "192.0.2.53" {
		t.Errorf("ResolverIpv4 = %v, want [192.0.2.53] from compiler config (never from dns_profile payload)", signed.Dns.ResolverIpv4)
	}
	if len(signed.Dns.ResolverIpv6) != 1 || signed.Dns.ResolverIpv6[0] != "2001:db8::53" {
		t.Errorf("ResolverIpv6 = %v, want [2001:db8::53] from compiler config", signed.Dns.ResolverIpv6)
	}

	// Verify memberMissingRate was omitted and recorded in compiled.OmittedMembers.
	if len(compiled.OmittedMembers) != 1 || compiled.OmittedMembers[0].MemberID != memberMissingRate || compiled.OmittedMembers[0].Reason != "plan_missing_rate" {
		t.Fatalf("OmittedMembers = %+v, want [{MemberID:%s Reason:plan_missing_rate}]", compiled.OmittedMembers, memberMissingRate)
	}
	for _, b := range signed.DeviceBindings {
		if b.MemberId == memberMissingRate {
			t.Fatalf("memberMissingRate (%s) must be omitted from DeviceBindings", memberMissingRate)
		}
	}

	if len(signed.MemberRatePolicies) != 3 {
		t.Fatalf("len(MemberRatePolicies) = %d, want 3 (memberMissingRate omitted)", len(signed.MemberRatePolicies))
	}

	byMember := map[uuid.UUID]api.MemberRatePolicy{}
	for _, mp := range signed.MemberRatePolicies {
		byMember[mp.MemberId] = mp
	}

	// 1. memberShadow: mode=shadow, generation=4, tier_index=0, down=60M (min(plan 100M, cap 60M)), up=30M (min(plan 50M, cap 30M)).
	if mp, ok := byMember[memberShadow]; !ok {
		t.Fatalf("memberShadow missing from MemberRatePolicies")
	} else {
		down := nullableValue(t, mp.DownBps)
		up := nullableValue(t, mp.UpBps)
		if mp.Mode != api.MemberRatePolicyModeShadow || mp.Generation != 4 || mp.TierIndex != 0 || down != 60_000_000 || up != 30_000_000 {
			t.Errorf("memberShadow rate policy = %+v (down=%d up=%d), want mode=shadow gen=4 tier=0 down=60000000 up=30000000", mp, down, up)
		}
	}

	// 2. memberBeforeEffective: before effective_at -> mode=shadow, generation=5, tier_index=0, down=60M, up=30M.
	if mp, ok := byMember[memberBeforeEffective]; !ok {
		t.Fatalf("memberBeforeEffective missing from MemberRatePolicies")
	} else {
		down := nullableValue(t, mp.DownBps)
		up := nullableValue(t, mp.UpBps)
		if mp.Mode != api.MemberRatePolicyModeShadow || mp.Generation != 5 || mp.TierIndex != 0 || down != 60_000_000 || up != 30_000_000 {
			t.Errorf("memberBeforeEffective rate policy = %+v (down=%d up=%d), want mode=shadow gen=5 tier=0 down=60000000 up=30000000", mp, down, up)
		}
	}

	// 3. memberEnforce: mode=enforce, generation=7, tier_index=2, priority=4, reason=fair_use_tier, down=15M, up=8M.
	if mp, ok := byMember[memberEnforce]; !ok {
		t.Fatalf("memberEnforce missing from MemberRatePolicies")
	} else {
		down := nullableValue(t, mp.DownBps)
		up := nullableValue(t, mp.UpBps)
		if mp.Mode != api.MemberRatePolicyModeEnforce || mp.Generation != 7 || mp.TierIndex != 2 || mp.Priority != 4 || mp.Reason != api.MemberRatePolicyReasonFairUseTier || down != 15_000_000 || up != 8_000_000 {
			t.Errorf("memberEnforce rate policy = %+v (down=%d up=%d), want mode=enforce gen=7 tier=2 prio=4 reason=fair_use_tier down=15000000 up=8000000", mp, down, up)
		}
	}

	// 4. Verify ReSign refuses a node whose incarnation is revoked or whose lifecycle_state is retired.
	if _, err := pool.Exec(ctx, `UPDATE node_incarnations SET revoked_at = $2 WHERE id = $1`, incID, now); err != nil {
		t.Fatalf("revoke incarnation: %v", err)
	}
	_, err = compiler.ReSign(ctx, nodeID)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.IdentityRevoked || ae.HTTPStatus() != 403 {
		t.Fatalf("ReSign with revoked incarnation err = %v, want 403 IDENTITY_REVOKED", err)
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

func nullableValue[T any](t *testing.T, n interface{ Get() (T, error) }) T {
	t.Helper()
	v, err := n.Get()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
