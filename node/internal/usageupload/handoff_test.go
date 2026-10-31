package usageupload

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

var presentingInc = uuid.MustParse("01000000-0000-4000-8000-000000000014")

type handoffVector struct {
	Cases []struct {
		Name     string          `json:"name"`
		Request  json.RawMessage `json:"request"`
		Response json.RawMessage `json:"response"`
	} `json:"cases"`
}

func vectorCase(t *testing.T, name string) (wireBatch, []byte, []byte) {
	t.Helper()
	raw, err := os.ReadFile("../../../contracts/vectors/usage-retired-handoff.json")
	if err != nil {
		t.Fatal(err)
	}
	var v handoffVector
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	for _, c := range v.Cases {
		if c.Name != name {
			continue
		}
		var wire wireBatch
		if err := json.Unmarshal(c.Request, &wire); err != nil {
			t.Fatal(err)
		}
		return wire, c.Request, c.Response
	}
	t.Fatalf("missing vector %s", name)
	return wireBatch{}, nil, nil
}

// Fixture precondition: the previous release persisted seq 1234 and had a
// verified checkpoint through 1233. No retired certificate is constructed.
func retiredVector(t *testing.T, name string) (*usagestore.Store, wireBatch, []byte, []byte) {
	t.Helper()
	wire, request, response := vectorCase(t, name)
	s := seed(t, 1, usagestore.Options{})
	path := s.Path()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := canonjson.Canonicalize(wire.Events[0])
	if err != nil {
		t.Fatal(err)
	}
	var ev api.UsageEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`UPDATE events SET seq = 1234, epoch_id = ?, interval_start = ?, interval_end = ?, payload = ?, payload_sha256 = ?, payload_len = ?`,
		ev.MeterEpochId.String(), ev.IntervalStart.Format(time.RFC3339Nano), ev.IntervalEnd.Format(time.RFC3339Nano), raw, ev.PayloadSha256, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE meta SET next_seq = 1235, acked_through = 1233, ledger_epoch = ?`, epochA.String()); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := usagestore.Open(path, upNode, presentingInc, usagestore.Options{RetirePreviousAt: ev.IntervalStart})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, wire, request, response
}

func currentEvent(t *testing.T, s *usagestore.Store) {
	t.Helper()
	at := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	ep := uuid.New()
	_, err := s.Commit(context.Background(), usagestore.Commit{Open: &usagestore.NewEpoch{ID: ep, Reason: "baseline", BootID: "new-boot"}, Counters: []usagestore.Counter{upCounter(0)}, SampledAt: at})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Commit(context.Background(), usagestore.Commit{Epoch: ep, Counters: []usagestore.Counter{upCounter(10)}, SampledAt: at.Add(10 * time.Second), Event: &usagestore.Draft{EpochID: ep, Start: at, End: at.Add(10 * time.Second), Quality: "complete", Records: []usagestore.DraftRecord{{MemberID: upMem, DeviceID: upDev, BindingRevision: 7, PeriodID: "2026-10", Upload: 10}}}})
	if err != nil {
		t.Fatal(err)
	}
}

func currentResponse() []byte {
	return bytes.ReplaceAll(resultBody("1", nil, [][2]string{{"1", "accepted"}}), []byte(upInc.String()), []byte(presentingInc.String()))
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	a, err := canonjson.Canonicalize(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := canonjson.Canonicalize(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("JSON changed\ngot %s\nwant %s", a, b)
	}
}

// Only the current test certificate exists; the server requires and verifies it.
func handoffTLS(t *testing.T, handler http.Handler) *HTTPTransport {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: presentingInc.String()}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) != 1 || r.TLS.PeerCertificates[0].Subject.CommonName != presentingInc.String() {
			t.Error("missing current mTLS identity")
			w.WriteHeader(403)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	srv.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	client := srv.Client()
	client.Transport.(*http.Transport).TLSClientConfig.Certificates = []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}
	return &HTTPTransport{BaseURL: srv.URL, Client: client}
}

func TestD105_HandoffAcceptedThenCurrent(t *testing.T) {
	ctx := context.Background()
	s, wire, request, response := retiredVector(t, "accepted")
	for _, name := range []string{"accepted", "idempotent-replay"} {
		ex, err := os.ReadFile("../../../contracts/examples/valid/UsageBatchRequest.handoff-" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, ex, request)
	}
	currentEvent(t, s)
	calls := 0
	tr := handoffTLS(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agent/usage-batches" || r.Method != http.MethodPost {
			t.Error("unexpected route")
		}
		if _, err := uuid.Parse(r.Header.Get("Idempotency-Key")); err != nil {
			t.Error(err)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		calls++
		if calls == 1 {
			assertJSONEqual(t, body, request)
			_, _ = w.Write(response)
			return
		}
		var batch wireBatch
		if err := json.Unmarshal(body, &batch); err != nil {
			t.Error(err)
		}
		if batch.OriginIncarnationID != nil || batch.NodeIncarnationID != presentingInc {
			t.Error("origin leaked to current batch")
		}
		_, _ = w.Write(currentResponse())
	}))
	u := newUp(t, s, tr, (&sleeper{}).Sleep)
	if err := u.Once(ctx); err != nil {
		t.Fatal(err)
	}
	head, _ := s.Head(ctx)
	streams, _ := s.RetiredStreams(ctx)
	if head.AckedThrough != 0 || streams[0].AckedThrough != 1234 {
		t.Fatal("ACK did not belong only to original stream")
	}
	for i := 0; i < 3; i++ {
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatalf("handoff did not stop: %d calls", calls)
	}
	head, _ = s.Head(ctx)
	if head.AckedThrough != 1 {
		t.Fatal("current stream did not progress")
	}
	ev, _ := s.Event(ctx, 1)
	if bytes.Contains(ev.Payload, []byte(wire.OriginIncarnationID.String())) {
		t.Fatal("new event in old incarnation")
	}
}

func TestMTR03_HandoffReplayAfterLostACK(t *testing.T) {
	ctx := context.Background()
	s, _, request, accepted := retiredVector(t, "accepted")
	_, replayRequest, duplicate := vectorCase(t, "idempotent-replay")
	assertJSONEqual(t, request, replayRequest)
	var calls []postCall
	tr := handoffTLS(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assertJSONEqual(t, body, request)
		calls = append(calls, postCall{key: r.Header.Get("Idempotency-Key"), body: body})
		if len(calls) == 1 {
			_, _ = w.Write(accepted)
		} else {
			_, _ = w.Write(duplicate)
		}
	}))
	u := newUp(t, s, tr, (&sleeper{}).Sleep)
	u.hook = func(step string) error {
		if step == StepApply {
			return errors.New("lost ACK")
		}
		return nil
	}
	if err := u.Once(ctx); err == nil {
		t.Fatal("expected lost ACK")
	}
	path := s.Path()
	_ = s.Close()
	st, err := usagestore.Open(path, upNode, presentingInc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	u = newUp(t, st, tr, (&sleeper{}).Sleep)
	if err := u.Once(ctx); err != nil {
		t.Fatal(err)
	}
	if err := u.Once(ctx); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].key != calls[1].key || !bytes.Equal(calls[0].body, calls[1].body) {
		t.Fatal("replay changed body/key or failed to stop")
	}
	streams, _ := st.RetiredStreams(ctx)
	if streams[0].AckedThrough != 1234 {
		t.Fatal("duplicate receipt not acknowledged")
	}
}

func TestD105_AfterWindowRecordedOnce(t *testing.T) {
	ctx := context.Background()
	s, _, request, response := retiredVector(t, "after-retirement-window")
	currentEvent(t, s)
	calls := 0
	tr := handoffTLS(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if calls == 1 {
			assertJSONEqual(t, body, request)
			_, _ = w.Write(response)
		} else {
			_, _ = w.Write(currentResponse())
		}
	}))
	u := newUp(t, s, tr, (&sleeper{}).Sleep)
	if err := u.Once(ctx); err != nil {
		t.Fatal(err)
	}
	path := s.Path()
	_ = s.Close()
	st, err := usagestore.Open(path, upNode, presentingInc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	u = newUp(t, st, tr, (&sleeper{}).Sleep)
	for i := 0; i < 3; i++ {
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
	}
	audit, err := st.HandoffRejections(ctx)
	if err != nil || len(audit) != 1 || audit[0].Seq != 1234 || audit[0].Code != "VALIDATION_FAILED" {
		t.Fatalf("audit %+v %v", audit, err)
	}
	streams, _ := st.RetiredStreams(ctx)
	if streams[0].AckedThrough != 1233 || streams[0].BlockedSeq == nil || *streams[0].BlockedSeq != 1234 || calls != 2 {
		t.Fatalf("rejection retried or ACK skipped: %+v calls=%d", streams, calls)
	}
}

func TestD105_FinalOriginRejectionAndTransientRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		code   string
		final  bool
	}{
		{"foreign", 409, "STATE_CONFLICT", true}, {"pre-retirement", 409, "STATE_CONFLICT", true},
		{"identity", 403, "INCARNATION_MISMATCH", true}, {"revoked-presenter", 403, "IDENTITY_REVOKED", false},
		{"frozen", 503, "DEPENDENCY_UNAVAILABLE", false}, {"epoch", 409, "LEDGER_EPOCH_MISMATCH", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _, _, response := retiredVector(t, "accepted")
			currentEvent(t, s)
			retiredCalls := 0
			tr := &fakeTR{postFn: func(key string, body []byte) (int, []byte, time.Duration, error) {
				var batch wireBatch
				if err := json.Unmarshal(body, &batch); err != nil {
					t.Fatal(err)
				}
				if batch.OriginIncarnationID == nil {
					return 200, currentResponse(), 0, nil
				}
				retiredCalls++
				if retiredCalls == 1 {
					return tc.status, []byte(`{"error":{"code":"` + tc.code + `","retryable":false}}`), 0, nil
				}
				return 200, response, 0, nil
			}, getFn: func() (int, []byte, error) { return 404, nil, nil }}
			u := newUp(t, s, tr, (&sleeper{}).Sleep)
			err := u.Once(ctx)
			if (err == nil) != tc.final {
				t.Fatalf("first outcome %v", err)
			}
			if err := u.Once(ctx); err != nil {
				t.Fatal(err)
			}
			head, _ := s.Head(ctx)
			if head.AckedThrough != 1 {
				t.Fatal("current starved by origin failure")
			}
			for i := 0; i < 2; i++ {
				if err := u.Once(ctx); err != nil {
					t.Fatal(err)
				}
			}
			audit, _ := s.HandoffRejections(ctx)
			if tc.final {
				if len(audit) != 1 || audit[0].Seq != 0 || retiredCalls != 1 {
					t.Fatal("final origin retried or not audited")
				}
			} else if len(audit) != 0 || retiredCalls != 2 {
				t.Fatal("transient rejection became final")
			}
		})
	}
}

func TestD105_WrongStreamACKRejected(t *testing.T) {
	s, _, _, response := retiredVector(t, "accepted")
	response = bytes.ReplaceAll(response, []byte(upInc.String()), []byte(presentingInc.String()))
	u := newUp(t, s, &fakeTR{postFn: func(string, []byte) (int, []byte, time.Duration, error) { return 200, response, 0, nil }}, (&sleeper{}).Sleep)
	if err := u.Once(context.Background()); err == nil {
		t.Fatal("ACK accepted for presenter instead of origin")
	}
	streams, _ := s.RetiredStreams(context.Background())
	if streams[0].AckedThrough != 1233 || streams[0].Pending != 1 {
		t.Fatal("wrong ACK mutated origin")
	}
}

func TestD105_OrderedBatchesAroundFinalReceipt(t *testing.T) {
	for _, rejectMiddle := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "rejected-middle"}[rejectMiddle], func(t *testing.T) {
			ctx := context.Background()
			old := seed(t, 3, usagestore.Options{})
			saved, err := old.Unacked(ctx, 0, 100, 512000)
			if err != nil {
				t.Fatal(err)
			}
			path := old.Path()
			_ = old.Close()
			st, err := usagestore.Open(path, upNode, presentingInc, usagestore.Options{RetirePreviousAt: saved[0].Start})
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			currentEvent(t, st)
			order := []uint64{}
			tr := &fakeTR{postFn: func(key string, body []byte) (int, []byte, time.Duration, error) {
				var batch wireBatch
				if err := json.Unmarshal(body, &batch); err != nil {
					t.Fatal(err)
				}
				if batch.OriginIncarnationID == nil {
					if len(order) != 3 {
						t.Error("current sent before held tail")
					}
					return 200, currentResponse(), 0, nil
				}
				if len(batch.Events) != 1 {
					t.Fatal("batch limit ignored")
				}
				var event api.UsageEvent
				if err := json.Unmarshal(batch.Events[0], &event); err != nil {
					t.Fatal(err)
				}
				seq := uint64(event.EventSeq)
				order = append(order, seq)
				if seq < 1 || seq > 3 {
					t.Fatal("event re-sequenced")
				}
				if !bytes.Equal(batch.Events[0], saved[seq-1].Payload) {
					t.Fatal("original payload changed")
				}
				highest := event.EventSeq.String()
				status := "accepted"
				var blocked *string
				if rejectMiddle && seq >= 2 {
					highest = "1"
					b := "2"
					blocked = &b
					if seq == 2 {
						status = "rejected"
					}
				}
				return 200, resultBody(highest, blocked, [][2]string{{event.EventSeq.String(), status}}), 0, nil
			}}
			u := newUp(t, st, tr, (&sleeper{}).Sleep)
			u.maxEv = 1
			for i := 0; i < 6; i++ {
				if err := u.Once(ctx); err != nil {
					t.Fatal(err)
				}
			}
			if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
				t.Fatalf("order %v", order)
			}
			streams, _ := st.RetiredStreams(ctx)
			want := uint64(3)
			if rejectMiddle {
				want = 1
			}
			if streams[0].AckedThrough != want || streams[0].Pending != 0 {
				t.Fatalf("stream %+v", streams[0])
			}
			audit, _ := st.HandoffRejections(ctx)
			if rejectMiddle && (len(audit) != 1 || audit[0].Seq != 2) {
				t.Fatalf("audit %+v", audit)
			}
			head, _ := st.Head(ctx)
			if head.AckedThrough != 1 {
				t.Fatal("current blocked by retired rejection")
			}
		})
	}
}

func TestD105_RetiredRetryBackoffBounded(t *testing.T) {
	s, _, _, _ := retiredVector(t, "accepted")
	calls := 0
	var sleeps []time.Duration
	stop := errors.New("stop after bounded attempts")
	u, err := New(Config{Store: s, Transport: &fakeTR{postFn: func(string, []byte) (int, []byte, time.Duration, error) { calls++; return 503, nil, 0, nil }}, BackoffBase: time.Second, BackoffCap: 4 * time.Second, Sleep: func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		if len(sleeps) == 5 {
			return stop
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Run(context.Background()); !errors.Is(err, stop) {
		t.Fatal(err)
	}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second, 4 * time.Second}
	if calls != 5 || len(sleeps) != len(want) {
		t.Fatalf("calls=%d sleeps=%v", calls, sleeps)
	}
	for i := range want {
		if sleeps[i] != want[i] {
			t.Fatalf("backoff %v", sleeps)
		}
	}
	audit, _ := s.HandoffRejections(context.Background())
	streams, _ := s.RetiredStreams(context.Background())
	if len(audit) != 0 || streams[0].Pending != 1 {
		t.Fatal("transient attempts discarded evidence")
	}
}
