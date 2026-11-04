//go:build e2e

package enrollment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// Only metadata is opened. All HTTP uses a local TLS stub, including the real
// cookie jar and CSRF header; no stack, Docker or external service is started.
func nodeWriteProbe(t *testing.T, handler http.HandlerFunc) *probe {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	state := t.TempDir()
	t.Setenv("FVPN_E2E_STATE", state)
	for name, body := range map[string][]byte{
		"stack.json": []byte(`{"prefix":"fvpn-test-only","host_ports":{"control":1}}`),
		"ca.pem":     pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}),
	} {
		if err := os.WriteFile(filepath.Join(state, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	stack, err := labstack.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stack.Close)
	client := server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.TLSClientConfig = tr.TLSClientConfig.Clone()
	tr.TLSClientConfig.ServerName = "example.com"
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr
	t.Cleanup(tr.CloseIdleConnections)
	client.Jar, err = cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(stack.PublicURL())
	if err != nil {
		t.Fatal(err)
	}
	client.Jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "TEST_ONLY_cookie", Secure: true}})
	return &probe{ctx: t.Context(), stack: stack, admin: labstack.BrowserSession{Client: client, CSRF: "TEST ONLY csrf"}, resultPath: filepath.Join(state, "checks.json")}
}

func TestF10_NodeWriteSites(t *testing.T) {
	for _, site := range []string{"freeze_capacity", "freeze_publication", "installer_capacity", "installer_publication", "revoke"} {
		t.Run(site, func(t *testing.T) {
			group, node := uuid.New(), uuid.New()
			capacity := strings.HasSuffix(site, "capacity")
			publication := strings.HasSuffix(site, "publication")
			method, suffix := "POST", "/revoke"
			desired := map[string]any{"lifecycle_state": "revoked"}
			if capacity {
				method, suffix = "PATCH", ""
				desired = map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}
			} else if publication {
				method, suffix = "PUT", "/publication"
				desired = map[string]any{"publication_intent": "published", "group_ids": []uuid.UUID{group}}
			}
			type scenario struct {
				name     string
				readback any
			}
			cases := []scenario{{"accepted", nil}, {"retry", map[string]any{}}, {"already_intended", desired}, {"other_412", nil}, {"exhausted", nil}, {"reread_failed", nil}, {"transport", nil}}
			if capacity {
				for _, field := range []string{"down_bps", "up_bps", "member_down_bps_cap", "member_up_bps_cap"} {
					for _, missing := range []bool{false, true} {
						fields := map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}
						fields[field] = 1
						if missing {
							delete(fields, field)
						}
						cases = append(cases, scenario{fmt.Sprintf("mismatch_%s_missing_%t", field, missing), map[string]any{"capacity": fields}})
					}
				}
			} else if publication {
				for name, value := range map[string]any{"wrong_group": []uuid.UUID{uuid.New()}, "extra_group": []uuid.UUID{group, uuid.New()}, "duplicates": []uuid.UUID{group, group}, "null_groups": nil} {
					cases = append(cases, scenario{name, map[string]any{"publication_intent": "published", "group_ids": value}})
				}
				cases = append(cases, scenario{"disabled", map[string]any{"publication_intent": "disabled", "group_ids": []uuid.UUID{group}}}, scenario{"missing_groups", map[string]any{"publication_intent": "published"}})
			} else {
				for _, state := range []string{"active", "retired", ""} {
					cases = append(cases, scenario{"wrong_lifecycle_" + state, map[string]any{"lifecycle_state": state}})
				}
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					writes, rereads, reads, readiness := 0, 0, 0, 0
					pending := false
					keys, bodies, etags := []string{}, []string{}, []string{}
					path := "/api/v1/admin/nodes/" + node.String()
					p := nodeWriteProbe(t, func(w http.ResponseWriter, r *http.Request) {
						cookie, err := r.Cookie("session")
						if err != nil || cookie.Value != "TEST_ONLY_cookie" {
							t.Error("caller cookie lost")
						}
						if r.Method == "GET" {
							reads++
							if r.URL.Path != path {
								t.Error("wrong read path", r.URL.Path)
							}
							w.Header().Set("ETag", fmt.Sprintf(`"%d"`, reads))
							if pending {
								pending = false
								rereads++
								if tc.name == "reread_failed" {
									w.WriteHeader(503)
									return
								}
								_ = json.NewEncoder(w).Encode(tc.readback)
								return
							}
							readiness++
							_, _ = io.WriteString(w, `{"provisioning_state":"ready","safety_status":"passed"}`)
							return
						}
						if r.Header.Get("X-CSRF-Token") != "TEST ONLY csrf" {
							t.Error("caller CSRF lost")
						}
						if r.Method != method {
							w.WriteHeader(202) // The other setup command.
							return
						}
						if r.URL.Path != path+suffix {
							t.Error("wrong write path", r.URL.Path)
						}
						writes++
						body, _ := io.ReadAll(r.Body)
						bodies = append(bodies, string(body))
						keys = append(keys, r.Header.Get("Idempotency-Key"))
						etags = append(etags, r.Header.Get("If-Match"))
						if tc.name == "transport" {
							conn, _, e := w.(http.Hijacker).Hijack()
							if e != nil {
								t.Error(e)
								return
							}
							_ = conn.Close()
							return
						}
						if tc.name != "accepted" && (writes == 1 || tc.name == "exhausted") {
							code := "RESOURCE_VERSION_CONFLICT"
							pending = true
							if tc.name == "other_412" {
								code, pending = "STATE_CONFLICT", false
							}
							w.WriteHeader(412)
							_, _ = fmt.Fprintf(w, `{"error":{"code":%q}}`, code)
							return
						}
						w.WriteHeader(202)
						_, _ = io.WriteString(w, `{"operation_id":"00000000-0000-4000-8000-000000000001"}`)
					})
					p.servingGroup = group
					detail := map[string]any{}
					var ok bool
					if strings.HasPrefix(site, "freeze") {
						ok = p.prepareFreezeNode(node, group, detail)
					} else if strings.HasPrefix(site, "installer") {
						ok = p.installerPublish(node, detail)
					} else {
						ok = p.revokeNode(node, detail).Accepted(202)
					}
					wantOK := tc.name != "other_412" && tc.name != "exhausted" && tc.name != "reread_failed" && tc.name != "transport"
					if ok != wantOK {
						t.Fatalf("accepted=%t, want %t: %+v", ok, wantOK, detail)
					}
					p.expect("AT07", site, ok, detail)
					raw, err := os.ReadFile(p.resultPath)
					if err != nil {
						t.Fatal(err)
					}
					var saved []struct {
						Detail map[string]json.RawMessage `json:"detail"`
					}
					if json.Unmarshal(raw, &saved) != nil || len(saved) != 1 {
						t.Fatal("check not saved")
					}
					rows := saved[0].Detail
					if strings.HasPrefix(site, "installer") {
						var publications []map[string]json.RawMessage
						if json.Unmarshal(rows["installer_publications"], &publications) != nil || len(publications) != 1 {
							t.Fatal("publication receipts lost")
						}
						rows = publications[0]
					}
					var record struct {
						Receipt  labstack.NodeWriteReceipt `json:"node_write"`
						Status   int                       `json:"http_status"`
						Intended bool                      `json:"already_intended"`
					}
					if json.Unmarshal(rows[method+suffix], &record) != nil {
						t.Fatal("missing receipt")
					}
					wantWrites, wantReads, outcome := 2, 1, "accepted"
					switch tc.name {
					case "accepted":
						wantWrites, wantReads = 1, 0
					case "already_intended":
						wantWrites, outcome = 1, "already_intended"
					case "other_412":
						wantWrites, wantReads, outcome = 1, 0, "non_conflict_response"
					case "exhausted":
						wantWrites, wantReads, outcome = 3, 3, "attempt_limit"
					case "reread_failed":
						wantWrites, outcome = 1, "reread_failed"
					case "transport":
						wantWrites, wantReads, outcome = 1, 0, "transport_error"
					}
					trace := record.Receipt
					if writes != wantWrites || rereads != wantReads || len(trace.Attempts) != writes || trace.Outcome != outcome || trace.MaxAttempts != 3 || trace.LimitSeconds != 10 || trace.StartedAt.IsZero() || trace.FinishedAt.Before(trace.StartedAt) {
						t.Fatalf("wrong receipt: %+v; writes=%d rereads=%d", trace, writes, rereads)
					}
					if record.Intended != (tc.name == "already_intended") || (record.Intended && record.Status != 412) {
						t.Fatal("invented HTTP success")
					}
					for i, attempt := range trace.Attempts {
						if _, err := uuid.Parse(keys[i]); err != nil {
							t.Fatal("invalid key", err)
						}
						if keys[i] != keys[0] || bodies[i] != bodies[0] || attempt.BodySHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(bodies[i]))) || attempt.IfMatch != etags[i] || attempt.IdempotencyKey != keys[i] || attempt.Path != path+suffix || attempt.Method != method {
							t.Fatal("attempt changed or receipt lost", attempt)
						}
						if i > 0 && etags[i] == etags[i-1] {
							t.Fatal("stale ETag reused")
						}
						if i < wantReads && (attempt.Reread == nil || attempt.Reread.Path != path || attempt.Reread.Intended != record.Intended) {
							t.Fatal("reread missing/inexact", attempt)
						}
					}
					var sent map[string]any
					if json.Unmarshal([]byte(bodies[0]), &sent) != nil {
						t.Fatal("invalid body")
					}
					if capacity || publication {
						delete(sent, "reason")
						want, _ := json.Marshal(desired)
						var normalized map[string]any
						_ = json.Unmarshal(want, &normalized)
						if !reflect.DeepEqual(sent, normalized) {
							t.Fatalf("wrong command body: %s", bodies[0])
						}
					} else if sent["reason"] != "TEST ONLY AT06 reinstall" {
						t.Fatal("wrong revoke reason")
					}
					if ok && site != "revoke" && readiness < 3 {
						t.Fatal("readiness check removed")
					}
					if strings.Contains(string(raw), "TEST_ONLY_cookie") || strings.Contains(string(raw), "TEST ONLY csrf") {
						t.Fatal("credentials leaked to receipt")
					}
					if ok && strings.HasPrefix(site, "installer") {
						first, _ := json.Marshal(detail["installer_publications"])
						if !p.installerPublish(node, detail) {
							t.Fatal("second publication failed")
						}
						all := detail["installer_publications"].([]map[string]any)
						preserved, _ := json.Marshal(all[:1])
						if len(all) != 2 || string(preserved) != string(first) {
							t.Fatal("initial receipts overwritten by reinstall")
						}
					}
				})
			}
		})
	}
}

