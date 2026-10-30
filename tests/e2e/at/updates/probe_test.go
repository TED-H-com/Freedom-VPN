//go:build e2e

package updates_test

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
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
	attestations   []json.RawMessage
	reader         *exec.Cmd
	readerStop     string
	readerError    string
	done           chan struct{}
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
	// The root-owned journal is intentionally unreadable to the host user.
	// Stream read-only snapshots inside our node; never chmod or copy identity.
	p.readerStop = filepath.Join(s.r.lab.state, "node", "var", "e2e", "updates-probe-reader-stop")
	_ = os.Remove(p.readerStop)
	p.reader = exec.CommandContext(s.ctx, "docker", "exec", s.r.lab.node, "sh", "-c", `
 i=0
 while [ "$i" -lt 6000 ] && [ ! -e /var/lib/vpn-node/e2e/updates-probe-reader-stop ]; do
   for kind in challenge attest; do
     raw=$(cat "/var/lib/vpn-node/update-journal/health-$kind.json" 2>/dev/null) && printf '{"kind":"%s","value":%s}\n' "$kind" "$raw"
   done
   i=$((i+1))
   sleep 0.05
 done`)
	out, err := p.reader.StdoutPipe()
	if err != nil {
		s.r.t.Fatal("probe snapshot pipe failed")
	}
	if err = p.reader.Start(); err != nil {
		s.r.t.Fatal("probe snapshot reader failed")
	}
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		decoder := json.NewDecoder(out)
		for {
			var event struct {
				Kind  string          `json:"kind"`
				Value json.RawMessage `json:"value"`
			}
			if err := decoder.Decode(&event); err != nil {
				if err != io.EOF {
					p.readerError = "invalid node snapshot stream"
				}
				break
			}
			p.capture(event.Kind, event.Value)
		}
		if err := p.reader.Wait(); err != nil {
			p.readerError = "node snapshot reader failed"
		}
	}()
	return p
}

func (p *updateProbeCapture) capture(kind string, raw json.RawMessage) {
	var ch struct {
		Nonce    string    `json:"nonce"`
		IssuedAt time.Time `json:"issued_at"`
	}
	if json.Unmarshal(raw, &ch) != nil || ch.Nonce == "" {
		return
	}
	if kind == "challenge" && !ch.IssuedAt.Before(p.start) && !p.seen[ch.Nonce] {
		p.seen[ch.Nonce] = true
		p.challenges = append(p.challenges, raw)
	}
	if kind == "attest" && p.seen[ch.Nonce] && !p.seen["att:"+ch.Nonce] {
		p.seen["att:"+ch.Nonce] = true
		p.attestations = append(p.attestations, raw)
	}
}

func (p *updateProbeCapture) finish() {
	if p.id != "AT23" && p.id != "AT25" {
		return
	}
	if os.WriteFile(p.readerStop, []byte("stop\n"), 0o600) != nil {
		p.s.r.t.Fatal("stop snapshot reader failed")
	}
	<-p.done
	_ = os.Remove(p.readerStop)
	s := p.s
	// Read the actual config again inside the node after the trigger was consumed.
	raw, err := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.json.done")
	if err == nil {
		actual := document(raw)
		p.actual = updateDocument{}
		for _, key := range []string{"node_id", "incarnation_id", "boot_id", "platform", "probe_peer", "agent_config", "result_url", "desired_state", "meter_checkpoint", "stop_command", "start_command", "release_trust", "desired_trust", "mtls_cert", "mtls_key", "mtls_ca"} {
			if v, exists := actual[key]; exists {
				p.actual[key] = v
			}
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
	ok := p.actual != nil && logErr == nil && p.readerError == "" && (p.id != "AT23" || len(p.challenges) > 0)
	if hasPeer {
		finding = "configured probe must appear in fresh challenge with no omission diagnostic"
		ok = ok && len(omissions) == 0 && agentErr == nil && (carries || p.id == "AT25" && len(p.challenges) == 0)
	}
	evidence := map[string]any{"capture_method": "read-only docker exec stream before updater trigger", "reader_error": p.readerError, "configurations_read_in_node": p.configurations, "actual_updater_config": p.actual, "has_probe_peer": hasPeer, "agent_config_path": agentConfig, "agent_config_exists": agentErr == nil, "fresh_challenges": p.challenges, "candidate_attestations": p.attestations, "omission_logs": omissions, "stderr_readable": logErr == nil, "config_load_invalid_probe_peer": strings.Contains(string(out), "updater: invalid probe_peer"), "finding": finding}
	if hasPeer && agentErr != nil {
		evidence["expected_refusal_class"] = "agent_config_unavailable"
		evidence["owner"] = "w3-devstack-e2e"
	}
	if p.id == "AT25" && len(p.challenges) == 0 {
		evidence["challenge_observation"] = "no fresh challenge during pre-health tampered-archive rejection"
	}
	if !hasPeer {
		evidence["owner"] = "w3-devstack-e2e"
		evidence["agent_handling"] = "No probe object issued; D-064 exception installation/refusal not exercised"
	} else {
		evidence["agent_handling"] = "Inspect captured challenge/attestation and fixed agent diagnostics; attestation alone does not prove exception installation"
	}
	agentLog, agentLogErr := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/e2e/agent.log")
	agentEvents := []map[string]any{}
	for _, line := range strings.Split(string(agentLog), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil && (event["msg"] == "probe exception" || event["msg"] == "health attestation") {
			safe := map[string]any{}
			for _, key := range []string{"time", "msg", "reason", "err"} {
				if v, exists := event[key]; exists {
					safe[key] = v
				}
			}
			agentEvents = append(agentEvents, safe)
		}
	}
	evidence["agent_diagnostics"] = agentEvents
	evidence["agent_log_readable"] = agentLogErr == nil
	evidence["authority_source"] = "node/agent/probeexc.go:66-71,125-127: matching challenge and another lock holder required; no probe installs nothing"
	s.note(p.id, "probe_issuance", evidence)
	s.note(p.id, "check_lab_probe_peer", hasPeer && agentErr == nil)
	s.note(p.id, "check_probe_inspection", ok)
}
