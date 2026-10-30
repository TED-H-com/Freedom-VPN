//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// §19 AT28/29/31: D-050 source inputs, actual addresses and counted packets.
func TestAT28_AT31_Followup7(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_FOLLOWUP7") != "1" {
		t.Skip("requires fresh assigned finite-grace stack")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 7, "run_id": uuid.New(), "hardware": "shared development host; not candidate hardware"}}
	p.open()
	defer p.db.Close(context.Background())
	defer p.tr.CloseIdleConnections()
	defer p.finish()
	stack, err := labstack.Open()
	if err != nil {
		t.Fatal("stack open")
	}
	defer stack.Close()
	admin := p.login("e2e-admin", "admin")
	p.planInputs(admin)
	p.proxyPrincipals(stack, admin)
	p.releaseSupport(admin)
	node := p.enrollAgent(admin)
	p.capacityInput(admin, node)
	p.serving(admin, node)
	members := make([]*liveSession, 20)
	for i := range members {
		members[i] = p.login(fmt.Sprintf("secops-member-%02d", i), "member")
		time.Sleep(time.Second)
	}
	p.grantMembers(admin, members)
	collect, err := stack.StartNodeTelemetry(t.Context(), 5*time.Second)
	if err != nil {
		t.Fatal("node telemetry start")
	}
	defer collect()
	p.metadata["gateway_before"] = p.gatewayInventory()
	p.scaleNative(members)
	p.cohortSnapshot(members)
	p.metadata["admission_node_telemetry"] = collect()
	peers := p.cohortPeers(members)
	p.protectedArtifacts(stack, peers)
	admin = p.login("e2e-admin", "admin")
	p.serving(admin, node)
	collect, err = stack.StartNodeTelemetry(t.Context(), 5*time.Second)
	if err != nil {
		t.Fatal("traffic telemetry start")
	}
	defer collect()
	p.countedCapacity(admin, node, members, peers)
	p.metadata["traffic_node_telemetry"] = collect()
	p.metadata["gateway_after"] = p.gatewayInventory()
	// Independent API subset continues even if one capacity peer lacks traffic.
	admin = p.login("e2e-admin", "admin")
	nodeB, cert := p.enrollHost(admin)
	p.anonymousIsolation(admin, cert, nodeB)
	p.loadCurrentBinding(members[0])
	devices := make([][]uuid.UUID, len(members))
	for id := range p.bindings {
		devices[0] = append(devices[0], id)
	}
	p.rateClasses(cert, nodeB, members, devices)
	p.deadlineMatrix(node, stack)
	p.twoExitTraffic(stack, node, peers)
	p.metadata["physical_restore"] = "unchanged since Follow-up 6; no new physical restore claim"
}

