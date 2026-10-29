package apierr

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestReviewC07_ExpiredAuthorizationRequiresNewJob(t *testing.T) {
	if AuthorizationExpired.HTTPStatus() != http.StatusConflict || AuthorizationExpired.DefaultRetryable() {
		t.Fatal("expired job must not be retried unchanged")
	}
}

func TestReviewC12_NoErrorAfterResponseStarts(t *testing.T) {
	for _, start := range []string{"body", "header", "error", "flush"} {
		t.Run(start, func(t *testing.T) {
			rec := httptest.NewRecorder()
			w := TrackResponse(rec)
			if TrackResponse(w) != w {
				t.Fatal("rewrapping lost state")
			}
			switch start {
			case "body":
				_, _ = w.Write([]byte("OK"))
			case "header":
				w.WriteHeader(http.StatusNoContent)
			case "error":
				Write(w, nil, New(StateConflict, "first"))
			case "flush":
				if err := http.NewResponseController(w).Flush(); err != nil {
					t.Fatal(err)
				}
			}
			code, body, headers := rec.Code, rec.Body.String(), rec.Header().Clone()
			Write(w, nil, New(MalformedRequest, "second"))
			w.WriteHeader(http.StatusInternalServerError)
			if rec.Code != code || rec.Body.String() != body || !reflect.DeepEqual(rec.Header(), headers) {
				t.Fatal("second error changed the response")
			}
		})
	}
	// Informational responses must still permit the final error response.
	rec := httptest.NewRecorder()
	w := TrackResponse(rec)
	w.WriteHeader(http.StatusEarlyHints)
	if w.Written() {
		t.Fatal("103 committed final response")
	}
	Write(w, nil, New(MalformedRequest, "final"))
	if !w.Written() || rec.Body.Len() == 0 {
		t.Fatal("final response was refused")
	}
}

func TestReviewC13_RequestIDMustBeUUID(t *testing.T) {
	for _, bad := range []string{"not-a-uuid", "", "11111111111141118111111111111111"} {
		rec := httptest.NewRecorder()
		rec.Header().Set(HeaderRequestID, bad)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderRequestID, "client-chosen")
		Write(rec, req, New(StateConflict, "conflict"))
		got := rec.Header().Get(HeaderRequestID)
		if _, err := uuid.Parse(got); err != nil || len(got) != 36 {
			t.Fatalf("bad response ID: %q", got)
		}
		if decode(t, rec).Error.RequestID != got {
			t.Fatal("body/header ID mismatch")
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(ContextWithRequestID(req.Context(), "invalid-context-id"))
	Write(rec, req, New(StateConflict, "conflict"))
	if _, err := uuid.Parse(rec.Header().Get(HeaderRequestID)); err != nil {
		t.Fatal(err)
	}
}
