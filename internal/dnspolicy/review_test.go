package dnspolicy

import (
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestReview_F02_ValidateSourceURLAndSSRFAddrBypasses(t *testing.T) {
	t.Run("isBlockedSSRFAddr_misses_special_and_zoned_ipv6", func(t *testing.T) {
		shouldBlock := []struct {
			name string
			addr netip.Addr
		}{
			{name: "ipv4_compatible_v6_metadata", addr: netip.MustParseAddr("::169.254.169.254")},
			{name: "ipv4_compatible_v6_loopback", addr: netip.MustParseAddr("::127.0.0.1")},
			{name: "ipv4_compatible_v6_rfc1918", addr: netip.MustParseAddr("::10.0.0.1")},
			{name: "rfc8215_local_nat64_metadata", addr: netip.MustParseAddr("64:ff9b:1::a9fe:a9fe")},
			{name: "rfc6666_discard_only", addr: netip.MustParseAddr("100::1")},
			{name: "zoned_rfc6052_nat64_metadata", addr: netip.MustParseAddr("64:ff9b::a9fe:a9fe%eth0")},
			{name: "zoned_6to4_metadata", addr: netip.MustParseAddr("2002:a9fe:a9fe::1%eth0")},
			{name: "zoned_documentation_v6", addr: netip.MustParseAddr("2001:db8::1%eth0")},
		}

		for _, tc := range shouldBlock {
			if _, blocked := isBlockedSSRFAddr(tc.addr); !blocked {
				t.Errorf("TODO(review) [F-02]: isBlockedSSRFAddr(%q) = false (%s); want true", tc.addr, tc.name)
			}
		}
	})

	t.Run("ValidateSourceURL_allows_non_443_ports_and_internal_special_domains", func(t *testing.T) {
		shouldReject := []string{
			"https://example.com:6379/feed.txt",
			"https://example.com:99999/feed.txt",
			"https://router.home.arpa/feed.txt",
			"https://internal.corp/feed.txt",
			"https://printer.lan/feed.txt",
			"https://1.0.0.127.in-addr.arpa/feed.txt",
		}

		for _, rawURL := range shouldReject {
			if err := ValidateSourceURL(rawURL); err == nil {
				t.Errorf("TODO(review) [F-02]: ValidateSourceURL(%q) = nil; want %s", rawURL, ErrCodeSourceInvalidURL)
			}
		}
	})
}

func TestReview_F07_SourceRegistryUpdateFromReaderOverwritesLastGoodOnParseErrors(t *testing.T) {
	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	reg := NewDefaultSourceStore()
	if err := reg.Register(ListSource{
		SourceID:          "src-corrupt-test",
		Version:           "1",
		License:           "CC0-1.0",
		LicenseStatus:     LicenseCommercialApproved,
		PermittedUse:      "TEST ONLY commercial simulation",
		DistributionNotes: "Server-side resolver compilation",
		AttributionNotes:  "Test Feed",
		URL:               "https://example.com/list.txt",
		Format:            SourceFormatPlain,
		Category:          CategoryMalware,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := reg.UpdateFromReader("src-corrupt-test", "1", strings.NewReader("initial-good.example.com\n"), DefaultParseOptions(), t0, nil); err != nil {
		t.Fatalf("seed UpdateFromReader: %v", err)
	}

	// Feed update contains 1 valid line and 2 malformed lines (ParseErrors == 2).
	// D-054 requires any parse error to fail the refresh and retain LastGood.
	corruptBody := "replacement.example.com\nhttps://malformed-url-not-a-domain.example.com/path\n@@invalid-line\n"
	err := reg.UpdateFromReader("src-corrupt-test", "2", strings.NewReader(corruptBody), DefaultParseOptions(), t0.Add(time.Minute), nil)

	src, _ := reg.Get("src-corrupt-test")
	if err == nil || src.Version != "1" || len(src.Entries) != 1 || src.Entries[0].Domain != "initial-good.example.com" {
		t.Errorf("TODO(review) [F-07]: UpdateFromReader accepted a body with ParseErrors>0 and overwrote LastGood (err=%v, version=%q, entries=%+v)",
			err, src.Version, src.Entries)
	}
}
