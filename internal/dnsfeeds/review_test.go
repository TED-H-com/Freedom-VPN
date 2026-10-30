package dnsfeeds

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

func TestReview_F01_TwoStepDilutionWipes100PercentOfPinnedRules(t *testing.T) {
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

	t.Run("two_step_dilution_deletes_all_pinned_bootstrap_domains", func(t *testing.T) {
		pinnedDomains := []string{
			"malware-1.example.com",
			"malware-2.example.com",
			"malware-3.example.com",
			"malware-4.example.com",
		}
		body := strings.Join(pinnedDomains, "\n") + "\n"
		client := localFeed(t, &body)
		p := testPayload(body, now)

		approved, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if reason != "" || approved == nil || approved.Kind != "approved_snapshot" {
			t.Fatalf("bootstrap failed: content=%+v reason=%q", approved, reason)
		}

		// Step 1 (Dilution): Upstream keeps all 4 pinned domains and adds 4 benign filler domains.
		// Removed = 0/4 (0% <= 50%), so last_good_content grows from 4 to 8 entries.
		fillerDomains := []string{
			"filler-1.example.invalid",
			"filler-2.example.invalid",
			"filler-3.example.invalid",
			"filler-4.example.invalid",
		}
		body = strings.Join(append(pinnedDomains, fillerDomains...), "\n") + "\n"
		step1, reason := Fetch(context.Background(), client, p, approved, now.Add(5*time.Minute), 50)
		if reason != "" || step1 == nil || len(step1.Entries) != 8 {
			t.Fatalf("step 1 dilution refresh failed unexpectedly: content=%+v reason=%q", step1, reason)
		}

		// Step 2 (Total Wipeout): Upstream removes 100% of the pinned malware domains,
		// serving only the 4 filler domains. Because Fetch compares against step1 (4/8 = 50% <= 50%)
		// instead of the pinned approved_snapshot (4/4 = 100% > 50%), the wipeout succeeds.
		body = strings.Join(fillerDomains, "\n") + "\n"
		step2, reason := Fetch(context.Background(), client, p, step1, now.Add(10*time.Minute), 50)

		pinnedRemaining := 0
		// Rejection returns nil, so inspect the unchanged last-good input for retention.
		// D-054 §4 requires retention, not a successful result on a rejected fetch.
		if step1 != nil {
			pinnedSet := make(map[string]bool, len(pinnedDomains))
			for _, d := range pinnedDomains {
				pinnedSet[d] = true
			}
			for _, e := range step1.Entries {
				if pinnedSet[e.Domain] {
					pinnedRemaining++
				}
			}
		}

		if reason != "delta_breach" || step2 != nil || pinnedRemaining != len(pinnedDomains) {
			t.Errorf("[F-01]: two-step dilution wiped %d/%d (100%%) of pinned bootstrap domains without triggering delta_breach (got reason=%q, pinnedRemaining=%d, step2Entries=%+v)",
				len(pinnedDomains)-pinnedRemaining, len(pinnedDomains), reason, pinnedRemaining, step2)
		}
	})

	t.Run("unbounded_addition_and_public_suffix_injection", func(t *testing.T) {
		body := "malware-1.example.com\n"
		client := localFeed(t, &body)
		p := testPayload(body, now)
		p.MaxSizeBytes = 1024 * 1024

		approved, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if reason != "" || approved == nil {
			t.Fatalf("bootstrap failed: %q", reason)
		}

		// On refresh, upstream keeps the 1 pinned rule and injects 2-label public suffixes
		// ("co.uk", "com.tw") and internal suffixes ("router.home.arpa", "intranet.corp")
		// with IncludeSubdomains=true (rendered as ||co.uk^ in AdGuard syntax), plus 500 extra rules.
		var sb strings.Builder
		sb.WriteString("malware-1.example.com\nco.uk\ncom.tw\nrouter.home.arpa\nintranet.corp\n")
		for i := range 500 {
			fmt.Fprintf(&sb, "injected-%d.example.com\n", i)
		}
		body = sb.String()

		refreshed, reason := Fetch(context.Background(), client, p, approved, now.Add(5*time.Minute), 50)
		if reason == "" && refreshed != nil {
			t.Errorf("[F-01]: refresh accepted 505x addition growth and public-suffix/internal wildcard rules (co.uk, com.tw, router.home.arpa, intranet.corp) with IncludeSubdomains=true (got reason=%q, entries=%d)",
				reason, len(refreshed.Entries))
		}
	})
}

