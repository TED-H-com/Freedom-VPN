//go:build integration

package dnsfeeds

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

func TestD054_DNS01_JobRetainsLastGoodAndAuditsEveryFailure(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	body := "a.example.com\nb.example.com\nc.example.com\nd.example.com\n"
	client := localFeed(t, &body)
	payload := testPayload(body, now)
	payload.Attribution.SetNull()
	raw, _ := json.Marshal(payload)
	code := "test-feed"
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))
	actor := uuid.New()
	rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY feed", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	metrics := prometheus.NewRegistry()
	s, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	event := func() db.ClaimedEvent {
		b, _ := json.Marshal(map[string]string{"revision_id": rev.RevisionID.String()})
		return db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: b}}
	}
	// Drafts are never fetched or given refresh state.
	if _, err := s.Handle(ctx, event()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dns_feed_refresh_state`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("draft state %d %v", count, err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY approve", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type='dns.feed.refresh'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("activation trigger: %d %v", count, err)
	}
	// Fail bootstrap, then recover with the approved snapshot.
	approved := body
	body = "wrong.example.com\n"
	e := event()
	if receipt, err := s.Handle(ctx, e); err != nil || !strings.Contains(receipt, "content_mismatch") {
		t.Fatalf("mismatch %s %v", receipt, err)
	}
	if _, err := s.Handle(ctx, e); err != nil {
		t.Fatal(err)
	} // no duplicate failure on replay
	var hash *string
	var failures int
	if err := pool.QueryRow(ctx, `SELECT last_good_content,consecutive_failures FROM dns_feed_refresh_state`).Scan(&hash, &failures); err != nil || hash != nil || failures != 1 {
		t.Fatalf("bootstrap state %v %d %v", hash, failures, err)
	}
	now = now.Add(6 * time.Hour)
	body = approved
	if _, err := s.Handle(ctx, event()); err != nil {
		t.Fatal(err)
	}
	firstHash := digest(approved)
	for i, tt := range []struct{ body, reason string }{
		{strings.Repeat("a", 4097), "oversize"}, {"a.example.com\nhttps://example.com/bad\n", "parse_error"}, {"# empty\n", "zero_rules"}, {"a.example.com\n", "delta_breach"},
	} {
		now = now.Add(6 * time.Hour)
		body = tt.body
		if receipt, err := s.Handle(ctx, event()); err != nil || !strings.Contains(receipt, tt.reason) {
			t.Fatalf("%s: %s %v", tt.reason, receipt, err)
		}
		var reason string
		if err := pool.QueryRow(ctx, `SELECT last_good_content,last_error_code,consecutive_failures FROM dns_feed_refresh_state`).Scan(&hash, &reason, &failures); err != nil || hash == nil || *hash != firstHash || reason != tt.reason || failures != i+1 {
			t.Fatalf("last-good/error state: %v %s %d %v", hash, reason, failures, err)
		}
	}
	// SSRF failure through DNS resolution, while keeping an approved URL fixed.
	s.opts.Client = dnspolicyRefusingClient()
	now = now.Add(6 * time.Hour)
	if receipt, err := s.Handle(ctx, event()); err != nil || !strings.Contains(receipt, "ssrf_refused") {
		t.Fatalf("SSRF %s %v", receipt, err)
	}
	s.opts.Client = client
	now = now.Add(6 * time.Hour)
	body = "a.example.com\nb.example.com\nnew.example.com\n"
	if _, err := s.Handle(ctx, event()); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dns_feed_contents`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("content versions %d %v", count, err)
	}
	var audits string
	if err := pool.QueryRow(ctx, `SELECT string_agg(after_diff::text,',') FROM audit_events WHERE action LIKE 'dns.feed.%'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(audits, "example.com") || strings.Contains(audits, "https:") {
		t.Fatal("audit leaked feed URL or body")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='dns.feed.refresh_failed' AND actor_type='system' AND resource='policy_revision:' || $1::text`, rev.RevisionID).Scan(&count); err != nil || count != 6 {
		t.Fatalf("failure audits %d %v", count, err)
	}
	if err := s.observeAges(ctx); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	gathered, err := metrics.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range gathered {
		if m.GetName() == "fvpn_dns_feed_last_good_age_seconds" {
			found = true
			if m.Metric[0].Gauge.GetValue() != 3600 {
				t.Fatal(m)
			}
		}
	}
	if !found {
		t.Fatal("age metric missing")
	}
	// Once a due event is absent, concurrent repair does not multiply jobs.
	if _, err := pool.Exec(ctx, `UPDATE outbox_events SET delivered_at=clock_timestamp() WHERE event_type='dns.feed.refresh'`); err != nil {
		t.Fatal(err)
	}
	now = now.Add(6 * time.Hour)
	for range 2 {
		if err := s.Schedule(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type='dns.feed.refresh' AND delivered_at IS NULL`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("schedule duplicates: %d %v", count, err)
	}
}

func dnspolicyRefusingClient() *http.Client {
	return dnspolicy.NewSourceHTTPClient(time.Second, func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}, nil)
}
