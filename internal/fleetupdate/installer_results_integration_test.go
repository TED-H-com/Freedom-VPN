//go:build integration

package fleetupdate

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Start with the canonical contract example, including its null batch/target,
// source, attempt, generation, and health/checkpoint shape. Only fixture-bound
// identities and content change; ValidateBody uses the embedded OpenAPI document.
func installerResult(t *testing.T, l *lab, i Identity) api.UpdateResult {
	t.Helper()
	raw, err := os.ReadFile("../../contracts/examples/valid/UpdateResult.installer.json")
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
	r.NodeId, r.IncarnationId = i.NodeID, i.IncarnationID
	r.RequestedReleaseId = l.new
	r.CurrentReleaseId.Set(l.new)
	r.PreviousReleaseId.Set(l.old)
	r.ResultId = uuid.New()
	if err = l.s.pool.QueryRow(l.ctx, `SELECT encode(manifest_hash,'hex') FROM releases WHERE id=$1`, l.new).Scan(&r.ManifestSha256); err != nil {
		t.Fatal(err)
	}
	r.JobId = installerjob.ID(r.RequestedReleaseId, r.ManifestSha256, i.IncarnationID)
	if err = l.f.Validator.ValidateBody("UpdateResult", payload(t, r)); err != nil {
		t.Fatal(err)
	}
	return r
}

