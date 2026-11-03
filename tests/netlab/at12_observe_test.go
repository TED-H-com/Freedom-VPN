//go:build netlab

package netlab

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TEST ONLY: snapshot the actual counter reader and adjacent rule handles.
// The read brackets expose packets racing either read; no bytes are excused.
func observeAT12Counters(t *testing.T, m enforce.Model, phase string) counterDoc {
	t.Helper()
	start := time.Now().UnixNano()
	doc := readCounters(t, m)
	end := time.Now().UnixNano()
	if os.Getenv("FVPN_NETLAB_AT12_OBSERVE") == "1" {
		ruleStart := time.Now().UnixNano()
		rules := mustExec(t, "nft", "-a", "-j", "list", "table", "inet", "fvpn")
		ruleEnd := time.Now().UnixNano()
		body := mustJSON(map[string]any{"phase": phase, "counter_start_ns": start, "counter_end_ns": end,
			"rule_start_ns": ruleStart, "rule_end_ns": ruleEnd, "counter_document": doc, "ruleset": json.RawMessage(rules)})
		save(t, phase+"-observation.json", string(body)+"\n")
	}
	return doc
}

func observeAT12Start(t *testing.T) {
	t.Helper()
	if os.Getenv("FVPN_NETLAB_AT12_OBSERVE") != "1" {
		return
	}
	// A full-suite driver starts before the first scenario. AT12-only callers
	// may start here. The observer survives stopCaptures (which owns tcpdump).
	if !scratchExists("/scratch/at12-all-origin.ready") {
		mustExec(t, "python3", "/opt/netlab/scripts/at12_observe.py", "start", "/scratch/at12-all")
	}
	save(t, "entry.json", string(mustJSON(map[string]any{"wall_ns": time.Now().UnixNano(),
		"sockets": execText("ip", "netns", "exec", "fv-a1", "ss", "-ntoep")}))+"\n")
	t.Cleanup(func() {
		out, stderr, err := dockerExec("python3", "/opt/netlab/scripts/at12_observe.py", "stop", "/scratch/at12-all")
		if err != nil {
			t.Errorf("AT12 observation incomplete: %v %s%s", err, out, stderr)
		}
		for _, role := range []string{"origin", "node", "responder"} {
			for _, suffix := range []string{".jsonl", ".meta.json", ".err"} {
				name := "at12-all-" + role + suffix
				if scratchExists("/scratch/" + name) {
					save(t, name, readScratch(t, "/scratch/"+name))
				}
			}
		}
	})
}
