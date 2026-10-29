package updater

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func TestUPD_ExactOriginRejectsQueryAndFragment(t *testing.T) {
	forced, err := url.Parse("https://10.1.2.3:8443?")
	if err != nil || !forced.ForceQuery || forced.RawQuery != "" {
		t.Fatalf("ForceQuery shape: %+v %v", forced, err)
	}
	hashed, err := url.Parse("https://10.1.2.3:8443#")
	if err != nil || hashed.Fragment != "" {
		t.Fatalf("bare hash shape: %+v %v", hashed, err)
	}
	cases := []struct {
		raw string
		ok  bool
	}{
		{raw: "https://10.1.2.3", ok: true},
		{raw: "https://10.1.2.3:8443", ok: true},
		{raw: "http://releases.internal", ok: true},
		{raw: "https://10.1.2.3:8443?", ok: false},
		{raw: "https://10.1.2.3:8443#", ok: false},
		{raw: "https://10.1.2.3:8443?x=1", ok: false},
		{raw: "https://10.1.2.3:8443#frag", ok: false},
		{raw: "https://10.1.2.3/manifest", ok: false},
		{raw: "https://user@10.1.2.3", ok: false},
		{raw: "ftp://10.1.2.3", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			if got := exactOrigin(tc.raw); got != tc.ok {
				t.Fatalf("exactOrigin = %v", got)
			}
			err := validateOrigins([]string{tc.raw})
			if tc.ok && err != nil {
				t.Fatal(err)
			}
			if !tc.ok && err == nil {
				t.Fatal("validateOrigins accepted the entry")
			}
		})
	}
}

func TestUPD_OriginEffectivePortAndScheme(t *testing.T) {
	cases := []struct {
		name    string
		scheme  string
		host    string
		port    string
		origins []string
		want    bool
	}{
		{name: "https omitted equals explicit 443", scheme: "https", host: "10.1.2.3", port: "443", origins: []string{"https://10.1.2.3"}, want: true},
		{name: "explicit 443 equals https omitted", scheme: "https", host: "10.1.2.3", port: "", origins: []string{"https://10.1.2.3:443"}, want: true},
		{name: "https omitted does not equal 8443", scheme: "https", host: "10.1.2.3", port: "8443", origins: []string{"https://10.1.2.3"}, want: false},
		{name: "dns https omitted equals 443", scheme: "https", host: "releases.internal", port: "443", origins: []string{"https://releases.internal"}, want: true},
		{name: "http 8443 does not match https 8443", scheme: "https", host: "releases.internal", port: "8443", origins: []string{"http://releases.internal:8443"}, want: false},
		{name: "http omitted equals 80", scheme: "http", host: "releases.internal", port: "80", origins: []string{"http://releases.internal"}, want: true},
		{name: "http omitted does not match https 80", scheme: "https", host: "releases.internal", port: "80", origins: []string{"http://releases.internal"}, want: false},
		{name: "empty scheme does not match", scheme: "", host: "releases.internal", port: "443", origins: []string{"https://releases.internal"}, want: false},
		{name: "different host", scheme: "https", host: "10.9.9.9", port: "443", origins: []string{"https://10.1.2.3"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := originsMatch(tc.scheme, tc.host, tc.port, tc.origins); got != tc.want {
				t.Fatalf("originsMatch = %v", got)
			}
			if got := hostPortAllowed(tc.scheme, tc.host, tc.port, tc.origins); got != tc.want {
				t.Fatalf("hostPortAllowed = %v", got)
			}
		})
	}
	literal, err := url.Parse("https://10.1.2.3:443/res")
	if err != nil {
		t.Fatal(err)
	}
	if !originAllowed(literal, []string{"https://10.1.2.3"}) {
		t.Fatal("ip literal :443 did not match an omitted https origin")
	}
	mismatched, err := url.Parse("https://releases.internal:8443/res")
	if err != nil {
		t.Fatal(err)
	}
	if originAllowed(mismatched, []string{"http://releases.internal:8443"}) {
		t.Fatal("https url matched an http origin")
	}
	if err := checkURLPolicy("https://10.1.2.3:443/res", urlPolicy{AllowedOrigins: []string{"https://10.1.2.3"}}); err != nil {
		t.Fatal(err)
	}
	if err := checkURLPolicy("https://10.1.2.3:8443/res", urlPolicy{AllowedOrigins: []string{"https://10.1.2.3"}}); err == nil {
		t.Fatal("8443 admitted by an omitted https origin")
	}
	if err := checkURLPolicy("https://releases.example/a", urlPolicy{}); err != nil {
		t.Fatal(err)
	}

	oldLookup, oldDial := lookupIPs, dialContextFn
	t.Cleanup(func() {
		lookupIPs, dialContextFn = oldLookup, oldDial
	})
	lookupIPs = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.1.2.3"), net.ParseIP("192.168.9.9")}, nil
	}
	var dialed string
	dialContextFn = func(_ context.Context, _ string, addr string) (net.Conn, error) {
		dialed = addr
		return nil, errors.New("dial stopped")
	}
	dialed = ""
	_, err = dialPolicy(withDialScheme(context.Background(), "https"), "tcp", "releases.internal:443", urlPolicy{AllowedOrigins: []string{"https://releases.internal"}})
	if dialed != "10.1.2.3:443" {
		t.Fatalf("default https dialed %q err %v", dialed, err)
	}
	dialed = ""
	_, err = dialPolicy(withDialScheme(context.Background(), "https"), "tcp", "releases.internal:8443", urlPolicy{AllowedOrigins: []string{"http://releases.internal:8443"}})
	if dialed != "" || asOp(err) == nil || asOp(err).Code != apierr.SignatureInvalid {
		t.Fatalf("scheme mismatch dialed %q err %v", dialed, err)
	}
	dialed = ""
	_, err = dialPolicy(withDialScheme(context.Background(), "http"), "tcp", "releases.internal:80", urlPolicy{AllowedOrigins: []string{"http://releases.internal"}})
	if dialed != "10.1.2.3:80" {
		t.Fatalf("default http dialed %q err %v", dialed, err)
	}
	dialed = ""
	_, err = dialPolicy(context.Background(), "tcp", "releases.internal:443", urlPolicy{AllowedOrigins: []string{"https://releases.internal"}})
	if dialed != "" || asOp(err) == nil || asOp(err).Code != apierr.SignatureInvalid {
		t.Fatalf("missing scheme dialed %q err %v", dialed, err)
	}
	dialed = ""
	sink := HTTPSink{URL: "https://releases.internal:8443/res", AllowedOrigins: []string{"http://releases.internal:8443"}}
	_, _, err = sink.Submit(context.Background(), []byte(`{}`))
	// Submit maps a dial refusal onto STORE_UNAVAILABLE. The dial itself must not run.
	if dialed != "" || err == nil {
		t.Fatalf("sink scheme mismatch dialed %q err %v", dialed, err)
	}
	dialed = ""
	sink = HTTPSink{URL: "https://releases.internal/res", AllowedOrigins: []string{"https://releases.internal"}}
	_, _, err = sink.Submit(context.Background(), []byte(`{}`))
	if dialed != "10.1.2.3:443" {
		t.Fatalf("sink default port dialed %q err %v", dialed, err)
	}
}
