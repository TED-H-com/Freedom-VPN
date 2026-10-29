package desiredstate

import (
	"testing"
	"time"
)

func TestComputeEffectiveRateBps(t *testing.T) {
	cap40M := int64(40_000_000)
	cap200M := int64(200_000_000)

	cases := []struct {
		name     string
		planRate int64
		fupRate  int64
		nodeCap  *int64
		want     int64
	}{
		{
			name:     "node cap is lowest",
			planRate: 100_000_000,
			fupRate:  50_000_000,
			nodeCap:  &cap40M,
			want:     40_000_000,
		},
		{
			name:     "fup tier rate is lowest",
			planRate: 100_000_000,
			fupRate:  25_000_000,
			nodeCap:  &cap200M,
			want:     25_000_000,
		},
		{
			name:     "plan rate is lowest with nil node cap",
			planRate: 30_000_000,
			fupRate:  80_000_000,
			nodeCap:  nil,
			want:     30_000_000,
		},
		{
			name:     "zero when plan rate missing",
			planRate: 0,
			fupRate:  0,
			nodeCap:  nil,
			want:     0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeEffectiveRateBps(tc.planRate, tc.fupRate, tc.nodeCap)
			if got != tc.want {
				t.Errorf("ComputeEffectiveRateBps(%d, %d, %v) = %d, want %d",
					tc.planRate, tc.fupRate, tc.nodeCap, got, tc.want)
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
