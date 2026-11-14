package exits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// netBirdReconcilePageSize is the keyset page length for ReconcileNetBird.
const netBirdReconcilePageSize = 200

// reconcilePageSize is the page length used by ReconcileNetBird.
// Tests in this package may lower it.
var reconcilePageSize = netBirdReconcilePageSize

// Reconcile fields owned by NetBird route read-back. Other drift elements stay.
const (
	driftFieldRoute    = "netbird_route"
	driftFieldConfig   = "netbird_route_config"
	driftFieldEvidence = "netbird_route_evidence"
)

// ReconcileItem describes the reconciliation result for a single exit.
type ReconcileItem struct {
	NodeID uuid.UUID
	ExitID uuid.UUID
	Drift  []api.StateDrift
}

type reconcileExit struct {
	exitID         uuid.UUID
	nodeID         uuid.UUID
	intent         string
	deploymentID   string
	externalID     string
	desiredHash    []byte
	observedHash   []byte
	syncState      *string
	lastError      *string
	lastObservedAt *time.Time
	lastVerifiedAt *time.Time
}

type reconcileWrite struct {
	reconcileExit
	drift    []api.StateDrift
	evidence bool
	observed ObservedExitState
}

// ReconcileNetBird compares observed NetBird routes with each exit's
// publication_intent and records drift on nodes.drift and external_bindings (§17.4).
// It never alters exits.publication_intent (EXIT03). One NetBird snapshot is taken
// per deployment in a page. A per-exit evidence error is recorded and the run
// continues. An unserved deployment is skipped. Any other error stops the run
// with nothing of that page written.
func (s *Service) ReconcileNetBird(ctx context.Context) ([]ReconcileItem, error) {
	if s.netbird == nil {
		return nil, errors.New("exits: netbird adapter is required for reconciliation")
	}
	pageSize := reconcilePageSize
	if pageSize < 1 {
		pageSize = netBirdReconcilePageSize
	}
	now := s.now().UTC()
	var results []ReconcileItem
	var cursor uuid.UUID
	for {
		page, err := s.queryExitPage(ctx, cursor, pageSize)
		if err != nil {
			return results, err
		}
		if len(page) == 0 {
			return results, nil
		}
		cursor = page[len(page)-1].exitID
		writes, err := s.readExitPage(ctx, page)
		if err != nil {
			return results, err
		}
		if len(writes) == 0 {
			if len(page) < pageSize {
				return results, nil
			}
			continue
		}
		var written []reconcileWrite
		if err := commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			kept, err := lockAndFilterPage(ctx, tx, writes)
			if err != nil {
				return err
			}
			pageWritten := make([]reconcileWrite, 0, len(kept))
			for _, item := range kept {
				updated, err := updateExitBinding(ctx, tx, item, now)
				if err != nil {
					return err
				}
				// The binding update runs before drift. A publisher can commit
				// a new binding while this update waits; the guarded UPDATE
				// then matches no row. Leave that item for the next run.
				if item.externalID != "" && !updated {
					continue
				}
				if err := mergeExitDrift(ctx, tx, item); err != nil {
					return err
				}
				pageWritten = append(pageWritten, item)
			}
			written = pageWritten
			return nil
		}); err != nil {
			return results, err
		}
		for _, item := range written {
			results = append(results, ReconcileItem{NodeID: item.nodeID, ExitID: item.exitID, Drift: item.drift})
		}
		if len(page) < pageSize {
			return results, nil
		}
	}
}

