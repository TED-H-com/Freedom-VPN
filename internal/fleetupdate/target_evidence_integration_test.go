//go:build integration

package fleetupdate

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// NewTargetEvidenceFixture is test-only access for the external HTTP test to
// compose the real GET handler with this domain lab, without an import cycle.
// Every receipt and ledger row is produced by the domain commands (D-050).
func NewTargetEvidenceFixture(t *testing.T, scenario string) (*Service, uuid.UUID, *api.UpdateResult) {
	t.Helper()
	var l *lab
	var i Identity
	var b api.UpdateBatchAccepted
	var r api.UpdateResult
	if scenario == "rolled_back" || scenario == "rolled_back_observed" {
		l, i, b, _, r = at26DirectRollback(t)
		if scenario == "rolled_back_observed" {
			if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: r.Generation, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		l = newLab(t)
		i = l.node("arm64", "disabled", "registered", true)
		b = l.batch(l.new)
		w := l.intent(i)
		switch scenario {
		case "no_result":
			return l.s, b.BatchId, nil
		case "succeeded":
			r = result(w, "succeeded")
			r.PreviousReleaseId.Set(l.old)
			// The input permits this field on other states; revision 7 requires
			// the target projection to suppress it unless the result rolled back.
			r.CandidateHealth.Set(nullableValue(r.Health))
		case "failed":
			r = d062Result(t, l, w, "pre-health-failure")
			r.Stage.Set(api.UpdateStageVerify)
			r.ErrorCode.Set(api.ErrorCode(apierr.ArtifactInvalid))
		default:
			t.Fatalf("unknown scenario %q", scenario)
		}
	}
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
	if err != nil || !ack.Accepted || ack.AcceptedAsHistory {
		t.Fatal(ack, err)
	}
	if scenario == "rolled_back_observed" && ack.AppliedToCurrentObservation {
		t.Fatal("same-generation heartbeat already supplied the version", ack)
	}
	return l.s, b.BatchId, &r
}

func assertTargetResultEvidence(t *testing.T, target api.UpdateTarget, want *api.UpdateResult) {
	t.Helper()
	var previous *uuid.UUID
	var health, candidate *api.UpdateHealth
	if want != nil {
		if v, err := want.PreviousReleaseId.Get(); err == nil {
			previous = &v
		}
		if v, err := want.Health.Get(); err == nil {
			health = &v
		}
		if v, err := want.CandidateHealth.Get(); err == nil && want.State == api.UpdateResultStateRolledBack {
			candidate = &v
		}
	}
	raw, err := json.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]any{"previous_release_id": previous, "health": health, "candidate_health": candidate} {
		expected, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var a, b any
		if err = json.Unmarshal(got[field], &a); err != nil {
			t.Fatalf("%s must be present, including explicit null: %s (%v)", field, raw, err)
		}
		if err = json.Unmarshal(expected, &b); err != nil || !reflect.DeepEqual(a, b) {
			t.Fatalf("%s = %s, want %s (%v)", field, got[field], expected, err)
		}
	}
}

func TestUPD05_TargetResultEvidenceHistory(t *testing.T) {
	l, i, b, w, r := at26DirectRollback(t)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
		t.Fatal(err)
	}
	assertTargetResultEvidence(t, l.get(b.BatchId).Targets[0], &r)
	newBatch := l.batch(w.Intent.ReleaseId)
	current := result(l.intent(i), "succeeded")
	current.PreviousReleaseId.Set(l.old)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, current)); err != nil {
		t.Fatal(err)
	}
	// This older-generation receipt arrives last, with different candidate
	// checks. Neither its own target nor the newer target may borrow it.
	late := r
	late.ResultId = uuid.New()
	late.CandidateHealth.Set(api.UpdateHealth{Processes: "failed", IdentityAndState: "failed", InternetOnly: "failed", DnsPolicy: "failed", TunnelProbe: "failed"})
	ack, err := l.s.SubmitResult(l.ctx, i, payload(t, late))
	if err != nil || !ack.AcceptedAsHistory {
		t.Fatal(ack, err)
	}
	assertTargetResultEvidence(t, l.get(b.BatchId).Targets[0], &r)
	target := l.get(newBatch.BatchId).Targets[0]
	assertTargetResultEvidence(t, target, &current)
	raw, _ := json.Marshal(target)
	t.Logf("LATE_HISTORY_TARGET=%s", raw)
}

