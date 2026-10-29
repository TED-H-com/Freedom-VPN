package dnspolicy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

const (
	// MaxLabelLength is the maximum length of a single DNS label in bytes (RFC 1035 §2.3.4).
	MaxLabelLength = 63
	// MaxDomainLength is the maximum length of a canonical ASCII domain name
	// without trailing root dot in bytes (RFC 1035 §2.3.4: 255 wire octets = 253 text bytes).
	MaxDomainLength = 253
)

// idnaLookupProfile and idnaRegistrationProfile implement a two-stage strict
// IDNA2008 non-transitional check (RFC 5891 §§4–5, RFC 5892, RFC 5893, UTS #46)
// for domain normalization (§12.4, DNS02).
//
// Why this profile combination is chosen:
//  1. idnaLookupProfile uses idna.MapForLookup() + idna.Transitional(false) to
//     perform NFC normalization and Unicode case-folding so uppercase inputs
//     (e.g. "MÜNCHEN.DE") map deterministically to canonical A-labels while
//     preserving IDNA2008 non-transitional characters such as "ß" ("faß.de" ->
//     "xn--fa-hia.de").
//  2. idnaRegistrationProfile uses idna.ValidateForRegistration() +
//     idna.VerifyDNSLength(false) on the lowercased A-label output to enforce
//     strict RFC 5891 §4 / RFC 5892 IDNA2008 code-point rules (rejecting emojis,
//     symbols, and unassigned code points that UTS #46 lookup mapping alone
//     would tolerate).
//  3. Both profiles enforce StrictDomainName(true), ValidateLabels(true),
//     CheckHyphens(true), CheckJoiners(true), and BidiRule().
//  4. DNS length checks (empty label, >63-byte label, >253-byte name) are executed
//     explicitly after IDNA validation so callers receive distinct UI error codes
//     (DOMAIN_EMPTY_LABEL, DOMAIN_LABEL_TOO_LONG, DOMAIN_TOO_LONG).
var (
	idnaLookupProfile = idna.New(
		idna.MapForLookup(),
		idna.Transitional(false),
		idna.StrictDomainName(true),
		idna.ValidateLabels(true),
		idna.CheckHyphens(true),
		idna.CheckJoiners(true),
		idna.BidiRule(),
	)
	idnaRegistrationProfile = idna.New(
		idna.ValidateForRegistration(),
		idna.Transitional(false),
		idna.StrictDomainName(true),
		idna.ValidateLabels(true),
		idna.CheckHyphens(true),
		idna.CheckJoiners(true),
		idna.BidiRule(),
		idna.VerifyDNSLength(false),
	)
)

// IDNAProfileDescription returns a human-readable summary of the IDNA profile
// and rationale used by NormalizeDomain.
func IDNAProfileDescription() string {
	return "IDNA2008 non-transitional lookup + RFC 5891 registration profile (UTS #46 MapForLookup + ValidateForRegistration, Transitional=false, StrictDomainName=true, ValidateLabels=true, CheckHyphens=true, CheckJoiners=true, BidiRule=true)"
}

// NormalizedDomain holds both the canonical ASCII (A-label) domain name used
// for storage and matching and the original user input preserved for display (§12.4).
type NormalizedDomain struct {
	Canonical string `json:"canonical"`
	Original  string `json:"original"`
}

var knownSchemes = []string{
	"http:", "https:", "ftp:", "ftps:", "ws:", "wss:",
	"dns:", "doh:", "dot:", "doq:", "mailto:", "file:",
	"data:", "javascript:", "urn:", "ssh:", "tcp:", "udp:",
}

