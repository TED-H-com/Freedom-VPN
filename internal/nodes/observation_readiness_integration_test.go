//go:build integration

package nodes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func buildAgentObservationReport(
	base api.ObservationReport,
	desired api.DesiredState,
	at time.Time,
	revocationPending bool,
	conntrackBlocked bool,
	conntrackDetail string,
) api.ObservationReport {
	at = at.UTC()
	relID := uuid.MustParse("019a0000-0000-7000-8000-0000000000c1")
	if len(desired.ServingReleaseIds) > 0 {
		relID = desired.ServingReleaseIds[0]
	}
	ds := desired
	if !ds.NodeCapacity.IsSpecified() || ds.NodeCapacity.IsNull() {
		ds.NodeCapacity = presence.NewNullableWithValue(api.NodeCapacity{
			UpBps:   200_000_000,
			DownBps: 200_000_000,
		})
	}
	membersList := make([]api.MemberLease, 0, len(ds.DeviceBindings))
	seen := make(map[uuid.UUID]bool, len(ds.DeviceBindings))
	for _, b := range ds.DeviceBindings {
		if seen[b.MemberId] {
			continue
		}
		seen[b.MemberId] = true
		end := at.Add(time.Hour)
		membersList = append(membersList, api.MemberLease{
			MemberId:            b.MemberId,
			LeaseEnd:            end,
			EffectiveServiceEnd: end,
		})
	}
	srvLease := &api.ServingLease{
		LeaseId:           uuid.New(),
		NodeId:            base.NodeId,
		NodeIncarnationId: base.NodeIncarnationId,
		ConfigGeneration:  ds.ConfigGeneration,
		IssuedAt:          at,
		ExpiresAt:         at.Add(time.Hour),
		MaxOfflineSeconds: 86400,
		Members:           membersList,
	}
	in := agent.Input{
		Now:               at,
		MonoNow:           2 * time.Second,
		Datapath:          enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.37")},
		Desired:           &ds,
		DesiredOK:         true,
		Lease:             srvLease,
		LeaseFresh:        true,
		LeaseMonoSent:     time.Second,
		AnchorOK:          true,
		Backend:           "kernel",
		ReleaseID:         relID,
		Enrolled:          true,
		BaselineOK:        true,
		StorageOK:         true,
		HostReserveBPS:    1_000_000,
		ProbeRan:          true,
		ProbeCheck:        api.HealthCheckStatusPassed,
		ProbeHealth:       api.DataHealthHealthy,
		DNSDocument:       true,
		RevocationPending: revocationPending,
		ConntrackBlocked:  conntrackBlocked,
		ConntrackDetail:   conntrackDetail,
	}
	d := agent.Decide(in)

	rep := base
	rep.ReportId = uuid.New()
	rep.ObservedAt = at
	if d.Applied > 0 {
		rep.AppliedConfigGeneration.Set(decstr.U64(d.Applied))
		rep.AppliedBindingRevision.Set(ds.BindingRevision)
		rep.AppliedPolicyRefs.Set(ds.PolicyRefs)
	} else {
		rep.AppliedConfigGeneration.SetNull()
		rep.AppliedBindingRevision.SetNull()
		rep.AppliedPolicyRefs.SetNull()
	}
	rep.MemberForwardingEnabled = d.Serve
	rep.ProvisioningState = d.Provisioning
	rep.SafetyStatus = d.Safety
	rep.DataHealth = d.DataHealth
	if !rep.DataHealth.Valid() {
		rep.DataHealth = api.DataHealthUnknown
	}
	if d.Reasons != nil {
		rep.ForwardingBlockReasons = d.Reasons
	} else {
		rep.ForwardingBlockReasons = []api.ForwardingBlockReason{}
	}
	if d.Checks != nil {
		rep.Checks = d.Checks
	} else {
		rep.Checks = []api.NodeCheck{}
	}
	if d.Errors != nil {
		rep.Errors = d.Errors
	} else {
		rep.Errors = []api.NodeError{}
	}
	rep.Lease.LeaseId.Set(srvLease.LeaseId)
	rep.Lease.ExpiresAt.Set(srvLease.ExpiresAt.UTC())
	rep.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	return rep
}

