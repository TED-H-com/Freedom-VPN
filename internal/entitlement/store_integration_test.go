//go:build integration

package entitlement

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"testing"
)

func TestAT04_RevisionAndOutboxAtomic(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	id := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO members(id) VALUES($1)`, id)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := NewStore(TestConfig())
	for n := range 2 {
		err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
			p, e := store.Recompute(ctx, tx, id)
			if e == nil && (p.Revision != 1 || p.Changed != (n == 0)) {
				t.Fatalf("bad projection %+v", p)
			}
			return e
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `UPDATE members SET admin_suspended=true WHERE id=$1`, id); e != nil {
			return e
		}
		p, e := store.Recompute(ctx, tx, id)
		if e == nil && (p.Revision != 2 || p.State != "suspended") {
			t.Fatal(p)
		}
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	var events, history int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM outbox_events WHERE event_type='member.entitlement_changed'),(SELECT count(*) FROM entitlement_history)`).Scan(&events, &history); err != nil || events != 2 || history != 2 {
		t.Fatalf("events=%d history=%d err=%v", events, history, err)
	}
}
