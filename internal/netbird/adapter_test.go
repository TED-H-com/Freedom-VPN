package netbird

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type testToken string

func (s testToken) Token(context.Context) (string, error) { return string(s), nil }

type fake struct {
	mu             sync.Mutex
	objects        map[string]map[string]object
	templates      map[string]object
	writes, serial int
	lose, stale    string
}

func newFake(t *testing.T) (*fake, *Client) {
	t.Helper()
	f := &fake{objects: map[string]map[string]object{}, templates: map[string]object{}}
	for _, kind := range []string{"groups", "routes", "policies", "peers", "setup-keys", "users", "dns/nameservers"} {
		f.objects[kind] = map[string]object{}
	}
	for kind, file := range map[string]string{"groups": "group", "routes": "route", "peers": "peer", "policies": "policy", "dns/nameservers": "nameserver"} {
		b, e := os.ReadFile(filepath.Join("testdata", file+".json"))
		if e != nil {
			t.Fatal(e)
		}
		var o object
		if e = json.Unmarshal(b, &o); e != nil {
			t.Fatal(e)
		}
		f.templates[kind] = o
	}
	f.objects["groups"]["TEST-all"] = object{"id": "TEST-all", "name": "All", "peers": []any{}, "resources": []any{}}
	f.objects["peers"]["TEST-node"] = object{"id": "TEST-node", "name": "TEST ONLY node", "user_id": "", "ephemeral": false, "ip": "100.64.0.10"}
	f.objects["peers"]["TEST-device"] = object{"id": "TEST-device", "name": "TEST ONLY member", "user_id": "TEST-user", "ephemeral": false, "ip": "100.64.0.11"}
	f.objects["peers"]["TEST-device-b"] = object{"id": "TEST-device-b", "name": "TEST ONLY member B", "user_id": "TEST-user-b", "ephemeral": false, "ip": "100.64.0.12"}
	f.objects["users"]["TEST-user"] = object{"id": "TEST-user", "auto_groups": []any{}, "is_service_user": false, "is_blocked": false}
	f.objects["users"]["TEST-user-b"] = object{"id": "TEST-user-b", "auto_groups": []any{}, "is_service_user": false, "is_blocked": false}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	c, e := New(Config{BaseURL: srv.URL + "/api", DeploymentID: "TEST ONLY", Token: testToken("TEST ONLY synthetic token"), Bindings: NewMemoryBindingStore(), AllowInsecureHTTP: true, Timeout: time.Second})
	if e != nil {
		t.Fatal(e)
	}
	return f, c
}
func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Token TEST ONLY synthetic token" {
		w.WriteHeader(401)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(path, "/")
	kind := parts[0]
	id := ""
	if kind == "dns" && len(parts) > 1 {
		kind += "/" + parts[1]
		parts = parts[1:]
	}
	if len(parts) > 1 {
		id = parts[1]
	}
	bucket, ok := f.objects[kind]
	if !ok {
		w.WriteHeader(404)
		return
	}
	respond := func(v any) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
	if r.Method == "GET" {
		if id != "" {
			o, ok := bucket[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			respond(o)
		} else {
			a := []object{}
			for _, o := range bucket {
				a = append(a, o)
			}
			respond(a)
		}
		return
	}
	f.writes++
	if r.Method == "DELETE" {
		delete(bucket, id)
		respond(object{})
		return
	}
	var request object
	if json.NewDecoder(r.Body).Decode(&request) != nil {
		w.WriteHeader(400)
		return
	}
	if kind == "routes" {
		if _, ok := request["peer_groups"]; ok {
			w.WriteHeader(400)
			return
		}
	}
	if r.Method == "POST" {
		f.serial++
		id = fmt.Sprintf("TEST-object-%d", f.serial)
	} else if bucket[id] == nil {
		w.WriteHeader(404)
		return
	}
	reply := object{}
	for k, v := range f.templates[kind] {
		reply[k] = v
	}
	for k, v := range request {
		reply[k] = v
	}
	reply["id"] = id
	switch kind {
	case "groups":
		a := []any{}
		for _, p := range ids(request["peers"]) {
			a = append(a, object{"id": p, "name": "TEST ONLY peer"})
		}
		reply["peers"] = a
		reply["peers_count"] = len(a)
	case "policies":
		rules := []any{}
		for _, r := range request["rules"].([]any) {
			m := r.(map[string]any)
			m["id"] = "TEST-rule"
			for _, side := range []string{"sources", "destinations"} {
				a := []any{}
				for _, g := range ids(m[side]) {
					a = append(a, object{"id": g, "name": "TEST ONLY group", "peers_count": 0, "resources_count": 0})
				}
				m[side] = a
			}
			rules = append(rules, m)
		}
		reply["rules"] = rules
	case "setup-keys":
		reply["key"] = "TEST ONLY generated setup secret"
		reply["expires"] = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	}
	if kind != f.stale {
		bucket[id] = decodeObject(reply)
	}
	if kind == "setup-keys" && bucket[id] != nil {
		delete(bucket[id], "key")
	}
	if f.lose == kind {
		f.lose = ""
		conn, _, e := w.(http.Hijacker).Hijack()
		if e == nil {
			conn.Close()
		}
		return
	}
	respond(reply)
}
func testDevice() MemberDevice {
	return MemberDevice{"TEST-member", "TEST-device", "TEST-device", "TEST-user", "TEST-identity", "1"}
}
func testExit(group string) Exit {
	return Exit{"TEST-exit", "TEST-node", "TEST-incarnation", "TEST-node", "TEST ONLY exit", []string{group}}
}
func require(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func TestAT08_PublicationReadBackAndNoop(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	route, e := c.PublishExit(ctx, testExit(g))
	require(t, e)
	if route.Masquerade || !route.SkipAutoApply || route.Network != "0.0.0.0/0" || route.Peer != "TEST-node" {
		t.Fatal("unsafe route")
	}
	before := f.writes
	_, e = c.PublishExit(ctx, testExit(g))
	require(t, e)
	if before != f.writes {
		t.Fatal("repeat publication wrote upstream")
	}
	require(t, c.WithdrawExit(ctx, "TEST-exit"))
	require(t, c.WithdrawExit(ctx, "TEST-exit"))
	if len(f.objects["routes"]) != 0 || len(f.objects["policies"]) != 0 {
		t.Fatal("withdrawal left route/policy")
	}
}
func TestAT08_LostCreateResponsesDoNotDuplicate(t *testing.T) {
	for _, kind := range []string{"groups", "policies", "routes"} {
		t.Run(kind, func(t *testing.T) {
			f, c := newFake(t)
			ctx := context.Background()
			g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
			require(t, e)
			f.lose = kind
			_, e = c.PublishExit(ctx, testExit(g))
			if !IsKind(e, Unavailable) {
				t.Fatalf("expected unknown write result, got %v", e)
			}
			_, e = c.PublishExit(ctx, testExit(g))
			require(t, e)
			if len(f.objects["routes"]) != 1 || len(f.objects["policies"]) != 1 || len(f.objects["groups"]) != 3 {
				t.Fatal("retry duplicated a resource")
			}
		})
	}
}
func TestAT02_MappingIsNotAuthorization(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	_, e = c.EnsureMemberDevice(ctx, testDevice())
	require(t, e)
	if len(ids(f.objects["groups"][g]["peers"])) != 0 {
		t.Fatal("mapping granted access")
	}
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), []string{g}))
	n := f.writes
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), []string{g}))
	if n != f.writes {
		t.Fatal("repeat authorization wrote")
	}
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), nil))
	if len(ids(f.objects["groups"][g]["peers"])) != 0 {
		t.Fatal("revoke left paid membership")
	}
	d := testDevice()
	d.UserID = "TEST-other"
	if _, e = c.EnsureMemberDevice(ctx, d); !IsKind(e, Conflict) {
		t.Fatal("wrong owner accepted")
	}
	d = testDevice()
	d.PeerID = "TEST-node"
	if _, e = c.EnsureMemberDevice(ctx, d); !IsKind(e, Conflict) {
		t.Fatal("setup-key peer accepted as member")
	}
	d = testDevice()
	d.DeviceID = "TEST-another-device"
	if _, e = c.EnsureMemberDevice(ctx, d); !IsKind(e, Conflict) {
		t.Fatal("peer rebound to another device")
	}
	f.objects["users"]["TEST-user"]["auto_groups"] = []any{g}
	if _, e = c.EnsureMemberDevice(ctx, testDevice()); !IsKind(e, Conflict) {
		t.Fatal("user auto-group accepted")
	}
}
func TestAT08_ConcurrentAuthorizationPreservesBothDevices(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	d := testDevice()
	d2 := MemberDevice{"TEST-member-b", "TEST-device-b", "TEST-device-b", "TEST-user-b", "TEST-identity-b", "1"}
	results := make(chan error, 2)
	for _, v := range []MemberDevice{d, d2} {
		go func() { results <- c.ApplyDeviceAuthorization(ctx, v, []string{g}) }()
	}
	require(t, <-results)
	require(t, <-results)
	if len(ids(f.objects["groups"][g]["peers"])) != 2 {
		t.Fatal("concurrent group write lost a member")
	}
}
func TestENR04_OneOffSecretAndLostResponse(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(fmt.Sprint(lost), func(t *testing.T) {
			f, c := newFake(t)
			ctx := context.Background()
			if lost {
				f.lose = "setup-keys"
			}
			k, e := c.CreateNodeSetupKey(ctx, "TEST-enrollment")
			if lost {
				if !IsKind(e, Unavailable) {
					t.Fatal(e)
				}
			} else {
				require(t, e)
				if !k.SecretAvailable || k.Secret.Reveal() == "" {
					t.Fatal("missing first secret")
				}
				if strings.Contains(fmt.Sprintf("%+v %#v", k, k.Secret), k.Secret.Reveal()) {
					t.Fatal("secret leaked in formatting")
				}
			}
			k, e = c.CreateNodeSetupKey(ctx, "TEST-enrollment")
			require(t, e)
			if k.SecretAvailable || k.Secret.Reveal() != "" || len(f.objects["setup-keys"]) != 1 {
				t.Fatal("retry recreated/revealed secret")
			}
		})
	}
}
func TestAT08_DriftDuplicatesAndUnconfirmedWrites(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	r, e := c.PublishExit(ctx, testExit(g))
	require(t, e)
	f.objects["routes"][r.ID]["masquerade"] = true
	_, e = c.PublishExit(ctx, testExit(g))
	require(t, e)
	f.objects["routes"]["TEST-duplicate"] = decodeObject(f.objects["routes"][r.ID])
	f.objects["routes"]["TEST-duplicate"]["id"] = "TEST-duplicate"
	_, e = c.PublishExit(ctx, testExit(g))
	if !IsKind(e, Conflict) {
		t.Fatal("duplicate route not rejected")
	}
	delete(f.objects["routes"], "TEST-duplicate")
	f.stale = "routes"
	x := testExit(g)
	x.Description = "TEST ONLY changed"
	_, e = c.PublishExit(ctx, x)
	if !IsKind(e, Unconfirmed) {
		t.Fatal("acknowledged but unapplied write counted as success")
	}
}
func TestAT28_ErrorRedactionAndRetry(t *testing.T) {
	for _, status := range []int{401, 403, 404, 409, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(status)
				fmt.Fprint(w, r.Header.Get("Authorization"))
			}))
			defer srv.Close()
			c, e := New(Config{BaseURL: srv.URL + "/api", DeploymentID: "TEST", Token: testToken("TEST ONLY private token"), Bindings: NewMemoryBindingStore(), AllowInsecureHTTP: true, Timeout: 2 * time.Second, ReadRetries: 1})
			require(t, e)
			_, e = c.ReadObservedState(context.Background())
			if e == nil || strings.Contains(e.Error(), "private token") {
				t.Fatal("error leaked token")
			}
			want := 1
			if status == 429 || status == 500 {
				want = 2
			}
			if calls != want {
				t.Fatalf("read attempts = %d, want %d", calls, want)
			}
			before := calls
			e = c.call(context.Background(), "POST", "/groups", object{}, &object{})
			if e == nil || calls != before+1 {
				t.Fatal("unsafe mutation retried")
			}
			if status == 429 {
				var typed *Error
				if !errors.As(e, &typed) || typed.RetryAfter != time.Second {
					t.Fatal("retry-after missing")
				}
			}
		})
	}
}
func TestAT28_CancellationRedirectAndTokenFile(t *testing.T) {
	_, c := newFake(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := c.ReadObservedState(ctx)
	if !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	p := filepath.Join(t.TempDir(), "token")
	require(t, os.WriteFile(p, []byte("TEST ONLY token"), 0644))
	if _, e := TokenFile(p).Token(context.Background()); !IsKind(e, Unauthorized) {
		t.Fatal("world-readable token accepted")
	}
	require(t, os.Chmod(p, 0600))
	if _, e := TokenFile(p).Token(context.Background()); e != nil {
		t.Fatal(e)
	}
	targetHits := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetHits++ }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	redirected, e := New(Config{BaseURL: source.URL + "/api", DeploymentID: "TEST", Token: testToken("TEST ONLY token"), Bindings: NewMemoryBindingStore(), AllowInsecureHTTP: true})
	require(t, e)
	_, e = redirected.ReadObservedState(context.Background())
	if e == nil || targetHits > 0 {
		t.Fatal("redirect forwarded credentials")
	}
}
func TestAT08_DNSAndOpsAreNarrow(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	a := AccessConfig{PaidGroupIDs: []string{g}, ResolverPeerIDs: []string{"TEST-device", "TEST-device-b"}, ResolverIPs: []string{"100.64.0.11", "100.64.0.12"}, AdminPeerIDs: []string{}, OpsPeerIDs: []string{"TEST-node"}, OpsPorts: []string{"9443"}}
	require(t, c.BootstrapAccess(ctx, a))
	n := f.writes
	require(t, c.BootstrapAccess(ctx, a))
	if n != f.writes {
		t.Fatal("bootstrap not idempotent")
	}
	for _, p := range f.objects["policies"] {
		for _, r := range p["rules"].([]any) {
			rule := r.(map[string]any)
			if rule["bidirectional"] != false || len(ids(rule["ports"])) != 1 || rule["protocol"] == "all" {
				t.Fatal("broad DNS/ops exception")
			}
		}
	}
	if len(f.objects["dns/nameservers"]) != 1 {
		t.Fatal("missing resolver distribution")
	}
}

