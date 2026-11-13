//go:build integration

package nodes_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func newD165RemovalWorld(t *testing.T) *d165World {
	t.Helper()
	w := newD165World(t, 1)
	rf := releasetest.New(t)
	w.r.Releases = &releases.Service{Pool: w.r.Pool, Validator: rf.Validator}
	if _, err := w.r.Releases.Register(t.Context(), releases.Command{ActorID: w.admin, IdempotencyKey: uuid.New()}, rf.Request(t, rf.Manifest)); err != nil {
		t.Fatal(err)
	}
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	raw, err := json.Marshal(support)
	if err != nil {
		t.Fatal(err)
	}
	store := policies.NewStore(w.r.Pool, policies.WithReleaseService(w.r.Releases))
	code := "test-only-d165-support"
	revision, err := store.CreateRevision(t.Context(), policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeReleaseSupport, SchemaVersion: 1, Reason: "TEST ONLY", ActorID: w.admin, Payload: raw})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: revision.DefinitionID, RevisionID: revision.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: w.admin}); err != nil {
		t.Fatal(err)
	}
	return w
}

func d165Suspend(t *testing.T, w *d165World, suspended bool) uuid.UUID {
	t.Helper()
	var version decstr.U64
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT resource_version::text FROM members WHERE id=$1`, w.member).Scan(&version); err != nil {
		t.Fatal(err)
	}
	reason := "TEST ONLY removal race"
	out, err := w.services.HTTP.Members.Patch(t.Context(), d165Command("admin", w.admin), w.member, &version, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason})
	if err != nil {
		t.Fatal(err)
	}
	if out.OperationID == nil {
		t.Fatal("missing suspension operation")
	}
	return *out.OperationID
}

// Claim real events, including those another fixture dispatch already claimed.
func d165RemovalEvent(t *testing.T, w *d165World, kind string, aggregate uuid.UUID) db.ClaimedEvent {
	t.Helper()
	if _, err := db.ClaimOutbox(t.Context(), w.r.Pool, "TEST ONLY removal", 1000, time.Hour); err != nil {
		t.Fatal(err)
	}
	var e db.ClaimedEvent
	err := w.r.Pool.QueryRow(t.Context(), `SELECT id,aggregate_id,aggregate_type,aggregate_version::text,event_type,schema_version,payload,lease_owner,lease_token FROM outbox_events WHERE event_type=$1 AND aggregate_id=$2 ORDER BY created_at DESC,id DESC LIMIT 1`, kind, aggregate).Scan(&e.ID, &e.AggregateID, &e.AggregateType, &e.AggregateVersion, &e.EventType, &e.SchemaVersion, &e.Payload, &e.Worker, &e.LeaseToken)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func d165RunEvent(t *testing.T, w *d165World, e db.ClaimedEvent) {
	t.Helper()
	if _, err := worker.MemberJobs()[e.EventType](worker.WithMembers(t.Context(), w.services), e); err != nil {
		t.Fatal(err)
	}
}
func d165ControlCompile(t *testing.T, w *d165World, present bool) decstr.U64 {
	t.Helper()
	r := w.reports[0]
	out, err := w.r.CompileControlState(t.Context(), r.NodeId, r.NodeIncarnationId)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range out.Envelope.Signed.DeviceBindings {
		if b.DeviceId == w.device {
			found = true
		}
	}
	if found != present {
		t.Fatalf("device present=%v, want %v", found, present)
	}
	return decstr.U64(out.Generation)
}
func d165Pin(t *testing.T, w *d165World, op uuid.UUID, generation decstr.U64, present bool) {
	t.Helper()
	var gen decstr.U64
	var got bool
	err := w.r.Pool.QueryRow(t.Context(), `SELECT generation::text,subject_present FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, op, w.reports[0].NodeId).Scan(&gen, &got)
	if err != nil || gen != generation || got != present {
		t.Fatalf("D165 removal target pin=%s present=%v, want %s/%v: %v", gen, got, generation, present, err)
	}
}
func d165RemovalCount(t *testing.T, w *d165World, kind string, subject uuid.UUID, want int) {
	t.Helper()
	var count int
	err := w.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_subject_removals WHERE node_id=$1 AND subject_type=$2 AND subject_id=$3`, w.reports[0].NodeId, kind, subject).Scan(&count)
	if err != nil || count != want {
		t.Fatalf("D165 outstanding removals=%d, want %d: %v", count, want, err)
	}
}
func d165ReportAt(w *d165World, generation decstr.U64, mode string) api.ObservationReport {
	r := w.report(w.reports[0], mode)
	r.AppliedConfigGeneration.Set(generation)
	return r
}
func d165EmptyCapture(t *testing.T, w *d165World, op uuid.UUID) {
	t.Helper()
	w.wantStep(op, true)
	var empty bool
	err := w.r.Pool.QueryRow(t.Context(), `SELECT c.outcomes='[]'::jsonb AND c.captured_at IS NOT NULL AND c.completed_at IS NOT NULL AND o.observed_refs->'nodes'->>'reference'='node_operation_capture:'||c.id::text AND NOT EXISTS(SELECT 1 FROM node_operation_targets t WHERE t.operation_id=o.id) FROM node_operation_captures c JOIN operations o ON o.id=c.operation_id WHERE o.id=$1`, op).Scan(&empty)
	if err != nil || !empty {
		t.Fatal("D165 expected empty capture evidence", empty, err)
	}
}

func TestD165_T12_ControlCompileRemovalBeforeDeviceCapture(t *testing.T) {
	w := newD165RemovalWorld(t)
	old, _ := w.reports[0].AppliedConfigGeneration.Get()
	op := w.revokeDevice()
	gen := d165ControlCompile(t, w, false)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	d165Pin(t, w, op, gen, false)
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, old, "clean"))
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, gen, "clean"))
	w.wantStep(op, true)
	w.target(op, w.reports[0].NodeId, "acknowledged")
}

// A real grant for another member selects the same exit. Its whole-node compile
// removes the suspended member before that member's own event is handled.
func d165OtherMemberBatch(t *testing.T, w *d165World) decstr.U64 {
	t.Helper()
	p, err := (auth.PostgresStore{Pool: w.r.Pool}).Login(t.Context(), auth.Identity{Issuer: "https://id.example.invalid", ClientScope: "portal", Subject: "TEST-ONLY-other-" + uuid.NewString(), Role: auth.Member, Label: "TEST ONLY", AuthTime: time.Now().UTC().Add(-time.Minute), Expires: w.now.Add(time.Hour)}, uuid.NewString(), "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	var revision uuid.UUID
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT plan_revision_id FROM entitlement_grants WHERE id=$1`, w.grant).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	start := w.now.Add(-time.Hour)
	if _, err = w.services.HTTP.Members.Grant(t.Context(), d165Command("admin", w.admin), p.ID, api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: w.now.Add(12 * time.Hour), PlanRevisionId: &revision, Reason: "TEST ONLY other member batch"}); err != nil {
		t.Fatal(err)
	}
	d165RunEvent(t, w, d165RemovalEvent(t, w, "member.entitlement_changed", p.ID))
	var gen decstr.U64
	var absent bool
	err = w.r.Pool.QueryRow(t.Context(), `SELECT generation::text,NOT EXISTS(SELECT 1 FROM jsonb_array_elements(envelope->'signed'->'device_bindings') b WHERE b->>'member_id'=$2) AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(envelope->'signed'->'member_rate_policies') b WHERE b->>'member_id'=$2) FROM desired_state_envelopes WHERE node_id=$1 AND incarnation_id=$3`, w.reports[0].NodeId, w.member.String(), w.reports[0].NodeIncarnationId).Scan(&gen, &absent)
	if err != nil || !absent {
		t.Fatal("other member batch did not remove subject", absent, err)
	}
	return gen
}
func TestD165_T13_OtherMemberBatchRemovalBeforeSuspendCapture(t *testing.T) {
	w := newD165RemovalWorld(t)
	old, _ := w.reports[0].AppliedConfigGeneration.Get()
	op := d165Suspend(t, w, true)
	gen := d165OtherMemberBatch(t, w)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "member.suspended", w.member))
	d165Pin(t, w, op, gen, false)
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, old, "clean"))
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, gen, "clean"))
	w.wantStep(op, true)
}
func TestD165_T14_RemovalAcknowledgedBeforeCapture(t *testing.T) {
	w := newD165RemovalWorld(t)
	op := w.revokeDevice()
	gen := d165ControlCompile(t, w, false)
	d165RemovalCount(t, w, "device", w.device, 1)
	w.observe(d165ReportAt(w, gen, "clean"))
	d165RemovalCount(t, w, "device", w.device, 0)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	d165EmptyCapture(t, w, op)
}
func TestD165_T15_LatestRemovalWins(t *testing.T) {
	w := newD165RemovalWorld(t)
	d165Suspend(t, w, true)
	g1 := d165ControlCompile(t, w, false)
	d165Suspend(t, w, false)
	g2 := d165ControlCompile(t, w, true)
	op := d165Suspend(t, w, true)
	g3 := d165ControlCompile(t, w, false)
	if !(g1 < g2 && g2 < g3) {
		t.Fatal("non-increasing generations", g1, g2, g3)
	}
	d165RunEvent(t, w, d165RemovalEvent(t, w, "member.suspended", w.member))
	d165Pin(t, w, op, g3, false)
	for _, gen := range []decstr.U64{g1, g2} {
		w.observe(d165ReportAt(w, gen, "clean"))
		w.wantStep(op, false)
	}
	w.observe(d165ReportAt(w, g3, "clean"))
	w.wantStep(op, true)
}
func TestD165_T16_AdditionPinDominatesOlderRemoval(t *testing.T) {
	w := newD165RemovalWorld(t)
	d165Suspend(t, w, true)
	g1 := d165ControlCompile(t, w, false)
	d165RemovalCount(t, w, "member", w.member, 1)
	op := d165Suspend(t, w, false)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "member.entitlement_changed", w.member))
	r := w.report(w.reports[0], "clean")
	g2, _ := r.AppliedConfigGeneration.Get()
	if g2 <= g1 {
		t.Fatal("addition did not raise generation", g1, g2)
	}
	d165Pin(t, w, op, g2, true)
	w.observe(d165ReportAt(w, g2, "closed"))
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, g1, "clean"))
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, g2, "clean"))
	w.wantStep(op, true)
}
func TestD165_T17_AcknowledgedRemovalDoesNotTargetLaterOperation(t *testing.T) {
	w := newD165RemovalWorld(t)
	w.revokeDevice()
	gen := d165ControlCompile(t, w, false)
	d165RemovalCount(t, w, "device", w.device, 1)
	w.observe(d165ReportAt(w, gen, "clean"))
	d165RemovalCount(t, w, "device", w.device, 0)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	// A new idempotency key on an already-revoked device creates a real new op.
	later := w.revokeDevice()
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	d165EmptyCapture(t, w, later)
}

