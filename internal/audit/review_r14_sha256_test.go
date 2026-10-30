package audit

import (
	"bytes"
	"testing"
)

// TestReviewR14_SHA256SuffixIsNotAClientBypass records the redactor exemption.
// A name ending in _sha256 is not sensitive, so an allowlisted value under that
// name is stored in clear text. Diff still drops every key the caller did not
// allowlist, and token_hash stays redacted. Production before/after maps are
// server-built allowlists; this test does not find a request field that is copied in.
func TestReviewR14_SHA256SuffixIsNotAClientBypass(t *testing.T) {
	secret := "plaintext-secret-TEST-ONLY"
	for _, key := range []string{"access_token_sha256", "password_sha256", "token_hash_sha256", "token_sha256"} {
		if sensitive(key) {
			t.Fatalf("%s is sensitive; the _sha256 exemption did not apply", key)
		}
	}
	// The exemption requires a name longer than "_sha256". The exact key
	// loses its leading underscore and is then "sha256", which is not a
	// secret term, so it is not sensitive either. token_hash is.
	if !sensitive("token_hash") || !sensitive("access_token") || sensitive("_sha256") {
		t.Fatal("token_hash or access_token was exempt, or the exact key _sha256 was treated as sensitive")
	}

	before := map[string]any{}
	after := map[string]any{
		"password_sha256": secret,
		"token_hash":      []byte{0x01, 0x02, 0x03},
		"reason":          "operator reason",
		"evil_sha256":     secret,
	}
	_, kept, err := Diff(before, after, "password_sha256", "token_hash", "reason")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(kept, []byte(secret)) {
		t.Fatalf("allowlisted password_sha256 did not keep the plaintext: %s", kept)
	}
	if bytes.Contains(kept, []byte("token_hash")) || bytes.Contains(kept, []byte{0x01}) {
		t.Fatalf("token_hash value was written: %s", kept)
	}
	_, onlyReason, err := Diff(before, after, "reason")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(onlyReason, []byte(secret)) || bytes.Contains(onlyReason, []byte("evil_sha256")) || bytes.Contains(onlyReason, []byte("password_sha256")) {
		t.Fatalf("a key outside the allowlist was stored: %s", onlyReason)
	}

	redacted, ok := redact(map[string]any{
		"password_sha256": secret,
		"token_hash":      secret,
	}).(map[string]any)
	if !ok || redacted["password_sha256"] != secret || redacted["token_hash"] != "[REDACTED]" {
		t.Fatalf("redact: %#v", redacted)
	}
	pem := "-----BEGIN TEST-----\nabc\n-----END TEST-----"
	redactedPEM, ok := redact(map[string]any{"password_sha256": pem}).(map[string]any)
	if !ok || redactedPEM["password_sha256"] != "[REDACTED]" {
		t.Fatalf("PEM under a _sha256 name was not redacted by value: %#v", redactedPEM)
	}
}
