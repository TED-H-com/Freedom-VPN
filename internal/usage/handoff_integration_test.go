//go:build integration

package usage_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type handoffVector struct {
	Context struct {
		RetiredAt  time.Time  `json:"retired_at"`
		EnrolledAt time.Time  `json:"presenting_enrolled_at"`
		Interval   int        `json:"sampling_interval_seconds"`
		Prefix     decstr.U64 `json:"previous_highest_contiguous_seq"`
	} `json:"context"`
	Cases []struct {
		Name     string                `json:"name"`
		Request  api.UsageBatchRequest `json:"request"`
		Response api.UsageBatchResult  `json:"response"`
		Status   int                   `json:"http_status"`
		Audited  bool                  `json:"audited"`
	} `json:"cases"`
}

func readHandoffJSON(t *testing.T, path string, out any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "contracts", path))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatal(err)
	}
}

func loadHandoffVector(t *testing.T) handoffVector {
	t.Helper()
	var v handoffVector
	readHandoffJSON(t, "vectors/usage-retired-handoff.json", &v)
	return v
}

type handoffFixture struct {
	pool *pgxpool.Pool
	svc  *usage.Service
	v    handoffVector
	id   usage.NodeIdentity
}

func newHandoffFixture(t *testing.T, bound time.Duration, enrolledAt time.Time) *handoffFixture {
	t.Helper()
	v := loadHandoffVector(t)
	pool := dbtest.New(t)
	uFix := usagetest.New(t)
	signer := usage.NewProviderSigner(uFix.Provider)
	svc, err := usage.NewService(usage.Config{
		Pool: pool, Clock: func() time.Time { return v.Cases[0].Response.ReceivedAt },
		Signer: signer, LedgerEpochSigner: signer, SamplingIntervalBound: bound,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := svc.GetLedgerState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// Only the random central genesis epoch differs from the contract fixture.
	for i := range v.Cases {
		v.Cases[i].Request.LedgerEpoch = presence.NewNullableWithValue(state.LedgerEpoch)
		v.Cases[i].Response.LedgerEpoch = state.LedgerEpoch
	}
	req := v.Cases[0].Request
	if enrolledAt.IsZero() {
		enrolledAt = v.Context.EnrolledAt
	}
	mustExec(t, pool, `INSERT INTO nodes(id,lifecycle_state) VALUES($1,'registered')`, req.NodeId)
	mustExec(t, pool, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch,created_at,revoked_at)
		VALUES($1,$2,decode(repeat('11',32),'hex'),'ubuntu','amd64',$3,$4)`,
		*req.OriginIncarnationId, req.NodeId, v.Context.RetiredAt.Add(-time.Hour), v.Context.RetiredAt)
	mustExec(t, pool, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch,created_at)
		VALUES($1,$2,decode(repeat('22',32),'hex'),'ubuntu','arm64',$3)`, req.NodeIncarnationId, req.NodeId, enrolledAt)
	mustExec(t, pool, `UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, req.NodeId, req.NodeIncarnationId)
	for i, rec := range req.Events[0].Records {
		mustExec(t, pool, `INSERT INTO members(id,status) VALUES($1,'active')`, rec.MemberId)
		mustExec(t, pool, `INSERT INTO entitlements(member_id,revision,device_limit,valid_from,effective_service_end)
			VALUES($1,1,5,'2026-01-01T00:00:00Z','2027-01-01T00:00:00Z')`, rec.MemberId)
		mustExec(t, pool, `INSERT INTO member_devices(id,member_id,deployment_id,peer_id,status)
			VALUES($1,$2,'handoff-test',$3,'authorized')`, rec.DeviceId, rec.MemberId, fmt.Sprintf("peer-%d", i))
		mustExec(t, pool, `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref)
			VALUES($1,$2,'handoff-test',$3,$4,'2026-01-01T00:00:00Z','2027-01-01T00:00:00Z',$5)`,
			rec.DeviceId, rec.MemberId, fmt.Sprintf("100.64.0.%d", i+10), rec.BindingRevision.String(), fmt.Sprintf("peer-%d", i))
	}
	// TEST ONLY: prior durable empty-event receipts establish the vector's 1233
	// contiguous prefix, without disabling the receipt/checkpoint guards.
	mustExec(t, pool, `INSERT INTO usage_events(node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count,ledger_epoch)
		SELECT $1,$2,seq,1,decode(repeat('33',32),'hex'),0,$3 FROM generate_series(1,$4::integer) seq`,
		req.NodeId, *req.OriginIncarnationId, state.LedgerEpoch, int(v.Context.Prefix))
	return &handoffFixture{pool: pool, svc: svc, v: v, id: usage.NodeIdentity{NodeID: req.NodeId, NodeIncarnationID: req.NodeIncarnationId}}
}

func assertHandoffJSON(t *testing.T, got, want any) {
	t.Helper()
	g, err := canonjson.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	w, err := canonjson.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(g, w) {
		t.Fatalf("JSON mismatch:\ngot  %s\nwant %s", g, w)
	}
}

func handoffCount(t *testing.T, f *handoffFixture, query string, args ...any) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(t.Context(), query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func assertHandoffAudit(t *testing.T, f *handoffFixture, code apierr.Code, reason string, seq *decstr.U64) {
	t.Helper()
	var actor, actorType, resource, result string
	var after []byte
	if err := f.pool.QueryRow(t.Context(), `SELECT actor,actor_type,resource,result,after_diff FROM audit_events
		WHERE action='usage.ingest_rejected' ORDER BY occurred_at DESC LIMIT 1`).Scan(&actor, &actorType, &resource, &result, &after); err != nil {
		t.Fatal(err)
	}
	if actor != f.id.NodeID.String() || actorType != "node" || resource != "node:"+actor || result != "failed" {
		t.Fatalf("invalid audit identity/result: %s %s %s %s", actor, actorType, resource, result)
	}
	var metadata map[string]any
	if err := json.Unmarshal(after, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["error_code"] != string(code) || metadata["reason"] != reason || metadata["presenting_incarnation_id"] != f.id.NodeIncarnationID.String() {
		t.Fatalf("invalid audit metadata: %s", after)
	}
	if seq != nil && metadata["event_seq"] != seq.String() {
		t.Fatalf("audit sequence is not the original decimal string: %s", after)
	}
	for key := range metadata {
		switch key {
		case "error_code", "reason", "event_seq", "presenting_incarnation_id", "origin_incarnation_id":
		default:
			t.Fatalf("unexpected audit field %s", key)
		}
	}
}

func TestMTR03_D105ContractVectorsAndExamples(t *testing.T) {
	v := loadHandoffVector(t)
	for i, c := range v.Cases {
		t.Run(c.Name, func(t *testing.T) {
			f := newHandoffFixture(t, time.Duration(v.Context.Interval)*time.Second, time.Time{})
			c := f.v.Cases[i]
			var exampleRequest api.UsageBatchRequest
			requestPath := "examples/valid/UsageBatchRequest.handoff-" + c.Name + ".json"
			if c.Name == "after-retirement-window" {
				requestPath = "examples/semantic-invalid/UsageBatchRequest.retired-window.json"
			}
			readHandoffJSON(t, requestPath, &exampleRequest)
			exampleRequest.LedgerEpoch = c.Request.LedgerEpoch
			assertHandoffJSON(t, exampleRequest, c.Request)
			var exampleResult api.UsageBatchResult
			readHandoffJSON(t, "examples/valid/UsageBatchResult.handoff-"+c.Name+".json", &exampleResult)
			exampleResult.LedgerEpoch = c.Response.LedgerEpoch
			assertHandoffJSON(t, exampleResult, c.Response)
			if c.Name == "idempotent-replay" {
				if _, err := f.svc.IngestBatch(t.Context(), f.id, f.v.Cases[0].Request); err != nil {
					t.Fatal(err)
				}
			}
			got, err := f.svc.IngestBatch(t.Context(), f.id, c.Request)
			if err != nil {
				t.Fatal(err)
			}
			assertHandoffJSON(t, got.UsageBatchResult, c.Response)
			if c.Audited {
				assertHandoffAudit(t, f, apierr.ValidationFailed, "retired_window", &c.Request.Events[0].EventSeq)
				if n := handoffCount(t, f, `SELECT count(*) FROM usage_event_rejections WHERE incarnation_id=$1 AND event_seq=$2`, *c.Request.OriginIncarnationId, c.Request.Events[0].EventSeq.String()); n != 1 {
					t.Fatalf("rejection rows=%d", n)
				}
			} else {
				if n := handoffCount(t, f, `SELECT count(*) FROM audit_events WHERE action='usage.ingest_rejected'`); n != 0 {
					t.Fatalf("unexpected rejection audit rows=%d", n)
				}
				for _, rec := range c.Request.Events[0].Records {
					if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id
						WHERE e.incarnation_id=$1 AND e.event_seq=$2 AND encode(e.payload_hash,'hex')=$3
						AND s.member_id=$4 AND s.device_id=$5 AND s.binding_revision=$6 AND s.upload_bytes=$7 AND s.download_bytes=$8`,
						*c.Request.OriginIncarnationId, c.Request.Events[0].EventSeq.String(), c.Request.Events[0].PayloadSha256,
						rec.MemberId, rec.DeviceId, rec.BindingRevision.String(), rec.UploadBytes.String(), rec.DownloadBytes.String()); n != 1 {
						t.Fatalf("immutable original sample count=%d", n)
					}
				}
			}
			if n := handoffCount(t, f, `SELECT count(*) FROM usage_checkpoints WHERE incarnation_id=$1`, f.id.NodeIncarnationID); n != 0 {
				t.Fatalf("hand-off touched current checkpoint: rows=%d", n)
			}
		})
	}
}

