// Package config centralizes bounded startup parameters. Durations use seconds.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Bucket struct {
	PerSecond float64 `json:"per_second"`
	Burst     int     `json:"burst"`
}
type Config struct {
	PublicAddress           string   `json:"public_address"`
	AgentAddress            string   `json:"agent_address"`
	MetricsAddress          string   `json:"metrics_address"`
	PublicOrigin            string   `json:"public_origin"`
	TLSCertificateFile      string   `json:"tls_certificate_file"`
	TLSKeyFile              string   `json:"tls_key_file"`
	AgentCAFile             string   `json:"agent_ca_file"`
	DatabaseURLFile         string   `json:"database_url_file"`
	OIDCIssuer              string   `json:"oidc_issuer"`
	OIDCClientID            string   `json:"oidc_client_id"`
	OIDCSecretFile          string   `json:"oidc_secret_file"`
	AdminClaim              string   `json:"admin_claim"`
	AdminValue              string   `json:"admin_value"`
	MFAAMR                  []string `json:"mfa_amr"`
	MFAACR                  []string `json:"mfa_acr"`
	RecentAuthSeconds       int      `json:"recent_auth_seconds"`
	SessionIdleSeconds      int      `json:"session_idle_seconds"`
	SessionAbsoluteSeconds  int      `json:"session_absolute_seconds"`
	LoginTTLSeconds         int      `json:"login_ttl_seconds"`
	IdempotencySeconds      int      `json:"idempotency_seconds"`
	ReadTimeoutSeconds      int      `json:"read_timeout_seconds"`
	WriteTimeoutSeconds     int      `json:"write_timeout_seconds"`
	IdleTimeoutSeconds      int      `json:"idle_timeout_seconds"`
	ShutdownSeconds         int      `json:"shutdown_seconds"`
	BrowserBodyBytes        int64    `json:"browser_body_bytes"`
	AgentBodyBytes          int64    `json:"agent_body_bytes"`
	UsageBodyBytes          int64    `json:"usage_body_bytes"`
	WebhookBodyBytes        int64    `json:"webhook_body_bytes"`
	MaxConcurrent           int      `json:"max_concurrent"`
	MaxRatePrincipals       int      `json:"max_rate_principals"`
	IngressRate             Bucket   `json:"ingress_rate"`
	BrowserRate             Bucket   `json:"browser_rate"`
	AgentRate               Bucket   `json:"agent_rate"`
	HeartbeatRate           Bucket   `json:"heartbeat_rate"`
	LeaseRate               Bucket   `json:"lease_rate"`
	UsageRate               Bucket   `json:"usage_rate"`
	AnonymousRate           Bucket   `json:"anonymous_rate"`
	WorkerConcurrency       int      `json:"worker_concurrency"`
	WorkerLeaseSeconds      int      `json:"worker_lease_seconds"`
	WorkerJobSeconds        int      `json:"worker_job_seconds"`
	WorkerPollSeconds       int      `json:"worker_poll_seconds"`
	WorkerAttempts          int      `json:"worker_attempts"`
	WorkerBackoffSeconds    int      `json:"worker_backoff_seconds"`
	WorkerBackoffMaxSeconds int      `json:"worker_backoff_max_seconds"`
	MigrateOnStart          bool     `json:"migrate_on_start"`
	MembersConfigFile       string   `json:"members_config_file"`
	DNSConfigFile           string   `json:"dns_config_file"`
	NodesConfigFile         string   `json:"nodes_config_file"`
}

func Defaults() Config {
	return Config{
		PublicAddress: ":8443", AgentAddress: ":9443", MetricsAddress: "127.0.0.1:9091", AdminClaim: "roles", AdminValue: "admin", MFAAMR: []string{"mfa"},
		RecentAuthSeconds: 900, SessionIdleSeconds: 1800, SessionAbsoluteSeconds: 43200, LoginTTLSeconds: 300, IdempotencySeconds: 604800,
		ReadTimeoutSeconds: 15, WriteTimeoutSeconds: 30, IdleTimeoutSeconds: 60, ShutdownSeconds: 30,
		BrowserBodyBytes: 1 << 20, AgentBodyBytes: 1 << 20, UsageBodyBytes: 8 << 20, WebhookBodyBytes: 1 << 20, MaxConcurrent: 128, MaxRatePrincipals: 10000,
		IngressRate: Bucket{20, 60}, BrowserRate: Bucket{10, 30}, AgentRate: Bucket{5, 10}, HeartbeatRate: Bucket{1, 5}, LeaseRate: Bucket{1, 5}, UsageRate: Bucket{2, 10}, AnonymousRate: Bucket{2, 10},
		WorkerConcurrency: 4, WorkerLeaseSeconds: 60, WorkerJobSeconds: 40, WorkerPollSeconds: 1, WorkerAttempts: 10, WorkerBackoffSeconds: 1, WorkerBackoffMaxSeconds: 300,
	}
}

