package dnspolicy

import (
	"strings"

	"golang.org/x/net/publicsuffix"
)

// IsReservedDomain is the shared DNS isolation baseline for admin rules,
// source URLs and feed entries. Input is canonical ASCII without a final dot.
func IsReservedDomain(domain string) bool {
	for _, suffix := range []string{"localhost", "local", "internal", "arpa", "corp", "lan", "intranet"} {
		if domain == suffix || strings.HasSuffix(domain, "."+suffix) {
			return true
		}
	}
	return false
}

// SafeFeedDomain prevents a list from blocking a whole registration boundary.
// The existing pinned x/net module supplies the offline public suffix list,
// including private suffixes. Exact hosts entries still obey local isolation.
func SafeFeedDomain(domain string, includeSubdomains bool) bool {
	if IsReservedDomain(domain) {
		return false
	}
	if includeSubdomains {
		suffix, _ := publicsuffix.PublicSuffix(domain)
		return suffix != domain
	}
	return true
}
