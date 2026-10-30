//go:build integration

package dnsfeeds

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnspublishstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestReview_F05_SupersededAndExpiredFeedRevisionCompiledIntoActiveProfileAndReactivationSkipped(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

	body := "old-malware.example.com\n"
	client := localFeed(t, &body)

	// 1. Create, activate, and bootstrap threat_feed Revision 1 (valid for 1 hour).
	attr := "TEST ONLY attribution"
	payload1 := testPayload(body, now)
	payload1.ValidUntil = now.Add(time.Hour)
	payload1.Attribution = presence.NewNullableWithValue(attr)
	raw1, _ := json.Marshal(payload1)

	feedCode := "test-feed"
	actor := uuid.New()
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))

	rev1, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &feedCode,
		Type:          api.PolicyTypeThreatFeed,
		SchemaVersion: 1,
		Payload:       raw1,
		Reason:        "TEST ONLY rev1",
		ActorID:       actor,
	})
	if err != nil {
		t.Fatalf("CreateRevision rev1: %v", err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: "1",
		Reason:          "TEST ONLY activate rev1",
		ActorID:         actor,
	}); err != nil {
		t.Fatalf("ActivateRevision rev1: %v", err)
	}

	svc, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("dnsfeeds.New: %v", err)
	}
	eventFor := func(revID uuid.UUID) db.ClaimedEvent {
		b, _ := json.Marshal(map[string]string{"revision_id": revID.String()})
		return db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: b}}
	}

	if receipt, err := svc.Handle(ctx, eventFor(rev1.RevisionID)); err != nil || receipt == "dns-feed:unchanged" {
		t.Fatalf("bootstrap rev1: receipt=%q err=%v", receipt, err)
	}

	// 2. Create and activate a dns_profile referencing rev1.RevisionID.
	en := "Standard DNS"
	profCode := "standard-dns"
	profPayload := api.DnsProfilePayload{
		ProfileCode:              "standard-dns",
		DisplayName:              api.LocalizedText{ZhTW: "標準防護", En: &en},
		DomainRuleSetRevisionIds: []openapi_types.UUID{},
		ThreatFeedRevisionIds:    []openapi_types.UUID{openapi_types.UUID(rev1.RevisionID)},
		BlockResponse:            api.DnsProfilePayloadBlockResponseNxdomain,
		BlockedTtlSeconds:        60,
	}
	rawProf, _ := json.Marshal(profPayload)
	profRev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &profCode,
		Type:          api.PolicyTypeDnsProfile,
		SchemaVersion: 1,
		Payload:       rawProf,
		Reason:        "TEST ONLY profile",
		ActorID:       actor,
	})
	if err != nil {
		t.Fatalf("CreateRevision profile: %v", err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    profRev.DefinitionID,
		RevisionID:      profRev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "TEST ONLY activate profile",
		ActorID:         actor,
	}); err != nil {
		t.Fatalf("ActivateRevision profile: %v", err)
	}

	pub, err := dnspublishstore.NewPublisher(pool, dnspublishstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	publishEventFor := func(revID uuid.UUID) db.ClaimedEvent {
		b, _ := json.Marshal(map[string]string{
			"definition_id": profRev.DefinitionID.String(),
			"revision_id":   revID.String(),
			"generation":    "1",
		})
		return db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: dnspublish.EventDNSPublish, SchemaVersion: 1, Payload: b}}
	}
	if _, err := pub.Handle(ctx, publishEventFor(profRev.RevisionID)); err != nil {
		t.Fatalf("initial Publish: %v", err)
	}

	// 3. Advance clock by 2 hours so rev1 is EXPIRED (now > rev1.ValidUntil).
	// Operator supersedes rev1 with rev2 containing new malware domains and extended validity.
	now = now.Add(2 * time.Hour)
	body = "old-malware.example.com\nnew-malware.example.com\n"
	payload2 := testPayload(body, now)
	payload2.Attribution = presence.NewNullableWithValue(attr)
	raw2, _ := json.Marshal(payload2)

	rev2, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{
		DefinitionID:  &rev1.DefinitionID,
		Type:          api.PolicyTypeThreatFeed,
		SchemaVersion: 1,
		Payload:       raw2,
		Reason:        "TEST ONLY rev2",
		ActorID:       actor,
	})
	if err != nil {
		t.Fatalf("CreateRevision rev2: %v", err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev2.RevisionID,
		ExpectedVersion: "2",
		Reason:          "TEST ONLY activate rev2",
		ActorID:         actor,
	}); err != nil {
		t.Fatalf("ActivateRevision rev2: %v", err)
	}

	var publishEventsBefore int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type='dns.publish'`).Scan(&publishEventsBefore)

	if receipt, err := svc.Handle(ctx, eventFor(rev2.RevisionID)); err != nil || receipt == "dns-feed:unchanged" {
		t.Fatalf("bootstrap rev2: receipt=%q err=%v", receipt, err)
	}

	var publishEventsAfter int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type='dns.publish'`).Scan(&publishEventsAfter)
	if publishEventsAfter == publishEventsBefore {
		t.Errorf("TODO(review) [F-05]: activating and refreshing superseded threat_feed rev2 enqueued 0 dns.publish events for active dns_profile %s (still pinned to superseded rev1)",
			profRev.RevisionID)
	}

	// 4. Even when the active dns_profile is republished at now+2h (after rev1.ValidUntil expired
	// and rev1 was superseded by rev2), pub.Handle succeeds and compiles rev1's expired snapshot!
	if receipt, err := pub.Handle(ctx, publishEventFor(profRev.RevisionID)); err == nil {
		t.Errorf("TODO(review) [F-05]: dnspublish compiled superseded and expired threat_feed rev1 (valid_until=%s, now=%s) into active dns_profile (receipt=%q) without error",
			payload1.ValidUntil.Format(time.RFC3339), now.Format(time.RFC3339), receipt)
	}

	// 5. Re-activating a previous revision before its next_attempt_at silently skips refresh.
	// rev2 was just refreshed at `now`, so rev2.next_attempt_at is `now + 5m`.
	// If the operator activates rev1 and then immediately re-activates rev2 at `now + 1m`,
	// the activation trigger's dns.feed.refresh event for rev2 returns "dns-feed:unchanged" without fetching!
	now = now.Add(time.Minute)
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: "3",
		Reason:          "TEST ONLY rollback to rev1",
		ActorID:         actor,
	}); err != nil {
		t.Fatalf("ActivateRevision rollback rev1: %v", err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev2.RevisionID,
		ExpectedVersion: "4",
		Reason:          "TEST ONLY re-activate rev2",
		ActorID:         actor,
	}); err != nil {
		t.Fatalf("ActivateRevision re-activate rev2: %v", err)
	}

	receipt, err := svc.Handle(ctx, eventFor(rev2.RevisionID))
	if err != nil {
		t.Fatalf("Handle re-activated rev2: %v", err)
	}
	if strings.HasPrefix(receipt, "dns-feed:unchanged:") {
		t.Errorf("TODO(review) [F-05]: explicit re-activation of threat_feed rev2 before next_attempt_at was skipped with receipt=%q instead of verifying/fan-outing",
			receipt)
	}
}

