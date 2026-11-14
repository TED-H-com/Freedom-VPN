package config

import (
	"errors"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestAT28_RateBucketPerSecondBounds(t *testing.T) {
	rates := []struct {
		name  string
		value float64
		valid bool
	}{
		{"minimum", MinRatePerSecond, true},
		{"maximum", MaxRatePerSecond, true},
		{"below_minimum", math.Nextafter(MinRatePerSecond, 0), false},
		{"duration_overflow_boundary", 1.08e-10, false},
		{"tiny", 1e-11, false},
		{"zero", 0, false},
		{"negative", -1, false},
		{"nan", math.NaN(), false},
		{"above_maximum", math.Nextafter(MaxRatePerSecond, math.Inf(1)), false},
		{"positive_infinity", math.Inf(1), false},
		{"negative_infinity", math.Inf(-1), false},
	}
	buckets := []struct {
		name string
		get  func(*Config) *Bucket
	}{
		{"ingress_rate", func(c *Config) *Bucket { return &c.IngressRate }},
		{"browser_rate", func(c *Config) *Bucket { return &c.BrowserRate }},
		{"agent_rate", func(c *Config) *Bucket { return &c.AgentRate }},
		{"heartbeat_rate", func(c *Config) *Bucket { return &c.HeartbeatRate }},
		{"lease_rate", func(c *Config) *Bucket { return &c.LeaseRate }},
		{"usage_rate", func(c *Config) *Bucket { return &c.UsageRate }},
		{"anonymous_rate", func(c *Config) *Bucket { return &c.AnonymousRate }},
		{"read_rate", func(c *Config) *Bucket { return &c.ReadRate }},
		{"login_rate", func(c *Config) *Bucket { return &c.LoginRate }},
	}

	for _, bucket := range buckets {
		t.Run(bucket.name, func(t *testing.T) {
			for _, rate := range rates {
				c := Defaults()
				c.DatabaseURLFile = "database"
				*bucket.get(&c) = Bucket{PerSecond: rate.value, Burst: 1}
				err := c.Validate()
				if rate.valid {
					if err != nil {
						t.Errorf("rate %s (%g): unexpected error: %v", rate.name, rate.value, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "invalid rate bucket: "+bucket.name) {
					t.Errorf("rate %s (%g): got %v, want named bucket error", rate.name, rate.value, err)
				}
			}
			for _, burst := range []int{0, 1, 100000, 100001} {
				c := Defaults()
				c.DatabaseURLFile = "database"
				*bucket.get(&c) = Bucket{PerSecond: 1, Burst: burst}
				err := c.Validate()
				if burst >= 1 && burst <= 100000 {
					if err != nil {
						t.Errorf("burst %d: unexpected error: %v", burst, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "invalid rate bucket: "+bucket.name) {
					t.Errorf("burst %d: got %v, want named bucket error", burst, err)
				}
			}
		})
	}

	t.Run("login_subject_per_minute", func(t *testing.T) {
		c := Defaults()
		c.DatabaseURLFile = "database"
		c.LoginSubjectPerMinute = 0
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "invalid rate bucket: login_subject_per_minute") {
			t.Fatalf("got %v, want named bucket error", err)
		}
	})

	t.Run("usage_central_bounds", func(t *testing.T) {
		for _, rate := range rates {
			c := Defaults()
			c.UsageRate = Bucket{PerSecond: rate.value, Burst: 1}
			err := ValidateUsageCentralBounds(c)
			if rate.valid {
				if err != nil {
					t.Errorf("rate %s (%g): unexpected error: %v", rate.name, rate.value, err)
				}
			} else {
				var settingErr *SettingError
				if !errors.As(err, &settingErr) || settingErr.Name != "usage_rate" {
					t.Errorf("rate %s (%g): got %v, want usage_rate SettingError", rate.name, rate.value, err)
				}
			}
		}
	})

	t.Run("load_environment_override", func(t *testing.T) {
		typ := reflect.TypeFor[Config]()
		for i := 0; i < typ.NumField(); i++ {
			name := "FVPN_" + strings.ToUpper(typ.Field(i).Tag.Get("json"))
			if _, ok := os.LookupEnv(name); ok {
				t.Setenv(name, "")
				if err := os.Unsetenv(name); err != nil {
					t.Fatal(err)
				}
			}
		}
		t.Setenv("FVPN_DATABASE_URL_FILE", "database")
		t.Setenv("FVPN_BROWSER_RATE", `{"per_second":1e-11,"burst":1}`)
		_, err := Load("")
		if err == nil || !strings.Contains(err.Error(), "invalid rate bucket: browser_rate") {
			t.Fatalf("got %v, want named bucket error", err)
		}
	})
}
