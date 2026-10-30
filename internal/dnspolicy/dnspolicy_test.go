package dnspolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

type vectorFile struct {
	SchemaVersion        int                   `json:"schema_version"`
	Description          string                `json:"description"`
	NormalizationVectors []normalizationVector `json:"normalization_vectors"`
	ConflictVectors      []conflictVector      `json:"conflict_vectors"`
	EvaluationProfile    Profile               `json:"evaluation_profile"`
	EvaluationVectors    []evaluationVector    `json:"evaluation_vectors"`
}

type normalizationVector struct {
	Name      string              `json:"name"`
	Input     string              `json:"input"`
	Valid     bool                `json:"valid"`
	Canonical string              `json:"canonical,omitempty"`
	ErrorCode ValidationErrorCode `json:"error_code,omitempty"`
}

type conflictVector struct {
	Name              string              `json:"name"`
	ExpectedErrorCode ValidationErrorCode `json:"expected_error_code"`
	Profile           Profile             `json:"profile"`
}

type evaluationVector struct {
	Name             string      `json:"name"`
	Input            string      `json:"input"`
	ExpectedAction   RuleAction  `json:"expected_action"`
	ExpectedLayer    PolicyLayer `json:"expected_layer"`
	ExpectedRuleID   string      `json:"expected_rule_id,omitempty"`
	ExpectedSourceID string      `json:"expected_source_id,omitempty"`
}

func loadVectors(t *testing.T) vectorFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "vectors.json"))
	if err != nil {
		t.Fatalf("read testdata/vectors.json: %v", err)
	}
	var vf vectorFile
	if err := json.Unmarshal(raw, &vf); err != nil {
		t.Fatalf("unmarshal testdata/vectors.json: %v", err)
	}
	return vf
}

func TestDNS02_AT20_NormalizationVectors(t *testing.T) {
	vf := loadVectors(t)
	if len(vf.NormalizationVectors) == 0 {
		t.Fatal("expected normalization_vectors in testdata/vectors.json")
	}
	if desc := IDNAProfileDescription(); desc == "" {
		t.Fatal("expected non-empty IDNAProfileDescription")
	}

	for _, tc := range vf.NormalizationVectors {
		t.Run(tc.Name, func(t *testing.T) {
			norm, err := NormalizeDomain(tc.Input)
			if tc.Valid {
				if err != nil {
					t.Fatalf("NormalizeDomain(%q) unexpected error: %v (code=%s)", tc.Input, err, ErrorCode(err))
				}
				if norm.Canonical != tc.Canonical {
					t.Errorf("NormalizeDomain(%q) canonical = %q, want %q", tc.Input, norm.Canonical, tc.Canonical)
				}
				if norm.Original != tc.Input {
					t.Errorf("NormalizeDomain(%q) original = %q, want %q", tc.Input, norm.Original, tc.Input)
				}
			} else {
				if err == nil {
					t.Fatalf("NormalizeDomain(%q) succeeded with canonical=%q, expected error code %s", tc.Input, norm.Canonical, tc.ErrorCode)
				}
				gotCode := ErrorCode(err)
				if gotCode != tc.ErrorCode {
					t.Errorf("NormalizeDomain(%q) error code = %q, want %q (err=%v)", tc.Input, gotCode, tc.ErrorCode, err)
				}
			}
		})
	}
}

func TestDNS01_AT20_ConflictAndDraftValidationVectors(t *testing.T) {
	vf := loadVectors(t)
	for _, tc := range vf.ConflictVectors {
		t.Run(tc.Name, func(t *testing.T) {
			err := ValidateDraft(tc.Profile, nil, nil)
			if err == nil {
				t.Fatalf("ValidateDraft(%s) succeeded, expected error code %s", tc.Name, tc.ExpectedErrorCode)
			}
			gotCode := ErrorCode(err)
			if gotCode != tc.ExpectedErrorCode {
				t.Errorf("ValidateDraft(%s) error code = %q, want %q (err=%v)", tc.Name, gotCode, tc.ExpectedErrorCode, err)
			}
		})
	}

	t.Run("duplicate_rule_id", func(t *testing.T) {
		p := Profile{
			ProfileID: "standard",
			Revision:  "1",
			Rules: []Rule{
				{RuleID: "dup-1", ProfileID: "standard", Domain: "a.example.com", Action: ActionBlock, Revision: "1"},
				{RuleID: "dup-1", ProfileID: "standard", Domain: "b.example.com", Action: ActionBlock, Revision: "1"},
			},
		}
		err := ValidateDraft(p, nil, nil)
		if ErrorCode(err) != ErrCodeRuleIDDuplicate {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeRuleIDDuplicate, err)
		}
	})

	t.Run("unknown_source_reference", func(t *testing.T) {
		reg := NewDefaultSourceRegistry()
		p := Profile{
			ProfileID:      "standard",
			Revision:       "1",
			EnabledSources: []string{"non-existent-feed"},
		}
		err := ValidateDraft(p, reg, nil)
		if ErrorCode(err) != ErrCodeSourceNotFound {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeSourceNotFound, err)
		}
	})

	t.Run("unapproved_source_license_rejected", func(t *testing.T) {
		reg := NewDefaultSourceRegistry()
		err := reg.Register(ListSource{
			SourceID:          "feed-pending",
			Version:           "1",
			License:           "GPL-3.0",
			LicenseStatus:     LicensePendingDecision,
			PermittedUse:      "Pending legal review",
			DistributionNotes: "Server-side only",
			AttributionNotes:  new("Example Feed"),
			URL:               "https://feeds.example.invalid/list.txt",
			Format:            SourceFormatPlain,
			Category:          CategoryMalware,
		})
		if err != nil {
			t.Fatalf("register source: %v", err)
		}
		p := Profile{
			ProfileID:      "standard",
			Revision:       "1",
			EnabledSources: []string{"feed-pending"},
		}
		err = ValidateDraft(p, reg, nil)
		if ErrorCode(err) != ErrCodeSourceLicenseUnapproved {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeSourceLicenseUnapproved, err)
		}
	})

	t.Run("rule_count_limits_enforced", func(t *testing.T) {
		p := Profile{
			ProfileID: "standard",
			Revision:  "1",
			Rules: []Rule{
				{RuleID: "r1", ProfileID: "standard", Domain: "a.example.com", Action: ActionBlock, Revision: "1"},
				{RuleID: "r2", ProfileID: "standard", Domain: "b.example.com", Action: ActionBlock, Revision: "1"},
				{RuleID: "r3", ProfileID: "standard", Domain: "c.example.com", Action: ActionBlock, Revision: "1"},
			},
		}
		err := ValidateDraft(p, nil, &DraftValidationLimits{MaxAdminRulesPerProfile: 2})
		if ErrorCode(err) != ErrCodeRuleLimitExceeded {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeRuleLimitExceeded, err)
		}
	})

	t.Run("invalid_revisions_rejected", func(t *testing.T) {
		badRevs := []string{"", "-1", "01", "1.0", "1e5", "abc", "18446744073709551616", "123456789012345678901"}
		for _, rev := range badRevs {
			if err := ValidateDecimalRevision(rev); ErrorCode(err) != ErrCodeInvalidRevision {
				t.Errorf("ValidateDecimalRevision(%q) = %v, want %s", rev, err, ErrCodeInvalidRevision)
			}
		}
		goodRevs := []string{"0", "1", "42", "18446744073709551615"}
		for _, rev := range goodRevs {
			if err := ValidateDecimalRevision(rev); err != nil {
				t.Errorf("ValidateDecimalRevision(%q) unexpected error: %v", rev, err)
			}
		}
	})
}

