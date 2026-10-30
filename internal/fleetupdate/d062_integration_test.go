//go:build integration

package fleetupdate

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// JSON fixtures supply evidence shapes, not projector rows. Identity, timestamps
// and hash are rebound to an actual authorized job in this isolated domain lab.
func d062Result(t *testing.T, l *lab, w *Work, name string) api.UpdateResult {
	t.Helper()
	raw, err := os.ReadFile("testdata/d062-" + name + ".json")
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
		t.Fatalf("invalid fixture hash: %v", err)
	}
	r.ResultId = uuid.New()
	r.NodeId = w.Intent.NodeId
	r.IncarnationId = w.Intent.IncarnationId
	r.JobId = w.Intent.JobId
	r.BatchId = w.Intent.BatchId
	r.TargetId = w.Intent.TargetId
	r.Attempt = w.Intent.Attempt
	r.Generation = w.Intent.Generation
	r.RequestedReleaseId = w.Intent.ReleaseId
	r.ManifestSha256 = w.Intent.ManifestSha256
	r.PublicationIntent = w.Intent.PublicationIntent
	r.CurrentReleaseId.Set(l.old)
	r.StartedAt = time.Now().UTC()
	r.FinishedAt.Set(r.StartedAt)
	if r.State == api.UpdateResultStateRolledBack {
		r.PreviousReleaseId.Set(w.Intent.ReleaseId)
	}
	return r
}

func TestD062_PreHealthFailure(t *testing.T) {
	for _, stage := range []api.UpdateStage{api.UpdateStageClaim, api.UpdateStageDownload, api.UpdateStageVerify, api.UpdateStagePreflight, api.UpdateStagePrepare, api.UpdateStageBlockForwarding, api.UpdateStageFinalSample, api.UpdateStageStopServices, api.UpdateStageMigrate, api.UpdateStageSwitch, api.UpdateStageStart} {
		t.Run(string(stage), func(t *testing.T) {
			l := newLab(t)
			i := l.node("amd64", "disabled", "registered", true)
			b := l.batch(l.new)
			w := l.intent(i)
			r := d062Result(t, l, w, "pre-health-failure")
			r.Stage.Set(stage)
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			if err != nil || !ack.Accepted {
				t.Fatal(ack, err)
			}
			target := l.get(b.BatchId).Targets[0]
			if target.State != "failed" || !target.NeedsAttention || nullableValue(target.CurrentReleaseId) != l.old {
				t.Fatal(target)
			}
			var attention bool
			if err = l.s.pool.QueryRow(l.ctx, `SELECT needs_attention FROM update_jobs WHERE id=$1`, w.Intent.JobId).Scan(&attention); err != nil || !attention {
				t.Fatal(attention, err)
			}
			replay, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			if err != nil || replay != ack {
				t.Fatal(replay, err)
			}
		})
	}
}

func TestD062_PreHealthEvidenceBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*api.UpdateResult)
	}{
		{"candidate current", func(r *api.UpdateResult) { r.CurrentReleaseId.Set(r.RequestedReleaseId) }},
		{"missing error", func(r *api.UpdateResult) { r.ErrorCode.SetNull() }},
		{"missing finish", func(r *api.UpdateResult) { r.FinishedAt.SetNull() }},
		{"invented health", func(r *api.UpdateResult) {
			r.Health.Set(api.UpdateHealth{Processes: "passed", IdentityAndState: "passed", InternetOnly: "passed", DnsPolicy: "passed", TunnelProbe: "passed"})
		}},
		{"checking needs health", func(r *api.UpdateResult) { r.Stage.Set(api.UpdateStageCheck) }},
		{"commit needs health", func(r *api.UpdateResult) { r.Stage.Set(api.UpdateStageCommit) }},
		{"rollback needs health", func(r *api.UpdateResult) { r.Stage.Set(api.UpdateStageRollback) }},
		{"report needs health", func(r *api.UpdateResult) { r.Stage.Set(api.UpdateStageReport) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLab(t)
			i := l.node("arm64", "disabled", "registered", true)
			b := l.batch(l.new)
			w := l.intent(i)
			r := d062Result(t, l, w, "pre-health-failure")
			tc.change(&r)
			_, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			var ae *apierr.Error
			if !errors.As(err, &ae) || ae.HTTPStatus() != 422 || ae.IsRetryable() {
				t.Fatal(err)
			}
			if l.get(b.BatchId).Counts.Failed != 0 {
				t.Fatal("rejected evidence changed target")
			}
		})
	}
}

