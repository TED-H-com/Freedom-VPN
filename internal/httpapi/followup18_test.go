package httpapi

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

func followup18Login(burst, principals int, proxies []string) (http.Handler, *int) {
	c := config.Defaults()
	c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: burst}
	c.MaxRatePrincipals = principals
	c.TrustedProxies = proxies
	return reviewLoginHandler(c)
}

func followup18Limited(t *testing.T, w *httptest.ResponseRecorder, message string) {
	t.Helper()
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" || !strings.Contains(w.Body.String(), `"code":"RATE_LIMITED"`) || !strings.Contains(w.Body.String(), message) {
		t.Fatalf("status=%d retry=%q body=%s", w.Code, w.Header().Get("Retry-After"), w.Body.String())
	}
}

func followup18Allowed(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code == http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// TestFollowup18_ConnectionPrincipal locks r12 §4: X-Forwarded-For selects a
// bucket only when the TCP peer is a configured trusted proxy. The walk starts
// at the right and skips trusted hops.
func TestFollowup18_ConnectionPrincipal(t *testing.T) {
	t.Run("untrusted peer ignores X-Forwarded-For", func(t *testing.T) {
		h, hits := followup18Login(1, 100, nil)
		first := reviewLogin(h, "192.0.2.8:1000", "/auth/login", map[string]string{
			"X-Forwarded-For": "198.51.100.9",
			"X-Real-IP":       "198.51.100.10",
			"Forwarded":       "for=198.51.100.11",
		})
		if first.Code != http.StatusNoContent {
			t.Fatal(first.Code, first.Body.String())
		}
		second := reviewLogin(h, "192.0.2.8:2000", "/auth/login", map[string]string{
			"X-Forwarded-For": "203.0.113.4",
		})
		followup18Limited(t, second, "Login request rate exceeded.")
		other := reviewLogin(h, "192.0.2.9:1000", "/auth/login", map[string]string{
			"X-Forwarded-For": "198.51.100.9",
		})
		if other.Code != http.StatusNoContent {
			t.Fatal(other.Code, other.Body.String())
		}
		if *hits != 2 {
			t.Fatalf("handler hits=%d", *hits)
		}
	})

	t.Run("one trusted hop keys the client", func(t *testing.T) {
		h, hits := followup18Login(1, 100, []string{"192.0.2.8/32"})
		proxy := "192.0.2.8:443"
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.9"}); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.10"}); w.Code != http.StatusNoContent {
			t.Fatal("second client shared the first client's bucket", w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.9"}), "Login request rate exceeded.")
		// A direct connection from that client shares the forwarded bucket.
		followup18Limited(t, reviewLogin(h, "198.51.100.9:9", "/auth/login", nil), "Login request rate exceeded.")
		if w := reviewLogin(h, "[::ffff:192.0.2.8]:443", "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.40"}); w.Code != http.StatusNoContent {
			t.Fatal("ipv4-mapped proxy peer did not match the ipv4 CIDR", w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, "[::ffff:192.0.2.8]:444", "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.40"}), "Login request rate exceeded.")
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.7:443"}); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, "198.51.100.7:9", "/auth/login", nil), "Login request rate exceeded.")
		if *hits != 4 {
			t.Fatalf("handler hits=%d", *hits)
		}
	})

	t.Run("two trusted hops ignore a spoofed left entry", func(t *testing.T) {
		h, _ := followup18Login(1, 100, []string{"10.0.0.0/8"})
		proxy := "10.1.2.3:443"
		header := map[string]string{"X-Forwarded-For": "203.0.113.50, 198.51.100.20, 10.9.9.9"}
		if w := reviewLogin(h, proxy, "/auth/login", header); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", header), "Login request rate exceeded.")
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "203.0.113.50"}); w.Code != http.StatusNoContent {
			t.Fatal("spoofed leftmost entry was the bucket", w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.20"}), "Login request rate exceeded.")
	})

	t.Run("a later header line is further right", func(t *testing.T) {
		h, _ := followup18Login(1, 100, []string{"10.0.0.0/8"})
		send := func(lines ...string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
			r.RemoteAddr = "10.1.2.3:443"
			for _, line := range lines {
				r.Header.Add("X-Forwarded-For", line)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
		// Header.Get would return only the first line and key the spoofed address.
		if w := send("203.0.113.77", "198.51.100.30, 10.9.9.9"); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, send("203.0.113.77", "198.51.100.30, 10.9.9.9"), "Login request rate exceeded.")
		if w := send("203.0.113.77"); w.Code != http.StatusNoContent {
			t.Fatal("first header line was the bucket", w.Code, w.Body.String())
		}
		followup18Limited(t, send("198.51.100.30"), "Login request rate exceeded.")
		got := rightmostForwarded([]string{"203.0.113.77", "198.51.100.30, 10.9.9.9"}, maxForwardedHops)
		if len(got) != 3 || got[0] != "10.9.9.9" || got[1] != "198.51.100.30" || got[2] != "203.0.113.77" {
			t.Fatal(got)
		}
	})

	t.Run("only trusted hops keep the TCP peer", func(t *testing.T) {
		h, _ := followup18Login(1, 100, []string{"10.0.0.0/8", "192.0.2.0/24"})
		if w := reviewLogin(h, "10.0.0.1:443", "/auth/login", map[string]string{"X-Forwarded-For": "192.0.2.5, 10.0.0.2"}); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, "10.0.0.1:444", "/auth/login", map[string]string{"X-Forwarded-For": "192.0.2.6"}), "Login request rate exceeded.")
		if w := reviewLogin(h, "10.0.0.2:443", "/auth/login", map[string]string{"X-Forwarded-For": "192.0.2.5, 10.0.0.1"}); w.Code != http.StatusNoContent {
			t.Fatal("a second proxy address shared the first proxy bucket", w.Code, w.Body.String())
		}
	})

	t.Run("malformed chosen entry is 429 and occupies no slot", func(t *testing.T) {
		c := config.Defaults()
		c.LoginRate = config.Bucket{PerSecond: 1000, Burst: 1000}
		c.MaxRatePrincipals = 1
		c.TrustedProxies = []string{"192.0.2.8/32"}
		h, hits := reviewLoginHandler(c)
		proxy := "192.0.2.8:443"
		for _, header := range []map[string]string{
			{"X-Forwarded-For": "not-an-ip"},
			{"X-Forwarded-For": "198.51.100.1, not-an-ip"},
			{"X-Forwarded-For": "198.51.100.7,"},
		} {
			followup18Limited(t, reviewLogin(h, proxy, "/auth/login", header), "Login request rate exceeded.")
		}
		followup18Limited(t, reviewLogin(h, "not-an-address", "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.7"}), "Login request rate exceeded.")
		if *hits != 0 {
			t.Fatalf("handler hits=%d", *hits)
		}
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.7"}); w.Code != http.StatusNoContent {
			t.Fatal("malformed entries occupied the only principal slot", w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.8"}), "Login request rate exceeded.")
	})

	t.Run("ipv6 prefix and ipv4-mapped forms", func(t *testing.T) {
		h, _ := followup18Login(1, 100, []string{"192.0.2.8/32"})
		proxy := "192.0.2.8:443"
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "[2001:db8:9:1::1]:443"}); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "2001:db8:9:1::2"}), "Login request rate exceeded.")
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "[2001:db8:9:1::3]"}), "Login request rate exceeded.")
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "2001:db8:9:2::1"}); w.Code != http.StatusNoContent {
			t.Fatal("a second /64 shared the first prefix", w.Code, w.Body.String())
		}
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "2001:db8:1:2::5%eth0"}); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "[2001:db8:1:2::9%eth0]:443"}), "Login request rate exceeded.")
		if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "::ffff:192.0.2.10"}); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, reviewLogin(h, "192.0.2.10:9", "/auth/login", nil), "Login request rate exceeded.")
	})

	t.Run("Forwarded and X-Real-IP stay ignored", func(t *testing.T) {
		h, _ := followup18Login(1, 100, []string{"192.0.2.8/32"})
		header := map[string]string{
			"X-Real-IP": "198.51.100.9",
			"Forwarded": `for=198.51.100.9`,
		}
		if w := reviewLogin(h, "192.0.2.8:443", "/auth/login", header); w.Code != http.StatusNoContent {
			t.Fatal(w.Code, w.Body.String())
		}
		header["Forwarded"] = "for=198.51.100.10"
		header["X-Real-IP"] = "198.51.100.10"
		followup18Limited(t, reviewLogin(h, "192.0.2.8:444", "/auth/login", header), "Login request rate exceeded.")
		if w := reviewLogin(h, "198.51.100.9:9", "/auth/login", nil); w.Code != http.StatusNoContent {
			t.Fatal("Forwarded selected the bucket", w.Code, w.Body.String())
		}
	})
}

