package fup

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type fakeUsageReader struct {
	mu        sync.Mutex
	snapshots map[string]UsageSnapshot
	calls     int
}

func newFakeUsageReader() *fakeUsageReader {
	return &fakeUsageReader{
		snapshots: make(map[string]UsageSnapshot),
	}
}

func (f *fakeUsageReader) set(snap UsageSnapshot) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshots[stateKey(snap.MemberID, snap.PeriodID)] = snap
}

func (f *fakeUsageReader) GetMemberPeriodUsage(_ context.Context, memberID uuid.UUID, periodID string) (UsageSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if snap, ok := f.snapshots[stateKey(memberID, periodID)]; ok {
		return snap, nil
	}
	pStart, pEnd, _ := PeriodBoundsForUTCMonth(periodID)
	return UsageSnapshot{
		MemberID:        memberID,
		PeriodID:        periodID,
		PeriodStart:     pStart,
		PeriodEnd:       pEnd,
		TotalBytes:      0,
		AggregateAsOf:   time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC),
		MeteringVersion: string(api.MeteringVersionForwardAdmissionV1),
	}, nil
}

type fakePolicyReader struct {
	mu     sync.Mutex
	policy MemberPolicyContext
}

func (f *fakePolicyReader) GetMemberFairUsePolicy(_ context.Context, memberID uuid.UUID, _ time.Time) (MemberPolicyContext, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := f.policy
	cp.MemberID = memberID
	return cp, nil
}

type fakeAckReader struct {
	mu     sync.Mutex
	status NodeAckStatus
}

func (f *fakeAckReader) GetMemberNodeAckStatus(_ context.Context, _ uuid.UUID, _ decstr.U64) (NodeAckStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status, nil
}

// testFairUsePolicy returns a TEST ONLY fair_use payload with 3 tiers.
func testFairUsePolicy(mode api.FairUsePayloadMode) api.FairUsePayload {
	en0 := "Standard"
	en1 := "Fair-Use Tier 1"
	en2 := "Fair-Use Tier 2"
	// TEST ONLY — NOT PRODUCTION RATES OR THRESHOLDS
	p := api.FairUsePayload{
		EffectiveAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectivePeriod: api.FairUsePayloadEffectivePeriodImmediate,
		MinimumRate: api.RatePair{
			DownBps: 5_000_000,
			UpBps:   2_000_000,
		},
		Mode:      mode,
		Overrides: []api.FairUseOverride{},
		Tiers: []api.FairUseTier{
			{
				ThresholdBytes: 0,
				DownBps:        100_000_000,
				UpBps:          50_000_000,
				Priority:       1,
				DisplayName:    api.LocalizedText{ZhTW: "標準速率", En: &en0},
			},
			{
				ThresholdBytes: 100_000_000_000, // 100 GB (TEST ONLY)
				DownBps:        25_000_000,
				UpBps:          10_000_000,
				Priority:       3,
				DisplayName:    api.LocalizedText{ZhTW: "公平使用第一階", En: &en1},
			},
			{
				ThresholdBytes: 9007199254740993, // 2^53 + 1 bytes (TEST ONLY)
				DownBps:        10_000_000,
				UpBps:          5_000_000,
				Priority:       5,
				DisplayName:    api.LocalizedText{ZhTW: "公平使用第二階", En: &en2},
			},
		},
	}
	p.AccountingBasis.Measure = api.FairUsePayloadAccountingBasisMeasureTotalBytes
	p.AccountingBasis.MeteringVersion = api.MeteringVersionForwardAdmissionV1
	p.AccountingBasis.PeriodType = api.FairUsePayloadAccountingBasisPeriodTypeUtcCalendarMonth
	return p
}

func int64Ptr(v int64) *int64 { return &v }
func intPtr(v int) *int       { return &v }

