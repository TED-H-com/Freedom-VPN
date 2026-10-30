package main

import "testing"

func TestChallengeCarriesProbeIgnoresKeyOrder(t *testing.T) {
	configured := map[string]any{"peer_id": "peer-1", "source_v4": "100.64.0.8/32", "destinations": []any{map[string]any{"addr_v4": "192.0.3.10/32", "proto": "udp", "port": 4443}}}
	raw := []byte(`{"nonce":"n","probe":{"source_v4":"100.64.0.8/32","peer_id":"peer-1","destinations":[{"proto":"udp","port":4443,"addr_v4":"192.0.3.10/32"}]}}`)
	if !challengeCarriesProbe(raw, configured) {
		t.Fatal("matching probe rejected")
	}
	if challengeCarriesProbe([]byte(`{"probe":null}`), configured) {
		t.Fatal("null probe accepted")
	}
}
