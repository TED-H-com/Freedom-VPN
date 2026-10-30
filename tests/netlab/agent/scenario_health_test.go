//go:build netlab

package agentlab

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestLab08_HealthAttestationUnpublished simulates the updater: it writes a
// 0600 challenge for this node and waits for the running agent to answer.
// Publication stays unpublished, so member forwarding stays deny.
// IssueHealthChallenge and ValidateHealthAnswer call healthattest.Issue and
// healthattest.Validate. This package is outside node/ and cannot import
// node/internal/healthattest.
func TestLab08_HealthAttestationUnpublished(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/publication", `{"intent":"unpublished"}`)
	defer func() {
		_, _, _ = dockerExec("rm", "-f", labDir+"/journal/health-challenge.json", labDir+"/journal/health-attest.json")
		admin(t, "/publication", `{"intent":"published"}`)
	}()
	defer stopAgent()

	nodeID, incID := labIdentity(t)
	bootID := strings.ToLower(strings.TrimSpace(mustExec(t, "cat", "/proc/sys/kernel/random/boot_id")))
	const (
		nodeAgent   = "0.0.0-test"
		netbird     = "0.76.2"
		nodeUpdater = "0.0.0-test"
		enforcement = "0.0.0-test"
	)
	// Issued slightly before this clock so the agent's wall time is strictly
	// after issued_at and still inside the 45s updater TTL.
	body, err := agent.IssueHealthChallenge(
		uuid.Must(uuid.NewV7()).String(),
		"019a0000-0000-7000-8000-0000000000c1",
		nodeID, incID, bootID,
		time.Now().UTC().Add(-5*time.Second),
		45*time.Second,
		nodeAgent, netbird, nodeUpdater, enforcement,
	)
	if err != nil {
		t.Fatal(err)
	}
	var issued struct {
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal(body, &issued); err != nil || issued.Nonce == "" {
		t.Fatal(err)
	}
	challengePath := labDir + "/journal/health-challenge.json"
	attestPath := labDir + "/journal/health-attest.json"
	writeScratch(t, challengePath, body)
	mustExec(t, "chmod", "600", challengePath)
	if mode := strings.TrimSpace(mustExec(t, "stat", "-c", "%a", challengePath)); mode != "600" {
		t.Fatalf("challenge mode %s", mode)
	}

	startAgent(t)
	deadline := time.Now().Add(20 * time.Second)
	var raw string
	var att labAttestation
	ok := false
	for time.Now().Before(deadline) {
		out, _, err := dockerExec("cat", attestPath)
		if err == nil && json.Unmarshal([]byte(out), &att) == nil && att.Nonce == issued.Nonce {
			raw = out
			ok = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !ok {
		t.Fatalf("agent did not answer the challenge:\n%s", tailLog())
	}
	if mode := strings.TrimSpace(mustExec(t, "stat", "-c", "%a", attestPath)); mode != "600" {
		t.Fatalf("attestation mode %s", mode)
	}
	reason, err := agent.ValidateHealthAnswer(body, []byte(raw), time.Now().UTC(), nodeAgent, netbird, nodeUpdater, enforcement)
	if err != nil || reason != "" {
		t.Fatalf("validate %s %v body %s", reason, err, raw)
	}
	if att.Published != api.PublicationIntentUnpublished {
		t.Fatalf("published %s", att.Published)
	}
	if att.AgentVersion != nodeAgent || att.NetbirdVersion != netbird {
		t.Fatalf("versions agent %q netbird %q", att.AgentVersion, att.NetbirdVersion)
	}
	if att.Checks.InternetOnly.Status != api.HealthCheckStatusPassed ||
		att.Checks.DNSPolicy.Status != api.HealthCheckStatusPassed ||
		att.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed {
		t.Fatalf("checks %+v", att.Checks)
	}
	if !waitProbe(t, "deny", 12*time.Second) {
		t.Fatalf("unpublished attestation opened member forwarding:\n%s", debugPlane(t))
	}
}

type labCheck struct {
	Status api.HealthCheckStatus `json:"status"`
}

type labAttestation struct {
	Nonce          string                `json:"nonce"`
	AgentVersion   string                `json:"agent_version"`
	NetbirdVersion string                `json:"netbird_version"`
	Published      api.PublicationIntent `json:"published"`
	Checks         struct {
		InternetOnly labCheck `json:"internet_only"`
		DNSPolicy    labCheck `json:"dns_policy"`
		TunnelProbe  labCheck `json:"tunnel_probe"`
	} `json:"checks"`
}

func labIdentity(t *testing.T) (string, string) {
	t.Helper()
	script := `import sqlite3
con = sqlite3.connect("/scratch/w2/state/agent.db")
row = con.execute("select node_id, node_incarnation_id from identity where id = 1").fetchone()
if row is None:
    raise SystemExit("no identity")
print(row[0])
print(row[1])
`
	writeScratch(t, labDir+"/identity_ids.py", []byte(script))
	out := strings.TrimSpace(mustExec(t, "python3", labDir+"/identity_ids.py"))
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || lines[0] == "" || lines[1] == "" {
		t.Fatalf("identity %q", out)
	}
	return strings.ToLower(lines[0]), strings.ToLower(lines[1])
}
