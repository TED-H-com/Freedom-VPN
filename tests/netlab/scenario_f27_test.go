//go:build netlab

package netlab

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Mutate only the installed set, leaving the verified rules, generation,
// counters and QoS untouched. The safety assertion deliberately fails on main.
func TestISO01_F27_LiveSetDrift(t *testing.T) {
	skipLab(t)
	m := functionalModel(time.Now().UTC(), 2700)
	m.DeniedV4 = []netip.Prefix{mustPrefix("192.0.2.0/25"), mustPrefix("198.51.100.0/24")}
	m.ExtraDeniedV4 = nil
	m.HostV4 = []netip.Prefix{mustPrefix("203.0.113.4/32"), mustPrefix("203.0.113.8/32")}
	for _, set := range []string{"denied_v4", "host_v4"} {
		for _, change := range []string{"add", "remove", "replace"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				applyAt(t, m)
				out, stderr, code := runLab(t, m, "verify")
				if code != 0 {
					t.Fatalf("positive verify: %d %s %s", code, out, stderr)
				}
				save(t, "before.json", mustExec(t, "nft", "-j", "list", "ruleset"))
				original := "192.0.2.0/25"
				if set == "host_v4" {
					original = "203.0.113.4"
				}
				var script string
				if change != "add" {
					script += fmt.Sprintf("delete element inet fvpn %s { %s }\n", set, original)
				}
				if change != "remove" {
					script += fmt.Sprintf("add element inet fvpn %s { 203.0.113.99 }\n", set)
				}
				if out, errout, err := dockerExecStdin([]string{"nft", "-f", "-"}, []byte(script)); err != nil {
					t.Fatalf("mutation: %v %s %s", err, out, errout)
				}
				save(t, "mutation.nft", script)
				save(t, "after.json", mustExec(t, "nft", "-j", "list", "ruleset"))
				out, stderr, code = runLab(t, m, "verify")
				save(t, "verify.json", out)
				save(t, "verdict.txt", fmt.Sprintf("exit=%d\n%s", code, stderr))
				if code != 3 || !strings.Contains(stderr, string(enforce.ReasonIsolation)) {
					t.Fatalf("live %s %s accepted: exit=%d %s", set, change, code, stderr)
				}
			})
		}
	}
}

func TestISO01_F27_RenderedCoverage(t *testing.T) {
	skipLab(t)
	for _, contents := range []string{"spec", "overlap"} {
		for _, mode := range []string{"serving", "serving_probe", "baseline_probe"} {
			t.Run(contents+"/"+mode, func(t *testing.T) {
				m := functionalModel(time.Now().UTC(), 2701)
				m.ExtraDeniedV4 = nil
				if contents == "spec" {
					m.DeniedV4 = nil
				} else {
					m.ExtraDeniedV4 = []netip.Prefix{mustPrefix("203.0.113.128/26"), mustPrefix("203.0.113.192/26"), mustPrefix("203.0.113.160/27")}
				}
				m.HostV4 = []netip.Prefix{mustPrefix("192.0.2.32/30"), mustPrefix("192.0.2.36/32")}
				if mode != "serving" {
					m.Probe = &enforce.ProbeException{PeerID: "peer-probe", SourceV4: mustPrefix("100.64.8.1/32"), ResolverDNS: true, ExpiresAt: m.Now.Add(60 * time.Second)}
				}
				if mode == "baseline_probe" {
					applyBaselineModel(t, m)
				} else {
					applyAt(t, m)
				}
				raw := mustExec(t, "nft", "-j", "list", "ruleset")
				save(t, "installed.json", raw)
				save(t, "model.json", string(mustJSON(m)))
				if mode == "baseline_probe" {
					if err := enforce.VerifyBaseline([]byte(raw), m); err != nil {
						t.Fatal(err)
					}
				} else {
					out, stderr, code := runLab(t, m, "verify")
					save(t, "verify.json", out)
					if code != 0 {
						t.Fatalf("verification: %d %s", code, stderr)
					}
				}
				// Transition from each complete form to the bare baseline. Historical
				// denial sets must disappear with the old rules in one transaction.
				applyBaseline(t)
				raw = mustExec(t, "nft", "-j", "list", "ruleset")
				save(t, "bare.json", raw)
				if err := enforce.VerifyBaseline([]byte(raw), enforce.Model{Datapath: m.Datapath}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

// Opt-in fixed-size timing population, separate from reload tracing. The
// existing CLI exposes VerifyServing's monotonic command/total brackets.
func TestISO01_F27_VerifyCost(t *testing.T) {
	skipLab(t)
	if os.Getenv("FVPN_NETLAB_F27_COST") != "1" {
		t.Skip("set FVPN_NETLAB_F27_COST=1 for the predeclared comparison")
	}
	for _, size := range []string{"spec", "expanded"} {
		t.Run(size, func(t *testing.T) {
			m := functionalModel(time.Now().UTC(), 2710)
			m.DeniedV4 = nil
			m.ExtraDeniedV4 = nil
			m.HostV4 = nil
			if size == "expanded" {
				m.DeniedV4 = enforce.Without(enforce.SpecDeniedV4(), mustPrefix("203.0.113.0/24"))
				for i := 0; i < 128; i++ {
					m.ExtraDeniedV4 = append(m.ExtraDeniedV4, mustPrefix(fmt.Sprintf("203.0.113.%d/32", 2*i)))
					m.HostV4 = append(m.HostV4, mustPrefix(fmt.Sprintf("198.51.100.%d/32", 2*i)))
				}
			}
			applyAt(t, m)
			save(t, "model.json", string(mustJSON(m)))
			save(t, "installed.json", mustExec(t, "nft", "-j", "list", "ruleset"))
			var rows []any
			for i := 0; i < 100; i++ {
				out, stderr, code := runLab(t, m, "verify")
				var body any
				if err := json.Unmarshal([]byte(out), &body); err != nil {
					body = out
				}
				rows = append(rows, map[string]any{"index": i, "exit": code, "stderr": stderr, "result": body})
				if code != 0 {
					t.Errorf("verify %d: exit=%d %s", i, code, stderr)
				}
			}
			save(t, "samples.json", string(mustJSON(rows)))
		})
	}
}
