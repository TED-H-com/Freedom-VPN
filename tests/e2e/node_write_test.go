package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAPI01_NodeWriteConflict(t *testing.T) {
	group := uuid.New()
	capacity := map[string]any{"capacity": map[string]int{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}
	for _, command := range []struct {
		method, suffix string
		body           any
		matches        func([]byte) bool
		intended       any
	}{
		{"PATCH", "", capacity, nodeFieldsMatch(capacity), capacity},
		{"PUT", "/publication", map[string]any{"publication_intent": "published", "group_ids": []uuid.UUID{group}, "reason": "TEST ONLY"}, publishedNode(group), map[string]any{"publication_intent": "published", "group_ids": []uuid.UUID{group}}},
		{"POST", "/revoke", map[string]string{"reason": "TEST ONLY"}, revokedNode, map[string]string{"lifecycle_state": "revoked"}},
	} {
		t.Run(command.method, func(t *testing.T) {
			for _, tc := range []struct {
				name, outcome string
				status        int
				code          string
				persistent    bool
				already       bool
				writes, reads int
				accepted      bool
			}{
				{"one_conflict", "accepted", 412, "RESOURCE_VERSION_CONFLICT", false, false, 2, 1, true},
				{"persistent", "attempt_limit", 412, "RESOURCE_VERSION_CONFLICT", true, false, nodeWriteAttempts, nodeWriteAttempts, false},
				{"already_intended", "already_intended", 412, "RESOURCE_VERSION_CONFLICT", true, true, 1, 1, true},
				{"wrong_412_code", "non_conflict_response", 412, "OTHER", false, false, 1, 0, false},
				{"unauthorized", "non_conflict_response", 401, "UNAUTHORIZED", false, false, 1, 0, false},
				{"forbidden", "non_conflict_response", 403, "FORBIDDEN", false, false, 1, 0, false},
				{"not_found", "non_conflict_response", 404, "NOT_FOUND", false, false, 1, 0, false},
				{"idempotency_conflict", "non_conflict_response", 409, "IDEMPOTENCY_CONFLICT", false, false, 1, 0, false},
				{"precondition_required", "non_conflict_response", 428, "PRECONDITION_REQUIRED", false, false, 1, 0, false},
				{"rate_limited", "non_conflict_response", 429, "RATE_LIMITED", false, false, 1, 0, false},
				{"server_error", "non_conflict_response", 500, "INTERNAL_ERROR", false, false, 1, 0, false},
				{"unavailable", "non_conflict_response", 503, "DEPENDENCY_UNAVAILABLE", false, false, 1, 0, false},
				{"success", "accepted", 202, "", false, false, 1, 0, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var mu sync.Mutex
					var methods, tags, keys, bodies []string
					writes, reads := 0, 0
					path := "/api/v1/admin/nodes/" + uuid.NewString()
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mu.Lock()
						defer mu.Unlock()
						methods = append(methods, r.Method)
						if r.Method == "GET" {
							reads++
							if r.URL.Path != path {
								t.Errorf("reread path: %s", r.URL.Path)
							}
							w.Header().Set("ETag", fmt.Sprintf(`"%d"`, reads+1))
							if tc.already {
								_ = json.NewEncoder(w).Encode(command.intended)
							} else {
								_, _ = io.WriteString(w, `{}`)
							}
							return
						}
						writes++
						if r.Method != command.method || r.URL.Path != path+command.suffix {
							t.Errorf("write: %s %s", r.Method, r.URL.Path)
						}
						raw, _ := io.ReadAll(r.Body)
						bodies = append(bodies, string(raw))
						tags = append(tags, r.Header.Get("If-Match"))
						keys = append(keys, r.Header.Get("Idempotency-Key"))
						if writes == 1 || tc.persistent {
							w.WriteHeader(tc.status)
							_, _ = fmt.Fprintf(w, `{"error":{"code":%q,"details":{"current_resource_version":%q}}}`, tc.code, strconv.Itoa(writes+1))
						} else {
							w.WriteHeader(202)
							_, _ = io.WriteString(w, `{"operation_id":"TEST ONLY"}`)
						}
					}))
					defer server.Close()
					client := &httpSession{http: server.Client()}
					var trace nodeWriteTrace
					notes := 0
					out := client.writeNode(t.Context(), command.method, server.URL+path+command.suffix, server.URL+path, command.body, `"1"`, command.matches, []int{202}, func(name, raw string) {
						notes++
						if name != "node_write" || json.Unmarshal([]byte(raw), &trace) != nil {
							t.Fatal("missing structured trace")
						}
					})
					mu.Lock()
					defer mu.Unlock()
					if notes != 1 || writes != tc.writes || reads != tc.reads || out.accepted(202) != tc.accepted || out.AlreadyIntended != tc.already || trace.Outcome != tc.outcome || len(trace.Attempts) != writes {
						t.Fatalf("writes=%d reads=%d result=%+v trace=%+v", writes, reads, out, trace)
					}
					wantStatus := StatusBlocked
					if tc.accepted {
						wantStatus = StatusPass
					}
					if trace.Status != wantStatus || trace.MaxAttempts != 3 || trace.LimitSeconds != 10 {
						t.Fatalf("bound/status: %+v", trace)
					}
					wantBody, _ := json.Marshal(command.body)
					previous := trace.StartedAt
					var expectedMethods []string
					for i, row := range trace.Attempts {
						expectedMethods = append(expectedMethods, command.method)
						if row.Method != command.method || row.Path != path+command.suffix || row.IfMatch != tags[i] || tags[i] != fmt.Sprintf(`"%d"`, i+1) || row.IdempotencyKey != keys[i] || keys[i] != keys[0] || bodies[i] != string(wantBody) {
							t.Fatalf("attempt %d: %+v", i, row)
						}
						if _, err := uuid.Parse(keys[i]); err != nil {
							t.Fatal(err)
						}
						if row.StartedAt.Before(previous) || row.FinishedAt.Before(row.StartedAt) || row.StartedAt.Location() != time.UTC || row.FinishedAt.Location() != time.UTC {
							t.Fatal("unordered/non-UTC attempt")
						}
						previous = row.FinishedAt
						if row.Reread != nil {
							expectedMethods = append(expectedMethods, "GET")
							read := row.Reread
							if row.Status != 412 || row.ErrorCode != "RESOURCE_VERSION_CONFLICT" || row.CurrentVersion != strconv.Itoa(i+2) || read.Method != "GET" || read.Path != path || read.Status != 200 || read.ETag != fmt.Sprintf(`"%d"`, i+2) || read.Intended != tc.already || read.StartedAt.Before(previous) || read.FinishedAt.Before(read.StartedAt) || read.StartedAt.Location() != time.UTC || read.FinishedAt.Location() != time.UTC {
								t.Fatalf("reread: %+v", row)
							}
							previous = read.FinishedAt
						}
						if i == writes-1 && !tc.already && tc.accepted && row.Status != 202 {
							t.Fatal("missing accepted response")
						}
					}
					if trace.FinishedAt.Before(previous) || !reflect.DeepEqual(methods, expectedMethods) {
						t.Fatalf("request order: %v expected %v", methods, expectedMethods)
					}
				})
			}
		})
	}
}