func TestFUP01_TierSelectionBoundariesAndLargeTotals(t *testing.T) {
	policy := testFairUsePolicy(api.FairUsePayloadModeEnforce)
	tiers := policy.Tiers

	cases := []struct {
		name      string
		total     decstr.U64
		wantIndex int
		wantDown  int64
	}{
		{
			name:      "zero_bytes_tier_0",
			total:     0,
			wantIndex: 0,
			wantDown:  100_000_000,
		},
		{
			name:      "one_byte_below_tier_1_threshold",
			total:     99_999_999_999,
			wantIndex: 0,
			wantDown:  100_000_000,
		},
		{
			name:      "exact_equality_with_tier_1_threshold_enters_tier_1",
			total:     100_000_000_000,
			wantIndex: 1,
			wantDown:  25_000_000,
		},
		{
			name:      "2_pow_53_one_byte_below_2_pow_53_plus_1_stays_tier_1",
			total:     9007199254740992, // 2^53
			wantIndex: 1,
			wantDown:  25_000_000,
		},
		{
			name:      "exact_2_pow_53_plus_1_enters_tier_2",
			total:     9007199254740993, // 2^53 + 1
			wantIndex: 2,
			wantDown:  10_000_000,
		},
		{
			name:      "max_uint64_stays_in_highest_tier",
			total:     18446744073709551615,
			wantIndex: 2,
			wantDown:  10_000_000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx, tier, err := SelectTier(tiers, tc.total)
			if err != nil {
				t.Fatalf("SelectTier(%s) unexpected error: %v", tc.total, err)
			}
			if idx != tc.wantIndex {
				t.Fatalf("SelectTier(%s) index = %d, want %d", tc.total, idx, tc.wantIndex)
			}
			if tier.DownBps != tc.wantDown {
				t.Fatalf("SelectTier(%s) down_bps = %d, want %d", tc.total, tier.DownBps, tc.wantDown)
			}
		})
	}

	if _, _, err := SelectTier(nil, 0); err == nil {
		t.Fatalf("expected error on empty tiers")
	}
}

func Test11_1_StricterOnlyAndAuditedRelaxation(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	periodID := "2026-10"
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	usageR := newFakeUsageReader()
	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
		},
	}
	engine := NewEngine(nil, usageR, polR, nil)

	// Step 1: Member crosses into Tier 1 (100 GB).
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    100_000_000_000,
		AggregateAsOf: now,
	})
	eval1, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Compute tier 1 failed: %v", err)
	}
	if eval1.EnforcedTierIndex != 1 || eval1.Generation != 1 || !eval1.GenerationIncremented {
		t.Fatalf("unexpected eval1: tier=%d gen=%d inc=%v", eval1.EnforcedTierIndex, eval1.Generation, eval1.GenerationIncremented)
	}

	// Step 2: Usage snapshot shows a smaller byte count WITHOUT an audited correction.
	// Within a period, normal evaluation must NOT relax the tier (§11.1).
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    50_000_000_000,
		AggregateAsOf: now.Add(time.Minute),
	})
	eval2, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Compute clamped failed: %v", err)
	}
	if eval2.EnforcedTierIndex != 1 || eval2.Generation != 1 || eval2.GenerationIncremented || eval2.Relaxed {
		t.Fatalf("expected tier to stay clamped at 1 with no new generation, got tier=%d gen=%d inc=%v relaxed=%v",
			eval2.EnforcedTierIndex, eval2.Generation, eval2.GenerationIncremented, eval2.Relaxed)
	}

	// Step 3: Audited negative ledger adjustment (`HasNegativeAdjustment: true`) permits relaxation to Tier 0.
	usageR.set(UsageSnapshot{
		MemberID:              memberID,
		PeriodID:              periodID,
		TotalBytes:            50_000_000_000,
		AggregateAsOf:         now.Add(2 * time.Minute),
		HasNegativeAdjustment: true,
		AdjustmentReason:      "duplicate batch reconciliation adjustment",
	})
	eval3, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Compute with audited relaxation failed: %v", err)
	}
	if eval3.EnforcedTierIndex != 0 || !eval3.Relaxed || eval3.Generation != 2 || !eval3.GenerationIncremented {
		t.Fatalf("expected relaxation to tier 0 with generation 2, got tier=%d relaxed=%v gen=%d inc=%v",
			eval3.EnforcedTierIndex, eval3.Relaxed, eval3.Generation, eval3.GenerationIncremented)
	}
}

