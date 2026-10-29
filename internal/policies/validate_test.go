package policies_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gopkg.in/yaml.v3"
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
						Reason:            "admin_rule",
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
					Reason:            "admin_rule",
					Revision:          decstr.U64(1),
				},
				{
					RuleId:            openapi_types.UUID(uuid.New()),
					ProfileId:         presence.NewNullableWithValue("standard"),
					Domain:            "bücher.example.com",
					IncludeSubdomains: false,
					Action:            api.DomainRuleActionAllow,
					Reason:            "false_positive",
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

func TestAPI01_PromotedPolicySchemasByteIdenticalAndReconciled(t *testing.T) {
	var oapiDoc struct {
		Components struct {
			Schemas map[string]map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(contracts.OpenAPI(), &oapiDoc); err != nil {
		t.Fatalf("unmarshal OpenAPI: %v", err)
	}
	comps := oapiDoc.Components.Schemas

	const (
		uint64Pattern    = "^(0|[1-9][0-9]{0,18}|1[0-7][0-9]{18}|18[0-3][0-9]{17}|184[0-3][0-9]{16}|1844[0-5][0-9]{15}|18446[0-6][0-9]{14}|184467[0-3][0-9]{13}|1844674[0-3][0-9]{12}|184467440[0-6][0-9]{10}|1844674407[0-2][0-9]{9}|18446744073[0-6][0-9]{8}|1844674407370[0-8][0-9]{6}|18446744073709[0-4][0-9]{5}|184467440737095[0-4][0-9]{4}|18446744073709550[0-9]{3}|18446744073709551[0-5][0-9]{2}|1844674407370955160[0-9]|1844674407370955161[0-5])$"
		posUint64Pattern = "^([1-9][0-9]{0,18}|1[0-7][0-9]{18}|18[0-3][0-9]{17}|184[0-3][0-9]{16}|1844[0-5][0-9]{15}|18446[0-6][0-9]{14}|184467[0-3][0-9]{13}|1844674[0-3][0-9]{12}|184467440[0-6][0-9]{10}|1844674407[0-2][0-9]{9}|18446744073[0-6][0-9]{8}|1844674407370[0-8][0-9]{6}|18446744073709[0-4][0-9]{5}|184467440737095[0-4][0-9]{4}|18446744073709550[0-9]{3}|18446744073709551[0-5][0-9]{2}|1844674407370955160[0-9]|1844674407370955161[0-5])$"
	)

	kinds := []string{"plan", "fair-use", "dns-profile", "domain-rule-set", "threat-feed", "release-support"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			file := kind + "-payload.schema.json"
			ownerDisk, err := os.ReadFile(filepath.Join("schemas", file))
			if err != nil {
				t.Fatalf("read owner schema %s: %v", file, err)
			}
			ownerEmbedded, err := fs.ReadFile(policies.SchemasFS(), "schemas/"+file)
			if err != nil {
				t.Fatalf("read embedded owner schema %s: %v", file, err)
			}
			promotedDisk, err := os.ReadFile(filepath.Join("..", "..", "contracts", "schemas", "policies", file))
			if err != nil {
				t.Fatalf("read promoted schema %s: %v", file, err)
			}
			promotedEmbedded, err := fs.ReadFile(contracts.Schemas(), "policies/"+file)
			if err != nil {
				t.Fatalf("read embedded promoted schema %s: %v", file, err)
			}
			if !bytes.Equal(ownerDisk, ownerEmbedded) {
				t.Fatalf("embedded owner schema %s differs from disk", file)
			}
			if !bytes.Equal(ownerDisk, promotedDisk) {
				t.Fatalf("owner schema %s and promoted contracts/schemas/policies/%s drifted", file, file)
			}
			if !bytes.Equal(promotedDisk, promotedEmbedded) {
				t.Fatalf("embedded promoted schema %s differs from disk", file)
			}

			var js map[string]any
			if err := json.Unmarshal(ownerDisk, &js); err != nil {
				t.Fatalf("unmarshal JSON Schema %s: %v", file, err)
			}
			title, _ := js["title"].(string)
			comp, ok := comps[title]
			if !ok {
				t.Fatalf("missing OpenAPI component %q", title)
			}
			defs, _ := js["$defs"].(map[string]any)
			normOpenAPI, err := normalizeSchemaForCompare(comp, func(ref string) (map[string]any, error) {
				name, ok := strings.CutPrefix(ref, "#/components/schemas/")
				if !ok || comps[name] == nil {
					return nil, fmt.Errorf("bad OpenAPI ref %q", ref)
				}
				return comps[name], nil
			})
			if err != nil {
				t.Fatalf("normalize OpenAPI %s: %v", title, err)
			}
			normJSONSchema, err := normalizeSchemaForCompare(js, func(ref string) (map[string]any, error) {
				name, ok := strings.CutPrefix(ref, "#/$defs/")
				if !ok || defs[name] == nil {
					return nil, fmt.Errorf("bad JSON Schema ref %q", ref)
				}
				return defs[name].(map[string]any), nil
			})
			if err != nil {
				t.Fatalf("normalize JSON Schema %s: %v", file, err)
			}

			// Documented intentional tighter owner decimal-string caps (D-044 / step 1 & 3):
			switch kind {
			case "fair-use":
				tb := normJSONSchema["properties"].(map[string]any)["tiers"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["threshold_bytes"].(map[string]any)
				if tb["pattern"] != uint64Pattern {
					t.Fatalf("fair-use threshold_bytes pattern = %v, want exact uint64 pattern", tb["pattern"])
				}
				normOpenAPI["properties"].(map[string]any)["tiers"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["threshold_bytes"].(map[string]any)["pattern"] = uint64Pattern
			case "domain-rule-set":
				rev := normJSONSchema["properties"].(map[string]any)["rules"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["revision"].(map[string]any)
				if rev["pattern"] != posUint64Pattern {
					t.Fatalf("domain-rule-set revision pattern = %v, want positive uint64 pattern", rev["pattern"])
				}
				normOpenAPI["properties"].(map[string]any)["rules"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["revision"].(map[string]any)["pattern"] = posUint64Pattern
			case "threat-feed":
				ms := normJSONSchema["properties"].(map[string]any)["max_size_bytes"].(map[string]any)
				if ms["pattern"] != uint64Pattern {
					t.Fatalf("threat-feed max_size_bytes pattern = %v, want exact uint64 pattern", ms["pattern"])
				}
				normOpenAPI["properties"].(map[string]any)["max_size_bytes"].(map[string]any)["pattern"] = uint64Pattern
			}

			if diff := cmp.Diff(normOpenAPI, normJSONSchema); diff != "" {
				t.Fatalf("reconciled JSON Schema %s differs from OpenAPI %s (-openapi +jsonschema):\n%s", file, title, diff)
			}
		})
	}
}

func normalizeSchemaForCompare(s map[string]any, resolve func(string) (map[string]any, error)) (map[string]any, error) {
	if ref, ok := s["$ref"].(string); ok {
		target, err := resolve(ref)
		if err != nil {
			return nil, err
		}
		return normalizeSchemaForCompare(target, resolve)
	}
	if anyOf, ok := s["anyOf"].([]any); ok {
		var nonNull []any
		nullable := false
		for _, b := range anyOf {
			bm := b.(map[string]any)
			if t, ok := bm["type"].(string); ok && t == "null" && len(bm) == 1 {
				nullable = true
				continue
			}
			nonNull = append(nonNull, b)
		}
		if nullable && len(nonNull) == 1 {
			out, err := normalizeSchemaForCompare(nonNull[0].(map[string]any), resolve)
			if err != nil {
				return nil, err
			}
			ts, _ := out["types"].([]string)
			ts = append(ts, "null")
			sort.Strings(ts)
			out["types"] = ts
			return out, nil
		}
		return nil, fmt.Errorf("unsupported anyOf")
	}
	out := map[string]any{}
	var types []string
	switch t := s["type"].(type) {
	case string:
		types = append(types, t)
	case []any:
		for _, x := range t {
			types = append(types, x.(string))
		}
	}
	if nb, _ := s["nullable"].(bool); nb {
		types = append(types, "null")
	}
	if len(types) > 0 {
		sort.Strings(types)
		out["types"] = types
	}
	for _, k := range []string{"pattern", "format", "minLength", "maxLength", "minimum", "maximum", "minItems", "maxItems", "uniqueItems", "minProperties", "maxProperties", "const", "default"} {
		if v, ok := s[k]; ok {
			switch n := v.(type) {
			case int:
				out[k] = float64(n)
			case int64:
				out[k] = float64(n)
			case uint64:
				out[k] = float64(n)
			default:
				out[k] = v
			}
		}
	}
	if req, ok := s["required"].([]any); ok {
		rs := make([]string, len(req))
		for i, x := range req {
			rs[i] = x.(string)
		}
		sort.Strings(rs)
		out["required"] = rs
	}
	if enum, ok := s["enum"].([]any); ok {
		var vals []string
		for _, e := range enum {
			if e != nil {
				vals = append(vals, fmt.Sprint(e))
			}
		}
		sort.Strings(vals)
		out["enum"] = vals
	}
	if props, ok := s["properties"].(map[string]any); ok {
		np := map[string]any{}
		for k, v := range props {
			c, err := normalizeSchemaForCompare(v.(map[string]any), resolve)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			np[k] = c
		}
		out["properties"] = np
	}
	if items, ok := s["items"].(map[string]any); ok {
		c, err := normalizeSchemaForCompare(items, resolve)
		if err != nil {
			return nil, fmt.Errorf("items: %w", err)
		}
		out["items"] = c
	}
	if ap, ok := s["additionalProperties"].(bool); ok {
		out["additionalProperties"] = ap
	}
	if allOf, ok := s["allOf"].([]any); ok {
		for _, b := range allOf {
			c, err := normalizeSchemaForCompare(b.(map[string]any), resolve)
			if err != nil {
				return nil, err
			}
			for k, v := range c {
				switch k {
				case "properties":
					dp, _ := out[k].(map[string]any)
					if dp == nil {
						dp = map[string]any{}
					}
					for pk, pv := range v.(map[string]any) {
						dp[pk] = pv
					}
					out[k] = dp
				case "required":
					set := map[string]bool{}
					if existing, ok := out[k].([]string); ok {
						for _, r := range existing {
							set[r] = true
						}
					}
					for _, r := range v.([]string) {
						set[r] = true
					}
					var all []string
					for r := range set {
						all = append(all, r)
					}
					sort.Strings(all)
					out[k] = all
				case "types":
					set := map[string]bool{}
					if existing, ok := out[k].([]string); ok {
						for _, t := range existing {
							set[t] = true
						}
					}
					for _, t := range v.([]string) {
						set[t] = true
					}
					var all []string
					for t := range set {
						all = append(all, t)
					}
					sort.Strings(all)
					out[k] = all
				default:
					if _, exists := out[k]; !exists {
						out[k] = v
					}
				}
			}
		}
	}
	return out, nil
}

func TestAPI01_Uint64AndSafeIntegerCaps(t *testing.T) {
	refTime := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	opts := policies.ValidationOptions{ReferenceTime: refTime}

	t.Run("fair_use threshold_bytes uint64 cap", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers = p.Tiers[:2]
		p.Tiers[1].ThresholdBytes = ^decstr.U64(0) // 18446744073709551615
		rawMax, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, rawMax, opts); err != nil {
			t.Fatalf("expected max uint64 threshold_bytes to pass, got: %v", err)
		}

		for _, overflow := range []string{"18446744073709551616", "99999999999999999999"} {
			var m map[string]any
			_ = json.Unmarshal(rawMax, &m)
			tiers := m["tiers"].([]any)
			tiers[1].(map[string]any)["threshold_bytes"] = overflow
			rawOverflow, _ := json.Marshal(m)
			if err := policies.ValidateJSONSchema(api.PolicyTypeFairUse, rawOverflow); err == nil {
				t.Fatalf("expected ValidateJSONSchema to reject threshold_bytes=%q", overflow)
			}
			if _, err := policies.ValidatePayload(api.PolicyTypeFairUse, 1, rawOverflow, opts); err == nil {
				t.Fatalf("expected ValidatePayload to reject threshold_bytes=%q", overflow)
			}
		}
	})

	t.Run("domain_rule_set revision positive uint64 cap", func(t *testing.T) {
		base := api.DomainRuleSetPayload{
			Rules: []api.DomainRule{
				{
					RuleId:            openapi_types.UUID(uuid.New()),
					ProfileId:         presence.NewNullableWithValue("standard"),
					Domain:            "example.com",
					IncludeSubdomains: true,
					Action:            api.DomainRuleActionBlock,
					Reason:            "admin_rule",
					Revision:          ^decstr.U64(0),
				},
			},
		}
		rawMax, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := policies.ValidatePayload(api.PolicyTypeDomainRuleSet, 1, rawMax, opts); err != nil {
			t.Fatalf("expected max uint64 revision to pass, got: %v", err)
		}

		// Revision "0" is rejected with ReasonDNSInvalidRevision.
		var mZero map[string]any
		_ = json.Unmarshal(rawMax, &mZero)
		mZero["rules"].([]any)[0].(map[string]any)["revision"] = "0"
		rawZero, _ := json.Marshal(mZero)
		if err := policies.ValidateJSONSchema(api.PolicyTypeDomainRuleSet, rawZero); !policies.HasReason(err, policies.ReasonDNSInvalidRevision) {
			t.Fatalf("expected ValidateJSONSchema to reject revision=\"0\" with ReasonDNSInvalidRevision, got: %v", err)
		}
		if _, err := policies.ValidatePayload(api.PolicyTypeDomainRuleSet, 1, rawZero, opts); !policies.HasReason(err, policies.ReasonDNSInvalidRevision) {
			t.Fatalf("expected ValidatePayload to reject revision=\"0\" with ReasonDNSInvalidRevision, got: %v", err)
		}

		// 20-digit strings exceeding uint64 max are rejected.
		for _, overflow := range []string{"18446744073709551616", "99999999999999999999"} {
			var m map[string]any
			_ = json.Unmarshal(rawMax, &m)
			m["rules"].([]any)[0].(map[string]any)["revision"] = overflow
			rawOverflow, _ := json.Marshal(m)
			if err := policies.ValidateJSONSchema(api.PolicyTypeDomainRuleSet, rawOverflow); err == nil {
				t.Fatalf("expected ValidateJSONSchema to reject revision=%q", overflow)
			}
			if _, err := policies.ValidatePayload(api.PolicyTypeDomainRuleSet, 1, rawOverflow, opts); err == nil {
				t.Fatalf("expected ValidatePayload to reject revision=%q", overflow)
			}
		}
	})

	t.Run("threat_feed max_size_bytes uint64 cap", func(t *testing.T) {
		rawExample, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "policies", "threat-feed.json"))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(rawExample, &m); err != nil {
			t.Fatal(err)
		}
		m["max_size_bytes"] = "18446744073709551615"
		rawMax, _ := json.Marshal(m)
		if _, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, rawMax, opts); err != nil {
			t.Fatalf("expected max uint64 max_size_bytes to pass, got: %v", err)
		}

		for _, overflow := range []string{"18446744073709551616", "99999999999999999999"} {
			m["max_size_bytes"] = overflow
			rawOverflow, _ := json.Marshal(m)
			if err := policies.ValidateJSONSchema(api.PolicyTypeThreatFeed, rawOverflow); err == nil {
				t.Fatalf("expected ValidateJSONSchema to reject max_size_bytes=%q", overflow)
			}
			if _, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, rawOverflow, opts); err == nil {
				t.Fatalf("expected ValidatePayload to reject max_size_bytes=%q", overflow)
			}
		}
	})

	t.Run("safe integer maximum on rates and amount_minor", func(t *testing.T) {
		p := validFairUsePayload()
		p.Tiers[0].DownBps = canonjson.MaxSafeInteger + 1
		if err := policies.ValidateFairUse(p, opts); !policies.HasReason(err, policies.ReasonSchemaInvalid) {
			t.Fatalf("expected ValidateFairUse to reject DownBps > MaxSafeInt, got: %v", err)
		}

		rawPlan, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "policies", "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		var plan api.PlanPayload
		if err := json.Unmarshal(rawPlan, &plan); err != nil {
			t.Fatal(err)
		}
		plan.Price.AmountMinor = canonjson.MaxSafeInteger + 1
		if err := policies.ValidatePlan(plan); !policies.HasReason(err, policies.ReasonPlanPriceInvalid) {
			t.Fatalf("expected ValidatePlan to reject AmountMinor > MaxSafeInt, got: %v", err)
		}

		plan.Price.AmountMinor = 30000
		plan.Price.Provider = "Invalid Provider"
		if err := policies.ValidatePlan(plan); !policies.HasReason(err, policies.ReasonPlanPriceInvalid) {
			t.Fatalf("expected ValidatePlan to reject invalid provider format, got: %v", err)
		}
	})
}
