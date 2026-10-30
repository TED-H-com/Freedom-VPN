package netbird

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestID01_D052ListUsersRequiresExplicitIdentityFlags(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"native", `[{"id":"TEST-public-subject","is_service_user":false,"is_blocked":false,"email":"ignored@example.invalid"}]`, true},
		{"service", `[{"id":"TEST-service","is_service_user":true,"is_blocked":false}]`, true},
		{"blocked", `[{"id":"TEST-blocked","is_service_user":false,"is_blocked":true}]`, true},
		{"empty", `[]`, true},
		{"missing flag", `[{"id":"TEST-subject","is_blocked":false}]`, false},
		{"null flag", `[{"id":"TEST-subject","is_service_user":false,"is_blocked":null}]`, false},
		{"blank id", `[{"id":" ","is_service_user":false,"is_blocked":false}]`, false},
		{"duplicate", `[{"id":"TEST-subject","is_service_user":false,"is_blocked":false},{"id":"TEST-subject","is_service_user":true,"is_blocked":true}]`, false},
		{"null response", `null`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/users" || r.Header.Get("Authorization") != "Token TEST ONLY synthetic token" {
					t.Error("user list bypassed the authenticated client")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c, err := New(Config{BaseURL: srv.URL + "/api", DeploymentID: "TEST ONLY", Token: testToken("TEST ONLY synthetic token"), Bindings: NewMemoryBindingStore(), AllowInsecureHTTP: true, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			users, err := c.ListUsers(t.Context())
			if tc.valid {
				if err != nil {
					t.Fatal(err)
				}
				if tc.name == "empty" && len(users) != 0 {
					t.Fatal(users)
				}
				if tc.name != "empty" && (len(users) != 1 || users[0].ServiceUser != (tc.name == "service") || users[0].Blocked != (tc.name == "blocked")) {
					t.Fatal(users)
				}
			} else if !IsKind(err, Invalid) || users != nil {
				t.Fatal("malformed user list accepted", users, err)
			}
		})
	}
}
