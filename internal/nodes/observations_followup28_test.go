package nodes

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// D-093 / ISO02: both report locations retain the node-local error code. The
// failed safety check overrides even a contradictory safety_status=passed.
func TestD093_ISO02_ObservationLocalFailureEvidence(t *testing.T) {
	for _, code := range []api.ErrorCode{api.ErrorCodeHEALTHCHECKFAILED, api.ErrorCodeTCFOREIGNFILTER} {
		for _, location := range []string{"errors", "checks", "both"} {
			t.Run(string(code)+"/"+location, func(t *testing.T) {
				now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
				report := api.ObservationReport{
					ObservedAt: now, SafetyStatus: api.SafetyStatusFailed,
					DataHealth: api.DataHealthDegraded, MemberForwardingEnabled: false,
					ForwardingBlockReasons: []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed},
					Checks: []api.NodeCheck{
						{Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusFailed, ObservedAt: now},
						{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: now},
					},
				}
				if location != "checks" {
					report.Errors = []api.NodeError{{Code: code, Stage: "apply", Message: "TEST ONLY local verification failure", ObservedAt: now}}
				}
				if location != "errors" {
					report.Checks[0].ErrorCode.Set(code)
					report.SafetyStatus = api.SafetyStatusPassed
				}
				safety, health, forwarding, errs := evaluateObservationEvidence(report)
				if safety != api.SafetyStatusFailed || health != api.DataHealthDegraded || forwarding {
					t.Fatalf("failure classified as %s/%s forwarding=%v", safety, health, forwarding)
				}
				if location != "checks" && (len(errs) == 0 || errs[0] != report.Errors[0]) {
					t.Fatal("normalized error lost its original code or evidence")
				}
			})
		}
	}
}
