//go:build netlab

package netlab

import (
	"os"
	"testing"
	"time"
)

func TestF22_D133_AT02_CommandLifecycle(t *testing.T) {
	skipLab(t)
	if os.Getenv("FVPN_NETLAB_F22") != "1" {
		t.Skip("set FVPN_NETLAB_F22=1 on a fresh topology")
	}
	flushConntrack(t)
	m := deviceModel(time.Now().UTC(), 2213, []idBind{
		{memberA, devA1, "peer-a1", "100.64.1.1"},
		{memberB, devB1, "peer-b1", "100.64.2.1"},
	}, generousQoS(), 48*time.Hour)
	writeScratch(t, "/scratch/f22-model.json", mustJSON(m))
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/f22_rollback.py", "measure", "/scratch/f22-model.json")
	retainF21(t, "/scratch/f22-rollback")
	save(t, "run.txt", out+errOut)
	if err != nil {
		t.Fatalf("command lifecycle rollback fixture: %v %s%s", err, out, errOut)
	}
	t.Log(out)
}
