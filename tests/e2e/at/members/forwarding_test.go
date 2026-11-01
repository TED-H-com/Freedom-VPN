//go:build e2e

package members

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type counterObservation struct {
	Available         bool                  `json:"available"`
	Generation        decstr.U64            `json:"generation"`
	DeviceUpload      decstr.U64            `json:"device_upload_bytes"`
	DeviceDownload    decstr.U64            `json:"device_download_bytes"`
	MemberAcceptRules int                   `json:"member_accept_rules"`
	Counters          map[string]decstr.U64 `json:"diagnostic_bytes"`
}
type forwardingObservation struct {
	Node                string                     `json:"node_container_role"`
	NodeIncarnation     string                     `json:"node_incarnation"`
	OriginVerified      bool                       `json:"origin_verified"`
	OriginRequests      int                        `json:"origin_requests"`
	OriginSNATMatches   bool                       `json:"origin_snat_matches"`
	SourceAT            string                     `json:"source_at,omitempty"`
	ProbeSHA256         string                     `json:"probe_sha256"`
	At                  string                     `json:"at"`
	FirstCountedByteAt  string                     `json:"first_counted_byte_at,omitempty"`
	Member              string                     `json:"member_container_role"`
	RouteSelected       bool                       `json:"route_selected"`
	RouteBefore         json.RawMessage            `json:"route_before"`
	RouteAfter          json.RawMessage            `json:"route_after"`
	PeerStatus          json.RawMessage            `json:"peer_status"`
	Before              counterObservation         `json:"node_before"`
	After               counterObservation         `json:"node_after"`
	Packets             map[string]json.RawMessage `json:"packet_observations"`
	Transfer            json.RawMessage            `json:"transfer"`
	Counted             bool                       `json:"counted"`
	FirstUnconfirmedHop string                     `json:"first_unconfirmed_hop"`
}

