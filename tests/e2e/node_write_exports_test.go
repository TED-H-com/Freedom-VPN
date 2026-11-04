//go:build e2e

package e2e

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAPI01_ExportedNodeWrite(t *testing.T) {
	for _, tc := range []struct {
		name, outcome string
		writes, reads int
		accepted      bool
	}{
		{"retry", "accepted", 2, 1, true},
		{"already", "already_intended", 1, 1, true},
		{"exhausted", "attempt_limit", 3, 3, false},
		{"other412", "non_conflict_response", 1, 0, false},
		{"redirect", "non_conflict_response", 1, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const path = "/api/v1/admin/nodes/test-only"
			const payload = `{"capacity":{"down_bps":200000000,"up_bps":200000000}}`
			writes, reads := 0, 0
			var keys, tags, bodies []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie("TEST_ONLY_session")
				if err != nil || cookie.Value != "test-only" || r.Header.Get("X-CSRF-Token") != "TEST ONLY csrf" {
					t.Error("caller cookie/CSRF missing")
				}
				if r.URL.Path != path {
					t.Errorf("redirect/path followed: %s", r.URL.Path)
				}
				if r.Method == "GET" {
					reads++
					w.Header().Set("ETag", fmt.Sprintf(`"%d"`, reads+1))
					if tc.name == "already" {
						_, _ = io.WriteString(w, payload)
					} else {
						_, _ = io.WriteString(w, `{}`)
					}
					return
				}
				writes++
				raw, _ := io.ReadAll(r.Body)
				keys = append(keys, r.Header.Get("Idempotency-Key"))
				tags = append(tags, r.Header.Get("If-Match"))
				bodies = append(bodies, string(raw))
				if tc.name == "redirect" {
					w.Header().Set("Location", "/redirected")
					w.WriteHeader(307)
				} else if tc.name == "other412" {
					w.WriteHeader(412)
					_, _ = io.WriteString(w, `{"error":{"code":"OTHER"}}`)
				} else if writes == 1 || tc.name == "exhausted" {
					w.WriteHeader(412)
					_, _ = io.WriteString(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT"}}`)
				} else {
					w.WriteHeader(202)
				}
			}))
			defer server.Close()
			jar, _ := cookiejar.New(nil)
			origin, _ := url.Parse(server.URL)
			jar.SetCookies(origin, []*http.Cookie{{Name: "TEST_ONLY_session", Value: "test-only"}})
			client := server.Client()
			client.Jar = jar
			stack := &Stack{lab: &lab{public: server.URL}}
			body := json.RawMessage(payload)
			result := stack.WriteNode(t.Context(), BrowserSession{Client: client, CSRF: "TEST ONLY csrf"}, "PATCH", path, path, body, `"1"`, NodeFieldsMatch(body), 202)
			trace := result.Receipt
			if result.Err != nil || result.Accepted(202) != tc.accepted || result.AlreadyIntended != (tc.name == "already") || writes != tc.writes || reads != tc.reads || trace.Outcome != tc.outcome || len(trace.Attempts) != tc.writes || trace.MaxAttempts != 3 || trace.LimitSeconds != 10 {
				t.Fatalf("result=%+v writes=%d reads=%d", result, writes, reads)
			}
			previous := trace.StartedAt
			for i, row := range trace.Attempts {
				if keys[i] == "" || keys[i] != keys[0] || bodies[i] != payload || row.IfMatch != tags[i] || tags[i] != fmt.Sprintf(`"%d"`, i+1) || row.IdempotencyKey != keys[i] || row.BodySHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(bodies[i]))) {
					t.Fatalf("lost same-key/body or fresh ETag: %+v", row)
				}
				checkUTC := func(start, end time.Time) {
					t.Helper()
					if start.Location() != time.UTC || end.Location() != time.UTC || start.Before(previous) || end.Before(start) {
						t.Fatal("unordered/non-UTC receipt")
					}
					previous = end
				}
				checkUTC(row.StartedAt, row.FinishedAt)
				if read := row.Reread; read != nil {
					checkUTC(read.StartedAt, read.FinishedAt)
					if row.Status != 412 || row.ErrorCode != "RESOURCE_VERSION_CONFLICT" || read.Status != 200 || read.IfMatch != "" || read.BodySHA256 != fmt.Sprintf("%x", sha256.Sum256(nil)) || read.ETag != fmt.Sprintf(`"%d"`, i+2) {
						t.Fatalf("reread=%+v", read)
					}
				}
			}
			if trace.FinishedAt.Before(previous) {
				t.Fatal("trace ended before requests")
			}
			// Receipt embeds directly into an AT record and round-trips losslessly.
			raw, err := json.Marshal(struct {
				NodeWrite NodeWriteReceipt `json:"node_write"`
			}{trace})
			var record struct {
				NodeWrite NodeWriteReceipt `json:"node_write"`
			}
			if err != nil || json.Unmarshal(raw, &record) != nil || !reflect.DeepEqual(record.NodeWrite, trace) {
				t.Fatal("receipt is not embeddable")
			}
		})
	}
}

func TestAPI01_ExportedNodePredicates(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	for _, tc := range []struct {
		groups []uuid.UUID
		want   bool
	}{
		{[]uuid.UUID{b, a}, true}, {[]uuid.UUID{a}, false}, {[]uuid.UUID{a, a}, false}, {[]uuid.UUID{a, c}, false}, {[]uuid.UUID{a, b, c}, false},
	} {
		raw, _ := json.Marshal(map[string]any{"publication_intent": "published", "group_ids": tc.groups})
		if NodePublished(a, b)(raw) != tc.want {
			t.Fatalf("group match: %s", raw)
		}
	}
	if NodeRevoked([]byte(`{"lifecycle_state":"retired"}`)) || !NodeRevoked([]byte(`{"lifecycle_state":"revoked"}`)) {
		t.Fatal("revoked predicate")
	}
}
