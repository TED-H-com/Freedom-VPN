//go:build integration

package httpwire_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestReviewR15_LeaseAndObservationLockOrder is two sessions on one node.
// SubmitObservation's commandtx takes fleet_config FOR UPDATE before
// LockAuthenticated takes nodes FOR UPDATE. RenewServingLease, when it
// already has the HTTP command transaction, takes nodes FOR UPDATE in
// ObserveEndpointTx and then fleet_config FOR SHARE in the serving_leases
// insert trigger. Those orders cycle.
func TestReviewR15_LeaseAndObservationLockOrder(t *testing.T) {
	r, enrolled, _ := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c1, err := r.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Release()
	c2, err := r.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Release()
	_, _ = c1.Exec(ctx, `SET deadlock_timeout = '200ms'`)
	_, _ = c2.Exec(ctx, `SET deadlock_timeout = '200ms'`)

	tx1, err := c1.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback(context.Background())
	tx2, err := c2.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx2.Rollback(context.Background())

	if _, err = tx2.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, enrolled.Response.NodeId); err != nil {
		t.Fatal(err)
	}
	// The observation session takes fleet_config before it asks for the node.
	// The lease session already holds the node and has not asked for the fleet
	// row yet. Reversing these two statements lets the lease session take the
	// share lock first, and the test then waits on a session that is not waiting.
	if _, err = tx1.Exec(ctx, `SELECT id FROM fleet_config FOR UPDATE`); err != nil {
		t.Fatalf("observation fleet lock: %v", err)
	}
	var pid int
	if err = tx2.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		_, e := tx2.Exec(ctx, `SELECT max_offline_lease_seconds FROM fleet_config WHERE id='00000000-0000-0000-0000-000000000001' FOR SHARE`)
		errCh <- e
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var wait string
		if err = r.Pool.QueryRow(ctx, `SELECT coalesce(wait_event_type,'') FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&wait); err != nil {
			t.Fatal(err)
		}
		if wait == "Lock" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("lease session did not wait on fleet_config; the lock order was not constructed")
		}
		time.Sleep(20 * time.Millisecond)
	}
	_, errNodes := tx1.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, enrolled.Response.NodeId)
	errFleet := <-errCh
	if codeOf(errNodes) == "40P01" || codeOf(errFleet) == "40P01" {
		t.Fatalf("TODO(review): concurrent lease renewal and observation on node %s deadlock (40P01). Observation holds fleet_config FOR UPDATE and waits for nodes FOR UPDATE; the lease holds nodes FOR UPDATE and waits for fleet_config FOR SHARE inside check_node_lease. db.InTx retries 40001 and 40P01 four times and then returns the error. lease=%v observation=%v",
			enrolled.Response.NodeId, errFleet, errNodes)
	}
	t.Fatalf("lock order did not deadlock: lease=%v observation=%v", errFleet, errNodes)
}

func codeOf(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
