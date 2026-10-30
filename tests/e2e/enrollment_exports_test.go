//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"
)

func TestAT06_PrivateEnrollmentPathsStayInsideTheContainer(t *testing.T) {
	for _, p := range []string{"/etc/key.pem", "/var/lib/vpn-node/e2e/harness/../identity/key.pem", "/var/lib/vpn-node/e2e/harness/a/../../token", "relative/key"} {
		if validHarnessPath(p) {
			t.Errorf("accepted path escape %q", p)
		}
	}
	if !validHarnessPath("/var/lib/vpn-node/e2e/harness/own/key.pem") {
		t.Fatal("private key path refused")
	}
}

func TestEnrollmentPeerIDsAcceptStringOrObject(t *testing.T) {
	var peers nbPeerIDs
	if err := json.Unmarshal([]byte(`["peer-a",{"id":"peer-b"}]`), &peers); err != nil {
		t.Fatal(err)
	}
	if len(peers) != 2 || peers[0] != "peer-a" || peers[1] != "peer-b" {
		t.Fatalf("peers = %#v", peers)
	}
	if err := json.Unmarshal([]byte(`[{}]`), &peers); err == nil {
		t.Fatal("object without an id accepted")
	}
}