func TestDNS01_DNS02_AT20_EvaluationVectors(t *testing.T) {
	vf := loadVectors(t)
	if err := ValidateDraft(vf.EvaluationProfile, nil, nil); err != nil {
		t.Fatalf("ValidateDraft(evaluation_profile) failed: %v", err)
	}

	for _, tc := range vf.EvaluationVectors {
		t.Run(tc.Name, func(t *testing.T) {
			dec := Evaluate(vf.EvaluationProfile, tc.Input)
			if dec.Err != nil {
				t.Fatalf("Evaluate(%q) returned unexpected error: %v", tc.Input, dec.Err)
			}
			if dec.Action != tc.ExpectedAction {
				t.Errorf("Evaluate(%q) action = %q, want %q (layer=%s, rule=%s, source=%s)",
					tc.Input, dec.Action, tc.ExpectedAction, dec.Layer, dec.MatchedRuleID, dec.MatchedSourceID)
			}
			if dec.Layer != tc.ExpectedLayer {
				t.Errorf("Evaluate(%q) layer = %q, want %q", tc.Input, dec.Layer, tc.ExpectedLayer)
			}
			if tc.ExpectedRuleID != "" && dec.MatchedRuleID != tc.ExpectedRuleID {
				t.Errorf("Evaluate(%q) matched_rule_id = %q, want %q", tc.Input, dec.MatchedRuleID, tc.ExpectedRuleID)
			}
			if tc.ExpectedSourceID != "" && dec.MatchedSourceID != tc.ExpectedSourceID {
				t.Errorf("Evaluate(%q) matched_source_id = %q, want %q", tc.Input, dec.MatchedSourceID, tc.ExpectedSourceID)
			}
		})
	}

	t.Run("evaluate_invalid_input_fails_closed", func(t *testing.T) {
		dec := Evaluate(vf.EvaluationProfile, "https://example.com/bad")
		if dec.Action != ActionBlock || dec.Layer != LayerInvalidInput || dec.Err == nil {
			t.Fatalf("expected fail-closed block on invalid domain input, got %+v", dec)
		}
	})
}

