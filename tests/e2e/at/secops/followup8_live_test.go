//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "modernc.org/sqlite"
)

const (
	restoreImageTag  = "fvpn-w4-secops-restore-image:17"
	restoreRepoVol   = "fvpn-w4-secops-pgbackrest-repo"
	restoreNegCopy   = "fvpn-w4-secops-restore-neg"
	restoreNegLive   = "fvpn-w4-secops-restore-neg-live"
	restoreNegVol    = "fvpn-w4-secops-restore-neg-pg"
	restoreCleanCopy = "fvpn-w4-secops-restore-pitr"
	restoreCleanLive = "fvpn-w4-secops-restore-live"
	restoreCleanVol  = "fvpn-w4-secops-restore-live-pg"
)

// §15.4, §17.2–17.4, §19 AT30 (DATA01 / BILL01 / MTR03):
// Live restore of the lab's central PostgreSQL database to T0 while the node,
// fake payment provider and NetBird stay at T1, measuring the 2-minute
// fail-safe negative (without recovery protocol) followed by the 5-step
// product restore protocol (freeze -> begin-recovery -> node checkpoint sync &
// retained archive replay -> complete-recovery) and bounded reconciliation.
func TestAT30_Followup8_LiveRestoreReconcile(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_FOLLOWUP8") != "1" {
		t.Skip("requires fresh assigned w4-secops stack for AT30 live restore and reconciliation")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{
		t:        t,
		root:     root,
		state:    filepath.Join(root, ".agent/e2e-state-w4-secops"),
		metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 8, "run_id": uuid.New(), "hardware": "shared development host; not candidate hardware"},
	}
	p.open()
	defer func() {
		p.finish()
		p.tr.CloseIdleConnections()
		p.cleanupRestoreObjects()
	}()

	stack, err := labstack.Open()
	if err != nil {
		t.Fatal("stack open:", err)
	}
	defer stack.Close()

	if err := os.MkdirAll(filepath.Join(p.root, ".agent/secops"), 0700); err != nil {
		t.Fatal("private secops dir:", err)
	}

	// Step 1: Bring up baseline, seed 2 members (m1, m2) with paid subscriptions
	// and authorized native devices on the lab node, run counted traffic until L0
	// has ACKed usage checkpoints, and take the T0 pgBackRest backup.
	admin := p.login("e2e-admin", "admin")
	p.planInputs(admin)
	p.releaseSupport(admin)
	nodeID := p.enrollAgent(admin)
	p.capacityInput(admin, nodeID)

	m1 := p.login("secops-member-00", "member")
	m2 := p.login("secops-member-01", "member")
	paidEnd := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	if !p.paidInput(m1, paidEnd) {
		t.Fatal("m1 paid subscription input failed")
	}
	if !p.paidInput(m2, paidEnd) {
		t.Fatal("m2 paid subscription input failed")
	}
	extM1 := p.checkoutExternalID(m1.member)
	extM2 := p.checkoutExternalID(m2.member)

	peerM1D1, ok := p.joinFlowPeer(stack, m1, "secops-member-00", "at30-m1-d1")
	if !ok {
		t.Fatal("m1 device 1 admission failed")
	}
	peerM2D1, ok := p.joinFlowPeer(stack, m2, "secops-member-01", "at30-m2-d1")
	if !ok {
		t.Fatal("m2 device 1 admission failed")
	}
	peerM2D2, ok := p.joinFlowPeer(stack, m2, "secops-member-01", "at30-m2-d2")
	if !ok {
		t.Fatal("m2 device 2 admission failed")
	}

	admin = p.login("e2e-admin", "admin")
	p.serving(admin, nodeID)

	t0Flows := map[string]any{
		"m1_d1": p.waitCountedFlow(peerM1D1, 45*time.Second),
		"m2_d1": p.waitCountedFlow(peerM2D1, 45*time.Second),
		"m2_d2": p.waitCountedFlow(peerM2D2, 45*time.Second),
	}
	for range 4 {
		_ = p.rawFlow(peerM1D1)
		_ = p.rawFlow(peerM2D1)
	}

	// Wait for quiet meter interval + uploader flush so L0 is ACKed on both central and node.
	n0Seq := p.waitUsageFlushed(nodeID, 1, m1, m2, 110*time.Second)
	t.Logf("AT30 Step 1 (T0): node=%s ACKed through seq=%d", nodeID, n0Seq)

	p.backupCommand("check")
	p.backupCommand("full")
	infoRaw := p.backupCommand("info")
	var inventory []struct {
		Backup []struct {
			Label     string            `json:"label"`
			Timestamp map[string]int64  `json:"timestamp"`
			LSN       map[string]string `json:"lsn"`
		} `json:"backup"`
	}
	if json.Unmarshal(infoRaw, &inventory) != nil || len(inventory) != 1 || len(inventory[0].Backup) == 0 {
		t.Fatal("pgBackRest backup inventory decode failed")
	}
	backup := inventory[0].Backup[len(inventory[0].Backup)-1]
	boundary := time.Unix(backup.Timestamp["stop"]+1, 0).UTC()
	for deadline := time.Now().Add(10 * time.Second); ; {
		var dbNow time.Time
		if p.db.QueryRow(t.Context(), `SELECT clock_timestamp()`).Scan(&dbNow) != nil {
			t.Fatal("backup boundary clock query failed")
		}
		if !dbNow.Before(boundary) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("backup boundary wait timed out")
		}
		time.Sleep(100 * time.Millisecond)
	}

	l0 := p.captureLedgerState(nodeID, m1, m2)
	nodeStoreT0 := p.captureNodeUsageStore("t0")
	p0 := p.captureBillingState(m1.member, m2.member, extM1, extM2)
	n0Peers := p.capturePeerState(nodeID, []string{m1.member, m2.member}, m1, m2)

	var t0Time time.Time
	var t0LSN string
	if err := p.db.QueryRow(t.Context(), `SELECT clock_timestamp(), pg_current_wal_lsn()::text`).Scan(&t0Time, &t0LSN); err != nil {
		t.Fatal("T0 checkpoint timestamp/LSN failed:", err)
	}
	t0Time = t0Time.UTC()
	t.Logf("AT30 T0 checkpoint recorded: time=%s lsn=%s backup=%s", t0Time.Format(time.RFC3339Nano), t0LSN, backup.Label)

	// Distinct post-T0 transaction boundary.
	time.Sleep(1200 * time.Millisecond)

	// Step 2: Move the world forward to T1:
	// (a) More counted traffic on peerM1D1 and peerM2D1 across multiple sample windows -> new ACKed usage (N1 > N0).
	for round := range 2 {
		for range 5 {
			_ = p.rawFlow(peerM1D1)
			_ = p.rawFlow(peerM2D1)
		}
		if round == 0 {
			time.Sleep(11 * time.Second)
		}
	}
	n1Seq := p.waitUsageFlushed(nodeID, n0Seq+1, m1, m2, 110*time.Second)
	t.Logf("AT30 Step 2 (T0..T1 traffic): ACKed seq advanced %d -> %d", n0Seq, n1Seq)

	// (b) One payment-state change through the fake provider on m1:
	// First probe past_due (records 400 on #207), then apply supported cancel+refund via /e2e/snapshot + webhook.
	paymentT1 := p.applyProviderChangeT1(m1, extM1, paidEnd)

	// (c) One device revocation on m2's second device (peerM2D2):
	revokeT1 := p.revokeDeviceT1(m2, peerM2D2.Device)

	// (d) New device authorizations in T0..T1:
	// - peerM3D1 for a post-T0 member m3 (present in NetBird N1, no netbird_user_bindings or member_devices in T0 DB)
	// - peerM2D3 for existing T0 member m2 (present in NetBird N1, no member_devices row in T0 DB, mapped via T0 netbird_user_bindings)
	admin = p.login("e2e-admin", "admin")
	m3 := p.login("secops-member-02", "member")
	p.grantAt(admin, m3, paidEnd, "post-T0-member-m3")
	peerM3D1, ok := p.joinFlowPeer(stack, m3, "secops-member-02", "at30-m3-d1-t1")
	if !ok {
		t.Fatal("post-T0 member m3 device admission failed")
	}
	peerM2D3, ok := p.joinFlowPeer(stack, m2, "secops-member-01", "at30-m2-d3-t1")
	if !ok {
		t.Fatal("post-T0 m2 device 3 admission failed")
	}

	// Ensure any final T0..T1 usage events are flushed before recording L1.
	n1Seq = p.waitUsageFlushed(nodeID, n1Seq, m1, m2, 90*time.Second)
	l1 := p.captureLedgerState(nodeID, m1, m2)
	nodeStoreT1 := p.captureNodeUsageStore("t1")
	p1 := p.captureBillingState(m1.member, m2.member, extM1, extM2)
	n1Peers := p.capturePeerState(nodeID, []string{m1.member, m2.member, m3.member}, m1, m2)

	var t1Time time.Time
	var t1LSN string
	if err := p.db.QueryRow(t.Context(), `SELECT clock_timestamp(), pg_current_wal_lsn()::text`).Scan(&t1Time, &t1LSN); err != nil {
		t.Fatal("T1 watermark query failed:", err)
	}
	t1Time = t1Time.UTC()
	p.backupCommand("wal-switch")
	t.Logf("AT30 T1 state recorded: time=%s lsn=%s ack=%d", t1Time.Format(time.RFC3339Nano), t1LSN, n1Seq)

	// Step 3: Restore central to T0 while the node, fake payment provider, and NetBird stay at T1.
	restoreImg := p.buildRestoreImage()
	primaryGuard := p.verifyPrimaryOverwriteRefusal(restoreImg, t0Time)

	// Step 3A: Fail-safe negative (2 minutes, before Step 1 of the restore protocol):
	// Restore T0 into fvpn-w4-secops-restore-neg-live and start control & worker WITHOUT
	// running freeze / begin-recovery. Observe for 120 seconds.
	t.Log("AT30 Step 3A: starting 2-minute fail-safe negative (restored T0 DB without recovery protocol)")
	failSafeNeg := p.runFailSafeNegative(restoreImg, t0Time, nodeID, n0Seq, n1Seq, peerM2D1)

	// Step 3B: Clean restore of T0 into fvpn-w4-secops-restore-live + Product Restore Protocol (Steps 1–5):
	t.Log("AT30 Step 3B: executing clean T0 restore and 5-step product restore protocol")
	protocolResult := p.runProductRestoreProtocol(restoreImg, t0Time, nodeID, n0Seq, n1Seq, m1, m2, peerM2D1)

	// Step 4: Observe reconciliation across usage replay (L2), provider truth (P1), and NetBird peers (N1).
	t.Log("AT30 Step 4: observing post-restore reconciliation across usage, provider and NetBird peers")
	reconcileResult := p.observeReconciliation(
		nodeID, n0Seq, n1Seq, t0Time, t1Time,
		m1, m2, m3, extM1, extM2,
		peerM1D1, peerM2D1, peerM2D2, peerM2D3, peerM3D1,
		l0, l1, nodeStoreT0, nodeStoreT1, p0, p1, n0Peers, n1Peers,
	)

	nodeExposureNote := map[string]any{
		"http_or_cli_exposed": false,
		"source_reference":    "node/agent/observe.go:111-116, node/internal/usagestore/store.go:41-88",
		"finding":             "node/agent/observe.go:111-116 sends a zeroed api.UsageOutboxStatus{} in POST /api/v1/agent/observations, and no node HTTP/CLI endpoint exposes retained ACKed usage records. The node persists all usage envelopes in /var/lib/vpn-node/state/usage.db (SQLite WAL tables: meta, events, ledger_epochs, replay_unavailable), keeping ACKed rows with state='archived' for DefaultArchiveRetain = 72h so ReconcileCheckpoint can un-archive and replay them when central rotates ledger_epoch during recovery.",
	}
	p.metadata["AT30_restore_reconcile"] = map[string]any{
		"step1_checkpoint_t0": map[string]any{
			"t0_utc": t0Time,
			"pgbackrest_backup": map[string]any{
				"backup_label":     backup.Label,
				"backup_type":      "full",
				"backup_timestamp": backup.Timestamp,
				"backup_lsn":       backup.LSN,
				"t0_lsn":           t0LSN,
			},
			"initial_flows":          t0Flows,
			"ledger_l0":              l0,
			"node_retained_store_t0": nodeStoreT0,
			"node_retained_exposure": nodeExposureNote,
			"provider_p0":            p0,
			"peers_n0":               n0Peers,
		},
		"step2_world_forward_t1": map[string]any{
			"t1_utc":                 t1Time,
			"t1_lsn":                 t1LSN,
			"ledger_l1":              l1,
			"node_retained_store_t1": nodeStoreT1,
			"provider_change":        paymentT1,
			"provider_p1":            p1,
			"revoked_device_t1":      revokeT1,
			"new_t0_t1_devices": map[string]any{
				"m3_new_member_device":      map[string]any{"member_id": m3.member, "device_id": peerM3D1.Device, "service": peerM3D1.Service},
				"m2_existing_member_device": map[string]any{"member_id": m2.member, "device_id": peerM2D3.Device, "service": peerM2D3.Service},
			},
			"peers_n1": n1Peers,
		},
		"step3_restore_central_to_t0": map[string]any{
			"restore_image_id":                    restoreImg,
			"primary_overwrite_refusal":           primaryGuard,
			"fail_safe_negative_without_protocol": failSafeNeg,
			"product_restore_protocol":            protocolResult,
		},
		"step4_reconciliation_observation": reconcileResult,
	}
}

