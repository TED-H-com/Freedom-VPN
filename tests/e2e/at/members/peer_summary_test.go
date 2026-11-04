package members

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAT08_SelectedPeerSummaryDropsIdentity(t *testing.T) {
	// TEST ONLY synthetic status shaped like the pinned native client output.
	raw := []byte(`{"daemonStatus":"Connected","publicKey":"TEST-PUBLIC-KEY","netbirdIp":"100.64.0.5/32","fqdn":"member.example.invalid","management":{"url":"https://management.example.invalid"},"peers":{"details":[{"netbirdIp":"100.64.0.8/32","netbirdIpv6":"2001:db8::8/128","publicKey":"TEST-PEER-KEY","fqdn":"node.example.invalid","status":"Connected","connectionType":"P2P","iceCandidateEndpoint":{"local":"192.0.2.10:5000","remote":"198.51.100.10:6000"},"relayAddress":"relay.example.invalid:443","lastWireguardHandshake":"2026-11-04T08:00:00Z","transferReceived":9007199254740993,"transferSent":18446744073709551615,"latency":1234000,"networks":["0.0.0.0/0","2001:db8:1::/64","identity.example.invalid","203.0.113.10"],"futureIdentity":{"token":"TEST-IDENTITY"}}]}}`)
	at := time.Date(2026, 11, 4, 8, 0, 7, 0, time.UTC)
	got := nativePeerView(raw, "100.64.0.8", "0.0.0.0/0", at)
	if !got.Available || !got.Found || got.Status != "connected" || got.Transport != "direct" || !got.IncludesSelectedNetwork || len(got.Routes) != 2 || got.HandshakeAgeNS == nil || *got.HandshakeAgeNS != "7000000000" || got.Received == nil || *got.Received != "9007199254740993" || got.Sent == nil || *got.Sent != "18446744073709551615" || got.LatencyNS == nil || *got.LatencyNS != "1234000" {
		t.Fatalf("safe view: %+v", got)
	}
	encoded, _ := json.Marshal(got)
	for _, bad := range []string{"publicKey", "TEST-", "netbirdIp", "fqdn", "Endpoint", "relayAddress", "example.invalid", "100.64.", "192.0.2.", "198.51.100.", "203.0.113.", "2001:db8::8"} {
		if strings.Contains(string(encoded), bad) {
			t.Errorf("identity escaped allowlist: %s", bad)
		}
	}
	// Even nominally safe string fields never copy unknown identity-like values.
	raw = []byte(strings.ReplaceAll(strings.ReplaceAll(string(raw), `"Connected","connectionType":"P2P"`, `"node.example.invalid","connectionType":"192.0.2.8"`), `"0.0.0.0/0"`, `"identity.example.invalid"`))
	got = nativePeerView(raw, "100.64.0.8", "0.0.0.0/0", at)
	if got.Status != "unknown" || got.Transport != "unknown" || got.IncludesSelectedNetwork {
		t.Fatal("unknown fields became asserted metadata")
	}
}

func TestAT08_MissingSelectedPeerIsNeverConnected(t *testing.T) {
	raw := []byte(`{"daemonStatus":"Connected","peers":{"details":[{"netbirdIp":"100.64.0.9/32","status":"Connected","connectionType":"Relayed"}]}}`)
	got := nativePeerView(raw, "100.64.0.8/32", "0.0.0.0/0", time.Now())
	if !got.Available || got.Found || got.Status != "missing" || got.IncludesSelectedNetwork || got.HandshakeAgeNS != nil {
		t.Fatalf("missing: %+v", got)
	}
	got = nativePeerView(raw, "100.64.0.9", "0.0.0.0/0", time.Now())
	if !got.Found || got.Transport != "relayed" || got.HandshakeAgeNS != nil {
		t.Fatalf("relayed: %+v", got)
	}
	for _, raw := range [][]byte{nil, []byte(`{"peers":`)} {
		got = nativePeerView(raw, "100.64.0.8", "0.0.0.0/0", time.Now())
		if got.Available || got.Found || got.Status != "unavailable" {
			t.Fatal("malformed status asserted a peer")
		}
	}
}
