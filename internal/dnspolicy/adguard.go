package dnspolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	// PinnedAdGuardVersion is the exact AdGuard Home version pinned for v1 (§3.2, §12.2).
	PinnedAdGuardVersion = "v0.107.59"

	// PinnedAdGuardImageDigest is the multi-arch OCI index digest for AdGuard Home v0.107.59.
	PinnedAdGuardImageDigest = "sha256:5d5e3aef39a8c8906a324f0c2da0a6123929e459b57c1e3f05e401dca5c7bf02"

	// PinnedAdGuardImage is the full pinned container image reference.
	PinnedAdGuardImage = "docker.io/adguard/adguardhome:" + PinnedAdGuardVersion + "@" + PinnedAdGuardImageDigest

	// PinnedAdGuardLicense is the upstream license of AdGuard Home.
	PinnedAdGuardLicense = "GPL-3.0"
)

// MultiResolverStatus summarizes the outcome of publishing a bundle across
// multiple resolvers (§12.5).
type MultiResolverStatus string

const (
	PublishAllApplied MultiResolverStatus = "all_applied"
	PublishPartial    MultiResolverStatus = "partial"
	PublishFailed     MultiResolverStatus = "failed"
)

// ResolverConfig holds connection and authentication parameters for a single
// AdGuard Home resolver instance (§12.2).
type ResolverConfig struct {
	ID            string
	AdminURL      string
	DNSAddr       string
	Username      string
	Password      string
	HTTPClient    *http.Client
	DNSClient     *DNSClient
	DataDir       string
	ContainerName string
}

// AppliedResolverState represents the live observed configuration and bundle
// metadata read back from an AdGuard Home resolver (§2, §12.5, §12.7).
type AppliedResolverState struct {
	ResolverID        string   `json:"resolver_id"`
	Version           string   `json:"version"`
	Running           bool     `json:"running"`
	ProtectionEnabled bool     `json:"protection_enabled"`
	FilteringEnabled  bool     `json:"filtering_enabled"`
	ProfileID         string   `json:"profile_id"`
	AppliedRevision   string   `json:"applied_revision"`
	AppliedSHA256     string   `json:"applied_sha256"`
	ValidFrom         string   `json:"valid_from,omitempty"`
	ValidUntil        string   `json:"valid_until,omitempty"`
	UserRules         []string `json:"user_rules"`
	ExternalFilters   int      `json:"external_filters_enabled"`
	QueryLogEnabled   bool     `json:"querylog_enabled"`
	StatisticsEnabled bool     `json:"statistics_enabled"`
	BlockingMode      string   `json:"blocking_mode"`
	BlockedTTL        uint32   `json:"blocked_response_ttl"`
	UpstreamDNS       []string `json:"upstream_dns"`
	BootstrapDNS      []string `json:"bootstrap_dns"`
	FallbackDNS       []string `json:"fallback_dns"`
	AllowedClients    []string `json:"allowed_clients"`
}

// CheckHostResponse models the JSON response from AdGuard Home's
// GET /control/filtering/check_host endpoint.
type CheckHostResponse struct {
	Reason string `json:"reason"`
	Rule   string `json:"rule"`
	Rules  []struct {
		Text         string `json:"text"`
		FilterListID int    `json:"filter_list_id"`
	} `json:"rules"`
	ServiceName string   `json:"service_name"`
	CNAME       string   `json:"cname"`
	IPAddrs     []string `json:"ip_addrs"`
}

// IsBlocked reports whether AdGuard Home's check_host reason indicates that the
// domain is blocked by filtering rules.
func (r *CheckHostResponse) IsBlocked() bool {
	if r == nil {
		return false
	}
	return strings.HasPrefix(r.Reason, "Filtered")
}

// ResolverPublishResult reports the outcome of publishing a bundle to a single
// resolver instance (§12.5).
type ResolverPublishResult struct {
	ResolverID       string `json:"resolver_id"`
	Success          bool   `json:"success"`
	NoOp             bool   `json:"noop"`
	RolledBack       bool   `json:"rolled_back"`
	TargetRevision   string `json:"target_revision"`
	AppliedRevision  string `json:"applied_revision"`
	AppliedSHA256    string `json:"applied_sha256"`
	PreviousRevision string `json:"previous_revision,omitempty"`
	Reason           string `json:"reason,omitempty"`
	Err              error  `json:"-"`
}

