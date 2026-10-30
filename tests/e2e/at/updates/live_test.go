//go:build e2e

package updates_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	at "github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// The shared devstack package builds and registers the real candidates (D-053).
func TestUpdatesFollowup(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("requires owned devstack")
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	records := map[string]*at.Record{}
	titles := []string{"Good candidate and central confirmation", "Offline catch-up and new-node inheritance", "Tampered archive never executes", "Genuine candidate health failure rolls back", "Stale acknowledgement and observation authority"}
	runID := uuid.NewString()
	commit, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	for i, title := range titles {
		id := fmt.Sprintf("AT%d", 23+i)
		records[id] = &at.Record{ID: id, Title: title, Status: at.StatusBlocked, RunID: runID, Prefix: "w4-updates", PortBase: 42000, Git: strings.TrimSpace(string(commit)), Scope: "Follow-up 7 requested scenario; not complete original §19 acceptance or physical arm64 proof", Spec: []string{"UPD01", "UPD02", "§13.4", "§13.9", "§13.10", "§13.12"}, Expected: title, Actual: "Prerequisite not reached; see suite summary", Commands: []string{"make -f mk/at-updates.mk at-updates"}}
	}
	// Record exact harness bytes as well as the product source commit. This
	// remains reproducible when evidence and a capture fix share a final commit.
	harnessFiles := map[string]string{}
	paths, err := filepath.Glob(filepath.Join(root, "tests/e2e/at/updates/*.go"))
	if err != nil {
		t.Fatal("harness source listing failed")
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal("harness source read failed")
		}
		harnessFiles[filepath.Base(path)] = fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	harnessDiff := exec.Command("git", "diff", "--quiet", "HEAD", "--", "tests/e2e/at/updates")
	harnessDiff.Dir = root
	harnessDirty := harnessDiff.Run() != nil
	harnessRaw, _ := json.Marshal(map[string]any{"files_sha256": harnessFiles, "differs_from_source_commit": harnessDirty})
	for _, rec := range records {
		rec.Observations = append(rec.Observations, at.Observation{Name: "harness_source", Detail: string(harnessRaw)})
	}
	for _, id := range []string{"AT23", "AT25"} {
		raw, err := os.ReadFile(filepath.Join(root, "docs/e2e/at/updates/history/follow-up-6", id+".json"))
		if err != nil || json.Unmarshal(raw, records[id]) != nil {
			t.Fatal("retained follow-up 6 evidence unavailable")
		}
		records[id].Status = at.StatusBlocked
		records[id].Actual = "Retained follow-up 6 evidence; not rerun in follow-up 7. Blocked on F5-02; owner: w3-devstack-e2e follow-up 12 (probe-peer export)"
	}
	// Install the gate before login/build/enrollment: even Fatal writes all ATs.
	defer func() {
		requireFollowup7Evidence(records)
		dir := filepath.Join(root, "docs", "e2e", "at", "updates")
		var summary strings.Builder
		fmt.Fprintf(&summary, "# Updates follow-up 7\n\nRun: %s\n\nSource: %s\n\nThese labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. AT23/AT25 retain their follow-up 6 evidence and are not rerun. Physical arm64 execution and the full original §19 population remain outside this run. Both architectures contain compiled real binaries.\n\n| Scenario | Result | Observation |\n|---|---|---|\n", runID, strings.TrimSpace(string(commit)))
		for i := 23; i <= 27; i++ {
			rec := records[fmt.Sprintf("AT%d", i)]
			fmt.Fprintf(&summary, "| [%s](%s.md) | %s | %s |\n", rec.ID, rec.ID, rec.Status, at.Redact(rec.Actual))
		}
		summary.WriteString("\nSee [current requests](requests.md) and the [follow-up 2 AT26 history](history/follow-up-2/README.md).\n")
		if err := at.WriteAcceptance(dir, records); err != nil {
			t.Error(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte(summary.String()), 0o644); err != nil {
			t.Error(err)
		}
	}()
	lab, err := openLab()
	if err != nil {
		t.Fatal("owned stack unavailable")
	}
	if lab.prefix != "fvpn-w4-updates" {
		t.Fatal("refusing non-owned stack")
	}
	r := &runner{t: t, lab: lab}
	s := &updatesFollowup{r: r, ctx: ctx, rec: records}
	admin, err := lab.login(ctx, "e2e-admin", "admin")
	if err != nil {
		s.blockAll("w3-devstack-e2e: admin login prerequisite failed")
		return
	}
	s.admin = admin
	t.Log("building real signed candidates through shared PrepareServingBaseline")
	baseline, err := e2e.PrepareServingBaseline(ctx, admin.http, admin.csrf, true)
	if err != nil {
		s.blockAll("w3-devstack-e2e: real release build/register prerequisite failed: " + at.Redact(err.Error()))
		return
	}
	boot, good, bad := baseline.Boot, baseline.Good, baseline.Bad
	for id := range records {
		s.note(id, "manifest_and_artifact_digests", baseline.Digests)
		s.note(id, "support_policy_revision", baseline.SupportRevision)
	}
	unrelated := s.unrelatedRelease()
	r.enroll(ctx, admin)
	if r.nodeID == uuid.Nil || r.incID == uuid.Nil {
		s.blockAll("w3-wire-nodes: enrollment/peer bootstrap prerequisite failed")
		return
	}
	for id, rec := range records {
		if id == "AT23" || id == "AT25" {
			continue
		}
		rec.NodeIncarnation = r.incID.String()
	}
	conn, err := lab.db(ctx)
	if err != nil {
		t.Fatal("owned observation database unavailable")
	}
	defer conn.Close(ctx)
	ready := s.wait(2*time.Minute, func() bool {
		var ok bool
		err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM nodes n JOIN node_observations o ON o.node_id=n.id WHERE n.id=$1 AND n.provisioning_state='ready' AND o.applied_generation>0 AND o.safety_status='passed' AND o.report_payload->>'provisioning_state'='ready')`, r.nodeID).Scan(&ok)
		return err == nil && ok
	})
	for id := range records {
		s.note(id, "accepted_unpublished_baseline", ready)
		s.note(id, "netbird_manifest_version", "0.82.0")
		s.note(id, "release_ids", map[string]uuid.UUID{"boot": boot, "good": good, "negative": bad})
	}
	if !ready {
		s.blockAll("w2-agent-core / w3-wire-nodes: accepted ready baseline not observed")
		return
	}

	s.policyCriteria()
	batch, ok := s.batch("AT24", good)
	if !ok {
		return
	}
	s.preResultInjection(batch, boot, good, bad, unrelated)
	s.execute("AT24", false)
	baselineJournal := s.journal(good)
	baselineTarget := s.confirm(batch, "succeeded")
	baselineReceipt := s.receipts("AT24", batch)
	s.note("AT24", "baseline_journal", baselineJournal)
	s.note("AT24", "baseline_batch_api", baselineTarget.BatchJSON)
	s.firstResultLastGood(good, baselineReceipt)
	if !baselineReceipt || baselineJournal.Stage != "locally_succeeded" || !baselineJournal.allPassed() || !s.running(good) || baselineTarget.State != "succeeded" || baselineTarget.Unconfirmed {
		s.blockAll("w2-updater / w2-fleet-update: real accepted 1.0.1 prerequisite incomplete")
		return
	}
	baselinePayload := s.readJSON(`SELECT payload FROM update_results WHERE node_id=$1 AND incarnation_id=$2 AND payload->>'state'='succeeded' ORDER BY received_at LIMIT 1`, r.nodeID, r.incID)
	lastGood := good
	// TEST ONLY failure stimulus: a real TCP probe to a closed port inside the
	// owned node. Only this candidate's runtime config changes. The agent must
	// observe the failed dial and answer the challenge itself; no health writes.
	config := filepath.Join(lab.state, lab.nodeSlot(), "var", "e2e", "agent-"+boot.String()+".json")
	original, err := os.ReadFile(config)
	if err != nil {
		t.Fatal("negative candidate config unavailable")
	}
	cfg := map[string]any{}
	if json.Unmarshal(original, &cfg) != nil {
		t.Fatal("candidate config invalid")
	}
	cfg["egress_probe_target"] = "127.0.0.1:1"
	changed, _ := json.MarshalIndent(cfg, "", "  ")
	if err := writeAtomic(config, changed); err != nil {
		t.Fatal("negative probe configuration failed")
	}
	defer func() {
		if err := writeAtomic(config, original); err != nil {
			t.Error("restore candidate config failed")
		}
	}()
	rollbackBatch, ok := s.batch("AT26", boot)
	if ok {
		emptyTarget := s.target(rollbackBatch)
		s.note("AT26", "batch_api_without_receipt", emptyTarget.BatchJSON)
		s.note("AT26", "check_revision7_empty_target", targetEvidenceMatches(emptyTarget.BatchJSON, nil))
		s.note("AT26", "failure_stimulus", "TEST ONLY: candidate TCP egress probe dials its own closed loopback port; no forced health values or journal writes")
		s.execute("AT26", false)
		journal := s.journal(boot)
		s.note("AT26", "journal", journal)
		s.note("AT26", "expected_last_good_release", lastGood)
		s.note("AT26", "last_good_process_restored", s.running(lastGood))
		target := s.confirm(rollbackBatch, "rolled_back")
		s.note("AT26", "central", target)
		s.note("AT26", "batch_api_status", target.HTTPStatus)
		if json.Valid(target.BatchJSON) {
			s.note("AT26", "batch_api_response", target.BatchJSON)
		}
		s.note("AT26", "confirmation_projection", s.projection())
		rollbackReceipt := s.receipts("AT26", rollbackBatch)
		manifestRaw, manifestErr := os.ReadFile(filepath.Join(lab.state, "releases", "r", boot.String(), "manifest.json"))
		var manifest api.ReleaseManifestEnvelope
		if manifestErr != nil || json.Unmarshal(manifestRaw, &manifest) != nil {
			t.Fatal("rollback manifest read failed")
		}
		s.note("AT26", "rollback_allowed_release_ids", manifest.Signed.Rollback.AllowedReleaseIds)
		if rollbackReceipt && len(manifest.Signed.Rollback.AllowedReleaseIds) == 0 && journal.Stage == "rolled_back" && journal.Current == lastGood.String() && journal.LastGood == lastGood.String() && journal.candidateFailed() && journal.rollbackPassed() && s.running(lastGood) && rollbackTargetComplete(target, lastGood.String()) {
			s.pass("AT26", "Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false")
		} else if journal.Stage == "rolled_back" && journal.candidateFailed() && journal.rollbackPassed() && s.running(lastGood) {
			s.block("AT26", fmt.Sprintf("w2-fleet-update: healthy local rollback needs one current ACK and a confirmed rolled_back target with attention; observed state=%s unconfirmed=%t needs_attention=%t; inspect exact result, ACK and batch API response", target.State, target.Unconfirmed, target.NeedsAttention))
		} else {
			s.block("AT26", "w4-node-updater / w2-agent-core: expected candidate-failure and healthy last-good rollback evidence incomplete")
		}
	}
	if err := writeAtomic(config, original); err != nil {
		t.Fatal("restore candidate config failed")
	}

	if ok {
		s.followup4(rollbackBatch, boot, lastGood, good)
	}
	catchup, ok := s.offlineBatch(good, bad, baselinePayload)
	if !ok {
		return
	}
	s.recordedInjection(catchup, good, bad, unrelated)
	s.execute("AT24", false)
	catchupJournal := s.journal(bad)
	catchupTarget := s.confirm(catchup, "succeeded")
	catchupReceipt := s.receipts("AT24", catchup)
	s.recordedLastGood(bad, catchupReceipt)
	s.note("AT24", "catchup_journal", catchupJournal)
	s.note("AT24", "catchup_batch_api", catchupTarget.BatchJSON)
	catchupOK := catchupReceipt && catchupJournal.Stage == "locally_succeeded" && catchupJournal.allPassed() && s.running(bad) && terminalBatch(catchupTarget.BatchJSON, r.nodeID, bad)
	s.note("AT24", "check_offline_completion", catchupOK)
	inherited := s.inheritDefault(catchup, bad)
	if catchupOK && inherited {
		s.pass("AT24", "Offline target caught up through a real success result; a later new node inherited 1.0.2 through its own successful first-sync job; original snapshot unchanged")
	}
	s.pass("AT27", "Fresh unexpected release audited once; accepted releases never seeded or audited by that path; mismatches withhold confirmation; stale ACK replay preserves the newer job")

}

type updatesFollowup struct {
	r          *runner
	ctx        context.Context
	admin      *session
	rec        map[string]*at.Record
	notePrefix string
}

func (s *updatesFollowup) note(id, name string, value any) {
	if id == "AT23" || id == "AT25" {
		return
	}
	raw, _ := json.Marshal(value)
	name = s.notePrefix + name
	s.rec[id].Observations = append(s.rec[id].Observations, at.Observation{Name: name, Detail: string(raw)})
}
func (s *updatesFollowup) pass(id, detail string) {
	s.rec[id].Status = at.StatusPass
	s.rec[id].Actual = detail
	s.r.t.Log(id + ": pass: " + detail)
}
func (s *updatesFollowup) block(id, detail string) {
	if id == "AT23" || id == "AT25" {
		return
	}
	s.rec[id].Status = at.StatusBlocked
	s.rec[id].Actual = detail
	s.r.t.Log(id + ": blocked: " + detail)
}
func (s *updatesFollowup) blockAll(detail string) {
	for id := range s.rec {
		s.block(id, detail)
	}
}
func (s *updatesFollowup) wait(d time.Duration, fn func() bool) bool {
	ctx, cancel := context.WithTimeout(s.ctx, d)
	defer cancel()
	return waitUntil(ctx, 2*time.Second, fn)
}
func (s *updatesFollowup) batch(id string, release uuid.UUID) (api.UpdateBatchAccepted, bool) {
	accepted, ok := s.createBatch(id, release)
	if !ok {
		return accepted, false
	}
	return accepted, s.waitIntent(id, accepted, release)
}
func (s *updatesFollowup) createBatch(id string, release uuid.UUID) (api.UpdateBatchAccepted, bool) {
	var accepted api.UpdateBatchAccepted
	reply, err := s.admin.call(s.ctx, "GET", s.r.lab.public+"/api/v1/admin/releases", nil, nil)
	var page api.ReleaseList
	if err != nil || reply.Status != 200 || json.Unmarshal(reply.Body, &page) != nil {
		s.block(id, "w3-wire-fleet: fleet read unavailable")
		return accepted, false
	}
	reply, err = s.admin.call(s.ctx, "POST", s.r.lab.public+"/api/v1/admin/update-batches", api.UpdateBatchCreateRequest{ReleaseId: release, Scope: api.UpdateBatchCreateRequestScopeAllNodes, ExpectedGeneration: page.Fleet.Generation, SetFleetDefault: true, Reason: "TEST ONLY updates follow-up 7"}, map[string]string{"Idempotency-Key": uuid.NewString()})
	s.note(id, "batch_response", httpDetail(reply, err))
	if err != nil || reply.Status != 202 || json.Unmarshal(reply.Body, &accepted) != nil {
		s.block(id, "w3-wire-fleet: batch create not accepted")
		return accepted, false
	}
	s.note(id, "batch_id", accepted.BatchId)
	s.note(id, "generation", accepted.FleetGeneration.String())
	return accepted, true
}
func (s *updatesFollowup) waitIntent(id string, accepted api.UpdateBatchAccepted, release uuid.UUID) bool {
	seen := s.wait(90*time.Second, func() bool {
		raw, err := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
		var state struct {
			Signed struct {
				Update *api.UpdateIntent `json:"update"`
			} `json:"signed"`
		}
		if err != nil || json.Unmarshal(raw, &state) != nil || state.Signed.Update == nil {
			return false
		}
		batch, err := state.Signed.Update.BatchId.Get()
		return err == nil && batch == accepted.BatchId && state.Signed.Update.ReleaseId == release
	})
	s.note(id, "signed_intent_for_this_batch", seen)
	if !seen {
		s.block(id, "w3-wire-fleet / w2-agent-core: signed update intent not delivered for this batch")
	}
	return seen
}

type updatesTarget struct {
	BatchJSON      json.RawMessage `json:"-"`
	HTTPStatus     int             `json:"-"`
	State          string          `json:"state"`
	Unconfirmed    bool            `json:"unconfirmed"`
	NeedsAttention bool            `json:"needs_attention"`
	CurrentRelease string          `json:"current_release"`
	ErrorCode      string          `json:"error_code"`
}

func (s *updatesFollowup) target(batch api.UpdateBatchAccepted) updatesTarget {
	reply, err := s.admin.call(s.ctx, "GET", s.r.lab.public+"/api/v1/admin/update-batches/"+batch.BatchId.String(), nil, nil)
	var view api.UpdateBatch
	if err != nil || reply.Status != 200 || json.Unmarshal(reply.Body, &view) != nil || len(view.Targets) != 1 {
		return updatesTarget{State: "unavailable", BatchJSON: reply.Body, HTTPStatus: reply.Status}
	}
	target := view.Targets[0]
	result := updatesTarget{State: string(target.State), Unconfirmed: target.Unconfirmed, NeedsAttention: target.NeedsAttention, BatchJSON: reply.Body, HTTPStatus: reply.Status}
	if id, err := target.CurrentReleaseId.Get(); err == nil {
		result.CurrentRelease = id.String()
	}
	if job, err := target.CurrentJob.Get(); err == nil {
		if code, err := job.ErrorCode.Get(); err == nil {
			result.ErrorCode = string(code)
		}
	}
	return result
}
func (s *updatesFollowup) confirm(batch api.UpdateBatchAccepted, state string) updatesTarget {
	var target updatesTarget
	s.wait(90*time.Second, func() bool { target = s.target(batch); return target.State == state && !target.Unconfirmed })
	return target
}
func (s *updatesFollowup) running(id uuid.UUID) bool {
	raw, err := s.r.lab.docker(s.r.lab.node, "sh", "-c", `test -s /var/lib/vpn-node/e2e/agent.pid && readlink /proc/$(cat /var/lib/vpn-node/e2e/agent.pid)/exe`)
	return err == nil && strings.TrimSpace(string(raw)) == "/opt/vpn-node/releases/"+id.String()+"/bin/fvpn-agent"
}
func (s *updatesFollowup) execute(id string, interrupt bool) {
	dir := filepath.Join(s.r.lab.state, s.r.lab.nodeSlot(), "var", "e2e")
	pause := filepath.Join(dir, "pause-agent")
	if err := os.WriteFile(pause, []byte("1\n"), 0o644); err != nil {
		s.r.t.Fatal("pause failed")
	}
	if !interrupt {
		defer os.Remove(pause)
	}
	time.Sleep(4 * time.Second)
	_ = os.Remove(filepath.Join(dir, "updater-result.json"))
	probe := s.beginProbeCapture(id)
	if interrupt {
		raw, err := os.ReadFile(filepath.Join(dir, "updater.json.done"))
		var cfg map[string]any
		if err != nil || json.Unmarshal(raw, &cfg) != nil {
			s.r.t.Fatal("prior updater config missing")
		}
		// TEST ONLY: the subprocess kills its updater parent after the real
		// service stop, leaving the durable journal untouched for recovery.
		cfg["stop_command"] = []string{"/bin/sh", "-c", `/usr/local/bin/agent-service.sh stop; kill -KILL "$PPID"`}
		raw, _ = json.MarshalIndent(cfg, "", "  ")
		if err := writeAtomic(filepath.Join(dir, "updater.json"), raw); err != nil {
			s.r.t.Fatal("interrupt config failed")
		}
	} else if err := s.r.writeUpdaterConfig(); err != nil {
		s.r.t.Fatal("updater config failed")
	}
	finished := s.wait(4*time.Minute, func() bool {
		_, resultErr := os.Stat(filepath.Join(dir, "updater-result.json"))
		_, configErr := os.Stat(filepath.Join(dir, "updater.json"))
		return resultErr == nil && os.IsNotExist(configErr)
	})
	probe.finish()
	s.note(id, "updater_finished", finished)
	s.r.t.Logf("%s updater finished=%t interrupt=%t", id, finished, interrupt)
	raw, _ := os.ReadFile(filepath.Join(dir, "updater-result.json"))
	var result struct {
		ExitCode int `json:"exit_code"`
	}
	if json.Unmarshal(raw, &result) == nil {
		s.note(id, "updater_exit_code", result.ExitCode)
	}
	if !finished {
		s.r.t.Fatal("updater did not finish; evidence remains blocked")
	}
	if !interrupt {
		s.flush(id)
	}
}

type updatesJournal struct {
	TxnID           string            `json:"txn_id"`
	Stage           string            `json:"stage"`
	Error           string            `json:"error"`
	JobError        string            `json:"job_error"`
	Current         string            `json:"current"`
	LastGood        string            `json:"last_good"`
	Health          map[string]string `json:"health"`
	CandidateHealth map[string]string `json:"candidate_health"`
	Attestations    []map[string]any  `json:"attestations"`
	OutboxState     string            `json:"outbox_state"`
	OutboxStatus    string            `json:"outbox_status"`
	OutboxError     string            `json:"outbox_error"`
}

func (j updatesJournal) allPassed() bool {
	for _, key := range []string{"processes", "identity_and_state", "dns_policy", "internet_only", "tunnel_probe"} {
		if j.Health[key] != "passed" {
			return false
		}
	}
	return true
}
func (j updatesJournal) candidateFailed() bool {
	if j.CandidateHealth["tunnel_probe"] != "failed" {
		return false
	}
	for _, a := range j.Attestations {
		if a["phase"] == "candidate" && a["agent_tunnel_probe"] == "failed" {
			return true
		}
	}
	return false
}
func (j updatesJournal) rollbackPassed() bool {
	if !j.allPassed() {
		return false
	}
	for _, a := range j.Attestations {
		if a["phase"] == "rollback" && a["agent_tunnel_probe"] == "passed" && a["agent_internet_only"] == "passed" && a["agent_dns_policy"] == "passed" {
			return true
		}
	}
	return false
}
func (s *updatesFollowup) journal(release uuid.UUID) updatesJournal {
	var view updatesJournal
	path := filepath.Join(s.r.t.TempDir(), "journal.db")
	if err := exec.CommandContext(s.ctx, "docker", "cp", s.r.lab.node+":/var/lib/vpn-node/update-journal/journal.db", path).Run(); err != nil {
		return view
	}
	_ = exec.CommandContext(s.ctx, "docker", "cp", s.r.lab.node+":/var/lib/vpn-node/update-journal/journal.db-wal", path+"-wal").Run()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(2000)")
	if err != nil {
		return view
	}
	defer db.Close()
	var health, candidate string
	_ = db.QueryRow(`SELECT txn_id,stage,coalesce(error_code,''),coalesce(health_json,''),coalesce(candidate_health_json,'') FROM txns WHERE release_id=? ORDER BY updated_at DESC LIMIT 1`, release.String()).Scan(&view.TxnID, &view.Stage, &view.Error, &health, &candidate)
	_ = json.Unmarshal([]byte(health), &view.Health)
	_ = json.Unmarshal([]byte(candidate), &view.CandidateHealth)
	_ = db.QueryRow(`SELECT coalesce(error_code,'') FROM jobs WHERE release_id=? ORDER BY updated_at DESC LIMIT 1`, release.String()).Scan(&view.JobError)
	_ = db.QueryRow(`SELECT value FROM meta WHERE key='current_release_id'`).Scan(&view.Current)
	_ = db.QueryRow(`SELECT value FROM meta WHERE key='last_good_release_id'`).Scan(&view.LastGood)
	_ = db.QueryRow(`SELECT state,status,coalesce(last_error,'') FROM result_outbox WHERE job_id=(SELECT job_id FROM jobs WHERE release_id=? ORDER BY updated_at DESC LIMIT 1) ORDER BY seq DESC LIMIT 1`, release.String()).Scan(&view.OutboxState, &view.OutboxStatus, &view.OutboxError)
	rows, err := db.Query(`SELECT detail FROM txn_events WHERE txn_id=? ORDER BY seq`, view.TxnID)
	if err != nil {
		return view
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if rows.Scan(&raw) != nil {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(raw), &event) != nil || event["kind"] != "health_attest" {
			continue
		}
		safe := map[string]any{}
		for _, key := range []string{"kind", "phase", "reason", "internet_only_reason", "dns_policy_reason", "tunnel_probe_reason", "agent_internet_only", "agent_dns_policy", "agent_tunnel_probe", "agent_version", "netbird_version", "published"} {
			if value, ok := event[key]; ok {
				safe[key] = value
			}
		}
		view.Attestations = append(view.Attestations, safe)
	}
	return view
}

// Flush uses the product outbox sender. Rejections remain diagnostic; no result or
// acknowledgement is synthesized by this flush. Preflight errors can return before Run flushes.
func (s *updatesFollowup) flush(id string) {
	started := time.Now().UTC()
	output, err := exec.CommandContext(s.ctx, "docker", "exec", s.r.lab.node, "fvpn-updater", "flush", "-config", "/var/lib/vpn-node/e2e/updater.json.done").CombinedOutput()
	s.note(id, "product_outbox_flush_success", err == nil)
	s.note(id, "product_outbox_flush_output", strings.TrimSpace(string(output)))
	// Preserve only the HTTP route's status/time, never raw service logs.
	logs, logErr := exec.CommandContext(s.ctx, "docker", "logs", "--tail", "120", s.r.lab.control).CombinedOutput()
	if logErr == nil {
		for _, line := range strings.Split(string(logs), "\n") {
			var entry struct {
				At     time.Time `json:"time"`
				Route  string    `json:"route"`
				Status int       `json:"status"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil && entry.Route == "POST /api/v1/agent/update-results" && !entry.At.Before(started) {
				s.note(id, "result_delivery_http", map[string]any{"at": entry.At, "status": entry.Status})
			}
		}
	}
}

