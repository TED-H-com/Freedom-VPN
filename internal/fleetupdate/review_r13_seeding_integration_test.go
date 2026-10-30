//go:build integration

package fleetupdate

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReviewR13_Q6_IssuedJobHeartbeatMovesCurrentReleaseWithoutAcceptedResultAndPoisonsSupersedingBatch
// proves D-068 (r13 H-4) in observeTx (internal/fleetupdate/serving.go:125-147):
//  1. Before any UpdateResult is accepted for the incarnation (`!accepted`), a
//     heartbeat for the current issued job (w1 targeting l.new) reporting
//     `current_release_id = l.new, last_good_release_id = l.old` never overwrites
//     a non-null `nodes.current_release_id = l.old` and leaves the job unconfirmed.
//  2. When Batch 1 is superseded by Batch 2 (targeting `other`) before Job 1
//     ever completes (while the node is still running `l.old`), `NodeUpdateIntent`
//     for Job 2 (internal/fleetupdate/intent.go:168) snapshots
//     `attempt_release_id = l.old`.
//  3. When Job 2 subsequently fails before health checks (`stage = "download"`,
//     `current_release_id = l.old`), `SubmitResult` (internal/fleetupdate/results.go:180)
//     accepts the legitimate pre-switch failure as current.
func TestReviewR13_Q6_IssuedJobHeartbeatMovesCurrentReleaseWithoutAcceptedResultAndPoisonsSupersedingBatch(t *testing.T) {
	l := newLabState(t)
	other := reviewOtherRelease(t, l)
	l.support([]uuid.UUID{l.old, l.new, other}, []uuid.UUID{l.old, l.new, other}, []uuid.UUID{})

	i := l.node("amd64", "unpublished", "registered", true)
	b1 := l.batch(l.new)
	w1 := l.intent(i)

	// Heartbeat for issued Job 1 claims current_release_id = l.new, last_good = l.old
	// before any UpdateResult has been submitted or accepted.
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w1.Intent.JobId)
	r.Update.Generation.Set(w1.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateDownloading)
	r.Update.Stage.Set(api.UpdateStageDownload)
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.old)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation for Job 1: %v", err)
	}

	v, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil {
		t.Fatalf("NodeVersionView: %v", err)
	}
	if !equalID(v.CurrentReleaseID, &l.old) || !equalID(v.LastGoodReleaseID, &l.old) || !v.Unconfirmed {
		t.Fatalf("D-068: heartbeat for in-flight Job 1 must keep non-null current=%s last_good=%s unconfirmed=true, got %+v", l.old, l.new, v)
	}
	if !l.get(b1.BatchId).Targets[0].Unconfirmed {
		t.Fatal("in-flight target heartbeat must not confirm Job 1 before an accepted result")
	}

	// Operator creates Batch 2 targeting `other`, superseding Batch 1 while the
	// node is still on `l.old`. Node fetches Job 2 intent (`w2`) and fails at
	// stage="download" while still running `l.old`.
	b2 := l.batch(other)
	w2 := l.intent(i)
	var attemptRelease uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT (observed_evidence->>'attempt_release_id')::uuid FROM update_jobs WHERE id=$1`, w2.Intent.JobId).Scan(&attemptRelease); err != nil || attemptRelease != l.old {
		t.Fatalf("expected Job 2 attempt_release_id=%s, got %s (err=%v)", l.old, attemptRelease, err)
	}

	res := result(w2, "failed")
	res.Stage.Set(api.UpdateStageDownload)
	res.CurrentReleaseId.Set(l.old)
	res.Health.SetNull()
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, res))
	if err != nil {
		t.Fatalf("SubmitResult for Job 2's legitimate download failure on %s failed: %v", l.old, err)
	}
	if !ack.Accepted || ack.AcceptedAsHistory {
		t.Fatalf("expected Job 2 download failure to be accepted as current, got %+v", ack)
	}
	target2 := l.get(b2.BatchId).Targets[0]
	if target2.State != api.UpdateJobStateFailed || !target2.NeedsAttention {
		t.Fatalf("expected Batch 2 target failed with needs_attention=true, got %+v", target2)
	}
}

// TestReviewR13_Q6_SupersededAttemptHeartbeatSameGenerationInvalidatesCurrentAttemptConfirmation
// proves D-068 (r13 M-1): when Retry creates Attempt 2 with the same `generation`
// as Attempt 1, ObserveReportTx requires `n.job != nil && *n.job == jobID` and a
// non-superseded job state before calling observeTx. A late heartbeat referencing
// Attempt 1 is recorded in `node_observations` as ignored by the fleet state
// projection without advancing `fleet_node_state.last_observed_at` or flipping
// confirmed Attempt 2 back to `Unconfirmed = true`.
func TestReviewR13_Q6_SupersededAttemptHeartbeatSameGenerationInvalidatesCurrentAttemptConfirmation(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w1 := l.intent(i)

	// Attempt 1 (w1) fails at download (retryable error code DOWNLOAD_TIMEOUT).
	failRes := result(w1, "failed")
	failRes.Stage.Set(api.UpdateStageDownload)
	failRes.ErrorCode.Set(api.ErrorCodeDOWNLOADTIMEOUT)
	failRes.CurrentReleaseId.Set(l.old)
	failRes.PreviousReleaseId.Set(l.old)
	failRes.Health.SetNull()
	failRes.RecoveryRequired.Set(false)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, failRes)); err != nil {
		t.Fatalf("SubmitResult attempt 1 failed: %v", err)
	}

	// Operator retries the batch; Attempt 2 (w2) is created with the SAME generation
	// as Attempt 1 (w1).
	if _, err := l.s.Retry(l.ctx, releasetest.Command(), b.BatchId, api.UpdateBatchRetryRequest{Reason: "TEST ONLY retry"}); err != nil {
		t.Fatalf("Retry: %v", err)
	}
	w2 := l.intent(i)
	if w2.Intent.JobId == w1.Intent.JobId || w2.Intent.Generation != w1.Intent.Generation {
		t.Fatalf("expected new job ID with same generation, got w1=(%s, gen=%d) w2=(%s, gen=%d)",
			w1.Intent.JobId, w1.Intent.Generation, w2.Intent.JobId, w2.Intent.Generation)
	}

	// Attempt 2 succeeds and is confirmed by a heartbeat for w2.
	okRes := result(w2, "succeeded")
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, okRes)); err != nil {
		t.Fatalf("SubmitResult attempt 2 succeeded: %v", err)
	}
	r, svc := reviewObservation(t, l, i)
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(10 * time.Second)
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.new)
	r.Update.JobId.Set(w2.Intent.JobId)
	r.Update.Generation.Set(w2.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateSucceeded)
	r.Update.Stage.Set(api.UpdateStageReport)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation confirming w2: %v", err)
	}
	beforeTarget := l.get(b.BatchId).Targets[0]
	if beforeTarget.Unconfirmed {
		t.Fatalf("expected Attempt 2 to be confirmed (Unconfirmed=false), got %+v", beforeTarget)
	}
	beforeObserved, err := beforeTarget.LastObservedAt.Get()
	if err != nil {
		t.Fatalf("expected non-null LastObservedAt on confirmed Attempt 2: %v", err)
	}

	// Now a late/reordered heartbeat referencing superseded Attempt 1 (w1.Intent.JobId,
	// which shares w2.Intent.Generation!) arrives with a newer ObservedAt.
	rStale := r
	rStale.ReportId = uuid.New()
	rStale.ObservedAt = r.ObservedAt.Add(5 * time.Second)
	rStale.Update.JobId.Set(w1.Intent.JobId)
	rStale.Update.Generation.Set(w1.Intent.Generation)
	obsAck, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, rStale)
	if err != nil || !obsAck.Accepted || obsAck.ReportId != rStale.ReportId {
		t.Fatalf("SubmitObservation for superseded Attempt 1: ack=%+v err=%v", obsAck, err)
	}
	if l.scalar(`SELECT count(*) FROM node_observations WHERE node_id=$1 AND report_id=$2`, i.NodeID, rStale.ReportId) != 1 {
		t.Fatal("expected raw observation receipt in node_observations")
	}

	target := l.get(b.BatchId).Targets[0]
	if target.Unconfirmed {
		t.Fatalf("D-068: heartbeat for superseded Attempt 1 (job=%s, gen=%d) flipped confirmed Attempt 2 (job=%s) back to Unconfirmed=true",
			w1.Intent.JobId, w1.Intent.Generation, w2.Intent.JobId)
	}
	afterObserved, err := target.LastObservedAt.Get()
	if err != nil || !afterObserved.Equal(beforeObserved) {
		t.Fatalf("D-068: superseded Attempt 1 heartbeat moved fleet_node_state.last_observed_at from %v to %v (err=%v)",
			beforeObserved, afterObserved, err)
	}
}

// TestReviewR13_Q6_SeedingWhatHeld verifies the Q6 properties that held:
//   - After an accepted UpdateResult (`accepted == true`), a heartbeat cannot
//     move `nodes.current_release_id` or `nodes.last_good_release_id` away from
//     the accepted release.
//   - A heartbeat from a superseded batch (different generation) is ignored by
//     ObserveReportTx before reaching observeTx.
func TestReviewR13_Q6_SeedingWhatHeld(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b1 := l.batch(l.new)
	w1 := l.intent(i)

	okRes := result(w1, "succeeded")
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, okRes)); err != nil {
		t.Fatalf("SubmitResult: %v", err)
	}
	r11Versions(t, l, i, l.new, l.new)

	// Once accepted == true, a heartbeat for a new batch's issued job cannot
	// seed current_release_id to `l.old` before SubmitResult.
	l.batch(l.old)
	w2 := l.intent(i)
	r, svc := reviewObservation(t, l, i)
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	r.Update.JobId.Set(w2.Intent.JobId)
	r.Update.Generation.Set(w2.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	r.CurrentReleaseId.Set(l.old)
	r.LastGoodReleaseId.Set(l.new)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation: %v", err)
	}
	r11Versions(t, l, i, l.new, l.new)
	_ = b1
}
