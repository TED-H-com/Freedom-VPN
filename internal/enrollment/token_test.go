package enrollment

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestENR02_TokenHashHexEncodingAndNoPlaintextLog(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	token := TokenPrefix + base64.RawURLEncoding.EncodeToString(secret)
	hash, err := tokenHash(token)
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32-byte SHA-256 digest, got %d bytes", len(hash))
	}
	gotHex := hex.EncodeToString(hash)
	wantSum := sha256.Sum256([]byte(token))
	wantHex := hex.EncodeToString(wantSum[:])
	if gotHex != wantHex {
		t.Fatalf("tokenHash hex=%s, want %s", gotHex, wantHex)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(gotHex) {
		t.Fatalf("tokenHash hex %q does not match ^[0-9a-f]{64}$", gotHex)
	}

	for _, bad := range []string{
		"",
		"wrong_prefix_" + base64.RawURLEncoding.EncodeToString(secret),
		TokenPrefix + "short",
		TokenPrefix + base64.StdEncoding.EncodeToString(secret),
	} {
		if _, err := tokenHash(bad); err == nil {
			t.Fatalf("expected invalid token %q to fail", bad)
		} else if bad != "" && strings.Contains(err.Error(), bad) {
			t.Fatalf("error message leaked invalid token input: %v", err)
		}
	}

	var buf bytes.Buffer
	s := &Service{cfg: Config{Logger: slog.New(slog.NewJSONHandler(&buf, nil))}}
	id := uuid.New()
	s.log(t.Context(), "enrollment created or replayed", id)
	if strings.Contains(buf.String(), token) || strings.Contains(buf.String(), TokenPrefix) {
		t.Fatalf("service log leaked token material: %s", buf.String())
	}
	if !strings.Contains(buf.String(), id.String()) {
		t.Fatalf("service log missing resource_id %s: %s", id, buf.String())
	}
}
