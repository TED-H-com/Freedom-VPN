//go:build installer

package installertest

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
)

func assertMarkerRemoved(t *testing.T, c string) {
	t.Helper()
	if _, code := dexec(t, c, "nft", "list", "table", "inet", "fvpn_install"); code == 0 {
		t.Fatal("removal left installer marker")
	}
	mustExec(t, c, "nft", "list", "table", "inet", "fvpn")
}

func TestAT07_D116InstallerMarkerAndAgentRules(t *testing.T) {
	const c = "fvpn-w5-node-installer-marker"
	startSystemd(t, c)
	stage(t, c)
	defRoute, resolv := seedForeign(t, c)
	url, ca, bundle := startControl(t, c)
	writeStdin(t, c, "/run/w5-token", token1+"\n", "0600")
	_, out, code := doInstall(t, c, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 {
		t.Logf("failed install nft read-back: %s", mustExec(t, c, "nft", "-j", "list", "ruleset"))
		t.Fatalf("install: %d %s", code, out)
	}
	waitPath(t, c, "/run/vpn-node/heartbeat")
	model := enforce.Model{Datapath: enforce.Datapath{
		MemberIngress: "fvmb0", Uplink: "fvup0", SNAT: netip.MustParseAddr("192.0.2.1"),
	}, IPv6Mode: enforce.IPv6Block, Now: time.Now()}
	raw := mustExec(t, c, "nft", "-j", "list", "ruleset")
	if err := enforce.VerifyBaseline([]byte(raw), model); err != nil {
		t.Fatalf("agent baseline verifier: %v", err)
	}
	assertChainSet(t, raw)
	_, verdict := enforce.VerifyServing(context.Background(), containerReadback{t, c}, model, nil)
	t.Logf("D-116: real agent VerifyBaseline=pass; VerifyServing=%v (enrollment-only control supplies no signed desired state or lease)", verdict)
	beat := mustExec(t, c, "cat", "/run/vpn-node/heartbeat")
	for _, line := range strings.Split(beat, "\n") {
		if strings.HasPrefix(line, "serving ") || strings.HasPrefix(line, "reason ") || strings.HasPrefix(line, "baseline_ok ") {
			t.Log(line)
		}
	}
	t.Logf("D-116 live nft list ruleset:\n%s", mustExec(t, c, "nft", "list", "ruleset"))
	// Freeze the agent to prove migration leaves every rendered byte intact.
	mustExec(t, c, "systemctl", "stop", "fvpn-agent")
	before := mustExec(t, c, "nft", "-j", "list", "table", "inet", "fvpn")
	// TEST ONLY: reproduce the exact follow-up 3 ownership layout.
	mustExec(t, c, "nft", "delete", "table", "inet", "fvpn_install")
	mustExec(t, c, "nft", "add", "chain", "inet", "fvpn", "owner", `{ comment "managed-by=fvpn-node-install"; }`)
	writeStdin(t, c, "/var/lib/vpn-node/install/owned.json", `{"created_table":true,"created_netns":true,"created_links":["fvmb0","fvup0"]}`, "0600")
	check := mustCheck(t, c, installer.ExitDrifted)
	if check.Marker.Present || !check.Marker.LegacyOwner {
		t.Fatalf("old marker not reported: %+v", check.Marker)
	}
	var markerBefore string
	for i := 0; i < 2; i++ {
		res, out, code := doInstall(t, c, relIn("1", rel1.ReleaseID), url, ca, bundle, "")
		if code != 0 || res.Status != "already_current" {
			t.Fatalf("rerun: %d %+v %s", code, res, out)
		}
		if after := mustExec(t, c, "nft", "-j", "list", "table", "inet", "fvpn"); before != after {
			t.Fatal("migration/rerun changed agent rules beyond removing legacy owner")
		}
		marker := mustExec(t, c, "nft", "-j", "list", "table", "inet", "fvpn_install")
		if i > 0 && marker != markerBefore {
			t.Fatal("rerun replaced or duplicated marker")
		}
		markerBefore = marker
		assertChainSet(t, mustExec(t, c, "nft", "-j", "list", "ruleset"))
	}
	mustExec(t, c, "systemctl", "start", "fvpn-agent")
	check = mustCheck(t, c, 0)
	if !check.Marker.Present || check.Marker.LegacyOwner {
		t.Fatalf("new marker not reported: %+v", check.Marker)
	}
	assertForeign(t, c, defRoute, resolv)
	t.Log("D-116 migration: legacy owner removed; four rendered chains unchanged; second rerun keeps the same single hookless marker; foreign table/default route/DNS preserved")
}

func assertChainSet(t *testing.T, raw string) {
	t.Helper()
	rs, err := enforce.ParseRuleset([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for name := range rs.Chains {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"forward", "input", "output", "postrouting"}) {
		t.Fatalf("agent chain set: %v", names)
	}
	markers := 0
	for _, ch := range rs.Foreign {
		if ch.Family == "inet" && ch.Table == "fvpn_install" {
			markers++
			if ch.Name != "owner" || ch.Hook != "" || ch.HasPrio || len(ch.Rules) != 0 {
				t.Fatalf("not a regular empty marker: %+v", ch)
			}
		}
	}
	if markers != 1 {
		t.Fatalf("marker count: %d", markers)
	}
}

// All privileged readbacks run in this test's dedicated container. No verifier
// result is substituted, and no nft/tc writes are permitted through this adapter.
type containerReadback struct {
	t *testing.T
	c string
}

func (r containerReadback) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	out, code := dexec(r.t, r.c, append([]string{name}, args...)...)
	if code != 0 {
		return nil, fmt.Errorf("container readback %s exited %d", name, code)
	}
	return []byte(out), nil
}
func (containerReadback) LookPath(name string) (string, error)   { return name, nil }
func (containerReadback) NftBatch(context.Context, string) error { return fmt.Errorf("read only") }
func (containerReadback) TCBatch(context.Context, string) error  { return fmt.Errorf("read only") }
