//go:build e2e && integration

package secops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Snapshot before another sub-case can replace agent.log on a normal restart.
// This is observational only; no new fault, retry or acceptance predicate.
func (p *liveProbe) restartSnapshot17(label string) {
	dir := filepath.Join(p.root, ".agent/secops/f17/restarts")
	if err := os.MkdirAll(dir, 0700); err != nil {
		p.t.Error(err)
		return
	}
	snapshot := map[string]any{"case_boundary": label, "captured_at": time.Now().UTC()}
	for _, name := range []string{"agent", "watchdog"} {
		raw, err := p.docker("node", "cat", "/var/lib/vpn-node/e2e/"+name+".log")
		file := strings.ReplaceAll(label, " ", "-") + "-" + name + ".log"
		if writeErr := os.WriteFile(filepath.Join(dir, file), raw, 0600); writeErr != nil {
			p.t.Error(writeErr)
		}
		lines := []string{}
		for i, line := range strings.Split(string(raw), "\n") {
			if name == "watchdog" || strings.Contains(line, "boot") || strings.Contains(line, "restart") || strings.Contains(line, "retained") || strings.Contains(line, "GENERATION_MISMATCH") {
				if line != "" {
					lines = append(lines, fmt.Sprintf("%d: %s", i+1, line))
				}
			}
		}
		snapshot[name] = map[string]any{"read_ok": err == nil, "source_file": name + ".log", "numbered_lines": lines}
	}
	raw, err := p.docker("node", "sh", "-c", `p=$(cat /var/lib/vpn-node/e2e/agent.pid) && awk '{print $1 " " $22 " " $3}' /proc/"$p"/stat`)
	snapshot["agent_pid_start_ticks_state"] = strings.TrimSpace(string(raw))
	snapshot["identity_read_ok"] = err == nil
	rows, _ := p.metadata["restart_snapshots"].([]map[string]any)
	p.metadata["restart_snapshots"] = append(rows, snapshot)
}
