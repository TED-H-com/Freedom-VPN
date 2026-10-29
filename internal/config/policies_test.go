package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPoliciesConfig_LoadAndValidate(t *testing.T) {
	if _, err := LoadPolicies(""); err == nil {
		t.Fatal("expected empty path to fail LoadPolicies")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "policies_config_file" {
			t.Fatalf("expected SettingError(policies_config_file), got %v", err)
		}
	}

	dir := t.TempDir()
	// Incomplete JSON without required fields fails LoadPolicies with policies_config_file.
	incomplete := filepath.Join(dir, "incomplete.json")
	if err := os.WriteFile(incomplete, []byte(`{"fup_mode":"shadow"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPolicies(incomplete); err == nil {
		t.Fatal("expected incomplete policies file to fail LoadPolicies")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "policies_config_file" {
			t.Fatalf("expected SettingError(policies_config_file), got %v", err)
		}
	}

	p := filepath.Join(dir, "policies.json")
	// TEST ONLY explicit settings.
	if err := os.WriteFile(p, []byte(`{"fup_mode":"shadow","fup_sweep_seconds":30,"fup_batch_size":128,"dns_upstream_profile":"TEST-ONLY"}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPolicies(p)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	if loaded.FUPMode != "shadow" || loaded.FUPSweepSeconds != 30 || loaded.FUPBatchSize != 128 || loaded.DNSUpstreamProfile != "TEST-ONLY" {
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
	if err := ValidatePoliciesCentralBounds(baseCfg); err != nil {
		t.Fatalf("ValidatePoliciesCentralBounds(Defaults()): %v", err)
	}
	baseCfg.IdempotencySeconds = 0
	if err := ValidatePoliciesCentralBounds(baseCfg); err == nil {
		t.Fatal("expected invalid IdempotencySeconds to fail ValidatePoliciesCentralBounds")
	} else {
		var se *SettingError
		if !errors.As(err, &se) || se.Setting() != "idempotency_seconds" {
			t.Fatalf("expected SettingError(idempotency_seconds), got %v", err)
		}
	}
}
