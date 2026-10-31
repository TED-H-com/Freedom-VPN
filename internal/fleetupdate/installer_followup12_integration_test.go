//go:build integration

package fleetupdate

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func installerCorrectionPair(t *testing.T, l *lab, i Identity, seed bool) (api.UpdateResult, api.UpdateResult) {
	t.Helper()
	first, next := installerResult(t, l, i), installerResult(t, l, i)
	if seed {
		installerNullBaseline(t, l, i)
		first = installerOutcome(first, api.UpdateResultStateFailed, l.old)
	} else {
		next = installerOutcome(next, api.UpdateResultStateRolledBack, l.old)
	}
	return first, next
}

func installerAccept(t *testing.T, l *lab, i Identity, r api.UpdateResult) api.UpdateResultAck {
	t.Helper()
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
	if err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	return ack
}

func installerReceiptSnapshot(t *testing.T, l *lab, i Identity, sequence int) string {
	t.Helper()
	var raw string
	if err := l.s.pool.QueryRow(l.ctx, `SELECT to_jsonb(r)::text FROM installer_update_results r WHERE node_id=$1 AND outcome_sequence=$2`, i.NodeID, sequence).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func installerReceiptCounts(t *testing.T, l *lab, i Identity, want int) {
	t.Helper()
	var rows, audits int
	err := l.s.pool.QueryRow(l.ctx, `SELECT (SELECT count(*) FROM installer_update_results WHERE node_id=$1),(SELECT count(*) FROM audit_events WHERE action='update_result.accept' AND after_diff->>'source'='installer')`, i.NodeID).Scan(&rows, &audits)
	if err != nil || rows != want || audits != want {
		t.Fatal("receipt/audit count", rows, audits, want, err)
	}
}

func testInstallerCorrectionProjection(t *testing.T, seed bool) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			l := newLabState(t)
			i := l.node(arch, "disabled", "registered", false)
			first, next := installerCorrectionPair(t, l, i, seed)
			firstAck := installerAccept(t, l, i, first)
			original := installerReceiptSnapshot(t, l, i, 1)
			before, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			central := installerCentralSnapshot(t, l)
			ack := installerAccept(t, l, i, next)
			if !ack.AppliedToCurrentObservation || ack.AcceptedAsHistory || ack.ObservedUpdateGeneration != 0 || !ack.ReceivedAt.After(firstAck.ReceivedAt) {
				t.Fatal(ack)
			}
			want, _ := next.CurrentReleaseId.Get()
			installerVersions(t, l, i, want, want)
			after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || after.ResourceVersion != before.ResourceVersion+1 {
				t.Fatal("correction ETag", after.ResourceVersion, err)
			}
			summary, err := after.Update.Get()
			received, _ := summary.LastResultAt.Get()
			if err != nil || string(summary.State) != string(next.State) || summary.JobId != next.JobId || summary.Source == nil || *summary.Source != api.UpdateResultSourceInstaller || !received.Equal(ack.ReceivedAt) || !summary.BatchId.IsNull() {
				t.Fatal("update_view correction", summary, err)
			}
			if original != installerReceiptSnapshot(t, l, i, 1) || central != installerCentralSnapshot(t, l) {
				t.Fatal("correction rewrote original receipt or central work")
			}
			installerReceiptCounts(t, l, i, 2)
		})
	}
}

func TestD127_SuccessAfterSeedMovesBothVersions(t *testing.T) {
	testInstallerCorrectionProjection(t, true)
}
func TestD127_RollbackAfterSuccessMovesBothVersions(t *testing.T) {
	testInstallerCorrectionProjection(t, false)
}

