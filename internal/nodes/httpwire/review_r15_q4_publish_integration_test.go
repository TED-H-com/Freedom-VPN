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

// TestReviewR15_SafetyFailedPublishWaitsAsReadinessUnconfirmed reports
// safety_status=failed at the desired generation and runs the publish worker.
// CheckPublishable treats that as not serviceable. executePublish labels every
// such failure readiness_unconfirmed when the generation is current, and the
// operation stays running until the worker attempt budget is spent.
func TestReviewR15_SafetyFailedPublishWaitsAsReadinessUnconfirmed(t *testing.T) {
	if config.Defaults().WorkerAttempts != 10 {
		t.Fatalf("WorkerAttempts = %d, want the production default 10", config.Defaults().WorkerAttempts)
	}
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
	report.CurrentReleaseId.SetNull()
	report.LastGoodReleaseId.SetNull()
	report.Lease.LeaseId.SetNull()
	report.Lease.ExpiresAt.SetNull()
	report.Netbird.PeerId.SetNull()
	report.MemberForwardingEnabled = false
	report.SafetyStatus = api.SafetyStatusFailed
	report.ProvisioningState = api.ProvisioningStateReady
	report.DataHealth = api.DataHealthHealthy
	report.Errors = []api.NodeError{{
		Code: api.ErrorCodeISOLATIONPOLICYINVALID, Stage: "apply",
		Message: "conntrack delete exceeded the retry bound", ObservedAt: report.ObservedAt,
	}}
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
	if safety != string(api.SafetyStatusFailed) || applied < desired {
		t.Fatalf("setup: safety=%s provisioning=%s applied=%s desired=%s, want failed at or past desired", safety, provisioning, applied, desired)
	}

	nb := exits.NewFakeNetBirdAdapter()
	d := f.publicationDispatcher(nb, 2)
	f.publicationDispatch(d, op.OperationId)
	waiting := f.publicationOperation(op.OperationId)
	var stepStatus, errCode, reason, pubObserved, opStatus string
	var opError *string
	if err = f.r.Pool.QueryRow(t.Context(), `
		SELECT o.status, o.error_code,
		       o.steps->'agent_policy'->>'status',
		       o.steps->'agent_policy'->>'error_code',
		       o.steps->'agent_policy'->>'reason',
		       e.publication_observed
		FROM operations o
		JOIN exits e ON e.node_id = o.resource_id
		WHERE o.id = $1`, op.OperationId).Scan(&opStatus, &opError, &stepStatus, &errCode, &reason, &pubObserved); err != nil {
		t.Fatal(err)
	}
	if opStatus != "running" || opError != nil || stepStatus != "pending" || reason != "readiness_unconfirmed" || pubObserved != "absent" {
		t.Fatalf("safety=failed gate on attempt 1 = op %s error %v step %s reason %s observed %s (API status %s); a broken node was not held as readiness_unconfirmed",
			opStatus, opError, stepStatus, reason, pubObserved, waiting.Status)
	}

	f.publicationDispatch(d, op.OperationId)
	exhausted := f.publicationOperation(op.OperationId)
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT o.status, e.publication_observed FROM operations o JOIN exits e ON e.node_id=o.resource_id WHERE o.id=$1`, op.OperationId).Scan(&opStatus, &pubObserved); err != nil {
		t.Fatal(err)
	}
	if exhausted.Status != api.OperationStatusFailed || opStatus != "failed" || pubObserved != "error" {
		t.Fatalf("exhaustion = api %s db %s observed %s, want failed/error", exhausted.Status, opStatus, pubObserved)
	}
	t.Fatalf("TODO(review): safety_status=failed at applied %s >= desired %s (provisioning %s) is gated as agent_policy reason readiness_unconfirmed. The operation stayed running with a null error and publication_observed=absent until worker attempt 2 of 2, then failed with publication_observed=error. It does not wait forever. config.Defaults WorkerAttempts is %d (backoff 1s doubling to 300s). There is no cancel operation. The portal headline is the operation status, so it does not say failed while the budget remains.",
		applied, desired, provisioning, config.Defaults().WorkerAttempts)
}
