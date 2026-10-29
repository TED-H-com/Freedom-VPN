//go:build integration

package audit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestSEC02_AuditRedactionFiltersAndStableCursor(t *testing.T) {
	pool := dbtest.New(t)
	actor, resource, request := uuid.New(), uuid.New(), uuid.New()
	e := db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		for i := 0; i < 5; i++ {
			e := Append(t.Context(), tx, db.AuditEvent{Actor: actor.String(), ActorType: "admin", Action: "node.update", Resource: "node:" + resource.String(), Result: "succeeded", RequestID: &request}, nil, map[string]any{"state": map[string]any{"status": "pending", "setup_key": "TEST-ONLY-SETUP", "nested": map[string]any{"authorization": "TEST-ONLY-AUTH"}}, "token": "TEST-ONLY-TOKEN"}, "state", "token")
			if e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	s := Store{Pool: pool}
	limit := 2
	typ := api.ResourceType("node")
	from, to := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	params := api.ListAuditEventsParams{Limit: &limit, ResourceType: &typ, ResourceId: &resource, ActorId: &actor, RequestId: &request, OccurredFrom: &from, OccurredTo: &to}
	seen := map[uuid.UUID]bool{}
	for {
		page, e := s.List(t.Context(), params)
		if e != nil {
			t.Fatal(e)
		}
		for _, event := range page.Items {
			if seen[event.AuditEventId] {
				t.Fatal("duplicate cursor row")
			}
			seen[event.AuditEventId] = true
		}
		raw, _ := json.Marshal(page)
		if strings.Contains(string(raw), "TEST-ONLY-") {
			t.Fatal("secret leaked", string(raw))
		}
		if !page.NextCursor.IsSpecified() || page.NextCursor.IsNull() {
			break
		}
		cursor := nullableValue(page.NextCursor)
		params.Cursor = &cursor
	}
	if len(seen) != 5 {
		t.Fatal("missing rows", len(seen))
	}
	params.ActorId = &resource
	if _, e := s.List(t.Context(), params); e == nil {
		t.Fatal("cursor reused with changed filters")
	}
	params = api.ListAuditEventsParams{RequestId: &resource}
	page, e := s.List(t.Context(), params)
	if e != nil || len(page.Items) != 0 {
		t.Fatal("request filter failed", e)
	}
	if _, e := pool.Exec(t.Context(), `UPDATE audit_events SET result='succeeded'`); e == nil {
		t.Fatal("audit history mutable")
	}
}

func TestSEC02_MalformedAuditRowsDoNotPoisonPages(t *testing.T) {
	pool := dbtest.New(t)
	var logs bytes.Buffer
	s := Store{Pool: pool, Logger: slog.New(slog.NewJSONHandler(&logs, nil))}
	base := db.AuditEvent{Actor: uuid.NewString(), ActorType: "admin", Action: "node.update", Resource: "node:" + uuid.NewString(), Result: "succeeded", SchemaVersion: 1, Before: []byte(`{}`), After: []byte(`{}`)}
	// IDs are tied on occurred_at within this transaction. Malformed rows
	// surround the valid rows so boundaries and empty pages are exercised.
	const marker = "TEST-ONLY-MALFORMED-SECRET"
	events := []db.AuditEvent{base, base, base, base, base, base, base, base}
	events[0].ActorType = marker
	events[1].Resource = marker
	events[3].Resource = "node:" + marker
	events[4].Actor = marker
	events[5].Action = marker
	events[6].Result = marker
	if err := db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		for i := range events {
			events[i].ID = uuid.Must(uuid.NewV7())
			if err := db.AppendAudit(t.Context(), tx, events[i]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, timestamp := range []string{"infinity", "-infinity", "10000-01-01T00:00:00Z"} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO audit_events(actor,actor_type,action,resource,result,occurred_at) VALUES('system','system','audit.test',$1,'succeeded',$2::timestamptz)`, base.Resource, timestamp); err != nil {
			t.Fatal(err)
		}
	}
	limit := 1
	params := api.ListAuditEventsParams{Limit: &limit}
	seen := map[uuid.UUID]bool{}
	emptyPages := 0
	pages := 0
	for {
		page, err := s.List(t.Context(), params)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		if pages > len(events)+3 {
			t.Fatal("pagination did not progress")
		}
		if len(page.Items) == 0 {
			emptyPages++
		}
		for _, event := range page.Items {
			if seen[event.AuditEventId] {
				t.Fatal("repeated valid row")
			}
			seen[event.AuditEventId] = true
		}
		if !page.NextCursor.IsSpecified() || page.NextCursor.IsNull() {
			break
		}
		if params.Cursor != nil && *params.Cursor == nullableValue(page.NextCursor) {
			t.Fatal("repeated cursor")
		}
		cursor := nullableValue(page.NextCursor)
		params.Cursor = &cursor
	}
	if pages != 11 || emptyPages != 9 || len(seen) != 2 || !seen[events[2].ID] || !seen[events[7].ID] || s.AnomalyCount() != 9 {
		t.Fatal("lost rows, invented identity, or incorrect anomaly count", pages, emptyPages, seen, s.AnomalyCount())
	}
	if strings.Contains(logs.String(), marker) || strings.Count(logs.String(), `"msg":"audit row omitted"`) != 9 || !strings.Contains(logs.String(), `"audit_anomalies_total":9`) {
		t.Fatal("anomaly logs missing or unsafe", logs.String())
	}
}

func TestPurgeRetained_AuditRowDecodesResourceRefAndNullSystemActor(t *testing.T) {
	pool := dbtest.New(t)
	if _, err := pool.Exec(t.Context(), `SELECT public.purge_retained('audit_events', '2000-01-01T00:00:00Z'::timestamptz, 'scheduled retention purge')`); err != nil {
		t.Fatalf("purge_retained: %v", err)
	}
	s := Store{Pool: pool}
	rt := api.ResourceType("fleet_config")
	rid := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	page, err := s.List(t.Context(), api.ListAuditEventsParams{
		ResourceType: &rt,
		ResourceId:   &rid,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 purge_retained audit event, got %d (anomalies=%d)", len(page.Items), s.AnomalyCount())
	}
	ev := page.Items[0]
	if ev.Action != "retention.purge" {
		t.Fatalf("expected action retention.purge, got %q", ev.Action)
	}
	if ev.Actor.Type != api.AuditEventActorTypeSystem || !ev.Actor.Id.IsSpecified() || !ev.Actor.Id.IsNull() {
		t.Fatalf("expected system actor with id=null, got %+v", ev.Actor)
	}
	if ev.Resource.Type != rt || ev.Resource.Id != rid {
		t.Fatalf("expected resource %s:%s, got %+v", rt, rid, ev.Resource)
	}
}
