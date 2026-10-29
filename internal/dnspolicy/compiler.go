package dnspolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// BundleSchemaVersion is the current schema version for compiled DNS bundles.
	BundleSchemaVersion = 1

	// BlockingModeNXDOMAIN is the default DNS block response mode (§12.5).
	BlockingModeNXDOMAIN = "nxdomain"

	// DefaultBlockedTTLSeconds is the recommended short default TTL (in seconds)
	// for blocked DNS responses (§12.5).
	DefaultBlockedTTLSeconds uint32 = 60

	// HeaderCommentPrefix is the prefix for metadata comments embedded in
	// AdGuard Home user_rules.
	HeaderProfileIDPrefix  = "! fvpn-profile-id: "
	HeaderRevisionPrefix   = "! fvpn-revision: "
	HeaderBundleHashPrefix = "! fvpn-bundle-sha256: "
	HeaderValidFromPrefix  = "! fvpn-valid-from: "
	HeaderValidUntilPrefix = "! fvpn-valid-until: "
)

// SmokeTestExpectation defines an embedded post-apply verification query and
// its expected policy decision (§12.5).
type SmokeTestExpectation struct {
	Domain         string      `json:"domain"`
	ExpectedAction RuleAction  `json:"expected_action"`
	ExpectedLayer  PolicyLayer `json:"expected_layer"`
}

// AdGuardSettings holds the backend-specific AdGuard Home configuration
// enforced by a compiled bundle (§12.2, §12.5, §12.7).
type AdGuardSettings struct {
	QueryLogEnabled        bool     `json:"querylog_enabled"`
	QueryLogFileEnabled    bool     `json:"querylog_file_enabled"`
	StatisticsEnabled      bool     `json:"statistics_enabled"`
	AnonymizeClientIP      bool     `json:"anonymize_client_ip"`
	BlockingMode           string   `json:"blocking_mode"`
	BlockedResponseTTL     uint32   `json:"blocked_response_ttl"`
	UpstreamDNS            []string `json:"upstream_dns"`
	BootstrapDNS           []string `json:"bootstrap_dns"`
	FallbackDNS            []string `json:"fallback_dns"`
	AllowedClients         []string `json:"allowed_clients"`
	ManagementBindAddress  string   `json:"management_bind_address"`
	FilteringEnabled       bool     `json:"filtering_enabled"`
	ProtectionEnabled      bool     `json:"protection_enabled"`
	ParentalEnabled        bool     `json:"parental_enabled"`
	SafeBrowsingEnabled    bool     `json:"safebrowsing_enabled"`
	SafeSearchEnabled      bool     `json:"safesearch_enabled"`
	HostsFileEnabled       bool     `json:"hostsfile_enabled"`
	UsePrivatePTRResolvers bool     `json:"use_private_ptr_resolvers"`
}

// CompiledBundle is an immutable, backend-specific resolver bundle with a
// revision, SHA-256 of the exact compiled bytes, validity window, and embedded
// smoke-test expectations (§12.5).
type CompiledBundle struct {
	SchemaVersion int                    `json:"schema_version"`
	ProfileID     string                 `json:"profile_id"`
	Revision      string                 `json:"revision"`
	SHA256        string                 `json:"sha256"`
	ValidFrom     time.Time              `json:"valid_from"`
	ValidUntil    time.Time              `json:"valid_until"`
	Settings      AdGuardSettings        `json:"settings"`
	UserRules     []string               `json:"user_rules"`
	RuleCount     int                    `json:"rule_count"`
	SmokeTests    []SmokeTestExpectation `json:"smoke_tests"`
	CompiledBytes []byte                 `json:"-"`
}

type canonicalBundlePayload struct {
	SchemaVersion int                    `json:"schema_version"`
	ProfileID     string                 `json:"profile_id"`
	Revision      string                 `json:"revision"`
	ValidFrom     string                 `json:"valid_from"`
	ValidUntil    string                 `json:"valid_until"`
	Settings      AdGuardSettings        `json:"settings"`
	UserRules     []string               `json:"user_rules"`
	RuleCount     int                    `json:"rule_count"`
	SmokeTests    []SmokeTestExpectation `json:"smoke_tests"`
}

