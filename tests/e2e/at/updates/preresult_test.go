//go:build e2e

package updates_test

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	at "github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates"
	"github.com/google/uuid"
)

func requireFollowup5Evidence(records map[string]*at.Record) {
	for id, names := range map[string][]string{
		"AT23": {"probe_inspection"}, "AT25": {"probe_inspection"},
		"AT24": {"pre_result_versions", "first_result_last_good"},
		"AT27": {"pre_result_audits", "first_result_last_good"},
	} {
		for _, name := range names {
			ok := false
			for _, ob := range records[id].Observations {
				if ob.Name == "check_"+name && ob.Detail == "true" {
					ok = true
				}
			}
			if !ok {
				records[id].Status = at.StatusBlocked
				records[id].Actual += "; follow-up 5 prerequisite/check incomplete: " + name
			}
		}
	}
}

// The fourth registered release keeps (a)'s unexpected current and (b)'s
// unexpected last-good distinct. Reuse the existing real builder and API path.
func (s *updatesFollowup) unrelatedRelease() uuid.UUID {
	stack, err := e2e.OpenStack()
	if err != nil {
		s.r.t.Fatal("shared stack unavailable")
	}
	defer stack.Close()
	built, err := stack.BuildRelease(s.ctx, s.r.t.TempDir(), "1.0.3", "4", "e2e-unrelated", e2e.ReleaseOptions{})
	if err != nil {
		s.r.t.Fatal("shared unrelated release build failed")
	}
	if err = stack.RegisterRelease(s.ctx, e2e.BrowserSession{Client: s.admin.http, CSRF: s.admin.csrf}, built); err != nil {
		s.r.t.Fatal("shared unrelated release registration failed: " + at.Redact(err.Error()))
	}
	var env api.ReleaseManifestEnvelope
	if json.Unmarshal(built.Manifest, &env) != nil {
		s.r.t.Fatal("unrelated manifest invalid")
	}
	var envelope struct {
		Signed json.RawMessage `json:"signed"`
	}
	_ = json.Unmarshal(built.Manifest, &envelope)
	signedHash, _ := canonjson.HashHex(envelope.Signed)
	digests := map[string]string{"manifest_sha256_1.0.3": built.ManifestSHA256, "signed_manifest_sha256_1.0.3": signedHash}
	for _, a := range env.Signed.Artifacts {
		digests["artifact_sha256_1.0.3_"+string(a.Architecture)] = a.Sha256
	}
	for id := range s.rec {
		s.note(id, "unrelated_release", map[string]any{"release_id": built.ID, "version": "1.0.3", "digests": digests, "purpose": "TEST ONLY registered, never installed or made fleet default"})
	}
	return built.ID
}

func (s *updatesFollowup) acceptedResultCount() int {
	raw := s.readJSON(`SELECT to_jsonb(count(*)) FROM update_results WHERE node_id=$1 AND incarnation_id=$2 AND ack->>'accepted'='true'`, s.r.nodeID, s.r.incID)
	var count int
	if json.Unmarshal(raw, &count) != nil {
		s.r.t.Fatal("result count unavailable")
	}
	return count
}

// Return complete API pages, not a manufactured audit summary. Counts use the
// requested job/release pair and the API's redacted changes.after object.
func (s *updatesFollowup) unexpectedAudit(job string) ([]json.RawMessage, map[string]int, bool) {
	pages := []json.RawMessage{}
	counts := map[string]int{}
	cursor := ""
	for pageNo := 0; pageNo < 20; pageNo++ {
		path := s.r.lab.public + "/api/v1/admin/audit-events?resource_type=node&resource_id=" + s.r.nodeID.String() + "&limit=100"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		response, err := s.admin.call(s.ctx, "GET", path, nil, nil)
		pages = append(pages, response.Body)
		var page api.AuditEventList
		if err != nil || response.Status != 200 || json.Unmarshal(response.Body, &page) != nil {
			return pages, counts, false
		}
		for _, event := range page.Items {
			after, err := event.Changes.After.Get()
			if event.Action == "update_observation.unexpected_release" && err == nil && after["job_id"] == job {
				release, ok := after["release_id"].(string)
				if !ok {
					return pages, counts, false
				}
				counts[release]++
			}
		}
		next, err := page.NextCursor.Get()
		if err != nil || next == "" {
			return pages, counts, true
		}
		if next == cursor {
			return pages, counts, false
		}
		cursor = next
	}
	return pages, counts, false
}

func unchangedVersions(before, after updateDocument) bool {
	for _, key := range []string{"current_release_id", "last_good_release_id", "recovery_required"} {
		if !sameJSON(before[key], after[key]) {
			return false
		}
	}
	return true
}

