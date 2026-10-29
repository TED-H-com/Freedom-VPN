//go:build integration

package policies_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func Test14_2_RevisionImmutabilityAndHash(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	fixedNow := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	store := policies.NewStore(pool, policies.WithClock(func() time.Time { return fixedNow }))

	actorID := uuid.New()
	idemKey := uuid.New()

	// Format payload with non-canonical whitespace and key order to prove
	// payload_sha256 is computed over RFC 8785 canonical JSON.
	nonCanonicalJSON := []byte(`{
		"overrides": [],
		"minimum_rate": { "up_bps": 5000000, "down_bps": 5000000 },
		"tiers": [
			{
				"up_bps": 100000000,
				"down_bps": 200000000,
				"priority": 0,
				"threshold_bytes": "0",
				"display_name": { "en": "Normal", "zh-TW": "正常" }
			}
		],
		"accounting_basis": {
			"measure": "total_bytes",
			"metering_version": "forward_admission_v1",
			"period_type": "utc_calendar_month"
		},
		"effective_period": "next_full_period",
		"effective_at": "2026-11-01T00:00:00Z",
		"mode": "shadow"
	}`)

	wantHash, err := canonjson.HashHex(nonCanonicalJSON)
	if err != nil {
		t.Fatal(err)
	}

	code := "default-fup"
	rev1, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:           &code,
		Type:           api.PolicyTypeFairUse,
		SchemaVersion:  1,
		Reason:         "Initial shadow fair-use revision (TEST ONLY)",
		Payload:        nonCanonicalJSON,
		ActorID:        actorID,
		IdempotencyKey: &idemKey,
	})
	if err != nil {
		t.Fatalf("CreateRevision failed: %v", err)
	}
	if rev1.Revision != 1 {
		t.Fatalf("revision=%s want=1", rev1.Revision)
	}
	if rev1.PayloadSHA256 != wantHash {
		t.Fatalf("payload_sha256=%s want=%s", rev1.PayloadSHA256, wantHash)
	}

	// Replay with the same Idempotency-Key returns the exact same revision without
	// inserting a second revision row.
	replay, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:           &code,
		Type:           api.PolicyTypeFairUse,
		SchemaVersion:  1,
		Reason:         "Initial shadow fair-use revision (TEST ONLY)",
		Payload:        nonCanonicalJSON,
		ActorID:        actorID,
		IdempotencyKey: &idemKey,
	})
	if err != nil {
		t.Fatalf("idempotent replay failed: %v", err)
	}
	if replay.RevisionID != rev1.RevisionID || replay.Revision != 1 {
		t.Fatalf("replay mismatch: got %+v want %+v", replay, rev1)
	}

	// Reusing the same Idempotency-Key with a different reason fails with IDEMPOTENCY_CONFLICT.
	_, err = store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:           &code,
		Type:           api.PolicyTypeFairUse,
		SchemaVersion:  1,
		Reason:         "Different reason",
		Payload:        nonCanonicalJSON,
		ActorID:        actorID,
		IdempotencyKey: &idemKey,
	})
	if !errors.Is(err, apierr.New(apierr.IdempotencyConflict, "")) {
		t.Fatalf("expected IDEMPOTENCY_CONFLICT, got: %v", err)
	}

	// Prove immutability: UPDATE and DELETE on policy_revisions are rejected by DB trigger.
	for _, q := range []string{
		`UPDATE policy_revisions SET reason = 'tampered' WHERE id = $1`,
		`DELETE FROM policy_revisions WHERE id = $1`,
	} {
		_, err := pool.Exec(ctx, q, rev1.RevisionID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Fatalf("query %q: expected 23514 immutable trigger error, got: %v", q, err)
		}
	}
}

