package enforce

import (
	"strings"
	"testing"
	"time"
)

// TestProbeElementLifetime checks a live probe by what is left, not by the
// original kernel timeout. timeout is the installed value and does not shrink.
// expires is the remaining time. The window is probeTimeout(ExpiresAt, Now).
func TestProbeElementLifetime(t *testing.T) {
	cases := []struct {
		name    string
		timeout uint32
		expires uint32
		hasExp  bool
		window  int
		pass    bool
		listing bool
		// bound is the set-and-limit text a rejection must carry.
		bound string
	}{
		{name: "300_287_290", timeout: 300, expires: 287, hasExp: true, window: 290, pass: true},
		{name: "300_287_280", timeout: 300, expires: 287, hasExp: true, window: 280, pass: false, bound: "remaining 287s exceeds bound 281s"},
		{name: "900_100_200", timeout: 900, expires: 100, hasExp: true, window: 200, pass: false, bound: "original timeout 900s exceeds ceiling 300s"},
		{name: "45_44_45", timeout: 45, expires: 44, hasExp: true, window: 45, pass: true},
		{name: "45_45_44", timeout: 45, expires: 45, hasExp: true, window: 44, pass: true},
		{name: "45_45_43", timeout: 45, expires: 45, hasExp: true, window: 43, pass: false, bound: "remaining 45s exceeds bound 44s"},
		{name: "300_900_5", timeout: 300, expires: 900, hasExp: true, window: 5, pass: false, bound: "original timeout 900s exceeds ceiling 300s"},
		{name: "expires_absent_45_45", timeout: 45, hasExp: false, window: 45, pass: true},
		{name: "string_5m_4m47s_290", window: 290, pass: true, listing: true},
		{name: "string_5m_4m47s_280", window: 280, pass: false, listing: true, bound: "remaining 287s exceeds bound 281s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := smallModel()
			cp := *probeModel().Probe
			m.Probe = &cp
			m.Probe.ExpiresAt = m.Now.Add(time.Duration(tc.window) * time.Second)
			var src, dst []any
			if tc.listing {
				src = []any{elemVal("100.64.8.1 timeout 5m expires 4m47s", nil)}
				dst = []any{elemVal("1.2.3.4 . tcp . 80 timeout 5m expires 4m47s", nil)}
			} else {
				extra := map[string]any{"timeout": tc.timeout}
				if tc.hasExp {
					extra["expires"] = tc.expires
				}
				src = []any{elemVal("100.64.8.1", extra)}
				dst = []any{elemVal(map[string]any{"concat": []any{"1.2.3.4", "tcp", 80}}, extra)}
			}
			err := VerifyBaseline(probeBaselineWithCustomElems(m, src, dst), m)
			if tc.pass {
				if err != nil {
					t.Fatalf("VerifyBaseline rejected timeout=%d expires=%d window=%ds: %v", tc.timeout, tc.expires, tc.window, err)
				}
				return
			}
			if ReasonCode(err) != ReasonProbeUnbounded || err == nil || !strings.Contains(err.Error(), "probe_v4") || !strings.Contains(err.Error(), tc.bound) {
				t.Fatalf("timeout=%d expires=%d window=%ds reason=%q err=%v, want PROBE_UNBOUNDED naming probe_v4 and %q", tc.timeout, tc.expires, tc.window, ReasonCode(err), err, tc.bound)
			}
		})
	}
}
