//go:build integration

package db_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func Test15_1_Uint64Bounds(t *testing.T) {
	p := dbtest.New(t)
	for _, value := range []string{"-1", "18446744073709551615", "18446744073709551616", "9007199254740993"} {
		t.Run(value, func(t *testing.T) {
			var got string
			err := p.QueryRow(context.Background(), `INSERT INTO members(resource_version) VALUES($1::numeric) RETURNING resource_version::text`, value).Scan(&got)
			if value == "-1" || value == "18446744073709551616" {
				sqlError(t, err, "23514", "")
			} else if err != nil || got != value {
				t.Fatalf("roundtrip %q: %v", got, err)
			}
		})
	}
	// Catalog-wide: every NUMERIC(20,0) column must carry the explicit uint64 check.
	rows, err := p.Query(context.Background(), `SELECT table_name,column_name FROM information_schema.columns WHERE table_schema='public' AND numeric_precision=20 AND numeric_scale=0`)
	if err != nil {
		t.Fatal(err)
	}
	type column struct{ table, name string }
	var cols []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.table, &c.name); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	for _, c := range cols {
		var ok bool
		err := p.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=$1::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%'||$2||' >= (0)%' AND pg_get_constraintdef(oid) LIKE '%'||$2||' <= %18446744073709551615%')`, c.table, c.name).Scan(&ok)
		if err != nil || !ok {
			t.Errorf("missing uint64 check %s.%s: %v", c.table, c.name, err)
		}
	}
}

func Test15_1_IntervalsAndFiniteExpiry(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	cases := []struct {
		name, q string
		args    []any
	}{
		{"grant", `INSERT INTO entitlement_grants(member_id,source,valid_from,valid_until,reason,created_by) VALUES($1,'test','2026-01-01Z',$2,'test','test')`, []any{f.member}},
		{"certificate", `INSERT INTO node_certificates(incarnation_id,issuer,serial,fingerprint,public_certificate,not_before,not_after) VALUES($1,'test',gen_random_uuid()::text,decode(repeat('01',32),'hex'),'\x01','2026-01-01Z',$2)`, []any{f.inc}},
		{"lease", `INSERT INTO serving_leases(node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash) VALUES($1,$2,1,'2026-01-01Z',$3,'test',decode(repeat('01',32),'hex'))`, []any{f.node, f.inc}},
		{"enrollment", `INSERT INTO node_enrollments(node_id,token_hash,created_at,expiry) VALUES($1,decode(repeat('02',32),'hex'),'2026-01-01Z',$2)`, []any{f.node}},
		{"idempotency", `INSERT INTO idempotency_requests(principal,method,route,key_hash,request_hash,resource_id,response_status,created_at,expiry) VALUES('test','POST','/test',decode(repeat('02',32),'hex'),decode(repeat('03',32),'hex'),gen_random_uuid(),201,'2026-01-01Z',$1)`, nil},
	}
	for _, c := range cases {
		for _, end := range []string{"2025-12-31Z", "2026-01-01Z", "infinity", "-infinity"} {
			t.Run(c.name+end, func(t *testing.T) {
				args := append(append([]any{}, c.args...), end)
				_, err := p.Exec(context.Background(), c.q, args...)
				sqlError(t, err, "23514", "")
			})
		}
	}
	for _, c := range cases {
		t.Run(c.name+"valid", func(t *testing.T) {
			args := append(append([]any{}, c.args...), "2026-01-02Z")
			exec(t, p, c.q, args...)
		})
	}
	_, err := p.Exec(context.Background(), `INSERT INTO entitlement_grants(member_id,source,valid_from,valid_until,reason,created_by) VALUES($1,'test',now(),NULL,'test','test')`, f.member)
	sqlError(t, err, "23502", "")
	for _, q := range []string{
		`UPDATE device_bindings SET valid_until=valid_from`,
		`UPDATE entitlements SET effective_service_end=valid_from`,
		`UPDATE subscriptions SET paid_from=now(),paid_through=now()`,
		`UPDATE subscriptions SET grace_from=now(),grace_until='infinity',grace_reason='test'`,
	} {
		_, err := p.Exec(context.Background(), q)
		sqlError(t, err, "23514", "")
	}
}