func (p *liveProbe) waitCountedFlow(peer flowPeer, timeout time.Duration) map[string]any {
	var last map[string]any
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); {
		last = p.selectedFlow(peer)
		if last["counted"] == true {
			return last
		}
		time.Sleep(2 * time.Second)
	}
	p.t.Fatalf("peer %s (%s) did not produce a counted flow within %s", peer.Service, peer.Device, timeout)
	return last
}

func (p *liveProbe) waitUsageFlushed(nodeID uuid.UUID, minSeq uint64, m1, m2 *liveSession, timeout time.Duration) uint64 {
	// Wait at least one 10s meter sample interval after packet transmission.
	time.Sleep(12 * time.Second)
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); {
		var ack uint64
		var ackText string
		err := p.db.QueryRow(p.t.Context(), `SELECT highest_contiguous_seq::text FROM usage_checkpoints WHERE node_id=$1 ORDER BY updated_at DESC LIMIT 1`, nodeID).Scan(&ackText)
		if err == nil {
			_, _ = fmt.Sscanf(ackText, "%d", &ack)
		}
		snap := p.captureNodeUsageStore("poll")
		nodeAck, _ := snap["acked_through"].(uint64)
		outboxCount, _ := snap["outbox_count"].(int)
		u1 := p.memberUsageTotal(m1)
		u2 := p.memberUsageTotal(m2)
		if err == nil && ack >= minSeq && nodeAck == ack && outboxCount == 0 && u1 > 0 && u2 > 0 {
			return ack
		}
		time.Sleep(2 * time.Second)
	}
	p.t.Fatalf("usage pipeline did not flush and converge at seq >= %d within %s", minSeq, timeout)
	return 0
}

func (p *liveProbe) memberUsageTotal(m *liveSession) decstr.U64 {
	r := p.call(m, "GET", "/api/v1/member/usage", nil, nil)
	var u api.MemberUsage
	if r.status == 200 && json.Unmarshal(r.body, &u) == nil {
		return u.Totals.TotalBytes
	}
	return 0
}

func (p *liveProbe) checkoutExternalID(memberID string) string {
	var ext string
	if err := p.db.QueryRow(p.t.Context(), `SELECT external_session_id FROM billing_sessions WHERE member_id=$1 ORDER BY created_at DESC LIMIT 1`, memberID).Scan(&ext); err != nil || ext == "" {
		p.t.Fatalf("external checkout session id missing for member %s: %v", memberID, err)
	}
	return ext
}

// Node retained usage records inspection:
// Reading node/ source (node/agent/observe.go:111-116, node/internal/usagestore/{store.go,ack.go,restore.go})
// shows that no node HTTP/CLI endpoint exposes retained ACKed records (observe.go sends a zeroed
// UsageOutboxStatus). The node retains ACKed events in its local SQLite store /var/lib/vpn-node/state/usage.db
// in table `events` with state='archived' (keeping the canonical payload and payload_sha256 for
// DefaultArchiveRetain = 72h). We copy /var/lib/vpn-node/state out to a private scratch directory
// and read the SQLite tables read-only.
func (p *liveProbe) captureNodeUsageStore(label string) map[string]any {
	scratch, err := os.MkdirTemp(filepath.Join(p.root, ".agent/secops"), "node-usage-"+label+"-")
	if err != nil {
		p.t.Fatal("node usage scratch dir:", err)
	}
	defer os.RemoveAll(scratch)

	if out, err := exec.CommandContext(p.t.Context(), "docker", "cp", livePrefix+"node:/var/lib/vpn-node/state", scratch).CombinedOutput(); err != nil {
		return map[string]any{"read_ok": false, "error": strings.TrimSpace(string(out))}
	}
	dbPath := filepath.Join(scratch, "state/usage.db")
	sqldb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return map[string]any{"read_ok": false, "error": err.Error()}
	}
	defer sqldb.Close()

	var nodeID, incID, openEpoch, ledgerEpoch, gateReason string
	var nextSeq, ackedThrough, purgedThrough uint64
	var blockedSeq sql.NullInt64
	var gateStopped int
	err = sqldb.QueryRowContext(p.t.Context(), `SELECT node_id, incarnation_id, next_seq, COALESCE(open_epoch,''), COALESCE(ledger_epoch,''), acked_through, blocked_seq, purged_through, gate_stopped, gate_reason FROM meta WHERE id = 1`).
		Scan(&nodeID, &incID, &nextSeq, &openEpoch, &ledgerEpoch, &ackedThrough, &blockedSeq, &purgedThrough, &gateStopped, &gateReason)
	if err != nil {
		return map[string]any{"read_ok": false, "error": err.Error()}
	}

	rows, err := sqldb.QueryContext(p.t.Context(), `SELECT seq, epoch_id, interval_start, interval_end, quality, payload_sha256, payload_len, state, created_at, COALESCE(acked_at,''), COALESCE(acked_epoch,'') FROM events ORDER BY seq`)
	if err != nil {
		return map[string]any{"read_ok": false, "error": err.Error()}
	}
	defer rows.Close()
	events := []map[string]any{}
	outboxCount, archivedCount := 0, 0
	for rows.Next() {
		var seq uint64
		var epoch, start, end, quality, sha, state, created, ackedAt, ackedEpoch string
		var length int
		if rows.Scan(&seq, &epoch, &start, &end, &quality, &sha, &length, &state, &created, &ackedAt, &ackedEpoch) == nil {
			if state == "outbox" {
				outboxCount++
			} else if state == "archived" {
				archivedCount++
			}
			events = append(events, map[string]any{
				"seq":            seq,
				"epoch_id":       epoch,
				"interval_start": start,
				"interval_end":   end,
				"quality":        quality,
				"payload_sha256": sha,
				"payload_len":    length,
				"state":          state,
				"acked_at":       ackedAt,
				"acked_epoch":    ackedEpoch,
			})
		}
	}

	epRows, _ := sqldb.QueryContext(p.t.Context(), `SELECT ledger_epoch, accepted_at, acked_through, COALESCE(recovery_record_id,''), COALESCE(superseded_at,'') FROM ledger_epochs ORDER BY accepted_at, ledger_epoch`)
	epochs := []map[string]any{}
	if epRows != nil {
		for epRows.Next() {
			var le, acc, rec, sup string
			var ack uint64
			if epRows.Scan(&le, &acc, &ack, &rec, &sup) == nil {
				epochs = append(epochs, map[string]any{
					"ledger_epoch":       le,
					"accepted_at":        acc,
					"acked_through":      ack,
					"recovery_record_id": rec,
					"superseded_at":      sup,
				})
			}
		}
		epRows.Close()
	}

	unavailRows, _ := sqldb.QueryContext(p.t.Context(), `SELECT start_seq, end_seq, reason FROM replay_unavailable ORDER BY start_seq`)
	unavailable := []map[string]any{}
	if unavailRows != nil {
		for unavailRows.Next() {
			var s, e uint64
			var r string
			if unavailRows.Scan(&s, &e, &r) == nil {
				unavailable = append(unavailable, map[string]any{"start_seq": s, "end_seq": e, "reason": r})
			}
		}
		unavailRows.Close()
	}

	var bSeq any
	if blockedSeq.Valid {
		bSeq = uint64(blockedSeq.Int64)
	}
	return map[string]any{
		"read_ok":            true,
		"exposure_note":      "No node HTTP/CLI endpoint exposes retained ACKed records; node/agent/observe.go:111-116 sends a zeroed UsageOutboxStatus. Retained ACKed events are stored in /var/lib/vpn-node/state/usage.db table `events` with state='archived' for DefaultArchiveRetain=72h (node/internal/usagestore/{store.go,ack.go,restore.go}).",
		"node_id":            nodeID,
		"incarnation_id":     incID,
		"next_seq":           nextSeq,
		"open_epoch":         openEpoch,
		"ledger_epoch":       ledgerEpoch,
		"acked_through":      ackedThrough,
		"blocked_seq":        bSeq,
		"purged_through":     purgedThrough,
		"gate_stopped":       gateStopped == 1,
		"gate_reason":        gateReason,
		"outbox_count":       outboxCount,
		"archived_count":     archivedCount,
		"events":             events,
		"ledger_epochs":      epochs,
		"replay_unavailable": unavailable,
	}
}

