//go:build e2e

package e2e

import "testing"

func TestParseForwardChainIgnoresOtherAccepts(t *testing.T) {
	baseline := `
table inet fvpn {
	chain forward {
		type filter hook forward priority filter; policy drop;
		counter name "fvpn_base" drop comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1"
		iifname "fvmb0" counter name "fvpn_base" drop
		oifname "fvmb0" counter name "fvpn_base" drop
	}
}
table inet fvpn {
	chain input {
		type filter hook input priority filter; policy accept;
	}
}
`
	got := parseForwardChain(baseline)
	if !got.Present || got.Policy != "drop" || !got.Baseline || got.MemberAccept != 0 {
		t.Fatalf("baseline: %+v", got)
	}
	serving := `
	chain forward {
		type filter hook forward priority filter; policy drop;
		iifname "fvmb0" oifname "fvup0" ip saddr 100.64.0.2 counter name "up" accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.64.0.2 counter name "down" accept
	}
`
	got = parseForwardChain(serving)
	if got.Policy != "drop" || got.Baseline || got.MemberAccept != 2 {
		t.Fatalf("serving: %+v", got)
	}
	// NetBird's own accepts share the forward chain. They are not member rules.
	netbird := `
	chain forward {
		type filter hook forward priority mangle; policy drop;
		oifname "fvnb0" ct state established,related counter packets 0 bytes 0 accept
		iifname "fvnb0" counter packets 0 bytes 0 accept
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1"
		iifname "fvmb0" counter name "ctbase" drop
		oifname "fvmb0" counter name "ctbase" drop
	}
`
	got = parseForwardChain(netbird)
	if got.Policy != "drop" || !got.Baseline || got.MemberAccept != 0 {
		t.Fatalf("netbird accepts: %+v", got)
	}
}

func TestPickAutoGroupSkipsAll(t *testing.T) {
	allFirst := []nbGroup{{ID: "all-id", Name: "All"}, {ID: "lab-id", Name: "fvpn-unpublished-nodes"}}
	if got := pickAutoGroup(allFirst); got != "lab-id" {
		t.Fatalf("pick: %q", got)
	}
	if got := pickAutoGroup([]nbGroup{{ID: "all-id", Name: "All"}}); got != "" {
		t.Fatalf("all group was selected: %q", got)
	}
	if got := pickAutoGroup(nil); got != "" {
		t.Fatalf("empty: %q", got)
	}
}
