//go:build integration

package nodes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type seededNode struct {
	NodeID        uuid.UUID
	IncarnationID uuid.UUID
	ExitID        uuid.UUID
	ReleaseID     uuid.UUID
	GroupID       uuid.UUID
}

func seedTestReleaseAndGroup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	relID := uuid.New()
	manifestHash := sha256.Sum256([]byte("manifest-" + suffix))
	sig := make([]byte, 64)
	if _, err := pool.Exec(ctx, `
		INSERT INTO releases (
			id, version, release_sequence, manifest_schema_version,
			manifest, manifest_hash, signature, signing_key_ref, component_versions
		) VALUES (
			$1, $2, $3, 1,
			'{"schema_version":1}'::jsonb, $4, $5, 'rel-key-1', '{"fvpn_agent":"1.0.0"}'::jsonb
		)`, relID, "2026.10.29-"+suffix, 1, manifestHash[:], sig); err != nil {
		t.Fatalf("insert release: %v", err)
	}

	groupID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO member_groups (id, code, name)
		VALUES ($1, $2, $3)`, groupID, "grp-"+suffix, "Group "+suffix); err != nil {
		t.Fatalf("insert member_group: %v", err)
	}
	return relID, groupID
}

func seedTestNode(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	name string,
	region string,
	intent api.PublicationIntent,
	observed string,
	control api.ControlStatus,
	relID uuid.UUID,
	groupID uuid.UUID,
) seededNode {
	t.Helper()
	nodeID := uuid.New()
	incID := uuid.New()
	exitID := uuid.New()
	fp := sha256.Sum256([]byte("pubkey-" + nodeID.String()))
	certFP := sha256.Sum256([]byte("cert-" + nodeID.String()))
	tokenHash := sha256.Sum256([]byte("tok-" + nodeID.String()))

	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes (
			id, lifecycle_state, provisioning_state,
			desired_config_generation, applied_config_generation,
			desired_release_id, current_release_id, last_good_release_id,
			control_status, data_health, safety_status,
			member_forwarding_observed
		) VALUES (
			$1, 'registered', 'ready',
			2, 1,
			$2, $2, $2,
			$3, 'healthy', 'passed',
			false
		)`, nodeID, relID, string(control)); err != nil {
		t.Fatalf("insert node: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO node_incarnations (
			id, node_id, public_key_fingerprint, deployment_id, peer_id, os, os_version, arch
		) VALUES (
			$1, $2, $3, 'nb-main', $4, 'ubuntu', '24.04', 'amd64'
		)`, incID, nodeID, fp[:], "peer-"+nodeID.String()[:8]); err != nil {
		t.Fatalf("insert node_incarnation: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE nodes SET current_incarnation_id = $1, resource_version = resource_version + 1 WHERE id = $2`, incID, nodeID); err != nil {
		t.Fatalf("set current_incarnation_id: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO node_enrollments (
			node_id, token_hash, created_at, expiry, state
		) VALUES (
			$1, $2, now() - interval '5 minutes', now() + interval '10 minutes', 'pending'
		)`, nodeID, tokenHash[:]); err != nil {
		t.Fatalf("insert node_enrollment: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO node_certificates (
			incarnation_id, issuer, serial, fingerprint, public_certificate, not_before, not_after
		) VALUES (
			$1, 'fvpn-ca', $2, $3, $4, now() - interval '1 hour', now() + interval '30 days'
		)`, incID, "ser-"+nodeID.String()[:8], certFP[:], []byte("TEST ONLY CERT")); err != nil {
		t.Fatalf("insert node_certificate: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO exits (
			id, node_id, display_name, region, publication_intent, publication_observed
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`, exitID, nodeID, name, region, string(intent), observed); err != nil {
		t.Fatalf("insert exit: %v", err)
	}

	if groupID != uuid.Nil {
		if _, err := pool.Exec(ctx, `
			INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, groupID); err != nil {
			t.Fatalf("insert exit_group: %v", err)
		}
	}

	return seededNode{
		NodeID:        nodeID,
		IncarnationID: incID,
		ExitID:        exitID,
		ReleaseID:     relID,
		GroupID:       groupID,
	}
}

