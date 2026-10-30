package httpwire

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ReconcileResolverDocuments delivers newly eligible DNS documents to current
// incarnations without waiting for an unrelated publication command (D-056).
func (r *Runtime) ReconcileResolverDocuments(ctx context.Context) error {
	profile, err := activeResolverProfile(ctx, r.Pool)
	if err != nil || profile == uuid.Nil {
		return err
	}
	if r.ResolverDocuments == nil {
		return errors.New("resolver document source required")
	}
	document, err := r.ResolverDocuments.LatestPublished(ctx, profile)
	if errors.Is(err, pgx.ErrNoRows) {
		// No eligible publication: never invent a reference or clear last-good.
		return nil
	}
	if err != nil {
		return err
	}
	if err = nodes.ValidateResolverDocument(document); err != nil {
		return err
	}
	ref, err := canonjson.Marshal(document.Reference)
	if err != nil {
		return err
	}
	// One fleet profile applies to all nodes, including unpublished baselines.
	// Page only existing envelopes whose complete reference differs. Initial
	// enrollment still obtains its first state through the normal compiler path.
	var cursor uuid.UUID
	var firstError error
	for {
		rows, err := r.Pool.Query(ctx, `SELECT n.id FROM nodes n
		 JOIN desired_state_envelopes e ON e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id
		 WHERE n.lifecycle_state='registered' AND n.id>$1
		 AND e.envelope->'signed'->'dns'->'resolver_document' IS DISTINCT FROM $2::jsonb
		 ORDER BY n.id LIMIT 100`, cursor, ref)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return firstError
		}
		for _, node := range ids {
			if err := r.reconcileResolverDocument(ctx, node, document.Reference); err != nil {
				if ctx.Err() != nil || errors.Is(err, db.ErrLeaseLost) {
					return err
				}
				// One invalid node must not starve delivery to the remaining fleet.
				// Retain failure so the dispatcher retries/dead-letters truthfully.
				if firstError == nil {
					firstError = err
				}
			}
		}
		cursor = ids[len(ids)-1]
	}
}

func (r *Runtime) reconcileResolverDocument(ctx context.Context, node uuid.UUID, ref api.ResolverDocumentRef) error {
	return commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var incarnation uuid.UUID
		var raw []byte
		err := tx.QueryRow(ctx, `SELECT n.current_incarnation_id,e.envelope->'signed'->'dns'->'resolver_document'
		 FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id
		 WHERE n.id=$1 AND n.lifecycle_state='registered' FOR UPDATE OF n`, node).Scan(&incarnation, &raw)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var previous presence.Nullable[api.ResolverDocumentRef]
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &previous); err != nil {
				return err
			}
		}
		if value, err := previous.Get(); err == nil && value == ref {
			return nil
		}
		// Recheck under the fleet/node locks and the dispatcher's claim. This
		// path preserves update intent and records issuance in this transaction.
		_, err = r.CompileControlState(commandtx.WithTransaction(ctx, tx), node, incarnation)
		return err
	})
}
