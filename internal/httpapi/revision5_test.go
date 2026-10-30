package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

type resolverNotModifiedResponse struct {
	etag string
}

func (r resolverNotModifiedResponse) VisitGetResolverDocumentResponse(w http.ResponseWriter) error {
	if r.etag != "" {
		w.Header().Set("ETag", r.etag)
	}
	w.WriteHeader(http.StatusNotModified)
	return nil
}

type stubResolverHandler struct {
	fn func(context.Context, gen.GetResolverDocumentRequestObject) (gen.GetResolverDocumentResponseObject, error)
}

func (s stubResolverHandler) GetResolverDocument(ctx context.Context, req gen.GetResolverDocumentRequestObject) (gen.GetResolverDocumentResponseObject, error) {
	return s.fn(ctx, req)
}

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

func TestDNS03_ResolverDocumentTransportCacheHeaders(t *testing.T) {
	raw := []byte(`{"entries":["test.example.invalid"],"note":"<>&"}`)
	sum := sha256.Sum256(raw)
	etag := `"sha256:` + hex.EncodeToString(sum[:]) + `"`
	docPath := "/api/v1/agent/resolver-documents/01900000-0000-7000-8000-000000000056"
	p := principal(auth.Agent)

	t.Run("200_private_max_age_0", func(t *testing.T) {
		h := fixture(t, p, true, nil, stubResolverHandler{
			fn: func(context.Context, gen.GetResolverDocumentRequestObject) (gen.GetResolverDocumentResponseObject, error) {
				return gen.GetResolverDocument200JSONResponse{
					Body:    json.RawMessage(raw),
					Headers: gen.GetResolverDocument200ResponseHeaders{ETag: etag, CacheControl: "private, max-age=0"},
				}, nil
			},
		})
		r := request("GET", docPath, "", p)
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := assertStatus(t, h, r, http.StatusOK)
		if got := w.Header().Get("Cache-Control"); got != "private, max-age=0" {
			t.Fatalf("200 Cache-Control = %q, want %q", got, "private, max-age=0")
		}
		if got := w.Header().Get("ETag"); got != etag {
			t.Fatalf("200 ETag = %q, want %q", got, etag)
		}
		if !bytes.Equal(w.Body.Bytes(), raw) {
			t.Fatalf("200 body = %q, want %q", w.Body.Bytes(), raw)
		}
	})

	t.Run("304_no_store", func(t *testing.T) {
		h := fixture(t, p, true, nil, stubResolverHandler{
			fn: func(context.Context, gen.GetResolverDocumentRequestObject) (gen.GetResolverDocumentResponseObject, error) {
				return resolverNotModifiedResponse{etag: etag}, nil
			},
		})
		r := request("GET", docPath, "", p)
		r.Header.Set("If-None-Match", etag)
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := assertStatus(t, h, r, http.StatusNotModified)
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("304 Cache-Control = %q, want %q", got, "no-store")
		}
	})

	t.Run("error_no_store", func(t *testing.T) {
		h := fixture(t, p, true, nil, stubResolverHandler{
			fn: func(context.Context, gen.GetResolverDocumentRequestObject) (gen.GetResolverDocumentResponseObject, error) {
				return nil, apierr.New(apierr.ResourceNotFound, "Resolver document not found.")
			},
		})
		r := request("GET", docPath, "", p)
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := assertStatus(t, h, r, http.StatusNotFound)
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("error Cache-Control = %q, want %q", got, "no-store")
		}
	})
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
