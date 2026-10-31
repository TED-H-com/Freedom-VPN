//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/google/uuid"
)

func TestAT06_BrowserBucketsUseAuthenticatedSessionRows(t *testing.T) {
	store := auth.PostgresStore{Pool: dbtest.New(t)}
	identity := auth.Identity{Issuer: "https://idp.example.invalid", ClientScope: "portal", Subject: "test-operator", Role: auth.Admin, MFA: true, AuthTime: time.Now().UTC(), Expires: time.Now().Add(time.Hour)}
	tokens := []string{uuid.NewString(), uuid.NewString()}
	var principals []auth.Principal
	for _, token := range tokens {
		if _, err := store.Login(t.Context(), identity, token, "", time.Hour, uuid.New()); err != nil {
			t.Fatal(err)
		}
		p, err := store.Session(t.Context(), token, time.Minute)
		if err != nil || p.SessionID == uuid.Nil {
			t.Fatal("authenticated session ID missing", err)
		}
		principals = append(principals, p)
	}
	if principals[0].ID != principals[1].ID || principals[0].Scope() != principals[1].Scope() || principals[0].SessionID == principals[1].SessionID {
		t.Fatal("sessions must share resource ownership but have distinct rate keys")
	}

	now := time.Now()
	setLimiterClock(t, func() time.Time { return now })
	c := config.Defaults()
	c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	for _, tc := range []struct {
		path  string
		burst int
	}{
		{"/api/v1/admin/nodes", 60},
		{"/api/v1/admin/nodes/" + uuid.NewString(), 30},
	} {
		t.Run(tc.path, func(t *testing.T) {
			sets, err := Compose()
			if err != nil {
				t.Fatal(err)
			}
			h, err := New(sets, Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(r *http.Request) (auth.Principal, error) {
				return store.Session(r.Context(), r.Header.Get("Test-Session"), time.Minute)
			})})
			if err != nil {
				t.Fatal(err)
			}
			invoke := func(token string) int {
				r := httptest.NewRequest(http.MethodGet, tc.path, nil)
				r.RemoteAddr = "192.0.2.41:443"
				r.Header.Set("Test-Session", token)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				return w.Code
			}
			for i := 0; i < tc.burst; i++ {
				if got := invoke(tokens[0]); got != http.StatusNotImplemented {
					t.Fatalf("first session request %d: %d", i+1, got)
				}
			}
			if got := invoke(tokens[0]); got != http.StatusTooManyRequests {
				t.Fatalf("first session past burst: %d", got)
			}
			if got := invoke(tokens[1]); got != http.StatusNotImplemented {
				t.Fatalf("second session of same admin shares exhausted bucket: %d", got)
			}
		})
	}
}
