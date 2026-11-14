//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// recoveryBlockedBackend returns a waiting backend and its blockers. The query
// fragments distinguish the real freeze and HTTP operations from the gate.
func recoveryBlockedBackend(ctx context.Context, pool *pgxpool.Pool, exclude int, first, second string) (int, string, []int32, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	if second == "" {
		second = first
	}
	for {
		var pid int
		var query string
		var blockers []int32
		err := pool.QueryRow(ctx, `SELECT pid,query,pg_blocking_pids(pid) FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock' AND pid<>$1
			AND (position($2 in query)>0 OR position($3 in query)>0) LIMIT 1`, exclude, first, second).Scan(&pid, &query, &blockers)
		if err == nil {
			return pid, query, blockers, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, "", nil, err
		}
		select {
		case <-ctx.Done():
			return 0, "", nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func recoveryHasBlocker(pids []int32, want int) bool {
	for _, pid := range pids {
		if pid == int32(want) {
			return true
		}
	}
	return false
}

func recoveryRenewalHTTP(ctx context.Context, t *testing.T, f *fixture, enrolled api.EnrollResponse, cert *x509.Certificate) (*httptest.ResponseRecorder, chan struct{}) {
	t.Helper()
	body, err := json.Marshal(api.LeaseRenewalRequest{
		RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
		CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "https://control.example.invalid/api/v1/agent/lease-renewals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())
	req.RemoteAddr = reviewDomainX
	req.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f.agent.ServeHTTP(response, req)
	}()
	return response, done
}

func recoveryFreeze(ctx context.Context, ledger *usage.Service) chan error {
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- ledger.FreezeIngest(ctx, "TEST ONLY actor", "TEST ONLY restore")
	}()
	return done
}

func TestAT29_HTTPRecoveryLeaseFreezeLockOrder(t *testing.T) {
	for _, freezeFirst := range []bool{true, false} {
		name := "renewal_first"
		if freezeFirst {
			name = "freeze_first"
		}
		t.Run(name, func(t *testing.T) {
			f, enrolled, cert, member, _ := servingFixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			gateNode := enrolled.NodeId
			if freezeFirst {
				// The freeze locks nodes in ID order. Gate a node before the
				// renewing node, leaving the latter available for a NOWAIT check.
				gateNode = uuid.MustParse("00000000-0000-0000-0000-000000000001")
				if gateNode.String() >= enrolled.NodeId.String() {
					t.Fatal("gate node must sort before renewing node")
				}
				f.sql(`INSERT INTO nodes(id) VALUES($1)`, gateNode)
			}
			gate, err := f.r.Pool.Begin(ctx)
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			var freezeDone chan error
			var renewalDone chan struct{}
			defer func() {
				cancel()
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				_ = gate.Rollback(cleanup)
				if freezeDone != nil {
					select {
					case <-freezeDone:
					case <-cleanup.Done():
						t.Error("freeze goroutine did not stop")
					}
				}
				if renewalDone != nil {
					select {
					case <-renewalDone:
					case <-cleanup.Done():
						t.Error("renewal goroutine did not stop")
					}
				}
			}()
			var gatePID int
			if err := gate.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&gatePID); err != nil {
				t.Fatal(err)
			}
			if _, err := gate.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, gateNode); err != nil {
				t.Fatal(err)
			}
			ledger, err := usage.NewService(usage.Config{Pool: f.r.Pool})
			if err != nil {
				t.Fatal(err)
			}
			state, err := ledger.GetLedgerState(ctx)
			if err != nil || state.Mode != usage.LedgerModeNormal {
				t.Fatalf("initial ledger mode = %s (error %v), want normal", state.Mode, err)
			}

			var response *httptest.ResponseRecorder
			if freezeFirst {
				freezeDone = recoveryFreeze(ctx, ledger)
				freezePID, query, blockers, err := recoveryBlockedBackend(ctx, f.r.Pool, 0, "lifecycle_state IN", "")
				if err != nil || !recoveryHasBlocker(blockers, gatePID) {
					t.Fatalf("freeze did not wait on earlier node: query=%q blockers=%v error=%v", query, blockers, err)
				}
				response, renewalDone = recoveryRenewalHTTP(ctx, t, f, enrolled, cert)
				_, query, blockers, err = recoveryBlockedBackend(ctx, f.r.Pool, freezePID, "usage_ledger_state", "")
				if err != nil || !recoveryHasBlocker(blockers, freezePID) {
					t.Fatalf("HTTP renewal did not wait on freeze ledger lock: query=%q blockers=%v error=%v", query, blockers, err)
				}
				// A successful independent NOWAIT lock proves HTTP has not
				// locked the renewing node while waiting for the ledger.
				if _, err := f.r.Pool.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE NOWAIT`, enrolled.NodeId); err != nil {
					t.Errorf("HTTP renewal locked its node before the ledger: %v", err)
				}
			} else {
				response, renewalDone = recoveryRenewalHTTP(ctx, t, f, enrolled, cert)
				renewalPID, query, blockers, err := recoveryBlockedBackend(ctx, f.r.Pool, 0, "SELECT id FROM nodes WHERE id=$1 FOR UPDATE", "")
				if err != nil || !recoveryHasBlocker(blockers, gatePID) {
					t.Fatalf("HTTP renewal did not wait on node gate: query=%q blockers=%v error=%v", query, blockers, err)
				}
				freezeDone = recoveryFreeze(ctx, ledger)
				_, query, blockers, err = recoveryBlockedBackend(ctx, f.r.Pool, renewalPID, "usage_ledger_state", "lifecycle_state IN")
				if err != nil || !strings.Contains(query, "usage_ledger_state") || !recoveryHasBlocker(blockers, renewalPID) {
					t.Errorf("freeze did not wait on HTTP renewal ledger lock: query=%q blockers=%v error=%v", query, blockers, err)
				}
			}

			if err := gate.Commit(ctx); err != nil {
				t.Fatalf("release node gate: %v", err)
			}
			select {
			case err := <-freezeDone:
				if err != nil {
					t.Errorf("freeze failed: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("freeze did not finish before deadline")
			}
			select {
			case <-renewalDone:
			case <-ctx.Done():
				t.Fatal("HTTP renewal did not finish before deadline")
			}
			lease := decode[api.ServingLeaseEnvelope](t, response, http.StatusOK)
			maxOffline := time.Duration(lease.Signed.MaxOfflineSeconds) * time.Second
			wantDuration := maxOffline
			if freezeFirst {
				wantDuration = min(maxOffline, time.Duration(leases.ProvisionalGrantlessLeaseSeconds)*time.Second)
				if len(lease.Signed.Members) != 0 {
					t.Errorf("frozen lease has %d member grants, want none", len(lease.Signed.Members))
				}
			} else if len(lease.Signed.Members) != 1 || lease.Signed.Members[0].MemberId != member {
				t.Errorf("normal lease grants = %v, want entitled member %s", lease.Signed.Members, member)
			}
			if got := lease.Signed.ExpiresAt.Sub(lease.Signed.IssuedAt); got != wantDuration {
				t.Errorf("lease duration = %s, want %s", got, wantDuration)
			}
		})
	}
}
