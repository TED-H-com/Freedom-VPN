// Package dnspublish compiles an active dns_profile revision into a
// deterministic resolver document for desired-state policy_refs
// (§12.4, §12.5, §12.7). Persistence lives in the store subpackage so node
// code can verify documents without a database driver.
package dnspublish

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

const (
	// SchemaVersion is the resolver document schema (§12.5).
	SchemaVersion = 1

	// Kind identifies the canonical document. It is part of the hashed bytes.
	Kind = "resolver_document"

	// EventDNSPublish is the worker event fanned out from a dns_profile activation.
	// policy.activated itself stays available for desired-state compilation (§14.5).
	EventDNSPublish = "dns.publish"

	// HeaderPolicyGenerationPrefix marks the policy_generation in resolver comments.
	// The generation is not part of the hashed document (§12.5).
	HeaderPolicyGenerationPrefix = "! fvpn-policy-generation: "

	// HeaderWithdrawnPrefix marks a fail-closed resolver with no active policy (§17.1).
	HeaderWithdrawnPrefix = "! fvpn-withdrawn: "

	// WithdrawUpstream is a blackhole inside the resolver network namespace.
	// It is not a public resolver (§8.2, §12.5, §17.1).
	WithdrawUpstream = "127.0.0.1:1"

	// DefaultBlockedTTLSeconds is the recommended short blocked-response TTL (§12.5).
	// The production value is 待決策; 60 is the spec's adjustable start.
	DefaultBlockedTTLSeconds = 60

	// MaxBlockedTTLSeconds matches the dns_profile payload schema.
	MaxBlockedTTLSeconds = 86400

	blockNXDOMAIN    = "nxdomain"
	blockRefused     = "refused"
	blockNullAddress = "null_address"

	layerAdminBlock = "admin_block"
	layerAdminAllow = "admin_allow"
)

// DefaultUpstreams is the worker default when no upstream list is configured.
// The host is documentation-only and is never contacted by the compiler.
var DefaultUpstreams = []string{"https://dns.example.com/dns-query"}

var profileCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Privacy is forced off in every document (§12.7, SEC02).
type Privacy struct {
	QueryLogEnabled     bool `json:"querylog_enabled"`
	QueryLogFileEnabled bool `json:"querylog_file_enabled"`
	StatisticsEnabled   bool `json:"statistics_enabled"`
}

// Rule is one admin domain rule in canonical ASCII form (§12.4).
type Rule struct {
	RuleID            string `json:"rule_id"`
	Domain            string `json:"domain"`
	IncludeSubdomains bool   `json:"include_subdomains"`
	Action            string `json:"action"`
	Layer             string `json:"layer"`
	Reason            string `json:"reason"`
}

// Feed is the identity of an enabled threat feed. The feed body is not stored
// on the policy revision, so the document pins the content hash only (§12.3).
type Feed struct {
	FeedCode      string   `json:"feed_code"`
	ContentSHA256 string   `json:"content_sha256"`
	Format        string   `json:"format"`
	Categories    []string `json:"categories"`
}

// Document is the resolver configuration hashed with RFC 8785 (§12.5).
// policy_generation is stored beside the document, not inside these bytes, so
// two activations of the same rules produce identical bytes.
type Document struct {
	SchemaVersion     int      `json:"schema_version"`
	Kind              string   `json:"kind"`
	ProfileCode       string   `json:"profile_code"`
	BlockResponse     string   `json:"block_response"`
	BlockedTTLSeconds int      `json:"blocked_ttl_seconds"`
	SafeSearch        bool     `json:"safe_search"`
	Privacy           Privacy  `json:"privacy"`
	Upstreams         []string `json:"upstreams"`
	BootstrapDNS      []string `json:"bootstrap_dns"`
	FallbackDNS       []string `json:"fallback_dns"`
	Rules             []Rule   `json:"rules"`
	Feeds             []Feed   `json:"feeds"`
}

// Input is the pure compiler input. The same input always yields the same bytes.
type Input struct {
	ProfileCode       string
	BlockResponse     string
	BlockedTTLSeconds int
	SafeSearch        bool
	Upstreams         []string
	BootstrapDNS      []string
	Rules             []Rule
	Feeds             []Feed
}