// NormalizeDomain validates and normalizes a domain name according to §12.4:
//   - Accepts only domain names; rejects scheme, user info, port, path, query,
//     fragment, wildcards, control characters, empty labels, labels > 63 bytes,
//     names > 253 bytes, and empty input.
//   - Strips at most one trailing root dot, runs the IDNA2008 non-transitional
//     check, lowercases, and returns the canonical ASCII (A-label) form while
//     keeping the original input for UI display.
func NormalizeDomain(raw string) (NormalizedDomain, error) {
	if raw == "" {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainEmpty, raw, "domain name must not be empty", nil)
	}
	if !utf8.ValidString(raw) {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, "domain name contains invalid UTF-8 bytes", nil)
	}

	// 1. Check for control characters and invisible formatting characters (except
	// ZWNJ \u200c and ZWJ \u200d which are validated by idna.CheckJoiners).
	for _, r := range raw {
		if r <= 0x1F || r == 0x7F || unicode.IsControl(r) || r == '\u00ad' {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainControlChar, raw, "domain name must not contain control characters", nil)
		}
		if unicode.Is(unicode.Cf, r) && r != '\u200c' && r != '\u200d' {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainControlChar, raw, "domain name must not contain invisible format characters", nil)
		}
	}

	s := raw

	// 2. Reject URL schemes before checking port/path so "https://example.com"
	// reports DOMAIN_HAS_SCHEME.
	lowerRaw := strings.ToLower(s)
	if strings.Contains(lowerRaw, "://") {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasScheme, raw, "domain name must not include a URL scheme", nil)
	}
	for _, scheme := range knownSchemes {
		if strings.HasPrefix(lowerRaw, scheme) {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainHasScheme, raw, "domain name must not include a URL scheme", nil)
		}
	}

	// 3. Reject userinfo (@) before port (:) so "user:pass@example.com" reports
	// DOMAIN_HAS_USERINFO.
	if strings.ContainsRune(s, '@') {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasUserInfo, raw, "domain name must not include user info ('@')", nil)
	}

	// 4. Reject wildcards (*).
	if strings.ContainsRune(s, '*') {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasWildcard, raw, "domain name must not include wildcards ('*'); use include_subdomains instead", nil)
	}

	// 5. Reject URL fragment (#), query (?), path (/ or \), and port (:).
	if strings.ContainsRune(s, '#') {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasFragment, raw, "domain name must not include a URL fragment ('#')", nil)
	}
	if strings.ContainsRune(s, '?') {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasQuery, raw, "domain name must not include a URL query ('?')", nil)
	}
	if strings.ContainsAny(s, "/\\") {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasPath, raw, "domain name must not include a URL path ('/' or '\\')", nil)
	}
	if strings.ContainsRune(s, ':') {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainHasPort, raw, "domain name must not include a port or colon (':')", nil)
	}

	// 6. Strip at most one trailing root dot (".").
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainEmpty, raw, "domain name must not be empty or a bare root dot", nil)
	}

	// 7. Reject non-ASCII fullwidth/ideographic dot lookalikes so only ASCII '.'
	// acts as a DNS label separator.
	for _, r := range s {
		if r == '\u3002' || r == '\uff0e' || r == '\uff61' {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, "domain name must use ASCII '.' as label separator", nil)
		}
		if unicode.IsSpace(r) {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, "domain name must not contain whitespace", nil)
		}
	}

	// 8. Check for empty labels before IDNA conversion (leading dot, trailing dot
	// after stripping one root dot, or consecutive dots "..").
	if strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") || strings.Contains(s, "..") {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainEmptyLabel, raw, "domain name contains an empty label", nil)
	}
	for _, rawLabel := range strings.Split(s, ".") {
		if strings.EqualFold(rawLabel, "xn--") {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, "domain label 'xn--' has empty punycode payload", nil)
		}
	}

	// 9. Run strict IDNA2008 non-transitional lookup mapping + RFC 5891/5892
	// registration & code-point category validation, and lowercase.
	ascii, err := idnaLookupProfile.ToASCII(s)
	if err != nil {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, fmt.Sprintf("invalid domain or IDNA label: %v", err), err)
	}
	ascii = strings.ToLower(ascii)
	if _, err := idnaRegistrationProfile.ToASCII(ascii); err != nil {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, fmt.Sprintf("domain fails IDNA2008 registration check: %v", err), err)
	}
	uni, err := idnaRegistrationProfile.ToUnicode(ascii)
	if err != nil {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, fmt.Sprintf("domain fails IDNA2008 ToUnicode check: %v", err), err)
	}
	for _, r := range uni {
		if !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r) && r != '-' && r != '.' && r != '\u200c' && r != '\u200d' {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, fmt.Sprintf("domain contains disallowed IDNA2008 character %U", r), nil)
		}
	}

	// 10. Validate labels and DNS lengths on the canonical ASCII representation.
	labels := strings.Split(ascii, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainEmptyLabel, raw, "domain name contains an empty label", nil)
		}
		if len(label) > MaxLabelLength {
			return NormalizedDomain{}, newDomainError(
				ErrCodeDomainLabelTooLong,
				raw,
				fmt.Sprintf("domain label %q is %d bytes, exceeding maximum of %d bytes", label, len(label), MaxLabelLength),
				nil,
			)
		}
		if !isValidLDHLabel(label) {
			return NormalizedDomain{}, newDomainError(ErrCodeDomainInvalidIDNA, raw, fmt.Sprintf("domain label %q contains invalid characters or hyphen placement", label), nil)
		}
	}

	if len(ascii) > MaxDomainLength {
		return NormalizedDomain{}, newDomainError(
			ErrCodeDomainTooLong,
			raw,
			fmt.Sprintf("canonical domain is %d bytes, exceeding maximum of %d bytes", len(ascii), MaxDomainLength),
			nil,
		)
	}

	if len(labels) < 2 {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainSingleLabel, raw, "domain name must have at least two labels", nil)
	}

	tld := labels[len(labels)-1]
	if isAllDigits(tld) {
		return NormalizedDomain{}, newDomainError(ErrCodeDomainNumericTLD, raw, "top-level domain must not be all-numeric (IP literals are not domain names)", nil)
	}

	return NormalizedDomain{
		Canonical: ascii,
		Original:  raw,
	}, nil
}

