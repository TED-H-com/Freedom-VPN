package usageupload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPTransportRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agent/usage-batches" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Idempotency-Key") != "00000000-0000-7000-8000-000000000001" {
			t.Errorf("key %s", r.Header.Get("Idempotency-Key"))
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"n":1}` {
			t.Errorf("body %s", b)
		}
		w.Header().Set("Retry-After", "4")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	tr := &HTTPTransport{BaseURL: srv.URL, Client: srv.Client()}
	status, _, retry, err := tr.PostUsageBatch(context.Background(), "00000000-0000-7000-8000-000000000001", []byte(`{"n":1}`))
	if err != nil || status != 429 || retry != 4*time.Second {
		t.Fatalf("status %d retry %s err %v", status, retry, err)
	}

	when := time.Now().UTC().Add(30 * time.Second).Truncate(time.Second)
	got := parseRetryAfter(when.Format(http.TimeFormat), when.Add(-30*time.Second))
	if got != 30*time.Second {
		t.Fatalf("date retry %s", got)
	}
	cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/v1/agent/usage-checkpoint") {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"signed":{},"signatures":[]}`))
	}))
	defer cp.Close()
	tr.BaseURL = cp.URL
	tr.Client = cp.Client()
	status, body, err := tr.GetUsageCheckpoint(context.Background())
	if err != nil || status != 200 || !strings.Contains(string(body), "signed") {
		t.Fatalf("checkpoint %d %s %v", status, body, err)
	}
}
