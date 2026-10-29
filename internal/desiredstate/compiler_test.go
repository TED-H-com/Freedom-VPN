package desiredstate

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func TestNodeCapacityRatePairAndFUPBuilder(t *testing.T) {
	cap40M := int64(40_000_000)
	cap200M := int64(200_000_000)
	fup50M := int64(50_000_000)
	fup25M := int64(25_000_000)
	fup80M := int64(80_000_000)
	reasonTier := api.MemberRatePolicyReasonFairUseTier
	prio2 := 2

	cases := []struct {
		name     string
		planDown int64
		fupDown  *int64
		nodeCap  *api.NodeCapacity
		wantDown int64
	}{
		{
			name:     "node member cap is lowest",
			planDown: 100_000_000,
			fupDown:  &fup50M,
			nodeCap: &api.NodeCapacity{
				DownBps:          1_000_000_000,
				UpBps:            1_000_000_000,
				MemberDownBpsCap: presence.NewNullableWithValue(cap40M),
				MemberUpBpsCap:   presence.NewNullableWithValue(cap40M),
			},
			wantDown: 40_000_000,
		},
		{
			name:     "fup tier rate is lowest",
			planDown: 100_000_000,
			fupDown:  &fup25M,
			nodeCap: &api.NodeCapacity{
				DownBps:          1_000_000_000,
				UpBps:            1_000_000_000,
				MemberDownBpsCap: presence.NewNullableWithValue(cap200M),
				MemberUpBpsCap:   presence.NewNullableWithValue(cap200M),
			},
			wantDown: 25_000_000,
		},
		{
			name:     "plan rate is lowest with nil node cap",
			planDown: 30_000_000,
			fupDown:  &fup80M,
			nodeCap:  nil,
			wantDown: 30_000_000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := &fup.EvaluationResult{
				MemberID:          uuid.New(),
				PeriodID:          "2026-10",
				Mode:              api.FairUsePayloadModeEnforce,
				Generation:        1,
				EnforcedTierIndex: 1,
				EnforcedDownBps:   tc.fupDown,
				EnforcedUpBps:     tc.fupDown,
				EnforcedPriority:  &prio2,
				EnforcedReason:    &reasonTier,
				ValidFrom:         time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				ValidUntil:        time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
			}
			pol := fup.BuildNodeMemberRatePolicy(eval, &api.RatePair{DownBps: tc.planDown, UpBps: tc.planDown}, nodeCapacityRatePair(tc.nodeCap))
			got, err := pol.DownBps.Get()
			if err != nil || got != tc.wantDown {
				t.Errorf("BuildNodeMemberRatePolicy DownBps = (%d, %v), want %d", got, err, tc.wantDown)
			}
		})
	}
}

func TestDefaultAccountingPeriod(t *testing.T) {
	ts := time.Date(2026, 10, 29, 14, 30, 0, 0, time.UTC)
	ap := DefaultAccountingPeriod(ts)
	if ap.PeriodId != "2026-10" {
		t.Errorf("PeriodId = %q, want 2026-10", ap.PeriodId)
	}
	if !ap.PeriodEnd.After(ap.PeriodStart) {
		t.Errorf("expected PeriodEnd > PeriodStart, got [%v, %v)", ap.PeriodStart, ap.PeriodEnd)
	}
}
