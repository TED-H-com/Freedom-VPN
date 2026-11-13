//go:build integration

package nodes_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// D-167 compile fence: the two-int namespace is disjoint from hashed bigint
// advisory locks. Literal SQL lets these tests run before the implementation.
const d167FenceNamespace = 0x4656504e // "FVPN"
const d167FencePurpose = 167

func d167Shared(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared($1::int,$2::int)`, d167FenceNamespace, d167FencePurpose); err != nil {
		t.Fatal(err)
	}
}
func d167Replace(t *testing.T, w *d165World) api.ObservationReport {
	t.Helper()
	old := w.reports[0]
	detail, _, err := w.r.Nodes.GetNode(t.Context(), old.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	cmd := enrollment.Command{Principal: "admin:" + w.admin.String(), Key: uuid.New()}
	if _, err = w.r.Enrollment.RevokeNode(t.Context(), cmd, old.NodeId, &detail.ResourceVersion, "TEST ONLY D167 replacement"); err != nil {
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
	r := cloneReport(old, old.NodeId, replacement.Response.NodeIncarnationId)
	var count int
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM desired_states WHERE node_id=$1 AND incarnation_id=$2`, r.NodeId, r.NodeIncarnationId).Scan(&count); err != nil || count != 0 {
		t.Fatal("replacement already has issued state", count, err)
	}
	w.reports[0] = r
	return r
}
func d167Disable(t *testing.T, w *d165World) {
	t.Helper()
	_, etag, err := w.r.Nodes.GetNode(t.Context(), w.reports[0].NodeId)
	if err != nil {
		t.Fatal(err)
	}
	groups := []uuid.UUID{}
	_, _, err = w.exits.SetPublication(t.Context(), exits.SetPublicationParams{NodeID: w.reports[0].NodeId, Principal: "admin:" + w.admin.String(), IdempotencyKey: uuid.New(), IfMatch: etag, Request: api.PublicationRequest{PublicationIntent: api.PublicationIntentDisabled, GroupIds: &groups, Reason: "TEST ONLY D167 detached carrier"}})
	if err != nil {
		t.Fatal(err)
	}
}
func d167NonemptyGroups(t *testing.T, w *d165World, e db.ClaimedEvent) {
	t.Helper()
	var nonempty bool
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT jsonb_array_length(payload->'exit_group_ids')>0 FROM outbox_events WHERE id=$1`, e.ID).Scan(&nonempty); err != nil || !nonempty {
		t.Fatal("test requires the real nonempty entitlement groups", nonempty, err)
	}
}
func d167Worker(ctx context.Context, w *d165World, event db.ClaimedEvent) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := worker.MemberJobs()[event.EventType](worker.WithMembers(ctx, w.services), event)
		done <- err
	}()
	return done
}
func d167WaitBlock(ctx context.Context, w *d165World, pid uint32) (uint32, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		var waiter uint32
		err := w.r.Pool.QueryRow(ctx, `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND $1::int=ANY(pg_blocking_pids(pid)) ORDER BY pid LIMIT 1`, pid).Scan(&waiter)
		if err == nil {
			return waiter, nil
		}
		if err != pgx.ErrNoRows {
			return 0, err
		}
		select {
		case <-time.After(5 * time.Millisecond):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}
func d167FenceWait(t *testing.T, ctx context.Context, w *d165World, pid uint32) {
	t.Helper()
	var waiting bool
	err := w.r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND classid=$2::oid AND objid=$3::oid AND objsubid=2 AND NOT granted)`, pid, d167FenceNamespace, d167FencePurpose).Scan(&waiting)
	if err != nil || !waiting {
		t.Fatalf("D167 expected transaction %d to wait on compile fence, waiting=%v: %v", pid, waiting, err)
	}
}
func d167NoDeadlocks(t *testing.T, ctx context.Context, w *d165World) {
	t.Helper()
	for _, conn := range w.r.Pool.AcquireAllIdle(ctx) {
		_, err := conn.Exec(ctx, `SELECT pg_stat_force_next_flush()`)
		conn.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := w.r.Pool.QueryRow(ctx, `SELECT deadlocks FROM pg_stat_database WHERE datname=current_database()`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("D167 database deadlocks=%d: %v", count, err)
	}
}

type d167AfterCompile struct {
	compiler membersruntime.Compiler
	after    func(context.Context) error
}

func (c d167AfterCompile) RecompileBatch(ctx context.Context, events []db.ClaimedEvent) ([]desiredstate.CompiledDesiredState, error) {
	out, err := c.compiler.RecompileBatch(ctx, events)
	if err == nil {
		err = c.after(ctx)
	}
	return out, err
}

func TestD167_T19_ReplacementFormerCarrierSelected(t *testing.T) {
	w := newD165World(t, 1)
	op := d165Suspend(t, w, true)
	d167Disable(t, w)
	r := d167Replace(t, w)
	r = w.report(r, "clean")
	r.AppliedConfigGeneration.Set(0)
	r.ProvisioningState = api.ProvisioningStateApplying
	w.observe(r)
	event := d165RemovalEvent(t, w, "member.entitlement_changed", w.member)
	d167NonemptyGroups(t, w, event)
	d165RunEvent(t, w, event)
	w.wantStep(op, false)
	var gen decstr.U64
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT generation::text FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, op, r.NodeId).Scan(&gen); err != nil {
		t.Fatal("replacement carrier missing", err)
	}
	w.observe(d165ReportAt(w, gen, "clean"))
	w.wantStep(op, true)
}

