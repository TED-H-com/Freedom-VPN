//go:build integration

package db_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func reviewTx(t *testing.T, p *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := p.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx
}

func reviewReject(t *testing.T, p *pgxpool.Pool, q string, args ...any) {
	t.Helper()
	_, err := p.Exec(context.Background(), q, args...)
	sqlError(t, err, "23514", "")
}

func reviewCount(t *testing.T, p *pgxpool.Pool, q string, want int, args ...any) {
	t.Helper()
	var n int
	if err := p.QueryRow(context.Background(), q, args...).Scan(&n); err != nil || n != want {
		t.Fatalf("count=%d want=%d err=%v", n, want, err)
	}
}

func reviewBatch(t *testing.T, tx pgx.Tx, f fixture, count int, restore bool) uuid.UUID {
	t.Helper()
	op, batch := uuid.New(), uuid.New()
	exec(t, tx, `INSERT INTO operations(id,kind,principal,resource_type) VALUES($1,'update','test','node')`, op)
	if restore {
		exec(t, tx, `INSERT INTO update_batches(id,operation_id,target_release_id,scope,created_by,target_count,generation,snapshot_xid) VALUES($1,$2,$3,'all_nodes','test', $4,2,'1'::xid8)`, batch, op, f.release, count)
	} else {
		exec(t, tx, `INSERT INTO update_batches(id,operation_id,target_release_id,scope,created_by,target_count,generation) VALUES($1,$2,$3,'all_nodes','test',$4,2)`, batch, op, f.release, count)
	}
	return batch
}

func reviewRollup(t *testing.T, p *pgxpool.Pool, f fixture, generation int) {
	t.Helper()
	exec(t, p, `INSERT INTO usage_rollups(member_id,period,granularity,projection_generation,metering_version,upload_bytes,download_bytes,last_observed_at,aggregate_as_of,period_start,period_end) VALUES($1,'2026-01','month',$2,'test-meter',1,2,now(),now(),'2026-01-01Z','2026-02-01Z')`, f.member, generation)
}