func TestAT17_AT19_PeriodRolloverAndLatePreviousPeriodEvent(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	oct := "2026-10"
	nov := "2026-11"
	octTime := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	novTime := time.Date(2026, 11, 1, 0, 5, 0, 0, time.UTC)

	usageR := newFakeUsageReader()
	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
		},
	}
	engine := NewEngine(nil, usageR, polR, nil)

	// In October, member is in Tier 1 (150 GB).
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      oct,
		TotalBytes:    150_000_000_000,
		AggregateAsOf: octTime,
	})
	octEval, err := engine.Compute(ctx, ComputeInput{
		MemberID:        memberID,
		PeriodID:        oct,
		CurrentPeriodID: oct,
		Now:             octTime,
	})
	if err != nil {
		t.Fatalf("october compute failed: %v", err)
	}
	if octEval.EnforcedTierIndex != 1 || octEval.Generation != 1 {
		t.Fatalf("expected oct tier 1 gen 1, got tier=%d gen=%d", octEval.EnforcedTierIndex, octEval.Generation)
	}

	// Period rolls over to November (`2026-11`) with 1 GB usage -> resets to Tier 0 (`normal`).
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      nov,
		TotalBytes:    1_000_000_000,
		AggregateAsOf: novTime,
	})
	novEval, err := engine.Compute(ctx, ComputeInput{
		MemberID:        memberID,
		PeriodID:        nov,
		CurrentPeriodID: nov,
		Now:             novTime,
	})
	if err != nil {
		t.Fatalf("november rollover compute failed: %v", err)
	}
	if novEval.EnforcedTierIndex != 0 || novEval.EnforcedState != EnforcedStateNormal || novEval.Generation != 2 {
		t.Fatalf("expected nov rollover to tier 0 state normal gen 2, got tier=%d state=%s gen=%d",
			novEval.EnforcedTierIndex, novEval.EnforcedState, novEval.Generation)
	}

	// A late usage event arrives for October (`2026-10`) pushing October above 2^53 + 1 (Tier 2).
	// It must only update `2026-10`'s record without issuing a new generation or changing November's tier!
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      oct,
		TotalBytes:    9007199254740993,
		AggregateAsOf: novTime.Add(time.Minute),
	})
	lateOctEval, err := engine.Compute(ctx, ComputeInput{
		MemberID:        memberID,
		PeriodID:        oct,
		CurrentPeriodID: nov,
		Now:             novTime.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("late october compute failed: %v", err)
	}
	if !lateOctEval.LatePeriodEvent || lateOctEval.GenerationIncremented || lateOctEval.WouldApplyTierIndex != 2 {
		t.Fatalf("expected late event to update would_apply_tier_index=2 with no generation increment, got late=%v inc=%v would=%d",
			lateOctEval.LatePeriodEvent, lateOctEval.GenerationIncremented, lateOctEval.WouldApplyTierIndex)
	}

	// Verify November state is still Tier 0, generation 2.
	novStatus, err := engine.GetAdminMemberStatus(ctx, memberID, nov)
	if err != nil {
		t.Fatalf("GetAdminMemberStatus(nov) failed: %v", err)
	}
	if novStatus.MemberStatus.State != api.FairUseStatusStateNormal || novStatus.Generation != 2 {
		t.Fatalf("late oct event corrupted nov state: state=%s gen=%d", novStatus.MemberStatus.State, novStatus.Generation)
	}
}

