package members

import (
	"encoding/json"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
)

// Retain the error reason, omitting arbitrary details and all response headers.
// Scrub before truncation so a truncated credential cannot evade redaction.
func memberResponseExcerpt(raw []byte, secrets ...string) string {
	var envelope struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}
	value := string(raw)
	if json.Unmarshal(raw, &envelope) == nil {
		b, _ := json.Marshal(envelope)
		value = string(b)
	}
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
	}
	value = e2e.Redact(value)
	runes := []rune(value)
	if len(runes) > 512 {
		value = string(runes[:512]) + "[truncated]"
	}
	return value
}
