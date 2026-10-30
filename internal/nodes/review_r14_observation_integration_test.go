//go:build integration

package nodes_test

import (
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// r14World is one isolated database. Node A comes from nodestest.BaselineObservation.
// Further nodes are enrolled through the same runtime. Desired generations come
// only from the compiler (D-050).
type r14World struct {
	t        *testing.T
	r        *httpwire.Runtime
	compiler *desiredstate.Compiler
	exits    *exits.Service
	now      time.Time
	admin    uuid.UUID
	primary  api.ObservationReport
}

func newR14World(t *testing.T) *r14World {
	t.Helper()
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	w := &r14World{t: t, r: r, now: report.ObservedAt.UTC(), admin: uuid.New(), primary: report}
	w.compiler = r.DesiredStateCompiler()
	w.exits = exits.NewService(r.Pool, w.compiler, nil, exits.WithClock(func() time.Time { return w.now }))
	w.observe(report)
	return w
}

func (w *r14World) observe(report api.ObservationReport) {
	w.t.Helper()
	if _, err := w.r.Nodes.SubmitObservation(w.t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		w.t.Fatalf("SubmitObservation %s: %v", report.NodeId, err)
	}
}

func (w *r14World) bump() time.Time {
	w.now = w.now.Add(time.Second)
	return w.now
}

func cloneReport(base api.ObservationReport, node, inc uuid.UUID) api.ObservationReport {
	base.NodeId, base.NodeIncarnationId, base.ReportId = node, inc, uuid.New()
	checks := make([]api.NodeCheck, len(base.Checks))
	copy(checks, base.Checks)
	base.Checks = checks
	base.Errors = nil
	base.MemberForwardingEnabled = false
	return base
}

func (w *r14World) enroll() api.ObservationReport {
	w.t.Helper()
	created, err := w.r.Enrollment.CreateEnrollment(w.t.Context(), enrollment.Command{Principal: "admin:" + w.admin.String(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller})
	if err != nil {
		w.t.Fatal(err)
	}
	token, err := created.Token.Get()
	if err != nil {
		w.t.Fatal(err)
	}
	request, _ := pkitest.Request(w.t)
	enrolled, err := w.r.Enrollment.Enroll(w.t.Context(), token, request)
	if err != nil {
		w.t.Fatal(err)
	}
	return cloneReport(w.primary, enrolled.Response.NodeId, enrolled.Response.NodeIncarnationId)
}

func (w *r14World) group(code, name string) uuid.UUID {
	w.t.Helper()
	var grp api.MemberGroup
	err := db.InTx(w.t.Context(), w.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var gErr error
		grp, gErr = w.r.Nodes.CreateGroupTx(w.t.Context(), tx, "admin:"+w.admin.String(), nil, api.MemberGroupCreate{Code: code, Name: name})
		return gErr
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return grp.GroupId
}

func (w *r14World) publish(node, group uuid.UUID) {
	w.t.Helper()
	_, etag, err := w.r.Nodes.GetNode(w.t.Context(), node)
	if err != nil {
		w.t.Fatal(err)
	}
	groups := []uuid.UUID{group}
	if _, _, err = w.exits.SetPublication(w.t.Context(), exits.SetPublicationParams{
		NodeID: node, Principal: "admin:" + w.admin.String(), IdempotencyKey: uuid.New(), IfMatch: etag,
		Request: api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY r14 publication", GroupIds: &groups},
	}); err != nil {
		w.t.Fatalf("SetPublication %s: %v", node, err)
	}
}

func (w *r14World) exitGroups(node uuid.UUID) int {
	w.t.Helper()
	var n int
	if err := w.r.Pool.QueryRow(w.t.Context(), `SELECT count(*) FROM exit_groups eg JOIN exits e ON e.id = eg.exit_id WHERE e.node_id=$1`, node).Scan(&n); err != nil {
		w.t.Fatal(err)
	}
	return n
}

// prepare compiles once and stamps that generation onto a copy of report.
// The returned binding count is the generation the report acknowledges.
func (w *r14World) prepare(report api.ObservationReport, forwarding, isolation bool) (api.ObservationReport, int) {
	w.t.Helper()
	comp, err := w.compiler.Compile(w.t.Context(), report.NodeId)
	if err != nil {
		w.t.Fatal(err)
	}
	at := w.bump()
	report.ReportId = uuid.New()
	report.ObservedAt = at
	report.AppliedConfigGeneration.Set(comp.Envelope.Signed.ConfigGeneration)
	report.MemberForwardingEnabled = forwarding
	report.SafetyStatus = api.SafetyStatusPassed
	report.ProvisioningState = api.ProvisioningStateReady
	report.DataHealth = api.DataHealthHealthy
	report.Errors = nil
	if isolation {
		report.Errors = []api.NodeError{{
			Code: api.ErrorCodeISOLATIONPOLICYINVALID, Stage: "apply",
			Message: "conntrack delete pending; serving ruleset verified", ObservedAt: at,
		}}
	}
	for i := range report.Checks {
		report.Checks[i].ObservedAt = at
	}
	return report, len(comp.Envelope.Signed.DeviceBindings)
}

type r14Member struct {
	id     uuid.UUID
	device uuid.UUID
}

func (w *r14World) bindDevice(label, overlay string, groups []uuid.UUID) r14Member {
	w.t.Helper()
	ctx := w.t.Context()
	memberID := uuid.New()
	identityID := uuid.New()
	planID := uuid.New()
	planRevID := uuid.New()
	user := "TEST ONLY " + label
	// The compiler and AuthorizeDevice compare validity to the database
	// clock. w.now follows observation time and is ahead of that clock
	// after prepare bumps it.
	now := time.Now().UTC()
	verified := now.Add(-time.Minute)
	if _, err := w.r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.r.Pool.Exec(ctx, `INSERT INTO identity_bindings (id, member_id, issuer, client_scope, subject, stable_claim_ref, binding_method, verified_at) VALUES ($1, $2, 'https://id.example.invalid', 'portal', $3, 'sub', 'oidc', $4)`, identityID, memberID, user, verified); err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.r.Pool.Exec(ctx, `INSERT INTO netbird_user_bindings (id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at) VALUES ($1, $2, $3, $4, $5, 'shared_issuer_public_subject:https://id.example.invalid', 1, $6)`, uuid.New(), memberID, w.r.Config.NetBirdDeployment, user, identityID, verified); err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, $2)`, planID, "plan-"+label); err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, groups); err != nil {
		w.t.Fatal(err)
	}
	store, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		w.t.Fatal(err)
	}
	store.Now = func() time.Time { return now }
	membersSvc := &members.Service{Pool: w.r.Pool, Entitlements: store}
	devicesSvc := &devices.Service{Pool: w.r.Pool, Entitlements: store}
	validFrom := now.Add(-time.Hour)
	validUntil := now.Add(24 * time.Hour)
	if _, err := membersSvc.Grant(ctx, entitlement.Command{Actor: entitlement.Actor{Role: "admin", ID: w.admin, RequestID: uuid.New()}, Key: uuid.New()}, memberID, api.GrantCreateRequest{
		Source: api.GrantSourceAdminGrant, ValidFrom: &validFrom, ValidUntil: validUntil, Reason: "TEST ONLY r14 grant", PlanRevisionId: &planRevID,
	}); err != nil {
		w.t.Fatal(err)
	}
	dev, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()}, Key: uuid.New()}, devices.Peer{
		DeploymentID: w.r.Config.NetBirdDeployment, ID: "TEST ONLY " + label + " peer", ExternalUserID: user, OverlayScope: "TEST ONLY", OverlayIP: netip.MustParseAddr(overlay), Platform: "linux",
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return r14Member{id: memberID, device: dev.DeviceID}
}

// revoke finishes the netbird step. The nodes step is what an observation can move.
func (w *r14World) revoke(m r14Member) uuid.UUID {
	w.t.Helper()
	ctx := w.t.Context()
	store, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		w.t.Fatal(err)
	}
	store.Now = func() time.Time { return w.now }
	devicesSvc := &devices.Service{Pool: w.r.Pool, Entitlements: store}
	out, err := devicesSvc.RevokeDevice(ctx, entitlement.Command{Actor: entitlement.Actor{Role: "member", ID: m.id, RequestID: uuid.New()}, Key: uuid.New()}, m.device)
	if err != nil || out.OperationID == nil {
		w.t.Fatalf("RevokeDevice: %v", err)
	}
	opID := *out.OperationID
	err = db.InTx(ctx, w.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var auditID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM audit_events LIMIT 1`).Scan(&auditID); err != nil {
			return err
		}
		return entitlement.TransitionOperation(ctx, tx, opID, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{
			Reference: "audit_event:" + auditID.String(), ObservedAt: w.now,
		})
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return opID
}

func (w *r14World) operation(id uuid.UUID) api.Operation {
	w.t.Helper()
	op, err := w.r.Nodes.GetOperation(w.t.Context(), id)
	if err != nil {
		w.t.Fatal(err)
	}
	return op
}

func nodesStep(op api.Operation) (api.OperationStep, bool) {
	for _, st := range op.Steps {
		if st.Name == "nodes" {
			return st, true
		}
	}
	return api.OperationStep{}, false
}

func stepText(st api.OperationStep) string {
	code, _ := st.ErrorCode.Get()
	msg, _ := st.Message.Get()
	return fmt.Sprintf("status=%s code=%s message=%s", st.Status, code, msg)
}

// TestReviewR14_AgentFailClosedDoesNotCompleteRevoke feeds the nodes step the
// report shape TestReviewR14_ConntrackBoundIsPerApplyAndSurvivesPartialSuccess
// builds: forwarding false, safety failed, provisioning ready, internet_only
// failed, and ISOLATION_POLICY_INVALID stage apply with
// "Conntrack delete exceeded the retry bound."
// Wire-nodes follow-up 20 is the in-flight alignment. This assertion records
// today's result and will fail when that change starts completing the step.
func TestReviewR14_AgentFailClosedDoesNotCompleteRevoke(t *testing.T) {
	w := newR14World(t)
	group := w.group("grp-r14-d070", "r14 D-070")
	w.publish(w.primary.NodeId, group)
	member := w.bindDevice("r14-d070", "100.64.0.31", []uuid.UUID{group})
	opID := w.revoke(member)
	rep, n := w.prepare(w.primary, false, false)
	if n != 0 {
		t.Fatalf("device still bound after revoke: %d", n)
	}
	const conntrackBound = "Conntrack delete exceeded the retry bound."
	rep.MemberForwardingEnabled = false
	rep.SafetyStatus = api.SafetyStatusFailed
	rep.ProvisioningState = api.ProvisioningStateReady
	rep.DataHealth = api.DataHealthDegraded
	detail := conntrackBound
	for i := range rep.Checks {
		if rep.Checks[i].Name == api.NodeCheckNameInternetOnly {
			rep.Checks[i].Status = api.HealthCheckStatusFailed
			rep.Checks[i].ErrorCode.Set(api.ErrorCodeISOLATIONPOLICYINVALID)
			rep.Checks[i].Detail = &detail
		}
	}
	rep.Errors = []api.NodeError{{
		Code: api.ErrorCodeISOLATIONPOLICYINVALID, Stage: "apply", Message: conntrackBound, ObservedAt: rep.ObservedAt,
	}}
	w.observe(rep)
	op := w.operation(opID)
	st, ok := nodesStep(op)
	if !ok {
		t.Fatal("nodes step missing")
	}
	if op.Status == api.OperationStatusSucceeded || st.Status == api.OperationStepStatusSucceeded {
		t.Fatalf("agent fail-closed observation completed the revoke: status=%s %s", op.Status, stepText(st))
	}
	t.Logf("D-070 today: operation=%s %s", op.Status, stepText(st))
}

// TestReviewR14_ExitGroupFenceHoldsRevokesForNodesThatDoNotHaveTheDevice
// is otherNodePendingIso (internal/nodes/observations.go). The predicate does
// not name the device and does not look at last_seen_at. A registered node
// with no exit_groups matches every member. A peer that reported a pending
// delete once keeps matching after it goes silent.
func TestReviewR14_ExitGroupFenceHoldsRevokesForNodesThatDoNotHaveTheDevice(t *testing.T) {
	t.Run("unpublished_node_holds_a_device_it_never_bound", func(t *testing.T) {
		w := newR14World(t)
		group := w.group("grp-r14-unpub", "r14 unpublished hostage")
		w.publish(w.primary.NodeId, group)
		member := w.bindDevice("r14-unpub", "100.64.0.41", []uuid.UUID{group})
		if _, n := w.prepare(w.primary, true, false); n != 1 {
			t.Fatalf("clean node bindings = %d, want 1", n)
		}
		opID := w.revoke(member)
		hostage := w.enroll()
		if w.exitGroups(hostage.NodeId) != 0 {
			t.Fatalf("unpublished node has exit groups")
		}
		dirty, n := w.prepare(hostage, true, true)
		if n != 0 {
			t.Fatalf("unpublished node compiled %d bindings for a device it does not publish", n)
		}
		w.observe(dirty)
		clean, n := w.prepare(w.primary, true, false)
		if n != 0 {
			t.Fatalf("clean node still bound after revoke: %d", n)
		}
		w.observe(clean)
		op := w.operation(opID)
		st, _ := nodesStep(op)
		if op.Status == api.OperationStatusSucceeded && st.Status == api.OperationStepStatusSucceeded {
			return
		}
		memberOp, err := w.r.Nodes.GetMemberOperation(w.t.Context(), member.id, opID)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("TODO(review): member_device.revoke %s for a device bound only to %s stayed %s (%s); member read %s; unpublished %s has no exit_groups and a pending delete while forwarding", opID, w.primary.NodeId, op.Status, stepText(st), memberOp.Status, hostage.NodeId)
	})

	t.Run("silent_same_group_peer_holds_the_revoke_an_hour_later", func(t *testing.T) {
		w := newR14World(t)
		group := w.group("grp-r14-dead", "r14 silent peer")
		peer := w.enroll()
		ready, _ := w.prepare(peer, false, false)
		w.observe(ready)
		w.publish(w.primary.NodeId, group)
		w.publish(peer.NodeId, group)
		member := w.bindDevice("r14-dead", "100.64.0.42", []uuid.UUID{group})
		if _, n := w.prepare(w.primary, true, false); n != 1 {
			t.Fatalf("clean node bindings = %d, want the shared group device", n)
		}
		if _, n := w.prepare(peer, true, false); n != 1 {
			t.Fatalf("peer bindings = %d, want the shared group device", n)
		}
		opID := w.revoke(member)
		dirty, n := w.prepare(peer, true, true)
		if n != 0 {
			t.Fatalf("peer still bound after revoke: %d", n)
		}
		w.observe(dirty)
		clean, n := w.prepare(w.primary, true, false)
		if n != 0 {
			t.Fatal(n)
		}
		w.observe(clean)
		if op := w.operation(opID); op.Status == api.OperationStatusSucceeded {
			t.Fatal("same-group dirty peer did not hold the revoke; the hour-later proof has nothing to observe")
		}
		var peerSeen time.Time
		if err := w.r.Pool.QueryRow(w.t.Context(), `SELECT last_observed_at FROM nodes WHERE id=$1`, peer.NodeId).Scan(&peerSeen); err != nil {
			t.Fatal(err)
		}
		w.now = w.now.Add(time.Hour)
		later, _ := w.prepare(w.primary, true, false)
		w.observe(later)
		op := w.operation(opID)
		st, _ := nodesStep(op)
		if op.Status == api.OperationStatusSucceeded && st.Status == api.OperationStepStatusSucceeded {
			return
		}
		t.Errorf("TODO(review): peer %s last reported at %s and then stayed silent; an hour later member_device.revoke %s is still %s (%s)", peer.NodeId, peerSeen.UTC().Format(time.RFC3339), opID, op.Status, stepText(st))
	})

	t.Run("different_group_does_not_hold_the_revoke", func(t *testing.T) {
		w := newR14World(t)
		home := w.group("grp-r14-home", "r14 home")
		other := w.group("grp-r14-other", "r14 other")
		peer := w.enroll()
		ready, _ := w.prepare(peer, false, false)
		w.observe(ready)
		w.publish(w.primary.NodeId, home)
		w.publish(peer.NodeId, other)
		member := w.bindDevice("r14-other", "100.64.0.43", []uuid.UUID{home})
		if _, n := w.prepare(w.primary, true, false); n != 1 {
			t.Fatalf("home bindings = %d, want 1", n)
		}
		if _, n := w.prepare(peer, true, false); n != 0 {
			t.Fatalf("other group compiled %d bindings", n)
		}
		opID := w.revoke(member)
		dirty, _ := w.prepare(peer, true, true)
		w.observe(dirty)
		clean, n := w.prepare(w.primary, true, false)
		if n != 0 {
			t.Fatal(n)
		}
		w.observe(clean)
		op := w.operation(opID)
		st, ok := nodesStep(op)
		if !ok || op.Status != api.OperationStatusSucceeded || st.Status != api.OperationStepStatusSucceeded {
			t.Fatalf("different-group dirty peer held the revoke: status=%s %s", op.Status, stepText(st))
		}
	})
}

// TestReviewR14_RepeatedStalePassedReportKeepsReady is the readiness OR in
// SubmitObservation. One in-flight passed report after a desired-generation
// advance stays ready, which is the concurrent-advance contract. A second
// report of that same old generation, still safety=passed, also stays ready.
// A safety=failed report of the old generation is downgraded.
func TestReviewR14_RepeatedStalePassedReportKeepsReady(t *testing.T) {
	t.Run("safety_failed_downgrades", func(t *testing.T) {
		w := newR14World(t)
		if _, err := w.compiler.Compile(w.t.Context(), w.primary.NodeId); err != nil {
			t.Fatal(err)
		}
		if _, err := w.compiler.Compile(w.t.Context(), w.primary.NodeId); err != nil {
			t.Fatal(err)
		}
		rep, _ := w.prepare(w.primary, false, false)
		baseline, err := w.primary.AppliedConfigGeneration.Get()
		if err != nil {
			t.Fatal(err)
		}
		rep.AppliedConfigGeneration.Set(baseline)
		rep.SafetyStatus = api.SafetyStatusFailed
		w.observe(rep)
		detail, _, err := w.r.Nodes.GetNode(w.t.Context(), w.primary.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		if detail.ProvisioningState != api.ProvisioningStateApplying || detail.SafetyStatus != api.SafetyStatusFailed {
			t.Fatalf("safety=failed stayed (%s, %s), want applying/failed", detail.ProvisioningState, detail.SafetyStatus)
		}
	})

	t.Run("second_stale_passed_report", func(t *testing.T) {
		w := newR14World(t)
		if _, err := w.compiler.Compile(w.t.Context(), w.primary.NodeId); err != nil {
			t.Fatal(err)
		}
		comp, err := w.compiler.Compile(w.t.Context(), w.primary.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		desired := comp.Envelope.Signed.ConfigGeneration
		first, _ := w.prepare(w.primary, false, false)
		// prepare compiled again. Re-stamp the pre-advance generation so this
		// report is the in-flight acknowledgement of the old generation.
		baseline, err := w.primary.AppliedConfigGeneration.Get()
		if err != nil {
			t.Fatal(err)
		}
		first.AppliedConfigGeneration.Set(baseline)
		w.observe(first)
		detail, _, err := w.r.Nodes.GetNode(w.t.Context(), w.primary.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		if detail.ProvisioningState != api.ProvisioningStateReady {
			t.Fatalf("first in-flight passed report became %s", detail.ProvisioningState)
		}
		second, _ := w.prepare(w.primary, false, false)
		second.AppliedConfigGeneration.Set(baseline)
		w.observe(second)
		detail, _, err = w.r.Nodes.GetNode(w.t.Context(), w.primary.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		applied, _ := detail.ConfigGeneration.Applied.Get()
		if detail.ProvisioningState == api.ProvisioningStateApplying && applied < detail.ConfigGeneration.Desired {
			return
		}
		t.Errorf("TODO(review): second safety=passed report of applied generation %s kept provisioning_state=%s safety=%s while desired is %s (compile had reached %s)", applied, detail.ProvisioningState, detail.SafetyStatus, detail.ConfigGeneration.Desired, desired)
	})
}
