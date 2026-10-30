//go:build e2e

package updates_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type updateProbeCapture struct {
	s              *updatesFollowup
	id             string
	start          time.Time
	configurations map[string]any
	actual         updateDocument
	challenges     []json.RawMessage
	seen           map[string]bool
}

// The entrypoint's stdout/stderr file is private to the node and overwritten
// for each invocation. Only the fixed omission message/reason is published.
func (s *updatesFollowup) beginProbeCapture(id string) *updateProbeCapture {
	p := &updateProbeCapture{s: s, id: id, start: time.Now().UTC(), configurations: map[string]any{}, seen: map[string]bool{}, challenges: []json.RawMessage{}}
	if id != "AT23" && id != "AT25" {
		return p
	}
	for _, path := range []string{"/etc/vpn-node/updater-recover.json", "/var/lib/vpn-node/e2e/updater.json", "/var/lib/vpn-node/e2e/updater.json.done"} {
		raw, err := s.r.lab.docker(s.r.lab.node, "cat", path)
		if err != nil {
			p.configurations[path] = map[string]any{"readable": false}
			continue
		}
		cfg := document(raw)
		if cfg == nil {
			p.configurations[path] = map[string]any{"readable": true, "valid_json": false}
			continue
		}
		// Explicit allowlist: future secret-bearing config additions cannot leak.
		safe := updateDocument{}
		for _, key := range []string{"node_id", "incarnation_id", "boot_id", "platform", "probe_peer", "agent_config", "result_url", "desired_state", "meter_checkpoint", "stop_command", "start_command", "release_trust", "desired_trust", "mtls_cert", "mtls_key", "mtls_ca"} {
			if value, ok := cfg[key]; ok {
				safe[key] = value
			}
		}
		p.configurations[path] = map[string]any{"readable": true, "config": safe}
		if path == "/var/lib/vpn-node/e2e/updater.json" {
			p.actual = safe
		}
	}
	return p
}

func (p *updateProbeCapture) capture() {
	if p.id != "AT23" && p.id != "AT25" {
		return
	}
	raw, err := os.ReadFile(filepath.Join(p.s.r.lab.state, "node", "var", "update-journal", "health-challenge.json"))
	if err != nil {
		return
	}
	var ch struct {
		Nonce    string    `json:"nonce"`
		IssuedAt time.Time `json:"issued_at"`
	}
	if json.Unmarshal(raw, &ch) != nil || ch.Nonce == "" || ch.IssuedAt.Before(p.start) || p.seen[ch.Nonce] {
		return
	}
	p.seen[ch.Nonce] = true
	p.challenges = append(p.challenges, json.RawMessage(raw))
}

func (p *updateProbeCapture) finish() {
	if p.id != "AT23" && p.id != "AT25" {
		return
	}
	s := p.s
	// Read the actual config again inside the node after the trigger was consumed.
	raw, err := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.json.done")
	if err == nil {
		actual := document(raw)
		if p.actual == nil {
			p.actual = actual
		}
	}
	agentConfig := p.actual.text("agent_config")
	if agentConfig == "" {
		agentConfig = "/etc/vpn-node/agent.json"
	}
	_, agentErr := s.r.lab.docker(s.r.lab.node, "test", "-f", agentConfig)
	hasPeer := len(p.actual["probe_peer"]) > 0 && string(p.actual["probe_peer"]) != "null"
	out, logErr := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.out")
	omissions := []map[string]any{}
	for _, line := range strings.Split(string(out), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil && event["msg"] == "health challenge probe omitted" {
			omissions = append(omissions, map[string]any{"time": event["time"], "message": event["msg"], "reason": event["reason"]})
		}
	}
	carries := false
	for _, raw := range p.challenges {
		ch := document(raw)
		carries = carries || sameJSON(ch["probe"], p.actual["probe_peer"])
	}
	finding := "no probe peer in lab"
	ok := p.actual != nil && logErr == nil
	if hasPeer {
		finding = "configured probe must appear in fresh challenge with no omission diagnostic"
		ok = ok && carries && len(omissions) == 0
	}
	evidence := map[string]any{"configurations_read_in_node": p.configurations, "actual_updater_config": p.actual, "has_probe_peer": hasPeer, "agent_config_path": agentConfig, "agent_config_exists": agentErr == nil, "fresh_challenges": p.challenges, "omission_logs": omissions, "stderr_readable": logErr == nil, "config_load_invalid_probe_peer": strings.Contains(string(out), "updater: invalid probe_peer"), "finding": finding}
	if hasPeer && agentErr != nil {
		evidence["expected_refusal_class"] = "agent_config_unavailable"
		evidence["owner"] = "w3-devstack-e2e"
	}
	if p.id == "AT25" && len(p.challenges) == 0 {
		evidence["challenge_observation"] = "no fresh challenge during pre-health tampered-archive rejection"
	}
	s.note(p.id, "probe_issuance", evidence)
	s.note(p.id, "check_probe_inspection", ok)
}
