package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeUsageConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "usage.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReviewR17_SamplingIntervalBoundConfig(t *testing.T) {
	base := `"signing_config_file":"secrets/signing.json","raw_usage_retention_days":60,"rollup_retention_months":24,"audit_retention_days":365,"projection_batch_size":512`
	for _, tc := range []struct {
		name string
		body string
		ok   bool
	}{
		{"zero", `{` + base + `,"sampling_interval_bound_seconds":0}`, false},
		{"negative", `{` + base + `,"sampling_interval_bound_seconds":-1}`, false},
		{"null", `{` + base + `,"sampling_interval_bound_seconds":null}`, false},
		{"overflow", `{` + base + `,"sampling_interval_bound_seconds":9223372037}`, false},
		{"two-to-the-31", `{` + base + `,"sampling_interval_bound_seconds":2147483648}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadUsage(writeUsageConfig(t, tc.body))
			if tc.ok {
				if err != nil || got.SamplingIntervalBoundSeconds == nil || *got.SamplingIntervalBoundSeconds != 2147483648 {
					t.Fatalf("2^31 bound: %+v %v", got.SamplingIntervalBoundSeconds, err)
				}
				return
			}
			var se *SettingError
			if !errors.As(err, &se) || se.Setting() != "sampling_interval_bound_seconds" {
				t.Fatal(err)
			}
		})
	}
}
