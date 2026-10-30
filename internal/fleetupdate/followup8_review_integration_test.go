//go:build integration

package fleetupdate

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// §13.8 / r12 H-3: a superseded job's heartbeat must not borrow the current
// job's target or attribute its unexpected-release evidence to that job.
func TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b1 := l.batch(l.new)
	w1 := l.intent(i)

	// Keep the review's original pair: job1/new superseded by job2/old.
	l.batch(l.old)
	var job2ID uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT id FROM update_jobs WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, i.NodeID).Scan(&job2ID); err != nil {
		t.Fatal(err)
	}
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w1.Intent.JobId)
	r.Update.Generation.Set(w1.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	submit := func(current uuid.UUID) {
		t.Helper()
		r.ReportId = uuid.New()
		r.ObservedAt = r.ObservedAt.Add(time.Second)
		r.CurrentReleaseId.Set(current)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
			t.Fatal(err)
		}
		r11Versions(t, l, i, l.old, l.old)
		if target := l.get(b1.BatchId).Targets[0]; !target.Unconfirmed || target.State != "superseded" {
			t.Fatal("old job must remain superseded and unconfirmed", target)
		}
	}
	submit(l.new)
	if count := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_observation.unexpected_release' AND after_diff->>'job_id'=$1`, job2ID.String()); count != 0 {
		t.Fatalf("old heartbeat attributed %d audits to current job", count)
	}
	r11AuditReadback(t, l, i, map[[2]uuid.UUID]bool{{w1.Intent.JobId, l.new}: false})

	// Unlike job2/old, this next target is not an already recorded version.
	// An issued current intent still cannot authorize seeding via job1's report.
	other := reviewOtherRelease(t, l)
	l.support([]uuid.UUID{l.old, l.new, other}, []uuid.UUID{l.old, l.new, other}, []uuid.UUID{})
	l.batch(other)
	l.intent(i)
	for range 2 {
		submit(other)
	}
	r11AuditReadback(t, l, i, map[[2]uuid.UUID]bool{{w1.Intent.JobId, l.new}: false, {w1.Intent.JobId, other}: false})

	// Always exercise the review's second scenario, even when job attribution
	// passes: an unexpected last-good must prevent a partial current write.
	i2 := l.node("arm64", "unpublished", "registered", true)
	b2 := l.batch(l.new)
	w2 := l.intent(i2)
	r2, svc2 := reviewObservation(t, l, i2)
	r2.Update.JobId.Set(w2.Intent.JobId)
	r2.Update.Generation.Set(w2.Intent.Generation)
	r2.Update.State.Set(api.UpdateJobStateChecking)
	r2.Update.Stage.Set(api.UpdateStageCheck)
	r2.CurrentReleaseId.Set(l.new)
	r2.LastGoodReleaseId.Set(other)
	if _, err := svc2.SubmitObservation(l.ctx, i2.NodeID, i2.IncarnationID, r2); err != nil {
		t.Fatal(err)
	}
	r11Versions(t, l, i2, l.old, l.old)
	if count := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_observation.unexpected_release' AND after_diff->>'job_id'=$1 AND after_diff->>'release_id'=$2`, w2.Intent.JobId.String(), other.String()); count != 1 {
		t.Fatalf("unexpected last-good audits = %d, want 1", count)
	}
	for _, target := range l.get(b2.BatchId).Targets {
		if target.NodeId == i2.NodeID && !target.Unconfirmed {
			t.Fatal("unexpected last-good confirmed the current job", target)
		}
	}
}

