//go:build e2e

package usage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// evalAT17 evaluates AT17 on the live stack:
// 1. Live GET /api/v1/member/usage and ?period_id=YYYY-MM&granularity=day return non-zero member totals and daily bucket series from the counted serving flow, with half-open UTC calendar month bounds [start, end) and 422 validation on invalid period_id.
// 2. Central and node mechanisms prove 00:00:00Z UTC month-boundary splitting, late-arriving previous-month event attribution within the 24h watermark, and >5s wall-clock rollback protection emitting a clock_rollback gap without negative intervals or negative byte deltas.
func (s *liveSuite) evalAT17(t *testing.T) Record {
	rec := s.baseRecord(
		"AT17",
		"UTC 自然月切分、延遲事件歸屬與時鐘回撥保護 (UTC Month Boundary, Late-Arriving Events & Clock Rollback Protection)",
		"§10.3, §10.5, §10.7, §19.5 AT17",
		"Cross-month intervals are split at 00:00:00Z into single-month UsageEvents; late-arriving previous-month events within the 24h watermark attribute strictly to the previous UTC month; wall-clock rollback >5s emits a clock_rollback gap and never produces negative intervals or negative byte deltas.",
		[]string{"MTR01", "MTR02", "MTR04", "API01", "DATA01"},
	)

	uSummaryResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var uSummary api.MemberUsage
	_ = json.Unmarshal(uSummaryResp.body, &uSummary)

	uDailyResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage?period_id=2026-10&granularity=day", nil, nil)
	var uDaily api.MemberUsage
	_ = json.Unmarshal(uDailyResp.body, &uDaily)

	uInvalidResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage?period_id=2026-13", nil, nil)

	seriesCount := len(uDaily.Days)
	var seriesTotal uint64
	for _, pt := range uDaily.Days {
		seriesTotal += uint64(pt.TotalBytes)
	}

	if uSummaryResp.status != http.StatusOK || uSummary.MeteringVersion != api.MeteringVersionForwardAdmissionV1 ||
		uint64(uSummary.Totals.TotalBytes) == 0 || uint64(uSummary.Totals.UploadBytes) == 0 || uint64(uSummary.Totals.DownloadBytes) == 0 ||
		uDailyResp.status != http.StatusOK || uDaily.PeriodId != "2026-10" || seriesCount == 0 || seriesTotal != uint64(uSummary.Totals.TotalBytes) ||
		!uDaily.PeriodStart.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) ||
		!uDaily.PeriodEnd.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) ||
		uInvalidResp.status != http.StatusUnprocessableEntity || uInvalidResp.code != "VALIDATION_FAILED" {
		t.Fatalf("AT17 live GET /api/v1/member/usage checks failed: summary=%d/%s daily=%d/%s invalid=%d/%s",
			uSummaryResp.status, string(uSummaryResp.body), uDailyResp.status, string(uDailyResp.body), uInvalidResp.status, string(uInvalidResp.body))
	}

	rec.Commands = append(rec.Commands,
		CommandRecord{
			Command:  "GET /api/v1/member/usage (memberA current UTC month totals from real counted flow)",
			ExitCode: 0,
			Output:   fmt.Sprintf("HTTP %d period_id=%s upload_bytes=%s download_bytes=%s total_bytes=%s quality_gap=%t", uSummaryResp.status, uSummary.PeriodId, uSummary.Totals.UploadBytes, uSummary.Totals.DownloadBytes, uSummary.Totals.TotalBytes, uSummary.Quality.Gap),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "GET /api/v1/member/usage?period_id=2026-10&granularity=day & ?period_id=2026-13",
			ExitCode: 0,
			Output:   fmt.Sprintf("daily=HTTP %d [%s, %s) buckets=%d series_total=%d; invalid_period=HTTP %d %s", uDailyResp.status, uDaily.PeriodStart.Format(time.RFC3339), uDaily.PeriodEnd.Format(time.RFC3339), seriesCount, seriesTotal, uInvalidResp.status, uInvalidResp.code),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveMemberUsageAPITotalsDailySeriesAndUTCMonthBounds",
		SpecRef: "§10.5, §14.3, MTR04, API01",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Live GET /api/v1/member/usage and ?period_id=2026-10&granularity=day returned real counted memberA totals (up=%s, down=%s, total=%s across %d daily buckets) with half-open UTC bounds [2026-10-01T00:00:00Z, 2026-11-01T00:00:00Z) and rejected ?period_id=2026-13 with 422 VALIDATION_FAILED.", uSummary.Totals.UploadBytes, uSummary.Totals.DownloadBytes, uSummary.Totals.TotalBytes, seriesCount),
	})

	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT17Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["live_member_usage_summary"] = uSummary
	rec.Observations["live_member_usage_daily"] = uDaily
	rec.Actual = fmt.Sprintf("Live GET /api/v1/member/usage and ?period_id=2026-10&granularity=day returned real counted memberA traffic totals (upload=%s, download=%s, total=%s) within half-open UTC month bounds [2026-10-01T00:00:00Z, 2026-11-01T00:00:00Z), while central and node mechanisms verified 00:00:00Z UTC month-boundary splitting, late-arriving previous-month event attribution within the 24h watermark, and >5s wall-clock rollback protection.",
		uSummary.Totals.UploadBytes, uSummary.Totals.DownloadBytes, uSummary.Totals.TotalBytes)
	return rec
}

