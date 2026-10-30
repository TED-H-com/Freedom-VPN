//go:build e2e

package members

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// AT02 diagnostic: a deliberately unknown device is only a negative control.
// Then use the real projected device and a contained interval. This proves
// receipt/binding validation only: the synthetic bytes are not metered traffic.
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
	var seq decstr.U64
	if l.pool.QueryRow(l.ctx, `SELECT (coalesce(max(event_seq),0)+1)::text FROM usage_events WHERE node_id=$1 AND incarnation_id=$2`, l.node, l.inc).Scan(&seq) != nil {
		r.block("usage sequence read unavailable")
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	event, err := usage.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: l.node, NodeIncarnationId: l.inc, MeterEpochId: uuid.New(), EventSeq: seq, IntervalStart: now.Add(-time.Minute), IntervalEnd: now, MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete, Records: []api.UsageRecord{{MemberId: uuid.MustParse(l.a.member), DeviceId: uuid.New(), BindingRevision: decstr.U64(1), PeriodId: now.Format("2006-01"), UploadBytes: decstr.U64(1), DownloadBytes: decstr.U64(1)}}})
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
	var device uuid.UUID
	var revision decstr.U64
	var from, until time.Time
	err = l.pool.QueryRow(l.ctx, `SELECT b.device_id,b.binding_revision::text,b.valid_from,b.valid_until FROM device_bindings b JOIN member_devices d ON d.id=b.device_id WHERE b.member_id=$1 AND d.status='authorized' AND b.valid_until>clock_timestamp() ORDER BY b.valid_from LIMIT 1`, l.a.member).Scan(&device, &revision, &from, &until)
	if err != nil {
		r.block("Usage positive control: no product-projected member binding available.")
		return
	}
	end := time.Now().UTC().Truncate(time.Second)
	start := end.Add(-time.Second)
	if start.Before(from) {
		start = from
	}
	if !end.After(start) || end.After(until) {
		r.block("Usage positive control has no elapsed interval within the binding")
		return
	}
	event.EventSeq = seq
	event.IntervalStart, event.IntervalEnd = start, end
	event.Records[0].DeviceId = device
	event.Records[0].BindingRevision = revision
	event, err = usage.WithComputedHash(event)
	if err != nil {
		r.block("positive usage canonical hash unavailable")
		return
	}
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
	r.check("real projected binding usage replay", "accepted then duplicate; one receipt and sample", fmt.Sprintf("first=%s; replay=%s; sequence=%s; device_revision=%s", statuses[0], statuses[1], seq.String(), revision.String()), "POST node-mTLS synthetic usage for real projected binding, interval inside [valid_from,valid_until), twice", statuses[0] == "accepted" && statuses[1] == "duplicate")
	receipts := l.count(`SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2 AND event_seq=$3::numeric`, l.node, l.inc, seq.String())
	samples := l.count(`SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3::numeric`, l.node, l.inc, seq.String())
	r.check("positive control durable evidence", "one receipt, one sample", fmt.Sprintf("receipts=%d; samples=%d", receipts, samples), "SELECT usage_events and usage_samples after duplicate", receipts == 1 && samples == 1)

}