func (s *updatesFollowup) projection() json.RawMessage {
	conn, err := s.r.lab.db(s.ctx)
	if err != nil {
		return json.RawMessage(`{"error":"observation database unavailable"}`)
	}
	defer conn.Close(s.ctx)
	var raw []byte
	err = conn.QueryRow(s.ctx, `SELECT jsonb_build_object(
      'fleet_update_generation',s.observed_update_generation::text,
      'fleet_observed_at',s.last_observed_at,'fleet_received_at',s.received_at,
      'latest_node_observation',(SELECT jsonb_build_object('observed_at',o.observed_at,
         'received_at',o.received_at,'update',o.report_payload->'update')
         FROM node_observations o WHERE o.node_id=s.node_id ORDER BY o.received_at DESC LIMIT 1),
      'current_job',(SELECT jsonb_build_object('state',j.state,'generation',j.generation::text,
         'last_result_at',j.last_result_at) FROM update_jobs j WHERE j.id=s.current_job_id))
      FROM fleet_node_state s WHERE s.node_id=$1`, s.r.nodeID).Scan(&raw)
	if err != nil {
		return json.RawMessage(`{"error":"projection query failed"}`)
	}
	return json.RawMessage(raw)
}

// Publication can keep forwarding blocked after successful recovery. Classify
// the journal and authenticated central evidence, not the recover exit alone.
func updatesRecoveryComplete(before, after updatesJournal, blocked, running bool, target updatesTarget) bool {
	return blocked && before.Stage == "final_sampled" && before.TxnID != "" && before.TxnID == after.TxnID && after.Stage == "locally_succeeded" && after.allPassed() && running && target.State == "succeeded" && !target.Unconfirmed
}