func TestDNS01_AT22_SourceParsers(t *testing.T) {
	t.Run("plain_domains_parser", func(t *testing.T) {
		input := strings.Join([]string{
			"# Malware domain list",
			"",
			"MALWARE.EXAMPLE.COM. # inline comment",
			"phishing.test",
			"münchen.example.com",
			"phishing.test", // duplicate
			"https://bad-url.example.com/path",
			"two tokens.example.com",
			"-invalid-hyphen.example.com",
		}, "\n")

		res, err := ParsePlainDomains(strings.NewReader(input), DefaultParseOptions())
		if err != nil {
			t.Fatalf("ParsePlainDomains failed: %v", err)
		}
		if res.RuleCount != 3 {
			t.Fatalf("RuleCount = %d, want 3 (entries=%+v)", res.RuleCount, res.Entries)
		}
		if res.ParseErrors != 3 {
			t.Errorf("ParseErrors = %d, want 3", res.ParseErrors)
		}
		wantDomains := []string{"malware.example.com", "phishing.test", "xn--mnchen-3ya.example.com"}
		for i, want := range wantDomains {
			if res.Entries[i].Domain != want || !res.Entries[i].IncludeSubdomains {
				t.Errorf("entry[%d] = %+v, want domain=%q include_subdomains=true", i, res.Entries[i], want)
			}
		}
	})

	t.Run("hosts_file_parser", func(t *testing.T) {
		input := strings.Join([]string{
			"# Standard hosts file",
			"127.0.0.1 localhost localhost.localdomain",
			"::1 localhost ip6-localhost ip6-loopback",
			"0.0.0.0 0.0.0.0",
			"0.0.0.0 tracker1.example.com tracker2.example.net # two hosts on one line",
			"127.0.0.1 BAD-HOST.TEST.",
			"192.0.2.55 spoofed-redirect.example.com", // non-sinkhole IP must be skipped
			"not-an-ip malformed.example.com",
			"0.0.0.0 *.wildcard.example.com",
			"0.0.0.0", // missing hostname
		}, "\n")

		res, err := ParseHostsFile(strings.NewReader(input), DefaultParseOptions())
		if err != nil {
			t.Fatalf("ParseHostsFile failed: %v", err)
		}
		if res.RuleCount != 3 {
			t.Fatalf("RuleCount = %d, want 3 (entries=%+v)", res.RuleCount, res.Entries)
		}
		if res.SkippedUnsupported != 7 {
			t.Errorf("SkippedUnsupported = %d, want 7", res.SkippedUnsupported)
		}
		if res.ParseErrors != 3 {
			t.Errorf("ParseErrors = %d, want 3", res.ParseErrors)
		}
		wantDomains := []string{"bad-host.test", "tracker1.example.com", "tracker2.example.net"}
		for i, want := range wantDomains {
			if res.Entries[i].Domain != want {
				t.Errorf("entry[%d].Domain = %q, want %q", i, res.Entries[i].Domain, want)
			}
		}
	})

	t.Run("adblock_domain_subset_parser", func(t *testing.T) {
		input := strings.Join([]string{
			"[Adblock Plus 2.0]",
			"! Title: Example Threat List",
			"# Comment line",
			"||Malware-One.Example.com^",
			"||phishing-two.test.^",
			"||аpple.example.com^",
			"@@||allowed-exception.example.com^",         // skipped unsupported (feeds cannot inject allows)
			"||forced-escalation.example.com^$important", // skipped unsupported (feeds cannot inject $important)
			"||path-rule.example.com/login^",             // skipped unsupported
			"||wild*.example.com^",                       // skipped unsupported
			"example.com##.banner-ad",                    // skipped unsupported cosmetic
			"/regex-rule/",                               // skipped unsupported
			"||^",                                        // parse error (empty domain)
			"||bad..domain.example.com^",                 // parse error (empty label)
			"random garbage line",                        // parse error
		}, "\n")

		res, err := ParseAdBlockDomainSubset(strings.NewReader(input), DefaultParseOptions())
		if err != nil {
			t.Fatalf("ParseAdBlockDomainSubset failed: %v", err)
		}
		if res.RuleCount != 3 {
			t.Fatalf("RuleCount = %d, want 3 (entries=%+v)", res.RuleCount, res.Entries)
		}
		if res.SkippedUnsupported != 6 {
			t.Errorf("SkippedUnsupported = %d, want 6", res.SkippedUnsupported)
		}
		if res.ParseErrors != 3 {
			t.Errorf("ParseErrors = %d, want 3", res.ParseErrors)
		}
		wantDomains := []string{"malware-one.example.com", "phishing-two.test", "xn--pple-43d.example.com"}
		for i, want := range wantDomains {
			if res.Entries[i].Domain != want {
				t.Errorf("entry[%d].Domain = %q, want %q", i, res.Entries[i].Domain, want)
			}
		}
	})
}

func TestDNS01_AT22_ParserMalformedAndOversized(t *testing.T) {
	t.Run("oversized_input_rejected", func(t *testing.T) {
		big := bytes.Repeat([]byte("a.example.com\n"), 200)
		_, err := ParsePlainDomains(bytes.NewReader(big), ParseOptions{
			MaxBytes:        1024,
			MaxRules:        1000,
			RequireNonEmpty: true,
		})
		if ErrorCode(err) != ErrCodeSourceTooLarge {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeSourceTooLarge, err)
		}
	})

	t.Run("rule_limit_exceeded_rejected", func(t *testing.T) {
		input := "d1.example.com\nd2.example.com\nd3.example.com\n"
		_, err := ParsePlainDomains(strings.NewReader(input), ParseOptions{
			MaxBytes:        10240,
			MaxRules:        2,
			RequireNonEmpty: true,
		})
		if ErrorCode(err) != ErrCodeRuleLimitExceeded {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeRuleLimitExceeded, err)
		}
	})

	t.Run("all_malformed_never_fails_open", func(t *testing.T) {
		input := "# only comments and invalid lines\nhttps://not-a-domain.example.com/path\n*.wild.com\n"
		_, err := ParsePlainDomains(strings.NewReader(input), DefaultParseOptions())
		if ErrorCode(err) != ErrCodeSourceEmpty {
			t.Fatalf("got error code %q, want %q (err=%v)", ErrorCode(err), ErrCodeSourceEmpty, err)
		}
	})
}

func TestDNS01_AT22_SourceURLValidationAndSSRF(t *testing.T) {
	invalidURLs := []string{
		"",
		"http://feeds.example.com/list.txt",
		"ftp://feeds.example.com/list.txt",
		"https://user:pass@feeds.example.com/list.txt",
		"https://127.0.0.1/list.txt",
		"https://169.254.169.254/latest/meta-data/",
		"https://10.0.0.1/list.txt",
		"https://192.168.1.1/list.txt",
		"https://100.64.0.1/list.txt",
		"https://[::1]/list.txt",
		"https://192.0.2.1/list.txt",
		"https://localhost/list.txt",
		"https://feed.local/list.txt",
		"https://feed.internal/list.txt",
	}
	for _, u := range invalidURLs {
		if err := ValidateSourceURL(u); ErrorCode(err) != ErrCodeSourceInvalidURL {
			t.Errorf("ValidateSourceURL(%q) = %v, want %s", u, err, ErrCodeSourceInvalidURL)
		}
	}

	if err := ValidateSourceURL("https://feeds.example.com/malware/domains.txt"); err != nil {
		t.Errorf("ValidateSourceURL valid URL failed: %v", err)
	}
}

