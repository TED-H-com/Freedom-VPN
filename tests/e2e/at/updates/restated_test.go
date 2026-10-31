//go:build e2e

package updates_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	at "github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates"
	"github.com/google/uuid"
)

func requireFollowup7Evidence(records map[string]*at.Record) {
	for id, names := range map[string][]string{
		"AT23": {"probe_inspection", "lab_probe_peer", "probe_traversal", "first_result_last_good"},
		"AT24": {"pre_result_versions", "null_last_good_seed", "first_result_last_good", "recorded_versions", "offline_snapshot", "offline_completion", "new_node_inheritance"},
		"AT25": {"probe_inspection", "lab_probe_peer", "failed_requires_cause"},
		"AT26": {"revision7_empty_target", "revision7_rollback_target", "conflict_bounds"},
		"AT27": {"pre_result_audits", "recorded_audits", "recorded_versions", "withheld_observation", "recorded_last_good", "stale_ack", "observation_authority"},
	} {
		for _, name := range names {
			passed := false
			for _, ob := range records[id].Observations {
				if ob.Name == "check_"+name {
					passed = ob.Detail == "true"
				}
			}
			if !passed {
				records[id].Status = at.StatusBlocked
				records[id].Actual += "; required follow-up 8 check incomplete: " + name
			}
		}
	}
}

func (s *updatesFollowup) policyCriteria() {
	path := "docs/updates/central-lifecycle.md"
	raw, err := os.ReadFile(filepath.Join(s.r.lab.root, path))
	if err != nil {
		s.r.t.Fatal("policy unavailable")
	}
	quotes := []map[string]any{}
	for _, quote := range []string{
		"Heartbeats after acceptance can confirm those\nvalues, but cannot change current/last-good or clear recovery.",
		"Once a result is accepted, seeding closes even for later jobs in that incarnation.",
		"A mismatch in\neither release keeps `unconfirmed=true` by withholding that job's observation\ntimestamp; a later agreeing report can confirm it.",
		"A generation-bearing heartbeat is attributed to the **node's current job only**\n(§13.2, §13.8; D-068, r13 M-1).",
		"Before that incarnation's first accepted result, pre-result heartbeat seeding\nfills **only a null `nodes.current_release_id`**",
		"On the current job, each distinct unexpected release/job pair\ngets one `update_observation.unexpected_release` audit",
		"A\nlater agreeing null last-good permits the issued target to fill the null\n`current_release_id`, without establishing `last_good_release_id`.",
	} {
		at := strings.Index(string(raw), quote)
		if at < 0 {
			s.r.t.Fatal("policy quote changed; review acceptance criteria")
		}
		quotes = append(quotes, map[string]any{"source": fmt.Sprintf("%s:%d", path, strings.Count(string(raw[:at]), "\n")+1), "quote": quote})
	}
	for _, id := range []string{"AT24", "AT27"} {
		s.note(id, "policy_quotes", quotes)
	}
	s.rec["AT24"].Expected = "Accepted 1.0.1 remains current before the later 1.0.2 result; offline batch targets complete consistently with real results; a new node inherits fleet default through its own job without changing the old snapshot"
	s.rec["AT27"].Expected = "After acceptance an unexpected heartbeat preserves versions/recovery, creates no seeding audit and withholds confirmation; before acceptance each unexpected job/release on the current job is audited exactly once; stale ACK replay never restores the older job or desired generation"
}