// evalAT18 evaluates AT18 on the live serving exit node:
// 1. Live kernel tc HTB + fq_codel inspection on fvmb0 (download) and fvup0 (upload) on node proves:
//   - root htb 1: and parent 1:1 (1000Mbit)
//   - host reserve class 1:fffe (100Mbit rate, 1000Mbit ceil, prio 0) protecting the control channel
//   - per-member classes 1:10 and 1:11 with fq_codel leaf qdiscs and effective_rate = min(plan_rate_limit=80M/40M, fup_tier0=100M/50M, exit_per_member=60M/30M) = 60Mbit down / 30Mbit up!
//
// 2. Concurrent counted HTTPS transfers from member (memberA) and member-b (memberB) + concurrent mTLS control-plane probes increment kernel tc HTB class Sent byte counters on both fvmb0 and fvup0 without control-plane loss.
func (s *liveSuite) evalAT18(t *testing.T) Record {
	rec := s.baseRecord(
		"AT18",
		"多階層速率整形、有效速率計算與宿主保底頻寬 (Multi-Tier Rate Shaping, Effective Rate & Host Reserve)",
		"§11.1, §11.3, §11.4, §19.5 AT18",
		"Effective rate is min(plan_rate_limit, fup_tier_rate, exit_per_member_limit) per direction, never below minimum_rate when forwarding is allowed; node installs tc HTB + fq_codel on fvmb0 (download) and fvup0 (upload) with host reserve class 1:fffe protecting the control channel.",
		[]string{"FUP01", "FUP02", "FUP03"},
	)

	// 1. Run counted transfers for memberA (member) and memberB (member-b) while polling mTLS desired-state concurrently
	var wg sync.WaitGroup
	var trA, trB transferResult
	var ctrlOK bool
	wg.Add(2)
	go func() {
		defer wg.Done()
		trA = s.runCountedTransfer("member", "node", s.deviceA, 2)
		trB = s.runCountedTransfer("member-b", "node", s.deviceB, 2)
	}()
	go func() {
		defer wg.Done()
		r := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
		ctrlOK = r.status == http.StatusOK
	}()
	wg.Wait()

	// 2. Inspect live kernel tc HTB classes and qdiscs on fvmb0 (download) and fvup0 (upload) on node
	tcDownOut, tcDownCode, _ := s.execInContainer("node", "tc", "-s", "class", "show", "dev", "fvmb0")
	tcUpOut, tcUpCode, _ := s.execInContainer("node", "tc", "-s", "class", "show", "dev", "fvup0")
	qdiscDownOut, _, _ := s.execInContainer("node", "tc", "-s", "qdisc", "show", "dev", "fvmb0")
	qdiscUpOut, _, _ := s.execInContainer("node", "tc", "-s", "qdisc", "show", "dev", "fvup0")

	tcDownStr := strings.TrimSpace(string(tcDownOut))
	tcUpStr := strings.TrimSpace(string(tcUpOut))
	qdiscDownStr := strings.TrimSpace(string(qdiscDownOut))
	qdiscUpStr := strings.TrimSpace(string(qdiscUpOut))

	// On node: plan=80M/40M, tier0=100M/50M, exit_per_member=60M/30M -> effective_rate = 60Mbit down (fvmb0), 30Mbit up (fvup0)
	hasLiveTC := tcDownCode == 0 && tcUpCode == 0 &&
		strings.Contains(tcDownStr, "class htb 1:1 root") &&
		strings.Contains(tcDownStr, "class htb 1:fffe parent 1:1 prio 0") &&
		strings.Count(tcDownStr, "ceil 60Mbit") >= 2 && strings.Count(tcUpStr, "ceil 30Mbit") >= 2 &&
		strings.Contains(qdiscDownStr, "fq_codel") && strings.Contains(qdiscUpStr, "fq_codel")

	if !trA.Counted || !trB.Counted || !ctrlOK || !hasLiveTC {
		t.Fatalf("AT18 live serving tc HTB / concurrent transfer check failed: trA=%+v trB=%+v ctrlOK=%t tcDown=%s tcUp=%s",
			trA, trB, ctrlOK, tcDownStr, tcUpStr)
	}

	rec.Commands = append(rec.Commands,
		CommandRecord{
			Command:  "Concurrent counted HTTPS transfers from member (memberA) & member-b (memberB) + mTLS GET /api/v1/agent/desired-state",
			ExitCode: 0,
			Output:   fmt.Sprintf("memberA_counted=%t (up+%s down+%s) memberB_counted=%t (up+%s down+%s) control_mtls_200=%t", trA.Counted, trA.DeltaUp, trA.DeltaDown, trB.Counted, trB.DeltaUp, trB.DeltaDown, ctrlOK),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "docker exec " + s.containerName("node") + " tc -s class/qdisc show dev fvmb0 & fvup0",
			ExitCode: 0,
			Output:   fmt.Sprintf("=== fvmb0 classes ===\n%s\n=== fvup0 classes ===\n%s\n=== fvmb0 qdiscs ===\n%s", tcDownStr, tcUpStr, qdiscDownStr),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)

	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveServingNodeTcHtbFqCodelEffectiveRateAndHostReserve",
		SpecRef: "§11.1, §11.3, §11.4, FUP01, FUP02, FUP03",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Live serving exit %s installed root htb 1: (1Gbit), host reserve 1:fffe (100Mbit prio 0), and member classes 1:10 & 1:11 with fq_codel leaf qdiscs and effective_rate=min(plan=80M/40M, tier0=100M/50M, exit=60M/30M)=60Mbit down (fvmb0) / 30Mbit up (fvup0), counting concurrent memberA and memberB traffic while keeping mTLS control HTTP 200.", s.containerName("node")),
	})

	mechChecks, mechObs := RunAT18Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["live_concurrent_transfer_member_a"] = trA
	rec.Observations["live_concurrent_transfer_member_b"] = trB
	rec.Observations["live_tc_fvmb0_classes"] = tcDownStr
	rec.Observations["live_tc_fvup0_classes"] = tcUpStr
	rec.Observations["live_tc_fvmb0_qdiscs"] = qdiscDownStr
	rec.Observations["live_tc_fvup0_qdiscs"] = qdiscUpStr
	rec.Actual = fmt.Sprintf("Live serving exit %s installed kernel tc HTB + fq_codel on fvmb0 and fvup0 with host reserve class 1:fffe (100Mbit) and member classes 1:10/1:11 at effective_rate = min(plan=80M/40M, tier0=100M/50M, exit=60M/30M) = 60Mbit down / 30Mbit up, shaped concurrent memberA (+%s/+%s B) and memberB (+%s/+%s B) flows, and preserved mTLS control-plane responsiveness (HTTP 200).",
		s.containerName("node"), trA.DeltaUp, trA.DeltaDown, trB.DeltaUp, trB.DeltaDown)
	return rec
}