func TestD062_RollbackEvidenceAndAllowlist(t *testing.T) {
	for _, mode := range []string{"empty", "allowed", "excluded", "revoked", "revoked_allowlisted", "serving_not_approved", "missing_candidate", "missing_error"} {
		t.Run(mode, func(t *testing.T) {
			l := newLab(t)
			m := l.f.Manifest
			m.ReleaseId = uuid.New()
			m.ProductVersion = "1.3.0-test"
			m.Rollback.AllowedReleaseIds = []uuid.UUID{}
			if mode == "allowed" || mode == "revoked_allowlisted" {
				m.Rollback.AllowedReleaseIds = []uuid.UUID{l.old}
			}
			if mode == "excluded" {
				m.Rollback.AllowedReleaseIds = []uuid.UUID{l.new}
			}
			if _, err := l.s.releases.Register(l.ctx, releasetest.Command(), l.f.Request(t, m)); err != nil {
				t.Fatal(err)
			}
			l.support([]uuid.UUID{l.old, l.new, m.ReleaseId}, []uuid.UUID{l.old, l.new, m.ReleaseId}, []uuid.UUID{})
			i := l.node("amd64", "disabled", "registered", true)
			b := l.batch(m.ReleaseId)
			w := l.intent(i)
			l.report(i, w, "checking")
			r := d062Result(t, l, w, "rollback")
			if mode == "revoked" || mode == "revoked_allowlisted" {
				l.support([]uuid.UUID{l.new, m.ReleaseId}, []uuid.UUID{l.new, m.ReleaseId}, []uuid.UUID{l.old})
			}
			if mode == "serving_not_approved" {
				l.support([]uuid.UUID{m.ReleaseId}, []uuid.UUID{l.old, m.ReleaseId}, []uuid.UUID{})
			}
			if mode == "missing_candidate" {
				r.PreviousReleaseId.SetNull()
			}
			if mode == "missing_error" {
				r.ErrorCode.SetNull()
			}
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			if mode == "excluded" || mode == "revoked" || mode == "revoked_allowlisted" || mode == "missing_candidate" || mode == "missing_error" {
				var ae *apierr.Error
				want := apierr.ValidationFailed
				if mode == "excluded" {
					want = apierr.SchemaIncompatible
				}
				if mode == "revoked" || mode == "revoked_allowlisted" {
					want = apierr.MinimumVersionViolation
				}
				if !errors.As(err, &ae) || ae.HTTPStatus() != 422 || ae.Code != want {
					t.Fatal(ack, err)
				}
				return
			}
			if err != nil || !ack.Accepted {
				t.Fatal(ack, err)
			}
			if got := l.get(b.BatchId); got.Counts.RolledBack != 1 || got.Counts.Succeeded != 0 {
				t.Fatal(got)
			}
			var raw []byte
			if err = l.s.pool.QueryRow(l.ctx, `SELECT payload FROM update_results WHERE result_id=$1`, r.ResultId).Scan(&raw); err != nil {
				t.Fatal(err)
			}
			var stored api.UpdateResult
			if err = json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			if nullableValue(stored.CurrentReleaseId) != l.old || nullableValue(stored.PreviousReleaseId) != m.ReleaseId || nullableValue(stored.ErrorCode) != api.ErrorCode(apierr.HealthCheckFailed) || !healthyValue(stored) {
				t.Fatal(stored)
			}
		})
	}
}
func healthyValue(r api.UpdateResult) bool {
	h, err := r.Health.Get()
	return err == nil && healthy(&h)
}

