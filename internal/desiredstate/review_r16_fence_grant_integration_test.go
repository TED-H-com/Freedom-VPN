//go:build integration

package desiredstate

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReviewR16_FencedDeviceReceivesNoGrant proves an authorization_unconfirmed
// device (a device_authorization_fences row, public status pending) is absent
// from signed desired-state bindings and from lease device ids. A sibling
// authorized device on the same member must still be granted, and a revoked
// device must stay absent, so a fixture miss cannot look like a pass.
func TestReviewR16_FencedDeviceReceivesNoGrant(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	group := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, $2, $2)`, group, "r16-fence-"+group.String()[:8]); err != nil {
		t.Fatal(err)
	}
	plan, planRev := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, $2)`, plan, "r16-fence-"+plan.String()[:8]); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO plan_revisions (id, plan_id, revision, device_limit, exit_group_ids, schema_version, payload)
		VALUES ($1, $2, 1, 3, $3, 1, '{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`,
		planRev, plan, []uuid.UUID{group}); err != nil {
		t.Fatal(err)
	}
	member := uuid.New()
	validFrom, validUntil := now.Add(-24*time.Hour), now.Add(14*24*time.Hour)
	if _, err := pool.Exec(ctx, `INSERT INTO members (id, email, display_name, status) VALUES ($1, $2, $2, 'active')`, member, member.String()+"@example.invalid"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlements (
			member_id, revision, state, plan_revision_id, device_limit,
			valid_from, effective_service_end, admin_suspended, projection
		) VALUES (
			$1, 2, 'active', $2, 3, $3, $4, false,
			jsonb_build_object('exit_group_ids', to_jsonb(ARRAY[$5::text]))
		)`, member, planRev, validFrom, validUntil, group.String()); err != nil {
		t.Fatal(err)
	}

	insertDevice := func(name, ip, status string, fence bool) uuid.UUID {
		t.Helper()
		id := uuid.New()
		peer := "peer-" + id.String()[:8]
		if _, err := pool.Exec(ctx, `
			INSERT INTO member_devices (id, member_id, deployment_id, peer_id, name, status, authorized_at)
			VALUES ($1, $2, 'nb-main', $3, $4, $5, $6)`, id, member, peer, name, status, validFrom); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO device_bindings (device_id, member_id, overlay_scope, overlay_ip, binding_revision, valid_from, valid_until, peer_ref)
			VALUES ($1, $2, 'overlay-main', $3, 1, $4, $5, $6)`, id, member, ip, validFrom, validUntil, peer); err != nil {
			t.Fatal(err)
		}
		if fence {
			if _, err := pool.Exec(ctx, `INSERT INTO device_authorization_fences (device_id, fenced_at, observed_group_ids) VALUES ($1, $2, '{}')`, id, now); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO device_authorization_observations (device_id, observed_authorized, observed_at, observed_version, observed_entitlement_revision)
				VALUES ($1, false, $2, 1, 2)`, id, now); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	open := insertDevice("open", "100.64.0.41", "authorized", false)
	fenced := insertDevice("fenced", "100.64.0.42", "authorized", true)
	revoked := insertDevice("revoked", "100.64.0.43", "revoked", false)

	nodeID, _, _, relID := seedCompilerTestNode(t, ctx, pool, group, api.PublicationIntentPublished, nil)
	signer, _ := testDesiredStateSignerAndTrustStore(t, now)
	compiler := NewCompiler(pool, signer,
		WithClock(func() time.Time { return now }),
		WithEntitlementReader(NewDBEntitlementReader(pool)),
		WithPolicyReader(NewFakePolicyReader(PolicySnapshot{
			DesiredReleaseID:  &relID,
			ServingReleaseIDs: []uuid.UUID{relID},
			AccountingPeriod:  DefaultAccountingPeriod(now),
		})))
	compiled, err := compiler.Compile(ctx, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	bound := map[uuid.UUID]bool{}
	for _, b := range compiled.Envelope.Signed.DeviceBindings {
		bound[b.DeviceId] = true
	}
	leasesOut, err := (leases.DBEntitlementReader{}).ListLeaseEntitlementsForGroups(ctx, pool, []uuid.UUID{group}, now)
	if err != nil {
		t.Fatal(err)
	}
	leased := map[uuid.UUID]bool{}
	for _, m := range leasesOut {
		if m.MemberID != member {
			continue
		}
		for _, id := range m.DeviceIDs {
			leased[id] = true
		}
	}
	if !bound[open] || !leased[open] {
		t.Fatalf("fixture missed the unfenced device: bound=%v leased=%v", bound[open], leased[open])
	}
	if bound[revoked] || leased[revoked] {
		t.Fatalf("revoked device was granted: bound=%v leased=%v", bound[revoked], leased[revoked])
	}
	if bound[fenced] || leased[fenced] {
		t.Fatalf("TODO(review): authorization_unconfirmed device %s is still granted (desired-state binding=%v, lease device id=%v) while member_devices.status stays authorized", fenced, bound[fenced], leased[fenced])
	}
}
