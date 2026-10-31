//go:build e2e

package updates_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	at "github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates"
	"github.com/google/uuid"
)

// Every extension is mandatory, including when an earlier prerequisite returns
// before its probe. An earlier successful base scenario cannot hide a missing check.
func requireFollowup4Evidence(records map[string]*at.Record) {
	for id, names := range map[string][]string{
		"AT25": {"failed_requires_cause"},
		"AT26": {"revision7_empty_target", "revision7_rollback_target", "conflict_bounds"},
		"AT27": {"observation_authority", "physical_rollback"},
	} {
		all := true
		for _, name := range names {
			passed := false
			for _, ob := range records[id].Observations {
				if ob.Name == "check_"+name && ob.Detail == "true" {
					passed = true
				}
			}
			if !passed {
				all = false
				records[id].Status = at.StatusBlocked
				records[id].Actual += "; w2-fleet-update: required " + name + " evidence absent or failed"
			}
		}
		if all {
			records[id].Actual += "; follow-up 4 verified: " + strings.Join(names, ", ")
		}
	}
}

type updateDocument map[string]json.RawMessage

func (d updateDocument) set(key string, value any) { d[key], _ = json.Marshal(value) }
func (d updateDocument) text(key string) string {
	var v string
	_ = json.Unmarshal(d[key], &v)
	return v
}
func document(raw []byte) updateDocument { var d updateDocument; _ = json.Unmarshal(raw, &d); return d }
func sameJSON(a, b json.RawMessage) bool {
	ca, ea := canonjson.Canonicalize(a)
	cb, eb := canonjson.Canonicalize(b)
	return ea == nil && eb == nil && string(ca) == string(cb)
}

// Check raw presence as well as value: generated nullable types alone cannot
// distinguish omitted revision 7 members from explicit null on the wire.
func targetEvidenceMatches(raw json.RawMessage, receipt updateDocument) bool {
	var batch struct {
		Targets []updateDocument `json:"targets"`
	}
	if json.Unmarshal(raw, &batch) != nil || len(batch.Targets) != 1 {
		return false
	}
	for _, key := range []string{"previous_release_id", "health", "candidate_health"} {
		actual, present := batch.Targets[0][key]
		expected := json.RawMessage("null")
		if receipt != nil {
			expected = receipt[key]
		}
		if !present || !sameJSON(actual, expected) {
			return false
		}
	}
	return true
}

// All database access in these probes is SELECT-only. Stimuli go through the
// product API, authenticated by the enrolled node; no projection or journal writes.
func (s *updatesFollowup) readJSON(query string, args ...any) json.RawMessage {
	conn, err := s.r.lab.db(s.ctx)
	if err != nil {
		s.r.t.Fatal("owned evidence database unavailable")
	}
	defer conn.Close(s.ctx)
	var raw []byte
	if err := conn.QueryRow(s.ctx, query, args...).Scan(&raw); err != nil {
		s.r.t.Fatal("read-only evidence query failed: " + at.Redact(err.Error()))
	}
	return raw
}

func (s *updatesFollowup) followup4(batch api.UpdateBatchAccepted, candidate, restored, next uuid.UUID) {
	receiptRaw := s.readJSON(`SELECT coalesce((SELECT r.payload FROM update_results r
 JOIN update_jobs j ON j.id=r.job_id JOIN update_targets t ON t.id=j.target_id
 WHERE t.batch_id=$1 AND r.ack->>'accepted'='true' AND r.ack->>'accepted_as_history'='false'
 AND r.payload->>'state'='rolled_back' ORDER BY r.received_at DESC LIMIT 1),'null'::jsonb)`, batch.BatchId)
	receipt := document(receiptRaw)
	if receipt == nil {
		return
	}
	target := s.target(batch)
	s.note("AT26", "revision7_batch_api_response", target.BatchJSON)
	s.note("AT26", "check_revision7_rollback_target", targetEvidenceMatches(target.BatchJSON, receipt) && receipt.text("previous_release_id") == candidate.String())
	s.observationAuthority(batch, receipt, candidate, restored)
	s.conflictBounds(receipt)
	if s.rec["AT25"].RunID == s.rec["AT26"].RunID {
		s.failedRequiresCause(receipt, next)
	} // Retained AT25 is not rerun in follow-up 7.
}

