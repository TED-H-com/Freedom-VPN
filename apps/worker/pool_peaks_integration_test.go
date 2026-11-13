//go:build integration

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TEST ONLY management endpoint. SQL bindings and all adapter/session locks are
// real. Only the remote management HTTP API is replaced.
type budgetManagement struct {
	mu      sync.Mutex
	buckets map[string]map[string]map[string]any
}

func newBudgetManagement(t *testing.T) (*budgetManagement, *httptest.Server) {
	t.Helper()
	m := &budgetManagement{buckets: map[string]map[string]map[string]any{}}
	for _, kind := range []string{"users", "peers", "groups", "routes", "policies", "dns/nameservers"} {
		m.buckets[kind] = map[string]map[string]any{}
	}
	server := httptest.NewServer(m)
	t.Cleanup(server.Close)
	return m, server
}
func (m *budgetManagement) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	kind, id, _ := strings.Cut(path, "/")
	if kind == "dns" {
		kind = path
		id = ""
	}
	bucket, ok := m.buckets[kind]
	if !ok {
		w.WriteHeader(404)
		return
	}
	respond := func(v any) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
	if r.Method == "GET" {
		if id != "" {
			if bucket[id] == nil {
				w.WriteHeader(404)
				return
			}
			respond(bucket[id])
			return
		}
		out := []map[string]any{}
		for _, v := range bucket {
			out = append(out, v)
		}
		respond(out)
		return
	}
	if r.Method == "DELETE" {
		delete(bucket, id)
		respond(map[string]any{})
		return
	}
	var obj map[string]any
	if json.NewDecoder(r.Body).Decode(&obj) != nil {
		w.WriteHeader(400)
		return
	}
	if id == "" {
		id = "TEST-" + uuid.NewString()
	}
	obj["id"] = id
	if kind == "groups" {
		peers := []map[string]any{}
		if ids, ok := obj["peers"].([]any); ok {
			for _, p := range ids {
				peers = append(peers, map[string]any{"id": p})
			}
		}
		obj["peers"] = peers
		obj["peers_count"] = len(peers)
	}
	bucket[id] = obj
	respond(obj)
}

// The barrier is reached after the member try-lock query has completed. Each
// event then owns capture + member sessions before any device sync proceeds.
type budgetTrace struct {
	pool         *pgxpool.Pool
	high         atomic.Int32
	barrier      chan struct{}
	participants int32
	arrived      atomic.Int32
	enabled      atomic.Bool
}
type budgetTraceKey struct{}

