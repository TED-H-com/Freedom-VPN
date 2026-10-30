//go:build e2e && integration

package secops

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/jackc/pgx/v5/pgxpool"
)

// §17.3–17.4 / AT30: read-only checks before recovery changes the restored DB.
// Audit continuity is resource/request correlation and canonical row equality;
// the product does not implement a cryptographic audit hash chain.
func verifyPhysicalRestore(t *testing.T, pool *pgxpool.Pool, m pitrManifest) map[string]any {
	t.Helper()
	ctx := t.Context()
	var recovering bool
	var archive string
	if pool.QueryRow(ctx, `SELECT pg_is_in_recovery(),current_setting('archive_mode')`).Scan(&recovering, &archive) != nil || recovering || archive != "off" {
		t.Fatal("restored copy must be promoted with archiving off")
	}
	migrations, err := db.MigrationStatus(ctx, pool)
	if err != nil || len(migrations) == 0 {
		t.Fatal("restored migration inventory")
	}
	for _, migration := range migrations {
		if string(migration.State) != "applied" {
			t.Fatal("pending restored migration")
		}
	}
	var events, samples, postEvents int
	var bytes, ack, maxSeq string
	err = pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE event_seq>7) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2`, m.Node, m.Incarnation).Scan(&events, &postEvents)
	if err != nil || events != 7 || postEvents != 0 {
		t.Fatal("PITR ledger transaction boundary")
	}
	err = pool.QueryRow(ctx, `SELECT highest_contiguous_seq::text,max_seen_seq::text FROM usage_checkpoints WHERE node_id=$1 AND incarnation_id=$2`, m.Node, m.Incarnation).Scan(&ack, &maxSeq)
	if err != nil || ack != "7" || maxSeq != "7" {
		t.Fatal("PITR checkpoint boundary")
	}
	err = pool.QueryRow(ctx, `SELECT count(*),sum(s.upload_bytes+s.download_bytes)::text FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1 AND e.incarnation_id=$2 AND s.member_id=$3`, m.Node, m.Incarnation, m.Member).Scan(&samples, &bytes)
	if err != nil || samples != 7 || bytes != "2100" {
		t.Fatal("PITR ledger sample consistency")
	}
	var pre, post, preOp, postOp, audits, postAudits int
	err = pool.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM entitlement_grants WHERE id=$1 AND member_id=$3 AND operation_id=$4),
 (SELECT count(*) FROM entitlement_grants WHERE id=$2),
 (SELECT count(*) FROM operations WHERE id=$4 AND resource_id=$1 AND kind='member.grant'),
 (SELECT count(*) FROM operations WHERE id=$5),
 (SELECT count(*) FROM audit_events a JOIN operations o ON o.request_id=a.request_id WHERE o.id=$4 AND a.resource='entitlement_grant:'||$1::text AND a.action='member.grant' AND a.result='succeeded'),
 (SELECT count(*) FROM audit_events WHERE resource='entitlement_grant:'||$2::text)`, m.PreGrant, m.PostGrant, m.Member, m.PreOperation, m.PostOperation).Scan(&pre, &post, &preOp, &postOp, &audits, &postAudits)
	if err != nil || pre != 1 || preOp != 1 || audits != 1 || post != 0 || postOp != 0 || postAudits != 0 {
		t.Fatal("PITR grant/operation/request/audit boundary")
	}
	var end time.Time
	err = pool.QueryRow(ctx, `SELECT effective_service_end FROM entitlements WHERE member_id=$1`, m.Member).Scan(&end)
	if err != nil || !end.Equal(m.PreEnd) {
		t.Fatal("PITR entitlement differs from pre-target grant")
	}
	for id, expected := range m.AuditHashes {
		var raw string
		if pool.QueryRow(ctx, `SELECT row_to_json(a)::text FROM audit_events a WHERE id=$1`, id).Scan(&raw) != nil {
			t.Fatal("restored audit row missing")
		}
		actual, e := canonjson.HashHex([]byte(raw))
		if e != nil || actual != expected {
			t.Fatal("restored audit canonical row changed")
		}
	}
	return map[string]any{"promoted": true, "archive_mode": archive, "applied_migrations": len(migrations), "ledger_rows": events, "sample_rows": samples, "sample_bytes": bytes, "ack": ack, "max_seen_seq": maxSeq, "post_target_events": postEvents, "pre_target_grant_present": true, "post_target_grant_absent": true, "pre_target_operation_present": true, "post_target_operation_absent": true, "pre_target_entitlement_end": end, "pre_target_audit_rows_unchanged": len(m.AuditHashes), "post_target_audit_absent": true, "audit_correlation": "grant resource -> operation -> request -> succeeded append-only audit", "cryptographic_audit_chain": false}
}
