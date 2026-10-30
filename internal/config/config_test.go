package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAPI01_ControlExampleDomainFiles(t *testing.T) {
	// Exercise the shipped file without ambient FVPN_* overrides masking it.
	typ := reflect.TypeFor[Config]()
	for i := 0; i < typ.NumField(); i++ {
		name := "FVPN_" + strings.ToUpper(typ.Field(i).Tag.Get("json"))
		if _, ok := os.LookupEnv(name); ok {
			t.Setenv(name, "") // Restore the original value during cleanup.
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		}
	}
	c, err := Load("../../docs/control/config.example.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateControl(); err != nil {
		t.Fatal(err)
	}
	got := []string{c.MembersConfigFile, c.DNSConfigFile, c.NodesConfigFile, c.PoliciesConfigFile, c.UsageConfigFile}
	want := []string{"members.json", "dns.json", "nodes.json", "policies.json", "usage.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("example domain files: got %v, want %v", got, want)
	}
	for _, p := range []string{
		c.TLSCertificateFile,
		c.TLSKeyFile,
		c.AgentCAFile,
		c.DatabaseURLFile,
		c.OIDCSecretFile,
		c.MembersConfigFile,
		c.DNSConfigFile,
		c.NodesConfigFile,
		c.PoliciesConfigFile,
		c.UsageConfigFile,
	} {
		if p == "" || filepath.IsAbs(p) {
			t.Fatalf("expected non-empty relative path in config.example.json, got %q", p)
		}
	}
	if c.LoginRate != (Bucket{PerSecond: 1, Burst: 40}) || c.LoginSubjectPerMinute != 10 {
		t.Fatalf("unexpected login rate defaults in example config: login_rate=%+v login_subject_per_minute=%d", c.LoginRate, c.LoginSubjectPerMinute)
	}
}

func TestSEC05_LoginRateValidationRejectsZero(t *testing.T) {
	base := Defaults()
	base.DatabaseURLFile = "secrets/database"
	if err := base.Validate(); err != nil {
		t.Fatalf("defaults should validate: %v", err)
	}

	zeroBurst := base
	zeroBurst.LoginRate.Burst = 0
	if err := zeroBurst.Validate(); err == nil {
		t.Fatal("expected zero login_rate.burst to be rejected")
	}

	zeroRefill := base
	zeroRefill.LoginRate.PerSecond = 0
	if err := zeroRefill.Validate(); err == nil {
		t.Fatal("expected zero login_rate.per_second to be rejected")
	}

	zeroSubject := base
	zeroSubject.LoginSubjectPerMinute = 0
	if err := zeroSubject.Validate(); err == nil {
		t.Fatal("expected zero login_subject_per_minute to be rejected")
	}
}

func TestSEC01_ConfigValidationAndRedaction(t *testing.T) {
	c := Defaults()
	if c.Validate() == nil {
		t.Fatal("missing database accepted")
	}
	c.DatabaseURLFile = "secrets/database"
	c.OIDCSecretFile = "secrets/oidc"
	c.TLSKeyFile = "secrets/key"
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(c.Effective())
	if strings.Contains(string(b), "secrets/") {
		t.Fatal("secret paths printed")
	}
	c.WorkerJobSeconds = c.WorkerLeaseSeconds
	if c.Validate() == nil {
		t.Fatal("lease permits overlap")
	}
}
func TestTrustedProxiesValidation(t *testing.T) {
	base := Defaults()
	base.DatabaseURLFile = "secrets/database"
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, cidr := range []string{"0.0.0.0/0", "::/0", "10.0.0.0/7", "2001:db8::/31", "0.0.0.0/1", "::/0"} {
		c := base
		c.TrustedProxies = []string{cidr}
		err := c.Validate()
		if err == nil || !strings.Contains(err.Error(), "trusting the world") || !strings.Contains(err.Error(), cidr) {
			t.Fatalf("%s: %v", cidr, err)
		}
	}
	c := base
	c.TrustedProxies = []string{"10.0.0.0/8", "192.0.2.9/32", "2001:db8::/32", "2001:db8:1:2::/64"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, cidr := range []string{"192.0.2.1", "not-a-cidr", "::ffff:192.0.2.1/128"} {
		bad := base
		bad.TrustedProxies = []string{cidr}
		if err := bad.Validate(); err == nil {
			t.Fatalf("%s accepted", cidr)
		}
	}
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(`{"database_url_file":"database","trusted_proxies":["192.0.2.1/32","2001:db8:1::/48"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.TrustedProxies) != 2 || loaded.TrustedProxies[0] != "192.0.2.1/32" {
		t.Fatal(loaded.TrustedProxies)
	}
	t.Setenv("FVPN_TRUSTED_PROXIES", `["203.0.113.0/24"]`)
	over, err := Load(p)
	if err != nil || len(over.TrustedProxies) != 1 || over.TrustedProxies[0] != "203.0.113.0/24" {
		t.Fatal(over.TrustedProxies, err)
	}
}

func TestAPI01_ConfigFileAndEnv(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if e := os.WriteFile(p, []byte(`{"database_url_file":"database","session_idle_seconds":10}`), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("FVPN_SESSION_IDLE_SECONDS", "20")
	c, e := Load(p)
	if e != nil || c.SessionIdleSeconds != 20 || c.IdempotencySeconds != 604800 {
		t.Fatal(c, e)
	}
	if e = os.WriteFile(p, []byte(`{"unknown":true}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Load(p); e == nil {
		t.Fatal("unknown setting accepted")
	}
}
