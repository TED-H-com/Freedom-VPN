//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
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
	At                  string                     `json:"at"`
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

func (l *live) forwardingGate(r *record) {
	var raw []byte
	ready := l.wait(60*time.Second, func() bool {
		var matched bool
		err := l.pool.QueryRow(l.ctx, `SELECT n.desired_config_generation=n.applied_config_generation AND s.policy_generation=n.applied_config_generation AND s.expires_at>clock_timestamp() FROM nodes n JOIN LATERAL (SELECT policy_generation,expires_at FROM serving_leases WHERE node_id=n.id ORDER BY issued_at DESC LIMIT 1) s ON true WHERE n.id=$1`, l.node).Scan(&matched)
		return err == nil && matched
	})
	err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('desired',n.desired_config_generation::text,'applied',n.applied_config_generation::text,'lease',(SELECT jsonb_build_object('generation',s.policy_generation::text,'member_count',jsonb_array_length(s.envelope->'signed'->'members'),'issued_at',s.issued_at,'expires_at',s.expires_at) FROM serving_leases s WHERE s.node_id=n.id ORDER BY s.issued_at DESC LIMIT 1),'observation',(SELECT jsonb_build_object('checks',o.report_payload->'checks','member_forwarding_enabled',o.report_payload->'member_forwarding_enabled','forwarding_block_reasons',o.report_payload->'forwarding_block_reasons') FROM node_observations o WHERE o.node_id=n.id ORDER BY received_at DESC LIMIT 1)) FROM nodes n WHERE n.id=$1`, l.node).Scan(&raw)
	r.check("current-generation lease after desired change", "uninterrupted agent: desired=applied=latest lease generation, unexpired", string(raw), "poll nodes, latest serving_leases and accepted observation; no process restart or projection writes", ready && err == nil)
	l.write(r)
}

func (l *live) counters(device uuid.UUID) counterObservation {
	out := counterObservation{Counters: map[string]decstr.U64{}}
	raw, err := l.inside("node", "nft", "-j", "list", "table", "inet", "fvpn")
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
	o := forwardingObservation{At: time.Now().UTC().Format(time.RFC3339), Member: service, Packets: map[string]json.RawMessage{}}
	if !l.installProbe() {
		r.block("Bounded forwarding probe build/copy unavailable")
		return false
	}
	if service != "member" {
		_, err := l.docker("cp", filepath.Join(l.root, ".agent/at-members/probe"), l.name(service)+":/tmp/at-members-probe")
		if err != nil {
			r.block("Extra native peer probe unavailable")
			return false
		}
	}
	o.RouteBefore = l.routeObservation(service)
	got := l.nb("GET", "/routes", nil)
	var routes []netbird.Route
	_ = json.Unmarshal(got.body, &routes)
	for _, route := range routes {
		if route.Network == "0.0.0.0/0" && route.Enabled {
			_, err := l.inside(service, "netbird", "routes", "select", route.NetworkID)
			o.RouteSelected = err == nil
			break
		}
	}
	time.Sleep(2 * time.Second)
	o.RouteAfter = l.routeObservation(service)
	status, _ := l.inside(service, "netbird", "status", "--json")
	var native map[string]json.RawMessage
	_ = json.Unmarshal(status, &native)
	// Never persist the full NetBird status (public keys, endpoint identities).
	selected := map[string]json.RawMessage{}
	for _, key := range []string{"management", "signal", "localPeerState", "routes"} {
		if v := native[key]; len(v) > 0 {
			if key == "localPeerState" || key == "routes" {
				selected[key] = v
			}
		}
	}
	o.PeerStatus, _ = json.Marshal(selected)
	o.Before = l.counters(device)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, point := range []string{service, "node", "origin"} {
		wg.Go(func() {
			raw, err := l.inside(point, "/tmp/at-members-probe", "observe", "bounded")
			if err != nil || !json.Valid(raw) {
				raw = []byte(`{"available":false}`)
			}
			mu.Lock()
			o.Packets[point] = raw
			mu.Unlock()
		})
	}
	time.Sleep(time.Second)
	raw, err := l.inside(service, "/tmp/at-members-probe", "transfer", "bounded")
	if err != nil || !json.Valid(raw) {
		raw = []byte(`{"origin_receipt":false,"probe_error":true}`)
	}
	o.Transfer = raw
	wg.Wait()
	o.After = l.counters(device)
	var transfer struct {
		Receipt bool `json:"origin_receipt"`
	}
	_ = json.Unmarshal(o.Transfer, &transfer)
	o.Counted = transfer.Receipt && o.After.DeviceUpload > o.Before.DeviceUpload && o.After.DeviceDownload > o.Before.DeviceDownload
	o.FirstUnconfirmedHop = "inspect per-interface packet observations; origin receipt and device admission must both succeed"
	if o.Counted {
		o.FirstUnconfirmedHop = "none: origin HTTP response and both device admission counters increased"
	}
	r.Forwarding = append(r.Forwarding, o)
	encoded, _ := json.Marshal(o)
	r.check("bounded member transfer observation", "record selected route, node counters, packet hops and origin receipt", string(encoded), "netbird routes select; ip -j route get controlled origin; concurrent 9-second AF_PACKET aggregate observers; bounded CA-verified HTTPS GET; nft counters before/after", true)
	if device == uuid.Nil {
		r.check("quota loser real transfer", "no origin response", fmt.Sprintf("origin_receipt=%t", transfer.Receipt), "bounded HTTPS from the losing native peer before slot reuse", !transfer.Receipt)
	} else if !o.Counted {
		r.block("F3-01: bounded member transfer did not produce both an origin receipt and device upload/download counter increments. See forwarding-observation.md and this JSON forwarding field; owner to resolve from first missing hop.")
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