func Test15_2_HalfOpenBindingsAndAssignments(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	q := `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,$3,'100.64.0.1',$4,$5,$6,'peer')`
	_, err := p.Exec(context.Background(), q, f.device, f.member, "overlay", "2", "2026-12-01Z", "2027-02-01Z")
	sqlError(t, err, "23P01", "device_bindings_no_overlap")
	exec(t, p, q, f.device, f.member, "overlay", "2", "2027-01-01Z", "2027-02-01Z")
	exec(t, p, q, f.device, f.member, "different-overlay", "3", "2026-01-01Z", "2027-01-01Z")
	for _, kind := range []string{"member_id", "plan_revision_id"} {
		id := f.member
		if kind == "plan_revision_id" {
			id = f.revision
		}
		q := fmt.Sprintf(`INSERT INTO assignments(%s,group_id,valid_from,valid_until) VALUES($1,$2,$3,$4)`, kind)
		exec(t, p, q, id, f.group, "2026-01-01Z", "2026-02-01Z")
		_, err := p.Exec(context.Background(), q, id, f.group, "2026-01-01Z", "2026-02-01Z")
		sqlError(t, err, "23P01", "")
		exec(t, p, q, id, f.group, "2026-02-01Z", "2026-03-01Z")
	}
}

func Test15_1_AppendOnlyAndAuditedPurge(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `INSERT INTO audit_events(actor,actor_type,action,resource,result,occurred_at) VALUES('test','admin','test','test','succeeded',now()-interval '200 days')`)
	exec(t, p, `INSERT INTO billing_records(member_id,provider,record_type,external_reference,amount_minor,currency,service_start,service_end,source_event_id) VALUES($1,'test','invoice','test',1,'USD','2026-01-01Z','2026-02-01Z',$2)`, f.member, f.inbox)
	exec(t, p, `INSERT INTO usage_adjustments(member_id,period,original_event_id,upload_bytes,download_bytes,reason,actor) VALUES($1,'2026-01',$2,-1,0,'correction','test')`, f.member, f.event)
	exec(t, p, `INSERT INTO projection_receipts(sample_id,projection_name,generation,period) VALUES($1,'monthly',1,'2026-01')`, f.sample)
	exec(t, p, `INSERT INTO usage_rollups(member_id,period,granularity,projection_generation,metering_version,upload_bytes,download_bytes,last_observed_at,aggregate_as_of,period_start,period_end) VALUES($1,'2026-01','month',1,'test-meter',1,2,now(),now(),'2026-01-01Z','2026-02-01Z')`, f.member)
	exec(t, p, `INSERT INTO usage_event_payloads(event_id,payload,created_at) VALUES($1,'\x01',now()-interval '40 days')`, f.event)
	for _, table := range []string{"audit_events", "billing_records", "usage_adjustments", "usage_events", "usage_samples", "projection_receipts", "usage_event_payloads"} {
		for _, q := range []string{"UPDATE " + table + " SET id=id", "DELETE FROM " + table} {
			_, err := p.Exec(context.Background(), q)
			sqlError(t, err, "23514", "")
		}
		// TRUNCATE can fail first on a foreign key; otherwise the trigger rejects it.
		tx, err := p.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(context.Background(), "TRUNCATE "+table+" CASCADE")
		sqlError(t, err, "23514", "")
		_ = tx.Rollback(context.Background())
	}
	var n int
	for _, target := range []string{"audit_events", "usage_event_payloads"} {
		days := 181
		if target == "usage_event_payloads" {
			days = 36
		}
		err := p.QueryRow(context.Background(), `SELECT purge_retained($1,now()-$2::int*interval '1 day','test retention')`, target, days).Scan(&n)
		if err != nil || n != 1 {
			t.Fatalf("purge %s count=%d err=%v", target, n, err)
		}
	}
	if err := p.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action='retention.purge'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("purge audit count=%d err=%v", n, err)
	}
	if err := p.QueryRow(context.Background(), `SELECT count(*) FROM usage_events`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("receipt count=%d err=%v", n, err)
	}
	for _, q := range []string{`SELECT purge_retained('audit_events',now(),'too early')`, `SELECT purge_retained('billing_records',now()-interval '10 years','not approved')`, `SELECT purge_retained('usage_events',now()-interval '10 years','not approved')`} {
		_, err := p.Exec(context.Background(), q)
		sqlError(t, err, "22023", "")
	}
	// PUBLIC cannot call the SECURITY DEFINER function (no global role creation).
	err := p.QueryRow(context.Background(), `SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl WHERE p.oid='purge_retained(text,timestamptz,text)'::regprocedure AND acl.grantee=0 AND acl.privilege_type='EXECUTE'`).Scan(&n)
	if err != nil || n != 0 {
		t.Fatalf("PUBLIC purge grants=%d err=%v", n, err)
	}
	// Transaction-local purge flag is reset, even for the owner.
	_, err = p.Exec(context.Background(), `DELETE FROM audit_events`)
	sqlError(t, err, "23514", "")
}

