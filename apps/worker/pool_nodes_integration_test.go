//go:build integration

package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	policyStore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func claimBudgetEvent(t *testing.T, pool *pgxpool.Pool, event db.ClaimedEvent) db.ClaimedEvent {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `UPDATE outbox_events SET available_at=CASE WHEN id=$1 THEN clock_timestamp() ELSE clock_timestamp()+interval '1 day' END`, event.ID); err != nil {
		t.Fatal(err)
	}
	claims, err := db.ClaimOutbox(t.Context(), pool, "TEST ONLY budget", 1, time.Minute)
	if err != nil || len(claims) != 1 || claims[0].ID != event.ID {
		t.Fatalf("claim expected event: count=%d error=%v", len(claims), err)
	}
	event.Worker = claims[0].Worker
	event.LeaseToken = claims[0].LeaseToken
	event.Attempt = claims[0].Attempt
	return event
}
func seedBudgetEvent(t *testing.T, pool *pgxpool.Pool, kind string, aggregate uuid.UUID, payload any) db.ClaimedEvent {
	t.Helper()
	event := db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), AggregateID: aggregate, AggregateType: "node", AggregateVersion: "1", EventType: kind, SchemaVersion: 1, Payload: budgetJSON(t, payload)}}
	if err := db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error { return db.EnqueueOutbox(t.Context(), tx, event.OutboxEvent) }); err != nil {
		t.Fatal(err)
	}
	return event
}

type budgetFetcher struct{}

func (budgetFetcher) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("TEST ONLY artifact")), nil
}

