package httpapi

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// D-056: mechanical strict-server adaptation preserves digest-bound bytes.
func TestDNS03_ResolverResponsePreservesExactBytes(t *testing.T) {
	raw := []byte(`{"entries":["test.example.invalid"],"note":"<>&"}`)
	sum := sha256.Sum256(raw)
	etag := `"sha256:` + hex.EncodeToString(sum[:]) + `"`
	response := gen.GetResolverDocument200JSONResponse{Body: json.RawMessage(raw), Headers: gen.GetResolverDocument200ResponseHeaders{ETag: etag, CacheControl: "private, max-age=0"}}
	w := httptest.NewRecorder()
	if err := response.VisitGetResolverDocumentResponse(w); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(w.Body.Bytes(), raw) || w.Code != 200 || w.Header().Get("Content-Type") != "application/json" || w.Header().Get("ETag") != etag || w.Header().Get("Cache-Control") != "private, max-age=0" {
		t.Fatalf("canonical response changed: %s %v", w.Body.Bytes(), w.Header())
	}
}

// The new methods are intentionally unimplemented until the nodes owner wires
// them. Pin the existing apierr.Write fallback, auth, and request validation.
func TestAPI01_Revision5Stubs(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		agent                    bool
		status                   int
	}{
		{"list groups", "GET", "/api/v1/admin/groups", "", false, 501},
		{"create group", "POST", "/api/v1/admin/groups", `{"code":"standard","name":"Standard exits"}`, false, 501},
		{"bad group", "POST", "/api/v1/admin/groups", `{"code":"INVALID","name":"Standard exits"}`, false, 422},
		{"document", "GET", "/api/v1/agent/resolver-documents/01900000-0000-7000-8000-000000000056", "", true, 501},
	} {
		t.Run(tc.name, func(t *testing.T) {
			role := auth.Admin
			if tc.agent {
				role = auth.Agent
			}
			p := principal(role)
			r := request(tc.method, tc.path, tc.body, p)
			if tc.agent {
				r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			}
			w := assertStatus(t, fixture(t, p, tc.agent, nil), r, tc.status)
			if tc.status == 501 {
				var env apierr.Envelope
				if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Error.Code != apierr.NotImplemented || env.Error.Retryable {
					t.Fatalf("invalid stub error: %s", w.Body.Bytes())
				}
			}
		})
	}
}