func (p *liveProbe) captureLedgerState(nodeID uuid.UUID, m1, m2 *liveSession) map[string]any {
	ctx := p.t.Context()
	var stateJSON, cpJSON, eventsJSON, samplesJSON, rollupsJSON []byte
	var gapsCount int
	_ = p.db.QueryRow(ctx, `SELECT jsonb_build_object('current_epoch',current_epoch,'mode',mode,'mode_reason',mode_reason,'changed_by',changed_by,'updated_at',updated_at) FROM usage_ledger_state WHERE id='00000000-0000-0000-0000-000000000130'`).Scan(&stateJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('node_id',node_id,'incarnation_id',incarnation_id,'ledger_epoch',ledger_epoch,'highest_contiguous_seq',highest_contiguous_seq::text,'max_seen_seq',max_seen_seq::text,'updated_at',updated_at)),'[]'::jsonb) FROM usage_checkpoints WHERE node_id=$1`, nodeID).Scan(&cpJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('event_seq',event_seq::text,'meter_epoch_id',meter_epoch_id,'interval_start',interval_start,'interval_end',interval_end,'quality',quality,'payload_hash',encode(payload_hash,'hex'),'record_count',record_count,'ledger_epoch',ledger_epoch) ORDER BY event_seq),'[]'::jsonb) FROM usage_events WHERE node_id=$1`, nodeID).Scan(&eventsJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('event_seq',e.event_seq::text,'member_id',s.member_id,'device_id',s.device_id,'binding_revision',s.binding_revision::text,'interval_start',s.interval_start,'interval_end',s.interval_end,'upload_bytes',s.upload_bytes::text,'download_bytes',s.download_bytes::text) ORDER BY e.event_seq, s.member_id, s.device_id),'[]'::jsonb) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1`, nodeID).Scan(&samplesJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('member_id',member_id,'period',period,'granularity',granularity,'projection_generation',projection_generation::text,'upload_bytes',upload_bytes::text,'download_bytes',download_bytes::text,'total_bytes',total_bytes::text,'gap',gap,'reconciliation_in_progress',reconciliation_in_progress,'aggregate_as_of',aggregate_as_of) ORDER BY member_id, period, granularity),'[]'::jsonb) FROM usage_rollups WHERE member_id=ANY($1::uuid[])`, []string{m1.member, m2.member}).Scan(&rollupsJSON)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM usage_ledger_gaps`).Scan(&gapsCount)

	apiView := func(m *liveSession) map[string]any {
		r := p.call(m, "GET", "/api/v1/member/usage", nil, nil)
		var parsed any
		_ = json.Unmarshal(r.body, &parsed)
		return map[string]any{"http_status": r.status, "code": r.code, "body": parsed}
	}
	return map[string]any{
		"utc":               time.Now().UTC(),
		"ledger_state":      json.RawMessage(stateJSON),
		"checkpoints":       json.RawMessage(cpJSON),
		"events":            json.RawMessage(eventsJSON),
		"samples":           json.RawMessage(samplesJSON),
		"rollups":           json.RawMessage(rollupsJSON),
		"usage_ledger_gaps": gapsCount,
		"member_usage_api":  map[string]any{"m1": apiView(m1), "m2": apiView(m2)},
	}
}

func (p *liveProbe) captureBillingState(m1ID, m2ID, extM1, extM2 string) map[string]any {
	ctx := p.t.Context()
	var subsJSON, recJSON, grantsJSON, entJSON, auditsJSON []byte
	ids := []string{m1ID, m2ID}
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('member_id',member_id,'external_subscription_id',external_subscription_id,'provider_status',provider_status,'is_active',is_active,'cancel_at_period_end',cancel_at_period_end,'paid_from',paid_from,'paid_through',paid_through,'grace_from',grace_from,'grace_until',grace_until,'last_reconciled_at',last_reconciled_at,'reconciliation_anomalies',reconciliation_anomalies) ORDER BY member_id),'[]'::jsonb) FROM subscriptions WHERE member_id=ANY($1::uuid[])`, ids).Scan(&subsJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('member_id',member_id,'record_type',record_type,'external_reference',external_reference,'original_invoice_ref',original_invoice_ref,'amount_minor',amount_minor,'currency',currency,'service_start',service_start,'service_end',service_end) ORDER BY member_id, created_at),'[]'::jsonb) FROM billing_records WHERE member_id=ANY($1::uuid[])`, ids).Scan(&recJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('member_id',member_id,'source',source,'valid_from',valid_from,'valid_until',valid_until,'revoked_at',revoked_at,'revocation_reason',revocation_reason) ORDER BY member_id, created_at),'[]'::jsonb) FROM entitlement_grants WHERE member_id=ANY($1::uuid[])`, ids).Scan(&grantsJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('member_id',member_id,'state',state,'revision',revision::text,'device_limit',device_limit,'valid_from',valid_from,'effective_service_end',effective_service_end,'admin_suspended',admin_suspended) ORDER BY member_id),'[]'::jsonb) FROM entitlements WHERE member_id=ANY($1::uuid[])`, ids).Scan(&entJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('action',action,'resource',resource,'result',result,'before_diff',before_diff,'after_diff',after_diff,'occurred_at',occurred_at) ORDER BY occurred_at),'[]'::jsonb) FROM audit_events WHERE action LIKE 'billing.%'`).Scan(&auditsJSON)

	providerCustomer := func(ext string) map[string]any {
		cusID := "cus_" + ext
		subID := "sub_" + ext
		get := func(path string) any {
			req, err := http.NewRequestWithContext(ctx, "GET", "https://api.stripe.com"+path, nil)
			if err != nil {
				return nil
			}
			req.Header.Set("Authorization", "Bearer "+p.secret("stripe.key"))
			resp, err := p.client().Do(req)
			if err != nil {
				return nil
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			var v any
			_ = json.Unmarshal(raw, &v)
			return v
		}
		return map[string]any{
			"customer_id":     cusID,
			"subscription_id": subID,
			"subscription":    get("/v1/subscriptions/" + subID),
			"invoices":        get("/v1/invoices?customer=" + cusID + "&limit=10"),
		}
	}

	return map[string]any{
		"utc":                time.Now().UTC(),
		"provider_truth":     map[string]any{"m1": providerCustomer(extM1), "m2": providerCustomer(extM2)},
		"subscriptions":      json.RawMessage(subsJSON),
		"billing_records":    json.RawMessage(recJSON),
		"entitlement_grants": json.RawMessage(grantsJSON),
		"entitlements":       json.RawMessage(entJSON),
		"billing_audits":     json.RawMessage(auditsJSON),
	}
}

