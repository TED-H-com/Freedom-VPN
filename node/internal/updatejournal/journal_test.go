package updatejournal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func openTest(t *testing.T) (*Journal, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "update-journal")
	j, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	return j, dir
}

func TestStageTransitionsTable(t *testing.T) {
	forward := []Stage{StagePrepared, StageBlocked, StageFinalSampled, StageStopped, StageMigrated, StageSwitched, StageChecking, StageLocallySucceeded}
	for i := 0; i+1 < len(forward); i++ {
		if !CanTransition(forward[i], forward[i+1]) {
			t.Errorf("%s -> %s must be allowed", forward[i], forward[i+1])
		}
	}
	cases := []struct {
		from, to Stage
		ok       bool
	}{
		{StagePrepared, StageSwitched, false}, // skipping the block, sample and stop is never allowed
		{StageBlocked, StageStopped, false},   // no stop without final sample (§13.6 step 8)
		{StageFinalSampled, StageSwitched, false},
		{StageStopped, StageChecking, false},
		{StageChecking, StageChecking, true},
		{StageChecking, StageRolledBack, false}, // a rollback is journaled first
		{StageRollingBack, StageRolledBack, true},
		{StageRollingBack, StageRollingBack, true},
		{StageRollingBack, StageLocallySucceeded, false},
		{StageLocallySucceeded, StageRollingBack, false}, // terminal
		{StageRolledBack, StagePrepared, false},
		{StageRecoveryRequired, StageRollingBack, false},
		{StagePrepared, Stage("bogus"), false},
	}
	for _, s := range AllStages {
		if s.Terminal() {
			continue
		}
		cases = append(cases, struct {
			from, to Stage
			ok       bool
		}{s, StageRecoveryRequired, true}, struct {
			from, to Stage
			ok       bool
		}{s, StageRollingBack, true})
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.ok {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestStagePhase(t *testing.T) {
	want := map[Stage]Phase{
		StagePrepared: PhaseApplying, StageBlocked: PhaseApplying, StageFinalSampled: PhaseApplying,
		StageStopped: PhaseApplying, StageMigrated: PhaseApplying, StageSwitched: PhaseApplying,
		StageChecking: PhaseChecking, StageRollingBack: PhaseApplying,
		StageLocallySucceeded: PhaseDone, StageRolledBack: PhaseDone, StageRecoveryRequired: PhaseDone,
	}
	for s, p := range want {
		if s.Phase() != p {
			t.Errorf("%s phase = %s, want %s", s, s.Phase(), p)
		}
	}
}

func TestUPD_ClaimJobGenerationMonotonic(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	job := Job{JobID: "j7", Attempt: 1, Generation: 7, ReleaseID: "r1", ManifestSHA256: "aa", IntentJSON: []byte(`{}`)}
	if _, created, err := j.ClaimJob(ctx, job); err != nil || !created {
		t.Fatalf("claim: created=%v err=%v", created, err)
	}
	if _, created, err := j.ClaimJob(ctx, job); err != nil || created {
		t.Fatalf("idempotent re-claim: created=%v err=%v", created, err)
	}
	// A new attempt of the same job and generation is accepted.
	job2 := job
	job2.Attempt = 2
	if _, created, err := j.ClaimJob(ctx, job2); err != nil || !created {
		t.Fatalf("new attempt: created=%v err=%v", created, err)
	}
	// Same job and attempt with a higher generation must not silently keep
	// the old claim or move the accepted generation (§13.8).
	bumped := job
	bumped.Generation = 9
	if _, _, err := j.ClaimJob(ctx, bumped); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("same attempt, new generation: %v", err)
	}
	// Same generation, different release: stale (no silent replacement).
	bad := job
	bad.JobID, bad.ReleaseID = "jX", "r2"
	if _, _, err := j.ClaimJob(ctx, bad); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("same generation other job: %v", err)
	}
	// Older generation: refused (no silent downgrade, §13.4).
	old := Job{JobID: "j6", Attempt: 1, Generation: 6, ReleaseID: "r0", ManifestSHA256: "bb", IntentJSON: []byte(`{}`)}
	if _, _, err := j.ClaimJob(ctx, old); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("older generation: %v", err)
	}
	newer := Job{JobID: "j8", Attempt: 1, Generation: 8, ReleaseID: "r2", ManifestSHA256: "cc", IntentJSON: []byte(`{}`)}
	if _, _, err := j.ClaimJob(ctx, newer); err != nil {
		t.Fatal(err)
	}
	g, _, ok, err := j.AcceptedGeneration(ctx)
	if err != nil || !ok || g != 8 {
		t.Fatalf("accepted generation = %v %v %v", g, ok, err)
	}
	// Generations near 2^64 keep their exact value (decimal strings, §10.4).
	huge := Job{JobID: "jh", Attempt: 1, Generation: decstr.MustParse("18446744073709551615"), ReleaseID: "r3", ManifestSHA256: "dd", IntentJSON: []byte(`{}`)}
	if _, _, err := j.ClaimJob(ctx, huge); err != nil {
		t.Fatal(err)
	}
	g, _, _, _ = j.AcceptedGeneration(ctx)
	if g.String() != "18446744073709551615" {
		t.Fatalf("huge generation = %s", g)
	}
}

