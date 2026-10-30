//go:build integration

package fleetupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// reviewObservation clones the contract example the way TestD062e does.
// Identity and the update block are rebound by the caller.
func reviewObservation(t *testing.T, l *lab, i Identity) (api.ObservationReport, *nodes.Service) {
	t.Helper()
	raw, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	report.NodeId, report.NodeIncarnationId, report.ReportId = i.NodeID, i.IncarnationID, uuid.New()
	report.ObservedAt = time.Now().UTC().Add(time.Hour)
	report.Update.JobId.SetNull()
	report.Update.Generation.SetNull()
	report.Update.State.SetNull()
	report.Update.Stage.SetNull()
	report.CurrentReleaseId.Set(l.old)
	report.LastGoodReleaseId.Set(l.old)
	report.Netbird.PeerId.SetNull()
	report.MemberForwardingEnabled = false
	report.AppliedConfigGeneration.SetNull()
	svc := nodes.NewService(l.s.pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, r api.ObservationReport) error {
		return l.s.ObserveReportTx(ctx, tx, i, r)
	}))
	return report, svc
}

func reviewHighWater(t *testing.T, l *lab, nodeID uuid.UUID) string {
	t.Helper()
	var generation string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT observed_update_generation::text FROM fleet_node_state WHERE node_id=$1`, nodeID).Scan(&generation); err != nil {
		t.Fatal(err)
	}
	return generation
}

func reviewJobAttention(t *testing.T, l *lab, jobID uuid.UUID) (state string, attention bool, release *uuid.UUID) {
	t.Helper()
	if err := l.s.pool.QueryRow(l.ctx, `SELECT state, needs_attention, actual_release_id FROM update_jobs WHERE id=$1`, jobID).Scan(&state, &attention, &release); err != nil {
		t.Fatal(err)
	}
	return state, attention, release
}

func TestReviewR9_D062_ResultIdentityReplayAndReceiptBounds(t *testing.T) {
	t.Run("same bytes are idempotent", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		r := result(w, "succeeded")
		raw := payload(t, r)
		ack, err := l.s.SubmitResult(l.ctx, i, raw)
		if err != nil || !ack.Accepted {
			t.Fatal(ack, err)
		}
		replay, err := l.s.SubmitResult(l.ctx, i, raw)
		if err != nil || replay.ResultId != ack.ResultId || !replay.Accepted || replay.AcceptedAsHistory {
			t.Fatal(replay, err)
		}
		if got := l.scalar(`SELECT count(*) FROM update_results WHERE result_id=$1`, r.ResultId); got != 1 {
			t.Fatal(got)
		}
		if got := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_result.accept' AND after_diff->>'result_id'=$1`, r.ResultId.String()); got != 1 {
			t.Fatal("replay wrote another accept audit", got)
		}
		if got := l.scalar(`SELECT count(*) FROM update_result_conflicts WHERE result_id=$1`, r.ResultId); got != 0 {
			t.Fatal(got)
		}
	})

	t.Run("changed bytes conflict without overwrite and audits grow", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		r := result(w, "succeeded")
		raw := payload(t, r)
		if _, err := l.s.SubmitResult(l.ctx, i, raw); err != nil {
			t.Fatal(err)
		}
		var storedPayload, storedAck []byte
		if err := l.s.pool.QueryRow(l.ctx, `SELECT payload, ack FROM update_results WHERE result_id=$1`, r.ResultId).Scan(&storedPayload, &storedAck); err != nil {
			t.Fatal(err)
		}
		var lastBad []byte
		for n := 0; n < 4; n++ {
			r.BootId = uuid.New()
			lastBad = payload(t, r)
			_, err := l.s.SubmitResult(l.ctx, i, lastBad)
			if !errors.Is(err, apierr.New(apierr.AckConflict, "")) {
				t.Fatal(err)
			}
		}
		var payloadNow, ackNow []byte
		if err := l.s.pool.QueryRow(l.ctx, `SELECT payload, ack FROM update_results WHERE result_id=$1`, r.ResultId).Scan(&payloadNow, &ackNow); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(payloadNow, storedPayload) || !bytes.Equal(ackNow, storedAck) {
			t.Fatal("TODO(review): conflicting bytes overwrote the stored result or its ack")
		}
		_, err := l.s.pool.Exec(l.ctx, `UPDATE update_results SET ack='{}'::jsonb WHERE result_id=$1`, r.ResultId)
		if err == nil {
			t.Fatal("TODO(review): rejection/result ack row accepted an update")
		}
		for n := 0; n < 5; n++ {
			if _, err = l.s.SubmitResult(l.ctx, i, lastBad); !errors.Is(err, apierr.New(apierr.AckConflict, "")) {
				t.Fatal(err)
			}
		}
		conflicts := l.scalar(`SELECT count(*) FROM update_result_conflicts WHERE result_id=$1`, r.ResultId)
		audits := l.scalar(`SELECT count(*) FROM audit_events WHERE action='update_result.conflict' AND after_diff->>'result_id'=$1`, r.ResultId.String())
		// One result id should keep a single conflict fact. Distinct hashes and
		// replays of one hash must not append rows or audits without a bound.
		if conflicts != 1 || audits != 1 {
			t.Fatalf("TODO(review): result %s payload and ack stayed immutable, but 4 distinct conflicting payloads plus 5 replays of the last one stored %d update_result_conflicts rows and %d update_result.conflict audits", r.ResultId, conflicts, audits)
		}
	})

	t.Run("other node job wrong incarnation and unknown attempt", func(t *testing.T) {
		l := newLabState(t)
		a := l.node("amd64", "unpublished", "registered", true)
		b := l.node("arm64", "unpublished", "registered", true)
		batch := l.batch(l.new)
		wb := l.intent(b)
		foreign := result(wb, "succeeded")
		foreign.NodeId = a.NodeID
		foreign.IncarnationId = a.IncarnationID
		_, err := l.s.SubmitResult(l.ctx, a, payload(t, foreign))
		if !errors.Is(err, apierr.New(apierr.IncarnationMismatch, "")) {
			t.Fatal(err)
		}
		if got := l.scalar(`SELECT count(*) FROM update_results WHERE job_id=$1`, wb.Intent.JobId); got != 0 {
			t.Fatal("other-node job stored a receipt", got)
		}

		old := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(old)
		if _, err = l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, old.IncarnationID); err != nil {
			t.Fatal(err)
		}
		_ = l.enroll(Identity{NodeID: old.NodeID}, "amd64")
		staleID := result(w, "succeeded")
		_, err = l.s.SubmitResult(l.ctx, old, payload(t, staleID))
		if !errors.Is(err, apierr.New(apierr.IncarnationMismatch, "")) {
			t.Fatal("superseded incarnation", err)
		}
		if got := l.scalar(`SELECT count(*) FROM update_results WHERE result_id=$1`, staleID.ResultId); got != 0 {
			t.Fatal("superseded incarnation stored a receipt", got)
		}

		wa := l.intent(a)
		unknown := result(wa, "succeeded")
		unknown.Attempt = wa.Intent.Attempt + 1
		_, err = l.s.SubmitResult(l.ctx, a, payload(t, unknown))
		if !errors.Is(err, apierr.New(apierr.StaleGeneration, "")) {
			t.Fatal(err)
		}
		var ackRaw []byte
		if err = l.s.pool.QueryRow(l.ctx, `SELECT ack FROM update_results WHERE result_id=$1`, unknown.ResultId).Scan(&ackRaw); err != nil {
			t.Fatal(err)
		}
		var ackDoc map[string]any
		if err = json.Unmarshal(ackRaw, &ackDoc); err != nil {
			t.Fatal(err)
		}
		if len(ackDoc) != 1 {
			t.Fatalf("TODO(review): rejection ack exposed %v", ackDoc)
		}
		errObj, _ := ackDoc["error"].(map[string]any)
		for key := range errObj {
			switch key {
			case "code", "message", "retryable", "details":
			default:
				t.Fatalf("TODO(review): rejection ack field %q is outside the bounded vocabulary: %s", key, ackRaw)
			}
		}
		if errObj["code"] != string(apierr.StaleGeneration) {
			t.Fatal(ackRaw)
		}
		if bytes.Contains(ackRaw, []byte(unknown.BootId.String())) {
			t.Fatalf("TODO(review): rejection ack echoes the payload: %s", ackRaw)
		}
		page := l.get(batch.BatchId)
		projected, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(projected, []byte(unknown.BootId.String())) || bytes.Contains(projected, []byte(unknown.ResultId.String())) {
			t.Fatalf("TODO(review): admin batch projection exposes the rejected result: %s", projected)
		}
		target := page.Targets[0]
		if target.State == "succeeded" || page.Counts.Succeeded != 0 {
			t.Fatal(page.Counts, target.State)
		}
		if _, err = l.s.pool.Exec(l.ctx, `UPDATE update_results SET payload='{}'::jsonb WHERE result_id=$1`, unknown.ResultId); err == nil {
			t.Fatal("TODO(review): rejection receipt payload was mutated")
		}
	})
}

