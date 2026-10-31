//go:build integration

package fleetupdate

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The harness records are redacted partial payloads. Keep their state, stage,
// error and health verbatim; bind IDs, hashes and times to real authorized jobs.
func TestD062a_PrecheckFailureHarnessDelivery(t *testing.T) {
	// D-132 source: docs/e2e/at/updates/history/follow-up-9/result-delivery.json; SHA-256 df19c0968d3e60e420fef36754f4b9c946f771279846fce29ec33450240f7d55.
	raw, err := os.ReadFile("testdata/d062-harness-result-delivery.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Results []api.UpdateResult `json:"results"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Results) == 0 {
		t.Fatal("missing harness precheck result")
	}
	recorded := fixture.Results[0]
	if recorded.State != "failed" || !recorded.Health.IsNull() {
		t.Fatal("harness precheck shape changed")
	}
	l := newLab(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	r := result(w, string(recorded.State))
	r.Stage = recorded.Stage
	r.Health = recorded.Health
	r.ErrorCode = recorded.ErrorCode
	r.CurrentReleaseId.Set(l.old)
	if _, err = l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
		t.Fatal(err)
	}
	page := l.get(b.BatchId)
	if page.Targets[0].State != "failed" || !page.Targets[0].NeedsAttention {
		t.Fatal(page.Targets[0])
	}
	var state, code string
	var attention bool
	if err = l.s.pool.QueryRow(l.ctx, `SELECT state,error_code,needs_attention FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&state, &code, &attention); err != nil || state != "failed" || code != "ARTIFACT_INVALID" || !attention {
		t.Fatal(state, code, attention, err)
	}
	for _, stage := range []api.UpdateStage{api.UpdateStageCheck, api.UpdateStageCommit, api.UpdateStageReport, api.UpdateStageRollback} {
		r.ResultId = uuid.New()
		r.Stage.Set(stage)
		if _, err = l.s.SubmitResult(l.ctx, i, payload(t, r)); err == nil {
			t.Fatal("post-check failure without health accepted", stage)
		}
	}
}

func TestD062c_RollbackHarnessRequiresRestoredHealthEmptyAllowlist(t *testing.T) {
	// D-132 source: docs/e2e/at/updates/history/follow-up-9/rollback-delivery-finding.json; SHA-256 c67f2d85cd75e494310e64a1f135cc1d67781fd4e97876190937bda428e1485c.
	raw, err := os.ReadFile("testdata/d062-harness-rollback-delivery-finding.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Observations []struct{ Name, Detail string }
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	var recorded api.UpdateResult
	var journal struct {
		Attestations []struct {
			Phase     string                `json:"phase"`
			DNS       api.HealthCheckStatus `json:"agent_dns_policy"`
			Isolation api.HealthCheckStatus `json:"agent_internet_only"`
			Tunnel    api.HealthCheckStatus `json:"agent_tunnel_probe"`
		} `json:"attestations"`
	}
	for _, o := range fixture.Observations {
		if o.Name == "journal" {
			if err = json.Unmarshal([]byte(o.Detail), &journal); err != nil {
				t.Fatal(err)
			}
		}
		if o.Name == "serialized_rollback_result" {
			if err = json.Unmarshal([]byte(o.Detail), &recorded); err != nil {
				t.Fatal(err)
			}
		}
	}
	if recorded.State != "rolled_back" {
		t.Fatal("missing harness rollback")
	}
	l := newLab(t)
	candidate := l.f.Manifest
	candidate.ReleaseId = uuid.New()
	candidate.ProductVersion = "1.3.0-test"
	candidate.Rollback.AllowedReleaseIds = []uuid.UUID{}
	if _, err = l.s.releases.Register(l.ctx, releasetest.Command(), l.f.Request(t, candidate)); err != nil {
		t.Fatal(err)
	}
	l.support([]uuid.UUID{l.old, candidate.ReleaseId}, []uuid.UUID{l.old, candidate.ReleaseId}, []uuid.UUID{})
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(candidate.ReleaseId)
	w := l.intent(i)
	// This covers progress followed by a terminal result. The real producer's
	// direct queued-to-rollback path is covered by TestAT26_DirectRollbackFromQueued.
	l.report(i, w, "applying")
	r := result(w, string(recorded.State))
	// D-062(b): rebind the recorded candidate identity with the authorized job.
	r.PreviousReleaseId.Set(w.Intent.ReleaseId)
	r.Stage = recorded.Stage
	r.Health = recorded.Health
	r.ErrorCode = recorded.ErrorCode
	r.CurrentReleaseId.Set(l.old)
	if _, err = l.s.SubmitResult(l.ctx, i, payload(t, r)); err == nil {
		t.Fatal("failed candidate health was accepted as restored health")
	}
	// The same record's rollback health_attest reports all five checks passed.
	health, _ := r.Health.Get()
	restored := false
	for _, evidence := range journal.Attestations {
		if evidence.Phase == "rollback" {
			health.DnsPolicy, health.InternetOnly, health.TunnelProbe = evidence.DNS, evidence.Isolation, evidence.Tunnel
			restored = true
		}
	}
	if !restored || !healthy(&health) {
		t.Fatal("recorded restored release is not healthy")
	}
	r.Health.Set(health)
	// D-062(d): corrected evidence is a new result, never a changed replay of
	// the permanently rejected payload above.
	r.ResultId = uuid.New()
	if _, err = l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
		t.Fatal("empty allowlist must not reject serving restored release", err)
	}
	if page := l.get(b.BatchId); page.Targets[0].State != "rolled_back" {
		t.Fatal(page.Targets[0])
	}
}

func TestD062e_NullUpdateGenerationRetainsOrderingWithoutConfirmation(t *testing.T) {
	l := newLabState(t)
	i := l.node("amd64", "unpublished", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	l.report(i, w, "succeeded") // advances stored version generation, not observation evidence
	raw, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	report.NodeId, report.NodeIncarnationId, report.ReportId = i.NodeID, i.IncarnationID, uuid.New()
	report.ObservedAt = time.Now().UTC()
	report.Update.JobId.SetNull()
	report.Update.Generation.SetNull()
	report.Update.State.SetNull()
	report.CurrentReleaseId.Set(l.new)
	report.LastGoodReleaseId.Set(l.new)
	report.Netbird.PeerId.SetNull()
	report.MemberForwardingEnabled = false
	report.AppliedConfigGeneration.SetNull()
	svc := nodes.NewService(l.s.pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, r api.ObservationReport) error {
		return l.s.ObserveReportTx(ctx, tx, i, r)
	}))
	if _, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
		t.Fatal(err)
	}
	projection, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || projection.ObservedGeneration != nil {
		t.Fatal("null update generation must remain absent in the node view", projection, err)
	}
	// The original harness assertion below checks the stored ordering fence.
	// D-062(e) makes the public observation projection nullable, so read the
	// numeric high-water mark separately instead of conflating the two values.
	var view struct {
		ObservedGeneration decstr.U64
		LastObservedAt     *time.Time
	}
	var storedGeneration string
	err = l.s.pool.QueryRow(l.ctx, `SELECT observed_update_generation::text,last_observed_at FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&storedGeneration, &view.LastObservedAt)
	if err == nil {
		view.ObservedGeneration, err = decstr.Parse(storedGeneration)
	}
	if err != nil || view.ObservedGeneration != w.Intent.Generation || view.LastObservedAt == nil || !view.LastObservedAt.Equal(report.ObservedAt.Truncate(time.Microsecond)) {
		t.Fatal("routine observation rejected or generation lowered", view, err)
	}
	if !l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("null generation fabricated job confirmation")
	}
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(time.Second)
	report.Update.Generation.Set(w.Intent.Generation)
	report.Update.JobId.Set(w.Intent.JobId)
	report.Update.State = presence.NewNullableWithValue(api.UpdateJobStateSucceeded)
	if _, err = svc.SubmitObservation(l.ctx, i.NodeID, i.IncarnationID, report); err != nil {
		t.Fatal(err)
	}
	if l.get(b.BatchId).Targets[0].Unconfirmed {
		t.Fatal("matching update generation did not confirm")
	}
}