func TestFollowup18_TenThousandEntries(t *testing.T) {
	const total = 10000
	trustedHop := "10.0.0.1"
	client := "198.51.100.77"
	var rightClient strings.Builder
	rightClient.Grow(total * 12)
	for i := 0; i < total-1; i++ {
		if i > 0 {
			rightClient.WriteByte(',')
		}
		rightClient.WriteString(trustedHop)
	}
	rightClient.WriteByte(',')
	rightClient.WriteString(client)

	var pastWindow strings.Builder
	pastWindow.Grow(total * 16)
	pastWindow.WriteString("not-an-ip")
	for i := 1; i < total; i++ {
		pastWindow.WriteByte(',')
		switch {
		case i == total-65:
			pastWindow.WriteString("203.0.113.77")
		case i >= total-64:
			pastWindow.WriteString(trustedHop)
		default:
			pastWindow.WriteString("198.51.100.1")
		}
	}
	stopped := rightmostForwarded([]string{pastWindow.String()}, maxForwardedHops)
	if len(stopped) != maxForwardedHops {
		t.Fatalf("examined %d entries", len(stopped))
	}
	for _, entry := range stopped {
		if entry != trustedHop {
			t.Fatalf("entry past the 64-hop window was parsed: %s", entry)
		}
	}

	h, _ := followup18Login(1, 100, []string{"10.0.0.0/8"})
	proxy := "10.2.2.2:443"
	start := time.Now()
	if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": rightClient.String()}); w.Code != http.StatusNoContent {
		t.Fatal(w.Code, w.Body.String())
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("rightmost client among %d entries took %s", total, elapsed)
	} else {
		t.Logf("rightmost client among %d entries: %s", total, elapsed)
	}
	followup18Limited(t, reviewLogin(h, client+":9", "/auth/login", nil), "Login request rate exceeded.")

	start = time.Now()
	if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": pastWindow.String()}); w.Code != http.StatusNoContent {
		t.Fatal("65th entry from the right became the bucket", w.Code, w.Body.String())
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("64-hop walk of %d entries took %s", total, elapsed)
	} else {
		t.Logf("64-hop walk of %d entries: %s", total, elapsed)
	}
	followup18Limited(t, reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": pastWindow.String()}), "Login request rate exceeded.")
	if w := reviewLogin(h, proxy, "/auth/login", map[string]string{"X-Forwarded-For": "203.0.113.77"}); w.Code != http.StatusNoContent {
		t.Fatal("the entry beyond 64 hops was the bucket", w.Code, w.Body.String())
	}
}