func TestReview_F06_ThreatFeedStatusUnpopulatedAndPrematureLastAttemptAt(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

	body := "a.example.com\nb.example.com\nc.example.com\nd.example.com\n"
	var prematureLastAttemptAt *time.Time
	var prematureLastSuccessAt *time.Time
	var prematureLastErrorCode *string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Query dns_feed_refresh_state WHILE the HTTP fetch is still in flight (before completion).
		_ = pool.QueryRow(ctx, `SELECT last_attempt_at, last_success_at, last_error_code FROM dns_feed_refresh_state LIMIT 1`).Scan(&prematureLastAttemptAt, &prematureLastSuccessAt, &prematureLastErrorCode)
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()

	client := server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr

	payload := testPayload(body, now)
	payload.Attribution.SetNull()
	raw, _ := json.Marshal(payload)
	code := "status-test-feed"
	actor := uuid.New()
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))

	rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &code,
		Type:          api.PolicyTypeThreatFeed,
		SchemaVersion: 1,
		Payload:       raw,
		Reason:        "TEST ONLY status test",
		ActorID:       actor,
	})
	if err != nil {
		t.Fatalf("CreateRevision: %v", err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev.DefinitionID,
		RevisionID:      rev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "TEST ONLY activate",
		ActorID:         actor,
	}); err != nil {
		t.Fatalf("ActivateRevision: %v", err)
	}

	svc, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("dnsfeeds.New: %v", err)
	}
	b, _ := json.Marshal(map[string]string{"revision_id": rev.RevisionID.String()})
	ev := db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: b}}
	if _, err := svc.Handle(ctx, ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// 1. Check premature last_attempt_at mutation during in-flight HTTP request.
	if prematureLastAttemptAt != nil && prematureLastSuccessAt == nil && prematureLastErrorCode == nil {
		t.Errorf("TODO(review) [F-06]: begin() committed last_attempt_at=%s to dns_feed_refresh_state before HTTP fetch completed (while last_success_at and last_error_code were still nil)",
			prematureLastAttemptAt.Format(time.RFC3339))
	}

	// 2. Check Contract R5 PolicyDefinition.ThreatFeedStatus population on GetDefinition.
	def, err := ps.GetDefinition(ctx, rev.DefinitionID)
	if err != nil {
		t.Fatalf("GetDefinition: %v", err)
	}
	if !def.ThreatFeedStatus.IsSpecified() || def.ThreatFeedStatus.IsNull() {
		t.Errorf("TODO(review) [F-06]: GetDefinition for active threat_feed returned ThreatFeedStatus specified=%v null=%v; OpenAPI Revision 5 requires threat_feed_status when type=threat_feed and active_revision_id is non-null",
			def.ThreatFeedStatus.IsSpecified(), def.ThreatFeedStatus.IsNull())
	}
}