func TestAPI01_NodeWriteStopsOnReadOrTransportFailure(t *testing.T) {
	for _, mode := range []string{"write_transport", "read_transport", "read_503", "missing_etag", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if mode == "write_transport" || mode == "read_transport" && r.Method == "GET" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
					return
				}
				if r.Method != "GET" {
					w.WriteHeader(412)
					_, _ = io.WriteString(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT","details":{"current_resource_version":"18446744073709551615"}}}`)
					return
				}
				if mode == "deadline" {
					<-r.Context().Done()
					return
				}
				if mode == "read_503" {
					w.WriteHeader(503)
				}
				_, _ = io.WriteString(w, `{}`)
			}))
			defer server.Close()
			ctx := t.Context()
			if mode == "deadline" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			var trace nodeWriteTrace
			start := time.Now()
			out := (&httpSession{http: server.Client()}).writeNode(ctx, "PATCH", server.URL+"/node", server.URL+"/node", map[string]string{"name": "TEST ONLY"}, `"1"`, nodeFieldsMatch(map[string]string{"name": "TEST ONLY"}), []int{202}, func(_, raw string) { _ = json.Unmarshal([]byte(raw), &trace) })
			if out.accepted(202) || trace.Status != StatusBlocked || len(trace.Attempts) != 1 || time.Since(start) > 2*time.Second {
				t.Fatalf("result %+v trace %+v", out, trace)
			}
			if mode != "write_transport" && trace.Attempts[0].CurrentVersion != "18446744073709551615" {
				t.Fatal("version lost precision")
			}
			if mode == "deadline" && (trace.Outcome != "time_limit" || trace.Attempts[0].Reread == nil || !trace.Attempts[0].Reread.Transport) {
				t.Fatalf("deadline receipt %+v", trace)
			}
		})
	}
}

