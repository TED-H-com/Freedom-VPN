//go:build integration

package exits

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TEST ONLY - NOT A SECRET
func testExitsCompiler(t *testing.T, pool *pgxpool.Pool, now func() time.Time) *desiredstate.Compiler {
	t.Helper()
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - exits-ds-key"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	signer := desiredstate.NewKeySigner(desiredstate.SigningKey{
		KeyID:      "ds-key-exits-1",
		Purpose:    envelope.PurposeDesiredState,
		PrivateKey: priv,
	})
	return desiredstate.NewCompiler(
		pool,
		signer,
		desiredstate.WithEntitlementReader(desiredstate.NewFakeEntitlementReader()),
		desiredstate.WithPolicyReader(desiredstate.NewFakePolicyReader(desiredstate.PolicySnapshot{})),
		desiredstate.WithClock(now),
	)
}

func seedExitTestNode(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	intent api.PublicationIntent,
	observed string,
) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	relID := uuid.New()
	manifestHash := sha256.Sum256([]byte("manifest-" + relID.String()))
	sig := make([]byte, 64)
	if _, err := pool.Exec(ctx, `
		INSERT INTO releases (
			id, version, release_sequence, manifest_schema_version,
			manifest, manifest_hash, signature, signing_key_ref, component_versions
		) VALUES (
			$1, $2, 1, 1,
			'{"schema_version":1}'::jsonb, $3, $4, 'rel-key-1', '{"fvpn_agent":"1.0.0"}'::jsonb
		)`, relID, "2026.10.29-"+relID.String()[:8], manifestHash[:], sig); err != nil {
		t.Fatalf("insert release: %v", err)
	}

	grpID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, $2, $3)`, grpID, "grp-"+grpID.String()[:8], "Group"); err != nil {
		t.Fatalf("insert group: %v", err)
	}

	nodeID := uuid.New()
	incID := uuid.New()
	exitID := uuid.New()
	fp := sha256.Sum256([]byte("pubkey-" + nodeID.String()))

	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes (
			id, lifecycle_state, provisioning_state,
			desired_config_generation, applied_config_generation,
			desired_release_id, current_release_id, last_good_release_id,
			control_status, data_health, safety_status,
			member_forwarding_observed
		) VALUES (
			$1, 'registered', 'ready',
			1, 1,
			$2, $2, $2,
			'online', 'healthy', 'passed',
			false
		)`, nodeID, relID); err != nil {
		t.Fatalf("insert node: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO node_incarnations (
			id, node_id, public_key_fingerprint, deployment_id, peer_id, os, os_version, arch
		) VALUES (
			$1, $2, $3, 'nb-main', $4, 'ubuntu', '24.04', 'amd64'
		)`, incID, nodeID, fp[:], "peer-"+nodeID.String()[:8]); err != nil {
		t.Fatalf("insert incarnation: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $1, resource_version = resource_version + 1 WHERE id = $2`, incID, nodeID); err != nil {
		t.Fatalf("update current_incarnation_id: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO exits (
			id, node_id, display_name, region, publication_intent, publication_observed
		) VALUES (
			$1, $2, 'tw-taipei-exit', 'tw-tpe', $3, $4
		)`, exitID, nodeID, string(intent), observed); err != nil {
		t.Fatalf("insert exit: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, grpID); err != nil {
		t.Fatalf("insert exit_group: %v", err)
	}
	return nodeID, incID, exitID, relID, grpID
}

func submitAckObservation(
	t *testing.T,
	ctx context.Context,
	nodesSvc *nodes.Service,
	pool *pgxpool.Pool,
	nodeID, incID, relID uuid.UUID,
	now time.Time,
	appliedGen uint64,
	forwardingEnabled bool,
) {
	t.Helper()
	gen := nodes.ParseU64(appliedGen)
	relCopy := relID
	rep := emptyObservationReport()
	rep.AgentVersion = "1.0.0"
	rep.BootId = uuid.New()
	rep.ReportId = uuid.New()
	rep.NodeId = nodeID
	rep.NodeIncarnationId = incID
	rep.ObservedAt = now
	rep.CurrentReleaseId.Set(relCopy)
	rep.LastGoodReleaseId.Set(relCopy)
	rep.AppliedConfigGeneration.Set(gen)
	rep.ProvisioningState = api.ProvisioningStateReady
	rep.SafetyStatus = api.SafetyStatusPassed
	rep.DataHealth = api.DataHealthHealthy
	rep.MemberForwardingEnabled = forwardingEnabled
	if forwardingEnabled {
		seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - publication lease"))
		signer := desiredstate.NewKeySigner(desiredstate.SigningKey{KeyID: "TEST-lease", Purpose: envelope.PurposeLease, PrivateKey: ed25519.NewKeyFromSeed(seed[:])})
		lease, err := leases.NewService(pool, signer, leases.WithClock(func() time.Time { return now })).Renew(ctx, nodeID, incID, api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: nodeID, NodeIncarnationId: incID, CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullableWithValue(gen)})
		if err != nil {
			t.Fatal(err)
		}
		rep.Lease.LeaseId.Set(lease.Signed.LeaseId)
		rep.Lease.ExpiresAt.Set(lease.Signed.ExpiresAt)
	}
	rep.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	rep.Checks = []api.NodeCheck{
		{Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusPassed, ObservedAt: now,
			ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
		{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: now,
			ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
	}
	for _, name := range []api.NodeCheckName{api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		rep.Checks = append(rep.Checks, api.NodeCheck{Name: name, Status: api.HealthCheckStatusPassed, ObservedAt: now, ErrorCode: presence.NewNullNullable[api.ErrorCode]()})
	}
	if _, err := nodesSvc.SubmitObservation(ctx, nodeID, incID, rep); err != nil {
		t.Fatalf("SubmitObservation(gen=%d, fwd=%v): %v", appliedGen, forwardingEnabled, err)
	}
}

func TestAT08_PublishHappyPathAndFailureRetryNoDuplicateRoute(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)

	nodesSvc := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now }))
	compiler := testExitsCompiler(t, pool, func() time.Time { return now })
	nb := NewFakeNetBirdAdapter()
	exitsSvc := NewService(pool, compiler, nb, WithClock(func() time.Time { return now }))

	nodeID, incID, exitID, relID, _ := seedExitTestNode(t, ctx, pool, api.PublicationIntentUnpublished, "absent")

	// 1. SetPublication(published) writes intent, resource_version, operation, and outbox in 1 tx.
	_, etag1, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	reason := "publish tw-taipei-exit"
	pubOp, etag2, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            reason,
		},
	})
	if err != nil {
		t.Fatalf("SetPublication(published): %v", err)
	}
	if etag2 == etag1 || string(pubOp.Kind) != nodes.OpKindPublish || pubOp.Status != api.OperationStatusPending {
		t.Fatalf("SetPublication result = (etag=%s, op=%+v), want new etag & pending node.publish", etag2, pubOp)
	}

	// 2. Gate branch 1 (D-074): ExecuteOperation before node ACKs the new desired-state generation.
	// Intent stays published, operation stays running with agent_policy pending (NODE_NOT_SERVICEABLE / stale_generation),
	// publication_observed stays applying, and no NetBird route is created yet.
	failedOp1, err := exitsSvc.ExecuteOperation(ctx, pubOp.OperationId)
	if err != nil {
		t.Fatalf("ExecuteOperation before node ACK: %v", err)
	}
	if failedOp1.Status != api.OperationStatusRunning || !failedOp1.Error.IsNull() {
		t.Fatalf("failedOp1 = (status=%s, err=%+v), want running with null error", failedOp1.Status, failedOp1.Error)
	}
	for _, st := range failedOp1.Steps {
		if st.Name == nodes.StepAgentPolicy {
			code, cErr := st.ErrorCode.Get()
			msg, mErr := st.Message.Get()
			if st.Status != api.OperationStepStatusPending || cErr != nil || code != api.ErrorCodeNODENOTSERVICEABLE || mErr != nil || msg != "member_forwarding_unconfirmed (stale_generation)" {
				t.Fatalf("agent_policy step = %+v, want pending with NODE_NOT_SERVICEABLE / member_forwarding_unconfirmed (stale_generation)", st)
			}
		}
	}
	if nb.RouteCount() != 0 {
		t.Fatalf("expected 0 NetBird routes before node policy ACK, got %d", nb.RouteCount())
	}
	detailAfterFail1, _, _ := nodesSvc.GetNode(ctx, nodeID)
	if detailAfterFail1.PublicationIntent != api.PublicationIntentPublished {
		t.Fatalf("PublicationIntent changed to %s after gated wait, must stay published", detailAfterFail1.PublicationIntent)
	}
	if detailAfterFail1.PublicationObserved != api.PublicationObservedAbsent {
		t.Fatalf("PublicationObserved = %s, want absent (unchanged before route read-back)", detailAfterFail1.PublicationObserved)
	}

	// 3. Failure branch 2: Node ACKs the compiled desired_config_generation, but NetBird fails.
	desiredGen := uint64(detailAfterFail1.ConfigGeneration.Desired)
	now = now.Add(5 * time.Second)
	submitAckObservation(t, ctx, nodesSvc, pool, nodeID, incID, relID, now, desiredGen, true)

	nb.SetPublishError(errors.New("netbird management API 503"))
	failedOp2, err := exitsSvc.ExecuteOperation(ctx, pubOp.OperationId)
	if err != nil {
		t.Fatalf("ExecuteOperation with NetBird error: %v", err)
	}
	if failedOp2.Status != api.OperationStatusFailed {
		t.Fatalf("failedOp2.Status = %s, want failed", failedOp2.Status)
	}

	// 4. Retry succeeds when NetBird recovers, and a second retry does NOT create a duplicate route!
	nb.SetPublishError(nil)
	succOp, err := exitsSvc.ExecuteOperation(ctx, pubOp.OperationId)
	if err != nil {
		t.Fatalf("ExecuteOperation retry: %v", err)
	}
	if succOp.Status != api.OperationStatusSucceeded {
		t.Fatalf("succOp.Status = %s, want succeeded", succOp.Status)
	}
	if nb.RouteCount() != 1 {
		t.Fatalf("nb.RouteCount() = %d, want 1", nb.RouteCount())
	}

	// Re-running ExecuteOperation or re-publishing must consult external_bindings and keep RouteCount() == 1.
	_, reloadETag, _ := nodesSvc.GetNode(ctx, nodeID)
	pubOp2, _, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        reloadETag,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            reason,
		},
	})
	if err != nil {
		t.Fatalf("SetPublication second time: %v", err)
	}
	// Compile + ACK new generation and run ExecuteOperation.
	_, _ = exitsSvc.ExecuteOperation(ctx, pubOp2.OperationId)
	detailMid, _, _ := nodesSvc.GetNode(ctx, nodeID)
	now = now.Add(5 * time.Second)
	submitAckObservation(t, ctx, nodesSvc, pool, nodeID, incID, relID, now, uint64(detailMid.ConfigGeneration.Desired), true)
	succOp2, err := exitsSvc.ExecuteOperation(ctx, pubOp2.OperationId)
	if err != nil || succOp2.Status != api.OperationStatusSucceeded {
		t.Fatalf("ExecuteOperation #2 = (%+v, %v), want succeeded", succOp2, err)
	}
	if nb.RouteCount() != 1 {
		t.Fatalf("duplicate NetBird route created on retry: RouteCount() = %d, want 1", nb.RouteCount())
	}

	obsState, err := nb.ReadExitState(ctx, "nb-main", exitID, "")
	if err != nil || !obsState.Exists || !obsState.Enabled {
		t.Fatalf("ReadExitState = (%+v, %v), want Exists=true Enabled=true", obsState, err)
	}
}

func TestAT13_UnpublishAndDisableWhenNodeOffline(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)

	nodesSvc := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now }))
	compiler := testExitsCompiler(t, pool, func() time.Time { return now })
	nb := NewFakeNetBirdAdapter()
	exitsSvc := NewService(pool, compiler, nb, WithClock(func() time.Time { return now }))

	nodeID, incID, exitID, relID, _ := seedExitTestNode(t, ctx, pool, api.PublicationIntentPublished, "active")

	// Seed an active NetBird route and mark node as currently forwarding traffic.
	nb.InjectRouteState(exitID, ObservedExitState{
		Exists:      true,
		ExternalID:  "nb-route-" + exitID.String(),
		PeerID:      "peer-" + nodeID.String()[:8],
		Enabled:     true,
		ObservedRef: "netbird:route:active",
	})
	if _, err := pool.Exec(ctx, `
		UPDATE nodes
		SET control_status = 'offline', member_forwarding_observed = true, resource_version = resource_version + 1
		WHERE id = $1`, nodeID); err != nil {
		t.Fatalf("mark node offline: %v", err)
	}

	// 1. Admin calls SetPublication(unpublished) while node is offline.
	_, etag1, _ := nodesSvc.GetNode(ctx, nodeID)
	reason := "maintenance withdrawal"
	unpubOp, _, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentUnpublished,
			Reason:            reason,
		},
	})
	if err != nil {
		t.Fatalf("SetPublication(unpublished): %v", err)
	}

	// 2. Worker executes node.unpublish while node is offline (does not ACK new generation).
	// NetBird route MUST still be withdrawn centrally, and operation status MUST be partially_succeeded
	// showing netbird_route_readback=succeeded and agent_policy=failed (§5.2 step 4, §18.3, AT13).
	partOp, err := exitsSvc.ExecuteOperation(ctx, unpubOp.OperationId)
	if err != nil {
		t.Fatalf("ExecuteOperation(unpublish offline): %v", err)
	}
	if partOp.Status != api.OperationStatusPartiallySucceeded {
		t.Fatalf("partOp.Status = %s, want partially_succeeded", partOp.Status)
	}

	obsRoute, err := nb.ReadExitState(ctx, "nb-main", exitID, "")
	if err != nil || obsRoute.Enabled {
		t.Fatalf("expected NetBird route to be withdrawn centrally even when node is offline, got (%+v, %v)", obsRoute, err)
	}

	var agentStep, nbStep api.OperationStepStatus
	for _, st := range partOp.Steps {
		switch st.Name {
		case nodes.StepAgentPolicy:
			agentStep = st.Status
		case nodes.StepNetBirdRouteReadback:
			nbStep = st.Status
		}
	}
	if agentStep != api.OperationStepStatusFailed || nbStep != api.OperationStepStatusSucceeded {
		t.Fatalf("steps = (agent_policy=%s, netbird_route_readback=%s), want (failed, succeeded)", agentStep, nbStep)
	}

	// 3. Node comes back online and reports member_forwarding_enabled=false at the new desired generation.
	// The partially_succeeded operation now completes to succeeded!
	detailOffline, _, _ := nodesSvc.GetNode(ctx, nodeID)
	now = now.Add(10 * time.Second)
	submitAckObservation(t, ctx, nodesSvc, pool, nodeID, incID, relID, now, uint64(detailOffline.ConfigGeneration.Desired), false)

	finalOp, err := nodesSvc.GetOperation(ctx, unpubOp.OperationId)
	if err != nil {
		t.Fatalf("GetOperation after node recovery: %v", err)
	}
	if finalOp.Status != api.OperationStatusSucceeded {
		t.Fatalf("finalOp.Status = %s, want succeeded after node confirmed withdrawal", finalOp.Status)
	}
}

func TestDisabledNode_NeverAutoReEnabledByHealthOrUpdate(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 16, 0, 0, 0, time.UTC)

	nodesSvc := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now }))
	compiler := testExitsCompiler(t, pool, func() time.Time { return now })
	nb := NewFakeNetBirdAdapter()
	exitsSvc := NewService(pool, compiler, nb, WithClock(func() time.Time { return now }))
	worker := NewWorker(pool, exitsSvc, compiler, "test-worker")

	nodeID, incID, _, relID, _ := seedExitTestNode(t, ctx, pool, api.PublicationIntentDisabled, "withdrawn")

	// Submit a completely healthy observation with applied_config_generation=1 and member_forwarding_enabled=false.
	submitAckObservation(t, ctx, nodesSvc, pool, nodeID, incID, relID, now, 1, false)

	// Run worker reconciliation.
	if _, err := worker.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	detail, _, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if detail.PublicationIntent != api.PublicationIntentDisabled {
		t.Fatalf("disabled node was changed to %s, must remain disabled (§5)", detail.PublicationIntent)
	}
	if detail.PublicationObserved != api.PublicationObservedWithdrawn {
		t.Fatalf("PublicationObserved = %s, want withdrawn", detail.PublicationObserved)
	}
}

func TestReconcileNetBird_DetectsDrift(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC)

	nodesSvc := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now }))
	compiler := testExitsCompiler(t, pool, func() time.Time { return now })
	nb := NewFakeNetBirdAdapter()
	exitsSvc := NewService(pool, compiler, nb, WithClock(func() time.Time { return now }))

	nodeID, incID, exitID, relID, _ := seedExitTestNode(t, ctx, pool, api.PublicationIntentUnpublished, "absent")

	// Publish exit cleanly.
	_, etag1, _ := nodesSvc.GetNode(ctx, nodeID)
	reason := "publish for drift test"
	pubOp, _, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            reason,
		},
	})
	if err != nil {
		t.Fatalf("SetPublication: %v", err)
	}
	_, _ = exitsSvc.ExecuteOperation(ctx, pubOp.OperationId)
	d1, _, _ := nodesSvc.GetNode(ctx, nodeID)
	now = now.Add(5 * time.Second)
	submitAckObservation(t, ctx, nodesSvc, pool, nodeID, incID, relID, now, uint64(d1.ConfigGeneration.Desired), true)
	if _, err := exitsSvc.ExecuteOperation(ctx, pubOp.OperationId); err != nil {
		t.Fatalf("ExecuteOperation: %v", err)
	}

	// Simulate external drift in NetBird: route gets disabled externally while publication_intent == published.
	nb.InjectRouteState(exitID, ObservedExitState{
		Exists:      true,
		ExternalID:  "nb-route-" + exitID.String(),
		PeerID:      "peer-" + nodeID.String()[:8],
		Enabled:     false,
		ObservedRef: "netbird:route:drifted",
	})

	items, err := exitsSvc.ReconcileNetBird(ctx)
	if err != nil {
		t.Fatalf("ReconcileNetBird: %v", err)
	}
	foundDrift := false
	for _, it := range items {
		if it.ExitID == exitID && len(it.Drift) > 0 {
			foundDrift = true
		}
	}
	if !foundDrift {
		t.Fatalf("expected ReconcileNetBird to detect drift for exit %s, got %+v", exitID, items)
	}

	afterDrift, _, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode after drift: %v", err)
	}
	if afterDrift.PublicationIntent != api.PublicationIntentPublished {
		t.Errorf("PublicationIntent changed to %s on drift, must stay published", afterDrift.PublicationIntent)
	}
	if len(afterDrift.Drift) == 0 {
		t.Errorf("expected non-empty Drift array on NodeDetail after NetBird drift")
	}
}

// emptyObservationReport supplies explicit nulls for required nullable wire fields.
func emptyObservationReport() api.ObservationReport {
	var r api.ObservationReport
	r.AppliedBindingRevision.SetNull()
	r.AppliedConfigGeneration.SetNull()
	r.AppliedPolicyRefs.SetNull()
	r.CurrentReleaseId.SetNull()
	r.LastGoodReleaseId.SetNull()
	r.Lease.ExpiresAt.SetNull()
	r.Lease.LeaseId.SetNull()
	r.Netbird.PeerId.SetNull()
	r.Netbird.Version.SetNull()
	r.Update.Generation.SetNull()
	r.Update.JobId.SetNull()
	r.Update.Stage.SetNull()
	r.Update.State.SetNull()
	r.Usage.BlockedSeq.SetNull()
	r.Usage.LedgerEpoch.SetNull()
	r.Usage.MeterEpochId.SetNull()
	r.Usage.OldestUnackedAt.SetNull()
	return r
}

func TestAudit_D029Conformance_NodesDesiredStateLeasesExits(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 18, 0, 0, 0, time.UTC)

	nodesSvc := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now }))
	compiler := testExitsCompiler(t, pool, func() time.Time { return now })
	nb := NewFakeNetBirdAdapter()
	exitsSvc := NewService(pool, compiler, nb, WithClock(func() time.Time { return now }))

	leaseSeed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - exits-lease-key"))
	leaseSigner := desiredstate.NewKeySigner(desiredstate.SigningKey{
		KeyID:      "lease-key-exits-1",
		Purpose:    envelope.PurposeLease,
		PrivateKey: ed25519.NewKeyFromSeed(leaseSeed[:]),
	})
	leaseSvc := leases.NewService(pool, leaseSigner, leases.WithClock(func() time.Time { return now }))

	nodeID, incID, exitID, relID, _ := seedExitTestNode(t, ctx, pool, api.PublicationIntentUnpublished, "absent")
	adminUUID := uuid.New()

	// 1. exits.SetPublication (writes exit:<uuid> audit row).
	_, etag1, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	pubOp, _, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            "publish for D-029 audit test",
		},
	})
	if err != nil {
		t.Fatalf("SetPublication: %v", err)
	}

	// 2. desiredstate.Compile (writes node:<uuid> audit row).
	compiled, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// 3. nodes.SubmitObservation (writes node:<uuid> audit row).
	now = now.Add(5 * time.Second)
	submitAckObservation(t, ctx, nodesSvc, pool, nodeID, incID, relID, now, compiled.Generation, true)
	if _, err := exitsSvc.ExecuteOperation(ctx, pubOp.OperationId); err != nil {
		t.Fatalf("ExecuteOperation: %v", err)
	}

	// 4. leases.Renew (delegates to RenewTx, writes node:<uuid> audit row).
	if _, err := leaseSvc.Renew(ctx, nodeID, incID, api.LeaseRenewalRequest{
		RenewalRequestId:        uuid.New(),
		NodeId:                  nodeID,
		NodeIncarnationId:       incID,
		AppliedConfigGeneration: presence.NewNullableWithValue(decstr.U64(compiled.Generation)),
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
	}); err != nil {
		t.Fatalf("Renew: %v", err)
	}

	// 5. nodes.UpdateNode, RevokeNode, RetireNode (write node:<uuid> audit rows).
	_, etag2, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode(2): %v", err)
	}
	notes := "updated notes for D-029"
	updRes, err := nodesSvc.UpdateNode(ctx, nodes.UpdateNodeParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: uuid.New(),
		IfMatch:        etag2,
		Patch:          api.NodePatch{Notes: &notes},
	})
	if err != nil {
		t.Fatalf("UpdateNode: %v", err)
	}
	_, etag3, err := nodesSvc.RevokeNode(ctx, nodes.RevokeNodeParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: uuid.New(),
		IfMatch:        updRes.ETag,
		Request:        api.ReasonRequest{Reason: "revoke for D-029 audit test"},
	})
	if err != nil {
		t.Fatalf("RevokeNode: %v", err)
	}
	if _, _, err := nodesSvc.RetireNode(ctx, nodes.RetireNodeParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: uuid.New(),
		IfMatch:        etag3,
		Reason:         "retire for D-029 audit test",
	}); err != nil {
		t.Fatalf("RetireNode: %v", err)
	}

	// 6. Prove with audit.Store.List that all rows are returned with zero anomalies.
	var rawAuditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&rawAuditCount); err != nil {
		t.Fatalf("count audit_events: %v", err)
	}
	if rawAuditCount < 7 {
		t.Fatalf("raw audit_events count = %d, want >= 7", rawAuditCount)
	}

	limit := 100
	auditStore := &audit.Store{Pool: pool}
	listed, err := auditStore.List(ctx, api.ListAuditEventsParams{Limit: &limit})
	if err != nil {
		t.Fatalf("audit.Store.List: %v", err)
	}
	if auditStore.AnomalyCount() != 0 {
		t.Fatalf("audit.Store.AnomalyCount() = %d, want 0", auditStore.AnomalyCount())
	}
	if len(listed.Items) != rawAuditCount {
		t.Fatalf("len(listed.Items) = %d, want %d (all audit rows must pass D-029 validation)", len(listed.Items), rawAuditCount)
	}

	seenActions := map[string]bool{}
	for _, ev := range listed.Items {
		seenActions[ev.Action] = true
		validNode := ev.Resource.Type == api.ResourceTypeNode && ev.Resource.Id == nodeID
		validExit := ev.Resource.Type == api.ResourceTypeExit && ev.Resource.Id == exitID
		if !validNode && !validExit {
			t.Errorf("unexpected audit resource %+v, want node:%s or exit:%s", ev.Resource, nodeID, exitID)
		}
		if ev.Actor.Type != api.AuditEventActorTypeSystem {
			actorID, err := ev.Actor.Id.Get()
			if err != nil || actorID == uuid.Nil {
				t.Errorf("audit event %s missing valid UUID actor: %+v (err=%v)", ev.Action, ev.Actor, err)
			}
		}
		if ev.Result != api.AuditEventResultSucceeded && ev.Result != api.AuditEventResultFailed && ev.Result != api.AuditEventResultDenied {
			t.Errorf("audit event %s has invalid result %q", ev.Action, ev.Result)
		}
	}
	for _, wantAct := range []string{
		"node.publish",
		"node.desired_state_compiled",
		"node.observation_applied",
		"node.lease_renewed",
		"node.update",
		"node.revoke",
		"node.retire",
	} {
		if !seenActions[wantAct] {
			t.Errorf("missing expected audit action %q in audit.Store.List results", wantAct)
		}
	}
}

func TestSetPublicationTx_CallerTransactionRollbackWrapperAndStaleIfMatch(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 19, 0, 0, 0, time.UTC)

	nodesSvc := nodes.NewService(pool, nodes.WithClock(func() time.Time { return now }))
	compiler := testExitsCompiler(t, pool, func() time.Time { return now })
	nb := NewFakeNetBirdAdapter()
	exitsSvc := NewService(pool, compiler, nb, WithClock(func() time.Time { return now }))

	nodeID, _, exitID, _, _ := seedExitTestNode(t, ctx, pool, api.PublicationIntentUnpublished, "absent")
	adminUUID := uuid.New()
	reqID := uuid.New()

	initialDetail, etag1, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode(initial): %v", err)
	}
	if initialDetail.ConfigGeneration.Desired != 1 || initialDetail.ResourceVersion != 2 {
		t.Fatalf("initial (desiredGen=%s, resourceVer=%s), want (1, 2)", initialDetail.ConfigGeneration.Desired, initialDetail.ResourceVersion)
	}

	// 1. Calling SetPublicationTx inside a caller transaction that rolls back leaves no rows or state changes.
	txRollback, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("pool.Begin(rollback): %v", err)
	}
	rbOp, rbETag, err := exitsSvc.SetPublicationTx(ctx, txRollback, SetPublicationParams{
		NodeID:    nodeID,
		Principal: adminUUID.String(),
		IfMatch:   etag1,
		RequestID: &reqID,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            "publish in rolled-back caller tx",
		},
	})
	if err != nil {
		_ = txRollback.Rollback(ctx)
		t.Fatalf("SetPublicationTx(rollback tx): %v", err)
	}
	if rbOp.Kind != api.OperationKindNodePublish || rbETag != `"3"` {
		_ = txRollback.Rollback(ctx)
		t.Fatalf("SetPublicationTx(rollback tx) = (kind=%s, etag=%s), want (node.publish, \"3\")", rbOp.Kind, rbETag)
	}
	if err := txRollback.Rollback(ctx); err != nil {
		t.Fatalf("txRollback.Rollback: %v", err)
	}

	var opCount, outboxCount, auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM operations WHERE resource_id = $1`, nodeID).Scan(&opCount); err != nil || opCount != 0 {
		t.Fatalf("after rollback operations count = %d (err=%v), want 0", opCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1`, nodeID).Scan(&outboxCount); err != nil || outboxCount != 0 {
		t.Fatalf("after rollback outbox_events count = %d (err=%v), want 0", outboxCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&auditCount); err != nil || auditCount != 0 {
		t.Fatalf("after rollback audit_events count = %d (err=%v), want 0", auditCount, err)
	}
	afterRollbackDetail, afterRollbackETag, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode(after rollback): %v", err)
	}
	if afterRollbackETag != etag1 || afterRollbackDetail.PublicationIntent != api.PublicationIntentUnpublished || afterRollbackDetail.ConfigGeneration.Desired != 1 {
		t.Fatalf("after rollback node = (etag=%s, intent=%s, gen=%s), want (%s, unpublished, 1)",
			afterRollbackETag, afterRollbackDetail.PublicationIntent, afterRollbackDetail.ConfigGeneration.Desired, etag1)
	}

	// 2. Stale If-Match fails under the row lock with 412 RESOURCE_VERSION_CONFLICT.
	err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		_, _, txErr := exitsSvc.SetPublicationTx(ctx, tx, SetPublicationParams{
			NodeID:    nodeID,
			Principal: adminUUID.String(),
			IfMatch:   `"999"`,
			RequestID: &reqID,
			Request: api.PublicationRequest{
				PublicationIntent: api.PublicationIntentPublished,
				Reason:            "stale if-match test",
			},
		})
		return txErr
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.ResourceVersionConflict || ae.HTTPStatus() != 412 {
		t.Fatalf("SetPublicationTx(stale If-Match) err = %v, want 412 RESOURCE_VERSION_CONFLICT", err)
	}

	// 3. SetPublicationTx for `published` inside a committed caller transaction (with caller-owned idempotency row).
	var pubOp api.Operation
	var etag2 string
	callerIdemKey := sha256.Sum256([]byte("caller-idem-key-1"))
	callerReqHash := sha256.Sum256([]byte("caller-req-hash-1"))
	err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rec, bErr := db.BeginIdempotent(ctx, tx, db.IdempotencyScope{
			Principal:   "admin:" + adminUUID.String(),
			Method:      "PUT",
			Route:       "/api/v1/admin/nodes/" + nodeID.String() + "/publication",
			KeyHash:     callerIdemKey[:],
			RequestHash: callerReqHash[:],
		}, 0)
		if bErr != nil {
			return bErr
		}
		var txErr error
		pubOp, etag2, txErr = exitsSvc.SetPublicationTx(ctx, tx, SetPublicationParams{
			NodeID:    nodeID,
			Principal: adminUUID.String(),
			IfMatch:   etag1,
			RequestID: &reqID,
			Request: api.PublicationRequest{
				PublicationIntent: api.PublicationIntentPublished,
				Reason:            "publish in committed caller tx",
			},
		})
		if txErr != nil {
			return txErr
		}
		nodeIDCopy := nodeID
		return db.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{
			OperationID:    &pubOp.OperationId,
			ResourceType:   "node",
			ResourceID:     &nodeIDCopy,
			ResponseStatus: 202,
		})
	})
	if err != nil {
		t.Fatalf("SetPublicationTx(published committed): %v", err)
	}
	if pubOp.Kind != api.OperationKindNodePublish || pubOp.Status != api.OperationStatusPending || etag2 != `"3"` {
		t.Fatalf("SetPublicationTx(published) = (kind=%s, status=%s, etag=%s), want (node.publish, pending, \"3\")",
			pubOp.Kind, pubOp.Status, etag2)
	}
	if gen, gErr := pubOp.DesiredGeneration.Get(); gErr != nil || gen != 2 {
		t.Fatalf("pubOp.DesiredGeneration = (%v, %v), want 2", gen, gErr)
	}
	wantSteps := map[string]bool{
		nodes.StepAgentPolicy:          false,
		nodes.StepNetBirdRouteReadback: false,
		nodes.StepDataPathProbe:        false,
	}
	for _, st := range pubOp.Steps {
		if _, ok := wantSteps[st.Name]; ok && st.Required && st.Status == api.OperationStepStatusPending {
			wantSteps[st.Name] = true
		}
	}
	for name, seen := range wantSteps {
		if !seen {
			t.Fatalf("missing required pending step %q on node.publish operation: %+v", name, pubOp.Steps)
		}
	}

	afterPubDetail, _, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode(after publish): %v", err)
	}
	if afterPubDetail.PublicationIntent != api.PublicationIntentPublished || afterPubDetail.ConfigGeneration.Desired != 2 || afterPubDetail.ResourceVersion != 3 {
		t.Fatalf("after publish node = (intent=%s, desiredGen=%s, ver=%s), want (published, 2, 3)",
			afterPubDetail.PublicationIntent, afterPubDetail.ConfigGeneration.Desired, afterPubDetail.ResourceVersion)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = $2`, nodeID, desiredstate.EventNodePublicationChanged).Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("outbox_events count = %d (err=%v), want 1", outboxCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE resource = $1 AND action = 'node.publish'`, "exit:"+exitID.String()).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("audit_events count = %d (err=%v), want 1", auditCount, err)
	}

	// 4. Wrapper SetPublication still behaves as before for unpublished and disabled (including idempotent replay).
	unpubKey := uuid.New()
	unpubOp, etag3, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: unpubKey,
		IfMatch:        etag2,
		RequestID:      &reqID,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentUnpublished,
			Reason:            "unpublish via wrapper",
		},
	})
	if err != nil {
		t.Fatalf("SetPublication(unpublished): %v", err)
	}
	if unpubOp.Kind != api.OperationKindNodeUnpublish || etag3 != `"4"` {
		t.Fatalf("SetPublication(unpublished) = (kind=%s, etag=%s), want (node.unpublish, \"4\")", unpubOp.Kind, etag3)
	}
	replayOp, replayETag, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: unpubKey,
		IfMatch:        etag2,
		RequestID:      &reqID,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentUnpublished,
			Reason:            "unpublish via wrapper",
		},
	})
	if err != nil || replayOp.OperationId != unpubOp.OperationId || replayETag != etag3 {
		t.Fatalf("SetPublication(unpublished replay) = (%+v, %s, %v), want op=%s etag=%s", replayOp, replayETag, err, unpubOp.OperationId, etag3)
	}

	disOp, etag4, err := exitsSvc.SetPublication(ctx, SetPublicationParams{
		NodeID:         nodeID,
		Principal:      adminUUID.String(),
		IdempotencyKey: uuid.New(),
		IfMatch:        etag3,
		RequestID:      &reqID,
		Request: api.PublicationRequest{
			PublicationIntent: api.PublicationIntentDisabled,
			Reason:            "disable via wrapper",
		},
	})
	if err != nil {
		t.Fatalf("SetPublication(disabled): %v", err)
	}
	if disOp.Kind != api.OperationKindNodeDisable || etag4 != `"5"` {
		t.Fatalf("SetPublication(disabled) = (kind=%s, etag=%s), want (node.disable, \"5\")", disOp.Kind, etag4)
	}

	afterDisDetail, _, err := nodesSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode(after disable): %v", err)
	}
	if afterDisDetail.PublicationIntent != api.PublicationIntentDisabled || afterDisDetail.ConfigGeneration.Desired != 4 || afterDisDetail.ResourceVersion != 5 {
		t.Fatalf("after disable node = (intent=%s, desiredGen=%s, ver=%s), want (disabled, 4, 5)",
			afterDisDetail.PublicationIntent, afterDisDetail.ConfigGeneration.Desired, afterDisDetail.ResourceVersion)
	}

	limit := 100
	auditStore := &audit.Store{Pool: pool}
	listed, err := auditStore.List(ctx, api.ListAuditEventsParams{Limit: &limit})
	if err != nil {
		t.Fatalf("audit.Store.List: %v", err)
	}
	if auditStore.AnomalyCount() != 0 || len(listed.Items) != 3 {
		t.Fatalf("audit.Store.List len=%d anomalies=%d, want len=3 anomalies=0", len(listed.Items), auditStore.AnomalyCount())
	}
}
