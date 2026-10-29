package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrVersionConflict     = errors.New("resource version conflict")
	ErrUniqueViolation     = errors.New("unique constraint violation")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrLeaseLost           = errors.New("outbox lease lost")
)

// ConstraintError exposes the constraint without leaking PostgreSQL detail
// (which can contain input values). errors.As still permits internal diagnosis.
type ConstraintError struct {
	Constraint string
	cause      *pgconn.PgError
}

func (e *ConstraintError) Error() string        { return ErrUniqueViolation.Error() + ": " + e.Constraint }
func (e *ConstraintError) Unwrap() error        { return e.cause }
func (e *ConstraintError) Is(target error) bool { return target == ErrUniqueViolation }

// MapError maps storage errors to domain-neutral sentinels. Do not return raw
// database errors through an HTTP handler; the API layer supplies its envelope.
func MapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var existing *ConstraintError
	if errors.As(err, &existing) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &ConstraintError{Constraint: pgErr.ConstraintName, cause: pgErr}
	}
	return err
}
