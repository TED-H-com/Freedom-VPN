package updatejournal

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestReviewR9_PendingV2EnrichmentIsAtomicAcrossCrash(t *testing.T) {
	j, dir := openTest(t)
	ctx := context.Background()
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "txn", JobID: "job", Attempt: 1, Generation: 8, ReleaseID: "candidate", ManifestSHA256: "aa"}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "txn", StageRollingBack, StageUpdate{Destructive: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "txn", StageRolledBack, StageUpdate{}); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"requested_release_id":"candidate","state":"rolled_back"}`)
	hash, err := canonjson.HashHex(payload, "payload_hash")
	if err != nil {
		t.Fatal(err)
	}
	entry := OutboxEntry{ResultID: "result-review", JobID: "job", Attempt: 1, Generation: 8, State: "rolled_back", Payload: payload, PayloadHash: hash}
	if err = j.Enqueue(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err = j.RecordDeliveryFailure(ctx, entry.ResultID, DeliveryFailure{Status: OutboxPending}); err != nil {
		t.Fatal(err)
	}
	if _, err = j.db.Exec(`CREATE TRIGGER fail_payload BEFORE UPDATE OF payload ON result_outbox BEGIN SELECT RAISE(ABORT, 'killed between payload and hash'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = j.PopulateCandidateHealth(ctx, entry.ResultID); err == nil {
		t.Fatal("faulted enrichment reported success")
	}
	pending, err := j.Pending(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].ResultID != entry.ResultID || !bytes.Equal(pending[0].Payload, payload) || pending[0].PayloadHash != hash || bytes.Contains(pending[0].Payload, []byte("candidate_health")) {
		t.Fatalf("TODO(review): half-enriched row is visible to flush: %+v %v", pending, err)
	}
	if err = j.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	pending, err = reopened.Pending(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].ResultID != entry.ResultID || !bytes.Equal(pending[0].Payload, payload) || pending[0].PayloadHash != hash {
		t.Fatalf("TODO(review): crash before ack resent different bytes: %+v %v", pending, err)
	}
	if _, err = reopened.db.Exec(`DROP TRIGGER fail_payload`); err != nil {
		t.Fatal(err)
	}
	got, err := reopened.PopulateCandidateHealth(ctx, entry.ResultID)
	if err != nil {
		t.Fatal(err)
	}
	covered, err := canonjson.HashHex(got.Payload, "payload_hash")
	if err != nil || covered != got.PayloadHash {
		t.Fatalf("TODO(review): covering hash does not match the enriched payload: %s %s %v", covered, got.PayloadHash, err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(got.Payload, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["candidate_health"]) != "null" {
		t.Fatal(string(got.Payload))
	}
	delete(fields, "candidate_health")
	withoutField, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	uncovered, err := canonjson.HashHex(withoutField, "payload_hash")
	if err != nil || uncovered == got.PayloadHash {
		t.Fatalf("TODO(review): covering hash does not cover candidate_health: %s", got.PayloadHash)
	}
	if err = reopened.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = again.Close() })
	pending, err = again.Pending(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].ResultID != got.ResultID || !bytes.Equal(pending[0].Payload, got.Payload) || pending[0].PayloadHash != got.PayloadHash {
		t.Fatalf("TODO(review): crash after enrichment resent different bytes: %+v %v", pending, err)
	}
}