// Compiled is a document plus the exact bytes that were hashed.
type Compiled struct {
	Document Document
	Bytes    []byte
	Hash     string
}

// Error is a compile or verify failure. It never carries query logs or secrets.
type Error struct {
	Code    string
	Field   string
	Message string
}

func (e *Error) Error() string {
	if e.Field != "" {
		return e.Code + " " + e.Field + ": " + e.Message
	}
	return e.Code + ": " + e.Message
}

// Compile builds the canonical resolver document (§12.4, §12.5, §12.7).
func Compile(in Input) (Compiled, error) {
	doc, err := normalize(in)
	if err != nil {
		return Compiled{}, err
	}
	raw, err := canonjson.Marshal(doc)
	if err != nil {
		return Compiled{}, fmt.Errorf("canonicalize resolver document: %w", err)
	}
	if len(raw) > canonjson.MaxBytes {
		return Compiled{}, &Error{Code: "document_too_large", Field: "document", Message: "canonical document exceeds 1 MiB"}
	}
	sum := sha256.Sum256(raw)
	return Compiled{Document: doc, Bytes: raw, Hash: hex.EncodeToString(sum[:])}, nil
}

// Hash returns the lowercase SHA-256 of the RFC 8785 bytes of doc.
func Hash(doc Document) (string, []byte, error) {
	raw, err := canonjson.Marshal(doc)
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), raw, nil
}

// Verify checks structural and privacy invariants of a document (§12.7).
func Verify(doc Document) error {
	if doc.SchemaVersion != SchemaVersion || doc.Kind != Kind {
		return &Error{Code: "document_invalid", Field: "schema_version", Message: "unsupported resolver document"}
	}
	if doc.Privacy != (Privacy{}) {
		return &Error{Code: "privacy_invalid", Field: "privacy", Message: "query log and statistics must be disabled"}
	}
	if len(doc.FallbackDNS) != 0 {
		return &Error{Code: "fallback_dns", Field: "fallback_dns", Message: "fallback DNS is not permitted"}
	}
	if len(doc.Upstreams) == 0 {
		return &Error{Code: "no_upstreams", Field: "upstreams", Message: "at least one upstream is required"}
	}
	if !validBlockResponse(doc.BlockResponse) {
		return &Error{Code: "block_response", Field: "block_response", Message: "unsupported block response"}
	}
	return nil
}

func normalize(in Input) (Document, error) {
	code := strings.TrimSpace(in.ProfileCode)
	if !profileCodePattern.MatchString(code) {
		return Document{}, &Error{Code: "profile_code", Field: "profile_code", Message: "profile code is empty or not canonical"}
	}
	block := strings.ToLower(strings.TrimSpace(in.BlockResponse))
	if block == "" {
		block = blockNXDOMAIN
	}
	if !validBlockResponse(block) {
		return Document{}, &Error{Code: "block_response", Field: "block_response", Message: "unsupported block response"}
	}
	ttl := in.BlockedTTLSeconds
	if ttl == 0 {
		ttl = DefaultBlockedTTLSeconds
	}
	if ttl < 0 || ttl > MaxBlockedTTLSeconds {
		return Document{}, &Error{Code: "blocked_ttl", Field: "blocked_ttl_seconds", Message: "blocked TTL is outside 0..86400"}
	}
	upstreams, err := cleanServers("upstreams", in.Upstreams, true)
	if err != nil {
		return Document{}, err
	}
	bootstrap, err := cleanServers("bootstrap_dns", in.BootstrapDNS, false)
	if err != nil {
		return Document{}, err
	}
	if len(bootstrap) == 0 {
		bootstrap = append([]string(nil), upstreams...)
	}
	rules, err := normalizeRules(in.Rules)
	if err != nil {
		return Document{}, err
	}
	feeds, err := normalizeFeeds(in.Feeds)
	if err != nil {
		return Document{}, err
	}
	if len(rules)+len(feeds) > dnspolicy.DefaultMaxTotalRulesPerBundle {
		return Document{}, &Error{Code: "rule_limit", Field: "rules", Message: "rule count exceeds the bundle limit"}
	}
	return Document{
		SchemaVersion:     SchemaVersion,
		Kind:              Kind,
		ProfileCode:       code,
		BlockResponse:     block,
		BlockedTTLSeconds: ttl,
		SafeSearch:        in.SafeSearch,
		Privacy:           Privacy{},
		Upstreams:         upstreams,
		BootstrapDNS:      bootstrap,
		FallbackDNS:       []string{},
		Rules:             rules,
		Feeds:             feeds,
	}, nil
}

