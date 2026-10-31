//go:build e2e

package updates_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

func marshalDoc(d updateDocument) []byte { b, _ := json.Marshal(d); return b }

func (h *handoffRun) journal() *sql.DB {
	h.s.r.t.Helper()
	if h.run("", "/usr/local/bin/fvpn-e2e-nodeconfig", "journal-copy").Exit != 0 {
		return nil
	}
	path := filepath.Join(h.s.r.t.TempDir(), "journal.db")
	receipt, err := h.stack.ExportContainerFile(h.s.ctx, "installer-node", "/var/lib/vpn-node/e2e/journal.snapshot.db", path)
	h.note("journal_snapshot", receipt)
	if err != nil {
		return nil
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return nil
	}
	return db
}
func (h *handoffRun) receipt(name string) updateDocument {
	db := h.journal()
	if db == nil {
		return nil
	}
	defer db.Close()
	var payload []byte
	var hash string
	if db.QueryRow(`SELECT payload,payload_hash FROM result_outbox ORDER BY seq DESC LIMIT 1`).Scan(&payload, &hash) != nil {
		// Apply may flush immediately. The local ACK history pins the accepted
		// seal hash; recover its immutable body from central and verify that pin.
		var resultID, jobID string
		if db.QueryRow(`SELECT result_id,job_id,payload_hash FROM result_history ORDER BY acked_at DESC LIMIT 1`).Scan(&resultID, &jobID, &hash) != nil {
			return nil
		}
		payload = h.s.readJSON(`SELECT payload FROM installer_update_results WHERE node_id=$1 AND job_id=$2 AND payload->>'result_id'=$3`, uuid.MustParse(h.nodeID), uuid.MustParse(jobID), resultID)
		h.note(name+"_source", "central immutable payload matched to local acknowledged result_id/job_id/hash")
	}
	sum, err := canonjson.HashHex(payload, "payload_hash")
	if err != nil || sum != hash {
		h.note(name+"_hash_valid", false)
		return nil
	}
	h.note(name+"_hash_valid", true)
	d := document(payload)
	h.note(name, d)
	return d
}
func (h *handoffRun) claimBoot() string {
	db := h.journal()
	if db == nil {
		return ""
	}
	defer db.Close()
	var raw []byte
	if db.QueryRow(`SELECT intent_json FROM jobs ORDER BY rowid DESC LIMIT 1`).Scan(&raw) != nil {
		return ""
	}
	d := document(raw)
	h.note("job_claim", d)
	return d.text("boot_id")
}
func (h *handoffRun) correlated(trace, kernel string) updateDocument {
	for _, line := range strings.Split(trace, "\n") {
		var sample struct {
			At         time.Time         `json:"at"`
			Boot       string            `json:"boot_id"`
			Latch      string            `json:"latch"`
			Checkpoint string            `json:"checkpoint"`
			Notes      map[string]string `json:"wait_notes"`
		}
		if json.Unmarshal([]byte(line), &sample) != nil {
			continue
		}
		cp := document([]byte(sample.Checkpoint))
		barrier := cp.text("barrier_id")
		if id, err := uuid.Parse(barrier); err != nil || id == uuid.Nil {
			continue
		}
		for _, raw := range sample.Notes {
			var wait struct {
				Boot     string    `json:"boot_id"`
				Identity string    `json:"identity"`
				Started  time.Time `json:"started"`
				Deadline time.Time `json:"deadline"`
			}
			if json.Unmarshal([]byte(raw), &wait) == nil && barrier == wait.Identity && sample.Latch == wait.Identity && sample.Boot == kernel && wait.Boot == kernel && !sample.At.Before(wait.Started) && sample.At.Before(wait.Deadline) {
				h.note("correlated_checkpoint", cp)
				h.note("correlated_wait", wait)
				return cp
			}
		}
	}
	return nil
}
func checkpointMatches(cp, result updateDocument) bool {
	usage := document(result["usage_checkpoint"])
	if cp == nil || usage == nil {
		return false
	}
	for _, k := range []string{"closed_meter_epoch_id", "last_durable_event_seq", "central_acked_through"} {
		if len(cp[k]) == 0 || !bytes.Equal(cp[k], usage[k]) {
			return false
		}
	}
	return usage.text("new_meter_epoch_id") != "" && usage.text("new_meter_epoch_id") != cp.text("closed_meter_epoch_id")
}

