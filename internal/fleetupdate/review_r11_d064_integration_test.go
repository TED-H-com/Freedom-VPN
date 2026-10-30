//go:build integration

package fleetupdate

import (
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func reviewOtherRelease(t *testing.T, l *lab) uuid.UUID {
	t.Helper()
	m := l.f.Manifest
	m.ReleaseId = uuid.Must(uuid.NewV7())
	m.ProductVersion = "1.3.0-review"
	m.Rollback.AllowedReleaseIds = []uuid.UUID{l.old}
	if _, err := l.s.releases.Register(l.ctx, releasetest.Command(), l.f.Request(t, m)); err != nil {
		t.Fatal(err)
	}
	return m.ReleaseId
}

func TestReviewR11_HeartbeatSeedsAnyReleaseUntilAResult(t *testing.T) {
	// Before the first accepted result, unrelated releases must not overwrite
	// the serving release, even though those releases are registered.
	l := newLabState(t)
	other := reviewOtherRelease(t, l)
	i := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	if l.scalar(`SELECT count(*) FROM update_jobs WHERE node_id=$1 AND release_id=$2`, i.NodeID, other) != 0 {
		t.Fatal("the extra release is a job target")
	}
	if err := l.s.Observe(l.ctx, i, Observation{
		Sequence: 2, AppliedUpdateGeneration: 0,
		CurrentReleaseID: &other, LastGoodReleaseID: &other,
		ObservedAt: time.Now().UTC().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var current, lastGood uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT current_release_id, last_good_release_id FROM nodes WHERE id=$1`, i.NodeID).Scan(&current, &lastGood); err != nil {
		t.Fatal(err)
	}
	again := l.old
	if err := l.s.Observe(l.ctx, i, Observation{
		Sequence: 3, AppliedUpdateGeneration: 0,
		CurrentReleaseID: &again, LastGoodReleaseID: &again,
		ObservedAt: time.Now().UTC().Add(2 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var moved uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT current_release_id FROM nodes WHERE id=$1`, i.NodeID).Scan(&moved); err != nil {
		t.Fatal(err)
	}
	if current == other && lastGood == other && moved == again && l.scalar(`SELECT count(*) FROM update_results WHERE node_id=$1 AND ack->>'accepted'='true'`, i.NodeID) == 0 {
		t.Fatalf("with no accepted result, heartbeat 2 set current_release_id and last_good_release_id to %s, which is not job target %s, and heartbeat 3 moved current_release_id to %s", other, l.new, again)
	}
	if current != l.old || lastGood != l.old || moved != again {
		t.Fatalf("serving release changed: after other heartbeat current %s last %s, after second heartbeat current %s", current, lastGood, moved)
	}
}

func TestReviewR11_HeartbeatConfirmationFlapsAroundAcceptedRelease(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, result(w, "succeeded"))); err != nil {
		t.Fatal(err)
	}
	report, svc := reviewObservation(t, l, i)
	report.Update.JobId.Set(w.Intent.JobId)
	report.Update.Generation.Set(w.Intent.Generation)
	report.Update.State.Set(api.UpdateJobStateSucceeded)
	report.Update.Stage.Set(api.UpdateStageReport)
	flip := func(agree bool) bool {
		t.Helper()
		report.ReportId = uuid.New()
		report.ObservedAt = report.ObservedAt.Add(time.Second)
		if agree {
			report.CurrentReleaseId.Set(l.new)
			report.LastGoodReleaseId.Set(l.new)
		} else {
			report.CurrentReleaseId.Set(l.old)
			report.LastGoodReleaseId.Set(l.old)
		}
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
			t.Fatal(err)
		}
		return l.get(b.BatchId).Targets[0].Unconfirmed
	}
	disagree := flip(false)
	agree := flip(true)
	disagreeAgain := flip(false)
	view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.new {
		t.Fatalf("accepted release moved: %+v %v", view, err)
	}
	if !disagree || agree || !disagreeAgain {
		t.Fatalf("confirmation did not flap: disagree %v agree %v disagree-again %v", disagree, agree, disagreeAgain)
	}
}

