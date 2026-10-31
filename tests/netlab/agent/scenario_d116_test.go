//go:build netlab

package agentlab

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func saveD116(t *testing.T, name, body string) {
	t.Helper()
	if strings.Contains(body, labToken) || strings.Contains(body, labSetupKey) || strings.Contains(body, "PRIVATE KEY") {
		t.Fatal("secret in evidence", name)
	}
	dir := os.Getenv("FVPN_NETLAB_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join("../../..", dir)
	}
	dir = filepath.Join(dir, t.Name())
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
func servingObservation(t *testing.T, want uint64) string {
	t.Helper()
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		raw, ok := adminTryGet(t, "/observation")
		var rep api.ObservationReport
		if ok && json.Unmarshal([]byte(raw), &rep) == nil {
			gen, err := rep.AppliedConfigGeneration.Get()
			if err == nil && uint64(gen) == want && rep.MemberForwardingEnabled && rep.SafetyStatus == api.SafetyStatusPassed {
				return raw
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("generation %d not serving: %s", want, tailLog())
	return ""
}

// D-116 / AT10: normal enrollment and real Engine, TEST ONLY fake central
// signatures/NetBird status plus the existing WireGuard/veth lab data plane.
func TestD116_AT10_InstallerMarkerEnrollment(t *testing.T) {
	skipLab(t)
	if os.Getenv("FVPN_D116_MARKER") != "1" {
		t.Skip("set FVPN_D116_MARKER=1 before TestMain enrollment")
	}
	stopAgent()
	stopWatch()
	flushConntrack(t)
	saveD116(t, "before-enrollment.json", readScratch(t, labDir+"/before-enrollment.json"))
	saveD116(t, "before-enrollment.txt", readScratch(t, labDir+"/before-enrollment.txt"))
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	script := "echo $$ > /scratch/trace-agent.pid; exec fvpn-netlab agent-trace -config " + labDir + "/agent.json >" + labDir + "/agent.log 2>&1"
	if err := dockerDetach("sh", "-c", script); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stopProc("fvpn-netlab", "/scratch/trace-agent.pid")
		assertNoSecrets(t)
		saveD116(t, "agent.txt", readScratch(t, labDir+"/agent.log"))
	})
	saveD116(t, "marker-observation.json", servingObservation(t, 1))
	if !waitProbe(t, "allow", 10*time.Second) {
		t.Fatal("marker serving packet failed", debugPlane(t))
	}
	assertMarkerRuleset(t, "marker-serving.json", true)
	// Compare the complete verifier/serving outcome with no foreign table.
	mustExec(t, "nft", "delete", "table", "inet", "fvpn_install")
	admin(t, "/bump", "")
	saveD116(t, "baseline-observation.json", servingObservation(t, 2))
	if !waitProbe(t, "allow", 10*time.Second) {
		t.Fatal("unmarked baseline packet failed")
	}
	assertMarkerRuleset(t, "baseline-serving.json", false)
	// Restore the unhooked marker for 20 measured real-engine reload cycles.
	mustExec(t, "nft", "add", "table", "inet", "fvpn_install")
	mustExec(t, "nft", "add", "chain", "inet", "fvpn_install", "owner", `{ comment "managed-by=fvpn-node-install"; }`)
	mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "start", "/scratch/agent-at10")
	if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/agent-stream.log"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Complete the last recorded read before ending the independent trace.
		mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "finish-stream", "/scratch/agent-stream.log")
		mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "stop", "/scratch/agent-at10")
		stopProc("unused-stream-name", "/scratch/agent-stream.log.pid")
		for _, name := range []string{"agent-at10-origin.txt", "agent-at10-origin.err", "agent-at10-node.txt", "agent-at10-node.err", "agent-at10-responder.txt", "agent-at10-responder.err", "agent-stream.log.jsonl", "agent-stream.log"} {
			saveD116(t, strings.ReplaceAll(name, ".log", ".txt"), readScratch(t, "/scratch/"+name))
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for !scratchExists("/scratch/agent-stream.log") || !strings.Contains(readScratch(t, "/scratch/agent-stream.log"), " ok") {
		if time.Now().After(deadline) {
			t.Fatal("stream not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	for gen := uint64(3); gen <= 22; gen++ {
		admin(t, "/bump", "")
		servingObservation(t, gen)
	}
	saveD116(t, "final-observation.json", servingObservation(t, 22))
	assertMarkerRuleset(t, "final-serving.json", true)
	// This is a diagnostic campaign, not a replacement AT10 continuity gate.
	// The original 100-reload gate still fails on every stream timeout.
	t.Log("D-116 marker before enrollment -> serving=1; unmarked baseline serving=1; owned table comment retained; 20 real-agent reloads complete")
}
func assertMarkerRuleset(t *testing.T, name string, marker bool) {
	t.Helper()
	raw := mustExec(t, "nft", "-j", "list", "ruleset")
	saveD116(t, name, raw)
	rs, err := enforce.ParseRuleset([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Chains) != 4 {
		t.Fatalf("owned chains=%d", len(rs.Chains))
	}
	count := 0
	for _, c := range rs.Foreign {
		if c.Table == "fvpn_install" {
			count++
			if c.Hook != "" || c.Name != "owner" {
				t.Fatalf("foreign base chain: %+v", c)
			}
		}
	}
	if (count == 1) != marker {
		t.Fatalf("marker=%v count=%d", marker, count)
	}
	// Read text too: nft 1.0.9 JSON may omit table metadata.
	text := mustExec(t, "nft", "list", "table", "inet", "fvpn")
	saveD116(t, strings.TrimSuffix(name, ".json")+".txt", text)
	if !strings.Contains(text, `comment "managed-by=fvpn-node-install"`) {
		t.Fatal("owned table comment lost after Apply")
	}
}
