//go:build netlab

package agentlab

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestLabExternalMeterReset flushes the forward chain from outside the agent
// while admission counters are already checkpointed. The next sample opens a
// meter epoch, records a counter_discontinuity gap, and the serving ruleset
// is back within one reconcile interval. Idle counters are covered by the
// agent unit test; this lab runs traffic so the counters are live.
func TestLabExternalMeterReset(t *testing.T) {
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

	startWatch(t)
	startAgent(t)
	if !waitProbe(t, "allow", 25*time.Second) {
		t.Fatalf("agent did not forward:\n%s", debugPlane(t))
	}
	if !waitLog(t, 0, "meter epoch opened", 15*time.Second) {
		t.Fatalf("meter did not open:\n%s", tailLog())
	}

	beforeNFT := nftUsageBytes(t)
	sendUDPBurst(t)
	time.Sleep(400 * time.Millisecond)
	afterNFT := nftUsageBytes(t)
	if afterNFT <= beforeNFT {
		t.Fatalf("udp burst did not move admission counters (%d -> %d)\n%s", beforeNFT, afterNFT, debugPlane(t))
	}

	var checkpointed epochSnap
	deadline := time.Now().Add(25 * time.Second)
	for {
		checkpointed = readEpoch(t)
		nftNow := nftUsageBytes(t)
		if checkpointed.OpenEpoch != "" && checkpointed.CounterBytes > 0 && checkpointed.CounterBytes == int64(nftNow) && nftNow >= afterNFT {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("sample did not checkpoint the burst: sqlite %+v nft %d", checkpointed, nftNow)
		}
		time.Sleep(300 * time.Millisecond)
	}
	beforeEpoch := checkpointed.OpenEpoch
	beforeDisc := checkpointed.Discontinuity

	mustExec(t, "nft", "flush", "chain", "inet", "fvpn", "forward")
	start := time.Now()
	deadline = start.Add(12 * time.Second)
	var last epochSnap
	var gen int
	restored := false
	for time.Now().Before(deadline) {
		last = readEpoch(t)
		gen = forwardGeneration(t)
		if last.OpenEpoch != "" && last.OpenEpoch != beforeEpoch && last.Discontinuity > beforeDisc && gen > 0 {
			if ok, _ := probeOK(t, "allow"); ok {
				restored = true
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	elapsed := time.Since(start).Round(time.Millisecond)
	if !restored {
		t.Fatalf("external reset not recovered in %s: epoch %+v generation %d\n%s", elapsed, last, gen, debugPlane(t))
	}
	fmt.Printf("SUMMARY external epoch=%s->%s gap=1 restored=1 elapsed=%s\n", beforeEpoch, last.OpenEpoch, elapsed)
}

var generationRe = regexp.MustCompile(`fvpn_generation=([0-9]+)`)

func forwardGeneration(t *testing.T) int {
	t.Helper()
	out, _, err := dockerExec("nft", "list", "chain", "inet", "fvpn", "forward")
	if err != nil {
		return 0
	}
	best := 0
	for _, m := range generationRe.FindAllStringSubmatch(out, -1) {
		n, conv := strconv.Atoi(m[1])
		if conv == nil && n > best {
			best = n
		}
	}
	return best
}

type epochSnap struct {
	OpenEpoch     string `json:"open_epoch"`
	OpenReason    string `json:"open_reason"`
	CounterBytes  int64  `json:"counter_bytes"`
	Discontinuity int    `json:"discontinuity"`
}

func readEpoch(t *testing.T) epochSnap {
	t.Helper()
	writeScratch(t, labDir+"/usage_epoch.py", []byte(`import json, sqlite3
uri = "file:/scratch/w2/state/usage.db?mode=ro"
con = sqlite3.connect(uri, uri=True)
meta = con.execute("select ifnull(open_epoch,''), ifnull(open_reason,'') from meta where id=1").fetchone()
epoch = meta[0]
total = 0
if epoch:
    total = con.execute("select coalesce(sum(bytes),0) from counters where epoch_id=?", (epoch,)).fetchone()[0]
disc = 0
for (payload,) in con.execute("select payload from events"):
    if isinstance(payload, str):
        payload = payload.encode()
    if b"counter_discontinuity" in payload:
        disc += 1
print(json.dumps({
    "open_epoch": epoch, "open_reason": meta[1],
    "counter_bytes": total, "discontinuity": disc,
}))
`))
	out, errOut, err := dockerExec("python3", labDir+"/usage_epoch.py")
	if err != nil {
		t.Fatalf("usage.db: %v\n%s%s\n%s", err, redact(out), redact(errOut), tailLog())
	}
	var snap epochSnap
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &snap); err != nil {
		t.Fatalf("epoch json: %v\n%s", err, out)
	}
	return snap
}