// §13.1: even an enrolling target with no execution incarnation can be cancelled.
func TestReviewF01_CancelledTargetsAndJobs(t *testing.T) {
	for _, state := range []string{"cancelled_revoked", "cancelled_retired"} {
		t.Run(state, func(t *testing.T) {
			p := dbtest.New(t)
			f := seed(t, p)
			exec(t, p, `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, f.inc, f.target)
			exec(t, p, `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation) VALUES($1,1,$2,1)`, f.target, f.inc)
			exec(t, p, `UPDATE update_jobs SET state=$1`, state)
			node := uuid.New()
			exec(t, p, `INSERT INTO nodes(id) VALUES($1)`, node)
			tx := reviewTx(t, p)
			batch := reviewBatch(t, tx, f, 2, false)
			exec(t, tx, `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) SELECT $1,id,current_incarnation_id,2 FROM nodes`, batch)
			if err := tx.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}
			exec(t, p, `UPDATE update_targets SET target_state=$1 WHERE batch_id=$2 AND node_id=$3`, state, batch, node)
			reviewCount(t, p, `SELECT count(*) FROM update_targets WHERE batch_id=$1 AND node_id=$2 AND target_state=$3 AND execution_incarnation_id IS NULL AND incarnation_at_snapshot IS NULL`, 1, batch, node, state)
			reviewReject(t, p, `UPDATE update_targets SET target_state='active' WHERE batch_id=$1 AND node_id=$2`, batch, node)
			reviewReject(t, p, `UPDATE update_targets SET target_generation=3 WHERE batch_id=$1`, batch)
		})
	}
}

// §15.4, §16.4: projection updates remain allowed, deletion requires audited age checks.
func TestReviewF02_RollupRetentionGuards(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	reviewRollup(t, p, f, 1)
	exec(t, p, `UPDATE usage_rollups SET upload_bytes=upload_bytes+1`)
	reviewReject(t, p, `DELETE FROM usage_rollups`)
	reviewReject(t, p, `TRUNCATE usage_rollups`)
	_, err := p.Exec(context.Background(), `SELECT purge_retained('usage_rollups',now(),'too early')`)
	sqlError(t, err, "22023", "")
	exec(t, p, `UPDATE usage_rollups SET period_start=now()-interval '16 months',period_end=now()-interval '15 months'`)
	reviewCount(t, p, `SELECT purge_retained('usage_rollups',now()-interval '14 months','retention test')`, 1)
	reviewCount(t, p, `SELECT count(*) FROM audit_events WHERE action='retention.purge' AND resource='usage_rollups'`, 1)
	reviewCount(t, p, `SELECT count(*) FROM usage_rollups`, 0)
}

// §10.8, §15.4, §16.4: one complete, non-reconciling generation must cover all records.
func TestReviewF03_PurgeRequiresOneCompleteGeneration(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	tx := reviewTx(t, p)
	event, a, b := uuid.New(), uuid.New(), uuid.New()
	exec(t, tx, `INSERT INTO usage_events(id,node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,$3,2,1,decode(repeat('01',32),'hex'),2)`, event, f.node, f.inc)
	exec(t, tx, `INSERT INTO usage_event_payloads(event_id,payload,created_at) VALUES($1,'\x01',now()-interval '40 days')`, event)
	for i, id := range []uuid.UUID{a, b} {
		exec(t, tx, `INSERT INTO usage_samples(id,event_id,record_index,member_id,device_id,binding_revision,meter_epoch_id,interval_start,interval_end,upload_bytes,download_bytes,metering_version,monotonic_elapsed_ns,clock_quality) VALUES($1,$2,$3,$4,$5,1,gen_random_uuid(),'2026-01-01Z','2026-01-02Z',1,2,'test-meter',1,'trusted')`, id, event, i, f.member, f.device)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, g := range []int{1, 2} {
		reviewRollup(t, p, f, g)
	}
	receipt := `INSERT INTO projection_receipts(sample_id,projection_name,generation,period) VALUES($1,'monthly',$2,'2026-01')`
	exec(t, p, receipt, a, 1)
	exec(t, p, receipt, b, 2)
	purge := `SELECT purge_retained('usage_event_payloads',now()-interval '36 days','test projection coverage')`
	reviewCount(t, p, purge, 0) // Both samples exist, but in disjoint generations.
	exec(t, p, receipt, a, 2)
	exec(t, p, `UPDATE usage_rollups SET reconciliation_in_progress=true WHERE projection_generation=2`)
	reviewCount(t, p, purge, 0)
	exec(t, p, `UPDATE usage_rollups SET reconciliation_in_progress=false,gap=true WHERE projection_generation=2`)
	reviewCount(t, p, purge, 0)
	exec(t, p, `UPDATE usage_rollups SET gap=false WHERE projection_generation=2`)
	reviewCount(t, p, purge, 1)
	reviewCount(t, p, `SELECT count(*) FROM usage_events WHERE id=$1`, 1, event)
}

// §10.6, §17.3: durable records and exact watermarks, including epoch reset.
func TestReviewF04_RecoverableCheckpoint(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `INSERT INTO usage_checkpoints(node_id,incarnation_id,highest_contiguous_seq,max_seen_seq,ledger_epoch) VALUES($1,$2,1,1,gen_random_uuid())`, f.node, f.inc)
	reviewReject(t, p, `DELETE FROM usage_checkpoints`)
	reviewReject(t, p, `TRUNCATE usage_checkpoints`)
	tx := reviewTx(t, p)
	exec(t, tx, `INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,2,1,decode(repeat('01',32),'hex'),5)`, f.node, f.inc)
	exec(t, tx, `UPDATE usage_checkpoints SET highest_contiguous_seq=2,max_seen_seq=2`)
	sqlError(t, tx.Commit(context.Background()), "23514", "")
	reviewCount(t, p, `SELECT count(*) FROM usage_events WHERE event_seq=2`, 0)
	// Either raw contents or all samples suffice; zero-record events are complete.
	for _, seq := range []int{2, 3, 10} {
		exec(t, p, `WITH e AS (INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,$3,1,decode(repeat('02',32),'hex'),5) RETURNING id) INSERT INTO usage_event_payloads(event_id,payload) SELECT id,'\x01' FROM e`, f.node, f.inc, seq)
	}
	reviewReject(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=1,max_seen_seq=1,ledger_epoch=gen_random_uuid()`)
	reviewReject(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=1,max_seen_seq=10,ledger_epoch=gen_random_uuid()`)
	reviewReject(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=3,max_seen_seq=3,ledger_epoch=gen_random_uuid()`)
	exec(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=3,max_seen_seq=10,ledger_epoch=gen_random_uuid()`)
	reviewReject(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=2`)
}