func TestFUP02_AT19_ShadowVsEnforceAndIdempotentGeneration(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	periodID := "2026-10"
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	usageR := newFakeUsageReader()
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    200_000_000_000, // Tier 1
		AggregateAsOf: now,
	})

	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeShadow),
		},
	}
	ackR := &fakeAckReader{
		status: NodeAckStatus{TotalServingNodes: 2, AckedNodes: 0, PendingNodes: 2},
	}
	engine := NewEngine(nil, usageR, polR, ackR)

	// 1. In shadow mode, WouldApplyTierIndex is 1, but Generation is 0 and member status is "normal".
	shadowEval, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("shadow compute failed: %v", err)
	}
	if shadowEval.WouldApplyTierIndex != 1 || shadowEval.WouldApplyDownBps != 25_000_000 {
		t.Fatalf("shadow would_apply mismatch: tier=%d down=%d", shadowEval.WouldApplyTierIndex, shadowEval.WouldApplyDownBps)
	}
	if shadowEval.Generation != 0 || shadowEval.GenerationIncremented {
		t.Fatalf("shadow mode must not produce a generation: gen=%d inc=%v", shadowEval.Generation, shadowEval.GenerationIncremented)
	}

	memView, err := engine.GetMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatalf("GetMemberStatus shadow failed: %v", err)
	}
	if memView.State != api.FairUseStatusStateNormal || !memView.Rate.IsNull() || memView.Applying {
		t.Fatalf("shadow mode must report normal with no throttling to member: %+v", memView)
	}

	// 2. Switch policy to enforce mode and recompute -> produces generation 1.
	polR.policy.FairUsePolicy = testFairUsePolicy(api.FairUsePayloadModeEnforce)
	enforceEval1, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("enforce compute 1 failed: %v", err)
	}
	if enforceEval1.Generation != 1 || !enforceEval1.GenerationIncremented || enforceEval1.EnforcedTierIndex != 1 {
		t.Fatalf("expected generation 1 on enforce transition, got gen=%d inc=%v tier=%d",
			enforceEval1.Generation, enforceEval1.GenerationIncremented, enforceEval1.EnforcedTierIndex)
	}

	// 3. Recompute again in enforce mode with more usage still inside Tier 1 (e.g., 250 GB) -> NO new generation (AT19).
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    250_000_000_000,
		AggregateAsOf: now.Add(2 * time.Minute),
	})
	enforceEval2, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("enforce compute 2 failed: %v", err)
	}
	if enforceEval2.Generation != 1 || enforceEval2.GenerationIncremented {
		t.Fatalf("expected idempotent generation 1 on unchanged tier, got gen=%d inc=%v",
			enforceEval2.Generation, enforceEval2.GenerationIncremented)
	}
}