func TestFollowup18_IngressAndAgentListener(t *testing.T) {
	t.Run("browser ingress", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		c.TrustedProxies = []string{"192.0.2.8/32"}
		c.IngressRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		c.BrowserRate = config.Bucket{PerSecond: 1000, Burst: 1000}
		p := principal(auth.Admin)
		h := fixtureConfig(t, p, false, nil, c)
		send := func(xff string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/nodes", nil)
			r.RemoteAddr = "192.0.2.8:443"
			r.Header.Set("X-Forwarded-For", xff)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
		if w := send("198.51.100.1"); w.Code != http.StatusNotImplemented {
			t.Fatal(w.Code, w.Body.String())
		}
		if w := send("198.51.100.2"); w.Code != http.StatusNotImplemented {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, send("198.51.100.1"), "Authentication request rate exceeded.")
	})

	t.Run("anonymous table", func(t *testing.T) {
		c := config.Defaults()
		c.TrustedProxies = []string{"192.0.2.8/32"}
		c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}
		c.AnonymousRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		sets, err := Compose()
		if err != nil {
			t.Fatal(err)
		}
		h, err := New(sets, Options{Config: c})
		if err != nil {
			t.Fatal(err)
		}
		send := func(xff string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", strings.NewReader(`{}`))
			r.RemoteAddr = "192.0.2.8:443"
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Forwarded-For", xff)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
		followup18Allowed(t, send("198.51.100.1"))
		followup18Allowed(t, send("198.51.100.2"))
		w := send("198.51.100.1")
		followup18Limited(t, w, `"message":"Request rate exceeded."`)
		if strings.Contains(w.Body.String(), "Authentication request rate exceeded.") {
			t.Fatal(w.Body.String())
		}
	})

	t.Run("agent listener ignores the header", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		c.TrustedProxies = []string{"10.0.0.0/8"}
		c.IngressRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		c.LeaseRate = config.Bucket{PerSecond: 1000, Burst: 1000}
		p := principal(auth.Agent)
		h := fixtureConfig(t, p, true, nil, c)
		body := fmt.Sprintf(`{"renewal_request_id":%q,"node_id":%q,"node_incarnation_id":%q,"current_lease_id":null,"applied_config_generation":null}`,
			"019a0000-0000-7000-8000-000000000001", p.NodeID, p.IncarnationID)
		send := func(remote, xff string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/lease-renewals", strings.NewReader(body))
			r.RemoteAddr = remote
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Forwarded-For", xff)
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
		if w := send("10.0.0.1:443", "198.51.100.1"); w.Code != http.StatusNotImplemented {
			t.Fatal(w.Code, w.Body.String())
		}
		followup18Limited(t, send("10.0.0.1:443", "198.51.100.2"), "Authentication request rate exceeded.")
		if w := send("10.0.0.2:443", "198.51.100.3"); w.Code != http.StatusNotImplemented {
			t.Fatal("a second agent peer shared the proxy bucket via the header", w.Code, w.Body.String())
		}
		nets, err := config.ParseTrustedProxies(c.TrustedProxies)
		if err != nil {
			t.Fatal(err)
		}
		key, ok := requestRateKey("10.0.0.1:443", []string{"198.51.100.9"}, nets, false)
		if !ok || key != "10.0.0.1" {
			t.Fatalf("agent principal=%q ok=%v", key, ok)
		}
	})

	t.Run("subject key stays the node", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		c.TrustedProxies = []string{"10.0.0.0/8"}
		c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}
		c.LeaseRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
		p := principal(auth.Agent)
		h := fixtureConfig(t, p, true, nil, c)
		body := fmt.Sprintf(`{"renewal_request_id":%q,"node_id":%q,"node_incarnation_id":%q,"current_lease_id":null,"applied_config_generation":null}`,
			"019a0000-0000-7000-8000-000000000002", p.NodeID, p.IncarnationID)
		send := func(remote, xff string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/lease-renewals", strings.NewReader(body))
			r.RemoteAddr = remote
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Forwarded-For", xff)
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w
		}
		if w := send("10.0.0.1:443", "198.51.100.1"); w.Code != http.StatusNotImplemented {
			t.Fatal(w.Code, w.Body.String())
		}
		w := send("10.0.0.1:443", "198.51.100.2")
		followup18Limited(t, w, `"message":"Request rate exceeded."`)
		if strings.Contains(w.Body.String(), "Authentication request rate exceeded.") {
			t.Fatal(w.Body.String())
		}
		followup18Limited(t, send("10.0.0.2:443", "198.51.100.3"), `"message":"Request rate exceeded."`)
	})
}

