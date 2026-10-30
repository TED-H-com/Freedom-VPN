//go:build integration

package worker

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	policyStore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestFairUseActivationRecompileAndNodeAck covers activation and member-generation
// recompilation onto the exits that carry the member, replay idempotency, and
// fair_use.applying clearing only after each carrying exit acknowledges the
// compiled generation through the observation writer.
func TestFairUseActivationRecompileAndNodeAck(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	periodID := fup.CurrentUTCMonthPeriodID(now)

	groupCarry := uuid.New()
	groupOther := uuid.New()
	for _, g := range []struct {
		id   uuid.UUID
		code string
	}{
		{groupCarry, "carry-" + uuid.NewString()[:8]},
		{groupOther, "other-" + uuid.NewString()[:8]},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, $2, $2)`, g.id, g.code); err != nil {
			t.Fatalf("insert group: %v", err)
		}
	}

	carry1, inc1 := seedFairUseExit(t, ctx, pool, groupCarry, api.PublicationIntentPublished)
	carry2, inc2 := seedFairUseExit(t, ctx, pool, groupCarry, api.PublicationIntentPublished)
	other, _ := seedFairUseExit(t, ctx, pool, groupOther, api.PublicationIntentPublished)
	unpublished, _ := seedFairUseExit(t, ctx, pool, groupCarry, api.PublicationIntentUnpublished)

	memberID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, email, display_name, status)
		VALUES ($1, $2, 'Fair-Use Member', 'active')`, memberID, memberID.String()+"@example.com"); err != nil {
		t.Fatalf("insert member: %v", err)
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
		t.Fatalf("insert entitlement: %v", err)
	}

	adminID := uuid.New()
	store := policyStore.NewStore(pool)
	payload, err := json.Marshal(testEnforceFairUsePayload())
	if err != nil {
		t.Fatal(err)
	}
	code := "fup-" + uuid.NewString()[:8]
	rev, err := store.CreateRevision(ctx, policyStore.CreateRevisionInput{
		Type:          api.PolicyTypeFairUse,
		Code:          &code,
		SchemaVersion: 1,
		Payload:       payload,
		Reason:        "TEST ONLY enforce fair use",
		ActorID:       adminID,
	})
	if err != nil {
		t.Fatalf("CreateRevision: %v", err)
	}
	if _, err := store.ActivateRevision(ctx, policyStore.ActivateRevisionInput{
		DefinitionID:    rev.DefinitionID,
		RevisionID:      rev.RevisionID,
		ExpectedVersion: "1",
		Reason:          "activate enforce fair use",
		ActorID:         adminID,
	}); err != nil {
		t.Fatalf("ActivateRevision: %v", err)
	}

	signer := testFairUseDesiredStateSigner(t)
	compiler := desiredstate.NewCompiler(pool, signer)
	services, err := NewPoliciesServices(pool, policiestest.New(t).Policies, compiler)
	if err != nil {
		t.Fatal(err)
	}
	jobCtx := WithPolicies(ctx, services)

	activated := loadLatestOutbox(t, ctx, pool, "policy.activated")
	if _, err := policyJob(jobCtx, activated); err != nil {
		t.Fatalf("policy.activated: %v", err)
	}
	assertDesiredGens(t, ctx, pool, map[uuid.UUID]int64{
		carry1: 1, carry2: 1, other: 0, unpublished: 0,
	})
	assertMemberRate(t, ctx, pool, carry1, memberID, 1, 0, 100_000_000, 50_000_000)
	assertMemberRate(t, ctx, pool, carry2, memberID, 1, 0, 100_000_000, 50_000_000)
	firstGen := loadOutboxGeneration(t, ctx, pool, "1")
	if _, err := policyJob(jobCtx, firstGen); err != nil {
		t.Fatalf("generation changed after activation: %v", err)
	}
	if _, err := policyJob(jobCtx, activated); err != nil {
		t.Fatalf("replay policy.activated: %v", err)
	}
	assertDesiredGens(t, ctx, pool, map[uuid.UUID]int64{
		carry1: 1, carry2: 1, other: 0, unpublished: 0,
	})

	pStart, pEnd, err := fup.PeriodBoundsForUTCMonth(periodID)
	if err != nil {
		t.Fatal(err)
	}
	// TEST ONLY threshold: tier 1 starts at 100_000_000_000 bytes.
	if _, err := pool.Exec(ctx, `
		INSERT INTO usage_rollups (
			member_id, period, granularity, projection_generation, metering_version,
			upload_bytes, download_bytes, last_observed_at, aggregate_as_of, period_start, period_end
		) VALUES ($1, $2, 'month', 1, 'forward_admission_v1', 0, 100000000000, $3, $3, $4, $5)`,
		memberID, periodID, now, pStart, pEnd); err != nil {
		t.Fatalf("insert usage_rollups: %v", err)
	}
	eval, err := services.FUPEngine.Compute(ctx, fup.ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Compute tier 1: %v", err)
	}
	if eval.Generation != 2 || eval.EnforcedTierIndex != 1 || eval.EnforcedDownBps == nil || *eval.EnforcedDownBps != 25_000_000 {
		t.Fatalf("tier 1 eval = gen %d tier %d down %v, want gen 2 tier 1 down 25000000", eval.Generation, eval.EnforcedTierIndex, eval.EnforcedDownBps)
	}

	changed := loadOutboxGeneration(t, ctx, pool, "2")
	if _, err := compiler.RecompileBatch(ctx, []db.ClaimedEvent{changed, changed}); err != nil {
		t.Fatalf("duplicate generation batch: %v", err)
	}
	assertDesiredGens(t, ctx, pool, map[uuid.UUID]int64{
		carry1: 2, carry2: 2, other: 0, unpublished: 0,
	})
	assertMemberRate(t, ctx, pool, carry1, memberID, 2, 1, 25_000_000, 10_000_000)
	assertMemberRate(t, ctx, pool, carry2, memberID, 2, 1, 25_000_000, 10_000_000)
	if _, err := policyJob(jobCtx, changed); err != nil {
		t.Fatalf("replay generation changed: %v", err)
	}
	if _, err := policyJob(jobCtx, activated); err != nil {
		t.Fatalf("replay activation after tier change: %v", err)
	}
	assertDesiredGens(t, ctx, pool, map[uuid.UUID]int64{
		carry1: 2, carry2: 2, other: 0, unpublished: 0,
	})

	st, err := services.FUPEngine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatalf("status before ack: %v", err)
	}
	if !st.MemberStatus.Applying || st.TotalServingNodes != 2 || st.PendingNodes != 2 || st.AckedNodes != 0 {
		t.Fatalf("before ack applying=%v total=%d pending=%d acked=%d summary=%q, want applying on 2",
			st.MemberStatus.Applying, st.TotalServingNodes, st.PendingNodes, st.AckedNodes, st.NodeSyncSummary)
	}

	nodeSvc := nodelib.NewService(pool)
	submitAppliedObservation(t, ctx, nodeSvc, carry1, inc1, 2, now)
	st, err = services.FUPEngine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.MemberStatus.Applying || st.AckedNodes != 1 || st.PendingNodes != 1 {
		t.Fatalf("after one exit applying=%v acked=%d pending=%d, want still applying with one pending",
			st.MemberStatus.Applying, st.AckedNodes, st.PendingNodes)
	}
	submitAppliedObservation(t, ctx, nodeSvc, carry2, inc2, 2, now.Add(time.Second))
	st, err = services.FUPEngine.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	if st.MemberStatus.Applying || st.AckedNodes != 2 || st.PendingNodes != 0 || st.NodeSyncSummary != "applied on 2 nodes" {
		t.Fatalf("after both exits applying=%v acked=%d pending=%d summary=%q, want applied on 2 nodes",
			st.MemberStatus.Applying, st.AckedNodes, st.PendingNodes, st.NodeSyncSummary)
	}
}

