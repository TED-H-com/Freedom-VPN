//go:build e2e

package updates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (s *suite) idempotency() {
	s.logf("idempotency")
	admin := s.admin()
	if admin == nil {
		return
	}
	fleet, err := s.lab.fleet(s.ctx, admin)
	if err != nil {
		s.breakf("fleet before idempotency: %v", err)
		return
	}
	key := uuid.NewString()
	req := api.UpdateBatchCreateRequest{
		ReleaseId: openapi_types.UUID(s.rel["old"].ID), Scope: api.UpdateBatchCreateRequestScopeAllNodes,
		ExpectedGeneration: fleet.Generation, SetFleetDefault: api.UpdateBatchCreateRequestSetFleetDefaultTrue,
		Reason: "at26 same body",
	}
	s.cmd("at26", "POST /api/v1/admin/update-batches (same Idempotency-Key twice, then a different body)")
	first, acc1, err := s.lab.postBatch(s.ctx, admin, req, key)
	s.obs("at26", "first", httpDetail(first, err)+" batch="+acc1.BatchId.String())
	second, acc2, err2 := s.lab.postBatch(s.ctx, admin, req, key)
	s.obs("at26", "replay", httpDetail(second, err2)+" batch="+acc2.BatchId.String())
	changed := req
	changed.ReleaseId = openapi_types.UUID(s.rel["new"].ID)
	third, _, err3 := s.lab.postBatch(s.ctx, admin, changed, key)
	s.obs("at26", "conflict", httpDetail(third, err3))
	rows, qerr := s.lab.query(s.ctx, `SELECT response_status::text, state, count(*)::text FROM idempotency_requests WHERE route='/api/v1/admin/update-batches' AND method='POST' GROUP BY 1, 2 ORDER BY 1`)
	if qerr != nil {
		s.obs("at26", "idempotency_table", qerr.Error())
	} else {
		s.obs("at26", "idempotency_table", strings.Join(rows, "; "))
	}
	if err != nil || first.Status != 202 || err2 != nil || second.Status != 202 || acc1.BatchId != acc2.BatchId || acc1.BatchId == uuid.Nil {
		s.breakf("idempotency replay did not return one batch")
		return
	}
	if err3 != nil || third.Status != 409 || third.Code != "IDEMPOTENCY_CONFLICT" {
		s.breakf("idempotency conflict: %s", httpDetail(third, err3))
		return
	}
	s.obs("at26", "one_batch", acc1.BatchId.String())
	s.obs("at26", "safe_switch", "blocked: stock NodeChecks cannot pass on a non-serving node, so a completed safe switch is not claimed")
}

func (s *suite) population() {
	s.logf("population")
	admin := s.admin()
	if admin == nil {
		return
	}
	s.cmd("at27", "PUT /api/v1/admin/nodes/{id}/publication")
	pub, err := s.setPublication(admin, s.primary.NodeID, api.PublicationIntentPublished)
	s.obs("at23", "publish_attempt", pub)
	s.obs("at27", "publish_attempt", pub)
	if err != nil {
		s.obs("at23", "published_role", "not created: "+err.Error())
	}
	var errD, errO error
	s.disabled, errD = s.enrollAux(admin, "disabled", s.lab.labIP+".42", true)
	s.offline, errO = s.enrollAux(admin, "offline", s.lab.labIP+".43", true)
	if errD != nil || errO != nil {
		s.breakf("aux enroll disabled=%v offline=%v", errD, errO)
		return
	}
	disPub, err := s.setPublication(admin, s.disabled.identity.NodeID, api.PublicationIntentDisabled)
	s.obs("at27", "disable", disPub)
	if err != nil || !strings.Contains(disPub, "disabled") {
		s.obs("at27", "disable_applied", "false")
	}
	held, token, err := s.issue(admin, "at-enrolling", 7200)
	if err != nil {
		s.breakf("held enrollment: %v", err)
		return
	}
	s.held = token
	s.heldID = held
	s.obs("at24", "enrolling_node", held.String())
	s.obs("at24", "enrolling_token", "withheld until after the batch")
	off, _, err := s.lab.getNode(s.ctx, admin, s.offline.identity.NodeID)
	if err == nil {
		var detail api.NodeDetail
		_ = json.Unmarshal(off.Body, &detail)
		s.obs("at24", "offline_control_status_before_batch", string(detail.ControlStatus))
	}
}

func (s *suite) enrollAux(admin *session, role, ip string, pause bool) (auxNode, error) {
	node, err := s.lab.startAux(role, ip, pause)
	if err != nil {
		return auxNode{}, err
	}
	s.aux = append(s.aux, node)
	id, token, err := s.issue(admin, "at-"+role, 7200)
	if err != nil {
		return node, err
	}
	if err := s.lab.writeToken(node.varDir, token); err != nil {
		return node, err
	}
	wait, cancel := context.WithTimeout(s.ctx, 4*time.Minute)
	ok := node.waitEnrolled(wait)
	cancel()
	if !ok {
		return node, fmt.Errorf("%s did not enroll", role)
	}
	ident, err := s.lab.readNodeIdentity(node.container)
	if err != nil {
		return node, err
	}
	if ident.NodeID != id {
		return node, fmt.Errorf("%s identity %s enrollment %s", role, ident.NodeID, id)
	}
	node.identity = ident
	for i := range s.aux {
		if s.aux[i].container == node.container {
			s.aux[i] = node
		}
	}
	return node, nil
}

