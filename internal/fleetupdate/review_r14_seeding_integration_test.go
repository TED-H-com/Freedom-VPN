//go:build integration

package fleetupdate

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func reviewNullCurrent(t *testing.T, l *lab, i Identity) {
	t.Helper()
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE nodes SET current_release_id=NULL, resource_version=resource_version+1 WHERE id=$1`, i.NodeID); err != nil {
		t.Fatal(err)
	}
}

func reviewVersions(t *testing.T, l *lab, i Identity) (current, lastGood *uuid.UUID) {
	t.Helper()
	v, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	return v.CurrentReleaseID, v.LastGoodReleaseID
}

func reviewHeartbeat(t *testing.T, l *lab, i Identity, w *Work, current, lastGood uuid.UUID, at time.Time) {
	t.Helper()
	r, svc := reviewObservation(t, l, i)
	r.ReportId = uuid.New()
	r.ObservedAt = at
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateDownloading)
	r.Update.Stage.Set(api.UpdateStageDownload)
	r.CurrentReleaseId.Set(current)
	r.LastGoodReleaseId.Set(lastGood)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation: %v", err)
	}
}

// TestReviewR14_NullCurrentIsTheOnlyHeartbeatSeed proves the D-068 predicate.
// A heartbeat fills current_release_id only when it is null, the reported
// last-good equals the stored last-good, and the reported current release is
// the job target or that last-good. A forged last-good, an unexpected release,
// and a stale incarnation do not write it.
func TestReviewR14_NullCurrentIsTheOnlyHeartbeatSeed(t *testing.T) {
	t.Run("null_current_and_agreeing_last_good_seeds_the_target", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		reviewNullCurrent(t, l, i)
		reviewHeartbeat(t, l, i, w, l.new, l.old, time.Now().UTC().Add(2*time.Hour))
		current, lastGood := reviewVersions(t, l, i)
		if !equalID(current, &l.new) || !equalID(lastGood, &l.old) {
			t.Fatalf("documented null seed: current=%v last_good=%v, want current %s last_good %s", current, lastGood, l.new, l.old)
		}
	})

	t.Run("forged_last_good_does_not_seed", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		reviewNullCurrent(t, l, i)
		reviewHeartbeat(t, l, i, w, l.new, l.new, time.Now().UTC().Add(2*time.Hour))
		current, lastGood := reviewVersions(t, l, i)
		if current != nil || !equalID(lastGood, &l.old) {
			t.Fatalf("forged last-good seeded current=%v last_good=%v", current, lastGood)
		}
	})

	t.Run("unexpected_release_does_not_seed", func(t *testing.T) {
		l := newLabState(t)
		other := reviewOtherRelease(t, l)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		reviewNullCurrent(t, l, i)
		reviewHeartbeat(t, l, i, w, other, l.old, time.Now().UTC().Add(2*time.Hour))
		current, _ := reviewVersions(t, l, i)
		if current != nil {
			t.Fatalf("unexpected release %s seeded current", other)
		}
	})

	t.Run("stale_incarnation_replay_does_not_seed", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		reviewNullCurrent(t, l, i)
		stale := i
		// A second active incarnation is rejected by node_incarnations_active_key.
		// Reinstall revokes the previous one, then mints the current one.
		if _, err := l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=now() WHERE id=$1`, stale.IncarnationID); err != nil {
			t.Fatal(err)
		}
		fresh := l.enroll(i, "amd64")
		if fresh.IncarnationID == stale.IncarnationID {
			t.Fatal("reinstall did not mint an incarnation")
		}
		r, svc := reviewObservation(t, l, stale)
		r.Update.JobId.Set(w.Intent.JobId)
		r.Update.Generation.Set(w.Intent.Generation)
		r.Update.State.Set(api.UpdateJobStateDownloading)
		r.Update.Stage.Set(api.UpdateStageDownload)
		r.CurrentReleaseId.Set(l.new)
		r.LastGoodReleaseId.Set(l.old)
		_, err := svc.SubmitObservation(l.ctx, stale.NodeID, stale.IncarnationID, r)
		// The active-incarnation index forces reinstall to revoke the old row
		// first. That replay is rejected as revoked and does not seed.
		if !errors.Is(err, apierr.New(apierr.IdentityRevoked, "")) && !errors.Is(err, apierr.New(apierr.IncarnationMismatch, "")) {
			t.Fatalf("stale incarnation: %v", err)
		}
		current, _ := reviewVersions(t, l, fresh)
		if current != nil {
			t.Fatalf("stale incarnation seeded current %s", current)
		}
	})
}

