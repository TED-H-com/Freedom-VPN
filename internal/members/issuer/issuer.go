// Package issuer defines the shared D-052 OIDC issuer representation.
package issuer

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrCanonical = errors.New("oidc_issuer must be a canonical HTTP(S) issuer")

// Canonical preserves the issuer path but normalizes scheme, host, default port
// and trailing slashes. Query, fragment, credentials and opaque URLs are invalid.
// Auth and discovery must use this same representation (D-052/F-09).
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
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
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