func TestNodeAggregate_ListGetFiltersCursorAndExitsJoin(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	svc := NewService(pool)

	relID, grpID := seedTestReleaseAndGroup(t, ctx, pool, "agg")
	n1 := seedTestNode(t, ctx, pool, "tw-taipei-01", "tw-tpe", api.PublicationIntentPublished, "active", api.ControlStatusOnline, relID, grpID)
	n2 := seedTestNode(t, ctx, pool, "jp-tokyo-01", "jp-tyo", api.PublicationIntentUnpublished, "withdrawn", api.ControlStatusOnline, relID, grpID)
	n3 := seedTestNode(t, ctx, pool, "sg-singapore-01", "sg-sin", api.PublicationIntentDisabled, "error", api.ControlStatusOffline, relID, uuid.Nil)

	// 1. GetNode verifies separate desired vs observed fields and ETag.
	detail, etag, err := svc.GetNode(ctx, n1.NodeID)
	if err != nil {
		t.Fatalf("GetNode(n1): %v", err)
	}
	if etag != FormatETag(detail.ResourceVersion.String()) {
		t.Fatalf("ETag = %q, want %q", etag, FormatETag(detail.ResourceVersion.String()))
	}
	if detail.PublicationIntent != api.PublicationIntentPublished {
		t.Errorf("PublicationIntent = %s, want published", detail.PublicationIntent)
	}
	if detail.PublicationObserved != api.PublicationObservedActive {
		t.Errorf("PublicationObserved = %s, want active", detail.PublicationObserved)
	}
	if uint64(detail.ConfigGeneration.Desired) != 2 || (!detail.ConfigGeneration.Applied.IsSpecified() || detail.ConfigGeneration.Applied.IsNull()) || uint64(nullableValue(t, detail.ConfigGeneration.Applied)) != 1 {
		t.Errorf("ConfigGeneration = %+v, want desired=2 applied=1", detail.ConfigGeneration)
	}
	if (!detail.Releases.DesiredReleaseId.IsSpecified() || detail.Releases.DesiredReleaseId.IsNull()) || nullableValue(t, detail.Releases.DesiredReleaseId) != relID {
		t.Errorf("DesiredReleaseId = %v, want %s", detail.Releases.DesiredReleaseId, relID)
	}
	if len(detail.MemberGroupIds) != 1 || detail.MemberGroupIds[0] != grpID {
		t.Errorf("MemberGroupIds = %v, want [%s]", detail.MemberGroupIds, grpID)
	}

	assertExplicitNulls(t, detail, "capacity", "notes", "retired_at", "update", "last_seen_at", "observed_exit_ipv4", "observed_exit_ipv6")

	// 2. ListNodes with filters.
	pubIntent := api.PublicationIntentPublished
	pubList, err := svc.ListNodes(ctx, ListNodesParams{PublicationIntent: &pubIntent})
	if err != nil {
		t.Fatalf("ListNodes(published): %v", err)
	}
	if len(pubList.Items) != 1 || pubList.Items[0].NodeId != n1.NodeID {
		t.Fatalf("ListNodes(published) returned %d items, want [n1]", len(pubList.Items))
	}

	assertExplicitNulls(t, pubList, "next_cursor")
	assertExplicitNulls(t, pubList.Items[0], "last_seen_at", "update")

	regFilter := "jp-tyo"
	regList, err := svc.ListNodes(ctx, ListNodesParams{Region: &regFilter})
	if err != nil {
		t.Fatalf("ListNodes(region=jp-tyo): %v", err)
	}
	if len(regList.Items) != 1 || regList.Items[0].NodeId != n2.NodeID {
		t.Fatalf("ListNodes(region=jp-tyo) returned %d items, want [n2]", len(regList.Items))
	}

	ctrlFilter := api.ControlStatusOffline
	ctrlList, err := svc.ListNodes(ctx, ListNodesParams{ControlStatus: &ctrlFilter})
	if err != nil {
		t.Fatalf("ListNodes(control_status=offline): %v", err)
	}
	if len(ctrlList.Items) != 1 || ctrlList.Items[0].NodeId != n3.NodeID {
		t.Fatalf("ListNodes(control_status=offline) returned %d items, want [n3]", len(ctrlList.Items))
	}

	// 3. Cursor pagination across all 3 nodes.
	var seen []uuid.UUID
	var cursor string
	for page := 0; page < 5; page++ {
		res, err := svc.ListNodes(ctx, ListNodesParams{Limit: 1, Cursor: cursor})
		if err != nil {
			t.Fatalf("ListNodes page %d: %v", page, err)
		}
		for _, item := range res.Items {
			seen = append(seen, item.NodeId)
		}
		if !res.NextCursor.IsSpecified() || res.NextCursor.IsNull() {
			break
		}
		cursor = nullableValue(t, res.NextCursor)
	}
	if len(seen) != 3 {
		t.Fatalf("cursor pagination visited %d nodes, want 3", len(seen))
	}

	// Invalid cursor returns 400.
	if _, err := svc.ListNodes(ctx, ListNodesParams{Cursor: "not-base64-json!!!"}); err == nil {
		t.Fatalf("expected invalid cursor to return error")
	}

	// 4. Member exit list returns only published exits in the member's D-030 serving exit groups (no assignments row).
	memberID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, email, display_name, status)
		VALUES ($1, 'member@example.com', 'Test Member', 'active')`, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlements (
			member_id, revision, state, device_limit, valid_from, effective_service_end, admin_suspended, projection
		) VALUES (
			$1, 1, 'active', 3, now() - interval '1 day', now() + interval '30 days', false,
			jsonb_build_object('exit_group_ids', to_jsonb(ARRAY[$2::text]))
		)`, memberID, grpID.String()); err != nil {
		t.Fatalf("insert entitlement: %v", err)
	}

	exitsList, err := svc.ListMemberExits(ctx, memberID)
	if err != nil {
		t.Fatalf("ListMemberExits: %v", err)
	}
	assertExplicitNulls(t, exitsList, "next_cursor")
	if len(exitsList.Items) != 1 || exitsList.Items[0].ExitId != n1.ExitID || exitsList.Items[0].Availability != api.MemberExitAvailabilityAvailable {
		t.Fatalf("ListMemberExits = %+v, want [n1 available]", exitsList.Items)
	}
	assertExplicitNulls(t, exitsList.Items[0], "note")
}