// Each factory populates state before installing the measured pool. The return
// function invokes the real registration, with a durable outbox claim.
func nodeBudgetCase(t *testing.T, kind string, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
	t.Helper()
	base, enrolled, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	fake := &discoveryManagement{groups: map[string]map[string]any{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/peers" {
			fake.mu.Lock()
			defer fake.mu.Unlock()
			peers := []map[string]any{}
			if !fake.peerDeleted {
				peers = append(peers, map[string]any{"id": "TEST-peer", "user_id": "", "ephemeral": false, "ip": "100.64.0.42"})
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(budgetJSON(t, peers))
			return
		}
		fake.ServeHTTP(w, req)
	}))
	t.Cleanup(server.Close)
	makeClient := func(pool *pgxpool.Pool) *netbird.Client {
		client, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: base.Config.NetBirdDeployment, Token: observationToken{}, Bindings: nodelib.NetBirdBindings{Pool: pool}, AllowInsecureHTTP: true})
		if err != nil {
			t.Fatal(err)
		}
		return client
	}
	jobs := NodeJobs(base, makeClient(base.Pool), budgetFetcher{}, 3)
	var event db.ClaimedEvent
	var serviceNow time.Time
	switch kind {
	case "node.provision.requested":
		event = loadLatestOutbox(t, t.Context(), base.Pool, kind)
	case "node.peer.discover", "node.peer.observe", "node.publication_changed":
		provision := claimBudgetEvent(t, base.Pool, loadLatestOutbox(t, t.Context(), base.Pool, "node.provision.requested"))
		if _, err := jobs[provision.EventType](t.Context(), provision); err != nil {
			t.Fatal(err)
		}
		if err := db.CompleteOutbox(t.Context(), base.Pool, provision); err != nil {
			t.Fatal(err)
		}
		fake.useKey("TEST-peer")
		// Management reads return peer objects; the discovery-only fixture also
		// accepts string IDs, but publication decodes the complete typed state.
		fake.mu.Lock()
		for _, group := range fake.groups {
			group["peers"] = []any{map[string]any{"id": "TEST-peer"}}
		}
		fake.mu.Unlock()
		if kind == "node.peer.discover" {
			event = loadLatestOutbox(t, t.Context(), base.Pool, kind)
			break
		}
		if kind == "node.peer.observe" {
			report.Netbird.PeerId.Set("TEST-peer")
			if _, err := base.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
				t.Fatal(err)
			}
			event = loadLatestOutbox(t, t.Context(), base.Pool, kind)
			break
		}
		discover := claimBudgetEvent(t, base.Pool, loadLatestOutbox(t, t.Context(), base.Pool, "node.peer.discover"))
		if _, err := jobs[discover.EventType](t.Context(), discover); err != nil {
			t.Fatal(err)
		}
		if err := db.CompleteOutbox(t.Context(), base.Pool, discover); err != nil {
			t.Fatal(err)
		}
		detail, _, err := base.Nodes.GetNode(t.Context(), report.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = base.Enrollment.RevokeNode(t.Context(), enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, report.NodeId, &detail.ResourceVersion, "TEST ONLY pool budget"); err != nil {
			t.Fatal(err)
		}
		event = loadLatestOutbox(t, t.Context(), base.Pool, kind)
	case "release.verify":
		release := releasetest.New(t)
		hash := sha256.Sum256([]byte("TEST ONLY artifact"))
		for i := range release.Manifest.Artifacts {
			release.Manifest.Artifacts[i].SizeBytes = decstr.U64(len("TEST ONLY artifact"))
			release.Manifest.Artifacts[i].Sha256 = hex.EncodeToString(hash[:])
		}
		base.Releases = &releases.Service{Pool: base.Pool, Validator: release.Validator, VerifyArtifacts: true}
		if _, err := base.Releases.Register(t.Context(), releasetest.Command(), release.Request(t, release.Manifest)); err != nil {
			t.Fatal(err)
		}
		event = loadLatestOutbox(t, t.Context(), base.Pool, kind)
	case "node.resolver_documents.reconcile":
		seedBudgetFleet(t, base)
		seedBudgetResolver(t, base)
		event = seedBudgetEvent(t, base.Pool, kind, enrolled.Response.NodeId, map[string]int{"schema_version": 1})
	case "node.staleness.sweep":
		if _, err := base.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
			t.Fatal(err)
		}
		serviceNow = time.Now().Add(time.Duration(base.Config.OnlineSeconds+1) * time.Second)
		event = seedBudgetEvent(t, base.Pool, kind, enrolled.Response.NodeId, map[string]int{"schema_version": 1})
	case nodelib.EventGenerationHoldCheck:
		serviceNow = seedBudgetHold(t, base, report)
		event = loadLatestOutbox(t, t.Context(), base.Pool, kind)
	default:
		seedBudgetFleet(t, base)
		serviceNow = seedBudgetHold(t, base, report)
		event = seedBudgetEvent(t, base.Pool, kind, enrolled.Response.NodeId, map[string]int{"schema_version": 1})
	}
	event = claimBudgetEvent(t, base.Pool, event)
	tr := &budgetTrace{}
	pool := budgetPool(t, base.Pool, max(1, size), tr)
	runtime, err := httpwire.Open(t.Context(), pool, base.Config, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The runtime and its fleet service share this release service pointer.
	// Preserve the ephemeral fixture's trust while moving all work to this pool.
	runtime.Releases.Validator = base.Releases.Validator
	if !serviceNow.IsZero() {
		nodelib.WithClock(func() time.Time { return serviceNow })(runtime.Nodes)
	}
	handler := NodeJobs(runtime, makeClient(pool), budgetFetcher{}, 3)[kind]
	return pool, func(ctx context.Context) (string, error) { return handler(ctx, event) }, tr
}

func runBudgetPair(t *testing.T, name string, peak int, factory func(*testing.T, int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace)) {
	t.Helper()
	for _, size := range []int{peak, peak - 1} {
		t.Run(fmt.Sprintf("connections_%d", size), func(t *testing.T) {
			pool, run, tr := factory(t, size)
			if size == 0 {
				conn, err := pool.Acquire(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Release()
			}
			tr.high.Store(0)
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			start := time.Now()
			receipt, err := run(ctx)
			t.Logf("%s peak=%d available=%d observed=%d elapsed=%s receipt=%t error=%v", name, peak, size, tr.high.Load(), time.Since(start), receipt != "", err)
			if size == peak && (err != nil || receipt == "") {
				t.Fatalf("declared peak cannot complete: %v", err)
			}
			if size < peak && err != nil && ctx.Err() == nil {
				t.Fatalf("smaller pool failed without deadline: %v", err)
			}
		})
	}
}
func TestAPI01_NodePoolPeaks(t *testing.T) {
	catalog := nodes().Describe().Jobs
	names := []string{}
	for name := range catalog {
		names = append(names, name)
	}
	// Stable output makes changes to the peak table straightforward to review.
	slices.Sort(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			runBudgetPair(t, name, catalog[name].Peak, func(t *testing.T, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
				return nodeBudgetCase(t, name, size)
			})
		})
	}
}

