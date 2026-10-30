//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// This management fixture is a separate synthetic peer/group/route. It never
// creates a member, device binding, product publication or admission rule.
func (r *probeFixture) netbirdAPI(ctx context.Context, method, path string, body, out any) error {
	token, err := r.lab.secret("netbird.token")
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, method, r.lab.netbird+"/api"+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Transport: r.lab.transport, Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("probe management transport")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("probe management %s %s HTTP %d", method, path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
func (r *probeFixture) prepareProbe(ctx context.Context, note func(string, string)) bool {
	var group struct {
		ID string `json:"id"`
	}
	if err := r.netbirdAPI(ctx, "POST", "/groups", map[string]any{"name": "TEST ONLY health probe " + uuid.NewString(), "peers": []string{}}, &group); err != nil {
		note("probe_setup", err.Error())
		return false
	}
	var key struct {
		Key string `json:"key"`
	}
	if err := r.netbirdAPI(ctx, "POST", "/setup-keys", map[string]any{"name": "TEST ONLY probe " + uuid.NewString(), "type": "one-off", "expires_in": 3600, "auto_groups": []string{group.ID}, "usage_limit": 1, "ephemeral": false}, &key); err != nil {
		note("probe_setup", err.Error())
		return false
	}
	container := r.lab.node + "-probe"
	image, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", r.lab.node).Output()
	if err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "--name", container, "--hostname", container, "--network", r.lab.prefix, "--add-host", "releases.example.invalid:"+r.lab.originIP, "--cap-add=NET_ADMIN", "--cap-add=NET_RAW", "--device=/dev/net/tun:/dev/net/tun", "--env", "FVPN_E2E_PROBE=1", "--env", "FVPN_E2E_MEMBER_NAME="+container, "--mount", "type=bind,src="+filepath.Join(r.lab.state, strings.TrimPrefix(r.lab.node, r.lab.prefix+"-"), "etc", "ca.pem")+",dst=/etc/vpn-node/ca.pem,readonly", "--mount", "type=volume,src="+container+",dst=/var/lib/vpn-node", "--entrypoint", "/usr/local/bin/entrypoint-member.sh", strings.TrimSpace(string(image)))
	if cmd.Run() != nil {
		return false
	}
	put := exec.CommandContext(ctx, "docker", "exec", "-i", container, "sh", "-c", "umask 077; mkdir -p /var/lib/vpn-node/e2e; cat > /var/lib/vpn-node/e2e/setup.key")
	put.Stdin = strings.NewReader(key.Key)
	if put.Run() != nil {
		return false
	}
	wait, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var peers []struct{ ID, Name, IP string }
	var peerID, ip string
	if !waitUntil(wait, time.Second, func() bool {
		if r.netbirdAPI(wait, "GET", "/peers", nil, &peers) != nil {
			return false
		}
		for _, p := range peers {
			if p.Name == container {
				peerID = p.ID
				ip = p.IP
				return ip != ""
			}
		}
		return false
	}) {
		note("probe_setup", "setup-key peer join timed out")
		return false
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		return false
	}
	defer conn.Close(ctx)
	var nodePeer string
	if err = conn.QueryRow(ctx, `SELECT peer_id FROM node_incarnations WHERE id=$1`, r.incID).Scan(&nodePeer); err != nil {
		note("probe_setup", "node incarnation peer read failed")
		return false
	}
	var target struct {
		ID string `json:"id"`
	}
	if err = r.netbirdAPI(ctx, "POST", "/groups", map[string]any{"name": "TEST ONLY probe target " + uuid.NewString(), "peers": []string{nodePeer}}, &target); err != nil {
		note("probe_target", err.Error())
		return false
	}
	var policy map[string]any
	if err = r.netbirdAPI(ctx, "POST", "/policies", map[string]any{"name": "TEST ONLY probe transport " + uuid.NewString(), "enabled": true, "rules": []any{map[string]any{"name": "TEST ONLY dedicated peer to node", "enabled": true, "action": "accept", "bidirectional": true, "protocol": "all", "sources": []string{group.ID}, "destinations": []string{target.ID}}}}, &policy); err != nil {
		note("probe_policy", err.Error())
		return false
	}
	var route map[string]any
	err = r.netbirdAPI(ctx, "POST", "/routes", map[string]any{"network_id": "e2e-health-probe-" + uuid.NewString()[:8], "description": "TEST ONLY dedicated synthetic path", "network": r.lab.originIP + "/32", "peer": nodePeer, "groups": []string{group.ID}, "enabled": true, "masquerade": false, "skip_auto_apply": true, "metric": 9999, "keep_route": false, "access_control_groups": []string{}}, &route)
	if err != nil {
		note("probe_route", err.Error())
		return false
	}
	selected := waitUntil(wait, time.Second, func() bool {
		_, _ = r.lab.docker(container, "netbird", "networks", "select", fmt.Sprint(route["network_id"]))
		raw, _ := r.lab.docker(container, "ip", "route", "get", r.lab.originIP)
		return strings.Contains(string(raw), "dev fvnb0")
	})
	var count int
	err = conn.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE peer_id=$1`, peerID).Scan(&count)
	note("probe_not_member", strconv.FormatBool(err == nil && count == 0))
	note("probe_route_selected", strconv.FormatBool(selected))
	if !selected || count != 0 || err != nil {
		return false
	}
	r.probePeer = map[string]any{"peer_id": peerID, "source_v4": ip + "/32", "destinations": []any{map[string]any{"addr_v4": r.lab.originIP + "/32", "proto": "udp", "port": 4443}}}
	note("probe_config_scope", "one dedicated setup-key peer /32; one isolated origin /32 UDP 4443; no member device or grant")
	return true
}

func (r *probeFixture) startProbeCapture(ctx context.Context, note func(string, string)) func() bool {
	container := r.lab.node + "-probe"
	_, _ = r.lab.docker(container, "rm", "-f", "/var/lib/vpn-node/e2e/probe-stop")
	_, _ = r.lab.docker(r.lab.node, "rm", "-f", "/var/lib/vpn-node/e2e/probe-stop")
	id := uuid.NewString()
	_, err := r.originProbe(ctx, "PUT", id)
	if err != nil {
		return func() bool { return false }
	}
	secret, e := r.lab.secret("origin.secret")
	if e != nil {
		return func() bool { return false }
	}
	put := exec.CommandContext(ctx, "docker", "exec", "-i", container, "sh", "-c", "umask 077; cat > /var/lib/vpn-node/e2e/probe.secret")
	put.Stdin = strings.NewReader(secret)
	if put.Run() != nil {
		return func() bool { return false }
	}
	var peerOut, watchOut bytes.Buffer
	peer := exec.CommandContext(ctx, "docker", "exec", container, "fvpn-e2e-nodeconfig", "probe-client", id)
	peer.Stdout = &peerOut
	watch := exec.CommandContext(ctx, "docker", "exec", r.lab.node, "fvpn-e2e-nodeconfig", "probe-watch")
	watch.Stdout = &watchOut
	pe := peer.Start()
	we := watch.Start()
	return func() bool {
		_, _ = r.lab.docker(container, "touch", "/var/lib/vpn-node/e2e/probe-stop")
		_, _ = r.lab.docker(r.lab.node, "touch", "/var/lib/vpn-node/e2e/probe-stop")
		if pe == nil {
			pe = peer.Wait()
		}
		if we == nil {
			we = watch.Wait()
		}
		_, _ = r.lab.docker(container, "rm", "-f", "/var/lib/vpn-node/e2e/probe.secret")
		note("probe_traffic", strings.TrimSpace(peerOut.String()))
		var peerResult struct {
			Replies int `json:"authenticated_udp_replies"`
		}
		_ = json.Unmarshal(peerOut.Bytes(), &peerResult)
		note("probe_window", strings.TrimSpace(watchOut.String()))
		origin, _ := r.originProbe(ctx, "GET", id)
		note("probe_origin_receipt", origin)
		var a struct {
			Requests int    `json:"requests"`
			Source   string `json:"source_address"`
		}
		_ = json.Unmarshal([]byte(origin), &a)
		var w struct {
			Grant    int     `json:"grant_samples"`
			Timeout  float64 `json:"max_timeout_seconds"`
			Up       uint64  `json:"probe_up_bytes"`
			Down     uint64  `json:"probe_down_bytes"`
			Before   uint64  `json:"member_bytes_before"`
			After    uint64  `json:"member_bytes_after"`
			Elements int     `json:"elements_after"`
		}
		_ = json.Unmarshal(watchOut.Bytes(), &w)
		ok := pe == nil && we == nil && peerResult.Replies > 0 && a.Requests > 0 && a.Source == r.targetSNAT() && w.Grant > 0 && w.Timeout > 0 && w.Timeout <= 45 && w.Up > 0 && w.Down > 0 && w.Before == w.After && w.Elements == 0
		note("probe_traversal_verified", strconv.FormatBool(ok))
		// No challenge/attestation is copied to public evidence.
		_ = os.Remove(filepath.Join(r.lab.state, strings.TrimPrefix(r.lab.node, r.lab.prefix+"-"), "var", "e2e", "probe-stop"))
		return ok
	}
}

func (r *probeFixture) originProbe(ctx context.Context, method, id string) (string, error) {
	secret, err := r.lab.secret("origin.secret")
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(map[string]string{"id": id})
	req, err := http.NewRequestWithContext(ctx, method, r.lab.releases+"/e2e/traffic", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-E2E-Secret", secret)
	resp, err := (&http.Client{Transport: r.lab.transport, Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return "", fmt.Errorf("origin probe HTTP %d", resp.StatusCode)
	}
	return strings.TrimSpace(string(body)), nil
}

// UpdaterProbe is a dedicated non-member peer. Config is the exact probe_peer
// value for an updater invocation; AgentConfig must accompany it. Neither is a
// credential. Notes are non-secret setup observations and omit the peer ID.
// Do not record the raw peer ID in public evidence.
type UpdaterProbe struct {
	Config      map[string]any
	AgentConfig string
	Notes       []Observation
	fixture     *probeFixture
}
type probeFixture struct {
	lab       *lab
	incID     uuid.UUID
	probePeer map[string]any
}

// ProbeObservation contains only bounded aggregate observer results.
type ProbeObservation struct {
	Verified     bool          `json:"verified"`
	Observations []Observation `json:"observations"`
}

// SetupUpdaterProbe creates transport through the enrolled node, never product
// admission or exception rules. A fresh stack supports one probe per slot.
func (s *Stack) SetupUpdaterProbe(ctx context.Context, slot string, node EnrolledNode) (*UpdaterProbe, error) {
	if slot != "node" && slot != "node2" {
		return nil, fmt.Errorf("unknown node slot")
	}
	l := *s.lab
	l.node = l.prefix + "-" + slot
	raw, err := l.docker(l.node, "cat", "/var/lib/vpn-node/identity/node.json")
	var identity struct {
		Node uuid.UUID `json:"node_id"`
		Inc  uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &identity) != nil || identity.Node != node.NodeID || identity.Inc != node.IncarnationID || identity.Node == uuid.Nil {
		return nil, fmt.Errorf("probe target identity mismatch")
	}
	f := &probeFixture{lab: &l, incID: node.IncarnationID}
	var notes []Observation
	if !f.prepareProbe(ctx, func(k, v string) { notes = append(notes, Observation{Name: k, Detail: v}) }) {
		failure := "no detail"
		if len(notes) > 0 {
			last := notes[len(notes)-1]
			failure = last.Name + ": " + last.Detail
		}
		return nil, fmt.Errorf("probe setup incomplete: %s", failure)
	}
	peer, _ := f.probePeer["peer_id"].(string)
	safe := make([]Observation, 0, len(notes))
	for _, n := range notes {
		if peer != "" && strings.Contains(n.Detail, peer) {
			continue
		}
		safe = append(safe, n)
	}
	return &UpdaterProbe{Config: f.probePeer, AgentConfig: "/etc/vpn-node/agent-run.json", Notes: safe, fixture: f}, nil
}

// configuredProbeIssued reports whether a fresh health-challenge document
// carries the configured probe. The raw peer ID is not retained by the caller.
func configuredProbeIssued(raw []byte, configured map[string]any) bool {
	if len(raw) == 0 || len(raw) > 8192 || configured == nil {
		return false
	}
	var doc struct {
		Probe json.RawMessage `json:"probe"`
	}
	if json.Unmarshal(raw, &doc) != nil || len(doc.Probe) == 0 || string(doc.Probe) == "null" {
		return false
	}
	want, err := json.Marshal(configured)
	if err != nil {
		return false
	}
	return jsonEqual(doc.Probe, want)
}

func jsonEqual(a, b []byte) bool {
	var left, right any
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	x, e1 := json.Marshal(left)
	y, e2 := json.Marshal(right)
	return e1 == nil && e2 == nil && bytes.Equal(x, y)
}

// Observe starts traffic plus a read-only nft observer before the real updater
// runs. Call the returned collector exactly once after health completes, even
// on failure. It never creates a challenge, attestation or exception element.
func (p *UpdaterProbe) Observe(ctx context.Context) (func() ProbeObservation, error) {
	if p == nil || p.fixture == nil {
		return nil, fmt.Errorf("probe is not initialized")
	}
	var result ProbeObservation
	finish := p.fixture.startProbeCapture(ctx, func(k, v string) { result.Observations = append(result.Observations, Observation{k, v}) })
	done := false
	return func() ProbeObservation {
		if !done {
			result.Verified = finish()
			done = true
		}
		return result
	}, nil
}

func (r *probeFixture) targetSNAT() string {
	raw, err := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/agent-run.json")
	var cfg struct {
		Datapath struct {
			SNAT string `json:"snat"`
		} `json:"datapath"`
	}
	if err != nil || json.Unmarshal(raw, &cfg) != nil {
		return ""
	}
	return cfg.Datapath.SNAT
}