func TestNodePatch_IfMatchAnd200Vs202(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	svc := NewService(pool)

	relID, grpID := seedTestReleaseAndGroup(t, ctx, pool, "patch")
	_, grpID2 := seedTestReleaseAndGroup(t, ctx, pool, "patch2")
	n := seedTestNode(t, ctx, pool, "tw-taipei-01", "tw-tpe", api.PublicationIntentUnpublished, "withdrawn", api.ControlStatusOnline, relID, grpID)

	_, etag1, err := svc.GetNode(ctx, n.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}

	// 1. Missing If-Match -> 428 PRECONDITION_REQUIRED.
	newName := "tw-taipei-renamed"
	_, err = svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        "",
		Patch:          api.NodePatch{Name: &newName},
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.PreconditionRequired || ae.HTTPStatus() != 428 {
		t.Fatalf("missing If-Match err = %v, want 428 PRECONDITION_REQUIRED", err)
	}

	// 2. Mismatched If-Match -> 412 RESOURCE_VERSION_CONFLICT with current_resource_version.
	_, err = svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        `"999"`,
		Patch:          api.NodePatch{Name: &newName},
	})
	if !errors.As(err, &ae) || ae.Code != apierr.ResourceVersionConflict || ae.HTTPStatus() != 412 {
		t.Fatalf("stale If-Match err = %v, want 412 RESOURCE_VERSION_CONFLICT", err)
	}
	if ae.Details["current_resource_version"] == nil {
		t.Fatalf("expected current_resource_version in 412 error details, got %+v", ae.Details)
	}

	// 3. clear_capacity + capacity -> 422 VALIDATION_FAILED.
	clearCap := api.NodePatchClearCapacity(true)
	bw := int64(500_000_000)
	_, err = svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Patch: api.NodePatch{
			ClearCapacity: &clearCap,
			Capacity: &api.NodeCapacity{DownBps: bw, UpBps: bw,
				MemberDownBpsCap: presence.NewNullNullable[int64](),
				MemberUpBpsCap:   presence.NewNullNullable[int64]()},
		},
	})
	if !errors.As(err, &ae) || ae.Code != apierr.ValidationFailed || ae.HTTPStatus() != 422 {
		t.Fatalf("clear_capacity + capacity err = %v, want 422 VALIDATION_FAILED", err)
	}

	// 4. Metadata-only PATCH (name, region, notes) -> 200 NodeDetail, no operation created.
	newNotes := "rack B2"
	res200, err := svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Patch: api.NodePatch{
			Name:  &newName,
			Notes: &newNotes,
		},
	})
	if err != nil {
		t.Fatalf("metadata UpdateNode: %v", err)
	}
	if res200.Status != 200 || res200.Node == nil || res200.Operation != nil {
		t.Fatalf("metadata UpdateNode result = %+v, want Status=200 with Node", res200)
	}
	if (!res200.Node.Name.IsSpecified() || res200.Node.Name.IsNull()) || nullableValue(t, res200.Node.Name) != newName || (!res200.Node.Notes.IsSpecified() || res200.Node.Notes.IsNull()) || nullableValue(t, res200.Node.Notes) != newNotes {
		t.Errorf("updated node = %+v, want name=%q notes=%q", res200.Node, newName, newNotes)
	}
	if res200.ETag == etag1 {
		t.Errorf("expected ETag to advance after metadata update, stayed %q", res200.ETag)
	}

	// 5. Policy-affecting PATCH (member_group_ids + capacity) -> 202 Operation + outbox event.
	idemKey := uuid.New()
	memberCap := int64(50_000_000)
	groups := []uuid.UUID{grpID, grpID2}
	res202, err := svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: idemKey,
		IfMatch:        res200.ETag,
		Patch: api.NodePatch{
			MemberGroupIds: &groups,
			Capacity: &api.NodeCapacity{
				DownBps:          bw,
				UpBps:            bw,
				MemberDownBpsCap: presence.NewNullableWithValue(memberCap),
				MemberUpBpsCap:   presence.NewNullableWithValue(memberCap),
			},
		},
	})
	if err != nil {
		t.Fatalf("policy UpdateNode: %v", err)
	}
	if res202.Status != 202 || res202.Operation == nil {
		t.Fatalf("policy UpdateNode result = %+v, want Status=202 with Operation", res202)
	}
	if string(res202.Operation.Kind) != OpKindUpdate || res202.Operation.Status != api.OperationStatusPending {
		t.Errorf("Operation = %+v, want kind=node.update status=pending", res202.Operation)
	}

	assertExplicitNulls(t, res202.Operation, "completed_at", "error", "observed_generation")
	for _, step := range res202.Operation.Steps {
		assertExplicitNulls(t, step, "error_code", "message")
	}

	// Idempotent replay with same Idempotency-Key returns same 202 operation.
	replay202, err := svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: idemKey,
		IfMatch:        res200.ETag,
		Patch: api.NodePatch{
			MemberGroupIds: &groups,
			Capacity: &api.NodeCapacity{
				DownBps:          bw,
				UpBps:            bw,
				MemberDownBpsCap: presence.NewNullableWithValue(memberCap),
				MemberUpBpsCap:   presence.NewNullableWithValue(memberCap),
			},
		},
	})
	if err != nil {
		t.Fatalf("replay UpdateNode: %v", err)
	}
	if replay202.Operation == nil || replay202.Operation.OperationId != res202.Operation.OperationId {
		t.Fatalf("replay operation = %+v, want %s", replay202.Operation, res202.Operation.OperationId)
	}

	// Reuse of Idempotency-Key with different payload -> 409 IDEMPOTENCY_CONFLICT.
	_, err = svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: idemKey,
		IfMatch:        res202.ETag,
		Patch:          api.NodePatch{Name: &newName},
	})
	if !errors.As(err, &ae) || ae.Code != apierr.IdempotencyConflict || ae.HTTPStatus() != 409 {
		t.Fatalf("idempotency reuse err = %v, want 409 IDEMPOTENCY_CONFLICT", err)
	}
}