func TestAT18_EffectiveRateAndTwoExitScope(t *testing.T) {
	memberID := uuid.New()
	nodeA := uuid.New()
	nodeB := uuid.New()

	// Verify rate_scope = (member_id, node_id, direction) keys are distinct per node and direction (§11.3).
	scopeADown := RateScopeKey(memberID, nodeA, DirectionDown)
	scopeAUp := RateScopeKey(memberID, nodeA, DirectionUp)
	scopeBDown := RateScopeKey(memberID, nodeB, DirectionDown)
	if scopeADown == scopeAUp || scopeADown == scopeBDown {
		t.Fatalf("expected distinct rate_scope keys: A_down=%s A_up=%s B_down=%s", scopeADown, scopeAUp, scopeBDown)
	}

	// Plan rate: 80 Mbps down / 40 Mbps up (TEST ONLY).
	planRate := &api.RatePair{
		DownBps: 80_000_000,
		UpBps:   40_000_000,
	}
	// FUP Tier 1 rate: 25 Mbps down / 10 Mbps up (TEST ONLY).
	reason := api.MemberRatePolicyReasonFairUseTier
	eval := &EvaluationResult{
		MemberID:          memberID,
		PeriodID:          "2026-10",
		Mode:              api.FairUsePayloadModeEnforce,
		Generation:        1,
		EnforcedState:     EnforcedStateAdjusted,
		EnforcedTierIndex: 1,
		EnforcedDownBps:   int64Ptr(25_000_000),
		EnforcedUpBps:     int64Ptr(10_000_000),
		EnforcedPriority:  intPtr(3),
		EnforcedReason:    &reason,
		ValidFrom:         time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		ValidUntil:        time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}

	// Node A has a per-member cap of 50 Mbps down / 20 Mbps up -> min(80M, 25M, 50M) = 25M down, min(40M, 10M, 20M) = 10M up.
	nodeACap := &api.RatePair{
		DownBps: 50_000_000,
		UpBps:   20_000_000,
	}
	polA := BuildNodeMemberRatePolicy(eval, planRate, nodeACap)
	if (!polA.DownBps.IsSpecified() || polA.DownBps.IsNull()) || nullableValue(polA.DownBps) != 25_000_000 || (!polA.UpBps.IsSpecified() || polA.UpBps.IsNull()) || nullableValue(polA.UpBps) != 10_000_000 {
		t.Fatalf("node A effective rate mismatch: down=%v up=%v", polA.DownBps, polA.UpBps)
	}

	// Node B has a tighter per-member cap of 15 Mbps down / 8 Mbps up -> min(80M, 25M, 15M) = 15M down, min(40M, 10M, 8M) = 8M up.
	// Both exits enforce their own rate R independently under their own rate_scope.
	nodeBCap := &api.RatePair{
		DownBps: 15_000_000,
		UpBps:   8_000_000,
	}
	polB := BuildNodeMemberRatePolicy(eval, planRate, nodeBCap)
	if (!polB.DownBps.IsSpecified() || polB.DownBps.IsNull()) || nullableValue(polB.DownBps) != 15_000_000 || (!polB.UpBps.IsSpecified() || polB.UpBps.IsNull()) || nullableValue(polB.UpBps) != 8_000_000 {
		t.Fatalf("node B effective rate mismatch: down=%v up=%v", polB.DownBps, polB.UpBps)
	}
}

func TestFUP04_AdminOverrideAndExpiryRecovery(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	actorID := uuid.New()
	periodID := "2026-10"
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	usageR := newFakeUsageReader()
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    150_000_000_000, // Tier 1 (25 Mbps / 10 Mbps)
		AggregateAsOf: now,
	})
	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
		},
	}
	engine := NewEngine(nil, usageR, polR, nil)

	// Initial evaluation puts member in Tier 1 (gen 1).
	eval1, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("initial compute failed: %v", err)
	}
	if eval1.EnforcedState != EnforcedStateAdjusted || eval1.Generation != 1 {
		t.Fatalf("expected adjusted state gen 1, got state=%s gen=%d", eval1.EnforcedState, eval1.Generation)
	}

	// Admin sets a 2-hour override granting 200 Mbps down / 100 Mbps up.
	ov, eval2, err := engine.SetOverride(ctx, SetOverrideInput{
		MemberID:   memberID,
		DownBps:    200_000_000,
		UpBps:      100_000_000,
		Priority:   0,
		ValidFrom:  now,
		ValidUntil: now.Add(2 * time.Hour),
		Reason:     "temporary diagnostic boost for support ticket #42",
		ActorID:    actorID,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}
	if ov == nil || eval2.EnforcedState != EnforcedStateOverride || *eval2.EnforcedReason != api.MemberRatePolicyReasonAdminOverride {
		t.Fatalf("expected override state and admin_override reason, got state=%s reason=%v", eval2.EnforcedState, eval2.EnforcedReason)
	}
	if *eval2.EnforcedDownBps != 200_000_000 || eval2.Generation != 2 {
		t.Fatalf("expected override down_bps=200M gen=2, got down=%d gen=%d", *eval2.EnforcedDownBps, eval2.Generation)
	}

	// After 2 hours + 1 second, the override expires and the member automatically recovers to Tier 1 (gen 3).
	expiredNow := now.Add(2*time.Hour + time.Second)
	eval3, err := engine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      expiredNow,
	})
	if err != nil {
		t.Fatalf("Compute after override expiry failed: %v", err)
	}
	if eval3.EnforcedState != EnforcedStateAdjusted || *eval3.EnforcedDownBps != 25_000_000 || eval3.Generation != 3 {
		t.Fatalf("expected automatic recovery to tier 1 (25M) gen 3 after override expiry, got state=%s down=%d gen=%d",
			eval3.EnforcedState, *eval3.EnforcedDownBps, eval3.Generation)
	}
}

