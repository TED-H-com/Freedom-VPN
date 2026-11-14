package watchdog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAT13_WatchdogTimingDurationIntegerBounds(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow examples require 64-bit int")
	}
	fields := []struct {
		name, key string
		set       func(*Config, int)
	}{
		{"stale", "watchdog_stale_seconds", func(c *Config, n int) { c.WatchdogStaleSeconds = n }},
		{"interval", "watchdog_interval_seconds", func(c *Config, n int) { c.WatchdogIntervalSeconds = n }},
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
		} {
			t.Run(field.name+"/"+tc.name, func(t *testing.T) {
				c := Config{Datapath: DatapathConfig{MemberIngress: "fv-in", Uplink: "fv-out", SNAT: "192.0.2.1"}}
				field.set(&c, int(tc.n))
				data, err := json.Marshal(c)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "agent.json")
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
				_, err = LoadConfig(path)
				if (err == nil) != tc.ok {
					t.Fatalf("LoadConfig error = %v", err)
				}
				if want := "watchdog: " + field.key + " is out of range"; !tc.ok && err.Error() != want {
					t.Fatalf("LoadConfig error = %v, want %q", err, want)
				}
			})
		}
	}
	t.Run("first_invalid_wins", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "agent.json")
		if err := os.WriteFile(path, []byte(`{"watchdog_stale_seconds":9223372037,"watchdog_interval_seconds":9223372037}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil || err.Error() != "watchdog: watchdog_stale_seconds is out of range" {
			t.Fatalf("LoadConfig error = %v", err)
		}
	})
}
