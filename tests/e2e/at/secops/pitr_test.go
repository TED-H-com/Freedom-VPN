//go:build e2e && integration

package secops

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type pitrManifest struct {
	Node, Incarnation, Member, PreGrant, PostGrant, PreOperation, PostOperation uuid.UUID
	Target, PreEnd, PostEnd, SourceWatermark                                    time.Time
	TargetLSN                                                                   string
	AuditHashes                                                                 map[string]string
	TargetACK                                                                   decstr.U64
}

// §17.2–17.4 / AT30. Three real sessions, API grants, native device admission,
// original mTLS uploads. This source half is independent of serving/recovery
// configuration and of the login-burst follow-up.
func TestAT30_LivePITRSource(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_PITR") != "1" {
		t.Skip("requires own running w4-secops lab")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "purpose": "AT30 physical PITR product-input source; not serving/capacity acceptance"}}
	p.open()
	revision, revisionErr := exec.Command("git", "rev-parse", "HEAD").Output()
	if revisionErr != nil {
		t.Fatal("source revision")
	}
	p.metadata["source_revision"] = strings.TrimSpace(string(revision))
	defer p.db.Close(context.Background())
	defer p.tr.CloseIdleConnections()
	defer p.finish()
	admin := p.login("e2e-admin", "admin")
	members := []*liveSession{}
	for _, subject := range []string{"secops-member-00", "secops-member-01", "secops-member-02"} {
		members = append(members, p.login(subject, "member"))
		time.Sleep(time.Second)
	}
	p.planInputs(admin)
	p.grantMembers(admin, members)
	b, node := p.enrollHost(admin)
	devices := p.admitMember(members)
	if len(p.bindings) != 1 || len(devices[0]) != 1 {
		t.Fatal("AT30 requires product-admitted binding")
	}
	p.physicalSource(admin, node, b, members[0], devices[0][0])
}

func (p *liveProbe) backupCommand(action string) []byte {
	ctx, cancel := context.WithTimeout(p.t.Context(), 150*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "deploy/backup/backup.sh", "--container", livePrefix+"postgres", action)
	cmd.Dir = p.root
	start := time.Now()
	raw, err := cmd.CombinedOutput()
	if os.WriteFile(filepath.Join(p.root, ".agent/secops/pitr-"+action+".log"), raw, 0600) != nil {
		p.t.Fatal("backup diagnostic write")
	}
	p.metadata["backup_"+action] = map[string]any{"command": []string{"deploy/backup/backup.sh", "--container", livePrefix + "postgres", action}, "elapsed_seconds": time.Since(start).Seconds(), "passed": err == nil}
	if err != nil {
		p.t.Fatalf("pgBackRest %s failed; protected log retained", action)
	}
	return raw
}

func (p *liveProbe) grantAt(admin, member *liveSession, end time.Time, label string) (uuid.UUID, uuid.UUID) {
	r := p.call(admin, "POST", "/api/v1/admin/members/"+member.member+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY PITR " + label, PlanRevisionId: &p.planRevision, ValidUntil: end}, nil)
	p.check("PITR "+label+" grant", r, 202, "")
	var op api.Operation
	if json.Unmarshal(r.body, &op) != nil || op.Resource.Id == uuid.Nil {
		p.t.Fatal("grant operation absent")
	}
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		var actual time.Time
		if p.db.QueryRow(p.t.Context(), `SELECT effective_service_end FROM entitlements WHERE member_id=$1`, member.member).Scan(&actual) == nil && actual.Equal(end) {
			return op.Resource.Id, op.OperationId
		}
		time.Sleep(time.Second)
	}
	p.t.Fatal("grant extension projection timeout")
	return uuid.Nil, uuid.Nil
}

