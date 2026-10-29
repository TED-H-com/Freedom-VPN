package exits

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Worker coordinates background outbox consumption for desired-state recompilation,
// publication/withdrawal operation execution, and NetBird read-back drift reconciliation (§5.2, §15.4, §17.4).
type Worker struct {
	pool     *pgxpool.Pool
	exits    *Service
	compiler *desiredstate.Compiler
	workerID string
	leaseTTL time.Duration
}

// NewWorker constructs a Worker for node desired-state, exit publication, and NetBird reconciliation jobs.
func NewWorker(pool *pgxpool.Pool, exitsSvc *Service, compiler *desiredstate.Compiler, workerID string) *Worker {
	if workerID == "" {
		workerID = "nodes-worker"
	}
	return &Worker{
		pool:     pool,
		exits:    exitsSvc,
		compiler: compiler,
		workerID: workerID,
		leaseTTL: time.Minute,
	}
}

// ProcessOnce claims up to limit outbox events, coalesces desired-state recompilations
// across the batch, executes any triggered publication/withdrawal operations, and completes
// or retries the claimed outbox events (§5.2, §15.4).
func (w *Worker) ProcessOnce(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	claimed, err := db.ClaimOutbox(ctx, w.pool, w.workerID, limit, w.leaseTTL)
	if err != nil || len(claimed) == 0 {
		return 0, err
	}
	if err := w.HandleEvents(ctx, claimed); err != nil {
		for _, ev := range claimed {
			_ = db.RetryOutbox(ctx, w.pool, ev, 5*time.Second)
		}
		return 0, err
	}
	for _, ev := range claimed {
		if err := db.CompleteOutbox(ctx, w.pool, ev); err != nil {
			return 0, err
		}
	}
	return len(claimed), nil
}

// HandleEvents processes a slice of claimed outbox events:
// 1. Coalesces and runs desired-state recompilation for all affected nodes.
// 2. Executes any associated publication/withdrawal/node operations.
func (w *Worker) HandleEvents(ctx context.Context, events []db.ClaimedEvent) error {
	if len(events) == 0 {
		return nil
	}

	// 1. Coalesced desired-state recompilation across the burst of events.
	if w.compiler != nil {
		if _, err := w.compiler.RecompileBatch(ctx, events); err != nil {
			return err
		}
	}

	// 2. Execute any publication/withdrawal operations referenced in the events.
	if w.exits != nil {
		seenOps := map[uuid.UUID]bool{}
		for _, ev := range events {
			var payload struct {
				OperationID string `json:"operation_id"`
			}
			if json.Unmarshal(ev.Payload, &payload) == nil && payload.OperationID != "" {
				if opID, err := uuid.Parse(payload.OperationID); err == nil && !seenOps[opID] {
					seenOps[opID] = true
					if _, err := w.exits.ExecuteOperation(ctx, opID); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// Reconcile runs NetBird read-back reconciliation and node staleness detection (§5.1, §17.4).
func (w *Worker) Reconcile(ctx context.Context) ([]ReconcileItem, error) {
	if w.exits == nil {
		return nil, nil
	}
	return w.exits.ReconcileNetBird(ctx)
}