func seedFairUseExit(t *testing.T, ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID, intent api.PublicationIntent) (uuid.UUID, uuid.UUID) {
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
		)`, relID, "2026.10.30-"+relID.String()[:8], manifestHash[:], sig); err != nil {
		t.Fatalf("insert release: %v", err)
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
			control_status, data_health, safety_status
		) VALUES (
			$1, 'registered', 'ready',
			0, 0,
			$2, $2, $2,
			'offline', 'unknown', 'unknown'
		)`, nodeID, relID); err != nil {
		t.Fatalf("insert node: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO node_incarnations (
			id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch
		) VALUES (
			$1, $2, $3, 'nb-main', $4, 'ubuntu', 'amd64'
		)`, incID, nodeID, fp[:], "peer-"+nodeID.String()[:8]); err != nil {
		t.Fatalf("insert incarnation: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $1, resource_version = resource_version + 1 WHERE id = $2`, incID, nodeID); err != nil {
		t.Fatalf("set incarnation: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO exits (id, node_id, display_name, region, publication_intent, publication_observed)
		VALUES ($1, $2, $3, 'tw-tpe', $4, 'active')`, exitID, nodeID, "exit-"+nodeID.String()[:8], string(intent)); err != nil {
		t.Fatalf("insert exit: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO exit_groups (exit_id, group_id) VALUES ($1, $2)`, exitID, groupID); err != nil {
		t.Fatalf("insert exit_group: %v", err)
	}
	return nodeID, incID
}