// The product's configured online window expires naturally while the real
// agent is stopped. Neither timestamps nor observation columns are rewritten.
func (s *updatesFollowup) offlineBatch(recorded, target uuid.UUID, oldResult json.RawMessage) (api.UpdateBatchAccepted, bool) {
	pause := filepath.Join(s.r.lab.state, s.r.lab.nodeSlot(), "var", "e2e", "pause-agent")
	if writeAtomic(pause, []byte("TEST ONLY offline catch-up\n")) != nil {
		s.r.t.Fatal("offline pause failed")
	}
	resume := func() {
		if _, err := s.r.lab.docker(s.r.lab.node, "agent-service.sh", "start", "/opt/vpn-node/releases/"+recorded.String()); err != nil {
			s.r.t.Error("offline resume failed")
		}
		_ = os.Remove(pause)
	}
	defer resume()
	time.Sleep(4 * time.Second)
	if _, err := s.r.lab.docker(s.r.lab.node, "agent-service.sh", "stop"); err != nil {
		s.r.t.Fatal("offline stop failed")
	}
	raw, err := os.ReadFile(filepath.Join(s.r.lab.state, "config", "nodes.json"))
	var cfg struct {
		OnlineSeconds int `json:"online_seconds"`
	}
	if err != nil || json.Unmarshal(raw, &cfg) != nil || cfg.OnlineSeconds <= 0 || cfg.OnlineSeconds > 300 {
		s.r.t.Fatal("online window unavailable")
	}
	s.note("AT24", "online_window_seconds", cfg.OnlineSeconds)
	s.r.t.Logf("AT24 waiting for the configured %ds online window to expire", cfg.OnlineSeconds)
	expired := s.wait(time.Duration(cfg.OnlineSeconds+30)*time.Second, func() bool {
		raw := s.readJSON(`SELECT to_jsonb(NOT EXISTS(SELECT 1 FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 AND received_at>clock_timestamp()-$3::int*interval '1 second') AND NOT EXISTS(SELECT 1 FROM fleet_node_state WHERE node_id=$1 AND received_at>clock_timestamp()-$3::int*interval '1 second'))`, s.r.nodeID, s.r.incID, cfg.OnlineSeconds)
		return string(raw) == "true"
	})
	if !expired {
		s.block("AT24", "offline observation window did not expire")
		return api.UpdateBatchAccepted{}, false
	}
	batch, ok := s.createBatch("AT24", target)
	if !ok {
		return batch, false
	}
	view := s.target(batch)
	var apiBatch api.UpdateBatch
	valid := json.Unmarshal(view.BatchJSON, &apiBatch) == nil && apiBatch.TargetCount == 1 && batch.OnlineAtSnapshot == 0 && len(apiBatch.Targets) == 1 && apiBatch.Targets[0].NodeId == s.r.nodeID && view.State == "waiting_online" && view.CurrentRelease == recorded.String()
	s.note("AT24", "offline_batch_accepted", batch)
	s.note("AT24", "offline_batch_api", view.BatchJSON)
	s.note("AT24", "check_offline_snapshot", valid)
	// Identical replay of a genuine old receipt, not a manufactured success.
	query := `SELECT jsonb_build_object('current_job_id',f.current_job_id,'desired_generation',n.update_generation::text,'observed_generation',f.observed_update_generation::text,'current_release_id',n.current_release_id,'last_good_release_id',n.last_good_release_id,'recovery_required',f.recovery_required) FROM nodes n JOIN fleet_node_state f ON f.node_id=n.id WHERE n.id=$1`
	before := s.readJSON(query, s.r.nodeID)
	count := s.acceptedResultCount()
	old := document(oldResult)
	originalACK := s.readJSON(`SELECT ack FROM update_results WHERE result_id=$1`, old.text("result_id"))
	response := s.nodePOST("/api/v1/agent/update-results", old)
	after := s.readJSON(query, s.r.nodeID)
	afterView := s.target(batch)
	staleOK := response.Status == 202 && sameJSON(response.Body, originalACK) && sameJSON(before, after) && s.acceptedResultCount() == count && afterView.State == "waiting_online"
	s.note("AT27", "stale_ack_replay", map[string]any{"request": old, "http_status": response.Status, "response": json.RawMessage(response.Body), "original_ack": originalACK, "before": before, "after": after, "new_batch_api": afterView.BatchJSON})
	s.note("AT27", "check_stale_ack", staleOK)
	resume()
	seen := s.waitIntent("AT24", batch, target)
	s.note("AT24", "online_issued_batch_api", s.target(batch).BatchJSON)
	return batch, seen
}

func terminalBatch(raw json.RawMessage, node, release uuid.UUID) bool {
	var batch api.UpdateBatch
	if json.Unmarshal(raw, &batch) != nil || batch.TargetCount != 1 || len(batch.Targets) != 1 || batch.State != "completed" {
		return false
	}
	target := batch.Targets[0]
	current, err := target.CurrentReleaseId.Get()
	return err == nil && target.NodeId == node && current == release && target.State == "succeeded" && !target.Unconfirmed && batch.Counts.Succeeded == 1
}

