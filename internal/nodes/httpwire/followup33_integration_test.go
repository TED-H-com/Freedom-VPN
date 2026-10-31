//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TEST ONLY database seed. Both HTTP runtimes subsequently share the same real
// enrollment, authorities and DB inputs; toggling grants never edits this seed.
func grantEligibilityFixture(t *testing.T) (*fixture, config.Nodes, api.EnrollResponse, *x509.Certificate, uuid.UUID) {
	t.Helper()
	pool := dbtest.New(t)
	cfg := nodestest.New(t, nodestest.NetBird{})
	c := config.Defaults()
	c.NodesConfigFile = cfg.Path
	domain, _ := nodeDomains(t)
	built, err := domain.Build(t.Context(), control.Dependencies{Config: c, Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	var runtime *httpwire.Runtime
	for _, handler := range built.Handlers {
		if h, ok := handler.(*httpwire.Handlers); ok {
			runtime = h.Runtime
		}
	}
	if runtime == nil {
		t.Fatal("production nodes constructor did not build the HTTP runtime")
	}
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease} {
		if _, err := runtime.Provider.(*signing.FileProvider).Rotate(t.Context(), purpose, 48*time.Hour, time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	f := setupRuntime(t, runtime)
	release := releasetest.New(t)
	runtime.Releases.Validator = release.Validator
	if _, err := runtime.Releases.Register(t.Context(), releasetest.Command(), release.Request(t, release.Manifest)); err != nil {
		t.Fatal(err)
	}
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{release.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	if err := db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return runtime.Releases.ApplySupportTx(t.Context(), tx, f.p.ID, support, "TEST ONLY grant eligibility")
	}); err != nil {
		t.Fatal(err)
	}
	_, enrolled, cert := f.enroll(f.create(nil))
	f.sql(`UPDATE nodes SET provisioning_state='ready',control_status='online',data_health='healthy',safety_status='passed',capacity='{"down_bps":1000000000,"up_bps":1000000000,"member_down_bps_cap":60000000,"member_up_bps_cap":30000000}',resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId)
	f.sql(`UPDATE exits SET publication_intent='published',resource_version=resource_version+1 WHERE node_id=$1`, enrolled.NodeId)
	group, plan, planRev, member, device := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.sql(`INSERT INTO member_groups(id,code,name) VALUES($1,'TEST ONLY group','TEST ONLY group')`, group)
	f.sql(`INSERT INTO exit_groups(exit_id,group_id) SELECT id,$2 FROM exits WHERE node_id=$1`, enrolled.NodeId, group)
	f.sql(`INSERT INTO plans(id,code) VALUES($1,'TEST ONLY plan')`, plan)
	f.sql(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRev, plan, []uuid.UUID{group})
	f.sql(`INSERT INTO members(id,status) VALUES($1,'active')`, member)
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.sql(`INSERT INTO entitlements(member_id,revision,state,plan_revision_id,device_limit,valid_from,effective_service_end,projection) VALUES($1,1,'active',$2,3,$3,$4,jsonb_build_object('exit_group_ids',to_jsonb(ARRAY[$5::text])))`, member, planRev, now.Add(-time.Hour), now.Add(12*time.Hour), group.String())
	f.sql(`INSERT INTO member_devices(id,member_id,deployment_id,peer_id,status,authorized_at) VALUES($1,$2,'TEST ONLY','TEST ONLY peer','authorized',$3)`, device, member, now.Add(-time.Hour))
	f.sql(`INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,'TEST ONLY','100.64.0.10',1,$3,$4,'TEST ONLY peer')`, device, member, now.Add(-time.Hour), now.Add(12*time.Hour))
	f.activate("release_support", uuid.Nil, 1, `{}`)
	validateFollowup28Responses(t, f)
	return f, cfg.Nodes, enrolled, cert, member
}

func TestD087_AT10_ProductionConstructorAndHTTPFixedGenerationGrants(t *testing.T) {
	production, cfg, enrolled, cert, member := grantEligibilityFixture(t)
	gate := nodestest.NewMemberGrantEligibility()
	calls := 0
	runtime, err := httpwire.Open(t.Context(), production.r.Pool, cfg, nil,
		httpwire.WithTestOnlyGrantEligibilityReaderFactory(func(tx pgx.Tx) leases.EntitlementReader {
			calls++
			if tx == nil {
				t.Fatal("factory needs the renewal transaction")
			}
			return gate.ReaderFactory(tx)
		}))
	if err != nil {
		t.Fatal(err)
	}
	injected := setupRuntime(t, runtime)
	validateFollowup28Responses(t, injected)
	trust, err := runtime.Provider.TrustBundle(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	const desiredPath = "/api/v1/agent/desired-state"
	const leasePath = "/api/v1/agent/lease-renewals"
	baseline := production.call("GET", desiredPath, nil, cert, nil)
	desired := decode[api.DesiredStateEnvelope](t, baseline, http.StatusOK)
	if len(desired.Signed.DeviceBindings) != 1 || len(desired.Signed.MemberRatePolicies) != 1 || desired.Signed.ConfigGeneration == 0 {
		t.Fatal("fixture must start with a real compiled generation and member bindings")
	}
	if _, _, err := envelope.Verify(baseline.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); err != nil {
		t.Fatal(err)
	}
	request := func() api.LeaseRenewalRequest {
		return api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
			CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullableWithValue(desired.Signed.ConfigGeneration)}
	}
	renew := func(f *fixture, req api.LeaseRenewalRequest, key string, count int) *httptest.ResponseRecorder {
		t.Helper()
		w := f.call("POST", leasePath, req, cert, map[string]string{"Idempotency-Key": key})
		got := decode[api.ServingLeaseEnvelope](t, w, http.StatusOK)
		if _, _, err := envelope.Verify(w.Body.Bytes(), trust, envelope.PurposeLease, time.Now()); err != nil {
			t.Fatal(err)
		}
		if len(got.Signed.Members) != count || got.Signed.ConfigGeneration != desired.Signed.ConfigGeneration {
			t.Fatalf("grants/generation changed unexpectedly: %+v", got.Signed)
		}
		if count > 0 && (got.Signed.Members[0].MemberId != member) {
			t.Fatal("production member/device eligibility was lost")
		}
		if got.Signed.ExpiresAt.Sub(got.Signed.IssuedAt) != time.Duration(got.Signed.MaxOfflineSeconds)*time.Second {
			t.Fatal("ready lease window was shortened by grant suppression")
		}
		return w
	}
	initialRequest, initialKey := request(), uuid.NewString()
	initial := renew(production, initialRequest, initialKey, 1)
	// Endpoint observation metadata is now initialized. Snapshot only the
	// invariants, not the normal last-seen timestamps or new lease/audit rows.
	snapshot := func() []byte {
		t.Helper()
		var raw []byte
		if err := runtime.Pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
			'node',jsonb_build_array(n.lifecycle_state,n.provisioning_state,n.safety_status,n.control_status,n.data_health,n.member_forwarding_observed,n.desired_config_generation::text,n.applied_config_generation::text,n.resource_version::text),
			'publication',jsonb_build_array(e.publication_intent,e.publication_observed,e.resource_version::text),
			'frozen',EXISTS(SELECT 1 FROM node_identity_endpoints WHERE incarnation_id=n.current_incarnation_id AND mapping_frozen),
			'compiled',(SELECT jsonb_agg(to_jsonb(d) ORDER BY d.generation) FROM desired_states d WHERE d.node_id=n.id),
			'envelope',(SELECT to_jsonb(s) FROM desired_state_envelopes s WHERE s.node_id=n.id),
			'bindings',(SELECT jsonb_agg(to_jsonb(b) ORDER BY b.device_id) FROM device_bindings b WHERE b.member_id=$2),
			'entitlement',(SELECT to_jsonb(ent) FROM entitlements ent WHERE ent.member_id=$2))
			FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1`, enrolled.NodeId, member).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := snapshot()
	unchanged := func() {
		t.Helper()
		previousCalls := calls
		for _, f := range []*fixture{production, injected} {
			got := f.call("GET", desiredPath, nil, cert, nil)
			decode[api.DesiredStateEnvelope](t, got, http.StatusOK)
			if !bytes.Equal(got.Body.Bytes(), baseline.Body.Bytes()) || got.Header().Get("ETag") != baseline.Header().Get("ETag") {
				t.Fatal("grant switch changed desired-state bytes or ETag")
			}
		}
		if calls != previousCalls {
			t.Fatal("grant factory leaked into desired-state compilation")
		}
		if !bytes.Equal(before, snapshot()) {
			t.Fatal("grant switch changed bindings, generation, readiness, publication, freeze or entitlement rows")
		}
	}

	gate.SetMemberGrantsReady(false)
	unchanged()
	emptyRequest, emptyKey := request(), uuid.NewString()
	empty := renew(injected, emptyRequest, emptyKey, 0)
	if calls != 1 {
		t.Fatalf("factory called %d times for one renewal", calls)
	}
	unchanged()
	// Production constructor has no override even while this fixture withholds.
	renew(production, request(), uuid.NewString(), 1)
	for _, key := range []string{initialKey, uuid.NewString(), ""} {
		if got := renew(injected, initialRequest, key, 1); !bytes.Equal(initial.Body.Bytes(), got.Body.Bytes()) {
			t.Fatal("withholding rewrote a previously issued populated lease")
		}
	}
	gate.SetMemberGrantsReady(true)
	unchanged()
	for _, key := range []string{emptyKey, uuid.NewString(), ""} {
		if got := renew(injected, emptyRequest, key, 0); !bytes.Equal(empty.Body.Bytes(), got.Body.Bytes()) {
			t.Fatal("readiness rewrote a previously issued empty lease")
		}
	}
	renew(injected, request(), uuid.NewString(), 1)
	unchanged()
	// A restored default reader continues to enforce real production eligibility.
	injected.sql(`UPDATE members SET admin_suspended=true,resource_version=resource_version+1 WHERE id=$1`, member)
	renew(injected, request(), uuid.NewString(), 0)
}

func TestD087_AT29_HTTPNilGrantReaderFailsClosed(t *testing.T) {
	f, cfg, enrolled, cert, _ := grantEligibilityFixture(t)
	runtime, err := httpwire.Open(t.Context(), f.r.Pool, cfg, nil,
		httpwire.WithTestOnlyGrantEligibilityReaderFactory(func(pgx.Tx) leases.EntitlementReader { return nil }))
	if err != nil {
		t.Fatal(err)
	}
	broken := setupRuntime(t, runtime)
	validateFollowup28Responses(t, broken)
	req := api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
		CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()}
	key := map[string]string{"Idempotency-Key": uuid.NewString()}
	w := broken.call("POST", "/api/v1/agent/lease-renewals", req, cert, key)
	errBody := decode[api.ErrorEnvelope](t, w, http.StatusServiceUnavailable)
	if errBody.Error.Code != api.ErrorCodeDEPENDENCYUNAVAILABLE {
		t.Fatal("nil reader did not fail closed")
	}
	for _, table := range []string{"serving_leases", "serving_lease_members"} {
		var count int
		if err := f.r.Pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("failed renewal persisted %s: %d %v", table, count, err)
		}
	}
	// No completed command receipt survives: the production constructor can
	// successfully retry the exact same request and key after fixture failure.
	got := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", req, cert, key), http.StatusOK)
	if len(got.Signed.Members) != 1 {
		t.Fatal("production retry did not use its default reader")
	}
}
