//go:build integration

package exits

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type intentGuardFixture struct {
	t                 *testing.T
	pool              *pgxpool.Pool
	nodes             *nodes.Service
	exits             *Service
	adapter           *FakeNetBirdAdapter
	node, incarnation uuid.UUID
	exit, release     uuid.UUID
	now               time.Time
}

func newIntentGuardFixture(t *testing.T, initial api.PublicationIntent) *intentGuardFixture {
	t.Helper()
	f := &intentGuardFixture{t: t, pool: dbtest.New(t), adapter: NewFakeNetBirdAdapter(), now: time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)}
	clock := func() time.Time { return f.now }
	f.nodes = nodes.NewService(f.pool, nodes.WithClock(clock))
	f.exits = NewService(f.pool, testExitsCompiler(t, f.pool, clock), f.adapter, WithClock(clock))
	f.node, f.incarnation, f.exit, f.release, _ = seedExitTestNode(t, t.Context(), f.pool, initial, "absent")
	return f
}

func (f *intentGuardFixture) set(intent api.PublicationIntent) api.Operation {
	f.t.Helper()
	_, etag, err := f.nodes.GetNode(f.t.Context(), f.node)
	if err != nil {
		f.t.Fatal(err)
	}
	op, _, err := f.exits.SetPublication(f.t.Context(), SetPublicationParams{
		NodeID: f.node, Principal: "admin@example.com", IdempotencyKey: uuid.New(), IfMatch: etag,
		Request: api.PublicationRequest{PublicationIntent: intent, Reason: "TEST ONLY intent ordering"},
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return op
}

func (f *intentGuardFixture) acknowledge() {
	f.t.Helper()
	detail, _, err := f.nodes.GetNode(f.t.Context(), f.node)
	if err != nil {
		f.t.Fatal(err)
	}
	f.now = f.now.Add(5 * time.Second)
	submitAckObservation(f.t, f.t.Context(), f.nodes, f.pool, f.node, f.incarnation, f.release, f.now, uint64(detail.ConfigGeneration.Desired), true)
}

func (f *intentGuardFixture) execute(id uuid.UUID) api.Operation {
	f.t.Helper()
	op, err := f.exits.ExecuteOperation(f.t.Context(), id)
	if err != nil {
		f.t.Fatal(err)
	}
	return op
}

func (f *intentGuardFixture) observed() string {
	f.t.Helper()
	var observed string
	if err := f.pool.QueryRow(f.t.Context(), `SELECT publication_observed FROM exits WHERE id=$1`, f.exit).Scan(&observed); err != nil {
		f.t.Fatal(err)
	}
	return observed
}

func TestRecoveryModeT5_PublicationIntentGuard(t *testing.T) {
	t.Run("publish_then_unpublish", func(t *testing.T) {
		f := newIntentGuardFixture(t, api.PublicationIntentUnpublished)
		publish := f.set(api.PublicationIntentPublished)
		withdraw := f.set(api.PublicationIntentUnpublished)
		if op := f.execute(withdraw.OperationId); op.Status != api.OperationStatusSucceeded && op.Status != api.OperationStatusPartiallySucceeded {
			t.Fatalf("withdrawal did not complete: %s", op.Status)
		}
		if f.adapter.WithdrawCalls() != 1 || f.observed() != "withdrawn" {
			t.Fatal("withdrawal was not observed")
		}
		// A current, ready observation makes an obsolete publish actionable if
		// the intent check is missing.
		f.acknowledge()
		calls := f.adapter.PublishCalls()
		if op := f.execute(publish.OperationId); op.Status != api.OperationStatusSuperseded {
			t.Errorf("obsolete publish status=%s, want superseded", op.Status)
		}
		if f.adapter.PublishCalls() != calls || f.observed() != "withdrawn" {
			t.Error("obsolete publish changed the withdrawn route")
		}
	})
	for _, intent := range []api.PublicationIntent{api.PublicationIntentUnpublished, api.PublicationIntentDisabled} {
		t.Run(string(intent)+"_then_publish", func(t *testing.T) {
			f := newIntentGuardFixture(t, api.PublicationIntentPublished)
			withdraw := f.set(intent)
			publish := f.set(api.PublicationIntentPublished)
			f.acknowledge()
			if op := f.execute(publish.OperationId); op.Status != api.OperationStatusSucceeded {
				t.Fatalf("current publish status=%s", op.Status)
			}
			if f.adapter.PublishCalls() != 1 {
				t.Fatal("current publish did not reach adapter")
			}
			calls := f.adapter.WithdrawCalls()
			if op := f.execute(withdraw.OperationId); op.Status != api.OperationStatusSuperseded {
				t.Errorf("obsolete withdrawal status=%s, want superseded", op.Status)
			}
			if f.adapter.WithdrawCalls() != calls || f.observed() != "active" {
				t.Error("obsolete withdrawal changed the published route")
			}
		})
	}
	t.Run("failed_publish_then_unpublish", func(t *testing.T) {
		f := newIntentGuardFixture(t, api.PublicationIntentUnpublished)
		publish := f.set(api.PublicationIntentPublished)
		f.acknowledge()
		f.adapter.SetPublishError(errors.New("TEST ONLY publish failure"))
		if op := f.execute(publish.OperationId); op.Status != api.OperationStatusFailed {
			t.Fatalf("publish failure status=%s, want failed", op.Status)
		}
		f.adapter.SetPublishError(nil)
		withdraw := f.set(api.PublicationIntentUnpublished)
		if op := f.execute(withdraw.OperationId); op.Status != api.OperationStatusSucceeded && op.Status != api.OperationStatusPartiallySucceeded {
			t.Fatalf("withdrawal did not complete: %s", op.Status)
		}
		if f.observed() != "withdrawn" {
			t.Fatal("withdrawal was not observed")
		}
		f.acknowledge()
		calls := f.adapter.PublishCalls()
		if op := f.execute(publish.OperationId); op.Status != api.OperationStatusSuperseded {
			t.Errorf("obsolete failed publish status=%s, want superseded", op.Status)
		}
		if f.adapter.PublishCalls() != calls || f.observed() != "withdrawn" {
			t.Error("obsolete failed publish changed the withdrawn route")
		}
	})
	t.Run("failed_unpublish_then_publish", func(t *testing.T) {
		f := newIntentGuardFixture(t, api.PublicationIntentPublished)
		withdraw := f.set(api.PublicationIntentUnpublished)
		f.adapter.SetWithdrawError(errors.New("TEST ONLY withdraw failure"))
		if op := f.execute(withdraw.OperationId); op.Status != api.OperationStatusFailed {
			t.Fatalf("withdraw failure status=%s, want failed", op.Status)
		}
		f.adapter.SetWithdrawError(nil)
		publish := f.set(api.PublicationIntentPublished)
		f.acknowledge()
		if op := f.execute(publish.OperationId); op.Status != api.OperationStatusSucceeded {
			t.Fatalf("current publish status=%s, want succeeded", op.Status)
		}
		calls := f.adapter.WithdrawCalls()
		if op := f.execute(withdraw.OperationId); op.Status != api.OperationStatusSuperseded {
			t.Errorf("obsolete failed withdrawal status=%s, want superseded", op.Status)
		}
		if f.adapter.WithdrawCalls() != calls || f.observed() != "active" {
			t.Error("obsolete failed withdrawal changed the published route")
		}
	})
	t.Run("partially_succeeded_withdrawal_is_kept", func(t *testing.T) {
		f := newIntentGuardFixture(t, api.PublicationIntentPublished)
		withdraw := f.set(api.PublicationIntentUnpublished)
		if op := f.execute(withdraw.OperationId); op.Status != api.OperationStatusPartiallySucceeded {
			t.Fatalf("unconfirmed withdrawal status=%s, want partially_succeeded", op.Status)
		}
		f.set(api.PublicationIntentPublished)
		superseded, err := f.exits.supersedeContradictoryPublication(t.Context(), withdraw.OperationId, f.node)
		if err != nil || superseded {
			t.Fatalf("partial withdrawal guard: superseded=%v err=%v", superseded, err)
		}
		op, err := nodes.GetOperation(t.Context(), f.pool, withdraw.OperationId)
		if err != nil {
			t.Fatal(err)
		}
		if op.Status != api.OperationStatusPartiallySucceeded {
			t.Errorf("unconfirmed withdrawal status=%s, want partially_succeeded", op.Status)
		}
	})
	for _, kind := range []string{nodes.OpKindRevoke, nodes.OpKindRetire} {
		t.Run(kind+"_with_published_intent", func(t *testing.T) {
			f := newIntentGuardFixture(t, api.PublicationIntentPublished)
			var withdrawal api.Operation
			err := db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
				var err error
				withdrawal, err = nodes.CreateOperation(t.Context(), tx, nodes.CreateOperationParams{Kind: kind, Principal: "TEST ONLY", ResourceID: f.node, Now: f.now})
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if op := f.execute(withdrawal.OperationId); (op.Status != api.OperationStatusSucceeded && op.Status != api.OperationStatusPartiallySucceeded) || f.adapter.WithdrawCalls() != 1 {
				t.Fatalf("%s status=%s withdrawals=%d", kind, op.Status, f.adapter.WithdrawCalls())
			}
		})
	}
	t.Run("current_publish", func(t *testing.T) {
		f := newIntentGuardFixture(t, api.PublicationIntentUnpublished)
		publish := f.set(api.PublicationIntentPublished)
		f.acknowledge()
		if op := f.execute(publish.OperationId); op.Status != api.OperationStatusSucceeded || f.adapter.PublishCalls() != 1 {
			t.Fatalf("current publish status=%s calls=%d", op.Status, f.adapter.PublishCalls())
		}
	})
	t.Run("no_exit", func(t *testing.T) {
		f := newIntentGuardFixture(t, api.PublicationIntentUnpublished)
		node := uuid.New()
		if _, err := f.pool.Exec(t.Context(), `INSERT INTO nodes(id) VALUES($1)`, node); err != nil {
			t.Fatal(err)
		}
		var publish api.Operation
		if err := db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var err error
			publish, err = nodes.CreateOperation(t.Context(), tx, nodes.CreateOperationParams{Kind: nodes.OpKindPublish, Principal: "TEST ONLY", ResourceID: node, Now: f.now})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		superseded, err := f.exits.supersedeContradictoryPublication(t.Context(), publish.OperationId, node)
		if err != nil || superseded {
			t.Fatalf("no-exit guard: superseded=%v err=%v", superseded, err)
		}
	})
}
