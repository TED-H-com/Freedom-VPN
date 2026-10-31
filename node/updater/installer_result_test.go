package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func TestD110_PendingIntegrityIsSeparateFromJobIdentity(t *testing.T) {
	releaseID, incarnation := uuid.New(), uuid.New()
	digest := strings.Repeat("ab", 32)
	var priorDigest string
	for range 2 {
		w := newWorld(t)
		w.eng.Node.Incarnation = incarnation
		path := filepath.Join(w.layout.Staging, PendingReleaseFile)
		raw, _ := json.Marshal(map[string]string{"release_id": releaseID.String(), "manifest_sha256": digest, "request_id": uuid.NewString()})
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		rec, err := readPending(path)
		if err != nil || rec.digest == priorDigest {
			t.Fatal("local record hash lost", err)
		}
		priorDigest = rec.digest
		if err := w.eng.ApplyPending(t.Context(), path); err == nil {
			t.Fatal("missing manifest accepted")
		}
		before, result := lastPayload(t, w)
		if result.JobId != installerjob.ID(releaseID, digest, incarnation) {
			t.Fatal("non D-110 job", result.JobId)
		}
		job, err := w.eng.Journal.GetJob(t.Context(), result.JobId.String(), 1)
		if err != nil {
			t.Fatal(err)
		}
		c, err := decodeClaimed(job.IntentJSON)
		if err != nil || c.PendingSHA256 != rec.digest {
			t.Fatal("pending integrity changed", err)
		}
		if err := w.eng.Journal.Close(); err != nil {
			t.Fatal(err)
		}
		w.eng.Journal = openJournal(t, w.layout.Journal)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if err := w.eng.ApplyPending(t.Context(), path); err == nil || !strings.Contains(err.Error(), "outcome=failed") {
			t.Fatal("sealed failure replay", err)
		}
		after, _ := lastPayload(t, w)
		if !bytes.Equal(before, after) {
			t.Fatal("replay rewrote seal")
		}
		assertConsumed(t, path, "failed")
	}
}

// Reproduce a pre-D-110 local failed receipt and cleanup using the real journal.
func legacyInstallerReceipt(t *testing.T, extension bool) (*world, string, []byte, updatejournal.OutboxEntry) {
	t.Helper()
	w := newWorld(t)
	rel := w.makeRelease("legacy", 1, []int{1})
	path := stageInstaller(t, w, rel)
	raw := fileBytes(t, path)
	rec, err := readPending(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := w.eng.installerClaim(t.Context(), path, rel.id, rel.sum)
	if err != nil {
		t.Fatal(err)
	}
	ns := uuid.NewSHA1(uuid.NameSpaceOID, []byte("fvpn-installer-apply|"+rel.id.String()+"|"+rel.sum))
	c.Intent.JobId = uuid.NewSHA1(ns, []byte(rec.digest+"|"+w.node.Incarnation.String()))
	c.PendingSHA256 = rec.digest
	intent, _ := json.Marshal(c)
	job, err := w.eng.Journal.RecordJob(t.Context(), updatejournal.Job{JobID: c.Intent.JobId.String(), Attempt: 1, Generation: c.Intent.Generation, ReleaseID: rel.id.String(), ManifestSHA256: rel.sum, IntentJSON: intent})
	if err != nil {
		t.Fatal(err)
	}
	payload, hash, id, err := buildResult(resultInput{Claimed: c, State: api.UpdateResultStateFailed, Stage: api.UpdateStageVerify, Code: apierr.CheckpointFailed, Schema: 1})
	if err != nil {
		t.Fatal(err)
	}
	if extension {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatal(err)
		}
		fields["legacy_external_seal"] = json.RawMessage(`"TEST ONLY opaque seal bound to old job"`)
		payload, hash = sealTestFields(t, fields)
	}
	ent := updatejournal.OutboxEntry{JobID: job.JobID, ResultID: id.String(), Attempt: 1, Generation: job.Generation, State: "failed", Payload: payload, PayloadHash: hash}
	if err := w.eng.Journal.Enqueue(t.Context(), ent); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.finishInstaller(t.Context(), c, job, "failed", "CHECKPOINT_FAILED"); err != nil {
		t.Fatal(err)
	}
	return w, path, raw, ent
}

func sealTestFields(t *testing.T, fields map[string]json.RawMessage) ([]byte, string) {
	t.Helper()
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		t.Fatal(err)
	}
	fields["payload_hash"], _ = json.Marshal(hash)
	raw, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, hash
}

