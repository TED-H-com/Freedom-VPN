//go:build netlab

package netlab

import (
	"os"
	"strings"
	"testing"
	"time"
)

// This is a measurement, not AT09 native-platform acceptance. The production
// IPv6 block stays in place; failed IPv6 transfers are retained as limitations.
func TestAT09_F802_MTU1280Measurement(t *testing.T) {
	skipLab(t)
	if os.Getenv("FVPN_NETLAB_MTU") != "1" {
		t.Skip("F8-02: set FVPN_NETLAB_MTU=1 on a fresh topology for the MTU measurement")
	}
	flushConntrack(t)
	m := functionalModel(time.Now().UTC(), 802)
	applyAt(t, m)
	save(t, "model.json", string(mustJSON(m))+"\n")
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/mtu_measure.py", "measure")
	save(t, "measurement.txt", out+errOut)
	files := mustExec(t, "find", "/scratch/f802", "-maxdepth", "1", "-type", "f", "-printf", "%f\n")
	for _, name := range strings.Fields(files) {
		if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".err") {
			save(t, name, readScratch(t, "/scratch/f802/"+name))
		}
	}
	if err != nil {
		t.Fatalf("F8-02 measurement incomplete: %v\n%s%s", err, out, errOut)
	}
	t.Log("F8-02 measurement complete; transfer outcomes and IPv6 limitations are in the retained evidence\n" + out)
}
