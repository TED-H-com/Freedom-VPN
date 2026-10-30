//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func (p *liveProbe) deviceBytes(id uuid.UUID) (map[string]decstr.U64, bool) {
	return p.deviceBytesAt("node", id)
}
func (p *liveProbe) deviceBytesAt(service string, id uuid.UUID) (map[string]decstr.U64, bool) {
	if service == "" {
		service = "node"
	}
	raw, e := p.docker(service, "nft", "-j", "list", "table", "inet", "fvpn")
	if e != nil {
		return nil, false
	}
	state, e := enforce.ParseState(raw)
	if e != nil {
		return nil, false
	}
	key := strings.ReplaceAll(id.String(), "-", "")
	up, ok := state.Counters["u_"+key]
	down, ok2 := state.Counters["d_"+key]
	return map[string]decstr.U64{"upload": decstr.U64(up.Bytes), "download": decstr.U64(down.Bytes), "generation": decstr.U64(state.Generation)}, ok && ok2
}
func (p *liveProbe) originAddress() (string, string) {
	return p.originAddressAt("node")
}
func (p *liveProbe) originAddressAt(service string) (string, string) {
	if service == "" {
		service = "node"
	}
	raw, _ := p.docker(service, "cat", "/etc/vpn-node/agent-run.json")
	var cfg struct {
		Datapath struct {
			SNAT string `json:"snat"`
		} `json:"datapath"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return "", ""
	}
	i := strings.LastIndex(cfg.Datapath.SNAT, ".")
	if i < 0 {
		return "", ""
	}
	return cfg.Datapath.SNAT[:i] + ".10", cfg.Datapath.SNAT
}
func (p *liveProbe) originProbe(method, id string) (map[string]any, bool) {
	raw, _ := json.Marshal(map[string]string{"id": id})
	r, e := http.NewRequestWithContext(p.t.Context(), method, "https://releases.example.invalid/e2e/traffic", bytes.NewReader(raw))
	if e != nil {
		return nil, false
	}
	r.Header.Set("X-E2E-Secret", p.secret("origin.secret"))
	resp, e := p.client().Do(r)
	if e != nil {
		return nil, false
	}
	defer resp.Body.Close()
	var out map[string]any
	if method == "PUT" {
		return nil, resp.StatusCode == 204
	}
	e = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&out)
	return out, e == nil && resp.StatusCode == 200
}
func (p *liveProbe) route(peer flowPeer, origin string) (string, string) {
	raw, e := p.docker(peer.Service, "ip", "route", "get", origin)
	if e != nil {
		return "", "route command failed"
	}
	full := strings.TrimSpace(string(raw))
	m := regexp.MustCompile(`\bdev (\S+)`).FindStringSubmatch(full)
	if len(m) != 2 {
		return "", full
	}
	return m[1], full
}

// The authenticated origin receipt and this exact device's counters are required
// after the native client selects the single API-published exit (§19 AT29/AT31).
func (p *liveProbe) selectedFlow(peer flowPeer) map[string]any {
	out := map[string]any{"utc": time.Now().UTC(), "peer": peer, "counted": false, "order": []string{"select advertised exit", "route dev fvnb0", "HTTPS 200", "authenticated node-SNAT receipt", "exact-device byte delta"}}
	origin, snat := p.originAddressAt(peer.NodeService)
	if origin == "" {
		out["blocker"] = "origin configuration absent"
		return out
	}
	raw, err := p.docker(peer.Service, "netbird", "networks", "list")
	re := regexp.MustCompile(`ID: (fvpn-exit-[a-z0-9]+)\s+Network: 0\.0\.0\.0/0`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	out["network_list"] = strings.TrimSpace(string(raw))
	out["network_list_ok"] = err == nil
	if peer.ExitID != "" {
		filtered := [][]string{}
		for _, m := range matches {
			if m[1] == peer.ExitID {
				filtered = append(filtered, m)
			}
		}
		matches = filtered
	}
	if len(matches) != 1 {
		out["blocker"] = "expected exactly one advertised product exit"
		p.preserveRouteFailure(peer, out)
		return out
	}
	selected, e := p.docker(peer.Service, "netbird", "networks", "select", matches[0][1])
	out["selection_ok"] = e == nil
	out["selection_output"] = strings.TrimSpace(string(selected))
	dev := ""
	route := ""
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		dev, route = p.route(peer, origin)
		if dev == "fvnb0" {
			break
		}
		time.Sleep(time.Second)
	}
	out["route_device"] = dev
	out["route_get"] = strings.ReplaceAll(strings.ReplaceAll(route, origin, "<controlled-origin>"), snat, "<node-snat>")
	if e != nil || dev != "fvnb0" {
		out["blocker"] = "selected exit did not route controlled origin to fvnb0"
		p.preserveRouteFailure(peer, out)
		return out
	}
	id := uuid.NewString()
	_, ok := p.originProbe("PUT", id)
	if !ok {
		out["blocker"] = "authenticated origin probe could not arm"
		return out
	}
	before, ok1 := p.deviceBytesAt(peer.NodeService, peer.Device)
	// Extra native peers do not inherit Compose origin aliases. The generated
	// origin certificate includes this controlled IP SAN; keep TLS verification.
	body, fetchErr := p.docker(peer.Service, "wget", "--server-response", "--timeout=3", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "--header=X-E2E-Probe: "+id, "-O", "-", "https://"+origin+"/healthz")
	out["http_200"] = fetchErr == nil && bytes.Contains(body, []byte("200 OK"))
	arrival, ok := p.originProbe("GET", id)
	out["origin_authenticated"] = ok
	out["origin_probe_matches"] = arrival["probe_id"] == id
	out["origin_node_snat"] = arrival["source_address"] == snat
	out["origin_requests"] = arrival["requests"]
	after, ok2 := p.deviceBytesAt(peer.NodeService, peer.Device)
	out["counter_reads_ok"] = ok1 && ok2
	out["before_bytes"] = before
	out["after_bytes"] = after
	delta := map[string]decstr.U64{}
	for _, k := range []string{"upload", "download"} {
		if after[k] >= before[k] {
			delta[k] = after[k] - before[k]
		}
	}
	out["admitted_bytes_delta"] = delta
	out["counted"] = out["http_200"] == true && ok && arrival["probe_id"] == id && arrival["source_address"] == snat && ok1 && ok2 && delta["upload"] > 0 && delta["download"] > 0
	return out
}
func (p *liveProbe) preserveRouteFailure(peer flowPeer, out map[string]any) {
	raw, _ := p.docker(peer.Service, "netbird", "status")
	dir := filepath.Join(p.root, ".agent/secops/route-failure")
	_ = os.MkdirAll(dir, 0700)
	// Full native status includes endpoint identities: preserve only privately.
	_ = os.WriteFile(filepath.Join(dir, peer.Service+"-status.txt"), raw, 0600)
	networks, _ := p.docker(peer.Service, "netbird", "networks", "list")
	_ = os.WriteFile(filepath.Join(dir, peer.Service+"-networks.txt"), networks, 0600)
	out["exact_native_output_retained"] = true
	out["public_status_scope"] = "full status private; network IDs and route semantics only published"
}