// CompileOptions provides backend environment parameters and validation bounds
// when compiling a Profile into a CompiledBundle.
type CompileOptions struct {
	ValidFrom             time.Time
	ValidUntil            time.Time
	UpstreamDNS           []string
	BootstrapDNS          []string
	AllowedClients        []string
	ManagementBindAddress string
	BlockedTTLSeconds     uint32
	Registry              *SourceRegistry
	Limits                *DraftValidationLimits
}

// RenderAdGuardRule renders a single canonical domain rule into AdGuard Home's
// filtering rule syntax preserving the 4-layer precedence model (§12.4, §12.5):
//
//   - Layer 1 (LayerAdminBlock):
//     includeSubdomains=true  -> "||domain^$important"
//     includeSubdomains=false -> "|domain^$important"
//   - Layer 2 (LayerAdminAllow):
//     includeSubdomains=true  -> "@@||domain^"
//     includeSubdomains=false -> "@@|domain^"
//   - Layer 3 (LayerSourceList):
//     includeSubdomains=true  -> "||domain^"
//     includeSubdomains=false -> "|domain^"
func RenderAdGuardRule(domain string, includeSubdomains bool, layer PolicyLayer) (string, error) {
	norm, err := NormalizeDomain(domain)
	if err != nil {
		return "", err
	}
	prefix := "|"
	if includeSubdomains {
		prefix = "||"
	}
	switch layer {
	case LayerAdminBlock:
		return prefix + norm.Canonical + "^$important", nil
	case LayerAdminAllow:
		return "@@" + prefix + norm.Canonical + "^", nil
	case LayerSourceList:
		return prefix + norm.Canonical + "^", nil
	default:
		return "", &ValidationError{
			Code:    ErrCodeRuleInvalidLayer,
			Field:   "layer",
			Input:   string(layer),
			Message: fmt.Sprintf("cannot render AdGuard rule for layer %q", layer),
		}
	}
}

// ValidateAdGuardRuleSyntax performs strict syntax validation on a single line
// of AdGuard Home user_rules before staging/applying (§12.5).
func ValidateAdGuardRuleSyntax(line string) error {
	if line == "" {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "user_rules",
			Input:   line,
			Message: "empty rule line is not permitted in compiled bundle",
		}
	}
	for _, r := range line {
		if r <= 0x1F || r == 0x7F {
			return &ValidationError{
				Code:    ErrCodeBundleSyntaxInvalid,
				Field:   "user_rules",
				Input:   line,
				Message: "rule line contains control characters",
			}
		}
	}
	if strings.HasPrefix(line, "!") {
		return nil
	}

	s := line
	isAllow := false
	if strings.HasPrefix(s, "@@") {
		isAllow = true
		s = s[2:]
	}

	hasImportant := false
	if strings.HasSuffix(s, "$important") {
		hasImportant = true
		s = strings.TrimSuffix(s, "$important")
	}

	// Never permit @@...$important because allow rules (Layer 2) must never beat
	// admin forced blocks (Layer 1, which use $important) (§12.4).
	if isAllow && hasImportant {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "user_rules",
			Input:   line,
			Message: "allow rule must not carry $important modifier (allow never beats forced block)",
		}
	}

	if strings.HasPrefix(s, "||") {
		s = s[2:]
	} else if strings.HasPrefix(s, "|") {
		s = s[1:]
	} else {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "user_rules",
			Input:   line,
			Message: "rule must start with '||' (subdomain match) or '|' (exact match)",
		}
	}

	if !strings.HasSuffix(s, "^") {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "user_rules",
			Input:   line,
			Message: "rule domain must end with '^' separator",
		}
	}
	domainPart := s[:len(s)-1]
	norm, err := NormalizeDomain(domainPart)
	if err != nil {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "user_rules",
			Input:   line,
			Message: fmt.Sprintf("rule contains invalid domain %q: %v", domainPart, err),
			Cause:   err,
		}
	}
	if norm.Canonical != domainPart {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "user_rules",
			Input:   line,
			Message: fmt.Sprintf("rule domain %q is not in canonical ASCII form %q", domainPart, norm.Canonical),
		}
	}

	return nil
}

