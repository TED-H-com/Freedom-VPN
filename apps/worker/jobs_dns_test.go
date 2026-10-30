package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
)

func TestD045_PoliciesAndUsageWorkerDomainsRequireConfigAndSigningFiles(t *testing.T) {
	ctx := context.Background()
	c := config.Defaults()

	// 1. Missing policies_config_file fails with SettingError naming policies_config_file.
	_, err := Open(ctx, Dependencies{Config: c}, policies(), usage())
	var setting *config.SettingError
	if !errors.As(err, &setting) || setting.Setting() != "policies_config_file" {
		t.Fatalf("missing policies_config_file: got %v", err)
	}

	// 2. Missing usage_config_file fails with SettingError naming usage_config_file.
	c.PoliciesConfigFile = policiestest.New(t).Path
	_, err = Open(ctx, Dependencies{Config: c}, policies(), usage())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "usage_config_file" {
		t.Fatalf("missing usage_config_file: got %v", err)
	}

	// 3. Missing signing_config_file inside usage_config_file fails with SettingError naming signing_config_file.
	dir := t.TempDir()
	usageNoSigning := filepath.Join(dir, "usage-no-signing.json")
	if err := os.WriteFile(usageNoSigning, []byte(`{
		"raw_usage_retention_days": 90,
		"rollup_retention_months": 24,
		"audit_retention_days": 365,
		"projection_batch_size": 256
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c.UsageConfigFile = usageNoSigning
	_, err = Open(ctx, Dependencies{Config: c}, usage())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "signing_config_file" {
		t.Fatalf("missing signing_config_file in usage file: got %v", err)
	}

	// 4. Unreadable signing_config_file inside usage_config_file fails with SettingError naming signing_config_file.
	usageBadVault := filepath.Join(dir, "usage-bad-vault.json")
	if err := os.WriteFile(usageBadVault, []byte(`{
		"signing_config_file": "`+filepath.Join(dir, "nonexistent-vault")+`",
		"raw_usage_retention_days": 90,
		"rollup_retention_months": 24,
		"audit_retention_days": 365,
		"projection_batch_size": 256
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c.UsageConfigFile = usageBadVault
	_, err = Open(ctx, Dependencies{Config: c}, usage())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "signing_config_file" {
		t.Fatalf("unreadable signing_config_file: got %v", err)
	}
}

func Test12_5_DNSDomainRequiresConfigFile(t *testing.T) {
	c := config.Defaults()
	c.DatabaseURLFile = "TEST-ONLY"
	_, err := Open(context.Background(), Dependencies{Config: c}, dns())
	var setting *config.SettingError
	if !errors.As(err, &setting) || setting.Setting() != "dns_config_file" {
		t.Fatalf("missing dns_config_file: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "dns.json")
	if err := os.WriteFile(path, []byte(`{"upstreams":["dns.example.com:53"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c.DNSConfigFile = path
	_, err = Open(context.Background(), Dependencies{Config: c}, dns())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "dns_config_file" {
		t.Fatalf("invalid dns_config_file: %v", err)
	}

	// TEST ONLY documentation prefix.
	if err := os.WriteFile(path, []byte(`{"upstreams":["192.0.2.53:53"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	built, err := dns().Build(context.Background(), Dependencies{Config: c})
	if err != nil {
		t.Fatal(err)
	}
	if built.Jobs[dnspublish.EventDNSPublish] == nil || len(built.Runners) != 1 || built.Jobs["dns.feed.refresh"] == nil {
		t.Fatalf("dns domain jobs=%d runners=%d", len(built.Jobs), len(built.Runners))
	}
}
