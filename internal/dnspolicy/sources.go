package dnspolicy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// SourceFormat identifies the syntax of a threat or category domain list (§12.3).
type SourceFormat string

const (
	SourceFormatPlain   SourceFormat = "plain_domains"
	SourceFormatHosts   SourceFormat = "hosts"
	SourceFormatAdBlock SourceFormat = "adblock"
)

// SourceCategory classifies the purpose of a list source (§12.3).
type SourceCategory string

const (
	CategoryMalware  SourceCategory = "malware"
	CategoryPhishing SourceCategory = "phishing"
	CategoryThreat   SourceCategory = "threat"
	CategoryAds      SourceCategory = "ads"
	CategoryFamily   SourceCategory = "family"
)

// LicenseStatus tracks whether a list source is approved for commercial paid-VPN
// use (§12.3). Only LicenseCommercialApproved sources may be enabled in active
// profiles.
type LicenseStatus string

const (
	LicenseCommercialApproved LicenseStatus = "commercial_approved"
	LicensePendingDecision    LicenseStatus = "pending_decision"
	LicenseNonCommercialOnly  LicenseStatus = "non_commercial_only"
	LicenseRejected           LicenseStatus = "rejected"
)

// SmokeTestStatus records the outcome of post-parse source validation (§12.3).
type SmokeTestStatus string

const (
	SmokeTestPassed   SmokeTestStatus = "passed"
	SmokeTestFailed   SmokeTestStatus = "failed"
	SmokeTestUntested SmokeTestStatus = "untested"
)

const (
	// PendingDecisionCadence marks the feed update cadence as a pending business
	// decision (§12.3, §21: 待決策).
	PendingDecisionCadence = "待決策"

	// DefaultMaxSourceBytes is the default maximum byte size for a single list
	// source download (32 MB decimal).
	DefaultMaxSourceBytes int64 = 32_000_000

	// DefaultMaxSourceRules is the default maximum number of parsed rules in a
	// single list source.
	DefaultMaxSourceRules = 500_000

	// DefaultFetchTimeout is the default timeout for external source HTTPS fetches.
	DefaultFetchTimeout = 15 * time.Second
)

// SourceQualityMetrics tracks parse and smoke-test quality metrics for a list
// source (§12.3).
type SourceQualityMetrics struct {
	RuleCount          int             `json:"rule_count"`
	Delta              int             `json:"delta"`
	ParseErrors        int             `json:"parse_errors"`
	SkippedUnsupported int             `json:"skipped_unsupported"`
	LastUpdate         time.Time       `json:"last_update"`
	SmokeTestResult    SmokeTestStatus `json:"smoke_test_result"`
}

// SourceEntry is a single normalized domain entry parsed from a list source.
type SourceEntry struct {
	Domain            string `json:"domain"`
	IncludeSubdomains bool   `json:"include_subdomains"`
}