// §13.8 / D-062(a): a pre-intent heartbeat cannot poison the unchanged release
// snapshotted by the first intent, so an honest pre-switch failure is accepted.
func TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	var jobID uuid.UUID
	var gen uint64
	var issued *time.Time
	if err := l.s.pool.QueryRow(l.ctx, `SELECT id, generation, issued_at FROM update_jobs WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, i.NodeID).Scan(&jobID, &gen, &issued); err != nil || issued != nil {
		t.Fatalf("unissued job: %v %v", issued, err)
	}

	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(jobID)
	r.Update.Generation.Set(decstr.U64(gen))
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.old)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	r11Versions(t, l, i, l.old, l.old)
	if !l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("unissued intent must not be confirmed")
	}
	r11AuditReadback(t, l, i, map[[2]uuid.UUID]bool{{jobID, l.new}: false})

	w := l.intent(i)
	var attemptRelease uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT (observed_evidence->>'attempt_release_id')::uuid FROM update_jobs WHERE id=$1`, jobID).Scan(&attemptRelease); err != nil || attemptRelease != l.old {
		t.Fatalf("poisoned attempt release: %s %v", attemptRelease, err)
	}
	res := result(w, "failed")
	res.Stage.Set(api.UpdateStageDownload)
	res.CurrentReleaseId.Set(l.old)
	res.Health.SetNull() // D-062(a): no checks ran before the download failure.
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, res))
	if err != nil || !ack.Accepted || ack.AcceptedAsHistory {
		t.Fatalf("legitimate pre-switch failure rejected or stored as history: %+v %v", ack, err)
	}
	target := l.get(b.BatchId).Targets[0]
	if target.State != "failed" || !target.NeedsAttention || !target.Health.IsNull() {
		t.Fatal("pre-switch failure evidence not projected", target)
	}
	r11Versions(t, l, i, l.old, l.old)
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
		// H-3 additionally preserves current when last-good disagrees.
		r11Versions(t, l, i, l.old, l.old)
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
		t.Fatal("another incarnation was accepted")
	}
	r11Versions(t, l, i, l.old, l.old)
	if count := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_observation.unexpected_release'`); count != 0 {
		t.Fatalf("known target as last-good produced %d unexpected-release audits", count)
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

// Updates harness case (c): a fresh incarnation has no centrally recorded boot
// version. Do not manufacture one; its claim remains one unexpected-release
// audit per job, even when repeated in otherwise valid target heartbeats.
func TestFollowup8_FreshIncarnationNullColumnsCaseC(t *testing.T) {
	l := newLabState(t)
	i := Identity{NodeID: uuid.Must(uuid.NewV7())}
	// Identity fixture only: version columns start with their schema defaults.
	if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO nodes(id,lifecycle_state) VALUES($1,'enrolling')`, i.NodeID); err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO exits(node_id,display_name,region,publication_intent) VALUES($1,'TEST ONLY fresh node','test','unpublished')`, i.NodeID); err != nil {
		t.Fatal(err)
	}
	i = l.enroll(i, "amd64")
	b := l.batch(l.new)
	w := l.intent(i)
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	r.CurrentReleaseId.Set(l.new)
	r.LastGoodReleaseId.Set(l.old)
	for range 4 {
		r.ReportId = uuid.New()
		r.ObservedAt = r.ObservedAt.Add(time.Second)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
			t.Fatal(err)
		}
		v, err := l.s.NodeVersionView(l.ctx, i.NodeID)
		if err != nil || v.CurrentReleaseID != nil || v.LastGoodReleaseID != nil || !v.Unconfirmed {
			t.Fatalf("unrecorded boot claim must preserve both null columns: %+v %v", v, err)
		}
		if !l.get(b.BatchId).Targets[0].Unconfirmed {
			t.Fatal("unrecorded last-good confirmed the job")
		}
	}
	if l.scalar(`SELECT count(*) FROM update_results WHERE node_id=$1`, i.NodeID) != 0 {
		t.Fatal("case (c) must precede accepted results")
	}
	r11AuditReadback(t, l, i, map[[2]uuid.UUID]bool{{w.Intent.JobId, l.old}: false})

	// Null agrees with the recorded null last-good. An issued current job may
	// then seed current, while last-good still waits for accepted health evidence.
	r.LastGoodReleaseId.SetNull()
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(time.Second)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	v, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || !equalID(v.CurrentReleaseID, &l.new) || v.LastGoodReleaseID != nil || v.Unconfirmed {
		t.Fatalf("issued target with agreeing null last-good should seed: %+v %v", v, err)
	}
}
