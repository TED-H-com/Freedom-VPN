package nodes

import (
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

func TestSECOPS11_RecordedBaselineDoesNotProveServing(t *testing.T) {
	// testdata/secops11_recorded_baseline.json is the SECOPS-11 recorded baseline
	// captured by the security-ops harness before #109 and is frozen here on purpose:
	// the harness's docs/e2e/at/secops/live.json is rewritten by every acceptance run
	// and must not be read by unit tests.
	raw, err := os.ReadFile("testdata/secops11_recorded_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		PublicationOperation struct {
			Status string `json:"status"`
		} `json:"publication_operation"`
		WatchdogState struct {
			Serving      string `json:"serving"`
			Reason       string `json:"reason"`
			PolicyExpiry string `json:"policy_expires_mono_ns"`
		} `json:"watchdog_state"`
		NodeAfterPublication struct {
			DataHealth       api.DataHealth       `json:"data_health"`
			ConfigGeneration api.ConfigGeneration `json:"config_generation"`
		} `json:"node_after_publication"`
		FlowProbe struct {
			CountedFlow bool `json:"counted_flow"`
			NodeServing bool `json:"node_serving"`
		} `json:"flow_probe"`
	}
	if err = json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.PublicationOperation.Status != "succeeded" || record.WatchdogState.Serving != "0" || record.WatchdogState.Reason != "recovery_required" || record.WatchdogState.PolicyExpiry != "0" || record.FlowProbe.CountedFlow || record.FlowProbe.NodeServing {
		t.Fatal("SECOPS-11 fixture shape changed")
	}
	now := time.Now().UTC()
	report := api.ObservationReport{ProvisioningState: api.ProvisioningStateReady, DataHealth: record.NodeAfterPublication.DataHealth, SafetyStatus: api.SafetyStatusPassed, MemberForwardingEnabled: record.FlowProbe.NodeServing}
	report.AppliedConfigGeneration = record.NodeAfterPublication.ConfigGeneration.Applied
	for _, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		report.Checks = append(report.Checks, api.NodeCheck{Name: name, Status: api.HealthCheckStatusPassed})
	}
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("baseline claimed member serving")
	}
	report.MemberForwardingEnabled = true
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("missing lease claimed member serving")
	}
	report.Lease.LeaseId.Set(uuid.New())
	report.Lease.ExpiresAt.Set(now.Add(time.Minute))
	if !PublicationForwardingEvidence(report, now) {
		t.Fatal("armed policy rejected")
	}
	report.AppliedConfigGeneration.Set(0)
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("generation zero claimed serving")
	}
	report.AppliedConfigGeneration.Set(1)
	report.Lease.ExpiresAt.Set(now)
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("expired lease claimed serving")
	}
}

func TestAT08_PublicationBindingAwareEvidence(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name  string
		empty bool
		edit  func(*api.ObservationReport)
		want  string
	}{
		{"empty_not_run", true, nil, ""},
		{"bound_not_run", false, nil, "serving_checks_unconfirmed"},
		{"empty_failed", true, func(r *api.ObservationReport) { r.Checks[3].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"bound_failed", false, func(r *api.ObservationReport) { r.Checks[3].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"failed_then_passed_duplicate", true, func(r *api.ObservationReport) {
			c := r.Checks[0]
			c.Status = api.HealthCheckStatusFailed
			r.Checks = append([]api.NodeCheck{c}, r.Checks...)
		}, "check_failed"},
		{"forwarding_off", true, func(r *api.ObservationReport) { r.MemberForwardingEnabled = false }, "forwarding_disabled"},
		{"tunnel_not_run", true, func(r *api.ObservationReport) { r.Checks[2].Status = api.HealthCheckStatusNotRun }, "tunnel_not_passed"},
		{"isolation_absent", true, func(r *api.ObservationReport) { r.Checks = r.Checks[1:] }, "isolation_unconfirmed"},
		{"serving_check_absent", true, func(r *api.ObservationReport) { r.Checks = r.Checks[:3] }, "serving_checks_unconfirmed"},
		{"degraded", true, func(r *api.ObservationReport) { r.DataHealth = api.DataHealthDegraded }, "readiness_unconfirmed"},
		{"isolation_apply_error_pending", true, func(r *api.ObservationReport) {
			r.Errors = []api.NodeError{{
				Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
				Stage:      "apply",
				Message:    "conntrack delete pending; serving ruleset verified",
				ObservedAt: now,
			}}
		}, "isolation_unconfirmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := api.ObservationReport{ProvisioningState: api.ProvisioningStateReady, SafetyStatus: api.SafetyStatusPassed, DataHealth: api.DataHealthHealthy, MemberForwardingEnabled: true}
			r.AppliedConfigGeneration.Set(2)
			for i, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
				status := api.HealthCheckStatusNotRun
				if i < 3 {
					status = api.HealthCheckStatusPassed
				}
				r.Checks = append(r.Checks, api.NodeCheck{Name: name, Status: status})
			}
			if tc.edit != nil {
				tc.edit(&r)
			}
			if cause := publicationForwardingFailure(r, now, tc.empty); cause != tc.want || len(cause) > 64 {
				t.Fatal("unexpected publication gate", cause)
			}
		})
	}
}

