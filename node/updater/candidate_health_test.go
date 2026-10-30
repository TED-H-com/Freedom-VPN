package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func TestUPD_D062_CandidateHealthFlush(t *testing.T) {
	for _, precheck := range []bool{false, true} {
		t.Run(map[bool]string{false: "checked_candidate", true: "before_candidate_checks"}[precheck], func(t *testing.T) {
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
			if precheck {
				w.eng.Services = candidateStartFailure{ServiceManager: w.eng.Services, dir: w.layout.releaseDir(next.id.String())}
			}
			work := w.work(next, "8", api.PublicationIntentPublished)
			work.Serving = []uuid.UUID{old.id, next.id}
			w.setWork(work)
			if err := w.run(); err != nil {
				t.Fatal(err)
			}
			entries, err := w.eng.Journal.Outbox(context.Background())
			if err != nil || len(entries) != 1 {
				t.Fatal(entries, err)
			}
			txn, err := w.eng.Journal.LastTxn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			wantCode := api.ErrorCode(apierr.HealthCheckFailed)
			wantCandidate := txn.CandidateHealth
			if precheck {
				wantCode = api.ErrorCode(apierr.StartFailed)
				if len(wantCandidate) != 0 {
					t.Fatal("candidate checks unexpectedly ran", string(wantCandidate))
				}
				wantCandidate = json.RawMessage("null")
			}
			// Compare the actual POST bytes to the sealed canonical JSON and
			// compare both profiles to the distinct recorded challenge answers.
			calls := 0
			srv := newTestServer(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				calls++
				raw, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(raw, entries[0].Payload) {
					t.Error("flush changed sealed JSON", err)
				}
				fields := jsonFields(t, raw)
				if !bytes.Equal(canonical(t, fields["candidate_health"]), canonical(t, wantCandidate)) ||
					!bytes.Equal(canonical(t, fields["health"]), canonical(t, txn.Health)) {
					t.Error("incorrect candidate/restored JSON", string(raw))
				}
				var res api.UpdateResult
				if err := json.Unmarshal(raw, &res); err != nil {
					t.Error(err)
					rw.WriteHeader(422)
					return
				}
				if res.State != api.UpdateResultStateRolledBack || !healthPassed(requiredValue(t, res.Health)) ||
					requiredValue(t, res.CurrentReleaseId) != old.id || requiredValue(t, res.PreviousReleaseId) != next.id ||
					requiredValue(t, res.ErrorCode) != wantCode || res.CandidateHealth.IsNull() != precheck {
					t.Error("incorrect rollback evidence", res)
				}
				assertResultHash(t, raw)
				rw.WriteHeader(202)
				_ = json.NewEncoder(rw).Encode(api.UpdateResultAck{Accepted: true, ResultId: res.ResultId})
			}))
			w.eng.Sender = HTTPSink{URL: srv.URL, Client: srv.Client(), AllowLoopback: true}
			if err := w.eng.Flush(context.Background()); err != nil || calls != 1 {
				t.Fatal("flush", calls, err)
			}
		})
	}
}

type candidateStartFailure struct {
	ServiceManager
	dir string
}

func (s candidateStartFailure) Start(ctx context.Context, dir string) error {
	if dir == s.dir {
		return errors.New("test candidate start failure")
	}
	return s.ServiceManager.Start(ctx, dir)
}

func TestUPD_D062_CandidateHealthOmittedOutsideRollback(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "succeeded", true: "failed"}[failed], func(t *testing.T) {
			w := newWorld(t)
			next := w.makeRelease("new", 1, []int{1})
			w.health.fail[next.id.String()] = failed
			w.setWork(w.work(next, "8", api.PublicationIntentUnpublished))
			if err := w.run(); err != nil && !errors.Is(err, ErrMembersBlocked) {
				t.Fatal(err)
			}
			res := w.oneResult()
			want := api.UpdateResultStateSucceeded
			if failed {
				want = api.UpdateResultStateFailed
			}
			if res.State != want || res.CandidateHealth.IsSpecified() {
				t.Fatalf("candidate_health on %s: %+v", want, res)
			}
			raw, err := json.Marshal(res)
			if err != nil || jsonFields(t, raw)["candidate_health"] != nil {
				t.Fatal("non-rollback member was not omitted", err)
			}
		})
	}
}

