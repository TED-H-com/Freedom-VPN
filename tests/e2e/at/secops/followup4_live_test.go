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

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// §19 AT28–AT31 / D-050: every mutation is an API command or an exported
// process/native-peer hook. In particular, no observation is synthesized.
func TestAT28_AT31_Followup4(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_LIVE") != "1" {
		t.Skip("requires assigned fresh w4-secops stack")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 4, "hardware": "shared development host; not candidate hardware"}}
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
	p.securityComplete = true
	p.roles(admin, members[0])
	p.planInputs(admin)
	p.grantMembers(admin, members)
	p.releaseSupport(admin)
	nodeA := p.enrollAgent(admin)
	nodeB, clientB := p.enrollHost(admin)
	p.identities(admin, clientB, nodeB, nodeA)
	t.Log("SECOPS-11: publish with no capacity and no admitted devices; real agent remains unarmed")
	p.serving(admin, nodeA)
	p.metadata["publication_without_capacity"] = p.publicationSnapshot(nodeA)
	path := "/api/v1/admin/nodes/" + nodeA.String()
	detail := p.call(admin, "GET", path, nil, nil)
	p.check("finite node capacity input", p.call(admin, "PATCH", path, map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}, map[string]string{"If-Match": detail.header.Get("ETag")}), 202, "")
	// Let the real agent acknowledge the capacity input before the next command.
	time.Sleep(12 * time.Second)
	t.Log("SECOPS-11: repeat publication with capacity and zero device bindings (#148)")
	p.serving(admin, nodeA)
	p.metadata["publication_empty_bindings"] = p.publicationSnapshot(nodeA)
	devices := p.admitMember(members)
	p.ownership(admin, members, devices)
	p.serving(admin, nodeA)
	p.metadata["publication_with_device"] = p.publicationSnapshot(nodeA)
	p.rateClasses(clientB, nodeB, members, devices)
	p.concurrentLogins(members)
	p.scaleNative(members)
	admin = p.login("e2e-admin", "admin")
	// The scale hook admits peers; it makes no claim that any flow is counted.
	p.serving(admin, nodeA)
	if p.metadata["counted_flow"] == true {
		p.load(members)
	} else {
		p.metadata["capacity_subset"] = map[string]any{"status": "blocked", "owner": "w3-devstack-e2e follow-up 9", "reason": "No counted serving flow; no mixed-load or deadline measurement starts."}
	}
	p.metadata["physical_restore"] = "Historical Follow-up 2 PITR evidence retained; no physical restore rerun in Follow-up 4."
	// A long scale run may outlive recent-auth; refresh by the documented login path.
	admin = p.login("e2e-admin", "admin")
	if len(devices[0]) > 0 {
		p.memberOperation(members)
	}
	p.revoke(admin, clientB, nodeB)
	p.loginThresholds(members)
}

func (p *liveProbe) publicationSnapshot(node uuid.UUID) map[string]any {
	out := map[string]any{"utc": time.Now().UTC()}
	for _, k := range []string{"publication_request", "publication_operation", "publication_negative", "publication_observation_before", "publication_observation_after", "node_after_publication", "watchdog_state", "flow_probe", "counted_flow", "serving_checks", "observed_lease_generation"} {
		out[k] = p.metadata[k]
	}
	var generation string
	var bindings int
	var intent string
	err := p.db.QueryRow(p.t.Context(), `SELECT generation::text,jsonb_array_length(envelope->'signed'->'device_bindings'),envelope->'signed'->>'publication_intent' FROM desired_state_envelopes WHERE node_id=$1 ORDER BY generation DESC LIMIT 1`, node).Scan(&generation, &bindings, &intent)
	out["signed_input_read_ok"] = err == nil
	if err == nil {
		out["signed_generation"] = generation
		out["signed_device_bindings"] = bindings
		out["signed_publication_intent"] = intent
	}
	return out
}

