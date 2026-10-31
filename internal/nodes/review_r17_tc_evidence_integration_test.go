//go:build integration

package nodes_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func reviewTCEvidence(sha string) api.TCReadbackEvidence {
	ev := api.TCReadbackEvidence{
		Attachments:         []api.TCReadAttachment{},
		CandidateGeneration: 7,
		ObjectsTotal:        1,
		Phase:               api.TCReadbackEvidencePhaseVerify,
		ReadAt:              time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC),
		ReadsTotal:          1,
		Sha256:              sha,
		Truncated:           false,
		Verified:            false,
	}
	ev.ErrorCode.Set(api.ErrorCodeTCFOREIGNFILTER)
	ev.Rejected.SetNull()
	return ev
}

func TestReviewR17_CentralStoresTCEvidenceWithoutRehashing(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	stale := strings.Repeat("ab", 32)
	report.TcEvidence.Set(reviewTCEvidence(stale))
	if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	var payload string
	if err := r.Pool.QueryRow(ctx, `SELECT report_payload::text FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, stale) {
		t.Fatalf("stored evidence dropped the presented digest: %s", payload)
	}

	replay := report
	replay.TcEvidence.Set(reviewTCEvidence(strings.Repeat("cd", 32)))
	_, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, replay)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.AckConflict {
		t.Fatalf("same report id with a different tc_evidence: %v", err)
	}
	var n int
	if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows=%d %v", n, err)
	}

	fresh := report
	fresh.ReportId = uuid.New()
	fresh.ObservedAt = report.ObservedAt.Add(time.Second)
	fresh.TcEvidence.Set(reviewTCEvidence(stale))
	if _, err := r.Nodes.SubmitObservation(ctx, fresh.NodeId, fresh.NodeIncarnationId, fresh); err != nil {
		t.Fatal(err)
	}
	var again string
	if err := r.Pool.QueryRow(ctx, `SELECT report_payload::text FROM node_observations WHERE report_id=$1`, fresh.ReportId).Scan(&again); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again, stale) || strings.Contains(again, strings.Repeat("cd", 32)) {
		t.Fatal("a new report id did not store the copied digest unchanged")
	}
}