func insertFairUsePlan(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	revID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO plans (id, code) VALUES ($1, $2)`, planID, "plan-"+planID.String()[:8]); err != nil {
		t.Fatalf("insert plan: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO plan_revisions (id, plan_id, revision, device_limit, schema_version, payload)
		VALUES ($1, $2, 1, 3, 1, '{"rate_limit":{"down_bps":200000000,"up_bps":100000000}}')`, revID, planID); err != nil {
		t.Fatalf("insert plan revision: %v", err)
	}
	return revID
}

// testEnforceFairUsePayload is a TEST ONLY fair-use table. Thresholds and rates
// are fixtures, not a production policy.
func testEnforceFairUsePayload() api.FairUsePayload {
	en0 := "Standard"
	en1 := "Fair-Use Tier 1"
	en2 := "Fair-Use Tier 2"
	p := api.FairUsePayload{
		EffectiveAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectivePeriod: api.FairUsePayloadEffectivePeriodImmediate,
		MinimumRate:     api.RatePair{DownBps: 5_000_000, UpBps: 2_000_000},
		Mode:            api.FairUsePayloadModeEnforce,
		Overrides:       []api.FairUseOverride{},
		Tiers: []api.FairUseTier{
			{ThresholdBytes: 0, DownBps: 100_000_000, UpBps: 50_000_000, Priority: 1, DisplayName: api.LocalizedText{ZhTW: "標準速率", En: &en0}},
			{ThresholdBytes: 100_000_000_000, DownBps: 25_000_000, UpBps: 10_000_000, Priority: 3, DisplayName: api.LocalizedText{ZhTW: "公平使用第一階", En: &en1}},
			{ThresholdBytes: 9007199254740993, DownBps: 10_000_000, UpBps: 5_000_000, Priority: 5, DisplayName: api.LocalizedText{ZhTW: "公平使用第二階", En: &en2}},
		},
	}
	p.AccountingBasis.Measure = api.FairUsePayloadAccountingBasisMeasureTotalBytes
	p.AccountingBasis.MeteringVersion = api.MeteringVersionForwardAdmissionV1
	p.AccountingBasis.PeriodType = api.FairUsePayloadAccountingBasisPeriodTypeUtcCalendarMonth
	return p
}

// TEST ONLY - NOT A SECRET
func testFairUseDesiredStateSigner(t *testing.T) desiredstate.Signer {
	t.Helper()
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET - desired-state-key-1"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return desiredstate.NewKeySigner(desiredstate.SigningKey{
		KeyID:      "ds-key-test-1",
		Purpose:    envelope.PurposeDesiredState,
		PrivateKey: priv,
	})
}

func loadLatestOutbox(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventType string) db.ClaimedEvent {
	t.Helper()
	var ev db.ClaimedEvent
	err := pool.QueryRow(ctx, `
		SELECT id, aggregate_id, aggregate_type, aggregate_version::text, event_type, schema_version, payload
		FROM outbox_events
		WHERE event_type = $1
		ORDER BY available_at DESC, id DESC
		LIMIT 1`, eventType).Scan(
		&ev.ID, &ev.AggregateID, &ev.AggregateType, &ev.AggregateVersion, &ev.EventType, &ev.SchemaVersion, &ev.Payload,
	)
	if err != nil {
		t.Fatalf("load %s: %v", eventType, err)
	}
	return ev
}

