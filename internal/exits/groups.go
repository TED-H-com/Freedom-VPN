package exits

import (
	"context"
	"errors"
	"slices"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The caller holds the node/exit locks and has checked If-Match. An explicit
// empty list clears membership; an omitted list never reaches this function (D-057).
func replacePublicationGroups(ctx context.Context, tx pgx.Tx, exitID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	groups := append([]uuid.UUID{}, ids...)
	slices.SortFunc(groups, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	if len(groups) > 100 || len(slices.Compact(slices.Clone(groups))) != len(groups) {
		return nil, apierr.Validation("Invalid publication groups.", apierr.FieldError{Field: "/group_ids", Reason: "at most 100 unique groups required"})
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM member_groups WHERE id=ANY($1::uuid[])`, groups).Scan(&count); err != nil {
		return nil, err
	}
	if count != len(groups) {
		return nil, apierr.Validation("Unknown publication group.", apierr.FieldError{Field: "/group_ids", Reason: "unknown group"})
	}
	if _, err := tx.Exec(ctx, `DELETE FROM exit_groups WHERE exit_id=$1`, exitID); err != nil {
		return nil, err
	}
	_, err := tx.Exec(ctx, `INSERT INTO exit_groups(exit_id,group_id) SELECT $1,unnest($2::uuid[])`, exitID, groups)
	// D-057 / §14.5: a group can disappear after validation. The failed
	// transaction must roll back the replacement and return the same field error
	// as an initially unknown group, without exposing PostgreSQL diagnostics.
	var constraint *pgconn.PgError
	if errors.As(err, &constraint) && constraint.Code == "23503" && constraint.ConstraintName == "exit_groups_group_id_fkey" {
		return nil, apierr.Validation("Unknown publication group.", apierr.FieldError{Field: "/group_ids", Reason: "unknown group"})
	}
	return groups, err
}
