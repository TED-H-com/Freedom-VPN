package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsageConfig_LoadValidateAndRedact(t *testing.T) {
	c := UsageDefaults()
	if err := c.Validate(); err != nil {
		t.Fatalf("UsageDefaults Validate: %v", err)
	}
	p := filepath.Join(t.TempDir(), "usage.json")
	if err := os.WriteFile(p, []byte(`{"signing_config_file":"secrets/signing.json","raw_usage_retention_days":60}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FVPN_PROJECTION_BATCH_SIZE", "512")
	loaded, err := LoadUsage(p)
	if err != nil {
		t.Fatalf("LoadUsage: %v", err)
	}
	if loaded.RawUsageRetentionDays != 60 || loaded.ProjectionBatchSize != 512 {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	raw, _ := json.Marshal(loaded.Effective())
	if strings.Contains(string(raw), "secrets/signing.json") {
		t.Fatalf("signing_config_file leaked in Effective(): %s", raw)
	}
	loaded.RawUsageRetentionDays = 10
	if err := loaded.Validate(); err == nil {
		t.Fatal("expected raw_usage_retention_days < 30 to fail validation")
	}
}
