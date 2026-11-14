//go:build integration

package worker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/dnsfeeds"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	policyStore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	usagepkg "github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func billingBudgetCase(t *testing.T, kind string, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
	t.Helper()
	f := newMemberPolicyFixture(t, true)
	// A separate member avoids overlapping an admin grant on another plan.
	f.member = uuid.New()
	f.sql(`INSERT INTO members(id) VALUES($1)`, f.member)
	now := time.Now().UTC().Truncate(time.Second)
	provider := billing.NewFakeProvider()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	vault, err := billing.NewVault("TEST ONLY", map[string][]byte{"TEST ONLY": key})
	clear(key)
	if err != nil {
		t.Fatal(err)
	}
	// A paid subscription exercises provider reads, ledger inserts, entitlement
	// recomputation and audit writes, all sequentially on one connection.
	plan, revision, sub := uuid.New(), uuid.New(), uuid.New()
	f.sql(`INSERT INTO plans(id,code) VALUES($1,'TEST ONLY billing')`, plan)
	var groups []uuid.UUID
	if err := f.pool.QueryRow(t.Context(), `SELECT exit_group_ids FROM plan_revisions LIMIT 1`).Scan(&groups); err != nil {
		t.Fatal(err)
	}
	f.sql(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,provider,provider_price_ref,schema_version,payload) VALUES($1,$2,1,3,$3,'fake','TEST-price',1,'{}')`, revision, plan, groups)
	f.sql(`INSERT INTO billing_customers(id,member_id,provider,external_customer_id) VALUES($1,$2,'fake','TEST-customer')`, uuid.New(), f.member)
	f.sql(`INSERT INTO subscriptions(id,member_id,provider,external_subscription_id,external_customer_id,plan_revision_id,provider_status,is_active) VALUES($1,$2,'fake','TEST-subscription','TEST-customer',$3,'incomplete',true)`, sub, f.member, revision)
	subscription := billing.Subscription{ID: "TEST-subscription", CustomerID: "TEST-customer", PriceID: "TEST-price", Status: "active"}
	provider.SetState(subscription, []billing.Invoice{{ID: "TEST-invoice", SubscriptionID: subscription.ID, CustomerID: subscription.CustomerID, Status: "paid", AmountMinor: 1000, Currency: "TWD", Period: entitlement.Period{Start: now.Add(-time.Hour), End: now.Add(24 * time.Hour)}}})
	service, err := billing.NewService(f.pool, provider, vault, f.services.HTTP.Members.Entitlements, billing.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	raw := budgetJSON(t, billing.Event{ID: "TEST-event", Type: "invoice.paid", SubscriptionID: subscription.ID})
	if _, err := service.Ingest(t.Context(), raw, provider.Sign(raw)); err != nil {
		t.Fatal(err)
	}
	event := loadLatestOutbox(t, t.Context(), f.pool, "billing.inbox_received")
	if kind == "billing.reconcile" {
		event.EventType = kind
		event.Payload = budgetJSON(t, scheduledPage{Run: uuid.New()})
	}
	tr := &budgetTrace{}
	pool := budgetPool(t, f.pool, max(1, size), tr)
	service.Pool = pool
	f.services.HTTP.Billing = service
	f.services.HTTP.Members.Pool = pool
	return pool, func(ctx context.Context) (string, error) {
		return MemberJobs()[kind](WithMembers(ctx, f.services), event)
	}, tr
}

func dnsBudgetCase(t *testing.T, kind string, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
	t.Helper()
	base := dbtest.New(t)
	ctx := t.Context()
	now := time.Now().UTC()
	const content = "malware.example.com\n"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, content) }))
	t.Cleanup(server.Close)
	client := server.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = transport
	t.Cleanup(transport.CloseIdleConnections)
	store := policyStore.NewStore(base)
	actor := uuid.New()
	code := "test-feed"
	payload := api.ThreatFeedPayload{Provider: "TEST ONLY", FeedCode: code, LicenseName: "TEST ONLY", LicenseUrl: "https://example.com/license", CommercialUseBasis: "TEST ONLY", Attribution: presence.NewNullNullable[string](), SourceUrl: "https://example.com/feed", Format: api.ThreatFeedPayloadFormatDomainList, Categories: []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware}, ContentSha256: fmt.Sprintf("%x", sha256.Sum256([]byte(content))), MaxSizeBytes: 4096, ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(48 * time.Hour), UpdateIntervalSeconds: 300}
	feed, err := store.CreateRevision(ctx, policyStore.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: budgetJSON(t, payload), Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateRevision(ctx, policyStore.ActivateRevisionInput{DefinitionID: feed.DefinitionID, RevisionID: feed.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	event := loadLatestOutbox(t, ctx, base, "dns.feed.refresh")
	if kind == "dns.publish" {
		feeds, err := dnsfeeds.New(base, dnsfeeds.Options{Client: client})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = feeds.Handle(ctx, event); err != nil {
			t.Fatal(err)
		}
		code = "test-profile"
		profile, err := store.CreateRevision(ctx, policyStore.CreateRevisionInput{Code: &code, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1, Payload: budgetJSON(t, api.DnsProfilePayload{ProfileCode: code, DisplayName: api.LocalizedText{ZhTW: "測試"}, DomainRuleSetRevisionIds: []uuid.UUID{}, ThreatFeedRevisionIds: []uuid.UUID{feed.RevisionID}, BlockResponse: api.DnsProfilePayloadBlockResponseNxdomain, BlockedTtlSeconds: 60}), Reason: "TEST ONLY", ActorID: actor})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = store.ActivateRevision(ctx, policyStore.ActivateRevisionInput{DefinitionID: profile.DefinitionID, RevisionID: profile.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
			t.Fatal(err)
		}
		event = loadLatestOutbox(t, ctx, base, kind)
	}
	tr := &budgetTrace{}
	pool := budgetPool(t, base, max(1, size), tr)
	feeds, err := dnsfeeds.New(pool, dnsfeeds.Options{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := dnsstore.NewPublisher(pool, dnsstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	handler := dnsJobs(publisher, feeds)[kind].Handler
	if kind == "dns.schedule" {
		handler = func(ctx context.Context, _ db.ClaimedEvent) (string, error) {
			err := feeds.Schedule(ctx)
			return "cycle", err
		}
	}
	return pool, func(ctx context.Context) (string, error) { return handler(ctx, event) }, tr
}

func policyBudgetCase(t *testing.T, kind string, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
	t.Helper()
	f := newMemberPolicyFixture(t, true)
	ctx := t.Context()
	store := policyStore.NewStore(f.pool)
	code := "test-fair-use"
	actor := uuid.New()
	revision, err := store.CreateRevision(ctx, policyStore.CreateRevisionInput{Code: &code, Type: api.PolicyTypeFairUse, SchemaVersion: 1, Payload: budgetJSON(t, testEnforceFairUsePayload()), Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateRevision(ctx, policyStore.ActivateRevisionInput{DefinitionID: revision.DefinitionID, RevisionID: revision.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	event := loadLatestOutbox(t, ctx, f.pool, "policy.activated")
	settings := policiestest.New(t).Policies
	if kind == "fup.member_generation_changed" {
		services, err := NewPoliciesServices(f.pool, settings, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = services.FUPEngine.Compute(ctx, fup.ComputeInput{MemberID: f.member, PeriodID: fup.CurrentUTCMonthPeriodID(time.Now())}); err != nil {
			t.Fatal(err)
		}
		event = loadLatestOutbox(t, ctx, f.pool, kind)
	}
	if kind == "fup.evaluate" {
		event.EventType = kind
		event.Payload = budgetJSON(t, map[string]any{"member_id": f.member})
	}
	tr := &budgetTrace{}
	pool := budgetPool(t, f.pool, max(1, size), tr)
	compiler := desiredstate.NewCompiler(pool, testFairUseDesiredStateSigner(t))
	services, err := NewPoliciesServices(pool, settings, compiler)
	if err != nil {
		t.Fatal(err)
	}
	return pool, func(ctx context.Context) (string, error) {
		return PolicyJobs()[kind](WithPolicies(ctx, services), event)
	}, tr
}

func usageBudgetCase(t *testing.T, kind string, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
	t.Helper()
	f := newMemberPolicyFixture(t, true)
	base := f.pool
	fixture := usagetest.New(t)
	signer := usagepkg.NewProviderSigner(fixture.Provider)
	seed, err := usagepkg.NewService(usagepkg.Config{Pool: base, Signer: signer, LedgerEpochSigner: signer})
	if err != nil {
		t.Fatal(err)
	}
	device, err := f.services.HTTP.Devices.AuthorizeDevice(t.Context(), memberPolicyCommand("system", f.member), f.peer.peer)
	if err != nil {
		t.Fatal(err)
	}
	var incarnation uuid.UUID
	if err = base.QueryRow(t.Context(), `SELECT current_incarnation_id FROM nodes WHERE id=$1`, f.node).Scan(&incarnation); err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC()
	end := start.Add(time.Second)
	sample, err := usagepkg.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: f.node, NodeIncarnationId: incarnation, EventSeq: 1, MeterEpochId: uuid.New(), IntervalStart: start, IntervalEnd: end, Quality: api.UsageQualityComplete, MeteringVersion: api.MeteringVersion(usagepkg.MeteringVersionForwardAdmissionV1), Records: []api.UsageRecord{{MemberId: f.member, DeviceId: device.DeviceID, BindingRevision: 1, PeriodId: start.Format("2006-01"), UploadBytes: 1000, DownloadBytes: 2000}}})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := seed.IngestBatch(t.Context(), usagepkg.NodeIdentity{NodeID: f.node, NodeIncarnationID: incarnation}, api.UsageBatchRequest{NodeId: f.node, NodeIncarnationId: incarnation, Events: []api.UsageEvent{sample}})
	if err != nil || batch.HighestContiguousSeq != 1 {
		t.Fatalf("seed usage: %+v %v", batch, err)
	}
	state, err := seed.GetLedgerState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// Retention must traverse a real eligible payload and persist its purge audit.
	err = db.InTx(t.Context(), base, db.TxOptions{}, func(tx pgx.Tx) error {
		id := uuid.New()
		if _, err := tx.Exec(t.Context(), `INSERT INTO usage_events(id,node_id,incarnation_id,event_seq,schema_version,payload_hash,record_count,ledger_epoch) VALUES($1,$2,$3,2,1,decode(repeat('00',32),'hex'),0,$4)`, id, f.node, incarnation, state.LedgerEpoch); err != nil {
			return err
		}
		_, err := tx.Exec(t.Context(), `INSERT INTO usage_event_payloads(event_id,payload,created_at) VALUES($1,$2,clock_timestamp()-interval '100 days')`, id, []byte(`{}`))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	tr := &budgetTrace{}
	pool := budgetPool(t, base, max(1, size), tr)
	service, err := usagepkg.NewService(usagepkg.Config{Pool: pool, Signer: signer, LedgerEpochSigner: signer})
	if err != nil {
		t.Fatal(err)
	}
	services := &UsageServices{Pool: pool, Config: fixture.Usage, Service: service}
	event := db.ClaimedEvent{OutboxEvent: db.OutboxEvent{EventType: kind, SchemaVersion: 1, Payload: json.RawMessage(`{}`)}}
	return pool, func(ctx context.Context) (string, error) { return UsageJobs()[kind](WithUsage(ctx, services), event) }, tr
}

func domainPeakEvents() []string {
	return []string{"billing.inbox_received", "billing.reconcile", "dns.publish", "dns.feed.refresh", "policy.activated", "fup.evaluate", "fup.member_generation_changed", "usage.projection_rebuild", "usage.retention_purge"}
}

func TestAPI01_DomainPoolPeaks(t *testing.T) {
	catalog, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range domainPeakEvents() {
		t.Run(kind, func(t *testing.T) {
			runBudgetPair(t, kind, catalog.Jobs[kind].Peak, func(t *testing.T, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
				switch kind {
				case "billing.inbox_received", "billing.reconcile":
					return billingBudgetCase(t, kind, size)
				case "dns.publish", "dns.feed.refresh":
					return dnsBudgetCase(t, kind, size)
				case "policy.activated", "fup.evaluate", "fup.member_generation_changed":
					return policyBudgetCase(t, kind, size)
				default:
					return usageBudgetCase(t, kind, size)
				}
			})
		})
	}
}
