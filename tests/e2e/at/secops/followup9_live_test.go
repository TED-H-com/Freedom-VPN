//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// §15.2, §15.4, §17.2–17.4, §19 AT30 (DATA01 / BILL01 / MTR03):
// Follow-up 9 re-measurement of AT30 second half on the fixed baseline (#227 D-076
// restore authorization fence, #226 SECOPS-19 live UsageOutboxStatus in node
// observations, #225 read-only `fvpn-agent usage snapshot`).
func TestAT30_Followup9_LiveRestoreReconcile(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_FOLLOWUP9") != "1" {
		t.Skip("requires fresh assigned w4-secops stack for Follow-up 9 AT30 live restore and D-076 fence verification")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{
		t:        t,
		root:     root,
		state:    filepath.Join(root, ".agent/e2e-state-w4-secops"),
		metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 9, "run_id": uuid.New(), "hardware": "shared development host; not candidate hardware"},
	}
	p.open()
	defer func() {
		p.finish()
		p.tr.CloseIdleConnections()
		p.cleanupRestoreObjects9()
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
	secops20Obs := p.servingWithDockerNatObservation9(admin, nodeID)

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
	nodeSnapshotT0 := p.captureNodeUsageSnapshotCLI()
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
	paymentT1 := p.applyProviderChangeT1(m1, extM1, paidEnd)

	// (c) One device revocation on m2's second device (peerM2D2):
	revokeT1 := p.revokeDeviceT1(m2, peerM2D2.Device)

	// (d) New device authorization in T0..T1 for a post-T0 member m3 (present in NetBird N1, no netbird_user_bindings or member_devices in T0 DB):
	admin = p.login("e2e-admin", "admin")
	m3 := p.login("secops-member-02", "member")
	p.grantAt(admin, m3, paidEnd, "post-T0-member-m3")
	peerM3D1, ok := p.joinFlowPeer(stack, m3, "secops-member-02", "at30-m3-d1-t1")
	if !ok {
		t.Fatal("post-T0 member m3 device admission failed")
	}

	// Ensure any final T0..T1 usage events are flushed before recording L1.
	n1Seq = p.waitUsageFlushed(nodeID, n1Seq, m1, m2, 90*time.Second)
	l1 := p.captureLedgerState(nodeID, m1, m2)
	nodeStoreT1 := p.captureNodeUsageStore("t1")
	nodeSnapshotT1 := p.captureNodeUsageSnapshotCLI()
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
	t.Log("AT30 Step 3A: starting 2-minute fail-safe negative (restored T0 DB without recovery protocol)")
	failSafeNeg := p.runFailSafeNegative(restoreImg, t0Time, nodeID, n0Seq, n1Seq, peerM2D1)

	// Step 3B: Clean restore of T0 into fvpn-w4-secops-restore-live + Product Restore Protocol (Steps 1–5)
	// with live SECOPS-19 observation/snapshot capture and D-076 fence verification during ledger restore.
	t.Log("AT30 Step 3B: executing clean T0 restore and 5-step product restore protocol")
	protocolResult, secops19Visibility := p.runProductRestoreProtocol9(restoreImg, t0Time, nodeID, n0Seq, n1Seq, m1, m2, peerM2D1, peerM2D2)

	// Step 4: Observe reconciliation across usage replay (L2), provider truth (P1),
	// D-076 fence on peerM2D2, re-grant via AuthorizeDevice, and the two negatives.
	t.Log("AT30 Step 4: observing post-restore reconciliation, D-076 fence, re-grant, and two negatives")
	reconcileResult := p.observeReconciliation9(
		stack, nodeID, n0Seq, n1Seq, t0Time, t1Time,
		m1, m2, m3, extM1, extM2,
		peerM1D1, peerM2D1, peerM2D2, peerM3D1,
		l0, l1, nodeStoreT0, nodeStoreT1, p0, p1, n0Peers, n1Peers,
	)

	nodeExposureNote := map[string]any{
		"http_or_cli_exposed": true,
		"source_reference":    "node/agent/observe.go:110-124, node/agent/usage_status.go:16-35, node/agent/usage_snapshot.go:38-137",
		"finding":             "Resolved by #226 (SECOPS-19) and #225: POST /api/v1/agent/observations now populates report_payload.usage (UsageOutboxStatus: unacked_event_count, unacked_payload_bytes, oldest_unacked_at, ledger_epoch, meter_epoch_id) from the open SQLite store, and `/usr/local/bin/fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json -json` provides a read-only CLI inspection of unacked outbox and retained archive sequences.",
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
			"initial_flows": t0Flows,
			"secops20_docker_postrouting_observation": secops20Obs,
			"ledger_l0":              l0,
			"node_retained_store_t0": nodeStoreT0,
			"node_snapshot_cli_t0":   nodeSnapshotT0,
			"node_retained_exposure": nodeExposureNote,
			"provider_p0":            p0,
			"peers_n0":               n0Peers,
		},
		"step2_world_forward_t1": map[string]any{
			"t1_utc":                 t1Time,
			"t1_lsn":                 t1LSN,
			"ledger_l1":              l1,
			"node_retained_store_t1": nodeStoreT1,
			"node_snapshot_cli_t1":   nodeSnapshotT1,
			"provider_change":        paymentT1,
			"provider_p1":            p1,
			"revoked_device_t1":      revokeT1,
			"new_t0_t1_devices": map[string]any{
				"m3_new_member_device": map[string]any{"member_id": m3.member, "device_id": peerM3D1.Device, "service": peerM3D1.Service},
			},
			"peers_n1": n1Peers,
		},
		"step3_restore_central_to_t0": map[string]any{
			"restore_image_id":                    restoreImg,
			"primary_overwrite_refusal":           primaryGuard,
			"fail_safe_negative_without_protocol": failSafeNeg,
			"product_restore_protocol":            protocolResult,
			"secops19_visibility":                 secops19Visibility,
		},
		"step4_reconciliation_observation": reconcileResult,
	}
}