func TestD114_RebindBeforeUploadAndReplayExactSeal(t *testing.T) {
	w, path, raw, old := legacyInstallerReceipt(t, false)
	beforeJob, _ := w.eng.Journal.GetJob(t.Context(), old.JobID, 1)
	var sent [][]byte
	status := 503
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sent = append(sent, body)
		var result api.UpdateResult
		if err := json.Unmarshal(body, &result); err != nil {
			t.Error(err)
		}
		if result.JobId != installerjob.ID(result.RequestedReleaseId, result.ManifestSha256, result.IncarnationId) || result.ResultId.String() == old.ResultID || r.Header.Get("Idempotency-Key") != result.ResultId.String() {
			t.Error("legacy identity uploaded")
		}
		rw.WriteHeader(status)
		if status == 202 {
			_, _ = rw.Write([]byte(`{"accepted":true,"accepted_as_history":true}`))
		}
	}))
	defer srv.Close()
	w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
	if err := w.eng.Flush(t.Context()); err == nil {
		t.Fatal("503 accepted")
	}
	out, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || len(out) != 2 || out[0].Status != updatejournal.OutboxRejected || out[0].LastError != "installer_job_id_rederived" || !bytes.Equal(out[0].Payload, old.Payload) || out[1].Status != updatejournal.OutboxPending {
		t.Fatalf("migration evidence: %+v %v", out, err)
	}
	for _, ent := range []updatejournal.OutboxEntry{old, out[1]} {
		got, err := canonjson.HashHex(ent.Payload, "payload_hash")
		if err != nil || got != ent.PayloadHash {
			t.Fatal("bad seal", err)
		}
	}
	var a, b map[string]json.RawMessage
	_ = json.Unmarshal(old.Payload, &a)
	_ = json.Unmarshal(out[1].Payload, &b)
	for _, key := range []string{"job_id", "result_id", "payload_hash"} {
		delete(a, key)
		delete(b, key)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("re-derivation changed outcome evidence")
	}
	if err := w.eng.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = openJournal(t, w.layout.Journal)
	status = 202
	if err := w.eng.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Flush(t.Context()); err != nil || len(sent) != 2 || !bytes.Equal(sent[0], sent[1]) {
		t.Fatal("restart replay changed bytes or retried ACK", err)
	}
	if afterJob, _ := w.eng.Journal.GetJob(t.Context(), old.JobID, 1); !reflect.DeepEqual(beforeJob, afterJob) {
		t.Fatal("local claim rewritten")
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.ApplyPending(t.Context(), path); err == nil {
		t.Fatal("failed legacy job replay returned success")
	}
	history, err := w.eng.Journal.History(t.Context(), "")
	if err != nil || len(history) != 1 || history[0].ResultID != out[1].ResultID {
		t.Fatal("replacement ACK lost", history, err)
	}
	if err := w.eng.Flush(t.Context()); err != nil || len(sent) != 2 {
		t.Fatal("legacy job re-minted", err)
	}
	assertConsumed(t, path, "failed")
}

func TestD114_AcceptedLegacyHistoryNeverReminted(t *testing.T) {
	w, path, raw, old := legacyInstallerReceipt(t, false)
	if _, err := w.eng.NoteAck(t.Context(), old.ResultID, old.PayloadHash, true, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	before, err := w.eng.Journal.History(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.ApplyPending(t.Context(), path); err == nil {
		t.Fatal("failed history returned success")
	}
	out, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || len(out) != 0 {
		t.Fatal("accepted legacy re-minted", out, err)
	}
	after, err := w.eng.Journal.History(t.Context(), "")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("accepted history changed", err)
	}
	// A fresh request ID cannot create new work for accepted legacy history either.
	var record map[string]any
	_ = json.Unmarshal(raw, &record)
	record["request_id"] = uuid.NewString()
	changed, _ := json.Marshal(record)
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.ApplyPending(t.Context(), path); err == nil || !strings.Contains(err.Error(), "pending record changed") {
		t.Fatal("legacy restaging minted new job", err)
	}
}

func TestD114_UnrecreatableSealMismatchIsTerminal(t *testing.T) {
	w, path, _, old := legacyInstallerReceipt(t, true)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, old.Payload) {
			t.Error("opaque seal rewritten")
		}
		rw.WriteHeader(409)
		_, _ = rw.Write([]byte(`{"error":{"code":"STATE_CONFLICT","message":"private detail","details":{"reason":"job_id_mismatch"}}}`))
	}))
	defer srv.Close()
	w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
	if err := w.eng.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = openJournal(t, w.layout.Journal)
	if err := w.eng.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	out, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || len(out) != 1 || calls != 1 || out[0].SendAttempts != 1 || out[0].LastHTTPStatus != 409 || out[0].LastAPICode != "STATE_CONFLICT" || out[0].LastError != "job_id_mismatch" || out[0].Status != updatejournal.OutboxRejected || !bytes.Equal(out[0].Payload, old.Payload) {
		t.Fatalf("terminal receipt: %+v calls=%d %v", out, calls, err)
	}
	assertConsumed(t, path, "failed")
}

func TestF903_ApplySealedCheckpointFailure(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("baseline", 1, []int{1})
	w.seed(old)
	next := w.makeRelease("next", 1, []int{1})
	w.meter.err = errors.New("TEST ONLY missing durable checkpoint")
	path := stageInstaller(t, w, next)
	err := w.eng.ApplyPending(context.Background(), path)
	if ExitCode(err) != 1 || !strings.Contains(err.Error(), "CHECKPOINT_FAILED") || !strings.Contains(err.Error(), "outcome=failed") {
		t.Fatalf("failed apply reported success: %v", err)
	}
	result := w.oneResult()
	if result.State != api.UpdateResultStateFailed || requiredValue(t, result.Stage) != api.UpdateStageFinalSample {
		t.Fatalf("wrong sealed outcome: %+v", result)
	}
	w.mustCurrent(old.id)
	if stops, starts := w.svc.counts(); stops != 0 || starts != 0 {
		t.Fatal("checkpoint failure restarted services")
	}
	assertConsumed(t, path, "failed")
}