func TestF10_RevokeOperationReadback(t *testing.T) {
	for _, mode := range []string{"found", "missing", "ambiguous", "old", "wrong_node", "wrong_action", "failed", "paginated", "http_error"} {
		t.Run(mode, func(t *testing.T) {
			node, operation := uuid.New(), uuid.New()
			retired := time.Now().UTC().Truncate(time.Microsecond)
			p := nodeWriteProbe(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api/v1/admin/audit-events" || r.URL.Query().Get("resource_id") != node.String() || r.URL.Query().Get("occurred_from") != retired.Format(time.RFC3339Nano) {
					t.Error("incorrect operation lookup", r.URL)
				}
				if mode == "http_error" {
					w.WriteHeader(503)
					return
				}
				event := map[string]any{"action": "node.revoke", "result": "succeeded", "resource": map[string]any{"type": "node", "id": node}, "operation_id": operation, "occurred_at": retired}
				switch mode {
				case "old":
					event["occurred_at"] = retired.Add(-time.Second)
				case "wrong_node":
					event["resource"] = map[string]any{"type": "node", "id": uuid.New()}
				case "wrong_action":
					event["action"] = "node.publish"
				case "failed":
					event["result"] = "failed"
				}
				items := []any{event}
				if mode == "missing" {
					items = nil
				}
				if mode == "ambiguous" {
					items = append(items, event)
				}
				var cursor any
				if mode == "paginated" {
					cursor = "TEST ONLY cursor"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "next_cursor": cursor})
			})
			result := labstack.NodeWriteResult{AlreadyIntended: true, Reply: labstack.Response{Status: 412}}
			got, ok := p.revokeOperation(node, result, retired, map[string]any{})
			if ok != (mode == "found") || (ok && got != operation) {
				t.Fatalf("lookup=%s/%t", got, ok)
			}
		})
	}
}