func Test13_8_ImmutabilityAndIncarnationBinding(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `INSERT INTO artifacts(release_id,platform,arch,digest,size_bytes,installed_size_bytes,storage_ref) VALUES($1,'ubuntu','arm64',decode(repeat('01',32),'hex'),1,1,'https://releases.example.invalid/test')`, f.release)
	for _, table := range []string{"plan_revisions", "policy_revisions", "releases", "artifacts", "update_batches"} {
		for _, q := range []string{"UPDATE " + table + " SET id=id", "DELETE FROM " + table} {
			_, err := p.Exec(context.Background(), q)
			sqlError(t, err, "23514", "")
		}
	}
	_, err := p.Exec(context.Background(), `UPDATE update_targets SET target_generation=2`)
	sqlError(t, err, "23514", "")
	exec(t, p, `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, f.inc, f.target)
	exec(t, p, `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, f.inc, f.target)
	_, err = p.Exec(context.Background(), `UPDATE update_targets SET execution_incarnation_id=NULL`)
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), `UPDATE update_targets SET execution_incarnation_id=gen_random_uuid()`)
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), `INSERT INTO update_targets(batch_id,node_id,target_generation) VALUES($1,$2,1)`, f.batch, f.node)
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation) VALUES($1,1,$2,2)`, f.target, f.inc)
	sqlError(t, err, "23503", "")
	exec(t, p, `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation) VALUES($1,1,$2,1)`, f.target, f.inc)
	_, err = p.Exec(context.Background(), `UPDATE update_jobs SET attempt=2`)
	sqlError(t, err, "23514", "")
	for _, table := range []string{"members", "nodes", "member_devices", "fleet_config"} {
		_, err := p.Exec(context.Background(), "DELETE FROM "+table)
		sqlError(t, err, "23514", "")
	}
	_, err = p.Exec(context.Background(), `INSERT INTO fleet_config(id) VALUES(gen_random_uuid())`)
	sqlError(t, err, "23514", "")
}

func TestAT15_ContiguousAckAndEpoch(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	epoch := uuid.New()
	exec(t, p, `INSERT INTO usage_checkpoints(node_id,incarnation_id,highest_contiguous_seq,max_seen_seq,ledger_epoch) VALUES($1,$2,1,1,$3)`, f.node, f.inc, epoch)
	q := `INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,$3,1,decode(repeat('01',32),'hex'),0)`
	exec(t, p, q, f.node, f.inc, "3")
	exec(t, p, `UPDATE usage_checkpoints SET max_seen_seq=3`)
	_, err := p.Exec(context.Background(), `UPDATE usage_checkpoints SET highest_contiguous_seq=3`)
	sqlError(t, err, "23514", "")
	exec(t, p, q, f.node, f.inc, "2")
	exec(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=3`)
	_, err = p.Exec(context.Background(), `UPDATE usage_checkpoints SET highest_contiguous_seq=1`)
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), `UPDATE usage_checkpoints SET highest_contiguous_seq=1,ledger_epoch=gen_random_uuid()`)
	sqlError(t, err, "23514", "")
	exec(t, p, `UPDATE usage_checkpoints SET highest_contiguous_seq=3,ledger_epoch=gen_random_uuid()`)
	_, err = p.Exec(context.Background(), `UPDATE usage_checkpoints SET max_seen_seq=4`)
	sqlError(t, err, "23514", "")
}

