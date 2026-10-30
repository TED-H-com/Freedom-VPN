package updater

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestUPD_D062_PreHealthFailureHasNoHealth(t *testing.T) {
	for _, failure := range []string{"download", "verify", "final_sample"} {
		t.Run(failure, func(t *testing.T) {
			w := newWorld(t)
			old := w.makeRelease("old", 1, []int{1})
			next := w.makeRelease("new", 1, []int{1})
			w.seed(old)
			work := w.work(next, "2", api.PublicationIntentPublished)
			work.Serving = []uuid.UUID{old.id, next.id}
			want := apierr.ArtifactInvalid
			switch failure {
			case "download":
				bad := append([]byte(nil), next.amd...)
				bad[0] ^= 1
				w.files["/a/"+next.id.String()+"/amd64"] = fileSpec{body: bad}
			case "verify":
				work.Intent.ManifestSha256 = strings.Repeat("0", 64)
			case "final_sample":
				w.eng.Meter = nil
				want = apierr.CheckpointFailed
			}
			w.setWork(work)
			_ = w.run()
			w.mustCurrent(old.id)
			res := w.oneResult()
			if res.State != api.UpdateResultStateFailed || !res.Health.IsNull() || res.FinishedAt.IsNull() ||
				requiredValue(t, res.CurrentReleaseId) != old.id || requiredValue(t, res.ErrorCode) != api.ErrorCode(want) {
				t.Fatalf("incorrect pre-health failure: %+v", res)
			}
		})
	}
}

func TestUPD_D062_RollbackReportsRestoredHealthAndKeepsCandidate(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	touchPersistent(t, w.layout)
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		at := goodAttest(ch, api.PublicationIntentPublished)
		if ch.ReleaseID == next.id.String() {
			at.Checks.TunnelProbe.Status = api.HealthCheckStatusFailed
		}
		return at, true
	})
	work := w.work(next, "3", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	res := w.oneResult()
	h := requiredValue(t, res.Health)
	if res.State != api.UpdateResultStateRolledBack || !healthPassed(h) ||
		requiredValue(t, res.ErrorCode) != api.ErrorCode(apierr.HealthCheckFailed) ||
		requiredValue(t, res.CurrentReleaseId) != old.id || requiredValue(t, res.PreviousReleaseId) != next.id {
		t.Fatalf("rollback evidence: %+v", res)
	}
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := decodeHealth(txn.CandidateHealth)
	if err != nil || candidate == nil || candidate.TunnelProbe != api.HealthCheckStatusFailed {
		t.Fatalf("candidate health lost: %s %v", txn.CandidateHealth, err)
	}
	snap, err := updatejournal.Snapshot(context.Background(), filepath.Join(w.layout.Journal, updatejournal.FileName))
	if err != nil || string(snap.CandidateHealth) != string(txn.CandidateHealth) || snap.State != "rolled_back" {
		t.Fatalf("snapshot: %+v %v", snap, err)
	}
	raw, err := json.Marshal(res)
	if err != nil || strings.Contains(string(raw), "candidate_health") {
		t.Fatal("candidate-only field leaked onto the wire", err)
	}
}

func TestUPD_D034e_EmptyRollbackAllowlistAddsNoRestriction(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	if len(next.manifest.Rollback.AllowedReleaseIds) != 0 {
		t.Fatal("test requires an empty manifest allowlist")
	}
	c := w.eng.claimFrom(w.work(next, "4", api.PublicationIntentPublished))
	c.Serving = []uuid.UUID{old.id}
	if got := DecideRollback(w.eng.evalLastGood(context.Background(), old.id.String(), c, &next.manifest)); !got.Restore {
		t.Fatalf("empty allowlist restricted serving release: %+v", got)
	}
	c.Serving = nil
	if got := DecideRollback(w.eng.evalLastGood(context.Background(), old.id.String(), c, &next.manifest)); got.Restore {
		t.Fatal("empty allowlist bypassed the serving set")
	}
}

