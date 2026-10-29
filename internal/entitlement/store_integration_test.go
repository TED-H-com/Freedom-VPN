//go:build integration

package entitlement

import (
	"context"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		if _, e := tx.Exec(ctx, `UPDATE members SET resource_version=resource_version+1,admin_suspended=true WHERE id=$1`, id); e != nil {
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

func TestENT01_PendingProjectionCannotReceiveServingLease(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	member, node, inc, lease := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO members(id) VALUES($1)`, member)
	exec(`INSERT INTO nodes(id) VALUES($1)`, node)
	exec(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,arch) VALUES($1,$2,decode(repeat('00',32),'hex'),'test','test-peer','ubuntu','amd64')`, inc, node)
	exec(`UPDATE nodes SET lifecycle_state='registered',current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, node, inc)
	engine, _ := NewStore(TestConfig())
	if err := db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error { _, err := engine.Recompute(ctx, tx, member); return err }); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"pending_payment", "active"} {
		t.Run(status, func(t *testing.T) {
			err := db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
				if _, err := tx.Exec(ctx, `UPDATE members SET status=$2,resource_version=resource_version+1 WHERE id=$1`, member, status); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash) VALUES($1,$2,$3,1,now(),now()+interval '1 hour','test-key',decode(repeat('00',32),'hex'))`, lease, node, inc); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, `INSERT INTO serving_lease_members(lease_id,member_id,entitlement_revision,valid_from,expires_at) VALUES($1,$2,1,clock_timestamp(),clock_timestamp()+interval '10 minutes')`, lease, member)
				return err
			})
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatal("absent service interval received a lease or failed for an unrelated reason", err)
			}
		})
	}
	// An inconsistent coarse status must not turn null interval comparisons
	// into SQL UNKNOWN and accidentally authorize a peer (F-05).
	exec(`UPDATE members SET status='active',resource_version=resource_version+1 WHERE id=$1`, member)
	_, err := pool.Exec(ctx, `INSERT INTO member_devices(member_id,deployment_id,peer_id,status) VALUES($1,'test','test-peer','authorized')`, member)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatal("null interval authorized a device", err)
	}
}