func TestDNS01_AT22_SourceLastGoodRetention(t *testing.T) {
	reg := NewDefaultSourceRegistry()
	if len(reg.List()) != 0 {
		t.Fatal("default source registry must have no feeds enabled by default")
	}

	err := reg.Register(ListSource{
		SourceID:          "src-threat",
		Version:           "1",
		License:           "CC0-1.0",
		LicenseStatus:     LicenseCommercialApproved,
		PermittedUse:      "TEST ONLY commercial simulation",
		DistributionNotes: "Server-side resolver compilation",
		AttributionNotes:  new("Test Feed"),
		URL:               "https://feeds.example.invalid/threat.txt",
		Format:            SourceFormatPlain,
		Category:          CategoryMalware,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	v1Content := "bad1.example.com\nbad2.example.com\n"
	if err := reg.UpdateFromReader("src-threat", "1", strings.NewReader(v1Content), DefaultParseOptions(), t0, nil); err != nil {
		t.Fatalf("initial UpdateFromReader failed: %v", err)
	}

	srcV1, _ := reg.Get("src-threat")
	if srcV1.Version != "1" || len(srcV1.Entries) != 2 || srcV1.LastGood == nil {
		t.Fatalf("unexpected v1 state: %+v", srcV1)
	}
	v1Hash := srcV1.ContentSHA256

	// 1. Failed parse (empty/corrupted content) must keep v1.
	if err := reg.UpdateFromReader("src-threat", "2", strings.NewReader("# corrupted empty file\n"), DefaultParseOptions(), t0.Add(time.Minute), nil); err == nil {
		t.Fatal("expected error on empty source update")
	}
	afterEmpty, _ := reg.Get("src-threat")
	if afterEmpty.Version != "1" || afterEmpty.ContentSHA256 != v1Hash || len(afterEmpty.Entries) != 2 {
		t.Fatalf("source failed to retain last_good after empty parse: %+v", afterEmpty)
	}

	// 2. Oversized content must keep v1.
	if err := reg.UpdateFromReader("src-threat", "3", strings.NewReader(strings.Repeat("a.example.com\n", 100)), ParseOptions{MaxBytes: 32, RequireNonEmpty: true}, t0.Add(2*time.Minute), nil); err == nil {
		t.Fatal("expected error on oversized source update")
	}
	afterOversize, _ := reg.Get("src-threat")
	if afterOversize.Version != "1" || afterOversize.ContentSHA256 != v1Hash || len(afterOversize.Entries) != 2 {
		t.Fatalf("source failed to retain last_good after oversized parse: %+v", afterOversize)
	}

	// 3. Smoke check failure must keep v1.
	smokeErr := errors.New("synthetic smoke check failure")
	if err := reg.UpdateFromReader("src-threat", "4", strings.NewReader("bad3.example.com\n"), DefaultParseOptions(), t0.Add(3*time.Minute), func([]SourceEntry) error {
		return smokeErr
	}); err == nil {
		t.Fatal("expected error on smoke check failure")
	}
	afterSmokeFail, _ := reg.Get("src-threat")
	if afterSmokeFail.Version != "1" || afterSmokeFail.ContentSHA256 != v1Hash || len(afterSmokeFail.Entries) != 2 {
		t.Fatalf("source failed to retain last_good after smoke check failure: %+v", afterSmokeFail)
	}

	// 4. Valid v2 update advances Version, computes Delta, and updates LastGood.
	v2Content := "bad1.example.com\nbad2.example.com\nbad3.example.com\n"
	if err := reg.UpdateFromReader("src-threat", "2", strings.NewReader(v2Content), DefaultParseOptions(), t0.Add(4*time.Minute), nil); err != nil {
		t.Fatalf("valid v2 update failed: %v", err)
	}
	afterV2, _ := reg.Get("src-threat")
	if afterV2.Version != "2" || len(afterV2.Entries) != 3 || afterV2.Quality.Delta != 1 {
		t.Fatalf("unexpected v2 state: %+v", afterV2)
	}
}

func TestDNS01_CompilerDeterminism(t *testing.T) {
	vf := loadVectors(t)
	validFrom := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	validUntil := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)

	opts := CompileOptions{
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		UpstreamDNS:    []string{"127.0.0.1:15353"},
		AllowedClients: []string{"127.0.0.0/8", "::1/128"},
	}

	baseBundle, err := CompileBundle(vf.EvaluationProfile, opts)
	if err != nil {
		t.Fatalf("CompileBundle failed: %v", err)
	}
	if err := VerifyBundle(baseBundle, validFrom.Add(time.Hour)); err != nil {
		t.Fatalf("VerifyBundle failed: %v", err)
	}

	// Reverse the order of Rules and Sources in copies of the profile and ensure
	// CompileBundle produces byte-for-byte identical output and SHA-256.
	for iter := 0; iter < 10; iter++ {
		shuffled := vf.EvaluationProfile
		shuffled.Rules = make([]Rule, len(vf.EvaluationProfile.Rules))
		for i := range vf.EvaluationProfile.Rules {
			shuffled.Rules[i] = vf.EvaluationProfile.Rules[len(vf.EvaluationProfile.Rules)-1-i]
		}
		shuffled.Sources = make([]ListSource, len(vf.EvaluationProfile.Sources))
		for i := range vf.EvaluationProfile.Sources {
			shuffled.Sources[i] = vf.EvaluationProfile.Sources[len(vf.EvaluationProfile.Sources)-1-i]
		}

		b, err := CompileBundle(shuffled, opts)
		if err != nil {
			t.Fatalf("iter %d CompileBundle failed: %v", iter, err)
		}
		if b.SHA256 != baseBundle.SHA256 {
			t.Fatalf("iter %d SHA256 = %s, want %s", iter, b.SHA256, baseBundle.SHA256)
		}
		if !bytes.Equal(b.CompiledBytes, baseBundle.CompiledBytes) {
			t.Fatalf("iter %d CompiledBytes mismatch", iter)
		}
	}
}

func TestDNS01_BundleVerification(t *testing.T) {
	vf := loadVectors(t)
	validFrom := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	validUntil := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	opts := CompileOptions{
		ValidFrom:   validFrom,
		ValidUntil:  validUntil,
		UpstreamDNS: []string{"127.0.0.1:15353"},
	}
	bundle, err := CompileBundle(vf.EvaluationProfile, opts)
	if err != nil {
		t.Fatalf("CompileBundle: %v", err)
	}

	// Expired check (half-open interval [ValidFrom, ValidUntil): ValidUntil is expired)
	if err := VerifyBundle(bundle, validUntil); ErrorCode(err) != ErrCodeBundleExpired {
		t.Errorf("VerifyBundle at ValidUntil got %v, want %s", err, ErrCodeBundleExpired)
	}

	// Hash tampering check
	tamperedHash := *bundle
	tamperedHash.SHA256 = strings.Repeat("0", 64)
	if err := VerifyBundle(&tamperedHash, validFrom); ErrorCode(err) != ErrCodeBundleHashMismatch {
		t.Errorf("VerifyBundle tampered hash got %v, want %s", err, ErrCodeBundleHashMismatch)
	}

	// Privacy violation check
	badPrivacy := *bundle
	badPrivacy.Settings.QueryLogEnabled = true
	if err := VerifyBundle(&badPrivacy, validFrom); ErrorCode(err) != ErrCodeBundlePrivacyInvalid {
		t.Errorf("VerifyBundle bad privacy got %v, want %s", err, ErrCodeBundlePrivacyInvalid)
	}

	// Syntax violation check (allow rule with $important)
	if err := ValidateAdGuardRuleSyntax("@@||example.com^$important"); ErrorCode(err) != ErrCodeBundleSyntaxInvalid {
		t.Errorf("ValidateAdGuardRuleSyntax(@@...$important) got %v, want %s", err, ErrCodeBundleSyntaxInvalid)
	}
}

// mockAdGuardServer simulates an AdGuard Home HTTP API for unit testing the
// publish flow, idempotency, and partial rollback.
type mockAdGuardServer struct {
	mu                sync.Mutex
	userRules         []string
	queryLogEnabled   bool
	statisticsEnabled bool
	upstreamDNS       []string
	allowedClients    []string
	blockingMode      string
	blockedTTL        uint32
	setRulesCount     int
}

func newMockAdGuardServer(t *testing.T) (*mockAdGuardServer, *httptest.Server) {
	t.Helper()
	m := &mockAdGuardServer{
		upstreamDNS:    []string{"127.0.0.1:15353"},
		allowedClients: []string{"127.0.0.0/8"},
		blockingMode:   "nxdomain",
		blockedTTL:     60,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/control/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":            PinnedAdGuardVersion,
			"protection_enabled": true,
			"running":            true,
		})
	})
	mux.HandleFunc("/control/dns_info", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"upstream_dns":         m.upstreamDNS,
			"fallback_dns":         []string{},
			"blocking_mode":        m.blockingMode,
			"blocked_response_ttl": m.blockedTTL,
			"protection_enabled":   true,
		})
	})
	mux.HandleFunc("/control/dns_config", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			UpstreamDNS        []string `json:"upstream_dns"`
			BlockingMode       string   `json:"blocking_mode"`
			BlockedResponseTTL uint32   `json:"blocked_response_ttl"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		m.upstreamDNS = body.UpstreamDNS
		m.blockingMode = body.BlockingMode
		m.blockedTTL = body.BlockedResponseTTL
		m.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/control/filtering/status", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"enabled":    true,
			"filters":    []any{},
			"user_rules": m.userRules,
		})
	})
	mux.HandleFunc("/control/filtering/config", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/control/filtering/set_rules", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Rules []string `json:"rules"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		m.userRules = body.Rules
		m.setRulesCount++
		m.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/control/filtering/check_host", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("name")
		m.mu.Lock()
		rules := append([]string(nil), m.userRules...)
		m.mu.Unlock()

		// Evaluate using our rule lines in precedence order ($important, @@, normal).
		for _, line := range rules {
			if strings.HasSuffix(line, "$important") {
				dom := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(line, "||"), "|"), "^$important")
				incSub := strings.HasPrefix(line, "||")
				if MatchesDomain(dom, incSub, q) {
					_ = json.NewEncoder(w).Encode(CheckHostResponse{Reason: "FilteredBlackList", Rule: line})
					return
				}
			}
		}
		for _, line := range rules {
			if strings.HasPrefix(line, "@@") {
				rest := strings.TrimPrefix(line, "@@")
				dom := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(rest, "||"), "|"), "^")
				incSub := strings.HasPrefix(rest, "||")
				if MatchesDomain(dom, incSub, q) {
					_ = json.NewEncoder(w).Encode(CheckHostResponse{Reason: "NotFilteredWhiteList", Rule: line})
					return
				}
			}
		}
		for _, line := range rules {
			if !strings.HasPrefix(line, "!") && !strings.HasPrefix(line, "@@") && !strings.HasSuffix(line, "$important") {
				dom := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(line, "||"), "|"), "^")
				incSub := strings.HasPrefix(line, "||")
				if MatchesDomain(dom, incSub, q) {
					_ = json.NewEncoder(w).Encode(CheckHostResponse{Reason: "FilteredBlackList", Rule: line})
					return
				}
			}
		}
		_ = json.NewEncoder(w).Encode(CheckHostResponse{Reason: "NotFilteredNotFound"})
	})
	mux.HandleFunc("/control/querylog/config", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"enabled": m.queryLogEnabled})
	})
	mux.HandleFunc("/control/querylog/config/update", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		m.queryLogEnabled = body.Enabled
		m.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/control/stats/config", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"enabled": m.statisticsEnabled})
	})
	mux.HandleFunc("/control/stats/config/update", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		m.statisticsEnabled = body.Enabled
		m.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/control/access/list", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"allowed_clients": m.allowedClients})
	})
	mux.HandleFunc("/control/access/set", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			AllowedClients []string `json:"allowed_clients"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		m.allowedClients = body.AllowedClients
		m.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/control/cache_clear", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return m, srv
}

func TestDNS01_AT21_PublishIdempotencyAndRollbackUnit(t *testing.T) {
	vf := loadVectors(t)
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	opts := CompileOptions{
		ValidFrom:   now.Add(-time.Hour),
		ValidUntil:  now.Add(24 * time.Hour),
		UpstreamDNS: []string{"127.0.0.1:15353"},
	}

	bundleRev12, err := CompileBundle(vf.EvaluationProfile, opts)
	if err != nil {
		t.Fatalf("CompileBundle rev 12: %v", err)
	}

	m1, srv1 := newMockAdGuardServer(t)
	_, srv2 := newMockAdGuardServer(t)

	ad1 := NewAdGuardAdapter(ResolverConfig{ID: "resolver-1", AdminURL: srv1.URL, Username: "admin", Password: "pw"})
	ad2 := NewAdGuardAdapter(ResolverConfig{ID: "resolver-2", AdminURL: srv2.URL, Username: "admin", Password: "pw"})
	ad1.SetNowFunc(func() time.Time { return now })
	ad2.SetNowFunc(func() time.Time { return now })

	pub := NewMultiResolverPublisher(ad1, ad2)
	ctx := context.Background()

	// 1. Initial publish to both resolvers -> all_applied.
	rep1, err := pub.Publish(ctx, bundleRev12)
	if err != nil {
		t.Fatalf("initial Publish failed: %v", err)
	}
	if rep1.Status != PublishAllApplied || !rep1.AllSucceeded {
		t.Fatalf("got status %s, want %s", rep1.Status, PublishAllApplied)
	}
	for _, r := range rep1.Results {
		if r.AppliedRevision != "12" || r.NoOp {
			t.Errorf("unexpected result: %+v", r)
		}
	}

	// 2. Re-publishing the exact same revision is an idempotent no-op.
	m1.mu.Lock()
	callsBefore := m1.setRulesCount
	m1.mu.Unlock()

	rep2, err := pub.Publish(ctx, bundleRev12)
	if err != nil {
		t.Fatalf("idempotent Publish failed: %v", err)
	}
	if rep2.Status != PublishAllApplied {
		t.Fatalf("got status %s, want %s", rep2.Status, PublishAllApplied)
	}
	for _, r := range rep2.Results {
		if !r.NoOp || r.AppliedRevision != "12" {
			t.Errorf("expected NoOp=true and AppliedRevision=12, got %+v", r)
		}
	}
	m1.mu.Lock()
	callsAfter := m1.setRulesCount
	m1.mu.Unlock()
	if callsAfter != callsBefore {
		t.Errorf("expected 0 set_rules calls on idempotent re-publish, got %d", callsAfter-callsBefore)
	}

	// 3. Compile revision 13, but inject a failure on resolver-2 so resolver-1
	// applies rev 13 while resolver-2 rolls back to rev 12 -> status is partial.
	prof13 := vf.EvaluationProfile
	prof13.Revision = "13"
	bundleRev13, err := CompileBundle(prof13, opts)
	if err != nil {
		t.Fatalf("CompileBundle rev 13: %v", err)
	}

	ad2.SetFaultHook(func(ctx context.Context, resolverID string, b *CompiledBundle) error {
		return errors.New("injected smoke test failure on resolver-2")
	})
	defer ad2.SetFaultHook(nil)

	rep3, err := pub.Publish(ctx, bundleRev13)
	if err == nil {
		t.Fatal("expected error from partial publish")
	}
	if rep3 == nil || rep3.Status != PublishPartial || rep3.AllSucceeded {
		t.Fatalf("expected status=%s and AllSucceeded=false, got %+v", PublishPartial, rep3)
	}
	if rep3.Results[0].AppliedRevision != "13" || !rep3.Results[0].Success {
		t.Errorf("resolver-1 expected AppliedRevision=13 Success=true, got %+v", rep3.Results[0])
	}
	if rep3.Results[1].AppliedRevision != "12" || rep3.Results[1].Success || !rep3.Results[1].RolledBack {
		t.Errorf("resolver-2 expected AppliedRevision=12 (rolled back to last_good), got %+v", rep3.Results[1])
	}

	// Verify live state on resolver-2 was actually restored to revision 12.
	st2, err := ad2.GetAppliedState(ctx)
	if err != nil {
		t.Fatalf("ad2.GetAppliedState: %v", err)
	}
	if st2.AppliedRevision != "12" {
		t.Errorf("resolver-2 live AppliedRevision = %q, want 12", st2.AppliedRevision)
	}
}

func TestDNS01_AT19_CompiledBundleJSONRoundTripAndPublish(t *testing.T) {
	vf := loadVectors(t)
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	opts := CompileOptions{
		ValidFrom:   now.Add(-time.Hour),
		ValidUntil:  now.Add(24 * time.Hour),
		UpstreamDNS: []string{"127.0.0.1:15353"},
	}

	compiled, err := CompileBundle(vf.EvaluationProfile, opts)
	if err != nil {
		t.Fatalf("CompileBundle: %v", err)
	}

	// Verify CompiledBytes is valid RFC 8785 canonical JSON.
	canonicalized, err := canonjson.Canonicalize(compiled.CompiledBytes)
	if err != nil {
		t.Fatalf("canonjson.Canonicalize(CompiledBytes): %v", err)
	}
	if !bytes.Equal(compiled.CompiledBytes, canonicalized) {
		t.Fatalf("CompiledBytes is not in RFC 8785 canonical form")
	}

	// 1. Round-trip via encoding/json.Marshal -> json.Unmarshal
	rawStd, err := json.Marshal(compiled)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var loadedStd CompiledBundle
	if err := json.Unmarshal(rawStd, &loadedStd); err != nil {
		t.Fatalf("json.Unmarshal(rawStd): %v", err)
	}
	if !bytes.Equal(loadedStd.CompiledBytes, compiled.CompiledBytes) {
		t.Fatalf("loadedStd.CompiledBytes mismatch after json.Marshal round-trip")
	}
	if err := VerifyBundle(&loadedStd, now); err != nil {
		t.Fatalf("VerifyBundle(&loadedStd) failed after json.Marshal round-trip: %v", err)
	}

	// 2. Round-trip via canonjson.Marshal -> json.Unmarshal (as dns_compiled_bundles stores it)
	rawCanon, err := canonjson.Marshal(compiled)
	if err != nil {
		t.Fatalf("canonjson.Marshal: %v", err)
	}
	var loadedDB CompiledBundle
	if err := json.Unmarshal(rawCanon, &loadedDB); err != nil {
		t.Fatalf("json.Unmarshal(rawCanon): %v", err)
	}
	if !bytes.Equal(loadedDB.CompiledBytes, compiled.CompiledBytes) {
		t.Fatalf("loadedDB.CompiledBytes mismatch after canonjson.Marshal round-trip")
	}
	if err := VerifyBundle(&loadedDB, now); err != nil {
		t.Fatalf("VerifyBundle(&loadedDB) failed after canonjson.Marshal round-trip: %v", err)
	}

	// 3. AdGuardAdapter.Publish accepts the unmarshaled bundle.
	_, srv := newMockAdGuardServer(t)
	adapter := NewAdGuardAdapter(ResolverConfig{
		ID:       "resolver-roundtrip",
		AdminURL: srv.URL,
		Username: "admin",
		Password: "pw",
	})
	adapter.SetNowFunc(func() time.Time { return now })

	pubRes, err := adapter.Publish(context.Background(), &loadedDB)
	if err != nil {
		t.Fatalf("adapter.Publish(&loadedDB) failed: %v", err)
	}
	if !pubRes.Success || pubRes.AppliedRevision != compiled.Revision || pubRes.AppliedSHA256 != compiled.SHA256 {
		t.Fatalf("unexpected publish result for unmarshaled bundle: %+v", pubRes)
	}

	// 4. Tampering with a field in the stored JSON before Unmarshal causes VerifyBundle to fail with BUNDLE_HASH_MISMATCH.
	tamperedJSON := bytes.Replace(rawCanon, []byte(`"revision":"12"`), []byte(`"revision":"99"`), 1)
	var tamperedLoaded CompiledBundle
	if err := json.Unmarshal(tamperedJSON, &tamperedLoaded); err != nil {
		t.Fatalf("json.Unmarshal(tamperedJSON): %v", err)
	}
	if err := VerifyBundle(&tamperedLoaded, now); ErrorCode(err) != ErrCodeBundleHashMismatch {
		t.Fatalf("VerifyBundle(&tamperedLoaded) = %v, want %s", err, ErrCodeBundleHashMismatch)
	}
}

func TestDNS01_AT22_SourceFetchSSRF_RedirectAndResolvedIPBlocked(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

	newSeededRegistry := func(t *testing.T, urlStr string) *SourceStore {
		t.Helper()
		reg := NewDefaultSourceStore()
		if err := reg.Register(ListSource{
			SourceID:          "src-ssrf-test",
			Version:           "1",
			License:           "CC0-1.0",
			LicenseStatus:     LicenseCommercialApproved,
			PermittedUse:      "TEST ONLY commercial simulation",
			DistributionNotes: "Server-side resolver compilation",
			AttributionNotes:  new("Test Feed"),
			URL:               urlStr,
			Format:            SourceFormatPlain,
			Category:          CategoryMalware,
		}); err != nil {
			t.Fatalf("Register(%q) failed: %v", urlStr, err)
		}
		if err := reg.UpdateFromReader("src-ssrf-test", "1", strings.NewReader("initial-good.example.com\n"), DefaultParseOptions(), t0, nil); err != nil {
			t.Fatalf("seed UpdateFromReader: %v", err)
		}
		return reg
	}

	// 1. Hostname that resolves to any blocked IP range is refused BEFORE connect (dialCalls == 0).
	blockedCases := []struct {
		name string
		ips  []netip.Addr
	}{
		{"loopback_v4", []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
		{"loopback_v6", []netip.Addr{netip.MustParseAddr("::1")}},
		{"loopback_v4_mapped_v6", []netip.Addr{netip.MustParseAddr("::ffff:127.0.0.1")}},
		{"rfc1918_10", []netip.Addr{netip.MustParseAddr("10.0.0.42")}},
		{"rfc1918_172_16", []netip.Addr{netip.MustParseAddr("172.16.5.4")}},
		{"rfc1918_192_168", []netip.Addr{netip.MustParseAddr("192.168.1.10")}},
		{"cgnat_100_64", []netip.Addr{netip.MustParseAddr("100.64.0.1")}},
		{"link_local_metadata_v4", []netip.Addr{netip.MustParseAddr("169.254.169.254")}},
		{"link_local_v6", []netip.Addr{netip.MustParseAddr("fe80::1")}},
		{"multicast_v4", []netip.Addr{netip.MustParseAddr("224.0.0.1")}},
		{"multicast_v6", []netip.Addr{netip.MustParseAddr("ff02::1")}},
		{"unspecified_v4", []netip.Addr{netip.MustParseAddr("0.0.0.0")}},
		{"unspecified_v6", []netip.Addr{netip.MustParseAddr("::")}},
		{"ula_v6", []netip.Addr{netip.MustParseAddr("fd12:3456:789a::1")}},
		{"doc_test_net_1", []netip.Addr{netip.MustParseAddr("192.0.2.10")}},
		{"doc_test_net_2", []netip.Addr{netip.MustParseAddr("198.51.100.10")}},
		{"doc_test_net_3", []netip.Addr{netip.MustParseAddr("203.0.113.10")}},
		{"doc_ipv6", []netip.Addr{netip.MustParseAddr("2001:db8::10")}},
		{"mixed_public_and_blocked", []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("10.0.0.1")}},
	}

	for _, tc := range blockedCases {
		t.Run("resolved_blocked_"+tc.name, func(t *testing.T) {
			reg := newSeededRegistry(t, "https://rebind.example.com/feed.txt")
			dialCalls := 0
			reg.SetLookupNetIP(func(_ context.Context, _, _ string) ([]netip.Addr, error) {
				return tc.ips, nil
			})
			reg.SetDialContext(func(_ context.Context, _, addr string) (net.Conn, error) {
				dialCalls++
				return nil, errors.New("unexpected dial to " + addr)
			})

			err := reg.FetchAndUpdate(ctx, nil, "src-ssrf-test", "2", DefaultParseOptions(), t0.Add(time.Minute), nil)
			if err == nil {
				t.Fatalf("expected error when hostname resolves to blocked IP %v, got nil", tc.ips)
			}
			if ErrorCode(err) != ErrCodeSourceInvalidURL {
				t.Errorf("ErrorCode(err) = %q, want %q (err=%v)", ErrorCode(err), ErrCodeSourceInvalidURL, err)
			}
			if dialCalls != 0 {
				t.Errorf("expected 0 dial calls (must refuse before connect), got %d", dialCalls)
			}
			src, _ := reg.Get("src-ssrf-test")
			if src.Version != "1" || len(src.Entries) != 1 || src.Entries[0].Domain != "initial-good.example.com" {
				t.Errorf("last_good was not preserved after blocked DNS resolution: %+v", src)
			}
		})
	}

	// 2. HTTP redirect to a blocked target (or any redirect) is refused.
	t.Run("redirect_to_blocked_target_refused", func(t *testing.T) {
		redirectHits := 0
		tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			redirectHits++
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
		}))
		t.Cleanup(tlsSrv.Close)

		srvListenerAddr := tlsSrv.Listener.Addr().String()
		reg := newSeededRegistry(t, "https://feeds.example.com/list.txt")

		var dialedAddresses []string
		reg.SetLookupNetIP(func(_ context.Context, _, host string) ([]netip.Addr, error) {
			if host == "feeds.example.com" {
				// Public routable IP (non-reserved, non-documentation)
				return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("169.254.169.254")}, nil
		})
		reg.SetDialContext(func(ctx context.Context, network, address string) (net.Conn, error) {
			dialedAddresses = append(dialedAddresses, address)
			return (&net.Dialer{}).DialContext(ctx, network, srvListenerAddr)
		})

		err := reg.FetchAndUpdate(ctx, tlsSrv.Client(), "src-ssrf-test", "2", DefaultParseOptions(), t0.Add(2*time.Minute), nil)
		if err == nil {
			t.Fatal("expected redirect to blocked target to be refused, got nil")
		}
		if ErrorCode(err) != ErrCodeSourceInvalidURL {
			t.Errorf("ErrorCode(err) = %q, want %q (err=%v)", ErrorCode(err), ErrCodeSourceInvalidURL, err)
		}
		if redirectHits != 1 {
			t.Errorf("redirectHits = %d, want 1", redirectHits)
		}
		if len(dialedAddresses) != 1 || dialedAddresses[0] != "93.184.216.34:443" {
			t.Errorf("dialedAddresses = %v, want [\"93.184.216.34:443\"]", dialedAddresses)
		}
		src, _ := reg.Get("src-ssrf-test")
		if src.Version != "1" || len(src.Entries) != 1 || src.Entries[0].Domain != "initial-good.example.com" {
			t.Errorf("last_good was not preserved after refused redirect: %+v", src)
		}
	})

	// 3. Valid public resolution dials the verified IP and enforces the body size limit.
	t.Run("verified_public_ip_dials_and_caps_body", func(t *testing.T) {
		tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/encoded.txt" {
				w.Header().Set("Content-Encoding", "gzip")
				// Refuse the encoding before parsing, even with an injected client.
				_, _ = w.Write([]byte("new-bad1.example.com\n"))
				return
			}
			if r.URL.Path == "/oversized.txt" {
				_, _ = w.Write([]byte(strings.Repeat("a.example.com\n", 100)))
				return
			}
			_, _ = w.Write([]byte("new-bad1.example.com\nnew-bad2.example.com\n"))
		}))
		t.Cleanup(tlsSrv.Close)

		srvListenerAddr := tlsSrv.Listener.Addr().String()
		reg := newSeededRegistry(t, "https://feeds.example.com/oversized.txt")

		var dialedAddr string
		reg.SetLookupNetIP(func(_ context.Context, _, _ string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
		})
		reg.SetDialContext(func(ctx context.Context, network, address string) (net.Conn, error) {
			dialedAddr = address
			return (&net.Dialer{}).DialContext(ctx, network, srvListenerAddr)
		})

		// Oversized response body is rejected and retains LastGood.
		err := reg.FetchAndUpdate(ctx, tlsSrv.Client(), "src-ssrf-test", "2", ParseOptions{MaxBytes: 32, RequireNonEmpty: true}, t0.Add(3*time.Minute), nil)
		if ErrorCode(err) != ErrCodeSourceTooLarge {
			t.Fatalf("expected %s on oversized fetch, got %v", ErrCodeSourceTooLarge, err)
		}
		if dialedAddr != "93.184.216.34:443" {
			t.Errorf("dialedAddr = %q, want 93.184.216.34:443", dialedAddr)
		}

		// Valid response body updates the source.
		src, _ := reg.Get("src-ssrf-test")
		src.URL = "https://feeds.example.com/valid.txt"
		if err := reg.Register(src); err != nil {
			t.Fatalf("re-register valid URL: %v", err)
		}
		if err := reg.FetchAndUpdate(ctx, tlsSrv.Client(), "src-ssrf-test", "2", DefaultParseOptions(), t0.Add(4*time.Minute), nil); err != nil {
			t.Fatalf("valid FetchAndUpdate failed: %v", err)
		}
		updated, _ := reg.Get("src-ssrf-test")
		if updated.Version != "2" || len(updated.Entries) != 2 {
			t.Fatalf("unexpected updated source state: %+v", updated)
		}
		updated.URL = "https://feeds.example.com/encoded.txt"
		if err := reg.Register(updated); err != nil {
			t.Fatal(err)
		}
		if err := reg.FetchAndUpdate(ctx, tlsSrv.Client(), "src-ssrf-test", "3", DefaultParseOptions(), t0.Add(5*time.Minute), nil); ErrorCode(err) != ErrCodeSourceFormatUnsupported {
			t.Fatalf("encoded response accepted: %v", err)
		}
		retained, _ := reg.Get("src-ssrf-test")
		if retained.Version != "2" || retained.ContentSHA256 != updated.ContentSHA256 {
			t.Fatal("encoded response replaced last-good")
		}
	})
}
