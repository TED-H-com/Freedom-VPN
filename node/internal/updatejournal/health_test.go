package updatejournal

import (
	"context"
	"encoding/json"
	"testing"
)

func TestD062_CandidateHealthFollowsResumedCheckButNotRollback(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "txn", JobID: "job", Attempt: 1, Generation: 1, ReleaseID: "candidate", ManifestSHA256: "aa"}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []Stage{StageBlocked, StageFinalSampled, StageStopped, StageMigrated, StageSwitched, StageChecking} {
		if _, err := j.Advance(ctx, "txn", s, StageUpdate{}); err != nil {
			t.Fatal(err)
		}
	}
	// A crash can occur after recording health but before committing the next
	// stage. The resumed challenge, not the earlier answer, decides rollback.
	passed := json.RawMessage(`{"tunnel_probe":"passed"}`)
	failed := json.RawMessage(`{"tunnel_probe":"failed"}`)
	if err := j.RecordHealth(ctx, "txn", passed); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "txn", StageChecking, StageUpdate{}); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordHealth(ctx, "txn", failed); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "txn", StageRollingBack, StageUpdate{Health: failed}); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordHealth(ctx, "txn", passed); err != nil {
		t.Fatal(err)
	}
	txn, err := j.Advance(ctx, "txn", StageRolledBack, StageUpdate{Health: passed})
	if err != nil || string(txn.Health) != string(passed) || string(txn.CandidateHealth) != string(failed) {
		t.Fatalf("wrong candidate/rollback health: %+v %v", txn, err)
	}
}
