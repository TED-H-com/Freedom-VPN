//go:build e2e

package secops

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// usagePeriodForStart keeps synthetic usage in the UTC month of its interval
// start; the exclusive end may already belong to the next month (§10.4, §10.9).
func usagePeriodForStart(start time.Time) api.PeriodId {
	return api.PeriodId(start.UTC().Format("2006-01"))
}

func TestMTR03_UsagePeriodOverlapsInterval(t *testing.T) {
	for _, tc := range []struct {
		name string
		now  string
	}{
		{"month_start", "2026-11-01T00:00:00Z"},
		{"first_ten_seconds", "2026-11-01T00:00:05Z"},
		{"mid_month", "2026-11-15T12:00:00Z"},
		{"year_start", "2027-01-01T00:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			for _, window := range []struct {
				name   string
				events int
			}{{"pitr", 10}, {"rate_class", 1}} {
				t.Run(window.name, func(t *testing.T) {
					for n := 0; n < window.events; n++ {
						start := now.Add(time.Duration(n-window.events) * time.Second)
						end := start.Add(time.Second)
						id := usagePeriodForStart(start)
						period, err := usage.ParseMonth(string(id))
						if err != nil {
							t.Fatal(err)
						}
						// Same strict overlap rule as internal/usage/ingest.go:
						// touching only the exclusive end is insufficient.
						if !period.Start.Before(end) || !period.End.After(start) {
							t.Errorf("event %d: period %s does not overlap [%s, %s)", n+1, id, start.Format(time.RFC3339), end.Format(time.RFC3339))
						}
					}
				})
			}
		})
	}
}
