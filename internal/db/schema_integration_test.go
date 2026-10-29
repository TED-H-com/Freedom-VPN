//go:build integration

package db_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	osexec "os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func exec(t *testing.T, p interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, sql string, args ...any) {
	t.Helper()
	if _, err := p.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("SQL failed: %v\n%s", err, sql)
	}
}
func sqlError(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var e *pgconn.PgError
	if !errors.As(err, &e) || e.Code != code || (constraint != "" && e.ConstraintName != constraint) {
		t.Fatalf("want %s / %s, got %v", code, constraint, err)
	}
}

// fixture stores a small connected graph. UUIDs/keys are generated at runtime.
// TEST ONLY - NOT A SECRET: repeated byte hashes represent synthetic content.
type fixture struct{ member, identity, plan, revision, sub, inbox, device, binding, group, node, inc, release, policyDef, policy, exit, event, sample, op, batch, target, lease uuid.UUID }

func seed(t *testing.T, p *pgxpool.Pool) fixture {
	t.Helper()
	tx, err := p.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	f := fixture{}
	for _, id := range []*uuid.UUID{&f.member, &f.identity, &f.plan, &f.revision, &f.sub, &f.inbox, &f.device, &f.binding, &f.group, &f.node, &f.inc, &f.release, &f.policyDef, &f.policy, &f.exit, &f.event, &f.sample, &f.op, &f.batch, &f.target, &f.lease} {
		*id = uuid.New()
	}
	exec(t, tx, `INSERT INTO members(id,status) VALUES($1,'active')`, f.member)
	exec(t, tx, `INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,'https://id.example.invalid','portal','subject','verified-claim','verified-link',now())`, f.identity, f.member)
	exec(t, tx, `INSERT INTO plans(id,code) VALUES($1,'test-plan')`, f.plan)
	exec(t, tx, `INSERT INTO plan_revisions(id,plan_id,revision,device_limit,schema_version,payload) VALUES($1,$2,1,2,1,'{}')`, f.revision, f.plan)
	exec(t, tx, `INSERT INTO subscriptions(id,member_id,provider,external_subscription_id,external_customer_id,plan_revision_id,provider_status,is_active) VALUES($1,$2,'test-provider','sub','customer',$3,'active',true)`, f.sub, f.member, f.revision)
	exec(t, tx, `INSERT INTO inbox_events(id,source,external_event_id,payload_hash,encrypted_payload) VALUES($1,'test-provider','event',decode(repeat('01',32),'hex'),'\x01')`, f.inbox)
	exec(t, tx, `INSERT INTO member_devices(id,member_id,deployment_id,peer_id) VALUES($1,$2,'test-deployment','peer')`, f.device, f.member)
	exec(t, tx, `INSERT INTO device_bindings(id,device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,$3,'overlay','100.64.0.1',1,'2026-01-01Z','2027-01-01Z','peer')`, f.binding, f.device, f.member)
	exec(t, tx, `INSERT INTO member_groups(id,code,name) VALUES($1,'test-group','Test group')`, f.group)
	exec(t, tx, `INSERT INTO releases(id,version,release_sequence,manifest_schema_version,manifest,manifest_hash,signature,signing_key_ref,component_versions) VALUES($1,'TEST ONLY',1,1,'{}',decode(repeat('02',32),'hex'),decode(repeat('03',64),'hex'),'test-key','{}')`, f.release)
	exec(t, tx, `INSERT INTO nodes(id) VALUES($1)`, f.node)
	exec(t, tx, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,arch) VALUES($1,$2,decode(repeat('04',32),'hex'),'test-deployment','node-peer','ubuntu','amd64')`, f.inc, f.node)
	exec(t, tx, `UPDATE nodes SET lifecycle_state='registered',current_incarnation_id=$1,resource_version=resource_version+1 WHERE id=$2`, f.inc, f.node)
	exec(t, tx, `INSERT INTO exits(id,node_id,display_name,region) VALUES($1,$2,'test','test-region')`, f.exit, f.node)
	exec(t, tx, `INSERT INTO policy_definitions(id,type,code) VALUES($1,'fup','test')`, f.policyDef)
	exec(t, tx, `INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,'fup',1,1,'{}',decode(repeat('05',32),'hex'),'test-key','test-actor')`, f.policy, f.policyDef)
	exec(t, tx, `INSERT INTO usage_events(id,node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,$3,1,1,decode(repeat('06',32),'hex'),1)`, f.event, f.node, f.inc)
	exec(t, tx, `INSERT INTO usage_samples(id,event_id,record_index,member_id,device_id,binding_revision,meter_epoch_id,interval_start,interval_end,upload_bytes,download_bytes,metering_version,monotonic_elapsed_ns,clock_quality) VALUES($1,$2,0,$3,$4,1,gen_random_uuid(),'2026-01-01Z','2026-01-02Z',1,2,'test-meter',1,'trusted')`, f.sample, f.event, f.member, f.device)
	exec(t, tx, `INSERT INTO operations(id,kind,principal,resource_type) VALUES($1,'test','test-actor','node')`, f.op)
	err = func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), `INSERT INTO update_batches(id,operation_id,target_release_id,scope,created_by,target_count,generation) VALUES($1,$2,$3,'all_nodes','test-actor',1,1)`, f.batch, f.op, f.release); err != nil {
			return err
		}
		_, err := tx.Exec(context.Background(), `INSERT INTO update_targets(id,batch_id,node_id,incarnation_at_snapshot,target_generation) VALUES($1,$2,$3,$4,1)`, f.target, f.batch, f.node, f.inc)
		return err
	}(tx)
	if err != nil {
		t.Fatal(err)
	}
	exec(t, tx, `INSERT INTO entitlements(member_id,revision,device_limit,valid_from,effective_service_end) VALUES($1,1,2,'2026-01-01Z','2027-01-01Z')`, f.member)
	exec(t, tx, `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash) VALUES($1,$2,$3,1,'2026-01-01Z','2026-01-02Z','test-key',decode(repeat('07',32),'hex'))`, f.lease, f.node, f.inc)
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	return f
}

// Derive expectations from the embedded source so later migration waves are
// checked automatically, including ordering and missing/pending versions.
func embeddedMigrationVersions(t *testing.T) []int64 {
	t.Helper()
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil || len(names) == 0 {
		t.Fatalf("embedded migration files: %v (%d files)", err, len(names))
	}
	versions := make([]int64, 0, len(names))
	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		version, err := strconv.ParseInt(prefix, 10, 64)
		if !ok || err != nil || version <= 0 {
			t.Fatalf("invalid embedded migration name %q", name)
		}
		versions = append(versions, version)
	}
	slices.Sort(versions)
	for i := 1; i < len(versions); i++ {
		if versions[i] == versions[i-1] {
			t.Fatalf("duplicate embedded migration version %d", versions[i])
		}
	}
	return versions
}

func Test15_5_Migrations(t *testing.T) {
	p := dbtest.NewUnmigrated(t)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() { errs <- db.Migrate(context.Background(), p) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Migrate(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	expected := embeddedMigrationVersions(t)
	status, err := db.MigrationStatus(context.Background(), p)
	if err != nil || len(status) != len(expected) {
		t.Fatalf("status count=%d want=%d error=%v", len(status), len(expected), err)
	}
	for i, s := range status {
		if s.Source.Version != expected[i] || string(s.State) != "applied" {
			t.Fatalf("status[%d]=%+v want version=%d applied", i, s, expected[i])
		}
	}
}

func Test15_1_SchemaGuards(t *testing.T) {
	p := dbtest.New(t)
	checks := map[string]string{
		"no floats or money":               `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND data_type IN ('real','double precision','money')`,
		"no timestamp without zone":        `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND data_type='timestamp without time zone'`,
		"no cascade":                       `SELECT count(*) FROM pg_constraint WHERE connamespace='public'::regnamespace AND contype='f' AND confdeltype='c'`,
		"every table has primary key":      `SELECT count(*) FROM pg_class c WHERE c.relnamespace='public'::regnamespace AND c.relkind='r' AND NOT EXISTS(SELECT 1 FROM pg_constraint k WHERE k.conrelid=c.oid AND k.contype='p')`,
		"every domain primary key is UUID": `SELECT count(*) FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum=ANY(k.conkey) WHERE c.relnamespace='public'::regnamespace AND k.contype='p' AND c.relname<>'goose_db_version' AND a.atttypid<>'uuid'::regtype`,
		"every domain table cites spec":    `SELECT count(*) FROM pg_class WHERE relnamespace='public'::regnamespace AND relkind='r' AND relname<>'goose_db_version' AND coalesce(obj_description(oid),'') NOT LIKE '%§%'`,
	}
	for name, q := range checks {
		t.Run(name, func(t *testing.T) {
			var n int
			if err := p.QueryRow(context.Background(), q).Scan(&n); err != nil || n != 0 {
				t.Fatalf("violations=%d: %v", n, err)
			}
		})
	}
}

func Test15_2_UniqueRules(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	type rule struct {
		name, setup, insert, constraint string
		args                            []any
	}
	rules := []rule{
		{"oidc scope", "", `INSERT INTO identity_bindings(member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,'https://id.example.invalid','portal','subject','claim','test',now())`, "identity_bindings_subject_key", []any{f.member}},
		{"netbird user", `INSERT INTO netbird_user_bindings(member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,'test','user',$2,'proof',1,now())`, `INSERT INTO netbird_user_bindings(member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,'test','user',$2,'proof',1,now())`, "netbird_user_bindings_external_key", []any{f.member, f.identity}},
		{"plan code", "", `INSERT INTO plans(code) VALUES('test-plan')`, "plans_code_key", nil},
		{"plan revision", "", `INSERT INTO plan_revisions(plan_id,revision,device_limit,schema_version,payload) VALUES($1,1,2,1,'{}')`, "plan_revisions_revision_key", []any{f.plan}},
		{"subscription external", "", `INSERT INTO subscriptions(member_id,provider,external_subscription_id,external_customer_id,plan_revision_id,provider_status,is_active) VALUES($1,'test-provider','sub','customer',$2,'active',false)`, "subscriptions_external_key", []any{f.member, f.revision}},
		{"subscription primary active", "", `INSERT INTO subscriptions(member_id,provider,external_subscription_id,external_customer_id,plan_revision_id,provider_status,is_active) VALUES($1,'test-provider','different-sub','customer',$2,'active',true)`, "subscriptions_primary_active_key", []any{f.member, f.revision}},
		{"billing reference", `INSERT INTO billing_records(member_id,subscription_id,provider,record_type,external_reference,amount_minor,currency,service_start,service_end,source_event_id) VALUES($1,$2,'test','invoice','invoice',100,'USD','2026-01-01Z','2026-02-01Z',$3)`, `INSERT INTO billing_records(member_id,subscription_id,provider,record_type,external_reference,amount_minor,currency,service_start,service_end,source_event_id) VALUES($1,$2,'test','invoice','invoice',100,'USD','2026-01-01Z','2026-02-01Z',$3)`, "billing_records_external_key", []any{f.member, f.sub, f.inbox}},
		{"entitlement member", "", `INSERT INTO entitlements(member_id,revision,device_limit,valid_from,effective_service_end) VALUES($1,2,2,'2026-01-01Z','2026-02-01Z')`, "entitlements_member_id_key", []any{f.member}},
		{"device peer", "", `INSERT INTO member_devices(member_id,deployment_id,peer_id) VALUES($1,'test-deployment','peer')`, "member_devices_peer_key", []any{f.member}},
		{"binding revision", "", `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,'overlay','100.64.0.2',1,'2026-01-01Z','2027-01-01Z','peer')`, "device_bindings_revision_key", []any{f.device, f.member}},
		{"group code", "", `INSERT INTO member_groups(code,name) VALUES('test-group','Other')`, "member_groups_code_key", nil},
		{"group name", "", `INSERT INTO member_groups(code,name) VALUES('other','Test group')`, "member_groups_name_key", nil},
		{"incarnation active", "", `INSERT INTO node_incarnations(node_id,public_key_fingerprint,os,arch) VALUES($1,decode(repeat('08',32),'hex'),'ubuntu','amd64')`, "node_incarnations_active_key", []any{f.node}},
		{"incarnation key", "", `INSERT INTO node_incarnations(node_id,public_key_fingerprint,os,arch,revoked_at) VALUES($1,decode(repeat('04',32),'hex'),'ubuntu','amd64',now())`, "node_incarnations_public_key_fingerprint_key", []any{f.node}},
		{"incarnation peer", "", `INSERT INTO node_incarnations(node_id,public_key_fingerprint,deployment_id,peer_id,os,arch,revoked_at) VALUES($1,decode(repeat('09',32),'hex'),'test-deployment','node-peer','ubuntu','amd64',now())`, "node_incarnations_peer_key", []any{f.node}},
		{"enrollment hash", `INSERT INTO node_enrollments(node_id,token_hash,expiry) VALUES($1,decode(repeat('0a',32),'hex'),now()+interval '1 hour')`, `INSERT INTO node_enrollments(node_id,token_hash,expiry) VALUES($1,decode(repeat('0a',32),'hex'),now()+interval '1 hour')`, "node_enrollments_token_hash_key", []any{f.node}},
		{"certificate issuer serial", `INSERT INTO node_certificates(incarnation_id,issuer,serial,fingerprint,public_certificate,not_before,not_after) VALUES($1,'issuer','serial',decode(repeat('0b',32),'hex'),'\x01',now(),now()+interval '1 day')`, `INSERT INTO node_certificates(incarnation_id,issuer,serial,fingerprint,public_certificate,not_before,not_after) VALUES($1,'issuer','serial',decode(repeat('0b',32),'hex'),'\x01',now(),now()+interval '1 day')`, "node_certificates_serial_key", []any{f.inc}},
		{"exit node", "", `INSERT INTO exits(node_id,display_name,region) VALUES($1,'test','test')`, "exits_node_id_key", []any{f.node}},
		{"exit group", `INSERT INTO exit_groups(exit_id,group_id) VALUES($1,$2)`, `INSERT INTO exit_groups(exit_id,group_id) VALUES($1,$2)`, "exit_groups_membership_key", []any{f.exit, f.group}},
		{"policy type code", "", `INSERT INTO policy_definitions(type,code) VALUES('fup','test')`, "policy_definitions_code_key", nil},
		{"policy revision", "", `INSERT INTO policy_revisions(definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,'fup',1,1,'{}',decode(repeat('0c',32),'hex'),'test-key','test')`, "policy_revisions_revision_key", []any{f.policyDef}},
		{"desired generation", `INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,config_hash,schema_version,signed_at) VALUES($1,$2,1,'{}',1,decode(repeat('0d',32),'hex'),1,now())`, `INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,config_hash,schema_version,signed_at) VALUES($1,$2,1,'{}',1,decode(repeat('0d',32),'hex'),1,now())`, "desired_states_generation_key", []any{f.node, f.inc}},
		{"report id", `INSERT INTO node_observations(node_id,incarnation_id,report_id,payload_hash,observed_at,applied_generation,schema_version,checks,control_status,data_health,safety_status) VALUES($1,$2,$3,decode(repeat('0e',32),'hex'),now(),1,1,'{}','online','healthy','passed')`, `INSERT INTO node_observations(node_id,incarnation_id,report_id,payload_hash,observed_at,applied_generation,schema_version,checks,control_status,data_health,safety_status) VALUES($1,$2,$3,decode(repeat('0e',32),'hex'),now(),1,1,'{}','online','healthy','passed')`, "node_observations_report_key", []any{f.node, f.inc, uuid.New()}},
		{"external internal", `INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash) VALUES('node',$1,'test','test','ext',decode(repeat('0f',32),'hex'))`, `INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash) VALUES('node',$1,'test','test','other-ext',decode(repeat('0f',32),'hex'))`, "external_bindings_internal_key", []any{f.node}},
		{"external external", "", `INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash) VALUES('node',gen_random_uuid(),'test','test','ext',decode(repeat('0f',32),'hex'))`, "external_bindings_external_key", nil},
		{"event stream", "", `INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count) VALUES($1,$2,1,1,decode(repeat('10',32),'hex'),0)`, "usage_events_stream_key", []any{f.node, f.inc}},
		{"sample record", "", `INSERT INTO usage_samples(event_id,record_index,member_id,device_id,binding_revision,meter_epoch_id,interval_start,interval_end,upload_bytes,download_bytes,metering_version,monotonic_elapsed_ns,clock_quality) VALUES($1,0,$2,$3,1,gen_random_uuid(),'2026-01-01Z','2026-01-02Z',0,0,'test',1,'trusted')`, "usage_samples_record_key", []any{f.event, f.member, f.device}},
		{"projection receipt", `INSERT INTO projection_receipts(sample_id,projection_name,generation,period) VALUES($1,'monthly',1,'2026-01')`, `INSERT INTO projection_receipts(sample_id,projection_name,generation,period) VALUES($1,'monthly',1,'2026-01')`, "projection_receipts_projection_key", []any{f.sample}},
		{"checkpoint stream", `INSERT INTO usage_checkpoints(node_id,incarnation_id,highest_contiguous_seq,max_seen_seq,ledger_epoch) VALUES($1,$2,1,1,gen_random_uuid())`, `INSERT INTO usage_checkpoints(node_id,incarnation_id,highest_contiguous_seq,max_seen_seq,ledger_epoch) VALUES($1,$2,1,1,gen_random_uuid())`, "usage_checkpoints_stream_key", []any{f.node, f.inc}},
		{"rollup period", `INSERT INTO usage_rollups(member_id,period,granularity,projection_generation,metering_version,upload_bytes,download_bytes,last_observed_at,aggregate_as_of,period_start,period_end) VALUES($1,'2026-01','month',1,'test',0,0,now(),now(),'2026-01-01Z','2026-02-01Z')`, `INSERT INTO usage_rollups(member_id,period,granularity,projection_generation,metering_version,upload_bytes,download_bytes,last_observed_at,aggregate_as_of,period_start,period_end) VALUES($1,'2026-01','month',1,'test',0,0,now(),now(),'2026-01-01Z','2026-02-01Z')`, "usage_rollups_period_key", []any{f.member}},
		{"release version", "", `INSERT INTO releases(version,release_sequence,manifest_schema_version,manifest,manifest_hash,signature,signing_key_ref,component_versions) VALUES('TEST ONLY',1,1,'{}',decode(repeat('11',32),'hex'),decode(repeat('12',64),'hex'),'test','{}')`, "releases_version_key", nil},
		{"artifact platform", `INSERT INTO artifacts(release_id,platform,arch,digest,size_bytes,installed_size_bytes,storage_ref) VALUES($1,'ubuntu-24.04','amd64',decode(repeat('13',32),'hex'),1,1,'https://releases.example.invalid/test')`, `INSERT INTO artifacts(release_id,platform,arch,digest,size_bytes,installed_size_bytes,storage_ref) VALUES($1,'ubuntu-24.04','amd64',decode(repeat('13',32),'hex'),1,1,'https://releases.example.invalid/test')`, "artifacts_platform_key", []any{f.release}},
		{"inbox event", "", `INSERT INTO inbox_events(source,external_event_id,payload_hash,encrypted_payload) VALUES('test-provider','event',decode(repeat('14',32),'hex'),'\x01')`, "inbox_events_external_key", nil},
		{"idempotency scope", `INSERT INTO idempotency_requests(principal,method,route,key_hash,request_hash,resource_id,response_status,expiry) VALUES('test','POST','/test',decode(repeat('15',32),'hex'),decode(repeat('16',32),'hex'),gen_random_uuid(),201,now()+interval '7 days')`, `INSERT INTO idempotency_requests(principal,method,route,key_hash,request_hash,resource_id,response_status,expiry) VALUES('test','POST','/test',decode(repeat('15',32),'hex'),decode(repeat('16',32),'hex'),gen_random_uuid(),201,now()+interval '7 days')`, "idempotency_requests_scope_key", nil},
	}
	for _, r := range rules {
		t.Run(r.name, func(t *testing.T) {
			if r.setup != "" {
				exec(t, p, r.setup, r.args...)
			}
			_, err := p.Exec(context.Background(), r.insert, r.args...)
			sqlError(t, err, "23505", r.constraint)
		})
	}
}

func TestUPD_UpdateTargetAndJobUnique(t *testing.T) {
	p := dbtest.New(t)
	f := seed(t, p)
	// Duplicate membership is tested in its allowed creation transaction.
	tx, err := p.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var batch uuid.UUID
	err = tx.QueryRow(context.Background(), `INSERT INTO update_batches(operation_id,target_release_id,scope,created_by,target_count,generation) VALUES($1,$2,'all_nodes','test',1,2) RETURNING id`, f.op, f.release).Scan(&batch)
	sqlError(t, err, "23505", "update_batches_operation_id_key")
	_ = tx.Rollback(context.Background())
	exec(t, p, `UPDATE update_targets SET execution_incarnation_id=$1 WHERE id=$2`, f.inc, f.target)
	q := `INSERT INTO update_jobs(target_id,attempt,execution_incarnation_id,generation) VALUES($1,1,$2,1)`
	exec(t, p, q, f.target, f.inc)
	_, err = p.Exec(context.Background(), q, f.target, f.inc)
	sqlError(t, err, "23505", "update_jobs_attempt_key")
	err = db.InTx(context.Background(), p, db.TxOptions{}, func(tx pgx.Tx) error {
		var op, b uuid.UUID
		if err := tx.QueryRow(context.Background(), `INSERT INTO operations(kind,principal,resource_type) VALUES('test','test','node') RETURNING id`).Scan(&op); err != nil {
			return err
		}
		if err := tx.QueryRow(context.Background(), `INSERT INTO update_batches(operation_id,target_release_id,scope,created_by,target_count,generation) VALUES($1,$2,'all_nodes','test',1,2) RETURNING id`, op, f.release).Scan(&b); err != nil {
			return err
		}
		q := `INSERT INTO update_targets(batch_id,node_id,incarnation_at_snapshot,target_generation) VALUES($1,$2,(SELECT current_incarnation_id FROM nodes WHERE id=$2),2)`
		if _, err := tx.Exec(context.Background(), q, b, f.node); err != nil {
			return err
		}
		_, err := tx.Exec(context.Background(), q, b, f.node)
		return err
	})
	sqlError(t, err, "23505", "update_targets_node_key")
}

// §15.5: the operator CLI still applies an empty schema, is idempotent, and
// reports every embedded migration. All subprocess scratch inherits the
// integration runner's disk TMPDIR; credentials are passed only in environment.
func Test15_5_MigrateCLI(t *testing.T) {
	p := dbtest.NewUnmigrated(t)
	binary := filepath.Join(t.TempDir(), "fvpn-migrate")
	build := osexec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/fvpn-migrate")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build migration CLI: %v\n%s", err, output)
	}
	run := func(arg string) string {
		t.Helper()
		cmd := osexec.CommandContext(t.Context(), binary, arg)
		cmd.Env = append(os.Environ(), "FVPN_DATABASE_URL="+p.Config().ConnConfig.ConnString())
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("migration CLI %s: %v\n%s", arg, err, output)
		}
		return string(output)
	}
	for range 2 {
		if output := run("up"); strings.TrimSpace(output) != "migrations applied" {
			t.Fatalf("unexpected up output: %s", output)
		}
	}
	versions := embeddedMigrationVersions(t)
	var expected strings.Builder
	for _, version := range versions {
		fmt.Fprintf(&expected, "%05d applied\n", version)
	}
	if output := run("status"); output != expected.String() {
		t.Fatalf("unexpected status: %s", output)
	}
	t.Logf("CLI: empty up, repeated up and all %d embedded migration statuses verified", len(versions))
}