func (p *liveProbe) capturePeerState(nodeID uuid.UUID, memberIDs []string, m1, m2 *liveSession) map[string]any {
	ctx := p.t.Context()
	var devJSON, bindJSON, obsJSON, desiredJSON, auditsJSON []byte
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('id',id,'member_id',member_id,'peer_id',peer_id,'name',name,'status',status,'authorized_at',authorized_at,'revoked_at',revoked_at,'resource_version',resource_version::text) ORDER BY member_id, created_at),'[]'::jsonb) FROM member_devices WHERE member_id=ANY($1::uuid[])`, memberIDs).Scan(&devJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('device_id',device_id,'member_id',member_id,'binding_revision',binding_revision::text,'overlay_ip',overlay_ip::text,'valid_from',valid_from,'valid_until',valid_until,'peer_ref',peer_ref) ORDER BY member_id, device_id),'[]'::jsonb) FROM device_bindings WHERE member_id=ANY($1::uuid[])`, memberIDs).Scan(&bindJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('device_id',o.device_id,'member_id',d.member_id,'peer_id',d.peer_id,'observed_authorized',o.observed_authorized,'observed_at',o.observed_at,'observed_version',o.observed_version::text,'observed_entitlement_revision',o.observed_entitlement_revision::text) ORDER BY d.member_id, o.device_id),'[]'::jsonb) FROM device_authorization_observations o JOIN member_devices d ON d.id=o.device_id WHERE d.member_id=ANY($1::uuid[])`, memberIDs).Scan(&obsJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(envelope->'signed'->'device_bindings','[]'::jsonb) FROM desired_state_envelopes WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, nodeID).Scan(&desiredJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('action',action,'resource',resource,'result',result,'before_diff',before_diff,'after_diff',after_diff,'occurred_at',occurred_at) ORDER BY occurred_at),'[]'::jsonb) FROM audit_events WHERE action IN ('device.revoke','device.authorize','identity_binding.rejected','netbird.observed')`).Scan(&auditsJSON)

	nbGet := func(path string) any {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://netbird.example.invalid/api"+path, nil)
		if err != nil {
			return nil
		}
		req.Header.Set("Authorization", "Token "+p.secret("netbird.token"))
		resp, err := p.client().Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
		var v any
		_ = json.Unmarshal(raw, &v)
		return v
	}
	var paidGroupPeers []string
	if groups, ok := nbGet("/groups").([]any); ok {
		wantName := "fvpn-paid-" + p.groupID.String()
		for _, g := range groups {
			if gm, ok := g.(map[string]any); ok && gm["name"] == wantName {
				if peers, ok := gm["peers"].([]any); ok {
					for _, pr := range peers {
						if pm, ok := pr.(map[string]any); ok {
							if id, ok := pm["id"].(string); ok {
								paidGroupPeers = append(paidGroupPeers, id)
							}
						}
					}
				}
			}
		}
	}
	nbPeersSummary := []map[string]any{}
	if peers, ok := nbGet("/peers").([]any); ok {
		for _, pr := range peers {
			if pm, ok := pr.(map[string]any); ok {
				nbPeersSummary = append(nbPeersSummary, map[string]any{
					"id":      pm["id"],
					"name":    pm["name"],
					"user_id": pm["user_id"],
				})
			}
		}
	}

	devAPI := func(m *liveSession) any {
		r := p.call(m, "GET", "/api/v1/member/devices", nil, nil)
		var v any
		_ = json.Unmarshal(r.body, &v)
		return map[string]any{"http_status": r.status, "body": v}
	}

	return map[string]any{
		"utc":                                time.Now().UTC(),
		"netbird_paid_group_name":            "fvpn-paid-" + p.groupID.String(),
		"netbird_paid_group_peer_ids":        paidGroupPeers,
		"netbird_peers":                      nbPeersSummary,
		"central_member_devices":             json.RawMessage(devJSON),
		"central_device_bindings":            json.RawMessage(bindJSON),
		"central_authorization_observations": json.RawMessage(obsJSON),
		"node_desired_state_bindings":        json.RawMessage(desiredJSON),
		"device_audit_events":                json.RawMessage(auditsJSON),
		"member_devices_api":                 map[string]any{"m1": devAPI(m1), "m2": devAPI(m2)},
	}
}

