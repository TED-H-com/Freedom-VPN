//go:build integration

package usage_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type usageHTTPStack struct {
	pool    *pgxpool.Pool
	prov    *signing.FileProvider
	svc     *usage.Service
	public  http.Handler
	agent   http.Handler
	adminP  auth.Principal
	memP    auth.Principal
	agentP  auth.Principal
	disp    *worker.Dispatcher
	memberA uuid.UUID
	deviceA uuid.UUID
	node1   uuid.UUID
	inc1    uuid.UUID
	node2   uuid.UUID
	inc2    uuid.UUID
}

func newUsageHTTPStack(t *testing.T) *usageHTTPStack {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()

	// Initialize a real PKI FileProvider for Recovery and LedgerEpoch signing (§16.3, §17.3).
	vaultDir := filepath.Join(t.TempDir(), "vault")
	prov, err := signing.InitializeFileProvider(vaultDir)
	if err != nil {
		t.Fatalf("InitializeFileProvider: %v", err)
	}
	for _, p := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		if _, err := prov.Rotate(ctx, p, 48*time.Hour, time.Hour); err != nil {
			t.Fatalf("Rotate %s: %v", p, err)
		}
	}
	t.Setenv("FVPN_SIGNING_CONFIG_FILE", vaultDir)

	svc, err := usage.NewService(usage.Config{
		Pool:   pool,
		Signer: usage.NewProviderSigner(prov),
	})
	if err != nil {
		t.Fatalf("usage.NewService: %v", err)
	}

	adminID := uuid.New()
	memberA := uuid.New()
	deviceA := uuid.New()
	node1 := uuid.New()
	inc1 := uuid.New()
	node2 := uuid.New()
	inc2 := uuid.New()

	// Seed member, entitlement, device, device_binding, and two nodes.
	mustExec(t, pool, `INSERT INTO members(id, status) VALUES ($1, 'active')`, memberA)
	mustExec(t, pool, `
		INSERT INTO entitlements(member_id, revision, device_limit, valid_from, effective_service_end)
		VALUES ($1, 1, 5, '2026-01-01T00:00:00Z', '2027-01-01T00:00:00Z')
	`, memberA)
	mustExec(t, pool, `
		INSERT INTO member_devices(id, member_id, deployment_id, peer_id, status)
		VALUES ($1, $2, 'dep-http-1', 'peer-http-a', 'authorized')
	`, deviceA, memberA)
	mustExec(t, pool, `
		INSERT INTO device_bindings(
			device_id, member_id, overlay_scope, overlay_ip,
			binding_revision, valid_from, valid_until, peer_ref
		) VALUES ($1, $2, 'overlay-main', '100.64.0.10', 1, '2026-01-01T00:00:00Z', '2026-12-31T00:00:00Z', 'peer-http-a')
	`, deviceA, memberA)
	mustExec(t, pool, `
		INSERT INTO nodes(id, lifecycle_state, provisioning_state)
		VALUES ($1, 'registered', 'ready'), ($2, 'registered', 'ready')
	`, node1, node2)
	mustExec(t, pool, `
		INSERT INTO node_incarnations(id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch)
		VALUES ($1, $2, decode(repeat('33', 32), 'hex'), 'dep-http-1', 'node-peer-1', 'ubuntu', 'amd64'),
		       ($3, $4, decode(repeat('44', 32), 'hex'), 'dep-http-1', 'node-peer-2', 'ubuntu', 'arm64')
	`, inc1, node1, inc2, node2)
	mustExec(t, pool, `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`, node1, inc1)
	mustExec(t, pool, `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`, node2, inc2)

	handlers, err := control.Handlers(pool, nil)
	if err != nil {
		t.Fatalf("control.Handlers: %v", err)
	}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.BrowserRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.AgentRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}

	adminP := auth.Principal{
		Role:            auth.Admin,
		ID:              adminID,
		MFAVerified:     true,
		CSRFToken:       "TEST-ONLY-ADMIN-CSRF",
		AuthenticatedAt: time.Now().UTC(),
	}
	memP := auth.Principal{
		Role:            auth.Member,
		ID:              memberA,
		CSRFToken:       "TEST-ONLY-MEMBER-CSRF",
		AuthenticatedAt: time.Now().UTC(),
	}
	agentP := auth.Principal{
		Role:          auth.Agent,
		NodeID:        node1,
		IncarnationID: inc1,
	}

	commands := &httpapi.Commands{Pool: pool, Retention: time.Hour}
	pubOpts := httpapi.Options{
		Config: c,
		Sessions: auth.SessionAuthenticatorFunc(func(r *http.Request) (auth.Principal, error) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/member/") {
				return memP, nil
			}
			return adminP, nil
		}),
		Commands: commands,
	}
	pubAPI, err := httpapi.New(handlers, pubOpts)
	if err != nil {
		t.Fatalf("httpapi.New public: %v", err)
	}

	st := &usageHTTPStack{
		pool:    pool,
		prov:    prov,
		svc:     svc,
		public:  httpapi.Transport(nil, pubAPI),
		adminP:  adminP,
		memP:    memP,
		agentP:  agentP,
		memberA: memberA,
		deviceA: deviceA,
		node1:   node1,
		inc1:    inc1,
		node2:   node2,
		inc2:    inc2,
	}

	agentOpts := pubOpts
	agentOpts.AgentListener = true
	agentOpts.Agents = auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
		return st.agentP, nil
	})
	agentAPI, err := httpapi.New(handlers, agentOpts)
	if err != nil {
		t.Fatalf("httpapi.New agent: %v", err)
	}
	st.agent = httpapi.Transport(nil, agentAPI)

	disp, err := worker.New(
		pool,
		worker.Jobs(),
		worker.Config{
			Concurrency: 1,
			Attempts:    3,
			Lease:       time.Minute,
			JobTimeout:  5 * time.Second,
			Poll:        time.Millisecond,
			Backoff:     time.Millisecond,
			MaxBackoff:  time.Second,
		},
		prometheus.NewRegistry(),
		nil,
	)
	if err != nil {
		t.Fatalf("worker.New: %v", err)
	}
	st.disp = disp
	return st
}