func TestUPD_D062_CandidateHealthContractVectors(t *testing.T) {
	for _, name := range []string{"UpdateResult.rolled-back.json", "UpdateResult.null-candidate-health.json", "UpdateResult.preflight-failed.json", "UpdateResult.json"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "valid", name))
			if err != nil {
				t.Fatal(err)
			}
			var vector api.UpdateResult
			if err := json.Unmarshal(raw, &vector); err != nil {
				t.Fatal(err)
			}
			roundTrip, err := json.Marshal(vector)
			if err != nil || !bytes.Equal(canonical(t, raw), canonical(t, roundTrip)) {
				t.Fatal("pkg/api changed vector member presence or shape", err)
			}
			code, _ := vector.ErrorCode.Get()
			in := resultInput{
				Claimed: claimed{BootID: vector.BootId, Work: Work{PolicyGeneration: vector.PolicyGeneration,
					Intent: api.UpdateIntent{Attempt: vector.Attempt, BatchId: vector.BatchId, TargetId: vector.TargetId,
						Generation: vector.Generation, IncarnationId: vector.IncarnationId, JobId: vector.JobId,
						ManifestSha256: vector.ManifestSha256, NodeId: vector.NodeId, PublicationIntent: vector.PublicationIntent,
						ReleaseId: vector.RequestedReleaseId}}},
				State: vector.State, Stage: requiredValue(t, vector.Stage), Code: apierr.Code(code),
				Current: requiredValue(t, vector.CurrentReleaseId).String(), Previous: requiredValue(t, vector.PreviousReleaseId).String(),
				Recovery: requiredValue(t, vector.RecoveryRequired), Forwarding: vector.MemberForwardingEnabled,
				Started: vector.StartedAt, Finished: requiredValue(t, vector.FinishedAt), Schema: vector.StateSchemaVersion,
				Health: nullableHealth(vector.Health), CandidateHealth: nullableHealth(vector.CandidateHealth),
			}
			cp := requiredValue(t, vector.UsageCheckpoint)
			in.Checkpoint = &cp
			got, _, id, err := buildResult(in)
			if err != nil {
				t.Fatal(err)
			}
			// Fixture IDs are examples; the updater uses its stable UUIDv5 ID.
			// The contract explicitly permits omission in place of non-rollback
			// null. All other members and their complete shapes must match.
			vector.ResultId = id
			if vector.State != api.UpdateResultStateRolledBack {
				if vector.CandidateHealth.IsSpecified() && !vector.CandidateHealth.IsNull() {
					t.Fatal("non-rollback vector contains candidate checks")
				}
				vector.CandidateHealth.SetUnspecified()
			}
			expected, err := json.Marshal(vector)
			if err != nil {
				t.Fatal(err)
			}
			vector.PayloadHash, err = canonjson.HashHex(expected, "payload_hash")
			if err != nil {
				t.Fatal(err)
			}
			expected, err = json.Marshal(vector)
			if err != nil || !bytes.Equal(got, canonical(t, expected)) {
				t.Fatalf("emitted shape differs from vector: %s\nwant: %s\n%v", got, expected, err)
			}
			assertResultHash(t, got)
		})
	}
}

func nullableHealth(n presence.Nullable[api.UpdateHealth]) *api.UpdateHealth {
	h, err := n.Get()
	if err != nil {
		return nil
	}
	return &h
}

