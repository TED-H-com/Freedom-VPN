//go:build netlab

package netlab

import (
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// One live check of the exact self-contained rollback batch, also exercised
// by the full-suite baseline consumers. State{} deliberately knows nothing
// about the candidate. No table flush/delete is part of the baseline batch.
func TestISO01_AT13_F32_UnknownStateBaselineAtomic(t *testing.T) {
	skipLab(t)
	t.Cleanup(func() { applyBaseline(t) })
	m := functionalModel(time.Now().UTC(), 3200)
	baseModel := enforce.Model{Datapath: m.Datapath}
	script, err := enforce.RenderBaseline(baseModel, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	save(t, "baseline.nft", script)
	save(t, "versions.txt", mustExec(t, "nft", "--version")+mustExec(t, "uname", "-r"))
	run := func(batch string, wantOK bool) {
		t.Helper()
		out, stderr, err := dockerExecStdin([]string{"nft", "-f", "-"}, []byte(batch))
		if (err == nil) != wantOK {
			t.Fatalf("nft success=%v want=%v: %s%s", err == nil, wantOK, out, stderr)
		}
		if !wantOK {
			save(t, "atomic-rejection.txt", out+stderr)
		}
	}
	check := func(label string) {
		t.Helper()
		raw := mustExec(t, "nft", "-j", "list", "ruleset")
		if err := enforce.VerifyBaseline([]byte(raw), baseModel); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		state, err := enforce.ParseState([]byte(mustExec(t, "nft", "-j", "list", "table", "inet", "fvpn")))
		if err != nil {
			t.Fatal(err)
		}
		if len(state.Chains) != 4 || len(state.Sets) != 1 || !state.Sets["allowed_v4"] {
			t.Fatalf("%s residual candidate objects: chains=%v sets=%v", label, state.Chains, state.Sets)
		}
		save(t, label+".json", raw)
	}
	// Start with no owned table; never touch another table.
	mustExec(t, "nft", "add", "table", "inet", "fvpn")
	mustExec(t, "nft", "delete", "table", "inet", "fvpn")
	run(script, true)
	check("cold")
	applyAt(t, m)
	requireProbe(t, runProbe(t, []probeCase{{"member-before", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}}))
	// A failing statement at the END must roll back chain deletion, admission
	// flush and set removals together; the old generation must still work.
	run(script+"add rule inet fvpn fv_missing_f32 drop\n", false)
	live := mustExec(t, "nft", "-j", "list", "table", "inet", "fvpn")
	state, err := enforce.ParseState([]byte(live))
	if err != nil || state.Generation != m.Generation {
		t.Fatalf("failed batch partly committed: gen=%d err=%v", state.Generation, err)
	}
	requireProbe(t, runProbe(t, []probeCase{{"member-after-abort", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}}))
	// Add all optional candidate objects and the deadline guard. Their contents
	// must be removed even though the renderer saw State{}.
	run("add set inet fvpn probe_v4 { type ipv4_addr; flags timeout; }\nadd element inet fvpn probe_v4 { 100.64.8.1 timeout 30s }\nadd set inet fvpn probe_dst_v4 { type ipv4_addr . inet_proto . inet_service; flags timeout; }\nadd element inet fvpn probe_dst_v4 { 203.0.113.10 . tcp . 80 timeout 30s }\nadd chain inet fvpn fv_deadline_guard { type filter hook forward priority -151; policy accept; }\nadd rule inet fvpn fv_deadline_guard iifname \"fv-fh\" drop\nadd rule inet fvpn fv_deadline_guard oifname \"fv-fh\" drop\n", true)
	before, err := enforce.ParseState([]byte(mustExec(t, "nft", "-j", "list", "table", "inet", "fvpn")))
	if err != nil {
		t.Fatal(err)
	}
	run(script, true)
	check("warm-guarded")
	after, err := enforce.ParseState([]byte(mustExec(t, "nft", "-j", "list", "table", "inet", "fvpn")))
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range before.Counters {
		if strings.HasPrefix(name, "u_") || strings.HasPrefix(name, "d_") {
			if after.Counters[name] != value {
				t.Fatalf("counter %s reset: %v -> %v", name, value, after.Counters[name])
			}
		}
	}
	requireProbe(t, runProbe(t, []probeCase{{"member-closed", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 500, "", "", "", "", 0}}))
	run(script, true)
	check("repeat")
	t.Log("cold/warm/guarded/repeat baseline verified; failed end statement aborted whole transaction; all six candidate sets cleared/removed; counters retained; member closed")
}
