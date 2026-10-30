//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func (p *liveProbe) classMetrics() map[string]int {
	raw, err := p.docker("control", "wget", "-q", "-T", "3", "-O", "-", "http://127.0.0.1:9091/metrics")
	if err != nil {
		p.t.Fatal("class metrics unavailable")
	}
	result := map[string]int{}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "fvpn_control_rate_limited_total{") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			p.t.Fatal("class metrics malformed")
		}
		n, e := strconv.ParseFloat(parts[1], 64)
		if e != nil {
			p.t.Fatal("class metrics value")
		}
		result[parts[0]] = int(n)
	}
	return result
}

// Burst requests deliberately avoid mutating observation reports. An invalid
// observation reaches schema validation (422) until the authenticated principal
// bucket refuses it (429). That proves middleware order, not agent telemetry.
func (p *liveProbe) classBurst(name, class, limiter string, burst int, request func(int) liveReply) map[string]any {
	p.t.Logf("D-060 class %s: refill and measure burst %d", name, burst)
	time.Sleep(6 * time.Second)
	before := p.classMetrics()
	records := []map[string]any{}
	started := time.Now()
	first := 0
	retry := ""
	code := ""
	for i := 1; i <= burst+2; i++ {
		r := request(i)
		records = append(records, map[string]any{"index": i, "utc": time.Now().UTC(), "status": r.status, "code": r.code, "retry_after": r.header.Get("Retry-After")})
		if r.status == 429 {
			first = i
			retry = r.header.Get("Retry-After")
			code = r.code
			break
		}
	}
	elapsed := time.Since(started)
	after := p.classMetrics()
	label := `fvpn_control_rate_limited_total{class="` + class + `",limiter="` + limiter + `"}`
	delta := after[label] - before[label]
	passed := first == burst+1 && code == "RATE_LIMITED" && retry == "1" && delta == 1 && elapsed < 500*time.Millisecond
	out := map[string]any{"expected_first_rejection": burst + 1, "actual_first_rejection": first, "first_retry_after": retry, "first_error_code": code, "elapsed_seconds": elapsed.Seconds(), "requests": records, "metrics_before": before, "metrics_after": after, "expected_metric": label, "metric_delta": delta, "passed": passed}
	if !passed {
		p.t.Errorf("rate class %s: first=%d retry=%s metric_delta=%d elapsed=%s", name, first, retry, delta, elapsed)
	}
	return out
}

