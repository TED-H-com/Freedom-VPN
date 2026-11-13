package nodes

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RecordSubjectRemovalsTx runs for every CompileTx caller, under the compiler's
// node lock. One outstanding row per subject/node retains the highest removal;
// repeated compiles of an already-absent subject do not change its generation.
func RecordSubjectRemovalsTx(ctx context.Context, tx pgx.Tx, node, incarnation uuid.UUID, generation string, before, after []byte) error {
	if len(before) == 0 {
		before = []byte(`{}`)
	}
	_, err := tx.Exec(ctx, `WITH previous AS (
  SELECT 'device' AS subject_type,b->>'device_id' AS subject_id FROM jsonb_array_elements(coalesce($4::jsonb->'signed'->'device_bindings','[]')) b
  UNION SELECT 'member',b->>'member_id' FROM jsonb_array_elements(coalesce($4::jsonb->'signed'->'device_bindings','[]')) b
  UNION SELECT 'member',b->>'member_id' FROM jsonb_array_elements(coalesce($4::jsonb->'signed'->'member_rate_policies','[]')) b
 ), current AS (
  SELECT 'device' AS subject_type,b->>'device_id' AS subject_id FROM jsonb_array_elements(coalesce($5::jsonb->'signed'->'device_bindings','[]')) b
  UNION SELECT 'member',b->>'member_id' FROM jsonb_array_elements(coalesce($5::jsonb->'signed'->'device_bindings','[]')) b
  UNION SELECT 'member',b->>'member_id' FROM jsonb_array_elements(coalesce($5::jsonb->'signed'->'member_rate_policies','[]')) b
 ), removed AS (SELECT * FROM previous EXCEPT SELECT * FROM current)
 INSERT INTO node_subject_removals(node_id,subject_type,subject_id,incarnation_id,generation)
 SELECT $1,subject_type,subject_id::uuid,$2,$3::numeric FROM removed
 WHERE EXISTS(SELECT 1 FROM exits WHERE node_id=$1)
 ORDER BY subject_type,subject_id
 ON CONFLICT(node_id,subject_type,subject_id) DO UPDATE SET incarnation_id=excluded.incarnation_id,generation=excluded.generation,compiled_at=excluded.compiled_at
 WHERE node_subject_removals.generation<excluded.generation`, node, incarnation, generation, before, after)
	return err
}

// captureRemovalTargetsTx uses the same node -> removal -> target -> operation
// lock order as compilation and observation. Nodes are locked in UUID order.
// The separate removal read sees any acknowledgement that committed while the
// node lock was awaited; otherwise the observer waits and sees the new target.
func captureRemovalTargetsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `SELECT n.id FROM nodes n
      WHERE n.lifecycle_state='registered' AND n.current_incarnation_id IS NOT NULL
      AND EXISTS(SELECT 1 FROM node_subject_removals r JOIN node_operation_captures c
        ON c.subject_type=r.subject_type AND c.subject_id=r.subject_id
        WHERE r.node_id=n.id AND c.operation_id=ANY($1) AND c.captured_at IS NULL)
      ORDER BY n.id FOR UPDATE OF n`, ids); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT c.operation_id,r.node_id,r.incarnation_id,r.generation::text,r.compiled_at
 FROM node_subject_removals r JOIN node_operation_captures c ON c.subject_type=r.subject_type AND c.subject_id=r.subject_id
 JOIN nodes n ON n.id=r.node_id AND n.lifecycle_state='registered' AND n.current_incarnation_id IS NOT NULL
 WHERE c.operation_id=ANY($1) AND c.captured_at IS NULL
 ORDER BY r.node_id,r.subject_type,r.subject_id,c.operation_id FOR UPDATE OF r`, ids)
	if err != nil {
		return err
	}
	type removal struct {
		Op, Node, Incarnation uuid.UUID
		Generation            string
		Compiled              time.Time
	}
	removals, err := pgx.CollectRows(rows, pgx.RowToStructByPos[removal])
	if err != nil {
		return err
	}
	for _, r := range removals {
		_, err = tx.Exec(ctx, `INSERT INTO node_operation_targets(operation_id,node_id,incarnation_id,generation,compiled_at,subject_present)
 VALUES($1,$2,$3,$4::numeric,$5,false)
 ON CONFLICT(operation_id,node_id) DO UPDATE SET incarnation_id=excluded.incarnation_id,generation=excluded.generation,compiled_at=excluded.compiled_at,
 subject_present=false,outcome='pending',reason=NULL,error_code=NULL,error_message=NULL,observed_at=NULL,report_id=NULL,acknowledged_incarnation_id=NULL,observed_ref=NULL
 WHERE node_operation_targets.generation<excluded.generation`, r.Op, r.Node, r.Incarnation, r.Generation, r.Compiled)
		if err != nil {
			return err
		}
	}
	return nil
}

func acknowledgeSubjectRemovals(ctx context.Context, tx pgx.Tx, node uuid.UUID, generation string, now time.Time) error {
	// Lock all qualifying rows in the same order as capture and compilation. The
	// DELETE and following target read are separate statements (READ COMMITTED).
	_, err := tx.Exec(ctx, `WITH acknowledged AS MATERIALIZED (
 SELECT node_id,subject_type,subject_id FROM node_subject_removals
 WHERE node_id=$1 AND generation<=$2::numeric AND compiled_at<=$3
 ORDER BY node_id,subject_type,subject_id FOR UPDATE)
 DELETE FROM node_subject_removals r USING acknowledged a
 WHERE r.node_id=a.node_id AND r.subject_type=a.subject_type AND r.subject_id=a.subject_id`, node, generation, now)
	return err
}