func seedBudgetResolver(t *testing.T, r *httpwire.Runtime) {
	t.Helper()
	store := policyStore.NewStore(r.Pool)
	code := "test-resolver"
	actor := uuid.New()
	revision, err := store.CreateRevision(t.Context(), policyStore.CreateRevisionInput{Code: &code, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1, Payload: budgetJSON(t, api.DnsProfilePayload{ProfileCode: code, DisplayName: api.LocalizedText{ZhTW: "測試"}, DomainRuleSetRevisionIds: []uuid.UUID{}, ThreatFeedRevisionIds: []uuid.UUID{}, BlockResponse: api.DnsProfilePayloadBlockResponseNxdomain, BlockedTtlSeconds: 60}), Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateRevision(t.Context(), policyStore.ActivateRevisionInput{DefinitionID: revision.DefinitionID, RevisionID: revision.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	publisher, err := dnsstore.NewPublisher(r.Pool, dnsstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publisher.Handle(t.Context(), loadLatestOutbox(t, t.Context(), r.Pool, "dns.publish")); err != nil {
		t.Fatal(err)
	}
}

func seedBudgetFleet(t *testing.T, r *httpwire.Runtime) {
	t.Helper()
	release := releasetest.New(t)
	service := &releases.Service{Pool: r.Pool, Validator: release.Validator}
	r.Releases.Validator = release.Validator
	if _, err := service.Register(t.Context(), releasetest.Command(), release.Request(t, release.Manifest)); err != nil {
		t.Fatal(err)
	}
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{release.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	store := policyStore.NewStore(r.Pool, policyStore.WithReleaseService(service))
	code, actor := "test-support", uuid.New()
	revision, err := store.CreateRevision(t.Context(), policyStore.CreateRevisionInput{Code: &code, Type: api.PolicyTypeReleaseSupport, SchemaVersion: 1, Payload: budgetJSON(t, support), Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateRevision(t.Context(), policyStore.ActivateRevisionInput{DefinitionID: revision.DefinitionID, RevisionID: revision.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	fleet, err := fleetupdate.New(r.Pool, service, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	var generation decstr.U64
	if err = r.Pool.QueryRow(t.Context(), `SELECT generation::text FROM fleet_config`).Scan(&generation); err != nil {
		t.Fatal(err)
	}
	if _, err = fleet.CreateBatch(t.Context(), releasetest.Command(), api.UpdateBatchCreateRequest{ExpectedGeneration: generation, ReleaseId: release.Manifest.ReleaseId, Reason: "TEST ONLY", Scope: "all_nodes", SetFleetDefault: true}); err != nil {
		t.Fatal(err)
	}
}

func seedBudgetHold(t *testing.T, r *httpwire.Runtime, report api.ObservationReport) time.Time {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	nodelib.WithClock(func() time.Time { return now })(r.Nodes)
	generation, _ := report.AppliedConfigGeneration.Get()
	leaseID := uuid.New()
	raw := budgetJSON(t, map[string]any{"signed": api.ServingLease{LeaseId: leaseID, NodeId: report.NodeId, NodeIncarnationId: report.NodeIncarnationId, ConfigGeneration: generation}})
	if _, err := r.Pool.Exec(t.Context(), `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash,envelope) VALUES($1,$2,$3,$4::text::numeric,$5,$6,'TEST ONLY',decode(repeat('00',32),'hex'),$7)`, leaseID, report.NodeId, report.NodeIncarnationId, generation.String(), now, now.Add(time.Hour), raw); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Pool.Exec(t.Context(), `UPDATE nodes SET desired_config_generation=desired_config_generation+1,resource_version=resource_version+1 WHERE id=$1`, report.NodeId); err != nil {
		t.Fatal(err)
	}
	report.AppliedConfigGeneration.Set(generation + 1)
	report.Lease.LeaseId.Set(leaseID)
	report.Lease.ExpiresAt.Set(now.Add(time.Hour))
	report.Errors = nil
	report.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonPolicyMissing}
	for i := range report.Checks {
		if report.Checks[i].Name == api.NodeCheckNameLease {
			report.Checks[i].Status = api.HealthCheckStatusFailed
			report.Checks[i].ErrorCode.Set(api.ErrorCodeSTALEGENERATION)
		}
	}
	if _, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	return now.Add(31 * time.Second)
}
