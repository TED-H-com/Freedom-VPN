// Package decstr implements the JSON decimal-string contract for unsigned
// 64-bit integers (Project Spec §10.4, §13.10, §14.1).
//
// Bytes, cumulative counters, event sequence numbers, generations, revisions
// and resource versions are exchanged as JSON strings such as "9876543210" so
// that they never pass through float64 or a JavaScript Number. The accepted
// grammar is exactly
//
//	^(0|[1-9][0-9]*)$
//
// with at most 20 digits and a value of at most 18446744073709551615
// (2^64-1). Signs, whitespace, decimal points, exponents, NaN and redundant
// leading zeros are rejected. Values that do not fit are rejected; nothing is
// truncated or reduced modulo 2^64.
//
// The zero value of U64 is 0 and is ready to use.
package decstr

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strconv"
)

// MaxDigits is the maximum number of digits of a valid decimal string.
const MaxDigits = 20

// Pattern is the regular expression every valid decimal string matches. It is
// the same pattern the OpenAPI document and JSON Schemas use.
const Pattern = `^(0|[1-9][0-9]*)$`

// Max is the largest representable value, 18446744073709551615.
const Max = U64(math.MaxUint64)

// U64 is an unsigned 64-bit integer that is encoded as a JSON decimal string.
type U64 uint64

// Sentinel errors. Errors returned by this package wrap one of them, so
// callers can use errors.Is.
var (
	// ErrSyntax reports input that is not a canonical decimal string.
	ErrSyntax = errors.New("decstr: invalid decimal string")
	// ErrRange reports a canonical decimal string above 18446744073709551615.
	ErrRange = errors.New("decstr: value out of range for unsigned 64-bit integer")
	// ErrOverflow reports checked arithmetic whose result exceeds Max.
	ErrOverflow = errors.New("decstr: arithmetic overflow")
	// ErrUnderflow reports checked subtraction whose result would be negative.
	ErrUnderflow = errors.New("decstr: arithmetic underflow")
	// ErrJSONType reports a JSON value that is not a string (for example a
	// raw JSON number or null).
	ErrJSONType = errors.New("decstr: JSON value must be a decimal string")
	// ErrScanType reports a database value of an unsupported Go type.
	ErrScanType = errors.New("decstr: unsupported database value type")
)

// SyntaxError describes why an input was rejected. It wraps ErrSyntax or
// ErrRange.
type SyntaxError struct {
	Input  string // the offending input, truncated to 32 bytes
	Reason string // human readable reason
	err    error
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%v: %q: %s", e.err, e.Input, e.Reason)
}

func (e *SyntaxError) Unwrap() error { return e.err }

func syntaxErr(s, reason string) error {
	return &SyntaxError{Input: truncate(s), Reason: reason, err: ErrSyntax}
}

func truncate(s string) string {
	if len(s) > 32 {
		return s[:32] + "..."
	}
	return s
}

// Parse parses a canonical decimal string.
func Parse(s string) (U64, error) {
	if s == "" {
		return 0, syntaxErr(s, "empty string")
	}
	if len(s) > MaxDigits {
		// Report non-digit content first, so " 1" style inputs that are
		// also long get the more useful message.
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				return 0, syntaxErr(s, "contains a non-digit character")
			}
		}
		if s[0] == '0' {
			return 0, syntaxErr(s, "leading zero")
		}
		return 0, &SyntaxError{Input: truncate(s), Reason: "more than 20 digits", err: ErrRange}
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, syntaxErr(s, "contains a non-digit character")
		}
	}
	if s[0] == '0' && len(s) > 1 {
		return 0, syntaxErr(s, "leading zero")
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// Only a range error is possible after the checks above.
		return 0, &SyntaxError{Input: truncate(s), Reason: "exceeds 18446744073709551615", err: ErrRange}
	}
	return U64(v), nil
}

// MustParse is like Parse but panics on error. Use it only for constants in
// tests and static tables.
func MustParse(s string) U64 {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Valid reports whether s is a canonical decimal string within range.
func Valid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// String returns the canonical decimal representation.
func (u U64) String() string { return strconv.FormatUint(uint64(u), 10) }

// Uint64 returns the value as a uint64.
func (u U64) Uint64() uint64 { return uint64(u) }

// MarshalText implements encoding.TextMarshaler.
func (u U64) MarshalText() ([]byte, error) {
	return strconv.AppendUint(nil, uint64(u), 10), nil
}

// UnmarshalText implements encoding.TextUnmarshaler using the strict grammar.
func (u *U64) UnmarshalText(b []byte) error {
	v, err := Parse(string(b))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

// MarshalJSON implements json.Marshaler. The value is always a JSON string.
func (u U64) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, MaxDigits+2)
	b = append(b, '"')
	b = strconv.AppendUint(b, uint64(u), 10)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements json.Unmarshaler. Only a JSON string holding a
// canonical decimal string is accepted. JSON numbers, null, booleans, objects
// and arrays are rejected, as are strings containing escape sequences (a
// canonical decimal string never needs one).
func (u *U64) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("%w: got %s", ErrJSONType, jsonKind(b))
	}
	inner := b[1 : len(b)-1]
	for _, c := range inner {
		if c == '\\' || c == '"' {
			return syntaxErr(string(inner), "contains a JSON escape or quote")
		}
	}
	return u.UnmarshalText(inner)
}

func jsonKind(b []byte) string {
	if len(b) == 0 {
		return "empty input"
	}
	switch c := b[0]; {
	case c == 'n':
		return "null"
	case c == 't' || c == 'f':
		return "boolean"
	case c == '{':
		return "object"
	case c == '[':
		return "array"
	case c == '-' || (c >= '0' && c <= '9'):
		return "number"
	default:
		return "invalid JSON"
	}
}

// Scan implements sql.Scanner for PostgreSQL NUMERIC(20,0) columns. It accepts
// string and []byte values in the canonical grammar and non-negative int64
// values. NULL is rejected; use a pointer or sql.Null[decstr.U64] for nullable
// columns.
func (u *U64) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return u.UnmarshalText([]byte(v))
	case []byte:
		return u.UnmarshalText(v)
	case int64:
		if v < 0 {
			return fmt.Errorf("%w: negative database value %d", ErrRange, v)
		}
		*u = U64(v)
		return nil
	case uint64:
		*u = U64(v)
		return nil
	case nil:
		return fmt.Errorf("%w: NULL", ErrScanType)
	default:
		return fmt.Errorf("%w: %T", ErrScanType, src)
	}
}

// Value implements driver.Valuer. The value is a decimal string, which
// PostgreSQL casts to NUMERIC(20,0) without loss.
func (u U64) Value() (driver.Value, error) { return u.String(), nil }

// Add returns u+v or ErrOverflow.
func (u U64) Add(v U64) (U64, error) {
	s, carry := bits.Add64(uint64(u), uint64(v), 0)
	if carry != 0 {
		return 0, fmt.Errorf("%w: %d + %d", ErrOverflow, u, v)
	}
	return U64(s), nil
}

// Sub returns u-v or ErrUnderflow if v > u.
func (u U64) Sub(v U64) (U64, error) {
	d, borrow := bits.Sub64(uint64(u), uint64(v), 0)
	if borrow != 0 {
		return 0, fmt.Errorf("%w: %d - %d", ErrUnderflow, u, v)
	}
	return U64(d), nil
}

// Sum returns the sum of vs or ErrOverflow. The sum of no values is 0.
func Sum(vs ...U64) (U64, error) {
	var total U64
	for _, v := range vs {
		var err error
		if total, err = total.Add(v); err != nil {
			return 0, err
		}
	}
	return total, nil
}
