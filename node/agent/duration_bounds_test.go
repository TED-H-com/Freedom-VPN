package agent

import (
	"fmt"
	"strconv"
	"testing"
)

func TestAT13_AgentTimingDurationIntegerBounds(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow examples require 64-bit int")
	}
	fields := []struct {
		name string
		set  func(*Config, int)
	}{
		{"renew_lead", func(c *Config, n int) { c.RenewLeadSeconds = n }},
		{"heartbeat", func(c *Config, n int) { c.HeartbeatSeconds = n }},
		{"watchdog_stale", func(c *Config, n int) { c.WatchdogStaleSeconds = n }},
		{"poll", func(c *Config, n int) { c.PollSeconds = n }},
		{"observation", func(c *Config, n int) { c.ObservationSeconds = n }},
	}
	// 18446744074 * 10^9 = 2^64 + 290448384 ns.
	for _, field := range fields {
		for _, tc := range []struct {
			name string
			n    int64
			ok   bool
		}{
			{"maximum", 9223372036, true},
			{"above_maximum", 9223372037, false},
			{"positive_wrap", 18446744074, false},
			{"negative_wrap", -18446744073, false},
		} {
			t.Run(field.name+"/"+tc.name, func(t *testing.T) {
				var c Config
				field.set(&c, int(tc.n))
				if tc.n > 0 {
					var err error
					c, err = LoadConfig(writeAgentConfig(t, fmt.Sprintf(`, %q: %d`, field.name+"_seconds", tc.n)))
					if err != nil {
						t.Fatalf("LoadConfig error = %v", err)
					}
				}
				err := validateTiming(c)
				if (err == nil) != tc.ok {
					t.Fatalf("validateTiming error = %v", err)
				}
				if !tc.ok {
					want := "agent: " + field.name + "_seconds is out of range"
					if err.Error() != want {
						t.Fatalf("validateTiming error = %v, want %q", err, want)
					}
					if _, err := Start(Options{Config: c}); err == nil || err.Error() != want {
						t.Fatalf("Start error = %v", err)
					}
				}
			})
		}
	}
	t.Run("first_invalid_wins", func(t *testing.T) {
		n64 := int64(9223372037)
		n := int(n64)
		c := Config{RenewLeadSeconds: n, HeartbeatSeconds: n, WatchdogStaleSeconds: n, PollSeconds: n, ObservationSeconds: n}
		if err := validateTiming(c); err == nil || err.Error() != "agent: renew_lead_seconds is out of range" {
			t.Fatalf("validateTiming error = %v", err)
		}
	})
}

func TestAT13_LeaseDurationIntegerBounds(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow examples require 64-bit int")
	}
	// 18446744074 * 10^9 = 2^64 + 290448384 ns.
	for _, tc := range []struct {
		name string
		n    int64
		ok   bool
	}{
		{"maximum", 9223372036, true},
		{"above_maximum", 9223372037, false},
		{"positive_wrap", 18446744074, false},
		{"negative_wrap", -18446744073, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := servingInput()
			in.Lease.MaxOfflineSeconds = int(tc.n)
			_, _, err := buildModel(in)
			if (err == nil) != tc.ok {
				t.Fatalf("buildModel error = %v", err)
			}
		})
	}
}
