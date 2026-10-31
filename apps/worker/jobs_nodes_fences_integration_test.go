//go:build integration

package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// TEST ONLY fault injection after the real device sync has committed its fence.
type fenceCompileFailure struct{}

func (fenceCompileFailure) RecompileBatch(context.Context, []db.ClaimedEvent) ([]desiredstate.CompiledDesiredState, error) {
	return nil, errors.New("TEST ONLY compiler unavailable")
}

func fenceState(t *testing.T, f *memberPolicyFixture, node uuid.UUID) api.DesiredState {
	t.Helper()
	var raw []byte
	var desired, applied decstr.U64
	if err := f.pool.QueryRow(t.Context(), `SELECT e.envelope,n.desired_config_generation::text,n.applied_config_generation::text FROM desired_state_envelopes e JOIN nodes n ON n.id=e.node_id WHERE e.node_id=$1`, node).Scan(&raw, &desired, &applied); err != nil {
		t.Fatal(err)
	}
	trust, err := f.provider.TrustBundle(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = envelope.Verify(raw, trust, envelope.PurposeDesiredState, time.Now()); err != nil {
		t.Fatal(err)
	}
	var env api.DesiredStateEnvelope
	if err = json.Unmarshal(raw, &env); err != nil || env.Signed.ConfigGeneration != desired || applied != 0 {
		t.Fatal("signed desired state mismatch or fabricated applied evidence", desired, applied, err)
	}
	return env.Signed
}

// D-077 / r16 H-1: both observation and sweep jobs remove an existing signed
// grant in the detecting dispatch, including exits no longer in current groups.
// A failed compilation keeps the original job retryable after the fence commits.
func TestD077_FenceRecompilesBoundExitsBeforeJobReceipt(t *testing.T) {
	for _, mode := range []string{"netbird.device_sync", "members.reconcile", "retry_after_commit", "mapping_lost_after_commit"} {
		t.Run(mode, func(t *testing.T) {
			f := newMemberPolicyFixture(t, true)
			app, err := Open(t.Context(), Dependencies{Config: f.config, Pool: f.pool}, f.memberDomain(), nodes())
			if err != nil {
				t.Fatal(err)
			}
			command := memberPolicyCommand("system", uuid.New())
			device, err := f.services.HTTP.Devices.AuthorizeDevice(t.Context(), command, f.peer.peer)
			if err != nil {
				t.Fatal(err)
			}
			f.assertCompleted(f.dispatch(app, "device.authorized"), 1, 1, 1)
			compiler := membersruntime.CompilerFromContext(app.context(t.Context()))
			seedExit := func(withGroup bool) uuid.UUID {
				t.Helper()
				node, incarnation, exit := uuid.New(), uuid.New(), uuid.New()
				fingerprint := sha256.Sum256(node[:])
				f.sql(`INSERT INTO nodes(id,lifecycle_state,provisioning_state,capacity) SELECT $1,lifecycle_state,provisioning_state,capacity FROM nodes WHERE id=$2`, node, f.node)
				f.sql(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,os_version,arch) VALUES($1,$2,$3,$4,$5,'ubuntu','24.04','amd64')`, incarnation, node, fingerprint[:], f.peer.peer.DeploymentID, "TEST ONLY "+node.String())
				f.sql(`UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, node, incarnation)
				f.sql(`INSERT INTO exits(id,node_id,display_name,region,publication_intent) VALUES($1,$2,'TEST ONLY exit','TEST ONLY region','published')`, exit, node)
				if withGroup {
					f.sql(`INSERT INTO exit_groups(exit_id,group_id) SELECT $1,eg.group_id FROM exit_groups eg JOIN exits e ON e.id=eg.exit_id WHERE e.node_id=$2`, exit, f.node)
				}
				if _, err := compiler.RecompileBatch(t.Context(), []db.ClaimedEvent{{OutboxEvent: db.OutboxEvent{EventType: desiredstate.EventNodeUpdated, AggregateID: node}}}); err != nil {
					t.Fatal(err)
				}
				return node
			}
			oldGroupNode, unrelated := seedExit(true), seedExit(false)
			if len(fenceState(t, f, oldGroupNode).DeviceBindings) != 1 || len(fenceState(t, f, unrelated).DeviceBindings) != 0 {
				t.Fatal("fixture did not establish bound and unrelated exits")
			}
			f.sql(`DELETE FROM exit_groups WHERE exit_id IN (SELECT id FROM exits WHERE node_id=$1)`, oldGroupNode)

			kind := "netbird.device_sync"
			if mode == "members.reconcile" {
				kind = mode
				if err = enqueueMemberWork(t.Context(), f.services, kind, scheduledPage{Run: uuid.New()}); err != nil {
					t.Fatal(err)
				}
			}
			// The first successful sync wrote observed_authorized=true. These two
			// independent withdrawn reads now cause SyncDevice to open the fence.
			f.peer.peer.Authorized, f.peer.peer.Groups = false, nil
			if mode == "retry_after_commit" {
				f.services.Compiler = fenceCompileFailure{}
			}
			if mode == "mapping_lost_after_commit" {
				f.peer.peer.IdentityUnmapped = true
			}
			job := f.dispatch(app, kind)
			assertJob := func(delivered bool) {
				t.Helper()
				var got, dead bool
				if err := f.pool.QueryRow(t.Context(), `SELECT delivered_at IS NOT NULL,dead_lettered_at IS NOT NULL FROM outbox_events WHERE id=$1`, job).Scan(&got, &dead); err != nil || got != delivered || dead {
					t.Fatal("job receipt mismatch", got, dead, err)
				}
			}
			var fenced bool
			var status string
			if err = f.pool.QueryRow(t.Context(), `SELECT status,EXISTS(SELECT 1 FROM device_authorization_fences WHERE device_id=$1) FROM member_devices WHERE id=$1`, device.DeviceID).Scan(&status, &fenced); err != nil || status != "authorized" || !fenced {
				t.Fatal("sync did not commit the authorization fence", status, fenced, err)
			}
			if mode == "retry_after_commit" {
				assertJob(false)
				if len(fenceState(t, f, f.node).DeviceBindings) != 1 {
					t.Fatal("fault injection did not leave a signed grant to retry")
				}
				f.services.Compiler = nil
				if retried := f.dispatch(app, kind); retried != job {
					t.Fatal("retry did not use the original durable event")
				}
			}
			assertJob(mode != "mapping_lost_after_commit")
			for _, node := range []uuid.UUID{f.node, oldGroupNode} {
				state := fenceState(t, f, node)
				if state.ConfigGeneration != 2 || len(state.DeviceBindings) != 0 {
					t.Fatal("fence left an active signed device grant", node, state)
				}
			}
			if fenceState(t, f, unrelated).ConfigGeneration != 1 {
				t.Fatal("fence recompiled an unrelated exit")
			}
			var groups []uuid.UUID
			if err = f.pool.QueryRow(t.Context(), `SELECT exit_group_ids FROM plan_revisions WHERE id=(SELECT plan_revision_id FROM entitlements WHERE member_id=$1)`, f.member).Scan(&groups); err != nil {
				t.Fatal(err)
			}
			assertLeased := func(want bool) {
				t.Helper()
				out, err := (leases.DBEntitlementReader{}).ListLeaseEntitlementsForGroups(t.Context(), f.pool, groups, time.Now())
				if err != nil || len(out) != 1 || slices.Contains(out[0].DeviceIDs, device.DeviceID) != want {
					t.Fatal("wrong lease device list", out, err)
				}
			}
			assertLeased(false)
			// Replaying observation work must not keep bumping generations once
			// every stored binding was removed, even while the fence remains.
			f.peer.peer.IdentityUnmapped = false
			var event db.ClaimedEvent
			event.EventType, event.SchemaVersion = "netbird.device_sync", 1
			if err = f.pool.QueryRow(t.Context(), `SELECT aggregate_id,aggregate_version::text,payload FROM outbox_events WHERE event_type='netbird.device_sync' AND aggregate_id=$1 LIMIT 1`, device.DeviceID).Scan(&event.AggregateID, &event.AggregateVersion, &event.Payload); err != nil {
				t.Fatal(err)
			}
			if _, err = memberJob(app.context(t.Context()), event); err != nil {
				t.Fatal(err)
			}
			if fenceState(t, f, f.node).ConfigGeneration != 2 || fenceState(t, f, oldGroupNode).ConfigGeneration != 2 {
				t.Fatal("repeat sync recompiled already withdrawn bindings")
			}
			// Original-key replay cannot clear the fence; a new explicit grant can.
			if _, err = f.services.HTTP.Devices.AuthorizeDevice(t.Context(), command, f.peer.peer); err != nil {
				t.Fatal(err)
			}
			assertLeased(false)
			if _, err = f.services.HTTP.Devices.AuthorizeDevice(t.Context(), memberPolicyCommand("admin", uuid.New()), f.peer.peer); err != nil {
				t.Fatal(err)
			}
			f.dispatch(app, "device.authorized")
			assertLeased(true)
			if len(fenceState(t, f, f.node).DeviceBindings) != 1 {
				t.Fatal("explicit re-authorization did not restore the signed binding")
			}
		})
	}
}

