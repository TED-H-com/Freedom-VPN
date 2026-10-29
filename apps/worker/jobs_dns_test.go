package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
)

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
	if built.Jobs[dnspublish.EventDNSPublish] == nil || len(built.Runners) != 0 {
		t.Fatalf("dns domain jobs=%d runners=%d", len(built.Jobs), len(built.Runners))
	}
}
