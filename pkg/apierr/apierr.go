// Package apierr defines the API error codes of the Project Spec (§13.11,
// §14.6 and domain codes from §6, §9, §10, §16) and the JSON error envelope
// of AGENTS.md §4:
//
//	{"error": {"code": "...", "message": "...", "request_id": "...", "retryable": false, "details": {}}}
//
// Write renders an error safely: only *Error values reach the client, and
// any other error becomes a generic INTERNAL_ERROR with the request ID. Stack
// traces, SQL, certificates and secrets are never written (§14.6).
package apierr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// HeaderRequestID is the response header that carries the central request ID.
const HeaderRequestID = "X-Request-ID"

// Error is an API error. Message must be safe to show to the caller; Cause
// is internal and never serialized.
type Error struct {
	Code    Code
	Message string
	// Retryable overrides the code's default retryable flag when non-nil.
	Retryable *bool
	// Details is a JSON object with machine-readable context, for example
	// {"current_resource_version": "43"} or {"fields": [...]}. It must not
	// contain secrets.
	Details map[string]any
	// RetryAfter sets the Retry-After header (whole seconds, rounded up).
	RetryAfter time.Duration
	cause      error
}

// New returns an error with the given code and safe message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf is like New with a formatted message. Never format secrets or
// internal error text into a message.
func Newf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap returns an error with code and message whose internal cause is err.
// The cause is available to errors.Is/As and logs but never sent to clients.
func Wrap(err error, code Code, message string) *Error {
	return &Error{Code: code, Message: message, cause: err}
}

func (e *Error) clone() *Error {
	c := *e
	c.Details = maps.Clone(e.Details)
	return &c
}

// WithDetail returns a copy of e with details[key] = value.
func (e *Error) WithDetail(key string, value any) *Error {
	c := e.clone()
	if c.Details == nil {
		c.Details = map[string]any{}
	}
	c.Details[key] = value
	return c
}

// WithRetryable returns a copy of e with an explicit retryable flag.
func (e *Error) WithRetryable(v bool) *Error {
	c := e.clone()
	c.Retryable = &v
	return c
}

// WithRetryAfter returns a copy of e with a Retry-After hint.
func (e *Error) WithRetryAfter(d time.Duration) *Error {
	c := e.clone()
	c.RetryAfter = d
	return c
}

// WithCause returns a copy of e with an internal cause.
func (e *Error) WithCause(err error) *Error {
	c := e.clone()
	c.cause = err
	return c
}

// Error implements error. It includes the cause for internal logging.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the internal cause.
func (e *Error) Unwrap() error { return e.cause }

// Is reports whether target is an *Error with the same code, so
// errors.Is(err, apierr.New(apierr.StateConflict, "")) matches by code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// HTTPStatus returns the HTTP status for the error's code.
func (e *Error) HTTPStatus() int { return e.Code.HTTPStatus() }

// IsRetryable returns the effective retryable flag.
func (e *Error) IsRetryable() bool {
	if e.Retryable != nil {
		return *e.Retryable
	}
	return e.Code.DefaultRetryable()
}

// FieldError describes one invalid field for VALIDATION_FAILED details.
type FieldError struct {
	Field  string `json:"field"`  // JSON pointer or dotted path, e.g. "/records/0/upload_bytes"
	Reason string `json:"reason"` // short machine-friendly reason, e.g. "decimal_string"
}

// Validation returns a VALIDATION_FAILED error listing the given fields in
// details.fields.
func Validation(message string, fields ...FieldError) *Error {
	e := New(ValidationFailed, message)
	if fields == nil {
		fields = []FieldError{}
	}
	return e.WithDetail("fields", fields)
}

// Body is the inner object of the error envelope.
type Body struct {
	Code      Code           `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details"`
}

// Envelope is the JSON error envelope.
type Envelope struct {
	Error Body `json:"error"`
}

type ctxKey struct{}

// ContextWithRequestID returns a context carrying the central request ID.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestIDFromContext returns the request ID stored by ContextWithRequestID.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok && id != ""
}

// Writer writes error responses. The zero value is ready to use and does not
// log. With Logger set, errors that are not *Error (and so are hidden from
// the client) are logged with their request ID; the log handler is
// responsible for redaction (AGENTS.md §3).
type Writer struct {
	Logger *slog.Logger
}

// Write is Writer{}.Write.
func Write(w http.ResponseWriter, r *http.Request, err error) {
	Writer{}.Write(w, r, err)
}

const internalMessage = "Internal error. Quote the request ID when reporting this problem."

// Write renders err as the error envelope.
//
// The request ID is taken from the request context (ContextWithRequestID),
// then from an X-Request-ID header already set on the response, and is
// generated otherwise. A client-supplied X-Request-ID request header is never
// used (§14.1). The response always carries X-Request-ID, Cache-Control:
// no-store and, for 429/503 or when RetryAfter is set, Retry-After.
func (wr Writer) Write(w http.ResponseWriter, r *http.Request, err error) {
	reqID := ""
	if r != nil {
		reqID, _ = RequestIDFromContext(r.Context())
	}
	if reqID == "" {
		reqID = w.Header().Get(HeaderRequestID)
	}
	if reqID == "" {
		reqID = newRequestID()
	}

	var ae *Error
	if err == nil || !errors.As(err, &ae) || !ae.Code.Valid() {
		if wr.Logger != nil {
			ctx := context.Background()
			if r != nil {
				ctx = r.Context()
			}
			wr.Logger.LogAttrs(ctx, slog.LevelError, "unhandled error rendered as INTERNAL_ERROR",
				slog.String("request_id", reqID), slog.String("error_type", fmt.Sprintf("%T", err)), slog.Any("error", err))
		}
		ae = New(InternalError, internalMessage)
	}

	details := ae.Details
	if details == nil {
		details = map[string]any{}
	}
	env := Envelope{Error: Body{
		Code:      ae.Code,
		Message:   ae.Message,
		RequestID: reqID,
		Retryable: ae.IsRetryable(),
		Details:   details,
	}}
	body, mErr := json.Marshal(env)
	if mErr != nil {
		// Details held something unencodable; drop them rather than leak
		// or fail.
		env.Error.Details = map[string]any{}
		body, _ = json.Marshal(env)
	}

	h := w.Header()
	h.Set(HeaderRequestID, reqID)
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	status := ae.HTTPStatus()
	if ra := retryAfterSeconds(ae, status); ra > 0 {
		h.Set("Retry-After", strconv.Itoa(ra))
	}
	h.Del("ETag")
	h.Del("Content-Length")
	w.WriteHeader(status)
	if r != nil && r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}

func retryAfterSeconds(e *Error, status int) int {
	if e.RetryAfter > 0 {
		s := int(e.RetryAfter / time.Second)
		if e.RetryAfter%time.Second != 0 {
			s++
		}
		return s
	}
	if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		return 1
	}
	return 0
}

func newRequestID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