// Load accepts strict JSON, then FVPN_<UPPERCASE_JSON_FIELD> overrides. Complex
// fields (buckets and string lists) use JSON in the environment as well.
func Load(path string) (Config, error) {
	c := Defaults()
	if path != "" {
		f, e := os.Open(path)
		if e != nil {
			return c, errors.New("cannot open configuration")
		}
		defer f.Close()
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		if d.Decode(&c) != nil {
			return c, errors.New("invalid configuration file")
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			return c, errors.New("trailing configuration data")
		}
	}
	v := reflect.ValueOf(&c).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		name := t.Field(i).Tag.Get("json")
		s, ok := os.LookupEnv("FVPN_" + strings.ToUpper(name))
		if !ok {
			continue
		}
		f := v.Field(i)
		var err error
		switch f.Kind() {
		case reflect.String:
			f.SetString(s)
		case reflect.Bool:
			var b bool
			b, err = strconv.ParseBool(s)
			f.SetBool(b)
		case reflect.Int, reflect.Int64:
			var n int64
			n, err = strconv.ParseInt(s, 10, 64)
			f.SetInt(n)
		default:
			err = json.Unmarshal([]byte(s), f.Addr().Interface())
		}
		if err != nil {
			return c, fmt.Errorf("invalid setting %s", name)
		}
	}
	return c, c.Validate()
}
func (c Config) Validate() error {
	if c.DatabaseURLFile == "" {
		return errors.New("database_url_file required")
	}
	v := reflect.ValueOf(c)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if (f.Kind() == reflect.Int || f.Kind() == reflect.Int64) && (f.Int() <= 0 || f.Int() > 1<<30) {
			return fmt.Errorf("invalid positive bound: %s", t.Field(i).Tag.Get("json"))
		}
	}
	for _, b := range []Bucket{c.IngressRate, c.BrowserRate, c.AgentRate, c.HeartbeatRate, c.LeaseRate, c.UsageRate, c.AnonymousRate} {
		if b.PerSecond <= 0 || b.PerSecond > 100000 || b.Burst < 1 || b.Burst > 100000 {
			return errors.New("invalid rate bucket")
		}
	}
	if c.WorkerJobSeconds >= c.WorkerLeaseSeconds || c.WorkerLeaseSeconds > 86400 || c.WorkerBackoffSeconds > c.WorkerBackoffMaxSeconds || c.WorkerBackoffMaxSeconds > 86400 || c.WorkerConcurrency > 1000 || c.SessionIdleSeconds > c.SessionAbsoluteSeconds {
		return errors.New("inconsistent timeout or worker bounds")
	}
	return nil
}
func (c Config) ValidateControl() error {
	if err := c.Validate(); err != nil {
		return err
	}
	u, e := url.Parse(c.PublicOrigin)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("public_origin must be an HTTPS origin")
	}
	iss, e := url.Parse(c.OIDCIssuer)
	if e != nil || iss.Scheme != "https" || iss.Host == "" || iss.User != nil {
		return errors.New("oidc_issuer must use HTTPS")
	}
	if c.OIDCClientID == "" || c.AdminClaim == "" || c.AdminValue == "" || len(c.MFAAMR)+len(c.MFAACR) == 0 || c.TLSCertificateFile == "" || c.TLSKeyFile == "" || c.AgentCAFile == "" {
		return errors.New("OIDC, MFA and TLS settings required")
	}
	return nil
}
func ReadSecret(path string) (string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", errors.New("cannot read secret file")
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", errors.New("empty secret file")
	}
	return s, nil
}

// Effective never includes secret contents, file locations, issuer or client IDs.
func (c Config) Effective() map[string]any {
	b, _ := json.Marshal(c)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for k := range m {
		if strings.Contains(k, "file") || strings.HasPrefix(k, "oidc_") {
			m[k] = "[REDACTED]"
		}
	}
	return m
}