func TestJobPhases(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	job := Job{JobID: "j1", Attempt: 1, Generation: 1, ReleaseID: "r1", ManifestSHA256: "aa", IntentJSON: []byte(`{}`)}
	if _, _, err := j.ClaimJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := j.SetJobPhase(ctx, "j1", 1, JobDownloading); err != nil {
		t.Fatal(err)
	}
	open, err := j.OpenJobs(ctx)
	if err != nil || len(open) != 1 || open[0].Phase != JobDownloading {
		t.Fatalf("open jobs = %+v %v", open, err)
	}
	if err := j.FinishJob(ctx, "j1", 1, "superseded", "STALE_GENERATION"); err != nil {
		t.Fatal(err)
	}
	if err := j.SetJobPhase(ctx, "j1", 1, JobApplying); !errors.Is(err, ErrNotFound) {
		t.Fatalf("done job reopened: %v", err)
	}
	got, err := j.GetJob(ctx, "j1", 1)
	if err != nil || got.Phase != JobDone || got.Outcome != "superseded" || got.ErrorCode != "STALE_GENERATION" {
		t.Fatalf("job = %+v %v", got, err)
	}
}

func TestTxnLifecycleAndBlocksForwarding(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	if b, _ := j.BlocksForwarding(ctx); b {
		t.Fatal("empty journal must not block")
	}
	txn, err := j.BeginTxn(ctx, Txn{TxnID: "t1", JobID: "j1", Attempt: 1, Generation: 7, ReleaseID: "new", ManifestSHA256: "aa",
		PreviousReleaseID: "old", LastGoodReleaseID: "old", StateSchemaFrom: 1, StateSchemaTo: 1})
	if err != nil {
		t.Fatal(err)
	}
	if txn.Stage != StagePrepared {
		t.Fatalf("stage = %s", txn.Stage)
	}
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "t2", JobID: "j2", Attempt: 1, Generation: 8, ReleaseID: "x", ManifestSHA256: "bb"}); !errors.Is(err, ErrActiveTransaction) {
		t.Fatalf("second active txn: %v", err)
	}
	if b, why := j.BlocksForwarding(ctx); !b || why == "" {
		t.Fatal("prepared txn must block forwarding")
	}
	if _, err := j.Advance(ctx, "t1", StageSwitched, StageUpdate{}); !errors.Is(err, ErrBadTransition) {
		t.Fatalf("skip stages: %v", err)
	}
	for _, s := range []Stage{StageBlocked, StageFinalSampled, StageStopped, StageMigrated, StageSwitched, StageChecking} {
		u := StageUpdate{Detail: string(s)}
		if s == StageFinalSampled {
			u.Checkpoint = map[string]string{"last_durable_event_seq": "12045"}
		}
		if s == StageStopped {
			u.Destructive = true
		}
		if txn, err = j.Advance(ctx, "t1", s, u); err != nil {
			t.Fatalf("advance %s: %v", s, err)
		}
	}
	if txn, err = j.Advance(ctx, "t1", StageChecking, StageUpdate{}); err != nil {
		t.Fatal(err)
	}
	if txn.CheckAttempts != 2 || !txn.Destructive || string(txn.Checkpoint) != `{"last_durable_event_seq":"12045"}` {
		t.Fatalf("txn = %+v", txn)
	}
	if txn, err = j.Advance(ctx, "t1", StageRollingBack, StageUpdate{ErrorCode: "HEALTH_CHECK_FAILED", ErrorStage: "check"}); err != nil {
		t.Fatal(err)
	}
	if txn, err = j.Advance(ctx, "t1", StageRecoveryRequired, StageUpdate{ErrorCode: "ROLLBACK_FAILED"}); err != nil {
		t.Fatal(err)
	}
	if txn.ErrorCode != "ROLLBACK_FAILED" || txn.FinishedAt.IsZero() {
		t.Fatalf("txn = %+v", txn)
	}
	if b, why := j.BlocksForwarding(ctx); !b || why != "update recovery_required" {
		t.Fatalf("recovery_required must block: %v %q", b, why)
	}
	ev, err := j.TxnEvents(ctx, "t1")
	if err != nil || len(ev) != 10 || ev[0].Stage != StagePrepared || ev[9].Stage != StageRecoveryRequired {
		t.Fatalf("events = %+v %v", ev, err)
	}
	// A later successful transaction clears the block.
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "t2", JobID: "j2", Attempt: 1, Generation: 8, ReleaseID: "new2", ManifestSHA256: "bb"}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []Stage{StageBlocked, StageFinalSampled, StageStopped, StageMigrated, StageSwitched, StageChecking, StageLocallySucceeded} {
		if _, err := j.Advance(ctx, "t2", s, StageUpdate{}); err != nil {
			t.Fatal(err)
		}
	}
	if b, _ := j.BlocksForwarding(ctx); b {
		t.Fatal("locally_succeeded must not block")
	}
	if _, err := j.ActiveTxn(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("active txn: %v", err)
	}
}

