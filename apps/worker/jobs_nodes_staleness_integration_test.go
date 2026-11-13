//go:build integration

package worker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const d169Sweep = "node.staleness.sweep"

type d169Fixture struct {
	t      *testing.T
	r      *httpwire.Runtime
	now    time.Time
	report api.ObservationReport
}

func newD169(t *testing.T) *d169Fixture {
	t.Helper()
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	f := &d169Fixture{t: t, r: r, now: time.Now().UTC().Truncate(time.Microsecond), report: report}
	// Change only this service's existing test clock, retaining its production
	// observation hook. Enrollment, reports and publication create all domain
	// rows; no last_seen_at, status or publication projections are seeded.
	nodelib.WithClock(func() time.Time { return f.now })(r.Nodes)
	r.Config.ReconcileSeconds = 7
	return f
}

func (f *d169Fixture) submit(report api.ObservationReport) api.ObservationReport {
	f.t.Helper()
	report.ReportId, report.ObservedAt = uuid.New(), f.now
	for i := range report.Checks {
		report.Checks[i].ObservedAt = f.now
	}
	if _, err := f.r.Nodes.SubmitObservation(f.t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		f.t.Fatal("accepted report:", err)
	}
	f.status(report.NodeId, api.ControlStatusOnline)
	return report
}

func (f *d169Fixture) enroll() api.ObservationReport {
	f.t.Helper()
	created, err := f.r.Enrollment.CreateEnrollment(f.t.Context(), enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller})
	if err != nil {
		f.t.Fatal(err)
	}
	token, err := created.Token.Get()
	if err != nil {
		f.t.Fatal(err)
	}
	request, _ := pkitest.Request(f.t)
	enrolled, err := f.r.Enrollment.Enroll(f.t.Context(), token, request)
	if err != nil {
		f.t.Fatal(err)
	}
	compiled, err := f.r.DesiredStateCompiler().Compile(f.t.Context(), enrolled.Response.NodeId)
	if err != nil {
		f.t.Fatal(err)
	}
	report := f.report
	report.NodeId, report.NodeIncarnationId = enrolled.Response.NodeId, enrolled.Response.NodeIncarnationId
	report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration)
	return report
}

func (f *d169Fixture) status(node uuid.UUID, want api.ControlStatus) {
	f.t.Helper()
	detail, _, err := f.r.Nodes.GetNode(f.t.Context(), node)
	if err != nil {
		f.t.Fatal(err)
	}
	if detail.ControlStatus != want {
		f.t.Fatalf("control_status = %s, want %s", detail.ControlStatus, want)
	}
}

