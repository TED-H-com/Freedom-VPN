package gen_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// The composition contract stays fail-closed until the members owner registers
// RevokeMemberGrant. Exercise the generated route through the real middleware.
func TestAT29_RevokeMemberGrantCompositionStub(t *testing.T) {
	for _, tc := range []struct {
		name            string
		role            auth.Role
		csrf, key, body string
		status          int
	}{
		{"admin stub", auth.Admin, "TEST ONLY csrf", "01000000-0000-4000-8000-000000000014", `{"reason":"TEST ONLY withdrawal"}`, 501},
		{"member forbidden", auth.Member, "TEST ONLY csrf", "01000000-0000-4000-8000-000000000014", `{"reason":"TEST ONLY withdrawal"}`, 403},
		{"missing csrf", auth.Admin, "", "01000000-0000-4000-8000-000000000014", `{"reason":"TEST ONLY withdrawal"}`, 403},
		{"missing key", auth.Admin, "TEST ONLY csrf", "", `{"reason":"TEST ONLY withdrawal"}`, 400},
		{"missing reason", auth.Admin, "TEST ONLY csrf", "01000000-0000-4000-8000-000000000014", `{}`, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := config.Defaults()
			c.PublicOrigin = "https://portal.example.invalid"
			p := auth.Principal{Role: tc.role, ID: uuid.New(), MFAVerified: true, CSRFToken: "TEST ONLY csrf", AuthenticatedAt: time.Now()}
			composite, err := httpapi.Compose()
			if err != nil {
				t.Fatal(err)
			}
			h, err := httpapi.New(composite, httpapi.Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return p, nil })})
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/api/v1/admin/members/01000000-0000-4000-8000-000000000001/grants/01000000-0000-4000-8000-000000000029/revoke", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", c.PublicOrigin)
			if tc.csrf != "" {
				r.Header.Set("X-CSRF-Token", tc.csrf)
			}
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			w := httptest.NewRecorder()
			httpapi.Transport(nil, h).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if tc.status == 501 {
				var env apierr.Envelope
				if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Error.Code != apierr.NotImplemented || env.Error.Retryable {
					t.Fatalf("invalid fallback: %s", w.Body.String())
				}
			}
		})
	}
}
