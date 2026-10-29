package apierr

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestTableInvariants(t *testing.T) {
	codeRE := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	required := []Code{
		// §14.6
		MalformedRequest, Unauthenticated, Forbidden, ResourceNotFound, IdempotencyConflict,
		StateConflict, ResourceVersionConflict, PreconditionRequired, ValidationFailed,
		NodeNotServiceable, RateLimited, DependencyUnavailable,
		// §13.11
		ReleaseNotApproved, UnsupportedPlatform, ControlAPIIncompatible, SchemaIncompatible,
		SignatureInvalid, KeyUntrusted, HashMismatch, SizeMismatch, DiskSpaceLow, DownloadTimeout,
		StoreUnavailable, UpdateBusy, AuthorizationExpired, StaleGeneration, IdentityRevoked,
		IncarnationMismatch, CheckpointFailed, MigrationFailed, StartFailed, HealthCheckFailed,
		MinimumVersionViolation, RollbackFailed, AckConflict,
		// domain
		UsageEventConflict, MeterPathInvalid, DeviceLimitReached, EnrollmentTokenInvalid,
		EnrollmentTokenExpired, EnrollmentTokenRevoked, EnrollmentKeyMismatch, LeaseExpired,
	}
	for _, c := range required {
		if !c.Valid() {
			t.Errorf("required code %s missing", c)
		}
	}
	for _, s := range All() {
		if !codeRE.MatchString(string(s.Code)) {
			t.Errorf("bad code %q", s.Code)
		}
		if s.HTTPStatus < 400 || s.HTTPStatus > 599 || http.StatusText(s.HTTPStatus) == "" {
			t.Errorf("%s: bad status %d", s.Code, s.HTTPStatus)
		}
		if s.Section == "" || s.Description == "" {
			t.Errorf("%s: missing section or description", s.Code)
		}
		got, ok := Lookup(s.Code)
		if !ok || got != s {
			t.Errorf("Lookup(%s)", s.Code)
		}
	}
}

// TestStatusesMatchSpec14_6 pins the statuses the spec states explicitly.
func TestStatusesMatchSpec14_6(t *testing.T) {
	want := map[Code]int{
		MalformedRequest: 400, Unauthenticated: 401, Forbidden: 403, ResourceNotFound: 404,
		IdempotencyConflict: 409, StateConflict: 409, ResourceVersionConflict: 412,
		PreconditionRequired: 428, ValidationFailed: 422, NodeNotServiceable: 422,
		RateLimited: 429, DependencyUnavailable: 503,
		UsageEventConflict: 409, AckConflict: 409, ReleaseNotApproved: 422,
	}
	for c, s := range want {
		if c.HTTPStatus() != s {
			t.Errorf("%s: status %d, want %d", c, c.HTTPStatus(), s)
		}
	}
	if !RateLimited.DefaultRetryable() || !DependencyUnavailable.DefaultRetryable() {
		t.Error("429/503 must be retryable")
	}
	if ValidationFailed.DefaultRetryable() || IdempotencyConflict.DefaultRetryable() || IdentityRevoked.DefaultRetryable() {
		t.Error("client errors must not be retryable")
	}
	if Code("NOPE").Valid() || Code("NOPE").HTTPStatus() != 500 || Code("NOPE").DefaultRetryable() {
		t.Error("unknown code handling")
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) Envelope {
	t.Helper()
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	var env Envelope
	if err := dec.Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return env
}

func TestWriteTypedError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/nodes/x", nil)
	req = req.WithContext(ContextWithRequestID(req.Context(), "a12d4141-4f3e-487b-b638-e1488be3d4ee"))
	rec.Header().Set("ETag", `"42"`)
	e := New(ResourceVersionConflict, "節點設定已有更新，請重新讀取後再儲存。").WithDetail("current_resource_version", "43")
	Write(rec, req, e)
	if rec.Code != 412 {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") != "a12d4141-4f3e-487b-b638-e1488be3d4ee" {
		t.Fatal("request id header")
	}
	if rec.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatal("headers")
	}
	if rec.Header().Get("ETag") != "" || rec.Header().Get("Retry-After") != "" {
		t.Fatal("unexpected ETag or Retry-After")
	}
	env := decode(t, rec)
	if env.Error.Code != ResourceVersionConflict || env.Error.Retryable || env.Error.RequestID != "a12d4141-4f3e-487b-b638-e1488be3d4ee" ||
		env.Error.Details["current_resource_version"] != "43" || env.Error.Message == "" {
		t.Fatalf("%+v", env)
	}
}

func TestWriteHidesUnknownErrors(t *testing.T) {
	secret := errors.New(`pq: duplicate key "token_hash" value=sk_live_abc SELECT * FROM members; goroutine 1 [running]`)
	var logBuf bytes.Buffer
	wr := Writer{Logger: slog.New(slog.NewJSONHandler(&logBuf, nil))}
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", "11111111-1111-4111-8111-111111111111")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "client-chosen") // must be ignored
	wr.Write(rec, req, secret)
	if rec.Code != 500 {
		t.Fatalf("status %d", rec.Code)
	}
	raw := rec.Body.String()
	for _, leak := range []string{"pq:", "token_hash", "sk_live", "SELECT", "goroutine"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("response leaks %q: %s", leak, raw)
		}
	}
	var e Envelope
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatal(err)
	}
	if e.Error.Code != InternalError || e.Error.RequestID != "11111111-1111-4111-8111-111111111111" || e.Error.Retryable {
		t.Fatalf("%+v", e)
	}
	if e.Error.Details == nil || len(e.Error.Details) != 0 {
		t.Fatal("details must be an empty object")
	}
	if !strings.Contains(logBuf.String(), "11111111-1111-4111-8111-111111111111") {
		t.Fatal("unknown error not logged with request id")
	}
}

func TestWriteGeneratesRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, nil, nil)
	id := rec.Header().Get("X-Request-ID")
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("request id %q is not a UUIDv7", id)
	}
	var e Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if e.Error.RequestID != id || e.Error.Code != InternalError {
		t.Fatalf("%+v", e)
	}
}

func TestWriteRetryAfterAndRetryable(t *testing.T) {
	cases := []struct {
		err       *Error
		status    int
		retry     string
		retryable bool
	}{
		{New(RateLimited, "slow down"), 429, "1", true},
		{New(RateLimited, "slow down").WithRetryAfter(2500 * time.Millisecond), 429, "3", true},
		{New(DependencyUnavailable, "try later").WithRetryAfter(5 * time.Second), 503, "5", true},
		{New(StateConflict, "x").WithRetryable(true), 409, "", true},
		{New(DependencyUnavailable, "x").WithRetryable(false), 503, "1", false},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		Write(rec, httptest.NewRequest(http.MethodPost, "/", nil), tc.err)
		if rec.Code != tc.status || rec.Header().Get("Retry-After") != tc.retry {
			t.Errorf("%s: status %d retry-after %q", tc.err.Code, rec.Code, rec.Header().Get("Retry-After"))
		}
		var e Envelope
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
		if e.Error.Retryable != tc.retryable {
			t.Errorf("%s: retryable %v", tc.err.Code, e.Error.Retryable)
		}
	}
}

func TestWriteHEADHasNoBody(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, httptest.NewRequest(http.MethodHead, "/", nil), New(ResourceNotFound, "not found"))
	if rec.Code != 404 || rec.Body.Len() != 0 {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
}

func TestWriteWrappedAndInvalidCodes(t *testing.T) {
	inner := Wrap(errors.New("db down"), DependencyUnavailable, "Service temporarily unavailable.")
	wrapped := errorsJoin(inner)
	rec := httptest.NewRecorder()
	Write(rec, httptest.NewRequest(http.MethodGet, "/", nil), wrapped)
	if rec.Code != 503 || strings.Contains(rec.Body.String(), "db down") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	// A typed error with an undefined code is treated as internal.
	rec = httptest.NewRecorder()
	Write(rec, nil, New("MADE_UP", "x"))
	if rec.Code != 500 {
		t.Fatalf("undefined code status %d", rec.Code)
	}
	// Unencodable details are dropped.
	rec = httptest.NewRecorder()
	Write(rec, nil, New(StateConflict, "x").WithDetail("f", func() {}))
	var e Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || len(e.Error.Details) != 0 || e.Error.Code != StateConflict {
		t.Fatalf("%s %v", rec.Body.String(), err)
	}
}

func errorsJoin(err error) error { return errors.Join(errors.New("context"), err) }

func TestErrorHelpers(t *testing.T) {
	base := New(StateConflict, "conflict")
	d := base.WithDetail("a", 1)
	if base.Details != nil {
		t.Fatal("WithDetail mutated receiver")
	}
	d2 := d.WithDetail("b", 2)
	if len(d.Details) != 1 || len(d2.Details) != 2 {
		t.Fatal("WithDetail shares map")
	}
	cause := errors.New("root cause")
	c := base.WithCause(cause)
	if !errors.Is(c, cause) || errors.Unwrap(c) != cause {
		t.Fatal("cause unwrap")
	}
	if !strings.Contains(c.Error(), "root cause") || base.Error() != "STATE_CONFLICT: conflict" {
		t.Fatal(c.Error())
	}
	if !errors.Is(c, New(StateConflict, "")) || errors.Is(c, New(ValidationFailed, "")) || c.Is(cause) {
		t.Fatal("Is by code")
	}
	if Newf(ValidationFailed, "bad %s", "x").Message != "bad x" {
		t.Fatal("Newf")
	}
	v := Validation("invalid", FieldError{Field: "/event_seq", Reason: "decimal_string"})
	if v.Code != ValidationFailed || len(v.Details["fields"].([]FieldError)) != 1 {
		t.Fatal("Validation")
	}
	if fs := Validation("invalid").Details["fields"].([]FieldError); fs == nil {
		t.Fatal("Validation fields must be a non-nil empty list")
	}
	if id, ok := RequestIDFromContext(ContextWithRequestID(httptest.NewRequest("GET", "/", nil).Context(), "")); ok || id != "" {
		t.Fatal("empty request id")
	}
	if New(ValidationFailed, "").HTTPStatus() != 422 || !New(RateLimited, "").IsRetryable() {
		t.Fatal("status/retryable")
	}
}
