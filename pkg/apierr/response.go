package apierr

import "net/http"

// ResponseWriter records whether a final response has started. Install it at
// the outer handler boundary, before any middleware can write. net/http does
// not expose write state on an arbitrary unwrapped ResponseWriter.
// Use http.ResponseController for optional operations such as flushing.
type ResponseWriter struct {
	http.ResponseWriter
	written bool
}

// TrackResponse wraps w before its first write. Rewrapping our writer keeps
// its state; middleware must pass this writer on to handlers and apierr.Write.
func TrackResponse(w http.ResponseWriter) *ResponseWriter {
	if tracked, ok := w.(*ResponseWriter); ok {
		return tracked
	}
	return &ResponseWriter{ResponseWriter: w}
}

func (w *ResponseWriter) Written() bool               { return w.written }
func (w *ResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *ResponseWriter) WriteHeader(status int) {
	if w.Written() {
		return
	}
	// Informational responses do not commit the final response, except the
	// switching-protocols response which gives up HTTP response ownership.
	if status == http.StatusSwitchingProtocols || status >= 200 {
		w.written = true
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *ResponseWriter) Write(body []byte) (int, error) {
	if !w.Written() {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
func (w *ResponseWriter) FlushError() error {
	if !w.Written() {
		w.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func responseWritten(w http.ResponseWriter) bool {
	for {
		if state, ok := w.(interface{ Written() bool }); ok && state.Written() {
			return true
		}
		inner, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = inner.Unwrap()
	}
}
