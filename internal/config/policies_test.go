package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPoliciesConfig_LoadAndValidate(t *testing.T) {
	c := PoliciesDefaults()
	if err := c.Validate(); err != nil {
		t.Fatalf("PoliciesDefaults Validate: %v", err)
	}
	p := filepath.Join(t.TempDir(), "policies.json")
	if err := os.WriteFile(p, []byte(`{"fup_mode":"shadow","fup_sweep_seconds":30,"fup_batch_size":128}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FVPN_FUP_MODE", "enforce")
	loaded, err := LoadPolicies(p)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	if loaded.FUPMode != "enforce" || loaded.FUPSweepSeconds != 30 || loaded.FUPBatchSize != 128 {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	loaded.FUPMode = "invalid"
	if err := loaded.Validate(); err == nil {
		t.Fatal("expected invalid fup_mode to fail validation")
	}
}