func (f *d169Fixture) sql(query string, args ...any) {
	f.t.Helper()
	if _, err := f.r.Pool.Exec(f.t.Context(), query, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *d169Fixture) seed() {
	f.t.Helper()
	// The scheduler seeds immediately. This bounded run lets its first
	// transaction complete, then checks shutdown; it does not age any node.
	ctx, cancel := context.WithTimeout(f.t.Context(), 250*time.Millisecond)
	defer cancel()
	if err := RunNodeSchedule(ctx, f.r); err != nil {
		f.t.Fatal("schedule:", err)
	}
}

func (f *d169Fixture) jobs() Registry {
	f.t.Helper()
	jobs := NodeJobs(f.r, nil, nil, 3)
	if jobs[d169Sweep] == nil {
		f.t.Fatal("node.staleness.sweep handler is not registered")
	}
	return jobs
}

func (f *d169Fixture) dispatcher(jobs Registry) *Dispatcher {
	f.t.Helper()
	d, err := New(f.r.Pool, jobs, Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Second, MaxBackoff: time.Second}, nil, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	return d
}

func (f *d169Fixture) focus(eventType string) {
	f.t.Helper()
	// Only outbox availability is controlled to choose the next real job;
	// node/report/publication data and the database clock remain untouched.
	f.sql(`UPDATE outbox_events SET available_at=CASE WHEN event_type=$1 THEN clock_timestamp() ELSE clock_timestamp()+interval '1 day' END WHERE delivered_at IS NULL AND dead_lettered_at IS NULL`, eventType)
}

func (f *d169Fixture) dispatch(d *Dispatcher, eventType string) {
	f.t.Helper()
	f.focus(eventType)
	if worked, err := d.Once(f.t.Context()); err != nil || !worked {
		f.t.Fatalf("dispatch %s: worked=%t err=%v", eventType, worked, err)
	}
}

func (f *d169Fixture) counts(eventType string, wantDelivered, wantPending int) {
	f.t.Helper()
	var delivered, pending, dead int
	if err := f.r.Pool.QueryRow(f.t.Context(), `SELECT count(*) FILTER (WHERE delivered_at IS NOT NULL AND delivery_receipt IS NOT NULL),count(*) FILTER (WHERE delivered_at IS NULL AND dead_lettered_at IS NULL),count(*) FILTER (WHERE dead_lettered_at IS NOT NULL) FROM outbox_events WHERE event_type=$1`, eventType).Scan(&delivered, &pending, &dead); err != nil {
		f.t.Fatal(err)
	}
	if delivered != wantDelivered || pending != wantPending || dead != 0 {
		f.t.Fatalf("%s: delivered=%d pending=%d dead=%d, want %d/%d/0", eventType, delivered, pending, dead, wantDelivered, wantPending)
	}
}

func (f *d169Fixture) sweep(d *Dispatcher) {
	f.t.Helper()
	f.dispatch(d, d169Sweep)
	var delivered int
	if err := f.r.Pool.QueryRow(f.t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type=$1 AND delivered_at IS NOT NULL AND delivery_receipt IS NOT NULL`, d169Sweep).Scan(&delivered); err != nil || delivered == 0 {
		f.t.Fatalf("sweep was not delivered: count=%d err=%v", delivered, err)
	}
}

func TestD169_T1_ScheduleCoalescesAndReschedules(t *testing.T) {
	for _, seconds := range []int{7, 3600} {
		t.Run(fmt.Sprintf("reconcile_seconds_%d", seconds), func(t *testing.T) {
			f := newD169(t)
			f.submit(f.report)
			f.r.Config.ReconcileSeconds = seconds
			f.seed()
			f.seed()
			f.counts(d169Sweep, 0, 1)
			jobs := f.jobs()
			d := f.dispatcher(jobs)
			f.focus(d169Sweep)
			events, err := db.ClaimOutbox(t.Context(), f.r.Pool, "TEST ONLY staleness worker", 1, time.Minute)
			if err != nil || len(events) != 1 || events[0].EventType != d169Sweep {
				t.Fatalf("sweep claim: count=%d err=%v", len(events), err)
			}
			lost := events[0]
			lost.LeaseToken = uuid.New()
			if _, err := jobs[d169Sweep](t.Context(), lost); !errors.Is(err, db.ErrLeaseLost) {
				t.Fatalf("lost sweep claim: %v, want ErrLeaseLost", err)
			}
			f.counts(d169Sweep, 0, 1)
			start := time.Now()
			receipt, err := jobs[d169Sweep](t.Context(), events[0])
			if err != nil || receipt == "" {
				t.Fatalf("sweep handler: receipt=%q err=%v", receipt, err)
			}
			if err := d.complete(t.Context(), events[0], receipt); err != nil {
				t.Fatal(err)
			}
			end := time.Now()
			f.counts(d169Sweep, 1, 1)
			var available time.Time
			if err := f.r.Pool.QueryRow(t.Context(), `SELECT available_at FROM outbox_events WHERE event_type=$1 AND delivered_at IS NULL AND dead_lettered_at IS NULL`, d169Sweep).Scan(&available); err != nil {
				t.Fatal(err)
			}
			period := time.Duration(min(seconds, 60)) * time.Second
			if available.Before(start.Add(period-time.Millisecond)) || available.After(end.Add(period+time.Millisecond)) {
				t.Fatalf("successor delay = %s, want %s after handler (within execution interval)", available.Sub(start), period)
			}
			// A scheduler restart must retain the delayed successor, not add a
			// second immediate event or pull the successor forward.
			f.seed()
			f.counts(d169Sweep, 1, 1)
			var retained time.Time
			if err := f.r.Pool.QueryRow(t.Context(), `SELECT available_at FROM outbox_events WHERE event_type=$1 AND delivered_at IS NULL`, d169Sweep).Scan(&retained); err != nil || !retained.Equal(available) {
				t.Fatalf("scheduler changed delayed successor: equal=%t err=%v", retained.Equal(available), err)
			}
		})
	}
}

func TestD169_T2_ReportingNodesAgeThroughStatuses(t *testing.T) {
	f := newD169(t)
	d := f.dispatcher(f.jobs())
	f.r.Config.OnlineSeconds = 240 // Also proves a non-default offline window.
	start := f.now
	offline := f.submit(f.report)
	stale := f.enroll()
	online := f.enroll()
	f.now = start.Add(180 * time.Second)
	stale = f.submit(stale)
	f.now = start.Add(240 * time.Second)
	online = f.submit(online)
	f.now = start.Add(300 * time.Second)
	f.seed()
	f.sweep(d)
	f.status(online.NodeId, api.ControlStatusOnline)   // silent 60s
	f.status(stale.NodeId, api.ControlStatusStale)     // silent 120s
	f.status(offline.NodeId, api.ControlStatusOffline) // silent 300s
}

func TestD169_T3_ShortOnlineWindowSkipsStale(t *testing.T) {
	f := newD169(t)
	d := f.dispatcher(f.jobs())
	f.r.Config.OnlineSeconds = 30
	report := f.submit(f.report)
	start := f.now
	f.seed()
	f.now = start.Add(30*time.Second - time.Microsecond)
	f.sweep(d)
	f.status(report.NodeId, api.ControlStatusOnline)
	f.now = start.Add(30 * time.Second)
	f.sweep(d)
	f.status(report.NodeId, api.ControlStatusOffline)
	f.counts(d169Sweep, 2, 1)
}

func TestD169_T4_AcceptedReportRecoversOfflineNode(t *testing.T) {
	f := newD169(t)
	d := f.dispatcher(f.jobs())
	report := f.submit(f.report)
	f.seed()
	f.now = f.now.Add(181 * time.Second)
	f.sweep(d)
	f.status(report.NodeId, api.ControlStatusOffline)
	f.now = f.now.Add(time.Second)
	f.submit(report)
	f.sweep(d)
	f.status(report.NodeId, api.ControlStatusOnline)
}

func TestD169_T5_OfflinePreservesPublishedIntent(t *testing.T) {
	f := newD169(t)
	d := f.dispatcher(f.jobs())
	report := f.submit(f.report)
	_, etag, err := f.r.Nodes.GetNode(t.Context(), report.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	publication := exits.NewService(f.r.Pool, f.r.DesiredStateCompiler(), nil)
	if _, _, err := publication.SetPublication(t.Context(), exits.SetPublicationParams{NodeID: report.NodeId, Principal: "admin:" + uuid.NewString(), IdempotencyKey: uuid.New(), IfMatch: etag, Request: api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY staleness preserves intent"}}); err != nil {
		t.Fatal(err)
	}
	before, _, err := f.r.Nodes.GetNode(t.Context(), report.NodeId)
	if err != nil || before.PublicationIntent != api.PublicationIntentPublished {
		t.Fatalf("publication setup: intent=%s err=%v", before.PublicationIntent, err)
	}
	f.seed()
	f.now = f.now.Add(181 * time.Second)
	f.sweep(d)
	f.status(report.NodeId, api.ControlStatusOffline)
	after, _, err := f.r.Nodes.GetNode(t.Context(), report.NodeId)
	if err != nil || after.PublicationIntent != before.PublicationIntent {
		t.Fatalf("offline changed publication_intent: got=%s want=%s err=%v", after.PublicationIntent, before.PublicationIntent, err)
	}
}

func TestD169_T6_SweepFailureIsIsolatedAndRetried(t *testing.T) {
	f := newD169(t)
	jobs := f.jobs()
	report := f.submit(f.report)
	f.seed()
	f.now = f.now.Add(181 * time.Second)
	// TEST ONLY fault in this test's isolated database. It rejects exactly
	// status changes, leaving the real node.reconcile dependencies available.
	f.sql(`CREATE FUNCTION d169_fail_status() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'TEST ONLY D169 sweep failure'; END $$`)
	f.sql(`CREATE TRIGGER d169_fail_status BEFORE UPDATE OF control_status ON nodes FOR EACH ROW WHEN (OLD.control_status IS DISTINCT FROM NEW.control_status) EXECUTE FUNCTION d169_fail_status()`)
	var sweepErr error
	handler := jobs[d169Sweep]
	jobs[d169Sweep] = func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		receipt, err := handler(ctx, event)
		sweepErr = err
		return receipt, err
	}
	d := f.dispatcher(jobs)
	f.dispatch(d, d169Sweep)
	var pgErr *pgconn.PgError
	if !errors.As(sweepErr, &pgErr) || pgErr.Message != "TEST ONLY D169 sweep failure" {
		t.Fatalf("sweep did not return injected database error: %v", sweepErr)
	}
	f.counts(d169Sweep, 0, 1)
	var attempt int
	var released bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT attempt,lease_owner IS NULL AND lease_token IS NULL AND lease_until IS NULL FROM outbox_events WHERE event_type=$1`, d169Sweep).Scan(&attempt, &released); err != nil || attempt != 1 || !released {
		t.Fatalf("failed sweep not released for retry: attempt=%d released=%t err=%v", attempt, released, err)
	}
	f.status(report.NodeId, api.ControlStatusOnline)
	f.dispatch(d, "node.reconcile")
	f.counts("node.reconcile", 1, 1)
	f.counts(d169Sweep, 0, 1)
	f.sql(`DROP TRIGGER d169_fail_status ON nodes`)
	f.sql(`DROP FUNCTION d169_fail_status()`)
	f.sweep(d)
	f.counts(d169Sweep, 1, 1)
	f.status(report.NodeId, api.ControlStatusOffline)
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT attempt FROM outbox_events WHERE event_type=$1 AND delivered_at IS NOT NULL`, d169Sweep).Scan(&attempt); err != nil || attempt != 2 {
		t.Fatalf("sweep retry did not finish original event: attempt=%d err=%v", attempt, err)
	}
}