func TestReviewR9_D062_EvidenceCompleteness(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	l.batch(l.new)
	w := l.intent(i)

	t.Run("succeeded with null health", func(t *testing.T) {
		r := result(w, "succeeded")
		r.Health.SetNull()
		_, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.HTTPStatus() < 400 || ae.HTTPStatus() >= 500 {
			t.Fatal(err)
		}
		state, attention, _ := reviewJobAttention(t, l, w.Intent.JobId)
		if state == "succeeded" || attention {
			t.Fatalf("TODO(review): null health changed the job state=%s attention=%v", state, attention)
		}
		if got := l.scalar(`SELECT count(*) FROM update_results WHERE result_id=$1 AND ack->>'error' IS NULL`, r.ResultId); got != 0 {
			t.Fatal("null health was stored as an acceptance", got)
		}
	})

	t.Run("failed at check without error code", func(t *testing.T) {
		fresh := newLabState(t)
		node := fresh.node("amd64", "unpublished", "registered", true)
		fresh.batch(fresh.new)
		work := fresh.intent(node)
		r := result(work, "failed")
		r.Stage.Set(api.UpdateStageCheck)
		r.ErrorCode.SetNull()
		r.CurrentReleaseId.Set(fresh.new)
		r.Health.Set(api.UpdateHealth{
			Processes: api.HealthCheckStatusFailed, IdentityAndState: api.HealthCheckStatusFailed,
			InternetOnly: api.HealthCheckStatusFailed, DnsPolicy: api.HealthCheckStatusFailed, TunnelProbe: api.HealthCheckStatusFailed,
		})
		_, err := fresh.s.SubmitResult(fresh.ctx, node, payload(t, r))
		view, viewErr := fresh.s.NodeVersionView(fresh.ctx, node.NodeID)
		state, attention, _ := reviewJobAttention(t, fresh, work.Intent.JobId)
		if err == nil {
			t.Fatalf("TODO(review): central accepted failed stage=check with null error_code; job=%s needs_attention=%v view=%+v viewErr=%v", state, attention, view, viewErr)
		}
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.HTTPStatus() < 400 {
			t.Fatal(err)
		}
		if state == "failed" || attention {
			t.Fatal(state, attention)
		}
	})

	t.Run("rolled back health is the candidate release", func(t *testing.T) {
		fresh := newLabState(t)
		node := fresh.node("amd64", "unpublished", "registered", true)
		batch := fresh.batch(fresh.new)
		work := fresh.intent(node)
		if _, err := fresh.s.SubmitResult(fresh.ctx, node, payload(t, result(work, "checking"))); err != nil {
			t.Fatal(err)
		}
		r := d062Result(t, fresh, work, "rollback")
		r.CurrentReleaseId.Set(work.Intent.ReleaseId)
		_, err := fresh.s.SubmitResult(fresh.ctx, node, payload(t, r))
		if err == nil {
			view, _ := fresh.s.NodeVersionView(fresh.ctx, node.NodeID)
			t.Fatalf("TODO(review): rolled_back whose current_release_id is the candidate was accepted; target=%+v view=%+v", fresh.get(batch.BatchId).Targets[0], view)
		}
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.HTTPStatus() < 400 || ae.HTTPStatus() >= 500 {
			t.Fatal(err)
		}
	})

	t.Run("candidate health on success and omitted on rollback", func(t *testing.T) {
		fresh := newLabState(t)
		node := fresh.node("arm64", "unpublished", "registered", true)
		batch := fresh.batch(fresh.new)
		work := fresh.intent(node)
		ok := result(work, "succeeded")
		ok.CandidateHealth.Set(api.UpdateHealth{
			Processes: api.HealthCheckStatusPassed, IdentityAndState: api.HealthCheckStatusPassed,
			InternetOnly: api.HealthCheckStatusPassed, DnsPolicy: api.HealthCheckStatusPassed, TunnelProbe: api.HealthCheckStatusPassed,
		})
		if _, err := fresh.s.SubmitResult(fresh.ctx, node, payload(t, ok)); err != nil {
			t.Fatal(err)
		}
		if target := fresh.get(batch.BatchId).Targets[0]; target.State != "succeeded" || target.NeedsAttention {
			t.Fatal(target)
		}
		view, err := fresh.s.NodeVersionView(fresh.ctx, node.NodeID)
		if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != fresh.new || !view.ReleaseServiceable {
			t.Fatal(view, err)
		}

		again := newLabState(t)
		node = again.node("arm64", "unpublished", "registered", true)
		batch = again.batch(again.new)
		work = again.intent(node)
		if _, err = again.s.SubmitResult(again.ctx, node, payload(t, result(work, "checking"))); err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"omitted", "null"} {
			rolled := d062Result(t, again, work, "rollback")
			rolled.ResultId = uuid.New()
			if mode == "omitted" {
				rolled.CandidateHealth.SetUnspecified()
			} else {
				rolled.CandidateHealth.SetNull()
			}
			if _, err = again.s.SubmitResult(again.ctx, node, payload(t, rolled)); err != nil {
				t.Fatal(mode, err)
			}
		}
		target := again.get(batch.BatchId).Targets[0]
		view, err = again.s.NodeVersionView(again.ctx, node.NodeID)
		if err != nil || target.State != "rolled_back" || !target.NeedsAttention || view.CurrentReleaseID == nil || *view.CurrentReleaseID != again.old {
			t.Fatalf("TODO(review): rolled_back candidate_health absence changed outcome target=%+v view=%+v err=%v", target, view, err)
		}
	})

	t.Run("health block is schema bounded", func(t *testing.T) {
		raw := payload(t, result(w, "succeeded"))
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		health, _ := doc["health"].(map[string]any)
		health["extra"] = "xxxxxxxx"
		delete(doc, "payload_hash")
		body, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		sum, err := canonjson.HashHex(body, "payload_hash")
		if err != nil {
			t.Fatal(err)
		}
		doc["payload_hash"] = sum
		body, err = json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		before := l.scalar(`SELECT count(*) FROM update_results`)
		_, err = l.s.SubmitResult(l.ctx, i, body)
		if err == nil {
			t.Fatal("TODO(review): health member outside UpdateHealth was accepted")
		}
		if got := l.scalar(`SELECT count(*) FROM update_results`); got != before {
			t.Fatal("TODO(review): rejected health block was stored", got, before)
		}
	})
}

