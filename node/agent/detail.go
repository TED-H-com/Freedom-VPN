package agent

import (
	"regexp"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

const maxCheckDetail = 500

var (
	reDetailURL   = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://\S+`)
	reDetailToken = regexp.MustCompile(`(?i)\b(token|bearer|key|secret|password|setup_key)\s*[=:]\s*\S+`)
	reDetailCN    = regexp.MustCompile(`(?i)\b(CN|subject|O|OU)\s*=\s*\S+`)
	reDetailEmail = regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)
	reDetailIPv4  = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	reDetailIPv6  = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{0,4}:){2,7}[0-9a-f]{0,4}\b`)
	reDetailHex   = regexp.MustCompile(`\b[0-9a-fA-F]{32,}\b`)
)

// sanitizeDetail returns an operator-facing check detail. It drops URLs,
// tokens, certificate subjects, email addresses, IP addresses and long hex
// strings, then caps the result at 500 characters (D-058).
func sanitizeDetail(s string) *string {
	s = strings.TrimSpace(s)
	s = reDetailURL.ReplaceAllString(s, "[redacted]")
	s = reDetailToken.ReplaceAllString(s, "[redacted]")
	s = reDetailCN.ReplaceAllString(s, "[redacted]")
	s = reDetailEmail.ReplaceAllString(s, "[redacted]")
	s = reDetailIPv4.ReplaceAllString(s, "[redacted]")
	s = reDetailIPv6.ReplaceAllString(s, "[redacted]")
	s = reDetailHex.ReplaceAllString(s, "[redacted]")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		s = "Check failed."
	}
	if len(s) > maxCheckDetail {
		s = s[:maxCheckDetail]
		for len(s) > 0 && s[len(s)-1]&0xC0 == 0x80 {
			s = s[:len(s)-1]
		}
	}
	return &s
}

// withDetail attaches a sanitized detail to a failed check. A tunnel check
// that ran also keeps its source sentence when it passed. not_run stays empty.
func withDetail(c api.NodeCheck, detail string) api.NodeCheck {
	if c.Status == api.HealthCheckStatusNotRun {
		return c
	}
	if c.Status != api.HealthCheckStatusFailed && c.Name != api.NodeCheckNameTunnelProbe {
		return c
	}
	if strings.TrimSpace(detail) == "" {
		if c.Status != api.HealthCheckStatusFailed {
			return c
		}
		detail = "Check failed."
	}
	c.Detail = sanitizeDetail(detail)
	return c
}