// MultiPublishReport aggregates publish results across all resolvers (§12.5).
// AllSucceeded is true if and only if Status == PublishAllApplied.
type MultiPublishReport struct {
	Status         MultiResolverStatus     `json:"status"`
	AllSucceeded   bool                    `json:"all_succeeded"`
	TargetRevision string                  `json:"target_revision"`
	TargetSHA256   string                  `json:"target_sha256"`
	Results        []ResolverPublishResult `json:"results"`
}

type resolverSnapshot struct {
	revision  string
	sha256    string
	userRules []string
	settings  AdGuardSettings
}

// AdGuardAdapter manages policy staging, syntax checking, atomic application,
// smoke testing, and last-good rollback for a single AdGuard Home resolver (§12.5).
type AdGuardAdapter struct {
	mu       sync.Mutex
	cfg      ResolverConfig
	httpCli  *http.Client
	dnsCli   *DNSClient
	lastGood *resolverSnapshot
	nowFunc  func() time.Time
	// faultHook is an optional test hook invoked after applying rules and before
	// completing smoke tests to test post-apply failure and rollback.
	faultHook func(ctx context.Context, resolverID string, bundle *CompiledBundle) error
}

// NewAdGuardAdapter creates a new AdGuardAdapter for a single resolver.
func NewAdGuardAdapter(cfg ResolverConfig) *AdGuardAdapter {
	httpCli := cfg.HTTPClient
	if httpCli == nil {
		httpCli = &http.Client{Timeout: 10 * time.Second}
	}
	dnsCli := cfg.DNSClient
	if dnsCli == nil {
		dnsCli = NewDNSClient(3 * time.Second)
	}
	return &AdGuardAdapter{
		cfg:     cfg,
		httpCli: httpCli,
		dnsCli:  dnsCli,
		nowFunc: time.Now,
	}
}

// ID returns the resolver identifier.
func (a *AdGuardAdapter) ID() string {
	return a.cfg.ID
}

// Config returns the resolver configuration.
func (a *AdGuardAdapter) Config() ResolverConfig {
	return a.cfg
}

// SetNowFunc overrides the clock function used for bundle validity checks in tests.
func (a *AdGuardAdapter) SetNowFunc(fn func() time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nowFunc = fn
}

// SetFaultHook installs or clears a test fault-injection hook that runs during
// the smoke-test phase of Publish.
func (a *AdGuardAdapter) SetFaultHook(fn func(ctx context.Context, resolverID string, bundle *CompiledBundle) error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.faultHook = fn
}

// StageAndValidate stages a bundle in memory and validates its hash, validity
// window, privacy settings, upstream settings, and every rule's AdGuard syntax (§12.5).
func (a *AdGuardAdapter) StageAndValidate(bundle *CompiledBundle) error {
	a.mu.Lock()
	nowFn := a.nowFunc
	a.mu.Unlock()
	var now time.Time
	if nowFn != nil {
		now = nowFn()
	}
	return VerifyBundle(bundle, now)
}

