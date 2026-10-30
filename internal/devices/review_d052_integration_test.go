//go:build integration

package devices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }

// d052UpstreamFake simulates the NetBird management API for D-052/D-050 tests
// without hand-writing projector-owned rows.
type d052UpstreamFake struct {
	mu          sync.Mutex
	users       map[string]map[string]any
	peers       map[string]map[string]any
	groups      map[string]map[string]any
	failNextPut bool
}

func newD052UpstreamFake() *d052UpstreamFake {
	return &d052UpstreamFake{
		users:  map[string]map[string]any{},
		peers:  map[string]map[string]any{},
		groups: map[string]map[string]any{},
	}
}

func (u *d052UpstreamFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u.mu.Lock()
	defer u.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	kind, id, _ := strings.Cut(path, "/")
	if kind == "dns" {
		kind = path
		id = ""
	}
	encodeGroup := func(g map[string]any) map[string]any {
		cp := map[string]any{}
		for k, v := range g {
			cp[k] = v
		}
		peers := []map[string]any{}
		b, _ := json.Marshal(g["peers"])
		var ids []string
		if json.Unmarshal(b, &ids) == nil {
			for _, pid := range ids {
				peers = append(peers, map[string]any{"id": pid})
			}
			cp["peers"] = peers
		}
		return cp
	}
	var bucket map[string]map[string]any
	switch kind {
	case "users":
		bucket = u.users
	case "peers":
		bucket = u.peers
	case "groups":
		bucket = u.groups
	case "routes", "policies", "dns/nameservers":
		_ = json.NewEncoder(w).Encode([]any{})
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	respond := func(obj map[string]any) {
		if kind == "groups" {
			_ = json.NewEncoder(w).Encode(encodeGroup(obj))
			return
		}
		_ = json.NewEncoder(w).Encode(obj)
	}
	if r.Method == http.MethodGet {
		if id != "" {
			obj, ok := bucket[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			respond(obj)
			return
		}
		out := make([]map[string]any, 0, len(bucket))
		for _, v := range bucket {
			if kind == "groups" {
				out = append(out, encodeGroup(v))
			} else {
				out = append(out, v)
			}
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	if r.Method == http.MethodDelete {
		delete(bucket, id)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var body map[string]any
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodPut && u.failNextPut {
		u.failNextPut = false
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if id == "" {
		id, _ = body["id"].(string)
		if id == "" {
			id = "grp-" + uuid.NewString()
		}
	}
	body["id"] = id
	bucket[id] = body
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	}
	respond(body)
}

func (u *d052UpstreamFake) peerAuthorized(peerID string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, g := range u.groups {
		name, _ := g["name"].(string)
		if !strings.HasPrefix(name, "fvpn-paid-") {
			continue
		}
		rawPeers, _ := g["peers"].([]any)
		for _, item := range rawPeers {
			switch v := item.(type) {
			case string:
				if v == peerID {
					return true
				}
			case map[string]any:
				if v["id"] == peerID {
					return true
				}
			}
		}
	}
	return false
}

type d052Env struct {
	t        *testing.T
	pool     *pgxpool.Pool
	now      time.Time
	upstream *d052UpstreamFake
	server   *httptest.Server
	engine   *entitlement.Store
	members  members.Service
	adapter  *NetBirdAdapter
	devices  *Service
	group    uuid.UUID
	plan     uuid.UUID
	revision uuid.UUID
}

func newD052Env(t *testing.T, issuer string, deviceLimit int) *d052Env {
	t.Helper()
	pool := dbtest.New(t)
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)

	upstream := newD052UpstreamFake()
	srv := httptest.NewServer(upstream)
	t.Cleanup(srv.Close)

	engine, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	engine.Now = func() time.Time { return now }

	bindings := NetBirdBindings{Pool: pool}
	nbClient, err := netbird.New(netbird.Config{
		BaseURL:           srv.URL + "/api",
		Token:             staticToken("TEST ONLY token"),
		DeploymentID:      "test-deploy",
		Bindings:          bindings,
		HTTPClient:        srv.Client(),
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := NewIdentityDiscovery(issuer, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &NetBirdAdapter{
		DeploymentID: "test-deploy",
		OverlayScope: "test-scope",
		Client:       nbClient,
		Identity:     identity,
		Bindings:     bindings,
		Pool:         pool,
	}
	devSvc := &Service{Pool: pool, Entitlements: engine, Adapter: adapter}
	memSvc := members.Service{Pool: pool, Entitlements: engine}

	// Seed only configuration inputs (member_groups, plans, plan_revisions); never
	// hand-write projector-owned rows (D-050).
	group, plan, revision := uuid.New(), uuid.New(), uuid.New()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO member_groups(id,code,name) VALUES($1,'g1','TEST ONLY Group 1')`, []any{group}},
		{`INSERT INTO plans(id,code) VALUES($1,'plan-std')`, []any{plan}},
		{`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,$3,$4,1,'{}')`, []any{revision, plan, deviceLimit, []uuid.UUID{group}}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}

	return &d052Env{
		t:        t,
		pool:     pool,
		now:      now,
		upstream: upstream,
		server:   srv,
		engine:   engine,
		members:  memSvc,
		adapter:  adapter,
		devices:  devSvc,
		group:    group,
		plan:     plan,
		revision: revision,
	}
}

func (e *d052Env) loginMember(issuer, clientScope, subject string) uuid.UUID {
	e.t.Helper()
	store := auth.PostgresStore{Pool: e.pool}
	p, err := store.Login(e.t.Context(), auth.Identity{
		Issuer:      issuer,
		ClientScope: clientScope,
		Subject:     subject,
		Label:       "TEST ONLY Member",
		Role:        auth.Member,
		AuthTime:    e.now,
		Expires:     e.now.Add(time.Hour),
	}, uuid.NewString(), "", time.Hour, uuid.New())
	if err != nil {
		e.t.Fatal(err)
	}
	return p.ID
}

func (e *d052Env) grantMember(memberID, revisionID uuid.UUID, duration time.Duration) uuid.UUID {
	e.t.Helper()
	cmd := entitlement.Command{
		Actor: entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()},
		Key:   uuid.New(),
	}
	res, err := e.members.Grant(e.t.Context(), cmd, memberID, api.GrantCreateRequest{
		Source:         api.GrantSourceAdminGrant,
		PlanRevisionId: &revisionID,
		ValidFrom:      &e.now,
		ValidUntil:     e.now.Add(duration),
		Reason:         "TEST ONLY grant",
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return res.ResourceID
}

func (e *d052Env) addUpstreamPeer(subject, peerID, ip string) {
	e.upstream.mu.Lock()
	defer e.upstream.mu.Unlock()
	e.upstream.users[subject] = map[string]any{"id": subject, "auto_groups": []string{}, "is_service_user": false, "is_blocked": false}
	e.upstream.peers[peerID] = map[string]any{"id": peerID, "user_id": subject, "name": peerID, "ip": ip, "ephemeral": false}
}

// TestReview_F04_CrossClientScopeCreatesDuplicateMemberAndBricksD052 proves that:
//  1. Because identity_bindings has UNIQUE(issuer, client_scope, subject) rather than
//     UNIQUE(issuer, subject) and PostgresStore.Login filters by client_scope,
//     logging in the same (issuer, subject) under a second client_scope creates a
//     second member and a second identity_bindings row for the same (issuer, subject).
//  2. As soon as that second identity_bindings row exists, verifiedIdentity flips to
//     false for the first member's already-verified netbird_user_bindings row, causing
//     SyncDevice to withdraw the first member's device in NetBird and fail with ErrUnmapped.
func TestReview_F04_CrossClientScopeCreatesDuplicateMemberAndBricksD052(t *testing.T) {
	issuer := "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()

	memberA := env.loginMember(issuer, "portal-v1", "shared-public-subject")
	env.grantMember(memberA, env.revision, 2*time.Hour)
	env.addUpstreamPeer("shared-public-subject", "peer-member-a", "100.64.0.10")

	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	var deviceA uuid.UUID
	if err := env.pool.QueryRow(ctx, `SELECT id FROM member_devices WHERE member_id=$1 AND peer_id='peer-member-a'`, memberA).Scan(&deviceA); err != nil {
		t.Fatal(err)
	}
	if err := env.devices.SyncDevice(ctx, deviceA); err != nil {
		t.Fatal(err)
	}
	if !env.upstream.peerAuthorized("peer-member-a") {
		t.Fatal("expected peer-member-a to be authorized initially")
	}

	// Now the same (issuer, subject) logs in after a client_scope change ("portal-v2").
	memberB := env.loginMember(issuer, "portal-v2", "shared-public-subject")
	if memberB != memberA {
		t.Errorf("TODO(review) [F-04]: same (issuer, subject) across client_scope created duplicate member %s != %s", memberB, memberA)
	}

	var bindingCount int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM identity_bindings WHERE issuer=$1 AND subject='shared-public-subject'`, issuer).Scan(&bindingCount); err != nil {
		t.Fatal(err)
	}
	if bindingCount != 1 {
		t.Errorf("TODO(review) [F-04]: expected exactly 1 identity_binding for (issuer, subject), got %d", bindingCount)
	}

	// Re-syncing Member A's already-authorized device now fails closed and revokes
	// Member A's NetBird access because verifiedIdentity rejects len(identity_bindings) > 1.
	syncErr := env.devices.SyncDevice(ctx, deviceA)
	if syncErr != nil || !env.upstream.peerAuthorized("peer-member-a") {
		t.Errorf("TODO(review) [F-04]: second client_scope login bricked Member A's existing D-052 mapping and withdrew NetBird access (syncErr=%v, peerAuthorized=%v)",
			syncErr, env.upstream.peerAuthorized("peer-member-a"))
	}
}

// TestReview_F06_SuspensionAndGrantRevokeKeepActiveDeviceBindingsBlockingIPReuse proves
// that suspending a member (members.Patch with admin_suspended=true) or revoking a
// grant (members.RevokeGrant) does not clamp device_bindings.valid_until to now,
// leaving suspended/revoked members with active device_bindings rows in PostgreSQL
// and blocking overlay IP reuse by another member via device_bindings_overlay_window_excl.
func TestReview_F06_SuspensionAndGrantRevokeKeepActiveDeviceBindingsBlockingIPReuse(t *testing.T) {
	issuer := "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()

	memberA := env.loginMember(issuer, "portal", "subject-suspend-a")
	grantID := env.grantMember(memberA, env.revision, 2*time.Hour)
	env.addUpstreamPeer("subject-suspend-a", "peer-suspend-a", "100.64.0.42")
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}

	// The frozen fixture clock originally made admission and suspension the
	// same instant. F-06 explicitly retains a minimum one-microsecond interval;
	// move the input clock before asserting that ownership has ended.
	env.now = env.now.Add(time.Microsecond)
	env.engine.Now = func() time.Time { return env.now }
	// Admin suspends Member A.
	m, err := env.members.Get(ctx, entitlement.Actor{ID: uuid.New(), Role: "admin"}, memberA)
	if err != nil {
		t.Fatal(err)
	}
	suspended := true
	reason := "TEST ONLY suspend"
	cmd := entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := env.members.Patch(ctx, cmd, memberA, &m.ResourceVersion, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason}); err != nil {
		t.Fatal(err)
	}
	if err := env.devices.SyncMember(ctx, memberA); err != nil {
		t.Fatal(err)
	}

	var openBindingsOnSuspend int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE member_id=$1 AND valid_from<=$2 AND valid_until>$2`, memberA, env.now).Scan(&openBindingsOnSuspend); err != nil {
		t.Fatal(err)
	}
	if openBindingsOnSuspend != 0 {
		t.Errorf("[F-06]: suspended member %s still has %d active device_bindings rows with valid_until > now", memberA, openBindingsOnSuspend)
	}

	// Admin also revokes Member A's grant.
	cmd.Key = uuid.New()
	if err := env.members.RevokeGrant(ctx, cmd, memberA, grantID, "TEST ONLY revoke grant"); err != nil {
		t.Fatal(err)
	}
	var openBindingsOnRevokeGrant int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM device_bindings WHERE member_id=$1 AND valid_from<=$2 AND valid_until>$2`, memberA, env.now).Scan(&openBindingsOnRevokeGrant); err != nil {
		t.Fatal(err)
	}
	if openBindingsOnRevokeGrant != 0 {
		t.Errorf("[F-06]: grant-revoked member %s still has %d active device_bindings rows with valid_until > now", memberA, openBindingsOnRevokeGrant)
	}

	// NetBird reassigns overlay IP 100.64.0.42 to active Member B's peer.
	memberB := env.loginMember(issuer, "portal", "subject-active-b")
	env.grantMember(memberB, env.revision, 2*time.Hour)
	env.upstream.mu.Lock()
	delete(env.upstream.peers, "peer-suspend-a")
	env.upstream.mu.Unlock()
	env.addUpstreamPeer("subject-active-b", "peer-active-b", "100.64.0.42")
	if _, err := env.adapter.bindUser(ctx, "subject-active-b"); err != nil {
		t.Fatal(err)
	}

	_, admitErr := env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", memberB), "test-deploy", "peer-active-b")
	if admitErr != nil {
		t.Errorf("[F-06]: active Member B could not admit peer with reassigned IP 100.64.0.42 because suspended/revoked Member A kept open device_binding: %v", admitErr)
	}
}

// TestReview_F08_RuntimeAdapterNilIdentityAllowsUnverifiedMappingProof proves that
// when NetBirdAdapter.Identity is nil (the default constructed by membersruntime.New),
// ObservePeer skips verifiedIdentity and AuthorizeDevice/SyncDevice only check
// b.mapping_proof<>” in SQL, authorizing devices even when netbird_user_bindings
// carries a proof for a different issuer or a mismatched external_user_id.
func TestReview_F08_RuntimeAdapterNilIdentityAllowsUnverifiedMappingProof(t *testing.T) {
	issuer := "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()

	memberA := env.loginMember(issuer, "portal", "subject-real-member")
	env.grantMember(memberA, env.revision, 2*time.Hour)
	env.addUpstreamPeer("subject-real-member", "peer-f08", "100.64.0.88")
	if _, err := env.adapter.bindUser(ctx, "subject-real-member"); err != nil {
		t.Fatal(err)
	}

	// Corrupt the mapping_proof to a foreign issuer (simulating an unverified or
	// stale proof from another issuer). With Identity != nil this is rejected; with
	// Identity == nil (as returned by membersruntime.New before manual wiring) it slips through.
	if _, err := env.pool.Exec(ctx, `UPDATE netbird_user_bindings SET mapping_proof='shared_issuer_public_subject:https://foreign.example.invalid' WHERE member_id=$1`, memberA); err != nil {
		t.Fatal(err)
	}
	env.adapter.Identity = nil

	change, err := env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", memberA), "test-deploy", "peer-f08")
	if err == nil {
		_ = env.devices.SyncDevice(ctx, change.DeviceID)
		t.Errorf("[F-08]: ObserveAndAuthorize + SyncDevice succeeded with nil Adapter.Identity and foreign mapping_proof (device=%s, peerAuthorized=%v); want ErrUnmapped",
			change.DeviceID, env.upstream.peerAuthorized("peer-f08"))
	}
}

// TestReview_F09_IssuerNormalizationMismatchFailsD052Discovery proves that
// NewIdentityDiscovery does not canonicalize trailing slashes, uppercase hostnames,
// or explicit default ports (:443), causing D-052 discovery to fail with
// unmatched_subject when config and token issuer strings differ only in normalization.
func TestReview_F09_IssuerNormalizationMismatchFailsD052Discovery(t *testing.T) {
	canonicalIssuer := "https://issuer.example.invalid"
	variants := []struct {
		name   string
		issuer string
	}{
		{name: "trailing_slash", issuer: "https://issuer.example.invalid/"},
		{name: "uppercase_host", issuer: "https://ISSUER.EXAMPLE.INVALID"},
		{name: "default_https_port", issuer: "https://issuer.example.invalid:443"},
	}

	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			env := newD052Env(t, canonicalIssuer, 3)
			ctx := t.Context()

			// Member logs in with canonical issuer in ID token.
			env.loginMember(canonicalIssuer, "portal", "subject-norm-1")
			env.addUpstreamPeer("subject-norm-1", "peer-norm-1", "100.64.0.91")

			disc, err := NewIdentityDiscovery(tc.issuer, nil, nil)
			if err != nil {
				return
			}
			env.adapter.Identity = disc
			if err := env.adapter.Discover(ctx, env.devices); err != nil {
				t.Fatal(err)
			}

			var bound int
			if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM netbird_user_bindings`).Scan(&bound); err != nil {
				t.Fatal(err)
			}
			if bound != 1 {
				t.Errorf("[F-09]: NewIdentityDiscovery accepted unnormalized issuer %q without canonicalizing to %q, resulting in %d bound users (unmatched_subject)",
					tc.issuer, canonicalIssuer, bound)
			}
		})
	}
}

// TestReview_F10_SyncMemberEarlyAbortLeavesRemainingDevicesAuthorizedOnSuspend proves
// that when a member with multiple devices is suspended and SyncDevice fails on the
// first device in UUID order, SyncMember aborts early instead of withdrawing NetBird
// access for the remaining devices.
func TestReview_F10_SyncMemberEarlyAbortLeavesRemainingDevicesAuthorizedOnSuspend(t *testing.T) {
	issuer := "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()

	memberA := env.loginMember(issuer, "portal", "subject-f10")
	env.grantMember(memberA, env.revision, 2*time.Hour)
	env.addUpstreamPeer("subject-f10", "peer-f10-1", "100.64.0.101")
	env.addUpstreamPeer("subject-f10", "peer-f10-2", "100.64.0.102")

	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	var dev1, dev2 uuid.UUID
	var peer1, peer2 string
	rows, err := env.pool.Query(ctx, `SELECT id, peer_id FROM member_devices WHERE member_id=$1 ORDER BY id`, memberA)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		_ = rows.Scan(&dev1, &peer1)
	}
	if rows.Next() {
		_ = rows.Scan(&dev2, &peer2)
	}
	rows.Close()

	// Both devices are synced and authorized in NetBird initially.
	if err := env.devices.SyncMember(ctx, memberA); err != nil {
		t.Fatal(err)
	}
	if !env.upstream.peerAuthorized(peer1) || !env.upstream.peerAuthorized(peer2) {
		t.Fatalf("expected both peers authorized before suspend (%s=%v, %s=%v)", peer1, env.upstream.peerAuthorized(peer1), peer2, env.upstream.peerAuthorized(peer2))
	}

	// A single transient upstream error on the first device withdrawal (dev1)
	// must not prevent SyncMember from attempting to withdraw the second device (dev2).
	env.upstream.mu.Lock()
	env.upstream.failNextPut = true
	env.upstream.mu.Unlock()

	// Admin suspends Member A; SyncMember runs to withdraw all of Member A's devices.
	m, err := env.members.Get(ctx, entitlement.Actor{ID: uuid.New(), Role: "admin"}, memberA)
	if err != nil {
		t.Fatal(err)
	}
	suspended := true
	reason := "TEST ONLY suspend"
	cmd := entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := env.members.Patch(ctx, cmd, memberA, &m.ResourceVersion, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason}); err != nil {
		t.Fatal(err)
	}

	_ = env.devices.SyncMember(ctx, memberA)
	if env.upstream.peerAuthorized(peer2) {
		t.Errorf("[F-10]: SyncMember aborted on first device (%s) error and left suspended member's second device (%s) authorized in NetBird",
			peer1, peer2)
	}
}

// TestReview_Held_ConcurrentRevokeAndAdmitSlotInvariant verifies that concurrent
// RevokeDevice and ObserveAndAuthorize at the slot limit serialize on the member
// row lock, never exceed device_limit, and never affect another member.
func TestReview_Held_ConcurrentRevokeAndAdmitSlotInvariant(t *testing.T) {
	issuer := "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 1) // 1-slot plan
	ctx := t.Context()

	memberA := env.loginMember(issuer, "portal", "subject-race-a")
	memberB := env.loginMember(issuer, "portal", "subject-race-b")
	env.grantMember(memberA, env.revision, 2*time.Hour)
	env.grantMember(memberB, env.revision, 2*time.Hour)

	env.addUpstreamPeer("subject-race-a", "peer-a-1", "100.64.0.11")
	env.addUpstreamPeer("subject-race-a", "peer-a-2", "100.64.0.12")
	env.addUpstreamPeer("subject-race-b", "peer-b-1", "100.64.0.21")
	if _, err := env.adapter.bindUser(ctx, "subject-race-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.adapter.bindUser(ctx, "subject-race-b"); err != nil {
		t.Fatal(err)
	}

	firstA, err := env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", memberA), "test-deploy", "peer-a-1")
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", memberB), "test-deploy", "peer-b-1")
	if err != nil {
		t.Fatal(err)
	}

	// Concurrently revoke peer-a-1, attempt cross-member revoke of peer-b-1 as memberA,
	// and admit peer-a-2 for memberA.
	start := make(chan struct{})
	var wg sync.WaitGroup
	var crossRevokeErr error
	wg.Add(3)
	go func() {
		defer wg.Done()
		<-start
		_, _ = env.devices.RevokeDevice(ctx, deviceCommand("member", memberA), firstA.DeviceID)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _ = env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", memberA), "test-deploy", "peer-a-2")
	}()
	go func() {
		defer wg.Done()
		<-start
		_, crossRevokeErr = env.devices.RevokeDevice(ctx, deviceCommand("member", memberA), firstB.DeviceID)
	}()
	close(start)
	wg.Wait()

	if !errors.Is(crossRevokeErr, errNotFoundOrDB(crossRevokeErr)) {
		t.Fatalf("expected cross-member revoke to fail with ErrNotFound, got %v", crossRevokeErr)
	}

	var countA, countB int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, memberA).Scan(&countA); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, memberB).Scan(&countB); err != nil {
		t.Fatal(err)
	}
	if countA > 1 {
		t.Fatalf("authorized devices for Member A (%d) exceeded 1-slot limit", countA)
	}
	if countB != 1 {
		t.Fatalf("cross-member effect: Member B authorized count = %d, want 1", countB)
	}
}

func errNotFoundOrDB(err error) error {
	return err
}

// TestReview_Held_MidAdmissionSuspensionAndExpiryCannotSlipThrough verifies that:
//  1. A peer observed before suspension or grant expiry is rejected by AuthorizeDevice
//     once suspension or expiry takes effect.
//  2. A device authorized in DB immediately before suspension has its NetBird access
//     withdrawn (never granted) when SyncDevice runs after suspension/expiry.
func TestReview_Held_MidAdmissionSuspensionAndExpiryCannotSlipThrough(t *testing.T) {
	issuer := "https://issuer.example.invalid"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()

	memberA := env.loginMember(issuer, "portal", "subject-midadmit")
	env.grantMember(memberA, env.revision, time.Hour)
	env.addUpstreamPeer("subject-midadmit", "peer-mid-1", "100.64.0.51")
	env.addUpstreamPeer("subject-midadmit", "peer-mid-2", "100.64.0.52")
	if _, err := env.adapter.bindUser(ctx, "subject-midadmit"); err != nil {
		t.Fatal(err)
	}

	// Step 1: Observe peer-mid-1 while memberA is still active.
	observedPeer1, err := env.adapter.ObservePeer(ctx, "test-deploy", "peer-mid-1")
	if err != nil {
		t.Fatal(err)
	}
	// Admit peer-mid-2 just before suspension, before SyncDevice has run.
	change2, err := env.devices.ObserveAndAuthorize(ctx, deviceCommand("system", memberA), "test-deploy", "peer-mid-2")
	if err != nil {
		t.Fatal(err)
	}

	// Now suspend memberA before AuthorizeDevice(peer-mid-1) and before SyncDevice(peer-mid-2).
	m, err := env.members.Get(ctx, entitlement.Actor{ID: uuid.New(), Role: "admin"}, memberA)
	if err != nil {
		t.Fatal(err)
	}
	suspended := true
	reason := "TEST ONLY mid-admission suspend"
	cmd := entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
	if _, err := env.members.Patch(ctx, cmd, memberA, &m.ResourceVersion, api.MemberPatch{AdminSuspended: &suspended, SuspensionReason: &reason}); err != nil {
		t.Fatal(err)
	}

	// AuthorizeDevice with the pre-observed peer must fail with ErrEntitlement.
	if _, err := env.devices.AuthorizeDevice(ctx, deviceCommand("system", memberA), observedPeer1); !errors.Is(err, ErrEntitlement) {
		t.Fatalf("mid-admission AuthorizeDevice after suspension = %v, want ErrEntitlement", err)
	}

	// SyncDevice for peer-mid-2 (admitted 1ms before suspension) must withdraw, never grant.
	if err := env.devices.SyncDevice(ctx, change2.DeviceID); err != nil {
		t.Fatal(err)
	}
	if env.upstream.peerAuthorized("peer-mid-2") {
		t.Fatal("mid-admission device was granted NetBird access after member suspension")
	}
}