// Observe PostgreSQL's actual wait graph, not a timing assumption. Only sessions
// in this test's isolated database can satisfy the barrier.
func d165BlockedBy(t *testing.T, w *d165World, pid uint32) uint32 {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for {
		var waiter uint32
		err := w.r.Pool.QueryRow(ctx, `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND $1::int=ANY(pg_blocking_pids(pid)) ORDER BY pid LIMIT 1`, pid).Scan(&waiter)
		if err == nil {
			return waiter
		}
		if err != pgx.ErrNoRows {
			t.Fatal("wait graph", err)
		}
		select {
		case <-time.After(5 * time.Millisecond):
		case <-ctx.Done():
			t.Fatal("expected database lock wait", ctx.Err())
		}
	}
}
func TestD165_T18_RemovalAckCaptureInterleavings(t *testing.T) {
	for _, first := range []string{"observation", "capture"} {
		t.Run(first, func(t *testing.T) {
			w := newD165RemovalWorld(t)
			op := w.revokeDevice()
			gen := d165ControlCompile(t, w, false)
			d165RemovalCount(t, w, "device", w.device, 1)
			event := d165RemovalEvent(t, w, "device.revoked", w.device)
			report := d165ReportAt(w, gen, "clean")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			blocker, err := w.r.Pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = blocker.Rollback(context.Background()) }()
			compiled := make(chan error, 1)
			var once sync.Once
			w.services.Compiler = d165AfterCompile{compiler: w.compiler, after: func() {
				once.Do(func() {
					var lockErr error
					if first == "observation" {
						_, lockErr = blocker.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, report.NodeId)
					} else {
						_, lockErr = blocker.Exec(ctx, `SELECT id FROM operations WHERE id=$1 FOR UPDATE`, op)
					}
					compiled <- lockErr
				})
			}}
			done := make(chan error, 1)
			go func() {
				_, e := worker.MemberJobs()[event.EventType](worker.WithMembers(ctx, w.services), event)
				done <- e
			}()
			if err = <-compiled; err != nil {
				t.Fatal(err)
			}
			capturePID := d165BlockedBy(t, w, blocker.Conn().PgConn().PID())
			if first == "observation" {
				if _, err = w.r.Nodes.SubmitObservationTx(ctx, blocker, report.NodeId, report.NodeIncarnationId, report); err != nil {
					t.Fatal(err)
				}
			}
			var observed chan error
			if first == "capture" {
				observed = make(chan error, 1)
				go func() {
					_, e := w.r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
					observed <- e
				}()
				d165BlockedBy(t, w, capturePID)
			}
			if err = blocker.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err = <-done; err != nil {
				t.Fatal("capture", err)
			}
			if observed != nil {
				if err = <-observed; err != nil {
					t.Fatal("observation", err)
				}
			}
			w.wantStep(op, true)
			d165RemovalCount(t, w, "device", w.device, 0)
			if first == "observation" {
				d165EmptyCapture(t, w, op)
			} else {
				w.target(op, report.NodeId, "acknowledged")
			}
		})
	}
}