// Prove the matched checkpoint names a real finalized durable epoch. The
// fixture is quiet; a zero watermark is not evidence of nonzero traffic.
func (h *handoffRun) finalization(cp updateDocument) bool {
	if cp == nil || h.run("", "/usr/local/bin/fvpn-e2e-nodeconfig", "usage-copy").Exit != 0 {
		return false
	}
	path := filepath.Join(h.s.r.t.TempDir(), "usage.db")
	exported, err := h.stack.ExportContainerFile(h.s.ctx, "installer-node", "/var/lib/vpn-node/e2e/usage.snapshot.db", path)
	h.note("usage_snapshot", exported)
	if err != nil {
		return false
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return false
	}
	defer db.Close()
	var closed, reason, boot string
	if db.QueryRow(`SELECT closed_at,close_reason,boot_id FROM epochs WHERE epoch_id=?`, cp.text("closed_meter_epoch_id")).Scan(&closed, &reason, &boot) != nil {
		return false
	}
	h.note("finalized_epoch", map[string]string{"epoch_id": cp.text("closed_meter_epoch_id"), "closed_at": closed, "close_reason": reason, "boot_id": boot})
	return closed != "" && reason == "final_sample"
}

type installerHTTP struct {
	status int
	body   updateDocument
}

func (h *handoffRun) submit(name string, d updateDocument, key string) installerHTTP {
	if d == nil {
		h.note(name, "no sealed receipt available")
		return installerHTTP{}
	}
	if h.put("/var/tmp/e2e-rel/result.json", marshalDoc(d)) != nil {
		return installerHTTP{}
	}
	out := h.run("", "curl", "--silent", "--show-error", "--max-time", "20", "--cacert", "/var/tmp/e2e-rel/ca.pem", "--cert", "/var/lib/vpn-node/identity/cert.pem", "--key", "/var/lib/vpn-node/identity/key.pem", "-H", "Content-Type: application/json", "-H", "Idempotency-Key: "+key, "--data-binary", "@/var/tmp/e2e-rel/result.json", "-w", "\nHTTP %{http_code}\n", h.s.r.lab.agent+"/api/v1/agent/update-results")
	i := strings.LastIndex(out.Stdout, "\nHTTP ")
	if out.Exit != 0 || i < 0 {
		h.note(name, out)
		return installerHTTP{}
	}
	code, _ := strconv.Atoi(strings.TrimSpace(out.Stdout[i+6:]))
	body := document([]byte(out.Stdout[:i]))
	h.note(name, map[string]any{"status": code, "body": body})
	return installerHTTP{code, body}
}
func reseal(d updateDocument) updateDocument {
	out := document(marshalDoc(d))
	delete(out, "payload_hash")
	hash, err := canonjson.HashHex(marshalDoc(out))
	if err != nil {
		return nil
	}
	out.set("payload_hash", hash)
	return out
}
func sameAck(a, b installerHTTP) bool {
	return a.status == 202 && b.status == 202 && bytes.Equal(marshalDoc(a.body), marshalDoc(b.body))
}
func conflictHTTP(a installerHTTP) bool {
	return a.status == 409 && document(a.body["error"]).text("code") == "ACK_CONFLICT"
}
func (h *handoffRun) resultAcceptance(node string, candidate e2e.BuiltRelease, result updateDocument) (installerHTTP, string) {
	if result == nil || result.text("state") != "succeeded" {
		h.note("central_success_not_reached", true)
		return installerHTTP{}, ""
	}
	// Independently reproduce D-110's documented UUIDv5 formula, not its helper.
	namespace := uuid.NewSHA1(uuid.NameSpaceOID, []byte("fvpn-installer-apply|"+candidate.ID.String()+"|"+candidate.ManifestSHA256))
	expected := uuid.NewSHA1(namespace, []byte(candidate.ManifestSHA256+"|"+h.rec.NodeIncarnation))
	h.note("recomputed_job_id", expected)
	h.check("job_id", result.text("job_id") == expected.String())
	key := uuid.NewString()
	ack := h.submit("success_submission", result, key)
	beforeReplay := h.central("central_after_success", node)
	h.check("central", ack.status == 202 && installerCentralSuccess(beforeReplay, candidate.ID.String()) && document(beforeReplay["releases"]).text("last_good_release_id") == candidate.ID.String())
	replay := h.submit("keyed_exact_replay", result, key)
	newKey := h.submit("new_key_exact_replay", result, uuid.NewString())
	afterReplay := h.central("central_after_replay", node)
	h.check("replay", sameAck(ack, replay) && sameAck(ack, newKey) && beforeReplay.text("resource_version") == afterReplay.text("resource_version"))
	modified := document(marshalDoc(result))
	modified.set("result_id", uuid.NewString())
	modified = reseal(modified)
	conflict := h.submit("modified_replay", modified, uuid.NewString())
	h.check("conflict", conflictHTTP(conflict))
	wrong := document(marshalDoc(result))
	wrong.set("job_id", uuid.NewString())
	wrong.set("result_id", uuid.NewString())
	wrong = reseal(wrong)
	wrongAck := h.submit("wrong_job_id", wrong, uuid.NewString())
	errDoc := document(wrongAck.body["error"])
	afterWrong := h.central("central_after_wrong_job", node)
	h.check("wrong_job", wrongAck.status == 409 && errDoc.text("code") == "STATE_CONFLICT" && document(errDoc["details"]).text("reason") == "job_id_mismatch" && beforeReplay.text("resource_version") == afterWrong.text("resource_version"))
	return ack, key
}

