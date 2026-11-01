//go:build integration

package worker

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestAT29_M3_DurableHoldDeadlineWorkerAndClaimFence(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	start := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	now := start
	r.Nodes = nodelib.NewService(r.Pool, nodelib.WithClock(func() time.Time { return now }))
	generation, _ := report.AppliedConfigGeneration.Get()
	leaseID := uuid.New()
	// TEST ONLY issuer row, isolated database. The deadline job uses the same
	// production observation path and dispatcher claim fence as other node jobs.
	env, err := json.Marshal(map[string]any{"signed": api.ServingLease{LeaseId: leaseID, NodeId: report.NodeId,
		NodeIncarnationId: report.NodeIncarnationId, ConfigGeneration: generation}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash,envelope)
		VALUES($1,$2,$3,$4::text::numeric,$5,$6,'TEST ONLY',decode(repeat('00',32),'hex'),$7)`, leaseID, report.NodeId, report.NodeIncarnationId, generation.String(), start, start.Add(time.Hour), env); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `UPDATE nodes SET desired_config_generation=desired_config_generation+1,resource_version=resource_version+1 WHERE id=$1`, report.NodeId); err != nil {
		t.Fatal(err)
	}
	report.AppliedConfigGeneration.Set(generation + 1)
	report.Lease.LeaseId.Set(leaseID)
	report.Lease.ExpiresAt.Set(start.Add(time.Hour))
	report.Errors = nil // absent summary is unknown; the central deadline still runs
	report.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonPolicyMissing}
	for i := range report.Checks {
		if report.Checks[i].Name == api.NodeCheckNameLease {
			report.Checks[i].Status = api.HealthCheckStatusFailed
			report.Checks[i].ErrorCode.Set(api.ErrorCodeSTALEGENERATION)
		}
	}
	if _, err = r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, nodelib.EventGenerationHoldCheck); err != nil {
		t.Fatal(err)
	}
	events, err := db.ClaimOutbox(ctx, r.Pool, "TEST ONLY hold worker", 1, time.Minute)
	if err != nil || len(events) != 1 || events[0].EventType != nodelib.EventGenerationHoldCheck {
		t.Fatalf("claim=%v err=%v", events, err)
	}
	now = start.Add(30 * time.Second)
	handler := NodeJobs(r, nil, nil, 3)[nodelib.EventGenerationHoldCheck]
	lost := events[0]
	lost.LeaseToken = uuid.New()
	if _, err = handler(ctx, lost); !errors.Is(err, db.ErrLeaseLost) {
		t.Fatalf("lost claim=%v", err)
	}
	for range 2 {
		if _, err = handler(ctx, events[0]); err != nil {
			t.Fatal(err)
		}
	}
	var alarms int
	if err = r.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='node.generation_hold.escalated'`).Scan(&alarms); err != nil || alarms != 1 {
		t.Fatalf("alarms=%d err=%v", alarms, err)
	}
}
