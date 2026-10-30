//go:build integration

package fleetupdate

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func r11Versions(t *testing.T, l *lab, i Identity, current, lastGood uuid.UUID) {
	t.Helper()
	v, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || !equalID(v.CurrentReleaseID, &current) || !equalID(v.LastGoodReleaseID, &lastGood) || v.RecoveryRequired {
		t.Fatalf("versions: %+v, error %v; want current %s last-good %s without recovery", v, err, current, lastGood)
	}
}

// §13.2, §13.8: target seeding is permitted, but cannot become last-good by
// repeating a heartbeat. Only an accepted terminal result establishes that.
func TestReviewR11_PreResultTargetSeedPreservesLastGood(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	submit := func(current, lastGood uuid.UUID, unconfirmed bool) {
		t.Helper()
		r.ReportId = uuid.New()
		r.ObservedAt = r.ObservedAt.Add(time.Second)
		r.CurrentReleaseId.Set(current)
		r.LastGoodReleaseId.Set(lastGood)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
			t.Fatal(err)
		}
		r11Versions(t, l, i, current, l.old)
		if target := l.get(b.BatchId).Targets[0]; target.Unconfirmed != unconfirmed || target.State != "queued" {
			t.Fatal("observation confirmation must not promote the job", target)
		}
	}
	submit(l.new, l.old, false) // The current target is a valid pre-result seed.
	submit(l.new, l.new, true)  // Target membership does not establish last-good.
	submit(l.new, l.new, true)  // Nor does the version seeded by the previous report.
	submit(l.old, l.old, false) // The recorded boot/last-good release remains valid.
	submit(l.new, l.old, false)
	if l.scalar(`SELECT count(*) FROM update_results WHERE node_id=$1`, i.NodeID) != 0 {
		t.Fatal("the seed must precede every result")
	}
	ack := l.report(i, w, "succeeded")
	if !ack.AppliedToCurrentObservation || ack.AcceptedAsHistory {
		t.Fatal("the same-generation result still must establish last-good", ack)
	}
	r11Versions(t, l, i, l.new, l.new)
}

func TestReviewR11_UnexpectedReleaseAuditBound(t *testing.T) {
	l := newLabState(t)
	other := reviewOtherRelease(t, l)
	i := l.node("arm64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.Update.State.Set(api.UpdateJobStateChecking)
	r.Update.Stage.Set(api.UpdateStageCheck)
	submit := func() {
		t.Helper()
		r.ReportId = uuid.New()
		r.ObservedAt = r.ObservedAt.Add(time.Second)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
			t.Fatal(err)
		}
	}
	submit()
	if l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("the matching boot release should initially confirm")
	}
	r.LastGoodReleaseId.Set(other)
	submit() // An unexpected last-good alone also withholds confirmation.
	r.CurrentReleaseId.Set(other)
	for range 4 {
		submit()
		r11Versions(t, l, i, l.old, l.old)
		if !l.get(b.BatchId).Targets[0].Unconfirmed {
			t.Fatal("unexpected release confirmed the job")
		}
	}
	if count := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_observation.unexpected_release'`); count != 1 {
		t.Fatalf("repeated unexpected release wrote %d audits", count)
	}
	// A different release is independent evidence. Claiming a revoked unrelated
	// release cannot poison recovery either. Register through its real producer;
	// node observation receipts require an existing release (D-050).
	m := l.f.Manifest
	m.ReleaseId, m.ProductVersion = uuid.New(), "1.4.0-test"
	if _, err := l.s.releases.Register(l.ctx, releasetest.Command(), l.f.Request(t, m)); err != nil {
		t.Fatal(err)
	}
	l.support([]uuid.UUID{l.old, l.new}, []uuid.UUID{l.old, l.new}, []uuid.UUID{m.ReleaseId})
	r.CurrentReleaseId.Set(m.ReleaseId)
	r.LastGoodReleaseId.Set(m.ReleaseId)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		r.ReportId = uuid.New()
		r.ObservedAt = r.ObservedAt.Add(time.Second)
		// Each concurrent receipt has its own identity and observation time.
		copy := r
		wg.Go(func() {
			_, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, copy)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	r11Versions(t, l, i, l.old, l.old)
	if !l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("parallel disagreements confirmed the job")
	}
	want := map[[2]uuid.UUID]bool{{w.Intent.JobId, other}: false, {w.Intent.JobId, m.ReleaseId}: false}
	// The same unexpected release on a new job is new evidence, not suppressed
	// by the previous job's audit. No result has been accepted in either job.
	l.batch(l.new)
	w = l.intent(i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.CurrentReleaseId.Set(other)
	r.LastGoodReleaseId.Set(other)
	submit()
	want[[2]uuid.UUID{w.Intent.JobId, other}] = false
	r11Versions(t, l, i, l.old, l.old)
	r11AuditReadback(t, l, i, want)
}

func r11AuditReadback(t *testing.T, l *lab, i Identity, want map[[2]uuid.UUID]bool) {
	t.Helper()
	store := audit.Store{Pool: l.s.pool}
	limit := 2
	var cursor *string
	count := 0
	for {
		page, err := store.List(l.ctx, api.ListAuditEventsParams{Limit: &limit, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range page.Items {
			if event.Action != "update_observation.unexpected_release" {
				continue
			}
			actor, actorErr := event.Actor.Id.Get()
			request, requestErr := event.RequestId.Get()
			after, afterErr := event.Changes.After.Get()
			if actorErr != nil || actor != i.NodeID || requestErr != nil || request == uuid.Nil || afterErr != nil || len(after) != 2 || event.Actor.Type != api.AuditEventActorTypeNode || event.Result != api.AuditEventResultFailed || event.Resource != (api.ResourceRef{Type: api.ResourceTypeNode, Id: i.NodeID}) {
				t.Fatal("noncanonical observation audit", event)
			}
			jobID, jobErr := uuid.Parse(after["job_id"].(string))
			releaseID, releaseErr := uuid.Parse(after["release_id"].(string))
			key := [2]uuid.UUID{jobID, releaseID}
			seen, exists := want[key]
			if jobErr != nil || releaseErr != nil || !exists || seen {
				t.Fatal("unexpected or repeated release/job audit", after)
			}
			want[key] = true
			if l.scalar(`SELECT count(*) FROM update_jobs WHERE id=$1 AND node_id=$2`, jobID, i.NodeID) != 1 {
				t.Fatal("audit job does not belong to the node")
			}
			raw, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			if err = l.f.Validator.ValidateBody("AuditEvent", raw); err != nil {
				t.Fatal(err)
			}
			count++
		}
		next, err := page.NextCursor.Get()
		if err != nil {
			break
		}
		cursor = &next
	}
	if count != len(want) || store.AnomalyCount() != 0 {
		t.Fatalf("D-029: %d audits, want %d; %d anomalies", count, len(want), store.AnomalyCount())
	}
	t.Logf("D-029: %d distinct release/job audits, zero anomalies; repeats and concurrent reports add no duplicates", count)
}

func TestReviewR11_SupersededTargetCannotSeed(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	l.intent(i)
	b := l.batch(l.old)
	w := l.intent(i)
	r, svc := reviewObservation(t, l, i)
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	r.CurrentReleaseId.Set(l.new)
	if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, r); err != nil {
		t.Fatal(err)
	}
	r11Versions(t, l, i, l.old, l.old)
	if !l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("a superseded target must not confirm the current job")
	}
	r11AuditReadback(t, l, i, map[[2]uuid.UUID]bool{{w.Intent.JobId, l.new}: false})
}