func Test16_3_MemberLeaseDeadline(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `UPDATE entitlements SET effective_service_end='2026-01-01T12:00:00Z'`)
	q := `INSERT INTO serving_lease_members(lease_id,member_id,entitlement_revision,valid_from,expires_at) VALUES($1,$2,1,'2026-01-01Z',$3)`
	for _, end := range []string{"2026-01-03Z", "2026-01-02Z", "2026-01-01T12:00:00Z"} {
		err := db.InTx(context.Background(), p, db.TxOptions{}, func(tx pgx.Tx) error {
			lease := uuid.New()
			if _, err := tx.Exec(context.Background(), `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash) VALUES($1,$2,$3,1,'2026-01-01Z','2026-01-02Z','test',decode(repeat('01',32),'hex'))`, lease, f.node, f.inc); err != nil {
				return err
			}
			if _, err := tx.Exec(context.Background(), q, lease, f.member, end); err != nil {
				return err
			}
			_, err := tx.Exec(context.Background(), q, lease, f.member, end)
			return err
		})
		if end == "2026-01-01T12:00:00Z" {
			sqlError(t, err, "23505", "serving_lease_members_member_key")
		} else {
			sqlError(t, err, "23514", "")
		}
	}
}

func Test15_5_SnapshotCountAtCommit(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	err := db.InTx(context.Background(), p, db.TxOptions{}, func(tx pgx.Tx) error {
		var op uuid.UUID
		if err := tx.QueryRow(context.Background(), `INSERT INTO operations(kind,principal,resource_type) VALUES('test','test','node') RETURNING id`).Scan(&op); err != nil {
			return err
		}
		_, err := tx.Exec(context.Background(), `INSERT INTO update_batches(operation_id,target_release_id,scope,created_by,target_count,generation) VALUES($1,$2,'all_nodes','test',1,2)`, op, f.release)
		return err
	})
	sqlError(t, err, "23514", "")
}

func Test15_1_MoneyAndJSON(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	for _, values := range []string{"-1,'USD'", "1,'usd'", "1,'US'"} {
		q := `INSERT INTO billing_records(member_id,provider,record_type,external_reference,amount_minor,currency,service_start,service_end,source_event_id) VALUES($1,'test','invoice','test',` + values + `,'2026-01-01Z','2026-02-01Z',$2)`
		_, err := p.Exec(context.Background(), q, f.member, f.inbox)
		sqlError(t, err, "23514", "")
	}
	for _, payload := range []string{`[]`, `null`, `"scalar"`} {
		_, err := p.Exec(context.Background(), `INSERT INTO plan_revisions(plan_id,revision,device_limit,schema_version,payload) VALUES($1,2,1,1,$2)`, f.plan, payload)
		sqlError(t, err, "23514", "")
	}
}

func Test15_4_OperationEvidenceAndTransitions(t *testing.T) {
	p := dbtest.New(t)
	exec(t, p, `INSERT INTO operations(kind,principal,resource_type) VALUES('publish','test','node')`)
	_, err := p.Exec(context.Background(), `UPDATE operations SET status='succeeded',completed_at=now(),observed_refs='{"route":"test"}'`)
	sqlError(t, err, "23514", "")
	exec(t, p, `UPDATE operations SET status='running'`)
	_, err = p.Exec(context.Background(), `UPDATE operations SET status='succeeded',completed_at=now(),observed_refs='{"route":"test"}'`)
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), `UPDATE operations SET status='succeeded',completed_at=now(),steps='{"route":{"required":true,"status":"succeeded","observed_ref":"missing"}}',observed_refs='{"route":"test"}'`)
	sqlError(t, err, "23514", "")
	exec(t, p, `UPDATE operations SET status='succeeded',completed_at=now(),steps='{"route":{"required":true,"status":"succeeded","observed_ref":"route"}}',observed_refs='{"route":"test"}'`)
	_, err = p.Exec(context.Background(), `UPDATE operations SET status='running'`)
	sqlError(t, err, "23514", "")
}

func Test16_3_ConfigurableLeaseMaximum(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	q := `INSERT INTO serving_leases(node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash) VALUES($1,$2,1,'2026-01-01Z','2026-01-03Z','test',decode(repeat('01',32),'hex'))`
	_, err := p.Exec(context.Background(), q, f.node, f.inc)
	sqlError(t, err, "23514", "")
	exec(t, p, `UPDATE fleet_config SET max_offline_lease_seconds=172800,resource_version=resource_version+1`)
	exec(t, p, q, f.node, f.inc)
}

func Test15_3_EnrollmentBindingOnce(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `INSERT INTO node_enrollments(node_id,token_hash,expiry) VALUES($1,decode(repeat('01',32),'hex'),now()+interval '1 hour')`, f.node)
	exec(t, p, `UPDATE node_enrollments SET public_key_fingerprint=decode(repeat('02',32),'hex'),state='consumed',consumed_at=now()`)
	_, err := p.Exec(context.Background(), `UPDATE node_enrollments SET public_key_fingerprint=decode(repeat('03',32),'hex')`)
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), `UPDATE node_enrollments SET expiry=expiry+interval '1 day'`)
	sqlError(t, err, "23514", "")
}

