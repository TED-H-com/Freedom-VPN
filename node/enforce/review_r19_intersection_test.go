package enforce

import (
	"bytes"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"testing"
)

func TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload(t *testing.T) {
	m := smallModel()
	m.ExtraDeniedV4 = append(m.ExtraDeniedV4, netip.MustParsePrefix("203.0.113.10/32"))
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	rules := forwardRules(plan.NFT)
	uploadDeny := fmt.Sprintf("iifname %q oifname %q ip daddr @denied_v4", m.Datapath.MemberIngress, m.Datapath.Uplink)
	established := fmt.Sprintf("iifname %q oifname %q ct state != { established, related }", m.Datapath.Uplink, m.Datapath.MemberIngress)
	uploadIdx, establishedIdx, acceptIdx, denyDownIdx := -1, -1, -1, -1
	for i, rule := range rules {
		switch {
		case strings.Contains(rule, uploadDeny) && strings.Contains(rule, "drop"):
			uploadIdx = i
		case strings.Contains(rule, "ip saddr @denied_v4") && strings.Contains(rule, "drop"):
			denyDownIdx = i
		case strings.Contains(rule, established) && strings.Contains(rule, "drop"):
			establishedIdx = i
		case strings.Contains(rule, fmt.Sprintf("iifname %q oifname %q", m.Datapath.Uplink, m.Datapath.MemberIngress)) &&
			strings.Contains(rule, " accept") && !strings.Contains(rule, "sport 53") && !strings.Contains(rule, "dport 53"):
			if acceptIdx < 0 {
				acceptIdx = i
			}
		}
	}
	if uploadIdx < 0 || establishedIdx < 0 || acceptIdx < 0 || establishedIdx > acceptIdx {
		t.Fatalf("rule order upload=%d established=%d accept=%d\n%s", uploadIdx, establishedIdx, acceptIdx, plan.NFT)
	}
	if denyDownIdx < 0 || denyDownIdx > acceptIdx {
		t.Fatalf("established download bypasses destination denial: deny=%d accept=%d", denyDownIdx, acceptIdx)
	}
	if !strings.Contains(plan.TC, "class replace") || !strings.Contains(plan.TC, " htb rate ") || !strings.Contains(plan.TC, " ceil ") {
		t.Fatalf("tc script lost the rate and ceil:\n%s", plan.TC)
	}
	again, err := Render(m, State{
		Chains: map[string]bool{"forward": true},
		Sets:   map[string]bool{"allowed_v4": true},
	}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again.NFT, "delete chain inet fvpn forward") || !strings.Contains(again.NFT, "flush set inet fvpn allowed_v4") {
		t.Fatal("a second render no longer deletes the forward chain or flushes allowed_v4")
	}
	src, err := os.ReadFile("apply.go")
	if err != nil {
		t.Fatal(err)
	}
	tc := bytes.Index(src, []byte("ex.TCBatch("))
	nft := bytes.Index(src, []byte("ex.NftBatch("))
	if tc < 0 || nft < 0 || tc > nft {
		t.Fatalf("Apply order changed: TCBatch=%d NftBatch=%d", tc, nft)
	}
	t.Log("current destination denial precedes the established download accept")

}

func TestReviewR19_Q4_ExistingCounterIsNotRecreated(t *testing.T) {
	m := smallModel()
	first, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	const mark = "add counter inet fvpn "
	i := strings.Index(first.NFT, mark)
	if i < 0 {
		t.Fatalf("no counter:\n%s", first.NFT)
	}
	rest := first.NFT[i+len(mark):]
	end := strings.IndexAny(rest, " \n")
	if end <= 0 {
		t.Fatalf("counter name: %q", rest)
	}
	name := rest[:end]
	second, err := Render(m, State{Counters: map[string]CounterValue{name: {}}}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(second.NFT, mark+name) {
		t.Fatalf("render recreated counter %s", name)
	}
}

func forwardRules(script string) []string {
	var out []string
	for _, line := range strings.Split(script, "\n") {
		if strings.Contains(line, "add rule inet fvpn forward ") {
			out = append(out, line)
		}
	}
	return out
}
