//go:build integration

package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func TestSEC02_DeadLettersUseContractResourceTypes(t *testing.T) {
	pool := dbtest.New(t)
	metrics := prometheus.NewRegistry()
	var logs bytes.Buffer
	d, err := New(pool, nil, Config{Concurrency: 1, Attempts: 1, Lease: time.Minute, JobTimeout: time.Second, Poll: time.Second, Backoff: time.Second, MaxBackoff: time.Second}, metrics, slog.New(slog.NewJSONHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for aggregate, want := range map[string]api.ResourceType{"node-incarnation": "node", "device": "member_device", "billing_session": "checkout_session", "fleet": "fleet_config", "fleet_update": "update_batch", "member": "member"} {
		resource := uuid.New()
		if err = db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
			return db.EnqueueOutbox(t.Context(), tx, db.OutboxEvent{AggregateType: aggregate, AggregateID: resource, AggregateVersion: "1", EventType: "test.failure", SchemaVersion: 1, Payload: []byte(`{}`)})
		}); err != nil {
			t.Fatal(err)
		}
		if _, err = d.Once(t.Context()); err != nil {
			t.Fatal(err)
		}
		var got string
		if err = pool.QueryRow(t.Context(), `SELECT resource FROM audit_events WHERE resource=$1`, string(want)+":"+resource.String()).Scan(&got); err != nil {
			t.Fatal("missing mapped audit", aggregate, err)
		}
	}
	reader := audit.Store{Pool: pool}
	page, err := reader.List(t.Context(), api.ListAuditEventsParams{})
	if err != nil || len(page.Items) != 6 || reader.AnomalyCount() != 0 {
		t.Fatal("unreadable dead-letter history", page, err)
	}
}

func TestSEC02_UnmappedDeadLettersAreTerminalAndObservable(t *testing.T) {
	for _, aggregate := range []string{"billing_inbox", "TEST-ONLY-UNKNOWN"} {
		t.Run(aggregate, func(t *testing.T) {
			pool := dbtest.New(t)
			metrics := prometheus.NewRegistry()
			var logs bytes.Buffer
			var claim db.ClaimedEvent
			const secret = "TEST ONLY private provider payload"
			registry := Registry{"test.failure": func(_ context.Context, event db.ClaimedEvent) (string, error) {
				claim = event
				return "", errors.New(secret)
			}}
			d, err := New(pool, registry, Config{Concurrency: 1, Attempts: 1, Lease: time.Minute, JobTimeout: time.Second, Poll: time.Second, Backoff: time.Second, MaxBackoff: time.Second}, metrics, slog.New(slog.NewJSONHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			event := db.OutboxEvent{ID: uuid.New(), AggregateType: aggregate, AggregateID: uuid.New(), AggregateVersion: "1", EventType: "test.failure", SchemaVersion: 1, Payload: []byte(`{"provider_payload":"` + secret + `"}`)}
			if err = db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
				return db.EnqueueOutbox(t.Context(), tx, event)
			}); err != nil {
				t.Fatal(err)
			}
			if worked, err := d.Once(t.Context()); err != nil || !worked {
				t.Fatal("unmapped event did not terminate", worked, err)
			}
			if err = d.deadLetter(t.Context(), claim, "HANDLER_FAILED"); !errors.Is(err, db.ErrLeaseLost) {
				t.Fatal("stale claim repeated dead letter", err)
			}
			if worked, err := d.Once(t.Context()); err != nil || worked {
				t.Fatal("dead letter reclaimed", worked, err)
			}
			var terminal bool
			if err = pool.QueryRow(t.Context(), `SELECT dead_lettered_at IS NOT NULL AND delivered_at IS NULL AND available_at='infinity' AND lease_owner IS NULL AND lease_token IS NULL AND lease_until IS NULL AND attempt=1 AND last_error_code='HANDLER_FAILED' FROM outbox_events WHERE id=$1`, event.ID).Scan(&terminal); err != nil || !terminal {
				t.Fatal("missing terminal state", terminal, err)
			}
			reader := audit.Store{Pool: pool}
			page, err := reader.List(t.Context(), api.ListAuditEventsParams{})
			if err != nil || len(page.Items) != 1 || reader.AnomalyCount() != 0 {
				t.Fatal("unreadable or duplicated fallback audit", page, err)
			}
			entry := page.Items[0]
			var fleetID uuid.UUID
			if err = pool.QueryRow(t.Context(), `SELECT id FROM fleet_config`).Scan(&fleetID); err != nil {
				t.Fatal(err)
			}
			if entry.Resource.Type != api.ResourceTypeFleetConfig || entry.Resource.Id != fleetID || entry.Action != "outbox.dead_letter" || entry.Result != api.AuditEventResultFailed || entry.Actor.Type != api.AuditEventActorTypeSystem {
				t.Fatal("incorrect fallback audit resource or outcome", entry)
			}
			after, err := entry.Changes.After.Get()
			if err != nil || len(after) != 3 || after["dead_lettered"] != true || after["aggregate_type"] != aggregate || after["aggregate_id"] != event.AggregateID.String() {
				t.Fatal("fallback diff lost aggregate identity or contains extra fields", after)
			}
			raw, err := json.Marshal(entry)
			if err != nil || strings.Contains(string(raw), secret) || strings.Contains(logs.String(), secret) || strings.Contains(logs.String(), aggregate) || strings.Contains(logs.String(), event.AggregateID.String()) {
				t.Fatal("dead-letter audit or logs exposed private context", err)
			}
			anomalies := 0
			for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
				var record map[string]any
				if err = json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["error_code"] == "AUDIT_RESOURCE_TYPE_UNKNOWN" {
					anomalies++
					if record["event_id"] != event.ID.String() || len(record) != 5 {
						t.Fatal("mapping log contains unexpected context", record)
					}
				}
			}
			if anomalies != 1 {
				t.Fatal("mapping anomaly must log once", anomalies)
			}
			families, err := metrics.Gather()
			if err != nil {
				t.Fatal(err)
			}
			counts := map[string]float64{}
			for _, family := range families {
				if family.GetName() == "fvpn_worker_events_total" {
					for _, m := range family.Metric {
						if len(m.Label) != 1 || m.Label[0].GetName() != "result" {
							t.Fatal("unexpected metric labels", m.Label)
						}
						counts[m.Label[0].GetValue()] = m.GetCounter().GetValue()
					}
				}
			}
			if len(counts) != 2 || counts["dead_letter_mapping_error"] != 1 || counts["dead_letter"] != 1 {
				t.Fatal("mapping anomaly and terminal event must count once with bounded labels", counts)
			}
		})
	}
}