func TestD127_ExactReplayOfEitherOutcomeReturnsStoredAck(t *testing.T) {
	for _, seed := range []bool{true, false} {
		t.Run(map[bool]string{true: "seed_success", false: "success_rollback"}[seed], func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			first, next := installerCorrectionPair(t, l, i, seed)
			a := installerAccept(t, l, i, first)
			b := installerAccept(t, l, i, next)
			before, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			original, correction := installerReceiptSnapshot(t, l, i, 1), installerReceiptSnapshot(t, l, i, 2)
			// Replays do not revalidate current release policy or project the old version.
			l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
			for _, pair := range []struct {
				result api.UpdateResult
				ack    api.UpdateResultAck
			}{{first, a}, {next, b}, {first, a}} {
				got := installerAccept(t, l, i, pair.result)
				if !reflect.DeepEqual(got, pair.ack) {
					t.Fatal("stored ACK changed", got, pair.ack)
				}
			}
			after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || after.ResourceVersion != before.ResourceVersion || !reflect.DeepEqual(after.Update, before.Update) {
				t.Fatal("replay changed node", err)
			}
			want, _ := next.CurrentReleaseId.Get()
			installerVersions(t, l, i, want, want)
			if original != installerReceiptSnapshot(t, l, i, 1) || correction != installerReceiptSnapshot(t, l, i, 2) {
				t.Fatal("replay rewrote receipt")
			}
			installerReceiptCounts(t, l, i, 2)
		})
	}
}

func TestD127_ThirdFlipIsAckConflict(t *testing.T) {
	for _, seed := range []bool{true, false} {
		t.Run(map[bool]string{true: "seed_success_rollback", false: "success_rollback_success"}[seed], func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			first, next := installerCorrectionPair(t, l, i, seed)
			installerAccept(t, l, i, first)
			installerAccept(t, l, i, next)
			third := installerResult(t, l, i)
			if seed {
				third = installerOutcome(third, api.UpdateResultStateRolledBack, l.old)
			}
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, third))
			installerError(t, ack, err, apierr.AckConflict, 409)
			want, _ := next.CurrentReleaseId.Get()
			installerVersions(t, l, i, want, want)
			installerReceiptCounts(t, l, i, 2)
		})
	}
}

func TestD127_OtherSecondEvidenceIsAckConflict(t *testing.T) {
	for _, kind := range []string{"same_outcome", "failure_after_success", "already_current_after_seed", "success_after_nonseed_failure", "success_after_health_failure", "reused_result_id", "rollback_after_already_current"} {
		t.Run(kind, func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			first, next := installerCorrectionPair(t, l, i, kind == "already_current_after_seed")
			switch kind {
			case "same_outcome":
				next = installerResult(t, l, i)
			case "failure_after_success":
				next = installerOutcome(next, api.UpdateResultStateFailed, l.new)
			case "already_current_after_seed":
				next.State = api.UpdateResultStateAlreadyCurrent
			case "success_after_nonseed_failure":
				first = installerOutcome(first, api.UpdateResultStateFailed, l.old)
				next = installerResult(t, l, i)
			case "success_after_health_failure":
				first.State = api.UpdateResultStateFailed
				first.ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
				next = installerResult(t, l, i)
			case "reused_result_id":
				next.ResultId = first.ResultId
			case "rollback_after_already_current":
				first.State = api.UpdateResultStateAlreadyCurrent
			}
			installerAccept(t, l, i, first)
			before, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, next))
			installerError(t, ack, err, apierr.AckConflict, 409)
			after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("conflict changed node", err)
			}
			installerReceiptCounts(t, l, i, 1)
		})
	}
}

func TestD127_IneligibleCorrectionDoesNotMoveVersions(t *testing.T) {
	for _, kind := range []string{"revoked_candidate", "unapproved_candidate", "revoked_rollback", "unsupported_rollback", "rollback_schema", "rollback_health", "success_health"} {
		t.Run(kind, func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			seed := kind == "revoked_candidate" || kind == "unapproved_candidate" || kind == "success_health"
			first, next := installerCorrectionPair(t, l, i, seed)
			installerAccept(t, l, i, first)
			code := apierr.MinimumVersionViolation
			switch kind {
			case "revoked_candidate":
				l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
			case "unapproved_candidate":
				l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
			case "revoked_rollback":
				l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{l.old})
			case "unsupported_rollback":
				l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{})
			case "rollback_schema":
				next.StateSchemaVersion++
				code = apierr.SchemaIncompatible
			case "rollback_health":
				h, _ := next.Health.Get()
				h.Processes = api.HealthCheckStatusFailed
				next.Health.Set(h)
				code = apierr.SchemaIncompatible
			case "success_health":
				h, _ := next.Health.Get()
				h.Processes = api.HealthCheckStatusFailed
				next.Health.Set(h)
				code = apierr.HealthCheckFailed
			}
			before, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, next))
			status := 422
			if kind == "success_health" {
				status = 500
			} // Existing HEALTH_CHECK_FAILED mapping.
			installerError(t, ack, err, code, status)
			after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("rejected correction changed node", err)
			}
			installerReceiptCounts(t, l, i, 1)
			if kind == "rollback_schema" || kind == "rollback_health" || kind == "success_health" {
				// Invalid evidence must not consume the single correction slot.
				valid := installerResult(t, l, i)
				if !seed {
					valid = installerOutcome(valid, api.UpdateResultStateRolledBack, l.old)
				}
				installerAccept(t, l, i, valid)
				installerReceiptCounts(t, l, i, 2)
			}
		})
	}
}

