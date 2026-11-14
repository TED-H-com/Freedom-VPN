//go:build integration

package exits

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (f *reconcileFixture) bindingTextFor(exitID uuid.UUID, deployment string) string {
	f.t.Helper()
	var text string
	err := f.pool.QueryRow(f.ctx, `
		SELECT b::text
		FROM external_bindings b
		WHERE resource_type = 'exit' AND resource_id = $1 AND deployment_id = $2`,
		exitID, deployment).Scan(&text)
	if err != nil {
		f.t.Fatal(err)
	}
	return text
}

func (f *reconcileFixture) seedCheckedBinding(exitID uuid.UUID, deployment, externalID string) {
	f.t.Helper()
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	f.exec(`
		INSERT INTO external_bindings(
			resource_type, resource_id, provider, deployment_id,
			external_id, desired_hash, observed_hash, sync_state,
			last_observed_at, last_verified_at, last_error)
		VALUES ('exit',$1,'netbird',$2,$3,$4,$5,'pending',$6,$6,$7)`,
		exitID, deployment, externalID,
		bytes.Repeat([]byte{0x11}, 32),
		bytes.Repeat([]byte{0x21}, 32),
		at, "seed-last-error")
}

func (f *reconcileFixture) seedPublishedDrift(exitID uuid.UUID, deployment string) (uuid.UUID, string) {
	f.t.Helper()
	nodeID := f.seedOrdered(exitID, deployment, string(api.PublicationIntentPublished))
	f.setDrift(nodeID, []api.StateDrift{{
		Field:    "capacity_report",
		Desired:  presence.NewNullableWithValue("stay"),
		Observed: presence.NewNullableWithValue("stay"),
	}})
	version, _ := f.nodeState(nodeID)
	state := withdrawRoute(exitID)
	state.ObservedHash = bytes.Repeat([]byte{0x44}, 32)
	f.nb.InjectRouteState(exitID, state)
	return nodeID, version
}

func assertReconcileSkipped(t *testing.T, f *reconcileFixture, nodeID uuid.UUID, version string, items []ReconcileItem) {
	t.Helper()
	if len(items) != 0 {
		t.Fatalf("results=%+v", items)
	}
	after, drift := f.nodeState(nodeID)
	versionPlus(t, version, after, 0)
	if len(routeElements(drift)) != 0 || len(drift) != 1 || drift[0].Field != "capacity_report" {
		t.Fatalf("drift=%+v", drift)
	}
}

func bindingRowText(ctx context.Context, tx pgx.Tx, exitID uuid.UUID, deployment string) (string, error) {
	var text string
	err := tx.QueryRow(ctx, `
		SELECT b::text
		FROM external_bindings b
		WHERE resource_type = 'exit' AND resource_id = $1 AND deployment_id = $2`,
		exitID, deployment).Scan(&text)
	return text, err
}

func TestReconcileNetBird_SkipsDeploymentChangedDuringNodeLock(t *testing.T) {
	f := newReconcileFixture(t)
	exitID := uuid.MustParse("00000000-0000-4000-8000-0000000000d1")
	const fromDeployment = "nb-main"
	const toDeployment = "nb-other"
	nodeID, version := f.seedPublishedDrift(exitID, fromDeployment)
	const externalID = "nb-twin"
	f.seedCheckedBinding(exitID, fromDeployment, externalID)
	f.seedCheckedBinding(exitID, toDeployment, externalID)
	before := map[string]string{
		fromDeployment: f.bindingTextFor(exitID, fromDeployment),
		toDeployment:   f.bindingTextFor(exitID, toDeployment),
	}
	ctx, cancel := context.WithTimeout(f.ctx, 45*time.Second)
	defer cancel()
	items, done, holder := f.reconcileBlockedByHook(ctx, func(tx pgx.Tx) error {
		var locked bool
		if err := tx.QueryRow(ctx, `SELECT true FROM nodes n WHERE n.id = $1 FOR UPDATE OF n`, nodeID).Scan(&locked); err != nil {
			return err
		}
		// The peer binding cannot be reassigned once set. This exit is already
		// in nb-main, so triggers are suspended for this transaction and the
		// worker's deployment update can move it to nb-other.
		if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
			return err
		}
		var incID uuid.UUID
		var peerID string
		if err := tx.QueryRow(ctx, `SELECT i.id, i.peer_id FROM node_incarnations i JOIN nodes n ON n.current_incarnation_id = i.id WHERE n.id = $1`, nodeID).Scan(&incID, &peerID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE node_incarnations SET deployment_id = $2, peer_id = $3 WHERE id = $1 AND revoked_at IS NULL`, incID, toDeployment, peerID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("deployment update matched no row")
		}
		return nil
	})
	defer holder.Rollback(context.Background())
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	assertReconcileSkipped(t, f, nodeID, version, *items)
	for _, deployment := range []string{fromDeployment, toDeployment} {
		if got := f.bindingTextFor(exitID, deployment); got != before[deployment] {
			t.Fatalf("binding %s changed\nbefore %s\nafter  %s", deployment, before[deployment], got)
		}
	}
	var deployment string
	if err := f.pool.QueryRow(ctx, `SELECT i.deployment_id FROM node_incarnations i JOIN nodes n ON n.current_incarnation_id = i.id WHERE n.id = $1`, nodeID).Scan(&deployment); err != nil || deployment != toDeployment {
		t.Fatalf("deployment %q: %v", deployment, err)
	}
}

func TestReconcileNetBird_SkipsBindingCreatedDuringNodeLock(t *testing.T) {
	f := newReconcileFixture(t)
	exitID := uuid.MustParse("00000000-0000-4000-8000-0000000000d2")
	const deployment = "nb-main"
	nodeID, version := f.seedPublishedDrift(exitID, deployment)
	var count int
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM external_bindings WHERE resource_type = 'exit' AND resource_id = $1`, exitID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("bindings = %d: %v", count, err)
	}
	insertedAt := time.Date(2026, 11, 14, 12, 30, 0, 0, time.UTC)
	ctx, cancel := context.WithTimeout(f.ctx, 45*time.Second)
	defer cancel()
	var left string
	items, done, holder := f.reconcileBlockedByHook(ctx, func(tx pgx.Tx) error {
		var locked bool
		if err := tx.QueryRow(ctx, `SELECT true FROM nodes n WHERE n.id = $1 FOR UPDATE OF n`, nodeID).Scan(&locked); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO external_bindings (
				resource_type, resource_id, provider, deployment_id,
				external_id, desired_hash, observed_hash, sync_state,
				last_observed_at, last_verified_at, last_error
			) VALUES (
				'exit', $1, $2, $3,
				$4, $5, $6, 'synced',
				$7, $7, NULL
			)`,
			exitID, ProviderNetBird, deployment,
			"nb-created", bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x32}, 32), insertedAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("binding insert matched no row")
		}
		left, err = bindingRowText(ctx, tx, exitID, deployment)
		return err
	})
	defer holder.Rollback(context.Background())
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	assertReconcileSkipped(t, f, nodeID, version, *items)
	if got := f.bindingText(exitID); got != left {
		t.Fatalf("binding changed\nleft %s\nafter %s", left, got)
	}
}