func (p *liveProbe) capacityInput(admin *liveSession, node uuid.UUID) {
	path := "/api/v1/admin/nodes/" + node.String()
	before := p.call(admin, "GET", path, nil, nil)
	p.check("finite capacity", p.call(admin, "PATCH", path, map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}, map[string]string{"If-Match": before.header.Get("ETag")}), 202, "")
	time.Sleep(12 * time.Second)
}
func (p *liveProbe) loadCurrentBinding(m *liveSession) {
	p.bindings = map[uuid.UUID]api.DeviceBinding{}
	var b api.DeviceBinding
	var revision string
	e := p.db.QueryRow(p.t.Context(), `SELECT member_id,device_id,binding_revision::text,valid_from,valid_until FROM device_bindings WHERE member_id=$1 AND valid_from<=now() AND valid_until>now() ORDER BY binding_revision DESC LIMIT 1`, m.member).Scan(&b.MemberId, &b.DeviceId, &revision, &b.ValidFrom, &b.ValidUntil)
	if e != nil {
		p.t.Fatal("current product binding unavailable")
	}
	if json.Unmarshal([]byte(`"`+revision+`"`), &b.BindingRevision) != nil {
		p.t.Fatal("binding revision")
	}
	p.bindings[b.DeviceId] = b
}
func (p *liveProbe) protectedArtifacts(stack *labstack.Stack, peers []flowPeer) {
	rows := []map[string]any{}
	for _, peer := range peers {
		scan, err := stack.ScanNativeJoinLog(p.t.Context(), livePrefix+peer.Service)
		rows = append(rows, map[string]any{"container_role": peer.Service, "scan_ok": err == nil, "post_join_present": scan.Present, "deletion_proven": err == nil && !scan.Present})
	}
	p.metadata["protected_login_artifacts"] = map[string]any{"classification": "D-066 protected transient credential artifact, separate from readable files/logs/captures", "pre_delete_source": "native_scale.peer_observations[].transient_log_scan (JoinNativeMember)", "post_delete": rows, "bulk_helper_per_stream": "blocked on w3-devstack-e2e follow-up 12; NativeScale currently exposes aggregate totals only"}
}
func (p *liveProbe) proxyPrincipals(stack *labstack.Stack, admin *liveSession) {
	out := map[string]any{"status": "blocked", "class": "anonymous", "source": "actual distinct container TCP peers; exact trusted gateway /32; forged XFF documentation values"}
	p.metadata["trusted_proxy"] = out
	raw, _ := os.ReadFile(filepath.Join(p.state, "stack.json"))
	var cfg struct {
		LabPrefix string `json:"lab_prefix"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return
	}
	containers := []string{}
	for _, name := range []string{"proxy-a", "proxy-b"} {
		c, e := stack.StartNativePeer(p.t.Context(), name)
		if e != nil {
			return
		}
		containers = append(containers, c)
		if exec.CommandContext(p.t.Context(), "docker", "cp", filepath.Join(p.root, ".agent/secops/build/proxyprobe"), c+":/tmp/secops-proxyprobe").Run() != nil {
			return
		}
	}
	_, token := p.enrollment(admin, "TEST ONLY proxy-principal replay")
	req, _ := pkitest.Request(p.t)
	burst := func(index int, dial string, count int, forge, valid bool) []map[string]any {
		body, _ := json.Marshal(map[string]any{})
		headers := map[string]string{}
		if valid {
			body, _ = json.Marshal(req)
			headers["Authorization"] = "Enrollment " + token
		}
		input, _ := json.Marshal(map[string]any{"Dial": dial, "Method": "POST", "Path": "/api/v1/agent/enroll", "Body": json.RawMessage(body), "Headers": headers, "Count": count, "Forge": forge})
		cmd := exec.CommandContext(p.t.Context(), "docker", "exec", "-i", containers[index], "/tmp/secops-proxyprobe")
		cmd.Stdin = bytes.NewReader(input)
		raw, e := cmd.Output()
		var rows []map[string]any
		if e != nil || json.Unmarshal(raw, &rows) != nil {
			return nil
		}
		return rows
	}
	gateway := cfg.LabPrefix + ".21:8443"
	direct := cfg.LabPrefix + ".30:8443"
	out["source_b_enroll"] = burst(1, gateway, 1, false, true)
	out["metrics_before"] = p.classMetrics()
	a := burst(0, gateway, 120, false, false)
	b := burst(1, gateway, 1, false, true)
	out["gateway_a"] = a
	out["gateway_b_replay"] = b
	out["metrics_after_gateway"] = p.classMetrics()
	time.Sleep(6 * time.Second)
	forged := burst(0, direct, 120, true, false)
	other := burst(1, direct, 1, true, true)
	out["untrusted_direct_a_rotating_xff"] = forged
	out["untrusted_direct_b_replay"] = other
	out["metrics_after_direct"] = p.classMetrics()
	limited := func(rows []map[string]any) bool {
		for _, r := range rows {
			if r["status"] == float64(429) && r["code"] == "RATE_LIMITED" && r["retry_after"] != "" {
				return true
			}
		}
		return false
	}
	if limited(a) && limited(forged) && len(b) == 1 && b[0]["status"] == float64(200) && len(other) == 1 && other[0]["status"] == float64(200) {
		out["status"] = "passed"
	}
}