func validBlockResponse(v string) bool {
	switch v {
	case blockNXDOMAIN, blockRefused, blockNullAddress:
		return true
	default:
		return false
	}
}

func cleanServers(field string, in []string, required bool) ([]string, error) {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		s := strings.TrimSpace(raw)
		if s == "" {
			return nil, &Error{Code: "empty_server", Field: field, Message: "server address is empty"}
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if required && len(out) == 0 {
		return nil, &Error{Code: "no_upstreams", Field: field, Message: "at least one upstream is required"}
	}
	return out, nil
}

func normalizeRules(in []Rule) ([]Rule, error) {
	if len(in) > dnspolicy.DefaultMaxAdminRulesPerProfile {
		return nil, &Error{Code: "rule_limit", Field: "rules", Message: "too many admin rules"}
	}
	out := make([]Rule, 0, len(in))
	seenID := make(map[string]struct{}, len(in))
	type layerKey struct {
		layer  string
		domain string
		sub    bool
	}
	seenLayer := make(map[layerKey]string, len(in))
	for i, rule := range in {
		id, err := uuid.Parse(strings.TrimSpace(rule.RuleID))
		if err != nil {
			return nil, &Error{Code: "rule_id", Field: fmt.Sprintf("rules[%d].rule_id", i), Message: "rule_id must be a UUID"}
		}
		idStr := id.String()
		if _, dup := seenID[idStr]; dup {
			return nil, &Error{Code: "duplicate_rule_id", Field: "rule_id", Message: "duplicate rule_id"}
		}
		seenID[idStr] = struct{}{}
		norm, err := dnspolicy.NormalizeDomain(rule.Domain)
		if err != nil {
			return nil, &Error{Code: "domain", Field: fmt.Sprintf("rules[%d].domain", i), Message: "domain is not a valid DNS name"}
		}
		action := strings.TrimSpace(rule.Action)
		layer := strings.TrimSpace(rule.Layer)
		switch action {
		case string(dnspolicy.ActionBlock):
			if layer == "" {
				layer = layerAdminBlock
			}
			if layer != layerAdminBlock {
				return nil, &Error{Code: "layer", Field: fmt.Sprintf("rules[%d].layer", i), Message: "block rules belong to admin_block"}
			}
		case string(dnspolicy.ActionAllow):
			if layer == "" {
				layer = layerAdminAllow
			}
			if layer != layerAdminAllow {
				return nil, &Error{Code: "layer", Field: fmt.Sprintf("rules[%d].layer", i), Message: "allow rules belong to admin_allow"}
			}
		default:
			return nil, &Error{Code: "action", Field: fmt.Sprintf("rules[%d].action", i), Message: "action must be block or allow"}
		}
		reason := strings.TrimSpace(rule.Reason)
		if reason == "" || len(reason) > 500 {
			return nil, &Error{Code: "reason", Field: fmt.Sprintf("rules[%d].reason", i), Message: "reason must be 1..500 characters"}
		}
		key := layerKey{layer: layer, domain: norm.Canonical, sub: rule.IncludeSubdomains}
		if _, dup := seenLayer[key]; dup {
			return nil, &Error{Code: "rule_conflict", Field: "domain", Message: "same-layer rules conflict on one domain"}
		}
		seenLayer[key] = idStr
		out = append(out, Rule{
			RuleID:            idStr,
			Domain:            norm.Canonical,
			IncludeSubdomains: rule.IncludeSubdomains,
			Action:            action,
			Layer:             layer,
			Reason:            reason,
		})
	}
	slices.SortFunc(out, func(a, b Rule) int {
		if c := strings.Compare(layerRank(a.Layer), layerRank(b.Layer)); c != 0 {
			return c
		}
		if c := strings.Compare(a.Domain, b.Domain); c != 0 {
			return c
		}
		if a.IncludeSubdomains != b.IncludeSubdomains {
			if a.IncludeSubdomains {
				return 1
			}
			return -1
		}
		return strings.Compare(a.RuleID, b.RuleID)
	})
	return out, nil
}

func layerRank(layer string) string {
	switch layer {
	case layerAdminBlock:
		return "1"
	case layerAdminAllow:
		return "2"
	default:
		return "9"
	}
}

func normalizeFeeds(in []Feed) ([]Feed, error) {
	if len(in) > dnspolicy.DefaultMaxSourcesPerProfile {
		return nil, &Error{Code: "feed_limit", Field: "feeds", Message: "too many feeds"}
	}
	out := make([]Feed, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for i, feed := range in {
		code := strings.TrimSpace(feed.FeedCode)
		if !profileCodePattern.MatchString(code) {
			return nil, &Error{Code: "feed_code", Field: fmt.Sprintf("feeds[%d].feed_code", i), Message: "feed code is empty or not canonical"}
		}
		if _, dup := seen[code]; dup {
			return nil, &Error{Code: "duplicate_feed", Field: "feed_code", Message: "duplicate feed code"}
		}
		seen[code] = struct{}{}
		sum := strings.ToLower(strings.TrimSpace(feed.ContentSHA256))
		if len(sum) != 64 || !isHex(sum) {
			return nil, &Error{Code: "content_sha256", Field: fmt.Sprintf("feeds[%d].content_sha256", i), Message: "content hash must be 64 hex characters"}
		}
		format := strings.TrimSpace(feed.Format)
		switch format {
		case "domain_list", "hosts", "adblock":
		default:
			return nil, &Error{Code: "feed_format", Field: fmt.Sprintf("feeds[%d].format", i), Message: "unsupported feed format"}
		}
		if len(feed.Categories) == 0 {
			return nil, &Error{Code: "categories", Field: fmt.Sprintf("feeds[%d].categories", i), Message: "a feed needs a category"}
		}
		cats := append([]string(nil), feed.Categories...)
		for j, c := range cats {
			cats[j] = strings.TrimSpace(c)
			switch cats[j] {
			case "malware", "phishing", "ads", "family":
			default:
				return nil, &Error{Code: "category", Field: fmt.Sprintf("feeds[%d].categories", i), Message: "unsupported feed category"}
			}
		}
		slices.Sort(cats)
		out = append(out, Feed{FeedCode: code, ContentSHA256: sum, Format: format, Categories: cats})
	}
	slices.SortFunc(out, func(a, b Feed) int { return strings.Compare(a.FeedCode, b.FeedCode) })
	return out, nil
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

// RenderRules renders AdGuard Home user rules for doc, with the document hash
// and policy generation in comments so the node can read them back (§12.5).
func RenderRules(doc Document, generation, hash string) ([]string, error) {
	if err := Verify(doc); err != nil {
		return nil, err
	}
	if generation == "" || hash == "" {
		return nil, errors.New("generation and document hash are required")
	}
	lines := []string{
		"! fvpn-bundle-version: 1",
		dnspolicy.HeaderProfileIDPrefix + doc.ProfileCode,
		HeaderPolicyGenerationPrefix + generation,
		dnspolicy.HeaderBundleHashPrefix + hash,
	}
	var blocks, allows []string
	for _, rule := range doc.Rules {
		line, err := dnspolicy.RenderAdGuardRule(rule.Domain, rule.IncludeSubdomains, dnspolicy.PolicyLayer(rule.Layer))
		if err != nil {
			return nil, err
		}
		switch rule.Layer {
		case layerAdminBlock:
			blocks = append(blocks, line)
		case layerAdminAllow:
			allows = append(allows, line)
		default:
			return nil, &Error{Code: "layer", Field: "layer", Message: "cannot render rule layer"}
		}
	}
	if len(blocks) > 0 {
		lines = append(lines, "! layer-1: admin_block")
		lines = append(lines, blocks...)
	}
	if len(allows) > 0 {
		lines = append(lines, "! layer-2: admin_allow")
		lines = append(lines, allows...)
	}
	return lines, nil
}
