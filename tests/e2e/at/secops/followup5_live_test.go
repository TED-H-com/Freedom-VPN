//go:build e2e && integration

package secops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

type flowPeer struct {
	Service     string       `json:"container_role"`
	NodeService string       `json:"node_role,omitempty"`
	ExitID      string       `json:"exit_network_id,omitempty"`
	Device      uuid.UUID    `json:"device_id"`
	Member      *liveSession `json:"-"`
}

// §19 AT28–AT31: product inputs and native client exit selection only (D-050).
func TestAT28_AT31_Followup5(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_FOLLOWUP5") != "1" {
		t.Skip("requires fresh assigned w4-secops stack")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 5, "run_id": uuid.New(), "hardware": "shared development host; not candidate hardware"}}
	p.open()
	defer p.db.Close(context.Background())
	defer p.tr.CloseIdleConnections()
	defer p.finish()
	admin := p.login("e2e-admin", "admin")
	members := make([]*liveSession, 20)
	for i := range members {
		members[i] = p.login(fmt.Sprintf("secops-member-%02d", i), "member")
		time.Sleep(time.Second)
	}
	p.metadata["members_authenticated"] = len(members)
	p.roles(admin, members[0])
	p.planInputs(admin)
	p.grantMembers(admin, members)
	p.releaseSupport(admin)
	node := p.enrollAgent(admin)
	nodeB, clientB := p.enrollHost(admin)
	p.identities(admin, clientB, nodeB, node)
	path := "/api/v1/admin/nodes/" + node.String()
	detail := p.call(admin, "GET", path, nil, nil)
	p.check("finite capacity input", p.call(admin, "PATCH", path, map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}, map[string]string{"If-Match": detail.header.Get("ETag")}), 202, "")
	time.Sleep(12 * time.Second)
	// Publish empty, then let the real agent satisfy the populated generation.
	p.serving(admin, node)
	devices := p.admitMember(members)
	p.serving(admin, node)
	if len(devices[0]) == 0 {
		p.metadata["followup5_blocker"] = "native first member not admitted"
		return
	}
	peers := []flowPeer{{Service: "member", Device: devices[0][0], Member: members[0]}}
	initial := p.selectedFlow(peers[0])
	p.metadata["selected_exit_flow"] = initial
	p.metadata["pre_selection_probe"] = p.metadata["flow_probe"]
	p.metadata["flow_probe"] = initial
	p.metadata["counted_flow"] = initial["counted"]
	if initial["counted"] == true {
		delete(p.metadata, "serving_blocker")
		p.metadata["serving_flow_positive"] = map[string]any{"status": "passed", "evidence": "selected_exit_flow"}
	}
	p.rateClasses(clientB, nodeB, members, devices)
	p.otherBodyBoundaries(admin, clientB, nodeB, node)
	p.concurrentLogins(members)
	if initial["counted"] != true {
		p.metadata["followup5_blocker"] = "explicit selected exit did not produce ordered counted-flow proof"
		p.revoke(admin, clientB, nodeB)
		return
	}
	stack, err := labstack.Open()
	if err != nil {
		t.Fatal("native helper open")
	}
	defer stack.Close()
	if os.Getenv("FVPN_SECOPS_SCALE60") == "1" {
		p.scaleNative(members)
		// Resolve only product-admitted native peer names, never write mappings.
		peers = p.cohortPeers(members)
	} else {
		p.metadata["native_scale_gate"] = map[string]any{"status": "blocked", "owner": "w3-devstack-e2e follow-up 10", "reason": "not present on measured main; retain historical 50/60 admission; drive a two-member counted subset"}
		if peer, ok := p.joinFlowPeer(stack, members[1], "secops-member-01", "secops-counted-b"); ok {
			peers = append(peers, peer)
		}
	}
	time.Sleep(12 * time.Second)
	p.countedCapacity(admin, node, members, peers)
	p.deadlineMatrix(node, stack)
	admin = p.login("e2e-admin", "admin")
	p.revokeCoherence(node, peers)
	p.revoke(admin, clientB, nodeB)
	p.metadata["physical_restore"] = "AT30 original PITR evidence retained; no new physical restore in Follow-up 5"
}

