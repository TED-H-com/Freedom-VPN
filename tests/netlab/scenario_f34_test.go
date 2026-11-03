//go:build netlab

package netlab

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// D-146/D-147: fixed IPv4-underlay kernel proof, not native AT09 acceptance.
func TestAT09_AT13_D146_D147(t *testing.T) {
	skipLab(t)
	before := os.Getenv("FVPN_NETLAB_F34_BEFORE") == "1"
	for _, mode := range []string{"receive", "router"} {
		t.Run(mode, func(t *testing.T) {
			flushConntrack(t)
			m := functionalModel(time.Now().UTC(), 3400)
			m.HostV4 = append(m.HostV4, mustPrefix("192.168.34.1/32"))
			applyAt(t, m)
			save(t, "model.json", string(mustJSON(m))+"\n")
			// Read the workspace copy so a focused baseline run needs no image rebuild.
			out, errOut, err := dockerExec("env", "PYTHONPATH=/opt/netlab/scripts", "python3", "-B", "/work/tests/netlab/scripts/f34_paths.py", mode)
			save(t, "measurement.txt", out+errOut)
			files := mustExec(t, "find", "/scratch/f34", "-maxdepth", "1", "-type", "f", "-printf", "%f\n")
			for _, name := range strings.Fields(files) {
				if strings.HasPrefix(name, mode+"-") && (strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".err")) {
					save(t, name, readScratch(t, "/scratch/f34/"+name))
				}
			}
			if err != nil {
				t.Fatalf("measurement: %v\n%s%s", err, out, errOut)
			}

			args := []string{"python3", "-B", "/work/tests/netlab/scripts/f34_check.py", "/scratch/f34", mode}
			if before {
				args = append(args, "before")
			}
			checked, checkErr, checkFailure := dockerExec(args...)
			save(t, "checks.txt", checked+checkErr)
			if checkFailure != nil {
				t.Fatalf("independent header/counter checks: %v\n%s%s", checkFailure, checked, checkErr)
			}
			directions := []string{"upload", "download"}
			if mode == "router" {
				directions = directions[:1]
			}
			for _, dir := range directions {
				var row struct {
					Result struct {
						Stdout string `json:"stdout"`
					} `json:"result"`
				}
				if err := json.Unmarshal([]byte(readScratch(t, "/scratch/f34/"+mode+"-"+dir+".json")), &row); err != nil {
					t.Fatal(err)
				}
				var result struct {
					Bytes int `json:"bytes"`
					PMTU  int `json:"pmtu"`
				}
				if err := json.Unmarshal([]byte(row.Result.Stdout), &result); err != nil {
					t.Fatal(err)
				}
				if !before && result.Bytes != 64064 {
					t.Errorf("%s %s: got %d/64064", mode, dir, result.Bytes)
				}
				if before && mode == "receive" && dir == "download" && result.Bytes == 64064 {
					t.Error("baseline unexpectedly completed black-hole download")
				}
				if before && mode == "router" && dir == "upload" && result.Bytes == 64064 {
					t.Error("baseline unexpectedly completed private-router PMTU upload")
				}
				if !before && mode == "router" && dir == "upload" && result.PMTU != 1000 {
					t.Errorf("PMTU=%d want 1000", result.PMTU)
				}
			}
		})
	}
}
