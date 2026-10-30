//go:build integration

package dnsfeeds

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnspublishstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

func TestD054_ApprovedBaselineSurvivesRestartAndReceiptSurvivesLaterAttempts(t *testing.T) {
	pool, ctx := dbtest.New(t), t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	approved := "a.example.com\nb.example.com\nc.example.com\nd.example.com\n"
	body := approved
	client := localFeed(t, &body)
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))
	p := testPayload(approved, now)
	p.Attribution.SetNull()
	raw, _ := json.Marshal(p)
	code, actor := "durable-feed", uuid.New()
	rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	activation, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	newService := func() *Service {
		t.Helper()
		s, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := newService()
	var original db.ClaimedEvent
	if err := pool.QueryRow(ctx, `SELECT id,event_type,schema_version,payload FROM outbox_events WHERE event_type='dns.feed.refresh' AND aggregate_id=$1`, rev.RevisionID).Scan(&original.ID, &original.EventType, &original.SchemaVersion, &original.Payload); err != nil {
		t.Fatal(err)
	}
	body = "unapproved.example.com\n"
	receipt, err := s.Handle(ctx, original)
	if err != nil || !strings.Contains(receipt, "content_mismatch") {
		t.Fatalf("bootstrap: %q %v", receipt, err)
	}
	var persisted string
	var delivered *time.Time
	if err := pool.QueryRow(ctx, `SELECT delivery_receipt,delivered_at FROM outbox_events WHERE id=$1`, original.ID).Scan(&persisted, &delivered); err != nil || persisted != receipt || delivered != nil {
		t.Fatalf("finish must persist receipt before dispatcher completion: %q %v %v", persisted, delivered, err)
	}
	op, err := ps.GetOperation(ctx, activation.Operation.OperationId)
	if err != nil {
		t.Fatal(err)
	}
	failure, err := op.Error.Get()
	if err != nil || failure.Code != api.ErrorCodeHASHMISMATCH {
		t.Fatalf("activation failure projection: %+v %v", op, err)
	}
	refresh := func(want string) {
		t.Helper()
		now = now.Add(6 * time.Hour)
		e := original
		e.ID = uuid.New()
		got, err := s.Handle(ctx, e)
		if err != nil || !strings.Contains(got, want) {
			t.Fatalf("refresh: %q want %q: %v", got, want, err)
		}
	}
	body = approved
	refresh("refreshed")
	addition := "e.example.com\nf.example.com\ng.example.com\nh.example.com\n"
	body = approved + addition
	refresh("refreshed")
	lastGood := digest(body)
	s = newService() // no in-memory copy of the immutable approved baseline
	body = addition
	refresh("delta_breach")
	var hash string
	if err := pool.QueryRow(ctx, `SELECT last_good_content FROM dns_feed_refresh_state WHERE feed_revision_id=$1`, rev.RevisionID).Scan(&hash); err != nil || hash != lastGood {
		t.Fatalf("dilution changed last-good: %s %v", hash, err)
	}
	// Simulate the crash after finish but before dispatcher Complete, followed
	// by other attempts. Replay must recover the original failure receipt.
	if replay, err := s.Handle(ctx, original); err != nil || replay != receipt {
		t.Fatalf("replay changed receipt: %q want %q: %v", replay, receipt, err)
	}
	var failures, audits int
	if err := pool.QueryRow(ctx, `SELECT consecutive_failures FROM dns_feed_refresh_state WHERE feed_revision_id=$1`, rev.RevisionID).Scan(&failures); err != nil || failures != 1 {
		t.Fatalf("replay failure count: %d %v", failures, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action LIKE 'dns.feed.%'`).Scan(&audits); err != nil || audits != 4 {
		t.Fatalf("replay duplicated audit: %d %v", audits, err)
	}
}

func TestD054_LifecycleInvalidatesDeliveryAndReactivationRestoresNewGeneration(t *testing.T) {
	pool, ctx := dbtest.New(t), t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Minute)
	body := "a.example.com\n"
	client := localFeed(t, &body)
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))
	actor, code := uuid.New(), "lifecycle-feed"
	p := testPayload(body, now)
	p.Attribution.SetNull()
	p.ValidUntil = now.Add(time.Hour)
	raw, _ := json.Marshal(p)
	rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	activate := func(id uuid.UUID) {
		t.Helper()
		def, err := ps.GetDefinition(ctx, rev.DefinitionID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: id, ExpectedVersion: def.ResourceVersion.String(), Reason: "TEST ONLY", ActorID: actor}); err != nil {
			t.Fatal(err)
		}
	}
	activate(rev.RevisionID)
	s, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	refresh := func(id uuid.UUID) {
		t.Helper()
		b, _ := json.Marshal(map[string]string{"revision_id": id.String()})
		if receipt, err := s.Handle(ctx, db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: b}}); err != nil || !strings.Contains(receipt, "refreshed") {
			t.Fatalf("refresh: %s %v", receipt, err)
		}
	}
	refresh(rev.RevisionID)
	code = "lifecycle-profile"
	profileRaw, _ := json.Marshal(api.DnsProfilePayload{ProfileCode: code, DisplayName: api.LocalizedText{ZhTW: "測試"}, DomainRuleSetRevisionIds: []uuid.UUID{}, ThreatFeedRevisionIds: []uuid.UUID{rev.RevisionID}, BlockResponse: api.DnsProfilePayloadBlockResponseNxdomain, BlockedTtlSeconds: 60})
	profile, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1, Payload: profileRaw, Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: profile.DefinitionID, RevisionID: profile.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	pub, err := dnspublishstore.NewPublisher(pool, dnspublishstore.Config{Upstreams: []string{"192.0.2.53:53"}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]string{"definition_id": profile.DefinitionID.String(), "revision_id": profile.RevisionID.String(), "generation": "1"})
	event := db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: dnspublish.EventDNSPublish, SchemaVersion: 1, Payload: b}}
	if _, err := pub.Handle(ctx, event); err != nil {
		t.Fatal(err)
	}
	reader := dnspublishstore.NewDocumentReader(pool)
	first, err := reader.LatestPublished(ctx, profile.DefinitionID)
	if err != nil {
		t.Fatal(err)
	}
	// Supersession cannot silently repoint an explicitly approved profile.
	raw, _ = json.Marshal(p)
	replacement, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{DefinitionID: &rev.DefinitionID, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY replacement", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	activate(replacement.RevisionID)
	if _, err := reader.GetByID(ctx, first.DocumentID); !errors.Is(err, dnspublish.ErrDocumentNotFound) {
		t.Fatalf("superseded feed delivered: %v", err)
	}
	if _, err := pub.Handle(ctx, event); err == nil {
		t.Fatal("superseded feed published")
	}
	activate(rev.RevisionID)
	refresh(rev.RevisionID) // within the original cadence
	if _, err := pub.Handle(ctx, event); err != nil {
		t.Fatal(err)
	}
	recovered, err := reader.LatestPublished(ctx, profile.DefinitionID)
	if err != nil || recovered.Generation <= first.Generation || recovered.ContentSHA256 != first.ContentSHA256 {
		t.Fatalf("recovery generation: %+v %v", recovered, err)
	}
	// Expiry alone also schedules invalidation without a feed hash change.
	if _, err := pool.Exec(ctx, `UPDATE outbox_events SET delivered_at=clock_timestamp() WHERE event_type='dns.publish'`); err != nil {
		t.Fatal(err)
	}
	now = p.ValidUntil
	if err := s.Schedule(ctx); err != nil {
		t.Fatal(err)
	}
	var pending int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type='dns.publish' AND delivered_at IS NULL`).Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("expiry repair count: %d %v", pending, err)
	}
	if _, err := pub.Handle(ctx, event); err == nil {
		t.Fatal("expired feed published")
	}
	if _, err := reader.LatestPublished(ctx, profile.DefinitionID); !errors.Is(err, dnspublish.ErrDocumentNotFound) {
		t.Fatalf("expired publication available: %v", err)
	}
}

