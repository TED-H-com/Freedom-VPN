package fakecontrol

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestUsageContiguousAndConflict(t *testing.T) {
	s := startUsage(t)
	node := uuid.Must(uuid.NewV7())
	inc := uuid.Must(uuid.NewV7())
	ev1 := usageEvent(t, 1, "a")
	ev2 := usageEvent(t, 2, "b")
	ev4 := usageEvent(t, 4, "d")

	rec := postUsage(t, s, node, inc, batchBody(t, nil, node, inc, ev1, ev2))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var okBody struct {
		Highest string `json:"highest_contiguous_seq"`
		Blocked any    `json:"blocked_seq"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &okBody); err != nil {
		t.Fatal(err)
	}
	if okBody.Highest != "2" || okBody.Blocked != nil {
		t.Fatalf("ack %+v", okBody)
	}

	// A gap is stored but does not move the contiguous ACK.
	rec = postUsage(t, s, node, inc, batchBody(t, nil, node, inc, ev4))
	if rec.Code != http.StatusOK {
		t.Fatalf("gap status %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &okBody); err != nil {
		t.Fatal(err)
	}
	if okBody.Highest != "2" {
		t.Fatalf("gap moved the ack to %s", okBody.Highest)
	}

	// Same payload is a duplicate and keeps the ack.
	rec = postUsage(t, s, node, inc, batchBody(t, nil, node, inc, ev1))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}

	// A different hash for sequence 1 is a per-event conflict on 200.
	// A later sequence in the same batch is not stored past the hole.
	bad := usageEvent(t, 1, "other")
	later := usageEvent(t, 5, "e")
	rec = postUsage(t, s, node, inc, batchBody(t, nil, node, inc, bad, later))
	if rec.Code != http.StatusOK {
		t.Fatalf("conflict %d %s", rec.Code, rec.Body.String())
	}
	var conflicted api.UsageBatchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &conflicted); err != nil {
		t.Fatalf("conflict body: %v %s", err, rec.Body.String())
	}
	blocked, err := conflicted.BlockedSeq.Get()
	if err != nil || blocked.String() != "1" || conflicted.HighestContiguousSeq.String() != "2" {
		t.Fatalf("blocked %+v highest %s err %v", conflicted.BlockedSeq, conflicted.HighestContiguousSeq, err)
	}
	if len(conflicted.Results) != 2 || conflicted.Results[0].Status != api.UsageEventResultStatusConflict || conflicted.Results[1].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("results %+v", conflicted.Results)
	}
	cerr, err := conflicted.Results[0].Error.Get()
	if err != nil || cerr.Code != api.ErrorCodeUSAGEEVENTCONFLICT || cerr.Message == "" {
		t.Fatalf("conflict error %+v %v", cerr, err)
	}
	view := s.UsageReport()
	if len(view.Nodes) != 1 || view.Nodes[0].Highest != 2 || view.Nodes[0].Events != 3 {
		t.Fatalf("ledger changed after conflict: %+v", view)
	}

	// The whole-request 409 path stays behind the knob and stores nothing.
	s.SetUsageConflict(1)
	rec = postUsage(t, s, node, inc, batchBody(t, nil, node, inc, later))
	if rec.Code != http.StatusConflict || !bytes.Contains(rec.Body.Bytes(), []byte("USAGE_EVENT_CONFLICT")) {
		t.Fatalf("409 knob %d %s", rec.Code, rec.Body.String())
	}
	view = s.UsageReport()
	if len(view.Nodes) != 1 || view.Nodes[0].Highest != 2 || view.Nodes[0].Events != 3 {
		t.Fatalf("409 knob stored the batch: %+v", view)
	}

	s.SetUsageUnavailable(1)
	rec = postUsage(t, s, node, inc, batchBody(t, nil, node, inc, ev1))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("503 got %d", rec.Code)
	}
	s.SetUsageRetryAfter(1, 3)
	rec = postUsage(t, s, node, inc, batchBody(t, nil, node, inc, ev1))
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("429 %d retry %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	if got := s.UsageReport(); got.Nodes[0].Highest != 2 {
		t.Fatalf("knobs moved the ack: %+v", got)
	}

	other := uuid.Must(uuid.NewV7())
	raw, _ := json.Marshal(other)
	rec = postUsage(t, s, node, inc, batchBody(t, raw, node, inc, ev1))
	if rec.Code != http.StatusConflict || !bytes.Contains(rec.Body.Bytes(), []byte("LEDGER_EPOCH_MISMATCH")) {
		t.Fatalf("epoch %d %s", rec.Code, rec.Body.String())
	}
}

func TestUsageCheckpointRoundTrip(t *testing.T) {
	s := startUsage(t)
	node := uuid.Must(uuid.NewV7())
	inc := uuid.Must(uuid.NewV7())
	ctx := context.Background()
	trust, err := envelope.ParseNodeTrustStoreJSON(s.TrustJSON())
	if err != nil {
		t.Fatal(err)
	}
	store, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), node, inc, usagestore.Options{Trust: trust})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	raw := getCheckpoint(t, s, node, inc)
	now := time.Now().UTC()
	if err := store.ApplyCheckpoint(ctx, raw, now); err != nil {
		t.Fatal(err)
	}
	head, err := store.Head(ctx)
	if err != nil || head.LedgerEpoch == uuid.Nil || head.AckedThrough != 0 {
		t.Fatalf("head %+v %v", head, err)
	}

	at := now.Add(-time.Minute)
	epoch := uuid.Must(uuid.NewV7())
	dev := uuid.Must(uuid.NewV7())
	mem := uuid.Must(uuid.NewV7())
	counter := usagestore.Counter{
		Name: "u_" + dev.String(), DeviceID: dev, Direction: "upload",
		Packets: 1, Bytes: 10, MemberID: mem, BindingRevision: 1,
	}
	if _, err := store.Commit(ctx, usagestore.Commit{
		Counters: []usagestore.Counter{counter},
		Open:     &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: "boot"},
		BootID:   "boot", SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	counter.Bytes = 30
	seq, err := store.Commit(ctx, usagestore.Commit{
		Epoch: epoch, Counters: []usagestore.Counter{counter}, BootID: "boot", SampledAt: at.Add(10 * time.Second),
		Event: &usagestore.Draft{
			EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []usagestore.DraftRecord{{
				MemberID: mem, DeviceID: dev, BindingRevision: 1, PeriodID: "2026-10", Upload: 20,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := store.Event(ctx, seq)
	if err != nil {
		t.Fatal(err)
	}
	posted := postUsage(t, s, node, inc, batchBody(t, nil, node, inc, ev.Payload))
	if posted.Code != http.StatusOK {
		t.Fatalf("post %d %s", posted.Code, posted.Body.String())
	}
	var result struct {
		Highest     string    `json:"highest_contiguous_seq"`
		LedgerEpoch uuid.UUID `json:"ledger_epoch"`
	}
	if err := json.Unmarshal(posted.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.LedgerEpoch != head.LedgerEpoch || result.Highest != "1" {
		t.Fatalf("result %+v want epoch %s", result, head.LedgerEpoch)
	}
	if err := store.ApplyACK(ctx, usagestore.BatchACK{
		LedgerEpoch: result.LedgerEpoch, Highest: 1,
		Results: []usagestore.EventResult{{Seq: seq, Status: usagestore.StatusAccepted}},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SetUsageCheckpoint("same", 0); err != nil {
		t.Fatal(err)
	}
	low := getCheckpoint(t, s, node, inc)
	if err := store.ApplyCheckpoint(ctx, low, time.Now().UTC()); !errors.Is(err, usagestore.ErrAckRegress) {
		t.Fatalf("same epoch: %v", err)
	}
	if got, _ := store.AckedThrough(ctx); got != 1 {
		t.Fatalf("acked %d", got)
	}

	next, err := s.SetUsageCheckpoint("new", 0)
	if err != nil || next == head.LedgerEpoch {
		t.Fatal(err)
	}
	fresh := getCheckpoint(t, s, node, inc)
	if err := store.ApplyCheckpoint(ctx, fresh, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.AckedThrough(ctx); got != 0 {
		t.Fatalf("new epoch acked %d", got)
	}
	ev, err = store.Event(ctx, seq)
	if err != nil || ev.State != "outbox" {
		t.Fatalf("event %+v %v", ev, err)
	}
	head, err = store.Head(ctx)
	if err != nil || head.LedgerEpoch != next {
		t.Fatalf("epoch %+v %v", head, err)
	}
}

func startUsage(t *testing.T) *Server {
	t.Helper()
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func usageEvent(t *testing.T, seq int, mark string) json.RawMessage {
	t.Helper()
	body := []byte(`{"event_seq":"` + itoa(seq) + `","mark":"` + mark + `","payload_sha256":""}`)
	sum, err := canonjson.HashHex(body, "payload_sha256")
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(`{"event_seq":"` + itoa(seq) + `","mark":"` + mark + `","payload_sha256":"` + sum + `"}`)
	sum2, err := canonjson.HashHex(body, "payload_sha256")
	if err != nil || sum2 != sum {
		t.Fatalf("hash %s %s %v", sum, sum2, err)
	}
	return body
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func batchBody(t *testing.T, epoch []byte, node, inc uuid.UUID, events ...[]byte) []byte {
	t.Helper()
	if epoch == nil {
		epoch = []byte("null")
	}
	var buf bytes.Buffer
	buf.WriteString(`{"events":[`)
	for i, ev := range events {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(ev)
	}
	buf.WriteString(`],"ledger_epoch":`)
	buf.Write(epoch)
	buf.WriteString(`,"node_id":"` + node.String() + `","node_incarnation_id":"` + inc.String() + `"}`)
	return buf.Bytes()
}

func postUsage(t *testing.T, s *Server, node, inc uuid.UUID, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := agentReq(http.MethodPost, "/api/v1/agent/usage-batches", body, node, inc)
	rec := httptest.NewRecorder()
	s.usage(rec, req)
	return rec
}

func getCheckpoint(t *testing.T, s *Server, node, inc uuid.UUID) []byte {
	t.Helper()
	req := agentReq(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, node, inc)
	rec := httptest.NewRecorder()
	s.usageCheckpoint(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("checkpoint %d %s", rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}

func agentReq(method, path string, body []byte, node, inc uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{
		URIs: []*url.URL{
			mustURI("urn:fvpn:node:" + node.String()),
			mustURI("urn:fvpn:incarnation:" + inc.String()),
		},
	}}}
	return req
}