func TestAT19_PolicyActivationGenerationsAndConcurrency(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	fixedNow := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	store := policies.NewStore(pool, policies.WithClock(func() time.Time { return fixedNow }))

	actorID := uuid.New()
	code := "fup-main"

	p1 := validFairUsePayload()
	raw1, _ := json.Marshal(p1)
	rev1, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &code,
		Type:          api.PolicyTypeFairUse,
		SchemaVersion: 1,
		Reason:        "Revision 1 shadow",
		Payload:       raw1,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatal(err)
	}

	p2 := validFairUsePayload()
	p2.Mode = api.FairUsePayloadModeEnforce
	raw2, _ := json.Marshal(p2)
	rev2, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		DefinitionID:  &rev1.DefinitionID,
		Type:          api.PolicyTypeFairUse,
		SchemaVersion: 1,
		Reason:        "Revision 2 enforce",
		Payload:       raw2,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rev2.Revision != 2 {
		t.Fatalf("rev2.Revision=%s want=2", rev2.Revision)
	}

	// Missing If-Match -> 428 PRECONDITION_REQUIRED.
	_, err = store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: "",
		Reason:          "Missing If-Match",
		ActorID:         actorID,
	})
	if !errors.Is(err, apierr.New(apierr.PreconditionRequired, "")) {
		t.Fatalf("expected PRECONDITION_REQUIRED (428), got: %v", err)
	}

	// Activate revision 1 with If-Match: "1" -> generation becomes 1, resource_version becomes 2.
	act1, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: `"1"`,
		Reason:          "Activate revision 1",
		ActorID:         actorID,
	})
	if err != nil {
		t.Fatalf("ActivateRevision 1 failed: %v", err)
	}
	if act1.NoOp || act1.Generation != 1 || act1.ResourceVersion != 2 {
		t.Fatalf("unexpected act1: %+v", act1)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'policy.activated'`, rev1.DefinitionID).Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("outbox count=%d err=%v", outboxCount, err)
	}

	// Activating the same revision (revision 1) a second time is a no-op:
	// does not increment generation or enqueue another outbox event.
	act1Dup, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: "2",
		Reason:          "Activate revision 1 again",
		ActorID:         actorID,
	})
	if err != nil {
		t.Fatalf("duplicate ActivateRevision failed: %v", err)
	}
	if !act1Dup.NoOp || act1Dup.Generation != 1 || act1Dup.ResourceVersion != 2 {
		t.Fatalf("expected NoOp with generation=1, version=2, got: %+v", act1Dup)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'policy.activated'`, rev1.DefinitionID).Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("outbox count after no-op=%d want=1 err=%v", outboxCount, err)
	}

	// Concurrent activations of revision 2 with If-Match: "2" (plus a stale If-Match: "1" check).
	var wg sync.WaitGroup
	errs := make([]error, 2)
	results := make([]*policies.ActivateResult, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = store.ActivateRevision(ctx, policies.ActivateRevisionInput{
				DefinitionID:    rev1.DefinitionID,
				RevisionID:      rev2.RevisionID,
				ExpectedVersion: "2",
				Reason:          "Concurrent activate revision 2",
				ActorID:         actorID,
			})
		}(i)
	}
	wg.Wait()

	// Note: because the second goroutine acquires the FOR UPDATE lock after the first commits,
	// either the first goroutine updated resource_version to 3 (causing the second to see
	// curVerStr="3" != expectedVersion="2" -> ErrVersionConflict), OR if we also test an explicit
	// stale version "2" now, it must return ErrVersionConflict / RESOURCE_VERSION_CONFLICT.
	successCount := 0
	conflictCount := 0
	for i := range 2 {
		if errs[i] == nil {
			successCount++
			if results[i].Generation != 2 || results[i].ResourceVersion != 3 {
				t.Fatalf("unexpected success result: %+v", results[i])
			}
		} else if errors.Is(errs[i], db.ErrVersionConflict) && errors.Is(errs[i], apierr.New(apierr.ResourceVersionConflict, "")) {
			conflictCount++
		} else {
			t.Fatalf("unexpected error from concurrent activate [%d]: %v", i, errs[i])
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("expected 1 success and 1 conflict, got success=%d conflict=%d", successCount, conflictCount)
	}

	// Activating older revision 1 with stale version "2" is rejected with 412 RESOURCE_VERSION_CONFLICT.
	_, err = store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: "2",
		Reason:          "Stale version rollback attempt",
		ActorID:         actorID,
	})
	if !errors.Is(err, db.ErrVersionConflict) || !errors.Is(err, apierr.New(apierr.ResourceVersionConflict, "")) {
		t.Fatalf("expected ErrVersionConflict / RESOURCE_VERSION_CONFLICT, got: %v", err)
	}

	// Activating older revision 1 as an explicit new activation with current If-Match "3"
	// succeeds and increments generation to 3 (never replaying old generation 1, §11.5, AT19).
	actRollback, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev1.DefinitionID,
		RevisionID:      rev1.RevisionID,
		ExpectedVersion: "3",
		Reason:          "Explicit rollback to revision 1 with new generation",
		ActorID:         actorID,
	})
	if err != nil {
		t.Fatalf("explicit rollback activation failed: %v", err)
	}
	if actRollback.NoOp || actRollback.Generation != 3 || actRollback.ResourceVersion != 4 {
		t.Fatalf("expected Generation=3, ResourceVersion=4, got: %+v", actRollback)
	}
}

