package issuer

import (
	"fmt"
	"testing"
)

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

// TestReviewR9_CanonicalIssuerTable ports the r9 issuer table. Rows the old
// auth helper echoed unchanged are ErrCanonical here; a trailing %2F is kept.
func TestReviewR9_CanonicalIssuerTable(t *testing.T) {
	cases := []struct {
		in, want string
		reject   bool
	}{
		{"https://idp.example", "https://idp.example", false},
		{"https://idp.example/", "https://idp.example", false},
		{"https://IDP.example:443", "https://idp.example", false},
		{"https://idp.example:443/", "https://idp.example", false},
		{"https://idp.example//", "https://idp.example", false},
		{"https://idp.example/realm/", "https://idp.example/realm", false},
		{"https://idp.example/realm", "https://idp.example/realm", false},
		{"http://idp.example:443", "http://idp.example:443", false},
		{"http://idp.example", "http://idp.example", false},
		{"http://idp.example:80/", "http://idp.example", false},
		{"https://user@idp.example", "", true},
		{"https://idp.example?x=1", "", true},
		{"https://idp.example#x", "", true},
		{"https://idp.example.evil.com", "https://idp.example.evil.com", false},
		{"https://idp.example/a%2F", "https://idp.example/a%2F", false},
		{"https://idp.example/a%2Fb", "https://idp.example/a%2Fb", false},
		{"https://idp.example/%52ealm", "https://idp.example/%52ealm", false},
		{"https://idp.example/Realm", "https://idp.example/Realm", false},
		{"https://bücher.example/realm/", "https://b%C3%BCcher.example/realm", false},
		{"https://xn--bcher-kva.example/realm", "https://xn--bcher-kva.example/realm", false},
		{"https://idp.example.", "https://idp.example.", false},
	}
	var table string
	for _, tc := range cases {
		got, err := Canonical(tc.in)
		if tc.reject {
			table += fmt.Sprintf("%s => ErrCanonical\n", tc.in)
			if err != ErrCanonical || got != "" {
				t.Errorf("Canonical(%q)=%q %v want ErrCanonical", tc.in, got, err)
			}
			if Validate(tc.in) == nil {
				t.Errorf("Validate accepted %q", tc.in)
			}
			continue
		}
		table += fmt.Sprintf("%s => %s\n", tc.in, got)
		if err != nil || got != tc.want {
			t.Errorf("Canonical(%q)=%q %v want %q", tc.in, got, err, tc.want)
		}
		if Validate(tc.want) != nil {
			t.Errorf("Validate rejected canonical %q", tc.want)
		}
		if tc.in != tc.want && Validate(tc.in) == nil {
			t.Errorf("Validate accepted noncanonical %q", tc.in)
		}
	}
	t.Log("\n" + table)

	bare, err := Canonical("https://idp.example")
	if err != nil {
		t.Fatal(err)
	}
	mustDiffer := []string{
		"http://idp.example",
		"http://idp.example:443",
		"https://user@idp.example",
		"https://idp.example?x=1",
		"https://idp.example#x",
		"https://idp.example/realm",
		"https://idp.example/realm/",
		"https://idp.example.evil.com",
		"https://other.example",
		"https://idp.example.",
		"https://user:secret@idp.example/",
	}
	for _, raw := range mustDiffer {
		got, cerr := Canonical(raw)
		if cerr != nil {
			if cerr != ErrCanonical {
				t.Fatalf("Canonical(%q) unexpected error %v", raw, cerr)
			}
			continue
		}
		if got == bare {
			t.Fatalf("%q collapses onto %q", raw, bare)
		}
	}
	realm, err := Canonical("https://idp.example/realm")
	if err != nil {
		t.Fatal(err)
	}
	other, err := Canonical("https://idp.example/other")
	if err != nil {
		t.Fatal(err)
	}
	realmCase, err := Canonical("https://idp.example/Realm")
	if err != nil {
		t.Fatal(err)
	}
	if other == realm || realmCase == realm {
		t.Fatal("distinct issuer paths collapsed")
	}
	encoded, err := Canonical("https://idp.example/a%2F")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Canonical("https://idp.example/a")
	if err != nil {
		t.Fatal(err)
	}
	if encoded == plain {
		t.Fatal("percent-encoded slash collapsed onto the trimmed path")
	}
	unicodeHost, uerr := Canonical("https://bücher.example")
	puny, perr := Canonical("https://xn--bcher-kva.example")
	if uerr == nil && perr == nil && unicodeHost == puny {
		t.Log("IDN unicode and punycode collapse")
	} else {
		t.Log("IDN unicode and punycode stay distinct")
	}
}

func TestCanonical_LiteralSlashAfterEncodedBytes(t *testing.T) {
	for _, tc := range []struct{ in, out string }{
		{"https://idp.example/a%2F/", "https://idp.example/a%2F"},
		{"https://idp.example/%2F", "https://idp.example/%2F"},
		{"https://idp.example/%2F/", "https://idp.example/%2F"},
		{"https://idp.example/%52ealm/", "https://idp.example/%52ealm"},
		{"https://[2001:db8::1]:443/a%2F", "https://[2001:db8::1]/a%2F"},
		{"https://idp.example/a%2f", "https://idp.example/a%2f"},
		{"https://idp.example/a%2F%2F", "https://idp.example/a%2F%2F"},
		{"https://idp.example/foo%2Fbar/", "https://idp.example/foo%2Fbar"},
		{"https://idp.example/foo%2Fbar//", "https://idp.example/foo%2Fbar"},
	} {
		got, err := Canonical(tc.in)
		if err != nil || got != tc.out {
			t.Fatalf("%q: %q %v", tc.in, got, err)
		}
		if Validate(got) != nil {
			t.Fatalf("canonical form rejected: %q", got)
		}
	}
}
