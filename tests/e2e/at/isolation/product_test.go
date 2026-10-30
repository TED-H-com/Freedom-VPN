//go:build e2e

package isolation

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestProduct_AT10_AT13 drives AT10–AT13 on the assembled dev stack (§19).
// A missing serving node is recorded as blocked with the operation step that
// stopped it. Every record is written before a non-pass fails the gate.
func TestProduct_AT10_AT13(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("set FVPN_E2E=1 to run isolation acceptance against the dev stack")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	dir := evidenceDir()
	meta := collectMeta()
	st, err := openStack()
	if err != nil {
		blockProduct(t, dir, meta, "stack state is not readable: "+err.Error())
		return
	}
	if !st.healthy(ctx) {
		blockProduct(t, dir, meta, "control /readyz is not 200 on prefix "+st.prefix)
		return
	}
	obs := st.prove(ctx, t)
	probe := exitProbe{members: st.memberCount(), ipv6: st.ipv6Fact()}
	if obs.stop == "" {
		probe = st.exerciseServing(t, st.nodeIPs())
	}
	writeProduct(t, dir, meta, obs, probe)
}

func (s *stack) prove(ctx context.Context, t *testing.T) productObs {
	t.Helper()
	out := productObs{publication: "not_attempted", forward: ForwardView{Generation: -1}}
	out.mechanism = mechanismLine(evidenceDir())
	out.targets = s.targetInventory()
	admin, err := s.login(ctx, "e2e-admin", "admin")
	if err != nil {
		out.stop = "admin login failed"
		t.Errorf("admin login: %v", err)
		return out
	}
	if err := s.ensureReleaseSupport(ctx); err != nil {
		out.stop = "release_support insert failed"
		out.notes = append(out.notes, Observation{Name: "release_support", Detail: "insert failed"})
		return out
	}
	name, region := "at-isolation-exit", "lab"
	created, err := admin.call(ctx, "POST", s.public+"/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller, Name: &name, Region: &region,
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	if err != nil || created.Status != 201 {
		out.stop = "enrollment create " + httpDetail(created, err)
		return out
	}
	var body api.EnrollmentCreated
	if json.Unmarshal(created.Body, &body) != nil || !body.SecretAvailable || body.Token.IsNull() {
		out.stop = "enrollment token absent"
		return out
	}
	token, _ := body.Token.Get()
	out.notes = append(out.notes, Observation{Name: "token_length", Detail: itoa(len(token))})
	tokenPath := filepath.Join(s.state, "node", "var", "e2e", "enrollment.token")
	if writeAtomic(tokenPath, []byte(token+"\n")) != nil {
		out.stop = "enrollment token file"
		token = ""
		return out
	}
	token = ""
	out.nodeID = nodeUUID(body.Enrollment.NodeId)
	enrollCtx, enrollCancel := context.WithTimeout(ctx, 4*time.Minute)
	enrolled := waitUntil(enrollCtx, time.Second, func() bool {
		return s.stage(s.node, "/var/lib/vpn-node/e2e/status.json") == "enrolled"
	})
	enrollCancel()
	if !enrolled {
		out.stop = "node stage " + emptyStage(s.stage(s.node, "/var/lib/vpn-node/e2e/status.json"))
		return out
	}
	keyCtx, keyCancel := context.WithTimeout(ctx, 2*time.Minute)
	keyReady := waitUntil(keyCtx, time.Second, func() bool {
		_, err := s.docker(s.node, "test", "-s", "/var/lib/vpn-node/identity/setup.key")
		return err == nil
	})
	keyCancel()
	out.notes = append(out.notes, Observation{Name: "setup_key_persisted", Detail: boolText(keyReady)})
	nbCtx, nbCancel := context.WithTimeout(ctx, 3*time.Minute)
	nbUp := waitUntil(nbCtx, 2*time.Second, func() bool {
		return s.stage(s.node, "/var/lib/vpn-node/e2e/netbird.json") == "netbird_up"
	})
	nbCancel()
	out.notes = append(out.notes, Observation{Name: "netbird", Detail: emptyStage(s.stage(s.node, "/var/lib/vpn-node/e2e/netbird.json"))})
	if !nbUp {
		out.stop = "node netbird did not reach netbird_up"
	}
	dsCtx, dsCancel := context.WithTimeout(ctx, 90*time.Second)
	var desiredRaw []byte
	var verified bool
	ok := waitUntil(dsCtx, 2*time.Second, func() bool {
		desiredRaw, verified = s.desired()
		return verified && len(desiredRaw) > 0
	})
	dsCancel()
	if ok {
		view := parseDesired(desiredRaw)
		out.policyRevision = view.Generation
		if out.incarnation == "" {
			out.incarnation = view.Incarnation
		}
		out.bindings = view.Bindings
		out.resolvers = view.Resolvers
		out.notes = append(out.notes, Observation{Name: "desired_publication_intent", Detail: emptyStage(view.PublicationIntent)})
		out.notes = append(out.notes, Observation{Name: "authorization_generation", Detail: emptyStage(view.AuthorizationGen)})
		out.notes = append(out.notes, Observation{Name: "dns_generation", Detail: emptyStage(view.DNSGeneration)})
	} else {
		out.notes = append(out.notes, Observation{Name: "desired_state", Detail: "absent or rejected"})
		if out.stop == "" {
			out.stop = "signed desired state did not appear"
		}
	}
	detail, err := admin.call(ctx, "GET", s.public+"/api/v1/admin/nodes/"+out.nodeID, nil, nil)
	if err != nil || detail.Status != 200 {
		if out.stop == "" {
			out.stop = "GET node " + httpDetail(detail, err)
		}
		return out
	}
	var node api.NodeDetail
	if json.Unmarshal(detail.Body, &node) != nil {
		out.stop = "node detail did not parse"
		return out
	}
	out.safety = string(node.SafetyStatus)
	out.provisioning = string(node.ProvisioningState)
	out.control = string(node.ControlStatus)
	out.dataHealth = string(node.DataHealth)
	if id, err := node.CurrentIncarnationId.Get(); err == nil {
		out.incarnation = uuid.UUID(id).String()
	}
	out.notes = append(out.notes, Observation{Name: "lifecycle", Detail: string(node.LifecycleState)})
	out.notes = append(out.notes, Observation{Name: "publication_intent_before", Detail: string(node.PublicationIntent)})
	etag := detail.Header.Get("ETag")
	out.notes = append(out.notes, Observation{Name: "etag_present", Detail: boolText(etag != "")})
	if etag == "" {
		if out.stop == "" {
			out.stop = "GET node returned no ETag"
		}
		return out
	}
	key := uuid.NewString()
	first, err := publishOnce(ctx, admin, s.public, out.nodeID, etag, key)
	out.notes = append(out.notes, Observation{Name: "publication_put", Detail: httpDetail(first, err)})
	checks := failedChecks(first.Body)
	if checks != "" {
		out.notes = append(out.notes, Observation{Name: "failed_checks", Detail: checks})
		out.notes = append(out.notes, Observation{Name: "publication_error_body", Detail: string(first.Body)})
	}
	if err != nil {
		out.stop = "publication transport"
		return out
	}
	second, err := publishOnce(ctx, admin, s.public, out.nodeID, etag, key)
	out.idempotentStatus = httpDetail(second, err)
	if first.Status != 202 {
		out.stop = "PUT publication " + httpDetail(first, nil)
		if checks != "" {
			out.stop += " failed_checks=" + checks
		}
		out.publication = string(node.PublicationObserved)
		out.forward = s.forwardChain()
		return out
	}
	var op api.Operation
	if json.Unmarshal(first.Body, &op) != nil {
		out.stop = "publication 202 did not parse as an operation"
		return out
	}
	out.operationKind = string(op.Kind)
	opID := uuid.UUID(op.OperationId).String()
	if second.Status == 202 {
		var again api.Operation
		if json.Unmarshal(second.Body, &again) == nil && uuid.UUID(again.OperationId).String() == opID {
			out.notes = append(out.notes, Observation{Name: "idempotent_same_operation", Detail: "true"})
		}
	}
	pollCtx, pollCancel := context.WithTimeout(ctx, 3*time.Minute)
	var last api.Operation
	terminal := waitUntil(pollCtx, 2*time.Second, func() bool {
		resp, err := admin.call(ctx, "GET", s.public+"/api/v1/admin/operations/"+opID, nil, nil)
		if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &last) != nil {
			return false
		}
		switch last.Status {
		case api.OperationStatusSucceeded, api.OperationStatusFailed, api.OperationStatusPartiallySucceeded, api.OperationStatusSuperseded:
			return true
		default:
			return false
		}
	})
	pollCancel()
	if !terminal {
		last = op
		out.operationStatus = string(op.Status) + " (poll timed out)"
	} else {
		out.operationStatus = string(last.Status)
	}
	out.steps = formatSteps(last)
	if e, err := last.Error.Get(); err == nil {
		out.operationError = string(e.Code) + " " + clip(e.Message, 240)
	}
	again, err := admin.call(ctx, "GET", s.public+"/api/v1/admin/nodes/"+out.nodeID, nil, nil)
	if err == nil && again.Status == 200 {
		var after api.NodeDetail
		if json.Unmarshal(again.Body, &after) == nil {
			out.publication = string(after.PublicationObserved)
			out.safety = string(after.SafetyStatus)
			out.provisioning = string(after.ProvisioningState)
			out.control = string(after.ControlStatus)
			out.dataHealth = string(after.DataHealth)
			out.notes = append(out.notes, Observation{Name: "publication_intent_after", Detail: string(after.PublicationIntent)})
			if id, err := after.CurrentIncarnationId.Get(); err == nil {
				out.incarnation = uuid.UUID(id).String()
			}
		}
	}
	if raw, verified := s.desired(); verified {
		view := parseDesired(raw)
		out.policyRevision = view.Generation
		out.bindings = view.Bindings
		out.resolvers = view.Resolvers
		out.notes = append(out.notes, Observation{Name: "desired_publication_intent_after", Detail: emptyStage(view.PublicationIntent)})
	}
	out.forward = s.forwardChain()
	out.memberRoute = s.memberRoutes()
	if out.publication != string(api.PublicationObservedActive) {
		out.stop = "publication_observed=" + emptyStage(out.publication) + " operation=" + emptyStage(out.operationStatus) + " steps=" + emptyStage(out.steps)
		if out.operationError != "" {
			out.stop += " error=" + out.operationError
		}
		return out
	}
	if out.forward.Generation < 1 || out.forward.MemberAccept == 0 {
		out.stop = "publication_observed=active but forward generation=" + itoa(out.forward.Generation) + " member_accept=" + itoa(out.forward.MemberAccept)
		return out
	}
	if out.bindings == 0 {
		out.stop = "node is published and the forward generation is " + itoa(out.forward.Generation) + " but desired state device_bindings is empty"
		return out
	}
	if !strings.Contains(out.memberRoute, "fvnb0") {
		out.stop = "member routes do not traverse fvnb0, so exit-path probes would not prove isolation"
		return out
	}
	out.stop = ""
	return out
}