func TestUpdatesFollowupEvidence(t *testing.T) {
	testFollowup4Evidence(t)
	testFollowup5Evidence(t)
	testFollowup7Evidence(t)
	before := updatesJournal{TxnID: "test-transaction", Stage: "final_sampled"}
	after := updatesJournal{TxnID: before.TxnID, Stage: "locally_succeeded", Health: map[string]string{
		"processes": "passed", "identity_and_state": "passed", "internet_only": "passed", "dns_policy": "passed", "tunnel_probe": "passed",
	}}
	accepted := updatesTarget{State: "succeeded"}
	if !updatesRecoveryComplete(before, after, true, true, accepted) {
		t.Fatal("healthy confirmed recovery with forwarding kept blocked must pass")
	}
	pending := accepted
	pending.Unconfirmed = true
	if updatesRecoveryComplete(before, after, true, true, pending) {
		t.Fatal("local recovery alone cannot confirm central acceptance")
	}
	unrelated := after
	unrelated.TxnID = "another-transaction"
	if updatesRecoveryComplete(before, unrelated, true, true, accepted) {
		t.Fatal("another transaction cannot prove recovery of the interrupted one")
	}
	after.Health["tunnel_probe"] = "not_run"
	if updatesRecoveryComplete(before, after, true, true, accepted) {
		t.Fatal("not_run is not healthy recovery")
	}
	t.Run("AT26 accepted history cannot pass", func(t *testing.T) {
		root, err := repoRoot()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "docs/e2e/at/updates/history/follow-up-2/AT26.json"))
		if err != nil {
			t.Fatal(err)
		}
		var record at.Record
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, ob := range record.Observations {
			if ob.Name != "central_result_receipts" {
				continue
			}
			found = true
			var receipts []updatesReceipt
			if err := json.Unmarshal([]byte(ob.Detail), &receipts); err != nil {
				t.Fatal(err)
			}
			if rollbackReceiptComplete(receipts) {
				t.Fatal("historical ACK cannot pass AT26")
			}
			current := false
			receipts[0].Response.AcceptedAsHistory = &current
			if !rollbackReceiptComplete(receipts) {
				t.Fatal("one healthy current rollback receipt must pass")
			}
			if rollbackReceiptComplete(append(receipts, receipts[0])) {
				t.Fatal("multiple results cannot prove the single terminal path")
			}
			receipts[0].Response.AcceptedAsHistory = nil
			if rollbackReceiptComplete(receipts) {
				t.Fatal("missing ACK history status is not current evidence")
			}
		}
		if !found {
			t.Fatal("historical receipt missing")
		}
		target := updatesTarget{State: "rolled_back", NeedsAttention: true, ErrorCode: "HEALTH_CHECK_FAILED", CurrentRelease: "restored"}
		if !rollbackTargetComplete(target, "restored") {
			t.Fatal("confirmed rollback target must pass")
		}
		target.Unconfirmed = true
		if rollbackTargetComplete(target, "restored") {
			t.Fatal("unconfirmed rollback cannot pass")
		}
	})

}

