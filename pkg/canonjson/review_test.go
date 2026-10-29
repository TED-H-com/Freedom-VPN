package canonjson

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestReviewC11_InputByteLimit(t *testing.T) {
	if MaxBytes != 1<<20 {
		t.Fatal("body limit drift")
	}
	atLimit := append([]byte{'"'}, bytes.Repeat([]byte{'a'}, MaxBytes-2)...)
	atLimit = append(atLimit, '"')
	if _, err := Canonicalize(atLimit); err != nil {
		t.Fatal(err)
	}
	overLimit := append(bytes.Clone(atLimit), ' ')
	for _, raw := range [][]byte{overLimit, append([]byte{'!'}, overLimit...)} {
		if _, err := Canonicalize(raw); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("Canonicalize: %v", err)
		}
		if _, err := CanonicalizeWithout(raw, "excluded"); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("CanonicalizeWithout: %v", err)
		}
		if _, err := HashHex(raw); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("HashHex: %v", err)
		}
	}
	if _, err := Marshal(strings.Repeat("a", MaxBytes)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Marshal: %v", err)
	}
}

func TestReviewC14_DuplicateKeyErrorDoesNotExposeName(t *testing.T) {
	_, err := Canonicalize([]byte(`{"secret_field":1,"secret_field":2}`))
	var e *Error
	if !errors.Is(err, ErrDuplicateKey) || !errors.As(err, &e) || e.Offset == 0 {
		t.Fatalf("missing class/offset: %v", err)
	}
	if strings.Contains(err.Error(), "secret_field") || e.Detail != "" {
		t.Fatalf("member leaked: %v", err)
	}
}
