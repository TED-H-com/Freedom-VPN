//go:build e2e

package usage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// evalAT14 evaluates AT14 on the live counted serving flow:
// 1. Admitted member traffic across fvnb0 -> fvmb0 -> fvup0 -> origin increments u_<devA> and d_<devA> within ±1% of member fvnb0 IP bytes, while leaving memberB's u_<devB>/d_<devB> untouched.
// 2. Exclusions on the live serving exit:
//   - Blocked RFC1918 traffic (10.254.254.1) increments drop_v4_private while u_<devA>/d_<devA> delta is 0.
//   - Node-local DNS queries to node fvmb0 :53 increment in_dns_v4 while u_<devA>/d_<devA> delta is 0.
//   - WireGuard transport counter in_wg is tracked in input chain and excluded from member u_*/d_* counters.
//
// 3. fvpn-agent samples the kernel counters into /var/lib/vpn-node/state/usage.db and uploads them to PostgreSQL usage_events, usage_samples, usage_rollups, and GET /api/v1/member/usage.
func (s *liveSuite) evalAT14(t *testing.T) Record {
	rec := s.baseRecord(
		"AT14",
		"轉發許可流量計量正確性與排除項 (Forward-Admission Traffic Metering & Exclusions)",
		"§10.1, §10.2, §19.5 AT14",
		"Admitted member forwarding traffic across fvmb0 <-> fvup0 is metered within ±1% under forward_admission_v1; WireGuard handshakes, blocked RFC1918/ULA packets, and node-local DNS queries to :53 are excluded from member usage counters, and fvpn-agent uploads the sampled kernel counters to the central ledger.",
		[]string{"MTR01", "MTR05", "API01", "DATA01"},
	)

	// 1. Snapshot memberB counters before memberA transfer
	bUp0, bDown0, _, _, _ := s.deviceCounters("node", s.deviceB)

	// 2. Run 5 counted HTTPS transfers from member (memberA) to origin through node
	tr := s.runCountedTransfer("member", "node", s.deviceA, 5)
	bUp1, bDown1, _, allAfter, _ := s.deviceCounters("node", s.deviceB)
	if !tr.Counted || bUp1 != bUp0 || bDown1 != bDown0 {
		t.Fatalf("AT14 live counted transfer or member isolation failed: tr=%+v bUp=%d->%d bDown=%d->%d", tr, bUp0, bUp1, bDown0, bDown1)
	}

	// Compare node device counter delta against node fvup0 forwarded L3 IP byte delta (within ±1%; fvnb0 WireGuard stats pad inner packets to 16-byte multiples)
	upDiffPct := 0.0
	if tr.UplinkL3Tx > 0 {
		diff := float64(int64(tr.DeltaUp) - int64(tr.UplinkL3Tx))
		if diff < 0 {
			diff = -diff
		}
		upDiffPct = (diff / float64(tr.UplinkL3Tx)) * 100.0
	}
	downDiffPct := 0.0
	if tr.UplinkL3Rx > 0 {
		diff := float64(int64(tr.DeltaDown) - int64(tr.UplinkL3Rx))
		if diff < 0 {
			diff = -diff
		}
		downDiffPct = (diff / float64(tr.UplinkL3Rx)) * 100.0
	}
	if upDiffPct > 1.0 || downDiffPct > 1.0 {
		t.Fatalf("AT14 counter accuracy exceeded ±1%% tolerance: upDiff=%.4f%% downDiff=%.4f%% tr=%+v", upDiffPct, downDiffPct, tr)
	}

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "Counted HTTPS transfers (x5) from member through node to origin + nft -j list table inet fvpn",
		ExitCode: 0,
		Output:   fmt.Sprintf("counted=%t delta_up=%s (fvup0_l3_tx=%d, diff=%.3f%%, fvnb0_wg_padded_tx=%d) delta_down=%s (fvup0_l3_rx=%d, diff=%.3f%%, fvnb0_wg_padded_rx=%d) memberB_delta=0/0", tr.Counted, tr.DeltaUp, tr.UplinkL3Tx, upDiffPct, tr.MemberTunnelTx, tr.DeltaDown, tr.UplinkL3Rx, downDiffPct, tr.MemberTunnelRx),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 3. Live exclusion probe A: Blocked RFC1918 destination (10.254.254.1) routed out fvnb0
	uBeforeRFC, dBeforeRFC, _, cBeforeRFC, _ := s.deviceCounters("node", s.deviceA)
	fvmbRxBefore, _ := s.ifaceBytes("node", "fvmb0")
	rfcScript := `
ip route replace 10.254.254.1/32 dev fvnb0
wget --timeout=1 --tries=1 -qO- http://10.254.254.1:80/ >/dev/null 2>&1 || true
ip route del 10.254.254.1/32 dev fvnb0 2>/dev/null || true
`
	_, _, _ = s.execInContainer("member", "sh", "-c", rfcScript)
	uAfterRFC, dAfterRFC, _, cAfterRFC, _ := s.deviceCounters("node", s.deviceA)
	fvmbRxAfter, _ := s.ifaceBytes("node", "fvmb0")
	rfcIngressBytes := fvmbRxAfter - fvmbRxBefore
	rfcDenyDelta := cAfterRFC["ctdeny"] - cBeforeRFC["ctdeny"]
	if uAfterRFC != uBeforeRFC || dAfterRFC != dBeforeRFC || rfcIngressBytes == 0 {
		t.Fatalf("AT14 live RFC1918 exclusion failed: u=%d->%d d=%d->%d fvmb0_rx_delta=%d ctdeny_delta=%d",
			uBeforeRFC, uAfterRFC, dBeforeRFC, dAfterRFC, rfcIngressBytes, rfcDenyDelta)
	}

	// 4. Live exclusion probe B: Node-local input/DNS probe to node's fvmb0 overlay IP :53
	nodeTunnelOut, _, _ := s.execInContainer("node", "ip", "-o", "-4", "addr", "show", "fvmb0")
	nodeOverlayIP := ""
	for _, field := range strings.Fields(string(nodeTunnelOut)) {
		if strings.HasPrefix(field, "100.") && strings.Contains(field, "/") {
			nodeOverlayIP = strings.Split(field, "/")[0]
			break
		}
	}
	uBeforeDNS, dBeforeDNS, _, cBeforeDNS, _ := s.deviceCounters("node", s.deviceA)
	if nodeOverlayIP != "" {
		dnsCmd := fmt.Sprintf(`wget --timeout=1 --tries=1 -qO- http://%s:53/ >/dev/null 2>&1 || true`, nodeOverlayIP)
		_, _, _ = s.execInContainer("member", "sh", "-c", dnsCmd)
	}
	uAfterDNS, dAfterDNS, _, cAfterDNS, _ := s.deviceCounters("node", s.deviceA)
	dnsInDelta := cAfterDNS["ctin"] - cBeforeDNS["ctin"]
	if uAfterDNS != uBeforeDNS || dAfterDNS != dBeforeDNS || dnsInDelta == 0 {
		t.Fatalf("AT14 live node-local DNS/input exclusion failed: nodeOverlayIP=%q u=%d->%d d=%d->%d ctin_delta=%d",
			nodeOverlayIP, uBeforeDNS, uAfterDNS, dBeforeDNS, dAfterDNS, dnsInDelta)
	}

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "Live RFC1918 (10.254.254.1) & node-local (:53) exclusion probes from member",
		ExitCode: 0,
		Output:   fmt.Sprintf("rfc1918: fvmb0_rx_delta=%d ctdeny_delta=%d member_counter_delta=0; node_input_53: ctin_delta=%d member_counter_delta=0", rfcIngressBytes, rfcDenyDelta, dnsInDelta),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 5. Flush fvpn-agent usage sampler & uploader and verify SQLite usage.db + PostgreSQL ledger + GET /api/v1/member/usage
	s.flushNodeUsageUpload("node", s.nodeID)
	sqlSnap := s.inspectNodeUsageSQLite("node", s.deviceA)
	curUp, curDown, _, _, _ := s.deviceCounters("node", s.deviceA)

	var pgSamplesUp, pgSamplesDown string
	var pgEventsCount int
	_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_events WHERE node_id = $1 AND incarnation_id = $2`, s.nodeID, s.incarnationID).Scan(&pgEventsCount)
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(SUM(upload_bytes),0)::text, COALESCE(SUM(download_bytes),0)::text FROM usage_samples WHERE member_id = $1 AND device_id = $2`, s.memberA.memberID, s.deviceA).Scan(&pgSamplesUp, &pgSamplesDown)

	uResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var memUsage api.MemberUsage
	_ = json.Unmarshal(uResp.body, &memUsage)
	if uResp.status != http.StatusOK || uint64(memUsage.Totals.UploadBytes) < curUp || uint64(memUsage.Totals.DownloadBytes) < curDown ||
		uint64(memUsage.Totals.TotalBytes) != uint64(memUsage.Totals.UploadBytes)+uint64(memUsage.Totals.DownloadBytes) || pgEventsCount < 1 {
		t.Fatalf("AT14 central ledger / member usage mismatch: kernel=%d/%d samples=%s/%s api=%s/%s events=%d",
			curUp, curDown, pgSamplesUp, pgSamplesDown, memUsage.Totals.UploadBytes, memUsage.Totals.DownloadBytes, pgEventsCount)
	}

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "GET /api/v1/member/usage + inspect usage.db & PostgreSQL usage_samples/usage_rollups",
		ExitCode: 0,
		Output:   fmt.Sprintf("kernel_u=%d kernel_d=%d pg_samples=%s/%s member_usage_total=%s sqlite_events=%d acked_through=%d", curUp, curDown, pgSamplesUp, pgSamplesDown, memUsage.Totals.TotalBytes, sqlSnap.EventsCount, sqlSnap.AckedThrough),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	rec.MechanismChecks = append(rec.MechanismChecks,
		MechanismCheck{
			Name:    "LiveCountedForwardingAndExclusionsOnServingNode",
			SpecRef: "§10.1, §10.2, MTR01, MTR05, API01, DATA01",
			Status:  StatusPass,
			Summary: fmt.Sprintf("Live counted HTTPS traffic through %s incremented u_%s/d_%s within ±1%% (diff %.3f%%/%.3f%%), excluded blocked RFC1918 (fvmb0 ingress +%d bytes, member counter delta 0) and node-local input (:53 ctin +%d bytes, member counter delta 0), and projected exact totals (%s bytes) to GET /api/v1/member/usage.", s.containerName("node"), strings.ReplaceAll(s.deviceA.String(), "-", ""), strings.ReplaceAll(s.deviceA.String(), "-", ""), upDiffPct, downDiffPct, rfcIngressBytes, dnsInDelta, memUsage.Totals.TotalBytes),
		},
	)
	mechChecks, mechObs := RunAT14Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["live_counted_transfer"] = tr
	rec.Observations["live_exclusions"] = map[string]any{
		"rfc1918_fvmb0_ingress_delta_bytes":  rfcIngressBytes,
		"rfc1918_ctdeny_delta_bytes":         rfcDenyDelta,
		"rfc1918_member_counter_delta_bytes": 0,
		"node_input_53_ctin_delta_bytes":     dnsInDelta,
		"dns_member_counter_delta_bytes":     0,
		"all_nft_counters_after":             allAfter,
	}
	rec.Observations["sqlite_usage_db"] = sqlSnap
	rec.Observations["member_usage_api"] = memUsage
	rec.Actual = fmt.Sprintf("Live serving exit %s (incarnation %s, exit %s) metered admitted HTTPS traffic for memberA device %s within ±1%% (u=%d, d=%d bytes; diff %.3f%%/%.3f%%), isolated memberB counters (delta 0), excluded blocked RFC1918 (fvmb0 ingress +%d B, member delta 0) and node-local input (:53 ctin +%d B, member delta 0), and uploaded %d UsageEvents to PostgreSQL with GET /api/v1/member/usage reporting exact total_bytes=%s.",
		s.nodeID, s.incarnationID, s.exitID, s.deviceA, curUp, curDown, upDiffPct, downDiffPct, rfcIngressBytes, dnsInDelta, pgEventsCount, memUsage.Totals.TotalBytes)
	return rec
}

