//go:build integration

package nodes_test

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/jackc/pgx/v5"
)

// TestReviewR15_StaleGraceCannotStayAtAnOldGeneration grants the one-shot
// grace, lets the node catch the desired generation, deletes the audit rows,
// and submits the old generation again. applied_config_generation only moves
// forward, so the old generation is no longer an in-flight ack. The grace
// column is cleared on the path that sets applying.
func TestReviewR15_StaleGraceCannotStayAtAnOldGeneration(t *testing.T) {
	w := newR14World(t)
	ctx := w.t.Context()
	if _, err := w.compiler.Compile(ctx, w.primary.NodeId); err != nil {
		t.Fatal(err)
	}
	if _, err := w.compiler.Compile(ctx, w.primary.NodeId); err != nil {
		t.Fatal(err)
	}
	stale, _ := w.prepare(w.primary, false, false)
	baseline, err := w.primary.AppliedConfigGeneration.Get()
	if err != nil {
		t.Fatal(err)
	}
	stale.AppliedConfigGeneration.Set(baseline)
	w.observe(stale)
	detail, _, err := w.r.Nodes.GetNode(ctx, w.primary.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ProvisioningState != api.ProvisioningStateReady {
		t.Fatalf("first stale report became %s", detail.ProvisioningState)
	}
	var graceSet bool
	if err = w.r.Pool.QueryRow(ctx, `SELECT stale_generation_grace_at IS NOT NULL FROM node_incarnations WHERE id=$1`, w.primary.NodeIncarnationId).Scan(&graceSet); err != nil {
		t.Fatal(err)
	}
	if !graceSet {
		t.Fatal("first stale report did not set stale_generation_grace_at")
	}

	// prepare compiles and stamps that generation. Do not compile again
	// before the observation: a further compile would make this report stale
	// and the one-shot grace would downgrade it to applying.
	caught, _ := w.prepare(w.primary, false, false)
	w.observe(caught)
	detail, _, err = w.r.Nodes.GetNode(ctx, w.primary.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ProvisioningState != api.ProvisioningStateReady {
		t.Fatalf("at-desired report became %s", detail.ProvisioningState)
	}
	var graceCleared bool
	if err = w.r.Pool.QueryRow(ctx, `SELECT stale_generation_grace_at IS NULL FROM node_incarnations WHERE id=$1`, w.primary.NodeIncarnationId).Scan(&graceCleared); err != nil {
		t.Fatal(err)
	}
	if !graceCleared {
		t.Fatal("catching the desired generation left stale_generation_grace_at set")
	}

	if err = db.InTx(ctx, w.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL fvpn.retention_purge = 'on'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM audit_events WHERE resource = $1`, "node:"+w.primary.NodeId.String())
		return err
	}); err != nil {
		t.Fatal(err)
	}

	again, _ := w.prepare(w.primary, false, false)
	again.AppliedConfigGeneration.Set(baseline)
	w.observe(again)
	detail, _, err = w.r.Nodes.GetNode(ctx, w.primary.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	var graceAfter bool
	if err = w.r.Pool.QueryRow(ctx, `SELECT stale_generation_grace_at IS NULL FROM node_incarnations WHERE id=$1`, w.primary.NodeIncarnationId).Scan(&graceAfter); err != nil {
		t.Fatal(err)
	}
	applied, _ := detail.ConfigGeneration.Applied.Get()
	if detail.ProvisioningState != api.ProvisioningStateApplying || !graceAfter || applied <= baseline {
		t.Fatalf("old generation after a catch-up stayed provisioning=%s grace_cleared=%v applied=%s baseline=%s desired=%s", detail.ProvisioningState, graceAfter, applied, baseline, detail.ConfigGeneration.Desired)
	}
}