// Keep the identity key on the node. GNU wget reads it there; only the bounded
// HTTP response body/status comes back. TEST ONLY requests use an owned bind mount.
func (s *updatesFollowup) nodePOST(route string, body any) reply {
	s.r.t.Helper()
	// Respect the product's default heartbeat (1/s) and agent (5/s) buckets,
	// shared with the real process. Each expected response is asserted once;
	// a rate-limit failure is never retried into a passing conflict assertion.
	select {
	case <-time.After(2 * time.Second):
	case <-s.ctx.Done():
		s.r.t.Fatal("node API probe context expired")
	}
	if route != "/api/v1/agent/observations" && route != "/api/v1/agent/update-results" {
		s.r.t.Fatal("unexpected node API probe route")
	}
	name := "authority-" + uuid.NewString()
	host := filepath.Join(s.r.lab.state, s.r.lab.nodeSlot(), "var", "e2e", name)
	raw, err := json.Marshal(body)
	if err != nil || writeAtomic(host+".json", raw) != nil {
		s.r.t.Fatal("node API probe write failed")
	}
	defer os.Remove(host + ".json")
	defer os.Remove(host + ".response")
	remote := "/var/lib/vpn-node/e2e/" + name
	headers, _ := s.r.lab.docker(s.r.lab.node, "wget", "--server-response", "--content-on-error", "-q",
		"--ca-certificate=/etc/vpn-node/ca.pem", "--certificate=/var/lib/vpn-node/identity/cert.pem",
		"--private-key=/var/lib/vpn-node/identity/key.pem", "--header=Content-Type: application/json",
		"--post-file="+remote+".json", "-O", remote+".response", s.r.lab.agent+route)
	response, readErr := os.ReadFile(host + ".response")
	if readErr != nil {
		s.r.t.Fatal("node API response unavailable")
	}
	result := reply{Body: response, Code: errorCode(response)}
	for _, line := range strings.Split(string(headers), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "HTTP/") {
			result.Status, _ = strconv.Atoi(fields[1])
		}
	}
	if result.Status == 0 {
		s.r.t.Fatal("node API HTTP status unavailable")
	}
	return result
}

func (s *updatesFollowup) authoritySnapshot() updateDocument {
	return document(s.readJSON(`SELECT jsonb_build_object('current_release_id',n.current_release_id,
 'last_good_release_id',n.last_good_release_id,'generation',f.observed_update_generation::text,
 'recovery_required',f.recovery_required,'observation_incarnation_id',f.observation_incarnation_id,
 'received_at',f.received_at) FROM nodes n JOIN fleet_node_state f ON f.node_id=n.id WHERE n.id=$1`, s.r.nodeID))
}

func sameAuthority(a, b updateDocument) bool {
	for _, k := range []string{"current_release_id", "last_good_release_id", "generation", "recovery_required"} {
		if !sameJSON(a[k], b[k]) {
			return false
		}
	}
	return true
}