func Test16_4_PurgeSettingCannotGrantAuthority(t *testing.T) {
	p := dbtest.New(t)
	var super bool
	if err := p.QueryRow(context.Background(), `SELECT rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&super); err != nil {
		t.Fatal(err)
	}
	if !super {
		t.Skip("role impersonation test requires superuser test admin; no cluster roles are created")
	}
	exec(t, p, `INSERT INTO audit_events(actor,actor_type,action,resource,result,occurred_at) VALUES('test','admin','test','test','succeeded',now()-interval '200 days')`)
	// Built-in role; grants are on this isolated database's objects only.
	exec(t, p, `GRANT DELETE ON audit_events TO pg_read_all_data`)
	tx, err := p.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `SET LOCAL ROLE pg_read_all_data`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `SET LOCAL fvpn.retention_purge='on'`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(context.Background(), `DELETE FROM audit_events`)
	sqlError(t, err, "23514", "")
	_ = tx.Rollback(context.Background())
	tx, err = p.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `SET LOCAL ROLE pg_read_all_data`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(context.Background(), `SELECT purge_retained('audit_events',now()-interval '181 days','test')`)
	sqlError(t, err, "42501", "")
	_ = tx.Rollback(context.Background())
}

func TestMTR03_UsageEventKeyUnique(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	for _, hash := range []string{"06", "07"} {
		_, err := p.Exec(context.Background(), `INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,1,1,decode(repeat($3,32),'hex'),1)`, f.node, f.inc, hash)
		sqlError(t, err, "23505", "usage_events_stream_key")
	}
	var hash string
	if err := p.QueryRow(context.Background(), `SELECT encode(payload_hash,'hex') FROM usage_events WHERE id=$1`, f.event).Scan(&hash); err != nil || hash != strings.Repeat("06", 32) {
		t.Fatalf("original receipt changed: %s %v", hash, err)
	}
}

func Test15_4_ProjectionReceiptsAtomicIncrement(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	ctx := context.Background()
	exec(t, p, `INSERT INTO usage_rollups(member_id,period,granularity,projection_generation,metering_version,upload_bytes,download_bytes,last_observed_at,aggregate_as_of,period_start,period_end) VALUES($1,'2026-01','month',1,'test',0,0,now(),now(),'2026-01-01Z','2026-02-01Z')`, f.member)
	apply := func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO projection_receipts(sample_id,projection_name,generation,period) VALUES($1,'monthly',1,'2026-01') ON CONFLICT DO NOTHING`, f.sample)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE usage_rollups SET upload_bytes=upload_bytes+18446744073709551615,download_bytes=download_bytes+9007199254740993 WHERE member_id=$1 AND projection_generation=1`, f.member)
		return err
	}
	abort := errors.New("test rollback")
	if err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		if err := apply(tx); err != nil {
			return err
		}
		return abort
	}); !errors.Is(err, abort) {
		t.Fatal(err)
	}
	for range 2 {
		if err := db.InTx(ctx, p, db.TxOptions{}, apply); err != nil {
			t.Fatal(err)
		}
	}
	var total string
	if err := p.QueryRow(ctx, `SELECT total_bytes::text FROM usage_rollups`).Scan(&total); err != nil || total != "18455751272964292608" {
		t.Fatalf("total=%s err=%v", total, err)
	}
}

func TestMTR03_HistoricalBindingAndRecordCount(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	var event uuid.UUID
	if err := p.QueryRow(context.Background(), `WITH e AS (INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,2,1,decode(repeat('01',32),'hex'),1) RETURNING id) INSERT INTO usage_event_payloads(event_id,payload) SELECT id,'\x01' FROM e RETURNING event_id`, f.node, f.inc).Scan(&event); err != nil {
		t.Fatal(err)
	}
	q := `INSERT INTO usage_samples(event_id,record_index,member_id,device_id,binding_revision,meter_epoch_id,interval_start,interval_end,upload_bytes,download_bytes,metering_version,monotonic_elapsed_ns,clock_quality) VALUES($1,$2,$3,$4,1,gen_random_uuid(),$5,$6,0,0,'test',1,'trusted')`
	_, err := p.Exec(context.Background(), q, event, 0, f.member, f.device, "2025-12-31Z", "2026-01-02Z")
	sqlError(t, err, "23514", "")
	_, err = p.Exec(context.Background(), q, event, 1, f.member, f.device, "2026-01-01Z", "2026-01-02Z")
	sqlError(t, err, "23514", "")
	exec(t, p, q, event, 0, f.member, f.device, "2026-01-01Z", "2026-01-02Z")
}

func Test16_4_PayloadPurgeRequiresWholeProjection(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	ctx := context.Background()
	event, sample := uuid.New(), uuid.New()
	err := db.InTx(ctx, p, db.TxOptions{}, func(tx pgx.Tx) error {
		exec(t, tx, `INSERT INTO usage_events(id,node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,$3,2,1,decode(repeat('01',32),'hex'),1)`, event, f.node, f.inc)
		exec(t, tx, `INSERT INTO usage_samples(id,event_id,record_index,member_id,device_id,binding_revision,meter_epoch_id,interval_start,interval_end,upload_bytes,download_bytes,metering_version,monotonic_elapsed_ns,clock_quality) VALUES($1,$2,0,$3,$4,1,gen_random_uuid(),'2026-01-31Z','2026-02-02Z',2,0,'test-meter',1,'trusted')`, sample, event, f.member, f.device)
		exec(t, tx, `INSERT INTO usage_event_payloads(event_id,payload,created_at) VALUES($1,'\x01',now()-interval '40 days')`, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt := `INSERT INTO projection_receipts(sample_id,projection_name,generation,period) VALUES($1,'monthly',$2,$3)`
	rollup := `INSERT INTO usage_rollups(member_id,period,granularity,projection_generation,metering_version,upload_bytes,download_bytes,last_observed_at,aggregate_as_of,period_start,period_end) VALUES($1,$2,'month',$3,'test-meter',1,0,now(),now(),$4,$5)`
	checkPurge := func(want int) {
		t.Helper()
		var n int
		if err := p.QueryRow(ctx, `SELECT purge_retained('usage_event_payloads',now()-interval '36 days','projection test')`).Scan(&n); err != nil || n != want {
			t.Fatalf("purged %d want %d: %v", n, want, err)
		}
	}
	exec(t, p, receipt, sample, "1", "2026-01")
	exec(t, p, rollup, f.member, "2026-01", "1", "2026-01-01Z", "2026-02-01Z")
	checkPurge(0)
	exec(t, p, receipt, sample, "2", "2026-02")
	exec(t, p, rollup, f.member, "2026-02", "2", "2026-02-01Z", "2026-03-01Z")
	checkPurge(0)
	exec(t, p, receipt, sample, "1", "2026-02")
	exec(t, p, rollup, f.member, "2026-02", "1", "2026-02-01Z", "2026-03-01Z")
	checkPurge(1)
}

func Test15_4_PayloadAndUpdateResultUnique(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `INSERT INTO usage_event_payloads(event_id,payload) VALUES($1,'\x01')`, f.event)
	_, err := p.Exec(context.Background(), `INSERT INTO usage_event_payloads(event_id,payload) VALUES($1,'\x02')`, f.event)
	sqlError(t, err, "23505", "usage_event_payloads_event_id_key")
	exec(t, p, `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, f.inc, f.target)
	result := uuid.New()
	exec(t, p, `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation,result_id,result_hash) VALUES($1,1,$2,1,$3,decode(repeat('01',32),'hex'))`, f.target, f.inc, result)
	_, err = p.Exec(context.Background(), `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation,result_id,result_hash) VALUES($1,2,$2,1,$3,decode(repeat('01',32),'hex'))`, f.target, f.inc, result)
	sqlError(t, err, "23505", "update_jobs_result_id_key")
}

func TestUPD_SnapshotIncarnationCannotChange(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	exec(t, p, `UPDATE node_incarnations SET revoked_at=now() WHERE id=$1`, f.inc)
	replacement := uuid.New()
	exec(t, p, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch) VALUES($1,$2,decode(repeat('09',32),'hex'),'ubuntu','amd64')`, replacement, f.node)
	_, err := p.Exec(context.Background(), `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, replacement, f.target)
	sqlError(t, err, "23514", "update_targets_snapshot_execution_check")
}