func TestReview_F02_SSRFGuardBypassesAndPortAndRedirectClassification(t *testing.T) {
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	body := "a.example.com\n"

	t.Run("bypass_addresses_dialed_instead_of_ssrf_refused", func(t *testing.T) {
		bypassIPs := []struct {
			name string
			addr netip.Addr
		}{
			{name: "ipv4_compatible_v6_metadata", addr: netip.MustParseAddr("::169.254.169.254")},
			{name: "ipv4_compatible_v6_loopback", addr: netip.MustParseAddr("::127.0.0.1")},
			{name: "rfc8215_local_nat64_metadata", addr: netip.MustParseAddr("64:ff9b:1::a9fe:a9fe")},
			{name: "rfc6666_discard_prefix", addr: netip.MustParseAddr("100::1")},
			{name: "zoned_rfc6052_nat64_metadata", addr: netip.MustParseAddr("64:ff9b::a9fe:a9fe%lo")},
			{name: "zoned_6to4_metadata", addr: netip.MustParseAddr("2002:a9fe:a9fe::1%lo")},
		}

		for _, tc := range bypassIPs {
			t.Run(tc.name, func(t *testing.T) {
				p := testPayload(body, now)
				dialed := ""
				client := dnspolicy.NewSourceHTTPClient(time.Second, func(context.Context, string, string) ([]netip.Addr, error) {
					return []netip.Addr{tc.addr}, nil
				}, func(_ context.Context, _, address string) (net.Conn, error) {
					dialed = address
					return nil, fmt.Errorf("abort dial to %s", address)
				})

				got, reason := Fetch(context.Background(), client, p, nil, now, 50)
				if dialed != "" || reason != "ssrf_refused" || got != nil {
					t.Errorf("[F-02]: resolved address %s bypassed SSRF guard and reached dialContext (dialed=%q, reason=%q; want dialed=\"\", reason=\"ssrf_refused\")",
						tc.addr, dialed, reason)
				}
			})
		}
	})

	t.Run("non_443_tcp_port_allowed_and_dialed", func(t *testing.T) {
		p := testPayload(body, now)
		p.SourceUrl = "https://example.com:6379/feed"
		dialed := ""
		client := dnspolicy.NewSourceHTTPClient(time.Second, func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
		}, func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = address
			return nil, fmt.Errorf("abort dial to %s", address)
		})

		got, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if dialed != "" || reason != "ssrf_refused" || got != nil {
			t.Errorf("[F-02]: non-443 port in source_url %q was not refused by SSRF guard (dialed=%q, reason=%q; want dialed=\"\", reason=\"ssrf_refused\")",
				p.SourceUrl, dialed, reason)
		}
	})

	t.Run("redirect_to_metadata_misclassified_as_fetch_failed", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
		}))
		defer server.Close()

		client := server.Client()
		tr := client.Transport.(*http.Transport).Clone()
		tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
		}
		client.Transport = tr

		p := testPayload(body, now)
		_, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if reason != "ssrf_refused" {
			t.Errorf("[F-02]: HTTP redirect to metadata endpoint was classified as reason=%q instead of \"ssrf_refused\" because fetch.go overwrites CheckRedirect",
				reason)
		}
	})
}

