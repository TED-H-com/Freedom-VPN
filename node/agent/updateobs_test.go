package agent

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

const updateHealthToken = "journal-secret-token-not-for-logs"

func TestUpdaterJournalConfig(t *testing.T) {
	cfg, err := LoadConfig(writeAgentConfig(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(updatejournal.DefaultDir, updatejournal.FileName)
	if cfg.UpdaterJournal != want || cfg.JournalDir != updatejournal.DefaultDir {
		t.Fatalf("journal dir %q file %q", cfg.JournalDir, cfg.UpdaterJournal)
	}

	explicit := filepath.Join(t.TempDir(), "custom.db")
	cfg, err = LoadConfig(writeAgentConfig(t, `, "updater_journal": "`+explicit+`"`))
	if err != nil || cfg.UpdaterJournal != explicit {
		t.Fatal(cfg.UpdaterJournal, err)
	}

	for _, bad := range []string{
		`, "updater_journal": "journal.db"`,
		`, "updater_journal": "https://example.invalid/journal.db"`,
	} {
		_, err = LoadConfig(writeAgentConfig(t, bad))
		if !errors.Is(err, errUpdaterJournal) {
			t.Fatal(err)
		}
		if strings.Contains(err.Error(), "journal.db") || strings.Contains(err.Error(), "example.invalid") {
			t.Fatalf("path echoed: %v", err)
		}
	}
	cfg.UpdaterJournal = "/tmp/\x00journal.db"
	if err = cfg.validate(); !errors.Is(err, errUpdaterJournal) || strings.Contains(err.Error(), "journal.db") {
		t.Fatal(err)
	}
	cfg.UpdaterJournal = "/" + strings.Repeat("a", 4096)
	if err = cfg.validate(); !errors.Is(err, errUpdaterJournal) {
		t.Fatal(err)
	}
}

func TestUpdateObservation_NoJournal(t *testing.T) {
	r := newRig(t)
	rep := mustReport(t, r, context.Background())
	assertNullUpdate(t, rep)
	if got, err := rep.CurrentReleaseId.Get(); err != nil || got != r.eng.cfg.ReleaseUUID() {
		t.Fatalf("current release %v %v", got, err)
	}

	dir := filepath.Join(t.TempDir(), "empty-journal")
	path := withJournal(t, dir, func(*updatejournal.Journal) {})
	r.eng.cfg.UpdaterJournal = path
	rep = mustReport(t, r, context.Background())
	assertNullUpdate(t, rep)
	if got, err := rep.CurrentReleaseId.Get(); err != nil || got != r.eng.cfg.ReleaseUUID() {
		t.Fatalf("empty journal release %v %v", got, err)
	}
}

func TestUpdateObservation_LocallySucceeded(t *testing.T) {
	r := newRig(t)
	job := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	release := uuid.MustParse("019a0000-0000-7000-8000-0000000000b1")
	path := succeededJournal(t, filepath.Join(t.TempDir(), "update-journal"), job, decstr.U64(42), release)
	r.eng.cfg.UpdaterJournal = path

	rep := mustReport(t, r, context.Background())
	assertUpdate(t, rep, decstr.U64(42), job, api.UpdateStageReport, api.UpdateJobStateSucceeded, release, release)
	if got, err := rep.CurrentReleaseId.Get(); err != nil || got == r.eng.cfg.ReleaseUUID() {
		t.Fatalf("configured release hid the journal release: %v %v", got, err)
	}
}

func TestUpdateObservation_RolledBack(t *testing.T) {
	r := newRig(t)
	job := uuid.MustParse("019a0000-0000-7000-8000-0000000000a2")
	restored := uuid.MustParse("019a0000-0000-7000-8000-0000000000d1")
	dir := filepath.Join(t.TempDir(), "update-journal")
	path := withJournal(t, dir, func(j *updatejournal.Journal) {
		seedUpdate(t, j, job, 9, restored.String())
		ctx := context.Background()
		if _, err := j.Advance(ctx, "txn", updatejournal.StageRollingBack, updatejournal.StageUpdate{Destructive: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := j.Advance(ctx, "txn", updatejournal.StageRolledBack, updatejournal.StageUpdate{
			Destructive: true,
			Pointers:    updatejournal.Pointers{CurrentRelease: restored.String(), LastGood: restored.String()},
		}); err != nil {
			t.Fatal(err)
		}
	})
	r.eng.cfg.UpdaterJournal = path
	rep := mustReport(t, r, context.Background())
	assertUpdate(t, rep, 9, job, api.UpdateStageReport, api.UpdateJobStateRolledBack, restored, restored)
}

func TestUpdateObservation_RecoveryRequired(t *testing.T) {
	r := newRig(t)
	job := uuid.MustParse("019a0000-0000-7000-8000-0000000000a3")
	current := uuid.MustParse("019a0000-0000-7000-8000-0000000000b3")
	last := uuid.MustParse("019a0000-0000-7000-8000-0000000000c3")
	dir := filepath.Join(t.TempDir(), "update-journal")
	path := withJournal(t, dir, func(j *updatejournal.Journal) {
		seedUpdate(t, j, job, 11, current.String())
		if _, err := j.Advance(context.Background(), "txn", updatejournal.StageRecoveryRequired, updatejournal.StageUpdate{
			Pointers: updatejournal.Pointers{CurrentRelease: current.String(), LastGood: last.String()},
		}); err != nil {
			t.Fatal(err)
		}
	})
	r.eng.cfg.UpdaterJournal = path
	rep := mustReport(t, r, context.Background())
	// Snapshot maps journal stage recovery_required to API stage rollback and state failed.
	assertUpdate(t, rep, 11, job, api.UpdateStageRollback, api.UpdateJobStateFailed, r.eng.cfg.ReleaseUUID(), last)
}

func TestUpdateObservation_DeadlineDoesNotBlockOrReuse(t *testing.T) {
	r := newRig(t)
	job := uuid.MustParse("019a0000-0000-7000-8000-0000000000a4")
	release := uuid.MustParse("019a0000-0000-7000-8000-0000000000b4")
	dir := filepath.Join(t.TempDir(), "update-journal")
	path := succeededJournal(t, dir, job, 8, release)
	r.eng.cfg.UpdaterJournal = path

	rep := mustReport(t, r, context.Background())
	assertUpdate(t, rep, 8, job, api.UpdateStageReport, api.UpdateJobStateSucceeded, release, release)

	expired, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancel()
	for range 2 {
		start := time.Now()
		rep = mustReport(t, r, expired)
		if elapsed := time.Since(start); elapsed >= updatejournal.SnapshotTimeout {
			t.Fatalf("report blocked for %s", elapsed)
		}
		assertNullUpdate(t, rep)
		if got, err := rep.CurrentReleaseId.Get(); err != nil || got != r.eng.cfg.ReleaseUUID() || got == release {
			t.Fatalf("stale journal release %v %v", got, err)
		}
	}
	if n := strings.Count(r.log.String(), "reason=deadline"); n != 1 {
		t.Fatalf("deadline logs %d\n%s", n, r.log.String())
	}
	if strings.Contains(r.log.String(), updateHealthToken) || strings.Contains(r.log.String(), dir) {
		t.Fatalf("journal contents logged: %s", r.log.String())
	}

	rep = mustReport(t, r, context.Background())
	assertUpdate(t, rep, 8, job, api.UpdateStageReport, api.UpdateJobStateSucceeded, release, release)
}

func TestUpdateObservation_SnapshotFailureDoesNotFailObserve(t *testing.T) {
	r := newRig(t)
	r.eng.enrolled = true
	r.eng.nodeID = uuid.MustParse("019a0000-0000-7000-8000-0000000000e1")
	r.eng.incID = uuid.MustParse("019a0000-0000-7000-8000-0000000000e2")
	if err := r.eng.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	items, err := r.eng.store.Pending(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("outbox %d %v", len(items), err)
	}
	var rep api.ObservationReport
	if err := json.Unmarshal(items[0].Payload, &rep); err != nil {
		t.Fatal(err)
	}
	assertNullUpdate(t, rep)
	if bytes.Contains(items[0].Payload, []byte(`"generation":"0"`)) || bytes.Contains(items[0].Payload, []byte(`"update":null`)) {
		t.Fatalf("payload %s", items[0].Payload)
	}
}

func succeededJournal(t *testing.T, dir string, job uuid.UUID, gen decstr.U64, release uuid.UUID) string {
	t.Helper()
	return withJournal(t, dir, func(j *updatejournal.Journal) {
		seedUpdate(t, j, job, gen, release.String())
		if err := j.RecordHealth(context.Background(), "txn", map[string]string{"note": updateHealthToken}); err != nil {
			t.Fatal(err)
		}
		path := []updatejournal.Stage{
			updatejournal.StageBlocked,
			updatejournal.StageFinalSampled,
			updatejournal.StageStopped,
			updatejournal.StageMigrated,
			updatejournal.StageSwitched,
			updatejournal.StageChecking,
			updatejournal.StageLocallySucceeded,
		}
		for _, stage := range path {
			u := updatejournal.StageUpdate{}
			if stage == updatejournal.StageLocallySucceeded {
				u.Pointers = updatejournal.Pointers{CurrentRelease: release.String(), LastGood: release.String()}
			}
			if _, err := j.Advance(context.Background(), "txn", stage, u); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func withJournal(t *testing.T, dir string, fn func(*updatejournal.Journal)) string {
	t.Helper()
	j, err := updatejournal.Open(dir, updatejournal.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = j.Close() }()
	fn(j)
	return filepath.Join(dir, updatejournal.FileName)
}

func seedUpdate(t *testing.T, j *updatejournal.Journal, job uuid.UUID, gen decstr.U64, release string) {
	t.Helper()
	ctx := context.Background()
	if _, _, err := j.ClaimJob(ctx, updatejournal.Job{
		JobID: job.String(), Attempt: 1, Generation: gen,
		ReleaseID: release, ManifestSHA256: "aa", IntentJSON: []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.BeginTxn(ctx, updatejournal.Txn{
		TxnID: "txn", JobID: job.String(), Attempt: 1, Generation: gen,
		ReleaseID: release, ManifestSHA256: "aa",
	}); err != nil {
		t.Fatal(err)
	}
}

func mustReport(t *testing.T, r *rig, ctx context.Context) api.ObservationReport {
	t.Helper()
	rep, err := r.eng.report(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func assertNullUpdate(t *testing.T, rep api.ObservationReport) {
	t.Helper()
	for _, field := range []struct {
		name string
		err  error
	}{
		{"generation", genErr(rep)},
		{"job_id", idErr(rep.Update.JobId)},
		{"stage", idErr(rep.Update.Stage)},
		{"state", idErr(rep.Update.State)},
	} {
		if !errors.Is(field.err, presence.ErrNull) {
			t.Fatalf("%s: %v", field.name, field.err)
		}
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(`"update":null`)) || bytes.Contains(raw, []byte(`"generation":"0"`)) || bytes.Contains(raw, []byte(`"generation":0`)) {
		t.Fatalf("update %s", raw)
	}
}

func assertUpdate(t *testing.T, rep api.ObservationReport, gen decstr.U64, job uuid.UUID, stage api.UpdateStage, state api.UpdateJobState, current, last uuid.UUID) {
	t.Helper()
	gotGen, err := rep.Update.Generation.Get()
	if err != nil || gotGen != gen {
		t.Fatalf("generation %v %v", gotGen, err)
	}
	gotJob, err := rep.Update.JobId.Get()
	if err != nil || gotJob != job {
		t.Fatalf("job %v %v", gotJob, err)
	}
	gotStage, err := rep.Update.Stage.Get()
	if err != nil || gotStage != stage {
		t.Fatalf("stage %v %v", gotStage, err)
	}
	gotState, err := rep.Update.State.Get()
	if err != nil || gotState != state {
		t.Fatalf("state %v %v", gotState, err)
	}
	gotCurrent, err := rep.CurrentReleaseId.Get()
	if err != nil || gotCurrent != current {
		t.Fatalf("current %v %v", gotCurrent, err)
	}
	gotLast, err := rep.LastGoodReleaseId.Get()
	if err != nil || gotLast != last {
		t.Fatalf("last good %v %v", gotLast, err)
	}
}

func genErr(rep api.ObservationReport) error {
	_, err := rep.Update.Generation.Get()
	return err
}

func idErr[T any](n presence.Nullable[T]) error {
	_, err := n.Get()
	return err
}

func TestReviewD063_CompletedJournalNeedsValidJobID(t *testing.T) {
	r := newRig(t)
	job := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	release := uuid.MustParse("019a0000-0000-7000-8000-0000000000b1")
	path := succeededJournal(t, t.TempDir(), job, 42, release)
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{"UPDATE jobs SET job_id = 'invalid'", "UPDATE txns SET job_id = 'invalid'"} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	r.eng.cfg.UpdaterJournal = path
	rep := mustReport(t, r, context.Background())
	if got, err := rep.CurrentReleaseId.Get(); err != nil || got != r.eng.cfg.ReleaseUUID() {
		t.Fatalf("invalid job overrode running release: %v %v", got, err)
	}
	if _, err := rep.Update.JobId.Get(); !errors.Is(err, presence.ErrNull) {
		t.Fatalf("invalid job ID reported: %v", err)
	}
}