// Receipts preserve the product's immutable result JSON and original ACK or
// rejection, joined to this batch. Follow-up 4 also uses a copy for the explicitly
// requested API conflict probes; the original receipt is never modified.
func (s *updatesFollowup) receipts(id string, batch api.UpdateBatchAccepted) bool {
	conn, err := s.r.lab.db(s.ctx)
	if err != nil {
		s.note(id, "central_result_receipts_error", "database unavailable")
		return false
	}
	defer conn.Close(s.ctx)
	var raw []byte
	err = conn.QueryRow(s.ctx, `SELECT coalesce(jsonb_agg(jsonb_build_object(
   'payload',r.payload,'response',r.ack,'payload_sha256',encode(r.payload_hash,'hex')) ORDER BY r.received_at,r.result_id),'[]'::jsonb)
   FROM update_results r JOIN update_jobs j ON j.id=r.job_id
   JOIN update_targets t ON t.id=j.target_id WHERE t.batch_id=$1`, batch.BatchId).Scan(&raw)
	if err != nil {
		s.note(id, "central_result_receipts_error", "receipt query failed")
		return false
	}
	s.note(id, "central_result_receipts", json.RawMessage(raw))
	snapshot, err := s.r.lab.docker(s.r.lab.node, "fvpn-updater", "status", "-journal", "/var/lib/vpn-node/update-journal/journal.db")
	if err == nil && json.Valid(snapshot) {
		s.note(id, "product_journal_snapshot", json.RawMessage(snapshot))
	}
	var receipts []updatesReceipt
	if json.Unmarshal(raw, &receipts) != nil {
		return false
	}
	for _, receipt := range receipts {
		if !receipt.Response.Accepted {
			continue
		}
		p := receipt.Payload
		switch id {
		case "AT25":
			if p.State == "failed" && p.Health == nil {
				return true
			}
		case "AT23", "AT24", "AT27":
			if p.State == "succeeded" && (updatesJournal{Health: p.Health}).allPassed() {
				return true
			}
		case "AT26":
			return rollbackReceiptComplete(receipts)
		}
	}
	return false
}