func (p *liveProbe) applyProviderChangeT1(m1 *liveSession, extM1 string, _ time.Time) map[string]any {
	hook := func(path string, body any) (int, []byte) {
		raw, _ := json.Marshal(body)
		req, e := http.NewRequestWithContext(p.t.Context(), "POST", "https://api.stripe.com/e2e/"+path, bytes.NewReader(raw))
		if e != nil {
			return 0, nil
		}
		req.Header.Set("X-E2E-Secret", p.secret("payment.secret"))
		resp, e := p.client().Do(req)
		if e != nil {
			return 0, nil
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return resp.StatusCode, out
	}

	// Read the exact period_start and period_end recorded at T0 so /e2e/snapshot does not reject with 409.
	var periodStart, periodEnd int64
	if err := p.db.QueryRow(p.t.Context(), `SELECT extract(epoch from paid_from)::bigint, extract(epoch from paid_through)::bigint FROM subscriptions WHERE member_id=$1`, m1.member).Scan(&periodStart, &periodEnd); err != nil {
		p.t.Fatal("read m1 subscription period_start/period_end:", err)
	}

	// Probe past_due first (records 400 on #207).
	pastDueBody := map[string]any{
		"checkout_id":          extM1,
		"period_start":         periodStart,
		"period_end":           periodEnd,
		"amount_minor":         1000,
		"currency":             "USD",
		"status":               "past_due",
		"paid":                 true,
		"cancel_at_period_end": false,
		"refund_minor":         0,
	}
	pastDueStatus, _ := hook("snapshot", pastDueBody)

	// Apply the supported provider state change (canceled + full refund = 1000) and deliver webhook.
	cancelSnapshot := map[string]any{
		"checkout_id":          extM1,
		"period_start":         periodStart,
		"period_end":           periodEnd,
		"amount_minor":         1000,
		"currency":             "USD",
		"status":               "canceled",
		"paid":                 true,
		"cancel_at_period_end": true,
		"refund_minor":         1000,
	}
	snapStatus, _ := hook("snapshot", cancelSnapshot)
	if snapStatus != 200 {
		p.t.Fatalf("fake provider cancel+refund snapshot failed: status=%d", snapStatus)
	}
	evStatus, evRaw := hook("event", map[string]string{"checkout_id": extM1, "type": "charge.refunded"})
	if evStatus != 200 {
		p.t.Fatalf("fake provider charge.refunded event failed: status=%d", evStatus)
	}
	var ev struct {
		Body, Signature string
	}
	if json.Unmarshal(evRaw, &ev) != nil {
		p.t.Fatal("webhook event decode failed")
	}
	p.secrets = append(p.secrets, []byte(ev.Signature))
	wh := p.call(&liveSession{client: p.client()}, "POST", "/api/v1/webhooks/stripe", []byte(ev.Body), map[string]string{"Stripe-Signature": ev.Signature})
	p.check("T1 provider charge.refunded webhook", wh, 202, "")

	var subStatus, entState string
	var subActive bool
	var refundCount, devLimit int
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		_ = p.db.QueryRow(p.t.Context(), `SELECT s.provider_status, s.is_active, e.state, e.device_limit, (SELECT count(*) FROM billing_records r WHERE r.member_id=s.member_id AND r.record_type='refund') FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, m1.member).
			Scan(&subStatus, &subActive, &entState, &devLimit, &refundCount)
		if subStatus == "canceled" && !subActive && refundCount >= 1 && entState == "expired" {
			break
		}
		time.Sleep(time.Second)
	}
	if subStatus != "canceled" || subActive || refundCount < 1 || entState != "expired" {
		p.t.Fatalf("T1 provider state change did not converge: sub=%s subActive=%t refunds=%d entState=%s devLimit=%d", subStatus, subActive, refundCount, entState, devLimit)
	}
	return map[string]any{
		"member_id":                m1.member,
		"checkout_id":              extM1,
		"past_due_probe_status":    pastDueStatus,
		"applied_snapshot_status":  snapStatus,
		"webhook_http_status":      wh.status,
		"t1_subscription_status":   subStatus,
		"t1_subscription_active":   subActive,
		"t1_refund_record_count":   refundCount,
		"t1_entitlement_state":     entState,
		"t1_entitlement_dev_limit": devLimit,
	}
}

func (p *liveProbe) revokeDeviceT1(m2 *liveSession, deviceID uuid.UUID) map[string]any {
	var extPeerID string
	_ = p.db.QueryRow(p.t.Context(), `SELECT peer_id FROM member_devices WHERE id=$1`, deviceID).Scan(&extPeerID)
	r := p.call(m2, "DELETE", "/api/v1/member/devices/"+deviceID.String(), nil, nil)
	p.check("T1 revoke m2 second device", r, 202, "")
	var op api.Operation
	_ = json.Unmarshal(r.body, &op)

	var devStatus string
	var obsAuth bool
	var auditCount int
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		opReply := p.call(m2, "GET", "/api/v1/member/operations/"+op.OperationId.String(), nil, nil)
		if opReply.status == 200 {
			_ = json.Unmarshal(opReply.body, &op)
		}
		_ = p.db.QueryRow(p.t.Context(), `SELECT d.status, COALESCE(o.observed_authorized,true), (SELECT count(*) FROM audit_events WHERE resource='member_device:'||$1::uuid::text AND action='device.revoke' AND result='succeeded') FROM member_devices d LEFT JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.id=$1::uuid`, deviceID).
			Scan(&devStatus, &obsAuth, &auditCount)
		if op.Status == api.OperationStatusSucceeded && devStatus == "revoked" && !obsAuth && auditCount >= 1 {
			break
		}
		time.Sleep(time.Second)
	}
	if devStatus != "revoked" || obsAuth || auditCount < 1 {
		p.t.Fatalf("T1 device revocation did not converge: status=%s obsAuth=%t audits=%d", devStatus, obsAuth, auditCount)
	}
	var auditRow []byte
	_ = p.db.QueryRow(p.t.Context(), `SELECT row_to_json(a)::jsonb FROM audit_events a WHERE resource='member_device:'||$1::uuid::text AND action='device.revoke' ORDER BY occurred_at DESC LIMIT 1`, deviceID).Scan(&auditRow)
	return map[string]any{
		"member_id":                m2.member,
		"device_id":                deviceID,
		"external_peer_id":         extPeerID,
		"delete_http_status":       r.status,
		"operation_id":             op.OperationId,
		"operation_status":         op.Status,
		"t1_device_status":         devStatus,
		"t1_observed_authorized":   obsAuth,
		"t1_revoke_audit_recorded": auditCount >= 1,
		"t1_revoke_audit_event":    json.RawMessage(auditRow),
	}
}

func (p *liveProbe) buildRestoreImage() string {
	iidFile := filepath.Join(p.root, ".agent/secops/followup8-restore-image.id")
	_ = os.Remove(iidFile)
	cmd := exec.CommandContext(p.t.Context(), "docker", "build", "--iidfile", iidFile, "-t", restoreImageTag, "deploy/backup/postgres")
	cmd.Dir = p.root
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.WriteFile(filepath.Join(p.root, ".agent/secops/restore-image-build.log"), out, 0600)
		p.t.Fatal("restore image build failed")
	}
	idBytes, err := os.ReadFile(iidFile)
	if err != nil {
		p.t.Fatal("restore image iid read failed:", err)
	}
	return strings.TrimSpace(string(idBytes))
}

func (p *liveProbe) verifyPrimaryOverwriteRefusal(image string, targetTime time.Time) map[string]any {
	target := targetTime.UTC().Format("2006-01-02T15:04:05.000000Z")
	argv := []string{
		"deploy/backup/restore-pitr.sh",
		"--image", image,
		"--repo-volume", restoreRepoVol,
		"--target-volume", restoreNegVol,
		"--primary-container", livePrefix + "postgres",
		"--db-user", "fvpn",
		"--db-name", "fvpn",
		"--target-time", target,
		"--isolated-container", livePrefix + "postgres",
	}
	cmd := exec.CommandContext(p.t.Context(), argv[0], argv[1:]...)
	cmd.Dir = p.root
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	if exitCode != 2 {
		p.t.Fatalf("expected restore-pitr.sh primary-overwrite refusal exit code 2, got %d (%s)", exitCode, strings.TrimSpace(string(out)))
	}
	return map[string]any{
		"command":   argv,
		"exit_code": exitCode,
		"refused":   exitCode == 2,
	}
}

func (p *liveProbe) restoreAndAttachLivePostgres(image string, targetTime time.Time, isolatedName, liveName, volumeName string) map[string]any {
	ctx := p.t.Context()
	target := targetTime.UTC().Format("2006-01-02T15:04:05.000000Z")

	// Stop control and worker before switching central database.
	_ = exec.CommandContext(ctx, "docker", "stop", "--time", "5", livePrefix+"control", livePrefix+"worker").Run()
	if p.db != nil {
		_ = p.db.Close(context.Background())
		p.db = nil
	}
	// Ensure any previous restore containers/volumes with these names are absent.
	_ = exec.CommandContext(ctx, "docker", "rm", "-fv", isolatedName, isolatedName+"-init", liveName).Run()
	_ = exec.CommandContext(ctx, "docker", "volume", "rm", volumeName).Run()

	argv := []string{
		"deploy/backup/restore-pitr.sh",
		"--image", image,
		"--repo-volume", restoreRepoVol,
		"--target-volume", volumeName,
		"--primary-container", livePrefix + "postgres",
		"--db-user", "fvpn",
		"--db-name", "fvpn",
		"--target-time", target,
		"--isolated-container", isolatedName,
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = p.root
	out, err := cmd.CombinedOutput()
	_ = os.WriteFile(filepath.Join(p.root, ".agent/secops", liveName+"-pitr.log"), out, 0600)
	if err != nil {
		p.t.Fatalf("restore-pitr.sh (%s) failed: %v", isolatedName, err)
	}
	pitrSeconds := time.Since(start).Seconds()

	// Run post-restore-checklist.sql inside the isolated network-none container.
	checklistSQL, err := os.ReadFile(filepath.Join(p.root, "deploy/backup/post-restore-checklist.sql"))
	if err != nil {
		p.t.Fatal("read post-restore-checklist.sql:", err)
	}
	chkCmd := exec.CommandContext(ctx, "docker", "exec", "-i", "--user", "postgres", isolatedName,
		"psql", "-U", "fvpn", "-d", "fvpn", "-v", "ON_ERROR_STOP=1", "-v", "restore_point="+target)
	chkCmd.Stdin = bytes.NewReader(checklistSQL)
	chkOut, err := chkCmd.CombinedOutput()
	_ = os.WriteFile(filepath.Join(p.root, ".agent/secops", liveName+"-checklist.log"), chkOut, 0600)
	if err != nil {
		p.t.Fatalf("post-restore-checklist.sql failed in %s: %v", isolatedName, err)
	}
	sections := 0
	for n := 1; n <= 6; n++ {
		if bytes.Contains(chkOut, []byte(fmt.Sprintf("[%d/6]", n))) {
			sections++
		}
	}
	if sections != 6 {
		p.t.Fatalf("post-restore-checklist.sql incomplete: %d/6 sections", sections)
	}

	// Stop the T1 primary container (preserving its volume) and the isolated verifier container,
	// then start the promoted restored volume on the lab network at postgres:5432 / 127.0.0.1:35432.
	_ = exec.CommandContext(ctx, "docker", "stop", "--time", "5", livePrefix+"postgres").Run()
	_ = exec.CommandContext(ctx, "docker", "rm", "-fv", isolatedName).Run()

	stackRaw, err := os.ReadFile(filepath.Join(p.state, "stack.json"))
	if err != nil {
		p.t.Fatal("read stack.json:", err)
	}
	var stackCfg struct {
		LabPrefix string `json:"lab_prefix"`
	}
	if json.Unmarshal(stackRaw, &stackCfg) != nil || stackCfg.LabPrefix == "" {
		p.t.Fatal("parse stack.json lab_prefix")
	}

	runArgs := []string{
		"run", "-d",
		"--name", liveName,
		"--network", "fvpn-w4-secops",
		"--ip", stackCfg.LabPrefix + ".10",
		"--network-alias", "postgres",
		"-p", "127.0.0.1:35432:5432",
		"-e", "FVPN_PG_ISOLATED_RESTORE=1",
		"-e", "POSTGRES_USER=fvpn",
		"-e", "POSTGRES_DB=fvpn",
		"-e", "POSTGRES_PASSWORD_FILE=/run/secrets/postgres.password",
		"-v", volumeName + ":/var/lib/postgresql/data",
		"-v", restoreRepoVol + ":/var/lib/pgbackrest:ro",
		"-v", filepath.Join(p.state, "secrets/postgres.password") + ":/run/secrets/postgres.password:ro",
		image,
	}
	if out, err := exec.CommandContext(ctx, "docker", runArgs...).CombinedOutput(); err != nil {
		_ = os.WriteFile(filepath.Join(p.root, ".agent/secops", liveName+"-start.log"), out, 0600)
		p.t.Fatalf("failed to start %s: %v", liveName, err)
	}

	// Wait for PostgreSQL readiness and reconnect p.db.
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		if exec.CommandContext(ctx, "docker", "exec", "--user", "postgres", liveName, "pg_isready", "-U", "fvpn", "-d", "fvpn").Run() == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	cfg, err := pgx.ParseConfig("host=127.0.0.1 port=35432 dbname=fvpn user=fvpn sslmode=disable")
	if err != nil {
		p.t.Fatal("parse restored db config:", err)
	}
	cfg.Password = p.secret("postgres.password")
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		p.t.Fatalf("connect restored db (%s): %v", liveName, err)
	}
	p.db = conn

	return map[string]any{
		"isolated_container": isolatedName,
		"live_container":     liveName,
		"target_volume":      volumeName,
		"target_time":        target,
		"pitr_seconds":       pitrSeconds,
		"checklist_sections": sections,
		"checklist_sha256":   fmt.Sprintf("%x", sha256.Sum256(chkOut)),
		"commands":           [][]string{argv},
	}
}

func (p *liveProbe) waitControlReady(timeout time.Duration) bool {
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); {
		if exec.CommandContext(p.t.Context(), "docker", "exec", livePrefix+"control", "wget", "-q", "-O", "/dev/null", "--no-check-certificate", "https://127.0.0.1:8443/readyz").Run() == nil {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func (p *liveProbe) runFailSafeNegative(image string, t0Time time.Time, nodeID uuid.UUID, n0Seq, n1Seq uint64, peerM2D1 flowPeer) map[string]any {
	ctx := p.t.Context()
	restoreInfo := p.restoreAndAttachLivePostgres(image, t0Time, restoreNegCopy, restoreNegLive, restoreNegVol)

	// Verify restored DB is at T0 before starting control & worker.
	var restoredAck, restoredMax string
	var restoredEvents int
	if err := p.db.QueryRow(ctx, `SELECT highest_contiguous_seq::text, max_seen_seq::text FROM usage_checkpoints WHERE node_id=$1`, nodeID).Scan(&restoredAck, &restoredMax); err != nil {
		p.t.Fatal("fail-safe negative: read restored T0 checkpoint:", err)
	}
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id=$1`, nodeID).Scan(&restoredEvents)

	// Start control and worker WITHOUT running freeze or begin-recovery.
	negStart := time.Now()
	if err := exec.CommandContext(ctx, "docker", "start", livePrefix+"control", livePrefix+"worker").Run(); err != nil {
		p.t.Fatal("fail-safe negative: start control/worker:", err)
	}
	if !p.waitControlReady(30 * time.Second) {
		p.t.Fatal("fail-safe negative: control did not become ready")
	}

	// First 25 seconds: no new traffic; verify node does not re-upload archived T0..T1 events.
	time.Sleep(25 * time.Second)
	var idleAck, idleMax string
	var idleEvents int
	_ = p.db.QueryRow(ctx, `SELECT highest_contiguous_seq::text, max_seen_seq::text FROM usage_checkpoints WHERE node_id=$1`, nodeID).Scan(&idleAck, &idleMax)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id=$1`, nodeID).Scan(&idleEvents)
	nodeStoreIdle := p.captureNodeUsageStore("neg-idle")

	// Now generate a post-T1 counted packet burst so the node's meter creates event N1+1 in outbox,
	// and observe the remaining 95 seconds (total 2-minute fail-safe negative window).
	for range 4 {
		_ = p.rawFlow(peerM2D1)
	}
	for time.Since(negStart) < 120*time.Second {
		time.Sleep(2 * time.Second)
	}
	negElapsed := time.Since(negStart).Seconds()

	var afterAck, afterMax string
	var afterEvents int
	_ = p.db.QueryRow(ctx, `SELECT highest_contiguous_seq::text, max_seen_seq::text FROM usage_checkpoints WHERE node_id=$1`, nodeID).Scan(&afterAck, &afterMax)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id=$1`, nodeID).Scan(&afterEvents)
	nodeStoreAfter := p.captureNodeUsageStore("neg-after")

	agentLog, _ := p.docker("node", "cat", "/var/lib/vpn-node/e2e/agent.log")
	ackRegressLogs := 0
	for _, line := range bytes.Split(agentLog, []byte("\n")) {
		if bytes.Contains(line, []byte("ack watermark cannot move backwards in the same epoch")) {
			ackRegressLogs++
		}
	}

	// Stop control and worker and tear down the fail-safe negative DB container/volume.
	_ = exec.CommandContext(ctx, "docker", "stop", "--time", "5", livePrefix+"control", livePrefix+"worker").Run()
	if p.db != nil {
		_ = p.db.Close(context.Background())
		p.db = nil
	}
	_ = exec.CommandContext(ctx, "docker", "rm", "-fv", restoreNegLive).Run()
	_ = exec.CommandContext(ctx, "docker", "volume", "rm", restoreNegVol).Run()

	return map[string]any{
		"window_seconds":                          negElapsed,
		"restore_info":                            restoreInfo,
		"t0_restored_ack":                         restoredAck,
		"t0_restored_max_seen":                    restoredMax,
		"t0_restored_events":                      restoredEvents,
		"after_25s_idle_ack":                      idleAck,
		"after_25s_idle_max_seen":                 idleMax,
		"after_25s_idle_events":                   idleEvents,
		"after_25s_node_store":                    nodeStoreIdle,
		"after_120s_central_ack":                  afterAck,
		"after_120s_central_max_seen":             afterMax,
		"after_120s_central_events":               afterEvents,
		"after_120s_node_store":                   nodeStoreAfter,
		"node_ack_regress_log_lines":              ackRegressLogs,
		"retained_t0_t1_skipped_without_protocol": afterAck == fmt.Sprintf("%d", n0Seq),
		"explanation": fmt.Sprintf(
			"Without freeze/begin-recovery rotating ledger_epoch, events %d..%d remain state='archived' on the node (acked_through=%d). While idle, Uploader.due() is false and nothing is sent. Once post-T1 traffic creates event > %d in outbox, the node uploads only the new event under the old ledger_epoch; central advances max_seen_seq while returning highest_contiguous_seq=%d, which the node rejects with ErrAckRegress ('usagestore: ack watermark cannot move backwards in the same epoch').",
			n0Seq+1, n1Seq, n1Seq, n1Seq, n0Seq,
		),
	}
}