func TestFUP03_MemberAndAdminStatusNodeAck(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	periodID := "2026-10"
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	usageR := newFakeUsageReader()
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    150_000_000_000,
		AggregateAsOf: now,
	})
	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
		},
	}
	ackR := &fakeAckReader{
		status: NodeAckStatus{
			TotalServingNodes: 3,
			AckedNodes:        1,
			PendingNodes:      2,
		},
	}
	engine := NewEngine(nil, usageR, polR, ackR)

	if _, err := engine.Compute(ctx, ComputeInput{MemberID: memberID, PeriodID: periodID, Now: now}); err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	// While 2 of 3 serving nodes are still pending ACK, Applying must be true and admin summary says "pending on 2 nodes".
	adminStatus, err := engine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatalf("GetAdminMemberStatus pending failed: %v", err)
	}
	if !adminStatus.MemberStatus.Applying {
		t.Fatalf("expected Applying=true while 2 nodes are pending ACK")
	}
	if adminStatus.NodeSyncSummary != "pending on 2 nodes" {
		t.Fatalf("NodeSyncSummary = %q, want %q", adminStatus.NodeSyncSummary, "pending on 2 nodes")
	}

	// Once all 3 serving nodes ACK the generation, Applying becomes false and summary says "applied on 3 nodes".
	ackR.mu.Lock()
	ackR.status = NodeAckStatus{TotalServingNodes: 3, AckedNodes: 3, PendingNodes: 0}
	ackR.mu.Unlock()

	adminStatus2, err := engine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatalf("GetAdminMemberStatus applied failed: %v", err)
	}
	if adminStatus2.MemberStatus.Applying {
		t.Fatalf("expected Applying=false once all 3 serving nodes ACKed")
	}
	if adminStatus2.NodeSyncSummary != "applied on 3 nodes" {
		t.Fatalf("NodeSyncSummary = %q, want %q", adminStatus2.NodeSyncSummary, "applied on 3 nodes")
	}

	for _, tc := range []struct {
		name     string
		status   NodeAckStatus
		applying bool
		summary  string
	}{
		{"no carrying exits", NodeAckStatus{}, false, "no published exit carries this member"},
		{"awaiting all envelopes", NodeAckStatus{TotalServingNodes: 2, PendingNodes: 2, NodesWithoutEnvelope: 2}, true, "pending desired-state compilation"},
		{"awaiting one envelope", NodeAckStatus{TotalServingNodes: 2, PendingNodes: 1, AckedNodes: 1, NodesWithoutEnvelope: 1}, true, "pending on 1 node"},
		{"frozen and unsafe exits", NodeAckStatus{TotalServingNodes: 3, PendingNodes: 2, AckedNodes: 1, FrozenNodes: 1, UnsafeNodes: 1}, true, "pending on 2 nodes (1 frozen, 1 unsafe)"},
		{"frozen before compilation", NodeAckStatus{TotalServingNodes: 1, PendingNodes: 1, NodesWithoutEnvelope: 1, FrozenNodes: 1}, true, "pending desired-state compilation (1 frozen)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ackR.mu.Lock()
			ackR.status = tc.status
			ackR.mu.Unlock()
			adminStatus, err := engine.GetAdminMemberStatus(ctx, memberID, periodID)
			if err != nil {
				t.Fatal(err)
			}
			if adminStatus.MemberStatus.Applying != tc.applying || adminStatus.NodeSyncSummary != tc.summary {
				t.Fatalf("applying=%v summary=%q, want %v %q", adminStatus.MemberStatus.Applying, adminStatus.NodeSyncSummary, tc.applying, tc.summary)
			}
			memberStatus, err := engine.GetMemberStatus(ctx, memberID, periodID)
			if err != nil {
				t.Fatal(err)
			}
			if memberStatus.Applying != tc.applying {
				t.Fatalf("member applying=%v, want %v", memberStatus.Applying, tc.applying)
			}
		})
	}
}

