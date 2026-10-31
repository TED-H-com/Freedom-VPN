package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAT22TransportFaultScopeAndRecovery(t *testing.T) {
	dir := t.TempDir()
	f := resolverFault{marker: filepath.Join(dir, "toggle"), evidence: filepath.Join(dir, "evidence")}
	for _, tc := range []struct {
		marker, path string
		status       int
		changed      bool
	}{
		{"", "/api/v1/agent/resolver-documents/one", 200, false},
		{"*", "/api/v1/agent/desired-state", 200, false},
		{"*", "/api/v1/agent/resolver-documents/one", 404, false},
		{"two", "/api/v1/agent/resolver-documents/one", 200, false},
		{"one", "/api/v1/agent/resolver-documents/one", 200, true},
		{"*", "/api/v1/agent/resolver-documents/two", 200, true},
		{"", "/api/v1/agent/resolver-documents/two", 200, false},
	} {
		if tc.marker == "" {
			_ = os.Remove(f.marker)
		} else if err := os.WriteFile(f.marker, []byte(tc.marker), 0600); err != nil {
			t.Fatal(err)
		}
		const body = `{"TEST ONLY":"synthetic document"}`
		resp := &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: httptest.NewRequest("GET", tc.path, nil), ContentLength: int64(len(body))}
		if err := f.modify(resp); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if (string(got) != body) != tc.changed || len(got) != len(body) || resp.ContentLength != int64(len(body)) {
			t.Fatalf("fault scope/length: %+v", tc)
		}
	}
	if f.count != 2 {
		t.Fatalf("fault count %d", f.count)
	}
}

func TestUsageHoldIsOnlyTheUsagePost(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "hold-usage")
	rec := httptest.NewRecorder()
	if withholdUsage(rec, httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", nil), marker) {
		t.Fatal("missing marker withheld")
	}
	if err := os.WriteFile(marker, []byte("hold\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if withholdUsage(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/agent/usage-batches", nil), marker) {
		t.Fatal("GET was withheld")
	}
	if withholdUsage(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/agent/desired-state", nil), marker) {
		t.Fatal("other POST was withheld")
	}
	rec = httptest.NewRecorder()
	if !withholdUsage(rec, httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", strings.NewReader("{}")), marker) {
		t.Fatal("usage POST was forwarded")
	}
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"code":"DEPENDENCY_UNAVAILABLE"`) {
		t.Fatalf("withhold response %d %s", rec.Code, rec.Body.String())
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if withholdUsage(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", nil), marker) {
		t.Fatal("released marker still withheld")
	}
}
