//go:build netlab

package agentlab

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestLabUsageLedger runs the agent, the watchdog and fake control together.
// It checks that admission counters become usage events, that a stop takes a
// final sample, that the upload ACK is contiguous, and that a lower watermark
// is refused until a new ledger epoch is signed.
func TestLabUsageLedger(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	admin(t, "/usage/unavailable", `{"n":0}`)
	admin(t, "/usage/checkpoint", `{"mode":"live","highest":0}`)
	defer func() {
		admin(t, "/usage/unavailable", `{"n":0}`)
		admin(t, "/usage/checkpoint", `{"mode":"live","highest":0}`)
		stopAgent()
		stopWatch()
	}()

	startAgent(t)
	if !waitProbe(t, "allow", 25*time.Second) {
		t.Fatalf("agent did not forward:\n%s", debugPlane(t))
	}
	startWatch(t)
	if !waitLog(t, 0, "meter epoch opened", 15*time.Second) {
		t.Fatalf("meter did not open:\n%s", tailLog())
	}
	if !waitLog(t, 0, "usage checkpoint applied", 20*time.Second) {
		t.Fatalf("startup did not apply a usage checkpoint:\n%s", tailLog())
	}

	before := nftUsageBytes(t)
	sendUDPBurst(t)
	time.Sleep(400 * time.Millisecond)
	moved := nftUsageBytes(t)
	if moved <= before {
		t.Fatalf("udp burst did not move admission counters (%d -> %d)\n%s", before, moved, debugPlane(t))
	}

	stopAgentGraceful(t)
	snap := readUsage(t)
	nftAfter := nftUsageBytes(t)
	if snap.Bytes == 0 || snap.Bytes != nftAfter {
		t.Fatalf("usage bytes %d, nft admission bytes %d, snap %+v", snap.Bytes, nftAfter, snap)
	}
	if snap.CloseReason != "final_sample" && !snap.hasClose("final_sample") {
		t.Fatalf("stop did not take a final sample: %+v", snap)
	}
	fmt.Printf("SUMMARY usage counters=%d final-sample=1\n", snap.Bytes)

	from := logLen(t)
	startAgent(t)
	if !waitLog(t, from, "usage batch acknowledged", 40*time.Second) {
		t.Fatalf("usage batch was not acknowledged:\n%s", tailLog())
	}
	if !waitLog(t, from, "after_final_sample", 20*time.Second) {
		t.Fatalf("clean hand-off did not open after_final_sample:\n%s", tailLog())
	}
	stopAgentGraceful(t)
	acked := readUsage(t)
	if acked.Bytes != snap.Bytes {
		t.Fatalf("clean hand-off recounted bytes %d -> %d", snap.Bytes, acked.Bytes)
	}
	if acked.MaxSeq == 0 || acked.Acked != acked.MaxSeq || acked.Outbox != 0 {
		t.Fatalf("contiguous ack missing: %+v", acked)
	}
	fmt.Printf("SUMMARY usage acked_through=%d events=%d\n", acked.Acked, acked.MaxSeq)

	admin(t, "/usage/checkpoint", `{"mode":"same","highest":0}`)
	from = logLen(t)
	startAgent(t)
	if !waitLog(t, from, "ACK regression in the same ledger epoch", 25*time.Second) {
		t.Fatalf("same-epoch lower watermark was not refused:\n%s", tailLog())
	}
	stopAgentGraceful(t)
	refused := readUsage(t)
	if refused.Acked != acked.Acked || refused.Ledger != acked.Ledger || len(refused.Ledgers) != len(acked.Ledgers) {
		t.Fatalf("refused watermark changed the ledger: before %+v after %+v", acked, refused)
	}
	fmt.Println("SUMMARY usage watermark-refused=1")

	admin(t, "/usage/unavailable", `{"n":1000}`)
	admin(t, "/usage/checkpoint", `{"mode":"new","highest":0}`)
	from = logLen(t)
	startAgent(t)
	if !waitLog(t, from, "usage checkpoint applied", 25*time.Second) {
		t.Fatalf("new epoch was not applied:\n%s", tailLog())
	}
	stopAgentGraceful(t)
	restored := readUsage(t)
	if restored.Acked != 0 || restored.Ledger == acked.Ledger || restored.Outbox == 0 {
		t.Fatalf("new epoch did not lower the watermark: %+v", restored)
	}
	fmt.Printf("SUMMARY usage watermark-new-epoch=%s outbox=%d\n", restored.Ledger, restored.Outbox)
	assertNoSecrets(t)
}

