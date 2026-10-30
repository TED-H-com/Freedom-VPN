//go:build integration

package fleetupdate

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// The captured real producer sends only this terminal result. Rebind identity,
// manifest and authorization times, keeping both health profiles and all other
// evidence. The fixture itself is an unmodified, canonically hashed capture.
func at26DirectRollback(t *testing.T) (*lab, Identity, api.UpdateBatchAccepted, *Work, api.UpdateResult) {
	t.Helper()
	l := newLab(t)
	raw, err := os.ReadFile("testdata/at26-direct-rollback.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = l.f.Validator.ValidateBody("UpdateResult", raw); err != nil {
		t.Fatal(err)
	}
	var r api.UpdateResult
	if err = json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	hash, err := canonjson.HashHex(raw, "payload_hash")
	if err != nil || hash != r.PayloadHash {
		t.Fatal("capture hash differs", hash, err)
	}
	candidateHealth, err := r.CandidateHealth.Get()
	if err != nil || candidateHealth.TunnelProbe != "failed" || !healthyValue(r) {
		t.Fatal("capture lost candidate/restored health", r, err)
	}
	candidate := l.f.Manifest
	candidate.ReleaseId = uuid.New()
	candidate.ProductVersion = "1.3.0-test"
	candidate.Rollback.AllowedReleaseIds = []uuid.UUID{}
	if _, err = l.s.releases.Register(l.ctx, releasetest.Command(), l.f.Request(t, candidate)); err != nil {
		t.Fatal(err)
	}
	l.support([]uuid.UUID{l.old, candidate.ReleaseId}, []uuid.UUID{l.old, candidate.ReleaseId}, []uuid.UUID{})
	i := l.node("amd64", "unpublished", "registered", true)
	// Reach the capture's fourth generation through real commands, never by
	// writing a job or observation projector row.
	for range 3 {
		l.batch(l.old)
	}
	b := l.batch(candidate.ReleaseId)
	w := l.intent(i)
	r.NodeId, r.IncarnationId, r.JobId = i.NodeID, i.IncarnationID, w.Intent.JobId
	r.BatchId, r.TargetId, r.Attempt = w.Intent.BatchId, w.Intent.TargetId, w.Intent.Attempt
	r.Generation, r.RequestedReleaseId, r.ManifestSha256 = w.Intent.Generation, w.Intent.ReleaseId, w.Intent.ManifestSha256
	r.CurrentReleaseId.Set(l.old)
	r.PreviousReleaseId.Set(w.Intent.ReleaseId)
	r.StartedAt = time.Now().UTC()
	r.FinishedAt.Set(r.StartedAt)
	return l, i, b, w, r
}