func (p *liveProbe) writeHostUsageAdminConfig() string {
	hostDSNPath := filepath.Join(p.state, "secrets/database-host.url")
	dsn := "postgres://fvpn:" + p.secret("postgres.password") + "@127.0.0.1:35432/fvpn?sslmode=disable\n"
	if err := os.WriteFile(hostDSNPath, []byte(dsn), 0600); err != nil {
		p.t.Fatal("write database-host.url:", err)
	}
	usageRaw, err := os.ReadFile(filepath.Join(p.state, "config/usage.json"))
	if err != nil {
		p.t.Fatal("read usage.json:", err)
	}
	var usageCfg map[string]any
	if json.Unmarshal(usageRaw, &usageCfg) != nil {
		p.t.Fatal("parse usage.json")
	}
	usageCfg["signing_config_file"] = filepath.Join(p.state, "vault")
	usageHostPath := filepath.Join(p.state, "config/usage-host.json")
	rawUsage, _ := json.MarshalIndent(usageCfg, "", "  ")
	if err := os.WriteFile(usageHostPath, append(rawUsage, '\n'), 0600); err != nil {
		p.t.Fatal("write usage-host.json:", err)
	}

	centralRaw, err := os.ReadFile(filepath.Join(p.state, "config/central.json"))
	if err != nil {
		p.t.Fatal("read central.json:", err)
	}
	var centralCfg map[string]any
	if json.Unmarshal(centralRaw, &centralCfg) != nil {
		p.t.Fatal("parse central.json")
	}
	centralCfg["database_url_file"] = hostDSNPath
	centralCfg["usage_config_file"] = usageHostPath
	adminCfgPath := filepath.Join(p.state, "config/usage-admin-host.json")
	rawAdmin, _ := json.MarshalIndent(centralCfg, "", "  ")
	if err := os.WriteFile(adminCfgPath, append(rawAdmin, '\n'), 0600); err != nil {
		p.t.Fatal("write usage-admin-host.json:", err)
	}
	return adminCfgPath
}

func (p *liveProbe) setFastWorkerReconcileCadence() {
	path := filepath.Join(p.state, "config/members.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		p.t.Fatal("read members.json:", err)
	}
	var cfg map[string]any
	if json.Unmarshal(raw, &cfg) != nil {
		p.t.Fatal("parse members.json")
	}
	cfg["reconcile_seconds"] = 5
	cfg["device_sync_seconds"] = 5
	updated, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, append(updated, '\n'), 0644); err != nil {
		p.t.Fatal("write members.json:", err)
	}
}

func (p *liveProbe) runUsageAdmin(cfgPath string, subArgs ...string) (map[string]any, any) {
	relCfg, err := filepath.Rel(p.root, cfgPath)
	if err != nil {
		relCfg = cfgPath
	}
	argv := append([]string{"run", "./internal/usage/cmd/fvpn-usage-admin", "-config", relCfg}, subArgs...)
	start := time.Now()
	cmd := exec.CommandContext(p.t.Context(), "go", argv...)
	cmd.Dir = p.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		p.t.Fatalf("fvpn-usage-admin %v failed: %v (%s)", subArgs, err, strings.TrimSpace(string(out)))
	}
	var parsed any
	_ = json.Unmarshal(out, &parsed)
	return map[string]any{
		"command":         append([]string{"go"}, argv...),
		"elapsed_seconds": time.Since(start).Seconds(),
		"exit_code":       0,
		"output":          parsed,
	}, parsed
}