func TestD167_T20_CaptureWaitsForInflightRemoval(t *testing.T) {
	w := newD165World(t, 1)
	op := d165Suspend(t, w, true)
	event := d165RemovalEvent(t, w, "member.entitlement_changed", w.member)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	writer, err := w.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Rollback(context.Background()) }()
	var g2, g3 decstr.U64
	ready := make(chan error, 1)
	w.services.Compiler = d167AfterCompile{compiler: w.compiler, after: func(ctx context.Context) error {
		// Retire the first removal, then create a higher one in a different real
		// transaction after the batch. No visible removal exists at final capture.
		w.observe(w.report(w.reports[0], "clean"))
		d165Suspend(t, w, false)
		back, e := w.compiler.Compile(ctx, w.reports[0].NodeId)
		if e != nil {
			return e
		}
		g2 = decstr.U64(back.Generation)
		d165Suspend(t, w, true)
		d167Shared(t, ctx, writer)
		removed, e := w.compiler.CompileTx(ctx, writer, w.reports[0].NodeId)
		if e != nil {
			return e
		}
		g3 = decstr.U64(removed.Generation)
		// On the old code this row lock only delays completion after the missed
		// removal reads. On D167 the capture must wait earlier, on the fence.
		_, e = writer.Exec(ctx, `SELECT id FROM operations WHERE id=$1 FOR UPDATE`, op)
		ready <- e
		return e
	}}
	done := d167Worker(ctx, w, event)
	select {
	case err = <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case err = <-done:
		t.Fatal("worker before interleaving", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	waiter, err := d167WaitBlock(ctx, w, writer.Conn().PgConn().PID())
	if err != nil {
		t.Fatal("capture did not wait for compile", err)
	}
	d167FenceWait(t, ctx, w, waiter)
	if err = writer.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	d165Pin(t, w, op, g3, false)
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, g2, "clean"))
	w.wantStep(op, false)
	w.observe(d165ReportAt(w, g3, "clean"))
	w.wantStep(op, true)
	d167NoDeadlocks(t, ctx, w)
}

