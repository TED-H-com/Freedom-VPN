//go:build integration

package fleetupdate

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// TestReviewR17_LaterInstallerOutcomeCanCorrectTheRecordedRelease is the
// D-110 retry. A pre-health failure on a null baseline seeds the old release.
// The installer then succeeds for the same release, manifest and incarnation,
// which is the same derived job id and a different result. The recorded
// version has to become the release the node is running.
func TestReviewR17_LaterInstallerOutcomeCanCorrectTheRecordedRelease(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	installerNullBaseline(t, l, i)
	failed := installerOutcome(installerResult(t, l, i), api.UpdateResultStateFailed, l.old)
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, failed))
	if err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	installerVersions(t, l, i, l.old, l.old)
	detail, _, err := nodesResource(t, l, i)
	if err != nil {
		t.Fatal(err)
	}

	success := installerResult(t, l, i)
	if success.JobId != failed.JobId || success.ResultId == failed.ResultId {
		t.Fatalf("fixture job=%s/%s result=%s/%s", success.JobId, failed.JobId, success.ResultId, failed.ResultId)
	}
	got, err := l.s.SubmitResult(l.ctx, i, payload(t, success))
	current, good := installerVersionIDs(t, l, i)
	if err != nil || !got.Accepted || current != l.new || good != l.new {
		t.Errorf("TODO(review): later eligible installer success after a pre-health seed was not recorded (ack=%+v err=%v current=%s last_good=%s want %s); D-110's one immutable receipt per release, manifest and incarnation leaves the fleet on the failed baseline", got, err, current, good, l.new)
	} else if after, _, err := nodesResource(t, l, i); err != nil || after <= detail {
		t.Fatal("corrected version did not advance the node ETag", after, detail, err)
	}

	// The inverse on a fresh incarnation: an accepted success must still be
	// correctable by a later eligible rollback of that same derived job.
	l = newLabState(t)
	i = l.node("amd64", "disabled", "registered", false)
	installerNullBaseline(t, l, i)
	okResult := installerResult(t, l, i)
	if ack, err = l.s.SubmitResult(l.ctx, i, payload(t, okResult)); err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	installerVersions(t, l, i, l.new, l.new)
	rolled := installerOutcome(installerResult(t, l, i), api.UpdateResultStateRolledBack, l.old)
	if rolled.JobId != okResult.JobId {
		t.Fatal("rollback job id drifted", rolled.JobId, okResult.JobId)
	}
	got, err = l.s.SubmitResult(l.ctx, i, payload(t, rolled))
	current, good = installerVersionIDs(t, l, i)
	if err != nil || !got.Accepted || current != l.old || good != l.old {
		t.Fatalf("TODO(review): later eligible installer rollback on the same incarnation was not recorded (ack=%+v err=%v current=%s last_good=%s want %s)", got, err, current, good, l.old)
	}
}

func nodesResource(t *testing.T, l *lab, i Identity) (int64, int64, error) {
	t.Helper()
	var version int64
	err := l.s.pool.QueryRow(l.ctx, `SELECT resource_version FROM nodes WHERE id=$1`, i.NodeID).Scan(&version)
	return version, version, err
}

func installerVersionIDs(t *testing.T, l *lab, i Identity) (uuid.UUID, uuid.UUID) {
	t.Helper()
	var current, good uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT current_release_id, last_good_release_id FROM nodes WHERE id=$1`, i.NodeID).Scan(&current, &good); err != nil {
		t.Fatal(err)
	}
	return current, good
}

// TestReviewR17_InstallerJobIDIsNotAChosenAlias proves a node cannot steer
// the derived id onto a central job, a random UUID, another node's row, or
// a second payload for an id that was already accepted.
func TestReviewR17_InstallerJobIDIsNotAChosenAlias(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	queued := l.node("amd64", "disabled", "registered", true)
	l.batch(l.new)
	work := l.intent(queued)

	hijack := installerResult(t, l, queued)
	hijack.JobId = work.Intent.JobId
	ack, err := l.s.SubmitResult(l.ctx, queued, payload(t, hijack))
	installerError(t, ack, err, apierr.StateConflict, 409)
	var apiError *apierr.Error
	if !errors.As(err, &apiError) || apiError.Details["reason"] == "job_id_mismatch" {
		t.Fatalf("central-job collision must stay a plain state conflict, got %v", err)
	}
	var rows int
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results WHERE node_id=$1`, queued.NodeID).Scan(&rows); err != nil || rows != 0 {
		t.Fatal(rows, err)
	}

	randomID := installerResult(t, l, i)
	randomID.JobId = uuid.New()
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, randomID))
	installerError(t, ack, err, apierr.StateConflict, 409)
	if !errors.As(err, &apiError) || apiError.Details["reason"] != "job_id_mismatch" {
		t.Fatalf("random job id: %v", err)
	}
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&rows); err != nil || rows != 0 {
		t.Fatal(rows, err)
	}

	forged := installerResult(t, l, i)
	forged.ManifestSha256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	forged.JobId = installerjob.ID(forged.RequestedReleaseId, forged.ManifestSha256, i.IncarnationID)
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, forged))
	installerError(t, ack, err, apierr.HashMismatch, 422)

	first := installerResult(t, l, i)
	raw := payload(t, first)
	ack, err = l.s.SubmitResult(l.ctx, i, raw)
	if err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	replay, err := l.s.SubmitResult(l.ctx, i, raw)
	if err != nil || replay.ResultId != ack.ResultId || !replay.Accepted {
		t.Fatal("stored replay", replay, err)
	}
	otherPayload := first
	otherPayload.ResultId = uuid.New()
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, otherPayload))
	installerError(t, ack, err, apierr.AckConflict, 409)
	installerVersions(t, l, i, l.new, l.new)

	other := l.node("amd64", "disabled", "registered", false)
	second := installerResult(t, l, other)
	if second.JobId == first.JobId {
		t.Fatal("two incarnations derived the same job id")
	}
	if ack, err = l.s.SubmitResult(l.ctx, other, payload(t, second)); err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&rows); err != nil || rows != 2 {
		t.Fatal(rows, err)
	}
}