func TestEvaluateObservationEvidence_IsolationApplyError(t *testing.T) {
	now := time.Now().UTC()
	report := api.ObservationReport{
		ProvisioningState:       api.ProvisioningStateReady,
		SafetyStatus:            api.SafetyStatusPassed,
		DataHealth:              api.DataHealthHealthy,
		MemberForwardingEnabled: true,
		ObservedAt:              now,
		Checks: []api.NodeCheck{
			{Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusPassed, ObservedAt: now},
			{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: now},
			{Name: api.NodeCheckNameTunnelProbe, Status: api.HealthCheckStatusPassed, ObservedAt: now},
		},
		Errors: []api.NodeError{
			{
				Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
				Stage:      "apply",
				Message:    "conntrack delete pending; serving ruleset verified",
				ObservedAt: now,
			},
		},
	}
	report.Netbird.Backend = api.ObservationReportNetbirdBackendKernel

	safety, health, forwarding, _ := evaluateObservationEvidence(report)
	if safety != api.SafetyStatusPassed {
		t.Fatalf("safety = %q, want passed while conntrack delete retry is pending", safety)
	}
	if health != api.DataHealthDegraded {
		t.Fatalf("health = %q, want degraded while ISOLATION_POLICY_INVALID apply error is present", health)
	}
	if !forwarding {
		t.Fatalf("forwarding = %v, want true", forwarding)
	}
	if _, ok := isPendingIsolationApplyError(report, forwarding); !ok {
		t.Fatal("expected isPendingIsolationApplyError to be true")
	}

	// When the node fails closed after exhausting retries, forwarding is disabled and safety is failed.
	failClosed := report
	failClosed.MemberForwardingEnabled = false
	failClosed.SafetyStatus = api.SafetyStatusFailed
	failClosed.ProvisioningState = api.ProvisioningStateApplying
	failClosed.Errors = []api.NodeError{
		{
			Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
			Stage:      "apply",
			Message:    "conntrack delete failed after the serving ruleset was verified",
			ObservedAt: now,
		},
	}
	safetyFC, healthFC, forwardingFC, _ := evaluateObservationEvidence(failClosed)
	if safetyFC != api.SafetyStatusFailed {
		t.Fatalf("fail-closed safety = %q, want failed", safetyFC)
	}
	if healthFC != api.DataHealthDegraded {
		t.Fatalf("fail-closed health = %q, want degraded", healthFC)
	}
	if forwardingFC {
		t.Fatalf("fail-closed forwarding = %v, want false", forwardingFC)
	}
	if _, ok := isPendingIsolationApplyError(failClosed, forwardingFC); ok {
		t.Fatal("expected isPendingIsolationApplyError to be false once node failed closed")
	}
}

func TestD067_AwaitingMemberGrantsObservationAndPublicationEvidence(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.awaiting-member-grants.json")
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	var report api.ObservationReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal example: %v", err)
	}
	if report.MemberForwardingEnabled {
		t.Fatal("expected member_forwarding_enabled=false on awaiting_member_grants example")
	}
	if len(report.ForwardingBlockReasons) != 1 || report.ForwardingBlockReasons[0] != api.ForwardingBlockReasonAwaitingMemberGrants {
		t.Fatalf("forwarding_block_reasons = %v, want [awaiting_member_grants]", report.ForwardingBlockReasons)
	}
	if !report.ForwardingBlockReasons[0].Valid() {
		t.Fatal("expected awaiting_member_grants to be a valid ForwardingBlockReason")
	}

	safety, health, forwarding, errs := evaluateObservationEvidence(report)
	if safety != api.SafetyStatusPassed {
		t.Fatalf("safety = %q, want passed (provisional node lease, not expired)", safety)
	}
	if health != api.DataHealthHealthy {
		t.Fatalf("health = %q, want healthy", health)
	}
	if forwarding {
		t.Fatalf("forwarding = %v, want false", forwarding)
	}
	if len(errs) != 0 {
		t.Fatalf("errors = %v, want empty", errs)
	}

	now := report.ObservedAt.UTC()
	if cause := publicationForwardingFailure(report, now, false); cause != "forwarding_disabled" {
		t.Fatalf("publicationForwardingFailure = %q, want forwarding_disabled", cause)
	}
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("PublicationForwardingEvidence must be false while awaiting_member_grants has member_forwarding_enabled=false")
	}
}