func (h *handoffRun) resultCorrection(node string, baseline, candidate e2e.BuiltRelease, result updateDocument, ack installerHTTP, key string) {
	if result == nil || ack.status != 202 {
		return
	}
	beforeReplay := h.central("central_before_correction", node)
	// D-127/r16 M-2 HTTP contract stimulus. This explicit synthetic rollback
	// is derived from a real receipt; it is not evidence of a physical rollback.
	// The regular AT26 independently proves the real updater rollback.
	correction := document(marshalDoc(result))
	correction.set("result_id", uuid.NewString())
	correction.set("state", "rolled_back")
	correction.set("stage", "report")
	correction.set("current_release_id", baseline.ID.String())
	correction.set("previous_release_id", candidate.ID.String())
	correction.set("error_code", "HEALTH_CHECK_FAILED")
	correction.set("finished_at", time.Now().UTC().Format(time.RFC3339))
	correction["candidate_health"] = result["health"]
	correction = reseal(correction)
	h.note("correction_scope", "TEST ONLY central HTTP rollback stimulus based on the genuine success; no local rollback or baseline health is claimed by this synthetic document")
	h.note("correction_payload", correction)
	correctionKey := uuid.NewString()
	corrected := h.submit("rollback_correction", correction, correctionKey)
	view := h.central("central_after_correction", node)
	replayFirst := h.submit("first_receipt_after_correction", result, key)
	replayCorrection := h.submit("correction_exact_replay", correction, correctionKey)
	replayCorrectionFresh := h.submit("correction_new_key_replay", correction, uuid.NewString())
	third := document(marshalDoc(result))
	third.set("result_id", uuid.NewString())
	third = reseal(third)
	thirdAck := h.submit("third_outcome_conflict", third, uuid.NewString())
	final := h.central("central_after_correction_replays", node)
	pointers := document(view["releases"])
	history := h.s.readJSON(`SELECT coalesce(jsonb_agg(jsonb_build_object('job_id',job_id,'outcome_sequence',outcome_sequence,'payload',payload,'ack',ack) ORDER BY received_at,outcome_sequence), '[]'::jsonb) FROM installer_update_results WHERE node_id=$1`, uuid.MustParse(node))
	h.note("central_installer_history", history)
	var rows []updateDocument
	_ = json.Unmarshal(history, &rows)
	sameJob := 0
	firstKept := false
	for _, row := range rows {
		if row.text("job_id") == result.text("job_id") {
			sameJob++
			if document(row["payload"]).text("payload_hash") == result.text("payload_hash") {
				firstKept = true
			}
		}
	}
	h.check("correction", corrected.status == 202 && pointers.text("current_release_id") == baseline.ID.String() && pointers.text("last_good_release_id") == baseline.ID.String() && document(view["update"]).text("state") == "rolled_back" && beforeReplay.text("resource_version") != view.text("resource_version") && view.text("resource_version") == final.text("resource_version") && sameAck(ack, replayFirst) && sameAck(corrected, replayCorrection) && sameAck(corrected, replayCorrectionFresh) && conflictHTTP(thirdAck) && sameJob == 2 && firstKept)
	h.check("etag", h.etags["central_after_success"] != "" && h.etags["central_after_correction"] != "" &&
		h.etags["central_after_success"] == h.etags["central_after_replay"] && h.etags["central_after_success"] == h.etags["central_after_wrong_job"] &&
		h.etags["central_after_success"] != h.etags["central_after_correction"] && h.etags["central_after_correction"] == h.etags["central_after_correction_replays"])
	h.note("receipt_cardinality", fmt.Sprintf("%d total receipts; %d outcomes for the success job", len(rows), sameJob))
	h.run("flush_success", installedUpdater, "flush", "--config", installerConfig)
}