// TestReviewR17_InstallerRollbackAndSeedRefuseIneligibleRelease proves a
// revoked or unpublished release cannot become current through rollback or
// through D-111's null baseline.
func TestReviewR17_InstallerRollbackAndSeedRefuseIneligibleRelease(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{l.old})
	rolled := installerOutcome(installerResult(t, l, i), api.UpdateResultStateRolledBack, l.old)
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, rolled))
	installerError(t, ack, err, apierr.MinimumVersionViolation, 422)
	installerVersions(t, l, i, l.old, l.old)
	var rows int
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&rows); err != nil || rows != 0 {
		t.Fatal(rows, err)
	}

	l = newLabState(t)
	i = l.node("amd64", "disabled", "registered", false)
	installerNullBaseline(t, l, i)
	l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{l.old})
	seed := installerOutcome(installerResult(t, l, i), api.UpdateResultStateFailed, l.old)
	ack, err = l.s.SubmitResult(l.ctx, i, payload(t, seed))
	if err == nil || ack.Accepted {
		t.Fatal("revoked baseline was seeded", ack, err)
	}
	var current, good *uuid.UUID
	if err = l.s.pool.QueryRow(l.ctx, `SELECT current_release_id, last_good_release_id FROM nodes WHERE id=$1`, i.NodeID).Scan(&current, &good); err != nil || current != nil || good != nil {
		t.Fatal(current, good, err)
	}
}

// TestReviewR17_LegacyJobIDDoesNotBlockTheDerivedID proves a pre-D-110 row
// under a different job id does not make the derived id a second copy of the
// same evidence, and does not prevent the derived receipt from moving the
// recorded version.
func TestReviewR17_LegacyJobIDDoesNotBlockTheDerivedID(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	legacy := installerOutcome(installerResult(t, l, i), api.UpdateResultStateFailed, l.old)
	legacy.JobId = uuid.New()
	raw := payload(t, legacy)
	var stored api.UpdateResult
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	sum, err := hex.DecodeString(stored.PayloadHash)
	if err != nil || len(sum) != 32 {
		t.Fatal(stored.PayloadHash, err)
	}
	ackRaw, err := json.Marshal(api.UpdateResultAck{Accepted: true, ResultId: stored.ResultId, ReceivedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = l.s.pool.Exec(l.ctx, `INSERT INTO installer_update_results(node_id,job_id,incarnation_id,release_id,state,payload,payload_hash,ack,received_at) VALUES($1,$2,$3,$4,'failed',$5,$6,$7,clock_timestamp())`, i.NodeID, legacy.JobId, i.IncarnationID, l.old, raw, sum, ackRaw); err != nil {
		t.Fatal(err)
	}
	derived := installerResult(t, l, i)
	if derived.JobId == legacy.JobId {
		t.Fatal("derived id collided with the legacy fixture")
	}
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, derived))
	if err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	installerVersions(t, l, i, l.new, l.new)
	var rows int
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results WHERE node_id=$1`, i.NodeID).Scan(&rows); err != nil || rows != 2 {
		t.Fatal("legacy and derived receipts", rows, err)
	}
	var latest uuid.UUID
	if err = l.s.pool.QueryRow(l.ctx, `SELECT job_id FROM installer_update_results WHERE node_id=$1 ORDER BY received_at DESC, job_id DESC LIMIT 1`, i.NodeID).Scan(&latest); err != nil || latest != derived.JobId {
		t.Fatal("latest projection is not the derived receipt", latest, derived.JobId, err)
	}
}
