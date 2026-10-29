package releases

import (
	"net/netip"
	"testing"
)

func TestSEC01_FetcherRejectsNonpublicAddresses(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "169.254.169.254", "10.0.0.1", "100.64.0.1", "192.0.2.1", "::1", "::ffff:127.0.0.1", "fe80::1", "fc00::1", "2001:db8::1", "64:ff9b::a00:1"} {
		if publicAddress(netip.MustParseAddr(ip)) {
			t.Fatalf("accepted %s", ip)
		}
	}
}
