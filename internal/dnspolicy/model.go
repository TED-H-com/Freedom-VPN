package dnspolicy

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RuleAction defines the filtering action of a DNS policy rule (§12.4).
type RuleAction string

const (
	ActionBlock RuleAction = "block"
	ActionAllow RuleAction = "allow"
)

// PolicyLayer identifies the precedence layer that produced a DNS policy
// decision (§12.4).
//
// Precedence order (highest first):
//  1. LayerAdminBlock     (priority 1: admin forced block)
//  2. LayerAdminAllow     (priority 2: admin explicit allow)
//  3. LayerSourceList     (priority 3: enabled security and category lists)
//  4. LayerDefaultResolve (priority 4: normal resolution via configured upstream)
//
// Note (§8.1, §12.4): Fixed network isolation and member entitlement sit above
// all DNS layers; no DNS allow rule ever loosens IP-layer isolation.
type PolicyLayer string

const (
	LayerAdminBlock     PolicyLayer = "admin_block"
	LayerAdminAllow     PolicyLayer = "admin_allow"
	LayerSourceList     PolicyLayer = "source_list"
	LayerDefaultResolve PolicyLayer = "default_resolve"
	LayerInvalidInput   PolicyLayer = "invalid_input"
)

// Rule represents an admin domain policy rule (§12.4).
type Rule struct {
	RuleID            string      `json:"rule_id"`
	ProfileID         string      `json:"profile_id"`
	Domain            string      `json:"domain"`
	DisplayDomain     string      `json:"display_domain,omitempty"`
	IncludeSubdomains bool        `json:"include_subdomains"`
	Action            RuleAction  `json:"action"`
	Layer             PolicyLayer `json:"layer,omitempty"`
	Reason            string      `json:"reason"`
	Revision          string      `json:"revision"`
}

// EffectiveLayer returns the precedence layer for r. By default, admin block
// rules belong to LayerAdminBlock (priority 1) and admin allow rules belong to
// LayerAdminAllow (priority 2) (§12.4). If r.Layer is explicitly set, it is
// returned directly.
func (r Rule) EffectiveLayer() PolicyLayer {
	if r.Layer != "" {
		return r.Layer
	}
	switch r.Action {
	case ActionBlock:
		return LayerAdminBlock
	case ActionAllow:
		return LayerAdminAllow
	default:
		return ""
	}
}

// DNSProfilePayload models the §14.2 dns_profile typed policy payload.
type DNSProfilePayload struct {
	SchemaVersion      int      `json:"schema_version"`
	ProfileID          string   `json:"profile_id"`
	Name               string   `json:"name"`
	Revision           string   `json:"revision"`
	DomainRuleSetRefs  []string `json:"domain_rule_set_refs"`
	ThreatFeedRefs     []string `json:"threat_feed_refs"`
	BlockMode          string   `json:"block_mode"`
	BlockedTTLSeconds  uint32   `json:"blocked_ttl_seconds"`
	IsolatePrivateNets bool     `json:"isolate_private_nets"`
}

// DomainRuleSetPayload models the §14.2 domain_rule_set typed policy payload.
type DomainRuleSetPayload struct {
	SchemaVersion int    `json:"schema_version"`
	RuleSetID     string `json:"rule_set_id"`
	ProfileID     string `json:"profile_id"`
	Revision      string `json:"revision"`
	Rules         []Rule `json:"rules"`
}

// ThreatFeedPayload models the §14.2 threat_feed typed policy payload.
type ThreatFeedPayload struct {
	SchemaVersion     int           `json:"schema_version"`
	SourceID          string        `json:"source_id"`
	Provider          string        `json:"provider"`
	Version           string        `json:"version"`
	License           string        `json:"license"`
	LicenseURL        string        `json:"license_url"`
	LicenseStatus     LicenseStatus `json:"license_status"`
	PermittedUse      string        `json:"permitted_use"`
	DistributionNotes string        `json:"distribution_notes"`
	AttributionNotes  string        `json:"attribution_notes"`
	URL               string        `json:"url"`
	Format            SourceFormat  `json:"format"`
	Category          SourceCategory `json:"category"`
	ContentSHA256     string        `json:"content_sha256"`
	ValidFrom         time.Time     `json:"valid_from"`
	ValidUntil        time.Time     `json:"valid_until"`
	Entries           []SourceEntry `json:"entries"`
}