func TestD165_RemovalRetirementAndReplacement(t *testing.T) {
	for _, mode := range []string{"failed_closed", "retired", "replacement"} {
		t.Run(mode, func(t *testing.T) {
			w := newD165RemovalWorld(t)
			op := w.revokeDevice()
			gen := d165ControlCompile(t, w, false)
			d165RemovalCount(t, w, "device", w.device, 1)
			if mode == "failed_closed" {
				w.observe(d165ReportAt(w, gen, "pending"))
				d165RemovalCount(t, w, "device", w.device, 1)
				w.observe(d165ReportAt(w, gen, "closed"))
			} else {
				r := w.reports[0]
				detail, _, err := w.r.Nodes.GetNode(t.Context(), r.NodeId)
				if err != nil {
					t.Fatal(err)
				}
				cmd := enrollment.Command{Principal: "admin:" + w.admin.String(), Key: uuid.New()}
				if mode == "retired" {
					_, err = w.r.Enrollment.RetireNode(t.Context(), cmd, r.NodeId, &detail.ResourceVersion, "TEST ONLY removal retirement")
				} else {
					_, err = w.r.Enrollment.RevokeNode(t.Context(), cmd, r.NodeId, &detail.ResourceVersion, "TEST ONLY replacement")
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err = w.r.Nodes.SweepStaleNodes(t.Context(), 0, 0); err != nil {
					t.Fatal(err)
				}
				if mode == "replacement" {
					d165RemovalCount(t, w, "device", w.device, 1)
					created, err := w.r.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: cmd.Principal, Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &r.NodeId})
					if err != nil {
						t.Fatal(err)
					}
					token, _ := created.Token.Get()
					request, _ := pkitest.Request(t)
					replacement, err := w.r.Enrollment.Enroll(t.Context(), token, request)
					if err != nil {
						t.Fatal(err)
					}
					fresh := cloneReport(r, r.NodeId, replacement.Response.NodeIncarnationId)
					fresh, _ = w.prepare(fresh, false, false)
					w.observe(fresh)
				}
			}
			d165RemovalCount(t, w, "device", w.device, 0)
			d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
			d165EmptyCapture(t, w, op)
		})
	}
}

