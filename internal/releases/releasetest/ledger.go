package releasetest

import (
	"context"
	"encoding/json"
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
func AssertLedger(t testing.TB, pool *pgxpool.Pool, validator *releases.Validator) {
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
	for _, id := range ids {
		op, err := (operations.Store{Pool: pool}).Get(ctx, id, auth.Principal{Role: auth.Admin, ID: uuid.New()})
		if err != nil {
			t.Fatal(err)
		}
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
	for {
		page, err := store.List(ctx, api.ListAuditEventsParams{Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range page.Items {
			if event.RequestId.IsNull() || !event.Resource.Type.Valid() || !event.Result.Valid() {
				t.Fatal(event)
			}
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
	if store.AnomalyCount() != 0 || count != persisted || count == 0 {
		t.Fatalf("audit persisted=%d returned=%d anomalies=%d", persisted, count, store.AnomalyCount())
	}
}
