//go:build integration

package agent

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestD088_RealHTTPWireFreezeConsumer(t *testing.T) {
	ctx := t.Context()
	cfg := nodestest.New(t, nodestest.NetBird{})
	runtime, err := httpwire.Open(ctx, dbtest.New(t), cfg.Nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease} {
		if _, err := runtime.Provider.(*signing.FileProvider).Rotate(ctx, purpose, 48*time.Hour, time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	enroll := func(node *uuid.UUID) (api.EnrollResponse, tls.Certificate) {
		t.Helper()
		created, err := runtime.Enrollment.CreateEnrollment(ctx, enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: node})
		if err != nil {
			t.Fatal(err)
		}
		token, err := created.Token.Get()
		if err != nil {
			t.Fatal(err)
		}
		req, key := pkitest.Request(t)
		enrolled, err := runtime.Enrollment.Enroll(ctx, token, req)
		if err != nil {
			t.Fatal(err)
		}
		cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
		return enrolled.Response, tls.Certificate{Certificate: [][]byte{cert.Raw}, PrivateKey: key, Leaf: cert}
	}
	enrolled, leaf := enroll(nil)
	compiled, err := runtime.DesiredStateCompiler().Compile(ctx, enrolled.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	handlers, err := httpapi.Compose(operations.Store{Pool: runtime.Pool}, &audit.Store{Pool: runtime.Pool}, &httpwire.Handlers{Runtime: runtime})
	if err != nil {
		t.Fatal(err)
	}
	hc := config.Defaults()
	hc.PublicOrigin = "https://portal.example.invalid"
	hc.AnonymousRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	hc.AgentRate = hc.AnonymousRate
	hc.IngressRate = hc.AnonymousRate
	hc.HeartbeatRate = hc.AnonymousRate
	hc.LeaseRate = hc.AnonymousRate
	h, err := httpapi.New(handlers, httpapi.Options{Config: hc, Agents: runtime.PKI, AgentListener: true, Commands: &httpapi.Commands{Pool: runtime.Pool, Retention: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.Transport(nil, httpwire.RequestContext(h))
	var mu sync.Mutex
	var reports []api.ObservationReport
	var receipts []api.ObservationAccepted
	var renewals []api.LeaseRenewalRequest
	var beats []watchdog.Beat
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Error(readErr)
			w.WriteHeader(500)
			return
		}
		req.Body = io.NopCloser(bytes.NewReader(raw))
		mu.Lock()
		switch req.URL.Path {
		case "/api/v1/agent/observations":
			var rep api.ObservationReport
			if err := json.Unmarshal(raw, &rep); err != nil {
				t.Error(err)
			}
			reports = append(reports, rep)
		case "/api/v1/agent/lease-renewals":
			var renewal api.LeaseRenewalRequest
			if err := json.Unmarshal(raw, &renewal); err != nil {
				t.Error(err)
			}
			renewals = append(renewals, renewal)
			beat, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath)
			if err != nil {
				t.Error(err)
			}
			beats = append(beats, beat)
		}
		mu.Unlock()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if req.URL.Path == "/api/v1/agent/observations" && rec.Code == http.StatusAccepted {
			var receipt api.ObservationAccepted
			if err := json.Unmarshal(rec.Body.Bytes(), &receipt); err != nil {
				t.Error(err)
			}
			mu.Lock()
			receipts = append(receipts, receipt)
			mu.Unlock()
		}
		for key, values := range rec.Header() {
			w.Header()[key] = values
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
	}))
	caRaw, err := os.ReadFile(cfg.Nodes.CACertificateFile)
	if err != nil {
		t.Fatal(err)
	}
	ca := x509.NewCertPool()
	if !ca.AppendCertsFromPEM(caRaw) {
		t.Fatal("CA")
	}
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: ca}
	server.StartTLS()
	t.Cleanup(server.Close)
	client := func(cert tls.Certificate) *controlclient.Client {
		t.Helper()
		roots := x509.NewCertPool()
		roots.AddCert(server.Certificate())
		c, err := controlclient.New(controlclient.Config{BaseURL: server.URL, RootCAs: roots, GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) { return &cert, nil }})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(c.Close)
		return c
	}
	c := client(leaf)
	r.eng.client = c
	r.eng.nodeID = enrolled.NodeId
	r.eng.incID = enrolled.NodeIncarnationId
	r.eng.cfg.ReleaseID = ""
	r.eng.desired = &compiled.Envelope.Signed
	r.eng.appliedGen = uint64(compiled.Envelope.Signed.ConfigGeneration)
	r.eng.trust, err = runtime.Provider.TrustBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Same real clone-conflict transaction as internal/nodes' observation fixture.
	for _, domain := range []string{"198.51.100.0/24", "203.0.113.0/24", "198.51.100.0/24"} {
		if _, err := runtime.Enrollment.ObserveEndpoint(ctx, leaf.Leaf, domain, 120*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	r.eng.lastObs = time.Time{}
	r.eng.leaseAttempted = true
	r.eng.leaseRetryAt = r.clock.Mono() + time.Hour
	if err := r.eng.observe(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(ctx); err != nil {
		t.Fatal(err)
	}
	r.eng.finishServingStep(ctx)
	mu.Lock()
	gotReports := append([]api.ObservationReport(nil), reports...)
	gotReceipts := append([]api.ObservationAccepted(nil), receipts...)
	gotRenewals := append([]api.LeaseRenewalRequest(nil), renewals...)
	gotBeats := append([]watchdog.Beat(nil), beats...)
	mu.Unlock()
	if len(gotReceipts) != 2 || len(gotRenewals) != 1 || len(gotReports) != 2 {
		t.Fatalf("receipts=%d renewals=%d reports=%d log=%s", len(gotReceipts), len(gotRenewals), len(gotReports), r.log.String())
	}
	for _, receipt := range gotReceipts {
		if receipt.IdentityFrozen == nil || !*receipt.IdentityFrozen {
			t.Fatal("real emitter omitted freeze")
		}
	}
	if gotReports[1].MemberForwardingEnabled || !gotReports[1].AppliedConfigGeneration.IsNull() || !hasMessage(gotReports[1].Errors, "identity_frozen") || !gotRenewals[0].AppliedConfigGeneration.IsNull() {
		t.Fatal("consumer leaked a serving claim or generation ACK")
	}
	if gotBeats[0].Serving || gotBeats[0].Reason != "identity_frozen" || len(r.plane.last.Bindings) != 0 || !r.eng.identityFrozen {
		t.Fatal("renewal preceded withdrawal/heartbeat")
	}
	// This enrolled central fixture is unpublished, so D-067's published
	// 60-second backstop does not apply. Verify the real grant-less response
	// belongs to this incarnation, rather than asserting the fake's lifetime.
	if len(r.eng.lease.Members) != 0 || r.eng.lease.NodeIncarnationId != enrolled.NodeIncarnationId || !r.eng.leaseFresh {
		t.Fatalf("real frozen lease was not accepted: members=%d window=%s log=%s", len(r.eng.lease.Members), r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt), r.log.String())
	}
	if latch, _, err := r.eng.store.Meta(ctx, "identity_frozen"); err != nil || latch != "1" {
		t.Fatal("latch not durable", err)
	}
	for _, key := range []uuid.UUID{gotReports[0].ReportId, uuid.New()} {
		replay, err := c.Observe(ctx, key, gotReports[0])
		if err != nil {
			t.Fatal(err)
		}
		if replay.IdentityFrozen == nil || !*replay.IdentityFrozen || !replay.ReceivedAt.Equal(gotReceipts[0].ReceivedAt) {
			t.Fatal("report/key replay lost live freeze or durable receipt")
		}
	}
	_, etag, err := runtime.Nodes.GetNode(ctx, enrolled.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = runtime.Nodes.RevokeNode(ctx, nodes.RevokeNodeParams{NodeID: enrolled.NodeId, Principal: "admin:" + uuid.NewString(), IdempotencyKey: uuid.New(), IfMatch: etag, Request: api.ReasonRequest{Reason: "TEST ONLY replace frozen incarnation"}})
	if err != nil {
		t.Fatal(err)
	}
	replacement, replacementLeaf := enroll(&enrolled.NodeId)
	next := gotReports[1]
	next.NodeIncarnationId = replacement.NodeIncarnationId
	next.ReportId = uuid.New()
	receipt, err := client(replacementLeaf).Observe(ctx, next.ReportId, next)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.IdentityFrozen != nil {
		t.Fatal("replacement inherited historical freeze")
	}
	// Restart through the normal durable-state loader; the original local latch
	// stays closed even though a different incarnation's receipt omitted the flag.
	restartEngine(t, r, r.plane)
	if !r.eng.identityFrozen || r.eng.serving {
		t.Fatal("restart lost freeze")
	}
	t.Log("real mTLS httpwire receipt -> withdrawal + durable latch + non-serving heartbeat before immediate ACK-less renewal; report/key replays retain flag; replacement omits flag")
}