func TestUPD_D062_RejectedResultDoesNotBlockRecoveredAndRollbackResults(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("next", 1, []int{1})
	bad := w.makeRelease("bad", 1, []int{1})
	w.seed(old)
	work := w.work(next, "2", api.PublicationIntentPublished)
	work.Intent.ManifestSha256 = strings.Repeat("0", 64)
	w.setWork(work)
	_ = w.run()
	// A genuine restart of an unfinished transaction produces the next result.
	work = w.work(next, "3", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	w.eng.Hooks.AfterStage = func(s string) error {
		if s == string(updatejournal.StageChecking) {
			return ErrCrashed
		}
		return nil
	}
	if err := w.run(); !errors.Is(err, ErrCrashed) {
		t.Fatal(err)
	}
	w.eng.Hooks.AfterStage = nil
	if err := w.eng.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	w.health.fail[bad.id.String()] = true
	work = w.work(bad, "4", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{next.id, bad.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	var received []api.UpdateResultState
	srv := newTestServer(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var res api.UpdateResult
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			t.Error(err)
			rw.WriteHeader(400)
			return
		}
		received = append(received, res.State)
		if len(received) == 1 {
			rw.WriteHeader(422)
			_, _ = rw.Write([]byte(`{"error":{"code":"VALIDATION_FAILED","message":"private diagnostic must not be stored"}}`))
			return
		}
		rw.WriteHeader(202)
		_ = json.NewEncoder(rw).Encode(api.UpdateResultAck{Accepted: true, ResultId: res.ResultId})
	}))
	w.eng.Sender = HTTPSink{URL: srv.URL, Client: srv.Client(), AllowLoopback: true}
	if err := w.eng.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []api.UpdateResultState{api.UpdateResultStateFailed, api.UpdateResultStateSucceeded, api.UpdateResultStateRolledBack}
	if !reflect.DeepEqual(received, want) {
		t.Fatalf("delivery order=%v, want %v", received, want)
	}
	left, err := w.eng.Journal.Outbox(context.Background())
	if err != nil || len(left) != 1 || left[0].Status != updatejournal.OutboxRejected || left[0].LastHTTPStatus != 422 || left[0].LastAPICode != "VALIDATION_FAILED" {
		t.Fatalf("rejection diagnostics=%+v err=%v", left, err)
	}
	if strings.Contains(left[0].LastError, "private") || left[0].LastError == "STORE_UNAVAILABLE" {
		t.Fatal("rejection was hidden or leaked response text")
	}
	if err := w.eng.Flush(context.Background()); err != nil || len(received) != 3 {
		t.Fatal("rejected result retried", err)
	}
	history, err := w.eng.Journal.History(context.Background(), "")
	if err != nil || len(history) != 2 {
		t.Fatalf("history=%+v %v", history, err)
	}
}

func TestUPD_D062_RetryAfterSurvivesRestart(t *testing.T) {
	for _, header := range []string{"10", fixedNow().Add(10 * time.Second).Format(http.TimeFormat)} {
		t.Run(header, func(t *testing.T) {
			w := newWorld(t)
			next := w.makeRelease("new", 1, []int{1})
			work := w.work(next, "1", api.PublicationIntentUnpublished)
			work.Intent.ManifestSha256 = strings.Repeat("0", 64)
			w.setWork(work)
			_ = w.run()
			hits := 0
			srv := newTestServer(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				hits++
				rw.Header().Set("Retry-After", header)
				rw.WriteHeader(429)
				_, _ = rw.Write([]byte(`{"error":{"code":"RATE_LIMITED"}}`))
			}))
			w.eng.Sender = HTTPSink{URL: srv.URL, Client: srv.Client(), AllowLoopback: true, Now: w.eng.now}
			if err := w.eng.Flush(context.Background()); asOp(err) == nil || asOp(err).Code != apierr.RateLimited {
				t.Fatal(err)
			}
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			w.eng.Journal = nil
			w.now = w.now.Add(9 * time.Second)
			_ = w.eng.Flush(context.Background())
			if hits != 1 {
				t.Fatal("retried before Retry-After", hits)
			}
			w.now = w.now.Add(time.Second)
			_ = w.eng.Flush(context.Background())
			if hits != 2 {
				t.Fatal("did not retry at Retry-After", hits)
			}
		})
	}
}

