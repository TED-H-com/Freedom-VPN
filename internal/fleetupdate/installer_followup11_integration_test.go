//go:build integration

package fleetupdate

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func installerNullBaseline(t *testing.T, l *lab, i Identity) {
	t.Helper()
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE nodes SET current_release_id=NULL,last_good_release_id=NULL,resource_version=resource_version+1 WHERE id=$1`, i.NodeID); err != nil {
		t.Fatal(err)
	}
}

func installerOutcome(r api.UpdateResult, state api.UpdateResultState, baseline uuid.UUID) api.UpdateResult {
	r.State = state
	switch state {
	case api.UpdateResultStateFailed:
		r.Stage.Set(api.UpdateStageVerify)
		r.ErrorCode.Set(api.ErrorCodeSIGNATUREINVALID)
		r.Health.SetNull()
		r.UsageCheckpoint.SetNull()
		r.CurrentReleaseId.Set(baseline)
	case api.UpdateResultStateRolledBack:
		r.CurrentReleaseId.Set(baseline)
		r.PreviousReleaseId.Set(r.RequestedReleaseId)
		r.ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
	}
	return r
}

func TestD111_InstallerNullBaseline(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		for _, state := range []api.UpdateResultState{api.UpdateResultStateFailed, api.UpdateResultStateRolledBack, api.UpdateResultStateSucceeded} {
			t.Run(arch+"/"+string(state), func(t *testing.T) {
				l := newLabState(t)
				i := l.node(arch, "disabled", "registered", false)
				installerNullBaseline(t, l, i)
				r := installerOutcome(installerResult(t, l, i), state, l.old)
				raw := payload(t, r)
				before := installerCentralSnapshot(t, l)
				detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
				if err != nil {
					t.Fatal(err)
				}
				ack, err := l.s.SubmitResult(l.ctx, i, raw)
				if err != nil || !ack.Accepted || !ack.AppliedToCurrentObservation || ack.AcceptedAsHistory || ack.ObservedUpdateGeneration != 0 {
					t.Fatal(ack, err)
				}
				want := l.old
				if state == api.UpdateResultStateSucceeded {
					want = l.new
				}
				installerVersions(t, l, i, want, want)
				if before != installerCentralSnapshot(t, l) {
					t.Fatal("seed changed central work")
				}
				after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
				if err != nil || after.ResourceVersion != detail.ResourceVersion+1 {
					t.Fatal("seed must advance ETag once", after.ResourceVersion, err)
				}
				summary, err := after.Update.Get()
				if err != nil || string(summary.State) != string(state) || summary.Source == nil || *summary.Source != api.UpdateResultSourceInstaller {
					t.Fatal(summary, err)
				}
				var stored []byte
				if err = l.s.pool.QueryRow(l.ctx, `SELECT payload FROM installer_update_results WHERE node_id=$1`, i.NodeID).Scan(&stored); err != nil {
					t.Fatal(err)
				}
				var evidence api.UpdateResult
				if err = json.Unmarshal(stored, &evidence); err != nil || evidence.State != state || !reflect.DeepEqual(evidence.ErrorCode, r.ErrorCode) || !reflect.DeepEqual(evidence.Health, r.Health) {
					t.Fatal("failure evidence changed", evidence, err)
				}
				replay, err := l.s.SubmitResult(l.ctx, i, raw)
				if err != nil || !reflect.DeepEqual(replay, ack) {
					t.Fatal("seed replay changed ACK", replay, err)
				}
				final, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
				if err != nil || final.ResourceVersion != after.ResourceVersion {
					t.Fatal("seed replay changed ETag", err)
				}
				installerAudit(t, l, i, r)
			})
		}
	}
}

func TestD111_InstallerNullBaselineRejectsUnprovenRelease(t *testing.T) {
	for _, kind := range []string{"candidate", "unknown", "unverified", "unapproved", "unsupported", "revoked", "invented_health"} {
		t.Run(kind, func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			installerNullBaseline(t, l, i)
			r := installerOutcome(installerResult(t, l, i), api.UpdateResultStateFailed, l.old)
			switch kind {
			case "candidate":
				r.CurrentReleaseId.Set(l.new)
			case "unknown":
				r.CurrentReleaseId.Set(uuid.New())
			case "unverified":
				if _, err := l.s.pool.Exec(l.ctx, `UPDATE release_verifications SET state='pending' WHERE release_id=$1`, l.old); err != nil {
					t.Fatal(err)
				}
			case "unapproved":
				l.support([]uuid.UUID{l.new}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
			case "unsupported":
				l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{})
			case "revoked":
				l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{l.old})
			case "invented_health":
				r.Health = installerResult(t, l, i).Health
			}
			detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err == nil || ack.Accepted {
				t.Fatal("accepted unproven seed", ack, err)
			}
			after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || !after.Releases.CurrentReleaseId.IsNull() || !after.Releases.LastGoodReleaseId.IsNull() || after.ResourceVersion != detail.ResourceVersion || !after.Update.IsNull() {
				t.Fatal("rejected seed changed node", after, err)
			}
			var rows int
			if err := l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&rows); err != nil || rows != 0 {
				t.Fatal(rows, err)
			}
		})
	}
}

func TestD110_InstallerProjectionETagAndGenerationFence(t *testing.T) {
	for _, state := range []api.UpdateResultState{api.UpdateResultStateFailed, api.UpdateResultStateRolledBack, api.UpdateResultStateSucceeded} {
		for _, newer := range []bool{false, true} {
			t.Run(string(state)+map[bool]string{false: "/same_generation", true: "/older_generation"}[newer], func(t *testing.T) {
				l := newLabState(t)
				i := l.node("amd64", "disabled", "registered", true)
				l.batch(l.new)
				w := l.intent(i)
				l.report(i, w, "succeeded")
				generation := w.Intent.Generation
				if newer {
					generation++
				}
				if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: generation, CurrentReleaseID: &l.new, LastGoodReleaseID: &l.new, ObservedAt: time.Now().UTC()}); err != nil {
					t.Fatal(err)
				}
				r := installerOutcome(installerResult(t, l, i), state, l.new)
				if state == api.UpdateResultStateRolledBack {
					r.CurrentReleaseId.Set(l.old)
				}
				r.Generation = w.Intent.Generation
				detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
				if err != nil {
					t.Fatal(err)
				}
				before := installerCentralSnapshot(t, l)
				ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
				applied := state == api.UpdateResultStateRolledBack && !newer
				if err != nil || !ack.Accepted || ack.AppliedToCurrentObservation != applied || ack.AcceptedAsHistory != newer || ack.ObservedUpdateGeneration != generation {
					t.Fatal(ack, err)
				}
				want := l.new
				if applied {
					want = l.old
				}
				installerVersions(t, l, i, want, want)
				after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
				version := detail.ResourceVersion
				if applied {
					version++
				}
				if err != nil || after.ResourceVersion != version || !reflect.DeepEqual(after.Update, detail.Update) || before != installerCentralSnapshot(t, l) {
					t.Fatal("hidden receipt changed projection/ETag/central work", after.ResourceVersion, version, err)
				}
			})
		}
	}
}

func TestD110_InstallerMigrationDown190Up(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	database := stdlib.OpenDBFromPool(l.s.pool)
	defer database.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.DownTo(l.ctx, 190); err != nil {
		t.Fatal("DownTo(190)", err)
	}
	var absent bool
	if err = l.s.pool.QueryRow(l.ctx, `SELECT to_regclass('installer_update_results') IS NULL AND to_regclass('installer_update_results_latest') IS NULL`).Scan(&absent); err != nil || !absent {
		t.Fatal("installer objects survived DownTo(190)", err)
	}
	if _, err = provider.Up(l.ctx); err != nil {
		t.Fatal("Up after DownTo(190)", err)
	}
	r := installerResult(t, l, i)
	body := payload(t, r)
	ack, err := l.s.SubmitResult(l.ctx, i, body)
	if err != nil || !ack.Accepted || !ack.AppliedToCurrentObservation {
		t.Fatal(ack, err)
	}
	replay, err := l.s.SubmitResult(l.ctx, i, body)
	if err != nil || !reflect.DeepEqual(replay, ack) {
		t.Fatal(replay, err)
	}
	changed := r
	changed.ResultId = uuid.New()
	conflict, err := l.s.SubmitResult(l.ctx, i, payload(t, changed))
	installerError(t, conflict, err, apierr.AckConflict, 409)
	installerVersions(t, l, i, l.new, l.new)
	installerAudit(t, l, i, r)
	for _, query := range []string{`UPDATE installer_update_results SET state=state`, `DELETE FROM installer_update_results`, `TRUNCATE installer_update_results`} {
		if _, err = l.s.pool.Exec(l.ctx, query); err == nil {
			t.Fatal("round-trip lost immutability guard", query)
		}
	}
}

// NewInstallerNullBaselineFixture exposes the domain fixture to HTTP boundary
// tests without replacing the production handler, command or replay paths.
func NewInstallerNullBaselineFixture(t *testing.T, state api.UpdateResultState) (*Service, *nodes.Service, *pgxpool.Pool, api.UpdateResult) {
	t.Helper()
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	installerNullBaseline(t, l, i)
	r := installerOutcome(installerResult(t, l, i), state, l.old)
	if err := json.Unmarshal(payload(t, r), &r); err != nil {
		t.Fatal(err)
	}
	return l.s, nodes.NewService(l.s.pool), l.s.pool, r
}

func TestD110_InstallerJobIDBoundsRowsAuditsAndETag(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	r := installerResult(t, l, i)
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
	if err != nil || !ack.Accepted {
		t.Fatal(ack, err)
	}
	detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := range 9 {
		other := r
		other.ResultId = uuid.New()
		other.JobId = uuid.New()
		if attempt == 0 {
			other.JobId = installerjob.ID(r.RequestedReleaseId, r.ManifestSha256, uuid.New())
		}
		got, err := l.s.SubmitResult(l.ctx, i, payload(t, other))
		installerError(t, got, err, apierr.StateConflict, 409)
		var apiError *apierr.Error
		if !errors.As(err, &apiError) || apiError.Details["reason"] != "job_id_mismatch" {
			t.Fatal("missing stable mismatch reason", err)
		}
	}
	var rows, audits int
	if err := l.s.pool.QueryRow(l.ctx, `SELECT (SELECT count(*) FROM installer_update_results),(SELECT count(*) FROM audit_events WHERE action='update_result.accept')`).Scan(&rows, &audits); err != nil || rows != 1 || audits != 1 {
		t.Fatal("unbounded evidence", rows, audits, err)
	}
	after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil || after.ResourceVersion != detail.ResourceVersion {
		t.Fatal("mismatched jobs changed ETag", err)
	}
	// A replacement incarnation has its own deterministic receipt identity.
	if _, err = l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, i.IncarnationID); err != nil {
		t.Fatal(err)
	}
	next := l.enroll(i, "amd64")
	r = installerResult(t, l, next)
	if got, err := l.s.SubmitResult(l.ctx, next, payload(t, r)); err != nil || !got.Accepted || got.AppliedToCurrentObservation {
		t.Fatal("replacement receipt", got, err)
	}
	if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&rows); err != nil || rows != 2 {
		t.Fatal("expected one row per incarnation", rows, err)
	}
}

func TestD110_InstallerOlderSuccessAndSeedCannotMoveObservation(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "succeeded", true: "failed_seed"}[empty], func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", true)
			if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: 9, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
				t.Fatal(err)
			}
			r := installerResult(t, l, i)
			r.Generation = 8
			if empty {
				installerNullBaseline(t, l, i)
				r = installerOutcome(r, api.UpdateResultStateFailed, l.old)
			}
			before := installerCentralSnapshot(t, l)
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			if err != nil || !ack.Accepted || !ack.AcceptedAsHistory || ack.AppliedToCurrentObservation || ack.ObservedUpdateGeneration != 9 {
				t.Fatal("older receipt crossed generation fence", ack, err)
			}
			detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			if empty {
				if !detail.Releases.CurrentReleaseId.IsNull() || !detail.Releases.LastGoodReleaseId.IsNull() {
					t.Fatal("older receipt seeded a newer observation", detail.Releases)
				}
			} else {
				installerVersions(t, l, i, l.old, l.old)
			}
			if before != installerCentralSnapshot(t, l) {
				t.Fatal("older receipt changed central work")
			}
		})
	}
}

func TestD110_InstallerLegacyReceiptStillReplays(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	r := installerResult(t, l, i)
	r.JobId = uuid.New() // TEST ONLY historical pre-D-110 receipt identity.
	raw := payload(t, r)
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	ack := api.UpdateResultAck{ResultId: r.ResultId, Accepted: true, ReceivedAt: time.Now().UTC().Truncate(time.Microsecond)}
	if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO installer_update_results(node_id,job_id,incarnation_id,release_id,state,payload,payload_hash,ack,received_at) VALUES($1,$2,$3,$4,$5,$6,decode($7,'hex'),$8,$9)`, i.NodeID, r.JobId, i.IncarnationID, r.RequestedReleaseId, r.State, raw, r.PayloadHash, mustJSON(t, ack), ack.ReceivedAt); err != nil {
		t.Fatal(err)
	}
	l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
	detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := l.s.SubmitResult(l.ctx, i, raw)
	if err != nil || !reflect.DeepEqual(got, ack) {
		t.Fatal("legacy replay revalidated job identity or eligibility", got, err)
	}
	r.ResultId = uuid.New()
	got, err = l.s.SubmitResult(l.ctx, i, payload(t, r))
	installerError(t, got, err, apierr.AckConflict, 409)
	after, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil || after.ResourceVersion != detail.ResourceVersion {
		t.Fatal("legacy replay changed ETag", err)
	}
}
