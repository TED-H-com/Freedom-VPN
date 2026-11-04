//go:build e2e

package updates_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	at "github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates"
)

func TestUPD02_NodeWriteRetiredPredicate(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "exact retired",
			body: `{"lifecycle_state":"retired"}`,
			want: true,
		},
		{
			name: "retired with full node detail fields",
			body: `{"id":"0192f000-0000-7000-8000-000000000001","lifecycle_state":"retired","resource_version":"9","name":"TEST ONLY installer handoff"}`,
			want: true,
		},
		{
			name: "registered rejected",
			body: `{"lifecycle_state":"registered","resource_version":"9"}`,
			want: false,
		},
		{
			name: "revoked rejected",
			body: `{"lifecycle_state":"revoked","resource_version":"9"}`,
			want: false,
		},
		{
			name: "enrolling rejected",
			body: `{"lifecycle_state":"enrolling","resource_version":"1"}`,
			want: false,
		},
		{
			name: "missing lifecycle_state rejected",
			body: `{"id":"0192f000-0000-7000-8000-000000000001","resource_version":"9"}`,
			want: false,
		},
		{
			name: "null lifecycle_state rejected",
			body: `{"lifecycle_state":null}`,
			want: false,
		},
		{
			name: "non-string lifecycle_state rejected",
			body: `{"lifecycle_state":1}`,
			want: false,
		},
		{
			name: "malformed json rejected",
			body: `{"lifecycle_state":`,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retiredNode([]byte(tc.body)); got != tc.want {
				t.Fatalf("retiredNode(%s) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestUPD02_InstallerRetireNodeWriteAndReceipts(t *testing.T) {
	const (
		csrfToken = "stub-csrf"
		nodeID    = "0192f000-0000-7000-8000-000000000024"
		nodePath  = "/api/v1/admin/nodes/" + nodeID
	)

	cases := []struct {
		name         string
		mode         string
		wantFatal    bool
		wantDeletes  int
		wantGets     int
		wantHTTPNote string
		wantOutcome  string
		wantStatus   at.Status
		wantAttempts int
		wantRereads  int
	}{
		{
			name:         "immediate 202",
			mode:         "immediate",
			wantFatal:    false,
			wantDeletes:  1,
			wantGets:     2,
			wantHTTPNote: "202",
			wantOutcome:  "accepted",
			wantStatus:   at.StatusPass,
			wantAttempts: 1,
			wantRereads:  0,
		},
		{
			name:         "conflict then retry 202",
			mode:         "retry",
			wantFatal:    false,
			wantDeletes:  2,
			wantGets:     3,
			wantHTTPNote: "202",
			wantOutcome:  "accepted",
			wantStatus:   at.StatusPass,
			wantAttempts: 2,
			wantRereads:  1,
		},
		{
			name:         "conflict already intended",
			mode:         "intended",
			wantFatal:    false,
			wantDeletes:  1,
			wantGets:     3,
			wantHTTPNote: "412 RESOURCE_VERSION_CONFLICT",
			wantOutcome:  "already_intended",
			wantStatus:   at.StatusPass,
			wantAttempts: 1,
			wantRereads:  1,
		},
		{
			name:         "other 412 not retried",
			mode:         "other412",
			wantFatal:    true,
			wantDeletes:  1,
			wantGets:     2,
			wantHTTPNote: "412 STATE_CONFLICT",
			wantOutcome:  "non_conflict_response",
			wantStatus:   at.StatusBlocked,
			wantAttempts: 1,
			wantRereads:  0,
		},
		{
			name:         "exhausted conflicts",
			mode:         "exhausted",
			wantFatal:    true,
			wantDeletes:  3,
			wantGets:     5,
			wantHTTPNote: "412 RESOURCE_VERSION_CONFLICT",
			wantOutcome:  "attempt_limit",
			wantStatus:   at.StatusBlocked,
			wantAttempts: 3,
			wantRereads:  3,
		},
		{
			name:         "accepted 202 but observed state still registered",
			mode:         "unapplied",
			wantFatal:    true,
			wantDeletes:  1,
			wantGets:     2,
			wantHTTPNote: "202",
			wantOutcome:  "accepted",
			wantStatus:   at.StatusPass,
			wantAttempts: 1,
			wantRereads:  0,
		},
		{
			name:         "missing initial ETag rejected with 428 not retried",
			mode:         "initial_get_500",
			wantFatal:    true,
			wantDeletes:  1,
			wantGets:     2,
			wantHTTPNote: "428 PRECONDITION_REQUIRED",
			wantOutcome:  "non_conflict_response",
			wantStatus:   at.StatusBlocked,
			wantAttempts: 1,
			wantRereads:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var deletes, gets int
			var seenKey string
			retiredState := false

			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c, err := r.Cookie("fvpn_session"); err != nil || c.Value != "stub-admin" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Header.Get("X-CSRF-Token") != csrfToken || r.Header.Get("Origin") != "https://portal.example.invalid:8443" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if r.URL.Path != nodePath {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")

				if r.Method == http.MethodGet {
					gets++
					if tc.mode == "initial_get_500" && gets == 1 {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR"}}`))
						return
					}
					version := fmt.Sprintf("%d", gets)
					w.Header().Set("ETag", `"`+version+`"`)
					state := "registered"
					if retiredState || (tc.mode == "intended" && gets >= 2) {
						state = "retired"
					}
					if tc.mode == "unapplied" {
						state = "registered"
					}
					_, _ = fmt.Fprintf(w, `{"id":%q,"lifecycle_state":%q,"resource_version":%q}`, nodeID, state, version)
					return
				}

				if r.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				deletes++
				key := r.Header.Get("Idempotency-Key")
				if key == "" {
					t.Error("missing Idempotency-Key on DELETE")
				}
				if seenKey == "" {
					seenKey = key
				} else if key != seenKey {
					t.Errorf("Idempotency-Key changed across retry: %q -> %q", seenKey, key)
				}
				if r.Header.Get("If-Match") == "" {
					w.WriteHeader(http.StatusPreconditionRequired)
					_, _ = w.Write([]byte(`{"error":{"code":"PRECONDITION_REQUIRED"}}`))
					return
				}

				switch {
				case tc.mode == "immediate" || tc.mode == "unapplied" || (tc.mode == "retry" && deletes == 2):
					retiredState = tc.mode != "unapplied"
					w.Header().Set("ETag", `"9"`)
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`{"operation_id":"0192f000-0000-7000-8000-000000000099","status":"accepted"}`))
				case tc.mode == "other412":
					w.WriteHeader(http.StatusPreconditionFailed)
					_, _ = w.Write([]byte(`{"error":{"code":"STATE_CONFLICT","details":{"current_resource_version":"2"}}}`))
				default:
					v := fmt.Sprintf("%d", deletes+1)
					w.WriteHeader(http.StatusPreconditionFailed)
					_, _ = fmt.Fprintf(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT","details":{"current_resource_version":%q}}}`, v)
				}
			}))
			defer srv.Close()

			stack, client := openStubUpdatesStack(t, srv)
			defer stack.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var fatalMsg string
			h := &handoffRun{
				s: &updatesFollowup{
					ctx:   ctx,
					admin: &session{http: client, csrf: csrfToken, role: "admin"},
					r:     &runner{t: t, lab: &lab{public: stack.PublicURL()}},
				},
				stack:  stack,
				checks: map[string]bool{},
				etags:  map[string]string{},
				nodeID: nodeID,
				onFatal: func(msg string) {
					fatalMsg = msg
				},
			}

			h.retire(nodeID)

			if (fatalMsg != "") != tc.wantFatal {
				t.Fatalf("fatalMsg = %q, wantFatal = %v", fatalMsg, tc.wantFatal)
			}
			if tc.wantFatal && fatalMsg != "installer fixture retirement not confirmed; refusing polluted fleet snapshots" {
				t.Fatalf("unexpected fatal message: %q", fatalMsg)
			}
			if deletes != tc.wantDeletes || gets != tc.wantGets {
				t.Fatalf("deletes=%d gets=%d, want deletes=%d gets=%d", deletes, gets, tc.wantDeletes, tc.wantGets)
			}

			httpIdx, receiptIdx, afterIdx := -1, -1, -1
			var gotHTTP string
			var gotReceipt e2e.NodeWriteReceipt
			for i, ob := range h.rec.Observations {
				switch ob.Name {
				case "fixture_retirement_http":
					httpIdx = i
					if err := json.Unmarshal([]byte(ob.Detail), &gotHTTP); err != nil {
						t.Fatalf("unmarshal fixture_retirement_http: %v", err)
					}
				case "fixture_retirement_node_write":
					receiptIdx = i
					if err := json.Unmarshal([]byte(ob.Detail), &gotReceipt); err != nil {
						t.Fatalf("unmarshal fixture_retirement_node_write: %v", err)
					}
				case "fixture_retired_http":
					afterIdx = i
				}
			}
			if httpIdx < 0 || receiptIdx != httpIdx+1 || afterIdx <= receiptIdx {
				t.Fatalf("unexpected observation order: httpIdx=%d receiptIdx=%d afterIdx=%d (observations=%+v)", httpIdx, receiptIdx, afterIdx, h.rec.Observations)
			}
			if gotHTTP != tc.wantHTTPNote {
				t.Fatalf("fixture_retirement_http = %q, want %q", gotHTTP, tc.wantHTTPNote)
			}
			if gotReceipt.Outcome != tc.wantOutcome || string(gotReceipt.Status) != string(tc.wantStatus) {
				t.Fatalf("receipt outcome=%q status=%q, want outcome=%q status=%q", gotReceipt.Outcome, gotReceipt.Status, tc.wantOutcome, tc.wantStatus)
			}
			if gotReceipt.MaxAttempts != 3 || gotReceipt.LimitSeconds != 10 {
				t.Fatalf("receipt bounds = %+v", gotReceipt)
			}
			if gotReceipt.StartedAt.IsZero() || gotReceipt.FinishedAt.Before(gotReceipt.StartedAt) {
				t.Fatalf("invalid receipt timestamps: %+v", gotReceipt)
			}
			if len(gotReceipt.Attempts) != tc.wantAttempts {
				t.Fatalf("len(receipt.Attempts) = %d, want %d", len(gotReceipt.Attempts), tc.wantAttempts)
			}
			rereads := 0
			for i, a := range gotReceipt.Attempts {
				if a.Method != http.MethodDelete || a.Path != nodePath || a.IdempotencyKey == "" || (tc.mode != "initial_get_500" && a.IfMatch == "") || len(a.BodySHA256) != 64 {
					t.Fatalf("attempt[%d] incomplete: %+v", i, a)
				}
				if i > 0 && a.IdempotencyKey != gotReceipt.Attempts[0].IdempotencyKey {
					t.Fatalf("attempt[%d] IdempotencyKey changed: %q != %q", i, a.IdempotencyKey, gotReceipt.Attempts[0].IdempotencyKey)
				}
				if a.Reread != nil {
					rereads++
					if a.Reread.Method != http.MethodGet || a.Reread.Path != nodePath || a.Reread.ETag == "" {
						t.Fatalf("attempt[%d] reread incomplete: %+v", i, a.Reread)
					}
				}
			}
			if rereads != tc.wantRereads {
				t.Fatalf("rereads = %d, want %d", rereads, tc.wantRereads)
			}
		})
	}
}

func openStubUpdatesStack(t *testing.T, srv *httptest.Server) (*e2e.Stack, *http.Client) {
	t.Helper()
	state := t.TempDir()
	cfg, err := json.Marshal(map[string]any{
		"prefix":     "fvpn-w4-updates",
		"host_ports": map[string]int{"control": 8443},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "stack.json"), cfg, 0o600); err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(filepath.Join(state, "ca.pem"), certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FVPN_E2E_STATE", state)
	stack, err := e2e.OpenStack()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(stack.PublicURL())
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "fvpn_session", Value: "stub-admin", Path: "/"}})
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	addr := srv.Listener.Addr().String()
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: "example.com"},
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
	return stack, client
}