// §9.2, §15.5.2–3: no recipe is needed for the database quota guard to serialize writers.
func TestReviewF05_DeviceQuotaAndSuspensionRace(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `UPDATE entitlements SET device_limit=1,valid_from=now()-interval '1 day',effective_service_end=now()+interval '1 day'`)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			<-start
			_, err := p.Exec(ctx, `INSERT INTO member_devices(member_id,deployment_id,peer_id,status) VALUES($1,'test',gen_random_uuid()::text,'authorized')`, f.member)
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	winners := 0
	for err := range errs {
		if err == nil {
			winners++
		} else {
			sqlError(t, err, "23514", "")
		}
	}
	if winners != 1 {
		t.Fatalf("quota winners=%d", winners)
	}
	exec(t, p, `UPDATE member_devices SET status='revoked',revoked_at=now(),resource_version=resource_version+1 WHERE status='authorized'`)
	// Reproduce the review's stale read: a writer holds member but has only read entitlement.
	tx := reviewTx(t, p)
	exec(t, tx, `SELECT id FROM members WHERE id=$1 FOR UPDATE`, f.member)
	exec(t, tx, `SELECT device_limit FROM entitlements WHERE member_id=$1`, f.member)
	exec(t, p, `UPDATE entitlements SET admin_suspended=true,device_limit=0,revision=revision+1 WHERE member_id=$1`, f.member)
	_, err := tx.Exec(ctx, `INSERT INTO member_devices(member_id,deployment_id,peer_id,status) VALUES($1,'test','stale','authorized')`, f.member)
	sqlError(t, err, "23514", "")
	_ = tx.Rollback(ctx)
	exec(t, p, `UPDATE entitlements SET admin_suspended=false,device_limit=2,revision=revision+1`)
	exec(t, p, `UPDATE members SET status='closed',closed_at=now(),resource_version=resource_version+1`)
	reviewReject(t, p, `UPDATE member_devices SET status='authorized',resource_version=resource_version+1 WHERE id=$1`, f.device)
}

// expireReplayTx expires the locked conflict between BeginIdempotent's two
// statements. This deterministic interleaving exercises the real database path.
type expireReplayTx struct {
	pgx.Tx
	expired bool
}

func (tx *expireReplayTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	if !tx.expired && strings.HasPrefix(q, "SELECT id,state,resource_type") {
		tx.expired = true
		_, err := tx.Tx.Exec(ctx, `UPDATE idempotency_requests SET created_at=clock_timestamp()-interval '2 days',expiry=clock_timestamp()-interval '1 day'`)
		if err != nil {
			return errorRow{err}
		}
	}
	return tx.Tx.QueryRow(ctx, q, args...)
}

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

