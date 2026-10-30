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
	raw, err := os.ReadFile("../../docs/e2e/at/secops/live.json")
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
	report := api.ObservationReport{DataHealth: record.NodeAfterPublication.DataHealth, SafetyStatus: api.SafetyStatusPassed, MemberForwardingEnabled: record.FlowProbe.NodeServing}
	report.AppliedConfigGeneration = record.NodeAfterPublication.ConfigGeneration.Applied
	for _, name := range []api.NodeCheckName{api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
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
