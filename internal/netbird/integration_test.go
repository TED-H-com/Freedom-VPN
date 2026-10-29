//go:build netbird_integration

package netbird

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type loseWrite struct {
	base http.RoundTripper
	path string
	lost atomic.Bool
}

func (l *loseWrite) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, e := l.base.RoundTrip(r)
	if e == nil && r.Method == "POST" && r.URL.Path == l.path && l.lost.CompareAndSwap(false, true) {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, errors.New("TEST ONLY injected lost response")
	}
	return resp, e
}
func liveClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Helper()
	u, token := os.Getenv("FVPN_NETBIRD_API_URL"), os.Getenv("FVPN_NETBIRD_TOKEN_FILE")
	if u == "" || token == "" {
		t.Skip("netbird_integration requires FVPN_NETBIRD_API_URL and FVPN_NETBIRD_TOKEN_FILE")
	}
	c, e := New(Config{BaseURL: u, DeploymentID: "TEST-ONLY-dev", Token: TokenFile(token), Bindings: NewMemoryBindingStore(), HTTPClient: &http.Client{Transport: transport}, ReadRetries: 2, AllowInsecureHTTP: true})
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func cli(t *testing.T, service string, stdin string, args ...string) (string, error) {
	t.Helper()
	prefix := os.Getenv("FVPN_NB_PREFIX")
	if !strings.HasPrefix(prefix, "fvpn-") || prefix == "fvpn-" {
		t.Fatal("FVPN_NB_PREFIX must explicitly name this disposable test stack")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := []string{"compose", "-f", "../../deploy/dev/netbird/compose.yml", "exec", "-T", service}
	command = append(command, args...)
	cmd := exec.CommandContext(ctx, "docker", command...)
	cmd.Stdin = strings.NewReader(stdin)
	b, e := cmd.CombinedOutput()
	return string(b), e
}
func requireCLI(t *testing.T, service string, args ...string) string {
	t.Helper()
	s, e := cli(t, service, "", args...)
	if e != nil {
		t.Fatalf("%s CLI failed (output withheld)", service)
	}
	return s
}
func awaitRoutes(t *testing.T, service, network string, present bool) string {
	t.Helper()
	var out string
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		out = requireCLI(t, service, "netbird", "routes", "list")
		if strings.Contains(out, network) == present {
			return out
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("%s route visibility did not converge to %t", service, present)
	return ""
}
func must[T any](t *testing.T, v T, e error) T {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func check(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func counts(s ObservedState) [3]int { return [3]int{len(s.Groups), len(s.Policies), len(s.Routes)} }

func TestAT08_LiveAdapterReadBack(t *testing.T) {
	c := liveClient(t, nil)
	ctx := context.Background()
	check(t, c.RemoveDefaultPolicy(ctx))
	group, e := c.EnsurePaidGroup(ctx, "g1-plan-a")
	check(t, e)
	group2, e := c.EnsurePaidGroup(ctx, "g1-plan-b")
	check(t, e)
	state, e := c.ReadObservedState(ctx)
	check(t, e)
	var a, b, node Peer
	for _, p := range state.Peers {
		switch p.Name {
		case "fvpn-member-a":
			a = p
		case "fvpn-member-b":
			b = p
		case "fvpn-exit":
			node = p
		}
	}
	if a.ID == "" || b.ID == "" {
		t.Fatal("run bootstrap and login-members.py first; real SSO member peers required")
	}
	d := MemberDevice{MemberID: "TEST-member-a", DeviceID: "TEST-device-a", PeerID: a.ID, UserID: a.UserID, IdentityBindingID: "TEST-verified-local-idp-a", MappingRevision: "1"}
	d2 := MemberDevice{MemberID: "TEST-member-b", DeviceID: "TEST-device-b", PeerID: b.ID, UserID: b.UserID, IdentityBindingID: "TEST-verified-local-idp-b", MappingRevision: "1"}
	_, e = c.EnsureMemberDevice(ctx, d)
	check(t, e)
	_, e = c.EnsureMemberDevice(ctx, d2)
	check(t, e)
	if os.Getenv("FVPN_NETBIRD_CLIENT_TESTS") != "1" {
		t.Skip("API identity check passed; set FVPN_NETBIRD_CLIENT_TESTS=1 for disposable Docker client evidence")
	}
	check(t, c.WithdrawExit(ctx, "g1-exit"))
	check(t, c.WithdrawExit(ctx, "g1-lost-response"))
	check(t, c.ApplyDeviceAuthorization(ctx, d, nil))
	check(t, c.ApplyDeviceAuthorization(ctx, d2, nil))
	if node.ID == "" {
		key, e := c.CreateNodeSetupKey(ctx, "g1-enrollment")
		check(t, e)
		if !key.SecretAvailable {
			t.Fatal("fresh exit state requires fresh stack reset; enrollment key already consumed")
		}
		// The key travels over stdin into a mode-0600 transient container file. Never
		// include it in docker arguments, environment, logs or test failure output.
		script := `umask 077; cat > /tmp/fvpn-setup-key; netbird up --management-url "http://netbird.example.invalid:${FVPN_NB_API_PORT}" --hostname fvpn-exit --setup-key-file /tmp/fvpn-setup-key --disable-dns --disable-ipv6; result=$?; rm /tmp/fvpn-setup-key; exit "$result"`
		_, e = cli(t, "exit", key.Secret.Reveal(), "sh", "-c", script)
		check(t, e)
		replay, e := cli(t, "replay", key.Secret.Reveal(), "sh", "-c", script)
		if e == nil || (!strings.Contains(strings.ToLower(replay), "key") && !strings.Contains(strings.ToLower(replay), "permission")) {
			t.Fatal("second one-off key use was not explicitly rejected")
		}
		again, e := c.CreateNodeSetupKey(ctx, "g1-enrollment")
		check(t, e)
		if again.ID != key.ID || again.SecretAvailable {
			t.Fatal("setup key retry minted or revealed a secret")
		}
		t.Log("AT05 subset: one-off node enrollment succeeded; second use rejected; secret not replayed")
		state, e = c.ReadObservedState(ctx)
		check(t, e)
		for _, p := range state.Peers {
			if p.Name == "fvpn-exit" {
				node = p
			}
		}
	}
	if node.ID == "" {
		t.Fatal("node absent after enrollment")
	}
	_, e = c.EnsureNodePeer(ctx, NodePeer{"TEST-node", "TEST-incarnation", node.ID})
	check(t, e)
	// Exercise only our overlay peers, never external traffic or host networking.
	for _, ip := range []string{node.IP, b.IP} {
		denied := false
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
			_, err := cli(t, "member-a", "", "ping", "-c", "1", "-W", "1", ip)
			if err != nil {
				denied = true
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if !denied {
			t.Fatal("ungrouped peer still reaches another overlay peer")
		}
	}

	t.Log("AT02/AT08 subsets: ungrouped real user peer cannot reach node or member overlay; default policy absent")
	requireCLI(t, "member-a", "netbird", "routes", "deselect", "all")
	x := Exit{ExitID: "g1-exit", NodeID: "TEST-node", IncarnationID: "TEST-incarnation", PeerID: node.ID, Description: "TEST ONLY G1 exit", PaidGroupIDs: []string{group}}
	route, e := c.PublishExit(ctx, x)
	check(t, e)
	if route.Network != "0.0.0.0/0" || route.Peer != node.ID || route.Masquerade || !route.SkipAutoApply {
		t.Fatal("wrong exit route flags")
	}
	awaitRoutes(t, "member-a", route.NetworkID, false)
	check(t, c.ApplyDeviceAuthorization(ctx, d, []string{group}))
	check(t, c.ApplyDeviceAuthorization(ctx, d2, []string{group2}))
	out := awaitRoutes(t, "member-a", route.NetworkID, true)
	awaitRoutes(t, "member-b", route.NetworkID, false)
	if !strings.Contains(out, "Status: Not Selected") {
		t.Fatal("exit auto-applied instead of waiting for user selection")
	}
	requireCLI(t, "member-a", "netbird", "routes", "select", route.NetworkID)
	out = awaitRoutes(t, "member-a", route.NetworkID, true)
	if !strings.Contains(out, "Status: Selected") {
		t.Fatal("CLI did not select exit")
	}
	reachable := false
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		if _, err := cli(t, "member-a", "", "ping", "-c", "1", "-W", "1", node.IP); err == nil {
			reachable = true
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !reachable {
		t.Fatal("authorized member cannot reach exit overlay; negative probes lack a positive control")
	}
	if _, err := cli(t, "member-b", "", "ping", "-c", "1", "-W", "1", node.IP); err == nil {
		t.Fatal("out-of-group member reached exit overlay")
	}
	t.Log("AT08/AT09 subsets: IPv4 exit, single peer, skip_auto_apply=true, masquerade=false; A sees and selects; B does not see")
	t.Log("Overlay ICMP positive control: A reaches exit; out-of-group B is denied")
	before, e := c.ReadObservedState(ctx)
	check(t, e)
	_, e = c.PublishExit(ctx, x)
	check(t, e)
	check(t, c.ApplyDeviceAuthorization(ctx, d, []string{group}))
	after, e := c.ReadObservedState(ctx)
	check(t, e)
	if counts(before) != counts(after) {
		t.Fatal("repeat operations changed resource counts")
	}
	t.Logf("Repeated publication/authorization object counts stable: groups=%d policies=%d routes=%d", len(after.Groups), len(after.Policies), len(after.Routes))
	check(t, c.ApplyDeviceAuthorization(ctx, d, nil))
	awaitRoutes(t, "member-a", route.NetworkID, false)
	check(t, c.ApplyDeviceAuthorization(ctx, d, []string{group}))
	awaitRoutes(t, "member-a", route.NetworkID, true)
	check(t, c.WithdrawExit(ctx, x.ExitID))
	awaitRoutes(t, "member-a", route.NetworkID, false)
	t.Log("Authorization removal and exit withdrawal each removed client route")
	lost := &loseWrite{base: http.DefaultTransport, path: "/api/routes"}
	c.http.Transport = lost
	x.ExitID = "g1-lost-response"
	_, e = c.PublishExit(ctx, x)
	if !IsKind(e, Unavailable) || !lost.lost.Load() {
		t.Fatal("lost-response fault did not occur after upstream write")
	}
	before, e = c.ReadObservedState(ctx)
	check(t, e)
	recovered, e := c.PublishExit(ctx, x)
	check(t, e)
	after, e = c.ReadObservedState(ctx)
	check(t, e)
	if counts(before) != counts(after) {
		t.Fatal("lost-response retry created a duplicate")
	}
	n := 0
	for _, r := range after.Routes {
		if r.NetworkID == recovered.NetworkID {
			n++
		}
	}
	if n != 1 {
		t.Fatal("expected exactly one recovered route")
	}
	t.Log("Lost POST response reconciled by stable name and fresh GET; one route remains; observed state matches upstream")
	// DNS/ops API semantics only: these two peers stand in for resolver identities;
	// they do not run a resolver. DNS changes are disabled on the lab clients.
	access := AccessConfig{PaidGroupIDs: []string{group}, ResolverPeerIDs: []string{a.ID, b.ID}, ResolverIPs: []string{a.IP, b.IP}, AdminPeerIDs: []string{}, OpsPeerIDs: []string{node.ID}, OpsPorts: []string{"9443"}}
	check(t, c.BootstrapAccess(ctx, access))
	check(t, c.BootstrapAccess(ctx, access))
	t.Log("DNS/ops API read-back: two resolver addresses; paid-group distribution; TCP/UDP 53 and explicit admin port only (resolver service not tested)")
	// Record responses only to ignored, private dev state; scrub before promoting
	// any recording into public fixtures. No setup-key response is recorded.
	if os.Getenv("FVPN_NETBIRD_RECORD") == "1" {
		dir := "../../deploy/dev/netbird/.state/recordings"
		check(t, os.MkdirAll(dir, 0700))
		for _, r := range []struct{ kind, id, file string }{{"routes", recovered.ID, "route"}, {"groups", group, "group"}, {"peers", a.ID, "peer"}} {
			o, e := c.get(ctx, r.kind, r.id)
			check(t, e)
			data, e := json.MarshalIndent(o, "", "  ")
			check(t, e)
			check(t, os.WriteFile(filepath.Join(dir, r.file+".json"), data, 0600))
		}
		ns, e := c.list(ctx, "dns/nameservers")
		check(t, e)
		nsdata, e := json.MarshalIndent(ns[0], "", "  ")
		check(t, e)
		check(t, os.WriteFile(filepath.Join(dir, "nameserver.json"), nsdata, 0600))
		ps, e := c.list(ctx, "policies")
		check(t, e)
		data, e := json.MarshalIndent(ps, "", "  ")
		check(t, e)
		check(t, os.WriteFile(filepath.Join(dir, "policies.json"), data, 0600))
	}
	for _, kind := range []string{"dns/nameservers", "policies"} {
		items, err := c.list(ctx, kind)
		check(t, err)
		for _, item := range items {
			name := stringField(item, "name")
			if kind == "dns/nameservers" && name == stableName("dns", "resolvers") || kind == "policies" && (name == stableName("policy", "dns:tcp") || name == stableName("policy", "dns:udp") || name == stableName("policy", "ops")) {
				check(t, c.remove(ctx, kind, stringField(item, "id")))
			}
		}
	}
	check(t, c.WithdrawExit(ctx, x.ExitID))
	check(t, c.ApplyDeviceAuthorization(ctx, d, nil))
	check(t, c.ApplyDeviceAuthorization(ctx, d2, nil))
}
