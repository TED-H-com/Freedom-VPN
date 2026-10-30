package policies

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestD058_FeedErrorCodesAreBoundedPublicValues(t *testing.T) {
	for code, reasons := range map[api.ErrorCode][]string{
		api.ErrorCodeDEPENDENCYUNAVAILABLE: {"fetch_failed", "http_status", "read_failed"},
		api.ErrorCodeHASHMISMATCH:          {"content_mismatch"},
		api.ErrorCodePAYLOADTOOLARGE:       {"oversize", "content_too_large"},
		api.ErrorCodeVALIDATIONFAILED:      {"invalid_config", "invalid_size", "outside_validity", "ssrf_refused", "parse_error", "zero_rules", "delta_breach"},
		api.ErrorCodeINTERNALERROR:         {"future_reason", "https://user:TEST-ONLY@example.invalid/feed", "feed.example.com\n", ""},
	} {
		for _, reason := range reasons {
			if got := feedPublicErrorCode(reason); got != code || !got.Valid() {
				t.Fatalf("public code %q, want %q", got, code)
			}
		}
	}
}