func (p *liveProbe) joinFlowPeer(stack *labstack.Stack, m *liveSession, subject, name string) (flowPeer, bool) {
	c, err := stack.StartNativePeer(p.t.Context(), name)
	if err != nil {
		p.metadata["native_join_blocker"] = "StartNativePeer failed"
		return flowPeer{}, false
	}
	r, err := stack.JoinNativeMember(p.t.Context(), labstack.BrowserSession{Client: m.client, CSRF: m.csrf, MemberID: m.member}, subject, c, c)
	if os.Getenv("FVPN_SECOPS_FOLLOWUP10") == "1" {
		p.recordNativeStream10(stack, r.Stream, c)
	}
	if err != nil || !r.Authorized || !r.IdentityMatched {
		p.metadata["native_join_blocker"] = "native join or identity/read-back failed"
		return flowPeer{}, false
	}
	var id uuid.UUID
	if err = p.db.QueryRow(p.t.Context(), `SELECT id FROM member_devices WHERE member_id=$1 AND status='authorized' ORDER BY created_at DESC LIMIT 1`, m.member).Scan(&id); err != nil {
		return flowPeer{}, false
	}
	return flowPeer{Service: c[len(livePrefix):], Device: id, Member: m}, true
}
func (p *liveProbe) cohortPeers(members []*liveSession) []flowPeer {
	out := []flowPeer{}
	// Hostnames are source inputs to the native hook, not user-controlled SQL.
	for i, m := range members {
		rows, err := p.db.Query(p.t.Context(), `SELECT id,name FROM member_devices WHERE member_id=$1 AND status='authorized' ORDER BY created_at`, m.member)
		if err != nil {
			p.t.Error("cohort read")
			continue
		}
		for rows.Next() {
			var id uuid.UUID
			var name string
			if rows.Scan(&id, &name) != nil {
				continue
			}
			service := name
			if i == 0 && len(out) == 0 && len(p.bindings) > 0 {
				service = "member"
			} else if len(name) > len(livePrefix) && name[:len(livePrefix)] == livePrefix {
				service = name[len(livePrefix):]
			}
			out = append(out, flowPeer{Service: service, Device: id, Member: m})
		}
		rows.Close()
	}
	return out
}
func (p *liveProbe) revokeCoherence(node uuid.UUID, peers []flowPeer) {
	if len(peers) < 2 {
		p.metadata["post_revoke"] = map[string]any{"status": "blocked", "reason": "second counted member unavailable"}
		return
	}
	before := p.selectedFlow(peers[1])
	stamp := time.Now().UTC()
	r := p.call(peers[0].Member, "DELETE", "/api/v1/member/devices/"+peers[0].Device.String(), nil, nil)
	observations := []json.RawMessage{}
	for until := time.Now().Add(40 * time.Second); time.Now().Before(until); {
		var raw []byte
		e := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('received_at',received_at,'applied_config_generation',report_payload->'applied_config_generation','member_forwarding_enabled',report_payload->'member_forwarding_enabled','forwarding_block_reasons',report_payload->'forwarding_block_reasons','checks',report_payload->'checks') FROM node_observations WHERE node_id=$1 AND received_at>$2 ORDER BY received_at LIMIT 1`, node, stamp).Scan(&raw)
		if e == nil {
			observations = append(observations, json.RawMessage(raw))
			break
		}
		time.Sleep(time.Second)
	}
	time.Sleep(12 * time.Second)
	p.metadata["post_revoke"] = map[string]any{"utc": stamp, "request_status": r.status, "code": r.code, "second_before": before, "next_observations": observations, "later_observation": p.publicationObservation(node), "second_after": p.selectedFlow(peers[1]), "revoked_after": p.selectedFlow(peers[0]), "conntrack_installed": p.conntrackPresent()}
}
func (p *liveProbe) conntrackPresent() bool {
	_, e := p.docker("node", "sh", "-c", "command -v conntrack")
	return e == nil
}
