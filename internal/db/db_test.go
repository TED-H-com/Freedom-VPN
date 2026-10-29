package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func Test15_1_ErrorMappingDoesNotExposeDetail(t *testing.T) {
	cause := &pgconn.PgError{Code: "23505", ConstraintName: "test_key", Detail: "sensitive input"}
	err := MapError(cause)
	if !errors.Is(err, ErrUniqueViolation) || strings.Contains(err.Error(), cause.Detail) {
		t.Fatal(err)
	}
	var e *ConstraintError
	if !errors.As(err, &e) || e.Constraint != "test_key" {
		t.Fatal(err)
	}
	if !errors.Is(MapError(pgx.ErrNoRows), ErrNotFound) {
		t.Fatal("missing not-found mapping")
	}
	other := errors.New("other")
	if MapError(other) != other || MapError(nil) != nil {
		t.Fatal("unexpected error conversion")
	}
	if MapError(err) != err {
		t.Fatal("error wrapped twice")
	}
}

func Test15_1_DecimalPrecisionAndJSONShape(t *testing.T) {
	for _, v := range []string{"0", "9007199254740993", "18446744073709551615"} {
		if !decimalUint64(v) {
			t.Error(v)
		}
	}
	for _, v := range []string{"", "-1", "01", "+1", "1.0", "18446744073709551616"} {
		if decimalUint64(v) {
			t.Error(v)
		}
	}
	for _, v := range []string{`{}`, `{"schema_version":1}`} {
		if !jsonObject([]byte(v)) {
			t.Error(v)
		}
	}
	for _, v := range []string{`null`, `[]`, `1`, `true`, `{} {}`, `{"x":`} {
		if jsonObject([]byte(v)) {
			t.Error(v)
		}
	}
	for _, v := range []string{"members", "resource_version"} {
		if !identifier(v) {
			t.Error(v)
		}
	}
	for _, v := range []string{"", "members;drop table members", "public.members", "1bad"} {
		if identifier(v) {
			t.Error(v)
		}
	}
}

func TestSEC02_OpenRedactsMalformedDSN(t *testing.T) {
	_, err := Open(context.Background(), "postgres://test:TEST_ONLY@[%invalid", OpenOptions{})
	if err == nil || strings.Contains(err.Error(), "TEST_ONLY") {
		t.Fatal("DSN was not redacted")
	}
}
