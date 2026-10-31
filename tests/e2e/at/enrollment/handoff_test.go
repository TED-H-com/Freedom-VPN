//go:build e2e

package enrollment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

const installerContainer = "fvpn-w4-enroll-installer-node"
const devstack18 = "docs/e2e/runs/2026-10-31/followup18/README.md"
const usageAT20 = "docs/e2e/at/usage/AT20.md"

// Follow the usage lane's AT20 path and named assertions. The only transitions
// here are public API retirement and the shared installer/service wrappers.
// SQL below extracts evidence from this lane's disposable DB or stopped copy.
func (p *probe) installerHooks(t *testing.T) {
	detail := map[string]any{"slot": "installer-node", "installer_fix_pr": 302, "agent_activation_pr": 312,
		"devstack_evidence": devstack18, "usage_assertions": usageAT20, "owner": "enrollment"}
	stage, owner := "prepare", "devstack"
	must := func(ok bool) {
		t.Helper()
		if !ok {
			detail["step"], detail["owner"] = stage, owner
			p.expect("AT07", "installer_prerequisite", false, detail)
			t.Fatal("installer campaign prerequisite failed; see typed evidence")
		}
	}
	step := func(s, o string) { stage, owner = s, o; t.Log("installer: " + s) }
	must(p.servingGroup != uuid.Nil && p.servingDevice != uuid.Nil)
	must(p.stack.Service(p.ctx, "stop", "") == nil)
	must(p.stack.PrepareInstallerNode(p.ctx) == nil)
	detail["image_digest"] = dockerText(p.ctx, "inspect", "--format={{.Image}}", installerContainer)
	releases, err := p.stack.RegisterInstallerReleases(p.ctx, p.admin)
	must(err == nil)
	ids := []uuid.UUID{p.bootRelease, releases[0].ID, releases[1].ID}
	_, _, err = p.stack.ActivatePolicy(p.ctx, p.admin, "release_support", "enrollment-installer", map[string]any{"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	must(err == nil)
	must(p.stack.HoldInstallerFirstPoll(p.ctx, true) == nil)
	token, ok := p.issue("installer-initial", 900)
	must(ok)
	must(p.stack.StageInstallerToken(p.ctx, "installer-node", token.token) == nil)
	step("initial install", "installer")
	installed, err := p.stack.RerunInstaller(p.ctx, "installer-node")
	detail["initial_install"] = installerOutcomeView(installed, err)
	detail["after_initial_install"] = p.installerDiagnostics("initial", installed)
	must(err == nil && installed.Exit == 0 && installed.Status == "installed")
	node, err := uuid.Parse(installed.NodeID)
	must(err == nil)
	origin, err := uuid.Parse(installed.Incarnation)
	must(err == nil)
	p.oneIncarnation("AT07", node, origin, token.enrollment.EnrollmentId)
	must(p.stack.ConfigureInstallerServing(p.ctx) == nil)
	must(p.stack.HoldInstallerFirstPoll(p.ctx, false) == nil)
	p.restoreInstaller("initial", installed, detail)
	step("publish real installer and select native exit", "agent")
	must(p.installerPublish(node))
	selected, err := p.stack.MemberTransportStack("fvpn-w4-enroll-member", "installer-node")
	must(err == nil)
	must(p.selectInstallerExit(node))
	initialRules, ok := p.installerServingReadback(node, time.Now().Add(-15*time.Second))
	detail["initial_serving"] = initialRules
	must(ok)
	step("count baseline traffic and upload", "agent")
	baseline, ok := p.installerTraffic(selected)
	detail["baseline_traffic"] = baseline
	must(ok)
	must(p.waitInstaller(90*time.Second, func() bool {
		snap, e := p.stack.UsageLedgerSnapshot(p.ctx, "installer-node")
		detail["baseline_upload_snapshot"], detail["baseline_upload_snapshot_read_ok"] = snap, e == nil
		return e == nil && snap.LastEventSeq != "0" && snap.AckedThrough == snap.LastEventSeq && snap.UnackedCount == 0
	}))
	step("hold counted usage", "agent")
	must(p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", true) == nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(p.ctx), 20*time.Second)
		defer cancel()
		_ = selected.StopEstablishedTransports(ctx)
		_ = p.stack.HoldUsageAcknowledgements(ctx, "installer-node", false)
	}()
	heldTraffic, ok := p.installerTraffic(selected)
	detail["held_traffic"] = heldTraffic
	must(ok)
	var held e2e.UsageSnapshot
	must(p.waitInstaller(45*time.Second, func() bool {
		held, err = p.stack.UsageLedgerSnapshot(p.ctx, "installer-node")
		u, d := snapshotBytes(held, p.servingMember.MemberID)
		return err == nil && u != "0" && d != "0"
	}))
	p.expect("AT07", "counted_unacked_usage", true, map[string]any{"owner": "agent", "traffic": heldTraffic, "held": held, "serving": initialRules, "devstack_rows": devstack18 + " installer-serving-readback / installer-usage"})
	step("same-release rerun with held ledger", "installer")
	again, err := p.stack.RerunInstaller(p.ctx, "installer-node")
	detail["rerun"] = installerOutcomeView(again, err)
	detail["after_rerun"] = p.installerDiagnostics("rerun", again)
	stable := err == nil && again.Exit == 0 && again.Status == "already_current" && again.Before == again.Hashes
	p.expect("AT07", "installer_stopped_hash_equality", stable, installerOutcomeView(again, err))
	p.expect("AT07", "installer_rerun_identity", stable && again.NodeID == installed.NodeID && again.Incarnation == installed.Incarnation, installerOutcomeView(again, err))
	must(stable)
	// Endpoint-scope helper releases the hold while all consumers are stopped.
	p.installerUsageHold()
	must(p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", true) == nil)
	restoredAt := time.Now().UTC()
	p.restoreInstaller("rerun", again, detail)
	rerunRules, ok := p.installerServingReadback(node, restoredAt)
	detail["rerun_serving"] = rerunRules
	must(ok)
	rerunTraffic, ok := p.installerTraffic(selected)
	detail["rerun_traffic"] = rerunTraffic
	must(ok)
	p.expect("AT07", "installer_rerun", true, map[string]any{"owner": "installer", "outcome": installerOutcomeView(again, err), "serving": rerunRules, "traffic": rerunTraffic, "devstack_rows": devstack18 + " installer-marker-migration / installer-wrappers"})
	// Pace deliberate systemd starts beyond its unchanged 120-second window.
	// No reset-failed or alteration of packaged start limits.
	step("pace restart campaign before quiescence", "enrollment")
	for i := 0; i < 121; i++ {
		must(waitRetry(p.ctx, "1"))
	}
	states, err := p.stack.InstallerServices(p.ctx, "installer-node")
	must(err == nil)
	must(p.stopInstaller())
	first, err := p.stack.UsageLedgerSnapshotBytes(p.ctx, "installer-node")
	must(err == nil)
	second, err := p.stack.UsageLedgerSnapshotBytes(p.ctx, "installer-node")
	must(err == nil && bytes.Equal(first, second))
	must(json.Unmarshal(first, &held) == nil)
	preDB := p.installerLedgerCopy(t)
	events := readHeldEvents(t, preDB)
	must(len(events) > 0)
	up, down := snapshotBytes(held, p.servingMember.MemberID)
	must(up != "0" && down != "0")
	period := time.Now().UTC().Format("2006-01")
	before, ok := p.memberUsage(period)
	must(ok)
	beforeInc, resp, err := p.listIncarnations(node, "")
	must(err == nil && resp.Status == 200 && len(beforeInc.Items) == 1)
	detail["origin_cli_before_reenrollment"], detail["member_period_before"], detail["incarnations_before"] = held, before, installerHistoryView(beforeInc)
	detail["held_upload_bytes"], detail["held_download_bytes"], detail["held_total_bytes"] = up, down, decimalSum(up, down)
	detail["quiescent_cli_byte_equal"] = true
	for _, ev := range events {
		var n int
		must(p.freezeSQL(fmt.Sprintf("SELECT count(*) FROM usage_events WHERE node_id='%s' AND incarnation_id='%s' AND event_seq=%s", node, origin, ev.Event.EventSeq), &n) && n == 0)
	}
	step("authorized API retirement and worker read-back", "control/worker")
	rev, err := p.revokeNodeReason(node, "TEST ONLY AT07 controlled installer reinstall")
	detail["revoke_response"] = safeResponse(rev, err)
	must(err == nil && rev.Status == 202)
	var accepted api.OperationAccepted
	must(json.Unmarshal(rev.Body, &accepted) == nil)
	var retiredAt time.Time
	must(p.waitInstaller(45*time.Second, func() bool {
		page, r, e := p.listIncarnations(node, "")
		if e != nil || r.Status != 200 {
			return false
		}
		for _, v := range page.Items {
			if v.IncarnationId == origin && v.State == api.NodeIncarnationStateRetired {
				retiredAt, _ = v.RetiredAt.Get()
				return !retiredAt.IsZero()
			}
		}
		return false
	}))
	must(p.waitInstaller(45*time.Second, func() bool {
		r, e := p.call("GET", "/api/v1/admin/operations/"+accepted.OperationId.String(), nil)
		var op api.Operation
		if e != nil || r.Status != 200 || json.Unmarshal(r.Body, &op) != nil {
			return false
		}
		detail["revoke_operation_readback"] = op
		route, identity := false, false
		for _, s := range op.Steps {
			if s.Name == "netbird_route_readback" {
				route = s.Status == api.OperationStepStatusSucceeded
			}
			if s.Name == "netbird_identity_cleanup" {
				identity = s.Status == api.OperationStepStatusSucceeded
			}
		}
		return route && identity
	}))
	detail["revocation_scope"] = "External route and identity steps read back; stopped old agent cannot ACK withdrawal, so full-operation completion is not claimed."
	step("controlled same-node reinstall", "installer")
	firstCheck := len(p.checks)
	fresh, ok := p.issueOn("installer-reinstall", 900, &node)
	for i := firstCheck; i < len(p.checks); i++ {
		p.checks[i].AT = "AT07"
	}
	must(ok)
	must(p.stack.HoldInstallerFirstPoll(p.ctx, true) == nil)
	reinstalled, err := p.stack.ControlledReinstall(p.ctx, "installer-node", fresh.token)
	detail["controlled_reinstall"] = installerOutcomeView(reinstalled, err)
	detail["after_controlled_reinstall"] = p.installerDiagnostics("reinstall", reinstalled)
	must(err == nil && reinstalled.Exit == 0 && reinstalled.Status == "reenrolled" && reinstalled.NodeID == node.String() && reinstalled.Incarnation != origin.String())
	current, err := uuid.Parse(reinstalled.Incarnation)
	must(err == nil)
	retained := reinstalled.RetiredMode == "700" && reinstalled.RetiredHashes.Key == reinstalled.Before.Key && reinstalled.RetiredHashes.Cert == reinstalled.Before.Cert && reinstalled.RetiredHashes.NodeJSON == reinstalled.Before.NodeJSON && len(reinstalled.BoundStateBefore) == 4 && sameStrings(reinstalled.BoundStateBefore, reinstalled.RetiredBoundState)
	page, r, err := p.listIncarnations(node, "")
	history := err == nil && r.Status == 200 && len(page.Items) == 2 && page.Items[0].IncarnationId == current && page.Items[0].Current && page.Items[0].CreatedAt.After(retiredAt) && page.Items[1].IncarnationId == origin && page.Items[1].State == "retired"
	p.expect("AT07", "installer_reinstall_identity", retained && history, map[string]any{"outcome": installerOutcomeView(reinstalled, err), "history": installerHistoryView(page), "retained_files_match": retained})
	must(retained && history)
	adopted := p.installerLedgerCopy(t)
	var localCurrent string
	must(adopted.QueryRow("SELECT incarnation_id FROM meta WHERE id=1").Scan(&localCurrent) == nil)
	table := "events"
	if localCurrent != origin.String() {
		must(localCurrent == current.String())
		table = "retired_events"
		var ack, last string
		must(adopted.QueryRow("SELECT CAST(acked_through AS TEXT),CAST(last_seq AS TEXT) FROM retired_streams WHERE incarnation_id=?", origin.String()).Scan(&ack, &last) == nil && ack == held.AckedThrough && last == held.LastEventSeq)
	}
	for _, ev := range events {
		var raw []byte
		var digest string
		q := "SELECT payload,payload_sha256 FROM " + table + " WHERE seq=?"
		args := []any{uint64(ev.Event.EventSeq)}
		if table == "retired_events" {
			q += " AND incarnation_id=?"
			args = append(args, origin.String())
		}
		must(adopted.QueryRow(q, args...).Scan(&raw, &digest) == nil && bytes.Equal(raw, ev.Raw) && digest == ev.Event.PayloadSha256)
	}
	detail["original_payload_bytes_unchanged"] = true
	detail["ledger_current_at_wrapper_return"] = localCurrent
	detail["reinstall_usage_files_byte_equal"] = reinstalled.Before.UsageDB == reinstalled.Hashes.UsageDB && reinstalled.Before.UsageWAL == reinstalled.Hashes.UsageWAL
	detail["hash_scope"] = "AT20 OriginalPayloadAndHistoricalAttribution: D-118 adoption can change DB/WAL bytes during bootstrap; original canonical events, hashes and ACK must stay unchanged."
	denials, denied := p.installerRetiredCertificate()
	p.expect("AT07", "old_certificate_denied", denied, denials)
	must(denied)
	step("autonomous origin upload with new presenter", "agent/usage")
	must(p.stack.ConfigureInstallerServing(p.ctx) == nil)
	must(p.stack.HoldInstallerFirstPoll(p.ctx, false) == nil)
	reinstalled.Services = states
	detail["handoff_restored_at"] = time.Now().UTC()
	p.restoreInstaller("reinstall", reinstalled, detail)
	must(p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", false) == nil)
	must(p.waitInstaller(150*time.Second, func() bool { return p.streamACK(node, origin) == held.LastEventSeq }))
	must(p.stopInstaller())
	postDB := p.installerLedgerCopy(t)
	currentCLI, err := p.stack.UsageLedgerSnapshot(p.ctx, "installer-node")
	must(err == nil && currentCLI.NodeIncarnationID == current.String())
	var oldACK, last, epoch, finalCode string
	must(postDB.QueryRow("SELECT CAST(acked_through AS TEXT),CAST(last_seq AS TEXT),ledger_epoch,final_code FROM retired_streams WHERE incarnation_id=?", origin.String()).Scan(&oldACK, &last, &epoch, &finalCode) == nil)
	independent := oldACK == held.LastEventSeq && last == held.LastEventSeq && finalCode == "" && held.LedgerEpoch != nil && epoch == *held.LedgerEpoch && p.streamACK(node, current) == "0" && currentCLI.AckedThrough == "0" && currentCLI.LastEventSeq == "0"
	detail["current_cli_after_handoff"], detail["retired_local_stream"] = currentCLI, map[string]string{"incarnation_id": origin.String(), "acked_through": oldACK, "last_seq": last, "ledger_epoch": epoch, "final_code": finalCode}
	detail["current_next_sequence"] = "1"
	p.expect("AT07", "OriginACKAndIndependentCurrentACK", independent, map[string]any{"assertion_source": usageAT20, "origin_ack": oldACK, "current_ack": currentCLI.AckedThrough, "current_last_seq": currentCLI.LastEventSeq, "current_next_seq": "1"})
	must(independent)
	originalOK := p.historicalEvents(events)
	after, ok := p.memberUsage(period)
	must(ok)
	afterInc, r, err := p.listIncarnations(node, "")
	must(err == nil && r.Status == 200 && len(afterInc.Items) == 2)
	originUsage, currentUsage := afterInc.Items[1].Usage, afterInc.Items[0].Usage
	attributed := afterInc.Items[1].IncarnationId == origin && afterInc.Items[1].State == "retired" && originUsage.UploadBytes.String() == decimalSum(beforeInc.Items[0].Usage.UploadBytes.String(), up) && originUsage.DownloadBytes.String() == decimalSum(beforeInc.Items[0].Usage.DownloadBytes.String(), down) && currentUsage.UploadBytes.String() == "0" && currentUsage.DownloadBytes.String() == "0"
	equality := after.Totals.UploadBytes.String() == decimalSum(before.Totals.UploadBytes.String(), up) && after.Totals.DownloadBytes.String() == decimalSum(before.Totals.DownloadBytes.String(), down) && after.Totals.TotalBytes.String() == decimalSum(before.Totals.TotalBytes.String(), decimalSum(up, down))
	detail["member_period_after"], detail["incarnations_after_handoff"], detail["handed_off_events"] = after, installerHistoryView(afterInc), events
	detail["before_plus_held_equals_after"], detail["historical_attribution_unchanged"] = equality, attributed && originalOK
	p.expect("AT07", "BeforePlusHeldByteEquality", equality, map[string]any{"assertion_source": usageAT20, "before": before, "held_upload": up, "held_download": down, "after": after})
	p.expect("AT07", "OriginalPayloadAndHistoricalAttribution", originalOK && attributed, map[string]any{"assertion_source": usageAT20, "events": events, "before": installerHistoryView(beforeInc), "after": installerHistoryView(afterInc)})
	must(equality && originalOK && attributed)
	// Saved production batches must present the new certificate identity while
	// selecting the original stream. The current stream has not issued seq 1 yet.
	rows, err := postDB.Query("SELECT body FROM upload_batches ORDER BY created_at")
	must(err == nil)
	var batches []api.UsageBatchRequest
	for rows.Next() {
		var raw []byte
		must(rows.Scan(&raw) == nil)
		var b api.UsageBatchRequest
		must(json.Unmarshal(raw, &b) == nil)
		if b.OriginIncarnationId != nil {
			must(*b.OriginIncarnationId == origin && b.NodeIncarnationId == current)
			batches = append(batches, b)
		}
	}
	must(rows.Err() == nil)
	_ = rows.Close()
	must(len(batches) > 0)
	detail["agent_origin_batches"] = batches
	p.expect("AT07", "identity_ledger_preserved", true, detail)
	p.expect("AT07", "historical_usage_retained", true, map[string]any{"owner": "usage", "assertion_source": usageAT20, "assertions": []string{"BeforePlusHeldByteEquality", "OriginalPayloadAndHistoricalAttribution", "OriginACKAndIndependentCurrentACK"}, "evidence": detail})
	// Prove the actual fresh stream starts at 1 after the retired stream is ACKed.
	step("fresh current-incarnation serving traffic", "agent")
	for i := 0; i < 121; i++ {
		must(waitRetry(p.ctx, "1"))
	}
	restoredAt = time.Now().UTC()
	must(p.stack.RestoreInstallerServices(p.ctx, "installer-node", states) == nil)
	must(p.installerPublish(node)) // revoke disables publication; restore through the public API
	freshRules, ok := p.installerServingReadback(node, restoredAt)
	must(ok)
	must(p.selectInstallerExit(node))
	freshTraffic, ok := p.installerTraffic(selected)
	must(ok)
	must(p.waitInstaller(90*time.Second, func() bool { ack := p.streamACK(node, current); return ack != "0" && ack != "unavailable" }))
	var firstSeq string
	must(p.freezeSQL(fmt.Sprintf("SELECT to_json(COALESCE(min(event_seq),0)::text) FROM usage_events WHERE node_id='%s' AND incarnation_id='%s'", node, current), &firstSeq) && firstSeq == "1")
	p.expect("AT07", "controlled_reinstall", true, map[string]any{"owner": "installer", "outcome": installerOutcomeView(reinstalled, nil), "serving": freshRules, "traffic": freshTraffic, "new_stream_first_sequence": firstSeq, "origin_ack_before_current_traffic": oldACK, "devstack_rows": devstack18 + " installer-reenroll / installer-serving-readback"})
}

func (p *probe) waitInstaller(d time.Duration, f func() bool) bool {
	until := time.Now().Add(d)
	for time.Now().Before(until) && p.ctx.Err() == nil {
		if f() {
			return true
		}
		if !waitRetry(p.ctx, "1") {
			return false
		}
	}
	return false
}
func (p *probe) stopInstaller() bool {
	for _, u := range []string{"fvpn-updater.timer", "fvpn-updater.service", "fvpn-watchdog.service", "fvpn-agent.service"} {
		if exec.CommandContext(p.ctx, "docker", "exec", installerContainer, "systemctl", "stop", u).Run() != nil {
			return false
		}
	}
	return true
}
func (p *probe) installerPublish(node uuid.UUID) bool {
	path := "/api/v1/admin/nodes/" + node.String()
	mutate := func(method, suffix string, body any) bool {
		r, e := p.call("GET", path, nil)
		if e != nil || r.Status != 200 {
			return false
		}
		r, e = p.callHeaders(method, path+suffix, body, map[string]string{"If-Match": r.Header.Get("ETag")})
		return e == nil && (r.Status == 200 || r.Status == 202)
	}
	if !mutate("PATCH", "", map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}) {
		return false
	}
	if !p.waitInstaller(90*time.Second, func() bool {
		v := p.identityProjection(node)
		return v["provisioning_state"] == "ready" && v["safety_status"] == "passed"
	}) {
		return false
	}
	return mutate("PUT", "/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, GroupIds: &[]uuid.UUID{p.servingGroup}, Reason: "TEST ONLY AT07 installer serving"})
}
func (p *probe) selectInstallerExit(node uuid.UUID) bool {
	var exit uuid.UUID
	if !p.waitInstaller(90*time.Second, func() bool {
		return p.freezeSQL(fmt.Sprintf("SELECT to_json(id) FROM exits WHERE node_id='%s'", node), &exit)
	}) {
		return false
	}
	sum := sha256.Sum256([]byte(exit.String()))
	network := "fvpn-exit-" + hex.EncodeToString(sum[:12])
	// Same explicit native selection used by AT20; no route or ledger mutation.
	return p.waitInstaller(60*time.Second, func() bool {
		raw := dockerText(p.ctx, "exec", "fvpn-w4-enroll-member", "netbird", "networks", "list")
		if !strings.Contains(raw, network) {
			return false
		}
		return exec.CommandContext(p.ctx, "docker", "exec", "fvpn-w4-enroll-member", "netbird", "networks", "select", network).Run() == nil
	})
}
func (p *probe) installerTraffic(s *e2e.Stack) (map[string]any, bool) {
	d := map[string]any{}
	var before e2e.DeviceCounters
	if !p.waitInstaller(60*time.Second, func() bool {
		var e error
		before, e = s.CountedMemberTraffic(p.ctx, p.servingDevice)
		return e == nil && before.Present
	}) {
		return d, false
	}
	_, err := s.StartEstablishedTransports(p.ctx, p.servingDevice)
	if err != nil {
		return d, false
	}
	defer func() { _ = s.StopEstablishedTransports(context.WithoutCancel(p.ctx)) }()
	var after e2e.DeviceCounters
	var status e2e.MemberTransportStatus
	ok := p.waitInstaller(15*time.Second, func() bool {
		after, err = s.CountedMemberTraffic(p.ctx, p.servingDevice)
		status, _ = s.EstablishedStatus(p.ctx)
		return err == nil && after.Upload > before.Upload && after.Download > before.Download && status.TCPEchoes > 0 && status.UDPReplies > 0
	})
	d["before"], d["after"], d["status"] = before, after, status
	return d, ok
}

// Text is retained: nft 1.0.9 JSON omits comments. Parse table scope explicitly;
// no global weakening of the four-chain verifier or removal of the owner marker.
func installerRuleShape(raw string) bool {
	table := ""
	depth := 0
	chains := map[string][]string{}
	marker := false
	hook := false
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if depth == 0 && len(f) >= 4 && f[0] == "table" {
			table = f[1] + " " + f[2]
		}
		if len(f) >= 2 && f[0] == "chain" {
			chains[table] = append(chains[table], f[1])
		}
		if table == "inet fvpn_install" {
			marker = marker || strings.Contains(line, `comment "managed-by=fvpn-node-install"`)
			hook = hook || strings.Contains(line, "hook ")
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth == 0 {
			table = ""
		}
	}
	sort.Strings(chains["inet fvpn"])
	return strings.Join(chains["inet fvpn"], ",") == "forward,input,output,postrouting" && strings.Join(chains["inet fvpn_install"], ",") == "owner" && marker && !hook
}
func (p *probe) installerServingReadback(node uuid.UUID, since time.Time) (map[string]any, bool) {
	d := map[string]any{}
	ok := p.waitInstaller(90*time.Second, func() bool {
		raw, e := exec.CommandContext(p.ctx, "docker", "exec", installerContainer, "nft", "list", "ruleset").Output()
		beat := dockerText(p.ctx, "exec", installerContainer, "cat", "/run/vpn-node/heartbeat")
		var wall time.Time
		serving := false
		for _, line := range strings.Split(beat, "\n") {
			k, v, _ := strings.Cut(line, " ")
			if k == "wall_unix_ns" {
				n, _ := strconv.ParseInt(v, 10, 64)
				wall = time.Unix(0, n).UTC()
			}
			if k == "serving" {
				serving = v == "1"
			}
		}
		var central struct {
			Received   time.Time `json:"received_at"`
			Forwarding bool      `json:"forwarding"`
			Isolation  string    `json:"internet_only"`
			Meter      string    `json:"meter_path"`
		}
		centralOK := p.freezeSQL(fmt.Sprintf("SELECT jsonb_build_object('received_at',received_at,'forwarding',report_payload->'member_forwarding_enabled','internet_only',report_payload->'checks'->>'internet_only','meter_path',report_payload->'checks'->>'meter_path') FROM node_observations WHERE node_id='%s' ORDER BY received_at DESC LIMIT 1", node), &central)
		active := exec.CommandContext(p.ctx, "docker", "exec", installerContainer, "systemctl", "is-active", "--quiet", "fvpn-agent.service").Run() == nil
		d["nft_ruleset_text"], d["heartbeat"], d["central"], d["read_ok"], d["agent_active"] = string(raw), beat, central, e == nil, active
		d["four_chains_hookless_marker"] = installerRuleShape(string(raw))
		return e == nil && installerRuleShape(string(raw)) && active && serving && wall.After(since) && time.Since(wall) < 15*time.Second && centralOK && central.Forwarding && central.Received.After(since)
	})
	return d, ok
}
func decimalSum(a, b string) string {
	x, ok := new(big.Int).SetString(a, 10)
	if !ok {
		panic("invalid decimal evidence")
	}
	y, ok := new(big.Int).SetString(b, 10)
	if !ok {
		panic("invalid decimal evidence")
	}
	return x.Add(x, y).String()
}
func snapshotBytes(s e2e.UsageSnapshot, member string) (string, string) {
	u, d := "0", "0"
	for _, e := range s.UnackedEvents {
		for _, r := range e.Records {
			if r.MemberID == member {
				u = decimalSum(u, r.UploadBytes)
				d = decimalSum(d, r.DownloadBytes)
			}
		}
	}
	return u, d
}
func (p *probe) memberUsage(period string) (api.MemberUsage, bool) {
	r, e := p.stack.Call(p.ctx, p.servingMember, "GET", "/api/v1/member/usage?period_id="+period, nil, nil)
	var v api.MemberUsage
	ok := e == nil && r.Status == 200 && json.Unmarshal(r.Body, &v) == nil
	return v, ok
}
func (p *probe) streamACK(node, inc uuid.UUID) string {
	var ack string
	if !p.freezeSQL(fmt.Sprintf("SELECT to_json(COALESCE((SELECT highest_contiguous_seq FROM usage_checkpoints WHERE node_id='%s' AND incarnation_id='%s'),0)::text)", node, inc), &ack) {
		return "unavailable"
	}
	return ack
}
func (p *probe) installerLedgerCopy(t *testing.T) *sql.DB {
	t.Helper()
	dir, err := os.MkdirTemp(os.Getenv("TMPDIR"), "installer-ledger-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	for _, name := range []string{"usage.db", "usage.db-wal"} {
		if name == "usage.db-wal" && exec.CommandContext(p.ctx, "docker", "exec", installerContainer, "test", "-f", "/var/lib/vpn-node/state/"+name).Run() != nil {
			continue
		}
		if _, err = p.stack.ExportContainerFile(p.ctx, "installer-node", "/var/lib/vpn-node/state/"+name, filepath.Join(dir, name)); err != nil {
			t.Fatal("coherent ledger export failed")
		}
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "usage.db")+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type retainedEvent struct {
	Event            api.UsageEvent  `json:"event"`
	Raw              json.RawMessage `json:"-"`
	PayloadUnchanged bool            `json:"canonical_payload_unchanged"`
	SamplesUnchanged bool            `json:"historical_samples_unchanged"`
	Count            int             `json:"central_count_by_payload_sha256"`
}

func readHeldEvents(t *testing.T, db *sql.DB) []retainedEvent {
	t.Helper()
	rows, err := db.Query("SELECT payload FROM events WHERE state='outbox' ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var events []retainedEvent
	for rows.Next() {
		var e retainedEvent
		if rows.Scan(&e.Raw) != nil || json.Unmarshal(e.Raw, &e.Event) != nil {
			t.Fatal("held event decode")
		}
		events = append(events, e)
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
	return events
}
func (p *probe) historicalEvents(events []retainedEvent) bool {
	all := true
	for i := range events {
		ev := &events[i]
		e := ev.Event
		var central struct {
			Payload     string
			Hash        string
			Incarnation string
			Sequence    string
			Count       int
		}
		query := fmt.Sprintf("SELECT jsonb_build_object('Payload',encode(p.payload,'hex'),'Hash',encode(e.payload_hash,'hex'),'Incarnation',e.incarnation_id,'Sequence',e.event_seq::text,'Count',(SELECT count(*) FROM usage_events WHERE payload_hash=e.payload_hash)) FROM usage_events e JOIN usage_event_payloads p ON p.event_id=e.id WHERE e.node_id='%s' AND e.incarnation_id='%s' AND e.event_seq=%s", e.NodeId, e.NodeIncarnationId, e.EventSeq)
		read := p.freezeSQL(query, &central)
		raw, err := hex.DecodeString(central.Payload)
		ev.PayloadUnchanged = read && err == nil && bytes.Equal(raw, ev.Raw) && central.Hash == e.PayloadSha256 && central.Incarnation == e.NodeIncarnationId.String() && central.Sequence == e.EventSeq.String()
		ev.Count = central.Count
		ev.SamplesUnchanged = true
		for idx, r := range e.Records {
			var n int
			q := fmt.Sprintf("SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id='%s' AND e.incarnation_id='%s' AND e.event_seq=%s AND s.record_index=%d AND s.member_id='%s' AND s.device_id='%s' AND s.binding_revision=%s AND s.upload_bytes=%s AND s.download_bytes=%s AND s.meter_epoch_id='%s' AND to_char(s.interval_start AT TIME ZONE 'UTC','YYYY-MM')='%s'", e.NodeId, e.NodeIncarnationId, e.EventSeq, idx, r.MemberId, r.DeviceId, r.BindingRevision, r.UploadBytes, r.DownloadBytes, e.MeterEpochId, r.PeriodId)
			ev.SamplesUnchanged = ev.SamplesUnchanged && p.freezeSQL(q, &n) && n == 1
		}
		all = all && ev.PayloadUnchanged && ev.SamplesUnchanged && ev.Count == 1
	}
	return all
}

func TestAT07_D116RejectsLegacyOrHookedMarker(t *testing.T) {
	good := `table inet fvpn_install {
 chain owner {
  comment "managed-by=fvpn-node-install"
 }
}
table inet fvpn {
 chain forward {
 }
 chain input {
 }
 chain output {
 }
 chain postrouting {
 }
}`
	if !installerRuleShape(good) {
		t.Fatal("valid separate marker rejected")
	}
	for _, bad := range []string{strings.Replace(good, "chain input {", "chain owner {", 1), strings.Replace(good, "comment \"managed-by=fvpn-node-install\"", "type filter hook forward priority filter;\n  comment \"managed-by=fvpn-node-install\"", 1), regexp.MustCompile(`table inet fvpn_install`).ReplaceAllString(good, "table inet foreign")} {
		if installerRuleShape(bad) {
			t.Fatal("invalid marker scope accepted")
		}
	}
}
