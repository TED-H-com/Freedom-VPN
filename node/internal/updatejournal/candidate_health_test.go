package updatejournal

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestD062_PopulateCandidateHealthAtomicAndPendingOnly(t *testing.T) {
	for _, status := range []OutboxStatus{OutboxPending, OutboxRejected, OutboxConflict} {
		t.Run(string(status), func(t *testing.T) {
			j, _ := openTest(t)
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
			entry := OutboxEntry{ResultID: "result", JobID: "job", Attempt: 1, Generation: 8, State: "rolled_back", Payload: payload, PayloadHash: hash}
			if err := j.Enqueue(ctx, entry); err != nil {
				t.Fatal(err)
			}
			if err := j.RecordDeliveryFailure(ctx, entry.ResultID, DeliveryFailure{Status: status}); err != nil {
				t.Fatal(err)
			}
			if status == OutboxPending {
				if _, err := j.db.Exec(`CREATE TRIGGER fail_payload BEFORE UPDATE OF payload ON result_outbox BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
					t.Fatal(err)
				}
				if _, err := j.PopulateCandidateHealth(ctx, entry.ResultID); err == nil {
					t.Fatal("failed commit reported success")
				}
				out, err := j.Outbox(ctx)
				if err != nil || len(out) != 1 || !bytes.Equal(out[0].Payload, payload) || out[0].PayloadHash != hash {
					t.Fatal("partial payload/hash update", out, err)
				}
				if _, err := j.db.Exec(`DROP TRIGGER fail_payload`); err != nil {
					t.Fatal(err)
				}
			}
			got, err := j.PopulateCandidateHealth(ctx, entry.ResultID)
			if err != nil {
				t.Fatal(err)
			}
			if status == OutboxPending {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(got.Payload, &fields); err != nil || string(fields["candidate_health"]) != "null" {
					t.Fatal("pre-check rollback must add null", string(got.Payload), err)
				}
				hash, err := canonjson.HashHex(got.Payload, "payload_hash")
				if err != nil || hash != got.PayloadHash {
					t.Fatal("hash not updated with payload", err)
				}
			} else if !bytes.Equal(got.Payload, payload) || got.PayloadHash != hash {
				t.Fatal("nonpending diagnostic evidence rewritten")
			}
			again, err := j.PopulateCandidateHealth(ctx, entry.ResultID)
			if err != nil || !bytes.Equal(again.Payload, got.Payload) || again.PayloadHash != got.PayloadHash {
				t.Fatal("compatibility update was not idempotent", err)
			}
			var version int
			if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 2 {
				t.Fatal("journal schema unexpectedly changed", version, err)
			}
		})
	}
}
