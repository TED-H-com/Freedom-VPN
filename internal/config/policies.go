// Package config centralizes bounded startup parameters. Durations use seconds.
package config

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
)

// PoliciesConfig holds bounded configuration for policy activation and the
// Fair-Use worker (§11, §14.5).
type PoliciesConfig struct {
	// FUPMode optionally overrides the active fair_use policy revision's mode.
	// Valid values are "" (use active policy revision mode), "shadow", or "enforce".
	FUPMode            string `json:"fup_mode"`
	FUPSweepSeconds    int    `json:"fup_sweep_seconds"`
	FUPBatchSize       int    `json:"fup_batch_size"`
	DNSUpstreamProfile string `json:"dns_upstream_profile"`
}

// PoliciesDefaults returns recommended default parameters for policies and FUP.
func PoliciesDefaults() PoliciesConfig {
	return PoliciesConfig{
		FUPMode:            "",
		FUPSweepSeconds:    60,
		FUPBatchSize:       64,
		DNSUpstreamProfile: "cloudflare",
	}
}

// LoadPolicies loads a strict JSON configuration file and validates all fields.
// No environment variables are read here; missing or invalid files/fields return
// a SettingError naming the offending setting.
func LoadPolicies(path string) (PoliciesConfig, error) {
	c := PoliciesDefaults()
	if strings.TrimSpace(path) == "" {
		return PoliciesConfig{}, &SettingError{Name: "policies_config_file", Err: errors.New("policies configuration file required")}
	}
	f, err := os.Open(path)
	if err != nil {
		return PoliciesConfig{}, &SettingError{Name: "policies_config_file", Err: errors.New("cannot open policies configuration")}
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return PoliciesConfig{}, &SettingError{Name: "policies_config_file", Err: errors.New("invalid policies configuration file")}
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return PoliciesConfig{}, &SettingError{Name: "policies_config_file", Err: errors.New("trailing policies configuration data")}
	}
	return c, c.Validate()
}

// PoliciesFromConfig validates the central Config fields required by the policies
// domain and returns the validated PoliciesConfig.
func PoliciesFromConfig(c Config) (PoliciesConfig, error) {
	if c.IdempotencySeconds <= 0 || c.IdempotencySeconds > 1<<30 {
		return PoliciesConfig{}, &SettingError{Name: "idempotency_seconds", Err: errors.New("invalid positive bound: idempotency_seconds")}
	}
	if c.BrowserBodyBytes <= 0 || c.BrowserBodyBytes > 1<<30 {
		return PoliciesConfig{}, &SettingError{Name: "browser_body_bytes", Err: errors.New("invalid positive bound: browser_body_bytes")}
	}
	if c.WorkerLeaseSeconds <= 0 || c.WorkerLeaseSeconds > 86400 {
		return PoliciesConfig{}, &SettingError{Name: "worker_lease_seconds", Err: errors.New("invalid positive bound: worker_lease_seconds")}
	}
	if c.WorkerJobSeconds <= 0 || c.WorkerJobSeconds >= c.WorkerLeaseSeconds {
		return PoliciesConfig{}, &SettingError{Name: "worker_job_seconds", Err: errors.New("invalid bound: worker_job_seconds")}
	}
	if c.WorkerPollSeconds <= 0 || c.WorkerPollSeconds > 1<<30 {
		return PoliciesConfig{}, &SettingError{Name: "worker_poll_seconds", Err: errors.New("invalid positive bound: worker_poll_seconds")}
	}
	if c.WorkerAttempts <= 0 || c.WorkerAttempts > 1<<30 {
		return PoliciesConfig{}, &SettingError{Name: "worker_attempts", Err: errors.New("invalid positive bound: worker_attempts")}
	}
	p := PoliciesDefaults()
	return p, p.Validate()
}

// Validate checks bounds and allowed enum values, returning a SettingError on failure.
func (c PoliciesConfig) Validate() error {
	switch c.FUPMode {
	case "", "shadow", "enforce":
	default:
		return &SettingError{Name: "fup_mode", Err: errors.New("invalid fup_mode: must be empty, shadow, or enforce")}
	}
	if c.FUPSweepSeconds <= 0 || c.FUPSweepSeconds > 86400 {
		return &SettingError{Name: "fup_sweep_seconds", Err: errors.New("invalid positive bound: fup_sweep_seconds")}
	}
	if c.FUPBatchSize <= 0 || c.FUPBatchSize > 10000 {
		return &SettingError{Name: "fup_batch_size", Err: errors.New("invalid positive bound: fup_batch_size")}
	}
	if strings.TrimSpace(c.DNSUpstreamProfile) == "" {
		return &SettingError{Name: "dns_upstream_profile", Err: errors.New("dns_upstream_profile required")}
	}
	return nil
}

// Effective returns a redacted map of effective settings.
func (c PoliciesConfig) Effective() map[string]any {
	b, _ := json.Marshal(c)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func (c PoliciesConfig) String() string       { return "[REDACTED policies configuration]" }
func (c PoliciesConfig) LogValue() slog.Value { return slog.StringValue(c.String()) }
