package policies

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

var (
	codePattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	sha256Pattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ValidationOptions configures time-sensitive validation rules (such as
// override expiration checks).
type ValidationOptions struct {
	// ReferenceTime, when non-zero, requires all fair_use overrides to have
	// valid_until strictly after ReferenceTime.
	ReferenceTime time.Time
}

// ValidatedPayload holds the canonicalized, hashed, and semantically validated
// policy payload.
type ValidatedPayload struct {
	Type          api.PolicyType
	SchemaVersion int
	CanonicalJSON []byte
	PayloadHash   [32]byte
	PayloadSHA256 string

	Plan           *api.PlanPayload
	FairUse        *api.FairUsePayload
	DNSProfile     *api.DnsProfilePayload
	DomainRuleSet  *api.DomainRuleSetPayload
	ThreatFeed     *api.ThreatFeedPayload
	ReleaseSupport *api.ReleaseSupportPayload
}

// ValidatePayload validates a raw policy payload against:
//  1. RFC 8785 canonical JSON constraints (no floats, integers within ±(2^53-1)),
//  2. Versioned JSON Schema 2020-12 for policyType,
//  3. Strict struct decoding into pkg/api types,
//  4. Type-specific semantic validation and fixed isolation baseline guards.
func ValidatePayload(policyType api.PolicyType, schemaVersion int, rawPayload []byte, opts ValidationOptions) (*ValidatedPayload, error) {
	if schemaVersion != 1 {
		return nil, newValidationError("/schema_version", ReasonSchemaVersionInvalid, fmt.Sprintf("unsupported schema_version %d (want 1)", schemaVersion))
	}
	if len(bytes.TrimSpace(rawPayload)) == 0 {
		return nil, newValidationError("/payload", ReasonCanonicalJSONInvalid, "policy payload is required")
	}

	// Pre-scan for domain rule IDNA normalization / isolation baseline before
	// strict regex rejection so non-ASCII IDNA duplicates/conflicts and
	// baseline violations report their specific semantic reason.
	if policyType == api.PolicyTypeDomainRuleSet {
		if err := precheckRawDomainRules(rawPayload); err != nil {
			return nil, err
		}
	}

	if err := ValidateJSONSchema(policyType, rawPayload); err != nil {
		return nil, err
	}

	var rawMap map[string]any
	dec := json.NewDecoder(bytes.NewReader(rawPayload))
	dec.UseNumber()
	if err := dec.Decode(&rawMap); err != nil {
		return nil, newValidationError("/payload", ReasonCanonicalJSONInvalid, fmt.Sprintf("decode payload object: %v", err))
	}
	canonicalBytes, err := canonjson.Marshal(rawMap)
	if err != nil {
		return nil, newValidationError("/payload", ReasonCanonicalJSONInvalid, fmt.Sprintf("canonicalize payload JSON: %v", err))
	}
	hash, err := canonjson.Hash(rawPayload)
	if err != nil {
		return nil, newValidationError("/payload", ReasonCanonicalJSONInvalid, fmt.Sprintf("hash payload JSON: %v", err))
	}

	res := &ValidatedPayload{
		Type:          policyType,
		SchemaVersion: schemaVersion,
		CanonicalJSON: canonicalBytes,
		PayloadHash:   hash,
		PayloadSHA256: hex.EncodeToString(hash[:]),
	}

	switch policyType {
	case api.PolicyTypeFairUse:
		var p api.FairUsePayload
		if err := strictDecode(canonicalBytes, &p); err != nil {
			return nil, newValidationError("/payload", ReasonSchemaInvalid, err.Error())
		}
		if err := ValidateFairUse(p, opts); err != nil {
			return nil, err
		}
		res.FairUse = &p

	case api.PolicyTypePlan:
		var p api.PlanPayload
		if err := strictDecode(canonicalBytes, &p); err != nil {
			return nil, newValidationError("/payload", ReasonSchemaInvalid, err.Error())
		}
		if err := ValidatePlan(p); err != nil {
			return nil, err
		}
		res.Plan = &p

	case api.PolicyTypeDnsProfile:
		var p api.DnsProfilePayload
		if err := strictDecode(canonicalBytes, &p); err != nil {
			return nil, newValidationError("/payload", ReasonSchemaInvalid, err.Error())
		}
		if err := ValidateDNSProfile(p); err != nil {
			return nil, err
		}
		res.DNSProfile = &p

	case api.PolicyTypeDomainRuleSet:
		var p api.DomainRuleSetPayload
		if err := strictDecode(canonicalBytes, &p); err != nil {
			return nil, newValidationError("/payload", ReasonSchemaInvalid, err.Error())
		}
		if err := ValidateDomainRuleSet(p); err != nil {
			return nil, err
		}
		res.DomainRuleSet = &p

	case api.PolicyTypeThreatFeed:
		var p api.ThreatFeedPayload
		if err := strictDecode(canonicalBytes, &p); err != nil {
			return nil, newValidationError("/payload", ReasonSchemaInvalid, err.Error())
		}
		if err := ValidateThreatFeed(p); err != nil {
			return nil, err
		}
		res.ThreatFeed = &p

	case api.PolicyTypeReleaseSupport:
		var p api.ReleaseSupportPayload
		if err := strictDecode(canonicalBytes, &p); err != nil {
			return nil, newValidationError("/payload", ReasonSchemaInvalid, err.Error())
		}
		if err := ValidateReleaseSupport(p); err != nil {
			return nil, err
		}
		res.ReleaseSupport = &p

	default:
		return nil, newValidationError("/type", ReasonUnknownPolicyType, fmt.Sprintf("unsupported policy type %q", policyType))
	}

	return res, nil
}

func strictDecode(data []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	return nil
}

// ValidateFairUse enforces all semantic rules of §11.1 on a FairUsePayload.
func ValidateFairUse(p api.FairUsePayload, opts ValidationOptions) error {
	switch p.Mode {
	case api.FairUsePayloadModeShadow, api.FairUsePayloadModeEnforce:
	default:
		return newValidationError("/payload/mode", ReasonFairUseModeInvalid, fmt.Sprintf("fair_use mode must be shadow or enforce, got %q", p.Mode))
	}

	if p.EffectiveAt.IsZero() {
		return newValidationError("/payload/effective_at", ReasonFairUseEffectiveAtInvalid, "effective_at is required and must be a finite UTC timestamp")
	}

	switch p.EffectivePeriod {
	case api.FairUsePayloadEffectivePeriodNextFullPeriod, api.FairUsePayloadEffectivePeriodImmediate:
	default:
		return newValidationError("/payload/effective_period", ReasonFairUseEffectivePeriodInvalid, fmt.Sprintf("effective_period must be next_full_period or immediate, got %q", p.EffectivePeriod))
	}

	if p.AccountingBasis.PeriodType != api.FairUsePayloadAccountingBasisPeriodTypeUtcCalendarMonth {
		return newValidationError("/payload/accounting_basis/period_type", ReasonFairUseAccountingBasisInvalid, fmt.Sprintf("accounting_basis.period_type must be utc_calendar_month, got %q", p.AccountingBasis.PeriodType))
	}
	if p.AccountingBasis.MeteringVersion != api.MeteringVersionForwardAdmissionV1 {
		return newValidationError("/payload/accounting_basis/metering_version", ReasonFairUseAccountingBasisInvalid, fmt.Sprintf("accounting_basis.metering_version must be forward_admission_v1, got %q", p.AccountingBasis.MeteringVersion))
	}
	if p.AccountingBasis.Measure != api.FairUsePayloadAccountingBasisMeasureTotalBytes {
		return newValidationError("/payload/accounting_basis/measure", ReasonFairUseAccountingBasisInvalid, fmt.Sprintf("accounting_basis.measure must be total_bytes, got %q", p.AccountingBasis.Measure))
	}

	if p.MinimumRate.DownBps < 0 {
		return newValidationError("/payload/minimum_rate/down_bps", ReasonFairUseNegativeRate, "minimum_rate.down_bps must be non-negative")
	}
	if p.MinimumRate.UpBps < 0 {
		return newValidationError("/payload/minimum_rate/up_bps", ReasonFairUseNegativeRate, "minimum_rate.up_bps must be non-negative")
	}

	if len(p.Tiers) == 0 {
		return newValidationError("/payload/tiers", ReasonFairUseEmptyTiers, "fair_use policy must define at least one tier")
	}
	if len(p.Tiers) > 16 {
		return newValidationError("/payload/tiers", ReasonSchemaInvalid, "fair_use policy may define at most 16 tiers")
	}

	if p.Tiers[0].ThresholdBytes != 0 {
		return newValidationError("/payload/tiers/0/threshold_bytes", ReasonFairUseTierZeroThreshold, fmt.Sprintf("tier 0 threshold_bytes must be \"0\" for complete coverage from zero, got %q", p.Tiers[0].ThresholdBytes.String()))
	}

	for i, tier := range p.Tiers {
		prefix := fmt.Sprintf("/payload/tiers/%d", i)

		if tier.DownBps < 0 {
			return newValidationError(prefix+"/down_bps", ReasonFairUseNegativeRate, "tier down_bps must be non-negative")
		}
		if tier.UpBps < 0 {
			return newValidationError(prefix+"/up_bps", ReasonFairUseNegativeRate, "tier up_bps must be non-negative")
		}
		if tier.Priority < 0 || tier.Priority > 7 {
			return newValidationError(prefix+"/priority", ReasonFairUsePriorityBounds, fmt.Sprintf("tier priority must be in [0, 7], got %d", tier.Priority))
		}
		if err := validateLocalizedText(tier.DisplayName, prefix+"/display_name", ReasonFairUseDisplayNameEmpty); err != nil {
			return err
		}

		// Every tier must be at or above minimum_rate in each direction (§11.1).
		if tier.DownBps < p.MinimumRate.DownBps {
			return newValidationError(prefix+"/down_bps", ReasonFairUseBelowMinimumRate, fmt.Sprintf("tier %d down_bps (%d) is below minimum_rate.down_bps (%d)", i, tier.DownBps, p.MinimumRate.DownBps))
		}
		if tier.UpBps < p.MinimumRate.UpBps {
			return newValidationError(prefix+"/up_bps", ReasonFairUseBelowMinimumRate, fmt.Sprintf("tier %d up_bps (%d) is below minimum_rate.up_bps (%d)", i, tier.UpBps, p.MinimumRate.UpBps))
		}

		if i > 0 {
			prev := p.Tiers[i-1]
			// Strictly increasing thresholds (§11.1).
			if tier.ThresholdBytes <= prev.ThresholdBytes {
				return newValidationError(prefix+"/threshold_bytes", ReasonFairUseThresholdOrder, fmt.Sprintf("tier %d threshold_bytes (%s) must be strictly greater than tier %d (%s)", i, tier.ThresholdBytes.String(), i-1, prev.ThresholdBytes.String()))
			}
			// Non-increasing rates across tiers: higher usage tier can never be faster (§11.1).
			if tier.DownBps > prev.DownBps {
				return newValidationError(prefix+"/down_bps", ReasonFairUseRateOrder, fmt.Sprintf("tier %d down_bps (%d) must not exceed tier %d down_bps (%d)", i, tier.DownBps, i-1, prev.DownBps))
			}
			if tier.UpBps > prev.UpBps {
				return newValidationError(prefix+"/up_bps", ReasonFairUseRateOrder, fmt.Sprintf("tier %d up_bps (%d) must not exceed tier %d up_bps (%d)", i, tier.UpBps, i-1, prev.UpBps))
			}
			// Non-decreasing HTB priority number across tiers (§11.1).
			if tier.Priority < prev.Priority {
				return newValidationError(prefix+"/priority", ReasonFairUsePriorityOrder, fmt.Sprintf("tier %d priority (%d) must be >= tier %d priority (%d)", i, tier.Priority, i-1, prev.Priority))
			}
		}
	}

	seenMembers := make(map[uuid.UUID]int, len(p.Overrides))
	for i, ov := range p.Overrides {
		prefix := fmt.Sprintf("/payload/overrides/%d", i)
		memberID := uuid.UUID(ov.MemberId)
		if memberID == uuid.Nil {
			return newValidationError(prefix+"/member_id", ReasonFairUseInvalidOverride, "override member_id must not be nil UUID")
		}
		if prevIdx, exists := seenMembers[memberID]; exists {
			return newConflictError(prefix+"/member_id", ReasonFairUseDuplicateOverride, fmt.Sprintf("duplicate override for member_id %s (already at index %d)", memberID, prevIdx))
		}
		seenMembers[memberID] = i

		if ov.DownBps < 0 {
			return newValidationError(prefix+"/down_bps", ReasonFairUseNegativeRate, "override down_bps must be non-negative")
		}
		if ov.UpBps < 0 {
			return newValidationError(prefix+"/up_bps", ReasonFairUseNegativeRate, "override up_bps must be non-negative")
		}
		if ov.Priority < 0 || ov.Priority > 7 {
			return newValidationError(prefix+"/priority", ReasonFairUsePriorityBounds, fmt.Sprintf("override priority must be in [0, 7], got %d", ov.Priority))
		}
		if uuid.UUID(ov.ActorId) == uuid.Nil {
			return newValidationError(prefix+"/actor_id", ReasonFairUseInvalidOverride, "override actor_id must not be nil UUID")
		}
		if strings.TrimSpace(ov.Reason) == "" || len(ov.Reason) > 500 {
			return newValidationError(prefix+"/reason", ReasonFairUseInvalidOverride, "override reason must be 1..500 non-whitespace characters")
		}
		if ov.ValidUntil.IsZero() {
			return newValidationError(prefix+"/valid_until", ReasonFairUseExpiredOverride, "override valid_until is required")
		}
		if !opts.ReferenceTime.IsZero() && !ov.ValidUntil.After(opts.ReferenceTime) {
			return newValidationError(prefix+"/valid_until", ReasonFairUseExpiredOverride, fmt.Sprintf("override valid_until (%s) must be in the future relative to %s", ov.ValidUntil.UTC().Format(time.RFC3339), opts.ReferenceTime.UTC().Format(time.RFC3339)))
		}
	}

	return nil
}

// ValidatePlan enforces semantic rules on a PlanPayload (§14.2, §15.2).
func ValidatePlan(p api.PlanPayload) error {
	if err := validateLocalizedText(p.DisplayName, "/payload/display_name", ReasonPlanDisplayNameEmpty); err != nil {
		return err
	}
	if p.DeviceLimit < 1 || p.DeviceLimit > 100 {
		return newValidationError("/payload/device_limit", ReasonPlanDeviceLimitInvalid, fmt.Sprintf("device_limit must be in [1, 100], got %d", p.DeviceLimit))
	}
	if len(p.ExitGroupIds) == 0 {
		return newValidationError("/payload/exit_group_ids", ReasonPlanExitGroupsEmpty, "exit_group_ids must not be empty")
	}
	seenGroups := make(map[uuid.UUID]struct{}, len(p.ExitGroupIds))
	for i, gid := range p.ExitGroupIds {
		id := uuid.UUID(gid)
		if id == uuid.Nil {
			return newValidationError(fmt.Sprintf("/payload/exit_group_ids/%d", i), ReasonPlanUnknownExitGroup, "exit_group_id must not be nil UUID")
		}
		if _, exists := seenGroups[id]; exists {
			return newValidationError(fmt.Sprintf("/payload/exit_group_ids/%d", i), ReasonPlanExitGroupDuplicate, fmt.Sprintf("duplicate exit_group_id %s", id))
		}
		seenGroups[id] = struct{}{}
	}

	if p.Price.AmountMinor < 0 {
		return newValidationError("/payload/price/amount_minor", ReasonPlanPriceInvalid, "price.amount_minor must be non-negative")
	}
	if !currencyPattern.MatchString(p.Price.Currency) {
		return newValidationError("/payload/price/currency", ReasonPlanPriceInvalid, fmt.Sprintf("price.currency %q must be a 3-letter uppercase ISO 4217 code", p.Price.Currency))
	}
	switch p.Price.BillingInterval {
	case api.PlanPriceBillingIntervalMonth, api.PlanPriceBillingIntervalYear:
	default:
		return newValidationError("/payload/price/billing_interval", ReasonPlanPriceInvalid, fmt.Sprintf("price.billing_interval must be month or year, got %q", p.Price.BillingInterval))
	}
	if strings.TrimSpace(p.Price.Provider) == "" {
		return newValidationError("/payload/price/provider", ReasonPlanPriceInvalid, "price.provider is required")
	}
	if strings.TrimSpace(p.Price.ProviderPriceRef) == "" {
		return newValidationError("/payload/price/provider_price_ref", ReasonPlanPriceInvalid, "price.provider_price_ref is required")
	}

	if rate, err := p.RateLimit.Get(); err == nil {
		if rate.DownBps < 0 || rate.UpBps < 0 {
			return newValidationError("/payload/rate_limit", ReasonPlanRateLimitInvalid, "rate_limit down_bps and up_bps must be non-negative")
		}
	}

	if uuid.UUID(p.FairUseRevisionId) == uuid.Nil {
		return newValidationError("/payload/fair_use_revision_id", ReasonPlanMissingFairUseRef, "fair_use_revision_id is required")
	}
	if uuid.UUID(p.DnsProfileRevisionId) == uuid.Nil {
		return newValidationError("/payload/dns_profile_revision_id", ReasonPlanMissingDNSProfile, "dns_profile_revision_id is required")
	}

	return nil
}

// ValidateDNSProfile enforces semantic rules on a DnsProfilePayload (§12.2, §12.4, §14.2).
func ValidateDNSProfile(p api.DnsProfilePayload) error {
	if !codePattern.MatchString(p.ProfileCode) {
		return newValidationError("/payload/profile_code", ReasonDNSInvalidProfile, fmt.Sprintf("invalid profile_code %q", p.ProfileCode))
	}
	if err := validateLocalizedText(p.DisplayName, "/payload/display_name", ReasonDNSInvalidProfile); err != nil {
		return err
	}
	switch p.BlockResponse {
	case api.DnsProfilePayloadBlockResponseNxdomain, api.DnsProfilePayloadBlockResponseRefused, api.DnsProfilePayloadBlockResponseNullAddress:
	default:
		return newValidationError("/payload/block_response", ReasonDNSInvalidProfile, fmt.Sprintf("unsupported block_response %q", p.BlockResponse))
	}
	if p.BlockedTtlSeconds < 0 || p.BlockedTtlSeconds > 86400 {
		return newValidationError("/payload/blocked_ttl_seconds", ReasonDNSInvalidProfile, fmt.Sprintf("blocked_ttl_seconds must be in [0, 86400], got %d", p.BlockedTtlSeconds))
	}

	seenRS := make(map[uuid.UUID]struct{}, len(p.DomainRuleSetRevisionIds))
	for i, id := range p.DomainRuleSetRevisionIds {
		u := uuid.UUID(id)
		if u == uuid.Nil {
			return newValidationError(fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i), ReasonDNSUnknownRuleSetRevision, "domain_rule_set_revision_id must not be nil UUID")
		}
		if _, dup := seenRS[u]; dup {
			return newValidationError(fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i), ReasonDNSInvalidProfile, fmt.Sprintf("duplicate domain_rule_set_revision_id %s", u))
		}
		seenRS[u] = struct{}{}
	}

	seenTF := make(map[uuid.UUID]struct{}, len(p.ThreatFeedRevisionIds))
	for i, id := range p.ThreatFeedRevisionIds {
		u := uuid.UUID(id)
		if u == uuid.Nil {
			return newValidationError(fmt.Sprintf("/payload/threat_feed_revision_ids/%d", i), ReasonDNSUnknownThreatFeedRev, "threat_feed_revision_id must not be nil UUID")
		}
		if _, dup := seenTF[u]; dup {
			return newValidationError(fmt.Sprintf("/payload/threat_feed_revision_ids/%d", i), ReasonDNSInvalidProfile, fmt.Sprintf("duplicate threat_feed_revision_id %s", u))
		}
		seenTF[u] = struct{}{}
	}
	return nil
}

