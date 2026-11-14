package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
)

func TestAT13_LabControlDurationIntegerBounds(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow examples require 64-bit int")
	}
	var srv fakecontrol.Server
	paths := []struct {
		name, errorBody string
		handler         http.HandlerFunc
	}{
		{"service", "seconds\n", serviceHandler(&srv)},
		{"usage_retry", "n\n", usageRetryHandler(&srv)},
	}
	// 18446744074 * 10^9 = 2^64 + 290448384 ns.
	for _, path := range paths {
		for _, tc := range []struct {
			name, seconds string
			status        int
		}{
			{"maximum", "9223372036", http.StatusNoContent},
			{"above_maximum", "9223372037", http.StatusBadRequest},
			{"positive_wrap", "18446744074", http.StatusBadRequest},
		} {
			t.Run(path.name+"/"+tc.name, func(t *testing.T) {
				body := fmt.Sprintf(`{"n":1,"seconds":%s}`, tc.seconds)
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
				w := httptest.NewRecorder()
				path.handler(w, req)
				if w.Code != tc.status {
					t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
				}
				if tc.status == http.StatusBadRequest && w.Body.String() != path.errorBody {
					t.Fatalf("error body = %q", w.Body.String())
				}
			})
		}
	}
}