func (p *liveProbe) rateClasses(node *liveSession, b api.EnrollResponse, members []*liveSession, devices [][]uuid.UUID) {
	out := map[string]any{"settings": "unmodified D-060 defaults after #145; ingress 20/s burst60 per class/address; heartbeat and lease 1/s burst5; usage and anonymous 2/s burst10", "source": "one host transport address; node B certificate; real agent uses node A identity", "observation_payload": "{} deliberately schema-invalid: no projected observation rows written"}
	p.metadata["rate_classes"] = out
	out["heartbeat"] = p.classBurst("observations", "heartbeat", "subject", 5, func(int) liveReply {
		return p.call(node, "POST", liveAgent+"/api/v1/agent/observations", json.RawMessage(`{}`), nil)
	})
	renewal := api.LeaseRenewalRequest{NodeId: b.NodeId, NodeIncarnationId: b.NodeIncarnationId, RenewalRequestId: uuid.New(), CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()}
	out["lease"] = p.classBurst("lease-renewals", "lease", "subject", 5, func(int) liveReply {
		return p.call(node, "POST", liveAgent+"/api/v1/agent/lease-renewals", renewal, map[string]string{"Idempotency-Key": ""})
	})
	anon := &liveSession{client: p.client()}
	enroll := func(int) liveReply {
		return p.call(anon, "POST", "/api/v1/agent/enroll", json.RawMessage(`{}`), map[string]string{"Authorization": "Enrollment TEST-ONLY-invalid"})
	}
	webhook := func(int) liveReply {
		return p.call(anon, "POST", "/api/v1/webhooks/stripe", json.RawMessage(`{}`), nil)
	}
	out["enrollment"] = p.classBurst("enrollment", "anonymous", "address", 10, enroll)
	out["webhook"] = p.classBurst("provider webhook", "anonymous", "address", 10, webhook)
	out["anonymous_shared"] = p.classBurst("mixed anonymous endpoints", "anonymous", "address", 10, func(i int) liveReply {
		if i%2 == 1 {
			return enroll(i)
		}
		return webhook(i)
	})
	network, err := exec.CommandContext(p.t.Context(), "docker", "network", "inspect", "fvpn-w4-secops", "--format", "{{.EnableIPv6}}").Output()
	out["ipv6"] = map[string]any{"network_read_ok": err == nil, "lab_network_ipv6": strings.TrimSpace(string(network)), "status": "needs operator run", "reason": "This live transport dials assigned IPv4 loopback ports. No IPv6-origin experiment or host/network changes; owner component tests cover /64 and invalid RemoteAddr."}
	if len(p.bindings) == 0 || len(devices[0]) == 0 {
		out["usage"] = map[string]any{"status": "blocked", "reason": "No product-admitted binding"}
		return
	}
	binding := p.bindings[devices[0][0]]
	now := time.Now().UTC().Truncate(time.Second)
	event, e := usage.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: b.NodeId, NodeIncarnationId: b.NodeIncarnationId, MeterEpochId: uuid.New(), EventSeq: 1, IntervalStart: now.Add(-time.Second), IntervalEnd: now, MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete, Records: []api.UsageRecord{{MemberId: uuid.MustParse(members[0].member), DeviceId: devices[0][0], BindingRevision: binding.BindingRevision, PeriodId: api.PeriodId(now.Format("2006-01")), UploadBytes: 100, DownloadBytes: 200}}})
	if e != nil {
		p.t.Fatal("usage class hash")
	}
	batch := api.UsageBatchRequest{NodeId: b.NodeId, NodeIncarnationId: b.NodeIncarnationId, LedgerEpoch: presence.NewNullNullable[uuid.UUID](), Events: []api.UsageEvent{event}}
	raw, e := json.Marshal(batch)
	if e != nil {
		p.t.Fatal("usage class marshal")
	}
	results := []api.UsageBatchResult{}
	out["usage"] = p.classBurst("usage-batches", "usage", "subject", 10, func(int) liveReply {
		r := p.call(node, "POST", liveAgent+"/api/v1/agent/usage-batches", raw, map[string]string{"Idempotency-Key": ""})
		if r.status == 200 {
			var ack api.UsageBatchResult
			if json.Unmarshal(r.body, &ack) == nil {
				results = append(results, ack)
			}
		}
		return r
	})
	out["usage_acknowledgements"] = results
	sizes := []map[string]any{}
	for _, n := range []int{(8 << 20) - 1, 8 << 20, (8 << 20) + 1} {
		time.Sleep(time.Second)
		padded := append(append([]byte{}, raw...), bytes.Repeat([]byte(" "), n-len(raw))...)
		r := p.call(node, "POST", liveAgent+"/api/v1/agent/usage-batches", padded, map[string]string{"Idempotency-Key": ""})
		want := 200
		if n > 8<<20 {
			want = 413
		}
		sizes = append(sizes, map[string]any{"bytes": n, "valid_json": json.Valid(padded), "expected_status": want, "actual_status": r.status, "code": r.code, "passed": r.status == want, "fixture": "identical valid one-event batch padded with legal JSON trailing whitespace; optional idempotency header omitted"})
		p.check("usage body size "+strconv.Itoa(n), r, want, "")
	}
	out["usage_body_limits"] = sizes
	var rows int
	err = p.db.QueryRow(p.t.Context(), `SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2 AND meter_epoch_id=$3`, b.NodeId, b.NodeIncarnationId, event.MeterEpochId).Scan(&rows)
	out["ledger"] = map[string]any{"read_ok": err == nil, "expected_rows": 1, "actual_rows": rows, "synthetic": true, "kernel_flow": false}
	if err != nil || rows != 1 {
		p.t.Error("rate probe duplicate ledger accounting")
	}
}
