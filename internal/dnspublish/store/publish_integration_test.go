//go:build integration

package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnsfeeds"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Test12_5_PublishActivationIsDeterministic proves an activated dns_profile
// becomes one canonical document, a second activation of the same rules matches
// byte for byte, and an older generation cannot move the head (§12.4, §12.5).
func Test12_5_PublishActivationIsDeterministic(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	fixedNow := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	store := policies.NewStore(pool, policies.WithClock(func() time.Time { return fixedNow }))
	actorID := uuid.New()

	rsPayload := api.DomainRuleSetPayload{
		Rules: []api.DomainRule{
			{
				RuleId:            openapi_types.UUID(uuid.MustParse("01900000-0000-7000-8000-0000000000b1")),
				ProfileId:         presence.NewNullableWithValue("standard"),
				Domain:            "tracker.example.com",
				IncludeSubdomains: true,
				Action:            api.DomainRuleActionBlock,
				Reason:            "admin_rule",
				Revision:          1,
			},
			{
				RuleId:            openapi_types.UUID(uuid.MustParse("01900000-0000-7000-8000-0000000000a1")),
				ProfileId:         presence.NewNullNullable[string](),
				Domain:            "cdn.example.com",
				IncludeSubdomains: false,
				Action:            api.DomainRuleActionAllow,
				Reason:            "false_positive",
				Revision:          1,
			},
		},
	}
	rsRaw, err := json.Marshal(rsPayload)
	if err != nil {
		t.Fatal(err)
	}
	rsCode := "standard-rules"
	rsRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code: &rsCode, Type: api.PolicyTypeDomainRuleSet, SchemaVersion: 1,
		Reason: "Create domain rules", Payload: rsRaw, ActorID: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}

	feedBody := "malware.example.com\n"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, feedBody) }))
	defer server.Close()
	client := server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr
	en := "Example feed"
	tfPayload := api.ThreatFeedPayload{
		Provider:              "Example Threat Intel",
		FeedCode:              "example-malware",
		LicenseName:           "Commercial Threat Feed License",
		LicenseUrl:            "https://feeds.example.com/license",
		CommercialUseBasis:    "Licensed for commercial SaaS/VPN redistribution (TEST ONLY).",
		Attribution:           presence.FromPointer(&en),
		SourceUrl:             "https://example.com/malware.txt",
		Format:                api.ThreatFeedPayloadFormatDomainList,
		Categories:            []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware},
		ContentSha256:         fmt.Sprintf("%x", sha256.Sum256([]byte(feedBody))),
		MaxSizeBytes:          1048576,
		ValidFrom:             fixedNow.Add(-time.Hour),
		ValidUntil:            fixedNow.Add(30 * 24 * time.Hour),
		UpdateIntervalSeconds: 3600,
	}
	tfRaw, err := json.Marshal(tfPayload)
	if err != nil {
		t.Fatal(err)
	}
	tfCode := "example-malware"
	tfRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code: &tfCode, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1,
		Reason: "Create threat feed", Payload: tfRaw, ActorID: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: tfRev.DefinitionID, RevisionID: tfRev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY approve feed", ActorID: actorID}); err != nil {
		t.Fatal(err)
	}
	feeds, err := dnsfeeds.New(pool, dnsfeeds.Options{Client: client, Clock: func() time.Time { return fixedNow }})
	if err != nil {
		t.Fatal(err)
	}
	refreshPayload, _ := json.Marshal(map[string]string{"revision_id": tfRev.RevisionID.String()})
	refresh := db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: dnsfeeds.EventRefresh, SchemaVersion: 1, Payload: refreshPayload}}
	if _, err := feeds.Handle(ctx, refresh); err != nil {
		t.Fatal(err)
	}
	dnsPayload := api.DnsProfilePayload{
		ProfileCode:              "standard",
		DisplayName:              api.LocalizedText{ZhTW: "標準防護", En: &en},
		DomainRuleSetRevisionIds: []openapi_types.UUID{openapi_types.UUID(rsRev.RevisionID)},
		ThreatFeedRevisionIds:    []openapi_types.UUID{openapi_types.UUID(tfRev.RevisionID)},
		BlockResponse:            api.DnsProfilePayloadBlockResponseNxdomain,
		BlockedTtlSeconds:        60,
	}
	dnsRaw, err := json.Marshal(dnsPayload)
	if err != nil {
		t.Fatal(err)
	}
	dnsCode := "standard"
	rev1, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code: &dnsCode, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1,
		Reason: "Create standard DNS profile", Payload: dnsRaw, ActorID: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID: rev1.DefinitionID, RevisionID: rev1.RevisionID,
		ExpectedVersion: "1", Reason: "Activate standard DNS profile", ActorID: actorID,
	}); err != nil {
		t.Fatal(err)
	}

	pub, err := NewPublisher(pool, Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	reader := NewDocumentReader(pool)
	if _, err := reader.LatestPublished(ctx, rev1.DefinitionID); !errors.Is(err, dnspublish.ErrDocumentNotFound) {
		t.Fatalf("unpublished lookup: %v", err)
	}
	if _, err := reader.GetByID(ctx, uuid.New()); !errors.Is(err, dnspublish.ErrDocumentNotFound) {
		t.Fatalf("unknown lookup: %v", err)
	}
	ev1 := loadPublishEvent(t, ctx, pool, 0)
	receipt, err := pub.Handle(ctx, ev1)
	if err != nil {
		t.Fatal(err)
	}
	again, err := pub.Handle(ctx, ev1)
	if err != nil || again != receipt {
		t.Fatalf("replay receipt %q err %v, want %q", again, err, receipt)
	}

	var outboxN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1`, rev1.DefinitionID).Scan(&outboxN); err != nil {
		t.Fatal(err)
	}
	if outboxN != 2 {
		t.Fatalf("activation fan-out produced %d outbox rows, want policy.activated and dns.publish", outboxN)
	}

	stored, err := pub.GetByRevision(ctx, rev1.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(receipt, "dns-document:") || stored.DocumentHash == "" || receipt != "dns-document:"+stored.DocumentHash {
		t.Fatalf("receipt %s hash %s", receipt, stored.DocumentHash)
	}
	if strings.Contains(string(stored.Canonical), "policy_generation") {
		t.Fatal("policy_generation leaked into the stored document")
	}
	if stored.PolicyGeneration.String() != "1" || stored.Status != statusActive {
		t.Fatalf("head publication %+v", stored)
	}
	if stored.Document.Rules[0].Domain != "tracker.example.com" {
		t.Fatalf("canonical domain %s", stored.Document.Rules[0].Domain)
	}
	if len(stored.Document.Feeds) != 1 || stored.Document.Feeds[0].ContentSHA256 == "" {
		t.Fatalf("feed pin missing: %+v", stored.Document.Feeds)
	}
	sum, raw, err := dnspublish.Hash(stored.Document)
	if err != nil || sum != stored.DocumentHash || string(raw) != string(stored.Canonical) {
		t.Fatalf("stored bytes do not match the canonical hash")
	}

	originalDocument := assertDocumentLookup(t, pool, reader, stored)

	var opStatus, observed string
	if err := pool.QueryRow(ctx, `SELECT status, observed_refs::text FROM operations WHERE id = $1`, uuid.MustParse(operationID(t, ev1))).Scan(&opStatus, &observed); err != nil {
		t.Fatal(err)
	}
	if opStatus != "pending" || !strings.Contains(observed, stored.DocumentHash) {
		t.Fatalf("operation status %s observed %s", opStatus, observed)
	}
	var operationEvidence struct {
		Document struct {
			Generation    string `json:"generation"`
			ContentSHA256 string `json:"content_sha256"`
		} `json:"dns_document"`
	}
	if err := json.Unmarshal([]byte(observed), &operationEvidence); err != nil || operationEvidence.Document.Generation != stored.PolicyGeneration.String() || operationEvidence.Document.ContentSHA256 != stored.DocumentHash {
		t.Fatalf("revision 5 publication evidence: %v", err)
	}
	var audit string
	if err := pool.QueryRow(ctx, `SELECT coalesce(string_agg(after_diff::text, ','), '') FROM audit_events WHERE action IN ('dns.publish', 'dns.withdraw')`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(audit, "tracker") || strings.Contains(audit, "example.com") {
		t.Fatalf("audit recorded a domain name: %s", audit)
	}

	rev2, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		DefinitionID: &rev1.DefinitionID, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1,
		Reason: "Activate the same rules again", Payload: dnsRaw, ActorID: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	var ver string
	if err := pool.QueryRow(ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, rev1.DefinitionID).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	act2, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID: rev1.DefinitionID, RevisionID: rev2.RevisionID,
		ExpectedVersion: ver, Reason: "Second activation of the same rules", ActorID: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.LatestPublished(ctx, rev1.DefinitionID); !errors.Is(err, dnspublish.ErrDocumentNotFound) {
		t.Fatalf("old head available while replacement is unpublished: %v", err)
	}
	ev2 := loadPublishEvent(t, ctx, pool, 1)
	if _, err := pub.Handle(ctx, ev2); err != nil {
		t.Fatal(err)
	}
	second, err := pub.GetByRevision(ctx, rev2.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	if string(second.Canonical) != string(stored.Canonical) || second.DocumentHash != stored.DocumentHash {
		t.Fatal("same rules produced different resolver bytes")
	}
	if second.PolicyGeneration.String() == stored.PolicyGeneration.String() || act2.Generation.String() != second.PolicyGeneration.String() {
		t.Fatalf("generations rev1 %s rev2 %s activate %s", stored.PolicyGeneration, second.PolicyGeneration, act2.Generation)
	}
	head, err := pub.Head(ctx, rev1.DefinitionID)
	if err != nil {
		t.Fatal(err)
	}
	if head.ProfileRevisionID != rev2.RevisionID || head.Status != statusActive {
		t.Fatalf("head did not advance: %+v", head.ProfileRevisionID)
	}
	secondDocument := assertDocumentLookup(t, pool, reader, second)
	if secondDocument.DocumentID == originalDocument.DocumentID {
		t.Fatal("new revision reused a document ID")
	}
	// Same content is a successful verification, but does not create a new
	// content version, document generation, or refresh publication intent.
	fixedNow = fixedNow.Add(6 * time.Hour)
	refresh.ID = uuid.New()
	if _, err := feeds.Handle(ctx, refresh); err != nil {
		t.Fatal(err)
	}
	var refreshPublications int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE event_type='dns.publish' AND payload->>'action'='feed_refresh'`).Scan(&refreshPublications); err != nil || refreshPublications != 0 {
		t.Fatalf("unchanged refresh fanout: %d %v", refreshPublications, err)
	}
	unchanged := assertDocumentLookup(t, pool, reader, second)
	if unchanged.DocumentID != secondDocument.DocumentID {
		t.Fatal("same input changed document ID")
	}
	// A moved last-good produces a new immutable document on the same revision.
	fixedNow = fixedNow.Add(6 * time.Hour)
	feedBody += "new-malware.example.com\n"
	refresh.ID = uuid.New()
	if _, err := feeds.Handle(ctx, refresh); err != nil {
		t.Fatal(err)
	}
	var refreshEvent db.ClaimedEvent
	refreshEvent.EventType = dnspublish.EventDNSPublish
	if err := pool.QueryRow(ctx, `SELECT payload FROM outbox_events WHERE event_type='dns.publish' AND payload->>'action'='feed_refresh'`).Scan(&refreshEvent.Payload); err != nil {
		t.Fatal(err)
	}
	if _, err := pub.Handle(ctx, refreshEvent); err != nil {
		t.Fatal(err)
	}
	moved, err := pub.GetByRevision(ctx, rev2.RevisionID)
	if err != nil || moved.PolicyGeneration <= second.PolicyGeneration || moved.DocumentHash == second.DocumentHash || moved.Document.Feeds[0].RuleCount != 2 {
		t.Fatalf("refresh publication: %+v %v", moved, err)
	}
	if _, err := pub.Handle(ctx, refreshEvent); err != nil {
		t.Fatal(err)
	}
	replay, err := pub.GetByRevision(ctx, rev2.RevisionID)
	if err != nil || string(replay.Canonical) != string(moved.Canonical) || replay.PolicyGeneration != moved.PolicyGeneration {
		t.Fatalf("refresh replay changed document: %v", err)
	}
	var versions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dns_resolver_documents WHERE profile_revision_id=$1`, rev2.RevisionID).Scan(&versions); err != nil || versions != 2 {
		t.Fatalf("immutable versions %d %v", versions, err)
	}
	historical, err := pub.GetByGeneration(ctx, rev2.RevisionID, second.PolicyGeneration)
	if err != nil || historical.DocumentHash != second.DocumentHash || historical.Status != statusSuperseded {
		t.Fatalf("retained generation: %+v %v", historical, err)
	}
	movedDocument := assertDocumentLookup(t, pool, reader, moved)
	if movedDocument.DocumentID == secondDocument.DocumentID || movedDocument.Generation <= secondDocument.Generation || movedDocument.ContentSHA256 == secondDocument.ContentSHA256 {
		t.Fatal("moved last-good did not create a new immutable reference")
	}
	for _, old := range []dnspublish.PublishedDocument{originalDocument, secondDocument} {
		retained, err := reader.GetByID(ctx, old.DocumentID)
		if err != nil || string(retained.Canonical) != string(old.Canonical) || retained.ContentSHA256 != old.ContentSHA256 || retained.Generation != old.Generation {
			t.Fatalf("historical ID followed the head: %v", err)
		}
	}
	second = moved
	older, err := pub.GetByRevision(ctx, rev1.RevisionID)
	if err != nil || older.Status != statusSuperseded {
		t.Fatalf("older document status %s err %v", older.Status, err)
	}

	if _, err := pub.Handle(ctx, ev1); err != nil {
		t.Fatal(err)
	}
	head, err = pub.Head(ctx, rev1.DefinitionID)
	if err != nil || head.PolicyGeneration != second.PolicyGeneration || head.ProfileRevisionID != rev2.RevisionID {
		t.Fatalf("stale activation moved the head: %+v", head)
	}

	staleWithdraw := ev1
	staleWithdraw.Payload = withAction(t, ev1.Payload, "withdraw")
	if _, err := pub.Handle(ctx, staleWithdraw); err != nil {
		t.Fatal(err)
	}
	head, err = pub.Head(ctx, rev1.DefinitionID)
	if err != nil || head.Status != statusActive || head.ProfileRevisionID != rev2.RevisionID {
		t.Fatalf("stale withdraw moved the head: %+v status %s", head.ProfileRevisionID, head.Status)
	}

	withdraw := ev2
	withdraw.Payload = withAction(t, ev2.Payload, "withdraw")
	got, err := pub.Handle(ctx, withdraw)
	if err != nil || !strings.HasPrefix(got, "dns-withdrawn:") {
		t.Fatalf("withdraw receipt %q err %v", got, err)
	}
	head, err = pub.Head(ctx, rev1.DefinitionID)
	if err != nil || head.Status != statusWithdrawn || head.DocumentHash != second.DocumentHash {
		t.Fatalf("withdrawn head status %s hash %s", head.Status, head.DocumentHash)
	}
	if _, err := reader.LatestPublished(ctx, rev1.DefinitionID); !errors.Is(err, dnspublish.ErrDocumentNotFound) {
		t.Fatalf("withdrawn lookup: %v", err)
	}
	retained, err := reader.GetByID(ctx, movedDocument.DocumentID)
	if err != nil || retained.ContentSHA256 != movedDocument.ContentSHA256 {
		t.Fatalf("withdrawal erased immutable reference: %v", err)
	}
	if _, err := pub.Handle(ctx, ev2); err == nil {
		t.Fatal("reactivating a withdrawn generation succeeded")
	}
	head, err = pub.Head(ctx, rev1.DefinitionID)
	if err != nil || head.Status != statusWithdrawn {
		t.Fatalf("failed reactivation changed the head: %s", head.Status)
	}

	_, err = pool.Exec(ctx, `UPDATE dns_resolver_documents SET profile_code = 'other' WHERE profile_revision_id = $1`, rev1.RevisionID)
	if err == nil {
		t.Fatal("resolver document update was accepted")
	}
	_, err = pool.Exec(ctx, `DELETE FROM dns_resolver_documents WHERE profile_revision_id = $1`, rev1.RevisionID)
	if err == nil {
		t.Fatal("resolver document delete was accepted")
	}
	_, err = pool.Exec(ctx, `
		UPDATE dns_publication_heads
		SET policy_generation = policy_generation - 1, resource_version = resource_version + 1
		WHERE definition_id = $1`, rev1.DefinitionID)
	if err == nil {
		t.Fatal("publication generation regression was accepted")
	}
	assertDNSAudit(t, ctx, pool, rev1.DefinitionID, []uuid.UUID{rev1.RevisionID, rev2.RevisionID})
}

func assertDNSAudit(t *testing.T, ctx context.Context, pool *pgxpool.Pool, defID uuid.UUID, revisions []uuid.UUID) {
	t.Helper()
	var stored, all int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('dns.publish', 'dns.withdraw')`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == 0 {
		t.Fatal("dns publication wrote no audit events")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&all); err != nil {
		t.Fatal(err)
	}
	limit := 100
	reader := audit.Store{Pool: pool}
	page, err := reader.List(ctx, api.ListAuditEventsParams{Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if reader.AnomalyCount() != 0 || len(page.Items) != all {
		t.Fatalf("audit read-back listed %d of %d rows with %d anomalies", len(page.Items), all, reader.AnomalyCount())
	}
	revisionsSeen := map[uuid.UUID]bool{}
	for _, id := range revisions {
		revisionsSeen[id] = true
	}
	systemID := audit.SystemActor.ID
	seen := map[string]map[api.ResourceType]int{}
	dnsRows := 0
	for _, ev := range page.Items {
		if ev.Action != "dns.publish" && ev.Action != "dns.withdraw" {
			continue
		}
		dnsRows++
		actorID, err := ev.Actor.Id.Get()
		if err != nil || ev.Actor.Type != api.AuditEventActorTypeSystem || uuid.UUID(actorID) != systemID {
			t.Fatalf("%s actor %+v err %v", ev.Action, ev.Actor, err)
		}
		if ev.Result != api.AuditEventResultSucceeded {
			t.Fatalf("%s result %s", ev.Action, ev.Result)
		}
		switch ev.Resource.Type {
		case api.ResourceTypePolicyDefinition:
			if ev.Resource.Id != defID {
				t.Fatalf("definition resource %s", ev.Resource.Id)
			}
		case api.ResourceTypePolicyRevision:
			if !revisionsSeen[ev.Resource.Id] {
				t.Fatalf("revision resource %s", ev.Resource.Id)
			}
		default:
			t.Fatalf("resource %s", ev.Resource.Type)
		}
		// Automatic content refreshes have no admin operation or request.
		if opID, err := ev.OperationId.Get(); err == nil {
			reqID, err := ev.RequestId.Get()
			if err != nil {
				t.Fatalf("%s missing request id", ev.Action)
			}
			var storedReq uuid.UUID
			if err := pool.QueryRow(ctx, `SELECT request_id FROM operations WHERE id=$1`, uuid.UUID(opID)).Scan(&storedReq); err != nil || storedReq != uuid.UUID(reqID) {
				t.Fatalf("request correlation: %v", err)
			}
		}
		after, err := ev.Changes.After.Get()
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"generation", "content_sha256", "status", "head", "definition_id", "profile_revision_id"} {
			value, _ := after[key].(string)
			if value == "" {
				t.Fatalf("%s missing %s in %#v", ev.Action, key, after)
			}
		}
		if after["definition_id"] != defID.String() {
			t.Fatalf("definition_id %v", after["definition_id"])
		}
		raw, err := json.Marshal(after)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "example.com") || strings.Contains(string(raw), "tracker") {
			t.Fatalf("audit leaked a name: %s", raw)
		}
		if seen[ev.Action] == nil {
			seen[ev.Action] = map[api.ResourceType]int{}
		}
		seen[ev.Action][ev.Resource.Type]++
	}
	if dnsRows != stored {
		t.Fatalf("listed %d dns audit rows, stored %d", dnsRows, stored)
	}
	for _, action := range []string{"dns.publish", "dns.withdraw"} {
		if seen[action][api.ResourceTypePolicyDefinition] == 0 || seen[action][api.ResourceTypePolicyDefinition] != seen[action][api.ResourceTypePolicyRevision] {
			t.Fatalf("%s resources %#v", action, seen[action])
		}
	}
}