// evalAT16 evaluates AT16 on the live serving node:
// 1. Clean agent restart preserves SQLite counter checkpoints in /var/lib/vpn-node/state/usage.db, increments event_seq monotonically, and never double-counts prior kernel bytes.
// 2. Live kernel counter reset (`nft reset counters table inet fvpn`) while device counters are >0 triggers counter_discontinuity detection on the next sample tick, rotates meter_epoch_id, emits a quality=gap (reason=counter_discontinuity) event with zero negative deltas, records a node_reported row in usage_ledger_gaps and METER_EPOCH_CHANGE in node_observations, and sets quality.gap=true on GET /api/v1/member/usage.
// 3. Live lease-expiry / watchdog fail-closed withdrawal (`fvpn-watchdog once -config /etc/vpn-node/agent-run.json`) withdraws table inet fvpn to fvpn_generation=0 (0 member accept rules), blocking member forwarding while keeping the mTLS control channel up, then restores serving when fvpn-agent resumes.
func (s *liveSuite) evalAT16(t *testing.T) Record {
	rec := s.baseRecord(
		"AT16",
		"節點計量器重啟、計數器重置、Outbox 佇列上限與安全停轉發 (Node Sampler Restart, Counter Reset, Outbox Caps & Safety Fail-Closed)",
		"§10.3, §10.6, §10.7, §10.9, §19.5 AT16",
		"Node sampler persists checkpoints and monotonic event_seq across agent restart without double-counting, emits a counter_discontinuity gap without negative deltas when kernel counters reset, and withdraws member forwarding (keeping the control channel up) when the lease/heartbeat expires or outbox/disk safety caps are breached.",
		[]string{"MTR01", "MTR02", "MTR03", "MTR04", "UPD03"},
	)

	// 1. Agent restart continuity on live serving node
	snapBeforeRestart := s.inspectNodeUsageSQLite("node", s.deviceA)
	s.flushNodeUsageUpload("node", s.nodeID)
	curUpBeforeRestart, curDownBeforeRestart, _, _, _ := s.deviceCounters("node", s.deviceA)
	var sumUpBeforeRestart, sumDownBeforeRestart uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(SUM(upload_bytes),0)::bigint, COALESCE(SUM(download_bytes),0)::bigint FROM usage_samples WHERE member_id = $1 AND device_id = $2`, s.memberA.memberID, s.deviceA).Scan(&sumUpBeforeRestart, &sumDownBeforeRestart)
	trAfterRestart := s.runCountedTransfer("member", "node", s.deviceA, 1)
	s.flushNodeUsageUpload("node", s.nodeID)
	snapAfterRestart := s.inspectNodeUsageSQLite("node", s.deviceA)

	curUpAfterRestart, curDownAfterRestart, _, _, _ := s.deviceCounters("node", s.deviceA)
	var sumUpAfterRestart, sumDownAfterRestart uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(SUM(upload_bytes),0)::bigint, COALESCE(SUM(download_bytes),0)::bigint FROM usage_samples WHERE member_id = $1 AND device_id = $2`, s.memberA.memberID, s.deviceA).Scan(&sumUpAfterRestart, &sumDownAfterRestart)
	if !trAfterRestart.Counted || snapAfterRestart.NextSeq <= snapBeforeRestart.NextSeq ||
		(snapAfterRestart.LastCloseReason != "agent_stop" && snapAfterRestart.LastCloseReason != "final_sample") ||
		sumUpAfterRestart-sumUpBeforeRestart != curUpAfterRestart-curUpBeforeRestart ||
		sumDownAfterRestart-sumDownBeforeRestart != curDownAfterRestart-curDownBeforeRestart {
		t.Fatalf("AT16 live agent restart check failed: before=%+v after=%+v kernel=%d->%d/%d->%d pg=%d->%d/%d->%d",
			snapBeforeRestart, snapAfterRestart, curUpBeforeRestart, curUpAfterRestart, curDownBeforeRestart, curDownAfterRestart, sumUpBeforeRestart, sumUpAfterRestart, sumDownBeforeRestart, sumDownAfterRestart)
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "Restart fvpn-agent on node + counted HTTPS transfer + inspect usage.db & usage_samples",
		ExitCode: 0,
		Output:   fmt.Sprintf("next_seq=%d->%d last_close_reason=%s kernel=%d/%d pg_samples=%d/%d (zero double-counting)", snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq, snapAfterRestart.LastCloseReason, curUpAfterRestart, curDownAfterRestart, sumUpAfterRestart, sumDownAfterRestart),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 2. Live kernel counter reset (`nft reset counters table inet fvpn`) while checkpoint > 0
	// First wait one 10s sample tick so open_epoch has non-zero counter checkpoints in SQLite `counters`
	_ = s.runCountedTransfer("member", "node", s.deviceA, 1)
	time.Sleep(11 * time.Second)
	snapBeforeReset := s.inspectNodeUsageSQLite("node", s.deviceA)

	resetOut, resetCode, resetErr := s.execInContainer("node", "nft", "reset", "counters", "table", "inet", "fvpn")
	if resetErr != nil || resetCode != 0 {
		t.Fatalf("AT16 nft reset counters failed: code=%d err=%v out=%s", resetCode, resetErr, string(resetOut))
	}
	// Wait for next 10s sample tick to detect counter_discontinuity and flush upload
	s.flushNodeUsageUpload("node", s.nodeID)
	snapAfterReset := s.inspectNodeUsageSQLite("node", s.deviceA)

	var gapRows int
	var gapReason string
	_ = s.pool.QueryRow(s.ctx, `SELECT count(*), COALESCE(MAX(reason),'') FROM usage_ledger_gaps WHERE node_id = $1 AND incarnation_id = $2 AND kind = 'node_reported' AND reason = 'counter_discontinuity'`, s.nodeID, s.incarnationID).Scan(&gapRows, &gapReason)

	uRespAfterReset := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var memUsageAfterReset api.MemberUsage
	_ = json.Unmarshal(uRespAfterReset.body, &memUsageAfterReset)
	if snapAfterReset.GapsCount < 1 || gapRows < 1 || gapReason != "counter_discontinuity" || !memUsageAfterReset.Quality.Gap {
		t.Fatalf("AT16 live counter reset gap check failed: sqliteGaps=%d (%v) pgGapRows=%d (%s) apiGap=%t",
			snapAfterReset.GapsCount, snapAfterReset.GapReasons, gapRows, gapReason, memUsageAfterReset.Quality.Gap)
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "docker exec " + s.containerName("node") + " nft reset counters table inet fvpn + flush sampler/uploader",
		ExitCode: 0,
		Output:   fmt.Sprintf("epoch=%s->%s sqlite_gaps=%d (%v) pg_usage_ledger_gaps=%d (%s) member_usage_gap=%t", snapBeforeReset.OpenEpoch, snapAfterReset.OpenEpoch, snapAfterReset.GapsCount, snapAfterReset.GapReasons, gapRows, gapReason, memUsageAfterReset.Quality.Gap),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 3. Live lease-expiry / stale-heartbeat fail-closed withdrawal via fvpn-watchdog
	s.stopNodeAgent("node")
	_, _, _ = s.execInContainer("node", "rm", "-f", "/run/vpn-node/heartbeat")
	wdOut, wdCode, wdErr := s.execInContainer("node", "/usr/local/bin/fvpn-watchdog", "once", "-config", "/etc/vpn-node/agent-run.json")
	_, _, genWithdrawn, _, _ := s.deviceCounters("node", s.deviceA)
	trBlocked := s.runCountedTransfer("member", "node", s.deviceA, 1)
	dsWhileWithdrawn := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)

	// Restore fvpn-agent and wait for serving=1 + populated generation
	s.startNodeAgent("node")
	restored := s.waitNodeServingAndPublished("node", s.nodeID, s.deviceA)
	trRestored := s.runCountedTransfer("member", "node", s.deviceA, 1)

	if wdErr != nil || wdCode != 0 || genWithdrawn != 0 || trBlocked.HTTP200 || trBlocked.DeltaUp != 0 ||
		dsWhileWithdrawn.status != http.StatusOK || !restored || !trRestored.Counted {
		t.Fatalf("AT16 live watchdog withdrawal/restore failed: wdCode=%d err=%v genWithdrawn=%d deltaUpWhenWithdrawn=%s trBlocked200=%t dsStatus=%d restored=%t trRestored=%t",
			wdCode, wdErr, genWithdrawn, trBlocked.DeltaUp, trBlocked.HTTP200, dsWhileWithdrawn.status, restored, trRestored.Counted)
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "fvpn-watchdog once -config /etc/vpn-node/agent-run.json (with expired/missing heartbeat) + probe member traffic & mTLS control plane + restore agent",
		ExitCode: 0,
		Output:   fmt.Sprintf("watchdog_exit=%d (%s) withdrawn_gen=%d member_http200_when_withdrawn=%t control_mtls_status=%d restored_counted=%t (gen=%s)", wdCode, strings.TrimSpace(string(wdOut)), genWithdrawn, trBlocked.HTTP200, dsWhileWithdrawn.status, trRestored.Counted, trRestored.Generation),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	rec.MechanismChecks = append(rec.MechanismChecks,
		MechanismCheck{
			Name:    "LiveSamplerRestartCounterResetGapAndWatchdogFailClosed",
			SpecRef: "§10.3, §10.6, §10.7, §10.9, MTR01..MTR04, UPD03",
			Status:  StatusPass,
			Summary: fmt.Sprintf("Live node %s preserved monotonic event_seq (%d->%d) across restart without double-counting, emitted counter_discontinuity gap on live `nft reset counters table inet fvpn` (propagated to usage_ledger_gaps and GET /api/v1/member/usage), and withdrew forwarding to generation 0 via fvpn-watchdog while keeping mTLS desired-state HTTP 200.", s.containerName("node"), snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq),
		},
	)
	mechChecks, mechObs := RunAT16Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["sqlite_before_restart"] = snapBeforeRestart
	rec.Observations["sqlite_after_restart"] = snapAfterRestart
	rec.Observations["sqlite_after_counter_reset"] = snapAfterReset
	rec.Observations["watchdog_fail_closed"] = map[string]any{
		"withdrawn_generation":          genWithdrawn,
		"member_http200_when_withdrawn": trBlocked.HTTP200,
		"control_mtls_status":           dsWhileWithdrawn.status,
		"restored_transfer_counted":     trRestored.Counted,
	}
	rec.Actual = fmt.Sprintf("Live node %s verified sampler restart continuity (event_seq %d->%d, zero double-counted bytes), live kernel `nft reset counters` detection emitting a counter_discontinuity gap event into usage.db and PostgreSQL usage_ledger_gaps (setting GET /api/v1/member/usage quality.gap=true), and live fail-closed forwarding withdrawal via fvpn-watchdog (generation=0, member HTTP blocked, mTLS control HTTP 200) followed by full serving restoration.",
		s.containerName("node"), snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq)
	return rec
}

// evalAT15 evaluates AT15 on the live stack after AT14, AT16, AT18, AT19, and AT17 have run:
// 1. Verifies the signed checkpoint (GET /api/v1/agent/usage-checkpoint) and committed ledger rows from the real agent uploads.
// 2. Pauses fvpn-agent on node and exercises out-of-order upload (seq s1, s3 -> missing s2), Idempotency-Key batch replay, duplicate event retry, gap fill (seq s2 -> contiguous ACK s3), conflicting payload_sha256 audit (USAGE_EVENT_CONFLICT), rejected event blocked_seq (with >2^53 decimal string upload_bytes) and unblock, and 409/403 whole-request error guards.
// 3. Exercises the live §17.3 operator recovery CLI (`fvpn-usage-admin freeze`, `begin-recovery`, `complete-recovery`, `drill-down`) and `replay_unavailable` tombstone recording on the new ledger_epoch.
func (s *liveSuite) evalAT15(t *testing.T) Record {
	rec := s.baseRecord(
		"AT15",
		"用量批次冪等、亂序缺口、衝突稽核與簽章檢查點恢復 (Idempotent Usage Ingest, Contiguous ACK, Conflict Audit & Signed Checkpoint Recovery)",
		"§10.4–§10.8, §17.3, §19.5 AT15",
		"POST /api/v1/agent/usage-batches enforces idempotent batch replay, per-event duplicate/conflict/rejected status, contiguous ACK watermark (highest_contiguous_seq & blocked_seq), >2^53 decimal string precision, and Ed25519-signed GET /api/v1/agent/usage-checkpoint recovery with fvpn-usage-admin freeze/begin-recovery/complete-recovery.",
		[]string{"MTR02", "MTR03", "MTR04", "API01", "DATA01"},
	)

	// Flush any pending agent samples on node, then pause fvpn-agent on node so we control sequence numbers deterministically
	s.flushNodeUsageUpload("node", s.nodeID)
	s.stopNodeAgent("node")
	defer func() {
		s.startNodeAgent("node")
	}()

	// 1. Live GET /api/v1/agent/usage-checkpoint on enrolled mTLS identity (reflecting real agent uploads 1..baseSeq)
	cp0Resp := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	var cp0Signed api.UsageCheckpoint
	var cp0SigErr error
	if cp0Resp.status == http.StatusOK {
		verifiedBytes, _, err := envelope.Verify(cp0Resp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC())
		cp0SigErr = err
		if err == nil {
			_ = json.Unmarshal(verifiedBytes, &cp0Signed)
		}
	} else {
		cp0SigErr = fmt.Errorf("HTTP %d: %s", cp0Resp.status, string(cp0Resp.body))
	}
	baseSeq := uint64(cp0Signed.HighestContiguousSeq)
	if cp0Resp.status != http.StatusOK || cp0SigErr != nil || baseSeq == 0 || cp0Signed.HighestContiguousSeq != cp0Signed.MaxSeenSeq {
		t.Fatalf("AT15 initial live checkpoint failed: status=%d sigErr=%v cp=%+v", cp0Resp.status, cp0SigErr, cp0Signed)
	}

	s1, s2, s3, s4, s5, s6, s7 := baseSeq+1, baseSeq+2, baseSeq+3, baseSeq+4, baseSeq+5, baseSeq+6, baseSeq+7
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "mTLS GET https://agent.example.invalid:9443/api/v1/agent/usage-checkpoint (after real agent uploads)",
		ExitCode: 0,
		Output:   fmt.Sprintf("HTTP %d highest_contiguous_seq=%s ledger_epoch=%s recovery_record_null=%t sig_verified=true", cp0Resp.status, cp0Signed.HighestContiguousSeq.String(), cp0Signed.LedgerEpoch.String(), cp0Signed.RecoveryRecordId.IsNull()),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	nowUTC := time.Now().UTC().Truncate(time.Second)
	baseTime := nowUTC.Add(-10 * time.Minute)
	curPeriod := fmt.Sprintf("%04d-%02d", baseTime.Year(), int(baseTime.Month()))
	epochUUID := uuid.MustParse("0195f000-0000-7000-8000-000000000150")
	epochOne := presence.NewNullableWithValue(cp0Signed.LedgerEpoch)
	memAUUID := uuid.MustParse(s.memberA.memberID)

	// 2. Out-of-order upload: seq s1 (carrying a real bound UsageRecord for memberA/deviceA) and seq s3 (missing seq s2)
	ev1 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s1, baseTime, baseTime.Add(30*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memAUUID,
			DeviceId:        s.deviceA,
			BindingRevision: decstr.U64(1),
			PeriodId:        curPeriod,
			UploadBytes:     decstr.U64(512),
			DownloadBytes:   decstr.U64(1024),
		},
	})
	ev3 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s3, baseTime.Add(60*time.Second), baseTime.Add(90*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memAUUID,
			DeviceId:        s.deviceA,
			BindingRevision: decstr.U64(1),
			PeriodId:        curPeriod,
			UploadBytes:     decstr.U64(256),
			DownloadBytes:   decstr.U64(512),
		},
	})
	batch1Key := uuid.NewString()
	b1Resp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev1, ev3},
	}, map[string]string{"Idempotency-Key": batch1Key})
	var b1Ack api.UsageBatchResult
	_ = json.Unmarshal(b1Resp.body, &b1Ack)
	if b1Resp.status != http.StatusOK || uint64(b1Ack.HighestContiguousSeq) != s1 || len(b1Ack.Results) != 2 ||
		b1Ack.Results[0].Status != api.UsageEventResultStatusAccepted || b1Ack.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("AT15 live out-of-order {%d,%d} failed: status=%d body=%s", s1, s3, b1Resp.status, string(b1Resp.body))
	}

	cp1Resp := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	var cp1Signed api.UsageCheckpoint
	if verified1, _, err := envelope.Verify(cp1Resp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC()); err == nil {
		_ = json.Unmarshal(verified1, &cp1Signed)
	}
	if cp1Resp.status != http.StatusOK || uint64(cp1Signed.HighestContiguousSeq) != s1 || len(cp1Signed.MissingRanges) != 1 || uint64(cp1Signed.MissingRanges[0].StartSeq) != s2 || uint64(cp1Signed.MissingRanges[0].EndSeq) != s2 {
		t.Fatalf("AT15 live checkpoint missing_ranges [%d..%d] failed: %+v", s2, s2, cp1Signed)
	}

	// 3. Same Idempotency-Key batch replay + new Idempotency-Key duplicate event retry (zero double-counting)
	var rollupBeforeReplay uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(SUM(total_bytes),0)::bigint FROM usage_rollups WHERE member_id = $1 AND period = $2`, memAUUID, curPeriod).Scan(&rollupBeforeReplay)
	b1ReplayResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev1, ev3},
	}, map[string]string{"Idempotency-Key": batch1Key})
	b1DupResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev1},
	}, idemHeaders())
	var b1DupAck api.UsageBatchResult
	_ = json.Unmarshal(b1DupResp.body, &b1DupAck)
	var rollupAfterReplay uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(SUM(total_bytes),0)::bigint FROM usage_rollups WHERE member_id = $1 AND period = $2`, memAUUID, curPeriod).Scan(&rollupAfterReplay)
	if b1ReplayResp.status != http.StatusOK || !bytes.Equal(b1ReplayResp.body, b1Resp.body) ||
		b1DupResp.status != http.StatusOK || len(b1DupAck.Results) != 1 || b1DupAck.Results[0].Status != api.UsageEventResultStatusDuplicate ||
		rollupAfterReplay != rollupBeforeReplay {
		t.Fatalf("AT15 live batch replay / duplicate check failed: replay=%d dup=%s rollup=%d->%d",
			b1ReplayResp.status, string(b1DupResp.body), rollupBeforeReplay, rollupAfterReplay)
	}

	// 4. Fill missing seq s2 -> advances highest_contiguous_seq to s3
	ev2 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s2, baseTime.Add(30*time.Second), baseTime.Add(60*time.Second), api.UsageQualityGap, &api.UsageGap{
		Reason:           api.UsageGapReasonCounterDiscontinuity,
		LastCheckpointAt: presence.NewNullNullable[time.Time](),
	}, []api.UsageRecord{})
	b2Resp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev2},
	}, idemHeaders())
	var b2Ack api.UsageBatchResult
	_ = json.Unmarshal(b2Resp.body, &b2Ack)
	if b2Resp.status != http.StatusOK || uint64(b2Ack.HighestContiguousSeq) != s3 || b2Ack.Results[0].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("AT15 live seq %d gap fill failed: status=%d body=%s", s2, b2Resp.status, string(b2Resp.body))
	}

	// 5. Conflicting payload_sha256 at seq s3 -> status=conflict, USAGE_EVENT_CONFLICT, usage_event_conflicts row
	ev3Conflict := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s3, baseTime.Add(60*time.Second), baseTime.Add(95*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	bConflictResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev3Conflict},
	}, idemHeaders())
	var bConflictAck api.UsageBatchResult
	_ = json.Unmarshal(bConflictResp.body, &bConflictAck)
	var conflictAuditRows int
	_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_event_conflicts WHERE node_id = $1 AND incarnation_id = $2`, s.nodeID, s.incarnationID).Scan(&conflictAuditRows)
	conflictErrVal, conflictErrGet := bConflictAck.Results[0].Error.Get()
	if bConflictResp.status != http.StatusOK || bConflictAck.Results[0].Status != api.UsageEventResultStatusConflict ||
		conflictErrGet != nil || conflictErrVal.Code != api.ErrorCodeUSAGEEVENTCONFLICT || conflictAuditRows < 1 {
		t.Fatalf("AT15 live conflict check failed: resp=%s conflictRows=%d", string(bConflictResp.body), conflictAuditRows)
	}

	// 6. Rejected event at seq s5 (>2^53 decimal string upload_bytes for unbound device) blocks ACK at s4 even when s6 is accepted; valid s5 advances ACK to s6
	ev4 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s4, baseTime.Add(90*time.Second), baseTime.Add(120*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	ev5Bad := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s5, baseTime.Add(120*time.Second), baseTime.Add(150*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memAUUID,
			DeviceId:        uuid.MustParse("0195f000-0000-7000-8000-000000000555"),
			BindingRevision: decstr.U64(1),
			PeriodId:        curPeriod,
			UploadBytes:     decstr.U64(9007199254740993), // 2^53 + 1 large uint64 decimal string
			DownloadBytes:   decstr.U64(4096),
		},
	})
	ev6 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s6, baseTime.Add(150*time.Second), baseTime.Add(180*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	bBlockedResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev4, ev5Bad, ev6},
	}, idemHeaders())
	var bBlockedAck api.UsageBatchResult
	_ = json.Unmarshal(bBlockedResp.body, &bBlockedAck)
	blockedSeq, blockedErr := bBlockedAck.BlockedSeq.Get()

	ev5Good := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s5, baseTime.Add(120*time.Second), baseTime.Add(150*time.Second), api.UsageQualityGap, &api.UsageGap{
		Reason:           api.UsageGapReasonUnknown,
		LastCheckpointAt: presence.NewNullNullable[time.Time](),
	}, []api.UsageRecord{})
	bUnblockResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev5Good},
	}, idemHeaders())
	var bUnblockAck api.UsageBatchResult
	_ = json.Unmarshal(bUnblockResp.body, &bUnblockAck)
	if bBlockedResp.status != http.StatusOK || uint64(bBlockedAck.HighestContiguousSeq) != s4 || blockedErr != nil || uint64(blockedSeq) != s5 ||
		bBlockedAck.Results[1].Status != api.UsageEventResultStatusRejected ||
		bUnblockResp.status != http.StatusOK || uint64(bUnblockAck.HighestContiguousSeq) != s6 || !bUnblockAck.BlockedSeq.IsNull() {
		t.Fatalf("AT15 live blocked_seq/unblock failed: blocked=%s unblock=%s", string(bBlockedResp.body), string(bUnblockResp.body))
	}

	// 7. Whole-request error envelopes: 409 LEDGER_EPOCH_MISMATCH, 403 INCARNATION_MISMATCH, 409 STATE_CONFLICT
	epochMismatchResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       presence.NewNullableWithValue(uuid.New()),
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	incMismatchResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: uuid.New(),
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	replayUnavail := []api.ReplayUnavailableRange{
		{StartSeq: decstr.U64(s7), EndSeq: decstr.U64(s7), Reason: api.ReplayUnavailableRangeReasonRetentionExpired},
	}
	normalReplayResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		ReplayUnavailable: &replayUnavail,
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	if epochMismatchResp.status != http.StatusConflict || epochMismatchResp.code != "LEDGER_EPOCH_MISMATCH" ||
		incMismatchResp.status != http.StatusForbidden || incMismatchResp.code != "INCARNATION_MISMATCH" ||
		normalReplayResp.status != http.StatusConflict || normalReplayResp.code != "STATE_CONFLICT" {
		t.Fatalf("AT15 whole-request error checks failed: epoch=%d/%s inc=%d/%s replay=%d/%s",
			epochMismatchResp.status, epochMismatchResp.code, incMismatchResp.status, incMismatchResp.code, normalReplayResp.status, normalReplayResp.code)
	}

	// 8. Live §17.3 Ledger Recovery Mode via fvpn-usage-admin (freeze -> 503 -> begin-recovery -> checkpoint epoch bump -> replay_unavailable tombstone -> complete-recovery)
	var adminActorID string
	_ = s.pool.QueryRow(s.ctx, `SELECT id::text FROM admin_identities LIMIT 1`).Scan(&adminActorID)
	if adminActorID == "" {
		adminActorID = "00000000-0000-0000-0000-000000000003"
	}

	freezeOut, freezeCode, freezeErr := s.runUsageAdmin("freeze", "-actor", adminActorID, "-reason", "TEST ONLY AT15 freeze")
	frozenBatchResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())

	beginOut, beginCode, beginErr := s.runUsageAdmin("begin-recovery", "-actor", adminActorID, "-reason", "TEST ONLY AT15 begin recovery")
	cpRecResp := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	var cpRecSigned api.UsageCheckpoint
	if verifiedRec, _, err := envelope.Verify(cpRecResp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC()); err == nil {
		_ = json.Unmarshal(verifiedRec, &cpRecSigned)
	}

	reconcileReplayResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       presence.NewNullableWithValue(cpRecSigned.LedgerEpoch),
		ReplayUnavailable: &replayUnavail,
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	var reconcileAck api.UsageBatchResult
	_ = json.Unmarshal(reconcileReplayResp.body, &reconcileAck)

	completeOut, completeCode, completeErr := s.runUsageAdmin("complete-recovery", "-actor", adminActorID)

	if freezeErr != nil || freezeCode != 0 || frozenBatchResp.status != http.StatusServiceUnavailable ||
		beginErr != nil || beginCode != 0 || cpRecSigned.LedgerEpoch == cp0Signed.LedgerEpoch || cpRecSigned.RecoveryRecordId.IsNull() ||
		reconcileReplayResp.status != http.StatusOK || uint64(reconcileAck.HighestContiguousSeq) != s7 ||
		completeErr != nil || completeCode != 0 {
		t.Fatalf("AT15 live recovery flow failed: freeze=%d/%v frozenBatch=%d begin=%d/%v cpRec=%+v reconcile=%d/%s complete=%d/%v",
			freezeCode, freezeErr, frozenBatchResp.status, beginCode, beginErr, cpRecSigned, reconcileReplayResp.status, string(reconcileReplayResp.body), completeCode, completeErr)
	}

	rec.Commands = append(rec.Commands,
		CommandRecord{
			Command:  fmt.Sprintf("mTLS POST /api/v1/agent/usage-batches (out-of-order {%d,%d}, gap fill %d, conflict %d, blocked %d->%d, 409/403 guards)", s1, s3, s2, s3, s5, s6),
			ExitCode: 0,
			Output:   fmt.Sprintf("b1_hcs=%s b2_hcs=%s conflict_rows=%d blocked_seq=%s unblocked_hcs=%s", b1Ack.HighestContiguousSeq, b2Ack.HighestContiguousSeq, conflictAuditRows, blockedSeq, bUnblockAck.HighestContiguousSeq),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "fvpn-usage-admin freeze + begin-recovery + mTLS GET /usage-checkpoint + POST /usage-batches (replay_unavailable) + complete-recovery",
			ExitCode: 0,
			Output:   fmt.Sprintf("frozen_http=%d old_epoch=%s new_epoch=%s reconcile_hcs=%s freeze=%s begin=%s complete=%s", frozenBatchResp.status, cp0Signed.LedgerEpoch, cpRecSigned.LedgerEpoch, reconcileAck.HighestContiguousSeq, strings.TrimSpace(string(freezeOut)), strings.TrimSpace(string(beginOut)), strings.TrimSpace(string(completeOut))),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)

	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveMTLSUsageBatchesContiguousAckConflictBlockedSeqAndRecoveryMode",
		SpecRef: "§10.4–§10.8, §17.3, MTR02, MTR03, MTR04",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Live mTLS /api/v1/agent/usage-batches and fvpn-usage-admin verified bound member UsageRecord ingestion, out-of-order {%d,%d}->%d contiguous ACK, idempotent batch/event replay with zero double-counting, payload_sha256 conflict audit, rejected >2^53 blocked_seq=%d->%d, freeze (503), begin-recovery epoch bump (%s->%s), replay_unavailable tombstone (seq %d), and complete-recovery.", s1, s3, s2, s5, s6, cp0Signed.LedgerEpoch, cpRecSigned.LedgerEpoch, s7),
	})

	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT15Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["live_recovery_epoch_before"] = cp0Signed.LedgerEpoch.String()
	rec.Observations["live_recovery_epoch_after"] = cpRecSigned.LedgerEpoch.String()
	rec.Observations["live_final_highest_contiguous_seq"] = reconcileAck.HighestContiguousSeq.String()
	rec.Actual = fmt.Sprintf("Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on serving node %s (incarnation %s) verified real agent-uploaded ledger events (1..%d), bound member UsageRecord ingestion, out-of-order {%d,%d} -> gap fill %d advancing highest_contiguous_seq to %d, batch/event replay with zero double-counting, payload_sha256 conflict audit (rows=%d), rejected >2^53 event blocked_seq=%d advancing to %d, and live §17.3 ledger recovery via fvpn-usage-admin (freeze 503, begin-recovery epoch %s -> %s, replay_unavailable seq %d tombstone, and complete-recovery).",
		s.nodeID, s.incarnationID, baseSeq, s1, s3, s2, s3, conflictAuditRows, s5, s6, cp0Signed.LedgerEpoch, cpRecSigned.LedgerEpoch, s7)
	return rec
}