// Profile is the in-memory representation of a DNS policy profile, its admin
// rules, and its enabled security/category list sources (§12.4, §12.5).
type Profile struct {
	ProfileID         string                 `json:"profile_id"`
	Name              string                 `json:"name,omitempty"`
	Revision          string                 `json:"revision"`
	Rules             []Rule                 `json:"rules"`
	EnabledSources    []string               `json:"enabled_sources,omitempty"`
	Sources           []ListSource           `json:"sources,omitempty"`
	BlockMode         string                 `json:"block_mode,omitempty"`
	BlockedTTLSeconds uint32                 `json:"blocked_ttl_seconds,omitempty"`
	SmokeTests        []SmokeTestExpectation `json:"smoke_tests,omitempty"`
}

// Decision is the result of evaluating a domain name against a Profile (§12.4).
type Decision struct {
	Domain          string      `json:"domain"`
	OriginalInput   string      `json:"original_input"`
	Action          RuleAction  `json:"action"`
	Layer           PolicyLayer `json:"layer"`
	MatchedRuleID   string      `json:"matched_rule_id,omitempty"`
	MatchedSourceID string      `json:"matched_source_id,omitempty"`
	MatchedDomain   string      `json:"matched_domain,omitempty"`
	Reason          string      `json:"reason"`
	Err             error       `json:"-"`
}

// DraftValidationLimits configures resource bounds checked during draft
// validation (§12.4, §12.5).
type DraftValidationLimits struct {
	MaxAdminRulesPerProfile int
	MaxTotalRulesPerBundle  int
	MaxSourcesPerProfile    int
}

const (
	DefaultMaxAdminRulesPerProfile = 10_000
	DefaultMaxTotalRulesPerBundle  = 500_000
	DefaultMaxSourcesPerProfile    = 32
)

// DefaultDraftLimits returns the recommended default draft validation limits.
func DefaultDraftLimits() DraftValidationLimits {
	return DraftValidationLimits{
		MaxAdminRulesPerProfile: DefaultMaxAdminRulesPerProfile,
		MaxTotalRulesPerBundle:  DefaultMaxTotalRulesPerBundle,
		MaxSourcesPerProfile:    DefaultMaxSourcesPerProfile,
	}
}

// MatchesDomain reports whether queryCanonical (an already-normalized canonical
// ASCII domain) matches ruleCanonical on a full DNS label boundary (§12.4, DNS02).
//
// When includeSubdomains is false, only the exact domain matches.
// When includeSubdomains is true, the apex and any subdomain ending with
// "." + ruleCanonical match. "example.com" never matches "notexample.com".
func MatchesDomain(ruleCanonical string, includeSubdomains bool, queryCanonical string) bool {
	if queryCanonical == "" || ruleCanonical == "" {
		return false
	}
	if queryCanonical == ruleCanonical {
		return true
	}
	if !includeSubdomains {
		return false
	}
	if len(queryCanonical) <= len(ruleCanonical)+1 {
		return false
	}
	return queryCanonical[len(queryCanonical)-len(ruleCanonical)-1] == '.' &&
		strings.HasSuffix(queryCanonical, ruleCanonical)
}