func operationID(t *testing.T, event db.ClaimedEvent) string {
	t.Helper()
	var body struct {
		OperationID string `json:"operation_id"`
	}
	if err := json.Unmarshal(event.Payload, &body); err != nil || body.OperationID == "" {
		t.Fatal("publish event has no operation_id")
	}
	return body.OperationID
}

func loadPublishEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, nth int) db.ClaimedEvent {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT event_type, payload
		FROM outbox_events
		WHERE event_type = 'dns.publish'
		ORDER BY created_at, id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var found []db.ClaimedEvent
	for rows.Next() {
		var ev db.ClaimedEvent
		if err := rows.Scan(&ev.EventType, &ev.Payload); err != nil {
			t.Fatal(err)
		}
		found = append(found, ev)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if nth < 0 || nth >= len(found) {
		t.Fatalf("dns.publish event %d not found (%d rows)", nth, len(found))
	}
	return found[nth]
}

func withAction(t *testing.T, raw []byte, action string) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	body["action"] = action
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestD054_DNS01_MissingLastGoodCannotPublish(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	ps := policies.NewStore(pool)
	actor := uuid.New()
	now := time.Now().UTC()
	feed := api.ThreatFeedPayload{Provider: "TEST ONLY", FeedCode: "missing-feed", LicenseName: "TEST ONLY", LicenseUrl: "https://example.com/license", CommercialUseBasis: "TEST ONLY fixture license", Attribution: presence.NewNullableWithValue("TEST ONLY attribution"), SourceUrl: "https://example.com/feed", Format: api.ThreatFeedPayloadFormatDomainList, Categories: []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware}, ContentSha256: strings.Repeat("a", 64), MaxSizeBytes: 4096, ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(time.Hour), UpdateIntervalSeconds: 300}
	raw, _ := json.Marshal(feed)
	code := "missing-feed"
	rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	profile := api.DnsProfilePayload{ProfileCode: "waiting", DisplayName: api.LocalizedText{ZhTW: "TEST ONLY"}, DomainRuleSetRevisionIds: []uuid.UUID{}, ThreatFeedRevisionIds: []uuid.UUID{rev.RevisionID}, BlockResponse: api.DnsProfilePayloadBlockResponseNxdomain, BlockedTtlSeconds: 60}
	raw, _ = json.Marshal(profile)
	code = "waiting"
	prof, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: prof.DefinitionID, RevisionID: prof.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	p, err := NewPublisher(pool, Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Handle(ctx, loadPublishEvent(t, ctx, pool, 0)); err == nil || err.Error() != "feed_last_good_missing" {
		t.Fatalf("missing content accepted: %v", err)
	}
	var heads int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dns_publication_heads`).Scan(&heads); err != nil || heads != 0 {
		t.Fatalf("head without content: %d %v", heads, err)
	}
}