func (s *usageHTTPStack) doAdmin(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		r.Header.Set("Idempotency-Key", uuid.NewString())
		r.Header.Set("Origin", "https://portal.example.invalid")
		r.Header.Set("X-CSRF-Token", s.adminP.CSRFToken)
	}
	w := httptest.NewRecorder()
	s.public.ServeHTTP(w, r)
	return w
}

func (s *usageHTTPStack) doMember(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	s.public.ServeHTTP(w, r)
	return w
}

func (s *usageHTTPStack) doAgent(t *testing.T, method, path, body, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if idemKey != "" {
		r.Header.Set("Idempotency-Key", idemKey)
	}
	w := httptest.NewRecorder()
	s.agent.ServeHTTP(w, r)
	return w
}

func TestAT14_AT15_AT16_AT17_AT19_HTTPUsageIngestBoundarySplitAndDrillDown(t *testing.T) {
	st := newUsageHTTPStack(t)
	ctx := context.Background()
	wCtx := worker.WithPool(ctx, st.pool)
	meterEpoch := uuid.New()
	id1 := usage.NodeIdentity{NodeID: st.node1, NodeIncarnationID: st.inc1}
	id2 := usage.NodeIdentity{NodeID: st.node2, NodeIncarnationID: st.inc2}
	base := time.Date(2026, 10, 31, 23, 40, 0, 0, time.UTC)

	// 1. Out-of-order and contiguous ACK over POST /api/v1/agent/usage-batches (AT15).
	ev1 := makeSignedEvent(t, id1, 1, meterEpoch, base, base.Add(5*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     1000,
			DownloadBytes:   2000,
		},
	})
	ev3 := makeSignedEvent(t, id1, 3, meterEpoch, base.Add(10*time.Minute), base.Add(15*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     500,
			DownloadBytes:   1500,
		},
	})

	idemBatch1 := uuid.NewString()
	req1Bytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev1, ev3},
	})
	w1 := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(req1Bytes), idemBatch1)
	if w1.Code != http.StatusOK {
		t.Fatalf("batch 1 status=%d body=%s", w1.Code, w1.Body.String())
	}
	var resp1 api.UsageBatchResult
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal resp1: %v", err)
	}
	if resp1.HighestContiguousSeq != decstr.U64(1) {
		t.Fatalf("expected highest_contiguous_seq=1 when seq=2 is missing, got %s", resp1.HighestContiguousSeq)
	}

	// Optional Idempotency-Key replay on POST /agent/usage-batches returns identical 200 response;
	// reusing the same Idempotency-Key with a different payload returns 409 IDEMPOTENCY_CONFLICT.
	w1Replay := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(req1Bytes), idemBatch1)
	if w1Replay.Code != http.StatusOK {
		t.Fatalf("batch 1 replay status=%d body=%s", w1Replay.Code, w1Replay.Body.String())
	}
	diffReqBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev1},
	})
	w1Conflict := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(diffReqBytes), idemBatch1)
	if w1Conflict.Code != http.StatusConflict || !strings.Contains(w1Conflict.Body.String(), "IDEMPOTENCY_CONFLICT") {
		t.Fatalf("expected 409 IDEMPOTENCY_CONFLICT, got status=%d body=%s", w1Conflict.Code, w1Conflict.Body.String())
	}

	// mTLS identity mismatch returns 403 INCARNATION_MISMATCH.
	badNodeReqBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node2),
		NodeIncarnationId: openapi_types.UUID(st.inc2),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev1},
	})
	wBadIdent := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(badNodeReqBytes), "")
	if wBadIdent.Code != http.StatusForbidden || !strings.Contains(wBadIdent.Body.String(), "INCARNATION_MISMATCH") {
		t.Fatalf("expected 403 INCARNATION_MISMATCH, got status=%d body=%s", wBadIdent.Code, wBadIdent.Body.String())
	}

	// Send duplicate ev1 + missing ev2: ev1 is duplicate, ev2 fills the gap and advances highest_contiguous_seq to 3.
	ev2 := makeSignedEvent(t, id1, 2, meterEpoch, base.Add(5*time.Minute), base.Add(10*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     2000,
			DownloadBytes:   3000,
		},
	})
	req2Bytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev1, ev2},
	})
	w2 := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(req2Bytes), "")
	if w2.Code != http.StatusOK {
		t.Fatalf("batch 2 status=%d body=%s", w2.Code, w2.Body.String())
	}
	var resp2 api.UsageBatchResult
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal resp2: %v", err)
	}
	if resp2.HighestContiguousSeq != decstr.U64(3) || resp2.Results[0].Status != api.UsageEventResultStatusDuplicate || resp2.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("unexpected resp2: %+v", resp2)
	}

	// Send conflicting seq=3 (different byte counts -> different hash): status=conflict (USAGE_EVENT_CONFLICT).
	ev3Conflict := makeSignedEvent(t, id1, 3, meterEpoch, base.Add(10*time.Minute), base.Add(15*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     9999,
			DownloadBytes:   9999,
		},
	})
	reqConflictBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev3Conflict},
	})
	wConflict := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(reqConflictBytes), "")
	if wConflict.Code != http.StatusOK {
		t.Fatalf("conflict batch status=%d body=%s", wConflict.Code, wConflict.Body.String())
	}
	var respConflict api.UsageBatchResult
	if err := json.Unmarshal(wConflict.Body.Bytes(), &respConflict); err != nil {
		t.Fatalf("unmarshal respConflict: %v", err)
	}
	if respConflict.Results[0].Status != api.UsageEventResultStatusConflict {
		t.Fatalf("expected status=conflict, got %+v", respConflict)
	}

	// 2. AT16/AT17 UTC month boundary split & rejection:
	// - Event 4 with corrupted payload_sha256 is rejected and sets blocked_seq=4.
	// - Event 4 crossing 2026-11-01T00:00:00Z ([23:55, 00:05)) is accepted and split 50/50 between 2026-10 and 2026-11.
	boundary := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	ev4BadHash := makeSignedEvent(t, id1, 4, meterEpoch, boundary.Add(-5*time.Minute), boundary, api.UsageQualityComplete, nil, nil)
	ev4BadHash.PayloadSha256 = "0000000000000000000000000000000000000000000000000000000000000000"

	reqBadBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev4BadHash},
	})
	wBad := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(reqBadBytes), "")
	var respBad api.UsageBatchResult
	_ = json.Unmarshal(wBad.Body.Bytes(), &respBad)
	blockedSeq, err := respBad.BlockedSeq.Get()
	if err != nil || blockedSeq != decstr.U64(4) || len(respBad.Results) != 1 || respBad.Results[0].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("expected corrupted-hash event to be rejected with blocked_seq=4, got %+v", respBad)
	}

	ev4Boundary := makeSignedEvent(t, id1, 4, meterEpoch, boundary.Add(-5*time.Minute), boundary.Add(5*time.Minute), api.UsageQualityBoundaryEstimated, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     800,
			DownloadBytes:   1200,
		},
	})
	ev5Nov := makeSignedEvent(t, id1, 5, meterEpoch, boundary.Add(5*time.Minute), boundary.Add(10*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-11",
			UploadBytes:     700,
			DownloadBytes:   800,
		},
	})

	reqSplitBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev4Boundary, ev5Nov},
	})
	wSplit := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(reqSplitBytes), "")
	if wSplit.Code != http.StatusOK {
		t.Fatalf("split batch status=%d body=%s", wSplit.Code, wSplit.Body.String())
	}

	// 3. AT17 multi-node aggregation + late previous-period event from node2 after 2026-11 has started.
	st.agentP = auth.Principal{Role: auth.Agent, NodeID: st.node2, IncarnationID: st.inc2}
	evNode2LateOct := makeSignedEvent(t, id2, 1, uuid.New(), boundary.Add(-10*time.Minute), boundary.Add(-5*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     1000,
			DownloadBytes:   1000,
		},
	})
	reqNode2Bytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node2),
		NodeIncarnationId: openapi_types.UUID(st.inc2),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{evNode2LateOct},
	})
	wNode2 := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(reqNode2Bytes), "")
	if wNode2.Code != http.StatusOK {
		t.Fatalf("node2 late batch status=%d body=%s", wNode2.Code, wNode2.Body.String())
	}
	st.agentP = auth.Principal{Role: auth.Agent, NodeID: st.node1, IncarnationID: st.inc1}

	// Drain outbox events.
	for i := 0; i < 10; i++ {
		worked, err := st.disp.Once(wCtx)
		if err != nil {
			t.Fatalf("worker Once[%d]: %v", i, err)
		}
		if !worked {
			break
		}
	}

	// 4. Verify GET /api/v1/member/usage for 2026-10 (month and day granularity) and 2026-11 over HTTP (AT14, AT16, AT17).
	// Expected 2026-10 total:
	// ev1 (3000) + ev2 (5000) + ev3 (2000) + ev4Oct (1000) + evNode2LateOct (2000) = 13000 bytes (upload 4900, download 8100).
	wMemOct := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-10&granularity=day")
	if wMemOct.Code != http.StatusOK {
		t.Fatalf("GET /member/usage 2026-10 status=%d body=%s", wMemOct.Code, wMemOct.Body.String())
	}
	var muOct api.MemberUsage
	if err := json.Unmarshal(wMemOct.Body.Bytes(), &muOct); err != nil {
		t.Fatalf("unmarshal muOct: %v", err)
	}
	if muOct.Totals.UploadBytes != decstr.U64(4900) || muOct.Totals.DownloadBytes != decstr.U64(8100) || muOct.Totals.TotalBytes != decstr.U64(13000) {
		t.Fatalf("unexpected 2026-10 totals: %+v", muOct.Totals)
	}
	if len(muOct.Days) == 0 {
		t.Fatalf("expected daily buckets for granularity=day")
	}

	wMemNov := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-11&granularity=month")
	if wMemNov.Code != http.StatusOK {
		t.Fatalf("GET /member/usage 2026-11 status=%d body=%s", wMemNov.Code, wMemNov.Body.String())
	}
	var muNov api.MemberUsage
	if err := json.Unmarshal(wMemNov.Body.Bytes(), &muNov); err != nil {
		t.Fatalf("unmarshal muNov: %v", err)
	}
	if muNov.Totals.TotalBytes != decstr.U64(2500) {
		t.Fatalf("expected 2026-11 total=2500 (1000 from boundary split + 1500 from ev5Nov), got %s", muNov.Totals.TotalBytes)
	}

	// 5. Admin drill-down and negative adjustment with audit + worker fup.evaluate job.
	drill, err := st.svc.GetMemberDrillDown(ctx, st.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberDrillDown: %v", err)
	}
	if len(drill.ByNode) != 2 || len(drill.ByDevice) != 1 || drill.Total.TotalBytes != 13000 {
		t.Fatalf("unexpected drill-down: nodes=%d devices=%d total=%d", len(drill.ByNode), len(drill.ByDevice), drill.Total.TotalBytes)
	}

	adj, err := st.svc.CreateAdjustment(ctx, usage.AdjustmentCreate{
		MemberID:      st.memberA,
		Period:        "2026-10",
		UploadBytes:   -900,
		DownloadBytes: -1100,
		Reason:        "Billing dispute credit",
		Actor:         st.adminP.ID.String(),
	})
	if err != nil || adj.ID == uuid.Nil {
		t.Fatalf("CreateAdjustment: adj=%+v err=%v", adj, err)
	}
	wMemAfterAdj := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-10")
	var muAfterAdj api.MemberUsage
	_ = json.Unmarshal(wMemAfterAdj.Body.Bytes(), &muAfterAdj)
	if muAfterAdj.Totals.TotalBytes != decstr.U64(11000) {
		t.Fatalf("expected 2026-10 total=11000 after -2000 adjustment, got %s", muAfterAdj.Totals.TotalBytes)
	}
}

