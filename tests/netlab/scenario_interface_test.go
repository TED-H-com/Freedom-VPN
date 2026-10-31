//go:build netlab

package netlab

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The isolation case adds a disposable host interface, then changes capacity.
// Preserve Apply's read-back before any test cleanup can erase the rejection.
func TestISO_F6_InterfaceApply(t *testing.T) {
	skipLab(t)
	applyBaseline(t)
	m := functionalModel(time.Now().UTC(), 14000)
	applyAt(t, m)
	t.Cleanup(func() { _, _, _ = dockerExec("ip", "link", "del", "fvisotest") })
	var evidence strings.Builder
	for cycle := range 10 {
		mustExec(t, "ip", "link", "add", "fvisotest", "type", "dummy")
		mustExec(t, "ip", "addr", "add", "198.51.100.10/32", "dev", "fvisotest")
		mustExec(t, "ip", "link", "set", "fvisotest", "up")
		// Generate host traffic through HTB's default class, as control I/O does.
		mustExec(t, "ping", "-c", "1", "-W", "1", "192.0.2.38")
		mustExec(t, "ip", "link", "del", "fvisotest")
		m.Generation++
		m.QoS.Down.CeilBPS += 1_000_000
		body, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		out, stderr, err := dockerExecStdin([]string{"fvpn-netlab", "apply"}, body)
		fmt.Fprintf(&evidence, "cycle=%d\n%s\n%s\n", cycle+1, out, stderr)
		save(t, "interface-apply.txt", evidence.String())
		if err != nil {
			t.Fatalf("interface Apply: %v %s %s", err, stderr, out)
		}
	}
}
