//go:build integration

package worker

import (
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// AT29 / D-077: both withdrawal event forms preserve fences, reconcile signed
// bindings before the job receipt, and require independent enforcement evidence.
func TestAT29_D077_GrantWithdrawalReconcilesFenceBeforeReceipt(t *testing.T) {
	for _, unchanged := range []bool{false, true} {
		kind := "member.entitlement_changed"
		if unchanged {
			kind = "member.grant_revoked"
		}
		t.Run(kind, func(t *testing.T) {
			f := newMemberPolicyFixture(t, true)
			app, err := Open(t.Context(), Dependencies{Config: f.config, Pool: f.pool}, f.memberDomain(), nodes())
			if err != nil {
				t.Fatal(err)
			}
			f.assertCompleted(f.dispatch(app, "member.entitlement_changed"), 1, 1, 0)
			device, err := f.services.HTTP.Devices.AuthorizeDevice(t.Context(), memberPolicyCommand("system", uuid.New()), f.peer.peer)
			if err != nil {
				t.Fatal(err)
			}
			f.assertCompleted(f.dispatch(app, "device.authorized"), 2, 1, 1)
			var grant, revision uuid.UUID
			var groups []uuid.UUID
			if err = f.pool.QueryRow(t.Context(), `SELECT g.id,g.plan_revision_id,r.exit_group_ids FROM entitlement_grants g JOIN plan_revisions r ON r.id=g.plan_revision_id WHERE g.member_id=$1`, f.member).Scan(&grant, &revision, &groups); err != nil {
				t.Fatal(err)
			}
			if unchanged {
				// Disconnected future rights do not change today's projection.
				start := time.Now().UTC().Add(18 * time.Hour)
				future, err := f.services.HTTP.Members.Grant(t.Context(), memberPolicyCommand("admin", uuid.New()), f.member, api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: start.Add(2 * time.Hour), Reason: "TEST ONLY future withdrawal", PlanRevisionId: &revision})
				if err != nil {
					t.Fatal(err)
				}
				grant = future.ResourceID
			}
			// Persist a real D-077 fence, then simulate a worker interruption before
			// its signed binding is reconciled. Withdrawal must recover that work.
			f.peer.peer.Authorized, f.peer.peer.Groups = false, nil
			if err = f.services.HTTP.Devices.SyncDevice(t.Context(), device.DeviceID); err != nil {
				t.Fatal(err)
			}
			var fencedAt time.Time
			if err = f.pool.QueryRow(t.Context(), `SELECT fenced_at FROM device_authorization_fences WHERE device_id=$1`, device.DeviceID).Scan(&fencedAt); err != nil {
				t.Fatal(err)
			}
			withdrawal, err := f.services.HTTP.Members.RevokeMemberGrant(t.Context(), memberPolicyCommand("admin", uuid.New()), f.member, grant, "TEST ONLY fenced member withdrawal")
			if err != nil || withdrawal.OperationID == nil {
				t.Fatal(withdrawal, err)
			}
			f.services.Compiler = fenceCompileFailure{}
			job := f.dispatch(app, kind)
			var retryable bool
			if err = f.pool.QueryRow(t.Context(), `SELECT delivered_at IS NULL AND delivery_receipt IS NULL AND dead_lettered_at IS NULL AND NOT EXISTS(SELECT 1 FROM audit_events WHERE id=$1) FROM outbox_events WHERE id=$1`, job).Scan(&retryable); err != nil || !retryable {
				t.Fatal("failed fence compilation acknowledged withdrawal job", retryable, err)
			}
			if state := fenceState(t, f, f.node); state.ConfigGeneration != 2 || len(state.DeviceBindings) != 1 {
				t.Fatal("fault injection did not leave a signed binding to retry", state)
			}
			f.services.Compiler = nil
			if retried := f.dispatch(app, kind); retried != job {
				t.Fatal("withdrawal retry replaced the original job")
			}
			var acknowledged, audited bool
			if err = f.pool.QueryRow(t.Context(), `SELECT attempt=2 AND delivered_at IS NOT NULL AND delivery_receipt='audit_event:'||id::text,EXISTS(SELECT 1 FROM audit_events a WHERE a.id=o.id AND a.action='member.policy_compiled' AND (a.after_diff->>'compiled_nodes')::int=1) FROM outbox_events o WHERE id=$1`, job).Scan(&acknowledged, &audited); err != nil || !acknowledged || !audited {
				t.Fatal("withdrawal retry lacks its policy receipt", acknowledged, audited, err)
			}
			state := fenceState(t, f, f.node)
			policies := 0
			if unchanged {
				policies = 1
			}
			if state.ConfigGeneration != 3 || len(state.DeviceBindings) != 0 || len(state.MemberRatePolicies) != policies {
				t.Fatal("withdrawal skipped fence fanout or compiled an exit twice", state)
			}
			var retained time.Time
			if err = f.pool.QueryRow(t.Context(), `SELECT fenced_at FROM device_authorization_fences WHERE device_id=$1`, device.DeviceID).Scan(&retained); err != nil || !retained.Equal(fencedAt) {
				t.Fatal("withdrawal changed the authorization fence", retained, err)
			}
			leased, err := (leases.DBEntitlementReader{}).ListLeaseEntitlementsForGroups(t.Context(), f.pool, groups, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			for _, member := range leased {
				if slices.Contains(member.DeviceIDs, device.DeviceID) {
					t.Fatal("withdrawal leased a fenced device")
				}
			}
			var observed, awaitingNodes bool
			if err = f.pool.QueryRow(t.Context(), `SELECT steps->'netbird'->>'status'='succeeded' AND steps->'netbird'->>'observed_at' IS NOT NULL,steps->'nodes'->>'status'='pending' AND steps->'nodes'->>'observed_at' IS NULL AND completed_at IS NULL AND status<>'succeeded' FROM operations WHERE id=$1`, withdrawal.OperationID).Scan(&observed, &awaitingNodes); err != nil || !observed || !awaitingNodes {
				t.Fatal("withdrawal skipped read-back or fabricated node enforcement", observed, awaitingNodes, err)
			}
		})
	}
}