// GetAppliedState queries the live AdGuard Home resolver API and returns its
// observed configuration, privacy settings, and applied bundle revision/hash.
func (a *AdGuardAdapter) GetAppliedState(ctx context.Context) (*AppliedResolverState, error) {
	var statusResp struct {
		Version           string `json:"version"`
		ProtectionEnabled bool   `json:"protection_enabled"`
		Running           bool   `json:"running"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/status", nil, &statusResp); err != nil {
		return nil, fmt.Errorf("get status from resolver %s: %w", a.cfg.ID, err)
	}

	var dnsInfo struct {
		UpstreamDNS        []string `json:"upstream_dns"`
		BootstrapDNS       []string `json:"bootstrap_dns"`
		FallbackDNS        []string `json:"fallback_dns"`
		BlockingMode       string   `json:"blocking_mode"`
		BlockedResponseTTL uint32   `json:"blocked_response_ttl"`
		ProtectionEnabled  bool     `json:"protection_enabled"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/dns_info", nil, &dnsInfo); err != nil {
		return nil, fmt.Errorf("get dns_info from resolver %s: %w", a.cfg.ID, err)
	}

	var filterStatus struct {
		Enabled bool `json:"enabled"`
		Filters []struct {
			ID      int    `json:"id"`
			URL     string `json:"url"`
			Enabled bool   `json:"enabled"`
		} `json:"filters"`
		UserRules []string `json:"user_rules"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/filtering/status", nil, &filterStatus); err != nil {
		return nil, fmt.Errorf("get filtering/status from resolver %s: %w", a.cfg.ID, err)
	}

	var qlConfig struct {
		Enabled bool `json:"enabled"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/querylog/config", nil, &qlConfig); err != nil {
		return nil, fmt.Errorf("get querylog/config from resolver %s: %w", a.cfg.ID, err)
	}

	var statsConfig struct {
		Enabled bool `json:"enabled"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/stats/config", nil, &statsConfig); err != nil {
		return nil, fmt.Errorf("get stats/config from resolver %s: %w", a.cfg.ID, err)
	}

	var accessList struct {
		AllowedClients []string `json:"allowed_clients"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/access/list", nil, &accessList); err != nil {
		return nil, fmt.Errorf("get access/list from resolver %s: %w", a.cfg.ID, err)
	}

	enabledFilters := 0
	for _, f := range filterStatus.Filters {
		if f.Enabled {
			enabledFilters++
		}
	}

	state := &AppliedResolverState{
		ResolverID:        a.cfg.ID,
		Version:           statusResp.Version,
		Running:           statusResp.Running,
		ProtectionEnabled: statusResp.ProtectionEnabled && dnsInfo.ProtectionEnabled,
		FilteringEnabled:  filterStatus.Enabled,
		UserRules:         filterStatus.UserRules,
		ExternalFilters:   enabledFilters,
		QueryLogEnabled:   qlConfig.Enabled,
		StatisticsEnabled: statsConfig.Enabled,
		BlockingMode:      dnsInfo.BlockingMode,
		BlockedTTL:        dnsInfo.BlockedResponseTTL,
		UpstreamDNS:       dnsInfo.UpstreamDNS,
		BootstrapDNS:      dnsInfo.BootstrapDNS,
		FallbackDNS:       dnsInfo.FallbackDNS,
		AllowedClients:    accessList.AllowedClients,
	}

	for _, line := range filterStatus.UserRules {
		switch {
		case strings.HasPrefix(line, HeaderProfileIDPrefix):
			state.ProfileID = strings.TrimSpace(strings.TrimPrefix(line, HeaderProfileIDPrefix))
		case strings.HasPrefix(line, HeaderRevisionPrefix):
			state.AppliedRevision = strings.TrimSpace(strings.TrimPrefix(line, HeaderRevisionPrefix))
		case strings.HasPrefix(line, HeaderBundleHashPrefix):
			state.AppliedSHA256 = strings.TrimSpace(strings.TrimPrefix(line, HeaderBundleHashPrefix))
		case strings.HasPrefix(line, HeaderValidFromPrefix):
			state.ValidFrom = strings.TrimSpace(strings.TrimPrefix(line, HeaderValidFromPrefix))
		case strings.HasPrefix(line, HeaderValidUntilPrefix):
			state.ValidUntil = strings.TrimSpace(strings.TrimPrefix(line, HeaderValidUntilPrefix))
		}
	}

	return state, nil
}

