//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type clampLeaseRow struct {
	expires  time.Time
	evidence []byte
	xmin     string
}

func readClampLease(t *testing.T, q nodes.Querier, id uuid.UUID) clampLeaseRow {
	t.Helper()
	var row clampLeaseRow
	if err := q.QueryRow(t.Context(), `SELECT expires_at, to_jsonb(l)-'expires_at', xmin::text FROM serving_leases l WHERE id=$1`, id).Scan(&row.expires, &row.evidence, &row.xmin); err != nil {
		t.Fatal(err)
	}
	row.expires = row.expires.UTC()
	return row
}

func requireClampImmutable(t *testing.T, err error) {
	t.Helper()
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) || pgerr.Code != "23514" || pgerr.Message != "immutable record" {
		t.Fatalf("want immutable record (23514), got %v", err)
	}
}

// D-084 / D-088 / AT29: the former overlay proof now exercises the actual
// conflict transaction, including rollback, stored deadlines and exact replay.
func TestReviewR15_StoredLeaseClampGuardProof(t *testing.T) {
	f, enrolled, cert, _, _ := servingFixture(t)
	ctx := t.Context()
	f.sql(`UPDATE nodes SET desired_config_generation=1,resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId)
	req := api.LeaseRenewalRequest{
		RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
		CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	idempotencyKey := uuid.NewString()
	firstResponse := f.callFrom("POST", "/api/v1/agent/lease-renewals", req, cert, reviewDomainX, map[string]string{"Idempotency-Key": idempotencyKey})
	first := decode[api.ServingLeaseEnvelope](t, firstResponse, 200)
	if len(first.Signed.Members) == 0 {
		t.Fatal("setup needs a grant-bearing lease")
	}
	// Exercise normal INSERTs with both the default cap and a shorter cap.
	f.sql(`UPDATE fleet_config SET max_offline_lease_seconds=30,resource_version=resource_version+1`)
	short := reviewRenew(t, f, enrolled, cert, reviewDomainX)
	f.sql(`UPDATE fleet_config SET max_offline_lease_seconds=86400,resource_version=resource_version+1`)
	second := reviewRenew(t, f, enrolled, cert, reviewDomainY)
	if len(short.Signed.Members) == 0 || len(second.Signed.Members) == 0 || reviewFrozen(t, f, enrolled.NodeIncarnationId) {
		t.Fatal("ordinary roaming or renewal INSERT lost grants")
	}
	_, other, otherCert := f.enroll(f.create(nil))
	otherLease := reviewRenew(t, f, other, otherCert, reviewDomainX)
	expired, historical := seedClampHistoricalLeases(t, f, enrolled)
	before := map[uuid.UUID]clampLeaseRow{}
	for _, id := range []uuid.UUID{first.Signed.LeaseId, second.Signed.LeaseId, short.Signed.LeaseId, otherLease.Signed.LeaseId, expired, historical} {
		before[id] = readClampLease(t, f.r.Pool, id)
	}
	for _, commit := range []bool{false, true} {
		tx, err := f.r.Pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		var freezeTime time.Time
		if err = tx.QueryRow(ctx, `SELECT now()`).Scan(&freezeTime); err != nil {
			t.Fatal(err)
		}
		freezeTime = freezeTime.UTC()
		allowed, err := f.r.Enrollment.ObserveEndpointTx(ctx, tx, cert, reviewDomainX, 120*time.Second)
		if err != nil || allowed {
			t.Fatalf("conflict: allowed=%v err=%v", allowed, err)
		}
		var frozen, failed bool
		if err = tx.QueryRow(ctx, `SELECT e.mapping_frozen,n.provisioning_state='failed' AND n.safety_status='failed'
			FROM nodes n JOIN node_identity_endpoints e ON e.incarnation_id=n.current_incarnation_id WHERE n.id=$1`, enrolled.NodeId).Scan(&frozen, &failed); err != nil || !frozen || !failed {
			t.Fatalf("freeze must be atomic with failed readiness: %v %v %v", frozen, failed, err)
		}
		for id, previous := range before {
			row := readClampLease(t, tx, id)
			want := previous.expires
			if id == first.Signed.LeaseId || id == second.Signed.LeaseId {
				want = freezeTime.Add(time.Minute)
			} else if row.xmin != previous.xmin {
				t.Fatal("short, expired, other-node or historical-incarnation lease was updated")
			}
			if !row.expires.Equal(want) || !bytes.Equal(row.evidence, previous.evidence) {
				t.Fatalf("clamp changed issuance evidence or wrong expiry: got %s want %s", row.expires, want)
			}
		}
		if !commit {
			if err = tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			for id, previous := range before {
				row := readClampLease(t, f.r.Pool, id)
				if !row.expires.Equal(previous.expires) || row.xmin != previous.xmin {
					t.Fatal("rolled-back freeze changed a stored lease")
				}
			}
			if reviewFrozen(t, f, enrolled.NodeIncarnationId) || reviewConflictCount(t, f, enrolled.NodeIncarnationId) != 0 {
				t.Fatal("rolled-back freeze retained identity changes")
			}
			var ready bool
			if err = f.r.Pool.QueryRow(ctx, `SELECT provisioning_state='ready' AND safety_status='passed' FROM nodes WHERE id=$1`, enrolled.NodeId).Scan(&ready); err != nil || !ready {
				t.Fatalf("rolled-back freeze retained failed readiness: %v", err)
			}
			t.Logf("rollback: freeze_tx=%s; original signed/stored expiry=%s restored", freezeTime.Format(time.RFC3339Nano), before[first.Signed.LeaseId].expires.Format(time.RFC3339Nano))
			continue
		}
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		t.Logf("commit: issued=%s signed_expiry=%s freeze_tx=%s stored_expiry=%s short_expiry=%s (untouched)", first.Signed.IssuedAt.Format(time.RFC3339Nano), first.Signed.ExpiresAt.Format(time.RFC3339Nano), freezeTime.Format(time.RFC3339Nano), readClampLease(t, f.r.Pool, first.Signed.LeaseId).expires.Format(time.RFC3339Nano), before[short.Signed.LeaseId].expires.Format(time.RFC3339Nano))
	}
	// Recreate the HTTP listeners against the same durable database, also
	// resetting the fixture's default five-request lease burst limit.
	f = setupRuntime(t, f.r)
	// Both replay paths retain the original envelope. Neither signs a new TTL
	// nor writes the central deadline back from the old signed deadline.
	for _, key := range []string{"", idempotencyKey} {
		w := f.callFrom("POST", "/api/v1/agent/lease-renewals", req, cert, reviewDomainX, map[string]string{"Idempotency-Key": key})
		if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), firstResponse.Body.Bytes()) {
			t.Fatalf("clamped replay changed envelope: status=%d", w.Code)
		}
	}
	fresh := reviewRenew(t, f, enrolled, cert, reviewDomainX)
	if len(fresh.Signed.Members) != 0 || reviewLeaseTTL(fresh) != time.Minute {
		t.Fatal("frozen renewal INSERT did not produce the grantless 60s backstop")
	}
	if row := readClampLease(t, f.r.Pool, fresh.Signed.LeaseId); row.expires.Sub(fresh.Signed.ExpiresAt).Abs() > time.Microsecond {
		t.Fatal("fresh INSERT deadline differs from signed expiry")
	}
	clamped := readClampLease(t, f.r.Pool, first.Signed.LeaseId)
	if !clamped.expires.Before(first.Signed.ExpiresAt) {
		t.Fatal("replay or renewal restored the original stored expiry")
	}

	// This reader feeds observation operation projections and publication
	// workers. A still-valid signed/report deadline cannot bypass the row cap.
	report := api.ObservationReport{NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
		ProvisioningState: api.ProvisioningStateReady, SafetyStatus: api.SafetyStatusPassed,
		DataHealth: api.DataHealthHealthy, MemberForwardingEnabled: true}
	report.AppliedConfigGeneration.Set(first.Signed.ConfigGeneration)
	report.Lease.LeaseId.Set(first.Signed.LeaseId)
	report.Lease.ExpiresAt.Set(first.Signed.ExpiresAt)
	for _, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		report.Checks = append(report.Checks, api.NodeCheck{Name: name, Status: api.HealthCheckStatusPassed})
	}
	for _, check := range []struct {
		at   time.Time
		want string
	}{
		{clamped.expires.Add(-time.Microsecond), ""}, {clamped.expires, "lease_unconfirmed"},
	} {
		cause, err := nodes.PublicationDataPathFailure(ctx, f.r.Pool, report, uint64(first.Signed.ConfigGeneration), check.at)
		if err != nil || cause != check.want {
			t.Fatalf("stored deadline at %s: cause=%q want=%q err=%v", check.at, cause, check.want, err)
		}
	}

	for _, mutation := range []struct{ name, set string }{
		{"extend", "expires_at=expires_at+interval '1 second'"},
		{"unchanged", "expires_at=expires_at"},
		{"id", "id=gen_random_uuid()"},
		{"node_id", "node_id=gen_random_uuid()"},
		{"incarnation_id", "incarnation_id=gen_random_uuid()"},
		{"policy_generation", "policy_generation=policy_generation+1"},
		{"member_grant_refs", "member_grant_refs=array_append(member_grant_refs,gen_random_uuid())"},
		{"issued_at", "issued_at=issued_at-interval '1 second'"},
		{"snapshot_xid", "snapshot_xid=pg_current_xact_id()"},
		{"key_id", "key_id=key_id||'-changed'"},
		{"payload_hash", "payload_hash=decode(repeat('ff',32),'hex')"},
		{"renewal_request_id", "renewal_request_id=NULL"},
		{"authorization_generation", "authorization_generation=authorization_generation+1"},
		{"envelope", "envelope='{}'::jsonb"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			set := mutation.set
			if mutation.name != "extend" && mutation.name != "unchanged" {
				set += ",expires_at=expires_at-interval '1 second'"
			}
			_, err := f.r.Pool.Exec(ctx, "UPDATE serving_leases SET "+set+" WHERE id=$1", first.Signed.LeaseId)
			requireClampImmutable(t, err)
		})
	}
	_, err := f.r.Pool.Exec(ctx, `DELETE FROM serving_leases WHERE id=$1`, first.Signed.LeaseId)
	requireClampImmutable(t, err)
	_, err = f.r.Pool.Exec(ctx, `TRUNCATE serving_leases CASCADE`)
	requireClampImmutable(t, err)
	t.Log("guard: extension, no-op, every other column, DELETE and TRUNCATE rejected with immutable record (23514); renewal INSERTs passed")
}

// TEST ONLY restore fixtures: an expired current lease and an unexpired lease
// from a retired incarnation must not be changed by the current freeze.
func seedClampHistoricalLeases(t *testing.T, f *fixture, enrolled api.EnrollResponse) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := t.Context()
	expired, historical, inc := uuid.New(), uuid.New(), uuid.New()
	fingerprint := sha256.Sum256([]byte(inc.String()))
	f.sql(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch,revoked_at)
		VALUES($1,$2,$3,'ubuntu','amd64',now())`, inc, enrolled.NodeId, fingerprint[:])
	tx, err := f.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL fvpn.restore='on'`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash)
		VALUES($1,$3,$4,1,now()-interval '2 hours',now()-interval '1 hour','TEST ONLY',$6),
		      ($2,$3,$5,1,now()-interval '1 hour',now()+interval '1 hour','TEST ONLY',$6)`, expired, historical, enrolled.NodeId, enrolled.NodeIncarnationId, inc, fingerprint[:]); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return expired, historical
}