func TestAT26_DirectRollbackFromQueued(t *testing.T) {
	for _, observed := range []bool{false, true} {
		name := "no_update_observation"
		if observed {
			name = "same_generation_observation"
		}
		t.Run(name, func(t *testing.T) {
			l, i, b, w, r := at26DirectRollback(t)
			if observed {
				// A restored-version heartbeat can precede result delivery, as in
				// the live capture. It is not a progress result or state promotion.
				if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: r.Generation, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
					t.Fatal(err)
				}
			}
			if before := l.get(b.BatchId); before.Targets[0].State != "queued" || l.scalar(`SELECT count(*) FROM update_results WHERE job_id=$1`, w.Intent.JobId) != 0 {
				t.Fatal("test must start queued without a progress result", before)
			}
			raw := payload(t, r)
			ack, err := l.s.SubmitResult(l.ctx, i, raw)
			if err != nil {
				t.Fatal(err)
			}
			batch := l.get(b.BatchId)
			ackJSON, err := json.Marshal(ack)
			if err != nil {
				t.Fatal(err)
			}
			batchJSON, err := json.Marshal(batch)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("AT26_ACK=%s", ackJSON)
			t.Logf("AT26_BATCH=%s", batchJSON)
			target := batch.Targets[0]
			job := nullableValue(target.CurrentJob)
			if !ack.Accepted || ack.AcceptedAsHistory || ack.AppliedToCurrentObservation == observed || ack.ObservedUpdateGeneration != r.Generation || target.State != "rolled_back" || !target.NeedsAttention || !target.Unconfirmed || nullableValue(target.CurrentReleaseId) != l.old || nullableValue(job.ErrorCode) != api.ErrorCode(apierr.HealthCheckFailed) || batch.Counts.RolledBack != 1 || batch.TargetCount != 1 || batch.State != "completed_with_exceptions" {
				t.Fatal("direct rollback must close the current job and target")
			}
			var state, code string
			var actual, resultID uuid.UUID
			var stored []byte
			if err = l.s.pool.QueryRow(l.ctx, `SELECT state,error_code,actual_release_id,result_id,observed_evidence FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&state, &code, &actual, &resultID, &stored); err != nil {
				t.Fatal(err)
			}
			var evidence api.UpdateResult
			if err = json.Unmarshal(stored, &evidence); err != nil || state != "rolled_back" || code != "HEALTH_CHECK_FAILED" || actual != l.old || resultID != r.ResultId || nullableValue(evidence.CandidateHealth).TunnelProbe != "failed" || !healthyValue(evidence) || nullableValue(evidence.PreviousReleaseId) != w.Intent.ReleaseId {
				t.Fatal(state, code, actual, resultID, err)
			}
			replay, err := l.s.SubmitResult(l.ctx, i, raw)
			if err != nil || replay != ack || l.scalar(`SELECT count(*) FROM update_results WHERE job_id=$1`, w.Intent.JobId) != 1 {
				t.Fatal("replay changed receipt or manufactured progress", replay, err)
			}
			if err = l.s.Observe(l.ctx, i, Observation{Sequence: 3, AppliedUpdateGeneration: r.Generation, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
				t.Fatal(err)
			}
			confirmed := l.get(b.BatchId).Targets[0]
			if confirmed.Unconfirmed || confirmed.State != "rolled_back" || !confirmed.NeedsAttention {
				t.Fatal("later authenticated observation must confirm without changing disposition", confirmed)
			}
		})
	}
}

func TestAT26_DirectSuccessAndFailureFromQueued(t *testing.T) {
	for _, state := range []string{"succeeded", "failed"} {
		t.Run(state, func(t *testing.T) {
			l := newLab(t)
			i := l.node("arm64", "disabled", "registered", true)
			b := l.batch(l.new)
			w := l.intent(i)
			r := result(w, state)
			current := l.new
			if state == "failed" {
				r = d062Result(t, l, w, "pre-health-failure")
				current = l.old
			}
			if l.get(b.BatchId).Targets[0].State != "queued" {
				t.Fatal("expected queued job")
			}
			raw := payload(t, r)
			ack, err := l.s.SubmitResult(l.ctx, i, raw)
			if err != nil || !ack.Accepted || ack.AcceptedAsHistory {
				t.Fatal(ack, err)
			}
			target := l.get(b.BatchId).Targets[0]
			var ledger string
			if err = l.s.pool.QueryRow(l.ctx, `SELECT state FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&ledger); err != nil || ledger != state || string(target.State) != state || target.NeedsAttention != (state == "failed") || !target.Unconfirmed || nullableValue(target.CurrentReleaseId) != current {
				t.Fatal(ledger, target, err)
			}
			replay, err := l.s.SubmitResult(l.ctx, i, raw)
			if err != nil || replay != ack {
				t.Fatal(replay, err)
			}
			var publication string
			if err = l.s.pool.QueryRow(l.ctx, `SELECT publication_intent FROM exits WHERE node_id=$1`, i.NodeID).Scan(&publication); err != nil || publication != "disabled" {
				t.Fatal(publication, err)
			}
		})
	}
}

func TestAT26_DirectRollbackPreservesEvidenceFences(t *testing.T) {
	for _, mode := range []string{"bad_health", "wrong_generation", "superseded", "terminal", "payload_conflict"} {
		t.Run(mode, func(t *testing.T) {
			l, i, b, w, r := at26DirectRollback(t)
			wantState := api.UpdateJobState("queued")
			var wantError apierr.Code
			var newer *Work
			switch mode {
			case "bad_health":
				health := nullableValue(r.Health)
				health.TunnelProbe = "failed"
				r.Health.Set(health)
				wantError = apierr.SchemaIncompatible
			case "wrong_generation":
				r.Generation++
				wantError = apierr.StaleGeneration
			case "superseded":
				l.batch(w.Intent.ReleaseId)
				newer = l.intent(i)
				wantState = api.UpdateJobState("superseded")
				// H-3: seed the new current target with agreeing recorded last-good.
				if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: newer.Intent.Generation, CurrentReleaseID: &newer.Intent.ReleaseId, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
					t.Fatal(err)
				}
			case "terminal":
				l.report(i, w, "succeeded")
				wantState = api.UpdateJobState("succeeded")
			case "payload_conflict":
				if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
					t.Fatal(err)
				}
				r.BootId = uuid.New()
				wantState = api.UpdateJobState("rolled_back")
				wantError = apierr.AckConflict
			}
			var before string
			if err := l.s.pool.QueryRow(l.ctx, `SELECT observed_evidence::text FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&before); err != nil {
				t.Fatal(err)
			}
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			if wantError != "" {
				if !errors.Is(err, apierr.New(wantError, "")) {
					t.Fatal(ack, err)
				}
			} else if err != nil || !ack.Accepted || !ack.AcceptedAsHistory || ack.AppliedToCurrentObservation {
				t.Fatal("old or terminal job must remain history", ack, err)
			}
			var after string
			if err = l.s.pool.QueryRow(l.ctx, `SELECT observed_evidence::text FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&after); err != nil || before != after || l.get(b.BatchId).Targets[0].State != wantState {
				t.Fatal("stale or invalid result rewrote ledger evidence", err)
			}
			if mode == "terminal" || newer != nil {
				view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
				if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != w.Intent.ReleaseId {
					t.Fatal("late rollback demoted current release", view, err)
				}
			}
			if newer != nil {
				var jobID uuid.UUID
				if err = l.s.pool.QueryRow(l.ctx, `SELECT current_job_id FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&jobID); err != nil || jobID != newer.Intent.JobId {
					t.Fatal("late rollback moved job pointer", jobID, err)
				}
			}
		})
	}
}