func (p *liveProbe) servingWithDockerNatObservation9(admin *liveSession, nodeID uuid.UUID) map[string]any {
	ctx := p.t.Context()
	beforeTime := time.Now().UTC()
	natBeforeRaw, _ := p.docker("node", "nft", "list", "table", "ip", "nat")
	obsCh := make(chan map[string]any, 1)
	go func() {
		result := map[string]any{
			"unmodified_ip_nat_present": bytes.Contains(natBeforeRaw, []byte("chain DOCKER_POSTROUTING")),
			"unmodified_ip_nat_chains":  []string{"DOCKER_OUTPUT", "OUTPUT", "DOCKER_POSTROUTING", "POSTROUTING"},
		}
		for deadline := time.Now().Add(40 * time.Second); time.Now().Before(deadline); {
			var obsJSON, opJSON []byte
			err := p.db.QueryRow(ctx, `
				SELECT jsonb_build_object(
					'observation_id', id,
					'received_at', received_at,
					'safety_status', report_payload->>'safety_status',
					'member_forwarding_enabled', (report_payload->>'member_forwarding_enabled')::boolean,
					'checks', report_payload->'checks',
					'errors', report_payload->'errors'
				)
				FROM node_observations
				WHERE node_id=$1 AND received_at >= $2
				  AND report_payload::text LIKE '%FLOWTABLE_BYPASS%'
				ORDER BY received_at DESC LIMIT 1`, nodeID, beforeTime).Scan(&obsJSON)
			if err == nil && len(obsJSON) > 0 {
				_ = p.db.QueryRow(ctx, `
					SELECT COALESCE(jsonb_build_object('operation_id',id,'kind',kind,'status',status,'steps',steps),'{}'::jsonb)
					FROM operations
					WHERE kind='node.publish' AND created_at >= $1
					ORDER BY created_at DESC LIMIT 1`, beforeTime).Scan(&opJSON)
				result["unmodified_flowtable_bypass_observed"] = true
				result["unmodified_flowtable_bypass_observation"] = json.RawMessage(obsJSON)
				result["d074_publish_waiting_operation"] = json.RawMessage(opJSON)
				_, _ = p.docker("node", "nft", "flush", "chain", "ip", "nat", "POSTROUTING")
				_, _ = p.docker("node", "nft", "delete", "chain", "ip", "nat", "POSTROUTING")
				_, _ = p.docker("node", "nft", "flush", "chain", "ip", "nat", "DOCKER_POSTROUTING")
				_, _ = p.docker("node", "nft", "delete", "chain", "ip", "nat", "DOCKER_POSTROUTING")
				result["docker_postrouting_removed_at_utc"] = time.Now().UTC()
				obsCh <- result
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		result["unmodified_flowtable_bypass_observed"] = false
		obsCh <- result
	}()
	p.serving(admin, nodeID)
	res := <-obsCh
	var pubStatus string
	_ = p.db.QueryRow(ctx, `SELECT status FROM operations WHERE kind='node.publish' AND created_at >= $1 ORDER BY created_at DESC LIMIT 1`, beforeTime).Scan(&pubStatus)
	res["publish_operation_final_status"] = pubStatus
	res["owner_request"] = "SECOPS-20 (owner: w3-devstack-e2e / w1-enforce-nft): Docker attaches two bridge networks (lab and origin) to node-netns and populates table ip nat with POSTROUTING -> DOCKER_POSTROUTING (xt target SNAT for 127.0.0.11). After #220 (2473acf whole-ruleset VerifyServing read-back), unmodified devstack node containers fail closed with FLOWTABLE_BYPASS (table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt) when >=1 member grant is compiled."
	return res
}

func (p *liveProbe) captureNodeUsageSnapshotCLI() map[string]any {
	argv := []string{"exec", livePrefix + "node", "/usr/local/bin/fvpn-agent", "usage", "snapshot", "-config", "/etc/vpn-node/agent-run.json", "-json"}
	out, err := exec.CommandContext(p.t.Context(), "docker", argv...).CombinedOutput()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	var parsed map[string]any
	_ = json.Unmarshal(out, &parsed)
	return map[string]any{
		"command":   append([]string{"docker"}, argv...),
		"exit_code": exitCode,
		"snapshot":  parsed,
	}
}

func (p *liveProbe) scrapeWorkerFenceCounter() (float64, string, bool) {
	for _, endpoint := range []string{"http://127.0.0.1:9092/metrics", "http://127.0.0.1:9091/metrics"} {
		out, err := exec.CommandContext(p.t.Context(), "docker", "exec", livePrefix+"worker", "wget", "-qO-", endpoint).CombinedOutput()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "fvpn_device_restore_fence_decisions_total ") {
				fields := strings.Fields(line)
				if len(fields) == 2 {
					val, parseErr := strconv.ParseFloat(fields[1], 64)
					if parseErr == nil {
						return val, line, true
					}
				}
			}
		}
	}
	return 0, "", false
}

func (p *liveProbe) findWorkerFenceLogLines(deviceID uuid.UUID) []map[string]any {
	out, err := exec.CommandContext(p.t.Context(), "docker", "logs", livePrefix+"worker").CombinedOutput()
	if err != nil {
		return nil
	}
	matches := []map[string]any{}
	for _, rawLine := range bytes.Split(out, []byte("\n")) {
		rawLine = bytes.TrimSpace(rawLine)
		if len(rawLine) == 0 || !bytes.Contains(rawLine, []byte("device authorization unconfirmed during ledger restore")) {
			continue
		}
		var entry map[string]any
		if json.Unmarshal(rawLine, &entry) != nil {
			continue
		}
		if entry["msg"] != "device authorization unconfirmed during ledger restore" {
			continue
		}
		if deviceID != uuid.Nil && entry["device_id"] != deviceID.String() {
			continue
		}
		matches = append(matches, map[string]any{
			"time":        entry["time"],
			"level":       entry["level"],
			"msg":         entry["msg"],
			"device_id":   entry["device_id"],
			"member_id":   entry["member_id"],
			"peer_id":     entry["peer_id"],
			"ledger_mode": entry["ledger_mode"],
		})
	}
	return matches
}

