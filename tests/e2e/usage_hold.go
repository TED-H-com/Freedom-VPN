//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// HoldUsageAcknowledgements withholds POST /api/v1/agent/usage-batches at the
// node resolver proxy and releases it by deleting the marker. The proxy is
// network-side: the agent dials only 127.0.0.1:9444, and an L4 drop cannot
// select that one route. Control-side fvpn-usage-admin freeze is the product
// fault, but the only return to normal is complete-recovery, which mints a
// new ledger epoch. This marker returns 503 without forwarding and without
// changing the epoch. It does not edit the outbox.
func (s *Stack) HoldUsageAcknowledgements(ctx context.Context, slot string, hold bool) error {
	container, _, err := s.usageSlot(slot)
	if err != nil {
		return err
	}
	const marker = "/var/lib/vpn-node/e2e/hold-usage"
	if !hold {
		return exec.CommandContext(ctx, "docker", "exec", container, "rm", "-f", marker).Run()
	}
	return writeContainerFile(ctx, container, marker, "600", []byte("hold\n"))
}

// UsageLedgerSnapshot runs `fvpn-agent usage snapshot -json` and parses it.
// The slot is "node", "node2", "node-clone", or "installer-node".
func (s *Stack) UsageLedgerSnapshot(ctx context.Context, slot string) (UsageSnapshot, error) {
	raw, err := s.UsageLedgerSnapshotBytes(ctx, slot)
	if err != nil {
		return UsageSnapshot{}, err
	}
	var snap UsageSnapshot
	if err = json.Unmarshal(raw, &snap); err != nil {
		return UsageSnapshot{}, err
	}
	return snap, nil
}

// UsageLedgerSnapshotBytes is the raw CLI stdout, including its trailing
// newline. Two calls are compared as bytes. Re-encoding the parsed snapshot
// is not that proof.
func (s *Stack) UsageLedgerSnapshotBytes(ctx context.Context, slot string) ([]byte, error) {
	container, config, err := s.usageSlot(slot)
	if err != nil {
		return nil, err
	}
	bin := "fvpn-agent"
	if slot == "installer-node" {
		bin = "/opt/vpn-node/current/fvpn-agent"
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", container, bin, "usage", "snapshot", "-config", config, "-json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("usage snapshot: %s", clipText(stderr.String(), 200))
	}
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("usage snapshot was empty")
	}
	return stdout.Bytes(), nil
}

func (s *Stack) usageSlot(slot string) (container, config string, err error) {
	if slot == "installer-node" {
		return s.installerContainer(), "/etc/vpn-node/agent.json", nil
	}
	container, err = s.nodeSlot(slot)
	return container, "/etc/vpn-node/agent-run.json", err
}

const (
	httpPostUsage = "post-usage"
	httpGetOther  = "get-other"
)

func (s *Stack) probeUsageRoute(ctx context.Context, slot, kind string) (int, string) {
	container, err := s.nodeSlot(slot)
	if err != nil {
		return 0, "slot"
	}
	args := []string{"exec", container, "wget", "-q", "-S", "-O", "-", "--content-on-error",
		"--ca-certificate=/etc/vpn-node/ca.pem",
		"--certificate=/var/lib/vpn-node/identity/cert.pem",
		"--private-key=/var/lib/vpn-node/identity/key.pem"}
	switch kind {
	case httpPostUsage:
		args = append(args, "--post-data={}", "https://127.0.0.1:9444/api/v1/agent/usage-batches")
	default:
		args = append(args, "https://127.0.0.1:9444/api/v1/agent/desired-state")
	}
	out, _ := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	text := string(out)
	return httpStatus(text), clipText(strings.TrimSpace(text), 500)
}

func httpStatus(text string) int {
	status := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "HTTP/") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			n := 0
			for _, c := range fields[1] {
				if c < '0' || c > '9' {
					n = 0
					break
				}
				n = n*10 + int(c-'0')
			}
			if n > 0 {
				status = n
			}
		}
	}
	return status
}
