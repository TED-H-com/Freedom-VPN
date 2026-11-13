//go:build integration

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestReviewR15_AwaitingMemberGrantsReportIsNotAFailure builds the report
// Decide and Engine.report emit for a fresh grantless lease, then submits
// that value through SubmitObservation. The checks and the single errors[]
// entry are the agent's, not a hand-written document.
func TestReviewR15_AwaitingMemberGrantsReportIsNotAFailure(t *testing.T) {
	r, enrolled, _ := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()
	nodeID := enrolled.Response.NodeId
	incID := enrolled.Response.NodeIncarnationId

	submit := func(gen decstr.U64, at time.Time) api.ObservationReport {
		t.Helper()
		rep := provisionalReport(t, gen, at)
		rep.NodeId = nodeID
		rep.NodeIncarnationId = incID
		rep.ReportId = uuid.New()
		rep.ObservedAt = at
		rep.CurrentReleaseId.SetNull()
		rep.LastGoodReleaseId.SetNull()
		if rep.Lease.LeaseId.IsNull() {
			rep.Lease.LeaseId.Set(uuid.New())
		}
		for i := range rep.Checks {
			rep.Checks[i].ObservedAt = at
		}
		for i := range rep.Errors {
			rep.Errors[i].ObservedAt = at
		}
		if len(rep.Errors) != 1 || rep.Errors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS || rep.Errors[0].Stage != "lease" {
			t.Fatalf("agent report errors = %+v, want one AWAITING_MEMBER_GRANTS at stage lease", rep.Errors)
		}
		if rep.SafetyStatus != api.SafetyStatusPassed || rep.ProvisioningState != api.ProvisioningStateReady || rep.MemberForwardingEnabled {
			t.Fatalf("agent report safety=%s provisioning=%s forwarding=%v", rep.SafetyStatus, rep.ProvisioningState, rep.MemberForwardingEnabled)
		}
		if _, err := r.Nodes.SubmitObservation(ctx, nodeID, incID, rep); err != nil {
			t.Fatalf("SubmitObservation: %v", err)
		}
		return rep
	}

	compiled, err := r.DesiredStateCompiler().Compile(ctx, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	rep := submit(compiled.Envelope.Signed.ConfigGeneration, now)

	detail, _, err := r.Nodes.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ProvisioningState != api.ProvisioningStateReady || detail.SafetyStatus != api.SafetyStatusPassed {
		t.Fatalf("stored provisioning=%s safety=%s, want ready/passed", detail.ProvisioningState, detail.SafetyStatus)
	}
	if detail.PublicationObserved != api.PublicationObservedAbsent {
		t.Fatalf("publication_observed=%s, want absent", detail.PublicationObserved)
	}
	if len(detail.RecentErrors) != 1 || detail.RecentErrors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS || detail.RecentErrors[0].Stage != "lease" {
		t.Fatalf("recent_errors=%+v", detail.RecentErrors)
	}
	fwd, err := detail.MemberForwardingObserved.Get()
	if err != nil || fwd {
		t.Fatalf("forwarding observed (%v, %v), want false", fwd, err)
	}

	var obsOK bool
	if err = r.Pool.QueryRow(ctx, `
		SELECT safety_status = 'passed' AND control_status = 'online'
		FROM node_observations WHERE node_id = $1
		ORDER BY received_at DESC, id DESC LIMIT 1`, nodeID).Scan(&obsOK); err != nil {
		t.Fatal(err)
	}
	if !obsOK {
		t.Fatal("FUP carryingExitAckStatus obsOK predicate is false for the provisional report")
	}

	admin := uuid.New()
	var grp api.MemberGroup
	if err = db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var gErr error
		grp, gErr = r.Nodes.CreateGroupTx(ctx, tx, "admin:"+admin.String(), nil, api.MemberGroupCreate{Code: "grp-r15-q5", Name: "r15 q5"})
		return gErr
	}); err != nil {
		t.Fatal(err)
	}
	exitsSvc := exits.NewService(r.Pool, r.DesiredStateCompiler(), nil)
	_, etag, err := r.Nodes.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	groups := []uuid.UUID{grp.GroupId}
	pubOp, _, err := exitsSvc.SetPublication(ctx, exits.SetPublicationParams{
		NodeID: nodeID, Principal: "admin:" + admin.String(), IdempotencyKey: uuid.New(), IfMatch: etag,
		Request: api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY r15 provisional publish", GroupIds: &groups},
	})
	if err != nil {
		t.Fatal(err)
	}
	compiled, err = r.DesiredStateCompiler().Compile(ctx, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	rep = submit(compiled.Envelope.Signed.ConfigGeneration, now.Add(time.Second))
	pubAfter, err := r.Nodes.GetOperation(ctx, pubOp.OperationId)
	if err != nil {
		t.Fatal(err)
	}
	var policy, probe api.OperationStepStatus
	for _, st := range pubAfter.Steps {
		switch st.Name {
		case nodes.StepAgentPolicy:
			policy = st.Status
		case nodes.StepDataPathProbe:
			probe = st.Status
		}
	}
	if pubAfter.Status == api.OperationStatusFailed || policy != api.OperationStepStatusSucceeded || probe == api.OperationStepStatusFailed {
		t.Fatalf("provisional lease failed node.publish %s policy=%s probe=%s", pubAfter.Status, policy, probe)
	}

	nb := exits.NewFakeNetBirdAdapter()
	if _, err = r.Pool.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, desiredstate.EventNodePublicationChanged); err != nil {
		t.Fatal(err)
	}
	d, err := worker.New(r.Pool, worker.NodeJobs(r, nil, nil, 3, nb), worker.Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Millisecond}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp() WHERE event_type=$1 AND payload->>'operation_id'=$2 AND delivered_at IS NULL`, desiredstate.EventNodePublicationChanged, pubOp.OperationId.String()); err != nil {
		t.Fatal(err)
	}
	if worked, err := d.Once(ctx); err != nil || !worked {
		t.Fatal("publish worker", worked, err)
	}
	var reason, opStatus string
	var opError *string
	if err = r.Pool.QueryRow(ctx, `SELECT status, error_code, steps->'data_path_probe'->>'reason' FROM operations WHERE id=$1`, pubOp.OperationId).Scan(&opStatus, &opError, &reason); err != nil {
		t.Fatal(err)
	}
	if opStatus == "failed" || opError != nil {
		t.Fatalf("worker treated the provisional report as a publish failure status=%s error=%v reason=%s", opStatus, opError, reason)
	}
	t.Logf("publication gate reason=%s operation=%s (wait, not a failure). agent errors=%d safety=%s", reason, opStatus, len(rep.Errors), rep.SafetyStatus)

	memberID := uuid.New()
	identityID := uuid.New()
	planID := uuid.New()
	planRevID := uuid.New()
	if _, err = r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `INSERT INTO identity_bindings (id, member_id, issuer, client_scope, subject, stable_claim_ref, binding_method, verified_at) VALUES ($1, $2, 'https://id.example.invalid', 'portal', $3, 'sub', 'oidc', $4)`, identityID, memberID, "TEST ONLY r15 q5", now); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `INSERT INTO netbird_user_bindings (id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at) VALUES ($1, $2, $3, $4, $5, 'shared_issuer_public_subject:https://id.example.invalid', 1, $6)`, uuid.New(), memberID, r.Config.NetBirdDeployment, "TEST ONLY r15 q5", identityID, now); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, $2)`, planID, "plan-r15-q5"); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, []uuid.UUID{grp.GroupId}); err != nil {
		t.Fatal(err)
	}
	store, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	store.Now = func() time.Time { return now }
	membersSvc := &members.Service{Pool: r.Pool, Entitlements: store}
	peer := &provisionalReportPeer{peer: devices.Peer{
		DeploymentID: r.Config.NetBirdDeployment, ID: "TEST ONLY r15 q5 peer", ExternalUserID: "TEST ONLY r15 q5", OverlayScope: "TEST ONLY", OverlayIP: netip.MustParseAddr("100.64.0.71"), Platform: "linux",
	}}
	devicesSvc := &devices.Service{Pool: r.Pool, Entitlements: store, Adapter: peer}
	services := &membersruntime.Services{HTTP: httpadapter.Services{Members: membersSvc, Devices: devicesSvc, Billing: &billing.Service{}}, Compiler: r.DesiredStateCompiler()}
	dispatch := func(want ...string) {
		t.Helper()
		events, err := db.ClaimOutbox(ctx, r.Pool, "TEST ONLY provisional report", 1000, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		var ran int
		// ClaimOutbox returns delivery order (available_at, id). Drain each
		// command before issuing the next so authorization is compiled before revoke.
		for _, event := range events {
			job := worker.MemberJobs()[event.EventType]
			if job == nil {
				continue
			}
			if ran >= len(want) || event.EventType != want[ran] {
				t.Fatalf("member worker event=%s at index %d, want %v", event.EventType, ran, want)
			}
			if _, err = job(worker.WithMembers(ctx, services), event); err != nil {
				t.Fatalf("member worker %s: %v", event.EventType, err)
			}
			if err = db.CompleteOutbox(ctx, r.Pool, event); err != nil {
				t.Fatal(err)
			}
			ran++
			t.Logf("member worker event=%s", event.EventType)
		}
		if ran != len(want) {
			t.Fatalf("member worker ran %d events, want %v", ran, want)
		}
	}
	validFrom := now.Add(-time.Hour)
	validUntil := now.Add(12 * time.Hour)
	if _, err = membersSvc.Grant(ctx, entitlement.Command{Actor: entitlement.Actor{Role: "admin", ID: admin, RequestID: uuid.New()}, Key: uuid.New()}, memberID, api.GrantCreateRequest{
		Source: api.GrantSourceAdminGrant, ValidFrom: &validFrom, ValidUntil: validUntil, Reason: "TEST ONLY r15 q5 grant", PlanRevisionId: &planRevID,
	}); err != nil {
		t.Fatal(err)
	}
	dispatch("member.entitlement_changed")
	dev, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()}, Key: uuid.New()}, peer.peer)
	if err != nil || dev.OperationID == nil {
		t.Fatal(err)
	}
	dispatch("device.authorized", "netbird.device_sync")
	var bindingsJSON []byte
	if err = r.Pool.QueryRow(ctx, `SELECT envelope->'signed'->'device_bindings' FROM desired_state_envelopes WHERE node_id=$1 AND incarnation_id=$2`, nodeID, incID).Scan(&bindingsJSON); err != nil {
		t.Fatal(err)
	}
	t.Logf("before revoke device_bindings=%s", bindingsJSON)
	var bindings []api.DeviceBinding
	if err = json.Unmarshal(bindingsJSON, &bindings); err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 || bindings[0].DeviceId != dev.DeviceID || bindings[0].MemberId != memberID {
		t.Fatalf("published node must carry the authorized device before revoke: %s", bindingsJSON)
	}
	// The addition has no removal fallback: its target proves that the
	// worker's compile records pins, before the revoke records absence.
	var additionPresent bool
	if err = r.Pool.QueryRow(ctx, `SELECT subject_present FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, *dev.OperationID, nodeID).Scan(&additionPresent); err != nil || !additionPresent {
		t.Fatalf("authorization target missing or absent: present=%v err=%v", additionPresent, err)
	}
	out, err := devicesSvc.RevokeDevice(ctx, entitlement.Command{Actor: entitlement.Actor{Role: "member", ID: memberID, RequestID: uuid.New()}, Key: uuid.New()}, dev.DeviceID)
	if err != nil || out.OperationID == nil {
		t.Fatal(err)
	}
	dispatch("device.revoked", "netbird.device_sync")
	var targetGeneration decstr.U64
	var targetIncarnation uuid.UUID
	var present bool
	var outcome, targetReason string
	readTarget := func(phase string) {
		t.Helper()
		if err := r.Pool.QueryRow(ctx, `SELECT incarnation_id,generation::text,subject_present,outcome,coalesce(reason,'') FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, *out.OperationID, nodeID).Scan(&targetIncarnation, &targetGeneration, &present, &outcome, &targetReason); err != nil {
			t.Fatalf("%s revoke target: %v", phase, err)
		}
		t.Logf("%s revoke target: operation_id=%s node_id=%s incarnation_id=%s generation=%s subject_present=%v outcome=%s reason=%s", phase, *out.OperationID, nodeID, targetIncarnation, targetGeneration, present, outcome, targetReason)
	}
	readTarget("before report")
	if targetIncarnation != incID || present || outcome != "pending" {
		t.Fatalf("before provisional report: incarnation=%s present=%v outcome=%s, want current incarnation, absent, pending", targetIncarnation, present, outcome)
	}
	before, err := r.Nodes.GetOperation(ctx, *out.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Status == api.OperationStatusSucceeded || before.Status == api.OperationStatusFailed {
		t.Fatalf("revoke must wait for the provisional report, got %s", before.Status)
	}
	rep = submit(targetGeneration, now.Add(2*time.Second))
	readTarget("after report")
	if outcome != "acknowledged" {
		t.Fatalf("provisional report did not acknowledge revoke target: outcome=%s", outcome)
	}
	if targetReason != "clean_ready" {
		t.Fatalf("provisional report revoke target reason=%s, want clean_ready", targetReason)
	}
	var reportID uuid.UUID
	if err = r.Pool.QueryRow(ctx, `SELECT report_id FROM node_operation_targets WHERE operation_id=$1 AND node_id=$2`, *out.OperationID, nodeID).Scan(&reportID); err != nil || reportID != rep.ReportId {
		t.Fatalf("revoke target report=%s, want provisional report=%s: %v", reportID, rep.ReportId, err)
	}
	revoked, err := r.Nodes.GetOperation(ctx, *out.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Status == api.OperationStatusFailed {
		t.Fatalf("provisional lease failed member_device.revoke: %+v", revoked)
	}
	if revoked.Status != api.OperationStatusSucceeded {
		t.Fatalf("member_device.revoke stayed %s under the provisional report (steps %+v)", revoked.Status, revoked.Steps)
	}
}

func provisionalReport(t *testing.T, gen decstr.U64, at time.Time) api.ObservationReport {
	t.Helper()
	in := servingInput()
	in.Now = at
	ds := *in.Desired
	ds.IssuedAt, ds.ExpiresAt = at.Add(-time.Minute), at.Add(time.Hour)
	ds.ConfigGeneration = gen
	ds.DeviceBindings = []api.DeviceBinding{oneBinding(at)}
	in.Desired = &ds
	lease := *in.Lease
	lease.ConfigGeneration = gen
	lease.Members = nil
	lease.LeaseId = uuid.New()
	lease.IssuedAt = at.Add(-time.Minute)
	lease.ExpiresAt = at.Add(time.Hour)
	in.Lease = &lease
	_, rep := reported(t, in)
	return rep
}

// provisionalReportPeer is TEST ONLY NetBird state. The real member worker owns
// authorization read-back, operation capture, compilation and completion.
type provisionalReportPeer struct{ peer devices.Peer }

func (p *provisionalReportPeer) ObservePeer(_ context.Context, deployment, id string) (devices.Peer, error) {
	if deployment != p.peer.DeploymentID || id != p.peer.ID {
		return devices.Peer{}, fmt.Errorf("TEST ONLY peer missing")
	}
	return p.peer, nil
}

func (p *provisionalReportPeer) SetAuthorization(_ context.Context, deployment, id string, allow bool, groups []uuid.UUID) error {
	if deployment != p.peer.DeploymentID || id != p.peer.ID {
		return fmt.Errorf("TEST ONLY peer missing")
	}
	p.peer.Authorized = allow
	p.peer.Groups = append([]uuid.UUID(nil), groups...)
	return nil
}
