//go:build integration

package fleetupdate

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type lab struct {
	t        *testing.T
	s        *Service
	f        *releasetest.Fixture
	old, new uuid.UUID
	ctx      context.Context
}

func newLab(t *testing.T, auditActions ...string) *lab {
	t.Helper()
	l := newLabState(t)
	t.Cleanup(func() {
		releasetest.AssertLedger(t, l.s.pool, l.f.Validator, append(auditActions, "release.register", "release_support.activate")...)
	})
	return l
}

// newLabState is also used by cross-domain tests whose observation audits are
// outside the release writer allowlist. Domain ledger checks stay in newLab.
func newLabState(t *testing.T) *lab {
	t.Helper()
	pool := dbtest.New(t)
	f := releasetest.New(t)
	r := &releases.Service{Pool: pool, Validator: f.Validator}
	s, e := New(pool, r, Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if e != nil {
		t.Fatal(e)
	}
	l := &lab{t: t, s: s, f: f, ctx: context.Background()}
	l.old = f.Manifest.ReleaseId
	if _, e = r.Register(l.ctx, releasetest.Command(), f.Request(t, f.Manifest)); e != nil {
		t.Fatal(e)
	}
	f.Manifest.ReleaseId = uuid.Must(uuid.NewV7())
	f.Manifest.ProductVersion = "1.2.0-test"
	f.Manifest.Rollback.AllowedReleaseIds = []uuid.UUID{l.old}
	l.new = f.Manifest.ReleaseId
	if _, e = r.Register(l.ctx, releasetest.Command(), f.Request(t, f.Manifest)); e != nil {
		t.Fatal(e)
	}
	l.support([]uuid.UUID{l.old, l.new}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
	return l
}
func (l *lab) support(approved, serving, revoked []uuid.UUID) {
	l.t.Helper()
	p := api.ReleaseSupportPayload{ApprovedReleaseIds: approved, RevokedReleaseIds: revoked}
	p.MinimumServingVersion.ServingReleaseIds = serving
	e := db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return l.s.releases.ApplySupportTx(l.ctx, tx, uuid.New(), p, "TEST ONLY policy")
	})
	if e != nil {
		l.t.Fatal(e)
	}
}
func (l *lab) node(arch, pub, lifecycle string, online bool) Identity {
	l.t.Helper()
	i := Identity{NodeID: uuid.Must(uuid.NewV7())}
	var fp [32]byte
	_, _ = rand.Read(fp[:])
	_, e := l.s.pool.Exec(l.ctx, `INSERT INTO nodes(id,lifecycle_state,current_release_id,last_good_release_id) VALUES($1,$2,$3,$3)`, i.NodeID, lifecycle, l.old)
	if e != nil {
		l.t.Fatal(e)
	}
	_, e = l.s.pool.Exec(l.ctx, `INSERT INTO exits(node_id,display_name,region,publication_intent) VALUES($1,'TEST ONLY node','test',$2)`, i.NodeID, pub)
	if e != nil {
		l.t.Fatal(e)
	}
	if lifecycle != "enrolling" {
		i.IncarnationID = uuid.Must(uuid.NewV7())
		_, e = l.s.pool.Exec(l.ctx, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch) VALUES($1,$2,$3,'ubuntu/24.04',$4)`, i.IncarnationID, i.NodeID, fp[:], arch)
		if e != nil {
			l.t.Fatal(e)
		}
		_, e = l.s.pool.Exec(l.ctx, `UPDATE nodes SET resource_version=resource_version+1,current_incarnation_id=$2 WHERE id=$1`, i.NodeID, i.IncarnationID)
		if e != nil {
			l.t.Fatal(e)
		}
		if online && lifecycle == "registered" {
			if e = l.s.Observe(l.ctx, i, Observation{Sequence: 1, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); e != nil {
				l.t.Fatal(e)
			}
		}
	}
	return i
}
func (l *lab) enroll(i Identity, arch string) Identity {
	l.t.Helper()
	i.IncarnationID = uuid.Must(uuid.NewV7())
	var fp [32]byte
	_, _ = rand.Read(fp[:])
	_, e := l.s.pool.Exec(l.ctx, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch) VALUES($1,$2,$3,'ubuntu/24.04',$4)`, i.IncarnationID, i.NodeID, fp[:], arch)
	if e != nil {
		l.t.Fatal(e)
	}
	_, e = l.s.pool.Exec(l.ctx, `UPDATE nodes SET resource_version=resource_version+1,current_incarnation_id=$2,lifecycle_state='registered' WHERE id=$1`, i.NodeID, i.IncarnationID)
	if e != nil {
		l.t.Fatal(e)
	}
	return i
}
func (l *lab) request(release uuid.UUID) api.UpdateBatchCreateRequest {
	l.t.Helper()
	var g string
	if e := l.s.pool.QueryRow(l.ctx, `SELECT generation::text FROM fleet_config`).Scan(&g); e != nil {
		l.t.Fatal(e)
	}
	gen, e := decstr.Parse(g)
	if e != nil {
		l.t.Fatal(e)
	}
	return api.UpdateBatchCreateRequest{ExpectedGeneration: gen, ReleaseId: release, Reason: "TEST ONLY update", Scope: "all_nodes", SetFleetDefault: true}
}
func (l *lab) batch(release uuid.UUID) api.UpdateBatchAccepted {
	l.t.Helper()
	b, e := l.s.CreateBatch(l.ctx, releasetest.Command(), l.request(release))
	if e != nil {
		l.t.Fatal(e)
	}
	return b
}
func (l *lab) intent(i Identity) *Work {
	l.t.Helper()
	w, e := l.s.NodeUpdateIntent(l.ctx, i)
	if e != nil || w == nil {
		l.t.Fatal("intent", w, e)
	}
	return w
}
func result(w *Work, state string) api.UpdateResult {
	now := time.Now().UTC()
	i := w.Intent
	r := api.UpdateResult{ResultId: uuid.Must(uuid.NewV7()), NodeId: i.NodeId, IncarnationId: i.IncarnationId, BootId: uuid.New(), BatchId: i.BatchId, TargetId: i.TargetId, JobId: i.JobId, Attempt: i.Attempt, Generation: i.Generation, State: api.UpdateResultState(state), RequestedReleaseId: i.ReleaseId, CurrentReleaseId: presence.NewNullableWithValue(i.ReleaseId), ManifestSha256: i.ManifestSha256, PublicationIntent: i.PublicationIntent, StartedAt: now, StateSchemaVersion: 1, Health: presence.NewNullableWithValue(api.UpdateHealth{Processes: "passed", IdentityAndState: "passed", InternetOnly: "passed", DnsPolicy: "passed", TunnelProbe: "passed"}), UsageCheckpoint: presence.NewNullableWithValue(api.UpdateUsageCheckpoint{LastDurableEventSeq: 10, CentralAckedThrough: 9,
		ClosedMeterEpochId: presence.NewNullNullable[uuid.UUID](),
		NewMeterEpochId:    presence.NewNullNullable[uuid.UUID]()}),
		ErrorCode:         presence.NewNullNullable[api.ErrorCode](),
		FinishedAt:        presence.NewNullNullable[time.Time](),
		PreviousReleaseId: presence.NewNullNullable[uuid.UUID](),
		RecoveryRequired:  presence.NewRequired(false),
		Stage:             presence.NewNullNullable[api.UpdateStage]()}
	if terminal(state) {
		r.FinishedAt.Set(now)
	}
	return r
}
func payload(t *testing.T, r api.UpdateResult) []byte {
	t.Helper()
	raw, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	r.PayloadHash, e = canonjson.HashHex(raw, "payload_hash")
	if e != nil {
		t.Fatal(e)
	}
	raw, e = json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func (l *lab) report(i Identity, w *Work, state string) api.UpdateResultAck {
	l.t.Helper()
	a, e := l.s.SubmitResult(l.ctx, i, payload(l.t, result(w, state)))
	if e != nil {
		l.t.Fatal(e)
	}
	return a
}
func (l *lab) get(id uuid.UUID) api.UpdateBatch {
	l.t.Helper()
	b, e := l.s.GetBatch(l.ctx, id, "", 100)
	if e != nil {
		l.t.Fatal(e)
	}
	return b
}
func (l *lab) scalar(query string, args ...any) int {
	l.t.Helper()
	var n int
	if e := l.s.pool.QueryRow(l.ctx, query, args...).Scan(&n); e != nil {
		l.t.Fatal(e)
	}
	return n
}

func TestAT23_AllStatesArchitecturesAndEvidence(t *testing.T) {
	l := newLab(t)
	nodes := []Identity{l.node("amd64", "published", "registered", true), l.node("arm64", "unpublished", "registered", true), l.node("amd64", "disabled", "registered", true), l.node("arm64", "unpublished", "enrolling", false)}
	l.node("arm64", "disabled", "revoked", false)
	l.node("amd64", "unpublished", "retired", false)
	b := l.batch(l.new)
	if b.TargetCount != 4 || b.OnlineAtSnapshot != 3 {
		t.Fatal(b)
	}
	for n, i := range nodes[:3] {
		w := l.intent(i)
		want := "amd64"
		if n == 1 {
			want = "arm64"
		}
		if string(w.Artifact.Architecture) != want {
			t.Fatal(w.Artifact)
		}
		bad := result(w, "succeeded")
		bad.Health.SetNull()
		if _, e := l.s.SubmitResult(l.ctx, i, payload(t, bad)); e == nil {
			t.Fatal("success without evidence")
		}
		l.report(i, w, "succeeded")
	}
	got := l.get(b.BatchId)
	if got.Counts.Succeeded != 3 || got.Counts.WaitingOnline != 1 || got.State != "waiting" {
		t.Fatal(got.Counts, got.State)
	}
	var publication string
	if e := l.s.pool.QueryRow(l.ctx, `SELECT publication_intent FROM exits WHERE node_id=$1`, nodes[2].NodeID).Scan(&publication); e != nil || publication != "disabled" {
		t.Fatal(publication, e)
	}
}
func TestAT24_OfflineEnrollmentAndFirstSync(t *testing.T) {
	l := newLab(t, "update_batch.create", "update_batch.first_sync", "update_result.accept")
	offline := l.node("amd64", "published", "registered", false)
	enrolling := l.node("arm64", "unpublished", "enrolling", false)
	b := l.batch(l.new)
	if l.scalar(`SELECT count(*) FROM update_jobs WHERE node_id=$1`, enrolling.NodeID) != 0 {
		t.Fatal("enrolling target must not have a job before incarnation binding")
	}
	unbound := l.get(b.BatchId)
	for _, target := range unbound.Targets {
		if target.NodeId == enrolling.NodeID && (!target.CurrentJob.IsNull() || target.Attempts != 0 || target.State != "waiting_online") {
			t.Fatal(target)
		}
	}
	raw, _ := json.Marshal(unbound)
	if e := l.s.releases.Validator.ValidateBody("UpdateBatch", raw); e != nil {
		t.Fatal(e)
	}
	w := l.intent(offline)
	if (!w.Intent.BatchId.IsSpecified() || w.Intent.BatchId.IsNull()) || nullableValue(w.Intent.BatchId) != b.BatchId {
		t.Fatal(w)
	}
	l.report(offline, w, "succeeded")
	enrolling = l.enroll(enrolling, "arm64")
	w = l.intent(enrolling)
	same := l.intent(enrolling)
	if same.Intent.JobId != w.Intent.JobId {
		t.Fatal("binding changed")
	}
	l.report(enrolling, w, "already_current")
	after := l.node("arm64", "disabled", "registered", true)
	first := l.intent(after)
	if (!first.Intent.BatchId.IsSpecified() || first.Intent.BatchId.IsNull()) || nullableValue(first.Intent.BatchId) == b.BatchId || (!first.Intent.TargetId.IsSpecified() || first.Intent.TargetId.IsNull()) || first.Intent.ReleaseId != l.new {
		t.Fatal(first)
	}
	if again := l.intent(after); again.Intent.JobId != first.Intent.JobId || nullableValue(again.Intent.BatchId) != nullableValue(first.Intent.BatchId) {
		t.Fatal("first-sync retry created duplicate work")
	}
	firstBatch := l.get(nullableValue(first.Intent.BatchId))
	if firstBatch.TargetCount != 1 || firstBatch.Reason != "first_sync" || firstBatch.Targets[0].NodeId != after.NodeID {
		t.Fatal(firstBatch)
	}
	if l.scalar(`SELECT count(*) FROM update_jobs WHERE target_id IS NULL OR execution_incarnation_id IS NULL`) != 0 {
		t.Fatal("unbound execution job")
	}
	l.report(after, first, "succeeded")
	got := l.get(b.BatchId)
	if got.TargetCount != 2 || len(got.Targets) != 2 || got.State != "completed" || got.Counts.AlreadyCurrent != 1 {
		t.Fatal(got)
	}
	for _, target := range got.Targets {
		if target.NodeId == enrolling.NodeID && (!target.IncarnationAtSnapshot.IsNull() || (!target.ExecutionIncarnationId.IsSpecified() || target.ExecutionIncarnationId.IsNull()) || nullableValue(target.ExecutionIncarnationId) != enrolling.IncarnationID) {
			t.Fatal(target)
		}
	}
	// Reinstall cannot rebind the historical target.
	_, e := l.s.pool.Exec(l.ctx, `UPDATE node_incarnations SET revoked_at=clock_timestamp() WHERE id=$1`, enrolling.IncarnationID)
	if e != nil {
		t.Fatal(e)
	}
	replacement := l.enroll(Identity{NodeID: enrolling.NodeID}, "arm64")
	next := l.intent(replacement)
	if (!next.Intent.TargetId.IsSpecified() || next.Intent.TargetId.IsNull()) || nullableValue(next.Intent.TargetId) == nullableValue(w.Intent.TargetId) || next.Intent.Generation <= w.Intent.Generation {
		t.Fatal(next)
	}
	if _, e = l.s.SubmitResult(l.ctx, enrolling, payload(t, result(w, "succeeded"))); !errors.Is(e, apierr.New(apierr.IncarnationMismatch, "")) {
		t.Fatal(e)
	}
}
func TestAT26_IdempotencyConcurrencyAndLateResults(t *testing.T) {
	l := newLab(t, "update_result.conflict")
	queued := l.node("amd64", "published", "registered", true)
	applying := l.node("arm64", "disabled", "registered", true)
	request := l.request(l.old)
	command := releasetest.Command()
	var group sync.WaitGroup
	batchIDs := make(chan uuid.UUID, 6)
	errs := make(chan error, 6)
	for range 6 {
		group.Go(func() { b, e := l.s.CreateBatch(l.ctx, command, request); batchIDs <- b.BatchId; errs <- e })
	}
	group.Wait()
	close(batchIDs)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var oldBatch uuid.UUID
	for id := range batchIDs {
		if oldBatch != uuid.Nil && oldBatch != id {
			t.Fatal("duplicate batches")
		}
		oldBatch = id
	}
	changed := request
	changed.ReleaseId = l.new
	if _, e := l.s.CreateBatch(l.ctx, command, changed); !errors.Is(e, apierr.New(apierr.IdempotencyConflict, "")) {
		t.Fatal(e)
	}
	qOld := l.intent(queued)
	aOld := l.intent(applying)
	l.report(applying, aOld, "applying")
	newBatch := l.batch(l.new)
	qNew := l.intent(queued)
	if qNew.Intent.Generation <= qOld.Intent.Generation {
		t.Fatal("generation did not advance")
	}
	still := l.intent(applying)
	if still.Intent.JobId != aOld.Intent.JobId {
		t.Fatal("interrupted applying")
	}
	oldResult := result(qOld, "succeeded")
	raw := payload(t, oldResult)
	ack, e := l.s.SubmitResult(l.ctx, queued, raw)
	if e != nil || !ack.AcceptedAsHistory || ack.AppliedToCurrentObservation {
		t.Fatal(ack, e)
	}
	again, e := l.s.SubmitResult(l.ctx, queued, raw)
	if e != nil || again != ack {
		t.Fatal(again, e)
	}
	oldResult.PolicyGeneration++
	if _, e = l.s.SubmitResult(l.ctx, queued, payload(t, oldResult)); !errors.Is(e, apierr.New(apierr.AckConflict, "")) {
		t.Fatal(e)
	}
	if l.scalar(`SELECT count(*) FROM update_result_conflicts`) != 1 {
		t.Fatal("missing conflict evidence")
	}
	l.report(applying, aOld, "succeeded")
	aNew := l.intent(applying)
	if aNew.Intent.Generation <= aOld.Intent.Generation {
		t.Fatal(aNew)
	}
	l.report(queued, qNew, "succeeded")
	l.report(applying, aNew, "succeeded")
	lateFailure := result(qOld, "failed")
	lateFailure.RecoveryRequired = presence.NewRequired(true)
	if _, e = l.s.SubmitResult(l.ctx, queued, payload(t, lateFailure)); e != nil {
		t.Fatal(e)
	}
	view, e := l.s.NodeVersionView(l.ctx, queued.NodeID)
	if e != nil || view.RecoveryRequired {
		t.Fatal("old result changed current recovery state", view, e)
	}

	old := l.get(oldBatch)
	if old.Counts.Superseded != 1 || old.Counts.Succeeded != 1 {
		t.Fatal(old.Counts)
	}
	if l.get(newBatch.BatchId).State != "completed" {
		t.Fatal("new batch not complete")
	}
	var pointer uuid.UUID
	if e = l.s.pool.QueryRow(l.ctx, `SELECT current_job_id FROM fleet_node_state WHERE node_id=$1`, queued.NodeID).Scan(&pointer); e != nil || pointer != qNew.Intent.JobId {
		t.Fatal(pointer, e)
	}
}
func TestAT27_ObservedVersionRollbackAndDisabled(t *testing.T) {
	l := newLab(t, "update_result.rollback_refused")
	i := l.node("amd64", "published", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	v, e := l.s.NodeVersionView(l.ctx, i.NodeID)
	if e != nil || !v.ReleaseServiceable || *v.CurrentReleaseID == *v.DesiredReleaseID {
		t.Fatal(v, e)
	}
	l.report(i, w, "applying")
	rollback := result(w, "rolled_back")
	rollback.CurrentReleaseId.Set(l.old)
	code := api.ErrorCode("HEALTH_CHECK_FAILED")
	rollback.ErrorCode.Set(code)
	if _, e = l.s.SubmitResult(l.ctx, i, payload(t, rollback)); e != nil {
		t.Fatal(e)
	}
	l.batch(l.new)
	w = l.intent(i)
	l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{})
	v, e = l.s.NodeVersionView(l.ctx, i.NodeID)
	if e != nil || v.ReleaseServiceable {
		t.Fatal(v, e)
	}
	l.report(i, w, "applying")
	rollback = result(w, "rolled_back")
	rollback.CurrentReleaseId.Set(l.old)
	rollback.ErrorCode.Set(code)
	if _, e = l.s.SubmitResult(l.ctx, i, payload(t, rollback)); !errors.Is(e, apierr.New(apierr.MinimumVersionViolation, "")) {
		t.Fatal(e)
	}
	l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{l.old})
	v, e = l.s.NodeVersionView(l.ctx, i.NodeID)
	if e != nil || !v.RecoveryRequired {
		t.Fatal(v, e)
	}
	if e = l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: w.Intent.Generation, CurrentReleaseID: &l.new, LastGoodReleaseID: &l.new, ObservedAt: time.Now().UTC()}); e != nil {
		t.Fatal(e)
	}
	old := result(w, "failed")
	old.CurrentReleaseId.Set(l.old)
	old.ErrorCode.Set(code)
	ack, e := l.s.SubmitResult(l.ctx, i, payload(t, old))
	if e != nil || ack.AppliedToCurrentObservation {
		t.Fatal(ack, e)
	}
	v, e = l.s.NodeVersionView(l.ctx, i.NodeID)
	if e != nil || *v.CurrentReleaseID != l.new {
		t.Fatal(v, e)
	}
}
func TestUPD05_RetryCancellationWatchdogAndAtomicity(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "disabled", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	r := result(w, "failed")
	code := api.ErrorCode("DOWNLOAD_TIMEOUT")
	r.ErrorCode.Set(code)
	r.CurrentReleaseId.Set(l.old)
	if _, e := l.s.SubmitResult(l.ctx, i, payload(t, r)); e != nil {
		t.Fatal(e)
	}
	cmd := releasetest.Command()
	request := api.UpdateBatchRetryRequest{Reason: "TEST ONLY retry"}
	retry, e := l.s.Retry(l.ctx, cmd, b.BatchId, request)
	if e != nil || retry.TargetCount != 1 {
		t.Fatal(retry, e)
	}
	again, e := l.s.Retry(l.ctx, cmd, b.BatchId, request)
	if e != nil || again != retry {
		t.Fatal(again, e)
	}
	w = l.intent(i)
	if w.Intent.Attempt != 2 {
		t.Fatal(w)
	}
	l.report(i, w, "applying")
	if _, e = l.s.pool.Exec(l.ctx, `UPDATE update_jobs SET phase_since=clock_timestamp()-interval '2 minutes' WHERE id=$1`, w.Intent.JobId); e != nil {
		t.Fatal(e)
	}
	if e = l.s.Reconcile(l.ctx); e != nil {
		t.Fatal(e)
	}
	evidence, e := l.s.Evidence(l.ctx, nullableValue(w.Intent.TargetId))
	if e != nil || !evidence.NeedsAttention {
		t.Fatal(evidence, e)
	}
	got := l.get(b.BatchId)
	if got.TargetCount != 1 || got.Counts.Applying != 1 {
		t.Fatal(got)
	}
	if _, e = l.s.pool.Exec(l.ctx, `UPDATE nodes SET resource_version=resource_version+1,lifecycle_state='revoked' WHERE id=$1`, i.NodeID); e != nil {
		t.Fatal(e)
	}
	got = l.get(b.BatchId)
	if got.Counts.Cancelled != 1 || got.Targets[0].State != "cancelled_revoked" {
		t.Fatal(got)
	}
	if _, e = l.s.NodeUpdateIntent(l.ctx, i); !errors.Is(e, apierr.New(apierr.IdentityRevoked, "")) {
		t.Fatal(e)
	}
	// Inject failure after snapshot and jobs have been written: none may commit.
	before := l.scalar(`SELECT count(*) FROM update_batches`)
	l.node("arm64", "unpublished", "registered", true)
	_, e = l.s.pool.Exec(l.ctx, `CREATE FUNCTION test_fail_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'TEST ONLY injected failure'; END $$; CREATE TRIGGER test_fail_outbox BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION test_fail_outbox()`)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = l.s.CreateBatch(l.ctx, releasetest.Command(), l.request(l.new)); e == nil {
		t.Fatal("expected transaction failure")
	}
	if l.scalar(`SELECT count(*) FROM update_batches`) != before {
		t.Fatal("partial snapshot committed")
	}
}

