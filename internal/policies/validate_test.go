package policies_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func strPtr(s string) *string { return &s }

func validFairUsePayload() api.FairUsePayload {
	// TEST ONLY - NOT A PRODUCTION VALUE (§11.1, §21: thresholds and rates are pending business decisions).
	return api.FairUsePayload{
		Mode:            api.FairUsePayloadModeShadow,
		EffectiveAt:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		EffectivePeriod: api.FairUsePayloadEffectivePeriodNextFullPeriod,
		AccountingBasis: struct {
			Measure         api.FairUsePayloadAccountingBasisMeasure    `json:"measure"`
			MeteringVersion api.MeteringVersion                         `json:"metering_version"`
			PeriodType      api.FairUsePayloadAccountingBasisPeriodType `json:"period_type"`
		}{
			PeriodType:      api.FairUsePayloadAccountingBasisPeriodTypeUtcCalendarMonth,
			MeteringVersion: api.MeteringVersionForwardAdmissionV1,
			Measure:         api.FairUsePayloadAccountingBasisMeasureTotalBytes,
		},
		Tiers: []api.FairUseTier{
			{
				ThresholdBytes: 0,
				DownBps:        200_000_000, // TEST ONLY
				UpBps:          100_000_000, // TEST ONLY
				Priority:       0,
				DisplayName:    api.LocalizedText{ZhTW: "正常速度", En: strPtr("Normal speed")},
			},
			{
				ThresholdBytes: 500_000_000_000,
				DownBps:        50_000_000, // TEST ONLY
				UpBps:          25_000_000, // TEST ONLY
				Priority:       1,
				DisplayName:    api.LocalizedText{ZhTW: "公平使用調整中", En: strPtr("Fair-use adjusted")},
			},
			{
				ThresholdBytes: 1_000_000_000_000,
				DownBps:        20_000_000, // TEST ONLY
				UpBps:          10_000_000, // TEST ONLY
				Priority:       2,
				DisplayName:    api.LocalizedText{ZhTW: "公平使用調整中", En: strPtr("Fair-use adjusted")},
			},
			{
				ThresholdBytes: 2_000_000_000_000,
				DownBps:        5_000_000, // TEST ONLY
				UpBps:          5_000_000, // TEST ONLY
				Priority:       3,
				DisplayName:    api.LocalizedText{ZhTW: "基本速度", En: strPtr("Basic speed")},
			},
		},
		MinimumRate: api.RatePair{
			DownBps: 5_000_000, // TEST ONLY
			UpBps:   5_000_000, // TEST ONLY
		},
		Overrides: []api.FairUseOverride{
			{
				MemberId:   openapi_types.UUID(uuid.MustParse("4e8ca24f-b523-45ef-a782-068cc1c9f809")),
				DownBps:    50_000_000, // TEST ONLY
				UpBps:      25_000_000, // TEST ONLY
				Priority:   1,
				ValidUntil: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
				Reason:     "TEST ONLY support relief",
				ActorId:    openapi_types.UUID(uuid.MustParse("6a7d5389-5347-41a4-9cd7-768e07072254")),
			},
		},
	}
}

