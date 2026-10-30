package httpapi

import (
	"net"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

func TestParseIPTextShapes(t *testing.T) {
	full := net.ParseIP("2001:db8:1:2:3:4:5:6")
	eight := net.ParseIP("2001:db8:1:2:3:4:5:6")
	compressed := net.ParseIP("2001:db8::1")
	mapped := net.ParseIP("192.0.2.77")
	v4 := net.ParseIP("192.0.2.1")
	absorbed := net.ParseIP("1::2:3:4:5:6")
	hostOfBracket := net.ParseIP("1::2:3:4:5")

	cases := []struct {
		name      string
		in        string
		forwarded net.IP
		remote    net.IP
	}{
		{"bare with compression", "2001:db8:1::2:3:4:5:6", full, full},
		{"eight hextets", "2001:db8:1:2:3:4:5:6", eight, eight},
		{"bracketed no port", "[2001:db8:1::2:3:4:5:6]", full, nil},
		{"bracketed with port", "[2001:db8::1]:443", compressed, compressed},
		{"bracketed zone and port", "[2001:db8::1%eth0]:443", compressed, compressed},
		{"zone without port", "2001:db8::1%eth0", compressed, compressed},
		{"ipv4-mapped", "::ffff:192.0.2.77", mapped, mapped},
		{"bracketed ipv4-mapped with port", "[::ffff:192.0.2.77]:44", mapped, mapped},
		{"unbracketed ipv4-mapped with port", "::ffff:192.0.2.77:44", mapped, mapped},
		{"ipv4 with port", "192.0.2.1:443", v4, v4},
		{"eight hextets plus port", "2001:db8:1:2:3:4:5:6:443", eight, eight},
		{"unbracketed parses whole", "1::2:3:4:5:6", absorbed, absorbed},
		{"bracketed host does not absorb", "[1::2:3:4:5]:6", hostOfBracket, hostOfBracket},
		{"garbage", "not-an-address", nil, nil},
		{"empty", "", nil, nil},
		{"bad hextet", "2001:db8::gggg", nil, nil},
		{"bracketed without port is not a remote", "[2001:db8::1]", compressed, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseForwardedIP(tc.in); !tc.forwarded.Equal(got) {
				t.Fatalf("forwarded %q = %v, want %v", tc.in, got, tc.forwarded)
			}
			if got := parseRemoteIP(tc.in); !tc.remote.Equal(got) {
				t.Fatalf("remote %q = %v, want %v", tc.in, got, tc.remote)
			}
		})
	}
}

func TestCompatibilityDomain(t *testing.T) {
	trusted, err := config.ParseTrustedProxies([]string{"192.0.2.0/24", "2001:db8:feed::/48"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		remoteAddr string
		forwarded  []string
		trusted    []*net.IPNet
		want       string
		wantOK     bool
	}{
		{
			name:       "ipv4 coarsened to /24",
			remoteAddr: "203.0.113.42:54321",
			want:       "203.0.113.0/24",
			wantOK:     true,
		},
		{
			name:       "ipv6 coarsened to /48",
			remoteAddr: "[2001:db8:abcd:1234::99]:54321",
			want:       "2001:db8:abcd::/48",
			wantOK:     true,
		},
		{
			name:       "trusted proxy honours X-Forwarded-For",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"198.51.100.77, 192.0.2.11"},
			trusted:    trusted,
			want:       "198.51.100.0/24",
			wantOK:     true,
		},
		{
			name:       "untrusted peer ignores X-Forwarded-For",
			remoteAddr: "203.0.113.99:443",
			forwarded:  []string{"198.51.100.77"},
			trusted:    trusted,
			want:       "203.0.113.0/24",
			wantOK:     true,
		},
		{
			name:       "unparseable remote address rejected",
			remoteAddr: "not-an-address",
			want:       "",
			wantOK:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := compatibilityDomain(tc.remoteAddr, tc.forwarded, tc.trusted)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("compatibilityDomain(%q, %v) = (%q, %v), want (%q, %v)", tc.remoteAddr, tc.forwarded, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