func TestReviewF06_IdempotencyReferenceAndExpiry(t *testing.T) {
	p := dbtest.New(t)
	ctx := context.Background()
	scope := db.IdempotencyScope{Principal: "test", Method: "POST", Route: "/test", KeyHash: bytes.Repeat([]byte{1}, 32), RequestHash: bytes.Repeat([]byte{2}, 32)}
	var record db.IdempotencyRecord
	op, resource := uuid.New(), uuid.New()
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		r, err := db.BeginIdempotent(ctx, tx, scope, 0)
		if err != nil {
			return err
		}
		record = r
		exec(t, tx, `INSERT INTO operations(id,kind,principal,resource_type) VALUES($1,'test','test','node')`, op)
		_, err = tx.Exec(ctx, `UPDATE idempotency_requests SET operation_id=$1,response_status=202,resource_type='node',resource_id=$2 WHERE id=$3`, op, resource, r.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	valid := db.IdempotencyResult{ResourceType: "node", ResourceID: &resource, OperationID: &op, ResponseStatus: 202}
	for _, field := range []string{"operation", "resource", "type", "status", "missing"} {
		t.Run(field, func(t *testing.T) {
			changed := valid
			id := uuid.New()
			switch field {
			case "operation":
				exec(t, p, `INSERT INTO operations(id,kind,principal,resource_type) VALUES($1,'other','test','node')`, id)
				changed.OperationID = &id
			case "resource":
				changed.ResourceID = &id
			case "type":
				changed.ResourceType = "member"
			case "status":
				changed.ResponseStatus = 200
			case "missing":
				changed.OperationID = nil
			}
			err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { return db.CompleteIdempotent(ctx, tx, record.ID, changed) })
			if !errors.Is(err, db.ErrIdempotencyConflict) {
				t.Fatalf("changed %s: %v", field, err)
			}
		})
	}
	for range 2 {
		if err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { return db.CompleteIdempotent(ctx, tx, record.ID, valid) }); err != nil {
			t.Fatal(err)
		}
	}
	orphan := scope
	orphan.KeyHash = bytes.Repeat([]byte{3}, 32)
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { _, err := db.BeginIdempotent(ctx, tx, orphan, 0); return err })
	sqlError(t, err, "23514", "")
	reviewCount(t, p, `SELECT count(*) FROM idempotency_requests`, 1)
	changed := scope
	changed.RequestHash = bytes.Repeat([]byte{4}, 32)
	err = db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		r, err := db.BeginIdempotent(ctx, &expireReplayTx{Tx: tx}, changed, 0)
		if err != nil {
			return err
		}
		if !r.New || r.ID == record.ID {
			return errors.New("returned expired conflict")
		}
		return db.CompleteIdempotent(ctx, tx, r.ID, valid)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReviewF07_SnapshotMembershipAndRestore(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	enrolling, revoked, retired := uuid.New(), uuid.New(), uuid.New()
	exec(t, p, `INSERT INTO nodes(id,lifecycle_state) VALUES($1,'enrolling'),($2,'revoked'),($3,'retired')`, enrolling, revoked, retired)
	for _, tc := range []struct {
		name string
		node uuid.UUID
		inc  any
	}{
		{"revoked", revoked, nil}, {"retired", retired, nil}, {"wrong incarnation", f.node, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := reviewTx(t, p)
			batch := reviewBatch(t, tx, f, 1, false)
			_, err := tx.Exec(context.Background(), `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) VALUES($1,$2,$3,2)`, batch, tc.node, tc.inc)
			sqlError(t, err, "23514", "")
		})
	}
	t.Run("missing eligible node", func(t *testing.T) {
		tx := reviewTx(t, p)
		batch := reviewBatch(t, tx, f, 1, false)
		exec(t, tx, `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) VALUES($1,$2,$3,2)`, batch, f.node, f.inc)
		sqlError(t, tx.Commit(context.Background()), "23514", "")
	})
	tx := reviewTx(t, p)
	batch := reviewBatch(t, tx, f, 2, false)
	exec(t, tx, `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) SELECT $1,id,current_incarnation_id,2 FROM nodes WHERE lifecycle_state NOT IN ('revoked','retired')`, batch)
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	reviewReject(t, p, `INSERT INTO update_targets(batch_id,node_id,target_generation) VALUES($1,$2,2)`, batch, enrolling)
	t.Run("historical xid sealed", func(t *testing.T) {
		tx := reviewTx(t, p)
		batch := reviewBatch(t, tx, f, 1, true)
		_, err := tx.Exec(context.Background(), `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) VALUES($1,$2,$3,2)`, batch, f.node, f.inc)
		sqlError(t, err, "23514", "")
	})
	// A dump contains historical membership; current lifecycle must not reinterpret it.
	exec(t, p, `UPDATE nodes SET lifecycle_state='revoked',resource_version=resource_version+1 WHERE id=$1`, f.node)
	tx = reviewTx(t, p)
	exec(t, tx, `SET LOCAL fvpn.restore='on'`)
	restored := reviewBatch(t, tx, f, 1, true)
	exec(t, tx, `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) VALUES($1,$2,$3,2)`, restored, f.node, f.inc)
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	reviewCount(t, p, `SELECT count(*) FROM update_batches WHERE id=$1 AND snapshot_xid='1'::xid8`, 1, restored)
	reviewReject(t, p, `INSERT INTO update_targets(batch_id,node_id,target_generation) VALUES($1,$2,2)`, restored, enrolling)
	t.Run("restore requires owner", func(t *testing.T) {
		var super bool
		if err := p.QueryRow(context.Background(), `SELECT rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&super); err != nil {
			t.Fatal(err)
		}
		if !super {
			t.Skip("role impersonation requires superuser test admin")
		}
		exec(t, p, `GRANT INSERT ON update_targets TO pg_read_all_data`)
		tx := reviewTx(t, p)
		exec(t, tx, `SET LOCAL ROLE pg_read_all_data`)
		exec(t, tx, `SET LOCAL fvpn.restore='on'`)
		_, err := tx.Exec(context.Background(), `INSERT INTO update_targets(batch_id,node_id,target_generation) VALUES($1,$2,2)`, restored, enrolling)
		sqlError(t, err, "23514", "")
	})
}

func TestReviewF08_OwnerMigrationBackfill(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	q := `UPDATE plan_revisions SET exit_group_ids=ARRAY[$1::uuid] WHERE id=$2`
	reviewReject(t, p, q, f.group, f.revision)
	tx := reviewTx(t, p)
	exec(t, tx, `SET LOCAL fvpn.migration='on'`)
	exec(t, tx, q, f.group, f.revision)
	// immutable_fields has the same exemption; other constraints remain active.
	exec(t, tx, `UPDATE subscriptions SET external_customer_id='backfilled',resource_version=resource_version+1 WHERE id=$1`, f.sub)
	exec(t, tx, `INSERT INTO audit_events(actor,actor_type,action,resource,result) VALUES('test','migration','schema.backfill','plan_revisions','succeeded')`)
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	reviewCount(t, p, `SELECT count(*) FROM plan_revisions WHERE id=$1 AND exit_group_ids=ARRAY[$2::uuid]`, 1, f.revision, f.group)
	reviewReject(t, p, q, f.group, f.revision)
	reviewReject(t, p, `UPDATE subscriptions SET external_customer_id='other',resource_version=resource_version+1 WHERE id=$1`, f.sub)
	var super bool
	if err := p.QueryRow(context.Background(), `SELECT rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&super); err != nil {
		t.Fatal(err)
	}
	if super {
		exec(t, p, `GRANT UPDATE ON plan_revisions TO pg_read_all_data`)
		tx = reviewTx(t, p)
		exec(t, tx, `SET LOCAL ROLE pg_read_all_data`)
		exec(t, tx, `SET LOCAL fvpn.migration='on'`)
		_, err := tx.Exec(context.Background(), q, f.group, f.revision)
		sqlError(t, err, "23514", "")
	}
}

func TestReviewF09_JobProgressAndFrozenResult(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, f.inc, f.target)
	exec(t, p, `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation) VALUES($1,1,$2,1)`, f.target, f.inc)
	reviewReject(t, p, `DELETE FROM update_jobs`)
	reviewReject(t, p, `TRUNCATE update_jobs`)
	for _, state := range []string{"waiting_online", "queued", "downloading", "verifying", "applying", "checking"} {
		exec(t, p, `UPDATE update_jobs SET state=$1`, state)
	}
	for _, state := range []string{"queued", "downloading", "superseded"} {
		reviewReject(t, p, `UPDATE update_jobs SET state=$1`, state)
	}
	result := uuid.New()
	exec(t, p, `UPDATE update_jobs SET state='succeeded',actual_release_id=$1,result_id=$2,result_hash=decode(repeat('04',32),'hex'),completed_at=now(),observed_evidence='{"healthy":true}'`, f.release, result)
	for _, change := range []string{
		`state='queued',actual_release_id=NULL,completed_at=NULL,observed_evidence='{}'`,
		`state='failed'`, `actual_release_id=NULL`, `last_good_release_id=$1`, `completed_at=completed_at+interval '1 second'`,
		`error_code='changed'`, `observed_evidence='{"healthy":false}'`, `result_hash=decode(repeat('05',32),'hex')`,
	} {
		if strings.Contains(change, "$1") {
			reviewReject(t, p, `UPDATE update_jobs SET `+change, f.release)
		} else {
			reviewReject(t, p, `UPDATE update_jobs SET `+change)
		}
	}
	// An identical retry is harmless, and read-back observation time can advance.
	exec(t, p, `UPDATE update_jobs SET result_id=$1,last_observed_at=now()`, result)
	exec(t, p, `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation) VALUES($1,2,$2,1)`, f.target, f.inc)
	exec(t, p, `UPDATE update_jobs SET state='failed' WHERE attempt=2`)
	reviewReject(t, p, `UPDATE update_jobs SET observed_evidence='{"changed":true}' WHERE attempt=2`)
	reviewReject(t, p, `UPDATE update_jobs SET state='queued' WHERE attempt=2`)
}

