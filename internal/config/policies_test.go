package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPoliciesConfig_LoadAndValidate(t *testing.T) {
	c := PoliciesDefaults()
	if err := c.Validate(); err != nil {
		t.Fatalf("PoliciesDefaults Validate: %v", err)
	}
	if _, err := LoadPolicies(""); err == nil {
		t.Fatal("expected empty path to fail LoadPolicies")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "policies_config_file" {
			t.Fatalf("expected SettingError(policies_config_file), got %v", err)
		}
	}

	p := filepath.Join(t.TempDir(), "policies.json")
	if err := os.WriteFile(p, []byte(`{"fup_mode":"shadow","fup_sweep_seconds":30,"fup_batch_size":128}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPolicies(p)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	if loaded.FUPMode != "shadow" || loaded.FUPSweepSeconds != 30 || loaded.FUPBatchSize != 128 {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	loaded.FUPMode = "invalid"
	if err := loaded.Validate(); err == nil {
		t.Fatal("expected invalid fup_mode to fail validation")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "fup_mode" {
			t.Fatalf("expected SettingError(fup_mode), got %v", err)
		}
	}

	baseCfg := Defaults()
	if _, err := PoliciesFromConfig(baseCfg); err != nil {
		t.Fatalf("PoliciesFromConfig(Defaults()): %v", err)
	}
	baseCfg.IdempotencySeconds = 0
	if _, err := PoliciesFromConfig(baseCfg); err == nil {
		t.Fatal("expected invalid IdempotencySeconds to fail PoliciesFromConfig")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "idempotency_seconds" {
			t.Fatalf("expected SettingError(idempotency_seconds), got %v", err)
		}
	}
}
