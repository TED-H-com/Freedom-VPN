//go:build integration

package audit

import (
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"testing"
)

func TestSEC02_TypedAuditReadableWithoutAnomalies(t *testing.T) {
	pool := dbtest.New(t)
	event, err := NewEvent(Entry{Actor: Actor{Type: api.AuditEventActorTypeSystem, ID: uuid.New()}, Action: "outbox.dead_letter", Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: uuid.New()}, Result: api.AuditEventResultFailed, After: map[string]any{"status": "failed", "token": "TEST ONLY"}, Fields: []string{"status", "token"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error { return AppendEvent(t.Context(), tx, event) }); err != nil {
		t.Fatal(err)
	}
	store := Store{Pool: pool}
	page, err := store.List(t.Context(), api.ListAuditEventsParams{})
	if err != nil || len(page.Items) != 1 || store.AnomalyCount() != 0 {
		t.Fatal(page, err)
	}
}
