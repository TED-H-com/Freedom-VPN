//go:build integration

package httpwire_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestReviewR15_SafetyFailedPublishWaitsAsReadinessUnconfirmed retains the
// review name and proves first-attempt failure evidence for current-generation
// failed/expired safety, while stale and unconfirmed states keep waiting.
func TestReviewR15_SafetyFailedPublishWaitsAsReadinessUnconfirmed(t *testing.T) {
	if config.Defaults().WorkerAttempts != 10 {
		t.Fatalf("WorkerAttempts = %d, want 10", config.Defaults().WorkerAttempts)
	}
	for _, tc := range []struct {
		name    string
		safety  api.SafetyStatus
		stale   bool
		failure bool
		reason  string
	}{
		{"failed_current", api.SafetyStatusFailed, false, true, "safety_failed"},
		{"expired_current", api.SafetyStatusExpired, false, true, "safety_expired"},
		{"failed_stale", api.SafetyStatusFailed, true, false, "stale_generation"},
		{"expired_stale", api.SafetyStatusExpired, true, false, "stale_generation"},
		{"unknown_current", api.SafetyStatusUnknown, false, false, "readiness_unconfirmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, enrolled, cert, _, _ := publicationFixture(t)
			op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)

			state := decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), 200)
			raw, err := os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
			if err != nil {
				t.Fatal(err)
			}
			var report api.ObservationReport
			if err = json.Unmarshal(raw, &report); err != nil {
				t.Fatal(err)
			}
			report.NodeId, report.NodeIncarnationId, report.ReportId = enrolled.NodeId, enrolled.NodeIncarnationId, uuid.New()
			report.ObservedAt = time.Now().UTC()
			report.AppliedConfigGeneration.Set(state.Signed.ConfigGeneration)
			if tc.stale {
				report.AppliedConfigGeneration.Set(state.Signed.ConfigGeneration - 1)
			}
			report.CurrentReleaseId.SetNull()
			report.LastGoodReleaseId.SetNull()
			report.Lease.LeaseId.SetNull()
			report.Lease.ExpiresAt.SetNull()
			report.Netbird.PeerId.SetNull()
			report.MemberForwardingEnabled = false
			report.SafetyStatus = tc.safety
			report.ProvisioningState = api.ProvisioningStateReady
			report.DataHealth = api.DataHealthHealthy
			report.Errors = []api.NodeError{}
			if tc.safety == api.SafetyStatusFailed {
				report.Errors = []api.NodeError{{
					Code: api.ErrorCodeISOLATIONPOLICYINVALID, Stage: "apply",
					Message: "conntrack delete exceeded the retry bound", ObservedAt: report.ObservedAt,
				}}
			}
			for i := range report.Checks {
				report.Checks[i].Status = api.HealthCheckStatusPassed
				report.Checks[i].ObservedAt = report.ObservedAt
				report.Checks[i].ErrorCode.SetNull()
			}
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)

			var safety, provisioning string
			var applied, desired decstr.U64
			if err = f.r.Pool.QueryRow(t.Context(), `SELECT safety_status, provisioning_state, applied_config_generation::text, desired_config_generation::text FROM nodes WHERE id=$1`, enrolled.NodeId).Scan(&safety, &provisioning, &applied, &desired); err != nil {
				t.Fatal(err)
			}
			if safety != string(tc.safety) || (applied < desired) != tc.stale {
				t.Fatalf("setup: safety=%s provisioning=%s applied=%s desired=%s", safety, provisioning, applied, desired)
			}
			nb := exits.NewFakeNetBirdAdapter()
			d := f.publicationDispatcher(nb, 2)
			f.publicationDispatch(d, op.OperationId)
			first := f.publicationOperation(op.OperationId)
			var stepStatus, errCode, reason, pubObserved, opStatus string
			var opError *string
			if err = f.r.Pool.QueryRow(t.Context(), `
		SELECT o.status, o.error_code,
		       o.steps->'agent_policy'->>'status',
		       o.steps->'agent_policy'->>'error_code',
		       o.steps->'agent_policy'->>'reason',
		       e.publication_observed
		FROM operations o JOIN exits e ON e.node_id=o.resource_id
		WHERE o.id=$1`, op.OperationId).Scan(&opStatus, &opError, &stepStatus, &errCode, &reason, &pubObserved); err != nil {
				t.Fatal(err)
			}
			wantStep, wantObserved := "pending", "absent"
			if tc.failure {
				wantStep, wantObserved = "failed", "error"
			}
			// Non-gate failures retain failure evidence while the worker returns the
			// operation to running for a scheduled retry, just like adapter failures.
			if opStatus != "running" || first.Status != api.OperationStatusRunning || stepStatus != wantStep || reason != tc.reason || pubObserved != wantObserved || errCode != "NODE_NOT_SERVICEABLE" || (opError != nil) != tc.failure {
				t.Fatalf("attempt 1: operation=%s error=%v step=%s code=%s reason=%s publication=%s", opStatus, opError, stepStatus, errCode, reason, pubObserved)
			}
			if tc.failure && *opError != "NODE_NOT_SERVICEABLE" {
				t.Fatalf("operation error=%s", *opError)
			}

			f.publicationDispatch(d, op.OperationId)
			exhausted := f.publicationOperation(op.OperationId)
			if err = f.r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id=$1`, enrolled.NodeId).Scan(&pubObserved); err != nil {
				t.Fatal(err)
			}
			if exhausted.Status != api.OperationStatusFailed || exhausted.Error.IsNull() || pubObserved != "error" {
				t.Fatalf("exhaustion: status=%s error=%v publication=%s", exhausted.Status, exhausted.Error, pubObserved)
			}
		})
	}
}