// precheckRawDomainRules inspects raw JSON domain rules before JSON Schema
// regex rejection so IDNA-equivalent duplicates/conflicts and isolation
// baseline violations return their exact semantic error.
func precheckRawDomainRules(rawPayload []byte) error {
	var raw struct {
		Rules []struct {
			RuleID            string  `json:"rule_id"`
			ProfileID         *string `json:"profile_id"`
			Domain            string  `json:"domain"`
			IncludeSubdomains bool    `json:"include_subdomains"`
			Action            string  `json:"action"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(rawPayload, &raw); err != nil {
		return nil
	}
	type domainKey struct {
		profile string
		domain  string
	}
	seenDomains := make(map[domainKey]int, len(raw.Rules))
	for i, r := range raw.Rules {
		if r.Domain == "" {
			continue
		}
		if err := checkDomainIsolationBaseline(r.Domain, fmt.Sprintf("/payload/rules/%d/domain", i)); err != nil {
			return err
		}
		norm, err := dnspolicy.NormalizeDomain(r.Domain)
		if err != nil {
			return newValidationError(fmt.Sprintf("/payload/rules/%d/domain", i), ReasonDNSDomainNotCanonical, fmt.Sprintf("invalid domain %q: %v", r.Domain, err))
		}
		if err := checkDomainIsolationBaseline(norm.Canonical, fmt.Sprintf("/payload/rules/%d/domain", i)); err != nil {
			return err
		}
		prof := ""
		if r.ProfileID != nil {
			prof = *r.ProfileID
		}
		for _, pKey := range []string{prof, "*"} {
			if pKey == "*" && prof == "" {
				continue
			}
			k := domainKey{profile: pKey, domain: norm.Canonical}
			if prevIdx, exists := seenDomains[k]; exists {
				return newConflictError(fmt.Sprintf("/payload/rules/%d/domain", i), ReasonDNSSameLayerConflict, fmt.Sprintf("conflicting or duplicate domain rule for canonical domain %q (conflicts with rule at index %d)", norm.Canonical, prevIdx))
			}
		}
		seenDomains[domainKey{profile: prof, domain: norm.Canonical}] = i
		if prof == "" {
			seenDomains[domainKey{profile: "*", domain: norm.Canonical}] = i
		}
	}
	return nil
}

// ValidateDomainRuleSet enforces semantic rules on a DomainRuleSetPayload,
// delegating IDNA2008 domain normalization to internal/dnspolicy and enforcing
// duplicate/conflict rejection and fixed isolation baseline guards (§12.4, §14.2).
func ValidateDomainRuleSet(p api.DomainRuleSetPayload) error {
	seenRuleIDs := make(map[uuid.UUID]int, len(p.Rules))
	type domainKey struct {
		profile string
		domain  string
	}
	seenDomains := make(map[domainKey]int, len(p.Rules))

	for i, r := range p.Rules {
		prefix := fmt.Sprintf("/payload/rules/%d", i)
		rid := uuid.UUID(r.RuleId)
		if rid == uuid.Nil {
			return newValidationError(prefix+"/rule_id", ReasonSchemaInvalid, "rule_id must not be nil UUID")
		}
		if prevIdx, exists := seenRuleIDs[rid]; exists {
			return newConflictError(prefix+"/rule_id", ReasonDNSDuplicateRuleID, fmt.Sprintf("duplicate rule_id %q (already at index %d)", rid, prevIdx))
		}
		seenRuleIDs[rid] = i

		if profile, err := r.ProfileId.Get(); err == nil && !codePattern.MatchString(profile) {
			return newValidationError(prefix+"/profile_id", ReasonDNSInvalidProfile, fmt.Sprintf("invalid profile_id %q", profile))
		}

		if err := checkDomainIsolationBaseline(r.Domain, prefix+"/domain"); err != nil {
			return err
		}

		norm, err := dnspolicy.NormalizeDomain(r.Domain)
		if err != nil {
			return newValidationError(prefix+"/domain", ReasonDNSDomainNotCanonical, fmt.Sprintf("domain %q failed IDNA2008 normalization: %v", r.Domain, err))
		}
		if err := checkDomainIsolationBaseline(norm.Canonical, prefix+"/domain"); err != nil {
			return err
		}

		prof, _ := r.ProfileId.Get()
		// Check for same-profile or service-wide conflict on the normalized canonical domain
		// BEFORE failing on non-canonical spelling so IDNA-equivalent duplicates/conflicts
		// are caught even when one side was given in Unicode.
		if prevIdx, exists := seenDomains[domainKey{profile: prof, domain: norm.Canonical}]; exists {
			return newConflictError(prefix+"/domain", ReasonDNSSameLayerConflict, fmt.Sprintf("duplicate or conflicting domain rule for (%q, %q) at index %d and %d", prof, norm.Canonical, prevIdx, i))
		}
		if prof != "" {
			if prevIdx, exists := seenDomains[domainKey{profile: "", domain: norm.Canonical}]; exists {
				return newConflictError(prefix+"/domain", ReasonDNSSameLayerConflict, fmt.Sprintf("profile-specific rule for %q conflicts with service-wide rule at index %d", norm.Canonical, prevIdx))
			}
		} else {
			for k, prevIdx := range seenDomains {
				if k.domain == norm.Canonical {
					return newConflictError(prefix+"/domain", ReasonDNSSameLayerConflict, fmt.Sprintf("service-wide rule for %q conflicts with profile-specific rule at index %d", norm.Canonical, prevIdx))
				}
			}
		}
		seenDomains[domainKey{profile: prof, domain: norm.Canonical}] = i

		if r.Domain != norm.Canonical {
			return newValidationError(prefix+"/domain", ReasonDNSDomainNotCanonical, fmt.Sprintf("domain %q must be in canonical ASCII form %q", r.Domain, norm.Canonical))
		}

		switch r.Action {
		case api.DomainRuleActionBlock, api.DomainRuleActionAllow:
		default:
			return newValidationError(prefix+"/action", ReasonSchemaInvalid, fmt.Sprintf("invalid domain rule action %q", r.Action))
		}

		if strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 500 {
			return newValidationError(prefix+"/reason", ReasonSchemaInvalid, "domain rule reason must be 1..500 non-whitespace characters")
		}
		if r.Revision == 0 {
			return newValidationError(prefix+"/revision", ReasonDNSInvalidRevision, "domain rule revision must be >= 1")
		}
	}

	return nil
}

// checkDomainIsolationBaseline rejects any domain rule that attempts to target
// or loosen the fixed IP/DNS isolation baseline (private prefixes, loopback,
// link-local, overlay, local zones, reverse zones, or metadata endpoints).
func checkDomainIsolationBaseline(rawDomain string, fieldPath string) error {
	trimmed := strings.TrimSpace(strings.ToLower(strings.TrimSuffix(rawDomain, ".")))
	if trimmed == "" {
		return nil
	}
	if _, err := netip.ParseAddr(trimmed); err == nil {
		return newValidationError(fieldPath, ReasonIsolationBaselineViolation, fmt.Sprintf("domain rule %q cannot target an IP literal (fixed isolation baseline)", rawDomain))
	}
	if _, err := netip.ParsePrefix(trimmed); err == nil {
		return newValidationError(fieldPath, ReasonIsolationBaselineViolation, fmt.Sprintf("domain rule %q cannot target an IP prefix (fixed isolation baseline)", rawDomain))
	}

	reservedSuffixes := []string{
		"localhost",
		"local",
		"internal",
		"home.arpa",
		"in-addr.arpa",
		"ip6.arpa",
		"corp",
		"lan",
		"intranet",
	}
	for _, suf := range reservedSuffixes {
		if trimmed == suf || strings.HasSuffix(trimmed, "."+suf) {
			return newValidationError(fieldPath, ReasonIsolationBaselineViolation, fmt.Sprintf("domain rule %q targets reserved/private zone %q and cannot loosen the fixed isolation baseline", rawDomain, suf))
		}
	}
	return nil
}

// ValidateThreatFeed enforces semantic rules on a ThreatFeedPayload, delegating
// HTTPS/SSRF source URL validation to internal/dnspolicy (§12.3, §14.2).
func ValidateThreatFeed(p api.ThreatFeedPayload) error {
	if strings.TrimSpace(p.Provider) == "" {
		return newValidationError("/payload/provider", ReasonSchemaInvalid, "provider is required")
	}
	if !codePattern.MatchString(p.FeedCode) {
		return newValidationError("/payload/feed_code", ReasonSchemaInvalid, fmt.Sprintf("invalid feed_code %q", p.FeedCode))
	}
	if strings.TrimSpace(p.LicenseName) == "" {
		return newValidationError("/payload/license_name", ReasonSchemaInvalid, "license_name is required")
	}
	if err := dnspolicy.ValidateSourceURL(p.LicenseUrl); err != nil {
		return newValidationError("/payload/license_url", ReasonThreatFeedInvalidURL, fmt.Sprintf("invalid license_url: %v", err))
	}
	if err := dnspolicy.ValidateSourceURL(p.SourceUrl); err != nil {
		return newValidationError("/payload/source_url", ReasonThreatFeedInvalidURL, fmt.Sprintf("invalid source_url: %v", err))
	}
	if !isConfirmedCommercialBasis(p.CommercialUseBasis) {
		return newValidationError("/payload/commercial_use_basis", ReasonThreatFeedUnconfirmedBasis, "commercial_use_basis must confirm commercial SaaS/VPN redistribution rights")
	}

	switch p.Format {
	case api.ThreatFeedPayloadFormatDomainList, api.ThreatFeedPayloadFormatHosts, api.ThreatFeedPayloadFormatAdblock:
	default:
		return newValidationError("/payload/format", ReasonSchemaInvalid, fmt.Sprintf("unsupported threat feed format %q", p.Format))
	}

	if len(p.Categories) == 0 {
		return newValidationError("/payload/categories", ReasonSchemaInvalid, "categories must not be empty")
	}
	seenCats := make(map[api.ThreatFeedPayloadCategories]struct{}, len(p.Categories))
	for i, cat := range p.Categories {
		switch cat {
		case api.ThreatFeedPayloadCategoriesMalware, api.ThreatFeedPayloadCategoriesPhishing, api.ThreatFeedPayloadCategoriesAds, api.ThreatFeedPayloadCategoriesFamily:
		default:
			return newValidationError(fmt.Sprintf("/payload/categories/%d", i), ReasonSchemaInvalid, fmt.Sprintf("unsupported threat feed category %q", cat))
		}
		if _, dup := seenCats[cat]; dup {
			return newValidationError(fmt.Sprintf("/payload/categories/%d", i), ReasonSchemaInvalid, fmt.Sprintf("duplicate category %q", cat))
		}
		seenCats[cat] = struct{}{}
	}

	if !sha256Pattern.MatchString(p.ContentSha256) {
		return newValidationError("/payload/content_sha256", ReasonThreatFeedInvalidDigest, "content_sha256 must be 64 lowercase hex characters")
	}
	if p.MaxSizeBytes == 0 {
		return newValidationError("/payload/max_size_bytes", ReasonThreatFeedInvalidSize, "max_size_bytes must be > 0")
	}
	if p.ValidFrom.IsZero() || p.ValidUntil.IsZero() || !p.ValidUntil.After(p.ValidFrom) {
		return newValidationError("/payload/valid_until", ReasonThreatFeedInvalidValidity, "valid_until must be strictly after valid_from")
	}
	if p.UpdateIntervalSeconds < 300 || p.UpdateIntervalSeconds > 604800 {
		return newValidationError("/payload/update_interval_seconds", ReasonSchemaInvalid, fmt.Sprintf("update_interval_seconds must be in [300, 604800], got %d", p.UpdateIntervalSeconds))
	}

	return nil
}

func isConfirmedCommercialBasis(basis string) bool {
	trimmed := strings.TrimSpace(basis)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	disallowedExact := map[string]bool{
		"unknown":             true,
		"unconfirmed":         true,
		"pending":             true,
		"tbd":                 true,
		"todo":                true,
		"none":                true,
		"n/a":                 true,
		"non-commercial":      true,
		"non_commercial":      true,
		"noncommercial":       true,
		"personal":            true,
		"personal use only":   true,
		"non-commercial only": true,
	}
	if disallowedExact[lower] {
		return false
	}
	if strings.Contains(lower, "non-commercial only") || strings.Contains(lower, "personal use only") {
		return false
	}
	return true
}

// ValidateReleaseSupport enforces explicit set-membership rules on a
// ReleaseSupportPayload (§13.2, §14.2, AT27):
//   - minimum_serving_version.serving_release_ids is non-empty and unique
//   - approved_release_ids is unique and a subset of serving_release_ids
//   - revoked_release_ids is unique and disjoint from both approved_release_ids
//     and serving_release_ids
//   - compatibility is never inferred from lexical version-string ordering.
func ValidateReleaseSupport(p api.ReleaseSupportPayload) error {
	servingList := p.MinimumServingVersion.ServingReleaseIds
	if len(servingList) == 0 {
		return newValidationError("/payload/minimum_serving_version/serving_release_ids", ReasonReleaseServingEmpty, "minimum_serving_version.serving_release_ids must not be empty")
	}

	servingSet := make(map[uuid.UUID]struct{}, len(servingList))
	for i, id := range servingList {
		u := uuid.UUID(id)
		if u == uuid.Nil {
			return newValidationError(fmt.Sprintf("/payload/minimum_serving_version/serving_release_ids/%d", i), ReasonReleaseInvalidID, "serving release_id must not be nil UUID")
		}
		if _, dup := servingSet[u]; dup {
			return newValidationError(fmt.Sprintf("/payload/minimum_serving_version/serving_release_ids/%d", i), ReasonReleaseDuplicateID, fmt.Sprintf("duplicate serving release_id %s", u))
		}
		servingSet[u] = struct{}{}
	}

	approvedSet := make(map[uuid.UUID]struct{}, len(p.ApprovedReleaseIds))
	for i, id := range p.ApprovedReleaseIds {
		u := uuid.UUID(id)
		if u == uuid.Nil {
			return newValidationError(fmt.Sprintf("/payload/approved_release_ids/%d", i), ReasonReleaseInvalidID, "approved release_id must not be nil UUID")
		}
		if _, dup := approvedSet[u]; dup {
			return newValidationError(fmt.Sprintf("/payload/approved_release_ids/%d", i), ReasonReleaseDuplicateID, fmt.Sprintf("duplicate approved release_id %s", u))
		}
		if _, ok := servingSet[u]; !ok {
			return newValidationError(fmt.Sprintf("/payload/approved_release_ids/%d", i), ReasonReleaseApprovedNotInServing, fmt.Sprintf("approved release_id %s must be a member of minimum_serving_version.serving_release_ids", u))
		}
		approvedSet[u] = struct{}{}
	}

	revokedSet := make(map[uuid.UUID]struct{}, len(p.RevokedReleaseIds))
	for i, id := range p.RevokedReleaseIds {
		u := uuid.UUID(id)
		if u == uuid.Nil {
			return newValidationError(fmt.Sprintf("/payload/revoked_release_ids/%d", i), ReasonReleaseInvalidID, "revoked release_id must not be nil UUID")
		}
		if _, dup := revokedSet[u]; dup {
			return newValidationError(fmt.Sprintf("/payload/revoked_release_ids/%d", i), ReasonReleaseDuplicateID, fmt.Sprintf("duplicate revoked release_id %s", u))
		}
		if _, inApproved := approvedSet[u]; inApproved {
			return newConflictError(fmt.Sprintf("/payload/revoked_release_ids/%d", i), ReasonReleaseRevokedOverlap, fmt.Sprintf("revoked release_id %s cannot also be in approved_release_ids", u))
		}
		if _, inServing := servingSet[u]; inServing {
			return newConflictError(fmt.Sprintf("/payload/revoked_release_ids/%d", i), ReasonReleaseRevokedOverlap, fmt.Sprintf("revoked release_id %s cannot also be in minimum_serving_version.serving_release_ids", u))
		}
		revokedSet[u] = struct{}{}
	}

	return nil
}

// ReleaseSupportEvaluator evaluates release eligibility strictly by set
// membership in an active ReleaseSupportPayload, never by lexical version
// ordering (§13.2, AT27).
type ReleaseSupportEvaluator struct {
	approved map[uuid.UUID]struct{}
	serving  map[uuid.UUID]struct{}
	revoked  map[uuid.UUID]struct{}
}

// NewReleaseSupportEvaluator validates p and builds a lookup evaluator.
func NewReleaseSupportEvaluator(p api.ReleaseSupportPayload) (*ReleaseSupportEvaluator, error) {
	if err := ValidateReleaseSupport(p); err != nil {
		return nil, err
	}
	ev := &ReleaseSupportEvaluator{
		approved: make(map[uuid.UUID]struct{}, len(p.ApprovedReleaseIds)),
		serving:  make(map[uuid.UUID]struct{}, len(p.MinimumServingVersion.ServingReleaseIds)),
		revoked:  make(map[uuid.UUID]struct{}, len(p.RevokedReleaseIds)),
	}
	for _, id := range p.ApprovedReleaseIds {
		ev.approved[uuid.UUID(id)] = struct{}{}
	}
	for _, id := range p.MinimumServingVersion.ServingReleaseIds {
		ev.serving[uuid.UUID(id)] = struct{}{}
	}
	for _, id := range p.RevokedReleaseIds {
		ev.revoked[uuid.UUID(id)] = struct{}{}
	}
	return ev, nil
}

// CheckServingAllowed verifies whether releaseID is permitted to hold a serving
// lease and carry member traffic (§13.2, AT27).
func (e *ReleaseSupportEvaluator) CheckServingAllowed(releaseID uuid.UUID) error {
	if _, isRevoked := e.revoked[releaseID]; isRevoked {
		return apierr.Newf(apierr.ReleaseNotApproved, "release %s is revoked and barred from serving", releaseID)
	}
	if _, isServing := e.serving[releaseID]; !isServing {
		return apierr.Newf(apierr.MinimumVersionViolation, "release %s is not in minimum_serving_version.serving_release_ids", releaseID)
	}
	return nil
}

// CheckUpdateTargetAllowed verifies whether releaseID is permitted as an
// update batch target (§13.2, AT27).
func (e *ReleaseSupportEvaluator) CheckUpdateTargetAllowed(releaseID uuid.UUID) error {
	if _, isRevoked := e.revoked[releaseID]; isRevoked {
		return apierr.Newf(apierr.ReleaseNotApproved, "release %s is revoked and cannot be targeted for updates", releaseID)
	}
	if _, isApproved := e.approved[releaseID]; !isApproved {
		return apierr.Newf(apierr.ReleaseNotApproved, "release %s is not in approved_release_ids", releaseID)
	}
	if _, isServing := e.serving[releaseID]; !isServing {
		return apierr.Newf(apierr.MinimumVersionViolation, "release %s is not in minimum_serving_version.serving_release_ids", releaseID)
	}
	return nil
}

// SupportState returns the ReleaseSupportState for display in GET /admin/releases.
func (e *ReleaseSupportEvaluator) SupportState(releaseID uuid.UUID) api.ReleaseSupportState {
	if _, isRevoked := e.revoked[releaseID]; isRevoked {
		return api.ReleaseSupportStateRevoked
	}
	if _, isApproved := e.approved[releaseID]; isApproved {
		return api.ReleaseSupportStateApproved
	}
	return api.ReleaseSupportStateNotApproved
}

func validateLocalizedText(lt api.LocalizedText, fieldPrefix string, reason RuleReason) error {
	if strings.TrimSpace(lt.ZhTW) == "" || len(lt.ZhTW) > 200 {
		return newValidationError(fieldPrefix+"/zh-TW", reason, "display_name.zh-TW must be 1..200 non-whitespace characters")
	}
	if lt.En != nil && (strings.TrimSpace(*lt.En) == "" || len(*lt.En) > 200) {
		return newValidationError(fieldPrefix+"/en", reason, "display_name.en must be 1..200 non-whitespace characters when present")
	}
	return nil
}

// Ensure decstr is referenced for uint64 decimal bounds.
var _ = decstr.U64(0)