func (s *stack) targetInventory() string {
	names, err := exec.Command("docker", "ps", "--filter", "name=^"+s.prefix, "--format", "{{.Names}}").Output()
	list := strings.TrimSpace(string(names))
	if err != nil {
		list = "docker ps failed"
	}
	return "containers=" + oneLine(list) + " lab_prefix=" + s.labPrefix + " origin_ip=" + s.originIP + " node_dummy=192.0.2.20"
}

func (s *stack) memberRoutes() string {
	out, err := s.docker(s.member, "ip", "route")
	if err != nil {
		return "route unavailable"
	}
	var kept []string
	for _, line := range strings.Split(string(out), "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		kept = append(kept, trim)
		if len(kept) == 6 {
			break
		}
	}
	return strings.Join(kept, " | ")
}

func formatSteps(op api.Operation) string {
	if len(op.Steps) == 0 {
		return ""
	}
	parts := make([]string, 0, len(op.Steps))
	for _, step := range op.Steps {
		item := step.Name + "=" + string(step.Status)
		if code, err := step.ErrorCode.Get(); err == nil && code != "" {
			item += ":" + string(code)
		}
		if msg, err := step.Message.Get(); err == nil && msg != "" {
			item += "(" + clip(msg, 120) + ")"
		}
		parts = append(parts, item)
	}
	return strings.Join(parts, "; ")
}