func canonical(t *testing.T, raw []byte) []byte {
	t.Helper()
	b, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func jsonFields(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func assertResultHash(t *testing.T, raw []byte) {
	t.Helper()
	hash, err := canonjson.HashHex(raw, "payload_hash")
	if err != nil || string(jsonFields(t, raw)["payload_hash"]) != `"`+hash+`"` {
		t.Fatal("result payload hash does not cover candidate_health", err)
	}
}

func TestUPD_D062_QueuedV2RollbackAddsCandidateHealthBeforeRetry(t *testing.T) {
	for _, checked := range []bool{false, true} {
		t.Run(map[bool]string{false: "no_candidate_check", true: "stored_candidate"}[checked], func(t *testing.T) {
			w := newWorld(t)
			ctx := context.Background()
			raw, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "valid", "UpdateResult.rolled-back.json"))
			if err != nil {
				t.Fatal(err)
			}
			var res api.UpdateResult
			if err := json.Unmarshal(raw, &res); err != nil {
				t.Fatal(err)
			}
			j := w.eng.Journal
			txnID := uuid.Must(uuid.NewV7()).String()
			if _, err := j.BeginTxn(ctx, updatejournal.Txn{TxnID: txnID, JobID: res.JobId.String(),
				Attempt: res.Attempt, Generation: res.Generation, ReleaseID: res.RequestedReleaseId.String(),
				ManifestSHA256: res.ManifestSha256}); err != nil {
				t.Fatal(err)
			}
			if checked {
				for _, stage := range []updatejournal.Stage{updatejournal.StageBlocked, updatejournal.StageFinalSampled,
					updatejournal.StageStopped, updatejournal.StageMigrated, updatejournal.StageSwitched, updatejournal.StageChecking} {
					if _, err := j.Advance(ctx, txnID, stage, updatejournal.StageUpdate{}); err != nil {
						t.Fatal(err)
					}
				}
				if err := j.RecordHealth(ctx, txnID, requiredValue(t, res.CandidateHealth)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := j.Advance(ctx, txnID, updatejournal.StageRollingBack, updatejournal.StageUpdate{Destructive: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := j.Advance(ctx, txnID, updatejournal.StageRolledBack, updatejournal.StageUpdate{Health: requiredValue(t, res.Health)}); err != nil {
				t.Fatal(err)
			}
			fields := jsonFields(t, raw)
			wantCandidate := fields["candidate_health"]
			if !checked {
				wantCandidate = json.RawMessage("null")
			}
			delete(fields, "candidate_health")
			legacy, legacyHash := hashResultFields(t, fields)
			if err := j.Enqueue(ctx, updatejournal.OutboxEntry{ResultID: res.ResultId.String(), JobID: res.JobId.String(),
				Attempt: res.Attempt, Generation: res.Generation, State: string(res.State), Payload: legacy, PayloadHash: legacyHash}); err != nil {
				t.Fatal(err)
			}
			// It was already sent by the previous binary, but is still pending.
			if err := j.RecordDeliveryFailure(ctx, res.ResultId.String(), updatejournal.DeliveryFailure{
				Status: updatejournal.OutboxPending, HTTPStatus: 503, ErrorCode: "STORE_UNAVAILABLE",
			}); err != nil {
				t.Fatal(err)
			}
			fields["candidate_health"] = wantCandidate
			want, wantHash := hashResultFields(t, fields)
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			w.eng.Journal = nil
			calls := 0
			srv := newTestServer(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				calls++
				body, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(body, want) || r.Header.Get("Idempotency-Key") != res.ResultId.String() {
					t.Error("queued result regenerated or wrong exact JSON", string(body), err)
				}
				assertResultHash(t, body)
				// The payload must be committed before the first HTTP attempt.
				out, err := w.eng.Journal.Outbox(ctx)
				if err != nil || len(out) != 1 || !bytes.Equal(out[0].Payload, body) || out[0].PayloadHash != wantHash {
					t.Error("uncommitted wire payload", out, err)
				}
				if calls == 1 {
					rw.WriteHeader(503)
					return
				}
				rw.WriteHeader(202)
				_ = json.NewEncoder(rw).Encode(api.UpdateResultAck{Accepted: true, ResultId: res.ResultId})
			}))
			w.eng.Sender = HTTPSink{URL: srv.URL, Client: srv.Client(), AllowLoopback: true}
			if err := w.eng.Flush(ctx); asOp(err) == nil || asOp(err).Code != apierr.StoreUnavailable {
				t.Fatal("expected retryable delivery", err)
			}
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			w.eng.Journal = nil
			if err := w.eng.Flush(ctx); err != nil || calls != 2 {
				t.Fatal("retry after restart", calls, err)
			}
			history, err := w.eng.Journal.History(ctx, res.JobId.String())
			if err != nil || len(history) != 1 || history[0].PayloadHash != wantHash || history[0].ResultID != res.ResultId.String() {
				t.Fatal("ACK did not match transmitted payload", history, err)
			}
		})
	}
}

func hashResultFields(t *testing.T, fields map[string]json.RawMessage) ([]byte, string) {
	t.Helper()
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		t.Fatal(err)
	}
	fields["payload_hash"] = json.RawMessage(`"` + hash + `"`)
	raw, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return canonical(t, raw), hash
}
