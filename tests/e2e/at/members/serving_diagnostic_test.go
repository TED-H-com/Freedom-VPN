package members

import (
	"slices"
	"testing"
	"time"
)

func TestD146_LaterServingReadClassifiesEveryCondition(t *testing.T) {
	str := func(s string) *string { return &s }
	yes, no := true, false
	empty, blocked := []string{}, []string{"awaiting_member_grants"}
	now := time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)
	good := servingDiagnosticRows{ReadAt: now, NodeFound: true, CurrentIncarnation: str("current"), Desired: str("76"), Applied: str("76"), LatestCurrent: &servingDiagnosticObservation{Incarnation: "current", Generation: str("76"), ReceivedAt: now.Add(-time.Second), Forwarding: &yes, Reasons: &empty}}
	cases := []struct {
		name   string
		change func(*servingDiagnosticRows)
		want   []string
	}{
		{"later_all_pass", func(*servingDiagnosticRows) {}, nil},
		{"desired", func(r *servingDiagnosticRows) { r.Desired = str("77") }, []string{"desired_equals_applied_generation"}},
		{"incarnation", func(r *servingDiagnosticRows) { r.LatestCurrent.Incarnation = "old" }, []string{"observation_incarnation_matches_current"}},
		{"generation", func(r *servingDiagnosticRows) { r.LatestCurrent.Generation = str("75") }, []string{"observation_generation_equals_applied", "observation_generation_equals_kernel"}},
		{"age_exact_bound", func(r *servingDiagnosticRows) { r.LatestCurrent.ReceivedAt = now.Add(-15 * time.Second) }, []string{"observation_age_below_15_seconds"}},
		{"age_just_inside", func(r *servingDiagnosticRows) {
			r.LatestCurrent.ReceivedAt = now.Add(-15*time.Second + time.Nanosecond)
		}, nil},
		{"forwarding", func(r *servingDiagnosticRows) { r.LatestCurrent.Forwarding = &no }, []string{"member_forwarding_enabled"}},
		{"reasons", func(r *servingDiagnosticRows) { r.LatestCurrent.Reasons = &blocked }, []string{"forwarding_block_reasons_empty"}},
		{"null_payload", func(r *servingDiagnosticRows) { r.LatestCurrent.Forwarding = nil; r.LatestCurrent.Reasons = nil }, []string{"member_forwarding_enabled", "forwarding_block_reasons_empty"}},
		{"missing_current", func(r *servingDiagnosticRows) { r.LatestAny = r.LatestCurrent; r.LatestCurrent = nil }, []string{"current_incarnation_observation_present"}},
		{"missing_node", func(r *servingDiagnosticRows) { *r = servingDiagnosticRows{ReadAt: now} }, []string{"node_present", "desired_equals_applied_generation", "current_incarnation_observation_present"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := good
			observation := *good.LatestCurrent
			rows.LatestCurrent = &observation
			tc.change(&rows)
			got := classifyServingRead(rows, "76")
			if !got.LaterRead || !slices.Equal(got.Failed, tc.want) {
				t.Fatalf("later classification: %+v", got)
			}
		})
	}
}
