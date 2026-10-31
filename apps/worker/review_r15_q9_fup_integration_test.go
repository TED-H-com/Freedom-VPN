//go:build integration

package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	policyStore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type reviewFairUse struct {
	pool    *pgxpool.Pool
	ctx     context.Context
	now     time.Time
	member  uuid.UUID
	carry   uuid.UUID
	carryIn uuid.UUID
	other   uuid.UUID
	period  string
	svc     *PoliciesServices
}

func reviewFairUseWorld(t *testing.T) reviewFairUse {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	groupCarry := uuid.New()
	groupOther := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, $2, $2), ($3, $4, $4)`,
		groupCarry, "carry-"+uuid.NewString()[:8], groupOther, "other-"+uuid.NewString()[:8]); err != nil {
		t.Fatal(err)
	}
	carry, inc := seedFairUseExit(t, ctx, pool, groupCarry, api.PublicationIntentPublished)
	other, _ := seedFairUseExit(t, ctx, pool, groupOther, api.PublicationIntentPublished)
	memberID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO members (id, email, display_name, status) VALUES ($1, $2, 'Fair-Use Member', 'active')`, memberID, memberID.String()+"@example.com"); err != nil {
		t.Fatal(err)
	}
	planRevID := insertFairUsePlan(t, ctx, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlements (
			member_id, revision, state, plan_revision_id, device_limit,
			valid_from, effective_service_end, admin_suspended, computed_at, projection
		) VALUES (
			$1, 1, 'active', $2, 3,
			$3, $4, false, $3, jsonb_build_object('exit_group_ids', to_jsonb(ARRAY[$5::text]))
		)`, memberID, planRevID, now.Add(-24*time.Hour), now.Add(10*24*time.Hour), groupCarry.String()); err != nil {
		t.Fatal(err)
	}
	adminID := uuid.New()
	store := policyStore.NewStore(pool)
	payload, err := json.Marshal(testEnforceFairUsePayload())
	if err != nil {
		t.Fatal(err)
	}
	code := "fup-" + uuid.NewString()[:8]
	rev, err := store.CreateRevision(ctx, policyStore.CreateRevisionInput{
		Type: api.PolicyTypeFairUse, Code: &code, SchemaVersion: 1, Payload: payload,
		Reason: "TEST ONLY enforce fair use", ActorID: adminID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateRevision(ctx, policyStore.ActivateRevisionInput{
		DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1",
		Reason: "activate enforce fair use", ActorID: adminID,
	}); err != nil {
		t.Fatal(err)
	}
	compiler := desiredstate.NewCompiler(pool, testFairUseDesiredStateSigner(t))
	services, err := NewPoliciesServices(pool, policiestest.New(t).Policies, compiler)
	if err != nil {
		t.Fatal(err)
	}
	jobCtx := WithPolicies(ctx, services)
	activated := loadLatestOutbox(t, ctx, pool, "policy.activated")
	if _, err = policyJob(jobCtx, activated); err != nil {
		t.Fatal(err)
	}
	if _, err = policyJob(jobCtx, activated); err != nil {
		t.Fatal(err)
	}
	assertDesiredGens(t, ctx, pool, map[uuid.UUID]int64{carry: 1, other: 0})
	return reviewFairUse{pool: pool, ctx: ctx, now: now, member: memberID, carry: carry, carryIn: inc, other: other, period: fup.CurrentUTCMonthPeriodID(now), svc: services}
}

// TestReviewR15_FrozenCarryingExitKeepsFairUseApplying acks the carrying exit,
// then freezes it. D-091 requires current node safety even before another
// observation arrives; a historical passing observation cannot ACK a frozen exit.
func TestReviewR15_FrozenCarryingExitKeepsFairUseApplying(t *testing.T) {
	w := reviewFairUseWorld(t)
	nodeSvc := nodelib.NewService(w.pool)
	submitAppliedObservation(t, w.ctx, nodeSvc, w.carry, w.carryIn, 1, w.now)
	st, err := w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if st.MemberStatus.Applying || st.PendingNodes != 0 || st.AckedNodes != 1 {
		t.Fatalf("before freeze applying=%v pending=%d acked=%d summary=%q", st.MemberStatus.Applying, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}
	sum := sha256.Sum256([]byte("frozen-" + w.carryIn.String()))
	if _, err = w.pool.Exec(w.ctx, `INSERT INTO node_identity_endpoints (incarnation_id, endpoint_hash, last_seen_at, mapping_frozen) VALUES ($1, $2, $3, true)`, w.carryIn, sum[:], w.now); err != nil {
		t.Fatal(err)
	}
	var historicalSafety string
	if err = w.pool.QueryRow(w.ctx, `SELECT safety_status FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC, id DESC LIMIT 1`, w.carry).Scan(&historicalSafety); err != nil {
		t.Fatal(err)
	}
	st, err = w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if historicalSafety != string(api.SafetyStatusPassed) || !st.MemberStatus.Applying || st.TotalServingNodes != 1 || st.PendingNodes != 1 || st.AckedNodes != 0 || st.NodeSyncSummary != "pending on 1 node (1 frozen)" {
		t.Fatalf("freeze before another observation: historical safety=%s applying=%v total=%d pending=%d acked=%d summary=%q", historicalSafety, st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}
	submitAppliedObservation(t, w.ctx, nodeSvc, w.carry, w.carryIn, 1, w.now.Add(time.Second))
	var safety, obsSafety string
	if err = w.pool.QueryRow(w.ctx, `SELECT safety_status FROM nodes WHERE id=$1`, w.carry).Scan(&safety); err != nil {
		t.Fatal(err)
	}
	if err = w.pool.QueryRow(w.ctx, `SELECT safety_status FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC, id DESC LIMIT 1`, w.carry).Scan(&obsSafety); err != nil {
		t.Fatal(err)
	}
	st, err = w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if safety != string(api.SafetyStatusFailed) || obsSafety != string(api.SafetyStatusFailed) || !st.MemberStatus.Applying || st.TotalServingNodes != 1 || st.PendingNodes != 1 || st.AckedNodes != 0 || st.NodeSyncSummary != "pending on 1 node (1 frozen)" {
		t.Fatalf("frozen exit safety=%s observation=%s applying=%v pending=%d acked=%d, want the node failed and applying still set", safety, obsSafety, st.MemberStatus.Applying, st.PendingNodes, st.AckedNodes)
	}
	var other int64
	if err = w.pool.QueryRow(w.ctx, `SELECT desired_config_generation::bigint FROM nodes WHERE id=$1`, w.other).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if other != 0 {
		t.Fatalf("unrelated exit generation = %d, want 0", other)
	}
}

// TestReviewR15_UnpublishDropsTheCarryingExitAndApplyingSticks unpublishes the
// only exit that carries the member after that exit has acked. D-091 requires
// applying=false with zero carrying exits, regardless of other registered nodes.
func TestReviewR15_UnpublishDropsTheCarryingExitAndApplyingSticks(t *testing.T) {
	w := reviewFairUseWorld(t)
	nodeSvc := nodelib.NewService(w.pool)
	submitAppliedObservation(t, w.ctx, nodeSvc, w.carry, w.carryIn, 1, w.now)
	st, err := w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if st.MemberStatus.Applying {
		t.Fatalf("acked carrying exit left applying true: %+v", st.MemberStatus)
	}
	if _, err = w.pool.Exec(w.ctx, `UPDATE exits SET publication_intent='unpublished', resource_version=resource_version+1 WHERE node_id=$1`, w.carry); err != nil {
		t.Fatal(err)
	}
	st, err = w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if st.MemberStatus.Applying || st.TotalServingNodes != 0 || st.PendingNodes != 0 || st.AckedNodes != 0 || st.NodeSyncSummary != "no published exit carries this member" {
		t.Fatalf("after unpublish: applying=%v total=%d pending=%d acked=%d summary=%q, want no carrying exits and applying=false", st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}
	// A later observation of the unpublished exit must not resurrect applying.
	submitAppliedObservation(t, w.ctx, nodeSvc, w.carry, w.carryIn, 1, w.now.Add(time.Second))
	st, err = w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if st.MemberStatus.Applying || st.TotalServingNodes != 0 || st.PendingNodes != 0 || st.AckedNodes != 0 || st.NodeSyncSummary != "no published exit carries this member" {
		t.Fatalf("unpublished exit observation: applying=%v total=%d pending=%d acked=%d summary=%q", st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}
	assertDesiredGens(t, w.ctx, w.pool, map[uuid.UUID]int64{w.carry: 1, w.other: 0})
}

// FUP03 / D-091: safety changes without a new observation must invalidate the
// earlier ACK independently of the incarnation's mapping freeze.
func TestFUP03_CarryingExitRequiresCurrentNodeSafety(t *testing.T) {
	w := reviewFairUseWorld(t)
	submitAppliedObservation(t, w.ctx, nodelib.NewService(w.pool), w.carry, w.carryIn, 1, w.now)
	for _, safety := range []string{"failed", "unknown", "expired", "passed"} {
		t.Run(safety, func(t *testing.T) {
			if _, err := w.pool.Exec(w.ctx, `UPDATE nodes SET safety_status=$2, resource_version=resource_version+1 WHERE id=$1`, w.carry, safety); err != nil {
				t.Fatal(err)
			}
			st, err := w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
			if err != nil {
				t.Fatal(err)
			}
			wantPending, wantAcked, wantSummary := 1, 0, "pending on 1 node (1 unsafe)"
			if safety == "passed" {
				wantPending, wantAcked, wantSummary = 0, 1, "applied on 1 node"
			}
			if st.MemberStatus.Applying != (wantPending > 0) || st.TotalServingNodes != 1 || st.PendingNodes != wantPending || st.AckedNodes != wantAcked || st.NodeSyncSummary != wantSummary {
				t.Fatalf("node safety=%s applying=%v total=%d pending=%d acked=%d summary=%q", safety, st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
			}
		})
	}
}

func TestFUP03_CompilationPendingOnlyForCarryingExitsWithoutEnvelopes(t *testing.T) {
	w := reviewFairUseWorld(t)
	nodeSvc := nodelib.NewService(w.pool)
	submitAppliedObservation(t, w.ctx, nodeSvc, w.carry, w.carryIn, 1, w.now)
	// The other published exit now carries the member but has no envelope.
	if _, err := w.pool.Exec(w.ctx, `INSERT INTO exit_groups (exit_id, group_id)
		SELECT e.id, eg.group_id FROM exits e
		JOIN exits carrying ON carrying.node_id=$1
		JOIN exit_groups eg ON eg.exit_id=carrying.id
		WHERE e.node_id=$2`, w.carry, w.other); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(w.ctx, `UPDATE nodes SET safety_status='passed', resource_version=resource_version+1 WHERE id=$1`, w.other); err != nil {
		t.Fatal(err)
	}
	st, err := w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if !st.MemberStatus.Applying || st.TotalServingNodes != 2 || st.PendingNodes != 1 || st.AckedNodes != 1 || st.NodeSyncSummary != "pending on 1 node" {
		t.Fatalf("one envelope missing: applying=%v total=%d pending=%d acked=%d summary=%q", st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}
	// Simulate a carrying set for which no current envelope has been compiled.
	if _, err := w.pool.Exec(w.ctx, `DELETE FROM desired_state_envelopes WHERE node_id=$1`, w.carry); err != nil {
		t.Fatal(err)
	}
	st, err = w.svc.FUPEngine.GetAdminMemberStatus(w.ctx, w.member, w.period)
	if err != nil {
		t.Fatal(err)
	}
	if !st.MemberStatus.Applying || st.TotalServingNodes != 2 || st.PendingNodes != 2 || st.AckedNodes != 0 || st.NodeSyncSummary != "pending desired-state compilation" {
		t.Fatalf("all envelopes missing: applying=%v total=%d pending=%d acked=%d summary=%q", st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}
}