func TestAT06_ObservationBaselineReadinessWithoutManagementPeer(t *testing.T) {
	for _, name := range []string{"healthy", "degraded", "failed_safety", "failed_evidence", "behind_generation", "missing_generation", "present_peer"} {
		t.Run(name, func(t *testing.T) {
			r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
			switch name {
			case "healthy":
				report.DataHealth = api.DataHealthHealthy
			case "failed_safety":
				report.SafetyStatus = api.SafetyStatusFailed
			case "failed_evidence":
				report.Checks[0].Status = api.HealthCheckStatusFailed
			case "behind_generation":
				report.AppliedConfigGeneration.Set(0)
			case "missing_generation":
				report.AppliedConfigGeneration.SetNull()
			case "present_peer":
				report.Netbird.PeerId.Set("TEST ONLY peer")
			}
			first, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report)
			if err != nil || first != replay {
				t.Fatal("observation replay", replay, err)
			}
			detail, _, err := r.Nodes.GetNode(t.Context(), report.NodeId)
			if err != nil {
				t.Fatal(err)
			}
			wantReady := name == "healthy" || name == "degraded"
			wantState := api.ProvisioningStateApplying
			if wantReady {
				wantState = api.ProvisioningStateReady
			}
			applied, err := detail.ConfigGeneration.Applied.Get()
			if err != nil || detail.ProvisioningState != wantState || !detail.NetbirdPeerId.IsNull() || detail.PublicationIntent != api.PublicationIntentUnpublished {
				t.Fatal("baseline/binding projection", detail, err)
			}
			readiness := nodes.ServingReadiness{LifecycleState: detail.LifecycleState, ProvisioningState: detail.ProvisioningState, PublicationIntent: detail.PublicationIntent, ControlStatus: detail.ControlStatus, SafetyStatus: detail.SafetyStatus, DataHealth: detail.DataHealth, DesiredConfigGeneration: uint64(detail.ConfigGeneration.Desired), AppliedConfigGeneration: uint64(applied)}
			if err := nodes.CheckPublishable(readiness); (err == nil) != wantReady {
				t.Fatal("publication acceptance", readiness, err)
			}
			if wantReady && (detail.SafetyStatus != api.SafetyStatusPassed || applied != detail.ConfigGeneration.Desired || readiness.IsServing()) {
				t.Fatal("readiness must not imply member serving", detail)
			}
			if (name == "failed_safety" || name == "failed_evidence") && detail.SafetyStatus != api.SafetyStatusFailed {
				t.Fatal("failed evidence was not retained", detail)
			}
			var receipts, events int
			if err := r.Pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM node_observations WHERE report_id=$1),(SELECT count(*) FROM outbox_events WHERE event_type='node.peer.observe' AND payload->>'report_id'=$1::text)`, report.ReportId).Scan(&receipts, &events); err != nil || receipts != 1 || events != 1 {
				t.Fatal("deduplicated store/worker intent", receipts, events, err)
			}
			if wantReady {
				// An older failed report stays in history and cannot undo the
				// current baseline acknowledgement.
				report.ReportId, report.ObservedAt = uuid.New(), report.ObservedAt.Add(-time.Second)
				report.SafetyStatus = api.SafetyStatusFailed
				if _, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
					t.Fatal(err)
				}
				after, _, err := r.Nodes.GetNode(t.Context(), report.NodeId)
				if err != nil || after.ProvisioningState != api.ProvisioningStateReady || after.SafetyStatus != api.SafetyStatusPassed {
					t.Fatal("stale report regressed readiness", after, err)
				}
			}
		})
	}
}

func TestM3_RevokeOperationHeldOnPendingIsolationApplyErrorAndCompletedOnResolution(t *testing.T) {
	t.Run("device_revoke_completes_on_next_clean_observation", func(t *testing.T) {
		r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
		ctx := t.Context()
		now := report.ObservedAt.UTC()

		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
			t.Fatalf("SubmitObservation baseline: %v", err)
		}

		adminID := uuid.New()
		var grp api.MemberGroup
		if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var gErr error
			grp, gErr = r.Nodes.CreateGroupTx(ctx, tx, "admin:"+adminID.String(), nil, api.MemberGroupCreate{
				Code: "grp-m3-clean",
				Name: "M3 Clean Group",
			})
			return gErr
		}); err != nil {
			t.Fatalf("CreateGroupTx: %v", err)
		}

		compiler := r.DesiredStateCompiler()
		exitsSvc := exits.NewService(r.Pool, compiler, nil, exits.WithClock(func() time.Time { return now }))
		_, etag, err := r.Nodes.GetNode(ctx, report.NodeId)
		if err != nil {
			t.Fatalf("GetNode: %v", err)
		}
		groupIDs := []uuid.UUID{grp.GroupId}
		if _, _, err := exitsSvc.SetPublication(ctx, exits.SetPublicationParams{
			NodeID:         report.NodeId,
			Principal:      "admin:" + adminID.String(),
			IdempotencyKey: uuid.New(),
			IfMatch:        etag,
			Request: api.PublicationRequest{
				PublicationIntent: api.PublicationIntentPublished,
				Reason:            "TEST ONLY M-3 revoke test",
				GroupIds:          &groupIDs,
			},
		}); err != nil {
			t.Fatalf("SetPublication: %v", err)
		}

		memberID := uuid.New()
		identityID := uuid.New()
		planID := uuid.New()
		planRevID := uuid.New()
		if _, err := r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
			t.Fatalf("insert member: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO identity_bindings (id, member_id, issuer, client_scope, subject, stable_claim_ref, binding_method, verified_at) VALUES ($1, $2, 'https://id.example.invalid', 'portal', 'TEST ONLY m3 user', 'sub', 'oidc', $3)`, identityID, memberID, now); err != nil {
			t.Fatalf("insert identity_binding: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO netbird_user_bindings (id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at) VALUES ($1, $2, $3, 'TEST ONLY m3 user', $4, 'shared_issuer_public_subject:https://id.example.invalid', 1, $5)`, uuid.New(), memberID, r.Config.NetBirdDeployment, identityID, now); err != nil {
			t.Fatalf("insert netbird_user_binding: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, 'plan-m3-clean')`, planID); err != nil {
			t.Fatalf("insert plan: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, []uuid.UUID{grp.GroupId}); err != nil {
			t.Fatalf("insert plan_revision: %v", err)
		}

		entStore, err := entitlement.NewStore(entitlement.TestConfig())
		if err != nil {
			t.Fatalf("entitlement.NewStore: %v", err)
		}
		entStore.Now = func() time.Time { return now }
		membersSvc := &members.Service{Pool: r.Pool, Entitlements: entStore}
		devicesSvc := &devices.Service{Pool: r.Pool, Entitlements: entStore}

		validFrom := now.Add(-time.Hour)
		validUntil := now.Add(12 * time.Hour)
		if _, err := membersSvc.Grant(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "admin", ID: adminID, RequestID: uuid.New()},
			Key:   uuid.New(),
		}, memberID, api.GrantCreateRequest{
			Source:         api.GrantSourceAdminGrant,
			ValidFrom:      &validFrom,
			ValidUntil:     validUntil,
			Reason:         "TEST ONLY M-3 grant",
			PlanRevisionId: &planRevID,
		}); err != nil {
			t.Fatalf("members.Grant: %v", err)
		}

		devOut, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()},
			Key:   uuid.New(),
		}, devices.Peer{
			DeploymentID:   r.Config.NetBirdDeployment,
			ID:             "TEST ONLY m3 peer 1",
			ExternalUserID: "TEST ONLY m3 user",
			OverlayScope:   "TEST ONLY",
			OverlayIP:      netip.MustParseAddr("100.64.0.21"),
			Platform:       "linux",
		})
		if err != nil {
			t.Fatalf("AuthorizeDevice 1: %v", err)
		}
		if _, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()},
			Key:   uuid.New(),
		}, devices.Peer{
			DeploymentID:   r.Config.NetBirdDeployment,
			ID:             "TEST ONLY m3 peer 2",
			ExternalUserID: "TEST ONLY m3 user",
			OverlayScope:   "TEST ONLY",
			OverlayIP:      netip.MustParseAddr("100.64.0.22"),
			Platform:       "linux",
		}); err != nil {
			t.Fatalf("AuthorizeDevice 2: %v", err)
		}

		compAuth, err := compiler.Compile(ctx, report.NodeId)
		if err != nil {
			t.Fatalf("Compile after AuthorizeDevice: %v", err)
		}
		now = now.Add(time.Second)
		authReadyReport := buildAgentObservationReport(report, compAuth.Envelope.Signed, now, false, false, "")
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, authReadyReport); err != nil {
			t.Fatalf("SubmitObservation auth ready: %v", err)
		}

		// Revoke device 1 via devices.Service.RevokeDevice and complete the netbird step via entitlement.TransitionOperation.
		revokeOut, err := devicesSvc.RevokeDevice(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "member", ID: memberID, RequestID: uuid.New()},
			Key:   uuid.New(),
		}, devOut.DeviceID)
		if err != nil || revokeOut.OperationID == nil {
			t.Fatalf("RevokeDevice: %v", err)
		}
		revokeOpID := *revokeOut.OperationID
		if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var auditID uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT id FROM audit_events LIMIT 1`).Scan(&auditID); err != nil {
				return err
			}
			return entitlement.TransitionOperation(ctx, tx, revokeOpID, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{
				Reference:  "audit_event:" + auditID.String(),
				ObservedAt: now,
			})
		}); err != nil {
			t.Fatalf("TransitionOperation netbird: %v", err)
		}

		compRevoke, err := compiler.Compile(ctx, report.NodeId)
		if err != nil {
			t.Fatalf("Compile after RevokeDevice: %v", err)
		}
		revokeGen := compRevoke.Envelope.Signed.ConfigGeneration
		if len(compRevoke.Envelope.Signed.DeviceBindings) != 1 {
			t.Fatalf("DeviceBindings after revoke = %d, want 1", len(compRevoke.Envelope.Signed.DeviceBindings))
		}

		// Submit observation on revokeGen built from agent.Decide with RevocationPending=true (D-070):
		// code=ISOLATION_POLICY_INVALID, stage=apply, message="conntrack delete pending; serving ruleset verified",
		// member_forwarding_enabled=true, safety_status=passed, every check passed.
		now = now.Add(time.Second)
		pendingReport := buildAgentObservationReport(report, compRevoke.Envelope.Signed, now, true, false, "")
		if !pendingReport.MemberForwardingEnabled || pendingReport.SafetyStatus != api.SafetyStatusPassed {
			t.Fatalf("pendingReport = (forwarding=%v, safety=%s), want (true, passed)", pendingReport.MemberForwardingEnabled, pendingReport.SafetyStatus)
		}
		for _, ch := range pendingReport.Checks {
			if ch.Status != api.HealthCheckStatusPassed {
				t.Fatalf("pendingReport check %s = %s, want passed", ch.Name, ch.Status)
			}
		}
		if len(pendingReport.Errors) != 1 || pendingReport.Errors[0].Code != api.ErrorCodeISOLATIONPOLICYINVALID || pendingReport.Errors[0].Stage != "apply" || pendingReport.Errors[0].Message != "conntrack delete pending; serving ruleset verified" {
			t.Fatalf("pendingReport.Errors = %+v, want single ISOLATION_POLICY_INVALID apply error", pendingReport.Errors)
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, pendingReport); err != nil {
			t.Fatalf("SubmitObservation with pending isolation error: %v", err)
		}

		// Verify node state reflects degraded health and recent_errors, and publication evidence is unconfirmed.
		nodeAfterPending, _, err := r.Nodes.GetNode(ctx, report.NodeId)
		if err != nil {
			t.Fatalf("GetNode after pending isolation error: %v", err)
		}
		if nodeAfterPending.DataHealth != api.DataHealthDegraded {
			t.Fatalf("DataHealth = %s, want degraded while ISOLATION_POLICY_INVALID is pending", nodeAfterPending.DataHealth)
		}
		cause, err := nodes.PublicationDataPathFailure(ctx, r.Pool, pendingReport, uint64(revokeGen), now)
		if err != nil || cause != "isolation_unconfirmed" {
			t.Fatalf("PublicationDataPathFailure = (%q, %v), want (isolation_unconfirmed, nil)", cause, err)
		}

		// Verify operation read models (Admin Store.Get, Member Store.Get, nodes.GetOperation, nodes.GetMemberOperation)
		// keep the operation running and the "nodes" step pending with ISOLATION_POLICY_INVALID surfaced.
		opStore := operations.Store{Pool: r.Pool}
		adminOp, err := opStore.Get(ctx, revokeOpID, auth.Principal{Role: auth.Admin, ID: adminID})
		if err != nil {
			t.Fatalf("opStore.Get(Admin): %v", err)
		}
		if adminOp.Status != api.OperationStatusRunning {
			t.Fatalf("adminOp.Status = %s, want running (must not complete while conntrack delete is pending)", adminOp.Status)
		}
		var foundNodesStep bool
		for _, st := range adminOp.Steps {
			if st.Name == "nodes" {
				foundNodesStep = true
				if st.Status != api.OperationStepStatusPending {
					t.Fatalf("adminOp nodes step Status = %s, want pending", st.Status)
				}
				code, cErr := st.ErrorCode.Get()
				if cErr != nil || code != api.ErrorCodeISOLATIONPOLICYINVALID {
					t.Fatalf("adminOp nodes step ErrorCode = (%v, %v), want ISOLATION_POLICY_INVALID", code, cErr)
				}
				msg, mErr := st.Message.Get()
				if mErr != nil || msg != "member_forwarding_unconfirmed" {
					t.Fatalf("adminOp nodes step Message = (%q, %v), want member_forwarding_unconfirmed", msg, mErr)
				}
			}
		}
		if !foundNodesStep {
			t.Fatal("nodes step missing on adminOp")
		}

		memberOpViaStore, err := opStore.Get(ctx, revokeOpID, auth.Principal{Role: auth.Member, ID: memberID})
		if err != nil {
			t.Fatalf("opStore.Get(Member): %v", err)
		}
		if memberOpViaStore.Status != api.OperationStatusRunning {
			t.Fatalf("memberOpViaStore.Status = %s, want running", memberOpViaStore.Status)
		}

		memberOpViaNodes, err := r.Nodes.GetMemberOperation(ctx, memberID, revokeOpID)
		if err != nil {
			t.Fatalf("Nodes.GetMemberOperation: %v", err)
		}
		if memberOpViaNodes.Status != api.OperationStatusRunning {
			t.Fatalf("memberOpViaNodes.Status = %s, want running", memberOpViaNodes.Status)
		}
		for _, st := range memberOpViaNodes.Steps {
			if st.Name == "nodes" {
				code, cErr := st.ErrorCode.Get()
				msg, mErr := st.Message.Get()
				if st.Status != api.OperationStepStatusPending || cErr != nil || code != api.ErrorCodeISOLATIONPOLICYINVALID || mErr != nil || msg != "member_forwarding_unconfirmed" {
					t.Fatalf("Nodes.GetMemberOperation nodes step = %+v, want pending with ISOLATION_POLICY_INVALID / member_forwarding_unconfirmed", st)
				}
			}
		}

		// Pending -> pending: a second pending-under-the-bound observation keeps the step pending and does not duplicate the pending audit.
		now = now.Add(time.Second)
		pendingReport2 := buildAgentObservationReport(report, compRevoke.Envelope.Signed, now, true, false, "")
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, pendingReport2); err != nil {
			t.Fatalf("SubmitObservation second pending isolation error: %v", err)
		}
		adminOpStillPending, err := opStore.Get(ctx, revokeOpID, auth.Principal{Role: auth.Admin, ID: adminID})
		if err != nil || adminOpStillPending.Status != api.OperationStatusRunning {
			t.Fatalf("adminOpStillPending = (%+v, %v), want running", adminOpStillPending, err)
		}

		// Verify audit_events recorded node.revoke_isolation_pending once and unreadable-anomaly count is 0.
		auditStore := &audit.Store{Pool: r.Pool}
		var pendingAuditCount int
		if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE operation_id = $1 AND action = 'node.revoke_isolation_pending' AND result = 'failed' AND after_diff->>'error_code' = 'ISOLATION_POLICY_INVALID' AND after_diff->>'reason' = 'conntrack_delete_pending'`, revokeOpID).Scan(&pendingAuditCount); err != nil || pendingAuditCount != 1 {
			t.Fatalf("node.revoke_isolation_pending audit count = %d (err=%v), want 1", pendingAuditCount, err)
		}

		// Clean observation built from agent.Decide on revokeGen -> revoke operation completes with conntrack_delete_succeeded.
		now = now.Add(time.Second)
		cleanReport := buildAgentObservationReport(report, compRevoke.Envelope.Signed, now, false, false, "")
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, cleanReport); err != nil {
			t.Fatalf("SubmitObservation clean after retry: %v", err)
		}

		adminOpAfter, err := opStore.Get(ctx, revokeOpID, auth.Principal{Role: auth.Admin, ID: adminID})
		if err != nil {
			t.Fatalf("opStore.Get(Admin) after clean observation: %v", err)
		}
		if adminOpAfter.Status != api.OperationStatusSucceeded {
			t.Fatalf("adminOpAfter.Status = %s, want succeeded", adminOpAfter.Status)
		}
		for _, st := range adminOpAfter.Steps {
			if st.Name == "nodes" {
				if st.Status != api.OperationStepStatusSucceeded || !st.ErrorCode.IsNull() {
					t.Fatalf("adminOpAfter nodes step = %+v, want succeeded with null ErrorCode", st)
				}
			}
		}

		var resolvedAuditCount int
		if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE operation_id = $1 AND action = 'node.revoke_isolation_resolved' AND result = 'succeeded' AND after_diff->>'resolution' = 'conntrack_delete_succeeded'`, revokeOpID).Scan(&resolvedAuditCount); err != nil || resolvedAuditCount != 1 {
			t.Fatalf("node.revoke_isolation_resolved audit count = %d (err=%v), want 1", resolvedAuditCount, err)
		}
		if auditStore.AnomalyCount() != 0 {
			t.Fatalf("auditStore.AnomalyCount() = %d, want 0", auditStore.AnomalyCount())
		}
	})

	for _, failClosedCase := range []struct {
		name   string
		detail string
	}{
		{name: "retry_bound_exceeded", detail: "Conntrack delete exceeded the retry bound."},
		{name: "conntrack_not_installed", detail: "Conntrack is not installed."},
	} {
		t.Run("member_suspend_and_grant_revoke_complete_when_node_fails_closed_"+failClosedCase.name, func(t *testing.T) {
			r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
			ctx := t.Context()
			now := report.ObservedAt.UTC()

			if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
				t.Fatalf("SubmitObservation baseline: %v", err)
			}

			adminID := uuid.New()
			var grp api.MemberGroup
			if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
				var gErr error
				grp, gErr = r.Nodes.CreateGroupTx(ctx, tx, "admin:"+adminID.String(), nil, api.MemberGroupCreate{
					Code: "grp-m3-fc",
					Name: "M3 FailClosed Group",
				})
				return gErr
			}); err != nil {
				t.Fatalf("CreateGroupTx: %v", err)
			}

			compiler := r.DesiredStateCompiler()
			exitsSvc := exits.NewService(r.Pool, compiler, nil, exits.WithClock(func() time.Time { return now }))
			_, etag, err := r.Nodes.GetNode(ctx, report.NodeId)
			if err != nil {
				t.Fatalf("GetNode: %v", err)
			}
			groupIDs := []uuid.UUID{grp.GroupId}
			if _, _, err := exitsSvc.SetPublication(ctx, exits.SetPublicationParams{
				NodeID:         report.NodeId,
				Principal:      "admin:" + adminID.String(),
				IdempotencyKey: uuid.New(),
				IfMatch:        etag,
				Request: api.PublicationRequest{
					PublicationIntent: api.PublicationIntentPublished,
					Reason:            "TEST ONLY M-3 fail-closed test",
					GroupIds:          &groupIDs,
				},
			}); err != nil {
				t.Fatalf("SetPublication: %v", err)
			}

			memberID := uuid.New()
			planID := uuid.New()
			planRevID := uuid.New()
			if _, err := r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
				t.Fatalf("insert member: %v", err)
			}
			if _, err := r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, 'plan-m3-fc')`, planID); err != nil {
				t.Fatalf("insert plan: %v", err)
			}
			if _, err := r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, []uuid.UUID{grp.GroupId}); err != nil {
				t.Fatalf("insert plan_revision: %v", err)
			}

			entStore, err := entitlement.NewStore(entitlement.TestConfig())
			if err != nil {
				t.Fatalf("entitlement.NewStore: %v", err)
			}
			entStore.Now = func() time.Time { return now }
			membersSvc := &members.Service{Pool: r.Pool, Entitlements: entStore}

			validFrom := now.Add(-time.Hour)
			validUntil := now.Add(12 * time.Hour)
			grantRes, err := membersSvc.Grant(ctx, entitlement.Command{
				Actor: entitlement.Actor{Role: "admin", ID: adminID, RequestID: uuid.New()},
				Key:   uuid.New(),
			}, memberID, api.GrantCreateRequest{
				Source:         api.GrantSourceAdminGrant,
				ValidFrom:      &validFrom,
				ValidUntil:     validUntil,
				Reason:         "TEST ONLY M-3 grant",
				PlanRevisionId: &planRevID,
			})
			if err != nil {
				t.Fatalf("members.Grant: %v", err)
			}
			grantID := grantRes.ResourceID

			compInitial, err := compiler.Compile(ctx, report.NodeId)
			if err != nil {
				t.Fatalf("Compile initial: %v", err)
			}
			now = now.Add(time.Second)
			initialReady := buildAgentObservationReport(report, compInitial.Envelope.Signed, now, false, false, "")
			if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, initialReady); err != nil {
				t.Fatalf("SubmitObservation initial ready: %v", err)
			}

			// Revoke grant and suspend member.
			if err := membersSvc.RevokeGrant(ctx, entitlement.Command{
				Actor: entitlement.Actor{Role: "admin", ID: adminID, RequestID: uuid.New()},
				Key:   uuid.New(),
			}, memberID, grantID, "TEST ONLY revoke grant for M-3"); err != nil {
				t.Fatalf("RevokeGrant: %v", err)
			}
			var grantRevokeOpID uuid.UUID
			if err := r.Pool.QueryRow(ctx, `SELECT id FROM operations WHERE resource_type = 'entitlement_grant' AND resource_id = $1 AND kind = 'member.grant_revoke'`, grantID).Scan(&grantRevokeOpID); err != nil {
				t.Fatalf("select grant_revoke op: %v", err)
			}

			var memberVer decstr.U64
			if err := r.Pool.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id = $1`, memberID).Scan(&memberVer); err != nil {
				t.Fatalf("select member version: %v", err)
			}
			suspended := true
			suspReason := "TEST ONLY suspend for M-3"
			suspRes, err := membersSvc.Patch(ctx, entitlement.Command{
				Actor: entitlement.Actor{Role: "admin", ID: adminID, RequestID: uuid.New()},
				Key:   uuid.New(),
			}, memberID, &memberVer, api.MemberPatch{
				AdminSuspended:   &suspended,
				SuspensionReason: &suspReason,
			})
			if err != nil || suspRes.OperationID == nil {
				t.Fatalf("members.Patch suspend: %v", err)
			}
			suspendOpID := *suspRes.OperationID

			for _, opID := range []uuid.UUID{grantRevokeOpID, suspendOpID} {
				if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
					var auditID uuid.UUID
					if err := tx.QueryRow(ctx, `SELECT id FROM audit_events LIMIT 1`).Scan(&auditID); err != nil {
						return err
					}
					return entitlement.TransitionOperation(ctx, tx, opID, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{
						Reference:  "audit_event:" + auditID.String(),
						ObservedAt: now,
					})
				}); err != nil {
					t.Fatalf("TransitionOperation netbird (%s): %v", opID, err)
				}
			}

			compRevoke, err := compiler.Compile(ctx, report.NodeId)
			if err != nil {
				t.Fatalf("Compile after suspend: %v", err)
			}

			// Observation 1: pending ISOLATION_POLICY_INVALID apply error built from agent.Decide holds both operations in running / nodes=pending.
			now = now.Add(time.Second)
			pendingReport := buildAgentObservationReport(report, compRevoke.Envelope.Signed, now, true, false, "")
			if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, pendingReport); err != nil {
				t.Fatalf("SubmitObservation pending isolation error: %v", err)
			}

			for _, opID := range []uuid.UUID{grantRevokeOpID, suspendOpID} {
				op, err := r.Nodes.GetOperation(ctx, opID)
				if err != nil {
					t.Fatalf("GetOperation(%s): %v", opID, err)
				}
				if op.Status != api.OperationStatusRunning {
					t.Fatalf("op %s Status = %s, want running while conntrack delete is pending", op.Kind, op.Status)
				}
			}

			// Observation 2: real fail-closed shape built from agent.Decide (ConntrackBlocked=true, internet_only=failed with ISOLATION_POLICY_INVALID).
			now = now.Add(time.Second)
			failClosedReport := buildAgentObservationReport(report, compRevoke.Envelope.Signed, now, false, true, failClosedCase.detail)
			if failClosedReport.MemberForwardingEnabled || failClosedReport.SafetyStatus != api.SafetyStatusFailed {
				t.Fatalf("failClosedReport = (forwarding=%v, safety=%s), want (false, failed)", failClosedReport.MemberForwardingEnabled, failClosedReport.SafetyStatus)
			}
			if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, failClosedReport); err != nil {
				t.Fatalf("SubmitObservation fail-closed: %v", err)
			}

			// Node is now fail-closed (not publishable / not forwarding), and both revoke operations complete with resolution=node_failed_closed.
			detailFC, _, err := r.Nodes.GetNode(ctx, report.NodeId)
			if err != nil {
				t.Fatalf("GetNode after fail-closed: %v", err)
			}
			fwdFC, _ := detailFC.MemberForwardingObserved.Get()
			if detailFC.ProvisioningState != api.ProvisioningStateApplying || detailFC.SafetyStatus != api.SafetyStatusFailed || fwdFC {
				t.Fatalf("node state after fail-closed = (%s, %s, forwarding=%v), want (applying, failed, false)", detailFC.ProvisioningState, detailFC.SafetyStatus, fwdFC)
			}

			for _, opID := range []uuid.UUID{grantRevokeOpID, suspendOpID} {
				op, err := r.Nodes.GetOperation(ctx, opID)
				if err != nil {
					t.Fatalf("GetOperation(%s) after fail-closed: %v", opID, err)
				}
				if op.Status != api.OperationStatusSucceeded {
					t.Fatalf("op %s Status after fail-closed = %s, want succeeded", op.Kind, op.Status)
				}
				var count int
				if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE operation_id = $1 AND action = 'node.revoke_isolation_resolved' AND result = 'succeeded' AND after_diff->>'resolution' = 'node_failed_closed'`, opID).Scan(&count); err != nil || count != 1 {
					t.Fatalf("op %s node_failed_closed audit count = %d (err=%v), want 1", op.Kind, count, err)
				}
			}
		})
	}

	t.Run("second_node_in_exit_group_still_pending_holds_operation_until_both_resolve", func(t *testing.T) {
		r, _, report1 := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
		ctx := t.Context()
		now := report1.ObservedAt.UTC()

		if _, err := r.Nodes.SubmitObservation(ctx, report1.NodeId, report1.NodeIncarnationId, report1); err != nil {
			t.Fatalf("SubmitObservation node1 baseline: %v", err)
		}

		// Enroll a second node in the same runtime and bring it to baseline ready.
		adminID := uuid.New()
		created2, err := r.Enrollment.CreateEnrollment(ctx, enrollment.Command{Principal: "admin:" + adminID.String(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller})
		if err != nil {
			t.Fatalf("CreateEnrollment node2: %v", err)
		}
		tok2, err := created2.Token.Get()
		if err != nil {
			t.Fatalf("Token.Get node2: %v", err)
		}
		req2, _ := pkitest.Request(t)
		enrolled2, err := r.Enrollment.Enroll(ctx, tok2, req2)
		if err != nil {
			t.Fatalf("Enroll node2: %v", err)
		}
		compiler := r.DesiredStateCompiler()
		compBase2, err := compiler.Compile(ctx, enrolled2.Response.NodeId)
		if err != nil {
			t.Fatalf("Compile node2 baseline: %v", err)
		}
		report2 := report1
		report2.NodeId = enrolled2.Response.NodeId
		report2.NodeIncarnationId = enrolled2.Response.NodeIncarnationId
		report2.ReportId = uuid.New()
		report2.AppliedConfigGeneration.Set(compBase2.Envelope.Signed.ConfigGeneration)
		if _, err := r.Nodes.SubmitObservation(ctx, report2.NodeId, report2.NodeIncarnationId, report2); err != nil {
			t.Fatalf("SubmitObservation node2 baseline: %v", err)
		}

		var grp api.MemberGroup
		if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var gErr error
			grp, gErr = r.Nodes.CreateGroupTx(ctx, tx, "admin:"+adminID.String(), nil, api.MemberGroupCreate{
				Code: "grp-m3-two-nodes",
				Name: "M3 Two Nodes Group",
			})
			return gErr
		}); err != nil {
			t.Fatalf("CreateGroupTx: %v", err)
		}

		exitsSvc := exits.NewService(r.Pool, compiler, nil, exits.WithClock(func() time.Time { return now }))
		groupIDs := []uuid.UUID{grp.GroupId}
		for _, nID := range []uuid.UUID{report1.NodeId, report2.NodeId} {
			_, etag, err := r.Nodes.GetNode(ctx, nID)
			if err != nil {
				t.Fatalf("GetNode(%s): %v", nID, err)
			}
			if _, _, err := exitsSvc.SetPublication(ctx, exits.SetPublicationParams{
				NodeID:         nID,
				Principal:      "admin:" + adminID.String(),
				IdempotencyKey: uuid.New(),
				IfMatch:        etag,
				Request: api.PublicationRequest{
					PublicationIntent: api.PublicationIntentPublished,
					Reason:            "TEST ONLY M-3 two-node test",
					GroupIds:          &groupIDs,
				},
			}); err != nil {
				t.Fatalf("SetPublication(%s): %v", nID, err)
			}
		}

		memberID := uuid.New()
		identityID := uuid.New()
		planID := uuid.New()
		planRevID := uuid.New()
		if _, err := r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
			t.Fatalf("insert member: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO identity_bindings (id, member_id, issuer, client_scope, subject, stable_claim_ref, binding_method, verified_at) VALUES ($1, $2, 'https://id.example.invalid', 'portal', 'TEST ONLY m3 user 2n', 'sub', 'oidc', $3)`, identityID, memberID, now); err != nil {
			t.Fatalf("insert identity_binding: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO netbird_user_bindings (id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at) VALUES ($1, $2, $3, 'TEST ONLY m3 user 2n', $4, 'shared_issuer_public_subject:https://id.example.invalid', 1, $5)`, uuid.New(), memberID, r.Config.NetBirdDeployment, identityID, now); err != nil {
			t.Fatalf("insert netbird_user_binding: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, 'plan-m3-2n')`, planID); err != nil {
			t.Fatalf("insert plan: %v", err)
		}
		if _, err := r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, []uuid.UUID{grp.GroupId}); err != nil {
			t.Fatalf("insert plan_revision: %v", err)
		}

		entStore, err := entitlement.NewStore(entitlement.TestConfig())
		if err != nil {
			t.Fatalf("entitlement.NewStore: %v", err)
		}
		entStore.Now = func() time.Time { return now }
		membersSvc := &members.Service{Pool: r.Pool, Entitlements: entStore}
		devicesSvc := &devices.Service{Pool: r.Pool, Entitlements: entStore}

		validFrom := now.Add(-time.Hour)
		validUntil := now.Add(12 * time.Hour)
		if _, err := membersSvc.Grant(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "admin", ID: adminID, RequestID: uuid.New()},
			Key:   uuid.New(),
		}, memberID, api.GrantCreateRequest{
			Source:         api.GrantSourceAdminGrant,
			ValidFrom:      &validFrom,
			ValidUntil:     validUntil,
			Reason:         "TEST ONLY M-3 two-node grant",
			PlanRevisionId: &planRevID,
		}); err != nil {
			t.Fatalf("members.Grant: %v", err)
		}

		devOut1, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()},
			Key:   uuid.New(),
		}, devices.Peer{
			DeploymentID:   r.Config.NetBirdDeployment,
			ID:             "TEST ONLY m3 2n peer 1",
			ExternalUserID: "TEST ONLY m3 user 2n",
			OverlayScope:   "TEST ONLY",
			OverlayIP:      netip.MustParseAddr("100.64.0.31"),
			Platform:       "linux",
		})
		if err != nil {
			t.Fatalf("AuthorizeDevice 1: %v", err)
		}
		if _, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()},
			Key:   uuid.New(),
		}, devices.Peer{
			DeploymentID:   r.Config.NetBirdDeployment,
			ID:             "TEST ONLY m3 2n peer 2",
			ExternalUserID: "TEST ONLY m3 user 2n",
			OverlayScope:   "TEST ONLY",
			OverlayIP:      netip.MustParseAddr("100.64.0.32"),
			Platform:       "linux",
		}); err != nil {
			t.Fatalf("AuthorizeDevice 2: %v", err)
		}

		compAuth1, err := compiler.Compile(ctx, report1.NodeId)
		if err != nil {
			t.Fatalf("Compile node1 auth: %v", err)
		}
		compAuth2, err := compiler.Compile(ctx, report2.NodeId)
		if err != nil {
			t.Fatalf("Compile node2 auth: %v", err)
		}
		now = now.Add(time.Second)
		if _, err := r.Nodes.SubmitObservation(ctx, report1.NodeId, report1.NodeIncarnationId, buildAgentObservationReport(report1, compAuth1.Envelope.Signed, now, false, false, "")); err != nil {
			t.Fatalf("SubmitObservation node1 auth: %v", err)
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report2.NodeId, report2.NodeIncarnationId, buildAgentObservationReport(report2, compAuth2.Envelope.Signed, now, false, false, "")); err != nil {
			t.Fatalf("SubmitObservation node2 auth: %v", err)
		}

		// Revoke device 1 and transition netbird step to succeeded.
		revokeOut, err := devicesSvc.RevokeDevice(ctx, entitlement.Command{
			Actor: entitlement.Actor{Role: "member", ID: memberID, RequestID: uuid.New()},
			Key:   uuid.New(),
		}, devOut1.DeviceID)
		if err != nil || revokeOut.OperationID == nil {
			t.Fatalf("RevokeDevice: %v", err)
		}
		revokeOpID := *revokeOut.OperationID
		if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var auditID uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT id FROM audit_events LIMIT 1`).Scan(&auditID); err != nil {
				return err
			}
			return entitlement.TransitionOperation(ctx, tx, revokeOpID, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{
				Reference:  "audit_event:" + auditID.String(),
				ObservedAt: now,
			})
		}); err != nil {
			t.Fatalf("TransitionOperation netbird: %v", err)
		}

		compRev1, err := compiler.Compile(ctx, report1.NodeId)
		if err != nil {
			t.Fatalf("Compile node1 revoke: %v", err)
		}
		compRev2, err := compiler.Compile(ctx, report2.NodeId)
		if err != nil {
			t.Fatalf("Compile node2 revoke: %v", err)
		}

		// Both nodes report pending conntrack delete under the bound.
		now = now.Add(time.Second)
		if _, err := r.Nodes.SubmitObservation(ctx, report1.NodeId, report1.NodeIncarnationId, buildAgentObservationReport(report1, compRev1.Envelope.Signed, now, true, false, "")); err != nil {
			t.Fatalf("SubmitObservation node1 pending: %v", err)
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report2.NodeId, report2.NodeIncarnationId, buildAgentObservationReport(report2, compRev2.Envelope.Signed, now, true, false, "")); err != nil {
			t.Fatalf("SubmitObservation node2 pending: %v", err)
		}

		// Node 1 resolves cleanly while Node 2 in the same exit group is still pending -> operation must remain held!
		now = now.Add(time.Second)
		if _, err := r.Nodes.SubmitObservation(ctx, report1.NodeId, report1.NodeIncarnationId, buildAgentObservationReport(report1, compRev1.Envelope.Signed, now, false, false, "")); err != nil {
			t.Fatalf("SubmitObservation node1 clean while node2 pending: %v", err)
		}
		opWhileNode2Pending, err := r.Nodes.GetOperation(ctx, revokeOpID)
		if err != nil {
			t.Fatalf("GetOperation while node2 pending: %v", err)
		}
		if opWhileNode2Pending.Status != api.OperationStatusRunning {
			t.Fatalf("operation status while node2 still pending = %s, want running (held)", opWhileNode2Pending.Status)
		}
		for _, st := range opWhileNode2Pending.Steps {
			if st.Name == "nodes" && st.Status != api.OperationStepStatusPending {
				t.Fatalf("nodes step while node2 still pending = %s, want pending", st.Status)
			}
		}

		// Node 2 now resolves (fails closed on retry bound) -> operation completes as succeeded.
		now = now.Add(time.Second)
		if _, err := r.Nodes.SubmitObservation(ctx, report2.NodeId, report2.NodeIncarnationId, buildAgentObservationReport(report2, compRev2.Envelope.Signed, now, false, true, "Conntrack delete exceeded the retry bound.")); err != nil {
			t.Fatalf("SubmitObservation node2 fail-closed: %v", err)
		}
		opAfterBoth, err := r.Nodes.GetOperation(ctx, revokeOpID)
		if err != nil {
			t.Fatalf("GetOperation after both nodes resolved: %v", err)
		}
		if opAfterBoth.Status != api.OperationStatusSucceeded {
			t.Fatalf("operation status after both nodes resolved = %s, want succeeded", opAfterBoth.Status)
		}
	})
}

func TestReadiness_PreservedAcrossConcurrentDesiredGenerationAdvance(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	ctx := t.Context()

	// 1. Node achieves initial baseline readiness on generation 1.
	if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatalf("SubmitObservation gen 1: %v", err)
	}
	detail1, _, err := r.Nodes.GetNode(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("GetNode gen 1: %v", err)
	}
	if detail1.ProvisioningState != api.ProvisioningStateReady || detail1.SafetyStatus != api.SafetyStatusPassed {
		t.Fatalf("initial state = (%s, %s), want (ready, passed)", detail1.ProvisioningState, detail1.SafetyStatus)
	}

	// 2. Central advances desired_config_generation twice (e.g. gen 2 and gen 3) while the node's
	// observation for gen 2 is in flight.
	compiler := r.DesiredStateCompiler()
	comp2, err := compiler.Compile(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("Compile gen 2: %v", err)
	}
	comp3, err := compiler.Compile(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("Compile gen 3: %v", err)
	}
	gen2 := comp2.Envelope.Signed.ConfigGeneration
	gen3 := comp3.Envelope.Signed.ConfigGeneration
	if gen3 <= gen2 {
		t.Fatalf("expected gen3 (%d) > gen2 (%d)", gen3, gen2)
	}

	// 3. In-flight observation for gen 2 with SafetyStatus=passed and ProvisioningState=ready arrives
	// while desired_config_generation is already gen 3.
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(time.Second)
	report.AppliedConfigGeneration.Set(gen2)
	for i := range report.Checks {
		report.Checks[i].ObservedAt = report.ObservedAt
	}
	if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatalf("SubmitObservation in-flight gen 2: %v", err)
	}

	// 4. Verify ProvisioningState remains 'ready' (not downgraded to 'applying'), while generation lag
	// is still accurately reflected by AppliedConfigGeneration (gen 2) < DesiredConfigGeneration (gen 3).
	detail2, _, err := r.Nodes.GetNode(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("GetNode after in-flight gen 2: %v", err)
	}
	applied2, _ := detail2.ConfigGeneration.Applied.Get()
	if detail2.ProvisioningState != api.ProvisioningStateReady {
		t.Fatalf("ProvisioningState downgraded to %s on concurrent generation bump, want ready", detail2.ProvisioningState)
	}
	if applied2 != gen2 || detail2.ConfigGeneration.Desired != gen3 {
		t.Fatalf("ConfigGeneration = (applied=%d, desired=%d), want (%d, %d)", applied2, detail2.ConfigGeneration.Desired, gen2, gen3)
	}

	// Ensure unused imports are referenced cleanly if needed.
	_ = context.Background
	_ = json.Marshal
}

func TestD067_AwaitingMemberGrantsReadModelAndOperationProjection(t *testing.T) {
	r, _, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.awaiting-member-grants.json")
	ctx := t.Context()

	// Restore the awaiting_member_grants example posture on the enrolled node's baseline report.
	leaseID := uuid.New()
	report.ProvisioningState = api.ProvisioningStateReady
	report.SafetyStatus = api.SafetyStatusPassed
	report.DataHealth = api.DataHealthHealthy
	report.MemberForwardingEnabled = false
	report.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonAwaitingMemberGrants}
	report.Lease.LeaseId.Set(leaseID)
	report.Lease.ExpiresAt.Set(report.ObservedAt.Add(3 * time.Minute))
	report.Checks = nil
	for _, name := range []api.NodeCheckName{
		api.NodeCheckNameInternetOnly,
		api.NodeCheckNameMeterPath,
		api.NodeCheckNameTunnelProbe,
		api.NodeCheckNameLease,
		api.NodeCheckNameQos,
		api.NodeCheckNameSourceBinding,
	} {
		status := api.HealthCheckStatusNotRun
		if name == api.NodeCheckNameInternetOnly || name == api.NodeCheckNameLease {
			status = api.HealthCheckStatusPassed
		}
		ch := api.NodeCheck{Name: name, Status: status, ObservedAt: report.ObservedAt}
		ch.ErrorCode.SetNull()
		report.Checks = append(report.Checks, ch)
	}

	// 1. Unknown block reason fails closed with 422 VALIDATION_FAILED.
	badReport := report
	badReport.ReportId = uuid.New()
	badReport.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReason("invalid_block_reason")}
	if _, err := r.Nodes.SubmitObservation(ctx, badReport.NodeId, badReport.NodeIncarnationId, badReport); err == nil {
		t.Fatal("expected invalid forwarding_block_reason to be rejected")
	}

	// 2. awaiting_member_grants is accepted and projected as healthy and provisional (not expired),
	// with member_forwarding_observed=false and publication state unchanged.
	if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatalf("SubmitObservation awaiting_member_grants: %v", err)
	}

	detail, _, err := r.Nodes.GetNode(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if detail.ProvisioningState != api.ProvisioningStateReady {
		t.Fatalf("ProvisioningState = %q, want ready", detail.ProvisioningState)
	}
	if detail.SafetyStatus != api.SafetyStatusPassed {
		t.Fatalf("SafetyStatus = %q, want passed (not expired)", detail.SafetyStatus)
	}
	if detail.DataHealth != api.DataHealthHealthy {
		t.Fatalf("DataHealth = %q, want healthy", detail.DataHealth)
	}
	fwd, err := detail.MemberForwardingObserved.Get()
	if err != nil || fwd {
		t.Fatalf("MemberForwardingObserved = (%v, %v), want (false, nil)", fwd, err)
	}
	if detail.PublicationIntent != api.PublicationIntentUnpublished || detail.PublicationObserved != api.PublicationObservedAbsent {
		t.Fatalf("publication = (%q, %q), want (unpublished, absent)", detail.PublicationIntent, detail.PublicationObserved)
	}

	list, err := r.Nodes.ListNodes(ctx, nodes.ListNodesParams{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("ListNodes: items=%d err=%v", len(list.Items), err)
	}
	item := list.Items[0]
	if item.SafetyStatus != api.SafetyStatusPassed || item.DataHealth != api.DataHealthHealthy || item.ProvisioningState != api.ProvisioningStateReady {
		t.Fatalf("ListNodes summary = (%s, %s, %s), want (passed, healthy, ready)", item.SafetyStatus, item.DataHealth, item.ProvisioningState)
	}
	itemFwd, err := item.MemberForwardingObserved.Get()
	if err != nil || itemFwd {
		t.Fatalf("ListNodes MemberForwardingObserved = (%v, %v), want (false, nil)", itemFwd, err)
	}

	// Verify node_observations.report_payload retains forwarding_block_reasons: ["awaiting_member_grants"]
	// and lease.member_grant_count == 0.
	var storedPayload []byte
	if err := r.Pool.QueryRow(ctx, `SELECT report_payload FROM node_observations WHERE report_id = $1`, report.ReportId).Scan(&storedPayload); err != nil {
		t.Fatalf("query node_observations.report_payload: %v", err)
	}
	var storedReport api.ObservationReport
	if err := json.Unmarshal(storedPayload, &storedReport); err != nil {
		t.Fatalf("unmarshal stored report_payload: %v", err)
	}
	if len(storedReport.ForwardingBlockReasons) != 1 || storedReport.ForwardingBlockReasons[0] != api.ForwardingBlockReasonAwaitingMemberGrants {
		t.Fatalf("stored ForwardingBlockReasons = %v, want [awaiting_member_grants]", storedReport.ForwardingBlockReasons)
	}
	if storedLeaseID, err := storedReport.Lease.LeaseId.Get(); err != nil || storedLeaseID != leaseID {
		t.Fatalf("stored Lease.LeaseId = (%v, %v), want %v", storedLeaseID, err, leaseID)
	}

	// 3. Verify operation completion and publication gate behavior under awaiting_member_grants:
	// Publish the node with a member group, grant a member, and authorize a device so bindings are desired.
	now := report.ObservedAt.UTC()
	adminID := uuid.New()
	var grp api.MemberGroup
	if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var gErr error
		grp, gErr = r.Nodes.CreateGroupTx(ctx, tx, "admin:"+adminID.String(), nil, api.MemberGroupCreate{
			Code: "grp-d067",
			Name: "D067 Group",
		})
		return gErr
	}); err != nil {
		t.Fatalf("CreateGroupTx: %v", err)
	}

	compiler := r.DesiredStateCompiler()
	exitsSvc := exits.NewService(r.Pool, compiler, nil, exits.WithClock(func() time.Time { return now }))
	_, etag, err := r.Nodes.GetNode(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("GetNode for publish: %v", err)
	}
	groupIDs := []uuid.UUID{grp.GroupId}
	pubOp, _, err := exitsSvc.SetPublication(ctx, exits.SetPublicationParams{
		NodeID:         report.NodeId,
		Principal:      "admin:" + adminID.String(),
		IdempotencyKey: uuid.New(),
		IfMatch:        etag,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            "TEST ONLY D-067 publish",
			GroupIds:          &groupIDs,
		},
	})
	if err != nil {
		t.Fatalf("SetPublication: %v", err)
	}

	memberID := uuid.New()
	identityID := uuid.New()
	planID := uuid.New()
	planRevID := uuid.New()
	if _, err := r.Pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := r.Pool.Exec(ctx, `INSERT INTO identity_bindings (id, member_id, issuer, client_scope, subject, stable_claim_ref, binding_method, verified_at) VALUES ($1, $2, 'https://id.example.invalid', 'portal', 'TEST ONLY d067 user', 'sub', 'oidc', $3)`, identityID, memberID, now); err != nil {
		t.Fatalf("insert identity_binding: %v", err)
	}
	if _, err := r.Pool.Exec(ctx, `INSERT INTO netbird_user_bindings (id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at) VALUES ($1, $2, $3, 'TEST ONLY d067 user', $4, 'shared_issuer_public_subject:https://id.example.invalid', 1, $5)`, uuid.New(), memberID, r.Config.NetBirdDeployment, identityID, now); err != nil {
		t.Fatalf("insert netbird_user_binding: %v", err)
	}
	if _, err := r.Pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, 'plan-d067')`, planID); err != nil {
		t.Fatalf("insert plan: %v", err)
	}
	if _, err := r.Pool.Exec(ctx, `INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload) VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRevID, planID, []uuid.UUID{grp.GroupId}); err != nil {
		t.Fatalf("insert plan_revision: %v", err)
	}

	entStore, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatalf("entitlement.NewStore: %v", err)
	}
	entStore.Now = func() time.Time { return now }
	membersSvc := &members.Service{Pool: r.Pool, Entitlements: entStore}
	devicesSvc := &devices.Service{Pool: r.Pool, Entitlements: entStore}

	validFrom := now.Add(-time.Hour)
	validUntil := now.Add(12 * time.Hour)
	if _, err := membersSvc.Grant(ctx, entitlement.Command{
		Actor: entitlement.Actor{Role: "admin", ID: adminID, RequestID: uuid.New()},
		Key:   uuid.New(),
	}, memberID, api.GrantCreateRequest{
		Source:         api.GrantSourceAdminGrant,
		ValidFrom:      &validFrom,
		ValidUntil:     validUntil,
		Reason:         "TEST ONLY D-067 grant",
		PlanRevisionId: &planRevID,
	}); err != nil {
		t.Fatalf("members.Grant: %v", err)
	}
	devOut, err := devicesSvc.AuthorizeDevice(ctx, entitlement.Command{
		Actor: entitlement.Actor{Role: "system", ID: uuid.New(), RequestID: uuid.New()},
		Key:   uuid.New(),
	}, devices.Peer{
		DeploymentID:   r.Config.NetBirdDeployment,
		ID:             "TEST ONLY d067 peer",
		ExternalUserID: "TEST ONLY d067 user",
		OverlayScope:   "TEST ONLY",
		OverlayIP:      netip.MustParseAddr("100.64.0.77"),
		Platform:       "linux",
	})
	if err != nil || devOut.OperationID == nil {
		t.Fatalf("AuthorizeDevice: %v", err)
	}
	if err := db.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var auditID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM audit_events LIMIT 1`).Scan(&auditID); err != nil {
			return err
		}
		return entitlement.TransitionOperation(ctx, tx, *devOut.OperationID, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{
			Reference:  "audit_event:" + auditID.String(),
			ObservedAt: now,
		})
	}); err != nil {
		t.Fatalf("TransitionOperation netbird: %v", err)
	}

	comp, err := compiler.Compile(ctx, report.NodeId)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	report.ReportId = uuid.New()
	report.ObservedAt = now.Add(time.Second)
	report.AppliedConfigGeneration.Set(comp.Envelope.Signed.ConfigGeneration)
	for i := range report.Checks {
		report.Checks[i].ObservedAt = report.ObservedAt
	}
	if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatalf("SubmitObservation post-compile awaiting_member_grants: %v", err)
	}

	// Verify node.publish operation: agent_policy succeeded, data_path_probe remains pending (forwarding_disabled).
	pubOpAfter, err := r.Nodes.GetOperation(ctx, pubOp.OperationId)
	if err != nil {
		t.Fatalf("GetOperation publish: %v", err)
	}
	var gotAgentPolicy, gotDataProbe api.OperationStepStatus
	for _, st := range pubOpAfter.Steps {
		switch st.Name {
		case nodes.StepAgentPolicy:
			gotAgentPolicy = st.Status
		case nodes.StepDataPathProbe:
			gotDataProbe = st.Status
		}
	}
	if gotAgentPolicy != api.OperationStepStatusSucceeded || gotDataProbe != api.OperationStepStatusPending {
		t.Fatalf("publish steps under awaiting_member_grants = (agent_policy=%s, data_path_probe=%s), want (succeeded, pending)", gotAgentPolicy, gotDataProbe)
	}

	// Verify member_device operation (follow-up 18): nodes step advances to succeeded because node applied generation with passed safety.
	devOpAfter, err := r.Nodes.GetOperation(ctx, *devOut.OperationID)
	if err != nil {
		t.Fatalf("GetOperation authorize_device: %v", err)
	}
	if devOpAfter.Status != api.OperationStatusSucceeded {
		t.Fatalf("authorize_device operation status under awaiting_member_grants = %s, want succeeded (steps=%+v)", devOpAfter.Status, devOpAfter.Steps)
	}
}

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
		st, ok := nodesStep(op)
		if !ok || op.Status != api.OperationStatusSucceeded || st.Status != api.OperationStepStatusSucceeded {
			memberOp, err := w.r.Nodes.GetMemberOperation(w.t.Context(), member.id, opID)
			if err != nil {
				t.Fatal(err)
			}
			t.Fatalf("member_device.revoke %s for a device bound only to %s stayed %s (%s); member read %s; unpublished %s has no exit_groups and a pending delete while forwarding", opID, w.primary.NodeId, op.Status, stepText(st), memberOp.Status, hostage.NodeId)
		}
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
		st, ok := nodesStep(op)
		if !ok || op.Status != api.OperationStatusSucceeded || st.Status != api.OperationStepStatusSucceeded {
			t.Fatalf("peer %s last reported at %s and then stayed silent; an hour later member_device.revoke %s is still %s (%s)", peer.NodeId, peerSeen.UTC().Format(time.RFC3339), opID, op.Status, stepText(st))
		}
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
		if detail.ProvisioningState != api.ProvisioningStateApplying || applied >= detail.ConfigGeneration.Desired {
			t.Fatalf("second safety=passed report of applied generation %s kept provisioning_state=%s safety=%s while desired is %s (compile had reached %s)", applied, detail.ProvisioningState, detail.SafetyStatus, detail.ConfigGeneration.Desired, desired)
		}
	})

	t.Run("decision_uses_incarnation_column_when_audit_event_absent", func(t *testing.T) {
		w := newR14World(t)
		ctx := w.t.Context()
		if _, err := w.compiler.Compile(ctx, w.primary.NodeId); err != nil {
			t.Fatal(err)
		}
		comp, err := w.compiler.Compile(ctx, w.primary.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		desired := comp.Envelope.Signed.ConfigGeneration
		first, _ := w.prepare(w.primary, false, false)
		baseline, err := w.primary.AppliedConfigGeneration.Get()
		if err != nil {
			t.Fatal(err)
		}
		first.AppliedConfigGeneration.Set(baseline)
		w.observe(first)

		var graceAt *time.Time
		if err := w.r.Pool.QueryRow(ctx, `SELECT stale_generation_grace_at FROM node_incarnations WHERE id=$1`, w.primary.NodeIncarnationId).Scan(&graceAt); err != nil {
			t.Fatal(err)
		}
		if graceAt == nil {
			t.Fatal("expected node_incarnations.stale_generation_grace_at to be set after first stale report")
		}

		// Delete the node.observation_applied audit rows for this node using the
		// owner retention setting so we prove the control-plane decision does not
		// consult audit_events (D-075).
		if err := db.InTx(ctx, w.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `SET LOCAL fvpn.retention_purge = 'on'`); err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `DELETE FROM audit_events WHERE resource = $1 AND action = 'node.observation_applied'`, "node:"+w.primary.NodeId.String())
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				t.Fatal("expected at least one node.observation_applied audit row to delete")
			}
			return nil
		}); err != nil {
			t.Fatalf("delete node.observation_applied audit rows: %v", err)
		}

		second, _ := w.prepare(w.primary, false, false)
		second.AppliedConfigGeneration.Set(baseline)
		w.observe(second)

		detail, _, err := w.r.Nodes.GetNode(ctx, w.primary.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		applied, _ := detail.ConfigGeneration.Applied.Get()
		if detail.ProvisioningState != api.ProvisioningStateApplying || applied >= detail.ConfigGeneration.Desired {
			t.Fatalf("with audit_events row deleted, second stale report kept provisioning_state=%s (applied=%s desired=%s, compile=%s); want applying", detail.ProvisioningState, applied, detail.ConfigGeneration.Desired, desired)
		}
		if err := w.r.Pool.QueryRow(ctx, `SELECT stale_generation_grace_at FROM node_incarnations WHERE id=$1`, w.primary.NodeIncarnationId).Scan(&graceAt); err != nil {
			t.Fatal(err)
		}
		if graceAt != nil {
			t.Fatalf("expected stale_generation_grace_at cleared to NULL after downgrade to applying, got %v", graceAt)
		}
	})
}
