//go:build integration

package nodes

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type holdFixture struct {
	t      *testing.T
	pool   *pgxpool.Pool
	svc    *Service
	now    time.Time
	n      seededNode
	report api.ObservationReport
}

func newHoldFixture(t *testing.T) *holdFixture {
	f := &holdFixture{t: t, pool: dbtest.New(t), now: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	f.svc = NewService(f.pool, WithClock(func() time.Time { return f.now }))
	rel, group := seedTestReleaseAndGroup(t, t.Context(), f.pool, "hold")
	f.n = seedTestNode(t, t.Context(), f.pool, "TEST ONLY hold", "test", api.PublicationIntentPublished, "active", api.ControlStatusOnline, rel, group)
	f.report = holdReport()
	f.report.NodeId, f.report.NodeIncarnationId = f.n.NodeID, f.n.IncarnationID
	empty := emptyObservationReport()
	f.report.CurrentReleaseId, f.report.LastGoodReleaseId = empty.CurrentReleaseId, empty.LastGoodReleaseId
	f.report.AppliedBindingRevision, f.report.AppliedPolicyRefs = empty.AppliedBindingRevision, empty.AppliedPolicyRefs
	f.report.Netbird.PeerId, f.report.Netbird.Version = empty.Netbird.PeerId, empty.Netbird.Version
	f.report.Update, f.report.Usage = empty.Update, empty.Usage
	f.report.AgentVersion = "1.0.0"
	f.report.Lease.LeaseId.Set(f.lease(f.n, 1))
	return f
}

func (f *holdFixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.t.Context(), sql, args...); err != nil {
		f.t.Fatal(err)
	}
}