func TestD127_CorrectionKeepsNinthRandomJobBound(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	first, next := installerCorrectionPair(t, l, i, false)
	installerAccept(t, l, i, first)
	installerAccept(t, l, i, next)
	for range 9 {
		forged := installerResult(t, l, i)
		forged.JobId = uuid.New()
		ack, err := l.s.SubmitResult(l.ctx, i, payload(t, forged))
		installerError(t, ack, err, apierr.StateConflict, 409)
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.Details["reason"] != "job_id_mismatch" {
			t.Fatal(err)
		}
	}
	installerVersions(t, l, i, l.old, l.old)
	installerReceiptCounts(t, l, i, 2)
}

func TestD127_CorrectionRespectsCentralGenerationHighWater(t *testing.T) {
	for _, seed := range []bool{true, false} {
		t.Run(map[bool]string{true: "seed_success", false: "success_rollback"}[seed], func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", true)
			first, next := installerCorrectionPair(t, l, i, seed)
			installerAccept(t, l, i, first)
			l.batch(l.new)
			w := l.intent(i)
			current, _ := first.CurrentReleaseId.Get()
			if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: w.Intent.Generation + 1, CurrentReleaseID: &current, LastGoodReleaseID: &current, ObservedAt: time.Now().UTC()}); err != nil {
				t.Fatal(err)
			}
			before, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			central := installerCentralSnapshot(t, l)
			ack := installerAccept(t, l, i, next)
			if !ack.AcceptedAsHistory || ack.AppliedToCurrentObservation || ack.ObservedUpdateGeneration != w.Intent.Generation+1 {
				t.Fatal(ack)
			}
			installerVersions(t, l, i, current, current)
			after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || !reflect.DeepEqual(before, after) || central != installerCentralSnapshot(t, l) {
				t.Fatal("history changed central projection", err)
			}
			installerReceiptCounts(t, l, i, 2)
		})
	}
}

func TestD127_ConcurrentCorrectionIsOneImmutableReceipt(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	first, next := installerCorrectionPair(t, l, i, false)
	installerAccept(t, l, i, first)
	body := payload(t, next)
	var wg sync.WaitGroup
	acks := make([]api.UpdateResultAck, 6)
	errs := make([]error, 6)
	for j := range acks {
		wg.Go(func() { acks[j], errs[j] = l.s.SubmitResult(l.ctx, i, body) })
	}
	wg.Wait()
	for j := range acks {
		if errs[j] != nil || !acks[j].Accepted || !reflect.DeepEqual(acks[j], acks[0]) {
			t.Fatal(acks[j], errs[j])
		}
	}
	installerVersions(t, l, i, l.old, l.old)
	installerReceiptCounts(t, l, i, 2)
}

func TestD127_CorrectionProjectionSurvivesClockStep(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	first, next := installerCorrectionPair(t, l, i, false)
	raw := payload(t, first)
	if err := json.Unmarshal(raw, &first); err != nil {
		t.Fatal(err)
	}
	// TEST ONLY: an accepted receipt from before a backward central clock step.
	at := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	original := api.UpdateResultAck{Accepted: true, AppliedToCurrentObservation: true, ResultId: first.ResultId, ReceivedAt: at}
	if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO installer_update_results(node_id,job_id,incarnation_id,release_id,state,payload,payload_hash,ack,received_at) VALUES($1,$2,$3,$4,$5,$6,decode($7,'hex'),$8,$9)`, i.NodeID, first.JobId, i.IncarnationID, first.RequestedReleaseId, first.State, raw, first.PayloadHash, mustJSON(t, original), at); err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE nodes SET current_release_id=$2,last_good_release_id=$2,resource_version=resource_version+1 WHERE id=$1`, i.NodeID, l.new); err != nil {
		t.Fatal(err)
	}
	ack := installerAccept(t, l, i, next)
	if !ack.ReceivedAt.Equal(at.Add(time.Microsecond)) || !ack.AppliedToCurrentObservation {
		t.Fatal("correction did not sort after first outcome", ack)
	}
	detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := detail.Update.Get()
	if err != nil || string(summary.State) != string(api.UpdateResultStateRolledBack) {
		t.Fatal("node summary retained first outcome", summary, err)
	}
	installerVersions(t, l, i, l.old, l.old)
	if got := installerAccept(t, l, i, first); !reflect.DeepEqual(got, original) {
		t.Fatal("clock step rewrote first ACK", got)
	}
}

