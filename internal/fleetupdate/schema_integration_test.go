//go:build integration

package fleetupdate

import (
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAT24_UnboundSupersessionAndCancellation(t *testing.T) {
	l := newLab(t)
	i := l.node("arm64", "disabled", "enrolling", false)
	old := l.batch(l.old)
	newer := l.batch(l.new)
	if got := l.get(old.BatchId); got.Counts.Superseded != 1 || got.Targets[0].CurrentJob != nil {
		t.Fatal(got)
	}
	if l.scalar(`SELECT count(*) FROM update_jobs`) != 0 {
		t.Fatal("unbound jobs")
	}
	target := l.get(newer.BatchId).Targets[0].TargetId
	_, err := l.s.Retry(l.ctx, releasetest.Command(), newer.BatchId, api.UpdateBatchRetryRequest{Reason: "TEST ONLY unbound", TargetIds: &[]uuid.UUID{target}})
	if !errors.Is(err, apierr.New(apierr.StateConflict, "")) {
		t.Fatal(err)
	}
	i = l.enroll(i, "arm64")
	w := l.intent(i)
	if *w.Intent.BatchId != newer.BatchId || w.Intent.Attempt != 1 || l.scalar(`SELECT count(*) FROM update_jobs`) != 1 {
		t.Fatal(w)
	}
	if got := l.get(old.BatchId); got.Counts.Superseded != 1 {
		t.Fatal(got)
	}
	for _, lifecycle := range []string{"revoked", "retired"} {
		t.Run(lifecycle, func(t *testing.T) {
			pending := l.node("amd64", "disabled", "enrolling", false)
			b := l.batch(l.new)
			if _, err := l.s.pool.Exec(l.ctx, `UPDATE nodes SET lifecycle_state=$2,resource_version=resource_version+1 WHERE id=$1`, pending.NodeID, lifecycle); err != nil {
				t.Fatal(err)
			}
			got := l.get(b.BatchId)
			for _, target := range got.Targets {
				if target.NodeId == pending.NodeID && (string(target.State) != "cancelled_"+lifecycle || target.CurrentJob != nil || target.Attempts != 0) {
					t.Fatal(target)
				}
			}
			if l.scalar(`SELECT count(*) FROM update_targets WHERE batch_id=$1 AND node_id=$2 AND target_state=$3`, b.BatchId, pending.NodeID, "cancelled_"+lifecycle) != 1 {
				t.Fatal("missing target cancellation")
			}
		})
	}
}

func TestAT24_FirstSyncSnapshotGuards(t *testing.T) {
	l := newLab(t)
	a := l.node("amd64", "unpublished", "registered", true)
	b := l.node("arm64", "unpublished", "registered", true)
	for _, tc := range []struct {
		name, scope, actor, reason string
		selected                   Identity
		count                      int
		insert                     bool
	}{
		{"wrong member", "first_sync", firstSyncActor, "first_sync", b, 1, true},
		{"missing member", "first_sync", firstSyncActor, "first_sync", a, 1, false},
		{"wrong actor", "first_sync", uuid.NewString(), "first_sync", a, 1, true},
		{"wrong reason", "first_sync", firstSyncActor, "admin", a, 1, true},
		{"wrong denominator", "first_sync", firstSyncActor, "first_sync", a, 2, true},
		{"all_nodes cannot claim exception", "all_nodes", firstSyncActor, "first_sync", a, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
				op, batch := uuid.New(), uuid.New()
				if _, err := tx.Exec(l.ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id) VALUES($1,'TEST ONLY','test','update_batch',$2)`, op, batch); err != nil {
					return err
				}
				var systemNode *uuid.UUID
				if tc.scope == "first_sync" {
					systemNode = &a.NodeID
				}
				if _, err := tx.Exec(l.ctx, `INSERT INTO update_batches(id,operation_id,target_release_id,scope,created_by,target_count,generation,reason,first_sync_node_id) VALUES($1,$2,$3,$4,$5,$6,1,$7,$8)`, batch, op, l.new, tc.scope, tc.actor, tc.count, tc.reason, systemNode); err != nil {
					return err
				}
				if tc.insert {
					_, err := tx.Exec(l.ctx, `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,execution_incarnation_id,target_generation) VALUES($1,$2,$3,$3,1)`, batch, tc.selected.NodeID, tc.selected.IncarnationID)
					return err
				}
				return nil
			})
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23514" {
				t.Fatalf("expected snapshot constraint: %v", err)
			}
		})
	}
	if l.scalar(`SELECT count(*) FROM update_batches`) != 0 {
		t.Fatal("invalid snapshot committed")
	}
}

func TestUPD02_JobReferencesAndForwardPhases(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "published", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	for _, q := range []string{
		`UPDATE fleet_node_state SET current_job_id=gen_random_uuid()`,
		`INSERT INTO update_results(result_id,job_id,node_id,incarnation_id,payload_hash,payload,ack) SELECT gen_random_uuid(),gen_random_uuid(),node_id,execution_incarnation_id,decode(repeat('00',32),'hex'),'{}','{}' FROM update_jobs LIMIT 1`,
		`INSERT INTO update_retry_jobs(operation_id,job_id) SELECT operation_id,gen_random_uuid() FROM update_batches LIMIT 1`,
	} {
		_, err := l.s.pool.Exec(l.ctx, q)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "23503" {
			t.Fatalf("expected reference constraint: %v", err)
		}
	}
	l.report(i, w, "applying")
	ack := l.report(i, w, "already_current")
	if !ack.AcceptedAsHistory || l.get(b.BatchId).Counts.Applying != 1 {
		t.Fatal("already_current rewrote applying")
	}
	l.report(i, w, "succeeded")
	ack = l.report(i, w, "failed")
	if !ack.AcceptedAsHistory || l.get(b.BatchId).Counts.Succeeded != 1 {
		t.Fatal("terminal evidence changed")
	}
}
