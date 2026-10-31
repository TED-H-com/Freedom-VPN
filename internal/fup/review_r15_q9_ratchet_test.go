package fup

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestReviewR15_TierOscillationEmitsOneUpwardGeneration is one member's own
// traffic around a tier threshold. SelectTier has no hysteresis: equality
// enters the tier. Within a period, dropping back below the threshold does
// not lower the tier and does not increment the generation. One member can
// therefore emit at most one fup.member_generation_changed event per upward
// tier crossing per period (two crossings on the three-tier test policy),
// not one event per sample.
func TestReviewR15_TierOscillationEmitsOneUpwardGeneration(t *testing.T) {
	ctx := context.Background()
	memberID := uuid.New()
	periodID := "2026-10"
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	usageR := newFakeUsageReader()
	polR := &fakePolicyReader{policy: MemberPolicyContext{
		FairUseRevisionID: uuid.New(),
		FairUsePolicy:     testFairUsePolicy(api.FairUsePayloadModeEnforce),
	}}
	engine := NewEngine(nil, usageR, polR, nil)

	// The test policy's first threshold is 100_000_000_000. Oscillate across
	// it after the upward cross. The second threshold is far above any of
	// these totals, so a second upward event does not happen here.
	const threshold decstr.U64 = 100_000_000_000
	steps := []struct {
		bytes decstr.U64
		want  bool
	}{
		{threshold - 1, true},  // first evaluation, generation 1 at tier 0
		{threshold, true},      // upward cross into tier 1
		{threshold - 1, false}, // back under the threshold: clamped, no event
		{threshold, false},     // equal again: already at tier 1
		{threshold + 50, false},
		{threshold - 50, false},
		{1, false},
	}
	increments := 0
	for i, st := range steps {
		usageR.set(UsageSnapshot{
			MemberID: memberID, PeriodID: periodID,
			TotalBytes: st.bytes, AggregateAsOf: now.Add(time.Duration(i) * time.Minute),
		})
		eval, err := engine.Compute(ctx, ComputeInput{MemberID: memberID, PeriodID: periodID, Now: now.Add(time.Duration(i) * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		if eval.GenerationIncremented {
			increments++
		}
		if eval.GenerationIncremented != st.want {
			t.Fatalf("step %d bytes=%s incremented=%v, want %v (tier %d gen %s)", i, st.bytes, eval.GenerationIncremented, st.want, eval.EnforcedTierIndex, eval.Generation)
		}
	}
	if increments != 2 {
		t.Fatalf("upward events = %d, want 2 (baseline generation plus one tier crossing) across %d evaluations", increments, len(steps))
	}
}