func TestAT28_TimeoutAndLockCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	c, e := New(Config{BaseURL: srv.URL + "/api", DeploymentID: "TEST", Token: testToken("TEST ONLY"), Bindings: NewMemoryBindingStore(), Timeout: 20 * time.Millisecond, AllowInsecureHTTP: true})
	require(t, e)
	_, e = c.ReadObservedState(context.Background())
	if !IsKind(e, Unavailable) {
		t.Fatal(e)
	}
	s := NewMemoryBindingStore()
	unlock, e := s.Lock(context.Background(), "TEST")
	require(t, e)
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, e = s.Lock(ctx, "TEST")
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
}
func TestAT08_DefaultRemovalAndPeerDeletion(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	f.objects["policies"]["TEST-default"] = decodeObject(object{"id": "TEST-default", "name": "Default", "rules": []object{rule("Default", "all", []string{"TEST-all"}, []string{"TEST-all"}, nil, true)}})
	require(t, c.RemoveDefaultPolicy(ctx))
	require(t, c.RemoveDefaultPolicy(ctx))
	if len(f.objects["policies"]) != 0 {
		t.Fatal("default survived")
	}
	_, e := c.EnsureNodePeer(ctx, NodePeer{"TEST-node", "TEST-incarnation", "TEST-node"})
	require(t, e)
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	_, e = c.PublishExit(ctx, testExit(g))
	require(t, e)
	if e = c.DeletePeer(ctx, "node:TEST-node:TEST-incarnation"); !IsKind(e, Conflict) {
		t.Fatal("published peer deleted")
	}
	require(t, c.WithdrawExit(ctx, "TEST-exit"))
	require(t, c.DeletePeer(ctx, "node:TEST-node:TEST-incarnation"))
	require(t, c.DeletePeer(ctx, "node:TEST-node:TEST-incarnation"))
	s, e := c.ReadObservedState(ctx)
	require(t, e)
	if len(s.Peers) != 2 || len(s.Routes) != 0 || s.ObservedAt.IsZero() {
		t.Fatal("observed state is not current")
	}
	f.objects["policies"]["TEST-broad"] = decodeObject(object{"id": "TEST-broad", "name": "Unexpected", "rules": []object{rule("Unexpected", "all", []string{"TEST-all"}, []string{"TEST-all"}, nil, true)}})
	if e = c.RemoveDefaultPolicy(ctx); !IsKind(e, Conflict) {
		t.Fatal("unknown broad policy accepted")
	}
}
func TestAT28_CanonicalHash(t *testing.T) {
	s, e := canonical(object{"b": "\u2028<\n", "a": 1})
	require(t, e)
	if s != "{\"a\":1,\"b\":\"\u2028<\\n\"}" {
		t.Fatalf("wrong JCS: %q", s)
	}
	for _, v := range []any{1.5, float64(9007199254740992), object{"非ASCII": 1}} {
		if _, e := canonical(v); e == nil {
			t.Fatal("unsupported canonical input accepted")
		}
	}
}

