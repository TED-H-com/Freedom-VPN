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
	DefaultUpdateCadence     = dnspolicy.DefaultFeedUpdateIntervalSeconds * time.Second // D-054; 待實測.
	DefaultMaxRemovalPercent = 50
	// Keep embedded feeds bounded within the existing 1 MiB resolver document.
	MaxEmbeddedFeedBytes = 512 * 1024
	MaxEmbeddedFeedRules = 12000
	// At most one approved set's worth of additions, even over many refreshes.
	MaxAdditionPercent = 100
)

// Content is an accepted immutable version. SHA256 hashes the original bytes;
// Entries are normalized and covered by the enclosing resolver document hash.
type Content struct {
	SHA256          string
	Size            int64
	Entries         []dnspolicy.SourceEntry
	ApprovedEntries []dnspolicy.SourceEntry
	Metrics         dnspolicy.SourceQualityMetrics
	Kind            string
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
	c.CheckRedirect = dnspolicy.RefuseSourceRedirect
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.SourceUrl, nil)
	if err != nil {
		return nil, "ssrf_refused"
	}
	req.Header.Set("Accept-Encoding", "identity")
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
	if resp.Uncompressed || (resp.Header.Get("Content-Encoding") != "" && resp.Header.Get("Content-Encoding") != "identity") {
		return nil, "parse_error"
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
	parsed, err := dnspolicy.ParseSourceList(bytes.NewReader(raw), format, dnspolicy.ParseOptions{MaxBytes: int64(limit), MaxRules: MaxEmbeddedFeedRules})
	if err != nil || parsed.ParseErrors > 0 {
		return nil, "parse_error"
	}
	if parsed.RuleCount == 0 {
		return nil, "zero_rules"
	}
	oldCount := 0
	approved := parsed.Entries
	if previous != nil {
		oldCount = len(previous.Entries)
		approved = previous.ApprovedEntries
		if len(approved) == 0 && previous.Kind == "approved_snapshot" {
			approved = previous.Entries
		}
		if len(approved) == 0 {
			return nil, "invalid_config"
		}
		next := make(map[dnspolicy.SourceEntry]bool, len(parsed.Entries))
		for _, entry := range parsed.Entries {
			next[entry] = true
		}
		for _, baseline := range [][]dnspolicy.SourceEntry{previous.Entries, approved} {
			removed := 0
			for _, entry := range baseline {
				if !next[entry] {
					removed++
				}
			}
			if removed*100 > len(baseline)*maxRemoval {
				return nil, "delta_breach"
			}
		}
		pinned := make(map[dnspolicy.SourceEntry]bool, len(approved))
		for _, entry := range approved {
			pinned[entry] = true
		}
		added := 0
		for _, entry := range parsed.Entries {
			if !pinned[entry] {
				added++
			}
		}
		if added*100 > len(approved)*MaxAdditionPercent {
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
	return &Content{SHA256: hash, Size: int64(len(raw)), Entries: parsed.Entries, ApprovedEntries: append([]dnspolicy.SourceEntry(nil), approved...), Kind: kind,
		Metrics: dnspolicy.SourceQualityMetrics{RuleCount: parsed.RuleCount, Delta: parsed.RuleCount - oldCount,
			ParseErrors: parsed.ParseErrors, SkippedUnsupported: parsed.SkippedUnsupported,
			LastUpdate: now.UTC(), SmokeTestResult: dnspolicy.SmokeTestUntested}}, ""
}