func TestReviewF10_OperationEvidenceAndLifecycle(t *testing.T) {
	p := dbtest.New(t)
	const evidence = `{"required":{"required":true,"status":"succeeded","observed_ref":"req"},"optional":{"required":false,"status":"succeeded","observed_ref":"opt"},"skip":{"required":false,"status":"not_required","reason":"not applicable"}}`
	const refs = `{"req":{"ok":true},"opt":{"ok":true}}`
	reviewReject(t, p, `INSERT INTO operations(kind,principal,resource_type,status,completed_at,steps,observed_refs) VALUES('test','test','node','succeeded',now(),$1,$2)`, evidence, refs)
	exec(t, p, `INSERT INTO operations(kind,principal,resource_type,status) VALUES('test','test','node','running')`)
	reviewReject(t, p, `UPDATE operations SET status='succeeded',completed_at=now(),steps='{"opt":{"required":false,"status":"not_required","reason":"skip"}}',observed_refs='{"dummy":true}'`)
	reviewReject(t, p, `UPDATE operations SET status='succeeded',completed_at=now(),steps=$1,observed_refs='{"req":{"ok":true}}'`, evidence)
	exec(t, p, `UPDATE operations SET status='succeeded',completed_at=now(),steps=$1,observed_refs=$2`, evidence, refs)
	for _, change := range []string{`principal='other'`, `kind='other'`, `steps='{}'`, `observed_refs='{"req":{"changed":true},"opt":{"ok":true}}'`, `completed_at=completed_at+interval '1 second'`, `status='running'`} {
		reviewReject(t, p, `UPDATE operations SET `+change)
	}
	reviewReject(t, p, `DELETE FROM operations`)
	reviewReject(t, p, `TRUNCATE operations CASCADE`)
	exec(t, p, `INSERT INTO operations(kind,principal,resource_type) VALUES('retry','test','node')`)
	for _, status := range []string{"running", "failed", "running", "partially_succeeded", "running", "superseded"} {
		exec(t, p, `UPDATE operations SET status=$1 WHERE kind='retry'`, status)
	}
	reviewReject(t, p, `UPDATE operations SET principal='changed' WHERE kind='retry'`)
}

