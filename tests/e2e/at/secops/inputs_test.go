//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
)

// D-050 / D-057: configuration is supplied through the group and policy APIs.
// Plans, revisions and member projections are product-written, never SQL fixtures.
func (p *liveProbe) planInputs(admin *liveSession) {
	r := p.call(admin, "POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "secops-input", Name: "TEST ONLY secops"}, nil)
	p.check("create member group input", r, 201, "")
	var group api.MemberGroup
	if json.Unmarshal(r.body, &group) != nil || group.GroupId == uuid.Nil {
		p.t.Fatal("group input absent")
	}
	p.groupID = group.GroupId
	raw, err := os.ReadFile(filepath.Join(p.root, "contracts/examples/policies/fair-use.json"))
	if err != nil {
		p.t.Fatal("fair-use input")
	}
	var fair map[string]any
	if json.Unmarshal(raw, &fair) != nil {
		p.t.Fatal("fair-use decode")
	}
	now := time.Now().UTC()
	fair["overrides"] = []any{}
	fair["effective_at"] = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	fup := p.policyInput(admin, "fair_use", fair)
	dns := p.policyInput(admin, "dns_profile", map[string]any{"profile_code": "secops-input", "display_name": map[string]string{"zh-TW": "TEST ONLY"}, "domain_rule_set_revision_ids": []string{}, "threat_feed_revision_ids": []string{}, "block_response": "nxdomain", "blocked_ttl_seconds": 60})
	p.planRevision = p.policyInput(admin, "plan", map[string]any{"display_name": map[string]string{"zh-TW": "TEST ONLY"}, "device_limit": 3, "exit_group_ids": []uuid.UUID{p.groupID}, "price": map[string]any{"amount_minor": 1000, "currency": "USD", "billing_interval": "month", "provider": "stripe", "provider_price_ref": "TEST-ONLY-secops"}, "rate_limit": nil, "fair_use_revision_id": fup, "dns_profile_revision_id": dns})
	p.metadata["sql_input_fixtures"] = []string{}
	p.metadata["policy_revision"] = p.planRevision
}

func (p *liveProbe) policyInput(admin *liveSession, kind string, payload any) uuid.UUID {
	r := p.call(admin, "POST", "/api/v1/admin/policies", map[string]any{"type": kind, "code": "secops-" + kind, "schema_version": 1, "reason": "TEST ONLY acceptance input", "payload": payload}, nil)
	p.check("create "+kind+" policy input", r, 201, "")
	var revision struct {
		ID         uuid.UUID `json:"revision_id"`
		Definition uuid.UUID `json:"definition_id"`
	}
	if json.Unmarshal(r.body, &revision) != nil || revision.ID == uuid.Nil {
		p.t.Fatal("policy input absent")
	}
	r = p.call(admin, "GET", "/api/v1/admin/policies?type="+kind, nil, nil)
	var page api.PolicyDefinitionList
	if json.Unmarshal(r.body, &page) != nil {
		p.t.Fatal("policy input list")
	}
	var version string
	for _, item := range page.Items {
		if item.DefinitionId == revision.Definition {
			version = item.ResourceVersion.String()
		}
	}
	if version == "" {
		p.t.Fatal("policy version absent")
	}
	r = p.call(admin, "POST", "/api/v1/admin/policies/"+revision.Definition.String()+"/activate", map[string]any{"revision_id": revision.ID, "reason": "TEST ONLY activation"}, map[string]string{"If-Match": `"` + version + `"`})
	p.check("activate "+kind+" policy input", r, 202, "")
	for deadline := time.Now().Add(90 * time.Second); time.Now().Before(deadline); {
		r = p.call(admin, "GET", "/api/v1/admin/policies?type="+kind, nil, nil)
		if r.status == 200 && json.Unmarshal(r.body, &page) == nil {
			for _, item := range page.Items {
				id, e := item.ActiveRevisionId.Get()
				if item.DefinitionId == revision.Definition && e == nil && id == revision.ID {
					return revision.ID
				}
			}
		}
		time.Sleep(time.Second)
	}
	p.t.Fatal("policy activation projection timeout")
	return uuid.Nil
}
func (p *liveProbe) grantMembers(admin *liveSession, members []*liveSession) {
	end := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	ids := make([]uuid.UUID, len(members))
	for i, m := range members {
		ids[i] = uuid.MustParse(m.member)
		r := p.call(admin, "POST", "/api/v1/admin/members/"+m.member+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY AT31 finite eligibility input", PlanRevisionId: &p.planRevision, ValidUntil: end}, nil)
		p.check(fmt.Sprintf("member %02d grant accepted", i), r, 202, "")
		time.Sleep(150 * time.Millisecond)
	}
	deadline := time.Now().Add(90 * time.Second)
	active := 0
	for time.Now().Before(deadline) {
		err := p.db.QueryRow(p.t.Context(), `SELECT count(*) FROM members m JOIN entitlements e ON e.member_id=m.id WHERE m.id=ANY($1::uuid[]) AND m.status='active' AND e.device_limit=3 AND e.valid_from<=now() AND e.effective_service_end>now() AND e.effective_service_end<=$2 AND NOT e.admin_suspended`, ids, end).Scan(&active)
		if err != nil {
			p.t.Fatal("read grant projections")
		}
		if active == len(members) {
			break
		}
		time.Sleep(2 * time.Second)
	}
	p.metadata["grant_projection"] = map[string]any{"commands": len(members), "active_members": active, "finite_end": end, "writes": "admin grants only; projected tables read-only"}
	if active != len(members) {
		p.t.Fatal("finite grant projections did not converge within 90 seconds")
	}
}

// The shared lab currently has one native member daemon/trigger. Admit that
// peer through the embedded Dex device flow. Sixty concurrent clients require
// a shared-lab scaling hook; one admitted peer must never be relabeled as 60.
func (p *liveProbe) admitMember(members []*liveSession) [][]uuid.UUID {
	devices := make([][]uuid.UUID, len(members))
	p.bindings = map[uuid.UUID]api.DeviceBinding{}
	p.metadata["requested_devices"] = 60
	p.metadata["authorized_devices"] = 0
	p.metadata["device_fixture"] = "native NetBird user device flow; production devices.discover writes admission/binding"
	trigger := filepath.Join(p.state, "member/var/e2e/user-auth")
	if os.WriteFile(trigger, []byte("1\n"), 0600) != nil {
		p.t.Fatal("native member trigger")
	}
	pattern := regexp.MustCompile(`https://netbird\.example\.invalid/[^\s\x1b"<>]+`)
	var flow string
	deadline := time.Now().Add(75 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := p.docker("member", "cat", "/var/lib/vpn-node/e2e/netbird-up.log")
		if err == nil {
			for _, candidate := range pattern.FindAllString(string(raw), -1) {
				u, e := url.Parse(candidate)
				if e == nil && u.Path == "/oauth2/device" && u.Query().Get("user_code") != "" {
					flow = candidate
					break
				}
			}
		}
		if flow != "" {
			break
		}
		time.Sleep(time.Second)
	}
	if flow == "" {
		p.metadata["admission_blocker"] = "Existing member user-auth hook produced no Dex device flow within 75 seconds"
		return devices
	}
	client := p.client()
	client.CheckRedirect = idplogin.RedirectLimit(20)
	status, err := idplogin.Walk(p.t.Context(), client, flow, members[0].cred.Email, members[0].cred.Password, idplogin.DeviceDone)
	p.metadata["native_device_login"] = map[string]any{"initial_http_status": status, "completed": err == nil, "flow_path": "/oauth2/device"}
	if err != nil {
		p.metadata["admission_blocker"] = "Native Dex device login did not complete; shared idplogin walker"
		return devices
	}
	deadline = time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		r := p.call(members[0], "GET", "/api/v1/member/devices", nil, nil)
		var list api.MemberDeviceList
		if r.status == 200 && json.Unmarshal(r.body, &list) == nil {
			for _, d := range list.Items {
				if d.Status == api.DeviceStatusAuthorized {
					devices[0] = []uuid.UUID{d.DeviceId}
					break
				}
			}
		}
		if len(devices[0]) > 0 {
			break
		}
		time.Sleep(2 * time.Second)
	}
	p.metadata["authorized_devices"] = len(devices[0])
	if len(devices[0]) == 0 {
		p.metadata["admission_blocker"] = "Native login completed but devices.discover projected no authorized member device within 120 seconds"
		p.admissionCounts(members[0])
		return devices
	}
	// Unpublished node B intentionally has no serving bindings in desired state.
	// Read the product projection for the synthetic ledger subset; do not write
	// it or infer a serving policy from it (D-050, §19 AT31).
	deadline = time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		var binding api.DeviceBinding
		var revision string
		err := p.db.QueryRow(p.t.Context(), `SELECT member_id,device_id,binding_revision::text,valid_from,valid_until FROM device_bindings WHERE member_id=$1 AND device_id=$2 AND valid_from<=now() AND valid_until>now() ORDER BY binding_revision DESC LIMIT 1`, members[0].member, devices[0][0]).Scan(&binding.MemberId, &binding.DeviceId, &revision, &binding.ValidFrom, &binding.ValidUntil)
		if err == nil {
			n, parseErr := strconv.ParseUint(revision, 10, 64)
			if parseErr != nil {
				p.t.Fatal("projected binding revision")
			}
			binding.BindingRevision = decstr.U64(n)
			p.bindings[binding.DeviceId] = binding
			break
		}
		time.Sleep(time.Second)
	}
	p.metadata["projected_bindings"] = len(p.bindings)
	p.metadata["ledger_binding_source"] = "read-only device_bindings produced by devices.discover; not a serving-policy assertion"
	p.admissionCounts(members[0])
	if len(p.bindings) == 0 {
		p.metadata["binding_blocker"] = "Authorized device has no current product-projected binding after 45 seconds"
	}
	p.metadata["device_scale_blocker"] = "deploy/devstack/entrypoint-member.sh exposes one fixed daemon/interface and one user-auth trigger. Need a documented 20-user/3-native-peer-per-user fixture hook before claiming 60 devices."
	// Observe the grant-derived entitlement after at least two 15s device sweeps.
	time.Sleep(31 * time.Second)
	var active int
	if err := p.db.QueryRow(p.t.Context(), `SELECT count(*) FROM members m JOIN entitlements e ON e.member_id=m.id WHERE m.id=$1 AND m.status='active' AND e.effective_service_end>now()`, members[0].member).Scan(&active); err != nil || active != 1 {
		p.t.Error("grant projection did not survive reconciliation")
	}
	p.metadata["grant_survived_two_device_sweeps"] = active == 1
	return devices
}
func (p *liveProbe) admissionCounts(m *liveSession) {
	var users, devices, bindings int
	var same bool
	err := p.db.QueryRow(p.t.Context(), `SELECT (SELECT count(*) FROM netbird_user_bindings WHERE member_id=$1),(SELECT count(*) FROM member_devices WHERE member_id=$1),(SELECT count(*) FROM device_bindings WHERE member_id=$1),EXISTS(SELECT 1 FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id WHERE n.member_id=$1 AND n.external_user_id=i.subject)`, m.member).Scan(&users, &devices, &bindings, &same)
	if err != nil {
		p.t.Error("read native projection evidence")
		return
	}
	p.metadata["native_projection"] = map[string]any{"user_bindings": users, "devices": devices, "bindings": bindings, "same_public_subject": same}
}

