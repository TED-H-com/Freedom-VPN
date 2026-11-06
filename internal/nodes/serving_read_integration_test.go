//go:build integration

package nodes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func servingReadTestEvidence(at time.Time, tag string) api.ServingReadEvidence {
	return api.ServingReadEvidence{ReadAt: at.Add(-24 * time.Hour), Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: decstr.U64(^uint64(0)), Verified: false, Diagnostic: tag}
}

func TestD164_SelectedServingReadOrderingAndPresence(t *testing.T) {
	r, _, base := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	now := base.ObservedAt.Add(time.Hour)
	svc := nodes.NewService(r.Pool, nodes.WithClock(func() time.Time { return now }))
	read := func() (api.NodeDetail, string) {
		t.Helper()
		d, tag, e := svc.GetNode(ctx, base.NodeId)
		if e != nil {
			t.Fatal(e)
		}
		return d, tag
	}
	initial, _ := read()
	if !initial.ServingRead.IsSpecified() || !initial.ServingRead.IsNull() {
		t.Fatal("initial read must be explicit null")
	}
	post := func(report api.ObservationReport) api.ObservationAccepted {
		t.Helper()
		a, e := svc.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	var previous api.ObservationReport
	for i, delta := range []time.Duration{0, -time.Minute, 0} {
		report := base
		report.ReportId = uuid.New()
		report.Checks = append([]api.NodeCheck(nil), base.Checks...)
		tag := fmt.Sprintf("TEST ONLY accepted %d", i)
		report.Checks[0].Detail = &tag
		evidence := servingReadTestEvidence(base.ObservedAt, tag)
		report.ServingRead.Set(evidence)
		now = now.Add(delta)
		// Control generated IDs before insertion; accepted records remain immutable.
		id := uuid.MustParse(fmt.Sprintf("ffffffff-ffff-4fff-8fff-%012d", 99-i))
		if _, e := r.Pool.Exec(ctx, "ALTER TABLE node_observations ALTER COLUMN id SET DEFAULT '"+id.String()+"'::uuid"); e != nil {
			t.Fatal(e)
		}
		post(report)
		d, etag := read()
		got, e := d.ServingRead.Get()
		if e != nil || got != evidence || d.SafetyStatus != api.SafetyStatusPassed || *d.Checks[0].Detail != tag || etag != `"`+d.ResourceVersion.String()+`"` {
			t.Fatalf("mixed projection: %+v %s %v", d, etag, e)
		}
		previous = report
	}
	if _, e := r.Pool.Exec(ctx, "ALTER TABLE node_observations ALTER COLUMN id SET DEFAULT gen_random_uuid()"); e != nil {
		t.Fatal(e)
	}
	d, etag := read()
	post(previous)
	replay, replayTag := read()
	if replayTag != etag || !reflect.DeepEqual(replay.ServingRead, d.ServingRead) {
		t.Fatal("replay changed selected read/version")
	}
	older := previous
	older.ReportId = uuid.New()
	older.ObservedAt = base.ObservedAt.Add(-time.Second)
	older.ServingRead.Set(servingReadTestEvidence(base.ObservedAt, "TEST ONLY stale"))
	post(older)
	after, _ := read()
	if !reflect.DeepEqual(after.ServingRead, d.ServingRead) {
		t.Fatal("older arrival replaced selection")
	}
	for _, omitted := range []bool{true, false} {
		report := base
		report.ReportId = uuid.New()
		report.ObservedAt = base.ObservedAt.Add(time.Second)
		if !omitted {
			report.ServingRead.SetNull()
		}
		ack := post(report)
		got, _ := read()
		if !got.ServingRead.IsSpecified() || !got.ServingRead.IsNull() {
			t.Fatal("selected no-read searched backward")
		}
		if again := post(report); again != ack {
			t.Fatal("no-read replay changed ACK")
		}
		if omitted {
			report.ServingRead.SetNull()
		} else {
			report.ServingRead = base.ServingRead
		}
		_, e := svc.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report)
		if e == nil || !strings.Contains(e.Error(), "ACK_CONFLICT") {
			t.Fatalf("presence conflict: %v", e)
		}
	}
	// Original rejected read and stale report survive subsequent null projections.
	for _, report := range []api.ObservationReport{previous, older} {
		var raw []byte
		if e := r.Pool.QueryRow(ctx, `SELECT report_payload FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		var stored api.ObservationReport
		if e := json.Unmarshal(raw, &stored); e != nil || !reflect.DeepEqual(stored.ServingRead, report.ServingRead) {
			t.Fatal("immutable read lost", e)
		}
	}
}

func TestD164_ServingReadReinstallAndFrozenIdentity(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	report.ServingRead.Set(servingReadTestEvidence(report.ObservedAt, "TEST ONLY retained rejected read"))
	if _, e := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
		t.Fatal(e)
	}
	if _, e := r.Pool.Exec(ctx, `INSERT INTO node_identity_endpoints(incarnation_id,endpoint_hash,last_seen_at,mapping_frozen) VALUES($1,decode(repeat('00',32),'hex'),clock_timestamp(),true) ON CONFLICT(incarnation_id) DO UPDATE SET mapping_frozen=true`, report.NodeIncarnationId); e != nil {
		t.Fatal(e)
	}
	report.ReportId = uuid.New()
	if _, e := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
		t.Fatal(e)
	}
	d, _, e := r.Nodes.GetNode(ctx, report.NodeId)
	if e != nil || !reflect.DeepEqual(d.ServingRead, report.ServingRead) || d.SafetyStatus != api.SafetyStatusFailed {
		t.Fatal("freeze erased diagnostic or changed precedence", e)
	}
	ver := d.ResourceVersion
	if _, e = r.Enrollment.RevokeNode(ctx, enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, report.NodeId, &ver, "TEST ONLY reinstall"); e != nil {
		t.Fatal(e)
	}
	created, e := r.Enrollment.CreateEnrollment(ctx, enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &report.NodeId})
	if e != nil {
		t.Fatal(e)
	}
	token, e := created.Token.Get()
	if e != nil {
		t.Fatal(e)
	}
	request, _ := pkitest.Request(t)
	next, e := r.Enrollment.Enroll(ctx, token, request)
	if e != nil {
		t.Fatal(e)
	}
	d, _, e = r.Nodes.GetNode(ctx, report.NodeId)
	if e != nil || !d.ServingRead.IsNull() {
		t.Fatal("old incarnation read leaked", e)
	}
	report.NodeIncarnationId = next.Response.NodeIncarnationId
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(-time.Second)
	if _, e = r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
		t.Fatal(e)
	}
	d, _, e = r.Nodes.GetNode(ctx, report.NodeId)
	if e != nil || !d.ServingRead.IsNull() {
		t.Fatal("older first report selected", e)
	}
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(2 * time.Second)
	if _, e = r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
		t.Fatal(e)
	}
	d, _, e = r.Nodes.GetNode(ctx, report.NodeId)
	if e != nil || !reflect.DeepEqual(d.ServingRead, report.ServingRead) {
		t.Fatal("eligible new incarnation read missing", e)
	}
}

func TestD164_ServingReadConcurrentSnapshot(t *testing.T) {
	r, _, base := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	seed := "TEST ONLY seed"
	base.Checks[0].Detail = &seed
	base.ServingRead.Set(servingReadTestEvidence(base.ObservedAt, seed))
	if _, e := r.Nodes.SubmitObservation(ctx, base.NodeId, base.NodeIncarnationId, base); e != nil {
		t.Fatal(e)
	}
	firstWrite, firstRead, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(finished)
		for i := 0; i < 20; i++ {
			report := base
			report.ReportId = uuid.New()
			report.Checks = append([]api.NodeCheck(nil), base.Checks...)
			tag := strconv.Itoa(i)
			report.Checks[0].Detail = &tag
			report.ServingRead.Set(servingReadTestEvidence(base.ObservedAt, tag))
			if _, e := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
				errs <- e
				return
			}
			if i == 0 {
				close(firstWrite)
				select {
				case <-firstRead:
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				}
			}
		}
	}()
	for reader := 0; reader < 3; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-firstWrite:
			case <-finished:
				return
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
			for {
				d, etag, e := r.Nodes.GetNode(ctx, base.NodeId)
				if e != nil {
					errs <- e
					return
				}
				value, e := d.ServingRead.Get()
				if e != nil || d.Checks[0].Detail == nil || *d.Checks[0].Detail != value.Diagnostic || etag != `"`+d.ResourceVersion.String()+`"` {
					errs <- fmt.Errorf("incoherent diagnostic/checks/version")
					return
				}
				once.Do(func() { close(firstRead) })
				select {
				case <-finished:
					return
				default:
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	select {
	case <-firstRead:
	default:
		t.Fatal("no non-null read overlapped writer sequence")
	}
}

func TestD164_ServingReadMigration(t *testing.T) {
	for _, scenario := range []string{"same_value", "omitted_and_null", "no_report", "ambiguous", "malformed", "null_source", "invalid_calendar"} {
		t.Run(scenario, func(t *testing.T) {
			r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
			ctx := t.Context()
			if scenario != "omitted_and_null" {
				report.ServingRead.Set(servingReadTestEvidence(report.ObservedAt, "TEST ONLY legacy"))
			}
			if scenario != "no_report" {
				if _, e := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
					t.Fatal(e)
				}
			}
			report.ReportId = uuid.New()
			if scenario == "omitted_and_null" || scenario == "ambiguous" {
				report.ServingRead.SetNull()
			}
			if scenario != "no_report" {
				if _, e := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); e != nil {
					t.Fatal(e)
				}
			}
			database := stdlib.OpenDBFromPool(r.Pool)
			defer database.Close()
			provider, e := goose.NewProvider(goose.DialectPostgres, database, migrations.FS)
			if e != nil {
				t.Fatal(e)
			}
			snapshot := func() string {
				t.Helper()
				var raw string
				if err := r.Pool.QueryRow(ctx, `SELECT coalesce(jsonb_agg(to_jsonb(o) ORDER BY report_id),'[]'::jsonb)::text FROM node_observations o WHERE node_id=$1`, report.NodeId).Scan(&raw); err != nil {
					t.Fatal(err)
				}
				return raw
			}
			originalRows := snapshot()
			beforeDown, _, e := r.Nodes.GetNode(ctx, report.NodeId)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = provider.DownTo(ctx, 194); e != nil {
				t.Fatal(e)
			}
			var afterDown int64
			if e = r.Pool.QueryRow(ctx, `SELECT resource_version::bigint FROM nodes WHERE id=$1`, report.NodeId).Scan(&afterDown); e != nil || afterDown != int64(beforeDown.ResourceVersion)+1 {
				t.Fatal("Down failed to invalidate ETag once", e)
			}
			if originalRows != snapshot() {
				t.Fatal("Down changed immutable reports")
			}

			if scenario == "malformed" || scenario == "null_source" || scenario == "invalid_calendar" {
				raw, _ := json.Marshal(servingReadTestEvidence(report.ObservedAt, "TEST ONLY corrupt legacy fixture"))
				var v map[string]any
				if e = json.Unmarshal(raw, &v); e != nil {
					t.Fatal(e)
				}
				switch scenario {
				case "malformed":
					v = map[string]any{}
				case "null_source":
					v["source"] = nil
				case "invalid_calendar":
					v["read_at"] = "2026-02-30T00:00:00Z"
				}
				bad, _ := json.Marshal(v)
				// Synthetic malformed legacy insert under the migration owner,
				// never rewriting an immutable accepted report or disabling guards.
				if _, e = r.Pool.Exec(ctx, `INSERT INTO node_observations
                  SELECT (jsonb_populate_record(NULL::node_observations,
                    to_jsonb(o)||jsonb_build_object('id',gen_random_uuid(),'report_id',gen_random_uuid(),
                    'observed_at',o.observed_at+interval '1 second',
                    'report_payload',jsonb_set(o.report_payload,'{serving_read}',$2::jsonb)))).*
                  FROM node_observations o WHERE o.node_id=$1 LIMIT 1`, report.NodeId, bad); e != nil {
					t.Fatal(e)
				}
				if _, e = r.Pool.Exec(ctx, `UPDATE nodes SET last_observed_at=last_observed_at+interval '1 second',resource_version=resource_version+1 WHERE id=$1`, report.NodeId); e != nil {
					t.Fatal(e)
				}
			}
			var beforeRows string
			if e = r.Pool.QueryRow(ctx, `SELECT coalesce(jsonb_agg(to_jsonb(o) ORDER BY report_id),'[]'::jsonb)::text FROM node_observations o WHERE node_id=$1`, report.NodeId).Scan(&beforeRows); e != nil {
				t.Fatal(e)
			}
			var beforeVersion int64
			if e = r.Pool.QueryRow(ctx, `SELECT resource_version::bigint FROM nodes WHERE id=$1`, report.NodeId).Scan(&beforeVersion); e != nil {
				t.Fatal(e)
			}

			_, e = provider.Up(ctx)
			migrationErr := e
			var afterRows string
			if e = r.Pool.QueryRow(ctx, `SELECT coalesce(jsonb_agg(to_jsonb(o) ORDER BY report_id),'[]'::jsonb)::text FROM node_observations o WHERE node_id=$1`, report.NodeId).Scan(&afterRows); e != nil {
				t.Fatal(e)
			}
			if beforeRows != afterRows {
				t.Fatal("migration changed immutable payload/hash records")
			}
			e = migrationErr
			if scenario == "ambiguous" || scenario == "malformed" || scenario == "null_source" {
				if e == nil {
					t.Fatal("unsafe legacy evidence accepted")
				}
				var cols int
				if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name='nodes' AND column_name='serving_read'`).Scan(&cols); err != nil || cols != 0 {
					t.Fatal("failed migration did not roll back schema", err)
				}
				var version int64
				if err := r.Pool.QueryRow(ctx, `SELECT resource_version::bigint FROM nodes WHERE id=$1`, report.NodeId).Scan(&version); err != nil || version != beforeVersion {
					t.Fatal("failed migration changed version", err)
				}
				if scenario == "ambiguous" && !strings.Contains(e.Error(), "Ambiguous legacy serving_read") {
					t.Fatal(e)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			d, _, e := r.Nodes.GetNode(ctx, report.NodeId)
			if scenario != "invalid_calendar" && e == nil && int64(d.ResourceVersion) != beforeVersion+1 {
				t.Fatal("activation did not increment version once")
			}
			if scenario == "invalid_calendar" {
				if e == nil {
					t.Fatal("malformed calendar date became readable/null")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if scenario == "same_value" && !reflect.DeepEqual(d.ServingRead, report.ServingRead) {
				t.Fatal("legacy read lost")
			}
			if (scenario == "omitted_and_null" || scenario == "no_report") && !d.ServingRead.IsNull() {
				t.Fatal("legacy null fabricated")
			}
		})
	}
}

func TestD164_ServingReadIncarnationForeignKey(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	other, e := r.Enrollment.CreateEnrollment(ctx, enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller})
	if e != nil {
		t.Fatal(e)
	}
	token, e := other.Token.Get()
	if e != nil {
		t.Fatal(e)
	}
	request, _ := pkitest.Request(t)
	enrolled, e := r.Enrollment.Enroll(ctx, token, request)
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.Pool.Exec(ctx, `UPDATE nodes SET serving_read_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, report.NodeId, enrolled.Response.NodeIncarnationId)
	var pgErr *pgconn.PgError
	if !errors.As(e, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("foreign-node incarnation accepted or wrong rejection: %v", e)
	}
}

func TestD164_TypedInvalidServingReadDoesNotPersist(t *testing.T) {
	r, _, base := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	for name, change := range map[string]func(*api.ServingReadEvidence){
		"source":     func(v *api.ServingReadEvidence) { v.Source = "invented" },
		"diagnostic": func(v *api.ServingReadEvidence) { v.Diagnostic = strings.Repeat("界", 501) },
		"time":       func(v *api.ServingReadEvidence) { v.ReadAt = v.ReadAt.In(time.FixedZone("TEST ONLY", 3600)) },
	} {
		t.Run(name, func(t *testing.T) {
			report := base
			report.ReportId = uuid.New()
			value := servingReadTestEvidence(report.ObservedAt, "TEST ONLY")
			change(&value)
			report.ServingRead.Set(value)
			_, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report)
			var failure *apierr.Error
			if !errors.As(err, &failure) || failure.Code != apierr.ValidationFailed {
				t.Fatalf("wrong typed validation error: %v", err)
			}
			var count int
			if err := r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&count); err != nil || count != 0 {
				t.Fatal("invalid typed report persisted", err)
			}
		})
	}
}
