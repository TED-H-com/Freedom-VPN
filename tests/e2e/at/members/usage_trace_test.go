//go:build e2e

package members

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// AT02 diagnostic: a deliberately unknown device is only a negative control.
// Replay a real accepted event instead of allocating the active meter's next
// sequence. Only the rejected unknown-device negative is synthetic.
func (l *live) usageTrace(r *record) {
	cert, err := l.inside("node", "cat", "/var/lib/vpn-node/identity/cert.pem")
	if err != nil {
		r.block("node certificate unavailable for usage negative control")
		return
	}
	key, err := l.inside("node", "cat", "/var/lib/vpn-node/identity/key.pem")
	if err != nil {
		r.block("node key unavailable for usage negative control")
		return
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		r.block("node TLS identity invalid")
		return
	}
	transport := l.transport.Clone()
	defer transport.CloseIdleConnections()
	transport.TLSClientConfig.Certificates = []tls.Certificate{pair}
	// This diagnostic follows all continuous-PID revoke/re-grant observations.
	// Stop through the lab supervisor so the rejected negative cannot race a
	// live meter upload. Resume the same selected release on every exit path.
	selected, err := l.inside("node", "cat", "/var/lib/vpn-node/e2e/agent-selected")
	release := strings.TrimSpace(string(selected))
	if err != nil || (release != "/usr/local" && !strings.HasPrefix(release, "/opt/vpn-node/releases/")) {
		r.block("usage diagnostic selected release unavailable")
		return
	}
	leaseBefore := l.storedLeaseReadback("node", false)
	stopStarted := time.Now().UTC()
	stopRaw, stopReadErr := l.inside("node", "/tmp/at-members-probe", "stop-observe", "bounded")
	var stopEvidence map[string]any
	stopDecoded := json.Unmarshal(stopRaw, &stopEvidence) == nil
	r.jsonCheck("r18 M-2 SIGTERM to process exit", "cooperative process exit observed within 13 seconds; retain parent wait receipt", stopEvidence, "owned-node helper: pause supervisor, SIGTERM exact PID, poll process identity; original Service stop/start follows", stopReadErr == nil && stopDecoded && stopEvidence["available"] == true && stopEvidence["within_13s"] == true && stopEvidence["cooperative_exit"] == true)
	err = l.hook.Service(l.ctx, "stop", "")
	stopEnded := time.Now().UTC()
	leaseStopped := l.storedLeaseReadback("node", false)
	l.recordAgentTransition(r, "usage stop", stopStarted, stopEnded, leaseBefore, leaseStopped, err)
	r.check("usage diagnostic supervisor stop", "agent reaped before rejected sequence probe", fmt.Sprintf("stopped=%t", err == nil), "Stack.Service(stop); after the continuous-agent traffic assertions", err == nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 30*time.Second)
		defer cancel()
		resumeStarted := time.Now().UTC()
		resumed := l.hook.Service(ctx, "start", release)
		resumeEnded := time.Now().UTC()
		l.recordAgentTransition(r, "usage resume", resumeStarted, resumeEnded, leaseStopped, l.storedLeaseReadback("node", false), resumed)
		r.check("usage diagnostic supervisor resume", "same selected release resumes", fmt.Sprintf("resumed=%t", resumed == nil), "Stack.Service(start, original selected release); no agent state/sequence changes", resumed == nil)
	}()
	if err != nil {
		return
	}
	conflictsBefore := l.count(`SELECT count(*) FROM usage_event_conflicts WHERE node_id=$1 AND incarnation_id=$2`, l.node, l.inc)
	var seq decstr.U64
	if l.pool.QueryRow(l.ctx, `SELECT (coalesce(max(event_seq),0)+1)::text FROM usage_events WHERE node_id=$1 AND incarnation_id=$2`, l.node, l.inc).Scan(&seq) != nil {
		r.block("usage sequence read unavailable")
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-time.Minute)
	event, err := usage.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: l.node, NodeIncarnationId: l.inc, MeterEpochId: uuid.New(), EventSeq: seq, IntervalStart: start, IntervalEnd: now, MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete, Records: []api.UsageRecord{{MemberId: uuid.MustParse(l.a.member), DeviceId: uuid.New(), BindingRevision: decstr.U64(1), PeriodId: start.Format("2006-01"), UploadBytes: decstr.U64(1), DownloadBytes: decstr.U64(1)}}})
	if err != nil {
		r.block("usage negative-control canonical hash unavailable")
		return
	}
	got := l.request(&browser{client: &http.Client{Transport: transport, Timeout: 15 * time.Second}}, "POST", "https://agent.example.invalid:9443/api/v1/agent/usage-batches", api.UsageBatchRequest{NodeId: l.node, NodeIncarnationId: l.inc, LedgerEpoch: presence.NewNullNullable[uuid.UUID](), Events: []api.UsageEvent{event}}, commandHeaders())
	var body struct {
		Results []struct {
			Status string `json:"status"`
			Error  *struct {
				Code   string           `json:"code"`
				Fields []api.FieldError `json:"fields"`
			} `json:"error"`
		} `json:"results"`
	}
	_ = json.Unmarshal(got.body, &body)
	// Emit only known protocol tokens, not payload identifiers or raw errors.
	known := false
	for _, result := range body.Results {
		if result.Error != nil && result.Error.Code == "VALIDATION_FAILED" {
			raw, _ := json.Marshal(result.Error.Fields)
			known = bytes.Contains(raw, []byte("binding_not_found"))
		}
	}
	r.check("mTLS usage binding negative control", "binding_not_found; no made-up binding accepted", fmt.Sprintf("HTTP=%d; per_event_results=%d; binding_not_found=%t", got.status, len(body.Results), known), "POST /agent/usage-batches with real node mTLS and explicitly unknown device", got.status == 200 && known)
	// The accepted payload is canonical JSON without its hash field. Rebuild
	// that field exactly; no new event, meter epoch or sequence is fabricated.
	var payload []byte
	var storedHash string
	var expectedSamples int
	err = l.pool.QueryRow(l.ctx, `SELECT p.payload,encode(e.payload_hash,'hex'),e.record_count FROM usage_events e JOIN usage_event_payloads p ON p.event_id=e.id WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.record_count>0 AND EXISTS(SELECT 1 FROM usage_samples s WHERE s.event_id=e.id AND s.member_id=$3) ORDER BY e.event_seq DESC LIMIT 1`, l.node, l.inc, l.a.member).Scan(&payload, &storedHash, &expectedSamples)
	if err != nil || json.Unmarshal(payload, &event) != nil {
		r.block("Real metered usage payload unavailable for duplicate replay")
		return
	}
	event, err = usage.WithComputedHash(event)
	if err != nil {
		r.block("Real metered usage hash unavailable")
		return
	}
	seq = event.EventSeq
	before := l.count(`SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3::numeric`, l.node, l.inc, seq.String())
	r.check("real metered event receipt", "existing accepted event with exact hash and samples", fmt.Sprintf("sequence=%s; hash_matches=%t; samples=%d; record_count=%d", seq, event.PayloadSha256 == storedHash, before, expectedSamples), "read production usage receipt and canonical payload; no synthetic positive event", event.PayloadSha256 == storedHash && before == expectedSamples && before > 0)
	batch := api.UsageBatchRequest{NodeId: l.node, NodeIncarnationId: l.inc, LedgerEpoch: presence.NewNullNullable[uuid.UUID](), Events: []api.UsageEvent{event}}
	statuses := []string{}
	for range 2 {
		got = l.request(&browser{client: &http.Client{Transport: transport, Timeout: 15 * time.Second}}, "POST", "https://agent.example.invalid:9443/api/v1/agent/usage-batches", batch, commandHeaders())
		body.Results = nil
		_ = json.Unmarshal(got.body, &body)
		state := "no_result"
		if len(body.Results) == 1 {
			state = body.Results[0].Status
		}
		statuses = append(statuses, state)
	}
	r.check("real metered usage replay", "duplicate twice; existing receipt is unchanged", fmt.Sprintf("first=%s; replay=%s; sequence=%s", statuses[0], statuses[1], seq.String()), "POST node-mTLS exact accepted canonical event twice", statuses[0] == "duplicate" && statuses[1] == "duplicate")
	receipts := l.count(`SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2 AND event_seq=$3::numeric`, l.node, l.inc, seq.String())
	samples := l.count(`SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3::numeric`, l.node, l.inc, seq.String())
	r.check("positive control durable evidence", "one receipt, unchanged sample count", fmt.Sprintf("receipts=%d; samples_before=%d; samples_after=%d", receipts, before, samples), "SELECT usage_events and usage_samples after duplicate", receipts == 1 && samples == before)

	conflictsAfter := l.count(`SELECT count(*) FROM usage_event_conflicts WHERE node_id=$1 AND incarnation_id=$2`, l.node, l.inc)
	r.check("usage diagnostic adds no sequence conflict", "conflict count unchanged", fmt.Sprintf("before=%d; after=%d", conflictsBefore, conflictsAfter), "read usage_event_conflicts for the real meter stream", conflictsBefore == conflictsAfter)
}
