//go:build e2e

package e2e

import "testing"

func TestConfiguredProbeIssuedComparesTheChallengeProbeOnly(t *testing.T) {
	configured := map[string]any{"peer_id": "peer-1", "source_v4": "100.64.0.8/32", "destinations": []any{map[string]any{"addr_v4": "192.0.3.10/32", "proto": "udp", "port": 4443}}}
	raw := []byte(`{"txn_id":"t","nonce":"n","probe":{"destinations":[{"port":4443,"proto":"udp","addr_v4":"192.0.3.10/32"}],"peer_id":"peer-1","source_v4":"100.64.0.8/32"}}`)
	if !configuredProbeIssued(raw, configured) {
		t.Fatal("matching challenge probe rejected")
	}
	if configuredProbeIssued([]byte(`{"txn_id":"t","probe":{"peer_id":"other","source_v4":"100.64.0.8/32","destinations":[{"addr_v4":"192.0.3.10/32","proto":"udp","port":4443}]}}`), configured) {
		t.Fatal("different peer accepted")
	}
	if configuredProbeIssued([]byte(`{"txn_id":"t","probe_omitted":{"peer_id":"peer-1","reason":"malformed"}}`), configured) {
		t.Fatal("omitted probe accepted")
	}
}
