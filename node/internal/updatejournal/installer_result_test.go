package updatejournal

import (
	"bytes"
	"testing"
)

func TestD114_ReplacementAtomicAndHistoryProtected(t *testing.T) {
	for _, mode := range []string{"commit", "abort", "acknowledged", "collision"} {
		t.Run(mode, func(t *testing.T) {
			j, _ := openTest(t)
			old := OutboxEntry{ResultID: "old-result", JobID: "old-job", Attempt: 1, Generation: 1, State: "failed", Payload: []byte(`{"test":"old"}`), PayloadHash: "old-hash"}
			next := old
			next.ResultID, next.JobID, next.PayloadHash = "new-result", "new-job", "new-hash"
			next.Payload = []byte(`{"test":"new"}`)
			if err := j.Enqueue(t.Context(), old); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "abort":
				if _, err := j.db.Exec(`CREATE TRIGGER fail_retirement BEFORE UPDATE ON result_outbox BEGIN SELECT RAISE(ABORT,'TEST ONLY failure'); END`); err != nil {
					t.Fatal(err)
				}
			case "acknowledged":
				if _, err := j.AckResult(t.Context(), Ack{ResultID: old.ResultID, PayloadHash: old.PayloadHash}); err != nil {
					t.Fatal(err)
				}
			case "collision":
				if err := j.Enqueue(t.Context(), next); err != nil {
					t.Fatal(err)
				}
			}
			_, err := j.ReplaceInstallerResult(t.Context(), old, next)
			if (err == nil) != (mode == "commit") {
				t.Fatalf("replacement: %v", err)
			}
			out, err := j.Outbox(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if mode == "acknowledged" {
				if len(out) != 0 {
					t.Fatal("rewrote acknowledged history")
				}
				history, err := j.History(t.Context(), old.JobID)
				if err != nil || len(history) != 1 || history[0].PayloadHash != old.PayloadHash {
					t.Fatal("history changed", err)
				}
				return
			}
			count := 1
			if mode == "commit" || mode == "collision" {
				count = 2
			}
			if len(out) != count || !bytes.Equal(out[0].Payload, old.Payload) {
				t.Fatal("partial or changed evidence", out)
			}
			want := OutboxPending
			if mode == "commit" {
				want = OutboxRejected
			}
			if out[0].Status != want {
				t.Fatal("incorrect retirement", out[0])
			}
			var links int
			if err := j.db.QueryRow(`SELECT count(*) FROM meta WHERE key LIKE 'installer_result_replacement/%'`).Scan(&links); err != nil {
				t.Fatal(err)
			}
			if (links == 1) != (mode == "commit") {
				t.Fatal("partial audit link", links)
			}
		})
	}
}