func TestFUP01_FairUseValidationRules(t *testing.T) {
	refTime := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	opts := policies.ValidationOptions{ReferenceTime: refTime}

	t.Run("passing policy", func(t *testing.T) {
		p := validFairUsePayload()
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		v, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if err != nil {
			t.Fatalf("expected valid fair_use policy, got: %v", err)
		}
		wantHash, err := canonjson.HashHex(raw)
		if err != nil {
			t.Fatal(err)
		}
		if v.PayloadSHA256 != wantHash {
			t.Fatalf("payload_sha256=%s want=%s", v.PayloadSHA256, wantHash)
		}
	})

	t.Run("missing tier (empty tiers)", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers = nil
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseEmptyTiers) {
			t.Fatalf("expected ReasonFairUseEmptyTiers, got: %v", err)
		}
	})

	t.Run("non-zero first threshold", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[0].ThresholdBytes = 100
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseTierZeroThreshold) {
			t.Fatalf("expected ReasonFairUseTierZeroThreshold, got: %v", err)
		}
	})

	t.Run("non-increasing thresholds", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[2].ThresholdBytes = p.Tiers[1].ThresholdBytes
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseThresholdOrder) {
			t.Fatalf("expected ReasonFairUseThresholdOrder, got: %v", err)
		}
	})

	t.Run("negative rate", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[0].DownBps = -1
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseNegativeRate) {
			t.Fatalf("expected ReasonFairUseNegativeRate, got: %v", err)
		}
	})

	t.Run("non-decreasing rate (higher tier faster)", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[2].UpBps = p.Tiers[1].UpBps + 1
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseRateOrder) {
			t.Fatalf("expected ReasonFairUseRateOrder, got: %v", err)
		}
	})

	t.Run("decreasing priority across tiers", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[2].Priority = 0 // lower than tier 1 priority (1)
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUsePriorityOrder) {
			t.Fatalf("expected ReasonFairUsePriorityOrder, got: %v", err)
		}
	})

	t.Run("rate below minimum_rate", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[3].DownBps = p.MinimumRate.DownBps - 1
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseBelowMinimumRate) {
			t.Fatalf("expected ReasonFairUseBelowMinimumRate, got: %v", err)
		}
	})

	t.Run("missing direction in tier", func(t *testing.T) {
		p := validFairUsePayload()
		raw, _ := json.Marshal(p)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		tiers := m["tiers"].([]any)
		tier0 := tiers[0].(map[string]any)
		delete(tier0, "up_bps")
		mutated, _ := json.Marshal(m)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, mutated, opts)
		if !policies.HasReason(err, policies.ReasonFairUseMissingDirection) {
			t.Fatalf("expected ReasonFairUseMissingDirection, got: %v", err)
		}
	})

	t.Run("duplicate override member", func(t *testing.T) {
		p := validFairUsePayload()
		p.Overrides = append(p.Overrides, p.Overrides[0])
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseDuplicateOverride) {
			t.Fatalf("expected ReasonFairUseDuplicateOverride, got: %v", err)
		}
	})

	t.Run("expired override", func(t *testing.T) {
		p := validFairUsePayload()
		p.Overrides[0].ValidUntil = refTime.Add(-1 * time.Second)
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, raw, opts)
		if !policies.HasReason(err, policies.ReasonFairUseExpiredOverride) {
			t.Fatalf("expected ReasonFairUseExpiredOverride, got: %v", err)
		}
	})
}

