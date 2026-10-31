//go:build netlab

package netlab

import (
	"os"
	"strings"
	"testing"
	"time"
)

func retainF21(t *testing.T, dir string) {
	t.Helper()
	files := mustExec(t, "find", dir, "-type", "f")
	for _, path := range strings.Fields(files) {
		if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".err") {
			name := strings.ReplaceAll(strings.TrimPrefix(path, dir+"/"), "/", "-")
			save(t, name, readScratch(t, path))
		}
	}
}

func skipF21(t *testing.T) {
	t.Helper()
	skipLab(t)
	if os.Getenv("FVPN_NETLAB_F21") != "1" {
		t.Skip("set FVPN_NETLAB_F21=1 on a fresh topology for follow-up 21 diagnostics")
	}
}

func TestF21_AT02_RollbackSibling(t *testing.T) {
	skipF21(t)
	flushConntrack(t)
	m := deviceModel(time.Now().UTC(), 2113, []idBind{
		{memberA, devA1, "peer-a1", "100.64.1.1"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, generousQoS(), 48*time.Hour)
	writeScratch(t, "/scratch/f21-model.json", mustJSON(m))
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/rollback_measure.py", "measure", "/scratch/f21-model.json")
	retainF21(t, "/scratch/f21-rollback")
	save(t, "run.txt", out+errOut)
	if err != nil {
		t.Fatalf("rollback measurement: %v %s%s", err, out, errOut)
	}
	t.Log(out)
}

// Run last: this explicitly changes a1's fixture underlay and tunnel MTUs.
func TestF21_AT09_IPv6Underlay(t *testing.T) {
	skipF21(t)
	flushConntrack(t)
	applyAt(t, functionalModel(time.Now().UTC(), 21802))
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/ipv6_underlay.py", "measure")
	retainF21(t, "/scratch/f21-ipv6")
	save(t, "run.txt", out+errOut)
	if err != nil {
		t.Fatalf("IPv6 underlay measurement: %v %s%s", err, out, errOut)
	}
	t.Log(out)
}