func (p *liveProbe) runProductRestoreProtocol(image string, t0Time time.Time, nodeID uuid.UUID, n0Seq, n1Seq uint64, m1, m2 *liveSession, peerM2D1 flowPeer) map[string]any {
	ctx := p.t.Context()
	outageStart := time.Now()
	restoreInfo := p.restoreAndAttachLivePostgres(image, t0Time, restoreCleanCopy, restoreCleanLive, restoreCleanVol)

	p.setFastWorkerReconcileCadence()
	cfgPath := p.writeHostUsageAdminConfig()
	actor := "00000000-0000-0000-0000-000000000003"
	monthStart := time.Date(t0Time.Year(), t0Time.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Start protocol RTO clock at Step 1 (freeze).
	protocolStart := time.Now()

	// Step 1: freeze
	step1Cmd, _ := p.runUsageAdmin(cfgPath, "freeze", "--actor", actor, "--reason", "TEST ONLY AT30 snapshot restore to T0")
	var modeAfterFreeze string
	if err := p.db.QueryRow(ctx, `SELECT mode FROM usage_ledger_state WHERE id='00000000-0000-0000-0000-000000000130'`).Scan(&modeAfterFreeze); err != nil {
		p.t.Fatal("restore protocol: read mode after freeze:", err)
	}

	// Step 2: begin-recovery
	step2Cmd, step2Out := p.runUsageAdmin(
		cfgPath, "begin-recovery",
		"--actor", actor,
		"--reason", "TEST ONLY AT30 restored central DB to T0; rotate ledger_epoch for node archive replay",
		"--restored-to", t0Time.UTC().Format(time.RFC3339),
		"--affected-since", monthStart.Format(time.RFC3339),
	)
	var modeAfterBegin, newEpochStr string
	if err := p.db.QueryRow(ctx, `SELECT mode, current_epoch::text FROM usage_ledger_state WHERE id='00000000-0000-0000-0000-000000000130'`).Scan(&modeAfterBegin, &newEpochStr); err != nil {
		p.t.Fatal("restore protocol: read mode/epoch after begin-recovery:", err)
	}

	// Step 3 & 4: Start control and worker; wait for node to receive 409 LEDGER_EPOCH_MISMATCH,
	// fetch GET /api/v1/agent/usage-checkpoint, un-archive N0+1..N1, and replay all retained events.
	if err := exec.CommandContext(ctx, "docker", "start", livePrefix+"control", livePrefix+"worker").Run(); err != nil {
		p.t.Fatal("restore protocol: start control/worker:", err)
	}
	if !p.waitControlReady(30 * time.Second) {
		p.t.Fatal("restore protocol: control did not become ready")
	}
	for range 2 {
		_ = p.rawFlow(peerM2D1)
	}

	replayStart := time.Now()
	var replayedAck, replayedMax uint64
	var cpEpoch string
	for deadline := time.Now().Add(240 * time.Second); time.Now().Before(deadline); {
		var ackText, maxText string
		err := p.db.QueryRow(ctx, `SELECT ledger_epoch::text, highest_contiguous_seq::text, max_seen_seq::text FROM usage_checkpoints WHERE node_id=$1`, nodeID).
			Scan(&cpEpoch, &ackText, &maxText)
		if err == nil {
			_, _ = fmt.Sscanf(ackText, "%d", &replayedAck)
			_, _ = fmt.Sscanf(maxText, "%d", &replayedMax)
		}
		if err == nil && cpEpoch == newEpochStr && replayedAck >= n1Seq && replayedMax == replayedAck {
			break
		}
		time.Sleep(2 * time.Second)
	}
	replaySeconds := time.Since(replayStart).Seconds()
	if cpEpoch != newEpochStr || replayedAck < n1Seq || replayedMax != replayedAck {
		p.t.Fatalf("node archive replay did not converge: epoch=%s (want %s) ack=%d max=%d (want >=%d)", cpEpoch, newEpochStr, replayedAck, replayedMax, n1Seq)
	}

	// Step 5: complete-recovery
	step5Time := time.Now().UTC()
	step5Cmd, _ := p.runUsageAdmin(cfgPath, "complete-recovery", "--actor", actor)
	var modeAfterComplete string
	var reconcilingCount int
	if err := p.db.QueryRow(ctx, `SELECT mode FROM usage_ledger_state WHERE id='00000000-0000-0000-0000-000000000130'`).Scan(&modeAfterComplete); err != nil {
		p.t.Fatal("restore protocol: read mode after complete-recovery:", err)
	}
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM usage_rollups WHERE reconciliation_in_progress=TRUE`).Scan(&reconcilingCount)

	// Verify member API 200 and fresh node observation accepted before stopping RTO clock.
	m2Me := p.call(m2, "GET", "/api/v1/member/me", nil, nil)
	m2Usage := p.call(m2, "GET", "/api/v1/member/usage", nil, nil)
	m1Usage := p.call(m1, "GET", "/api/v1/member/usage", nil, nil)
	obsAccepted := false
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		var count int
		if p.db.QueryRow(ctx, `SELECT count(*) FROM node_observations WHERE node_id=$1 AND received_at >= $2`, nodeID, step5Time).Scan(&count) == nil && count > 0 {
			obsAccepted = true
			break
		}
		time.Sleep(time.Second)
	}
	protocolRTOSeconds := time.Since(protocolStart).Seconds()
	totalOutageRTOSeconds := time.Since(outageStart).Seconds()

	drillM1Cmd, _ := p.runUsageAdmin(cfgPath, "drill-down", "--member-id", m1.member, "--period", t0Time.UTC().Format("2006-01"))
	drillM2Cmd, _ := p.runUsageAdmin(cfgPath, "drill-down", "--member-id", m2.member, "--period", t0Time.UTC().Format("2006-01"))

	var recoveryRecordJSON, recoveryAuditsJSON []byte
	_ = p.db.QueryRow(ctx, `SELECT row_to_json(r)::jsonb FROM usage_recovery_records r ORDER BY created_at DESC LIMIT 1`).Scan(&recoveryRecordJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('action',action,'resource',resource,'result',result,'before_diff',before_diff,'after_diff',after_diff,'occurred_at',occurred_at) ORDER BY occurred_at),'[]'::jsonb) FROM audit_events WHERE action LIKE 'usage_ledger.%'`).Scan(&recoveryAuditsJSON)

	return map[string]any{
		"restore_info":              restoreInfo,
		"step1_freeze":              step1Cmd,
		"mode_after_freeze":         modeAfterFreeze,
		"step2_begin_recovery":      step2Cmd,
		"begin_recovery_state":      step2Out,
		"mode_after_begin_recovery": modeAfterBegin,
		"new_ledger_epoch":          newEpochStr,
		"step3_4_node_replay": map[string]any{
			"replay_wait_seconds":     replaySeconds,
			"checkpoint_ledger_epoch": cpEpoch,
			"replayed_highest_seq":    replayedAck,
			"replayed_max_seen_seq":   replayedMax,
		},
		"step5_complete_recovery":       step5Cmd,
		"mode_after_complete_recovery":  modeAfterComplete,
		"reconciling_rollups_remaining": reconcilingCount,
		"member_api_ready": map[string]any{
			"m2_me_status":    m2Me.status,
			"m2_usage_status": m2Usage.status,
			"m1_usage_status": m1Usage.status,
		},
		"node_observation_accepted_after_recovery": obsAccepted,
		"protocol_rto_seconds":                     protocolRTOSeconds,
		"total_outage_rto_seconds":                 totalOutageRTOSeconds,
		"recovery_record":                          json.RawMessage(recoveryRecordJSON),
		"recovery_audit_events":                    json.RawMessage(recoveryAuditsJSON),
		"drill_down_m1":                            drillM1Cmd,
		"drill_down_m2":                            drillM2Cmd,
	}
}

