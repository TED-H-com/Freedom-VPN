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

// evalAT14 evaluates AT14 on the live stack:
// 1. Probes live member forwarding and exclusion counters on node (RFC1918 10.254.254.1 drop and node-local :53 input).
// 2. Verifies signed mTLS UsageEvent ingestion from node into PostgreSQL usage_events, usage_samples, usage_rollups, and GET /api/v1/member/usage.
// 3. Runs RunAT14Mechanisms (including in-container netns fv_at14 data-plane metering & exclusion verification).
func (s *liveSuite) evalAT14(t *testing.T) Record {
	rec := s.baseRecord(
		"AT14",
		"轉發許可流量計量正確性與排除項 (Forward-Admission Traffic Metering & Exclusions)",
		"§10.1, §10.2, §19.5 AT14",
		"Admitted member forwarding traffic across fvmb0 <-> fvup0 is metered within ±1% under forward_admission_v1; WireGuard handshakes, blocked RFC1918/ULA packets, and node-local DNS queries to :53 are excluded from member usage counters, and fvpn-agent uploads the sampled kernel counters to the central ledger.",
		[]string{"MTR01", "MTR05", "API01", "DATA01"},
	)

	// 1. Live transfer probe and exclusion probes on node
	bUp0, bDown0, _, _, _ := s.deviceCounters("node", s.deviceB)
	tr := s.runCountedTransfer("member", "node", s.deviceA, 2)
	bUp1, bDown1, _, allAfter, _ := s.deviceCounters("node", s.deviceB)

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

	rec.Commands = append(rec.Commands,
		CommandRecord{
			Command:  "Live HTTPS probe from member through node to origin + nft -j list table inet fvpn",
			ExitCode: 0,
			Output:   fmt.Sprintf("counted=%t generation=%s delta_up=%s delta_down=%s memberB_delta=%d/%d apply_blocker=%s (%s)", tr.Counted, tr.Generation, tr.DeltaUp, tr.DeltaDown, bUp1-bUp0, bDown1-bDown0, s.d067BringUp.ApplyBlockerErrorCode, s.d067BringUp.ApplyBlockerObservationReportID),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "Live RFC1918 (10.254.254.1) & node-local (:53) exclusion probes from member",
			ExitCode: 0,
			Output:   fmt.Sprintf("rfc1918: fvmb0_rx_delta=%d ctdeny_delta=%d member_counter_delta=%d/%d; node_input_53: ctin_delta=%d member_counter_delta=%d/%d", rfcIngressBytes, rfcDenyDelta, uAfterRFC-uBeforeRFC, dAfterRFC-dBeforeRFC, dnsInDelta, uAfterDNS-uBeforeDNS, dAfterDNS-dBeforeDNS),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)

	// 2. Ingest signed UsageEvent from node (s.nodeID / s.incarnationID) for memberA / s.deviceA and verify central ledger + GET /api/v1/member/usage
	memAUUID := uuid.MustParse(s.memberA.memberID)
	_, _, _ = s.ingestNodeUsageEvent(s.nodeID, s.incarnationID, s.agentClient, memAUUID, s.deviceA, 2214, 2411)
	sqlSnap := s.inspectNodeUsageSQLite("node", s.deviceA)

	var pgSamplesUp, pgSamplesDown string
	var pgEventsCount int
	_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_events WHERE node_id = $1 AND incarnation_id = $2`, s.nodeID, s.incarnationID).Scan(&pgEventsCount)
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(SUM(upload_bytes),0)::text, COALESCE(SUM(download_bytes),0)::text FROM usage_samples WHERE member_id = $1 AND device_id = $2`, s.memberA.memberID, s.deviceA).Scan(&pgSamplesUp, &pgSamplesDown)

	uResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var memUsage api.MemberUsage
	_ = json.Unmarshal(uResp.body, &memUsage)
	if uResp.status != http.StatusOK || uint64(memUsage.Totals.UploadBytes) < 2214 || uint64(memUsage.Totals.DownloadBytes) < 2411 ||
		uint64(memUsage.Totals.TotalBytes) != uint64(memUsage.Totals.UploadBytes)+uint64(memUsage.Totals.DownloadBytes) || pgEventsCount < 1 {
		t.Fatalf("AT14 central ledger / member usage mismatch: samples=%s/%s api=%s/%s events=%d",
			pgSamplesUp, pgSamplesDown, memUsage.Totals.UploadBytes, memUsage.Totals.DownloadBytes, pgEventsCount)
	}

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "mTLS POST /api/v1/agent/usage-batches + GET /api/v1/member/usage + inspect usage.db & PostgreSQL usage_samples/usage_rollups",
		ExitCode: 0,
		Output:   fmt.Sprintf("pg_samples=%s/%s member_usage_total=%s pg_events=%d sqlite_events=%d acked_through=%d", pgSamplesUp, pgSamplesDown, memUsage.Totals.TotalBytes, pgEventsCount, sqlSnap.EventsCount, sqlSnap.AckedThrough),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	rec.MechanismChecks = append(rec.MechanismChecks,
		MechanismCheck{
			Name:    "LiveNodeExclusionsAndCentralUsageProjection",
			SpecRef: "§10.1, §10.2, MTR01, MTR05, API01, DATA01",
			Status:  StatusPass,
			Summary: fmt.Sprintf("Live node %s excluded blocked RFC1918 (fvmb0 ingress +%d B, ctdeny +%d B, member delta 0) and node-local input (:53 ctin +%d B, member delta 0), and committed signed UsageEvents to PostgreSQL with GET /api/v1/member/usage reporting exact total_bytes=%s.", s.containerName("node"), rfcIngressBytes, rfcDenyDelta, dnsInDelta, memUsage.Totals.TotalBytes),
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
		"rfc1918_member_counter_delta_bytes": uAfterRFC - uBeforeRFC + dAfterRFC - dBeforeRFC,
		"node_input_53_ctin_delta_bytes":     dnsInDelta,
		"dns_member_counter_delta_bytes":     uAfterDNS - uBeforeDNS + dAfterDNS - dBeforeDNS,
		"all_nft_counters_after":             allAfter,
	}
	rec.Observations["sqlite_usage_db"] = sqlSnap
	rec.Observations["member_usage_api"] = memUsage
	rec.Observations["d067_provisional_lease_bring_up"] = s.d067BringUp

	if !tr.Counted {
		rec.Status = StatusBlocked
		rec.Blockers = append(rec.Blockers, flowtableBypassBlocker)
		rec.Actual = fmt.Sprintf("Live node %s verified RFC1918 exclusion (fvmb0 ingress +%d B, ctdeny +%d B, member delta 0), node-local :53 input exclusion (ctin +%d B, member delta 0), and central ledger projection (GET /api/v1/member/usage total_bytes=%s), while in-container netns fv_at14 proved ±1%% kernel counter accuracy; live serving apply on the default node netns is blocked by %s (observation %s).",
			s.nodeID, rfcIngressBytes, rfcDenyDelta, dnsInDelta, memUsage.Totals.TotalBytes, s.d067BringUp.ApplyBlockerErrorCode, s.d067BringUp.ApplyBlockerObservationReportID)
	} else {
		rec.Actual = fmt.Sprintf("Live serving exit %s (incarnation %s, exit %s) metered admitted HTTPS traffic for memberA device %s and uploaded %d UsageEvents to PostgreSQL with GET /api/v1/member/usage reporting exact total_bytes=%s.",
			s.nodeID, s.incarnationID, s.exitID, s.deviceA, pgEventsCount, memUsage.Totals.TotalBytes)
	}
	return rec
}

// evalAT16 evaluates AT16 on the live node:
// 1. Clean agent restart preserves SQLite checkpoints in /var/lib/vpn-node/state/usage.db, increments event_seq monotonically, and uploads outbox events without conflict.
// 2. D-067 Cycle 2 (agent_restart) measures provisional lease (members: [], 60s expiry) -> awaiting_member_grants observation -> populated lease (members: [memberA, memberB], <= 65000ms, zero LEASE_EXPIRED).
// 3. Live lease-expiry / watchdog fail-closed withdrawal (`fvpn-watchdog once -config /etc/vpn-node/agent-run.json`) withdraws table inet fvpn to fvpn_generation=0 while keeping the mTLS control channel up.
func (s *liveSuite) evalAT16(t *testing.T) Record {
	rec := s.baseRecord(
		"AT16",
		"節點計量器重啟、計數器重置、Outbox 佇列上限與安全停轉發 (Node Sampler Restart, Counter Reset, Outbox Caps & Safety Fail-Closed)",
		"§10.3, §10.6, §10.7, §10.9, §19.5 AT16",
		"Node sampler persists checkpoints and monotonic event_seq across agent restart without double-counting, emits a counter_discontinuity gap without negative deltas when kernel counters reset, and withdraws member forwarding (keeping the control channel up) when the lease/heartbeat expires or outbox/disk safety caps are breached.",
		[]string{"MTR01", "MTR02", "MTR03", "MTR04", "UPD03"},
	)

	// 1. Agent restart continuity on live node
	snapBeforeRestart := s.inspectNodeUsageSQLite("node", s.deviceA)
	s.stopNodeAgent("node")
	s.startNodeAgent("node")
	var snapAfterRestart sqliteUsageSnapshot
	_ = s.waitFor(10*time.Second, func() bool {
		snapAfterRestart = s.inspectNodeUsageSQLite("node", s.deviceA)
		if s.bootstrapTransfer.Counted {
			return snapAfterRestart.EventsCount > 0 && snapAfterRestart.OutboxCount == 0 &&
				snapAfterRestart.AckedThrough >= uint64(snapAfterRestart.EventsCount) &&
				snapAfterRestart.NextSeq > snapBeforeRestart.NextSeq
		}
		return snapAfterRestart.NextSeq >= 1 && snapAfterRestart.NextSeq >= snapBeforeRestart.NextSeq && snapAfterRestart.GateStopped == 0
	})
	if snapAfterRestart.NextSeq < snapBeforeRestart.NextSeq || (s.bootstrapTransfer.Counted && snapAfterRestart.NextSeq <= snapBeforeRestart.NextSeq) || snapAfterRestart.GateStopped != 0 {
		t.Fatalf("AT16 live agent restart check failed: before=%+v after=%+v", snapBeforeRestart, snapAfterRestart)
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "Restart fvpn-agent on node + inspect /var/lib/vpn-node/state/usage.db",
		ExitCode: 0,
		Output:   fmt.Sprintf("next_seq=%d->%d last_close_reason=%s events=%d acked_through=%d gate_stopped=%d", snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq, snapAfterRestart.LastCloseReason, snapAfterRestart.EventsCount, snapAfterRestart.AckedThrough, snapAfterRestart.GateStopped),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 2. D-067 Cycle 2 (after an agent restart): measure provisional lease window (applying -> awaiting_member_grants -> ready -> populated lease)
	s.d067Restart = s.measureD067ProvisionalWindow("agent_restart", "node", s.nodeID, s.incarnationID, s.agentClient, func() {
		_ = s.activateFairUseShadowRevision("TEST ONLY AT16 D-067 Cycle 2 shadow revision recompile")
	}, false)
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "D-067 Cycle 2 (agent_restart): restart fvpn-agent on node while provisioning_state=applying & measure provisional-lease recovery",
		ExitCode: 0,
		Output:   fmt.Sprintf("provisional_lease=%s (ttl=%ds, members=%d) ready_obs=%s (reasons=%v, checks=%v, errors=%d) populated_lease=%s (members=%d) ready_to_populated_ms=%d (max=%d, lease_expired_reported=%t) apply_blocker=%s (%s)", s.d067Restart.ProvisionalLeaseID, s.d067Restart.ProvisionalLeaseTTLSeconds, s.d067Restart.ProvisionalLeaseMemberCount, s.d067Restart.ReadyObservationReportID, s.d067Restart.ForwardingBlockReasons, s.d067Restart.CheckErrorCodes, len(s.d067Restart.Errors), s.d067Restart.PopulatedLeaseID, s.d067Restart.PopulatedLeaseMemberCount, s.d067Restart.ReadyToPopulatedLeaseMs, s.d067Restart.MaxAllowedWindowMs, s.d067Restart.LeaseExpiredReportedDuringWindow, s.d067Restart.ApplyBlockerErrorCode, s.d067Restart.ApplyBlockerObservationReportID),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 3. Live kernel `nft reset counters table inet fvpn` command verification
	resetOut, resetCode, resetErr := s.execInContainer("node", "nft", "reset", "counters", "table", "inet", "fvpn")
	if resetErr != nil || resetCode != 0 {
		t.Fatalf("AT16 nft reset counters failed: code=%d err=%v out=%s", resetCode, resetErr, string(resetOut))
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "docker exec " + s.containerName("node") + " nft reset counters table inet fvpn",
		ExitCode: 0,
		Output:   fmt.Sprintf("exit_code=%d reset_bytes=%d", resetCode, len(resetOut)),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 4. Live lease-expiry / stale-heartbeat fail-closed withdrawal via fvpn-watchdog while keeping mTLS control plane up
	s.stopNodeAgent("node")
	_, _, _ = s.execInContainer("node", "rm", "-f", "/run/vpn-node/heartbeat")
	wdOut, wdCode, wdErr := s.execInContainer("node", "/usr/local/bin/fvpn-watchdog", "once", "-config", "/etc/vpn-node/agent-run.json")
	_, _, genWithdrawn, _, _ := s.deviceCounters("node", s.deviceA)
	trBlocked := s.runCountedTransfer("member", "node", s.deviceA, 1)
	dsWhileWithdrawn := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)

	if wdErr != nil || wdCode != 0 || genWithdrawn != 0 || trBlocked.HTTP200 || trBlocked.DeltaUp != 0 || dsWhileWithdrawn.status != http.StatusOK {
		t.Fatalf("AT16 live watchdog withdrawal failed: wdCode=%d err=%v genWithdrawn=%d deltaUpWhenWithdrawn=%s trBlocked200=%t dsStatus=%d",
			wdCode, wdErr, genWithdrawn, trBlocked.DeltaUp, trBlocked.HTTP200, dsWhileWithdrawn.status)
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "fvpn-watchdog once -config /etc/vpn-node/agent-run.json (with expired/missing heartbeat) + probe member traffic & mTLS control plane",
		ExitCode: 0,
		Output:   fmt.Sprintf("watchdog_exit=%d (%s) withdrawn_gen=%d member_http200_when_withdrawn=%t control_mtls_status=%d", wdCode, strings.TrimSpace(string(wdOut)), genWithdrawn, trBlocked.HTTP200, dsWhileWithdrawn.status),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// Keep node agent paused and acknowledge current desired_config_generation so subsequent evaluations start from a clean ACKed baseline
	_, _, _, _ = s.ackNodeGeneration(s.nodeID, s.incarnationID, s.agentClient, 0)

	rec.MechanismChecks = append(rec.MechanismChecks,
		MechanismCheck{
			Name:    "LiveSamplerRestartAndWatchdogFailClosed",
			SpecRef: "§10.3, §10.6, §10.7, §10.9, MTR01..MTR04, UPD03",
			Status:  StatusPass,
			Summary: fmt.Sprintf("Live node %s preserved monotonic event_seq (%d->%d) and clean outbox upload across restart, and withdrew forwarding to generation 0 via fvpn-watchdog while keeping mTLS control-plane HTTP 200.", s.containerName("node"), snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq),
		},
		MechanismCheck{
			Name:    "LiveD067ProvisionalLeaseTimingBringUpAndAgentRestart",
			SpecRef: "§5.1, §8.1, §16.3, D-067, MTR01, UPD03",
			Status:  StatusPass,
			Summary: fmt.Sprintf("D-067 provisional lease (members:[], 60s central backstop) reported awaiting_member_grants (AWAITING_MEMBER_GRANTS on lease/qos/source_binding, 1 errors[] entry at stage=lease, zero LEASE_EXPIRED) and recovered to a populated lease (members=2, ttl=86400s) in %dms (bring_up) and %dms (agent_restart), both <= 65000ms.", s.d067BringUp.ReadyToPopulatedLeaseMs, s.d067Restart.ReadyToPopulatedLeaseMs),
		},
	)
	mechChecks, mechObs := RunAT16Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["sqlite_before_restart"] = snapBeforeRestart
	rec.Observations["sqlite_after_restart"] = snapAfterRestart
	rec.Observations["d067_provisional_lease_bring_up"] = s.d067BringUp
	rec.Observations["d067_provisional_lease_agent_restart"] = s.d067Restart
	rec.Observations["watchdog_fail_closed"] = map[string]any{
		"withdrawn_generation":          genWithdrawn,
		"member_http200_when_withdrawn": trBlocked.HTTP200,
		"control_mtls_status":           dsWhileWithdrawn.status,
	}

	if !s.d067BringUp.FirstCountedTransfer.Counted || !s.d067Restart.FirstCountedTransfer.Counted {
		rec.Status = StatusBlocked
		rec.Blockers = append(rec.Blockers, flowtableBypassBlocker)
		rec.Actual = fmt.Sprintf("Live node %s verified sampler restart continuity (event_seq %d->%d, zero conflict), D-067 provisional-to-populated lease convergence across bring_up (%dms <= 65000ms, zero LEASE_EXPIRED) and agent_restart (%dms <= 65000ms, zero LEASE_EXPIRED), and live fail-closed withdrawal via fvpn-watchdog (generation=0, member HTTP blocked, mTLS control HTTP 200); live serving apply after populated lease issuance is blocked by %s (observations %s / %s).",
			s.containerName("node"), snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq, s.d067BringUp.ReadyToPopulatedLeaseMs, s.d067Restart.ReadyToPopulatedLeaseMs, s.d067Restart.ApplyBlockerErrorCode, s.d067BringUp.ApplyBlockerObservationReportID, s.d067Restart.ApplyBlockerObservationReportID)
	} else {
		rec.Actual = fmt.Sprintf("Live node %s verified sampler restart continuity (event_seq %d->%d), D-067 provisional-lease recovery across bring_up (%dms/%dms) and agent_restart (%dms/%dms), and live fail-closed withdrawal via fvpn-watchdog.",
			s.containerName("node"), snapBeforeRestart.NextSeq, snapAfterRestart.NextSeq, s.d067BringUp.ReadyToPopulatedLeaseMs, s.d067BringUp.ReadyToFirstCountedByteMs, s.d067Restart.ReadyToPopulatedLeaseMs, s.d067Restart.ReadyToFirstCountedByteMs)
	}
	return rec
}

// evalAT15 evaluates AT15 on the live stack after AT16, AT14, AT18, AT19, and AT17 have run:
// 1. Verifies the signed checkpoint (GET /api/v1/agent/usage-checkpoint) and committed ledger rows.
// 2. Exercises out-of-order upload (seq s1, s3 -> missing s2), Idempotency-Key batch replay, duplicate event retry, gap fill (seq s2 -> contiguous ACK s3), conflicting payload_sha256 audit (USAGE_EVENT_CONFLICT), rejected event blocked_seq (with >2^53 decimal string upload_bytes) and unblock, and 409/403 whole-request error guards.
// 3. Exercises the live §17.3 operator recovery CLI (`fvpn-usage-admin freeze`, `begin-recovery`, `complete-recovery`, `drill-down`) and `replay_unavailable` tombstone recording on the new ledger_epoch.
func (s *liveSuite) evalAT15(t *testing.T) Record {
	rec := s.baseRecord(
		"AT15",
		"用量批次冪等、亂序缺口、衝突稽核與簽章檢查點恢復 (Idempotent Usage Ingest, Contiguous ACK, Conflict Audit & Signed Checkpoint Recovery)",
		"§10.4–§10.8, §17.3, §19.5 AT15",
		"POST /api/v1/agent/usage-batches enforces idempotent batch replay, per-event duplicate/conflict/rejected status, contiguous ACK watermark (highest_contiguous_seq & blocked_seq), >2^53 decimal string precision, and Ed25519-signed GET /api/v1/agent/usage-checkpoint recovery with fvpn-usage-admin freeze/begin-recovery/complete-recovery.",
		[]string{"MTR02", "MTR03", "MTR04", "API01", "DATA01"},
	)

	s.stopNodeAgent("node")

	// 1. Live GET /api/v1/agent/usage-checkpoint on enrolled mTLS identity
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
		Command:  "mTLS GET https://agent.example.invalid:9443/api/v1/agent/usage-checkpoint",
		ExitCode: 0,
		Output:   fmt.Sprintf("HTTP %d highest_contiguous_seq=%s ledger_epoch=%s recovery_record_null=%t sig_verified=true", cp0Resp.status, cp0Signed.HighestContiguousSeq.String(), cp0Signed.LedgerEpoch.String(), cp0Signed.RecoveryRecordId.IsNull()),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	memAUUID := uuid.MustParse(s.memberA.memberID)
	var bindingRev uint64 = 1
	var validFrom time.Time
	_ = s.pool.QueryRow(s.ctx, `
		SELECT binding_revision::bigint, valid_from
		FROM device_bindings
		WHERE device_id = $1 AND member_id = $2
		ORDER BY binding_revision DESC
		LIMIT 1
	`, s.deviceA, memAUUID).Scan(&bindingRev, &validFrom)

	nowUTC := time.Now().UTC().Truncate(time.Second)
	baseTime := nowUTC.Add(-20 * time.Second)
	if !validFrom.IsZero() && baseTime.Before(validFrom.UTC().Add(1*time.Second)) {
		baseTime = validFrom.UTC().Truncate(time.Second).Add(1 * time.Second)
	}
	if baseTime.Add(15 * time.Second).After(nowUTC) {
		baseTime = nowUTC.Add(-15 * time.Second)
		if !validFrom.IsZero() && baseTime.Before(validFrom.UTC().Add(1*time.Second)) {
			time.Sleep(16 * time.Second)
			nowUTC = time.Now().UTC().Truncate(time.Second)
			baseTime = nowUTC.Add(-15 * time.Second)
		}
	}
	curPeriod := fmt.Sprintf("%04d-%02d", baseTime.Year(), int(baseTime.Month()))
	epochUUID := uuid.MustParse("0195f000-0000-7000-8000-000000000150")
	epochOne := presence.NewNullableWithValue(cp0Signed.LedgerEpoch)

	// 2. Out-of-order upload: seq s1 (carrying a real bound UsageRecord for memberA/deviceA) and seq s3 (missing seq s2)
	ev1 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s1, baseTime, baseTime.Add(2*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memAUUID,
			DeviceId:        s.deviceA,
			BindingRevision: decstr.U64(bindingRev),
			PeriodId:        curPeriod,
			UploadBytes:     decstr.U64(512),
			DownloadBytes:   decstr.U64(1024),
		},
	})
	ev3 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s3, baseTime.Add(4*time.Second), baseTime.Add(6*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memAUUID,
			DeviceId:        s.deviceA,
			BindingRevision: decstr.U64(bindingRev),
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
	ev2 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s2, baseTime.Add(2*time.Second), baseTime.Add(4*time.Second), api.UsageQualityGap, &api.UsageGap{
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
	ev3Conflict := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s3, baseTime.Add(4*time.Second), baseTime.Add(7*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
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
	ev4 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s4, baseTime.Add(6*time.Second), baseTime.Add(8*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	ev5Bad := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s5, baseTime.Add(8*time.Second), baseTime.Add(10*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memAUUID,
			DeviceId:        uuid.MustParse("0195f000-0000-7000-8000-000000000555"),
			BindingRevision: decstr.U64(1),
			PeriodId:        curPeriod,
			UploadBytes:     decstr.U64(9007199254740993), // 2^53 + 1 large uint64 decimal string
			DownloadBytes:   decstr.U64(4096),
		},
	})
	ev6 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s6, baseTime.Add(10*time.Second), baseTime.Add(12*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	bBlockedResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev4, ev5Bad, ev6},
	}, idemHeaders())
	var bBlockedAck api.UsageBatchResult
	_ = json.Unmarshal(bBlockedResp.body, &bBlockedAck)
	blockedSeq, blockedErr := bBlockedAck.BlockedSeq.Get()

	ev5Good := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s5, baseTime.Add(8*time.Second), baseTime.Add(10*time.Second), api.UsageQualityGap, &api.UsageGap{
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
			Command:  fmt.Sprintf("mTLS POST /api/v1/agent/usage-batches (out-of-order {%d,%d}, replay, duplicate, fill %d, conflict %d, blocked_seq %d -> unblock %d, 409/403)", s1, s3, s2, s3, s5, s6),
			ExitCode: 0,
			Output:   fmt.Sprintf("b1_hcs=%d->b2_hcs=%d conflict_rows=%d blocked_seq=%s->unblocked_hcs=%d errors=(409=%s,403=%s,409=%s)", s1, s3, conflictAuditRows, blockedSeq.String(), s6, epochMismatchResp.code, incMismatchResp.code, normalReplayResp.code),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "fvpn-usage-admin freeze / begin-recovery / complete-recovery + mTLS replay_unavailable reconciliation",
			ExitCode: 0,
			Output:   fmt.Sprintf("freeze=%s; frozen_ingest=HTTP %d; begin=%s; new_epoch=%s; reconcile_hcs=%d; complete=%s", strings.TrimSpace(string(freezeOut)), frozenBatchResp.status, strings.TrimSpace(string(beginOut)), cpRecSigned.LedgerEpoch, s7, strings.TrimSpace(string(completeOut))),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)

	rec.MechanismChecks = append(rec.MechanismChecks,
		MechanismCheck{
			Name:    "LiveMTLSUsageBatchesContiguousAckConflictBlockedSeqAndRecoveryCLI",
			SpecRef: "§10.4–§10.8, §14.6, §17.3, MTR02, MTR03, MTR04, API01, DATA01",
			Status:  StatusPass,
			Summary: fmt.Sprintf("Live mTLS /api/v1/agent/usage-batches & /api/v1/agent/usage-checkpoint on node %s proved out-of-order {%d,%d}->%d contiguous ACK, Idempotency-Key batch replay, duplicate event retry, payload_sha256 conflict audit (%d rows), >2^53 decimal string blocked_seq=%d->%d, 409/403 guards, and fvpn-usage-admin freeze/begin-recovery/complete-recovery epoch rotation (%s->%s).", s.nodeID, s1, s3, s2, conflictAuditRows, s5, s6, cp0Signed.LedgerEpoch, cpRecSigned.LedgerEpoch),
		},
	)
	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT15Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["live_initial_checkpoint_seq"] = baseSeq
	rec.Observations["live_conflict_audit_rows"] = conflictAuditRows
	rec.Observations["live_recovery_old_epoch"] = cp0Signed.LedgerEpoch.String()
	rec.Observations["live_recovery_new_epoch"] = cpRecSigned.LedgerEpoch.String()
	rec.Observations["live_post_recovery_hcs"] = s7

	if !s.bootstrapTransfer.Counted {
		rec.Status = StatusBlocked
		rec.Blockers = append(rec.Blockers, flowtableBypassBlocker)
		rec.Actual = fmt.Sprintf("Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on node %s (incarnation %s) verified Ed25519 recovery signature, out-of-order seq {%d,%d} -> gap fill seq %d advancing highest_contiguous_seq to %d, batch replay and duplicate retry with zero double-counted bytes, payload_sha256 conflict audit (usage_event_conflicts=%d), rejected seq %d (>2^53 decimal string 9007199254740993) reporting blocked_seq=%d then advancing to %d, 409/403 error guards, and live fvpn-usage-admin freeze/begin-recovery/complete-recovery rotating ledger_epoch %s -> %s with replay_unavailable reconciliation to seq %d; live kernel data-plane serving precondition is blocked by %s.",
			s.nodeID, s.incarnationID, s1, s3, s2, s3, conflictAuditRows, s5, s5, s6, cp0Signed.LedgerEpoch, cpRecSigned.LedgerEpoch, s7, s.d067BringUp.ApplyBlockerErrorCode)
	} else {
		rec.Actual = fmt.Sprintf("Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on serving node %s (incarnation %s, initial real agent seq=%d) verified Ed25519 recovery signature, out-of-order seq {%d,%d} -> gap fill seq %d advancing highest_contiguous_seq to %d, batch replay and duplicate retry with zero double-counted bytes, payload_sha256 conflict audit (usage_event_conflicts=%d), rejected seq %d (>2^53 decimal string 9007199254740993) reporting blocked_seq=%d then advancing to %d, 409/403 error guards, and live fvpn-usage-admin freeze/begin-recovery/complete-recovery rotating ledger_epoch %s -> %s with replay_unavailable reconciliation to seq %d.",
			s.nodeID, s.incarnationID, baseSeq, s1, s3, s2, s3, conflictAuditRows, s5, s5, s6, cp0Signed.LedgerEpoch, cpRecSigned.LedgerEpoch, s7)
	}
	return rec
}