func (s *suite) setPublication(admin *session, id uuid.UUID, intent api.PublicationIntent) (string, error) {
	detail, _, err := s.lab.getNode(s.ctx, admin, id)
	if err != nil || detail.Status != 200 {
		return httpDetail(detail, err), fmt.Errorf("get node")
	}
	etag := detail.Header.Get("ETag")
	resp, err := admin.call(s.ctx, "PUT", s.lab.public+"/api/v1/admin/nodes/"+id.String()+"/publication", api.PublicationRequest{
		PublicationIntent: intent, Reason: "at acceptance",
	}, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": etag})
	if err != nil {
		return httpDetail(resp, err), err
	}
	got := ""
	deadline := time.Now().Add(45 * time.Second)
	for {
		again, view, err2 := s.lab.getNode(s.ctx, admin, id)
		if err2 == nil && again.Status == 200 {
			got = string(view.PublicationIntent)
			if got == string(intent) {
				break
			}
		}
		if resp.Status >= 400 || time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return httpDetail(resp, nil) + " etag=" + etag + " intent=" + got, nil
}

func (s *suite) fleetBatch() {
	s.logf("fleet batch")
	admin := s.admin()
	if admin == nil {
		return
	}
	s.cmd("at23", "POST /api/v1/admin/update-batches scope=all_nodes set_fleet_default=true")
	s.lab.unpause(s.varDir)
	onlineWait, onlineCancel := context.WithTimeout(s.ctx, 75*time.Second)
	primaryOnline := waitUntil(onlineWait, 3*time.Second, func() bool {
		_, view, err := s.lab.getNode(s.ctx, admin, s.primary.NodeID)
		return err == nil && string(view.ControlStatus) == "online"
	})
	onlineCancel()
	s.obs("at23", "primary_online_before_batch", fmt.Sprintf("%t", primaryOnline))
	resp, acc, err := s.lab.createBatch(s.ctx, admin, s.rel["new"].ID, "at23 one fleet batch", uuid.NewString())
	s.obs("at23", "batch", httpDetail(resp, err))
	if err != nil || resp.Status != 202 {
		s.breakf("fleet batch: %s", httpDetail(resp, err))
		return
	}
	s.batch = acc
	s.obs("at23", "batch_id", acc.BatchId.String())
	s.obs("at23", "fleet_generation", acc.FleetGeneration.String())
	s.obs("at23", "fleet_default", acc.FleetDefaultReleaseId.String())
	s.obs("at23", "target_count", fmt.Sprintf("%d", acc.TargetCount))
	s.obs("at23", "online_at_snapshot", fmt.Sprintf("%d", acc.OnlineAtSnapshot))
	s.obs("at23", "offline_at_snapshot", fmt.Sprintf("%d", acc.OfflineAtSnapshot))
	s.obs("at24", "batch_id", acc.BatchId.String())
	s.obs("at24", "fleet_default", acc.FleetDefaultReleaseId.String())
	got, err := s.lab.batch(s.ctx, admin, acc.BatchId)
	if err != nil {
		s.breakf("get batch: %v", err)
		return
	}
	want := map[uuid.UUID]string{
		s.primary.NodeID:           "primary",
		s.disabled.identity.NodeID: "disabled",
		s.offline.identity.NodeID:  "offline",
		s.heldID:                   "enrolling",
	}
	seen := map[uuid.UUID]bool{}
	for _, target := range got.Targets {
		role := want[target.NodeId]
		if role == "" {
			role = "other"
		}
		seen[target.NodeId] = true
		job := "none"
		if !target.CurrentJob.IsNull() {
			job = string(target.State)
		}
		inc := "null"
		if !target.IncarnationAtSnapshot.IsNull() {
			inc = "set"
		}
		s.obs("at23", "target_"+role, fmt.Sprintf("online=%t state=%s job=%s incarnation_at_snapshot=%s needs_attention=%t unconfirmed=%t",
			target.OnlineAtSnapshot, target.State, job, inc, target.NeedsAttention, target.Unconfirmed))
	}
	for id, role := range want {
		if !seen[id] {
			s.breakf("snapshot missing %s", role)
		}
	}
	if acc.TargetCount != len(want) {
		s.obs("at23", "target_count_note", fmt.Sprintf("got %d want %d", acc.TargetCount, len(want)))
		if acc.TargetCount < len(want) {
			s.breakf("target_count %d", acc.TargetCount)
		}
	}
	s.obs("at23", "ssh", "no per-node SSH and no per-batch approval were used")
}

func (s *suite) finding() {
	s.logf("finding")
	s.cmd("update-good-finding", "fvpn-updater run on the primary node, first install")
	before := s.lab.identitySHA(s.lab.node)
	release := s.rel["old"].ID.String()
	if s.batch.BatchId != uuid.Nil {
		release = s.rel["new"].ID.String()
	}
	run := s.runRelease(s.lab.node, s.varDir, s.primary, release, "amd64", "", "", "", nil, 4*time.Minute)
	view := s.lab.journalForRelease(s.lab.node, s.varDir, release)
	if view.ErrorCode == "" {
		view = run.Journal
	}
	class := ClassifyFirstInstallRollback(RollbackFacts{
		HealthJSON: view.Health, ErrorCode: view.ErrorCode, ErrorDetail: view.ErrorDetail,
		LastGood: view.LastGood, RollbackRelease: view.RollbackRelease, Events: view.Events,
	})
	s.obs("update-good-finding", "exit_code", fmt.Sprintf("%d", run.Exit))
	s.obs("update-good-finding", "finished", fmt.Sprintf("%t", run.Finished))
	s.obs("update-good-finding", "classification", class)
	s.obs("update-good-finding", "journal", clip(view.Text, 500))
	s.obs("update-good-finding", "events", clip(view.Events, 500))
	s.obs("update-good-finding", "updater_out", clip(run.Out, 400))
	s.obs("update-good-finding", "artifact_sha", view.ArtifactSHA)
	s.obs("update-good-finding", "amd64_sha", s.rel["new"].AMDSHA)
	s.obs("update-good-finding", "arm64_sha", s.rel["new"].ARMSHA)
	s.obs("update-good-finding", "downloaded_amd64", fmt.Sprintf("%t", view.ArtifactSHA == s.rel["new"].AMDSHA && view.ArtifactSHA != ""))
	s.obs("update-good-finding", "downloaded_arm64", fmt.Sprintf("%t", view.ArtifactSHA == s.rel["new"].ARMSHA && view.ArtifactSHA != ""))
	s.obs("update-good-finding", "identity_unchanged", fmt.Sprintf("%t", before != "" && before == s.lab.identitySHA(s.lab.node)))
	s.obs("update-good-finding", "forward_chain", s.lab.forwardSummary(s.lab.node))
	s.obs("update-good-finding", "cited_run", "docs/e2e/runs/2026-10-29/update-good.md 2026-10-29T22:48:08Z")
	s.obs("at23", "primary_exit", fmt.Sprintf("%d", run.Exit))
	s.obs("at23", "primary_health", clip(view.Health, 300))
	s.obs("at23", "primary_error", view.ErrorCode)
	s.obs("at23", "amd64_artifact_selected", fmt.Sprintf("%t", view.ArtifactSHA == s.rel["new"].AMDSHA && view.ArtifactSHA != ""))
	s.obs("at23", "current_release", view.Current)
	s.obs("at23", "last_good", empty(view.LastGood))
	if view.ErrorCode != "ROLLBACK_FAILED" {
		s.miss = true
		s.rec["update-good-finding"].Actual = "error_code=" + empty(view.ErrorCode)
		return
	}
	s.rec["update-good-finding"].Actual = "error_code=ROLLBACK_FAILED classification=" + class
	switch class {
	case FindingLabPrecondition:
		s.obs("update-good-finding", "decision", "Lab precondition. node/updater/health.go NodeChecks.Isolation, DNSPolicy and TunnelProbe return failed (internet_only, dns_policy, tunnel_probe). apply.go stepCheck then stepRollback. decide.go DecideRollback returns ROLLBACK_FAILED when HasLastGood is false. pickRollback does not call SwitchSymlink when the target is empty. No updater defect is filed from this journal.")
		s.rec["update-good-finding"].Status = StatusBlocked
	case FindingUpdaterDefect:
		s.obs("update-good-finding", "decision", "Journal shows a last-good release or a restore. Treat as an updater defect in node/updater DecideRollback / stepRollback.")
		s.rec["update-good-finding"].Status = StatusBlocked
	default:
		s.obs("update-good-finding", "decision", "ROLLBACK_FAILED was recorded but the health JSON did not match the stock fail-closed checks.")
	}
	s.rec["at23"].Actual = "batch snapshotted the mixed set; primary health did not pass, so the batch is not a success"
	s.rec["at23"].Status = StatusBlocked
	s.obs("at23", "success_rule", "locally_succeeded was not observed; AT23 success is not claimed")
}

func (s *suite) waitRelease(container, varDir, release string) bool {
	s.lab.unpause(varDir)
	wait, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	defer cancel()
	ok := waitUntil(wait, 2*time.Second, func() bool { return s.lab.desiredHas(container, release) })
	_ = s.lab.pause(varDir)
	time.Sleep(2 * time.Second)
	return ok
}

func (s *suite) runRelease(container, varDir string, node nodeIdentity, release, arch, trust, staging, releases string, stop []string, timeout time.Duration) updaterRun {
	s.logf("updater %s release %s arch %s", container, release, arch)
	seen := s.waitRelease(container, varDir, release)
	if !seen {
		s.logf("desired state never contained %s", release)
	}
	run, err := s.lab.runUpdater(s.ctx, runOpts{
		container: container, varDir: varDir, node: node, arch: arch,
		releaseTrust: trust, staging: staging, releases: releases, stop: stop, timeout: timeout,
	})
	if err != nil {
		s.logf("updater: %v", err)
	}
	return run
}

func empty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (s *suite) catchUp() {
	s.logf("catch-up")
	admin := s.admin()
	if admin == nil {
		return
	}
	s.cmd("at24", "unpause offline node and run fvpn-updater; deliver held enrollment; create the post-batch node")
	offRun := s.runRelease(s.offline.container, s.offline.varDir, s.offline.identity, s.rel["new"].ID.String(), "amd64", "", "", "", nil, 4*time.Minute)
	s.obs("at24", "offline_updater_exit", fmt.Sprintf("%d", offRun.Exit))
	s.obs("at24", "offline_error", offRun.Journal.ErrorCode)
	got, err := s.lab.batch(s.ctx, admin, s.batch.BatchId)
	if err == nil {
		for _, target := range got.Targets {
			if target.NodeId == s.offline.identity.NodeID {
				execInc := "null"
				if !target.ExecutionIncarnationId.IsNull() {
					execInc = "set"
				}
				s.obs("at24", "offline_after_auth", fmt.Sprintf("state=%s execution_incarnation=%s", target.State, execInc))
			}
			if target.NodeId == s.heldID {
				execInc := "null"
				if !target.ExecutionIncarnationId.IsNull() {
					execInc = "set"
				}
				job := "none"
				if !target.CurrentJob.IsNull() {
					job = "present"
				}
				s.obs("at24", "enrolling_before_join", fmt.Sprintf("incarnation_at_snapshot_null=%t job=%s execution=%s", target.IncarnationAtSnapshot.IsNull(), job, execInc))
			}
		}
	}
	node, err := s.lab.startAux("enrolling", s.lab.labIP+".44", false)
	if err != nil {
		s.obs("at24", "enrolling_start", err.Error())
	} else {
		s.aux = append(s.aux, node)
		s.enroll = node
		if err := s.lab.writeToken(node.varDir, s.held); err != nil {
			s.obs("at24", "enrolling_token_write", "failed")
		}
		s.held = ""
		wait, cancel := context.WithTimeout(s.ctx, 4*time.Minute)
		ok := node.waitEnrolled(wait)
		cancel()
		s.obs("at24", "enrolling_enrolled", fmt.Sprintf("%t", ok))
		if ok {
			ident, err := s.lab.readNodeIdentity(node.container)
			if err == nil {
				s.enroll.identity = ident
				_ = s.runRelease(node.container, node.varDir, ident, s.rel["new"].ID.String(), "amd64", "", "", "", nil, 4*time.Minute)
			}
		}
	}
	lateID, lateToken, err := s.issue(admin, "at-late", 7200)
	if err != nil {
		s.obs("at24", "late_issue", err.Error())
		s.rec["at24"].Actual = "late node was not created"
		return
	}
	late, err := s.lab.startAux("late", s.lab.labIP+".45", false)
	if err != nil {
		s.obs("at24", "late_start", err.Error())
		return
	}
	s.aux = append(s.aux, late)
	s.late = late
	_ = s.lab.writeToken(late.varDir, lateToken)
	wait, cancel := context.WithTimeout(s.ctx, 4*time.Minute)
	enrolled := late.waitEnrolled(wait)
	cancel()
	s.obs("at24", "late_enrolled", fmt.Sprintf("%t", enrolled))
	s.obs("at24", "late_node", lateID.String())
	if enrolled {
		s.lab.unpause(late.varDir)
		wait, cancel = context.WithTimeout(s.ctx, 3*time.Minute)
		found := waitUntil(wait, 3*time.Second, func() bool {
			rows, err := s.lab.query(s.ctx, `SELECT id::text, reason, target_count::text FROM update_batches WHERE first_sync_node_id=$1`, lateID)
			return err == nil && len(rows) > 0
		})
		cancel()
		rows, _ := s.lab.query(s.ctx, `SELECT id::text, reason, target_count::text FROM update_batches WHERE first_sync_node_id=$1`, lateID)
		s.obs("at24", "first_sync", fmt.Sprintf("found=%t %s", found, strings.Join(rows, "; ")))
	}
	again, err := s.lab.batch(s.ctx, admin, s.batch.BatchId)
	if err != nil {
		s.obs("at24", "original_batch", err.Error())
		return
	}
	s.obs("at24", "original_target_count", fmt.Sprintf("%d", again.TargetCount))
	lateInOld := false
	for _, target := range again.Targets {
		if target.NodeId == lateID {
			lateInOld = true
		}
	}
	s.obs("at24", "late_in_original_snapshot", fmt.Sprintf("%t", lateInOld))
	fleet, err := s.lab.fleet(s.ctx, admin)
	if err == nil && !fleet.FleetDefaultReleaseId.IsNull() {
		def, _ := fleet.FleetDefaultReleaseId.Get()
		s.obs("at24", "fleet_default_now", uuid.UUID(def).String())
	}
	detail, _, err := s.lab.getNode(s.ctx, admin, lateID)
	if err == nil {
		var view api.NodeDetail
		_ = json.Unmarshal(detail.Body, &view)
		desired := "null"
		if !view.Releases.DesiredReleaseId.IsNull() {
			id, _ := view.Releases.DesiredReleaseId.Get()
			desired = uuid.UUID(id).String()
		}
		s.obs("at24", "late_desired", desired)
		s.obs("at24", "late_provisioning", string(view.ProvisioningState))
		s.obs("at24", "late_lifecycle", string(view.LifecycleState))
	}
	sameCount := again.TargetCount == s.batch.TargetCount && !lateInOld
	s.rec["at24"].Actual = fmt.Sprintf("original target_count stayed %d; late backfilled=%t", again.TargetCount, lateInOld)
	if sameCount {
		s.obs("at24", "inheritance", "the post-batch node is absent from the original snapshot")
	}
	s.rec["at24"].Status = StatusBlocked
	s.obs("at24", "note", "Catch-up ran. Health confirmation is blocked by the same non-serving NodeChecks, so AT24 is not passed.")
}

func (s *suite) staleAck() {
	s.logf("stale ack")
	admin := s.admin()
	if admin == nil {
		return
	}
	s.cmd("at26", "POST /api/v1/agent/update-results applying, then a newer batch, then the old terminal result")
	resp, acc, err := s.lab.createBatch(s.ctx, admin, s.rel["old"].ID, "at26 applying window", uuid.NewString())
	s.obs("at26", "applying_batch", httpDetail(resp, err))
	if err != nil || resp.Status != 202 {
		s.obs("at26", "stale", "batch was not accepted")
		return
	}
	if !s.waitRelease(s.lab.node, s.varDir, s.rel["old"].ID.String()) {
		s.obs("at26", "issued", "desired state did not contain the release; the job was not authorized")
		return
	}
	job, ok := s.lab.parseIssued(s.lab.node)
	if !ok || job.JobID == "" {
		s.obs("at26", "issued", "update intent absent")
		return
	}
	s.obs("at26", "job", job.JobID+" generation="+job.Generation+" attempt="+fmt.Sprintf("%d", job.Attempt))
	before, _ := s.lab.scalar(s.ctx, `SELECT COALESCE(current_release_id::text,'') FROM nodes WHERE id=$1`, s.primary.NodeID)
	status, ack, err := s.lab.postResult(s.lab.node, s.varDir, s.primary, job, "applying", false)
	s.obs("at26", "applying_post", fmt.Sprintf("http %d %s %v", status, clip(ack, 180), err))
	next, _, err := s.lab.createBatch(s.ctx, admin, s.rel["new"].ID, "at26 newer batch", uuid.NewString())
	s.obs("at26", "newer_batch", httpDetail(next, err))
	oldState, _ := s.lab.scalar(s.ctx, `SELECT state FROM update_jobs WHERE id=$1`, job.JobID)
	s.obs("at26", "old_job_state_after_newer_batch", oldState)
	superseded, _ := s.lab.query(s.ctx, `SELECT node_id::text, state FROM update_jobs WHERE state='superseded' AND generation < (SELECT generation FROM update_jobs WHERE id=$1) OR id IN (SELECT id FROM update_jobs WHERE state='superseded')`, job.JobID)
	s.obs("at26", "superseded_rows", clip(strings.Join(superseded, "; "), 300))
	status, ack, err = s.lab.postResult(s.lab.node, s.varDir, s.primary, job, "failed", true)
	s.obs("at26", "stale_post", fmt.Sprintf("http %d %s %v", status, clip(ack, 240), err))
	var parsed api.UpdateResultAck
	_ = json.Unmarshal([]byte(ack), &parsed)
	after, _ := s.lab.scalar(s.ctx, `SELECT COALESCE(current_release_id::text,'') FROM nodes WHERE id=$1`, s.primary.NodeID)
	s.obs("at26", "accepted_as_history", fmt.Sprintf("%t", parsed.AcceptedAsHistory))
	s.obs("at26", "applied_to_current", fmt.Sprintf("%t", parsed.AppliedToCurrentObservation))
	s.obs("at26", "current_release_before", empty(before))
	s.obs("at26", "current_release_after", empty(after))
	s.obs("at26", "history_vs_current", fmt.Sprintf("history=%t current_unchanged=%t", parsed.AcceptedAsHistory, before == after))
	if status == 202 && parsed.AcceptedAsHistory && before == after && oldState == "applying" {
		s.obs("at26", "stale_ack_clause", "holds")
	} else {
		s.obs("at26", "stale_ack_clause", "not fully observed")
	}
	_ = acc
}

func (s *suite) faults() {
	s.logf("faults")
	s.cmd("at25", "tamper amd64 object; strip release trust; riscv64 platform; schema-from 2; tmpfs staging; nft reject origin; docker kill during stop")
	s.preSwitch("hash", s.rel["tamper"].ID, func() error {
		body, err := s.lab.getObject(s.ctx, "r/"+s.rel["tamper"].ID.String()+"/amd64.tar.zst")
		if err != nil {
			return err
		}
		if len(body) == 0 {
			return fmt.Errorf("empty artifact")
		}
		body[len(body)/2] ^= 0xff
		return s.lab.putObject(s.ctx, "r/"+s.rel["tamper"].ID.String()+"/amd64.tar.zst", body)
	}, "amd64", "", "", "", nil, 4*time.Minute, "HASH_MISMATCH")

	trustPath := s.varDir + "/e2e/trust-no-release.json"
	trustRaw, terr := s.lab.cat(s.lab.node, "/etc/vpn-node/trust.json")
	if terr != nil {
		s.obs("at25", "trust_strip", terr.Error())
	} else if err := writeTrustWithoutRelease(trustRaw, trustPath); err != nil {
		s.obs("at25", "trust_strip", err.Error())
	} else {
		s.preSwitch("key", s.rel["old"].ID, nil, "amd64", "/var/lib/vpn-node/e2e/trust-no-release.json", "", "", nil, 4*time.Minute, "KEY_UNTRUSTED")
	}
	s.preSwitch("arch", s.rel["old"].ID, nil, "riscv64", "", "", "", nil, 4*time.Minute, "UNSUPPORTED_PLATFORM")
	s.preSwitch("schema", s.rel["schema"].ID, nil, "amd64", "", "", "", nil, 4*time.Minute, "SCHEMA_INCOMPATIBLE")

	mountOut, mountErr := s.lab.exec(20*time.Second, s.lab.node, "sh", "-c", "mkdir -p /var/lib/vpn-node/e2e/tiny && mount -t tmpfs -o size=16384 tmpfs /var/lib/vpn-node/e2e/tiny")
	s.obs("at25", "tmpfs", clip(string(mountOut), 200))
	if mountErr != nil {
		s.obs("at25", "disk", "tmpfs mount failed; DISK_SPACE_LOW was not exercised")
	} else {
		// §13.7: the releases directory must hold the artifact, the installed
		// tree and required_free_bytes. Pointing it at the 16KiB tmpfs fails
		// that reserve before any tree is written.
		s.preSwitch("disk", s.rel["bulky"].ID, nil, "amd64", "", "/var/lib/vpn-node/e2e/tiny", "/var/lib/vpn-node/e2e/tiny", nil, 4*time.Minute, "DISK_SPACE_LOW")
		_, _ = s.lab.exec(15*time.Second, s.lab.node, "umount", "/var/lib/vpn-node/e2e/tiny")
	}

	s.interrupt()
	s.powerLoss()
	s.rec["at25"].Status = StatusBlocked
	s.rec["at25"].Actual = "pre-switch faults were recorded individually; restore to a healthy last-good is blocked because NodeChecks never pass and last_good stays empty"
	s.obs("at25", "restore", "blocked on the serving-node health precondition")
}

func (s *suite) preSwitch(label string, release uuid.UUID, mutate func() error, arch, trust, staging, releases string, stop []string, timeout time.Duration, want string) {
	s.logf("fault %s", label)
	admin := s.admin()
	if admin == nil {
		return
	}
	if mutate != nil {
		if err := mutate(); err != nil {
			s.obs("at25", label+"_setup", err.Error())
			return
		}
	}
	beforeLink := s.lab.currentLink(s.lab.node)
	beforeID := s.lab.identitySHA(s.lab.node)
	resp, _, err := s.lab.createBatch(s.ctx, admin, release, "at25 "+label, uuid.NewString())
	if err != nil || resp.Status != 202 {
		s.obs("at25", label+"_batch", httpDetail(resp, err))
		return
	}
	run := s.runRelease(s.lab.node, s.varDir, s.primary, release.String(), arch, trust, staging, releases, stop, timeout)
	if !run.Finished {
		s.lab.stopUpdater(s.lab.node, s.varDir)
	}
	view := s.lab.journalForRelease(s.lab.node, s.varDir, release.String())
	installed := s.lab.releaseTree(s.lab.node, release.String())
	s.obs("at25", label, fmt.Sprintf("want=%s error=%s exit=%d installed=%t link_unchanged=%t identity_unchanged=%t out=%s",
		want, view.ErrorCode, run.Exit, installed, s.lab.currentLink(s.lab.node) == beforeLink, s.lab.identitySHA(s.lab.node) == beforeID, clip(run.Out, 180)))
}

func writeTrustWithoutRelease(b []byte, dst string) error {
	var doc struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	kept := make([]map[string]any, 0, len(doc.Keys))
	dropped := 0
	for _, key := range doc.Keys {
		raw, _ := json.Marshal(key["purposes"])
		if strings.Contains(string(raw), "release") {
			dropped++
			continue
		}
		kept = append(kept, key)
	}
	if dropped == 0 {
		return fmt.Errorf("no release key")
	}
	out, err := json.Marshal(map[string]any{"keys": kept})
	if err != nil {
		return err
	}
	return os.WriteFile(dst, append(out, '\n'), 0o644)
}

func (s *suite) interrupt() {
	before := s.lab.currentLink(s.lab.node)
	beforeID := s.lab.identitySHA(s.lab.node)
	script := fmt.Sprintf("nft add table inet atupd && nft add chain inet atupd out '{ type filter hook output priority 0 ; policy accept ; }' && nft add rule inet atupd out ip daddr %s reject", s.lab.originIP)
	if out, err := s.lab.exec(15*time.Second, s.lab.node, "sh", "-c", script); err != nil {
		s.obs("at25", "interrupt_nft", clip(string(out), 200)+" "+err.Error())
		return
	}
	defer func() { _, _ = s.lab.exec(15*time.Second, s.lab.node, "nft", "delete", "table", "inet", "atupd") }()
	admin := s.admin()
	if admin == nil {
		return
	}
	resp, _, err := s.lab.createBatch(s.ctx, admin, s.rel["bulky"].ID, "at25 interrupt", uuid.NewString())
	if err != nil || resp.Status != 202 {
		s.obs("at25", "interrupt_batch", httpDetail(resp, err))
		return
	}
	run := s.runRelease(s.lab.node, s.varDir, s.primary, s.rel["bulky"].ID.String(), "amd64", "", "", "", nil, 3*time.Minute)
	if !run.Finished {
		s.lab.stopUpdater(s.lab.node, s.varDir)
	}
	view := s.lab.journalForRelease(s.lab.node, s.varDir, s.rel["bulky"].ID.String())
	s.obs("at25", "interrupt", fmt.Sprintf("error=%s exit=%d link_unchanged=%t identity_unchanged=%t installed=%t",
		view.ErrorCode, run.Exit, s.lab.currentLink(s.lab.node) == before, s.lab.identitySHA(s.lab.node) == beforeID, s.lab.releaseTree(s.lab.node, s.rel["bulky"].ID.String())))
}

func (s *suite) powerLoss() {
	admin := s.admin()
	if admin == nil {
		return
	}
	resp, _, err := s.lab.createBatch(s.ctx, admin, s.rel["old"].ID, "at25 power", uuid.NewString())
	if err != nil || resp.Status != 202 {
		s.obs("at25", "power_batch", httpDetail(resp, err))
		return
	}
	if !s.waitRelease(s.lab.node, s.varDir, s.rel["old"].ID.String()) {
		s.obs("at25", "power", "desired state was not issued")
		return
	}
	beforeID := s.lab.identitySHA(s.lab.node)
	_, err = s.lab.runUpdater(s.ctx, runOpts{
		container: s.lab.node, varDir: s.varDir, node: s.primary,
		stop: []string{"/bin/sleep", "45"}, start: []string{"/bin/true"}, timeout: 1 * time.Second,
	})
	// timeout 1s returns before the sleep ends; the entrypoint still runs the updater.
	_ = err
	deadline := time.Now().Add(40 * time.Second)
	stage := ""
	for time.Now().Before(deadline) {
		view := s.lab.readJournal(s.lab.node, s.varDir)
		stage = view.Stage
		if stage == "final_sampled" || stage == "stopped" || stage == "migrated" || stage == "switched" {
			break
		}
		time.Sleep(time.Second)
	}
	s.lab.hideToken(s.varDir)
	kill := exec.Command("docker", "kill", s.lab.node)
	killOut, killErr := kill.CombinedOutput()
	// The entrypoint runs whatever updater.json is still present. Recover
	// uses its own file, so the killed run's config must not start again.
	_ = os.Remove(s.varDir + "/e2e/updater.json")
	start := exec.Command("docker", "start", s.lab.node)
	startOut, startErr := start.CombinedOutput()
	s.obs("at25", "power_kill", fmt.Sprintf("stage_at_kill=%s kill=%s %v start=%s %v", stage, clip(string(killOut), 80), killErr, clip(string(startOut), 80), startErr))
	time.Sleep(3 * time.Second)
	cfg := s.recoverConfig()
	if cfg != nil {
		raw, _ := json.MarshalIndent(cfg, "", "  ")
		_ = os.WriteFile(s.varDir+"/e2e/updater-recover.json", append(raw, '\n'), 0o644)
		out, err := s.lab.exec(3*time.Minute, s.lab.node, "fvpn-updater", "recover", "-config", "/var/lib/vpn-node/e2e/updater-recover.json")
		s.obs("at25", "power_recover", clip(string(out), 300)+" "+fmt.Sprint(err))
	}
	s.obs("at25", "power_identity_unchanged", fmt.Sprintf("%t", beforeID != "" && beforeID == s.lab.identitySHA(s.lab.node)))
	s.obs("at25", "power_boot", s.lab.forwardSummary(s.lab.node))
}

func (s *suite) recoverConfig() map[string]any {
	bootRaw, err := s.lab.exec(15*time.Second, s.lab.node, "cat", "/etc/vpn-node/boot-id")
	if err != nil {
		return nil
	}
	boot := strings.TrimSpace(string(bootRaw))
	return map[string]any{
		"node_id": s.primary.NodeID.String(), "incarnation_id": s.primary.Incarnation.String(),
		"boot_id": boot, "revoked": false,
		"platform": map[string]any{
			"os_id": "ubuntu", "os_version": "24.04", "architecture": "amd64", "updater_version": "1.0.0",
			"control_api_major": 1, "policy_schema": 1, "usage_schema": 1, "state_schema": 1,
		},
		"release_trust": "/etc/vpn-node/trust.json", "desired_trust": "/etc/vpn-node/trust.json",
		"desired_state": "/var/lib/vpn-node/state/desired-state.json", "meter_checkpoint": "/var/lib/vpn-node/e2e/meter.json",
		"stop_command": []string{"/bin/true"}, "start_command": []string{"/bin/true"},
		"result_url":               s.lab.agent + "/api/v1/agent/update-results",
		"allowed_artifact_origins": []string{s.lab.agent, s.lab.releases},
		"mtls_cert":                "/var/lib/vpn-node/identity/cert.pem", "mtls_key": "/var/lib/vpn-node/identity/key.pem",
		"mtls_ca": "/etc/vpn-node/ca.pem",
	}
}

func (s *suite) versions() {
	s.logf("versions")
	admin := s.admin()
	if admin == nil {
		return
	}
	s.cmd("at27", "GET node publication; docker restart disabled; POST batch for a release removed from the allowlist; nft list chain inet fvpn forward")
	before, _, err := s.lab.getNode(s.ctx, admin, s.disabled.identity.NodeID)
	var beforeIntent string
	if err == nil {
		var view api.NodeDetail
		_ = json.Unmarshal(before.Body, &view)
		beforeIntent = string(view.PublicationIntent)
	}
	s.lab.hideToken(s.disabled.varDir)
	out, err := exec.Command("docker", "restart", "-t", "10", s.disabled.container).CombinedOutput()
	s.obs("at27", "disabled_restart", clip(string(out), 120)+" "+fmt.Sprint(err))
	time.Sleep(3 * time.Second)
	after, _, err := s.lab.getNode(s.ctx, admin, s.disabled.identity.NodeID)
	afterIntent := ""
	if err == nil {
		var view api.NodeDetail
		_ = json.Unmarshal(after.Body, &view)
		afterIntent = string(view.PublicationIntent)
		s.obs("at27", "disabled_lifecycle", string(view.LifecycleState))
	}
	s.obs("at27", "disabled_intent_before", empty(beforeIntent))
	s.obs("at27", "disabled_intent_after", empty(afterIntent))
	if beforeIntent == "disabled" && afterIntent == "disabled" {
		s.obs("at27", "disabled_stays_disabled", "true")
	}
	kept := []uuid.UUID{s.rel["old"].ID, s.rel["new"].ID, s.rel["tamper"].ID, s.rel["bulky"].ID}
	if err := s.lab.allowReleases(s.ctx, kept); err != nil {
		s.obs("at27", "allowlist", err.Error())
	}
	resp, _, err := s.lab.createBatch(s.ctx, admin, s.rel["schema"].ID, "at27 below minimum", uuid.NewString())
	s.obs("at27", "below_minimum_batch", httpDetail(resp, err))
	_ = s.lab.allowReleases(s.ctx, append(kept, s.rel["schema"].ID))
	primary, _, err := s.lab.getNode(s.ctx, admin, s.primary.NodeID)
	if err == nil {
		var view api.NodeDetail
		_ = json.Unmarshal(primary.Body, &view)
		s.obs("at27", "primary_publication", string(view.PublicationIntent))
		s.obs("at27", "primary_desired", nullableID(view.Releases.DesiredReleaseId))
		s.obs("at27", "primary_current", nullableID(view.Releases.CurrentReleaseId))
		s.obs("at27", "primary_last_good", nullableID(view.Releases.LastGoodReleaseId))
	}
	fleet, err := s.lab.fleet(s.ctx, admin)
	if err == nil {
		s.obs("at27", "serving_ids", fmt.Sprintf("%d", len(fleet.ServingReleaseIds)))
		s.obs("at27", "fleet_generation", fleet.Generation.String())
	}
	s.obs("at27", "forward_chain", s.lab.forwardSummary(s.lab.node))
	s.obs("at27", "traffic", "not claimed; the lab node has no member accept rules")
	s.obs("at27", "desired_ne_current", "service was already not forwarding; this run does not show a serving node staying up")
	s.obs("at27", "last_good_restore", "blocked: no last_good release was confirmed because health checks fail closed")
	s.rec["at27"].Actual = "disabled intent " + empty(beforeIntent) + " -> " + empty(afterIntent) + "; below-minimum batch " + httpDetail(resp, err)
	s.rec["at27"].Status = StatusBlocked
}

func nullableID(v interface{ IsNull() bool }) string {
	type getter interface {
		IsNull() bool
		Get() (openapi_types.UUID, error)
	}
	g, ok := v.(getter)
	if !ok || g.IsNull() {
		return "-"
	}
	id, err := g.Get()
	if err != nil {
		return "-"
	}
	return uuid.UUID(id).String()
}

func (s *suite) arm64() {
	s.logf("arm64")
	s.cmd("at23-arm64", "GOOS=linux GOARCH=arm64 go build ./node/updater/cmd/fvpn-updater")
	s.cmd("at23-arm64", "see docs/e2e/at/updates/operator-run.md")
	outDir := s.lab.root + "/.agent/tmp"
	_ = os.MkdirAll(outDir, 0o755)
	bin := outDir + "/fvpn-updater-arm64"
	cmd := exec.Command("go", "build", "-trimpath", "-o", bin, "./node/updater/cmd/fvpn-updater")
	cmd.Dir = s.lab.root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.obs("at23-arm64", "cross_build", clip(string(out), 200)+" "+err.Error())
		s.rec["at23-arm64"].Actual = "cross build failed"
		return
	}
	b, err := os.ReadFile(bin)
	if err == nil {
		sum := sha256.Sum256(b)
		s.obs("at23-arm64", "updater_arm64_sha256", hex.EncodeToString(sum[:]))
	}
	probe := exec.Command("docker", "run", "--rm", "--name", s.lab.prefix+"-armprobe", "--platform", "linux/arm64",
		"-v", bin+":/fvpn-updater:ro", "ubuntu:24.04", "/fvpn-updater", "version")
	pctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	defer cancel()
	probe = exec.CommandContext(pctx, probe.Args[0], probe.Args[1:]...)
	// CommandContext drops Env; the docker CLI does not need the Go env.
	probeOut, probeErr := probe.CombinedOutput()
	s.obs("at23-arm64", "emulation", clip(string(probeOut), 200)+" "+fmt.Sprint(probeErr))
	if probeErr != nil {
		s.obs("at23-arm64", "hardware", "needs operator run; this host did not execute the arm64 binary")
	} else {
		s.obs("at23-arm64", "hardware", "the arm64 binary printed its version under emulation; a full node update still needs operator hardware")
	}
	s.obs("at23-arm64", "sources", "preinstalled box, cloud Ubuntu and coordinated third-party Ubuntu are operator-run.md; the lab container is one Ubuntu 24.04 amd64 source")
	s.rec["at23-arm64"].Actual = "arm64 updater cross-built; node image in this lab is amd64 only"
	s.rec["at23-arm64"].Status = StatusNeedsOperatorRun
}