const reviewLeaseSQL = `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash) VALUES($1,$2,$3,1,'2026-01-01Z','2026-01-02Z','test',decode(repeat('01',32),'hex'))`
const reviewLeaseMemberSQL = `INSERT INTO serving_lease_members(lease_id,member_id,entitlement_revision,valid_from,expires_at) VALUES($1,$2,1,'2026-01-01Z','2026-01-01T12:00:00Z')`

func TestReviewF11_LeaseEligibilityAndSeal(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	ctx := context.Background()
	for _, state := range []string{"enrolling", "revoked", "retired"} {
		t.Run(state, func(t *testing.T) {
			tx := reviewTx(t, p)
			exec(t, tx, `UPDATE nodes SET lifecycle_state=$1,resource_version=resource_version+1 WHERE id=$2`, state, f.node)
			_, err := tx.Exec(ctx, reviewLeaseSQL, uuid.New(), f.node, f.inc)
			sqlError(t, err, "23514", "")
		})
	}
	t.Run("revoked incarnation", func(t *testing.T) {
		tx := reviewTx(t, p)
		exec(t, tx, `UPDATE node_incarnations SET revoked_at=now() WHERE id=$1`, f.inc)
		_, err := tx.Exec(ctx, reviewLeaseSQL, uuid.New(), f.node, f.inc)
		sqlError(t, err, "23514", "")
	})
	for _, state := range []string{"closed", "suspended"} {
		t.Run(state, func(t *testing.T) {
			tx := reviewTx(t, p)
			lease := uuid.New()
			exec(t, tx, reviewLeaseSQL, lease, f.node, f.inc)
			exec(t, tx, `UPDATE members SET status=$1,resource_version=resource_version+1 WHERE id=$2`, state, f.member)
			_, err := tx.Exec(ctx, reviewLeaseMemberSQL, lease, f.member)
			sqlError(t, err, "23514", "")
		})
	}
	reviewReject(t, p, reviewLeaseMemberSQL, f.lease, f.member)
	t.Run("restore requires owner", func(t *testing.T) {
		var super bool
		if err := p.QueryRow(ctx, `SELECT rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&super); err != nil {
			t.Fatal(err)
		}
		if !super {
			t.Skip("role impersonation requires superuser test admin")
		}
		exec(t, p, `GRANT INSERT ON serving_lease_members TO pg_read_all_data`)
		exec(t, p, `GRANT UPDATE ON serving_leases TO pg_read_all_data`)
		tx := reviewTx(t, p)
		exec(t, tx, `SET LOCAL ROLE pg_read_all_data`)
		exec(t, tx, `SET LOCAL fvpn.restore='on'`)
		_, err := tx.Exec(ctx, reviewLeaseMemberSQL, f.lease, f.member)
		sqlError(t, err, "23514", "")
	})
	tx := reviewTx(t, p)
	lease := uuid.New()
	exec(t, tx, reviewLeaseSQL, lease, f.node, f.inc)
	exec(t, tx, reviewLeaseMemberSQL, lease, f.member)
	// All eligibility rows stay locked until signing/membership insertion commits.
	for _, q := range []string{
		`UPDATE nodes SET lifecycle_state='revoked',resource_version=resource_version+1`,
		`UPDATE node_incarnations SET revoked_at=now()`,
		`UPDATE members SET status='closed',resource_version=resource_version+1`,
		`UPDATE entitlements SET admin_suspended=true`,
	} {
		t.Run("locks "+q, func(t *testing.T) {
			other := reviewTx(t, p)
			exec(t, other, `SET LOCAL lock_timeout='100ms'`)
			_, err := other.Exec(ctx, q)
			sqlError(t, err, "55P03", "")
		})
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	// Restore can replay a historical signed lease even after revocation/closure.
	exec(t, p, `UPDATE nodes SET lifecycle_state='revoked',resource_version=resource_version+1`)
	exec(t, p, `UPDATE node_incarnations SET revoked_at=now()`)
	exec(t, p, `UPDATE members SET status='closed',resource_version=resource_version+1`)
	tx = reviewTx(t, p)
	exec(t, tx, `SET LOCAL fvpn.restore='on'`)
	lease = uuid.New()
	exec(t, tx, reviewLeaseSQL, lease, f.node, f.inc)
	exec(t, tx, reviewLeaseMemberSQL, lease, f.member)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	reviewReject(t, p, reviewLeaseSQL, uuid.New(), f.node, f.inc)
}

func TestReviewF12_FiniteBindingsPreserveUsage(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	for _, bounds := range [][2]string{{"2026-01-01Z", "infinity"}, {"-infinity", "2026-01-01Z"}, {"-infinity", "infinity"}} {
		reviewReject(t, p, `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,'other','100.64.0.2',2,$3,$4,'test')`, f.device, f.member, bounds[0], bounds[1])
		reviewReject(t, p, `INSERT INTO assignments(member_id,group_id,valid_from,valid_until) VALUES($1,$2,$3,$4)`, f.member, f.group, bounds[0], bounds[1])
	}
	reviewReject(t, p, `UPDATE device_bindings SET valid_until='2026-01-01T00:01:00Z' WHERE id=$1`, f.binding)
	exec(t, p, `UPDATE device_bindings SET valid_until='2026-01-02Z' WHERE id=$1`, f.binding)
	exec(t, p, `INSERT INTO assignments(member_id,group_id,valid_from,valid_until) VALUES($1,$2,'2026-01-01Z','2026-01-02Z')`, f.member, f.group)
	// A concurrent sample writer holds the binding while an endpoint update waits.
	tx := reviewTx(t, p)
	exec(t, tx, `WITH e AS (INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($3,$4,2,1,decode(repeat('02',32),'hex'),1) RETURNING id) INSERT INTO usage_samples(event_id,record_index,member_id,device_id,binding_revision,meter_epoch_id,interval_start,interval_end,upload_bytes,download_bytes,metering_version,monotonic_elapsed_ns,clock_quality) SELECT id,0,$1,$2,1,gen_random_uuid(),'2026-01-01Z','2026-01-02Z',1,1,'test',1,'trusted' FROM e`, f.member, f.device, f.node, f.inc)
	other := reviewTx(t, p)
	exec(t, other, `SET LOCAL lock_timeout='100ms'`)
	_, err := other.Exec(context.Background(), `UPDATE device_bindings SET valid_until='2026-01-01T12:00:00Z' WHERE id=$1`, f.binding)
	sqlError(t, err, "55P03", "")
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReviewF13_HistoricalIdentityAndRevocation(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	other := uuid.New()
	exec(t, p, `INSERT INTO members(id) VALUES($1)`, other)
	_, err := p.Exec(context.Background(), `INSERT INTO billing_records(member_id,subscription_id,provider,record_type,external_reference,amount_minor,currency,service_start,service_end,source_event_id) VALUES($1,$2,'test','invoice','test',1,'USD','2026-01-01Z','2026-02-01Z',$3)`, other, f.sub, f.inbox)
	sqlError(t, err, "23503", "billing_records_subscription_member_fk")
	reviewReject(t, p, `UPDATE subscriptions SET member_id=$1,resource_version=resource_version+1 WHERE id=$2`, other, f.sub)
	exec(t, p, `INSERT INTO node_certificates(incarnation_id,issuer,serial,fingerprint,public_certificate,not_before,not_after) VALUES($1,'test','test',decode(repeat('01',32),'hex'),'\x01','2026-01-01Z','2026-02-01Z')`, f.inc)
	exec(t, p, `INSERT INTO entitlement_grants(member_id,source,valid_from,valid_until,reason,created_by) VALUES($1,'manual','2026-01-01Z','2026-02-01Z','test reason','test actor')`, f.member)
	exec(t, p, `INSERT INTO node_observations(node_id,incarnation_id,report_id,payload_hash,observed_at,applied_generation,schema_version,checks,control_status,data_health,safety_status) VALUES($1,$2,gen_random_uuid(),decode(repeat('01',32),'hex'),now(),1,1,'{}','online','healthy','passed')`, f.node, f.inc)
	for _, tc := range []struct{ table, change string }{
		{"subscriptions", `provider='other',resource_version=resource_version+1`},
		{"node_certificates", `public_certificate='\x02'`},
		{"entitlement_grants", `valid_until='2026-03-01Z'`},
		{"inbox_events", `encrypted_payload='\x02'`},
		{"node_observations", `checks='{"rewritten":true}'`},
	} {
		reviewReject(t, p, `UPDATE `+tc.table+` SET `+tc.change)
	}
	for _, table := range []string{"node_incarnations", "node_certificates", "entitlement_grants", "member_devices"} {
		suffix := ""
		if table == "entitlement_grants" {
			suffix = ",revocation_reason='test revoke'"
		}
		if table == "member_devices" {
			suffix = ",status='revoked',resource_version=resource_version+1"
		}
		exec(t, p, `UPDATE `+table+` SET revoked_at=now()`+suffix)
		if table == "entitlement_grants" {
			suffix = ""
		}
		reviewReject(t, p, `UPDATE `+table+` SET revoked_at=NULL`+suffix)
		reviewReject(t, p, `UPDATE `+table+` SET revoked_at=revoked_at+interval '1 second'`+suffix)
	}
	for _, tc := range []struct{ table, field string }{{"members", "closed_at"}, {"member_devices", "closed_at"}, {"nodes", "retired_at"}} {
		exec(t, p, fmt.Sprintf(`UPDATE %s SET %s=now(),resource_version=resource_version+1`, tc.table, tc.field))
		reviewReject(t, p, fmt.Sprintf(`UPDATE %s SET %s=NULL,resource_version=resource_version+1`, tc.table, tc.field))
	}
	for _, table := range []string{"retention_config", "subscriptions", "entitlement_grants", "node_certificates", "node_observations", "inbox_events", "exits", "device_bindings", "node_incarnations"} {
		reviewReject(t, p, `DELETE FROM `+table)
		reviewReject(t, p, `TRUNCATE `+table+` CASCADE`)
	}
	exec(t, p, `UPDATE inbox_events SET processing_status='succeeded',attempt=attempt+1`)
	exec(t, p, `UPDATE subscriptions SET provider_status='cancelled',resource_version=resource_version+1`)
}

func TestReviewF14_StrictResourceVersions(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	for _, table := range []string{"members", "plans", "subscriptions", "member_devices", "nodes", "exits", "policy_definitions", "fleet_config"} {
		t.Run(table, func(t *testing.T) {
			reviewReject(t, p, `UPDATE `+table+` SET resource_version=resource_version`)
			reviewReject(t, p, `UPDATE `+table+` SET resource_version=resource_version-1`)
			exec(t, p, `UPDATE `+table+` SET resource_version=resource_version+1`)
		})
	}
	exec(t, p, `INSERT INTO netbird_user_bindings(member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,'test','test',$2,'test',10,now())`, f.member, f.identity)
	reviewReject(t, p, `UPDATE netbird_user_bindings SET revision=1`)
	exec(t, p, `UPDATE netbird_user_bindings SET revision=11`)
}

func TestReviewF15_PrimitiveValidationAndClaimOrder(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	ctx := context.Background()
	evt := db.OutboxEvent{AggregateID: f.member, AggregateType: "member", AggregateVersion: "1", EventType: "changed", SchemaVersion: 1, Payload: []byte(`{}`)}
	for _, field := range []string{"aggregate", "type", "event"} {
		bad := evt
		switch field {
		case "aggregate":
			bad.AggregateID = uuid.Nil
		case "type":
			bad.AggregateType = " \t"
		case "event":
			bad.EventType = ""
		}
		err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { return db.EnqueueOutbox(ctx, tx, bad) })
		if err == nil {
			t.Errorf("accepted outbox %s", field)
		}
	}
	audit := db.AuditEvent{Actor: "test", ActorType: "admin", Action: "test", Resource: "member", Result: "accepted", SchemaVersion: 1, Before: []byte(`{}`), After: []byte(`{}`)}
	for _, field := range []string{"actor", "type", "action", "resource", "result"} {
		bad := audit
		switch field {
		case "actor":
			bad.Actor = ""
		case "type":
			bad.ActorType = "  "
		case "action":
			bad.Action = ""
		case "resource":
			bad.Resource = "\t"
		case "result":
			bad.Result = ""
		}
		err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error { return db.AppendAudit(ctx, tx, bad) })
		if err == nil {
			t.Errorf("accepted audit %s", field)
		}
	}
	reviewCount(t, p, `SELECT count(*) FROM outbox_events`, 0)
	reviewCount(t, p, `SELECT count(*) FROM audit_events`, 0)
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		if err := db.AppendAudit(ctx, tx, audit); err != nil {
			return err
		}
		// Reverse physical insertion order, with ties in available_at.
		for i := 19; i >= 0; i-- {
			e := evt
			e.ID = uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1))
			e.AvailableAt = time.Date(2020, 1, 1, 0, i/3, 0, 0, time.UTC)
			if err := db.EnqueueOutbox(ctx, tx, e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	events, err := db.ClaimOutbox(ctx, p, "test", 20, time.Minute)
	if err != nil || len(events) != 20 {
		t.Fatalf("claims=%d err=%v", len(events), err)
	}
	for i, e := range events {
		want := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1))
		if e.ID != want {
			t.Errorf("claim %d is %s want %s", i, e.ID, want)
		}
	}
	for _, blank := range []string{"", "   ", "\t\n"} {
		for _, field := range []string{"reason", "creator", "revocation"} {
			reason, creator, revoke := "test reason", "test actor", "test revoke"
			switch field {
			case "reason":
				reason = blank
			case "creator":
				creator = blank
			case "revocation":
				revoke = blank
			}
			reviewReject(t, p, `INSERT INTO entitlement_grants(member_id,source,valid_from,valid_until,reason,created_by,revoked_at,revocation_reason) VALUES($1,'manual','2026-01-01Z','2026-01-02Z',$2,$3,now(),$4)`, f.member, reason, creator, revoke)
		}
		reviewReject(t, p, `UPDATE subscriptions SET grace_from='2026-01-01Z',grace_until='2026-01-02Z',grace_reason=$1,resource_version=resource_version+1`, blank)
	}
	exec(t, p, `UPDATE subscriptions SET grace_from='2026-01-01Z',grace_until='2026-01-02Z',grace_reason='approved TEST ONLY grace',resource_version=resource_version+1`)
}