// CheckHost queries AdGuard Home's /control/filtering/check_host endpoint for name.
func (a *AdGuardAdapter) CheckHost(ctx context.Context, name string) (*CheckHostResponse, error) {
	norm, err := NormalizeDomain(name)
	if err != nil {
		return nil, err
	}
	path := "/control/filtering/check_host?name=" + url.QueryEscape(norm.Canonical)
	var out CheckHostResponse
	if err := a.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetQueryLogItems returns the entries in AdGuard Home's query log via
// GET /control/querylog (used to verify SEC02 / AT22 data minimization).
func (a *AdGuardAdapter) GetQueryLogItems(ctx context.Context) ([]json.RawMessage, error) {
	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/querylog", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetStatsSummary returns the statistics object from AdGuard Home via
// GET /control/stats (used to verify SEC02 / AT22 data minimization).
func (a *AdGuardAdapter) GetStatsSummary(ctx context.Context) (map[string]any, error) {
	var resp map[string]any
	if err := a.doJSON(ctx, http.MethodGet, "/control/stats", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ClearCache flushes the resolver's in-memory DNS cache via POST /control/cache_clear.
func (a *AdGuardAdapter) ClearCache(ctx context.Context) error {
	return a.doJSON(ctx, http.MethodPost, "/control/cache_clear", nil, nil)
}

// Publish executes the §12.5 publish flow on this resolver:
//  1. Stage and syntax-check the bundle.
//  2. Read current state; if the exact revision and SHA-256 are already applied
//     and settings match, return an idempotent no-op.
//  3. Record the current live state as the rollback point (if not already set).
//  4. Apply privacy settings, upstream/access settings, and user_rules; clear cache.
//  5. Execute smoke tests (via check_host API and real DNS queries if DNSAddr is set)
//     and verify read-back of the applied revision.
//  6. On any failure, restore the last good bundle and report the failure reason.
func (a *AdGuardAdapter) Publish(ctx context.Context, bundle *CompiledBundle) (*ResolverPublishResult, error) {
	targetRev := ""
	if bundle != nil {
		targetRev = bundle.Revision
	}

	// Step 1: Stage and syntax check before touching the live resolver.
	if err := a.StageAndValidate(bundle); err != nil {
		prevRev, prevHash := a.currentOrLastGoodMeta(ctx)
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			RolledBack:       prevRev != "",
			TargetRevision:   targetRev,
			AppliedRevision:  prevRev,
			AppliedSHA256:    prevHash,
			PreviousRevision: prevRev,
			Reason:           fmt.Sprintf("stage/syntax validation failed: %v", err),
			Err:              err,
		}
		return res, err
	}

	// Step 2: Inspect current live state for idempotency and rollback snapshot.
	currentState, err := a.GetAppliedState(ctx)
	if err != nil {
		prevRev, prevHash := a.lastGoodMeta()
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			TargetRevision:   targetRev,
			AppliedRevision:  prevRev,
			AppliedSHA256:    prevHash,
			PreviousRevision: prevRev,
			Reason:           fmt.Sprintf("failed to read resolver state: %v", err),
			Err:              err,
		}
		return res, err
	}

	a.mu.Lock()
	if a.lastGood == nil && currentState.AppliedRevision != "" {
		rulesCopy := make([]string, len(currentState.UserRules))
		copy(rulesCopy, currentState.UserRules)
		a.lastGood = &resolverSnapshot{
			revision:  currentState.AppliedRevision,
			sha256:    currentState.AppliedSHA256,
			userRules: rulesCopy,
			settings: AdGuardSettings{
				QueryLogEnabled:     false,
				QueryLogFileEnabled: false,
				StatisticsEnabled:   false,
				BlockingMode:        currentState.BlockingMode,
				BlockedResponseTTL:  currentState.BlockedTTL,
				UpstreamDNS:         currentState.UpstreamDNS,
				BootstrapDNS:        currentState.BootstrapDNS,
				FallbackDNS:         []string{},
				AllowedClients:      currentState.AllowedClients,
				FilteringEnabled:    true,
				ProtectionEnabled:   true,
			},
		}
	}
	prevSnap := a.lastGood
	faultFn := a.faultHook
	a.mu.Unlock()

	prevRev := currentState.AppliedRevision
	if prevRev == "" && prevSnap != nil {
		prevRev = prevSnap.revision
	}

	// Idempotency check: if the exact same revision and bundle SHA-256 are
	// already applied and privacy settings are intact, re-applying is a no-op.
	if currentState.AppliedRevision == bundle.Revision &&
		strings.EqualFold(currentState.AppliedSHA256, bundle.SHA256) &&
		!currentState.QueryLogEnabled &&
		!currentState.StatisticsEnabled &&
		currentState.FilteringEnabled &&
		currentState.ProtectionEnabled &&
		currentState.ExternalFilters == 0 &&
		currentState.BlockingMode == bundle.Settings.BlockingMode {
		a.mu.Lock()
		rulesSaved := bundle.AdGuardUserRulesWithHash()
		a.lastGood = &resolverSnapshot{
			revision:  bundle.Revision,
			sha256:    bundle.SHA256,
			userRules: rulesSaved,
			settings:  bundle.Settings,
		}
		a.mu.Unlock()
		return &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          true,
			NoOp:             true,
			TargetRevision:   bundle.Revision,
			AppliedRevision:  currentState.AppliedRevision,
			AppliedSHA256:    currentState.AppliedSHA256,
			PreviousRevision: prevRev,
			Reason:           "already at target revision and sha256 (noop)",
		}, nil
	}

	// Step 3: Apply settings and rules to the resolver.
	rulesToApply := bundle.AdGuardUserRulesWithHash()
	if applyErr := a.applySnapshot(ctx, bundle.Settings, rulesToApply); applyErr != nil {
		rbOk, rbRev, rbHash := a.rollbackToLastGood(ctx, prevSnap)
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			RolledBack:       rbOk,
			TargetRevision:   bundle.Revision,
			AppliedRevision:  rbRev,
			AppliedSHA256:    rbHash,
			PreviousRevision: prevRev,
			Reason:           fmt.Sprintf("apply failed: %v", applyErr),
			Err:              applyErr,
		}
		return res, applyErr
	}

	// Step 4: Run smoke tests and read-back verification.
	var smokeErr error
	if faultFn != nil {
		smokeErr = faultFn(ctx, a.cfg.ID, bundle)
	}
	if smokeErr == nil {
		smokeErr = a.runSmokeTests(ctx, bundle)
	}
	if smokeErr != nil {
		rbOk, rbRev, rbHash := a.rollbackToLastGood(ctx, prevSnap)
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			RolledBack:       rbOk,
			TargetRevision:   bundle.Revision,
			AppliedRevision:  rbRev,
			AppliedSHA256:    rbHash,
			PreviousRevision: prevRev,
			Reason:           fmt.Sprintf("smoke test failed (restored last_good): %v", smokeErr),
			Err:              smokeErr,
		}
		return res, smokeErr
	}

	// Step 5: Verify read-back state from resolver.
	afterState, err := a.GetAppliedState(ctx)
	if err != nil {
		rbOk, rbRev, rbHash := a.rollbackToLastGood(ctx, prevSnap)
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			RolledBack:       rbOk,
			TargetRevision:   bundle.Revision,
			AppliedRevision:  rbRev,
			AppliedSHA256:    rbHash,
			PreviousRevision: prevRev,
			Reason:           fmt.Sprintf("read-back verification failed: %v", err),
			Err:              err,
		}
		return res, err
	}
	if afterState.AppliedRevision != bundle.Revision || !strings.EqualFold(afterState.AppliedSHA256, bundle.SHA256) {
		verifyErr := fmt.Errorf("read-back mismatch: got revision=%q sha256=%q, want revision=%q sha256=%q", afterState.AppliedRevision, afterState.AppliedSHA256, bundle.Revision, bundle.SHA256)
		rbOk, rbRev, rbHash := a.rollbackToLastGood(ctx, prevSnap)
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			RolledBack:       rbOk,
			TargetRevision:   bundle.Revision,
			AppliedRevision:  rbRev,
			AppliedSHA256:    rbHash,
			PreviousRevision: prevRev,
			Reason:           verifyErr.Error(),
			Err:              verifyErr,
		}
		return res, verifyErr
	}
	if afterState.QueryLogEnabled || afterState.StatisticsEnabled {
		privErr := errors.New("read-back privacy check failed: querylog or statistics is still enabled")
		rbOk, rbRev, rbHash := a.rollbackToLastGood(ctx, prevSnap)
		res := &ResolverPublishResult{
			ResolverID:       a.cfg.ID,
			Success:          false,
			RolledBack:       rbOk,
			TargetRevision:   bundle.Revision,
			AppliedRevision:  rbRev,
			AppliedSHA256:    rbHash,
			PreviousRevision: prevRev,
			Reason:           privErr.Error(),
			Err:              privErr,
		}
		return res, privErr
	}

	// Update last_good snapshot only after smoke tests and read-back succeed.
	a.mu.Lock()
	rulesSaved := make([]string, len(rulesToApply))
	copy(rulesSaved, rulesToApply)
	a.lastGood = &resolverSnapshot{
		revision:  bundle.Revision,
		sha256:    bundle.SHA256,
		userRules: rulesSaved,
		settings:  bundle.Settings,
	}
	a.mu.Unlock()

	// Clear DNS cache one more time after smoke tests so smoke test queries do
	// not linger in cache.
	_ = a.ClearCache(ctx)

	return &ResolverPublishResult{
		ResolverID:       a.cfg.ID,
		Success:          true,
		NoOp:             false,
		TargetRevision:   bundle.Revision,
		AppliedRevision:  afterState.AppliedRevision,
		AppliedSHA256:    afterState.AppliedSHA256,
		PreviousRevision: prevRev,
		Reason:           "applied and verified",
	}, nil
}