func (s *updatesFollowup) inheritDefault(original api.UpdateBatchAccepted, release uuid.UUID) bool {
	stack, err := e2e.OpenStack()
	if err != nil {
		s.block("AT24", "w3-devstack-e2e: shared stack unavailable")
		return false
	}
	defer stack.Close()
	if err = stack.StartSecondExit(s.ctx); err != nil {
		s.block("AT24", "w3-devstack-e2e: shared second node hook failed: "+at.Redact(err.Error()))
		return false
	}
	if err = s.r.lab.verifyKernelBoot("node2"); err != nil {
		s.block("AT24", "second node kernel boot identity adapter failed")
		return false
	}
	// Renew recent auth before a new enrollment after the genuine offline wait.
	admin, err := s.r.lab.login(s.ctx, "e2e-admin", "admin")
	if err != nil {
		s.block("AT24", "admin renewal failed")
		return false
	}
	s.admin = admin
	node, err := stack.EnrollAt(s.ctx, e2e.BrowserSession{Client: admin.http, CSRF: admin.csrf}, "TEST ONLY post-batch inherited node", "node2")
	if err != nil {
		s.block("AT24", "w3-devstack-e2e: second enrollment failed: "+at.Redact(err.Error()))
		return false
	}
	second := *s.r.lab
	second.node = second.prefix + "-node2"
	child := &updatesFollowup{r: &runner{t: s.r.t, lab: &second, nodeID: node.NodeID, incID: node.IncarnationID}, ctx: s.ctx, admin: admin, rec: s.rec, notePrefix: "new_node_"}
	var intent api.UpdateIntent
	seen := s.wait(2*time.Minute, func() bool {
		raw, err := second.docker(second.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
		var d struct {
			Signed struct {
				Update *api.UpdateIntent `json:"update"`
			} `json:"signed"`
		}
		if err != nil || json.Unmarshal(raw, &d) != nil || d.Signed.Update == nil {
			return false
		}
		intent = *d.Signed.Update
		return intent.ReleaseId == release && intent.NodeId == node.NodeID && intent.IncarnationId == node.IncarnationID
	})
	batchID, batchErr := intent.BatchId.Get()
	s.note("AT24", "new_node_inherited_intent", intent)
	if !seen || batchErr != nil || batchID == original.BatchId {
		s.block("AT24", "w2-fleet-update: new node did not inherit a separate fleet-default job")
		return false
	}
	before := child.target(api.UpdateBatchAccepted{BatchId: batchID})
	s.note("AT24", "new_node_before_result_batch_api", before.BatchJSON)
	// Only version inputs change; preserve node2's own generated datapath and
	// resolver config. Artifacts/signatures still use the shared release builder.
	raw, err := second.docker(second.node, "cat", "/etc/vpn-node/agent-run.json")
	cfg := document(raw)
	if err != nil || cfg == nil {
		s.block("AT24", "w3-devstack-e2e: second agent config absent")
		return false
	}
	cfg.set("release_id", release.String())
	cfg.set("agent_release", "1.0.2")
	raw, _ = json.Marshal(cfg)
	if writeAtomic(filepath.Join(second.state, "node2", "var", "e2e", "agent-"+release.String()+".json"), raw) != nil {
		s.r.t.Fatal("second candidate version config failed")
	}
	child.execute("AT24", false)
	journal := child.journal(release)
	target := child.confirm(api.UpdateBatchAccepted{BatchId: batchID}, "succeeded")
	receipt := child.receipts("AT24", api.UpdateBatchAccepted{BatchId: batchID})
	child.note("AT24", "journal", journal)
	child.note("AT24", "batch_api", target.BatchJSON)
	originalAfter := s.target(original)
	s.note("AT24", "original_batch_after_inheritance", originalAfter.BatchJSON)
	// Read only the provenance relationship; all creation came from enrollment
	// and the product's first-sync path, never direct harness inserts.
	provenance := s.readJSON(`SELECT jsonb_build_object('scope',b.scope,'first_sync_node_id',b.first_sync_node_id,'new_node_created_after_original_batch',n.created_at>o.created_at,'original_target_count',o.target_count,'new_node_in_original_snapshot',EXISTS(SELECT 1 FROM update_targets WHERE batch_id=o.id AND node_id=n.id),'fleet_default_release_id',f.desired_release_id) FROM update_batches b JOIN nodes n ON n.id=b.first_sync_node_id CROSS JOIN update_batches o CROSS JOIN fleet_config f WHERE b.id=$1 AND o.id=$2`, batchID, original.BatchId)
	s.note("AT24", "inheritance_provenance", provenance)
	p := document(provenance)
	ok := node.NodeID != s.r.nodeID && node.IncarnationID != s.r.incID && receipt && journal.Stage == "locally_succeeded" && journal.allPassed() && child.running(release) && terminalBatch(target.BatchJSON, node.NodeID, release) && terminalBatch(originalAfter.BatchJSON, s.r.nodeID, release) && p.text("scope") == "first_sync" && string(p["new_node_created_after_original_batch"]) == "true" && string(p["new_node_in_original_snapshot"]) == "false" && p.text("fleet_default_release_id") == release.String()
	s.note("AT24", "check_new_node_inheritance", ok)
	if !ok {
		s.block("AT24", "w3-devstack-e2e / w2-fleet-update: inspect new node job result and unchanged snapshot evidence")
	}
	return ok
}

func testFollowup7Evidence(t *testing.T) {
	t.Helper()
	records := map[string]*at.Record{}
	for _, id := range []string{"AT23", "AT24", "AT25", "AT26", "AT27"} {
		records[id] = &at.Record{Status: at.StatusPass}
	}
	requireFollowup7Evidence(records)
	for _, id := range []string{"AT23", "AT24", "AT25", "AT26", "AT27"} {
		if records[id].Status != at.StatusBlocked {
			t.Fatalf("%s bypassed required follow-up 8 evidence", id)
		}
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	s := &updatesFollowup{r: &runner{t: t, lab: &lab{root: root}}, rec: records}
	s.policyCriteria()
	raw := redactChallenge(json.RawMessage(`{"probe":{"peer_id":"secret-peer-id","source_v4":"100.64.0.9/32"}}`))
	if strings.Contains(string(raw), "secret-peer-id") || !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("peer_id was not redacted: %s", string(raw))
	}
}
