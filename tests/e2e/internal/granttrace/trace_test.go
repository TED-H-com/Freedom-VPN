package granttrace

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestD087_RetainsOriginalWireBytesAndActualStatus(t *testing.T) {
	const request = "{\n  \"report_id\": \"test\"\n}\n"
	const response = "{\"accepted\": true}\n"
	recorder := &Recorder{}
	handler := recorder.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(202)
		_, _ = w.Write([]byte(response))
	}))
	q := httptest.NewRequest("POST", "/api/v1/agent/observations", strings.NewReader(request))
	q.Header.Set("Authorization", "secret never retained")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, q)
	if len(recorder.events) != 1 {
		t.Fatal("missing original event")
	}
	e := recorder.events[0]
	h := sha256.Sum256([]byte(response))
	if e.Status != 202 || e.RetryAfter != "5" || e.RequestWire != request || e.ResponseWire != response || e.ResponseSHA256 != hex.EncodeToString(h[:]) || w.Body.String() != response {
		t.Fatalf("wire bytes/status changed: %+v", e)
	}
}