func (a *AdGuardAdapter) applySnapshot(ctx context.Context, settings AdGuardSettings, rules []string) error {
	// 1. Enforce querylog disabled (§12.7).
	qlPayload := map[string]any{
		"enabled":             false,
		"interval":            86400000,
		"anonymize_client_ip": true,
		"ignored":             []string{},
	}
	if err := a.doJSON(ctx, http.MethodPut, "/control/querylog/config/update", qlPayload, nil); err != nil {
		return fmt.Errorf("disable querylog: %w", err)
	}

	// 2. Enforce statistics disabled (§12.7).
	statsPayload := map[string]any{
		"enabled":  false,
		"interval": 86400000,
		"ignored":  []string{},
	}
	if err := a.doJSON(ctx, http.MethodPut, "/control/stats/config/update", statsPayload, nil); err != nil {
		return fmt.Errorf("disable statistics: %w", err)
	}

	// 3. Disable/remove any built-in external filter lists so only our compiled
	// bundle rules are active (§12.3).
	var filterStatus struct {
		Filters []struct {
			ID      int    `json:"id"`
			URL     string `json:"url"`
			Enabled bool   `json:"enabled"`
		} `json:"filters"`
	}
	if err := a.doJSON(ctx, http.MethodGet, "/control/filtering/status", nil, &filterStatus); err == nil {
		for _, f := range filterStatus.Filters {
			delPayload := map[string]any{
				"url":       f.URL,
				"whitelist": false,
			}
			_ = a.doJSON(ctx, http.MethodPost, "/control/filtering/remove_url", delPayload, nil)
		}
	}

	// 4. Enable filtering engine with auto-update interval 0.
	filterCfg := map[string]any{
		"enabled":  true,
		"interval": 0,
	}
	if err := a.doJSON(ctx, http.MethodPost, "/control/filtering/config", filterCfg, nil); err != nil {
		return fmt.Errorf("configure filtering: %w", err)
	}

	// 5. Configure DNS upstreams, no fallback, NXDOMAIN blocking mode, and TTL (§12.2, §12.5).
	bootstraps := settings.BootstrapDNS
	if len(bootstraps) == 0 {
		bootstraps = settings.UpstreamDNS
	}
	blockingMode := settings.BlockingMode
	if blockingMode == "" {
		blockingMode = BlockingModeNXDOMAIN
	}
	blockedTTL := settings.BlockedResponseTTL
	if blockedTTL == 0 {
		blockedTTL = DefaultBlockedTTLSeconds
	}
	dnsCfg := map[string]any{
		"upstream_dns":              settings.UpstreamDNS,
		"bootstrap_dns":             bootstraps,
		"fallback_dns":              []string{},
		"protection_enabled":        true,
		"ratelimit":                 0,
		"blocking_mode":             blockingMode,
		"blocked_response_ttl":      blockedTTL,
		"edns_cs_enabled":           false,
		"dnssec_enabled":            false,
		"disable_ipv6":              false,
		"use_private_ptr_resolvers": false,
		"local_ptr_upstreams":       []string{},
	}
	if err := a.doJSON(ctx, http.MethodPost, "/control/dns_config", dnsCfg, nil); err != nil {
		return fmt.Errorf("configure dns_config: %w", err)
	}

	// 6. Configure client access restrictions (§12.2, DNS03).
	if len(settings.AllowedClients) > 0 {
		accessCfg := map[string]any{
			"allowed_clients":    settings.AllowedClients,
			"disallowed_clients": []string{},
			"blocked_hosts":      []string{"version.bind", "id.server", "hostname.bind"},
		}
		if err := a.doJSON(ctx, http.MethodPost, "/control/access/set", accessCfg, nil); err != nil {
			return fmt.Errorf("configure access/set: %w", err)
		}
	}

	// 7. Atomically set compiled user_rules (§12.5).
	rulesPayload := map[string]any{
		"rules": rules,
	}
	if err := a.doJSON(ctx, http.MethodPost, "/control/filtering/set_rules", rulesPayload, nil); err != nil {
		return fmt.Errorf("set filtering rules: %w", err)
	}

	// 8. Clear DNS cache so new rules take effect immediately (§12.5).
	if err := a.ClearCache(ctx); err != nil {
		return fmt.Errorf("clear dns cache: %w", err)
	}

	return nil
}