func TestObservations_DedupeMonotonicSafetyAndOperationEvidence(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	svc := NewService(pool, WithClock(func() time.Time { return now }))

	relID, grpID := seedTestReleaseAndGroup(t, ctx, pool, "obs")
	n := seedTestNode(t, ctx, pool, "tw-taipei-01", "tw-tpe", api.PublicationIntentUnpublished, "withdrawn", api.ControlStatusOnline, relID, grpID)

	// Create a pending node.update operation expecting desired_generation = 3.
	_, etag0, _ := svc.GetNode(ctx, n.NodeID)
	clearCap := api.NodePatchClearCapacity(true)
	updRes, err := svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         n.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag0,
		Patch:          api.NodePatch{ClearCapacity: &clearCap},
	})
	if err != nil {
		t.Fatalf("UpdateNode: %v", err)
	}
	opID := updRes.Operation.OperationId

	makeReport := func(reportID uuid.UUID, obsTime time.Time, appliedGen uint64, backend api.ObservationReportNetbirdBackend, internetOnlyStatus api.HealthCheckStatus) api.ObservationReport {
		gen := ParseU64(appliedGen)
		relCopy := relID
		r := emptyObservationReport()
		r.AgentVersion = "1.0.0"
		r.BootId = uuid.New()
		r.ReportId = reportID
		r.NodeId = n.NodeID
		r.NodeIncarnationId = n.IncarnationID
		r.ObservedAt = obsTime
		r.CurrentReleaseId.Set(relCopy)
		r.LastGoodReleaseId.Set(relCopy)
		r.AppliedConfigGeneration.Set(gen)
		r.ProvisioningState = api.ProvisioningStateReady
		r.SafetyStatus = api.SafetyStatusPassed
		r.DataHealth = api.DataHealthHealthy
		r.MemberForwardingEnabled = false
		r.Netbird.Backend = backend
		r.Checks = []api.NodeCheck{
			{Name: api.NodeCheckNameInternetOnly, Status: internetOnlyStatus, ObservedAt: obsTime,
				ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
			{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: obsTime,
				ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
			{Name: api.NodeCheckNameDnsPolicy, Status: api.HealthCheckStatusPassed, ObservedAt: obsTime,
				ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
		}
		return r
	}

	// 1. Submit an observation at applied_config_generation=1 (< desired_generation=3).
	// A 202 observation alone must NOT complete the operation step!
	rep1ID := uuid.New()
	obsTime1 := now.Add(-10 * time.Second)
	rep1 := makeReport(rep1ID, obsTime1, 1, api.ObservationReportNetbirdBackendKernel, api.HealthCheckStatusPassed)
	acc1, err := svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, rep1)
	if err != nil {
		t.Fatalf("SubmitObservation(rep1): %v", err)
	}
	if !acc1.ReceivedAt.Equal(now) {
		t.Errorf("ReceivedAt = %v, want %v (separate from ObservedAt %v)", acc1.ReceivedAt, now, obsTime1)
	}

	opAfter1, err := svc.GetOperation(ctx, opID)
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}
	if opAfter1.Status == api.OperationStatusSucceeded {
		t.Fatalf("operation succeeded prematurely on applied_config_generation=1")
	}

	// 2. Duplicate report_id with identical payload -> 202 with original received_at.
	now = now.Add(5 * time.Second)
	accDup, err := svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, rep1)
	if err != nil {
		t.Fatalf("SubmitObservation duplicate: %v", err)
	}
	if !accDup.ReceivedAt.Equal(acc1.ReceivedAt) {
		t.Errorf("duplicate ReceivedAt = %v, want original %v", accDup.ReceivedAt, acc1.ReceivedAt)
	}

	// 3. Duplicate report_id with modified payload -> 409 ACK_CONFLICT.
	rep1Conflict := rep1
	gen99 := ParseU64(99)
	rep1Conflict.AppliedConfigGeneration.Set(gen99)
	_, err = svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, rep1Conflict)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.AckConflict || ae.HTTPStatus() != 409 {
		t.Fatalf("duplicate conflict err = %v, want 409 ACK_CONFLICT", err)
	}

	// 4. Submit newer observation at applied_config_generation=3 -> completes node.update operation!
	rep2 := makeReport(uuid.New(), obsTime1.Add(5*time.Second), 3, api.ObservationReportNetbirdBackendKernel, api.HealthCheckStatusPassed)
	if _, err := svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, rep2); err != nil {
		t.Fatalf("SubmitObservation(rep2): %v", err)
	}
	opAfter2, err := svc.GetOperation(ctx, opID)
	if err != nil {
		t.Fatalf("GetOperation after rep2: %v", err)
	}
	if opAfter2.Status != api.OperationStatusSucceeded {
		t.Fatalf("opAfter2.Status = %s, want succeeded", opAfter2.Status)
	}

	// 5. Submit an out-of-order OLDER observation (ObservedAt before rep2) -> accepted,
	// but does NOT overwrite applied_config_generation=3 back to 2.
	repOld := makeReport(uuid.New(), obsTime1.Add(-30*time.Second), 2, api.ObservationReportNetbirdBackendKernel, api.HealthCheckStatusPassed)
	if _, err := svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, repOld); err != nil {
		t.Fatalf("SubmitObservation(repOld): %v", err)
	}
	detailAfterOld, _, err := svc.GetNode(ctx, n.NodeID)
	if err != nil {
		t.Fatalf("GetNode after repOld: %v", err)
	}
	if (!detailAfterOld.ConfigGeneration.Applied.IsSpecified() || detailAfterOld.ConfigGeneration.Applied.IsNull()) || uint64(nullableValue(t, detailAfterOld.ConfigGeneration.Applied)) != 3 {
		t.Fatalf("AppliedConfigGeneration regressed to %v, want 3", detailAfterOld.ConfigGeneration.Applied)
	}

	// 6. Userspace NetBird backend fails safety even when report claims SafetyStatusPassed.
	repUserspace := makeReport(uuid.New(), obsTime1.Add(20*time.Second), 3, api.ObservationReportNetbirdBackendUserspace, api.HealthCheckStatusPassed)
	if _, err := svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, repUserspace); err != nil {
		t.Fatalf("SubmitObservation(repUserspace): %v", err)
	}
	detailUserspace, _, err := svc.GetNode(ctx, n.NodeID)
	if err != nil {
		t.Fatalf("GetNode after repUserspace: %v", err)
	}
	if detailUserspace.SafetyStatus != api.SafetyStatusFailed {
		t.Fatalf("SafetyStatus with userspace backend = %s, want failed", detailUserspace.SafetyStatus)
	}

	// 7. SweepStaleNodes marks node stale and offline without changing publication_intent.
	now = now.Add(100 * time.Second)
	sweep1, err := svc.SweepStaleNodes(ctx, DefaultStaleTimeout, DefaultOfflineTimeout)
	if err != nil {
		t.Fatalf("SweepStaleNodes(stale): %v", err)
	}
	if sweep1.MarkedStale != 1 || sweep1.MarkedOffline != 0 {
		t.Fatalf("SweepStaleNodes(stale) = %+v, want MarkedStale=1 MarkedOffline=0", sweep1)
	}
	now = now.Add(100 * time.Second)
	sweep2, err := svc.SweepStaleNodes(ctx, DefaultStaleTimeout, DefaultOfflineTimeout)
	if err != nil {
		t.Fatalf("SweepStaleNodes(offline): %v", err)
	}
	if sweep2.MarkedOffline != 1 {
		t.Fatalf("SweepStaleNodes(offline) = %+v, want MarkedOffline=1", sweep2)
	}
}

