// Package config centralizes bounded startup parameters. Durations use seconds.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// UsageConfig holds bounded configuration for the usage ledger, checkpoint/epoch
// signing provider, and retention bounds (§10.9, §15.4, §16.3, §17.3).
type UsageConfig struct {
	SigningConfigFile     string `json:"signing_config_file"`
	RawUsageRetentionDays int    `json:"raw_usage_retention_days"`
	RollupRetentionMonths int    `json:"rollup_retention_months"`
	AuditRetentionDays    int    `json:"audit_retention_days"`
	ProjectionBatchSize   int    `json:"projection_batch_size"`
}

// UsageDefaults returns the spec-recommended defaults (§15.4).
func UsageDefaults() UsageConfig {
	return UsageConfig{
		RawUsageRetentionDays: 90,
		RollupRetentionMonths: 24,
		AuditRetentionDays:    365,
		ProjectionBatchSize:   256,
	}
}

// LoadUsage loads a JSON configuration file (when path is non-empty) and
// applies FVPN_<UPPERCASE_JSON_FIELD> environment variable overrides.
func LoadUsage(path string) (UsageConfig, error) {
	c := UsageDefaults()
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return c, errors.New("cannot open usage configuration")
		}
		defer f.Close()
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		if err := d.Decode(&c); err != nil {
			return c, errors.New("invalid usage configuration file")
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			return c, errors.New("trailing usage configuration data")
		}
	}
	if v, ok := os.LookupEnv("FVPN_SIGNING_CONFIG_FILE"); ok {
		c.SigningConfigFile = strings.TrimSpace(v)
	}
	if v, ok := os.LookupEnv("FVPN_USAGE_SIGNING_CONFIG_FILE"); ok && strings.TrimSpace(v) != "" {
		c.SigningConfigFile = strings.TrimSpace(v)
	}
	if v, ok := os.LookupEnv("FVPN_RAW_USAGE_RETENTION_DAYS"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return c, fmt.Errorf("invalid setting raw_usage_retention_days")
		}
		c.RawUsageRetentionDays = n
	}
	if v, ok := os.LookupEnv("FVPN_ROLLUP_RETENTION_MONTHS"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return c, fmt.Errorf("invalid setting rollup_retention_months")
		}
		c.RollupRetentionMonths = n
	}
	if v, ok := os.LookupEnv("FVPN_AUDIT_RETENTION_DAYS"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return c, fmt.Errorf("invalid setting audit_retention_days")
		}
		c.AuditRetentionDays = n
	}
	if v, ok := os.LookupEnv("FVPN_PROJECTION_BATCH_SIZE"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return c, fmt.Errorf("invalid setting projection_batch_size")
		}
		c.ProjectionBatchSize = n
	}
	return c, c.Validate()
}

// LoadUsageFromEnv loads UsageDefaults with environment overrides.
func LoadUsageFromEnv() (UsageConfig, error) {
	return LoadUsage("")
}

// Validate enforces the §15.4 retention minimums and positive batch bounds.
func (c UsageConfig) Validate() error {
	if c.RawUsageRetentionDays < 30 || c.RawUsageRetentionDays > 3650 {
		return errors.New("invalid bound: raw_usage_retention_days must be between 30 and 3650")
	}
	if c.RollupRetentionMonths < 12 || c.RollupRetentionMonths > 120 {
		return errors.New("invalid bound: rollup_retention_months must be between 12 and 120")
	}
	if c.AuditRetentionDays < 180 || c.AuditRetentionDays > 3650 {
		return errors.New("invalid bound: audit_retention_days must be between 180 and 3650")
	}
	if c.ProjectionBatchSize <= 0 || c.ProjectionBatchSize > 10000 {
		return errors.New("invalid positive bound: projection_batch_size")
	}
	return nil
}

// Effective never includes secret file paths.
func (c UsageConfig) Effective() map[string]any {
	b, _ := json.Marshal(c)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for k := range m {
		if strings.Contains(k, "file") {
			m[k] = "[REDACTED]"
		}
	}
	return m
}
