//go:build integration

package usage_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// Exact ENR-07 times/counts, with test-generated identities. Exercise migration
// on a populated 00192 ledger and retry the same sealed HTTP request/key.
func TestMTR03_ENR07_D130MigrationRetryACKAndStoredReplay(t *testing.T) {
	ctx := t.Context()
	pool := dbtest.NewUnmigrated(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 192); err != nil {
		t.Fatal(err)
	}
	st := newUsageHTTPStackWithPool(t, pool)
	state, err := st.svc.GetLedgerState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	start := precisionTime(t, "2026-10-31T14:01:15.198144331Z")
	end := precisionTime(t, "2026-10-31T14:01:15.198144332Z")
	id := usage.NodeIdentity{NodeID: st.node1, NodeIncarnationID: st.inc1}
	records := []api.UsageRecord{{MemberId: st.memberA, DeviceId: st.deviceA,
		BindingRevision: 1, PeriodId: "2026-10", UploadBytes: 550, DownloadBytes: 358}}
	gap := &api.UsageGap{Reason: api.UsageGapReasonUnknown, LastCheckpointAt: presence.NewNullNullable[time.Time]()}
	// Existing strict intervals exercise all three replacement constraints.
	oldID := usage.NodeIdentity{NodeID: st.node2, NodeIncarnationID: st.inc2}
	old := makeSignedEvent(t, oldID, 1, uuid.New(), start.Add(-time.Hour), end, api.UsageQualityGap, gap, records)
	oldResult, err := st.svc.IngestBatch(ctx, oldID, api.UsageBatchRequest{
		NodeId: oldID.NodeID, NodeIncarnationId: oldID.NodeIncarnationID, Events: []api.UsageEvent{old},
	})
	if err != nil || oldResult.HighestContiguousSeq != 1 {
		t.Fatalf("seed 00192 ledger: %+v, %v", oldResult, err)
	}

	ev1 := makeSignedEvent(t, id, 1, uuid.New(), start, end, api.UsageQualityGap, gap, []api.UsageRecord{})
	ev2 := makeSignedEvent(t, id, 2, uuid.New(),
		precisionTime(t, "2026-10-31T14:03:33.15863239Z"),
		precisionTime(t, "2026-10-31T14:03:38.180117348Z"), api.UsageQualityComplete, nil, records)
	batch := api.UsageBatchRequest{NodeId: id.NodeID, NodeIncarnationId: id.NodeIncarnationID,
		LedgerEpoch: presence.NewNullableWithValue(state.LedgerEpoch), Events: []api.UsageEvent{ev1}}
	key := uuid.NewString()
	failed := postHandoff(t, st.agent, batch, key, nil)
	if failed.Code != http.StatusServiceUnavailable || !strings.Contains(failed.Body.String(), `"code":"DEPENDENCY_UNAVAILABLE"`) ||
		!strings.Contains(failed.Body.String(), `"retryable":true`) {
		t.Fatalf("00192 failure classification: %d %s", failed.Code, failed.Body.String())
	}
	if strings.Contains(failed.Body.String(), "usage_events_interval_check") {
		t.Fatal("response exposed SQL constraint details")
	}
	before := precisionLedgerSnapshot(t, pool)
	if _, err := provider.UpTo(ctx, 193); err != nil {
		t.Fatalf("migrate populated 00192 ledger: %v", err)
	}
	if after := precisionLedgerSnapshot(t, pool); after != before {
		t.Fatal("migration mutated existing ledger rows")
	}
	// Down/Up works while all persisted intervals still satisfy the old rule.
	if _, err := provider.DownTo(ctx, 192); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 193); err != nil {
		t.Fatal(err)
	}
	assertPrecisionConstraints(t, pool)

	first := postHandoff(t, st.agent, batch, key, nil)
	assertPrecisionHTTPResult(t, first.Code, first.Body.Bytes(), 1, api.UsageEventResultStatusAccepted)
	assertPrecisionEvent(t, pool, ev1)
	var from, until time.Time
	if err := pool.QueryRow(ctx, `SELECT affected_from, affected_until FROM usage_ledger_gaps
		WHERE node_id=$1 AND incarnation_id=$2`, id.NodeID, id.NodeIncarnationID).Scan(&from, &until); err != nil {
		t.Fatal(err)
	}
	if !from.Equal(start.Truncate(time.Microsecond)) || !until.Equal(from) {
		t.Fatalf("gap projection: %v to %v", from, until)
	}
	batch2 := batch
	batch2.Events = []api.UsageEvent{ev2}
	key2 := uuid.NewString()
	second := postHandoff(t, st.agent, batch2, key2, nil)
	assertPrecisionHTTPResult(t, second.Code, second.Body.Bytes(), 2, api.UsageEventResultStatusAccepted)
	assertPrecisionEvent(t, pool, ev2)
	assertPrecisionSample(t, pool, ev2)
	var up, down, total string
	if err := pool.QueryRow(ctx, `SELECT upload_bytes::text, download_bytes::text, total_bytes::text
		FROM usage_rollup_details WHERE node_id=$1 AND incarnation_id=$2 AND device_id=$3
		AND period='2026-10' AND granularity='month'`, id.NodeID, id.NodeIncarnationID, st.deviceA).Scan(&up, &down, &total); err != nil {
		t.Fatal(err)
	}
	if up != "550" || down != "358" || total != "908" {
		t.Fatalf("current stream attribution = %s/%s/%s", up, down, total)
	}
	// Same key returns the original result, including ACK 1 after ACK reached 2.
	for _, replay := range []struct {
		request api.UsageBatchRequest
		key     string
		body    []byte
	}{{batch, key, first.Body.Bytes()}, {batch2, key2, second.Body.Bytes()}} {
		w := postHandoff(t, st.agent, replay.request, replay.key, nil)
		if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), replay.body) {
			t.Fatalf("same-key stored result changed: %d %s", w.Code, w.Body.String())
		}
	}
	duplicate := postHandoff(t, st.agent, batch, uuid.NewString(), nil)
	assertPrecisionHTTPResult(t, duplicate.Code, duplicate.Body.Bytes(), 2, api.UsageEventResultStatusDuplicate)
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id
		WHERE e.node_id=$1 AND e.incarnation_id=$2`, id.NodeID, id.NodeIncarnationID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("replay sample count = %d, %v", count, err)
	}
	// Downgrade must refuse to erase accepted equal-endpoint evidence and roll back DDL.
	if _, err := provider.DownTo(ctx, 192); err == nil {
		t.Fatal("downgrade accepted equal projected endpoints")
	}
	assertPrecisionConstraints(t, pool)
	assertPrecisionEvent(t, pool, ev1)
}

func TestMTR04_D130NanosecondSamplesAndRebuild(t *testing.T) {
	for _, rawStart := range []string{
		"2026-10-31T14:01:15.198144331Z", // exact ENR-07 interval, now carrying bytes
		"2026-10-31T14:01:15.198144831Z", // truncation, never rounding up
		"2026-10-31T00:00:00Z",           // equal projection at UTC day boundary
	} {
		t.Run(rawStart, func(t *testing.T) {
			env := setupTestEnv(t)
			ctx := t.Context()
			start := precisionTime(t, rawStart)
			// Old receipt time permits the real retention path to purge payloads.
			env.now = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			records := []api.UsageRecord{{MemberId: env.memberA, DeviceId: env.deviceA,
				BindingRevision: 1, PeriodId: "2026-10", UploadBytes: 550, DownloadBytes: 358}}
			ev := makeSignedEvent(t, env.identity1, 1, uuid.New(), start, start.Add(time.Nanosecond), api.UsageQualityComplete, nil, records)
			gap := makeSignedEvent(t, env.identity1, 2, uuid.New(), start, start.Add(time.Nanosecond),
				api.UsageQualityGap, &api.UsageGap{Reason: api.UsageGapReasonUnknown, LastCheckpointAt: presence.NewNullNullable[time.Time]()}, []api.UsageRecord{})
			result, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
				NodeId: env.node1, NodeIncarnationId: env.inc1, Events: []api.UsageEvent{ev},
			})
			if err != nil || result.HighestContiguousSeq != 1 || len(result.Results) != 1 ||
				result.Results[0].Status != api.UsageEventResultStatusAccepted {
				t.Fatalf("1ns byte event: %+v, %v", result, err)
			}
			assertPrecisionEvent(t, env.pool, ev)
			assertPrecisionSample(t, env.pool, ev)
			assertTotal := func(wantGap bool) {
				t.Helper()
				total, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
				if err != nil || total.UploadBytes != 550 || total.DownloadBytes != 358 || total.TotalBytes != 908 || total.Estimated || total.Gap != wantGap {
					t.Fatalf("1ns projection: %+v, %v", total, err)
				}
				var daily string
				var estimated, gap bool
				if err := env.pool.QueryRow(ctx, `SELECT total_bytes::text, estimated, gap FROM usage_rollups
					WHERE member_id=$1 AND period='2026-10-31' AND granularity='day' AND projection_generation=$2::numeric`,
					env.memberA, total.ProjectionGeneration.String()).Scan(&daily, &estimated, &gap); err != nil || daily != "908" || estimated || gap != wantGap {
					t.Fatalf("daily projection = %s estimated=%v gap=%v, %v", daily, estimated, gap, err)
				}
			}
			assertTotal(false)
			// Purge the complete sample before reporting a gap: gap-marked
			// rollups intentionally retain sample payloads under existing policy.
			purged, err := env.svc.PurgeRetainedPayloads(ctx, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "TEST ONLY D-130 rebuild after retention")
			if err != nil || purged != 1 {
				t.Fatalf("payload purge = %d, %v", purged, err)
			}
			result, err = env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
				NodeId: env.node1, NodeIncarnationId: env.inc1, Events: []api.UsageEvent{gap},
			})
			if err != nil || result.HighestContiguousSeq != 2 || len(result.Results) != 1 || result.Results[0].Status != api.UsageEventResultStatusAccepted {
				t.Fatalf("1ns gap: %+v, %v", result, err)
			}
			assertTotal(true)
			purged, err = env.svc.PurgeRetainedPayloads(ctx, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "TEST ONLY D-130 gap retention")
			if err != nil || purged != 1 {
				t.Fatalf("gap payload purge = %d, %v", purged, err)
			}
			var payloads int
			if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM usage_event_payloads`).Scan(&payloads); err != nil || payloads != 0 {
				t.Fatalf("payloads still retained: %d, %v", payloads, err)
			}
			if _, err := env.svc.RebuildProjection(ctx); err != nil {
				t.Fatalf("rebuild collapsed persisted intervals: %v", err)
			}
			assertTotal(true)
		})
	}
}

