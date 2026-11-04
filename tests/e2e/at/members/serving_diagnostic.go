package members

import (
	"strconv"
	"time"
)

// This is a later read, never an alternative serving predicate. Nil fields
// preserve absent SQL values instead of silently turning them into agreement.
type servingDiagnosticRows struct {
	ReadAt             time.Time                     `json:"read_at"`
	NodeFound          bool                          `json:"node_found"`
	CurrentIncarnation *string                       `json:"current_incarnation"`
	Desired            *string                       `json:"desired_generation"`
	Applied            *string                       `json:"applied_generation"`
	LatestCurrent      *servingDiagnosticObservation `json:"latest_current_incarnation_observation"`
	LatestAny          *servingDiagnosticObservation `json:"latest_node_observation"`
}
type servingDiagnosticObservation struct {
	ReportID    string    `json:"report_id"`
	Incarnation string    `json:"incarnation"`
	Generation  *string   `json:"generation"`
	ReceivedAt  time.Time `json:"received_at"`
	Forwarding  *bool     `json:"member_forwarding_enabled"`
	Reasons     *[]string `json:"forwarding_block_reasons"`
}
type servingDiagnostic struct {
	LaterRead        bool                  `json:"later_read"`
	Scope            string                `json:"scope"`
	KernelGeneration string                `json:"kernel_generation"`
	Rows             servingDiagnosticRows `json:"rows"`
	ObservationAgeNS *string               `json:"observation_age_ns"`
	AgeBoundNS       string                `json:"age_bound_ns"`
	Failed           []string              `json:"failed_conditions"`
}

func classifyServingRead(rows servingDiagnosticRows, kernel string) servingDiagnostic {
	out := servingDiagnostic{LaterRead: true, Scope: "Second read after the combined serving query returned no row; conditions describe this later read, not a proven retrospective cause.", KernelGeneration: kernel, Rows: rows, AgeBoundNS: "15000000000", Failed: []string{}}
	fail := func(ok bool, name string) {
		if !ok {
			out.Failed = append(out.Failed, name)
		}
	}
	equal := func(a, b *string) bool { return a != nil && b != nil && *a == *b }
	fail(rows.NodeFound, "node_present")
	fail(equal(rows.Desired, rows.Applied), "desired_equals_applied_generation")
	current := rows.LatestCurrent
	if current == nil {
		out.Failed = append(out.Failed, "current_incarnation_observation_present")
		return out
	}
	fail(rows.CurrentIncarnation != nil && current.Incarnation == *rows.CurrentIncarnation, "observation_incarnation_matches_current")
	fail(equal(current.Generation, rows.Applied), "observation_generation_equals_applied")
	fail(equal(current.Generation, &kernel), "observation_generation_equals_kernel")
	age := rows.ReadAt.Sub(current.ReceivedAt)
	v := strconv.FormatInt(age.Nanoseconds(), 10)
	out.ObservationAgeNS = &v
	// SQL is strictly received_at > read_at - 15 seconds; no added future bound.
	fail(!current.ReceivedAt.IsZero() && age < 15*time.Second, "observation_age_below_15_seconds")
	fail(current.Forwarding != nil && *current.Forwarding, "member_forwarding_enabled")
	fail(current.Reasons != nil && len(*current.Reasons) == 0, "forwarding_block_reasons_empty")
	return out
}
