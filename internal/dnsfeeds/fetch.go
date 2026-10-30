// Package dnsfeeds implements the active threat-feed content projection (D-054,
// §12.3). Only normalized policy entries cross the persistence boundary.
package dnsfeeds

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

const (
	EventRefresh             = "dns.feed.refresh"
	DefaultUpdateCadence     = 6 * time.Hour // D-054; 待實測.
	DefaultMaxRemovalPercent = 50
	// Keep embedded feeds bounded within the existing 1 MiB resolver document.
	MaxEmbeddedFeedBytes = 512 * 1024
)

// Content is an accepted immutable version. SHA256 hashes the original bytes;
// Entries are normalized and covered by the enclosing resolver document hash.
type Content struct {
	SHA256  string
	Size    int64
	Entries []dnspolicy.SourceEntry
	Metrics dnspolicy.SourceQualityMetrics
	Kind    string
}

// Fetch verifies the bootstrap pin, then bounded set removal on later refreshes.
// client is a trusted dependency injection seam. Production uses
// dnspolicy.NewSourceHTTPClient; URLs are always validated, redirects refused.
// Failures return only a stable reason code, never the upstream error/body/URL.
func Fetch(ctx context.Context, client *http.Client, p api.ThreatFeedPayload, previous *Content, now time.Time, maxRemoval int) (*Content, string) {
	if maxRemoval < 0 || maxRemoval > DefaultMaxRemovalPercent {
		return nil, "invalid_config"
	}
	if now.Before(p.ValidFrom) || !now.Before(p.ValidUntil) {
		return nil, "outside_validity"
	}
	if dnspolicy.ValidateSourceURL(p.SourceUrl) != nil {
		return nil, "ssrf_refused"
	}
	if client == nil {
		client = dnspolicy.NewSourceHTTPClient(dnspolicy.DefaultFetchTimeout, nil, nil)
	}
	c := *client
	c.Timeout = dnspolicy.DefaultFetchTimeout
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("redirect_refused") }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.SourceUrl, nil)
	if err != nil {
		return nil, "ssrf_refused"
	}
	resp, err := c.Do(req)
	if err != nil {
		var ve *dnspolicy.ValidationError
		if errors.As(err, &ve) && ve.Code == dnspolicy.ErrCodeSourceInvalidURL {
			return nil, "ssrf_refused"
		}
		return nil, "fetch_failed"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "http_status"
	}
	limit := min(uint64(p.MaxSizeBytes), uint64(dnspolicy.DefaultMaxSourceBytes))
	if limit == 0 {
		return nil, "invalid_size"
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return nil, "read_failed"
	}
	if uint64(len(raw)) > limit {
		return nil, "oversize"
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	if previous == nil && hash != p.ContentSha256 {
		return nil, "content_mismatch"
	}
	format := dnspolicy.SourceFormat(p.Format)
	if p.Format == api.ThreatFeedPayloadFormatDomainList {
		format = dnspolicy.SourceFormatPlain
	}
	parsed, err := dnspolicy.ParseSourceList(bytes.NewReader(raw), format, dnspolicy.ParseOptions{MaxBytes: int64(limit), MaxRules: dnspolicy.DefaultMaxSourceRules})
	if err != nil || parsed.ParseErrors > 0 {
		return nil, "parse_error"
	}
	if parsed.RuleCount == 0 {
		return nil, "zero_rules"
	}
	// Count removals from the actual previous set; replacement by unrelated rules
	// cannot mask a catastrophic removal by keeping the same total count.
	removed, oldCount := 0, 0
	if previous != nil {
		oldCount = len(previous.Entries)
		next := make(map[dnspolicy.SourceEntry]bool, len(parsed.Entries))
		for _, e := range parsed.Entries {
			next[e] = true
		}
		for _, e := range previous.Entries {
			if !next[e] {
				removed++
			}
		}
		if removed*100 > oldCount*maxRemoval {
			return nil, "delta_breach"
		}
	}
	encoded, err := json.Marshal(parsed.Entries)
	if err != nil || len(encoded) > MaxEmbeddedFeedBytes {
		return nil, "content_too_large"
	}
	kind := "refreshed"
	if previous == nil {
		kind = "approved_snapshot"
	}
	return &Content{SHA256: hash, Size: int64(len(raw)), Entries: parsed.Entries, Kind: kind,
		Metrics: dnspolicy.SourceQualityMetrics{RuleCount: parsed.RuleCount, Delta: parsed.RuleCount - oldCount,
			ParseErrors: parsed.ParseErrors, SkippedUnsupported: parsed.SkippedUnsupported,
			LastUpdate: now.UTC(), SmokeTestResult: dnspolicy.SmokeTestUntested}}, ""
}
