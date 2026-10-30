//go:build e2e

package secops

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func boundary(t *testing.T, p auth.Principal, agent bool, log *slog.Logger, sets ...any) http.Handler {
	t.Helper()
	c, err := httpapi.Compose(sets...)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.PublicOrigin = "https://portal.example.invalid"
	h, err := httpapi.New(c, httpapi.Options{Config: cfg, AgentListener: agent,
		Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) {
			if p.Role == "" {
				return p, apierr.New(apierr.Unauthenticated, "Sign in required.")
			}
			return p, nil
		}), Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) { return p, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	return httpapi.Transport(log, h)
}

func refusal(t *testing.T, h http.Handler, r *http.Request, status int, code string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Keep the literal contract field names in refusal evidence.
	var wire struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wire); err != nil {
		t.Fatalf("non-JSON refusal: status=%d", w.Code)
	}
	if w.Code != status || wire.Error.Code != code {
		t.Fatalf("refusal status=%d code=%s; want %d %s", w.Code, wire.Error.Code, status, code)
	}
	if _, err := uuid.Parse(wire.Error.RequestID); err != nil || wire.Error.RequestID != w.Header().Get("X-Request-ID") {
		t.Fatal("refusal missing correlated request ID")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("refusal can be cached")
	}
}

// Every protected operation is enumerated from the generated HTTP authority.
// This proves the outer role/listener boundary, not object-level ownership;
// member/device/operation ownership is exercised by the domain integration suites.
func TestAT28_AllContractRoleBoundaries(t *testing.T) {
	doc, err := gen.GetSpec()
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, doc.Paths.Len())
	for path := range doc.Paths.Map() {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	params := regexp.MustCompile(`\{[^}]+\}`)
	operations, protected := 0, 0
	for _, path := range paths {
		methods := doc.Paths.Value(path).Operations()
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			op := methods[method]
			if op == nil {
				continue
			}
			operations++
			if len(op.Tags) != 1 {
				t.Fatalf("invalid security tag for %s", op.OperationID)
			}
			tag := op.Tags[0]
			if tag != "admin" && tag != "member" && (tag != "agent" || strings.EqualFold(op.OperationID, "enrollNode")) {
				continue
			}
			protected++
			t.Run(op.OperationID, func(t *testing.T) {
				url := "/api/v1" + params.ReplaceAllString(path, uuid.NewString())
				for _, role := range []auth.Role{"", auth.Member, auth.Admin, auth.Agent} {
					if string(role) == tag {
						continue
					}
					t.Run("wrong_role_"+string(role), func(t *testing.T) {
						p := auth.Principal{Role: role, ID: uuid.New(), MFAVerified: true, AuthenticatedAt: time.Now()}
						want, code := 403, "FORBIDDEN"
						if role == "" || tag == "agent" {
							want, code = 401, "UNAUTHENTICATED"
						}
						r := httptest.NewRequest(method, url, nil)
						r.Header.Set("X-Node-ID", uuid.NewString())
						r.Header.Set("X-Client-Cert", "TEST ONLY forged identity")
						refusal(t, boundary(t, p, tag == "agent", nil), r, want, code)
					})
				}
				t.Run("wrong_listener", func(t *testing.T) {
					p := auth.Principal{Role: auth.Role(tag), ID: uuid.New(), MFAVerified: true}
					refusal(t, boundary(t, p, tag != "agent", nil), httptest.NewRequest(method, url, nil), 401, "UNAUTHENTICATED")
				})
				if tag == "admin" {
					t.Run("missing_mfa", func(t *testing.T) {
						p := auth.Principal{Role: auth.Admin, ID: uuid.New()}
						refusal(t, boundary(t, p, false, nil), httptest.NewRequest(method, url, nil), 403, "FORBIDDEN")
					})
				}
			})
		}
	}
	// D-065 / contracts revision 9 and revision 10 (#191) bring the contract to 47 operations (45 protected).
	if operations != 47 || protected != 45 {
		t.Fatalf("contract coverage changed: operations=%d protected=%d", operations, protected)
	}
	t.Logf("contract operations=%d protected=%d; enrollment and provider webhook use distinct credential checks", operations, protected)
}

func TestAT28_TransportSecretCanaries(t *testing.T) {
	var logs bytes.Buffer
	h := boundary(t, auth.Principal{}, false, slog.New(slog.NewJSONHandler(&logs, nil)))
	canaries := make([]string, 0, 6)
	r := httptest.NewRequest("GET", "/api/v1/admin/nodes", nil)
	for _, key := range []string{"Authorization", "Cookie", "X-Client-Cert", "X-Setup-Key", "X-Webhook-Signature", "X-Request-ID"} {
		v := "TEST-ONLY-" + uuid.NewString()
		canaries = append(canaries, v)
		r.Header.Set(key, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 || logs.Len() == 0 {
		t.Fatal("missing refusal or log stream")
	}
	for _, stream := range []string{logs.String(), w.Body.String(), w.Header().Get("X-Request-ID")} {
		for _, canary := range canaries {
			if strings.Contains(stream, canary) {
				t.Fatal("secret canary leaked; value withheld")
			}
		}
	}
	t.Log("scanned transport log, refusal body and correlation header: 6 runtime canaries, 0 findings; database and stack streams require separate coverage")
}