func (a *AdGuardAdapter) runSmokeTests(ctx context.Context, bundle *CompiledBundle) error {
	for _, st := range bundle.SmokeTests {
		chk, err := a.CheckHost(ctx, st.Domain)
		if err != nil {
			return fmt.Errorf("smoke test check_host(%q): %w", st.Domain, err)
		}
		gotAction := ActionAllow
		if chk.IsBlocked() {
			gotAction = ActionBlock
		}
		if gotAction != st.ExpectedAction {
			return fmt.Errorf("smoke test check_host(%q) action=%s (reason=%s, rule=%s), expected action=%s", st.Domain, gotAction, chk.Reason, chk.Rule, st.ExpectedAction)
		}

		// When a live DNS address is configured, also verify real DNS query
		// resolution for blocked expectations (must return NXDOMAIN).
		if a.cfg.DNSAddr != "" && st.ExpectedAction == ActionBlock {
			resp, err := a.dnsCli.Query(ctx, a.cfg.DNSAddr, st.Domain, dnsmessage.TypeA)
			if err != nil {
				return fmt.Errorf("smoke test dns query(%q) to %s: %w", st.Domain, a.cfg.DNSAddr, err)
			}
			if !resp.IsNXDOMAIN() {
				return fmt.Errorf("smoke test dns query(%q) returned RCode %v, expected NXDOMAIN", st.Domain, resp.RCode)
			}
		}
	}
	return nil
}