func TestReviewR11_ParallelConflictIsOneRow(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	original := result(w, "succeeded")
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, original)); err != nil {
		t.Fatal(err)
	}
	alt := original
	alt.BootId = uuid.Must(uuid.NewV7())
	body := payload(t, alt)
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, err := l.s.SubmitResult(l.ctx, i, body)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err == nil {
			t.Fatal("conflicting submission was accepted")
		}
	}
	conflicts := l.scalar(`SELECT count(*) FROM update_result_conflicts WHERE result_id=$1`, original.ResultId)
	audits := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_result.conflict' AND after_diff->>'result_id'=$1`, original.ResultId.String())
	if conflicts != 1 || audits != 1 {
		t.Fatalf("two identical conflicting submissions stored %d conflict rows and %d audits", conflicts, audits)
	}
}

func TestReviewR11_FailedObservationDoesNotSkipErrorCode(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	report, svc := reviewObservation(t, l, i)
	report.Update.JobId.Set(w.Intent.JobId)
	report.Update.Generation.Set(w.Intent.Generation)
	report.Update.State.Set(api.UpdateJobStateFailed)
	report.Update.Stage.Set(api.UpdateStageCheck)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
		t.Fatal(err)
	}
	var state string
	var code *string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT state, error_code FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&state, &code); err != nil {
		t.Fatal(err)
	}
	if state == "failed" && (code == nil || *code == "") {
		t.Fatalf("observation state=failed with no error_code stored job state %s code %v", state, code)
	}
	if state == "failed" {
		t.Fatalf("observation moved the job to failed (%s) without a result error_code", *code)
	}
}

func TestReviewR11_RolledBackCandidateHealthOmittedAndNull(t *testing.T) {
	store := func(t *testing.T, mode string) (target api.UpdateTarget, evidence string) {
		t.Helper()
		l := newLabState(t)
		i := l.node("arm64", "unpublished", "registered", true)
		b := l.batch(l.new)
		w := l.intent(i)
		if _, err := l.s.SubmitResult(l.ctx, i, payload(t, result(w, "checking"))); err != nil {
			t.Fatal(err)
		}
		rolled := d062Result(t, l, w, "rollback")
		if mode == "omitted" {
			rolled.CandidateHealth.SetUnspecified()
		} else {
			rolled.CandidateHealth.SetNull()
		}
		if _, err := l.s.SubmitResult(l.ctx, i, payload(t, rolled)); err != nil {
			t.Fatal(mode, err)
		}
		if err := l.s.pool.QueryRow(l.ctx, `SELECT observed_evidence::text FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&evidence); err != nil {
			t.Fatal(err)
		}
		return l.get(b.BatchId).Targets[0], evidence
	}
	omittedTarget, omittedEvidence := store(t, "omitted")
	nullTarget, nullEvidence := store(t, "null")
	if omittedTarget.State != "rolled_back" || nullTarget.State != "rolled_back" {
		t.Fatalf("states %s %s", omittedTarget.State, nullTarget.State)
	}
	_, omittedErr := omittedTarget.CandidateHealth.Get()
	_, nullErr := nullTarget.CandidateHealth.Get()
	if omittedErr == nil || nullErr == nil {
		t.Fatalf("target candidate_health distinguished the modes: omitted err %v null err %v", omittedErr, nullErr)
	}
	t.Logf("both accepted; target candidate_health is null either way; omitted evidence %s; null evidence %s", omittedEvidence, nullEvidence)
}

func TestReviewR11_SubmitFailedStillRequiresErrorCode(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	stages := []api.UpdateStage{api.UpdateStageDownload, api.UpdateStageCheck, api.UpdateStageReport}
	for _, stage := range stages {
		failed := result(w, "failed")
		failed.ResultId = uuid.Must(uuid.NewV7())
		failed.Stage.Set(stage)
		failed.ErrorCode = presence.NewNullNullable[api.ErrorCode]()
		if stage == api.UpdateStageCheck || stage == api.UpdateStageReport {
			failed.CurrentReleaseId.Set(l.old)
			failed.Health.Set(api.UpdateHealth{
				Processes: api.HealthCheckStatusFailed, IdentityAndState: api.HealthCheckStatusFailed,
				InternetOnly: api.HealthCheckStatusFailed, DnsPolicy: api.HealthCheckStatusFailed, TunnelProbe: api.HealthCheckStatusFailed,
			})
		}
		if _, err := l.s.SubmitResult(l.ctx, i, payload(t, failed)); err == nil {
			t.Fatalf("stage %s accepted failed without error_code", stage)
		}
	}
	if l.scalar(`SELECT count(*) FROM update_jobs WHERE id=$1 AND state='failed' AND error_code IS NULL`, w.Intent.JobId) != 0 {
		t.Fatal("a failed job was stored without error_code")
	}
}
