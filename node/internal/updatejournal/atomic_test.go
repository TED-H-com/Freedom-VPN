package updatejournal

import (
	"context"
	"testing"
)

func TestUPD_D053_StagePointersAndSchemaCommitTogether(t *testing.T) {
	for _, terminal := range []Stage{StageLocallySucceeded, StageRolledBack} {
		t.Run(string(terminal), func(t *testing.T) {
			ctx := context.Background()
			j, dir := openTest(t)
			if err := j.SetPointers(ctx, Pointers{CurrentRelease: "old", LastGood: "old"}); err != nil {
				t.Fatal(err)
			}
			if err := j.SetStateSchema(ctx, 1); err != nil {
				t.Fatal(err)
			}
			if _, err := j.BeginTxn(ctx, Txn{TxnID: "txn", JobID: "job", Attempt: 1, Generation: 1, ReleaseID: "new"}); err != nil {
				t.Fatal(err)
			}
			stages := []Stage{StageBlocked, StageFinalSampled, StageStopped, StageMigrated, StageSwitched, StageChecking}
			if terminal == StageRolledBack {
				stages = append(stages, StageRollingBack)
			}
			for _, stage := range stages {
				if _, err := j.Advance(ctx, "txn", stage, StageUpdate{}); err != nil {
					t.Fatal(err)
				}
			}
			before := stages[len(stages)-1]
			// Abort the final metadata write, after stage, event, gate and
			// pointer writes have executed within Advance's transaction.
			if _, err := j.db.ExecContext(ctx, `CREATE TRIGGER fail_schema BEFORE INSERT ON meta
				WHEN NEW.key = 'state_schema_version' BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
				t.Fatal(err)
			}
			u := StageUpdate{Pointers: Pointers{CurrentRelease: "new", LastGood: "new"}, StateSchema: 2}
			if _, err := j.Advance(ctx, "txn", terminal, u); err == nil {
				t.Fatal("expected injected write failure")
			}
			assert := func(want Stage, release string, schema int) {
				t.Helper()
				// A second connection observes only the durable committed state.
				reader, err := Open(dir, Options{})
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				txn, err := reader.GetTxn(ctx, "txn")
				if err != nil || txn.Stage != want {
					t.Fatalf("stage=%s err=%v, want %s", txn.Stage, err, want)
				}
				p, err := reader.Pointers(ctx)
				if err != nil || p.CurrentRelease != release || p.LastGood != release {
					t.Fatalf("pointers=%+v err=%v", p, err)
				}
				s, err := reader.StateSchema(ctx)
				if err != nil || s != schema {
					t.Fatalf("schema=%d err=%v, want %d", s, err, schema)
				}
				events, err := reader.TxnEvents(ctx, "txn")
				if err != nil || events[len(events)-1].Stage != want {
					t.Fatalf("events=%+v err=%v", events, err)
				}
			}
			assert(before, "old", 1)
			if _, err := j.db.ExecContext(ctx, `DROP TRIGGER fail_schema`); err != nil {
				t.Fatal(err)
			}
			if _, err := j.Advance(ctx, "txn", terminal, u); err != nil {
				t.Fatal(err)
			}
			assert(terminal, "new", 2)
		})
	}
}