// TEST ONLY trusted issuer row: the join must use this immutable envelope's
// generation and identity, not a summary or a newer lease for the same node.
func (f *holdFixture) lease(n seededNode, gen decstr.U64, members ...uuid.UUID) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	err := db.InTx(f.t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		for _, member := range members {
			if _, err := tx.Exec(f.t.Context(), `INSERT INTO members(id,status) VALUES($1,'active')`, member); err != nil {
				return err
			}
			if _, err := tx.Exec(f.t.Context(), `INSERT INTO entitlements(member_id,revision,state,device_limit,valid_from,effective_service_end,projection)
				VALUES($1,1,'active',1,$2,$3,'{}')`, member, f.now.Add(-time.Hour), f.now.Add(time.Hour)); err != nil {
				return err
			}
		}
		env, err := json.Marshal(map[string]any{"signed": api.ServingLease{LeaseId: id, NodeId: n.NodeID, NodeIncarnationId: n.IncarnationID, ConfigGeneration: gen}})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(f.t.Context(), `INSERT INTO serving_leases(id,node_id,incarnation_id,policy_generation,issued_at,expires_at,key_id,payload_hash,envelope)
			VALUES($1,$2,$3,$4::text::numeric,$5,$6,'TEST ONLY',decode(repeat('00',32),'hex'),$7)`, id, n.NodeID, n.IncarnationID, gen.String(), f.now, f.now.Add(time.Hour), env); err != nil {
			return err
		}
		for _, member := range members {
			if _, err := tx.Exec(f.t.Context(), `INSERT INTO serving_lease_members(lease_id,member_id,entitlement_revision,valid_from,expires_at) VALUES($1,$2,1,$3,$4)`, id, member, f.now, f.now.Add(time.Hour)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *holdFixture) submit() {
	f.t.Helper()
	if _, err := f.svc.SubmitObservation(f.t.Context(), f.n.NodeID, f.n.IncarnationID, f.report); err != nil {
		f.t.Fatal(err)
	}
}

func (f *holdFixture) next(after time.Duration, steps, elapsed uint64, alarm bool) {
	f.now = f.now.Add(after)
	f.report.ReportId, f.report.ObservedAt = uuid.New(), f.now
	f.report.Errors[0].Message = holdSummaryText(2, 1, steps, elapsed, alarm)
}

func (f *holdFixture) counts(incidents, samples, alarms int) {
	f.t.Helper()
	for _, v := range []struct {
		sql  string
		want int
	}{
		{`SELECT count(*) FROM node_generation_holds`, incidents},
		{`SELECT count(*) FROM node_generation_hold_reports`, samples},
		{`SELECT count(*) FROM audit_events WHERE action='node.generation_hold.escalated'`, alarms},
	} {
		var got int
		if err := f.pool.QueryRow(f.t.Context(), v.sql).Scan(&got); err != nil || got != v.want {
			f.t.Fatalf("%s: got=%d want=%d err=%v", v.sql, got, v.want, err)
		}
	}
}

func TestAT29_M3_HoldLeaseJoinAndOrdering(t *testing.T) {
	f := newHoldFixture(t)
	other := seedTestNode(t, t.Context(), f.pool, "TEST ONLY other", "test", api.PublicationIntentPublished, "active", api.ControlStatusOnline, f.n.ReleaseID, f.n.GroupID)
	foreign := f.lease(other, 1)
	// A later matching lease exists, but the report still names the stale one.
	matching := f.lease(f.n, 2)
	for _, tc := range []struct {
		name    string
		minimum uint64
		edit    func(*api.ObservationReport)
		want    string
	}{
		{"equal_ack", 2, func(*api.ObservationReport) {}, "stale_generation"},
		{"applied_below_desired", 3, func(*api.ObservationReport) {}, "stale_generation"},
		{"safety_before_hold", 2, func(r *api.ObservationReport) { r.Checks[0].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"safety_before_lag", 3, func(r *api.ObservationReport) { r.Checks[1].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"unknown", 2, func(r *api.ObservationReport) { r.Lease.LeaseId.Set(uuid.New()) }, "lease_unconfirmed"},
		{"unknown_below_desired", 3, func(r *api.ObservationReport) { r.Lease.LeaseId.Set(uuid.New()) }, "lease_unconfirmed"},
		{"foreign_incarnation", 2, func(r *api.ObservationReport) { r.Lease.LeaseId.Set(foreign) }, "lease_unconfirmed"},
		{"matching_lease_with_hold_marker", 2, func(r *api.ObservationReport) { r.Lease.LeaseId.Set(matching) }, "stale_generation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := f.report
			r.Checks = append([]api.NodeCheck(nil), r.Checks...)
			tc.edit(&r)
			cause, err := PublicationDataPathFailure(t.Context(), f.pool, r, tc.minimum, f.now)
			if err != nil || cause != tc.want {
				t.Fatalf("cause=%s want=%s err=%v", cause, tc.want, err)
			}
		})
	}
	for _, id := range []uuid.UUID{uuid.New(), foreign} {
		f.report.ReportId = uuid.New()
		f.report.Lease.LeaseId.Set(id)
		f.submit()
	}
	f.counts(0, 0, 0)
	// A genuine D-075 one-shot readiness grace still neither starts a hold nor
	// makes a lagging ACK publication evidence.
	f.report.ForwardingBlockReasons = nil
	f.report.Errors = nil
	for i := range f.report.Checks {
		f.report.Checks[i].Status = api.HealthCheckStatusPassed
		f.report.Checks[i].ErrorCode.SetNull()
	}
	f.report.AppliedConfigGeneration.Set(2)
	f.report.ReportId = uuid.New()
	f.exec(`UPDATE nodes SET provisioning_state='ready',desired_config_generation=3,resource_version=resource_version+1 WHERE id=$1`, f.n.NodeID)
	f.submit()
	var grace *time.Time
	if err := f.pool.QueryRow(t.Context(), `SELECT stale_generation_grace_at FROM node_incarnations WHERE id=$1`, f.n.IncarnationID).Scan(&grace); err != nil || grace == nil {
		t.Fatalf("D-075 grace=%v err=%v", grace, err)
	}
	f.counts(0, 0, 0)
	cause, err := PublicationDataPathFailure(t.Context(), f.pool, f.report, 3, f.now)
	if err != nil || cause != "stale_generation" {
		t.Fatalf("grace is not convergence: %s %v", cause, err)
	}
}

func TestAT29_M3_HoldRecognitionDoesNotDependOnACK(t *testing.T) {
	for _, ack := range []string{"same_as_lease", "unknown", "zero"} {
		t.Run(ack, func(t *testing.T) {
			f := newHoldFixture(t)
			switch ack {
			case "same_as_lease":
				f.report.AppliedConfigGeneration.Set(1)
			case "unknown":
				f.report.AppliedConfigGeneration.SetNull()
			case "zero":
				f.report.AppliedConfigGeneration.Set(0)
			}
			cause, err := PublicationDataPathFailure(t.Context(), f.pool, f.report, 2, f.now)
			if err != nil || cause != "stale_generation" {
				t.Fatalf("marker lost: %s %v", cause, err)
			}
			f.submit()
			f.counts(1, 1, 0)
			var stored *string
			if err := f.pool.QueryRow(t.Context(), `SELECT applied_generation::text FROM node_generation_hold_reports`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if ack == "unknown" && stored != nil {
				t.Fatal("unknown ACK fabricated", *stored)
			}
			f.now = f.now.Add(30 * time.Second)
			if err := f.svc.ReconcileGenerationHolds(t.Context(), uuid.Nil); err != nil {
				t.Fatal(err)
			}
			f.counts(1, 1, 1)
		})
	}
}

func TestAT29_M3_DelayedAlarmEscalatesWithoutRegressingLatestSample(t *testing.T) {
	f := newHoldFixture(t)
	f.submit()
	f.next(10*time.Second, 1, 0, false)
	f.report.BootId = uuid.New()
	f.submit()
	latest := f.report.ReportId
	f.next(time.Second, 3, 1, true)
	f.report.ObservedAt = f.now.Add(-5 * time.Second)
	f.submit()
	f.counts(1, 3, 1)
	var snapshot, alarm uuid.UUID
	if err := f.pool.QueryRow(t.Context(), `SELECT (after_diff->>'report_id')::uuid,(after_diff->>'alarm_report_id')::uuid FROM audit_events WHERE action='node.generation_hold.escalated'`).Scan(&snapshot, &alarm); err != nil || snapshot != latest || alarm != f.report.ReportId {
		t.Fatalf("delayed alarm evidence: %s %s %v", snapshot, alarm, err)
	}
}

func TestAT29_M3_RestartsRetainIncidentAndCentralDeadline(t *testing.T) {
	f := newHoldFixture(t)
	start := f.now
	f.submit()
	f.submit()
	f.counts(1, 1, 0)
	var available time.Time
	if err := f.pool.QueryRow(t.Context(), `SELECT available_at FROM outbox_events WHERE event_type=$1`, EventGenerationHoldCheck).Scan(&available); err != nil || !available.Equal(start.Add(30*time.Second)) {
		t.Fatalf("deadline=%v err=%v", available, err)
	}
	f.next(10*time.Second, 2, 10000, false)
	f.submit()
	// Process restart may keep the same kernel boot ID; a later kernel reboot
	// changes it. Neither starts another incident or rewrites previous counters.
	f.next(5*time.Second, 1, 0, false)
	f.svc = NewService(f.pool, WithClock(func() time.Time { return f.now }))
	f.submit()
	f.report.BootId = uuid.New()
	f.next(5*time.Second, 1, 0, false)
	f.submit()
	f.counts(1, 4, 0)
	var first, last time.Time
	var boots int
	if err := f.pool.QueryRow(t.Context(), `SELECT first_seen_at,last_seen_at FROM node_generation_holds`).Scan(&first, &last); err != nil || !first.Equal(start) || !last.Equal(f.now) {
		t.Fatalf("incident reset: %v %v %v", first, last, err)
	}
	if err := f.pool.QueryRow(t.Context(), `SELECT count(DISTINCT boot_id) FROM node_generation_hold_reports`).Scan(&boots); err != nil || boots != 2 {
		t.Fatalf("boots=%d %v", boots, err)
	}
	f.now = start.Add(30*time.Second - time.Microsecond)
	if err := f.svc.ReconcileGenerationHolds(t.Context(), uuid.Nil); err != nil {
		t.Fatal(err)
	}
	f.counts(1, 4, 0)
	f.now = start.Add(30 * time.Second)
	// No report at the boundary. Two replicas/replayed worker work still alarm once.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() { errs <- f.svc.ReconcileGenerationHolds(t.Context(), f.n.NodeID) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	f.counts(1, 4, 1)
	f.next(time.Second, 1, 0, false)
	f.submit()
	f.counts(1, 5, 1)
	var alarm bool
	var steps string
	if err := f.pool.QueryRow(t.Context(), `SELECT (after_diff->>'reported_alarm')::boolean,after_diff->>'reported_steps' FROM audit_events WHERE action='node.generation_hold.escalated'`).Scan(&alarm, &steps); err != nil || alarm || steps != "1" {
		t.Fatalf("rewrote local counters: %v %s %v", alarm, steps, err)
	}
}

func TestAT29_M3_AlarmRollbackAndMalformedSummary(t *testing.T) {
	f := newHoldFixture(t)
	f.report.Errors[0].Message = holdSummaryText(2, 1, 3, 1, true)
	abort := errors.New("TEST ONLY rollback")
	err := commandtx.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := f.svc.SubmitObservationTx(t.Context(), tx, f.n.NodeID, f.n.IncarnationID, f.report); err != nil {
			return err
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatal(err)
	}
	f.counts(0, 0, 0)
	f.submit()
	f.submit()
	f.counts(1, 1, 1)
	// A malformed report remains a hold, with unknown counters. It neither
	// pretends recovery nor manufactures a reported alarm value.
	f.next(time.Second, 1, 0, false)
	f.report.Errors[0].Message = "Serving lease generation hold: malformed"
	f.submit()
	var unknown bool
	if err := f.pool.QueryRow(t.Context(), `SELECT NOT summary_valid AND reported_steps IS NULL AND reported_elapsed_ms IS NULL AND reported_alarm IS NULL FROM node_generation_hold_reports WHERE report_id=$1`, f.report.ReportId).Scan(&unknown); err != nil || !unknown {
		t.Fatalf("malformed=%v err=%v", unknown, err)
	}
	f.counts(1, 2, 1)
}

func TestAT29_M3_CurrentIncarnationCannotBorrowOldLease(t *testing.T) {
	f := newHoldFixture(t)
	newInc := uuid.New()
	f.exec(`UPDATE node_incarnations SET revoked_at=$2 WHERE id=$1`, f.n.IncarnationID, f.now)
	f.exec(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,os_version,arch)
		VALUES($1,$2,decode(repeat('01',32),'hex'),'ubuntu','24.04','amd64')`, newInc, f.n.NodeID)
	f.exec(`UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, f.n.NodeID, newInc)
	f.n.IncarnationID = newInc
	f.report.NodeIncarnationId = newInc
	cause, err := PublicationDataPathFailure(t.Context(), f.pool, f.report, 2, f.now)
	if err != nil || cause != "lease_unconfirmed" {
		t.Fatalf("old incarnation lease: %s %v", cause, err)
	}
	f.submit()
	f.counts(0, 0, 0)
}

func TestAT29_M3_MalformedFirstSummaryStillEscalatesAndPartialRevocationRetainsHold(t *testing.T) {
	f := newHoldFixture(t)
	first, second := uuid.New(), uuid.New()
	f.report.Lease.LeaseId.Set(f.lease(f.n, 1, first, second))
	f.report.Errors[0].Message = "Serving lease generation hold: malformed"
	f.submit()
	f.counts(1, 1, 0)
	f.exec(`UPDATE members SET status='suspended',resource_version=resource_version+1 WHERE id=$1`, first)
	f.now = f.now.Add(30 * time.Second)
	if err := f.svc.ReconcileGenerationHolds(t.Context(), uuid.Nil); err != nil {
		t.Fatal(err)
	}
	f.counts(1, 1, 1)
	var active bool
	if err := f.pool.QueryRow(t.Context(), `SELECT ended_at IS NULL FROM node_generation_holds`).Scan(&active); err != nil || !active {
		t.Fatalf("survivor incident lost: %v %v", active, err)
	}
	// Valid large values are stored exactly, not maxed with a prior boot or
	// coerced into a central-computed alarm value.
	f.next(time.Second, 18446744073709551615, 9007199254740993, false)
	f.submit()
	var steps, elapsed string
	if err := f.pool.QueryRow(t.Context(), `SELECT reported_steps::text,reported_elapsed_ms::text FROM node_generation_hold_reports WHERE report_id=$1`, f.report.ReportId).Scan(&steps, &elapsed); err != nil || steps != "18446744073709551615" || elapsed != "9007199254740993" {
		t.Fatalf("counter precision: %s %s %v", steps, elapsed, err)
	}
	// No false recovery when a summary names the wrong applied generation.
	f.next(time.Second, 3, 2, true)
	f.report.Errors[0].Message = holdSummaryText(9, 1, 3, 2, true)
	f.submit()
	var valid, alarm bool
	if err := f.pool.QueryRow(t.Context(), `SELECT summary_valid,reported_alarm FROM node_generation_hold_reports WHERE report_id=$1`, f.report.ReportId).Scan(&valid, &alarm); err != nil || valid || !alarm {
		t.Fatalf("summary mismatch: %v %v %v", valid, alarm, err)
	}
	f.counts(1, 3, 1)
}

func TestAT29_M3_HoldEndsOnlyOnEvidence(t *testing.T) {
	for _, ending := range []string{"generation_matched", "identity_frozen", "identity_inactive", "members_revoked"} {
		t.Run(ending, func(t *testing.T) {
			f := newHoldFixture(t)
			var member uuid.UUID
			if ending == "members_revoked" {
				member = uuid.New()
				f.report.Lease.LeaseId.Set(f.lease(f.n, 1, member))
			}
			f.submit()
			old := f.report
			old.Errors = append([]api.NodeError(nil), old.Errors...)
			f.next(time.Second, 1, 0, false)
			switch ending {
			case "generation_matched":
				// Unknown lease/no marker is not evidence that the mismatch ended.
				f.report.Lease.LeaseId.Set(uuid.New())
				f.report.ForwardingBlockReasons = nil
				f.report.Errors = nil
				f.submit()
				f.report.ReportId = uuid.New()
				f.report.ObservedAt = f.now.Add(time.Second)
				f.report.Lease.LeaseId.Set(f.lease(f.n, 2))
				f.submit()
			case "identity_frozen":
				f.exec(`INSERT INTO node_identity_endpoints(incarnation_id,endpoint_hash,last_seen_at,mapping_frozen) VALUES($1,decode(repeat('00',32),'hex'),$2,true)`, f.n.IncarnationID, f.now)
			case "identity_inactive":
				f.exec(`UPDATE node_incarnations SET revoked_at=$2 WHERE id=$1`, f.n.IncarnationID, f.now)
			case "members_revoked":
				f.exec(`UPDATE members SET status='suspended',resource_version=resource_version+1 WHERE id=$1`, member)
			}
			f.now = f.now.Add(time.Minute)
			if err := f.svc.ReconcileGenerationHolds(t.Context(), uuid.Nil); err != nil {
				t.Fatal(err)
			}
			var reason string
			if err := f.pool.QueryRow(t.Context(), `SELECT end_reason FROM node_generation_holds WHERE ended_at IS NOT NULL`).Scan(&reason); err != nil || reason != ending {
				t.Fatalf("reason=%q err=%v", reason, err)
			}
			f.counts(1, 1, 0)
			if ending != "identity_inactive" {
				// A delayed report from before closure must not resurrect the incident.
				old.ReportId = uuid.New()
				f.report = old
				f.submit()
				f.counts(1, 1, 0)
			}
		})
	}
}