func TestUPD05_TargetResultEvidenceCurrentReceiptSelection(t *testing.T) {
	for _, progress := range []bool{false, true} {
		name := "history_only"
		if progress {
			name = "current_progress_before_history"
		}
		t.Run(name, func(t *testing.T) {
			l, i, b, w, r := at26DirectRollback(t)
			var want *api.UpdateResult
			if progress {
				p := result(w, "applying")
				p.CurrentReleaseId.Set(l.old)
				p.PreviousReleaseId.Set(l.old)
				if _, err := l.s.SubmitResult(l.ctx, i, payload(t, p)); err != nil {
					t.Fatal(err)
				}
				assertTargetResultEvidence(t, l.get(b.BatchId).Targets[0], &p)
				// A second current receipt replaces the first, including its
				// explicit null. Later history must retain this latest evidence.
				p.ResultId = uuid.New()
				p.State = api.UpdateResultStateChecking
				p.PreviousReleaseId.SetNull()
				health := nullableValue(p.Health)
				health.TunnelProbe = "failed"
				p.Health.Set(health)
				if _, err := l.s.SubmitResult(l.ctx, i, payload(t, p)); err != nil {
					t.Fatal(err)
				}
				want = &p
			}
			l.batch(w.Intent.ReleaseId)
			ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r))
			if err != nil || !ack.AcceptedAsHistory {
				t.Fatal(ack, err)
			}
			if progress {
				// An old applying job may finish its ledger as history. Its
				// ledger pointer alone therefore cannot select current evidence.
				if l.scalar(`SELECT count(*) FROM update_jobs WHERE id=$1 AND observed_evidence->>'result_id'=$2`, w.Intent.JobId, r.ResultId.String()) != 1 {
					t.Fatal("expected history to finish the old applying ledger")
				}
			}
			assertTargetResultEvidence(t, l.get(b.BatchId).Targets[0], want)
		})
	}
}

func TestUPD05_TargetResultEvidencePagingRejectionAndRetry(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "disabled", "registered", true)
	l.node("arm64", "unpublished", "enrolling", false)
	b := l.batch(l.new)
	w := l.intent(i)
	bad := result(w, "succeeded")
	bad.Health.SetNull()
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, bad)); err == nil {
		t.Fatal("expected rejected receipt")
	}
	if l.scalar(`SELECT count(*) FROM update_results WHERE result_id=$1 AND ack ? 'error'`, bad.ResultId) != 1 {
		t.Fatal("expected durable rejection")
	}
	for _, target := range l.get(b.BatchId).Targets {
		assertTargetResultEvidence(t, target, nil)
	}
	r := d062Result(t, l, w, "pre-health-failure")
	r.Stage.Set(api.UpdateStageDownload)
	r.ErrorCode.Set(api.ErrorCode(apierr.DownloadTimeout))
	r.PreviousReleaseId.Set(l.old)
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
		t.Fatal(err)
	}
	cursor := ""
	seen := map[uuid.UUID]bool{}
	for {
		page, err := l.s.GetBatch(l.ctx, b.BatchId, cursor, 1)
		if err != nil || len(page.Targets) != 1 || page.TargetCount != 2 {
			t.Fatal(page, err)
		}
		target := page.Targets[0]
		if seen[target.TargetId] {
			t.Fatal("duplicate target page")
		}
		seen[target.TargetId] = true
		var want *api.UpdateResult
		if target.NodeId == i.NodeID {
			want = &r
		}
		assertTargetResultEvidence(t, target, want)
		if page.NextCursor.IsNull() {
			break
		}
		cursor = nullableValue(page.NextCursor)
	}
	if len(seen) != 2 {
		t.Fatal("lost target in cursor page")
	}
	if _, err := l.s.Retry(l.ctx, releasetest.Command(), b.BatchId, api.UpdateBatchRetryRequest{Reason: "TEST ONLY retry"}); err != nil {
		t.Fatal(err)
	}
	for _, target := range l.get(b.BatchId).Targets {
		assertTargetResultEvidence(t, target, nil)
	}
}
