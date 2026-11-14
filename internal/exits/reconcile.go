package exits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	exitID       uuid.UUID
	nodeID       uuid.UUID
	intent       string
	deploymentID string
	externalID   string
	desiredHash  []byte
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
		if err := commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			for _, item := range writes {
				if err := mergeExitDrift(ctx, tx, item); err != nil {
					return err
				}
				if err := updateExitBinding(ctx, tx, item, now); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return results, err
		}
		for _, item := range writes {
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
			eb.desired_hash
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
		if err := rows.Scan(&item.exitID, &item.nodeID, &item.intent, &item.deploymentID, &item.externalID, &item.desiredHash); err != nil {
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

func mergeExitDrift(ctx context.Context, tx pgx.Tx, item reconcileWrite) error {
	raw, err := json.Marshal(item.drift)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE nodes AS n
		SET drift = calc.new_drift,
		    resource_version = n.resource_version + 1
		FROM (
			SELECT d.id,
			       COALESCE((
			           SELECT jsonb_agg(elem ORDER BY ord)
			           FROM jsonb_array_elements(d.drift) WITH ORDINALITY AS t(elem, ord)
			           WHERE COALESCE(elem->>'field', '') NOT IN (
			               'netbird_route', 'netbird_route_config', 'netbird_route_evidence')
			       ), '[]'::jsonb) || $2::jsonb AS new_drift
			FROM nodes d
			WHERE d.id = $1
		) AS calc
		WHERE n.id = calc.id
		  AND n.drift IS DISTINCT FROM calc.new_drift`, item.nodeID, raw)
	return err
}

func updateExitBinding(ctx context.Context, tx pgx.Tx, item reconcileWrite, now time.Time) error {
	if item.externalID == "" {
		return nil
	}
	if item.evidence {
		cause, err := item.drift[0].Observed.Get()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE external_bindings
			SET sync_state = 'error',
			    last_error = $4
			WHERE resource_type = 'exit'
			  AND resource_id = $1
			  AND provider = $2
			  AND deployment_id = $3`,
			item.exitID, ProviderNetBird, item.deploymentID, "evidence: "+cause)
		return err
	}
	syncState := "synced"
	var lastErr *string
	if len(item.drift) > 0 {
		syncState = "drifted"
		msg := fmt.Sprintf("drift on %s", item.drift[0].Field)
		lastErr = &msg
	}
	_, err := tx.Exec(ctx, `
		UPDATE external_bindings
		SET sync_state = $4,
		    observed_hash = coalesce($5, observed_hash),
		    last_verified_at = $6,
		    last_error = $7
		WHERE resource_type = 'exit'
		  AND resource_id = $1
		  AND provider = $2
		  AND deployment_id = $3`,
		item.exitID, ProviderNetBird, item.deploymentID, syncState, item.observed.ObservedHash, now, lastErr)
	return err
}
