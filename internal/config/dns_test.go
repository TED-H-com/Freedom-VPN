package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test12_5_DNSConfigIsExplicitAndRedacted(t *testing.T) {
	if got := Defaults().DNSConfigFile; got != "" {
		t.Fatal("unexpected implicit dns configuration")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "dns.json")
	// TEST ONLY. 192.0.2.0/24 and 2001:db8::/32 are documentation prefixes.
	body := []byte(`{"upstreams":["192.0.2.53:0053","[2001:db8::53]:53"],"bootstrap_dns":["192.0.2.54:53"]}`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDNS(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Upstreams, ",") != "192.0.2.53:53,[2001:db8::53]:53" {
		t.Fatalf("upstreams = %v", got.Upstreams)
	}
	if len(got.BootstrapDNS) != 1 || got.BootstrapDNS[0] != "192.0.2.54:53" {
		t.Fatalf("bootstrap = %v", got.BootstrapDNS)
	}
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("config", "dns", got)
	if strings.Contains(logs.String(), "192.0.2.53") || strings.Contains(logs.String(), "2001:db8") {
		t.Fatal("dns configuration leaked a resolver address")
	}

	for _, raw := range []string{
		`{}`,
		`{"upstreams":[]}`,
		`{"upstreams":["dns.example.com:53"]}`,
		`{"upstreams":["https://192.0.2.53/dns-query"]}`,
		`{"upstreams":["192.0.2.53"]}`,
		`{"upstreams":["192.0.2.53:0"]}`,
		`{"upstreams":["192.0.2.53:65536"]}`,
		`{"upstreams":["192.0.2.53:53","192.0.2.53:53"]}`,
		`{"upstreams":["192.0.2.53:53"],"bootstrap_dns":["192.0.2.1"]}`,
		`{"upstreams":["192.0.2.53:53"],"fallback_dns":["192.0.2.1:53"]}`,
		`{"upstreams":["192.0.2.53:53"]}{"upstreams":["192.0.2.54:53"]}`,
	} {
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDNS(path); err == nil {
			t.Fatalf("accepted invalid dns configuration: %s", raw)
		}
	}
	if _, err := LoadDNS(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("missing dns configuration accepted")
	}
}