func (p *liveProbe) physicalSource(admin, node *liveSession, b api.EnrollResponse, member *liveSession, device uuid.UUID) {
	now := time.Now().UTC().Truncate(time.Second)
	binding := p.bindings[device]
	if now.Add(-10*time.Second).Before(binding.ValidFrom) || !now.Before(binding.ValidUntil) {
		p.t.Fatal("binding does not cover retained interval")
	}
	batch := api.UsageBatchRequest{NodeId: b.NodeId, NodeIncarnationId: b.NodeIncarnationId, LedgerEpoch: presence.NewNullNullable[uuid.UUID](), Events: []api.UsageEvent{}}
	meter := uuid.New()
	for n := 1; n <= 10; n++ {
		ev, err := usage.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: b.NodeId, NodeIncarnationId: b.NodeIncarnationId, MeterEpochId: meter, EventSeq: decstr.U64(n), IntervalStart: now.Add(time.Duration(n-11) * time.Second), IntervalEnd: now.Add(time.Duration(n-10) * time.Second), MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete, Records: []api.UsageRecord{{MemberId: uuid.MustParse(member.member), DeviceId: device, BindingRevision: binding.BindingRevision, PeriodId: api.PeriodId(now.Format("2006-01")), UploadBytes: 100, DownloadBytes: 200}}})
		if err != nil {
			p.t.Fatal("usage hash")
		}
		batch.Events = append(batch.Events, ev)
	}
	send := func(n int) {
		part := batch
		part.Events = batch.Events[:n]
		r := p.call(node, "POST", liveAgent+"/api/v1/agent/usage-batches", part, nil)
		p.check("PITR upload", r, 200, "")
		var ack api.UsageBatchResult
		if json.Unmarshal(r.body, &ack) != nil || ack.HighestContiguousSeq != decstr.U64(n) {
			p.t.Fatalf("PITR expected ACK %d", n)
		}
	}
	p.backupCommand("check")
	send(5)
	p.backupCommand("full")
	info := p.backupCommand("info")
	var inventory []struct {
		Backup []struct {
			Label     string            `json:"label"`
			Timestamp map[string]int64  `json:"timestamp"`
			LSN       map[string]string `json:"lsn"`
		} `json:"backup"`
	}
	if json.Unmarshal(info, &inventory) != nil || len(inventory) != 1 || len(inventory[0].Backup) == 0 {
		p.t.Fatal("physical backup inventory")
	}
	backup := inventory[0].Backup[len(inventory[0].Backup)-1]
	// pgBackRest backup selection compares whole-second stop times. Wait on
	// PostgreSQL's clock so a fractional target in that same second cannot be
	// mistaken for a target preceding the completed backup. Never adjust clocks.
	boundary := time.Unix(backup.Timestamp["stop"]+1, 0).UTC()
	for deadline := time.Now().Add(10 * time.Second); ; {
		var databaseNow time.Time
		if p.db.QueryRow(p.t.Context(), `SELECT clock_timestamp()`).Scan(&databaseNow) != nil {
			p.t.Fatal("backup target boundary clock")
		}
		if !databaseNow.Before(boundary) {
			break
		}
		if time.Now().After(deadline) {
			p.t.Fatal("backup target boundary timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}
	send(7)
	manifest := pitrManifest{Node: b.NodeId, Incarnation: b.NodeIncarnationId, Member: uuid.MustParse(member.member), PreEnd: now.Add(70 * time.Minute), PostEnd: now.Add(80 * time.Minute), TargetACK: 7, AuditHashes: map[string]string{}}
	manifest.PreGrant, manifest.PreOperation = p.grantAt(admin, member, manifest.PreEnd, "before-target")
	// Capture only committed grant audit rows, including their canonical operation
	// resource and request correlation. The DB has append-only audit rows, not a hash chain.
	rows, err := p.db.Query(p.t.Context(), `SELECT a.id,row_to_json(a)::text FROM audit_events a WHERE a.resource IN (SELECT 'entitlement_grant:'||id FROM entitlement_grants WHERE member_id=$1) ORDER BY a.id`, manifest.Member)
	if err != nil {
		p.t.Fatal("source audit read")
	}
	for rows.Next() {
		var id uuid.UUID
		var raw string
		if rows.Scan(&id, &raw) != nil {
			p.t.Fatal("source audit decode")
		}
		hash, e := canonjson.HashHex([]byte(raw))
		if e != nil {
			p.t.Fatal("source audit hash")
		}
		manifest.AuditHashes[id.String()] = hash
	}
	rows.Close()
	if rows.Err() != nil || len(manifest.AuditHashes) < 2 {
		p.t.Fatal("pre-target grant audit chain missing")
	}
	if err = p.db.QueryRow(p.t.Context(), `SELECT clock_timestamp(),pg_current_wal_lsn()::text`).Scan(&manifest.Target, &manifest.TargetLSN); err != nil {
		p.t.Fatal("database target clock")
	}
	manifest.Target = manifest.Target.UTC()
	if !manifest.Target.After(time.Unix(backup.Timestamp["stop"], 0)) {
		p.t.Fatal("PITR target must follow backup completion")
	}
	time.Sleep(time.Second) // distinct, observable post-target transaction boundary
	send(10)
	manifest.PostGrant, manifest.PostOperation = p.grantAt(admin, member, manifest.PostEnd, "after-target")
	if p.db.QueryRow(p.t.Context(), `SELECT clock_timestamp()`).Scan(&manifest.SourceWatermark) != nil {
		p.t.Fatal("source watermark")
	}
	manifest.SourceWatermark = manifest.SourceWatermark.UTC()
	p.backupCommand("wal-switch")
	writePrivate := func(name string, v any) {
		raw, e := json.MarshalIndent(v, "", "  ")
		if e != nil || os.WriteFile(filepath.Join(p.root, ".agent/secops", name), raw, 0600) != nil {
			p.t.Fatal("PITR fixture write")
		}
	}
	writePrivate("pitr-manifest.json", manifest)
	writePrivate("retained.json", batch)
	var grants, events, audit int
	err = p.db.QueryRow(p.t.Context(), `SELECT (SELECT count(*) FROM entitlement_grants WHERE id=ANY($1::uuid[])),(SELECT count(*) FROM usage_events WHERE node_id=$2 AND incarnation_id=$3),(SELECT count(*) FROM audit_events WHERE resource='entitlement_grant:'||$4::text AND request_id IS NOT NULL AND action='member.grant' AND result='succeeded')`, []uuid.UUID{manifest.PreGrant, manifest.PostGrant}, b.NodeId, b.NodeIncarnationId, manifest.PostGrant).Scan(&grants, &events, &audit)
	if err != nil || grants != 2 || events != 10 || audit != 1 {
		p.t.Fatal("source positive controls missing")
	}
	p.metadata["physical_source"] = map[string]any{"backup_label": backup.Label, "backup_timestamp": backup.Timestamp, "backup_lsn": backup.LSN, "target_time": manifest.Target, "target_lsn": manifest.TargetLSN, "source_watermark": manifest.SourceWatermark, "expected_restored_ack": manifest.TargetACK, "source_ack": decstr.U64(10), "pre_and_post_grants_present": grants == 2, "post_target_audit_present": audit == 1, "pre_target_audit_rows": len(manifest.AuditHashes), "db_rpo_seconds": manifest.SourceWatermark.Sub(manifest.Target).Seconds(), "full_service_rpo_seconds": nil, "full_service_rto_seconds": nil, "archive_source": "original mTLS-uploaded events retained by test sender; not node SQLite archive", "projector_owned_rows_written": []string{}}
}
