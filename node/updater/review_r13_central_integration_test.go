//go:build integration

package updater

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestReviewR13_Q5_CentralAcceptsPreCheckRecovery submits the sealed pre-check
// recovery shape to a real fleetupdate.SubmitResult. fakecontrol does not apply
// the D-062 health rule. The fleetupdate lab helpers are unexported, so this
// test builds the same isolated database the central integration harness uses.
//
// The rejected control is the pre-fix document: stage report and health null.
// SubmitResult returns 422 VALIDATION_FAILED with
// "Terminal result at or after health checks requires health evidence".
func TestReviewR13_Q5_CentralAcceptsPreCheckRecovery(t *testing.T) {
	sealed := sealedPreCheckRecovery(t)
	stage := requiredValue(t, sealed.Stage)
	if stage != api.UpdateStageRollback || !sealed.Health.IsSpecified() || sealed.Health.IsNull() {
		t.Fatalf("updater did not seal a rollback recovery with health: stage %s", stage)
	}

	l := newCentralLab(t)
	id := l.node("amd64", "disabled", "registered", true)
	l.batch(l.next)
	work := l.intent(id)

	rejected := recoveryForJob(t, work, sealed, l.old)
	rejected.ResultId = uuid.Must(uuid.NewV7())
	rejected.Stage.Set(api.UpdateStageReport)
	rejected.Health.SetNull()
	rawRejected := resultPayload(t, rejected)
	_, err := l.svc.SubmitResult(l.ctx, id, rawRejected)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.HTTPStatus() != 422 || ae.Code != apierr.ValidationFailed || !strings.Contains(ae.Message, centralHealthEvidenceMessage) {
		t.Fatalf("pre-fix document was not rejected by central: %v\n%s", err, rawRejected)
	}

	accepted := recoveryForJob(t, work, sealed, l.old)
	ack, err := l.svc.SubmitResult(l.ctx, id, resultPayload(t, accepted))
	if err != nil || !ack.Accepted {
		t.Fatalf("central rejected the sealed recovery: ack=%+v err=%v", ack, err)
	}
	var state, storedStage string
	var recovery bool
	if err = l.pool.QueryRow(l.ctx, `SELECT state, recovery_required, stage FROM update_jobs WHERE id=$1`, work.Intent.JobId).Scan(&state, &recovery, &storedStage); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || !recovery || storedStage != string(api.UpdateStageRollback) {
		t.Fatalf("job state=%s recovery=%v stage=%s", state, recovery, storedStage)
	}
}