func stopAgentGraceful(t *testing.T) {
	t.Helper()
	script := `pidfile=/scratch/w2/agent.pid
if [ -f "$pidfile" ]; then
  pid=$(cat "$pidfile")
  kill -TERM "$pid" 2>/dev/null || true
  i=0
  while [ "$i" -lt 60 ]; do
    if ! kill -0 "$pid" 2>/dev/null; then
      rm -f "$pidfile"
      exit 0
    fi
    i=$((i+1))
    sleep 0.1
  done
  kill -KILL "$pid" 2>/dev/null || true
  sleep 0.2
  rm -f "$pidfile"
fi
pkill -x fvpn-agent 2>/dev/null || true
`
	_, _, _ = dockerExec("sh", "-c", script)
}

func sendUDPBurst(t *testing.T) {
	t.Helper()
	writeScratch(t, labDir+"/udpburst.py", []byte(`import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("100.64.1.1", 0))
pkt = b"u" * 200
for _ in range(40):
    s.sendto(pkt, ("203.0.113.10", 9))
s.close()
`))
	mustExec(t, "ip", "netns", "exec", "fv-a1", "python3", labDir+"/udpburst.py")
}

func logLen(t *testing.T) int {
	t.Helper()
	if !scratchExists(labDir + "/agent.log") {
		return 0
	}
	return len(readScratch(t, labDir+"/agent.log"))
}

func waitLog(t *testing.T, from int, needle string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !scratchExists(labDir + "/agent.log") {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		body := readScratch(t, labDir+"/agent.log")
		if from > len(body) {
			from = 0
		}
		if strings.Contains(body[from:], needle) {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

func nftUsageBytes(t *testing.T) int {
	t.Helper()
	writeScratch(t, labDir+"/nft_usage.py", []byte(`import json, subprocess, sys
try:
    raw = subprocess.check_output(["nft", "-j", "list", "table", "inet", "fvpn"], stderr=subprocess.DEVNULL)
except subprocess.CalledProcessError:
    print(0)
    sys.exit(0)
doc = json.loads(raw)
total = 0
for item in doc.get("nftables", []):
    c = item.get("counter")
    if not isinstance(c, dict):
        continue
    name = str(c.get("name", ""))
    if len(name) == 34 and (name.startswith("u_") or name.startswith("d_")):
        total += int(c.get("bytes") or 0)
print(total)
`))
	out := strings.TrimSpace(mustExec(t, "python3", labDir+"/nft_usage.py"))
	var n int
	if _, err := fmt.Sscan(out, &n); err != nil {
		t.Fatalf("nft bytes %q", out)
	}
	return n
}

type usageSnap struct {
	Acked       int      `json:"acked"`
	Ledger      string   `json:"ledger"`
	CloseReason string   `json:"close_reason"`
	Bytes       int      `json:"bytes"`
	MaxSeq      int      `json:"max_seq"`
	Outbox      int      `json:"outbox"`
	Archived    int      `json:"archived"`
	EpochCloses []string `json:"epoch_closes"`
	Ledgers     []struct {
		Epoch string `json:"epoch"`
		Acked int    `json:"acked"`
	} `json:"ledgers"`
}

func (u usageSnap) hasClose(reason string) bool {
	if u.CloseReason == reason {
		return true
	}
	for _, c := range u.EpochCloses {
		if c == reason {
			return true
		}
	}
	return false
}

func readUsage(t *testing.T) usageSnap {
	t.Helper()
	writeScratch(t, labDir+"/usage_read.py", []byte(`import json, sqlite3, sys
uri = "file:/scratch/w2/state/usage.db?mode=ro"
con = sqlite3.connect(uri, uri=True)
meta = con.execute("select acked_through, ifnull(ledger_epoch,''), ifnull(last_close_reason,'') from meta where id=1").fetchone()
total = 0
max_seq = 0
outbox = 0
archived = 0
for seq, state, payload in con.execute("select seq, state, payload from events"):
    ev = json.loads(payload)
    for rec in ev.get("records") or []:
        total += int(rec.get("upload_bytes") or 0)
        total += int(rec.get("download_bytes") or 0)
    if seq > max_seq:
        max_seq = seq
    if state == "outbox":
        outbox += 1
    elif state == "archived":
        archived += 1
closes = [row[0] or "" for row in con.execute("select ifnull(close_reason,'') from epochs")]
ledgers = [{"epoch": row[0], "acked": row[1]} for row in con.execute("select ledger_epoch, acked_through from ledger_epochs order by accepted_at")]
print(json.dumps({
    "acked": meta[0], "ledger": meta[1], "close_reason": meta[2],
    "bytes": total, "max_seq": max_seq, "outbox": outbox, "archived": archived,
    "epoch_closes": closes, "ledgers": ledgers,
}))
`))
	out, errOut, err := dockerExec("python3", labDir+"/usage_read.py")
	if err != nil {
		t.Fatalf("usage.db: %v\n%s%s\n%s", err, redact(out), redact(errOut), tailLog())
	}
	var snap usageSnap
	if err := json.Unmarshal([]byte(out), &snap); err != nil {
		t.Fatalf("usage json: %v\n%s", err, out)
	}
	return snap
}
