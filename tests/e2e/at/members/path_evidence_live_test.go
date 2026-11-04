//go:build e2e

package members

import (
	"bytes"
	"encoding/json"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture"
	"github.com/google/uuid"
)

type underlayAddress struct{ address, device string }

func (l *live) underlayAddresses(role string) []underlayAddress {
	raw, err := l.inside(role, "ip", "-j", "address", "show")
	var rows []struct {
		Ifname   string
		AddrInfo []struct{ Local string } `json:"addr_info"`
	}
	out := []underlayAddress{}
	if err != nil || json.Unmarshal(raw, &rows) != nil {
		return out
	}
	for _, r := range rows {
		if r.Ifname == "lo" || r.Ifname == "fvnb0" || r.Ifname == "fvmb0" {
			continue
		}
		for _, a := range r.AddrInfo {
			ip, e := netip.ParseAddr(a.Local)
			if e == nil && ip.IsGlobalUnicast() {
				out = append(out, underlayAddress{a.Local, safeInterface(r.Ifname)})
			}
		}
	}
	return out
}
func (l *live) routeNeighbour(role, address string) map[string]any {
	start := time.Now().UTC()
	// Address is parsed first and passed as an argument, never shell source.
	if _, err := netip.ParseAddr(address); err != nil {
		return map[string]any{"available": false}
	}
	raw, err := l.inside(role, "sh", "-c", `ip -j route get "$1"; printf '\nF23-NEIGH\n'; ip -j neigh show to "$1"`, "path", address)
	parts := strings.SplitN(string(raw), "\nF23-NEIGH\n", 2)
	out := map[string]any{"started_at": start, "ended_at": time.Now().UTC(), "destination_label": addressLabel(address), "family": addressFamily(address), "command_ok": err == nil, "routes": []pathRoute{}, "neighbours": []pathNeighbour{}}
	if len(parts) == 2 {
		out["routes"] = parsePathRoutes([]byte(parts[0]))
		out["neighbours"] = parsePathNeighbours([]byte(parts[1]))
	}
	return out
}
func (l *live) pairEnd(role string, status []byte, target string, candidates []underlayAddress) map[string]any {
	start := time.Now().UTC()
	peer, err := nativecapture.Select(status, target)
	var identity struct {
		Peers struct {
			Details []struct {
				IP  string `json:"netbirdIp"`
				Key string `json:"publicKey"`
			} `json:"details"`
		} `json:"peers"`
	}
	_ = json.Unmarshal(status, &identity)
	key := ""
	for _, p := range identity.Peers.Details {
		if strings.Split(p.IP, "/")[0] == strings.Split(target, "/")[0] && target != "" {
			key = p.Key
		}
	}
	wgRaw, wgErr := l.inside(role, "sh", "-c", `command -v wg >/dev/null || exit 127; data=$(wg show all dump) || exit; printf '%s\n' "$data" | awk 'NF == 9 { $3="(redacted)"; print }'`)
	wg := parseWireGuardDump(wgRaw, key, target)
	paths := []map[string]any{}
	selected, _, _ := net.SplitHostPort(peer.Endpoint.Remote)
	seen := map[string]bool{}
	for _, a := range append(candidates, underlayAddress{selected, ""}) {
		if a.address == "" || seen[a.address] {
			continue
		}
		seen[a.address] = true
		p := l.routeNeighbour(role, a.address)
		p["remote_interface"] = a.device
		p["selected_remote"] = a.address == selected
		paths = append(paths, p)
	}
	return map[string]any{"started_at": start, "ended_at": time.Now().UTC(), "peer_found": err == nil, "selected_local": safePathEndpoint(peer.Endpoint.Local), "selected_remote": safePathEndpoint(peer.Endpoint.Remote), "wireguard_command_ok": wgErr == nil, "wireguard": wg, "paths": paths, "scope": "route-get and neighbour entries for the selected endpoint and all opposite underlay candidates; pre-transfer idle selection may be absent; candidates permit joining the later selection without a retrospective read"}
}
func (l *live) transferPathViews(member string, memberRaw, nodeRaw []byte) map[string]any {
	start := time.Now().UTC()
	node := l.nodeContainer()
	m, n := l.underlayAddresses(member), l.underlayAddresses(node)
	return map[string]any{"started_at": start, "member": l.pairEnd(member, memberRaw, nativeAddress(nodeRaw), n), "node": l.pairEnd(node, nodeRaw, nativeAddress(memberRaw), m), "node_role": node, "ended_at": time.Now().UTC()}
}