func sealedPreCheckRecovery(t *testing.T) api.UpdateResult {
	t.Helper()
	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	w.seed(r1)
	w.rollbackAllow = []uuid.UUID{r1.id}
	r2 := w.makeRelease("r2", 1, []int{1})
	w.svc.stopErr = errors.New("stop failed before health check")
	work := w.work(r2, "5", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{r2.id}
	w.setWork(work)
	_ = w.run()
	return w.oneResult()
}

func recoveryForJob(t *testing.T, w *fleetupdate.Work, sealed api.UpdateResult, current uuid.UUID) api.UpdateResult {
	t.Helper()
	now := time.Now().UTC().Add(time.Second).Truncate(time.Second)
	r := sealed
	i := w.Intent
	r.ResultId = uuid.Must(uuid.NewV7())
	r.NodeId = i.NodeId
	r.IncarnationId = i.IncarnationId
	r.BootId = uuid.Must(uuid.NewV7())
	r.BatchId = i.BatchId
	r.TargetId = i.TargetId
	r.JobId = i.JobId
	r.Attempt = i.Attempt
	r.Generation = i.Generation
	r.RequestedReleaseId = i.ReleaseId
	r.ManifestSha256 = i.ManifestSha256
	r.PublicationIntent = i.PublicationIntent
	r.CurrentReleaseId.Set(current)
	r.PreviousReleaseId.Set(current)
	r.StartedAt = now
	r.FinishedAt.Set(now)
	r.PayloadHash = ""
	if r.StateSchemaVersion < 1 {
		r.StateSchemaVersion = 1
	}
	return r
}

func resultPayload(t *testing.T, r api.UpdateResult) []byte {
	t.Helper()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	r.PayloadHash, err = canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// centralLab is the smallest real control plane that can authorize a job and
// accept an UpdateResult. It follows the fleetupdate integration harness.
type centralLab struct {
	t    *testing.T
	svc  *fleetupdate.Service
	rel  *releases.Service
	pool *pgxpool.Pool
	ctx  context.Context
	old  uuid.UUID
	next uuid.UUID
}

func newCentralLab(t *testing.T) *centralLab {
	t.Helper()
	pool := dbtest.New(t)
	fix := releasetest.New(t)
	rel := &releases.Service{Pool: pool, Validator: fix.Validator}
	svc, err := fleetupdate.New(pool, rel, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	l := &centralLab{t: t, svc: svc, rel: rel, pool: pool, ctx: context.Background(), old: fix.Manifest.ReleaseId}
	if _, err = rel.Register(l.ctx, releasetest.Command(), fix.Request(t, fix.Manifest)); err != nil {
		t.Fatalf("register current release: %v", err)
	}
	fix.Manifest.ReleaseId = uuid.Must(uuid.NewV7())
	fix.Manifest.ProductVersion = "1.2.0-test"
	fix.Manifest.Rollback.AllowedReleaseIds = []uuid.UUID{l.old}
	l.next = fix.Manifest.ReleaseId
	if _, err = rel.Register(l.ctx, releasetest.Command(), fix.Request(t, fix.Manifest)); err != nil {
		t.Fatalf("register candidate release: %v", err)
	}
	l.support([]uuid.UUID{l.old, l.next}, []uuid.UUID{l.old, l.next}, []uuid.UUID{})
	return l
}

func (l *centralLab) support(approved, serving, revoked []uuid.UUID) {
	l.t.Helper()
	p := api.ReleaseSupportPayload{ApprovedReleaseIds: approved, RevokedReleaseIds: revoked}
	p.MinimumServingVersion.ServingReleaseIds = serving
	err := db.InTx(l.ctx, l.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return l.rel.ApplySupportTx(l.ctx, tx, uuid.New(), p, "TEST ONLY policy")
	})
	if err != nil {
		l.t.Fatalf("release support: %v", err)
	}
}

func (l *centralLab) node(arch, pub, lifecycle string, online bool) fleetupdate.Identity {
	l.t.Helper()
	id := fleetupdate.Identity{NodeID: uuid.Must(uuid.NewV7())}
	var fp [32]byte
	_, _ = rand.Read(fp[:])
	if _, err := l.pool.Exec(l.ctx, `INSERT INTO nodes(id,lifecycle_state,current_release_id,last_good_release_id) VALUES($1,$2,$3,$3)`, id.NodeID, lifecycle, l.old); err != nil {
		l.t.Fatal(err)
	}
	if _, err := l.pool.Exec(l.ctx, `INSERT INTO exits(node_id,display_name,region,publication_intent) VALUES($1,'TEST ONLY node','test',$2)`, id.NodeID, pub); err != nil {
		l.t.Fatal(err)
	}
	id.IncarnationID = uuid.Must(uuid.NewV7())
	if _, err := l.pool.Exec(l.ctx, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch) VALUES($1,$2,$3,'ubuntu/24.04',$4)`, id.IncarnationID, id.NodeID, fp[:], arch); err != nil {
		l.t.Fatal(err)
	}
	if _, err := l.pool.Exec(l.ctx, `UPDATE nodes SET resource_version=resource_version+1,current_incarnation_id=$2 WHERE id=$1`, id.NodeID, id.IncarnationID); err != nil {
		l.t.Fatal(err)
	}
	if online {
		if err := l.svc.Observe(l.ctx, id, fleetupdate.Observation{Sequence: 1, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
			l.t.Fatal(err)
		}
	}
	return id
}

func (l *centralLab) batch(release uuid.UUID) {
	l.t.Helper()
	var g string
	if err := l.pool.QueryRow(l.ctx, `SELECT generation::text FROM fleet_config`).Scan(&g); err != nil {
		l.t.Fatal(err)
	}
	gen, err := decstr.Parse(g)
	if err != nil {
		l.t.Fatal(err)
	}
	req := api.UpdateBatchCreateRequest{ExpectedGeneration: gen, ReleaseId: release, Reason: "TEST ONLY update", Scope: "all_nodes", SetFleetDefault: true}
	if _, err = l.svc.CreateBatch(l.ctx, releasetest.Command(), req); err != nil {
		l.t.Fatal(err)
	}
}

func (l *centralLab) intent(id fleetupdate.Identity) *fleetupdate.Work {
	l.t.Helper()
	w, err := l.svc.NodeUpdateIntent(l.ctx, id)
	if err != nil || w == nil {
		l.t.Fatal("intent", w, err)
	}
	return w
}
