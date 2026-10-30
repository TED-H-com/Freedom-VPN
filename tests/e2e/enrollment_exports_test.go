//go:build e2e

package e2e

import "testing"

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
