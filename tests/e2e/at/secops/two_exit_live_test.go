//go:build e2e && integration

package secops

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (p *liveProbe) twoExitTraffic(stack *labstack.Stack, primary uuid.UUID, peers []flowPeer) {
	out := map[string]any{"status": "blocked", "requested_seconds": 120, "relay_profile": map[string]any{"status": "blocked", "owner": "w3-devstack-e2e", "missing_export": "bounded force-relay member transport hook with read-back and restoration; StartSecondExit creates a node, not a relay path"}}
	p.metadata["two_exit_traffic"] = out
	if len(peers) < 4 {
		out["blocker"] = "fewer than two native members"
		return
	}
	re := regexp.MustCompile(`ID: (fvpn-exit-[a-z0-9]+)\s+Network: 0\.0\.0\.0/0`)
	raw, e := p.docker(peers[0].Service, "netbird", "networks", "list")
	old := re.FindAllStringSubmatch(string(raw), -1)
	if e != nil || len(old) != 1 {
		out["blocker"] = "primary route identifier unavailable"
		return
	}
	admin := p.login("e2e-admin", "admin")
	if stack.StartSecondExit(p.t.Context()) != nil {
		out["blocker"] = "StartSecondExit failed"
		return
	}
	second, e := stack.EnrollAt(p.t.Context(), labstack.BrowserSession{Client: admin.client, CSRF: admin.csrf}, "TEST ONLY second exit", "node2")
	if e != nil {
		out["blocker"] = "EnrollAt node2 failed"
		return
	}
	out["node_id"] = second.NodeID
	out["incarnation_id"] = second.IncarnationID
	primaryInc := p.metadata["agent_node_incarnation"]
	p.capacityInput(admin, second.NodeID)
	p.serving(admin, second.NodeID)
	out["publication_operation"] = p.metadata["publication_operation"]
	p.metadata["agent_node_incarnation"] = primaryInc
	out["observation"] = p.publicationObservation(second.NodeID)
	chosen := ""
	for until := time.Now().Add(45 * time.Second); time.Now().Before(until); {
		raw, _ = p.docker(peers[3].Service, "netbird", "networks", "list")
		for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
			if m[1] != old[0][1] {
				chosen = m[1]
			}
		}
		if chosen != "" {
			break
		}
		time.Sleep(time.Second)
	}
	if chosen == "" {
		out["blocker"] = "second publication does not advertise a second exit"
		return
	}
	pair := []flowPeer{peers[0], peers[3]}
	pair[0].ExitID = old[0][1]
	pair[1].ExitID = chosen
	pair[1].NodeService = "node2"
	awaitCounted := func(peer flowPeer) map[string]any {
		var r map[string]any
		for until := time.Now().Add(45 * time.Second); time.Now().Before(until); {
			r = p.selectedFlow(peer)
			if r["counted"] == true {
				break
			}
			time.Sleep(2 * time.Second)
		}
		return r
	}
	proofs := []map[string]any{awaitCounted(pair[0]), awaitCounted(pair[1])}
	out["initial_flows"] = proofs
	if proofs[0]["counted"] != true || proofs[1]["counted"] != true {
		out["blocker"] = "two actual exits did not both produce counted flow"
		return
	}
	collect, e := stack.StartNodeTelemetry(p.t.Context(), 5*time.Second)
	if e != nil {
		out["blocker"] = "node telemetry failed"
		return
	}
	defer collect()
	rows := make([][]map[string]any, 2)
	var wg sync.WaitGroup
	start := time.Now()
	for i, peer := range pair {
		wg.Go(func() {
			for time.Since(start) < 120*time.Second {
				tick := time.Now()
				rows[i] = append(rows[i], p.rawFlow(peer))
				time.Sleep(time.Until(tick.Add(time.Second)))
			}
		})
	}
	secondary := []json.RawMessage{}
	for time.Since(start) < 120*time.Second {
		raw, err := p.docker("node2", "fvpn-e2e-nodeconfig", "telemetry")
		if err == nil && json.Valid(raw) {
			secondary = append(secondary, json.RawMessage(raw))
		}
		time.Sleep(5 * time.Second)
	}
	wg.Wait()
	out["primary_node_telemetry"] = collect()
	out["secondary_node_telemetry"] = secondary
	out["samples"] = rows
	out["duration_seconds"] = time.Since(start).Seconds()
	out["final_flows"] = []map[string]any{p.selectedFlow(pair[0]), p.selectedFlow(pair[1])}
	paths := []map[string]any{}
	for _, peer := range pair {
		raw, e := p.docker(peer.Service, "netbird", "status", "--detail")
		paths = append(paths, map[string]any{"peer": peer.Service, "read_ok": e == nil, "relayed_markers": strings.Count(string(raw), "Relayed"), "p2p_markers": strings.Count(string(raw), "P2P"), "scope": "aggregate native status markers; no endpoint addresses or keys published"})
	}
	out["observed_paths"] = paths
	out["status"] = "measured two-exit HTTPS subset"
	out["primary_observation"] = p.publicationObservation(primary)
}