// ValidateDraft validates a draft Profile, its admin rules, and its referenced
// list sources (§12.4, §12.5):
//   - Validates profile_id and decimal revision.
//   - Normalizes and validates every rule's domain.
//   - Checks that every rule's profile_id matches profile.ProfileID.
//   - Rejects duplicate rule_ids and same-layer conflicts on the same canonical
//     domain name (never "last write wins").
//   - Verifies that every enabled source exists, has an approved commercial
//     license, valid HTTPS URL, and passed smoke tests.
//   - Enforces admin rule-count, source-count, and total rule-count limits.
func ValidateDraft(profile Profile, registry *SourceRegistry, limits *DraftValidationLimits) error {
	lim := DefaultDraftLimits()
	if limits != nil {
		if limits.MaxAdminRulesPerProfile > 0 {
			lim.MaxAdminRulesPerProfile = limits.MaxAdminRulesPerProfile
		}
		if limits.MaxTotalRulesPerBundle > 0 {
			lim.MaxTotalRulesPerBundle = limits.MaxTotalRulesPerBundle
		}
		if limits.MaxSourcesPerProfile > 0 {
			lim.MaxSourcesPerProfile = limits.MaxSourcesPerProfile
		}
	}

	if strings.TrimSpace(profile.ProfileID) == "" {
		return &ValidationError{
			Code:    ErrCodeProfileIDEmpty,
			Field:   "profile_id",
			Message: "profile_id must not be empty",
		}
	}
	if err := ValidateDecimalRevision(profile.Revision); err != nil {
		return err
	}

	if len(profile.Rules) > lim.MaxAdminRulesPerProfile {
		return &ValidationError{
			Code:    ErrCodeRuleLimitExceeded,
			Field:   "rules",
			Message: fmt.Sprintf("profile %q has %d admin rules, exceeding limit of %d", profile.ProfileID, len(profile.Rules), lim.MaxAdminRulesPerProfile),
		}
	}

	seenRuleIDs := make(map[string]struct{}, len(profile.Rules))
	type layerDomainKey struct {
		layer  PolicyLayer
		domain string
	}
	seenInLayer := make(map[layerDomainKey]Rule, len(profile.Rules))

	for i, rule := range profile.Rules {
		if strings.TrimSpace(rule.RuleID) == "" {
			return &ValidationError{
				Code:    ErrCodeRuleIDEmpty,
				Field:   fmt.Sprintf("rules[%d].rule_id", i),
				Message: "rule_id must not be empty",
			}
		}
		if _, dup := seenRuleIDs[rule.RuleID]; dup {
			return &ValidationError{
				Code:    ErrCodeRuleIDDuplicate,
				Field:   "rule_id",
				Input:   rule.RuleID,
				Message: fmt.Sprintf("duplicate rule_id %q in profile %q", rule.RuleID, profile.ProfileID),
			}
		}
		seenRuleIDs[rule.RuleID] = struct{}{}

		if rule.ProfileID != profile.ProfileID {
			return &ValidationError{
				Code:    ErrCodeProfileRefMismatch,
				Field:   "profile_id",
				Input:   rule.ProfileID,
				Message: fmt.Sprintf("rule %q references profile_id %q, expected %q", rule.RuleID, rule.ProfileID, profile.ProfileID),
			}
		}

		if rule.Action != ActionBlock && rule.Action != ActionAllow {
			return &ValidationError{
				Code:    ErrCodeRuleInvalidAction,
				Field:   "action",
				Input:   string(rule.Action),
				Message: fmt.Sprintf("rule %q has invalid action %q (must be 'block' or 'allow')", rule.RuleID, rule.Action),
			}
		}

		layer := rule.EffectiveLayer()
		if layer != LayerAdminBlock && layer != LayerAdminAllow {
			return &ValidationError{
				Code:    ErrCodeRuleInvalidLayer,
				Field:   "layer",
				Input:   string(layer),
				Message: fmt.Sprintf("rule %q has invalid admin layer %q", rule.RuleID, layer),
			}
		}

		if err := ValidateDecimalRevision(rule.Revision); err != nil {
			return &ValidationError{
				Code:    ErrCodeInvalidRevision,
				Field:   fmt.Sprintf("rules[%d].revision", i),
				Input:   rule.Revision,
				Message: fmt.Sprintf("rule %q has invalid revision %q: %v", rule.RuleID, rule.Revision, err),
				Cause:   err,
			}
		}

		norm, err := NormalizeDomain(rule.Domain)
		if err != nil {
			return err
		}

		key := layerDomainKey{
			layer:  layer,
			domain: norm.Canonical,
		}
		if prev, exists := seenInLayer[key]; exists {
			return &ValidationError{
				Code:  ErrCodeSameLayerConflict,
				Field: "domain",
				Input: rule.Domain,
				Message: fmt.Sprintf(
					"same-layer conflict in layer %q on canonical domain %q between rule %q (action=%s, include_subdomains=%t) and rule %q (action=%s, include_subdomains=%t)",
					layer, norm.Canonical,
					prev.RuleID, prev.Action, prev.IncludeSubdomains,
					rule.RuleID, rule.Action, rule.IncludeSubdomains,
				),
			}
		}
		seenInLayer[key] = rule
	}

	resolvedSources, err := resolveProfileSources(profile, registry)
	if err != nil {
		return err
	}
	if len(resolvedSources) > lim.MaxSourcesPerProfile {
		return &ValidationError{
			Code:    ErrCodeRuleLimitExceeded,
			Field:   "enabled_sources",
			Message: fmt.Sprintf("profile %q enables %d sources, exceeding limit of %d", profile.ProfileID, len(resolvedSources), lim.MaxSourcesPerProfile),
		}
	}

	totalRules := len(profile.Rules)
	for _, src := range resolvedSources {
		if err := src.ValidateMetadata(); err != nil {
			return err
		}
		for _, entry := range src.Entries {
			if _, err := NormalizeDomain(entry.Domain); err != nil {
				return fmt.Errorf("source %q contains invalid domain %q: %w", src.SourceID, entry.Domain, err)
			}
		}
		totalRules += len(src.Entries)
	}

	if totalRules > lim.MaxTotalRulesPerBundle {
		return &ValidationError{
			Code:    ErrCodeRuleLimitExceeded,
			Field:   "total_rules",
			Message: fmt.Sprintf("profile %q total rule count %d exceeds bundle limit of %d", profile.ProfileID, totalRules, lim.MaxTotalRulesPerBundle),
		}
	}

	return nil
}

