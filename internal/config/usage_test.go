package config

import (
	"encoding/json"
	"errors"
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
	if _, err := LoadUsage(""); err == nil {
		t.Fatal("expected empty path to fail LoadUsage")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "usage_config_file" {
			t.Fatalf("expected SettingError(usage_config_file), got %v", err)
		}
	}

	p := filepath.Join(t.TempDir(), "usage.json")
	if err := os.WriteFile(p, []byte(`{"signing_config_file":"secrets/signing.json","raw_usage_retention_days":60,"projection_batch_size":512}`), 0600); err != nil {
		t.Fatal(err)
	}
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
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "raw_usage_retention_days" {
			t.Fatalf("expected SettingError(raw_usage_retention_days), got %v", err)
		}
	}

	baseCfg := Defaults()
	if _, err := UsageFromConfig(baseCfg); err != nil {
		t.Fatalf("UsageFromConfig(Defaults()): %v", err)
	}
	baseCfg.UsageBodyBytes = 0
	if _, err := UsageFromConfig(baseCfg); err == nil {
		t.Fatal("expected invalid UsageBodyBytes to fail UsageFromConfig")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "usage_body_bytes" {
			t.Fatalf("expected SettingError(usage_body_bytes), got %v", err)
		}
	}
}
