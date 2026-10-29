// Package presence preserves required JSON member presence in generated API
// models (§14.1). An unset field cannot be marshaled into a valid-looking value.
// Raw requests must still pass schema validation before decoding into models.
package presence

import (
	"bytes"
	"encoding/json"
	"errors"
)

var (
	ErrUnspecified = errors.New("required JSON member is unspecified")
	ErrNull        = errors.New("JSON member is null")
)

// Nullable represents an unspecified member, explicit null, or a value. Its
// zero value is unspecified and fails marshaling; null must be explicit.
// Required members must not use pointers or omission tags. Optional nullable
// members use omitzero to omit the unspecified zero value while preserving null
// and explicit values; omitempty alone cannot omit this struct.
type Nullable[T any] struct {
	value     T
	specified bool
	null      bool
}

func NewNullableWithValue[T any](v T) Nullable[T] { return Nullable[T]{value: v, specified: true} }
func NewNullNullable[T any]() Nullable[T]         { return Nullable[T]{specified: true, null: true} }

// FromPointer converts an existing nullable storage value into an explicitly
// present JSON member. A nil storage pointer means null, never unspecified.
func FromPointer[T any](v *T) Nullable[T] {
	if v == nil {
		return NewNullNullable[T]()
	}
	return NewNullableWithValue(*v)
}

func (n Nullable[T]) IsSpecified() bool { return n.specified }
func (n Nullable[T]) IsNull() bool      { return n.specified && n.null }
func (n Nullable[T]) Get() (T, error) {
	if !n.specified {
		return n.value, ErrUnspecified
	}
	if n.null {
		return n.value, ErrNull
	}
	return n.value, nil
}
func (n *Nullable[T]) Set(v T)         { *n = NewNullableWithValue(v) }
func (n *Nullable[T]) SetNull()        { *n = NewNullNullable[T]() }
func (n *Nullable[T]) SetUnspecified() { *n = Nullable[T]{} }
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.specified {
		return nil, ErrUnspecified
	}
	if n.null {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}
func (n *Nullable[T]) UnmarshalJSON(raw []byte) error {
	n.SetUnspecified()
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		n.SetNull()
		return nil
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	n.Set(v)
	return nil
}

// Required preserves presence for a non-nullable member, including false.
// Use NewRequired(false) to explicitly send a required false boolean.
type Required[T any] struct {
	value     T
	specified bool
}

func NewRequired[T any](v T) Required[T] { return Required[T]{value: v, specified: true} }
func (r Required[T]) IsSpecified() bool  { return r.specified }
func (r Required[T]) Get() (T, error) {
	if !r.specified {
		return r.value, ErrUnspecified
	}
	return r.value, nil
}
func (r *Required[T]) Set(v T) { *r = NewRequired(v) }
func (r Required[T]) MarshalJSON() ([]byte, error) {
	if !r.specified {
		return nil, ErrUnspecified
	}
	return json.Marshal(r.value)
}
func (r *Required[T]) UnmarshalJSON(raw []byte) error {
	*r = Required[T]{}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ErrNull
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	r.Set(v)
	return nil
}