// TEST ONLY signer pauses after the compiler has read device eligibility.
type fencePausedSigner struct {
	provider signing.Provider
	entered  chan struct{}
	release  chan struct{}
}

func (s fencePausedSigner) Sign(ctx context.Context, purpose envelope.Purpose, signed any) (envelope.Envelope, error) {
	close(s.entered)
	select {
	case <-s.release:
		return s.provider.SignEnvelope(ctx, signing.Purpose(purpose), signed)
	case <-ctx.Done():
		return envelope.Envelope{}, ctx.Err()
	}
}

// D-077: a first compile must commit before the fence, or read the committed
// fence. Otherwise fanout can see no stored binding and acknowledge too early.
func TestD077_FenceSerializesWithFirstBindingCompile(t *testing.T) {
	f := newMemberPolicyFixture(t, true)
	app, err := Open(t.Context(), Dependencies{Config: f.config, Pool: f.pool}, f.memberDomain(), nodes())
	if err != nil {
		t.Fatal(err)
	}
	device, err := f.services.HTTP.Devices.AuthorizeDevice(t.Context(), memberPolicyCommand("system", uuid.New()), f.peer.peer)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.services.HTTP.Devices.SyncDevice(t.Context(), device.DeviceID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	pause := fencePausedSigner{provider: f.provider, entered: make(chan struct{}), release: make(chan struct{})}
	var release sync.Once
	unblock := func() { release.Do(func() { close(pause.release) }) }
	t.Cleanup(func() { unblock(); cancel() })
	compiler := desiredstate.NewCompiler(f.pool, pause)
	compiled := make(chan error, 1)
	go func() { _, err := compiler.Compile(ctx, f.node); compiled <- err }()
	select {
	case <-pause.entered:
	case err := <-compiled:
		t.Fatal("first compile failed before reaching signer", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	f.peer.peer.Authorized, f.peer.peer.Groups = false, nil
	var event db.ClaimedEvent
	event.EventType, event.SchemaVersion = "netbird.device_sync", 1
	if err = f.pool.QueryRow(ctx, `SELECT aggregate_id,aggregate_version::text,payload FROM outbox_events WHERE event_type='netbird.device_sync' AND aggregate_id=$1 LIMIT 1`, device.DeviceID).Scan(&event.AggregateID, &event.AggregateVersion, &event.Payload); err != nil {
		t.Fatal(err)
	}
	job := make(chan error, 1)
	go func() { _, err := memberJob(app.context(ctx), event); job <- err }()
	// Wait for either real PostgreSQL lock evidence or the unsafe early job
	// completion. No scheduling sleep is used to decide which ordering occurred.
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	finished := false
waitFence:
	for {
		select {
		case err = <-job:
			if err != nil {
				t.Fatal(err)
			}
			finished = true
			break waitFence
		case <-tick.C:
			var waiting bool
			if err = f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query='SELECT id FROM members WHERE id=$1 FOR UPDATE')`).Scan(&waiting); err != nil {
				t.Fatal(err)
			}
			if waiting {
				break waitFence
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	unblock()
	if err = <-compiled; err != nil {
		t.Fatal(err)
	}
	if !finished {
		if err = <-job; err != nil {
			t.Fatal(err)
		}
	}
	state := fenceState(t, f, f.node)
	if len(state.DeviceBindings) != 0 || state.ConfigGeneration != 2 {
		t.Fatalf("fence job acknowledged before concurrent first compile was invalidated: early=%v generation=%s bindings=%v", finished, state.ConfigGeneration, state.DeviceBindings)
	}
}
