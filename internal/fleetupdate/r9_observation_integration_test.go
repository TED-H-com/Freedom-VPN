//go:build integration

package fleetupdate

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestReviewR9_D062_JoblessAndInvalidVocabulary(t *testing.T) {
	for _, mode := range []string{"jobless", "jobless_same_generation", "invalid_stage", "invalid_state_without_generation"} {
		t.Run(mode, func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "unpublished", "registered", true)
			b := l.batch(l.new)
			w := l.intent(i)
			l.report(i, w, "succeeded")
			r, svc := reviewObservation(t, l, i)
			r.Update.Generation.Set(w.Intent.Generation)
			r.Update.JobId.Set(w.Intent.JobId)
			r.Update.State.Set(api.UpdateJobStateSucceeded)
			r.Update.Stage.Set(api.UpdateStageReport)
			r.CurrentReleaseId.Set(l.new)
			r.LastGoodReleaseId.Set(l.new)
			if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
				t.Fatal(err)
			}
			if l.get(b.BatchId).Targets[0].Unconfirmed {
				t.Fatal("matching observation did not confirm")
			}
			r.ReportId = uuid.New()
			r.ObservedAt = r.ObservedAt.Add(time.Second)
			r.CurrentReleaseId.Set(l.old)
			r.LastGoodReleaseId.Set(l.old)
			switch mode {
			case "jobless":
				r.Update.JobId.SetNull()
				r.Update.Generation.Set(w.Intent.Generation + 5)
			case "jobless_same_generation":
				r.Update.JobId.SetNull()
			case "invalid_stage":
				r.Update.Stage.Set(api.UpdateStage("not-a-stage"))
			case "invalid_state_without_generation":
				r.Update.Generation.SetNull()
				r.Update.State.Set(api.UpdateJobState("not-a-state"))
			}
			_, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r)
			jobless := mode == "jobless" || mode == "jobless_same_generation"
			if jobless && err != nil || !jobless && !errors.Is(err, apierr.New(apierr.ValidationFailed, "")) {
				t.Fatal(err)
			}
			view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
			if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.new || reviewHighWater(t, l, i.NodeID) != w.Intent.Generation.String() {
				t.Fatal(view, err)
			}
			if target := l.get(b.BatchId).Targets[0]; target.Unconfirmed != jobless {
				t.Fatal("jobless removes confirmation; invalid input changes nothing", target)
			}
			if !jobless && l.scalar(`SELECT count(*) FROM node_observations WHERE report_id=$1`, r.ReportId) != 0 {
				t.Fatal("invalid vocabulary persisted")
			}
		})
	}
}

func TestReviewR9_D062_PreResultSeedAndTerminalAuthority(t *testing.T) {
	l := newLabState(t)
	i := l.node("arm64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	r, svc := reviewObservation(t, l, i)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	// No accepted result for this incarnation yet: the explicit, bound seed
	// path can reconcile installed versions. A terminal result still wins.
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.new)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	seeded, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || seeded.CurrentReleaseID == nil || *seeded.CurrentReleaseID != l.new {
		t.Fatal(seeded, err)
	}
	rolled := d062Result(t, l, w, "rollback")
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, rolled))
	if err != nil || ack.AcceptedAsHistory || !ack.AppliedToCurrentObservation {
		t.Fatal("same-generation terminal result must replace the pre-result seed", ack, err)
	}
	view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.old || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.old {
		t.Fatal(view, err)
	}
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	if _, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	if target := l.get(b.BatchId).Targets[0]; !target.Unconfirmed || nullableValue(target.CurrentReleaseId) != l.old {
		t.Fatal("accepted result did not fence later seed attempts", target)
	}
}

func TestReviewR9_D062_RecoveryAndRetryAttemptAuthority(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	failed := result(w, "failed")
	failed.Stage.Set(api.UpdateStageCheck)
	failed.ErrorCode.Set(api.ErrorCodeDOWNLOADTIMEOUT)
	failed.CurrentReleaseId.Set(l.old)
	failed.RecoveryRequired = presence.NewRequired(true)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, failed)); err != nil {
		t.Fatal(err)
	}
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateFailed)
	r.Update.Stage.Set(api.UpdateStageCheck)
	r.CurrentReleaseId.Set(l.old)
	r.LastGoodReleaseId.Set(l.new) // disagree only about last-good
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || !view.RecoveryRequired || !view.Unconfirmed || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.old {
		t.Fatal(view, err)
	}
	if _, err = l.s.Retry(l.ctx, releasetest.Command(), b.BatchId, api.UpdateBatchRetryRequest{Reason: "TEST ONLY recovery"}); err != nil {
		t.Fatal(err)
	}
	next := l.intent(i)
	if next.Intent.Generation != w.Intent.Generation || next.Intent.Attempt != w.Intent.Attempt+1 {
		t.Fatal(next.Intent)
	}
	l.report(i, next, "succeeded")
	view, err = l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || view.RecoveryRequired || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.new {
		t.Fatal("only a current accepted result can clear recovery", view, err)
	}
	// Same generation, wrong attempt/job: even matching versions cannot confirm.
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	r.CurrentReleaseId.Set(l.new)
	if _, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	if !l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("prior attempt confirmed a retry")
	}
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	r.Update.JobId.Set(next.Intent.JobId)
	r.Update.State.Set(api.UpdateJobStateSucceeded)
	r.Update.Stage.Set(api.UpdateStageReport)
	if _, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	if l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("retry job observation did not confirm")
	}
}

func TestReviewR9_D062_FailedRequiresErrorAtEveryStage(t *testing.T) {
	l := newLabState(t)
	i := l.node("arm64", "disabled", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	stages := []api.UpdateStage{"", api.UpdateStageClaim, api.UpdateStageDownload,
		api.UpdateStageVerify, api.UpdateStagePreflight, api.UpdateStagePrepare,
		api.UpdateStageBlockForwarding, api.UpdateStageFinalSample, api.UpdateStageStopServices,
		api.UpdateStageMigrate, api.UpdateStageSwitch, api.UpdateStageStart,
		api.UpdateStageCheck, api.UpdateStageCommit, api.UpdateStageRollback, api.UpdateStageReport}
	for _, stage := range stages {
		r := result(w, "failed")
		r.ErrorCode.SetNull()
		r.CurrentReleaseId.Set(l.old)
		if stage != "" {
			r.Stage.Set(stage)
		}
		if beforeHealth(&stage) {
			r.Health.SetNull()
		}
		_, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
		var rejected *apierr.Error
		if !errors.As(err, &rejected) || rejected.Code != apierr.ValidationFailed || rejected.HTTPStatus() != 422 || rejected.IsRetryable() {
			t.Fatal(stage, err)
		}
		if _, replay := l.s.SubmitResult(l.ctx, i, payload(t, r)); replay == nil || replay.Error() != err.Error() {
			t.Fatal("rejection replay differs", stage, replay, err)
		}
	}
	if l.scalar(`SELECT count(*) FROM update_results WHERE job_id=$1 AND ack->'error'->>'code'='VALIDATION_FAILED'`, w.Intent.JobId) != len(stages) {
		t.Fatal("failed results without causes must all have rejection receipts")
	}
	if target := l.get(b.BatchId).Targets[0]; target.State != "queued" || target.NeedsAttention || nullableValue(target.CurrentReleaseId) != l.old || !target.Health.IsNull() {
		t.Fatal("invalid failures changed accepted state or evidence", target)
	}
}
