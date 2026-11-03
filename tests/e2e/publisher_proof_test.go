package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// publisherTrace32 checks the live proof without requiring optional error
// details. The real API rejects the TEST ONLY "0" in ParseIfMatch, before a
// version lookup, so that 412 has empty details. Ordinary version mismatches
// include current_resource_version; writeNode records it whenever supplied.
func publisherTrace32(trace nodeWriteTrace, forced bool) (conflicts int, valid bool) {
	if len(trace.Attempts) == 0 {
		return 0, false
	}
	first := trace.Attempts[0]
	valid = trace.Status == StatusPass && trace.Outcome == "accepted" && first.IdempotencyKey != "" && len(first.BodySHA256) == 64
	for i, a := range trace.Attempts {
		valid = valid && !a.Transport && a.IdempotencyKey == first.IdempotencyKey && a.BodySHA256 == first.BodySHA256
		if a.Status == 412 && a.ErrorCode == "RESOURCE_VERSION_CONFLICT" {
			conflicts++
			read := a.Reread
			valid = valid && read != nil && read.Method == "GET" && read.Status == 200 && !read.Transport && read.ErrorCode == ""
			valid = valid && (a.CurrentVersion == "" || decimalVersion(a.CurrentVersion))
			if i+1 < len(trace.Attempts) {
				valid = valid && read != nil && trace.Attempts[i+1].IfMatch == read.ETag
			}
		} else {
			valid = valid && i == len(trace.Attempts)-1 && a.Status == 202 && a.Reread == nil
		}
	}
	valid = valid && trace.Attempts[len(trace.Attempts)-1].Status == 202
	if forced {
		valid = valid && len(trace.Attempts) == 2 && conflicts == 1 && first.IfMatch == `"0"`
	}
	return conflicts, valid
}

func TestAPI01_PublisherProofOptionalConflictVersion(t *testing.T) {
	for _, details := range []string{`{}`, `{"current_resource_version":"6"}`} {
		t.Run(details, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method)
				switch len(requests) {
				case 1:
					w.WriteHeader(412)
					_, _ = fmt.Fprintf(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT","details":%s}}`, details)
				case 2:
					w.Header().Set("ETag", `"6"`)
					_, _ = io.WriteString(w, `{}`)
				case 3:
					w.WriteHeader(202)
				default:
					t.Error("unexpected extra request")
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			var trace nodeWriteTrace
			body := map[string]string{"name": "TEST ONLY"}
			out := (&httpSession{http: server.Client()}).writeNode(t.Context(), "PATCH", server.URL+"/node", server.URL+"/node", body, `"0"`, nodeFieldsMatch(body), []int{202}, func(_, raw string) {
				if err := json.Unmarshal([]byte(raw), &trace); err != nil {
					t.Fatal(err)
				}
			})
			mu.Lock()
			defer mu.Unlock()
			conflicts, valid := publisherTrace32(trace, true)
			if !out.accepted(202) || !valid || conflicts != 1 || fmt.Sprint(requests) != "[PATCH GET PATCH]" || trace.Attempts[1].IfMatch != `"6"` {
				t.Fatalf("proof rejected successful conflict recovery: %+v", trace)
			}
			wantVersion := "6"
			if details == `{}` {
				wantVersion = ""
			}
			if trace.Attempts[0].CurrentVersion != wantVersion {
				t.Fatal("current version was invented or lost")
			}
			trace.Attempts[1].IfMatch = `"5"`
			if _, valid := publisherTrace32(trace, true); valid {
				t.Fatal("proof accepted retry without the re-read ETag")
			}
		})
	}
}
