package auth

import (
	"fmt"
	"testing"
)

func TestReviewR9_CanonicalIssuerTable(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://idp.example", "https://idp.example"},
		{"https://idp.example/", "https://idp.example"},
		{"https://IDP.example:443", "https://idp.example"},
		{"https://idp.example:443/", "https://idp.example"},
		{"https://idp.example//", "https://idp.example"},
		{"https://idp.example/realm/", "https://idp.example/realm"},
		{"https://idp.example/realm", "https://idp.example/realm"},
		{"http://idp.example:443", "http://idp.example:443"},
		{"http://idp.example", "http://idp.example"},
		{"http://idp.example:80/", "http://idp.example"},
		{"https://user@idp.example", "https://user@idp.example"},
		{"https://idp.example?x=1", "https://idp.example?x=1"},
		{"https://idp.example#x", "https://idp.example#x"},
		{"https://idp.example.evil.com", "https://idp.example.evil.com"},
		{"https://idp.example/a%2F", "https://idp.example/a"},
		{"https://idp.example/a%2Fb", "https://idp.example/a%2Fb"},
		{"https://idp.example/%52ealm", "https://idp.example/%52ealm"},
		{"https://idp.example/Realm", "https://idp.example/Realm"},
		{"https://bücher.example/realm/", "https://b%C3%BCcher.example/realm"},
		{"https://xn--bcher-kva.example/realm", "https://xn--bcher-kva.example/realm"},
		{"https://idp.example.", "https://idp.example."},
	}
	var table string
	for _, tc := range cases {
		got := canonicalIssuer(tc.in)
		table += fmt.Sprintf("%s => %s\n", tc.in, got)
		if got != tc.want {
			t.Errorf("canonicalIssuer(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	t.Log("\n" + table)

	bare := canonicalIssuer("https://idp.example")
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
		if canonicalIssuer(raw) == bare {
			t.Fatalf("TODO(review): %q collapses onto %q", raw, bare)
		}
	}
	realm := canonicalIssuer("https://idp.example/realm")
	if canonicalIssuer("https://idp.example/other") == realm || canonicalIssuer("https://idp.example/Realm") == realm {
		t.Fatal("TODO(review): distinct issuer paths collapsed")
	}
	if canonicalIssuer("https://idp.example/a%2F") == canonicalIssuer("https://idp.example/a") {
		t.Fatal("TODO(review): percent-encoded slash collapsed onto the trimmed path")
	}
	if canonicalIssuer("https://bücher.example") == canonicalIssuer("https://xn--bcher-kva.example") {
		t.Log("IDN unicode and punycode collapse")
	} else {
		t.Log("IDN unicode and punycode stay distinct")
	}
}
