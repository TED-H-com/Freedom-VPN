package e2e

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestD162_InstallerCadenceUnchanged(t *testing.T) {
	input := []byte(`{"heartbeat_seconds":10,"poll_seconds":10,"observation_seconds":30,"state_dir":"/test-only/state","release_id":"TEST ONLY","datapath":{"member_ingress":"fvtest"}}`)
	raw, err := installerServingConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]any
	if json.Unmarshal(input, &before) != nil || json.Unmarshal(raw, &after) != nil {
		t.Fatal("invalid JSON")
	}
	for _, key := range []string{"heartbeat_seconds", "poll_seconds", "observation_seconds"} {
		if after[key] != float64(5) {
			t.Fatalf("%s = %v, want 5", key, after[key])
		}
		delete(before, key)
		delete(after, key)
	}
	if after["test_mode"] != true || after["dns"] == nil {
		t.Fatal("installer serving overrides missing")
	}
	delete(after, "test_mode")
	delete(after, "dns")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("unrelated installer values changed: before=%v after=%v", before, after)
	}
	if _, err := installerServingConfig([]byte(`{"observation_seconds":`)); err == nil {
		t.Fatal("accepted malformed installer configuration")
	}
}