func TestD165_FirstReplacementCompileRecordsPriorCarrierRemoval(t *testing.T) {
	w := newD165RemovalWorld(t)
	op := w.revokeDevice()
	old := w.reports[0]
	detail, _, err := w.r.Nodes.GetNode(t.Context(), old.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	cmd := enrollment.Command{Principal: "admin:" + w.admin.String(), Key: uuid.New()}
	if _, err = w.r.Enrollment.RevokeNode(t.Context(), cmd, old.NodeId, &detail.ResourceVersion, "TEST ONLY replacement before removal compile"); err != nil {
		t.Fatal(err)
	}
	created, err := w.r.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: cmd.Principal, Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &old.NodeId})
	if err != nil {
		t.Fatal(err)
	}
	token, _ := created.Token.Get()
	request, _ := pkitest.Request(t)
	replacement, err := w.r.Enrollment.Enroll(t.Context(), token, request)
	if err != nil {
		t.Fatal(err)
	}
	fresh := cloneReport(old, old.NodeId, replacement.Response.NodeIncarnationId)
	fresh, _ = w.prepare(fresh, false, false)
	d165RemovalCount(t, w, "device", w.device, 1)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	gen, _ := fresh.AppliedConfigGeneration.Get()
	d165Pin(t, w, op, gen, false)
	w.wantStep(op, false)
	w.observe(fresh)
	w.wantStep(op, true)
	d165RemovalCount(t, w, "device", w.device, 0)
}
