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
	if _, err := LoadUsage(""); err == nil {
		t.Fatal("expected empty path to fail LoadUsage")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "usage_config_file" {
			t.Fatalf("expected SettingError(usage_config_file), got %v", err)
		}
	}

	dir := t.TempDir()
	// Missing signing_config_file fails with SettingError naming signing_config_file.
	noSigning := filepath.Join(dir, "no-signing.json")
	if err := os.WriteFile(noSigning, []byte(`{"raw_usage_retention_days":90,"rollup_retention_months":24,"audit_retention_days":365,"projection_batch_size":256}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadUsage(noSigning); err == nil {
		t.Fatal("expected missing signing_config_file to fail LoadUsage")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "signing_config_file" {
			t.Fatalf("expected SettingError(signing_config_file), got %v", err)
		}
	}

	p := filepath.Join(dir, "usage.json")
	// TEST ONLY explicit settings.
	if err := os.WriteFile(p, []byte(`{"signing_config_file":"secrets/signing.json","raw_usage_retention_days":60,"rollup_retention_months":24,"audit_retention_days":365,"projection_batch_size":512}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadUsage(p)
	if err != nil {
		t.Fatalf("LoadUsage: %v", err)
	}
	if loaded.RawUsageRetentionDays != 60 || loaded.RollupRetentionMonths != 24 || loaded.AuditRetentionDays != 365 || loaded.ProjectionBatchSize != 512 {
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
	if err := ValidateUsageCentralBounds(baseCfg); err != nil {
		t.Fatalf("ValidateUsageCentralBounds(Defaults()): %v", err)
	}
	baseCfg.UsageBodyBytes = 0
	if err := ValidateUsageCentralBounds(baseCfg); err == nil {
		t.Fatal("expected invalid UsageBodyBytes to fail ValidateUsageCentralBounds")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "usage_body_bytes" {
			t.Fatalf("expected SettingError(usage_body_bytes), got %v", err)
		}
	}
}