func TestD127_Migration192RoundTripPreservesEvidenceAndBounds(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	first, next := installerCorrectionPair(t, l, i, true)
	originalAck := installerAccept(t, l, i, first)
	original := installerReceiptSnapshot(t, l, i, 1)
	database := stdlib.OpenDBFromPool(l.s.pool)
	defer database.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.DownTo(l.ctx, 191); err != nil {
		t.Fatal("DownTo(191)", err)
	}
	var columns int
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name='installer_update_results' AND column_name='outcome_sequence'`).Scan(&columns); err != nil || columns != 0 {
		t.Fatal("192 column survived Down", columns, err)
	}
	if _, err = provider.Up(l.ctx); err != nil {
		t.Fatal("Up", err)
	}
	if original != installerReceiptSnapshot(t, l, i, 1) {
		t.Fatal("round-trip changed existing receipt")
	}
	if got := installerAccept(t, l, i, first); !reflect.DeepEqual(got, originalAck) {
		t.Fatal("round-trip changed ACK", got)
	}
	installerAccept(t, l, i, next)
	// A duplicate key or a sequence outside 1..2 cannot extend the evidence stream.
	for _, seq := range []int{1, 2, 0, 3} {
		_, err = l.s.pool.Exec(l.ctx, `INSERT INTO installer_update_results(node_id,job_id,incarnation_id,release_id,state,payload,payload_hash,ack,received_at,outcome_sequence) SELECT node_id,job_id,incarnation_id,release_id,state,jsonb_set(payload,'{result_id}',to_jsonb($3::text)),payload_hash,ack,received_at,$1 FROM installer_update_results WHERE node_id=$2 AND outcome_sequence=1`, seq, i.NodeID, uuid.NewString())
		var pe *pgconn.PgError
		want := "23514"
		if seq == 1 || seq == 2 {
			want = "23505"
		}
		if !errors.As(err, &pe) || pe.Code != want {
			t.Fatal("sequence constraint", seq, err)
		}
	}
	for _, query := range []string{`UPDATE installer_update_results SET state=state`, `DELETE FROM installer_update_results`, `TRUNCATE installer_update_results`} {
		if _, err = l.s.pool.Exec(l.ctx, query); err == nil {
			t.Fatal("lost immutable guard", query)
		}
	}
	// A populated correction cannot be downgraded by discarding history. Goose
	// rolls the failed unique-key restoration back, leaving both rows usable.
	if _, err = provider.DownTo(l.ctx, 191); err == nil {
		t.Fatal("destructive downgrade accepted")
	}
	if original != installerReceiptSnapshot(t, l, i, 1) {
		t.Fatal("failed downgrade changed history")
	}
	installerReceiptCounts(t, l, i, 2)
	installerVersions(t, l, i, l.new, l.new)
	if got := installerAccept(t, l, i, first); !reflect.DeepEqual(got, originalAck) {
		t.Fatal("history ACK changed", got)
	}
	conflict := installerResult(t, l, i)
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, conflict))
	installerError(t, ack, err, apierr.AckConflict, 409)
}

// NewInstallerCorrectionFixture exposes the real service and generated payloads
// to HTTP tests; no node handler or replay implementation is replaced.
func NewInstallerCorrectionFixture(t *testing.T, seed bool) (*Service, *nodes.Service, *pgxpool.Pool, []api.UpdateResult) {
	t.Helper()
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	first, next := installerCorrectionPair(t, l, i, seed)
	results := []api.UpdateResult{first, next}
	for j := range results {
		if err := json.Unmarshal(payload(t, results[j]), &results[j]); err != nil {
			t.Fatal(err)
		}
	}
	return l.s, nodes.NewService(l.s.pool), l.s.pool, results
}
