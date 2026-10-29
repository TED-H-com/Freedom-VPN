package presence

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestReviewC04_NullableStateTransitions(t *testing.T) {
	var n Nullable[int]
	if _, err := n.Get(); !errors.Is(err, ErrUnspecified) {
		t.Fatal(err)
	}
	if _, err := json.Marshal(n); !errors.Is(err, ErrUnspecified) {
		t.Fatal(err)
	}
	n.SetNull()
	if !n.IsSpecified() || !n.IsNull() {
		t.Fatal("null state")
	}
	if _, err := n.Get(); !errors.Is(err, ErrNull) {
		t.Fatal(err)
	}
	n.Set(7)
	if value, err := n.Get(); err != nil || value != 7 {
		t.Fatalf("%d %v", value, err)
	}
	if raw, err := json.Marshal(n); err != nil || string(raw) != "7" {
		t.Fatalf("%s %v", raw, err)
	}
	if err := json.Unmarshal([]byte(`"wrong"`), &n); err == nil || n.IsSpecified() {
		t.Fatal("failed decode retained old value")
	}
	if err := json.Unmarshal([]byte("null"), &n); err != nil || !n.IsNull() {
		t.Fatal(err)
	}
	n.SetUnspecified()
	if n.IsSpecified() {
		t.Fatal("reset state")
	}
}

func TestReviewC06_RequiredBooleanStateTransitions(t *testing.T) {
	var r Required[bool]
	if _, err := r.Get(); !errors.Is(err, ErrUnspecified) {
		t.Fatal(err)
	}
	r.Set(false)
	if !r.IsSpecified() {
		t.Fatal("false lost")
	}
	if value, err := r.Get(); err != nil || value {
		t.Fatalf("%v %v", value, err)
	}
	if err := json.Unmarshal([]byte(`"false"`), &r); err == nil || r.IsSpecified() {
		t.Fatal("failed decode retained old value")
	}
	if err := json.Unmarshal([]byte("null"), &r); !errors.Is(err, ErrNull) {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte("true"), &r); err != nil {
		t.Fatal(err)
	}
	if raw, err := json.Marshal(r); err != nil || string(raw) != "true" {
		t.Fatalf("%s %v", raw, err)
	}
}