func TestD167_T21_CompileWaitsOutsideCaptureAndObservation(t *testing.T) {
	w := newD165World(t, 1)
	op := d165Suspend(t, w, true)
	event := d165RemovalEvent(t, w, "member.entitlement_changed", w.member)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	blocker, err := w.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	var later uuid.UUID
	var g2 decstr.U64
	ready := make(chan error, 1)
	w.services.Compiler = d167AfterCompile{compiler: w.compiler, after: func(ctx context.Context) error {
		d165Suspend(t, w, false)
		back, e := w.compiler.Compile(ctx, w.reports[0].NodeId)
		if e != nil {
			return e
		}
		g2 = decstr.U64(back.Generation)
		later = d165Suspend(t, w, true)
		_, e = blocker.Exec(ctx, `SELECT id FROM operations WHERE id=$1 FOR UPDATE`, op)
		ready <- e
		return e
	}}
	captured := d167Worker(ctx, w, event)
	select {
	case err = <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case err = <-captured:
		t.Fatal("worker before interleaving", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	capturePID, err := d167WaitBlock(ctx, w, blocker.Conn().PgConn().PID())
	if err != nil {
		t.Fatal(err)
	}
	writer, err := w.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Rollback(context.Background()) }()
	compiled := make(chan error, 1)
	var g3 decstr.U64
	go func() {
		_, e := writer.Exec(ctx, `SELECT pg_advisory_xact_lock_shared($1::int,$2::int)`, d167FenceNamespace, d167FencePurpose)
		if e == nil {
			var out desiredstate.CompiledDesiredState
			out, e = w.compiler.CompileTx(ctx, writer, w.reports[0].NodeId)
			g3 = decstr.U64(out.Generation)
		}
		if e == nil {
			e = writer.Commit(ctx)
		}
		compiled <- e
	}()
	waiter, err := d167WaitBlock(ctx, w, capturePID)
	if err != nil {
		t.Fatal("compile did not wait on capture", err)
	}
	d167FenceWait(t, ctx, w, waiter)
	report := d165ReportAt(w, g2, "clean")
	observed := make(chan error, 1)
	go func() {
		_, e := w.r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
		observed <- e
	}()
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for _, done := range []<-chan error{captured, compiled, observed} {
		if err = <-done; err != nil {
			t.Fatal(err)
		}
	}
	w.wantStep(op, true)
	d165RemovalCount(t, w, "member", w.member, 1)
	w.services.Compiler = w.compiler
	d165RunEvent(t, w, d165RemovalEvent(t, w, "member.entitlement_changed", w.member))
	d165Pin(t, w, later, g3, false)
	w.wantStep(later, false)
	w.observe(d165ReportAt(w, g3, "clean"))
	w.wantStep(later, true)
	d167NoDeadlocks(t, ctx, w)
}

func TestD167_T22_BarrierWaitsForPrechangeCompile(t *testing.T) {
	w := newD165World(t, 1)
	unrelated := w.enroll()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	writer, err := w.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Rollback(context.Background()) }()
	d167Shared(t, ctx, writer)
	if _, err = w.compiler.CompileTx(ctx, writer, unrelated.NodeId); err != nil {
		t.Fatal(err)
	}
	// A serving member read takes FOR SHARE, which prevents a simultaneous real
	// suspend from committing. This unrelated real compile has no such row
	// overlap: only the database-wide barrier can make the worker wait for it.
	op := d165Suspend(t, w, true)
	event := d165RemovalEvent(t, w, "member.entitlement_changed", w.member)
	d167NonemptyGroups(t, w, event)
	done := d167Worker(ctx, w, event)
	waiter, err := d167WaitBlock(ctx, w, writer.Conn().PgConn().PID())
	if err != nil {
		t.Fatal("D167 worker crossed pre-selection barrier before pre-change compile committed", err)
	}
	d167FenceWait(t, ctx, w, waiter)
	var captured bool
	if err = w.r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM node_operation_captures WHERE operation_id=$1)`, op).Scan(&captured); err != nil || captured {
		t.Fatal("barrier must precede capture linking", captured, err)
	}
	if err = writer.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	w.wantStep(op, false)
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(op, true)
}

func TestD167_T23_UnissuedReplacementCannotAcknowledgeAddition(t *testing.T) {
	w := newD165World(t, 1, false)
	var pin decstr.U64
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT generation::text FROM node_operation_targets WHERE operation_id=$1`, w.addition).Scan(&pin); err != nil {
		t.Fatal(err)
	}
	d167Replace(t, w)
	w.observe(d165ReportAt(w, pin, "clean"))
	w.wantStep(w.addition, false)
	w.target(w.addition, w.reports[0].NodeId, "pending")
	fresh, _ := w.prepare(w.reports[0], false, false)
	w.observe(fresh)
	w.wantStep(w.addition, true)
}
func TestD167_T24_UnissuedReplacementCannotRetireRemoval(t *testing.T) {
	w := newD165RemovalWorld(t)
	op := w.revokeDevice()
	pin := d165ControlCompile(t, w, false)
	d167Replace(t, w)
	w.observe(d165ReportAt(w, pin, "clean"))
	d165RemovalCount(t, w, "device", w.device, 1)
	fresh, _ := w.prepare(w.reports[0], false, false)
	w.observe(fresh)
	d165RemovalCount(t, w, "device", w.device, 0)
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	d165EmptyCapture(t, w, op)
}