type nativeRowWindow struct {
	l                 *live
	r                 *record
	label, kind, role string
	start             time.Time
	beforeLog         []byte
	beforeLogOK       bool
	startStep         int
	beforePath        map[string]any
}

func (l *live) exportObservationLog() ([]byte, bool) {
	dir := filepath.Join(l.root, ".agent/tmp/f23-observations", l.runID, uuid.NewString())
	p := filepath.Join(dir, "agent.log")
	_, err := l.hook.ExportContainerFile(l.ctx, l.nodeContainer(), "/var/lib/vpn-node/e2e/agent.log", p)
	if err != nil {
		return nil, false
	}
	b, e := os.ReadFile(p)
	return b, e == nil
}
func (l *live) startNativeRow(r *record, label, kind string) *nativeRowWindow {
	w := &nativeRowWindow{l: l, r: r, label: label, kind: kind, role: l.nodeContainer(), start: time.Now().UTC(), startStep: len(r.Steps)}
	w.beforePath = l.controlPath()
	w.beforeLog, w.beforeLogOK = l.exportObservationLog()
	return w
}
func (w *nativeRowWindow) stop() {
	endPath := w.l.controlPath()
	after, logOK := w.l.exportObservationLog()
	end := time.Now().UTC()
	raw, err := w.l.d146ObservationWindow(w.start, end)
	logs := observationLogWindow(w.beforeLog, after, w.start, end)
	input, _ := json.Marshal(map[string]any{"before": string(w.beforeLog), "after": string(after), "start": w.start, "end": end, "observations": observationsInWindow(raw, w.start, end)})
	cmd := exec.CommandContext(w.l.ctx, "python3", "-B", "tests/e2e/at/members/observation_records.py")
	cmd.Dir, cmd.Stdin = w.l.root, bytes.NewReader(input)
	parsed, parseErr := cmd.Output()
	if parseErr == nil && json.Unmarshal(parsed, &logs) == nil {
		logs["extraction_ok"] = true
	} else {
		logs["extraction_ok"] = false
		logs["reason"] = "observation receipt extraction unavailable"
	}
	logs["start_export_ok"] = w.beforeLogOK
	logs["end_export_ok"] = logOK
	out := map[string]any{"row": w.label, "fixture": w.kind, "node_role": w.role, "started_at": w.start, "ended_at": end, "stored_observations_read_ok": err == nil, "stored_observations": observationsInWindow(raw, w.start, end), "agent_log": logs, "control_path_start": w.beforePath, "control_path_end": endPath, "fixture_path_assessment": fixtureControlAssessment(w.kind, w.beforePath, endPath)}
	// Add metadata to an existing row, never a verdict or a new check/index.
	for i := w.startStep; i < len(w.r.Steps); i++ {
		if w.r.Steps[i].Name == w.label+" transfer" || (i == w.startStep && len(w.r.Steps) == w.startStep+1) {
			var actual map[string]any
			if json.Unmarshal([]byte(w.r.Steps[i].Actual), &actual) != nil || actual == nil {
				actual = map[string]any{"original_actual": w.r.Steps[i].Actual}
			}
			actual["observation_window"] = out
			encoded, _ := json.Marshal(actual)
			w.r.Steps[i].Actual = string(encoded)
			return
		}
	}
	// An early setup failure still retains the diagnostic window independently.
	encoded, _ := json.Marshal(out)
	p := filepath.Join(w.l.root, "docs/e2e/at/members", "F23-MTU-early-"+uuid.NewString()+".json")
	if e := os.WriteFile(p, append(encoded, '\n'), 0600); e != nil {
		w.l.t.Error("MTU window retention unavailable")
	}
}
func fixtureControlAssessment(kind string, paths ...map[string]any) map[string]any {
	return map[string]any{"fixture": kind, "applies_to_control_route": false, "basis": "native input MTU/PTB changes the member namespace; known-PMTU constrains only the selected node-to-member host route; routed fixture changes the member/router link. Control route/source and TCP metrics are read independently at both boundaries.", "boundary_paths_available": len(paths) == 2 && paths[0]["available"] == true && paths[1]["available"] == true, "limit": "endpoint route evidence does not prove absence of indirect shared-host effects"}
}
func (l *live) controlPath() map[string]any {
	start := time.Now().UTC()
	out := map[string]any{"available": false, "started_at": start}
	// Only control_url is decoded, held in memory and resolved inside the node.
	args, argsErr := l.inside(l.nodeContainer(), "sh", "-c", `pid=$(cat /var/lib/vpn-node/e2e/agent.pid) || exit; case "$pid" in *[!0-9]*|"") exit 1;; esac; cat "/proc/$pid/cmdline"`)
	config := ""
	argv := strings.Split(string(args), "\x00")
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "-config" {
			config = argv[i+1]
		}
	}
	if argsErr != nil || (!strings.HasPrefix(config, "/etc/vpn-node/") && !strings.HasPrefix(config, "/var/lib/vpn-node/e2e/")) {
		out["reason"] = "active agent configuration unavailable"
		return out
	}
	raw, err := l.inside(l.nodeContainer(), "cat", config)
	var cfg struct {
		ControlURL string `json:"control_url"`
	}
	if err != nil || json.Unmarshal(raw, &cfg) != nil {
		out["reason"] = "agent configuration unavailable"
		return out
	}
	u, e := url.Parse(cfg.ControlURL)
	if e != nil || u.Hostname() == "" {
		out["reason"] = "control target unavailable"
		return out
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	address := u.Hostname()
	if net.ParseIP(address) == nil {
		raw, e = l.inside(l.nodeContainer(), "getent", "ahosts", address)
		if e != nil {
			out["reason"] = "control resolution unavailable"
			return out
		}
		f := strings.Fields(string(raw))
		if len(f) == 0 || net.ParseIP(f[0]) == nil {
			out["reason"] = "control address unavailable"
			return out
		}
		address = f[0]
	}
	out["route"] = l.routeNeighbour(l.nodeContainer(), address)
	ss, ssErr := l.inside(l.nodeContainer(), "ss", "-Htinp", "dst", address, "dport", "=", port)
	out["ss_command_ok"] = ssErr == nil
	out["tcp_connections"] = parseControlTCP(ss, address, port)
	if ssErr != nil {
		out["tcp_unavailable_reason"] = "ss missing or command failed; no TCP metrics inferred"
	}
	out["available"] = true
	out["ended_at"] = time.Now().UTC()
	return out
}

// The shared StartNativeCapture already covers every interface with -i any.
func (l *live) retainUnderlayCapture(c *nativeCaptureSession) json.RawMessage {
	if c == nil {
		return nil
	}
	input := filepath.Join(c.directory, "f23-input.json")
	if os.WriteFile(input, append(append([]byte(`{"capture":`), c.result...), '}'), 0600) != nil {
		return json.RawMessage(`{"available":false}`)
	}
	cmd := exec.CommandContext(l.ctx, "python3", "-B", "tests/e2e/at/members/underlay_evidence.py", "--input", input, "--private", c.directory)
	cmd.Dir = l.root
	raw, err := cmd.Output()
	if err != nil || !json.Valid(raw) {
		return json.RawMessage(`{"available":false,"reason":"existing capture export or header decode unavailable"}`)
	}
	return raw
}