// CompileBundle validates profile and compiles it with its rules and enabled
// sources into a deterministic, immutable CompiledBundle (§12.4, §12.5).
func CompileBundle(profile Profile, opts CompileOptions) (*CompiledBundle, error) {
	if err := ValidateDraft(profile, opts.Registry, opts.Limits); err != nil {
		return nil, err
	}

	validFrom := opts.ValidFrom.UTC()
	validUntil := opts.ValidUntil.UTC()
	if validFrom.IsZero() || validUntil.IsZero() || !validFrom.Before(validUntil) {
		return nil, &ValidationError{
			Code:    ErrCodeBundleInvalidWindow,
			Field:   "valid_until",
			Message: "bundle validity interval [valid_from, valid_until) must have non-zero times with valid_from < valid_until",
		}
	}

	if len(opts.UpstreamDNS) == 0 {
		return nil, &ValidationError{
			Code:    ErrCodeBundleNoUpstreams,
			Field:   "upstream_dns",
			Message: "at least one configured upstream DNS server is required (no fallback to arbitrary public DNS)",
		}
	}
	upstreams := make([]string, 0, len(opts.UpstreamDNS))
	for _, u := range opts.UpstreamDNS {
		trimmed := strings.TrimSpace(u)
		if trimmed == "" {
			return nil, &ValidationError{
				Code:    ErrCodeBundleNoUpstreams,
				Field:   "upstream_dns",
				Message: "upstream DNS entry must not be empty",
			}
		}
		upstreams = append(upstreams, trimmed)
	}

	bootstraps := make([]string, 0, len(opts.BootstrapDNS))
	for _, b := range opts.BootstrapDNS {
		if trimmed := strings.TrimSpace(b); trimmed != "" {
			bootstraps = append(bootstraps, trimmed)
		}
	}

	allowedClients := make([]string, 0, len(opts.AllowedClients))
	for _, c := range opts.AllowedClients {
		if trimmed := strings.TrimSpace(c); trimmed != "" {
			allowedClients = append(allowedClients, trimmed)
		}
	}
	if len(allowedClients) == 0 {
		allowedClients = []string{"127.0.0.0/8", "::1/128", "100.64.0.0/10"}
	}

	mgmtAddr := strings.TrimSpace(opts.ManagementBindAddress)
	if mgmtAddr == "" {
		mgmtAddr = "127.0.0.1:8080"
	}

	blockMode := strings.ToLower(strings.TrimSpace(profile.BlockMode))
	if blockMode == "" {
		blockMode = BlockingModeNXDOMAIN
	}
	if blockMode != BlockingModeNXDOMAIN {
		return nil, &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "block_mode",
			Input:   profile.BlockMode,
			Message: fmt.Sprintf("unsupported block_mode %q (v1 requires %q)", profile.BlockMode, BlockingModeNXDOMAIN),
		}
	}

	blockedTTL := profile.BlockedTTLSeconds
	if opts.BlockedTTLSeconds > 0 {
		blockedTTL = opts.BlockedTTLSeconds
	}
	if blockedTTL == 0 {
		blockedTTL = DefaultBlockedTTLSeconds
	}

	resolvedSources, err := resolveProfileSources(profile, opts.Registry)
	if err != nil {
		return nil, err
	}
	// Attach resolved sources to profile copy so Evaluate sees the full picture.
	evalProfile := profile
	evalProfile.Sources = resolvedSources

	type renderedEntry struct {
		canonical         string
		includeSubdomains bool
		ruleID            string
		rendered          string
	}

	var layer1 []renderedEntry
	var layer2 []renderedEntry

	for _, r := range profile.Rules {
		norm, err := NormalizeDomain(r.Domain)
		if err != nil {
			return nil, err
		}
		layer := r.EffectiveLayer()
		line, err := RenderAdGuardRule(norm.Canonical, r.IncludeSubdomains, layer)
		if err != nil {
			return nil, err
		}
		re := renderedEntry{
			canonical:         norm.Canonical,
			includeSubdomains: r.IncludeSubdomains,
			ruleID:            r.RuleID,
			rendered:          line,
		}
		switch layer {
		case LayerAdminBlock:
			layer1 = append(layer1, re)
		case LayerAdminAllow:
			layer2 = append(layer2, re)
		}
	}

	sortRendered := func(slice []renderedEntry) {
		sort.Slice(slice, func(i, j int) bool {
			if slice[i].canonical != slice[j].canonical {
				return slice[i].canonical < slice[j].canonical
			}
			if slice[i].includeSubdomains != slice[j].includeSubdomains {
				return slice[i].includeSubdomains
			}
			return slice[i].ruleID < slice[j].ruleID
		})
	}
	sortRendered(layer1)
	sortRendered(layer2)

	// Sort sources deterministically by SourceID.
	sort.Slice(resolvedSources, func(i, j int) bool {
		return resolvedSources[i].SourceID < resolvedSources[j].SourceID
	})

	type sourceKey struct {
		canonical         string
		includeSubdomains bool
	}
	seenSourceRules := make(map[sourceKey]struct{})
	var layer3 []renderedEntry

	for _, src := range resolvedSources {
		for _, entry := range src.Entries {
			norm, err := NormalizeDomain(entry.Domain)
			if err != nil {
				return nil, err
			}
			sk := sourceKey{canonical: norm.Canonical, includeSubdomains: entry.IncludeSubdomains}
			if _, exists := seenSourceRules[sk]; exists {
				continue
			}
			seenSourceRules[sk] = struct{}{}
			line, err := RenderAdGuardRule(norm.Canonical, entry.IncludeSubdomains, LayerSourceList)
			if err != nil {
				return nil, err
			}
			layer3 = append(layer3, renderedEntry{
				canonical:         norm.Canonical,
				includeSubdomains: entry.IncludeSubdomains,
				ruleID:            src.SourceID,
				rendered:          line,
			})
		}
	}
	sortRendered(layer3)

	userRules := []string{
		"! fvpn-bundle-version: 1",
		HeaderProfileIDPrefix + profile.ProfileID,
		HeaderRevisionPrefix + profile.Revision,
		HeaderValidFromPrefix + validFrom.Format(time.RFC3339),
		HeaderValidUntilPrefix + validUntil.Format(time.RFC3339),
	}

	if len(layer1) > 0 {
		userRules = append(userRules, "! layer-1: admin_block")
		for _, item := range layer1 {
			userRules = append(userRules, item.rendered)
		}
	}
	if len(layer2) > 0 {
		userRules = append(userRules, "! layer-2: admin_allow")
		for _, item := range layer2 {
			userRules = append(userRules, item.rendered)
		}
	}
	if len(layer3) > 0 {
		userRules = append(userRules, "! layer-3: source_list")
		for _, item := range layer3 {
			userRules = append(userRules, item.rendered)
		}
	}

	activeRuleCount := len(layer1) + len(layer2) + len(layer3)

	// Build deterministic smoke tests.
	smokeDomainsMap := make(map[string]SmokeTestExpectation)
	for _, st := range profile.SmokeTests {
		norm, err := NormalizeDomain(st.Domain)
		if err != nil {
			return nil, err
		}
		dec := Evaluate(evalProfile, norm.Canonical)
		expAction := st.ExpectedAction
		if expAction == "" {
			expAction = dec.Action
		}
		expLayer := st.ExpectedLayer
		if expLayer == "" {
			expLayer = dec.Layer
		}
		if expAction != dec.Action || expLayer != dec.Layer {
			return nil, &ValidationError{
				Code:    ErrCodeBundleSyntaxInvalid,
				Field:   "smoke_tests",
				Input:   st.Domain,
				Message: fmt.Sprintf("smoke test expectation for %q (action=%s, layer=%s) contradicts Evaluate decision (action=%s, layer=%s)", norm.Canonical, expAction, expLayer, dec.Action, dec.Layer),
			}
		}
		smokeDomainsMap[norm.Canonical] = SmokeTestExpectation{
			Domain:         norm.Canonical,
			ExpectedAction: expAction,
			ExpectedLayer:  expLayer,
		}
	}

	// Automatically embed representative smoke test expectations from each populated layer.
	addAutoSmoke := func(canonical string) {
		if _, exists := smokeDomainsMap[canonical]; exists {
			return
		}
		dec := Evaluate(evalProfile, canonical)
		smokeDomainsMap[canonical] = SmokeTestExpectation{
			Domain:         canonical,
			ExpectedAction: dec.Action,
			ExpectedLayer:  dec.Layer,
		}
	}
	if len(layer1) > 0 {
		addAutoSmoke(layer1[0].canonical)
	}
	if len(layer2) > 0 {
		addAutoSmoke(layer2[0].canonical)
	}
	if len(layer3) > 0 {
		addAutoSmoke(layer3[0].canonical)
	}

	smokeTests := make([]SmokeTestExpectation, 0, len(smokeDomainsMap))
	for _, st := range smokeDomainsMap {
		smokeTests = append(smokeTests, st)
	}
	sort.Slice(smokeTests, func(i, j int) bool {
		return smokeTests[i].Domain < smokeTests[j].Domain
	})

	settings := AdGuardSettings{
		QueryLogEnabled:        false,
		QueryLogFileEnabled:    false,
		StatisticsEnabled:      false,
		AnonymizeClientIP:      true,
		BlockingMode:           blockMode,
		BlockedResponseTTL:     blockedTTL,
		UpstreamDNS:            upstreams,
		BootstrapDNS:           bootstraps,
		FallbackDNS:            []string{},
		AllowedClients:         allowedClients,
		ManagementBindAddress:  mgmtAddr,
		FilteringEnabled:       true,
		ProtectionEnabled:      true,
		ParentalEnabled:        false,
		SafeBrowsingEnabled:    false,
		SafeSearchEnabled:      false,
		HostsFileEnabled:       false,
		UsePrivatePTRResolvers: false,
	}

	payload := canonicalBundlePayload{
		SchemaVersion: BundleSchemaVersion,
		ProfileID:     profile.ProfileID,
		Revision:      profile.Revision,
		ValidFrom:     validFrom.Format(time.RFC3339),
		ValidUntil:    validUntil.Format(time.RFC3339),
		Settings:      settings,
		UserRules:     userRules,
		RuleCount:     activeRuleCount,
		SmokeTests:    smokeTests,
	}

	compiledBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical bundle payload: %w", err)
	}
	sum := sha256.Sum256(compiledBytes)
	digest := hex.EncodeToString(sum[:])

	return &CompiledBundle{
		SchemaVersion: BundleSchemaVersion,
		ProfileID:     profile.ProfileID,
		Revision:      profile.Revision,
		SHA256:        digest,
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
		Settings:      settings,
		UserRules:     userRules,
		RuleCount:     activeRuleCount,
		SmokeTests:    smokeTests,
		CompiledBytes: compiledBytes,
	}, nil
}

