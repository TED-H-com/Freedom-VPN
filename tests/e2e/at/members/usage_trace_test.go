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
// It reproduces the lab's binding_not_found response; it cannot prove real
// usage delivery until native discovery produces a member device and interval.
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
	now := time.Now().UTC().Truncate(time.Second)
	event, err := usage.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: l.node, NodeIncarnationId: l.inc, MeterEpochId: uuid.New(), EventSeq: decstr.U64(1), IntervalStart: now.Add(-time.Minute), IntervalEnd: now, MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete, Records: []api.UsageRecord{{MemberId: uuid.MustParse(l.a.member), DeviceId: uuid.New(), BindingRevision: decstr.U64(1), PeriodId: now.Format("2006-01"), UploadBytes: decstr.U64(1), DownloadBytes: decstr.U64(1)}}})
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
}