func TestAPI01_NodeIntendedRequiresEveryDesiredValue(t *testing.T) {
	group := uuid.New()
	for _, body := range []string{`{}`, `{"publication_intent":"published"}`, `{"publication_intent":"published","group_ids":[]}`, `{"publication_intent":"unpublished","group_ids":["` + group.String() + `"]}`} {
		if publishedNode(group)([]byte(body)) {
			t.Fatalf("incomplete intent matched: %s", body)
		}
	}
	capacity := map[string]any{"capacity": map[string]int64{"down_bps": 9007199254740991, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}
	if nodeFieldsMatch(capacity)([]byte(`{"capacity":{"down_bps":9007199254740990,"up_bps":200000000,"member_down_bps_cap":100000000,"member_up_bps_cap":100000000}}`)) {
		t.Fatal("different capacity matched")
	}
}

func TestAPI01_NodeWriteTotalDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(412)
			_, _ = io.WriteString(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT"}}`)
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	var trace nodeWriteTrace
	start := time.Now()
	out := (&httpSession{http: server.Client()}).writeNode(t.Context(), "PATCH", server.URL+"/node", server.URL+"/node", nil, `"1"`, revokedNode, []int{202}, func(_, raw string) { _ = json.Unmarshal([]byte(raw), &trace) })
	elapsed := time.Since(start)
	if out.accepted(202) || trace.Status != StatusBlocked || trace.Outcome != "time_limit" || len(trace.Attempts) != 1 || trace.Attempts[0].Reread == nil || !trace.Attempts[0].Reread.Transport || elapsed < nodeWriteTimeout || elapsed > nodeWriteTimeout+2*time.Second {
		t.Fatalf("total deadline: elapsed=%s trace=%+v", elapsed, trace)
	}
}

func TestAPI01_NodeWriteNoImplicitReplayOrRedirect(t *testing.T) {
	for _, mode := range []string{"reused_connection_eof", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			var writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				n := writes.Add(1)
				if n > 1 {
					w.WriteHeader(202)
					return
				}
				if mode == "redirect" {
					w.Header().Set("Location", "/redirected")
					w.WriteHeader(307)
					return
				}
				conn, _, err := w.(http.Hijacker).Hijack()
				if err == nil {
					_ = conn.Close()
				}
			}))
			defer server.Close()
			client := &httpSession{http: server.Client()}
			// Warm the original connection. Default net/http may replay a keyed
			// PATCH when this reused connection closes before its first byte.
			if _, err := client.call(t.Context(), "GET", server.URL+"/node", nil, nil); err != nil {
				t.Fatal(err)
			}
			var trace nodeWriteTrace
			out := client.writeNode(t.Context(), "PATCH", server.URL+"/node", server.URL+"/node", map[string]string{"name": "TEST ONLY"}, `"1"`, revokedNode, []int{202}, func(_, raw string) { _ = json.Unmarshal([]byte(raw), &trace) })
			if out.accepted(202) || writes.Load() != 1 || len(trace.Attempts) != 1 || trace.Attempts[0].Reread != nil {
				t.Fatalf("unrecorded/retried request: writes=%d trace=%+v", writes.Load(), trace)
			}
			if mode == "redirect" && out.Reply.Status != 307 || mode == "reused_connection_eof" && out.Err == nil {
				t.Fatalf("unexpected result: %+v", out)
			}
		})
	}
}