func TestAT27_ReleaseSupportExplicitSetMembership(t *testing.T) {
	rel1 := uuid.MustParse("cdb25ffa-5395-400c-ad3e-b32899f14bb8")
	rel2 := uuid.MustParse("6d2797d0-b9a5-4724-96af-9b3039d351b1")
	relRevoked := uuid.MustParse("11111111-2222-4333-8444-555555555555")
	relUnknownNewerVersion := uuid.MustParse("ffffffff-ffff-4fff-bfff-ffffffffffff")

	validPayload := api.ReleaseSupportPayload{
		ApprovedReleaseIds: []openapi_types.UUID{openapi_types.UUID(rel2)},
		RevokedReleaseIds:  []openapi_types.UUID{openapi_types.UUID(relRevoked)},
		MinimumServingVersion: struct {
			ServingReleaseIds []openapi_types.UUID `json:"serving_release_ids"`
		}{
			ServingReleaseIds: []openapi_types.UUID{
				openapi_types.UUID(rel1),
				openapi_types.UUID(rel2),
			},
		},
	}

	ev, err := policies.NewReleaseSupportEvaluator(validPayload)
	if err != nil {
		t.Fatalf("expected valid release_support payload, got: %v", err)
	}

	// rel2 is both approved and serving.
	if err := ev.CheckServingAllowed(rel2); err != nil {
		t.Fatalf("rel2 should be allowed to serve: %v", err)
	}
	if err := ev.CheckUpdateTargetAllowed(rel2); err != nil {
		t.Fatalf("rel2 should be allowed as update target: %v", err)
	}
	if got := ev.SupportState(rel2); got != api.ReleaseSupportStateApproved {
		t.Fatalf("rel2 state=%s want=%s", got, api.ReleaseSupportStateApproved)
	}

	// rel1 is in serving_release_ids but not approved_release_ids: allowed to keep serving,
	// rejected as a new update target (AT27).
	if err := ev.CheckServingAllowed(rel1); err != nil {
		t.Fatalf("rel1 should be allowed to serve: %v", err)
	}
	if err := ev.CheckUpdateTargetAllowed(rel1); !errors.Is(err, apierr.New(apierr.ReleaseNotApproved, "")) {
		t.Fatalf("rel1 should fail CheckUpdateTargetAllowed with RELEASE_NOT_APPROVED, got: %v", err)
	}
	if got := ev.SupportState(rel1); got != api.ReleaseSupportStateNotApproved {
		t.Fatalf("rel1 state=%s want=%s", got, api.ReleaseSupportStateNotApproved)
	}

	// relRevoked is rejected for both serving and update targeting.
	if err := ev.CheckServingAllowed(relRevoked); !errors.Is(err, apierr.New(apierr.ReleaseNotApproved, "")) {
		t.Fatalf("revoked release should fail CheckServingAllowed, got: %v", err)
	}
	if err := ev.CheckUpdateTargetAllowed(relRevoked); !errors.Is(err, apierr.New(apierr.ReleaseNotApproved, "")) {
		t.Fatalf("revoked release should fail CheckUpdateTargetAllowed, got: %v", err)
	}
	if got := ev.SupportState(relRevoked); got != api.ReleaseSupportStateRevoked {
		t.Fatalf("relRevoked state=%s want=%s", got, api.ReleaseSupportStateRevoked)
	}

	// relUnknownNewerVersion is not in serving_release_ids: even if its UUID or product version
	// string sorts higher lexically, explicit set membership rejects it.
	if err := ev.CheckServingAllowed(relUnknownNewerVersion); !errors.Is(err, apierr.New(apierr.MinimumVersionViolation, "")) {
		t.Fatalf("unlisted release should fail CheckServingAllowed with MINIMUM_VERSION_VIOLATION, got: %v", err)
	}

	// Approved not a subset of serving -> rejected.
	badSubset := validPayload
	badSubset.ApprovedReleaseIds = []openapi_types.UUID{openapi_types.UUID(relUnknownNewerVersion)}
	if err := policies.ValidateReleaseSupport(badSubset); !policies.HasReason(err, policies.ReasonReleaseApprovedNotInServing) {
		t.Fatalf("expected ReasonReleaseApprovedNotInServing, got: %v", err)
	}

	// Revoked overlaps serving -> rejected.
	badOverlap := validPayload
	badOverlap.RevokedReleaseIds = []openapi_types.UUID{openapi_types.UUID(rel1)}
	if err := policies.ValidateReleaseSupport(badOverlap); !policies.HasReason(err, policies.ReasonReleaseRevokedOverlap) {
		t.Fatalf("expected ReasonReleaseRevokedOverlap, got: %v", err)
	}

	// Empty serving_release_ids -> rejected.
	emptyServing := validPayload
	emptyServing.MinimumServingVersion.ServingReleaseIds = nil
	emptyServing.ApprovedReleaseIds = nil
	if err := policies.ValidateReleaseSupport(emptyServing); !policies.HasReason(err, policies.ReasonReleaseServingEmpty) {
		t.Fatalf("expected ReasonReleaseServingEmpty, got: %v", err)
	}
}