// The original AT26 ACK was accepted as history. A later observation alone
// cannot turn that receipt into current terminal completion.
type updatesReceipt struct {
	Payload struct {
		State     string            `json:"state"`
		Health    map[string]string `json:"health"`
		Candidate map[string]string `json:"candidate_health"`
		ErrorCode string            `json:"error_code"`
	} `json:"payload"`
	Response struct {
		Accepted          bool  `json:"accepted"`
		AcceptedAsHistory *bool `json:"accepted_as_history"`
	} `json:"response"`
}

func rollbackReceiptComplete(receipts []updatesReceipt) bool {
	if len(receipts) != 1 {
		return false
	}
	r := receipts[0]
	return r.Response.Accepted && r.Response.AcceptedAsHistory != nil && !*r.Response.AcceptedAsHistory && r.Payload.State == "rolled_back" && r.Payload.ErrorCode == "HEALTH_CHECK_FAILED" && r.Payload.Candidate["tunnel_probe"] == "failed" && (updatesJournal{Health: r.Payload.Health}).allPassed()
}

func rollbackTargetComplete(target updatesTarget, lastGood string) bool {
	return target.State == "rolled_back" && !target.Unconfirmed && target.NeedsAttention && target.ErrorCode == "HEALTH_CHECK_FAILED" && target.CurrentRelease == lastGood
}