// Full rows as well as counts: installer acceptance must not even change a
// timestamp, retry budget, confirmation or operation on central batch work.
func installerCentralSnapshot(t *testing.T, l *lab) string {
	t.Helper()
	var out strings.Builder
	for _, table := range []string{"update_batches", "update_targets", "update_jobs", "update_results", "update_batch_progress", "fleet_node_state", "fleet_config", "operations", "outbox_events", "exits"} {
		var count int
		var rows []byte
		if err := l.s.pool.QueryRow(l.ctx, `SELECT count(*),coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]'::jsonb) FROM `+table+` t`).Scan(&count, &rows); err != nil {
			t.Fatal(err)
		}
		out.WriteString(table)
		b, _ := json.Marshal(count)
		out.Write(b)
		out.Write(rows)
	}
	var nodeState []byte
	if err := l.s.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(to_jsonb(n)-ARRAY['current_release_id','last_good_release_id','resource_version','updated_at'] ORDER BY id),'[]'::jsonb) FROM nodes n`).Scan(&nodeState); err != nil {
		t.Fatal(err)
	}
	out.Write(nodeState)
	return out.String()
}

func installerVersions(t *testing.T, l *lab, i Identity, current, good uuid.UUID) {
	t.Helper()
	detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	c, ce := detail.Releases.CurrentReleaseId.Get()
	g, ge := detail.Releases.LastGoodReleaseId.Get()
	if ce != nil || ge != nil || c != current || g != good {
		t.Fatalf("versions current=%v good=%v: %v %v", c, g, ce, ge)
	}
}

func TestD089_InstallerAcceptReplayAndIsolation(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			l := newLabState(t)
			i := l.node(arch, "disabled", "registered", false)
			r := installerResult(t, l, i)
			// Installer generations cannot become the accepted central high-water.
			r.Generation = 9007199254740993
			before := installerCentralSnapshot(t, l)
			raw := payload(t, r)
			ack, err := l.s.SubmitResult(l.ctx, i, raw)
			if err != nil || !ack.Accepted || ack.AcceptedAsHistory || !ack.AppliedToCurrentObservation || ack.ObservedUpdateGeneration != 0 || ack.ResultId != r.ResultId {
				t.Fatalf("ack=%+v err=%v", ack, err)
			}
			if err = l.f.Validator.ValidateBody("UpdateResultAck", mustJSON(t, ack)); err != nil {
				t.Fatal(err)
			}
			if after := installerCentralSnapshot(t, l); before != after {
				t.Fatal("installer acceptance changed central work")
			}
			installerVersions(t, l, i, l.new, l.new)
			installerAudit(t, l, i, r)
			detail, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil {
				t.Fatal(err)
			}
			summary, err := detail.Update.Get()
			if err != nil || summary.Source == nil || *summary.Source != api.UpdateResultSourceInstaller || summary.JobId != r.JobId || !summary.BatchId.IsNull() || !summary.LastObservedAt.IsNull() {
				t.Fatalf("summary=%+v err=%v", summary, err)
			}
			at, err := summary.LastResultAt.Get()
			if err != nil || !at.Equal(ack.ReceivedAt) {
				t.Fatal("receipt time differs", at, ack.ReceivedAt, err)
			}
			// Concurrent replays through a fresh service preserve ACK and one audit.
			service, err := New(l.s.pool, l.s.releases, l.s.config)
			if err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					got, e := service.SubmitResult(l.ctx, i, raw)
					if e != nil || !reflect.DeepEqual(got, ack) {
						t.Errorf("replay=%+v err=%v", got, e)
					}
				})
			}
			wg.Wait()
			replayed, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
			if err != nil || replayed.ResourceVersion != detail.ResourceVersion {
				t.Fatal("replay changed node ETag", err)
			}
			sourceless := r
			sourceless.Source = nil
			missing, missingErr := l.s.SubmitResult(l.ctx, i, payload(t, sourceless))
			installerError(t, missing, missingErr, apierr.ResourceNotFound, 404)
			changed := r
			changed.ResultId = uuid.New()
			for range 2 {
				got, e := l.s.SubmitResult(l.ctx, i, payload(t, changed))
				installerError(t, got, e, apierr.AckConflict, 409)
			}
			var saved []byte
			var count, audits int
			if err = l.s.pool.QueryRow(l.ctx, `SELECT payload,(SELECT count(*) FROM installer_update_results),(SELECT count(*) FROM audit_events WHERE action='update_result.accept' AND after_diff->>'source'='installer') FROM installer_update_results WHERE node_id=$1 AND job_id=$2`, i.NodeID, r.JobId).Scan(&saved, &count, &audits); err != nil {
				t.Fatal(err)
			}
			var retained api.UpdateResult
			if err = json.Unmarshal(saved, &retained); err != nil || retained.ResultId != r.ResultId || count != 1 || audits != 1 {
				t.Fatal("first evidence/audit changed", count, audits, err)
			}
			for _, sql := range []string{`UPDATE installer_update_results SET state=state`, `DELETE FROM installer_update_results`, `TRUNCATE installer_update_results`} {
				if _, err = l.s.pool.Exec(l.ctx, sql); err == nil {
					t.Fatalf("immutable evidence accepted %s", sql)
				}
			}
			if before != installerCentralSnapshot(t, l) {
				t.Fatal("replay or conflict changed central work")
			}
			// Replays are durable receipts, even after publication policy changes.
			l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
			got, err := service.SubmitResult(l.ctx, i, raw)
			if err != nil || !reflect.DeepEqual(got, ack) {
				t.Fatal("replay after revocation", got, err)
			}
		})
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func installerError(t *testing.T, ack api.UpdateResultAck, err error, code apierr.Code, status int) {
	t.Helper()
	var ae *apierr.Error
	if ack.Accepted || !errors.As(err, &ae) || ae.Code != code || ae.HTTPStatus() != status {
		t.Fatalf("ack=%+v err=%v; want %s/%d", ack, err, code, status)
	}
}

func TestD089_InstallerRejections(t *testing.T) {
	cases := []struct {
		name   string
		code   apierr.Code
		status int
		mutate func(*testing.T, *lab, *Identity, *api.UpdateResult)
	}{
		{"foreign_node", apierr.IncarnationMismatch, 403, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			*i = l.node("arm64", "disabled", "registered", false)
		}},
		{"body_incarnation", apierr.IncarnationMismatch, 403, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.IncarnationId = uuid.New() }},
		{"stale_incarnation", apierr.IncarnationMismatch, 403, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			if _, e := l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, i.IncarnationID); e != nil {
				t.Fatal(e)
			}
			l.enroll(*i, "amd64")
		}},
		{"unknown_release", apierr.ResourceNotFound, 404, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			r.RequestedReleaseId = uuid.New()
			r.CurrentReleaseId.Set(r.RequestedReleaseId)
		}},
		{"unpublished_release", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
		}},
		{"unverified_release", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			if _, e := l.s.pool.Exec(l.ctx, `UPDATE release_verifications SET state='pending' WHERE release_id=$1`, l.new); e != nil {
				t.Fatal(e)
			}
		}},
		{"revoked_release", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
		}},
		{"central_job", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			l.batch(l.new)
			r.JobId = l.intent(*i).Intent.JobId
		}},
		{"foreign_central_job", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			other := l.node("amd64", "disabled", "registered", false)
			l.batch(l.new)
			r.JobId = l.intent(other).Intent.JobId
		}},
		{"source_less_unknown_job", apierr.ResourceNotFound, 404, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.Source = nil }},
		{"explicit_central_unknown_job", apierr.ResourceNotFound, 404, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			source := api.UpdateResultSourceCentral
			r.Source = &source
		}},
		{"batch_binding", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.BatchId.Set(uuid.New()) }},
		{"target_binding", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.TargetId.Set(uuid.New()) }},
		{"attempt", apierr.StateConflict, 409, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.Attempt = 2 }},
		{"manifest_digest", apierr.HashMismatch, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			r.ManifestSha256 = strings.Repeat("0", 64)
		}},
		{"missing_version", apierr.ValidationFailed, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.CurrentReleaseId.SetNull() }},
		{"wrong_version", apierr.HealthCheckFailed, 500, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.CurrentReleaseId.Set(l.old) }},
		{"missing_finish", apierr.ValidationFailed, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.FinishedAt.SetNull() }},
		{"missing_health", apierr.ValidationFailed, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.Health.SetNull() }},
		{"bad_health", apierr.HealthCheckFailed, 500, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			h, _ := r.Health.Get()
			h.TunnelProbe = "failed"
			r.Health.Set(h)
		}},
		{"missing_checkpoint", apierr.HealthCheckFailed, 500, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.UsageCheckpoint.SetNull() }},
		{"bad_checkpoint", apierr.CheckpointFailed, 500, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) {
			u, _ := r.UsageCheckpoint.Get()
			u.CentralAckedThrough = u.LastDurableEventSeq + 1
			r.UsageCheckpoint.Set(u)
		}},
		{"schema", apierr.SchemaIncompatible, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.StateSchemaVersion = 99 }},
		{"forwarding", apierr.NodeNotServiceable, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.MemberForwardingEnabled = true }},
		{"failure_without_error", apierr.ValidationFailed, 422, func(t *testing.T, l *lab, i *Identity, r *api.UpdateResult) { r.State = api.UpdateResultStateFailed }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			r := installerResult(t, l, i)
			tc.mutate(t, l, &i, &r)
			before := installerCentralSnapshot(t, l)
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			installerError(t, ack, err, tc.code, tc.status)
			if before != installerCentralSnapshot(t, l) {
				t.Fatal("rejection changed central work")
			}
			var count int
			if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&count); err != nil || count != 0 {
				t.Fatal("rejected evidence stored", count, err)
			}
			installerVersions(t, l, Identity{NodeID: r.NodeId}, l.old, l.old)
		})
	}
	t.Run("payload_hash", func(t *testing.T) {
		l := newLabState(t)
		i := l.node("amd64", "disabled", "registered", false)
		r := installerResult(t, l, i)
		r.PayloadHash = strings.Repeat("0", 64)
		ack, err := l.s.SubmitResult(l.ctx, i, mustJSON(t, r))
		installerError(t, ack, err, apierr.HashMismatch, 422)
	})
}

func TestD089_InstallerFailuresAndTransaction(t *testing.T) {
	for _, state := range []api.UpdateResultState{api.UpdateResultStateFailed, api.UpdateResultStateRolledBack, api.UpdateResultStateAlreadyCurrent} {
		t.Run(string(state), func(t *testing.T) {
			l := newLabState(t)
			i := l.node("amd64", "disabled", "registered", false)
			r := installerResult(t, l, i)
			r.State = state
			if state != api.UpdateResultStateAlreadyCurrent {
				r.ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
			}
			if state == api.UpdateResultStateRolledBack {
				r.CurrentReleaseId.Set(l.old)
				r.PreviousReleaseId.Set(l.new)
			}
			raw := payload(t, r)
			tx, err := l.s.pool.Begin(l.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = l.s.SubmitResultTx(l.ctx, tx, i, raw); err != nil {
				_ = tx.Rollback(l.ctx)
				t.Fatal(err)
			}
			if err = tx.Rollback(l.ctx); err != nil {
				t.Fatal(err)
			}
			installerVersions(t, l, i, l.old, l.old)
			var count int
			if err = l.s.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results`).Scan(&count); err != nil || count != 0 {
				t.Fatal(count, err)
			}
			var ack api.UpdateResultAck
			err = db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error { var e error; ack, e = l.s.SubmitResultTx(l.ctx, tx, i, raw); return e })
			if err != nil || !ack.Accepted {
				t.Fatal(ack, err)
			}
			want := l.old
			if state == api.UpdateResultStateAlreadyCurrent {
				want = l.new
			}
			installerVersions(t, l, i, want, want)
			installerAudit(t, l, i, r)
			if ack.AppliedToCurrentObservation != (state == api.UpdateResultStateAlreadyCurrent) {
				t.Fatal(ack)
			}
		})
	}
}