func TestReview_F03_ParserMemoryAmplificationAndMaxRulesMismatch(t *testing.T) {
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

	t.Run("hosts_single_line_non_sinkhole_allocates_100k_fields_with_zero_parse_errors", func(t *testing.T) {
		// A single line starting with a non-sinkhole IP ("1.2.3.4") followed by 100,000 tokens
		// forces strings.Fields(line) to allocate 100,001 string headers before isHostsSinkholeIP
		// skips the line via SkippedUnsupported++ with ParseErrors == 0.
		body := "0.0.0.0 valid.example.com\n1.2.3.4 " + strings.Repeat("a ", 100000) + "\n"
		client := localFeed(t, &body)
		p := testPayload(body, now)
		p.Format = api.ThreatFeedPayloadFormatHosts
		p.MaxSizeBytes = 1000000

		got, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if reason != "parse_error" || got != nil {
			t.Errorf("[F-03]: hosts parser split a 200KB single line into 100,001 fields and accepted the feed with ParseErrors=0 (reason=%q, skippedUnsupported=%d)",
				reason, got.Metrics.SkippedUnsupported)
		}
	})

	t.Run("transparent_gzip_expansion_and_bytes_split_line_amplification", func(t *testing.T) {
		// 200,000 empty lines + 1 valid domain compresses to ~250 bytes on the wire via gzip,
		// transparently decompresses in http.Transport, and allocates 200,002 []byte headers in bytes.Split.
		expanded := strings.Repeat("\n", 200000) + "a.example.com\n"
		var gzBuf bytes.Buffer
		gz := gzip.NewWriter(&gzBuf)
		_, _ = gz.Write([]byte(expanded))
		_ = gz.Close()
		compressedBytes := gzBuf.Bytes()

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(compressedBytes)
		}))
		defer server.Close()

		client := server.Client()
		tr := client.Transport.(*http.Transport).Clone()
		tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
		}
		client.Transport = tr

		p := testPayload(expanded, now)
		p.MaxSizeBytes = 500000

		got, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if len(compressedBytes) < 1024 && reason == "" && got != nil {
			t.Errorf("[F-03]: %d-byte gzip response transparently expanded to %d bytes of newlines, allocated 200,002 slice headers in bytes.Split, and succeeded (reason=%q)",
				len(compressedBytes), len(expanded), reason)
		}
	})

	t.Run("max_rules_500k_runs_idna_before_512kib_embedded_cap", func(t *testing.T) {
		var sb strings.Builder
		for i := range 15000 {
			fmt.Fprintf(&sb, "sub-%d.example.com\n", i)
		}
		raw := sb.String()
		client := localFeed(t, &raw)
		p := testPayload(raw, now)
		p.MaxSizeBytes = 32000000

		_, reason := Fetch(context.Background(), client, p, nil, now, 50)
		if reason == "content_too_large" {
			t.Errorf("[F-03]: Fetch ran full IDNA normalization and sorting on 15,000 rules (MaxRules=500000) before failing post-parse size check with %q; MaxRules should be bounded to fit MaxEmbeddedFeedBytes (~12k rules)",
				reason)
		}
	})
}

func TestReview_F06_MetricsLastGoodAgePoisonedBySupersededRevision(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	reg := prometheus.NewRegistry()
	m := newFeedMetrics(func() time.Time { return now })
	reg.MustRegister(m)

	rev1 := uuid.New()
	rev2 := uuid.New()

	// Revision 1 of "malware-feed" fails bootstrap (lastGood is zero time -> age = +Inf).
	m.set(rev1, "malware-feed", time.Time{})

	// Operator activates Revision 2 of the same "malware-feed", which succeeds bootstrap at now (age = 0).
	m.set(rev2, "malware-feed", now)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	var gotAge float64 = -1
	for _, fam := range families {
		if fam.GetName() == "fvpn_dns_feed_last_good_age_seconds" {
			for _, metric := range fam.GetMetric() {
				for _, lp := range metric.GetLabel() {
					if lp.GetName() == "feed_code" && lp.GetValue() == "malware-feed" {
						gotAge = metric.GetGauge().GetValue()
					}
				}
			}
		}
	}

	if math.IsInf(gotAge, 1) || gotAge != 0 {
		t.Errorf("[F-06]: fvpn_dns_feed_last_good_age_seconds{feed_code=\"malware-feed\"} = %v after superseded rev1 failed (+Inf) and active rev2 succeeded (0); stale revision entries in m.ages poison the gauge",
			gotAge)
	}
}