func TestUPD02_ResultBindingExpiryAndRestore(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "published", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	cases := []struct {
		name   string
		change func(*api.UpdateResult)
		code   apierr.Code
	}{
		{"identity", func(r *api.UpdateResult) { r.NodeId = uuid.New() }, apierr.IncarnationMismatch},
		{"incarnation", func(r *api.UpdateResult) { r.IncarnationId = uuid.New() }, apierr.IncarnationMismatch},
		{"generation", func(r *api.UpdateResult) { r.Generation++ }, apierr.StaleGeneration},
		{"attempt", func(r *api.UpdateResult) { r.Attempt++ }, apierr.StaleGeneration},
		{"target", func(r *api.UpdateResult) { id := uuid.New(); r.TargetId.Set(id) }, apierr.StaleGeneration},
		{"manifest", func(r *api.UpdateResult) {
			r.ManifestSha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}, apierr.HashMismatch},
		{"health", func(r *api.UpdateResult) { h := nullableValue(r.Health); h.InternetOnly = "failed"; r.Health.Set(h) }, apierr.HealthCheckFailed},
		{"version", func(r *api.UpdateResult) { r.CurrentReleaseId.Set(l.old) }, apierr.HealthCheckFailed},
		{"checkpoint", func(r *api.UpdateResult) {
			c := nullableValue(r.UsageCheckpoint)
			c.CentralAckedThrough = 11
			r.UsageCheckpoint.Set(c)
		}, apierr.CheckpointFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := result(w, "succeeded")
			tc.change(&r)
			if _, e := l.s.SubmitResult(l.ctx, i, payload(t, r)); !errors.Is(e, apierr.New(tc.code, "")) {
				t.Fatal(e)
			}
		})
	}
	if _, e := l.s.pool.Exec(l.ctx, `UPDATE update_jobs SET authorization_valid_until=clock_timestamp()-interval '1 second' WHERE id=$1`, w.Intent.JobId); e != nil {
		t.Fatal(e)
	}
	if _, e := l.s.SubmitResult(l.ctx, i, payload(t, result(w, "applying"))); !errors.Is(e, apierr.New(apierr.AuthorizationExpired, "")) {
		t.Fatal(e)
	}
	w = l.intent(i)
	l.report(i, w, "checking")
	// A reordered progress report is accepted as history, without moving checking backwards.
	ack, e := l.s.SubmitResult(l.ctx, i, payload(t, result(w, "downloading")))
	if e != nil || !ack.AcceptedAsHistory {
		t.Fatal(ack, e)
	}
	if l.get(b.BatchId).Counts.Checking != 1 {
		t.Fatal("progress regressed")
	}
	// Restore reconciliation sees an agent ahead of the database and refuses automatic work.
	if e = l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: 100, CurrentReleaseID: &l.new, LastGoodReleaseID: &l.new, ObservedAt: time.Now().UTC()}); e != nil {
		t.Fatal(e)
	}
	if _, e = l.s.NodeUpdateIntent(l.ctx, i); !errors.Is(e, apierr.New(apierr.StaleGeneration, "")) {
		t.Fatal(e)
	}
	next := l.batch(l.new)
	var generation string
	if e = l.s.pool.QueryRow(l.ctx, `SELECT target_generation::text FROM update_targets WHERE batch_id=$1`, next.BatchId).Scan(&generation); e != nil || generation != "101" {
		t.Fatal(generation, e)
	}
	// A delayed lower observation sequence cannot overwrite the current version.
	if e = l.s.Observe(l.ctx, i, Observation{Sequence: 1, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC().Add(time.Hour)}); e != nil {
		t.Fatal(e)
	}
	view, e := l.s.NodeVersionView(l.ctx, i.NodeID)
	if e != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new {
		t.Fatal(view, e)
	}
}
func TestUPD05_PaginationRevokedTargetsAndRetryCompletion(t *testing.T) {
	l := newLab(t, "update_batch.retry")
	i := l.node("amd64", "disabled", "registered", true)
	l.node("arm64", "unpublished", "registered", true)
	b := l.batch(l.new)
	first, e := l.s.GetBatch(l.ctx, b.BatchId, "", 1)
	if e != nil || len(first.Targets) != 1 || (!first.NextCursor.IsSpecified() || first.NextCursor.IsNull()) {
		t.Fatal(first, e)
	}
	second, e := l.s.GetBatch(l.ctx, b.BatchId, nullableValue(first.NextCursor), 1)
	if e != nil || len(second.Targets) != 1 || first.Targets[0].TargetId == second.Targets[0].TargetId || !second.NextCursor.IsNull() {
		t.Fatal(second, e)
	}
	raw, _ := json.Marshal(first)
	if e = l.s.releases.Validator.ValidateBody("UpdateBatch", raw); e != nil {
		t.Fatal("response contract", e, string(raw))
	}
	raw, _ = json.Marshal(b)
	if e = l.s.releases.Validator.ValidateBody("UpdateBatchAccepted", raw); e != nil {
		t.Fatal(e)
	}
	w := l.intent(i)
	r := result(w, "failed")
	code := api.ErrorCode("STORE_UNAVAILABLE")
	r.ErrorCode.Set(code)
	if _, e = l.s.SubmitResult(l.ctx, i, payload(t, r)); e != nil {
		t.Fatal(e)
	}
	retry, e := l.s.Retry(l.ctx, releasetest.Command(), b.BatchId, api.UpdateBatchRetryRequest{Reason: "TEST ONLY retry"})
	if e != nil {
		t.Fatal(e)
	}
	w = l.intent(i)
	l.report(i, w, "succeeded")
	var status string
	if e = l.s.pool.QueryRow(l.ctx, `SELECT status FROM operations WHERE id=$1`, retry.OperationID).Scan(&status); e != nil || status != "succeeded" {
		t.Fatal(status, e)
	}
	l.support([]uuid.UUID{l.old}, []uuid.UUID{l.old}, []uuid.UUID{l.new})
	if _, e = l.s.CreateBatch(l.ctx, releasetest.Command(), l.request(l.new)); !errors.Is(e, apierr.New(apierr.ReleaseNotApproved, "")) {
		t.Fatal(e)
	}
	// Revocation is durable and cannot be silently undone by activating an old policy.
	p := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{l.old, l.new}, RevokedReleaseIds: []uuid.UUID{}}
	p.MinimumServingVersion.ServingReleaseIds = []uuid.UUID{l.old, l.new}
	e = db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return l.s.releases.ApplySupportTx(l.ctx, tx, uuid.New(), p, "TEST ONLY resurrection")
	})
	if !errors.Is(e, apierr.New(apierr.StateConflict, "")) {
		t.Fatal(e)
	}
}

