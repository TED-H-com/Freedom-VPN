//go:build integration

package leases

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type recoveryLeaseFixture struct {
	pool                  *pgxpool.Pool
	lease                 *Service
	usage                 *usage.Service
	trust                 *envelope.TrustStore
	node, inc, member     uuid.UUID
	unpublished, unpubInc uuid.UUID
	maxOffline            time.Duration
}

func newRecoveryLeaseFixture(t *testing.T) *recoveryLeaseFixture {
	t.Helper()
	ctx := t.Context()
	pool := dbtest.New(t)
	now := time.Now().UTC().Truncate(time.Second)
	group, member := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups(id,code,name) VALUES($1,'TEST ONLY recovery','TEST ONLY recovery')`, group); err != nil {
		t.Fatal(err)
	}
	node, inc := seedLeaseTestNode(t, ctx, pool, group, api.PublicationIntentPublished)
	unpublished, unpubInc := seedLeaseTestNode(t, ctx, pool, group, api.PublicationIntentUnpublished)
	if _, err := pool.Exec(ctx, `INSERT INTO members(id,status) VALUES($1,'active')`, member); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO entitlements
		(member_id,revision,state,device_limit,valid_from,effective_service_end,projection)
		VALUES($1,1,'active',3,$2,$3,jsonb_build_object('exit_group_ids',to_jsonb(ARRAY[$4::text])))`,
		member, now.Add(-time.Hour), now.Add(7*24*time.Hour), group.String()); err != nil {
		t.Fatal(err)
	}
	signer, trust := testLeaseSignerAndTrustStore(t, now)
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - ledger epoch recovery lease test"))
	ledgerSigner := usage.NewLedgerEpochSigner("TEST ONLY epoch key", ed25519.NewKeyFromSeed(seed[:]))
	usageService, err := usage.NewService(usage.Config{Pool: pool, LedgerEpochSigner: ledgerSigner, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	var maxOfflineSec int
	if err := pool.QueryRow(ctx, `SELECT max_offline_lease_seconds FROM fleet_config`).Scan(&maxOfflineSec); err != nil {
		t.Fatal(err)
	}
	return &recoveryLeaseFixture{
		pool: pool, lease: NewService(pool, signer, WithClock(func() time.Time { return now })), usage: usageService, trust: trust,
		node: node, inc: inc, member: member, unpublished: unpublished, unpubInc: unpubInc,
		maxOffline: time.Duration(maxOfflineSec) * time.Second,
	}
}

func leaseRequest(node, inc uuid.UUID) api.LeaseRenewalRequest {
	return api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: node, NodeIncarnationId: inc}
}

func (f *recoveryLeaseFixture) renew(t *testing.T, req api.LeaseRenewalRequest) api.ServingLeaseEnvelope {
	t.Helper()
	env, err := f.lease.Renew(t.Context(), req.NodeId, req.NodeIncarnationId, req)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	return env
}

func (f *recoveryLeaseFixture) freeze(t *testing.T) {
	t.Helper()
	if err := f.usage.FreezeIngest(t.Context(), "TEST ONLY actor", "TEST ONLY restore"); err != nil {
		t.Fatalf("freeze: %v", err)
	}
}

func (f *recoveryLeaseFixture) beginRecovery(t *testing.T) {
	t.Helper()
	if _, err := f.usage.BeginRecovery(t.Context(), usage.RecoveryRequest{Actor: "TEST ONLY actor", Reason: "TEST ONLY restore"}); err != nil {
		t.Fatalf("begin recovery: %v", err)
	}
}