func TestRevokeAndRetire_TombstonePreservesLedgerAndPendingWithdrawal(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	svc := NewService(pool)

	relID, grpID := seedTestReleaseAndGroup(t, ctx, pool, "retire")
	nRevoke := seedTestNode(t, ctx, pool, "tw-taipei-revoke", "tw-tpe", api.PublicationIntentPublished, "active", api.ControlStatusOnline, relID, grpID)
	nRetire := seedTestNode(t, ctx, pool, "tw-taipei-retire", "tw-tpe", api.PublicationIntentPublished, "active", api.ControlStatusOffline, relID, grpID)

	// Mark nRetire as having member_forwarding_observed = true (unconfirmed withdrawal) and a usage_events row.
	if _, err := pool.Exec(ctx, `UPDATE nodes SET member_forwarding_observed = true, resource_version = resource_version + 1 WHERE id = $1`, nRetire.NodeID); err != nil {
		t.Fatalf("update member_forwarding_observed: %v", err)
	}
	usageHash := sha256.Sum256([]byte("usage-event-1"))
	if _, err := pool.Exec(ctx, `
		INSERT INTO usage_events (node_id, incarnation_id, event_seq, schema_version, payload_hash, record_count)
		VALUES ($1, $2, 1, 1, $3, 0)`, nRetire.NodeID, nRetire.IncarnationID, usageHash[:]); err != nil {
		t.Fatalf("insert usage_event: %v", err)
	}

	// 1. Revoke nRevoke.
	_, etagRev, _ := svc.GetNode(ctx, nRevoke.NodeID)
	opRev, _, err := svc.RevokeNode(ctx, RevokeNodeParams{
		NodeID:         nRevoke.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etagRev,
		Request:        api.ReasonRequest{Reason: "compromised host key"},
	})
	if err != nil {
		t.Fatalf("RevokeNode: %v", err)
	}
	if string(opRev.Kind) != OpKindRevoke {
		t.Errorf("opRev.Kind = %s, want node.revoke", opRev.Kind)
	}
	revDetail, _, err := svc.GetNode(ctx, nRevoke.NodeID)
	if err != nil {
		t.Fatalf("GetNode(nRevoke): %v", err)
	}
	if revDetail.LifecycleState != api.LifecycleStateRevoked || revDetail.PublicationIntent != api.PublicationIntentDisabled {
		t.Errorf("revDetail = (%s, %s), want (revoked, disabled)", revDetail.LifecycleState, revDetail.PublicationIntent)
	}

	// 2. Retire nRetire (DELETE /admin/nodes/{id}) -> tombstone keeping usage_events & audit_events intact.
	_, etagRet, _ := svc.GetNode(ctx, nRetire.NodeID)
	opRet, _, err := svc.RetireNode(ctx, RetireNodeParams{
		NodeID:         nRetire.NodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etagRet,
		Reason:         "decommissioned hardware",
	})
	if err != nil {
		t.Fatalf("RetireNode: %v", err)
	}
	if string(opRet.Kind) != OpKindRetire || opRet.Status != api.OperationStatusPending {
		t.Errorf("opRet = (%s, %s), want (node.retire, pending)", opRet.Kind, opRet.Status)
	}

	retDetail, _, err := svc.GetNode(ctx, nRetire.NodeID)
	if err != nil {
		t.Fatalf("GetNode(nRetire) after retire: %v", err)
	}
	if retDetail.LifecycleState != api.LifecycleStateRetired || (!retDetail.RetiredAt.IsSpecified() || retDetail.RetiredAt.IsNull()) {
		t.Errorf("retDetail = (%s, retired_at=%v), want (retired, non-nil)", retDetail.LifecycleState, retDetail.RetiredAt)
	}
	if retDetail.PublicationIntent != api.PublicationIntentDisabled {
		t.Errorf("retDetail.PublicationIntent = %s, want disabled", retDetail.PublicationIntent)
	}

	// Verify usage_events and audit_events references remain intact.
	var usageCount, auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id = $1`, nRetire.NodeID).Scan(&usageCount); err != nil || usageCount != 1 {
		t.Fatalf("usage_events count after retire = %d (err=%v), want 1", usageCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE resource = $1`, "node:"+nRetire.NodeID.String()).Scan(&auditCount); err != nil || auditCount < 1 {
		t.Fatalf("audit_events count after retire = %d (err=%v), want >= 1", auditCount, err)
	}
}

