package nodes

import (
	"fmt"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func holdSummaryText(applied, lease, steps, elapsed uint64, alarm bool) string {
	return fmt.Sprintf("Serving lease generation hold: applied=%d lease=%d steps=%d elapsed_ms=%d alarm=%t.", applied, lease, steps, elapsed, alarm)
}

func holdReport() api.ObservationReport {
	r := api.ObservationReport{NodeId: uuid.New(), NodeIncarnationId: uuid.New(), ReportId: uuid.New(), BootId: uuid.New(),
		ObservedAt: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC), ProvisioningState: api.ProvisioningStateReady,
		SafetyStatus: api.SafetyStatusPassed, DataHealth: api.DataHealthHealthy,
		ForwardingBlockReasons: []api.ForwardingBlockReason{api.ForwardingBlockReasonPolicyMissing}}
	r.AppliedConfigGeneration.Set(2)
	r.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	r.Lease.LeaseId.Set(uuid.New())
	r.Lease.ExpiresAt.Set(r.ObservedAt.Add(time.Hour))
	for _, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe} {
		r.Checks = append(r.Checks, api.NodeCheck{Name: name, Status: api.HealthCheckStatusPassed, ObservedAt: r.ObservedAt,
			ErrorCode: presence.NewNullNullable[api.ErrorCode]()})
	}
	for _, v := range []struct {
		name   api.NodeCheckName
		code   api.ErrorCode
		detail string
	}{{api.NodeCheckNameLease, api.ErrorCodeSTALEGENERATION, "Serving lease is not for the applied generation."},
		{api.NodeCheckNameQos, api.ErrorCodeVALIDATIONFAILED, "Rate policy could not be applied."},
		{api.NodeCheckNameSourceBinding, api.ErrorCodeVALIDATIONFAILED, "No member binding was admitted."}} {
		detail := v.detail
		r.Checks = append(r.Checks, api.NodeCheck{Name: v.name, Status: api.HealthCheckStatusFailed,
			ErrorCode: presence.NewNullableWithValue(v.code), Detail: &detail, ObservedAt: r.ObservedAt})
	}
	r.Errors = []api.NodeError{{Stage: "lease", Code: api.ErrorCodeSTALEGENERATION, ObservedAt: r.ObservedAt,
		Message: holdSummaryText(2, 1, 1, 0, false)}}
	return r
}

func TestAT29_M3_HoldClassificationOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*api.ObservationReport)
		want string
	}{
		{"alone", func(*api.ObservationReport) {}, "stale_generation"},
		{"lease_check_only", func(r *api.ObservationReport) { r.Errors = nil }, "stale_generation"},
		{"lease_error_only", func(r *api.ObservationReport) { r.Checks = r.Checks[:3] }, "stale_generation"},
		{"no_policy_missing", func(r *api.ObservationReport) { r.ForwardingBlockReasons = nil }, "check_failed"},
		{"internet_first", func(r *api.ObservationReport) { r.Checks[0].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"meter_last", func(r *api.ObservationReport) {
			c := r.Checks[1]
			c.Status = api.HealthCheckStatusFailed
			r.Checks = append(r.Checks, c)
		}, "check_failed"},
		{"retained_qos", func(r *api.ObservationReport) { r.Checks[4].ErrorCode.Set(api.ErrorCodeTCFOREIGNFILTER) }, "check_failed"},
		{"retained_lease", func(r *api.ObservationReport) { r.Checks[3].ErrorCode.Set(api.ErrorCodeDEPENDENCYUNAVAILABLE) }, "check_failed"},
		{"error_safety", func(r *api.ObservationReport) {
			r.Errors = append(r.Errors, api.NodeError{Code: api.ErrorCodeMETERPATHINVALID, Stage: "meter"})
		}, "check_failed"},
		{"safety_failed", func(r *api.ObservationReport) { r.SafetyStatus = api.SafetyStatusFailed }, "isolation_unconfirmed"},
		{"malformed_summary", func(r *api.ObservationReport) { r.Errors[0].Message = "bad summary" }, "stale_generation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := holdReport()
			tc.edit(&r)
			if got := publicationForwardingFailure(r, r.ObservedAt, false); got != tc.want {
				t.Fatalf("cause=%s want %s", got, tc.want)
			}
		})
	}
	r := holdReport()
	r.MemberForwardingEnabled = true
	_, _, forwarding, _ := evaluateObservationEvidence(r)
	if forwarding {
		t.Fatal("hold ACK inferred forwarding")
	}
}

func TestAT29_M3_SummaryStrictAndLossless(t *testing.T) {
	for _, text := range []string{
		"", "Serving lease is not for the desired generation.",
		holdSummaryText(2, 1, 1, 0, false) + " extra",
		"Serving lease generation hold: applied=02 lease=1 steps=1 elapsed_ms=0 alarm=false.",
		"Serving lease generation hold: applied=2 lease=1 steps=-1 elapsed_ms=0 alarm=false.",
		"Serving lease generation hold: applied=2 lease=1 steps=18446744073709551616 elapsed_ms=0 alarm=false.",
		"Serving lease generation hold: applied=2 lease=1 steps=1 elapsed_ms=0 alarm=1.",
	} {
		r := holdReport()
		r.Errors[0].Message = text
		if _, ok := parseGenerationHoldSummary(r); ok {
			t.Fatalf("accepted malformed summary %q", text)
		}
	}
	r := holdReport()
	r.Errors[0].Message = holdSummaryText(2, 1, 18446744073709551615, 9007199254740993, false)
	s, ok := parseGenerationHoldSummary(r)
	if !ok || s.Steps.String() != "18446744073709551615" || s.ElapsedMS.String() != "9007199254740993" || s.Alarm {
		t.Fatalf("lossy parse: %+v %v", s, ok)
	}
	r.Errors = append(r.Errors, r.Errors[0])
	if _, ok := parseGenerationHoldSummary(r); ok {
		t.Fatal("duplicate summaries accepted")
	}
}