func resolveProfileSources(profile Profile, registry *SourceRegistry) ([]ListSource, error) {
	inlineMap := make(map[string]ListSource, len(profile.Sources))
	for _, s := range profile.Sources {
		inlineMap[s.SourceID] = s
	}

	seenSourceID := make(map[string]struct{})
	var out []ListSource

	for _, id := range profile.EnabledSources {
		if _, dup := seenSourceID[id]; dup {
			continue
		}
		seenSourceID[id] = struct{}{}
		if s, ok := inlineMap[id]; ok {
			out = append(out, s)
			continue
		}
		if registry != nil {
			if s, ok := registry.Get(id); ok {
				out = append(out, s)
				continue
			}
		}
		return nil, &ValidationError{
			Code:    ErrCodeSourceNotFound,
			Field:   "enabled_sources",
			Input:   id,
			Message: fmt.Sprintf("profile %q references unknown source_id %q", profile.ProfileID, id),
		}
	}

	// Also include any inline profile.Sources not already listed in EnabledSources
	// when EnabledSources is omitted.
	if len(profile.EnabledSources) == 0 && len(profile.Sources) > 0 {
		for _, s := range profile.Sources {
			if _, dup := seenSourceID[s.SourceID]; dup {
				continue
			}
			seenSourceID[s.SourceID] = struct{}{}
			out = append(out, s)
		}
	}

	return out, nil
}

