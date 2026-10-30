package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

func reviewLoginHandler(c config.Config) (http.Handler, *int) {
	hits := 0
	h := BrowserRoutes(c, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusNoContent)
	}))
	return h, &hits
}

func reviewLogin(h http.Handler, remote, path string, header map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.RemoteAddr = remote
	for k, v := range header {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestReviewR9_LoginLimiterAddressIdentityBoundsAndRoutes(t *testing.T) {
	t.Run("remote address only", func(t *testing.T) {
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		h, hits := reviewLoginHandler(c)
		first := reviewLogin(h, "192.0.2.8:1000", "/auth/login?subject=secret-subject-do-not-store", map[string]string{
			"X-Forwarded-For": "198.51.100.9",
			"X-Real-IP":       "198.51.100.10",
			"Forwarded":       "for=198.51.100.11",
		})
		if first.Code != http.StatusNoContent {
			t.Fatal(first.Code, first.Body.String())
		}
		second := reviewLogin(h, "192.0.2.8:2000", "/auth/login?subject=other-secret", map[string]string{
			"X-Forwarded-For": "203.0.113.4",
			"Authorization":   "Bearer claimed-user",
		})
		if second.Code != http.StatusTooManyRequests {
			t.Fatal("forwarded header or source port minted a bucket", second.Code, second.Body.String())
		}
		if strings.Contains(second.Body.String(), "secret-subject") || strings.Contains(second.Body.String(), "claimed-user") {
			t.Fatal(second.Body.String())
		}
		retry, err := strconv.Atoi(second.Header().Get("Retry-After"))
		if err != nil || retry < 1 {
			t.Fatalf("Retry-After=%q err=%v", second.Header().Get("Retry-After"), err)
		}
		other := reviewLogin(h, "192.0.2.9:1000", "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.9"})
		if other.Code != http.StatusNoContent {
			t.Fatal("forwarded header was trusted", other.Code, other.Body.String())
		}
		if *hits != 2 {
			t.Fatal("limited request reached the login handler", *hits)
		}
	})

	t.Run("identity claim does not bypass the address bucket", func(t *testing.T) {
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		h, _ := reviewLoginHandler(c)
		passed := 0
		for i := 0; i < 6; i++ {
			w := reviewLogin(h, "192.0.2.20:9", fmt.Sprintf("/auth/login?login_hint=user-%d&subject=user-%d", i, i), map[string]string{
				"Authorization": fmt.Sprintf("Bearer user-%d", i),
			})
			if w.Code == http.StatusNoContent {
				passed++
			}
		}
		if passed != 1 {
			t.Fatalf("TODO(review): one address completed %d login requests by varying the pre-auth identity", passed)
		}
	})

	t.Run("ipv6 prefix", func(t *testing.T) {
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		c.MaxRatePrincipals = 100
		h, _ := reviewLoginHandler(c)
		passed := 0
		for i := 1; i <= 8; i++ {
			w := reviewLogin(h, fmt.Sprintf("[2001:db8:0:1::%x]:1234", i), "/auth/login", nil)
			if w.Code == http.StatusNoContent {
				passed++
			}
		}
		if passed > 1 {
			t.Fatalf("TODO(review): %d distinct addresses in 2001:db8:0:1::/64 each received a full login bucket; SplitHostPort keeps the /128 and MaxRatePrincipals is %d", passed, c.MaxRatePrincipals)
		}
	})

	t.Run("principal cap and unbracketed ipv6", func(t *testing.T) {
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		c.MaxRatePrincipals = 3
		h, _ := reviewLoginHandler(c)
		for i := 1; i <= 3; i++ {
			if w := reviewLogin(h, fmt.Sprintf("192.0.2.%d:9", i), "/auth/login", nil); w.Code != http.StatusNoContent {
				t.Fatal(i, w.Code, w.Body.String())
			}
		}
		if w := reviewLogin(h, "192.0.2.40:9", "/auth/login", nil); w.Code != http.StatusTooManyRequests {
			t.Fatal("fresh key inserted past MaxRatePrincipals", w.Code)
		}
		bare := config.Defaults()
		bare.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		hb, _ := reviewLoginHandler(bare)
		if reviewLogin(hb, "2001:db8::1:1234", "/auth/login", nil).Code != http.StatusNoContent {
			t.Fatal("unbracketed ipv6")
		}
		if w := reviewLogin(hb, "2001:db8::2:1234", "/auth/login", nil); w.Code != http.StatusTooManyRequests {
			t.Fatal("unbracketed ipv6 did not share one key", w.Code)
		}
	})

	t.Run("login routes share a bucket", func(t *testing.T) {
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 2}
		h, _ := reviewLoginHandler(c)
		for _, path := range []string{"/auth/login", "/auth/login?role=admin"} {
			if w := reviewLogin(h, "192.0.2.50:9", path, nil); w.Code != http.StatusNoContent {
				t.Fatal(path, w.Code, w.Body.String())
			}
		}
		if w := reviewLogin(h, "192.0.2.50:9", "/auth/callback?code=x&state=y", nil); w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "RATE_LIMITED") {
			t.Fatal("callback", w.Code, w.Body.String(), w.Header().Get("Retry-After"))
		}
		if w := reviewLogin(h, "192.0.2.50:9", "/auth/session", nil); w.Code != http.StatusNoContent {
			t.Fatal("session class shared the login bucket", w.Code, w.Body.String())
		}
		if w := reviewLogin(h, "192.0.2.50:9", "/auth/logout", nil); w.Code != http.StatusNoContent {
			t.Fatal("logout class", w.Code, w.Body.String())
		}
	})
}