func (p *liveProbe) concurrentLogins(members []*liveSession) {
	if os.Getenv("FVPN_SECOPS_LOGIN_BURST") != "1" {
		raw, err := os.ReadFile(filepath.Join(p.root, "docs/e2e/at/secops/followup1-login.json"))
		var historical map[string]any
		if err != nil || json.Unmarshal(raw, &historical) != nil {
			p.t.Fatal("historical login evidence")
		}
		p.metadata["concurrent_sign_in"] = historical
		p.metadata["login_41st_request"] = map[string]any{"status": "blocked on control follow-up 10 / D-060", "expected_status": 429, "actual_status": nil, "retry_after": nil, "remeasured": false}
		return
	}
	p.loginDefaults()
	p.loginRefill(41)
	metricsBefore := p.loginMetrics()
	type outcome struct {
		client        *http.Client
		authenticated bool
		status        int
		lastHTTP      int
		wrong         bool
		millis        float64
		requests      []loginHTTPRecord
	}
	outcomes := make([]outcome, len(members))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, m := range members {
		wg.Go(func() {
			c := p.client()
			trace := &loginStatusTransport{base: c.Transport}
			c.Transport = trace
			c.CheckRedirect = idplogin.RedirectLimit(20)
			<-start
			now := time.Now()
			err := idplogin.PortalLogin(p.t.Context(), c, livePublic, m.cred.Email, m.cred.Password)
			o := outcome{client: c, lastHTTP: trace.last, millis: float64(time.Since(now).Microseconds()) / 1000}
			if err == nil {
				r := p.call(&liveSession{client: c}, "GET", "/auth/session?role=member", nil, nil)
				o.status = r.status
				var v struct {
					Authenticated bool   `json:"authenticated"`
					MemberID      string `json:"member_id"`
				}
				if json.Unmarshal(r.body, &v) == nil {
					o.authenticated = v.Authenticated
					o.wrong = v.Authenticated && v.MemberID != m.member
				}
			}
			o.requests = trace.responses
			outcomes[i] = o
		})
	}
	close(start)
	wg.Wait()
	success, wrong := 0, 0
	codes := map[int]int{}
	lastCodes := map[int]int{}
	latencies := []float64{}
	requests := [][]loginHTTPRecord{}
	req, _ := http.NewRequest("GET", livePublic, nil)
	for _, o := range outcomes {
		if o.authenticated {
			success++
		}
		if o.wrong {
			wrong++
		}
		codes[o.status]++
		lastCodes[o.lastHTTP]++
		latencies = append(latencies, o.millis)
		requests = append(requests, o.requests)
		for _, c := range o.client.Jar.Cookies(req.URL) {
			p.secrets = append(p.secrets, []byte(c.Value))
		}
	}
	p.metadata["concurrent_sign_in"] = map[string]any{"attempts": len(members), "authenticated": success, "wrong_member": wrong, "session_http_status": codes, "last_login_http_status": lastCodes, "login_wall_ms": latencies, "requests_by_identity": requests, "metrics_before": metricsBefore, "metrics_after": p.loginMetrics(), "barrier_started": true, "issuer": "embedded Dex, separate cookie jars; no shared assume slot", "zero_status": "login walker did not complete; not a measured session HTTP response", "supersedes": "followup1-login.json; D-060 / PR #106"}
	if wrong > 0 || success != len(members) {
		p.t.Error("D-060 concurrent Dex login did not complete every distinct member identity")
	}
	time.Sleep(8 * time.Second)
}