func TestJournalSurvivesReopen(t *testing.T) {
	j, dir := openTest(t)
	ctx := context.Background()
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "t1", JobID: "j1", Attempt: 1, Generation: 3, ReleaseID: "r", ManifestSHA256: "aa"}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "t1", StageBlocked, StageUpdate{}); err != nil {
		t.Fatal(err)
	}
	if err := j.SetPointers(ctx, Pointers{CurrentRelease: "old", LastGood: "old"}); err != nil {
		t.Fatal(err)
	}
	_ = j.Close()
	j2, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer j2.Close()
	a, err := j2.ActiveTxn(ctx)
	if err != nil || a.Stage != StageBlocked {
		t.Fatalf("active after reopen = %+v %v", a, err)
	}
	p, _ := j2.Pointers(ctx)
	if p.CurrentRelease != "old" || p.LastGood != "old" {
		t.Fatalf("pointers = %+v", p)
	}
	fi, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("journal file mode = %v %v", fi.Mode(), err)
	}
}

func TestJournalNewerSchemaRefused(t *testing.T) {
	j, dir := openTest(t)
	if _, err := j.db.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatal(err)
	}
	_ = j.Close()
	if _, err := Open(dir, Options{}); err == nil {
		t.Fatal("a journal written by a newer updater must not be opened")
	}
}