func isValidLDHLabel(label string) bool {
	if len(label) == 0 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for i := 0; i < len(label); i++ {
		c := label[i]
		isLower := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		isHyphen := c == '-'
		if !isLower && !isDigit && !isHyphen {
			return false
		}
	}
	return true
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// ValidateDecimalRevision validates that rev is a canonical unsigned 64-bit
// decimal string matching ^(0|[1-9][0-9]*)$ and <= 18446744073709551615 (§10.4, §14.1).
func ValidateDecimalRevision(rev string) error {
	if rev == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidRevision,
			Field:   "revision",
			Input:   rev,
			Message: "revision must not be empty",
		}
	}
	if len(rev) > 20 {
		return &ValidationError{
			Code:    ErrCodeInvalidRevision,
			Field:   "revision",
			Input:   rev,
			Message: "revision exceeds 20 decimal digits",
		}
	}
	if rev != "0" {
		if rev[0] < '1' || rev[0] > '9' {
			return &ValidationError{
				Code:    ErrCodeInvalidRevision,
				Field:   "revision",
				Input:   rev,
				Message: "revision must be '0' or start with a non-zero digit",
			}
		}
		for i := 1; i < len(rev); i++ {
			if rev[i] < '0' || rev[i] > '9' {
				return &ValidationError{
					Code:    ErrCodeInvalidRevision,
					Field:   "revision",
					Input:   rev,
					Message: "revision must contain only ASCII digits",
				}
			}
		}
	}
	if _, err := strconv.ParseUint(rev, 10, 64); err != nil {
		return &ValidationError{
			Code:    ErrCodeInvalidRevision,
			Field:   "revision",
			Input:   rev,
			Message: fmt.Sprintf("revision out of uint64 range: %v", err),
			Cause:   err,
		}
	}
	return nil
}
