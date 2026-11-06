//go:build integration

package members

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestD164_ImmutableSQLTimelinePreservesServingReads(t *testing.T) {
	r, _, base := nodestest.BaselineObservation(t, "../../../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	now := base.ObservedAt.Add(time.Hour)
	since := now.Format(time.RFC3339Nano)
	svc := nodes.NewService(r.Pool, nodes.WithClock(func() time.Time { return now }))
	evidence := api.ServingReadEvidence{ReadAt: base.ObservedAt.Add(-time.Hour).Add(123456789 * time.Nanosecond), Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: decstr.U64(^uint64(0)), Verified: false, Diagnostic: "TEST ONLY clipped {"}
	reads := []presence.Nullable[api.ServingReadEvidence]{{}, presence.NewNullNullable[api.ServingReadEvidence](), presence.NewNullableWithValue(evidence)}
	evidence.ReadAt = evidence.ReadAt.Add(time.Nanosecond)
	reads = append(reads, presence.NewNullableWithValue(evidence))
	evidence.Source = api.ServingReadEvidenceSourceApply
	reads = append(reads, presence.NewNullableWithValue(evidence))
	evidence.CandidateGeneration = decstr.U64(9007199254740993)
	reads = append(reads, presence.NewNullableWithValue(evidence))
	evidence.Verified = true
	reads = append(reads, presence.NewNullableWithValue(evidence))
	evidence.Diagnostic = strings.Repeat("x", 499) + "{"
	reads = append(reads, presence.NewNullableWithValue(evidence))
	post := func(report api.ObservationReport) {
		t.Helper()
		report.ReportId = uuid.New()
		if _, err := svc.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Microsecond)
	}
	for _, read := range reads {
		report := base
		report.ServingRead = read
		// Identical evidence compresses despite distinct report IDs and receipts.
		post(report)
		post(report)
	}
	// A real reinstall changes only incarnation in the exported observation state.
	detail, _, err := r.Nodes.GetNode(ctx, base.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	version := detail.ResourceVersion
	if _, err = r.Enrollment.RevokeNode(ctx, enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, base.NodeId, &version, "TEST ONLY diagnostic timeline"); err != nil {
		t.Fatal(err)
	}
	request := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller}
	request.NodeId = &base.NodeId
	created, err := r.Enrollment.CreateEnrollment(ctx, enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, request)
	if err != nil {
		t.Fatal(err)
	}
	token, err := created.Token.Get()
	if err != nil {
		t.Fatal(err)
	}
	csr, _ := pkitest.Request(t)
	enrolled, err := r.Enrollment.Enroll(ctx, token, csr)
	if err != nil {
		t.Fatal(err)
	}
	report := base
	report.NodeIncarnationId = enrolled.Response.NodeIncarnationId
	report.ServingRead = reads[len(reads)-1]
	post(report)
	snapshot := func() string {
		t.Helper()
		var out string
		if err := r.Pool.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(o) ORDER BY id)::text FROM node_observations o WHERE node_id=$1`, base.NodeId).Scan(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := snapshot()
	leases, raw, ok := readNodeLeaseTimeline(ctx, r.Pool, base.NodeId, since)
	if !ok || !bytes.Equal(leases, []byte("[]")) {
		t.Fatal("timeline query failed", ok, string(leases))
	}
	var groups []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &groups); err != nil {
		t.Fatal(err)
	}
	if len(groups) != len(reads)+1 {
		t.Fatalf("read/incarnation changes compressed: %d groups, want %d: %s", len(groups), len(reads)+1, raw)
	}
	for i, group := range groups {
		expected := reads[min(i, len(reads)-1)]
		got, specified := group["serving_read"]
		if specified != expected.IsSpecified() {
			t.Fatal("presence lost", i, string(raw))
		}
		if specified {
			want, err := json.Marshal(expected)
			if err != nil {
				t.Fatal(err)
			}
			a, err := canonjson.Canonicalize(got)
			if err != nil {
				t.Fatal(err)
			}
			b, err := canonjson.Canonicalize(want)
			if err != nil || !bytes.Equal(a, b) {
				t.Fatal("read changed", i, string(got), string(want), err)
			}
		}
		wantCount := "2"
		inc := base.NodeIncarnationId
		if i == len(reads) {
			wantCount = "1"
			inc = enrolled.Response.NodeIncarnationId
		}
		if string(group["observations"]) != wantCount || string(group["node_incarnation_id"]) != `"`+inc.String()+`"` {
			t.Fatal("compression/incarnation lost", i, string(raw))
		}
		for _, key := range []string{"generation", "forwarding", "reasons", "safety_status", "lease_check", "qos_check", "source_binding_check", "errors"} {
			if _, ok := group[key]; !ok || !reflect.DeepEqual(group[key], groups[0][key]) {
				t.Fatal("legacy diagnostic column changed", key, i)
			}
		}
	}
	_, again, ok := readNodeLeaseTimeline(ctx, r.Pool, base.NodeId, since)
	if !ok || !bytes.Equal(again, raw) || before != snapshot() {
		t.Fatal("read mutated payload/hash history or changed order")
	}
}

func TestD164_SQLTimelineEqualReceiptOrdering(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	now := report.ObservedAt.Add(time.Hour)
	svc := nodes.NewService(r.Pool, nodes.WithClock(func() time.Time { return now }))
	ids := []string{"ffffffff-ffff-4fff-8fff-ffffffffffff", "00000000-0000-4000-8000-000000000001"}
	for _, id := range ids {
		// Fixture default controls immutable row identity before the real insert.
		if _, err := r.Pool.Exec(ctx, "ALTER TABLE node_observations ALTER COLUMN id SET DEFAULT '"+id+"'::uuid"); err != nil {
			t.Fatal(err)
		}
		report.ReportId = uuid.New()
		report.ServingRead.Set(api.ServingReadEvidence{ReadAt: report.ObservedAt, Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: 1, Verified: false, Diagnostic: id})
		if _, err := svc.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Pool.Exec(ctx, "ALTER TABLE node_observations ALTER COLUMN id SET DEFAULT gen_random_uuid()"); err != nil {
		t.Fatal(err)
	}
	_, raw, ok := readNodeLeaseTimeline(ctx, r.Pool, report.NodeId, now.Format(time.RFC3339Nano))
	if !ok {
		t.Fatal("timeline failed")
	}
	var groups []struct {
		Read  api.ServingReadEvidence `json:"serving_read"`
		Count int                     `json:"observations"`
	}
	if err := json.Unmarshal(raw, &groups); err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Read.Diagnostic != ids[1] || groups[1].Read.Diagnostic != ids[0] || groups[0].Count != 1 || groups[1].Count != 1 {
		t.Fatal("equal receipts lost deterministic history", string(raw))
	}
}