func TestUPD_ResultOutboxAckAndLateDuplicate(t *testing.T) {
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	dir := filepath.Join(t.TempDir(), "j")
	j, err := Open(dir, Options{Now: func() time.Time { now = now.Add(time.Second); return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	ctx := context.Background()
	e := OutboxEntry{ResultID: "r1", JobID: "j1", Attempt: 1, Generation: 7, State: "succeeded", Payload: []byte(`{"a":1}`), PayloadHash: "h1"}
	if err := j.Enqueue(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := j.Enqueue(ctx, e); err != nil {
		t.Fatalf("same result enqueued twice: %v", err)
	}
	e2 := e
	e2.PayloadHash = "other"
	if err := j.Enqueue(ctx, e2); !errors.Is(err, ErrResultConflict) {
		t.Fatalf("same id, different hash: %v", err)
	}
	if err := j.Enqueue(ctx, OutboxEntry{ResultID: "r2", JobID: "j1", Attempt: 1, Generation: 7, State: "checking", Payload: []byte(`{}`), PayloadHash: "h2"}); err != nil {
		t.Fatal(err)
	}
	p, err := j.Pending(ctx, 10)
	if err != nil || len(p) != 2 || p[0].ResultID != "r1" {
		t.Fatalf("pending = %+v %v", p, err)
	}
	if err := j.RecordSendFailure(ctx, "r1", "STORE_UNAVAILABLE"); err != nil {
		t.Fatal(err)
	}
	// An ACK answering a different payload is a conflict and changes nothing.
	if _, err := j.AckResult(ctx, Ack{ResultID: "r1", PayloadHash: "nope"}); !errors.Is(err, ErrResultConflict) {
		t.Fatalf("mismatched ack: %v", err)
	}
	dup, err := j.AckResult(ctx, Ack{ResultID: "r1", PayloadHash: "h1", Raw: []byte(`{"accepted":true}`)})
	if err != nil || dup {
		t.Fatalf("ack: dup=%v err=%v", dup, err)
	}
	// Late duplicate ACK (§13.8): no-op, still acknowledged.
	dup, err = j.AckResult(ctx, Ack{ResultID: "r1", PayloadHash: "h1"})
	if err != nil || !dup {
		t.Fatalf("late duplicate ack: dup=%v err=%v", dup, err)
	}
	// Enqueueing an acknowledged result again is a no-op, not a resend.
	if err := j.Enqueue(ctx, e); err != nil {
		t.Fatal(err)
	}
	p, _ = j.Pending(ctx, 10)
	if len(p) != 1 || p[0].ResultID != "r2" {
		t.Fatalf("pending after ack = %+v", p)
	}
	if err := j.MarkConflict(ctx, "r2", "ACK_CONFLICT"); err != nil {
		t.Fatal(err)
	}
	p, _ = j.Pending(ctx, 10)
	if len(p) != 0 {
		t.Fatalf("conflict entries are never resent: %+v", p)
	}
	all, _ := j.Outbox(ctx)
	if len(all) != 1 || all[0].Status != OutboxConflict {
		t.Fatalf("conflict evidence kept: %+v", all)
	}
	h, err := j.History(ctx, "j1")
	if err != nil || len(h) != 1 || h[0].ResultID != "r1" {
		t.Fatalf("history = %+v %v", h, err)
	}
	if _, err := j.AckResult(ctx, Ack{ResultID: "unknown"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown ack: %v", err)
	}
}

func TestReleasesImmutableAndRetention(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	r := Release{ReleaseID: "r1", ManifestSHA256: "m", ManifestEnvelope: []byte(`{}`), ArtifactSHA256: "a", TreeHash: "t", StateSchemaTo: 1}
	if err := j.PutRelease(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := j.PutRelease(ctx, r); err != nil {
		t.Fatalf("identical re-record: %v", err)
	}
	r2 := r
	r2.TreeHash = "other"
	if err := j.PutRelease(ctx, r2); err == nil {
		t.Fatal("a release_id must be immutable (§13.3)")
	}
	if err := j.SetPointers(ctx, Pointers{CurrentRelease: "r1", LastGood: "r1"}); err != nil {
		t.Fatal(err)
	}
	if err := j.DeleteRelease(ctx, "r1"); err == nil {
		t.Fatal("current/last_good must be retained")
	}
	got, err := j.GetRelease(ctx, "r1")
	if err != nil || got.TreeHash != "t" {
		t.Fatalf("release = %+v %v", got, err)
	}
	if err := j.PutArtifact(ctx, Artifact{SHA256: "s", ReleaseID: "r1", Path: "/x", Size: 3}); err != nil {
		t.Fatal(err)
	}
	a, err := j.GetArtifact(ctx, "s")
	if err != nil || a.Size != 3 {
		t.Fatalf("artifact = %+v %v", a, err)
	}
	if err := j.DeleteArtifact(ctx, "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := j.GetArtifact(ctx, "s"); !errors.Is(err, ErrNotFound) {
		t.Fatal("artifact not deleted")
	}
	if err := j.SetStateSchema(ctx, 3); err != nil {
		t.Fatal(err)
	}
	if v, _ := j.StateSchema(ctx); v != 3 {
		t.Fatalf("state schema = %d", v)
	}
	if err := j.SaveDesiredState(ctx, []byte(`{"x":1}`)); err != nil {
		t.Fatal(err)
	}
	if ds, _ := j.DesiredState(ctx); string(ds) != `{"x":1}` {
		t.Fatalf("desired state = %s", ds)
	}
}

func TestJournalHealthStaysWithItsReasons(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	if _, err := j.BeginTxn(ctx, Txn{TxnID: "t1", JobID: "j1", Attempt: 1, Generation: 1, ReleaseID: "r", ManifestSHA256: "aa"}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []Stage{StageBlocked, StageFinalSampled, StageStopped, StageMigrated, StageSwitched, StageChecking} {
		if _, err := j.Advance(ctx, "t1", s, StageUpdate{}); err != nil {
			t.Fatal(err)
		}
	}
	first := map[string]string{"processes": "passed", "internet_only": "failed"}
	if err := j.RecordHealth(ctx, "t1", first); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordEvent(ctx, "t1", `{"kind":"health_attest","reason":"probe refused"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "t1", StageRollingBack, StageUpdate{
		ErrorCode: "HEALTH_CHECK_FAILED",
	}); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordHealth(ctx, "t1", map[string]string{"processes": "passed", "internet_only": "passed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "t1", StageRecoveryRequired, StageUpdate{
		Health: map[string]string{"internet_only": "passed"}, ErrorCode: "ROLLBACK_FAILED",
	}); err != nil {
		t.Fatal(err)
	}
	txn, err := j.GetTxn(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if string(txn.CandidateHealth) != `{"internet_only":"failed","processes":"passed"}` {
		t.Fatalf("candidate health overwritten: %s", txn.CandidateHealth)
	}
	if string(txn.Health) != `{"internet_only":"passed"}` {
		t.Fatalf("latest health missing: %s", txn.Health)
	}
	ev, err := j.TxnEvents(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	var saw bool
	for _, e := range ev {
		if e.Detail == `{"kind":"health_attest","reason":"probe refused"}` && e.Stage == StageChecking {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("detail missing from %+v", ev)
	}
	if err := j.RecordEvent(ctx, "missing", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestRecordJobLeavesAcceptedGeneration(t *testing.T) {
	j, _ := openTest(t)
	ctx := context.Background()
	var version int
	if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != schemaVersion {
		t.Fatalf("user_version %d %v", version, err)
	}
	job := Job{
		JobID: "installer-job", Attempt: 1, Generation: decstr.MustParse("1"),
		ReleaseID: "rel", ManifestSHA256: "aa", IntentJSON: []byte(`{"source":"installer"}`),
	}
	stored, err := j.RecordJob(ctx, job)
	if err != nil || stored.Phase != JobClaimed {
		t.Fatalf("record %+v %v", stored.Phase, err)
	}
	if _, _, ok, err := j.AcceptedGeneration(ctx); err != nil || ok {
		t.Fatalf("accepted generation moved ok=%v %v", ok, err)
	}
	again, err := j.RecordJob(ctx, job)
	if err != nil || again.JobID != stored.JobID || again.Phase != JobClaimed {
		t.Fatalf("idempotent %+v %v", again, err)
	}
	job.ReleaseID = "other"
	if _, err := j.RecordJob(ctx, job); !errors.Is(err, ErrStaleGeneration) {
		t.Fatal(err)
	}
	if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != schemaVersion {
		t.Fatalf("user_version after record %d %v", version, err)
	}
}
