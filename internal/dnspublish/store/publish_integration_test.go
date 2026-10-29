//go:build integration

package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
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

	en := "Example feed"
	tfPayload := api.ThreatFeedPayload{
		Provider:              "Example Threat Intel",
		FeedCode:              "example-malware",
		LicenseName:           "Commercial Threat Feed License",
		LicenseUrl:            "https://feeds.example.com/license",
		CommercialUseBasis:    "Licensed for commercial SaaS/VPN redistribution (TEST ONLY).",
		Attribution:           presence.FromPointer(&en),
		SourceUrl:             "https://feeds.example.com/malware.txt",
		Format:                api.ThreatFeedPayloadFormatDomainList,
		Categories:            []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware},
		ContentSha256:         "7777777777777777777777777777777777777777777777777777777777777777",
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

	pub := NewPublisher(pool, DefaultConfig())
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
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events`).Scan(&outboxN); err != nil {
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

	var opStatus, observed string
	if err := pool.QueryRow(ctx, `SELECT status, observed_refs::text FROM operations WHERE id = $1`, uuid.MustParse(operationID(t, ev1))).Scan(&opStatus, &observed); err != nil {
		t.Fatal(err)
	}
	if opStatus != "pending" || !strings.Contains(observed, stored.DocumentHash) {
		t.Fatalf("operation status %s observed %s", opStatus, observed)
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
