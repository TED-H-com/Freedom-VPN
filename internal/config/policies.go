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

// LoadPolicies loads a JSON configuration file (when path is non-empty) and
// applies FVPN_<UPPERCASE_JSON_FIELD> environment variable overrides.
func LoadPolicies(path string) (PoliciesConfig, error) {
	c := PoliciesDefaults()
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return c, errors.New("cannot open policies configuration")
		}
		defer f.Close()
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		if err := d.Decode(&c); err != nil {
			return c, errors.New("invalid policies configuration file")
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			return c, errors.New("trailing policies configuration data")
		}
	}
	if v, ok := os.LookupEnv("FVPN_FUP_MODE"); ok {
		c.FUPMode = strings.TrimSpace(v)
	}
	if v, ok := os.LookupEnv("FVPN_FUP_SWEEP_SECONDS"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return c, fmt.Errorf("invalid setting fup_sweep_seconds")
		}
		c.FUPSweepSeconds = n
	}
	if v, ok := os.LookupEnv("FVPN_FUP_BATCH_SIZE"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return c, fmt.Errorf("invalid setting fup_batch_size")
		}
		c.FUPBatchSize = n
	}
	if v, ok := os.LookupEnv("FVPN_DNS_UPSTREAM_PROFILE"); ok {
		c.DNSUpstreamProfile = strings.TrimSpace(v)
	}
	return c, c.Validate()
}

// LoadPoliciesFromEnv loads PoliciesDefaults with environment overrides.
func LoadPoliciesFromEnv() (PoliciesConfig, error) {
	return LoadPolicies("")
}

// Validate checks bounds and allowed enum values.
func (c PoliciesConfig) Validate() error {
	switch c.FUPMode {
	case "", "shadow", "enforce":
	default:
		return errors.New("invalid fup_mode: must be empty, shadow, or enforce")
	}
	if c.FUPSweepSeconds <= 0 || c.FUPSweepSeconds > 86400 {
		return errors.New("invalid positive bound: fup_sweep_seconds")
	}
	if c.FUPBatchSize <= 0 || c.FUPBatchSize > 10000 {
		return errors.New("invalid positive bound: fup_batch_size")
	}
	if strings.TrimSpace(c.DNSUpstreamProfile) == "" {
		return errors.New("dns_upstream_profile required")
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