func (s *updatesFollowup) preResultInjection(batch api.UpdateBatchAccepted, boot, target, unexpectedCurrent, unexpectedLastGood uuid.UUID) {
	pause := filepath.Join(s.r.lab.state, "node", "var", "e2e", "pause-agent")
	if writeAtomic(pause, []byte("TEST ONLY pre-result observation injection\n")) != nil {
		s.r.t.Fatal("pause request failed")
	}
	defer func() {
		_, err := s.r.lab.docker(s.r.lab.node, "/usr/local/bin/agent-service.sh", "start", "/usr/local")
		_ = os.Remove(pause)
		if err != nil {
			s.r.t.Error("restore boot agent failed")
		}
	}()
	select {
	case <-time.After(4 * time.Second):
	case <-s.ctx.Done():
		return
	}
	if _, err := s.r.lab.docker(s.r.lab.node, "/usr/local/bin/agent-service.sh", "stop"); err != nil {
		s.r.t.Fatal("stop pre-result agent failed")
	}
	before := s.authoritySnapshot()
	count := s.acceptedResultCount()
	original := s.readJSON(`SELECT report_payload FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 ORDER BY received_at DESC LIMIT 1`, s.r.nodeID, s.r.incID)
	report := document(original)
	raw, err := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
	var desired struct {
		Signed struct {
			Update api.UpdateIntent `json:"update"`
		} `json:"signed"`
	}
	if err != nil || json.Unmarshal(raw, &desired) != nil {
		s.r.t.Fatal("pre-result authorized intent unavailable")
	}
	intent := desired.Signed.Update
	boundBatch, batchErr := intent.BatchId.Get()
	bootRecorded := before.text("current_release_id") == boot.String() || before.text("last_good_release_id") == boot.String()
	valid := count == 0 && batchErr == nil && boundBatch == batch.BatchId && intent.ReleaseId == target && intent.IncarnationId == s.r.incID
	evidence := map[string]any{"before": before, "baseline_observation": json.RawMessage(original), "accepted_results_before": count, "boot_release_id": boot, "boot_recorded_before": bootRecorded, "target_release_id": target, "job_id": intent.JobId, "generation": intent.Generation, "unrelated_current_release_id": unexpectedCurrent, "unrelated_last_good_release_id": unexpectedLastGood}
	steps := []any{}
	auditOK := true
	for index, scenario := range []struct {
		name              string
		current, lastGood uuid.UUID
	}{
		{"a_unrelated_current_and_last_good", unexpectedCurrent, unexpectedCurrent},
		{"b_target_current_unrelated_last_good", target, unexpectedLastGood},
		{"c_target_current_boot_last_good", target, boot},
	} {
		requests := []any{}
		for repeat := 0; repeat < 4; repeat++ { // initial heartbeat plus three fresh report-ID repetitions
			report.set("report_id", uuid.NewString())
			report.set("observed_at", time.Now().UTC())
			report.set("current_release_id", scenario.current)
			report.set("last_good_release_id", scenario.lastGood)
			report.set("update", map[string]any{"job_id": intent.JobId, "generation": intent.Generation, "stage": "claim", "state": "queued"})
			response := s.nodePOST("/api/v1/agent/observations", report)
			after := s.authoritySnapshot()
			view := s.target(batch)
			versions := after.text("current_release_id") == target.String() && sameJSON(before["last_good_release_id"], after["last_good_release_id"]) && sameJSON(before["recovery_required"], after["recovery_required"])
			if index == 0 {
				versions = unchangedVersions(before, after) && view.Unconfirmed
			}
			zero := s.acceptedResultCount() == 0
			valid = valid && response.Status == 202 && versions && zero
			// Snapshot the request map now: the next repetition changes ID and time.
			requestRaw, _ := json.Marshal(report)
			requests = append(requests, map[string]any{"repeat": repeat, "request": json.RawMessage(requestRaw), "http_status": response.Status, "response": json.RawMessage(response.Body), "batch_api_response": view.BatchJSON, "projection": after, "versions_match": versions, "no_accepted_result": zero})
		}
		pages, counts, complete := s.unexpectedAudit(intent.JobId.String())
		expected := map[string]int{unexpectedCurrent.String(): 1}
		if index > 0 {
			expected[unexpectedLastGood.String()] = 1
		}
		countsOK := complete && reflect.DeepEqual(counts, expected)
		auditOK = auditOK && countsOK
		steps = append(steps, map[string]any{"case": scenario.name, "heartbeats": requests, "audit_pages": pages, "audit_counts": counts, "expected_audit_counts": expected, "audit_matches": countsOK})
	}
	evidence["steps"] = steps
	for _, id := range []string{"AT24", "AT27"} {
		s.note(id, "pre_result_injection", evidence)
		s.note(id, "check_pre_result_versions", valid && bootRecorded)
		s.note(id, "check_pre_result_audits", auditOK)
	}
	s.r.t.Logf("pre-result injection: versions=%t boot_recorded=%t audits=%t", valid, bootRecorded, auditOK)
}

func (s *updatesFollowup) firstResultLastGood(good uuid.UUID, accepted bool) {
	after := s.authoritySnapshot()
	count := s.acceptedResultCount()
	ok := accepted && count == 1 && after.text("current_release_id") == good.String() && after.text("last_good_release_id") == good.String()
	for _, id := range []string{"AT23", "AT24", "AT27"} {
		s.note(id, "first_result_last_good", map[string]any{"projection": after, "accepted_result_count": count, "expected_release": good})
		s.note(id, "check_first_result_last_good", ok)
	}
}

func testFollowup5Evidence(t *testing.T) {
	t.Helper()
	t.Run("pre-result evidence cannot be skipped", func(t *testing.T) {
		records := map[string]*at.Record{}
		for _, id := range []string{"AT23", "AT24", "AT25", "AT26", "AT27"} {
			records[id] = &at.Record{Status: at.StatusPass}
		}
		requireFollowup5Evidence(records)
		for _, id := range []string{"AT23", "AT24", "AT25", "AT27"} {
			if records[id].Status != at.StatusBlocked {
				t.Fatalf("%s bypassed follow-up 5", id)
			}
		}
	})
}