func (s *updatesFollowup) observationAuthority(batch api.UpdateBatchAccepted, receipt updateDocument, candidate, restored uuid.UUID) {
	pause := filepath.Join(s.r.lab.state, s.r.lab.nodeSlot(), "var", "e2e", "pause-agent")
	if writeAtomic(pause, []byte("TEST ONLY controlled API observation probes\n")) != nil {
		s.r.t.Fatal("pause request failed")
	}
	defer func() {
		_, err := s.r.lab.docker(s.r.lab.node, "/usr/local/bin/agent-service.sh", "start", "/opt/vpn-node/releases/"+restored.String())
		_ = os.Remove(pause)
		if err != nil {
			s.r.t.Error("restore real agent process failed")
		}
	}()
	select {
	case <-time.After(4 * time.Second):
	case <-s.ctx.Done():
		return
	}
	if _, err := s.r.lab.docker(s.r.lab.node, "/usr/local/bin/agent-service.sh", "stop"); err != nil {
		s.r.t.Fatal("stop agent for controlled observation probes failed")
	}
	// Copy the last real report, retaining all actual checks. Only the requested
	// update/current-release stimuli, report ID and observation time are changed.
	report := document(s.readJSON(`SELECT report_payload FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 ORDER BY received_at DESC LIMIT 1`, s.r.nodeID, s.r.incID))
	before := s.authoritySnapshot()
	s.note("AT27", "authority_before", before)
	send := func(name, current string, job any, generation string) (updatesTarget, updateDocument, bool) {
		report.set("report_id", uuid.NewString())
		report.set("observed_at", time.Now().UTC())
		report.set("current_release_id", current)
		report.set("last_good_release_id", restored)
		report.set("update", map[string]any{"job_id": job, "generation": generation, "stage": "report", "state": "rolled_back"})
		resp := s.nodePOST("/api/v1/agent/observations", report)
		target := s.target(batch)
		snapshot := s.authoritySnapshot()
		s.note("AT27", name, map[string]any{
			"request":     map[string]any{"report_id": report["report_id"], "observed_at": report["observed_at"], "current_release_id": report["current_release_id"], "last_good_release_id": report["last_good_release_id"], "update": report["update"]},
			"http_status": resp.Status, "response": json.RawMessage(resp.Body), "batch_api_response": target.BatchJSON, "projection": snapshot,
		})
		return target, snapshot, resp.Status == 202
	}
	disagree, afterDisagree, ok1 := send("disagreeing_heartbeat", candidate.String(), receipt.text("job_id"), receipt.text("generation"))
	disagreement := ok1 && disagree.State == "rolled_back" && disagree.CurrentRelease == restored.String() && disagree.Unconfirmed && sameAuthority(before, afterDisagree)
	agree, afterAgree, ok2 := send("agreeing_heartbeat", restored.String(), receipt.text("job_id"), receipt.text("generation"))
	agreement := ok2 && rollbackTargetComplete(agree, restored.String()) && sameAuthority(before, afterAgree)
	generation, err := strconv.ParseUint(receipt.text("generation"), 10, 64)
	if err != nil || generation > ^uint64(0)-100 {
		s.r.t.Fatal("invalid probe generation")
	}
	_, jobless, ok3 := send("jobless_generation_report", candidate.String(), nil, strconv.FormatUint(generation+100, 10))
	arrivalOnly := ok3 && sameAuthority(before, jobless) && string(jobless["observation_incarnation_id"]) == "null" && !sameJSON(jobless["received_at"], afterAgree["received_at"])
	// Leave the last valid observation agreeing before restarting the real agent.
	final, finalState, ok4 := send("restored_confirmation", restored.String(), receipt.text("job_id"), receipt.text("generation"))
	s.note("AT27", "check_observation_authority", disagreement && agreement && arrivalOnly && ok4 && rollbackTargetComplete(final, restored.String()) && sameAuthority(before, finalState))
}

func (s *updatesFollowup) rehashResult(doc updateDocument) updateDocument {
	raw, err := json.Marshal(doc)
	if err != nil {
		s.r.t.Fatal(err)
	}
	hash, err := canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		s.r.t.Fatal(err)
	}
	doc.set("payload_hash", hash)
	return doc
}

func (s *updatesFollowup) conflictBounds(receipt updateDocument) {
	hashes := []string{}
	valid := true
	for distinct := 0; distinct < 2; distinct++ {
		raw, _ := json.Marshal(receipt)
		conflict := document(raw)
		conflict.set("boot_id", uuid.NewString())
		s.rehashResult(conflict)
		hashes = append(hashes, conflict.text("payload_hash"))
		attempts := 1
		if distinct == 1 {
			attempts = 4
		}
		for replay := 0; replay < attempts; replay++ {
			response := s.nodePOST("/api/v1/agent/update-results", conflict)
			valid = valid && response.Status == 409 && response.Code == "ACK_CONFLICT"
			s.note("AT26", fmt.Sprintf("conflict_%d_replay_%d", distinct+1, replay), map[string]any{"request": conflict, "http_status": response.Status, "response": json.RawMessage(response.Body)})
		}
	}
	counts := s.readJSON(`SELECT jsonb_build_object(
 'conflicts',(SELECT coalesce(jsonb_object_agg(hash,n),'{}'::jsonb) FROM
 (SELECT encode(conflicting_hash,'hex') hash,count(*) n FROM update_result_conflicts WHERE result_id=$1 GROUP BY conflicting_hash) c),
 'audits',(SELECT coalesce(jsonb_object_agg(hash,n),'{}'::jsonb) FROM
 (SELECT after_diff->>'conflicting_hash' hash,count(*) n FROM audit_events
 WHERE action='update_result.conflict' AND after_diff->>'result_id'=$1::text GROUP BY after_diff->>'conflicting_hash') a))`, receipt.text("result_id"))
	s.note("AT26", "conflict_persistence_counts", counts)
	var got struct{ Conflicts, Audits map[string]int }
	expected := map[string]int{hashes[0]: 1, hashes[1]: 1}
	valid = valid && json.Unmarshal(counts, &got) == nil && len(expected) == 2 && reflect.DeepEqual(got.Conflicts, expected) && reflect.DeepEqual(got.Audits, expected)
	original := s.readJSON(`SELECT payload FROM update_results WHERE result_id=$1`, receipt.text("result_id"))
	raw, _ := json.Marshal(receipt)
	s.note("AT26", "accepted_receipt_unchanged", sameJSON(raw, original))
	s.note("AT26", "check_conflict_bounds", valid && sameJSON(raw, original))
}

