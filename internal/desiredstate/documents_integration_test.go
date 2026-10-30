//go:build integration

package desiredstate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestD056_D058_DocumentIssuanceObserverSharesCompilerTransaction(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	now := time.Now().UTC()
	group := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups(id,code,name) VALUES($1,'test-documents','TEST ONLY documents')`, group); err != nil {
		t.Fatal(err)
	}
	node, _, _, _ := seedCompilerTestNode(t, ctx, pool, group, api.PublicationIntentUnpublished, nil)
	signer, _ := testDesiredStateSignerAndTrustStore(t, now)
	rejected := errors.New("TEST ONLY issuance persistence failure")
	resolved, observed := false, false
	compiler := NewCompiler(pool, signer, WithEntitlementReader(NewFakeEntitlementReader()), WithPolicyReader(NewFakePolicyReader(PolicySnapshot{})), WithDocumentInputs(
		func(_ context.Context, _ pgx.Tx, state *api.DesiredState, _ []uuid.UUID) error {
			resolved = true
			state.EgressProbe.Set(api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "192.0.2.80", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}})
			return nil
		},
		func(ctx context.Context, tx pgx.Tx, state api.DesiredState) error {
			var persisted bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM desired_state_envelopes WHERE node_id=$1 AND generation=$2::numeric AND envelope->'signed'->'egress_probe'->'targets'->0->>'address'='192.0.2.80')`, node, state.ConfigGeneration.String()).Scan(&persisted); err != nil {
				return err
			}
			if !persisted {
				t.Fatal("issuance callback preceded signed persistence")
			}
			observed = true
			return rejected
		},
	))
	if _, err := compiler.Compile(ctx, node); !errors.Is(err, rejected) || !resolved || !observed {
		t.Fatal(err, resolved, observed)
	}
	var clean bool
	if err := pool.QueryRow(ctx, `SELECT desired_config_generation=0 AND NOT EXISTS(SELECT 1 FROM desired_states WHERE node_id=$1) AND NOT EXISTS(SELECT 1 FROM desired_state_envelopes WHERE node_id=$1) FROM nodes WHERE id=$1`, node).Scan(&clean); err != nil || !clean {
		t.Fatal("failed issuance escaped its transaction", clean, err)
	}
}
