package usageupload

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// TestAT07_SnapshotWithheldACK restarts the uploader against fakecontrol
// while the central usage route withholds acknowledgements. Counted samples
// stay in the outbox. The snapshot hashes, sequences and watermark do not
// change across that restart (§10.6, §10.8, AT07).
func TestAT07_SnapshotWithheldACK(t *testing.T) {
	ctx := context.Background()
	srv, err := fakecontrol.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	srv.SetUsageUnavailable(100)

	node := upNode
	inc := upInc
	path := filepath.Join(t.TempDir(), "usage.db")
	clk := &wall{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
	store, err := usagestore.Open(path, node, inc, usagestore.Options{Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	src := &countSrc{}
	dev := upDev
	mem := upMem
	src.set(countObs("boot-1", dev, mem, 7, 0, 0))
	m, err := meter.New(meter.Config{
		NodeID: node, IncarnationID: inc, Store: store, Source: src,
		Interval: 10 * time.Second, Now: clk.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(countObs("boot-1", dev, mem, 7, 100, 4))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(countObs("boot-1", dev, mem, 7, 250, 9))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	m.Close()

	tr := &withholdTR{srv: srv, node: node, inc: inc}
	up, err := New(Config{Store: store, Transport: tr, Sleep: func(context.Context, time.Duration) error { return nil }, Rand: func() float64 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	first, err := usagestore.ReadSnapshot(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := first.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if first.UnackedCount != 2 || first.AckedThrough != "0" || first.LastEventSeq != "2" {
		t.Fatalf("before upload %+v", first)
	}
	if err := up.Once(ctx); err == nil {
		t.Fatal("withheld usage endpoint acknowledged the batch")
	}
	if tr.posts() != 1 || tr.lastStatus() != http.StatusServiceUnavailable {
		t.Fatalf("posts %d status %d", tr.posts(), tr.lastStatus())
	}
	second, err := usagestore.ReadSnapshot(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := second.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("upload tore the snapshot\n%s\n%s", firstJSON, secondJSON)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = usagestore.Open(path, node, inc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	up, err = New(Config{Store: store, Transport: tr, Sleep: func(context.Context, time.Duration) error { return nil }, Rand: func() float64 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	if err := up.Once(ctx); err == nil {
		t.Fatal("restarted uploader was acknowledged")
	}
	if tr.posts() != 2 || tr.lastStatus() != http.StatusServiceUnavailable {
		t.Fatalf("after restart posts %d status %d", tr.posts(), tr.lastStatus())
	}
	third, err := usagestore.ReadSnapshot(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	thirdJSON, err := third.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, thirdJSON) {
		t.Fatalf("restart changed the snapshot\n%s\n%s", firstJSON, thirdJSON)
	}
	if third.AckedThrough != "0" || third.LastEventSeq != "2" || len(third.UnackedEvents) != 2 {
		t.Fatalf("watermark %+v", third)
	}
	for i, ev := range third.UnackedEvents {
		stored, err := store.Event(ctx, uint64(i+1))
		if err != nil {
			t.Fatal(err)
		}
		sum, err := canonjson.HashHex(stored.Payload, "payload_sha256")
		if err != nil || ev.PayloadSHA256 != sum || string(stored.Payload) == "" {
			t.Fatalf("seq %d hash %s %s", i+1, ev.PayloadSHA256, sum)
		}
		if !bytes.Contains(firstJSON, []byte(ev.PayloadSHA256)) {
			t.Fatal("hash missing from the first snapshot")
		}
	}
	if srv.UsageReport().Nodes != nil && len(srv.UsageReport().Nodes) != 0 {
		t.Fatalf("fakecontrol stored events while withholding: %+v", srv.UsageReport())
	}
}

type wall struct {
	mu sync.Mutex
	t  time.Time
}

func (w *wall) Now() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.t
}

func (w *wall) Set(t time.Time) {
	w.mu.Lock()
	w.t = t
	w.mu.Unlock()
}

type countSrc struct {
	mu  sync.Mutex
	obs meter.Observation
}

func (c *countSrc) Read(context.Context) (meter.Observation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.obs, nil
}

func (c *countSrc) set(obs meter.Observation) {
	c.mu.Lock()
	c.obs = obs
	c.mu.Unlock()
}

func countObs(boot string, dev, mem uuid.UUID, rev, upBytes, upPackets uint64) meter.Observation {
	return meter.Observation{
		BootID: boot,
		Bindings: []enforce.Binding{{
			MemberID: mem, DeviceID: dev, BindingRevision: rev,
		}},
		Counters: map[string]enforce.CounterValue{
			meter.CounterName(dev, meter.DirUpload): {Packets: upPackets, Bytes: upBytes},
		},
	}
}

// withholdTR posts through fakecontrol's usage handler. The server is set
// to withhold acknowledgements. The test does not edit outbox rows.
type withholdTR struct {
	srv  *fakecontrol.Server
	node uuid.UUID
	inc  uuid.UUID

	mu     sync.Mutex
	n      int
	status int
}

func (w *withholdTR) posts() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.n
}

func (w *withholdTR) lastStatus() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

func (w *withholdTR) PostUsageBatch(ctx context.Context, key string, body []byte) (int, []byte, time.Duration, error) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", bytes.NewReader(body))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	req.TLS = peerTLS(w.node, w.inc)
	rec := httptest.NewRecorder()
	w.srv.ServeHTTP(rec, req)
	w.mu.Lock()
	w.n++
	w.status = rec.Code
	w.mu.Unlock()
	return rec.Code, rec.Body.Bytes(), 0, nil
}

func (w *withholdTR) GetUsageCheckpoint(context.Context) (int, []byte, error) {
	return http.StatusNotFound, []byte(`{"error":{"code":"NOT_FOUND","message":"No checkpoint.","request_id":"req","retryable":false}}`), nil
}

func peerTLS(node, inc uuid.UUID) *tls.ConnectionState {
	return &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{
		URIs: []*url.URL{
			mustURL("urn:fvpn:node:" + node.String()),
			mustURL("urn:fvpn:incarnation:" + inc.String()),
		},
	}}}
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
