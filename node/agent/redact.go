package agent

import (
	"context"
	"log/slog"
	"strings"
)

// secretKeys are log attribute names that must never be recorded (§16.2).
var secretKeys = map[string]bool{
	"token": true, "setup_key": true, "authorization": true,
	"private_key": true, "proof": true, "password": true,
}

// RedactHandler drops secret attributes and scrubs secret-shaped words from
// messages. It does not try to discover a token it has not been told about;
// callers must not log secrets in the first place.
type RedactHandler struct{ Next slog.Handler }

// Enabled reports whether the next handler wants the record.
func (h RedactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.Next == nil {
		return false
	}
	return h.Next.Enabled(ctx, level)
}

// Handle writes a redacted record.
func (h RedactHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.Next == nil {
		return nil
	}
	nr := slog.NewRecord(r.Time, r.Level, scrub(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(scrubAttr(a))
		return true
	})
	return h.Next.Handle(ctx, nr)
}

// WithAttrs returns a handler with additional redacted attributes.
func (h RedactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		out[i] = scrubAttr(a)
	}
	if h.Next == nil {
		return h
	}
	h.Next = h.Next.WithAttrs(out)
	return h
}

// WithGroup returns a handler with the group applied to the next handler.
func (h RedactHandler) WithGroup(name string) slog.Handler {
	if h.Next == nil {
		return h
	}
	h.Next = h.Next.WithGroup(name)
	return h
}

func scrubAttr(a slog.Attr) slog.Attr {
	if secretKeys[strings.ToLower(a.Key)] {
		a.Value = slog.StringValue("[redacted]")
		return a
	}
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(scrub(a.Value.String()))
	}
	return a
}

func scrub(s string) string {
	if strings.Contains(s, "Enrollment ") {
		return "agent: redacted authorization"
	}
	return s
}