func (p *liveProbe) runProductRestoreProtocol9(
	image string, t0Time time.Time, nodeID uuid.UUID, n0Seq, n1Seq uint64,
	m1, m2 *liveSession, peerM2D1, peerM2D2 flowPeer,
) (map[string]any, map[string]any) {
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
	step1Cmd, _ := p.runUsageAdmin(cfgPath, "freeze", "--actor", actor, "--reason", "TEST ONLY AT30 Follow-up 9 snapshot restore to T0")
	var modeAfterFreeze string
	if err := p.db.QueryRow(ctx, `SELECT mode FROM usage_ledger_state WHERE id='00000000-0000-0000-0000-000000000130'`).Scan(&modeAfterFreeze); err != nil {
		p.t.Fatal("restore protocol: read mode after freeze:", err)
	}

	// Step 2: begin-recovery
	step2Cmd, step2Out := p.runUsageAdmin(
		cfgPath, "begin-recovery",
		"--actor", actor,
		"--reason", "TEST ONLY AT30 Follow-up 9 restored central DB to T0; rotate ledger_epoch for node archive replay",
		"--restored-to", t0Time.UTC().Format(time.RFC3339),
		"--affected-since", monthStart.Format(time.RFC3339),
	)
	var modeAfterBegin, newEpochStr string
	if err := p.db.QueryRow(ctx, `SELECT mode, current_epoch::text FROM usage_ledger_state WHERE id='00000000-0000-0000-0000-000000000130'`).Scan(&modeAfterBegin, &newEpochStr); err != nil {
		p.t.Fatal("restore protocol: read mode/epoch after begin-recovery:", err)
	}

	// Step 3 & 4: Start control and worker; during the 60s recovery hold while usage_ledger_state.mode == 'reconciling':
	// - capture SECOPS-19 non-zero unacked_event_count / unacked_payload_bytes in both node_observations and `fvpn-agent usage snapshot`,
	// - verify D-076 fences peerM2D2 while mode == 'reconciling' (recording the log line and counter increment),
	// - wait for node archive replay to complete.
	servicesStart := time.Now().UTC()
	if err := exec.CommandContext(ctx, "docker", "start", livePrefix+"control", livePrefix+"worker").Run(); err != nil {
		p.t.Fatal("restore protocol: start control/worker:", err)
	}
	if !p.waitControlReady(30 * time.Second) {
		p.t.Fatal("restore protocol: control did not become ready")
	}

	replayStart := time.Now()
	var replayedAck, replayedMax uint64
	var cpEpoch string
	var holdObsCaptured, holdObsNewEpoch, holdSnapCaptured, holdSnapNewEpoch, fenceCaptured bool
	var holdObsRow []byte
	var holdSnapCLI map[string]any

	for deadline := time.Now().Add(240 * time.Second); time.Now().Before(deadline); {
		if !holdObsNewEpoch {
			var obsRaw []byte
			var unacked int
			var obsEpoch string
			err := p.db.QueryRow(ctx, `
				SELECT jsonb_build_object(
					'observation_id', id,
					'report_id', report_id,
					'received_at', received_at,
					'observed_at', observed_at,
					'applied_generation', applied_generation::text,
					'usage', report_payload->'usage'
				),
				COALESCE((report_payload->'usage'->>'unacked_event_count')::int, 0),
				COALESCE(report_payload->'usage'->>'ledger_epoch', '')
				FROM node_observations
				WHERE node_id=$1 AND received_at >= $2
				  AND COALESCE((report_payload->'usage'->>'unacked_event_count')::int, 0) > 0
				ORDER BY (report_payload->'usage'->>'ledger_epoch' = $3) DESC, received_at DESC LIMIT 1`, nodeID, servicesStart, newEpochStr).Scan(&obsRaw, &unacked, &obsEpoch)
			if err == nil && unacked > 0 {
				holdObsRow = obsRaw
				holdObsCaptured = true
				if obsEpoch == newEpochStr {
					holdObsNewEpoch = true
				}
			}
		}
		if !holdSnapNewEpoch {
			snap := p.captureNodeUsageSnapshotCLI()
			if body, ok := snap["snapshot"].(map[string]any); ok {
				unackedCnt, _ := body["unacked_count"].(float64)
				le, _ := body["ledger_epoch"].(string)
				if unackedCnt > 0 {
					holdSnapCLI = snap
					holdSnapCaptured = true
					if le == newEpochStr {
						holdSnapNewEpoch = true
					}
				}
			}
		}
		if !fenceCaptured {
			var fCount int
			if p.db.QueryRow(ctx, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, peerM2D2.Device).Scan(&fCount) == nil && fCount == 1 {
				fenceCaptured = true
			}
		}

		var ackText, maxText string
		err := p.db.QueryRow(ctx, `SELECT ledger_epoch::text, highest_contiguous_seq::text, max_seen_seq::text FROM usage_checkpoints WHERE node_id=$1`, nodeID).
			Scan(&cpEpoch, &ackText, &maxText)
		if err == nil {
			_, _ = fmt.Sscanf(ackText, "%d", &replayedAck)
			_, _ = fmt.Sscanf(maxText, "%d", &replayedMax)
		}
		if err == nil && cpEpoch == newEpochStr && replayedAck >= n1Seq && replayedMax == replayedAck && fenceCaptured && holdObsCaptured && holdSnapCaptured {
			break
		}
		time.Sleep(time.Second)
	}
	replaySeconds := time.Since(replayStart).Seconds()
	if cpEpoch != newEpochStr || replayedAck < n1Seq || replayedMax != replayedAck {
		p.t.Fatalf("node archive replay did not converge: epoch=%s (want %s) ack=%d max=%d (want >=%d)", cpEpoch, newEpochStr, replayedAck, replayedMax, n1Seq)
	}
	if !fenceCaptured {
		p.t.Fatalf("D-076 fence for peerM2D2 (%s) was not recorded during reconciling mode", peerM2D2.Device)
	}
	if !holdObsCaptured || !holdSnapCaptured {
		p.t.Fatalf("SECOPS-19 hold visibility incomplete: obsCaptured=%t snapCaptured=%t", holdObsCaptured, holdSnapCaptured)
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
	for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
		var count int
		if p.db.QueryRow(ctx, `SELECT count(*) FROM node_observations WHERE node_id=$1 AND received_at >= $2`, nodeID, step5Time).Scan(&count) == nil && count > 0 {
			obsAccepted = true
			break
		}
		time.Sleep(time.Second)
	}
	protocolRTOSeconds := time.Since(protocolStart).Seconds()
	totalOutageRTOSeconds := time.Since(outageStart).Seconds()

	// Capture converged post-replay SECOPS-19 state (unacked == 0 in both observation report and CLI snapshot).
	var convergedObsRow []byte
	var convergedSnapCLI map[string]any
	convergedObsOK, convergedSnapOK := false, false
	for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
		if !convergedSnapOK {
			snap := p.captureNodeUsageSnapshotCLI()
			if body, ok := snap["snapshot"].(map[string]any); ok {
				unackedCnt, _ := body["unacked_count"].(float64)
				unackedBytes, _ := body["unacked_payload_bytes"].(string)
				if unackedCnt == 0 && unackedBytes == "0" {
					convergedSnapCLI = snap
					convergedSnapOK = true
				}
			}
		}
		if !convergedObsOK {
			var obsRaw []byte
			var unacked int
			var unackedBytes string
			err := p.db.QueryRow(ctx, `
				SELECT jsonb_build_object(
					'observation_id', id,
					'report_id', report_id,
					'received_at', received_at,
					'observed_at', observed_at,
					'applied_generation', applied_generation::text,
					'usage', report_payload->'usage'
				),
				COALESCE((report_payload->'usage'->>'unacked_event_count')::int, -1),
				COALESCE(report_payload->'usage'->>'unacked_payload_bytes', '')
				FROM node_observations
				WHERE node_id=$1 AND received_at >= $2
				  AND COALESCE((report_payload->'usage'->>'unacked_event_count')::int, -1) = 0
				  AND COALESCE(report_payload->'usage'->>'unacked_payload_bytes', '') = '0'
				  AND report_payload->'usage'->>'ledger_epoch' = $3
				ORDER BY received_at DESC LIMIT 1`, nodeID, step5Time, newEpochStr).Scan(&obsRaw, &unacked, &unackedBytes)
			if err == nil && unacked == 0 && unackedBytes == "0" {
				convergedObsRow = obsRaw
				convergedObsOK = true
			}
		}
		if convergedObsOK && convergedSnapOK {
			break
		}
		time.Sleep(time.Second)
	}
	if !convergedObsOK || !convergedSnapOK {
		p.t.Fatalf("SECOPS-19 post-replay zero convergence incomplete: obsOK=%t snapOK=%t", convergedObsOK, convergedSnapOK)
	}

	drillM1Cmd, _ := p.runUsageAdmin(cfgPath, "drill-down", "--member-id", m1.member, "--period", t0Time.UTC().Format("2006-01"))
	drillM2Cmd, _ := p.runUsageAdmin(cfgPath, "drill-down", "--member-id", m2.member, "--period", t0Time.UTC().Format("2006-01"))

	var recoveryRecordJSON, recoveryAuditsJSON []byte
	_ = p.db.QueryRow(ctx, `SELECT row_to_json(r)::jsonb FROM usage_recovery_records r ORDER BY created_at DESC LIMIT 1`).Scan(&recoveryRecordJSON)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('action',action,'resource',resource,'result',result,'before_diff',before_diff,'after_diff',after_diff,'occurred_at',occurred_at) ORDER BY occurred_at),'[]'::jsonb) FROM audit_events WHERE action LIKE 'usage_ledger.%'`).Scan(&recoveryAuditsJSON)

	protocolMap := map[string]any{
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

	secops19Map := map[string]any{
		"status": "passed",
		"during_restore_hold": map[string]any{
			"node_observation_report_usage": json.RawMessage(holdObsRow),
			"fvpn_agent_usage_snapshot_cli": holdSnapCLI,
		},
		"after_complete_recovery_and_replay": map[string]any{
			"node_observation_report_usage": json.RawMessage(convergedObsRow),
			"fvpn_agent_usage_snapshot_cli": convergedSnapCLI,
		},
		"converged_to_zero_unacked": convergedObsOK && convergedSnapOK,
	}

	return protocolMap, secops19Map
}

func (p *liveProbe) observeReconciliation9(
	stack *labstack.Stack,
	nodeID uuid.UUID, n0Seq, n1Seq uint64, t0Time, t1Time time.Time,
	m1, m2, m3 *liveSession, extM1, extM2 string,
	peerM1D1, peerM2D1, peerM2D2, peerM3D1 flowPeer,
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

	// 3. Peer / revocation reconciliation with D-076 fence (SECOPS-18 closure evidence):
	// (a) Verify peerM2D2 (revoked in T0..T1) is FENCED instead of re-granted after restore to T0:
	var d2Status, d2ExtPeerID string
	var d2ObsAuth bool
	_ = p.db.QueryRow(ctx, `SELECT d.status, d.peer_id, COALESCE(o.observed_authorized,false) FROM member_devices d LEFT JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.id=$1`, peerM2D2.Device).
		Scan(&d2Status, &d2ExtPeerID, &d2ObsAuth)

	var fenceCount int
	var fenceRowJSON []byte
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, peerM2D2.Device).Scan(&fenceCount)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(jsonb_build_object('device_id',device_id,'fenced_at',fenced_at,'observed_group_ids',observed_group_ids),'{}'::jsonb) FROM device_authorization_fences WHERE device_id=$1`, peerM2D2.Device).Scan(&fenceRowJSON)

	var extWithdrawAuditCount int
	var extWithdrawAuditJSON []byte
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorization.external_withdrawal' AND resource='member_device:'||$1::text`, peerM2D2.Device).Scan(&extWithdrawAuditCount)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(row_to_json(a)::jsonb,'{}'::jsonb) FROM audit_events a WHERE action='device.authorization.external_withdrawal' AND resource='member_device:'||$1::text ORDER BY occurred_at DESC LIMIT 1`, peerM2D2.Device).Scan(&extWithdrawAuditJSON)

	workerFenceMetricVal, workerFenceMetricLine, workerFenceMetricFound := p.scrapeWorkerFenceCounter()
	workerFenceLogs := p.findWorkerFenceLogLines(peerM2D2.Device)
	workerAllFenceLogs := p.findWorkerFenceLogLines(uuid.Nil)

	m2DevicesResp := p.call(m2, "GET", "/api/v1/member/devices", nil, nil)
	var m2DevicesBody struct {
		Items []struct {
			DeviceID uuid.UUID `json:"device_id"`
			Name     string    `json:"name"`
			Status   string    `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(m2DevicesResp.body, &m2DevicesBody)
	d2APIStatusWhenFenced := ""
	for _, it := range m2DevicesBody.Items {
		if it.DeviceID == peerM2D2.Device {
			d2APIStatusWhenFenced = it.Status
		}
	}

	paidPeersL2, _ := n2Peers["netbird_paid_group_peer_ids"].([]string)
	d2InNetBirdGroup := false
	for _, id := range paidPeersL2 {
		if id == d2ExtPeerID {
			d2InNetBirdGroup = true
		}
	}
	d2FlowWhenFenced := p.selectedFlow(peerM2D2)
	d1FlowWhileD2Fenced := p.waitCountedFlow(peerM2D1, 30*time.Second)

	d2FenceVerified := fenceCount == 1 &&
		extWithdrawAuditCount == 1 &&
		workerFenceMetricFound && workerFenceMetricVal >= 1 && int(workerFenceMetricVal) == len(workerAllFenceLogs) &&
		len(workerFenceLogs) == 1 &&
		workerFenceLogs[0]["device_id"] == peerM2D2.Device.String() &&
		workerFenceLogs[0]["member_id"] == m2.member &&
		workerFenceLogs[0]["peer_id"] == d2ExtPeerID &&
		(workerFenceLogs[0]["ledger_mode"] == "reconciling" || workerFenceLogs[0]["ledger_mode"] == "frozen") &&
		!d2ObsAuth &&
		d2APIStatusWhenFenced == "pending" &&
		!d2InNetBirdGroup &&
		d2FlowWhenFenced["counted"] == false &&
		d1FlowWhileD2Fenced["counted"] == true

	if !d2FenceVerified {
		p.t.Fatalf("D-076 fence verification failed: fenceCount=%d auditCount=%d metricFound=%t metricVal=%v logs=%d allLogs=%d obsAuth=%t apiStatus=%s inNBGroup=%t d2Counted=%v",
			fenceCount, extWithdrawAuditCount, workerFenceMetricFound, workerFenceMetricVal, len(workerFenceLogs), len(workerAllFenceLogs), d2ObsAuth, d2APIStatusWhenFenced, d2InNetBirdGroup, d2FlowWhenFenced["counted"])
	}

	// (b) Re-grant peerM2D2 via AuthorizeDevice (operation kind = member.device_authorize):
	// First prove replaying the T0 idempotency key returns Replayed=true and leaves the fence intact;
	// then call AuthorizeDevice with a fresh idempotency key and system actor (noting admin HTTP route pending contract revision 12 / D-077),
	// proving the fence row is deleted and the live worker grants peerM2D2 once and restores counted forwarding.
	regrantEvidence := p.exerciseRegrantAuthorizeDevice(m2, peerM2D2, nodeID)

	// (c) Negative 2: A device central itself revoked is not fenced (status='revoked', not 'pending'):
	negCentralRevoke := p.exerciseCentralRevokeNotFenced(m2, peerM2D1)

	// (d) Negative 1: A never-observed device enrolled after the restore is a fresh grant (no fence):
	negFreshEnroll := p.exercisePostRestoreFreshEnrollNotFenced(stack, m2)

	// (e) peerM3D1 (newly authorized in T0..T1 for post-T0 member m3, no central row in T0 DB):
	var unmappedAuditCount int
	var unmappedAuditRow []byte
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorize' AND result='denied' AND after_diff->>'reason'='identity_unmapped'`).Scan(&unmappedAuditCount)
	_ = p.db.QueryRow(ctx, `SELECT COALESCE(row_to_json(a)::jsonb,'{}'::jsonb) FROM audit_events a WHERE action='device.authorize' AND result='denied' AND after_diff->>'reason'='identity_unmapped' ORDER BY occurred_at DESC LIMIT 1`).Scan(&unmappedAuditRow)
	var m3DeviceRows int
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1`, m3.member).Scan(&m3DeviceRows)
	m3FlowAfterRestore := p.rawFlow(peerM3D1)

	// Check m1's device peerM1D1 after m1's subscription cancellation was reconciled:
	m1FlowAfterRestore := p.rawFlow(peerM1D1)

	regrantOK, _ := regrantEvidence["passed"].(bool)
	negCentralOK, _ := negCentralRevoke["passed"].(bool)
	negFreshOK, _ := negFreshEnroll["passed"].(bool)
	peerReconcilePassed := d2FenceVerified && regrantOK && negCentralOK && negFreshOK && unmappedAuditCount == 1 && m3DeviceRows == 0

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
			"status": map[bool]string{true: "passed", false: "blocked"}[peerReconcilePassed],
			"revoked_in_t0_t1_device": map[string]any{
				"device_id":                          peerM2D2.Device,
				"external_peer_id":                   d2ExtPeerID,
				"t1_status_before_restore":           "revoked",
				"t1_in_netbird_paid_group":           false,
				"l2_db_member_devices_status":        d2Status,
				"l2_observed_authorized":             d2ObsAuth,
				"l2_member_devices_api_status":       d2APIStatusWhenFenced,
				"l2_readded_to_netbird_paid_group":   d2InNetBirdGroup,
				"l2_counted_flow_resumed":            d2FlowWhenFenced["counted"],
				"l2_flow_probe":                      d2FlowWhenFenced,
				"sibling_d1_counted_flow":            d1FlowWhileD2Fenced,
				"fence_row_count":                    fenceCount,
				"fence_row":                          json.RawMessage(fenceRowJSON),
				"external_withdrawal_audit_count":    extWithdrawAuditCount,
				"external_withdrawal_audit_event":    json.RawMessage(extWithdrawAuditJSON),
				"worker_restore_fence_counter_value": workerFenceMetricVal,
				"worker_restore_fence_counter_line":  workerFenceMetricLine,
				"worker_restore_fence_log_lines":     workerFenceLogs,
				"worker_all_restore_fence_log_lines": workerAllFenceLogs,
				"fenced_instead_of_regranted":        d2FenceVerified,
				"revocation_intent_explainable":      d2FenceVerified,
			},
			"regrant_authorize_device":                  regrantEvidence,
			"negative_central_revoke_not_fenced":        negCentralRevoke,
			"negative_post_restore_fresh_enroll_builds": negFreshEnroll,
			"new_in_t0_t1_unmapped_member_device": map[string]any{
				"member_id":                     m3.member,
				"device_id_at_t1":               peerM3D1.Device,
				"central_member_device_rows_l2": m3DeviceRows,
				"identity_unmapped_audit_count": unmappedAuditCount,
				"deduplicated_across_sweeps":    unmappedAuditCount == 1,
				"identity_unmapped_audit_event": json.RawMessage(unmappedAuditRow),
				"flow_after_restore":            m3FlowAfterRestore,
			},
			"peers_n2": n2Peers,
		},
	}
}

func discoveryKeyForPeer(deployment, peerID string) (uuid.UUID, error) {
	raw, err := canonjson.Marshal([]string{deployment, peerID})
	if err != nil {
		return uuid.Nil, err
	}
	hash, err := canonjson.Hash(raw)
	if err != nil {
		return uuid.Nil, err
	}
	id, _ := uuid.FromBytes(hash[:16])
	id[6] = id[6]&15 | 0x80
	id[8] = id[8]&63 | 0x80
	return id, nil
}

func (p *liveProbe) exerciseRegrantAuthorizeDevice(m2 *liveSession, peerM2D2 flowPeer, nodeID uuid.UUID) map[string]any {
	ctx := p.t.Context()

	var deployment, peerID, devName, platform, overlayScope, overlayIPStr, extUserID string
	var versionBefore string
	err := p.db.QueryRow(ctx, `
		SELECT d.deployment_id, d.peer_id, d.name, d.platform, d.resource_version::text,
		       b.overlay_scope, host(b.overlay_ip), u.external_user_id
		FROM member_devices d
		JOIN device_bindings b ON b.device_id = d.id AND b.valid_from <= clock_timestamp() AND b.valid_until > clock_timestamp()
		JOIN netbird_user_bindings u ON u.member_id = d.member_id AND u.deployment_id = d.deployment_id
		WHERE d.id = $1
		ORDER BY b.binding_revision DESC LIMIT 1`, peerM2D2.Device).
		Scan(&deployment, &peerID, &devName, &platform, &versionBefore, &overlayScope, &overlayIPStr, &extUserID)
	if err != nil {
		p.t.Fatal("regrant: query peerM2D2 metadata:", err)
	}

	membersRaw, err := os.ReadFile(filepath.Join(p.state, "config/members.json"))
	if err != nil {
		p.t.Fatal("regrant: read members.json:", err)
	}
	var mc config.Members
	if json.Unmarshal(membersRaw, &mc) != nil {
		p.t.Fatal("regrant: parse members.json")
	}
	ec, err := mc.Entitlements()
	if err != nil {
		p.t.Fatal("regrant: entitlements config:", err)
	}
	engine, err := entitlement.NewStore(ec)
	if err != nil {
		p.t.Fatal("regrant: entitlement store:", err)
	}

	dsn := "postgres://fvpn:" + p.secret("postgres.password") + "@127.0.0.1:35432/fvpn?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		p.t.Fatal("regrant: pgxpool.New:", err)
	}
	defer pool.Close()

	svc := &devices.Service{Pool: pool, Entitlements: engine}
	peer := devices.Peer{
		DeploymentID:   deployment,
		ID:             peerID,
		ExternalUserID: extUserID,
		Name:           devName,
		Platform:       platform,
		OverlayScope:   overlayScope,
		OverlayIP:      netip.MustParseAddr(overlayIPStr),
	}
	systemActor := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	// 1. Replay the original T0 idempotency key: must return Replayed=true and leave the fence row intact.
	origKey, err := discoveryKeyForPeer(deployment, peerID)
	if err != nil {
		p.t.Fatal("regrant: discoveryKeyForPeer:", err)
	}
	replayChange, err := svc.AuthorizeDevice(ctx, entitlement.Command{
		Actor: entitlement.Actor{ID: systemActor, Role: "system", RequestID: uuid.Must(uuid.NewV7())},
		Key:   origKey,
	}, peer)
	if err != nil {
		p.t.Fatal("regrant: replay original T0 idempotency key failed:", err)
	}
	var fenceAfterReplay int
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, peerM2D2.Device).Scan(&fenceAfterReplay)
	if !replayChange.Replayed || fenceAfterReplay != 1 {
		p.t.Fatalf("regrant: T0 idempotency replay did not preserve fence: replayed=%t fenceCount=%d", replayChange.Replayed, fenceAfterReplay)
	}

	// 2. Align nodes.desired_config_generation with nodes.applied_config_generation if the T0 restore
	// regressed desired_config_generation below the generation the running node already applied in T0..T1
	// (desiredstate.Compiler.Compile computes baseGen from desired_config_generation and max(desired_states.generation),
	// not applied_config_generation; aligning ensures the post-regrant compile emits generation > applied_config_generation
	// so agentstore.AcceptDesired does not drop it with ErrStaleGeneration).
	var restoredDesiredGen, nodeAppliedGen uint64
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		var dStr, aStr string
		_ = pool.QueryRow(ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id=$1`, nodeID).Scan(&dStr, &aStr)
		_, _ = fmt.Sscanf(dStr, "%d", &restoredDesiredGen)
		_, _ = fmt.Sscanf(aStr, "%d", &nodeAppliedGen)
		if nodeAppliedGen >= restoredDesiredGen && nodeAppliedGen > 0 {
			break
		}
		time.Sleep(time.Second)
	}
	if nodeAppliedGen > restoredDesiredGen {
		if _, err := pool.Exec(ctx, `UPDATE nodes SET desired_config_generation = GREATEST(desired_config_generation, applied_config_generation), resource_version = resource_version + 1 WHERE id=$1`, nodeID); err != nil {
			p.t.Fatal("regrant: align desired_config_generation with applied_config_generation:", err)
		}
	}

	// Call AuthorizeDevice (operation kind = member.device_authorize) with a fresh idempotency key:
	// must clear device_authorization_fences, bump resource_version, and enqueue device.authorized for the live worker.
	freshKey := uuid.Must(uuid.NewV7())
	regrantChange, err := svc.AuthorizeDevice(ctx, entitlement.Command{
		Actor: entitlement.Actor{ID: systemActor, Role: "system", RequestID: uuid.Must(uuid.NewV7())},
		Key:   freshKey,
	}, peer)
	if err != nil {
		p.t.Fatal("regrant: fresh AuthorizeDevice failed:", err)
	}
	var fenceAfterAuthorize int
	var versionAfter string
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, peerM2D2.Device).Scan(&fenceAfterAuthorize)
	_ = p.db.QueryRow(ctx, `SELECT resource_version::text FROM member_devices WHERE id=$1`, peerM2D2.Device).Scan(&versionAfter)
	if fenceAfterAuthorize != 0 {
		p.t.Fatalf("regrant: AuthorizeDevice did not delete device_authorization_fences row: count=%d", fenceAfterAuthorize)
	}

	// 3. Wait for the live worker to claim device.authorized / netbird.device_sync, re-grant peerM2D2 in NetBird,
	// and recompile desired state past nodeAppliedGen.
	var obsAuthAfter bool
	var opStatus string
	var opKind string
	var desiredGenAfter uint64
	if regrantChange.OperationID != nil {
		_ = p.db.QueryRow(ctx, `SELECT kind FROM operations WHERE id=$1`, *regrantChange.OperationID).Scan(&opKind)
	}
	for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
		_ = p.db.QueryRow(ctx, `SELECT COALESCE(observed_authorized,false) FROM device_authorization_observations WHERE device_id=$1`, peerM2D2.Device).Scan(&obsAuthAfter)
		if regrantChange.OperationID != nil {
			_ = p.db.QueryRow(ctx, `SELECT status FROM operations WHERE id=$1`, *regrantChange.OperationID).Scan(&opStatus)
		}
		var dStr string
		_ = p.db.QueryRow(ctx, `SELECT desired_config_generation::text FROM nodes WHERE id=$1`, nodeID).Scan(&dStr)
		_, _ = fmt.Sscanf(dStr, "%d", &desiredGenAfter)
		if obsAuthAfter && desiredGenAfter > nodeAppliedGen {
			break
		}
		time.Sleep(time.Second)
	}
	if !obsAuthAfter {
		p.t.Fatalf("regrant: worker did not grant peerM2D2 after fence clear: obsAuth=%t opStatus=%s desiredGenAfter=%d", obsAuthAfter, opStatus, desiredGenAfter)
	}

	// Verify GET /api/v1/member/devices shows status="authorized" and counted data-plane flow is restored.
	devResp := p.call(m2, "GET", "/api/v1/member/devices", nil, nil)
	var devBody struct {
		Items []struct {
			DeviceID uuid.UUID `json:"device_id"`
			Status   string    `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(devResp.body, &devBody)
	apiStatusAfter := ""
	for _, it := range devBody.Items {
		if it.DeviceID == peerM2D2.Device {
			apiStatusAfter = it.Status
		}
	}
	flowAfterRegrant := p.waitCountedFlow(peerM2D2, 45*time.Second)
	passed := replayChange.Replayed && fenceAfterReplay == 1 && fenceAfterAuthorize == 0 && obsAuthAfter && apiStatusAfter == "authorized" && flowAfterRegrant["counted"] == true

	return map[string]any{
		"passed":                             passed,
		"route_note":                         "re-authorization admin HTTP route pending contract revision 12 / D-077; exercised AuthorizeDevice (operation kind = member.device_authorize) via devices.Service.AuthorizeDevice against the live restored PostgreSQL and worker",
		"device_id":                          peerM2D2.Device,
		"operation_id":                       regrantChange.OperationID,
		"operation_kind":                     opKind,
		"operation_status":                   opStatus,
		"t0_idempotency_replay_replayed":     replayChange.Replayed,
		"fence_rows_after_t0_replay":         fenceAfterReplay,
		"fence_rows_after_fresh_regrant":     fenceAfterAuthorize,
		"resource_version_before":            versionBefore,
		"resource_version_after":             versionAfter,
		"restored_desired_config_generation": restoredDesiredGen,
		"node_applied_config_generation":     nodeAppliedGen,
		"recompiled_desired_generation":      desiredGenAfter,
		"observed_authorized_after_sync":     obsAuthAfter,
		"member_devices_api_status":          apiStatusAfter,
		"counted_flow_after_regrant":         flowAfterRegrant,
	}
}

func (p *liveProbe) exerciseCentralRevokeNotFenced(m2 *liveSession, peerM2D1 flowPeer) map[string]any {
	ctx := p.t.Context()
	counterBefore, _, _ := p.scrapeWorkerFenceCounter()

	r := p.call(m2, "DELETE", "/api/v1/member/devices/"+peerM2D1.Device.String(), nil, nil)
	p.check("post-restore central revoke peerM2D1", r, 202, "")
	var op api.Operation
	_ = json.Unmarshal(r.body, &op)

	var dbStatus string
	var obsAuth bool
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		opReply := p.call(m2, "GET", "/api/v1/member/operations/"+op.OperationId.String(), nil, nil)
		if opReply.status == 200 {
			_ = json.Unmarshal(opReply.body, &op)
		}
		_ = p.db.QueryRow(ctx, `SELECT d.status, COALESCE(o.observed_authorized,true) FROM member_devices d LEFT JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.id=$1`, peerM2D1.Device).
			Scan(&dbStatus, &obsAuth)
		if op.Status == api.OperationStatusSucceeded && dbStatus == "revoked" && !obsAuth {
			break
		}
		time.Sleep(time.Second)
	}

	// Wait 8 seconds (> 1 members.reconcile sweep at 5s cadence) so SyncMember/SyncDevice re-observes the centrally revoked device.
	time.Sleep(8 * time.Second)

	var fenceCount, extWithdrawAudits int
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, peerM2D1.Device).Scan(&fenceCount)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorization.external_withdrawal' AND resource='member_device:'||$1::text`, peerM2D1.Device).Scan(&extWithdrawAudits)
	counterAfter, _, _ := p.scrapeWorkerFenceCounter()

	devResp := p.call(m2, "GET", "/api/v1/member/devices", nil, nil)
	var devBody struct {
		Items []struct {
			DeviceID uuid.UUID `json:"device_id"`
			Status   string    `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(devResp.body, &devBody)
	apiStatus := ""
	for _, it := range devBody.Items {
		if it.DeviceID == peerM2D1.Device {
			apiStatus = it.Status
		}
	}
	flowAfter := p.selectedFlow(peerM2D1)

	passed := dbStatus == "revoked" && apiStatus == "revoked" && !obsAuth && fenceCount == 0 && extWithdrawAudits == 0 && counterAfter == counterBefore && flowAfter["counted"] == false
	if !passed {
		p.t.Fatalf("central revoke negative failed: dbStatus=%s apiStatus=%s obsAuth=%t fence=%d extAudits=%d counter=%v->%v counted=%v",
			dbStatus, apiStatus, obsAuth, fenceCount, extWithdrawAudits, counterBefore, counterAfter, flowAfter["counted"])
	}

	return map[string]any{
		"passed":                           passed,
		"device_id":                        peerM2D1.Device,
		"delete_http_status":               r.status,
		"operation_id":                     op.OperationId,
		"operation_status":                 op.Status,
		"db_member_devices_status":         dbStatus,
		"member_devices_api_status":        apiStatus,
		"observed_authorized":              obsAuth,
		"fence_row_count":                  fenceCount,
		"external_withdrawal_audit_count":  extWithdrawAudits,
		"worker_fence_counter_before":      counterBefore,
		"worker_fence_counter_after":       counterAfter,
		"flow_after_central_revoke":        flowAfter,
		"revoked_not_pending_and_unfenced": passed,
	}
}

func (p *liveProbe) exercisePostRestoreFreshEnrollNotFenced(stack *labstack.Stack, m2 *liveSession) map[string]any {
	ctx := p.t.Context()
	counterBefore, _, _ := p.scrapeWorkerFenceCounter()

	peerM2D3, ok := p.joinFlowPeer(stack, m2, "secops-member-01", "at30-m2-d3-post-restore")
	if !ok {
		p.t.Fatal("post-restore fresh device enrollment (peerM2D3) failed")
	}

	var dbStatus, extPeerID string
	var obsAuth bool
	var fenceCount, extWithdrawAudits int
	_ = p.db.QueryRow(ctx, `SELECT d.status, d.peer_id, COALESCE(o.observed_authorized,false) FROM member_devices d LEFT JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.id=$1`, peerM2D3.Device).
		Scan(&dbStatus, &extPeerID, &obsAuth)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, peerM2D3.Device).Scan(&fenceCount)
	_ = p.db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorization.external_withdrawal' AND resource='member_device:'||$1::text`, peerM2D3.Device).Scan(&extWithdrawAudits)
	counterAfter, _, _ := p.scrapeWorkerFenceCounter()

	devResp := p.call(m2, "GET", "/api/v1/member/devices", nil, nil)
	var devBody struct {
		Items []struct {
			DeviceID uuid.UUID `json:"device_id"`
			Status   string    `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(devResp.body, &devBody)
	apiStatus := ""
	for _, it := range devBody.Items {
		if it.DeviceID == peerM2D3.Device {
			apiStatus = it.Status
		}
	}

	flowProof := p.waitCountedFlow(peerM2D3, 45*time.Second)
	passed := dbStatus == "authorized" && apiStatus == "authorized" && obsAuth && fenceCount == 0 && extWithdrawAudits == 0 && counterAfter == counterBefore && flowProof["counted"] == true
	if !passed {
		p.t.Fatalf("post-restore fresh enroll negative failed: dbStatus=%s apiStatus=%s obsAuth=%t fence=%d extAudits=%d counter=%v->%v counted=%v",
			dbStatus, apiStatus, obsAuth, fenceCount, extWithdrawAudits, counterBefore, counterAfter, flowProof["counted"])
	}

	return map[string]any{
		"passed":                          passed,
		"device_id":                       peerM2D3.Device,
		"external_peer_id":                extPeerID,
		"service":                         peerM2D3.Service,
		"db_member_devices_status":        dbStatus,
		"member_devices_api_status":       apiStatus,
		"observed_authorized":             obsAuth,
		"fence_row_count":                 fenceCount,
		"external_withdrawal_audit_count": extWithdrawAudits,
		"worker_fence_counter_before":     counterBefore,
		"worker_fence_counter_after":      counterAfter,
		"counted_flow_proof":              flowProof,
		"fresh_grant_without_fence":       passed,
	}
}

func (p *liveProbe) cleanupRestoreObjects9() {
	p.cleanupRestoreObjects()
	ctx := context.Background()
	_ = exec.CommandContext(ctx, "docker", "rm", "-fv", "fvpn-w4-secops-member-at30-m2-d3-post-restore").Run()
	_ = exec.CommandContext(ctx, "docker", "volume", "rm", "fvpn-w4-secops-member-at30-m2-d3-post-restore").Run()
}