// TestReviewR14_SupersededJobHeartbeatDoesNotStarveNodeLiveness is an agent
// still reporting the pre-retry job. fleet_node_state.last_observed_at stays
// on the current attempt. nodes.last_observed_at, last_seen_at and the
// node_observations row still move, which is what control_status uses.
func TestReviewR14_SupersededJobHeartbeatDoesNotStarveNodeLiveness(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w1 := l.intent(i)
	reviewHeartbeat(t, l, i, w1, l.old, l.old, time.Now().UTC().Add(time.Hour))

	failRes := result(w1, "failed")
	failRes.Stage.Set(api.UpdateStageDownload)
	failRes.ErrorCode.Set(api.ErrorCodeDOWNLOADTIMEOUT)
	failRes.CurrentReleaseId.Set(l.old)
	failRes.PreviousReleaseId.Set(l.old)
	failRes.Health.SetNull()
	failRes.RecoveryRequired.Set(false)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, failRes)); err != nil {
		t.Fatalf("attempt 1: %v", err)
	}
	if _, err := l.s.Retry(l.ctx, releasetest.Command(), b.BatchId, api.UpdateBatchRetryRequest{Reason: "TEST ONLY retry"}); err != nil {
		t.Fatal(err)
	}
	w2 := l.intent(i)
	if w2.Intent.JobId == w1.Intent.JobId {
		t.Fatal("retry did not mint a job")
	}

	var fleetBefore, nodeBefore time.Time
	if err := l.s.pool.QueryRow(l.ctx, `SELECT last_observed_at FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&fleetBefore); err != nil {
		t.Fatal(err)
	}
	if err := l.s.pool.QueryRow(l.ctx, `SELECT last_observed_at FROM nodes WHERE id=$1`, i.NodeID).Scan(&nodeBefore); err != nil {
		t.Fatal(err)
	}
	r, svc := reviewObservation(t, l, i)
	r.ReportId = uuid.New()
	r.ObservedAt = time.Now().UTC().Add(3 * time.Hour)
	r.Update.JobId.Set(w1.Intent.JobId)
	r.Update.Generation.Set(w1.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateDownloading)
	r.Update.Stage.Set(api.UpdateStageDownload)
	r.CurrentReleaseId.Set(l.old)
	r.LastGoodReleaseId.Set(l.old)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	if l.scalar(`SELECT count(*) FROM node_observations WHERE node_id=$1 AND report_id=$2`, i.NodeID, r.ReportId) != 1 {
		t.Fatal("superseded heartbeat was not stored in node_observations")
	}
	var fleetAfter, nodeAfter, seen time.Time
	var control string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT last_observed_at FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&fleetAfter); err != nil {
		t.Fatal(err)
	}
	if err := l.s.pool.QueryRow(l.ctx, `SELECT last_observed_at, last_seen_at, control_status FROM nodes WHERE id=$1`, i.NodeID).Scan(&nodeAfter, &seen, &control); err != nil {
		t.Fatal(err)
	}
	if !fleetAfter.Equal(fleetBefore) {
		t.Fatalf("superseded job moved fleet_node_state.last_observed_at from %s to %s", fleetBefore, fleetAfter)
	}
	if !nodeAfter.After(nodeBefore) || seen.IsZero() || control != "online" {
		t.Fatalf("node liveness did not move: observed %s -> %s seen %s control %s", nodeBefore, nodeAfter, seen, control)
	}
	view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || view.LastObservedAt == nil || !view.LastObservedAt.Equal(fleetBefore.UTC()) && !view.LastObservedAt.Equal(fleetAfter.UTC()) {
		t.Fatalf("version view clock: %+v %v", view, err)
	}
}

func reviewFailedRollback(t *testing.T, w *Work, current uuid.UUID) api.UpdateResult {
	t.Helper()
	r := result(w, "failed")
	r.Stage.Set(api.UpdateStageRollback)
	r.RecoveryRequired.Set(true)
	r.CurrentReleaseId.Set(current)
	r.PreviousReleaseId.Set(current)
	r.Health.Set(api.UpdateHealth{
		Processes: api.HealthCheckStatusFailed, IdentityAndState: api.HealthCheckStatusFailed,
		InternetOnly: api.HealthCheckStatusFailed, DnsPolicy: api.HealthCheckStatusFailed,
		TunnelProbe: api.HealthCheckStatusFailed,
	})
	return r
}

// TestReviewR14_BlankRollbackProfileCannotRetargetAnotherJob submits the
// all-failed rollback health profile for job 1, then tries to reuse its
// result id and to present job 2's ids from node 1.
func TestReviewR14_BlankRollbackProfileCannotRetargetAnotherJob(t *testing.T) {
	l := newLabState(t)
	a := l.node("amd64", "unpublished", "registered", true)
	b := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	wa := l.intent(a)
	wb := l.intent(b)
	if wa.Intent.JobId == wb.Intent.JobId {
		t.Fatal("one batch did not mint a job per node")
	}
	res := reviewFailedRollback(t, wa, l.old)
	raw := payload(t, res)
	ack, err := l.s.SubmitResult(l.ctx, a, raw)
	if err != nil || !ack.Accepted || ack.AcceptedAsHistory {
		t.Fatalf("job 1 blank rollback profile: ack=%+v err=%v", ack, err)
	}
	state, attention, release := reviewJobAttention(t, l, wa.Intent.JobId)
	if state != "failed" || !attention || release == nil || *release != l.old {
		t.Fatalf("job 1 after accept: state=%s attention=%v release=%v", state, attention, release)
	}
	replay, err := l.s.SubmitResult(l.ctx, a, raw)
	if err != nil || replay.ResultId != ack.ResultId || !replay.Accepted {
		t.Fatalf("identical replay: ack=%+v err=%v", replay, err)
	}
	changed := res
	changed.BootId = uuid.New()
	if _, err = l.s.SubmitResult(l.ctx, a, payload(t, changed)); !errors.Is(err, apierr.New(apierr.AckConflict, "")) {
		t.Fatalf("same result_id different hash: %v", err)
	}
	cross := reviewFailedRollback(t, wb, l.old)
	cross.NodeId = a.NodeID
	cross.IncarnationId = a.IncarnationID
	if _, err = l.s.SubmitResult(l.ctx, a, payload(t, cross)); !errors.Is(err, apierr.New(apierr.IncarnationMismatch, "")) {
		t.Fatalf("node 1 presenting job 2 ids: %v", err)
	}
	stateB, _, releaseB := reviewJobAttention(t, l, wb.Intent.JobId)
	if stateB != "queued" || releaseB != nil {
		t.Fatalf("job 2 moved: state=%s release=%v", stateB, releaseB)
	}
	currentA, _ := reviewVersions(t, l, a)
	currentB, _ := reviewVersions(t, l, b)
	if !equalID(currentA, &l.old) || !equalID(currentB, &l.old) {
		t.Fatalf("current releases moved: a=%v b=%v", currentA, currentB)
	}
}
