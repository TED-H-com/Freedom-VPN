package releasetest

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AssertLedger reads domain writes through the public audit/operation stores.
// D-029: SQL insertion alone cannot prove records are usable by control-core.
func AssertLedger(t testing.TB, pool *pgxpool.Pool, validator *releases.Validator, wantActions ...string) {
	t.Helper()
	ctx := context.Background()
	rows, err := pool.Query(ctx, `SELECT id FROM operations ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatal(err)
	}
	ops := map[uuid.UUID]api.Operation{}
	for _, id := range ids {
		op, err := (operations.Store{Pool: pool}).Get(ctx, id, auth.Principal{Role: auth.Admin, ID: uuid.New()})
		if err != nil {
			t.Fatal(err)
		}
		ops[id] = op
		raw, err := json.Marshal(op)
		if err != nil {
			t.Fatal(err)
		}
		if err = validator.ValidateBody("Operation", raw); err != nil {
			t.Fatalf("operation contract: %v: %s", err, raw)
		}
	}
	store := audit.Store{Pool: pool}
	var cursor *string
	count := 0
	actions := map[string]int{}
	limit := 2 // Exercise reader cursor pages even in the smallest writer fixture.
	for {
		page, err := store.List(ctx, api.ListAuditEventsParams{Cursor: cursor, Limit: &limit})
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range page.Items {
			actor, actorErr := event.Actor.Id.Get()
			request, requestErr := event.RequestId.Get()
			if actorErr != nil || actor == uuid.Nil || requestErr != nil || request == uuid.Nil {
				t.Fatalf("audit identity/correlation: %+v", event)
			}
			if opID, err := event.OperationId.Get(); err == nil {
				op, ok := ops[opID]
				if !ok || op.Resource != event.Resource || op.RequestId != request {
					t.Fatalf("audit/operation correlation: %+v", event)
				}
			}
			assertWriter(t, ctx, pool, event, actor)
			actions[event.Action]++
		}
		count += len(page.Items)
		next, err := page.NextCursor.Get()
		if err != nil {
			break
		}
		cursor = &next
	}
	var persisted int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	t.Logf("D-029 readback: %d operations, %d audit events, %d anomalies", len(ids), count, store.AnomalyCount())
	t.Logf("D-029 writers: %v", actions)
	for _, action := range wantActions {
		if actions[action] == 0 {
			t.Errorf("writer not exercised: %s", action)
		}
	}
	if store.AnomalyCount() != 0 || count != persisted || count == 0 {
		t.Fatalf("audit persisted=%d returned=%d anomalies=%d", persisted, count, store.AnomalyCount())
	}
}

// The reader accepts historical system actors without UUIDs. Check producers
// more strictly, and resolve actual entities so changing a prefix cannot hide
// a job/result UUID incorrectly relabeled as a target or node (D-029).
func assertWriter(t testing.TB, ctx context.Context, pool *pgxpool.Pool, event api.AuditEvent, actor uuid.UUID) {
	t.Helper()
	after, err := event.Changes.After.Get()
	if err != nil {
		t.Fatal(err)
	}
	wantActor, wantResource, wantResult := api.AuditEventActorTypeAdmin, api.ResourceTypeRelease, api.AuditEventResultSucceeded
	var fields []string
	switch event.Action {
	case "release.register":
		fields = []string{"verification_state"}
	case "release.verify":
		wantActor = api.AuditEventActorTypeSystem
		fields = []string{"verification_state", "error_code"}
		if after["verification_state"] == "failed" {
			wantResult = api.AuditEventResultFailed
		}
	case "release_support.activate":
		fields = []string{"approved", "serving", "revoked"}
	case "update_batch.create":
		wantResource = api.ResourceTypeUpdateBatch
		fields = []string{"target_count", "release_id"}
	case "update_batch.retry":
		wantResource = api.ResourceTypeUpdateBatch
		fields = []string{"target_count"}
	case "update_batch.first_sync":
		wantActor, wantResource = api.AuditEventActorTypeSystem, api.ResourceTypeUpdateBatch
		fields = []string{"node_id", "release_id"}
	case "update_result.accept":
		wantActor, wantResource = api.AuditEventActorTypeNode, api.ResourceTypeUpdateTarget
		fields = []string{"state", "accepted_as_history", "result_id", "job_id"}
	case "update_result.rollback_refused":
		wantActor, wantResource, wantResult = api.AuditEventActorTypeNode, api.ResourceTypeUpdateTarget, api.AuditEventResultFailed
		fields = []string{"recovery_required", "result_id", "job_id"}
	case "update_result.conflict":
		wantActor, wantResource, wantResult = api.AuditEventActorTypeNode, api.ResourceTypeNode, api.AuditEventResultFailed
		fields = []string{"result_id", "conflicting_hash"}
	default:
		t.Fatalf("uncovered writer: %s", event.Action)
	}
	if event.Actor.Type != wantActor || event.Resource.Type != wantResource || event.Result != wantResult {
		t.Fatalf("writer metadata: %+v", event)
	}
	if wantActor == api.AuditEventActorTypeSystem && actor != audit.SystemActor.ID {
		t.Fatalf("wrong system identity: %s", actor)
	}
	for field := range after {
		if !slices.Contains(fields, field) {
			t.Fatalf("unallowlisted audit field: %s in %s", field, event.Action)
		}
	}
	exists := func(query string, args ...any) {
		t.Helper()
		var found bool
		if err := pool.QueryRow(ctx, query, args...).Scan(&found); err != nil || !found {
			t.Fatalf("audit entity mismatch: action=%s resource=%+v error=%v", event.Action, event.Resource, err)
		}
	}
	switch wantResource {
	case api.ResourceTypeRelease:
		exists(`SELECT EXISTS(SELECT 1 FROM releases WHERE id=$1)`, event.Resource.Id)
	case api.ResourceTypeUpdateBatch:
		exists(`SELECT EXISTS(SELECT 1 FROM update_batches WHERE id=$1)`, event.Resource.Id)
	case api.ResourceTypeUpdateTarget:
		exists(`SELECT EXISTS(SELECT 1 FROM update_targets t JOIN update_jobs j ON j.target_id=t.id WHERE t.id=$1 AND t.node_id=$2 AND j.id=$3)`, event.Resource.Id, actor, after["job_id"])
	case api.ResourceTypeNode:
		if event.Resource.Id != actor {
			t.Fatal("conflict resource must identify the authenticated node")
		}
		exists(`SELECT EXISTS(SELECT 1 FROM update_results r JOIN update_result_conflicts c USING(result_id) WHERE r.result_id=$1 AND r.node_id=$2 AND encode(c.conflicting_hash,'hex')=$3)`, after["result_id"], actor, after["conflicting_hash"])
	}
	if event.Action == "update_result.accept" {
		exists(`SELECT EXISTS(SELECT 1 FROM update_results WHERE result_id=$1 AND node_id=$2 AND job_id=$3)`, after["result_id"], actor, after["job_id"])
	}
}
