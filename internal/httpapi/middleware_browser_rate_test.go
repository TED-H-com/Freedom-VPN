package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// TestFollowup21_BrowserClassCalibration pins the operator subject budgets.
// Admin list reads use class read at 30/s burst 60. Node and member detail
// stay on class browser at 10/s burst 30. The 31st detail request on one
// session is 429 RATE_LIMITED with Retry-After. A second session on the same
// address still has its own browser burst.
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
	if d.ReadRate != (config.Bucket{PerSecond: 30, Burst: 60}) {
		t.Fatalf("read_rate = %+v, want 30/s burst 60", d.ReadRate)
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
	wantClasses := []string{"browser", "read", "agent", "heartbeat", "lease", "usage", "anonymous"}
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
	opID := uuid.NewString()
	for _, path := range []string{
		"/api/v1/admin/audit-events",
		"/api/v1/admin/nodes",
		"/api/v1/admin/nodes/" + nodeID + "/incarnations",
		"/api/v1/admin/members",
		"/api/v1/admin/devices",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		route, _, err := findRoute(router, req)
		if err != nil {
			t.Fatal(path, err)
		}
		class, bucket, _ := classify(route.Operation, d)
		if class != "read" || bucket != d.ReadRate {
			t.Fatalf("%s OperationID=%s class=%s bucket=%+v", path, route.Operation.OperationID, class, bucket)
		}
	}
	for _, path := range []string{
		"/api/v1/admin/nodes/" + nodeID,
		"/api/v1/admin/members/" + memberID,
		"/api/v1/admin/operations/" + opID,
	} {
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
	paths := []string{
		"/api/v1/admin/nodes/" + nodeID,
		"/api/v1/admin/members/" + memberID,
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
	p := principal(auth.Admin)
	p.SessionID = uuid.New()
	other := p
	other.SessionID = uuid.New()
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

// TestFollowup21_AdminListReadsPageWithout429 exercises generated parameter
// binding and successful fixture handlers through the real middleware. The
// limiter clock models 15 pages in 1.5 s without wall-clock timing assertions.
func TestFollowup21_AdminListReadsPageWithout429(t *testing.T) {
	for _, scenario := range []string{"three_lists_in_two_seconds", "session_burst_and_refill"} {
		t.Run(scenario, func(t *testing.T) {
			c := config.Defaults()
			c.PublicOrigin = "https://portal.example.invalid"
			now := time.Date(2026, 10, 30, 22, 26, 0, 0, time.UTC)
			setLimiterClock(t, func() time.Time { return now })
			p := principal(auth.Admin)
			p.SessionID = uuid.New()
			other := p
			other.SessionID = uuid.New()
			pages := &calibrationPages{t: t}
			sets, err := Compose(pages)
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
			nodeID := uuid.NewString()
			lists := []string{
				"/api/v1/admin/audit-events?limit=100",
				"/api/v1/admin/nodes?limit=100",
				"/api/v1/admin/nodes/" + nodeID + "/incarnations?limit=100",
			}
			invoke := func(path, address string, second bool) *httptest.ResponseRecorder {
				t.Helper()
				r := request(http.MethodGet, path, "", p)
				r.RemoteAddr = address + ":443"
				if second {
					r.Header.Set("Test-Other", "true")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				return w
			}
			assertPage := func(w *httptest.ResponseRecorder) string {
				t.Helper()
				if w.Code != http.StatusOK {
					t.Fatalf("page: status=%d body=%s", w.Code, w.Body.String())
				}
				var page struct {
					NextCursor string `json:"next_cursor"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || page.NextCursor == "" {
					t.Fatalf("page cursor: %s (%v)", w.Body.String(), err)
				}
				return page.NextCursor
			}
			assertLimited := func(w *httptest.ResponseRecorder) {
				t.Helper()
				// Distinguish the subject rejection from the address limiter,
				// which uses "Authentication request rate exceeded.".
				if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "1" ||
					!strings.Contains(w.Body.String(), `"code":"RATE_LIMITED"`) ||
					!strings.Contains(w.Body.String(), `"message":"Request rate exceeded."`) {
					t.Fatalf("read subject limit: status=%d retry=%q body=%s", w.Code, w.Header().Get("Retry-After"), w.Body.String())
				}
			}
			if scenario == "three_lists_in_two_seconds" {
				start := now
				cursors := make([]string, len(lists))
				for page := 0; page < 5; page++ {
					for i, path := range lists {
						if cursors[i] != "" {
							path += "&cursor=" + url.QueryEscape(cursors[i])
						}
						cursors[i] = assertPage(invoke(path, "192.0.2.41", false))
						now = now.Add(100 * time.Millisecond)
					}
				}
				if pages.calls != 15 || pages.cursorCalls != 12 || now.Sub(start) > 2*time.Second {
					t.Fatalf("paging: calls=%d cursor calls=%d duration=%s", pages.calls, pages.cursorCalls, now.Sub(start))
				}
				return
			}
			// Two addresses leave ingress capacity available, so the 61st
			// request proves the per-session read burst with default config.
			for i := 0; i < c.ReadRate.Burst; i++ {
				address := fmt.Sprintf("192.0.2.%d", 41+i%2)
				assertPage(invoke(lists[i%len(lists)], address, false))
			}
			assertLimited(invoke(lists[0], "192.0.2.41", false))
			assertPage(invoke(lists[0], "192.0.2.41", true))
			if w := invoke("/api/v1/admin/nodes/"+nodeID, "192.0.2.41", false); w.Code != http.StatusNotImplemented {
				t.Fatalf("detail must reach the unregistered handler on its browser bucket: %d %s", w.Code, w.Body.String())
			}
			// Honour Retry-After and prove exactly 30 read tokens replenish.
			now = now.Add(time.Second)
			for i := 0; i < 30; i++ {
				assertPage(invoke(lists[0], "192.0.2.43", false))
			}
			assertLimited(invoke(lists[0], "192.0.2.43", false))
		})
	}
}

// These fixtures exercise pagination transport, not domain query semantics.
type calibrationPages struct {
	t                  *testing.T
	calls, cursorCalls int
}

func (p *calibrationPages) next(limit *gen.Limit, cursor *gen.Cursor) string {
	p.t.Helper()
	if limit == nil || *limit != 100 {
		p.t.Fatalf("page limit = %v, want 100", limit)
	}
	p.calls++
	if cursor != nil {
		p.cursorCalls++
		if !strings.HasPrefix(*cursor, "test-page-") {
			p.t.Fatalf("cursor was not returned by fixture: %q", *cursor)
		}
	}
	return fmt.Sprintf("test-page-%d", p.calls)
}

func (p *calibrationPages) ListAuditEvents(_ context.Context, r gen.ListAuditEventsRequestObject) (gen.ListAuditEventsResponseObject, error) {
	body := gen.AuditEventList{Items: []gen.AuditEvent{}}
	body.NextCursor.Set(p.next(r.Params.Limit, r.Params.Cursor))
	return gen.ListAuditEvents200JSONResponse{Body: body}, nil
}

func (p *calibrationPages) ListNodes(_ context.Context, r gen.ListNodesRequestObject) (gen.ListNodesResponseObject, error) {
	body := gen.NodeList{Items: []gen.NodeSummary{}}
	body.NextCursor.Set(p.next(r.Params.Limit, r.Params.Cursor))
	return gen.ListNodes200JSONResponse{Body: body}, nil
}

func (p *calibrationPages) ListNodeIncarnations(_ context.Context, r gen.ListNodeIncarnationsRequestObject) (gen.ListNodeIncarnationsResponseObject, error) {
	body := gen.NodeIncarnationList{Items: []gen.NodeIncarnation{}}
	body.NextCursor.Set(p.next(r.Params.Limit, r.Params.Cursor))
	return gen.ListNodeIncarnations200JSONResponse{Body: body}, nil
}