func (s *Service) queryExitPage(ctx context.Context, cursor uuid.UUID, limit int) ([]reconcileExit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			e.id,
			e.node_id,
			e.publication_intent,
			coalesce(inc.deployment_id, $1),
			coalesce(eb.external_id, ''),
			eb.desired_hash,
			eb.observed_hash,
			eb.sync_state,
			eb.last_error,
			eb.last_observed_at,
			eb.last_verified_at
		FROM exits e
		JOIN nodes n ON n.id = e.node_id
		LEFT JOIN node_incarnations inc ON inc.id = n.current_incarnation_id
		LEFT JOIN external_bindings eb
		  ON eb.resource_type = 'exit'
		 AND eb.resource_id = e.id
		 AND eb.provider = $2
		 AND eb.deployment_id = coalesce(inc.deployment_id, $1)
		WHERE e.id > $3
		ORDER BY e.id
		LIMIT $4`, DefaultDeploymentID, ProviderNetBird, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []reconcileExit
	for rows.Next() {
		var item reconcileExit
		if err := rows.Scan(
			&item.exitID, &item.nodeID, &item.intent, &item.deploymentID, &item.externalID, &item.desiredHash,
			&item.observedHash, &item.syncState, &item.lastError, &item.lastObservedAt, &item.lastVerifiedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) readExitPage(ctx context.Context, page []reconcileExit) ([]reconcileWrite, error) {
	groups := groupExitsByDeployment(page)
	var writes []reconcileWrite
	for _, group := range groups {
		refs := make([]ExitPageRef, len(group.items))
		for i, item := range group.items {
			refs[i] = ExitPageRef{ExitID: item.exitID, ExternalID: item.externalID}
		}
		read, err := s.netbird.ReadExitPage(ctx, group.deploymentID, refs)
		if errors.Is(err, ErrDeploymentNotServed) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(read) != len(group.items) {
			return nil, errors.New("exits: netbird page result length mismatch")
		}
		for i, item := range group.items {
			write, err := reconcileOne(item, read[i])
			if err != nil {
				return nil, err
			}
			writes = append(writes, write)
		}
	}
	return writes, nil
}

type deploymentExits struct {
	deploymentID string
	items        []reconcileExit
}

func groupExitsByDeployment(page []reconcileExit) []deploymentExits {
	index := map[string]int{}
	var groups []deploymentExits
	for _, item := range page {
		i, ok := index[item.deploymentID]
		if !ok {
			index[item.deploymentID] = len(groups)
			groups = append(groups, deploymentExits{deploymentID: item.deploymentID})
			i = len(groups) - 1
		}
		groups[i].items = append(groups[i].items, item)
	}
	return groups
}

func reconcileOne(item reconcileExit, read ExitPageResult) (reconcileWrite, error) {
	write := reconcileWrite{reconcileExit: item}
	if read.Evidence != nil {
		var classified PublicationEvidenceError
		if !errors.As(read.Evidence, &classified) {
			return reconcileWrite{}, read.Evidence
		}
		write.evidence = true
		write.drift = []api.StateDrift{{
			Field:    driftFieldEvidence,
			Desired:  presence.NewNullableWithValue(item.intent),
			Observed: presence.NewNullableWithValue(PublicationFailureCause(classified, "readback")),
		}}
		return write, nil
	}
	write.observed = read.State
	write.drift = routeDrift(item, read.State)
	return write, nil
}

func routeDrift(item reconcileExit, obs ObservedExitState) []api.StateDrift {
	wantPublished := item.intent == string(api.PublicationIntentPublished)
	if wantPublished {
		if !obs.Exists || !obs.Enabled {
			return []api.StateDrift{{
				Field:    driftFieldRoute,
				Desired:  presence.NewNullableWithValue("published"),
				Observed: presence.NewNullableWithValue("withdrawn"),
			}}
		}
		if len(item.desiredHash) > 0 && len(obs.ObservedHash) > 0 && !bytes.Equal(item.desiredHash, obs.ObservedHash) {
			return []api.StateDrift{{
				Field:    driftFieldConfig,
				Desired:  presence.NewNullableWithValue(fmt.Sprintf("%x", item.desiredHash[:8])),
				Observed: presence.NewNullableWithValue(fmt.Sprintf("%x", obs.ObservedHash[:8])),
			}}
		}
		return []api.StateDrift{}
	}
	if obs.Exists && obs.Enabled {
		return []api.StateDrift{{
			Field:    driftFieldRoute,
			Desired:  presence.NewNullableWithValue(item.intent),
			Observed: presence.NewNullableWithValue("published"),
		}}
	}
	return []api.StateDrift{}
}

// samePageRead reports whether publication intent, the deployment and every
// binding value still equal the page read. Timestamps match as instants.
// A NULL binding value equals only NULL.
func samePageRead(page, current reconcileExit) bool {
	if page.intent != current.intent || page.deploymentID != current.deploymentID || page.externalID != current.externalID {
		return false
	}
	if (page.desiredHash == nil) != (current.desiredHash == nil) || !bytes.Equal(page.desiredHash, current.desiredHash) {
		return false
	}
	if (page.observedHash == nil) != (current.observedHash == nil) || !bytes.Equal(page.observedHash, current.observedHash) {
		return false
	}
	if (page.syncState == nil) != (current.syncState == nil) || (page.syncState != nil && *page.syncState != *current.syncState) {
		return false
	}
	if (page.lastError == nil) != (current.lastError == nil) || (page.lastError != nil && *page.lastError != *current.lastError) {
		return false
	}
	if (page.lastObservedAt == nil) != (current.lastObservedAt == nil) || (page.lastObservedAt != nil && !page.lastObservedAt.Equal(*current.lastObservedAt)) {
		return false
	}
	if (page.lastVerifiedAt == nil) != (current.lastVerifiedAt == nil) || (page.lastVerifiedAt != nil && !page.lastVerifiedAt.Equal(*current.lastVerifiedAt)) {
		return false
	}
	return true
}

// lockAndFilterPage locks every node of the page in id order, then reads
// publication intent, the deployment and every binding value.
// Items whose values still match the page read are returned in node id
// order. A changed or missing node is omitted. Neither statement locks
// exits or external_bindings.
//
// The lock is one statement with no join:
//
//	SELECT n.id FROM nodes n WHERE n.id = ANY($1::uuid[]) ORDER BY n.id FOR UPDATE
//
// A locking statement does not refresh joined rows after a wait. Under
// READ COMMITTED only the locked row is current; joined rows keep the
// snapshot taken before the wait. The read is a separate statement, run
// only after every locked id has been read, so its snapshot is taken
// after every node lock is held. The node locks keep publication_intent
// and the deployment unchanged until this transaction commits, because
// their writers lock the node row first; binding values are written
// without a node lock, so updateExitBinding repeats every binding value
// in its WHERE.
func lockAndFilterPage(ctx context.Context, tx pgx.Tx, writes []reconcileWrite) ([]reconcileWrite, error) {
	if len(writes) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(writes))
	for i, item := range writes {
		ids[i] = item.nodeID
	}
	lockRows, err := tx.Query(ctx, `
		SELECT n.id
		FROM nodes n
		WHERE n.id = ANY($1::uuid[])
		ORDER BY n.id
		FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	for lockRows.Next() {
		var id uuid.UUID
		if err := lockRows.Scan(&id); err != nil {
			lockRows.Close()
			return nil, err
		}
	}
	lockErr := lockRows.Err()
	lockRows.Close()
	if lockErr != nil {
		return nil, lockErr
	}
	rows, err := tx.Query(ctx, `
		SELECT
			n.id,
			e.publication_intent,
			coalesce(inc.deployment_id, $1),
			coalesce(eb.external_id, ''),
			eb.desired_hash,
			eb.observed_hash,
			eb.sync_state,
			eb.last_error,
			eb.last_observed_at,
			eb.last_verified_at
		FROM exits e
		JOIN nodes n ON n.id = e.node_id
		LEFT JOIN node_incarnations inc ON inc.id = n.current_incarnation_id
		LEFT JOIN external_bindings eb
		  ON eb.resource_type = 'exit'
		 AND eb.resource_id = e.id
		 AND eb.provider = $2
		 AND eb.deployment_id = coalesce(inc.deployment_id, $1)
		WHERE n.id = ANY($3::uuid[])
		ORDER BY n.id`, DefaultDeploymentID, ProviderNetBird, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	current := make(map[uuid.UUID]reconcileExit, len(writes))
	for rows.Next() {
		var item reconcileExit
		if err := rows.Scan(
			&item.nodeID, &item.intent, &item.deploymentID, &item.externalID, &item.desiredHash,
			&item.observedHash, &item.syncState, &item.lastError, &item.lastObservedAt, &item.lastVerifiedAt,
		); err != nil {
			return nil, err
		}
		current[item.nodeID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	kept := make([]reconcileWrite, 0, len(writes))
	for _, item := range writes {
		got, ok := current[item.nodeID]
		if !ok || !samePageRead(item.reconcileExit, got) {
			continue
		}
		kept = append(kept, item)
	}
	sort.Slice(kept, func(i, j int) bool {
		return bytes.Compare(kept[i].nodeID[:], kept[j].nodeID[:]) < 0
	})
	return kept, nil
}

// mergedDriftSQL is the new drift array: foreign elements of this row, in order,
// then the route elements. It is one expression so the guard and the write
// cannot diverge. It reads n.drift on the row being updated. A second scan of
// nodes would keep the snapshot from before this statement waited for the row lock.
const mergedDriftSQL = `COALESCE((
			SELECT jsonb_agg(elem ORDER BY ord)
			FROM jsonb_array_elements(n.drift) WITH ORDINALITY AS t(elem, ord)
			WHERE COALESCE(elem->>'field', '') NOT IN (
				'netbird_route', 'netbird_route_config', 'netbird_route_evidence')
		), '[]'::jsonb) || $2::jsonb`

func mergeExitDrift(ctx context.Context, tx pgx.Tx, item reconcileWrite) error {
	raw, err := json.Marshal(item.drift)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE nodes AS n
		SET drift = `+mergedDriftSQL+`,
		    resource_version = n.resource_version + 1
		WHERE n.id = $1
		  AND n.drift IS DISTINCT FROM `+mergedDriftSQL, item.nodeID, raw)
	return err
}

// updateExitBinding writes the binding this item was checked with. It reports
// whether a row was updated. An exit with no binding is unchanged and reports
// false. Both statements require every binding value the page read saw,
// not only the external id and desired hash.
// desired_hash uses IS NOT DISTINCT FROM because it can be NULL. An UPDATE
// that waits for the binding row rechecks WHERE on the newest version, so a
// binding the publisher changed is not overwritten.
func updateExitBinding(ctx context.Context, tx pgx.Tx, item reconcileWrite, now time.Time) (bool, error) {
	if item.externalID == "" {
		return false, nil
	}
	if item.evidence {
		cause, err := item.drift[0].Observed.Get()
		if err != nil {
			return false, err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE external_bindings
			SET sync_state = 'error',
			    last_error = $4
			WHERE resource_type = 'exit'
			  AND resource_id = $1
			  AND provider = $2
			  AND deployment_id = $3
			  AND external_id = $5
			  AND desired_hash IS NOT DISTINCT FROM $6
			  AND observed_hash IS NOT DISTINCT FROM $7
			  AND sync_state IS NOT DISTINCT FROM $8
			  AND last_error IS NOT DISTINCT FROM $9
			  AND last_observed_at IS NOT DISTINCT FROM $10
			  AND last_verified_at IS NOT DISTINCT FROM $11`,
			item.exitID, ProviderNetBird, item.deploymentID, "evidence: "+cause, item.externalID, item.desiredHash,
			item.observedHash, item.syncState, item.lastError, item.lastObservedAt, item.lastVerifiedAt)
		if err != nil {
			return false, err
		}
		return tag.RowsAffected() == 1, nil
	}
	syncState := "synced"
	var lastErr *string
	if len(item.drift) > 0 {
		syncState = "drifted"
		msg := fmt.Sprintf("drift on %s", item.drift[0].Field)
		lastErr = &msg
	}
	tag, err := tx.Exec(ctx, `
		UPDATE external_bindings
		SET sync_state = $4,
		    observed_hash = coalesce($5, observed_hash),
		    last_verified_at = $6,
		    last_error = $7
		WHERE resource_type = 'exit'
		  AND resource_id = $1
		  AND provider = $2
		  AND deployment_id = $3
		  AND external_id = $8
		  AND desired_hash IS NOT DISTINCT FROM $9
		  AND observed_hash IS NOT DISTINCT FROM $10
		  AND sync_state IS NOT DISTINCT FROM $11
		  AND last_error IS NOT DISTINCT FROM $12
		  AND last_observed_at IS NOT DISTINCT FROM $13
		  AND last_verified_at IS NOT DISTINCT FROM $14`,
		item.exitID, ProviderNetBird, item.deploymentID, syncState, item.observed.ObservedHash, now, lastErr, item.externalID, item.desiredHash,
		item.observedHash, item.syncState, item.lastError, item.lastObservedAt, item.lastVerifiedAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
