//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// Read the real HTTP compiler with the node's own mTLS identity. The private
// key remains inside the enrolled node. Record only bounded response metadata.
func (p *liveProbe) desiredHTTP11() map[string]any {
	raw, err := p.docker("node", "wget", "--server-response", "--content-on-error", "--timeout=10", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "--certificate=/var/lib/vpn-node/identity/cert.pem", "--private-key=/var/lib/vpn-node/identity/key.pem", "-O", "-", "https://agent.example.invalid:9443/api/v1/agent/desired-state")
	var body struct {
		Signed struct {
			Generation string `json:"config_generation"`
		} `json:"signed"`
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}
	i := bytes.IndexByte(raw, '{')
	decoded := i >= 0 && json.Unmarshal(bytes.TrimSpace(raw[i:]), &body) == nil
	return map[string]any{"utc": time.Now().UTC(), "http_200": bytes.Contains(raw, []byte("HTTP/1.1 200")), "http_503": bytes.Contains(raw, []byte("HTTP/1.1 503")), "command_ok": err == nil, "decoded": decoded, "generation": body.Signed.Generation, "error_code": body.Error.Code, "message": body.Error.Message, "retryable": body.Error.Retryable}
}
func (p *liveProbe) restoreHoldSnapshot11(node uuid.UUID) json.RawMessage {
	var raw []byte
	err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('utc',clock_timestamp(),'drift',n.drift,'desired',n.desired_config_generation::text,'applied',n.applied_config_generation::text,'stored_count',(SELECT count(*) FROM desired_states WHERE node_id=n.id),'envelope_md5',(SELECT md5(envelope::text) FROM desired_state_envelopes WHERE node_id=n.id),'hold_operations',(SELECT jsonb_agg(jsonb_build_object('status',status,'steps',steps,'created_at',created_at)) FROM operations WHERE resource_id=n.id AND principal='system:restore-observation')) FROM nodes n WHERE id=$1`, node).Scan(&raw)
	if err != nil {
		p.t.Fatal("restore hold snapshot", err)
	}
	return raw
}
func (p *liveProbe) verifyRestoreHold11(node uuid.UUID) {
	before := p.restoreHoldSnapshot11(node)
	response := p.desiredHTTP11()
	admin := p.login("e2e-admin", "admin")
	detail := p.call(admin, "GET", "/api/v1/admin/nodes/"+node.String(), nil, nil)
	var nodeView struct {
		Drift json.RawMessage `json:"drift"`
	}
	if json.Unmarshal(detail.body, &nodeView) != nil {
		p.t.Fatal("node hold detail decode")
	}
	after := p.restoreHoldSnapshot11(node)
	var a, b map[string]any
	_ = json.Unmarshal(before, &a)
	_ = json.Unmarshal(after, &b)
	passed := response["http_503"] == true && response["decoded"] == true && response["error_code"] == "DEPENDENCY_UNAVAILABLE" && response["message"] == "pending fresh observation after restore" && response["retryable"] == true && a["desired"] == b["desired"] && a["stored_count"] == b["stored_count"] && a["envelope_md5"] == b["envelope_md5"] && detail.status == 200 && bytes.Contains(nodeView.Drift, []byte("pending fresh observation after restore"))
	p.metadata["D-095"] = map[string]any{"passed": passed, "before": before, "response": response, "after": after, "node_get_status": detail.status, "node_drift": nodeView.Drift, "freeze_before_services_start": true, "agent_stopped_for_pre_observation_probe": true}
	if !passed {
		p.t.Fatal("D-095 pre-observation HTTP/node hold not proved")
	}
	p.t.Log("D-095: pre-observation HTTP 503 and node drift pending fresh observation after restore; no desired/envelope change")
}
func (p *liveProbe) stopRestoreAgent11() func() {
	stack, err := labstack.Open()
	if err != nil {
		p.t.Fatal(err)
	}
	if err = stack.Service(p.t.Context(), "stop", ""); err != nil {
		p.t.Fatal("stop retained agent", err)
	}
	return func() {
		if err := stack.Service(context.Background(), "start", "/usr/local"); err != nil {
			p.t.Error("restart retained agent", err)
		}
		stack.Close()
	}
}
func (p *liveProbe) sampleFirstHTTPAfterObservation11(node uuid.UUID, since time.Time) {
	for until := time.Now().Add(45 * time.Second); time.Now().Before(until); {
		var count int
		if p.db.QueryRow(p.t.Context(), `SELECT count(*) FROM node_observations WHERE node_id=$1 AND received_at >= $2`, node, since).Scan(&count) == nil && count > 0 {
			p.metadata["first_http_probe_after_fresh_observation"] = p.desiredHTTP11()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	p.metadata["first_http_probe_after_fresh_observation"] = map[string]any{"blocked": "no fresh observation within 45 seconds"}
}
func (p *liveProbe) centralStopped11() bool {
	for _, s := range []string{"control", "worker"} {
		raw, err := exec.CommandContext(p.t.Context(), "docker", "inspect", "--format", "{{.State.Running}}", livePrefix+s).Output()
		if err != nil || string(bytes.TrimSpace(raw)) != "false" {
			return false
		}
	}
	return true
}
