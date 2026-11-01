package nativecapture

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAT01_SelectedPeerDoesNotLeakOtherKeys(t *testing.T) {
	raw := []byte(`{"netbirdIp":"100.64.0.1/16","peers":{"details":[{"netbirdIp":"100.64.0.2/16","publicKey":"TEST-ONLY-KEY","status":"Connected","connectionType":"P2P","iceCandidateEndpoint":{"local":"[2001:db8::1]:1234","remote":"[2001:db8::2]:4321"}},{"netbirdIp":"100.64.0.3/16","status":"Disconnected"}]}}`)
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
