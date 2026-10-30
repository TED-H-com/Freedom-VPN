//go:build integration

package nodes_test

import (
	"context"
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
			ID:             "TEST ONLY m3 peer",
			ExternalUserID: "TEST ONLY m3 user",
			OverlayScope:   "TEST ONLY",
			OverlayIP:      netip.MustParseAddr("100.64.0.21"),
			Platform:       "linux",
		})
		if err != nil {
			t.Fatalf("AuthorizeDevice: %v", err)
		}

		compAuth, err := compiler.Compile(ctx, report.NodeId)
		if err != nil {
			t.Fatalf("Compile after AuthorizeDevice: %v", err)
		}
		now = now.Add(time.Second)
		report.ReportId = uuid.New()
		report.ObservedAt = now
		report.AppliedConfigGeneration.Set(compAuth.Envelope.Signed.ConfigGeneration)
		report.MemberForwardingEnabled = true
		report.DataHealth = api.DataHealthHealthy
		report.SafetyStatus = api.SafetyStatusPassed
		report.ProvisioningState = api.ProvisioningStateReady
		for i := range report.Checks {
			report.Checks[i].Status = api.HealthCheckStatusPassed
			report.Checks[i].ObservedAt = now
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
			t.Fatalf("SubmitObservation auth ready: %v", err)
		}

		// Revoke device via devices.Service.RevokeDevice and complete the netbird step via entitlement.TransitionOperation.
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
		if len(compRevoke.Envelope.Signed.DeviceBindings) != 0 {
			t.Fatalf("DeviceBindings after revoke = %d, want 0", len(compRevoke.Envelope.Signed.DeviceBindings))
		}

		// Submit observation on revokeGen with pending ISOLATION_POLICY_INVALID apply error while forwarding is still enabled.
		now = now.Add(time.Second)
		report.ReportId = uuid.New()
		report.ObservedAt = now
		report.AppliedConfigGeneration.Set(revokeGen)
		report.MemberForwardingEnabled = true
		report.SafetyStatus = api.SafetyStatusPassed
		report.ProvisioningState = api.ProvisioningStateReady
		report.DataHealth = api.DataHealthHealthy
		report.Errors = []api.NodeError{{
			Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
			Stage:      "apply",
			Message:    "conntrack delete pending; serving ruleset verified",
			ObservedAt: now,
		}}
		for i := range report.Checks {
			report.Checks[i].ObservedAt = now
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
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
		cause, err := nodes.PublicationDataPathFailure(ctx, r.Pool, report, uint64(revokeGen), now)
		if err != nil || cause != "isolation_unconfirmed" {
			t.Fatalf("PublicationDataPathFailure = (%q, %v), want (isolation_unconfirmed, nil)", cause, err)
		}

		// Verify operation read models (Admin Store.Get, Member Store.Get, nodes.GetOperation, nodes.GetMemberOperation, nodes.ListMemberOperations)
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

		// Verify audit_events recorded node.revoke_isolation_pending and unreadable-anomaly count is 0.
		auditStore := &audit.Store{Pool: r.Pool}
		var pendingAuditCount int
		if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE operation_id = $1 AND action = 'node.revoke_isolation_pending' AND result = 'failed' AND after_diff->>'error_code' = 'ISOLATION_POLICY_INVALID' AND after_diff->>'reason' = 'conntrack_delete_pending'`, revokeOpID).Scan(&pendingAuditCount); err != nil || pendingAuditCount != 1 {
			t.Fatalf("node.revoke_isolation_pending audit count = %d (err=%v), want 1", pendingAuditCount, err)
		}

		// Next observation on revokeGen has no isolation apply error -> revoke operation completes.
		now = now.Add(time.Second)
		report.ReportId = uuid.New()
		report.ObservedAt = now
		report.Errors = []api.NodeError{}
		for i := range report.Checks {
			report.Checks[i].ObservedAt = now
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
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

	t.Run("member_suspend_and_grant_revoke_complete_when_node_fails_closed", func(t *testing.T) {
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
				Code: "grp-m3-failclosed",
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
		report.ReportId = uuid.New()
		report.ObservedAt = now
		report.AppliedConfigGeneration.Set(compInitial.Envelope.Signed.ConfigGeneration)
		report.MemberForwardingEnabled = true
		report.DataHealth = api.DataHealthHealthy
		report.SafetyStatus = api.SafetyStatusPassed
		report.ProvisioningState = api.ProvisioningStateReady
		for i := range report.Checks {
			report.Checks[i].Status = api.HealthCheckStatusPassed
			report.Checks[i].ObservedAt = now
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
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
		revokeGen := compRevoke.Envelope.Signed.ConfigGeneration

		// Observation 1: pending ISOLATION_POLICY_INVALID apply error holds both operations in running / nodes=pending.
		now = now.Add(time.Second)
		report.ReportId = uuid.New()
		report.ObservedAt = now
		report.AppliedConfigGeneration.Set(revokeGen)
		report.MemberForwardingEnabled = true
		report.SafetyStatus = api.SafetyStatusPassed
		report.ProvisioningState = api.ProvisioningStateReady
		report.DataHealth = api.DataHealthHealthy
		report.Errors = []api.NodeError{{
			Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
			Stage:      "apply",
			Message:    "conntrack delete pending; serving ruleset verified",
			ObservedAt: now,
		}}
		for i := range report.Checks {
			report.Checks[i].ObservedAt = now
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
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

		// Observation 2: node exhausts bounded retry and fails closed (MemberForwardingEnabled=false, SafetyStatus=failed, ProvisioningState=applying, internet_only=passed).
		now = now.Add(time.Second)
		report.ReportId = uuid.New()
		report.ObservedAt = now
		report.MemberForwardingEnabled = false
		report.SafetyStatus = api.SafetyStatusFailed
		report.ProvisioningState = api.ProvisioningStateApplying
		report.DataHealth = api.DataHealthDegraded
		report.Errors = []api.NodeError{{
			Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
			Stage:      "apply",
			Message:    "conntrack delete failed after the serving ruleset was verified",
			ObservedAt: now,
		}}
		for i := range report.Checks {
			report.Checks[i].ObservedAt = now
		}
		if _, err := r.Nodes.SubmitObservation(ctx, report.NodeId, report.NodeIncarnationId, report); err != nil {
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
