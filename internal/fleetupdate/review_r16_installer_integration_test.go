//go:build integration

package fleetupdate

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestReviewR16_InstallerRollbackMovesRecordedVersion(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	ok := installerResult(t, l, i)
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, ok)); err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	installerVersions(t, l, i, l.new, l.new)

	rb := installerResult(t, l, i)
	rb.JobId = uuid.New()
	rb.ResultId = uuid.New()
	rb.State = api.UpdateResultStateRolledBack
	rb.CurrentReleaseId.Set(l.old)
	rb.PreviousReleaseId.Set(l.new)
	rb.ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, rb))
	if err != nil || !ack.Accepted {
		t.Fatalf("healthy rollback evidence was not accepted: %+v %v", ack, err)
	}
	var current, good string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT current_release_id::text, last_good_release_id::text FROM nodes WHERE id=$1`, i.NodeID).Scan(&current, &good); err != nil {
		t.Fatal(err)
	}
	if current != l.old.String() || good != l.old.String() {
		t.Fatalf("TODO(review): accepted installer rolled_back left current_release_id=%s last_good_release_id=%s; want both %s (the release the node is running)", current, good, l.old)
	}
}

func TestReviewR16_InstallerResultCardinalityIsBounded(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	const bound = 8
	accepted := 0
	var last error
	for n := 0; n < bound+1; n++ {
		r := installerResult(t, l, i)
		r.JobId = uuid.New()
		r.ResultId = uuid.New()
		if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
			last = err
			break
		}
		accepted++
	}
	var rows, audits, version int
	if err := l.s.pool.QueryRow(l.ctx, `SELECT (SELECT count(*) FROM installer_update_results WHERE node_id=$1), (SELECT count(*) FROM audit_events WHERE action='update_result.accept' AND after_diff->>'source'='installer'), (SELECT resource_version FROM nodes WHERE id=$1)`, i.NodeID).Scan(&rows, &audits, &version); err != nil {
		t.Fatal(err)
	}
	if accepted > bound {
		t.Fatalf("TODO(review): one node credential inserted %d installer_update_results rows, %d installer accept audits, resource_version=%d, with no cardinality rejection (last=%v)", rows, audits, version, last)
	}
}

func TestReviewR16_InstallerCannotSelfPromote(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	queued := l.node("amd64", "disabled", "registered", true)
	batch := l.batch(l.new)
	work := l.intent(queued)
	var jobID uuid.UUID
	var jobState string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT current_job_id, (SELECT state FROM update_jobs WHERE id=fleet_node_state.current_job_id) FROM fleet_node_state WHERE node_id=$1`, queued.NodeID).Scan(&jobID, &jobState); err != nil {
		t.Fatal(err)
	}
	if jobID != work.Intent.JobId {
		t.Fatalf("intent job = %s stored %s", work.Intent.JobId, jobID)
	}
	hijack := installerResult(t, l, queued)
	hijack.JobId = work.Intent.JobId
	ack, err := l.s.SubmitResult(l.ctx, queued, payload(t, hijack))
	installerError(t, ack, err, apierr.StateConflict, 409)
	var jobIDAfter uuid.UUID
	var jobStateAfter string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT current_job_id, (SELECT state FROM update_jobs WHERE id=fleet_node_state.current_job_id) FROM fleet_node_state WHERE node_id=$1`, queued.NodeID).Scan(&jobIDAfter, &jobStateAfter); err != nil {
		t.Fatal(err)
	}
	if jobIDAfter != jobID || jobStateAfter != jobState || jobStateAfter == "succeeded" {
		t.Fatalf("installer result cleared or completed the central job: before=%s/%s after=%s/%s", jobID, jobState, jobIDAfter, jobStateAfter)
	}
	if got := l.get(batch.BatchId); got.Counts.Succeeded != 0 || got.State == "completed" || got.State == "completed_with_exceptions" {
		t.Fatalf("installer result completed the batch: %+v", got.Counts)
	}

	unknown := installerResult(t, l, i)
	unknown.RequestedReleaseId = uuid.New()
	unknown.CurrentReleaseId.Set(unknown.RequestedReleaseId)
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, unknown))
	installerError(t, ack, err, apierr.ResourceNotFound, 404)
	installerVersions(t, l, i, l.old, l.old)

	l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
	unpub := installerResult(t, l, i)
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, unpub))
	installerError(t, ack, err, apierr.StateConflict, 409)
	installerVersions(t, l, i, l.old, l.old)

	// Re-approving a release that was never revoked is allowed. Revocation
	// itself cannot be undone, so the revoked-release case runs last.
	l.support([]uuid.UUID{l.old, l.new}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
	body := installerResult(t, l, i)
	body.IncarnationId = uuid.New()
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, body))
	installerError(t, ack, err, apierr.IncarnationMismatch, 403)

	ok := installerResult(t, l, i)
	raw := payload(t, ok)
	ack, err = l.s.SubmitResult(l.ctx, i, raw)
	if err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	installerVersions(t, l, i, l.new, l.new)
	var version int
	if err := l.s.pool.QueryRow(l.ctx, `SELECT resource_version FROM nodes WHERE id=$1`, i.NodeID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	replay, err := l.s.SubmitResult(l.ctx, i, raw)
	if err != nil || replay.ResultId != ack.ResultId || replay.ReceivedAt != ack.ReceivedAt {
		t.Fatalf("exact replay = %+v %v want %+v", replay, err, ack)
	}
	var versionAfter int
	if err := l.s.pool.QueryRow(l.ctx, `SELECT resource_version FROM nodes WHERE id=$1`, i.NodeID).Scan(&versionAfter); err != nil || versionAfter != version {
		t.Fatalf("exact replay moved resource_version %d -> %d (%v)", version, versionAfter, err)
	}
	conflict := ok
	conflict.ResultId = uuid.New()
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, conflict))
	installerError(t, ack, err, apierr.AckConflict, 409)
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, conflict))
	installerError(t, ack, err, apierr.AckConflict, 409)

	stale := i
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, stale.IncarnationID); err != nil {
		t.Fatal(err)
	}
	next := l.enroll(i, "amd64")
	ack, err = l.s.SubmitResult(l.ctx, stale, raw)
	installerError(t, ack, err, apierr.IncarnationMismatch, 403)
	ack, err = l.s.SubmitResult(l.ctx, next, raw)
	installerError(t, ack, err, apierr.IncarnationMismatch, 403)
	var rows int
	if err := l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results WHERE node_id=$1`, i.NodeID).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("incarnation swap stored extra evidence: %d %v", rows, err)
	}
	installerVersions(t, l, i, l.new, l.new)

	stayed := l.node("amd64", "disabled", "registered", false)
	l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
	revoked := installerResult(t, l, stayed)
	ack, err = l.s.SubmitResult(l.ctx, stayed, payload(t, revoked))
	installerError(t, ack, err, apierr.StateConflict, 409)
	installerVersions(t, l, stayed, l.old, l.old)
	installerVersions(t, l, i, l.new, l.new)
}