func TestAT30_HTTPRecoveryFreezeEpochRotationReplayAndAuditPagination(t *testing.T) {
	st := newUsageHTTPStack(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	id1 := usage.NodeIdentity{NodeID: st.node1, NodeIncarnationID: st.inc1}
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)

	// Ingest events 1..3 before disaster.
	var initialEvents []api.UsageEvent
	for seq := uint64(1); seq <= 3; seq++ {
		ev := makeSignedEvent(t, id1, seq, meterEpoch, base.Add(time.Duration(seq-1)*time.Minute), base.Add(time.Duration(seq)*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
			{
				MemberId:        st.memberA,
				DeviceId:        st.deviceA,
				BindingRevision: 1,
				PeriodId:        "2026-10",
				UploadBytes:     1000,
				DownloadBytes:   1000,
			},
		})
		initialEvents = append(initialEvents, ev)
	}
	initBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            initialEvents,
	})
	wInit := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(initBytes), "")
	if wInit.Code != http.StatusOK {
		t.Fatalf("initial ingest status=%d body=%s", wInit.Code, wInit.Body.String())
	}
	var respInit api.UsageBatchResult
	_ = json.Unmarshal(wInit.Body.Bytes(), &respInit)
	oldEpoch := uuid.UUID(respInit.LedgerEpoch)

	// 1. Freeze ingest (§17.3, AT30): POST /api/v1/agent/usage-batches returns 503 and never ACKs.
	if err := st.svc.FreezeIngest(ctx, st.adminP.ID.String(), "AT30 backup restore freeze"); err != nil {
		t.Fatalf("FreezeIngest: %v", err)
	}
	wFrozen := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(initBytes), "")
	if wFrozen.Code != http.StatusServiceUnavailable || !strings.Contains(wFrozen.Body.String(), "DEPENDENCY_UNAVAILABLE") {
		t.Fatalf("expected 503 DEPENDENCY_UNAVAILABLE while frozen, got status=%d body=%s", wFrozen.Code, wFrozen.Body.String())
	}

	// 2. BeginRecovery rotates ledger_epoch and signs the recovery record with PurposeLedgerEpoch.
	recState, err := st.svc.BeginRecovery(ctx, usage.RecoveryRequest{
		RestoredTo:    base.Add(3 * time.Minute),
		AffectedSince: base,
		Reason:        "AT30 restore from backup",
		Actor:         st.adminP.ID.String(),
	})
	if err != nil {
		t.Fatalf("BeginRecovery: %v", err)
	}
	if recState.NewLedgerEpoch == oldEpoch {
		t.Fatalf("expected new ledger_epoch after BeginRecovery")
	}

	// Verify the recovery record signature envelope with pkg/envelope and PurposeLedgerEpoch.
	ts, err := st.prov.TrustBundle(ctx)
	if err != nil {
		t.Fatalf("TrustBundle: %v", err)
	}
	recEnvBytes, err := recState.SignedRecord.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical recovery envelope: %v", err)
	}
	if _, _, err := envelope.Verify(recEnvBytes, ts, envelope.PurposeLedgerEpoch, time.Now().UTC()); err != nil {
		t.Fatalf("envelope.Verify PurposeLedgerEpoch failed: %v", err)
	}

	// 3. POST /api/v1/agent/usage-batches with stale oldEpoch returns 409 LEDGER_EPOCH_MISMATCH.
	staleBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullableWithValue(openapi_types.UUID(oldEpoch)),
		Events:            initialEvents,
	})
	wStale := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(staleBytes), "")
	if wStale.Code != http.StatusConflict || !strings.Contains(wStale.Body.String(), "LEDGER_EPOCH_MISMATCH") {
		t.Fatalf("expected 409 LEDGER_EPOCH_MISMATCH, got status=%d body=%s", wStale.Code, wStale.Body.String())
	}

	// 4. GET /api/v1/agent/usage-checkpoint returns 200 and verifies with pkg/envelope (PurposeRecovery).
	wCp := st.doAgent(t, http.MethodGet, "/api/v1/agent/usage-checkpoint", "", "")
	if wCp.Code != http.StatusOK {
		t.Fatalf("GET /agent/usage-checkpoint status=%d body=%s", wCp.Code, wCp.Body.String())
	}
	verifiedCpBytes, _, err := envelope.Verify(wCp.Body.Bytes(), ts, envelope.PurposeRecovery, time.Now().UTC())
	if err != nil {
		t.Fatalf("envelope.Verify PurposeRecovery on checkpoint failed: %v", err)
	}
	var cp api.UsageCheckpoint
	if err := json.Unmarshal(verifiedCpBytes, &cp); err != nil {
		t.Fatalf("unmarshal verified checkpoint: %v", err)
	}
	if uuid.UUID(cp.LedgerEpoch) != recState.NewLedgerEpoch || cp.HighestContiguousSeq != decstr.U64(3) || cp.RecoveryRecordId.IsNull() {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}

	// 5. Replay events 1..3 + replay_unavailable [4, 5] + new event 6 over POST /api/v1/agent/usage-batches.
	ev6 := makeSignedEvent(t, id1, 6, meterEpoch, base.Add(5*time.Minute), base.Add(6*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        st.memberA,
			DeviceId:        st.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     2000,
			DownloadBytes:   2000,
		},
	})
	replayAvail := []api.ReplayUnavailableRange{
		{
			StartSeq: decstr.U64(4),
			EndSeq:   decstr.U64(5),
			Reason:   api.ReplayUnavailableRangeReasonRetentionExpired,
		},
	}
	replayEvents := append(append([]api.UsageEvent{}, initialEvents...), ev6)
	replayBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(st.node1),
		NodeIncarnationId: openapi_types.UUID(st.inc1),
		LedgerEpoch:       presence.NewNullableWithValue(openapi_types.UUID(recState.NewLedgerEpoch)),
		Events:            replayEvents,
		ReplayUnavailable: &replayAvail,
	})
	wReplay := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(replayBytes), "")
	if wReplay.Code != http.StatusOK {
		t.Fatalf("replay batch status=%d body=%s", wReplay.Code, wReplay.Body.String())
	}
	var respReplay api.UsageBatchResult
	if err := json.Unmarshal(wReplay.Body.Bytes(), &respReplay); err != nil {
		t.Fatalf("unmarshal respReplay: %v", err)
	}
	if respReplay.HighestContiguousSeq != decstr.U64(6) {
		t.Fatalf("expected highest_contiguous_seq=6 after replay + replay_unavailable [4,5] + seq 6, got %s", respReplay.HighestContiguousSeq)
	}

	// Complete recovery and verify GET /api/v1/member/usage shows total=10000 (no double counting) and gap=true.
	if err := st.svc.CompleteRecovery(ctx, st.adminP.ID.String()); err != nil {
		t.Fatalf("CompleteRecovery: %v", err)
	}
	wMem := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-10")
	if wMem.Code != http.StatusOK {
		t.Fatalf("GET /member/usage status=%d body=%s", wMem.Code, wMem.Body.String())
	}
	var mu api.MemberUsage
	if err := json.Unmarshal(wMem.Body.Bytes(), &mu); err != nil {
		t.Fatalf("unmarshal MemberUsage: %v", err)
	}
	if mu.Totals.TotalBytes != decstr.U64(10000) || !mu.Quality.Gap || mu.Quality.ReconciliationInProgress {
		t.Fatalf("expected total=10000 gap=true reconciliation_in_progress=false, got totals=%+v quality=%+v", mu.Totals, mu.Quality)
	}

	// 6. Audit pagination with one malformed row and one purge_retained row over GET /api/v1/admin/audit-events (§14.8).
	mustExec(t, st.pool, `
		INSERT INTO audit_events (actor, actor_type, action, resource, result, schema_version)
		VALUES ('not-a-uuid', 'admin', 'usage.corrupt_test', 'usage_ledger_state:00000000-0000-0000-0000-000000000130', 'succeeded', 1)
	`)
	mustExec(t, st.pool, `SELECT public.purge_retained('audit_events', '2000-01-01T00:00:00Z'::timestamptz, 'scheduled retention purge')`)

	var allAudit []api.AuditEvent
	cursor := ""
	for page := 0; page < 20; page++ {
		path := "/api/v1/admin/audit-events?limit=2"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		wAudit := st.doAdmin(t, http.MethodGet, path, "")
		if wAudit.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, wAudit.Code, wAudit.Body.String())
		}
		var list api.AuditEventList
		if err := json.Unmarshal(wAudit.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal AuditEventList: %v", err)
		}
		allAudit = append(allAudit, list.Items...)
		next, err := list.NextCursor.Get()
		if err != nil || next == "" {
			break
		}
		cursor = next
	}

	foundPurge := false
	foundBeginRecovery := false
	for _, item := range allAudit {
		if item.Action == "usage.corrupt_test" {
			t.Fatalf("malformed audit row must be skipped, never returned")
		}
		if item.Action == "retention.purge" {
			foundPurge = true
			if item.Resource.Type != "fleet_config" || uuid.UUID(item.Resource.Id).String() != "00000000-0000-0000-0000-000000000001" {
				t.Fatalf("unexpected retention.purge resource: %+v", item.Resource)
			}
			if !item.Actor.Id.IsNull() {
				t.Fatalf("expected retention.purge system role actor.id to be null")
			}
		}
		if item.Action == "usage_ledger.begin_recovery" {
			foundBeginRecovery = true
		}
	}
	if !foundPurge || !foundBeginRecovery {
		t.Fatalf("expected retention.purge (%v) and usage_ledger.begin_recovery (%v) in paginated audit events", foundPurge, foundBeginRecovery)
	}
}