// SourceSnapshot holds the last known good state of a ListSource so failed
// fetches or parses never fail open (§12.3, §12.5).
type SourceSnapshot struct {
	Version       string               `json:"version"`
	ContentSHA256 string               `json:"content_sha256"`
	Entries       []SourceEntry        `json:"entries"`
	Quality       SourceQualityMetrics `json:"quality"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// ListSource represents a registered domain list feed (§12.3, §14.2 threat_feed).
type ListSource struct {
	SourceID          string               `json:"source_id"`
	Name              string               `json:"name,omitempty"`
	Version           string               `json:"version"`
	ContentSHA256     string               `json:"content_sha256"`
	License           string               `json:"license"`
	LicenseURL        string               `json:"license_url,omitempty"`
	LicenseStatus     LicenseStatus        `json:"license_status"`
	PermittedUse      string               `json:"permitted_use"`
	DistributionNotes string               `json:"distribution_notes"`
	AttributionNotes  *string              `json:"attribution_notes"` // nil means no attribution required
	URL               string               `json:"url"`
	Format            SourceFormat         `json:"format"`
	Category          SourceCategory       `json:"category"`
	UpdateCadence     string               `json:"update_cadence"`
	EnabledByDefault  bool                 `json:"enabled_by_default"`
	Quality           SourceQualityMetrics `json:"quality"`
	Entries           []SourceEntry        `json:"entries,omitempty"`
	LastGood          *SourceSnapshot      `json:"last_good,omitempty"`
}

// Clone returns a deep copy of the ListSource.
func (s ListSource) Clone() ListSource {
	out := s
	if s.AttributionNotes != nil {
		attribution := *s.AttributionNotes
		out.AttributionNotes = &attribution
	}
	if s.Entries != nil {
		out.Entries = make([]SourceEntry, len(s.Entries))
		copy(out.Entries, s.Entries)
	}
	if s.LastGood != nil {
		lg := *s.LastGood
		if s.LastGood.Entries != nil {
			lg.Entries = make([]SourceEntry, len(s.LastGood.Entries))
			copy(lg.Entries, s.LastGood.Entries)
		}
		out.LastGood = &lg
	}
	return out
}

// ValidateMetadata validates the static registration metadata of a ListSource
// for activation in a profile (§12.3, §14.2).
func (s ListSource) ValidateMetadata() error {
	if strings.TrimSpace(s.SourceID) == "" {
		return &ValidationError{
			Code:    ErrCodeSourceNotFound,
			Field:   "source_id",
			Message: "source_id must not be empty",
		}
	}
	if err := ValidateDecimalRevision(s.Version); err != nil {
		return &ValidationError{
			Code:    ErrCodeInvalidRevision,
			Field:   "source.version",
			Input:   s.Version,
			Message: fmt.Sprintf("source %q has invalid version %q: %v", s.SourceID, s.Version, err),
			Cause:   err,
		}
	}
	if err := ValidateSourceURL(s.URL); err != nil {
		return err
	}
	if s.LicenseStatus != LicenseCommercialApproved {
		return &ValidationError{
			Code:    ErrCodeSourceLicenseUnapproved,
			Field:   "source.license_status",
			Input:   string(s.LicenseStatus),
			Message: fmt.Sprintf("source %q license status is %q (must be %q with verified commercial terms before enabling in a profile)", s.SourceID, s.LicenseStatus, LicenseCommercialApproved),
		}
	}
	if strings.TrimSpace(s.License) == "" || strings.TrimSpace(s.PermittedUse) == "" || strings.TrimSpace(s.DistributionNotes) == "" || (s.AttributionNotes != nil && strings.TrimSpace(*s.AttributionNotes) == "") {
		return &ValidationError{
			Code:    ErrCodeSourceLicenseUnapproved,
			Field:   "source.license",
			Input:   s.SourceID,
			Message: fmt.Sprintf("source %q must record non-empty license, permitted_use, distribution_notes, and either null or non-empty attribution_notes", s.SourceID),
		}
	}
	if s.Quality.SmokeTestResult == SmokeTestFailed {
		return &ValidationError{
			Code:    ErrCodeSourceLicenseUnapproved,
			Field:   "source.quality.smoke_test_result",
			Input:   string(s.Quality.SmokeTestResult),
			Message: fmt.Sprintf("source %q failed smoke test and cannot be activated", s.SourceID),
		}
	}
	return nil
}

var blockedSSRFPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // Unspecified / current network
	netip.MustParsePrefix("10.0.0.0/8"),      // RFC1918 private
	netip.MustParsePrefix("100.64.0.0/10"),   // CGNAT / overlay
	netip.MustParsePrefix("127.0.0.0/8"),     // Loopback
	netip.MustParsePrefix("169.254.0.0/16"),  // Link-local / cloud metadata
	netip.MustParsePrefix("172.16.0.0/12"),   // RFC1918 private
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1 documentation
	netip.MustParsePrefix("192.168.0.0/16"),  // RFC1918 private
	netip.MustParsePrefix("198.18.0.0/15"),   // Benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2 documentation
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3 documentation
	netip.MustParsePrefix("224.0.0.0/4"),     // IPv4 multicast
	netip.MustParsePrefix("240.0.0.0/4"),     // Reserved / broadcast
	netip.MustParsePrefix("::/128"),          // IPv6 unspecified
	netip.MustParsePrefix("::1/128"),         // IPv6 loopback
	netip.MustParsePrefix("64:ff9b::/96"),    // NAT64
	netip.MustParsePrefix("2001::/32"),       // Teredo
	netip.MustParsePrefix("2001:db8::/32"),   // IPv6 documentation
	netip.MustParsePrefix("2002::/16"),       // 6to4
	netip.MustParsePrefix("fc00::/7"),        // IPv6 ULA
	netip.MustParsePrefix("fe80::/10"),       // IPv6 link-local
	netip.MustParsePrefix("ff00::/8"),        // IPv6 multicast
}

func isBlockedSSRFAddr(ip netip.Addr) (netip.Prefix, bool) {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return netip.Prefix{}, true
	}
	for _, pfx := range blockedSSRFPrefixes {
		if pfx.Contains(ip) {
			return pfx, true
		}
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return netip.PrefixFrom(ip, ip.BitLen()), true
	}
	return netip.Prefix{}, false
}

// ValidateSourceURL ensures a feed URL is a fixed HTTPS URL and prevents SSRF
// to internal, loopback, link-local, overlay, documentation, or cloud metadata addresses (§12.3, §16.2).
func ValidateSourceURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: "source URL must not be empty",
		}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: fmt.Sprintf("malformed source URL: %v", err),
			Cause:   err,
		}
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: "source URL must use fixed https:// scheme",
		}
	}
	if u.User != nil {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: "source URL must not contain userinfo credentials",
		}
	}
	host := u.Hostname()
	if host == "" {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: "source URL must contain a valid hostname",
		}
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if pfx, blocked := isBlockedSSRFAddr(addr); blocked {
			return &ValidationError{
				Code:    ErrCodeSourceInvalidURL,
				Field:   "url",
				Input:   rawURL,
				Message: fmt.Sprintf("source URL host %q is in forbidden internal/metadata IP range %s", host, pfx),
			}
		}
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: "source URL must use a verifiable HTTPS domain name, not a raw IP literal",
		}
	}
	norm, err := NormalizeDomain(host)
	if err != nil {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: fmt.Sprintf("source URL hostname %q is invalid: %v", host, err),
			Cause:   err,
		}
	}
	if norm.Canonical == "localhost" || strings.HasSuffix(norm.Canonical, ".localhost") || strings.HasSuffix(norm.Canonical, ".local") || strings.HasSuffix(norm.Canonical, ".internal") {
		return &ValidationError{
			Code:    ErrCodeSourceInvalidURL,
			Field:   "url",
			Input:   rawURL,
			Message: fmt.Sprintf("source URL hostname %q is a forbidden local/internal name", norm.Canonical),
		}
	}
	return nil
}

// ParseOptions configures size limits and validation strictness for list parsers.
type ParseOptions struct {
	MaxBytes        int64
	MaxRules        int
	RequireNonEmpty bool
}

// DefaultParseOptions returns the recommended default parser limits.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		MaxBytes:        DefaultMaxSourceBytes,
		MaxRules:        DefaultMaxSourceRules,
		RequireNonEmpty: true,
	}
}

// ParseResult contains the normalized entries, SHA-256 digest, and line-level
// quality metrics from parsing a list source.
type ParseResult struct {
	Entries            []SourceEntry
	ContentSHA256      string
	RuleCount          int
	ParseErrors        int
	SkippedUnsupported int
}

var hostsBoilerplateNames = map[string]struct{}{
	"localhost":             {},
	"localhost.localdomain": {},
	"local":                 {},
	"broadcasthost":         {},
	"ip6-localhost":         {},
	"ip6-loopback":          {},
	"ip6-localnet":          {},
	"ip6-mcastprefix":       {},
	"ip6-allnodes":          {},
	"ip6-allrouters":        {},
	"ip6-allhosts":          {},
	"0.0.0.0":               {},
}

var sinkholeIPs = map[string]struct{}{
	"0.0.0.0":   {},
	"127.0.0.1": {},
	"::":        {},
	"::1":       {},
	"::0":       {},
}

// ParseSourceList reads up to opts.MaxBytes from r and parses it according to
// format (plain_domains, hosts, or adblock ||name^ subset). It enforces per-line
// domain normalization, skips unsupported constructs with counts, and never fails
// open (§12.3).
func ParseSourceList(r io.Reader, format SourceFormat, opts ParseOptions) (ParseResult, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxSourceBytes
	}
	maxRules := opts.MaxRules
	if maxRules <= 0 {
		maxRules = DefaultMaxSourceRules
	}

	limited := io.LimitReader(r, maxBytes+1)
	rawBytes, err := io.ReadAll(limited)
	if err != nil {
		return ParseResult{}, fmt.Errorf("read source list: %w", err)
	}
	if int64(len(rawBytes)) > maxBytes {
		return ParseResult{}, &ValidationError{
			Code:    ErrCodeSourceTooLarge,
			Field:   "source_content",
			Message: fmt.Sprintf("source content exceeds maximum size of %d bytes", maxBytes),
		}
	}

	sum := sha256.Sum256(rawBytes)
	contentHash := hex.EncodeToString(sum[:])

	var res ParseResult
	res.ContentSHA256 = contentHash

	seen := make(map[string]struct{})
	var entries []SourceEntry

	addDomain := func(canonical string) error {
		if _, exists := seen[canonical]; exists {
			return nil
		}
		if len(entries) >= maxRules {
			return &ValidationError{
				Code:    ErrCodeRuleLimitExceeded,
				Field:   "source_rules",
				Message: fmt.Sprintf("source exceeds maximum allowed rule count of %d", maxRules),
			}
		}
		seen[canonical] = struct{}{}
		entries = append(entries, SourceEntry{
			Domain:            canonical,
			IncludeSubdomains: format != SourceFormatHosts,
		})
		return nil
	}

	lines := bytes.Split(rawBytes, []byte{'\n'})
	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimSuffix(string(rawLine), "\r"))
		if line == "" {
			continue
		}

		switch format {
		case SourceFormatPlain:
			if strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.IndexByte(line, '#'); idx >= 0 {
				line = strings.TrimSpace(line[:idx])
			}
			if line == "" {
				continue
			}
			// Plain domain lists must have a single domain token per line.
			if strings.ContainsAny(line, " \t") {
				res.ParseErrors++
				continue
			}
			norm, err := NormalizeDomain(line)
			if err != nil {
				res.ParseErrors++
				continue
			}
			if err := addDomain(norm.Canonical); err != nil {
				return ParseResult{}, err
			}

		case SourceFormatHosts:
			if strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.IndexByte(line, '#'); idx >= 0 {
				line = strings.TrimSpace(line[:idx])
			}
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				res.ParseErrors++
				continue
			}
			ipToken := fields[0]
			if _, ok := sinkholeIPs[ipToken]; !ok {
				// Non-sinkhole IP or invalid first token: skip as unsupported/parse error
				// so external feeds can never inject arbitrary DNS redirection IPs.
				if _, ipErr := netip.ParseAddr(ipToken); ipErr == nil {
					res.SkippedUnsupported++
				} else {
					res.ParseErrors++
				}
				continue
			}
			for _, hostToken := range fields[1:] {
				lowerHost := strings.ToLower(strings.TrimSuffix(hostToken, "."))
				if _, isBoilerplate := hostsBoilerplateNames[lowerHost]; isBoilerplate {
					res.SkippedUnsupported++
					continue
				}
				norm, err := NormalizeDomain(hostToken)
				if err != nil {
					res.ParseErrors++
					continue
				}
				if err := addDomain(norm.Canonical); err != nil {
					return ParseResult{}, err
				}
			}

		case SourceFormatAdBlock:
			// Comments and AdBlock headers
			if strings.HasPrefix(line, "!") || strings.HasPrefix(line, "#") || (strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")) {
				continue
			}
			// External feeds are strictly Layer 3 blocklists and may only use the
			// domain-only subset ||name^ (§12.3). Exception rules (@@), modifiers ($),
			// cosmetic filters (##, #@#, #?#), paths (/), or wildcards (*) are skipped.
			if strings.HasPrefix(line, "@@") ||
				strings.Contains(line, "##") ||
				strings.Contains(line, "#@#") ||
				strings.Contains(line, "#?#") {
				res.SkippedUnsupported++
				continue
			}
			if !strings.HasPrefix(line, "||") || !strings.HasSuffix(line, "^") {
				if strings.Contains(line, "$") || strings.HasPrefix(line, "/") || strings.HasPrefix(line, "|") {
					res.SkippedUnsupported++
				} else {
					res.ParseErrors++
				}
				continue
			}
			body := line[2 : len(line)-1]
			if body == "" {
				res.ParseErrors++
				continue
			}
			if strings.ContainsAny(body, "$/*^|") {
				res.SkippedUnsupported++
				continue
			}
			norm, err := NormalizeDomain(body)
			if err != nil {
				res.ParseErrors++
				continue
			}
			if err := addDomain(norm.Canonical); err != nil {
				return ParseResult{}, err
			}

		default:
			return ParseResult{}, &ValidationError{
				Code:    ErrCodeSourceFormatUnsupported,
				Field:   "format",
				Input:   string(format),
				Message: fmt.Sprintf("unsupported source list format %q", format),
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Domain < entries[j].Domain
	})
	res.Entries = entries
	res.RuleCount = len(entries)

	if opts.RequireNonEmpty && res.RuleCount == 0 {
		return res, &ValidationError{
			Code:    ErrCodeSourceEmpty,
			Field:   "source_content",
			Message: "parsed source list produced 0 valid domain rules (refusing to fail open)",
		}
	}

	return res, nil
}

// ParsePlainDomains parses a plain one-domain-per-line list.
func ParsePlainDomains(r io.Reader, opts ParseOptions) (ParseResult, error) {
	return ParseSourceList(r, SourceFormatPlain, opts)
}

// ParseHostsFile parses a standard hosts-file format blocklist.
func ParseHostsFile(r io.Reader, opts ParseOptions) (ParseResult, error) {
	return ParseSourceList(r, SourceFormatHosts, opts)
}

// ParseAdBlockDomainSubset parses the domain-only subset of AdBlock syntax (||name^).
func ParseAdBlockDomainSubset(r io.Reader, opts ParseOptions) (ParseResult, error) {
	return ParseSourceList(r, SourceFormatAdBlock, opts)
}

// LookupNetIPFunc resolves hostnames to IP addresses during SSRF-safe dialing.
type LookupNetIPFunc func(ctx context.Context, network, host string) ([]netip.Addr, error)

// DialContextFunc establishes a network connection to a verified IP:port address.
type DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

func makeSSRFDialContext(timeout time.Duration, lookupIP LookupNetIPFunc, dial DialContextFunc) DialContextFunc {
	if timeout <= 0 {
		timeout = DefaultFetchTimeout
	}
	if lookupIP == nil {
		lookupIP = net.DefaultResolver.LookupNetIP
	}
	if dial == nil {
		dial = (&net.Dialer{Timeout: timeout}).DialContext
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := lookupIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, &ValidationError{
				Code:    ErrCodeSourceInvalidURL,
				Field:   "url",
				Input:   host,
				Message: "source DNS lookup failed",
				Cause:   err,
			}
		}
		for _, ip := range ips {
			if pfx, blocked := isBlockedSSRFAddr(ip); blocked {
				return nil, &ValidationError{
					Code:    ErrCodeSourceInvalidURL,
					Field:   "url",
					Input:   host,
					Message: fmt.Sprintf("source DNS resolved %q to blocked nonpublic address %s (prefix %s)", host, ip.Unmap(), pfx),
				}
			}
		}
		return dial(ctx, network, net.JoinHostPort(ips[0].Unmap().String(), port))
	}
}

func refuseSourceRedirect(req *http.Request, _ []*http.Request) error {
	target := ""
	if req != nil && req.URL != nil {
		target = req.URL.String()
	}
	return &ValidationError{
		Code:    ErrCodeSourceInvalidURL,
		Field:   "url",
		Input:   target,
		Message: "source redirects are forbidden",
	}
}

// NewSourceHTTPClient constructs a dedicated SSRF-safe HTTP client for fetching
// external threat feeds (§12.3, §16.2), mirroring internal/releases.NewHTTPSFetcher:
//   - never uses host environment proxy settings;
//   - CheckRedirect refuses every HTTP redirect;
//   - Transport.DialContext resolves via net.DefaultResolver.LookupNetIP (or lookupIP seam),
//     rejects any address in blockedSSRFPrefixes (loopback, RFC1918, CGNAT 100.64/10,
//     link-local, multicast, unspecified, ULA, documentation), and dials the verified IP.
func NewSourceHTTPClient(timeout time.Duration, lookupIP LookupNetIPFunc, dial DialContextFunc) *http.Client {
	if timeout <= 0 {
		timeout = DefaultFetchTimeout
	}
	tr := &http.Transport{
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		DisableKeepAlives:     true,
		DialContext:           makeSSRFDialContext(timeout, lookupIP, dial),
	}
	return &http.Client{
		Transport:     tr,
		Timeout:       timeout,
		CheckRedirect: refuseSourceRedirect,
	}
}

// SourceRegistry manages registered threat and category feeds and guarantees
// last-good retention when fetches or parses fail (§12.3, §12.5).
type SourceRegistry struct {
	mu       sync.RWMutex
	sources  map[string]ListSource
	client   *http.Client
	lookupIP LookupNetIPFunc
	dial     DialContextFunc
}

// SourceStore is an alias for SourceRegistry.
type SourceStore = SourceRegistry

// NewDefaultSourceRegistry returns a new SourceRegistry with a dedicated SSRF-safe
// HTTP client and no real external feed enabled by default because commercial feed
// licensing is 待決策 (§12.3).
func NewDefaultSourceRegistry() *SourceRegistry {
	return &SourceRegistry{
		sources: make(map[string]ListSource),
		client:  NewSourceHTTPClient(DefaultFetchTimeout, nil, nil),
	}
}

// NewDefaultSourceStore is an alias for NewDefaultSourceRegistry.
func NewDefaultSourceStore() *SourceStore {
	return NewDefaultSourceRegistry()
}

// SetLookupNetIP overrides the DNS resolver function used by the registry's
// dedicated HTTP client (for deterministic unit testing without real DNS).
func (sr *SourceRegistry) SetLookupNetIP(fn LookupNetIPFunc) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.lookupIP = fn
	sr.client = NewSourceHTTPClient(DefaultFetchTimeout, sr.lookupIP, sr.dial)
}

// SetDialContext overrides the underlying network dialer used after IP verification
// by the registry's dedicated HTTP client (for deterministic unit testing).
func (sr *SourceRegistry) SetDialContext(fn DialContextFunc) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.dial = fn
	sr.client = NewSourceHTTPClient(DefaultFetchTimeout, sr.lookupIP, sr.dial)
}

func (sr *SourceRegistry) httpClient(override *http.Client) *http.Client {
	sr.mu.RLock()
	lookupIP := sr.lookupIP
	dial := sr.dial
	dedicated := sr.client
	sr.mu.RUnlock()

	if override == nil {
		if dedicated != nil {
			return dedicated
		}
		return NewSourceHTTPClient(DefaultFetchTimeout, lookupIP, dial)
	}

	timeout := override.Timeout
	if timeout <= 0 {
		timeout = DefaultFetchTimeout
	}
	var tr *http.Transport
	if baseTr, ok := override.Transport.(*http.Transport); ok && baseTr != nil {
		tr = baseTr.Clone()
	} else {
		tr = &http.Transport{
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
		}
	}
	tr.Proxy = nil
	tr.DisableKeepAlives = true
	tr.DialContext = makeSSRFDialContext(timeout, lookupIP, dial)

	c := *override
	c.Timeout = timeout
	c.Transport = tr
	c.CheckRedirect = refuseSourceRedirect
	return &c
}

// Register adds or updates a ListSource definition in the registry.
// If the source already has valid entries and SmokeTestPassed, its initial
// state is also recorded as LastGood.
func (sr *SourceRegistry) Register(src ListSource) error {
	if strings.TrimSpace(src.SourceID) == "" {
		return &ValidationError{
			Code:    ErrCodeSourceNotFound,
			Field:   "source_id",
			Message: "source_id must not be empty",
		}
	}
	if err := ValidateSourceURL(src.URL); err != nil {
		return err
	}
	if src.UpdateCadence == "" {
		src.UpdateCadence = PendingDecisionCadence
	}
	cloned := src.Clone()
	if len(cloned.Entries) > 0 && cloned.LastGood == nil && cloned.Quality.SmokeTestResult != SmokeTestFailed {
		if cloned.Quality.RuleCount == 0 {
			cloned.Quality.RuleCount = len(cloned.Entries)
		}
		if cloned.Quality.SmokeTestResult == "" {
			cloned.Quality.SmokeTestResult = SmokeTestPassed
		}
		entriesCopy := make([]SourceEntry, len(cloned.Entries))
		copy(entriesCopy, cloned.Entries)
		cloned.LastGood = &SourceSnapshot{
			Version:       cloned.Version,
			ContentSHA256: cloned.ContentSHA256,
			Entries:       entriesCopy,
			Quality:       cloned.Quality,
			UpdatedAt:     cloned.Quality.LastUpdate,
		}
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.sources[cloned.SourceID] = cloned
	return nil
}

// Get returns a copy of a registered ListSource by ID.
func (sr *SourceRegistry) Get(sourceID string) (ListSource, bool) {
	if sr == nil {
		return ListSource{}, false
	}
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	s, ok := sr.sources[sourceID]
	if !ok {
		return ListSource{}, false
	}
	return s.Clone(), true
}

// List returns copies of all registered sources sorted by SourceID.
func (sr *SourceRegistry) List() []ListSource {
	if sr == nil {
		return nil
	}
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	out := make([]ListSource, 0, len(sr.sources))
	for _, s := range sr.sources {
		out = append(out, s.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SourceID < out[j].SourceID
	})
	return out
}

// UpdateFromReader parses new content for sourceID, runs smokeCheck (if non-nil),
// and atomically updates the source. If reading, parsing, or smokeCheck fails,
// the previous LastGood entries, version, and hash remain active (§12.3, §12.5).
func (sr *SourceRegistry) UpdateFromReader(
	sourceID string,
	newVersion string,
	r io.Reader,
	opts ParseOptions,
	now time.Time,
	smokeCheck func(entries []SourceEntry) error,
) error {
	if err := ValidateDecimalRevision(newVersion); err != nil {
		return err
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	src, ok := sr.sources[sourceID]
	if !ok {
		return &ValidationError{
			Code:    ErrCodeSourceNotFound,
			Field:   "source_id",
			Input:   sourceID,
			Message: fmt.Sprintf("source %q is not registered", sourceID),
		}
	}

	parsed, err := ParseSourceList(r, src.Format, opts)
	if err != nil {
		// Retain last good state on parse failure.
		if src.LastGood != nil {
			src.Version = src.LastGood.Version
			src.ContentSHA256 = src.LastGood.ContentSHA256
			src.Entries = make([]SourceEntry, len(src.LastGood.Entries))
			copy(src.Entries, src.LastGood.Entries)
		}
		sr.sources[sourceID] = src
		return err
	}

	if smokeCheck != nil {
		if err := smokeCheck(parsed.Entries); err != nil {
			if src.LastGood != nil {
				src.Version = src.LastGood.Version
				src.ContentSHA256 = src.LastGood.ContentSHA256
				src.Entries = make([]SourceEntry, len(src.LastGood.Entries))
				copy(src.Entries, src.LastGood.Entries)
			}
			sr.sources[sourceID] = src
			return fmt.Errorf("source %q failed smoke test: %w", sourceID, err)
		}
	}

	prevCount := 0
	if src.LastGood != nil {
		prevCount = src.LastGood.Quality.RuleCount
	} else {
		prevCount = len(src.Entries)
	}

	quality := SourceQualityMetrics{
		RuleCount:          parsed.RuleCount,
		Delta:              parsed.RuleCount - prevCount,
		ParseErrors:        parsed.ParseErrors,
		SkippedUnsupported: parsed.SkippedUnsupported,
		LastUpdate:         now.UTC(),
		SmokeTestResult:    SmokeTestPassed,
	}

	src.Version = newVersion
	src.ContentSHA256 = parsed.ContentSHA256
	src.Entries = parsed.Entries
	src.Quality = quality

	entriesCopy := make([]SourceEntry, len(parsed.Entries))
	copy(entriesCopy, parsed.Entries)
	src.LastGood = &SourceSnapshot{
		Version:       newVersion,
		ContentSHA256: parsed.ContentSHA256,
		Entries:       entriesCopy,
		Quality:       quality,
		UpdatedAt:     now.UTC(),
	}

	sr.sources[sourceID] = src
	return nil
}

// FetchAndUpdate fetches a source's HTTPS URL and updates its entries. If the
// HTTP fetch, status check, parse, or smoke check fails, the source retains its
// last good version (§12.3, §12.5).
func (sr *SourceRegistry) FetchAndUpdate(
	ctx context.Context,
	client *http.Client,
	sourceID string,
	newVersion string,
	opts ParseOptions,
	now time.Time,
	smokeCheck func(entries []SourceEntry) error,
) error {
	src, ok := sr.Get(sourceID)
	if !ok {
		return &ValidationError{
			Code:    ErrCodeSourceNotFound,
			Field:   "source_id",
			Input:   sourceID,
			Message: fmt.Sprintf("source %q is not registered", sourceID),
		}
	}
	if err := ValidateSourceURL(src.URL); err != nil {
		return err
	}
	httpCli := sr.httpClient(client)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return fmt.Errorf("create request for source %q: %w", sourceID, err)
	}
	resp, err := httpCli.Do(req)
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			return ve
		}
		return fmt.Errorf("fetch source %q: %w", sourceID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch source %q returned HTTP %d", sourceID, resp.StatusCode)
	}

	return sr.UpdateFromReader(sourceID, newVersion, resp.Body, opts, now, smokeCheck)
}