func (p *liveProbe) serving(admin *liveSession, node uuid.UUID) {
	var view api.NodeDetail
	var detail liveReply
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		detail = p.call(admin, "GET", "/api/v1/admin/nodes/"+node.String(), nil, nil)
		if detail.status == 200 && json.Unmarshal(detail.body, &view) == nil && view.ProvisioningState == api.ProvisioningStateReady {
			break
		}
		time.Sleep(2 * time.Second)
	}
	p.check("enrolled node detail", detail, 200, "")
	p.metadata["agent_node_incarnation"] = view.CurrentIncarnationId
	p.metadata["node_service_precondition"] = map[string]any{"lifecycle": view.LifecycleState, "provisioning": view.ProvisioningState, "data_health": view.DataHealth, "safety": view.SafetyStatus, "publication": view.PublicationIntent, "observed_publication": view.PublicationObserved, "config_generation": view.ConfigGeneration, "management_peer_bound": view.NetbirdPeerId.IsSpecified() && !view.NetbirdPeerId.IsNull()}
	p.metadata["publication_observation_before"] = p.publicationObservation(node)
	r := p.call(admin, "PUT", "/api/v1/admin/nodes/"+node.String()+"/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, GroupIds: &[]uuid.UUID{p.groupID}, Reason: "TEST ONLY serving precondition"}, map[string]string{"If-Match": detail.header.Get("ETag")})
	p.metadata["publication_request"] = map[string]any{"status": r.status, "code": r.code}
	if r.status == 202 {
		var op api.Operation
		_ = json.Unmarshal(r.body, &op)
		deadline = time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			r = p.call(admin, "GET", "/api/v1/admin/operations/"+op.OperationId.String(), nil, nil)
			if r.status == 200 {
				_ = json.Unmarshal(r.body, &op)
				if op.Status == api.OperationStatusSucceeded || op.Status == api.OperationStatusFailed {
					break
				}
			}
			time.Sleep(2 * time.Second)
		}
		steps := []map[string]any{}
		negativePassed := false
		for _, step := range op.Steps {
			steps = append(steps, map[string]any{"name": step.Name, "status": step.Status, "error_code": step.ErrorCode, "message": step.Message, "required": step.Required, "observed_at": step.ObservedAt})
			message, _ := step.Message.Get()
			code, _ := step.ErrorCode.Get()
			negativePassed = negativePassed || (step.Name == "data_path_probe" && step.Status == api.OperationStepStatusFailed && message == "member_forwarding_unconfirmed" && code == api.ErrorCodeNODENOTSERVICEABLE)
		}
		p.metadata["publication_operation"] = map[string]any{"operation_id": op.OperationId, "status": op.Status, "steps": steps}
		p.metadata["publication_negative"] = map[string]any{"expected": "failed data_path_probe: NODE_NOT_SERVICEABLE / member_forwarding_unconfirmed", "passed": negativePassed && op.Status == api.OperationStatusFailed, "owner_fix": "wire-nodes follow-up 12 / PR #109"}
		if os.Getenv("FVPN_SECOPS_FOLLOWUP3") == "1" && !negativePassed {
			p.t.Error("SECOPS-11 unarmed lab publication did not fail with member_forwarding_unconfirmed")
		}
	}
	// Read back after publication. A completed operation still does not prove
	// this native member's packets traverse forward_admission_v1.
	detail = p.call(admin, "GET", "/api/v1/admin/nodes/"+node.String(), nil, nil)
	if detail.status == 200 {
		_ = json.Unmarshal(detail.body, &view)
	}
	p.metadata["node_after_publication"] = map[string]any{"publication": view.PublicationIntent, "observed_publication": view.PublicationObserved, "config_generation": view.ConfigGeneration, "data_health": view.DataHealth, "provisioning": view.ProvisioningState}
	p.metadata["publication_observation_after"] = p.publicationObservation(node)
	raw, err := p.docker("node", "nft", "list", "chain", "inet", "fvpn", "forward")
	p.metadata["forward_chain_observed"] = err == nil
	p.metadata["forward_baseline_generation_zero"] = bytes.Contains(raw, []byte("fvpn_generation=0"))
	accepts := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "fvmb0") && strings.Contains(line, "accept") {
			accepts++
		}
	}
	p.metadata["member_accept_rules"] = accepts
	hb, err := p.docker("node", "cat", "/run/vpn-node/heartbeat")
	p.metadata["watchdog_heartbeat_present"] = err == nil && len(hb) > 0
	heartbeat := map[string]string{}
	for _, line := range strings.Split(string(hb), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			switch fields[0] {
			case "mono_ns", "wall_unix_ns", "policy_expires_mono_ns", "baseline_ok", "serving", "generation", "reason":
				heartbeat[fields[0]] = fields[1]
			}
		}
	}
	p.metadata["watchdog_state"] = heartbeat
	_, recoverErr := p.docker("node", "test", "-f", "/var/lib/vpn-node/e2e/updater.json")
	p.metadata["configured_updater_recovery_file_present"] = recoverErr == nil
	p.flowProbe(accepts, heartbeat["serving"] == "1")
	p.metadata["publication_positive"] = map[string]any{"status": "blocked", "owner": "w3-devstack-e2e follow-up 7", "precondition": "real agent forwarding observation passes PublicationForwardingEvidence; watchdog serving=1; admitted device counter advances", "counted_flow": p.metadata["counted_flow"], "reason": "No serving fixture or forged agent observation is permitted (D-050)."}
	p.metadata["serving_probe"] = "API state is insufficient: require admitted device, timed member allowset, selected native exit, and increasing node forward-admission counter for the controlled flow."
	checks := []map[string]any{}
	for _, check := range view.Checks {
		checks = append(checks, map[string]any{"name": check.Name, "status": check.Status, "error_code": check.ErrorCode})
	}
	p.metadata["serving_checks"] = checks
	if heartbeat["reason"] == "recovery_required" && recoverErr != nil {
		p.metadata["serving_blocker"] = "Fresh node boot invokes updater recover with missing /var/lib/vpn-node/e2e/updater.json; watchdog reports recovery_required, serving=0, policy_expires_mono_ns=0. writeAgentConfigs must provide a persistent boot recovery config separate from the updater run trigger."
	}
}

