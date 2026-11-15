//go:build integration

package worker

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestNetBirdRoutesReconcile_ScheduleCoalescesAndReconciles(t *testing.T) {
	const eventType = "node.netbird_routes.reconcile"
	r, enrolled, _ := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	if _, err := r.Pool.Exec(t.Context(), `UPDATE exits SET publication_intent='published', resource_version=resource_version+1 WHERE node_id=$1`, enrolled.Response.NodeId); err != nil {
		t.Fatal(err)
	}
	if err := nodeScheduleCycle(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	if err := nodeScheduleCycle(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	var pending int
	if err := r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type=$1 AND delivered_at IS NULL AND dead_lettered_at IS NULL`, eventType).Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("undelivered %s = %d err=%v", eventType, pending, err)
	}
	if _, err := r.Pool.Exec(t.Context(), `UPDATE outbox_events SET available_at=CASE WHEN event_type=$1 THEN clock_timestamp() ELSE clock_timestamp()+interval '1 day' END WHERE delivered_at IS NULL AND dead_lettered_at IS NULL`, eventType); err != nil {
		t.Fatal(err)
	}
	claims, err := db.ClaimOutbox(t.Context(), r.Pool, "TEST ONLY netbird routes", 1, time.Minute)
	if err != nil || len(claims) != 1 || claims[0].EventType != eventType {
		t.Fatalf("claim count=%d err=%v", len(claims), err)
	}
	nb := exits.NewFakeNetBirdAdapter()
	jobs := NodeJobs(r, nil, nil, 3, nb)
	receipt, err := jobs[eventType](t.Context(), claims[0])
	if err != nil {
		t.Fatal(err)
	}
	wantReceipt := "netbird-routes-reconciliation:" + claims[0].ID.String()
	if receipt != wantReceipt {
		t.Fatalf("receipt %q want %q", receipt, wantReceipt)
	}
	var field string
	if err := r.Pool.QueryRow(t.Context(), `SELECT elem->>'field' FROM nodes n, jsonb_array_elements(n.drift) elem WHERE n.id=$1`, enrolled.Response.NodeId).Scan(&field); err != nil || field != "netbird_route" {
		t.Fatalf("drift field=%s err=%v", field, err)
	}
	var intent string
	if err := r.Pool.QueryRow(t.Context(), `SELECT publication_intent FROM exits WHERE node_id=$1`, enrolled.Response.NodeId).Scan(&intent); err != nil || intent != string(api.PublicationIntentPublished) {
		t.Fatalf("publication_intent=%s err=%v", intent, err)
	}
	if nb.PublishCalls() != 0 || nb.WithdrawCalls() != 0 {
		t.Fatalf("publish=%d withdraw=%d", nb.PublishCalls(), nb.WithdrawCalls())
	}
	var successor uuid.UUID
	if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM outbox_events WHERE event_type=$1 AND delivered_at IS NULL AND dead_lettered_at IS NULL AND id<>$2`, eventType, claims[0].ID).Scan(&successor); err != nil || successor == claims[0].ID {
		t.Fatalf("successor=%s err=%v", successor, err)
	}
	d, err := New(r.Pool, jobs, Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Second, MaxBackoff: time.Second}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.complete(t.Context(), claims[0], receipt); err != nil {
		t.Fatal(err)
	}
	var delivered, still int
	if err := r.Pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE delivered_at IS NOT NULL), count(*) FILTER (WHERE delivered_at IS NULL AND dead_lettered_at IS NULL) FROM outbox_events WHERE event_type=$1`, eventType).Scan(&delivered, &still); err != nil || delivered != 1 || still != 1 {
		t.Fatalf("delivered=%d undelivered=%d err=%v", delivered, still, err)
	}
}