func TestMTR03_D112WindowBoundaryAndDefault(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bound time.Duration
		end   time.Duration
		ok    bool
	}{
		{"default-inclusive", 0, 10 * time.Second, true},
		{"default-no-skew", 0, 10*time.Second + time.Nanosecond, false},
		{"override-inclusive", 30 * time.Second, 30 * time.Second, true},
		{"override-overrun", 30 * time.Second, 30*time.Second + time.Nanosecond, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newHandoffFixture(t, tc.bound, time.Time{})
			req := f.v.Cases[0].Request
			req.Events[0].IntervalEnd = f.v.Context.RetiredAt.Add(tc.end)
			var err error
			req.Events[0], err = usage.WithComputedHash(req.Events[0])
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.svc.IngestBatch(t.Context(), f.id, req)
			if err != nil {
				t.Fatal(err)
			}
			want := api.UsageEventResultStatusRejected
			if tc.ok {
				want = api.UsageEventResultStatusAccepted
			}
			if got.Results[0].Status != want {
				t.Fatalf("window result=%+v", got)
			}
		})
	}
}

func TestMTR03_D105IneligibleAndMixedStreams(t *testing.T) {
	v := loadHandoffVector(t)
	for _, name := range []string{"foreign-node", "unknown-origin", "active-origin", "enrolled-before", "enrolled-at-retirement", "mixed-streams", "event-node", "batch-identity", "noncurrent-presenter", "superseded-presenter"} {
		t.Run(name, func(t *testing.T) {
			enrolled := v.Context.EnrolledAt
			if name == "enrolled-before" {
				enrolled = v.Context.RetiredAt.Add(-time.Nanosecond * 1000)
			} else if name == "enrolled-at-retirement" {
				enrolled = v.Context.RetiredAt
			}
			f := newHandoffFixture(t, 0, enrolled)
			req := f.v.Cases[0].Request
			code, reason := apierr.StateConflict, "origin_ineligible"
			switch name {
			case "foreign-node":
				node, origin := uuid.New(), uuid.New()
				mustExec(t, f.pool, `INSERT INTO nodes(id) VALUES($1)`, node)
				mustExec(t, f.pool, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch,revoked_at)
					VALUES($1,$2,decode(repeat('44',32),'hex'),'ubuntu','amd64',$3)`, origin, node, v.Context.RetiredAt)
				req.OriginIncarnationId = &origin
				req.Events[0].NodeIncarnationId = origin
			case "unknown-origin":
				origin := uuid.New()
				req.OriginIncarnationId = &origin
				req.Events[0].NodeIncarnationId = origin
			case "active-origin":
				req.OriginIncarnationId = &req.NodeIncarnationId
				req.Events[0].NodeIncarnationId = req.NodeIncarnationId
			case "mixed-streams":
				ev := req.Events[0]
				ev.NodeIncarnationId = f.id.NodeIncarnationID
				req.Events = append(req.Events, ev)
				code, reason = apierr.IncarnationMismatch, "event_identity"
			case "event-node":
				req.Events[0].NodeId = uuid.New()
				code, reason = apierr.IncarnationMismatch, "event_identity"
			case "batch-identity":
				req.NodeId = uuid.New()
				code, reason = apierr.IncarnationMismatch, "batch_identity"
			case "noncurrent-presenter":
				mustExec(t, f.pool, `UPDATE nodes SET current_incarnation_id=NULL,resource_version=resource_version+1 WHERE id=$1`, f.id.NodeID)
				code, reason = apierr.IncarnationMismatch, "presenter_inactive"
			case "superseded-presenter":
				// Retire the presenter; its old body cannot authorize hand-off.
				mustExec(t, f.pool, `UPDATE node_incarnations SET revoked_at=$2 WHERE id=$1`, f.id.NodeIncarnationID, v.Context.EnrolledAt.Add(time.Minute))
				code, reason = apierr.IdentityRevoked, "presenter_inactive"
			}
			_, err := f.svc.IngestBatch(t.Context(), f.id, req)
			var ae *apierr.Error
			if !errors.As(err, &ae) || ae.Code != code {
				t.Fatalf("got %v, want %s", err, code)
			}
			assertHandoffAudit(t, f, code, reason, nil)
			if n := handoffCount(t, f, `SELECT count(*) FROM usage_events`); n != int(v.Context.Prefix) {
				t.Fatalf("whole-batch rejection changed ledger: %d events", n)
			}
			if n := handoffCount(t, f, `SELECT count(*) FROM usage_checkpoints`); n != 0 {
				t.Fatalf("whole-batch rejection created checkpoint: %d", n)
			}
		})
	}
}

// newHandoffHTTP exercises the existing durable replay wiring, using a real PKI
// authenticator when provided. The handler uses D-112's default 10-second bound.
func newHandoffHTTP(t *testing.T, f *handoffFixture, agents auth.AgentAuthenticator) http.Handler {
	t.Helper()
	u, p := usagetest.New(t), policiestest.New(t)
	h, err := control.NewUsageHandler(f.pool, nil, u.Usage, p.Policies, nil)
	if err != nil {
		t.Fatal(err)
	}
	composite, err := httpapi.Compose(h)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	if agents == nil {
		agents = auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
			return auth.Principal{Role: auth.Agent, NodeID: f.id.NodeID, IncarnationID: f.id.NodeIncarnationID}, nil
		})
	}
	handler, err := httpapi.New(composite, httpapi.Options{Config: c, AgentListener: true, Agents: agents, Commands: &httpapi.Commands{Pool: f.pool, Retention: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	return httpapi.Transport(nil, handler)
}

func postHandoff(t *testing.T, h http.Handler, req api.UsageBatchRequest, key string, cert *x509.Certificate) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", key)
	if cert == nil {
		cert = &x509.Certificate{}
	}
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestMTR03_D105HTTPSameKeyStoredReplayAndStreamACK(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	req := f.v.Cases[0].Request
	key := uuid.NewString()
	first := postHandoff(t, newHandoffHTTP(t, f, nil), req, key, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	// A new handler instance must replay the stored original response, even if
	// central is now frozen. It does not execute ingestion or mint another ACK.
	mustExec(t, f.pool, `UPDATE usage_ledger_state SET mode='frozen'`)
	replay := postHandoff(t, newHandoffHTTP(t, f, nil), req, key, nil)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body.String())
	}
	var got, stored api.UsageBatchResult
	if err := json.Unmarshal(first.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &stored); err != nil {
		t.Fatal(err)
	}
	assertHandoffJSON(t, stored, got)
	if got.NodeIncarnationId != *req.OriginIncarnationId || got.Results[0].Status != api.UsageEventResultStatusAccepted || got.HighestContiguousSeq != 1234 {
		t.Fatalf("wrong stored original-stream ACK: %+v", got)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != 2 {
		t.Fatalf("same-key replay duplicated bytes: samples=%d", n)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_batch_replays WHERE incarnation_id=$1`, f.id.NodeIncarnationID); n != 1 {
		t.Fatalf("replay receipt not owned by presenting identity: %d", n)
	}
}

func TestENR05_D105RetiredCertificateRefused(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	provider, ca := pkitest.Authority(t)
	pkisvc, err := pki.NewService(f.pool, ca, provider, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandoffHTTP(t, f, pkisvc)
	issue := func(inc uuid.UUID) *x509.Certificate {
		_, csr := pkitest.CSR(t)
		cert, err := ca.Issue(t.Context(), pki.NodeIdentity{NodeID: f.id.NodeID, IncarnationID: inc}, csr)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			_, err := pki.StoreCertificate(t.Context(), tx, inc, cert)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return cert.Certificate
	}
	req := f.v.Cases[0].Request
	currentCert, oldCert := issue(f.id.NodeIncarnationID), issue(*req.OriginIncarnationId)
	if _, err := ca.Verify(oldCert); err != nil {
		t.Fatalf("old certificate must still be cryptographically valid: %v", err)
	}
	w := postHandoff(t, h, req, uuid.NewString(), oldCert)
	if w.Code != http.StatusForbidden || !bytes.Contains(w.Body.Bytes(), []byte("IDENTITY_REVOKED")) {
		t.Fatalf("retired certificate status=%d body=%s", w.Code, w.Body.String())
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_events`); n != int(f.v.Context.Prefix) {
		t.Fatal("retired certificate reached ingest")
	}
	w = postHandoff(t, h, req, uuid.NewString(), currentCert)
	if w.Code != http.StatusOK {
		t.Fatalf("current certificate hand-off status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestMTR03_D105PartialBatchAndIndependentACKs(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	req := f.v.Cases[0].Request
	valid := req.Events[0]
	late := f.v.Cases[1].Request.Events[0]
	late.EventSeq++
	laterValid := valid
	laterValid.EventSeq += 2
	var err error
	late, err = usage.WithComputedHash(late)
	if err != nil {
		t.Fatal(err)
	}
	laterValid, err = usage.WithComputedHash(laterValid)
	if err != nil {
		t.Fatal(err)
	}
	req.Events = []api.UsageEvent{valid, late, laterValid}
	got, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.HighestContiguousSeq != valid.EventSeq || nullableValue(got.BlockedSeq) != late.EventSeq || got.MaxSeenSeq != laterValid.EventSeq ||
		got.Results[0].Status != api.UsageEventResultStatusAccepted || got.Results[1].Status != api.UsageEventResultStatusRejected || got.Results[2].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("partial batch must preserve valid later event without ACKing overrun: %+v", got)
	}
	current := makeSignedEvent(t, f.id, 1, uuid.New(), f.v.Context.EnrolledAt, f.v.Context.EnrolledAt.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{})
	currentResult, err := f.svc.IngestBatch(t.Context(), f.id, api.UsageBatchRequest{NodeId: f.id.NodeID, NodeIncarnationId: f.id.NodeIncarnationID, LedgerEpoch: req.LedgerEpoch, Events: []api.UsageEvent{current}})
	if err != nil {
		t.Fatal(err)
	}
	if currentResult.NodeIncarnationId != f.id.NodeIncarnationID || currentResult.HighestContiguousSeq != 1 || !currentResult.BlockedSeq.IsNull() {
		t.Fatalf("current stream ACK must be independent: %+v", currentResult)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_checkpoints WHERE incarnation_id=$1 AND highest_contiguous_seq=1234 AND max_seen_seq=1236`, *req.OriginIncarnationId); n != 1 {
		t.Fatal("current stream changed retired stream checkpoint")
	}
}

func TestMTR03_D105ChangedDigestConflictPreservesOriginal(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	req := f.v.Cases[0].Request
	originalHash := req.Events[0].PayloadSha256
	if _, err := f.svc.IngestBatch(t.Context(), f.id, req); err != nil {
		t.Fatal(err)
	}
	req.Events[0].Records[0].UploadBytes++
	var err error
	req.Events[0], err = usage.WithComputedHash(req.Events[0])
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Results[0].Status != api.UsageEventResultStatusConflict || nullableValue(got.Results[0].Error).Code != api.ErrorCode(apierr.UsageEventConflict) {
		t.Fatalf("changed digest=%+v", got)
	}
	assertHandoffAudit(t, f, apierr.UsageEventConflict, "event_conflict", &req.Events[0].EventSeq)
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_events WHERE incarnation_id=$1 AND event_seq=1234 AND encode(payload_hash,'hex')=$2`, *req.OriginIncarnationId, originalHash); n != 1 {
		t.Fatal("conflict overwrote original digest")
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != 2 {
		t.Fatal("conflict added samples")
	}
}

func TestAT30_D105FreezeRecoveryAndOriginalReplayUnavailable(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	req := f.v.Cases[0].Request
	actor := "admin:" + uuid.NewString()
	if err := f.svc.FreezeIngest(t.Context(), actor, "TEST ONLY restore drill"); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.IngestBatch(t.Context(), f.id, req)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.DependencyUnavailable {
		t.Fatalf("frozen hand-off=%v", err)
	}
	recovery, err := f.svc.BeginRecovery(t.Context(), usage.RecoveryRequest{Actor: actor, Reason: "TEST ONLY restore drill"})
	if err != nil {
		t.Fatal(err)
	}
	if recovery.RecomputedStreams != 1 {
		t.Fatalf("recovery omitted original retired stream: %+v", recovery)
	}
	_, err = f.svc.IngestBatch(t.Context(), f.id, req)
	if !errors.As(err, &ae) || ae.Code != apierr.LedgerEpochMismatch {
		t.Fatalf("stale-epoch hand-off=%v", err)
	}
	req.LedgerEpoch = presence.NewNullableWithValue(recovery.NewLedgerEpoch)
	// The surviving ledger lost sequence 1234; its retained next event remains
	// in the retired stream. The declared gap must not become a current event.
	req.Events[0].EventSeq++
	req.Events[0], err = usage.WithComputedHash(req.Events[0])
	if err != nil {
		t.Fatal(err)
	}
	ranges := []api.ReplayUnavailableRange{{StartSeq: 1234, EndSeq: 1234, Reason: api.ReplayUnavailableRangeReasonLocalDataLost}}
	req.ReplayUnavailable = &ranges
	got, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.NodeIncarnationId != *req.OriginIncarnationId || got.HighestContiguousSeq != 1235 {
		t.Fatalf("recovery ACK belongs to original stream: %+v", got)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_events WHERE incarnation_id=$1 AND kind='replay_unavailable' AND event_seq=1234`, *req.OriginIncarnationId); n != 1 {
		t.Fatal("replay-unavailable did not target origin")
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_checkpoints WHERE incarnation_id=$1`, f.id.NodeIncarnationID); n != 0 {
		t.Fatal("recovery hand-off changed current stream")
	}
	if err := f.svc.CompleteRecovery(t.Context(), actor); err != nil {
		t.Fatal(err)
	}
}

func TestAT30_D105FreezeFencesInFlightHandoff(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	var freezeErr error
	ctx := usage.WithCheckpointInitTrace(t.Context(), &usage.CheckpointInitTrace{
		AfterInitialMiss: func() {
			freezeErr = f.svc.FreezeIngest(t.Context(), "admin:"+uuid.NewString(), "TEST ONLY in-flight hand-off freeze")
		},
	})
	_, err := f.svc.IngestBatch(ctx, f.id, f.v.Cases[0].Request)
	if freezeErr != nil {
		t.Fatal(freezeErr)
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.DependencyUnavailable {
		t.Fatalf("in-flight hand-off freeze=%v", err)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_events`); n != int(f.v.Context.Prefix) {
		t.Fatal("in-flight hand-off committed after freeze")
	}
}

func TestMTR03_D105AuditFailureRollsBackWindowRejection(t *testing.T) {
	f := newHandoffFixture(t, 0, time.Time{})
	mustExec(t, f.pool, `CREATE FUNCTION handoff_test_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'TEST ONLY audit failure'; END $$`)
	mustExec(t, f.pool, `CREATE TRIGGER handoff_test_audit_failure BEFORE INSERT ON audit_events
		FOR EACH ROW EXECUTE FUNCTION handoff_test_audit_failure()`)
	_, err := f.svc.IngestBatch(t.Context(), f.id, f.v.Cases[1].Request)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.DependencyUnavailable {
		t.Fatalf("audit failure=%v", err)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_event_rejections`); n != 0 {
		t.Fatal("window rejection committed without audit")
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_checkpoints`); n != 0 {
		t.Fatal("window rejection checkpoint committed without audit")
	}
}

func TestMTR03_D105HTTPRejectionsPersistAudit(t *testing.T) {
	for _, name := range []string{"origin-ineligible", "mixed-streams", "retired-window"} {
		t.Run(name, func(t *testing.T) {
			f := newHandoffFixture(t, 0, time.Time{})
			req := f.v.Cases[0].Request
			status, code, reason := http.StatusConflict, apierr.StateConflict, "origin_ineligible"
			var seq *decstr.U64
			switch name {
			case "origin-ineligible":
				origin := uuid.New()
				req.OriginIncarnationId = &origin
				req.Events[0].NodeIncarnationId = origin
			case "mixed-streams":
				ev := req.Events[0]
				ev.NodeIncarnationId = f.id.NodeIncarnationID
				req.Events = append(req.Events, ev)
				status, code, reason = http.StatusForbidden, apierr.IncarnationMismatch, "event_identity"
			case "retired-window":
				req = f.v.Cases[1].Request
				status, code, reason = http.StatusOK, apierr.ValidationFailed, "retired_window"
				seq = &req.Events[0].EventSeq
			}
			w := postHandoff(t, newHandoffHTTP(t, f, nil), req, uuid.NewString(), nil)
			if w.Code != status || !bytes.Contains(w.Body.Bytes(), []byte(code)) {
				t.Fatalf("status=%d body=%s, want %d %s", w.Code, w.Body.String(), status, code)
			}
			assertHandoffAudit(t, f, code, reason, seq)
			if n := handoffCount(t, f, `SELECT count(*) FROM audit_events WHERE action='usage.ingest_rejected' AND request_id IS NOT NULL`); n != 1 {
				t.Fatal("HTTP rejection audit lost request correlation on command rollback")
			}
			if n := handoffCount(t, f, `SELECT count(*) FROM usage_events`); n != int(f.v.Context.Prefix) {
				t.Fatal("rejected HTTP hand-off changed ledger")
			}
		})
	}
}
