package usageupload

import (
	"strconv"
	"testing"
	"time"
)

func TestMTR03_RetryAfterDurationIntegerBounds(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow examples require 64-bit int")
	}
	// 18446744074 * 10^9 = 2^64 + 290448384 ns.
	for _, tc := range []struct {
		name, header string
		want         time.Duration
	}{
		{"maximum", "9223372036", 9223372036 * time.Second},
		{"above_maximum", "9223372037", 0},
		{"positive_wrap", "18446744074", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseRetryAfter(tc.header, time.Unix(0, 0)); got != tc.want {
				t.Fatalf("parseRetryAfter(%q) = %v, want %v", tc.header, got, tc.want)
			}
		})
	}
}
