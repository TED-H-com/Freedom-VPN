//go:build netlab

package agentlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLabF36_AT13_RestartChangedSetsStopsEstablishedFlow(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	original := readScratch(t, labDir+"/agent.json")
	defer writeScratch(t, labDir+"/agent.json", []byte(original))
	var cfg map[string]any
	if err := json.Unmarshal([]byte(original), &cfg); err != nil {
		t.Fatal(err)
	}
	// Establish the tuple during a quiet poll interval, then exercise ordinary
	// cooperative replacement. A TERM inside unrelated periodic Apply can
	// legitimately leave a baseline and would not test retained-set adoption.
	cfg["heartbeat_seconds"], cfg["poll_seconds"], cfg["watchdog_stale_seconds"] = 10, 10, 30
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, labDir+"/agent.json", raw)
	startAgent(t)
	defer stopAgent()
	defer admin(t, "/release", "")
	if !waitProbe(t, "allow", 20*time.Second) {
		t.Fatal("initial forwarding unavailable", debugPlane(t))
	}
	// The helper keeps a single socket/conntrack tuple across the restart,
	// holds renewal through Boot and the first control timeout, then releases
	// it to prove that the first full Apply also blocks that established flow.
	out, errOut, err := dockerExec("python3", "/work/tests/netlab/agent/restart-sets.py")
	if dir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR"); dir != "" {
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		raw, _, readErr := dockerExec("cat", labDir+"/f36-result.json")
		if readErr == nil {
			if e := os.WriteFile(filepath.Join(dir, "f36-result.json"), []byte(raw), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	if err != nil {
		t.Fatalf("restart flow: %v\n%s%s\n%s", err, out, errOut, debugPlane(t))
	}
	fmt.Print(out)
}