func TestFollowup18_TrustedProxyConfig(t *testing.T) {
	base := config.Defaults()
	base.DatabaseURLFile = "secrets/database"
	for _, cidr := range []string{"0.0.0.0/0", "::/0", "10.0.0.0/7", "2001:db8::/31"} {
		c := base
		c.TrustedProxies = []string{cidr}
		err := c.Validate()
		if err == nil || !strings.Contains(err.Error(), "trusting the world") || !strings.Contains(err.Error(), cidr) {
			t.Fatalf("%s: %v", cidr, err)
		}
	}
	c := base
	c.TrustedProxies = []string{"10.0.0.0/8", "::/32", "192.0.2.9/32", "192.0.2.9/32"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, cidr := range []string{"192.0.2.1", "::ffff:192.0.2.1/128"} {
		bad := base
		bad.TrustedProxies = []string{cidr}
		if err := bad.Validate(); err == nil {
			t.Fatalf("%s accepted", cidr)
		}
	}
	rejected := base
	rejected.TrustedProxies = []string{"0.0.0.0/0"}
	sets, err := Compose()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = New(sets, Options{Config: rejected}); err == nil || !strings.Contains(err.Error(), "trusting the world") {
		t.Fatal(err)
	}
	// A handler built without startup validation keeps the TCP peer.
	rejected.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
	h, _ := reviewLoginHandler(rejected)
	if w := reviewLogin(h, "192.0.2.8:443", "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.9"}); w.Code != http.StatusNoContent {
		t.Fatal(w.Code, w.Body.String())
	}
	followup18Limited(t, reviewLogin(h, "192.0.2.8:444", "/auth/login", map[string]string{"X-Forwarded-For": "198.51.100.10"}), "Login request rate exceeded.")
}
