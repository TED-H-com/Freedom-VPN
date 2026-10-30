//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// Fixed five-minute dev-host subset: 20 portal readers, native HTTPS flows,
// actual agent sampling, two API capacity-policy changes. It is not the unrun
// two-node, UDP/forced-relay or candidate-hardware profile (§19 AT31).
func (p *liveProbe) countedCapacity(admin *liveSession, node uuid.UUID, members []*liveSession, peers []flowPeer) {
	out := map[string]any{"status": "blocked", "requested_seconds": 300, "warmup_seconds": 60, "portal_members": 20, "native_peers": len(peers), "hardware": "shared dev host; not candidate hardware", "relay_forced": false, "udp": false, "source": "real kernel admission and real agent sampling; synthetic node B ledger excluded"}
	p.metadata["counted_capacity"] = out
	origin, _ := p.originAddress()
	before := map[string]map[string]decstr.U64{}
	proofs := []map[string]any{}
	for _, peer := range peers {
		r := p.selectedFlow(peer)
		proofs = append(proofs, r)
		if r["counted"] != true {
			out["blocker"] = "at least one cohort peer lacks counted flow"
			out["initial_flows"] = proofs
			return
		}
	}
	out["initial_flows"] = proofs
	p.t.Logf("AT31 counted subset: %d native peers, 20 readers, fixed 60s warmup + 300s measure", len(peers))
	time.Sleep(60 * time.Second)
	for _, peer := range peers {
		b, _ := p.deviceBytes(peer.Device)
		before[peer.Device.String()] = b
	}
	started := time.Now()
	var mu sync.Mutex
	var wg sync.WaitGroup
	latency := []float64{}
	codes := map[int]int{}
	wrong := 0
	flows := map[string]map[string]int{}
	for _, m := range members {
		wg.Go(func() {
			for time.Since(started) < 300*time.Second {
				tick := time.Now()
				r := p.call(m, "GET", "/api/v1/member/me", nil, nil)
				elapsed := float64(time.Since(tick).Microseconds()) / 1000
				mu.Lock()
				latency = append(latency, elapsed)
				codes[r.status]++
				if r.status == 200 && !bytes.Contains(r.body, []byte(m.member)) {
					wrong++
				}
				mu.Unlock()
				time.Sleep(time.Until(tick.Add(time.Second)))
			}
		})
	}
	for _, peer := range peers {
		wg.Go(func() {
			counts := map[string]int{}
			for time.Since(started) < 300*time.Second {
				tick := time.Now()
				raw, e := p.docker(peer.Service, "wget", "--server-response", "--timeout=2", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "-O", "/dev/null", "https://"+origin+"/healthz")
				if e == nil && bytes.Contains(raw, []byte("200 OK")) {
					counts["http_200"]++
				} else {
					counts["failed"]++
				}
				time.Sleep(time.Until(tick.Add(time.Second)))
			}
			mu.Lock()
			flows[peer.Device.String()] = counts
			mu.Unlock()
		})
	}
	resources := []map[string]any{}
	policies := []map[string]any{}
	nextChange := 0
	for time.Since(started) < 300*time.Second {
		elapsed := time.Since(started).Seconds()
		if nextChange < 2 && elapsed >= []float64{60, 180}[nextChange] {
			path := "/api/v1/admin/nodes/" + node.String()
			r := p.call(admin, "GET", path, nil, nil)
			capacity := []int{190000000, 200000000}[nextChange]
			mark := time.Now()
			var previous string
			_ = p.db.QueryRow(p.t.Context(), `SELECT desired_config_generation::text FROM nodes WHERE id=$1`, node).Scan(&previous)
			changed := p.call(admin, "PATCH", path, map[string]any{"capacity": map[string]any{"down_bps": capacity, "up_bps": capacity, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}, map[string]string{"If-Match": r.header.Get("ETag")})
			ack := false
			var desired, applied string
			for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
				e := p.db.QueryRow(p.t.Context(), `SELECT desired_config_generation::text,applied_config_generation::text FROM nodes WHERE id=$1`, node).Scan(&desired, &applied)
				if e == nil && desired == applied && desired != previous {
					ack = true
					break
				}
				time.Sleep(time.Second)
			}
			policies = append(policies, map[string]any{"offset_seconds": elapsed, "http_status": changed.status, "desired_generation": desired, "applied_generation": applied, "ack_seconds": time.Since(mark).Seconds(), "ack_within_60s": ack, "counted_after": p.selectedFlow(peers[0])})
			nextChange++
		}
		raw, e := exec.CommandContext(p.t.Context(), "docker", "stats", "--no-stream", "--format", "{{json .}}", livePrefix+"control", livePrefix+"worker", livePrefix+"postgres", livePrefix+"node").Output()
		if e == nil {
			for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
				var v map[string]any
				if json.Unmarshal(line, &v) == nil {
					delete(v, "ID")
					delete(v, "Container")
					v["elapsed_seconds"] = elapsed
					resources = append(resources, v)
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
	wg.Wait()
	sort.Float64s(latency)
	after := map[string]map[string]decstr.U64{}
	deltas := map[string]map[string]decstr.U64{}
	for _, peer := range peers {
		b, _ := p.deviceBytes(peer.Device)
		key := peer.Device.String()
		after[key] = b
		delta := map[string]decstr.U64{}
		for _, k := range []string{"upload", "download"} {
			if b[k] >= before[key][k] {
				delta[k] = b[k] - before[key][k]
			}
		}
		deltas[key] = delta
	}
	out["started_at"] = started.UTC()
	out["duration_seconds"] = time.Since(started).Seconds()
	out["read_status_counts"] = codes
	out["wrong_member_responses"] = wrong
	out["flow_requests"] = flows
	out["before_bytes"] = before
	out["after_bytes"] = after
	out["admitted_deltas"] = deltas
	out["resources"] = resources
	out["policy_changes"] = policies
	if len(latency) > 0 {
		out["read_p50_ms"] = latency[len(latency)/2]
		out["read_p95_ms"] = latency[(len(latency)*95+99)/100-1]
		out["read_p99_ms"] = latency[(len(latency)*99+99)/100-1]
		out["reads"] = len(latency)
	}
	out["status"] = "measured subset; full AT31 remains blocked"
	out["usage_visibility"] = p.usageVisibility(node, peers)
	if wrong > 0 {
		p.t.Error("counted load returned another member's data")
	}
}
func (p *liveProbe) usageVisibility(node uuid.UUID, peers []flowPeer) map[string]any {
	rows := []map[string]any{}
	deadline := time.Now().Add(120 * time.Second)
	for _, peer := range peers {
		var end time.Time
		var deviceBytes string
		e := p.db.QueryRow(p.t.Context(), `SELECT max(s.interval_end),coalesce(sum(s.upload_bytes+s.download_bytes),0)::text FROM usage_samples s JOIN usage_events e ON e.id=s.event_id WHERE e.node_id=$1 AND s.device_id=$2`, node, peer.Device).Scan(&end, &deviceBytes)
		row := map[string]any{"device_id": peer.Device, "sample_read_ok": e == nil, "device_ledger_bytes": deviceBytes, "sample_interval_end": end.UTC(), "visible": false}
		rows = append(rows, row)
		for e == nil && time.Now().Before(deadline) {
			r := p.call(peer.Member, "GET", "/api/v1/member/usage", nil, nil)
			var usage api.MemberUsage
			valid := r.status == 200 && json.Unmarshal(r.body, &usage) == nil
			row["http_status"] = r.status
			if valid {
				row["usage"] = usage
				asof, err := usage.AggregateAsOf.Get()
				if err == nil && !asof.Before(end) && usage.Totals.TotalBytes > 0 {
					row["visible"] = true
					row["observed_latency_upper_bound_seconds"] = time.Since(end).Seconds()
					row["within_120_seconds"] = time.Since(end) <= 120*time.Second
					break
				}
			}
			time.Sleep(time.Second)
		}
	}
	var ledger []byte
	err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('events',count(*),'highest_seq',max(event_seq)::text,'sample_records',coalesce(sum(record_count),0)::text,'actual_samples',(SELECT count(*) FROM usage_samples WHERE event_id IN (SELECT id FROM usage_events WHERE node_id=$1)),'checkpoint',(SELECT jsonb_build_object('highest_contiguous_seq',highest_contiguous_seq::text,'max_seen_seq',max_seen_seq::text) FROM usage_checkpoints WHERE node_id=$1 LIMIT 1)) FROM usage_events WHERE node_id=$1`, node).Scan(&ledger)
	return map[string]any{"scope": "actual node samples and member usage API, watermark-to-read upper bound; excludes node B synthetic ledger; browser rendering unmeasured", "rows": rows, "ledger_read_ok": err == nil, "ledger": json.RawMessage(ledger)}
}