// AdGuardUserRulesWithHash returns the UserRules slice with the bundle SHA-256
// header line inserted for atomic application and read-back verification on
// AdGuard Home.
func (b *CompiledBundle) AdGuardUserRulesWithHash() []string {
	if b == nil {
		return nil
	}
	out := make([]string, 0, len(b.UserRules)+1)
	inserted := false
	for _, line := range b.UserRules {
		out = append(out, line)
		if !inserted && strings.HasPrefix(line, HeaderRevisionPrefix) {
			out = append(out, HeaderBundleHashPrefix+b.SHA256)
			inserted = true
		}
	}
	if !inserted {
		out = append([]string{HeaderBundleHashPrefix + b.SHA256}, out...)
	}
	return out
}

// VerifyBundle validates a CompiledBundle's integrity, validity window [ValidFrom, ValidUntil),
// privacy settings (§12.7), upstream settings, and every rule's AdGuard syntax (§12.5).
func VerifyBundle(bundle *CompiledBundle, now time.Time) error {
	if bundle == nil {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "bundle",
			Message: "bundle must not be nil",
		}
	}
	if strings.TrimSpace(bundle.ProfileID) == "" {
		return &ValidationError{
			Code:    ErrCodeProfileIDEmpty,
			Field:   "profile_id",
			Message: "bundle profile_id must not be empty",
		}
	}
	if err := ValidateDecimalRevision(bundle.Revision); err != nil {
		return err
	}
	if bundle.ValidFrom.IsZero() || bundle.ValidUntil.IsZero() || !bundle.ValidFrom.Before(bundle.ValidUntil) {
		return &ValidationError{
			Code:    ErrCodeBundleInvalidWindow,
			Field:   "valid_until",
			Message: "bundle validity interval [valid_from, valid_until) is invalid",
		}
	}
	if !now.IsZero() {
		nowUTC := now.UTC()
		if nowUTC.Before(bundle.ValidFrom) || !nowUTC.Before(bundle.ValidUntil) {
			return &ValidationError{
				Code:    ErrCodeBundleExpired,
				Field:   "valid_until",
				Message: fmt.Sprintf("bundle valid window [%s, %s) does not cover current time %s", bundle.ValidFrom.Format(time.RFC3339), bundle.ValidUntil.Format(time.RFC3339), nowUTC.Format(time.RFC3339)),
			}
		}
	}

	// Verify exact compiled bytes SHA-256.
	if len(bundle.CompiledBytes) == 0 {
		return &ValidationError{
			Code:    ErrCodeBundleHashMismatch,
			Field:   "compiled_bytes",
			Message: "bundle compiled_bytes is empty",
		}
	}
	sum := sha256.Sum256(bundle.CompiledBytes)
	computed := hex.EncodeToString(sum[:])
	if !strings.EqualFold(computed, bundle.SHA256) {
		return &ValidationError{
			Code:    ErrCodeBundleHashMismatch,
			Field:   "sha256",
			Input:   bundle.SHA256,
			Message: fmt.Sprintf("bundle sha256 %q does not match compiled bytes digest %q", bundle.SHA256, computed),
		}
	}

	// Verify mandatory privacy and security settings (§12.2, §12.7).
	if bundle.Settings.QueryLogEnabled || bundle.Settings.QueryLogFileEnabled || bundle.Settings.StatisticsEnabled {
		return &ValidationError{
			Code:    ErrCodeBundlePrivacyInvalid,
			Field:   "settings",
			Message: "bundle must enforce querylog.enabled=false, querylog.file_enabled=false, and statistics.enabled=false (§12.7, SEC02)",
		}
	}
	if len(bundle.Settings.UpstreamDNS) == 0 {
		return &ValidationError{
			Code:    ErrCodeBundleNoUpstreams,
			Field:   "settings.upstream_dns",
			Message: "bundle must configure at least one upstream DNS resolver",
		}
	}
	if len(bundle.Settings.FallbackDNS) > 0 {
		return &ValidationError{
			Code:    ErrCodeBundleNoUpstreams,
			Field:   "settings.fallback_dns",
			Message: "bundle must not configure fallback_dns to arbitrary DNS servers (§12.2)",
		}
	}
	if bundle.Settings.BlockingMode != BlockingModeNXDOMAIN {
		return &ValidationError{
			Code:    ErrCodeBundleSyntaxInvalid,
			Field:   "settings.blocking_mode",
			Input:   bundle.Settings.BlockingMode,
			Message: fmt.Sprintf("bundle blocking_mode must be %q", BlockingModeNXDOMAIN),
		}
	}

	for _, line := range bundle.UserRules {
		if err := ValidateAdGuardRuleSyntax(line); err != nil {
			return err
		}
	}

	return nil
}