func TestUPD04_ContractHeartbeatAdapter(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "published", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	r := api.ObservationReport{ReportId: uuid.New(), NodeId: i.NodeID, NodeIncarnationId: i.IncarnationID, ObservedAt: time.Now().UTC(), CurrentReleaseId: presence.NewNullableWithValue(l.new), LastGoodReleaseId: presence.NewNullableWithValue(l.new),
		AppliedBindingRevision:  presence.NewNullNullable[decstr.U64](),
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
		AppliedPolicyRefs:       presence.NewNullNullable[api.PolicyRefs]()}
	r.Update.JobId.Set(w.Intent.JobId)
	r.Update.Generation.Set(w.Intent.Generation)
	insertAndObserve := func(r api.ObservationReport) error {
		return db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			// The node owner holds the fleet lock before inserting its validated receipt.
			if e := l.s.lock(l.ctx, tx); e != nil {
				return e
			}
			_, e := tx.Exec(l.ctx, `INSERT INTO node_observations(node_id,incarnation_id,report_id,payload_hash,observed_at,applied_generation,actual_release_id,schema_version,checks,control_status,data_health,safety_status) VALUES($1,$2,$3,$4,$5,0,$6,1,'{}','online','healthy','passed') ON CONFLICT(node_id,incarnation_id,report_id) DO NOTHING`, i.NodeID, i.IncarnationID, r.ReportId, make([]byte, 32), r.ObservedAt, nullableValue(r.CurrentReleaseId))
			if e != nil {
				return e
			}
			return l.s.ObserveReportTx(l.ctx, tx, i, r)
		})
	}
	if e := insertAndObserve(r); e != nil {
		t.Fatal(e)
	}
	if e := insertAndObserve(r); e != nil {
		t.Fatal(e)
	}
	r.ReportId = uuid.New()
	r.ObservedAt = r.ObservedAt.Add(-time.Second)
	r.CurrentReleaseId.Set(l.old)
	if e := insertAndObserve(r); e != nil {
		t.Fatal(e)
	}
	view, e := l.s.NodeVersionView(l.ctx, i.NodeID)
	if e != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new {
		t.Fatal(view, e)
	}
}