func TestD054_MetricsRetainOnlyActiveRevisionsAndAllDuplicateCodes(t *testing.T) {
	pool, ctx := dbtest.New(t), t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	body := "a.example.com\n"
	client := localFeed(t, &body)
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))
	reg := prometheus.NewRegistry()
	s, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }, Metrics: reg})
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	create := func(code string, definition *uuid.UUID) uuid.UUID {
		t.Helper()
		p := testPayload(body, now)
		p.Attribution.SetNull()
		raw, _ := json.Marshal(p)
		rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, DefinitionID: definition, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
		if err != nil {
			t.Fatal(err)
		}
		def, err := ps.GetDefinition(ctx, rev.DefinitionID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: def.ResourceVersion.String(), Reason: "TEST ONLY", ActorID: actor}); err != nil {
			t.Fatal(err)
		}
		return rev.RevisionID
	}
	first := create("metrics-one", nil)
	var definition uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT definition_id FROM policy_revisions WHERE id=$1`, first).Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if err := s.observeAges(ctx); err != nil {
		t.Fatal(err)
	}
	second := create("metrics-one", &definition)
	refresh := func(id uuid.UUID) {
		t.Helper()
		raw, _ := json.Marshal(map[string]string{"revision_id": id.String()})
		if _, err := s.Handle(ctx, db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: raw}}); err != nil {
			t.Fatal(err)
		}
	}
	refresh(second)
	age := func() float64 {
		t.Helper()
		families, err := reg.Gather()
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range families {
			if f.GetName() == "fvpn_dns_feed_last_good_age_seconds" {
				if len(f.Metric) != 1 {
					t.Fatal(f)
				}
				return f.Metric[0].Gauge.GetValue()
			}
		}
		t.Fatal("missing gauge")
		return 0
	}
	if got := age(); got != 0 {
		t.Fatalf("superseded bootstrap poisons active code: %v", got)
	}
	third := create("metrics-two", nil) // same payload.feed_code, separate definition
	if err := s.observeAges(ctx); err != nil {
		t.Fatal(err)
	}
	if got := age(); !math.IsInf(got, 1) {
		t.Fatalf("duplicate active empty feed hidden: %v", got)
	}
	refresh(third)
	if got := age(); got != 0 {
		t.Fatalf("recovered duplicate: %v", got)
	}
}
