package enforce

import (
	"bytes"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"testing"
)

func TestReviewR19_Q3_TCDigestIgnoresCandidateGeneration(t *testing.T) {
	base := ServingEvidence{TC: []TCReadEvidence{{
		Interface: "fv-ih", Attachment: "root",
		Objects: []TCObject{{Kind: "htb", Handle: "1:1", Root: true}},
	}}}
	low := base
	low.retainTC(4, "verify", nil)
	high := ServingEvidence{TC: append([]TCReadEvidence{}, base.TC...)}
	high.retainTC(5, "verify", nil)
	if low.TCSummary.SHA256 == "" || low.TCSummary.SHA256 != high.TCSummary.SHA256 {
		t.Fatalf("digest changed with generation: %s vs %s", low.TCSummary.SHA256, high.TCSummary.SHA256)
	}
	if low.TCSummary.Generation == high.TCSummary.Generation || low.tcReadback == nil || high.tcReadback == nil {
		t.Fatalf("snapshot missing or generations collapsed: %+v %+v", low.TCSummary, high.TCSummary)
	}
	if low.tcReadback.evidence.Sha256 != high.tcReadback.evidence.Sha256 ||
		low.tcReadback.evidence.CandidateGeneration == high.tcReadback.evidence.CandidateGeneration {
		t.Fatalf("D-106 snapshot did not keep one digest: %+v %+v", low.tcReadback.evidence, high.tcReadback.evidence)
	}
	t.Fatalf("TODO(review): retainTC generations %s and %s share sha256 %s. evidence.go hashes reads, objects, truncated and tc, and copies Generation into candidate_generation separately",
		low.TCSummary.Generation, high.TCSummary.Generation, low.TCSummary.SHA256)
}

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
	uploadIdx, establishedIdx, acceptIdx := -1, -1, -1
	var accept string
	for i, rule := range rules {
		switch {
		case strings.Contains(rule, uploadDeny) && strings.Contains(rule, "drop"):
			uploadIdx = i
		case strings.Contains(rule, established) && strings.Contains(rule, "drop"):
			establishedIdx = i
		case strings.Contains(rule, fmt.Sprintf("iifname %q oifname %q", m.Datapath.Uplink, m.Datapath.MemberIngress)) &&
			strings.Contains(rule, " accept") && !strings.Contains(rule, "sport 53") && !strings.Contains(rule, "dport 53"):
			if acceptIdx < 0 {
				acceptIdx = i
				accept = rule
			}
		}
	}
	if uploadIdx < 0 || establishedIdx < 0 || acceptIdx < 0 || establishedIdx > acceptIdx {
		t.Fatalf("rule order upload=%d established=%d accept=%d\n%s", uploadIdx, establishedIdx, acceptIdx, plan.NFT)
	}
	if strings.Contains(accept, "@denied_v4") || strings.Contains(accept, "ip saddr") {
		t.Fatal("download accept grew a remote-source match; the review assumption is stale")
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
	t.Fatalf("TODO(review): download accept follows the established drop and does not match @denied_v4 (%s). Upload deny is an earlier drop. A render against an existing forward chain deletes that chain and flushes allowed_v4. Apply calls TCBatch before NftBatch", accept)
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

func TestReviewR19_Q4_ShorterKernelTimeoutIsNotAnInput(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{Sets: map[string]bool{
		"allowed_v4": true, "denied_v4": true, "host_v4": true, "resolver_v4": true,
	}}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.NFT, "flush set inet fvpn allowed_v4") {
		t.Fatal("render did not flush allowed_v4")
	}
	got := allowedElementTimeout(plan.NFT)
	if got == "" {
		t.Fatalf("allowed element has no timeout:\n%s", plan.NFT)
	}
	stateSrc, err := os.ReadFile("state.go")
	if err != nil {
		t.Fatal(err)
	}
	deadlineSrc, err := os.ReadFile("deadline.go")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stateSrc, []byte("AllowedRemain")) || bytes.Contains(stateSrc, []byte("ElementTimeout")) {
		t.Fatal("State grew a live allowed-element timeout; the review assumption is stale")
	}
	if !bytes.Contains(deadlineSrc, []byte("func elementTimeout(binding Binding, lease MemberLease, now time.Time)")) {
		t.Fatal("elementTimeout no longer ignores a live kernel timeout")
	}
	t.Fatalf("TODO(review): with allowed_v4 already present, Render flushes it and writes timeout %sms from the model. State has no live element lifetime, and elementTimeout takes only the binding, the lease and now. A shorter kernel timeout is not an input", got)
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

func allowedElementTimeout(script string) string {
	const mark = "add element inet fvpn allowed_v4 { "
	i := strings.Index(script, mark)
	if i < 0 {
		return ""
	}
	rest := script[i+len(mark):]
	j := strings.Index(rest, " timeout ")
	k := strings.Index(rest, "ms")
	if j < 0 || k < 0 || k < j {
		return ""
	}
	return strings.TrimSpace(rest[j+len(" timeout ") : k])
}