func TestD062_RejectedReplay(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "disabled", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	l.report(i, w, "checking")
	l.support([]uuid.UUID{l.new}, []uuid.UUID{l.new}, []uuid.UUID{})
	r := d062Result(t, l, w, "rejected-result")
	raw := payload(t, r)
	_, first := l.s.SubmitResult(l.ctx, i, raw)
	if !errors.Is(first, apierr.New(apierr.MinimumVersionViolation, "")) {
		t.Fatal(first)
	}
	l.support([]uuid.UUID{l.old, l.new}, []uuid.UUID{l.old, l.new}, []uuid.UUID{})
	// A new service instance rules out an in-memory rejection cache.
	other, err := New(l.s.pool, l.s.releases, l.s.config)
	if err != nil {
		t.Fatal(err)
	}
	_, again := other.SubmitResult(l.ctx, i, raw)
	if again == nil || again.Error() != first.Error() {
		t.Fatalf("rejected result changed after policy activation: first=%v replay=%v", first, again)
	}
	var ae *apierr.Error
	if !errors.As(again, &ae) || ae.HTTPStatus() != 422 || ae.IsRetryable() {
		t.Fatal(again)
	}
}

// Receipt rows are immutable input owned by heartbeat ingestion, not handmade
// fleet/job projector state (D-039/D-050). Projection runs only through the API.
func d062Observe(t *testing.T, l *lab, i Identity, w *Work, known bool) {
	t.Helper()
	r := api.ObservationReport{ReportId: uuid.New(), NodeId: i.NodeID, NodeIncarnationId: i.IncarnationID, ObservedAt: time.Now().UTC()}
	r.Update.Generation.SetNull()
	r.Update.JobId.SetNull()
	r.Update.Stage.SetNull()
	r.Update.State.SetNull()
	r.CurrentReleaseId.Set(l.old)
	r.LastGoodReleaseId.Set(l.old)
	if known {
		r.Update.Generation.Set(w.Intent.Generation)
		r.Update.JobId.Set(w.Intent.JobId)
		r.Update.State.Set(api.UpdateJobStateSucceeded)
		r.Update.Stage.Set(api.UpdateStageReport)
		r.CurrentReleaseId.Set(l.new)
		r.LastGoodReleaseId.Set(l.new)
	}
	// Build a raw receipt from the observation block; other node-owner validation
	// is outside this domain fixture.
	raw, err := json.Marshal(map[string]any{"update": r.Update})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := canonjson.Hash(raw)
	if err != nil {
		t.Fatal(err)
	}
	err = db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if err := l.s.lock(l.ctx, tx); err != nil {
			return err
		}
		_, err := tx.Exec(l.ctx, `INSERT INTO node_observations(node_id,incarnation_id,report_id,payload_hash,observed_at,applied_generation,actual_release_id,schema_version,checks,control_status,data_health,safety_status,report_payload) VALUES($1,$2,$3,$4,$5,0,$6,1,'{}','online','healthy','passed',$7)`, i.NodeID, i.IncarnationID, r.ReportId, hash[:], r.ObservedAt, nullableValue(r.CurrentReleaseId), raw)
		if err != nil {
			return err
		}
		return l.s.ObserveReportTx(l.ctx, tx, i, r)
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestD062_NullGenerationNeverConfirmsOrDemotes(t *testing.T) {
	l := newLab(t)
	i := l.node("arm64", "disabled", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	l.report(i, w, "succeeded")
	for _, known := range []bool{false, true, false} {
		d062Observe(t, l, i, w, known)
		if !known {
			// Losing current update evidence must not lose the ordering fence.
			if err := l.s.Observe(l.ctx, i, Observation{Sequence: 100, AppliedUpdateGeneration: 0, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
				t.Fatal(err)
			}
		}
		view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(view)
		var obj map[string]any
		_ = json.Unmarshal(raw, &obj)
		if !known && obj["ObservedGeneration"] != nil {
			t.Fatalf("null generation projected as %v", obj["ObservedGeneration"])
		}
		target := l.get(b.BatchId).Targets[0]
		evidence, err := l.s.Evidence(l.ctx, target.TargetId)
		if err != nil || target.Unconfirmed == known || view.Unconfirmed == known || evidence.Unconfirmed == known || target.State != "succeeded" || nullableValue(target.CurrentReleaseId) != l.new {
			t.Fatal(view, target, evidence, err)
		}
	}
}

func TestD062_ConflictAndRateLimitRetryMeaning(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "disabled", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	r := result(w, "succeeded")
	r.Generation++
	_, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.StaleGeneration || !ae.IsRetryable() {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/update-results", nil)
	for _, e := range []*apierr.Error{ae, apierr.New(apierr.RateLimited, "Request rate exceeded").WithRetryAfter(time.Second)} {
		response := httptest.NewRecorder()
		apierr.Write(response, req, e)
		if response.Code != 409 && response.Code != 429 {
			t.Fatal(response.Code)
		}
		if response.Code == 429 && response.Header().Get("Retry-After") == "" {
			t.Fatal("missing Retry-After")
		}
		var body struct {
			Error struct {
				Retryable bool `json:"retryable"`
			}
		}
		if err = json.Unmarshal(response.Body.Bytes(), &body); err != nil || !body.Error.Retryable {
			t.Fatal(response.Body.String(), err)
		}
	}
}

func TestD062_RejectedReceiptTransaction(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "disabled", "registered", true)
	b := l.batch(l.new)
	w := l.intent(i)
	r := d062Result(t, l, w, "pre-health-failure")
	r.Stage.Set(api.UpdateStageCheck)
	var rejected error
	err := db.InTx(l.ctx, l.s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		_, rejected = l.s.SubmitResultTx(l.ctx, tx, i, payload(t, r))
		var ae *apierr.Error
		if errors.As(rejected, &ae) && ae.Code == apierr.ValidationFailed {
			return nil
		} // Commit the rejection receipt, not success.
		return rejected
	})
	if err != nil || rejected == nil {
		t.Fatal(rejected, err)
	}
	_, again := l.s.SubmitResult(l.ctx, i, payload(t, r))
	if again == nil || again.Error() != rejected.Error() {
		t.Fatal(again)
	}
	if got := l.get(b.BatchId); got.Counts.Failed != 0 || got.Counts.Queued != 1 {
		t.Fatal(got)
	}
}

func TestD062_UnknownAndGreaterGeneration(t *testing.T) {
	l := newLab(t)
	i := l.node("arm64", "disabled", "registered", false)
	view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
	if err != nil || view.ObservedGeneration != nil {
		t.Fatal(view, err)
	}
	b := l.batch(l.new)
	w := l.intent(i)
	l.report(i, w, "succeeded")
	err = l.s.Observe(l.ctx, i, Observation{Sequence: 1, AppliedUpdateGeneration: w.Intent.Generation + 1, CurrentReleaseID: &l.new, LastGoodReleaseID: &l.new, ObservedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if target := l.get(b.BatchId).Targets[0]; !target.Unconfirmed || target.State != "succeeded" {
		t.Fatal(target)
	}
}

func TestD062_PermanentRejectionCodes(t *testing.T) {
	for _, code := range []apierr.Code{apierr.MalformedRequest, apierr.Unauthenticated, apierr.IncarnationMismatch, apierr.ResourceNotFound, apierr.HealthCheckFailed} {
		err := resultError(apierr.New(code, "TEST ONLY precise rejection"))
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.IsRetryable() {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		apierr.Write(w, httptest.NewRequest(http.MethodPost, "/api/v1/agent/update-results", nil), err)
		var body struct {
			Error struct {
				Code      apierr.Code `json:"code"`
				Retryable bool        `json:"retryable"`
			}
		}
		if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil || body.Error.Code != code || body.Error.Retryable || w.Code != code.HTTPStatus() {
			t.Fatal(w.Body.String(), e)
		}
	}
}
