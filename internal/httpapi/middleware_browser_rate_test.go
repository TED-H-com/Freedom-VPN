package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/google/uuid"
)

// TestFollowup21_BrowserClassCalibration pins the operator subject budget.
// Audit paging, incarnation history, and node and member detail share class
// browser at 10/s burst 30. The 31st request on one session is 429
// RATE_LIMITED with Retry-After. A second session on the same address still
// has its own burst. Ingress stays 20/s burst 60, so this refusal is the
// subject bucket.
func TestFollowup21_BrowserClassCalibration(t *testing.T) {
	d := config.Defaults()
	if d.BrowserRate != (config.Bucket{PerSecond: 10, Burst: 30}) {
		t.Fatalf("browser_rate = %+v, want 10/s burst 30", d.BrowserRate)
	}
	if d.IngressRate != (config.Bucket{PerSecond: 20, Burst: 60}) {
		t.Fatalf("ingress_rate = %+v, want 20/s burst 60", d.IngressRate)
	}
	if d.BrowserRate.PerSecond*2 != d.IngressRate.PerSecond || d.BrowserRate.Burst*2 != d.IngressRate.Burst {
		t.Fatalf("browser budget is not half of ingress: browser %+v ingress %+v", d.BrowserRate, d.IngressRate)
	}
	held := []struct {
		name string
		got  config.Bucket
		want config.Bucket
	}{
		{"agent", d.AgentRate, config.Bucket{PerSecond: 5, Burst: 10}},
		{"heartbeat", d.HeartbeatRate, config.Bucket{PerSecond: 1, Burst: 5}},
		{"lease", d.LeaseRate, config.Bucket{PerSecond: 1, Burst: 5}},
		{"usage", d.UsageRate, config.Bucket{PerSecond: 2, Burst: 10}},
		{"anonymous", d.AnonymousRate, config.Bucket{PerSecond: 2, Burst: 10}},
	}
	for _, tc := range held {
		if tc.got != tc.want {
			t.Fatalf("%s rate = %+v, want %+v", tc.name, tc.got, tc.want)
		}
	}
	wantClasses := []string{"browser", "agent", "heartbeat", "lease", "usage", "anonymous"}
	if len(rateClasses) != len(wantClasses) {
		t.Fatalf("rate classes = %q", rateClasses)
	}
	for i, class := range wantClasses {
		if rateClasses[i] != class {
			t.Fatalf("rate classes = %q", rateClasses)
		}
	}

	doc, err := gen.GetSpec()
	if err != nil {
		t.Fatal(err)
	}
	doc.Servers = nil
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	nodeID := uuid.NewString()
	memberID := uuid.NewString()
	paths := []string{
		"/api/v1/admin/audit-events",
		"/api/v1/admin/nodes/" + nodeID + "/incarnations",
		"/api/v1/admin/nodes/" + nodeID,
		"/api/v1/admin/members/" + memberID,
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		route, _, err := findRoute(router, req)
		if err != nil {
			t.Fatal(path, err)
		}
		class, bucket, _ := classify(route.Operation, d)
		if class != "browser" || bucket != d.BrowserRate {
			t.Fatalf("%s OperationID=%s class=%s bucket=%+v", path, route.Operation.OperationID, class, bucket)
		}
	}
	for _, tc := range []struct {
		method, path, class string
		bucket              config.Bucket
	}{
		{http.MethodPost, "/api/v1/agent/enroll", "anonymous", d.AnonymousRate},
		{http.MethodPost, "/api/v1/agent/observations", "heartbeat", d.HeartbeatRate},
		{http.MethodPost, "/api/v1/agent/lease-renewals", "lease", d.LeaseRate},
		{http.MethodPost, "/api/v1/agent/usage-batches", "usage", d.UsageRate},
		{http.MethodGet, "/api/v1/agent/desired-state", "agent", d.AgentRate},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		route, _, err := findRoute(router, req)
		if err != nil {
			t.Fatal(tc.path, err)
		}
		class, bucket, _ := classify(route.Operation, d)
		if class != tc.class || bucket != tc.bucket {
			t.Fatalf("%s class=%s bucket=%+v, want %s %+v", tc.path, class, bucket, tc.class, tc.bucket)
		}
	}

	now := time.Date(2026, 10, 30, 19, 41, 0, 0, time.UTC)
	setLimiterClock(t, func() time.Time { return now })
	c := d
	c.PublicOrigin = "https://portal.example.invalid"
	p, other := principal(auth.Admin), principal(auth.Admin)
	sets, err := Compose()
	if err != nil {
		t.Fatal(err)
	}
	inner, err := New(sets, Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(r *http.Request) (auth.Principal, error) {
		if r.Header.Get("Test-Other") == "true" {
			return other, nil
		}
		return p, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	h := Transport(nil, inner)
	invoke := func(path string, second bool) *httptest.ResponseRecorder {
		t.Helper()
		r := request(http.MethodGet, path, "", p)
		r.RemoteAddr = "192.0.2.21:443"
		if second {
			r.Header.Set("Test-Other", "true")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for i := 0; i < d.BrowserRate.Burst; i++ {
		w := invoke(paths[i%len(paths)], false)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d of burst %d was 429: %s", i+1, d.BrowserRate.Burst, w.Body.String())
		}
	}
	w := invoke(paths[0], false)
	retry, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil || w.Code != http.StatusTooManyRequests || retry < 1 || !strings.Contains(w.Body.String(), `"code":"RATE_LIMITED"`) || !strings.Contains(w.Body.String(), "Request rate exceeded.") {
		t.Fatalf("status=%d retry=%q err=%v body=%s", w.Code, w.Header().Get("Retry-After"), err, w.Body.String())
	}
	if retry != 1 {
		t.Fatalf("Retry-After=%d, want 1 (one token at 10/s is 0.1s, rounded up)", retry)
	}
	if second := invoke(paths[1], true); second.Code == http.StatusTooManyRequests {
		t.Fatalf("second session shared the exhausted bucket: %s", second.Body.String())
	}
}