func Test11_1_CoalescedWorkerAndWatermark(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	periodID := "2026-10"
	watermark := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	now := time.Date(2026, 10, 15, 14, 31, 0, 0, time.UTC)

	usageR := newFakeUsageReader()
	usageR.set(UsageSnapshot{
		MemberID:      memberID,
		PeriodID:      periodID,
		TotalBytes:    120_000_000_000,
		AggregateAsOf: watermark,
	})
	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
		},
	}
	engine := NewEngine(nil, usageR, polR, nil)
	worker := NewWorker(engine)

	// Enqueue 5 ledger notifications for the same (memberID, periodID).
	for i := 0; i < 5; i++ {
		worker.NotifyLedgerChange(memberID, periodID)
	}
	if got := worker.PendingCount(); got != 1 {
		t.Fatalf("PendingCount() = %d, want 1 coalesced job", got)
	}

	results, err := worker.Flush(ctx, now)
	if err != nil {
		t.Fatalf("worker.Flush failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if usageR.calls != 1 {
		t.Fatalf("usageReader called %d times, want 1 coalesced call", usageR.calls)
	}

	wm, ok := worker.WatermarkFor(memberID, periodID)
	if !ok {
		t.Fatalf("expected watermark record for member")
	}
	if !wm.LastComputedAt.Equal(now) || !wm.LedgerWatermark.Equal(watermark) || wm.EvaluationCount != 1 {
		t.Fatalf("unexpected watermark record: %+v", wm)
	}
}

type errUsageReader struct {
	rawTotal string
}

func (r *errUsageReader) GetMemberPeriodUsage(_ context.Context, _ uuid.UUID, _ string) (UsageSnapshot, error) {
	u, err := parseFUPUint64("total_bytes", r.rawTotal)
	if err != nil {
		return UsageSnapshot{}, err
	}
	return UsageSnapshot{TotalBytes: u}, nil
}

func TestFUP01_Uint64OverflowFailsClosed(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	periodID := "2026-10"
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	polR := &fakePolicyReader{
		policy: MemberPolicyContext{
			FairUseRevisionID: uuid.New(),
			FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
		},
	}

	// Exact math.MaxUint64 succeeds.
	maxEngine := NewEngine(nil, &errUsageReader{rawTotal: "18446744073709551615"}, polR, nil)
	evalMax, err := maxEngine.Compute(ctx, ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Compute at math.MaxUint64 failed: %v", err)
	}
	if evalMax.TotalBytes != ^decstr.U64(0) {
		t.Fatalf("TotalBytes = %s, want 18446744073709551615", evalMax.TotalBytes)
	}

	// math.MaxUint64 + 1 fails closed with decstr.ErrOverflow and ErrCodeUint64Overflow, never clamping.
	for _, rawOverflow := range []string{"18446744073709551616", "99999999999999999999", "-1"} {
		ovEngine := NewEngine(nil, &errUsageReader{rawTotal: rawOverflow}, polR, nil)
		_, err := ovEngine.Compute(ctx, ComputeInput{
			MemberID: memberID,
			PeriodID: periodID,
			Now:      now,
		})
		if !errors.Is(err, decstr.ErrOverflow) {
			t.Fatalf("rawTotal=%q: expected errors.Is(err, decstr.ErrOverflow), got: %v", rawOverflow, err)
		}
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.Code != apierr.InternalError || ae.Details["detail_code"] != usage.ErrCodeUint64Overflow {
			t.Fatalf("rawTotal=%q: expected InternalError with detail_code=%s, got: %#v", rawOverflow, usage.ErrCodeUint64Overflow, err)
		}
	}
}