func loadOutboxGeneration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, generation string) db.ClaimedEvent {
	t.Helper()
	var ev db.ClaimedEvent
	err := pool.QueryRow(ctx, `
		SELECT id, aggregate_id, aggregate_type, aggregate_version::text, event_type, schema_version, payload
		FROM outbox_events
		WHERE event_type = 'fup.member_generation_changed' AND payload->>'generation' = $1
		ORDER BY available_at DESC, id DESC
		LIMIT 1`, generation).Scan(
		&ev.ID, &ev.AggregateID, &ev.AggregateType, &ev.AggregateVersion, &ev.EventType, &ev.SchemaVersion, &ev.Payload,
	)
	if err != nil {
		t.Fatalf("load generation %s: %v", generation, err)
	}
	return ev
}

func assertDesiredGens(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want map[uuid.UUID]int64) {
	t.Helper()
	for id, gen := range want {
		var got int64
		if err := pool.QueryRow(ctx, `SELECT desired_config_generation::bigint FROM nodes WHERE id = $1`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != gen {
			t.Errorf("node %s desired_config_generation = %d, want %d", id, got, gen)
		}
	}
}

func assertMemberRate(t *testing.T, ctx context.Context, pool *pgxpool.Pool, nodeID, memberID uuid.UUID, generation uint64, tier int, down, up int64) {
	t.Helper()
	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT envelope FROM desired_state_envelopes WHERE node_id = $1`, nodeID).Scan(&raw); err != nil {
		t.Fatalf("envelope %s: %v", nodeID, err)
	}
	var env api.DesiredStateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	for _, pol := range env.Signed.MemberRatePolicies {
		if uuid.UUID(pol.MemberId) != memberID {
			continue
		}
		gotDown, err := pol.DownBps.Get()
		if err != nil {
			t.Fatal(err)
		}
		gotUp, err := pol.UpBps.Get()
		if err != nil {
			t.Fatal(err)
		}
		if uint64(pol.Generation) != generation || pol.TierIndex != tier || pol.Mode != api.MemberRatePolicyModeEnforce || gotDown != down || gotUp != up {
			t.Errorf("node %s member rate gen=%d tier=%d mode=%s down=%d up=%d, want gen=%d tier=%d enforce down=%d up=%d",
				nodeID, pol.Generation, pol.TierIndex, pol.Mode, gotDown, gotUp, generation, tier, down, up)
		}
		return
	}
	t.Errorf("node %s envelope has no rate policy for %s", nodeID, memberID)
}

func submitAppliedObservation(t *testing.T, ctx context.Context, svc *nodelib.Service, nodeID, incID uuid.UUID, applied uint64, observedAt time.Time) {
	t.Helper()
	rep := emptyFairUseObservation()
	rep.AgentVersion = "1.0.0"
	rep.BootId = uuid.New()
	rep.ReportId = uuid.New()
	rep.NodeId = nodeID
	rep.NodeIncarnationId = incID
	rep.ObservedAt = observedAt
	rep.AppliedConfigGeneration.Set(nodelib.ParseU64(applied))
	rep.ProvisioningState = api.ProvisioningStateReady
	rep.SafetyStatus = api.SafetyStatusPassed
	rep.DataHealth = api.DataHealthHealthy
	rep.MemberForwardingEnabled = false
	rep.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	rep.Checks = []api.NodeCheck{
		{Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusPassed, ObservedAt: observedAt, ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
		{Name: api.NodeCheckNameMeterPath, Status: api.HealthCheckStatusPassed, ObservedAt: observedAt, ErrorCode: presence.NewNullNullable[api.ErrorCode]()},
	}
	if _, err := svc.SubmitObservation(ctx, nodeID, incID, rep); err != nil {
		t.Fatalf("SubmitObservation %s: %v", nodeID, err)
	}
}

// emptyFairUseObservation supplies the explicit nulls the observation writer marshals.
// The stored report_payload is that marshaled report, not a hand-written document.
func emptyFairUseObservation() api.ObservationReport {
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
