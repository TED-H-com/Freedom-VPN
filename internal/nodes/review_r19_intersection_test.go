package nodes

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestReviewR19_Q3_ProposedReportIsRejectedAndIsNotTheHold(t *testing.T) {
	nodeID, incID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	report := proposedIntersectionReport(nodeID, incID)
	_, err := (&Service{}).SubmitObservation(context.Background(), nodeID, incID, report)
	var ae *apierr.Error
	if err == nil || !asAPI(err, &ae) || ae.Code != apierr.ValidationFailed {
		t.Fatalf("generation_intersection was not rejected before a database write: %v", err)
	}
	stripped := report
	stripped.ForwardingBlockReasons = nil
	if reportsGenerationHold(stripped) {
		t.Fatal("STALE_GENERATION without policy_missing is the follow-up 34 hold")
	}
	withMissing := stripped
	withMissing.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonPolicyMissing}
	if !reportsGenerationHold(withMissing) {
		t.Fatal("policy_missing plus the lease STALE_GENERATION check is no longer the hold")
	}
	if cause := publicationForwardingFailure(stripped, stripped.ObservedAt, false); cause != "check_failed" {
		t.Fatalf("failed lease check cause=%s want check_failed", cause)
	}
	ready := ServingReadiness{
		LifecycleState: api.LifecycleStateRegistered, ProvisioningState: api.ProvisioningStateReady,
		PublicationIntent: api.PublicationIntentPublished, ControlStatus: api.ControlStatusOnline,
		SafetyStatus: api.SafetyStatusPassed, DataHealth: api.DataHealthHealthy,
		DesiredConfigGeneration: 5, AppliedConfigGeneration: 4,
	}
	if ready.IsServing() {
		t.Fatal("applied generation 4 against desired 5 is serving")
	}
	ready.AppliedConfigGeneration = 5
	if !ready.IsServing() {
		t.Fatal("matched generations failed IsServing; the forwarding bit is not an input and this fixture should pass")
	}
	forwardingOnly := api.ObservationReport{
		MemberForwardingEnabled: true, SafetyStatus: api.SafetyStatusPassed,
		DataHealth: api.DataHealthHealthy, ProvisioningState: api.ProvisioningStateReady,
	}
	forwardingOnly.AppliedConfigGeneration.Set(5)
	if publicationForwardingFailure(forwardingOnly, time.Now().UTC(), false) == "" {
		t.Fatal("member_forwarding_enabled=true alone was serving evidence")
	}
	src, err := os.ReadFile("publication_evidence.go")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(src, []byte("generation_intersection")) {
		t.Fatal("a partial-service cause now exists; the review assumption is stale")
	}
	t.Fatalf("TODO(review): SubmitObservation rejects forwarding_block_reason generation_intersection with VALIDATION_FAILED before SQL. Stripping the token leaves a failed STALE_GENERATION lease check that is check_failed, not reportsGenerationHold, unless policy_missing is added. IsServing is false while applied 4 < desired 5 and does not read the forwarding bit. PublicationDataPathFailure has no distinct partial-service cause")
}

func proposedIntersectionReport(nodeID, incID uuid.UUID) api.ObservationReport {
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	detail := "Serving lease is not for the applied generation."
	r := api.ObservationReport{
		NodeId: nodeID, NodeIncarnationId: incID, ReportId: uuid.Must(uuid.NewV7()), BootId: uuid.Must(uuid.NewV7()),
		ObservedAt: at, ProvisioningState: api.ProvisioningStateReady, SafetyStatus: api.SafetyStatusPassed,
		DataHealth: api.DataHealthHealthy, MemberForwardingEnabled: true,
		ForwardingBlockReasons: []api.ForwardingBlockReason{"generation_intersection"},
		Checks: []api.NodeCheck{{
			Name: api.NodeCheckNameLease, Status: api.HealthCheckStatusFailed, ObservedAt: at,
			ErrorCode: presence.NewNullableWithValue(api.ErrorCodeSTALEGENERATION), Detail: &detail,
		}},
		Errors: []api.NodeError{{
			Stage: "lease", Code: api.ErrorCodeSTALEGENERATION, ObservedAt: at,
			Message: "Serving lease is not for the desired generation.",
		}},
	}
	r.AppliedConfigGeneration.Set(4)
	r.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	return r
}

func asAPI(err error, out **apierr.Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*apierr.Error)
	if !ok {
		return false
	}
	*out = e
	return true
}
