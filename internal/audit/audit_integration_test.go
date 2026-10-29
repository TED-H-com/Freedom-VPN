//go:build integration

package audit

import (
	"encoding/json"
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
		if page.NextCursor == nil {
			break
		}
		params.Cursor = page.NextCursor
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
