// Package issuer defines the shared D-052 OIDC issuer representation.
package issuer

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrCanonical = errors.New("oidc_issuer must be a canonical HTTP(S) issuer")

// Canonical preserves the issuer path but normalizes scheme, host and default
// port, and trims only literal trailing slashes. Encoded bytes, including a
// trailing %2F, stay as written. Query, fragment, credentials and opaque URLs
// are invalid. Auth and discovery must use this same representation (D-052/F-09).
func Canonical(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, "?#") || u.User != nil || u.Opaque != "" || u.Hostname() == "" || strings.HasSuffix(u.Host, ":") {
		return "", ErrCanonical
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", ErrCanonical
	}
	host, port := strings.ToLower(u.Hostname()), u.Port()
	if u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	u.Host = host
	// Path is stored decoded, so TrimRight on it would also delete a trailing
	// %2F. Remove only literal '/' bytes from the escaped path.
	escaped := u.EscapedPath()
	trimmed := strings.TrimRight(escaped, "/")
	if trimmed != escaped {
		if trimmed == "" {
			u.Path, u.RawPath = "", ""
		} else {
			n := len(escaped) - len(trimmed)
			if len(u.Path) < n || !strings.HasSuffix(u.Path, strings.Repeat("/", n)) {
				return "", ErrCanonical
			}
			u.Path = u.Path[:len(u.Path)-n]
			u.RawPath = trimmed
		}
	}
	return u.String(), nil
}

// Validate rejects aliases at configuration load; it never silently points
// token validation at a different issuer than discovery.
func Validate(raw string) error {
	canonical, err := Canonical(raw)
	if err != nil || canonical != raw {
		return ErrCanonical
	}
	return nil
}