type d167FailAuthorization struct{ devices.Adapter }

func (a d167FailAuthorization) SetAuthorization(context.Context, string, string, bool, []uuid.UUID) error {
	return errors.New("TEST ONLY transient authorization failure")
}
func TestD167_T25_MemberRetryFinishesCapture(t *testing.T) {
	w := newD165World(t, 1)
	out, err := w.services.HTTP.Members.RevokeMemberGrant(t.Context(), d165Command("admin", w.admin), w.member, w.grant, "TEST ONLY D167 retry")
	if err != nil {
		t.Fatal(err)
	}
	event := d165RemovalEvent(t, w, "member.entitlement_changed", w.member)
	working := w.services.HTTP.Devices.Adapter
	w.services.HTTP.Devices.Adapter = d167FailAuthorization{working}
	if _, err = worker.MemberJobs()[event.EventType](worker.WithMembers(t.Context(), w.services), event); err == nil {
		t.Fatal("missing injected failure")
	}
	if op := w.operation(*out.OperationID); op.Status != api.OperationStatusFailed {
		t.Fatal("expected failed operation before retry", op.Status)
	}
	w.services.HTTP.Devices.Adapter = working
	d165RunEvent(t, w, event)
	w.observe(w.report(w.reports[0], "clean"))
	if _, err = w.r.Nodes.SweepStaleNodes(t.Context(), 0, 0); err != nil {
		t.Fatal(err)
	}
	var finished bool
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT captured_at IS NOT NULL FROM node_operation_captures WHERE operation_id=$1`, out.OperationID).Scan(&finished); err != nil || !finished {
		t.Fatalf("D167 retry left capture unfinished: finished=%v %v", finished, err)
	}
	w.wantStep(*out.OperationID, true)
}
func TestD167_T26_DatabaseReceiptClockCountsPostcompileAck(t *testing.T) {
	w := newD165World(t, 1)
	op := w.revokeDevice()
	w.services.Compiler = d167AfterCompile{compiler: w.compiler, after: func(ctx context.Context) error {
		var databaseNow time.Time
		if err := w.r.Pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&databaseNow); err != nil {
			return err
		}
		skew := databaseNow.Add(-time.Second)
		w.r.Nodes = nodes.NewService(w.r.Pool, nodes.WithClock(func() time.Time { return skew }))
		report := w.report(w.reports[0], "clean")
		accepted, err := w.r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
		if err != nil {
			return err
		}
		if !accepted.ReceivedAt.Before(databaseNow) {
			return fmt.Errorf("service clock fixture did not trail database")
		}
		return nil
	}}
	d165RunEvent(t, w, d165RemovalEvent(t, w, "device.revoked", w.device))
	w.target(op, w.reports[0].NodeId, "acknowledged")
	d165RemovalCount(t, w, "device", w.device, 0)
	w.wantStep(op, true)
}
func TestD167_T27_CompileTxRequiresFence(t *testing.T) {
	w := newD165World(t, 1)
	tx, err := w.r.Pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err = w.compiler.CompileTx(t.Context(), tx, w.reports[0].NodeId)
	if err == nil || !strings.Contains(err.Error(), "compile fence") {
		t.Fatalf("D167 CompileTx without compile fence must fail, got %v", err)
	}
}
