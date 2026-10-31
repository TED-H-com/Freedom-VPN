// Package config centralizes bounded startup parameters. Durations use seconds.
package config

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// UsageConfig holds explicit bounded configuration for the usage ledger, checkpoint/epoch
// signing provider, and retention bounds (§10.9, §15.4, §16.3, §17.3, D-045).
// SigningConfigFile is required; there are no compiled-in business defaults.
type UsageConfig struct {
	SigningConfigFile     string `json:"signing_config_file"`
	RawUsageRetentionDays int    `json:"raw_usage_retention_days"`
	RollupRetentionMonths int    `json:"rollup_retention_months"`
	AuditRetentionDays    int    `json:"audit_retention_days"`
	ProjectionBatchSize   int    `json:"projection_batch_size"`
	// Nil omits the D-112/D-115 setting and selects the 10-second default.
	// A pointer distinguishes omission from an explicitly invalid zero value.
	SamplingIntervalBoundSeconds *int `json:"sampling_interval_bound_seconds,omitempty"`
}

const defaultUsageSamplingIntervalBoundSeconds = 10

// LoadUsage loads a strict JSON configuration file and validates all fields.
// The optional sampling interval bound defaults to 10 seconds (D-112/D-115).
// Explicit zero, negative and null values are invalid. Other settings remain
// required; missing or invalid files/fields return a SettingError.
func LoadUsage(path string) (UsageConfig, error) {
	bound := defaultUsageSamplingIntervalBoundSeconds
	c := UsageConfig{SamplingIntervalBoundSeconds: &bound}
	if strings.TrimSpace(path) == "" {
		return UsageConfig{}, &SettingError{Name: "usage_config_file", Err: errors.New("usage configuration file required")}
	}
	f, err := os.Open(path)
	if err != nil {
		return UsageConfig{}, &SettingError{Name: "usage_config_file", Err: errors.New("cannot open usage configuration")}
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return UsageConfig{}, &SettingError{Name: "usage_config_file", Err: errors.New("invalid usage configuration file")}
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return UsageConfig{}, &SettingError{Name: "usage_config_file", Err: errors.New("trailing usage configuration data")}
	}
	if c.SamplingIntervalBoundSeconds == nil {
		return UsageConfig{}, &SettingError{Name: "sampling_interval_bound_seconds", Err: errors.New("sampling interval bound must not be null")}
	}
	return c, c.Validate()
}

// ValidateUsageCentralBounds checks the central Config bounds required by the
// usage domain in addition to loading usage_config_file.
func ValidateUsageCentralBounds(c Config) error {
	if c.UsageBodyBytes <= 0 || c.UsageBodyBytes > 1<<30 {
		return &SettingError{Name: "usage_body_bytes", Err: errors.New("invalid positive bound: usage_body_bytes")}
	}
	if c.UsageRate.PerSecond <= 0 || c.UsageRate.PerSecond > 100000 || c.UsageRate.Burst < 1 || c.UsageRate.Burst > 100000 {
		return &SettingError{Name: "usage_rate", Err: errors.New("invalid usage_rate bucket")}
	}
	if c.IdempotencySeconds <= 0 || c.IdempotencySeconds > 1<<30 {
		return &SettingError{Name: "idempotency_seconds", Err: errors.New("invalid positive bound: idempotency_seconds")}
	}
	if c.WorkerLeaseSeconds <= 0 || c.WorkerLeaseSeconds > 86400 {
		return &SettingError{Name: "worker_lease_seconds", Err: errors.New("invalid positive bound: worker_lease_seconds")}
	}
	if c.WorkerJobSeconds <= 0 || c.WorkerJobSeconds >= c.WorkerLeaseSeconds {
		return &SettingError{Name: "worker_job_seconds", Err: errors.New("invalid bound: worker_job_seconds")}
	}
	if c.WorkerPollSeconds <= 0 || c.WorkerPollSeconds > 1<<30 {
		return &SettingError{Name: "worker_poll_seconds", Err: errors.New("invalid positive bound: worker_poll_seconds")}
	}
	if c.WorkerAttempts <= 0 || c.WorkerAttempts > 1<<30 {
		return &SettingError{Name: "worker_attempts", Err: errors.New("invalid positive bound: worker_attempts")}
	}
	return nil
}

// Validate enforces that signing_config_file is present and checks the §15.4
// retention minimums and positive batch bounds, returning a SettingError on failure.
func (c UsageConfig) Validate() error {
	if strings.TrimSpace(c.SigningConfigFile) == "" {
		return &SettingError{Name: "signing_config_file", Err: errors.New("signing_config_file required")}
	}
	if c.RawUsageRetentionDays < 30 || c.RawUsageRetentionDays > 3650 {
		return &SettingError{Name: "raw_usage_retention_days", Err: errors.New("invalid bound: raw_usage_retention_days must be between 30 and 3650")}
	}
	if c.RollupRetentionMonths < 12 || c.RollupRetentionMonths > 120 {
		return &SettingError{Name: "rollup_retention_months", Err: errors.New("invalid bound: rollup_retention_months must be between 12 and 120")}
	}
	if c.AuditRetentionDays < 180 || c.AuditRetentionDays > 3650 {
		return &SettingError{Name: "audit_retention_days", Err: errors.New("invalid bound: audit_retention_days must be between 180 and 3650")}
	}
	if c.ProjectionBatchSize <= 0 || c.ProjectionBatchSize > 10000 {
		return &SettingError{Name: "projection_batch_size", Err: errors.New("invalid positive bound: projection_batch_size")}
	}
	if c.SamplingIntervalBoundSeconds != nil {
		seconds := *c.SamplingIntervalBoundSeconds
		if seconds <= 0 || int64(seconds) > int64((1<<63-1)/time.Second) {
			return &SettingError{Name: "sampling_interval_bound_seconds", Err: errors.New("sampling interval bound must be positive and fit a duration")}
		}
	}
	return nil
}

// Effective never includes secret file paths.
func (c UsageConfig) Effective() map[string]any {
	if c.SamplingIntervalBoundSeconds == nil {
		bound := defaultUsageSamplingIntervalBoundSeconds
		c.SamplingIntervalBoundSeconds = &bound
	}
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

func (c UsageConfig) String() string       { return "[REDACTED usage configuration]" }
func (c UsageConfig) LogValue() slog.Value { return slog.StringValue(c.String()) }