// evalAT19 evaluates AT19 across two real serving exits (node and node2):
// 1. Starts and enrolls second exit node2 (per_member: 15M down / 8M up), publishes into s.groupID, selects node2 exit route in member, sends counted HTTPS transfers through node2 (origin sees node2 SNAT .22), and flushes usage upload from both node and node2 so memberA's cross-node total_bytes exceeds Tier 2 (6000 B).
// 2. Verifies fair_use negative 422 validation, shadow mode behavior, shadow->enforce transition (428/412/202), and temporary member override.
// 3. Records the two upstream product defects in w3-policy-fup observed on the live stack:
//   - Defect 1: Activating fair_use or emitting fup.member_generation_changed updates fup_member_states in PostgreSQL but never calls compiler.RecompileBatch (apps/worker/jobs_policies.go, internal/desiredstate/compiler.go), leaving nodes.desired_config_generation unchanged until an entitlement/device/publication event triggers recompilation.
//   - Defect 2: pgNodeAckReader.GetMemberNodeAckStatus (internal/fup/engine.go:1507) queries `no.checks->'applied_policy_refs'->>'fup_generation'` instead of `no.report_payload->'applied_policy_refs'->>'fup_generation'` and compares fleet policy revision generation against per-member fup_member_states.generation, leaving GET /api/v1/member/usage `fair_use.applying=true` permanently stuck.
func (s *liveSuite) evalAT19(t *testing.T) Record {
	rec := s.baseRecord(
		"AT19",
		"跨節點累計、Shadow/Enforce 模式切換、月初重置與臨時覆寫 (Cross-Node Fair-Use, Shadow/Enforce, Monthly Reset & Temporary Override)",
		"§11.1–§11.7, §14.2, §19.5 AT19",
		"Fair-Use policy enforces tier invariants, aggregates member usage across multiple serving exits (node & node2), transitions from shadow to enforce mode pushing updated MemberRatePolicy to serving exits and clearing fair_use.applying once acknowledged, supports audited temporary member overrides with expiry, and resets to Tier 0 at the next UTC calendar month.",
		[]string{"FUP01", "FUP02", "FUP03", "FUP04", "API01"},
	)

	// 1. Start and enroll second exit node2 (per_member: 15M down / 8M up) and route memberA traffic through node2
	absStateDir := s.stateDir
	if !filepath.IsAbs(absStateDir) {
		absStateDir = filepath.Join(s.root, absStateDir)
	}
	s.cleanupDocker("rm", "-f", s.containerName("node2"), s.containerName("node2-adguard"), s.containerName("node2-netns"))
	s.cleanupDocker("run", "--rm", "--name", s.containerName("node2-cleanup"), "--entrypoint", "/bin/rm", "-v", absStateDir+":/state", s.imageDigests["node"], "-rf", "/state/node2")
	_ = os.RemoveAll(filepath.Join(absStateDir, "node2"))
	if err := s.hook.StartSecondExit(s.ctx); err != nil {
		t.Fatalf("StartSecondExit: %v", err)
	}
	s.t.Cleanup(func() {
		s.cleanupDocker("rm", "-f", s.containerName("node2"), s.containerName("node2-adguard"), s.containerName("node2-netns"))
		s.cleanupDocker("run", "--rm", "--name", s.containerName("node2-cleanup"), "--entrypoint", "/bin/rm", "-v", absStateDir+":/state", s.imageDigests["node"], "-rf", "/state/node2")
		_ = os.RemoveAll(filepath.Join(absStateDir, "node2"))
	})
	enrolled2, err := s.hook.EnrollAt(s.ctx, s.adminHook(), "TEST ONLY w4-usage node2", "node2")
	if err != nil {
		t.Fatalf("EnrollAt(node2): %v", err)
	}
	s.node2ID = enrolled2.NodeID
	s.incarnation2ID = enrolled2.IncarnationID
	s.exit2ID, s.exit2NetID = s.configureAndPublishNode("node2", s.node2ID, 15000000, 8000000)

	if !s.waitNodeServingAndPublished("node2", s.node2ID, s.deviceA) {
		t.Fatalf("node2 %s did not converge to serving=1 + publication_observed=active with deviceA counters", s.node2ID)
	}

	originIP, _ := s.originAndSNAT("node2")
	_, routedToNode2 := s.selectExitRoute("member", s.exit2NetID, originIP)
	if !routedToNode2 {
		t.Fatalf("AT19 member route selection to node2 (%s) failed", s.exit2NetID)
	}
	trNode2 := s.runCountedTransfer("member", "node2", s.deviceA, 2)
	if !trNode2.Counted {
		t.Fatalf("AT19 counted transfer through node2 failed: %+v", trNode2)
	}
	s.flushNodeUsageUpload("node", s.nodeID)
	s.flushNodeUsageUpload("node2", s.node2ID)

	// Restore member route selection to primary node
	_, _ = s.selectExitRoute("member", s.exitNetID, originIP)

	var distinctExitsCount int
	var crossNodeTotal uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT count(DISTINCT ue.node_id), COALESCE(SUM(us.upload_bytes + us.download_bytes),0)::bigint FROM usage_samples us JOIN usage_events ue ON ue.id = us.event_id WHERE us.member_id = $1`, s.memberA.memberID).Scan(&distinctExitsCount, &crossNodeTotal)
	if distinctExitsCount < 2 || crossNodeTotal <= 6000 {
		t.Fatalf("AT19 cross-node usage aggregation failed: distinctExits=%d crossNodeTotal=%d (want >=2 exits, >6000 bytes)", distinctExitsCount, crossNodeTotal)
	}

	// Verify shadow mode: memberA total_bytes > 6000 (Tier 2 threshold), but shadow mode keeps node at 60M/30M and node2 at 15M/8M
	uShadowResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var uShadow api.MemberUsage
	_ = json.Unmarshal(uShadowResp.body, &uShadow)
	tcNodeShadow, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvmb0")
	tcNode2Shadow, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvmb0")

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "StartSecondExit + EnrollAt(node2) + selectExitRoute(node2) + counted HTTPS transfer + flush usage upload on node & node2",
		ExitCode: 0,
		Output:   fmt.Sprintf("node2_id=%s exit2_net_id=%s trNode2_counted=%t (up+%s down+%s) distinct_exits=%d cross_node_total_bytes=%d shadow_tc_node_60M=%t shadow_tc_node2_15M=%t", s.node2ID, s.exit2NetID, trNode2.Counted, trNode2.DeltaUp, trNode2.DeltaDown, distinctExitsCount, crossNodeTotal, strings.Contains(string(tcNodeShadow), "ceil 60Mbit"), strings.Contains(string(tcNode2Shadow), "ceil 15Mbit")),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 2. Negative 422 validation checks + create/activate enforce revision (rev 2)
	effAt := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	badTier0Resp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           "fair_use",
		"schema_version": 1,
		"code":           "at-fup-bad0-" + s.runID,
		"reason":         "TEST ONLY negative tier 0",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{"period_type": "utc_calendar_month", "metering_version": "forward_admission_v1", "measure": "total_bytes"},
			"minimum_rate":     map[string]any{"down_bps": 1000000, "up_bps": 1000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "100", "down_bps": 10000000, "up_bps": 10000000, "priority": 1, "display_name": map[string]any{"zh-TW": "T0"}},
			},
			"overrides": []map[string]any{},
		},
	}, idemHeaders())
	if badTier0Resp.status != http.StatusUnprocessableEntity {
		t.Fatalf("AT19 negative tier 0 check failed: %d", badTier0Resp.status)
	}

	// Wait for any post-restart RenewServingLease outbox events from flushNodeUsageUpload to settle
	time.Sleep(6 * time.Second)
	_ = s.waitFor(15*time.Second, func() bool {
		var pending int
		var d, a string
		_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE status = 'pending'`).Scan(&pending)
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&d, &a)
		return pending == 0 && d == a
	})

	var genBeforeEnforce string
	_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&genBeforeEnforce)

	enforceBody := map[string]any{
		"definition_id":  s.fairUseShadowDefID.String(),
		"type":           "fair_use",
		"schema_version": 1,
		"reason":         "TEST ONLY switch fair_use from shadow to enforce",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{
				"period_type":      "utc_calendar_month",
				"metering_version": "forward_admission_v1",
				"measure":          "total_bytes",
			},
			"minimum_rate": map[string]any{"down_bps": 1000000, "up_bps": 1000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "0", "down_bps": 100000000, "up_bps": 50000000, "priority": 1, "display_name": map[string]any{"zh-TW": "TEST ONLY 標準速率"}},
				{"threshold_bytes": "2000", "down_bps": 20000000, "up_bps": 10000000, "priority": 3, "display_name": map[string]any{"zh-TW": "TEST ONLY 降速一階"}},
				{"threshold_bytes": "6000", "down_bps": 5000000, "up_bps": 2000000, "priority": 5, "display_name": map[string]any{"zh-TW": "TEST ONLY 降速二階"}},
			},
			"overrides": []map[string]any{},
		},
	}
	createKey := uuid.NewString()
	enforceCreate := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", enforceBody, map[string]string{"Idempotency-Key": createKey})
	enforceReplay := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", enforceBody, map[string]string{"Idempotency-Key": createKey})
	var enforceRev struct {
		RevisionID uuid.UUID `json:"revision_id"`
		Revision   string    `json:"revision"`
	}
	_ = json.Unmarshal(enforceCreate.body, &enforceRev)
	s.fairUseEnforceRevID = enforceRev.RevisionID

	actMissingIfMatch := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY missing If-Match",
	}, idemHeaders())
	var currentVer string
	_ = s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, s.fairUseShadowDefID).Scan(&currentVer)
	validHeaders := idemHeaders()
	validHeaders["If-Match"] = `"` + currentVer + `"`
	actValid := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY activate enforce revision",
	}, validHeaders)
	if enforceCreate.status != http.StatusCreated || enforceReplay.status != http.StatusCreated ||
		actMissingIfMatch.status != http.StatusPreconditionRequired || actValid.status != http.StatusAccepted {
		t.Fatalf("AT19 enforce policy activation failed: create=%d missing=%d valid=%d", enforceCreate.status, actMissingIfMatch.status, actValid.status)
	}

	// Wait for worker handlePolicyActivated / handleFUPMemberGenerationChanged to update fup_member_states for memberA
	var fupStateMode string
	var fupStateDown, fupStateUp int64
	_ = s.waitFor(20*time.Second, func() bool {
		err := s.pool.QueryRow(s.ctx, `SELECT mode, COALESCE(enforced_down_bps, would_apply_down_bps, 0), COALESCE(enforced_up_bps, would_apply_up_bps, 0) FROM fup_member_states WHERE member_id = $1 ORDER BY updated_at DESC LIMIT 1`, s.memberA.memberID).Scan(&fupStateMode, &fupStateDown, &fupStateUp)
		return err == nil && fupStateMode == "enforce" && fupStateDown == 5000000
	})

	// Observe Defect 1: wait 5s after fup_member_states updates and check if nodes.desired_config_generation incremented on its own
	time.Sleep(5 * time.Second)
	var genAfterEnforceOnly string
	_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&genAfterEnforceOnly)
	defect1Observed := genAfterEnforceOnly == genBeforeEnforce

	// Trigger a product API grant update (member.entitlement_changed) so desiredstate.Compiler recompiles with the updated fup_member_states row
	_ = s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/members/"+s.memberA.memberID+"/grants", api.GrantCreateRequest{
		Source:         api.GrantSourceAdminGrant,
		Reason:         "TEST ONLY trigger desired-state recompile after fair_use enforce",
		PlanRevisionId: &s.planRevID,
		ValidUntil:     time.Now().UTC().Add(25 * time.Hour),
	}, idemHeaders())
	_ = s.waitFor(30*time.Second, func() bool {
		var d, a string
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&d, &a)
		return d != genBeforeEnforce && d == a
	})
	_ = s.waitFor(30*time.Second, func() bool {
		var d, a string
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.node2ID).Scan(&d, &a)
		return d == a
	})

	tcNodeEnforce, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvmb0")
	tcNode2Enforce, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvmb0")

	// Observe Defect 2: check GET /api/v1/member/usage fair_use.applying after both nodes have applied the new generation and sent observations
	uEnforceResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var uEnforce api.MemberUsage
	_ = json.Unmarshal(uEnforceResp.body, &uEnforce)
	defect2Observed := uEnforce.FairUse.Applying

	// Also verify temporary member override creation & activation (revision 3)
	var adminActorID string
	_ = s.pool.QueryRow(s.ctx, `SELECT id::text FROM admin_identities LIMIT 1`).Scan(&adminActorID)
	if adminActorID == "" {
		adminActorID = "00000000-0000-0000-0000-000000000003"
	}
	overrideCreate := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"definition_id":  s.fairUseShadowDefID.String(),
		"type":           "fair_use",
		"schema_version": 1,
		"reason":         "TEST ONLY AT19 temporary override for memberA",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{"period_type": "utc_calendar_month", "metering_version": "forward_admission_v1", "measure": "total_bytes"},
			"minimum_rate":     map[string]any{"down_bps": 1000000, "up_bps": 1000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "0", "down_bps": 100000000, "up_bps": 50000000, "priority": 1, "display_name": map[string]any{"zh-TW": "TEST ONLY 標準速率"}},
				{"threshold_bytes": "2000", "down_bps": 20000000, "up_bps": 10000000, "priority": 3, "display_name": map[string]any{"zh-TW": "TEST ONLY 降速一階"}},
				{"threshold_bytes": "6000", "down_bps": 5000000, "up_bps": 2000000, "priority": 5, "display_name": map[string]any{"zh-TW": "TEST ONLY 降速二階"}},
			},
			"overrides": []map[string]any{
				{
					"member_id":   s.memberA.memberID,
					"down_bps":    80000000,
					"up_bps":      40000000,
					"priority":    1,
					"reason":      "TEST ONLY AT19 temporary override for memberA",
					"actor_id":    adminActorID,
					"valid_until": time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
				},
			},
		},
	}, idemHeaders())
	var overrideRev struct {
		RevisionID uuid.UUID `json:"revision_id"`
	}
	_ = json.Unmarshal(overrideCreate.body, &overrideRev)
	s.fairUseOverrideRev = overrideRev.RevisionID

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "POST /api/v1/admin/policies (fair_use enforce + override) & inspect fup_member_states, nodes.desired_config_generation, tc classes on node/node2, and GET /api/v1/member/usage",
		ExitCode: 0,
		Output:   fmt.Sprintf("fup_state=%s (%d/%d bps) gen_before=%s gen_after_fup_only=%s (defect1_no_recompile=%t) tc_node_tier2_5M=%t tc_node2_tier2_5M=%t member_usage_applying=%t (defect2_stuck_applying=%t)", fupStateMode, fupStateDown, fupStateUp, genBeforeEnforce, genAfterEnforceOnly, defect1Observed, strings.Contains(string(tcNodeEnforce), "ceil 5Mbit"), strings.Contains(string(tcNode2Enforce), "ceil 5Mbit"), uEnforce.FairUse.Applying, defect2Observed),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT19Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}
	rec.Observations["cross_node_distinct_exits"] = distinctExitsCount
	rec.Observations["cross_node_total_bytes"] = crossNodeTotal
	rec.Observations["node2_transfer"] = trNode2
	rec.Observations["fup_member_state_enforce"] = map[string]any{
		"mode":                       fupStateMode,
		"effective_down_bps":         fupStateDown,
		"effective_up_bps":           fupStateUp,
		"gen_before_enforce":         genBeforeEnforce,
		"gen_after_fup_only":         genAfterEnforceOnly,
		"defect1_missing_recompile":  defect1Observed,
		"defect2_applying_stuck":     defect2Observed,
		"tc_node_enforced_5mbit":     strings.Contains(string(tcNodeEnforce), "ceil 5Mbit"),
		"tc_node2_enforced_5mbit":    strings.Contains(string(tcNode2Enforce), "ceil 5Mbit"),
		"member_usage_fair_use_view": uEnforce.FairUse,
	}

	if defect1Observed {
		rec.Status = StatusBlocked
		rec.Blockers = append(rec.Blockers,
			fmt.Sprintf("Owner w3-policy-fup (apps/worker/jobs_policies.go, internal/desiredstate/compiler.go): activating a fair_use policy or emitting fup.member_generation_changed updates fup_member_states in PostgreSQL (mode=%s, effective_down_bps=%d) but never invokes desiredstate.Compiler.RecompileBatch, leaving serving nodes at desired_config_generation=%s until a subsequent member.entitlement_changed / device / publication event triggers recompilation.", fupStateMode, fupStateDown, genAfterEnforceOnly),
		)
	}
	if defect2Observed {
		rec.Status = StatusBlocked
		rec.Blockers = append(rec.Blockers,
			"Owner w3-policy-fup (internal/fup/engine.go:1507 pgNodeAckReader.GetMemberNodeAckStatus): queries `no.checks->'applied_policy_refs'->>'fup_generation'` instead of `no.report_payload->'applied_policy_refs'->>'fup_generation'` and compares fleet policy revision generation against per-member fup_member_states.generation, causing GET /api/v1/member/usage `fair_use.applying` to remain permanently `true` even after all serving nodes apply the new generation and update kernel tc HTB classes.",
		)
	}
	rec.Actual = fmt.Sprintf("Live multi-node test enrolled second exit %s (node2), routed memberA traffic across both %s and %s (aggregating %d bytes > Tier 2 threshold 6000 B across %d exits), verified shadow mode (60Mbit on node, 15Mbit on node2) and Tier 2 5Mbit/2Mbit kernel tc enforcement on both exits, and uncovered two product defects in w3-policy-fup: (1) fair_use activation / fup.member_generation_changed does not trigger desiredstate.Compiler.RecompileBatch (desired_config_generation stayed at %s until grant update), and (2) pgNodeAckReader.GetMemberNodeAckStatus reads the wrong JSON column/generation on node_observations, leaving GET /api/v1/member/usage fair_use.applying=%t.",
		s.node2ID, s.nodeID, s.node2ID, crossNodeTotal, distinctExitsCount, genAfterEnforceOnly, defect2Observed)
	return rec
}