func (tr *budgetTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if tr.pool != nil {
		n := tr.pool.Stat().AcquiredConns()
		for old := tr.high.Load(); n > old; old = tr.high.Load() {
			if tr.high.CompareAndSwap(old, n) {
				break
			}
		}
	}
	if tr.enabled.Load() && strings.Contains(data.SQL, "pg_try_advisory_lock") && len(data.Args) > 0 {
		if key, ok := data.Args[0].(string); ok && strings.HasPrefix(key, "member:") {
			return context.WithValue(ctx, budgetTraceKey{}, true)
		}
	}
	return ctx
}
func (tr *budgetTrace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if ctx.Value(budgetTraceKey{}) != true || data.Err != nil {
		return
	}
	if tr.arrived.Add(1) == tr.participants {
		close(tr.barrier)
	}
	select {
	case <-tr.barrier:
	case <-ctx.Done():
	}
}
func budgetPool(t *testing.T, base *pgxpool.Pool, size int, tr *budgetTrace) *pgxpool.Pool {
	t.Helper()
	cfg := base.Config()
	cfg.MaxConns = int32(max(1, size))
	cfg.MinConns = 0
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "5000"
	if tr != nil {
		cfg.ConnConfig.Tracer = tr
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal("budget pool construction failed")
	}
	t.Cleanup(pool.Close)
	if tr != nil {
		tr.pool = pool
	}
	// pgx forbids a zero-size pool. Holding its sole connection models zero
	// available connections, without silently falling back to pgx's default.
	if size == 0 {
		conn, err := pool.Acquire(t.Context())
		if err != nil {
			t.Fatal("zero budget reservation failed")
		}
		t.Cleanup(conn.Release)
	}
	return pool
}
func budgetJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func memberBudgetWorld(t *testing.T, count, deviceCount, size int, tr *budgetTrace, discovery ...bool) (*memberPolicyFixture, []db.ClaimedEvent) {
	t.Helper()
	f := newMemberPolicyFixture(t, true)
	management, server := newBudgetManagement(t)
	setup := f.pool
	var revision uuid.UUID
	if err := setup.QueryRow(t.Context(), `SELECT plan_revision_id FROM entitlements WHERE member_id=$1`, f.member).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{f.member}
	for i := 1; i < count; i++ {
		member, identity := uuid.New(), uuid.New()
		user := fmt.Sprintf("TEST-user-%d", i)
		f.sql(`INSERT INTO members(id,status) VALUES($1,'active')`, member)
		f.sql(`INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,'https://id.example.invalid','portal',$3,'sub','oidc',clock_timestamp())`, identity, member, user)
		f.sql(`INSERT INTO netbird_user_bindings(id,member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,$2,$3,$4,$5,'shared_issuer_public_subject:https://id.example.invalid',1,clock_timestamp())`, uuid.New(), member, f.nodes.NetBirdDeployment, user, identity)
		start := time.Now().Add(-time.Hour)
		if _, err := f.services.HTTP.Members.Grant(t.Context(), memberPolicyCommand("admin", uuid.New()), member, api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: time.Now().Add(12 * time.Hour), Reason: "TEST ONLY pool budget", PlanRevisionId: &revision}); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, member)
	}
	for i, member := range ids {
		user := fmt.Sprintf("TEST-user-%d", i)
		if i == 0 {
			user = f.peer.peer.ExternalUserID
		}
		management.buckets["users"][user] = map[string]any{"id": user, "auto_groups": []any{}, "is_service_user": false, "is_blocked": false}
		for j := range deviceCount {
			peer := fmt.Sprintf("TEST-peer-%d-%d", i, j)
			ip := fmt.Sprintf("100.64.0.%d", 2+i*deviceCount+j)
			management.buckets["peers"][peer] = map[string]any{"id": peer, "user_id": user, "ephemeral": false, "ip": ip}
			if len(discovery) > 0 && discovery[0] && j == deviceCount-1 {
				continue
			}
			p := devices.Peer{DeploymentID: f.nodes.NetBirdDeployment, ID: peer, ExternalUserID: user, OverlayScope: "TEST ONLY", OverlayIP: netip.MustParseAddr(ip), Platform: "linux"}
			if _, err := f.services.HTTP.Devices.AuthorizeDevice(t.Context(), memberPolicyCommand("system", member), p); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Use the complete services on the measured pool, including the actual signer
	// and compiler. Fixture writes never consume the measured connection budget.
	pool := budgetPool(t, setup, size, tr)
	f.pool = pool
	f.services.HTTP.Members.Pool = pool
	f.services.HTTP.Devices.Pool = pool
	f.services.HTTP.Billing.Pool = pool
	bindings := devices.NetBirdBindings{Pool: pool}
	client, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: f.nodes.NetBirdDeployment, Token: observationToken{}, Bindings: bindings, HTTPClient: server.Client(), AllowInsecureHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := devices.NewIdentityDiscovery("https://id.example.invalid", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &devices.NetBirdAdapter{Pool: pool, Client: client, Bindings: bindings, DeploymentID: f.nodes.NetBirdDeployment, OverlayScope: "TEST ONLY", Identity: identity}
	f.services.Adapter = adapter
	f.services.HTTP.Devices.Adapter = adapter
	runtime, err := httpwire.Open(t.Context(), pool, f.nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.services.Compiler = runtime.DesiredStateCompiler()
	events := []db.ClaimedEvent{}
	for _, member := range ids {
		var event db.ClaimedEvent
		if err := setup.QueryRow(t.Context(), `SELECT id,aggregate_id,aggregate_type,aggregate_version::text,event_type,schema_version,payload FROM outbox_events WHERE event_type='member.entitlement_changed' AND aggregate_id=$1 ORDER BY created_at DESC LIMIT 1`, member).Scan(&event.ID, &event.AggregateID, &event.AggregateType, &event.AggregateVersion, &event.EventType, &event.SchemaVersion, &event.Payload); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return f, events
}

func memberPeakEvents() []string {
	return []string{"member.entitlement_changed", "member.grant_revoked", "member.suspended", "device.authorized", "device.revoked", "netbird.device_sync", "members.reconcile", "devices.discover"}
}

func TestAT02_MemberPoolPeaks(t *testing.T) {
	for _, kind := range memberPeakEvents() {
		t.Run(kind, func(t *testing.T) {
			peak := memberJobs()[kind].Peak
			for _, size := range []int{peak, peak - 1} {
				t.Run(fmt.Sprintf("connections_%d", size), func(t *testing.T) {
					tr := &budgetTrace{}
					// Discovery has a one-connection budget; defer the zero-budget reservation
					// until all configuration constructors have completed.
					f, events := memberBudgetWorld(t, 1, 3, max(1, size), tr, kind == "devices.discover")
					event := events[0]
					event.EventType = kind
					if strings.HasPrefix(kind, "device.") || kind == "netbird.device_sync" {
						event = loadLatestOutbox(t, t.Context(), f.pool, "device.authorized")
						event.EventType = kind
					}
					if kind == "members.reconcile" {
						event.Payload = budgetJSON(t, scheduledPage{Run: uuid.New()})
					}
					event = claimBudgetEvent(t, f.pool, event)
					if size == 0 {
						conn, err := f.pool.Acquire(t.Context())
						if err != nil {
							t.Fatal(err)
						}
						defer conn.Release()
					}
					ctx, cancel := context.WithTimeout(WithMembers(t.Context(), f.services), 2*time.Second)
					defer cancel()
					tr.high.Store(0)
					start := time.Now()
					receipt, err := MemberJobs()[kind](ctx, event)
					elapsed := time.Since(start)
					t.Logf("peak=%d available=%d observed=%d elapsed=%s receipt=%t error=%v", peak, size, tr.high.Load(), elapsed, receipt != "", err)
					if size == peak && (err != nil || receipt == "") {
						t.Fatalf("declared peak cannot complete: %v", err)
					}
					if size < peak && err != nil && ctx.Err() == nil {
						t.Fatalf("smaller pool failed without a deadline: %v", err)
					}
				})
			}
		})
	}
}

func TestAT02_MemberPoolStallSchedule(t *testing.T) {
	derived, err := PoolSize(4)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{int(derived), 10} {
		t.Run(fmt.Sprintf("connections_%d", size), func(t *testing.T) {
			tr := &budgetTrace{barrier: make(chan struct{}), participants: 4}
			f, events := memberBudgetWorld(t, 4, 1, size, tr)
			ids := []uuid.UUID{}
			for _, e := range events {
				ids = append(ids, e.ID)
			}
			f.sql(`UPDATE outbox_events SET available_at=CASE WHEN id=ANY($1) THEN clock_timestamp() ELSE clock_timestamp()+interval '1 day' END`, ids)
			var mu sync.Mutex
			errorsSeen := []string{}
			jobs := MemberJobs()
			original := jobs["member.entitlement_changed"]
			jobs["member.entitlement_changed"] = func(ctx context.Context, event db.ClaimedEvent) (string, error) {
				start := time.Now()
				receipt, err := original(ctx, event)
				mu.Lock()
				defer mu.Unlock()
				t.Logf("pool=%d member_event attempt=%d elapsed=%s error=%v", size, event.Attempt, time.Since(start), err)
				if err != nil {
					errorsSeen = append(errorsSeen, err.Error())
				}
				return receipt, err
			}
			d, err := New(f.pool, jobs, Config{Concurrency: 4, Attempts: 3, Lease: time.Minute, JobTimeout: 2 * time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Second}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithMembers(t.Context(), f.services)
			tr.enabled.Store(true)
			done := make(chan error, 4)
			for range 4 {
				go func() { _, err := d.Once(ctx); done <- err }()
			}
			for range 4 {
				if err := <-done; err != nil {
					t.Fatal(err)
				}
			}
			tr.enabled.Store(false)
			var delivered, retries int
			if err := f.pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE delivered_at IS NOT NULL),count(*) FILTER(WHERE delivered_at IS NULL AND attempt=1 AND lease_owner IS NULL) FROM outbox_events WHERE id=ANY($1)`, ids).Scan(&delivered, &retries); err != nil {
				t.Fatal(err)
			}
			t.Logf("pool=%d first_wave delivered=%d retries_scheduled=%d handler_errors=%d observed_connections=%d synchronized_members=%d", size, delivered, retries, len(errorsSeen), tr.high.Load(), tr.arrived.Load())
			if size == int(derived) && (delivered != 4 || retries != 0 || len(errorsSeen) != 0 || tr.arrived.Load() != 4) {
				t.Fatal("derived pool did not complete the concurrent schedule")
			}
			// The 10-connection case is observational. Recover any retries sequentially
			// to record the persisted attempts, without assuming that it must stall.
			if retries > 0 {
				f.sql(`UPDATE outbox_events SET available_at=clock_timestamp() WHERE id=ANY($1) AND delivered_at IS NULL`, ids)
				for range retries {
					if _, err := d.Once(ctx); err != nil {
						t.Fatal(err)
					}
				}
			}
			rows, err := f.pool.Query(ctx, `SELECT attempt,delivered_at IS NOT NULL FROM outbox_events WHERE id=ANY($1) ORDER BY id`, ids)
			if err != nil {
				t.Fatal(err)
			}
			attempts := []int{}
			for rows.Next() {
				var attempt int
				var complete bool
				if err := rows.Scan(&attempt, &complete); err != nil {
					t.Fatal(err)
				}
				attempts = append(attempts, attempt)
				if !complete {
					t.Error("event still pending after recovery")
				}
			}
			rows.Close()
			if rows.Err() != nil {
				t.Fatal(rows.Err())
			}
			slices.Sort(attempts)
			t.Logf("pool=%d recovered_attempts=%v", size, attempts)
		})
	}
}
