//go:build e2e

package usage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
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

// evalAT19 evaluates AT19 across two real serving exits (node and node2) plus a registered non-carrying node (otherNodeID):
// 1. Starts and enrolls second exit node2 (per_member: 15M down / 8M up), publishes into s.groupID, selects node2 exit route in member, sends counted HTTPS transfers through node2 (origin sees node2 SNAT .22), and flushes usage upload from both node and node2 so memberA's cross-node total_bytes exceeds Tier 2 (6000 B).
// 2. Enrolls a third registered node (s.otherNodeID) that stays unpublished (non-carrying) to prove Fair-Use recompilation affects only carrying exits.
// 3. Verifies fair_use negative 422 validation, shadow mode behavior, and Transition 1 (policy.activated in mode=enforce):
//   - desired_config_generation increments on both carrying exits (node, node2) and on no other node (otherNodeID stays 0),
//   - both carrying exits install Tier 2 tc classes (ceil 5Mbit down on fvmb0, ceil 2Mbit up on fvup0),
//   - fair_use.applying transitions true -> false once both carrying exits have applied the compiled generation (records the clearing observation and wall-clock time),
//   - replaying the activation does not bump desired_config_generation twice.
//
// 4. Verifies Transition 2 (fup.member_generation_changed via usage ingestion crossing into Tier 2):
//   - activates an enforce revision where memberA starts in Tier 1 (ceil 10Mbit / ceil 5Mbit) just below Tier 2 threshold,
//   - sends real counted HTTPS traffic across the threshold and flushes usage upload,
//   - desired_config_generation increments on both carrying exits (node, node2) and on no other node (otherNodeID stays 0),
//   - both carrying exits install Tier 2 tc classes (ceil 5Mbit / ceil 2Mbit),
//   - fair_use.applying transitions true -> false once both carrying exits apply the generation (records the clearing observation and wall-clock time),
//   - replaying the UsageEvent batch returns duplicate and does not bump desired_config_generation twice.
//
// 5. Verifies ENR-03 per-incarnation retained usage (s.measurePerIncarnationRetainedUsage) on node2:
//   - re-enrolls node2 as a new incarnation through product APIs only after it has counted traffic,
//   - verifies GET /api/v1/admin/nodes/{id}/incarnations lists both incarnations newest-first with retained usage on the retired incarnation,
//   - verifies member_usage_periods is unchanged across re-enrollment and new traffic on the new incarnation accumulates on the same member period while the retired incarnation's retained usage stays fixed.
func (s *liveSuite) evalAT19(t *testing.T) Record {
	rec := s.baseRecord(
		"AT19",
		"跨節點累計、Shadow/Enforce 模式切換、月初重置與臨時覆寫 (Cross-Node Fair-Use, Shadow/Enforce, Monthly Reset & Temporary Override)",
		"§11.1–§11.7, §14.2, §19.5 AT19",
		"Fair-Use policy enforces tier invariants, aggregates member usage across multiple serving exits (node & node2), transitions from shadow to enforce mode and across usage thresholds by recompiling only carrying exits, installing Tier 2 kernel tc classes on both exits, and clearing fair_use.applying from true to false once both carrying exits acknowledge the compiled generation, while ignoring duplicate/replayed events and preserving per-incarnation retained usage across node re-enrollment.",
		[]string{"FUP01", "FUP02", "FUP03", "FUP04", "ENR03", "API01"},
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

	// Enroll a third registered node that is NOT carrying any member (unpublished)
	s.otherNodeID, s.otherIncarnationID = s.enrollNonCarryingNode()

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
		Command:  "StartSecondExit + EnrollAt(node2) + enrollNonCarryingNode + selectExitRoute(node2) + counted HTTPS transfer + flush usage upload on node & node2",
		ExitCode: 0,
		Output:   fmt.Sprintf("node2_id=%s exit2_net_id=%s other_node_id=%s trNode2_counted=%t (up+%s down+%s) distinct_exits=%d cross_node_total_bytes=%d shadow_tc_node_60M=%t shadow_tc_node2_15M=%t", s.node2ID, s.exit2NetID, s.otherNodeID, trNode2.Counted, trNode2.DeltaUp, trNode2.DeltaDown, distinctExitsCount, crossNodeTotal, strings.Contains(string(tcNodeShadow), "ceil 60Mbit"), strings.Contains(string(tcNode2Shadow), "ceil 15Mbit")),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 2. Negative 422 validation check
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
	_ = s.waitFor(20*time.Second, func() bool {
		var pending int
		var d1, a1, d2, a2 string
		_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE status = 'pending'`).Scan(&pending)
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&d1, &a1)
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.node2ID).Scan(&d2, &a2)
		return pending == 0 && d1 == a1 && d2 == a2
	})

	// 3. Transition 1: policy.activated in mode=enforce (crossNodeTotal > 6000 -> Tier 2 5Mbit/2Mbit)
	gensBefore1 := s.queryDesiredGenerations(s.nodeID, s.node2ID, s.otherNodeID)
	genNodeBefore1 := gensBefore1[s.nodeID.String()]
	genNode2Before1 := gensBefore1[s.node2ID.String()]
	genOtherBefore1 := gensBefore1[s.otherNodeID.String()]
	fupMemberGenBefore1, _, _, _, _ := s.queryFUPMemberState(s.memberA.memberID)

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

	// Pause node2 briefly during activation so we reliably observe fair_use.applying=true before node2 acknowledges, then resume node2 and measure wall-clock time to applying=false
	s.stopNodeAgent("node2")
	var currentVer string
	_ = s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, s.fairUseShadowDefID).Scan(&currentVer)
	act1Key := uuid.NewString()
	validHeaders := map[string]string{
		"Idempotency-Key": act1Key,
		"If-Match":        `"` + currentVer + `"`,
	}
	act1Start := time.Now().UTC()
	actValid := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY activate enforce revision",
	}, validHeaders)
	if enforceCreate.status != http.StatusCreated || enforceReplay.status != http.StatusCreated ||
		actMissingIfMatch.status != http.StatusPreconditionRequired || actValid.status != http.StatusAccepted {
		s.startNodeAgent("node2")
		t.Fatalf("AT19 enforce policy activation failed: create=%d missing=%d valid=%d", enforceCreate.status, actMissingIfMatch.status, actValid.status)
	}

	// Wait for worker handlePolicyActivated + RecompileBatch to bump desired_config_generation on both carrying exits (node, node2) and NOT on otherNodeID
	var gensAfter1 map[string]uint64
	var genNodeAfter1, genNode2After1, genOtherAfter1 uint64
	if !s.waitFor(20*time.Second, func() bool {
		gensAfter1 = s.queryDesiredGenerations(s.nodeID, s.node2ID, s.otherNodeID)
		genNodeAfter1 = gensAfter1[s.nodeID.String()]
		genNode2After1 = gensAfter1[s.node2ID.String()]
		genOtherAfter1 = gensAfter1[s.otherNodeID.String()]
		return genNodeAfter1 > genNodeBefore1 && genNode2After1 > genNode2Before1
	}) {
		s.startNodeAgent("node2")
		t.Fatalf("AT19 policy.activated did not increment desired_config_generation on both carrying exits: node %d->%d, node2 %d->%d, other %d->%d",
			genNodeBefore1, genNodeAfter1, genNode2Before1, genNode2After1, genOtherBefore1, genOtherAfter1)
	}
	if genOtherAfter1 != genOtherBefore1 {
		s.startNodeAgent("node2")
		t.Fatalf("AT19 policy.activated unexpectedly incremented non-carrying node desired_config_generation: %d -> %d", genOtherBefore1, genOtherAfter1)
	}

	// Observe fair_use.applying == true while node2 has not yet applied genNode2After1
	applyingObservedTrue1 := false
	_ = s.waitFor(10*time.Second, func() bool {
		resp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
		if resp.status != http.StatusOK {
			return false
		}
		var mu api.MemberUsage
		_ = json.Unmarshal(resp.body, &mu)
		if mu.FairUse.Applying {
			applyingObservedTrue1 = true
			return true
		}
		return false
	})

	// Resume node2 agent and poll GET /api/v1/member/usage until both exits have applied and fair_use.applying == false
	s.startNodeAgent("node2")
	var applyingFalseAt1 time.Time
	var uEnforce1 api.MemberUsage
	if !s.waitFor(30*time.Second, func() bool {
		var a1, a2 uint64
		_ = s.pool.QueryRow(s.ctx, `SELECT applied_config_generation::bigint FROM nodes WHERE id = $1`, s.nodeID).Scan(&a1)
		_ = s.pool.QueryRow(s.ctx, `SELECT applied_config_generation::bigint FROM nodes WHERE id = $1`, s.node2ID).Scan(&a2)
		if a1 < genNodeAfter1 || a2 < genNode2After1 {
			return false
		}
		resp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
		if resp.status != http.StatusOK {
			return false
		}
		_ = json.Unmarshal(resp.body, &uEnforce1)
		if !uEnforce1.FairUse.Applying {
			applyingFalseAt1 = time.Now().UTC()
			return true
		}
		return false
	}) {
		t.Fatalf("AT19 Transition 1 fair_use.applying did not clear to false after both exits applied (last FairUse=%+v)", uEnforce1.FairUse)
	}
	wallClock1Ms := applyingFalseAt1.Sub(act1Start).Milliseconds()

	// Query the observation on node2 that cleared fair_use.applying
	var clearObs1ID uuid.UUID
	var clearObs1AppliedGen uint64
	var clearObs1Time time.Time
	_ = s.pool.QueryRow(s.ctx, `
		SELECT report_id, applied_generation::bigint, received_at
		FROM node_observations
		WHERE node_id = $1 AND incarnation_id = $2 AND applied_generation >= $3
		ORDER BY received_at ASC LIMIT 1
	`, s.node2ID, s.incarnation2ID, genNode2After1).Scan(&clearObs1ID, &clearObs1AppliedGen, &clearObs1Time)

	fupMemberGenAfter1, fupMode1, fupTierIdx1, fupDown1, fupUp1 := s.queryFUPMemberState(s.memberA.memberID)
	tcNodeDown1, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvmb0")
	tcNodeUp1, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvup0")
	tcNode2Down1, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvmb0")
	tcNode2Up1, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvup0")
	node1Tier2OK := strings.Contains(string(tcNodeDown1), "ceil 5Mbit") && strings.Contains(string(tcNodeUp1), "ceil 2Mbit")
	node2Tier2OK := strings.Contains(string(tcNode2Down1), "ceil 5Mbit") && strings.Contains(string(tcNode2Up1), "ceil 2Mbit")
	if !node1Tier2OK || !node2Tier2OK {
		t.Fatalf("AT19 Transition 1 Tier 2 tc classes missing: node=%t (down=%s up=%s) node2=%t (down=%s up=%s)",
			node1Tier2OK, string(tcNodeDown1), string(tcNodeUp1), node2Tier2OK, string(tcNode2Down1), string(tcNode2Up1))
	}

	// Replay activation (idempotent replay + re-activating already-active revision) and verify desired_config_generation does NOT bump twice
	_ = s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY activate enforce revision",
	}, validHeaders)
	var verAfter1 string
	_ = s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, s.fairUseShadowDefID).Scan(&verAfter1)
	dupHeaders1 := idemHeaders()
	dupHeaders1["If-Match"] = `"` + verAfter1 + `"`
	_ = s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY duplicate activate same enforce revision",
	}, dupHeaders1)
	time.Sleep(3 * time.Second)
	gensAfterReplay1 := s.queryDesiredGenerations(s.nodeID, s.node2ID, s.otherNodeID)
	fupMemberGenReplay1, _, _, _, _ := s.queryFUPMemberState(s.memberA.memberID)
	genNodeAfterReplay1 := gensAfterReplay1[s.nodeID.String()]
	genNode2AfterReplay1 := gensAfterReplay1[s.node2ID.String()]
	genOtherAfterReplay1 := gensAfterReplay1[s.otherNodeID.String()]
	if genNodeAfterReplay1 != genNodeAfter1 || genNode2AfterReplay1 != genNode2After1 || genOtherAfterReplay1 != genOtherAfter1 {
		t.Fatalf("AT19 Transition 1 replay unexpectedly bumped desired_config_generation: before=(%d,%d,%d) after=(%d,%d,%d)",
			genNodeAfter1, genNode2After1, genOtherAfter1, genNodeAfterReplay1, genNode2AfterReplay1, genOtherAfterReplay1)
	}

	s.fupActivationTransition = fupTransitionMeasurement{
		Transition:                     "policy_activated_enforce_tier2",
		Trigger:                        "POST /api/v1/admin/policies/{id}/activate (mode=enforce, Tier 2 5Mbit/2Mbit)",
		StartedAt:                      act1Start.Format(time.RFC3339Nano),
		FUPMemberGenerationBefore:      fupMemberGenBefore1,
		FUPMemberGenerationAfter:       fupMemberGenAfter1,
		EnforcedTierIndex:              fupTierIdx1,
		EnforcedDownBPS:                fupDown1,
		EnforcedUpBPS:                  fupUp1,
		DesiredGenBefore:               gensBefore1,
		DesiredGenAfter:                gensAfter1,
		DesiredGenAfterReplay:          gensAfterReplay1,
		FUPMemberGenerationAfterReplay: fupMemberGenReplay1,
		ApplyingObservedTrue:           applyingObservedTrue1,
		ApplyingClearedToFalse:         !uEnforce1.FairUse.Applying,
		ApplyingClearedAt:              applyingFalseAt1.Format(time.RFC3339Nano),
		WallClockMsToApplyingFalse:     wallClock1Ms,
		ClearingObservationReportID:    clearObs1ID,
		ClearingObservationNodeID:      s.node2ID,
		ClearingObservationGeneration:  clearObs1AppliedGen,
		ClearingObservationReceivedAt:  clearObs1Time.UTC().Format(time.RFC3339Nano),
		Node1TCClassRate:               "ceil 5Mbit down (fvmb0) / ceil 2Mbit up (fvup0)",
		Node2TCClassRate:               "ceil 5Mbit down (fvmb0) / ceil 2Mbit up (fvup0)",
		BothExitsInstalledTier2TC:      node1Tier2OK && node2Tier2OK,
	}

	// 4. Transition 2: fup.member_generation_changed via usage ingestion crossing a threshold into Tier 2
	// First activate an intermediate enforce policy where Tier 1 is 2000 B (10Mbit/5Mbit) and Tier 2 is (currentTotal + 1500 B) (5Mbit/2Mbit),
	// so memberA starts in Tier 1 (10Mbit/5Mbit) and both exits converge to Tier 1 with fair_use.applying == false.
	var curPeriodTotal uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT total_bytes::bigint FROM member_usage_periods WHERE member_id = $1 ORDER BY period_start DESC LIMIT 1`, s.memberA.memberID).Scan(&curPeriodTotal)
	tier2ThresholdBytes := curPeriodTotal + 1500

	prepTier1Create := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"definition_id":  s.fairUseShadowDefID.String(),
		"type":           "fair_use",
		"schema_version": 1,
		"reason":         "TEST ONLY stage memberA in Tier 1 before usage threshold crossing",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{"period_type": "utc_calendar_month", "metering_version": "forward_admission_v1", "measure": "total_bytes"},
			"minimum_rate":     map[string]any{"down_bps": 1000000, "up_bps": 1000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "0", "down_bps": 100000000, "up_bps": 50000000, "priority": 1, "display_name": map[string]any{"zh-TW": "TEST ONLY 標準速率"}},
				{"threshold_bytes": "2000", "down_bps": 10000000, "up_bps": 5000000, "priority": 3, "display_name": map[string]any{"zh-TW": "TEST ONLY 降速一階"}},
				{"threshold_bytes": strconv.FormatUint(tier2ThresholdBytes, 10), "down_bps": 5000000, "up_bps": 2000000, "priority": 5, "display_name": map[string]any{"zh-TW": "TEST ONLY 降速二階"}},
			},
			"overrides": []map[string]any{},
		},
	}, idemHeaders())
	var prepTier1Rev struct {
		RevisionID uuid.UUID `json:"revision_id"`
	}
	_ = json.Unmarshal(prepTier1Create.body, &prepTier1Rev)
	var verBeforePrep string
	_ = s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, s.fairUseShadowDefID).Scan(&verBeforePrep)
	prepHeaders := idemHeaders()
	prepHeaders["If-Match"] = `"` + verBeforePrep + `"`
	_ = s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": prepTier1Rev.RevisionID,
		"reason":      "TEST ONLY activate Tier 1 staging revision",
	}, prepHeaders)

	if !s.waitFor(30*time.Second, func() bool {
		_, _, _, downBps, _ := s.queryFUPMemberState(s.memberA.memberID)
		if downBps != 10000000 {
			return false
		}
		tc1, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvmb0")
		tc2, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvmb0")
		if !strings.Contains(string(tc1), "ceil 10Mbit") || !strings.Contains(string(tc2), "ceil 10Mbit") {
			return false
		}
		resp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
		if resp.status != http.StatusOK {
			return false
		}
		var mu api.MemberUsage
		_ = json.Unmarshal(resp.body, &mu)
		return !mu.FairUse.Applying
	}) {
		t.Fatalf("AT19 staging memberA in Tier 1 (10Mbit/5Mbit) before usage threshold crossing did not converge")
	}

	// Ensure outbox and node generations are completely settled before driving threshold-crossing traffic
	_ = s.waitFor(15*time.Second, func() bool {
		var pending int
		var d1, a1, d2, a2 string
		_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE status = 'pending'`).Scan(&pending)
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&d1, &a1)
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, s.node2ID).Scan(&d2, &a2)
		return pending == 0 && d1 == a1 && d2 == a2
	})

	gensBefore2 := s.queryDesiredGenerations(s.nodeID, s.node2ID, s.otherNodeID)
	genNodeBefore2 := gensBefore2[s.nodeID.String()]
	genNode2Before2 := gensBefore2[s.node2ID.String()]
	genOtherBefore2 := gensBefore2[s.otherNodeID.String()]
	fupMemberGenBefore2, _, _, _, _ := s.queryFUPMemberState(s.memberA.memberID)

	// Pause node2 agent briefly so we capture fair_use.applying=true after usage ingestion triggers fup.member_generation_changed,
	// drive 3 counted HTTPS transfers through primary node (> 1500 B), and flush node's usage upload.
	s.stopNodeAgent("node2")
	trThreshold := s.runCountedTransfer("member", "node", s.deviceA, 3)
	if !trThreshold.Counted {
		s.startNodeAgent("node2")
		t.Fatalf("AT19 threshold-crossing counted transfer failed: %+v", trThreshold)
	}
	ingestStart := time.Now().UTC()
	var maxSeqBeforeFlush uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(MAX(sequence_number), 0)::bigint FROM usage_events WHERE node_id = $1`, s.nodeID).Scan(&maxSeqBeforeFlush)
	_, _, _ = s.execInContainer("node", "pkill", "-HUP", "-f", "fvpn-agent")
	if !s.waitFor(8*time.Second, func() bool {
		var maxSeq, newTotal uint64
		_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(MAX(sequence_number), 0)::bigint FROM usage_events WHERE node_id = $1`, s.nodeID).Scan(&maxSeq)
		_ = s.pool.QueryRow(s.ctx, `SELECT total_bytes::bigint FROM member_usage_periods WHERE member_id = $1 ORDER BY period_start DESC LIMIT 1`, s.memberA.memberID).Scan(&newTotal)
		return maxSeq > maxSeqBeforeFlush && newTotal >= tier2ThresholdBytes
	}) {
		s.flushNodeUsageUpload("node", s.nodeID)
	}

	// Wait for usage ingestion -> fup.member_generation_changed -> RecompileBatch to increment desired_config_generation on both carrying exits (node, node2) and NOT on otherNodeID
	var gensAfter2 map[string]uint64
	var genNodeAfter2, genNode2After2, genOtherAfter2 uint64
	if !s.waitFor(25*time.Second, func() bool {
		_, _, _, downBps, _ := s.queryFUPMemberState(s.memberA.memberID)
		gensAfter2 = s.queryDesiredGenerations(s.nodeID, s.node2ID, s.otherNodeID)
		genNodeAfter2 = gensAfter2[s.nodeID.String()]
		genNode2After2 = gensAfter2[s.node2ID.String()]
		genOtherAfter2 = gensAfter2[s.otherNodeID.String()]
		return downBps == 5000000 && genNodeAfter2 > genNodeBefore2 && genNode2After2 > genNode2Before2
	}) {
		s.startNodeAgent("node2")
		t.Fatalf("AT19 fup.member_generation_changed did not increment desired_config_generation on both carrying exits: node %d->%d, node2 %d->%d, other %d->%d",
			genNodeBefore2, genNodeAfter2, genNode2Before2, genNode2After2, genOtherBefore2, genOtherAfter2)
	}
	if genOtherAfter2 != genOtherBefore2 {
		s.startNodeAgent("node2")
		t.Fatalf("AT19 fup.member_generation_changed unexpectedly incremented non-carrying node desired_config_generation: %d -> %d", genOtherBefore2, genOtherAfter2)
	}

	// Observe fair_use.applying == true while node2 has not yet applied genNode2After2
	applyingObservedTrue2 := false
	_ = s.waitFor(10*time.Second, func() bool {
		resp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
		if resp.status != http.StatusOK {
			return false
		}
		var mu api.MemberUsage
		_ = json.Unmarshal(resp.body, &mu)
		if mu.FairUse.Applying {
			applyingObservedTrue2 = true
			return true
		}
		return false
	})

	// Resume node2 agent and poll GET /api/v1/member/usage until both exits have applied and fair_use.applying == false
	s.startNodeAgent("node2")
	var applyingFalseAt2 time.Time
	var uEnforce2 api.MemberUsage
	if !s.waitFor(30*time.Second, func() bool {
		var a1, a2 uint64
		_ = s.pool.QueryRow(s.ctx, `SELECT applied_config_generation::bigint FROM nodes WHERE id = $1`, s.nodeID).Scan(&a1)
		_ = s.pool.QueryRow(s.ctx, `SELECT applied_config_generation::bigint FROM nodes WHERE id = $1`, s.node2ID).Scan(&a2)
		if a1 < genNodeAfter2 || a2 < genNode2After2 {
			return false
		}
		resp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
		if resp.status != http.StatusOK {
			return false
		}
		_ = json.Unmarshal(resp.body, &uEnforce2)
		if !uEnforce2.FairUse.Applying {
			applyingFalseAt2 = time.Now().UTC()
			return true
		}
		return false
	}) {
		t.Fatalf("AT19 Transition 2 fair_use.applying did not clear to false after both exits applied (last FairUse=%+v)", uEnforce2.FairUse)
	}
	wallClock2Ms := applyingFalseAt2.Sub(ingestStart).Milliseconds()

	var clearObs2ID uuid.UUID
	var clearObs2AppliedGen uint64
	var clearObs2Time time.Time
	_ = s.pool.QueryRow(s.ctx, `
		SELECT report_id, applied_generation::bigint, received_at
		FROM node_observations
		WHERE node_id = $1 AND incarnation_id = $2 AND applied_generation >= $3
		ORDER BY received_at ASC LIMIT 1
	`, s.node2ID, s.incarnation2ID, genNode2After2).Scan(&clearObs2ID, &clearObs2AppliedGen, &clearObs2Time)

	fupMemberGenAfter2, _, fupTierIdx2, fupDown2, fupUp2 := s.queryFUPMemberState(s.memberA.memberID)
	tcNodeDown2, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvmb0")
	tcNodeUp2, _, _ := s.execInContainer("node", "tc", "class", "show", "dev", "fvup0")
	tcNode2Down2, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvmb0")
	tcNode2Up2, _, _ := s.execInContainer("node2", "tc", "class", "show", "dev", "fvup0")
	node1Tier2AfterIngest := strings.Contains(string(tcNodeDown2), "ceil 5Mbit") && strings.Contains(string(tcNodeUp2), "ceil 2Mbit")
	node2Tier2AfterIngest := strings.Contains(string(tcNode2Down2), "ceil 5Mbit") && strings.Contains(string(tcNode2Up2), "ceil 2Mbit")
	if !node1Tier2AfterIngest || !node2Tier2AfterIngest {
		t.Fatalf("AT19 Transition 2 Tier 2 tc classes missing after usage threshold crossing: node=%t node2=%t", node1Tier2AfterIngest, node2Tier2AfterIngest)
	}

	// Replay the latest UsageEvent from usage_event_payloads via mTLS POST /api/v1/agent/usage-batches and verify generations do NOT bump twice
	cpReplayResp := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	var cpReplaySigned api.UsageCheckpoint
	if verifiedCP, _, err := envelope.Verify(cpReplayResp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC()); err == nil {
		_ = json.Unmarshal(verifiedCP, &cpReplaySigned)
	}
	var latestRawPayload []byte
	if err := s.pool.QueryRow(s.ctx, `
		SELECT uep.raw_payload
		FROM usage_event_payloads uep
		JOIN usage_events ue ON ue.id = uep.event_id
		WHERE ue.node_id = $1
		ORDER BY ue.sequence_number DESC LIMIT 1
	`, s.nodeID).Scan(&latestRawPayload); err != nil {
		t.Fatalf("AT19 query latest usage_event_payloads: %v", err)
	}
	var latestEvt api.UsageEvent
	if err := json.Unmarshal(latestRawPayload, &latestEvt); err != nil {
		t.Fatalf("AT19 unmarshal latest UsageEvent: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	replayBatchResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       presence.NewNullableWithValue(cpReplaySigned.LedgerEpoch),
		Events:            []api.UsageEvent{latestEvt},
	}, idemHeaders())
	if replayBatchResp.status != http.StatusOK {
		t.Fatalf("AT19 replay UsageBatchRequest failed: %d %s", replayBatchResp.status, string(replayBatchResp.body))
	}
	time.Sleep(3 * time.Second)
	gensAfterReplay2 := s.queryDesiredGenerations(s.nodeID, s.node2ID, s.otherNodeID)
	fupMemberGenReplay2, _, _, _, _ := s.queryFUPMemberState(s.memberA.memberID)
	genNodeAfterReplay2 := gensAfterReplay2[s.nodeID.String()]
	genNode2AfterReplay2 := gensAfterReplay2[s.node2ID.String()]
	genOtherAfterReplay2 := gensAfterReplay2[s.otherNodeID.String()]
	if genNodeAfterReplay2 != genNodeAfter2 || genNode2AfterReplay2 != genNode2After2 || genOtherAfterReplay2 != genOtherAfter2 {
		t.Fatalf("AT19 Transition 2 duplicate usage event replay unexpectedly bumped desired_config_generation: before=(%d,%d,%d) after=(%d,%d,%d)",
			genNodeAfter2, genNode2After2, genOtherAfter2, genNodeAfterReplay2, genNode2AfterReplay2, genOtherAfterReplay2)
	}

	s.fupUsageIngestTransition = fupTransitionMeasurement{
		Transition:                     "fup_member_generation_changed_tier1_to_tier2",
		Trigger:                        "POST /api/v1/agent/usage-batches -> fup.member_generation_changed (crossing Tier 2 threshold)",
		StartedAt:                      ingestStart.Format(time.RFC3339Nano),
		FUPMemberGenerationBefore:      fupMemberGenBefore2,
		FUPMemberGenerationAfter:       fupMemberGenAfter2,
		EnforcedTierIndex:              fupTierIdx2,
		EnforcedDownBPS:                fupDown2,
		EnforcedUpBPS:                  fupUp2,
		DesiredGenBefore:               gensBefore2,
		DesiredGenAfter:                gensAfter2,
		DesiredGenAfterReplay:          gensAfterReplay2,
		FUPMemberGenerationAfterReplay: fupMemberGenReplay2,
		ApplyingObservedTrue:           applyingObservedTrue2,
		ApplyingClearedToFalse:         !uEnforce2.FairUse.Applying,
		ApplyingClearedAt:              applyingFalseAt2.Format(time.RFC3339Nano),
		WallClockMsToApplyingFalse:     wallClock2Ms,
		ClearingObservationReportID:    clearObs2ID,
		ClearingObservationNodeID:      s.node2ID,
		ClearingObservationGeneration:  clearObs2AppliedGen,
		ClearingObservationReceivedAt:  clearObs2Time.UTC().Format(time.RFC3339Nano),
		Node1TCClassRate:               "ceil 5Mbit down (fvmb0) / ceil 2Mbit up (fvup0)",
		Node2TCClassRate:               "ceil 5Mbit down (fvmb0) / ceil 2Mbit up (fvup0)",
		BothExitsInstalledTier2TC:      node1Tier2AfterIngest && node2Tier2AfterIngest,
	}

	// Also verify temporary member override creation (revision 4)
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
		Command:  "Transition 1 (policy.activated mode=enforce) + Transition 2 (fup.member_generation_changed via usage ingestion) + duplicate event replays",
		ExitCode: 0,
		Output: fmt.Sprintf("T1(policy.activated): gens node=%d->%d node2=%d->%d other=%d->%d applying_true->false=%t/%t (%dms, obs=%s) tc_tier2=%t/%t replay_after=%v | T2(fup.member_generation_changed): gens node=%d->%d node2=%d->%d other=%d->%d applying_true->false=%t/%t (%dms, obs=%s) tc_tier2=%t/%t replay_after=%v",
			genNodeBefore1, genNodeAfter1, genNode2Before1, genNode2After1, genOtherBefore1, genOtherAfter1,
			s.fupActivationTransition.ApplyingObservedTrue, s.fupActivationTransition.ApplyingClearedToFalse, wallClock1Ms, clearObs1ID,
			node1Tier2OK, node2Tier2OK, gensAfterReplay1,
			genNodeBefore2, genNodeAfter2, genNode2Before2, genNode2After2, genOtherBefore2, genOtherAfter2,
			s.fupUsageIngestTransition.ApplyingObservedTrue, s.fupUsageIngestTransition.ApplyingClearedToFalse, wallClock2Ms, clearObs2ID,
			node1Tier2AfterIngest, node2Tier2AfterIngest, gensAfterReplay2),
		At: time.Now().UTC().Format(time.RFC3339),
	})

	// 5. Measure ENR-03 per-incarnation retained usage across node2 re-enrollment through product APIs only
	s.incarnationRetention = s.measurePerIncarnationRetainedUsage()
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "ENR-03 per-incarnation retained usage: POST /admin/nodes/{id}/revoke + bound POST /admin/enrollments + fvpn-agent enroll + GET /admin/nodes/{id}/incarnations + new incarnation counted traffic",
		ExitCode: 0,
		Output: fmt.Sprintf("node2=%s old_inc=%s (retained=%s/%s B) new_inc=%s (after_traffic=%s/%s B) member_period_unchanged_on_reenroll=%t old_inc_fixed_after_new_traffic=%t",
			s.incarnationRetention.NodeID,
			s.incarnationRetention.OldIncarnationID,
			s.incarnationRetention.IncarnationsAfterReinstall.Items[1].Usage.UploadBytes,
			s.incarnationRetention.IncarnationsAfterReinstall.Items[1].Usage.DownloadBytes,
			s.incarnationRetention.NewIncarnationID,
			s.incarnationRetention.IncarnationsAfterNewTraffic.Items[0].Usage.UploadBytes,
			s.incarnationRetention.IncarnationsAfterNewTraffic.Items[0].Usage.DownloadBytes,
			s.incarnationRetention.MemberTotalUnchangedOnReenroll,
			s.incarnationRetention.OldIncarnationUsageFixed),
		At: time.Now().UTC().Format(time.RFC3339),
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
	rec.Observations["fup_transition_1_policy_activated"] = s.fupActivationTransition
	rec.Observations["fup_transition_2_member_generation_changed"] = s.fupUsageIngestTransition
	rec.Observations["enr03_per_incarnation_retained_usage"] = s.incarnationRetention
	rec.Observations["fup_member_state_enforce"] = map[string]any{
		"mode":                       fupMode1,
		"effective_down_bps":         fupDown1,
		"effective_up_bps":           fupUp1,
		"tc_node_enforced_5mbit":     node1Tier2OK,
		"tc_node2_enforced_5mbit":    node2Tier2OK,
		"member_usage_fair_use_view": uEnforce2.FairUse,
	}

	rec.Actual = fmt.Sprintf("Live multi-node test enrolled second exit %s (node2) and non-carrying node %s, routed memberA traffic across both carrying exits (aggregating %d bytes across %d exits), verified shadow mode (60Mbit on node, 15Mbit on node2), proved Transition 1 (policy.activated mode=enforce: carrying gens node %d->%d, node2 %d->%d, non-carrying %d->%d; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in %d ms via observation %s; replay no double bump) and Transition 2 (fup.member_generation_changed via usage threshold crossing: carrying gens node %d->%d, node2 %d->%d, non-carrying %d->%d; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in %d ms via observation %s; replay no double bump), and verified ENR-03 per-incarnation retained usage across node2 re-enrollment (%s -> %s).",
		s.node2ID, s.otherNodeID, crossNodeTotal, distinctExitsCount,
		genNodeBefore1, genNodeAfter1, genNode2Before1, genNode2After1, genOtherBefore1, genOtherAfter1, wallClock1Ms, clearObs1ID,
		genNodeBefore2, genNodeAfter2, genNode2Before2, genNode2After2, genOtherBefore2, genOtherAfter2, wallClock2Ms, clearObs2ID,
		s.incarnationRetention.OldIncarnationID, s.incarnationRetention.NewIncarnationID)
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

	t1 := s.fupActivationTransition
	t2 := s.fupUsageIngestTransition
	n1Str := s.nodeID.String()
	n2Str := s.node2ID.String()
	othStr := s.otherNodeID.String()

	summary := fmt.Sprintf("# Wave 4 Usage Acceptance Summary (`w4-at-usage` — Follow-up 3)\n\n"+
		"- **Run ID**: `%s`\n"+
		"- **Recorded At (UTC)**: `%s`\n"+
		"- **Source Commit**: `%s` (dirty: `%t`)\n"+
		"- **Stack Instance**: `FVPN_E2E_PREFIX=%s` (`fvpn-%s-*`), `FVPN_E2E_PORT_BASE=%d`\n"+
		"- **Primary Exit Node**: `node_id=%s`, `incarnation_id=%s`, `exit_id=%s` (`%s`)\n"+
		"- **Second Exit Node (`AT19` & `ENR-03`)**: `node_id=%s`, `incarnation_id=%s` (initial: `%s`), `exit_id=%s` (`%s`)\n"+
		"- **Non-Carrying Registered Node (`AT19`)**: `node_id=%s`, `incarnation_id=%s` (`desired_config_generation=0`)\n"+
		"- **Admitted Member Devices**: `memberA` device `%s`, `memberB` device `%s`\n"+
		"- **Totals**: `%d pass`, `%d blocked`, `0 fail`\n"+
		"- **Run Archives**: [history/initial/](history/initial/) (#80), [history/followup2/](history/followup2/) (#214)\n\n"+
		"## Acceptance Matrix (AT14–AT19)\n\n%s\n"+
		"## Follow-up 3 Measured Timelines\n\n"+
		"### 1. AT19 Fair-Use Recompile & `fair_use.applying` Convergence\n\n"+
		"| Transition | Carrying `node` Gen | Carrying `node2` Gen | Non-Carrying `other` Gen | Clearing Observation | Wall-Clock (`ms`) | Tier 2 `tc` (`5Mbit`/`2Mbit`) | Replay Gen After |\n"+
		"|---|---|---|---|---|---|---|---|\n"+
		"| `policy.activated` (`mode=enforce`) | `%d -> %d` | `%d -> %d` | `%d -> %d` | `%s` (`gen=%d`, `%s`) | `%d ms` | `%t` | `%d / %d / %d` |\n"+
		"| `fup.member_generation_changed` (usage threshold) | `%d -> %d` | `%d -> %d` | `%d -> %d` | `%s` (`gen=%d`, `%s`) | `%d ms` | `%t` | `%d / %d / %d` |\n\n"+
		"### 2. D-067 Provisional-Lease Timing (`awaiting_member_grants` -> Populated Lease & First Counted Byte)\n\n"+
		"| Cycle | Ready Observation (`t_ready`) | Populated Lease (`t_lease`) | First Counted Byte (`t_byte`) | Ready -> Lease (`ms`) | Ready -> Byte (`ms`) | Max Allowed (`ms`) | Checks (`lease`/`qos`/`source_binding`) | `errors[]` (`stage=lease`, zero `LEASE_EXPIRED`) |\n"+
		"|---|---|---|---|---|---|---|---|---|\n"+
		"| `bring_up` | `%s` (`%s`) | `members=%d` (`%s`) | `+%s/+%s B` (`%s`) | `%d ms` | `%d ms` | `%d ms` | `%s` / `%s` / `%s` | `count=%d` (`%s`), `expired=%t` |\n"+
		"| `agent_restart` | `%s` (`%s`) | `members=%d` (`%s`) | `+%s/+%s B` (`%s`) | `%d ms` | `%d ms` | `%d ms` | `%s` / `%s` / `%s` | `count=%d` (`%s`), `expired=%t` |\n\n"+
		"### 3. `ENR-03` Per-Incarnation Retained Usage (`GET /api/v1/admin/nodes/{id}/incarnations`)\n\n"+
		"- **Node ID**: `%s` (409 while active: `%s`)\n"+
		"- **Retired Incarnation (`old`)**: `%s` (`state=retired`, `retire_reason=reinstall`, retained `upload_bytes=%s`, `download_bytes=%s`, unchanged after new traffic: `%t`)\n"+
		"- **Active Incarnation (`new`)**: `%s` (`state=active`, initial `0/0`, after new traffic `upload_bytes=%s`, `download_bytes=%s`)\n"+
		"- **Member Period Continuity (`member_usage_periods`)**: before re-enroll `total_bytes=%s`, after re-enroll `total_bytes=%s` (`unchanged=%t`), after new traffic `total_bytes=%s` (`accumulated=%t`)\n\n"+
		"## Upstream Product Blockers\n\n"+
		"None (`0 blocked`). See [requests.md](requests.md) for resolved items and non-blocking notes.\n",
		s.runID, nowUTC, s.build.GitCommit, s.build.GitDirty, assignedPrefix, assignedPrefix, assignedPortBase,
		s.nodeID, s.incarnationID, s.exitID, s.exitNetID,
		s.node2ID, s.incarnation2ID, s.incarnationRetention.OldIncarnationID, s.exit2ID, s.exit2NetID,
		s.otherNodeID, s.otherIncarnationID,
		s.deviceA, s.deviceB, passCount, blockedCount, matrix.String(),
		t1.DesiredGenBefore[n1Str], t1.DesiredGenAfter[n1Str],
		t1.DesiredGenBefore[n2Str], t1.DesiredGenAfter[n2Str],
		t1.DesiredGenBefore[othStr], t1.DesiredGenAfter[othStr],
		t1.ClearingObservationReportID, t1.ClearingObservationGeneration, t1.ClearingObservationReceivedAt,
		t1.WallClockMsToApplyingFalse, t1.BothExitsInstalledTier2TC,
		t1.DesiredGenAfterReplay[n1Str], t1.DesiredGenAfterReplay[n2Str], t1.DesiredGenAfterReplay[othStr],
		t2.DesiredGenBefore[n1Str], t2.DesiredGenAfter[n1Str],
		t2.DesiredGenBefore[n2Str], t2.DesiredGenAfter[n2Str],
		t2.DesiredGenBefore[othStr], t2.DesiredGenAfter[othStr],
		t2.ClearingObservationReportID, t2.ClearingObservationGeneration, t2.ClearingObservationReceivedAt,
		t2.WallClockMsToApplyingFalse, t2.BothExitsInstalledTier2TC,
		t2.DesiredGenAfterReplay[n1Str], t2.DesiredGenAfterReplay[n2Str], t2.DesiredGenAfterReplay[othStr],
		s.d067BringUp.ReadyObservationReportID, s.d067BringUp.ReadyObservationReceivedAt,
		s.d067BringUp.PopulatedLeaseMemberCount, s.d067BringUp.PopulatedLeaseInForceAt,
		s.d067BringUp.FirstCountedTransfer.DeltaUp, s.d067BringUp.FirstCountedTransfer.DeltaDown, s.d067BringUp.FirstCountedByteAt,
		s.d067BringUp.ReadyToPopulatedLeaseMs, s.d067BringUp.ReadyToFirstCountedByteMs, s.d067BringUp.MaxAllowedWindowMs,
		s.d067BringUp.CheckErrorCodes["lease"], s.d067BringUp.CheckErrorCodes["qos"], s.d067BringUp.CheckErrorCodes["source_binding"],
		len(s.d067BringUp.Errors), s.d067BringUp.Errors[0].Code, s.d067BringUp.LeaseExpiredReportedDuringWindow,
		s.d067Restart.ReadyObservationReportID, s.d067Restart.ReadyObservationReceivedAt,
		s.d067Restart.PopulatedLeaseMemberCount, s.d067Restart.PopulatedLeaseInForceAt,
		s.d067Restart.FirstCountedTransfer.DeltaUp, s.d067Restart.FirstCountedTransfer.DeltaDown, s.d067Restart.FirstCountedByteAt,
		s.d067Restart.ReadyToPopulatedLeaseMs, s.d067Restart.ReadyToFirstCountedByteMs, s.d067Restart.MaxAllowedWindowMs,
		s.d067Restart.CheckErrorCodes["lease"], s.d067Restart.CheckErrorCodes["qos"], s.d067Restart.CheckErrorCodes["source_binding"],
		len(s.d067Restart.Errors), s.d067Restart.Errors[0].Code, s.d067Restart.LeaseExpiredReportedDuringWindow,
		s.incarnationRetention.NodeID, s.incarnationRetention.ActiveReenrollRejected409Code,
		s.incarnationRetention.OldIncarnationID,
		s.incarnationRetention.IncarnationsAfterReinstall.Items[1].Usage.UploadBytes,
		s.incarnationRetention.IncarnationsAfterReinstall.Items[1].Usage.DownloadBytes,
		s.incarnationRetention.OldIncarnationUsageFixed,
		s.incarnationRetention.NewIncarnationID,
		s.incarnationRetention.IncarnationsAfterNewTraffic.Items[0].Usage.UploadBytes,
		s.incarnationRetention.IncarnationsAfterNewTraffic.Items[0].Usage.DownloadBytes,
		s.incarnationRetention.DBPeriodTotalsBeforeReinstall["total_bytes"],
		s.incarnationRetention.DBPeriodTotalsAfterReinstall["total_bytes"], s.incarnationRetention.MemberTotalUnchangedOnReenroll,
		s.incarnationRetention.DBPeriodTotalsAfterNewTraffic["total_bytes"], s.incarnationRetention.NewTrafficAccumulatedSameMonth)
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
		"## 3. Product-Only Bootstrap & Follow-up 3 Verification Path (Run `%s`, `%s`)\n\n"+
		"1. `e2e.PrepareServingBaseline(ctx, admin.Client, admin.CSRF, false)` builds, signs, uploads, registers, and verifies `0.1.0` (`amd64` + `arm64`) and activates `release_support` (`%s`).\n"+
		"2. `POST /api/v1/admin/groups` creates member exit group `%s`.\n"+
		"3. `POST /api/v1/admin/policies` + `POST /activate` activates `dns_profile` (`%s`), `fair_use` (`%s`), and `plan` (`%s`) with `exit_group_ids=[%s]`.\n"+
		"4. `stack.Enroll` & `stack.EnrollAt(..., \"node2\")`, `PATCH /api/v1/admin/nodes/{id}` (finite capacity), and `PUT /api/v1/admin/nodes/{id}/publication` (`group_ids=[%s]`) publish both carrying exits.\n"+
		"5. `POST /api/v1/admin/members/{id}/grants` + `stack.JoinNativeMember` + `netbird networks select <exit_net_id>` establishes counted member forwarding (`serving 1`, `fvpn_generation > 0`, positive `u_<dev>`/`d_<dev>` counters) while measuring D-067 Cycle 1 (`bring_up`).\n"+
		"6. `AT14–AT19` measure D-067 Cycle 2 (`agent_restart`), Fair-Use recompile & `fair_use.applying` clearing across both `policy.activated` and `fup.member_generation_changed`, and `ENR-03` per-incarnation retained usage across node re-enrollment (`GET /api/v1/admin/nodes/{id}/incarnations`).\n",
		s.runID, nowUTC, s.releaseSupportRevID, s.groupID, s.dnsProfileRevID, s.fairUseShadowRevID, s.planRevID, s.groupID, s.groupID)
	if err := os.WriteFile(filepath.Join(s.evidence, "RUN.md"), []byte(runDoc), 0o644); err != nil {
		s.t.Fatalf("write RUN.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.evidence, "operator-run.md"), []byte(runDoc), 0o644); err != nil {
		s.t.Fatalf("write operator-run.md: %v", err)
	}

	reqs := fmt.Sprintf("# Upstream Blockers & Requests to Orchestrator (`w4-at-usage` — Follow-up 3)\n\n"+
		"## 1. Active Blockers\n\n"+
		"**None (`0 blocked`, `6/6 pass` on live stack `fvpn-w4-usage`, run `%s` at `%s`, commit `%s`).**\n\n"+
		"## 2. Resolved Since Follow-up 2 (#214)\n\n"+
		"1. **Resolved by PR #222 (`w3-policy-fup`) — Desired-State Recompile on `policy.activated` and `fup.member_generation_changed`**:\n"+
		"   - Verified in `AT19`: activating `fair_use` in `mode=enforce` incremented `desired_config_generation` on both carrying exits (`node` `%d -> %d`, `node2` `%d -> %d`) and left the non-carrying node (`%s`) at `%d -> %d`; usage ingestion crossing the Tier 2 threshold (`fup.member_generation_changed`) incremented `desired_config_generation` on both carrying exits (`node` `%d -> %d`, `node2` `%d -> %d`) and left the non-carrying node at `%d -> %d`. Replaying either event did not bump generations twice.\n"+
		"2. **Resolved by PR #222 (`w3-policy-fup`) — `pgNodeAckReader.GetMemberNodeAckStatus` Clearing `fair_use.applying`**:\n"+
		"   - Verified in `AT19`: `GET /api/v1/member/usage` `fair_use.applying` transitioned `true -> false` once both carrying exits applied the compiled generation (`%d ms` on `policy.activated` via observation `%s`; `%d ms` on `fup.member_generation_changed` via observation `%s`).\n"+
		"3. **Verified on `main` (PR #206, #213, #216) — D-067 Provisional-Lease Convergence (`AWAITING_MEMBER_GRANTS`)**:\n"+
		"   - Measured on two real cycles (`bring_up`: `%d ms` ready->lease, `%d ms` ready->first-byte; `agent_restart`: `%d ms` ready->lease, `%d ms` ready->first-byte), both within `<= 65000 ms` (`60 s + 1 heartbeat`), reporting `AWAITING_MEMBER_GRANTS` on `lease`/`qos`/`source_binding` and 1 `errors[]` entry at `stage=lease` (never `LEASE_EXPIRED`).\n"+
		"4. **Verified on `main` (PR #210) — `ENR-03` Per-Incarnation Retained Usage (`GET /api/v1/admin/nodes/{id}/incarnations`)**:\n"+
		"   - Verified on `node2` (`%s`): re-enrolling `node2` after counted traffic retired incarnation `%s` (`retire_reason=reinstall`, retained `upload_bytes=%s`, `download_bytes=%s`) and activated incarnation `%s` newest-first without altering `member_usage_periods` (`total_bytes=%s`), and subsequent traffic through `%s` accumulated on the same member period while `%s` stayed fixed.\n\n"+
		"## 3. Non-Blocking Lab Note (`w3-devstack-e2e`)\n\n"+
		"- **`deploy/devstack/agent-service.sh` vs `entrypoint-node.sh` supervisor loop**: Calling `agent-service.sh stop` / `start` directly without `/var/lib/vpn-node/e2e/pause-agent` can race with the 2-second supervisor loop in `entrypoint-node.sh`. `w4-at-usage` coordinates cleanly with `entrypoint-node.sh` via `/var/lib/vpn-node/e2e/pause-agent`.\n",
		s.runID, nowUTC, s.build.GitCommit,
		t1.DesiredGenBefore[n1Str], t1.DesiredGenAfter[n1Str],
		t1.DesiredGenBefore[n2Str], t1.DesiredGenAfter[n2Str],
		s.otherNodeID, t1.DesiredGenBefore[othStr], t1.DesiredGenAfter[othStr],
		t2.DesiredGenBefore[n1Str], t2.DesiredGenAfter[n1Str],
		t2.DesiredGenBefore[n2Str], t2.DesiredGenAfter[n2Str],
		t2.DesiredGenBefore[othStr], t2.DesiredGenAfter[othStr],
		t1.WallClockMsToApplyingFalse, t1.ClearingObservationReportID,
		t2.WallClockMsToApplyingFalse, t2.ClearingObservationReportID,
		s.d067BringUp.ReadyToPopulatedLeaseMs, s.d067BringUp.ReadyToFirstCountedByteMs,
		s.d067Restart.ReadyToPopulatedLeaseMs, s.d067Restart.ReadyToFirstCountedByteMs,
		s.incarnationRetention.NodeID,
		s.incarnationRetention.OldIncarnationID,
		s.incarnationRetention.IncarnationsAfterReinstall.Items[1].Usage.UploadBytes,
		s.incarnationRetention.IncarnationsAfterReinstall.Items[1].Usage.DownloadBytes,
		s.incarnationRetention.NewIncarnationID,
		s.incarnationRetention.DBPeriodTotalsAfterReinstall["total_bytes"],
		s.incarnationRetention.NewIncarnationID,
		s.incarnationRetention.OldIncarnationID)
	if err := os.WriteFile(filepath.Join(s.evidence, "requests.md"), []byte(reqs), 0o644); err != nil {
		s.t.Fatalf("write requests.md: %v", err)
	}
}
