package apierr

import (
	"net/http/httptest"
	"testing"
)

func TestAT07_AT28_ExtensionErrorCodes(t *testing.T) {
	for _, tc := range []struct {
		code      Code
		status    int
		retryable bool
	}{
		{BootstrapConsumed, 410, false}, {BootstrapNotReady, 409, true}, {BootstrapExpired, 410, false}, {NodeIncarnationActive, 409, false}, {NodeRetired, 409, false},
	} {
		rec := httptest.NewRecorder()
		Write(rec, nil, New(tc.code, "test"))
		body := decode(t, rec).Error
		if rec.Code != tc.status || body.Retryable != tc.retryable || body.Code != tc.code || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %+v", tc.code, rec.Code, body)
		}
	}
}