func TestUPD_D062_HTTPSinkClassifiesResponses(t *testing.T) {
	for _, status := range []int{200, 400, 401, 403, 409, 422, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"code":"VALIDATION_FAILED","message":"https://secret.example.com"}}`))
			}))
			sink := HTTPSink{URL: srv.URL, Client: srv.Client(), AllowLoopback: true}
			ack, conflict, err := sink.Submit(context.Background(), []byte(`{}`))
			if status == 409 {
				if err != nil || !conflict || ack.HTTPStatus != status || ack.APICode != "VALIDATION_FAILED" {
					t.Fatalf("conflict evidence: %+v %v", ack, err)
				}
				return
			}
			var delivery *DeliveryError
			if !errors.As(err, &delivery) || delivery.HTTPStatus != status || delivery.APICode != "VALIDATION_FAILED" {
				t.Fatal("lost HTTP/API evidence", err)
			}
			if strings.Contains(err.Error(), "secret") || (asOp(err).Code == apierr.StoreUnavailable) != (status >= 500) {
				t.Fatal("unsafe or incorrect error", err)
			}
			if delivery.rejected() != (status >= 400 && status < 500 && status != 429) {
				t.Fatal("incorrect rejection class", err)
			}
		})
	}
}

func TestUPD_D062_PostCheckFailureReportsObservedHealth(t *testing.T) {
	w := newWorld(t)
	next := w.makeRelease("new", 1, []int{1})
	w.health.fail[next.id.String()] = true
	w.setWork(w.work(next, "5", api.PublicationIntentUnpublished))
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	res := w.oneResult()
	if res.Health.IsNull() || healthPassed(requiredValue(t, res.Health)) || !requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("lost observed failure: %+v", res)
	}
}

func TestUPD_D062_RecoveryReusesPreviouslySealedResult(t *testing.T) {
	for _, acked := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "acknowledged"}[acked], func(t *testing.T) {
			w := newWorld(t)
			old := w.makeRelease("old", 1, []int{1})
			next := w.makeRelease("new", 1, []int{1})
			w.seed(old)
			w.health.fail[next.id.String()] = true
			work := w.work(next, "6", api.PublicationIntentPublished)
			work.Serving = []uuid.UUID{old.id, next.id}
			w.setWork(work)
			w.eng.Hooks.AfterStage = func(s string) error {
				if s == string(updatejournal.StageRolledBack) {
					return ErrCrashed
				}
				return nil
			}
			if err := w.run(); !errors.Is(err, ErrCrashed) {
				t.Fatal(err)
			}
			w.eng.Hooks.AfterStage = nil
			txn, err := w.eng.Journal.LastTxn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			// Reproduce the old wire policy: candidate health and original
			// previous pointer. The new journal contains corrected evidence.
			health, err := decodeHealth(txn.CandidateHealth)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.eng.enqueueResult(context.Background(), resultInput{
				Claimed: w.eng.claimFrom(work), State: api.UpdateResultStateRolledBack, Stage: api.UpdateStageReport,
				Code: apierr.HealthCheckFailed, Current: old.id.String(), Previous: old.id.String(),
				Health: health, Schema: 1, Started: txn.StartedAt, Finished: txn.FinishedAt,
			}); err != nil {
				t.Fatal(err)
			}
			entries, err := w.eng.Journal.Outbox(context.Background())
			if err != nil || len(entries) != 1 {
				t.Fatal(entries, err)
			}
			before := entries[0]
			if acked {
				if _, err := w.eng.Journal.AckResult(context.Background(), updatejournal.Ack{
					ResultID: before.ResultID, PayloadHash: before.PayloadHash,
				}); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.eng.Recover(context.Background()); err != nil {
				t.Fatal(err)
			}
			job, err := w.eng.Journal.GetJob(context.Background(), txn.JobID, txn.Attempt)
			if err != nil || job.Phase != updatejournal.JobDone {
				t.Fatalf("replay did not finish: %+v %v", job, err)
			}
			entries, err = w.eng.Journal.Outbox(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if acked {
				if len(entries) != 0 {
					t.Fatal("acknowledged result recreated", entries)
				}
			} else if len(entries) != 1 || string(entries[0].Payload) != string(before.Payload) || entries[0].PayloadHash != before.PayloadHash || entries[0].Status != updatejournal.OutboxPending {
				t.Fatal("previously sealed evidence rewritten or conflicted", entries)
			}
		})
	}
}
