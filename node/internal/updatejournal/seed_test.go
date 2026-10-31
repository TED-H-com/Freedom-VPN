package updatejournal

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestD089_SeedAtomicReplayAndBusy(t *testing.T) {
	j, dir := openTest(t)
	ctx := context.Background()
	r := Release{ReleaseID: "baseline", ManifestSHA256: "manifest", ManifestEnvelope: []byte(`{}`), ArtifactSHA256: "artifact", TreeHash: "tree", StateSchemaTo: 1}
	// Fail the final insert after the release and pointer writes. SQLite must
	// roll back all of them; an interrupted seed cannot leave partial evidence.
	_, err := j.db.ExecContext(ctx, `CREATE TRIGGER fail_seed BEFORE INSERT ON meta
		WHEN NEW.key LIKE 'seeded_baseline/%' BEGIN SELECT RAISE(ABORT, 'TEST ONLY seed fault'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.SeedBaseline(ctx, r, "boot"); err == nil {
		t.Fatal("injected seed fault passed")
	}
	p, err := j.Pointers(ctx)
	if err != nil || p.CurrentRelease != "" {
		t.Fatalf("partial pointer: %+v %v", p, err)
	}
	if _, err := j.GetRelease(ctx, r.ReleaseID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial release: %v", err)
	}
	if _, err := j.db.ExecContext(ctx, `DROP TRIGGER fail_seed`); err != nil {
		t.Fatal(err)
	}
	if err := j.SetGate(ctx, GateRecord{PublicationIntent: "disabled", ForwardingBlocked: true, TerminalOutcome: OutcomeRecoveryRequired}); err != nil {
		t.Fatal(err)
	}
	gate, _ := j.GateRecord(ctx)
	for range 2 {
		if err := j.SeedBaseline(ctx, r, "boot"); err != nil {
			t.Fatal(err)
		}
	}
	var seeds, releases int
	if err := j.db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM meta WHERE key LIKE 'seeded_baseline/%'), (SELECT count(*) FROM releases)`).Scan(&seeds, &releases); err != nil || seeds != 1 || releases != 1 {
		t.Fatalf("rows seeds=%d releases=%d: %v", seeds, releases, err)
	}
	if got, _ := j.GateRecord(ctx); !reflect.DeepEqual(gate, got) {
		t.Fatal("seed changed serving/recovery authorization")
	}
	snapshot, err := Snapshot(ctx, filepath.Join(dir, FileName))
	if err != nil || snapshot.SeededBaseline == nil || !snapshot.RecoveryRequired || snapshot.LastGoodRelease != "" || snapshot.Health != nil {
		t.Fatalf("seed invented health: %+v %v", snapshot, err)
	}
	if _, err := j.RecordJob(ctx, Job{JobID: "pending", Attempt: 1, Generation: 1, ReleaseID: "next", ManifestSHA256: "next", IntentJSON: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := j.SeedBaseline(ctx, r, "later-boot"); !errors.Is(err, ErrActiveTransaction) {
		t.Fatalf("seed ignored open work: %v", err)
	}
}