func (p *liveProbe) flowProbe(accepts int, serving bool) {
	// A bounded request to the stack origin may bypass the VPN. Only increased
	// per-device admission counters plus serving gates can count as a positive.
	counters := func() (map[string]uint64, bool) {
		raw, err := p.docker("node", "nft", "-j", "list", "table", "inet", "fvpn")
		var doc struct {
			NFTables []struct {
				Counter *struct {
					Name  string
					Bytes uint64
				} `json:"counter"`
			} `json:"nftables"`
		}
		if err != nil || json.Unmarshal(raw, &doc) != nil {
			return nil, false
		}
		result := map[string]uint64{}
		for _, obj := range doc.NFTables {
			if obj.Counter != nil {
				n := obj.Counter.Name
				if strings.HasPrefix(n, "u_") || strings.HasPrefix(n, "d_") {
					result[n] = obj.Counter.Bytes
				}
			}
		}
		return result, true
	}
	before, okBefore := counters()
	routes, routesErr := p.docker("member", "netbird", "networks", "list")
	// Exit selection is meaningful only after the node is serving. The current
	// lab's missing recovery config keeps it closed, so no route is forced here.
	_, flowErr := p.docker("member", "wget", "--timeout=3", "--tries=1", "-q", "-O", "/dev/null", "https://releases.example.invalid/healthz")
	after, okAfter := counters()
	var delta uint64
	for name, n := range after {
		if old, exists := before[name]; exists && n >= old {
			delta += n - old
		}
	}
	counted := serving && accepts > 0 && len(p.bindings) > 0 && okBefore && okAfter && delta > 0 && flowErr == nil
	p.metadata["counted_flow"] = counted
	p.metadata["flow_probe"] = map[string]any{"command": []string{"docker", "exec", livePrefix + "member", "wget", "--timeout=3", "--tries=1", "-q", "-O", "/dev/null", "https://releases.example.invalid/healthz"}, "request_succeeded": flowErr == nil, "counter_reads_ok": okBefore && okAfter, "device_counter_count": len(after), "admitted_bytes_delta": decstr.U64(delta), "node_serving": serving, "networks_read_ok": routesErr == nil, "no_native_networks": bytes.Contains(routes, []byte("No networks available")), "counted_flow": counted, "note": "Controlled-origin reachability alone can be a direct lab bridge path; it is not VPN throughput."}
	if !counted {
		p.metadata["serving_blocker"] = "No positive serving flow: inspect watchdog state, publication and admission counter delta; outage/deadline and mixed-load assertions cannot start."
	}
}