func (s *updatesFollowup) failedRequiresCause(receipt updateDocument, next uuid.UUID) {
	batch, ok := s.batch("AT25", next)
	if !ok {
		return
	}
	raw, err := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
	var desired struct {
		Signed struct {
			Update updateDocument `json:"update"`
		} `json:"signed"`
	}
	if err != nil || json.Unmarshal(raw, &desired) != nil || desired.Signed.Update == nil {
		s.r.t.Fatal("negative result intent unavailable")
	}
	intent := desired.Signed.Update
	raw, _ = json.Marshal(receipt)
	malformed := document(raw)
	for _, key := range []string{"attempt", "batch_id", "generation", "incarnation_id", "job_id", "manifest_sha256", "node_id", "publication_intent", "target_id"} {
		malformed[key] = intent[key]
	}
	malformed["requested_release_id"] = intent["release_id"]
	malformed.set("result_id", uuid.NewString())
	malformed.set("state", "failed")
	malformed.set("stage", "check")
	malformed.set("error_code", nil)
	malformed.set("candidate_health", nil)
	malformed.set("previous_release_id", nil)
	malformed.set("started_at", time.Now().UTC())
	malformed.set("finished_at", time.Now().UTC())
	// This deliberately invalid TEST ONLY result is an API validation probe, not
	// updater evidence. Its health remains the real restored checks from the receipt.
	s.rehashResult(malformed)
	response := s.nodePOST("/api/v1/agent/update-results", malformed)
	target := s.target(batch)
	s.note("AT25", "failed_without_cause", map[string]any{"request": malformed, "http_status": response.Status, "response": json.RawMessage(response.Body), "batch_api_response": target.BatchJSON})
	s.note("AT25", "check_failed_requires_cause", response.Status == 422 && response.Code == "VALIDATION_FAILED" && target.State == "queued" && !target.NeedsAttention)
}

func testFollowup4Evidence(t *testing.T) {
	t.Helper()
	t.Run("revision7 raw presence and separate health", func(t *testing.T) {
		receipt := document([]byte(`{"previous_release_id":"candidate","health":{"tunnel_probe":"passed"},"candidate_health":{"tunnel_probe":"failed"}}`))
		batch := json.RawMessage(`{"targets":[{"previous_release_id":"candidate","health":{"tunnel_probe":"passed"},"candidate_health":{"tunnel_probe":"failed"}}]}`)
		if !targetEvidenceMatches(batch, receipt) {
			t.Fatal("matching rollback evidence rejected")
		}
		if targetEvidenceMatches(batch, nil) {
			t.Fatal("no receipt cannot borrow another result's health")
		}
		if targetEvidenceMatches(json.RawMessage(`{"targets":[{}]}`), nil) {
			t.Fatal("omitted fields are not explicit null")
		}
		if !targetEvidenceMatches(json.RawMessage(`{"targets":[{"previous_release_id":null,"health":null,"candidate_health":null}]}`), nil) {
			t.Fatal("explicit null evidence rejected")
		}
		receipt["health"] = receipt["candidate_health"]
		if targetEvidenceMatches(batch, receipt) {
			t.Fatal("candidate and restored checks are not interchangeable")
		}
	})
	t.Run("base pass cannot bypass followup4", func(t *testing.T) {
		records := map[string]*at.Record{}
		for _, id := range []string{"AT25", "AT26", "AT27"} {
			records[id] = &at.Record{ID: id, Status: at.StatusPass}
		}
		requireFollowup4Evidence(records)
		for id, rec := range records {
			if rec.Status != at.StatusBlocked {
				t.Fatalf("%s passed without required probes", id)
			}
		}
	})
}
