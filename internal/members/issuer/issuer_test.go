package issuer

import "testing"

func TestD052_F09CanonicalIssuer(t *testing.T) {
	for _, tc := range []struct{ in, out string }{
		{"HTTPS://ISSUER.EXAMPLE.INVALID:443/", "https://issuer.example.invalid"},
		{"http://ISSUER.EXAMPLE.INVALID:80/realm/", "http://issuer.example.invalid/realm"},
		{"https://[2001:db8::1]:443/", "https://[2001:db8::1]"},
		{"https://issuer.example.invalid:8443/realm", "https://issuer.example.invalid:8443/realm"},
	} {
		got, err := Canonical(tc.in)
		if err != nil || got != tc.out {
			t.Fatalf("%q: %q %v", tc.in, got, err)
		}
		if Validate(got) != nil {
			t.Fatal("canonical issuer rejected")
		}
		if tc.in != got && Validate(tc.in) == nil {
			t.Fatal("noncanonical config accepted")
		}
	}
	for _, raw := range []string{"", "issuer.example.invalid", "https://issuer.example.invalid?", "https://issuer.example.invalid/#", "https://user@issuer.example.invalid", "https://issuer.example.invalid:", "file:///issuer", " https://issuer.example.invalid"} {
		if _, err := Canonical(raw); err == nil {
			t.Fatal("unsafe issuer accepted", raw)
		}
	}
}
