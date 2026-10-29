//go:build integration

package worker

import (
	"bytes"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"strings"
	"testing"
	"time"
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
	err = d.deadLetter(t.Context(), db.ClaimedEvent{ID: uuid.New(), AggregateType: "TEST-ONLY-UNKNOWN-SECRET", AggregateID: uuid.New()}, "HANDLER_FAILED")
	if err == nil || errors.Is(err, db.ErrLeaseLost) {
		t.Fatal("unknown type did not fail mapping", err)
	}
	if !strings.Contains(logs.String(), "AUDIT_RESOURCE_TYPE_UNKNOWN") || strings.Contains(logs.String(), "TEST-ONLY-UNKNOWN-SECRET") {
		t.Fatal("unsafe or missing anomaly log")
	}
	families, err := metrics.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, family := range families {
		if family.GetName() == "fvpn_worker_events_total" {
			for _, m := range family.Metric {
				for _, label := range m.Label {
					if label.GetValue() == "dead_letter_mapping_error" && m.GetCounter().GetValue() == 1 {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("missing mapping error metric")
	}
}