func TestISO01_DNSIsolationBaselineAndConflicts(t *testing.T) {
	// Baseline guard: attempting to allow private/reserved zones or IP literals in domain_rule_set is rejected.
	baselineCases := []string{
		"localhost",
		"sub.localhost",
		"printer.local",
		"metadata.google.internal",
		"router.home.arpa",
		"10.in-addr.arpa",
		"168.192.in-addr.arpa",
		"d.f.ip6.arpa",
		"10.0.0.1",
		"100.64.0.1",
		"192.168.1.0/24",
	}
	for _, badDomain := range baselineCases {
		t.Run("baseline_"+badDomain, func(t *testing.T) {
			p := api.DomainRuleSetPayload{
				Rules: []api.DomainRule{
					{
						RuleId:            openapi_types.UUID(uuid.New()),
						ProfileId:         presence.NewNullableWithValue("standard"),
						Domain:            badDomain,
						IncludeSubdomains: true,
						Action:            api.DomainRuleActionAllow,
						Reason:            "attempt to bypass baseline",
						Revision:          decstr.U64(1),
					},
				},
			}
			raw, _ := json.Marshal(p)
			_, err := policies.ValidatePayload(api.PolicyTypeDomainRuleSet, 1, raw, policies.ValidationOptions{})
			if !policies.HasReason(err, policies.ReasonIsolationBaselineViolation) {
				t.Fatalf("domain %q: expected ReasonIsolationBaselineViolation, got: %v", badDomain, err)
			}
		})
	}

	t.Run("IDNA normalization conflict detection", func(t *testing.T) {
		// "bücher.example.com" normalizes to "xn--bcher-kva.example.com".
		p := api.DomainRuleSetPayload{
			Rules: []api.DomainRule{
				{
					RuleId:            openapi_types.UUID(uuid.New()),
					ProfileId:         presence.NewNullableWithValue("standard"),
					Domain:            "xn--bcher-kva.example.com",
					IncludeSubdomains: true,
					Action:            api.DomainRuleActionBlock,
					Reason:            "block punycode",
					Revision:          decstr.U64(1),
				},
				{
					RuleId:            openapi_types.UUID(uuid.New()),
					ProfileId:         presence.NewNullableWithValue("standard"),
					Domain:            "bücher.example.com",
					IncludeSubdomains: false,
					Action:            api.DomainRuleActionAllow,
					Reason:            "conflicting unicode rule",
					Revision:          decstr.U64(1),
				},
			},
		}
		raw, _ := json.Marshal(p)
		_, err := policies.ValidatePayload(api.PolicyTypeDomainRuleSet, 1, raw, policies.ValidationOptions{})
		if !policies.HasReason(err, policies.ReasonDNSSameLayerConflict) {
			t.Fatalf("expected ReasonDNSSameLayerConflict for IDNA-equivalent domains, got: %v", err)
		}
	})

	t.Run("duplicate rule_id rejected", func(t *testing.T) {
		dupID := openapi_types.UUID(uuid.New())
		p := api.DomainRuleSetPayload{
			Rules: []api.DomainRule{
				{
					RuleId:            dupID,
					ProfileId:         presence.NewNullableWithValue("standard"),
					Domain:            "one.example.com",
					IncludeSubdomains: true,
					Action:            api.DomainRuleActionBlock,
					Reason:            "first",
					Revision:          decstr.U64(1),
				},
				{
					RuleId:            dupID,
					ProfileId:         presence.NewNullableWithValue("standard"),
					Domain:            "two.example.com",
					IncludeSubdomains: true,
					Action:            api.DomainRuleActionBlock,
					Reason:            "second",
					Revision:          decstr.U64(1),
				},
			},
		}
		if err := policies.ValidateDomainRuleSet(p); !policies.HasReason(err, policies.ReasonDNSDuplicateRuleID) {
			t.Fatalf("expected ReasonDNSDuplicateRuleID, got: %v", err)
		}
	})
}

func TestContractExamplesValidate(t *testing.T) {
	validFiles := []string{
		"PolicyRevisionCreate.fair-use.json",
		"PolicyRevisionCreate.plan.json",
		"PolicyRevisionCreate.dns-profile.json",
		"PolicyRevisionCreate.domain-rule-set.json",
		"PolicyRevisionCreate.threat-feed.json",
		"PolicyRevisionCreate.release-support.json",
	}
	for _, name := range validFiles {
		t.Run("valid_"+name, func(t *testing.T) {
			path := filepath.Join("..", "..", "contracts", "examples", "valid", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var env struct {
				Type          api.PolicyType  `json:"type"`
				SchemaVersion int             `json:"schema_version"`
				Payload       json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal(raw, &env); err != nil {
				t.Fatal(err)
			}
			if _, err := policies.ValidatePayload(env.Type, env.SchemaVersion, env.Payload, policies.ValidationOptions{}); err != nil {
				t.Fatalf("contract valid example %s failed validation: %v", name, err)
			}
		})
	}

	invalidFiles := []string{
		"PolicyRevisionCreate.domain-rule-not-canonical.json",
		"PolicyRevisionCreate.domain-rule-url.json",
		"PolicyRevisionCreate.fair-use-numeric-threshold.json",
		"PolicyRevisionCreate.release-support-empty-serving.json",
		"PolicyRevisionCreate.type-payload-mismatch.json",
	}
	for _, name := range invalidFiles {
		t.Run("invalid_"+name, func(t *testing.T) {
			path := filepath.Join("..", "..", "contracts", "examples", "invalid", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var env struct {
				Type          api.PolicyType  `json:"type"`
				SchemaVersion int             `json:"schema_version"`
				Payload       json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal(raw, &env); err != nil {
				t.Fatal(err)
			}
			if _, err := policies.ValidatePayload(env.Type, env.SchemaVersion, env.Payload, policies.ValidationOptions{}); err == nil {
				t.Fatalf("contract invalid example %s unexpectedly passed validation", name)
			}
		})
	}
}
