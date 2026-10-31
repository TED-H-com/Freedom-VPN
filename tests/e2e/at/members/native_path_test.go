package members

import "testing"

func TestAT09_SelectedExitFamilyExcludesOtherPeer(t *testing.T) {
	raw := []byte(`{"peers":{"details":[{"netbirdIp":"100.64.0.2/16","status":"Connected","connectionType":"P2P","iceCandidateEndpoint":{"remote":"198.51.100.2:51820"}},{"netbirdIp":"100.64.0.3","status":"Connected","connectionType":"P2P","iceCandidateEndpoint":{"remote":"[2001:db8::3]:51820"}}]}}`)
	got, err := selectNativePath(raw, "100.64.0.3")
	if err != nil || !got.Connected || got.Family != "ipv6" {
		t.Fatalf("selected path: %+v %v", got, err)
	}
	if _, err = selectNativePath(raw, "100.64.0.4"); err == nil {
		t.Fatal("missing selected exit was substituted")
	}
}