func TestD088_ServingLeaseClampMigrationRoundTrip(t *testing.T) {
	f, enrolled, cert, _, _ := servingFixture(t)
	lease := reviewRenew(t, f, enrolled, cert, reviewDomainX)
	ctx := t.Context()
	sqlDB := stdlib.OpenDBFromPool(f.r.Pool)
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	trigger := func() string {
		t.Helper()
		var definition string
		if err := f.r.Pool.QueryRow(ctx, `SELECT pg_get_triggerdef(oid) FROM pg_trigger WHERE tgrelid='serving_leases'::regclass AND tgname='serving_leases_immutable'`).Scan(&definition); err != nil {
			t.Fatal(err)
		}
		return definition
	}
	up := trigger()
	before := readClampLease(t, f.r.Pool, lease.Signed.LeaseId)
	if _, err = provider.ApplyVersion(ctx, 190, false); err != nil {
		t.Fatal(err)
	}
	t.Logf("Down: %s", trigger())
	_, err = f.r.Pool.Exec(ctx, `UPDATE serving_leases SET expires_at=expires_at-interval '1 second' WHERE id=$1`, lease.Signed.LeaseId)
	requireClampImmutable(t, err)
	if _, err = provider.ApplyVersion(ctx, 190, true); err != nil {
		t.Fatal(err)
	}
	if trigger() != up {
		t.Fatal("reapplied trigger differs")
	}
	t.Logf("Up: %s", up)
	after := readClampLease(t, f.r.Pool, lease.Signed.LeaseId)
	if !after.expires.Equal(before.expires) || !bytes.Equal(after.evidence, before.evidence) || after.xmin != before.xmin {
		t.Fatal("migration round trip changed stored lease")
	}
	// The original owner-only migration UPDATE escape remains available.
	tx, err := f.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL fvpn.migration='on'`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE serving_leases SET expires_at=expires_at+interval '1 second',key_id=key_id||'-migration' WHERE id=$1`, lease.Signed.LeaseId); err != nil {
		t.Fatal("owner migration escape changed:", err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	// The normal runtime guard permits shortening even without a freeze; the
	// freeze transaction is responsible for choosing the incarnation and cap.
	if _, err = f.r.Pool.Exec(ctx, `UPDATE serving_leases SET expires_at=expires_at-interval '1 second' WHERE id=$1`, lease.Signed.LeaseId); err != nil {
		t.Fatal("reapplied Up rejected shortening:", err)
	}
	var pgerr *pgconn.PgError
	_, err = f.r.Pool.Exec(ctx, `UPDATE serving_leases SET expires_at=issued_at WHERE id=$1`, lease.Signed.LeaseId)
	if !errors.As(err, &pgerr) || pgerr.Code != "23514" {
		t.Fatalf("issued_at < expires_at constraint lost: %v", err)
	}
}
