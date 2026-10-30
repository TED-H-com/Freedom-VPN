//go:build integration

package httpwire_test

import (
	"crypto/x509"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func (f *fixture) sql(query string, args ...any) {
	f.t.Helper()
	if _, err := f.r.Pool.Exec(f.t.Context(), query, args...); err != nil {
		f.t.Fatal(err)
	}
}
func (f *fixture) activate(typ string, definition uuid.UUID, revision int, payload string) uuid.UUID {
	f.t.Helper()
	if definition == uuid.Nil {
		definition = uuid.New()
		f.sql(`INSERT INTO policy_definitions(id,type,code) VALUES($1,$2,$3)`, definition, typ, "TEST ONLY "+definition.String())
	}
	id := uuid.New()
	hash, err := canonjson.Hash([]byte(payload))
	if err != nil {
		f.t.Fatal(err)
	}
	f.sql(`INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,$3,$4,1,$5,$6,'TEST ONLY', $7)`, id, definition, typ, revision, payload, hash[:], f.p.ID.String())
	f.sql(`UPDATE policy_definitions SET active_revision_id=$2,generation=$3,resource_version=resource_version+1 WHERE id=$1`, definition, id, revision)
	return id
}

// TEST ONLY serving fixture uses the D-030 entitlement projection, with no assignments.
func servingFixture(t *testing.T) (*fixture, api.EnrollResponse, *x509.Certificate, uuid.UUID, uuid.UUID) {
	t.Helper()
	f, enrolled, cert, member, revision := publicationFixture(t)
	f.sql(`UPDATE exits SET publication_intent='published',resource_version=resource_version+1 WHERE node_id=$1`, enrolled.NodeId)
	return f, enrolled, cert, member, revision
}

func publicationFixture(t *testing.T) (*fixture, api.EnrollResponse, *x509.Certificate, uuid.UUID, uuid.UUID) {
	t.Helper()
	f := setup(t)
	vault := f.r.Provider.(*signing.FileProvider)
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease} {
		if _, err := vault.Rotate(t.Context(), purpose, 48*time.Hour, time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	var err error
	f.r.Fleet, err = fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	f.r.Config.DesiredStateSeconds = 600
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	_, enrolled, cert := f.enroll(f.create(nil))
	f.sql(`UPDATE nodes SET provisioning_state='ready',control_status='online',data_health='healthy',safety_status='passed',capacity='{"down_bps":1000000000,"up_bps":1000000000,"member_down_bps_cap":60000000,"member_up_bps_cap":30000000}',resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId)
	group, plan, planRev, member := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.sql(`INSERT INTO member_groups(id,code,name) VALUES($1,'TEST ONLY group','TEST ONLY group')`, group)
	f.sql(`INSERT INTO exit_groups(exit_id,group_id) SELECT id,$2 FROM exits WHERE node_id=$1`, enrolled.NodeId, group)
	f.sql(`INSERT INTO plans(id,code) VALUES($1,'TEST ONLY plan')`, plan)
	f.sql(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, planRev, plan, []uuid.UUID{group})
	f.sql(`INSERT INTO members(id,status) VALUES($1,'active')`, member)
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.sql(`INSERT INTO entitlements(member_id,revision,state,plan_revision_id,device_limit,valid_from,effective_service_end,projection) VALUES($1,1,'active',$2,3,$3,$4,jsonb_build_object('exit_group_ids',to_jsonb(ARRAY[$5::text])))`, member, planRev, now.Add(-time.Hour), now.Add(12*time.Hour), group.String())
	f.activate("release_support", uuid.Nil, 1, `{}`)
	revision := f.activate("fair_use", uuid.Nil, 6, `{"mode":"enforce"}`)
	period := desiredstate.DefaultAccountingPeriod(now)
	f.sql(`INSERT INTO fup_member_states(member_id,period_id,period_start,period_end,policy_revision_id,mode,would_apply_tier_index,would_apply_down_bps,would_apply_up_bps,would_apply_priority,would_apply_reason,enforced_state,enforced_tier_index,enforced_down_bps,enforced_up_bps,enforced_priority,enforced_reason,generation,aggregate_as_of) VALUES($1,$2,$3,$4,$5,'enforce',1,40000000,10000000,2,'fair_use_tier','adjusted',1,40000000,10000000,2,'fair_use_tier',7,$6)`, member, period.PeriodId, period.PeriodStart, period.PeriodEnd, revision, now)
	return f, enrolled, cert, member, revision
}

func TestAPI01_PublishedDesiredStateDBPolicyAndEligibilityCache(t *testing.T) {
	f, enrolled, cert, member, revision := servingFixture(t)
	trust, err := f.r.Provider.TrustBundle(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	read := func(tag string) (*httptest.ResponseRecorder, api.DesiredStateEnvelope) {
		t.Helper()
		w := f.call("GET", "/api/v1/agent/desired-state", nil, cert, map[string]string{"If-None-Match": tag})
		out := decode[api.DesiredStateEnvelope](t, w, 200)
		if _, _, err := envelope.Verify(w.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); err != nil {
			t.Fatal(err)
		}
		return w, out
	}
	first, state := read("")
	if len(state.Signed.MemberRatePolicies) != 1 {
		t.Fatalf("missing member policies: %+v", state.Signed)
	}
	rate := state.Signed.MemberRatePolicies[0]
	if rate.MemberId != member || mustValue(t, rate.DownBps) != 40000000 || mustValue(t, rate.UpBps) != 10000000 || rate.Generation != 7 || rate.TierIndex != 1 || rate.Priority != 2 || state.Signed.PolicyRefs.FupGeneration != 6 || state.Signed.PolicyRefs.DnsGeneration != 0 {
		t.Fatalf("wrong plan/cap/FUP policy: %+v", state.Signed)
	}
	cached := f.call("GET", "/api/v1/agent/desired-state", nil, cert, map[string]string{"If-None-Match": first.Header().Get("ETag")})
	if cached.Code != 304 {
		t.Fatalf("unchanged state not cached: %d %s", cached.Code, cached.Body.String())
	}
	var compiledRows int
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM desired_states WHERE node_id=$1`, enrolled.NodeId).Scan(&compiledRows); err != nil || compiledRows != 1 {
		t.Fatal("cache hit wrote another desired state", compiledRows, err)
	}
	var definition uuid.UUID
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT definition_id FROM policy_revisions WHERE id=$1`, revision).Scan(&definition); err != nil {
		t.Fatal(err)
	}
	f.activate("fair_use", definition, 8, `{"mode":"enforce"}`)
	second, state := read(first.Header().Get("ETag"))
	if state.Signed.PolicyRefs.FupGeneration != 8 || second.Header().Get("ETag") == first.Header().Get("ETag") {
		t.Fatal("active revision did not invalidate cache")
	}
	// A concrete policy revision must invalidate the cache even if FUP/DNS max
	// generations stay unchanged (for example, another plan revision).
	f.activate("plan", uuid.Nil, 1, `{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}`)
	third, _ := read(second.Header().Get("ETag"))
	if third.Header().Get("ETag") == second.Header().Get("ETag") {
		t.Fatal("policy refs omitted from cache inputs")
	}
	dnsDoc, _ := publishResolverProfile(t, f, nil, 60)
	fourth, state := read(third.Header().Get("ETag"))
	if state.Signed.PolicyRefs.DnsGeneration != dnsDoc.Reference.Generation {
		t.Fatal("DNS head generation missing")
	}
	// FUP changes need no active-policy revision change to invalidate an envelope.
	f.sql(`UPDATE fup_member_states SET mode='shadow',generation=generation+1 WHERE member_id=$1`, member)
	fourth, state = read(fourth.Header().Get("ETag"))
	rate = state.Signed.MemberRatePolicies[0]
	if rate.Mode != api.MemberRatePolicyModeShadow || mustValue(t, rate.DownBps) != 60000000 || mustValue(t, rate.UpBps) != 30000000 || rate.TierIndex != 0 {
		t.Fatal("shadow FUP/cap evidence did not invalidate cache", rate)
	}
	f.sql(`UPDATE members SET admin_suspended=true,resource_version=resource_version+1 WHERE id=$1`, member)
	fifth, state := read(fourth.Header().Get("ETag"))
	if len(state.Signed.MemberRatePolicies) != 0 || len(state.Signed.DeviceBindings) != 0 {
		t.Fatal("suspended member retained by cache")
	}
	f.sql(`UPDATE members SET admin_suspended=false,resource_version=resource_version+1 WHERE id=$1`, member)
	f.sql(`UPDATE exits SET publication_intent='unpublished',resource_version=resource_version+1 WHERE node_id=$1`, enrolled.NodeId)
	_, state = read(fifth.Header().Get("ETag"))
	if len(state.Signed.MemberRatePolicies) != 0 {
		t.Fatal("unpublished member grants")
	}
}

func TestAPI01_DesiredStateRequiresExplicitDNSAndReleaseSupport(t *testing.T) {
	f, _, cert, _, _ := servingFixture(t)
	f.r.Config.DNSResolverIPv4 = nil
	w := f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "Node DNS configuration required") {
		t.Fatalf("missing DNS accepted: %d %s", w.Code, w.Body.String())
	}
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	f.sql(`UPDATE policy_definitions SET active_revision_id=NULL,resource_version=resource_version+1 WHERE type='release_support'`)
	w = f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "Release support policy required") {
		t.Fatalf("missing release policy accepted: %d %s", w.Code, w.Body.String())
	}
}
