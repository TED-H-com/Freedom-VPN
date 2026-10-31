//go:build e2e

package usage

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func (s *liveSuite) currentEvent(db *sql.DB, seq int, node, inc uuid.UUID) handoffEvent {
	var ev handoffEvent
	if err := db.QueryRowContext(s.ctx, `SELECT payload FROM events WHERE seq=?`, seq).Scan(&ev.Raw); err != nil {
		s.t.Fatal(err)
	}
	if err := json.Unmarshal(ev.Raw, &ev.Event); err != nil {
		s.t.Fatal(err)
	}
	if ev.Event.NodeId != node || ev.Event.NodeIncarnationId != inc || uint64(ev.Event.EventSeq) != uint64(seq) {
		s.t.Fatal("D-130 local current event identity/sequence mismatch")
	}
	events := []handoffEvent{ev}
	s.verifyHistoricalEvents(events)
	return events[0]
}

// The expected projection is computed from the immutable canonical event, never
// by casting text in PostgreSQL (which can round rather than truncate).
func (s *liveSuite) intervalProjection(ev handoffEvent) map[string]any {
	e := ev.Event
	start, end := e.IntervalStart.UTC(), e.IntervalEnd.UTC()
	if start.Nanosecond()%1000 != 0 || end.Nanosecond()%1000 != 0 || end.Sub(start) < time.Microsecond {
		s.t.Fatalf("D-130 newly minted seq %s is not a whole-microsecond positive interval", e.EventSeq)
	}
	const query = `SELECT 'usage_events',e.interval_start,e.interval_end FROM usage_events e
 WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3
 UNION ALL SELECT 'usage_samples',s.interval_start,s.interval_end FROM usage_samples s JOIN usage_events e ON e.id=s.event_id
 WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3
 UNION ALL SELECT 'usage_ledger_gaps',g.affected_from,g.affected_until FROM usage_ledger_gaps g JOIN usage_events e ON e.id=g.source_event_id
 WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3`
	rows, err := s.pool.Query(s.ctx, query, e.NodeId, e.NodeIncarnationId, e.EventSeq.String())
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	var stored []map[string]any
	counts := map[string]int{}
	for rows.Next() {
		var table string
		var a, b time.Time
		if err = rows.Scan(&table, &a, &b); err != nil {
			s.t.Fatal(err)
		}
		projected := a.Equal(start.Truncate(time.Microsecond)) && b.Equal(end.Truncate(time.Microsecond))
		if !projected || a.After(b) {
			s.t.Fatalf("D-130 %s seq %s projection/order mismatch", table, e.EventSeq)
		}
		counts[table]++
		stored = append(stored, map[string]any{"table": table, "start": a.UTC(), "end": b.UTC(), "equals_canonical_microsecond_projection": projected, "start_lte_end": !a.After(b), "endpoints_equal": a.Equal(b)})
	}
	if err = rows.Err(); err != nil {
		s.t.Fatal(err)
	}
	gaps := 0
	if e.Gap != nil {
		gaps = 1
	}
	if counts["usage_events"] != 1 || counts["usage_samples"] != len(e.Records) || counts["usage_ledger_gaps"] != gaps {
		s.t.Fatalf("D-130 unexpected stored interval row counts: %v", counts)
	}
	return map[string]any{"event_seq": e.EventSeq, "canonical_start": start, "canonical_end": end, "width_ns": fmt.Sprint(end.Sub(start).Nanoseconds()), "query": query, "stored_rows": stored, "new_event_microsecond_rule": true}
}

func (s *liveSuite) verifyFreshGap(rec *Record, db *sql.DB, node, current uuid.UUID) {
	ev := s.currentEvent(db, 1, node, current)
	rec.Observations["d130_fresh_gap"] = ev
	if ev.Event.Quality != api.UsageQualityGap || ev.Event.Gap == nil || ev.Event.Gap.Reason != api.UsageGapReasonUnknown || len(ev.Event.Records) != 0 {
		s.t.Fatal("D-130 first current event is not the empty unknown-continuity gap")
	}
	rec.Observations["d130_gap_projection"] = s.intervalProjection(ev)
	rec.Observations["d130_gap_ack"] = s.centralStreamACK(node, current)
	var constraints json.RawMessage
	const query = `SELECT jsonb_object_agg(conname,pg_get_constraintdef(oid)) FROM pg_constraint
 WHERE conname IN ('usage_events_interval_check','usage_samples_check','usage_ledger_gaps_check1')`
	if err := s.pool.QueryRow(s.ctx, query).Scan(&constraints); err != nil {
		s.t.Fatal(err)
	}
	var checks map[string]string
	if err := json.Unmarshal(constraints, &checks); err != nil {
		s.t.Fatal(err)
	}
	if len(checks) != 3 {
		s.t.Fatal("D-130 interval constraints missing")
	}
	for _, check := range checks {
		if !strings.Contains(check, " <= ") || !strings.Contains(check, "isfinite") {
			s.t.Fatal("D-130 stored interval finite/nondecreasing constraint missing")
		}
	}
	rec.Observations["d130_stored_constraints"] = map[string]any{"query": query, "checks": checks}
	rec.Observations["d130_scope"] = "New production events from the recorded post-#330 node image. No pre-D-130 image is built or sealed event rewritten. Recovery/retry of already sealed 1 ns events belongs to enrollment follow-up 9; this run does not claim that path."
}

