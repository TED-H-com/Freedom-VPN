//go:build integration

package worker

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func TestAPI01_UnknownMemberEventsDeadLetterObservably(t *testing.T) {
	for _, eventType := range []string{"billing.session_requested", "test.unknown"} {
		t.Run(eventType, func(t *testing.T) {
			pool := dbtest.New(t)
			ctx := t.Context()
			id, aggregate := uuid.New(), uuid.New()
			// TEST ONLY payload must not reach logs or metric labels. No member services
			// are supplied: an unknown event must never invoke a domain handler.
			payload := []byte(`{"note":"TEST ONLY private event payload"}`)
			err := db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
				return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{ID: id, AggregateID: aggregate, AggregateType: "billing_session", AggregateVersion: "1", EventType: eventType, SchemaVersion: 1, Payload: payload})
			})
			if err != nil {
				t.Fatal(err)
			}
			metrics := prometheus.NewRegistry()
			var logs bytes.Buffer
			dispatcher, err := New(pool, MemberJobs(), Config{Concurrency: 1, Attempts: 2, Lease: time.Minute, JobTimeout: time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Second}, metrics, slog.New(slog.NewJSONHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			for attempt := 1; attempt <= 2; attempt++ {
				// Any panic fails this regression, including attempts to call a nil handler.
				if worked, err := dispatcher.Once(ctx); err != nil || !worked {
					t.Fatalf("attempt %d: worked=%v error=%v", attempt, worked, err)
				}
				if attempt == 1 {
					var retried bool
					err = pool.QueryRow(ctx, `SELECT attempt=1 AND delivered_at IS NULL AND dead_lettered_at IS NULL AND lease_owner IS NULL AND lease_token IS NULL AND lease_until IS NULL FROM outbox_events WHERE id=$1`, id).Scan(&retried)
					if err != nil || !retried {
						t.Fatal("unknown event was not retried", err)
					}
					if _, err = pool.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp() WHERE id=$1`, id); err != nil {
						t.Fatal(err)
					}
				}
			}
			var dead bool
			var code string
			err = pool.QueryRow(ctx, `SELECT attempt=2 AND dead_lettered_at IS NOT NULL AND delivered_at IS NULL AND delivery_receipt IS NULL AND available_at='infinity'::timestamptz AND lease_owner IS NULL AND lease_token IS NULL AND lease_until IS NULL,last_error_code FROM outbox_events WHERE id=$1`, id).Scan(&dead, &code)
			if err != nil || !dead || code != "HANDLER_MISSING" {
				t.Fatalf("unknown event disposition: dead=%v code=%q error=%v", dead, code, err)
			}
			if worked, err := dispatcher.Once(ctx); err != nil || worked {
				t.Fatal("dead letter was reclaimed", worked, err)
			}
			var audits int
			err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='outbox.dead_letter' AND resource=$1`, string(api.ResourceTypeCheckoutSession)+":"+aggregate.String()).Scan(&audits)
			if err != nil || audits != 1 {
				t.Fatal("dead-letter audit missing or duplicated", audits, err)
			}
			assertUnknownEventMetrics(t, metrics)

			decoder := json.NewDecoder(&logs)
			var record map[string]any
			if err := decoder.Decode(&record); err != nil {
				t.Fatal("missing structured dead-letter log", err)
			}
			if record["msg"] != "outbox dead letter" || record["event_id"] != id.String() || record["error_code"] != "HANDLER_MISSING" {
				t.Fatal("incorrect dead-letter log", record)
			}
			// The event ID is the sole correlation identity. Existing safe severity,
			// timestamp, attempt and error-code metadata remain; no type/payload/owner.
			for key := range record {
				switch key {
				case "time", "level", "msg", "event_id", "attempt", "error_code":
				default:
					t.Errorf("unexpected dead-letter log field %q", key)
				}
			}
			encoded, _ := json.Marshal(record)
			for _, private := range []string{eventType, aggregate.String(), "TEST ONLY private event payload"} {
				if bytes.Contains(encoded, []byte(private)) {
					t.Error("dead-letter log leaked event contents")
				}
			}
			if err := decoder.Decode(&record); err != io.EOF {
				t.Fatal("unexpected extra event log", err)
			}
		})
	}
}

func assertUnknownEventMetrics(t *testing.T, registry *prometheus.Registry) {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]float64{}
	for _, family := range families {
		if family.GetName() != "fvpn_worker_dead_letters" && family.GetName() != "fvpn_worker_events_total" {
			continue
		}
		for _, metric := range family.Metric {
			key := family.GetName()
			for _, label := range metric.Label {
				if label.GetName() != "result" {
					t.Errorf("unexpected event metric label %q", label.GetName())
				}
				key += "/" + label.GetValue()
			}
			if strings.HasPrefix(key, "fvpn_worker_events_total/") {
				values[key] = metric.GetCounter().GetValue()
			} else {
				values[key] = metric.GetGauge().GetValue()
			}
		}
	}
	for _, key := range []string{"fvpn_worker_dead_letters", "fvpn_worker_events_total/dead_letter", "fvpn_worker_events_total/retry"} {
		if values[key] != 1 {
			t.Errorf("%s=%v, want 1", key, values[key])
		}
	}
	if values["fvpn_worker_events_total/delivered"] != 0 {
		t.Fatal("unknown event counted as delivered")
	}
}
