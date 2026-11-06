package nodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// validateServingRead applies the same constraints as the optional HTTP field to
// typed service callers. A read is diagnostic only: neither freshness nor its
// relationship to the applied generation or checks is a validity condition.
func validateServingRead(read presence.Nullable[api.ServingReadEvidence]) error {
	v, err := read.Get()
	if err != nil {
		return nil
	}
	field, reason := "", ""
	switch {
	case !v.Source.Valid():
		field, reason = "source", "must be apply or live"
	case !utf8.ValidString(v.Diagnostic) || utf8.RuneCountInString(v.Diagnostic) > 500:
		field, reason = "diagnostic", "must be valid UTF-8 with at most 500 characters"
	default:
		_, offset := v.ReadAt.Zone()
		if offset != 0 || v.ReadAt.Year() < 0 || v.ReadAt.Year() > 9999 {
			field, reason = "read_at", "must be a UTC RFC3339 timestamp"
		}
	}
	if field != "" {
		return apierr.Validation("Invalid serving read evidence.", apierr.FieldError{Field: "/serving_read/" + field, Reason: reason})
	}
	return nil
}

// decodeServingRead must not turn a corrupt or incomplete persisted object into
// a successful zero value (or null). Schema validation on ingestion does not
// validate legacy rows or direct database writes.
func decodeServingRead(raw []byte) (presence.Nullable[api.ServingReadEvidence], error) {
	out := presence.NewNullNullable[api.ServingReadEvidence]()
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return out, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return out, fmt.Errorf("decode serving read: %w", err)
	}
	keys := []string{"read_at", "source", "candidate_generation", "verified", "diagnostic"}
	if len(fields) != len(keys) {
		return out, fmt.Errorf("invalid persisted serving read fields")
	}
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return out, fmt.Errorf("invalid persisted serving read field %s", key)
		}
	}
	var v api.ServingReadEvidence
	if err := json.Unmarshal(raw, &v); err != nil {
		return out, fmt.Errorf("decode serving read: %w", err)
	}
	out.Set(v)
	if err := validateServingRead(out); err != nil {
		// Persisted corruption is an internal failure, not invalid client input.
		// Do not wrap apierr.Validation: HTTP error mapping follows errors.As.
		return presence.NewNullNullable[api.ServingReadEvidence](), fmt.Errorf("invalid persisted serving read value")
	}
	return out, nil
}
