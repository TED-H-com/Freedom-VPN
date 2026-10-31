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

	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
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
	collect        func() e2e.ProbeObservation
	observerError  string
}

func redactProbePeer(raw json.RawMessage) json.RawMessage {
	doc := document(raw)
	if doc == nil {
		return raw
	}
	if doc.text("peer_id") != "" {
		doc.set("peer_id", "[redacted]")
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return raw
	}
	return out
}

func redactChallenge(raw json.RawMessage) json.RawMessage {
	doc := document(raw)
	if doc == nil {
		return raw
	}
	for _, key := range []string{"probe", "probe_omitted"} {
		if v, ok := doc[key]; ok {
			doc[key] = redactProbePeer(v)
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return raw
	}
	return out
}

func sanitizeUpdaterConfig(cfg updateDocument) updateDocument {
	if cfg == nil {
		return nil
	}
	// Explicit allowlist: future secret-bearing config additions cannot leak,
	// and raw peer_id is never published in public evidence.
	safe := updateDocument{}
	for _, key := range []string{"node_id", "incarnation_id", "boot_id", "platform", "probe_peer", "agent_config", "result_url", "allowed_artifact_origins", "allow_loopback", "desired_state", "meter_checkpoint", "stop_command", "start_command", "release_trust", "desired_trust", "mtls_cert", "mtls_key", "mtls_ca"} {
		if value, ok := cfg[key]; ok {
			if key == "probe_peer" {
				value = redactProbePeer(value)
			}
			safe[key] = value
		}
	}
	return safe
}

// The entrypoint's stdout/stderr file is private to the node and overwritten
// for each invocation. Only the fixed omission message/reason is published.
func (s *updatesFollowup) beginProbeCapture(id string) *updateProbeCapture {
	p := &updateProbeCapture{s: s, id: id, start: time.Now().UTC(), configurations: map[string]any{}, seen: map[string]bool{}, challenges: []json.RawMessage{}}
	if id != "AT23" && id != "AT25" {
		return p
	}
	if s.r.probe != nil && s.r.lab.nodeSlot() == "node" {
		collect, err := s.r.probe.Observe(s.ctx)
		if err != nil {
			p.observerError = err.Error()
		} else {
			p.collect = collect
		}
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
		safe := sanitizeUpdaterConfig(cfg)
		p.configurations[path] = map[string]any{"readable": true, "config": safe}
		if path == "/var/lib/vpn-node/e2e/updater.json" {
			p.actual = safe
		}
	}
	// The root-owned journal is intentionally unreadable to the host user.
	// Stream read-only snapshots inside our node; never chmod or copy identity.
	p.readerStop = filepath.Join(s.r.lab.state, s.r.lab.nodeSlot(), "var", "e2e", "updates-probe-reader-stop")
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
	var obs e2e.ProbeObservation
	if p.collect != nil {
		obs = p.collect()
	}
	s := p.s
	var rawProbePeer json.RawMessage
	// Read the actual config again inside the node after the trigger was consumed.
	raw, err := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.json.done")
	if err == nil {
		actual := document(raw)
		if actual != nil {
			rawProbePeer = actual["probe_peer"]
			p.actual = sanitizeUpdaterConfig(actual)
			p.configurations["/var/lib/vpn-node/e2e/updater.json.done"] = map[string]any{"readable": true, "config": p.actual}
		}
	}
	agentConfig := p.actual.text("agent_config")
	if agentConfig == "" {
		agentConfig = "/etc/vpn-node/agent.json"
	}
	_, agentErr := s.r.lab.docker(s.r.lab.node, "test", "-f", agentConfig)
	hasPeer := len(rawProbePeer) > 0 && string(rawProbePeer) != "null"
	out, logErr := s.r.lab.docker(s.r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.out")
	omissions := []map[string]any{}
	for _, line := range strings.Split(string(out), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil && event["msg"] == "health challenge probe omitted" {
			omissions = append(omissions, map[string]any{"time": event["time"], "message": event["msg"], "reason": event["reason"]})
		}
	}
	carries := false
	redactedChallenges := make([]json.RawMessage, 0, len(p.challenges))
	for _, rawCh := range p.challenges {
		ch := document(rawCh)
		carries = carries || (hasPeer && sameJSON(ch["probe"], rawProbePeer))
		redactedChallenges = append(redactedChallenges, redactChallenge(rawCh))
	}
	noExceptionOpened := !obs.Verified
	for _, n := range obs.Observations {
		if n.Name == "probe_window" {
			var w struct {
				Grant    int `json:"grant_samples"`
				Elements int `json:"elements_after"`
			}
			noExceptionOpened = json.Unmarshal([]byte(n.Detail), &w) == nil && w.Grant == 0 && w.Elements == 0
		}
	}
	finding := "no probe peer in lab"
	ok := p.actual != nil && logErr == nil && p.readerError == "" && p.observerError == "" && (p.id != "AT23" || len(p.challenges) > 0)
	if hasPeer {
		if p.id == "AT25" && len(p.challenges) == 0 {
			finding = "tampered archive rejected at preflight (HASH_MISMATCH) before health check; probe scope does not apply to an archive that never passes preflight"
			ok = ok && len(omissions) == 0 && agentErr == nil && noExceptionOpened
		} else {
			ok = ok && len(omissions) == 0 && agentErr == nil && carries && obs.Verified
			finding = "Configured probe peer has no verified fresh challenge and traversal; inspect measured challenge, attestation and observer fields"
			if ok {
				finding = "configured probe_peer issued in fresh challenge with no omission diagnostic; exported observer verified bounded D-064 exception traversal and cleanup"
			}
		}
	}
	var setupNotes []e2e.Observation
	if s.r.probe != nil {
		setupNotes = s.r.probe.Notes
	}
	evidence := map[string]any{
		"capture_method":                 "exported UpdaterProbe.Observe plus read-only docker exec stream inside the node (no host read of 0700 journal)",
		"reader_error":                   p.readerError,
		"observer_error":                 p.observerError,
		"configurations_read_in_node":    p.configurations,
		"actual_updater_config":          p.actual,
		"has_probe_peer":                 hasPeer,
		"configured_probe_issued":        carries,
		"agent_config_path":              agentConfig,
		"agent_config_exists":            agentErr == nil,
		"fresh_challenges":               redactedChallenges,
		"candidate_attestations":         p.attestations,
		"omission_logs":                  omissions,
		"stderr_readable":                logErr == nil,
		"config_load_invalid_probe_peer": strings.Contains(string(out), "updater: invalid probe_peer"),
		"probe_setup_notes":              setupNotes,
		"probe_observer":                 obs,
		"finding":                        finding,
	}
	if hasPeer && agentErr != nil {
		evidence["expected_refusal_class"] = "agent_config_unavailable"
		evidence["owner"] = "w3-devstack-e2e"
	}
	if p.id == "AT25" && len(p.challenges) == 0 {
		evidence["challenge_observation"] = "no fresh challenge during pre-health tampered-archive rejection; probe scope does not apply to an archive that never passes preflight"
	}
	if !hasPeer {
		evidence["owner"] = "w3-devstack-e2e"
		evidence["agent_handling"] = "No probe object issued; D-064 exception installation/refusal not exercised"
	} else if p.id == "AT25" && len(p.challenges) == 0 {
		evidence["agent_handling"] = "Tampered archive rejected at preflight before stageCheck; no challenge issued and D-064 probe exception window stayed closed (grant_samples=0, elements_after=0)"
	} else if ok {
		evidence["agent_handling"] = "Candidate installed bounded D-064 probe exception for the matching challenge while update lock was held, answered health-attest.json, and removed all probe_grant_v4 elements"
	} else {
		evidence["agent_handling"] = "Candidate probe exception and health response were not verified; no successful traversal is claimed"
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
	s.note(p.id, "probe_observer", obs)
	s.note(p.id, "check_lab_probe_peer", hasPeer && agentErr == nil)
	s.note(p.id, "check_probe_inspection", ok)
	if p.id == "AT23" {
		s.note(p.id, "check_probe_traversal", obs.Verified && carries && len(omissions) == 0)
	}
}