func TestD097_InstallerProjectionCentralPrecedenceAndIncarnation(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	// A central accepted terminal result and its confirmed batch stay byte-for-byte
	// unchanged when installer evidence arrives, even on the same release.
	l.report(i, w, "succeeded")
	if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: w.Intent.Generation, CurrentReleaseID: &l.new, LastGoodReleaseID: &l.new, ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	before := installerCentralSnapshot(t, l)
	batchBefore := l.get(b.BatchId)
	r := installerResult(t, l, i)
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil || !ack.Accepted || ack.ObservedUpdateGeneration != w.Intent.Generation {
		t.Fatal(ack, err)
	}
	if before != installerCentralSnapshot(t, l) || !reflect.DeepEqual(batchBefore, l.get(b.BatchId)) {
		t.Fatal("installer altered central batch state")
	}
	for _, target := range l.get(b.BatchId).Targets {
		if target.Source != nil && *target.Source != api.UpdateResultSourceCentral {
			t.Fatal("installer target", target)
		}
		j, err := target.CurrentJob.Get()
		if err == nil && j.Source != nil && *j.Source != api.UpdateResultSourceCentral {
			t.Fatal("installer job", j)
		}
	}
	read := func() api.NodeUpdateSummary {
		t.Helper()
		d, _, e := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
		if e != nil {
			t.Fatal(e)
		}
		s, e := d.Update.Get()
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	s := read()
	if s.Source == nil || *s.Source != api.UpdateResultSourceCentral || s.JobId != w.Intent.JobId {
		t.Fatal("central job lost precedence", s)
	}
	// Explicit projection-boundary fixture: production retains the central job
	// pointer. Clear only this pointer to exercise the granted fallback reader.
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE fleet_node_state SET current_job_id=NULL WHERE node_id=$1`, i.NodeID); err != nil {
		t.Fatal(err)
	}
	s = read()
	if s.Source == nil || *s.Source != api.UpdateResultSourceInstaller || !s.BatchId.IsNull() {
		t.Fatal("latest installer missing", s)
	}
	// A later accepted central history receipt wins when no current job exists.
	late := result(w, "succeeded")
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, late)); err != nil || !ack.AcceptedAsHistory {
		t.Fatal(ack, err)
	}
	s = read()
	if s.Source == nil || *s.Source != api.UpdateResultSourceCentral || s.JobId != w.Intent.JobId {
		t.Fatal("latest central missing", s)
	}
	// Rejected central receipts must never win the fallback selection.
	bad := result(w, "succeeded")
	bad.Health.SetNull()
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, bad)); err == nil {
		t.Fatal("accepted missing health")
	}
	if got := read(); !reflect.DeepEqual(got, s) {
		t.Fatal("rejection displaced accepted evidence")
	}
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, i.IncarnationID); err != nil {
		t.Fatal(err)
	}
	replacement := l.enroll(i, "arm64")
	d, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, replacement.NodeID)
	if err != nil || !d.Update.IsNull() {
		t.Fatal("old incarnation evidence leaked", d.Update, err)
	}
}

func installerAudit(t *testing.T, l *lab, i Identity, r api.UpdateResult) {
	t.Helper()
	store := audit.Store{Pool: l.s.pool}
	page, err := store.List(l.ctx, api.ListAuditEventsParams{ResourceId: &i.NodeID})
	if err != nil || len(page.Items) != 1 || store.AnomalyCount() != 0 {
		t.Fatal("installer audit readback", page, err)
	}
	e := page.Items[0]
	actor, ae := e.Actor.Id.Get()
	request, re := e.RequestId.Get()
	after, ce := e.Changes.After.Get()
	if ae != nil || re != nil || ce != nil || actor != i.NodeID || request == uuid.Nil || e.Resource.Type != api.ResourceTypeNode || e.Resource.Id != i.NodeID || e.Action != "update_result.accept" || e.Result != api.AuditEventResultSucceeded || after["source"] != "installer" || after["job_id"] != r.JobId.String() || after["result_id"] != r.ResultId.String() {
		t.Fatalf("installer audit=%+v", e)
	}
	if err = l.f.Validator.ValidateBody("AuditEvent", mustJSON(t, e)); err != nil {
		t.Fatal(err)
	}
}

func TestD089_InstallerDoesNotCompleteQueuedBatch(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	before := installerCentralSnapshot(t, l)
	batchBefore := l.get(b.BatchId)
	r := installerResult(t, l, i)
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil || !ack.Accepted || ack.ObservedUpdateGeneration != 0 {
		t.Fatal(ack, err)
	}
	// This existing field reads nodes.current_release_id, independently of job
	// evidence. Only the actual node version changes; every batch result stays.
	batchBefore.Targets[0].CurrentReleaseId.Set(l.new)
	if before != installerCentralSnapshot(t, l) || !reflect.DeepEqual(batchBefore, l.get(b.BatchId)) {
		t.Fatal("installer completed or altered queued work")
	}
	d, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	s, err := d.Update.Get()
	if err != nil || s.JobId != w.Intent.JobId || s.Source == nil || *s.Source != api.UpdateResultSourceCentral {
		t.Fatal("queued central projection displaced", s, err)
	}
	// Explicit central source still follows the unchanged authorized-job path.
	central := result(w, "already_current")
	source := api.UpdateResultSourceCentral
	central.Source = &source
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, central)); err != nil || !ack.Accepted || ack.AcceptedAsHistory {
		t.Fatal(ack, err)
	}
}

func TestD089_InstallerPreHealthBaseline(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	r := installerResult(t, l, i)
	r.State = api.UpdateResultStateFailed
	r.Stage.Set(api.UpdateStageDownload)
	r.ErrorCode.Set(api.ErrorCodeDOWNLOADTIMEOUT)
	r.Health.SetNull()
	// A pre-health failure cannot attest the candidate as installed.
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
	installerError(t, ack, err, apierr.ValidationFailed, 422)
	r.CurrentReleaseId.Set(l.old)
	if ack, err = l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil || !ack.Accepted || ack.AppliedToCurrentObservation {
		t.Fatal(ack, err)
	}
	installerVersions(t, l, i, l.old, l.old)
}

// NewInstallerNodeViewFixture exposes only the real node reader to external
// HTTP tests; all setup and installer acceptance stay in the fleetupdate lab.
func NewInstallerNodeViewFixture(t *testing.T) (*nodes.Service, api.UpdateResult) {
	t.Helper()
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	r := installerResult(t, l, i)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
		t.Fatal(err)
	}
	return nodes.NewService(l.s.pool), r
}

func TestD089_InstallerAcceptanceClosesHeartbeatSeeding(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "disabled", "registered", false)
	// Initial fixture: an incarnation without a centrally known serving version.
	if _, err := l.s.pool.Exec(l.ctx, `UPDATE nodes SET current_release_id=NULL,last_good_release_id=NULL,resource_version=resource_version+1 WHERE id=$1`, i.NodeID); err != nil {
		t.Fatal(err)
	}
	l.batch(l.new)
	w := l.intent(i)
	r := installerResult(t, l, i)
	r.State = api.UpdateResultStateFailed
	r.ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil || !ack.Accepted || ack.AppliedToCurrentObservation {
		t.Fatal(ack, err)
	}
	if err := l.s.Observe(l.ctx, i, Observation{Sequence: 1, AppliedUpdateGeneration: w.Intent.Generation, CurrentReleaseID: &l.new, ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	d, _, err := nodes.NewService(l.s.pool).GetNode(l.ctx, i.NodeID)
	if err != nil || !d.Releases.CurrentReleaseId.IsNull() || !d.Releases.LastGoodReleaseId.IsNull() {
		t.Fatal("heartbeat seeded after installer evidence", d.Releases, err)
	}
}