func TestReviewR9_D062_ObservationGenerationAndServingRelease(t *testing.T) {
	t.Run("null generation keeps the high water", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		b := l.batch(l.new)
		w := l.intent(i)
		l.report(i, w, "succeeded")
		before := reviewHighWater(t, l, i.NodeID)
		report, svc := reviewObservation(t, l, i)
		report.CurrentReleaseId.Set(l.old)
		report.LastGoodReleaseId.Set(l.old)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
			t.Fatal(err)
		}
		if got := reviewHighWater(t, l, i.NodeID); got != before || got == "0" {
			t.Fatalf("null generation changed high water from %s to %s", before, got)
		}
		view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
		target := l.get(b.BatchId).Targets[0]
		if err != nil || view.ObservedGeneration != nil || !target.Unconfirmed || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new {
			t.Fatal(view, target, err)
		}
	})

	t.Run("other node job at n plus five", func(t *testing.T) {
		l := newLabState(t)
		a := l.node("amd64", "unpublished", "registered", true)
		b := l.node("arm64", "unpublished", "registered", true)
		l.batch(l.new)
		wb := l.intent(b)
		before := reviewHighWater(t, l, a.NodeID)
		releaseBefore, err := l.s.NodeVersionView(l.ctx, a.NodeID)
		if err != nil {
			t.Fatal(err)
		}
		report, svc := reviewObservation(t, l, a)
		report.Update.JobId.Set(wb.Intent.JobId)
		report.Update.Generation.Set(wb.Intent.Generation + 5)
		report.Update.State.Set(api.UpdateJobStateSucceeded)
		report.Update.Stage.Set(api.UpdateStageReport)
		report.CurrentReleaseId.Set(l.new)
		_, err = svc.SubmitObservation(l.ctx, a.NodeID, a.IncarnationID, report)
		if !errors.Is(err, apierr.New(apierr.StaleGeneration, "")) {
			t.Fatal(err)
		}
		after, err := l.s.NodeVersionView(l.ctx, a.NodeID)
		if err != nil || reviewHighWater(t, l, a.NodeID) != before || after.CurrentReleaseID == nil || releaseBefore.CurrentReleaseID == nil || *after.CurrentReleaseID != *releaseBefore.CurrentReleaseID {
			t.Fatalf("TODO(review): foreign job observation stored or moved release: before=%+v after=%+v water=%s err=%v", releaseBefore, after, reviewHighWater(t, l, a.NodeID), err)
		}
		if got := l.scalar(`SELECT count(*) FROM node_observations WHERE report_id=$1`, report.ReportId); got != 0 {
			t.Fatal("rejected observation was stored", got)
		}
	})

	t.Run("stage and state outside the vocabulary", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		b := l.batch(l.new)
		w := l.intent(i)
		l.report(i, w, "succeeded")
		report, svc := reviewObservation(t, l, i)
		report.Update.JobId.Set(w.Intent.JobId)
		report.Update.Generation.Set(w.Intent.Generation)
		report.Update.State.Set(api.UpdateJobState("not-a-state"))
		report.Update.Stage.Set(api.UpdateStage("not-a-stage"))
		report.CurrentReleaseId.Set(l.new)
		report.LastGoodReleaseId.Set(l.new)
		body, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		wireErr := l.f.Validator.ValidateBody("ObservationReport", body)
		_, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report)
		target := l.get(b.BatchId).Targets[0]
		if err == nil {
			t.Fatalf("TODO(review): domain stored update state/stage outside the vocabulary (wire=%v) and unconfirmed=%v", wireErr, target.Unconfirmed)
		}
		if wireErr == nil {
			t.Fatal("wire schema accepted the vocabulary", err)
		}
		if !target.Unconfirmed {
			t.Fatal("rejected vocabulary confirmed the job", target)
		}
	})

	t.Run("previous incarnation cannot confirm", func(t *testing.T) {
		l := newLabState(t)
		old := l.node("amd64", "unpublished", "registered", true)
		if _, err := l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, old.IncarnationID); err != nil {
			t.Fatal(err)
		}
		next := l.enroll(Identity{NodeID: old.NodeID}, "amd64")
		b := l.batch(l.new)
		w := l.intent(next)
		l.report(next, w, "succeeded")
		report, svc := reviewObservation(t, l, old)
		report.Update.JobId.Set(w.Intent.JobId)
		report.Update.Generation.Set(w.Intent.Generation)
		report.Update.State.Set(api.UpdateJobStateSucceeded)
		report.Update.Stage.Set(api.UpdateStageReport)
		report.CurrentReleaseId.Set(l.new)
		_, err := svc.SubmitObservation(l.ctx, old.NodeID, old.IncarnationID, report)
		if !errors.Is(err, apierr.New(apierr.IdentityRevoked, "")) && !errors.Is(err, apierr.New(apierr.IncarnationMismatch, "")) {
			t.Fatal(err)
		}
		if !l.get(b.BatchId).Targets[0].Unconfirmed {
			t.Fatal("previous incarnation confirmed the current job")
		}
	})

	t.Run("heartbeat replaces the accepted serving release", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		b := l.batch(l.new)
		w := l.intent(i)
		l.report(i, w, "succeeded")
		report, svc := reviewObservation(t, l, i)
		report.Update.JobId.Set(w.Intent.JobId)
		report.Update.Generation.Set(w.Intent.Generation)
		report.Update.State.Set(api.UpdateJobStateSucceeded)
		report.Update.Stage.Set(api.UpdateStageReport)
		report.CurrentReleaseId.Set(l.old)
		report.LastGoodReleaseId.Set(l.old)
		if _, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
			t.Fatal(err)
		}
		view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
		target := l.get(b.BatchId).Targets[0]
		var observed *time.Time
		if scanErr := l.s.pool.QueryRow(l.ctx, `SELECT last_observed_at FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&observed); scanErr != nil {
			t.Fatal(scanErr)
		}
		applied := observed != nil && observed.UTC().Equal(report.ObservedAt.Truncate(time.Microsecond))
		if !applied {
			t.Fatalf("observation was not applied; proof inconclusive: observed=%v want=%v err=%v", observed, report.ObservedAt, err)
		}
		if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.new {
			t.Fatalf("TODO(review): generation-matched observation replaced the accepted serving release; view=%+v target_state=%s needs_attention=%v unconfirmed=%v", view, target.State, target.NeedsAttention, target.Unconfirmed)
		}
	})

	t.Run("heartbeat clears recovery required", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		l.batch(l.new)
		w := l.intent(i)
		failed := result(w, "failed")
		failed.Stage.Set(api.UpdateStageCheck)
		failed.ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
		failed.RecoveryRequired = presence.NewRequired(true)
		failed.CurrentReleaseId.Set(l.old)
		if _, err := l.s.SubmitResult(l.ctx, i, payload(t, failed)); err != nil {
			t.Fatal(err)
		}
		before, err := l.s.NodeVersionView(l.ctx, i.NodeID)
		if err != nil || !before.RecoveryRequired {
			t.Fatal(before, err)
		}
		report, svc := reviewObservation(t, l, i)
		report.Update.JobId.Set(w.Intent.JobId)
		report.Update.Generation.Set(w.Intent.Generation)
		report.Update.State.Set(api.UpdateJobStateFailed)
		report.Update.Stage.Set(api.UpdateStageCheck)
		report.CurrentReleaseId.Set(l.old)
		report.LastGoodReleaseId.Set(l.old)
		if _, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
			t.Fatal(err)
		}
		after, err := l.s.NodeVersionView(l.ctx, i.NodeID)
		if err != nil {
			t.Fatal(err)
		}
		if !after.RecoveryRequired {
			t.Fatalf("TODO(review): observation cleared recovery_required set by the accepted result; before=%+v after=%+v", before, after)
		}
	})

	t.Run("null job id at n plus five", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "unpublished", "registered", true)
		b := l.batch(l.new)
		w := l.intent(i)
		l.report(i, w, "succeeded")
		report, svc := reviewObservation(t, l, i)
		report.Update.JobId.SetNull()
		report.Update.Generation.Set(w.Intent.Generation + 5)
		report.Update.State.Set(api.UpdateJobState("not-a-state"))
		report.Update.Stage.Set(api.UpdateStage("not-a-stage"))
		report.CurrentReleaseId.Set(l.old)
		report.LastGoodReleaseId.Set(l.old)
		_, err := svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report)
		view, viewErr := l.s.NodeVersionView(l.ctx, i.NodeID)
		water := reviewHighWater(t, l, i.NodeID)
		target := l.get(b.BatchId).Targets[0]
		if err == nil && (water != w.Intent.Generation.String() || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new) {
			t.Fatalf("TODO(review): jobless generation N+5 was stored (wire vocabulary unchecked) and moved high-water to %s; view=%+v unconfirmed=%v err=%v", water, view, target.Unconfirmed, viewErr)
		}
		if err != nil && !errors.Is(err, apierr.New(apierr.StaleGeneration, "")) && !errors.Is(err, apierr.New(apierr.ValidationFailed, "")) {
			t.Fatal(err)
		}
	})
}