// SECOPS-10: exercise the documented StartNativePeer/JoinNativeMember hooks
// against the same 20 sessions and finite grants. Stop at the first unavailable
// precondition; retain exact counts instead of manufacturing the missing peers.
func (p *liveProbe) scaleNative(members []*liveSession) {
	stack, err := labstack.Open()
	if err != nil {
		p.t.Fatal("native scale stack open")
	}
	defer stack.Close()
	ctx, cancel := context.WithTimeout(p.t.Context(), 40*time.Minute)
	defer cancel()
	started := time.Now()
	attempts, matches, authorized := 0, 0, 0
	rows := []map[string]any{}
	out := map[string]any{"requested_users": 20, "requested_peers": 60, "hook": "StartNativePeer + JoinNativeMember", "projector_writes": false, "users": rows}
	defer func() {
		out["attempted_new_peers"] = attempts
		out["new_identity_matches"] = matches
		out["new_authorized_peers"] = authorized
		out["status"] = "blocked"
		if authorized+len(p.bindings) == 60 && len(rows) == 20 {
			out["status"] = "passed"
		}
		out["authorized_peer_observations"] = authorized + len(p.bindings)
		out["elapsed_seconds"] = time.Since(started).Seconds()
		out["users"] = rows
		p.metadata["native_scale"] = out
	}()
	for i, m := range members {
		subject := fmt.Sprintf("secops-member-%02d", i)
		first := 1
		if i == 0 && len(p.bindings) > 0 {
			first = 2
		}
		for peer := first; peer <= 3; peer++ {
			name := fmt.Sprintf("secops-u%02d-p%d", i, peer)
			container, e := stack.StartNativePeer(ctx, name)
			if e != nil {
				out["blocker"] = "StartNativePeer failed"
				return
			}
			attempts++
			result, e := stack.JoinNativeMember(ctx, labstack.BrowserSession{Client: m.client, CSRF: m.csrf, MemberID: m.member}, subject, container, container)
			if result.IdentityMatched {
				matches++
			}
			if result.Authorized {
				authorized++
			}
			if e != nil {
				out["blocker"] = fmt.Sprintf("JoinNativeMember user=%d peer=%d: %s", i, peer, e)
				return
			}
		}

		var r liveReply
		count := 0
		reads := []map[string]any{}
		for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
			r = p.call(m, "GET", "/api/v1/member/devices", nil, nil)
			count = 0
			var page api.MemberDeviceList
			valid := r.status == 200 && json.Unmarshal(r.body, &page) == nil
			if valid {
				for _, d := range page.Items {
					if d.Status == api.DeviceStatusAuthorized {
						count++
					}
				}
			}
			reads = append(reads, map[string]any{"utc": time.Now().UTC(), "http_status": r.status, "code": r.code, "valid_device_page": valid})
			if valid && count == 3 {
				break
			}
			time.Sleep(2 * time.Second)
		}
		row := map[string]any{"user_index": i, "authorized": count, "http_status": r.status, "code": r.code, "reads": reads}
		if r.status != 200 {
			row["authorized"] = nil
		}
		rows = append(rows, row)
		p.t.Logf("SECOPS-10 native user %d/20: %d authorized peers", i+1, count)
		if count != 3 {
			out["blocker"] = "member device list did not contain exactly three authorized peers"
			return
		}
	}
	p.metadata["authorized_devices"] = 60
	delete(p.metadata, "device_scale_blocker")
	out["status"] = "passed"
}

// Isolated SECOPS-10 follow-up: admission needs no enrolled serving node.
// This never replaces the earlier publication/traffic measurements.
func TestSECOPS10_NativeScale(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_SCALE_ONLY") != "1" {
		t.Skip("requires fresh assigned native-scale stack")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "purpose": "isolated 20-user × 3-peer admission recheck; no forwarding node or capacity claim", "node_incarnation": nil}}
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
	p.planInputs(admin)
	p.grantMembers(admin, members)
	p.scaleNative(members)
}