func TestAT32_CentralEnrollConfigureObserveLifecycle(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 13, 0, 0, 0, time.UTC)
	svc := NewService(pool, WithClock(func() time.Time { return now }))

	relID, grpID := seedTestReleaseAndGroup(t, ctx, pool, "at32")

	// Step 1: Enroll fixture (node starts registered, provisioning=pending, desired_config_generation=1, applied=0).
	nodeID := uuid.New()
	incID := uuid.New()
	exitID := uuid.New()
	fp := sha256.Sum256([]byte("at32-pubkey"))
	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes (
			id, lifecycle_state, provisioning_state,
			desired_config_generation, applied_config_generation,
			desired_release_id, control_status, data_health, safety_status
		) VALUES (
			$1, 'registered', 'pending', 1, 0, $2, 'offline', 'unknown', 'unknown'
		)`, nodeID, relID); err != nil {
		t.Fatalf("insert node: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO node_incarnations (id, node_id, public_key_fingerprint, deployment_id, peer_id, os, os_version, arch)
		VALUES ($1, $2, $3, 'nb-main', 'peer-at32', 'ubuntu', '24.04', 'amd64')`, incID, nodeID, fp[:]); err != nil {
		t.Fatalf("insert incarnation: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $1, resource_version = resource_version + 1 WHERE id = $2`, incID, nodeID); err != nil {
		t.Fatalf("update current_incarnation_id: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO exits (id, node_id, display_name, region, publication_intent, publication_observed)
		VALUES ($1, $2, 'tw-at32', 'tw-tpe', 'unpublished', 'absent')`, exitID, nodeID); err != nil {
		t.Fatalf("insert exit: %v", err)
	}

	// Step 2: Admin configures groups and capacity via PATCH.
	_, etag1, err := svc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode initial: %v", err)
	}
	bw := int64(1_000_000_000)
	groups := []uuid.UUID{grpID}
	patchRes, err := svc.UpdateNode(ctx, UpdateNodeParams{
		NodeID:         nodeID,
		Principal:      "admin@example.com",
		IdempotencyKey: uuid.New(),
		IfMatch:        etag1,
		Patch: api.NodePatch{
			MemberGroupIds: &groups,
			Capacity: &api.NodeCapacity{DownBps: bw, UpBps: bw,
				MemberDownBpsCap: presence.NewNullNullable[int64](),
				MemberUpBpsCap:   presence.NewNullNullable[int64]()},
		},
	})
	if err != nil || patchRes.Status != 202 {
		t.Fatalf("UpdateNode config = (%+v, %v), want 202", patchRes, err)
	}

	// Verify desired vs observed are distinct before agent observation arrives.
	beforeObs, _, err := svc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode before observation: %v", err)
	}
	if (!beforeObs.Releases.DesiredReleaseId.IsSpecified() || beforeObs.Releases.DesiredReleaseId.IsNull()) || nullableValue(t, beforeObs.Releases.DesiredReleaseId) != relID {
		t.Errorf("DesiredReleaseId = %v, want %s", beforeObs.Releases.DesiredReleaseId, relID)
	}
	if !beforeObs.Releases.CurrentReleaseId.IsNull() {
		t.Errorf("CurrentReleaseId = %v, want nil before first observation", beforeObs.Releases.CurrentReleaseId)
	}
	if beforeObs.ControlStatus != api.ControlStatusOffline {
		t.Errorf("ControlStatus = %s, want offline before observation", beforeObs.ControlStatus)
	}

	// Step 3: Agent sends observation reporting ready state at generation 2.
	gen2 := ParseU64(2)
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
	rep.AppliedConfigGeneration.Set(gen2)
	rep.ProvisioningState = api.ProvisioningStateReady
	rep.SafetyStatus = api.SafetyStatusPassed
	rep.DataHealth = api.DataHealthHealthy
	rep.MemberForwardingEnabled = false
	rep.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	rep.Checks = []api.NodeCheck{
		{Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusPassed, ObservedAt: now,
			ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
		{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: now,
			ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
	}

	if _, err := svc.SubmitObservation(ctx, nodeID, incID, rep); err != nil {
		t.Fatalf("SubmitObservation: %v", err)
	}

	afterObs, _, err := svc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode after observation: %v", err)
	}
	if afterObs.ControlStatus != api.ControlStatusOnline || afterObs.ProvisioningState != api.ProvisioningStateReady || afterObs.SafetyStatus != api.SafetyStatusPassed {
		b, _ := json.MarshalIndent(afterObs, "", "  ")
		t.Fatalf("unexpected node state after observation: %s", fmt.Sprintf("%s", b))
	}
	if (!afterObs.Releases.CurrentReleaseId.IsSpecified() || afterObs.Releases.CurrentReleaseId.IsNull()) || nullableValue(t, afterObs.Releases.CurrentReleaseId) != relID {
		t.Errorf("CurrentReleaseId = %v, want %s", afterObs.Releases.CurrentReleaseId, relID)
	}
	if afterObs.PublicationIntent != api.PublicationIntentUnpublished || afterObs.PublicationObserved != api.PublicationObservedAbsent {
		t.Errorf("Publication = (%s, %s), want (unpublished, absent)", afterObs.PublicationIntent, afterObs.PublicationObserved)
	}
}

func nullableValue[T any](t *testing.T, n interface{ Get() (T, error) }) T {
	t.Helper()
	v, err := n.Get()
	if err != nil {
		t.Fatal(err)
	}
	return v
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

func assertExplicitNulls(t *testing.T, value any, fields ...string) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %T: %v", value, err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	for _, field := range fields {
		if string(object[field]) != "null" {
			t.Errorf("%T.%s = %s, want explicit null", value, field, object[field])
		}
	}
}

func TestMemberGrantOperation_NodeEnforcementACKAndLookup(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	svc := NewService(pool, WithClock(func() time.Time { return now }))

	relID, grpID := seedTestReleaseAndGroup(t, ctx, pool, "grant-ack")
	n := seedTestNode(t, ctx, pool, "tw-taipei-grant", "tw-tpe", api.PublicationIntentPublished, "active", api.ControlStatusOnline, relID, grpID)

	memberID := uuid.New()
	grantID := uuid.New()
	opID := uuid.New()
	adminID := uuid.New()
	reqID := uuid.New()
	validFrom := now.Add(-1 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	if _, err := pool.Exec(ctx, `INSERT INTO members (id, email, display_name, status) VALUES ($1, 'grant@example.com', 'Grant Member', 'active')`, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlement_grants (
			id, member_id, source, valid_from, valid_until, reason, created_by
		) VALUES ($1, $2, 'admin_grant', $3, $4, 'TEST ONLY grant', $5)`,
		grantID, memberID, validFrom, validUntil, adminID.String()); err != nil {
		t.Fatalf("insert entitlement_grant: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlements (
			member_id, revision, state, device_limit, valid_from, effective_service_end, admin_suspended, projection
		) VALUES (
			$1, 2, 'active', 3, $2, $3, false, jsonb_build_object('exit_group_ids', to_jsonb(ARRAY[$4::text]))
		)`, memberID, validFrom, validUntil, grpID.String()); err != nil {
		t.Fatalf("insert entitlement: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO operations (
			id, kind, principal, resource_type, resource_id, desired_generation, request_id, steps
		) VALUES (
			$1, 'member.grant', $2, 'entitlement_grant', $3, 2, $4,
			'{"netbird":{"required":true,"status":"pending"},"nodes":{"required":true,"status":"pending"}}'::jsonb
		)`, opID, "admin:"+adminID.String(), grantID, reqID); err != nil {
		t.Fatalf("insert member.grant operation: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE entitlement_grants SET operation_id = $2 WHERE id = $1`, grantID, opID); err != nil {
		t.Fatalf("link grant operation_id: %v", err)
	}

	// 1. Lookup by memberID follows resource_type = 'entitlement_grant' through entitlement_grants.member_id.
	ops, err := svc.ListMemberOperations(ctx, memberID)
	if err != nil {
		t.Fatalf("ListMemberOperations: %v", err)
	}
	if len(ops) != 1 || ops[0].OperationId != opID || ops[0].Resource.Type != api.ResourceTypeEntitlementGrant || ops[0].Resource.Id != grantID {
		t.Fatalf("ListMemberOperations = %+v, want [op %s on entitlement_grant %s]", ops, opID, grantID)
	}
	gotOp, err := svc.GetMemberOperation(ctx, memberID, opID)
	if err != nil || gotOp.OperationId != opID {
		t.Fatalf("GetMemberOperation = (%+v, %v), want %s", gotOp, err, opID)
	}
	if _, err := svc.GetMemberOperation(ctx, uuid.New(), opID); err == nil {
		t.Fatalf("expected GetMemberOperation for unrelated member to return not found")
	}

	// 2. Simulate NetBird step succeeding first; operation remains running while "nodes" step is pending.
	if err := db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		evID := uuid.Must(uuid.NewV7())
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			ID:            evID,
			Actor:         SystemActorUUID.String(),
			ActorType:     "system",
			Action:        "netbird.observed",
			Resource:      "member:" + memberID.String(),
			Result:        "succeeded",
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         []byte(`{"observed":true}`),
		}); err != nil {
			return err
		}
		return entitlement.TransitionOperation(ctx, tx, opID, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{
			Reference:  "audit_event:" + evID.String(),
			ObservedAt: now,
		})
	}); err != nil {
		t.Fatalf("TransitionOperation(netbird): %v", err)
	}

	midOp, err := svc.GetMemberOperation(ctx, memberID, opID)
	if err != nil {
		t.Fatalf("GetMemberOperation after netbird: %v", err)
	}
	if midOp.Status != api.OperationStatusRunning {
		t.Fatalf("op status after netbird only = %s, want running", midOp.Status)
	}

	// 3. Submit node observation with applied_config_generation >= desired_config_generation and safety=passed.
	report := emptyObservationReport()
	report.AgentVersion = "1.0.0"
	report.BootId = uuid.New()
	report.ReportId = uuid.New()
	report.NodeId = n.NodeID
	report.NodeIncarnationId = n.IncarnationID
	report.ObservedAt = now
	report.ProvisioningState = api.ProvisioningStateReady
	report.SafetyStatus = api.SafetyStatusPassed
	report.DataHealth = api.DataHealthHealthy
	report.MemberForwardingEnabled = true
	report.AppliedConfigGeneration = presence.NewNullableWithValue(ParseU64(2))
	report.CurrentReleaseId = presence.NewNullableWithValue(relID)
	report.LastGoodReleaseId = presence.NewNullableWithValue(relID)
	report.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	report.Checks = []api.NodeCheck{
		{Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusPassed, ObservedAt: now, ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
		{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: now, ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
		{Name: api.NodeCheckNameTunnelProbe, Status: api.HealthCheckStatusPassed, ObservedAt: now, ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
	}
	if _, err := svc.SubmitObservation(ctx, n.NodeID, n.IncarnationID, report); err != nil {
		t.Fatalf("SubmitObservation: %v", err)
	}

	finalOp, err := svc.GetMemberOperation(ctx, memberID, opID)
	if err != nil {
		t.Fatalf("GetMemberOperation after node observation: %v", err)
	}
	if finalOp.Status != api.OperationStatusSucceeded {
		t.Fatalf("finalOp.Status = %s, want succeeded (steps=%+v)", finalOp.Status, finalOp.Steps)
	}
	for _, st := range finalOp.Steps {
		if st.Status != api.OperationStepStatusSucceeded || st.ObservedAt.IsNull() {
			t.Fatalf("step %s = %+v, want succeeded with non-null ObservedAt", st.Name, st)
		}
	}
}
