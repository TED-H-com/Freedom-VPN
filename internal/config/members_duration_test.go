package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
)

func TestBILL01_MemberDurationIntegerBounds(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow examples require 64-bit int")
	}
	const maxSeconds = 366 * 24 * 60 * 60
	// 18446744074 * 10^9 = 2^64 + 290448384 ns.
	wrap := int64(18446744074)
	fields := []struct {
		name string
		set  func(*Members, int)
	}{
		{"grace", func(c *Members, n int) { c.GraceSeconds = &n }},
		{"trial", func(c *Members, n int) { c.TrialMaxSeconds = &n }},
		{"grant", func(c *Members, n int) { c.GrantMaxSeconds = n }},
	}
	for _, field := range fields {
		for _, tc := range []struct {
			name string
			n    int64
			ok   bool
		}{
			{"maximum", maxSeconds, true},
			{"above_maximum", maxSeconds + 1, false},
			{"positive_wrap", wrap, false},
		} {
			t.Run(field.name+"/"+tc.name, func(t *testing.T) {
				c := validMemberSettings()
				c.GrantMaxSeconds = maxSeconds
				c.GraceReason = "test grace"
				field.set(&c, int(tc.n))
				out, err := c.Entitlements()
				if tc.ok {
					if err != nil {
						t.Fatalf("maximum rejected: %v", err)
					}
					var duration time.Duration
					switch field.name {
					case "grace":
						duration = out.GraceDuration
					case "trial":
						duration = out.MaxTrialDuration
					case "grant":
						duration = out.MaxGrantDuration
					}
					if duration != 366*24*time.Hour {
						t.Fatalf("duration = %v", duration)
					}
				} else if !errors.Is(err, entitlement.ErrInvalid) {
					t.Fatalf("Entitlements error = %v, want invalid input", err)
				}
				data, err := json.Marshal(c)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "members.json")
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
				_, err = LoadMembers(path)
				if tc.ok && err != nil || !tc.ok && (err == nil || err.Error() != "invalid member business settings") {
					t.Fatalf("LoadMembers error = %v", err)
				}
			})
		}
	}
	t.Run("trial_above_configured_grant", func(t *testing.T) {
		c := validMemberSettings()
		trial := c.GrantMaxSeconds + 1
		c.TrialMaxSeconds = &trial
		if _, err := c.Entitlements(); !errors.Is(err, entitlement.ErrInvalid) {
			t.Fatalf("Entitlements error = %v", err)
		}
	})
}
