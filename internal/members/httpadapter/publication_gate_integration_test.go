//go:build integration

package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestD074_AdminPublicationGateCauseHTTP(t *testing.T) {
	f := newFixture(t)
	admin := f.login("TEST-publication-admin", "admin")
	opID := uuid.New()
	if err := db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		_, err := nodes.CreateOperation(t.Context(), tx, nodes.CreateOperationParams{
			ID: opID, Kind: nodes.OpKindPublish, Principal: "admin:" + admin.id.String(),
			ResourceID: uuid.New(), Now: f.now,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// The persisted pending outcome from exits.pendingPublishStepOutcome contains
	// both diagnostics. Read it through the production admin route and validator.
	steps := nodes.NewNodeOperationSteps(nodes.OpKindPublish, f.now)
	step := steps[nodes.StepDataPathProbe]
	code, message := "NODE_NOT_SERVICEABLE", "member_forwarding_unconfirmed"
	step.ErrorCode, step.ErrorMessage, step.Reason = &code, &message, "lease_unconfirmed"
	steps[nodes.StepDataPathProbe] = step
	raw, err := json.Marshal(steps)
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE operations SET status='running',steps=$2::jsonb,resource_version=resource_version+1 WHERE id=$1`, opID, raw)
	w := f.call(admin, http.MethodGet, "/api/v1/admin/operations/"+opID.String(), "", nil)
	requireStatus(t, w, http.StatusOK)
	op := decode[api.Operation](t, w)
	if op.Status != api.OperationStatusRunning || !op.Error.IsNull() || !op.CompletedAt.IsNull() {
		t.Fatalf("held publication became terminal: %+v", op)
	}
	for _, got := range op.Steps {
		if got.Name != nodes.StepDataPathProbe {
			continue
		}
		gotMessage, err := got.Message.Get()
		if err != nil || gotMessage != "member_forwarding_unconfirmed (lease_unconfirmed)" {
			t.Fatalf("step message = %q (%v), want both stored error message and gate cause", gotMessage, err)
		}
		gotCode, err := got.ErrorCode.Get()
		if err != nil || gotCode != api.ErrorCodeNODENOTSERVICEABLE || got.Status != api.OperationStepStatusPending || !got.ObservedAt.IsNull() {
			t.Fatalf("held step changed state or evidence: %+v", got)
		}
		return
	}
	t.Fatal("missing data_path_probe step")
}
