//go:build integration

package nodes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Only the external identity and NetBird read-back are TEST ONLY inputs. All
// registry, account, plan, binding, operation and outbox rows use real commands.
type d165Peer struct{ peer devices.Peer }

func (p *d165Peer) ObservePeer(context.Context, string, string) (devices.Peer, error) {
	return p.peer, nil
}
func (p *d165Peer) SetAuthorization(_ context.Context, _, _ string, allow bool, groups []uuid.UUID) error {
	p.peer.Authorized = allow
	p.peer.Groups = append([]uuid.UUID(nil), groups...)
	return nil
}

type d165World struct {
	*r14World
	services                        *membersruntime.Services
	reports                         []api.ObservationReport
	member, grant, device, addition uuid.UUID
}

func d165Command(role string, id uuid.UUID) entitlement.Command {
	return entitlement.Command{Actor: entitlement.Actor{Role: role, ID: id, RequestID: uuid.New()}, Key: uuid.New()}
}
func newD165World(t *testing.T, exits int, acknowledge ...bool) *d165World {
	w := &d165World{r14World: newR14World(t)}
	w.r.Nodes = nodes.NewService(w.r.Pool, nodes.WithClock(func() time.Time { return w.now }))
	group := w.group("d165", "TEST ONLY D165")
	for i := 0; i < exits; i++ {
		r := w.primary
		if i > 0 {
			r = w.enroll()
			r, _ = w.prepare(r, false, false)
			w.observe(r)
		}
		w.publish(r.NodeId, group)
		w.reports = append(w.reports, r)
	}
	ctx := t.Context()
	principal, err := (auth.PostgresStore{Pool: w.r.Pool}).Login(ctx, auth.Identity{Issuer: "https://id.example.invalid", ClientScope: "portal", Subject: "TEST-ONLY-d165", Role: auth.Member, Label: "TEST ONLY", AuthTime: time.Now().UTC().Add(-time.Minute), Expires: w.now.Add(time.Hour)}, uuid.NewString(), "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	w.member = principal.ID
	store, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	store.Now = func() time.Time { return time.Now().UTC() }
	peer := &d165Peer{peer: devices.Peer{DeploymentID: w.r.Config.NetBirdDeployment, ID: "TEST-ONLY-peer", ExternalUserID: "TEST-ONLY-d165", OverlayScope: "TEST ONLY", OverlayIP: netip.MustParseAddr("100.64.0.72"), Platform: "linux"}}
	w.services = &membersruntime.Services{HTTP: httpadapter.Services{Members: &members.Service{Pool: w.r.Pool, Entitlements: store}, Devices: &devices.Service{Pool: w.r.Pool, Entitlements: store, Adapter: peer}, Billing: &billing.Service{}}, Compiler: w.compiler}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/users" {
			fmt.Fprint(rw, `[{"id":"TEST-ONLY-d165","is_service_user":false,"is_blocked":false}]`)
		} else {
			fmt.Fprint(rw, `[]`)
		}
	}))
	t.Cleanup(server.Close)
	client, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: w.r.Config.NetBirdDeployment, Token: netbird.TokenFile(w.r.Config.NetBirdTokenFile), AllowInsecureHTTP: true, Bindings: devices.NetBirdBindings{Pool: w.r.Pool}})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := devices.NewIdentityDiscovery("https://id.example.invalid", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &devices.NetBirdAdapter{Pool: w.r.Pool, Client: client, DeploymentID: w.r.Config.NetBirdDeployment, OverlayScope: "TEST ONLY", Identity: identity}
	if err = adapter.Discover(ctx, w.services.HTTP.Devices); err != nil {
		t.Fatal(err)
	}
	policyStore := policies.NewStore(w.r.Pool)
	create := func(kind api.PolicyType, payload map[string]any) uuid.UUID {
		t.Helper()
		raw, e := json.Marshal(payload)
		if e != nil {
			t.Fatal(e)
		}
		code := "d165-" + string(kind)
		rev, e := policyStore.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: kind, SchemaVersion: 1, Reason: "TEST ONLY D165", Payload: raw, ActorID: w.admin})
		if e != nil {
			t.Fatal(e)
		}
		return rev.RevisionID
	}
	example := func(name string) map[string]any {
		t.Helper()
		raw, e := os.ReadFile("../../contracts/examples/valid/PolicyRevisionCreate." + name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		var doc struct {
			Payload map[string]any `json:"payload"`
		}
		if e = json.Unmarshal(raw, &doc); e != nil {
			t.Fatal(e)
		}
		return doc.Payload
	}
	fair := example("fair-use")
	fair["overrides"] = []any{}
	fup := create(api.PolicyTypeFairUse, fair)
	dns := example("dns-profile")
	dns["domain_rule_set_revision_ids"] = []any{}
	dns["threat_feed_revision_ids"] = []any{}
	dnsID := create(api.PolicyTypeDnsProfile, dns)
	plan := example("plan")
	plan["exit_group_ids"] = []uuid.UUID{group}
	plan["fair_use_revision_id"] = fup
	plan["dns_profile_revision_id"] = dnsID
	plan["rate_limit"] = map[string]int{"down_bps": 100000000, "up_bps": 50000000}
	revision := create(api.PolicyTypePlan, plan)
	start := w.now.Add(-time.Hour)
	grant, err := w.services.HTTP.Members.Grant(ctx, d165Command("admin", w.admin), w.member, api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: w.now.Add(12 * time.Hour), PlanRevisionId: &revision, Reason: "TEST ONLY D165"})
	if err != nil {
		t.Fatal(err)
	}
	w.grant = grant.ResourceID
	w.dispatch("member.entitlement_changed")
	device, err := w.services.HTTP.Devices.AuthorizeDevice(ctx, d165Command("system", uuid.New()), peer.peer)
	if err != nil {
		t.Fatal(err)
	}
	w.device = device.DeviceID
	w.addition = *device.OperationID
	w.dispatch("device.authorized")
	if len(acknowledge) == 0 || acknowledge[0] {
		for i, r := range w.reports {
			w.reports[i] = w.report(r, "clean")
			w.observe(w.reports[i])
		}
	}
	return w
}
func (w *d165World) dispatch(kind string) {
	w.t.Helper()
	events, err := db.ClaimOutbox(w.t.Context(), w.r.Pool, "TEST ONLY D165", 1000, time.Hour)
	if err != nil {
		w.t.Fatal(err)
	}
	for _, event := range events {
		if event.EventType == kind {
			if _, err = worker.MemberJobs()[kind](worker.WithMembers(w.t.Context(), w.services), event); err != nil {
				w.t.Fatal(err)
			}
			return
		}
	}
	w.t.Fatalf("no unclaimed %s event", kind)
}
func (w *d165World) revokeDevice() uuid.UUID {
	w.t.Helper()
	out, err := w.services.HTTP.Devices.RevokeDevice(w.t.Context(), d165Command("member", w.member), w.device)
	if err != nil {
		w.t.Fatal(err)
	}
	return *out.OperationID
}
func (w *d165World) report(base api.ObservationReport, mode string) api.ObservationReport {
	w.t.Helper()
	r := cloneReport(base, base.NodeId, base.NodeIncarnationId)
	r.ObservedAt = w.bump()
	r.ProvisioningState = api.ProvisioningStateReady
	r.SafetyStatus = api.SafetyStatusPassed
	r.DataHealth = api.DataHealthHealthy
	var gen decstr.U64
	if err := w.r.Pool.QueryRow(w.t.Context(), `SELECT desired_config_generation::text FROM nodes WHERE id=$1`, r.NodeId).Scan(&gen); err != nil {
		w.t.Fatal(err)
	}
	r.AppliedConfigGeneration.Set(gen)
	for i := range r.Checks {
		r.Checks[i].ObservedAt = r.ObservedAt
	}
	if mode != "clean" {
		r.Errors = []api.NodeError{{Code: api.ErrorCodeISOLATIONPOLICYINVALID, Stage: "apply", Message: "TEST ONLY conntrack pending", ObservedAt: r.ObservedAt}}
	}
	if mode == "closed" {
		r.SafetyStatus = api.SafetyStatusFailed
		r.ProvisioningState = api.ProvisioningStateFailed
		r.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed}
		for i := range r.Checks {
			if r.Checks[i].Name == api.NodeCheckNameInternetOnly {
				r.Checks[i].Status = api.HealthCheckStatusFailed
				r.Checks[i].ErrorCode.Set(api.ErrorCodeISOLATIONPOLICYINVALID)
			}
		}
	}
	return r
}
func (w *d165World) wantStep(op uuid.UUID, succeeded bool) {
	w.t.Helper()
	st, ok := nodesStep(w.operation(op))
	if !ok || (st.Status == api.OperationStepStatusSucceeded) != succeeded {
		w.t.Fatalf("D165 nodes succeeded=%v, want %v (%s)", st.Status == api.OperationStepStatusSucceeded, succeeded, stepText(st))
	}
}
func (w *d165World) target(op, node uuid.UUID, outcome string) {
	w.t.Helper()
	var got, reason string
	err := w.r.Pool.QueryRow(w.t.Context(), `SELECT outcome,coalesce(reason,'') FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, op, node).Scan(&got, &reason)
	if err != nil {
		w.t.Fatalf("D165 durable target: %v", err)
	}
	if got != outcome || (outcome == "unconfirmed" && reason == "") {
		w.t.Fatalf("D165 target outcome=%s reason=%s, want %s with reason", got, reason, outcome)
	}
}
func TestD165_T1_EveryFreshExitAcknowledges(t *testing.T) {
	w := newD165World(t, 2)
	op := w.revokeDevice()
	w.dispatch("device.revoked")
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(op, false)
	w.observe(w.report(w.reports[1], "clean"))
	w.wantStep(op, true)
}
func TestD165_T2_PrecompileReportCannotComplete(t *testing.T) {
	w := newD165World(t, 1)
	op := w.revokeDevice()
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(op, false)
	w.dispatch("device.revoked")
	w.wantStep(op, false)
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(op, true)
}
func TestD165_T3_EmptyProjectionWaitsForFormerCarrier(t *testing.T) {
	w := newD165World(t, 1)
	unrelated := w.enroll()
	unrelated, _ = w.prepare(unrelated, false, false)
	w.observe(unrelated)
	out, err := w.services.HTTP.Members.RevokeMemberGrant(t.Context(), d165Command("admin", w.admin), w.member, w.grant, "TEST ONLY remove sole grant")
	if err != nil {
		t.Fatal(err)
	}
	w.dispatch("member.entitlement_changed")
	w.observe(w.report(unrelated, "clean"))
	w.wantStep(*out.OperationID, false)
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(*out.OperationID, true)
}
func TestD165_T4_SweepRecordsUnconfirmedAndLateAck(t *testing.T) {
	w := newD165World(t, 2)
	op := w.revokeDevice()
	w.dispatch("device.revoked")
	w.observe(w.report(w.reports[0], "clean"))
	w.now = w.now.Add(nodes.DefaultStaleTimeout + time.Second)
	if _, err := w.r.Nodes.SweepStaleNodes(t.Context(), 0, 0); err != nil {
		t.Fatal(err)
	}
	w.wantStep(op, true)
	w.target(op, w.reports[1].NodeId, "unconfirmed")
	w.observe(w.report(w.reports[1], "clean"))
	w.target(op, w.reports[1].NodeId, "acknowledged")
	w.wantStep(op, true)
}
func TestD165_T5_ZeroTargetsCompleteAtCapture(t *testing.T) {
	w := newD165World(t, 0)
	op := w.revokeDevice()
	w.dispatch("device.revoked")
	w.wantStep(op, true)
	var evidence bool
	err := w.r.Pool.QueryRow(t.Context(), `SELECT o.observed_refs->'nodes'->>'reference'='node_operation_capture:'||c.id::text AND c.captured_at IS NOT NULL FROM operations o JOIN node_operation_captures c ON c.operation_id=o.id WHERE o.id=$1`, op).Scan(&evidence)
	if err != nil || !evidence {
		t.Fatal("D165 empty capture evidence", evidence, err)
	}
}
func TestD165_T6_AdditionRequiresCleanAck(t *testing.T) {
	w := newD165World(t, 1, false)
	w.observe(w.report(w.reports[0], "pending"))
	w.wantStep(w.addition, false)
	w.observe(w.report(w.reports[0], "closed"))
	w.wantStep(w.addition, false)
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(w.addition, true)
}
func TestD165_T7_IsolationPendingIsPerTarget(t *testing.T) {
	for _, resolution := range []string{"clean", "closed"} {
		t.Run(resolution, func(t *testing.T) {
			w := newD165World(t, 1)
			op := w.revokeDevice()
			w.dispatch("device.revoked")
			w.observe(w.report(w.reports[0], "pending"))
			w.wantStep(op, false)
			st, _ := nodesStep(w.operation(op))
			code, _ := st.ErrorCode.Get()
			if code != api.ErrorCodeISOLATIONPOLICYINVALID {
				t.Fatalf("D070 code=%s", code)
			}
			w.target(op, w.reports[0].NodeId, "pending")
			w.observe(w.report(w.reports[0], resolution))
			w.wantStep(op, true)
			w.target(op, w.reports[0].NodeId, "acknowledged")
		})
	}
}
func TestD165_T8_ConcurrentLastAcknowledgements(t *testing.T) {
	w := newD165World(t, 2)
	op := w.revokeDevice()
	w.dispatch("device.revoked")
	reports := []api.ObservationReport{w.report(w.reports[0], "clean"), w.report(w.reports[1], "clean")}
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, r := range reports {
		wg.Go(func() {
			<-start
			_, err := w.r.Nodes.SubmitObservation(t.Context(), r.NodeId, r.NodeIncarnationId, r)
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	w.wantStep(op, true)
	for _, r := range reports {
		w.target(op, r.NodeId, "acknowledged")
	}
}
func TestD165_T9_UnrelatedCompileDoesNotMovePin(t *testing.T) {
	w := newD165World(t, 1)
	op := w.revokeDevice()
	w.dispatch("device.revoked")
	pinned := w.report(w.reports[0], "clean")
	if _, err := w.compiler.Compile(t.Context(), pinned.NodeId); err != nil {
		t.Fatal(err)
	}
	w.observe(pinned)
	w.wantStep(op, true)
}
func TestD165_T10_ReplacementIncarnationAcknowledgesPin(t *testing.T) {
	w := newD165World(t, 1)
	op := w.revokeDevice()
	w.dispatch("device.revoked")
	old := w.reports[0]
	detail, _, err := w.r.Nodes.GetNode(t.Context(), old.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	cmd := enrollment.Command{Principal: "admin:" + w.admin.String(), Key: uuid.New()}
	if _, err = w.r.Enrollment.RevokeNode(t.Context(), cmd, old.NodeId, &detail.ResourceVersion, "TEST ONLY reinstall"); err != nil {
		t.Fatal(err)
	}
	created, err := w.r.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: cmd.Principal, Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &old.NodeId})
	if err != nil {
		t.Fatal(err)
	}
	token, _ := created.Token.Get()
	request, _ := pkitest.Request(t)
	replacement, err := w.r.Enrollment.Enroll(t.Context(), token, request)
	if err != nil {
		t.Fatal(err)
	}
	fresh := cloneReport(old, old.NodeId, replacement.Response.NodeIncarnationId)
	fresh, _ = w.prepare(fresh, false, false)
	w.observe(fresh)
	w.wantStep(op, true)
	w.target(op, old.NodeId, "acknowledged")
	var pin uuid.UUID
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT incarnation_id FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, op, old.NodeId).Scan(&pin); err != nil || pin != old.NodeIncarnationId {
		t.Fatal("D165 pin changed with incarnation", pin, err)
	}
}

type d165AfterCompile struct {
	compiler membersruntime.Compiler
	after    func()
}

func (c d165AfterCompile) RecompileBatch(ctx context.Context, events []db.ClaimedEvent) ([]desiredstate.CompiledDesiredState, error) {
	out, err := c.compiler.RecompileBatch(ctx, events)
	if err == nil {
		c.after()
	}
	return out, err
}
func TestD165_T11_ReportBeforeCaptureCommitCounts(t *testing.T) {
	w := newD165World(t, 1)
	op := w.revokeDevice()
	w.services.Compiler = d165AfterCompile{compiler: w.compiler, after: func() { w.observe(w.report(w.reports[0], "clean")); w.wantStep(op, false) }}
	w.dispatch("device.revoked")
	w.wantStep(op, true)
	w.target(op, w.reports[0].NodeId, "acknowledged")
}

// captureExistingMemberOperations routes older readiness fixtures through the
// same worker boundary. SQL reads supply only the TEST ONLY external adapter;
// target/capture/operation writes remain owned by product code.
func captureExistingMemberOperations(t *testing.T, r *httpwire.Runtime, compiler *desiredstate.Compiler) {
	t.Helper()
	store, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	peers := &d165Peers{peers: map[string]devices.Peer{}}
	rows, err := r.Pool.Query(t.Context(), `SELECT d.deployment_id,d.peer_id,u.external_user_id,b.overlay_scope,host(b.overlay_ip),d.status='authorized',coalesce(e.projection->'exit_group_ids','[]') FROM member_devices d JOIN netbird_user_bindings u ON u.member_id=d.member_id AND u.deployment_id=d.deployment_id JOIN LATERAL (SELECT overlay_scope,overlay_ip FROM device_bindings WHERE device_id=d.id ORDER BY binding_revision DESC LIMIT 1) b ON true LEFT JOIN entitlements e ON e.member_id=d.member_id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var p devices.Peer
		var ip string
		var raw []byte
		if err = rows.Scan(&p.DeploymentID, &p.ID, &p.ExternalUserID, &p.OverlayScope, &ip, &p.Authorized, &raw); err != nil {
			t.Fatal(err)
		}
		p.OverlayIP = netip.MustParseAddr(ip)
		p.Platform = "linux"
		if err = json.Unmarshal(raw, &p.Groups); err != nil {
			t.Fatal(err)
		}
		peers.peers[p.ID] = p
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	services := &membersruntime.Services{HTTP: httpadapter.Services{Members: &members.Service{Pool: r.Pool, Entitlements: store}, Devices: &devices.Service{Pool: r.Pool, Entitlements: store, Adapter: peers}, Billing: &billing.Service{}}, Compiler: compiler}
	events, err := db.ClaimOutbox(t.Context(), r.Pool, "TEST ONLY existing capture", 1000, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	// A latest entitlement event includes earlier committed member operations.
	slices.Reverse(events)
	for _, event := range events {
		switch event.EventType {
		case "member.entitlement_changed", "member.suspended", "member.grant_revoked", "device.authorized", "device.revoked":
			if _, err = worker.MemberJobs()[event.EventType](worker.WithMembers(t.Context(), services), event); err != nil {
				t.Fatal(err)
			}
		}
	}
}

type d165Peers struct{ peers map[string]devices.Peer }

func (p *d165Peers) ObservePeer(_ context.Context, _, id string) (devices.Peer, error) {
	v, ok := p.peers[id]
	if !ok {
		return v, fmt.Errorf("TEST ONLY peer missing")
	}
	return v, nil
}
func (p *d165Peers) SetAuthorization(_ context.Context, _, id string, allow bool, groups []uuid.UUID) error {
	v := p.peers[id]
	v.Authorized = allow
	v.Groups = slices.Clone(groups)
	p.peers[id] = v
	return nil
}

func TestD165_RetryPreservesPinsAndAcceptedOutOfOrderAck(t *testing.T) {
	w := newD165World(t, 1)
	op := w.revokeDevice()
	events, err := db.ClaimOutbox(t.Context(), w.r.Pool, "TEST ONLY retry", 1000, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var event db.ClaimedEvent
	for _, e := range events {
		if e.EventType == "device.revoked" {
			event = e
		}
	}
	if event.ID == uuid.Nil {
		t.Fatal("missing revoke event")
	}
	ctx := worker.WithMembers(t.Context(), w.services)
	if _, err = worker.MemberJobs()[event.EventType](ctx, event); err != nil {
		t.Fatal(err)
	}
	pinned := w.report(w.reports[0], "clean")
	var original string
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT generation::text FROM node_operation_targets WHERE operation_id=$1`, op).Scan(&original); err != nil {
		t.Fatal(err)
	}
	// An idempotent event retry may compile again but must not replace its pin.
	if _, err = worker.MemberJobs()[event.EventType](ctx, event); err != nil {
		t.Fatal(err)
	}
	var again string
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT generation::text FROM node_operation_targets WHERE operation_id=$1`, op).Scan(&again); err != nil || again != original {
		t.Fatal("retry moved pin", again, original, err)
	}
	w.observe(w.report(w.reports[0], "pending"))
	// Arrives later with an earlier observed_at; the node projection stays newer,
	// but the accepted evidence still proves this operation's immutable pin.
	w.observe(pinned)
	w.wantStep(op, true)
	w.target(op, pinned.NodeId, "acknowledged")
}

func TestD165_RevokedAndRetiredTargetsAreUnconfirmed(t *testing.T) {
	for _, retire := range []bool{false, true} {
		t.Run(fmt.Sprintf("retired_%v", retire), func(t *testing.T) {
			w := newD165World(t, 1)
			op := w.revokeDevice()
			w.dispatch("device.revoked")
			r := w.reports[0]
			detail, _, err := w.r.Nodes.GetNode(t.Context(), r.NodeId)
			if err != nil {
				t.Fatal(err)
			}
			cmd := enrollment.Command{Principal: "admin:" + w.admin.String(), Key: uuid.New()}
			reason := "node_revoked"
			if retire {
				_, err = w.r.Enrollment.RetireNode(t.Context(), cmd, r.NodeId, &detail.ResourceVersion, "TEST ONLY retirement")
				reason = "node_retired"
			} else {
				_, err = w.r.Enrollment.RevokeNode(t.Context(), cmd, r.NodeId, &detail.ResourceVersion, "TEST ONLY revocation")
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.r.Nodes.SweepStaleNodes(t.Context(), 0, 0); err != nil {
				t.Fatal(err)
			}
			w.wantStep(op, true)
			w.target(op, r.NodeId, "unconfirmed")
			var got string
			if err = w.r.Pool.QueryRow(t.Context(), `SELECT reason FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, op, r.NodeId).Scan(&got); err != nil || got != reason {
				t.Fatal("unconfirmed reason", got, err)
			}
		})
	}
}

func TestD165_FutureGrantLinksThroughMovedRevision(t *testing.T) {
	w := newD165World(t, 1)
	removed, err := w.services.HTTP.Members.RevokeMemberGrant(t.Context(), d165Command("admin", w.admin), w.member, w.grant, "TEST ONLY disconnected future grant")
	if err != nil {
		t.Fatal(err)
	}
	w.dispatch("member.entitlement_changed")
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(*removed.OperationID, true)

	var revision uuid.UUID
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT plan_revision_id FROM entitlement_grants WHERE id=$1`, w.grant).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(500 * time.Millisecond)
	future, err := w.services.HTTP.Members.Grant(t.Context(), d165Command("admin", w.admin), w.member, api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: start.Add(13 * time.Hour), PlanRevisionId: &revision, Reason: "TEST ONLY future grant"})
	if err != nil {
		t.Fatal(err)
	}
	var before decstr.U64
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT desired_generation::text FROM operations WHERE id=$1`, future.OperationID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	// Replaying a previously claimed event must not capture the future operation.
	var old db.ClaimedEvent
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT id,aggregate_id,aggregate_type,aggregate_version::text,event_type,schema_version,payload,lease_owner,lease_token FROM outbox_events WHERE event_type='member.entitlement_changed' AND aggregate_id=$1 ORDER BY created_at,id LIMIT 1`, w.member).Scan(&old.ID, &old.AggregateID, &old.AggregateType, &old.AggregateVersion, &old.EventType, &old.SchemaVersion, &old.Payload, &old.Worker, &old.LeaseToken); err != nil {
		t.Fatal(err)
	}
	if _, err = worker.MemberJobs()[old.EventType](worker.WithMembers(t.Context(), w.services), old); err != nil {
		t.Fatal(err)
	}
	var captured bool
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM node_operation_captures WHERE operation_id=$1)`, future.OperationID).Scan(&captured); err != nil || captured {
		t.Fatal("future grant captured before activation", captured, err)
	}
	timer := time.NewTimer(time.Until(start.Add(time.Millisecond)))
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	if err = db.InTx(t.Context(), w.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		_, e := w.services.HTTP.Members.Entitlements.Recompute(t.Context(), tx, w.member)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	var after decstr.U64
	if err = w.r.Pool.QueryRow(t.Context(), `SELECT desired_generation::text FROM operations WHERE id=$1`, future.OperationID).Scan(&after); err != nil || after <= before {
		t.Fatal("future grant did not move to activation revision", before, after, err)
	}
	w.dispatch("member.entitlement_changed")
	w.wantStep(*future.OperationID, false)
	w.observe(w.report(w.reports[0], "clean"))
	w.wantStep(*future.OperationID, true)
	w.target(*future.OperationID, w.reports[0].NodeId, "acknowledged")
}
