//go:build netlab

package agentlab

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Assemble the real agent, kernel meter, TLS lease issuer and independent
// packet collector. No central projector rows or signed generation are edited.
func TestLabD087_D104_GrantRenewalExpiryCorrelation(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	original := readScratch(t, labDir+"/agent.json")
	var cfg map[string]any
	if err := json.Unmarshal([]byte(original), &cfg); err != nil {
		t.Fatal(err)
	}
	cfg["renew_lead_seconds"], cfg["observation_seconds"] = 30, 3600
	cfg["test_timing_path"] = labDir + "/operations.jsonl"
	// Match assembled isolation liveness settings; retain watchdog reasons.
	cfg["watchdog_stale_seconds"], cfg["watchdog_interval_seconds"] = 30, 1
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, labDir+"/agent.json", raw)
	// Preserve every observation, including the short provisional transition.
	allReports := func() []api.ObservationReport {
		t.Helper()
		raw, ok := adminTryGet(t, "/observations")
		var reports []api.ObservationReport
		if !ok || json.Unmarshal([]byte(raw), &reports) != nil {
			t.Fatal("receiver history unavailable")
		}
		return reports
	}
	before := len(allReports())
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	admin(t, "/lease-window", `{"seconds":60}`)
	admin(t, "/member-grants", `{"ready":false}`)
	t.Cleanup(func() {
		stopAgent()
		admin(t, "/release", "")
		admin(t, "/member-grants", `{"ready":true}`)
		admin(t, "/lease-window", `{"seconds":3600}`)
		writeScratch(t, labDir+"/agent.json", []byte(original))
	})
	startAgent(t)
	var provisional api.ObservationReport
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		for _, rep := range allReports()[before:] {
			if qualifyingProvisional(rep) {
				provisional = rep
				break
			}
		}
		if qualifyingProvisional(provisional) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !qualifyingProvisional(provisional) {
		t.Fatal("no qualifying provisional observation", debugPlane(t))
	}
	gen, err := provisional.AppliedConfigGeneration.Get()
	if err != nil {
		t.Fatal(err)
	}
	if !waitProbe(t, "deny", 3*time.Second) {
		t.Fatal("provisional grant admitted traffic")
	}
	readyAt := time.Now()
	admin(t, "/member-grants", `{"ready":true}`)
	if !waitProbe(t, "allow", 15*time.Second) {
		t.Fatal("fixed generation did not resume counted forwarding", debugPlane(t))
	}
	counterBefore := nftUsageBytes(t)
	sendUDPBurst(t)
	counterAfter := nftUsageBytes(t)
	if counterAfter <= counterBefore {
		t.Fatal("missing counted forwarding after grant")
	}
	var granted api.ObservationReport
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, rep := range allReports()[before:] {
			if rep.MemberForwardingEnabled && rep.ObservedAt.After(provisional.ObservedAt) {
				granted = rep
				break
			}
		}
		if granted.MemberForwardingEnabled {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	newGen, err := granted.AppliedConfigGeneration.Get()
	if err != nil || newGen != gen || !granted.MemberForwardingEnabled || granted.ReportId == provisional.ReportId {
		t.Fatal("fixed generation transition lost", granted)
	}
	checkCleanEvidence(t, granted)
	t.Logf("D087 qualifying report=%s generation=%s; grant-ready to counted forwarding=%s counters=%d->%d; observation interval=3600s", provisional.ReportId, gen, time.Since(readyAt), counterBefore, counterAfter)

	startWatch(t)
	defer stopWatch()
	// The foreground collector includes all ordinary one-second guarded
	// Applies and at least two natural renewals; 80ms deadlines never use D-120.
	mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "trace", "renewal", "0", labDir+"/renewal.json")
	admin(t, "/hold", "")
	// Read the actual accepted request anchor and signed whole-node deadline.
	d := mustExec(t, "python3", "-c", `import sqlite3,json
c=sqlite3.connect('file:/scratch/w2/state/agent.db?mode=ro',uri=True)
r=c.execute('select envelope,mono_sent_ns from lease').fetchone()
from datetime import datetime
l=json.loads(r[0])['signed']
a=datetime.fromisoformat(l['issued_at'].replace('Z','+00:00')); b=datetime.fromisoformat(l['expires_at'].replace('Z','+00:00'))
print(int(r[1])+round((b-a).total_seconds()*1000000000))`)
	// Passing through the shell is unnecessary; trim the printed newline.
	var mono int64
	if _, err := fmt.Sscan(d, &mono); err != nil {
		t.Fatal(err)
	}
	mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "trace", "expiry_restart", fmt.Sprint(mono), labDir+"/expiry.json")
	stopAgent()
	outDir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR")
	if outDir == "" {
		outDir = filepath.Join("..", "..", "..", ".agent", "f27-evidence")
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		t.Fatal(err)
	}
	outDir = filepath.Join(outDir, "assembled")
	if err := os.MkdirAll(outDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"operations.jsonl", "renewal.json", "expiry.json", "expiry.json.restart.json", "agent.log", "watchdog.log"} {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(readScratch(t, labDir+"/"+name)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := json.MarshalIndent(allReports()[before:], "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "observations.json"), reports, 0600); err != nil {
		t.Fatal(err)
	}
	summary, err := exec.Command("python3", "timing-analysis.py", outDir).CombinedOutput()
	t.Logf("D104 producer correlation:\n%s", summary)
	if err != nil {
		t.Fatal("correlation evidence failed", err)
	}
}

func qualifyingProvisional(rep api.ObservationReport) bool {
	if rep.MemberForwardingEnabled || rep.ProvisioningState != api.ProvisioningStateReady || rep.SafetyStatus != api.SafetyStatusPassed || len(rep.ForwardingBlockReasons) != 1 || rep.ForwardingBlockReasons[0] != api.ForwardingBlockReasonAwaitingMemberGrants || len(rep.Errors) != 1 || rep.Errors[0].Stage != "lease" || rep.Errors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS {
		return false
	}
	found := 0
	for _, c := range rep.Checks {
		switch c.Name {
		case api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding:
			code, _ := c.ErrorCode.Get()
			if c.Status != api.HealthCheckStatusFailed || code != api.ErrorCodeAWAITINGMEMBERGRANTS {
				return false
			}
			found++
		}
	}
	return found == 3
}