// Readiness is evidence, not a forwarding assertion. The intended device is
// mandatory for positive probes; generic callers only wait for serving.
func (l *live) forwardingGate(r *record, devices ...uuid.UUID) bool {
	device := uuid.Nil
	if len(devices) > 0 {
		device = devices[0]
	}
	var raw []byte
	var joined json.RawMessage
	stage := "central_readiness"
	var readerError string
	ready := l.wait(90*time.Second, func() bool {
		stage = "central_readiness"
		readerError = ""
		err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('desired',n.desired_config_generation::text,'applied',n.applied_config_generation::text,'lease_id',s.id,'lease_generation',s.policy_generation::text,'member_count',jsonb_array_length(s.envelope->'signed'->'members'),'expires_at',s.expires_at,'observation',o.report_payload,'device_id',($2::uuid)::text)
 FROM nodes n JOIN LATERAL (SELECT * FROM node_observations WHERE node_id=n.id AND incarnation_id=n.current_incarnation_id ORDER BY received_at DESC LIMIT 1) o ON true
 JOIN serving_leases s ON s.id::text=o.report_payload->'lease'->>'lease_id' AND s.node_id=n.id AND s.incarnation_id=n.current_incarnation_id
 WHERE n.id=$1 AND n.desired_config_generation=n.applied_config_generation AND s.policy_generation=n.applied_config_generation AND o.applied_generation=n.applied_config_generation
 AND s.expires_at>clock_timestamp() AND o.received_at>clock_timestamp()-interval '15 seconds'
 AND o.report_payload->>'member_forwarding_enabled'='true' AND o.report_payload->'forwarding_block_reasons'='[]'::jsonb
 AND jsonb_array_length(s.envelope->'signed'->'members')>0
 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR EXISTS (SELECT 1 FROM device_bindings b JOIN member_devices d ON d.id=b.device_id JOIN serving_lease_members m ON m.lease_id=s.id AND m.member_id=b.member_id WHERE b.device_id=$2 AND d.status='authorized' AND b.valid_from<=clock_timestamp() AND b.valid_until>clock_timestamp() AND m.valid_from<=clock_timestamp() AND m.expires_at>clock_timestamp()))`, l.node, device).Scan(&raw)
		if err != nil {
			return false
		}
		var state struct {
			LeaseID     uuid.UUID             `json:"lease_id"`
			Generation  decstr.U64            `json:"applied"`
			Observation api.ObservationReport `json:"observation"`
		}
		stage = "accepted_report_decode"
		if json.Unmarshal(raw, &state) != nil {
			return false
		}
		stage = "local_binding"
		if device != uuid.Nil {
			body, e := l.inside(l.nodeContainer(), "cat", "/var/lib/vpn-node/state/desired-state.json")
			var desired struct {
				Signed struct {
					Generation decstr.U64 `json:"config_generation"`
					Bindings   []struct {
						Device uuid.UUID `json:"device_id"`
					} `json:"device_bindings"`
				} `json:"signed"`
			}
			if e != nil || json.Unmarshal(body, &desired) != nil || desired.Signed.Generation != state.Generation {
				return false
			}
			found := false
			for _, b := range desired.Signed.Bindings {
				found = found || b.Device == device
			}
			if !found {
				return false
			}
		}
		// F10-02 discipline: retain the exact producer row, not an inferred clock.
		stage = "producer_stream_export"
		trace, e := l.readEnforcementTiming()
		readerError = ""
		if e != nil {
			readerError = "producer_stream_unavailable"
			return false
		}
		stage = "producer_serving_join"
		joined = servingProducer(trace, state.Observation.BootId, state.LeaseID, state.Generation)
		return joined != nil
	})
	r.jsonCheck("populated-grant serving readiness", "fresh accepted serving observation; desired=applied=observed lease generation; unexpired populated lease, live intended binding and same-boot producer serving row", map[string]any{"ready": ready, "device_id": device, "accepted": json.RawMessage(raw), "producer_record": joined, "last_stage": stage, "reader_error": readerError}, "read-only SQL + local desired state + streamed ExportContainerFile; stop positive probe when unavailable", ready)
	l.write(r)
	return ready
}

func (l *live) counters(device uuid.UUID) counterObservation {
	out := counterObservation{Counters: map[string]decstr.U64{}}
	raw, err := l.inside(l.nodeContainer(), "nft", "-j", "list", "table", "inet", "fvpn")
	if err != nil {
		return out
	}
	st, err := enforce.ParseState(raw)
	if err != nil {
		return out
	}
	out.Available = true
	out.Generation = decstr.U64(st.Generation)
	key := strings.ReplaceAll(device.String(), "-", "")
	out.DeviceUpload = decstr.U64(st.Counters["u_"+key].Bytes)
	out.DeviceDownload = decstr.U64(st.Counters["d_"+key].Bytes)
	for k, v := range st.Counters {
		if !strings.HasPrefix(k, "u_") && !strings.HasPrefix(k, "d_") {
			out.Counters[k] = decstr.U64(v.Bytes)
		}
	}
	// Retain the count of actual forward accept rules, not a byte count inferred
	// from device rows. Named admission counters are sampled independently above.
	var table struct {
		NFTables []struct {
			Rule struct {
				Chain string           `json:"chain"`
				Expr  []map[string]any `json:"expr"`
			} `json:"rule"`
		} `json:"nftables"`
	}
	_ = json.Unmarshal(raw, &table)
	for _, entry := range table.NFTables {
		if entry.Rule.Chain == "forward" {
			for _, expr := range entry.Rule.Expr {
				if _, ok := expr["accept"]; ok {
					out.MemberAcceptRules++
				}
			}
		}
	}
	return out
}

// §19 AT01/02: route selection uses the pinned native client's API; observations
// run only inside our existing containers, retaining no packets or credentials.
func (l *live) traceForwarding(r *record, service string, device uuid.UUID) bool {
	return l.traceForwardingExpected(r, service, device, device != uuid.Nil)
}

func (l *live) traceForwardingExpected(r *record, service string, device uuid.UUID, authorized bool) bool {
	if authorized && (device == uuid.Nil || !l.forwardingGate(r, device)) {
		r.block("Positive forwarding not launched: populated-grant serving readiness unavailable")
		return false
	}
	o := forwardingObservation{At: time.Now().UTC().Format(time.RFC3339), Member: service, Node: l.nodeContainer(), NodeIncarnation: l.inc.String(), Packets: map[string]json.RawMessage{}}
	if !l.installProbe() {
		r.block("Bounded forwarding probe build/copy unavailable")
		return false
	}
	if service != "member" {
		_, err := l.docker("cp", l.probePath(), l.name(service)+":/tmp/at-members-probe")
		if err != nil {
			r.block("Extra native peer probe unavailable")
			return false
		}
	}

	if raw, e := os.ReadFile(l.probePath()); e == nil {
		hash := sha256.Sum256(raw)
		o.ProbeSHA256 = hex.EncodeToString(hash[:])
	}
	o.RouteBefore = l.routeObservation(service)
	// Native distribution can lag central publication; poll the advertised
	// network before selecting it. Negative peers are probed once, never enrolled.
	selectRoute := func() bool {
		got := l.nb("GET", "/routes", nil)
		var routes []netbird.Route
		_ = json.Unmarshal(got.body, &routes)
		if got.status != 200 {
			return false
		}
		advertised, _ := l.inside(service, "netbird", "routes", "list")
		var targetPeer string
		_ = l.pool.QueryRow(l.ctx, `SELECT peer_id FROM node_incarnations WHERE id=$1`, l.inc).Scan(&targetPeer)
		for _, route := range routes {
			if route.Peer == targetPeer && route.Network == "0.0.0.0/0" && route.Enabled && strings.Contains(string(advertised), route.NetworkID) {
				_, err := l.inside(service, "netbird", "routes", "select", route.NetworkID)
				return err == nil
			}
		}
		return false
	}
	if authorized {
		o.RouteSelected = l.wait(45*time.Second, selectRoute)
	} else {
		o.RouteSelected = selectRoute()
	}
	time.Sleep(2 * time.Second)
	o.RouteAfter = l.routeObservation(service)
	status, _ := l.inside(service, "netbird", "status", "--json")
	var native map[string]json.RawMessage
	_ = json.Unmarshal(status, &native)
	// Never persist the full NetBird status (public keys, endpoint identities).
	selected := map[string]json.RawMessage{}
	for _, key := range []string{"daemonVersion", "daemonStatus", "usesKernelInterface", "networks"} {
		if v := native[key]; len(v) > 0 {
			selected[key] = v
		}
	}
	o.PeerStatus, _ = json.Marshal(selected)
	o.Before = l.counters(device)
	l.probeSequence++
	if l.probeSequence > 9999 {
		l.t.Fatal("probe source port range exhausted")
	}
	port := fmt.Sprint(20000 + l.probeSequence)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, point := range []string{service, l.nodeContainer(), "origin"} {
		wg.Go(func() {
			raw, err := l.inside(point, "/tmp/at-members-probe", "observe", "bounded", port)
			if err != nil || !json.Valid(raw) {
				raw = []byte(`{"available":false}`)
			}
			mu.Lock()
			o.Packets[point] = raw
			mu.Unlock()
		})
	}
	time.Sleep(time.Second)
	probeID := uuid.NewString()
	armed := l.request(nil, "PUT", "https://releases.example.invalid/e2e/traffic", map[string]string{"id": probeID}, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	raw, err := l.inside(service, "/tmp/at-members-probe", "transfer", probeID, port)
	if err != nil || !json.Valid(raw) {
		raw = []byte(`{"origin_receipt":false,"probe_error":true}`)
	}
	if device != uuid.Nil {
		if c := l.counters(device); c.DeviceUpload > o.Before.DeviceUpload && c.DeviceDownload > o.Before.DeviceDownload {
			l.lastFirstByteAt = time.Now().UTC()
			o.FirstCountedByteAt = l.lastFirstByteAt.Format(time.RFC3339Nano)
		}
	}
	o.Transfer = raw
	wg.Wait()
	o.After = l.counters(device)
	if device != uuid.Nil && o.FirstCountedByteAt == "" && o.After.DeviceUpload > o.Before.DeviceUpload && o.After.DeviceDownload > o.Before.DeviceDownload {
		l.lastFirstByteAt = time.Now().UTC()
		o.FirstCountedByteAt = l.lastFirstByteAt.Format(time.RFC3339Nano)
	}
	received := l.request(nil, "GET", "https://releases.example.invalid/e2e/traffic", nil, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	var origin struct {
		ID       string `json:"probe_id"`
		Requests int    `json:"requests"`
		Source   string `json:"source_address"`
	}
	_ = json.Unmarshal(received.body, &origin)
	expectedSNAT := "192.0.4.20"
	if l.nodeContainer() == "node2" {
		expectedSNAT = "192.0.4.22"
	}
	o.OriginVerified = armed.status == 204 && received.status == 200 && origin.ID == probeID
	o.OriginRequests = origin.Requests
	o.OriginSNATMatches = armed.status == 204 && received.status == 200 && origin.ID == probeID && origin.Requests == 1 && origin.Source == expectedSNAT
	var transfer struct {
		Receipt bool `json:"origin_receipt"`
	}
	_ = json.Unmarshal(o.Transfer, &transfer)
	o.Counted = transfer.Receipt && o.OriginSNATMatches && o.After.DeviceUpload > o.Before.DeviceUpload && o.After.DeviceDownload > o.Before.DeviceDownload
	o.FirstUnconfirmedHop = "selected route to node or later path; consult counters and packet observations"
	if !o.RouteSelected {
		o.FirstUnconfirmedHop = "member exit selection: controlled-origin route remains on underlay; no selected exit"
	}
	if !o.Counted && o.After.Generation == 0 && o.After.Counters["ctbase"] > o.Before.Counters["ctbase"] {
		o.FirstUnconfirmedHop = "node forward-chain baseline: ctbase increases, zero member accept rules, no origin response"
	}
	if o.Counted {
		o.FirstUnconfirmedHop = "none: origin HTTP response and both device admission counters increased"
	}
	r.Forwarding = append(r.Forwarding, o)
	encoded, _ := json.Marshal(o)
	r.check("bounded member transfer observation", "record selected route, node counters, packet hops and origin receipt", string(encoded), "netbird routes select; ip -j route get controlled origin; concurrent 9-second AF_PACKET aggregate observers; bounded CA-verified HTTPS GET; nft counters before/after", true)
	if !authorized {
		r.check("unauthorized real transfer", "no origin response", fmt.Sprintf("origin_receipt=%t", transfer.Receipt), "bounded HTTPS from native peer expected to have no authorized exit", o.OriginVerified && !transfer.Receipt && origin.Requests == 0)
	} else if !o.Counted && o.After.Generation == 0 && o.After.Counters["ctbase"] > o.Before.Counters["ctbase"] {
		r.block("F4-01: member packets reach generation-0 node baseline; inspect this run's accepted health and apply errors. Owner: w2-agent-core.")
	} else if !o.Counted {
		r.block("F4-01: bounded member transfer did not produce both an origin receipt and device upload/download counter increments. See forwarding-observation.md and this JSON forwarding field; owner to resolve from first missing hop.")
	}
	l.write(r)
	return o.Counted
}
func (l *live) routeObservation(service string) json.RawMessage {
	raw, err := l.inside(service, "ip", "-j", "route", "get", "192.0.4.10")
	if err != nil {
		return json.RawMessage(`{"available":false}`)
	}
	var routes []map[string]any
	_ = json.Unmarshal(raw, &routes)
	// Private bridge addresses are not published. Preserve only routing semantics.
	for _, route := range routes {
		for k := range route {
			switch k {
			case "dev", "table", "flags", "uid", "cache":
			default:
				delete(route, k)
			}
		}
	}
	out, _ := json.Marshal(routes)
	return out
}

func (o forwardingObservation) String() string {
	return fmt.Sprintf("member=%s selected=%t counted=%t", o.Member, o.RouteSelected, o.Counted)
}

// §19 AT02: latest accepted health must agree with the surviving serving path.
// No strict revoked-source ingress assertion is inferred from route withdrawal;
// the shared device-revoke record retains that independent measurement limit.
func (l *live) revocationReadback(r *record) {
	_, missing := l.inside(l.nodeContainer(), "sh", "-c", "command -v conntrack")
	var raw []byte
	err := l.pool.QueryRow(l.ctx, `SELECT report_payload FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC LIMIT 1`, l.node).Scan(&raw)
	var observation api.ObservationReport
	decode := json.Unmarshal(raw, &observation)
	var safe map[string]json.RawMessage
	_ = json.Unmarshal(raw, &safe)
	evidence, _ := json.Marshal(map[string]any{"checks": safe["checks"], "forwarding": safe["member_forwarding_enabled"], "reasons": safe["forwarding_block_reasons"], "errors": safe["errors"]})
	r.check("post-revoke enforcement dependency and health", "conntrack present; typed accepted observation retained", fmt.Sprintf("conntrack_present=%t; observation=%s", missing == nil, evidence), "command -v conntrack; read latest accepted api.ObservationReport (generated enum)", missing == nil && err == nil && decode == nil)
}