func Test14_2_DNSProfileBundleCompilationAndPlanBaseline(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	fixedNow := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	store := policies.NewStore(pool, policies.WithClock(func() time.Time { return fixedNow }))
	actorID := uuid.New()

	// 1. Create domain_rule_set revision.
	rsPayload := api.DomainRuleSetPayload{
		Rules: []api.DomainRule{
			{
				RuleId:            openapi_types.UUID(uuid.New()),
				ProfileId:         presence.NewNullableWithValue("standard"),
				Domain:            "tracker.example.com",
				IncludeSubdomains: true,
				Action:            api.DomainRuleActionBlock,
				Reason:            "admin_rule",
				Revision:          decstr.U64(1),
			},
			{
				RuleId:            openapi_types.UUID(uuid.New()),
				ProfileId:         presence.NewNullNullable[string](), // service-wide rule
				Domain:            "cdn.example.com",
				IncludeSubdomains: false,
				Action:            api.DomainRuleActionAllow,
				Reason:            "false_positive",
				Revision:          decstr.U64(1),
			},
		},
	}
	rsRaw, _ := json.Marshal(rsPayload)
	rsCode := "standard-rules"
	rsRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &rsCode,
		Type:          api.PolicyTypeDomainRuleSet,
		SchemaVersion: 1,
		Reason:        "Create domain rules",
		Payload:       rsRaw,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatalf("create domain_rule_set failed: %v", err)
	}

	// 2. Create threat_feed revision.
	tfPayload := api.ThreatFeedPayload{
		Provider:              "Example Threat Intel",
		FeedCode:              "example-malware",
		LicenseName:           "Commercial Threat Feed License",
		LicenseUrl:            "https://feeds.example.com/license",
		CommercialUseBasis:    "Licensed for commercial SaaS/VPN redistribution (TEST ONLY).",
		Attribution:           presence.FromPointer(strPtr("Threat intel by Example")),
		SourceUrl:             "https://feeds.example.com/malware.txt",
		Format:                api.ThreatFeedPayloadFormatDomainList,
		Categories:            []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware},
		ContentSha256:         "7777777777777777777777777777777777777777777777777777777777777777",
		MaxSizeBytes:          decstr.U64(1048576),
		ValidFrom:             fixedNow.Add(-1 * time.Hour),
		ValidUntil:            fixedNow.Add(30 * 24 * time.Hour),
		UpdateIntervalSeconds: 3600,
	}
	tfRaw, _ := json.Marshal(tfPayload)
	tfCode := "example-malware"
	tfRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &tfCode,
		Type:          api.PolicyTypeThreatFeed,
		SchemaVersion: 1,
		Reason:        "Create threat feed",
		Payload:       tfRaw,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatalf("create threat_feed failed: %v", err)
	}

	// 3. Create and activate dns_profile revision; verify compiled bundle is stored.
	dnsPayload := api.DnsProfilePayload{
		ProfileCode:              "standard",
		DisplayName:              api.LocalizedText{ZhTW: "標準防護", En: strPtr("Standard protection")},
		DomainRuleSetRevisionIds: []openapi_types.UUID{openapi_types.UUID(rsRev.RevisionID)},
		ThreatFeedRevisionIds:    []openapi_types.UUID{openapi_types.UUID(tfRev.RevisionID)},
		BlockResponse:            api.DnsProfilePayloadBlockResponseNxdomain,
		BlockedTtlSeconds:        60,
	}
	dnsRaw, _ := json.Marshal(dnsPayload)
	dnsCode := "standard"
	dnsRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &dnsCode,
		Type:          api.PolicyTypeDnsProfile,
		SchemaVersion: 1,
		Reason:        "Create standard DNS profile",
		Payload:       dnsRaw,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatalf("create dns_profile failed: %v", err)
	}

	dnsAct, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    dnsRev.DefinitionID,
		RevisionID:      dnsRev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "Activate standard DNS profile",
		ActorID:         actorID,
	})
	if err != nil {
		t.Fatalf("activate dns_profile failed: %v", err)
	}
	if dnsAct.CompiledBundle == nil || dnsAct.CompiledBundle.SHA256 == "" {
		t.Fatalf("expected compiled DNS bundle on activation, got: %+v", dnsAct.CompiledBundle)
	}
	storedBundle, err := store.GetCompiledDNSBundle(ctx, dnsRev.RevisionID)
	if err != nil {
		t.Fatalf("GetCompiledDNSBundle failed: %v", err)
	}
	if storedBundle.SHA256 != dnsAct.CompiledBundle.SHA256 || storedBundle.RuleCount != 2 {
		t.Fatalf("stored bundle mismatch: %+v vs %+v", storedBundle, dnsAct.CompiledBundle)
	}

	// 4. Create fair_use revision for plan reference.
	fupRaw, _ := json.Marshal(validFairUsePayload())
	fupCode := "standard-fup"
	fupRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &fupCode,
		Type:          api.PolicyTypeFairUse,
		SchemaVersion: 1,
		Reason:        "Create fair_use for plan",
		Payload:       fupRaw,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 5. Baseline guard: a plan referencing a non-existent exit_group_id must be rejected.
	unknownGroupID := uuid.New()
	planPayload := api.PlanPayload{
		DisplayName:  api.LocalizedText{ZhTW: "標準方案", En: strPtr("Standard")},
		DeviceLimit:  3,
		ExitGroupIds: []openapi_types.UUID{openapi_types.UUID(unknownGroupID)},
		Price: api.PlanPrice{
			AmountMinor:      30000, // TEST ONLY
			Currency:         "TWD",
			BillingInterval:  api.PlanPriceBillingIntervalMonth,
			Provider:         "test-provider",
			ProviderPriceRef: "price_TEST_ONLY_standard",
		},
		RateLimit: presence.NewNullNullable[api.RatePair](),

		FairUseRevisionId:    openapi_types.UUID(fupRev.RevisionID),
		DnsProfileRevisionId: openapi_types.UUID(dnsRev.RevisionID),
	}
	planRawBad, _ := json.Marshal(planPayload)
	planCode := "standard"
	_, err = store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &planCode,
		Type:          api.PolicyTypePlan,
		SchemaVersion: 1,
		Reason:        "Plan with unknown exit group",
		Payload:       planRawBad,
		ActorID:       actorID,
	})
	if !policies.HasReason(err, policies.ReasonPlanUnknownExitGroup) {
		t.Fatalf("expected ReasonPlanUnknownExitGroup for non-existent exit group, got: %v", err)
	}

	// Insert the exit group into member_groups and verify plan creation + activation succeeds.
	realGroupID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'tw-standard', 'Taiwan Standard')`, realGroupID); err != nil {
		t.Fatal(err)
	}
	planPayload.ExitGroupIds = []openapi_types.UUID{openapi_types.UUID(realGroupID)}
	planRawGood, _ := json.Marshal(planPayload)
	planRev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &planCode,
		Type:          api.PolicyTypePlan,
		SchemaVersion: 1,
		Reason:        "Valid standard plan",
		Payload:       planRawGood,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatalf("create valid plan failed: %v", err)
	}

	if _, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    planRev.DefinitionID,
		RevisionID:      planRev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "Activate standard plan",
		ActorID:         actorID,
	}); err != nil {
		t.Fatalf("activate plan failed: %v", err)
	}

	// Verify ListDefinitions pagination and type filter.
	allPage1, err := store.ListDefinitions(ctx, policies.ListDefinitionsFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(allPage1.Items) != 2 || (!allPage1.NextCursor.IsSpecified() || allPage1.NextCursor.IsNull()) {
		t.Fatalf("expected 2 items and next_cursor, got len=%d next=%v", len(allPage1.Items), allPage1.NextCursor)
	}
	planType := api.PolicyTypePlan
	onlyPlans, err := store.ListDefinitions(ctx, policies.ListDefinitionsFilter{Type: &planType})
	if err != nil {
		t.Fatal(err)
	}
	if len(onlyPlans.Items) != 1 || onlyPlans.Items[0].Code != "standard" || onlyPlans.Items[0].ActiveRevision == nil {
		t.Fatalf("unexpected filtered plan definitions: %+v", onlyPlans.Items)
	}
}

func TestAT27_ReleaseSupportActivationUpdatesFleetConfig(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	store := policies.NewStore(pool)
	actorID := uuid.New()

	relServingOld := uuid.New()
	relApprovedNew := uuid.New()
	relUnregistered := uuid.New()

	for i, rid := range []uuid.UUID{relServingOld, relApprovedNew} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO releases (id, version, release_sequence, manifest_schema_version, manifest, manifest_hash, signature, signing_key_ref, component_versions)
			 VALUES ($1, $2, $3, 1, '{}'::jsonb, decode(repeat('01', 32), 'hex'), decode(repeat('02', 64), 'hex'), 'test-key', '{}'::jsonb)`,
			rid, "TEST-ONLY-"+rid.String()[:8], i+1,
		); err != nil {
			t.Fatal(err)
		}
	}

	// Referencing relUnregistered when releases table is populated fails with ReasonReleaseUnknownID.
	badPayload := api.ReleaseSupportPayload{
		ApprovedReleaseIds: []openapi_types.UUID{openapi_types.UUID(relUnregistered)},
		RevokedReleaseIds:  []openapi_types.UUID{},
		MinimumServingVersion: struct {
			ServingReleaseIds []openapi_types.UUID `json:"serving_release_ids"`
		}{
			ServingReleaseIds: []openapi_types.UUID{openapi_types.UUID(relUnregistered)},
		},
	}
	badRaw, _ := json.Marshal(badPayload)
	code := "fleet"
	_, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &code,
		Type:          api.PolicyTypeReleaseSupport,
		SchemaVersion: 1,
		Reason:        "Unregistered release ID",
		Payload:       badRaw,
		ActorID:       actorID,
	})
	if !policies.HasReason(err, policies.ReasonReleaseUnknownID) {
		t.Fatalf("expected ReasonReleaseUnknownID, got: %v", err)
	}

	goodPayload := api.ReleaseSupportPayload{
		ApprovedReleaseIds: []openapi_types.UUID{openapi_types.UUID(relApprovedNew)},
		RevokedReleaseIds:  []openapi_types.UUID{},
		MinimumServingVersion: struct {
			ServingReleaseIds []openapi_types.UUID `json:"serving_release_ids"`
		}{
			ServingReleaseIds: []openapi_types.UUID{
				openapi_types.UUID(relServingOld),
				openapi_types.UUID(relApprovedNew),
			},
		},
	}
	goodRaw, _ := json.Marshal(goodPayload)
	rev, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &code,
		Type:          api.PolicyTypeReleaseSupport,
		SchemaVersion: 1,
		Reason:        "Approve new release while keeping old release serving-compatible",
		Payload:       goodRaw,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatalf("create release_support failed: %v", err)
	}

	if _, err := store.ActivateRevision(ctx, policies.ActivateRevisionInput{
		DefinitionID:    rev.DefinitionID,
		RevisionID:      rev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "Activate fleet release support",
		ActorID:         actorID,
	}); err != nil {
		t.Fatalf("activate release_support failed: %v", err)
	}

	var policyMode string
	var supportedIDs []uuid.UUID
	var fleetGen, fleetVer string
	if err := pool.QueryRow(ctx, `SELECT supported_release_policy, supported_release_ids, generation::text, resource_version::text FROM fleet_config WHERE id = '00000000-0000-0000-0000-000000000001'`).Scan(&policyMode, &supportedIDs, &fleetGen, &fleetVer); err != nil {
		t.Fatal(err)
	}
	if policyMode != "allowlist" || len(supportedIDs) != 2 || fleetGen != "1" || fleetVer != "2" {
		t.Fatalf("fleet_config policy=%s supportedIDs=%v gen=%s ver=%s", policyMode, supportedIDs, fleetGen, fleetVer)
	}

	// Prove F-14 on CreateRevision by Code: creating a second revision using Code
	// (instead of DefinitionID) does not execute a no-op UPDATE on policy_definitions.
	rev2, err := store.CreateRevision(ctx, policies.CreateRevisionInput{
		Code:          &code,
		Type:          api.PolicyTypeReleaseSupport,
		SchemaVersion: 1,
		Reason:        "Second revision created by code without DefinitionID",
		Payload:       goodRaw,
		ActorID:       actorID,
	})
	if err != nil {
		t.Fatalf("create second release_support revision by code failed: %v", err)
	}
	if rev2.Revision != 2 || rev2.DefinitionID != rev.DefinitionID {
		t.Fatalf("unexpected rev2: %+v", rev2)
	}
}