func mechanismLine(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "lab-AT10.json"))
	if err != nil {
		return "mechanism evidence absent"
	}
	var rec Record
	if json.Unmarshal(raw, &rec) != nil {
		return "mechanism evidence unreadable"
	}
	return "lab-AT10 " + rec.Status + "; lab-AT11 " + statusFile(dir, "lab-AT11.json") + "; lab-AT12 " + statusFile(dir, "lab-AT12.json") + "; lab-AT13 " + statusFile(dir, "lab-AT13.json")
}

func statusFile(dir, name string) string {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "absent"
	}
	var rec Record
	if json.Unmarshal(raw, &rec) != nil {
		return "unreadable"
	}
	return rec.Status
}

func evidenceDir() string {
	if dir := os.Getenv("FVPN_AT_EVIDENCE_DIR"); dir != "" {
		if !filepath.IsAbs(dir) {
			if root, err := repoRoot(); err == nil {
				return filepath.Join(root, dir)
			}
		}
		return dir
	}
	root, err := repoRoot()
	if err != nil {
		return "docs/e2e/at/isolation"
	}
	return filepath.Join(root, "docs", "e2e", "at", "isolation")
}

func collectMeta() Record {
	rec := Record{
		Build: gitSHA(),
		Commands: []string{
			"FVPN_E2E_PREFIX=w4-isolation FVPN_E2E_PORT_BASE=36000 make -f mk/e2e.mk e2e-up",
			"FVPN_E2E=1 FVPN_E2E_PREFIX=w4-isolation FVPN_E2E_PORT_BASE=36000 go test -tags e2e -count=1 -timeout 25m -p 1 -parallel 1 ./tests/e2e/at/isolation",
			"make -f mk/at-isolation.mk at-isolation",
		},
	}
	for _, name := range []string{
		"fvpn-w4-isolation-node:local",
		"fvpn-w4-isolation-control:local",
		"fvpn-w4-isolation-worker:local",
		"fvpn-w4-isolation-lab:lab",
	} {
		rec.ImageDigests = append(rec.ImageDigests, name+" "+imageID(name))
	}
	return rec
}

func gitSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func imageID(name string) string {
	out, err := exec.Command("docker", "image", "inspect", "-f", "{{.Id}}", name).Output()
	if err != nil {
		return "absent"
	}
	return strings.TrimSpace(string(out))
}

func blockProduct(t *testing.T, dir string, meta Record, reason string) {
	t.Helper()
	obs := productObs{stop: reason, forward: ForwardView{Generation: -1}, mechanism: mechanismLine(dir)}
	writeProduct(t, dir, meta, obs, exitProbe{})
}

func writeProduct(t *testing.T, dir string, meta Record, obs productObs, probe exitProbe) {
	t.Helper()
	at10, at11, at12, at13 := decideProduct(obs, probe)
	cases := []struct {
		id, title, expected string
		spec                []string
		result              caseResult
	}{
		{"AT10", "Host and upstream isolation", "RFC1918, cloud-private, public-prefix internal, and the node management addresses are refused through the exit. The controlled Internet endpoint stays reachable. An interface change does not open an allow-all window (§19 AT10, ISO01, ISO02).", []string{"§19 AT10", "ISO01", "ISO02"}, at10},
		{"AT11", "IPv6, metadata and special addresses", "The selected IPv6 mode either routes or blocks. ULA, link-local, loopback, metadata and IPv4-mapped addresses are unreachable. There is no unverified bypass (§19 AT11, ISO01, ISO03).", []string{"§19 AT11", "ISO01", "ISO03"}, at11},
		{"AT12", "Cross-member, management plane and DNS exceptions", "Peer, SSH, agent API, DNS admin, arbitrary overlay and a spoofed source are refused. TCP and UDP 53 to the designated resolvers succeed and nothing else on those hosts does (§19 AT12, ISO01, ISO04, DNS03).", []string{"§19 AT12", "ISO01", "ISO04", "DNS03"}, at12},
		{"AT13", "Revocation under restart, update and live connections", "Long TCP and UDP flows die on revocation. Apply, agent SIGKILL, NetBird restart, node reboot and an updater switch never open an allow-all window. established/related does not bypass revocation. Forwarding stays off until a valid policy is held (§19 AT13, ISO02, ENT02, SEC01).", []string{"§19 AT13", "ISO02", "ENT02", "SEC01"}, at13},
	}
	for _, c := range cases {
		rec := meta
		rec.ID = c.id
		rec.Title = c.title
		rec.Spec = c.spec
		rec.Status = c.result.status
		rec.Expected = c.expected
		rec.Actual = c.result.actual
		rec.PolicyRevision = emptyStage(obs.policyRevision)
		rec.NodeIncarnation = emptyStage(obs.incarnation)
		rec.Observations = []Observation{
			{Name: "node_id", Detail: emptyStage(obs.nodeID)},
			{Name: "publication_observed", Detail: emptyStage(obs.publication)},
			{Name: "operation_kind", Detail: emptyStage(obs.operationKind)},
			{Name: "operation_status", Detail: emptyStage(obs.operationStatus)},
			{Name: "operation_steps", Detail: emptyStage(obs.steps)},
			{Name: "operation_error", Detail: emptyStage(obs.operationError)},
			{Name: "idempotent_put", Detail: emptyStage(obs.idempotentStatus)},
			{Name: "device_bindings", Detail: itoa(obs.bindings)},
			{Name: "resolver_ipv4", Detail: itoa(obs.resolvers)},
			{Name: "safety_status", Detail: emptyStage(obs.safety)},
			{Name: "provisioning_state", Detail: emptyStage(obs.provisioning)},
			{Name: "control_status", Detail: emptyStage(obs.control)},
			{Name: "data_health", Detail: emptyStage(obs.dataHealth)},
			{Name: "forward_policy", Detail: emptyStage(obs.forward.Policy)},
			{Name: "forward_generation", Detail: itoa(obs.forward.Generation)},
			{Name: "forward_baseline", Detail: boolText(obs.forward.Baseline)},
			{Name: "member_accept_rules", Detail: itoa(obs.forward.MemberAccept)},
			{Name: "forward_excerpt", Detail: emptyStage(obs.forward.Excerpt)},
			{Name: "member_routes", Detail: emptyStage(obs.memberRoute)},
			{Name: "stack_targets", Detail: emptyStage(obs.targets)},
			{Name: "exit_probes", Detail: emptyStage(strings.Join(probe.lines, "; "))},
			{Name: "ipv6", Detail: emptyStage(probe.ipv6)},
			{Name: "members", Detail: emptyStage(probe.members)},
			{Name: "agent_sigkill", Detail: emptyStage(probe.agentKill)},
			{Name: "netbird_restart", Detail: emptyStage(probe.netbird)},
			{Name: "node_reboot", Detail: emptyStage(probe.reboot)},
			{Name: "updater_switch", Detail: emptyStage(probe.updater)},
			{Name: "mechanism_lab", Detail: emptyStage(obs.mechanism)},
			{Name: "operator_platforms", Detail: "needs operator run; see operator-run.md"},
		}
		rec.Observations = append(rec.Observations, obs.notes...)
		if err := Write(dir, rec); err != nil {
			t.Errorf("evidence %s: %v", c.id, err)
		}
	}
	for _, c := range cases {
		if c.result.status != StatusPass {
			t.Errorf("%s: %s; see %s.json", c.id, c.result.status, c.id)
		}
	}
}
