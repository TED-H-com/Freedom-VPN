//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// EnableEnforcementTiming enables the product's existing TEST ONLY detached
// timing observer before enrollment. It never adds an in-guard callback.
func (s *Stack) EnableEnforcementTiming(ctx context.Context) error {
	if exec.CommandContext(ctx, "docker", "exec", s.lab.node, "test", "!", "-e", "/var/lib/vpn-node/identity/node.json").Run() != nil {
		return fmt.Errorf("enable timing before enrollment")
	}
	raw, err := s.lab.docker(s.lab.node, "cat", "/etc/vpn-node/agent-run.json")
	if err != nil {
		return err
	}
	var cfg map[string]any
	if json.Unmarshal(raw, &cfg) != nil || cfg["test_mode"] != true {
		return fmt.Errorf("timing requires test mode")
	}
	cfg["test_timing_path"] = "/var/lib/vpn-node/e2e/enforcement-timing.jsonl"
	raw, err = json.Marshal(cfg)
	if err != nil {
		return err
	}
	return writeContainerFile(ctx, s.lab.node, "/etc/vpn-node/agent-run.json", "600", raw)
}

// EnforcementTiming returns the original detached agent records, including
// failed operations and renewal request/acceptance bounds. Consumers join by
// boot, run and sequence, and reject sequence gaps or missing clock samples.
func (s *Stack) EnforcementTiming(ctx context.Context) ([]json.RawMessage, error) {
	raw, err := s.lab.docker(s.lab.node, "cat", "/var/lib/vpn-node/e2e/enforcement-timing.jsonl")
	if err != nil {
		return nil, err
	}
	out := []json.RawMessage{}
	scan := bufio.NewScanner(bytes.NewReader(raw))
	scan.Buffer(make([]byte, 4096), 1<<20)
	for scan.Scan() {
		b := scan.Bytes()
		if !json.Valid(b) {
			return out, fmt.Errorf("incomplete enforcement timing record")
		}
		out = append(out, append(json.RawMessage(nil), b...))
	}
	return out, scan.Err()
}
