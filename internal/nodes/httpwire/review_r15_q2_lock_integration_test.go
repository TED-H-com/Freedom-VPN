//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestReviewR15_LeaseAndObservationLockOrder drives the real HTTP lease path
// while an observation transaction owns fleet_config. Renewal must wait before
// locking nodes, so the observation can take that node and finish without a cycle.
func TestReviewR15_LeaseAndObservationLockOrder(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		name := "without_idempotency_header"
		if keyed {
			name = "with_idempotency_header"
		}
		t.Run(name, func(t *testing.T) {
			f, enrolled, cert, _, _ := servingFixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			observation, err := f.r.Pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer observation.Rollback(context.Background())
			if _, err = observation.Exec(ctx, `SELECT id FROM fleet_config FOR UPDATE`); err != nil {
				t.Fatal(err)
			}
			var pid int
			if err = observation.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
				t.Fatal(err)
			}

			body, err := json.Marshal(api.LeaseRenewalRequest{
				RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
				CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "https://control.example.invalid/api/v1/agent/lease-renewals", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = reviewDomainX
			req.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
			if keyed {
				req.Header.Set("Idempotency-Key", uuid.NewString())
			}
			response := httptest.NewRecorder()
			done := make(chan struct{})
			go func() {
				defer close(done)
				f.agent.ServeHTTP(response, req)
			}()
			defer func() { cancel(); <-done }()

			for {
				var waiting bool
				if err := f.r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND $1=ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				select {
				case <-done:
					t.Fatalf("renewal completed without waiting on fleet_config: %d %s", response.Code, response.Body.String())
				case <-ctx.Done():
					t.Fatal("renewal did not reach the fleet lock")
				case <-time.After(10 * time.Millisecond):
				}
			}
			// NOWAIT makes the old reversed order fail immediately, without
			// relying on the deadlock detector or db.InTx's retry budget.
			if _, err := observation.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE NOWAIT`, enrolled.NodeId); err != nil {
				t.Fatalf("renewal took nodes before fleet_config: %v", err)
			}
			report := loadObservationReport(t, enrolled.NodeId, enrolled.NodeIncarnationId, uuid.Nil, time.Now().UTC())
			report.CurrentReleaseId.SetNull()
			report.LastGoodReleaseId.SetNull()
			report.Lease.LeaseId.SetNull()
			report.Lease.ExpiresAt.SetNull()
			report.Netbird.PeerId.SetNull()
			if _, err := f.r.Nodes.SubmitObservationTx(ctx, observation, enrolled.NodeId, enrolled.NodeIncarnationId, report); err != nil {
				t.Fatal(err)
			}
			if err := observation.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("renewal failed to finish after observation committed")
			}
			decode[api.ServingLeaseEnvelope](t, response, http.StatusOK)
			var count int
			if err := f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM serving_leases WHERE node_id=$1`, enrolled.NodeId).Scan(&count); err != nil || count != 1 {
				t.Fatalf("renewal receipts=%d, error=%v", count, err)
			}
		})
	}
}
