//go:build linux

package timeanchor

import (
	"testing"
	"time"
)

func TestAT13_MonotonicDurationIntegerBounds(t *testing.T) {
	// 18446744074 * 10^9 = 2^64 + 290448384 ns.
	for _, tc := range []struct {
		name          string
		seconds, nano int64
		want          time.Duration
		ok            bool
	}{
		{"maximum", 9223372036, 854775807, 1<<63 - 1, true},
		{"above_maximum", 9223372036, 854775808, 0, false},
		{"minimum", -9223372036, -854775808, -1 << 63, true},
		{"below_minimum", -9223372036, -854775809, 0, false},
		{"seconds_above_maximum", 9223372037, 0, 0, false},
		{"positive_wrap", 18446744074, 0, 0, false},
		{"negative_wrap", -18446744073, 0, 0, false},
		{"representable_extra_nanoseconds", 1, 1000000000, 2 * time.Second, true},
		{"representable_negative", -1, 0, -time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := MonotonicDuration(tc.seconds, tc.nano)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("MonotonicDuration(%d, %d) = %v, %v; want %v, %v", tc.seconds, tc.nano, got, ok, tc.want, tc.ok)
			}
		})
	}
}
