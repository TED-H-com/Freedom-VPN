//go:build integration

package nodes_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Publish through the DNS owner, including its real activation outbox event.
func d170Publish(t *testing.T, w *d165World) nodes.ResolverDocument {
	t.Helper()
	var definition, revision uuid.UUID
	var version string
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT p.id,r.id,p.resource_version::text FROM policy_definitions p JOIN policy_revisions r ON r.definition_id=p.id WHERE p.code='d165-dns_profile' ORDER BY r.revision DESC LIMIT 1`).Scan(&definition, &revision, &version); err != nil {
		t.Fatal(err)
	}
	if _, err := policies.NewStore(w.r.Pool).ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: definition, RevisionID: revision, ExpectedVersion: version, Reason: "TEST ONLY D170", ActorID: w.admin}); err != nil {
		t.Fatal(err)
	}
	pub, err := dnsstore.NewPublisher(w.r.Pool, dnsstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	var event db.ClaimedEvent
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT event_type,payload FROM outbox_events WHERE event_type='dns.publish' AND payload->>'revision_id'=$1`, revision.String()).Scan(&event.EventType, &event.Payload); err != nil {
		t.Fatal(err)
	}
	if _, err = pub.Handle(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	document, err := w.r.ResolverDocuments.LatestPublished(t.Context(), definition)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func d170Pool(t *testing.T, w *d165World, size int32) *pgxpool.Pool {
	t.Helper()
	cfg := w.r.Pool.Config()
	cfg.MaxConns = size
	p, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func d170Runtime(t *testing.T, w *d165World, pool *pgxpool.Pool) *httpwire.Runtime {
	t.Helper()
	r, err := httpwire.Open(t.Context(), pool, w.r.Config, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Releases.Validator = w.r.Releases.Validator
	r.Releases.VerifyArtifacts = w.r.Releases.VerifyArtifacts
	r.Nodes = nodes.NewService(pool, nodes.WithClock(func() time.Time { return w.now }))
	return r
}

func TestD170_T28_SingleConnectionCompileEntrypoints(t *testing.T) {
	for _, entry := range []string{"Compile", "ReSign", "ReSign_fallback", "CompileControlState", "resolver_reconcile"} {
		t.Run(entry, func(t *testing.T) {
			w := newD165RemovalWorld(t)
			// Reconciliation needs an existing envelope with the previous DNS reference.
			report := w.reports[0]
			if entry == "ReSign_fallback" {
				report = w.enroll()
			}
			document := d170Publish(t, w)
			if entry == "ReSign" {
				if _, err := w.r.DesiredStateCompiler().Compile(t.Context(), report.NodeId); err != nil {
					t.Fatal(err)
				}
			}
			pool := d170Pool(t, w, 1)
			runtime := d170Runtime(t, w, pool)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			var err error
			switch entry {
			case "Compile":
				_, err = runtime.DesiredStateCompiler().Compile(ctx, report.NodeId)
			case "ReSign", "ReSign_fallback":
				_, err = runtime.DesiredStateCompiler().ReSign(ctx, report.NodeId)
			case "CompileControlState":
				_, err = runtime.CompileControlState(ctx, report.NodeId, report.NodeIncarnationId)
			case "resolver_reconcile":
				err = runtime.ReconcileResolverDocuments(ctx)
			}
			if err != nil {
				t.Fatalf("D170 %s must complete using its single connection without a nested pool wait: %v", entry, err)
			}
			var got api.DesiredStateEnvelope
			if err = w.r.Pool.QueryRow(t.Context(), `SELECT envelope FROM desired_state_envelopes WHERE node_id=$1 AND incarnation_id=$2`, report.NodeId, report.NodeIncarnationId).Scan(&got); err != nil {
				t.Fatal(err)
			}
			ref, err := got.Signed.Dns.ResolverDocument.Get()
			if err != nil || ref != document.Reference {
				t.Fatal("D170 published resolver document was not issued", ref, err)
			}
		})
	}
}

type d170ResolverPause struct {
	nodes.ResolverDocumentSource
	entered, resume chan struct{}
	once            sync.Once
}

func (s *d170ResolverPause) pause(ctx context.Context) {
	s.once.Do(func() {
		close(s.entered)
		select {
		case <-s.resume:
		case <-ctx.Done():
		}
	})
}
func (s *d170ResolverPause) LatestPublished(ctx context.Context, id uuid.UUID) (nodes.ResolverDocument, error) {
	s.pause(ctx)
	return s.ResolverDocumentSource.LatestPublished(ctx, id)
}
func (s *d170ResolverPause) LatestPublishedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (nodes.ResolverDocument, error) {
	s.pause(ctx)
	// The local interface lets the regression compile on the pre-D170 base.
	return s.ResolverDocumentSource.(interface {
		LatestPublishedTx(context.Context, pgx.Tx, uuid.UUID) (nodes.ResolverDocument, error)
	}).LatestPublishedTx(ctx, tx, id)
}

func TestD170_T29_FullPoolCompileCaptureCompletes(t *testing.T) {
	w := newD165RemovalWorld(t)
	unrelated := w.enroll()
	op := w.revokeDevice()
	event := d165RemovalEvent(t, w, "device.revoked", w.device)
	d170Publish(t, w)
	pool := d170Pool(t, w, 10)
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	runtime := d170Runtime(t, w, pool)
	source := &d170ResolverPause{ResolverDocumentSource: runtime.ResolverDocuments, entered: make(chan struct{}), resume: make(chan struct{})}
	runtime.ResolverDocuments = source
	compiler := runtime.DesiredStateCompiler()
	first := make(chan error, 1)
	go func() { _, err := compiler.Compile(ctx, w.reports[0].NodeId); first <- err }()
	select {
	case <-source.entered:
	case err := <-first:
		t.Fatal("compile did not reach real resolver lookup", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var firstPID uint32
	if err := w.r.Pool.QueryRow(ctx, `SELECT pid FROM pg_locks WHERE locktype='advisory' AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND classid=$1::oid AND objid=$2::oid AND objsubid=2 AND mode='ShareLock' AND granted`, d167FenceNamespace, d167FencePurpose).Scan(&firstPID); err != nil {
		t.Fatal(err)
	}
	services := *w.services
	members := *services.HTTP.Members
	members.Pool = pool
	devices := *services.HTTP.Devices
	devices.Pool = pool
	billing := *services.HTTP.Billing
	billing.Pool = pool
	services.HTTP.Members = &members
	services.HTTP.Devices = &devices
	services.HTTP.Billing = &billing
	services.Compiler = compiler
	w.services = &services
	capture := d167Worker(ctx, w, event)
	capturePID, err := d167WaitBlock(ctx, w, firstPID)
	if err != nil {
		t.Fatal("worker did not wait on barrier", err)
	}
	d167FenceWait(t, ctx, w, capturePID)
	queued := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() { _, e := compiler.Compile(ctx, unrelated.NodeId); queued <- e }()
		deadline, done := context.WithTimeout(ctx, 5*time.Second)
		for {
			var n int
			err = w.r.Pool.QueryRow(deadline, `SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND classid=$1::oid AND objid=$2::oid AND objsubid=2 AND mode='ShareLock' AND NOT granted`, d167FenceNamespace, d167FencePurpose).Scan(&n)
			if err != nil {
				done()
				t.Fatal(err)
			}
			if n == i+1 {
				break
			}
			select {
			case <-time.After(5 * time.Millisecond):
			case <-deadline.Done():
				done()
				t.Fatal("later compile did not queue", deadline.Err())
			}
		}
		done()
	}
	if pool.Stat().AcquiredConns() != 10 || pool.Stat().IdleConns() != 0 {
		t.Fatal("D170 pool saturation schedule not established", pool.Stat().AcquiredConns(), pool.Stat().IdleConns())
	}
	t.Log("D170 schedule: ten connections occupied; one shared holder, one exclusive waiter, eight shared waiters")
	close(source.resume)
	// Drain every participant even on failure, so the isolated DB can be removed.
	var failures []error
	for i, done := range []<-chan error{first, capture, queued, queued, queued, queued, queued, queued, queued, queued} {
		if e := <-done; e != nil {
			failures = append(failures, fmt.Errorf("participant %d: %w", i, e))
		}
	}
	if len(failures) > 0 {
		t.Fatalf("D170 fenced lookup must not wait for another pool connection or fail capture with 55P03: %v", failures)
	}
	for _, conn := range pool.AcquireAllIdle(ctx) {
		_, err = conn.Exec(ctx, `SELECT pg_stat_force_next_flush()`)
		conn.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	d167NoDeadlocks(t, ctx, w)
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(op, true)
}

func TestD170_T30_ControlFencePrecedesFleetAndNode(t *testing.T) {
	w := newD165RemovalWorld(t)
	op := w.revokeDevice()
	event := d165RemovalEvent(t, w, "device.revoked", w.device)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	blocker, err := w.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	ready := make(chan error, 1)
	w.services.Compiler = d167AfterCompile{compiler: w.compiler, after: func(ctx context.Context) error {
		_, e := blocker.Exec(ctx, `SELECT id FROM operations WHERE id=$1 FOR UPDATE`, op)
		ready <- e
		return e
	}}
	capture := d167Worker(ctx, w, event)
	if err = <-ready; err != nil {
		t.Fatal(err)
	}
	capturePID, err := d167WaitBlock(ctx, w, blocker.Conn().PgConn().PID())
	if err != nil {
		t.Fatal(err)
	}
	control := make(chan error, 1)
	go func() {
		_, e := w.r.CompileControlState(ctx, w.reports[0].NodeId, w.reports[0].NodeIncarnationId)
		control <- e
	}()
	pid, err := d167WaitBlock(ctx, w, capturePID)
	if err != nil {
		t.Fatal(err)
	}
	var fence bool
	if err = w.r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND classid=$2::oid AND objid=$3::oid AND objsubid=2 AND NOT granted)`, pid, d167FenceNamespace, d167FencePurpose).Scan(&fence); err != nil || !fence {
		t.Fatalf("D170 control compile must wait on the compile fence before fleet/node locks: fence_wait=%v error=%v", fence, err)
	}
	// NOWAIT proves the control request has not retained the fleet row either.
	if _, err = blocker.Exec(ctx, `SELECT id FROM fleet_config FOR UPDATE NOWAIT`); err != nil {
		t.Fatal("D170 control compile took fleet lock before waiting on fence", err)
	}
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for _, done := range []<-chan error{capture, control} {
		if err = <-done; err != nil {
			t.Fatal(err)
		}
	}
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(op, true)
	d167NoDeadlocks(t, ctx, w)
}

func TestD170_T31_SweepAndCaptureLockNodesInOrder(t *testing.T) {
	w := newD165World(t, 2)
	// The existing control-status index can scan last_seen_at in the opposite
	// order to UUIDs. Establish that order using real accepted reports.
	w.observe(w.report(w.reports[1], "clean"))
	w.observe(w.report(w.reports[0], "clean"))
	op := w.revokeDevice()
	event := d165RemovalEvent(t, w, "device.revoked", w.device)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	blocker, err := w.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	ready := make(chan error, 1)
	w.services.Compiler = d167AfterCompile{compiler: w.compiler, after: func(ctx context.Context) error {
		_, e := blocker.Exec(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, w.reports[0].NodeId)
		ready <- e
		return e
	}}
	capture := d167Worker(ctx, w, event)
	if err = <-ready; err != nil {
		t.Fatal(err)
	}
	capturePID, err := d167WaitBlock(ctx, w, blocker.Conn().PgConn().PID())
	if err != nil {
		t.Fatal(err)
	}
	cfg := w.r.Pool.Config()
	cfg.MaxConns = 1
	// Only this test connection's planner settings change; no host settings or
	// product rows are rewritten to manufacture a physical tuple order.
	cfg.ConnConfig.RuntimeParams["enable_seqscan"] = "off"
	cfg.ConnConfig.RuntimeParams["enable_bitmapscan"] = "off"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	now := w.now.Add(3 * time.Minute)
	rows, err := pool.Query(ctx, `SELECT id FROM nodes WHERE control_status IN ('online','stale') AND (last_seen_at IS NULL OR last_seen_at<=$1)`, now.Add(-90*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	order, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(order) != 2 || order[0] != w.reports[1].NodeId {
		t.Fatal("D170 opposing unordered scan fixture not established", order, err)
	}
	sweep := make(chan error, 1)
	go func() {
		_, e := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now })).SweepStaleNodes(ctx, 0, 0)
		sweep <- e
	}()
	if _, err = d167WaitBlock(ctx, w, capturePID); err != nil {
		t.Fatal("sweep did not join capture node-lock queue", err)
	}
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for _, done := range []<-chan error{capture, sweep} {
		if err = <-done; err != nil {
			t.Fatal(err)
		}
	}
	for _, conn := range pool.AcquireAllIdle(ctx) {
		_, err = conn.Exec(ctx, `SELECT pg_stat_force_next_flush()`)
		conn.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	d167NoDeadlocks(t, ctx, w)
	w.wantStep(op, true)
	for _, r := range w.reports {
		w.target(op, r.NodeId, "unconfirmed")
	}
}

// Keep the transaction-carrying observation path visible to compiler-order tests.
var _ = commandtx.WithTransaction