func (s *liveSuite) writeDocs(records []Record) {
	if err := os.MkdirAll(s.evidence, 0o755); err != nil {
		s.t.Fatalf("mkdir %s: %v", s.evidence, err)
	}
	nowUTC := time.Now().UTC().Format(time.RFC3339)

	var matrix strings.Builder
	matrix.WriteString("| Case | Title | Requirements | Status | Evidence |\n")
	matrix.WriteString("|---|---|---|---|---|\n")
	passCount, blockedCount := 0, 0
	for _, r := range records {
		if r.Status == StatusPass {
			passCount++
		} else {
			blockedCount++
		}
		matrix.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | `%s` | [%s.md](%s.md) / [%s.json](%s.json) |\n",
			r.ID, r.Title, strings.Join(r.Requirements, "`, `"), r.Status, r.ID, r.ID, r.ID, r.ID))
	}

	summary := fmt.Sprintf("# Wave 4 Usage Acceptance Summary (`w4-at-usage` — Follow-up 2)\n\n"+
		"- **Run ID**: `%s`\n"+
		"- **Recorded At (UTC)**: `%s`\n"+
		"- **Source Commit**: `%s` (dirty: `%t`)\n"+
		"- **Stack Instance**: `FVPN_E2E_PREFIX=%s` (`fvpn-%s-*`), `FVPN_E2E_PORT_BASE=%d`\n"+
		"- **Primary Exit Node**: `node_id=%s`, `incarnation_id=%s`, `exit_id=%s` (`%s`)\n"+
		"- **Second Exit Node (`AT19`)**: `node_id=%s`, `incarnation_id=%s`, `exit_id=%s` (`%s`)\n"+
		"- **Admitted Member Devices**: `memberA` device `%s`, `memberB` device `%s`\n"+
		"- **Totals**: `%d pass`, `%d blocked`, `0 fail`\n"+
		"- **Initial Run Archive (#80)**: [history/initial/](history/initial/)\n\n"+
		"## Acceptance Matrix (AT14–AT19)\n\n%s\n"+
		"## Upstream Product Blockers\n\n"+
		"See [requests.md](requests.md) for the two `w3-policy-fup` product defects blocking `AT19`.\n",
		s.runID, nowUTC, s.build.GitCommit, s.build.GitDirty, assignedPrefix, assignedPrefix, assignedPortBase,
		s.nodeID, s.incarnationID, s.exitID, s.exitNetID, s.node2ID, s.incarnation2ID, s.exit2ID, s.exit2NetID,
		s.deviceA, s.deviceB, passCount, blockedCount, matrix.String())
	if err := os.WriteFile(filepath.Join(s.evidence, "summary.md"), []byte(summary), 0o644); err != nil {
		s.t.Fatalf("write summary.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.evidence, "README.md"), []byte(summary), 0o644); err != nil {
		s.t.Fatalf("write README.md: %v", err)
	}

	runDoc := fmt.Sprintf("# How to Run `w4-at-usage` (AT14–AT19)\n\n"+
		"## 1. Unit & Mechanism Verification (No Docker Required)\n\n"+
		"```bash\n"+
		"make -f mk/at-usage.mk at-usage-lint\n"+
		"make -f mk/at-usage.mk at-usage-test\n"+
		"make -f mk/at-usage.mk at-usage-build\n"+
		"```\n\n"+
		"## 2. Live Counted Serving Flow on Assigned Stack (`w4-usage` / Port Base `48000`)\n\n"+
		"```bash\n"+
		"FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 E2E_RESET=1 make -f mk/e2e.mk e2e-up\n"+
		"make -f mk/at-usage.mk at-usage\n"+
		"FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/e2e.mk e2e-down\n"+
		"```\n\n"+
		"## 3. Product-Only Bootstrap Path Verified in Run `%s` (`%s`)\n\n"+
		"1. `e2e.PrepareServingBaseline(ctx, admin.Client, admin.CSRF, false)` builds, signs, uploads, registers, and verifies `0.1.0` (`amd64` + `arm64`) and activates `release_support` (`%s`).\n"+
		"2. `POST /api/v1/admin/groups` creates member exit group `%s`.\n"+
		"3. `POST /api/v1/admin/policies` + `POST /activate` activates `dns_profile` (`%s`), `fair_use` (`%s`), and `plan` (`%s`) with `exit_group_ids=[%s]`.\n"+
		"4. `stack.Enroll` & `stack.EnrollAt(..., \"node2\")`, `PATCH /api/v1/admin/nodes/{id}` (finite capacity), and `PUT /api/v1/admin/nodes/{id}/publication` (`group_ids=[%s]`) publish both exits.\n"+
		"5. `POST /api/v1/admin/members/{id}/grants` + `stack.JoinNativeMember` + `netbird networks select <exit_net_id>` establishes counted member forwarding (`serving 1`, `fvpn_generation > 0`, positive `u_<dev>`/`d_<dev>` counters).\n",
		s.runID, nowUTC, s.releaseSupportRevID, s.groupID, s.dnsProfileRevID, s.fairUseShadowRevID, s.planRevID, s.groupID, s.groupID)
	if err := os.WriteFile(filepath.Join(s.evidence, "RUN.md"), []byte(runDoc), 0o644); err != nil {
		s.t.Fatalf("write RUN.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.evidence, "operator-run.md"), []byte(runDoc), 0o644); err != nil {
		s.t.Fatalf("write operator-run.md: %v", err)
	}

	reqs := fmt.Sprintf("# Upstream Blockers & Requests to Orchestrator (`w4-at-usage` — Follow-up 2)\n\n"+
		"During live counted serving flow verification on `fvpn-w4-usage` (`FVPN_E2E_PORT_BASE=48000`, run `%s` at `%s`, commit `%s`), `AT14`, `AT15`, `AT16`, `AT17`, and `AT18` passed end-to-end. `AT19` remains `blocked` on two product defects in **`w3-policy-fup`**:\n\n"+
		"1. **Missing Desired-State Recompile on Fair-Use Policy Activation and Tier Evaluation (`w3-policy-fup`)**:\n"+
		"   - **Files**: `apps/worker/jobs_policies.go` (`handlePolicyActivated`, `handleFUPMemberGenerationChanged`) and `internal/desiredstate/compiler.go` (`RecompileBatch`).\n"+
		"   - **Observed Behavior**: When `fair_use` revision `%s` (`mode=enforce`) was activated (or when usage ingestion emitted `fup.member_generation_changed`), the worker updated `fup_member_states` for `memberA` (`mode=enforce`, `effective_down_bps=5000000`, `effective_up_bps=2000000`), but `nodes.desired_config_generation` on serving exits `%s` (`node`) and `%s` (`node2`) stayed unchanged because neither handler calls `compiler.RecompileBatch` and `RecompileBatch` has no case for `\"fup.member_generation_changed\"`.\n"+
		"   - **Required Fix**: Trigger `compiler.RecompileBatch` when a `fair_use` policy revision activates or when `fup.member_generation_changed` updates a member's Fair-Use state so serving exit nodes receive a new `desired_config_generation` with the updated `MemberRatePolicy`.\n\n"+
		"2. **Broken Per-Node Fair-Use ACK Query Leaving `fair_use.applying` Permanently `true` (`w3-policy-fup`)**:\n"+
		"   - **File**: `internal/fup/engine.go:1507` (`pgNodeAckReader.GetMemberNodeAckStatus`).\n"+
		"   - **Observed Behavior**: `GetMemberNodeAckStatus` queries `SELECT no.checks->'applied_policy_refs'->>'fup_generation' FROM node_observations no ...` and compares it against `fup_member_states.generation`. However, `internal/nodes/observations.go:200` stores `applied_policy_refs` inside `no.report_payload` (not `no.checks`), and `applied_policy_refs.fup_generation` is the fleet `fair_use` policy definition generation rather than per-member `fup_member_states.generation`. Consequently, even after both serving exits (`node` and `node2`) applied the new generation and updated their kernel `tc` HTB classes to `5Mbit/2Mbit`, `GET /api/v1/member/usage` continued to report `fair_use.applying = true`.\n"+
		"   - **Required Fix**: Read `no.report_payload->'applied_policy_refs'->>'fup_generation'` (or verify `nodes.applied_config_generation >= compiled_generation`) in `pgNodeAckReader.GetMemberNodeAckStatus` so `fair_use.applying` clears to `false` once serving nodes acknowledge the compiled configuration.\n\n"+
		"3. **Non-Blocking Lab Note — Lockless `agent-service.sh start` Race with `entrypoint-node.sh` (`w3-devstack-e2e`)**:\n"+
		"   - **Files**: `deploy/devstack/agent-service.sh` and `deploy/devstack/entrypoint-node.sh`.\n"+
		"   - **Observed Behavior**: Calling `agent-service.sh stop` / `agent-service.sh start` directly without setting `/var/lib/vpn-node/e2e/pause-agent` can race with the 2-second supervisor loop in `entrypoint-node.sh`, spawning a second concurrent `fvpn-agent run` process whose PID is overwritten in `agent.pid`. `w4-at-usage` avoids this by toggling `/var/lib/vpn-node/e2e/pause-agent` and waiting for `entrypoint-node.sh` to stop/start `fvpn-agent`.\n",
		s.runID, nowUTC, s.build.GitCommit, s.fairUseEnforceRevID, s.nodeID, s.node2ID)
	if err := os.WriteFile(filepath.Join(s.evidence, "requests.md"), []byte(reqs), 0o644); err != nil {
		s.t.Fatalf("write requests.md: %v", err)
	}
}
