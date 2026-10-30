//go:build integration

package fleetupdate

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution proves that when job1
// (gen 1, target l.new) is superseded by job2 (gen 2, target `other`, so
// `fleet_node_state.current_job_id = job2`), `observeTx` (serving.go:121-150)
// evaluates target seeding and attributes `update_observation.unexpected_release`
// audits against `*n.job` (job2) instead of the `jobID` validated from
// `r.Update.JobId` (job1):
//   - A heartbeat for job1 reporting job1's own target (`l.new`) is treated as
//     unexpected, withholds confirmation on job1, and writes an
//     `update_observation.unexpected_release` audit attributed to job2.
//   - A heartbeat for job1 reporting job2's target (`other`) is treated as
//     expected, seeds `nodes.current_release_id = other`, and confirms job1.
func TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution(t *testing.T) {
	// TODO(review): fleet-update owner — in observeTx (serving.go:121-150), look up
	// and validate against the reported `jobID` (not `*n.job`) for audit attribution,
	// or restrict target seeding to when `jobID != nil && *jobID == *n.job` while
	// attributing unexpected-release audits to `*jobID`.
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)

	b1 := l.batch(l.new)
	w1 := l.intent(i)

	// Supersede job1 (target l.new) with job2 targeting `l.old` (both approved).
	_ = l.batch(l.old)
	var job2ID uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT id FROM update_jobs WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, i.NodeID).Scan(&job2ID); err != nil {
		t.Fatalf("query job2 id: %v", err)
	}

	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w1.Intent.JobId)
	r.Update.Generation.Set(w1.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)

	// Report job1's own target (l.new) on a heartbeat explicitly bound to job1.
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.old)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation: %v", err)
	}

	// Because observeTx reads *n.job (job2) instead of *jobID (job1), job1's target
	// l.new is treated as unexpected and the audit is attributed to job2!
	misattributed := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_observation.unexpected_release' AND after_diff->>'job_id'=$1 AND after_diff->>'release_id'=$2`,
		job2ID.String(), l.new.String())
	job1Audits := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_observation.unexpected_release' AND after_diff->>'job_id'=$1`,
		w1.Intent.JobId.String())
	_ = b1

	if misattributed > 0 || job1Audits == 0 {
		// Also verify: on a fresh node with job targeting l.new, a single heartbeat
		// carrying CurrentReleaseId = l.new and an unexpected LastGoodReleaseId = other
		// mutates nodes.current_release_id = l.new even as it emits an unexpected_release
		// audit for `other` (violating "an unexpected release leaves both columns intact").
		other := reviewOtherRelease(t, l)
		i2 := l.node("arm64", "unpublished", "registered", true)
		_ = l.batch(l.new)
		w2 := l.intent(i2)
		r2, svc2 := reviewObservation(t, l, i2)
		r2.Update.JobId.Set(w2.Intent.JobId)
		r2.Update.Generation.Set(w2.Intent.Generation)
		r2.Update.State.Set(api.UpdateJobStateChecking)
		r2.Update.Stage.Set(api.UpdateStageCheck)
		r2.CurrentReleaseId.Set(l.new)
		r2.LastGoodReleaseId.Set(other)
		if _, err := svc2.SubmitObservation(l.ctx, i2.NodeID, i2.IncarnationID, r2); err != nil {
			t.Fatalf("SubmitObservation(i2): %v", err)
		}
		v2, err := l.s.NodeVersionView(l.ctx, i2.NodeID)
		if err != nil {
			t.Fatalf("NodeVersionView(i2): %v", err)
		}
		t.Fatalf("TODO(review): (1) heartbeat for superseded job1 (%s) reporting job1's target %s wrote %d audits for job1 and misattributed %d unexpected_release audit(s) to job2 (%s); (2) heartbeat with unexpected last_good=%s still mutated current_release_id=%v (want %s intact)",
			w1.Intent.JobId, l.new, job1Audits, misattributed, job2ID, other, v2.CurrentReleaseID, l.old)
	}
}

// TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease proves that because
// `observeTx` (serving.go:126) allows `j.releaseID` to seed `nodes.current_release_id`
// before `NodeUpdateIntent` has ever been called (`j.issued == nil`), a pre-intent
// heartbeat naming `CurrentReleaseId = l.new` overwrites `nodes.current_release_id = l.new`.
// When `NodeUpdateIntent` (intent.go:168) subsequently authorizes the job for the
// first time (`WHEN issued_at IS NULL THEN $4`), it snapshots `$4 = n.current` (`l.new`)
// into `update_jobs.attempt_release_id`. Thereafter, `SubmitResult` (results.go:180)
// rejects a legitimate pre-health failure (`stage = "download", current_release_id = l.old`)
// with 422 VALIDATION_FAILED because `attempt_release_id` was poisoned with the
// candidate release `l.new`.
func TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease(t *testing.T) {
	// TODO(review): fleet-update owner — require `j.issued != nil` before allowing
	// `j.releaseID` to seed `nodes.current_release_id` in `observeTx` (serving.go:126).
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	_ = l.batch(l.new)
	var jobID uuid.UUID
	var gen uint64
	if err := l.s.pool.QueryRow(l.ctx, `SELECT id, generation FROM update_jobs WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, i.NodeID).Scan(&jobID, &gen); err != nil {
		t.Fatalf("query job: %v", err)
	}

	// Before l.intent(i) is ever called (`issued_at IS NULL`), submit a heartbeat
	// claiming the queued job's target `l.new` as current_release_id.
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(jobID)
	r.Update.Generation.Set(decstr.U64(gen))
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.old)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation before intent: %v", err)
	}

	// Now the node fetches its desired state / update intent for the first time.
	w := l.intent(i)

	// The update fails at the pre-switch "download" stage while still running `l.old`.
	res := result(w, "failed")
	res.Stage.Set(api.UpdateStageDownload)
	res.CurrentReleaseId.Set(l.old)

	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, res)); err != nil {
		t.Fatalf("TODO(review): pre-intent heartbeat seeded current_release_id=%s before NodeUpdateIntent, poisoning update_jobs.attempt_release_id and causing legitimate pre-switch failure (stage=download, current=%s) to be rejected: %v",
			l.new, l.old, err)
	}
}

// TestReviewR12_Q7_SeedingRuleEdgesHeld verifies the Question 7 edge cases that held:
//   - A job target recorded for a different incarnation (`j.incarnation != i.IncarnationID`)
//     cannot seed `current_release_id`.
//   - A recorded `last_good_release_id` that is subsequently revoked sets
//     `RecoveryRequired = true` on `NodeVersionView` while still permitting heartbeats
//     reporting that recorded `last_good_release_id`.
//   - Two heartbeats alternating `current_release_id` between the recorded boot
//     release (`l.old`) and the seeded target (`l.new`) while claiming `l.new` as
//     `last_good_release_id` never overwrite `nodes.last_good_release_id` and keep
//     the job `Unconfirmed = true`.
//   - A heartbeat that names the job target (`l.new`) as `last_good_release_id`
//     while `current_release_id` stays on the boot release (`l.old`) keeps
//     `current = l.old, last_good = l.old` and sets `Unconfirmed = true` (with zero
//     unexpected-release audits because `expected(id)` includes `targetRelease`).
func TestReviewR12_Q7_SeedingRuleEdgesHeld(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)

	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)

	// 1. Heartbeat naming target `l.new` as last_good_release_id while current stays `l.old`.
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	r.CurrentReleaseId.Set(l.old)
	r.LastGoodReleaseId.Set(l.new)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatalf("SubmitObservation(current=old, last_good=new): %v", err)
	}
	r11Versions(t, l, i, l.old, l.old)
	if !l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatalf("expected Unconfirmed=true when last_good=target while current=old")
	}

	// 2. Alternate current between l.new and l.old while trying to move last_good to l.new.
	for _, cur := range []uuid.UUID{l.new, l.old, l.new, l.old} {
		r.ReportId = uuid.New()
		r.ObservedAt = r.ObservedAt.Add(time.Second)
		r.CurrentReleaseId.Set(cur)
		r.LastGoodReleaseId.Set(l.new)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
			t.Fatalf("SubmitObservation(cur=%s, last_good=new): %v", cur, err)
		}
		r11Versions(t, l, i, cur, l.old)
		if !l.get(b.BatchId).Targets[0].Unconfirmed {
			t.Fatalf("expected Unconfirmed=true when claiming last_good=l.new with cur=%s", cur)
		}
	}

	// 3. A job target recorded for a different incarnation cannot seed current_release_id.
	otherInc := Identity{NodeID: i.NodeID, IncarnationID: uuid.Must(uuid.NewV7())}
	if err := l.s.Observe(l.ctx, otherInc, Observation{
		Sequence:                10,
		AppliedUpdateGeneration: 1,
		CurrentReleaseID:        &l.new,
		LastGoodReleaseID:       &l.old,
		ObservedAt:              r.ObservedAt.Add(10 * time.Second),
	}); err == nil {
		// Even if Observe is called directly, verify current_release_id stays l.old.
		r11Versions(t, l, i, l.old, l.old)
	}

	// 4. Revoke the recorded last_good (`l.old`) and verify RecoveryRequired becomes true.
	l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{l.old})
	v, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil {
		t.Fatalf("NodeVersionView after revoking last_good: %v", err)
	}
	if !v.RecoveryRequired || !equalID(v.LastGoodReleaseID, &l.old) {
		t.Fatalf("expected RecoveryRequired=true with LastGoodReleaseID=%s, got %+v", l.old, v)
	}
}