func TestMEM03_LostAuthorizationWrite(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	f.lose = "groups"
	if e = c.ApplyDeviceAuthorization(ctx, testDevice(), []string{g}); !IsKind(e, Unavailable) {
		t.Fatal(e)
	}
	writes := f.writes
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), []string{g}))
	if f.writes != writes {
		t.Fatal("lost PUT retry wrote again")
	}
	f.lose = "groups"
	if e = c.ApplyDeviceAuthorization(ctx, testDevice(), nil); !IsKind(e, Unavailable) {
		t.Fatal(e)
	}
	writes = f.writes
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), nil))
	if f.writes != writes {
		t.Fatal("lost revocation retry wrote again")
	}
}

func TestMEM03_BlockedUserCanStillBeRevoked(t *testing.T) {
	f, c := newFake(t)
	ctx := context.Background()
	g, e := c.EnsurePaidGroup(ctx, "TEST-plan")
	require(t, e)
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), []string{g}))
	f.objects["users"]["TEST-user"]["is_blocked"] = true
	if e = c.ApplyDeviceAuthorization(ctx, testDevice(), []string{g}); !IsKind(e, Conflict) {
		t.Fatal("blocked user granted access")
	}
	wrong := testDevice()
	wrong.MemberID = "TEST-other-member"
	if e = c.ApplyDeviceAuthorization(ctx, wrong, nil); !IsKind(e, Conflict) {
		t.Fatal("foreign device binding accepted")
	}
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), nil))
	if len(ids(f.objects["groups"][g]["peers"])) != 0 {
		t.Fatal("blocked user retained paid access")
	}
	delete(f.objects["users"], "TEST-user")
	require(t, c.ApplyDeviceAuthorization(ctx, testDevice(), nil))
}
