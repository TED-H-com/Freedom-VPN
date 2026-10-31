//go:build integration

package nodes_test

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReviewR15_FrozenObservationStoresEffectiveSafety proves that every new
// observation row carries the frozen override, including out-of-order reports.
// The immutable raw payload remains the node's claim, for hash/replay evidence.
func TestReviewR15_FrozenObservationStoresEffectiveSafety(t *testing.T) {
	r, enrolled, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode([]byte(enrolled.Response.CertificatePem))
	if block == nil {
		t.Fatal("missing enrolled leaf")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"198.51.100.0/24", "203.0.113.0/24", "198.51.100.0/24"} {
		if _, err := r.Enrollment.ObserveEndpoint(ctx, cert, domain, 120*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	for _, age := range []time.Duration{time.Second, -time.Second} {
		report.ReportId = uuid.New()
		report.ObservedAt = report.ObservedAt.Add(age)
		report.MemberForwardingEnabled = true
		accepted, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
		if err != nil {
			t.Fatal(err)
		}
		var safety, nodeSafety, provisioning string
		var forwarding bool
		var payload []byte
		if err := r.Pool.QueryRow(ctx, `SELECT o.safety_status, n.safety_status, o.provisioning_state, o.member_forwarding_enabled, o.report_payload FROM node_observations o JOIN nodes n ON n.id=o.node_id WHERE o.report_id=$1`, report.ReportId).Scan(&safety, &nodeSafety, &provisioning, &forwarding, &payload); err != nil {
			t.Fatal(err)
		}
		if safety != "failed" || nodeSafety != "failed" || provisioning != "failed" || forwarding {
			t.Fatalf("frozen evidence: observation=%s node=%s provisioning=%s forwarding=%v", safety, nodeSafety, provisioning, forwarding)
		}
		var raw api.ObservationReport
		if err := json.Unmarshal(payload, &raw); err != nil {
			t.Fatal(err)
		}
		if raw.SafetyStatus != api.SafetyStatusPassed || !raw.MemberForwardingEnabled {
			t.Fatal("raw report was rewritten instead of retaining the agent's claim")
		}
		replay, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
		if err != nil || !replay.ReceivedAt.Equal(accepted.ReceivedAt) {
			t.Fatalf("frozen observation replay changed: %v", err)
		}
	}
}