// Replays the actual saved agent batches, including the old ACK 1 result after
// ACK 2 has been reached. No synthetic event or key replaces the agent's request.
func (s *liveSuite) replayCurrentBatches(db *sql.DB, node, current uuid.UUID, client *http.Client) []handoffBatch {
	rows, err := db.QueryContext(s.ctx, `SELECT idempotency_key,body FROM upload_batches ORDER BY created_at`)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	var batches []handoffBatch
	acks := map[string]bool{}
	for rows.Next() {
		var b handoffBatch
		if err = rows.Scan(&b.Key, &b.Body); err != nil {
			s.t.Fatal(err)
		}
		var req api.UsageBatchRequest
		if err = json.Unmarshal(b.Body, &req); err != nil {
			s.t.Fatal(err)
		}
		if req.OriginIncarnationId != nil {
			continue
		}
		if req.NodeId != node || req.NodeIncarnationId != current {
			s.t.Fatal("D-130 saved current batch identity mismatch")
		}
		hash := sha256.Sum256([]byte(b.Key))
		if err = s.pool.QueryRow(s.ctx, `SELECT b.response_body FROM usage_batch_replays b JOIN idempotency_requests i ON i.id=b.id WHERE i.key_hash=$1`, hash[:]).Scan(&b.Stored); err != nil {
			s.t.Fatal(err)
		}
		response := s.agentCallWithClient(client, http.MethodPost, "/api/v1/agent/usage-batches", []byte(b.Body), map[string]string{"Idempotency-Key": b.Key})
		b.Replayed = response.body
		canonical, err := canonjson.Canonicalize(b.Replayed)
		if err != nil {
			s.t.Fatal(err)
		}
		b.ByteEqual, b.CanonicalEqual = bytes.Equal(b.Stored, b.Replayed), bytes.Equal(b.Stored, canonical)
		var result api.UsageBatchResult
		if err = json.Unmarshal(b.Stored, &result); err != nil {
			s.t.Fatal(err)
		}
		if response.status != http.StatusOK || !b.CanonicalEqual || result.NodeId != node || result.NodeIncarnationId != current || len(req.Events) != 1 || len(result.Results) != 1 || result.Results[0].EventSeq != req.Events[0].EventSeq || result.Results[0].Status != api.UsageEventResultStatusAccepted || result.HighestContiguousSeq != req.Events[0].EventSeq {
			s.t.Fatalf("D-130 stored current result mismatch: HTTP %d ACK %s", response.status, result.HighestContiguousSeq)
		}
		acks[result.HighestContiguousSeq.String()] = true
		batches = append(batches, b)
	}
	if err = rows.Err(); err != nil {
		s.t.Fatal(err)
	}
	if len(batches) != 2 || !acks["1"] || !acks["2"] {
		s.t.Fatalf("D-130 expected distinct saved ACK 1/2 batches, got %v", acks)
	}
	return batches
}

