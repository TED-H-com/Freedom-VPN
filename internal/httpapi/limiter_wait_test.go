package httpapi

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

func TestAT28_UnrepresentableWaitStaysLimited(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start
	setLimiterClock(t, func() time.Time { return now })

	if maxRateWait != 100000*time.Second {
		t.Fatalf("maxRateWait = %v, want 100000 seconds", maxRateWait)
	}
	for _, tc := range []struct {
		name string
		rate float64
	}{
		{"minimum", config.MinRatePerSecond},
		{"near_duration_limit", 1.1e-10},
		{"duration_overflow_boundary", 1.08e-10},
		{"tiny", 1e-11},
		{"smallest_positive", math.SmallestNonzeroFloat64},
		{"zero", 0},
		{"negative", -1},
		{"nan", math.NaN()},
		{"positive_infinity", math.Inf(1)},
		{"negative_infinity", math.Inf(-1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now = start
			lim := newLimiter(1)
			bucket := config.Bucket{PerSecond: tc.rate, Burst: 1}
			if got := lim.allow("client", bucket); got != 0 {
				t.Fatalf("first request: wait = %v, want 0", got)
			}
			if got := lim.allow("client", bucket); got <= 0 || got > maxRateWait || got != maxRateWait {
				t.Fatalf("second request: wait = %v, want positive capped wait %v", got, maxRateWait)
			}
		})
	}

	t.Run("unchanged_wait_and_refill", func(t *testing.T) {
		now = start
		lim := newLimiter(1)
		bucket := config.Bucket{PerSecond: 2, Burst: 1}
		if got := lim.allow("client", bucket); got != 0 {
			t.Fatalf("first request: wait = %v, want 0", got)
		}
		if got := lim.allow("client", bucket); got != 500*time.Millisecond {
			t.Fatalf("second request: wait = %v, want 500ms", got)
		}
		now = now.Add(250 * time.Millisecond)
		if got := lim.allow("client", bucket); got != 250*time.Millisecond {
			t.Fatalf("after 250ms: wait = %v, want 250ms", got)
		}
	})

	t.Run("other_valid_rates", func(t *testing.T) {
		for _, tc := range []struct {
			rate float64
			wait time.Duration
		}{
			{0.0001, 10000 * time.Second},
			{3, 333333334 * time.Nanosecond},
			{config.MaxRatePerSecond, 10 * time.Microsecond},
		} {
			now = start
			lim := newLimiter(1)
			bucket := config.Bucket{PerSecond: tc.rate, Burst: 1}
			if got := lim.allow("client", bucket); got != 0 {
				t.Errorf("rate %g first request: wait = %v, want 0", tc.rate, got)
			}
			if got := lim.allow("client", bucket); got != tc.wait {
				t.Errorf("rate %g second request: wait = %v, want %v", tc.rate, got, tc.wait)
			}
		}
	})

	t.Run("no_silent_refill", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			rate float64
		}{{"nan", math.NaN()}, {"zero", 0}} {
			now = start
			lim := newLimiter(1)
			bucket := config.Bucket{PerSecond: tc.rate, Burst: 1}
			if got := lim.allow("client", bucket); got != 0 {
				t.Errorf("%s first request: wait = %v, want 0", tc.name, got)
			}
			now = now.Add(time.Hour)
			if got := lim.allow("client", bucket); got != maxRateWait {
				t.Errorf("%s after an hour: wait = %v, want %v", tc.name, got, maxRateWait)
			}
		}
	})

	t.Run("smallest_wait_and_nan_calculation", func(t *testing.T) {
		now = start
		lim := newLimiter(1)
		bucket := config.Bucket{PerSecond: math.MaxFloat64, Burst: 1}
		lim.allow("client", bucket)
		lim.entries["client"].tokens = math.Nextafter(1, 0)
		if got := lim.allow("client", bucket); got != time.Nanosecond {
			t.Errorf("underflowing wait = %v, want 1ns", got)
		}
		if got := rateWait(math.NaN(), 1); got != maxRateWait {
			t.Errorf("NaN wait calculation = %v, want %v", got, maxRateWait)
		}
	})

	t.Run("http_retry_after", func(t *testing.T) {
		now = start
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 1e-11, Burst: 1}
		calls := 0
		h := BrowserRoutesWithMetrics(c, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(http.StatusNoContent)
		}), nil)
		request := func() *httptest.ResponseRecorder {
			r := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
			r.RemoteAddr = "192.0.2.1:1234"
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
		if w := request(); w.Code != http.StatusNoContent || calls != 1 {
			t.Fatalf("first request: status = %d, handler calls = %d", w.Code, calls)
		}
		if w := request(); w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "100000" || calls != 1 {
			t.Fatalf("second request: status = %d, Retry-After = %q, handler calls = %d", w.Code, w.Header().Get("Retry-After"), calls)
		}
	})
}
