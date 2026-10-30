//go:build e2e

package members

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
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
	SourceAT            string                     `json:"source_at,omitempty"`
	ProbeSHA256         string                     `json:"probe_sha256"`
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

	if raw, e := os.ReadFile(filepath.Join(l.root, ".agent/at-members/probe")); e == nil {
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
		for _, route := range routes {
			if route.Network == "0.0.0.0/0" && route.Enabled && strings.Contains(string(advertised), route.NetworkID) {
				_, err := l.inside(service, "netbird", "routes", "select", route.NetworkID)
				return err == nil
			}
		}
		return false
	}
	if device != uuid.Nil {
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
	o.FirstUnconfirmedHop = "selected route to node or later path; consult counters and packet observations"
	if !o.RouteSelected {
		o.FirstUnconfirmedHop = "member exit selection: controlled-origin route remains on underlay; no selected exit"
	}
	if !o.Counted && o.After.Generation == 0 && o.After.Counters["ctbase"] > o.Before.Counters["ctbase"] {
		o.FirstUnconfirmedHop = "node forward-chain baseline: ctbase increases, zero member accept rules, no origin response (F3-02)"
	}
	if o.Counted {
		o.FirstUnconfirmedHop = "none: origin HTTP response and both device admission counters increased"
	}
	r.Forwarding = append(r.Forwarding, o)
	encoded, _ := json.Marshal(o)
	r.check("bounded member transfer observation", "record selected route, node counters, packet hops and origin receipt", string(encoded), "netbird routes select; ip -j route get controlled origin; concurrent 9-second AF_PACKET aggregate observers; bounded CA-verified HTTPS GET; nft counters before/after", true)
	if device == uuid.Nil {
		r.check("quota loser real transfer", "no origin response", fmt.Sprintf("origin_receipt=%t", transfer.Receipt), "bounded HTTPS from the losing native peer before slot reuse", !transfer.Receipt)
	} else if !o.Counted && o.After.Generation == 0 && o.After.Counters["ctbase"] > o.Before.Counters["ctbase"] {
		r.block("F3-02: member packets reach the node but increment its generation-0 baseline drop counter. Inspect required conntrack after real revocation; see requests.md.")
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

// Read-only diagnosis after the first real revocation; no missing package is
// installed by the AT harness, and no enforcement decision is bypassed.
func (l *live) revocationReadback(r *record) {
	// The later paid-member attempt measures where traffic stops after AT02's
	// revoke. Link that unchanged same-run trace, labeling its original case.
	if r.ID == "AT02" {
		var later record
		raw, err := os.ReadFile(filepath.Join(l.root, "docs/e2e/at/members/AT03.json"))
		already := false
		for _, o := range r.Forwarding {
			already = already || o.SourceAT == "AT03"
		}
		if !already && err == nil && json.Unmarshal(raw, &later) == nil && later.RunID == r.RunID && later.NodeIncarnation == r.NodeIncarnation {
			for _, o := range later.Forwarding {
				if !o.Counted && o.After.Generation == 0 && o.After.Counters["ctbase"] > o.Before.Counters["ctbase"] {
					o.SourceAT = "AT03"
					o.FirstUnconfirmedHop = "node forward-chain baseline after AT02 revocation: ctbase increases; zero accept rules; no origin receipt (F3-02)"
					r.Forwarding = append(r.Forwarding, o)
					r.check("same-run post-revoke transfer", "link subsequent AT03 trace without claiming a new probe", o.String(), "read same-run AT03 forwarding record; matching run UUID and incarnation", true)
				}
			}
		}
	}

	_, missing := l.inside("node", "sh", "-c", "command -v conntrack")
	var raw []byte
	err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('first_apply_failure',(SELECT min(received_at) FROM node_observations WHERE node_id=$1 AND report_payload->'errors' @> '[{"stage":"apply","message":"apply failed"}]'::jsonb),'apply_failure_observations',(SELECT count(*) FROM node_observations WHERE node_id=$1 AND report_payload->'errors' @> '[{"stage":"apply","message":"apply failed"}]'::jsonb))`, l.node).Scan(&raw)
	r.check("post-revoke enforcement dependency", "record required conntrack executable and actual apply failures", fmt.Sprintf("conntrack_present=%t; observations=%s", missing == nil, raw), "docker exec node command -v conntrack; read immutable node observation errors", err == nil)
	if missing != nil {
		r.block("F3-02: required conntrack executable is absent from the running lab node. The first device revocation is followed by repeated HEALTH_CHECK_FAILED/apply failed and generation-0 baseline; later paid-member packets increment ctbase instead of member counters. Owner: devstack deploy/devstack/Dockerfile.node; node/enforce/apply.go:deleteConntrack requires conntrack -D. No package was installed and no projection was bypassed.")
	}
}