func (a *AdGuardAdapter) rollbackToLastGood(ctx context.Context, snap *resolverSnapshot) (bool, string, string) {
	if snap == nil {
		return false, "", ""
	}
	if err := a.applySnapshot(ctx, snap.settings, snap.userRules); err != nil {
		return false, "", ""
	}
	if st, err := a.GetAppliedState(ctx); err == nil {
		return true, st.AppliedRevision, st.AppliedSHA256
	}
	return true, snap.revision, snap.sha256
}

func (a *AdGuardAdapter) lastGoodMeta() (string, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastGood == nil {
		return "", ""
	}
	return a.lastGood.revision, a.lastGood.sha256
}

func (a *AdGuardAdapter) currentOrLastGoodMeta(ctx context.Context) (string, string) {
	if st, err := a.GetAppliedState(ctx); err == nil && st.AppliedRevision != "" {
		return st.AppliedRevision, st.AppliedSHA256
	}
	return a.lastGoodMeta()
}

func (a *AdGuardAdapter) doJSON(ctx context.Context, method, path string, reqBody any, respOut any) error {
	base := strings.TrimRight(a.cfg.AdminURL, "/")
	fullURL := base + path

	var bodyReader io.Reader
	hasBody := false
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body for %s %s: %w", method, path, err)
		}
		bodyReader = bytes.NewReader(b)
		hasBody = true
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request %s %s: %w", method, path, err)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.cfg.Username != "" || a.cfg.Password != "" {
		req.SetBasicAuth(a.cfg.Username, a.cfg.Password)
	}

	resp, err := a.httpCli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return fmt.Errorf("read response %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBytes))
		if len(msg) > 256 {
			msg = msg[:256]
		}
		return fmt.Errorf("HTTP %d from %s %s: %s", resp.StatusCode, method, path, msg)
	}

	if respOut != nil && len(bytes.TrimSpace(respBytes)) > 0 {
		if err := json.Unmarshal(respBytes, respOut); err != nil {
			return fmt.Errorf("decode response %s %s: %w", method, path, err)
		}
	}
	return nil
}

