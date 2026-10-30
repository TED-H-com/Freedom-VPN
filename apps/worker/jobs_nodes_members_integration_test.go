//go:build integration

package worker

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TEST ONLY adapter provides independent read-back of the last authorization.
type memberPolicyPeer struct{ peer devices.Peer }

func (p *memberPolicyPeer) ObservePeer(context.Context, string, string) (devices.Peer, error) {
	return p.peer, nil
}
func (p *memberPolicyPeer) SetAuthorization(_ context.Context, _, _ string, allow bool, groups []uuid.UUID) error {
	p.peer.Authorized, p.peer.Groups = allow, slices.Clone(groups)
	return nil
}

type memberPolicyFixture struct {
	t        *testing.T
	pool     *pgxpool.Pool
	config   config.Config
	nodes    config.Nodes
	provider *signing.FileProvider
	services *membersruntime.Services
	node     uuid.UUID
	member   uuid.UUID
	peer     *memberPolicyPeer
}

func newMemberPolicyFixture(t *testing.T, withSigner bool) *memberPolicyFixture {
	t.Helper()
	f := &memberPolicyFixture{t: t, pool: dbtest.New(t), node: uuid.New(), member: uuid.New()}
	n := nodestest.New(t, nodestest.NetBird{})
	f.nodes, f.config = n.Nodes, config.Defaults()
	f.config.NodesConfigFile = n.Path
	f.config.WorkerAttempts = 2 // TEST ONLY; no production retry override.
	var err error
	f.provider, err = signing.OpenFileProvider(n.Nodes.ProviderDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if withSigner {
		if _, err = f.provider.Rotate(t.Context(), signing.DesiredState, 48*time.Hour, time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	write := func(name string, raw []byte) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		t.Fatal(err)
	}
	zero, live := 0, false
	// TEST ONLY explicit business settings and generated keys, no external calls.
	m := config.Members{
		Provider: "stripe", APIKeyFile: write("payment-key", []byte("TEST ONLY payment key")),
		WebhookSecretFiles: []string{write("webhook-key", []byte("TEST ONLY webhook key"))},
		VaultActiveKey:     "TEST ONLY", VaultKeyFiles: map[string]string{"TEST ONLY": write("vault-key", key)},
		LiveMode: &live, PriceIDs: []string{"TEST ONLY price"}, ReturnURLs: []string{"https://portal.example.invalid/billing/return"},
		FullRefundPolicy: billing.RevokeRefundedPeriod, GraceSeconds: &zero, TrialMaxSeconds: &zero,
		GrantMaxSeconds: 86400, DeviceLimit: 3, PortalTTLSeconds: 60, BillingLeaseSeconds: 60,
		MaxWebhookBytes: 1 << 20, ReconcileSeconds: 3600, DeviceSyncSeconds: 3600,
		NetBirdURL: f.nodes.NetBirdURL, NetBirdDeployment: f.nodes.NetBirdDeployment,
		NetBirdTokenFile: f.nodes.NetBirdTokenFile, OverlayScope: "TEST ONLY",
	}
	clear(key)
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	f.config.MembersConfigFile = write("members.json", raw)
	f.services, err = membersruntime.New(f.pool, m, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.peer = &memberPolicyPeer{peer: devices.Peer{DeploymentID: m.NetBirdDeployment, ID: "TEST ONLY peer", ExternalUserID: "TEST ONLY user", OverlayScope: m.OverlayScope, OverlayIP: netip.MustParseAddr("100.64.0.2"), Platform: "linux"}}
	f.services.HTTP.Devices.Adapter = f.peer

	group, plan, revision, incarnation, exit, identity := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.sql(`INSERT INTO members(id,status) VALUES($1,'active')`, f.member)
	f.sql(`INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,'https://id.example.invalid','portal','TEST ONLY subject','TEST ONLY claim','verified-link',$3)`, identity, f.member, now)
	f.sql(`INSERT INTO netbird_user_bindings(id,member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,$2,$3,$4,$5,'TEST ONLY proof',1,$6)`, uuid.New(), f.member, m.NetBirdDeployment, f.peer.peer.ExternalUserID, identity, now)
	f.sql(`INSERT INTO member_groups(id,code,name) VALUES($1,'TEST ONLY group','TEST ONLY group')`, group)
	f.sql(`INSERT INTO plans(id,code) VALUES($1,'TEST ONLY plan')`, plan)
	f.sql(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, revision, plan, []uuid.UUID{group})
	f.sql(`INSERT INTO entitlement_grants(id,member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,$2,'admin_grant',$3,$4,'TEST ONLY finite grant','TEST ONLY actor',$5)`, uuid.New(), f.member, now.Add(-time.Hour), now.Add(12*time.Hour), revision)
	f.sql(`INSERT INTO nodes(id,lifecycle_state,provisioning_state,capacity) VALUES($1,'registered','ready','{"down_bps":1000000000,"up_bps":1000000000,"member_down_bps_cap":60000000,"member_up_bps_cap":30000000}')`, f.node)
	f.sql(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,os_version,arch) VALUES($1,$2,$3,$4,'TEST ONLY node peer','ubuntu','24.04','amd64')`, incarnation, f.node, make([]byte, 32), m.NetBirdDeployment)
	f.sql(`UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, f.node, incarnation)
	f.sql(`INSERT INTO exits(id,node_id,display_name,region,publication_intent) VALUES($1,$2,'TEST ONLY exit','TEST ONLY region','published')`, exit, f.node)
	f.sql(`INSERT INTO exit_groups(exit_id,group_id) VALUES($1,$2)`, exit, group)
	if err = db.InTx(t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		_, err := f.services.HTTP.Members.Entitlements.Recompute(t.Context(), tx, f.member)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *memberPolicyFixture) sql(query string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.t.Context(), query, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *memberPolicyFixture) memberDomain() Domain {
	return Domain{Name: "members", Build: func(context.Context, Dependencies) (Built, error) {
		return Built{Jobs: MemberJobs(), Context: func(ctx context.Context) context.Context { return WithMembers(ctx, f.services) }}, nil
	}}
}

func (f *memberPolicyFixture) dispatch(app *Application, kind string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(f.t.Context(), `SELECT id FROM outbox_events WHERE event_type=$1 AND delivered_at IS NULL ORDER BY created_at,id LIMIT 1`, kind).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	// Defer unrelated events without consuming or changing their delivery state.
	f.sql(`UPDATE outbox_events SET available_at=CASE WHEN id=$1 THEN clock_timestamp() ELSE clock_timestamp()+interval '1 day' END WHERE delivered_at IS NULL AND dead_lettered_at IS NULL`, id)
	if worked, err := app.Once(f.t.Context()); err != nil || !worked {
		f.t.Fatal("member event not dispatched", worked, err)
	}
	return id
}

func (f *memberPolicyFixture) assertCompleted(id uuid.UUID, generation decstr.U64, policies, bindings int) {
	f.t.Helper()
	var acknowledged, audited bool
	err := f.pool.QueryRow(f.t.Context(), `SELECT attempt=1 AND delivered_at IS NOT NULL AND dead_lettered_at IS NULL AND last_error_code IS NULL AND delivery_receipt='audit_event:'||id::text,
		EXISTS(SELECT 1 FROM audit_events a WHERE a.id=o.id AND a.action='member.policy_compiled' AND (a.after_diff->>'compiled_nodes')::int=1)
		FROM outbox_events o WHERE id=$1`, id).Scan(&acknowledged, &audited)
	if err != nil || !acknowledged || !audited {
		f.t.Fatal("event was retried, dead-lettered, or lacks compilation evidence", acknowledged, audited, err)
	}
	var raw []byte
	var desired, applied decstr.U64
	err = f.pool.QueryRow(f.t.Context(), `SELECT n.desired_config_generation::text,n.applied_config_generation::text,e.envelope FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id WHERE n.id=$1`, f.node).Scan(&desired, &applied, &raw)
	if err != nil || desired != generation || applied != 0 {
		f.t.Fatal("desired generation missing or applied evidence fabricated", desired, applied, err)
	}
	trust, err := f.provider.TrustBundle(f.t.Context())
	if err != nil {
		f.t.Fatal(err)
	}
	if _, _, err = envelope.Verify(raw, trust, envelope.PurposeDesiredState, time.Now()); err != nil {
		f.t.Fatal("configured signing key not used", err)
	}
	var doc api.DesiredStateEnvelope
	if err = json.Unmarshal(raw, &doc); err != nil {
		f.t.Fatal(err)
	}
	state := doc.Signed
	if state.ConfigGeneration != generation || len(state.MemberRatePolicies) != policies || len(state.DeviceBindings) != bindings {
		f.t.Fatal("wrong compiled policy", state)
	}
	if state.ExpiresAt.Sub(state.IssuedAt) != time.Duration(f.nodes.DesiredStateSeconds)*time.Second || !slices.Equal(state.Dns.ResolverIpv4, f.nodes.DNSResolverIPv4) || !slices.Equal(state.Dns.ResolverIpv6, f.nodes.DNSResolverIPv6) {
		f.t.Fatal("compiler did not use nodes TTL/DNS", state)
	}
	if policies > 0 {
		p := state.MemberRatePolicies[0]
		down, de := p.DownBps.Get()
		up, ue := p.UpBps.Get()
		if p.MemberId != f.member || de != nil || ue != nil || down != 60000000 || up != 30000000 {
			f.t.Fatal("DB plan and node capacity not applied", p)
		}
	}
}

func memberPolicyCommand(role string, id uuid.UUID) entitlement.Command {
	return entitlement.Command{Actor: entitlement.Actor{Role: role, ID: id, RequestID: uuid.New()}, Key: uuid.New()}
}

// §7.3/§14.5: the real nodes domain supplies the compiler to member jobs in
// either composition order. Only external peer read-back is TEST ONLY.
func TestAT02_AT04_MemberEventsCompileNodePolicies(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "members_then_nodes"
		if reverse {
			name = "nodes_then_members"
		}
		t.Run(name, func(t *testing.T) {
			f := newMemberPolicyFixture(t, true)
			domains := []Domain{f.memberDomain(), nodes()}
			if reverse {
				slices.Reverse(domains)
			}
			app, err := Open(t.Context(), Dependencies{Config: f.config, Pool: f.pool}, domains...)
			if err != nil {
				t.Fatal(err)
			}
			device, err := f.services.HTTP.Devices.AuthorizeDevice(t.Context(), memberPolicyCommand("system", uuid.New()), f.peer.peer)
			if err != nil {
				t.Fatal(err)
			}
			f.assertCompleted(f.dispatch(app, "device.authorized"), 1, 1, 1)
			f.assertCompleted(f.dispatch(app, "member.entitlement_changed"), 2, 1, 1)
			if _, err = f.services.HTTP.Devices.RevokeDevice(t.Context(), memberPolicyCommand("member", f.member), device.DeviceID); err != nil {
				t.Fatal(err)
			}
			f.assertCompleted(f.dispatch(app, "device.revoked"), 3, 1, 0)
			suspended, reason := true, "TEST ONLY suspension"
			var version decstr.U64
			if err = f.pool.QueryRow(t.Context(), `SELECT resource_version::text FROM members WHERE id=$1`, f.member).Scan(&version); err != nil {
				t.Fatal(err)
			}
			if _, err = f.services.HTTP.Members.Patch(t.Context(), memberPolicyCommand("admin", uuid.New()), f.member, &version, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason}); err != nil {
				t.Fatal(err)
			}
			f.assertCompleted(f.dispatch(app, "member.suspended"), 4, 0, 0)
		})
	}
}

func TestD026_D045_ProductionWorkerSuppliesMemberCompiler(t *testing.T) {
	f := newMemberPolicyFixture(t, true)
	path := filepath.Join(t.TempDir(), "dns.json")
	if err := os.WriteFile(path, []byte(`{"upstreams":["192.0.2.53:53"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	f.config.DNSConfigFile = path
	f.config.PoliciesConfigFile = policiestest.New(t).Path
	f.config.UsageConfigFile = usagetest.New(t).Path
	// Production list and builds, with all D-045 required settings. No device
	// exists yet, so SyncMember needs no external effects in this composition test.
	app, err := Open(t.Context(), Dependencies{Config: f.config, Pool: f.pool})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := membersruntime.CompilerFromContext(app.context(t.Context())).(*desiredstate.Compiler); !ok {
		t.Fatal("production worker has no DB-backed member compiler")
	}
	f.assertCompleted(f.dispatch(app, "member.entitlement_changed"), 1, 1, 0)
}

func TestAT02_MemberCompilationFailureDoesNotAcknowledge(t *testing.T) {
	f := newMemberPolicyFixture(t, false) // TEST ONLY: no desired-state signing key.
	app, err := Open(t.Context(), Dependencies{Config: f.config, Pool: f.pool}, f.memberDomain(), nodes())
	if err != nil {
		t.Fatal(err)
	}
	id := f.dispatch(app, "member.entitlement_changed")
	var retried bool
	if err = f.pool.QueryRow(t.Context(), `SELECT attempt=1 AND delivered_at IS NULL AND dead_lettered_at IS NULL AND delivery_receipt IS NULL AND lease_owner IS NULL AND available_at>created_at FROM outbox_events WHERE id=$1`, id).Scan(&retried); err != nil || !retried {
		t.Fatal("unsigned policy acknowledged", retried, err)
	}
	var clean bool
	if err = f.pool.QueryRow(t.Context(), `SELECT desired_config_generation=0 AND NOT EXISTS(SELECT 1 FROM desired_states WHERE node_id=$1) AND NOT EXISTS(SELECT 1 FROM audit_events WHERE id=$2) FROM nodes WHERE id=$1`, f.node, id).Scan(&clean); err != nil || !clean {
		t.Fatal("failed compilation leaked desired state or success evidence", clean, err)
	}
}
