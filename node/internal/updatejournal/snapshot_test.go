package updatejournal

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func snapshotJob(t *testing.T, j *Journal, generation decstr.U64, id string) {
	t.Helper()
	if _, _, err := j.ClaimJob(context.Background(), Job{JobID: id, Attempt: 1, Generation: generation,
		ReleaseID: "candidate", ManifestSHA256: "aa", IntentJSON: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
}

func TestD062_SnapshotDuringWriteTransaction(t *testing.T) {
	j, dir := openTest(t)
	ctx := context.Background()
	snapshotJob(t, j, decstr.MustParse("18446744073709551615"), "job")
	if err := j.SetPointers(ctx, Pointers{CurrentRelease: "old", LastGood: "old"}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "txn", JobID: "job", Attempt: 1,
		Generation: decstr.MustParse("18446744073709551615"), ReleaseID: "candidate", ManifestSHA256: "aa"}); err != nil {
		t.Fatal(err)
	}
	// Hold an uncommitted writer open on the updater's own connection. The
	// observer must read the committed WAL snapshot without waiting for it.
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := setMeta(ctx, tx, metaCurrentRelease, "uncommitted"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE txns SET stage = 'recovery_required'`); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	s, err := Snapshot(ctx, filepath.Join(dir, FileName))
	if err != nil || time.Since(start) >= SnapshotTimeout {
		t.Fatalf("snapshot blocked on writer: %v", err)
	}
	if s.Generation == nil || s.Generation.String() != "18446744073709551615" || s.JobID != "job" ||
		s.Stage != "prepare" || s.State != "applying" || s.JournalStage != StagePrepared ||
		s.CurrentRelease != "old" || s.LastGoodRelease != "old" || s.RecoveryRequired {
		t.Fatalf("inconsistent snapshot: %+v", s)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	s, err = Snapshot(ctx, filepath.Join(dir, FileName))
	if err != nil || !s.RecoveryRequired || s.CurrentRelease != "uncommitted" || s.Stage != "rollback" || s.State != "failed" {
		t.Fatalf("committed snapshot: %+v %v", s, err)
	}
}

func TestD062_SnapshotResultsAndRejections(t *testing.T) {
	j, dir := openTest(t)
	ctx := context.Background()
	snapshotJob(t, j, 9, "failed-job")
	entry := OutboxEntry{ResultID: "result", JobID: "failed-job", Attempt: 1, Generation: 9,
		State: "failed", PayloadHash: "hash", Payload: []byte(`{"stage":"verify","health":null}`)}
	if err := j.Enqueue(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordDeliveryFailure(ctx, entry.ResultID, DeliveryFailure{
		Status: OutboxRejected, HTTPStatus: 422, APICode: "VALIDATION_FAILED", ErrorCode: "VALIDATION_FAILED",
	}); err != nil {
		t.Fatal(err)
	}
	s, err := Snapshot(ctx, filepath.Join(dir, FileName))
	if err != nil || s.Generation == nil || *s.Generation != 9 || s.Stage != "verify" || s.State != "failed" ||
		s.LastResultState != "failed" || s.Health != nil || s.LastRejection == nil || s.LastRejection.HTTPStatus != 422 ||
		s.LastRejection.APICode != "VALIDATION_FAILED" || s.LastRejection.At.IsZero() {
		t.Fatalf("failure snapshot: %+v %v", s, err)
	}
	// A newer generation's open job must not be hidden by an older result.
	snapshotJob(t, j, 10, "new-job")
	if err := j.SetJobPhase(ctx, "new-job", 1, JobDownloading); err != nil {
		t.Fatal(err)
	}
	s, err = Snapshot(ctx, filepath.Join(dir, FileName))
	if err != nil || s.Generation == nil || *s.Generation != 10 || s.JobID != "new-job" || s.State != "downloading" ||
		s.Stage != "download" || s.LastResultState != "failed" || s.LastRejection == nil {
		t.Fatalf("new job snapshot: %+v %v", s, err)
	}
	// ACK history retains a pre-health stage after removing the payload.
	entry.ResultID, entry.JobID, entry.Generation = "acked", "new-job", 10
	if err := j.Enqueue(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := j.AckResult(ctx, Ack{ResultID: entry.ResultID, PayloadHash: entry.PayloadHash, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	s, err = Snapshot(ctx, filepath.Join(dir, FileName))
	if err != nil || s.Stage != "verify" || s.State != "failed" || s.LastResultState != "failed" || s.LastRejection == nil {
		t.Fatalf("acknowledged snapshot: %+v %v", s, err)
	}
}

func TestD062_SnapshotNeverCreatesOrMigrates(t *testing.T) {
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing", FileName)
	if _, err := Snapshot(ctx, missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("snapshot created a missing directory", err)
	}
	j, dir := openTest(t)
	if _, err := j.db.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(ctx, filepath.Join(dir, FileName)); err == nil {
		t.Fatal("unsupported schema accepted")
	}
	var version int
	if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 99 {
		t.Fatal("snapshot changed the schema", version, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Snapshot(canceled, filepath.Join(dir, FileName)); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled snapshot: %v", err)
	}
}

func TestD062_MigrateV1PreservesCandidateAndPendingPayload(t *testing.T) {
	for _, recordedRollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "no_rollback_evidence", true: "recorded_rollback"}[recordedRollback], func(t *testing.T) {
			j, dir := openTest(t)
			ctx := context.Background()
			snapshotJob(t, j, 1, "job")
			if _, err := j.BeginTxn(ctx, Txn{TxnID: "txn", JobID: "job", Attempt: 1, Generation: 1,
				ReleaseID: "candidate", ManifestSHA256: "aa"}); err != nil {
				t.Fatal(err)
			}
			candidate := `{"tunnel_probe":"failed"}`
			restored := `{"tunnel_probe":"passed"}`
			if _, err := j.Advance(ctx, "txn", StageRollingBack, StageUpdate{Destructive: true}); err != nil {
				t.Fatal(err)
			}
			if recordedRollback {
				if err := j.RecordEvent(ctx, "txn", `{"kind":"health_attest","phase":"rollback","statuses":`+restored+`}`); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := j.Advance(ctx, "txn", StageRolledBack, StageUpdate{Health: json.RawMessage(candidate)}); err != nil {
				t.Fatal(err)
			}
			entry := OutboxEntry{ResultID: "sealed", JobID: "job", Attempt: 1, Generation: 1, State: "rolled_back",
				PayloadHash: "immutable-hash", Payload: []byte(`{"stage":"report","health":` + candidate + `}`)}
			if err := j.Enqueue(ctx, entry); err != nil {
				t.Fatal(err)
			}
			// Reproduce the complete v1 layout without the v2 columns.
			for _, stmt := range []string{
				`ALTER TABLE txns DROP COLUMN candidate_health_json`,
				`ALTER TABLE result_outbox DROP COLUMN last_http_status`,
				`ALTER TABLE result_outbox DROP COLUMN last_api_code`,
				`ALTER TABLE result_outbox DROP COLUMN retry_not_before`,
				`ALTER TABLE result_history DROP COLUMN stage`,
				`PRAGMA user_version = 1`,
			} {
				if _, err := j.db.ExecContext(ctx, stmt); err != nil {
					t.Fatal(err)
				}
			}
			s, err := Snapshot(ctx, filepath.Join(dir, FileName))
			if err != nil || string(s.CandidateHealth) != candidate || s.Health != nil || s.State != "rolled_back" {
				t.Fatalf("v1 read-only evidence: %+v %v", s, err)
			}
			var version int
			if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 1 {
				t.Fatal("snapshot migrated v1", version, err)
			}
			if err := j.Close(); err != nil {
				t.Fatal(err)
			}
			j, err = Open(dir, Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			txn, err := j.GetTxn(ctx, "txn")
			if err != nil || string(txn.CandidateHealth) != candidate {
				t.Fatalf("migrated candidate: %+v %v", txn, err)
			}
			want := ""
			if recordedRollback {
				want = restored
			}
			if string(txn.Health) != want {
				t.Fatalf("invented/lost restored health: %s", txn.Health)
			}
			out, err := j.Outbox(ctx)
			if err != nil || len(out) != 1 || string(out[0].Payload) != string(entry.Payload) || out[0].PayloadHash != entry.PayloadHash || out[0].ResultID != entry.ResultID {
				t.Fatalf("sealed v1 evidence rewritten: %+v %v", out, err)
			}
		})
	}
}