// MultiResolverPublisher coordinates bundle publishing across multiple
// AdGuardAdapter instances and reports all_applied, partial, or failed (§12.5).
type MultiResolverPublisher struct {
	adapters []*AdGuardAdapter
}

// NewMultiResolverPublisher creates a MultiResolverPublisher for the given adapters.
func NewMultiResolverPublisher(adapters ...*AdGuardAdapter) *MultiResolverPublisher {
	return &MultiResolverPublisher{
		adapters: adapters,
	}
}

// Adapters returns the configured resolver adapters.
func (p *MultiResolverPublisher) Adapters() []*AdGuardAdapter {
	return p.adapters
}

// Publish publishes bundle to all configured resolvers and aggregates the
// per-resolver results. It never claims full success on partial application (§12.5).
func (p *MultiResolverPublisher) Publish(ctx context.Context, bundle *CompiledBundle) (*MultiPublishReport, error) {
	perResolver := make(map[string]*CompiledBundle, len(p.adapters))
	for _, ad := range p.adapters {
		perResolver[ad.ID()] = bundle
	}
	return p.PublishPerResolver(ctx, bundle, perResolver)
}

// PublishPerResolver publishes the bundle mapped for each resolver ID (falling
// back to defaultBundle) and aggregates the results into all_applied, partial,
// or failed (§12.5).
func (p *MultiResolverPublisher) PublishPerResolver(
	ctx context.Context,
	defaultBundle *CompiledBundle,
	perResolver map[string]*CompiledBundle,
) (*MultiPublishReport, error) {
	if len(p.adapters) == 0 {
		return &MultiPublishReport{
			Status:       PublishFailed,
			AllSucceeded: false,
		}, errors.New("no resolvers configured in publisher")
	}

	targetRev := ""
	targetHash := ""
	if defaultBundle != nil {
		targetRev = defaultBundle.Revision
		targetHash = defaultBundle.SHA256
	}

	results := make([]ResolverPublishResult, 0, len(p.adapters))
	succeededCount := 0
	var errs []error

	for _, ad := range p.adapters {
		b := defaultBundle
		if perResolver != nil {
			if custom, exists := perResolver[ad.ID()]; exists {
				b = custom
			}
		}
		res, err := ad.Publish(ctx, b)
		if res != nil {
			results = append(results, *res)
		}
		if err == nil && res != nil && res.Success && res.AppliedRevision == targetRev {
			succeededCount++
		} else if err != nil {
			errs = append(errs, fmt.Errorf("resolver %s: %w", ad.ID(), err))
		}
	}

	var status MultiResolverStatus
	switch {
	case succeededCount == len(p.adapters):
		status = PublishAllApplied
	case succeededCount > 0:
		status = PublishPartial
	default:
		status = PublishFailed
	}

	report := &MultiPublishReport{
		Status:         status,
		AllSucceeded:   status == PublishAllApplied,
		TargetRevision: targetRev,
		TargetSHA256:   targetHash,
		Results:        results,
	}

	if status != PublishAllApplied {
		return report, fmt.Errorf("multi-resolver publish status=%s (%d/%d succeeded): %w", status, succeededCount, len(p.adapters), errors.Join(errs...))
	}
	return report, nil
}
