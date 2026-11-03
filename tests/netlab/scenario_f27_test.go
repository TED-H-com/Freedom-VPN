//go:build netlab

package netlab

import (
	"fmt"
	"net/netip"
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
