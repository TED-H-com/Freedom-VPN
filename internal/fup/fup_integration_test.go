//go:build integration

package fup

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func strPtr(s string) *string { return &s }

func TestFUP02_AT19_PostgresShadowToEnforceAndAppendOnlyGuards(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	policyStore := policies.NewStore(pool)
	engine := NewEngine(pool, nil, nil, nil)

	adminID := uuid.New()
	memberID := uuid.New()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	periodID := "2026-10"
	pStart, pEnd, err := PeriodBoundsForUTCMonth(periodID)
	if err != nil {
		t.Fatalf("PeriodBoundsForUTCMonth: %v", err)
	}

	// Insert member.
	_, err = pool.Exec(ctx, `
		INSERT INTO members (id, email, display_name, status)
		VALUES ($1, 'member1@example.com', 'Test Member', 'active')
	`, memberID)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}

	// Insert usage_rollups row for 2026-10 with 150 GB (Tier 1).
	_, err = pool.Exec(ctx, `
		INSERT INTO usage_rollups (
			member_id, period, granularity, projection_generation, metering_version,
			upload_bytes, download_bytes, last_observed_at, aggregate_as_of, period_start, period_end
		) VALUES ($1, $2, 'month', 1, 'forward_admission_v1', 50000000000, 100000000000, $3, $3, $4, $5)
	`, memberID, periodID, now, pStart, pEnd)
	if err != nil {
		t.Fatalf("insert usage_rollups: %v", err)
	}

	// 1. Create and activate a shadow fair_use policy revision via policies.Store.
	shadowPayload := testFairUsePolicy(api.FairUsePayloadModeShadow)
	shadowBytes, err := json.Marshal(shadowPayload)
	if err != nil {
		t.Fatalf("marshal shadow payload: %v", err)
	}
	shadowRev, err := policyStore.CreateRevision(ctx, policies.CreateRevisionInput{
		Type:          api.PolicyTypeFairUse,
		Code:          strPtr("default-fup"),
		SchemaVersion: 1,
		Payload:       shadowBytes,
		Reason:        "initial shadow rollout",
		ActorID:       adminID,
	})
	if err != nil {
		t.Fatalf("CreateRevision shadow: %v", err)
	}
	_, err = policyStore.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    shadowRev.DefinitionID,
		RevisionID:      shadowRev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "activate shadow FUP",
		ActorID:         adminID,
	})
	if err != nil {
		t.Fatalf("ActivateRevision shadow: %v", err)
	}

	// Run Compute in shadow mode -> records would_apply_tier_index=1, generation=0, no FUP outbox event.
	evalShadow, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Compute shadow: %v", err)
	}
	if evalShadow.WouldApplyTierIndex != 1 || evalShadow.Generation != 0 || evalShadow.GenerationIncremented {
		t.Fatalf("unexpected shadow eval: tier=%d gen=%d inc=%v",
			evalShadow.WouldApplyTierIndex, evalShadow.Generation, evalShadow.GenerationIncremented)
	}

	var fupOutboxCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'fup.member_generation_changed'
	`).Scan(&fupOutboxCount)
	if err != nil || fupOutboxCount != 0 {
		t.Fatalf("expected 0 fup outbox events in shadow mode, got %d (err=%v)", fupOutboxCount, err)
	}

	// 2. Create and activate an enforce fair_use policy revision.
	enforcePayload := testFairUsePolicy(api.FairUsePayloadModeEnforce)
	enforceBytes, err := json.Marshal(enforcePayload)
	if err != nil {
		t.Fatalf("marshal enforce payload: %v", err)
	}
	defID := shadowRev.DefinitionID
	enforceRev, err := policyStore.CreateRevision(ctx, policies.CreateRevisionInput{
		DefinitionID:  &defID,
		Type:          api.PolicyTypeFairUse,
		SchemaVersion: 1,
		Payload:       enforceBytes,
		Reason:        "promote FUP to enforce mode",
		ActorID:       adminID,
	})
	if err != nil {
		t.Fatalf("CreateRevision enforce: %v", err)
	}
	_, err = policyStore.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    defID,
		RevisionID:      enforceRev.RevisionID,
		ExpectedVersion: "2",
		Reason:          "activate enforce FUP",
		ActorID:         adminID,
	})
	if err != nil {
		t.Fatalf("ActivateRevision enforce: %v", err)
	}

	// Run Compute in enforce mode -> produces generation 1 and 1 outbox event.
	evalEnforce1, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Compute enforce 1: %v", err)
	}
	if evalEnforce1.Generation != 1 || !evalEnforce1.GenerationIncremented || evalEnforce1.EnforcedTierIndex != 1 {
		t.Fatalf("unexpected enforce eval 1: gen=%d inc=%v tier=%d",
			evalEnforce1.Generation, evalEnforce1.GenerationIncremented, evalEnforce1.EnforcedTierIndex)
	}

	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'fup.member_generation_changed'
	`).Scan(&fupOutboxCount)
	if err != nil || fupOutboxCount != 1 {
		t.Fatalf("expected 1 fup outbox event after enforce transition, got %d (err=%v)", fupOutboxCount, err)
	}

	// Recompute with unchanged tier -> generation stays 1, no duplicate outbox event (AT19).
	evalEnforce2, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Compute enforce 2: %v", err)
	}
	if evalEnforce2.Generation != 1 || evalEnforce2.GenerationIncremented {
		t.Fatalf("expected idempotent generation 1, got gen=%d inc=%v", evalEnforce2.Generation, evalEnforce2.GenerationIncremented)
	}
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'fup.member_generation_changed'
	`).Scan(&fupOutboxCount)
	if err != nil || fupOutboxCount != 1 {
		t.Fatalf("expected outbox count to remain 1, got %d (err=%v)", fupOutboxCount, err)
	}

	// Verify DB guards: fup_evaluations is append-only, and fup_member_states.generation is monotonic.
	if _, err := pool.Exec(ctx, `DELETE FROM fup_evaluations WHERE id = $1`, evalEnforce2.EvaluationID); err == nil {
		t.Fatalf("expected guard_append_only to reject DELETE on fup_evaluations")
	}
	if _, err := pool.Exec(ctx, `UPDATE fup_member_states SET generation = 0 WHERE id = $1`, evalEnforce2.StateID); err == nil {
		t.Fatalf("expected monotonic_fields trigger to reject decreasing generation on fup_member_states")
	}
}

func TestFUP03_FUP04_PostgresOverridesRelaxationAndNodeAck(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	policyStore := policies.NewStore(pool)
	engine := NewEngine(pool, nil, nil, nil)

	adminID := uuid.New()
	memberID := uuid.New()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	periodID := "2026-10"
	pStart, pEnd, _ := PeriodBoundsForUTCMonth(periodID)

	// Insert member.
	_, err := pool.Exec(ctx, `
		INSERT INTO members (id, email, display_name, status)
		VALUES ($1, 'member2@example.com', 'Override Member', 'active')
	`, memberID)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}

	// Insert 2 registered serving nodes with incarnations.
	node1ID, inc1ID := uuid.New(), uuid.New()
	node2ID, inc2ID := uuid.New(), uuid.New()
	fp1 := make([]byte, 32)
	fp1[0] = 1
	fp2 := make([]byte, 32)
	fp2[0] = 2

	for _, n := range []struct {
		nodeID uuid.UUID
		incID  uuid.UUID
		fp     []byte
		peerID string
	}{
		{node1ID, inc1ID, fp1, "peer-1"},
		{node2ID, inc2ID, fp2, "peer-2"},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO nodes (id, lifecycle_state, provisioning_state, applied_config_generation)
			VALUES ($1, 'registered', 'ready', 0)
		`, n.nodeID)
		if err != nil {
			t.Fatalf("insert node: %v", err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO node_incarnations (id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch)
			VALUES ($1, $2, $3, 'dep-1', $4, 'ubuntu', 'amd64')
		`, n.incID, n.nodeID, n.fp, n.peerID)
		if err != nil {
			t.Fatalf("insert node_incarnation: %v", err)
		}
	}

	// Activate enforce fair_use policy.
	enforceBytes, _ := json.Marshal(testFairUsePolicy(api.FairUsePayloadModeEnforce))
	rev, err := policyStore.CreateRevision(ctx, policies.CreateRevisionInput{
		Type:          api.PolicyTypeFairUse,
		Code:          strPtr("enforce-fup"),
		SchemaVersion: 1,
		Payload:       enforceBytes,
		Reason:        "enforce FUP for test",
		ActorID:       adminID,
	})
	if err != nil {
		t.Fatalf("CreateRevision: %v", err)
	}
	_, err = policyStore.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev.DefinitionID,
		RevisionID:      rev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "activate enforce FUP",
		ActorID:         adminID,
	})
	if err != nil {
		t.Fatalf("ActivateRevision: %v", err)
	}

	// Insert usage_rollups with 2^53 + 1 bytes (9007199254740993) -> Tier 2.
	_, err = pool.Exec(ctx, `
		INSERT INTO usage_rollups (
			member_id, period, granularity, projection_generation, metering_version,
			upload_bytes, download_bytes, last_observed_at, aggregate_as_of, period_start, period_end
		) VALUES ($1, $2, 'month', 1, 'forward_admission_v1', 9007199254740993, 0, $3, $3, $4, $5)
	`, memberID, periodID, now, pStart, pEnd)
	if err != nil {
		t.Fatalf("insert usage_rollups: %v", err)
	}

	eval1, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Compute tier 2: %v", err)
	}
	if eval1.EnforcedTierIndex != 2 || eval1.Generation != 1 || eval1.TotalBytes != decstr.U64(9007199254740993) {
		t.Fatalf("expected tier 2 gen 1 total 2^53+1, got tier=%d gen=%d total=%s",
			eval1.EnforcedTierIndex, eval1.Generation, eval1.TotalBytes)
	}

	// Check status before nodes ACK generation 1 -> "pending on 2 nodes", Applying=true.
	stPending, err := engine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatalf("GetAdminMemberStatus: %v", err)
	}
	if !stPending.MemberStatus.Applying || stPending.NodeSyncSummary != "pending on 2 nodes" {
		t.Fatalf("expected Applying=true 'pending on 2 nodes', got applying=%v summary=%q",
			stPending.MemberStatus.Applying, stPending.NodeSyncSummary)
	}

	// Both nodes report applied_policy_refs.fup_generation = "1" in node_observations.
	obsHash := make([]byte, 32)
	for _, n := range []struct {
		nodeID uuid.UUID
		incID  uuid.UUID
	}{
		{node1ID, inc1ID},
		{node2ID, inc2ID},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO node_observations (
				node_id, incarnation_id, report_id, payload_hash, observed_at,
				applied_generation, schema_version, checks, control_status, data_health, safety_status
			) VALUES (
				$1, $2, $3, $4, $5,
				1, 1, '{"applied_policy_refs": {"fup_generation": "1"}}'::jsonb, 'online', 'healthy', 'passed'
			)
		`, n.nodeID, n.incID, uuid.New(), obsHash, now.Add(10*time.Second))
		if err != nil {
			t.Fatalf("insert node_observations: %v", err)
		}
	}

	stApplied, err := engine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatalf("GetAdminMemberStatus after ACK: %v", err)
	}
	if stApplied.MemberStatus.Applying || stApplied.NodeSyncSummary != "applied on 2 nodes" {
		t.Fatalf("expected Applying=false 'applied on 2 nodes', got applying=%v summary=%q",
			stApplied.MemberStatus.Applying, stApplied.NodeSyncSummary)
	}

	// Admin sets a time-limited override (FUP04) -> Generation 2.
	ov, evalOv, err := engine.SetOverride(ctx, SetOverrideInput{
		MemberID:   memberID,
		DownBps:    150_000_000,
		UpBps:      75_000_000,
		Priority:   0,
		ValidFrom:  now.Add(time.Minute),
		ValidUntil: now.Add(30 * time.Minute),
		Reason:     "temporary bandwidth grant for customer verification",
		ActorID:    adminID,
		Now:        now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("SetOverride: %v", err)
	}
	if evalOv.Generation != 2 || evalOv.EnforcedState != EnforcedStateOverride {
		t.Fatalf("expected generation 2 and override state, got gen=%d state=%s", evalOv.Generation, evalOv.EnforcedState)
	}

	// Admin revokes the override early -> Generation 3, back to Tier 2.
	evalRevoked, err := engine.RevokeOverride(ctx, RevokeOverrideInput{
		OverrideID: ov.ID,
		MemberID:   memberID,
		ActorID:    adminID,
		Reason:     "verification completed early",
		Now:        now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("RevokeOverride: %v", err)
	}
	if evalRevoked.Generation != 3 || evalRevoked.EnforcedState != EnforcedStateAdjusted || evalRevoked.EnforcedTierIndex != 2 {
		t.Fatalf("expected generation 3 back to tier 2 after revoke, got gen=%d state=%s tier=%d",
			evalRevoked.Generation, evalRevoked.EnforcedState, evalRevoked.EnforcedTierIndex)
	}

	// Insert a negative usage_adjustment and new rollup projection (10 GB -> Tier 0) -> audited relaxation to Tier 0 (Gen 4).
	_, err = pool.Exec(ctx, `
		INSERT INTO usage_adjustments (
			member_id, period, interval_start, interval_end, upload_bytes, download_bytes, reason, actor
		) VALUES ($1, $2, $3, $4, -9000000000000000, 0, 'billing ledger correction', $5)
	`, memberID, periodID, pStart, pEnd, adminID.String())
	if err != nil {
		t.Fatalf("insert usage_adjustments: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO usage_rollups (
			member_id, period, granularity, projection_generation, metering_version,
			upload_bytes, download_bytes, last_observed_at, aggregate_as_of, period_start, period_end
		) VALUES ($1, $2, 'month', 2, 'forward_admission_v1', 10000000000, 0, $3, $3, $4, $5)
	`, memberID, periodID, now.Add(6*time.Minute), pStart, pEnd)
	if err != nil {
		t.Fatalf("insert corrected usage_rollups: %v", err)
	}

	evalRelaxed, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Compute after negative adjustment: %v", err)
	}
	if !evalRelaxed.Relaxed || evalRelaxed.EnforcedTierIndex != 0 || evalRelaxed.Generation != 4 {
		t.Fatalf("expected relaxed=true tier=0 gen=4, got relaxed=%v tier=%d gen=%d",
			evalRelaxed.Relaxed, evalRelaxed.EnforcedTierIndex, evalRelaxed.Generation)
	}
}
