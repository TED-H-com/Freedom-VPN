package nativecapture

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAT01_SelectedPeerDoesNotLeakOtherKeys(t *testing.T) {
	raw := []byte(`{"netbirdIp":"100.64.0.1/16","peers":{"details":[{"netbirdIp":"100.64.0.2/16","publicKey":"TEST-ONLY-KEY","status":"Connected","connectionType":"P2P","iceCandidateEndpoint":{"local":"[2001:db8::1]:1234","remote":"[2001:db8::2]:4321"}},{"netbirdIp":"100.64.0.3/16","status":"Disconnected"}]}}`)
	if selectedPublicKey(raw, "100.64.0.2") != "TEST-ONLY-KEY" || selectedPublicKey(raw, "100.64.0.9") != "" {
		t.Fatal("private key join selected wrong peer")
	}
	p, e := Select(raw, "100.64.0.2")
	if e != nil || p.Status != "Connected" {
		t.Fatalf("selected peer: %+v %v", p, e)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), "TEST-ONLY-KEY") {
		t.Fatal("key leaked")
	}
	if _, e = Select(raw, "100.64.0.9"); e == nil {
		t.Fatal("absent peer substituted")
	}
	view := EndpointView(p.Endpoint.Remote)
	if view.Family != "ipv6" || view.Port != "4321" || view.Address != Token("2001:db8::2") {
		t.Fatalf("view %+v", view)
	}
	if Token("2001:db8::2") != Token("2001:db8:0:0::2") {
		t.Fatal("noncanonical address token")
	}
}

func TestAT01_KernelReadbacksAreContentFree(t *testing.T) {
	counters := NumericCounters([]byte("#nstat\nIp6InHdrErrors 9007199254740993 0.0\ninvalid.example.invalid 4\n"))
	if len(counters) != 1 || counters["Ip6InHdrErrors"] != "9007199254740993" {
		t.Fatal(counters)
	}
	tuples := LoopbackConntrack([]byte("udp 17 30 src=::1 dst=2001:db8::41 sport=51820 dport=51820 src=2001:db8::41 dst=::1 sport=51820 dport=51820 [UNREPLIED]\nudp 17 30 src=2001:db8::40 dst=2001:db8::41 sport=1 dport=2\n"))
	if len(tuples) != 2 || tuples[0].Source != Token("::1") || tuples[1].Destination != Token("::1") {
		t.Fatal(tuples)
	}
	b, _ := json.Marshal(tuples)
	if strings.Contains(string(b), "2001:") {
		t.Fatal("address leaked")
	}
}

func TestF1101_NativeHandshakeRemainsSeparate(t *testing.T) {
	raw := []byte(`{"peers":{"details":[{"netbirdIp":"100.64.0.2/32","lastWireguardHandshake":"2026-11-01T10:00:00.123Z"},{"netbirdIp":"100.64.0.3/32","lastWireguardHandshake":"0001-01-01T00:00:00Z"}]}}`)
	p, err := Select(raw, "100.64.0.2")
	if err != nil || p.LastHandshake.Unix() != 1793527200 || p.LastHandshake.Nanosecond() != 123000000 {
		t.Fatalf("timestamp: %v %v", p.LastHandshake, err)
	}
	p, err = Select(raw, "100.64.0.3")
	if err != nil || !p.LastHandshake.IsZero() {
		t.Fatal("unknown handshake became nonzero")
	}
}
