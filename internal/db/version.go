package db

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UpdateWithVersion performs §14.5 optimistic concurrency in the caller's
// transaction. table/column names must come from trusted application code.
// Values are bound parameters; the returned uint64 version remains decimal text.
func UpdateWithVersion(ctx context.Context, tx pgx.Tx, table string, id uuid.UUID, expected string, changes map[string]any) (string, error) {
	if !identifier(table) || !decimalUint64(expected) || len(changes) == 0 {
		return "", errors.New("invalid versioned update")
	}
	columns := make([]string, 0, len(changes))
	for column := range changes {
		if !identifier(column) || column == "id" || column == "resource_version" {
			return "", errors.New("invalid versioned column")
		}
		columns = append(columns, column)
	}
	slices.Sort(columns)
	args := []any{id, expected}
	sets := []string{"resource_version = resource_version + 1"}
	for _, column := range columns {
		args = append(args, changes[column])
		sets = append(sets, fmt.Sprintf("%s = $%d", pgx.Identifier{column}.Sanitize(), len(args)))
	}
	query := "UPDATE " + pgx.Identifier{"public", table}.Sanitize() + " SET " + strings.Join(sets, ", ") + " WHERE id=$1 AND resource_version=$2::numeric RETURNING resource_version::text"
	var version string
	err := tx.QueryRow(ctx, query, args...).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrVersionConflict
	}
	return version, MapError(err)
}

func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range []byte(s) {
		if !(c == '_' || c >= 'a' && c <= 'z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func decimalUint64(s string) bool {
	if len(s) == 0 || len(s) > 20 || len(s) > 1 && s[0] == '0' {
		return false
	}
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return false
		}
	}
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}
