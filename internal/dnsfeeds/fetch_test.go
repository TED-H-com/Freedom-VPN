package dnsfeeds

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// TEST ONLY: map the approved HTTPS name to a local TLS fixture. Never perform
// external DNS or use a public feed. Production receives no injected transport.
func localFeed(t *testing.T, body *string) *http.Client {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, *body) }))
	t.Cleanup(server.Close)
	client := server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr
	return client
}
func digest(body string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(body))) }
func testPayload(body string, now time.Time) api.ThreatFeedPayload {
	return api.ThreatFeedPayload{Provider: "TEST ONLY", FeedCode: "test-feed", LicenseName: "TEST ONLY", LicenseUrl: "https://example.com/license", CommercialUseBasis: "TEST ONLY permission for fixture use", SourceUrl: "https://example.com/feed", ContentSha256: digest(body), Format: api.ThreatFeedPayloadFormatDomainList, Categories: []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware}, MaxSizeBytes: 4096, ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(30 * 24 * time.Hour), UpdateIntervalSeconds: 300}
}

func TestD054_DNS01_BootstrapAndRefreshGuards(t *testing.T) {
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	body := "A.example.com\nb.example.com\nc.example.com\nd.example.com\n"
	client := localFeed(t, &body)
	p := testPayload(body, now)
	previous, reason := Fetch(context.Background(), client, p, nil, now, 50)
	if reason != "" || previous.Kind != "approved_snapshot" || previous.Entries[0].Domain != "a.example.com" {
		t.Fatalf("bootstrap: %+v %s", previous, reason)
	}
	original := append([]dnspolicy.SourceEntry(nil), previous.Entries...)
	for _, tt := range []struct {
		name, body, reason string
		mutate             func(*api.ThreatFeedPayload)
		bootstrap          bool
	}{
		{name: "mismatch", body: "new.example.com\n", reason: "content_mismatch", bootstrap: true},
		{name: "oversize", body: strings.Repeat("a", 4097), reason: "oversize"},
		{name: "parse", body: "a.example.com\nhttps://invalid.example.com\n", reason: "parse_error"},
		{name: "empty", body: "# comment\n", reason: "zero_rules"},
		{name: "delta", body: "a.example.com\n", reason: "delta_breach"},
		{name: "replacement_same_count", body: "w.example.com\nx.example.com\ny.example.com\nz.example.com\n", reason: "delta_breach"},
		{name: "ssrf", body: "a.example.com\n", reason: "ssrf_refused", mutate: func(p *api.ThreatFeedPayload) { p.SourceUrl = "https://127.0.0.1/feed" }},
		{name: "expired", body: "a.example.com\n", reason: "outside_validity", mutate: func(p *api.ThreatFeedPayload) { p.ValidUntil = now }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body = tt.body
			cp := p
			if tt.mutate != nil {
				tt.mutate(&cp)
			}
			prev := previous
			if tt.bootstrap {
				prev = nil
			}
			got, reason := Fetch(context.Background(), client, cp, prev, now, 50)
			if got != nil || reason != tt.reason {
				t.Fatalf("got %+v reason %q want %q", got, reason, tt.reason)
			}
			if !reflect.DeepEqual(previous.Entries, original) {
				t.Fatal("last good mutated")
			}
		})
	}
	body = "a.example.com\nb.example.com\nnew.example.com\n"
	refreshed, reason := Fetch(context.Background(), client, p, previous, now, 50)
	if reason != "" || refreshed.Kind != "refreshed" || refreshed.SHA256 == p.ContentSha256 || refreshed.Metrics.Delta != -1 {
		t.Fatalf("50 percent boundary: %+v %s", refreshed, reason)
	}
}

func TestD054_SEC02_ResolvedSSRFAndRedirectRefused(t *testing.T) {
	now := time.Now()
	body := "a.example.com\n"
	p := testPayload(body, now)
	client := dnspolicy.NewSourceHTTPClient(time.Second, func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}, func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("SSRF dial occurred")
		return nil, nil
	})
	if c, reason := Fetch(context.Background(), client, p, nil, now, 50); c != nil || reason != "ssrf_refused" {
		t.Fatalf("SSRF: %s", reason)
	}
	reached := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/private" {
			reached = true
		}
		http.Redirect(w, r, "https://example.com/private", http.StatusFound)
	}))
	defer server.Close()
	client = server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr
	if c, reason := Fetch(context.Background(), client, p, nil, now, 50); c != nil || reason != "ssrf_refused" || reached {
		t.Fatalf("redirect: %s reached=%v", reason, reached)
	}
}

func TestD054_DNS02_HostsAreExactAndAdblockIsLayerThree(t *testing.T) {
	now := time.Now()
	body := "0.0.0.0 HOST.example.com\n"
	client := localFeed(t, &body)
	p := testPayload(body, now)
	p.Format = api.ThreatFeedPayloadFormatHosts
	c, reason := Fetch(context.Background(), client, p, nil, now, 50)
	if reason != "" || c.Entries[0].IncludeSubdomains {
		t.Fatalf("hosts scope: %+v %s", c, reason)
	}
	body = "||a.example.com^\n@@||b.example.com^\n||c.example.com^$important\n"
	p = testPayload(body, now)
	p.Format = api.ThreatFeedPayloadFormatAdblock
	c, reason = Fetch(context.Background(), client, p, nil, now, 50)
	if reason != "" || len(c.Entries) != 1 || c.Metrics.SkippedUnsupported != 2 {
		t.Fatalf("adblock: %+v %s", c, reason)
	}
}

func TestD054_DNS01_NormalizedContentIsBounded(t *testing.T) {
	var body strings.Builder
	for i := range 10000 {
		fmt.Fprintf(&body, "entry-%d.example.com\n", i)
	}
	raw := body.String()
	client := localFeed(t, &raw)
	now := time.Now()
	p := testPayload(raw, now)
	p.MaxSizeBytes = 32000000
	if c, reason := Fetch(context.Background(), client, p, nil, now, 50); c != nil || reason != "content_too_large" {
		t.Fatalf("embedded bound: %s", reason)
	}
}