// Evaluate evaluates a domain query against profile and returns the reference
// Decision (layer, matched rule or source, and action) per §12.4.
//
// Precedence order (highest first):
//  1. Admin forced block (LayerAdminBlock)
//  2. Admin explicit allow (LayerAdminAllow)
//  3. Enabled security and category lists (LayerSourceList)
//  4. Normal resolution (LayerDefaultResolve)
//
// If name is not a valid domain according to NormalizeDomain, Evaluate fails
// closed with ActionBlock and LayerInvalidInput.
func Evaluate(profile Profile, name string) Decision {
	norm, err := NormalizeDomain(name)
	if err != nil {
		return Decision{
			Domain:        "",
			OriginalInput: name,
			Action:        ActionBlock,
			Layer:         LayerInvalidInput,
			Reason:        ErrorCode(err).String(),
			Err:           err,
		}
	}

	type normalizedRule struct {
		rule      Rule
		canonical string
	}

	var forcedBlocks []normalizedRule
	var explicitAllows []normalizedRule

	for _, r := range profile.Rules {
		rn, err := NormalizeDomain(r.Domain)
		if err != nil {
			continue
		}
		switch r.EffectiveLayer() {
		case LayerAdminBlock:
			if r.Action == ActionBlock {
				forcedBlocks = append(forcedBlocks, normalizedRule{rule: r, canonical: rn.Canonical})
			}
		case LayerAdminAllow:
			if r.Action == ActionAllow {
				explicitAllows = append(explicitAllows, normalizedRule{rule: r, canonical: rn.Canonical})
			}
		}
	}

	// Deterministic specificity sort within each layer: longest canonical domain
	// first (most specific match), then exact match before subdomain match, then
	// RuleID ascending.
	sortRules := func(slice []normalizedRule) {
		sort.Slice(slice, func(i, j int) bool {
			if len(slice[i].canonical) != len(slice[j].canonical) {
				return len(slice[i].canonical) > len(slice[j].canonical)
			}
			if slice[i].rule.IncludeSubdomains != slice[j].rule.IncludeSubdomains {
				return !slice[i].rule.IncludeSubdomains
			}
			return slice[i].rule.RuleID < slice[j].rule.RuleID
		})
	}
	sortRules(forcedBlocks)
	sortRules(explicitAllows)

	// Layer 1: Admin forced block (highest precedence; an allow never beats a forced block).
	for _, cand := range forcedBlocks {
		if MatchesDomain(cand.canonical, cand.rule.IncludeSubdomains, norm.Canonical) {
			reason := cand.rule.Reason
			if reason == "" {
				reason = "admin_block"
			}
			return Decision{
				Domain:        norm.Canonical,
				OriginalInput: name,
				Action:        ActionBlock,
				Layer:         LayerAdminBlock,
				MatchedRuleID: cand.rule.RuleID,
				MatchedDomain: cand.canonical,
				Reason:        reason,
			}
		}
	}

	// Layer 2: Admin explicit allow (overrides Layer 3 lists, never overrides Layer 1).
	for _, cand := range explicitAllows {
		if MatchesDomain(cand.canonical, cand.rule.IncludeSubdomains, norm.Canonical) {
			reason := cand.rule.Reason
			if reason == "" {
				reason = "admin_allow"
			}
			return Decision{
				Domain:        norm.Canonical,
				OriginalInput: name,
				Action:        ActionAllow,
				Layer:         LayerAdminAllow,
				MatchedRuleID: cand.rule.RuleID,
				MatchedDomain: cand.canonical,
				Reason:        reason,
			}
		}
	}

	// Layer 3: Enabled security and category lists.
	sources, _ := resolveProfileSources(profile, nil)
	// Sort sources deterministically by SourceID if not already ordered.
	sortedSources := make([]ListSource, len(sources))
	copy(sortedSources, sources)
	sort.Slice(sortedSources, func(i, j int) bool {
		return sortedSources[i].SourceID < sortedSources[j].SourceID
	})

	type sourceMatch struct {
		sourceID  string
		category  SourceCategory
		canonical string
	}
	var bestSourceMatch *sourceMatch

	for _, src := range sortedSources {
		for _, entry := range src.Entries {
			en, err := NormalizeDomain(entry.Domain)
			if err != nil {
				continue
			}
			if MatchesDomain(en.Canonical, entry.IncludeSubdomains, norm.Canonical) {
				if bestSourceMatch == nil ||
					len(en.Canonical) > len(bestSourceMatch.canonical) ||
					(len(en.Canonical) == len(bestSourceMatch.canonical) && src.SourceID < bestSourceMatch.sourceID) {
					bestSourceMatch = &sourceMatch{
						sourceID:  src.SourceID,
						category:  src.Category,
						canonical: en.Canonical,
					}
				}
			}
		}
	}
	if bestSourceMatch != nil {
		reason := string(bestSourceMatch.category)
		if reason == "" {
			reason = "source_list"
		}
		return Decision{
			Domain:          norm.Canonical,
			OriginalInput:   name,
			Action:          ActionBlock,
			Layer:           LayerSourceList,
			MatchedSourceID: bestSourceMatch.sourceID,
			MatchedDomain:   bestSourceMatch.canonical,
			Reason:          reason,
		}
	}

	// Layer 4: Normal resolution.
	return Decision{
		Domain:        norm.Canonical,
		OriginalInput: name,
		Action:        ActionAllow,
		Layer:         LayerDefaultResolve,
		Reason:        "default_resolve",
	}
}

func (c ValidationErrorCode) String() string {
	return string(c)
}
