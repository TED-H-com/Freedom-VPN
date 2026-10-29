package auth

import (
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAT28_RoleCookiesCoexistAndLogoutIndependently(t *testing.T) {
	for _, first := range []Role{Admin, Member} {
		t.Run(string(first), func(t *testing.T) {
			store := &memoryStore{sessions: map[string]Principal{"admin-token": {ID: uuid.New(), Role: Admin, MFAVerified: true}, "member-token": {ID: uuid.New(), Role: Member}}}
			browser := &Browser{store: store, config: BrowserConfig{Idle: time.Hour, Origin: "https://portal.example.invalid"}}
			mux := http.NewServeMux()
			browser.Register(mux)
			req := func(method, path string) *http.Request {
				r := httptest.NewRequest(method, path, nil)
				r.AddCookie(&http.Cookie{Name: AdminSessionCookie, Value: "admin-token"})
				r.AddCookie(&http.Cookie{Name: MemberSessionCookie, Value: "member-token"})
				return r
			}
			for _, role := range []Role{Admin, Member} {
				p, e := browser.AuthenticateSession(req("GET", "/api/v1/"+string(role)+"/operations/"+uuid.NewString()+"?role=other"))
				if e != nil || p.Role != role {
					t.Fatal("route cookie selection", p, e)
				}
			}
			ambiguous := httptest.NewRecorder()
			mux.ServeHTTP(ambiguous, req("GET", "/auth/session"))
			if ambiguous.Code != 400 {
				t.Fatal("ambiguous role", ambiguous.Code)
			}
			r := req("GET", "/auth/session?role="+string(first))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"role":"`+string(first)+`"`) {
				t.Fatal(w.Code, w.Body.String())
			}
			logout := req("POST", "/auth/logout?role="+string(first))
			logout.Header.Set("Origin", browser.config.Origin)
			logout.Header.Set("X-CSRF-Token", w.Header().Get("X-CSRF-Token"))
			out := httptest.NewRecorder()
			mux.ServeHTTP(out, logout)
			if out.Code != 204 {
				t.Fatal(out.Code, out.Body.String())
			}
			cookies := out.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != cookieNameForRole(first) || cookies[0].MaxAge != -1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Path != "/" || cookies[0].SameSite != http.SameSiteLaxMode {
				t.Fatal(cookies)
			}
			other := Member
			if first == Member {
				other = Admin
			}
			out = httptest.NewRecorder()
			mux.ServeHTTP(out, req("GET", "/auth/session?role="+string(other)))
			if out.Code != 200 || !strings.Contains(out.Body.String(), `"authenticated":true`) {
				t.Fatal("other session lost", out.Body.String())
			}
			// Moving a member token into an administrator cookie cannot escalate roles.
			forged := httptest.NewRequest("GET", "/api/v1/admin/nodes", nil)
			forged.AddCookie(&http.Cookie{Name: AdminSessionCookie, Value: "member-token"})
			if _, err := browser.AuthenticateSession(forged); err == nil {
				t.Fatal("cross-role token accepted")
			}
		})
	}
}

func TestAT28_BrowserSelectorRequiresBothCookies(t *testing.T) {
	for _, tc := range []struct {
		name, selector string
		cookies        []Role
		status         int
		role           Role
	}{
		{"no cookie", "role=invalid", nil, 200, ""},
		{"sole member ignores admin selector", "role=admin", []Role{Member}, 200, Member},
		{"sole admin ignores member selector", "role=member", []Role{Admin}, 200, Admin},
		{"sole admin ignores repeated invalid selector", "role=invalid&role=other", []Role{Admin}, 200, Admin},
		{"both need selector", "", []Role{Admin, Member}, 400, ""},
		{"both reject invalid selector", "role=invalid", []Role{Admin, Member}, 400, ""},
		{"both reject repeated selector", "role=admin&role=member", []Role{Admin, Member}, 400, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &memoryStore{sessions: map[string]Principal{"admin-token": {ID: uuid.New(), Role: Admin, MFAVerified: true}, "member-token": {ID: uuid.New(), Role: Member}}}
			browser := &Browser{store: store, config: BrowserConfig{Idle: time.Hour}}
			mux := http.NewServeMux()
			browser.Register(mux)
			r := httptest.NewRequest("GET", "/auth/session?"+tc.selector, nil)
			for _, role := range tc.cookies {
				r.AddCookie(&http.Cookie{Name: cookieNameForRole(role), Value: string(role) + "-token"})
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.role != "" && !strings.Contains(w.Body.String(), `"role":"`+string(tc.role)+`"`) {
				t.Fatal("selector changed the sole session identity", w.Body.String())
			}
			if len(tc.cookies) == 0 && !strings.Contains(w.Body.String(), `"authenticated":false`) {
				t.Fatal("selector created an identity", w.Body.String())
			}
		})
	}
}