func TestMTR03_D130CanonicalIntervalOrderStillStrict(t *testing.T) {
	env := setupTestEnv(t)
	start := precisionTime(t, "2026-10-31T14:01:15.198144331Z")
	for i, end := range []time.Time{start, start.Add(-time.Nanosecond)} {
		ev := makeSignedEvent(t, env.identity1, uint64(i+1), uuid.New(), start, end,
			api.UsageQualityGap, &api.UsageGap{Reason: api.UsageGapReasonUnknown, LastCheckpointAt: presence.NewNullNullable[time.Time]()}, []api.UsageRecord{})
		result, err := env.svc.IngestBatch(t.Context(), env.identity1, api.UsageBatchRequest{
			NodeId: env.node1, NodeIncarnationId: env.inc1, Events: []api.UsageEvent{ev},
		})
		if err != nil || result.HighestContiguousSeq != 0 || len(result.Results) != 1 || result.Results[0].Status != api.UsageEventResultStatusRejected {
			t.Fatalf("invalid order accepted: %+v, %v", result, err)
		}
		got, err := result.Results[0].Error.Get()
		if err != nil || got.Code != api.ErrorCodeVALIDATIONFAILED || len(got.Fields) != 1 ||
			got.Fields[0].Field != "/events/0/interval_end" || got.Fields[0].Reason != "interval_order" {
			t.Fatalf("wrong canonical validation error: %+v, %v", got, err)
		}
	}
	var events, rejections int
	if err := env.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM usage_events),
		(SELECT count(*) FROM usage_event_rejections)`).Scan(&events, &rejections); err != nil || events != 0 || rejections != 2 {
		t.Fatalf("invalid interval persistence: events=%d rejections=%d, %v", events, rejections, err)
	}
}

func precisionTime(t *testing.T, raw string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func assertPrecisionHTTPResult(t *testing.T, status int, raw []byte, ack decstr.U64, eventStatus api.UsageEventResultStatus) {
	t.Helper()
	var result api.UsageBatchResult
	if err := json.Unmarshal(raw, &result); err != nil || status != http.StatusOK || result.HighestContiguousSeq != ack ||
		len(result.Results) != 1 || result.Results[0].Status != eventStatus {
		t.Fatalf("HTTP result status=%d body=%s error=%v", status, raw, err)
	}
}

func assertPrecisionEvent(t *testing.T, pool *pgxpool.Pool, ev api.UsageEvent) {
	t.Helper()
	var start, end time.Time
	var hash, payload []byte
	if err := pool.QueryRow(t.Context(), `SELECT e.interval_start, e.interval_end, e.payload_hash, p.payload
		FROM usage_events e JOIN usage_event_payloads p ON p.event_id=e.id
		WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3::numeric`,
		ev.NodeId, ev.NodeIncarnationId, ev.EventSeq.String()).Scan(&start, &end, &hash, &payload); err != nil {
		t.Fatal(err)
	}
	wantHash, _, canonical, err := usage.ComputeEventHash(ev)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(ev.IntervalStart.Truncate(time.Microsecond)) || !end.Equal(ev.IntervalEnd.Truncate(time.Microsecond)) ||
		!bytes.Equal(hash, wantHash[:]) || !bytes.Equal(payload, canonical) {
		t.Fatal("stored projection or original canonical bytes/hash changed")
	}
}

func assertPrecisionSample(t *testing.T, pool *pgxpool.Pool, ev api.UsageEvent) {
	t.Helper()
	var start, end time.Time
	var member, device uuid.UUID
	var revision, up, down, elapsed string
	if err := pool.QueryRow(t.Context(), `SELECT s.interval_start, s.interval_end, s.member_id, s.device_id,
		s.binding_revision::text, s.upload_bytes::text, s.download_bytes::text, s.monotonic_elapsed_ns::text
		FROM usage_samples s JOIN usage_events e ON e.id=s.event_id
		WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3::numeric`,
		ev.NodeId, ev.NodeIncarnationId, ev.EventSeq.String()).Scan(&start, &end, &member, &device, &revision, &up, &down, &elapsed); err != nil {
		t.Fatal(err)
	}
	rec := ev.Records[0]
	if !start.Equal(ev.IntervalStart.Truncate(time.Microsecond)) || !end.Equal(ev.IntervalEnd.Truncate(time.Microsecond)) ||
		member != rec.MemberId || device != rec.DeviceId || revision != rec.BindingRevision.String() ||
		up != rec.UploadBytes.String() || down != rec.DownloadBytes.String() || elapsed != decstr.U64(ev.IntervalEnd.Sub(ev.IntervalStart).Nanoseconds()).String() {
		t.Fatal("sample timestamps, attribution, byte counts or nanosecond elapsed changed")
	}
}

func precisionLedgerSnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var snapshot string
	if err := pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
		'events', (SELECT jsonb_agg(e ORDER BY id) FROM usage_events e),
		'samples', (SELECT jsonb_agg(s ORDER BY id) FROM usage_samples s),
		'payloads', (SELECT jsonb_agg(p ORDER BY id) FROM usage_event_payloads p),
		'gaps', (SELECT jsonb_agg(g ORDER BY id) FROM usage_ledger_gaps g),
		'checkpoints', (SELECT jsonb_agg(c ORDER BY id) FROM usage_checkpoints c))::text`).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func assertPrecisionConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	// Read each actual validated constraint into a private temporary table so
	// null, infinity and reversed endpoints can be checked without ledger writes.
	for _, target := range []struct{ table, constraint, start, end string }{
		{"usage_events", "usage_events_interval_check", "interval_start", "interval_end"},
		{"usage_samples", "usage_samples_check", "interval_start", "interval_end"},
		{"usage_ledger_gaps", "usage_ledger_gaps_check1", "affected_from", "affected_until"},
	} {
		var expression string
		var validated bool
		if err := pool.QueryRow(t.Context(), `SELECT pg_get_constraintdef(oid), convalidated FROM pg_constraint
			WHERE conrelid=$1::regclass AND conname=$2`, target.table, target.constraint).Scan(&expression, &validated); err != nil || !validated {
			t.Fatalf("constraint %s not validated: %v", target.constraint, err)
		}
		conn, err := pool.Acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer conn.Release()
			tx, err := conn.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(t.Context()) // private SQL scratch disappears on rollback
			nullability := " NOT NULL"
			if target.table == "usage_events" {
				nullability = ""
			}
			_, err = tx.Exec(t.Context(), `CREATE TEMP TABLE precision_check (`+target.start+` timestamptz`+nullability+`, `+
				target.end+` timestamptz`+nullability+`, `+expression+`) ON COMMIT DROP`)
			if err != nil {
				t.Fatal(err)
			}
			for _, pair := range []struct{ start, end, code string }{
				{"2026-10-31 14:01:15.198144Z", "2026-10-31 14:01:15.198144Z", ""},
				{"2026-10-31 14:01:15.198144Z", "2026-10-31 14:01:15.198145Z", ""},
				{"2026-10-31 14:01:15.198145Z", "2026-10-31 14:01:15.198144Z", "23514"},
				{"infinity", "infinity", "23514"},
				{"-infinity", "2026-10-31T00:00:00Z", "23514"},
				{"", "2026-10-31T00:00:00Z", "null"},
				{"2026-10-31T00:00:00Z", "", "null"},
				{"", "", "both_null"},
			} {
				var start, end any = pair.start, pair.end
				if pair.start == "" {
					start = nil
				}
				if pair.end == "" {
					end = nil
				}
				wantCode := pair.code
				if pair.code == "null" || pair.code == "both_null" {
					wantCode = "23502"
					if target.table == "usage_events" {
						wantCode = "23514"
						if pair.code == "both_null" {
							wantCode = ""
						}
					}
				}
				sub, err := tx.Begin(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				_, insertErr := sub.Exec(t.Context(), `INSERT INTO precision_check VALUES ($1::text::timestamptz, $2::text::timestamptz)`, start, end)
				if err := sub.Rollback(t.Context()); err != nil {
					t.Fatal(err)
				}
				var pgerr *pgconn.PgError
				if (wantCode == "" && insertErr != nil) || (wantCode != "" && (!errors.As(insertErr, &pgerr) || pgerr.Code != wantCode)) {
					t.Fatalf("%s endpoints %+v: %v, want SQLSTATE %s", target.constraint, pair, insertErr, wantCode)
				}
			}
		}()
	}
}