func assertRecoveryLease(t *testing.T, f *recoveryLeaseFixture, env api.ServingLeaseEnvelope, memberCount int, duration time.Duration) {
	t.Helper()
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := envelope.Verify(raw, f.trust, envelope.PurposeLease, env.Signed.IssuedAt); err != nil {
		t.Errorf("lease signature: %v", err)
	}
	if len(env.Signed.Members) != memberCount {
		t.Errorf("member grants = %d, want %d", len(env.Signed.Members), memberCount)
	} else if memberCount == 1 && env.Signed.Members[0].MemberId != f.member {
		t.Errorf("member grant = %s, want %s", env.Signed.Members[0].MemberId, f.member)
	}
	if got := env.Signed.ExpiresAt.Sub(env.Signed.IssuedAt); got != duration {
		t.Errorf("lease duration = %s, want %s", got, duration)
	}
	if env.Signed.MaxOfflineSeconds != int(f.maxOffline/time.Second) {
		t.Errorf("max offline seconds changed: %d", env.Signed.MaxOfflineSeconds)
	}
	var rows int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM serving_lease_members WHERE lease_id=$1`, env.Signed.LeaseId).Scan(&rows); err != nil || rows != memberCount {
		t.Errorf("persisted member grants = %d, want %d (error %v)", rows, memberCount, err)
	}
}

type recoveryForbiddenReader struct{ calls int }

func (r *recoveryForbiddenReader) ListLeaseEntitlementsForGroups(context.Context, nodes.Querier, []uuid.UUID, time.Time) ([]MemberLeaseInput, error) {
	r.calls++
	return nil, errors.New("entitlements must not be read during recovery")
}

func TestAT29_RecoveryLeaseFence(t *testing.T) {
	f := newRecoveryLeaseFixture(t)
	normal := f.renew(t, leaseRequest(f.node, f.inc))
	assertRecoveryLease(t, f, normal, 1, f.maxOffline)
	f.freeze(t)
	reader := f.lease.entitlements
	forbidden := &recoveryForbiddenReader{}
	f.lease.entitlements = forbidden
	for _, mode := range []usage.LedgerMode{usage.LedgerModeFrozen, usage.LedgerModeReconciling} {
		if mode == usage.LedgerModeReconciling {
			f.beginRecovery(t)
		}
		t.Run(string(mode), func(t *testing.T) {
			state, err := f.usage.GetLedgerState(t.Context())
			if err != nil || state.Mode != mode {
				t.Fatalf("mode = %s, want %s (error %v)", state.Mode, mode, err)
			}
			grantless := f.renew(t, leaseRequest(f.node, f.inc))
			assertRecoveryLease(t, f, grantless, 0, min(f.maxOffline, time.Duration(ProvisionalGrantlessLeaseSeconds)*time.Second))
			if grantless.Signed.NodeId != normal.Signed.NodeId || grantless.Signed.NodeIncarnationId != normal.Signed.NodeIncarnationId ||
				grantless.Signed.SchemaVersion != normal.Signed.SchemaVersion || grantless.Signed.ConfigGeneration != normal.Signed.ConfigGeneration ||
				grantless.Signed.AuthorizationGeneration != normal.Signed.AuthorizationGeneration {
				t.Error("recovery changed unrelated lease fields")
			}
			assertRecoveryLease(t, f, f.renew(t, leaseRequest(f.unpublished, f.unpubInc)), 0, f.maxOffline)
		})
	}
	if forbidden.calls != 0 {
		t.Errorf("entitlement reader called %d times in recovery", forbidden.calls)
	}
	f.lease.entitlements = reader
	if err := f.usage.CompleteRecovery(t.Context(), "TEST ONLY actor"); err != nil {
		t.Fatalf("complete recovery: %v", err)
	}
	assertRecoveryLease(t, f, f.renew(t, leaseRequest(f.node, f.inc)), 1, f.maxOffline)
}

func TestAT29_RecoveryLeaseRespectsSmallerOfflineCap(t *testing.T) {
	f := newRecoveryLeaseFixture(t)
	if _, err := f.pool.Exec(t.Context(), `UPDATE fleet_config SET max_offline_lease_seconds=30, resource_version=resource_version+1`); err != nil {
		t.Fatal(err)
	}
	f.maxOffline = 30 * time.Second
	f.freeze(t)
	assertRecoveryLease(t, f, f.renew(t, leaseRequest(f.node, f.inc)), 0, f.maxOffline)
}

func assertRecoveryReplayRefused(t *testing.T, err error) {
	t.Helper()
	var classified *apierr.Error
	if !errors.As(err, &classified) || classified.Code != apierr.DependencyUnavailable || classified.Message != "usage ledger recovery in progress" {
		t.Errorf("replay error = %v, want DEPENDENCY_UNAVAILABLE: usage ledger recovery in progress", err)
	}
}

func TestAT29_RecoveryStoredLeaseReplay(t *testing.T) {
	f := newRecoveryLeaseFixture(t)
	grantReq := leaseRequest(f.node, f.inc)
	grant := f.renew(t, grantReq)
	assertRecoveryLease(t, f, grant, 1, f.maxOffline)
	f.freeze(t)
	emptyReq := leaseRequest(f.node, f.inc)
	empty := f.renew(t, emptyReq)
	assertRecoveryLease(t, f, empty, 0, min(f.maxOffline, time.Duration(ProvisionalGrantlessLeaseSeconds)*time.Second))
	for _, mode := range []usage.LedgerMode{usage.LedgerModeFrozen, usage.LedgerModeReconciling} {
		if mode == usage.LedgerModeReconciling {
			f.beginRecovery(t)
		}
		t.Run(string(mode), func(t *testing.T) {
			var before, after [3]int
			query := `SELECT (SELECT count(*) FROM serving_leases), (SELECT count(*) FROM serving_lease_members),
				(SELECT count(*) FROM audit_events WHERE action='node.lease_renewed')`
			if err := f.pool.QueryRow(t.Context(), query).Scan(&before[0], &before[1], &before[2]); err != nil {
				t.Fatal(err)
			}
			_, err := f.lease.Renew(t.Context(), f.node, f.inc, grantReq)
			assertRecoveryReplayRefused(t, err)
			_, err = f.lease.ReadRenewal(t.Context(), f.node, f.inc, grantReq.RenewalRequestId)
			assertRecoveryReplayRefused(t, err)
			got := f.renew(t, emptyReq)
			if !reflect.DeepEqual(got, empty) {
				t.Error("grantless renewal replay changed stored envelope")
			}
			read, err := f.lease.ReadRenewal(t.Context(), f.node, f.inc, emptyReq.RenewalRequestId)
			if err != nil || !reflect.DeepEqual(read, empty) {
				t.Errorf("grantless receipt replay changed stored envelope: %v", err)
			}
			if err := f.pool.QueryRow(t.Context(), query).Scan(&after[0], &after[1], &after[2]); err != nil || after != before {
				t.Errorf("replay changed lease, member, or audit counts from %v to %v (error %v)", before, after, err)
			}
		})
	}
}

func TestAT29_RecoveryLedgerReadFailureFailsClosed(t *testing.T) {
	f := newRecoveryLeaseFixture(t)
	ctx := t.Context()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	// The singleton's delete guard and mode CHECK are normally enforced. Disable
	// the guard only in this rolled-back test transaction to exercise no row.
	if _, err = tx.Exec(ctx, `ALTER TABLE usage_ledger_state DISABLE TRIGGER usage_ledger_state_keep`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM usage_ledger_state`); err != nil {
		t.Fatal(err)
	}
	_, err = f.lease.RenewTx(ctx, tx, f.node, f.inc, leaseRequest(f.node, f.inc))
	if errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("missing ledger error retains pgx.ErrNoRows: %v", err)
	}
	// db.InTx returns db.MapError(err) for a non-retryable transaction error.
	mappedErr := db.MapError(err)
	var classified *apierr.Error
	if !errors.As(mappedErr, &classified) || classified.Code != apierr.DependencyUnavailable {
		t.Errorf("mapped missing ledger error = %v, want %s", mappedErr, apierr.DependencyUnavailable)
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM serving_leases`).Scan(&count); err != nil || count != 0 {
		t.Errorf("leases after ledger read failure = %d (error %v), want 0", count, err)
	}
}

// waitForLock returns the blocked backend's current query and blockers, or
// finds a backend blocked by blocker when pid is zero.
func waitForLock(ctx context.Context, pool *pgxpool.Pool, pid, blocker int) (int, string, []int32, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var found int
		var query string
		var blockers []int32
		err := pool.QueryRow(ctx, `SELECT pid,query,pg_blocking_pids(pid) FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND ($1::int=0 OR pid=$1) AND ($2::int=0 OR $2=ANY(pg_blocking_pids(pid))) LIMIT 1`, pid, blocker).Scan(&found, &query, &blockers)
		if err == nil {
			return found, query, blockers, nil
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

func TestAT29_RecoveryFreezeRenewalLockOrder(t *testing.T) {
	t.Run("renewal_first", func(t *testing.T) {
		f := newRecoveryLeaseFixture(t)
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()
		tx, err := f.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		var pid int
		if err = tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		if _, err = f.lease.RenewTx(ctx, tx, f.node, f.inc, leaseRequest(f.node, f.inc)); err != nil {
			t.Fatalf("renewal: %v", err)
		}
		done := make(chan error, 1)
		go func() { done <- f.usage.FreezeIngest(ctx, "TEST ONLY actor", "TEST ONLY restore") }()
		_, query, _, err := waitForLock(ctx, f.pool, 0, pid)
		if err != nil || !strings.Contains(query, "usage_ledger_state") {
			t.Errorf("freeze did not wait for renewal ledger lock: query=%q error=%v", query, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("renewal commit: %v", err)
		}
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("freeze after renewal: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("freeze did not complete before deadline")
		}
	})

	t.Run("freeze_first", func(t *testing.T) {
		f := newRecoveryLeaseFixture(t)
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()
		// Gate the node so the real freeze holds the ledger while waiting for
		// nodes. This exposes the order in which the renewal attempts its locks.
		gate, err := f.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer gate.Rollback(context.Background())
		var gatePID int
		if err = gate.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&gatePID); err != nil {
			t.Fatal(err)
		}
		if _, err = gate.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, f.node); err != nil {
			t.Fatal(err)
		}
		freezeDone := make(chan error, 1)
		go func() { freezeDone <- f.usage.FreezeIngest(ctx, "TEST ONLY actor", "TEST ONLY restore") }()
		freezePID, query, _, err := waitForLock(ctx, f.pool, 0, gatePID)
		if err != nil || !strings.Contains(query, "FROM nodes") {
			t.Fatalf("freeze did not reach node lock: query=%q error=%v", query, err)
		}
		renewal, err := f.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer renewal.Rollback(context.Background())
		var renewalPID int
		if err = renewal.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&renewalPID); err != nil {
			t.Fatal(err)
		}
		renewDone := make(chan error, 1)
		go func() {
			_, err := f.lease.RenewTx(ctx, renewal, f.node, f.inc, leaseRequest(f.node, f.inc))
			renewDone <- err
		}()
		_, query, blockers, err := waitForLock(ctx, f.pool, renewalPID, 0)
		if err != nil || !strings.Contains(query, "usage_ledger_state") || !containsPID(blockers, int32(freezePID)) {
			t.Errorf("renewal did not wait for ledger before node: query=%q blockers=%v error=%v", query, blockers, err)
		}
		if err := gate.Commit(ctx); err != nil {
			t.Fatalf("release node gate: %v", err)
		}
		select {
		case err := <-freezeDone:
			if err != nil {
				t.Errorf("freeze: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("freeze did not complete before deadline")
		}
		select {
		case err := <-renewDone:
			if err != nil {
				t.Fatalf("renewal after freeze: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("renewal did not complete before deadline")
		}
		if err := renewal.Commit(ctx); err != nil {
			t.Fatalf("renewal commit: %v", err)
		}
	})
}

func containsPID(pids []int32, want int32) bool {
	for _, pid := range pids {
		if pid == want {
			return true
		}
	}
	return false
}