func (s *liveSuite) verifyFreshCountedStream(rec *Record, node, current, origin uuid.UUID, originACK, period string, states map[string]string, presenter *http.Client, before api.MemberUsage, beforeInc api.NodeIncarnationList) string {
	// Respect the shipped systemd start limit after deliberate fixture restarts.
	s.t.Log("AT20 waiting beyond the unchanged 120-second systemd start window")
	startAfter := time.Now().Add(121 * time.Second)
	if !s.waitFor(125*time.Second, func() bool { return time.Now().After(startAfter) }) {
		s.t.Fatal("D-130 restart pacing interrupted")
	}
	if err := s.hook.RestoreInstallerServices(s.ctx, "installer-node", states); err != nil {
		s.t.Fatal(err)
	}
	if !s.waitFor(90*time.Second, func() bool {
		var ready bool
		_ = s.pool.QueryRow(s.ctx, `SELECT provisioning_state='ready' AND safety_status='passed' FROM nodes WHERE id=$1`, node).Scan(&ready)
		return ready
	}) {
		s.t.Fatal("D-130 current node did not reach ready")
	}
	publishedAt := time.Now().UTC()
	pub := s.nodeMutation(node, http.MethodPut, "/publication", map[string]any{"publication_intent": "published", "group_ids": []string{s.groupID.String()}, "reason": "TEST ONLY AT20 D-130 current usage"})
	if pub.status != http.StatusAccepted {
		s.t.Fatalf("D-130 publication HTTP %d %s", pub.status, pub.code)
	}
	var observation json.RawMessage
	if !s.waitFor(90*time.Second, func() bool {
		return s.pool.QueryRow(s.ctx, `SELECT report_payload FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 AND received_at>$3 AND safety_status='passed' AND report_payload->>'member_forwarding_enabled'='true' ORDER BY received_at DESC LIMIT 1`, node, current, publishedAt).Scan(&observation) == nil
	}) {
		s.t.Fatal("D-130 current forwarding observation missing")
	}
	rec.Observations["d130_current_serving_observation"] = observation
	var exit uuid.UUID
	if err := s.pool.QueryRow(s.ctx, `SELECT id FROM exits WHERE node_id=$1`, node).Scan(&exit); err != nil {
		s.t.Fatal(err)
	}
	originIP, _ := s.originAndSNAT("installer-node")
	if _, ok := s.selectExitRoute("member", exitNetworkID(exit), originIP); !ok {
		s.t.Fatal("D-130 current exit selection failed")
	}
	traffic := s.runCountedTransfer("member", "installer-node", s.deviceA, 1)
	rec.Observations["d130_counted_transfer"] = traffic
	if !traffic.Counted {
		s.t.Fatal("D-130 current transfer did not count")
	}
	if !s.waitFor(90*time.Second, func() bool {
		var ack string
		if s.pool.QueryRow(s.ctx, `SELECT highest_contiguous_seq::text FROM usage_checkpoints WHERE node_id=$1 AND incarnation_id=$2`, node, current).Scan(&ack) != nil || ack != "2" {
			return false
		}
		snap, err := s.hook.UsageLedgerSnapshot(s.ctx, "installer-node")
		return err == nil && snap.AckedThrough == "2" && snap.LastEventSeq == "2"
	}) {
		s.t.Fatal("D-130 counted current event did not reach local and central ACK 2")
	}
	s.stopInstaller()
	rec.Observations["d130_current_cli_after_counted"] = s.originCLI("installer-node")
	db := s.handoffLedgerCopy("installer-node")
	gap, counted := s.currentEvent(db, 1, node, current), s.currentEvent(db, 2, node, current)
	rec.Observations["d130_counted_event"] = counted
	rec.Observations["d130_counted_projection"] = s.intervalProjection(counted)
	if counted.Event.Quality != api.UsageQualityComplete || len(counted.Event.Records) != 1 {
		s.t.Fatal("D-130 expected exactly one original member/device record")
	}
	r := counted.Event.Records[0]
	if r.MemberId.String() != s.memberA.memberID || r.DeviceId != s.deviceA || r.PeriodId != period || r.UploadBytes != traffic.DeltaUp || r.DownloadBytes != traffic.DeltaDown {
		s.t.Fatal("D-130 counted event differs from exact measured direction counters or original member/device/period")
	}
	after := s.memberPeriodUsage(period)
	up, down := r.UploadBytes.String(), r.DownloadBytes.String()
	if after.Totals.UploadBytes.String() != decimalSum(before.Totals.UploadBytes.String(), up) || after.Totals.DownloadBytes.String() != decimalSum(before.Totals.DownloadBytes.String(), down) || after.Totals.TotalBytes.String() != decimalSum(before.Totals.TotalBytes.String(), decimalSum(up, down)) {
		s.t.Fatal("D-130 member period before + measured bytes != after")
	}
	afterInc := s.incarnationView(node)
	if len(afterInc.Items) != 2 || len(beforeInc.Items) != 2 {
		s.t.Fatal("D-130 incarnation views missing")
	}
	for _, row := range afterInc.Items {
		if row.IncarnationId == current && (row.Usage.UploadBytes.String() != up || row.Usage.DownloadBytes.String() != down) {
			s.t.Fatal("D-130 current incarnation attribution differs from measured counters")
		}
		for _, previous := range beforeInc.Items {
			if row.IncarnationId == origin && previous.IncarnationId == origin && !sameIncarnationUsage(row.Usage, previous.Usage) {
				s.t.Fatal("D-130 current traffic changed retired origin usage")
			}
		}
	}
	rec.Observations["d130_member_period_before"] = before
	rec.Observations["d130_member_period_after"] = after
	rec.Observations["d130_incarnations_after"] = afterInc
	rec.Observations["d130_exact_counter_and_attribution_equality"] = true
	rec.Observations["d130_current_batches_and_stored_replays"] = s.replayCurrentBatches(db, node, current, presenter)
	s.verifyHistoricalEvents([]handoffEvent{gap, counted})
	replayed := s.memberPeriodUsage(period)
	if replayed.Totals != after.Totals || s.centralStreamACK(node, current) != "2" || s.centralStreamACK(node, origin) != originACK {
		s.t.Fatal("D-130 replay changed totals or either stream ACK")
	}
	rec.Observations["d130_member_period_after_replay"] = replayed
	rec.Observations["d130_current_ack_after_replays"] = "2"
	rec.Observations["d130_origin_ack_after_current_replays"] = originACK
	return fmt.Sprintf("D-130 fresh unknown gap accepted at ACK 1, followed by counted event 2 at ACK 2: measured %s upload + %s download = %s bytes, exactly attributed to the original member/device/period once. All stored event/sample/gap endpoints equal the canonical UTC microsecond projection; same-key replays retain stored ACK 1 and ACK 2 results without changing attribution.", up, down, decimalSum(up, down))
}
