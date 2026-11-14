//go:build integration

package worker

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	memberscore "github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func recoveryEvent(t *testing.T, pool *pgxpool.Pool, eventType string, payload any, at time.Time) uuid.UUID {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	err = db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return db.EnqueueOutbox(t.Context(), tx, db.OutboxEvent{
			ID: id, AggregateID: uuid.New(), AggregateType: "node", AggregateVersion: "1",
			EventType: eventType, SchemaVersion: 1, Payload: raw, AvailableAt: at,
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func recoveryOperation(t *testing.T, pool *pgxpool.Pool, kind string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO operations(id,kind,principal,resource_type,resource_id) VALUES($1,$2,'TEST ONLY','node',$3)`, id, kind, uuid.New()); err != nil {
		t.Fatal(err)
	}
	return id
}

func recoverySetMode(t *testing.T, pool *pgxpool.Pool, mode string) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `UPDATE usage_ledger_state SET mode=$1`, mode); err != nil {
		t.Fatal(err)
	}
}

func recoveryRemoveLedgerState(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	// Exercise a missing singleton in this disposable database only.
	if _, err := pool.Exec(t.Context(), `ALTER TABLE usage_ledger_state DISABLE TRIGGER usage_ledger_state_keep`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `DELETE FROM usage_ledger_state`); err != nil {
		t.Fatal(err)
	}
}

func recoveryClaim(t *testing.T, pool *pgxpool.Pool) []db.ClaimedEvent {
	t.Helper()
	events, err := db.ClaimOutboxByLedgerMode(t.Context(), pool, "TEST ONLY recovery", 1000, time.Minute, recoveryAllowedTypes, recoveryFilteredType, recoveryWithdrawalKinds)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestRecoveryModeT1_ClaimByMode(t *testing.T) {
	pool := dbtest.New(t)
	registration, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	// This expected set is independent of the worker's classification.
	allowed := map[string]bool{}
	for _, eventType := range []string{
		"member.entitlement_changed", "member.grant_revoked", "member.suspended", "device.authorized", "device.revoked", "netbird.device_sync", "members.reconcile",
		"billing.reconcile", "billing.inbox_received", "node.staleness.sweep", "node.generation_hold.check", "node.peer.observe", "node.resolver_documents.reconcile",
		"node.netbird_routes.reconcile", "dns.publish", "dns.feed.refresh",
	} {
		allowed[eventType] = true
	}
	all, restricted := map[uuid.UUID]string{}, map[uuid.UUID]string{}
	at := time.Now().Add(-time.Hour)
	for eventType := range registration.Jobs {
		if eventType == "node.publication_changed" {
			continue
		}
		id := recoveryEvent(t, pool, eventType, map[string]any{}, at)
		all[id] = eventType
		if allowed[eventType] {
			restricted[id] = eventType
		}
	}
	// Include the planned type even when it is not yet registered.
	if _, registered := registration.Jobs["node.netbird_routes.reconcile"]; !registered {
		id := recoveryEvent(t, pool, "node.netbird_routes.reconcile", map[string]any{}, at)
		all[id], restricted[id] = "node.netbird_routes.reconcile", "node.netbird_routes.reconcile"
	}
	for _, eventType := range []string{"node.revoked", "node.retired", "test.unknown"} {
		all[recoveryEvent(t, pool, eventType, map[string]any{}, at)] = eventType
	}
	var withdrawalEvent, withdrawalOperation uuid.UUID
	for _, kind := range []string{"node.publish", "node.unpublish", "node.disable", "node.revoke", "node.retire"} {
		op := recoveryOperation(t, pool, kind)
		id := recoveryEvent(t, pool, "node.publication_changed", map[string]any{"operation_id": op}, at)
		if kind == "node.revoke" {
			withdrawalEvent, withdrawalOperation = id, op
		}
		all[id] = kind
		if kind != "node.publish" {
			restricted[id] = kind
		}
	}
	for _, mode := range []string{"normal", "frozen", "reconciling"} {
		t.Run(mode, func(t *testing.T) {
			recoverySetMode(t, pool, mode)
			if _, err := pool.Exec(t.Context(), `UPDATE outbox_events SET lease_owner=NULL,lease_token=NULL,lease_until=NULL,attempt=0`); err != nil {
				t.Fatal(err)
			}
			got := map[uuid.UUID]string{}
			for _, event := range recoveryClaim(t, pool) {
				got[event.ID] = all[event.ID]
			}
			want := restricted
			if mode == "normal" {
				want = all
			}
			if !reflect.DeepEqual(got, want) {
				for id, name := range want {
					if _, ok := got[id]; !ok {
						t.Errorf("missing claim: %s", name)
					}
				}
				for id, name := range got {
					if _, ok := want[id]; !ok {
						t.Errorf("unexpected claim: %s", name)
					}
				}
			}
		})
	}
	t.Run("does_not_lock_ledger_or_operation", func(t *testing.T) {
		if _, err := pool.Exec(t.Context(), `UPDATE outbox_events SET lease_owner=NULL,lease_token=NULL,lease_until=NULL WHERE id=$1`, withdrawalEvent); err != nil {
			t.Fatal(err)
		}
		tx, err := pool.Begin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(t.Context()) }()
		if _, err := tx.Exec(t.Context(), `SELECT id FROM usage_ledger_state FOR UPDATE`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(t.Context(), `SELECT id FROM operations WHERE id=$1 FOR UPDATE`, withdrawalOperation); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		events, err := db.ClaimOutboxByLedgerMode(ctx, pool, "TEST ONLY unlocked reads", 1, time.Minute, recoveryAllowedTypes, recoveryFilteredType, recoveryWithdrawalKinds)
		if err != nil || len(events) != 1 || events[0].ID != withdrawalEvent {
			t.Fatalf("claim with ledger and operation locked: events=%v err=%v", events, err)
		}
	})
	t.Run("missing_ledger_state", func(t *testing.T) {
		recoveryRemoveLedgerState(t, pool)
		if _, err := pool.Exec(t.Context(), `UPDATE outbox_events SET lease_owner=NULL,lease_token=NULL,lease_until=NULL,attempt=0`); err != nil {
			t.Fatal(err)
		}
		got := map[uuid.UUID]string{}
		for _, event := range recoveryClaim(t, pool) {
			got[event.ID] = all[event.ID]
		}
		if !reflect.DeepEqual(got, restricted) {
			t.Errorf("missing ledger state claimed %d events, want %d restricted events", len(got), len(restricted))
		}
	})
}

func recoverySnapshots(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(t.Context(), `SELECT id,to_jsonb(e)::text FROM outbox_events e ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var row string
		if err := rows.Scan(&id, &row); err != nil {
			t.Fatal(err)
		}
		result[id] = row
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func recoveryDispatcher(t *testing.T, pool *pgxpool.Pool, registry Registry) *Dispatcher {
	t.Helper()
	d, err := New(pool, registry, Config{Concurrency: 1, Attempts: 2, Lease: time.Minute, JobTimeout: time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Second}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestRecoveryModeT2_WaitingRowsUntouched(t *testing.T) {
	pool := dbtest.New(t)
	recoverySetMode(t, pool, "reconciling")
	at := time.Now().Add(-time.Hour)
	first := recoveryEvent(t, pool, "devices.discover", map[string]any{}, at)
	for i, eventType := range recoveryWaitingTypes[1:] {
		id := recoveryEvent(t, pool, eventType, map[string]any{}, at.Add(time.Duration(i+1)*time.Second))
		if eventType == "node.reconcile" {
			if _, err := pool.Exec(t.Context(), `UPDATE outbox_events SET attempt=2,lease_owner='TEST ONLY',lease_token=$2,lease_until=clock_timestamp()-interval '1 minute',last_error_code='TEST_RETRY' WHERE id=$1`, id, uuid.New()); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, eventType := range []string{"node.revoked", "node.retired", "test.unknown"} {
		recoveryEvent(t, pool, eventType, map[string]any{}, at.Add(time.Minute))
	}
	op := recoveryOperation(t, pool, "node.publish")
	recoveryEvent(t, pool, "node.publication_changed", map[string]any{"operation_id": op}, at.Add(time.Minute))
	recoveryEvent(t, pool, "node.publication_changed", map[string]any{"operation_id": "invalid"}, at.Add(time.Minute))
	dead := recoveryEvent(t, pool, "test.dead", map[string]any{}, at.Add(time.Minute))
	if _, err := pool.Exec(t.Context(), `UPDATE outbox_events SET attempt=4,dead_lettered_at=clock_timestamp(),last_error_code='TEST_FAILURE',available_at='infinity' WHERE id=$1`, dead); err != nil {
		t.Fatal(err)
	}
	before := recoverySnapshots(t, pool)
	d := recoveryDispatcher(t, pool, Registry{})
	for i := 0; i < 20; i++ {
		if worked, err := d.Once(t.Context()); err != nil || worked {
			t.Fatalf("restricted dispatch %d: worked=%v err=%v", i, worked, err)
		}
	}
	if after := recoverySnapshots(t, pool); !reflect.DeepEqual(before, after) {
		t.Fatal("waiting outbox rows changed")
	}
	recoverySetMode(t, pool, "normal")
	events, err := db.ClaimOutboxByLedgerMode(t.Context(), pool, "TEST ONLY normal", 1, time.Minute, recoveryAllowedTypes, recoveryFilteredType, recoveryWithdrawalKinds)
	if err != nil || len(events) != 1 || events[0].ID != first || events[0].Attempt != 1 {
		t.Fatalf("first normal claim: events=%v err=%v", events, err)
	}
}

func TestRecoveryModeT3_MalformedPublicationPayloads(t *testing.T) {
	pool := dbtest.New(t)
	recoverySetMode(t, pool, "reconciling")
	for _, payload := range []map[string]any{{}, {"operation_id": "not-a-uuid"}, {"operation_id": 123}, {"operation_id": uuid.NewString()}} {
		recoveryEvent(t, pool, "node.publication_changed", payload, time.Now().Add(-time.Minute))
	}
	before := recoverySnapshots(t, pool)
	if events := recoveryClaim(t, pool); len(events) != 0 {
		t.Fatalf("malformed publication events claimed: %v", events)
	}
	if after := recoverySnapshots(t, pool); !reflect.DeepEqual(before, after) {
		t.Fatal("malformed publication events changed")
	}
}

func TestRecoveryModeT4_NoHeadOfLineBlocking(t *testing.T) {
	pool := dbtest.New(t)
	recoverySetMode(t, pool, "reconciling")
	at := time.Now().Add(-time.Hour)
	for i := 0; i < 50; i++ {
		recoveryEvent(t, pool, "devices.discover", map[string]any{"index": i}, at.Add(time.Duration(i)*time.Second))
	}
	want := recoveryEvent(t, pool, "members.reconcile", map[string]any{}, at.Add(time.Minute))
	var claimed uuid.UUID
	d := recoveryDispatcher(t, pool, Registry{"members.reconcile": func(_ context.Context, event db.ClaimedEvent) (string, error) {
		claimed = event.ID
		return "TEST ONLY receipt", nil
	}})
	if worked, err := d.Once(t.Context()); err != nil || !worked || claimed != want {
		t.Fatalf("allowed event behind waiting rows: worked=%v claimed=%s err=%v", worked, claimed, err)
	}
}

func TestRecoveryModeT6_MemberSchedule(t *testing.T) {
	for _, mode := range []string{"reconciling", "normal", "missing"} {
		t.Run(mode, func(t *testing.T) {
			pool := dbtest.New(t)
			if mode == "missing" {
				recoveryRemoveLedgerState(t, pool)
			} else {
				recoverySetMode(t, pool, mode)
			}
			s := &membersruntime.Services{HTTP: httpadapter.Services{Members: &memberscore.Service{Pool: pool}}, ReconcileInterval: time.Hour, DeviceInterval: time.Hour}
			if err := memberScheduleCycle(t.Context(), s); err != nil {
				t.Fatal(err)
			}
			for _, eventType := range []string{"billing.reconcile", "members.reconcile", "devices.discover"} {
				var count int
				if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type=$1`, eventType).Scan(&count); err != nil {
					t.Fatal(err)
				}
				want := 1
				if mode != "normal" && eventType == "devices.discover" {
					want = 0
				}
				if count != want {
					t.Errorf("%s: got %d rows, want %d", eventType, count, want)
				}
			}
		})
	}
}
