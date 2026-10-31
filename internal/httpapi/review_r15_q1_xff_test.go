package httpapi

import (
	"net"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

// TestReviewR15_DeployProxyTakesTheRightmostUntrustedHop is the devstack
// shape: nginx at 172.28.30.21 appends $proxy_add_x_forwarded_for, and the
// control process trusts that proxy as a /32. A clone that writes the
// victim's address at the left of X-Forwarded-For is not the compatibility
// domain. The domain is the rightmost hop the proxy added.
func TestReviewR15_DeployProxyTakesTheRightmostUntrustedHop(t *testing.T) {
	trusted, err := config.ParseTrustedProxies([]string{"172.28.30.21/32"})
	if err != nil {
		t.Fatal(err)
	}
	const (
		remote   = "172.28.30.21:443"
		xff      = "198.51.100.10, 203.0.113.44"
		victim   = "198.51.100.0/24"
		appended = "203.0.113.0/24"
	)
	got, ok := compatibilityDomain(remote, []string{xff}, trusted)
	if !ok || got != appended {
		t.Fatalf("compatibilityDomain(%q, %q) = (%q, %v), want the proxy-appended hop %s", remote, xff, got, ok, appended)
	}
	if got == victim {
		t.Fatal("leftmost client-supplied address became the compatibility domain")
	}

	// A TCP peer that is not the trusted proxy keeps its own /24. The header
	// is not read, so a leaf that connects directly cannot borrow a victim.
	direct, ok := compatibilityDomain("203.0.113.44:50000", []string{"198.51.100.10"}, trusted)
	if !ok || direct != appended {
		t.Fatalf("untrusted peer domain = (%q, %v), want %s", direct, ok, appended)
	}

	// The same header with no appended hop, from the trusted proxy, is the
	// client-supplied address. deploy/devstack/nginx.conf appends; a proxy
	// that only forwards the header would hide a clone behind the victim.
	naked, ok := compatibilityDomain(remote, []string{"198.51.100.10"}, trusted)
	if !ok || naked != victim {
		t.Fatalf("proxy that does not append: domain = (%q, %v), want %s", naked, ok, victim)
	}

	// An unparseable chosen entry is rejected. It is not replaced by an
	// entry further left.
	if _, ok := compatibilityDomain(remote, []string{"198.51.100.10, not-an-address"}, trusted); ok {
		t.Fatal("unparseable rightmost hop fell through to the leftmost address")
	}

	// The peer itself must parse. net.ParseIP is not the subject of this
	// test; the empty result is.
	if ip := net.ParseIP("172.28.30.21"); ip == nil {
		t.Fatal("proxy address")
	}
}
