//go:build integration

package nodes_test

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestAT06_ObservationBaselineReadinessWithoutManagementPeer(t *testing.T) {
	for _, name := range []string{"healthy", "degraded", "failed_safety", "failed_evidence", "behind_generation", "missing_generation", "present_peer"} {
		t.Run(name, func(t *testing.T) {
			r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
			switch name {
			case "healthy":
				report.DataHealth = api.DataHealthHealthy
			case "failed_safety":
				report.SafetyStatus = api.SafetyStatusFailed
			case "failed_evidence":
				report.Checks[0].Status = api.HealthCheckStatusFailed
			case "behind_generation":
				report.AppliedConfigGeneration.Set(0)
			case "missing_generation":
				report.AppliedConfigGeneration.SetNull()
			case "present_peer":
				report.Netbird.PeerId.Set("TEST ONLY peer")
			}
			first, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report)
			if err != nil || first != replay {
				t.Fatal("observation replay", replay, err)
			}
			detail, _, err := r.Nodes.GetNode(t.Context(), report.NodeId)
			if err != nil {
				t.Fatal(err)
			}
			wantReady := name == "healthy" || name == "degraded"
			wantState := api.ProvisioningStateApplying
			if wantReady {
				wantState = api.ProvisioningStateReady
			}
			applied, err := detail.ConfigGeneration.Applied.Get()
			if err != nil || detail.ProvisioningState != wantState || !detail.NetbirdPeerId.IsNull() || detail.PublicationIntent != api.PublicationIntentUnpublished {
				t.Fatal("baseline/binding projection", detail, err)
			}
			readiness := nodes.ServingReadiness{LifecycleState: detail.LifecycleState, ProvisioningState: detail.ProvisioningState, PublicationIntent: detail.PublicationIntent, ControlStatus: detail.ControlStatus, SafetyStatus: detail.SafetyStatus, DataHealth: detail.DataHealth, DesiredConfigGeneration: uint64(detail.ConfigGeneration.Desired), AppliedConfigGeneration: uint64(applied)}
			if err := nodes.CheckPublishable(readiness); (err == nil) != wantReady {
				t.Fatal("publication acceptance", readiness, err)
			}
			if wantReady && (detail.SafetyStatus != api.SafetyStatusPassed || applied != detail.ConfigGeneration.Desired || readiness.IsServing()) {
				t.Fatal("readiness must not imply member serving", detail)
			}
			if (name == "failed_safety" || name == "failed_evidence") && detail.SafetyStatus != api.SafetyStatusFailed {
				t.Fatal("failed evidence was not retained", detail)
			}
			var receipts, events int
			if err := r.Pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM node_observations WHERE report_id=$1),(SELECT count(*) FROM outbox_events WHERE event_type='node.peer.observe' AND payload->>'report_id'=$1::text)`, report.ReportId).Scan(&receipts, &events); err != nil || receipts != 1 || events != 1 {
				t.Fatal("deduplicated store/worker intent", receipts, events, err)
			}
			if wantReady {
				// An older failed report stays in history and cannot undo the
				// current baseline acknowledgement.
				report.ReportId, report.ObservedAt = uuid.New(), report.ObservedAt.Add(-time.Second)
				report.SafetyStatus = api.SafetyStatusFailed
				if _, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
					t.Fatal(err)
				}
				after, _, err := r.Nodes.GetNode(t.Context(), report.NodeId)
				if err != nil || after.ProvisioningState != api.ProvisioningStateReady || after.SafetyStatus != api.SafetyStatusPassed {
					t.Fatal("stale report regressed readiness", after, err)
				}
			}
		})
	}
}
