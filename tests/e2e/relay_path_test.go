package e2e

import "testing"

func TestClassifyNetbirdCountsPathsOnly(t *testing.T) {
	raw := []byte(`{"peers":{"details":[
		{"fqdn":"exit.example.invalid","status":"Connected","connectionType":"Relayed","relayAddress":"rel://198.51.100.8:443"},
		{"fqdn":"other.example.invalid","status":"Connected","connectionType":"P2P","direct":true},
		{"fqdn":"idle.example.invalid","status":"Idle","connectionType":"Relayed","relayed":true}
	]}}`)
	relayed, direct := classifyNetbird(raw)
	if relayed != 1 || direct != 1 {
		t.Fatalf("json counts relayed=%d direct=%d", relayed, direct)
	}
	text := []byte("Peers detail:\n Connection type: Relayed\n Connection type: P2P\n Status: Connected\n")
	relayed, direct = classifyNetbird(text)
	if relayed != 1 || direct != 1 {
		t.Fatalf("text counts relayed=%d direct=%d", relayed, direct)
	}
	relayed, direct = classifyNetbird([]byte(`{"peers":[{"status":"Connected","relayed":false}]}`))
	if relayed != 0 || direct != 1 {
		t.Fatalf("bool counts relayed=%d direct=%d", relayed, direct)
	}
}