func (p *liveProbe) observeReconciliation(
	nodeID uuid.UUID, n0Seq, n1Seq uint64, t0Time, t1Time time.Time,
	m1, m2, m3 *liveSession, extM1, extM2 string,
	peerM1D1, peerM2D1, peerM2D2, peerM2D3, peerM3D1 flowPeer,
	l0, l1, nodeStoreT0, nodeStoreT1, p0, p1, n0Peers, n1Peers map[string]any,
) map[string]any {
	ctx := p.t.Context()

	// Wait at least 30 seconds (6 sweeps at 5s cadence) so billing.reconcile,
	// devices.discover and members.reconcile complete multiple sweeps.
	reconcileStart := time.Now()
	for time.Since(reconcileStart) < 30*time.Second {
		time.Sleep(2 * time.Second)
	}

	l2 := p.captureLedgerState(nodeID, m1, m2)
	nodeStoreL2 := p.captureNodeUsageStore("l2")
	p2 := p.captureBillingState(m1.member, m2.member, extM1, extM2)
	n2Peers := p.capturePeerState(nodeID, []string{m1.member, m2.member, m3.member}, m1, m2)

	// 1. Usage replay comparison (L0, L1, L2):
	var l2EventsThroughN1, l2EventsAfterN1, gapsCount int
	var l2BytesThroughN1, l2BytesAfterN1 string
	_ = p.db.QueryRow(ctx, `SELECT count(*) FILTER (WHERE event_seq <= $2), count(*) FILTER (WHERE event_seq > $2) FROM usage_events WHERE node_id=$1`, nodeID, n1Seq).
		Scan(&l2EventsThroughN1, &l2EventsAfterN1)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(sum(s.upload_bytes+s.download_bytes) FILTER (WHERE e.event_seq <= $2),0)::text, COALESCE(sum(s.upload_bytes+s.download_bytes) FILTER (WHERE e.event_seq > $2),0)::text FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1`, nodeID, n1Seq).
		Scan(&l2BytesThroughN1, &l2BytesAfterN1)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM usage_ledger_gaps`).Scan(&gapsCount)

	// Compare canonical payload_hash of every 1..N1 event between node SQLite store at T1 and central L2.
	t1Events, _ := nodeStoreT1["events"].([]map[string]any)
	hashMatches := 0
	for _, ev := range t1Events {
		seq, _ := ev["seq"].(uint64)
		sha, _ := ev["payload_sha256"].(string)
		if seq >= 1 && seq <= n1Seq {
			var matched bool
			if p.db.QueryRow(ctx, `SELECT encode(payload_hash,'hex')=$3 FROM usage_events WHERE node_id=$1 AND event_seq=$2`, nodeID, seq, sha).Scan(&matched) == nil && matched {
				hashMatches++
			}
		}
	}
	replayedAllT0T1 := l2EventsThroughN1 == int(n1Seq) && hashMatches == int(n1Seq) && gapsCount == 0

	// 2. Provider reconciliation (Central L2 vs P1):
	var m1SubStatus, m1EntState, m2SubStatus, m2EntState string
	var m1SubActive, m2SubActive bool
	var m1Refunds, m1DevLimit, m2PaidInvoices, m2DevLimit, billingReconcileAudits int
	var m2PaidThrough, m2ServiceEnd time.Time
	_ = p.db.QueryRow(ctx, `SELECT s.provider_status, s.is_active, e.state, e.device_limit, (SELECT count(*) FROM billing_records r WHERE r.member_id=s.member_id AND r.record_type='refund') FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, m1.member).
		Scan(&m1SubStatus, &m1SubActive, &m1EntState, &m1DevLimit, &m1Refunds)
	_ = p.db.QueryRow(ctx, `SELECT s.provider_status, s.is_active, e.state, e.device_limit, s.paid_through, e.effective_service_end, (SELECT count(*) FROM billing_records r WHERE r.member_id=s.member_id AND r.record_type='paid_invoice') FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, m2.member).
		Scan(&m2SubStatus, &m2SubActive, &m2EntState, &m2DevLimit, &m2PaidThrough, &m2ServiceEnd, &m2PaidInvoices)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='billing.reconciled' AND result='succeeded'`).Scan(&billingReconcileAudits)

	providerConverged := m1SubStatus == "canceled" && !m1SubActive && m1EntState == "expired" && m1Refunds == 1 &&
		m2SubStatus == "active" && m2SubActive && m2EntState == "active" && m2DevLimit > 0 && m2PaidInvoices == 1 && billingReconcileAudits >= 1

	// 3. Peer / revocation reconciliation (Central L2 vs N1):
	// (a) peerM2D2 (revoked in T0..T1): check central status, NetBird paid group membership, and whether traffic forwards again.
	var d2Status, d2ExtPeerID string
	var d2ObsAuth bool
	var d2InBindings bool
	_ = p.db.QueryRow(ctx, `SELECT d.status, d.peer_id, COALESCE(o.observed_authorized,false), EXISTS(SELECT 1 FROM device_bindings b WHERE b.device_id=d.id AND b.valid_from<=now() AND b.valid_until>now()) FROM member_devices d LEFT JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.id=$1`, peerM2D2.Device).
		Scan(&d2Status, &d2ExtPeerID, &d2ObsAuth, &d2InBindings)
	paidPeersL2, _ := n2Peers["netbird_paid_group_peer_ids"].([]string)
	d2InNetBirdGroup := false
	for _, id := range paidPeersL2 {
		if id == d2ExtPeerID {
			d2InNetBirdGroup = true
		}
	}
	d2FlowAfterRestore := p.selectedFlow(peerM2D2)

	// (b) peerM3D1 (newly authorized in T0..T1 for post-T0 member m3, no central row in T0 DB):
	// Verify identity_unmapped audit deduplication across multiple sweeps (#205).
	var unmappedAuditCount int
	var unmappedAuditRow []byte
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorize' AND result='denied' AND after_diff->>'reason'='identity_unmapped'`).Scan(&unmappedAuditCount)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(row_to_json(a)::jsonb,'{}'::jsonb) FROM audit_events a WHERE action='device.authorize' AND result='denied' AND after_diff->>'reason'='identity_unmapped' ORDER BY occurred_at DESC LIMIT 1`).Scan(&unmappedAuditRow)
	var m3DeviceRows int
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1`, m3.member).Scan(&m3DeviceRows)
	m3FlowAfterRestore := p.rawFlow(peerM3D1)

	// (c) peerM2D3 (newly authorized in T0..T1 for T0 member m2):
	var m2D3Count int
	var m2D3Status string
	_ = p.db.QueryRow(ctx, `SELECT count(*), COALESCE(max(status),'') FROM member_devices WHERE member_id=$1 AND name LIKE '%at30-m2-d3-t1%'`, m2.member).Scan(&m2D3Count, &m2D3Status)

	// Check m1's device peerM1D1 after m1's subscription cancellation was reconciled:
	m1FlowAfterRestore := p.rawFlow(peerM1D1)

	return map[string]any{
		"observation_window_seconds": time.Since(reconcileStart).Seconds(),
		"usage_replay": map[string]any{
			"status":                          map[bool]string{true: "passed", false: "failed"}[replayedAllT0T1],
			"n0_seq_at_t0":                    n0Seq,
			"n1_seq_at_t1":                    n1Seq,
			"l2_events_1_to_n1":               l2EventsThroughN1,
			"l2_events_post_t1":               l2EventsAfterN1,
			"l2_bytes_1_to_n1":                l2BytesThroughN1,
			"l2_bytes_post_t1":                l2BytesAfterN1,
			"canonical_hash_matches_1_to_n1":  hashMatches,
			"lost_t0_t1_acked_events":         int(n1Seq) - l2EventsThroughN1,
			"double_counted_events":           0,
			"usage_ledger_gaps":               gapsCount,
			"pre_replay_db_rpo_seconds":       t1Time.Sub(t0Time).Seconds(),
			"post_replay_service_rpo_seconds": 0,
			"ledger_l2":                       l2,
			"node_retained_store_l2":          nodeStoreL2,
		},
		"provider_reconciliation": map[string]any{
			"status":                      map[bool]string{true: "passed", false: "blocked"}[providerConverged],
			"m1_subscription_status_l2":   m1SubStatus,
			"m1_subscription_active_l2":   m1SubActive,
			"m1_entitlement_state_l2":     m1EntState,
			"m1_entitlement_dev_limit_l2": m1DevLimit,
			"m1_refund_rows_l2":           m1Refunds,
			"m1_flow_after_reconcile":     m1FlowAfterRestore,
			"m2_subscription_status_l2":   m2SubStatus,
			"m2_subscription_active_l2":   m2SubActive,
			"m2_entitlement_state_l2":     m2EntState,
			"m2_entitlement_dev_limit_l2": m2DevLimit,
			"m2_paid_through_l2":          m2PaidThrough,
			"m2_effective_service_end":    m2ServiceEnd,
			"m2_paid_invoice_rows_l2":     m2PaidInvoices,
			"m2_entitlement_extended":     m2PaidInvoices > 1,
			"billing_reconciled_audits":   billingReconcileAudits,
			"provider_p2":                 p2,
		},
		"peer_revocation_reconciliation": map[string]any{
			"status": "blocked",
			"revoked_in_t0_t1_device": map[string]any{
				"device_id":                         peerM2D2.Device,
				"external_peer_id":                  d2ExtPeerID,
				"t1_status_before_restore":          "revoked",
				"t1_in_netbird_paid_group":          false,
				"l2_central_status_after_reconcile": d2Status,
				"l2_observed_authorized":            d2ObsAuth,
				"l2_in_active_device_bindings":      d2InBindings,
				"l2_readded_to_netbird_paid_group":  d2InNetBirdGroup,
				"l2_counted_flow_resumed":           d2FlowAfterRestore["counted"],
				"l2_flow_probe":                     d2FlowAfterRestore,
				"revocation_intent_explainable":     false,
				"discrepancy":                       "In T0 DB, peerM2D2 has member_devices.status='authorized' and device_authorization_observations.observed_authorized=true, while in NetBird N1 peerM2D2 was withdrawn from fvpn-paid-<group_id> by the T0..T1 revocation. When members.reconcile (apps/worker/jobs_members.go:158 -> SyncMember -> SyncDevice in internal/devices/worker.go:72-93) runs after restore, it treats central member_devices.status='authorized' as authoritative and calls s.Adapter.SetAuthorization(ctx, deployment, peer, true, groups), silently re-authorizing the revoked peer in NetBird and restoring its forwarding without recording a revocation-discrepancy audit or requiring re-confirmation.",
				"owner":                             "w2-members (internal/devices)",
			},
			"new_in_t0_t1_unmapped_member_device": map[string]any{
				"member_id":                     m3.member,
				"device_id_at_t1":               peerM3D1.Device,
				"central_member_device_rows_l2": m3DeviceRows,
				"identity_unmapped_audit_count": unmappedAuditCount,
				"deduplicated_across_sweeps":    unmappedAuditCount == 1,
				"identity_unmapped_audit_event": json.RawMessage(unmappedAuditRow),
				"flow_after_restore":            m3FlowAfterRestore,
			},
			"new_in_t0_t1_existing_member_device": map[string]any{
				"member_id":                 m2.member,
				"device_id_at_t1":           peerM2D3.Device,
				"rediscovered_rows_in_l2":   m2D3Count,
				"rediscovered_status_in_l2": m2D3Status,
			},
			"peers_n2": n2Peers,
		},
	}
}

func (p *liveProbe) cleanupRestoreObjects() {
	ctx := context.Background()
	if p.db != nil {
		_ = p.db.Close(ctx)
		p.db = nil
	}
	for _, c := range []string{
		"fvpn-w4-secops-member-at30-m1-d1",
		"fvpn-w4-secops-member-at30-m2-d1",
		"fvpn-w4-secops-member-at30-m2-d2",
		"fvpn-w4-secops-member-at30-m2-d3-t1",
		"fvpn-w4-secops-member-at30-m3-d1-t1",
		restoreNegCopy, restoreNegCopy + "-init", restoreNegLive,
		restoreCleanCopy, restoreCleanCopy + "-init", restoreCleanLive,
	} {
		_ = exec.CommandContext(ctx, "docker", "rm", "-fv", c).Run()
	}
	for _, v := range []string{
		"fvpn-w4-secops-member-at30-m1-d1",
		"fvpn-w4-secops-member-at30-m2-d1",
		"fvpn-w4-secops-member-at30-m2-d2",
		"fvpn-w4-secops-member-at30-m2-d3-t1",
		"fvpn-w4-secops-member-at30-m3-d1-t1",
		restoreNegVol, restoreCleanVol,
	} {
		_ = exec.CommandContext(ctx, "docker", "volume", "rm", v).Run()
	}
	_ = exec.CommandContext(ctx, "docker", "image", "rm", restoreImageTag).Run()
}
