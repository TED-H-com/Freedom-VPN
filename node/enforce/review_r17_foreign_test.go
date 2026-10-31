package enforce

import (
	"strings"
	"testing"
)

func reviewDatapath() Model {
	return Model{Datapath: Datapath{MemberIngress: "fv-mi", Uplink: "eth0"}}
}

func foreignBase(family, table, hook, policy string, dev []string) ForeignChain {
	return ForeignChain{
		Family: family, Table: table, Name: "owner", Hook: hook, Policy: policy,
		Dev: dev, Prio: 0, HasPrio: hook != "",
	}
}

func TestReviewR17_InstallerMarkerIsNotASpecialForeignLever(t *testing.T) {
	m := reviewDatapath()
	hookless := Ruleset{Foreign: []ForeignChain{foreignBase("inet", "fvpn_install", "", "accept", nil)}}
	if err := verifyForeign(hookless, m); err != nil {
		t.Fatalf("hookless installer marker: %v", err)
	}
	hooked := Ruleset{Foreign: []ForeignChain{foreignBase("inet", "fvpn_install", "forward", "drop", nil)}}
	hookedErr := verifyForeign(hooked, m)
	if ReasonCode(hookedErr) != ReasonIsolation || !strings.Contains(hookedErr.Error(), "fvpn_install") {
		t.Fatalf("hooked drop: %v", hookedErr)
	}
	for _, family := range []string{"ip", "ip6", "bridge", "netdev"} {
		err := verifyForeign(Ruleset{Foreign: []ForeignChain{foreignBase(family, "filter", "forward", "drop", nil)}}, m)
		if ReasonCode(err) != ReasonIsolation {
			t.Fatalf("%s forward drop was invisible: %v", family, err)
		}
	}
	if err := verifyForeign(Ruleset{Foreign: []ForeignChain{foreignBase("arp", "filter", "forward", "drop", nil)}}, m); err != nil {
		t.Fatalf("arp is not a forwarded-member family: %v", err)
	}
	if err := verifyForeign(Ruleset{Foreign: []ForeignChain{foreignBase("ip", "filter", "input", "drop", nil)}}, m); err != nil {
		t.Fatalf("input hook: %v", err)
	}
	lo := foreignBase("netdev", "filter", "ingress", "drop", []string{"lo"})
	if err := verifyForeign(Ruleset{Foreign: []ForeignChain{lo}}, m); err != nil {
		t.Fatalf("ingress on an unrelated device: %v", err)
	}
	uplink := foreignBase("netdev", "filter", "ingress", "drop", []string{"eth0"})
	if ReasonCode(verifyForeign(Ruleset{Foreign: []ForeignChain{uplink}}, m)) != ReasonIsolation {
		t.Fatal("ingress on the uplink was invisible")
	}
	// An accept-only base chain is the same tolerance as any other foreign chain.
	accept := Ruleset{Foreign: []ForeignChain{foreignBase("inet", "fvpn_install", "forward", "accept", nil)}}
	if err := verifyForeign(accept, m); err != nil {
		t.Fatalf("accept-only base chain was a special failure: %v", err)
	}
}
